package logfmt

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

// unescapeRef is a byte-at-a-time reference for AppendUnescape: walk the input
// one byte at a time, and on a backslash consume the escape. No searching, no
// chunk copying, no modes — the thing AppendUnescape is a fast version of.
//
// This exists because the unescape half of the API had no oracle at all.
// FuzzGetManyAgainstRef checks AppendValue by calling AppendUnescape on the
// other side of the comparison, so a bug INSIDE AppendUnescape cancels out
// there and is invisible; and AppendUnescape is where a search bug does the
// most damage, since it silently returns plausible-looking wrong bytes rather
// than failing.
//
// One thing it deliberately shares: decodeUnicodeEscape, and therefore the
// surrogate-pair and malformed-\u rules. A bug in there cancels out on both
// sides exactly as isSpace does for iterateRef — which is why, as with isSpace,
// that piece has its own direct test (Test_Unit_Unescape_Unicode) rather than
// relying on this. What this fuzzer is an independent check of is everything
// around it: WHERE the escapes are found, and which bytes are copied between
// them.
func unescapeRef(raw []byte) []byte {
	out := []byte{}
	for i := 0; i < len(raw); {
		if raw[i] != '\\' {
			out = append(out, raw[i])
			i++
			continue
		}
		i++
		if i >= len(raw) {
			out = append(out, '\\') // trailing lone backslash, kept verbatim
			break
		}
		c := raw[i]
		i++
		switch c {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'u':
			if r, adv, ok := decodeUnicodeEscape(raw[i:]); ok {
				out = utf8.AppendRune(out, r)
				i += adv
			} else {
				out = append(out, '\\', 'u') // malformed, kept verbatim
			}
		default:
			out = append(out, c) // \" and \\ included: the byte itself
		}
	}
	return out
}

func FuzzAppendUnescapeAgainstRef(f *testing.F) {
	seeds := []string{
		"",
		"plain",
		`a\tb`,
		`a\nb\rc`,
		`\"quoted\"`,
		`back\\slash`,
		`C:\Users\bob`, // \U and \b are not escapes; both pass through as bytes
		`\u0041\u00e9`,
		`\ud83d\ude00`, // surrogate pair
		`\ud83d`,       // lone high surrogate -> U+FFFD
		`\uZZZZ`,       // malformed, kept verbatim
		`\u00`,         // truncated
		`trailing\`,
		`\`,
		`\\`,
		// Escape-DENSE input, the shape the parse side now has a second mode
		// for and the one where a search bug would hide behind plausible output.
		`{\"user\":\"bob\",\"id\":42,\"ok\":true}`,
		`\"\"\"\"\"\"\"\"\"\"\"\"`,
		// Long clean runs between escapes, so the search crosses whole words
		// and its tail handling matters.
		`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\tbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n`,
		"\x00\x01\x02\\\x7f\x80\xff",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		want := unescapeRef(raw)
		got := AppendUnescape(nil, raw)
		if !bytes.Equal(got, want) {
			t.Fatalf("AppendUnescape(%q):\n got  %q\n want %q", raw, got, want)
		}

		// Appending is part of the contract, and a search that miscounts an
		// offset can still land the right bytes in an empty dst while
		// corrupting a non-empty one.
		const prefix = "keep-me"
		if got := AppendUnescape([]byte(prefix), raw); !bytes.Equal(got, append([]byte(prefix), want...)) {
			t.Fatalf("AppendUnescape(%q, %q) did not append cleanly: %q", prefix, raw, got)
		}

		// NeedsUnescape is the guard callers are told to use to skip the
		// decode, so a false negative from it is a silently missed decode.
		if !NeedsUnescape(raw) && !bytes.Equal(want, raw) {
			t.Fatalf("NeedsUnescape(%q) is false but decoding changes it to %q", raw, want)
		}
	})
}
