# logfmt microbenchmarks

- generated 2026-07-02T19:43:57Z
- go version go1.26.3 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 469.5 | — | 0 | 0 |
| GetMany_TimestampLevel | 96.3 | — | 0 | 0 |
| Unescape | 29.0 | — | 0 | 0 |
| DecodeKeyval_Custom | 696530.0 | 717.84 MB/s | 0 | 0 |
| LevelTS_LogFmt | 76.5 | — | 0 | 0 |
| LevelTS_Regex | 15932.0 | — | 1077 | 4 |
| ParseTime_RFC3339 | 76.1 | — | 0 | 0 |
| ParseTime_Custom | 404.6 | — | 164 | 4 |
| ParseTime_Unix | 81.3 | — | 0 | 0 |
