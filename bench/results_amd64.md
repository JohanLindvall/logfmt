# logfmt parser comparison

- generated 2026-09-09T07:33:48Z
- go version go1.27.1 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 321 | 4360.23 MB/s | 0 | 0 | 7.5× |
| kr/logfmt | 1462 | 957.44 MB/s | 80 | 1 | 1.6× |
| Grafana Loki | 1553 | 901.36 MB/s | 80 | 1 | 1.5× |
| go-logfmt | 2394 | 584.72 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 72 | 1866.56 MB/s | 0 | 0 | 14.2× |
| kr/logfmt | 125 | 1079.31 MB/s | 0 | 0 | 8.2× |
| Grafana Loki | 166 | 813.57 MB/s | 0 | 0 | 6.2× |
| go-logfmt | 1025 | 131.70 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 213 | 700.21 MB/s | 0 | 0 | 5.8× |
| kr/logfmt | 310 | 480.86 MB/s | 112 | 3 | 3.9× |
| Grafana Loki | 326 | 457.12 MB/s | 112 | 3 | 3.8× |
| go-logfmt | 1224 | 121.74 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 78 | — | 0 | 0 | 14.0× |
| Grafana Loki | 275 | — | 80 | 1 | 4.0× |
| go-logfmt | 1086 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1524 | — | 152 | 4 | 0.7× |
