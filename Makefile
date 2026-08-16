.PHONY: test test-short lint fmt build clean run help

# Default target runs linter, tests, and builds
all: lint test build

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/## //'

## test: Run all unit and integration tests (requires Docker for Neo4j)
test:
	go test -v ./... -cover

## test-short: Run only unit tests (skips containerized integration tests)
test-short:
	go test -short -v ./... -cover

## lint: Run golangci-lint static analysis checks
lint:
	golangci-lint run

## fmt: Format Go code according to standard style guidelines
fmt:
	gofmt -w .
	go fmt ./...

## build: Compile the Echo BFF HTTP API server binary
build:
	@mkdir -p bin
	go build -o bin/api ./cmd/api/main.go

## clean: Remove all compiled binary assets
clean:
	rm -rf bin

## run: Build and launch the Echo HTTP API server locally
run: build
	./bin/api
