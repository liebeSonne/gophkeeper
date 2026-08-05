VERSION := v1.0.0
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date +'%Y/%m/%d %H:%M:%S')
LDFLAGS := -X main.buildVersion=$(VERSION) -X 'main.buildDate=$(BUILD_TIME)' -X main.buildCommit=$(COMMIT)

PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64

.DEFAULT_GOAL := all

.PHONY: all
all: clean build test lint

.PHONY: help
help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }'

.PHONY: build-server
build-server: ## Build server binary
	@CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/gophkeeper-server ./cmd/server

.PHONY: build-client
build-client: ## Build client binary
	@CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/gophkeeper-client ./cmd/client

.PHONY: build
build: build-server build-client

.PHONY: build-server-%
build-server-%: ## Build server for specific platform (e.g., linux-amd64)
	@GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) CGO_ENABLED=0 \
		go build -ldflags="$(LDFLAGS)" -o bin/gophkeeper-server-$* ./cmd/server

.PHONY: build-client-%
build-client-%: ## Build client for specific platform (e.g., linux-amd64)
	@GOOS=$(word 1,$(subst -, ,$*)) GOARCH=$(word 2,$(subst -, ,$*)) CGO_ENABLED=0 \
		go build -ldflags="$(LDFLAGS)" -o bin/gophkeeper-client-$* ./cmd/client

.PHONY: build-all
build-all: ## Build all platforms
	@mkdir -p bin
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $${platform} | cut -d'-' -f1) GOARCH=$$(echo $${platform} | cut -d'-' -f2) CGO_ENABLED=0 \
			go build -ldflags="$(LDFLAGS)" -o bin/gophkeeper-server-$${platform} ./cmd/server && \
		GOOS=$$(echo $${platform} | cut -d'-' -f1) GOARCH=$$(echo $${platform} | cut -d'-' -f2) CGO_ENABLED=0 \
			go build -ldflags="$(LDFLAGS)" -o bin/gophkeeper-client-$${platform} ./cmd/client; \
	done

.PHONY: clean
clean: ## Remove binaries
	@rm -rf bin/

.PHONY: test
test: ## Run tests
	@go test -v -cover ./...

.PHONY: cover
cover: ## Run tests coverage
	@go test -coverprofile=coverage.out ./...
	@grep -v -E "\.pb\.go|mock\.go|_gen\.go" coverage.out > coverage.clean.out
	@go tool cover -func=coverage.clean.out | grep total

.PHONY: lint
lint: ## Run linter
	@golangci-lint run ./...

.PHONY: create-migration
create-migration: ## Run create migration
	@migrate create -ext sql -dir ./migrations -format "20060102150405" $(name)

.PHONY: generate-api-server
generate-api-server: ## Run generate openapi server
	@go tool oapi-codegen -config ./api/swagger/config.yaml ./api/swagger/openapi.yml

.PHONY: generate-client
generate-client: ## Run generate openapi client
	@go tool oapi-codegen -config ./api/swagger/client_config.yaml ./api/swagger/openapi.yml

.PHONY: generate
generate: generate-api-server generate-client ## Generate all API code

.PHONY: mocks
mocks: ## Run generate mocks
	@mockery
