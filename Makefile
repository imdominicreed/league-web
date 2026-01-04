.PHONY: help dev dev-backend dev-frontend db db-stop build test lint clean docker-up docker-down sync-champions

# Default target
help:
	@echo "League Draft Website - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make dev           - Start both backend and frontend (requires tmux)"
	@echo "  make dev-backend   - Start Go backend server"
	@echo "  make dev-frontend  - Start React dev server"
	@echo "  make db            - Start PostgreSQL database"
	@echo "  make db-stop       - Stop PostgreSQL database"
	@echo ""
	@echo "Build:"
	@echo "  make build         - Build Go backend"
	@echo "  make build-frontend- Build React frontend"
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
	cd frontend && npm run dev

dev:
	@echo "Starting development servers..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:3000"
	@tmux new-session -d -s league-draft 'make dev-backend' \; split-window -h 'make dev-frontend' \; attach

# Database
db:
	@echo "Starting PostgreSQL..."
	docker compose up -d postgres

db-stop:
	@echo "Stopping PostgreSQL..."
	docker compose stop postgres

# Build
build:
	@echo "Building Go backend..."
	go build -o bin/server ./cmd/server

build-frontend:
	@echo "Building React frontend..."
	cd frontend && npm run build

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
	curl -X POST http://localhost:8080/api/v1/champions/sync

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
