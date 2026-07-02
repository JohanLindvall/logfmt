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
// # Aliasing and concurrency
//
// Returned slices alias the input, a caller-provided buffer, or (for bare
// keys) a shared constant — treat them as read-only, and copy any that must
// outlive the input. The package holds no state, so it is safe for concurrent
// use as long as callers honour that rule.
//
// # Leniency
//
// The parser is deliberately lenient — built for reading real-world logs, it
// never rejects input it can read as key/value pairs. The only errors are an
// unterminated quoted value and a closing quote followed by a non-space byte.
// This differs from go-logfmt in a few ways:
//
//   - Whitespace after '=' is skipped: "key= value" parses as ("key",
//     "value"), where go-logfmt reports an empty value and a separate bare
//     key.
//   - A '"' inside an unquoted value is a literal byte, not a syntax error.
//   - Unknown escapes decode leniently (the escaped byte itself) instead of
//     being rejected.
//
// A bare key with no '=' is reported with the value "true", matching logfmt
// convention for boolean flags.
package logfmt
