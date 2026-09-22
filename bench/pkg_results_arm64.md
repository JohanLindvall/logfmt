# logfmt microbenchmarks

- generated 2026-09-22T20:31:09Z
- go version go1.27.1 linux/arm64
- cpu: unknown (4 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 296.0 | — | 0 | 0 |
| GetMany_TimestampLevel | 65.3 | — | 0 | 0 |
| Unescape | 24.8 | — | 0 | 0 |
| IterateEscaped/esc=0 | 35.4 | 29121.47 MB/s | 0 | 0 |
| IterateEscaped/esc=8 | 105.5 | 9766.16 MB/s | 0 | 0 |
| IterateEscaped/esc=32 | 195.3 | 5273.14 MB/s | 0 | 0 |
| IterateEscaped/esc=128 | 293.3 | 3512.11 MB/s | 0 | 0 |
| IterateEscaped/esc=500 | 721.2 | 1428.16 MB/s | 0 | 0 |
| UnescapeEscaped/esc=0 | 48.0 | 21357.09 MB/s | 0 | 0 |
| UnescapeEscaped/esc=8 | 129.2 | 7925.24 MB/s | 0 | 0 |
| UnescapeEscaped/esc=32 | 268.6 | 3812.20 MB/s | 0 | 0 |
| UnescapeEscaped/esc=128 | 740.6 | 1382.62 MB/s | 0 | 0 |
| UnescapeEscaped/esc=500 | 850.4 | 1204.13 MB/s | 0 | 0 |
| IterateJSONMsg | 96.0 | 2698.22 MB/s | 0 | 0 |
| UnescapeJSONMsg | 187.4 | 1056.64 MB/s | 0 | 0 |
| DecodeKeyval_Custom | 594260.0 | 841.38 MB/s | 0 | 0 |
| IterateEscapedGap/gap=016 | 227.4 | 4530.24 MB/s | 0 | 0 |
| IterateEscapedGap/gap=032 | 195.3 | 5275.09 MB/s | 0 | 0 |
| IterateEscapedGap/gap=040 | 186.5 | 5521.82 MB/s | 0 | 0 |
| IterateEscapedGap/gap=048 | 227.3 | 4531.68 MB/s | 0 | 0 |
| IterateEscapedGap/gap=064 | 176.1 | 5849.47 MB/s | 0 | 0 |
| IterateEscapedGap/gap=128 | 105.5 | 9766.81 MB/s | 0 | 0 |
| IterateEscapedGap/gap=256 | 69.0 | 14937.76 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=016 | 419.0 | 2443.88 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=032 | 273.5 | 3743.55 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=040 | 353.2 | 2899.37 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=048 | 294.7 | 3474.40 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=064 | 220.4 | 4645.60 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=128 | 128.7 | 7959.47 MB/s | 0 | 0 |
| UnescapeEscapedGap/gap=256 | 80.3 | 12749.76 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=008 | 54.2 | 2452.07 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=032 | 54.3 | 2892.63 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=064 | 165.6 | 1141.64 MB/s | 0 | 0 |
| IteratePrefixJSON/prefix=160 | 167.8 | 1698.95 MB/s | 0 | 0 |
| UnescapeUnicode | 71.9 | 1696.28 MB/s | 0 | 0 |
| AppendValueUnicode | 104.2 | 1430.06 MB/s | 0 | 0 |
| Get/level | 59.1 | — | 0 | 0 |
| Get/session_attr_client_locale | 142.1 | — | 0 | 0 |
| Get/trace_id | 308.4 | — | 0 | 0 |
| IterateFieldShape/unquoted | 44510.0 | 718.94 MB/s | 0 | 0 |
| IterateFieldShape/second-word | 24678.0 | 2269.19 MB/s | 0 | 0 |
| IterateFieldShape/quoted | 127202.0 | 377.35 MB/s | 0 | 0 |
| IterateFieldShape/bare | 42956.0 | 558.71 MB/s | 0 | 0 |
| LevelTS_LogFmt | 56.6 | — | 0 | 0 |
| LevelTS_Regex | 13771.0 | — | 1076 | 4 |
| GetManyScale/keys=000 | 3.3 | — | 0 | 0 |
| GetManyScale/keys=001 | 17.2 | — | 0 | 0 |
| GetManyScale/keys=002 | 29.7 | — | 0 | 0 |
| GetManyScale/keys=008 | 113.3 | — | 0 | 0 |
| GetManyScale/keys=016 | 253.1 | — | 0 | 0 |
| GetManyScale/keys=032 | 544.9 | — | 0 | 0 |
| GetManyScale/keys=064 | 1062.0 | — | 0 | 0 |
| GetManyScale/keys=128 | 2098.0 | — | 0 | 0 |
| UnescapeBuffer/json/reuse=false | 295.7 | — | 288 | 1 |
| UnescapeBuffer/json/reuse=true | 192.7 | — | 0 | 0 |
| UnescapeBuffer/dense/reuse=false | 985.9 | — | 1024 | 1 |
| UnescapeBuffer/dense/reuse=true | 766.7 | — | 0 | 0 |
| UnescapeBuffer/unicode/reuse=false | 122.0 | — | 128 | 1 |
| UnescapeBuffer/unicode/reuse=true | 72.5 | — | 0 | 0 |
| UnescapeBuffer/surrogates/reuse=false | 241.6 | — | 192 | 1 |
| UnescapeBuffer/surrogates/reuse=true | 178.9 | — | 0 | 0 |
| UnescapeBuffer/malformed/reuse=false | 313.0 | — | 160 | 1 |
| UnescapeBuffer/malformed/reuse=true | 233.2 | — | 0 | 0 |
| UnescapeBuffer/plain/reuse=false | 270.6 | — | 1024 | 1 |
| UnescapeBuffer/plain/reuse=true | 48.2 | — | 0 | 0 |
| GetManyShapes/keys=016/ordered | 246.1 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse | 579.1 | — | 0 | 0 |
| GetManyShapes/keys=016/missing | 132.7 | — | 0 | 0 |
| GetManyShapes/keys=016/missing-same-length | 888.4 | — | 0 | 0 |
| GetManyShapes/keys=016/collisions | 256.3 | — | 0 | 0 |
| GetManyShapes/keys=016/reverse-collisions | 584.2 | — | 0 | 0 |
| GetManyShapes/keys=016/short-record | 33.6 | — | 0 | 0 |
| GetManyShapes/keys=064/ordered | 1025.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse | 1109.0 | — | 0 | 0 |
| GetManyShapes/keys=064/missing | 514.4 | — | 0 | 0 |
| GetManyShapes/keys=064/missing-same-length | 965.1 | — | 0 | 0 |
| GetManyShapes/keys=064/collisions | 1065.0 | — | 0 | 0 |
| GetManyShapes/keys=064/reverse-collisions | 2355.0 | — | 0 | 0 |
| GetManyShapes/keys=064/short-record | 83.7 | — | 0 | 0 |
| GetManyShapes/keys=256/ordered | 4007.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse | 6530.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing | 1991.0 | — | 0 | 0 |
| GetManyShapes/keys=256/missing-same-length | 6591.0 | — | 0 | 0 |
| GetManyShapes/keys=256/collisions | 4175.0 | — | 0 | 0 |
| GetManyShapes/keys=256/reverse-collisions | 18450.0 | — | 0 | 0 |
| GetManyShapes/keys=256/short-record | 281.7 | — | 0 | 0 |
| ParseTime_RFC3339 | 68.7 | — | 0 | 0 |
| ParseTime_Custom | 382.1 | — | 164 | 4 |
| ParseTime_Unix | 71.2 | — | 0 | 0 |
