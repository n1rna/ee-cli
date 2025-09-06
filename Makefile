BINARY_NAME=ee
BUILD_DIR=build
COVERAGE_DIR=coverage
VERSION?=0.1.0
GIT_COMMIT=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildTime=${BUILD_TIME}"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Files to watch for changes
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

.PHONY: all build clean test coverage deps fmt lint vet help install uninstall dev

all: clean build test ## Run clean, build, and test

build: ## Build the binary
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) ${LDFLAGS} -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ee

clean: ## Clean build directory
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	$(GOCLEAN)

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

deps: ## Download and verify dependencies
	$(GOMOD) download
	$(GOMOD) verify

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) ./...

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint is not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

install: build ## Install binary to $GOPATH/bin
	@echo "Installing..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

uninstall: ## Remove binary from $GOPATH/bin
	@echo "Uninstalling..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)

dev: ## Run the application in development mode
	@$(GORUN) ${LDFLAGS} ./cmd/ee

# Cross compilation targets
.PHONY: build-linux build-windows build-darwin
build-linux: ## Build for Linux
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/ee

build-windows: ## Build for Windows
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/ee

build-darwin: ## Build for macOS
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/ee

.PHONY: build-all
build-all: build-linux build-windows build-darwin ## Build for all platforms

# Release management
.PHONY: release
release: ## Create a new release (usage: make release VERSION=0.2.0)
	@if [ "$(VERSION)" = "" ]; then \
		echo "Error: VERSION is required. Use: make release VERSION=0.2.0"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	@-git tag -d v$(VERSION)
	@-git push origin :refs/tags/v$(VERSION)
	git tag -a v$(VERSION) -m "Release $(VERSION)"
	git push origin v$(VERSION)

# Cleanup old builds and temporary files
.PHONY: distclean
distclean: clean ## Remove all generated files
	@rm -rf vendor/
	@rm -f go.sum

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Check for required tools
.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@echo "Checking required tools..."
	@which golangci-lint >/dev/null || (echo "golangci-lint is not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)

# Run all verification steps
.PHONY: verify
verify: fmt vet lint test ## Run all verification steps