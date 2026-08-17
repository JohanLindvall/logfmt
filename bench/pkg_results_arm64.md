# logfmt microbenchmarks

- generated 2026-08-17T14:11:21Z
- go version go1.26.6 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 377.4 | — | 0 | 0 |
| GetMany_TimestampLevel | 74.5 | — | 0 | 0 |
| Unescape | 25.3 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.6 | 28964.77 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 108.9 | 9462.11 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 220.7 | 4666.91 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 557.3 | 1848.13 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2110.0 | 488.14 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 49.9 | 20519.28 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 136.0 | 7531.39 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 289.2 | 3540.72 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 732.2 | 1398.53 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 2986.0 | 342.88 MB/s | 0 | 0 |
| IterateJSONMsg | 168.2 | 1540.25 MB/s | 0 | 0 |
| UnescapeJSONMsg | 190.0 | 1041.88 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 724223.0 | 690.40 MB/s | 0 | 0 |
| LevelTS_LogFmt | 64.7 | — | 0 | 0 |
| LevelTS_Regex | 14383.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 70.7 | — | 0 | 0 |
| ParseTime_Custom | 399.2 | — | 164 | 4 |
| ParseTime_Unix | 68.8 | — | 0 | 0 |
