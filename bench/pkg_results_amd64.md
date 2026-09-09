# logfmt microbenchmarks

- generated 2026-09-09T07:31:58Z
- go version go1.27.1 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 323.7 | — | 0 | 0 |
| GetMany_TimestampLevel | 77.2 | — | 0 | 0 |
| Unescape | 28.1 | — | 0 | 0 |
| IterateEscaped/esc=0 | 27.0 | 38180.98 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 83.7 | 12299.46 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 279.6 | 3683.31 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 503.2 | 2046.73 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 1339.0 | 769.28 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 32.9 | 31163.77 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 139.7 | 7332.18 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 342.4 | 2990.27 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 817.5 | 1252.54 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 3257.0 | 314.35 MB/s | 0 | 0 |
| IterateJSONMsg | 144.1 | 1796.93 MB/s | 0 | 0 |
| UnescapeJSONMsg | 199.6 | 992.13 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 575635.0 | 868.61 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 365.6 | 2817.00 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 278.1 | 3703.88 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 257.0 | 4008.19 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 194.6 | 5292.91 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 133.8 | 7700.27 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 86.1 | 11957.59 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 55.4 | 18591.43 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 530.3 | 1931.13 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 340.8 | 3004.87 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 396.5 | 2582.37 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 336.6 | 3042.36 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 250.9 | 4081.75 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 134.1 | 7633.69 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 78.9 | 12980.04 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 79.0 | 1682.95 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 79.3 | 1979.96 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 145.3 | 1301.07 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 147.3 | 1935.40 MB/s | 0 | 0 |
| UnescapeUnicode | 95.6 | 1276.36 MB/s | 0 | 0 |
| AppendValueUnicode | 130.4 | 1142.72 MB/s | 0 | 0 |
| LevelTS_LogFmt | 64.8 | — | 0 | 0 |
| LevelTS_Regex | 15395.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 90.3 | — | 0 | 0 |
| ParseTime_Custom | 389.9 | — | 164 | 4 |
| ParseTime_Unix | 84.9 | — | 0 | 0 |
