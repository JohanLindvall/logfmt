# logfmt parser comparison

- generated 2026-08-17T16:46:24Z
- go version go1.26.6 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 391 | 3577.45 MB/s | 0 | 0 | 7.0× |
| kr/logfmt | 1503 | 931.67 MB/s | 80 | 1 | 1.8× |
| Grafana Loki | 1916 | 730.68 MB/s | 80 | 1 | 1.4× |
| go-logfmt | 2757 | 507.71 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 87 | 1553.18 MB/s | 0 | 0 | 11.7× |
| kr/logfmt | 139 | 968.23 MB/s | 0 | 0 | 7.3× |
| Grafana Loki | 162 | 834.78 MB/s | 0 | 0 | 6.3× |
| go-logfmt | 1014 | 133.07 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 223 | 668.53 MB/s | 0 | 0 | 5.6× |
| kr/logfmt | 351 | 424.42 MB/s | 112 | 3 | 3.5× |
| Grafana Loki | 362 | 411.73 MB/s | 112 | 3 | 3.4× |
| go-logfmt | 1246 | 119.62 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 90 | — | 0 | 0 | 13.0× |
| Grafana Loki | 344 | — | 80 | 1 | 3.4× |
| go-logfmt | 1173 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1587 | — | 152 | 4 | 0.7× |
