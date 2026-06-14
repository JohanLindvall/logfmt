# logfmt

A fast, allocation-free reader for the [logfmt](https://brandur.org/logfmt) line
format in Go:

```
level=info msg="user login" user=john id=42 success=true
```

The package operates on `[]byte` and reports keys and values as sub-slices of
the input, so iterating a line performs **zero allocations**. It has no
dependencies outside the standard library.

## Install

```sh
go get github.com/JohanLindvall/logfmt
```

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
  unescaped — backslash escapes are left intact. Use `UnescapeInto` to decode
  them.

### Look up a single key, unescaped

`GetValue` finds a key and returns its **unescaped** value. When the value needs
decoding it is written into the caller-provided buffer (reusable across calls);
when it needs none, a sub-slice of the input is returned without copying. The
result thus aliases either the buffer or the input, so copy it if it must outlive
them.

```go
var buf []byte
val, err := logfmt.GetValue(line, []byte("msg"), buf[:0])
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

### Unescape a raw value

`UnescapeInto` decodes the escapes in a raw value (as returned by `Iterate`,
`Get` or `GetMany`), appending to a destination buffer. It recognises `\n`, `\r`
and `\t`; any other escaped byte (such as `\"` or `\\`) is emitted as-is.

```go
dst := logfmt.UnescapeInto(nil, []byte(`hello\tworld`)) // "hello\tworld" -> hello<TAB>world
```

`NeedsUnescape` reports whether a raw value actually contains a backslash escape,
so you can skip the decode (and its copy) when it is unnecessary:

```go
v, _ := logfmt.Get(line, "msg")
if logfmt.NeedsUnescape(v) {
    v = logfmt.UnescapeInto(buf[:0], v)
}
```

### Parse a timestamp value

`ParseTime` parses a logfmt timestamp value and reports whether it succeeded. It
accepts an RFC3339Nano string, a `2006-01-02 15:04:05.999 -0700 MST` string, or a
unix epoch (10 integer digits with an optional fractional part). Trailing
delimiters left over from a slightly malformed line (e.g. a stray `}`) are trimmed
first, and on success the returned time is normalized to UTC.

```go
t, ok := logfmt.ParseTime("1748239806.3691056")
if ok {
    fmt.Println(t) // 2025-05-26 06:10:06.3691056 +0000 UTC
}
```

## Errors

| Error             | Meaning                                                        |
| ----------------- | -------------------------------------------------------------- |
| `ErrBadFormat`    | Malformed input, e.g. an unterminated quoted value.            |
| `ErrKeyNotFound`  | `GetValue` or `Get` could not find the requested key.          |

## Benchmarks

```sh
go test -bench=. -benchmem
```

`Iterate`, `Get` and `GetMany` allocate nothing on the hot path (and `GetValue`
when its buffer is reused); the included benchmarks run against representative
single- and multi-row logfmt input.
