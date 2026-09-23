# CLAUDE.md — logfmt

## Latest pass: arm64-native escape paths from hardware counters (2026-09-23)

Neoverse N2 (the Azure arm64 VM), Go 1.27.1, decided on per-op cycles from
`bench/counters.py` — which now counts the Armv8 PMUv3 events this VM exposes
(`OP_RETIRED`, `BR_RETIRED`, `STALL_BACKEND`, `STALL_FRONTEND`, … — the old
note that only generic events count here is wrong) — and confirmed by pinned,
alternating timing series. Numbers, controls and the rejected list are in
"2026-09-23 pass" below and in [bench/perf_2026-09-23_arm64.md](bench/perf_2026-09-23_arm64.md).
**amd64 compiles to the same machine code as before, function for function**
(`bench/asmdiff.py`), so nothing here is an x86 claim.

- **arm64 has its own escape paths now**, in build-tagged files beside the
  generic ones: `scan_arm64.go` (the walk, spelled so a clean word costs 16
  instructions instead of 22 and a lane's byte is re-checked from the register;
  `escClean` = 8 and `escGap` = 64; and a sparse scan that hands a value back to
  the walk once) and `scan_other.go` (the amd64-tuned code, moved verbatim out
  of `logfmt.go`). Escape rows −9…−29%, prose-then-JSON −64%.
- **The KNOWN GAP is closed on arm64**, and the trap in closing it is written
  down: letting the sparse scan hand ANY value back after every short gap cost
  +28–34% on alternating short/long gaps (JSON with long string values); only a
  value the walk declined on arrival may go back, and only once.
  `Benchmark_IterateEscapedAlternating` pins that shape.
- **`AppendUnescape` decodes into spare capacity on arm64** (`unescape_spare.go`,
  chosen by the constant `runtime.GOARCH == "arm64"`, so amd64 compiles the
  dispatch away): no `memmove` call per literal run up to 32 bytes, an adaptive
  8-word probe, surrogate pairs without a call. Decode rows −7…−33%, JSON
  −17%; `Unescape` (a short value, two escapes) +3.6%.
- **`Iterate` itself is unchanged**, after a long list of measured attempts
  (below). Short fields on this core are bound by the callback's round trip and
  the key-verify chain as a whole; no single removal gives back more than 1%.
  The +0.7% `IterateOur` shows is placement — `Iterate` moved from offset 32 to
  16 of a 64-byte block — and a padding control that moves it back matches
  the baseline to ±0.0%. On this core the same code retires ~40 fewer ops per
  parse at 32-byte-aligned offsets: compare-and-branch fusion depends on where
  the pair falls.
- New tools: `bench/asmdiff.py` (which functions' machine code differs between
  two builds — run it cross-compiled for every architecture a change is NOT
  meant for), `counters.py` on arm64 (plus a mispredicts column), and
  `make bsfdep` now cross-compiles, so it means the same thing on an arm64 host.

## Previous pass: hardware counters and IBS on amd64 (2026-09-22)

Driven by `perf stat` counters and AMD IBS per-op sampling rather than by the
stopwatch; the numbers, the mechanisms and the rejected list are in "2026-09-22
pass" below. Ryzen 8840HS, Go 1.27.1, decided on cycles/op, which clock and
power-state drift cannot move.

- **A BSF waits for its destination register — check every BSF after touching
  `Iterate`.** At GOAMD64=v1 `bits.TrailingZeros64` is `BSF`, which leaves the
  destination unchanged for a zero source, so the core treats the old value as
  an input (measured: dependent BSF steps 2 cycles each, 0.6 with the
  destination zeroed first). The register allocator hands a BSF the lowest
  free register, and that is often the one a byte load has just released,
  which chains the load onto the per-field critical path. Removing the
  `quoted` out-parameter produced an instruction stream IDENTICAL to HEAD bar
  register names that ran short unquoted fields +37% and DecodeKeyval +12%;
  loading the three scan constants in another order fixed it. Run
  `bench/bsfdep.py` on a test binary after any change to `Iterate` and treat a
  SUSPECT line as a regression until measured. This mechanism is a candidate
  explanation for much of what earlier passes filed as layout noise.
- `Iterate` is the parser now: the unexported `iterate` and its `quoted *bool`
  protocol are gone, and `GetQuoted` reads the bit from the byte past the value
  (`wasQuoted`), which relies on `Iterate` never capping what it delivers.
- The escape walk drains a pair with `m &= (m | -m) << 9` — four dependent ops
  where `m &^= low | low<<8` took six — and takes the lane from
  `TrailingZeros64(m)`, which the loop condition proves non-zero (no CMOV):
  esc=500 −22%, embedded JSON −9%, prose-then-JSON −13%, Iterate −1%.
- `AppendUnescape` decodes every escape but `\u` through a 256-byte table. The
  dense decode loop is FRONTEND-bound (55% of dispatch slots empty), and the
  `switch` was a scattered binary search costing 3–4 taken branches per escape:
  esc=500 −32%, esc=128 −9%, JSON msg −4%.
- One-key lookups −4…−7% and LevelTS −8% (the quoted-flag load/store per field
  is gone); `GetMany`'s two paths share `resetSlots`/`lenSet`/`settle`.
- New benchmarks: `Benchmark_Get` (one-key lookups by depth, and absent) and
  `Benchmark_IterateFieldShape`, whose rows show a change to the per-field
  chain at full size. New tools: `bench/counters.py` (per-op cycles,
  macro-ops, branches) and `bench/bsfdep.py`.

## Follow-up: lookup and decoder work on arm64 (2026-09-09)

See [the local performance report](bench/perf_2026-09-09_arm64.md) for this
follow-up to the amd64 pass recorded below, including controls, tradeoffs, and
reproduction commands. The general parser and `time.go` are unchanged.

- `GetMany` returns immediately for zero queries. For 32–256 queries on records
  at least four bytes per requested key, it uses a lazy, bounded stack index.
  Its 64 bucket heads and 256 links retain query order; settling a non-empty
  value removes a slot, while provisional empties remain. Full comparisons
  verify every match. The small-query and very-short-record paths remain linear.
  Older blanket linear-matching/crossover descriptions below predate this work.
- Keep the key-length rejection before index construction: eager construction
  regressed all-absent workloads. `keyBucket` accepts strings and byte slices
  directly through a generic helper; converting parsed keys to strings here
  copies them and allocates for long keys. Equality conversions remain free.
- `AppendUnescape` performs its first search and prefix copy before the loop,
  bypasses scanning/copying between adjacent escapes, and reserves raw length
  once for destinations with zero capacity. This last change can retain more
  capacity than the decoded result needs; existing buffers still avoid allocating
  if they fit the decoded output, even when they do not fit the raw input.
  `hex4` is inline on the common Unicode path; `decodeSurrogateEscape` receives
  the parsed first code unit instead of re-reading it. `unescWindow` stays 4,
  and the `s >= 0` bounds-proof hint remains on the probe loop.
- `perf_lookup_test.go` adds a duplicate-slot-aware reference and fuzzer. The
  older three-key fuzzer cannot reach the index, and its reference deliberately
  skips duplicate queries. Run both lookup fuzzers after state-machine changes.
  `perf_shapes_test.go` pins exact decoded-capacity reuse and fresh-buffer
  allocation counts; the new benchmarks cover query order, collisions,
  missing keys, short records, fresh buffers, surrogates, and malformed escapes.
- `bench/compare.py` runs alternating, CPU-pinned A/B rounds over prebuilt
  binaries. Use identical test sources on both sides; run an A/A control first.


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
- `scan_other.go` / `scan_arm64.go` (2026-09-23) — the escape walk
  (`scanQuotedEscapeDense`), the sparse quoted scan (`scanQuotedSparse`) and
  their thresholds `escClean`/`escGap`, per architecture. `scan_other.go`
  (`!arm64`) is the amd64-tuned code moved verbatim out of `logfmt.go`, and
  builds to the same machine code as before; `scan_arm64.go` is the N2-tuned
  version (register byte re-check, a quiet limit instead of a counter, 8/64,
  the one-time hand-back from the sparse scan). `Iterate` calls whichever is
  built and its own source is shared.
- `unescape_spare.go` (2026-09-23) — `unescapeInto`, the decoder for a
  destination with room for all of `raw`, and `unescapeIntoSpare =
  runtime.GOARCH == "arm64"`, the constant `AppendUnescape` dispatches on after
  its first `IndexByte`. Portable Go, used on arm64 only because it was measured
  there only; on amd64 the constant-false branch is removed before code
  generation (checked with `bench/asmdiff.py`) and the linker drops the rest.
- `time.go` — `ParseTime` (see warning above).
- `logfmt_swar_test.go` — `FuzzIterateAgainstRef`: differential fuzz of the
  SWAR parser (`Iterate`; the unexported `iterate` of earlier notes is gone)
  against a byte-by-byte reference. **Run this after any change
  to the parser.** It compares FOUR facts per pair, not two: key, value,
  `IsBareKey(v)` and `quoted`. The last two were added 2026-08-08 because
  comparing `string(k)`/`string(v)` alone left it blind to properties the package
  exports — replacing either `trueSlice` call site with a fresh `[]byte("true")`
  is byte-identical yet flips `IsBareKey`, and survived 6.89 M execs. It also no
  longer skips the pair comparison when an error came back (the old
  `gotErr == nil &&` guard discarded the check on ~12% of short inputs, i.e. the
  whole quoted-value error region), and it carries malformed seeds — before those
  were added, **no seed reached an error path at all**, so CI, which runs seeds
  rather than a corpus, never exercised it. Since 2026-08-17 it also carries
  escape-dense seeds (JSON in `msg=`, backslash runs of both parities before a
  quote, an escaping backslash as the last byte of an 8-byte word and as the
  last byte of the input) for the SWAR follow-up scan in the quoted branch, and
  drives the parser through `iterateQ`, the test-side adapter that adds the
  quoted bit — since 2026-09-22 derived by `wasQuoted`, exactly as `GetQuoted`
  derives it, so the comparison against the reference's own state machine is
  what checks that derivation on every pair (see "The quoted-bit protocol").
  Also `Test_Unit_SWARMasks` (exhaustive: every byte value in every lane, for
  all four masks — `hasKeyStop`, `hasCtrlOrSpace`, `hasQuoteOrBackslash`,
  `hasBackslash` — plus "the lower of two stops wins"; since 2026-09-22 the
  argument-free forms are one-line calls of the register-fed `*R` forms with
  `swarRegs`' values, so the exhaustive check covers what `Iterate` runs and
  each formula is written once) and `Test_Unit_IsSpace`.
  `isSpace` needs its own test precisely *because* the reference above shares
  it — a bug there cancels out and the fuzzer sees nothing.
- `getmany_fuzz_test.go` — `FuzzGetManyAgainstRef`: differential fuzz of
  `GetMany`/`Get`/`GetQuoted`'s first-non-empty duplicate resolution against a
  naive collect-all reference, which also tracks the quoted flag so
  `AppendValue`'s decode-only-if-quoted rule is checked rather than assumed.
  **Run after any change to the lookup state machine.** It uses
  `AppendUnescape` as its own oracle for `AppendValue`, so a bug inside
  `AppendUnescape` cancels on both sides there — which is what the next file
  is for.
- `testdata/fuzz/FuzzIterateAgainstRef/ee6d5b3abecfadf7` — a committed corpus
  entry, not a seed in code: the input that found a real panic in the
  escape-dense scan (the hand-back to `bytes.IndexByte` could leave the position
  at `n+1`, where `data[i:]` does not merely fail to match but crashes). It came
  from the maintainer's `perf/escape-dense-quoted-scan` branch (PR #2) and is
  kept because CI runs corpus files as well as seeds.
- `unescape_fuzz_test.go` — `FuzzAppendUnescapeAgainstRef` (added 2026-08-17,
  when `AppendUnescape` gained its SWAR follow-up scan): a byte-at-a-time
  reference decoder spelled out independently, checked three ways — append to
  nil, append behind a prefix, and **decode in place** (`dst = raw[:0]`, legal
  because decoding never lengthens, and the one case a read-ahead can break) —
  plus `NeedsUnescape(raw) == false ⇒ raw decodes to itself`. **Run after any
  change to `AppendUnescape`.** Mutation-checked: swapping `\r` for `\n` in the
  decoder fails on the seeds alone. Since 2026-09-23 it checks five placements,
  not three — also a prefix WITH spare capacity and an in-place decode behind a
  prefix in the same buffer, the two that reach arm64's `unescapeInto` besides
  `dst = raw[:0]` — and carries seeds whose literal runs sit on every boundary
  of that decoder's copies (1, 2, 4, 8, 16, 32 bytes, `memmove`) and of its
  probe window. **The runs are non-periodic on purpose**: with runs of `x`, a
  copy that read the last word one byte early survived, because it copied the
  same bytes; four such mutations of the copies now each fail on the seeds.
