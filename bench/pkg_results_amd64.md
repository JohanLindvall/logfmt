# logfmt microbenchmarks

- generated 2026-07-26T11:53:16Z
- go version go1.26.5 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 442.9 | — | 0 | 0 |
| GetMany_TimestampLevel | 95.8 | — | 0 | 0 |
| Unescape | 28.9 | — | 0 | 0 |
| DecodeKeyval_Custom | 681753.0 | 733.40 MB/s | 0 | 0 |
| LevelTS_LogFmt | 74.2 | — | 0 | 0 |
| LevelTS_Regex | 15535.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 80.8 | — | 0 | 0 |
| ParseTime_Custom | 394.0 | — | 164 | 4 |
| ParseTime_Unix | 75.9 | — | 0 | 0 |
