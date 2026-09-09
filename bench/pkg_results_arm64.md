# logfmt microbenchmarks

- generated 2026-09-09T07:31:56Z
- go version go1.27.1 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 299.4 | — | 0 | 0 |
| GetMany_TimestampLevel | 66.7 | — | 0 | 0 |
| Unescape | 25.0 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.6 | 28933.92 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 105.5 | 9761.30 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 198.0 | 5202.54 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 288.0 | 3575.79 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 735.3 | 1400.76 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 49.4 | 20728.37 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 131.4 | 7795.76 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 274.4 | 3731.47 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 735.2 | 1392.79 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 2996.0 | 341.82 MB/s | 0 | 0 |
| IterateJSONMsg | 96.5 | 2682.95 MB/s | 0 | 0 |
| UnescapeJSONMsg | 187.6 | 1055.32 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 595254.0 | 839.98 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 228.0 | 4517.06 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 197.6 | 5213.14 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 188.0 | 5478.84 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 226.9 | 4539.08 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 178.0 | 5787.75 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 105.2 | 9792.73 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 69.4 | 14850.67 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 415.3 | 2465.42 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 273.1 | 3749.37 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 357.6 | 2863.66 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 302.7 | 3382.47 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 227.2 | 4507.61 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 130.4 | 7851.33 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 79.4 | 12900.71 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 54.2 | 2454.76 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 54.3 | 2893.25 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 165.1 | 1144.94 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 168.8 | 1688.15 MB/s | 0 | 0 |
| UnescapeUnicode | 80.8 | 1510.31 MB/s | 0 | 0 |
| AppendValueUnicode | 114.0 | 1306.95 MB/s | 0 | 0 |
| LevelTS_LogFmt | 58.7 | — | 0 | 0 |
| LevelTS_Regex | 13958.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 69.4 | — | 0 | 0 |
| ParseTime_Custom | 377.4 | — | 164 | 4 |
| ParseTime_Unix | 69.5 | — | 0 | 0 |
