# logfmt microbenchmarks

- generated 2026-06-24T19:52:52Z
- go version go1.26.3 linux/amd64
- cpu: AMD EPYC 9V74 80-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 470.6 | — | 0 | 0 |
| GetMany_TimestampLevel | 91.1 | — | 0 | 0 |
| Unescape | 27.7 | — | 0 | 0 |
| DecodeKeyval_Custom | 664977.0 | 751.91 MB/s | 0 | 0 |
| LevelTS_LogFmt | 73.9 | — | 0 | 0 |
| LevelTS_Regex | 14666.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 69.5 | — | 0 | 0 |
| ParseTime_Custom | 331.7 | — | 164 | 4 |
| ParseTime_Unix | 78.6 | — | 0 | 0 |
