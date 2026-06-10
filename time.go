package logfmt

import (
	"strconv"
	"strings"
	"time"
)

var logFmtLayouts = []string{time.RFC3339Nano, "2006-01-02 15:04:05.999 -0700 MST"}

// ParseTime parses a logfmt timestamp value and reports whether it succeeded. It
// accepts an RFC3339Nano string, a "2006-01-02 15:04:05.999 -0700 MST" string, or
// a unix epoch (10 integer digits with an optional fractional part). Trailing
// delimiters left over from a slightly malformed line (e.g. a stray '}') are
// trimmed first. On success the returned time is normalized to UTC.
func ParseTime(ts string) (time.Time, bool) {
	ts = strings.TrimRight(ts, "}],)\"")
	if t, ok := parseUnixTS(ts); ok {
		return t, true
	}
	for _, layout := range logFmtLayouts {
		// Only RFC3339Nano carries a 'T' date/time separator at index 10, so a
		// 'T'-vs-space disagreement there means time.Parse would fail for nothing.
		if len(ts) > 10 && len(layout) > 10 && (layout[10] == 'T') != (ts[10] == 'T') {
			continue
		}
		if t, err := time.Parse(layout, ts); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// parseUnixTS parses a unix epoch timestamp of exactly 10 integer digits with an
// optional fractional part of up to 9 digits (e.g. "1748239806.3691056").
func parseUnixTS(ts string) (time.Time, bool) {
	intPart, fracPart := ts, ""
	if dot := strings.IndexByte(ts, '.'); dot >= 0 {
		intPart, fracPart = ts[:dot], ts[dot+1:]
	}
	if len(intPart) != 10 || len(fracPart) > 9 || !allDigits(intPart) || !allDigits(fracPart) {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	var nsec int64
	if fracPart != "" {
		nsec, _ = strconv.ParseInt(fracPart, 10, 64)
		for mul := len(fracPart); mul < 9; mul++ {
			nsec *= 10
		}
	}
	return time.Unix(sec, nsec).UTC(), true
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
