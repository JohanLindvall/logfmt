# CLAUDE.md — logfmt

A fast, allocation-free, dependency-free reader for the logfmt line format.
Read-only `[]byte` parsing with direct key extraction. This file records the
performance design and, importantly, what has already been tried so it is not
re-attempted.

## ⚠️ Do NOT optimize `time.go` unless explicitly requested

`time.go` (`ParseTime`, `parseUnixTS`) is intentionally left on the simple,
correct `time.Parse`-based implementation. A faster hand-rolled parser is
possible (see "Rejected / parked" below) but `time.Parse`'s exact
acceptance/rejection semantics are full of quirks (e.g. it ignores a numeric
offset when the zone name is `UTC`; it validates day-of-month against the
month/leap-year). Matching them exactly is high-risk for little real benefit.
**Only touch `time.go` if the user explicitly asks to optimize timestamp
parsing.** The `Benchmark_ParseTime_*` benchmarks may stay as measurement.

## Layout

- `doc.go` — package documentation: API map, aliasing/read-only rules, and the
  documented leniency divergences from go-logfmt.
- `logfmt.go` — the core parser and key-lookup API (the "general parsing").
- `time.go` — `ParseTime` (see warning above).
- `logfmt_swar_test.go` — `FuzzIterateAgainstRef`: differential fuzz of the
  SWAR `Iterate` against a byte-by-byte reference. **Run this after any change
  to the parser.**
- `getmany_fuzz_test.go` — `FuzzGetManyAgainstRef`: differential fuzz of
  `GetMany`/`Get`'s first-non-empty duplicate resolution against a naive
  collect-all reference. **Run after any change to the lookup state machine.**
- `*_test.go` — unit tests, benchmarks, and a regex-vs-logfmt comparison.

## Public API (read-only, raw-by-default; keys are `string` everywhere)

- `Iterate(data, func(k, v) bool) error` — core primitive; calls back per pair,
  `k`/`v` alias `data` (bare key → shared `"true"` constant; all results are
  read-only). Quoted values have quotes stripped but escapes left intact (raw).
  `false` from the callback stops.
- `Get(data, key string) ([]byte, error)` — raw value, aliases `data`, zero-copy.
- `GetMany(data, keys, buf) ([][]byte, error)` — multi-key single pass, raw
  aliasing values, **`nil` for absent** (present-but-empty is a non-nil
  zero-length slice — distinct from absent), reusable outer `buf`, early-stop.
- `GetValue(data, key string, dst) ([]byte, error)` — unescaped; delegates to
  `Get` then `Unescape`, decoding into `dst` only when needed (result may alias
  `dst` *or* `data`).
- **Duplicate keys resolve identically in all three lookups: first non-empty
  occurrence wins; an empty value only if no non-empty one exists.** Guarded by
  `FuzzGetManyAgainstRef`.
- `Unescape(dst, raw)` / `NeedsUnescape(raw)` — decode `\n \r \t` and JSON-style
  `\uXXXX` incl. surrogate pairs (go-logfmt writes control chars as `\u00XX`,
  so this is required for round-trip interop); other escapes pass through, and
  malformed `\u` stays verbatim. `NeedsUnescape` is a single `IndexByte('\\')`
  so callers skip the decode when unnecessary — keep it a single expression so
  it stays inlinable (a SWAR helper here measurably regressed).

## Current benchmarks (Ryzen 7 8840HS, amd64; ~ns, machine-state dependent)

| Benchmark | ns/op | allocs |
|---|---:|---:|
| `Iterate` (sample2, ~900B real line) | ~275 | 0 |
| `GetMany` (timestamp+level, early-stop) | ~55 | 0 |
| `DecodeKeyval` (10k short-field rows) | ~1.28 GB/s | 0 |
| `LevelTS` logfmt vs regex | ~45 vs ~8900 | 0 vs 4 |
| `Unescape` | ~16 | 0 |

Everything on the hot path is **zero-allocation**. `Iterate` went 681 → ~275 ns
over the optimization history (~60% faster).

## How the general parser is optimized (logfmt.go)

- **SWAR scanning** (`hasCtrlOrSpace`, `hasByte`): scans keys/values 8 bytes per
  iteration. `hasCtrlOrSpace` flags bytes `<= 0x20` with one subtract (covers
  all whitespace); the located byte is re-checked so rare non-whitespace control
  bytes (0x00–0x08, 0x0E–0x1F) fall back to the scalar tail. Masks are only
  **OR-ed** then `TrailingZeros64`'d — never subtracted from each other (a borrow
  can set spurious high bits *above* a true match, which is fine for OR+find-
  first but breaks subtraction; this was a real fuzz-caught bug).
