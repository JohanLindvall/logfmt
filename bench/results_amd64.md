# logfmt parser comparison

- generated 2026-09-01T21:46:05Z
- go version go1.27.0 linux/amd64
- cpu: INTEL(R) XEON(R) PLATINUM 8573C (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 247 | 5674.11 MB/s | 0 | 0 | 9.1× |
| kr/logfmt | 1320 | 1060.99 MB/s | 80 | 1 | 1.7× |
| Grafana Loki | 1338 | 1046.40 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2233 | 627.09 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 54 | 2486.46 MB/s | 0 | 0 | 19.9× |
| kr/logfmt | 119 | 1135.84 MB/s | 0 | 0 | 9.1× |
| Grafana Loki | 136 | 989.83 MB/s | 0 | 0 | 7.9× |
| go-logfmt | 1078 | 125.20 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 167 | 890.25 MB/s | 0 | 0 | 7.5× |
| kr/logfmt | 286 | 520.71 MB/s | 112 | 3 | 4.4× |
| Grafana Loki | 293 | 507.88 MB/s | 112 | 3 | 4.3× |
| go-logfmt | 1253 | 118.89 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 63 | — | 0 | 0 | 18.3× |
| Grafana Loki | 267 | — | 80 | 1 | 4.3× |
| go-logfmt | 1157 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1446 | — | 152 | 4 | 0.8× |
