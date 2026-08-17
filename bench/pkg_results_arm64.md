# logfmt microbenchmarks

- generated 2026-08-17T16:44:35Z
- go version go1.26.6 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 377.1 | — | 0 | 0 |
| GetMany_TimestampLevel | 75.1 | — | 0 | 0 |
| Unescape | 25.3 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.5 | 28977.45 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 108.5 | 9493.03 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 220.6 | 4669.75 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 557.2 | 1848.62 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2109.0 | 488.38 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 49.9 | 20517.05 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 131.9 | 7760.59 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 282.2 | 3628.93 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 733.8 | 1395.50 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 2976.0 | 344.12 MB/s | 0 | 0 |
| IterateJSONMsg | 168.2 | 1539.88 MB/s | 0 | 0 |
| UnescapeJSONMsg | 187.8 | 1054.59 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 724302.0 | 690.32 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 331.7 | 3104.80 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 220.5 | 4670.58 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 211.7 | 4866.51 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 233.8 | 4405.69 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 181.2 | 5685.22 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 108.6 | 9484.35 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 70.4 | 14626.14 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 423.9 | 2415.71 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 279.7 | 3660.89 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 364.4 | 2809.75 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 304.9 | 3358.15 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 229.3 | 4465.11 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 132.0 | 7755.51 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 82.0 | 12493.63 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 93.3 | 1425.02 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 95.7 | 1640.23 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 175.9 | 1074.70 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 178.2 | 1599.11 MB/s | 0 | 0 |
| UnescapeUnicode | 81.9 | 1489.20 MB/s | 0 | 0 |
| AppendValueUnicode | 116.7 | 1277.12 MB/s | 0 | 0 |
| LevelTS_LogFmt | 64.9 | — | 0 | 0 |
| LevelTS_Regex | 13740.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 72.0 | — | 0 | 0 |
| ParseTime_Custom | 371.9 | — | 164 | 4 |
| ParseTime_Unix | 69.3 | — | 0 | 0 |
