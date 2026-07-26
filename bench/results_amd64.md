# logfmt parser comparison

- generated 2026-07-26T11:53:41Z
- go version go1.26.5 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 444 | 3156.50 MB/s | 0 | 0 | 6.3× |
| kr/logfmt | 1473 | 950.73 MB/s | 80 | 1 | 1.9× |
| Grafana Loki | 1967 | 711.73 MB/s | 80 | 1 | 1.4× |
| go-logfmt | 2779 | 503.71 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 95 | 1419.66 MB/s | 0 | 0 | 10.2× |
| kr/logfmt | 125 | 1081.25 MB/s | 0 | 0 | 7.8× |
| Grafana Loki | 165 | 820.27 MB/s | 0 | 0 | 5.9× |
| go-logfmt | 974 | 138.64 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 228 | 652.85 MB/s | 0 | 0 | 5.2× |
| kr/logfmt | 320 | 465.06 MB/s | 112 | 3 | 3.7× |
| Grafana Loki | 371 | 401.54 MB/s | 112 | 3 | 3.2× |
| go-logfmt | 1178 | 126.46 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 96 | — | 0 | 0 | 11.8× |
| Grafana Loki | 349 | — | 80 | 1 | 3.3× |
| go-logfmt | 1135 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1560 | — | 152 | 4 | 0.7× |
