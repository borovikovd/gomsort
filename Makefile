# Variables
BINARY_NAME := gomsort
GO_VERSION := 1.24.5
GOLANGCI_LINT_VERSION := v1.64.8

# Add Go bin to PATH
GOPATH := $(shell go env GOPATH)
export PATH := $(GOPATH)/bin:$(PATH)

# Build targets
.PHONY: build
build:
	go build -o $(BINARY_NAME) .

.PHONY: build-all
build-all:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe .

# Test targets
.PHONY: test
test:
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-integration
test-integration: build
	./$(BINARY_NAME) -n testdata/

# Lint targets
.PHONY: lint
lint:
	PATH="$(GOPATH)/bin:$(PATH)" golangci-lint run

.PHONY: lint-fix
lint-fix:
	PATH="$(GOPATH)/bin:$(PATH)" golangci-lint run --fix

.PHONY: install-golangci-lint
install-golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# Format targets
.PHONY: fmt
fmt:
	gofmt -s -w .
	PATH="$(GOPATH)/bin:$(PATH)" goimports -w .

.PHONY: fmt-check
fmt-check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Go files are not formatted. Please run 'make fmt'"; \
		exit 1; \
	fi

# Clean targets
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install targets
.PHONY: install
install:
	go install .

# Development targets
.PHONY: dev
dev: fmt lint test

.PHONY: ci
ci: fmt-check lint test

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build              - Build the binary"
	@echo "  build-all          - Build binaries for all platforms"
	@echo "  test               - Run all tests"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-integration   - Run integration tests"
	@echo "  lint               - Run golangci-lint"
	@echo "  lint-fix           - Run golangci-lint with --fix"
	@echo "  install-golangci-lint - Install golangci-lint"
	@echo "  fmt                - Format Go code"
	@echo "  fmt-check          - Check if Go code is formatted"
	@echo "  clean              - Clean build artifacts"
	@echo "  install            - Install the binary"
	@echo "  dev                - Development workflow (fmt + lint + test)"
	@echo "  ci                 - CI workflow (fmt-check + lint + test)"
	@echo "  help               - Show this help message"
	@echo ""
	@echo "Note: gomsort processes directories recursively by default (like go fmt)"