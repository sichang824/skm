SHELL := /bin/sh

BACKEND_DIR := backend
FRONTEND_DIR := frontend
DB_FILE := $(BACKEND_DIR)/data/app.db

BACKEND_PORT ?= 8080
FRONTEND_PORT ?= 5173
FRONTEND_HOST ?= 0.0.0.0

BACKEND_URL := http://localhost:$(BACKEND_PORT)
FRONTEND_URL := http://localhost:$(FRONTEND_PORT)

.DEFAULT_GOAL := help

.PHONY: help check-tools install reset seed dev dev/seed dev-backend dev-frontend test build clean app-dev app-build

help: ## Show available workspace commands
	@echo "SKM workspace commands"
	@echo ""
	@grep -E '^[a-zA-Z0-9_/-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'
	@echo ""
	@echo "Variable overrides:"
	@echo "  BACKEND_PORT=$(BACKEND_PORT) FRONTEND_PORT=$(FRONTEND_PORT) FRONTEND_HOST=$(FRONTEND_HOST)"

check-tools: ## Check required local tools
	@command -v go >/dev/null 2>&1 || { echo "Error: go is required"; exit 1; }
	@command -v pnpm >/dev/null 2>&1 || { echo "Error: pnpm is required"; exit 1; }

install: check-tools ## Install frontend deps and preload backend modules
	@$(MAKE) -C $(FRONTEND_DIR) install
	@cd $(BACKEND_DIR) && go mod download
	@go mod download

reset: ## Remove the local SQLite database files
	@echo "Resetting local database..."
	@rm -f $(DB_FILE) $(DB_FILE)-shm $(DB_FILE)-wal
	@echo "✅ Database reset complete"

seed: check-tools ## Seed default providers into the local backend database and exit
	@$(MAKE) -C $(BACKEND_DIR) seed PORT=$(BACKEND_PORT)

dev: check-tools ## Start backend and frontend together
	@echo "Starting SKM dev stack"
	@echo "  Backend : $(BACKEND_URL)"
	@echo "  Frontend: $(FRONTEND_URL)"
	@backend_pid=""; frontend_pid=""; \
	cleanup() { \
		status=$$?; \
		echo ""; \
		echo "Stopping SKM dev stack..."; \
		if [ -n "$$frontend_pid" ]; then kill $$frontend_pid >/dev/null 2>&1 || true; fi; \
		if [ -n "$$backend_pid" ]; then kill $$backend_pid >/dev/null 2>&1 || true; fi; \
		wait $$frontend_pid $$backend_pid >/dev/null 2>&1 || true; \
		exit $$status; \
	}; \
	trap cleanup INT TERM EXIT; \
	$(MAKE) -C $(BACKEND_DIR) run PORT=$(BACKEND_PORT) & backend_pid=$$!; \
	VITE_PROXY_TARGET=$(BACKEND_URL) $(MAKE) -C $(FRONTEND_DIR) dev PORT=$(FRONTEND_PORT) HOST=$(FRONTEND_HOST) & frontend_pid=$$!; \
	wait $$backend_pid $$frontend_pid

dev/seed: reset check-tools ## Reset the local DB, then start backend and frontend with provider seeding enabled
	@echo "Starting SKM dev stack with provider seeding"
	@echo "  Backend : $(BACKEND_URL)"
	@echo "  Frontend: $(FRONTEND_URL)"
	@backend_pid=""; frontend_pid=""; \
	cleanup() { \
		status=$$?; \
		echo ""; \
		echo "Stopping SKM dev stack..."; \
		if [ -n "$$frontend_pid" ]; then kill $$frontend_pid >/dev/null 2>&1 || true; fi; \
		if [ -n "$$backend_pid" ]; then kill $$backend_pid >/dev/null 2>&1 || true; fi; \
		wait $$frontend_pid $$backend_pid >/dev/null 2>&1 || true; \
		exit $$status; \
	}; \
	trap cleanup INT TERM EXIT; \
	SEED=true $(MAKE) -C $(BACKEND_DIR) run PORT=$(BACKEND_PORT) & backend_pid=$$!; \
	VITE_PROXY_TARGET=$(BACKEND_URL) $(MAKE) -C $(FRONTEND_DIR) dev PORT=$(FRONTEND_PORT) HOST=$(FRONTEND_HOST) & frontend_pid=$$!; \
	wait $$backend_pid $$frontend_pid

dev-backend: check-tools ## Start only the backend service
	@$(MAKE) -C $(BACKEND_DIR) run PORT=$(BACKEND_PORT)

dev-frontend: check-tools ## Start only the frontend dev server
	@VITE_PROXY_TARGET=$(BACKEND_URL) $(MAKE) -C $(FRONTEND_DIR) dev PORT=$(FRONTEND_PORT) HOST=$(FRONTEND_HOST)

test: check-tools ## Run backend and frontend tests
	@$(MAKE) -C $(BACKEND_DIR) test
	@$(MAKE) -C $(FRONTEND_DIR) test

build: check-tools ## Build backend binary and frontend assets
	@$(MAKE) -C $(BACKEND_DIR) build
	@$(MAKE) -C $(FRONTEND_DIR) build

app-dev: check-tools ## Start the Wails desktop app in development mode
	@go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 dev

app-build: check-tools ## Build the macOS desktop app bundle with Wails
	@go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -platform darwin/universal

clean: ## Clean backend and frontend build artifacts
	@$(MAKE) -C $(BACKEND_DIR) clean
	@$(MAKE) -C $(FRONTEND_DIR) clean
