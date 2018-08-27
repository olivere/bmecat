default: build

.PHONY: build

build:
	go build ./cmd/bmecat

test:
	go test -race -tags integration ./...
