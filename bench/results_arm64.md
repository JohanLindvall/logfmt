# logfmt parser comparison

- generated 2026-09-01T21:46:00Z
- go version go1.27.0 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 302 | 4636.89 MB/s | 0 | 0 | 8.3× |
| kr/logfmt | 1274 | 1099.16 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1488 | 940.69 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2506 | 558.57 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 64 | 2099.64 MB/s | 0 | 0 | 16.8× |
| kr/logfmt | 115 | 1171.41 MB/s | 0 | 0 | 9.4× |
| Grafana Loki | 144 | 935.79 MB/s | 0 | 0 | 7.5× |
| go-logfmt | 1078 | 125.24 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 194 | 767.65 MB/s | 0 | 0 | 6.7× |
| kr/logfmt | 344 | 432.60 MB/s | 112 | 3 | 3.8× |
| Grafana Loki | 363 | 410.80 MB/s | 112 | 3 | 3.6× |
| go-logfmt | 1310 | 113.72 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 71 | — | 0 | 0 | 17.6× |
| Grafana Loki | 337 | — | 80 | 1 | 3.7× |
| go-logfmt | 1246 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1380 | — | 152 | 4 | 0.9× |
