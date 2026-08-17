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
  two malformed shapes only the walk can reach.
  **Added 2026-08-17 by the amd64 review, all four to close blind spots that had
  each already cost a tuning pass:**
  `Benchmark_IterateEscapedGap` / `Benchmark_UnescapeEscapedGap` sweep the same
  axis as the two above but parameterised by the **distance between escapes**
  rather than by a count, with points at 16/32/40/48/64/128/256 — i.e. bracketing
  the decisions instead of sampling evenly. The count-parameterised sweep's blind
  spot has now hidden a regression (the 32-byte window, −7% GetMany) *and* a win
  (`escClean` 5, −10% at a 48-byte gap read as `~` on every committed row).
  `Benchmark_IteratePrefixJSON` pins the `escGap` cliff (see Known limits).
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
  both before it lands.
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
`6a79ce6`**, the escape-scan correction — `00d92ad` regenerated both of them at
14:11Z. (An earlier version of this paragraph said they stopped at `b28092c`
and were stale for the correction; that was written before `00d92ad` landed and
was already false when it was read. Check the `generated` stamp in the file
against `git log -- bench/` rather than trusting this line.) They are **stale
for the 2026-08-17 amd64 review** — `escClean` 5, the `hex4` table and the
`AppendUnescape` bounds-check hint all postdate them. The CI arm64 numbers
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
densest case and no per-escape call at all; it hands back to `IndexByte` when
the escapes turn out to be sparse (`escGap`/`escClean`, see the optimization
notes). Embedded JSON in a `msg=` field — every JSON quote becomes `\"`, one
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
- **Escape-dense values: two scans, chosen by how far apart the escapes are**
  (2026-08-17, revised the same day — see the correction note below). The quoted
  branch still opens with `bytes.IndexByte('"')` plus the backslash-run parity
  walk, and a value with no escaped quote never leaves that path. When the walk
  says *escaped*, `scanQuotedEscapeDense` takes over: it walks forward a word at
  a time with `hasQuoteOrBackslash`, consuming each `\` together with the byte
  it escapes — stepping over the pair IS the parity rule, so no walk is needed
  inside it — and a `"` it reaches is the closing quote by construction. It
  declines the job two ways, both handing back to `scanQuotedSparse` (the
  `IndexByte`+parity loop, out of line): **`escGap` = 48**, the first escape sat
  more than 48 bytes into the value, so the escapes are sparse and `IndexByte`'s
  stride wins; and **`escClean` = 5**, five consecutive words went by with
  neither byte in them. Handing back carries nothing, because the parity rule is
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
- **`if m := hasKeyStop(w); m != 0 {`, not `m := …` on its own line** (2026-08-17).
  A call to an inlined function that is alone on its source line leaves the
  compiler's inline mark with no real instruction to attach to, and it becomes a
  literal `NOOP` (`HINT $0` on arm64, `XCHGL AX, AX` on amd64) — one per SWAR
  iteration, in both scan loops. Putting the compare on the call's line gives
  the mark a home. Measured neutral (the slot was free), kept because it is the
  same source shape and a cleaner loop; note the mark can come back if the
  first statement of the `if` body changes (a `:=` declaration there did it in
  one variant — check with `go tool objdump -s 'logfmt\.iterate$' | grep NOOP`).
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
    from 2026-07-27, this is a coin flip with real complexity. **Not
    attempted; prototype under `FuzzIterateAgainstRef` with a control-clean
    series before landing.**

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

## Commands

```sh
go test ./...                                              # unit tests
go test -run='^$' -fuzz=FuzzIterateAgainstRef -fuzztime=20s # parser fuzz
go test -run='^$' -fuzz=FuzzGetManyAgainstRef -fuzztime=20s  # lookup state machine
go test -run='^$' -fuzz=FuzzAppendUnescapeAgainstRef -fuzztime=20s # decoder
go test -run='^$' -bench=. -benchmem -count=3             # benchmarks
go vet ./... && gofmt -l .                                # lint/format
```
