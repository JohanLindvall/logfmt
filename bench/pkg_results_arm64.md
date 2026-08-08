# logfmt microbenchmarks

- generated 2026-08-08T16:36:33Z
- go version go1.26.5 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 391.3 | — | 0 | 0 |
| GetMany_TimestampLevel | 81.4 | — | 0 | 0 |
| Unescape | 28.3 | — | 0 | 0 |
| IterateEscaped/esc=0 | 36.2 | 28462.20 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 107.6 | 9568.26 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 335.2 | 3072.54 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 1195.0 | 862.26 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 4726.0 | 217.93 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 725031.0 | 689.63 MB/s | 0 | 0 |
| LevelTS_LogFmt | 72.1 | — | 0 | 0 |
| LevelTS_Regex | 13754.0 | — | 1077 | 4 |
| ParseTime_RFC3339 | 69.8 | — | 0 | 0 |
| ParseTime_Custom | 402.7 | — | 164 | 4 |
| ParseTime_Unix | 69.2 | — | 0 | 0 |
