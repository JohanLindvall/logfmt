# logfmt parser comparison

- generated 2026-07-02T19:44:27Z
- go version go1.26.3 linux/arm64
- cpu: unknown (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 437 | 3201.07 MB/s | 0 | 0 | 5.6× |
| kr/logfmt | 1251 | 1119.25 MB/s | 80 | 1 | 2.0× |
| Grafana Loki | 1486 | 942.07 MB/s | 80 | 1 | 1.7× |
| go-logfmt | 2456 | 570.02 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 88 | 1527.81 MB/s | 0 | 0 | 12.0× |
| kr/logfmt | 113 | 1190.75 MB/s | 0 | 0 | 9.4× |
| Grafana Loki | 143 | 945.43 MB/s | 0 | 0 | 7.4× |
| go-logfmt | 1063 | 126.94 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 220 | 677.76 MB/s | 0 | 0 | 5.9× |
| kr/logfmt | 317 | 470.24 MB/s | 112 | 3 | 4.1× |
| Grafana Loki | 357 | 417.54 MB/s | 112 | 3 | 3.6× |
| go-logfmt | 1302 | 114.44 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 86 | — | 0 | 0 | 14.3× |
| Grafana Loki | 316 | — | 80 | 1 | 3.9× |
| go-logfmt | 1235 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 1386 | — | 152 | 4 | 0.9× |
