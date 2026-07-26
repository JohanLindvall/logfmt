package logfmt

import (
	"bytes"
	"strconv"
	"time"
)

var logFmtLayouts = []string{time.RFC3339Nano, "2006-01-02 15:04:05.999 -0700 MST"}

// ParseTime parses a logfmt timestamp value and reports whether it succeeded. It
// accepts an RFC3339Nano string, a "2006-01-02 15:04:05.999 -0700 MST" string, or
// a unix epoch (10 integer digits with an optional fractional part). Trailing
// delimiters left over from a slightly malformed line (e.g. a stray '}') are
// trimmed first. On success the returned time is normalized to UTC.
//
// The accepted set is deliberately narrow — these are the shapes real logfmt
// emitters produce — so several plausible-looking timestamps are rejected
// rather than guessed at:
//
//   - Epochs must have exactly 10 integer digits, which bounds them to
//     2001-09-09 .. 2286-11-20 and excludes negative (pre-1970) values.
//   - Millisecond and microsecond epochs (13 and 16 digits, as written by
//     JavaScript's Date.now or Java's System.currentTimeMillis) are rejected;
//     the digit count alone cannot distinguish them from a far-future second
//     epoch. Divide them yourself, or parse with strconv and time.UnixMilli.
//   - Date-only ("2006-01-02"), time-only and other layouts are rejected.
//
// Callers with a known emitter should prefer time.Parse with that emitter's
// exact layout; ParseTime is for mixed-source logs where the layout is not
// known up front. It performs no allocations for the RFC3339 and epoch forms.
func ParseTime(ts []byte) (time.Time, bool) {
	ts = bytes.TrimRight(ts, "}],)\"")
	if t, ok := parseUnixTS(ts); ok {
		return t, true
	}
	for _, layout := range logFmtLayouts {
		// Only RFC3339Nano carries a 'T' date/time separator at index 10, so a
		// 'T'-vs-space disagreement there means time.Parse would fail for nothing.
		if len(ts) > 10 && len(layout) > 10 && (layout[10] == 'T') != (ts[10] == 'T') {
			continue
		}
		// The conversion does not escape into time.Parse, so it costs no
		// allocation (pinned by Test_Unit_ParseTime_Allocs).
		if t, err := time.Parse(layout, string(ts)); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// parseUnixTS parses a unix epoch timestamp of exactly 10 integer digits with an
// optional fractional part of up to 9 digits (e.g. "1748239806.3691056").
func parseUnixTS(ts []byte) (time.Time, bool) {
	intPart, fracPart := ts, []byte(nil)
	if dot := bytes.IndexByte(ts, '.'); dot >= 0 {
		intPart, fracPart = ts[:dot], ts[dot+1:]
	}
	if len(intPart) != 10 || len(fracPart) > 9 || !allDigits(intPart) || !allDigits(fracPart) {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(string(intPart), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	var nsec int64
	if len(fracPart) > 0 {
		nsec, _ = strconv.ParseInt(string(fracPart), 10, 64)
		for mul := len(fracPart); mul < 9; mul++ {
			nsec *= 10
		}
	}
	return time.Unix(sec, nsec).UTC(), true
}

func allDigits(s []byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
