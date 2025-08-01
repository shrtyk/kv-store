.PHONY: run/main

run/main:
	@go run ./cmd/app -cfg_path=./config/config.yml

run/tests:
	@go test ./internal/... ./pkg/... -v -cover