- `*_test.go` — unit tests, benchmarks, and a regex-vs-logfmt comparison.
  `Test_Unit_HotPath_Allocs` pins the allocation-free contract across all 14
  entry points that claim one (previously only 2 were asserted, and three
  injected allocations left the suite green); `Test_Unit_Malformed_Allocs` pins
  the single carve-out. `Test_Unit_Lookups_CapValues` now asserts `Get` really
  aliases the input at a known offset — `cap == len` plus "append didn't touch
  the line" is satisfied by a heap copy, so the old form passed with `Get`
  copying. `Test_Unit_Unquoted_Backslashes_Are_Literal` pins the quoted/unquoted
  decode split. `Benchmark_IterateEscaped` sweeps escape density at fixed
  length and `Benchmark_UnescapeEscaped` sweeps it for the decoder (from PR #2),
  which is the slower half once escapes are dense; `Benchmark_IterateJSONMsg` /
  `Benchmark_UnescapeJSONMsg` (2026-08-17) are the realistic point on that axis —
  a structured event serialised into a `msg=` field, one escape per ~7 bytes —
  for the parser and the decoder respectively. Tune the escape constants against
  those two and the 1.4 KB sample, NOT against the synthetic sweep alone: the
  sweep jumps straight from 32-byte gaps to 8-byte gaps, and real logfmt sits in
  between (the sample's own escapes are 38 bytes apart).
  `Test_Unit_Quoted_EscapeDense_Scan` (from PR #2, adapted to `escGap`/
  `escClean`) pins both scans and every transition between them, including the
  two malformed shapes only the walk can reach, and — since 2026-09-09, when
  the walk began draining a word rather than re-anchoring on each escape — a
  word holding four escapes and the two **spurious-lane** shapes (`\"#`, `\\]`)
  that only the drain can ever read. Those two are also fuzz seeds: a walk that
  trusted such a lane would read the `#` as a closing quote or step two bytes
  over the `]`, and nothing else in the suite puts a byte one greater than an
  escape immediately after it.
  **Added 2026-08-17 by the amd64 review, all four to close blind spots that had
  each already cost a tuning pass:**
  `Benchmark_IterateEscapedGap` / `Benchmark_UnescapeEscapedGap` sweep the same
  axis as the two above but parameterised by the **distance between escapes**
  rather than by a count, with points at 16/32/40/48/64/128/256 — i.e. bracketing
  the decisions instead of sampling evenly. The count-parameterised sweep's blind
  spot has now hidden a regression (the 32-byte window, −7% GetMany) *and* a win
  (`escClean` 5, −10% at a 48-byte gap read as `~` on every committed row).
  `Benchmark_IteratePrefixJSON` pins the `escGap` cliff (see Known limits) —
  closed on arm64 since 2026-09-23, where `Benchmark_IterateEscapedAlternating`
  (escapes alternating short and long gaps, i.e. JSON with long string values)
  pins the shape the closing must not break.
  `Benchmark_UnescapeUnicode` / `Benchmark_AppendValueUnicode` are the first
  benchmarks in this package's history to execute `hex4` or
  `decodeUnicodeEscape` **at all** — every escaped sample here carries `\" \\ \t
  \n` and none carried `\u`, though decoding `\u00XX` is the documented
  round-trip requirement for go-logfmt's own output. Keep them even if `hex4`
  changes shape again: an unmeasured path is exactly how that one drifted.
- `bench/` — separate module, **declares go 1.23** (above the library floor) so
  it can host `TestAllRangeOverFunc`, the consumer-side proof that `All` works
  with `for … range`. CI skips this module on the 1.21 floor job.
  Its `lokifmt/` package is a benchmark stand-in for Loki's in-tree decoder,
  **reimplemented from go-logfmt v0.6.1 (MIT, licence text in the directory)
  — do NOT re-vendor from the Loki tree**: `pkg/logql` is AGPL-3.0-only (it is
  not in Loki's Apache-2.0 exception list). Loki's only behavioural edit
  (unquoteBytes accepts control bytes) is mirrored; the stand-in was
  differentially verified against the old vendored copy (pairs/err/msg/pos
  identical on the samples + a malformed battery) and A/B'd clean (control
  clean; big line −3% from the dead resync code going away).
- `bench/counters.py` (2026-09-22) — per-op hardware counters for prebuilt
  test binaries: cycles, macro-ops, instructions and branches per op by the
  difference method (two iteration counts, subtracted), pinned, interleaved,
  best of `--reps`. Cycles are immune to the clock drift that moves this
  machine's ns/op by up to 30%, and the op counts are exact, so it screens a
  change in seconds and stays readable when the machine is too busy for a
  timing series. `--portable` counts generic events only. Since 2026-09-23 it
  picks the event set by host: on arm64 the PMUv3 common events by number
  (`OP_RETIRED`, `BR_RETIRED`, `STALL_BACKEND`, `STALL_FRONTEND`,
  `BR_MIS_PRED_RETIRED`), adding the two stall shares (`be%`, `fe%`) as
  columns; both sets print mispredicts per op.
  `bench/bsfdep.py` (2026-09-22) — lists every possible previous writer of each
  BSF's destination register in a function (see the 2026-09-22 pass for why
  that matters); run it on `Iterate` after every change there. `make bsfdep`
  cross-compiles for amd64, so it can run from any host.
  `bench/asmdiff.py` (2026-09-23) — lists the package functions whose machine
  code differs between two test binaries, ignoring addresses, line numbers and
  arm64's page-relative offsets. Build both with the GOARCH a change is NOT
  meant for and it proves the change left that architecture alone: that is how
  every arm64-only change of 2026-09-23 was shown to leave amd64 identical.
  `bench/perf_2026-09-23_arm64.md` — the report of that pass.
- `logfmtbench_test.go` also holds, since 2026-09-22, `Benchmark_Get` (one-key
  lookups: a shallow key, a deep one, an absent one — the package had none) and
  `Benchmark_IterateFieldShape` (records of one repeated field shape: short
  unquoted, second-word, short quoted, bare). The shape rows isolate the
  per-field chain and are where a change to it shows at full size: the BSF
  dependency of 2026-09-22 cost the unquoted row 37% and the sample line 3%.
- `testdata/sample_big.txt` — the shared ~1.4 KB benchmark line, read by both
  root (`sample2`) and bench (`sampleBig`). Keep it a single file: the
  cross-suite ratios rely on the two suites parsing identical bytes, which is
  why the former duplicated literals were replaced.
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

- `Iterate(data, func(k, v) bool) error` — the parser itself since 2026-09-22
  (it used to be an adapter over an unexported `iterate` that also reported the
  quoted bit; see "The quoted-bit protocol"). Calls back per pair,
  `k`/`v` alias `data` (bare key → shared `trueSlice`; all results read-only).
  Quoted values have quotes stripped but escapes left intact (raw). `false` from
  the callback stops. **The only function that reports errors alongside data.**
  `key=` before whitespace is an **empty value**, and the whitespace still
  separates the next token: `"key= value"` yields `("key", "")` then the bare
  key `("value", "true")`. (The doc comment claimed the opposite until
  2026-07-27 — it was never updated when 5323cb2 changed the behaviour.)
- `All(data) func(yield func(k, v []byte) bool)` — range-over-func wrapper over
  `Iterate`. Deliberately the bare func type, **not** `iter.Seq2`: that keeps the
  `iter` import (and the go 1.23 floor) out of this module, while consumers on
  1.23+ can still `for k, v := range`. Proven by `TestAllRangeOverFunc` in the
  bench module, which declares 1.23 for exactly that purpose.
- `Get(data, key) ([]byte, bool)` — raw value, aliases `data`, zero-copy, capped.
  A bare key yields the shared `trueSlice`, the one result that does not alias
  `data`.
- `GetQuoted(data, key) ([]byte, bool, bool)` — `Get` plus **whether the value
  was double-quoted**, which is the bit that decides whether unescaping it is
  correct at all. Added 2026-08-08 with the `iterate` split; it is the
  zero-copy-and-correct decode path (`GetQuoted` + `NeedsUnescape` +
  `AppendUnescape`), where plain `Get` + `NeedsUnescape` is the recipe that
  silently corrupted unquoted values.
- `GetMany(data, keys, buf) [][]byte` — multi-key single pass, raw aliasing
  capped values, **`nil` for absent** (present-but-empty is a non-nil
  zero-length slice — distinct from absent), reusable outer `buf`, early-stop.
- `AppendValue(dst, data, key) ([]byte, bool)` — unescaped, **always appends**;
  never aliases `data`. Absent key returns `dst` untouched and false. Decodes
  **only quoted values**: an unquoted `path=C:\Users\bob` is copied through
  byte for byte.
- `Validate(data) error` — full parse for callers who need the error the
  lookups structurally cannot give them.
- `SplitRecord(data) (record, rest)` — record framing (see limits below);
  trims a trailing `\r`, caps `record`.
- `IsBareKey(val)` — identity test against `trueSlice`, the only way to tell
  `debug` from `debug=true`.
- **Duplicate keys resolve identically in all four lookups: first non-empty
  occurrence wins; an empty value only if no non-empty one exists.** Guarded by
  `FuzzGetManyAgainstRef`.
- `AppendUnescape(dst, raw)` / `NeedsUnescape(raw)` — decode `\n \r \t` and
  JSON-style `\uXXXX` incl. surrogate pairs (go-logfmt writes control chars as
  `\u00XX`, so this is required for round-trip interop); other escapes pass
  through, and malformed `\u` stays verbatim. `NeedsUnescape` is a single
  `IndexByte('\\')` so callers skip the decode when unnecessary — keep it a
  single expression so it stays inlinable (a SWAR helper here measurably
  regressed). Inside `AppendUnescape` the *first* backslash is still found by
  `IndexByte`; after each decoded escape it probes the next `unescWindow` (4)
  words inline with `hasBackslash` before calling `IndexByte` again (2026-08-17:
  `Unescape` −8.5%, the JSON `msg=` value −40%; see "Escape-dense values").
  `unescWindow` is its own constant, deliberately not the parser's `escClean`:
  re-measured on `Benchmark_UnescapeEscaped`, eight words costs the 128-byte-gap
  row 18.5% and gains nothing anywhere (geomean +2.0%), because this scan only
  looks for the next escape where the parser's consumes them as it goes.
  **Unlike `escClean`, `unescWindow` = 4 re-measured the same on amd64**
  (2026-08-17): 2 costs the 32-byte-gap row 18.8% and `UnescapeJSONMsg` 4.3%, 8
  costs the 128-byte-gap row 28.5%. Both arches agree, so a change here needs
  both before it lands. **On arm64 since 2026-09-23 a destination with room for
  all of `raw` takes `unescapeInto` instead** (`unescape_spare.go`), which keeps
  `unescWindow` for no one: its probe is `spareWindow` = 8 words, cheaper per
  word (constants in registers) and skipped after a run of 64+ bytes until a
  short run turns up — the adaptive part is what made 8 words pay (a fixed 8
  measured −25% at 48-byte gaps and +16% at 128; adaptive, −25% and −7%). The
  append-as-you-go loop above still serves arm64 buffers with less room than
  `raw`, and every other architecture, unchanged.
  **The probe loop carries a redundant-looking `s >= 0` and it is load-bearing**
  (2026-08-17): it is what removes the bounds check on the `Uint64` load. The
  `uint(i) < uint(n)` / `i <= n-8` recipe that clears `iterate`'s two SWAR loads
  does *not* reach here, because `s` is not this loop's induction variable
  (`quiet` is) so the prove pass never learns `s` is non-negative — this was the
  one checked SWAR load left in the package. Without it the load pays a `LEA`
  and two compare-and-branch pairs into `panicBounds` per probed word: 28
  instructions in that region against 23. Worth −14% at 8-byte gaps, −13% on
  `UnescapeJSONMsg`, −15% on `UnescapeEscaped/esc=128`; costs +3–4% at 128-byte
  gaps, where all four probe words are wasted anyway and the extra compare has
  nothing to amortise against. Three other spellings were tried — a `uint` outer
  head, `uint(s) <= uint(n-8)` with the `n>=8` guard hoisted, and a precomputed
  limit with `s` as the induction variable — and **none of them eliminates the
  check**. Verify with `-d=ssa/check_bce/debug=1` before touching that line.
- `hex4` / `decodeUnicodeEscape` — **a 256-byte `int8` table since 2026-08-17**,
  with the sign bit of `h0|h1|h2|h3` doing the validity test in one branch. It
  was a chain of data-dependent range compares (up to six branches a digit, 24
  an escape) and had stayed that way because **nothing in the suite executed it**
  — see the two new benchmarks in Layout. Worth −26% on `UnescapeUnicode` and
  −21% on `AppendValueUnicode`, fuzz-clean at 40 s. Third win for the
  table-beats-arithmetic pattern, after `spaceTable` twice; the branches here are
  genuinely unpredictable, so it is the strongest case of the three.
- `ParseTime(ts []byte)` — `[]byte` like everything else. A caller holding a
  `[]byte` pays the same allocs on the named-zone layout either way (measured
  both sides); the old `string` benchmark only looked cheaper because it fed a
  compile-time constant. Since `c04fbed` the counts are **4** for a zone
  abbreviation the runtime cannot resolve (the fabricated `Location`) and **5**
  for a value matching no layout at all (the discarded `*ParseError`); every
  other accepted shape is 0. The unresolvable-zone case is host- AND
  date-dependent, since `time.Parse` reuses `Local` only when the abbreviation
  matches Local's at that instant.

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
  is still the faster pattern — but it is `GetQuoted` + `NeedsUnescape`, not
  `Get` + `NeedsUnescape`: without the quoted bit the decode is wrong on any
  unquoted value containing a backslash.
- **Escapes are a property of the QUOTED form, not of the value.** Anything that
  unescapes has to know how the value was written, so the parser reports it
  rather than letting callers guess. This is why `iterate` carries a third
  callback argument and why `GetQuoted` exists.
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
  `Iterate` does **not** cap to the value's own length (−4.5%, see below), so
  callback values still carry capacity into the input — but only as far as the
  end of the record: `Iterate` now opens with
  `data = data[:len(data):len(data)]`, which it does for bounds-check
  elimination and which incidentally stops a callback's `append` reaching past
  the record. A strict tightening, never a loosening. Pinned by
  `Test_Unit_Lookups_CapValues`, which also guards the thing capping could have
  broken: slicing keeps a present-but-empty value non-nil, which is how absence
  stays distinguishable.
- **The bare-key `trueSlice` is a shared global** — mutating it is process-wide.
- **A caller's stack buffer is forced to the heap** (found 2026-08-17). `iterate`
  hands `data`'s sub-slices to an opaque `fn`, so escape analysis can only say
  `leaking param: data`, and that verdict propagates out through **every**
  exported entry point — `Iterate`, `Get`, `GetQuoted`, `GetMany`, `AppendValue`,
  `Validate`, `All`. A caller that assembles a record in a `var buf [64]byte` and
  passes it in pays **1 alloc / 64 B per call** (38.9 ns against 14.2 ns for the
  identical caller that scans the buffer itself; `-gcflags=-m` says
  `moved to heap: buf`). The package's own work really is allocation-free, so
  `Test_Unit_HotPath_Allocs` is right not to catch this — it measures
  package-level inputs — but "allocates nothing" is a claim about the package,
  not about the caller, and `doc.go` should say so: reuse a heap buffer, not a
  stack array. Also worth noting as **new evidence for a rejected item**: the
  callback-free lookup loop was turned down at "only ~4.5%, and it duplicated the
  parser", and an allocation per call was not on that ledger.
- **The `escGap` entry decision is one-way** (priced 2026-08-17, deliberately not
  fixed). `escGap` is asked once, at the value's first escaped quote. `escClean`
  can take a value *off* the walk when its escapes thin out, but
  `scanQuotedSparse` has no way back *on*, so a value that starts sparse and
  turns dense is scanned by one `IndexByte` call per escape to the closing quote.
  The shape is ordinary — a prose prefix longer than `escGap` then embedded JSON,
  `msg="failed to process request: {\"id\":...}"` — and it costs **60%**, as a
  hard cliff exactly at `escGap` rather than a gradient (`Benchmark_
  IteratePrefixJSON` pins it: flat either side, a ~55% step between `prefix=032`
  and `prefix=064`). A prototype letting `scanQuotedSparse` report "these escapes
  turned dense, resume the walk here" is fuzz-clean and measures **−32%** on that
  shape and **+3–4%** on values whose escapes really are 64–256 B apart — a trade
  that depends on the input distribution, not on the stopwatch, which is why it
  was left out. **If it is ever attempted: the upgrade threshold must sit
  strictly below the distance `escClean` gives up at, or the two scans oscillate
  handing back to each other — setting it equal to `escGap` measured +26% at a
  48-byte gap.** Raising or deleting `escGap` are the cheap alternatives and both
  measure worse: deleting it fixes the cliff but costs one wasted 40-byte probe
  on *every* sparse value (+6–9% at 64–128 B gaps).
  **Closed on arm64, 2026-09-23** (`scanQuotedSparse` in `scan_arm64.go`), and
  the threshold rule above turned out to be necessary but not sufficient. With
  the upgrade at 48 against a give-up at 64 the prefix shape measured −61%, but
  a value handed back after EVERY short gap measured **+28–34%** on escapes
  alternating a short gap and a long one (8/80, 40/80, 40/120, 20/70 bytes) —
  JSON whose string values are long: each long gap made the walk give up after
  64 wasted bytes and the next short gap brought it straight back. What landed
  hands a value back **once, and only if the walk declined it on arrival** (the
  sparse scan tells the two cases apart by the byte before its start: an escaped
  quote is left there only by a decline, never by a clean run). Prefix shape
  −64%, alternating gaps +0.3–0.9%, 128/256-byte gaps ~/+1.5%. Still open on
  amd64, on the numbers above; if it is ported, port the once-only rule with it
  and re-measure the alternating rows, which exist for exactly this.
- **Keys are never quoted**: `"a b"=c` → bare key `"a`, then `b"`=c. Quoting is
  position-dependent (value position only) — the same property that defeats the
  SIMD substring search below.
- **Empty keys and literal `=` in unquoted values are accepted** (2026-08-06):
  `=v` → `(""="v")` and `a==b` → `("a"="=b")`, where go-logfmt rejects both
  with "unexpected '='". Doc'd in doc.go's Leniency list, pinned by a unit
  case; `Get(data, "")` can genuinely match.
- **`ParseTime` epochs are exactly 10 digits** → 1970-01-01 .. 2286-11-20, no
  negatives, and ms/µs epochs (13/16 digits) are rejected by design. Ten is a
  digit COUNT, so a zero-padded `0000000000` is accepted and is the epoch
  itself; the lower bound is 1970, not the 2001-09-09 that unpadded ten-digit
  values start at. Don't "fix" this by rejecting a leading zero — `0999999999`
  is a legitimate epoch.
- Statement coverage is 99.7%, and the one uncovered statement is
  `parseUnixTS`'s defensive ParseInt error guard, which is unreachable (10
  digits cannot overflow int64). The value-scan SWAR control-byte break —
  formerly the other uncovered line, reachable on CI by no seed — is pinned by
  a fuzz seed and a unit case since 2026-08-06; the wide key loop's bare-key
  dispatch, which briefly held that title during the 2026-08-22 pass (every
  older bare-key seed sat within 15 bytes of the record's end and so reached
  keyBare through the narrow loop), is pinned by a seed the same way. All
  three differential fuzzers pass clean after the 2026-08-22 pass
  (`FuzzIterateAgainstRef` 90 s, `FuzzGetManyAgainstRef` 60 s,
  `FuzzAppendUnescapeAgainstRef` 30 s, no new failures), and again after the
  2026-09-01 pass at the same durations. That pass gave each of the three
  places a value's stop is now verified — the view's first word, its second,
  and the value loop — its own control-byte seed, because the 2026-08-06 seed
  reached only the first once the other two had verifies of their own;
  coverage is 99.7% with the same single unreachable statement.

## Current benchmarks (Ryzen 7 8840HS, amd64)

Quoted as **before → after from one interleaved run**, not as standalone
absolutes. This machine's power state moves the absolute numbers by ~30% between
sessions (`Iterate` measured 270 ns, 307 ns and 358 ns for the *same* code
within one afternoon), so a bare ns/op figure here ages into a lie and invites
exactly the stale-baseline comparison the methodology section forbids. The
ratios are the portable part.

2026-07-27 pass, n=10 interleaved pinned rounds, control (two identical trees)
`~` on every row at +0.13% geomean:

| Benchmark | before | after | Δ | allocs |
|---|---:|---:|---:|---:|
| `Iterate` (sample2, 1.4 KB real line) | 358.1 ns | 308.8 ns | **−13.8%** | 0 |
| `ParseAll_Big_Mine` (same line, bench/) | 359.7 ns | 309.9 ns | **−13.8%** | 0 |
| `ParseAll_Typical_Mine` (130 B line) | 72.4 ns | 65.9 ns | −9.0% | 0 |
| `LevelTS` logfmt (vs ~8900 ns regex) | 57.1 ns | 53.5 ns | −6.3% | 0 |
| `Extract_Mine` / `GetMany` (early-stop) | 71.2 ns | 67.1 ns | −5.7% | 0 |
| `ParseEscaped_Mine` | 171.2 ns | 164.7 ns | −3.9% | 0 |
| `DecodeKeyval` (10k short-field rows) | 521.9 µs | 517.6 µs | −0.8% | 0 |
| `Unescape` (untouched by the pass) | 21.7 ns | 22.0 ns | `~` | 0 |

Everything on the hot path is **zero-allocation**. `Iterate` has come down from
681 ns over the optimization history on comparable hardware; the 2026-07-27 pass
took −13.8% off what remained. Geomean −8.2% across the `bench/` suite, big-line
throughput +16.0%. Every benchmark improved or stayed flat; none regressed —
including `DecodeKeyval`, which is the *worst* case for this pass (its rows end
`x=sf   \n`, a four-byte separator run, the one shape the skip-loop removal
pessimises) and which still came out ahead.

CI-generated tables live in `bench/pkg_results_<arch>.md` and
`bench/results_<arch>.md`, each stamped with the CPU and Go version that
produced it. **The README no longer copies any figure out of them** — it links
them and quotes only order-of-magnitude ratios in prose. It used to carry a full
table, which silently went two generations stale and disagreed with the file it
linked (README said EPYC 7763 / 444 ns; the table said EPYC 9V74 / 386 ns), and
claimed "regenerated by CI" although nothing in `bench.yml` ever writes README.
Don't reintroduce that pattern; if the numbers must appear in two places, have
the renderers splice into marker-delimited blocks so the claim is true.

Table currency, as of 2026-09-23: the tables were regenerated right after
the 2026-09-22 pass (`3cb3ae9`, stamped 2026-09-22T20:31Z), so the next
paragraph's "stale for the 2026-09-22 pass" was out of date within hours — a
fifth time. They are stale for the 2026-09-23 arm64 changes (the escape and
decode rows of `pkg_results_arm64.md` above all) until `bench.yml` is
dispatched; the amd64 tables are not, since amd64's machine code did not change.

Table currency, as of 2026-09-22: `git log -- bench/` shows the tables
regenerated after every pass through `3e58443` (`bb04b30`, `cf5b672`,
`89e9b0a`), so the paragraph below was already out of date when this one was
written — which is the paragraph's own warning, a fourth time. They are stale
for the 2026-09-22 pass until `bench.yml` is dispatched.

Table currency, as of 2026-09-09: the committed tables are **stale for the
2026-09-01 and 2026-09-09 passes** (nothing dispatched since 2026-08-22), and
they are also a toolchain behind — this machine is on go1.27.1 now. The escape
rows are where the gap is largest: `esc=500` and `IterateJSONMsg` moved 49% and
26% on 2026-09-09 alone. The paragraph below describes their state before all
of that. As of 2026-08-22 (second revision that day): the committed
tables were **current through `5ff3f57`, the seeded value scan** — the
maintainer dispatched `bench.yml` minutes after that push and `4a289b9`
committed the result, whose diff supplied the cross-arch verdict recorded in
the pass section. (THREE earlier versions of this paragraph have now each
declared a staleness that a regeneration had already fixed or fixed within
minutes — the version this one replaces was stale before CI finished. Check
the `generated` stamp in the file against `git log -- bench/` rather than
trusting this line.) The CI arm64 numbers
reproduce this machine's to within 0.1% (`391.3 / 81.4 / 72.1` at `90c6c75`,
`381.9 / 80.5 / 69.8` at `b28092c`, against local `391.4 / 81.3 / 72.0` and
`382.3 / 80.4 / 69.8`), so the arm64 runner is this machine class and ratios
measured here travel to it.

**The amd64 tables caught the 32-byte-window regression independently, and then
recorded the correction fixing it.** EPYC 7763, `90c6c75` → `b28092c`:
`Iterate` 414.2 → 394.0 (−4.9%), `LevelTS` 75.5 → 72.6 (−3.8%),
`DecodeKeyval` 691.9 → 681.5 µs (−1.5%), `esc=128` 1129 → 671 ns (−40.5%),
`esc=500` 4410 → 2549 ns (−42.2%) — but **`GetMany` 89.2 → 95.3 (+6.8%)** and
**`esc=8` 92.0 → 154.8 (+68.3%)**, the 32-byte-window failure, worse there than
the +7.3% / +27% it cost on arm64.

**The correction (`b28092c` → `6a79ce6`) recovered both on x86**, which settles
a question this file used to leave open ("nobody has measured the correction on
x86 … say 'should' until `bench.yml` is dispatched again" — it had been, in
`00d92ad`, and the data was sitting in the repo). Same EPYC 7763 tables:
**`GetMany` 95.3 → 87.5 (−8.2%)** and **`esc=8` 154.8 → 94.8 (−38.8%)**, plus
`LevelTS` 72.6 → 66.3 (−8.7%), `esc=32` 294.0 → 261.9 (−10.9%), `esc=128`
671.2 → 630.5 (−6.1%), `esc=500` 2549 → 2372 (−6.9%), `IterateJSONMsg` 196.2 →
186.1 (−5.1%), `Iterate` 394.0 → 392.4 (−0.4%). The prediction held; the
mechanism was indeed identical. Diffing two committed tables is the cheapest
way to answer this class of question — check for it before re-measuring.

Treat these tables as indicative, never as A/B results: they are `-count=1`
single runs with no interleaving and no control, and the same diff moves
`ParseTime_Unix` 75.7 → 91.0 (+20%) on code that this pass never touched. A row
only means something here when it is large and has a mechanism, as the two
above do. Note `bench.yml` is `workflow_dispatch` only, so nothing refreshes
them automatically; run `make bench-md` deliberately (dispatch the workflow).
Don't hand-edit them with laptop figures.

2026-07-27 **evening** pass (same machine, faster power state — do not compare
absolutes across the two tables), n=8 interleaved pinned 1 s rounds, control
clean: the uint/n-8 loop bounds and the guard-free backslash walk landed
together. Iterate 231.2 → 228.2 ns (−1.3%), GetMany 50.6 → 48.9 ns (−3.4%),
Extract 51.0 → 49.3 ns (−3.4%), ParseEscaped 124.3 → 120.2 ns (−3.3%), LevelTS
−1.4%, ParseAll_Typical −0.6%, ParseAll_Big/Unescape `~`, **DecodeKeyval 391.0
→ 395.6 µs (+1.2%)** — the accepted trade (see the optimization-notes bullet).
Geomean −1.0% (root) / −2.0% (bench). CI tables are stale for this pass too.

### 2026-08-17 pass — Neoverse N2 (Azure Cobalt-class arm64, 2 vCPU), Go 1.26.5

A different machine from everything above: **do not compare its absolutes with
the Ryzen tables**, only its ratios. It is, however, a far better benchmarking
platform — a quiet VM, no turbo/power-state drift, run-to-run variance ±0%,
and an A/A control of two identical trees came back `~` on every row at
+0.03% geomean. It is also the same microarchitecture as GitHub's
`ubuntu-24.04-arm` runners, so the committed `*_arm64.md` tables should track
these ratios once `bench.yml` is dispatched.

