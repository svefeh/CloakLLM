# Project variables
APP_NAME := cloakllm-daemon
BUILD_DIR := bin
MAIN_FILE := ./cmd/cloakllm-daemon

# Go specific variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod

# LDFLAGS for smaller, optimized binaries (stripping debug symbols)
LDFLAGS := -ldflags="-s -w"

.PHONY: all build build-linux build-windows build-mac container clean test run tidy help

# Default target
all: clean build

# 1. Standard build for the current host OS
build:
	@echo ">>> Building $(APP_NAME) for the current system..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo ">>> Done! Binary is located in $(BUILD_DIR)/"

# Initialize local development environment with pre-commit hooks
init:
	@echo ">>> Installing pre-commit hooks..."
	pre-commit install
	pre-commit install --hook-type commit-msg
	@echo ">>> Done! Use 'cz c' to commit your changes following the Conventional Commits standard."

# 2. Cross-compilation for Linux (amd64)
build-linux:
	@echo ">>> Building $(APP_NAME) for Linux (amd64)..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_FILE)

# 3. Cross-compilation for Windows (amd64)
build-windows:
	@echo ">>> Building $(APP_NAME) for Windows (amd64)..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(MAIN_FILE)

# 4. Cross-compilation for macOS (Apple Silicon)
build-mac:
	@echo ">>> Building $(APP_NAME) for macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN_FILE)

# Build binaries for all supported platforms at once
build-all: build-linux build-windows build-mac
	@echo ">>> All platform builds completed successfully!"

# Build the OCI container locally using Podman
container:
	@echo ">>> Building OCI container image using Podman..."
	podman build -t svefeh/cloakllm:local -f Containerfile .

# Clean up the build directory
clean:
	@echo ">>> Cleaning up old builds..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f cloakllm-daemon

# Run unit tests
test:
	@echo ">>> Running unit tests..."
	$(GOTEST) -v ./...

# Clean up and verify go.mod dependencies
tidy:
	@echo ">>> Checking go.mod and downloading missing modules..."
	$(GOMOD) tidy

# Run the application directly in development mode
run:
	@echo ">>> Starting $(APP_NAME) in development mode..."
	$(GOCMD) run $(MAIN_FILE)

# Display the help menu
help:
	@echo "Available make commands:"
	@echo "  make build          - Build the project for the current host OS"
	@echo "  make build-linux    - Build the project for Linux (amd64)"
	@echo "  make build-windows  - Build the project for Windows (.exe)"
	@echo "  make build-mac      - Build the project for macOS (Apple Silicon)"
	@echo "  make container      - Build the OCI image locally using Podman"
	@echo "  make run            - Start the daemon directly (without compiling a file)"
	@echo "  make test           - Run all unit tests"
	@echo "  make clean          - Remove the build directory"