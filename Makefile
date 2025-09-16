# GoDash - System Monitoring Tool
# Makefile for build, test, and deployment tasks

# Variables
APP_NAME := godash
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go variables
GO_VERSION := 1.19
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Docker variables
DOCKER_IMAGE := godash
DOCKER_TAG := $(VERSION)
DOCKER_REGISTRY := ghcr.io/eyzaun

# Build flags
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
PURPLE := \033[0;35m
CYAN := \033[0;36m
WHITE := \033[0;37m
NC := \033[0m # No Color

.PHONY: help build test clean run dev docker deps lint security fmt vet check install uninstall

# Default target
.DEFAULT_GOAL := help

## Help - Show this help message
help:
	@echo "$(CYAN)╔══════════════════════════════════════════════╗$(NC)"
	@echo "$(CYAN)║            GoDash - Build System             ║$(NC)"
	@echo "$(CYAN)╚══════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Environment:$(NC)"
	@echo "  App Name:    $(PURPLE)$(APP_NAME)$(NC)"
	@echo "  Version:     $(PURPLE)$(VERSION)$(NC)"
	@echo "  Go Version:  $(PURPLE)$(GO_VERSION)$(NC)"
	@echo "  Platform:    $(PURPLE)$(GOOS)/$(GOARCH)$(NC)"
	@echo "  Git Commit:  $(PURPLE)$(GIT_COMMIT)$(NC)"
	@echo ""

## Build - Build the application
build: clean
	@echo "$(BLUE)Building $(APP_NAME) v$(VERSION) for $(GOOS)/$(GOARCH)...$(NC)"
	@mkdir -p build
	@go build $(LDFLAGS) -o build/$(APP_NAME) .
	@go build $(LDFLAGS) -o build/$(APP_NAME)-cli ./cmd/godash-cli
	@echo "$(GREEN)✓ Build completed successfully$(NC)"
	@ls -la build/

## Build-all - Build for all platforms
build-all: clean
	@echo "$(BLUE)Building $(APP_NAME) for all platforms...$(NC)"
	@mkdir -p build/dist
	
	# Linux builds
	@echo "$(YELLOW)Building for Linux...$(NC)"
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/dist/$(APP_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o build/dist/$(APP_NAME)-linux-arm64 .
	
	# Windows builds
	@echo "$(YELLOW)Building for Windows...$(NC)"
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/dist/$(APP_NAME)-windows-amd64.exe .
	
	# macOS builds
	@echo "$(YELLOW)Building for macOS...$(NC)"
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/dist/$(APP_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/dist/$(APP_NAME)-darwin-arm64 .
	
	@echo "$(GREEN)✓ Cross-platform build completed$(NC)"
	@ls -la build/dist/

## Test - Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Tests completed$(NC)"
	@echo "$(YELLOW)Coverage report: coverage.html$(NC)"

## Test-short - Run short tests only
test-short:
	@echo "$(BLUE)Running short tests...$(NC)"
	@go test -short -v ./...
	@echo "$(GREEN)✓ Short tests completed$(NC)"

## Benchmark - Run benchmarks
benchmark:
	@echo "$(BLUE)Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...
	@echo "$(GREEN)✓ Benchmarks completed$(NC)"

## Lint - Run linters
lint:
	@echo "$(BLUE)Running linters...$(NC)"
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "$(YELLOW)Installing golangci-lint...$(NC)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run --timeout=5m
	@echo "$(GREEN)✓ Linting completed$(NC)"

## Security - Run security checks
security:
	@echo "$(BLUE)Running security checks...$(NC)"
	@if ! command -v gosec &> /dev/null; then \
		echo "$(YELLOW)Installing gosec...$(NC)"; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -no-fail -fmt sarif -out results.sarif ./...
	@gosec ./...
	@echo "$(GREEN)✓ Security checks completed$(NC)"

## Fmt - Format code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

## Vet - Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ Vet completed$(NC)"

## Check - Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "$(GREEN)✓ All checks passed$(NC)"

## Deps - Download and tidy dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@go mod verify
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

## Clean - Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf build/
	@rm -f coverage.out coverage.html
	@rm -f results.sarif
	@go clean -cache -testcache -modcache
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

## Run - Run the application locally
run: build
	@echo "$(BLUE)Running $(APP_NAME)...$(NC)"
	@./build/$(APP_NAME)

## Dev - Run in development mode with hot reload
dev:
	@echo "$(BLUE)Starting development server with hot reload...$(NC)"
	@if ! command -v air &> /dev/null; then \
		echo "$(YELLOW)Installing air for hot reload...$(NC)"; \
		go install github.com/air-verse/air@latest; \
	fi
	@air

## DB-up - Start database with Docker Compose
db-up:
	@echo "$(BLUE)Starting PostgreSQL database...$(NC)"
	@docker-compose up -d postgres redis
	@echo "$(GREEN)✓ Database started$(NC)"
	@echo "$(YELLOW)Connection: postgresql://godash:password@localhost:5433/godash$(NC)"

## DB-down - Stop database
db-down:
	@echo "$(BLUE)Stopping database...$(NC)"
	@docker-compose down
	@echo "$(GREEN)✓ Database stopped$(NC)"

## DB-reset - Reset database (drop and recreate)
db-reset: db-down
	@echo "$(BLUE)Resetting database...$(NC)"
	@docker-compose down -v
	@docker-compose up -d postgres redis
	@sleep 5
	@echo "$(GREEN)✓ Database reset completed$(NC)"

## DB-migrate - Run database migrations
db-migrate:
	@echo "$(BLUE)Running database migrations...$(NC)"
	@echo "$(YELLOW)Note: Auto-migration runs automatically when starting the app$(NC)"
	@echo "$(GREEN)✓ Auto-migration will run on next app start$(NC)"

## Docker-build - Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest .
	@echo "$(GREEN)✓ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

## Docker-run - Run Docker container
docker-run: docker-build
	@echo "$(BLUE)Running Docker container...$(NC)"
	@docker-compose up --build

## Docker-push - Push Docker image to registry
docker-push: docker-build
	@echo "$(BLUE)Pushing Docker image to $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)...$(NC)"
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker tag $(DOCKER_IMAGE):latest $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	@echo "$(GREEN)✓ Docker image pushed$(NC)"

## Install - Install the application globally
install: build
	@echo "$(BLUE)Installing $(APP_NAME) globally...$(NC)"
	@sudo cp build/$(APP_NAME) /usr/local/bin/
	@sudo cp build/$(APP_NAME)-cli /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(APP_NAME)
	@sudo chmod +x /usr/local/bin/$(APP_NAME)-cli
	@echo "$(GREEN)✓ $(APP_NAME) installed to /usr/local/bin/$(NC)"

## Uninstall - Remove the application
uninstall:
	@echo "$(BLUE)Uninstalling $(APP_NAME)...$(NC)"
	@sudo rm -f /usr/local/bin/$(APP_NAME)
	@sudo rm -f /usr/local/bin/$(APP_NAME)-cli
	@echo "$(GREEN)✓ $(APP_NAME) uninstalled$(NC)"

## Release - Create a release (tag and build)
release: check
	@echo "$(BLUE)Creating release $(VERSION)...$(NC)"
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "$(RED)Error: Cannot release dev version$(NC)"; \
		exit 1; \
	fi
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)
	@$(MAKE) build-all
	@echo "$(GREEN)✓ Release $(VERSION) created$(NC)"

## Stats - Show project statistics
stats:
	@echo "$(CYAN)╔══════════════════════════════════════════════╗$(NC)"
	@echo "$(CYAN)║              Project Statistics              ║$(NC)"
	@echo "$(CYAN)╚══════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)Code Statistics:$(NC)"
	@echo "  Go files:     $(PURPLE)$$(find . -name '*.go' -not -path './vendor/*' | wc -l)$(NC)"
	@echo "  Lines of code: $(PURPLE)$$(find . -name '*.go' -not -path './vendor/*' -exec wc -l {} \; | awk '{sum += $$1} END {print sum}')$(NC)"
	@echo "  Test files:   $(PURPLE)$$(find . -name '*_test.go' -not -path './vendor/*' | wc -l)$(NC)"
	@echo ""
	@echo "$(YELLOW)Dependencies:$(NC)"
	@echo "  Direct deps:  $(PURPLE)$$(go list -m all | grep -v '^$(shell go list -m)$$' | wc -l)$(NC)"
	@echo "  Go version:   $(PURPLE)$$(go version | cut -d' ' -f3)$(NC)"
	@echo ""
	@echo "$(YELLOW)Git Information:$(NC)"
	@echo "  Branch:       $(PURPLE)$$(git branch --show-current 2>/dev/null || echo 'unknown')$(NC)"
	@echo "  Last commit:  $(PURPLE)$$(git log -1 --format='%h - %s (%cr)' 2>/dev/null || echo 'unknown')$(NC)"
	@echo "  Contributors: $(PURPLE)$$(git shortlog -sn | wc -l 2>/dev/null || echo 'unknown')$(NC)"

## Debug - Show debug information
debug:
	@echo "$(CYAN)╔══════════════════════════════════════════════╗$(NC)"
	@echo "$(CYAN)║              Debug Information               ║$(NC)"
	@echo "$(CYAN)╚══════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)Build Variables:$(NC)"
	@echo "  APP_NAME:     $(PURPLE)$(APP_NAME)$(NC)"
	@echo "  VERSION:      $(PURPLE)$(VERSION)$(NC)"
	@echo "  BUILD_TIME:   $(PURPLE)$(BUILD_TIME)$(NC)"
	@echo "  GIT_COMMIT:   $(PURPLE)$(GIT_COMMIT)$(NC)"
	@echo "  GOOS:         $(PURPLE)$(GOOS)$(NC)"
	@echo "  GOARCH:       $(PURPLE)$(GOARCH)$(NC)"
	@echo ""
	@echo "$(YELLOW)Docker Variables:$(NC)"
	@echo "  DOCKER_IMAGE: $(PURPLE)$(DOCKER_IMAGE)$(NC)"
	@echo "  DOCKER_TAG:   $(PURPLE)$(DOCKER_TAG)$(NC)"
	@echo "  REGISTRY:     $(PURPLE)$(DOCKER_REGISTRY)$(NC)"
	@echo ""
	@echo "$(YELLOW)LDFLAGS:$(NC)"
	@echo "  $(PURPLE)$(LDFLAGS)$(NC)"

## API-docs - Generate API documentation
api-docs:
	@echo "$(BLUE)Generating API documentation...$(NC)"
	@if ! command -v swag &> /dev/null; then \
		echo "$(YELLOW)Installing swag...$(NC)"; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@swag init --dir ./ --generalInfo main.go --output ./docs
	@echo "$(GREEN)✓ API documentation generated in ./docs$(NC)"

## Init - Initialize development environment
init:
	@echo "$(BLUE)Initializing development environment...$(NC)"
	@$(MAKE) deps
	@$(MAKE) db-up
	@sleep 10
	@$(MAKE) db-migrate
	@echo "$(GREEN)✓ Development environment initialized$(NC)"
	@echo "$(YELLOW)Next steps:$(NC)"
	@echo "  1. Run '$(GREEN)make dev$(NC)' to start development server"
	@echo "  2. Visit '$(GREEN)http://localhost:8080/health$(NC)' to check status"
	@echo "  3. API docs at '$(GREEN)http://localhost:8080/api/v1/$(NC)'"

# Include OS-specific targets
ifeq ($(GOOS),darwin)
include Makefile.darwin
endif

ifeq ($(GOOS),linux)
include Makefile.linux
endif

ifeq ($(GOOS),windows)
include Makefile.windows
endif

# (local analysis helpers removed)