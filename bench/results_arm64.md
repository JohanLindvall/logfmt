# logfmt parser comparison

- generated 2026-09-09T07:33:43Z
- go version go1.27.1 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 300 | 4668.13 MB/s | 0 | 0 | 8.3× |
| kr/logfmt | 1276 | 1097.31 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1465 | 955.92 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2500 | 559.98 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 64 | 2103.00 MB/s | 0 | 0 | 16.8× |
| kr/logfmt | 114 | 1178.65 MB/s | 0 | 0 | 9.4× |
| Grafana Loki | 144 | 936.48 MB/s | 0 | 0 | 7.5× |
| go-logfmt | 1077 | 125.30 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 189 | 789.78 MB/s | 0 | 0 | 6.9× |
| kr/logfmt | 341 | 436.40 MB/s | 112 | 3 | 3.8× |
| Grafana Loki | 367 | 406.48 MB/s | 112 | 3 | 3.6× |
| go-logfmt | 1303 | 114.39 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 66 | — | 0 | 0 | 19.0× |
| Grafana Loki | 324 | — | 80 | 1 | 3.9× |
| go-logfmt | 1253 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1377 | — | 152 | 4 | 0.9× |
