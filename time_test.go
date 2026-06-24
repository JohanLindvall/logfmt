package logfmt

import (
	"fmt"
	"testing"
	"time"
)

func Test_Unit_ParseTime(t *testing.T) {
	for i, tt := range []struct {
		ts       string
		expected time.Time
	}{
		{
			"2025-05-26T06:10:06.3691056Z",
			time.Date(2025, 5, 26, 6, 10, 6, 369105600, time.UTC),
		},
		{
			"2025-05-26T08:10:06+02:00",
			time.Date(2025, 5, 26, 6, 10, 6, 0, time.UTC),
		},
		{
			"2025-05-26 08:10:06.369 +0200 CEST",
			time.Date(2025, 5, 26, 6, 10, 6, 369000000, time.UTC),
		},
		{
			"1748239806",
			time.Date(2025, 5, 26, 6, 10, 6, 0, time.UTC),
		},
		{
			"1748239806.3691056",
			time.Date(2025, 5, 26, 6, 10, 6, 369105600, time.UTC),
		},
		{
			// trailing delimiters left over from a slightly malformed line
			`2025-05-26T06:10:06Z}`,
			time.Date(2025, 5, 26, 6, 10, 6, 0, time.UTC),
		},
		{
			"1748239806.3691056\"",
			time.Date(2025, 5, 26, 6, 10, 6, 369105600, time.UTC),
		},
	} {
		t.Run(fmt.Sprintf("test-%d-%s", i, tt.ts), func(t *testing.T) {
			got, ok := ParseTime(tt.ts)
			if !ok {
				t.Fatalf("ParseTime(%q) reported failure, want success", tt.ts)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("ParseTime(%q) = %v, want %v", tt.ts, got, tt.expected)
			}
			if got.Location() != time.UTC {
				t.Errorf("ParseTime(%q) location = %v, want UTC", tt.ts, got.Location())
			}
		})
	}
}

func Benchmark_ParseTime_RFC3339(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, ok := ParseTime("2025-05-26T06:10:06.3691056Z"); !ok {
			b.Fatal("failed")
		}
	}
}

func Benchmark_ParseTime_Custom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, ok := ParseTime("2025-05-26 08:10:06.369 +0200 CEST"); !ok {
			b.Fatal("failed")
		}
	}
}

func Benchmark_ParseTime_Unix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, ok := ParseTime("1748239806.3691056"); !ok {
			b.Fatal("failed")
		}
	}
}

func Test_Unit_ParseTime_Invalid(t *testing.T) {
	for i, tt := range []string{
		"",
		"not a time",
		"174823980",             // 9 digits, too short for unix epoch
		"17482398066",           // 11 digits, too long for unix epoch
		"1748239806.1234567890", // fractional part of 10 digits, too long
		"1748239806.abc",        // non-digit fractional part
		"2025-13-26T06:10:06Z",  // invalid month
	} {
		t.Run(fmt.Sprintf("test-%d-%s", i, tt), func(t *testing.T) {
			if got, ok := ParseTime(tt); ok {
				t.Errorf("ParseTime(%q) = %v, want failure", tt, got)
			}
		})
	}
}
