# logfmt microbenchmarks

- generated 2026-09-01T21:44:20Z
- go version go1.27.0 linux/amd64
- cpu: INTEL(R) XEON(R) PLATINUM 8573C (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 249.1 | — | 0 | 0 |
| GetMany_TimestampLevel | 60.5 | — | 0 | 0 |
| Unescape | 22.6 | — | 0 | 0 |
| IterateEscaped/esc=0 | 21.5 | 48003.22 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 79.6 | 12932.80 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 204.7 | 5031.90 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 581.0 | 1772.66 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 2213.0 | 465.37 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 24.5 | 41746.58 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 96.5 | 10610.40 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 283.1 | 3616.59 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 664.5 | 1541.04 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 2685.0 | 381.36 MB/s | 0 | 0 |
| IterateJSONMsg | 158.2 | 1636.87 MB/s | 0 | 0 |
| UnescapeJSONMsg | 168.5 | 1175.23 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 480879.0 | 1039.76 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 353.4 | 2914.23 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 203.3 | 5066.70 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 180.3 | 5714.10 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 168.1 | 6128.15 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 142.3 | 7235.90 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 79.3 | 12995.70 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 45.9 | 22438.48 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 432.1 | 2369.93 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 290.0 | 3530.83 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 323.4 | 3166.16 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 278.4 | 3678.08 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 194.7 | 5259.34 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 96.3 | 10635.34 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 55.1 | 18577.03 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 84.9 | 1566.11 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 88.8 | 1768.19 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 129.5 | 1459.41 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 142.7 | 1997.59 MB/s | 0 | 0 |
| UnescapeUnicode | 79.9 | 1527.55 MB/s | 0 | 0 |
| AppendValueUnicode | 105.2 | 1416.46 MB/s | 0 | 0 |
| LevelTS_LogFmt | 49.9 | — | 0 | 0 |
| LevelTS_Regex | 12869.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 72.5 | — | 0 | 0 |
| ParseTime_Custom | 307.2 | — | 164 | 4 |
| ParseTime_Unix | 77.6 | — | 0 | 0 |
