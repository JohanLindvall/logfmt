# logfmt

[![CI](https://github.com/JohanLindvall/logfmt/actions/workflows/ci.yml/badge.svg)](https://github.com/JohanLindvall/logfmt/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/JohanLindvall/logfmt.svg)](https://pkg.go.dev/github.com/JohanLindvall/logfmt)
[![Go Report Card](https://goreportcard.com/badge/github.com/JohanLindvall/logfmt)](https://goreportcard.com/report/github.com/JohanLindvall/logfmt)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A fast, allocation-free **reader** for the [logfmt](https://brandur.org/logfmt)
line format in Go:

```
level=info msg="user login" user=john id=42 success=true
```

The package operates on `[]byte` and reports keys and values as sub-slices of
the input, so iterating a line performs **zero allocations**. It has no
dependencies outside the standard library, and parses a ~1.4 KB line at
~3 GB/s — roughly 6× go-logfmt, with key extraction 12× faster still.

## Install

```sh
go get github.com/JohanLindvall/logfmt
```

Requires Go 1.21 or newer. (CI tests that floor on every push on amd64, and
the current stable release on both amd64 and arm64.) Ranging over `All` needs
Go 1.23 in *your* module — the library itself stays at 1.21, so it never forces
a toolchain upgrade on you.

## Usage

### Iterate over every key/value pair

`Iterate` calls your callback once per pair. The `key` and `val` slices alias
the input buffer, so copy them if you need to keep them past the call. Return
`false` from the callback to stop early.

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
  the callback hands you (see [Escapes belong to quoted values
  only](#escapes-belong-to-quoted-values-only)).
- Returned slices alias the input (or, for bare keys, a shared constant) —
  treat them as read-only and copy anything that must outlive the input.
- The parser is deliberately lenient and diverges from go-logfmt in a few
  documented ways (e.g. a stray `"` in an unquoted value is a literal byte, not
  an error). See the [package documentation](https://pkg.go.dev/github.com/JohanLindvall/logfmt).
- All lookups (`Get`, `GetQuoted`, `AppendValue`, `GetMany`) resolve duplicate keys the same
  way: the **first non-empty occurrence wins**; an empty value is used only
  when the key never appears with a non-empty one.

### One record per call

Newlines are ordinary whitespace to this parser — there is no record framing.
Hand it a multi-line buffer and you get every line's pairs as one flat stream,
with no boundary marker, and lookups can match a key from a later line.
`SplitRecord` peels off one record at a time without allocating, and handles
CRLF:

```go
for len(data) > 0 {
    var rec []byte
    rec, data = logfmt.SplitRecord(data)
    level, _ := logfmt.Get(rec, "level")
    fmt.Printf("%s\n", level)
}
```

### Look up a single key, unescaped

`AppendValue` finds a key and appends its **unescaped** value to your buffer,
returning the extended slice and whether the key was present. It always appends,
so the result never aliases the input and is yours to keep.

```go
var buf []byte
val, ok := logfmt.AppendValue(buf[:0], line, "msg")
if !ok {
    // key absent
}
fmt.Printf("msg = %s\n", val)
```

### Look up a single key, raw

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

### Look up several keys in one pass

`GetMany` extracts multiple keys in a single scan, stopping early once all are
found. Each returned value is **raw** and aliases the input; a missing key is
reported as `nil`. (A present but empty value, such as from `key=`, is a non-nil
zero-length slice, so it stays distinct from an absent key.) Pass a `[][]byte` to
reuse as the result slice across calls and avoid allocating it each time.

```go
keys := []string{"timestamp", "level"}
var buf [][]byte // reuse across calls

vals := logfmt.GetMany(line, keys, buf)
for i, v := range vals {
    if v == nil {
        continue // keys[i] not present
    }
    fmt.Printf("%s = %s\n", keys[i], v)
}
```

Keys are matched linearly against each parsed field, which is fastest for the
handful of keys these lookups target. Measured on a 24-field line, `GetMany`
stays ahead up to roughly ten keys; past that, use `Iterate` with a map keyed by
`string(k)` (20 keys: ~505 ns vs ~385 ns) — the compiler optimizes that
conversion away in a map index.

### Unescape a raw value

`AppendUnescape` decodes the escapes in a raw value (as returned by `Iterate`,
`All`, `Get` or `GetMany`), appending to a destination buffer. It recognises `\n`, `\r`,
`\t` and JSON-style `\uXXXX` unicode escapes including surrogate pairs — so
values encoded by go-logfmt (which writes control characters as `\u00XX`)
round-trip correctly. Any other escaped byte (such as `\"` or `\\`) is emitted
as-is; malformed `\u` sequences and a trailing lone backslash are kept verbatim.

It always appends, so the result never aliases the input.

```go
dst := logfmt.AppendUnescape(nil, []byte(`hello\tworld`)) // -> hello<TAB>world
```

Most values contain no escapes at all, so guard with `NeedsUnescape` when you
want to skip the copy entirely. Use `GetQuoted` rather than `Get` here — it
reports whether the value was quoted, which is what makes the decode safe (see
below):

```go
if v, quoted, ok := logfmt.GetQuoted(line, "msg"); ok {
    if quoted && logfmt.NeedsUnescape(v) {
        v = logfmt.AppendUnescape(buf[:0], v) // decoded into buf
    }
    // otherwise v still aliases line, with no copy made
}
```

### Escapes belong to quoted values only

A backslash means something only inside quotes. `msg="a\nb"` holds an escape;
`path=C:\Users\bob` holds three literal backslashes — and go-logfmt's encoder
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

### Parse a timestamp value

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

- **`Get` and `GetMany` cap their results** (`cap == len`), so `append(v, …)`
  copies instead of overwriting whatever follows the value in the input. They
  set a slot once per lookup, so the capping is free. `AppendValue` and
  `AppendUnescape` go further and never alias the input at all.
- **`Iterate` does not cap** what it passes the callback. Doing so costs ~4.5%
  on field-dense input (measured) because it lands once per *field*. Inside a
  callback, copy before appending — `append(dst[:0], v...)`, `string(v)` — or
  re-slice with `v[:len(v):len(v)]` yourself.

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

Absence is uniform: `Get` and `AppendValue` return `false`, `GetMany` leaves the
slot `nil`.

## Scope

A reader, deliberately and only. Not included: an encoder, an `io.Reader`
streaming decoder, typed accessors (int/bool/duration), and map building.
Values come back as `[]byte` for you to convert with `strconv`. That is what
keeps the package dependency-free, allocation-free, and small enough to fuzz
the whole parser against a byte-by-byte reference implementation on every
change. To *write* logfmt, use go-logfmt or your logging library's encoder.

## Benchmarks

```sh
go test -bench=. -benchmem      # this package's microbenchmarks
make bench-md                   # regenerate the committed tables in bench/
```

`Iterate`, `All`, `Get`, `GetQuoted`, `GetMany` and `SplitRecord` allocate
nothing on a well-formed record (and `AppendValue`/`AppendUnescape` nothing
beyond growing your buffer). The one exception is a malformed record, which
costs a single 24-byte `*SyntaxError` — and the lookups pay it only when they
have to walk past the fault to settle their keys.

Cost splits into a fixed per-field overhead of ~5 ns plus scanning: ~11.7 GB/s
through unquoted values (word-at-a-time SWAR) and ~27 GB/s through quoted ones
(`bytes.IndexByte`, SIMD in the stdlib). A short unquoted value that ends
within a few bytes of its `=` is cheaper still: the key scan has already seen
those bytes and settles the value without a second scan. Short fields are
therefore overhead-bound, long values scan-bound. Lookups are linear in how
deep the key sits: ~7 ns per field skipped.

That 27 GB/s is for quoted values with **no escaped quotes in them**. Escaped
quotes cost extra, but boundedly: the first `\"` in a value restarts
`bytes.IndexByte`, and from then on — provided the escapes are close enough
together to be worth it — the parser walks a word at a time looking for the next
`"` or `\`, consuming each escape as it goes, and falls back to `bytes.IndexByte`
as soon as they thin out again. So a value dense with escapes — embedded JSON,
where every quote is one — costs a few nanoseconds per escape rather than a
fresh `IndexByte` call each, while a value with two escapes 200 bytes apart
never leaves the fast path it was already on. A 1 KB value with 500 escapes
parses roughly 59× slower than a clean 1 KB; `Benchmark_IterateEscaped` sweeps
that axis, and `Benchmark_UnescapeEscaped` sweeps it for `AppendUnescape`, which
uses the same trick while decoding and is the slower half at high density.

On amd64, building with `GOAMD64=v3` (Haswell+, 2013 onwards) makes the parser
1–2% faster (BMI's `TZCNT` for the word-at-a-time scanning). It is a consumer
build flag, not something the module can set.

### vs other Go logfmt parsers

The `bench/` module is a separate module, so the root package stays
dependency-free; it compares against go-logfmt, kr/logfmt and Grafana Loki's
in-tree decoder. (The Loki entry is a stand-in adapted from go-logfmt under
MIT rather than a vendored copy — Loki's own tree is AGPL-licensed — verified
equivalent to Loki's decoder on these inputs; see `bench/lokifmt`.)

**The numbers live in the generated tables, not here.** They are produced by
`make bench-md` (and by the `bench` CI workflow, which commits them), each
stamped with the CPU and Go version that produced it — no figure is copied into
this file, because a copied one goes stale silently and this one did:

- [bench/results_amd64.md](bench/results_amd64.md) — cross-library comparison
- [bench/results_arm64.md](bench/results_arm64.md) — same, on arm64
- [bench/pkg_results_amd64.md](bench/pkg_results_amd64.md) — this package's own
  microbenchmarks

For orientation only, on the ~1.4 KB sample line this package parses every pair
roughly **7× faster than go-logfmt** with zero allocations, and extracts two
keys roughly **14× faster** by stopping as soon as both are found. Consult the
tables for the actual figures on actual hardware; treat any ratio quoted in
prose as approximate and possibly a release behind.

## License

MIT — see [LICENSE](LICENSE).
