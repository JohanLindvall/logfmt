# Benchmarks

[Project overview](../README.md) · [Contributing](../CONTRIBUTING.md)

The library's microbenchmarks live in the root module. This separate module
compares it with other parsers without adding dependencies for library users.
Run the commands below from the repository root with Go 1.23 or newer.

## Published results

| Suite | amd64 | arm64 |
|---|---|---|
| Parser comparisons | [Results](results_amd64.md) | [Results](results_arm64.md) |
| Package microbenchmarks | [Results](pkg_results_amd64.md) | [Results](pkg_results_arm64.md) |

The generated tables include their date, Go version, reported CPU, time,
throughput where applicable, and allocations. They are a snapshot of those
workloads and environments, not a guarantee for a different application.
The [shared 1.4 KB sample](../testdata/sample_big.txt) is used by both suites.

## What the comparisons measure

Read [bench_test.go](bench_test.go) for the exact inputs and buffer lifetimes.

| Benchmark | Work performed | Interpretation |
|---|---|---|
| `ParseAll_Big` / `ParseAll_Typical` | Visit all pairs using each parser's normal API | This package returns raw values; the other parsers decode eagerly. These rows do not measure equivalent decoding work on escaped values. |
| `ParseEscaped` | Parse and decode values with escapes | This package explicitly calls `AppendUnescape` and reuses a destination buffer. All backslashes in this fixture belong to quoted values. |
| `Extract` | Read `timestamp` and `level` | Parsers stop early where their APIs allow it. kr/logfmt continues scanning. `GetMany` reuses its result slice. |

Decoder construction/reset follows the code in each benchmark and can affect
allocations. Compare these lifetimes with your own application. A new comparison
should state whether it measures raw parsing, decoding, or extraction.

The Loki entry is a [stand-in](lokifmt/decode.go) adapted from go-logfmt under
MIT and verified against Loki's decoder on these inputs. It is not vendored
from Loki's AGPL-licensed tree; preserve the [license notices](lokifmt/LICENSE).

## Run locally

Run the complete suites without modifying the published tables:

```sh
make bench
```

For a smaller sample:

```sh
go test -run='^$' -bench='Benchmark_(Get|IterateJSONMsg)$' -benchmem -count=5 .
(cd bench && go test -run='^$' -bench='Benchmark_(ParseEscaped|Extract)_' -benchmem -count=5 .)
```

To regenerate this machine's architecture-specific Markdown tables, install
Bash and Python 3, then run:

```sh
make bench-md
```

That command runs each case for two seconds and overwrites
`bench/results_<arch>.md` and `bench/pkg_results_<arch>.md`. Inspect the diff
before including generated results in a pull request. The manually dispatched
[benchmark workflow](../.github/workflows/bench.yml) runs both architectures
and commits their tables together.

## Measure a proposed optimization

Use the same Go toolchain, build flags, benchmark source, and input on both
revisions. Build test binaries with `go test -c -o /tmp/logfmt-before.test .`
from the baseline checkout and `go test -c -o /tmp/logfmt-after.test .` from the
candidate checkout. If you add a benchmark, add it to both before comparing.

On Linux, [compare.py](compare.py) alternates A/B order and pins the binaries
to one CPU. It requires `taskset` and `benchstat` on `PATH`. First run an A/A
control using the baseline binary twice, then compare the candidate. Choose a
CPU from the allowed list printed by `taskset -pc $$`:

```sh
BENCH_CPU=2 # replace with an available CPU on your machine
python3 bench/compare.py /tmp/logfmt-before.test /tmp/logfmt-before.test \
    --cpu "$BENCH_CPU" --output /tmp/logfmt-aa \
    --bench 'Benchmark_(IterateOur|Get|GetMany|Unescape)'
python3 bench/compare.py /tmp/logfmt-before.test /tmp/logfmt-after.test \
    --cpu "$BENCH_CPU" --output /tmp/logfmt-ab \
    --bench 'Benchmark_(IterateOur|Get|GetMany|Unescape)'
```

Keep builds, fuzzers, and other benchmarks out of the measurement interval.
Investigate an unstable control before attributing a timing change to code.
Report the environment, commands, allocation changes, improvements, and
regressions together.

For hardware-counter work, [counters.py](counters.py) reports per-operation
cycles, operations, branches, and misses using `perf`. See `--help` and
[the measurement notes](../CLAUDE.md#methodology-use-this-for-any-future-perf-work)
for event support and interpretation.

## Choose representative workloads

Short fields emphasize callback and per-field costs. Long values emphasize
scanning. Escaped JSON, Unicode, absent keys, duplicate queries, and buffer
capacity each exercise different paths. Use the existing benchmarks to cover
the shapes affected by a change:

- `Benchmark_IterateFieldShape` and `Benchmark_Get` for field traversal.
- `Benchmark_IterateEscaped`, `Benchmark_IterateEscapedGap`,
  `Benchmark_IterateEscapedAlternating`, `Benchmark_IteratePrefixJSON`, and
  `Benchmark_IterateJSONMsg` for escape density and transitions.
- `Benchmark_UnescapeEscaped`, `Benchmark_UnescapeEscapedGap`,
  `Benchmark_UnescapeUnicode`, and `Benchmark_AppendValueUnicode` for decoding.
- `Benchmark_GetManyShapes` and the lookup benchmarks in
  [perf_bench_test.go](../perf_bench_test.go) for query count and lookup behavior.

`GetMany` needs space in its result slice to avoid allocating. Append functions
need destination capacity; benchmark fresh and reused buffers separately.
Malformed records can allocate a `SyntaxError`. Timestamp parsing has its own
allocation behavior, documented on `ParseTime`.

## Architecture details

The escape scans and spare-capacity decoder are tuned separately for arm64.
Results on one architecture do not establish a win on another. Read the
[arm64 lookup report](perf_2026-09-09_arm64.md),
[arm64 escape report](perf_2026-09-23_arm64.md), and
[performance history](../CLAUDE.md) before repeating earlier experiments.

On amd64, consumer builds can choose `GOAMD64=v3` for CPUs supporting that
feature level. Earlier measurements found gains on some typical lines and
regressions on some escape-dense inputs. Keep the flag identical in A/B builds
and report it; the library does not require v3.

After a change to `Iterate`, run `make bsfdep` to inspect possible amd64 BSF
register dependencies. It cross-compiles, so it also works on an arm64 host.
For a change intended for only one architecture, cross-compile both revisions
for the other and run:

```sh
python3 bench/asmdiff.py /tmp/logfmt-before.test /tmp/logfmt-after.test
```

The comparison ignores addresses and line numbers to show which functions'
machine code changed. Record any platform you could not measure.
