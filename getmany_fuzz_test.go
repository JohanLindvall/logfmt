package logfmt

import (
	"testing"
)

// getManyRef is an obviously-correct reference for GetMany's semantics: for
// each key, the first non-empty occurrence wins; failing that, the first
// (empty) occurrence; failing that, nil. It collects every pair first, with no
// early-stop or slot state machine.
// It also reports, per key, whether the winning value was quoted — the bit
// AppendValue needs to decide whether decoding escapes is correct at all.
func getManyRef(data []byte, keys []string) ([][]byte, []bool) {
	type pair struct {
		k, v   []byte
		quoted bool
	}
	var pairs []pair
	// The error is deliberately ignored: iterate delivers every pair preceding
	// a fault, and that valid prefix is exactly what the lookups see too.
	_ = iterate(data, func(k, v []byte, quoted bool) bool {
		pairs = append(pairs, pair{k, v, quoted})
		return true
	})
	out := make([][]byte, len(keys))
	quoted := make([]bool, len(keys))
	for j, key := range keys {
		for _, p := range pairs {
			if string(p.k) != key {
				continue
			}
			if len(p.v) > 0 {
				out[j], quoted[j] = p.v, p.quoted
				break
			}
			if out[j] == nil {
				out[j], quoted[j] = p.v, p.quoted // provisional empty; keep looking
			}
		}
	}
	return out, quoted
}

// FuzzGetManyAgainstRef checks GetMany's early-stop/provisional-empty state
// machine (and Get, which shares the semantics) against the naive reference.
func FuzzGetManyAgainstRef(f *testing.F) {
	f.Add(`a=1 b=2 c=3`, "a", "b", "missing")
	f.Add(`dup="" dup=second`, "dup", "", "x")
	f.Add(`a= b="" a=1 b=`, "a", "b", "a")
	f.Add(`msg="k=v inside" k=real`, "k", "msg", "v")
	f.Add(`flag other=1`, "flag", "other", "")
	f.Add(`path=C:\Users\bob msg="a\tb"`, "path", "msg", "x")
	f.Add(``, "a", "b", "c")
	f.Fuzz(func(t *testing.T, data, k1, k2, k3 string) {
		keys := []string{k1, k2, k3}
		// GetMany fills duplicate query keys with successive occurrences — a
		// degenerate case the reference doesn't model; skip it.
		if k1 == k2 || k1 == k3 || k2 == k3 {
			return
		}
		// Neither side reports syntax errors any more: the reference consumes
		// the same valid prefix Iterate delivers before giving up, and early
		// stopping cannot change a first-non-empty-wins result. So the two must
		// agree exactly, malformed input included — a stronger property than
		// the error-case carve-outs this fuzzer used to need.
		got := GetMany([]byte(data), keys, nil)
		want, wantQuoted := getManyRef([]byte(data), keys)
		for j := range keys {
			if (got[j] == nil) != (want[j] == nil) {
				t.Fatalf("key %q nil mismatch: got %v want %v for %q",
					keys[j], got[j], want[j], data)
			}
			if string(got[j]) != string(want[j]) {
				t.Fatalf("key %q = %q, want %q for %q", keys[j], got[j], want[j], data)
			}
		}
		// Get and GetQuoted must agree with GetMany's per-key result, including
		// on whether the key was found at all and how it was written.
		for j := range keys {
			gv, ok := Get([]byte(data), keys[j])
			if ok != (want[j] != nil) {
				t.Fatalf("Get(%q) ok = %v, want %v for %q", keys[j], ok, want[j] != nil, data)
			}
			if string(gv) != string(want[j]) || (gv == nil) != (want[j] == nil) {
				t.Fatalf("Get(%q) = %q disagrees with ref %q for %q",
					keys[j], gv, want[j], data)
			}
			qv, quoted, qok := GetQuoted([]byte(data), keys[j])
			if qok != ok || string(qv) != string(gv) || (qv == nil) != (gv == nil) {
				t.Fatalf("GetQuoted(%q) = %q/%v disagrees with Get %q/%v for %q",
					keys[j], qv, qok, gv, ok, data)
			}
			if ok && quoted != wantQuoted[j] {
				t.Fatalf("GetQuoted(%q) quoted = %v, want %v for %q",
					keys[j], quoted, wantQuoted[j], data)
			}
		}

		// AppendValue must agree with Get on presence, and on contents — decoding
		// escapes ONLY for a value that was quoted. Unescaping an unquoted value
		// would eat backslashes the emitter meant literally, so the reference
		// branches on the same bit rather than decoding unconditionally.
		for j := range keys {
			av, ok := AppendValue(nil, []byte(data), keys[j])
			if ok != (want[j] != nil) {
				t.Fatalf("AppendValue(%q) ok = %v, want %v for %q", keys[j], ok, want[j] != nil, data)
			}
			if ok {
				decoded := want[j]
				if wantQuoted[j] {
					decoded = AppendUnescape(nil, want[j])
				}
				if string(av) != string(decoded) {
					t.Fatalf("AppendValue(%q) = %q, want %q (quoted=%v) for %q",
						keys[j], av, decoded, wantQuoted[j], data)
				}
			}
		}
	})
}