- **`binary.LittleEndian.Uint64(buf[i:i+8])`** (fixed-size slice, not `buf[i:]`)
  — this single change was a large win (337 → 278 ns): it lets the compiler emit
  a tighter load. Keep the `i+8` slice form.
- **`isSpace` is a 256-byte table lookup**, not arithmetic — measured faster
  (the table beat `b==' ' || b-'\t' <= '\r'-'\t'`, which mispredicts).
- **Verify-order**: at SWAR stop points, test the cheap expected byte first
  (`c == '=' || isSpace(c)` for keys, `c == ' ' || isSpace(c)` for values) so the
  common case short-circuits past the `isSpace` table load. `IsAbsent`/nil-style
  short-circuits similarly in `GetMany`.
- **`GetMany` uses `buf` itself as the found-marker** (slots start `nil`, a match
  fills them) — no parallel bitmask. Raw aliasing makes it zero-alloc and
  found-values are never nil, so `nil` == absent unambiguously.
- **Closing-quote verify tests `' '` first** (`c != ' ' && !isSpace(c)`) — same
  short-circuit trick as the SWAR verifies; ~1% on quoted-heavy lines.
- **`GOAMD64=v3` builds are ~4% faster** (measured: Iterate 267 vs 279 ns) —
  BMI's TZCNT helps the SWAR `TrailingZeros64`. A user build flag, not
  something the module can set; noted in the README.

## Rejected / parked (do NOT re-attempt without new evidence)

All measured back-to-back (averaged, benchstat-style — single runs are ±3–4 ns
noisy). Each was **neutral or worse**:

- **SIMD assembly (AVX2 32B and SSE2 16B)** for the key/value scan: **~17–21%
  slower**. `Iterate` calls the scanner ~once per key and per value (~50×/line)
  over short (~22B) fields; assembly **can't inline**, so per-call overhead
  (arg marshaling, `VZEROUPPER`, broadcast setup) overwhelms the wider scan.
  The lightning `pkg/unstable` team reached the same conclusion — their SIMD
  block-skip is used *only* on the bulk skip path; they note the two-stage SIMD
  feed "sank" for typed/every-field extraction. SWAR (inlined, 8B/iter) is the
  right tool for this access pattern. A whole-line tokenize-in-one-asm-call would
  amortize, but it sacrifices the zero-alloc streaming callback API.
- **`bytes.IndexByte('=')` for the key scan**: slightly slower even as an
  unchecked ceiling — 29 non-inlinable calls/line cost more than inlined SWAR.
- **Register-extract of the verify byte** (`byte(w >> (tz &^ 7))` instead of
  `buf[i]`): neutral — the reload is an L1 hit the CPU pipelines.
