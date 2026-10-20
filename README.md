# logfmt

[![CI](https://github.com/JohanLindvall/logfmt/actions/workflows/ci.yml/badge.svg)](https://github.com/JohanLindvall/logfmt/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/JohanLindvall/logfmt.svg)](https://pkg.go.dev/github.com/JohanLindvall/logfmt)
[![Version](https://img.shields.io/github/v/tag/JohanLindvall/logfmt?sort=semver&label=version)](https://pkg.go.dev/github.com/JohanLindvall/logfmt?tab=versions)
[![Go Report Card](https://goreportcard.com/badge/github.com/JohanLindvall/logfmt)](https://goreportcard.com/report/github.com/JohanLindvall/logfmt)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Read logfmt in Go. Extract the fields you need without building a map.**

A fast [logfmt](https://brandur.org/logfmt) parser for log processors, command-line
tools, and observability pipelines. Read directly from `[]byte`, look up one or
several keys, or visit every pair in order.

- **Zero-copy reads:** keys and raw values are slices of your input; parsing
  well-formed records allocates nothing.
- **Selective lookups:** stop as soon as the requested keys have non-empty values.
- **Decode when needed:** keep values raw or unescape quoted values into a buffer
  you can reuse.
- **Small dependency footprint:** standard library only, Go 1.21+, with tests on
  amd64 and arm64 and differential fuzzers for parsing, lookups, and decoding.

[Quick start](#quick-start) · [Choose an API](#choose-an-api) · [Usage guide](docs/usage.md) · [Benchmarks](#benchmarks) · [Contributing](CONTRIBUTING.md)

## Quick start

Install in your Go module:

```sh
go get github.com/JohanLindvall/logfmt
```

Save this as `main.go`, then run `go run .`:

```go
package main

import (
    "fmt"

    "github.com/JohanLindvall/logfmt"
)

func main() {
    line := []byte(`level=error msg="disk \"sda\" full" retries=3`)

    // One key, zero-copy: level is a sub-slice of line.
    level, _ := logfmt.Get(line, "level")
    fmt.Printf("level: %s\n", level)

    // Escapes decoded into a buffer you own.
    msg, _ := logfmt.AppendValue(nil, line, "msg")
    fmt.Printf("msg: %s\n", msg)

    // Every pair in order, values raw; return false to stop early.
    err := logfmt.Iterate(line, func(key, val []byte) bool {
        fmt.Printf("%s=%s\n", key, val)
        return true
    })
    if err != nil {
        fmt.Println(err)
    }
}
```

Output:

```text
level: error
msg: disk "sda" full
level=error
msg=disk \"sda\" full
retries=3
```

**[Run this example in your browser](https://pkg.go.dev/github.com/JohanLindvall/logfmt#example-package)**
— no installation needed. The quick start is also an executable Go example in
[example_test.go](example_test.go).

## Choose an API

All parsing functions accept `[]byte`; lookup keys are strings. A **raw** value
has its surrounding quotes removed and its backslash escapes left intact.

| I want to… | Use | Result |
|---|---|---|
| Read one raw value | [`Get`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#Get) | Value and whether the key exists |
| Read several raw values in one pass | [`GetMany`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#GetMany) | Values in query order; `nil` means absent |
| Copy a value and decode its escapes | [`AppendValue`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#AppendValue) | Appends to your buffer; decodes only quoted values |
| Decode only when necessary | [`GetQuoted`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#GetQuoted) | Raw value, quoted flag, and found flag |
| Visit pairs and receive syntax errors | [`Iterate`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#Iterate) | Callback per pair; return `false` to stop |
| Visit pairs with a range loop | [`All`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#All) | Go 1.23+ range iterator; no error result |
| Check a whole record | [`Validate`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#Validate) | First syntax error and its byte offset |
| Split a buffer into records | [`SplitRecord`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#SplitRecord) | One line and the remaining buffer; handles CRLF |
| Distinguish `debug` from `debug=true` | [`IsBareKey`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#IsBareKey) | Whether a value is the bare-key sentinel |
| Parse a timestamp | [`ParseTime`](https://pkg.go.dev/github.com/JohanLindvall/logfmt#ParseTime) | UTC time and success flag |

For decoding values you know were quoted, use `NeedsUnescape` and
`AppendUnescape`. See the [decoding recipe](docs/usage.md#unescape-a-raw-value).

## Everyday usage

Extract several fields and reuse the result buffer across records:

```go
keys := []string{"level", "status"}
buf := make([][]byte, len(keys))

for _, line := range lines {
    buf = logfmt.GetMany(line, keys, buf)
    if buf[0] != nil {
        fmt.Printf("level=%s\n", buf[0])
    }
    if buf[1] != nil {
        fmt.Printf("status=%s\n", buf[1])
    }
}
```

On Go 1.23+, visit every pair with a range loop:

```go
for key, val := range logfmt.All(line) {
    fmt.Printf("%s=%s\n", key, val)
}
```

The [usage guide](docs/usage.md) covers reading a file or stream, reusing decode
buffers, working with duplicate keys, timestamps, and error handling. Individual
functions also have [runnable examples](https://pkg.go.dev/github.com/JohanLindvall/logfmt#pkg-examples).

## Behavior to know

- **One record at a time.** Newlines are whitespace to the parser. Use
  `SplitRecord` or `bufio.Scanner` when reading multiple log lines.
- **Treat returned slices as read-only.** They alias the input, except for a
  bare key's shared `true` value. Copy values you need to keep before reusing or
  modifying the input buffer.
- **Escapes belong to quoted values.** `AppendValue` safely decodes
  `msg="first\nsecond"` and preserves an unquoted `path=C:\Users\bob` literally.
- **Duplicate keys:** lookups return the first non-empty value, or an empty
  value if no non-empty occurrence exists. A present empty value is distinct
  from an absent key.
- **Validation is explicit.** Lookups and `All` return no syntax errors.
  `Validate` checks the full record; `Iterate` returns an error after delivering
  the valid prefix.

This is a lenient reader. For an encoder, use
[go-logfmt](https://github.com/go-logfmt/logfmt) or your logging library.
See the [package documentation](https://pkg.go.dev/github.com/JohanLindvall/logfmt)
for parsing differences, ownership rules, and the full API.

## Benchmarks

In the committed September 2026 runs, parsing every pair in the shared 1.4 KB
sample was about **8× faster than go-logfmt**, and extracting two keys was about
**15–20× faster**, across amd64 and arm64. Those are workload-specific results;
input shape, decoding, buffer reuse, hardware, and Go version all matter.

| Results | amd64 | arm64 |
|---|---|---|
| Comparison with other parsers | [Results](bench/results_amd64.md) | [Results](bench/results_arm64.md) |
| Package microbenchmarks | [Results](bench/pkg_results_amd64.md) | [Results](bench/pkg_results_arm64.md) |

Each table records the environment and allocations. `ParseAll` returns raw
values here, while the other parsers decode eagerly; use `ParseEscaped` to
compare parsing **and** decoding. Extraction benchmarks measure early stopping.
`GetMany` needs capacity in its result buffer to avoid allocating; append
functions may grow their destination. A malformed record can allocate a
`SyntaxError`.

```sh
go test -run='^$' -bench=. -benchmem .
```

See [benchmark methodology and reproduction](bench/README.md) for comparisons,
architecture details, and measuring a proposed optimization.

## Help and contributing

[Ask a question](https://github.com/JohanLindvall/logfmt/issues/new?template=question.yml),
[report a bug](https://github.com/JohanLindvall/logfmt/issues/new?template=bug_report.yml),
or [suggest an improvement](https://github.com/JohanLindvall/logfmt/issues/new?template=feature_request.yml).
Use a small, sanitized log sample when reporting a problem. For suspected
vulnerabilities, use the private route in [SECURITY.md](SECURITY.md).

Documentation fixes, examples, and reproducible bug reports are welcome first
contributions. Start with [CONTRIBUTING.md](CONTRIBUTING.md) for setup, tests, and
performance checks, or browse issues marked
[good first issue](https://github.com/JohanLindvall/logfmt/labels/good%20first%20issue).
We follow a [code of conduct](CODE_OF_CONDUCT.md).

Maintained by [Johan Lindvall](https://github.com/JohanLindvall) with help from
[contributors](https://github.com/JohanLindvall/logfmt/graphs/contributors).
If logfmt is useful to you, a star helps other Go developers discover it;
sharing a real use case or a reproducible benchmark helps too.

## License

[MIT](LICENSE). Source files include SPDX license identifiers.
