//go:build arm64

package logfmt

import (
	"bytes"
	"encoding/binary"
	"math/bits"
)

// The escape walk and its two thresholds as tuned on arm64 (Neoverse N2); every
// other architecture uses scan_other.go. The history of both thresholds is the
// comment that starts "Thresholds for the two scans" in logfmt.go.
//
// Eight and 64, not the 5 and 48 amd64 settled on, because this walk is cheaper
// per byte and IndexByte dearer per call here: the walk costs about 0.57 cycles
// a byte, and each hand-back to the sparse scan costs an IndexByte call of ~30
// cycles plus the parity walk, so the crossover sits near 64-byte gaps. At the
// amd64 values the 48-byte-gap row ran 28% and the 64-byte-gap row 11% slower
// than here, with every other row flat (bench/counters.py, 2026-09-23).
const (
	escClean = 8  // consecutive clean words after which the walk gives up
	escGap   = 64 // bytes to the first escape at or below which the walk wins
)

// scanQuotedEscapeDense walks a quoted value from just past an escaped quote,
// a word at a time, consuming each backslash together with the byte it escapes
// — stepping over the pair IS the run-parity rule, so no backslash walk is
// needed here. It returns the position it stopped at and whether that position
// is the value's unescaped closing quote. It declines the job exactly as the
// generic version in scan_other.go does, whose comment explains the contract;
// this one is the same algorithm spelled for arm64 code generation.
//
// The generic spelling compiles here to 22 instructions per clean word: the
// clean-word counter and the index trade registers every iteration, n-8 is
// recomputed, an inline mark leaves a NOP, and the word's address is kept live
// for the drain's byte re-check. This one is 16, and draining an escape does
// not touch memory:
//
//   - The give-up test is a limit on the index (quiet), not a counter, so the
//     loop carries one induction variable. It is reset past every word that
//     held an escape, which is when the counter used to reset.
//   - The drain re-checks a lane's byte from the word already in a register,
//     byte(w >> (tz & 56)), where the generic walk loads it through a slice of
//     the word. A variable shift is one cycle from any register on arm64 — on
//     amd64 its count must sit in CL, which is what has kept that spelling off
//     x86 three times — and it takes a 4-cycle load off every escape and the
//     word's address out of the loop.
//   - tz is the mask bit's index, 8t+7 for lane t, so tz & 56 is 8t, a lane-7
//     backslash is tz == 63, and the lane itself is tz >> 3.
//
// Measured against the generic walk on a Neoverse N2 (cycles, bench/
// counters.py): 16-byte gaps -11.0%, 40-byte gaps -9.6%, esc=128 -8.9%,
// esc=32 -7.5%, prose-then-JSON -6%, JSON in msg= -3.9%, sparse rows flat.
func scanQuotedEscapeDense(data []byte, i, vStart int) (int, bool) {
	if i-1-vStart > escGap {
		return i, false // already sparse on arrival; never mind the walk
	}
	n := len(data)
	lim := n - 8
	quiet := i + 8*escClean
	kq, kb, klo, khi := swarRegs.quote, swarRegs.bslash, swarRegs.lo, swarRegs.hi
	// i >= 0 is tested per word on purpose: the prove pass cannot carry it
	// through the loop's two back edges, and without it the load pays a
	// bounds check. The test is a TBNZ that is never taken.
	for i >= 0 && i <= lim {
		w := binary.LittleEndian.Uint64(data[i : i+8])
		if m := hasQuoteOrBackslashR(w, kq, kb, klo, khi); m != 0 {
			base := i
			i += 8 // the whole word is consumed unless a lane-7 pair says otherwise
			// Drain the word of every escape before loading the next; see the
			// generic walk for why the byte is re-checked (a borrow can flag
			// a spurious lane above a true match) and why a lane-7 backslash
			// leaves the word.
			for {
				tz := uint(bits.TrailingZeros64(m))
				c := byte(w >> (tz & 56))
				if c == '"' {
					return base + int(tz>>3), true
				}
				if c != '\\' {
					m &= m - 1 // spurious lane: not a real escape, take the next
				} else if tz == 63 {
					i = base + 9 // the escaped byte is the next word's first
					break
				} else {
					// Keep only the lanes from t+2 up; see the generic walk.
					m &= (m | -m) << 9
				}
				if m == 0 {
					break
				}
			}
			quiet = i + 8*escClean
			continue
		}
		i += 8
		if i >= quiet {
			return i, false // a long clean run: IndexByte covers it faster
		}
	}
	// Fewer than eight bytes left: finish a byte at a time. i can end at n+1
	// here, one past the end, when a trailing backslash escapes the byte after
	// the input; the caller's unsigned loop head is what makes that safe.
	for i < n {
		switch data[i] {
		case '"':
			return i, true
		case '\\':
			i += 2
		default:
			i++
		}
	}
	return i, false
}

