# Contributing to logfmt

Documentation improvements, examples, bug reports, and code changes are welcome.
You do not need to understand the parser's performance tuning to contribute.
Please follow our [code of conduct](CODE_OF_CONDUCT.md).

For usage help, see [support](.github/SUPPORT.md). Report suspected
vulnerabilities through [SECURITY.md](SECURITY.md).

## Start here

- Fix an unclear explanation or add an example for a real use case.
- Reduce a parsing or decoding problem to a small input and expected output.
- Browse [good first issues](https://github.com/JohanLindvall/logfmt/labels/good%20first%20issue)
  or [help wanted](https://github.com/JohanLindvall/logfmt/labels/help%20wanted).
- For a new API or a substantial optimization, open an issue describing the
  problem and approach before investing in a large change. Small fixes can go
  straight to a pull request.

The package is intentionally a small reader: no encoder, map-building API, or
typed accessors beyond timestamps. Preserve the read-only, raw-by-default API,
duplicate-key behavior, and allocation guarantees described in
[the usage guide](docs/usage.md).

## Set up a checkout

Use the current stable Go release for development. The library supports
**Go 1.21+**; the separate `bench/` module requires **Go 1.23+** to test consumers
that range over `All`. Keep those minimum versions unchanged.

Fork the repository on GitHub, clone your fork, and create a branch:

```sh
git clone https://github.com/YOUR-USERNAME/logfmt.git
cd logfmt
git switch -c describe-your-change
make check
```

`make check` checks formatting, runs `go vet`, and tests both modules. It does
not rewrite source files or install a linter. Go downloads the comparison
module's dependencies on the first run; the library itself uses only the
standard library.

Make is optional. These are the equivalent checks:

```sh
gofmt -l .            # should print no filenames
go vet ./...
go test -cover ./...
cd bench
go vet ./...
go test ./...
```

On Go 1.21, use `make test-root` or `go test ./...` from the repository root.
Running `go test ./...` there does **not** test the nested `bench/` module.

## Everyday commands

Run `make` or `make help` to see the commands without changing any files.

| Command | Purpose |
|---|---|
| `make check` | Format check, vet, and tests for both modules |
| `make test-root` / `make test-bench` | Test either module separately |
| `make test-race` | Test both modules with Go's race detector |
| `make examples` | Run the examples displayed on pkg.go.dev |
| `make fmt` | Apply `gofmt` |
| `make lint` | Run golangci-lint in both modules; install it if missing |
| `make fuzz FUZZTIME=60s` | Run all four differential fuzzers, 60 seconds each |
| `make bench` | Run both benchmark suites without rewriting tables |

`make lint` uses the installed golangci-lint binary. `make update-tools`
installs the pinned version used by CI; the installation needs a toolchain
supported by that linter. `make fix` explicitly formats files and runs
`go mod tidy` in both modules.

## Validate your change

For documentation, check links and run examples you changed. The README quick
start and the package-level `Example` in `example_test.go` should stay in sync.
Runnable examples belong in that file so `go test` verifies their output.
Do not add range-over-func syntax to root-module examples; it needs Go 1.23.

For code, add a regression test that fails before the fix and run `make check`.
CI also runs the race detector on amd64 and arm64 and tests the Go 1.21 floor.
If you cannot check an architecture locally, say so in the pull request.

Run the relevant differential fuzzers after behavior changes:

| Changed behavior | Fuzzer |
|---|---|
| Parsing | `FuzzIterateAgainstRef` |
| Lookup resolution | Both `FuzzGetManyAgainstRef` and `FuzzGetManyIndexedAgainstRef` |
| Escape decoding | `FuzzAppendUnescapeAgainstRef` |

To run one fuzzer:

```sh
go test -run='^$' -fuzz='^FuzzIterateAgainstRef$' -fuzztime=20s -parallel=2 .
```

`make fuzz` runs all four with two workers each; override `FUZZTIME` and
`FUZZPARALLEL` as needed. A new failing input should become a regression test
or a committed corpus entry. Tests also exercise existing seeds and committed
corpus entries without starting a fuzzing session.

Give new source files an `SPDX-License-Identifier: MIT` header. In Go files,
place it before build tags and separate it from the package comment with a
blank line.

## Performance changes

Correctness and the allocation contract come first. Include the workload, Go
version, CPU, architecture, commands, and before/after results in your pull
request. Measure with identical test sources on both sides; avoid drawing
conclusions from one timing run.

See [bench/README.md](bench/README.md) for reproduction and comparison commands.
Read the relevant sections of [CLAUDE.md](CLAUDE.md) before changing hot paths:
it records measurements and approaches already tried. In particular:

- After changing `Iterate`, run `make bsfdep` to inspect amd64 BSF dependencies.
- For changes intended for one architecture, cross-compile both revisions for
  the other and compare them with `bench/asmdiff.py`.
- Test realistic log lines and escape-density/gap shapes, not only a single
  synthetic benchmark.
- Leave `time.go` alone unless timestamp parsing is the explicit scope of the
  change. Compatibility with `time.Parse` is deliberate.

The `bench/lokifmt/` stand-in has its own license notices. Preserve them and do
not replace it with code copied from Loki's AGPL-licensed tree.

## Open a pull request

Explain the problem, the resulting behavior, and how you checked it. Link a
related issue if there is one. Keep unrelated refactors and generated benchmark
tables out of a focused fix; include table updates only when they are part of
the change.

Draft pull requests are welcome for work in progress. A small documentation
fix does not need benchmark results. Code affecting performance does.

The module is pre-1.0. CI tags each green `main` build with the next patch
version; contributors do not need to change version numbers or create tags.
