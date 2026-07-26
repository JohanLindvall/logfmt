# logfmt parser comparison

- generated 2026-07-26T12:28:13Z
- go version go1.26.5 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 456 | 3066.75 MB/s | 0 | 0 | 6.0× |
| kr/logfmt | 1494 | 937.12 MB/s | 80 | 1 | 1.8× |
| Grafana Loki | 1923 | 727.99 MB/s | 80 | 1 | 1.4× |
| go-logfmt | 2744 | 510.20 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 96 | 1406.39 MB/s | 0 | 0 | 10.6× |
| kr/logfmt | 141 | 956.62 MB/s | 0 | 0 | 7.2× |
| Grafana Loki | 164 | 825.46 MB/s | 0 | 0 | 6.2× |
| go-logfmt | 1018 | 132.59 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 229 | 649.61 MB/s | 0 | 0 | 5.2× |
| Grafana Loki | 356 | 418.86 MB/s | 112 | 3 | 3.4× |
| kr/logfmt | 368 | 405.11 MB/s | 112 | 3 | 3.2× |
| go-logfmt | 1195 | 124.64 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 96 | — | 0 | 0 | 12.0× |
| Grafana Loki | 344 | — | 80 | 1 | 3.4× |
| go-logfmt | 1154 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1593 | — | 152 | 4 | 0.7× |
