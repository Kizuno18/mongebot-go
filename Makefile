.PHONY: build run dev test clean docker-up docker-down lint release-all

# Project
BINARY=mongebot
BUILD_DIR=bin
DIST_DIR=dist
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

# ============ Development ============

build:
	@echo "Building $(BINARY)..."
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/mongebot/

build-cli:
	@echo "Building $(BINARY)-cli..."
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-cli ./cmd/cli/

build-all: build build-cli

run: build
	@./$(BUILD_DIR)/$(BINARY)

dev:
	@go run ./cmd/mongebot/

dev-cli:
	@go run ./cmd/cli/ --channel $(CHANNEL) --workers $(or $(WORKERS),50)

# ============ Testing ============

test:
	@go test -v -race -count=1 ./...

test-short:
	@go test -short ./...

test-cover:
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	@go test -bench=. -benchmem ./...

lint:
	@golangci-lint run ./...

vet:
	@go vet ./...

# ============ Cross-Compilation ============

PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

release-all: clean
	@mkdir -p $(DIST_DIR)
	@$(foreach platform,$(PLATFORMS),\
		$(eval GOOS=$(firstword $(subst /, ,$(platform))))\
		$(eval GOARCH=$(lastword $(subst /, ,$(platform))))\
		$(eval EXT=$(if $(filter windows,$(GOOS)),.exe,))\
		echo "Building $(GOOS)/$(GOARCH)..." && \
		GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 \
		go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-$(GOOS)-$(GOARCH)$(EXT) ./cmd/mongebot/ && \
		GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 \
		go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-cli-$(GOOS)-$(GOARCH)$(EXT) ./cmd/cli/ && \
	) true
	@echo "Release builds complete: $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/

release-linux:
	@mkdir -p $(DIST_DIR)
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-linux-amd64 ./cmd/mongebot/
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-linux-arm64 ./cmd/mongebot/

release-darwin:
	@mkdir -p $(DIST_DIR)
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-amd64 ./cmd/mongebot/
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-arm64 ./cmd/mongebot/

release-windows:
	@mkdir -p $(DIST_DIR)
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe ./cmd/mongebot/

# ============ Docker ============

docker-up:
	@docker compose up -d --build

docker-down:
	@docker compose down

docker-logs:
	@docker compose logs -f

docker-build:
	@docker build -f docker/Dockerfile.backend -t mongebot:$(VERSION) .

# ============ Frontend ============

frontend-install:
	@cd frontend && npm install

frontend-dev:
	@cd frontend && npm run dev

frontend-build:
	@cd frontend && npm run build

frontend-tauri-dev:
	@cd frontend && npm run tauri dev

frontend-tauri-build:
	@cd frontend && npm run tauri build

# ============ Full Stack ============

dev-all:
	@make -j2 dev frontend-dev

install: frontend-install
	@go mod download
	@echo "All dependencies installed."

# ============ Cleanup ============

clean:
	@rm -rf $(BUILD_DIR) $(DIST_DIR) coverage.out coverage.html
	@echo "Cleaned build artifacts."

# ============ Info ============

info:
	@echo "MongeBot $(VERSION)"
	@echo "Go:    $(shell go version 2>/dev/null || echo 'not installed')"
	@echo "Node:  $(shell node --version 2>/dev/null || echo 'not installed')"
	@echo "Rust:  $(shell rustc --version 2>/dev/null || echo 'not installed')"
	@echo ""
	@echo "Files: $(shell find . -name '*.go' -not -path './.git/*' | wc -l) Go files"
	@echo "Tests: $(shell find . -name '*_test.go' -exec grep -c 'func Test' {} + 2>/dev/null | awk -F: '{sum+=$$2} END {print sum}') test cases"

help:
	@echo "MongeBot Makefile"
	@echo ""
	@echo "Development:"
	@echo "  make build        Build main binary"
	@echo "  make build-all    Build main + CLI binaries"
	@echo "  make dev          Run with hot reload"
	@echo "  make dev-all      Run backend + frontend"
	@echo "  make install      Install all dependencies"
	@echo ""
	@echo "Testing:"
	@echo "  make test         Run all tests"
	@echo "  make test-cover   Run tests with coverage report"
	@echo "  make lint         Run linter"
	@echo ""
	@echo "Release:"
	@echo "  make release-all  Cross-compile for all platforms"
	@echo "  make release-linux/darwin/windows"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up    Start containers"
	@echo "  make docker-down  Stop containers"
	@echo ""
	@echo "Frontend:"
	@echo "  make frontend-dev         Vite dev server"
	@echo "  make frontend-tauri-dev   Tauri dev mode"
	@echo "  make frontend-tauri-build Tauri production build"
