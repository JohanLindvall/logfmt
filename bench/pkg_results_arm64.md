# logfmt microbenchmarks

- generated 2026-06-24T19:11:23Z
- go version go1.26.3 linux/amd64
- note GOARCH=arm64 under QEMU emulation — timings are indicative only; native numbers come from CI (ubuntu-24.04-arm)
- cpu: ARMv8 Processor rev 0 (v8l) (16 cores)

The Benchmark* functions in the root logfmt module (parser, lookups, unescape, ParseTime), as opposed to the cross-library comparison suite in this `bench/` module (see `results_<arch>.md`). Lower ns/op is better; throughput (MB/s) and allocations are reported by `-benchmem`.

| Benchmark | ns/op | Throughput | B/op | allocs/op |
|---|--:|--:|--:|--:|
| IterateOur | 7796.0 | — | 0 | 0 |
| GetMany_TimestampLevel | 1082.0 | — | 0 | 0 |
| UnescapeInto | 550.3 | — | 0 | 0 |
| DecodeKeyval_Custom | 6782190.0 | 73.72 MB/s | 0 | 0 |
| LevelTS_LogFmt | 826.7 | — | 0 | 0 |
| LevelTS_Regex | 114466.0 | — | 1076 | 4 |
| ParseTime_RFC3339 | 696.9 | — | 0 | 0 |
| ParseTime_Custom | 2853.0 | — | 0 | 0 |
| ParseTime_Unix | 529.9 | — | 0 | 0 |
