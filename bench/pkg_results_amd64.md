# logfmt microbenchmarks

- generated 2026-07-26T12:27:46Z
- go version go1.26.5 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 460.7 | — | 0 | 0 |
| GetMany_TimestampLevel | 93.6 | — | 0 | 0 |
| Unescape | 28.0 | — | 0 | 0 |
| DecodeKeyval_Custom | 676908.0 | 738.65 MB/s | 0 | 0 |
| LevelTS_LogFmt | 73.6 | — | 0 | 0 |
| LevelTS_Regex | 15057.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 81.6 | — | 0 | 0 |
| ParseTime_Custom | 437.2 | — | 212 | 5 |
| ParseTime_Unix | 75.5 | — | 0 | 0 |
