# logfmt microbenchmarks

- generated 2026-08-22T22:51:16Z
- go version go1.27.0 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 332.0 | — | 0 | 0 |
| GetMany_TimestampLevel | 74.9 | — | 0 | 0 |
| Unescape | 25.4 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.3 | 29200.84 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 104.6 | 9846.58 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 218.7 | 4709.39 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 556.4 | 1851.35 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2106.0 | 489.02 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 49.5 | 20702.10 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 132.4 | 7735.61 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 275.2 | 3721.27 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 740.0 | 1383.74 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 2986.0 | 342.96 MB/s | 0 | 0 |
| IterateJSONMsg | 163.9 | 1579.80 MB/s | 0 | 0 |
| UnescapeJSONMsg | 189.0 | 1047.59 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 614325.0 | 813.90 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 331.4 | 3108.04 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 218.6 | 4712.08 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 208.0 | 4951.61 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 226.3 | 4551.26 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 176.4 | 5837.74 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 105.6 | 9749.77 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 68.6 | 15015.37 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 418.8 | 2444.96 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 274.6 | 3729.47 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 357.9 | 2860.78 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 301.7 | 3394.36 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 228.1 | 4489.20 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 131.3 | 7797.06 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 80.1 | 12784.11 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 93.7 | 1419.26 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 94.7 | 1657.43 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 164.8 | 1146.93 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 168.3 | 1693.41 MB/s | 0 | 0 |
| UnescapeUnicode | 79.5 | 1534.16 MB/s | 0 | 0 |
| AppendValueUnicode | 113.6 | 1311.72 MB/s | 0 | 0 |
| LevelTS_LogFmt | 61.5 | — | 0 | 0 |
| LevelTS_Regex | 13802.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 70.6 | — | 0 | 0 |
| ParseTime_Custom | 376.3 | — | 164 | 4 |
| ParseTime_Unix | 69.6 | — | 0 | 0 |
