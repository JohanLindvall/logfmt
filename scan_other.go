//go:build !arm64

package logfmt

import (
	"bytes"
	"encoding/binary"
	"math/bits"
)

// The escape walk and its two thresholds as tuned on amd64; arm64 has its own
// in scan_arm64.go. The history of both thresholds is the comment that starts
// "Thresholds for the two scans" in logfmt.go.
const (
	escClean = 5  // consecutive clean words after which the walk gives up
	escGap   = 48 // bytes to the first escape at or below which the walk wins
)

// scanQuotedEscapeDense walks a quoted value from just past an escaped quote,
// a word at a time, consuming each backslash together with the byte it escapes
// — stepping over the pair IS the run-parity rule, so no backslash walk is
// needed here. It returns the position it stopped at and whether that position
// is the value's unescaped closing quote.
//
// It declines the job in two ways, both handing back to scanQuotedSparse: the
// first escape sat more than escGap bytes into the value (so the escapes are
// sparse and IndexByte's wider stride wins), or escClean consecutive words went
// by with neither byte in them (so they have become sparse part way through).
// Handing back needs nothing carried across, because the parity rule is
// context-free — the caller resumes with a plain IndexByte from wherever this
// stopped.
//
// n is derived here rather than passed: the caller's n IS len(data), and
// spelling it that way inside this function is what lets the prove pass relate
// i to the slice at all. The loop head is unsigned for the reason iterate's is
// — it means i < n while also supplying the i >= 0 fact — which matters because
// the i += 2 step below can leave i at n+1, and that value must reach the head
// as "stop", not as a negative-looking index that costs every bounds check in
// the loop.
func scanQuotedEscapeDense(data []byte, i, vStart int) (int, bool) {
	if i-1-vStart > escGap {
		return i, false // already sparse on arrival; never mind the walk
	}
	n := len(data)
	clean := 0
	kq, kb, klo, khi := swarRegs.quote, swarRegs.bslash, swarRegs.lo, swarRegs.hi
	// One word is loaded, masked and then DRAINED of every escape it holds
	// before the next is loaded. The reload was this walk's whole dependency
	// chain — load, mask, find, verify, step, load again, about fourteen
	// cycles per escape with nothing else to overlap — and at the densities
	// this scan exists for (embedded JSON escapes every two to eight bytes)
	// a word holds several. Draining costs a few bit operations per escape
	// instead, and the word loads become independent of each other.
	//
	// Two things the outer walk got for free have to be paid for here. A
	// spurious lane — the borrow above a true match described on
	// hasQuoteOrBackslash — used to be impossible because every mask was
	// taken from a freshly anchored word, where only the LOWEST bit is read
	// and that one is genuine; draining reads the ones above it too, so the
	// byte is checked for '\\' as well as '"' and an impostor is simply
	// cleared. And a backslash in the last lane escapes a byte the word does
	// not contain, so that case leaves the word and resumes past the pair.
	for i >= 0 && i <= n-8 && clean < escClean {
		w := binary.LittleEndian.Uint64(data[i : i+8])
		m := hasQuoteOrBackslashR(w, kq, kb, klo, khi)
		if m == 0 {
			i += 8
			clean++
			continue
		}
		clean = 0
		base := i
		// The sub-slice and the "& 7" are both load-bearing, and only
		// together: the byte re-check below runs once per escape rather than
		// once per word, and spelled data[base+t] it pays a bounds check
		// every time, because the prove pass will not combine i <= n-8 with
		// t's range. Against a slice whose length it knows is 8 the masked
		// index needs no check at all. (-d=ssa/check_bce/debug=1 reports
		// nothing in this function.)
		word := data[i : i+8]
		i += 8 // the whole word is consumed unless a lane-7 pair says otherwise
		for m != 0 {
			// The lane comes from m itself, not from m & -m: the loop condition
			// proves m non-zero, where TrailingZeros64 of the isolated bit
			// compiled to a BSF plus a CMOV for a zero input that cannot occur.
			t := bits.TrailingZeros64(m) >> 3 & 7
			c := word[t]
			if c == '"' {
				return base + t, true
			}
			if c != '\\' {
				m &= m - 1 // spurious lane: not a real escape, take the next
				continue
			}
			if t == 7 {
				i = base + 9 // the escaped byte is the next word's first
				break
			}
			// Step over the backslash AND what it escapes by keeping only the
			// lanes from t+2 up: m | -m sets every bit from m's lowest one,
			// bit 7 of lane t, upward, and shifting that left by 9 starts it
			// at bit 0 of lane t+2 (or clears it when there is no such lane).
			// Four dependent operations per escape, where clearing the pair
			// with m &^= low | low<<8 took six — and this chain is the walk's
			// whole cost once a word holds several escapes.
			m &= (m | -m) << 9
		}
	}
	if clean >= escClean {
		return i, false // a long clean run: IndexByte covers it faster
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

// scanQuotedSparse finds the unescaped closing quote of a value whose escapes
// are far enough apart that one bytes.IndexByte call per escape beats walking
// the bytes between them. It returns the closing quote's index, or -1 if the
// value is never closed. i may arrive at n+1 from the dense scan's last step;
// the unsigned head treats that as "nothing left", which is what stops the
// data[i:] below from panicking rather than merely failing to match.
func scanQuotedSparse(data []byte, i int) int {
	n := len(data)
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
		i++
	}
	return -1
}
