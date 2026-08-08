package logfmt

import (
	"bytes"
	"strconv"
	"time"
	"unsafe"
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
//     1970-01-01 .. 2286-11-20 and excludes negative (pre-1970) values. Ten
//     digits is a digit COUNT, not a magnitude: a zero-padded "0000000000" is
//     accepted and is the epoch itself, so the lower bound is 1970 rather than
//     the 2001-09-09 that unpadded ten-digit values start at.
//   - Millisecond and microsecond epochs (13 and 16 digits, as written by
//     JavaScript's Date.now or Java's System.currentTimeMillis) are rejected;
//     the digit count alone cannot distinguish them from a far-future second
//     epoch. Divide them yourself, or parse with strconv and time.UnixMilli.
//   - Date-only ("2006-01-02"), time-only and other layouts are rejected.
//
// Callers with a known emitter should prefer time.Parse with that emitter's
// exact layout; ParseTime is for mixed-source logs where the layout is not
// known up front.
//
// It copies nothing, at any input length. The only allocations it can incur
// are time.Parse's own, on two shapes this package cannot avoid without
// hand-rolling the layouts:
//
//   - A zone abbreviation the runtime cannot resolve — "+0200 CEST" on a host
//     whose local zone is not CEST — costs 4 allocations for the fabricated
//     Location. This one is an ACCEPTED form: it returns the right instant and
//     ok == true, so "every accepted form is allocation-free" would be wrong.
//     Whether a given abbreviation resolves depends on the host's local zone
//     and on the parsed date, so the same layout can cost 0 or 4.
//   - A value matching no layout at all costs 5, for the discarded
//     *time.ParseError.
//
// Everything else — epochs, RFC3339Nano, and a numeric-offset-plus-UTC value —
// is allocation-free, which is what Test_Unit_ParseTime_Allocs pins.
func ParseTime(ts []byte) (time.Time, bool) {
	ts = bytes.TrimRight(ts, "}],)\"")
	if t, ok := parseUnixTS(ts); ok {
		return t, true
	}
	if len(ts) == 0 {
		return time.Time{}, false
	}
	// time.Parse wants a string and we hold bytes. A `string(ts)` conversion
	// here is provably non-escaping, but the compiler only keeps a
	// non-escaping conversion off the heap while it fits its 32-byte stack
	// buffer — and the timestamps that reach this loop routinely do not: the
	// "-0700 MST" form is 33 bytes ("2026-03-14 06:11:46.397 +0000 UTC") and
	// an RFC3339Nano with nanoseconds and a numeric offset is 35. That made
	// every such call allocate, once per attempted layout, on what is a
	// per-log-line path for this package's callers.
	//
	// Aliasing the caller's bytes avoids the copy at every length. It is safe
	// because ts is read-only for the duration of the call and time.Parse does
	// not retain value: the compiler proves it (an escaping value would put
	// the plain conversion on the heap too, which Test_Unit_ParseTime_Allocs
	// would catch at ANY length), and the one place a parsed Time can capture
	// a slice of value — the fabricated FixedZone for a zone abbreviation the
	// layout matched but the runtime cannot resolve — is discarded by the
	// .UTC() below before the Time is returned. Nothing reachable by the
	// caller aliases ts.
	s := unsafe.String(unsafe.SliceData(ts), len(ts))
	for _, layout := range logFmtLayouts {
		// Only RFC3339Nano carries a 'T' date/time separator at index 10, so a
		// 'T'-vs-space disagreement there means time.Parse would fail for nothing.
		if len(ts) > 10 && len(layout) > 10 && (layout[10] == 'T') != (ts[10] == 'T') {
			continue
		}
		if t, err := time.Parse(layout, s); err == nil {
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
