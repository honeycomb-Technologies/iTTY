VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
BINARY := itty

.PHONY: build test lint clean install run

## build: Compile the daemon binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/itty

## run: Build and run the daemon
run: build
	./bin/$(BINARY)

## test: Run all tests
test:
	go test -race -count=1 ./...

## lint: Run linter
lint:
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -rf bin/

## install: Install the daemon binary to $GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/itty

## cross: Build for Linux and macOS
cross:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/itty
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 ./cmd/itty
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-amd64 ./cmd/itty
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-darwin-arm64 ./cmd/itty

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
