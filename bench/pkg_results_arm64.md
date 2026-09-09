# logfmt microbenchmarks

- generated 2026-09-09T10:19:58Z
- go version go1.27.1 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 299.9 | — | 0 | 0 |
| GetMany_TimestampLevel | 65.9 | — | 0 | 0 |
| Unescape | 25.6 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.6 | 28920.11 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 106.0 | 9720.83 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 197.8 | 5207.57 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 288.1 | 3575.62 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 734.6 | 1402.09 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 48.0 | 21339.15 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 131.4 | 7792.41 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 275.3 | 3719.14 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 738.2 | 1387.23 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 1026.0 | 998.25 MB/s | 0 | 0 |
| IterateJSONMsg | 96.4 | 2686.75 MB/s | 0 | 0 |
| UnescapeJSONMsg | 187.6 | 1055.50 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 593634.0 | 842.27 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 228.0 | 4517.57 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 197.9 | 5205.78 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 187.9 | 5481.42 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 226.8 | 4542.15 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 178.0 | 5787.95 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 105.1 | 9799.86 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 69.5 | 14820.33 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 416.6 | 2458.23 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 275.8 | 3713.46 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 356.9 | 2869.33 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 300.0 | 3412.79 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 223.0 | 4592.89 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 131.5 | 7788.08 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 80.0 | 12798.91 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 53.9 | 2466.36 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 54.0 | 2908.34 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 165.0 | 1145.27 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 168.7 | 1689.06 MB/s | 0 | 0 |
| UnescapeUnicode | 75.5 | 1615.31 MB/s | 0 | 0 |
| AppendValueUnicode | 109.9 | 1355.56 MB/s | 0 | 0 |
| LevelTS_LogFmt | 58.7 | — | 0 | 0 |
| LevelTS_Regex | 13720.0 | — | 1076 | 4 |
| GetManyScale/keys=000 | 3.2 | — | 0 | 0 |
| GetManyScale/keys=001 | 17.5 | — | 0 | 0 |
| GetManyScale/keys=002 | 30.1 | — | 0 | 0 |
| GetManyScale/keys=008 | 116.4 | — | 0 | 0 |
| GetManyScale/keys=016 | 263.0 | — | 0 | 0 |
| GetManyScale/keys=032 | 538.6 | — | 0 | 0 |
| GetManyScale/keys=064 | 1062.0 | — | 0 | 0 |
| GetManyScale/keys=128 | 2071.0 | — | 0 | 0 |
| UnescapeBuffer/json/reuse=false | 289.7 | — | 288 | 1 |
| UnescapeBuffer/json/reuse=true | 193.1 | — | 0 | 0 |
| UnescapeBuffer/dense/reuse=false | 1019.0 | — | 1024 | 1 |
| UnescapeBuffer/dense/reuse=true | 794.3 | — | 0 | 0 |
| UnescapeBuffer/unicode/reuse=false | 122.1 | — | 128 | 1 |
| UnescapeBuffer/unicode/reuse=true | 76.0 | — | 0 | 0 |
| UnescapeBuffer/surrogates/reuse=false | 239.2 | — | 192 | 1 |
| UnescapeBuffer/surrogates/reuse=true | 179.1 | — | 0 | 0 |
| UnescapeBuffer/malformed/reuse=false | 319.7 | — | 160 | 1 |
| UnescapeBuffer/malformed/reuse=true | 249.9 | — | 0 | 0 |
| UnescapeBuffer/plain/reuse=false | 258.7 | — | 1024 | 1 |
| UnescapeBuffer/plain/reuse=true | 48.2 | — | 0 | 0 |
| GetManyShapes/keys=016/ordered | 254.7 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse | 588.5 | — | 0 | 0 |
| GetManyShapes/keys=016/missing | 131.8 | — | 0 | 0 |
| GetManyShapes/keys=016/missing-same-length | 916.8 | — | 0 | 0 |
| GetManyShapes/keys=016/collisions | 264.6 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse-collisions | 599.5 | — | 0 | 0 |
| GetManyShapes/keys=016/short-record | 34.1 | — | 0 | 0 |
| GetManyShapes/keys=064/ordered | 1014.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse | 1096.0 | — | 0 | 0 |
| GetManyShapes/keys=064/missing | 523.8 | — | 0 | 0 |
| GetManyShapes/keys=064/missing-same-length | 966.7 | — | 0 | 0 |
| GetManyShapes/keys=064/collisions | 1053.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse-collisions | 2296.0 | — | 0 | 0 |
| GetManyShapes/keys=064/short-record | 83.9 | — | 0 | 0 |
| GetManyShapes/keys=256/ordered | 3976.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse | 6459.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing | 2025.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing-same-length | 6595.0 | — | 0 | 0 |
| GetManyShapes/keys=256/collisions | 4131.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse-collisions | 18450.0 | — | 0 | 0 |
| GetManyShapes/keys=256/short-record | 282.0 | — | 0 | 0 |
| ParseTime_RFC3339 | 69.0 | — | 0 | 0 |
| ParseTime_Custom | 365.5 | — | 164 | 4 |
| ParseTime_Unix | 71.3 | — | 0 | 0 |
