# Makefile

.PHONY: all run build test vet lint clean up down migrate-create migrate-up migrate-down push push-fast push-new git-setup help

# Variables
BINARY_NAME=payflow
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/server/main.go
DOCKER_COMPOSE=docker-compose
DSN ?= postgres://payflow_user:payflow_secret@localhost:5433/payflow_db?sslmode=disable

# Default target
all: vet test build

# --- Application ---

run:
	@echo "Running application..."
	@go run $(MAIN_PATH)

build:
	@echo "Building binary..."
	@go build -o $(BINARY_PATH) $(MAIN_PATH)

# --- Quality ---

test:
	@echo "Running tests..."
	@go test ./internal/... -count=1 -race

vet:
	@echo "Running vet..."
	@go vet ./...

lint:
	@echo "Linting code..."
	@golangci-lint run

# --- Docker ---

up:
	@echo "Starting services..."
	@$(DOCKER_COMPOSE) up -d

down:
	@echo "Stopping services..."
	@$(DOCKER_COMPOSE) down

logs:
	@$(DOCKER_COMPOSE) logs -f app

# --- Database Migrations ---

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

migrate-up:
	@echo "Applying migrations..."
	@migrate -database "$(DSN)" -path ./migrations up

migrate-down:
	@echo "Reverting last migration..."
	@migrate -database "$(DSN)" -path ./migrations down 1

migrate-status:
	@echo "Migration status..."
	@migrate -database "$(DSN)" -path ./migrations version

# --- Cleanup ---

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_PATH)

# --- Git ---

REPO_URL = https://github.com/Orchestrae/payflow-backend.git

git-setup:
	@echo "Setting up git remote..."
	@if [ -n "$$GH_TOKEN" ] && [ -n "$$GIT_USERNAME" ]; then \
		git remote set-url origin https://$$GIT_USERNAME:$$GH_TOKEN@github.com/Orchestrae/payflow-backend.git; \
		echo "Remote configured (GIT_USERNAME:GH_TOKEN)"; \
	elif command -v gh >/dev/null 2>&1; then \
		git remote set-url origin https://$$(gh auth token)@github.com/Orchestrae/payflow-backend.git; \
		echo "Remote configured (gh auth token)"; \
	else \
		read -p "GitHub username: " u; read -sp "Token: " t; echo; \
		git remote set-url origin https://$$u:$$t@github.com/Orchestrae/payflow-backend.git; \
		echo "Remote configured"; \
	fi

push:
	@echo "Pushing to remote..."
	@BRANCH=$$(git branch --show-current); \
	if [ -n "$$GH_TOKEN" ] && [ -n "$$GIT_USERNAME" ]; then \
		git remote set-url origin https://$$GIT_USERNAME:$$GH_TOKEN@github.com/Orchestrae/payflow-backend.git; \
	elif command -v gh >/dev/null 2>&1; then \
		git remote set-url origin https://$$(gh auth token)@github.com/Orchestrae/payflow-backend.git; \
	fi; \
	if git ls-remote --heads origin $$BRANCH 2>/dev/null | grep -q $$BRANCH; then \
		git push origin $$BRANCH; \
	else \
		git push --set-upstream origin $$BRANCH; \
	fi; \
	git remote set-url origin $(REPO_URL); \
	echo "Push successful!"

push-fast:
	@git push -u origin $$(git branch --show-current)

push-new:
	@read -p "Enter branch name: " branch; \
	git checkout -b $$branch; \
	git push -u origin $$branch

# --- Help ---

help:
	@echo ""
	@echo "PayFlow Makefile"
	@echo "================"
	@echo ""
	@echo "  Application:"
	@echo "    run              Run the server"
	@echo "    build            Build binary to ./bin/payflow"
	@echo ""
	@echo "  Quality:"
	@echo "    test             Run tests with race detector"
	@echo "    vet              Run go vet"
	@echo "    lint             Run golangci-lint"
	@echo ""
	@echo "  Docker:"
	@echo "    up               Start all services (PostgreSQL, Redis, MailHog)"
	@echo "    down             Stop all services"
	@echo "    logs             Follow app logs"
	@echo ""
	@echo "  Migrations:"
	@echo "    migrate-create   Create new migration (prompts for name)"
	@echo "    migrate-up       Apply all pending migrations"
	@echo "    migrate-down     Revert last migration"
	@echo "    migrate-status   Show current migration version"
	@echo ""
	@echo "  Git:"
	@echo "    push             Push current branch (handles auth)"
	@echo "    push-fast        Push without auth setup"
	@echo "    push-new         Create and push new branch"
	@echo ""
	@echo "  DSN=<url> make migrate-up   (custom database URL)"
	@echo ""
