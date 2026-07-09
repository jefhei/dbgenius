.PHONY: build run test lint clean help

# Binary name
BINARY_NAME=dbgenius
BUILD_DIR=./bin

# Go configuration
GO=go
GOFLAGS=-ldflags="-s -w"
CGO_ENABLED=0

# Default target
help:
	@echo "dbgenius — Local-first AI Database Browser"
	@echo ""
	@echo "Usage:"
	@echo "  make build     Build the binary"
	@echo "  make run       Build and run the binary"
	@echo "  make test      Run tests"
	@echo "  make lint      Run golangci-lint"
	@echo "  make clean     Remove build artifacts"
	@echo "  make help      Show this help"

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dbgenius/.
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

run: build
	@./$(BUILD_DIR)/$(BINARY_NAME)

test:
	$(GO) test -v -race -count=1 ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed — run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		$(GO) vet ./...; \
	fi

clean:
	@rm -rf $(BUILD_DIR)/
	@rm -f coverage.* *.out *.prof *.cov
	@echo "Cleaned build artifacts"
