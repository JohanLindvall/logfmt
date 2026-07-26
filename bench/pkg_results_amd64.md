# logfmt microbenchmarks

- generated 2026-07-26T17:38:12Z
- go version go1.26.5 linux/amd64
- cpu: AMD EPYC 9V74 80-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 386.6 | — | 0 | 0 |
| GetMany_TimestampLevel | 84.3 | — | 0 | 0 |
| Unescape | 28.5 | — | 0 | 0 |
| DecodeKeyval_Custom | 655986.0 | 762.21 MB/s | 0 | 0 |
| LevelTS_LogFmt | 66.9 | — | 0 | 0 |
| LevelTS_Regex | 14922.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 73.2 | — | 0 | 0 |
| ParseTime_Custom | 364.8 | — | 212 | 5 |
| ParseTime_Unix | 76.9 | — | 0 | 0 |
