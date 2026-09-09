package logfmt

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// Vary query order and selectivity as well as count: an index that wins only
// when every requested key appears in order is not a useful general speedup.
func Benchmark_GetManyShapes(b *testing.B) {
	for _, n := range []int{16, 64, 256} {
		for _, shape := range []string{"ordered", "reverse", "missing", "missing-same-length", "collisions", "reverse-collisions", "short-record"} {
			keys := make([]string, n)
			var data []byte
			for j := range keys {
				key := fmt.Sprintf("field_%03d", j)
				if strings.Contains(shape, "collisions") {
					key += "_same"
				}
				keys[j] = key
				data = fmt.Appendf(data, "%s=value ", key)
			}
			switch shape {
			case "reverse", "reverse-collisions":
				for j := 0; j < n/2; j++ {
					keys[j], keys[n-1-j] = keys[n-1-j], keys[j]
				}
			case "missing":
				for j := range keys {
					keys[j] = "other" + keys[j]
				}
			case "missing-same-length":
				for j := range keys {
					keys[j] = "other" + keys[j][5:]
				}
			case "short-record":
				data = []byte("field_000=one")
			}
			b.Run(fmt.Sprintf("keys=%03d/%s", n, shape), func(b *testing.B) {
				buf := make([][]byte, n)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					buf = GetMany(data, keys, buf)
				}
			})
		}
	}
}

func TestAppendUnescapeBufferReuse(t *testing.T) {
	for _, raw := range []string{`\n`, strings.Repeat(`\n`, 512), string(sampleUnicodeValue), `\uD83D\uDE00\uZZZZ\`, `plain`} {
		input := []byte(raw)
		want := unescapeRef(input)
		prefix := []byte("prefix:")
		// Enough for decoded bytes but too small for the raw representation:
		// reserving len(raw) unconditionally would allocate unnecessarily.
		dst := make([]byte, len(prefix), len(prefix)+len(want))
		copy(dst, prefix)
		if allocs := testing.AllocsPerRun(100, func() {
			out := AppendUnescape(dst, input)
			if !bytes.Equal(out[:len(prefix)], prefix) || !bytes.Equal(out[len(prefix):], want) {
				t.Fatal("incorrect decode with exact-capacity destination")
			}
		}); allocs != 0 {
			t.Fatalf("%q: exact-capacity destination allocated %g times", raw, allocs)
		}
		if allocs := testing.AllocsPerRun(100, func() { AppendUnescape(nil, input) }); allocs != 1 {
			t.Fatalf("fresh buffer allocated %g times, want 1", allocs)
		}
		if string(input) != raw {
			t.Fatal("decoder modified source")
		}
	}
}
