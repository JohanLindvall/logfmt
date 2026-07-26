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

- `doc.go` — package documentation: API map, record-framing rules (newlines are
  plain whitespace; the caller splits records), aliasing/read-only rules,
  streaming error semantics, leniency divergences from go-logfmt, and the
  explicit non-goals under "Scope".
- `logfmt.go` — the core parser and key-lookup API (the "general parsing").
- `time.go` — `ParseTime` (see warning above).
- `logfmt_swar_test.go` — `FuzzIterateAgainstRef`: differential fuzz of the
  SWAR `Iterate` against a byte-by-byte reference. **Run this after any change
  to the parser.**
- `getmany_fuzz_test.go` — `FuzzGetManyAgainstRef`: differential fuzz of
  `GetMany`/`Get`'s first-non-empty duplicate resolution against a naive
  collect-all reference. **Run after any change to the lookup state machine.**
- `*_test.go` — unit tests, benchmarks, and a regex-vs-logfmt comparison.
- `bench/` — separate module, **declares go 1.23** (above the library floor) so
  it can host `TestAllRangeOverFunc`, the consumer-side proof that `All` works
  with `for … range`. CI skips this module on the 1.21 floor job.
- `LICENSE` — MIT.

## Compatibility floor: `go 1.21`

`go.mod` declares `go 1.21` **on purpose** — a library's go directive is the
minimum toolchain every importer must have, not the version it was developed
on. CI runs the suite at 1.21 as well as at `stable`, so anything newer breaks
the build there. `clear()`, `min`/`max` and the `slices`/`maps` packages are all
1.21 and therefore fair game; the `iter` package (1.23) is **not**.
Range-over-func did **not** require the bump: `All` returns the bare
`func(yield func(k, v []byte) bool)` type rather than `iter.Seq2`, so the 1.23
requirement lands on the consumer's module, not this one. `bench/go.mod` is
1.23 precisely so it can be that consumer in a test.

## Public API (read-only, raw-by-default; `[]byte` in and out, keys are `string`)

Reshaped 2026-07-26 in one breaking pass, while the module was still v0.x — see
"API design rules" below before changing any of it.

- `Iterate(data, func(k, v) bool) error` — core primitive; calls back per pair,
  `k`/`v` alias `data` (bare key → shared `trueSlice`; all results read-only).
  Quoted values have quotes stripped but escapes left intact (raw). `false` from
  the callback stops. **The only function that reports errors alongside data.**
- `All(data) func(yield func(k, v []byte) bool)` — range-over-func wrapper over
  `Iterate`. Deliberately the bare func type, **not** `iter.Seq2`: that keeps the
  `iter` import (and the go 1.23 floor) out of this module, while consumers on
  1.23+ can still `for k, v := range`. Proven by `TestAllRangeOverFunc` in the
  bench module, which declares 1.23 for exactly that purpose.
- `Get(data, key) ([]byte, bool)` — raw value, aliases `data`, zero-copy, capped.
- `GetMany(data, keys, buf) [][]byte` — multi-key single pass, raw aliasing
  capped values, **`nil` for absent** (present-but-empty is a non-nil
  zero-length slice — distinct from absent), reusable outer `buf`, early-stop.
- `AppendValue(dst, data, key) ([]byte, bool)` — unescaped, **always appends**;
  never aliases `data`. Absent key returns `dst` untouched and false.
- `Validate(data) error` — full parse for callers who need the error the
  lookups structurally cannot give them.
- `SplitRecord(data) (record, rest)` — record framing (see limits below);
  trims a trailing `\r`, caps `record`.
- `IsBareKey(val)` — identity test against `trueSlice`, the only way to tell
  `debug` from `debug=true`.
- **Duplicate keys resolve identically in all three lookups: first non-empty
  occurrence wins; an empty value only if no non-empty one exists.** Guarded by
  `FuzzGetManyAgainstRef`.
- `AppendUnescape(dst, raw)` / `NeedsUnescape(raw)` — decode `\n \r \t` and
  JSON-style `\uXXXX` incl. surrogate pairs (go-logfmt writes control chars as
  `\u00XX`, so this is required for round-trip interop); other escapes pass
  through, and malformed `\u` stays verbatim. `NeedsUnescape` is a single
  `IndexByte('\\')` so callers skip the decode when unnecessary — keep it a
  single expression so it stays inlinable (a SWAR helper here measurably
  regressed).
- `ParseTime(ts []byte)` — `[]byte` like everything else. A caller holding a
  `[]byte` pays the same 5 allocs on the named-zone layout either way (measured
  both sides); the old `string` benchmark only looked cheaper because it fed a
  compile-time constant.

