# logfmt parser comparison

- generated 2026-09-22T20:35:13Z
- go version go1.27.1 linux/amd64
- cpu: AMD EPYC 9V45 96-Core Processor (4 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 182 | 7711.93 MB/s | 0 | 0 | 7.4× |
| kr/logfmt | 758 | 1845.81 MB/s | 80 | 1 | 1.8× |
| Grafana Loki | 769 | 1819.64 MB/s | 80 | 1 | 1.8× |
| go-logfmt | 1352 | 1035.13 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 39 | 3457.92 MB/s | 0 | 0 | 16.1× |
| kr/logfmt | 69 | 1953.12 MB/s | 0 | 0 | 9.1× |
| Grafana Loki | 77 | 1743.28 MB/s | 0 | 0 | 8.1× |
| go-logfmt | 630 | 214.39 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 103 | 1442.01 MB/s | 0 | 0 | 6.8× |
| Grafana Loki | 168 | 885.76 MB/s | 112 | 3 | 4.2× |
| kr/logfmt | 175 | 853.00 MB/s | 112 | 3 | 4.0× |
| go-logfmt | 707 | 210.63 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 42 | — | 0 | 0 | 15.9× |
| Grafana Loki | 140 | — | 80 | 1 | 4.8× |
| go-logfmt | 665 | — | 4224 | 3 | 1.0× |
| kr/logfmt | 809 | — | 152 | 4 | 0.8× |
