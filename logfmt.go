package logfmt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/bits"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// ErrBadFormat is the sentinel every syntax error matches: the input is not
// valid logfmt, for example a quoted value that is never closed or that is
// followed by a non-space byte. Errors returned by Iterate and Validate are
// *SyntaxError values carrying the offset; test them with
// errors.Is(err, ErrBadFormat).
var ErrBadFormat = errors.New("bad logfmt format")

// SyntaxError describes a malformed logfmt record and where the parser gave up.
// Iterate and Validate return it; errors.Is(err, ErrBadFormat) reports true for
// any of them, so sentinel comparisons keep working.
type SyntaxError struct {
	// Offset is the byte index in the input at which the fault was detected:
	// the opening quote of an unterminated value, or the offending byte after
	// a closing quote.
	Offset int
	// Reason is a short description of the fault, without position.
	Reason string
}

func (e *SyntaxError) Error() string {
	return "logfmt: " + e.Reason + " at offset " + strconv.Itoa(e.Offset)
}

// Is makes every SyntaxError match ErrBadFormat under errors.Is.
func (e *SyntaxError) Is(target error) bool { return target == ErrBadFormat }

var trueSlice = []byte("true")

var spaceTable = [256]bool{
	' ':  true,
	'\t': true,
	'\n': true,
	'\r': true,
	'\f': true,
	'\v': true,
}

func isSpace(b byte) bool {
	return spaceTable[b]
}

// SWAR (SIMD-within-a-register) helpers scan eight bytes of a key or value at
// a time. They set the high bit (0x80) of byte positions that match; we locate
// the first match with bits.TrailingZeros64. Spurious high bits can appear in
// bytes MORE significant than a true match (a borrow propagates upward), but
// never on or below it, so the lowest set bit is always a real match as long
// as we only ever OR these masks together (never subtract them).
const (
	swarLo = 0x0101010101010101 // 0x01 in every byte
	swarHi = 0x8080808080808080 // 0x80 in every byte
)

// denseGap is the gap between escaped quotes, in bytes, at or below which the
// quoted value scan switches from bytes.IndexByte to the inline SWAR scan, and
// above which it switches back. One SWAR word.
//
// It is measured, and the measurement is not the obvious one. On the synthetic
// sweep alone a much wider threshold looks better: at 32 it also catches
// Benchmark_IterateEscaped's 32-byte-gap case for a further −9.4%. But the same
// setting sent the bench module's realistic escaped line (four escapes, gaps of
// 18/2/2/9) into the dense scan and cost it 7.3%, because that value has too
// few escapes to amortise entering the mode at all. Dropping to one word gives
// up the synthetic case, returns the realistic line to parity, and keeps the
// full −33% where the escapes are actually dense. It is the same trade the
// 2026-07-27 pass made against DecodeKeyval: the realistic suite wins.
//
// Both ends of the switch use it, and they have to agree: a value whose escapes
// sit exactly denseGap apart must not enter dense mode only to hand itself
// straight back, paying both scans for every escape. The entry test measures
// the gap to the quote and the exit test measures the whole step over the
// escape pair, which is why the exit is the strictly-greater one.
const denseGap = 8

// hasCtrlOrSpace flags every byte of w that is <= 0x20. This covers all logfmt
// whitespace ('\t'..'\r' and ' '); the only other bytes it flags are control
// bytes 0x00..0x08 and 0x0E..0x1F, which the caller rules out by re-checking
// the located byte. UTF-8 continuation/lead bytes (>= 0x80) are never flagged.
func hasCtrlOrSpace(w uint64) uint64 {
	return (w - swarLo*0x21) &^ w & swarHi
}

// hasQuoteOrEsc flags every byte of w that is '"' or '\\'. It is used only once
// a quoted value is known to contain an escaped quote, where the two bytes
// together are what drives the scan: a '\\' means "skip the next byte", a '"'
// means the value ends here.
//
// Unlike the two masks around it these are exact equalities, so the terms
// cannot share a subtrahend the way hasKeyStop's do; each is the classic
// "does any byte equal zero" test applied to w XOR-ed with a broadcast of the
// byte being sought. They are still only OR-ed together, never subtracted, so
// the borrow caveat above stays harmless: every spurious high bit sits above a
// true match, and only the lowest set bit is ever read.
func hasQuoteOrEsc(w uint64) uint64 {
	q := w ^ (swarLo * '"')
	e := w ^ (swarLo * '\\')
	return ((q-swarLo)&^q | (e-swarLo)&^e) & swarHi
}

