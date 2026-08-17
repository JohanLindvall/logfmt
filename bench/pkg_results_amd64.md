# logfmt microbenchmarks

- generated 2026-08-17T13:16:29Z
- go version go1.26.6 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 394.0 | — | 0 | 0 |
| GetMany_TimestampLevel | 95.3 | — | 0 | 0 |
| Unescape | 29.1 | — | 0 | 0 |
| IterateEscaped/esc=0 | 27.4 | 37570.18 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 154.8 | 6651.78 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 294.0 | 3503.05 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 671.2 | 1534.67 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2549.0 | 404.12 MB/s | 0 | 0 |
| IterateJSONMsg | 196.2 | 1320.04 MB/s | 0 | 0 |
| UnescapeJSONMsg | 218.3 | 907.05 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 681523.0 | 733.65 MB/s | 0 | 0 |
| LevelTS_LogFmt | 72.6 | — | 0 | 0 |
| LevelTS_Regex | 15018.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 70.8 | — | 0 | 0 |
| ParseTime_Custom | 427.0 | — | 164 | 4 |
| ParseTime_Unix | 91.0 | — | 0 | 0 |
