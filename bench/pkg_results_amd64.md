# logfmt microbenchmarks

- generated 2026-09-23T09:05:23Z
- go version go1.27.1 linux/amd64
- cpu: AMD EPYC 9V74 80-Core Processor (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 299.2 | — | 0 | 0 |
| GetMany_TimestampLevel | 72.4 | — | 0 | 0 |
| Unescape | 25.4 | — | 0 | 0 |
| IterateEscaped/esc=0 | 29.0 | 35485.66 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 83.8 | 12284.78 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 238.3 | 4322.62 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 343.0 | 3002.80 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 842.5 | 1222.49 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 31.5 | 32548.76 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 140.6 | 7285.64 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 320.6 | 3193.74 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 913.1 | 1121.47 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 1169.0 | 875.86 MB/s | 0 | 0 |
| IterateJSONMsg | 111.5 | 2322.20 MB/s | 0 | 0 |
| UnescapeJSONMsg | 194.5 | 1017.76 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 532236.0 | 939.43 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 285.4 | 3608.75 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 237.8 | 4330.94 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 227.8 | 4520.62 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 183.3 | 5617.75 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 135.1 | 7624.64 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 83.3 | 12358.16 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 56.2 | 18323.62 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 525.7 | 1947.71 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 323.0 | 3169.96 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 425.7 | 2405.41 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 354.7 | 2886.85 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 262.6 | 3899.83 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 139.6 | 7337.78 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 78.9 | 12984.52 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 58.2 | 2285.64 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 59.0 | 2661.04 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 144.3 | 1310.05 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 146.4 | 1946.70 MB/s | 0 | 0 |
| IterateEscapedAlternating/gaps=8-80 | 213.4 | 4975.80 MB/s | 0 | 0 |
| IterateEscapedAlternating/gaps=40-120 | 135.7 | 8298.59 MB/s | 0 | 0 |
| UnescapeUnicode | 87.9 | 1388.60 MB/s | 0 | 0 |
| AppendValueUnicode | 119.5 | 1247.21 MB/s | 0 | 0 |
| Get/level | 59.5 | — | 0 | 0 |
| Get/session_attr_client_locale | 153.2 | — | 0 | 0 |
| Get/trace_id | 332.1 | — | 0 | 0 |
| IterateFieldShape/unquoted | 42476.0 | 753.37 MB/s | 0 | 0 |
| IterateFieldShape/second-word | 21820.0 | 2566.47 MB/s | 0 | 0 |
| IterateFieldShape/quoted | 111362.0 | 431.03 MB/s | 0 | 0 |
| IterateFieldShape/bare | 36930.0 | 649.88 MB/s | 0 | 0 |
| LevelTS_LogFmt | 56.8 | — | 0 | 0 |
| LevelTS_Regex | 14481.0 | — | 1076 | 4 |
| GetManyScale/keys=000 | 2.8 | — | 0 | 0 |
| GetManyScale/keys=001 | 20.2 | — | 0 | 0 |
| GetManyScale/keys=002 | 36.7 | — | 0 | 0 |
| GetManyScale/keys=008 | 144.7 | — | 0 | 0 |
| GetManyScale/keys=016 | 348.2 | — | 0 | 0 |
| GetManyScale/keys=032 | 872.2 | — | 0 | 0 |
| GetManyScale/keys=064 | 1261.0 | — | 0 | 0 |
| GetManyScale/keys=128 | 2489.0 | — | 0 | 0 |
| UnescapeBuffer/json/reuse=false | 255.2 | — | 288 | 1 |
| UnescapeBuffer/json/reuse=true | 201.1 | — | 0 | 0 |
| UnescapeBuffer/dense/reuse=false | 1149.0 | — | 1024 | 1 |
| UnescapeBuffer/dense/reuse=true | 983.7 | — | 0 | 0 |
| UnescapeBuffer/unicode/reuse=false | 118.2 | — | 128 | 1 |
| UnescapeBuffer/unicode/reuse=true | 88.5 | — | 0 | 0 |
| UnescapeBuffer/surrogates/reuse=false | 273.2 | — | 192 | 1 |
| UnescapeBuffer/surrogates/reuse=true | 234.2 | — | 0 | 0 |
| UnescapeBuffer/malformed/reuse=false | 323.4 | — | 160 | 1 |
| UnescapeBuffer/malformed/reuse=true | 286.2 | — | 0 | 0 |
| UnescapeBuffer/plain/reuse=false | 198.5 | — | 1024 | 1 |
| UnescapeBuffer/plain/reuse=true | 31.4 | — | 0 | 0 |
| GetManyShapes/keys=016/ordered | 339.0 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse | 707.8 | — | 0 | 0 |
| GetManyShapes/keys=016/missing | 143.2 | — | 0 | 0 |
| GetManyShapes/keys=016/missing-same-length | 1289.0 | — | 0 | 0 |
| GetManyShapes/keys=016/collisions | 349.6 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse-collisions | 718.5 | — | 0 | 0 |
| GetManyShapes/keys=016/short-record | 39.5 | — | 0 | 0 |
| GetManyShapes/keys=064/ordered | 1224.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse | 1291.0 | — | 0 | 0 |
| GetManyShapes/keys=064/missing | 572.6 | — | 0 | 0 |
| GetManyShapes/keys=064/missing-same-length | 1092.0 | — | 0 | 0 |
| GetManyShapes/keys=064/collisions | 1252.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse-collisions | 2505.0 | — | 0 | 0 |
| GetManyShapes/keys=064/short-record | 100.1 | — | 0 | 0 |
| GetManyShapes/keys=256/ordered | 4770.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse | 6781.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing | 2225.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing-same-length | 7150.0 | — | 0 | 0 |
| GetManyShapes/keys=256/collisions | 4969.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse-collisions | 19905.0 | — | 0 | 0 |
| GetManyShapes/keys=256/short-record | 352.7 | — | 0 | 0 |
| ParseTime_RFC3339 | 79.1 | — | 0 | 0 |
| ParseTime_Custom | 346.5 | — | 164 | 4 |
| ParseTime_Unix | 86.7 | — | 0 | 0 |