- **16-byte unrolled key scan**: no change (loop overhead wasn't the bottleneck;
  it's memory-latency bound).
- **Arithmetic `isSpace`**, **combined key-stop lookup table**, **`len(buf)` in
  the loop bound for BCE**: neutral or worse.
- **Inlining the parser into `GetMany`** (drop the callback indirection): only
  ~4.5% and it duplicated the parser — the prototype immediately diverged on
  bare keys under differential fuzz. Not worth the duplication/risk.
- **`GetMany` inner-loop comparison order**: the current settled-check first
  (`len(buf[j]) > 0 || string(k) != keys[j]`) is already fastest (54.8 ns).
  String-compare-first (55.4) and a found-prefix `start`-skip (56.3) both
  regress. `GetMany` is parse-bound — the match loop is ~15 ns of ~55 ns.
- **SWAR helper for the backslash search** (`indexBackslash` used by
  `NeedsUnescape`/`Unescape`): regressed both (Unescape 16→20.6 ns,
  ParseEscaped 126→136 ns). A SWAR scan needs a loop → the helper can't inline
  → every call pays a frame, where `bytes.IndexByte` leaves only the asm call
  and the `NeedsUnescape` wrapper inlines entirely. Corollary: the
  guard-then-decode pattern (`if NeedsUnescape(v) { Unescape(...) }`) beats
  calling `Unescape` unconditionally (127 vs 186 ns) for the same reason.
- **`len(data)` instead of a copied `n` throughout `Iterate`** (hoping the
  prove pass would drop the bounds checks): the checks all *remain* and it is
  ~3.7% slower. Note `-gcflags=all=-B` shows bounds checks cost ~8% — but that
  ceiling is not reachable from Go source; the prove pass keeps every hot check
  under both spellings.
- **PGO (`default.pgo` from the benchmark profile)**: mixed within noise
  (Iterate −2%, GetMany +3%). Also structurally pointless for a library: a
  committed profile affects only this module's own test builds, never
  importers' builds (PGO comes from the main module). Don't commit one.
- **Consuming the known-whitespace delimiter after an unquoted value**
  (`if i < n { i++ }` at valEnd, mirroring the quoted branch): −5% — the extra
  branch in the hot loop costs more than the saved top-of-loop `isSpace` load.
- **Inline first-word `hasByte(w,'"')` before `IndexByte` in the quoted scan**
  (to spare short quoted values the call overhead): −3% on `Iterate` (long
  quoted values pay the wasted word check) and neutral on `DecodeKeyval` —
  the short-quote saving never materialised.
- **Benchmarking note**: this machine drifts between power states *mid-session*
  (same code measured 283 → 297 ns minutes apart). Never compare against a
  stale baseline — interleave A/B runs (A,B,A,B…) and compare means.
- **Porting Rust's `memchr2` (AVX2 SIMD 2-byte search) to Go**: implemented and
  differential-tested correct; it beats stdlib `bytes.IndexAny` ~2.6× (the slow
  multi-byte fallback). But it **loses to inlined SWAR for logfmt-shaped fields**
  (5-key set: 38 ns vs SWAR 22 ns). Measured crossover vs SWAR: ~8 B → SWAR 2×
  faster; ~32 B → tied/slight memchr2; 128 B → memchr2 4×; 512 B → memchr2 6.7×.
  logfmt keys/values are mostly < 32 B, so SWAR (inlined, 8 B/iter, zero call
  overhead) wins; and the quoted-value scan already uses single-byte
  `bytes.IndexByte` (SIMD). memchr2 helps nothing here — removed. The portable
  takeaway: Rust gets a fast multi-byte SIMD search free (`memchr2/3`), Go does
  not, which is *why* this parser uses SWAR; but for short fields SWAR is the
  better tool regardless of language.

- **SIMD `key=` substring search for `Get`/`GetMany`** (jump straight to the key
  instead of walking fields): the find is real headroom — `bytes.Index` (already
  SIMD) locates `key=` ~3–4× faster than the sequential parse reaches it (level:
  13 vs 46 ns; deep key: 77 vs 297 ns). **But it cannot be made correct cheaply**
  and was not pursued. Two blockers: (1) `key=` occurs inside quoted values
  (`msg="set level=debug"`) preceded by an in-quote space, so a boundary check
  passes — a false match. (2) **logfmt quoting is position-dependent**: a `"`
  starts a string only at a value position (after `key=`); elsewhere it is a
  literal (`a=x" b=c` → `a`'s value is `x"`, and `b=c` is a real pair). So you
  cannot compute an in-string mask from quote positions — the simdjson /
  lightning-`skipfast` prefix-XOR technique is **invalid for logfmt**. Validating
  "not inside a quoted value" requires parsing field structure from the start,
  which negates the substring speedup. The only correct specializations
  (no-quotes line, or key before the first quote) are too restrictive for real
  logfmt. This is the core reason SWAR field-walking is the right design:
  logfmt's context-sensitive quoting defeats the context-free SIMD tricks that
  work for JSON.

The parser is **memory-latency / per-field-overhead bound**, not scan-throughput
bound. Further wins require an API change (non-callback) or accepting a
correctness/maintainability cost. Don't chase sub-ns micro-ops; they read as
wins in `-count=1` runs but vanish when averaged.

## Methodology (use this for any future perf work)

- **Differential fuzz** every parser change: `go test -run='^$'
  -fuzz=FuzzIterateAgainstRef -fuzztime=20s` (compares against a byte-by-byte
  reference). This has caught real bugs (SWAR borrow, inline-GetMany bare keys).
- **A/B with averaging**, not single runs: `-count=8 -benchtime=2s` and compare
  medians/means; ±3–4 ns is noise on this machine, and machine power-state
  drifts between sessions (absolute numbers shift ~30%).
- **Profile cumulative + line-level**: `-cpuprofile`, then `go tool pprof -top
  -cum` and `-list=Iterate`. Beware skid: `isSpace` and verify-line "flat %" are
  often attribution of dependent-load latency, not removable work.
- **Keep only measured wins; revert neutral changes** for clarity.

## Commands

```sh
go test ./...                                              # unit tests
go test -run='^$' -fuzz=FuzzIterateAgainstRef -fuzztime=20s # parser fuzz
go test -run='^$' -bench=. -benchmem -count=3             # benchmarks
go vet ./... && gofmt -l .                                # lint/format
```
