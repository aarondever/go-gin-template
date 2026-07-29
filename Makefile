# Go related variables
BINARY_NAME := main
MAIN_PATH := ./cmd/server
BUILD_DIR := ./build
TMP_DIR := ./tmp

# Build flags
LDFLAGS := -s -w

.DEFAULT_GOAL := help

# Help target
.PHONY: help
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development targets
.PHONY: dev
dev: ## Start development server with hot reload
	@air

# Build targets
.PHONY: build
build: ## Build the application
	@go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Testing targets
.PHONY: test
test: ## Run tests
	@go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

.PHONY: test-race
test-race: ## Run tests with race detection
	@go test -v -race ./...

# Cleaning targets
.PHONY: clean
clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR) $(TMP_DIR) coverage.out coverage.html
	@go clean -cache

# Install development dependencies
.PHONY: install-deps
install-deps: ## Install development dependencies
	@go install github.com/air-verse/air@latest
