# logfmt parser comparison

- generated 2026-07-26T11:53:39Z
- go version go1.26.5 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 425 | 3295.76 MB/s | 0 | 0 | 5.7× |
| kr/logfmt | 1243 | 1126.70 MB/s | 80 | 1 | 1.9× |
| Grafana Loki | 1483 | 944.22 MB/s | 80 | 1 | 1.6× |
| go-logfmt | 2419 | 578.67 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 88 | 1533.06 MB/s | 0 | 0 | 11.5× |
| kr/logfmt | 113 | 1196.58 MB/s | 0 | 0 | 9.0× |
| Grafana Loki | 143 | 941.50 MB/s | 0 | 0 | 7.1× |
| go-logfmt | 1016 | 132.86 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 218 | 684.03 MB/s | 0 | 0 | 5.8× |
| kr/logfmt | 329 | 452.96 MB/s | 112 | 3 | 3.8× |
| Grafana Loki | 374 | 398.38 MB/s | 112 | 3 | 3.4× |
| go-logfmt | 1258 | 118.46 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 85 | — | 0 | 0 | 14.1× |
| Grafana Loki | 328 | — | 80 | 1 | 3.7× |
| go-logfmt | 1201 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1404 | — | 152 | 4 | 0.9× |
