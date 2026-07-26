# logfmt parser comparison

- generated 2026-07-26T06:09:20Z
- go version go1.26.3 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 435 | 3221.66 MB/s | 0 | 0 | 5.7× |
| kr/logfmt | 1241 | 1127.92 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1484 | 943.15 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2485 | 563.27 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 88 | 1528.12 MB/s | 0 | 0 | 12.1× |
| kr/logfmt | 114 | 1188.19 MB/s | 0 | 0 | 9.4× |
| Grafana Loki | 143 | 946.96 MB/s | 0 | 0 | 7.5× |
| go-logfmt | 1066 | 126.64 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 217 | 685.64 MB/s | 0 | 0 | 6.0× |
| kr/logfmt | 329 | 452.53 MB/s | 112 | 3 | 3.9× |
| Grafana Loki | 366 | 407.40 MB/s | 112 | 3 | 3.6× |
| go-logfmt | 1300 | 114.61 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 86 | — | 0 | 0 | 14.5× |
| Grafana Loki | 319 | — | 80 | 1 | 3.9× |
| go-logfmt | 1241 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1386 | — | 152 | 4 | 0.9× |
