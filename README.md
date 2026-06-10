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

### Look up a single key

`GetValue` finds a key and returns its **unescaped** value, writing into a
caller-provided buffer you can reuse across calls.

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

### Unescape a raw value

`UnescapeInto` decodes the escapes in a raw value (as returned by `Iterate`),
appending to a destination buffer. It recognises `\n`, `\r` and `\t`; any other
escaped byte (such as `\"` or `\\`) is emitted as-is.

```go
dst := logfmt.UnescapeInto(nil, []byte(`hello\tworld`)) // "hello\tworld" -> hello<TAB>world
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
| `ErrKeyNotFound`  | `GetValue` could not find the requested key.                   |

## Benchmarks

```sh
go test -bench=. -benchmem
```

`Iterate` and `GetValue` allocate nothing on the hot path; the included
benchmarks run against representative single- and multi-row logfmt input.
