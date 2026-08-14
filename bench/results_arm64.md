# logfmt parser comparison

- generated 2026-08-14T10:06:27Z
- go version go1.26.5 linux/arm64
- cpu: unknown (2 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 380 | 3684.30 MB/s | 0 | 0 | 6.6× |
| kr/logfmt | 1252 | 1118.06 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1503 | 931.63 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2512 | 557.33 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 85 | 1593.96 MB/s | 0 | 0 | 12.9× |
| kr/logfmt | 114 | 1183.49 MB/s | 0 | 0 | 9.5× |
| Grafana Loki | 142 | 953.22 MB/s | 0 | 0 | 7.7× |
| go-logfmt | 1089 | 123.92 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 214 | 695.88 MB/s | 0 | 0 | 6.2× |
| kr/logfmt | 304 | 490.18 MB/s | 112 | 3 | 4.4× |
| Grafana Loki | 344 | 432.89 MB/s | 112 | 3 | 3.9× |
| go-logfmt | 1331 | 111.91 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 76 | — | 0 | 0 | 16.4× |
| Grafana Loki | 325 | — | 80 | 1 | 3.8× |
| go-logfmt | 1248 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1388 | — | 152 | 4 | 0.9× |
