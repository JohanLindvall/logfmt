package logfmt

import (
	"regexp"
	"testing"
)

// Regular-expression baseline for extracting the log level and timestamp from
// a line, handling either field ordering. Compared against the logfmt parser
// in the benchmarks below.
var rfc3339NanoExpr = `"?(?P<time>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z)"?`
var rfc3339NanoSpaceExpr = `"?(?P<time>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?Z)"?`
var logFmtTSExpr = `(` + rfc3339NanoExpr + `|"?(?P<time>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?(\s\+0+\sUTC)?)"?|(?P<unixts>\d{10}(\.\d{1,9})?)` + `)`

var levelTSRe = regexp.MustCompile(
	`(\blevel=(?P<level>[a-zA-Z0-9]+)\s.*?\b(t|ts|time|timestamp)=` + logFmtTSExpr +
		`)|(\b(t|ts|time|timestamp)=` + logFmtTSExpr + `\s.*?\blevel=(?P<level>[a-zA-Z0-9]+)(\s|$))`)

// regexLevelTS extracts the level and timestamp via the regular expression.
func regexLevelTS(line []byte) (level, ts string) {
	m := levelTSRe.FindSubmatch(line)
	if m == nil {
		return
	}
	for i, name := range levelTSRe.SubexpNames() {
		if len(m[i]) == 0 {
			continue
		}
		switch name {
		case "level":
			level = string(m[i])
		case "time", "unixts":
			ts = string(m[i])
		}
	}
	return
}

// logfmtLevelTS extracts the level and timestamp via the logfmt parser.
func logfmtLevelTS(line []byte) (level, ts []byte) {
	_ = Iterate(line, func(k, v []byte) bool {
		switch string(k) {
		case "level":
			level = v
		case "t", "ts", "time", "timestamp":
			ts = v
		}
		return level == nil || ts == nil // stop once both are found
	})
	return
}

func Test_LevelTS_Agree(t *testing.T) {
	rl, rts := regexLevelTS(sample2)
	ll, lts := logfmtLevelTS(sample2)
	if rl != string(ll) {
		t.Errorf("level mismatch: regex %q logfmt %q", rl, ll)
	}
	if rts != string(lts) {
		t.Errorf("timestamp mismatch: regex %q logfmt %q", rts, lts)
	}
	if rl != "warn" {
		t.Errorf("level = %q, want warn", rl)
	}
	if rts != "2025-01-01 00:00:00.000 +0000 UTC" {
		t.Errorf("timestamp = %q", rts)
	}
}

func Benchmark_LevelTS_LogFmt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		level, ts := logfmtLevelTS(sample2)
		if level == nil || ts == nil {
			b.Fatal("not found")
		}
	}
}

func Benchmark_LevelTS_Regex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		level, ts := regexLevelTS(sample2)
		if level == "" || ts == "" {
			b.Fatal("not found")
		}
	}
}
