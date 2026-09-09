# logfmt parser comparison

- generated 2026-09-09T10:23:39Z
- go version go1.27.1 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 300 | 4668.32 MB/s | 0 | 0 | 8.4× |
| kr/logfmt | 1255 | 1115.52 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1475 | 949.09 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2514 | 556.86 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 65 | 2090.84 MB/s | 0 | 0 | 17.1× |
| kr/logfmt | 114 | 1189.33 MB/s | 0 | 0 | 9.7× |
| Grafana Loki | 141 | 958.32 MB/s | 0 | 0 | 7.8× |
| go-logfmt | 1106 | 122.11 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 177 | 843.91 MB/s | 0 | 0 | 7.5× |
| kr/logfmt | 332 | 449.06 MB/s | 112 | 3 | 4.0× |
| Grafana Loki | 347 | 429.46 MB/s | 112 | 3 | 3.8× |
| go-logfmt | 1332 | 111.87 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 66 | — | 0 | 0 | 19.4× |
| Grafana Loki | 321 | — | 80 | 1 | 4.0× |
| go-logfmt | 1269 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1348 | — | 152 | 4 | 0.9× |