// escUpgrade is the gap between two escaped quotes at or below which
// scanQuotedSparse hands a value back to the walk: arm64 closes the one-way
// escGap decision described under "KNOWN GAP" in logfmt.go. It sits strictly
// below the 8*escClean bytes the walk gives up at, so a value handed back is
// not handed straight out again.
const escUpgrade = 48

// scanQuotedSparse finds the unescaped closing quote of a value whose escapes
// are far enough apart that one bytes.IndexByte call per escape beats walking
// the bytes between them, and returns its index, or -1 if the value is never
// closed — exactly as the generic version in scan_other.go does, whose comment
// explains the i == n+1 hand-off it survives.
//
// Unlike that version, it hands a value that turns dense back to the walk, but
// only one kind of value and only once: one the walk declined on arrival, its
// first escape more than escGap bytes in, whose escaped quotes then come
// escUpgrade bytes apart or closer. That is prose followed by embedded JSON,
// msg="failed to process request: {\"id\":...}", which otherwise pays an
// IndexByte call and a parity walk for every JSON quote. A value the walk gave
// up on — a long clean run after escapes it had been handling — stays here, as
// before: JSON whose string values are long alternates short gaps with long
// ones, and handing that back after every short gap made each long one cost a
// wasted 64-byte walk as well as the IndexByte call (+28-34% at 1:1).
//
// The caller hands over either just past an escaped quote (the walk declined on
// arrival) or just past a clean run the walk gave up on, and only the first
// leaves a '"' in front of i: a clean run holds no quote.
//
// Measured on a Neoverse N2 (cycles, bench/counters.py, 2026-09-23):
// Benchmark_IteratePrefixJSON prefix=064 and 160 -61%, the testdata sample -1%,
// escapes 128 bytes apart +1.5% (the distance test per escaped quote), 256
// apart ~. The unrestricted hand-back measured the same on amd64 as -32% and
// +3-4% (see "KNOWN GAP"), which is why this is arm64's.
func scanQuotedSparse(data []byte, i int) int {
	n := len(data)
	last := -1 // the previous escaped quote, while a hand-back is still allowed
	if uint(i-1) < uint(n) && data[i-1] == '"' {
		last = i - 1
	}
	for uint(i) < uint(n) {
		q := bytes.IndexByte(data[i:], '"')
		if q < 0 {
			break
		}
		i += q
		bs := 0
		for j := i - 1; data[j] == '\\'; j-- {
			bs++
		}
		if bs&1 == 0 {
			return i
		}
		if last >= 0 {
			if i-last <= escUpgrade {
				// vStart = i makes the walk's own escGap test pass: this
				// escape is known to be close to the last one.
				pos, done := scanQuotedEscapeDense(data, i+1, i)
				if done {
					return pos
				}
				i, last = pos, -1 // gave up on a clean run: sparse from here on
				continue
			}
			last = i
		}
		i++
	}
	return -1
}
