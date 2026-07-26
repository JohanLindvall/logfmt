# logfmt parser comparison

- generated 2026-07-26T06:09:20Z
- go version go1.26.3 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 456 | 3069.52 MB/s | 0 | 0 | 6.0× |
| kr/logfmt | 1470 | 952.55 MB/s | 80 | 1 | 1.9× |
| Grafana Loki | 1968 | 711.49 MB/s | 80 | 1 | 1.4× |
| go-logfmt | 2758 | 507.71 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 94 | 1431.33 MB/s | 0 | 0 | 10.5× |
| kr/logfmt | 125 | 1076.42 MB/s | 0 | 0 | 7.9× |
| Grafana Loki | 164 | 824.50 MB/s | 0 | 0 | 6.1× |
| go-logfmt | 994 | 135.85 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 227 | 655.51 MB/s | 0 | 0 | 5.3× |
| kr/logfmt | 321 | 464.15 MB/s | 112 | 3 | 3.7× |
| Grafana Loki | 372 | 401.12 MB/s | 112 | 3 | 3.2× |
| go-logfmt | 1197 | 124.52 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 95 | — | 0 | 0 | 12.0× |
| Grafana Loki | 348 | — | 80 | 1 | 3.3× |
| go-logfmt | 1139 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1563 | — | 152 | 4 | 0.7× |
