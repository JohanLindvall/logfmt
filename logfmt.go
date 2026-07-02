package logfmt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/bits"
	"unicode/utf16"
	"unicode/utf8"
)

// ErrBadFormat is returned when the input is not valid logfmt, for example a
// quoted value that is never closed or that is followed by a non-space byte.
var ErrBadFormat = errors.New("bad logfmt format")

// ErrKeyNotFound is returned by Get and GetValue when the requested key is
// absent.
var ErrKeyNotFound = errors.New("key not found")

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

// hasByte flags every byte of w equal to c.
func hasByte(w uint64, c byte) uint64 {
	x := w ^ (swarLo * uint64(c))
	return (x - swarLo) &^ x & swarHi
}

// hasCtrlOrSpace flags every byte of w that is <= 0x20. This covers all logfmt
// whitespace ('\t'..'\r' and ' '); the only other bytes it flags are control
// bytes 0x00..0x08 and 0x0E..0x1F, which the caller rules out by re-checking
// the located byte. UTF-8 continuation/lead bytes (>= 0x80) are never flagged.
func hasCtrlOrSpace(w uint64) uint64 {
	return (w - swarLo*0x21) &^ w & swarHi
}

// Iterate parses data as a logfmt record and calls fn once for each key/value
// pair, in order. key and val are sub-slices that alias data — except for a
// bare key with no '=' (for example "debug", or a trailing token), whose val is
// a shared constant "true". Treat both as read-only, and copy them if they must
// outlive the call.
//
// Whitespace after '=' is skipped, so "key= value" yields ("key", "value");
// go-logfmt instead reports an empty value and a separate bare key. A quoted
// value is returned without its surrounding double quotes but is NOT
// unescaped — backslash escapes are left intact; pass val to Unescape to
// decode them.
//
// fn may return false to stop iteration early, in which case Iterate returns
// nil. Iterate returns ErrBadFormat if data contains a malformed quoted value,
// and otherwise nil. It performs no allocations.
func Iterate(data []byte, fn func(key, val []byte) bool) error {
	for i, n := 0, len(data); i < n; {
		for i < n && isSpace(data[i]) {
			i++
		}

		kStart := i
		for i+8 <= n {
			w := binary.LittleEndian.Uint64(data[i : i+8])
			m := hasCtrlOrSpace(w) | hasByte(w, '=')
			if m != 0 {
				i += bits.TrailingZeros64(m) >> 3
				// '=' first: keys overwhelmingly end there, so the cheap
				// compare short-circuits past the isSpace call.
				if c := data[i]; c == '=' || isSpace(c) {
					goto keyEnd
				}
				break // rare non-whitespace control byte; finish scalar
			}
			i += 8
		}
		for i < n && !isSpace(data[i]) && data[i] != '=' {
			i++
		}
	keyEnd:

		if i >= n {
			if kStart < n {
				fn(data[kStart:n], trueSlice)
			}
			return nil
		}

		kEnd := i

		if data[i] != '=' {
			if !fn(data[kStart:i], trueSlice) {
				return nil
			}
			continue
		}
		i++

		for i < n && isSpace(data[i]) {
			i++
		}

		vStart, vEnd := i, i

		if i >= n {
			fn(data[kStart:kEnd], data[vStart:vEnd])
			return nil
		}

		if data[i] == '"' {
			i++
			vStart = i
			for {
				q := bytes.IndexByte(data[i:], '"')
				if q == -1 {
					return ErrBadFormat
				}
				i += q

				// Determine whether this quote is escaped by counting the
				// run of backslashes immediately preceding it: an odd count
				// means the quote is escaped and we keep scanning.
				bs := 0
				for j := i - 1; j >= vStart && data[j] == '\\'; j-- {
					bs++
				}
				if bs%2 == 1 {
					i++
					continue
				}

				vEnd = i
				i++
				if i < n {
					// ' ' first: the usual delimiter short-circuits past the
					// isSpace table load.
					if c := data[i]; c != ' ' && !isSpace(c) {
						return ErrBadFormat
					}
					i++
				}
				break
			}
		} else {
			vStart = i
			for i+8 <= n {
				w := binary.LittleEndian.Uint64(data[i : i+8])
				m := hasCtrlOrSpace(w)
				if m != 0 {
					i += bits.TrailingZeros64(m) >> 3
					// ' ' first: it is the usual value delimiter, so the cheap
					// compare short-circuits past the isSpace call.
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
		}

		if !fn(data[kStart:kEnd], data[vStart:vEnd]) {
			return nil
		}
	}

	return nil
}

// Unescape decodes the backslash escapes in a raw logfmt value and appends
// the result to dst, returning the extended slice. It recognises \n, \r, \t
// and JSON-style \uXXXX unicode escapes (including surrogate pairs, as emitted
// by go-logfmt for control characters); any other escaped byte (such as \" or
// \\) is emitted as the byte itself. A lone surrogate half decodes to U+FFFD,
// matching encoding/json. A malformed \u (bad or truncated hex) and a trailing
// lone backslash are kept verbatim rather than rejected.
//
// Pass dst[:0] to reuse an existing buffer and avoid allocation. As a fast path,
// when raw contains no escapes and dst is empty, raw is returned unchanged with
// no copy (the result then aliases raw); so callers may invoke it
// unconditionally without a NeedsUnescape pre-check.
func Unescape(dst []byte, raw []byte) []byte {
	i, n := 0, len(raw)
	for i < n {
		q := bytes.IndexByte(raw[i:], '\\')
		if q < 0 {
			if i == 0 && len(dst) == 0 {
				return raw
			}
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

// GetValue returns the unescaped value associated with key in data. When the
// value needs decoding it is written into dst (pass dst[:0] to reuse its
// backing array); when it needs none, a sub-slice of data is returned directly
// without copying. The result therefore aliases either dst or data and is valid
// only until whichever it aliases is modified — so copy it if it must outlive
// that. (Use the returned slice, not dst, since dst may be untouched.)
//
// Duplicate keys resolve exactly as in Get and GetMany: the first non-empty
// occurrence wins, an empty value is returned only when no non-empty one
// exists. GetValue returns ErrKeyNotFound if key is absent, or ErrBadFormat if
// data is malformed.
func GetValue(data []byte, key string, dst []byte) ([]byte, error) {
	raw, err := Get(data, key)
	if err != nil {
		return nil, err
	}
	// Unescape short-circuits to return raw (aliasing data) when it has no
	// escapes, so this is zero-copy in the common case and decodes into dst
	// only when needed.
	return Unescape(dst[:0], raw), nil
}

// NeedsUnescape reports whether raw contains a backslash escape, i.e. whether
// passing it through Unescape would change it. Values returned by Iterate,
// Get and GetMany are raw; use this to skip the unescape step (and its copy)
// when decoding is unnecessary.
func NeedsUnescape(raw []byte) bool {
	return bytes.IndexByte(raw, '\\') >= 0
}

// Get returns the raw value for key in data: the value as it appears in the
// input, with any surrounding quotes removed but escape sequences left intact.
// The result aliases data (treat it as read-only) and is valid only until data
// is modified; decode escapes with Unescape if needed. It returns
// ErrKeyNotFound if the key is absent, or use GetValue for an unescaped,
// buffer-reusing lookup.
//
// Duplicate keys resolve as in GetValue and GetMany: the first non-empty
// occurrence wins (iteration stops there); an empty value is returned only
// when the key never appears with a non-empty one.
func Get(data []byte, key string) ([]byte, error) {
	var rawVal []byte // nil = not found; may hold a provisional empty value

	err := Iterate(data, func(k, v []byte) bool {
		if string(k) != key {
			return true
		}
		if len(v) > 0 {
			rawVal = v
			return false // settled: first non-empty occurrence wins
		}
		if rawVal == nil {
			rawVal = v // provisional empty; keep looking for a non-empty one
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if rawVal == nil {
		return nil, ErrKeyNotFound
	}
	return rawVal, nil
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
// The returned values alias data (treat them as read-only) and are valid only
// until data is modified; decode escapes with Unescape if needed. buf is reused
// as the result slice when it is large enough, avoiding a [][]byte allocation;
// pass back a previous result. If a key appears more than once with a non-empty
// value, the first such occurrence wins; iteration stops once every key has a
// non-empty value. ErrBadFormat is returned if data is malformed.
//
// Each parsed field is matched against keys linearly, so lookups are intended
// for small key sets (a handful of keys); for large sets, use Iterate with a
// map keyed by string(k).
func GetMany(data []byte, keys []string, buf [][]byte) ([][]byte, error) {
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
	err := Iterate(data, func(k, v []byte) bool {
		for j := range keys {
			// Length check first: a key already settled with a non-empty value
			// short-circuits cheaply on every later field, skipping the key
			// compare. Slots that are nil or hold a provisional empty value are
			// still open and fall through to the key compare.
			if len(buf[j]) > 0 || string(k) != keys[j] {
				continue
			}
			if len(v) > 0 {
				buf[j] = v
				remaining-- // settled: a non-empty value won't be overridden
			} else if buf[j] == nil {
				buf[j] = v // record the empty value, but keep looking
			}
			break
		}
		return remaining > 0 // stop once every key has a non-empty value
	})
	if err != nil {
		return nil, err
	}
	return buf, nil
}
