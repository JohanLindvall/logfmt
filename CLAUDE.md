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
  SWAR `iterate` against a byte-by-byte reference. **Run this after any change
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
  drives the parser through `iterateQ`, the test-side adapter that reads and
  resets the `*bool` out-parameter (see "The quoted-bit protocol").
  Also `Test_Unit_SWARMasks` (exhaustive: every byte value in every lane, for
  all four masks — `hasKeyStop`, `hasCtrlOrSpace`, `hasQuoteOrBackslash`,
  `hasBackslash` — plus "the lower of two stops wins") and `Test_Unit_IsSpace`.
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
- `unescape_fuzz_test.go` — `FuzzAppendUnescapeAgainstRef` (added 2026-08-17,
  when `AppendUnescape` gained its SWAR follow-up scan): a byte-at-a-time
  reference decoder spelled out independently, checked three ways — append to
  nil, append behind a prefix, and **decode in place** (`dst = raw[:0]`, legal
  because decoding never lengthens, and the one case a read-ahead can break) —
  plus `NeedsUnescape(raw) == false ⇒ raw decodes to itself`. **Run after any
  change to `AppendUnescape`.** Mutation-checked: swapping `\r` for `\n` in the
  decoder fails on the seeds alone.
- `*_test.go` — unit tests, benchmarks, and a regex-vs-logfmt comparison.
  `Test_Unit_HotPath_Allocs` pins the allocation-free contract across all 14
  entry points that claim one (previously only 2 were asserted, and three
  injected allocations left the suite green); `Test_Unit_Malformed_Allocs` pins
  the single carve-out. `Test_Unit_Lookups_CapValues` now asserts `Get` really
  aliases the input at a known offset — `cap == len` plus "append didn't touch
  the line" is satisfied by a heap copy, so the old form passed with `Get`
  copying. `Test_Unit_Unquoted_Backslashes_Are_Literal` pins the quoted/unquoted
  decode split. `Benchmark_IterateEscaped` sweeps escape density at fixed
  length; `Benchmark_IterateJSONMsg` / `Benchmark_UnescapeJSONMsg` (2026-08-17)
  are the realistic point on that axis — a structured event serialised into a
  `msg=` field, one escape per ~7 bytes — for the parser and the decoder
  respectively.
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

- `Iterate(data, func(k, v) bool) error` — exported adapter over the unexported
  `iterate`, whose callback takes a third `quoted bool` (see "The `iterate` /
  `Iterate` split" below). Calls back per pair,
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
  `IndexByte`; after each decoded escape it probes the next `escWindow` words
  inline with `hasBackslash` before calling `IndexByte` again (2026-08-17:
  `Unescape` −8.5%, the JSON `msg=` value −40%; see "Escape-dense values").
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
- Statement coverage is 99.6%, and the one uncovered statement is
  `parseUnixTS`'s defensive ParseInt error guard, which is unreachable (10
  digits cannot overflow int64). The value-scan SWAR control-byte break —
  formerly the other uncovered line, reachable on CI by no seed — is pinned by
  a fuzz seed and a unit case since 2026-08-06. All three differential fuzzers
  pass clean after the 2026-08-17 pass (`FuzzIterateAgainstRef` 90 s,
  `FuzzGetManyAgainstRef` 60 s, `FuzzAppendUnescapeAgainstRef` 60 s, no new
  failures).

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

Table currency, as of 2026-08-17: the committed tables are **current through
`b83eade`** (`90c6c75`, stamped 2026-08-08T16:36Z, is its direct child) and
**stale for the 2026-08-17 pass**. The CI arm64 table's `IterateOur 391.3 ns` /
`GetMany 81.4 ns` are byte-for-byte what the Neoverse N2 VM used for that pass
measures for the same commit, so the arm64 CI runner is that machine class and
the pass's ratios should reproduce there. Note `bench.yml` is
`workflow_dispatch` only, so nothing refreshes them automatically; run
`make bench-md` deliberately (dispatch the workflow). Don't hand-edit them with
laptop figures.

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

Before = `90c6c75` (HEAD), after = this pass; n=8 pinned (`taskset -c 1`)
interleaved 1 s rounds, order rotated per round; benchstat p=0.000 on every
non-`~` row:

| Benchmark | before | after | Δ | allocs |
|---|---:|---:|---:|---:|
| `Iterate` (sample2, 1.4 KB, 29 fields) | 391.4 ns | 382.3 ns | −2.3% | 0 |
| `ParseAll_Big_Mine` (bench/) | 391.6 ns | 383.4 ns | −2.1% | 0 |
| `ParseAll_Typical_Mine` (130 B) | 85.1 ns | 83.4 ns | −1.9% | 0 |
| `LevelTS` logfmt | 72.0 ns | 69.9 ns | −3.0% | 0 |
| `GetMany` / `Extract_Mine` | 81.3 / 81.5 ns | 80.4 / 80.2 ns | −1.0% / −1.6% | 0 |
| `DecodeKeyval` (10k short rows) | 726.1 µs | 720.1 µs | −0.8% | 0 |
| `ParseEscaped_Mine` (bench/) | 214.1 ns | 193.3 ns | **−9.7%** | 0 |
| `Unescape` | 28.3 ns | 26.0 ns | −8.5% | 0 |
| `IterateJSONMsg` (JSON in `msg=`, 28 escapes) | 301.5 ns | 171.0 ns | **−43.3%** | 0 |
| `UnescapeJSONMsg` (that value, decoded) | 318.3 ns | 191.8 ns | **−39.7%** | 0 |
| `IterateEscaped/esc=0` (1 KB clean) | 36.2 ns | 35.1 ns | −3.2% | 0 |
| `IterateEscaped/esc=8` | 107.8 ns | 137.7 ns | **+27.7%** (accepted, see below) | 0 |
| `IterateEscaped/esc=32` | 333.6 ns | 250.3 ns | −25.0% | 0 |
| `IterateEscaped/esc=128` | 1194 ns | 570 ns | −52.3% | 0 |
| `IterateEscaped/esc=500` | 4.72 µs | 2.17 µs | −54.0% | 0 |