// hasKeyStop flags every byte of w that can end a key: '=' or <= 0x20. It is
// two "<= 0x20" tests OR-ed together, with everything they share factored out.
//
// The second test is what handles '=', without an equality mask: XOR-ing by
// 0x1d turns "b <= 0x20" into a test for exactly {0x00..0x1f} + {'='}. That
// holds because 0x1d < 0x20, so the XOR permutes the low 32 values among
// themselves, while 0x3d ^ 0x1d == 0x20 pulls '=' in and nothing else reaches
// 0x20 or below (bits 5 and 6 are untouched by 0x1d). Union it with the plain
// "b <= 0x20" term and the result is exactly {0x00..0x20} + {'='}.
//
// Doing it that way lets both terms subtract the SAME word (swarSpW), so the
// scan needs three live broadcast masks instead of four. The shared "&^ w" is
// factored out too, which is sound because 0x1d has bit 7 clear: bit 7 of every
// byte of x therefore equals bit 7 of w, and bit 7 is the only position the
// result is ever read at.
//
// Combining the terms with OR (never subtraction) is what keeps the borrow
// caveat above harmless: every spurious bit sits above a true match, so the
// lowest set bit of the union is still a genuine stop.
func hasKeyStop(w uint64) uint64 {
	x := w ^ (swarLo * 0x1d)
	return ((w - swarLo*0x21) | (x - swarLo*0x21)) &^ w & swarHi
}

// Iterate parses data as a logfmt record and calls fn once for each key/value
// pair, in order. key and val are sub-slices that alias data — except for a
// bare key with no '=' (for example "debug", or a trailing token), whose val is
// a shared constant "true". Treat both as read-only, and copy them if they must
// outlive the call.
//
// `key=` followed by whitespace is an EMPTY value, and that whitespace still
// separates the next token: "key= value" yields ("key", "") and then the bare
// key ("value", "true"). go-logfmt reads it the same way, bar the bare-key
// sentinel. A quoted value is returned without its surrounding double quotes
// but is NOT unescaped — backslash escapes are left intact.
//
// val does not record whether it was quoted, and that distinction is the one
// that decides whether decoding it is correct: escapes mean something only
// inside quotes, so an unquoted path=C:\Users\bob holds three literal
// backslashes. Running val through AppendUnescape from inside this callback
// therefore corrupts every unquoted value that contains one. Use AppendValue,
// which decodes only what was quoted, or GetQuoted, which hands back the bit.
//
// fn may return false to stop iteration early, in which case Iterate returns
// nil. Iterate returns a *SyntaxError (which errors.Is matches against
// ErrBadFormat) if data contains a malformed quoted value, and otherwise nil.
// Every pair before the fault has already been delivered. It allocates nothing
// on well-formed input; a returned SyntaxError is the only allocation it can
// make.
func Iterate(data []byte, fn func(key, val []byte) bool) error {
	return iterate(data, func(key, val []byte, _ bool) bool { return fn(key, val) })
}

