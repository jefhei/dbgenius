.PHONY: build run test lint clean help all build-all \
        build-linux build-linux-arm build-darwin build-darwin-arm

# Binary name
BINARY_NAME=dbgenius
BUILD_DIR=./bin

# Go configuration
GO=go
CGO_ENABLED=0
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

# Version from git tag or commit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Default target
all: build

help:
	@echo "dbgenius — Local-first AI Database Browser"
	@echo ""
	@echo "Usage:"
	@echo "  make build             Build for current platform"
	@echo "  make run               Build and run the binary"
	@echo "  make test              Run tests"
	@echo "  make lint              Run golangci-lint or go vet"
	@echo "  make clean             Remove build artifacts"
	@echo "  make build-all         Build for all platforms"
	@echo "  make build-linux       Build for linux/amd64"
	@echo "  make build-linux-arm   Build for linux/arm64"
	@echo "  make build-darwin      Build for darwin/amd64"
	@echo "  make build-darwin-arm  Build for darwin/arm64"
	@echo "  make release           Create release archives"
	@echo "  make help              Show this help"

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dbgenius/.
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

run: build
	@./$(BUILD_DIR)/$(BINARY_NAME)

test:
	$(GO) test -v -race -count=1 ./...

test-short:
	$(GO) test -short -count=1 ./...

test-coverage:
	$(GO) test -coverprofile=coverage.out -count=1 ./...
	$(GO) tool cover -func=coverage.out | tail -5

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed — running go vet instead"; \
		$(GO) vet ./...; \
	fi

clean:
	@rm -rf $(BUILD_DIR)/
	@rm -f coverage.* *.out *.prof *.cov *.tar.gz *.zip
	@echo "Cleaned build artifacts"

# Cross-platform builds
build-linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/dbgenius/.
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

build-linux-arm:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/dbgenius/.
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

build-darwin:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/dbgenius/.
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"

build-darwin-arm:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/dbgenius/.
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

build-all: build-linux build-linux-arm build-darwin build-darwin-arm
	@echo "All builds complete"

# Release archives
release: build-all
	@cd $(BUILD_DIR) && for f in $(BINARY_NAME)-*; do \
		ext=""; \
		case $$f in *.exe) ext=".exe";; esac; \
		name=$${f%$$ext}; \
		mkdir -p /tmp/dbgenius-release/$$name; \
		cp $$f /tmp/dbgenius-release/$$name/; \
		cp ../README.md ../LICENSE 2>/dev/null || true; \
		cd /tmp/dbgenius-release/$$name && tar czf ../$$name.tar.gz *; \
		cd - > /dev/null; \
		rm -rf /tmp/dbgenius-release/$$name; \
	done
	@mv /tmp/dbgenius-release/*.tar.gz $(BUILD_DIR)/ 2>/dev/null || true
	@rm -rf /tmp/dbgenius-release
	@echo "Release archives in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/*.tar.gz 2>/dev/null || echo "No release archives created"

# Install locally
install:
	$(GO) install ./cmd/dbgenius/.
	@echo "Installed to $$GOPATH/bin/$(BINARY_NAME)"
