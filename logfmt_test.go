package logfmt

import (
	"errors"
	"fmt"
	"slices"
	"strings"
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
			// `key=` is an empty value; the whitespace after '=' is NOT
			// skipped, so the following token is its own pair (or a bare key).
			`foo=   bar   `,
			[]string{"foo", "", "bar", "true"},
		},
		{
			`err= level=info msg=x`,
			[]string{"err", "", "level", "info", "msg", "x"},
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
		},
		{
			// A non-whitespace control byte is an ordinary value byte; this
			// one sits where the value SWAR scan finds it (>= 8 bytes left),
			// pinning the finish-scalar dispatch deterministically.
			"k=aaaa\x06bbbbbbbbbbbb next=ok",
			[]string{"k", "aaaa\x06bbbbbbbbbbbb", "next", "ok"},
		},
		{
			// '=' with no key before it yields a pair with an empty key, and
			// a '=' inside an unquoted value is a literal byte — both
			// documented divergences from go-logfmt, which rejects each.
			`=v a==b`,
			[]string{"", "v", "a", "=b"},
		},
		{
			// A '"' inside an unquoted value is a literal byte, not a syntax
			// error. Quoting is position-dependent: only a '"' immediately
			// after '=' opens a string.
			`a=x" b=c`,
			[]string{"a", `x"`, "b", "c"},
		},
		{
			// Keys are never unquoted, for the same reason: the leading '"' is
			// an ordinary key byte, so this is a bare key followed by a pair
			// whose key ends in a quote.
			`"a b"=c`,
			[]string{`"a`, "true", `b"`, "c"},
		},
		{
			// A '\' in an unquoted value is a literal byte, never an escape
			// introducer. go-logfmt's encoder emits Windows paths exactly this
			// way, because '\' is not one of the bytes that force quoting.
			`path=C:\Users\bob re=\d+\s`,
			[]string{"path", `C:\Users\bob`, "re", `\d+\s`},
		},
		{
			// Escape-dense: after the first escaped quote the parser switches
			// to a word-at-a-time scan that consumes each backslash with the
			// byte it escapes. JSON in a msg= field is the shape that matters;
			// the backslash runs check both parities before a quote, and the
			// long tail checks the hand-back to bytes.IndexByte.
			`msg="{\"user\":\"bob\",\"n\":1}" a="\\\" b=1" c="\\\\" ` +
				`d="x\"yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy\"z" e=2`,
			[]string{"msg", `{\"user\":\"bob\",\"n\":1}`, "a", `\\\" b=1`, "c", `\\\\`,
				"d", `x\"yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy\"z`, "e", "2"},
		},
		{
			// An escaping backslash as the last byte of an 8-byte word steps the
			// scan onto the following word's first byte, here a closing quote.
			`a="\"xxxxxxx\\" b=2`,
			[]string{"a", `\"xxxxxxx\\`, "b", "2"},
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
			if !slices.Equal(result, tt.expected) {
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

func Test_Unit_AppendUnescape(t *testing.T) {
	// AppendUnescape always appends: even with nothing to decode and an empty
	// dst, the result is a copy, never an alias of raw. (The old Unescape
	// returned raw itself here; callers who want that now guard with
	// NeedsUnescape, which is both explicit and faster.)
	raw := []byte("plain value")
	got := AppendUnescape(nil, raw)
	if string(got) != "plain value" {
		t.Errorf("no-escape: got %q", got)
	}
	if len(got) > 0 && &got[0] == &raw[0] {
		t.Error("AppendUnescape must copy, not alias raw")
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
		if got := AppendUnescape(nil, []byte(tt.in)); string(got) != tt.want {
			t.Errorf("Unescape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	// Non-empty dst: append semantics preserved (no short-circuit).
	if got := AppendUnescape([]byte("prefix:"), []byte("plain")); string(got) != "prefix:plain" {
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
		v, ok := Get(line, tt.key)
		if !ok {
			t.Errorf("Get(%q): not found", tt.key)
			continue
		}
		if string(v) != tt.want {
			t.Errorf("Get(%q) = %q, want %q", tt.key, v, tt.want)
		}
	}

	if v, ok := Get(line, "missing"); ok || v != nil {
		t.Errorf("Get(missing) = %q, %v; want nil, false", v, ok)
	}

	// A malformed record is not an error here: Get returns what the valid
	// prefix yielded. The key sits before the fault, so it is still found.
	if v, ok := Get([]byte(`a=1 b="unterminated`), "a"); !ok || string(v) != "1" {
		t.Errorf("Get(a) on malformed tail = %q, %v; want 1, true", v, ok)
	}
	// ...and a key that only appears past the fault is simply absent.
	if _, ok := Get([]byte(`a="unterminated b=2`), "b"); ok {
		t.Error("Get(b) past a fault: want not found")
	}
}

func Test_Unit_GetMany(t *testing.T) {
	// empty="" yields a present but empty value, distinct from a missing key.
	// "dup" appears first empty then with a real value, so the non-empty value
	// must override the provisional empty one. r holds an escape sequence that
	// must be returned raw (not decoded).
	line := []byte(`level=info msg="user login" id=42 r="a\tb" empty="" dup="" dup=second`)
	keys := []string{"id", "level", "missing", "msg", "empty", "r", "dup"}

	got := GetMany(line, keys, nil)
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
	got = GetMany([]byte(`level=warn id=7`), []string{"level", "id", "msg"}, got)
	if string(got[0]) != "warn" || string(got[1]) != "7" || got[2] != nil {
		t.Errorf("reuse got = [%q %q %v], want [warn 7 nil]", got[0], got[1], got[2])
	}

	// Empty key set.
	if res := GetMany(line, nil, nil); len(res) != 0 {
		t.Errorf("GetMany(nil keys) = %v; want empty", res)
	}

	// Malformed input yields the valid prefix rather than an error.
	if res := GetMany([]byte(`a=1 b="x`), []string{"a", "b"}, nil); string(res[0]) != "1" || res[1] != nil {
		t.Errorf("GetMany on malformed tail = [%q %v]; want [1 nil]", res[0], res[1])
	}
}

// Sinks keep the compiler from optimizing away the work being measured.
var (
	sinkBytes []byte
	sinkBool  bool
	sinkErr   error
	sinkMany  [][]byte
)

// Test_Unit_HotPath_Allocs pins the allocation-free contract across every entry
// point that documents one — not just the two that used to be covered. The line
// deliberately carries a quoted value, an escape and a bare key, because
// Iterate's quoted branch and the bare-key path are outside the reach of a
// plain unquoted sample.
//
// Append* are measured against a pre-sized dst[:0]: with a nil dst they must
// allocate the buffer itself, so asserting zero there would be asserting
// something the API never promised.
func Test_Unit_HotPath_Allocs(t *testing.T) {
	line := []byte(`level=info msg="user login" r="esc\tval" id=42 path=C:\tmp flag`)
	keys := []string{"level", "id", "missing"}
	many := make([][]byte, len(keys))
	dst := make([]byte, 0, 128)
	buf := []byte(`a=1 b=2` + "\n" + `c=3`)

	for _, tt := range []struct {
		name string
		fn   func()
	}{
		{"Iterate", func() {
			sinkErr = Iterate(line, func(k, v []byte) bool { sinkBytes = v; return true })
		}},
		{"Iterate/sample2", func() {
			sinkErr = Iterate(sample2, func(k, v []byte) bool { sinkBytes = v; return true })
		}},
		{"All", func() {
			All(line)(func(k, v []byte) bool { sinkBytes = v; return true })
		}},
		{"Validate", func() { sinkErr = Validate(line) }},
		{"Get", func() { sinkBytes, sinkBool = Get(line, "r") }},
		{"Get/absent", func() { sinkBytes, sinkBool = Get(line, "nope") }},
		{"GetQuoted", func() { sinkBytes, sinkBool, sinkBool = GetQuoted(line, "r") }},
		{"GetMany", func() { sinkMany = GetMany(line, keys, many) }},
		{"SplitRecord", func() { sinkBytes, sinkBytes = SplitRecord(buf) }},
		{"IsBareKey", func() { sinkBool = IsBareKey(line) }},
		{"NeedsUnescape", func() { sinkBool = NeedsUnescape(line) }},
		{"AppendValue/quoted", func() { sinkBytes, sinkBool = AppendValue(dst[:0], line, "r") }},
		{"AppendValue/unquoted", func() { sinkBytes, sinkBool = AppendValue(dst[:0], line, "path") }},
		{"AppendUnescape", func() { sinkBytes = AppendUnescape(dst[:0], []byte(`esc\tval`)) }},
	} {
		if n := testing.AllocsPerRun(100, tt.fn); n != 0 {
			t.Errorf("%s allocs/op = %v, want 0", tt.name, n)
		}
	}
}

// A malformed record costs exactly one allocation — the *SyntaxError — and only
// when the parse actually reaches the fault. That is the single carve-out in the
// package's allocation-free claim, so pin the shape of it rather than leaving
// the docs to assert it alone.
func Test_Unit_Malformed_Allocs(t *testing.T) {
	bad := []byte(`level=info b="unterminated`)

	if n := testing.AllocsPerRun(100, func() { sinkErr = Validate(bad) }); n != 1 {
		t.Errorf("Validate(malformed) allocs/op = %v, want 1 (the SyntaxError)", n)
	}
	// A lookup that settles before the fault never reaches it, so it stays free.
	if n := testing.AllocsPerRun(100, func() { sinkBytes, sinkBool = Get(bad, "level") }); n != 0 {
		t.Errorf("Get(malformed, key before fault) allocs/op = %v, want 0", n)
	}
	// One that has to walk past the fault pays for the error it then discards.
	if n := testing.AllocsPerRun(100, func() { sinkBytes, sinkBool = Get(bad, "nope") }); n != 1 {
		t.Errorf("Get(malformed, absent key) allocs/op = %v, want 1 (the discarded SyntaxError)", n)
	}
}

func Test_Unit_GetMany_Allocs(t *testing.T) {
	line := []byte(`ts=2025-01-01 level=info id=42 msg=hello`)
	keys := []string{"level", "id", "ts"}

	buf := make([][]byte, len(keys))
	buf = GetMany(line, keys, buf)

	// Raw values alias data and buf is reused, so a warm call allocates nothing.
	allocs := testing.AllocsPerRun(100, func() {
		buf = GetMany(line, keys, buf)
	})
	if allocs != 0 {
		t.Errorf("GetMany allocs/op = %v, want 0", allocs)
	}
}

// Values from the aliasing lookups are capped to their length, so a caller that
// appends to one gets a copy instead of scribbling over the rest of the input
// line. Iterate deliberately does NOT cap (it would cost ~4.5% on field-dense
// input), so this guarantee is specific to Get and GetMany; AppendValue cannot
// alias the input at all, since it always copies into the caller's buffer.
func Test_Unit_Lookups_CapValues(t *testing.T) {
	const orig = `a=hi b=there empty= q="x"`

	// v must alias line; appending to it must copy rather than touch line.
	check := func(t *testing.T, what string, v, line []byte) {
		t.Helper()
		if cap(v) != len(v) {
			t.Errorf("%s: cap = %d, want %d (len)", what, cap(v), len(v))
		}
		_ = append(v, "XXXX"...) //nolint:gocritic // appendAssign: the copy is the point
		if string(line) != orig {
			t.Errorf("%s: append overwrote the input: %q", what, line)
		}
	}

	line := []byte(orig)
	for _, key := range []string{"a", "b", "empty", "q"} {
		v, ok := Get(line, key)
		if !ok {
			t.Fatalf("Get(%q): not found", key)
		}
		check(t, "Get("+key+")", v, line)

		// The checks above are all satisfied by a heap COPY — cap == len holds
		// for one, and appending to one cannot touch line either. Zero-copy is
		// the actual contract, so assert the value really is a window onto line:
		// its bytes must live at the offset where the key's value sits.
		off := strings.Index(orig, key+"=") + len(key) + 1
		if orig[off] == '"' {
			off++ // quoted: the value starts past the opening quote
		}
		if len(v) > 0 && &v[0] != &line[off] {
			t.Errorf("Get(%q) does not alias line at offset %d — it copied", key, off)
		}

		// AppendValue writes into the caller's buffer, so its result must not
		// point into line at all. Compared against line's own byte rather than
		// against v, which would be vacuous if Get had started copying.
		av, ok := AppendValue(nil, line, key)
		if !ok {
			t.Fatalf("AppendValue(%q): not found", key)
		}
		if len(av) > 0 && &av[0] == &line[off] {
			t.Errorf("AppendValue(%q) aliases the input; it must copy", key)
		}
	}

	// Get must not allocate: it hands back a window, never a copy. This is the
	// other half of the zero-copy pin above, and the one that fails loudly if a
	// future edit reintroduces a defensive copy.
	if n := testing.AllocsPerRun(100, func() { Get(line, "q") }); n != 0 {
		t.Errorf("Get allocs/op = %v, want 0 (it must alias, not copy)", n)
	}

	keys := []string{"a", "b", "empty", "q", "missing"}
	vals := GetMany(line, keys, nil)
	for i, key := range keys {
		if key == "missing" {
			if vals[i] != nil {
				t.Errorf("GetMany(%q) = %q, want nil", key, vals[i])
			}
			continue
		}
		check(t, "GetMany("+key+")", vals[i], line)
	}

	// Capping must not turn a present-but-empty value into nil — that is how
	// GetMany distinguishes it from an absent key, and how Get reports "empty="
	// as found rather than missing.
	if v, ok := Get(line, "empty"); !ok || v == nil || len(v) != 0 {
		t.Errorf("Get(empty) = %v (nil? %v), %v; want present, non-nil, empty", v, v == nil, ok)
	}
	if v, ok := Get([]byte(`x=1 trailing=`), "trailing"); !ok || v == nil {
		t.Errorf("Get(trailing=) = %v (nil? %v), %v; want present, non-nil", v, v == nil, ok)
	}
	if vals[2] == nil {
		t.Error("GetMany(empty) is nil; a present-but-empty value must stay non-nil")
	}
}

func Test_Unit_Unescape_Unicode(t *testing.T) {
	bs := "\\" // single backslash
	for _, tt := range []struct{ in, want string }{
		{bs + "u0007ab", "\aab"},                    // control char, as go-logfmt encodes
		{bs + "u00e9", "é"},                         // lowercase hex
		{bs + "u00E9", "é"},                         // uppercase hex
		{bs + "ud834" + bs + "udd1e", "\U0001D11E"}, // surrogate pair
		{bs + "ud834", "�"},                         // lone high surrogate
		{bs + "ud834A", "�A"},                       // high surrogate, no pair
		{bs + "udd1e", "�"},                         // lone low surrogate
		{bs + "uZZZZ", bs + "uZZZZ"},                // malformed hex: verbatim
		{bs + "u00", bs + "u00"},                    // truncated: verbatim
		{"x" + bs + "u", "x" + bs + "u"},            // bare \u at end: verbatim
		{"a" + bs + "tb" + bs + "u0041", "a\tbA"},   // mixed with \t
	} {
		if got := AppendUnescape(nil, []byte(tt.in)); string(got) != tt.want {
			t.Errorf("Unescape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func Test_Unit_Get_Duplicates(t *testing.T) {
	line := []byte(`dup="" mid=x dup=second dup=third`)

	// First non-empty occurrence wins over an earlier empty one.
	if v, ok := Get(line, "dup"); !ok || string(v) != "second" {
		t.Errorf("Get(dup) = %q, %v; want second", v, ok)
	}
	// Only-empty occurrences are found, with an empty value.
	if v, ok := Get([]byte(`e= x=1`), "e"); !ok || v == nil || len(v) != 0 {
		t.Errorf("Get(bare e=) = %q, %v; want present empty", v, ok)
	}
	if v, ok := Get([]byte(`e= x=1`), "x"); !ok || string(v) != "1" {
		t.Errorf("Get(x after empty e=) = %q, %v; want 1 — an empty value must not swallow the next pair", v, ok)
	}
	if v, ok := Get([]byte(`e="" x=1`), "e"); !ok || v == nil || len(v) != 0 {
		t.Errorf("Get(e) = %q, %v; want present empty", v, ok)
	}
	if v, ok := Get([]byte(`x=1 e=`), "e"); !ok || v == nil || len(v) != 0 {
		t.Errorf("Get(trailing e=) = %q, %v; want present empty", v, ok)
	}
	// AppendValue agrees with Get.
	if v, ok := AppendValue(nil, line, "dup"); !ok || string(v) != "second" {
		t.Errorf("AppendValue(dup) = %q, %v; want second", v, ok)
	}
	// GetMany agrees too.
	if m := GetMany(line, []string{"dup"}, nil); string(m[0]) != "second" {
		t.Errorf("GetMany(dup) = %q; want second", m[0])
	}
}

func Test_Unit_AppendValue(t *testing.T) {
	line := []byte(`level=info msg="user login" r="esc\tval" empty=`)

	var buf []byte
	for _, tt := range []struct{ key, want string }{
		{"level", "info"},
		{"msg", "user login"},
		{"r", "esc\tval"}, // unescaped, unlike Get
		{"empty", ""},
	} {
		v, ok := AppendValue(buf[:0], line, tt.key)
		if !ok {
			t.Errorf("AppendValue(%q): not found", tt.key)
			continue
		}
		if string(v) != tt.want {
			t.Errorf("AppendValue(%q) = %q, want %q", tt.key, v, tt.want)
		}
	}

	// Absent key: dst comes back untouched, ok is false.
	dst := append([]byte(nil), "keep"...)
	v, ok := AppendValue(dst, line, "missing")
	if ok {
		t.Error("AppendValue(missing) ok = true, want false")
	}
	if string(v) != "keep" {
		t.Errorf("AppendValue(missing) = %q, want the dst it was given (%q)", v, "keep")
	}

	// Append semantics: the value extends dst rather than replacing it.
	if v, ok := AppendValue(dst, line, "level"); !ok || string(v) != "keepinfo" {
		t.Errorf("AppendValue appended = %q, %v; want keepinfo, true", v, ok)
	}

	// A malformed record is not an error; the reachable prefix still resolves.
	if v, ok := AppendValue(nil, []byte(`a=1 b="x`), "a"); !ok || string(v) != "1" {
		t.Errorf("AppendValue on malformed tail = %q, %v; want 1, true", v, ok)
	}
}

// Escapes are meaningful only inside quotes. Unescaping an unquoted value eats
// backslashes the emitter meant literally — go-logfmt writes path=C:\Users\bob
// unquoted, since '\' is not one of the bytes that force quoting — so a lookup
// that decodes blindly turns \U into U, \b into b and \n into a newline. That is
// silent corruption: every byte of the input is valid logfmt either way.
func Test_Unit_Unquoted_Backslashes_Are_Literal(t *testing.T) {
	line := []byte(`path=C:\Users\bob\new re=\d+\s msg="ok\tdone" q="a\"b" u=\u0041 empty= flag`)

	for _, tt := range []struct {
		key           string
		raw           string
		quoted        bool
		appended      string
		isBare        bool
		needsUnescape bool
	}{
		{"path", `C:\Users\bob\new`, false, `C:\Users\bob\new`, false, true},
		{"re", `\d+\s`, false, `\d+\s`, false, true},
		{"u", `\u0041`, false, `\u0041`, false, true},
		{"msg", `ok\tdone`, true, "ok\tdone", false, true},
		{"q", `a\"b`, true, `a"b`, false, true},
		{"empty", "", false, "", false, false},
		{"flag", "true", false, "true", true, false},
	} {
		t.Run(tt.key, func(t *testing.T) {
			raw, quoted, ok := GetQuoted(line, tt.key)
			if !ok {
				t.Fatalf("GetQuoted(%q): not found", tt.key)
			}
			if string(raw) != tt.raw {
				t.Errorf("raw = %q, want %q", raw, tt.raw)
			}
			if quoted != tt.quoted {
				t.Errorf("quoted = %v, want %v", quoted, tt.quoted)
			}
			if got := NeedsUnescape(raw); got != tt.needsUnescape {
				t.Errorf("NeedsUnescape = %v, want %v", got, tt.needsUnescape)
			}
			// Get must agree with GetQuoted on everything but the flag.
			if gv, gok := Get(line, tt.key); gok != ok || string(gv) != tt.raw {
				t.Errorf("Get = %q/%v, disagrees with GetQuoted %q/%v", gv, gok, raw, ok)
			}
			// The whole point: AppendValue decodes only what was quoted.
			av, ok := AppendValue(nil, line, tt.key)
			if !ok {
				t.Fatalf("AppendValue(%q): not found", tt.key)
			}
			if string(av) != tt.appended {
				t.Errorf("AppendValue = %q, want %q", av, tt.appended)
			}
			if got := IsBareKey(raw); got != tt.isBare {
				t.Errorf("IsBareKey(GetQuoted result) = %v, want %v", got, tt.isBare)
			}
		})
	}

	// An absent key reports quoted == false, like every other field of a
	// not-found result.
	if v, quoted, ok := GetQuoted(line, "missing"); ok || quoted || v != nil {
		t.Errorf("GetQuoted(missing) = %q, %v, %v; want nil, false, false", v, quoted, ok)
	}

	// Duplicate resolution is unchanged, and the flag follows the value that
	// actually won rather than the first occurrence.
	dup := []byte(`d= d=C:\x d="y\tz"`)
	if v, quoted, ok := GetQuoted(dup, "d"); !ok || string(v) != `C:\x` || quoted {
		t.Errorf("GetQuoted(dup) = %q, quoted=%v, ok=%v; want C:\\x, false, true", v, quoted, ok)
	}
	if v, ok := AppendValue(nil, dup, "d"); !ok || string(v) != `C:\x` {
		t.Errorf("AppendValue(dup) = %q; want C:\\x undecoded", v)
	}
}

// Test_Unit_Quoted_EscapeDense_Scan pins the two-scan split a quoted value with
// escapes is handled by: the bytes.IndexByte scan a value with no escaped quote
// never leaves, the inline SWAR walk an escape-dense one switches to, and — the
// part nothing else covers — every transition between them. The transitions are
// where the bugs live: the walk can decline on arrival (the first escape sits
// more than escGap bytes in), give up part way (escClean clean words go by), and
// hand back with the position already one byte PAST the end of the input.
func Test_Unit_Quoted_EscapeDense_Scan(t *testing.T) {
	// Longer than the walk's clean-run bailout, so a gap spanning it hands the
	// rest of the value back to the IndexByte scan; also longer than escGap, so
	// a value whose first escape sits behind it is declined on arrival.
	long := strings.Repeat("x", escClean*8+8)

	for _, tt := range []struct {
		name    string
		line    string
		raw     string // the value as Iterate reports it, escapes intact
		decoded string // the same value as AppendValue reports it
	}{{
		name:    "dense throughout",
		line:    `msg="{\"user\":\"bob\",\"id\":42}" next=ok`,
		raw:     `{\"user\":\"bob\",\"id\":42}`,
		decoded: `{"user":"bob","id":42}`,
	}, {
		name:    "escaped backslash reached from the walk",
		line:    `msg="x\"y\\" next=ok`,
		raw:     `x\"y\\`,
		decoded: `x"y\`,
	}, {
		// Escapes seven bytes apart, so a backslash and the quote it protects
		// straddle two SWAR word loads.
		name:    "escape pair straddling a word",
		line:    `msg="aaaaaaa\"aaaaaaa\"aaaaaaa\"" next=ok`,
		raw:     `aaaaaaa\"aaaaaaa\"aaaaaaa\"`,
		decoded: `aaaaaaa"aaaaaaa"aaaaaaa"`,
	}, {
		name:    "dense, then gaps open up",
		line:    `msg="\"` + long + `\"yy" next=ok`,
		raw:     `\"` + long + `\"yy`,
		decoded: `"` + long + `"yy`,
	}, {
		// The first escape sits beyond escGap, so the walk declines on arrival
		// and the whole value is settled by the IndexByte scan.
		name:    "first escape too far in: walk declined on arrival",
		line:    `msg="` + long + `\"\"\"" next=ok`,
		raw:     long + `\"\"\"`,
		decoded: long + `"""`,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			line := []byte(tt.line)
			if err := Validate(line); err != nil {
				t.Fatalf("Validate: unexpected error: %v", err)
			}
			got, quoted, ok := GetQuoted(line, "msg")
			if !ok || !quoted {
				t.Fatalf("GetQuoted: got ok=%v quoted=%v, want both true", ok, quoted)
			}
			if string(got) != tt.raw {
				t.Errorf("raw value:\n got  %q\n want %q", got, tt.raw)
			}
			if dec, _ := AppendValue(nil, line, "msg"); string(dec) != tt.decoded {
				t.Errorf("decoded value:\n got  %q\n want %q", dec, tt.decoded)
			}
			// The pair after it must still be found, which is the check that
			// the scan stopped on the right quote rather than one further on.
			if v, _ := Get(line, "next"); string(v) != "ok" {
				t.Errorf("next: got %q, want \"ok\"", v)
			}
		})
	}

	// Malformed shapes only reachable through the walk.
	for _, tt := range []struct{ name, line string }{{
		name: "unterminated after handing back",
		line: `msg="\"` + long + `\"yy`,
	}, {
		// The escape runs off the end while the walk is still running, and it
		// steps two bytes at a time, so it lands PAST the end rather than on
		// it. Handing that position back to the IndexByte scan used to panic;
		// testdata/fuzz/FuzzIterateAgainstRef/ee6d5b3abecfadf7 is the input
		// that found it.
		name: "trailing lone backslash mid-walk",
		line: `msg="\"` + strings.Repeat("0", escClean*8-8) + `\`,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate([]byte(tt.line))
			if !errors.Is(err, ErrBadFormat) {
				t.Fatalf("Validate: got %v, want ErrBadFormat", err)
			}
		})
	}
}

func Test_Unit_Validate_SyntaxError(t *testing.T) {
	if err := Validate([]byte(`a=1 b="ok" c=3`)); err != nil {
		t.Errorf("Validate(well-formed) = %v, want nil", err)
	}

	for _, tt := range []struct {
		line       string
		wantOffset int
		wantReason string
	}{
		// Offset points at the opening quote that is never closed.
		{`a=1 b="unterminated`, 6, "unterminated quoted value"},
		// Same, reached from the escape-dense scan: the escaping backslash is
		// the last byte of the input, so the scan steps one past the end and
		// has to be clamped before IndexByte is handed the tail.
		{`a="\"xxxxxxx\`, 2, "unterminated quoted value"},
		{`a="\"\`, 2, "unterminated quoted value"},
		// Offset points at the offending byte itself.
		{`a="x"y`, 5, "unexpected byte after closing quote"},
	} {
		err := Validate([]byte(tt.line))
		if err == nil {
			t.Fatalf("Validate(%q) = nil, want an error", tt.line)
		}
		if !errors.Is(err, ErrBadFormat) {
			t.Errorf("Validate(%q): errors.Is(err, ErrBadFormat) = false, want true", tt.line)
		}
		var se *SyntaxError
		if !errors.As(err, &se) {
			t.Fatalf("Validate(%q) = %T, want *SyntaxError", tt.line, err)
		}
		if se.Offset != tt.wantOffset {
			t.Errorf("Validate(%q) offset = %d, want %d", tt.line, se.Offset, tt.wantOffset)
		}
		if se.Reason != tt.wantReason {
			t.Errorf("Validate(%q) reason = %q, want %q", tt.line, se.Reason, tt.wantReason)
		}
		// The message carries the position, which is the point of the type.
		if !strings.Contains(err.Error(), fmt.Sprint(tt.wantOffset)) {
			t.Errorf("Validate(%q) message %q omits the offset", tt.line, err)
		}
	}
}

func Test_Unit_SplitRecord(t *testing.T) {
	for _, tt := range []struct {
		in         string
		wantRec    string
		wantRest   string
		wantNilest bool
	}{
		{"a=1\nb=2\n", "a=1", "b=2\n", false},
		{"a=1\r\nb=2", "a=1", "b=2", false}, // CRLF: the \r is trimmed
		{"only=1", "only=1", "", true},      // no newline: all record, nil rest
		{"\nb=2", "", "b=2", false},         // leading blank line
		{"", "", "", true},                  // empty input
	} {
		rec, rest := SplitRecord([]byte(tt.in))
		if string(rec) != tt.wantRec {
			t.Errorf("SplitRecord(%q) record = %q, want %q", tt.in, rec, tt.wantRec)
		}
		if string(rest) != tt.wantRest {
			t.Errorf("SplitRecord(%q) rest = %q, want %q", tt.in, rest, tt.wantRest)
		}
		if (rest == nil) != tt.wantNilest {
			t.Errorf("SplitRecord(%q) rest nil = %v, want %v", tt.in, rest == nil, tt.wantNilest)
		}
		// The record is capped, so appending to it cannot reach into the rest.
		if cap(rec) != len(rec) {
			t.Errorf("SplitRecord(%q) record cap = %d, want %d", tt.in, cap(rec), len(rec))
		}
	}

	// The documented loop terminates and yields each record in order.
	data := []byte("a=1\nb=2\r\nc=3")
	var got []string
	for len(data) > 0 {
		var rec []byte
		rec, data = SplitRecord(data)
		got = append(got, string(rec))
	}
	if want := []string{"a=1", "b=2", "c=3"}; !slices.Equal(got, want) {
		t.Errorf("split loop = %q, want %q", got, want)
	}

	// Splitting is what keeps a lookup from crossing a record boundary.
	multi := []byte("level=info\nlevel=error")
	if v, _ := Get(multi, "level"); string(v) != "info" {
		t.Errorf("Get on unsplit input = %q; want info (first line)", v)
	}
	rec, _ := SplitRecord(multi)
	if v, _ := Get(rec, "level"); string(v) != "info" {
		t.Errorf("Get on first record = %q, want info", v)
	}
}

// All is exercised here by direct invocation, which is precisely what a range
// loop desugars to; this module declares go 1.21, so it cannot use the range
// form itself. TestAllRangeOverFunc in the bench module (go 1.23) proves the
// consumer-facing promise that `for k, v := range logfmt.All(line)` compiles.
func Test_Unit_All(t *testing.T) {
	var got []string
	All([]byte(`a=1 b="two" c`))(func(k, v []byte) bool {
		got = append(got, string(k)+"="+string(v))
		return true
	})
	if want := []string{"a=1", "b=two", "c=true"}; !slices.Equal(got, want) {
		t.Errorf("All = %q, want %q", got, want)
	}

	// Returning false stops iteration — what `break` compiles to.
	var n int
	All([]byte(`a=1 b=2 c=3`))(func(k, v []byte) bool {
		n++
		return false
	})
	if n != 1 {
		t.Errorf("stopping after the first pair yielded %d, want 1", n)
	}

	// A malformed record yields the valid prefix and stops; the error is
	// Validate's job, since a range loop has nowhere to report one.
	var pairs int
	All([]byte(`a=1 b=2 c="unterminated`))(func(k, v []byte) bool {
		pairs++
		return true
	})
	if pairs != 2 {
		t.Errorf("All over malformed input yielded %d pairs, want 2 (the valid prefix)", pairs)
	}
}

func Test_Unit_IsBareKey(t *testing.T) {
	// A bare key and an explicit true are indistinguishable by content; only
	// IsBareKey tells them apart.
	var bare, explicit []byte
	_ = Iterate([]byte(`flag other=true`), func(k, v []byte) bool {
		switch string(k) {
		case "flag":
			bare = v
		case "other":
			explicit = v
		}
		return true
	})
	if string(bare) != "true" || string(explicit) != "true" {
		t.Fatalf("setup: bare = %q, explicit = %q; want both true", bare, explicit)
	}
	if !IsBareKey(bare) {
		t.Error("IsBareKey(bare key value) = false, want true")
	}
	if IsBareKey(explicit) {
		t.Error(`IsBareKey(value of other=true) = true, want false`)
	}
	// Anything not handed out by this package reports false.
	if IsBareKey([]byte("true")) {
		t.Error("IsBareKey on a caller's own []byte = true, want false")
	}
	if IsBareKey(nil) {
		t.Error("IsBareKey(nil) = true, want false")
	}
}

func Test_Unit_ParseTime_Allocs(t *testing.T) {
	// ParseTime takes []byte like the rest of the package and must not copy
	// them to reach time.Parse. Every case here has to stay at zero, INCLUDING
	// the ones past 32 bytes: a plain string([]byte) conversion is kept off the
	// heap only while it fits the compiler's stack buffer, so testing only
	// short timestamps hides the copy on exactly the two layouts that are long.
	//
	// Two shapes are deliberately absent because their cost is time.Parse's
	// own and predates the []byte signature — a zone abbreviation it cannot
	// resolve ("...+0200 CEST", 4 allocs, the fabricated Location) and a value
	// matching no layout ("...+07:00xxx", 5 allocs, the discarded *ParseError).
	// Both allocate identically when time.Parse is handed a plain string with
	// no conversion at all, so pinning them here would assert something this
	// package cannot deliver short of hand-rolling the layouts.
	for _, ts := range [][]byte{
		[]byte("1748239806.3691056"),                  // epoch, returns before the layout loop
		[]byte("2025-05-26T06:10:06.3691056Z"),        // 28 bytes, fits the stack buffer
		[]byte("2026-03-14 06:11:46.397 +0000 UTC"),   // 33 bytes, "-0700 MST" layout
		[]byte("2026-03-14T06:11:46.123456789+07:00"), // 35 bytes, RFC3339Nano
	} {
		if allocs := testing.AllocsPerRun(100, func() { ParseTime(ts) }); allocs != 0 {
			t.Errorf("ParseTime(%q) allocs = %v, want 0", ts, allocs)
		}
	}
}

// Test_Unit_ParseTime_NoAliasing pins the safety argument behind ParseTime's
// unsafe.String: nothing the caller can reach may alias the input bytes, so
// mutating (or reusing) the buffer afterwards must not disturb the result.
// The zone abbreviation is the interesting case — time.Parse fabricates a
// FixedZone named by a slice of the value when it cannot resolve one.
func Test_Unit_ParseTime_NoAliasing(t *testing.T) {
	for _, in := range []string{
		"2026-03-14 06:11:46.397 +0200 CEST", // unresolvable abbreviation
		"2026-03-14 06:11:46.397 +0000 UTC",
		"2026-03-14T06:11:46.123456789+07:00",
	} {
		buf := []byte(in)
		got, ok := ParseTime(buf)
		if !ok {
			t.Fatalf("ParseTime(%q) failed", in)
		}
		want := got.String()
		wantZone, wantOff := got.Zone()

		for i := range buf { // scribble over the bytes ParseTime just read
			buf[i] = 'x'
		}
		if got.String() != want {
			t.Errorf("ParseTime(%q): time changed after input was overwritten: %q -> %q", in, want, got.String())
		}
		if z, off := got.Zone(); z != wantZone || off != wantOff {
			t.Errorf("ParseTime(%q): zone changed after input was overwritten: %q/%d -> %q/%d", in, wantZone, wantOff, z, off)
		}
	}
}
