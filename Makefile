BINARY_NAME=kv-store
CMD_PATH=./cmd/app

# Config path
CONFIG_PATH=./config/config.yml

# Docker parameters
DOCKER_IMAGE_NAME=kv-store

.PHONY: help build run test test-cover test-perf lint clean docker-build docker-up docker-down swag

build: ## Build the Go binary
	@go build -o $(BINARY_NAME) -ldflags="-w -s" $(CMD_PATH)

run: ## Run the application locally
	@go run $(CMD_PATH) -cfg_path=$(CONFIG_PATH)

test: ## Run all unit tests
	@go test ./internal/... ./pkg/... -v

test-cover: ## Run unit tests and generate HTML coverage report
	@go test ./internal/... ./pkg/... -v -coverprofile=coverage.out
	@echo "Coverage report generated: coverage.html"
	@go tool cover -html=coverage.out -o coverage.html

test-perf: ## Run performance tests against a running instance
	@go test -v -run=TestFunctional_BulkPutDeletePerformance ./tests

lint: ## Lint the Go code using golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not found. Please install it: https://golangci-lint.run/usage/install/"; exit 1)
	@golangci-lint run ./...

swag: ## Regenerate OpenAPI documentation
	@swag init -g cmd/app/main.go -o api/http

clean: ## Clean up build artifacts and coverage reports
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html

docker-build: ## Build the Docker image
	@DOCKER_BUILDKIT=1 docker build -t $(DOCKER_IMAGE_NAME) .

docker-up: ## Start services with Docker Compose in detached mode
	@docker-compose up -d

docker-down: ## Stop and remove services started with Docker Compose
	@docker-compose down
