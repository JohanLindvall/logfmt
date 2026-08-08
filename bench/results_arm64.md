# logfmt parser comparison

- generated 2026-08-08T16:37:12Z
- go version go1.26.5 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 391 | 3576.73 MB/s | 0 | 0 | 6.3× |
| kr/logfmt | 1250 | 1119.90 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1512 | 925.99 MB/s | 80 | 1 | 1.6× |
| go-logfmt | 2459 | 569.37 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 85 | 1588.34 MB/s | 0 | 0 | 12.2× |
| kr/logfmt | 114 | 1184.33 MB/s | 0 | 0 | 9.1× |
| Grafana Loki | 144 | 940.14 MB/s | 0 | 0 | 7.2× |
| go-logfmt | 1038 | 130.06 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 214 | 696.78 MB/s | 0 | 0 | 6.0× |
| kr/logfmt | 332 | 448.65 MB/s | 112 | 3 | 3.8× |
| Grafana Loki | 375 | 397.33 MB/s | 112 | 3 | 3.4× |
| go-logfmt | 1275 | 116.90 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 81 | — | 0 | 0 | 14.9× |
| Grafana Loki | 342 | — | 80 | 1 | 3.6× |
| go-logfmt | 1215 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1394 | — | 152 | 4 | 0.9× |