Before = `90c6c75`, after = the pass as it finally stands (two commits:
`b28092c`, then the escape-scan correction below); n=8 pinned (`taskset -c 1`)
interleaved 1 s rounds, order rotated per round; benchstat p=0.000 on every
non-`~` row. The middle column is `b28092c`, kept because the difference
between the two is the lesson:

| Benchmark | `90c6c75` | `b28092c` | final | Δ overall | allocs |
|---|---:|---:|---:|---:|---:|
| `Iterate` (sample2, 1.4 KB, 29 fields) | 391.4 ns | 382.3 ns | 377.0 ns | −3.7% | 0 |
| `ParseAll_Big_Mine` (bench/) | 391.6 ns | 383.1 ns | 377.8 ns | −3.5% | 0 |
| `ParseAll_Typical_Mine` (130 B) | 85.1 ns | 83.4 ns | 82.9 ns | −2.6% | 0 |
| `LevelTS` logfmt | 72.0 ns | 69.8 ns | 64.8 ns | **−10.0%** | 0 |
| `GetMany` | 81.3 ns | 80.4 ns | 74.5 ns | **−8.4%** | 0 |
| `Extract_Mine` (bench/) | 81.5 ns | 80.2 ns | 75.7 ns | **−7.1%** | 0 |
| `DecodeKeyval` (10k short rows) | 726.1 µs | 719.7 µs | 725.2 µs | `~` | 0 |
| `ParseEscaped_Mine` (bench/) | 214.1 ns | 193.3 ns | 192.8 ns | **−9.9%** | 0 |
| `Unescape` | 28.3 ns | 26.0 ns | 25.6 ns | −9.5% | 0 |
| `IterateJSONMsg` (JSON in `msg=`, 28 escapes) | 301.5 ns | 171.0 ns | 168.2 ns | **−44.2%** | 0 |
| `UnescapeJSONMsg` (that value, decoded) | 318.3 ns | 191.6 ns | 189.7 ns | **−40.4%** | 0 |
| `IterateEscaped/esc=0` (1 KB clean) | 36.2 ns | 35.0 ns | 35.6 ns | −1.7% | 0 |
| `IterateEscaped/esc=8` (128 B gaps) | 107.8 ns | 137.7 ns | 108.7 ns | `~` | 0 |
| `IterateEscaped/esc=32` (32 B gaps) | 333.6 ns | 250.2 ns | 220.5 ns | **−33.9%** | 0 |
| `IterateEscaped/esc=128` (8 B gaps) | 1194 ns | 569.7 ns | 557.3 ns | **−53.3%** | 0 |
| `IterateEscaped/esc=500` (2 B gaps) | 4.72 µs | 2.17 µs | 2.11 µs | **−55.3%** | 0 |
| `UnescapeEscaped/esc=500` (decode) | — | — | 2.98 µs | new | 0 |

Four changes landed: the quoted-bit protocol (above), the two-scan split for
escape-dense quoted values and the probe in `AppendUnescape` (see the
optimization notes), and the `if m := hasKeyStop(w); m != 0` spelling (drops a
real `NOOP` per SWAR iteration on both arches; measured neutral, kept for the
cleaner loop). **No row regressed overall, and `esc=8` is back to parity.**

The two-commit story is the point. `b28092c` shipped the escape scan as one
always-entered 32-byte probe, tuned on the synthetic sweep, and it cost
`GetMany` 7.3% and `LevelTS` 7.2% against what was available — masked in that
commit's own A/B because the quoted-bit protocol landed alongside and more than
covered it. The correction (a `escGap`/`escClean` split, ported from the
maintainer's open branch `perf/escape-scan-and-adapter`, PR #1) recovered it:
vs `b28092c`, GetMany −7.3%, LevelTS −7.2%, `esc=8` −21.1%, `esc=32` −11.9%,
`esc=128` −2.2%, `esc=500` −2.8%, Iterate −1.4%, `DecodeKeyval` +0.8%,
`esc=0` +1.5%, root geomean −4.8% / bench geomean −2.0%. The finished tree also
beats that branch itself (Unescape −10.4%, GetMany −1.7%, Iterate −0.5%,
DecodeKeyval −0.7%, escape rows at parity), because it keeps this pass's
`AppendUnescape` probe and set-only protocol on top of the branch's gating.

All three differential fuzzers pass clean (90 s / 60 s / 60 s); coverage 99.6%
(the one uncovered statement is still `parseUnixTS`'s unreachable guard), and
both new scan helpers are at 100%.

What `perf stat` says about this core, for whoever optimizes next (the
hypervisor exposes only the generic events; the N2 IMPDEF ones read 0):
`Iterate` runs at **IPC 4.41 with 0.00% branch mispredictions** (111 K misses
in 7.8 G branches), ~200 instructions and ~46 cycles per field at 3.4 GHz.
Cutting 14% of the instructions (holding the two non-bitmask SWAR constants in
registers, see Rejected) moved cycles by only 0.4% and pushed IPC to 3.81 with
backend stalls doubling — the removed instructions were free filler. Adding
two dependent cycles to each scan's `load→mask→tz→i` chain (a semantically
neutral `m|m<<1` before the `TrailingZeros64`) cost +3.9%, i.e. about 40% of
the added latency showed through. So the per-field cost is roughly half
dependency chain (key hit ~13 cycles, value hit ~11, plus a store→load
forward of the spilled `i` around every callback, since Go's ABI has no
callee-saved registers) and half throughput limits that instruction count
alone does not move. Neither lever is cheap any more.

### 2026-08-17 amd64 review — Ryzen 7 8840HS, Go 1.26.5, GOAMD64=v1

A review pass on x86, aimed deliberately at what the arm64 pass could not see:
constants tuned on one arch, paths with no benchmark, and the codegen invariants
this file says to keep re-checking. Three changes landed — `escClean` 8 → 5, the
`hex4` table, and `AppendUnescape`'s bounds-check hint — each written up at its
own site above. Measured against `00d92ad`, n=6–8 pinned interleaved rounds,
**A/A control geomean −0.03% with every row `~`**:

| Benchmark | before | after | Δ |
|---|---:|---:|---:|
| `IterateEscapedGap/gap=048` (new) | 144.5 ns | 130.1 ns | **−10.0%** |
| `UnescapeUnicode` (new) | 71.8 ns | 58.3 ns | **−18.8%** |
| `AppendValueUnicode` (new) | 91.5 ns | 77.4 ns | **−15.4%** |
| `UnescapeEscaped/esc=128` | 584.8 ns | 495.7 ns | **−15.3%** |
| `UnescapeEscaped/esc=500` | 2.120 µs | 1.806 µs | **−14.8%** |
| `UnescapeJSONMsg` | 137.6 ns | 118.0 ns | −14.2% |
| `Unescape` | 17.86 ns | 17.29 ns | −3.2% |
| `Iterate` / `GetMany` / `LevelTS` / `IterateJSONMsg` | | | `~` |
| `UnescapeEscaped/esc=8` | 86.34 ns | 90.19 ns | **+4.5%** |
| `DecodeKeyval` | 405.3 µs | 410.9 µs | +1.4% |

Geomean −4.5% on the root suite; the `bench/` module's four `_Mine` rows all
`~`. Only the `esc=8` regression has a mechanism (the probe's added compare on
the row where all four probe words are wasted); the other two are layout. All
three differential fuzzers clean (45 s / 45 s / 30 s), coverage still 99.6%.

**Environment caveat, and it is a real one.** This was measured on a working
laptop with a browser at ~60% CPU and load average ~2.6 — worse conditions than
the quiet N2 VM. Pinning plus interleaving plus the clean A/A control keeps the
large effects honest, and every number quoted above is either ≥8% or has a
mechanism. But **sub-3% rows from this pass are not resolved**, and the
code-layout sensitivity the arm64 pass measured at ±2% on the `*Escaped/*` rows
measured **up to ±8% here** — a constant change that alters nothing on a row's
code path moved it 8.5% once. Treat any small delta on those rows as noise
unless a padding control says otherwise.

**Worst-case shapes, characterised (no bugs found, numbers worth keeping).**
`Iterate` over 1 KB of each: backslash runs of 1/7/31/127 before quotes →
1422/1407/1371/286 ns, so **the parity walk has no quadratic case**, which is
the one this file's design most invites. One long unquoted value 15.3 GB/s; one
long key 9.9 GB/s (`hasKeyStop`'s extra ops); all-whitespace 2.2 GB/s (the
scalar drain, 1 B/iter, linear); `Get` on the sample's last field 233.8 ns, i.e.
≈ a full `Iterate`, as the cost model predicts. `GetMany` at 1/2/5/10/20 keys →
17.5/25.4/63.8/138.9/557.7 ns, so the doc comment's "ahead up to roughly ten
keys, ~505 ns at 20" still holds.
**The separator-run trade, finally priced:** 5.54 ns/field at one separator,
**8.32 at two (+50%)**, 9.04 at four, 12.32 at sixteen. The expensive step is
1 → 2, and it is not the drain loop — it is the wasted field iteration that
detects the empty key; each further byte is only ~0.5 ns. Left alone on purpose
(the cost is structural to having no whitespace-skip loop, and reintroducing a
skip is what measured worse), but column-aligned or double-spaced emitter output
is one format string away, so the number belongs here rather than the assertion
that "no real emitter writes one".

**Codegen invariants re-checked on go1.26.5/amd64, all still holding:** no
inline-mark `NOOP`s in `iterate`, `AppendUnescape` or `scanQuotedEscapeDense`;
both of `iterate`'s SWAR loads bounds-check free; all five SWAR helpers still
inline (cost 8–26 against budget 80) and `iterate` still far from inlinable at
747; all 14 alloc-free entry points still alloc-free.

### 2026-08-22 pass — the lookahead-seeded value scan lands (Ryzen 8840HS, amd64, Go 1.26.5)

The item the 2026-08-17 list called "the one structural idea left with real
upside", attempted by its own protocol — prototype under
`FuzzIterateAgainstRef` first, then n=8 pinned (`taskset -c 6`) interleaved
1 s rounds with order rotated per round — on a quiet machine this time (load
~0.6, no browser). The in-session A/A control came back `~` on every row at
+0.27% geomean. Mechanism and the load-bearing spellings are written up in
"The key scan seeds the value scan" under the optimization notes; the numbers,
before → after from the deciding series:

| Benchmark | before | after | Δ | allocs |
|---|---:|---:|---:|---:|
| `Iterate` (sample2, 1.4 KB, 29 fields) | 230.1 ns | 199.9 ns | **−13.1%** | 0 |
| `ParseAll_Big_Mine` (bench/) | 232.4 ns | 201.2 ns | **−13.5%** | 0 |
| `ParseAll_Typical_Mine` (130 B) | 50.53 ns | 40.58 ns | **−19.7%** | 0 |
| `DecodeKeyval` (10k short rows) | 402.8 µs | 335.4 µs | **−16.7%** | 0 |
| `LevelTS` logfmt | 39.15 ns | 38.03 ns | −2.9% | 0 |
| `IterateJSONMsg` | 110.0 ns | 106.9 ns | −2.8% | 0 |
| `GetMany` | 49.01 ns | 48.72 ns | `~` | 0 |
| `Extract_Mine` (bench/) | 49.08 ns | 48.80 ns | `~` | 0 |
| `ParseEscaped_Mine` (bench/) | 126.4 ns | 125.5 ns | `~` | 0 |
| `IterateEscaped/*`, `IterateEscapedGap/*`, `IteratePrefixJSON/*` | | | `~` | 0 |

No row regressed. The escape sweeps' geomean moved −0.01% and their largest
single movement (`esc=32` +1.2%, p=0.022) is inside this machine's documented
±8% layout band for those rows, with no mechanism — the quoted path gained
only one compare. The `escGap` cliff is intact (`IteratePrefixJSON` flat on
both sides, the step still between 032 and 064).

**It took two variants, and the difference is the finding worth keeping.** The
first loaded both words of the view eagerly in the loop body, reasoning the
second word's load latency had to be in flight before the `=` hit needed its
bytes. It won everything big (Iterate −10.1%, DecodeKeyval −15.5%) but cost
**GetMany +2.5% and Extract +2.0%** (p≤0.01): the early-stop lookups pay the
loop's extra load on every key word and collect nothing per value — the same
"+2 loop instructions = +2.3%" arithmetic the N2 pass measured. Moving the
load inside the `=` branch, after the first word's value mask has already come
back empty, recovered both AND beat the eager form on every other row too
(Iterate a further −3.5%, LevelTS −2.9%, GetMany −2.8%, DecodeKeyval −0.9%):
the out-of-order window hides an L1 load issued at the hit just fine, so the
eager preload's only real effect was its cost. Lazy is not the compromise
here; it dominates.

**Mechanism, confirmed with counters** (perf stat, pinned, `Iterate`): the
landed form retires **+4.5% instructions per parse in −14% cycles** — IPC
4.62 → 5.64, branch misses ~0.0007% on both sides. Every prior pass cut
instructions; this is the first to shorten the per-field dependency chain
instead, which the 2026-08-17 counter work identified as half the remaining
cost, and Zen 4's width absorbs the added parallel work for free. **That is
also the portability caveat: unmeasured on arm64.** The N2 ran the old shape
at IPC 4.41 with less width to spare, so do not quote these ratios for arm64
until the series is rerun there; the next `bench.yml` dispatch's arm64 table
gives a first indication either way. **That indication arrived within the
hour** (`4a289b9`, dispatched right after the push): the arm64 table moved
IterateOur 377.1 → 332.0 ns (−12.0%), DecodeKeyval 724.3 → 614.3 µs (−15.2%),
LevelTS 64.9 → 61.5 (−5.2%), with GetMany and the escape rows `~` — the same
shape as amd64, so the width concern did not materialise; the N2 swallows the
extra parallel work too. The EPYC table agrees (IterateOur −9.5%, DecodeKeyval
−14.8%). These are `-count=1` indicative runs, not A/B — but every moved row
is large with a known mechanism, which is the one condition under which this
file trusts a table diff. A pinned N2 series is still the proper confirmation
if these ratios are ever quoted as arm64 results.

Codegen state after the pass: `iterate` has FOUR SWAR loads now (wide-key
pair halves, narrow-key word, value word) — all bounds-check free, the two
new ones by the spellings in the optimization bullet; 0 inline-mark `NOOP`s;
`iterate` grew from 429 to 531 objdump lines, which is the layout risk the
escape sweeps were re-measured to price (they came back flat). All three
differential fuzzers clean on the final form (90 s / 60 s / 30 s); coverage
99.7% — ABOVE the pre-pass 99.6%, because the seed batch added for the view
boundaries also pinned the wide-loop bare-key dispatch that turned out to
have no CI-visible exercise at all.

### 2026-09-01 pass — the scan is bound by the four integer ALUs (Ryzen 8840HS, amd64, Go 1.26.5)

The pass that finally measured WHAT the per-field floor is, after four passes
of guessing at it. Read the mechanism paragraph before optimizing this parser
again on x86: it invalidates the cost intuition every earlier section here was
written with, including the one this pass started from.

Before = `bae3a0e` (HEAD), after = the working tree as it stands; n=8 pinned
(`taskset -c 10`) interleaved 1 s rounds, order rotated per round, nothing
else running (see the harness lessons below for why that clause is there);
the A/A control at the start of the session was `~` on every row at +0.8%
geomean with n=4. The `Δ` column is from that final series; every non-`~` row
is p≤0.01.

| Benchmark | before | after | Δ | allocs |
|---|---:|---:|---:|---:|
| `Iterate` (sample2, 1.4 KB, 29 fields) | 213.1 ns | 193.8 ns | **−9.0%** | 0 |
| `ParseAll_Big_Mine` (bench/) | 214.9 ns | 194.7 ns | **−9.4%** | 0 |
| `ParseAll_Typical_Mine` (130 B) | 43.21 ns | 41.15 ns | −4.8% | 0 |
| `DecodeKeyval` (10k short rows) | 358.7 µs | 345.5 µs | −3.7% | 0 |
| `GetMany` | 51.98 ns | 49.80 ns | −4.2% | 0 |
| `LevelTS` logfmt | 40.74 ns | 39.08 ns | −4.1% | 0 |
| `IterateJSONMsg` | 111.6 ns | 104.1 ns | **−6.7%** | 0 |
| `IterateEscaped/esc=32` | 164.5 ns | 149.5 ns | **−9.1%** | 0 |
| `IterateEscaped/esc=128` | 367.4 ns | 336.4 ns | **−8.5%** | 0 |
| `IterateEscaped/esc=500` | 1.407 µs | 1.296 µs | **−7.9%** | 0 |
| `IterateEscapedGap/gap=016` / `032` / `040` | | | −4.9% / −8.9% / −6.7% | 0 |
| `IterateEscapedGap/gap=048..256`, `esc=8`, `Unescape` | | | `~` | 0 |
| `IterateEscaped/esc=0` (ONE field, 1 KB clean quoted) | 18.49 ns | 19.34 ns | **+4.6%** | 0 |
| `Extract_Mine` / `ParseEscaped_Mine` (bench/) | 52.16 / 135.0 ns | 53.42 / 137.2 ns | +2.4% / +1.7% | 0 |

Geomean −3.7% on the root suite. Two rows need reading with their mechanism:
`esc=0` is the per-call constant discussed below, and the two `bench/` rows are
**code layout in that binary, not the code**: `Extract` retires 0.4% FEWER
macro-ops per op than the previous candidate build yet takes 7% more cycles,
while the root binary's `GetMany` — the identical code on the identical bytes
— is −4.2% at p=0.000. `ParseEscaped` is the same shape (−0.7% ops, +4%
cycles). That is the layout sensitivity the 2026-08-17 review measured at up
to ±8% on this machine, now landing on a lookup row of the comparison binary
rather than on an escape row; the padding control that pins it is recorded
under the harness lessons. `GOAMD64=v3` on the finished tree: `Iterate` −4.6%,
`ParseAll_Typical` −3.5%, `LevelTS` −1.9%, geomean −2.0% over v1 (n=4) — up
from the 1.4% of 2026-08-17, because `ANDN` now also replaces a `NOT`+`AND`
pair per word in an ALU-bound loop.

**Mechanism, and how it was found.** Counters first
(`perf stat -e cycles,instructions,ex_ret_ops,de_no_dispatch_per_slot.*,
ex_no_retire.*`, pinned): `Iterate` retired ~4930 macro-ops in ~1020 cycles per
parse — 4.8 of Zen 4's 6 dispatch slots — with 19% of slots stalled on the
backend, 0.3% on the frontend, 0.0007% branch misses, every op from the op
cache, no scheduler or register-file stall worth naming. Then the cutting
experiments: three variants removed 6–10% of the instructions per field
(exact-mask ops, bounds checks, the mask dance, a NOP per word) and each landed
on the SAME ~1010 cycles, the stall share simply growing to fill the gap, while
every variant that ADDED instructions paid for them at ~1/5 cycle each. So
neither "dispatch-bound" nor "chain-bound" fits; what fits is **integer-ALU
port throughput**. A standalone copy of the bare mask loop over the sample
bytes — no branching per word, no field structure at all — runs at 3.6 cycles
per 8-byte word, and its body has ~12 ops that need one of Zen 4's four integer
ALUs: the XOR/SUB/SUB/OR/NOT/AND/AND of the masks, the loop's ADD and fused
CMP+Jcc, and **the three `MOVQ $imm64` that rematerialise the broadcast
constants every word** — those are ALU ops too. ~140 ALU ops per field at four
per cycle is the 35-cycle floor. Loads and stores go to the three AGUs, which
sit half idle; register moves are eliminated at rename; NOPs cost nothing.
`BSF` is a 1-cycle op here (micro-benchmarked, v1 `BSF`+`CMOV` chain 5 cycles
against 4 for `TZCNT`), not the 3 cycles assumed earlier.

Two chain probes complete the picture, and they are the cheapest measurement
this file has ever recorded: `m |= m<<1` before the key mask's `TrailingZeros`
(two dependent ALU ops on the hit chain, semantically neutral) cost `Iterate`
**+7.2%** — every added cycle showed; two dependent adds on `i` after the
value stop cost `Iterate` +1.3% but `ParseAll_Typical` +4.5% and `DecodeKeyval`
+3.0%. And replacing the three verify loads (`=`, quote, value stop) by the
constants they return on the sample — a timing probe, not a change — was worth
only 4% together. Long keys are ALU-bound in the loop; short fields are bound
by the post-value chain into the next field; the verify loads are neither.

**What landed, in order of worth.**
1. The three scan constants live in registers, re-read from a package `var`
   (`swarRegs`) at the top of every field. This is the "package-level vars"
   idea rejected on 2026-07-27 and parked on arm64 on 2026-08-17, and the
   difference is where the values come from: read once per call and kept
   across the callback they are spilled at entry and reloaded through the
   store buffer, which is what the first version measured (a store-forwarding
   stall on the first field of every call; `esc=0` +7%); read afresh per field
   they die at the callback, nothing crosses it, and the key loop drops from
   14 ALU ops per word to 10. The register allocator will spill INSIDE the loop the moment one more value
   is held across it — hoisting `n-8` as well as `n-16` did exactly that, and
   cost the loop a store, two reloads and a NOT on the mask chain. Keep the live
   set small; check the loop body for `(SP)` after any change here.
2. The mask tail reassociated as `& (hi &^ w)` (`hasKeyStopR`,
   `hasCtrlOrSpaceR`, `hasQuoteOrBackslashR`): the `hi &^ w` term is computed
   in parallel with the subtractions, one cycle shorter to the `TEST`. Only
   possible because `hi` is now a variable — with a constant the compiler
   canonicalises it straight back (the 2026-08-17 note). −1.8% on `Iterate`,
   measured alone.
3. The cap-zero mask on the callback's two slices folded to nothing, via facts
   the prove pass can use: its `Slicemask` folding needs a NUMERIC limit on the
   cap, and `detectSliceLenRelation` supplies one only from an ordering of the
   form `index <= len-K`. `uint(i) < uint(n)` is not that shape;
   `&& i <= n-1` in the loop condition is, compiles to nothing, and folds the
   key slice's four ALU ops; `vStart > n-1 ||` on the callback test is a real
   fused compare that folds the value slice's four. (`-d=ssa/prove/debug=1`
   prints `Proved slicemask not needed (by limit)` for each.)
