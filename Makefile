# Tools (gofumpt, govulncheck, staticcheck) are managed as Go tool dependencies
# in go.mod and invoked via `go tool`. See the `tool` directive in go.mod.
# modernize lives inside gopls (an internal package), so it is run ad-hoc via
# `go run` to avoid pulling all of gopls into the module graph.
MODERNIZE = golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest

.PHONY: default
default: build

.PHONY: build
build:
	go build ./cmd/bmecat

.PHONY: test
test:
	go test -race -tags integration ./...

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
