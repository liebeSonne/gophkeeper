VERSION := v1.0.0
COMMIT := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date +'%Y/%m/%d %H:%M:%S')
LDFLAGS := -X main.buildVersion=$(VERSION) -X 'main.buildDate=$(BUILD_TIME)' -X main.buildCommit=$(COMMIT)

PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64
DOCKER_NETWORK := gophkeeper-net

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
	@grep -v -E "\.pb\.go|mock_.*\.gen\.go|_gen\.go|client\.gen\.go|openapi.*gen\.go" coverage.out > coverage.clean.out
	@go tool cover -func=coverage.clean.out | grep total

.PHONY: cover-check
cover-check: ## Check test coverage >= 70% (excluding generated/mocks/cmd/tui/repo-db)
	@CGO_ENABLED=0 go test -coverprofile=/tmp/coverage_all.out ./... 2>&1 | grep -v "no test files" || true
	@grep -v -E "\.pb\.go|mock_.*\.gen\.go|_gen\.go|client\.gen\.go|openapi.*gen\.go" \
		/tmp/coverage_all.out > /tmp/coverage_clean.out || true
	@grep -v -E "cmd/|internal/client/cmd|internal/client/tui|internal/server/repository/db" \
		/tmp/coverage_clean.out > /tmp/coverage_filtered.out || true
	@COV=$$(go tool cover -func=/tmp/coverage_filtered.out | grep total | \
		awk '{print $$3}' | tr -d '%' | cut -d. -f1); \
	echo "Coverage: $$COV%%"; \
	if [ "$$COV" -lt 70 ]; then \
		echo "ERROR: Coverage $$COV%% < 70%%"; exit 1; \
	fi

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

# Docker targets

.PHONY: docker-up
docker-up: ## Start dependencies (PostgreSQL, Vault, MinIO)
	@docker network create $(DOCKER_NETWORK) 2>/dev/null || true
	@docker compose -f deployments/docker-compose.yml up -d
	@echo "Waiting for services to be ready..."
	@docker compose -f deployments/docker-compose.yml wait || true
	@echo "Services started"

.PHONY: docker-down
docker-down: ## Stop all services
	@docker stop gophkeeper-server 2>/dev/null || true
	@docker compose -f deployments/docker-compose.yml down

.PHONY: docker-logs
docker-logs: ## Show logs
	@docker compose -f deployments/docker-compose.yml logs -f

.PHONY: docker-clean
docker-clean: ## Stop and remove volumes
	@docker stop gophkeeper-server 2>/dev/null || true
	@docker rm gophkeeper-server 2>/dev/null || true
	@docker compose -f deployments/docker-compose.yml down -v
	@docker network rm $(DOCKER_NETWORK) 2>/dev/null || true

.PHONY: docker-server-build
docker-server-build: ## Build server Docker image
	@docker build -f Dockerfile.server -t gophkeeper-server:latest .

.PHONY: docker-server-run
docker-server-run: docker-up docker-server-build ## Run server with dependencies
	@docker stop gophkeeper-server 2>/dev/null || true
	@docker rm gophkeeper-server 2>/dev/null || true
	@docker run -d --name gophkeeper-server \
		--network $(DOCKER_NETWORK) \
		-p 8080:8080 \
		-e LOG_LEVEL=debug \
		-e SERVER_ADDRESS=0.0.0.0:8080 \
		-e ENABLE_HTTPS=false \
		-e TLS_CERT= \
		-e TLS_KEY= \
		-e DATABASE_URI=postgres://gophkeeper:gophkeeper@postgres:5432/gophkeeper \
		-e JWT_SECRET=dev-jwt-secret-change-in-production \
		-e JWT_ACCESS_TTL=15m \
		-e JWT_REFRESH_TTL=24h \
		-e ENABLE_VAULT=false \
		-e VAULT_ADDRESS=http://vault:8200 \
		-e VAULT_TOKEN=dev-token \
		-e VAULT_KEY_PATH=secret/data/gophkeeper/encryption \
		-e ENCRYPTION_KEY=V+EqNpqMktOtjoRiz0/6YEdzRJGw3gU+46shFQwz3zw= \
		-e STORAGE_ENDPOINT=minio:9000 \
		-e STORAGE_ACCESS_KEY=minioadmin \
		-e STORAGE_SECRET_KEY=minioadmin \
		-e STORAGE_BUCKET=gophkeeper \
		-e STORAGE_SECURE=false \
		gophkeeper-server:latest
