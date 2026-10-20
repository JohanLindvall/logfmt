# SPDX-License-Identifier: MIT

.DEFAULT_GOAL := help
.PHONY: all help check test test-root test-bench test-race examples fuzz vet fmt-check lint bench bench-md bsfdep fmt fix update-tools

FUZZTIME ?= 20s
FUZZPARALLEL ?= 2
GOLANGCI_LINT_VERSION ?= v2.13.1

GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
PATH := $(GOBIN):$(PATH)
export PATH

help:
	@printf '%s\n' \
		'Development commands (Go 1.23+ for both modules):' \
		'  make check        Check formatting, vet, and test both modules' \
		'  make test-root    Test the library alone (Go 1.21+)' \
		'  make test-bench   Test comparisons and Go 1.23 range examples' \
		'  make test-race    Test both modules with the race detector' \
		'  make examples     Run the documentation examples' \
		'  make fuzz         Run all four fuzzers (20s each, 2 workers)' \
		'                    Override with FUZZTIME=60s FUZZPARALLEL=4' \
		'  make lint         Run golangci-lint (installs it if missing)' \
		'  make fmt          Format Go files' \
		'  make fix          Format Go files and tidy both modules' \
		'  make bench        Run both benchmark suites' \
		'  make bench-md     Regenerate benchmark tables (Bash + Python 3)' \
		'  make bsfdep       Check amd64 BSF dependencies (Python 3)' \
		'  make update-tools Install the pinned golangci-lint version'

all: check

# Read-only checks using the Go toolchain; no formatter writes or tool installs.
check: fmt-check vet test

# Unit tests for both modules. The root module is dependency-free; the bench
# module carries the comparison-only dependencies.
test: test-root test-bench

test-root:
	go test -cover ./...

test-bench:
	cd bench && go test ./...

test-race:
	go test -race ./...
	cd bench && go test -race ./...

examples:
	go test -run='^Example' -v ./...

fuzz:
	go test -run='^$$' -fuzz='^FuzzIterateAgainstRef$$' -fuzztime=$(FUZZTIME) -parallel=$(FUZZPARALLEL) .
	go test -run='^$$' -fuzz='^FuzzGetManyAgainstRef$$' -fuzztime=$(FUZZTIME) -parallel=$(FUZZPARALLEL) .
	go test -run='^$$' -fuzz='^FuzzGetManyIndexedAgainstRef$$' -fuzztime=$(FUZZTIME) -parallel=$(FUZZPARALLEL) .
	go test -run='^$$' -fuzz='^FuzzAppendUnescapeAgainstRef$$' -fuzztime=$(FUZZTIME) -parallel=$(FUZZPARALLEL) .

vet:
	go vet ./...
	cd bench && go vet ./...

fmt-check:
	@files=$$(gofmt -l .) || exit $$?; \
	if [ -n "$$files" ]; then \
		printf 'Run make fmt to format:\n%s\n' "$$files"; \
		exit 1; \
	fi

lint: $(GOBIN)/golangci-lint
	golangci-lint run ./...
	cd bench && golangci-lint run ./...

# Run every benchmark (root microbenchmarks, then the comparison suite) without
# rendering markdown — a quick smoke check.
bench:
	go test -run='^$$' -bench=. -benchmem .
	cd bench && go test -run='^$$' -bench=. -benchmem .

# Regenerate the committed, architecture-specific benchmark tables:
#   bench/pkg_results_<arch>.md   (root microbenchmarks)
#   bench/results_<arch>.md       (cross-library comparison)
# Both suites run each case for 2s for steadier committed numbers.
bench-md:
	BENCHTIME=2s bash pkg_bench.sh
	BENCHTIME=2s bash bench/run_bench.sh

# List every BSF in Iterate with its destination register's possible previous
# writers (amd64). A BSF waits for that value, so one flagged SUSPECT can put
# an unrelated load on the per-field chain; see CLAUDE.md, 2026-09-22. The
# binary is cross-compiled, so the check means the same thing on an arm64 host.
bsfdep:
	GOARCH=amd64 go test -c -o logfmt.test .
	python3 bench/bsfdep.py logfmt.test 'logfmt\.Iterate$$'

fmt:
	gofmt -w .

fix: fmt
	go mod tidy
	cd bench && go mod tidy

GOLANGCI_LINT_INSTALL := go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

update-tools:
	$(GOLANGCI_LINT_INSTALL)

$(GOBIN)/golangci-lint:
	$(GOLANGCI_LINT_INSTALL)
