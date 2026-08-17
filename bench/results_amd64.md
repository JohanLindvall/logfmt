# logfmt parser comparison

- generated 2026-08-17T14:12:13Z
- go version go1.26.6 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 386 | 3631.85 MB/s | 0 | 0 | 7.3× |
| kr/logfmt | 1479 | 946.35 MB/s | 80 | 1 | 1.9× |
| Grafana Loki | 1919 | 729.73 MB/s | 80 | 1 | 1.5× |
| go-logfmt | 2801 | 499.77 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 86 | 1560.82 MB/s | 0 | 0 | 11.4× |
| kr/logfmt | 126 | 1075.82 MB/s | 0 | 0 | 7.9× |
| Grafana Loki | 170 | 792.96 MB/s | 0 | 0 | 5.8× |
| go-logfmt | 987 | 136.78 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 221 | 672.98 MB/s | 0 | 0 | 5.4× |
| kr/logfmt | 321 | 464.01 MB/s | 112 | 3 | 3.7× |
| Grafana Loki | 364 | 409.63 MB/s | 112 | 3 | 3.3× |
| go-logfmt | 1198 | 124.38 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 87 | — | 0 | 0 | 13.2× |
| Grafana Loki | 345 | — | 80 | 1 | 3.3× |
| go-logfmt | 1147 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1603 | — | 152 | 4 | 0.7× |
