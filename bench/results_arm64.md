# logfmt parser comparison

- generated 2026-08-22T22:53:02Z
- go version go1.27.0 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 331 | 4226.06 MB/s | 0 | 0 | 7.5× |
| kr/logfmt | 1275 | 1097.63 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1464 | 956.25 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2492 | 561.84 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 67 | 2005.45 MB/s | 0 | 0 | 16.0× |
| kr/logfmt | 114 | 1180.46 MB/s | 0 | 0 | 9.4× |
| Grafana Loki | 144 | 937.74 MB/s | 0 | 0 | 7.5× |
| go-logfmt | 1079 | 125.07 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 200 | 744.82 MB/s | 0 | 0 | 6.6× |
| kr/logfmt | 313 | 476.37 MB/s | 112 | 3 | 4.2× |
| Grafana Loki | 338 | 440.54 MB/s | 112 | 3 | 3.9× |
| go-logfmt | 1320 | 112.86 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 74 | — | 0 | 0 | 17.0× |
| Grafana Loki | 303 | — | 80 | 1 | 4.2× |
| go-logfmt | 1258 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1366 | — | 152 | 4 | 0.9× |
