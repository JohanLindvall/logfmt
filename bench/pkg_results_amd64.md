# logfmt microbenchmarks

- generated 2026-08-22T22:51:18Z
- go version go1.27.0 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 359.1 | — | 0 | 0 |
| GetMany_TimestampLevel | 86.9 | — | 0 | 0 |
| Unescape | 27.6 | — | 0 | 0 |
| IterateEscaped/esc=0 | 27.2 | 37846.37 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 82.6 | 12476.19 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 249.0 | 4136.08 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 630.5 | 1633.69 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2377.0 | 433.36 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 33.0 | 30988.44 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 142.3 | 7197.35 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 342.3 | 2991.13 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 818.3 | 1251.43 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 3215.0 | 318.54 MB/s | 0 | 0 |
| IterateJSONMsg | 183.4 | 1412.28 MB/s | 0 | 0 |
| UnescapeJSONMsg | 199.9 | 990.27 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 585004.0 | 854.70 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 367.5 | 2802.76 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 249.9 | 4121.97 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 228.2 | 4513.85 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 192.9 | 5340.43 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 141.8 | 7261.23 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 82.7 | 12448.85 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 55.4 | 18582.28 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 531.6 | 1926.10 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 340.5 | 3006.93 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 397.4 | 2576.47 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 334.6 | 3060.82 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 251.4 | 4073.67 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 143.5 | 7138.37 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 78.8 | 12998.97 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 105.7 | 1258.42 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 106.9 | 1467.99 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 146.0 | 1294.29 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 149.1 | 1912.08 MB/s | 0 | 0 |
| UnescapeUnicode | 95.0 | 1284.40 MB/s | 0 | 0 |
| AppendValueUnicode | 136.0 | 1095.44 MB/s | 0 | 0 |
| LevelTS_LogFmt | 65.5 | — | 0 | 0 |
| LevelTS_Regex | 19511.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 90.9 | — | 0 | 0 |
| ParseTime_Custom | 402.7 | — | 164 | 4 |
| ParseTime_Unix | 95.0 | — | 0 | 0 |
