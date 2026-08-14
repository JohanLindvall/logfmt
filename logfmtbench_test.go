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
// more than sample2 could show it for the parse half.
//
// AppendUnescape restarts bytes.IndexByte at every escape, so like the quoted
// value scan its cost is O(escapes) non-inlinable calls rather than O(bytes) of
// copying — and the same input drives both, since a value dense enough to be
// worth this benchmark is one whose escapes the parser just had to scan past.
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
		})
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
