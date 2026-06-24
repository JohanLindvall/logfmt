// Package logfmt provides a fast, allocation-free reader for the logfmt
// key/value line format (key=value key2="quoted value" ...).
//
// Iterate is the core primitive: it walks a line and hands each key/value pair
// to a callback as sub-slices of the input, performing no allocations. Values
// are reported exactly as they appear in the input (quotes stripped but escape
// sequences left intact); UnescapeInto decodes those escapes when needed, and
// GetValue combines the two to look up and unescape a single key.
package logfmt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/bits"
)

// ErrBadFormat is returned when the input is not valid logfmt, for example a
// quoted value that is never closed or that is followed by a non-space byte.
var ErrBadFormat = errors.New("bad logfmt format")

// ErrKeyNotFound is returned by GetValue when the requested key is absent.
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

// Iterate parses buf as a logfmt record and calls fn once for each key/value
// pair, in order. Both key and val are sub-slices that alias buf; they are
// only valid until buf is modified, so copy them if they must outlive the
// call.
//
// A bare key with no '=' (for example "debug", or a trailing token) is
// reported with val set to the literal "true". A quoted value is returned
// without its surrounding double quotes but is NOT unescaped — backslash
// escapes are left intact; pass val to UnescapeInto to decode them.
//
// fn may return false to stop iteration early, in which case Iterate returns
// nil. Iterate returns ErrBadFormat if buf contains a malformed quoted value,
// and otherwise nil. It performs no allocations.
func Iterate(buf []byte, fn func(key, val []byte) bool) error {
	for i, n := 0, len(buf); i < n; {
		for i < n && isSpace(buf[i]) {
			i++
		}

		kStart := i
		for i+8 <= n {
			w := binary.LittleEndian.Uint64(buf[i : i+8])
			m := hasCtrlOrSpace(w) | hasByte(w, '=')
			if m != 0 {
				i += bits.TrailingZeros64(m) >> 3
				// '=' first: keys overwhelmingly end there, so the cheap
				// compare short-circuits past the isSpace call.
				if c := buf[i]; c == '=' || isSpace(c) {
					goto keyEnd
				}
				break // rare non-whitespace control byte; finish scalar
			}
			i += 8
		}
		for i < n && !isSpace(buf[i]) && buf[i] != '=' {
			i++
		}
	keyEnd:

		if i >= n {
			if kStart < n {
				fn(buf[kStart:n], trueSlice)
			}
			return nil
		}

		kEnd := i

		if buf[i] != '=' {
			if !fn(buf[kStart:i], trueSlice) {
				return nil
			}
			continue
		}
		i++

		for i < n && isSpace(buf[i]) {
			i++
		}

		vStart, vEnd := i, i

		if i >= n {
			fn(buf[kStart:kEnd], buf[vStart:vEnd])
			return nil
		}

		if buf[i] == '"' {
			i++
			vStart = i
			for {
				q := bytes.IndexByte(buf[i:], '"')
				if q == -1 {
					return ErrBadFormat
				}
				i += q

				// Determine whether this quote is escaped by counting the
				// run of backslashes immediately preceding it: an odd count
				// means the quote is escaped and we keep scanning.
				bs := 0
				for j := i - 1; j >= vStart && buf[j] == '\\'; j-- {
					bs++
				}
				if bs%2 == 1 {
					i++
					continue
				}

				vEnd = i
				i++
				if i < n {
					if !isSpace(buf[i]) {
						return ErrBadFormat
					}
					i++
				}
				break
			}
		} else {
			vStart = i
			for i+8 <= n {
				w := binary.LittleEndian.Uint64(buf[i : i+8])
				m := hasCtrlOrSpace(w)
				if m != 0 {
					i += bits.TrailingZeros64(m) >> 3
					// ' ' first: it is the usual value delimiter, so the cheap
					// compare short-circuits past the isSpace call.
					if c := buf[i]; c == ' ' || isSpace(c) {
						goto valEnd
					}
					break // rare non-whitespace control byte; finish scalar
				}
				i += 8
			}
			for i < n && !isSpace(buf[i]) {
				i++
			}
		valEnd:
			vEnd = i
		}

		if !fn(buf[kStart:kEnd], buf[vStart:vEnd]) {
			return nil
		}
	}

	return nil
}

