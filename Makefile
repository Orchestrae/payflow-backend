# Makefile

.PHONY: all run build test lint clean help

# Variables
BINARY_NAME=payflow
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/server/main.go
DOCKER_COMPOSE=docker-compose

# Default target
all: build

# Runs the application using go run
run:
	@echo "Running application..."
	@go run cmd/server/main.go

# Builds the application binary
build:
	@echo "Building binary..."
	@go build -o $(BINARY_PATH) $(MAIN_PATH)

# Runs all tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Lints the code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Removes the built binary
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_PATH)

# Runs docker-compose up
up:
	@echo "Starting services with docker-compose..."
	@$(DOCKER_COMPOSE) up -d

# Runs docker-compose down
down:
	@echo "Stopping services with docker-compose..."
	@$(DOCKER_COMPOSE) down

# --- Database Migrations ---
# Ensure DSN is set in your environment or .env file for local use
# DSN="postgres://payflow_user:payflow_secret@localhost:5432/payflow_db?sslmode=disable"
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

migrate-up:
	@echo "Applying migrations..."
	@migrate -database "$(DSN)" -path ./migrations up

migrate-down:
	@echo "Reverting last migration..."
	@migrate -database "$(DSN)" -path ./migrations down 1

# --- Git Commands ---
# Git username (can be set via GIT_USERNAME env var or will be prompted)
GIT_USERNAME ?= $(shell if [ -z "$$GIT_USERNAME" ]; then read -p "Enter GitHub username: " username && echo $$username; else echo $$GIT_USERNAME; fi)

# Ensures remote URL is set correctly with specified username
git-setup:
	@echo "Setting up git remote..."
	@if [ -z "$(GIT_USERNAME)" ]; then \
		read -p "Enter GitHub username: " username; \
		git remote set-url origin https://$$username@github.com/Orchestrae/payflow-backend.git; \
	else \
		git remote set-url origin https://$(GIT_USERNAME)@github.com/Orchestrae/payflow-backend.git; \
	fi
	@echo "✅ Remote URL configured"

# Pushes current branch to origin (prompts for username if not set)
push:
	@echo "Pushing to remote..."
	@if [ -z "$$GIT_USERNAME" ]; then \
		read -p "Enter GitHub username: " username; \
		git remote set-url origin https://$$username@github.com/Orchestrae/payflow-backend.git; \
	else \
		git remote set-url origin https://$$GIT_USERNAME@github.com/Orchestrae/payflow-backend.git; \
	fi
	@git push -u origin $$(git branch --show-current)
	@echo "✅ Push successful!"

# Pushes current branch (assumes remote is already configured)
push-fast:
	@echo "Pushing to remote..."
	@git push -u origin $$(git branch --show-current)
	@echo "✅ Push successful!"

# Creates a new branch and pushes it
push-new:
	@read -p "Enter branch name: " branch; \
	if [ -z "$$GIT_USERNAME" ]; then \
		read -p "Enter GitHub username: " username; \
		git remote set-url origin https://$$username@github.com/Orchestrae/payflow-backend.git; \
	else \
		git remote set-url origin https://$$GIT_USERNAME@github.com/Orchestrae/payflow-backend.git; \
	fi; \
	git checkout -b $$branch; \
	git push -u origin $$branch; \
	echo "✅ Branch $$branch created and pushed!"

# Displays help message
help:
	@echo "Available commands:"
	@echo "  run           - Run the application"
	@echo "  build         - Build the application binary"
	@echo "  test          - Run tests"
	@echo "  lint          - Lint the code"
	@echo "  clean         - Remove the built binary"
	@echo "  up            - Start services with docker-compose"
	@echo "  down          - Stop services with docker-compose"
	@echo "  migrate-create - Create a new SQL migration file"
	@echo "  migrate-up    - Apply all up migrations"
	@echo "  migrate-down  - Revert the last migration"
	@echo ""
	@echo "Git commands:"
	@echo "  git-setup     - Configure git remote (prompts for username)"
	@echo "  push          - Push current branch (prompts for username if GIT_USERNAME not set)"
	@echo "  push-fast     - Push current branch (assumes remote configured)"
	@echo "  push-new      - Create new branch and push it"
	@echo ""
	@echo "  Usage: GIT_USERNAME=username make push  (to avoid prompts)"