#!/bin/bash
set -e

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-021
 * CONTEXT:   Build script for eBPF-enabled builds with dependency checking
 * REASON:    Need simple way to build eBPF programs with proper error handling
 * CHANGE:    Initial implementation.
 * PREVENTION:Check for all required dependencies before attempting build
 * RISK:      Low - Build script provides clear error messages for missing dependencies
 */

echo "Claude Monitor eBPF Build Script"
echo "================================"

# Check for required tools
check_tool() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is not installed or not in PATH"
        echo "Please install $1 and try again"
        exit 1
    fi
}

echo "Checking dependencies..."
check_tool "clang"
check_tool "go"

# Check clang version (need 10+)
CLANG_VERSION=$(clang --version | head -n1 | grep -o '[0-9]\+' | head -n1)
if [ "$CLANG_VERSION" -lt 10 ]; then
    echo "Error: clang version $CLANG_VERSION is too old"
    echo "Please install clang 10 or newer"
    exit 1
fi

echo "✓ clang version $CLANG_VERSION found"

# Check Go version  
GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | cut -c3-)
echo "✓ Go version $GO_VERSION found"

# Check for Linux kernel (eBPF requirement)
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "Warning: eBPF requires Linux. This build may not work on $OSTYPE"
fi

# Check if running in WSL
if grep -qi microsoft /proc/version 2>/dev/null; then
    echo "✓ WSL environment detected"
    echo "Note: Ensure WSL2 is being used for eBPF support"
fi

echo
echo "Building eBPF programs..."

# Change to project root
cd "$(dirname "$0")/.."

# Generate eBPF code
echo "Generating eBPF Go bindings..."
make generate-ebpf || {
    echo "Error: eBPF code generation failed"
    echo "Make sure clang is properly installed and in PATH"
    exit 1
}

# Build with eBPF support
echo "Building Go programs with eBPF support..."
CGO_ENABLED=1 go build -tags ebpf -ldflags="-s -w" -o bin/claude-daemon ./cmd/claude-daemon || {
    echo "Error: Daemon build failed"
    exit 1
}

CGO_ENABLED=1 go build -tags ebpf -ldflags="-s -w" -o bin/claude-monitor ./cmd/claude-monitor || {
    echo "Error: CLI build failed"  
    exit 1
}

echo
echo "✅ Build completed successfully!"
echo
echo "Binaries created:"
echo "  - bin/claude-daemon (requires root to run)"
echo "  - bin/claude-monitor"
echo
echo "To run the daemon:"
echo "  sudo ./bin/claude-daemon"
echo
echo "To check status:"
echo "  ./bin/claude-monitor status"