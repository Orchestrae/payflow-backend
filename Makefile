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
	@go run $(MAIN_PATH)

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