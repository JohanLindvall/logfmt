package logfmt

import (
	"bytes"
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