## API design rules (why it looks like this)

- **Errors only where they can be honest.** The lookups early-stop, so they
  cannot see a fault past the keys they settled. They therefore return no error
  at all rather than a `nil` that means "resolved" instead of "valid".
  `Validate` exists for callers who want the real answer.
- **Absence is comma-ok, uniformly** (`GetMany`: a `nil` slot). Not an error:
  a missing key is routine control flow, and `errors.Is` on a hot path is
  noise. There is no `ErrKeyNotFound` any more.
- **`Append*` means it appends.** Both append functions always copy into `dst`
  and never alias the input; the conditional "returns raw if dst is empty"
  behaviour the old `Unescape` had was a trap. Zero-copy is still available and
  is still the faster pattern — `Get` + `NeedsUnescape` — just explicit now.
- **Destination first** (`AppendUnescape(dst, raw)`, `AppendValue(dst, data,
  key)`), matching `append` and the stdlib `Append*` family.

## Known functional limits (documented, not bugs — audited 2026-07-26)

Deliberate behaviours that surprise people; all are now in `doc.go`/README.

- **No record framing in the parser.** `'\n'`/`'\r'` are plain whitespace, so a
  multi-line buffer parses as one flat pair stream and a lookup can match a key
  from a later line. `SplitRecord` is the supported way to split; the parser
  itself stays framing-free. (`Benchmark_DecodeKeyval_Custom` relies on this.)
- **Lookups do not report syntax errors** — they early-stop, so a malformed tail
  past the settled keys is never reached. They return what the reachable prefix
  holds; `Validate` is the honest full-parse. This also made
  `FuzzGetManyAgainstRef` *stronger*: both sides now consume the same valid
  prefix, so they must agree exactly, with no error-case carve-outs.
- **`Iterate` delivers the valid prefix before returning its error** (a
  `*SyntaxError` carrying the fault offset; `errors.Is(err, ErrBadFormat)` still
  matches, via the type's `Is` method).
- **Capping is asymmetric, on purpose.** `Get`/`GetMany` return
  values with `cap == len` (`v[:len(v):len(v)]` at the assignment sites), so a
  caller's `append` copies instead of overwriting the rest of the line — free
  there, measured: GetMany 55.8 → 55.5 ns over 3 interleaved A/B rounds.
  `Iterate` does **not** cap (−4.5%, see below), so callback values still carry
  capacity into the input. Pinned by `Test_Unit_Lookups_CapValues`, which also
  guards the thing capping could have broken: slicing keeps a present-but-empty
  value non-nil, which is how absence stays distinguishable.
- **The bare-key `trueSlice` is a shared global** — mutating it is process-wide.
- **Keys are never quoted**: `"a b"=c` → bare key `"a`, then `b"`=c. Quoting is
  position-dependent (value position only) — the same property that defeats the
  SIMD substring search below.
- **`ParseTime` epochs are exactly 10 digits** → 2001-09-09 .. 2286-11-20, no
  negatives, and ms/µs epochs (13/16 digits) are rejected by design.
- Statement coverage is 98.4%; both differential fuzzers pass 45 s clean
  (17.1 M and 15.4 M execs, no new failures).

## Current benchmarks (Ryzen 7 8840HS, amd64; ~ns, machine-state dependent)

| Benchmark | ns/op | allocs |
|---|---:|---:|
| `Iterate` (sample2, ~900B real line) | ~266 | 0 |
| `GetMany` (timestamp+level, early-stop) | ~55 | 0 |
| `DecodeKeyval` (10k short-field rows) | ~1.28 GB/s | 0 |
| `LevelTS` logfmt vs regex | ~45 vs ~8900 | 0 vs 4 |
| `Unescape` | ~16 | 0 |

Everything on the hot path is **zero-allocation**. `Iterate` went 681 → ~266 ns
over the optimization history (~61% faster); the last step was `hasKeyStop`
(2026-07-26, −3.3%).

CI-generated tables (EPYC 7763, ~1.7× slower than this laptop) live in
`bench/pkg_results_<arch>.md` and `bench/results_<arch>.md`; the README quotes
those, since they are the reproducible ones.

### Cost model (measured 2026-07-26, synthetic field-size sweep)

| Shape | Cost |
|---|---|
| Fixed per-field overhead (4–16 B values) | ~5.8–7.2 ns/field |
| Unquoted value scan (SWAR), 256 B values | ~11.7 GB/s |
| Quoted value scan (`bytes.IndexByte`), 512 B | ~27 GB/s |
| `Get` per skipped field | ~8.6 ns (10.7 ns for field 0, 551 ns for field 63) |

Reading: short fields are **overhead-bound** (~6 ns of loop/callback per pair,
scan is noise), long values are **scan-bound**. The 8 B/iter SWAR only starts
paying off above ~32 B — which is why the memchr2/SIMD experiments below lost.
`sample2` averages ~8.3 ns/field, consistent with the sweep.

## How the general parser is optimized (logfmt.go)

- **SWAR scanning** (`hasCtrlOrSpace`, `hasByte`): scans keys/values 8 bytes per
  iteration. `hasCtrlOrSpace` flags bytes `<= 0x20` with one subtract (covers
  all whitespace); the located byte is re-checked so rare non-whitespace control
  bytes (0x00–0x08, 0x0E–0x1F) fall back to the scalar tail. Masks are only
  **OR-ed** then `TrailingZeros64`'d — never subtracted from each other (a borrow
  can set spurious high bits *above* a true match, which is fine for OR+find-
  first but breaks subtraction; this was a real fuzz-caught bug).
