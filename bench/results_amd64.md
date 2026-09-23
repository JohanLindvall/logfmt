# logfmt parser comparison

- generated 2026-09-23T09:09:33Z
- go version go1.27.1 linux/amd64
- cpu: AMD EPYC 9V74 80-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 297 | 4713.03 MB/s | 0 | 0 | 8.0× |
| Grafana Loki | 1539 | 909.40 MB/s | 80 | 1 | 1.5× |
| kr/logfmt | 1564 | 895.05 MB/s | 80 | 1 | 1.5× |
| go-logfmt | 2375 | 589.55 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 64 | 2109.07 MB/s | 0 | 0 | 14.9× |
| kr/logfmt | 150 | 901.09 MB/s | 0 | 0 | 6.4× |
| Grafana Loki | 163 | 825.95 MB/s | 0 | 0 | 5.8× |
| go-logfmt | 955 | 141.37 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 179 | 833.47 MB/s | 0 | 0 | 6.7× |
| kr/logfmt | 325 | 458.83 MB/s | 112 | 3 | 3.7× |
| Grafana Loki | 325 | 458.45 MB/s | 112 | 3 | 3.7× |
| go-logfmt | 1196 | 124.56 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 71 | — | 0 | 0 | 15.2× |
| Grafana Loki | 275 | — | 80 | 1 | 3.9× |
| go-logfmt | 1078 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1605 | — | 152 | 4 | 0.7× |
