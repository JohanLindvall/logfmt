package logfmt

import (
	"bytes"
	"fmt"
	"testing"
)

func Benchmark_GetManyScale(b *testing.B) {
	for _, count := range []int{0, 1, 2, 8, 16, 32, 64, 128} {
		var data []byte
		keys := make([]string, count)
		for i := range keys {
			keys[i] = fmt.Sprintf("field_%03d", i)
			data = fmt.Appendf(data, "%s=value_%03d ", keys[i], i)
		}
		if count == 0 {
			data = sample2
		}
		b.Run(fmt.Sprintf("keys=%03d", count), func(b *testing.B) {
			buf := make([][]byte, count)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf = GetMany(data, keys, buf)
			}
		})
	}
}

func Benchmark_UnescapeBuffer(b *testing.B) {
	for _, shape := range []struct {
		name string
		raw  []byte
	}{
		{"json", sampleJSONMsg},
		{"dense", bytes.Repeat([]byte(`\n`), 512)},
		{"unicode", sampleUnicodeValue},
		{"surrogates", bytes.Repeat([]byte(`\ud83d\ude00`), 16)},
		{"malformed", bytes.Repeat([]byte(`\uZZZZ\u12`), 16)},
		{"plain", bytes.Repeat([]byte("a"), 1024)},
	} {
		for _, reuse := range []bool{false, true} {
			b.Run(fmt.Sprintf("%s/reuse=%t", shape.name, reuse), func(b *testing.B) {
				var dst []byte
				if reuse {
					dst = make([]byte, 0, len(shape.raw))
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					out := AppendUnescape(dst, shape.raw)
					if len(out) == 0 {
						b.Fatal("empty output")
					}
				}
			})
		}
	}
}
