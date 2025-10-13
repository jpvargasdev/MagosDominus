APP_NAME := magos-dominus
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)
GO_FILES := $(shell find . -type f -name '*.go')
IMAGE := ghcr.io/jpvargasdev/$(APP_NAME)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all dev build run clean lint fmt test

## Default target
all: build

## 🧱 Build binary
build: $(GO_FILES)
	@echo "→ Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "-X main.Version=$(VERSION)" -o $(BIN_PATH) ./cmd/magos-dominus
	@echo "✅ Built at $(BIN_PATH)"

## 🧠 Run in development mode (Docker Compose + Air)
dev:
	@echo "→ Starting $(APP_NAME) in dev mode (Compose)..."
	@docker compose up --build

## 🚀 Run local binary
run: build
	@echo "→ Running $(APP_NAME)..."
	@$(BIN_PATH) run

## 🧹 Clean build artifacts
clean:
	@echo "→ Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@docker compose down -v --remove-orphans 2>/dev/null || true
	@echo "✅ Clean"

## 🧪 Run unit tests
test:
	@go test ./... -cover -count=1

## 🔍 Lint and format code
lint:
	@go vet ./...
	@golangci-lint run || true

fmt:
	@go fmt ./...

## 🐳 Build docker image
image:
	@echo "→ Building Docker image $(IMAGE):$(VERSION)"
	@docker build -t $(IMAGE):$(VERSION) -f Dockerfile.dev .
	@echo "✅ Image built: $(IMAGE):$(VERSION)"

## 🧼 Reset repo clone for fresh reconcile
reset:
	@echo "→ Removing tmp repo clones..."
	@rm -rf /tmp/git /tmp/magos || true
	@echo "✅ Clean state"
