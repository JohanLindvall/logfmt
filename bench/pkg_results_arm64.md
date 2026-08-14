# logfmt microbenchmarks

- generated 2026-08-14T10:05:51Z
- go version go1.26.5 linux/arm64
- cpu: unknown (2 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 379.4 | — | 0 | 0 |
| GetMany_TimestampLevel | 76.0 | — | 0 | 0 |
| Unescape | 28.6 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.9 | 28703.44 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 108.9 | 9456.52 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 220.5 | 4671.64 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 557.0 | 1849.07 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2112.0 | 487.79 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 730096.0 | 684.84 MB/s | 0 | 0 |
| LevelTS_LogFmt | 65.2 | — | 0 | 0 |
| LevelTS_Regex | 13602.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 68.3 | — | 0 | 0 |
| ParseTime_Custom | 387.8 | — | 164 | 4 |
| ParseTime_Unix | 68.9 | — | 0 | 0 |
