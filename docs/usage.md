# Usage guide

[Quick start](../README.md#quick-start) · [API reference](https://pkg.go.dev/github.com/JohanLindvall/logfmt) · [Runnable examples](../example_test.go)

The snippets below go inside a Go function and use the `logfmt` package from
the quick start. Add standard-library imports such as `fmt`, `bufio`, `strings`,
`log`, and `errors` when a snippet uses them.

- [Visit every pair](#iterate-over-every-keyvalue-pair)
- [Split records](#one-record-per-call) or [read a stream](#read-a-file-or-stream)
- [Copy and decode a value](#look-up-a-single-key-unescaped)
- [Read raw values](#look-up-a-single-key-raw) or [several keys](#look-up-several-keys-in-one-pass)
- [Decode only quoted values](#unescape-a-raw-value)
- [Handle errors](#errors) and [retain values safely](#read-only-really-means-read-only)

## Iterate over every key/value pair

`Iterate` calls your callback once per pair. The `key` and `val` slices alias
the input buffer and remain valid until it is modified or reused. Copy them
if you need an independent value. Return `false` from the callback to stop early.

```go
line := []byte(`level=info msg="user login" user=john id=42`)

err := logfmt.Iterate(line, func(key, val []byte) bool {
    fmt.Printf("%s = %s\n", key, val)
    return true // return false to stop early
})
if err != nil {
    log.Fatal(err)
}
```

On Go 1.23 or newer, `All` is the same walk as a range loop:

```go
for key, val := range logfmt.All(line) {
    fmt.Printf("%s = %s\n", key, val)
}
```

Notes:

- A bare key with no `=` (e.g. `debug`) is reported with `val` equal to the
  literal `true`. `IsBareKey(val)` tells that apart from an explicit
  `debug=true`, which is otherwise byte-identical.
- Quoted values are returned **without** the surrounding quotes but are **not**
  unescaped — backslash escapes are left intact. `val` doesn't say whether it
  was quoted, and escapes only mean anything inside quotes, so decode with
  `AppendValue` or `GetQuoted` rather than calling `AppendUnescape` on whatever
  the callback hands you (see [Escapes belong to quoted values only](#escapes-belong-to-quoted-values-only)).
- Returned slices alias the input (or, for bare keys, a shared constant) —
  treat them as read-only and copy anything that must outlive the input.
- The parser is deliberately lenient and diverges from go-logfmt in a few
  documented ways (e.g. a stray `"` in an unquoted value is a literal byte, not
  an error). See the [package documentation](https://pkg.go.dev/github.com/JohanLindvall/logfmt).
- All lookups (`Get`, `GetQuoted`, `AppendValue`, `GetMany`) resolve duplicate keys the same
  way: the **first non-empty occurrence wins**; an empty value is used only
  when the key never appears with a non-empty one.

## One record per call

Newlines are ordinary whitespace to this parser — there is no record framing.
Hand it a multi-line buffer and you get every line's pairs as one flat stream,
with no boundary marker, and lookups can match a key from a later line.
`SplitRecord` peels off one record at a time without allocating, and handles
CRLF:

```go
data := []byte("level=info msg=started\r\nlevel=error msg=failed\n")

for len(data) > 0 {
    var rec []byte
    rec, data = logfmt.SplitRecord(data)
    level, _ := logfmt.Get(rec, "level")
    fmt.Printf("%s\n", level)
}
```

## Read a file or stream

Use `bufio.Scanner` to supply one line at a time from an `io.Reader`, such as
an open file or `os.Stdin`. Set a maximum record size appropriate for your
application and check the scanner's error:

```go
input := strings.NewReader("level=info msg=started\nlevel=error msg=failed\n")
scanner := bufio.NewScanner(input)
// Choose a maximum record size for your application; Scanner's default
// is 64 KiB. Allow up to 1 MiB here, including the line ending.
scanner.Buffer(make([]byte, 64*1024), 1024*1024)

for scanner.Scan() {
    level, _ := logfmt.Get(scanner.Bytes(), "level")
    // Use the value before Scan reuses its buffer; copy it to retain it.
    fmt.Printf("%s\n", level)
}
if err := scanner.Err(); err != nil {
    fmt.Println("read logs:", err)
}
```

`Scanner.Bytes()` and raw lookup results share storage. Use `string(value)` or
`bytes.Clone(value)` to retain a value across calls to `Scan`.
See [Scanner.Buffer](https://pkg.go.dev/bufio#Scanner.Buffer) for the size limit.
This recipe is also a [runnable example](https://pkg.go.dev/github.com/JohanLindvall/logfmt#example-package-ReadStream).

## Look up a single key, unescaped

`AppendValue` finds a key and appends its **unescaped** value to your buffer,
returning the extended slice and whether the key was present. With a separate
destination buffer, the result is independent of the input.

```go
lines := [][]byte{
    []byte(`msg="first\nsecond"`),
    []byte(`msg=done`),
}
buf := make([]byte, 0, 64)

for _, line := range lines {
    var ok bool
    // Reset the length, then keep the returned slice for the next call.
    buf, ok = logfmt.AppendValue(buf[:0], line, "msg")
    if ok {
        fmt.Printf("%q\n", buf)
    }
}
```

Keeping the returned slice matters: an append may allocate a larger buffer.
Reusing `buf[:0]` overwrites the previous decoded value, so copy it if you need
to retain values from several records. An absent key returns the destination
unchanged and `false`.

## Look up a single key, raw

`Get` returns the **raw** value (surrounding quotes removed, escape sequences
left intact) and whether the key was found. The result aliases the input — no
copy, no allocation — and is valid until the input is modified. Use
`AppendValue` instead when you want the value unescaped into your own buffer.

```go
val, ok := logfmt.Get(line, "msg")
if !ok {
    // key absent
}
fmt.Printf("msg = %s\n", val) // raw value, aliasing line
```

A key present with an empty value (`msg=`) returns `ok == true` and a non-nil
empty slice, so it stays distinct from an absent key.

## Look up several keys in one pass

`GetMany` extracts multiple keys in a single scan, stopping early once all are
settled with non-empty values. Each returned value is **raw** and aliases the input; a missing key is
reported as `nil`. (A present but empty value, such as from `key=`, is a non-nil
zero-length slice, so it stays distinct from an absent key.) Pass a `[][]byte` to
reuse as the result slice across calls and avoid allocating it each time.

```go
lines := [][]byte{
    []byte(`level=info status=200`),
    []byte(`level=error msg="request failed"`),
}
keys := []string{"level", "status"}
buf := make([][]byte, len(keys))

for _, line := range lines {
    // Keep the returned slice so the next call can reuse its storage.
    buf = logfmt.GetMany(line, keys, buf)
    for i, v := range buf {
        if v == nil {
            fmt.Printf("%s: absent\n", keys[i])
            continue
        }
        fmt.Printf("%s: %s\n", keys[i], v)
    }
}
```

Reuse replaces the slots in `buf`, including resetting absent values to `nil`.
The outer slice needs capacity for `len(keys)` entries to avoid allocating.
Each value still shares storage with its input record. An empty key list
returns without parsing.

Lookups resolve `msg= msg=ready msg=later` as `ready`: an empty value remains
provisional until a non-empty occurrence appears. `Iterate` and `All` instead
visit every occurrence in input order.

## Unescape a raw value

`AppendUnescape` decodes the escapes in a raw value you know was quoted,
appending to a destination buffer. It recognises `\n`, `\r`,
`\t` and JSON-style `\uXXXX` unicode escapes including surrogate pairs — so
values encoded by go-logfmt (which writes control characters as `\u00XX`)
round-trip correctly. Any other escaped byte (such as `\"` or `\\`) is emitted
as-is; malformed `\u` sequences and a trailing lone backslash are kept verbatim.

It always appends to the destination. Use a separate destination when you need
an independent copy.

```go
dst := logfmt.AppendUnescape(nil, []byte(`hello\tworld`)) // -> hello<TAB>world
```

Most values contain no escapes at all, so guard with `NeedsUnescape` when you
want to skip the copy entirely. Use `GetQuoted` rather than `Get` here — it
reports whether the value was quoted, which is what makes the decode safe (see
below):

```go
var buf []byte
if v, quoted, ok := logfmt.GetQuoted(line, "msg"); ok {
    if quoted && logfmt.NeedsUnescape(v) {
        buf = logfmt.AppendUnescape(buf[:0], v)
        v = buf
    }
    // otherwise v still aliases line, with no copy made
}
```

## Escapes belong to quoted values only

A backslash means something only inside quotes. `msg="a\nb"` holds an escape;
`path=C:\Users\bob\new` holds three literal backslashes — and go-logfmt's encoder
writes it exactly that way, because a backslash is not one of the bytes that
force quoting. Decoding without knowing which of the two you have is silent
corruption, since both are perfectly valid logfmt:

```go
line := []byte(`path=C:\Users\bob\new`)

logfmt.Get(line, "path")            // C:\Users\bob\new   — raw, correct
logfmt.AppendValue(nil, line, "path") // C:\Users\bob\new — knows it was unquoted
// AppendUnescape(nil, raw) would give "C:Usersbob<NL>ew": \U→U, \b→b, \n→newline
```

`Iterate`, `All`, `Get` and `GetMany` hand out quoted and unquoted values alike
without distinguishing them. Two entry points carry the missing bit:

- **`AppendValue`** decodes for you, and only when the value was quoted.
- **`GetQuoted`** returns the bit, for the zero-copy path above.

Reach for `AppendUnescape` directly only on a value you already know was quoted.

## Parse a timestamp value

`ParseTime` parses a logfmt timestamp value and reports whether it succeeded. It
accepts an RFC3339Nano string, a `2006-01-02 15:04:05.999 -0700 MST` string, or a
unix epoch (exactly 10 integer digits with an optional fractional part). Trailing
delimiters left over from a slightly malformed line (e.g. a stray `}`) are trimmed
first, and on success the returned time is normalized to UTC.

```go
t, ok := logfmt.ParseTime([]byte("1748239806.3691056"))
if ok {
    fmt.Println(t) // 2025-05-26 06:10:06.3691056 +0000 UTC
}
```

Millisecond/microsecond epochs (13 or 16 digits) and date-only strings are
rejected rather than guessed at — see the package docs for the full accepted set.
If you know your emitter's layout, `time.Parse` with that layout is both faster
and stricter.

## Read-only really means read-only

Returned slices are windows onto the input, so treat every one of them as
read-only: writing through a value overwrites your log line, and a bare key's
`true` is a package-level constant shared by every caller in the process.

Appending is the subtler case, and the two APIs differ deliberately:

- **`Get`, `GetQuoted`, and `GetMany` cap their results** (`cap == len`), so `append(v, …)`
  copies instead of overwriting whatever follows the value in the input. They
  remain read-only: capping does not prevent a direct write such as `v[0] = 'x'`.
- **`Iterate` and `All` do not cap values to their own length.** Appending can
  overwrite later fields in the same record. Inside a callback, copy before
  appending — `append(dst[:0], v...)` with a separate `dst` — or re-slice with
  `v[:len(v):len(v)]` yourself.
- **`AppendValue` and `AppendUnescape` write into your destination.** With a
  separate buffer, the result is independent of the input. `AppendUnescape`
  also supports decoding in place with `AppendUnescape(raw[:0], raw)` when you
  own the raw buffer; that deliberately overwrites the original bytes.

## Errors

Only `Iterate` and `Validate` report syntax errors, and both return a
`*SyntaxError` carrying the byte offset of the fault:

```go
if err := logfmt.Validate(line); err != nil {
    var se *logfmt.SyntaxError
    if errors.As(err, &se) {
        fmt.Printf("bad record at byte %d: %s\n", se.Offset, se.Reason)
    }
}
```

`errors.Is(err, logfmt.ErrBadFormat)` matches any of them, so sentinel checks
work too. There are exactly two faults: an unterminated quoted value, and a
closing quote followed by a non-space byte.

Two consequences of streaming:

- When `Iterate` returns an error, every pair *before* the fault has already
  been delivered to your callback. That prefix is valid.
- **The lookups report no errors at all.** They stop as soon as their keys are
  settled, so a malformed tail beyond that point is never examined — an error
  return would promise a validation they do not perform. They give you what the
  reachable prefix holds; call `Validate` when a record's validity matters.

Absence is uniform: `Get`, `GetQuoted`, and `AppendValue` return a false found
flag; `GetMany` leaves the slot `nil`.

## Scope

A reader, deliberately and only. Not included: an encoder, an `io.Reader`
streaming decoder, typed accessors (int/bool/duration), and map building.
Values come back as `[]byte` for you to convert with `strconv`. That is what
keeps the package dependency-free, allocation-free, and small enough to fuzz
the whole parser against a byte-by-byte reference implementation on every
change. To *write* logfmt, use go-logfmt or your logging library's encoder.
