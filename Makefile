BINARY_NAME=kv-store
CMD_PATH=./cmd/app

# Config path
CONFIG_PATH=./config/config.yml

# Docker parameters
DOCKER_IMAGE_NAME=kv-store

UNIT_TESTS_PKGS := $(shell go list ./... | grep -v /mocks | grep -v /gen | grep -v /testutils | grep -v /cmd | grep -v /api)

.PHONY: help build run test test-cover test-perf lint clean docker-build docker-up docker-down swag proto-grpc/compile proto-entries/compile

docker/up: # Build containers and start them in background
	@docker compose up -d --build

docker/down: # Stop and remove containers with their volumes
	@docker compose down --volumes

test: ## Run all unit tests
	@mkdir -p coverage
	@go test -v -race \
    -coverprofile=coverage/coverage.out -covermode=atomic ${UNIT_TESTS_PKGS}

test-cover: ## Run unit tests and generate HTML coverage report
	@go test ${UNIT_TESTS_PKGS} -v -coverprofile=coverage.out
	@echo "Coverage report generated: coverage.html"
	@go tool cover -html=coverage.out -o coverage.html

lint: ## Lint the Go code using golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint not found. Please install it: https://golangci-lint.run/usage/install/"; exit 1)
	@golangci-lint run ./...

swag: ## Regenerate OpenAPI documentation
	@swag init -g cmd/app/main.go -o api/openapi

proto-grpc/compile: # Recompile proto grpc
	@mkdir -p proto/grpc/gen
	@protoc -I ./proto/grpc \
	--go_out ./proto/grpc/gen --go_opt=paths=source_relative \
	--go-grpc_out ./proto/grpc/gen --go-grpc_opt=paths=source_relative \
	./proto/grpc/kv-store.proto

proto-entries/compile: # Recompile proto log entries
	@mkdir -p proto/log_entries/gen
	@protoc -I ./proto/log_entries \
	--go_out ./proto/log_entries/gen --go_opt=paths=source_relative \
	--go-grpc_out ./proto/log_entries/gen --go-grpc_opt=paths=source_relative \
	./proto/log_entries/entries.proto

proto-fsm/compile: # Recompile proto fsm
	@mkdir -p proto/fsm/gen
	@protoc -I ./proto/fsm \
	--go_out ./proto/fsm/gen --go_opt=paths=source_relative \
	./proto/fsm/commands.proto

load-test/run: # Run k6 load test
	@docker run --rm -i \
	--network=host \
	-v $(CURDIR)/internal/tests/k6_scenarios:/src \
	-e BASE_URL=http://localhost:8081 \
	grafana/k6 run /src/load_test.js
