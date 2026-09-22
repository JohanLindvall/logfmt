# logfmt parser comparison

- generated 2026-09-22T20:35:13Z
- go version go1.27.1 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 298 | 4691.47 MB/s | 0 | 0 | 8.5× |
| kr/logfmt | 1257 | 1114.18 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1482 | 944.68 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2540 | 551.19 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 64 | 2105.24 MB/s | 0 | 0 | 17.5× |
| kr/logfmt | 114 | 1188.15 MB/s | 0 | 0 | 9.9× |
| Grafana Loki | 140 | 962.96 MB/s | 0 | 0 | 8.0× |
| go-logfmt | 1122 | 120.36 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 177 | 842.43 MB/s | 0 | 0 | 7.6× |
| kr/logfmt | 345 | 431.34 MB/s | 112 | 3 | 3.9× |
| Grafana Loki | 365 | 408.63 MB/s | 112 | 3 | 3.7× |
| go-logfmt | 1344 | 110.87 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 65 | — | 0 | 0 | 19.9× |
| Grafana Loki | 337 | — | 80 | 1 | 3.8× |
| go-logfmt | 1289 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1358 | — | 152 | 4 | 0.9× |
