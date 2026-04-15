.PHONY: help install dev test test-watch test-ui build build-analyze lint lint-fix format type-check preview clean clean-all

PORT ?= 5173
HOST ?= 0.0.0.0

help:
	@echo ""
	@echo "📦 Frontend Development Commands"
	@echo ""
	@echo "Setup:"
	@echo "  make install          Install dependencies with pnpm"
	@echo "  make clean            Remove dist folder"
	@echo "  make clean-all        Remove node_modules, dist and lockfile"
	@echo ""
	@echo "Development:"
	@echo "  make dev              Start dev server"
	@echo "                        Override with PORT=$(PORT) HOST=$(HOST)"
	@echo "  make preview          Preview production build"
	@echo ""
	@echo "Testing:"
	@echo "  make test             Run tests once"
	@echo "  make test-watch       Run tests in watch mode"
	@echo "  make test-ui          Run tests with UI"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint             Check code with ESLint"
	@echo "  make lint-fix         Fix ESLint issues"
	@echo "  make format           Format code with Prettier"
	@echo "  make type-check       Run TypeScript type checking"
	@echo ""
	@echo "Build:"
	@echo "  make build            Build for production"
	@echo "  make build-analyze    Build with bundle analyzer"
	@echo ""

install:
	@echo "📦 Installing dependencies..."
	pnpm install

dev:
	@echo "🚀 Starting dev server on http://$(HOST):$(PORT)..."
	pnpm dev -- --host $(HOST) --port $(PORT)

test:
	@echo "🧪 Running tests..."
	pnpm test -- --run

test-watch:
	@echo "🧪 Running tests in watch mode..."
	pnpm test

test-ui:
	@echo "🧪 Running tests with UI..."
	pnpm test -- --ui

build:
	@echo "🏗️  Building for production..."
	pnpm build

build-analyze:
	@echo "🏗️  Building with bundle analyzer..."
	pnpm build
	@echo "📊 Opening bundle analyzer..."
	@open dist/stats.html || xdg-open dist/stats.html || start dist/stats.html

lint:
	@echo "🔍 Linting code..."
	pnpm lint

lint-fix:
	@echo "🔧 Fixing lint issues..."
	pnpm lint -- --fix

format:
	@echo "✨ Formatting code..."
	@npx prettier --write "src/**/*.{ts,tsx,js,jsx,json,css,md}"

type-check:
	@echo "🔎 Type checking..."
	@npx tsc --noEmit

preview:
	@echo "👀 Previewing production build..."
	pnpm preview

clean:
	@echo "🧹 Cleaning dist folder..."
	rm -rf dist

clean-all:
	@echo "🧹 Cleaning node_modules, dist and pnpm lockfile..."
	rm -rf node_modules dist pnpm-lock.yaml


