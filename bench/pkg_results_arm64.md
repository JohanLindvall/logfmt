# logfmt microbenchmarks

- generated 2026-07-26T17:38:13Z
- go version go1.26.5 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 388.5 | — | 0 | 0 |
| GetMany_TimestampLevel | 82.0 | — | 0 | 0 |
| Unescape | 28.3 | — | 0 | 0 |
| DecodeKeyval_Custom | 716565.0 | 697.77 MB/s | 0 | 0 |
| LevelTS_LogFmt | 69.5 | — | 0 | 0 |
| LevelTS_Regex | 13688.0 | — | 1077 | 4 |
| ParseTime_RFC3339 | 71.8 | — | 0 | 0 |
| ParseTime_Custom | 419.2 | — | 212 | 5 |
| ParseTime_Unix | 69.2 | — | 0 | 0 |
