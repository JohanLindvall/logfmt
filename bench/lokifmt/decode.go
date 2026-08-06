// Package lokifmt is a benchmark stand-in for Grafana Loki's in-tree logfmt
// decoder (pkg/logql/log/logfmt): go-logfmt's scanner operating on a caller
// supplied []byte line instead of an io.Reader, with Reset for reuse.
//
// It is adapted directly from go-logfmt/logfmt v0.6.1 (MIT — see the LICENSE
// file in this directory), which is also what Loki's decoder is, so no
// Loki-licensed code is vendored. The scanning loops — the perf-relevant part
// — are identical to go-logfmt's (and therefore to Loki's), and Loki's one
// behavioural edit is mirrored as a deletion (unquoteBytes accepts control
// bytes; see jsonstring.go). Loki's post-error resync (skip_value/EOL) is not
// reproduced: it is unobservable through the ScanKeyval loop the benchmarks
// drive, which stops at the first false. Differentially verified against the
// previously vendored Loki copy: identical pairs, error presence, message and
// position on the benchmark samples and a malformed-input battery, with
// identical allocation counts and timings equal within noise (the big-line
// row measured ~3% faster with the dead resync code gone — pinned interleaved
// A/B, n=6, control clean).
package lokifmt

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

// A Decoder decodes logfmt key/value pairs from a single record held in
// memory.
type Decoder struct {
	pos   int
	key   []byte
	value []byte
	line  []byte
	err   error
}

// NewDecoder returns a new decoder that reads from line.
func NewDecoder(line []byte) *Decoder {
	return &Decoder{line: line}
}

// Reset rewinds the decoder onto a new line, reusing the allocation.
func (dec *Decoder) Reset(line []byte) {
	dec.pos = 0
	dec.line = line
	dec.err = nil
}

// ScanKeyval advances the Decoder to the next key/value pair of the record,
// which can then be retrieved with the Key and Value methods. It returns
// false when decoding stops, either by reaching the end of the record or an
// error.
func (dec *Decoder) ScanKeyval() bool {
	dec.key, dec.value = nil, nil
	if dec.err != nil {
		return false
	}

	line := dec.line

	// garbage
	for p, c := range line[dec.pos:] {
		if c > ' ' {
			dec.pos += p
			goto key
		}
	}
	dec.pos = len(line)
	return false

key:
	const invalidKeyError = "invalid key"

	start, multibyte := dec.pos, false
	for p, c := range line[dec.pos:] {
		switch {
		case c == '=':
			dec.pos += p
			if dec.pos > start {
				dec.key = line[start:dec.pos]
				if multibyte && bytes.ContainsRune(dec.key, utf8.RuneError) {
					dec.syntaxError(invalidKeyError)
					return false
				}
			}
			if dec.key == nil {
				dec.unexpectedByte(c)
				return false
			}
			goto equal
		case c == '"':
			dec.pos += p
			dec.unexpectedByte(c)
			return false
		case c <= ' ':
			dec.pos += p
			if dec.pos > start {
				dec.key = line[start:dec.pos]
				if multibyte && bytes.ContainsRune(dec.key, utf8.RuneError) {
					dec.syntaxError(invalidKeyError)
					return false
				}
			}
			return true
		case c >= utf8.RuneSelf:
			multibyte = true
		}
	}
	dec.pos = len(line)
	if dec.pos > start {
		dec.key = line[start:dec.pos]
		if multibyte && bytes.ContainsRune(dec.key, utf8.RuneError) {
			dec.syntaxError(invalidKeyError)
			return false
		}
	}
	return true

equal:
	dec.pos++
	if dec.pos >= len(line) {
		return true
	}
	switch c := line[dec.pos]; {
	case c <= ' ':
		return true
	case c == '"':
		goto qvalue
	}

	// value
	start = dec.pos
	for p, c := range line[dec.pos:] {
		switch {
		case c == '=' || c == '"':
			dec.pos += p
			dec.unexpectedByte(c)
			return false
		case c <= ' ':
			dec.pos += p
			if dec.pos > start {
				dec.value = line[start:dec.pos]
			}
			return true
		}
	}
	dec.pos = len(line)
	if dec.pos > start {
		dec.value = line[start:dec.pos]
	}
	return true

qvalue:
	const (
		untermQuote  = "unterminated quoted value"
		invalidQuote = "invalid quoted value"
	)

	hasEsc, esc := false, false
	start = dec.pos
	for p, c := range line[dec.pos+1:] {
		switch {
		case esc:
			esc = false
		case c == '\\':
			hasEsc, esc = true, true
		case c == '"':
			dec.pos += p + 2
			if hasEsc {
				v, ok := unquoteBytes(line[start:dec.pos])
				if !ok {
					dec.syntaxError(invalidQuote)
					return false
				}
				dec.value = v
			} else {
				start++
				end := dec.pos - 1
				if end > start {
					dec.value = line[start:end]
				}
			}
			return true
		}
	}
	dec.pos = len(line)
	dec.syntaxError(untermQuote)
	return false
}

// Key returns the most recent key found by a call to ScanKeyval. The returned
// slice may point to internal buffers and is only valid until the next call
// to Reset. It does no allocation.
func (dec *Decoder) Key() []byte {
	return dec.key
}

// Value returns the most recent value found by a call to ScanKeyval. The
// returned slice may point to internal buffers and is only valid until the
// next call to Reset. It does no allocation when the value has no escape
// sequences.
func (dec *Decoder) Value() []byte {
	return dec.value
}

// Err returns the first error that was encountered by the Decoder.
func (dec *Decoder) Err() error {
	return dec.err
}

func (dec *Decoder) syntaxError(msg string) {
	dec.err = &SyntaxError{
		Msg: msg,
		Pos: dec.pos + 1,
	}
}

func (dec *Decoder) unexpectedByte(c byte) {
	dec.err = &SyntaxError{
		Msg: fmt.Sprintf("unexpected %q", c),
		Pos: dec.pos + 1,
	}
}

// A SyntaxError represents a syntax error in the logfmt input.
type SyntaxError struct {
	Msg string
	Pos int
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("logfmt syntax error at pos %d: %s", e.Pos, e.Msg)
}
