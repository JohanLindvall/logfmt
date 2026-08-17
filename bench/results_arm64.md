# logfmt parser comparison

- generated 2026-08-17T14:12:15Z
- go version go1.26.6 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 378 | 3705.71 MB/s | 0 | 0 | 6.5× |
| kr/logfmt | 1251 | 1118.91 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1477 | 947.65 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2442 | 573.35 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 83 | 1629.29 MB/s | 0 | 0 | 12.5× |
| kr/logfmt | 114 | 1185.80 MB/s | 0 | 0 | 9.1× |
| Grafana Loki | 141 | 957.95 MB/s | 0 | 0 | 7.4× |
| go-logfmt | 1037 | 130.21 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 193 | 773.20 MB/s | 0 | 0 | 6.8× |
| kr/logfmt | 320 | 465.34 MB/s | 112 | 3 | 4.1× |
| Grafana Loki | 367 | 406.42 MB/s | 112 | 3 | 3.6× |
| go-logfmt | 1308 | 113.88 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 77 | — | 0 | 0 | 15.8× |
| Grafana Loki | 317 | — | 80 | 1 | 3.8× |
| go-logfmt | 1214 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1395 | — | 152 | 4 | 0.9× |
