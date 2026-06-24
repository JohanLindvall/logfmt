package logfmt

import (
	"bytes"
	"testing"
)

// iterateRef is a straightforward byte-by-byte reference implementation used
// to validate the SWAR-accelerated Iterate. It must stay behaviourally
// identical to the scalar version Iterate was derived from.
func iterateRef(buf []byte, fn func(key, val []byte) bool) error {
	for i, n := 0, len(buf); i < n; {
		for i < n && isSpace(buf[i]) {
			i++
		}

		kStart := i
		for i < n && !isSpace(buf[i]) && buf[i] != '=' {
			i++
		}

		if i >= n {
			if kStart < n {
				fn(buf[kStart:n], trueSlice)
			}
			return nil
		}

		kEnd := i

		if buf[i] != '=' {
			if !fn(buf[kStart:i], trueSlice) {
				return nil
			}
			continue
		}
		i++

		for i < n && isSpace(buf[i]) {
			i++
		}

		vStart, vEnd := i, i

		if i >= n {
			fn(buf[kStart:kEnd], buf[vStart:vEnd])
			return nil
		}

		if buf[i] == '"' {
			i++
			vStart = i
			for {
				q := bytes.IndexByte(buf[i:], '"')
				if q == -1 {
					return ErrBadFormat
				}
				i += q
				bs := 0
				for j := i - 1; j >= vStart && buf[j] == '\\'; j-- {
					bs++
				}
				if bs%2 == 1 {
					i++
					continue
				}
				vEnd = i
				i++
				if i < n {
					if !isSpace(buf[i]) {
						return ErrBadFormat
					}
					i++
				}
				break
			}
		} else {
			vStart = i
			for i < n && !isSpace(buf[i]) {
				i++
			}
			vEnd = i
		}

		if !fn(buf[kStart:kEnd], buf[vStart:vEnd]) {
			return nil
		}
	}

	return nil
}

func collectPairs(it func([]byte, func(k, v []byte) bool) error, buf []byte) ([]string, error) {
	var out []string
	err := it(buf, func(k, v []byte) bool {
		out = append(out, string(k), string(v))
		return true
	})
	return out, err
}

func FuzzIterateAgainstRef(f *testing.F) {
	seeds := []string{
		"",
		"foo",
		"foo bar",
		"foo=",
		"foo=   bar   ",
		`level=info msg="user login" user=john id=42 success=true `,
		`level=info msg="hello\\nworld" user=john`,
		`a="escaped\"quote\nnewline" b=plain`,
		"a=1 b=\"bar\" ƒ=2h3s r=\"esc\\tmore stuff\" d x=sf   \n",
		string(sample2),
		"\x00\x01\x02\x08\x09\x0a\x0b\x0c\x0d\x0e=\x1f\x20\x7f\x80\xff",
		"longkeywithcontrol\x05inside=value verylongunquotedvalue\x06here next=ok",
		"ƒƒƒƒƒƒƒƒ=ƒ aaaaaaaaaaaaaaaaaa=bbbbbbbbbbbbbbbbbbbb",
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, buf []byte) {
		gotV, gotErr := collectPairs(Iterate, buf)
		wantV, wantErr := collectPairs(iterateRef, buf)
		if (gotErr == nil) != (wantErr == nil) {
			t.Fatalf("err mismatch: got %v want %v for %q", gotErr, wantErr, buf)
		}
		if gotErr == nil && !slicesEqual(gotV, wantV) {
			t.Fatalf("value mismatch for %q:\n got  %q\n want %q", buf, gotV, wantV)
		}
	})
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
