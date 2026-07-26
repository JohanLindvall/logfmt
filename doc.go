// Package logfmt provides a fast, allocation-free reader for the logfmt
// key/value line format:
//
//	level=info msg="user login" user=john id=42 success=true
//
// # API
//
// Iterate is the core primitive: it walks a line and hands each key/value pair
// to a callback as sub-slices of the input, performing no allocations. Values
// are raw — reported exactly as they appear in the input, with surrounding
// quotes stripped but escape sequences left intact. On top of it:
//
//   - Get returns the raw value for one key (zero-copy).
//   - GetMany returns the raw values for several keys in a single pass,
//     stopping early once all are found; a missing key yields nil, while a
//     present-but-empty value is a non-nil empty slice.
//   - GetValue returns the unescaped value for one key, decoding into a
//     caller-provided buffer only when needed.
//   - Unescape decodes escape sequences (\n, \r, \t, and JSON-style \uXXXX);
//     NeedsUnescape reports whether decoding would change anything.
//   - ParseTime parses the timestamp formats that commonly appear in logfmt.
//
// All three lookups resolve duplicate keys the same way: the first non-empty
// occurrence wins, and an empty value is used only when the key never appears
// with a non-empty one.
//
// # Records and framing
//
// The parser has no notion of a record boundary: '\n' and '\r' are ordinary
// whitespace, exactly like ' '. Passing a multi-line buffer to Iterate
// therefore yields the pairs of every line as one flat sequence, with no
// indication of where one line ended and the next began — and a lookup will
// happily match a key from a later line. Callers that need per-record
// semantics must split the input themselves (bytes.IndexByte(data, '\n'),
// bufio.Scanner, and so on) and call Iterate once per line. Feeding whole
// buffers in is supported and fast, but only when the flat view is what you
// want.
//
// # Aliasing and concurrency
//
// Returned slices alias the input, a caller-provided buffer, or (for bare
// keys) a shared package-level constant — treat them as read-only, and copy
// any that must outlive the input.
//
// Read-only means what it says: these slices are windows onto a larger array,
// so their capacity generally runs to the end of that array. Appending to one
// writes into the bytes that follow it — for a value from Get that is the rest
// of the log line, and for a bare key's "true" it is a constant shared by every
// caller in the process. Copy first (append(dst[:0], v...), string(v)) or
// re-slice to v[:len(v):len(v)] before appending.
//
// The package holds no state, so it is safe for concurrent use as long as
// callers honour that rule.
//
// # Errors
//
// The only malformed inputs are an unterminated quoted value and a closing
// quote followed by a non-space byte; both yield ErrBadFormat. Because parsing
// is streaming, Iterate has already delivered every pair preceding the fault
// before it returns that error — treat the callback's output as a valid prefix,
// not as something to discard.
//
// For the same reason the lookups report malformation only on a best-effort
// basis: Get, GetValue and GetMany stop as soon as their keys are settled, so a
// malformed tail later in the line is never reached and no error is reported.
// A nil error from a lookup means "your keys were resolved", not "the whole
// line is well-formed". Use Iterate to full completion if you need validation.
//
// Get and GetValue return ErrKeyNotFound for an absent key. GetMany instead
// reports absence as a nil slot, which is how it distinguishes an absent key
// from one present with an empty value.
//
// # Leniency
//
// The parser is deliberately lenient — built for reading real-world logs, it
// never rejects input it can read as key/value pairs. This differs from
// go-logfmt in a few ways:
//
//   - A '"' inside an unquoted value is a literal byte, not a syntax error
//     (a=x" b=c yields a="x"" and b="c").
//   - Unknown escapes decode leniently (the escaped byte itself) instead of
//     being rejected, and a malformed \uXXXX is kept verbatim.
//   - Control bytes other than whitespace (0x00–0x08, 0x0E–0x1F) are ordinary
//     key/value bytes.
//   - Keys are never unquoted: "a b"=c parses as the bare key `"a` and the
//     pair `b"`=c. Quoting is meaningful only in value position, immediately
//     after '='.
//   - 'key=' followed by whitespace is an empty value, and the following token
//     starts a new pair.
//
// A bare key with no '=' is reported with the value "true", matching logfmt
// convention for boolean flags.
//
// # Scope
//
// This is a reader only, by design. There is no encoder, no io.Reader-based
// streaming decoder, no typed accessors (integers, booleans, durations) and no
// map-building convenience: values come back as []byte for the caller to
// convert. That keeps the package dependency-free, allocation-free and its
// semantics small enough to fuzz against a reference implementation. Write
// logfmt with go-logfmt or your logging library's encoder; convert values with
// strconv.
package logfmt
