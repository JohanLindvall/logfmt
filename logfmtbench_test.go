package logfmt

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

// sample2 is a real-world ~1.4 KB line with many fields and several quoted
// values (one containing escaped quotes). It lives in testdata so the bench
// module's comparison suite can read the same bytes (as sampleBig) instead of
// keeping a copy that could drift; the cross-suite ratios rely on the inputs
// staying identical.
var sample2 = func() []byte {
	b, err := os.ReadFile("testdata/sample_big.txt")
	if err != nil {
		panic(err)
	}
	return bytes.TrimSuffix(b, []byte("\n"))
}()

// Benchmark_IterateOur-22          2278376               515.5 ns/op             0 B/op          0 allocs/op
// Benchmark_IterateOur-16          2726078               443.3 ns/op             0 B/op          0 allocs/op  (scalar IndexByte)
// Benchmark_IterateOur-16          8696526               277.7 ns/op             0 B/op          0 allocs/op  (SWAR key/value scan)
func Benchmark_IterateOur(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Iterate(sample2, func(k, v []byte) bool {
			return true
		})
	}
}

func Benchmark_GetMany_TimestampLevel(b *testing.B) {
	keys := []string{"timestamp", "level"}
	buf := make([][]byte, len(keys))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = GetMany(sample2, keys, buf)
	}
}

func Benchmark_Unescape(b *testing.B) {
	buffer := []byte(`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb cccccccccccccccccccccccccccccccccccc foo=\"bar baz\" qux`)
	dst := make([]byte, 0, len(buffer)*2)
	for i := 0; i < b.N; i++ {
		_ = AppendUnescape(dst[:0], buffer)
	}
}

