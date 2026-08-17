# logfmt microbenchmarks

- generated 2026-08-17T14:11:17Z
- go version go1.26.6 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 392.4 | — | 0 | 0 |
| GetMany_TimestampLevel | 87.5 | — | 0 | 0 |
| Unescape | 29.1 | — | 0 | 0 |
| IterateEscaped/esc=0 | 27.3 | 37734.26 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 94.8 | 10863.50 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 261.9 | 3933.14 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 630.5 | 1633.73 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2372.0 | 434.27 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 35.9 | 28501.37 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 139.8 | 7326.17 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 369.4 | 2771.70 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 932.4 | 1098.24 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 3490.0 | 293.37 MB/s | 0 | 0 |
| IterateJSONMsg | 186.1 | 1392.04 MB/s | 0 | 0 |
| UnescapeJSONMsg | 218.9 | 904.64 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 687474.0 | 727.30 MB/s | 0 | 0 |
| LevelTS_LogFmt | 66.3 | — | 0 | 0 |
| LevelTS_Regex | 15555.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 74.9 | — | 0 | 0 |
| ParseTime_Custom | 412.9 | — | 164 | 4 |
| ParseTime_Unix | 75.6 | — | 0 | 0 |
