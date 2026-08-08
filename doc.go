// Package logfmt provides a fast, allocation-free reader for the logfmt
// key/value line format:
//
//	level=info msg="user login" user=john id=42 success=true
//
// # API
//
// Iterate is the core primitive: it walks a line and hands each key/value pair
// to a callback as sub-slices of the input, allocating nothing on a well-formed
// record. Values are raw — reported exactly as they appear in the input, with
// surrounding quotes stripped but escape sequences left intact. On top of it:
//
//   - All is the same walk as a range-over-func iterator:
//     for key, val := range logfmt.All(line). Ranging needs Go 1.23 in the
//     calling module; the package itself still builds at Go 1.21.
//   - Get returns the raw value for one key (zero-copy) and whether it was
//     present.
//   - GetMany returns the raw values for several keys in a single pass,
//     stopping early once all are found; a missing key yields nil, while a
//     present-but-empty value is a non-nil empty slice.
//   - GetQuoted is Get plus whether the value was written quoted — the bit that
//     decides whether unescaping it is correct.
//   - AppendValue appends one key's unescaped value to a caller-provided
//     buffer, decoding only values that were quoted.
//   - AppendUnescape decodes escape sequences (\n, \r, \t, and JSON-style
//     \uXXXX) in a quoted value; NeedsUnescape reports whether raw holds a
//     backslash at all, so the decode can be skipped when it cannot.
//   - SplitRecord peels one record off a multi-line buffer (see Records and
//     framing below).
//   - Validate parses a record to completion and reports the first syntax
//     error, which the early-stopping lookups cannot.
//   - IsBareKey distinguishes a bare key's implicit "true" from a real one.
//   - ParseTime parses the timestamp formats that commonly appear in logfmt.
//
// The lookups — Get, GetQuoted, GetMany and AppendValue — resolve duplicate
// keys the same way: the first non-empty occurrence wins, and an empty value is
// used only when the key never appears with a non-empty one. Data and values are
// []byte throughout — nothing asks the caller to convert input or results to
// string; only the lookup keys, in practice compile-time constants, are
// strings.
//
// # Escapes belong to quoted values only
//
// A backslash means something only inside quotes. msg="a\nb" holds an escape;
// path=C:\Users\bob holds three literal backslashes, and go-logfmt's encoder
// writes it exactly that way, because a backslash is not one of the bytes that
// force quoting. Unescaping without knowing which of the two you have turns
// C:\Users\bob into C:Usersbob with an embedded newline — silently, since every
// byte of it is valid logfmt.
//
// Iterate, All, Get and GetMany hand out quoted and unquoted values alike and do
// not distinguish them, so the raw value alone cannot tell you which it was. Two entry points
// carry the missing bit:
//
//   - AppendValue decodes for you, and only when the value was quoted.
//   - GetQuoted returns the bit, for callers who want to skip the copy when
//     nothing needs decoding.
//
// Reach for AppendUnescape directly only on a value you already know was
// quoted.
//
// # Records and framing
//
// The parser has no notion of a record boundary: '\n' and '\r' are ordinary
// whitespace, exactly like ' '. Passing a multi-line buffer to Iterate
// therefore yields the pairs of every line as one flat sequence, with no
// indication of where one line ended and the next began — and a lookup will
// happily match a key from a later line. Callers that need per-record
// semantics must split the input themselves. SplitRecord does it without
// allocating, and handles CRLF:
//
//	for len(data) > 0 {
//		var rec []byte
//		rec, data = logfmt.SplitRecord(data)
//		level, _ := logfmt.Get(rec, "level")
//		// ...
//	}
//
// bufio.Scanner works too when the input arrives as a stream. Feeding whole
// buffers to Iterate is supported and fast, but only when the flat view is
// what you want.
//
// # Aliasing and concurrency
//
// Returned slices alias the input, a caller-provided buffer, or (for bare
// keys) a shared package-level constant — treat them as read-only, and copy
// any that must outlive the input. The bare-key sentinel is the one result that
// does not alias the input at all: it outlives and ignores any change to it, and
// IsBareKey reports true for it whether it arrived through a callback or through
// Get, GetQuoted or GetMany.
//
// Read-only means what it says: these slices are windows onto the input, and a
// bare key's "true" is a constant shared by every caller in the process, so
// writing through any of them corrupts something you do not own.
//
// Appending is handled differently by the entry points, deliberately. Get,
// GetQuoted and GetMany return values capped to their length, so appending to one copies
// rather than overwriting the bytes that follow it in the input; they cap once
// per lookup, which costs nothing measurable. AppendValue and AppendUnescape
// always copy into the caller's buffer, so their results never alias the input
// at all. Iterate and All do not cap what they hand the callback to the value's
// own length, because that lands once per field and measures ~4.5% on
// field-dense input; they do bound it at the end of the record, so an append
// cannot run past the data you handed in, but it can still overwrite later
// fields of the same record. Inside a callback, copy first
// (append(dst[:0], v...), string(v)) or re-slice to v[:len(v):len(v)] before
// appending.
//
// The package holds no state, so it is safe for concurrent use as long as
// callers honour that rule.
//
// # Errors
//
// The only malformed inputs are an unterminated quoted value and a closing
// quote followed by a non-space byte. Iterate and Validate report both as a
// *SyntaxError carrying the byte offset; errors.Is(err, ErrBadFormat) matches
// any of them, and a *SyntaxError type assertion or errors.As gets the offset.
// Note that the returned error is never ErrBadFormat itself, so compare with
// errors.Is rather than ==. Because parsing is streaming, Iterate has already
// delivered every pair preceding the fault before it returns — treat the
// callback's output as a valid prefix, not as something to discard.
//
// That *SyntaxError is the one allocation this package makes on its own behalf.
// Every entry point is allocation-free on a well-formed record; a malformed one
// costs a single 24-byte error value, and the lookups pay it only when they have
// to walk past the fault to settle their keys (they discard it, since they
// report no errors). "Allocation-free" throughout this documentation means
// exactly that.
//
// The lookups report no syntax errors at all. Get, GetQuoted, AppendValue and
// GetMany stop as soon as their keys are settled, so a malformed tail beyond that point is
// never examined; reporting an error they cannot reliably detect would promise
// a validation they do not perform. They return what the reachable prefix
// yields. Call Validate when a record's validity matters.
//
// Absence is uniform: Get, GetQuoted and AppendValue return false, GetMany
// leaves the slot nil. A key present with an empty value stays distinct from an
// absent one — Get returns true with a non-nil empty slice, GetMany a non-nil
// empty slot.
//
// # Leniency
//
// The parser is deliberately lenient — built for reading real-world logs, it
// never rejects input it can read as key/value pairs. This differs from
// go-logfmt in a few ways:
//
//   - A '"' inside an unquoted value is a literal byte, not a syntax error
//     (a=x" b=c yields a="x"" and b="c").
//   - A '\' inside an unquoted value is a literal byte too, never an escape
//     introducer (a=C:\n yields the four bytes C:\n, not C: followed by a
//     newline). go-logfmt agrees; see "Escapes belong to quoted values only".
//   - Unknown escapes decode leniently (the escaped byte itself) instead of
//     being rejected, and a malformed \uXXXX is kept verbatim.
//   - Control bytes other than whitespace (0x00–0x08, 0x0E–0x1F) are ordinary
//     key/value bytes.
//   - Keys are never unquoted: "a b"=c parses as the bare key `"a` and the
//     pair `b"`=c. Quoting is meaningful only in value position, immediately
//     after '='.
//   - 'key=' followed by whitespace is an empty value, and the following token
//     starts a new pair.
//   - A '=' with no key before it yields a pair with an EMPTY key: =v parses
//     as the pair ""="v", where go-logfmt reports "unexpected '='". A lookup
//     for the key "" can therefore genuinely match.
//   - A '=' inside an unquoted value is a literal byte: a==b parses as the
//     pair "a"="=b", which go-logfmt also rejects. Only the first '=' of a
//     field separates key from value.
//
// A bare key with no '=' is reported with the value "true", matching logfmt
// convention for boolean flags. That value is a shared sentinel, so IsBareKey
// can tell "debug" from "debug=true" — by content the two are identical.
//
// # Scope
//
// This is a reader only, by design. There is no encoder, no io.Reader-based
// streaming decoder, no typed accessors (integers, booleans, durations) and no
// map-building convenience: values come back as []byte for the caller to
// convert. ParseTime is the one concession, because timestamp formats in real
// logs vary enough to be worth centralising. That keeps the package
// dependency-free, allocation-free and its semantics small enough to fuzz
// against a reference implementation. Write logfmt with go-logfmt or your
// logging library's encoder; convert values with strconv.
package logfmt
