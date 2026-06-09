package logfmt

import (
	"testing"
)

var sample2 = []byte(`timestamp="2025-01-01 00:00:00.000 +0000 UTC" kind=log message="[AF] on conversion CUID set to: \"00000000-0000-4000-8000-000000000000\"" level=warn sdk_version=1.0.0 app_name=Sample-Client app_version=20250101.01 session_attr_cf_colo=XXX session_attr_cf_ray=0000000000000000 session_attr_client_brand=example session_attr_client_id=11111111-1111-4111-8111-111111111111 session_attr_client_ip=203.0.113.0 session_attr_client_jurisdiction=zz session_attr_client_locale=en session_attr_client_location=zz session_attr_ff_casino_lobby_swimlanes_orientation_ab_flag=true session_attr_ff_f_registration_sheet_ab_flag=false session_attr_visit_id=22222222-2222-4222-8222-222222222222 session_attr_visitor_id=33333333-3333-4333-8333-333333333333 session_attr_visitor_location=ZZ session_attr_wrapper_name=ExampleSportsAndroid session_attr_wrapper_type=SampleSport_Android session_attr_wrapper_version=14.20250101.1 page_url="https://example.com/zz/en/sports?wrapperName=ExampleSportsAndroid&wrapperType=SampleSport_Android&wrapperVersion=14.20250101.1&wrapperAlias=example.com&deviceId=0000000000000000&wrapperStore=exampleStore" browser_name="Chrome WebView" browser_version=140 browser_os="Android unknown" browser_mobile=true browser_userAgent="Mozilla/5.0 (Linux; Android 15; SM-X000X Build/AAAA.000000.000.A0; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/140.0.0.0 Mobile Safari/537.36"`)

// Benchmark_IterateOur-22          2278376               515.5 ns/op             0 B/op          0 allocs/op
// Benchmark_IterateOur-16          2726078               443.3 ns/op             0 B/op          0 allocs/op
func Benchmark_IterateOur(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Iterate(sample2, func(k, v []byte) bool {
			return true
		})
	}
}

func Benchmark_UnescapeInto(b *testing.B) {
	buffer := []byte(`aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb cccccccccccccccccccccccccccccccccccc foo=\"bar baz\" qux`)
	dst := make([]byte, 0, len(buffer)*2)
	for i := 0; i < b.N; i++ {
		_ = UnescapeInto(dst[:0], buffer)
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