// iterate is the parser; every exported entry point funnels through it. It
// reports one fact Iterate's callback signature has no room for: whether the
// value came from a double-quoted token, which is the only position where a
// backslash escape means anything. The lookups take that bit straight from here
// rather than guessing at it afterwards, which is what stops AppendValue from
// "decoding" a Windows path.
func iterate(data []byte, fn func(key, val []byte, quoted bool) bool) error {
	// cap == len stops a callback's append reaching past the end of the record
	// — a tightening of the read-only contract, never a loosening — for one
	// instruction once per call. (Its original bounds-check role is now played
	// by the loop spellings below; see the loop-invariant comment.)
	data = data[:len(data):len(data)]
	n := len(data)
	// Loop invariant worth stating because two steps below rely on it: a field
	// consumes its own trailing separator, so i can finish a step at n+1 rather
	// than n. Only one step actually leaves i at n+1 (the unconditional i++ at
	// valEnd), and the very next bound it meets is this loop's head. That head
	// and the two SWAR heads are spelled "uint(i) < uint(n)" and "i <= n-8", and
	// both treat n+1 exactly as they treat n — which is what lets that step be
	// branchless. (The remaining "i < n" bounds further down are ordinary
	// scalar-tail guards, reached only with i <= n; they are not part of this
	// invariant, and an earlier version of this comment wrongly claimed EVERY
	// bound in the function used the two spellings above.)
	//
	// The spellings are deliberate: i is never negative, so the
	// unsigned compare means i < n, but it also hands the prove pass the
	// i >= 0 fact it otherwise never has, and "i <= n-8" proves i+8 <= n
	// without an add that could overflow. The pair is what eliminates every
	// bounds check on the SWAR loads and in the separator-drain loop; neither
	// half does it alone (the half-applied form measures WORSE — see
	// CLAUDE.md on the previously rejected bare "lim := n-8" attempt).
	for i := 0; uint(i) < uint(n); {
		// There is no whitespace-skip loop here, on purpose. The previous field
		// already stepped past its separator, so i points at the key. Leading
		// whitespace, a run of separators, or a '\t'/'\n' delimiter instead
		// leaves the scan below stopping at offset zero with an empty key,
		// which keyBare handles once, off the hot path.
		kStart := i
		// Declared up here so the gotos, which jump forward, skip no declaration.
		// quoted re-zeroes every iteration by virtue of being declared inside the
		// loop body; only the quoted branch below ever sets it.
		var kEnd, vStart, vEnd int
		var quoted bool

		for i <= n-8 {
			w := binary.LittleEndian.Uint64(data[i : i+8])
			m := hasKeyStop(w)
			if m != 0 {
				i += bits.TrailingZeros64(m) >> 3
				// '=' first: keys overwhelmingly end there. Each stop byte
				// jumps straight to the label that already knows both what was
				// found and that i < n, so neither label reloads data[i] nor
				// re-tests a bound this hit has already established.
				c := data[i]
				if c == '=' {
					goto keyEq
				}
				if isSpace(c) {
					goto keyBare
				}
				break // rare non-whitespace control byte; finish scalar
			}
			i += 8
		}
		for i < n && !isSpace(data[i]) && data[i] != '=' {
			i++
		}

		if i >= n {
			if kStart < n {
				fn(data[kStart:n], trueSlice, false)
			}
			return nil
		}
		if data[i] == '=' {
			goto keyEq
		}

	keyBare:
		// data[i] is whitespace: the key scan stops at nothing else. An EMPTY
		// key means i was sitting on a separator run rather than on a key, so
		// drain the run and start the field over. This is the only place a run
		// costs anything, and no real emitter writes one.
		if i == kStart {
			i++
			for i < n && isSpace(data[i]) {
				i++
			}
			continue
		}
		if !fn(data[kStart:i], trueSlice, false) {
			return nil
		}
		i++ // step past the separator the key stopped on
		continue

	keyEq:
		kEnd = i
		i++

		vStart, vEnd = i, i

		if i >= n {
			fn(data[kStart:kEnd], data[vStart:vEnd], false)
			return nil
		}

		// `key=` followed by whitespace is an EMPTY value — it is NOT a
		// lookahead past the separator for the next token. Skipping the
		// whitespace here made "err= level=info" parse as a single pair
		// err="level=info", swallowing the following key outright: the value
		// is garbage and the key is simply gone. That shape is what every real
		// emitter produces for an empty string (go-kit's and logrus's logfmt
		// encoders both write a bare "key="), whereas the "key= value" spelling
		// this leniency was meant to accept is something no encoder writes and
		// only a human hand-types. Trading a common correct parse for a rare
		// convenience was the wrong way round.
		//
		// No explicit test is needed for it: whitespace is not '"', so control
		// falls into the unquoted scan, which stops on the very first byte and
		// leaves vEnd == vStart — the same empty value, one branch and one
		// isSpace cheaper on every field.
		if data[i] == '"' {
			quoted = true
			i++
			vStart = i

			// Finding the closing quote runs in two modes, because no single
			// scan wins on both shapes of quoted value.
			//
			// bytes.IndexByte is SIMD and unbeatable on a clean span — one
			// call, one scan, done — and that is what a quoted value almost
			// always is. What it is bad at is an escape-DENSE value, where
			// every escaped quote restarts the non-inlinable call over a span
			// one or two bytes long, so the cost becomes O(escapes) calls
			// instead of O(bytes/8) inline steps. Embedded JSON is exactly that
			// shape — every quote in it arrives as \" — and it is the commonest
			// hard case in real logfmt.
			//
			// So: start in IndexByte mode, and switch to the inline SWAR scan
			// only once an escape has been seen AND the gap that preceded it
			// was short enough for the call overhead to dominate; switch back
			// as soon as the gaps grow again. Measured on 1 KB values
			// (Benchmark_IterateEscaped): −33% at 8-byte gaps, −33% at 2-byte
			// ones, and at 128-byte gaps +2.5%, which is one compare per
			// escaped quote and the whole price of the mode test. A value with
			// no escaped quote in it never even reaches that compare.
			for {
				q := bytes.IndexByte(data[i:], '"')
				if q < 0 {
					// vStart is just past the opening quote, which is the
					// position worth reporting.
					return &SyntaxError{Offset: vStart - 1, Reason: "unterminated quoted value"}
				}
				i += q

				// Determine whether this quote is escaped by counting the
				// run of backslashes immediately preceding it: an odd count
				// means the quote is escaped and the scan goes on. The walk
				// needs no lower-bound guard: data[vStart-1] is the opening
				// quote (this path is entered only via '=' then '"'), any
				// earlier escaped quote inside the value is also '"', and
				// neither is a backslash, so the run always stops on its own.
				// bs&1 rather than bs%2 spares the signed-modulo dance the
				// compiler emits when it cannot prove bs >= 0 across the loop.
				// Parity is a purely local property of the bytes before the
				// quote, which is why the two modes can hand the scan back and
				// forth mid-value without agreeing on where it started.
				bs := 0
				for j := i - 1; data[j] == '\\'; j-- {
					bs++
				}
				if bs&1 == 0 {
					break // unescaped: this is the closing quote
				}
				i++
				if q > denseGap {
					continue // escapes are far apart; IndexByte still wins
				}

				// Escape-dense from here. The scan itself lives in its own
				// function rather than inline, and that is a measured choice,
				// not a stylistic one: inlining it grew iterate by enough to
				// cost GetMany ~5% and Extract ~3% — benchmarks that never
				// reach this code at all — while the call it replaces is paid
				// once per dense value rather than once per escape.
				var closed bool
				if i, closed = scanEscapeDense(data, i); closed {
					break
				}
				if uint(i) >= uint(n) {
					return &SyntaxError{Offset: vStart - 1, Reason: "unterminated quoted value"}
				}
				// Otherwise the gaps opened up and the scan was handed back.
			}

			vEnd = i
			i++
			if i < n {
				// ' ' first: the usual delimiter short-circuits past
				// the isSpace test.
				if c := data[i]; c != ' ' && !isSpace(c) {
					return &SyntaxError{Offset: i, Reason: "unexpected byte after closing quote"}
				}
				i++
			}
		} else {
			vStart = i
			for i <= n-8 {
				w := binary.LittleEndian.Uint64(data[i : i+8])
				m := hasCtrlOrSpace(w)
				if m != 0 {
					i += bits.TrailingZeros64(m) >> 3
					// ' ' first: it is the usual value delimiter, so the cheap
					// compare short-circuits past the isSpace test.
					if c := data[i]; c == ' ' || isSpace(c) {
						goto valEnd
					}
					break // rare non-whitespace control byte; finish scalar
				}
				i += 8
			}
			for i < n && !isSpace(data[i]) {
				i++
			}
		valEnd:
			vEnd = i
			// The scan stopped on whitespace or ran out of input, so stepping
			// past the separator needs no guard: i == n+1 fails every bound
			// exactly as i == n does. Unlike "if i < n { i++ }" this costs no
			// branch, and it is what removes the whitespace-skip loop that
			// used to open every field.
			i++
		}

		if !fn(data[kStart:kEnd], data[vStart:vEnd], quoted) {
			return nil
		}
	}

	return nil
}

