.PHONY: run worker build test test-unit test-integration test-contract test-architecture test-failure \
	lint fmt vet migrate-up migrate-down migrate-version migrate-force docker-up docker-down generate proto seed

# `go build` tries to embed VCS info by walking up to the nearest .git
# directory; this repository can be nested under an unrelated/incomplete
# repo on some machines, which makes VCS stamping fail with "exit status
# 128". -buildvcs=false sidesteps that unconditionally.
GOFLAGS := -buildvcs=false
GO := go

run: ## Run the API server (needs DATABASE_* pointing at a reachable Postgres)
	$(GO) run $(GOFLAGS) ./cmd/api

worker: ## Run the background worker (outbox + reconciliation)
	$(GO) run $(GOFLAGS) ./cmd/worker

build: ## Build all three binaries into ./bin
	mkdir -p bin
	$(GO) build $(GOFLAGS) -o bin/api ./cmd/api
	$(GO) build $(GOFLAGS) -o bin/worker ./cmd/worker
	$(GO) build $(GOFLAGS) -o bin/migrate ./cmd/migrate

test: test-unit test-architecture test-contract test-integration test-failure ## Run every test suite

test-unit: ## Unit tests (no external dependencies)
	$(GO) test $(GOFLAGS) ./tests/unit/... ./internal/...

test-integration: ## Integration tests (requires Docker for Testcontainers)
	$(GO) test $(GOFLAGS) -tags=integration ./tests/integration/...

test-contract: ## Contract tests (no network required)
	$(GO) test $(GOFLAGS) ./tests/contract/...

test-architecture: ## Import-graph / layering rules
	$(GO) test $(GOFLAGS) ./tests/architecture/...

test-failure: ## Failure-injection scenarios
	$(GO) test $(GOFLAGS) -tags=integration ./tests/failure/...

lint: ## go vet (no golangci-lint dependency required)
	$(GO) vet $(GOFLAGS) ./...

fmt: ## gofmt every file
	gofmt -l -w .

vet: lint

migrate-up: ## Apply all pending migrations
	$(GO) run $(GOFLAGS) ./cmd/migrate up

migrate-down: ## Revert the most recent migration (or N: make migrate-down N=2)
	$(GO) run $(GOFLAGS) ./cmd/migrate down $(N)

migrate-version: ## Print the current schema version
	$(GO) run $(GOFLAGS) ./cmd/migrate version

migrate-force: ## Force the tracked schema version without running SQL: make migrate-force V=3
	$(GO) run $(GOFLAGS) ./cmd/migrate force $(V)

seed: ## Apply the optional manual paper-account seed
	@echo "psql \$$DATABASE_URL -f internal/infrastructure/database/seeds/paper_account.sql"

docker-up: ## Start the full stack (postgres, migrate, api, worker)
	docker compose up --build

docker-down: ## Stop and remove the stack
	docker compose down -v

generate: proto ## Alias for `make proto`

proto: ## Regenerate gRPC/protobuf code for the Quant Engine contract
	@command -v protoc >/dev/null 2>&1 || { echo "protoc is not installed; see internal/contracts/grpc/quant/generated/doc.go"; exit 1; }
	protoc \
		--go_out=. --go_opt=module=trading-core \
		--go-grpc_out=. --go-grpc_opt=module=trading-core \
		internal/contracts/grpc/quant/quant_engine.proto
