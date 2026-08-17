# logfmt parser comparison

- generated 2026-08-17T16:46:21Z
- go version go1.26.6 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 376 | 3718.08 MB/s | 0 | 0 | 6.8× |
| kr/logfmt | 1253 | 1117.61 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1478 | 947.45 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2567 | 545.34 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 83 | 1625.44 MB/s | 0 | 0 | 13.6× |
| kr/logfmt | 114 | 1184.31 MB/s | 0 | 0 | 9.9× |
| Grafana Loki | 143 | 943.45 MB/s | 0 | 0 | 7.9× |
| go-logfmt | 1133 | 119.11 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 194 | 768.48 MB/s | 0 | 0 | 7.1× |
| kr/logfmt | 306 | 487.19 MB/s | 112 | 3 | 4.5× |
| Grafana Loki | 346 | 430.25 MB/s | 112 | 3 | 4.0× |
| go-logfmt | 1384 | 107.66 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 76 | — | 0 | 0 | 17.3× |
| Grafana Loki | 305 | — | 80 | 1 | 4.3× |
| go-logfmt | 1304 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1393 | — | 152 | 4 | 0.9× |
