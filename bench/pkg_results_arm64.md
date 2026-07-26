# logfmt microbenchmarks

- generated 2026-07-26T12:27:50Z
- go version go1.26.5 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 423.0 | — | 0 | 0 |
| GetMany_TimestampLevel | 84.8 | — | 0 | 0 |
| Unescape | 28.5 | — | 0 | 0 |
| DecodeKeyval_Custom | 697402.0 | 716.95 MB/s | 0 | 0 |
| LevelTS_LogFmt | 72.7 | — | 0 | 0 |
| LevelTS_Regex | 13819.0 | — | 1077 | 4 |
| ParseTime_RFC3339 | 71.5 | — | 0 | 0 |
| ParseTime_Custom | 426.9 | — | 212 | 5 |
| ParseTime_Unix | 69.0 | — | 0 | 0 |
