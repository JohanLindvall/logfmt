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

Requires Go 1.21 or newer. (CI tests that floor on every push, alongside the
current stable release, on both amd64 and arm64.)

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

Notes:

- A bare key with no `=` (e.g. `debug`) is reported with `val` equal to the
  literal `true`.
- Quoted values are returned **without** the surrounding quotes but are **not**
  unescaped — backslash escapes are left intact. Use `Unescape` to decode
  them.
- Returned slices alias the input (or, for bare keys, a shared constant) —
  treat them as read-only and copy anything that must outlive the input.
- The parser is deliberately lenient and diverges from go-logfmt in a few
  documented ways (e.g. a stray `"` in an unquoted value is a literal byte, not
  an error). See the [package documentation](https://pkg.go.dev/github.com/JohanLindvall/logfmt).
- All lookups (`Get`, `GetValue`, `GetMany`) resolve duplicate keys the same
  way: the **first non-empty occurrence wins**; an empty value is used only
  when the key never appears with a non-empty one.

### One record per call

Newlines are ordinary whitespace to this parser — there is no record framing.
Hand it a multi-line buffer and you get every line's pairs as one flat stream,
with no boundary marker, and lookups can match a key from a later line. Split
the input yourself and call `Iterate` once per record:

```go
for len(data) > 0 {
    line := data
    if i := bytes.IndexByte(data, '\n'); i >= 0 {
        line, data = data[:i], data[i+1:]
    } else {
        data = nil
    }
    _ = logfmt.Iterate(line, func(k, v []byte) bool { /* ... */ return true })
}
```

### Look up a single key, unescaped

`GetValue` finds a key and returns its **unescaped** value. When the value needs
decoding it is written into the caller-provided buffer (reusable across calls);
when it needs none, a sub-slice of the input is returned without copying. The
result thus aliases either the buffer or the input, so copy it if it must outlive
them.

```go
var buf []byte
val, err := logfmt.GetValue(line, "msg", buf[:0])
switch {
case errors.Is(err, logfmt.ErrKeyNotFound):
    // key absent
case err != nil:
    log.Fatal(err)
default:
    fmt.Printf("msg = %s\n", val)
}
```

### Look up a single key, raw

`Get` returns the **raw** value (surrounding quotes removed, escape sequences
left intact). The result aliases the input — no copy, no allocation — and is
valid until the input is modified. Use `GetValue` instead when you want the
value unescaped into your own buffer.

```go
val, err := logfmt.Get(line, "msg")
switch {
case errors.Is(err, logfmt.ErrKeyNotFound):
    // key absent
case err != nil:
    log.Fatal(err)
default:
    fmt.Printf("msg = %s\n", val) // raw value, aliasing line
}
```

### Look up several keys in one pass

`GetMany` extracts multiple keys in a single scan, stopping early once all are
found. Each returned value is **raw** and aliases the input; a missing key is
reported as `nil`. (A present but empty value, such as from `key=`, is a non-nil
zero-length slice, so it stays distinct from an absent key.) Pass a `[][]byte` to
reuse as the result slice across calls and avoid allocating it each time.

```go
keys := []string{"timestamp", "level"}
var buf [][]byte // reuse across calls

vals, err := logfmt.GetMany(line, keys, buf)
if err != nil {
    log.Fatal(err)
}
for i, v := range vals {
    if v == nil {
        continue // keys[i] not present
    }
    fmt.Printf("%s = %s\n", keys[i], v)
}
```

Keys are matched linearly against each parsed field, so this is meant for small
key sets (a handful). For many keys, use `Iterate` with a map keyed by
`string(k)` — the compiler optimizes that conversion away in a map index.

### Unescape a raw value

`Unescape` decodes the escapes in a raw value (as returned by `Iterate`,
`Get` or `GetMany`), appending to a destination buffer. It recognises `\n`, `\r`,
`\t` and JSON-style `\uXXXX` unicode escapes including surrogate pairs — so
values encoded by go-logfmt (which writes control characters as `\u00XX`)
round-trip correctly. Any other escaped byte (such as `\"` or `\\`) is emitted
as-is; malformed `\u` sequences and a trailing lone backslash are kept verbatim.

As a fast path, when the value contains no escape at all the buffer is left
untouched and the value is returned directly — so the result may alias either the
destination buffer or the input. Use the returned slice, not the buffer you
passed in.

```go
dst := logfmt.Unescape(nil, []byte(`hello\tworld`)) // "hello\tworld" -> hello<TAB>world
```

`NeedsUnescape` reports whether a raw value actually contains a backslash escape.
`Unescape` already skips the copy on its own when there is nothing to decode, but
`NeedsUnescape` lets you branch before deciding whether to involve a buffer:

```go
v, _ := logfmt.Get(line, "msg")
if logfmt.NeedsUnescape(v) {
    v = logfmt.Unescape(buf[:0], v)
}
```

### Parse a timestamp value

`ParseTime` parses a logfmt timestamp value and reports whether it succeeded. It
accepts an RFC3339Nano string, a `2006-01-02 15:04:05.999 -0700 MST` string, or a
unix epoch (exactly 10 integer digits with an optional fractional part). Trailing
delimiters left over from a slightly malformed line (e.g. a stray `}`) are trimmed
first, and on success the returned time is normalized to UTC.

```go
t, ok := logfmt.ParseTime("1748239806.3691056")
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

- **`Get`, `GetValue` and `GetMany` cap their results** (`cap == len`), so
  `append(v, …)` copies instead of overwriting whatever follows the value in the
  input. They set a slot once per lookup, so the capping is free.
- **`Iterate` does not cap** what it passes the callback. Doing so costs ~4.5%
  on field-dense input (measured) because it lands once per *field*. Inside a
  callback, copy before appending — `append(dst[:0], v...)`, `string(v)` — or
  re-slice with `v[:len(v):len(v)]` yourself.

## Errors

| Error            | Meaning                                                        |
| ---------------- | -------------------------------------------------------------- |
| `ErrBadFormat`   | Unterminated quoted value, or a closing quote followed by a non-space byte. |
| `ErrKeyNotFound` | `Get` or `GetValue` could not find the requested key (`GetMany` reports absence as a `nil` slot). |

Two streaming consequences:

- When `Iterate` returns `ErrBadFormat`, every pair *before* the fault has
  already been delivered to your callback. That prefix is valid.
- The lookups stop as soon as their keys are settled, so a malformed tail beyond
  that point is never seen. A `nil` error from `Get`/`GetMany`/`GetValue` means
  "your keys resolved", not "the line is well-formed". Run `Iterate` to
  completion if you need validation.

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

`Iterate`, `Get` and `GetMany` allocate nothing on the hot path (and `GetValue`
when its buffer is reused). Cost splits into a fixed per-field overhead of
~6 ns plus scanning: ~11.7 GB/s through unquoted values (word-at-a-time SWAR)
and ~27 GB/s through quoted ones (`bytes.IndexByte`, SIMD in the stdlib). Short
fields are therefore overhead-bound, long values scan-bound. Lookups are linear
in how deep the key sits: ~8.6 ns per field skipped.

On amd64, building with `GOAMD64=v3` (Haswell+, 2013 onwards) makes the parser
~3% faster (BMI's `TZCNT` for the word-at-a-time scanning). It is a consumer
build flag, not something the module can set.

### vs other Go logfmt parsers

Parsing the same ~1.4 KB line, measured on GitHub Actions `ubuntu-latest`
(AMD EPYC 7763, Go 1.26); lower is better, speedup relative to `go-logfmt`. The
`bench/` module is a separate module, so the root package stays
dependency-free; it compares against go-logfmt, kr/logfmt and Grafana Loki's
in-tree decoder. Full tables, including arm64 and shorter/escaped inputs, are in
[bench/results_amd64.md](bench/results_amd64.md) and
[bench/results_arm64.md](bench/results_arm64.md).

Parse every key/value pair:

| Parser | ns/op | Throughput | allocs/op | Speedup |
|---|--:|--:|--:|--:|
| **this package** | **444** | **3157 MB/s** | **0** | **6.3×** |
| kr/logfmt | 1473 | 951 MB/s | 1 | 1.9× |
| Grafana Loki | 1967 | 712 MB/s | 1 | 1.4× |
| go-logfmt | 2779 | 504 MB/s | 4 | 1.0× |

Extract two keys (`timestamp`+`level`), each parser stopping once both are found
(where its API allows — `kr/logfmt` is push-based and can't stop its scan):

| Parser | ns/op | allocs/op | Speedup |
|---|--:|--:|--:|
| **this package** (`GetMany`) | **96** | **0** | **11.8×** |
| Grafana Loki | 349 | 1 | 3.3× |
| go-logfmt | 1135 | 3 | 1.0× |
| kr/logfmt | 1560 | 4 | 0.7× |

Faster hardware roughly halves these: the same two benchmarks measure 266 ns and
54 ns on a Ryzen 7 8840HS.

## License

MIT — see [LICENSE](LICENSE).
