#!/usr/bin/env bash
#
# Run the root-module microbenchmarks (parser, lookups, unescape, ParseTime) and
# render an architecture-specific markdown summary (bench/pkg_results_<goarch>.md),
# so runs on different CPUs do not clobber each other's committed results. The
# cross-library comparison suite is separate (bench/run_bench.sh).
#
# Usage: pkg_bench.sh   (or `make bench-md`, which also runs the comparison suite)
# Env: BENCHTIME (default 1s), BENCHCOUNT (default 1), NOTE (extra header line,
#      e.g. "GOARCH=arm64 under QEMU (emulated; timings indicative only)").
set -u

cd "$(dirname "$0")" # the root module

RESULTS="bench/pkg_results.txt"
RESULTSMD="bench/pkg_results_$(go env GOARCH).md"
BENCHTIME="${BENCHTIME:-1s}"
BENCHCOUNT="${BENCHCOUNT:-1}"

{
	echo "# logfmt microbenchmarks"
	echo "# generated $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "# $(go version)"
	[ -n "${NOTE:-}" ] && echo "# note $NOTE"
	echo
} >"$RESULTS"

status=0
if go test -run='^$' -bench="${1:-.}" -benchmem -timeout=0 \
	-benchtime="$BENCHTIME" -count="$BENCHCOUNT" . >>"$RESULTS" 2>&1; then
	echo "  ok"
else
	echo "  benchmark failed (see ${RESULTS})" >&2
	status=1
fi

if command -v python3 >/dev/null 2>&1; then
	python3 bench/pkg_results_md.py "$RESULTS" "$RESULTSMD" || true
fi

echo "results written to ${RESULTS} and ${RESULTSMD}"
exit $status