// Benchmark_IterateEscaped sweeps escape DENSITY at a fixed line length, which
// is the one axis sample2 cannot show: it carries 2 escaped quotes in 1.4 KB, so
// the quoted scan looks like pure bytes.IndexByte throughput there.
//
// Every escaped quote makes the quoted-value loop restart the non-inlinable
// bytes.IndexByte and re-walk the preceding backslash run, so the cost is
// O(escapes) calls rather than O(bytes/8) SWAR steps. Embedded JSON in a msg=
// field — the commonest hard shape in real logfmt — turns every JSON quote into
// \", which is the worst case: roughly one call per two bytes. Keep the length
// fixed across the sweep so the numbers isolate density from size.
func Benchmark_IterateEscaped(b *testing.B) {
	const width = 1024
	for _, esc := range []int{0, 8, 32, 128, 500} {
		payload := escapedPayload(width, esc)
		line := append(append([]byte(`msg="`), payload...), '"')
		b.Run(fmt.Sprintf("esc=%d", esc), func(b *testing.B) {
			b.SetBytes(int64(len(line)))
			for i := 0; i < b.N; i++ {
				if err := Iterate(line, func(k, v []byte) bool { return true }); err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// escapedPayload builds width bytes carrying esc escaped quotes, spread evenly.
// Shared by the two density sweeps so they measure the same shape at the same
// densities: one parses it, the other decodes it, and their numbers are only
// comparable if the bytes are.
func escapedPayload(width, esc int) []byte {
	payload := make([]byte, 0, width)
	for len(payload) < width {
		if esc > 0 && len(payload)*esc/width < (len(payload)+2)*esc/width {
			payload = append(payload, '\\', '"')
		} else {
			payload = append(payload, 'a')
		}
	}
	return payload[:width]
}

// Benchmark_UnescapeEscaped sweeps escape density for the DECODE half of the
// package, which Benchmark_Unescape (2 escapes in 130 bytes) cannot show any
// more than sample2 could show it for the parse half. The parser and the
// decoder are handed the same bytes at the same densities on purpose: a value
// dense enough to be worth this benchmark is one whose escapes the parser has
// just had to scan past, and at high density decoding is the slower half.
func Benchmark_UnescapeEscaped(b *testing.B) {
	const width = 1024
	for _, esc := range []int{0, 8, 32, 128, 500} {
		payload := escapedPayload(width, esc)
		dst := make([]byte, 0, width)
		b.Run(fmt.Sprintf("esc=%d", esc), func(b *testing.B) {
			b.SetBytes(int64(len(payload)))
			for i := 0; i < b.N; i++ {
				dst = AppendUnescape(dst[:0], payload)
			}
			if len(dst) == 0 && esc == 0 {
				b.Fatal("empty decode")
			}
		})
	}
}

// sampleJSONMsg is the commonest hard shape in real logfmt: a structured event
// serialised into a msg= field by an encoder that escapes every inner quote.
// It is what the escape-dense scan in the parser (and the matching one in
// AppendUnescape) exists for; the density here — one escape per ~7 bytes — is
// between the esc=128 and esc=500 rows of the synthetic sweep above.
var sampleJSONMsg = []byte(`time=2025-01-01T00:00:00Z level=info msg="{\"level\":\"info\",\"ts\":\"2025-01-01T00:00:00Z\",\"caller\":\"server/handler.go:42\",\"msg\":\"request completed\",\"user\":\"bob\",\"path\":\"/api/v1/users\",\"status\":200,\"duration_ms\":12.4}" request_id=abc123`)

func Benchmark_IterateJSONMsg(b *testing.B) {
	b.SetBytes(int64(len(sampleJSONMsg)))
	for i := 0; i < b.N; i++ {
		if err := Iterate(sampleJSONMsg, func(k, v []byte) bool { return true }); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// Benchmark_UnescapeJSONMsg decodes the msg value of sampleJSONMsg — the step
// a consumer of embedded JSON cannot skip, and one that used to cost more than
// parsing the whole line.
func Benchmark_UnescapeJSONMsg(b *testing.B) {
	raw, quoted, ok := GetQuoted(sampleJSONMsg, "msg")
	if !ok || !quoted {
		b.Fatal("msg not found or not quoted")
	}
	dst := make([]byte, 0, len(raw))
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = AppendUnescape(dst[:0], raw)
	}
	if len(dst) == 0 {
		b.Fatal("empty decode")
	}
}

// go test -bench=Benchmark_DecodeKeyval -benchmem -memprofile memprofile.out -cpuprofile profile.out -benchtime=30s
// Benchmark_DecodeKeyval-22           2197            549836 ns/op         909.36 MB/s       40000 B/op      10000 allocs/op
func Benchmark_DecodeKeyval_Custom(b *testing.B) {
	const rows = 10000
	data := []byte{}
	for i := 0; i < rows; i++ {
		data = append(data, "a=1 b=\"bar\" ƒ=2h3s r=\"esc\\tmore stuff\" d x=sf   \n"...)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Iterate(data, func(k, v []byte) bool {
			return true
		})
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// gapPayload builds width bytes carrying an escaped quote every gap bytes, with
// the FIRST escape also gap bytes in — so one parameter drives both quantities
// escGap and escClean trade against: escGap reads the distance to the first
// escape, escClean the distance between later ones.
func gapPayload(width, gap int) []byte {
	p := make([]byte, 0, width+2)
	for len(p)+gap <= width {
		for k := 0; k < gap-2; k++ {
			p = append(p, 'a')
		}
		p = append(p, '\\', '"')
	}
	for len(p) < width {
		p = append(p, 'a')
	}
	return p[:width]
}

// escGapPoints brackets the two decisions rather than sampling evenly: 32-64 is
// where the walk hands over to bytes.IndexByte on amd64, and 16 / 128 / 256 are
// the far ends that must not move when the thresholds are retuned.
var escGapPoints = []int{16, 32, 40, 48, 64, 128, 256}

// Benchmark_IterateEscapedGap sweeps the SAME axis as Benchmark_IterateEscaped
// but parameterised by the distance between escapes rather than by a count, and
// it exists because the count-parameterised sweep has a blind spot that has now
// cost two tuning passes. Its rows jump straight from 32-byte gaps to 8-byte
// gaps, and real logfmt sits in between — testdata/sample_big.txt's own escapes
// are 38 bytes apart. The first version of the escape-dense scan lost 7% on
// GetMany inside that gap without any committed row moving, and escClean=8 was
// later found to be costing 8.6% at a 48-byte gap with, again, every committed
// row reading ~.
//
// Tune escGap and escClean against THIS, the 1.4 KB sample and the JSON-msg
// benchmarks together. Keep Benchmark_IterateEscaped as well: its numbers are
// what the committed tables have been tracking across releases.
func Benchmark_IterateEscapedGap(b *testing.B) {
	for _, gap := range escGapPoints {
		line := append(append([]byte(`msg="`), gapPayload(1024, gap)...), '"')
		b.Run(fmt.Sprintf("gap=%03d", gap), func(b *testing.B) {
			b.SetBytes(int64(len(line)))
			for i := 0; i < b.N; i++ {
				if err := Iterate(line, func(k, v []byte) bool { return true }); err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Benchmark_UnescapeEscapedGap is the same axis for the decoder, which tunes
// unescWindow rather than escClean. Four words is the measured setting on both
// arm64 and amd64 — unlike escClean, this one did travel — so a change here
// wants both arches before it lands.
func Benchmark_UnescapeEscapedGap(b *testing.B) {
	for _, gap := range escGapPoints {
		p := gapPayload(1024, gap)
		dst := make([]byte, 0, 1024)
		b.Run(fmt.Sprintf("gap=%03d", gap), func(b *testing.B) {
			b.SetBytes(int64(len(p)))
			for i := 0; i < b.N; i++ {
				dst = AppendUnescape(dst[:0], p)
			}
		})
	}
}

// Benchmark_IteratePrefixJSON pins the escGap cliff documented beside the
// constants: a prose prefix, then an escape-dense JSON tail. The distance to
// the first escape is the prefix length, so the prefix alone decides which scan
// handles the whole value, and there is no way back once it is decided. Expect
// a step of roughly 60% between the prefix=032 and prefix=064 rows, flat on
// either side — that step IS the cliff, and it is the row to watch if the
// sparse-to-dense upgrade is ever implemented.
func Benchmark_IteratePrefixJSON(b *testing.B) {
	tail := []byte(`{\"level\":\"info\",\"caller\":\"server/handler.go:42\",` +
		`\"msg\":\"request completed\",\"user\":\"bob\",\"status\":200}`)
	for _, pre := range []int{8, 32, 64, 160} {
		line := append([]byte(`msg="`), bytes.Repeat([]byte("x"), pre)...)
		line = append(append(append(line, ' '), tail...), '"')
		b.Run(fmt.Sprintf("prefix=%03d", pre), func(b *testing.B) {
			b.SetBytes(int64(len(line)))
			for i := 0; i < b.N; i++ {
				if err := Iterate(line, func(k, v []byte) bool { return true }); err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// sampleUnicodeValue is the shape go-logfmt's encoder produces for a message
// carrying control characters: each becomes a JSON-style \u00XX escape. The
// doubled backslashes are Go source escaping, so the value really holds six
// bytes per escape, exactly as they arrive on the wire.
//
// Until 2026-08-17 NO benchmark in this package reached hex4 or
// decodeUnicodeEscape at all — every escaped sample here carries \" \\ \t or \n
// and none carries \u — even though decoding \u00XX is the documented
// requirement for round-tripping go-logfmt's own output. hex4 sat on a chain of
// data-dependent range compares for as long as nothing measured it.
var sampleUnicodeValue = []byte("parse failed\\u0009at line 5\\u000acol 12" +
	"\\u0009near token \\u001b[31m<eof>\\u001b[0m\\u000awhile reading" +
	" header\\u0009field name")

var sampleUnicodeLine = append(append(
	[]byte(`level=error msg="`), sampleUnicodeValue...),
	[]byte(`" code=500`)...)

func Benchmark_UnescapeUnicode(b *testing.B) {
	dst := make([]byte, 0, len(sampleUnicodeValue))
	b.SetBytes(int64(len(sampleUnicodeValue)))
	for i := 0; i < b.N; i++ {
		dst = AppendUnescape(dst[:0], sampleUnicodeValue)
	}
	if len(dst) == 0 {
		b.Fatal("empty decode")
	}
}

// Benchmark_AppendValueUnicode is the whole one-call path a consumer of
// go-logfmt output takes: look the key up, then decode it.
func Benchmark_AppendValueUnicode(b *testing.B) {
	dst := make([]byte, 0, len(sampleUnicodeValue))
	b.SetBytes(int64(len(sampleUnicodeLine)))
	for i := 0; i < b.N; i++ {
		var ok bool
		if dst, ok = AppendValue(dst[:0], sampleUnicodeLine, "msg"); !ok {
			b.Fatal("msg missing")
		}
	}
}
