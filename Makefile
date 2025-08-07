# Claude Monitor - Single Self-Installing Binary
# Zero-dependency build system for work hour tracking

.PHONY: build clean install test help daemon monitor status config

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Default target - build single binary
build:
	@echo "🚀 Building Claude Monitor v$(VERSION)..."
	@echo "   Build Time: $(BUILD_TIME)"
	@echo "   Git Commit: $(GIT_COMMIT)"
	cd cmd/claude-monitor && go build $(LDFLAGS) -o ../../claude-monitor .
	@echo "✅ Single binary built: ./claude-monitor"
	@echo "📏 Binary size: $(shell du -h claude-monitor 2>/dev/null | cut -f1 || echo 'Unknown')"

# Cross-compilation for release
build-all: build-linux build-darwin build-windows
	@echo "🎯 All platform binaries built:"
	@ls -la claude-monitor-* 2>/dev/null || echo "No binaries found"

build-linux:
	@echo "🐧 Building for Linux amd64..."
	cd cmd/claude-monitor && GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ../../claude-monitor-linux-amd64 .

build-darwin:
	@echo "🍎 Building for macOS..."
	cd cmd/claude-monitor && GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o ../../claude-monitor-darwin-amd64 .
	cd cmd/claude-monitor && GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o ../../claude-monitor-darwin-arm64 .

build-windows:
	@echo "🪟 Building for Windows amd64..."
	cd cmd/claude-monitor && GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o ../../claude-monitor-windows-amd64.exe .

# Installation and setup
install: build
	@echo "📦 Self-installing Claude Monitor..."
	./claude-monitor install
	@echo "✅ Installation complete! Next steps:"
	@echo "   1. Start daemon: ./claude-monitor daemon &"
	@echo "   2. Configure Claude Code: ./claude-monitor config"
	@echo "   3. Check status: ./claude-monitor status"

# Development and testing targets
daemon: build
	@echo "🔄 Starting Claude Monitor daemon..."
	./claude-monitor daemon

status: build
	@echo "🔍 Checking Claude Monitor status..."
	./claude-monitor status

config: build
	@echo "🔧 Showing Claude Code configuration..."
	./claude-monitor config

today: build
	@echo "📊 Today's work summary..."
	./claude-monitor today

# Complete development workflow
dev-setup:
	@echo "🛠️ Setting up development environment..."
	go mod tidy
	go mod download
	@echo "✅ Development environment ready"

# Testing
test:
	@echo "🧪 Running tests..."
	go test -v ./...

test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -cover ./...

test-integration: build
	@echo "🔗 Running integration tests..."
	@if [ -f ./tests/integration-test.sh ]; then ./tests/integration-test.sh; else echo "No integration tests found"; fi

# Code quality
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "⚠️ golangci-lint not available, skipping..."; fi

fmt:
	@echo "✨ Formatting code..."
	go fmt ./...

vet:
	@echo "🔎 Running go vet..."
	go vet ./...

# Cleanup
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f claude-monitor
	rm -f claude-monitor-*
	go clean ./...
	@echo "✅ Clean complete"

# Release workflow
release-prep: clean fmt vet test build-all
	@echo "🎉 Release preparation complete!"
	@echo "📦 Available binaries:"
	@ls -la claude-monitor-* 2>/dev/null | awk '{print "   " $$9 " (" $$5 " bytes)"}' || echo "No binaries found"

# Quick development iteration
quick: clean build daemon

# Demo workflow
demo: build install
	@echo "🎬 Starting demo workflow..."
	@echo "1. Installing system..."
	@sleep 2
	@./claude-monitor daemon > /dev/null 2>&1 &
	@echo "2. Daemon started in background"
	@sleep 2
	@echo "3. System status:"
	@./claude-monitor status || echo "Status check failed (daemon may still be starting)"
	@echo "4. Configuration guide:"
	@./claude-monitor config | head -20
	@echo ""
	@echo "✅ Demo complete! Claude Monitor is ready to use."

# Help system
help:
	@echo "🎯 Claude Monitor - Single Binary Build System"
	@echo ""
	@echo "🚀 Quick Start:"
	@echo "  make install              # Build and self-install"
	@echo "  make daemon               # Start background service"  
	@echo "  make today                # Show today's work report"
	@echo ""
	@echo "🛠️ Build Targets:"
	@echo "  build                     # Build single binary for current platform"
	@echo "  build-all                 # Cross-compile for all platforms"  
	@echo "  clean                     # Remove build artifacts"
	@echo ""
	@echo "📊 Operations:"
	@echo "  status                    # Check system health"
	@echo "  config                    # Show Claude Code setup guide"
	@echo "  today                     # Display today's work summary"
	@echo ""
	@echo "🧪 Development:"
	@echo "  dev-setup                 # Initialize development environment"
	@echo "  test                      # Run unit tests"
	@echo "  test-coverage             # Run tests with coverage"
	@echo "  fmt                       # Format all source code"
	@echo "  lint                      # Run code linter"
	@echo ""
	@echo "🎁 Release:"
	@echo "  release-prep              # Prepare multi-platform release"
	@echo "  demo                      # Full demo installation"
	@echo ""
	@echo "💡 Example Workflow:"
	@echo "  make build                # Build the binary"
	@echo "  ./claude-monitor install  # Self-install to system"
	@echo "  ./claude-monitor daemon & # Start background daemon"
	@echo "  ./claude-monitor config   # Get Claude Code setup instructions"
	@echo "  ./claude-monitor today    # Check today's work activity"
	@echo ""
	@echo "📋 Binary Features:"
	@echo "  • Zero external dependencies"
	@echo "  • Self-installing with embedded assets"
	@echo "  • Cross-platform support (Linux, macOS, Windows)"
	@echo "  • Sub-10ms hook execution"
	@echo "  • Beautiful CLI with colors and tables"
	@echo "  • AI-optimized Claude Code integration"

# Advanced targets
check-deps:
	@echo "🔍 Checking dependencies..."
	go list -m all
	@echo ""
	@echo "Direct dependencies:"
	@go list -m all | grep -v "indirect" | head -10

size-analysis: build
	@echo "📏 Binary size analysis..."
	@echo "Total binary size: $(shell du -h claude-monitor 2>/dev/null | cut -f1 || echo 'Unknown')"
	@echo "Embedded assets:"
	@find cmd/claude-monitor/assets -type f -exec du -h {} \; 2>/dev/null | sort || echo "No assets found"

benchmark: build
	@echo "⚡ Performance benchmarks..."
	@echo "Hook execution time test:"
	@time ./claude-monitor hook --debug 2>/dev/null || echo "Hook test complete"

# Legacy compatibility (maintain old build system interface)
all: build
build-daemon: build
build-hook: build
run-daemon: daemon
run-hook: 
	@echo "Testing hook..."
	./claude-monitor hook --debug

# Version information
version: build
	./claude-monitor version