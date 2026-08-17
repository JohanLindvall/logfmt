# logfmt parser comparison

- generated 2026-08-17T13:17:12Z
- go version go1.26.6 linux/amd64
- cpu: AMD EPYC 7763 64-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 395 | 3541.33 MB/s | 0 | 0 | 7.0× |
| kr/logfmt | 1501 | 932.97 MB/s | 80 | 1 | 1.8× |
| Grafana Loki | 1913 | 731.87 MB/s | 80 | 1 | 1.4× |
| go-logfmt | 2764 | 506.59 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 87 | 1548.45 MB/s | 0 | 0 | 13.7× |
| kr/logfmt | 139 | 974.01 MB/s | 0 | 0 | 8.6× |
| Grafana Loki | 163 | 830.38 MB/s | 0 | 0 | 7.3× |
| go-logfmt | 1191 | 113.36 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 225 | 661.46 MB/s | 0 | 0 | 5.6× |
| kr/logfmt | 356 | 417.90 MB/s | 112 | 3 | 3.5× |
| Grafana Loki | 367 | 406.42 MB/s | 112 | 3 | 3.4× |
| go-logfmt | 1258 | 118.42 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 93 | — | 0 | 0 | 12.9× |
| Grafana Loki | 350 | — | 80 | 1 | 3.4× |
| go-logfmt | 1199 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1591 | — | 152 | 4 | 0.8× |
