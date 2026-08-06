// Package bench compares this logfmt parser against other Go logfmt parsers on
// representative input. It lives in its own module (see go.mod) so the root
// package keeps zero dependencies.
//
//	cd bench && go test -bench=. -benchmem
package bench

import (
	"bytes"
	"os"
	"testing"

	mine "github.com/JohanLindvall/logfmt"
	lokifmt "github.com/JohanLindvall/logfmt/bench/lokifmt"
	golog "github.com/go-logfmt/logfmt"
	krlog "github.com/kr/logfmt"
)

// sampleBig is a real-world ~1.4 KB line with many fields and several quoted
// values (one containing escaped quotes) — the same bytes the root package
// benchmarks as sample2, read from the shared testdata file so the two suites
// cannot drift apart. (This module always sits inside the repo — its go.mod
// replaces the root module with ../ — so the relative path is safe.)
var sampleBig = func() []byte {
	b, err := os.ReadFile("../testdata/sample_big.txt")
	if err != nil {
		panic(err)
	}
	return bytes.TrimSuffix(b, []byte("\n"))
}()

// sampleTypical is a shorter, everyday application log line.
var sampleTypical = []byte(`time=2025-01-01T00:00:00Z level=info msg="request completed" method=GET path=/api/v1/users status=200 duration=12.4ms request_id=abc123`)

// sampleEscaped mixes values that need decoding (escaped quotes, \n, \t,
// backslashes) with plain ones, to exercise the unescape path.
var sampleEscaped = []byte(`level=error ts=2025-01-01T00:00:00Z msg="parse failed: \"unexpected token\"\n\tat line 5" path="C:\\logs\\app.log" detail="a\tb\tc" code=500 ok=false`)

// escSink keeps the consumed value lengths live so the compiler cannot elide the
// decode work.
var escSink int

// --- Parse every key/value pair --------------------------------------------

func parseAllMine(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_ = mine.Iterate(data, func(k, v []byte) bool { return true })
	}
}

func parseAllGoLogfmt(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		d := golog.NewDecoder(bytes.NewReader(data))
		for d.ScanRecord() {
			for d.ScanKeyval() {
				_ = d.Key()
				_ = d.Value()
			}
		}
		if err := d.Err(); err != nil {
			b.Fatal(err)
		}
	}
}

func parseAllKr(b *testing.B, data []byte) {
	h := krlog.HandlerFunc(func(key, val []byte) error { return nil })
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		if err := krlog.Unmarshal(data, h); err != nil {
			b.Fatal(err)
		}
	}
}

