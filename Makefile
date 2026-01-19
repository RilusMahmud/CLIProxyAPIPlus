# Makefile for CLI Proxy API Plus
# Build instructions following .goreleaser.yml conventions

# Version variables
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
DOCKER_USERNAME ?= rilusmahmud

# Build variables
BINARY_NAME = cli-proxy-api-plus
MAIN_PATH = ./cmd/server
LDFLAGS = -s -w \
	-X 'main.Version=$(VERSION)-plus' \
	-X 'main.Commit=$(COMMIT)' \
	-X 'main.BuildDate=$(BUILD_DATE)'

# Docker variables
IMAGE_NAME = $(DOCKER_USERNAME)/cli-proxy-api-plus
DOCKER_TAG ?= $(VERSION)

# Output directories
BUILD_DIR = build
DIST_DIR = dist

.PHONY: help
help: ## Show this help message
	@echo "CLI Proxy API Plus - Build Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: build-macos
build-macos: ## Build for macOS (current architecture)
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=$(shell go env GOARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-$(shell go env GOARCH) \
		$(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-$(shell go env GOARCH)"

.PHONY: build-macos-amd64
build-macos-amd64: ## Build for macOS AMD64
	@echo "Building for macOS AMD64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 \
		$(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"

.PHONY: build-macos-arm64
build-macos-arm64: ## Build for macOS ARM64 (Apple Silicon)
	@echo "Building for macOS ARM64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 \
		$(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

.PHONY: build-linux
build-linux: ## Build for Linux (current architecture)
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=$(shell go env GOARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-$(shell go env GOARCH) \
		$(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-$(shell go env GOARCH)"

.PHONY: build-linux-amd64
build-linux-amd64: ## Build for Linux AMD64
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 \
		$(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

.PHONY: build-linux-arm64
build-linux-arm64: ## Build for Linux ARM64
	@echo "Building for Linux ARM64..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 \
		$(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

.PHONY: build-all
build-all: build-macos-amd64 build-macos-arm64 build-linux-amd64 build-linux-arm64 ## Build for all platforms
	@echo "All builds complete!"

.PHONY: clean
clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@echo "Clean complete!"

.PHONY: version
version: ## Show version information
	@echo "Version:    $(VERSION)-plus"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

.PHONY: docker-build
docker-build: ## Build Docker image with version info
	@echo "Building Docker image..."
	@echo "Version:    $(VERSION)-plus"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Image Tag:  $(IMAGE_NAME):$(DOCKER_TAG)"
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE_NAME):$(DOCKER_TAG) \
		-t $(IMAGE_NAME):latest \
		.
	@echo "Docker build complete: $(IMAGE_NAME):$(DOCKER_TAG)"
