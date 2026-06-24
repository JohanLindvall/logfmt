# logfmt parser comparison

- generated 2026-06-24T19:18:56Z
- go version go1.26.3 linux/amd64
- cpu: AMD Ryzen 7 8840HS w/ Radeon 780M Graphics (16 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 369 | 3795.44 MB/s | 0 | 0 | 7.9× |
| kr/logfmt | 1233 | 1135.55 MB/s | 80 | 1 | 2.4× |
| Grafana Loki | 1647 | 849.77 MB/s | 80 | 1 | 1.8× |
| go-logfmt | 2907 | 481.54 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 72 | 1862.92 MB/s | 0 | 0 | 11.5× |
| kr/logfmt | 118 | 1138.97 MB/s | 0 | 0 | 7.0× |
| Grafana Loki | 138 | 981.98 MB/s | 0 | 0 | 6.0× |
| go-logfmt | 831 | 162.40 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 167 | 892.32 MB/s | 0 | 0 | 6.6× |
| kr/logfmt | 286 | 521.75 MB/s | 112 | 3 | 3.8× |
| Grafana Loki | 315 | 473.12 MB/s | 112 | 3 | 3.5× |
| go-logfmt | 1097 | 135.83 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 71 | — | 0 | 0 | 40.5× |
| kr/logfmt | 1358 | — | 152 | 4 | 2.1× |
| Grafana Loki | 1700 | — | 80 | 1 | 1.7× |
| go-logfmt | 2891 | — | 4224 | 3 | 1.0× |
