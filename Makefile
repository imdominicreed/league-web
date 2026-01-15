.PHONY: help dev dev-backend dev-frontend run-all db db-stop db-clean build build-frontend build-all test lint clean docker-up docker-down sync-champions dc-build dc-up dc-down dc-shell dc-setup dc-logs dc-clean

# Load environment variables from .env file
include .env
export

# Default target
help:
	@echo "League Draft Website - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make dev           - Start both backend and frontend (requires tmux)"
	@echo "  make run-all       - Alias for 'make dev'"
	@echo "  make dev-backend   - Start Go backend server"
	@echo "  make dev-frontend  - Start React dev server"
	@echo "  make db            - Start PostgreSQL database"
	@echo "  make db-stop       - Stop PostgreSQL database"
	@echo "  make db-clean      - Remove database volume (fresh start)"
	@echo ""
	@echo "Dev Container (isolated env with your dotfiles):"
	@echo "  make dc-build      - Build dev container image"
	@echo "  make dc-up         - Start dev container + postgres"
	@echo "  make dc-shell      - Attach to dev container (runs setup on first use)"
	@echo "  make dc-down       - Stop dev container"
	@echo "  make dc-logs       - View container logs"
	@echo "  make dc-clean      - Remove containers and volumes"
	@echo ""
	@echo "Lobby Simulator:"
	@echo "  make dev-lobby     - Create 10-player lobby ready for draft"
	@echo "  make dev-lobby-populate LOBBY=<code> - Add players to existing lobby"
	@echo ""
	@echo "Build:"
	@echo "  make build         - Build Go backend"
	@echo "  make build-frontend- Build React frontend"
	@echo "  make build-all     - Build both backend and frontend"
	@echo "  make simulator-build - Build lobby simulator CLI"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up     - Start all services with Docker Compose"
	@echo "  make docker-down   - Stop all Docker services"
	@echo ""
	@echo "Utilities:"
	@echo "  make sync-champions- Sync champion data from Riot API"
	@echo "  make test          - Run Go tests"
	@echo "  make lint          - Run linters"
	@echo "  make clean         - Clean build artifacts"

# Development
dev-backend:
	@echo "Starting Go backend..."
	go run ./cmd/server

dev-frontend:
	@echo "Starting React dev server..."
	cd frontend && npm run dev -- --host 0.0.0.0

dev:
	@echo "Starting development servers..."
	@echo "Backend: http://localhost:$(PORT)"
	@echo "Frontend: http://localhost:3000"
	@tmux new-session -d -s league-draft 'make dev-backend' \; split-window -h 'make dev-frontend' \; attach

run-all: dev

# Database
db:
	@echo "Starting PostgreSQL..."
	docker compose up -d postgres

db-stop:
	@echo "Stopping PostgreSQL..."
	docker compose stop postgres

db-clean:
	@echo "Cleaning database..."
	docker compose down postgres -v
	docker volume rm league-draft-website_postgres_data 2>/dev/null || true
	@echo "Database cleaned. Run 'make db' to start fresh."

# Build
build:
	@echo "Building Go backend..."
	go build -o bin/server ./cmd/server

build-frontend:
	@echo "Building React frontend..."
	cd frontend && npm run build

build-all: build build-frontend
	@echo "Build complete: backend and frontend"

# Docker
docker-up:
	@echo "Starting all services..."
	docker compose up -d

docker-down:
	@echo "Stopping all services..."
	docker compose down

docker-logs:
	docker compose logs -f

# Utilities
sync-champions:
	@echo "Syncing champion data from Riot API..."
	curl -X POST http://localhost:$(PORT)/api/v1/champions/sync

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linters..."
	go vet ./...
	cd frontend && npm run lint

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf frontend/dist/
	rm -rf frontend/node_modules/

# Install dependencies
install:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

# Simulator commands
simulator-build:
	@echo "Building lobby simulator..."
	go build -o bin/simulator ./cmd/simulator

# Quick lobby population for development - creates 10-player lobby ready for draft
dev-lobby: simulator-build
	@echo "Creating 10-player lobby..."
	./bin/simulator full

# Populate existing lobby with fake users
dev-lobby-populate: simulator-build
	@echo "Usage: make dev-lobby-populate LOBBY=<code> COUNT=9"
	./bin/simulator populate --lobby=$(LOBBY) --count=$(COUNT)

# Dev container commands (isolated environment with chezmoi dotfiles)
DC_COMPOSE = docker compose -f .devcontainer/docker-compose.yml
DC_SETUP_MARKER = .devcontainer/.setup-done

dc-build:
	@echo "Building dev container..."
	$(DC_COMPOSE) build

dc-up:
	@echo "Starting dev container..."
	$(DC_COMPOSE) up -d

dc-down:
	@echo "Stopping dev container..."
	$(DC_COMPOSE) down

dc-shell: dc-up
	@if [ ! -f $(DC_SETUP_MARKER) ]; then \
		echo "First run - setting up environment..."; \
		$(DC_COMPOSE) exec dev /workspace/.devcontainer/post-create.sh; \
		touch $(DC_SETUP_MARKER); \
	fi
	@echo "Attaching to dev container..."
	$(DC_COMPOSE) exec dev bash

dc-setup:
	@echo "Running setup in dev container..."
	$(DC_COMPOSE) exec dev /workspace/.devcontainer/post-create.sh
	@touch $(DC_SETUP_MARKER)

dc-logs:
	$(DC_COMPOSE) logs -f

dc-clean:
	@echo "Removing dev container and volumes..."
	$(DC_COMPOSE) down -v
	@rm -f $(DC_SETUP_MARKER)
	@echo "Cleaned dev container resources"
