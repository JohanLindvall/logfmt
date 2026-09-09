package logfmt

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// Unlike getManyRef, this reference models duplicate query slots too. Each
// field can fill only the first matching slot that is still unresolved.
func getManySlotsRef(data []byte, keys []string) [][]byte {
	out := make([][]byte, len(keys))
	_ = Iterate(data, func(k, v []byte) bool {
		for j, key := range keys {
			if len(out[j]) != 0 || string(k) != key {
				continue
			}
			if out[j] == nil || len(v) != 0 {
				out[j] = v[:len(v):len(v)]
			}
			break
		}
		return true
	})
	return out
}

func checkManySlots(t *testing.T, data []byte, keys []string) {
	t.Helper()
	want := getManySlotsRef(data, keys)
	buf := make([][]byte, len(keys)+1)
	for j := range buf {
		buf[j] = []byte("stale")
	}
	for _, dst := range [][][]byte{nil, buf[:0]} {
		got := GetMany(data, keys, dst)
		if len(got) != len(want) {
			t.Fatalf("got %d slots, want %d", len(got), len(want))
		}
		for j := range got {
			if !bytes.Equal(got[j], want[j]) || (got[j] == nil) != (want[j] == nil) || IsBareKey(got[j]) != IsBareKey(want[j]) || cap(got[j]) != len(got[j]) {
				t.Fatalf("slot %d key %q: got %q (nil=%t, cap=%d), want %q (nil=%t)", j, keys[j], got[j], got[j] == nil, cap(got[j]), want[j], want[j] == nil)
			}
			if len(got[j]) > 0 && &got[j][0] != &want[j][0] {
				t.Fatalf("slot %d no longer aliases its source", j)
			}
		}
	}
}

func TestGetManyIndexed(t *testing.T) {
	for _, n := range []int{0, 1, 7, 8, 15, 16, 31, 32, 63, 64, 65, 128, 255, 256, 257} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			keys := make([]string, n)
			var data []byte
			for j := range keys {
				// Common suffixes force collisions; repeats test slot ordering.
				keys[j] = fmt.Sprintf("key_%d_same", j%19)
				data = fmt.Appendf(data, "%s= %s=value%d ", keys[j], keys[j], j)
			}
			checkManySlots(t, data, keys)
			checkManySlots(t, []byte("different_length_key=value "+strings.Repeat(" ", n*4)), keys)
			reversed := append([]string(nil), keys...)
			for j := 0; j < len(reversed)/2; j++ {
				reversed[j], reversed[len(reversed)-1-j] = reversed[len(reversed)-1-j], reversed[j]
			}
			checkManySlots(t, data, reversed)
			checkManySlots(t, []byte(`key_0_same="" key_0_same key_1_same= key_1_same="unterminated`+strings.Repeat(" ", n*4)), keys)
			if n > 0 {
				keys[n-1] = strings.Repeat("long", 32)
				data = append(data, (keys[n-1] + "=last")...)
				checkManySlots(t, data, keys)
			}
			buf := make([][]byte, n)
			if allocs := testing.AllocsPerRun(100, func() { GetMany(data, keys, buf) }); allocs != 0 {
				t.Fatalf("%g allocations with reusable buffer", allocs)
			}
		})
	}
}

func FuzzGetManyIndexedAgainstRef(f *testing.F) {
	f.Add(`a= a=1 b=2 a=3`, "a\x00b\x00", uint8(16))
	f.Add(`x= x="" x=first x=second`, "x\x00x\x00missing", uint8(64))
	f.Add(`=empty flag x="" x=value x=last`, "\x00flag\x00x", uint8(64))
	f.Add(`x=good x="broken`, "x\x00missing", uint8(32))
	f.Fuzz(func(t *testing.T, data, query string, count uint8) {
		parts := strings.Split(query, "\x00")
		keys := make([]string, int(count))
		for j := range keys {
			keys[j] = parts[j%len(parts)]
		}
		checkManySlots(t, []byte(data), keys)
		// Pad short records so the indexed path also sees malformed and bare
		// key inputs that would otherwise use the small-record fallback.
		checkManySlots(t, []byte(data+strings.Repeat(" ", len(keys)*4)), keys)
	})
}
