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

		raw, err := mine.Get(line, "k")
		if err != nil {
			t.Fatalf("Get on %q: %v", line, err)
		}
		if got := mine.Unescape(nil, raw); string(got) != val {
			t.Errorf("round-trip %q via %q = %q", val, line, got)
		}
	}
}
