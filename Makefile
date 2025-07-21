.PHONY: run/main

run/main:
	@go run ./cmd/app

run/tests:
	@go test ./... -v -cover
