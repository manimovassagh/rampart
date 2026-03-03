APP_NAME := rampart
BUILD_DIR := bin
GO_FILES := $(shell find . -name '*.go' -not -path './client/*' -not -path './vendor/*')

# Version info (injected at build time later)
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

.PHONY: all build run clean test lint fmt vet check dev-setup help

## help: show this help message
help:
	@echo "🏰 Rampart — Development Commands"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## build: compile the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/rampart

## run: build and run the server
run: build
	./$(BUILD_DIR)/$(APP_NAME)

## test: run all tests with race detector
test:
	go test -race -count=1 ./...

## test-cover: run tests with coverage report
test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: format all Go files
fmt:
	gofmt -s -w $(GO_FILES)

## vet: run go vet
vet:
	go vet ./...

## check: run fmt + vet + lint + test (full quality gate)
check: fmt vet lint test
	@echo "✅ All checks passed"

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

## dev-setup: install development tools
dev-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "✅ Dev tools installed"

## vuln: check for known vulnerabilities
vuln:
	govulncheck ./...
