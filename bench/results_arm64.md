# logfmt parser comparison

- generated 2026-06-24T19:53:26Z
- go version go1.26.3 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 437 | 3206.46 MB/s | 0 | 0 | 5.6× |
| kr/logfmt | 1245 | 1124.35 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1482 | 944.61 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2460 | 569.11 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 88 | 1534.18 MB/s | 0 | 0 | 12.0× |
| kr/logfmt | 113 | 1196.13 MB/s | 0 | 0 | 9.4× |
| Grafana Loki | 142 | 948.31 MB/s | 0 | 0 | 7.4× |
| go-logfmt | 1058 | 127.55 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 220 | 678.88 MB/s | 0 | 0 | 5.9× |
| kr/logfmt | 327 | 456.02 MB/s | 112 | 3 | 4.0× |
| Grafana Loki | 374 | 398.82 MB/s | 112 | 3 | 3.5× |
| go-logfmt | 1304 | 114.31 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 87 | — | 0 | 0 | 27.6× |
| kr/logfmt | 1396 | — | 152 | 4 | 1.7× |
| Grafana Loki | 1492 | — | 80 | 1 | 1.6× |
| go-logfmt | 2411 | — | 4224 | 3 | 1.0× |
