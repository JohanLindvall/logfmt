# logfmt microbenchmarks

- generated 2026-09-01T21:44:12Z
- go version go1.27.0 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 301.5 | — | 0 | 0 |
| GetMany_TimestampLevel | 70.5 | — | 0 | 0 |
| Unescape | 24.9 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.6 | 28944.27 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 105.8 | 9731.16 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 208.4 | 4943.18 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 515.9 | 1996.69 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 1948.0 | 528.84 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 49.4 | 20728.80 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 132.2 | 7743.51 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 272.4 | 3758.78 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 735.6 | 1392.09 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 2988.0 | 342.75 MB/s | 0 | 0 |
| IterateJSONMsg | 153.1 | 1692.01 MB/s | 0 | 0 |
| UnescapeJSONMsg | 187.5 | 1055.90 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 594502.0 | 841.04 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 316.4 | 3255.24 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 208.4 | 4942.48 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 188.7 | 5459.33 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 226.6 | 4546.44 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 177.9 | 5791.16 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 106.1 | 9712.40 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 69.1 | 14905.72 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 415.8 | 2462.87 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 274.2 | 3735.15 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 356.9 | 2869.22 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 301.7 | 3394.66 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 225.9 | 4533.58 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 130.8 | 7827.78 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 79.5 | 12878.64 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 90.0 | 1478.01 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 90.2 | 1741.34 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 165.1 | 1144.45 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 168.4 | 1692.18 MB/s | 0 | 0 |
| UnescapeUnicode | 80.9 | 1508.36 MB/s | 0 | 0 |
| AppendValueUnicode | 114.0 | 1306.62 MB/s | 0 | 0 |
| LevelTS_LogFmt | 59.3 | — | 0 | 0 |
| LevelTS_Regex | 13949.0 | — | 1077 | 4 |
| ParseTime_RFC3339 | 69.8 | — | 0 | 0 |
| ParseTime_Custom | 379.1 | — | 164 | 4 |
| ParseTime_Unix | 69.7 | — | 0 | 0 |
