# Tools (gofumpt, govulncheck, staticcheck) are managed as Go tool dependencies
# in go.mod and invoked via `go tool`. See the `tool` directive in go.mod.
# modernize lives inside gopls (an internal package), so it is run ad-hoc via
# `go run` to avoid pulling all of gopls into the module graph.
MODERNIZE = golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest

# Benchmark knobs (override on the command line, e.g. `make bench BENCH=Write`).
# BENCHCOUNT samples each benchmark so benchstat has something to compare.
BENCH ?= .
BENCHCOUNT ?= 6
BENCHFILE ?= bench.txt

.PHONY: default
default: build

.PHONY: build
build:
	go build ./cmd/bmecat

.PHONY: test
test:
	go test -race -tags integration ./...

.PHONY: bench
bench:
	go test -run '^$$' -bench '$(BENCH)' -benchmem -count=$(BENCHCOUNT) ./...

# Save benchmark results to BENCHFILE (default bench.txt) for benchstat.
.PHONY: bench-save
bench-save:
	go test -run '^$$' -bench '$(BENCH)' -benchmem -count=$(BENCHCOUNT) ./... | tee $(BENCHFILE)

# Compare two saved runs: make benchstat OLD=base.txt NEW=bench.txt
.PHONY: benchstat
benchstat:
	go tool benchstat $(OLD) $(NEW)

.PHONY: fmt
fmt:
	go tool gofumpt -w .

.PHONY: lint
lint:
	go tool gofumpt -l .
	go vet ./...
	go tool staticcheck ./...

.PHONY: vulncheck
vulncheck:
	go tool govulncheck ./...

.PHONY: modernize
modernize:
	go run $(MODERNIZE) -fix ./...

# Run the full set of checks, as CI does.
.PHONY: check
check: lint test vulncheck
