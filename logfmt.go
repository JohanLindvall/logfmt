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
	"errors"
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
		for i < n && !isSpace(buf[i]) && buf[i] != '=' {
			i++
		}

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
			for i < n && !isSpace(buf[i]) {
				i++
			}
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
// Pass dst[:0] to reuse an existing buffer and avoid allocation.
func UnescapeInto(dst []byte, raw []byte) []byte {
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
// line. The result is written into dst (pass dst[:0] to reuse its backing
// array) and the returned slice may alias dst.
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
	if bytes.ContainsAny(rawVal, "\\ \"") {
		return UnescapeInto(dst[:0], rawVal), nil
	}
	return append(dst[:0], rawVal...), nil
}
