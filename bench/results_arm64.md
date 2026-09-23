# logfmt parser comparison

- generated 2026-09-23T09:09:29Z
- go version go1.27.1 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 296 | 4732.45 MB/s | 0 | 0 | 8.6× |
| kr/logfmt | 1278 | 1095.72 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1491 | 938.90 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2530 | 553.29 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 64 | 2107.89 MB/s | 0 | 0 | 17.4× |
| kr/logfmt | 115 | 1177.33 MB/s | 0 | 0 | 9.7× |
| Grafana Loki | 144 | 938.72 MB/s | 0 | 0 | 7.8× |
| go-logfmt | 1116 | 120.93 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 178 | 836.59 MB/s | 0 | 0 | 7.6× |
| kr/logfmt | 313 | 475.59 MB/s | 112 | 3 | 4.3× |
| Grafana Loki | 337 | 442.62 MB/s | 112 | 3 | 4.0× |
| go-logfmt | 1352 | 110.19 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 64 | — | 0 | 0 | 20.3× |
| Grafana Loki | 322 | — | 80 | 1 | 4.0× |
| go-logfmt | 1303 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1372 | — | 152 | 4 | 0.9× |
