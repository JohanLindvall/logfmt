# AGENTS.md — logfmt

A fast, allocation-free, dependency-free reader for logfmt. Preserve its small,
read-only, raw-by-default API.

## Scope and constraints

- **Only touch `time.go` when explicitly asked to optimize timestamp parsing.**
  Its `time.Parse` behavior is intentional, including acceptance, rejection,
  timezone, and calendar quirks.
- Keep the root module at **Go 1.21** and the separate `bench/` module at
  **Go 1.23**. Do not import `iter` in the library. `All` returns a bare function
  type; range-over-func examples belong in the bench module.
- Keep the library dependency-free. Comparison dependencies belong in `bench/`.
- Keep the package a reader: no encoder, streaming decoder, map-building API,
  or typed accessors beyond timestamps.
- Add an `SPDX-License-Identifier: MIT` header to new source files. In Go,
  separate it from the package comment and place it before build tags.
- Preserve the license notices in `bench/lokifmt/`. It is a go-logfmt-based
  stand-in; do not replace it with code copied from Loki's AGPL-licensed tree.
- Keep the README quick start and package-level `Example` in `example_test.go`
  in sync. Usage recipes should have executable examples where practical.

## Repository map

| Location | Purpose |
|---|---|
| `logfmt.go` | Parser, lookups, SWAR masks, and general escape decoder |
| `scan_arm64.go`, `scan_other.go` | Architecture-specific dense and sparse quoted scans |
| `unescape_spare.go` | Spare-capacity decoder selected on arm64 |
| `time.go` | Timestamp parsing; restricted scope above |
| `doc.go` | API contracts, framing, aliasing, errors, and leniency |
| `example_test.go`, `docs/usage.md` | Runnable examples and usage recipes |
| `*_test.go`, `testdata/fuzz/` | Unit tests, benchmarks, differential fuzzers, and corpus |
| `testdata/sample_big.txt` | Shared benchmark input for both modules |
| `bench/` | Comparison module, measurement tools, and result tables |
| `CONTRIBUTING.md`, `bench/README.md` | Development checks and benchmark reproduction |

## Behavioral contracts

- Inputs and values are `[]byte`; lookup keys are `string`. Parsed slices are
  read-only and alias the input, except bare keys' shared `trueSlice`.
  `IsBareKey` tests sentinel identity, not the bytes `"true"`.
- Values are raw, with surrounding quotes removed and escapes retained.
  Backslashes have escape meaning only in quoted values. `AppendValue` must
  copy unquoted values literally; callers decoding raw values need the bit
  from `GetQuoted`.
- All lookups resolve duplicates identically: the first non-empty occurrence
  wins; an empty value wins only if no non-empty occurrence exists.
- Absence is comma-ok, or a `nil` slot in `GetMany`. A present empty value is a
  non-nil empty slice. Duplicate query slots must each receive their result.
- `Get`, `GetQuoted`, and `GetMany` cap returned values to their length.
  `Iterate` and `All` leave capacity through the end of the record.
  **Do not cap callback values to their length:** `wasQuoted` reads the byte
  just past a value to distinguish a closing quote from whitespace or EOF.
- `AppendValue` and `AppendUnescape` always append to the destination.
  A separate destination makes the result independent of the input.
  `AppendUnescape(raw[:0], raw)` and decoding behind a prefix in the same
  buffer must remain safe.
- `Iterate` delivers the valid prefix before returning a `*SyntaxError`;
  `errors.Is(err, ErrBadFormat)` must match. Returning false from the callback
  stops parsing. Lookups and `All` discard errors; `Validate` checks the entire
  input.
- Newlines and carriage returns are whitespace, not record boundaries.
  Callers use `SplitRecord` for framing; it trims a trailing `\r` and caps the
  record.
- Preserve the leniency documented in `doc.go`: empty keys, bare keys, literal
  quotes/backslashes/equals in unquoted values, non-whitespace control bytes,
  and unquoted keys. `key= value` yields an empty `key` then a bare `value`.
  Unknown escapes decode to the escaped byte; malformed Unicode escapes stay
  verbatim. Valid Unicode escapes include surrogate pairs.
- Preserve allocation guarantees with reused buffers. A reached syntax error
  allocates; result/destination growth and `ParseTime` have separate allocation
  behavior. Callback escape analysis can also move a caller's input buffer to
  the heap.

## Hot-path invariants

### Parser

- SWAR masks can set spurious bits above a real match because of byte borrows.
  Combine masks with OR, never subtraction. Re-check a selected stop byte;
  the escape walk must re-check every drained lane for both quote and backslash.
- A backslash in lane 7 escapes a byte outside the loaded word. The walk can
  hand back `i == len(data)+1` after a trailing backslash; the sparse scan must
  report an unterminated value without slicing beyond the input.
- Keep fixed-width `binary.LittleEndian.Uint64` loads and the bounds proofs
  around them. The unsigned outer bound, `i <= n-8`, and explicit non-negative
  checks work together. Verify changes with
  `go test -gcflags='-d=ssa/check_bce/debug=1' .`.
- The wide key scan seeds the value scan from its sixteen-byte view. Load the
  second word only when needed; preserve the narrow loop for the final 8–15
  bytes and the control-byte fallback.
- Load `swarRegs` at each field, keeping constants out of callback spills.
  Load order (`sub`, `hi`, `xor`) affects register allocation. On amd64 at
  `GOAMD64=v1`, BSF can depend on its destination's previous value:
  run `make bsfdep` after changing `Iterate` and investigate every `SUSPECT`.