4. The dense quoted scan's four constants in registers the same way
   (`scanQuotedEscapeDense` loads them once; no calls inside): −7..−9% on every
   escape-dense row and −6.9% on `IterateJSONMsg`. NOT in `AppendUnescape`,
   where the same change measured +5.5% on `UnescapeJSONMsg` — that loop has
   no registers to spare and spills instead.
5. `GetMany`'s callback rejects a field by a 64-bit mask of the keys' lengths
   before entering the compare loop: −2% on `GetMany`/`Extract`.
6. Small and free, kept: the wide loop's limit hoisted (`lim16`, −1 ALU op per
   word); the value loop's `i >= 0` told once by an enclosing `if` instead of
   per word; the seeded stop's verify inline with `i = base+1+t` as an add
   independent of `vEnd`; the seeded quoted path no longer re-tests its quote.
   The inline-mark NOP the seeded scan had left in the key loop is gone too
   (the load spelled `data[i:i+8]`, whose `i+8` CSEs with the increment and
   gives the mark a home) — and it was worth exactly nothing, because a NOP is
   not an ALU op. Do not count NOPs again.

**The one regression, and what it is not.** `IterateEscaped/esc=0` — one field,
a 1 KB clean quoted value — is +4.8%, about 1 ns per call, and it is a per-CALL
constant: ~25 more macro-ops between entry and the first callback (the
constant loads, the hoisted limit's store and reload, the loop-condition
compare), the same on every parse and amortised by the second field. It is not
the entry spill (re-reading the constants per field, which removed the spill,
left it) and not the quoted path (its code after `IndexByte` is instruction
for instruction the old one). Records of one field pay it; everything with two
or more measured ahead, including `ParseAll_Typical` at eight.

**Measured and rejected this pass**, all fuzz-clean before losing on the
stopwatch, each with the mechanism that killed it — most are variations on one
theme, *the register allocator spills whatever a shorter chain needs to hold*:
- *Verify bytes from a 16-byte view* (`pair[t]`, no bounds checks on the post-hit
  loads): neutral. The view's pointer is a two-argument LEA, which `tighten`
  never moves out of the loop header, so it costs one op per word to save two
  fused compares per field.
- *Inexact masks* (drop the `&^ w`, fix up at the verify): 2 ALU ops fewer per
  word, `~` on every ASCII row, and **DecodeKeyval +8.9%** from the one
  non-ASCII key in that benchmark (`ƒ`) taking the fix-up path on every row.
  A 2.5× cliff on unquoted non-ASCII values for nothing measurable.
- *The '=' verify as a shift* (`byte(w >> (tz&^7))`): the shift count must be
  in CL at GOAMD64=v1, and the allocator spilled `base` and `w` to make room —
  the quote test then reloaded both through the store buffer.
- *Every verify as a mask-lane test* (`hasByte(w,'=') & (m&-m)`, the byte after
  the '=' from a view shifted down a byte, space as bit 5 of the stop lane):
  the branch chains shorten from ~11 cycles to ~4, the compiler emits +25 to
  +28 macro-ops per field for it, and the three versions measured **+12%,
  +14% and +17%** on `Iterate`. The cheapest single piece — '=' is the one
  stop absent from the value mask, `m&-m&mv == 0` — needs `m` twice, spills it,
  and loses on its own.
- *An inline SWAR search for the closing quote before `IndexByte`* (8 words):
  the loop runs at 3.5 cycles per word, not one, so the wasted words on long
  values cost `esc=0` **+37%** and the 128/256-byte-gap rows +14–17%. The
  2026-07 rejection of the one-word version stands, for the same reason.
- *Preloading the next key's first word before the callback* (so the field after
  it starts from a word in a spill slot, not from an index plus a 5-cycle load):
  three shapes, one as a loop-carried variable, one peeling the first word under
  a label, one rotating the loop; each turned the word into a memory-resident
  phi that the allocator stored every iteration, +26–28% instructions, +14%.
  The idea is sound and the compiler has no shape for it.
- *Reading `swarRegs` at every use instead of once per field*: 3 loads per word,
  +4% on `Iterate` against the per-field form.

**Harness lessons, both expensive.** A fuzzer run pinned to other cores while a
series ran on core 10 still put ±70–300% on the rows it overlapped — pinning
does not isolate the boost budget or the memory system; run nothing during a
series. Two series on two cores at once contaminate each other the same way
(every row `~` at ±10–15% where the same trees measured p=0.000 alone). Cores
differ: the same binary is ~5% slower in absolute terms on core 12 than on
core 10 here, so compare within a core only. And a harness that `cd`s into a
tree copy leaves the shell there for the next command — one edit meant for the
working tree landed in a scratch copy and was found only by `grep`; use
absolute paths. **The padding control for the `bench/` binary**, run for the
two rows above: a never-called recursive function appended to
`bench_test.go`, nothing else changed, moved `Extract_Mine` from 237 to 221
cycles per op and `ParseEscaped_Mine` from 627 to 598, at the same macro-op
counts — a 7% swing from an address shift. On this machine a 2–3% delta on a
`bench/` lookup row is inside the layout band; decide on the root binary's
rows, and re-run the padding control when a `bench/` row disagrees with them.

**Portability.** Unmeasured on arm64, where the constants trick should be worth
more, not less: each 64-bit constant there is `MOVZ`+3×`MOVK`, and the
2026-08-17 N2 pass already measured the register-held form at −2.75% on
`Iterate` before the seeded scan changed the loop. The next `bench.yml`
dispatch's arm64 table is the first indication; a pinned series on the N2 box
is the confirmation. The Slicemask folding and the reassociated tail are
arch-independent.

### 2026-09-09 pass — the escape walk drains a word (Ryzen 8840HS, amd64, Go 1.27.1)

First pass measured on **go1.27.1**; every section above it is 1.26.5, and the
toolchain alone moved the absolutes (`Iterate` 193.8 → 181 ns for code that did
not change), so read only the ratios across that boundary. Quiet machine, load
~0.3, nothing else running; n=6 pinned (`taskset -c 10`) interleaved 1 s rounds
with the order rotated per round; the A/A control came back `~` on every row at
**+0.09%** geomean and run-to-run spread on the core rows is ±1%.

**Neither landed change touches `iterate`.** The parser's own field loop came
out of this pass byte for byte as it went in — the four ways that were tried
and why they all fail are the first entry under "measured and rejected" below —
and both wins are in code around it.

| Benchmark | before | after | Δ | allocs |
|---|---:|---:|---:|---:|
| `IteratePrefixJSON/prefix=032` | 59.05 ns | 40.82 ns | **−30.9%** | 0 |
| `IteratePrefixJSON/prefix=008` | 57.49 ns | 40.57 ns | **−29.4%** | 0 |
| `IterateEscaped/esc=500` (2 B gaps) | 1272 ns | 654.5 ns | **−48.6%** | 0 |
| `IterateEscaped/esc=128` (8 B gaps) | 328.3 ns | 237.2 ns | **−27.8%** | 0 |
| `IterateJSONMsg` (JSON in `msg=`) | 100.4 ns | 74.61 ns | **−25.7%** | 0 |
| `IterateEscapedGap/gap=016` | 208.2 ns | 186.2 ns | **−10.6%** | 0 |
| `GetMany` / `Extract_Mine` (bench/) | 46.26 / 46.20 ns | 44.48 / 44.22 ns | −3.9% / −4.3% | 0 |
| `ParseEscaped_Mine` (bench/) | 120.8 ns | 119.1 ns | −1.4% | 0 |
| `Iterate`, `ParseAll_Big/Typical`, `DecodeKeyval`, `Unescape` | | | `~` | 0 |
| `IterateEscaped/esc=0`, `esc=8`, `gap=064..256`, `prefix=064/160` | | | `~` | 0 |
| `LevelTS` logfmt | 36.27 ns | 36.53 ns | +0.7%, see below | 0 |
| `IterateEscapedGap/gap=048` | 109.7 ns | 112.0 ns | +2.1% | 0 |
| `IterateEscaped/esc=32` (32 B gaps) | 141.4 ns | 149.6 ns | +5.8% | 0 |
| `IterateEscapedGap/gap=032` | 140.5 ns | 150.4 ns | +7.1% | 0 |
| `IterateEscapedGap/gap=040` | 128.2 ns | 143.2 ns | **+11.7%** | 0 |

Root geomean −8.3%, `bench/` geomean −1.5%. All three differential fuzzers
clean on the final tree (`FuzzIterateAgainstRef` 60 s / 19.8 M execs,
`FuzzGetManyAgainstRef` 45 s, `FuzzAppendUnescapeAgainstRef` 30 s); coverage
99.7% with the same single unreachable statement.

**1. `scanQuotedEscapeDense` drains a word instead of re-anchoring on it.** The
walk used to handle one escape per pass — mask a word, take the lowest bit,
check the byte, step over the pair, load a word starting at the new position —
so every escape paid a full load→mask→find→step chain, about fourteen cycles
with nothing to overlap it. At the densities this scan exists for a word holds
several escapes, so it now takes the mask apart lane by lane before loading
anything else, and the word loads become independent of each other. Two things
the old shape got for free are paid for in the drain, both written up at the
function: a lane above a true match can be **spurious** (the borrow described
on `hasQuoteOrBackslash`), which the old walk never read because it only ever
read the lowest bit, so the byte is now checked for `\` as well as `"` and an
impostor is cleared; and a backslash in **lane 7** escapes a byte the word does
not contain, so that case leaves the word and resumes at `base+9`.

**2. `GetMany`'s per-call setup is one loop, not two plus a runtime call.**
`clear(buf)` on a `[][]byte` is a call to `runtime.memclrNoHeapPointers`, and
building `lenMask` walked `keys` a second time; both showed up in GetMany's own
profile (2.8% and 9.2%). Clearing the slot inside the `lenMask` loop is
**−4.8%** on `GetMany` measured alone, and it is the whole of that row's win
here — the parse side of GetMany is `~`.

**The medium-density regressions are real and have a mechanism.** Draining
costs about seven extra instructions per escape (`m & -m`, the pair clear, the
second byte compare, the lane-7 test) and only refunds them when a word holds
more than one escape. At gaps of 32–48 bytes it never does, so those rows pay
the bookkeeping for nothing — and the walk also re-anchors less tightly, since
it resumes at the word boundary rather than at the escape+2 the old shape used,
which costs about half a word per escape. The realistic shapes are on the other
side of that line: `sample_big.txt`'s escapes are 38 bytes apart but there are
only two of them (`LevelTS` +0.7%, the whole realistic cost of this trade),
while embedded JSON — the shape the scan was written for — is 25% to 31%
faster. That +0.7% is at this harness's floor, incidentally: it was p=0.009 in
the deciding series and `~` (p=0.093) in a confirming one, which is about what
two escaped quotes are worth. **`escClean` = 4 was measured as the obvious answer to the regression
and is not it**: it does fix `gap=040` (+11.7% → `~`) and does nothing for
`gap=032`, but it drops the sample line off the walk for **`LevelTS` +20.6%,
`GetMany` +9.8%, `Iterate` +3.5%** — the same cliff the constant's own comment
records from 2026-08-17, deeper now because the drain made the walk better for
exactly that value. Five stands.

**Codegen invariants re-checked on go1.27.1, and two of them have drifted with
the toolchain.** All five SWAR helpers still inline and `iterate` still does
not; every SWAR load in `iterate` and in the escape walk is still bounds-check
free (`-d=ssa/check_bce/debug=1` reports 33 checks in the package, one fewer
than at HEAD — the one this pass removed); all 14 alloc-free entry points still
are. But the prove pass now folds **eight** cap-zero masks by limit where the
2026-09-01 note recorded five, and `iterate` carries **20 NOPs** where that
note asserts none — go1.27.1 emits alignment padding the older toolchain did
not. Neither is a regression (the same 20 are in the HEAD binary, and
2026-09-01 priced a NOP at exactly zero on this core), but the "no NOOPs"
invariant as written no longer means what it says: check the count against a
build of HEAD rather than against zero.

**Measured and rejected this pass** — each was correctness-verified before
losing on the stopwatch:
- *The exactness term taken out of the key mask.* `hasKeyStop`'s `&^ w` costs a
  NOT, an AND and a copy of `w` — three of the wide loop's fifteen instructions
  — and it is what excludes bytes >= 0x80. Dropping it and letting the existing
  byte re-check reject the spurious stops is worth **`Iterate` −5.2%**, the
  largest single number this pass found, and it cannot be kept: the exact
  `hasCtrlOrSpace` the seeded value scan needs then has to be built on the hit
  path instead of CSE-ing out of the loop, which moves two operations from the
  per-word path to the per-field one, and short-key input pays that per field.
  **`DecodeKeyval` +14.6%**, `GetMany` +1.6%. Three ways round it were measured
  and all three cost more than the win: ANDing the term back in on the hit path
  (exact again, no cliff — `Iterate` −2.0% but `DecodeKeyval` +4.1%, `LevelTS`
  +1.8%, geomean +0.8%); stepping one byte on and re-scanning with
  `i >= 0 && i <= lim16` in the loop head so the prove pass keeps the load
  bounds-check free (`Iterate` −2.3%, `DecodeKeyval` still +8.8%); and an
  in-word retry loop that clears the offending lane, which the register
  allocator answered by spilling the word itself inside the loop
  (`Iterate` +5.7%, everything worse). The rule this pass adds: **an
  instruction moved from a scan loop to the per-field path costs about five
  times what it saved**, so trading per-word work for per-field work needs
  three or more words per field to break even, and the sample line's 3.07 is
  not enough of a margin to carry the short-field rows with it.
- *Dropping `bytes.IndexByte` from the quoted scan for an inline SWAR search.*
  Measured as the extreme case first (SWAR only, no call at all): `LevelTS`
  −3.8%, `DecodeKeyval` −2.5%, `IterateJSONMsg` −3.5%, `GetMany` −1.3% and
  **`Iterate` +7.7%**, the last from the sample's two long quoted values
  (203 B and 159 B). A bounded probe is the obvious middle, and the crossover
  says it cannot pay here: in a latency-chained micro-benchmark
  `bytes.IndexByte` costs a flat ~4.7 ns up to a 56-byte match against a called
  SWAR scan's 3.7 ns + 0.078 ns/byte, so the call is worth only about **18–24
  bytes** of SWAR and a probe long enough to catch the sample's 33-byte values
  wastes more on its 203-byte one than it wins on the other four. This is the
  same conclusion as the 2026-09-01 rejection of the 8-word probe, reached from
  the cost curve rather than from `esc=0`.
- *Arithmetic `isSpace` with a branch* (`b == ' ' || b-'\t' <= '\r'-'\t'`),
  tried because the compiler materialises `spaceTable`'s base address with a
  LEA on hot paths that never index it — including the `c == ' '` fast path
  after a value stop. Geomean +0.03%, every row `~` bar `IterateJSONMsg`
  +1.3%. The table wins a third time, and the finding is really about the cost
  model: a single ALU op that is not on the per-field dependency chain is free
  here.
- *Spending `escClean` as a precomputed limit* rather than counting words down,
  to take an increment and a compare out of the dense scan's inner loop: the
  prove pass will not relate the limit to `len(data)`, so the SWAR load gets
  its bounds check back and the word gets spilled. Slower, and it would only
  have helped the sparse case anyway.
- *Draining with an escaped-lane marker* (`esc := t+1`, skip that lane, clear
  one bit per pass) instead of clearing both lanes at once. Fewer instructions
  per pass, but `\"` — every escape in embedded JSON — has both lanes flagged,
  so it runs the loop twice per escape: `esc=128` −15.2% against the landed
  −27.8%, `IterateJSONMsg` −20.1% against −25.7%, and every gap row worse.

### 2026-09-22 pass — hardware counters and IBS (Ryzen 8840HS, amd64, Go 1.27.1)

Run on counters rather than on the stopwatch, because the machine was too busy
for timings to decide anything small: the same binary ran ~20% slower in wall
time than at the session's start (a browser and a local Kubernetes cluster
shared it), and the A/A timing control's per-row spread was ±5–12%. Core
cycles do not move with the clock and per-op counts are exact, so decisions
were made on `bench/counters.py` — cycles, macro-ops, instructions and
branches per op by the difference method, pinned to core 10, best of three
interleaved repetitions, reproducible to about 1% — with AMD IBS for
per-instruction latency and a timed `compare.py` series as the secondary
check. HEAD `89e9b0a` → this pass, cycles per op (`FieldShape/quoted`
re-measured alone; see the harness lessons):

| Benchmark | before | after | Δ |
|---|---:|---:|---:|
| `IterateOur` | 863.2 | 852.0 | −1.3% |
| `GetMany_TimestampLevel` | 208.3 | 205.5 | −1.3% |
| `LevelTS_LogFmt` | 173.7 | 160.0 | −7.9% |
| `DecodeKeyval_Custom` | 1548431.0 | 1508914.3 | −2.6% |
| `IterateJSONMsg` | 345.6 | 314.9 | −8.9% |
| `Unescape` | 74.5 | 72.6 | −2.6% |
| `UnescapeJSONMsg` | 590.3 | 567.5 | −3.9% |
| `UnescapeUnicode` | 253.6 | 253.6 | ±0.0% |
| `AppendValueUnicode` | 344.8 | 344.2 | −0.2% |
| `Get/level` | 185.5 | 171.7 | −7.4% |
| `Get/session_attr_client_locale` | 464.9 | 441.1 | −5.1% |
| `Get/trace_id` | 977.5 | 938.4 | −4.0% |
| `IterateFieldShape/unquoted` | 125103.1 | 121719.7 | −2.7% |
| `IterateFieldShape/second-word` | 63925.3 | 62996.8 | −1.5% |
| `IterateFieldShape/quoted` | 313793.8 | 317757.6 | +1.3% |
| `IterateFieldShape/bare` | 108748.3 | 107881.9 | −0.8% |
| `IterateEscaped/esc=0` | 86.3 | 82.4 | −4.5% |
| `IterateEscaped/esc=8` | 235.8 | 232.0 | −1.6% |
| `IterateEscaped/esc=32` | 716.9 | 678.5 | −5.4% |
| `IterateEscaped/esc=128` | 1111.2 | 963.1 | −13.3% |
| `IterateEscaped/esc=500` | 3086.1 | 2399.3 | −22.3% |
| `UnescapeEscaped/esc=0` | 91.1 | 90.9 | −0.2% |
| `UnescapeEscaped/esc=8` | 396.7 | 401.0 | +1.1% |
| `UnescapeEscaped/esc=32` | 930.7 | 898.0 | −3.5% |
| `UnescapeEscaped/esc=128` | 2642.3 | 2416.5 | −8.5% |
| `UnescapeEscaped/esc=500` | 4404.1 | 3004.9 | −31.8% |
| `IterateEscapedGap/gap=016` | 883.9 | 811.3 | −8.2% |
| `IterateEscapedGap/gap=032` | 720.7 | 682.3 | −5.3% |
| `IterateEscapedGap/gap=040` | 675.9 | 648.3 | −4.1% |
| `IterateEscapedGap/gap=048` | 538.9 | 520.2 | −3.5% |
| `IterateEscapedGap/gap=064` | 392.3 | 392.6 | +0.1% |
| `IterateEscapedGap/gap=128` | 233.3 | 237.4 | +1.8% |
| `IterateEscapedGap/gap=256` | 162.6 | 160.5 | −1.3% |
| `IteratePrefixJSON/prefix=008` | 192.9 | 165.6 | −14.2% |
| `IteratePrefixJSON/prefix=032` | 193.3 | 168.5 | −12.8% |
| `IteratePrefixJSON/prefix=064` | 413.2 | 411.4 | −0.4% |
| `IteratePrefixJSON/prefix=160` | 423.9 | 422.6 | −0.3% |