// scanEscapeDense finds the closing quote of a quoted value that has turned out
// to be escape-dense, scanning a word at a time inline instead of restarting
// bytes.IndexByte at every escape. It is entered with i just past an escaped
// quote — a position that cannot itself be inside an escape sequence — which is
// what lets it step over two bytes at a '\\' and never walk a backslash run
// again: stepping over the pair IS the run-parity rule, since `\\` lands on the
// following quote and reports it unescaped while `\"` steps past it.
//
// It returns where it stopped and whether that is the closing quote. There are
// three ways to stop, which those two values tell apart:
//
//   - the closing quote, at i < n, closed true;
//   - the gaps between escapes grew past denseGap, at i < n, closed false — the
//     caller hands the rest of the value back to bytes.IndexByte, which wins
//     again once there are enough bytes between calls to pay for one;
//   - the input ran out, at i >= n, closed false: the value is unterminated.
//     i can be n+1 here, because the step over an escape pair may cross the end.
func scanEscapeDense(data []byte, i int) (int, bool) {
	// n is derived here rather than passed: the caller's n IS len(data), and
	// spelling it that way inside this function is what lets the prove pass
	// relate i to the slice at all. Passed in as a plain int it cannot, and
	// every access below — the word load included — keeps a bounds check.
	n := len(data)
	for {
		// Both of this loop's bounds are spelled unsigned, and iterate's loop
		// heads explain why: it reads "i >= n" (i is never negative) while also
		// handing the prove pass the i >= 0 fact — which the n+1 step at the
		// bottom otherwise destroys, taking every bounds-check elimination in
		// this loop with it, the word load's included. The pair is what keeps
		// this scan checked as lightly as the other two; neither half does it
		// alone.
		if uint(i) >= uint(n) {
			return i, false
		}
		dStart := i
		for i <= n-8 {
			w := binary.LittleEndian.Uint64(data[i : i+8])
			if m := hasQuoteOrEsc(w); m != 0 {
				i += bits.TrailingZeros64(m) >> 3
				break
			}
			i += 8
		}
		// Reached both on a SWAR hit, where it exits at once because the stop
		// byte fails both tests, and on the final fewer-than-8 bytes.
		for uint(i) < uint(n) && data[i] != '"' && data[i] != '\\' {
			i++
		}
		if uint(i) >= uint(n) {
			return i, false // nothing left that could close the value
		}
		if data[i] == '"' {
			return i, true
		}
		i += 2 // '\\': step over it and the byte it escapes
		if i-dStart > denseGap && i < n {
			// The i < n half is not redundant: the step above can leave i at
			// n+1, where the caller's bytes.IndexByte(data[i:]) does not merely
			// fail to match but panics. Staying in the loop sends that case to
			// the check at the top instead, which reports it as the
			// unterminated value it is.
			return i, false
		}
	}
}

