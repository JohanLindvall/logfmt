# logfmt parser comparison

- generated 2026-09-09T10:23:44Z
- go version go1.27.1 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 320 | 4369.93 MB/s | 0 | 0 | 8.2× |
| kr/logfmt | 1445 | 968.69 MB/s | 80 | 1 | 1.8× |
| Grafana Loki | 1769 | 791.40 MB/s | 80 | 1 | 1.5× |
| go-logfmt | 2638 | 530.73 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 71 | 1896.58 MB/s | 0 | 0 | 13.6× |
| kr/logfmt | 136 | 992.10 MB/s | 0 | 0 | 7.1× |
| Grafana Loki | 170 | 792.84 MB/s | 0 | 0 | 5.7× |
| go-logfmt | 969 | 139.32 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 203 | 733.76 MB/s | 0 | 0 | 5.8× |
| kr/logfmt | 341 | 436.74 MB/s | 112 | 3 | 3.4× |
| Grafana Loki | 357 | 417.17 MB/s | 112 | 3 | 3.3× |
| go-logfmt | 1176 | 126.72 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 78 | — | 0 | 0 | 14.5× |
| Grafana Loki | 314 | — | 80 | 1 | 3.6× |
| go-logfmt | 1124 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1528 | — | 152 | 4 | 0.7× |
