# logfmt microbenchmarks

- generated 2026-07-02T19:44:04Z
- go version go1.26.3 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 437.3 | — | 0 | 0 |
| GetMany_TimestampLevel | 86.0 | — | 0 | 0 |
| Unescape | 28.6 | — | 0 | 0 |
| DecodeKeyval_Custom | 706096.0 | 708.12 MB/s | 0 | 0 |
| LevelTS_LogFmt | 73.3 | — | 0 | 0 |
| LevelTS_Regex | 13605.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 65.8 | — | 0 | 0 |
| ParseTime_Custom | 384.8 | — | 164 | 4 |
| ParseTime_Unix | 66.8 | — | 0 | 0 |