No row regressed beyond the ±2% these repetitions reproduce to. All four
differential fuzzers clean (30 s each), `-race` clean, golangci-lint v2.13.1
clean on both modules (v2.12.2 cannot typecheck go1.27's stdlib; CI pins
v2.13.1 for that reason). The timed series (`compare.py`, n=6 × 1 s, pinned,
alternating; its A/A control was `~` everywhere at ±5–12% per row, and rows
spread up to ±77% during the series itself) resolved only the large rows, all
the same way round: parse `esc=500` −28%, decode `esc=500` −36% and `esc=128`
−13%, JSON msg −11.5%, prose-then-JSON −16%, 16-byte gaps −13%, `Get/level`
−12%, geomean −8.8%, nothing significantly worse.

**Where the time goes, measured rather than modelled.** Microbenchmarks of
this core's ports: a fused compare-and-branch issues at **2 per cycle**, as do
`SHR imm` and a three-component `LEA`; plain ALU ops and `BSF` (1-cycle
latency) issue at 4. `Iterate` on the sample retires ~4150 macro-ops and 969
branches per parse in ~850 cycles, so the two branch-capable ALUs need ~485 of
those cycles for branches alone, and the schedulers that feed them (IBS and
`de_dis_dispatch_token_stalls2`: sch3 ≫ sch0 > sch1, sch2) are the ones that
fill. A seeded short field costs ~21 branch ops (7 of them bounds checks, 3
taken `JMP`s) and ~27 other ALU ops around a ~12-cycle chain (load 5 → mask 3
→ BSF 1 → SHR 1 → two LEAs 2): 15.2 cycles per field on
`FieldShape/unquoted`. A short QUOTED field costs ~40: IBS puts the reload of
`bytes.IndexByte`'s result ~49 cycles after its dispatch, because the ABI0
call passes arguments and result through the stack, and the result is on the
per-field chain. The dense decode loop is the opposite case — frontend-bound,
55% of dispatch slots empty, all taken branches.

**1. The BSF output dependency (the finding of this pass).** `BSF` leaves its
destination unchanged for a zero source, so the core treats the destination's
old value as an input; a microbenchmark shows dependent `BSF`+`LEA` steps at
2 cycles each, and 0.6 with the destination zeroed first. At GOAMD64=v1 every
`bits.TrailingZeros64` is a `BSF` (v3's `TZCNT` has no such input; Go even
zeroes its destination), and Go's register allocator gives the result the
lowest free register not reserved for something else. In `Iterate` that is
often the register a byte load has just released: removing the `quoted`
parameter (item 2) moved the '"' test's byte load into `R11`, the value scan's
`BSF` then took `R11`, and the load — which waits on the key's `BSF`, `SHR`
and `LEA` — landed on the value chain. The instruction stream was IDENTICAL to
HEAD's but for register names, and it ran `FieldShape/unquoted` +37%,
DecodeKeyval +12%, the sample +3%; IBS showed the `BSF` completing ~8.6 cycles
after its operand where HEAD's completes ~0.6 after. Loading the scan
constants `sub, hi, xor` instead of `xor, sub, hi` made every `BSF` inherit a
register written by one of those loads, long complete — and the pass's gains
then showed. Moving `Iterate` 704 bytes changed nothing (not aliasing); four
source respellings of the seeded dispatch changed nothing (the allocation is
stable against local edits); the load order did. `bench/bsfdep.py` walks the
control-flow graph back from each `BSF` and lists its destination's possible
previous writers; a writer that is a load from data is flagged SUSPECT.
Nothing in the language pins a register, so **run it after any change to
`Iterate`**, and suspect this mechanism first when a change's cycles move
without its op counts moving — earlier passes filed several such swings as
layout noise.

**2. The quoted bit, derived instead of reported.** See the protocol section:
`Iterate` lost its `quoted *bool` parameter and becomes the parser; `GetQuoted`
keeps the uncapped winning value and calls `wasQuoted` once. Worth −1.4…−3%
of macro-ops on the lookups, and it removed four throwaway `var quoted bool`s.

**3. The drain's pair step.** The escape walk used to clear an escape pair
with `m &^= low | low<<8` after `low := m & -m` — six dependent operations per
escape, and at the densities the walk exists for, that chain IS the walk. Now
`m &= (m | -m) << 9`: `m | -m` sets every bit from m's lowest, bit 7 of lane
t, upward, and shifting by nine starts it at bit 0 of lane t+2 (and clears it
when there is no such lane, since lane 7 leaves the word first). Four
operations. The lane comes from `TrailingZeros64(m)`, which the loop condition
proves non-zero, where `TrailingZeros64(m & -m)` compiled to `BSF` plus a
`CMOV` for a zero input that cannot occur; with `low` gone from the live set,
the per-escape spill of the hi constant went too. −22% at 2-byte gaps, −9% on
embedded JSON, −13% on prose-then-JSON, every escape row ahead or level.

**4. The decode table.** `AppendUnescape`'s `switch` on the escaped byte was a
binary search whose blocks sat far apart: 4.4 taken branches per escape in a
loop whose frontend delivered ops to only 45% of dispatch slots. A 256-byte
table (`unescapeByte`: n, r, t mapped, every other byte itself) for all but
`\u` leaves 3.4 per escape: −32% at 500 escapes per KB, −9% at 128, −4% on
the JSON message, Unicode and sparse rows level despite ~4% more macro-ops. The
three taken branches left are `append`'s grow block laid out inline (Go's
layout honours the "unlikely" hint by scheduling the split critical edge,
then places the grow block before the shared store) and the merge after the
`\u` branch; see the rejected list for the attempts on both.

**5. DRY, at zero codegen cost.** `GetMany`'s small and indexed paths share
`resetSlots`, `lenSet.excludes` and `settle` (the first-non-empty rule, once);
the argument-free SWAR predicates call the `*R` forms, so each formula exists
once; the two unterminated-value errors share `errUnterminated`. The machine
code of `Iterate`, the escape walk and `AppendUnescape` hashed identical before
and after. The indexed `GetMany` path moved −1…+5% on counters from alignment
NOPs alone; it is not a path the realistic suite runs.

**Measured and rejected** (each correctness-checked before losing):
- *The 16-byte view for the seeded path's verify loads*, removing three bounds
  checks per field (−73 branch ops per parse): Go CSEs the view's pointer with
  the key word's address and materialises its `LEA` in the per-word key loop,
  which spends what the checks saved (the 2026-09-01 result, confirmed), and
  no constant-load order kept its value `BSF` off the '"' byte's register.
  Second-word row −3%, unquoted row +34%.
- *`base + 1` precomputed* so the seeded next-field index is one add: the
  compiler folds the constant back out into the same two `LEA`s.
- *The quoted scan started with the SWAR walk*, no `IndexByte` first: short
  quoted fields −30%, DecodeKeyval −10%, JSON −13%, but LevelTS +11%, GetMany
  +6%, `esc=0` +25% — 30–60-byte values (timestamps, messages) walk five words
  and then call `IndexByte` anyway.
- *An inline two-word probe for the closing quote*: short quoted −30%, but the
  probe's word is spilled and reloaded twice inside `Iterate`, LevelTS +29%.
  In a register-ABI helper (`closingQuote`, which also merged the two error
  sites): LevelTS +11.5%, GetMany +10%; without the probe that helper is a
  wash. The quoted path's cost is the `IndexByte` call's ABI0 boundary, and
  every way around it so far charges the realistic 30–60-byte values more
  than it saves the short ones.
- *`AppendValue` handing quoted values straight to `AppendUnescape`* (one
  backslash search, not two): escaped values −3…−5%, clean quoted values +1.8%
  for the call into `AppendUnescape`'s large frame — a loss on the common
  case. Splitting the decode loop out to avoid that made `AppendValue` best
  everywhere but moved a NOP and a rematerialised `LEA` into the frontend-bound
  dense loop: `esc=500` +20%.
- *`\u` as an out-of-line fix-up after an unconditional table append*, to let
  the common path fall through: dense −5%, JSON −4%, Unicode +8%, 48-byte gaps
  +9% (the probe loop's constants rematerialised by the new allocation).
- *`escClean` 6/7 and `escGap` 56/64*, re-swept after the faster drain: 5/48
  still win (6 costs 48-byte gaps +25%, 64 costs 64-byte gaps +7%). The drain
  cut the per-escape cost, not the clean-run crossover those constants encode.
- *Dropping the deliver block's `vStart > n-1` guard*, now that branch ops are
  known to be the scarcer port: −29 branch ops per parse, but the cap-zero mask
  it folds comes back as more ALU work — Iterate +4.3%, DecodeKeyval +1.7%.
  The 2026-09-01 trade stands.
- *Recognising the '=' stop from the masks* (`m&-m&mv == 0`: '=' is the one key
  stop the value mask lacks) instead of loading the byte: −35 branch ops per
  parse and short unquoted fields −3.9%, but Iterate +3.6%, second-word +5.4%,
  bare keys +5.9%. The 2026-09-01 attempt spilled `m`; this one did not, and
  still lost.

**`GOAMD64=v3` on this tree** (TZCNT has no destination input, so v3 builds
are immune to the BSF lottery): `Iterate` −4.9%, deep `Get` −3.9%, short
unquoted fields −3.2%, GetMany −2.6%, LevelTS −1.2%, DecodeKeyval `~` — but
`IterateJSONMsg` +5.3% and `esc=128` +16% at identical op counts, so the
escape walk's v3 code is slower than its v1 code (the README now says so).
Against HEAD built at v3, this tree is still ahead on those rows (JSON −11%,
`esc=128` −9.5%): the old drain was already four operations there, since v3
spells `m & -m` as one `BLSI`.

**Harness lessons.** `-test.bench` splits its pattern on `/` and matches each
part unanchored, so `IterateFieldShape/quoted$` also ran `unquoted` and one
counter row summed both; `counters.py` anchors every part. IBS works only
system-wide, so it needs root: run the benchmark pinned to core 10 as the
user, and `sudo perf record -C 10 -e ibs_op/cnt_ctl=1/ -R -c 20000 -- sleep 2`
beside it; `perf report -D` decodes each sample, and `TagToRetCtr −
CompToRetCtr` is the op's dispatch-to-completion latency, which is how the
late `BSF` was found. Where `cycles:u` sampling attributes stalls to the
instruction after the culprit, IBS attributes them to the op.

### 2026-09-23 pass — arm64-native escape paths (Neoverse N2, Azure, 2 vCPU, Go 1.27.1)

The first pass driven by counters on arm64. Baseline `3cb3ae9`; before and
after built from identical test sources; decisions on per-op cycles
(`bench/counters.py`, CPU 1, best of 2–3 interleaved repetitions), claims from
`bench/compare.py` (n=6 × 0.5 s, pinned, alternating; the A/A control read `~`
on every row at +0.01% geomean, spread ±0–1%). Full tables, controls and
reproduction: [bench/perf_2026-09-23_arm64.md](bench/perf_2026-09-23_arm64.md).
**amd64 machine code is identical to the baseline for every package function**
(`bench/asmdiff.py` on cross-compiled binaries), so nothing below says anything
about x86.

**The core, measured** (Go-assembly microbenchmarks, independent chains): ~5
ops/cycle dispatch; 4 integer ALUs (1-cycle `ADD`/logical/`RBIT`/`CLZ`/`LSRV`/
`UBFX`/`CSEL`; `ADD` with an `LSR`-shifted operand is 2 cycles on 2 pipes, with
`LSL` ≤ 4 1 cycle); 2 branch units, and `CMP`/`TST`+`B.cond` fuse into one op;
3 loads per cycle, L1 load-to-use 4 cycles for immediate and register offsets
alike; store-to-load forwarding ~5 cycles; register `MOV`s are partly
eliminated (~0.7 cycles each in a chain) but take an ALU slot; a non-encodable
64-bit constant is 4 ALU ops. `bytes.IndexByte` costs ~30 cycles flat for a
match within 24 bytes (~34 to 64, 45 at 200) — mostly the NEON↔GPR round trip
— against 12 cycles at 1 byte and 22 at 32 for a called SWAR scan: the
crossover is ~48–56 bytes. `Iterate` on the sample runs at IPC 4.3 (3.9
ops/cycle), 20% backend-stalled; a short quoted field costs 54 cycles against
19 for a short unquoted one.

**Where the parser's time goes, by elimination.** Not calling the callback at
all is −21% on short fields and −11.6% on the sample (~4 cycles a field: `i`
goes through the stack across the call, ~5 cycles of store-to-load forwarding,
plus the argument moves). Halving the key loop's stride prices it at 2.9
cycles per key word, ~26% of the sample; the value loop is ~1%. Everything
else was tried one piece at a time and none of it moves short fields by more
than ~1% (rejected list below): not the quote test, not the `vStart > n-1`
guard, not the `'='` verify, not the `+1` on the chain, not the op count. The
cycle-sampled profile (arm64's PMU interrupt lands on the oldest unretired
instruction) puts half the samples on the key-verify chain — mask, `RBIT`/
`CLZ`, shifted add, byte load, compare — but shortening that chain from the
register instead of a load made short fields 9–11% SLOWER. `Iterate` is
therefore unchanged in this pass; its arm64 machine code is the baseline's.

**What landed, all on the escape paths, all arm64-only:**
1. **The walk, spelled for arm64** (`scan_arm64.go`): a quiet limit on the
   index instead of a clean-word counter (the counter and the index traded
   registers every iteration), `n-8` hoisted, the drain re-checking a lane's
   byte from the register with `byte(w >> (tz & 56))` — one cycle here, where
   amd64 needs the count in `CL` and has rejected the spelling three times — so
   the word's address leaves the loop and the drain loses a 4-cycle load. 22
   instructions per clean word → 16. Escape rows −9…−11%, JSON in `msg=` −4%.
2. **`escClean` = 8, `escGap` = 64 on arm64** (5 and 48 stay everywhere else):
   with the walk at ~0.57 cycles a byte and each hand-off to the sparse scan an
   `IndexByte` call, the crossover is ~64 bytes. 48-byte gaps −29%, 64-byte −12%,
   every other row flat.
3. **The sparse scan hands a value back to the walk, once** (see "The escGap
   entry decision is one-way" under Known limits for the full story, including
   the +28–34% the unrestricted version cost): prose-then-JSON −64%.
4. **`unescapeInto`** (`unescape_spare.go`), for destinations with room for all
   of `raw`: the old loop spent ~92 instructions per escape at 8-byte gaps,
   most of them a `memmove` call per literal run and the seven registers spilled
   around it. Runs up to 32 bytes are now copied inline — two to four
   overlapping loads, then as many stores, which keeps an in-place decode
   correct; longer runs still call `memmove`, which wins past ~32 bytes (an
   8-byte copy loop for every run was +66% on 128-byte runs). The next-escape
   probe keeps its constants in registers (spelled as constants, each word
   rebuilt `0x5c5c…` with `MOVZ`+3×`MOVK`), is 8 words, and is skipped after a
   run of 64+ bytes until a short one turns up. `decodeSurrogateEscape` is
   written out inline (as a call, it made each surrogate pair spill and reload
   the loop's state twice: +5% on that row, now −3%), and `copy` for long runs
   is given an exact-length destination. Decode rows −7…−33%, JSON −17%,
   `esc=128` −19%; `Unescape` +3.6% (one call more per decode — folding the loop
   into `AppendUnescape` to save it lost 3 points on every dense row), 256-byte
   gaps +1.4%. Allocation counts and bytes unchanged on every row.

**Measured and rejected** (each fuzz- or test-clean before it lost):
- *A SWAR probe for short quoted values* before `IndexByte`, 1/2/4/6 words:
  short quoted fields −38%, `DecodeKeyval` −6%, but `LevelTS`/`GetMany`/
  `Get/level` +2–10% — the sample's quoted values are 16+ bytes, and each miss
  costs 2–5 cycles in context (1.5–2 in an isolated sweep). The same trade as
  the rejected fused value scan: the realistic line decides.
- *The key loop's test as a fused `TST`+branch* (`orr &^ w & 0x8080… != 0`, the
  exact mask rebuilt at the hit from the same `ORR` so the key chain keeps its
  length): 2 fewer ops per key word, `IterateOur` −2.2%, `Get/trace_id` −3.1%,
  but `GetMany` +0.9%, `DecodeKeyval` +0.9%, second-word fields +3.0%; geomean
  −0.06%. The simpler form (mask rebuilt after the branch) put a cycle on the
  key-end chain: bare keys +5.5%.
- *The view's second word from `base+7`* (so its address shares nothing with
  the loop and the loop loses its per-word `ADD`): the slice's bounds check
  comes back, worse than the `ADD`. `data[base+8:base+16]` CSEs with the loop
  increment and trades the `ADD` for a `MOV`; an eager `pair[8:16]` load gets
  no `LDP` (the pair pass wants two immediate-offset loads) and two `NOP`s.
- *Stop bytes verified from the register* (`'='`, `'"'` and the value stop,
  no byte loads): +9–11% on short fields.
- *The next field's index from the value mask moved up a lane*
  (`TrailingZeros64((hb<<1)&(sw<<1))>>3`, which drops the `+1` from the chain at
  the same op count and compiles exactly as intended — `TST`, `ADD`, `AND` with
  a shifted operand): short fields −0.3%.
- *Arithmetic `isSpace`* (no `ADRP`+`ADD` for the table's address, which the
  allocator rematerialises on many edges): −2.5% ops, cycles flat.
  *Scan constants hoisted out of the field loop*: +2–4%, spilled across the
  callback exactly as on amd64 in 2026-07-27.
- *Decoder*: a fixed 6- or 8-word probe (see above); the loop inside
  `AppendUnescape` itself; probe constants reloaded per probe instead of held
  (−3 points on dense rows); `utf8.EncodeRune` into the buffer (+1 point on
  `\u` rows against `AppendRune`).

**Harness lessons.** (1) A padding control needs a REACHABLE pad: a
never-called unexported function is removed by the linker and moves nothing —
reference it from a test file (`var _ = pad(3)`). With one, `Iterate` placed
at offset 0 or 32 of a 64-byte block measured the baseline to ±0.4% on every
core row, at 16 or 48 +0.7–0.9%; the same code retires ~40 fewer ops per parse
at the 32-byte-aligned offsets, so compare-and-branch fusion depends on where
a pair falls. This is the N2's layout band, and a consumer's binary places
`Iterate` wherever it lands. (2) A copy-correctness seed must not be periodic:
a run of `x` let an off-by-one copy through. (3) The VM has two cores; run
fuzzers and builds between series, never during one.

### Cost model (measured 2026-07-26, synthetic field-size sweep)

