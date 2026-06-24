#!/usr/bin/env bash
#
# Run the cross-library comparison suite (this package vs go-logfmt, kr/logfmt,
# and Grafana Loki's vendored decoder) and render an architecture-specific
# markdown summary (results_<goarch>.md). Run from inside the bench module.
#
# Usage: bench/run_bench.sh   (or `make bench-md`, which also runs pkg_bench.sh)
# Env: BENCHTIME (default 1s), BENCHCOUNT (default 1), NOTE (extra header line).
set -u

cd "$(dirname "$0")" # the bench module

RESULTS="results.txt"
RESULTSMD="results_$(go env GOARCH).md"
BENCHTIME="${BENCHTIME:-1s}"
BENCHCOUNT="${BENCHCOUNT:-1}"

{
	echo "# logfmt parser comparison"
	echo "# generated $(date -u +%Y-%m-%dT%H:%M:%SZ)"
	echo "# $(go version)"
	[ -n "${NOTE:-}" ] && echo "# note $NOTE"
	echo
} >"$RESULTS"

status=0
if go test -run='^$' -bench=. -benchmem -timeout=0 \
	-benchtime="$BENCHTIME" -count="$BENCHCOUNT" . >>"$RESULTS" 2>&1; then
	echo "  ok"
else
	echo "  benchmark failed (see ${RESULTS})" >&2
	status=1
fi

if command -v python3 >/dev/null 2>&1; then
	python3 results_md.py "$RESULTS" "$RESULTSMD" || true
fi

echo "results written to bench/${RESULTS} and bench/${RESULTSMD}"
exit $status
