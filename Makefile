APP_NAME ?= rpg-backend
BIN_DIR ?= bin
API_CMD ?= ./cmd/api
GAME_CMD ?= ./cmd/game
MIGRATOR_CMD ?= ./cmd/migrator

PROTO_SRC ?= proto/messages.proto
PROTO_OUT ?= internal/protocol
PROTOC ?= protoc

.PHONY: help env-up env-down env-logs run-api run-game build proto test tidy fmt vet clean

help:
	@echo "Available commands:"
	@echo "  make env-up      Start local development environment (PostgreSQL, Redis)"
	@echo "  make env-down    Stop local development environment"
	@echo "  make env-logs    View logs of local development environment"
	@echo "  make run-api     Run API server"
	@echo "  make run-game    Run game server"
	@echo "  make build       Build binaries for API and game server"
	@echo "  make proto       Regenerate protobuf Go code"
	@echo "  make test        Run tests"
	@echo "  make tidy        Clean up go.mod/go.sum"
	@echo "  make fmt         Format Go code"
	@echo "  make vet         Run Go vet for static analysis"
	@echo "  make clean       Remove built binaries"

env-up:
	docker compose up -d

env-down:
	docker compose down

env-logs:
	docker compose logs -f

run-api:
	go run $(API_CMD)

run-game:
	go run $(GAME_CMD)

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/api $(API_CMD)
	go build -o $(BIN_DIR)/game $(GAME_CMD)

proto:
	$(PROTOC) --go_out=$(PROTO_OUT) --go_opt=paths=source_relative $(PROTO_SRC)

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR)


