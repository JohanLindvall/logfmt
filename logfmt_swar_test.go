package logfmt

import (
	"bytes"
	"encoding/binary"
	"math/bits"
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

		vStart, vEnd := i, i

		if i >= n {
			fn(buf[kStart:kEnd], buf[vStart:vEnd])
			return nil
		}

		// `key=` before whitespace is an empty value (see Iterate).
		if isSpace(buf[i]) {
			if !fn(buf[kStart:kEnd], buf[vStart:vEnd]) {
				return nil
			}
			continue
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

// isSpace is shared by Iterate and by iterateRef, the differential fuzzer's
// reference implementation — so a bug in it cancels out there and goes unseen.
// Pin it against the definition instead.
func Test_Unit_IsSpace(t *testing.T) {
	for b := 0; b < 256; b++ {
		want := b == ' ' || b == '\t' || b == '\n' || b == '\v' || b == '\f' || b == '\r'
		if got := isSpace(byte(b)); got != want {
			t.Errorf("isSpace(%#02x) = %v, want %v", b, got, want)
		}
	}
}

// firstStop reports which byte of a SWAR mask is flagged first, or 8 if none
// is. Only the first match is contractual: a borrow can set spurious high bits
// in bytes ABOVE a true match, which is precisely why the masks may only ever
// be OR-ed together and never subtracted from one another.
func firstStop(m uint64) int {
	if m == 0 {
		return 8
	}
	return bits.TrailingZeros64(m) >> 3
}

func Test_Unit_SWARMasks(t *testing.T) {
	// 'a' (0x61) satisfies neither predicate, so it is a safe filler.
	for pos := 0; pos < 8; pos++ {
		for v := 0; v < 256; v++ {
			buf := []byte("aaaaaaaa")
			buf[pos] = byte(v)
			w := binary.LittleEndian.Uint64(buf)

			want := 8
			if v <= 0x20 || v == '=' {
				want = pos
			}
			if got := firstStop(hasKeyStop(w)); got != want {
				t.Fatalf("hasKeyStop: byte %#02x at %d: stop %d, want %d", v, pos, got, want)
			}

			want = 8
			if v <= 0x20 {
				want = pos
			}
			if got := firstStop(hasCtrlOrSpace(w)); got != want {
				t.Fatalf("hasCtrlOrSpace: byte %#02x at %d: stop %d, want %d", v, pos, got, want)
			}
		}
	}

	// Two stops in one word: the lower position must win. This is the property
	// the OR-only rule exists to protect — subtracting one mask from another
	// would let a borrow from the low match manufacture a false stop above it.
	for lo := 0; lo < 8; lo++ {
		for hi := lo + 1; hi < 8; hi++ {
			for _, pair := range [][2]byte{
				{'=', ' '}, {' ', '='}, {'\n', '='}, {0x00, '='}, {0x01, ' '}, {'\t', '\r'},
			} {
				buf := []byte("aaaaaaaa")
				buf[lo], buf[hi] = pair[0], pair[1]
				w := binary.LittleEndian.Uint64(buf)
				if got := firstStop(hasKeyStop(w)); got != lo {
					t.Fatalf("hasKeyStop: %#v at %d,%d: stop %d, want %d", pair, lo, hi, got, lo)
				}
				if pair[0] <= 0x20 {
					if got := firstStop(hasCtrlOrSpace(w)); got != lo {
						t.Fatalf("hasCtrlOrSpace: %#v at %d,%d: stop %d, want %d", pair, lo, hi, got, lo)
					}
				}
			}
		}
	}
}
