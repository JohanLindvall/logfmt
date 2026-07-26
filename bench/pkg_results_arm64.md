# logfmt microbenchmarks

- generated 2026-07-26T11:53:16Z
- go version go1.26.5 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 421.8 | — | 0 | 0 |
| GetMany_TimestampLevel | 84.2 | — | 0 | 0 |
| Unescape | 28.7 | — | 0 | 0 |
| DecodeKeyval_Custom | 695692.0 | 718.71 MB/s | 0 | 0 |
| LevelTS_LogFmt | 72.5 | — | 0 | 0 |
| LevelTS_Regex | 13605.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 65.4 | — | 0 | 0 |
| ParseTime_Custom | 397.2 | — | 164 | 4 |
| ParseTime_Unix | 65.0 | — | 0 | 0 |
