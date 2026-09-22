# logfmt microbenchmarks

- generated 2026-09-22T20:31:07Z
- go version go1.27.1 linux/amd64
- cpu: AMD EPYC 9V45 96-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 183.3 | — | 0 | 0 |
| GetMany_TimestampLevel | 41.8 | — | 0 | 0 |
| Unescape | 14.0 | — | 0 | 0 |
| IterateEscaped/esc=0 | 16.1 | 63925.55 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 53.6 | 19197.69 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 143.6 | 7174.19 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 192.7 | 5346.11 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 471.7 | 2183.81 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 18.6 | 55133.21 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 78.2 | 13099.07 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 194.0 | 5278.27 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 450.4 | 2273.50 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 490.8 | 2086.29 MB/s | 0 | 0 |
| IterateJSONMsg | 62.3 | 4154.76 MB/s | 0 | 0 |
| UnescapeJSONMsg | 109.0 | 1816.07 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 331363.0 | 1508.92 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 166.2 | 6197.96 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 146.3 | 7039.64 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 138.2 | 7452.84 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 129.3 | 7967.91 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 92.1 | 11178.86 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 54.5 | 18885.43 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 36.2 | 28418.84 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 270.8 | 3780.77 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 189.4 | 5405.50 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 243.5 | 4205.52 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 209.0 | 4898.86 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 155.8 | 6571.16 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 76.0 | 13473.94 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 44.9 | 22788.35 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 33.8 | 3936.85 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 34.5 | 4547.50 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 92.5 | 2043.03 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 92.9 | 3068.97 MB/s | 0 | 0 |
| UnescapeUnicode | 51.0 | 2391.71 MB/s | 0 | 0 |
| AppendValueUnicode | 66.5 | 2241.98 MB/s | 0 | 0 |
| Get/level | 35.6 | — | 0 | 0 |
| Get/session_attr_client_locale | 90.3 | — | 0 | 0 |
| Get/trace_id | 208.8 | — | 0 | 0 |
| IterateFieldShape/unquoted | 23331.0 | 1371.57 MB/s | 0 | 0 |
| IterateFieldShape/second-word | 15228.0 | 3677.50 MB/s | 0 | 0 |
| IterateFieldShape/quoted | 73072.0 | 656.89 MB/s | 0 | 0 |
| IterateFieldShape/bare | 22889.0 | 1048.53 MB/s | 0 | 0 |
| LevelTS_LogFmt | 33.9 | — | 0 | 0 |
| LevelTS_Regex | 8177.0 | — | 1076 | 4 |
| GetManyScale/keys=000 | 1.6 | — | 0 | 0 |
| GetManyScale/keys=001 | 11.0 | — | 0 | 0 |
| GetManyScale/keys=002 | 19.7 | — | 0 | 0 |
| GetManyScale/keys=008 | 81.5 | — | 0 | 0 |
| GetManyScale/keys=016 | 188.8 | — | 0 | 0 |
| GetManyScale/keys=032 | 354.5 | — | 0 | 0 |
| GetManyScale/keys=064 | 698.4 | — | 0 | 0 |
| GetManyScale/keys=128 | 1410.0 | — | 0 | 0 |
| UnescapeBuffer/json/reuse=false | 152.0 | — | 288 | 1 |
| UnescapeBuffer/json/reuse=true | 116.3 | — | 0 | 0 |
| UnescapeBuffer/dense/reuse=false | 533.6 | — | 1024 | 1 |
| UnescapeBuffer/dense/reuse=true | 403.0 | — | 0 | 0 |
| UnescapeBuffer/unicode/reuse=false | 69.5 | — | 128 | 1 |
| UnescapeBuffer/unicode/reuse=true | 49.4 | — | 0 | 0 |
| UnescapeBuffer/surrogates/reuse=false | 159.7 | — | 192 | 1 |
| UnescapeBuffer/surrogates/reuse=true | 134.0 | — | 0 | 0 |
| UnescapeBuffer/malformed/reuse=false | 191.5 | — | 160 | 1 |
| UnescapeBuffer/malformed/reuse=true | 173.1 | — | 0 | 0 |
| UnescapeBuffer/plain/reuse=false | 125.4 | — | 1024 | 1 |
| UnescapeBuffer/plain/reuse=true | 19.5 | — | 0 | 0 |
| GetManyShapes/keys=016/ordered | 183.9 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse | 449.4 | — | 0 | 0 |
| GetManyShapes/keys=016/missing | 80.7 | — | 0 | 0 |
| GetManyShapes/keys=016/missing-same-length | 647.7 | — | 0 | 0 |
| GetManyShapes/keys=016/collisions | 188.0 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse-collisions | 439.2 | — | 0 | 0 |
| GetManyShapes/keys=016/short-record | 21.5 | — | 0 | 0 |
| GetManyShapes/keys=064/ordered | 668.2 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse | 759.3 | — | 0 | 0 |
| GetManyShapes/keys=064/missing | 336.3 | — | 0 | 0 |
| GetManyShapes/keys=064/missing-same-length | 657.9 | — | 0 | 0 |
| GetManyShapes/keys=064/collisions | 694.8 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse-collisions | 1692.0 | — | 0 | 0 |
| GetManyShapes/keys=064/short-record | 56.7 | — | 0 | 0 |
| GetManyShapes/keys=256/ordered | 2661.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse | 4783.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing | 1299.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing-same-length | 4722.0 | — | 0 | 0 |
| GetManyShapes/keys=256/collisions | 2800.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse-collisions | 14826.0 | — | 0 | 0 |
| GetManyShapes/keys=256/short-record | 222.6 | — | 0 | 0 |
| ParseTime_RFC3339 | 37.7 | — | 0 | 0 |
| ParseTime_Custom | 209.7 | — | 164 | 4 |
| ParseTime_Unix | 44.2 | — | 0 | 0 |