// UnescapeInto decodes the backslash escapes in a raw logfmt value and appends
// the result to dst, returning the extended slice. It recognises \n, \r and
// \t; any other escaped byte (such as \" or \\) is emitted as the byte itself.
// A trailing lone backslash is kept verbatim.
//
// Pass dst[:0] to reuse an existing buffer and avoid allocation. As a fast path,
// when raw contains no escapes and dst is empty, raw is returned unchanged with
// no copy (the result then aliases raw); so callers may invoke it
// unconditionally without a NeedsUnescape pre-check.
func UnescapeInto(dst []byte, raw []byte) []byte {
	if len(dst) == 0 && bytes.IndexByte(raw, '\\') < 0 {
		return raw
	}
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

// GetValue returns the unescaped value associated with key in a single logfmt
// line. When the value needs decoding it is written into dst (pass dst[:0] to
// reuse its backing array); when it needs none, a sub-slice of line is returned
// directly without copying. The result therefore aliases either dst or line and
// is valid only until whichever it aliases is modified — so copy it if it must
// outlive that. (Use the returned slice, not dst, since dst may be untouched.)
//
// The first matching key wins; iteration stops there. GetValue returns
// ErrKeyNotFound if key is absent, or ErrBadFormat if the line is malformed.
func GetValue(line []byte, key []byte, dst []byte) ([]byte, error) {
	var found bool
	var rawVal []byte

	err := Iterate(line, func(k, v []byte) bool {
		if bytes.Equal(k, key) {
			found = true
			rawVal = v
			return false // stop early
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrKeyNotFound
	}
	// UnescapeInto short-circuits to return rawVal (aliasing line) when it has
	// no escapes, so this is zero-copy in the common case and decodes into dst
	// only when needed.
	return UnescapeInto(dst[:0], rawVal), nil
}

// NeedsUnescape reports whether raw contains a backslash escape, i.e. whether
// passing it through UnescapeInto would change it. Values returned by Iterate,
// Get and GetMany are raw; use this to skip the unescape step (and its copy)
// when decoding is unnecessary.
func NeedsUnescape(raw []byte) bool {
	return bytes.IndexByte(raw, '\\') >= 0
}

// Get returns the raw value for key in data: the value as it appears in the
// input, with any surrounding quotes removed but escape sequences left intact.
// The result aliases data and is valid only until data is modified; decode
// escapes with UnescapeInto if needed. It returns ErrKeyNotFound if the key is
// absent, or use GetValue for an unescaped, buffer-reusing lookup.
func Get(data []byte, key string) ([]byte, error) {
	var found bool
	var rawVal []byte

	err := Iterate(data, func(k, v []byte) bool {
		if string(k) == key {
			found = true
			rawVal = v
			return false // stop early
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	if !found {
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
// The returned values alias data and are valid only until data is modified;
// decode escapes with UnescapeInto if needed. buf is reused as the result slice
// when it is large enough, avoiding a [][]byte allocation; pass back a previous
// result. If a key appears more than once in data, the first occurrence wins;
// iteration stops once every key has been found. ErrBadFormat is returned if
// data is malformed.
func GetMany(data []byte, keys []string, buf [][]byte) ([][]byte, error) {
	n := len(keys)
	if cap(buf) < n {
		buf = make([][]byte, n)
	}
	buf = buf[:n]

	// Reset slots to nil; a match fills its slot, so a slot left nil records a
	// missing key. Found values alias data and are never nil, even when empty.
	for j := range buf {
		buf[j] = nil
	}

	remaining := n
	err := Iterate(data, func(k, v []byte) bool {
		for j := range keys {
			// nil check first: once a key is found it short-circuits cheaply on
			// every later field, skipping the key compare. It also makes the
			// first match win (a duplicate key sees a filled, non-nil slot).
			if buf[j] != nil || string(k) != keys[j] {
				continue
			}
			buf[j] = v
			remaining--
			break
		}
		return remaining > 0 // stop once every key is found
	})
	if err != nil {
		return nil, err
	}
	return buf, nil
}
