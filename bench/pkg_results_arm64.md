# logfmt microbenchmarks

- generated 2026-08-17T13:16:29Z
- go version go1.26.6 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 381.9 | — | 0 | 0 |
| GetMany_TimestampLevel | 80.5 | — | 0 | 0 |
| Unescape | 25.5 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.0 | 29396.13 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 137.6 | 7486.63 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 250.3 | 4115.86 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 569.6 | 1808.23 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2170.0 | 474.69 MB/s | 0 | 0 |
| IterateJSONMsg | 171.0 | 1514.94 MB/s | 0 | 0 |
| UnescapeJSONMsg | 190.7 | 1038.21 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 719579.0 | 694.85 MB/s | 0 | 0 |
| LevelTS_LogFmt | 69.8 | — | 0 | 0 |
| LevelTS_Regex | 13711.0 | — | 1077 | 4 |
| ParseTime_RFC3339 | 70.2 | — | 0 | 0 |
| ParseTime_Custom | 372.5 | — | 164 | 4 |
| ParseTime_Unix | 69.0 | — | 0 | 0 |
