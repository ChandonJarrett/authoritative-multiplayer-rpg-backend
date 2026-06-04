APP_NAME := rpg-backend
BIN_DIR  := bin

API_CMD  := ./cmd/api
GAME_CMD := ./cmd/game

PROTO_SRC := proto/messages.proto
PROTO_OUT := internal/protocol
PROTOC    := protoc

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

GO_FILES := $(shell git ls-files '*.go')

.PHONY: help
help:
	@printf "%s\n" \
	"Available commands:" \
	"" \
	"Setup:" \
	"  make setup           Initialize local development environment" \
	"  make doctor          Check required local/devcontainer tools" \
	"  make hooks           Install repository Git hooks" \
	"  make tools           Install Go tools declared in go.mod tool block" \
	"" \
	"Environment:" \
	"  make env-up          Start PostgreSQL and Redis, then wait for health" \
	"  make env-down        Stop local environment" \
	"  make env-reset       Destroy and recreate local environment volumes" \
	"  make env-ps          List local environment containers" \
	"  make env-health      Check PostgreSQL and Redis health" \
	"  make env-logs        Follow local environment logs" \
	"  make db-shell        Open psql shell" \
	"  make redis-shell     Open redis-cli shell" \
	"" \
	"Run/build:" \
	"  make run-api         Run API server" \
	"  make run-game        Run game server" \
	"  make build           Build API and game server binaries" \
	"  make clean           Remove built binaries" \
	"" \
	"Code generation:" \
	"  make proto           Regenerate protobuf Go code" \
	"  make proto-check     Verify protobuf generated code is current" \
	"" \
	"Quality:" \
	"  make check           Run local quality checks" \
	"  make ci-check        Run full CI checks" \
	"  make precommit       Run fast pre-commit checks" \
	"  make tidy            Run go mod tidy" \
	"  make tidy-check      Verify go.mod/go.sum are tidy" \
	"  make fmt             Format Go code" \
	"  make fmt-check       Verify Go formatting/imports" \
	"  make vet             Run go vet" \
	"  make lint            Run golangci-lint" \
	"  make vuln            Run govulncheck" \
	"  make test            Run tests" \
	"  make test-race       Run tests with race detector"

# --- Setup ---
.PHONY: setup doctor hooks tools .env

setup: hooks tools .env
	go mod download
	@echo "Setup complete"

doctor:
	@echo "Checking development environment..."
	@go version
	@go env GOMOD
	@$(PROTOC) --version
	@pkg-config --exists libenet
	@echo "libenet found"
	@go tool golangci-lint version >/dev/null
	@go tool -n goimports >/dev/null
	@go tool govulncheck -version >/dev/null
	@go mod download
	@if command -v docker >/dev/null 2>&1; then \
		docker version >/dev/null; \
		docker compose version >/dev/null; \
		docker compose config --quiet; \
		echo "Docker and Compose found"; \
	else \
		echo "Docker CLI not found in this shell; skipping Docker checks"; \
	fi
	@echo "Development environment looks usable."

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit
	@echo "Git hooks installed from .githooks"

tools:
	# Tools declared in go.mod tool block.
	go install tool

.env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env from .env.example"; \
	else \
		echo ".env already exists"; \
	fi

# --- Environment ---
.PHONY: env-up env-down env-reset env-ps env-health env-logs db-shell redis-shell

env-up:
	docker compose up -d
	@echo "Waiting for PostgreSQL..."
	@until docker compose exec -T postgres pg_isready -U "$${POSTGRES_USER:-postgres}" -d "$${POSTGRES_DB:-rpg}" >/dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Waiting for Redis..."
	@until docker compose exec -T redis redis-cli ping | grep -q PONG; do \
		sleep 1; \
	done
	@echo "Environment is ready."

env-down:
	docker compose down

env-reset:
	docker compose down -v
	$(MAKE) env-up

env-ps:
	docker compose ps

env-health:
	docker compose ps
	docker compose exec -T postgres pg_isready -U "$${POSTGRES_USER:-postgres}" -d "$${POSTGRES_DB:-rpg}"
	docker compose exec -T redis redis-cli ping

env-logs:
	docker compose logs -f

db-shell:
	docker compose exec postgres psql -U "$${POSTGRES_USER:-postgres}" -d "$${POSTGRES_DB:-rpg}"

redis-shell:
	@if [ -n "$${REDIS_PASSWORD:-}" ]; then \
		docker compose exec redis redis-cli -a "$${REDIS_PASSWORD}"; \
	else \
		docker compose exec redis redis-cli; \
	fi

# --- Run ---
.PHONY: run-api run-game

run-api:
	go run $(API_CMD)

run-game:
	go run $(GAME_CMD)

# --- Build ---
.PHONY: build clean

build: $(BIN_DIR)/api $(BIN_DIR)/game

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(BIN_DIR)/api: | $(BIN_DIR)
	go build -o $@ $(API_CMD)

$(BIN_DIR)/game: | $(BIN_DIR)
	go build -o $@ $(GAME_CMD)

clean:
	rm -rf $(BIN_DIR)

# --- Protobuf ---
.PHONY: proto proto-check

proto:
	@PROTOC_GEN_GO="$$(go tool -n protoc-gen-go)"; \
	PROTOC_GEN_GO_GRPC="$$(go tool -n protoc-gen-go-grpc)"; \
	PATH="$$(dirname "$$PROTOC_GEN_GO"):$$(dirname "$$PROTOC_GEN_GO_GRPC"):$$PATH" \
	$(PROTOC) \
		--go_out=$(PROTO_OUT) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_SRC)

proto-check:
	$(MAKE) proto
	git diff --exit-code -- $(PROTO_SRC) $(PROTO_OUT)

# --- Quality ---
.PHONY: check ci-check precommit tidy tidy-check fmt fmt-check vet lint vuln test test-race 

check: tidy-check fmt-check vet lint test

ci-check: tidy-check fmt-check proto-check vet lint vuln test-race

precommit: tidy-check fmt-check vet lint test

tidy:
	go mod tidy

tidy-check:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

fmt:
	@if [ -n "$(GO_FILES)" ]; then \
		gofmt -w $(GO_FILES); \
		go tool goimports -w $(GO_FILES); \
	fi

fmt-check:
	@echo "Running gofmt and goimports checks..."
	@files="$$(gofmt -l $(GO_FILES))"; \
	if [ -n "$$files" ]; then \
		echo "Go files need gofmt:"; \
		echo "$$files"; \
		echo "Run: make fmt"; \
		exit 1; \
	fi
	@files="$$(go tool goimports -l $(GO_FILES))"; \
	if [ -n "$$files" ]; then \
		echo "Go files need goimports:"; \
		echo "$$files"; \
		echo "Run: make fmt"; \
		exit 1; \
	fi

vet:
	go vet ./...

lint:
	go tool golangci-lint run ./...

vuln:
	go tool govulncheck ./...

test:
	go test ./...

test-race:
	go test -race ./...