.PHONY: help docker-build docker-run docker-test test lint clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

docker-build: ## Build Docker image
	docker build --no-cache -t tesseract-ocr-app:latest .

docker-run: ## Run application in Docker
	docker compose up

docker-test: ## Run tests in Docker
	docker build --no-cache -t tesseract-ocr-app:test .
	docker run --rm tesseract-ocr-app:test go test -v ./internal/domain/... ./internal/usecase/... ./internal/config/...

docker-shell: ## Open shell in Docker container
	docker run --rm -it tesseract-ocr-app:latest /bin/sh

test: ## Run tests locally (without Infrastructure layer)
	go test -v -cover ./internal/domain/... ./internal/usecase/... ./internal/config/...

test-coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./internal/domain/... ./internal/usecase/... ./internal/config/...
	go tool cover -html=coverage.out

lint: ## Run linter (without Infrastructure layer)
	golangci-lint run --build-tags=no_tesseract,no_opencv ./...

clean: ## Clean build artifacts
	rm -f tesseract-ocr-app
	rm -f coverage.out
	go clean -cache -testcache

.DEFAULT_GOAL := help

