# logfmt microbenchmarks

- generated 2026-07-26T06:08:55Z
- go version go1.26.3 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 466.5 | — | 0 | 0 |
| GetMany_TimestampLevel | 98.4 | — | 0 | 0 |
| Unescape | 28.3 | — | 0 | 0 |
| DecodeKeyval_Custom | 693150.0 | 721.34 MB/s | 0 | 0 |
| LevelTS_LogFmt | 72.5 | — | 0 | 0 |
| LevelTS_Regex | 15764.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 78.7 | — | 0 | 0 |
| ParseTime_Custom | 406.5 | — | 164 | 4 |
| ParseTime_Unix | 85.2 | — | 0 | 0 |
