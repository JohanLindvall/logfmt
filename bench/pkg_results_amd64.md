# logfmt microbenchmarks

- generated 2026-08-17T16:44:35Z
- go version go1.26.6 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 396.7 | — | 0 | 0 |
| GetMany_TimestampLevel | 87.6 | — | 0 | 0 |
| Unescape | 28.8 | — | 0 | 0 |
| IterateEscaped/esc=0 | 27.2 | 37851.34 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 94.3 | 10920.09 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 264.5 | 3894.25 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 631.1 | 1632.06 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2371.0 | 434.35 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 35.9 | 28536.35 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 140.5 | 7286.78 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 360.0 | 2844.09 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 852.5 | 1201.18 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 3328.0 | 307.69 MB/s | 0 | 0 |
| IterateJSONMsg | 186.6 | 1387.98 MB/s | 0 | 0 |
| UnescapeJSONMsg | 206.7 | 957.90 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 686483.0 | 728.35 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 374.3 | 2751.55 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 262.8 | 3919.86 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 242.1 | 4253.63 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 218.7 | 4710.14 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 155.0 | 6644.58 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 94.5 | 10898.00 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 63.9 | 16109.29 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 554.4 | 1846.88 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 359.5 | 2848.35 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 414.4 | 2471.33 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 356.2 | 2874.71 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 260.1 | 3937.22 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 140.1 | 7309.75 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 82.5 | 12416.96 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 105.1 | 1265.09 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 106.4 | 1476.03 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 163.6 | 1155.46 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 168.5 | 1691.83 MB/s | 0 | 0 |
| UnescapeUnicode | 100.5 | 1213.61 MB/s | 0 | 0 |
| AppendValueUnicode | 136.9 | 1088.46 MB/s | 0 | 0 |
| LevelTS_LogFmt | 66.3 | — | 0 | 0 |
| LevelTS_Regex | 15426.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 70.8 | — | 0 | 0 |
| ParseTime_Custom | 415.1 | — | 164 | 4 |
| ParseTime_Unix | 80.1 | — | 0 | 0 |