// parseAllLoki uses the lokifmt stand-in for Grafana Loki's in-tree decoder
// (go-logfmt's scanner on a []byte line; see lokifmt's package doc for why it
// is a reimplementation rather than a vendored copy).
func parseAllLoki(b *testing.B, data []byte) {
	dec := lokifmt.NewDecoder(data)
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		dec.Reset(data)
		for dec.ScanKeyval() {
			_ = dec.Key()
			_ = dec.Value()
		}
		if err := dec.Err(); err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_ParseAll_Big_Mine(b *testing.B)         { parseAllMine(b, sampleBig) }
func Benchmark_ParseAll_Big_GoLogfmt(b *testing.B)     { parseAllGoLogfmt(b, sampleBig) }
func Benchmark_ParseAll_Big_Loki(b *testing.B)         { parseAllLoki(b, sampleBig) }
func Benchmark_ParseAll_Big_Kr(b *testing.B)           { parseAllKr(b, sampleBig) }
func Benchmark_ParseAll_Typical_Mine(b *testing.B)     { parseAllMine(b, sampleTypical) }
func Benchmark_ParseAll_Typical_GoLogfmt(b *testing.B) { parseAllGoLogfmt(b, sampleTypical) }
func Benchmark_ParseAll_Typical_Loki(b *testing.B)     { parseAllLoki(b, sampleTypical) }
func Benchmark_ParseAll_Typical_Kr(b *testing.B)       { parseAllKr(b, sampleTypical) }

// --- Parse + decode (unescape) every value ---------------------------------
// go-logfmt, kr/logfmt and Loki unescape values eagerly; this package returns
// raw values, so it decodes with Unescape here for an apples-to-apples
// comparison on escaped input.

func Benchmark_ParseEscaped_Mine(b *testing.B) {
	var dst []byte
	n := 0
	b.SetBytes(int64(len(sampleEscaped)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mine.Iterate(sampleEscaped, func(k, v []byte) bool {
			if mine.NeedsUnescape(v) {
				dst = mine.AppendUnescape(dst[:0], v)
				n += len(dst)
			} else {
				n += len(v)
			}
			return true
		})
	}
	escSink = n
}

func Benchmark_ParseEscaped_GoLogfmt(b *testing.B) {
	n := 0
	b.SetBytes(int64(len(sampleEscaped)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := golog.NewDecoder(bytes.NewReader(sampleEscaped))
		for d.ScanRecord() {
			for d.ScanKeyval() {
				n += len(d.Value())
			}
		}
		if err := d.Err(); err != nil {
			b.Fatal(err)
		}
	}
	escSink = n
}

func Benchmark_ParseEscaped_Loki(b *testing.B) {
	dec := lokifmt.NewDecoder(sampleEscaped)
	n := 0
	b.SetBytes(int64(len(sampleEscaped)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dec.Reset(sampleEscaped)
		for dec.ScanKeyval() {
			n += len(dec.Value())
		}
		if err := dec.Err(); err != nil {
			b.Fatal(err)
		}
	}
	escSink = n
}

func Benchmark_ParseEscaped_Kr(b *testing.B) {
	n := 0
	h := krlog.HandlerFunc(func(key, val []byte) error {
		n += len(val)
		return nil
	})
	b.SetBytes(int64(len(sampleEscaped)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := krlog.Unmarshal(sampleEscaped, h); err != nil {
			b.Fatal(err)
		}
	}
	escSink = n
}

// --- Targeted extraction of two keys (timestamp + level) -------------------
// All variants stop scanning once both keys are found, where the API allows it:
// GetMany and the pull-based decoders (go-logfmt, Loki) break out of the loop;
// kr/logfmt is push-based (Unmarshal drives a handler and cannot be told to
// stop — its scanner records the first handler error but keeps scanning), so it
// can only skip the per-pair work, not the scan itself.

func Benchmark_Extract_Mine(b *testing.B) {
	keys := []string{"timestamp", "level"}
	buf := make([][]byte, len(keys))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf = mine.GetMany(sampleBig, keys, buf)
	}
}

func Benchmark_Extract_GoLogfmt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var ts, lvl []byte
		d := golog.NewDecoder(bytes.NewReader(sampleBig))
	scan:
		for d.ScanRecord() {
			for d.ScanKeyval() {
				switch string(d.Key()) {
				case "timestamp":
					ts = d.Value()
				case "level":
					lvl = d.Value()
				}
				if ts != nil && lvl != nil {
					break scan // both found: stop scanning
				}
			}
		}
		_, _ = ts, lvl
	}
}

func Benchmark_Extract_Loki(b *testing.B) {
	dec := lokifmt.NewDecoder(sampleBig)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var ts, lvl []byte
		dec.Reset(sampleBig)
		for dec.ScanKeyval() {
			switch string(dec.Key()) {
			case "timestamp":
				ts = dec.Value()
			case "level":
				lvl = dec.Value()
			}
			if ts != nil && lvl != nil {
				break // both found: stop scanning
			}
		}
		_, _ = ts, lvl
	}
}

func Benchmark_Extract_Kr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var ts, lvl []byte
		h := krlog.HandlerFunc(func(key, val []byte) error {
			if ts != nil && lvl != nil {
				return nil // both found: skip work (the scan can't be stopped)
			}
			switch string(key) {
			case "timestamp":
				ts = val
			case "level":
				lvl = val
			}
			return nil
		})
		_ = krlog.Unmarshal(sampleBig, h)
		_, _ = ts, lvl
	}
}
