# logfmt microbenchmarks

- generated 2026-09-09T10:19:56Z
- go version go1.27.1 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 328.6 | — | 0 | 0 |
| GetMany_TimestampLevel | 79.6 | — | 0 | 0 |
| Unescape | 27.6 | — | 0 | 0 |
| IterateEscaped/esc=0 | 27.9 | 36950.21 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 83.5 | 12340.37 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 280.3 | 3674.08 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 505.1 | 2039.02 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 1343.0 | 767.05 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 32.3 | 31663.23 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 149.9 | 6829.82 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 368.4 | 2779.21 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 893.3 | 1146.27 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 1238.0 | 827.01 MB/s | 0 | 0 |
| IterateJSONMsg | 144.4 | 1793.70 MB/s | 0 | 0 |
| UnescapeJSONMsg | 214.0 | 925.19 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 575099.0 | 869.42 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 360.6 | 2856.15 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 278.9 | 3692.77 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 256.5 | 4015.99 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 195.0 | 5281.39 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 143.7 | 7168.36 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 83.7 | 12305.20 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 55.3 | 18630.37 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 569.3 | 1798.74 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 367.2 | 2788.74 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 420.4 | 2435.99 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 356.9 | 2869.37 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 265.1 | 3862.02 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 148.3 | 6906.96 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 82.8 | 12361.79 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 79.2 | 1679.61 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 79.3 | 1978.88 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 145.1 | 1302.53 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 147.1 | 1937.54 MB/s | 0 | 0 |
| UnescapeUnicode | 99.9 | 1220.88 MB/s | 0 | 0 |
| AppendValueUnicode | 134.5 | 1107.74 MB/s | 0 | 0 |
| LevelTS_LogFmt | 64.8 | — | 0 | 0 |
| LevelTS_Regex | 15384.0 | — | 1077 | 4 |
| GetManyScale/keys=000 | 3.1 | — | 0 | 0 |
| GetManyScale/keys=001 | 22.8 | — | 0 | 0 |
| GetManyScale/keys=002 | 39.5 | — | 0 | 0 |
| GetManyScale/keys=008 | 162.0 | — | 0 | 0 |
| GetManyScale/keys=016 | 369.4 | — | 0 | 0 |
| GetManyScale/keys=032 | 674.8 | — | 0 | 0 |
| GetManyScale/keys=064 | 1334.0 | — | 0 | 0 |
| GetManyScale/keys=128 | 2690.0 | — | 0 | 0 |
| UnescapeBuffer/json/reuse=false | 271.7 | — | 288 | 1 |
| UnescapeBuffer/json/reuse=true | 219.2 | — | 0 | 0 |
| UnescapeBuffer/dense/reuse=false | 1119.0 | — | 1024 | 1 |
| UnescapeBuffer/dense/reuse=true | 973.5 | — | 0 | 0 |
| UnescapeBuffer/unicode/reuse=false | 130.3 | — | 128 | 1 |
| UnescapeBuffer/unicode/reuse=true | 98.7 | — | 0 | 0 |
| UnescapeBuffer/surrogates/reuse=false | 261.3 | — | 192 | 1 |
| UnescapeBuffer/surrogates/reuse=true | 226.9 | — | 0 | 0 |
| UnescapeBuffer/malformed/reuse=false | 346.6 | — | 160 | 1 |
| UnescapeBuffer/malformed/reuse=true | 312.5 | — | 0 | 0 |
| UnescapeBuffer/plain/reuse=false | 193.2 | — | 1024 | 1 |
| UnescapeBuffer/plain/reuse=true | 31.2 | — | 0 | 0 |
| GetManyShapes/keys=016/ordered | 361.6 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse | 822.3 | — | 0 | 0 |
| GetManyShapes/keys=016/missing | 159.8 | — | 0 | 0 |
| GetManyShapes/keys=016/missing-same-length | 1277.0 | — | 0 | 0 |
| GetManyShapes/keys=016/collisions | 372.1 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse-collisions | 831.8 | — | 0 | 0 |
| GetManyShapes/keys=016/short-record | 42.2 | — | 0 | 0 |
| GetManyShapes/keys=064/ordered | 1325.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse | 1465.0 | — | 0 | 0 |
| GetManyShapes/keys=064/missing | 673.1 | — | 0 | 0 |
| GetManyShapes/keys=064/missing-same-length | 1291.0 | — | 0 | 0 |
| GetManyShapes/keys=064/collisions | 1389.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse-collisions | 2665.0 | — | 0 | 0 |
| GetManyShapes/keys=064/short-record | 114.6 | — | 0 | 0 |
| GetManyShapes/keys=256/ordered | 5110.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse | 7897.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing | 2509.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing-same-length | 8229.0 | — | 0 | 0 |
| GetManyShapes/keys=256/collisions | 5271.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse-collisions | 21942.0 | — | 0 | 0 |
| GetManyShapes/keys=256/short-record | 353.0 | — | 0 | 0 |
| ParseTime_RFC3339 | 80.3 | — | 0 | 0 |
| ParseTime_Custom | 390.7 | — | 164 | 4 |
| ParseTime_Unix | 87.1 | — | 0 | 0 |
