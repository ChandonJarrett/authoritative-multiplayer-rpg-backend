APP_NAME := rpg-backend
BIN_DIR  := bin

API_CMD       := ./cmd/api
GAME_CMD      := ./cmd/game
MIGRATOR_CMD  := ./cmd/migrator

PROTO_SRC := proto/messages.proto
PROTO_OUT := internal/protocol
PROTOC    := protoc

# Use POSIX shell explicitly
SHELL := /bin/sh

.PHONY: help setup hooks env-up env-down env-logs run-api run-game build proto test tidy fmt vet clean

help:
	@printf "%s\n" \
	"Available commands:" \
	"  make setup       Initialize local development environment" \
	"  make hooks       Install Git hooks" \
	"  make env-up      Start local development environment (PostgreSQL, Redis)" \
	"  make env-down    Stop local development environment" \
	"  make env-logs    View logs of local development environment" \
	"  make run-api     Run API server" \
	"  make run-game    Run game server" \
	"  make build       Build binaries for API and game server" \
	"  make proto       Regenerate protobuf Go code" \
	"  make test        Run tests" \
	"  make tidy        Clean up go.mod/go.sum" \
	"  make fmt         Format Go code" \
	"  make vet         Run Go vet for static analysis" \
	"  make clean       Remove built binaries"

# --- Setup ---

setup: hooks .env
	go mod download
	@echo "Setup complete"

.env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env from .env.example"; \
	else \
		echo ".env already exists"; \
	fi

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit
	@echo "Git hooks installed"

# --- Environment (Docker) ---

env-up:
	docker compose up -d

env-down:
	docker compose down

env-logs:
	docker compose logs -f

# --- Run ---

run-api:
	go run $(API_CMD)

run-game:
	go run $(GAME_CMD)

# --- Build ---

build: $(BIN_DIR)/api $(BIN_DIR)/game

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(BIN_DIR)/api: | $(BIN_DIR)
	go build -o $@ $(API_CMD)

$(BIN_DIR)/game: | $(BIN_DIR)
	go build -o $@ $(GAME_CMD)

# --- Protobuf ---

proto:
	$(PROTOC) \
		--go_out=$(PROTO_OUT) \
		--go_opt=paths=source_relative \
		$(PROTO_SRC)

# --- Quality ---

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

# --- Cleanup ---

clean:
	rm -rf $(BIN_DIR)