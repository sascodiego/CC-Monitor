# Claude Monitor Build System
# Build system for Claude Monitor with proper Go build configuration

.PHONY: all build clean test daemon cli install deps lint generate-ebpf

# Build variables
BINARY_DIR=bin
DAEMON_BINARY=$(BINARY_DIR)/claude-daemon
CLI_BINARY=$(BINARY_DIR)/claude-monitor
GO_VERSION=1.21

# Build flags (CGO enabled for SQLite)
BUILD_FLAGS=-ldflags="-s -w"
CGO_ENABLED=1

all: build

# Create binary directory
$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

# Build daemon
daemon: $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -o $(DAEMON_BINARY) ./cmd/claude-daemon

# Build simple daemon for testing
daemon-simple: $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -o $(BINARY_DIR)/claude-daemon-simple ./cmd/claude-daemon-simple

# Build enhanced daemon with activity monitoring
daemon-enhanced: $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -o $(BINARY_DIR)/claude-daemon-enhanced ./cmd/claude-daemon-enhanced

# Build CLI
cli: $(BINARY_DIR) 
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -o $(CLI_BINARY) ./cmd/claude-monitor

# Build daemon with eBPF support
daemon-ebpf: $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build -tags ebpf $(BUILD_FLAGS) -o $(DAEMON_BINARY) ./cmd/claude-daemon

# Build CLI with eBPF support
cli-ebpf: $(BINARY_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build -tags ebpf $(BUILD_FLAGS) -o $(CLI_BINARY) ./cmd/claude-monitor

# Generate eBPF code
generate-ebpf:
	@echo "Generating eBPF code with bpf2go..."
	cd internal/ebpf && go generate
	@echo "eBPF code generation complete"

# Build both binaries (without eBPF by default)
build: daemon cli

# Build with eBPF support (requires clang and kernel headers)
build-ebpf: generate-ebpf daemon-ebpf cli-ebpf

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint code
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...

# Check Go version
check-go-version:
	@go version | grep -q "go$(GO_VERSION)" || (echo "Go $(GO_VERSION) is required" && exit 1)

# Install binaries to system
install: build
	sudo cp $(DAEMON_BINARY) /usr/local/bin/
	sudo cp $(CLI_BINARY) /usr/local/bin/
	sudo chmod +x /usr/local/bin/claude-daemon
	sudo chmod +x /usr/local/bin/claude-monitor

# Clean build artifacts
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html
	rm -f internal/ebpf/claudemonitor_bpfeb.go internal/ebpf/claudemonitor_bpfel.go internal/ebpf/claudemonitor_bpfeb.o internal/ebpf/claudemonitor_bpfel.o

# Development helpers
dev-daemon: daemon
	sudo ./$(DAEMON_BINARY)

dev-status: cli
	./$(CLI_BINARY) status

# Create systemd service file
systemd-service:
	@echo "[Unit]" > claude-monitor.service
	@echo "Description=Claude Monitor Daemon" >> claude-monitor.service
	@echo "After=network.target" >> claude-monitor.service
	@echo "" >> claude-monitor.service
	@echo "[Service]" >> claude-monitor.service
	@echo "Type=forking" >> claude-monitor.service
	@echo "ExecStart=/usr/local/bin/claude-daemon" >> claude-monitor.service
	@echo "PIDFile=/var/run/claude-monitor.pid" >> claude-monitor.service
	@echo "User=root" >> claude-monitor.service
	@echo "Group=root" >> claude-monitor.service
	@echo "" >> claude-monitor.service
	@echo "[Install]" >> claude-monitor.service
	@echo "WantedBy=multi-user.target" >> claude-monitor.service
	@echo "Systemd service file created: claude-monitor.service"

# Show help
help:
	@echo "Claude Monitor Build System"
	@echo "=========================="
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build both daemon and CLI binaries (without eBPF)"
	@echo "  build-ebpf    - Build with eBPF support (requires clang)"
	@echo "  daemon        - Build daemon binary only"
	@echo "  cli           - Build CLI binary only"
	@echo "  daemon-ebpf   - Build daemon with eBPF support"
	@echo "  cli-ebpf      - Build CLI with eBPF support"
	@echo "  generate-ebpf - Generate eBPF Go code from C programs"
	@echo "  deps          - Install Go dependencies"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  install       - Install binaries to system"
	@echo "  clean         - Clean build artifacts"
	@echo "  systemd-service - Create systemd service file"
	@echo "  help          - Show this help"
	@echo ""
	@echo "Development targets:"
	@echo "  dev-daemon    - Build and run daemon (requires sudo)"
	@echo "  dev-status    - Build CLI and show status"