- **`hasKeyStop` fuses the two key-scan masks** (`'='` and `<= 0x20`) by
  factoring the `& swarHi` that both terms end in out of the OR: one ALU op
  fewer per word, **−3.3 to −3.9% on `Iterate`** (p≤0.04 across two pinned
  runs), neutral everywhere else. Both terms must still be combined with OR
  only — the borrow caveat above is unchanged. Don't re-split it into
  `hasCtrlOrSpace(w) | hasByte(w, '=')`; `hasByte` was removed with it.
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
- **`GOAMD64=v3` builds are ~3–4% faster** (re-measured 2026-07-26, interleaved
  A/B: Iterate 275.2 → 266.9 ns = 3.0%; GetMany 54.9 → 53.4 ns = 2.7%) —
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
- **Three-index capping of the slices handed to `fn`** (`data[kStart:kEnd:kEnd]`,
  to stop a caller's `append` from scribbling over the rest of the input):
  correct and tests pass, but **−4.5% on `DecodeKeyval`** (391.5 → 410.1 µs,
  1277 → 1222 MB/s; consistent across 3 interleaved A/B rounds) and ~−0.7% on
  `Iterate`. Field-dense input pays it per pair. Rejected **for `Iterate` only**
  — the read-only contract is documented instead. The lookups *do* cap (below);
  the asymmetry is the whole point and is documented in both README and doc.go,
  so don't "fix" it in either direction without re-measuring.
- **Fusing the value scan into the key scan's word** (when `key=value ` fits in
  one 8-byte word, the key mask already locates the value's end, so the pair can
  be emitted with no second load and no second SWAR loop). Implemented, fuzz-
  clean, and it does exactly what it promises on short fields: **−6.1% on
  `DecodeKeyval`** (p=0.002). Rejected anyway, because that shape is the
  synthetic one. On real input the wasted check costs more than the hits save:
  **`ParseAll_Typical` +2.3%, `Extract` +2.3%, `LevelTS` +4.6%** (all p≤0.04),
  `ParseAll_Big`/`Iterate`/`GetMany` neutral — geomean **+1.6% on the realistic
  suite**. Fields only fuse when key+value+delimiter ≤ 8 B; real logfmt keys
  (`session_attr_*`, `timestamp`) blow that instantly, and every quoted value
  pays the failed check. Related: consuming the known delimiter in the fused
  path (`i = ve + 1`) was a further −1.7%, so if this is ever revisited on a
  short-field-only workload, include it.
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
- **The A/B harness itself needs three things, learned the hard way (2026-07-26)
  when a naive one manufactured a 4–6% "regression" out of nothing:**
  1. **Alternate the order** (A,B then B,A per round). Running A-then-B every
     round makes whichever goes second look ~3–5% slower.
  2. **Pin to a core** (`taskset -c 6`). This cut run-to-run variance from
     ±8–16% to ±1–2%, which is the difference between deciding and guessing.
  3. **Run a control first**: A/B two *identical* copies and confirm benchstat
     reports `~` on everything. If the control shows a delta, the harness is
     lying and no result from it counts. The tell in the cross-library suite was
     go-logfmt/Loki/kr — code neither variant touches — "regressing" 4–10%.
  Use `benchstat` (p-values), not eyeballed means; with n=6 pinned rounds a real
  3% effect lands at p≤0.05 and noise stays at `~`.
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
