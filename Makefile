.PHONY: help build test lint fmt clean run docker-build docker-run check coverage install-deps

# Variables
GO := go
GOFLAGS := -v
BINARY_NAME := tado-exporter
BINARY_PATH := ./$(BINARY_NAME)
DOCKER_IMAGE := tado-prometheus-exporter
DOCKER_TAG := latest
PORT ?= 9100
TOKEN_PATH ?= ~/.tado-exporter/token.json
TOKEN_PASSPHRASE ?=
SCRAPE_TIMEOUT ?= 10

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[0;33m
NC := \033[0m # No Color

# Default target
help:
	@echo "$(BLUE)tado-prometheus-exporter - Development Makefile$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@echo "  $(YELLOW)build$(NC)              - Build the exporter binary"
	@echo "  $(YELLOW)test$(NC)               - Run all tests"
	@echo "  $(YELLOW)test-verbose$(NC)       - Run tests with verbose output"
	@echo "  $(YELLOW)test-coverage$(NC)      - Run tests with coverage report"
	@echo "  $(YELLOW)coverage$(NC)           - Generate and open coverage report (HTML)"
	@echo "  $(YELLOW)lint$(NC)               - Run golangci-lint"
	@echo "  $(YELLOW)fmt$(NC)                - Format code with gofmt -s (modifies files)"
	@echo "  $(YELLOW)fmt-check$(NC)          - Check code formatting without modifying files"
	@echo "  $(YELLOW)vet$(NC)                - Run go vet"
	@echo "  $(YELLOW)check$(NC)              - Run build, fmt-check, lint, vet, test (full check)"
	@echo "  $(YELLOW)clean$(NC)              - Remove binary and build artifacts"
	@echo "  $(YELLOW)run$(NC)                - Build and run the exporter"
	@echo "  $(YELLOW)docker-build$(NC)       - Build Docker image"
	@echo "  $(YELLOW)docker-run$(NC)         - Run Docker container"
	@echo "  $(YELLOW)docker-push$(NC)        - Push Docker image to registry"
	@echo "  $(YELLOW)install-deps$(NC)       - Install development dependencies"
	@echo "  $(YELLOW)mod-tidy$(NC)           - Tidy go.mod and go.sum"
	@echo "  $(YELLOW)mod-download$(NC)       - Download go.mod dependencies"
	@echo "  $(YELLOW)help$(NC)               - Show this help message"
	@echo ""
	@echo "$(GREEN)Usage examples:$(NC)"
	@echo "  make build"
	@echo "  make test"
	@echo "  make check"
	@echo "  make run TOKEN_PASSPHRASE=my-secret-passphrase"
	@echo "  make docker-build DOCKER_TAG=v1.0.0"
	@echo "  make docker-run TOKEN_PASSPHRASE=my-secret-passphrase"

# Build the exporter binary
build:
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	$(GO) build $(GOFLAGS) -o $(BINARY_PATH) ./cmd/exporter
	@echo "$(GREEN)✓ Binary built: $(BINARY_PATH)$(NC)"

# Run all tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	$(GO) test -v ./... -timeout 30s
	@echo "$(GREEN)✓ Tests passed$(NC)"

# Run tests with verbose output
test-verbose:
	@echo "$(BLUE)Running tests (verbose)...$(NC)"
	$(GO) test -v -race ./... -timeout 30s
	@echo "$(GREEN)✓ Tests passed$(NC)"

# Run tests with coverage
test-coverage:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	$(GO) test -v -coverprofile=coverage.out ./... -timeout 30s
	@echo "$(GREEN)✓ Coverage report: coverage.out$(NC)"

# Generate HTML coverage report
coverage: test-coverage
	@echo "$(BLUE)Generating coverage report...$(NC)"
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report: coverage.html$(NC)"
	@if command -v open > /dev/null; then open coverage.html; fi

# Run golangci-lint
lint:
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run --timeout=5m ./...; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not found. Install with: make install-deps$(NC)"; \
	fi
	@echo "$(GREEN)✓ Linting passed$(NC)"

# Format code with gofmt -s (simplified formatting)
fmt:
	@echo "$(BLUE)Formatting code with gofmt -s...$(NC)"
	gofmt -s -w .
	@echo "$(GREEN)✓ Code formatted$(NC)"

# Check code formatting without modifying files
fmt-check:
	@echo "$(BLUE)Checking code formatting...$(NC)"
	@unformatted=$$(gofmt -s -l . 2>/dev/null); \
	if [ -n "$$unformatted" ]; then \
		echo "$(RED)✗ Code formatting issues found:$(NC)"; \
		echo "$$unformatted"; \
		echo "$(YELLOW)Run 'make fmt' to fix formatting$(NC)"; \
		exit 1; \
	fi; \
	echo "$(GREEN)✓ Code formatting is correct$(NC)"

# Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	$(GO) vet ./...
	@echo "$(GREEN)✓ Vet passed$(NC)"

# Full check: build, fmt-check, lint, vet, test
check: build fmt-check lint vet test
	@echo "$(GREEN)✓ All checks passed$(NC)"

# Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -f $(BINARY_PATH)
	rm -f coverage.out coverage.html
	$(GO) clean -testcache
	@echo "$(GREEN)✓ Cleaned$(NC)"

# Build and run the exporter
run: build
	@echo "$(BLUE)Starting exporter...$(NC)"
	@if [ -z "$(TOKEN_PASSPHRASE)" ]; then \
		echo "$(RED)✗ TOKEN_PASSPHRASE is required$(NC)"; \
		echo "$(YELLOW)Usage: make run TOKEN_PASSPHRASE=your-passphrase$(NC)"; \
		exit 1; \
	fi
	$(BINARY_PATH) \
		--token-path=$(TOKEN_PATH) \
		--token-passphrase=$(TOKEN_PASSPHRASE) \
		--port=$(PORT) \
		--scrape-timeout=$(SCRAPE_TIMEOUT)

# Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)✓ Docker image built$(NC)"

# Run Docker container
docker-run: docker-build
	@echo "$(BLUE)Running Docker container...$(NC)"
	@if [ -z "$(TOKEN_PASSPHRASE)" ]; then \
		echo "$(RED)✗ TOKEN_PASSPHRASE is required$(NC)"; \
		echo "$(YELLOW)Usage: make docker-run TOKEN_PASSPHRASE=your-passphrase$(NC)"; \
		exit 1; \
	fi
	docker run -it --rm \
		-e TADO_TOKEN_PASSPHRASE=$(TOKEN_PASSPHRASE) \
		-e TADO_PORT=$(PORT) \
		-v ~/.tado-exporter:/root/.tado-exporter \
		-p $(PORT):$(PORT) \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Push Docker image to registry
docker-push:
	@echo "$(BLUE)Pushing Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"
	@if [ -z "$(REGISTRY)" ]; then \
		echo "$(YELLOW)⚠ REGISTRY not set. Using local Docker image.$(NC)"; \
		echo "$(YELLOW)Usage: make docker-push REGISTRY=docker.io/myrepo$(NC)"; \
	else \
		docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG); \
		docker push $(REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG); \
		echo "$(GREEN)✓ Docker image pushed$(NC)"; \
	fi

# Install development dependencies
install-deps:
	@echo "$(BLUE)Installing development dependencies...$(NC)"
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

# Tidy go.mod and go.sum
mod-tidy:
	@echo "$(BLUE)Tidying go.mod and go.sum...$(NC)"
	$(GO) mod tidy
	@echo "$(GREEN)✓ Tidied$(NC)"

# Download go.mod dependencies
mod-download:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	$(GO) mod download
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

# Info target
info:
	@echo "$(BLUE)Project Information:$(NC)"
	@echo "  Binary:        $(BINARY_PATH)"
	@echo "  Go Version:    $$($(GO) version)"
	@echo "  Docker Image:  $(DOCKER_IMAGE):$(DOCKER_TAG)"
	@echo "  Port:          $(PORT)"
	@echo ""
	@echo "$(BLUE)Build Variables:$(NC)"
	@echo "  CLIENT_ID:     $(CLIENT_ID)"
	@echo "  TOKEN_PATH:    $(TOKEN_PATH)"
	@echo "  SCRAPE_TIMEOUT: $(SCRAPE_TIMEOUT)"

# Quick development loop: watch for changes and rebuild
dev:
	@echo "$(BLUE)Starting development mode...$(NC)"
	@echo "$(YELLOW)Run tests and build on changes (requires fswatch or similar)$(NC)"
	@if command -v fswatch > /dev/null; then \
		fswatch -o . | xargs -n1 -I {} make test build; \
	else \
		echo "$(RED)✗ fswatch not found. Install with: brew install fswatch$(NC)"; \
		exit 1; \
	fi
