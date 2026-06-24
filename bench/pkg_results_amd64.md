# logfmt microbenchmarks

- generated 2026-06-24T19:08:58Z
- go version go1.26.3 linux/amd64
- cpu: AMD Ryzen 7 8840HS w/ Radeon 780M Graphics (16 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 372.5 | — | 0 | 0 |
| GetMany_TimestampLevel | 72.9 | — | 0 | 0 |
| UnescapeInto | 21.5 | — | 0 | 0 |
| DecodeKeyval_Custom | 531931.0 | 939.97 MB/s | 0 | 0 |
| LevelTS_LogFmt | 61.3 | — | 0 | 0 |
| LevelTS_Regex | 12198.0 | — | 1077 | 4 |
| ParseTime_RFC3339 | 54.7 | — | 0 | 0 |
| ParseTime_Custom | 200.3 | — | 0 | 0 |
| ParseTime_Unix | 61.1 | — | 0 | 0 |
