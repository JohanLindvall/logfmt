# logfmt microbenchmarks

- generated 2026-09-23T09:05:23Z
- go version go1.27.1 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 298.4 | — | 0 | 0 |
| GetMany_TimestampLevel | 64.3 | — | 0 | 0 |
| Unescape | 25.8 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.5 | 28987.55 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 105.5 | 9764.45 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 175.8 | 5857.66 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 268.4 | 3838.26 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 687.0 | 1499.30 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 47.9 | 21388.80 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 119.8 | 8548.89 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 248.9 | 4113.30 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 601.1 | 1703.50 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 805.4 | 1271.44 MB/s | 0 | 0 |
| IterateJSONMsg | 92.2 | 2810.00 MB/s | 0 | 0 |
| UnescapeJSONMsg | 155.7 | 1271.98 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 595242.0 | 839.99 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 205.9 | 5003.08 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 175.9 | 5856.42 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 167.2 | 6160.85 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 161.9 | 6361.04 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 155.2 | 6638.56 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 105.0 | 9805.37 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 70.0 | 14716.01 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 371.9 | 2753.21 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 247.8 | 4132.03 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 236.5 | 4329.96 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 221.7 | 4619.87 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 191.1 | 5358.58 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 119.4 | 8576.03 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 81.5 | 12569.61 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 51.0 | 2606.50 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 51.2 | 3064.90 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 58.9 | 3211.38 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 61.3 | 4647.45 MB/s | 0 | 0 |
| IterateEscapedAlternating/gaps=8-80 | 249.8 | 4250.78 MB/s | 0 | 0 |
| IterateEscapedAlternating/gaps=40-120 | 167.9 | 6706.53 MB/s | 0 | 0 |
| UnescapeUnicode | 71.8 | 1698.33 MB/s | 0 | 0 |
| AppendValueUnicode | 102.6 | 1451.85 MB/s | 0 | 0 |
| Get/level | 59.4 | — | 0 | 0 |
| Get/session_attr_client_locale | 142.0 | — | 0 | 0 |
| Get/trace_id | 309.0 | — | 0 | 0 |
| IterateFieldShape/unquoted | 44487.0 | 719.32 MB/s | 0 | 0 |
| IterateFieldShape/second-word | 25212.0 | 2221.17 MB/s | 0 | 0 |
| IterateFieldShape/quoted | 126898.0 | 378.26 MB/s | 0 | 0 |
| IterateFieldShape/bare | 42955.0 | 558.72 MB/s | 0 | 0 |
| LevelTS_LogFmt | 57.3 | — | 0 | 0 |
| LevelTS_Regex | 13761.0 | — | 1077 | 4 |
| GetManyScale/keys=000 | 3.2 | — | 0 | 0 |
| GetManyScale/keys=001 | 17.3 | — | 0 | 0 |
| GetManyScale/keys=002 | 29.9 | — | 0 | 0 |
| GetManyScale/keys=008 | 114.8 | — | 0 | 0 |
| GetManyScale/keys=016 | 257.6 | — | 0 | 0 |
| GetManyScale/keys=032 | 544.1 | — | 0 | 0 |
| GetManyScale/keys=064 | 1060.0 | — | 0 | 0 |
| GetManyScale/keys=128 | 2097.0 | — | 0 | 0 |
| UnescapeBuffer/json/reuse=false | 262.0 | — | 288 | 1 |
| UnescapeBuffer/json/reuse=true | 160.4 | — | 0 | 0 |
| UnescapeBuffer/dense/reuse=false | 940.6 | — | 1024 | 1 |
| UnescapeBuffer/dense/reuse=true | 709.0 | — | 0 | 0 |
| UnescapeBuffer/unicode/reuse=false | 124.0 | — | 128 | 1 |
| UnescapeBuffer/unicode/reuse=true | 72.6 | — | 0 | 0 |
| UnescapeBuffer/surrogates/reuse=false | 245.9 | — | 192 | 1 |
| UnescapeBuffer/surrogates/reuse=true | 173.3 | — | 0 | 0 |
| UnescapeBuffer/malformed/reuse=false | 315.2 | — | 160 | 1 |
| UnescapeBuffer/malformed/reuse=true | 234.8 | — | 0 | 0 |
| UnescapeBuffer/plain/reuse=false | 277.3 | — | 1024 | 1 |
| UnescapeBuffer/plain/reuse=true | 48.2 | — | 0 | 0 |
| GetManyShapes/keys=016/ordered | 250.1 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse | 569.3 | — | 0 | 0 |
| GetManyShapes/keys=016/missing | 132.5 | — | 0 | 0 |
| GetManyShapes/keys=016/missing-same-length | 875.1 | — | 0 | 0 |
| GetManyShapes/keys=016/collisions | 259.5 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse-collisions | 573.9 | — | 0 | 0 |
| GetManyShapes/keys=016/short-record | 33.8 | — | 0 | 0 |
| GetManyShapes/keys=064/ordered | 1022.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse | 1105.0 | — | 0 | 0 |
| GetManyShapes/keys=064/missing | 522.9 | — | 0 | 0 |
| GetManyShapes/keys=064/missing-same-length | 964.7 | — | 0 | 0 |
| GetManyShapes/keys=064/collisions | 1064.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse-collisions | 2302.0 | — | 0 | 0 |
| GetManyShapes/keys=064/short-record | 83.8 | — | 0 | 0 |
| GetManyShapes/keys=256/ordered | 4006.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse | 6517.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing | 2015.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing-same-length | 6567.0 | — | 0 | 0 |
| GetManyShapes/keys=256/collisions | 4166.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse-collisions | 18445.0 | — | 0 | 0 |
| GetManyShapes/keys=256/short-record | 281.9 | — | 0 | 0 |
| ParseTime_RFC3339 | 69.2 | — | 0 | 0 |
| ParseTime_Custom | 363.5 | — | 164 | 4 |
| ParseTime_Unix | 68.9 | — | 0 | 0 |
