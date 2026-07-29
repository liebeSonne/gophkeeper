VERSION := v1.0.0
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date +'%Y/%m/%d %H:%M:%S')
LDFLAGS := -X main.buildVersion=$(VERSION) -X 'main.buildDate=$(BUILD_TIME)' -X main.buildCommit=$(COMMIT)

.DEFAULT_GOAL := all

.PHONY: all
all: build-server build-client build clean test lint

.PHONY: help
help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }'

.PHONY: build-server
build-server: ## Build server binary
	@go build -ldflags="$(LDFLAGS)" -o bin/gophkeeper-server ./cmd/server

.PHONY: build-client
build-client: ## Build client binary
	@go build -ldflags="$(LDFLAGS)" -o bin/gophkeeper-client ./cmd/client

.PHONY: build
build: build-server build-client

.PHONY: clean
clean: ## Remove binaries
	@rm -rf bin/

.PHONY: test
test: ## Run tests
	@go test -v -cover ./...

.PHONY: lint
lint: ## Run linter
	@golangci-lint run ./...