**Not re-measured after the 2026-07-27 pass**, which attacked the fixed
per-field term specifically. `sample_big.txt` holds **29** pairs (counted, not
estimated), so that pass moved it from 358.1/29 = 12.3 to 308.8/29 = 10.6
ns/field; earlier notes here divided by a wrong field count and read ~7.7 to
~6.7, which is why the overhead row below is not to be trusted as-is. The scan rows are
unaffected (neither scan loop's inner shape changed for values). Re-run the
sweep before quoting the overhead figure again. **Doubly stale since
2026-08-22**: the seeded value scan attacked the same term a second time
(sample2 at 199.9/29 ≈ 6.9 ns/field within that pass's own session — an
absolute, so not comparable across sessions on this machine), and it helps
exactly the 4–16 B values the overhead row was measured on, so the sweep would
now show a DIFFERENT overhead for values that fit the view than for ones that
do not. The `Get`-per-skipped-field row shrinks with it (skipping is a full
parse per field). The long-value scan rows still stand. **And the reading
under the table is wrong in kind since 2026-09-01**: the loop is not
"memory-latency bound"; on Zen 4 it is bound by the four integer ALUs, with
the post-value chain exposed on short fields — see that pass's section.

| Shape | Cost |
|---|---|
| Fixed per-field overhead (4–16 B values) | ~5.8–7.2 ns/field |
| Unquoted value scan (SWAR), 256 B values | ~11.7 GB/s |
| Quoted value scan (`bytes.IndexByte`), 512 B, **no escaped quotes** | ~27 GB/s |
| Quoted value scan, **per escaped quote** | ~10 ns each (see below) |
| `Get` per skipped field | ~8.6 ns (10.7 ns for field 0, 551 ns for field 63) |

**The 27 GB/s row is escape-free only** — do not quote it unqualified. Each `\"`
found by `IndexByte` costs a fresh, non-inlinable call plus a re-walk of the
preceding backslash run: ~9.4 ns each on the N2 (Ryzen: ~10). Since 2026-08-17
that price is paid only for the *first* escaped quote of a cluster: from there
the parser scans forward a word at a time for `"` or `\` (`hasQuoteOrBackslash`),
consuming each backslash with the byte it escapes, and no per-escape call at
all; it hands back to `IndexByte` when the escapes turn out to be sparse
(`escGap`/`escClean`, see the optimization notes). Since 2026-09-09 that walk
drains each word of every escape before loading the next, which took the
densest case from ~2.5 ns to **~1.3 ns per escape** (`esc=500`: 1272 → 654 ns
for 500 escapes in 1 KB) and left the sparse end of the same sweep 6–12%
slower, where the drain's bookkeeping has no second escape to amortise
against. Since 2026-09-22 the drain's pair step is four dependent operations
instead of six: **~4.8 cycles (about 1 ns) per escape** at that density, and
the fixed-width sweep's ratio is ~29× (2399 against 82 cycles per op), not
the 59× below. Embedded JSON in a `msg=` field — every JSON quote becomes `\"`, one
escape per ~2–8 bytes — is the realistic shape this serves:
`Benchmark_IterateJSONMsg` −44%, `Benchmark_UnescapeJSONMsg` −40%. At a
**fixed** 1 KB value the parse sweep now runs 35.6 ns (0 escapes) → 2.11 µs
(500), about 59× (it was 36 → 4.72 µs, 130×, on this machine; 65 → 6100 ns, 90×,
on the Ryzen). `Benchmark_IterateEscaped` pins that axis and
`Benchmark_UnescapeEscaped` pins the same axis for the decoder, which is the
slower half once escapes are dense (2.98 µs against the parser's 2.11 µs at
esc=500). Both exist because `sample_big.txt` has 2 escaped quotes in 1.4 KB
(~4%) and so makes the quoted scan look like pure `IndexByte` throughput — but
note the sweep's own blind spot, between its 32-byte and 8-byte gap points,
which is where that sample's 38-byte gap sits and where the first version of
this scan lost 7% on GetMany.

This is **not** the rejected "inline first-word `hasByte` before `IndexByte`"
item, which was about sparing short *clean* values a call — clean values still
never touch the SWAR path.

Reading: short fields are **overhead-bound** (~6 ns of loop/callback per pair,
scan is noise), long values are **scan-bound**. The 8 B/iter SWAR only starts
paying off above ~32 B — which is why the memchr2/SIMD experiments below lost.
`sample2` averages ~8.3 ns/field, consistent with the sweep.

## The quoted-bit protocol (2026-08-08 split, measured and reshaped 2026-08-17)

**Superseded 2026-09-22: there is no protocol any more.** `Iterate` delivers
values uncapped, so the byte just past a quoted value is its closing quote,
while an unquoted value ends only at whitespace or at the end of the record
(the value scan stops on nothing else, so it can never stop in front of a
`"`). `wasQuoted(v)` reads that byte, `GetQuoted` calls it once for the value
that won, and the parser lost a parameter, a store on every quoted field and
the flag's load and store in `GetQuoted`'s callback on every field: one-key
lookups −4…−7%, LevelTS −8%, Iterate −1% (cycles, 2026-09-22 pass). This is
the "Not taken" alternative at the end of this section, and its objection —
that it couples `GetQuoted` to the uncapped-value property — is accepted as
the price: capping inside `Iterate` was rejected at −4.5% anyway, and the
fuzzer's `iterateQ` now derives the bit the same way, so any capping change
fails `FuzzIterateAgainstRef` against the reference's own quoted flag at once.
The history below explains why the bit exists and what the protocol cost.

`iterate` reported whether a value was double-quoted — the only position where a
backslash escape means anything. That bit exists because of a **correctness**
fix, not an optimization: `AppendValue` used to run `AppendUnescape` over every
value it found, including unquoted ones, so `path=C:\Users\bob\new` came back
as `C:Usersbob` with an embedded newline (`\U`→`U`, `\b`→`b`, `\n`→newline).
Escapes are meaningful only inside quotes, the raw value cannot tell you which
it was, and go-logfmt's encoder does **not** quote a value merely for containing
a backslash — so this was silent corruption on ordinary input. `GetQuoted`
exports the bit.

**How it travels (since 2026-08-17):** `iterate(data, quoted *bool, fn
func(k, v []byte) bool)`. The parser sets `*quoted = true` just before
delivering a quoted value and **never clears it**; a caller that wants the bit
reads and resets it inside its callback (`GetQuoted`, and the two fuzz
references via `iterateQ`), everyone else hands in a throwaway local. The
protocol is lopsided on purpose: the common unquoted path executes no store at
all, and `Iterate`/`All` hand the user's callback straight to the parser.

**Why not the 2026-08-08 shape** (three-argument parser callback, `Iterate`
wrapping the user's two-argument one in an adapter closure): measured on the
quiet Neoverse N2 box, pre-split `f8d9551` vs `b83eade`, n=6 pinned interleaved
rounds, control clean: **Iterate +2.33%, LevelTS +4.44%, DecodeKeyval +1.14%,
GetMany +0.51%**. The adapter is a full non-leaf function per field (stack
check, frame, two arg spills for the GC, context load, indirect call,
epilogue — ~15 instructions and a second `CALL`/`RET` pair).

**Why not the obvious out-parameter** (store `*quoted = q` before *every*
callback): Iterate −1.69% but **GetMany +1.79%** and Get/GetQuoted worse too —
the pointer reload plus store on every field is not free on this core (geomean
−0.1%, i.e. a wash). The set-only protocol landed instead: Iterate −2.36%,
LevelTS −2.39%, GetMany −0.89%, DecodeKeyval −0.83%; the price is a
load+store per field inside `GetQuoted`'s closure to consume the flag,
**Get/GetQuoted +0.6%** on a deep key (measured with a temporary
`Get(sample2, "session_attr_client_locale")` benchmark), and `Validate` `~`.
`Get` deliberately shares `GetQuoted`'s closure rather than duplicating the
first-non-empty state machine to claw that 0.6% back.

**Not taken:** deriving the bit inside `GetQuoted`'s callback from the value's
capacity (`data[len(data)-cap(v)-1] == '"'` — sound because `iterate` hands
out uncapped sub-slices of the cap-pinned `data`) would cost nothing anywhere
in the parser, but it couples `GetQuoted` to the uncapped-`v` property that the
`Iterate` capping trade-off could one day revisit. Also, the compiler already
materialises the flag at the call site as `c == '"'` from the spilled byte, so
the "derive `quoted` at the callback site" idea from the 2026-08-08 notes was
happening implicitly and bought nothing.

## How the general parser is optimized (logfmt.go)

- **SWAR scanning** (`hasCtrlOrSpace`, `hasKeyStop`): scans keys/values 8 bytes per
  iteration. `hasCtrlOrSpace` flags bytes `<= 0x20` with one subtract (covers
  all whitespace); the located byte is re-checked so rare non-whitespace control
  bytes (0x00–0x08, 0x0E–0x1F) fall back to the scalar tail. Masks are only
  **OR-ed** then `TrailingZeros64`'d — never subtracted from each other (a borrow
  can set spurious high bits *above* a true match, which is fine for OR+find-
  first but breaks subtraction; this was a real fuzz-caught bug).
- **`hasKeyStop` is two `<= 0x20` tests, not a `<= 0x20` test plus an equality
  test.** XOR-ing by `0x1d` turns `b <= 0x20` into a test for exactly
  `{0x00..0x1f} ∪ {'='}`: `0x1d < 0x20` so the XOR permutes the low 32 values
  among themselves, `0x3d ^ 0x1d == 0x20` pulls `'='` in, and bits 5–6 are
  untouched so nothing else reaches `0x20`. Union with the plain term is exactly
  the key-stop set. `0x1d` is *forced*, not a lucky pick: the fold needs
  `k < 0x20` **and** `k ^ 0x20 == 0x3d`. Pinned exhaustively by
  `Test_Unit_SWARMasks`.
  The payoff is that both terms subtract the **same** word, so the scan needs
  three broadcast constants instead of four. The shared `&^ w` factors out on
  top of the `& swarHi` that was already factored, which is sound because
  `0x1d` has bit 7 clear: bit 7 of each byte of `x` therefore equals bit 7 of
  `w`, and bit 7 is the only position the result is ever read at. Both terms
  must still be combined with OR only — the borrow caveat above is unchanged.
- **No per-field whitespace-skip loop.** Every field used to open with
  `for i < n && isSpace(data[i]) { i++ }`, which costs *two* iterations (one to
  eat the single separator, one to notice the key started) — about eighteen
  instructions before the key scan could even begin. Instead the value scan and
  the bare-key path consume the separator they have already located, with an
  **unconditional** `i++`: `i` may reach `n+1`, and every bound in `Iterate` is
  `i < n` or `i+8 <= n`, both of which treat `n+1` exactly as they treat `n`.
  A separator *run* (or leading whitespace, or a `\t`/`\n` delimiter) then
  leaves the key scan stopping at offset zero with an empty key, and that empty
  key is the signal to drain the run — correct, and off the hot path. This is
  why the "consume the known delimiter at valEnd" experiment below measured
  −5%: on its own it adds a branch *and* still pays for the skip loop.
- **The SWAR verify dispatches straight to `keyEq`/`keyBare`.** Each label
  already knows both what byte was found and that `i < n`, so neither re-loads
  `data[i]` nor re-tests a bound the hit established.
- **The key scan seeds the value scan** (2026-08-22 — the 2026-08-17 parked
  "lookahead-seeded value scan", landed on amd64). The key loop's wide form
  holds a sixteen-byte view — `pair := data[i:i+16]`, one bounds proof, two
  constant-offset words — and an `=` hit answers the value question from it
  directly: `hasCtrlOrSpace(w)` locates the value's first stop with NO lane
  masking, because every lane below the `=` is a non-stop byte (ctrl-or-space
  is a subset of the key-stop set), the `=` itself is not flagged, and a
  borrow only sets spurious bits ABOVE a true match — so the mask's lowest set
  bit, if any, is the genuine stop. A miss consults the view's second word the
  same way; a double miss means 15−t clean value bytes are already proven and
  the ordinary value loop resumes at base+16. Values that resolve in-view —
  nearly everything in `DecodeKeyval`, most of `sampleTypical` — never enter
  the value loop at all, which takes that loop's load→mask→find chain off the
  per-field critical path: Iterate −13.1%, DecodeKeyval −16.7%,
  ParseAll_Typical −19.7%, LevelTS −2.9%, everything else `~` (2026-08-22
  benchmarks section; +4.5% instructions, −14% cycles, IPC 4.62 → 5.64).
  Four spellings are load-bearing:
  1. The view's SECOND word is loaded lazily, inside the `=` branch, only
     after the first word's value mask has come back empty — never beside `w`
     in the loop body. Loaded eagerly it costs the early-stop lookups
     (GetMany +2.5%, Extract +2.0%) and buys nothing anywhere: the
     out-of-order window hides an L1 load issued at the hit just fine, so
     lazy dominates eager on every row (see the pass notes).
  2. `pair` plus constant-index halves, not `data[i+8:i+16]` — the latter's
     IsSliceInBounds survives the prove pass (the `i+8` lower bound is one
     derivation too far), where `pair`'s own check folds exactly as
     `data[i:i+8]`'s does.
  3. The unquoted value scan now sits at section-level labels
     (`valLoop`/`valTail`/`valStop`/`valEnd`) so the seeded dispatch can join
     the ordinary stop-verify instead of duplicating it, and `valLoop`'s head
     carries the AppendUnescape `i >= 0` trick: i is now a phi of the
     straight-line entry and the seeded `base+16`, one merge more than prove
     follows, and without the test the value load pays a per-word bounds
     check. The narrow key loop below needs the same test for the same
     reason (its entry i is a phi of the wide loop's exits).
  4. A narrow single-word key loop covers the record's last 8..15 bytes,
     where there is no sixteenth byte to view (it also re-finds a control
     byte the wide loop broke on — one wasted iteration on that rare path,
     none on the hot one).
  Separator runs and bare keys pay nothing new; since 2026-09-01 the seeded
  quoted path jumps past the `data[i] == '"'` re-test, and the seeded stops
  verify inline and jump straight to `deliver` (the shared `valStop` /
  `valEnd` labels serve only the value loop now). Fuzz seeds pin the view-boundary shapes: stops at base+15 and
  base+16, `=` at lane 7, an 8-byte key, a control byte in-view, and a field
  whose key scan starts inside the narrow loop's territory.
- **The scan constants are registers, read from a package `var` once per
  field** (2026-09-01; `swarRegs`, the `hasKeyStopR` / `hasCtrlOrSpaceR` /
  `hasQuoteOrBackslashR` helpers). Spelled as constants, each broadcast mask
  is a `MOVQ $imm64` re-executed at every word, and on Zen 4 that is an
  integer-ALU op competing with the mask arithmetic for the four ALUs that
  bound the scan loop (the 2026-09-01 section). Three loads per field from a
  never-written global replace three ALU ops per word. Two rules keep it a
  win: read them at the top of the FIELD, not once per call — held across the
  callback they are spilled at entry and reloaded through the store buffer,
  a stall on every call's first field — and keep the loop's live set small,
  because one more value held across the wide loop (a hoisted `n-8` beside
  `n-16` did it) makes the allocator spill the word itself inside the loop.
  `Test_Unit_SWARMasks` pins the register-fed helpers bit-for-bit equal to the
  constant ones for every byte value in every lane.
- **The three scan constants are loaded `sub, hi, xor`, and the order is
  load-bearing** (2026-09-22): it decides which registers the allocator has
  free when `Iterate`'s `BSF`s are placed, and a `BSF` waits for its
  destination's previous value. `xor, sub, hi` put the value scan's `BSF` on
  the register the '"' test loads a byte into: +37% on short fields. See the
  2026-09-22 pass; check with `bench/bsfdep.py`.
- **The escape walk steps over a pair with `m &= (m | -m) << 9`** (2026-09-22):
  four dependent operations per escape where `m &^= low | low<<8` took six,
  and the lane comes from `TrailingZeros64(m)` (proven non-zero by the loop
  condition, so no `CMOV`). The step relies on the lowest set bit of `m` being
  the backslash's lane t, so shifting "everything from bit 7 of lane t up" by
  nine keeps exactly the lanes from t+2 — lane 7 never gets here, it leaves
  the word first.
- **The mask tail is `& (hi &^ w)`, not `&^ w & hi`** (2026-09-01): with `hi`
  a variable the compiler keeps the association, `hi &^ w` is computed beside
  the subtractions instead of after the OR, and the chain to the `TEST` is a
  cycle shorter — −1.8% on `Iterate` by itself. With `swarHi` a constant the
  rewrite rules canonicalise it back (the 2026-08-17 note), which is why this
  was a no-op before.
- **The callback's slices carry no cap-zero mask** (2026-09-01). `data[lo:hi]`
  costs `MOV`/`NEG`/`SAR`/`AND` to avoid advancing the pointer when the cap is
  zero unless the prove pass can show `cap - lo > 0`, and its `Slicemask`
  folding takes only a NUMERIC limit, which `detectSliceLenRelation` derives
  only from an ordering of the shape `index <= len-K`. So the outer loop's
  condition carries `&& i <= n-1` — implied by `uint(i) < uint(n)`, compiled
  to nothing, and exactly the shape that folds the key slice's mask — and the
  callback test carries `vStart > n-1 ||`, a real fused compare (vStart is a
  merge of three paths, beyond what prove follows) that folds the value
  slice's. Eight ALU ops per field for one. Check with
  `-d=ssa/prove/debug=1`: five `Proved slicemask not needed (by limit)` lines.
- **`lim16 := n - 16` is hoisted** (2026-09-01) and `n-8` deliberately is not:
  the hoisted one saves the `LEA` the compiler re-executed every word (Go does
  no loop-invariant code motion), the second costs a register the wide loop
  cannot spare (above). The 2026-07-27 rejection of a hoisted bound was about
  bounds checks, which the `uint(i) < uint(n)` head now supplies either way.
- **`data = data[:len(data):len(data)]` at the top of `Iterate`.** Originally
  for bounds-check elimination (worth ~1.1% then); since the 2026-07-27 evening
  pass the loop spellings below carry that role, and the cap-pin stays for the
  documented contract tightening (a callback's `append` cannot reach past the
  record).
- **`uint(i) < uint(n)` outer loop + `i <= n-8` SWAR loop heads** (2026-07-27
  evening). The unsigned compare is `i < n` (i is never negative) plus the
  `i >= 0` fact the prove pass otherwise never has; `i <= n-8` proves
  `i+8 <= n` without materialising an add that could overflow. Together they
  kill the IsSliceInBounds overflow check that had survived the cap-pin on
  both SWAR loads (~2 instructions + 2 never-taken branches per iteration),
  plus the bare-key-tail slice check and the separator-drain IsInBounds.
  Neither half works alone: uint-outer alone leaves the overflow check, and
  `n-8` alone (without `i >= 0`) removes nothing and adds an op — which is
  exactly why the old "hoisted `lim := n - 8`" attempt measured worse (see
  Rejected, now superseded).
- **The backslash-run walk before a closing quote has no lower-bound guard**
  (2026-07-27 evening): `data[vStart-1]` is the opening quote, any earlier
  escaped quote inside the value is also `"`, and neither is a backslash, so
  the run self-terminates and `j >= vStart` was semantically dead. `bs&1`
  replaces `bs%2` — the compiler cannot prove `bs >= 0` across the loop phi
  and spells `%2` as a 6-op signed-modulo dance. Measured together with the
  loop-bounds change as one pass (n=8 pinned interleaved rounds, control
  clean): geomean −1.0% (root suite) / −2.0% (bench suite); ParseEscaped
  −3.3%, GetMany −3.4%, Extract −3.4%, Iterate −1.3%, LevelTS −1.4%,
  ParseAll_Typical −0.6%, ParseAll_Big/Unescape `~` — and **DecodeKeyval
  +1.2% (p=0.000), the one accepted regression**: the first pass to trade the
  synthetic worst-case shape for the realistic suite. A variant keeping the
  old value-loop head halved that cost (+0.6%) but gave back roughly half the
  GetMany/Extract/LevelTS wins and lost on both geomeans.
- **`binary.LittleEndian.Uint64(buf[i:i+8])`** (fixed-size slice, not `buf[i:]`)
  — this single change was a large win (337 → 278 ns): it lets the compiler emit
  a tighter load. Keep the `i+8` slice form.
- **`key=` before whitespace needs no explicit branch.** Whitespace is not `"`,
  so control falls into the unquoted scan, which stops on the first byte and
  leaves `vEnd == vStart` — the same empty value the explicit test produced,
  one branch and one `isSpace` cheaper on every field.
- **`isSpace` is a 256-byte table lookup**, not arithmetic — measured faster
  twice now. The second time (2026-07-27) was against a *branchless* spelling,
  `(b == ' ') != (b-'\t' <= '\r'-'\t')`, which compiles to SETcc/SETcc/XOR with
  no branch and no memory reference. It was introduced to free the register the
  table's base address occupies, back when the SWAR masks were being held in
  registers; the XOR fold made that unnecessary and the table won on its own
  merits: **`DecodeKeyval` −1.29% with the table vs +2.12% branchless**, for
  ~0.3% on the other benchmarks (n=20, control `~` at +0.13%). Two instructions
  beat six when nothing is competing for the register.
- **Verify-order**: at SWAR stop points, test the cheap expected byte first
  (`c == '=' || isSpace(c)` for keys, `c == ' ' || isSpace(c)` for values) so the
  common case short-circuits past the `isSpace` table load. `IsAbsent`/nil-style
  short-circuits similarly in `GetMany`.
- **`GetMany` uses `buf` itself as the found-marker** (slots start `nil`, a match
  fills them) — no parallel bitmask. Raw aliasing makes it zero-alloc and
  found-values are never nil, so `nil` == absent unambiguously.
- **`GetMany`'s per-call setup is a single loop over `keys`** (2026-09-09):
  it nils the slot and ORs the key's length into `lenMask` in the same pass.
  Written as `clear(buf)` plus a second walk it was a call to
  `runtime.memclrNoHeapPointers` — `[][]byte` holds pointers, so `clear` does
  not compile to anything cheaper — and 12% of `GetMany`'s own profile between
  them. Fusing is **−4.8%** on `GetMany` and −4.3% on `Extract_Mine`, which is
  most of what a two-key lookup on a 1.4 KB line has left to give: the parse
  side of that row measured `~` in the same series.
- **Closing-quote verify tests `' '` first** (`c != ' ' && !isSpace(c)`) — same
  short-circuit trick as the SWAR verifies; ~1% on quoted-heavy lines.
- **Escape-dense values: two scans, chosen by how far apart the escapes are**
  (2026-08-17, revised the same day — see the correction note below). The quoted
  branch still opens with `bytes.IndexByte('"')` plus the backslash-run parity
  walk, and a value with no escaped quote never leaves that path. When the walk
  says *escaped*, `scanQuotedEscapeDense` takes over: it walks forward a word at
  a time with `hasQuoteOrBackslash`, consuming each `\` together with the byte
  it escapes — stepping over the pair IS the parity rule, so no walk is needed
  inside it — and a `"` it reaches is the closing quote by construction.
  **Since 2026-09-09 it drains a word of every escape before loading the next**
  rather than re-anchoring on each one, which is worth 25–49% wherever escapes
  are 8 bytes apart or closer (and 6–12% the other way at 32–48 byte gaps; see
  that pass's section for the trade, and for why `escClean` = 4 is not the
  answer to it). Two consequences are load-bearing and are commented at the
  function: the drain reads mask bits ABOVE the lowest one, where a spurious
  lane can sit, so the byte is re-checked for `\` as well as `"`; and a
  backslash in lane 7 escapes a byte outside the word, so that case resumes at
  `base+9`. The re-check is spelled `word[t]` against an 8-byte sub-slice with
  `t` masked to 3 bits, which is what keeps it bounds-check free — `data[base+t]`
  is not, because the prove pass will not combine `i <= n-8` with `t`'s range.
  It declines the job two ways, both handing back to `scanQuotedSparse` (the
  `IndexByte`+parity loop, out of line): **`escGap` = 48**, the first escape sat
  more than 48 bytes into the value, so the escapes are sparse and `IndexByte`'s
  stride wins; and **`escClean` = 5**, five consecutive words went by with
  neither byte in them. (Those are the amd64 values, in `scan_other.go`. Since
  2026-09-23 arm64 has its own walk and 64/8 in `scan_arm64.go` — and a sparse
  scan that can hand a value back, once; see the 2026-09-23 pass.) Handing back carries nothing, because the parity rule is
  context-free. The crossover behind both is `IndexByte` scanning clean bytes
  ~4.5× faster than the walk but costing a call per escape.
  **The 4.5× ratio travels between machines; the crossover does not** — it also
  depends on the per-escape call cost, which moves relative to `IndexByte`'s
  throughput. Measured 4.5× on arm64 (26.5 vs 5.9 GB/s) *and* on amd64 (~60 vs
  13 GB/s, Ryzen 8840HS), yet the crossover is ~70 bytes on the first and
  **32–40 bytes** on the second. An earlier version of this bullet said the
  ratio alone set these constants; that is wrong for `escClean` (below), right
  for `escGap` and `unescWindow`, both of which re-measured the same on x86.
  `scanQuotedSparse` returning −1, and its unsigned loop head, are what
  make the `i == n+1` hand-back (a trailing backslash stepped over) report an
  unterminated value instead of panicking — pinned by
  `testdata/fuzz/FuzzIterateAgainstRef/ee6d5b3abecfadf7` and by
  `Test_Unit_Quoted_EscapeDense_Scan`.
  **The correction:** this landed first (`b28092c`) as a single always-entered
  probe with a 32-byte window and no `escGap`, tuned on `Benchmark_IterateEscaped`
  and an embedded-JSON line alone. `sample_big.txt`'s two escaped quotes are
  **38 bytes** apart, just outside that window, so every parse of the shared
  sample paid a failed probe *and* a restarted `IndexByte`: **GetMany 7.3% and
  LevelTS 7.2% slower than it had to be**, invisible in that pass because the
  quoted-bit protocol landing beside it more than covered the loss. Tune this
  axis on the realistic line, not only on the synthetic sweep — the sweep has no
  point between 32-byte and 8-byte gaps, which is exactly where real logfmt sits.
  `AppendUnescape` keeps the same shape with its own, smaller window
  (`unescWindow`, above).
  **`escClean` 8 → 5 (2026-08-17, amd64 review).** Eight words is a 64-byte
  clean run, nearly twice the amd64 crossover, so a value with escapes 48 bytes
  apart was walked end to end where `IndexByte` was 17% faster. Five is forced
  from both sides rather than fitted: the largest value that gives up at the
  crossover, and the smallest that keeps `sample_big.txt` on the walk — that
  value's clean run is **exactly four words**, which `escClean` = 4 proves by
  dropping it off the dense path for **GetMany +11.9% and LevelTS +14.8%**
  (the same failure mode as the 32-byte window above, found the same way). At 5:
  the 48-byte-gap shape −8.6%/−10.0% in two series, every other row `~`, control
  clean. 6 is indistinguishable from 8 (at a 48-byte gap the walk needs 6 clean
  words, so only ≤5 gives up). **This is an x86-only result so far**, in a band
  neither machine's committed sweep sampled; it wants one confirming N2 run.
  Note the blind spot cuts both ways now — the committed `IterateEscaped` rows
  read `~` for this change, which is why `Benchmark_IterateEscapedGap` was added.
- **`if m := hasKeyStopR(w, …); m != 0 {`, not `m := …` on its own line** (2026-08-17).
  A call to an inlined function that is alone on its source line leaves the
  compiler's inline mark with no real instruction to attach to, and it becomes a
  literal `NOOP` (`HINT $0` on arm64, `XCHGL AX, AX` on amd64) — one per SWAR
  iteration, in both scan loops. Putting the compare on the call's line gives
  the mark a home. Measured neutral (the slot was free), kept because it is the
  same source shape and a cleaner loop; note the mark can come back if the
  first statement of the `if` body changes (a `:=` declaration there did it in
  one variant — check with `go tool objdump -s 'logfmt\.iterate$' | grep NOOP`).
  **Priced 2026-09-01: a NOP costs nothing here.** The seeded scan had put one
  back in the key loop (the inlined `Uint64` call's mark, with `pair[0:8]`
  leaving its line no instruction), it was removed by spelling the load
  `data[i:i+8]` — whose `i+8` CSEs with the loop increment and gives the mark
  a home — and the change measured exactly zero, because the loop is bound by
  the integer ALUs and a NOP never occupies one. Keep loops NOP-free for
  tidiness; do not expect a number from it.
- **`GOAMD64=v3` builds are ~1.5% faster** — BMI's TZCNT helps the SWAR
  `TrailingZeros64`. A user build flag, not something the module can set; noted
  in the README. **Re-measured 2026-08-17** on go1.26.5 (Ryzen 8840HS, n=6
  pinned interleaved): Iterate 230.8 → 227.6 ns = 1.37%, geomean 1.67%,
  GetMany/LevelTS/DecodeKeyval all `~`. The earlier figure — "~3–4%", Iterate
  3.0% / GetMany 2.7%, measured 2026-07-26 on the same machine class — no longer
  reproduces; roughly half of it has gone, presumably to codegen changes between
  toolchains. Still positive, still worth mentioning to users, but don't quote
  the old number.

## The two superseded branches (read before re-deriving their findings)

`perf/escape-scan-and-adapter` (PR #1) and `perf/escape-dense-quoted-scan`
(PR #2) attacked exactly this pass's two ideas, a few days earlier, and were
open while `b28092c` was pushed. Neither merges any more (both branch off
`90c6c75` and conflict in `logfmt.go`, `CLAUDE.md`, `README.md` and the tests),
and everything measurably better in them is now in main — but their **x86
measurements are the only ones this repo has for these changes**, since the
current machine is arm64 only. Keep the branches or their PR bodies reachable
rather than re-deriving:

- PR #1 is where `escGap`/`escClean` and the `scanQuotedEscapeDense` /
  `scanQuotedSparse` split come from; main's version is that design with this
  pass's `AppendUnescape` probe and set-only quoted protocol on top, and it
  beats the branch on every row it differs on.
- PR #2's rejected alternatives, measured on an Intel Core Ultra 9 185H:
  an always-dense scan is −32% at 8-byte gaps but **+48% at 128-byte gaps** and
  +6.4% on GetMany; a 32-byte entry gate costs the realistic escaped line
  **+7.3%**; and **inlining the dense scan into `iterate` costs GetMany 5.1%
  and Extract 3.1%** on that machine, where hoisting it out costs 2.7% on this
  one. Treat the placement of that scan as machine-dependent.
- PR #1's body reports the 2026-08-08 adapter hop as **invisible on x86**,
  where it measured +2.3% Iterate / +4.4% LevelTS here. Both can be true; it
  means the set-only protocol's win is an arm64 result until someone re-runs it
  on x86, and neither result should be quoted as universal. (The one x86 data
  point since — the CI amd64 table at `b28092c` — has `Iterate` −4.9% and
  `LevelTS` −3.8% for the protocol and scan together, so nothing there
  contradicts it either way.)

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
  `buf[i]`): neutral — the reload is an L1 hit the CPU pipelines. Re-tested
  2026-07-27 evening under the changed-circumstances rule (it also removes a
  real per-field IsInBounds the prover cannot drop): still no — DecodeKeyval
  +1.1% (p=0.002), all else `~`; the shift-chain dependency costs what the
  eliminated check saves. **Third time, 2026-09-01**, prompted by the chain
  probe that showed the mask→verify chain fully exposed: at GOAMD64=v1 the
  variable shift wants its count in CL, the allocator spilled `base` and `w`
  to free it, and the reloads landed on the very chain it was meant to
  shorten. Not on this toolchain.
- **16-byte unrolled key scan**: no change (loop overhead wasn't the bottleneck;
  it's memory-latency bound).
- **Arithmetic `isSpace`**, **combined key-stop lookup table**, **`len(buf)` in
  the loop bound for BCE**: neutral or worse. The arithmetic one has now lost
  three times, in three spellings — the branchless XOR (2026-07-27), the
  `wsMask` bit test (2026-07-27 evening) and a plain branching
  `b == ' ' || b-'\t' <= '\r'-'\t'` (2026-09-09, geomean +0.03%). The last was
  tried for a different reason than the first two — the compiler materialises
  the table's base with a LEA even on the `c == ' '` fast path that never
  indexes it — and its result is really a statement about the cost model: an
  ALU op that is not on the per-field chain does not show up.
- **Inlining the parser into `GetMany`** (drop the callback indirection): only
  ~4.5% and it duplicated the parser — the prototype immediately diverged on
  bare keys under differential fuzz. Not worth the duplication/risk. The
  ceiling was re-measured on 2026-09-09 by short-circuiting the call itself
  (`fn` never invoked, slices never built): `Iterate` −4.2%, `DecodeKeyval`
  −1.8%, i.e. **1.3 cycles per field** for the indirect call, both slices and
  the spills around them. Whatever a callback-free lookup is worth, it is less
  than that.
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
  **Superseded 2026-07-27**, and the reason is instructive: the idea was right
  and the *spelling* was wrong twice over. Make the step unconditional (`n+1`
  fails every bound exactly as `n` does, so no branch) **and** delete the
  whitespace-skip loop it makes redundant, rather than paying for both. See
  "No per-field whitespace-skip loop" above. A half-applied optimization can
  measure worse than not applying it at all — which is exactly what happened
  here, and it kept the idea parked for a release.
- **Sourcing the SWAR broadcast masks from package-level `var`s** so the
  register allocator has to keep them in registers instead of rematerialising
  `MOVQ $imm64` at every use (four per 8-byte key-scan iteration, ~a fifth of
  the loop). It does exactly what it promises in the disassembly — the masks
  live in R9–R12 and the loop drops to 16 instructions — but it is **neutral to
  negative** once the XOR fold has cut the mask count to three: holding them
  costs a spill/reload around every `fn` callback, which short-field workloads
  cannot amortise. Measured head to head (n=8, control clean): plain consts beat
  package vars on `ParseAll_Typical` (−7.18% vs −4.65%), `ParseEscaped`
  (−6.38% vs `~`) and `DecodeKeyval` (+1.76% vs +2.56%), and tie elsewhere.
  Fewer constants beat pinned constants. Don't reach for the `var` trick until
  you have first tried to need fewer values.
  **Superseded 2026-09-01, and the finding above was right about the wrong
  thing.** The cost was never "holding them" but where they came from: loaded
  once per call, the values must survive the callback, so they are spilled at
  entry and every field reloads them through the store buffer. Read from the
  global at the top of every field instead they never cross the call, and
  they are the largest single win this parser has had on amd64 since the
  seeded scan (`Iterate` −8.7% with the rest of the pass; the constants alone
  were −5.5%). The mechanism the 2026-07-27 note lacked is that a `MOVQ
  $imm64` is an integer-ALU op on Zen 4 and the loop is ALU-bound. Third
  instance of the valEnd lesson (right idea, wrong spelling), and the most
  expensive one.
- **A shrinking sub-slice window** (`s := data[i:]; for len(s) >= 8 { …;
  s = s[8:] }`) to get bounds-check elimination: Go emits a *conditional*
  pointer advance for `s[8:]` (`MOV`/`NEG`/`SAR`/`AND`/`ADD`) because it must
  not advance the pointer when the result's capacity reaches zero. Five
  instructions to remove two. `binary.LittleEndian.Uint64(data[i:])` pays the
  same dance, which is the mechanism behind the `data[i:i+8]` win above.
  `data = data[:len(data):len(data)]` is the cheap way to get the same check
  eliminated — one instruction, once per call.
- **A hoisted `lim := n - 8` loop bound** replacing `i+8 <= n`: more
  instructions, not fewer; the compiler already folds the `i+8` form well.
  **Superseded 2026-07-27 evening**: the idea was right and half-applied —
  `i <= n-8` pays off only when the outer loop's `uint(i) < uint(n)` supplies
  the `i >= 0` fact, and the pair (and only the pair) eliminates every bounds
  check in both SWAR loops. Second instance of the valEnd lesson.
- **Inline first-word `hasByte(w,'"')` before `IndexByte` in the quoted scan**
  (to spare short quoted values the call overhead): −3% on `Iterate` (long
  quoted values pay the wasted word check) and neutral on `DecodeKeyval` —
  the short-quote saving never materialised.
  **Priced properly 2026-09-09, and the reason no probe length works is the
  crossover, not the tuning.** Chained so the next call's argument depends on
  the last result — which is the parser's situation, since the value's end
  gates everything after it — `bytes.IndexByte` costs a flat ~4.7 ns up to a
  56-byte match on this machine, against 3.7 ns + 0.078 ns/byte for a called
  SWAR scan: the call is worth about **18–24 bytes** of SWAR, so a probe long
  enough to catch the sample's 33-byte quoted values loses more on its 203-byte
  one than it wins on the four short ones. The extreme (SWAR only, no
  `IndexByte` at all in the quoted branch) measures `LevelTS` −3.8%,
  `DecodeKeyval` −2.5%, `IterateJSONMsg` −3.5%, `GetMany` −1.3% and `Iterate`
  **+7.7%** — which is also the measurement that caps the whole call's cost and
  exposes the profiler's 3.3× over-attribution (see Methodology).
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

- **2026-07-27 evening pass — measured and rejected** (multi-agent ideation +
  three adversarial skeptics, then serial pinned A/B; every item below was
  correctness-verified and fuzz-clean before losing on the stopwatch):
  - *Split `fn` call sites for the quoted/unquoted value paths* (kill the
    kStart/kEnd spill round-trip around the callback): Extract +8.4%,
    DecodeKeyval +3.8%, Iterate +2.6%, ParseAll_Big +2.9% — code growth beats
    the saved store-forwarded spills, decisively.
  - *GetMany inline byte-compare* replacing `string(k) != keys[j]` (+ a
    prover-guard to BCE the closure): GetMany +7.0%, Extract +6.9%.
    `runtime.memequal` is SIMD; a byte loop over realistic key lengths loses
    even after deleting the call and shrinking the closure frame 0x58→0x20.
  - *AppendUnescape fused chunk+escape append* via a 256-byte self-map table:
    verifiably fewer instructions, measured `~` everywhere (Unescape 17.12 →
    17.12 ns). Escapes are too rare on real shapes for the emit path to matter.
  - *wsMask bit-test verify* (`(wsMask>>c)&1` for the stop-byte whitespace
    test — valid because SWAR stop bytes are provably < 64): no significant
    wins, Extract +4.5%. The two-instruction spaceTable load keeps winning
    (third spelling of "arithmetic isSpace" to lose to it).
  - *First-probe backslash unroll* (`if data[i-1] == '\\'` before the run
    walk): ParseEscaped −1.9% but LevelTS +2.2% and DecodeKeyval leaning
    positive — the added branch in the quoted path costs more than the probe
    saves. Superseded by the guard-drop spelling that landed (same
    ParseEscaped win, no new branch).
  - *AppendUnescape 8-byte word-copy for short inter-escape chunks*: killed on
    correctness before measurement — it writes into `dst[len:cap]`, which
    corrupts an in-place `AppendUnescape(raw[:0], raw)` decode.
  - *Seeding the value scan from the key word's remaining mask bits*
    (mask-carry across `=`): parked unmeasured. A scratch prototype verified
    correct, but it restructures the whole value state machine for a gain the
    cost model caps at ~1 word-load per short field, and the split-callsites
    result above is a fresh warning about code-growth effects in exactly that
    region. Revisit only with the prototype under differential fuzz and a
    control-clean series.

- **2026-08-17 pass (Neoverse N2, arm64) — measured and rejected.** Every item
  was correctness-verified (suite + fuzzers) before losing on the stopwatch;
  the harness had a clean A/A control and ±0% run-to-run variance, so these
  are real, but note the layout caveat in Methodology: on the `IterateEscaped/*`
  and `Unescape` rows a pure code-layout shift moves numbers ±2%, on the four
  core rows ±0.2%.
  - *Holding the two non-bitmask-immediate SWAR constants (`0x2121…`,
    `0x1d1d…`) in registers via package-level `var`s loaded once per
    `iterate`.* On arm64 each is rematerialised as `MOVZ`+3×`MOVK` (four
    instructions apiece, eight per key-scan iteration; amd64 pays one `MOVQ`
    each) and the var trick cuts the key loop from 21 to 12 instructions.
    Standalone: Iterate −2.75%, GetMany −1.5%, LevelTS −1.0%, **DecodeKeyval
    +1.2%**; on top of the quoted-bit protocol only Iterate −1.2%, GetMany
    −0.8%, DecodeKeyval +0.6%. Instructions −14%, cycles −0.4%: the
    rematerialisation was free filler (see the perf-stat paragraph in the
    benchmarks section). Not worth an arm64-only build-tagged pair of files
    for ~1%, and the amd64 measurement from 2026-07-27 was already negative;
    parked. If someone wants it: `//go:build arm64` file with `var`, other
    file with `const`, same names, `sub, xor := …` at the top of `iterate` and
    pass them into the mask helpers.
    **Landed 2026-09-01 for both arches, without the build tag** — see the
    `swarRegs` bullet in the optimization notes: the amd64 measurement was
    negative because of the entry spill, not the idea. Unmeasured on arm64
    since; the −2.75% / −1.2% above was the "once per call" form.
  - *Bounds-check-free post-hit byte load* — `word := data[i:i+8]`, `off :=
    TrailingZeros64(m)>>3 & 7`, `c := word[off]` — really does remove the
    `CMP/BLS` after each SWAR hit and compiles the `>>3 & 7` to one `UBFX`, but
    the word pointer materialises inside the loop (+1 instruction per
    iteration) and the inline-mark NOOP returns: **Iterate +2.3%**, LevelTS
    +0.9%, rest `~`. Predicted-never-taken checks are free here; loop-body
    growth is not. (Same lesson as the 2026-07-27 register-extract item.)
  - *`*quoted = q` stored before every callback* (the obvious out-parameter):
    Iterate −1.7% but GetMany +1.8%, Get/GetQuoted worse — a wash (geomean
    −0.1%). Superseded by the set-only protocol.
  - *Gating the escape probe on the PREVIOUS gap being short* (`dense :=
    i-esc <= 32` in the parser (`escWindow` as it was then), `dense = j-i < 32` in
    `AppendUnescape`): fixes the sparse synthetic row (`esc=8` +28% → +7%) but
    `Unescape` +11% (its two escapes are close together after 118 bytes of
    prose, and the gate says "sparse" from the first gap) and the realistic
    prose shape — `\"word\"` pairs, close together even when the pairs are far
    apart — loses the probe on exactly the quote it would have found. Rejected,
    and note what replaced it: gating on the distance to the value's *first*
    escape (`escGap`) is a different question with a different answer, because
    it is asked once per value rather than once per escape.
  - *Hoisting the probe out of `iterate` into a helper*, on the theory that the
    GetMany/LevelTS gap against the branch in PR #1 came from the hot loop
    growing: GetMany +2.7%, LevelTS +2.2%, `esc=8` +4.5%, geomean +1.25%. It
    does not, and the gap was the gating (`escGap`) instead. Worth knowing that
    out-of-lining this path costs rather than saves here, since PR #2's notes
    report the opposite on x86 (inlining its dense scan cost GetMany 5.1%
    there) — the two machines disagree, so re-measure before moving it again.
  - *Decoder window of 8 words* (`unescWindow`, matching the parser's
    `escClean`): `UnescapeEscaped/esc=8` +18.5%, geomean +2.0% across the
    decode benchmarks, no row gained; reproduced on a rerun. Four stays.
  - *Byte-loop copy of literal runs ≤ 16 B in `AppendUnescape`* instead of
    `append(dst, raw[i:j]...)` (a `memmove` call): JSON −3%, `Unescape` +11%.
    Rejected. (The 2026-07-27 word-copy variant remains killed on correctness:
    it corrupts an in-place decode; the new unescape fuzzer now checks that
    case.)
  - *Reassociating the mask tail* as `(w - c) & (swarHi &^ w)` to shorten the
    dependency chain by one op: the compiler canonicalises it back to
    `&^ w & swarHi` (identical codegen), so it is a no-op. Kept the readable
    spelling.
  - *Lookahead-seeded value scan* — load `data[i+8:i+16]` in the key loop as
    well, so that on the `=` hit the value's end can be found from the key
    word's remaining bytes and the already-loaded next word without waiting on
    a load whose address depends on the hit (`hasCtrlOrSpace` shares the
    `w - 0x2121…` term with `hasKeyStop`, so the second mask is two extra ops
    per iteration) — is the one structural idea left with real upside: it
    would take the value hit's ~11-cycle chain off the per-field critical path
    for values that end within ~8–15 bytes of the `=`, i.e. most of
    `sample_big`'s. Best case by the chain-exposure measurement above is
    ~−10%, realistically half the fields qualify, and it costs a load plus ~7
    ops per key iteration, a tighter loop bound (`i <= n-16`) with its own
    tail, and a second entry into the value state machine. Given that +2 loop
    instructions measured +2.3% just above, and the split-callsites result
    from 2026-07-27, this is a coin flip with real complexity. **Landed
    2026-08-22 on amd64** — see that pass's benchmarks section and "The key
    scan seeds the value scan" in the optimization notes. The coin flip
    resolved cleanly because the cost side of this paragraph was avoidable:
    the second word does NOT need loading per key iteration (lazily, inside
    the `=` branch, it costs the loop nothing and loses no latency), and the
    "second entry into the value state machine" is a `goto` into the existing
    stop-verify, not a duplicate of it. The two predicted wins were real and
    the estimate was low — Iterate −13%, the short-field shapes −17..−20%.
    The width-to-absorb-it worry about the N2 shrank the same day:
    `4a289b9`'s arm64 tables read −12% Iterate / −15% DecodeKeyval
    (indicative single runs; the pass section has the caveats).

- **2026-08-17 amd64 review — measured and NOT landed** (the pass's three
  landed changes are in the benchmarks section):
  - *A sparse→dense upgrade in `scanQuotedSparse`*, closing the one-way `escGap`
    entry decision. **Prototyped, fuzz-clean, and it works** — −32% on the
    prefix-then-dense shape — but +3–4% on values whose escapes really are
    64–256 B apart, which makes it a bet on the input distribution rather than a
    stopwatch result. Full writeup, including the oscillation trap that must be
    avoided if it is revisited, is in Known functional limits above. Deliberately
    left out; the cliff is documented in `logfmt.go` and pinned by
    `Benchmark_IteratePrefixJSON`, so it cannot be lost.
  - *Deleting `escGap` and relying on `escClean` alone.* Fixes the cliff for free
    in code terms, but every sparse value then pays one wasted probe of up to
    `escClean` words before handing back: +6.2% at 64 B gaps, +6.0% at 96 B,
    +8.8% at 128 B. Rejected — the entry gate is earning its keep.
  - *`escClean` = 4 and = 6.* 4 drops `sample_big.txt` off the walk (GetMany
    +11.9%, LevelTS +14.8%); 6 is indistinguishable from 8 because a 48-byte gap
    needs 6 clean words. 5 is the only value that does both jobs — see the
    constant's own comment.
  - *`unescWindow` = 2 and = 8 on amd64.* 2: −11% at 128 B gaps but +18.8% at
    32 B and +4.3% on `UnescapeJSONMsg`. 8: +28.5% at 128 B gaps. 4 stands on
    both arches now.
  - *Three alternative spellings of the `AppendUnescape` probe bound* (`uint`
    outer head; `uint(s) <= uint(n-8)` with `n>=8` hoisted; precomputed limit
    with `s` as the induction variable). None removes the bounds check; the
    unsigned form is actively worse (26 instructions vs 23). Only the explicit
    `s >= 0` works — see that loop's comment.

- **2026-09-22 pass (counters and IBS) — measured and rejected**, each written
  up with its numbers in that pass's section: the 16-byte view for the seeded
  verify loads (the view pointer's `LEA` lands in the key loop; unlucky `BSF`
  in every constant order), `base + 1` precomputed (refolded by the compiler),
  the quoted scan started with the SWAR walk, an inline or out-of-line
  two-word closing-quote probe, `AppendValue` handing quoted values straight to
  `AppendUnescape` (with and without splitting the decode loop out), `\u` as an
  out-of-line fix-up, and `escClean` 6/7 / `escGap` 56/64. The quoted-path
  items share one lesson: a short quoted field's cost is `IndexByte`'s ABI0
  boundary (~25 cycles of latency on the per-field chain), and every way
  around the call so far charges the realistic 30–60-byte values more than it
  saves the short ones.

- **2026-09-23 pass (arm64, Neoverse N2) — measured and rejected**, each with
  its numbers in that pass's section: a SWAR probe for short quoted values
  (−38% on short quoted fields, +2–10% on the realistic line — the same verdict
  as on amd64, from a different machine), the key loop's test as a fused
  `TST`+branch (a wash), the view's second word from `base+7` or
  `data[base+8:base+16]` (a bounds check or a `MOV` for the `ADD` it saves),
  stop bytes verified from the register (+9–11% on short fields), the value
  mask moved up a lane to drop the `+1` (−0.3%), arithmetic `isSpace` and
  hoisted scan constants (no gain / +2–4%), and for the decoder a fixed 6- or
  8-word probe, the fast loop inside `AppendUnescape` itself, and probe
  constants reloaded per probe. Also the unrestricted sparse-to-dense hand-back
  (+28–34% on alternating gaps), which landed restricted instead.

The parser is **memory-latency / per-field-overhead bound**, not scan-throughput
bound (confirmed on arm64 with counters, see the 2026-08-17 benchmarks
section: IPC 4.4, no mispredictions, instruction count barely moves cycles).
Further wins require an API change (non-callback) or accepting a
correctness/maintainability cost. Don't chase sub-ns micro-ops; they read as
wins in `-count=1` runs but vanish when averaged.
**Revised 2026-09-01 for amd64:** on Zen 4 the scan loop is bound by the four
integer ALUs and short fields by the post-value chain, and "instruction count
barely moves cycles" is true only of instructions that are not ALU ops. What
does not count: bounds checks, NOPs, register moves, loads. What does: every
mask op, every `MOVQ $imm64`, every fused compare, and every cycle on the
chain from one field's value stop to the next field's first load. Whether the
N2 is the same — its IPC 4.4 sits below its width — is unmeasured.
**Sharpened 2026-09-09, and this is the number to reason with:** an instruction
inside a scan loop costs about **0.2 cycles**, an instruction on the per-field
dependency chain about **1**. Both were measured directly rather than modelled
— three instructions off the wide key loop is 89 words × 3 = −5.2% on
`Iterate`, while four instructions added to the per-field hit path is +4% on
`DecodeKeyval` — and together they say where the remaining time is. A field
costs ~32 cycles on this machine, of which the key scan is ~6 (2.1 cycles per
8-byte word, measured by halving the scan's stride), `bytes.IndexByte` for
quoted values ~4, the callback and both slices ~1.3 (measured by not calling
it: `Iterate` −4.2%, `DecodeKeyval` −1.8%, which also caps the long-parked
callback-free lookup loop at ~4%), and the remaining ~20 is one field's
dependency chain: load (5) → mask (3–4) → `BSF` (3) → `SHR` → `LEA` → the next
field's load. That chain is why moving work off the per-word path and onto the
per-field path keeps losing, and why the two wins this pass found are both in
code that had a chain to shorten rather than instructions to shed.
**Refined 2026-09-22 with port measurements:** not every ALU op is equal. A
fused compare-and-branch, a shift and a three-component `LEA` issue at 2 per
cycle on this core, plain ALU ops and `BSF` at 4, so branch ops are the
scarcer currency — ~21 of a seeded field's ~48 ALU ops are branches, 7 of them
bounds checks — and a `BSF`'s destination is a hidden input (the 2026-09-22
pass). The chain above is ~12 cycles for a short field whose value settles in
the view, measured at 15.2 cycles per field all told.
**The N2, measured 2026-09-23 (the "unmeasured" above):** a short unquoted
field costs ~19 cycles and ~73 ops, a sample field ~35 cycles and ~135 ops, at
3.9 ops/cycle against a ~5-wide dispatch with 20% of cycles backend-stalled.
Of a short field's 19 cycles, ~4 are the callback (−21% without it: `i`'s
store-to-load round trip is ~5 cycles here); the key loop costs 2.9 cycles per
word (26% of the sample). Neither regime of the Zen 4 rules holds cleanly:
cutting 2.5% of the ops (arithmetic `isSpace`) moved nothing, and cutting a
cycle off the value chain moved short fields 0.3%. What did pay on this core
was taking calls and their spills off hot paths (`memmove` per decoded run,
`IndexByte` per escaped quote), which is where every win of that pass is.

## Methodology (use this for any future perf work)

- **Counters before the stopwatch (2026-09-22).** `bench/counters.py` gives
  cycles, macro-ops, instructions and branches per op for any list of
  benchmarks across prebuilt binaries in seconds. Cycles ignore clock drift,
  so it reads a change on a machine too busy for a timing series, and the op
  counts say whether a cycle delta came from work or from latency: cycles
  moving while ops stand still is a dependency or a layout effect, and the
  first thing to check then is `bench/bsfdep.py`. It does not replace the
  interleaved timing series for a final claim; it decides what is worth one.
- **Counters on arm64 (2026-09-23).** The N2 VMs count the PMUv3 common
  events, and `counters.py` uses them there unasked: ops, branches,
  mispredicts and the two stall shares per op. The events that read 0 (IMPDEF,
  `STALL_SLOT*`) are the hypervisor's filter, not a perf syntax problem — raw
  codes (`r003a`) work for everything it passes. There is no SPE and no IBS: a
  `cycles:u` sample lands on the oldest unretired instruction, which is useful
  for finding a stalled chain but says nothing about why.
- **Change one architecture, prove the others unchanged (2026-09-23).** Put
  the arch-specific code in `_arm64.go`/`_other.go` files or behind a constant
  like `runtime.GOARCH == "arm64"` (removed before code generation where it is
  false), then run `bench/asmdiff.py` on binaries cross-compiled for the other
  architecture at the baseline and after. Zero package functions different is
  the claim; anything else needs that architecture's own measurement.
- **Check every BSF after touching `Iterate` (2026-09-22).**
  `python3 bench/bsfdep.py <test binary> 'logfmt\.Iterate$'`; a SUSPECT line
  means a `BSF` may wait on a data load it has nothing to do with.
- **IBS for "which instruction is late" (2026-09-22).** Needs root: run the
  benchmark pinned to core 10, and beside it
  `sudo perf record -C 10 -e ibs_op/cnt_ctl=1/ -R -c 20000 -o ibs.data -- sleep 2`;
  `perf report -i ibs.data -D` prints each sampled op's `IbsOpRip`,
  `TagToRetCtr` and `CompToRetCtr`, whose difference is its dispatch-to-
  completion latency. An op whose latency jumps between two builds with the
  same instructions is the bug. (Aggregate per address and join with
  `go tool objdump` — a 60-line script.)
- **Anchor every part of a sub-benchmark pattern.** `-test.bench` splits on
  `/` and matches each part on its own, unanchored: `Shape/quoted$` also runs
  `Shape/unquoted`. `^Shape$/^quoted$` runs one.

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
- **Budget the series** (maintainer preference, 2026-07-27): keep a full A/B
  series within ~10 minutes wall clock — 1 s benchtime with n=6–8 interleaved
  pinned rounds resolves ≥1% effects; escalate to 2 s / n≥10 only for a single
  ambiguous finalist, not for broad sweeps.
- **Profile cumulative + line-level**: `-cpuprofile`, then `go tool pprof -top
  -cum` and `-list=Iterate`. Beware skid: `isSpace` and verify-line "flat %" are
  often attribution of dependent-load latency, not removable work.
- **Keep only measured wins; revert neutral changes** for clarity.
- **Calibrate for code layout, not just for noise (learned 2026-08-17).** On a
  quiet arm64 VM the harness above resolves ±0.2% — good enough to see effects
  that are *real but not yours*: adding a never-called function after
  `iterate` (pure address shift, zero instruction change in any hot path)
  moved `IterateEscaped/esc=32` +2.0%, `esc=8` +1.2%, `Unescape` +0.7%, while
  `Iterate`, `GetMany`, `DecodeKeyval` and `LevelTS` stayed within ±0.2%. So:
  run that padding control once per machine, treat the layout-sensitive rows
  as ±2% no matter what benchstat's p-value says, and only believe sub-2%
  deltas on rows the control showed to be layout-stable.
  **The ±2% figure is arm64's; do not carry it to x86.** On the Ryzen 8840HS
  (2026-08-17) the same class of shift moved escape-density rows by up to
  **8.5%** — a constant change that altered nothing on a row's code path moved
  it that far — while `Iterate`/`GetMany`/`DecodeKeyval`/`LevelTS` again stayed
  layout-stable. Same lesson, larger number: the four core rows are the ones to
  decide on, and a single-digit delta on an `*Escaped/*` row means nothing
  without a mechanism. Note also that a busy desktop still resolves the large
  effects — the A/A control came back at −0.03% with a browser eating 60% of the
  machine — but it widens the per-row ± enough that sub-3% findings should be
  re-run somewhere quiet rather than believed. The 2026-08-17 recipe:
  `go test -c` once per tree, then a shell loop that runs the two binaries
  pinned (`taskset -c 1`) with `-test.count=1`, alternating which goes first
  each round, appending to two files for `benchstat`; each tree is a plain
  copy of the repo (`git archive` / `rsync`), and the bench module's
  `replace ../` makes copies self-contained. `perf stat -e
  cycles,instructions,branches,branch-misses` works in the VM for the generic
  counters (`taskset -c 1 perf stat … ./x.test -test.bench=…`), the IMPDEF ones
  do not; a dependent-add loop puts the clock at ~3.4 GHz.

- **Diagnose with counters before cutting instructions (2026-09-01).** Three
  numbers per op from `perf stat` — `cycles`, `ex_ret_ops` (macro-ops, the
  unit dispatch and the ALUs count in) and `instructions` — plus
  `de_no_dispatch_per_slot.backend_stalls` / `no_ops_from_frontend` and
  `ex_no_retire.not_complete`, pinned, `-benchtime=2s`, divided by the
  benchmark's iteration count. Then the two experiments that identify the
  regime in ten minutes: remove ~10% of the instructions that are NOT ALU ops
  (bounds checks, NOPs, register moves) and add two dependent ALU ops to a
  suspected chain. If the first moves nothing and the second moves cycles
  one-for-one, the loop is ALU-port-bound with an exposed chain, and only ALU
  ops and chain length count — which is this parser on Zen 4. A standalone
  micro-benchmark of the bare loop — a throwaway module that runs just the
  mask loop over `sample_big.txt`, OR-ing every word's mask into an
  accumulator, no field structure at all — gives the per-word cost directly;
  3.6 cycles per word for ~12 ALU ops is the whole story.
  `perf annotate` on `cycles:u` is still useful for WHERE (the samples land on
  the instruction after a stalled one); IBS needs root here.
- **Nothing else runs during a series (2026-09-01).** A fuzzer pinned to the
  other eight cores still put ±70–300% on the rows it overlapped (boost budget
  and the memory system are shared), and two series on two cores at once put
  every row at `~` ±10–15% where the same trees measured p=0.000 alone. Cores
  are not interchangeable either (core 12 ran the same binary ~5% slower than
  core 10); compare only within one core. Fuzz, build and lint between series,
  never during one.

- **`perf` attribution lies about `bytes.IndexByte` here — by 3.3× (2026-09-09).**
  A `cycles:u` profile of `Benchmark_IterateOur` puts **38.6%** of the parse in
  `indexbytebody` plus `internal/bytealg.IndexByte.abi0`, with the samples piled
  on the argument reloads in the ABI0 wrapper and on the `PUNPCKLBW` byte
  broadcast — both of them the instruction *after* something that stalls, which
  is where AMD's non-precise sampling puts them. Replacing every `IndexByte` in
  the quoted scan with SWAR (which is 35 ns of work on that line) made `Iterate`
  only 13.8 ns *slower*, so the call's whole removable cost is ≤ 11.7%, and the
  practical figure is lower still. Treat a Go asm leaf's share in these profiles
  as an upper bound with a large multiplier on it.
- **Price a scan loop in situ by halving its stride (2026-09-09).** `i += 8` →
  `i += 4` in the wide key loop is *correct* — overlapping windows still find
  the first stop — and it doubles the iteration count with every other
  instruction unchanged, so the delta over the known extra word count is the
  loop's real marginal cost (2.1 cycles per key word, 89 words = 20% of the
  parse). It beats both a profile and a standalone micro-benchmark, because the
  loop stays inside the function that feeds it. The same trick reversed —
  short-circuiting the callback with a condition that is never true — prices the
  delivery path.
- **A discriminating experiment beats another profile.** When two measurements
  disagree about where the time is, build the variant whose two hypotheses
  predict opposite signs (here: quoted scan with no `IndexByte` at all —
  −19% if the profile were right, +8% if the micro-benchmark was) and run it.
  Three of this pass's findings came from one such run each.

## Commands

```sh
go test ./...                                              # unit tests
go test -run='^$' -fuzz=FuzzIterateAgainstRef -fuzztime=20s # parser fuzz
go test -run='^$' -fuzz=FuzzGetManyAgainstRef -fuzztime=20s  # lookup state machine
go test -run='^$' -fuzz=FuzzAppendUnescapeAgainstRef -fuzztime=20s # decoder
go test -run='^$' -bench=. -benchmem -count=3             # benchmarks
go vet ./... && gofmt -l .                                # lint/format
make bsfdep                                               # BSF dependency screen of Iterate (amd64)
python3 bench/counters.py 'Benchmark_IterateOur,Benchmark_Get/level' \
    /tmp/before.test /tmp/after.test --cpu 10             # per-op cycles/ops/branches (AMD or Arm PMU)
python3 bench/asmdiff.py /tmp/before.test /tmp/after.test # which functions' machine code changed
```
