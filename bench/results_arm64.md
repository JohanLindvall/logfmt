# logfmt parser comparison

- generated 2026-06-24T19:19:47Z
- go version go1.26.3 linux/amd64
- note GOARCH=arm64 under QEMU emulation — timings are indicative only; native numbers come from CI (ubuntu-24.04-arm)
- cpu: ARMv8 Processor rev 0 (v8l) (16 cores)

This package vs other Go logfmt parsers on the same input. Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`. **Speedup** is relative to the `go-logfmt/logfmt` baseline.

## ParseAll_Big

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 4604 | 304.07 MB/s | 0 | 0 | 3.5× |
| kr/logfmt | 9256 | 151.25 MB/s | 80 | 1 | 1.7× |
| Grafana Loki | 10187 | 137.43 MB/s | 80 | 1 | 1.6× |
| go-logfmt | 16021 | 87.38 MB/s | 4352 | 4 | 1.0× |

## ParseAll_Typical

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 984 | 137.18 MB/s | 0 | 0 | 7.3× |
| kr/logfmt | 1073 | 125.84 MB/s | 0 | 0 | 6.7× |
| Grafana Loki | 1188 | 113.63 MB/s | 0 | 0 | 6.1× |
| go-logfmt | 7195 | 18.76 MB/s | 4272 | 3 | 1.0× |

## ParseEscaped

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| Grafana Loki | 2631 | 56.63 MB/s | 112 | 3 | 2.9× |
| kr/logfmt | 2663 | 55.94 MB/s | 112 | 3 | 2.9× |
| this (logfmt) | 2844 | 52.40 MB/s | 0 | 0 | 2.7× |
| go-logfmt | 7746 | 19.24 MB/s | 4384 | 6 | 1.0× |

## Extract

| Parser | ns/op | Throughput | B/op | allocs/op | Speedup |
|---|--:|--:|--:|--:|--:|
| this (logfmt) | 1031 | — | 0 | 0 | 15.4× |
| Grafana Loki | 9845 | — | 80 | 1 | 1.6× |
| kr/logfmt | 10815 | — | 152 | 4 | 1.5× |
| go-logfmt | 15830 | — | 4224 | 3 | 1.0× |
