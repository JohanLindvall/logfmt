# logfmt microbenchmarks

- generated 2026-07-26T06:08:56Z
- go version go1.26.3 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 432.6 | — | 0 | 0 |
| GetMany_TimestampLevel | 86.3 | — | 0 | 0 |
| Unescape | 28.6 | — | 0 | 0 |
| DecodeKeyval_Custom | 699571.0 | 714.72 MB/s | 0 | 0 |
| LevelTS_LogFmt | 72.8 | — | 0 | 0 |
| LevelTS_Regex | 13615.0 | — | 1077 | 4 |
| ParseTime_RFC3339 | 65.9 | — | 0 | 0 |
| ParseTime_Custom | 393.4 | — | 164 | 4 |
| ParseTime_Unix | 65.5 | — | 0 | 0 |
