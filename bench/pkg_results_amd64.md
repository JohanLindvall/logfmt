# logfmt microbenchmarks

- generated 2026-08-08T16:36:35Z
- go version go1.26.5 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 414.2 | — | 0 | 0 |
| GetMany_TimestampLevel | 89.2 | — | 0 | 0 |
| Unescape | 28.8 | — | 0 | 0 |
| IterateEscaped/esc=0 | 28.6 | 36016.32 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 92.0 | 11191.54 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 305.2 | 3375.29 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 1129.0 | 911.97 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 4410.0 | 233.56 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 691945.0 | 722.60 MB/s | 0 | 0 |
| LevelTS_LogFmt | 75.5 | — | 0 | 0 |
| LevelTS_Regex | 19038.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 71.7 | — | 0 | 0 |
| ParseTime_Custom | 441.2 | — | 164 | 4 |
| ParseTime_Unix | 75.7 | — | 0 | 0 |
