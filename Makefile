.PHONY: help docker-build docker-run docker-test test lint clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

docker-build: ## Build Docker image
	docker build --no-cache -t vision-api-app:latest .

docker-run: ## Run application in Docker
	docker compose up

docker-test: ## Run tests in Docker
	docker build --no-cache -t vision-api-app:test .
	docker run --rm vision-api-app:test go test -v ./...

docker-shell: ## Open shell in Docker container
	docker run --rm -it vision-api-app:latest /bin/sh

test: ## Run tests locally
	go test -v -cover ./...

test-coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint: ## Run linter
	golangci-lint run ./...

clean: ## Clean build artifacts
	rm -f vision-api-app
	rm -f coverage.out
	go clean -cache -testcache

.DEFAULT_GOAL := help

