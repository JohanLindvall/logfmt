.PHONY: all check test test-bench lint bench bench-md fix update-tools

GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
PATH := $(GOBIN):$(PATH)
export PATH

all: fix check

# Lint + the full test suite (root module and the separate bench module).
check: lint test

# Unit tests for both modules. The root module is dependency-free; the bench
# module carries the comparison-only dependencies.
test:
	go test -cover ./...
	cd bench && go test ./...

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

fix:
	gofmt -w .
	go mod tidy
	cd bench && go mod tidy

update-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

$(GOBIN)/golangci-lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
