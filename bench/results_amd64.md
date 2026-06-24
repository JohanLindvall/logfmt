# logfmt parser comparison

- generated 2026-06-24T19:53:16Z
- go version go1.26.3 linux/amd64
- cpu: AMD EPYC 9V74 80-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 472 | 2967.86 MB/s | 0 | 0 | 6.0× |
| kr/logfmt | 1607 | 870.96 MB/s | 80 | 1 | 1.8× |
| Grafana Loki | 2209 | 633.79 MB/s | 80 | 1 | 1.3× |
| go-logfmt | 2843 | 492.44 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 93 | 1458.70 MB/s | 0 | 0 | 10.5× |
| kr/logfmt | 142 | 949.51 MB/s | 0 | 0 | 6.8× |
| Grafana Loki | 174 | 775.12 MB/s | 0 | 0 | 5.6× |
| go-logfmt | 970 | 139.19 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 212 | 702.16 MB/s | 0 | 0 | 5.5× |
| kr/logfmt | 356 | 418.74 MB/s | 112 | 3 | 3.3× |
| Grafana Loki | 371 | 401.31 MB/s | 112 | 3 | 3.1× |
| go-logfmt | 1166 | 127.80 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 90 | — | 0 | 0 | 31.3× |
| kr/logfmt | 1685 | — | 152 | 4 | 1.7× |
| Grafana Loki | 2119 | — | 80 | 1 | 1.3× |
| go-logfmt | 2813 | — | 4224 | 3 | 1.0× |
