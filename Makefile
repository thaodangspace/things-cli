BINARY ?= things-cli
MAIN ?= ./cmd/things-cli
PKGS ?= ./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -s -w -X github.com/thaodangspace/things-cli/cli.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help build install test test-race fmt vet tidy check clean snapshot

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the things-cli binary
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN)

install: ## Install the CLI with go install
	go install -trimpath -ldflags "$(LDFLAGS)" $(MAIN)

test: ## Run tests
	go test $(PKGS)

test-race: ## Run tests with the race detector
	go test -race $(PKGS)

fmt: ## Format Go code
	go fmt $(PKGS)

vet: ## Run go vet
	go vet $(PKGS)

tidy: ## Tidy Go modules
	go mod tidy

check: fmt tidy vet test ## Format, tidy, vet, and test

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist

snapshot: ## Build a local GoReleaser snapshot
	goreleaser release --snapshot --clean
