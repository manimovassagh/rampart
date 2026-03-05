APP_NAME := rampart
BUILD_DIR := bin
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html
COVERAGE_THRESHOLD := 50

# Version info (injected at build time)
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: all build run clean test test-cover test-threshold lint fmt vet \
        security gosec vuln check dev-setup docker-build css help

## help: show this help message
help:
	@echo "Rampart - Development Commands"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## css: rebuild Tailwind CSS from templates
css:
	tailwindcss --input internal/handler/static/input.css --output internal/handler/static/admin.css --minify

## build: compile the binary (rebuilds CSS first)
build: css
	@mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/rampart

## run: build and run the server
run: build
	./$(BUILD_DIR)/$(APP_NAME)

## test: run all tests with race detector
test:
	go test -race -count=1 ./...

## test-cover: run tests with coverage report
test-cover:
	go test -race -count=1 -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"

## test-threshold: run tests and enforce coverage threshold
test-threshold: test-cover
	@total=$$(go tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $${total}%"; \
	if [ $$(echo "$${total} < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "FAIL: Coverage $${total}% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi; \
	echo "OK: Coverage $${total}% meets threshold $(COVERAGE_THRESHOLD)%"

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: format all Go files
fmt:
	gofmt -s -w $(GO_FILES)

## fmt-check: check formatting without modifying files
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; \
		echo "$$unformatted"; \
		echo ""; \
		echo "Run 'make fmt' to fix."; \
		exit 1; \
	fi

## vet: run go vet
vet:
	go vet ./...

## gosec: run gosec security scanner
gosec:
	gosec -severity medium -confidence medium ./...

## vuln: check for known vulnerabilities
vuln:
	govulncheck ./...

## security: run all security checks (vuln + gosec)
security: vuln gosec

## check: run all quality gates (matches CI pipeline)
check: fmt-check vet lint test-threshold security
	@echo "All checks passed"

## docker-build: build Docker image locally
docker-build:
	docker build -t $(APP_NAME):$(VERSION) .

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR) $(COVERAGE_FILE) $(COVERAGE_HTML)

## dev-setup: install development tools
dev-setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Dev tools installed"

## tidy: run go mod tidy
tidy:
	go mod tidy
