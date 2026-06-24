# logfmt microbenchmarks

- generated 2026-06-24T19:53:03Z
- go version go1.26.3 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 436.7 | — | 0 | 0 |
| GetMany_TimestampLevel | 87.7 | — | 0 | 0 |
| Unescape | 28.9 | — | 0 | 0 |
| DecodeKeyval_Custom | 706513.0 | 707.70 MB/s | 0 | 0 |
| LevelTS_LogFmt | 73.8 | — | 0 | 0 |
| LevelTS_Regex | 13679.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 66.3 | — | 0 | 0 |
| ParseTime_Custom | 402.0 | — | 164 | 4 |
| ParseTime_Unix | 64.9 | — | 0 | 0 |
