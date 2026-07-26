# logfmt parser comparison

- generated 2026-07-26T17:38:40Z
- go version go1.26.5 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 388 | 3608.02 MB/s | 0 | 0 | 6.4× |
| kr/logfmt | 1251 | 1119.44 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1491 | 938.68 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2478 | 565.06 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 83 | 1624.32 MB/s | 0 | 0 | 13.2× |
| kr/logfmt | 114 | 1181.53 MB/s | 0 | 0 | 9.6× |
| Grafana Loki | 142 | 949.88 MB/s | 0 | 0 | 7.7× |
| go-logfmt | 1093 | 123.53 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 208 | 716.40 MB/s | 0 | 0 | 6.5× |
| kr/logfmt | 304 | 489.60 MB/s | 112 | 3 | 4.5× |
| Grafana Loki | 346 | 430.42 MB/s | 112 | 3 | 3.9× |
| go-logfmt | 1356 | 109.92 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 82 | — | 0 | 0 | 15.6× |
| Grafana Loki | 308 | — | 80 | 1 | 4.1× |
| go-logfmt | 1279 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1393 | — | 152 | 4 | 0.9× |
