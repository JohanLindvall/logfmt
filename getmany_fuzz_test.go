package logfmt

import (
	"testing"
)

// getManyRef is an obviously-correct reference for GetMany's semantics: for
// each key, the first non-empty occurrence wins; failing that, the first
// (empty) occurrence; failing that, nil. It collects every pair first, with no
// early-stop or slot state machine.
func getManyRef(data []byte, keys []string) ([][]byte, error) {
	type pair struct{ k, v []byte }
	var pairs []pair
	err := Iterate(data, func(k, v []byte) bool {
		pairs = append(pairs, pair{k, v})
		return true
	})
	if err != nil {
		return nil, err
	}
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
	return out, nil
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
		got, gerr := GetMany([]byte(data), keys, nil)
		want, werr := getManyRef([]byte(data), keys)
		if gerr != nil {
			// GetMany parses a prefix of what the reference parses, so any
			// error it saw must also be seen by the full scan.
			if werr == nil {
				t.Fatalf("GetMany errored (%v) but reference did not for %q", gerr, data)
			}
			return
		}
		if werr != nil {
			// Legal only via early-stop: GetMany stops once every key has a
			// non-empty value, never reaching the malformed tail the full-scan
			// reference trips on. Verify that is indeed the state.
			for j := range keys {
				if len(got[j]) == 0 {
					t.Fatalf("GetMany returned nil error with unsettled key %q "+
						"while reference errored (%v) for %q", keys[j], werr, data)
				}
			}
			return
		}
		for j := range keys {
			if (got[j] == nil) != (want[j] == nil) {
				t.Fatalf("key %q nil mismatch: got %v want %v for %q",
					keys[j], got[j], want[j], data)
			}
			if string(got[j]) != string(want[j]) {
				t.Fatalf("key %q = %q, want %q for %q", keys[j], got[j], want[j], data)
			}
		}
		// Get must agree with GetMany's per-key result.
		for j := range keys {
			gv, err := Get([]byte(data), keys[j])
			switch {
			case err == ErrKeyNotFound:
				gv = nil
			case err != nil:
				t.Fatalf("Get(%q) unexpected error %v for %q", keys[j], err, data)
			}
			if string(gv) != string(want[j]) || (gv == nil) != (want[j] == nil) {
				t.Fatalf("Get(%q) = %q disagrees with ref %q for %q",
					keys[j], gv, want[j], data)
			}
		}
	})
}
