# logfmt parser comparison

- generated 2026-08-08T16:37:13Z
- go version go1.26.5 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 414 | 3377.70 MB/s | 0 | 0 | 6.7× |
| kr/logfmt | 1507 | 929.21 MB/s | 80 | 1 | 1.8× |
| Grafana Loki | 1917 | 730.24 MB/s | 80 | 1 | 1.4× |
| go-logfmt | 2776 | 504.29 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 92 | 1470.45 MB/s | 0 | 0 | 11.1× |
| kr/logfmt | 140 | 961.01 MB/s | 0 | 0 | 7.2× |
| Grafana Loki | 162 | 833.50 MB/s | 0 | 0 | 6.3× |
| go-logfmt | 1018 | 132.57 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 234 | 636.58 MB/s | 0 | 0 | 5.2× |
| kr/logfmt | 351 | 424.92 MB/s | 112 | 3 | 3.5× |
| Grafana Loki | 360 | 413.78 MB/s | 112 | 3 | 3.4× |
| go-logfmt | 1221 | 122.05 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 88 | — | 0 | 0 | 13.0× |
| Grafana Loki | 344 | — | 80 | 1 | 3.3× |
| go-logfmt | 1147 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1581 | — | 152 | 4 | 0.7× |
