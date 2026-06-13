SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

BIN_DIR  := bin
API_CMD  := ./cmd/api
GAME_CMD := ./cmd/game

GO_FILES = $(shell git ls-files '*.go') $(shell git ls-files --others --exclude-standard '*.go')

ENV_RUN := scripts/env-run.sh

MIGRATE_URL = $$($(ENV_RUN) scripts/postgres-url.sh)
MIGRATE = migrate -path=migrations -database "$(MIGRATE_URL)"

TEST_EXCLUDE := /internal/protocol|/cmd/
TEST_PKGS := $(shell go list ./... | grep -Ev '$(TEST_EXCLUDE)')
COVER_PKGS_CSV := $(shell go list ./... | grep -Ev '$(TEST_EXCLUDE)' | paste -sd, -)

.PHONY: help
help:
	@printf "%s\n" \
	"Setup:" \
	"  make setup             Initialize dev/local environment" \
	"  make doctor            Check required tools" \
	"" \
	"Environment:" \
	"  make env-up            Start PostgreSQL and Redis" \
	"  make env-down          Stop services" \
	"  make env-reset         Recreate service volumes" \
	"  make env-logs          Follow service logs" \
	"" \
	"Migrations:" \
	"  make migrate-up        Apply migrations" \
	"  make migrate-down      Roll back one migration" \
	"  make migrate-reset     Reset all migrations" \
	"  make migrate-version   Show migration version" \
	"" \
	"Development:" \
	"  make run-api           Run API server" \
	"  make run-game          Run game server" \
	"  make proto             Regenerate protobuf code" \
	"  make test              Run unit and integration tests" \
	"  make ci-fast           Fast pre-commit checks" \
	"  make ci                Full CI checks"

# --- Setup ---
.PHONY: setup doctor compose-check env-init hooks tools tools-go tools-migrate

setup: env-init hooks tools
	go mod download
	$(MAKE) migrate-up
	$(MAKE) doctor
	@echo "Setup complete"

doctor:
	@go version
	@go env GOMOD
	@pkg-config --exists libenet
	@go tool buf --version >/dev/null
	@go tool golangci-lint version >/dev/null
	@go tool -n goimports >/dev/null
	@go tool govulncheck -version >/dev/null
	@migrate -version >/dev/null
	@echo "Environment OK"

compose-check:
	docker compose -f compose.yaml -f .devcontainer/compose.yaml config --quiet

env-init:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env from .env.example"; \
	else \
		echo ".env already exists"; \
	fi

hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks installed"

tools: tools-go tools-migrate

tools-go:
	go install tool

tools-migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

# --- Environment ---
.PHONY: env-up env-down env-reset env-logs db-shell redis-shell

env-up:
	docker compose up -d --wait

env-down:
	docker compose down

env-reset:
	docker compose down -v
	docker compose up -d --wait

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

# --- Migrations ---
.PHONY: migrate-up migrate-down migrate-reset migrate-version

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

migrate-reset:
	$(MIGRATE) down -all
	$(MIGRATE) up

migrate-version:
	$(MIGRATE) version

# --- Development ---
.PHONY: run-api run-game build clean \
	proto proto-check proto-lint proto-breaking \
	tidy tidy-check fmt fmt-check vet lint vuln \
	test test-unit test-integration test-race \
	coverage coverage-integration \
	ci-fast ci

run-api:
	$(ENV_RUN) go run $(API_CMD)

run-game:
	$(ENV_RUN) go run $(GAME_CMD)

build: $(BIN_DIR)
	go build -o $(BIN_DIR)/api $(API_CMD)
	go build -o $(BIN_DIR)/game $(GAME_CMD)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -rf $(BIN_DIR)

# Protobuf generation and checks
proto:
	@PROTOC_GEN_GO="$$(go tool -n protoc-gen-go)"; \
	PROTOC_GEN_GO_GRPC="$$(go tool -n protoc-gen-go-grpc)"; \
	PROTOC_GEN_CONNECT_GO="$$(go tool -n protoc-gen-connect-go)"; \
	PATH="$$(dirname "$$PROTOC_GEN_GO"):$$(dirname "$$PROTOC_GEN_GO_GRPC"):$$(dirname "$$PROTOC_GEN_CONNECT_GO"):$$PATH" \
	go tool buf generate
	$(MAKE) fmt

proto-check:
	$(MAKE) proto
	@git diff --exit-code -- proto internal/protocol

proto-lint:
	go tool buf lint

proto-breaking:
	@if ! git rev-parse --verify origin/main >/dev/null 2>&1; then \
		echo "Skipping proto breaking check: origin/main not found"; \
	elif ! git ls-tree -r --name-only origin/main -- proto | grep -q '\.proto$$'; then \
		echo "Skipping proto breaking check: origin/main has no proto files"; \
	else \
		go tool buf breaking --against '.git#branch=origin/main'; \
	fi

# Quality checks
tidy:
	go mod tidy

tidy-check:
	go mod tidy --diff

fmt:
	@if [ -n "$(GO_FILES)" ]; then \
		gofmt -w $(GO_FILES); \
		go tool goimports -w $(GO_FILES); \
	fi

fmt-check:
	@files="$$(gofmt -l $(GO_FILES))"; \
	if [ -n "$$files" ]; then \
		echo "Go files need gofmt:"; \
		echo "$$files"; \
		exit 1; \
	fi
	@files="$$(go tool goimports -l $(GO_FILES))"; \
	if [ -n "$$files" ]; then \
		echo "Go files need goimports:"; \
		echo "$$files"; \
		exit 1; \
	fi

vet:
	go vet ./...

lint:
	go tool golangci-lint run ./...

vuln:
	go tool govulncheck ./...

# Tests
test: test-unit test-integration

test-unit:
	go test -count=1 $(TEST_PKGS)

test-integration:
	$(ENV_RUN) env APP_ENV=testing go test -count=1 -tags=integration $(TEST_PKGS)

test-race:
	go test -race -count=1 $(TEST_PKGS)

coverage:
	go test -count=1 -covermode=atomic -coverpkg="$(COVER_PKGS_CSV)" -coverprofile=coverage.out $(TEST_PKGS)
	go tool cover -func=coverage.out

coverage-integration:
	$(ENV_RUN) env APP_ENV=testing go test -tags=integration -count=1 -covermode=atomic -coverpkg="$(COVER_PKGS_CSV)" -coverprofile=coverage-integration.out $(TEST_PKGS)
	go tool cover -func=coverage-integration.out

# CI targets
ci-fast: tidy-check fmt-check proto-lint vet lint test-unit

ci: tidy-check fmt-check proto-lint proto-check vet lint vuln test-race test-integration