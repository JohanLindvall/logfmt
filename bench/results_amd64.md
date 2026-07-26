# logfmt parser comparison

- generated 2026-07-26T17:38:39Z
- go version go1.26.5 linux/amd64
- cpu: AMD EPYC 9V74 80-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 386 | 3624.19 MB/s | 0 | 0 | 7.3× |
| kr/logfmt | 1552 | 902.27 MB/s | 80 | 1 | 1.8× |
| Grafana Loki | 2078 | 673.82 MB/s | 80 | 1 | 1.4× |
| go-logfmt | 2833 | 494.25 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 83 | 1620.27 MB/s | 0 | 0 | 11.9× |
| kr/logfmt | 146 | 926.21 MB/s | 0 | 0 | 6.8× |
| Grafana Loki | 173 | 780.41 MB/s | 0 | 0 | 5.7× |
| go-logfmt | 992 | 136.06 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 204 | 728.47 MB/s | 0 | 0 | 5.9× |
| kr/logfmt | 326 | 456.66 MB/s | 112 | 3 | 3.7× |
| Grafana Loki | 359 | 414.65 MB/s | 112 | 3 | 3.4× |
| go-logfmt | 1208 | 123.33 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 84 | — | 0 | 0 | 13.8× |
| Grafana Loki | 362 | — | 80 | 1 | 3.2× |
| go-logfmt | 1154 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1664 | — | 152 | 4 | 0.7× |