// AppendUnescape decodes the backslash escapes in a raw logfmt value, appends
// the result to dst and returns the extended slice. It recognises \n, \r, \t
// and JSON-style \uXXXX unicode escapes (including surrogate pairs, as emitted
// by go-logfmt for control characters); any other escaped byte (such as \" or
// \\) is emitted as the byte itself. A lone surrogate half decodes to U+FFFD,
// matching encoding/json. A malformed \u (bad or truncated hex) and a trailing
// lone backslash are kept verbatim rather than rejected.
//
// Pass it ONLY a value that was quoted in the input. Escapes are meaningful
// only inside quotes: an emitter writes path=C:\Users\bob unquoted and means
// every byte literally, so decoding that yields C:Usersbob with an embedded
// newline. Iterate, All, Get and GetMany all hand out quoted and unquoted
// values alike without distinguishing them; GetQuoted reports which it was, and
// AppendValue applies this function only when it should. Feeding raw values
// through here unconditionally is the one way to corrupt data with this package.
//
// It always appends — the result never aliases raw — so it composes like the
// other Append functions in the standard library. Pass dst[:0] to reuse a
// buffer without allocating. To skip the copy entirely for values that need no
// decoding, guard with NeedsUnescape; that pattern is also faster than decoding
// unconditionally, since most values contain no escapes at all.
func AppendUnescape(dst []byte, raw []byte) []byte {
	i, n := 0, len(raw)
	for i < n {
		q := bytes.IndexByte(raw[i:], '\\')
		if q < 0 {
			// no more escapes
			return append(dst, raw[i:]...)
		}
		dst = append(dst, raw[i:i+q]...)
		i += q + 1
		if i < n {
			next := raw[i]
			i++
			switch next {
			case 'n':
				dst = append(dst, '\n')
			case 'r':
				dst = append(dst, '\r')
			case 't':
				dst = append(dst, '\t')
			case 'u':
				if r, adv, ok := decodeUnicodeEscape(raw[i:]); ok {
					dst = utf8.AppendRune(dst, r)
					i += adv
				} else {
					dst = append(dst, '\\', 'u') // malformed: keep verbatim
				}
			default:
				dst = append(dst, next)
			}
		} else {
			dst = append(dst, '\\')
			break
		}
	}
	return dst
}

