package logfmt

import (
	"testing"
)

// getManyRef is an obviously-correct reference for GetMany's semantics: for
// each key, the first non-empty occurrence wins; failing that, the first
// (empty) occurrence; failing that, nil. It collects every pair first, with no
// early-stop or slot state machine.
func getManyRef(data []byte, keys []string) [][]byte {
	type pair struct{ k, v []byte }
	var pairs []pair
	// The error is deliberately ignored: Iterate delivers every pair preceding
	// a fault, and that valid prefix is exactly what the lookups see too.
	_ = Iterate(data, func(k, v []byte) bool {
		pairs = append(pairs, pair{k, v})
		return true
	})
	out := make([][]byte, len(keys))
	for j, key := range keys {
		for _, p := range pairs {
			if string(p.k) != key {
				continue
			}
			if len(p.v) > 0 {
				out[j] = p.v
				break
			}
			if out[j] == nil {
				out[j] = p.v // provisional empty; keep looking
			}
		}
	}
	return out
}

// FuzzGetManyAgainstRef checks GetMany's early-stop/provisional-empty state
// machine (and Get, which shares the semantics) against the naive reference.
func FuzzGetManyAgainstRef(f *testing.F) {
	f.Add(`a=1 b=2 c=3`, "a", "b", "missing")
	f.Add(`dup="" dup=second`, "dup", "", "x")
	f.Add(`a= b="" a=1 b=`, "a", "b", "a")
	f.Add(`msg="k=v inside" k=real`, "k", "msg", "v")
	f.Add(`flag other=1`, "flag", "other", "")
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
		want := getManyRef([]byte(data), keys)
		for j := range keys {
			if (got[j] == nil) != (want[j] == nil) {
				t.Fatalf("key %q nil mismatch: got %v want %v for %q",
					keys[j], got[j], want[j], data)
			}
			if string(got[j]) != string(want[j]) {
				t.Fatalf("key %q = %q, want %q for %q", keys[j], got[j], want[j], data)
			}
		}
		// Get must agree with GetMany's per-key result, including on whether the
		// key was found at all.
		for j := range keys {
			gv, ok := Get([]byte(data), keys[j])
			if ok != (want[j] != nil) {
				t.Fatalf("Get(%q) ok = %v, want %v for %q", keys[j], ok, want[j] != nil, data)
			}
			if string(gv) != string(want[j]) || (gv == nil) != (want[j] == nil) {
				t.Fatalf("Get(%q) = %q disagrees with ref %q for %q",
					keys[j], gv, want[j], data)
			}
		}

		// AppendValue must agree with Get on presence, and on contents once the
		// escapes are decoded.
		for j := range keys {
			av, ok := AppendValue(nil, []byte(data), keys[j])
			if ok != (want[j] != nil) {
				t.Fatalf("AppendValue(%q) ok = %v, want %v for %q", keys[j], ok, want[j] != nil, data)
			}
			if ok {
				if decoded := AppendUnescape(nil, want[j]); string(av) != string(decoded) {
					t.Fatalf("AppendValue(%q) = %q, want %q for %q", keys[j], av, decoded, data)
				}
			}
		}
	})
}
