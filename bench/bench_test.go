// Package bench compares this logfmt parser against other Go logfmt parsers on
// representative input. It lives in its own module (see go.mod) so the root
// package keeps zero dependencies.
//
//	cd bench && go test -bench=. -benchmem
package bench

import (
	"bytes"
	"testing"

	mine "github.com/JohanLindvall/logfmt"
	lokifmt "github.com/JohanLindvall/logfmt/bench/lokifmt"
	golog "github.com/go-logfmt/logfmt"
	krlog "github.com/kr/logfmt"
)

// sampleBig is a real-world ~1.4 KB line with many fields and several quoted
// values (one containing escaped quotes) — the same line the root package
// benchmarks as sample2.
var sampleBig = []byte(`timestamp="2025-01-01 00:00:00.000 +0000 UTC" kind=log message="[AF] on conversion CUID set to: \"00000000-0000-4000-8000-000000000000\"" level=warn sdk_version=1.0.0 app_name=Sample-Client app_version=20250101.01 session_attr_cf_colo=XXX session_attr_cf_ray=0000000000000000 session_attr_client_brand=example session_attr_client_id=11111111-1111-4111-8111-111111111111 session_attr_client_ip=203.0.113.0 session_attr_client_jurisdiction=zz session_attr_client_locale=en session_attr_client_location=zz session_attr_ff_casino_lobby_swimlanes_orientation_ab_flag=true session_attr_ff_f_registration_sheet_ab_flag=false session_attr_visit_id=22222222-2222-4222-8222-222222222222 session_attr_visitor_id=33333333-3333-4333-8333-333333333333 session_attr_visitor_location=ZZ session_attr_wrapper_name=ExampleSportsAndroid session_attr_wrapper_type=SampleSport_Android session_attr_wrapper_version=14.20250101.1 page_url="https://example.com/zz/en/sports?wrapperName=ExampleSportsAndroid&wrapperType=SampleSport_Android&wrapperVersion=14.20250101.1&wrapperAlias=example.com&deviceId=0000000000000000&wrapperStore=exampleStore" browser_name="Chrome WebView" browser_version=140 browser_os="Android unknown" browser_mobile=true browser_userAgent="Mozilla/5.0 (Linux; Android 15; SM-X000X Build/AAAA.000000.000.A0; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/140.0.0.0 Mobile Safari/537.36"`)

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

// parseAllLoki uses Grafana Loki's in-tree zero-alloc decoder (vendored).
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
				dst = mine.Unescape(dst[:0], v)
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
// Mine can early-stop and alias; the others must scan the whole line.

func Benchmark_Extract_Mine(b *testing.B) {
	keys := []string{"timestamp", "level"}
	buf := make([][]byte, len(keys))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, _ = mine.GetMany(sampleBig, keys, buf)
	}
}

func Benchmark_Extract_GoLogfmt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var ts, lvl []byte
		d := golog.NewDecoder(bytes.NewReader(sampleBig))
		for d.ScanRecord() {
			for d.ScanKeyval() {
				switch string(d.Key()) {
				case "timestamp":
					ts = d.Value()
				case "level":
					lvl = d.Value()
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
		}
		_, _ = ts, lvl
	}
}

func Benchmark_Extract_Kr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var ts, lvl []byte
		h := krlog.HandlerFunc(func(key, val []byte) error {
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