Root geomean −21.0% (dominated by the escape rows); bench/ `Mine` geomean
−3.9%; the four layout-stable core rows (Iterate, GetMany, DecodeKeyval,
LevelTS) −0.8% to −3.0%, none regressed. Three changes landed: the quoted-bit
protocol (above), the escape-dense follow-up scan in the quoted branch and in
`AppendUnescape` (next section), and the `if m := hasKeyStop(w); m != 0`
spelling (drops a real `NOOP` per SWAR iteration on both arches; measured
neutral, kept for the cleaner loop). The `esc=8` row is the one accepted
regression: one escaped quote per 128 bytes pays a wasted four-word probe
(~3.7 ns) per escape; gating the probe on the previous gap being short fixed
that row (+7%) but cost `Unescape` +11% and the realistic prose shape
(`\"word\"` pairs are close together even when the pairs are far apart), so the
ungated form stayed. Both differential fuzzers plus the new unescape one pass
90 s / 60 s / 60 s clean; coverage 99.6% (the one uncovered statement is still
`parseUnixTS`'s unreachable guard).

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

### Cost model (measured 2026-07-26, synthetic field-size sweep)

**Not re-measured after the 2026-07-27 pass**, which attacked the fixed
per-field term specifically. `sample_big.txt` holds **29** pairs (counted, not
estimated), so that pass moved it from 358.1/29 = 12.3 to 308.8/29 = 10.6
ns/field; earlier notes here divided by a wrong field count and read ~7.7 to
~6.7, which is why the overhead row below is not to be trusted as-is. The scan rows are
unaffected (neither scan loop's inner shape changed for values). Re-run the
sweep before quoting the overhead figure again.

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
consuming each backslash with the byte it escapes, at ~4.3 ns per escape in the
densest case and no per-escape call at all; after `escWindow` (4) quiet words
it hands back to `IndexByte`. Embedded JSON in a `msg=` field — every JSON
quote becomes `\"`, one escape per ~2–8 bytes — is the realistic shape this
serves: `Benchmark_IterateJSONMsg` −43%, `Benchmark_UnescapeJSONMsg` −40%. At a
**fixed** 1 KB value the sweep now runs 35 ns (0 escapes) → 2.17 µs (500), about
60× (it was 36 → 4.72 µs, 130×, on this machine; 65 → 6100 ns, 90×, on the
Ryzen). `Benchmark_IterateEscaped` pins that axis; it exists because
`sample_big.txt` has 2 escaped quotes in 1.4 KB (~4%) and so makes the quoted
scan look like pure `IndexByte` throughput.

This is **not** the rejected "inline first-word `hasByte` before `IndexByte`"
item, which was about sparing short *clean* values a call — clean values still
never touch the SWAR path.

Reading: short fields are **overhead-bound** (~6 ns of loop/callback per pair,
scan is noise), long values are **scan-bound**. The 8 B/iter SWAR only starts
paying off above ~32 B — which is why the memchr2/SIMD experiments below lost.
`sample2` averages ~8.3 ns/field, consistent with the sweep.

## The quoted-bit protocol (2026-08-08 split, measured and reshaped 2026-08-17)

`iterate` reports whether a value was double-quoted — the only position where a
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
- **Closing-quote verify tests `' '` first** (`c != ' ' && !isSpace(c)`) — same
  short-circuit trick as the SWAR verifies; ~1% on quoted-heavy lines.
- **Escape-dense values: SWAR follow-up scan after the first escaped quote**
  (2026-08-17). The quoted branch still opens with `bytes.IndexByte('"')` plus
  the backslash-run parity walk — clean values never see anything else. When
  the walk says *escaped*, the loop steps past the quote and probes forward a
  word at a time with `hasQuoteOrBackslash` (two has-zero-byte tests OR-ed; the
  borrow caveat applies and is harmless as always): a `\` found means "skip it
  and the byte it escapes" (`i += 2`), a `"` found is the closing quote by
  construction (every backslash before it was consumed pairwise, so no walk is
  needed) and jumps to the shared `closingQuote:` label; a word with neither
  bumps `quiet`, and after `escWindow` = 4 quiet words the loop `continue`s to
  `IndexByte` from wherever it stopped — sound because the parity walk is
  context-free, so nothing needs carrying across. `i = min(i, n)` before that
  hand-back covers the one way `i` reaches `n+1`: an escaping backslash as the
  last byte of the input (pinned by unit case and seed). Window size was
  measured at 2/4/8: 4 wins — 2 loses the `esc=32` row (+17%) because a JSON
  string value longer than 16 bytes drops back to `IndexByte` every time, 8
  costs the sparse row +59% for no dense gain. `AppendUnescape` applies the same
  idea with `hasBackslash` after every decoded escape.
- **`if m := hasKeyStop(w); m != 0 {`, not `m := …` on its own line** (2026-08-17).
  A call to an inlined function that is alone on its source line leaves the
  compiler's inline mark with no real instruction to attach to, and it becomes a
  literal `NOOP` (`HINT $0` on arm64, `XCHGL AX, AX` on amd64) — one per SWAR
  iteration, in both scan loops. Putting the compare on the call's line gives
  the mark a home. Measured neutral (the slot was free), kept because it is the
  same source shape and a cleaner loop; note the mark can come back if the
  first statement of the `if` body changes (a `:=` declaration there did it in
  one variant — check with `go tool objdump -s 'logfmt\.iterate$' | grep NOOP`).
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
  `buf[i]`): neutral — the reload is an L1 hit the CPU pipelines. Re-tested
  2026-07-27 evening under the changed-circumstances rule (it also removes a
  real per-field IsInBounds the prover cannot drop): still no — DecodeKeyval
  +1.1% (p=0.002), all else `~`; the shift-chain dependency costs what the
  eliminated check saves.
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
  - *Gating the escape probe on the previous gap being short* (`dense :=
    i-esc <= escWindow*8` in the parser, `dense = j-i < escWindow*8` in
    `AppendUnescape`): fixes the sparse synthetic row (`esc=8` +28% → +7%) but
    `Unescape` +11% (its two escapes are close together after 118 bytes of
    prose, and the gate says "sparse" from the first gap) and the realistic
    prose shape — `\"word\"` pairs, close together even when the pairs are far
    apart — loses the probe on exactly the quote it would have found. The
    ungated form is also simpler. Rejected.
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
    from 2026-07-27, this is a coin flip with real complexity. **Not
    attempted; prototype under `FuzzIterateAgainstRef` with a control-clean
    series before landing.**

The parser is **memory-latency / per-field-overhead bound**, not scan-throughput
bound (confirmed on arm64 with counters, see the 2026-08-17 benchmarks
section: IPC 4.4, no mispredictions, instruction count barely moves cycles).
Further wins require an API change (non-callback) or accepting a
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
  deltas on rows the control showed to be layout-stable. The 2026-08-17 recipe:
  `go test -c` once per tree, then a shell loop that runs the two binaries
  pinned (`taskset -c 1`) with `-test.count=1`, alternating which goes first
  each round, appending to two files for `benchstat`; each tree is a plain
  copy of the repo (`git archive` / `rsync`), and the bench module's
  `replace ../` makes copies self-contained. `perf stat -e
  cycles,instructions,branches,branch-misses` works in the VM for the generic
  counters (`taskset -c 1 perf stat … ./x.test -test.bench=…`), the IMPDEF ones
  do not; a dependent-add loop puts the clock at ~3.4 GHz.

## Commands

```sh
go test ./...                                              # unit tests
go test -run='^$' -fuzz=FuzzIterateAgainstRef -fuzztime=20s # parser fuzz
go test -run='^$' -fuzz=FuzzGetManyAgainstRef -fuzztime=20s  # lookup state machine
go test -run='^$' -fuzz=FuzzAppendUnescapeAgainstRef -fuzztime=20s # decoder
go test -run='^$' -bench=. -benchmem -count=3             # benchmarks
go vet ./... && gofmt -l .                                # lint/format
```