// decodeUnicodeEscape decodes the hex payload of a \uXXXX escape at the start
// of b (the caller has consumed the "\u"). It returns the rune, the number of
// payload bytes consumed (4, or 10 when a low-surrogate escape follows and the
// two combine), and whether the payload was well-formed. Surrogate handling
// matches encoding/json: a valid high+low pair combines; a lone half yields
// U+FFFD.
func decodeUnicodeEscape(b []byte) (rune, int, bool) {
	r1 := hex4(b)
	if r1 < 0 {
		return 0, 0, false
	}
	if !utf16.IsSurrogate(r1) {
		return r1, 4, true
	}
	// A high surrogate may combine with an immediately following \uXXXX low
	// surrogate. Anything else (lone half, invalid pair) becomes U+FFFD.
	if len(b) >= 10 && b[4] == '\\' && b[5] == 'u' {
		if r2 := hex4(b[6:]); r2 >= 0 {
			if r := utf16.DecodeRune(r1, r2); r != utf8.RuneError {
				return r, 10, true
			}
		}
	}
	return utf8.RuneError, 4, true
}

// hex4 parses exactly four hex digits from the start of b, returning -1 if b is
// too short or contains a non-hex byte.
func hex4(b []byte) rune {
	if len(b) < 4 {
		return -1
	}
	var r rune
	for _, c := range b[:4] {
		r <<= 4
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			r |= rune(c-'A') + 10
		default:
			return -1
		}
	}
	return r
}

// AppendValue looks up key in data, appends its unescaped value to dst and
// returns the extended slice along with whether the key was present. When the
// key is absent it returns dst unchanged and false.
//
// Only a quoted value carries escapes, and AppendValue knows which values were
// quoted, so an unquoted one is copied through byte for byte — path=C:\Users\bob
// comes back intact rather than "decoded" into nonsense.
//
// It always appends, so the result never aliases data and is safe to keep.
// Callers who would rather not copy values that need no decoding should use
// GetQuoted with NeedsUnescape instead:
//
//	if v, quoted, ok := logfmt.GetQuoted(line, "msg"); ok {
//		if quoted && logfmt.NeedsUnescape(v) {
//			v = logfmt.AppendUnescape(buf[:0], v)
//		}
//		// v now aliases line (no copy) or buf (decoded)
//	}
//
// Duplicate keys resolve exactly as in Get and GetMany: the first non-empty
// occurrence wins, and an empty value is used only when no non-empty one
// exists. A malformed record yields whatever could be parsed before the fault;
// use Validate when you need to know.
func AppendValue(dst, data []byte, key string) ([]byte, bool) {
	raw, quoted, ok := GetQuoted(data, key)
	if !ok {
		return dst, false
	}
	// Only a quoted value carries escapes. Decoding an unquoted one would eat
	// the backslashes an emitter meant literally — path=C:\Users\bob becomes
	// C:Usersbob with an embedded newline — so it is copied through verbatim.
	if !quoted || !NeedsUnescape(raw) {
		return append(dst, raw...), true
	}
	return AppendUnescape(dst, raw), true
}

// NeedsUnescape reports whether raw contains a backslash at all. Values
// returned by Iterate, All, Get, GetQuoted and GetMany are raw; use this to skip
// the decode (and its copy) when it is unnecessary.
//
// It is conservative in one direction: a false result guarantees AppendUnescape
// would not change raw, but a true one does not guarantee it would. The
// sequences AppendUnescape deliberately keeps verbatim — a malformed \u such as
// `\uZZZZ`, and a trailing lone backslash — contain a backslash and so report
// true while decoding to themselves. That costs a needless copy, never a wrong
// answer.
//
// A true result also does not mean decoding is CORRECT: escapes are meaningful
// only inside quotes, so check the value was quoted first (GetQuoted), or let
// AppendValue handle both questions.
func NeedsUnescape(raw []byte) bool {
	return bytes.IndexByte(raw, '\\') >= 0
}

// IsBareKey reports whether val is the sentinel that Iterate and All substitute
// for a bare key — one written with no '=' at all, such as the "debug" in
// "level=info debug" — as opposed to a real value that happens to read "true".
// The two are otherwise indistinguishable, since both arrive as the bytes
// "true".
//
// It compares identity, not contents. Values delivered by Iterate and All and
// values returned by Get, GetQuoted and GetMany all report true for a bare key;
// AppendValue's result never does, since it copies. A []byte of the caller's own
// reports false however it reads.
func IsBareKey(val []byte) bool {
	return len(val) == len(trueSlice) && &val[0] == &trueSlice[0]
}

