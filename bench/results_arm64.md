# logfmt parser comparison

- generated 2026-08-17T13:17:10Z
- go version go1.26.6 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 383 | 3654.86 MB/s | 0 | 0 | 6.5× |
| kr/logfmt | 1242 | 1127.14 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1509 | 927.46 MB/s | 80 | 1 | 1.6× |
| go-logfmt | 2488 | 562.63 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 83 | 1619.70 MB/s | 0 | 0 | 12.7× |
| kr/logfmt | 113 | 1193.51 MB/s | 0 | 0 | 9.4× |
| Grafana Loki | 152 | 885.94 MB/s | 0 | 0 | 6.9× |
| go-logfmt | 1058 | 127.61 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 193 | 772.97 MB/s | 0 | 0 | 6.8× |
| kr/logfmt | 311 | 479.63 MB/s | 112 | 3 | 4.2× |
| Grafana Loki | 348 | 428.35 MB/s | 112 | 3 | 3.8× |
| go-logfmt | 1306 | 114.05 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 80 | — | 0 | 0 | 15.3× |
| Grafana Loki | 323 | — | 80 | 1 | 3.8× |
| go-logfmt | 1227 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1384 | — | 152 | 4 | 0.9× |
