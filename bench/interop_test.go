package bench

import (
	"bytes"
	"testing"

	mine "github.com/JohanLindvall/logfmt"
	golog "github.com/go-logfmt/logfmt"
)

// TestUnescapeInterop verifies that values encoded by go-logfmt — which writes
// control characters as JSON-style \u00XX escapes — decode back to the original
// bytes with this package's Unescape, exactly as go-logfmt's own decoder does.
func TestUnescapeInterop(t *testing.T) {
	for _, val := range []string{
		"a\x07b",          // control char -> 
		"tab\tnl\n",       // short escapes
		`quote"back\`,     // \" and \\
		"café \U0001D11E", // multibyte UTF-8 passes through unescaped
	} {
		var out bytes.Buffer
		e := golog.NewEncoder(&out)
		if err := e.EncodeKeyval("k", val); err != nil {
			t.Fatalf("encode %q: %v", val, err)
		}
		line := out.Bytes()

		raw, ok := mine.Get(line, "k")
		if !ok {
			t.Fatalf("Get on %q: key not found", line)
		}
		if got := mine.AppendUnescape(nil, raw); string(got) != val {
			t.Errorf("round-trip %q via %q = %q", val, line, got)
		}
	}
}

// TestAllRangeOverFunc is the consumer-side proof of what All promises: this
// module declares go 1.23, so `for k, v := range logfmt.All(line)` must
// compile and iterate here, even though the library itself still builds at
// go 1.21 and cannot use the range form in its own tests.
func TestAllRangeOverFunc(t *testing.T) {
	var got []string
	for k, v := range mine.All([]byte(`a=1 b="two" c`)) {
		got = append(got, string(k)+"="+string(v))
	}
	want := []string{"a=1", "b=two", "c=true"}
	if len(got) != len(want) {
		t.Fatalf("All = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("All = %q, want %q", got, want)
		}
	}

	// break must stop the iteration.
	var n int
	for range mine.All([]byte(`a=1 b=2 c=3`)) {
		n++
		break
	}
	if n != 1 {
		t.Errorf("break yielded %d pairs, want 1", n)
	}
}
