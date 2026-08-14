package logfmt

import (
	"bytes"
	"encoding/binary"
	"math/bits"
	"slices"
	"strconv"
	"testing"
)

// iterateRef is a straightforward byte-by-byte reference implementation used
// to validate the SWAR-accelerated iterate. It must stay behaviourally
// identical to the scalar version iterate was derived from, including the
// quoted flag.
func iterateRef(buf []byte, fn func(key, val []byte, quoted bool) bool) error {
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
				fn(buf[kStart:n], trueSlice, false)
			}
			return nil
		}

		kEnd := i

		if buf[i] != '=' {
			if !fn(buf[kStart:i], trueSlice, false) {
				return nil
			}
			continue
		}
		i++

		vStart, vEnd := i, i
		quoted := false

		if i >= n {
			fn(buf[kStart:kEnd], buf[vStart:vEnd], false)
			return nil
		}

		// `key=` before whitespace is an empty value (see iterate).
		if isSpace(buf[i]) {
			if !fn(buf[kStart:kEnd], buf[vStart:vEnd], false) {
				return nil
			}
			continue
		}

		if buf[i] == '"' {
			quoted = true
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

		if !fn(buf[kStart:kEnd], buf[vStart:vEnd], quoted) {
			return nil
		}
	}

	return nil
}

// collectPairs records four facts per pair, not two. The key and value are the
// obvious ones; the other two exist because comparing only string(k)/string(v)
// makes the fuzzer blind to properties this package exports:
//
//   - IsBareKey(v) pins the trueSlice sentinel's IDENTITY. Without it, replacing
//     either trueSlice call site with a fresh []byte("true") is invisible —
//     byte-identical, but it flips IsBareKey, the only way a caller can tell
//     `debug` from `debug=true`.
//   - quoted pins the flag AppendValue relies on to decide whether unescaping a
//     value is correct at all.
func collectPairs(it func([]byte, func(k, v []byte, quoted bool) bool) error, buf []byte) ([]string, error) {
	var out []string
	err := it(buf, func(k, v []byte, quoted bool) bool {
		out = append(out, string(k), string(v),
			strconv.FormatBool(IsBareKey(v)), strconv.FormatBool(quoted))
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
		// A control byte inside an unquoted VALUE, with >= 8 bytes left at the
		// scan position, so the value SWAR loop (not the scalar tail, and not
		// the key scan like the seed above) locates it and takes its
		// finish-scalar break. Fuzzing finds this shape quickly, but that
		// corpus is machine-local; CI executes only these seeds.
		"k=aaaa\x06bbbbbbbbbbbb next=ok",
		"ƒƒƒƒƒƒƒƒ=ƒ aaaaaaaaaaaaaaaaaa=bbbbbbbbbbbbbbbbbbbb",
		// A TRAILING bare key, which leaves iterate through a different
		// trueSlice call site than a mid-line one ("the i >= n exit"), and is
		// the commonest shape for a bare flag.
		"level=info debug",
		// Backslashes in an UNQUOTED value: literal bytes, never escapes. The
		// quoted flag collectPairs records is what keeps this distinct from the
		// quoted seeds above.
		`path=C:\Users\bob re=\d+\s`,
		// Escape-DENSE quoted values, which are what switches a value over to
		// the SWAR escape scan. No seed above reaches it: each carries at most
		// one escaped quote, near the end of a short value, so the scan's word
		// loop, its skip-two step and its own unterminated exit were all
		// seed-unreachable — and CI runs seeds, not a corpus. Embedded JSON is
		// the shape that hits this hardest in the wild.
		`msg="a\"bbbbbbbbbbbbbbbbbbbb\"ccccccccccccccccc\"d" next=ok`,
		`json="{\"user\":\"bob\",\"id\":42,\"ok\":true}" level=info`,
		// Gaps of seven bytes, so an escape pair straddles two SWAR loads: the
		// backslash is the last byte of one word and the quote it protects is
		// the first byte of the next, where a scan that forgot the pair was
		// split would read that quote as the closing one.
		`k="aaaaaaa\"aaaaaaa\"aaaaaaa\"aaaaaaa\"" trailing=1`,
		// Gaps of six, for the same pair landing wholly inside one word.
		`k="aaaaaa\"aaaaaa\"aaaaaa\"aaaaaa\"" trailing=1`,
		// An escaped backslash reached from inside the scan: skipping two lands
		// on the following quote, which must then read as the CLOSING quote.
		`a="x\"y\\" b=2`,
		// The two mode TRANSITIONS, which nothing else here reaches. A value
		// that starts dense and then opens up hands the scan back to
		// bytes.IndexByte mid-value; the second seed does that with the escape
		// running off the end of the input, which is the shape that panicked
		// (the hand-back left i at n+1 and the outer data[i:] is a slice, not a
		// lookup — it does not fail to match, it crashes).
		`a="\"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\"yy" b=2`,
		`a="\"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\`,
		// Unterminated after the switch — the escape scan has its own exit for
		// that, distinct from the first probe's.
		`a="x\"yyyyyyyyyyyyyyyyyyyyyyyy`,
		// A trailing lone backslash in escape mode, where skipping two steps
		// clean over the end of the input rather than landing on it.
		`a="x\"y\`,
		// Malformed shapes. Until these were added no seed reached an error path
		// at all, so on CI — which runs seeds, not a corpus — the entire error
		// space of iterate went unexercised. The pair comparison below runs on
		// these too, so the "valid prefix before the error" promise is checked.
		`a=1 b="unterminated`,
		`a="x"y`,
		`a="x\"y`,
		`a="" b="x"z c=3`,
		`a=1 b="\\" c="x`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, buf []byte) {
		gotV, gotErr := collectPairs(iterate, buf)
		wantV, wantErr := collectPairs(iterateRef, buf)
		if (gotErr == nil) != (wantErr == nil) {
			t.Fatalf("err mismatch: got %v want %v for %q", gotErr, wantErr, buf)
		}
		// Compared unconditionally, including on malformed input. Both sides
		// promise to deliver every pair preceding the fault, so both must
		// deliver the SAME ones; skipping the comparison whenever an error came
		// back discarded that check on ~12% of short inputs — exactly the region
		// where the quoted-value state machine is trickiest.
		if !slicesEqual(gotV, wantV) {
			t.Fatalf("value mismatch for %q (err %v/%v):\n got  %q\n want %q",
				buf, gotErr, wantErr, gotV, wantV)
		}
	})
}

func slicesEqual(a, b []string) bool {
	return slices.Equal(a, b)
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

			want = 8
			if v == '"' || v == '\\' {
				want = pos
			}
			if got := firstStop(hasQuoteOrEsc(w)); got != want {
				t.Fatalf("hasQuoteOrEsc: byte %#02x at %d: stop %d, want %d", v, pos, got, want)
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

			// Same property for the escape scan, whose two terms are exact
			// equalities rather than ranges: a '\\' immediately before a '"'
			// is the shape it walks through constantly, and reading anything
			// but the lower stop would end the value at the wrong byte.
			for _, pair := range [][2]byte{
				{'"', '\\'}, {'\\', '"'}, {'"', '"'}, {'\\', '\\'},
			} {
				buf := []byte("aaaaaaaa")
				buf[lo], buf[hi] = pair[0], pair[1]
				w := binary.LittleEndian.Uint64(buf)
				if got := firstStop(hasQuoteOrEsc(w)); got != lo {
					t.Fatalf("hasQuoteOrEsc: %#v at %d,%d: stop %d, want %d", pair, lo, hi, got, lo)
				}
			}
		}
	}
}
