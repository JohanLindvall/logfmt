package logfmt

import (
	"bytes"
	"encoding/binary"
	"math/bits"
	"runtime"
	"unicode/utf16"
	"unicode/utf8"
)

// unescapeIntoSpare selects the spare-capacity decoder below for AppendUnescape
// when dst has room for all of raw — a fresh destination, which gets exactly
// that much, or a reused buffer sized for its input. A buffer with less room
// stays on AppendUnescape's append-as-you-go loop, because such a buffer may
// still fit the shorter decoded value and must then decode without allocating
// (TestAppendUnescapeBufferReuse).
//
// The decoder is portable Go, but it was measured on arm64 only (a Neoverse
// N2), where the memmove call per literal run it avoids was most of the cost of
// an escape. On amd64 the decoder is left as it was tuned there — nothing here
// was measured on x86 — and a constant false compiles the dispatch away,
// leaving AppendUnescape's machine code as it was.
const unescapeIntoSpare = runtime.GOARCH == "arm64"

// spareWindow is how many words unescapeInto probes for the next backslash
// before calling bytes.IndexByte, twice AppendUnescape's unescWindow. The
// probe here is cheaper (its constants stay in registers) and a missed one
// costs little, because after a run of 64 bytes or more the next search skips
// the probe and goes straight to IndexByte, until a shorter run turns up.
// Escapes 40-64 bytes apart — prose with an occasional \n or \" — are found by
// the probe instead of by a call, and long gaps no longer pay for a probe on
// every escape.
const spareWindow = 8

// unescapeInto decodes raw, whose first backslash is raw[j], into dst's spare
// capacity, which the caller guarantees holds len(raw) bytes. That is enough
// because decoding never lengthens, so no write here needs a capacity check or
// an append: the output index d simply advances.
//
// The point is the literal runs between escapes. AppendUnescape appends each
// one, which is a call to memmove — and on a Neoverse N2 that call, with the
// registers spilled and reloaded around it, was most of the cost of an escape:
// 92 instructions per escape at 8-byte gaps, 11-15% of the profile in memmove
// itself. Here a run of up to 32 bytes is copied inline, as two to four
// overlapping loads followed by as many stores, and only longer runs call
// memmove, whose wide copies win once there is that much to move.
//
// Every copy loads all of its bytes before it stores any, and writes only the
// bytes of its own run. That keeps an in-place decode correct: with dst =
// raw[:0] (or any destination whose write position never passes the read
// position, which decoding guarantees once it holds at the start) a store can
// only land on input already read, never on input still to come. The escape
// decoders read their whole escape before writing its (shorter) output, for the
// same reason. FuzzAppendUnescapeAgainstRef checks both placements.
//
// Measured against AppendUnescape's own loop on a Neoverse N2 (cycles, bench/
// counters.py, 2026-09-23): JSON in msg= -13%, esc=500 -11%, esc=128 -11%,
// esc=32 -9%, escapes 16/32/40/48/64/128 bytes apart -10/-8/-32/-25/-13/-7%,
// 256 bytes apart ~, \u escapes ~. Prose with two escapes (Benchmark_Unescape)
// is +5%: a decode this short cannot amortise the one call more it makes (this
// function), and folding the loop into AppendUnescape to save it measured worse
// on every dense row.
func unescapeInto(dst, raw []byte, j int) []byte {
	buf := dst[:cap(dst)]
	d := len(dst)
	n := len(raw)
	d += copy(buf[d:], raw[:j])
	kb, klo, khi := swarRegs.bslash, swarRegs.lo, swarRegs.hi
	far := false // the last run was too long for the probe; see spareWindow
	for {
		// The escape at raw[j].
		i := j + 1
		if i >= n {
			buf[d] = '\\' // a trailing lone backslash is kept
			return buf[:d+1]
		}
		next := raw[i]
		i++
		if next != 'u' {
			buf[d] = unescapeByte[next]
			d++
		} else if r := hex4(raw[i:]); r >= 0 {
			adv := 4
			if utf16.IsSurrogate(r) {
				// decodeSurrogateEscape, written out: as a call it made the
				// pair path spill and reload this loop's state twice.
				r1 := r
				r = utf8.RuneError
				if b := raw[i:]; len(b) >= 10 && b[4] == '\\' && b[5] == 'u' {
					if r2 := hex4(b[6:]); r2 >= 0 {
						if p := utf16.DecodeRune(r1, r2); p != utf8.RuneError {
							r, adv = p, 10
						}
					}
				}
			}
			d = len(utf8.AppendRune(buf[:d], r))
			i += adv
		} else {
			buf[d] = '\\' // a malformed \u is kept verbatim
			buf[d+1] = 'u'
			d += 2
		}
		if i >= n {
			return buf[:d]
		}
		if raw[i] == '\\' {
			j = i // adjacent escapes: no literal run between them
			continue
		}
		// The next escape: a few words inline, then IndexByte. The probe is
		// AppendUnescape's, with its constants held in registers — spelled as
		// constants, every word rebuilt 0x5c5c... with a MOVZ and three MOVKs.
		j = -1
		s := i
		if !far {
			for quiet := 0; quiet < spareWindow && s >= 0 && s <= n-8; quiet++ {
				w := binary.LittleEndian.Uint64(raw[s : s+8])
				b := w ^ kb
				if m := (b - klo) & (khi &^ b); m != 0 {
					j = s + bits.TrailingZeros64(m)>>3
					break
				}
				s += 8
			}
		}
		if j < 0 {
			q := bytes.IndexByte(raw[s:], '\\')
			if q < 0 {
				d += copy(buf[d:], raw[i:])
				return buf[:d]
			}
			j = s + q
			far = j-i >= 8*spareWindow
		}
		// The literal run raw[i:j], at least one byte long.
		switch run := j - i; {
		case run > 32:
			copy(buf[d:d+run], raw[i:j])
		case run > 16:
			a, b := binary.LittleEndian.Uint64(raw[i:i+8]), binary.LittleEndian.Uint64(raw[i+8:i+16])
			c, e := binary.LittleEndian.Uint64(raw[j-16:j-8]), binary.LittleEndian.Uint64(raw[j-8:j])
			binary.LittleEndian.PutUint64(buf[d:d+8], a)
			binary.LittleEndian.PutUint64(buf[d+8:d+16], b)
			binary.LittleEndian.PutUint64(buf[d+run-16:d+run-8], c)
			binary.LittleEndian.PutUint64(buf[d+run-8:d+run], e)
		case run >= 8:
			a, b := binary.LittleEndian.Uint64(raw[i:i+8]), binary.LittleEndian.Uint64(raw[j-8:j])
			binary.LittleEndian.PutUint64(buf[d:d+8], a)
			binary.LittleEndian.PutUint64(buf[d+run-8:d+run], b)
		case run >= 4:
			a, b := binary.LittleEndian.Uint32(raw[i:i+4]), binary.LittleEndian.Uint32(raw[j-4:j])
			binary.LittleEndian.PutUint32(buf[d:d+4], a)
			binary.LittleEndian.PutUint32(buf[d+run-4:d+run], b)
		case run >= 2:
			a, b := binary.LittleEndian.Uint16(raw[i:i+2]), binary.LittleEndian.Uint16(raw[j-2:j])
			binary.LittleEndian.PutUint16(buf[d:d+2], a)
			binary.LittleEndian.PutUint16(buf[d+run-2:d+run], b)
		default:
			buf[d] = raw[i]
		}
		d += j - i
	}
}
