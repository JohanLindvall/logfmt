# logfmt parser comparison

- generated 2026-08-22T22:53:05Z
- go version go1.27.0 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 347 | 4031.14 MB/s | 0 | 0 | 6.9× |
| kr/logfmt | 1469 | 953.09 MB/s | 80 | 1 | 1.6× |
| Grafana Loki | 1531 | 914.53 MB/s | 80 | 1 | 1.6× |
| go-logfmt | 2390 | 585.74 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 76 | 1775.06 MB/s | 0 | 0 | 12.7× |
| kr/logfmt | 125 | 1079.40 MB/s | 0 | 0 | 7.7× |
| Grafana Loki | 155 | 871.59 MB/s | 0 | 0 | 6.2× |
| go-logfmt | 967 | 139.58 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 221 | 675.29 MB/s | 0 | 0 | 5.4× |
| kr/logfmt | 310 | 480.88 MB/s | 112 | 3 | 3.9× |
| Grafana Loki | 321 | 464.64 MB/s | 112 | 3 | 3.7× |
| go-logfmt | 1202 | 123.94 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 86 | — | 0 | 0 | 12.6× |
| Grafana Loki | 274 | — | 80 | 1 | 4.0× |
| go-logfmt | 1082 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1551 | — | 152 | 4 | 0.7× |
