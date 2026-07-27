package logfmt

import (
	"errors"
	"fmt"
	"reflect"
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

		// AppendValue writes into the caller's buffer, so its result must not
		// point into line at all.
		av, ok := AppendValue(nil, line, key)
		if !ok {
			t.Fatalf("AppendValue(%q): not found", key)
		}
		if len(av) > 0 && &av[0] == &v[0] {
			t.Errorf("AppendValue(%q) aliases the input; it must copy", key)
		}
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
	if want := []string{"a=1", "b=2", "c=3"}; !reflect.DeepEqual(got, want) {
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
	if want := []string{"a=1", "b=two", "c=true"}; !reflect.DeepEqual(got, want) {
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