- Escape thresholds differ by architecture: arm64 uses `escClean=8`,
  `escGap=64`; other architectures use `5`, `48`. On arm64 a sparse scan can
  return to the dense walk only once, and only for a value declined on arrival.
  Keep the upgrade threshold below the clean-run exit threshold and test
  alternating short/long gaps to prevent repeated failed probes.

### Lookups

- `GetMany` returns immediately for zero queries. It uses a lazy stack index
  for 32–256 queries when the record is at least four bytes per requested key;
  other shapes use linear matching.
- Keep key-length rejection before index construction. Bucket chains retain
  query order; full comparisons verify matches. Settling a non-empty value
  removes a slot, while an empty value remains provisional.
- `resetSlots`, `lenSet`, and `settle` keep both paths consistent. Slots
  themselves record whether a result is absent, empty, or settled.
- `keyBucket` accepts strings and byte slices directly. Converting a parsed
  key to a string there copies it and can allocate; equality conversions have
  different compiler behavior.

### Decoding

- Keep `NeedsUnescape` a single `bytes.IndexByte` expression so it stays
  inlinable. `AppendUnescape` also uses `IndexByte` for the first backslash.
- The general decoder uses `unescWindow=4` and skips scanning between adjacent
  escapes. Preserve the `s >= 0` probe-loop guard: it supplies the bounds proof
  for the word load.
- A destination with zero capacity reserves raw length once. Existing buffers
  must still avoid allocating when they fit the decoded output but not all of
  the raw input.
- `unescapeIntoSpare` selects the spare-capacity decoder only on arm64, when
  the destination has room for all of `raw`. Its `spareWindow=8` probe is
  adaptive. Literal copies must load their bytes before storing and must not
  overwrite unread input.
- Preserve the table-based non-Unicode escape decoding and `hex4` validation.
  Measure Unicode, surrogate, malformed, adjacent-escape, fresh-buffer, and
  reused-buffer paths when changing decoding.

## Validation

Run `make check` for code changes; it checks formatting, vets, and tests both
modules without rewriting source. Root `go test ./...` does not cover `bench/`.
For documentation changes, check links and run any examples changed.

```sh
make check
make test-race
make examples
make fuzz FUZZTIME=20s FUZZPARALLEL=2
```

Run the relevant differential fuzzers after behavior changes:

| Changed behavior | Required fuzzers |
|---|---|
| Parsing | `FuzzIterateAgainstRef` |
| Lookup resolution | `FuzzGetManyAgainstRef` and `FuzzGetManyIndexedAgainstRef` |
| Escape decoding | `FuzzAppendUnescapeAgainstRef` |

To run one:

```sh
go test -run='^$' -fuzz='^FuzzIterateAgainstRef$' -fuzztime=20s -parallel=2 .
```

Preserve these checks when modifying tests:

- Parser comparisons include key, value, `IsBareKey`, and quoted state, even
  when parsing returns an error. Keep malformed and word-boundary seeds.
- Keep exhaustive SWAR-mask tests and the separate `isSpace` test; the parser
  reference shares `isSpace`, so differential fuzzing cannot detect its errors.
- Both lookup fuzzers are needed for duplicate resolution and indexed query
  slots. Their use of `AppendUnescape` is not an independent decoder oracle.
- Decoder fuzzing checks fresh, prefixed, spare-capacity, and overlapping
  destinations. Keep non-periodic literal runs around copy/probe boundaries
  so off-by-one copies cannot hide behind repeated bytes.
- Retain allocation, actual input-aliasing, capped-value, exact-capacity reuse,
  and unquoted-backslash tests. Commit new failing fuzz inputs as regressions.

## Performance work

- Use identical test sources, inputs, toolchains, and flags for both builds.
  Prebuild binaries, run an A/A control, then use `bench/compare.py` for
  alternating A/B rounds pinned to one CPU. Compare with `benchstat`.
- Keep builds, fuzzers, lint, and other benchmark series outside measurement
  intervals. Aim to keep a full series within ten minutes; extend only an
  ambiguous finalist.
- Use `bench/counters.py` to inspect per-operation cycles, operations,
  instructions, branches, and misses. It selects host-specific events;
  `--portable` uses generic events. Confirm candidates with timing series.
- Inspect generated code and register dependencies. Profile samples can land
  after a stall; fewer instructions or a large flat profile share alone do
  not establish a speedup. Calibrate small changes against code-layout effects.
- For architecture-specific changes, isolate the implementation with build
  tags or a compile-time architecture constant. Cross-compile both revisions
  for unaffected architectures and compare with `bench/asmdiff.py`; any
  machine-code difference needs measurement on that architecture.
- Include realistic inputs and affected shapes: short fields, shallow/deep/
  absent lookups, escape-density and gap sweeps, prose followed by JSON,
  alternating gaps, Unicode, collisions, duplicate queries, and buffer reuse.
  Keep `testdata/sample_big.txt` shared between suites.
- Retain measured improvements with understood tradeoffs; revert neutral
  complexity. Report CPU, architecture, Go version, commands, allocation
  changes, improvements, and regressions. Distinguish raw parsing from decoding.
- Anchor every slash-separated benchmark component, for example
  `^Benchmark_IterateFieldShape$/^quoted$`, so it cannot also select `unquoted`.
  See [bench/README.md](bench/README.md) for reproduction commands.
