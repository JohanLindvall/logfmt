package logfmt

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func Test_Unit_LogFmt_Values(t *testing.T) {
	for i, tt := range []struct {
		line     string
		expected []string
	}{
		{
			`foo`,
			[]string{"foo", "true"},
		},
		{
			`foo bar`,
			[]string{"foo", "true", "bar", "true"},
		},
		{
			`foo=`,
			[]string{"foo", ""},
		},
		{
			`foo=   bar   `,
			[]string{"foo", "bar"},
		},
		{
			`level=info msg="user login" user=john id=42 success=true `,
			[]string{"level", "info", "msg", "user login", "user", "john", "id", "42", "success", "true"},
		},
		{
			`level=info msg="hello\\nworld" user=john`,
			[]string{"level", "info", "msg", "hello\\\\nworld", "user", "john"},
		},
		{
			`a="escaped\"quote\nnewline" b=plain`,
			[]string{"a", "escaped\\\"quote\\nnewline", "b", "plain"},
		},
		{
			"a=1 b=\"bar\" ƒ=2h3s r=\"esc\\tmore stuff\" d x=sf   ",
			[]string{"a", "1", "b", "bar", "ƒ", "2h3s", "r", "esc\\tmore stuff", "d", "true", "x", "sf"},
		}} {
		t.Run(fmt.Sprintf("test-%d-%s", i, tt.line), func(t *testing.T) {
			var result []string
			err := Iterate([]byte(tt.line), func(k, v []byte) bool {
				result = append(result, string(k), string(v))
				return true
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func Test_Unit_LogFmt_Values_Invalid(t *testing.T) {
	for i, tt := range []string{
		`foo="bar"xx`,
	} {
		t.Run(fmt.Sprintf("test-%d-%s", i, tt), func(t *testing.T) {
			err := Iterate([]byte(tt), func(k, v []byte) bool {
				return true
			})
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func Test_Unit_NeedsUnescape(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want bool
	}{
		{"", false},
		{"plain", false},
		{"with space", false}, // space alone needs no decoding
		{`with"quote`, false}, // a bare quote needs no decoding
		{`esc\tval`, true},    // backslash escape
		{`trailing\`, true},   // lone trailing backslash
		{`a\\b`, true},        // escaped backslash
	} {
		if got := NeedsUnescape([]byte(tt.in)); got != tt.want {
			t.Errorf("NeedsUnescape(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func Test_Unit_Unescape(t *testing.T) {
	// No escapes + empty dst: returned unchanged and aliasing raw (zero-copy).
	raw := []byte("plain value")
	got := Unescape(nil, raw)
	if string(got) != "plain value" {
		t.Errorf("no-escape: got %q", got)
	}
	if len(got) == 0 || &got[0] != &raw[0] {
		t.Error("no-escape with empty dst must alias raw (zero-copy)")
	}

	for _, tt := range []struct{ in, want string }{
		{`a\nb`, "a\nb"},
		{`a\tb`, "a\tb"},
		{`a\rb`, "a\rb"},
		{`a\"b`, `a"b`},            // \" -> "
		{`a\\b`, `a\b`},            // \\ -> \
		{`a\xb`, "axb"},            // unknown escape -> the literal byte
		{`trailing\`, `trailing\`}, // lone trailing backslash kept
	} {
		if got := Unescape(nil, []byte(tt.in)); string(got) != tt.want {
			t.Errorf("Unescape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	// Non-empty dst: append semantics preserved (no short-circuit).
	if got := Unescape([]byte("prefix:"), []byte("plain")); string(got) != "prefix:plain" {
		t.Errorf("append to non-empty dst = %q, want prefix:plain", got)
	}
}

func Test_Unit_Get(t *testing.T) {
	line := []byte(`level=info msg="user login" id=42 r="esc\tval"`)

	for _, tt := range []struct {
		key  string
		want string
	}{
		{"level", "info"},
		{"msg", "user login"}, // quoted: surrounding quotes stripped
		{"id", "42"},
		{"r", `esc\tval`}, // raw: escape left intact
	} {
		v, err := Get(line, tt.key)
		if err != nil {
			t.Errorf("Get(%q) error: %v", tt.key, err)
			continue
		}
		if string(v) != tt.want {
			t.Errorf("Get(%q) = %q, want %q", tt.key, v, tt.want)
		}
	}

	if _, err := Get(line, "missing"); !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("Get(missing) error = %v, want ErrKeyNotFound", err)
	}

	if _, err := Get([]byte(`a="unterminated`), "a"); err == nil {
		t.Error("Get on malformed input: expected error, got nil")
	}
}

func Test_Unit_GetMany(t *testing.T) {
	// empty="" yields a present but empty value, distinct from a missing key.
	// "dup" appears first empty then with a real value, so the non-empty value
	// must override the provisional empty one. r holds an escape sequence that
	// must be returned raw (not decoded).
	line := []byte(`level=info msg="user login" id=42 r="a\tb" empty="" dup="" dup=second`)
	keys := []string{"id", "level", "missing", "msg", "empty", "r", "dup"}

	got, err := GetMany(line, keys, nil)
	if err != nil {
		t.Fatalf("GetMany: %v", err)
	}
	if len(got) != len(keys) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(keys))
	}
	want := map[string]string{"id": "42", "level": "info", "msg": "user login", "empty": "", "r": `a\tb`, "dup": "second"}
	for i, k := range keys {
		if k == "missing" {
			if got[i] != nil {
				t.Errorf("got[%d] (%s) = %q, want nil (absent)", i, k, got[i])
			}
			continue
		}
		// A present key (including the empty value) must be non-nil.
		if got[i] == nil {
			t.Errorf("got[%d] (%s) is nil, want present", i, k)
		}
		if string(got[i]) != want[k] {
			t.Errorf("got[%d] (%s) = %q, want %q", i, k, got[i], want[k])
		}
	}

	// Reuse the previous result's storage for a second line.
	got, err = GetMany([]byte(`level=warn id=7`), []string{"level", "id", "msg"}, got)
	if err != nil {
		t.Fatalf("GetMany reuse: %v", err)
	}
	if string(got[0]) != "warn" || string(got[1]) != "7" || got[2] != nil {
		t.Errorf("reuse got = [%q %q %v], want [warn 7 nil]", got[0], got[1], got[2])
	}

	// Empty key set.
	if res, err := GetMany(line, nil, nil); err != nil || len(res) != 0 {
		t.Errorf("GetMany(nil keys) = %v, %v; want empty, nil", res, err)
	}

	// Malformed input.
	if _, err := GetMany([]byte(`a="x`), []string{"a"}, nil); err == nil {
		t.Error("GetMany on malformed input: expected error, got nil")
	}
}

func Test_Unit_GetMany_Allocs(t *testing.T) {
	line := []byte(`ts=2025-01-01 level=info id=42 msg=hello`)
	keys := []string{"level", "id", "ts"}

	buf := make([][]byte, len(keys))
	buf, _ = GetMany(line, keys, buf)

	// Raw values alias data and buf is reused, so a warm call allocates nothing.
	allocs := testing.AllocsPerRun(100, func() {
		buf, _ = GetMany(line, keys, buf)
	})
	if allocs != 0 {
		t.Errorf("GetMany allocs/op = %v, want 0", allocs)
	}
}