// Get returns the raw value for key in data — the value as it appears in the
// input, with any surrounding quotes removed but escape sequences left intact —
// and whether the key was present. A bare key (one written with no '=' at all)
// yields the shared "true" sentinel that IsBareKey recognises, which is the one
// result that does not alias data.
//
// Raw means raw: do not pass the result to AppendUnescape without first
// establishing that the value was quoted, since escapes are meaningful only
// inside quotes. GetQuoted reports that, and AppendValue handles it for you. An absent key yields (nil, false); a key
// present with an empty value yields a non-nil empty slice and true, so the two
// stay distinguishable. Decode escapes with AppendUnescape, or use AppendValue
// for a one-call unescaped lookup.
//
// The result aliases data (treat it as read-only) and is valid only until data
// is modified. It has capacity equal to its length, so appending to it copies
// rather than overwriting the bytes that follow the value in data. (Iterate,
// which calls back once per field rather than once per lookup, does not cap
// what it hands the callback — capping there costs measurably.)
//
// Duplicate keys resolve as in AppendValue and GetMany: the first non-empty
// occurrence wins (iteration stops there); an empty value is returned only
// when the key never appears with a non-empty one.
//
// Get reports no syntax errors. It stops as soon as the key is settled, so a
// malformed tail beyond that point is never examined; what it can reach, it
// returns. Call Validate when you need the record checked.
func Get(data []byte, key string) ([]byte, bool) {
	val, _, found := GetQuoted(data, key)
	return val, found
}

// GetQuoted is Get plus the one fact Get throws away: whether the value was
// written as a double-quoted token.
//
// That matters because logfmt escape sequences are meaningful only inside
// quotes. An emitter writes path=C:\Users\bob unquoted and means every byte of
// it literally, so unescaping a value without knowing how it was written turns
// \U into U and \n into a newline. GetQuoted is the zero-copy way to decode
// correctly:
//
//	if v, quoted, ok := logfmt.GetQuoted(line, "msg"); ok {
//		if quoted && logfmt.NeedsUnescape(v) {
//			v = logfmt.AppendUnescape(buf[:0], v)
//		}
//		// v now aliases line (no copy) or buf (decoded)
//	}
//
// AppendValue does the same job in one call, at the cost of always copying.
//
// quoted is false for an absent key and for a bare key's implicit "true".
// Everything else — aliasing, capping, duplicate resolution, the absence of
// syntax errors — is exactly as described on Get.
func GetQuoted(data []byte, key string) (val []byte, quoted, found bool) {
	_ = iterate(data, func(k, v []byte, q bool) bool {
		if string(k) != key {
			return true
		}
		// cap == len, so a caller's append cannot reach into data.
		if len(v) > 0 {
			val, quoted, found = v[:len(v):len(v)], q, true
			return false // settled: first non-empty occurrence wins
		}
		if !found {
			// Provisional empty; keep looking for a non-empty one. Slicing a
			// non-nil slice keeps it non-nil even at zero length, so a
			// present-but-empty value stays distinct from an absent key.
			val, quoted, found = v[:len(v):len(v)], q, true
		}
		return true
	})
	return val, quoted, found
}

