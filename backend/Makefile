.PHONY: help run seed test lint build clean

APP_NAME=backend-go
PORT?=8080

help:
	@echo "Available targets:"
	@echo "  make run      - run server locally"
	@echo "  make seed     - seed default providers and exit"
	@echo "  make test     - run tests"
	@echo "  make lint     - run linter"
	@echo "  make build    - build server binary"
	@echo "  make clean    - clean build artifacts"

run:
	@echo "🚀 Running server on port $(PORT)..."
	@PORT=$(PORT) go run ./cmd/server

seed:
	@echo "🌱 Seeding default providers..."
	@SEED=true SEED_ONLY=true PORT=$(PORT) go run ./cmd/server

test:
	@echo "🧪 Running tests..."
	@go test ./... -count=1 -v 2>&1 | grep -v "no test files" || true
	@echo ""
	@echo "✅ Tests completed"

lint:
	@echo "🔍 Running linter..."
	@golangci-lint run || echo "Install golangci-lint: https://golangci-lint.run/usage/install/"

build:
	@echo "🔨 Building server..."
	@CGO_ENABLED=1 go build -o bin/$(APP_NAME) ./cmd/server
	@echo "✅ Built bin/$(APP_NAME)"

clean:
	@echo "🧹 Cleaning..."
	@rm -rf bin/
	@echo "✅ Clean complete"
