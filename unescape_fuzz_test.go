package logfmt

import (
	"bytes"
	"strconv"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

// unescapeRef is a byte-at-a-time reference for AppendUnescape. It exists
// because the other differential fuzzer (FuzzGetManyAgainstRef) uses
// AppendUnescape as its own oracle for AppendValue, so a bug in the decoder
// cancels out there. This one spells the rules out independently: \n \r \t,
// \uXXXX with surrogate pairs (a lone half is U+FFFD, a malformed payload is
// kept verbatim), any other escaped byte is itself, and a trailing lone
// backslash is kept.
func unescapeRef(raw []byte) []byte {
	out := []byte{}
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c != '\\' {
			out = append(out, c)
			continue
		}
		if i+1 >= len(raw) {
			out = append(out, '\\')
			break
		}
		i++
		switch raw[i] {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'u':
			r1, ok := hex4Ref(raw[i+1:])
			if !ok {
				out = append(out, '\\', 'u')
				continue
			}
			i += 4
			if !utf16.IsSurrogate(r1) {
				out = utf8.AppendRune(out, r1)
				continue
			}
			r := utf8.RuneError
			if i+2 < len(raw) && raw[i+1] == '\\' && raw[i+2] == 'u' {
				if r2, ok := hex4Ref(raw[i+3:]); ok {
					if p := utf16.DecodeRune(r1, r2); p != utf8.RuneError {
						r = p
						i += 6
					}
				}
			}
			out = utf8.AppendRune(out, r)
		default:
			out = append(out, raw[i])
		}
	}
	return out
}

func hex4Ref(b []byte) (rune, bool) {
	if len(b) < 4 {
		return 0, false
	}
	for _, c := range b[:4] {
		isHex := c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
		if !isHex {
			return 0, false
		}
	}
	v, err := strconv.ParseUint(string(b[:4]), 16, 32)
	if err != nil {
		return 0, false
	}
	return rune(v), true
}

// FuzzAppendUnescapeAgainstRef checks AppendUnescape against unescapeRef three
// ways: appending to nil, appending behind an existing prefix (the result must
// keep the prefix intact and never alias raw), and decoding IN PLACE over the
// raw bytes (dst = raw[:0]), which is legal because decoding never lengthens
// the input — and which the escape-dense SWAR read-ahead must not disturb.
// It also pins NeedsUnescape's contract: false means the bytes decode to
// themselves.
func FuzzAppendUnescapeAgainstRef(f *testing.F) {
	for _, s := range []string{
		``, `plain`, `a\nb\tc\rd`, `q\"q\\`, `trailing\`, `éA`,
		`😀 pair`, `\ud83d lone`, `\ude00 low`, `\uZZZZ bad`, `\u12`,
		`{\"user\":\"bob\",\"action\":\"login\",\"path\":\"/api/v1/users\"}`,
		`\\\\\\\\\\\\\\\\\"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\"`,
		`\ud83d\u`, `\ud83d\uZZZZ`, `\ud83d\ud83d`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		want := unescapeRef(raw)
		if got := AppendUnescape(nil, raw); !bytes.Equal(got, want) {
			t.Fatalf("AppendUnescape(nil, %q) = %q, want %q", raw, got, want)
		}
		if NeedsUnescape(raw) == false && !bytes.Equal(want, raw) {
			t.Fatalf("NeedsUnescape(%q) = false but it decodes to %q", raw, want)
		}
		prefix := []byte("prefix:")
		if got := AppendUnescape(prefix, raw); !bytes.Equal(got, append([]byte("prefix:"), want...)) {
			t.Fatalf("AppendUnescape(prefix, %q) = %q, want prefix:%q", raw, got, want)
		}
		cp := append([]byte(nil), raw...)
		if got := AppendUnescape(cp[:0], cp); !bytes.Equal(got, want) {
			t.Fatalf("in-place AppendUnescape(%q) = %q, want %q", raw, got, want)
		}
	})
}