// GetMany looks up several keys in a single pass over data. It returns a slice
// the same length as keys, where the i-th element is the raw value for keys[i]
// (any surrounding quotes removed, escape sequences left intact), or nil if that
// key is not present. A present but empty value (for example from "key=") aliases
// data and is a non-nil zero-length slice, so it is distinct from a missing
// key's nil.
//
// A key matched by both an empty and a non-empty value resolves to the first
// non-empty one: an empty value is recorded only provisionally and is overridden
// by any later non-empty value for the same key.
//
// Entries in keys are expected to be distinct. A key listed twice is a
// degenerate case: each parsed field fills the first slot for that key not yet
// settled with a non-empty value, so duplicate slots are filled by successive
// occurrences in data, and when the key occurs fewer times than it is listed
// the extra slots stay nil — reading as absent even though the key is present.
//
// The returned values alias data (treat them as read-only) and are valid only
// until data is modified; each has capacity equal to its length, so appending
// to one copies rather than overwriting the bytes that follow it in data. A
// bare key yields the shared "true" sentinel, which does not alias data.
//
// GetMany does not report which values were quoted, so it cannot tell you which
// ones AppendUnescape may safely decode; use GetQuoted or AppendValue for keys
// whose values you intend to unescape. buf is reused as the result
// slice when it is large enough, avoiding a [][]byte allocation; pass back a
// previous result. If a key appears more than once with a non-empty value, the
// first such occurrence wins; iteration stops once every key has a non-empty
// value.
//
// Like Get, GetMany reports no syntax errors — it early-stops, so a malformed
// tail past the settled keys is never reached. Call Validate when you need the
// record checked.
//
// Each parsed field is matched against keys linearly, which is the fastest
// arrangement for the handful of keys these lookups are meant for. Measured on
// a 24-field line, GetMany stays ahead up to roughly ten keys; past that,
// Iterate with a map keyed by string(k) wins (20 keys: ~505 ns versus ~385 ns).
func GetMany(data []byte, keys []string, buf [][]byte) [][]byte {
	n := len(keys)
	if cap(buf) < n {
		buf = make([][]byte, n)
	}
	buf = buf[:n]

	// Reset slots to nil; a match fills its slot, so a slot left nil records a
	// missing key. A slot may hold a provisional empty value (non-nil, length
	// zero) that a later non-empty value for the same key replaces.
	clear(buf)

	remaining := n
	_ = iterate(data, func(k, v []byte, _ bool) bool {
		for j := range keys {
			// Length check first: a key already settled with a non-empty value
			// short-circuits cheaply on every later field, skipping the key
			// compare. Slots that are nil or hold a provisional empty value are
			// still open and fall through to the key compare.
			if len(buf[j]) > 0 || string(k) != keys[j] {
				continue
			}
			if len(v) > 0 {
				buf[j] = v[:len(v):len(v)] // cap == len, so a caller's append cannot reach into data
				remaining--                // settled: a non-empty value won't be overridden
			} else if buf[j] == nil {
				// Record the empty value, but keep looking. Slicing a non-nil
				// slice keeps it non-nil even at zero length, so a present-empty
				// value stays distinguishable from an absent key's nil.
				buf[j] = v[:len(v):len(v)]
			}
			break
		}
		return remaining > 0 // stop once every key has a non-empty value
	})
	return buf
}

// Validate parses data to completion and reports the first syntax error, or nil
// if the whole record is well-formed. The lookups deliberately do not do this:
// they early-stop, so they cannot see a fault past the keys they settled. Use
// Validate when a record's validity matters, and errors.Is(err, ErrBadFormat)
// or a *SyntaxError type assertion to inspect the result.
func Validate(data []byte) error {
	return iterate(data, func(key, val []byte, quoted bool) bool { return true })
}

// All returns an iterator over data's key/value pairs, for use with range:
//
//	for key, val := range logfmt.All(line) {
//		fmt.Printf("%s=%s\n", key, val)
//	}
//
// The pairs, their aliasing and the bare-key sentinel are exactly as described
// on Iterate. Ranging over a function requires Go 1.23 in the calling module;
// this package itself still builds on Go 1.21, where All is callable directly.
//
// A range loop yields two values, so All cannot report whether a value was
// quoted any more than it can report an error. Values that need unescaping are
// GetQuoted's and AppendValue's job.
//
// A range loop has nowhere to deliver an error, so All simply stops at a
// malformed field, having yielded the valid prefix. Call Validate if you need
// to know; use Iterate to get the error and the pairs in one pass.
func All(data []byte) func(yield func(key, val []byte) bool) {
	return func(yield func(key, val []byte) bool) {
		_ = iterate(data, func(key, val []byte, _ bool) bool { return yield(key, val) })
	}
}

// SplitRecord splits off the first logfmt record from data, returning it and
// the remainder. A record ends at the first '\n' (a trailing '\r' is trimmed,
// so CRLF input works); if there is no newline, the whole of data is the record
// and rest is nil.
//
// This package treats '\n' as ordinary whitespace, so a multi-line buffer
// handed straight to Iterate parses as one flat run of pairs with no record
// boundaries — and a lookup can match a key from a later line. Split first:
//
//	for len(data) > 0 {
//		var rec []byte
//		rec, data = logfmt.SplitRecord(data)
//		level, _ := logfmt.Get(rec, "level")
//		// ...
//	}
//
// The returned record aliases data and is capped to its length. It may be
// empty (for a blank line); Iterate and the lookups handle that as a record
// with no pairs.
func SplitRecord(data []byte) (record, rest []byte) {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		record, rest = data[:i], data[i+1:]
	} else {
		record = data
	}
	if n := len(record); n > 0 && record[n-1] == '\r' {
		record = record[:n-1]
	}
	return record[:len(record):len(record)], rest
}
