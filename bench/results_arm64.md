# logfmt parser comparison

- generated 2026-07-26T12:28:17Z
- go version go1.26.5 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 422 | 3315.01 MB/s | 0 | 0 | 5.8× |
| kr/logfmt | 1245 | 1124.06 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1485 | 942.51 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2454 | 570.43 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 88 | 1536.83 MB/s | 0 | 0 | 12.4× |
| kr/logfmt | 113 | 1192.22 MB/s | 0 | 0 | 9.6× |
| Grafana Loki | 143 | 941.44 MB/s | 0 | 0 | 7.6× |
| go-logfmt | 1085 | 124.43 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 216 | 688.47 MB/s | 0 | 0 | 6.1× |
| kr/logfmt | 327 | 456.14 MB/s | 112 | 3 | 4.0× |
| Grafana Loki | 372 | 401.02 MB/s | 112 | 3 | 3.5× |
| go-logfmt | 1313 | 113.45 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 85 | — | 0 | 0 | 14.8× |
| Grafana Loki | 325 | — | 80 | 1 | 3.9× |
| go-logfmt | 1258 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1390 | — | 152 | 4 | 0.9× |
