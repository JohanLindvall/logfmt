# logfmt parser comparison

- generated 2026-07-02T19:44:22Z
- go version go1.26.3 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 476 | 2939.12 MB/s | 0 | 0 | 5.8× |
| kr/logfmt | 1497 | 935.47 MB/s | 80 | 1 | 1.8× |
| Grafana Loki | 1917 | 730.14 MB/s | 80 | 1 | 1.4× |
| go-logfmt | 2741 | 510.79 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 99 | 1370.41 MB/s | 0 | 0 | 9.6× |
| kr/logfmt | 139 | 972.58 MB/s | 0 | 0 | 6.8× |
| Grafana Loki | 163 | 826.57 MB/s | 0 | 0 | 5.8× |
| go-logfmt | 946 | 142.68 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 236 | 632.65 MB/s | 0 | 0 | 5.1× |
| kr/logfmt | 348 | 427.74 MB/s | 112 | 3 | 3.4× |
| Grafana Loki | 358 | 416.82 MB/s | 112 | 3 | 3.3× |
| go-logfmt | 1192 | 125.03 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 101 | — | 0 | 0 | 11.1× |
| Grafana Loki | 343 | — | 80 | 1 | 3.3× |
| go-logfmt | 1126 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1595 | — | 152 | 4 | 0.7× |
