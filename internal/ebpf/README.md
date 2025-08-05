# eBPF Implementation for Claude Monitor

This package provides kernel-level monitoring of Claude CLI processes using eBPF (Extended Berkeley Packet Filter) programs.

## Overview

The eBPF implementation captures system events at the kernel level with minimal overhead:

- **execve syscalls**: Detects when Claude processes are launched
- **connect syscalls**: Monitors connections to Anthropic API endpoints  
- **process exits**: Tracks when Claude processes terminate

## Architecture

### eBPF Programs (`claude_monitor.c`)

- **trace_execve**: Captures process execution events for Claude binaries
- **trace_connect**: Monitors network connections from Claude processes
- **trace_exit**: Handles process termination cleanup

### Go Integration (`manager.go`)

- **Manager**: Implements the `arch.EBPFManager` interface
- **Event Processing**: Ring buffer-based communication from kernel to userspace
- **Filtering**: Kernel and userspace filtering for relevant events only

## Requirements

### System Requirements

- **Linux Kernel**: 5.4+ with eBPF support
- **Root Privileges**: Required for loading eBPF programs
- **Dependencies**: clang, kernel headers, libbpf

### WSL Requirements

For Windows Subsystem for Linux:
- WSL2 (WSL1 does not support eBPF)
- Custom kernel with eBPF support enabled

## Installation

### Install Dependencies

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install clang llvm libbpf-dev linux-headers-$(uname -r)

# Red Hat/CentOS
sudo yum install clang llvm libbpf-devel kernel-devel
```

### Build eBPF Programs

```bash
# Generate Go code from eBPF C programs
make generate-ebpf

# Build the complete project
make build
```

## Usage

The eBPF manager is automatically initialized when the daemon starts:

```bash
# Start daemon (requires root)
sudo ./bin/claude-daemon

# Check status
./bin/claude-monitor status
```

## Event Flow

1. **Kernel Events**: eBPF programs capture syscalls in kernel space
2. **Ring Buffer**: Events are efficiently transferred to userspace
3. **Filtering**: Multiple levels of filtering reduce noise:
   - Kernel-level: Only Claude processes
   - Userspace: Only Anthropic API connections
4. **Processing**: Validated events are sent to the daemon core

## Monitoring & Debugging

### Performance Statistics

```bash
# View eBPF statistics
./bin/claude-monitor status --verbose
```

### Troubleshooting

#### Permission Denied
- Ensure running as root: `sudo ./bin/claude-daemon`
- Check kernel eBPF support: `zgrep CONFIG_BPF /proc/config.gz`

#### Program Load Failures
- Verify kernel version: `uname -r` (need 5.4+)
- Check kernel headers: `ls /lib/modules/$(uname -r)/build`
- Validate clang version: `clang --version` (need 10+)

#### WSL-Specific Issues
- Confirm WSL2: `wsl --status`
- Custom kernel may be needed for eBPF support

### Debugging eBPF Programs

```bash
# View loaded programs
sudo bpftool prog list

# View maps
sudo bpftool map list

# Trace events (for debugging)
sudo bpftool prog tracelog
```

## Security Considerations

- **Root Privileges**: Required for kernel access
- **Resource Limits**: Programs have built-in limits to prevent DoS
- **Data Filtering**: No sensitive data is captured or logged
- **Memory Safety**: All kernel memory access is bounds-checked

## Performance

### Overhead Characteristics

- **Kernel Overhead**: < 1% CPU impact under normal load
- **Memory Usage**: ~256KB ring buffer + ~8KB maps
- **Event Rate**: Can handle 10k+ events/second

### Optimization Features

- **Kernel Filtering**: Process filtering in kernel space
- **Ring Buffer**: Zero-copy event communication
- **Batch Processing**: Events processed in batches
- **Efficient Parsing**: Minimal allocations in hot path

## Development

### Modifying eBPF Programs

1. Edit `claude_monitor.c`
2. Run `make generate-ebpf` to regenerate Go bindings
3. Test with `go test ./internal/ebpf/...`
4. Build and test: `make build && sudo ./bin/claude-daemon`

### Testing

```bash
# Unit tests (no root required)
go test ./internal/ebpf/...

# Integration tests (requires root)
sudo go test -tags integration ./internal/ebpf/...

# Benchmarks
go test -bench=. ./internal/ebpf/...
```

### Code Generation

The build system uses `bpf2go` to generate Go bindings:

```bash
# Manual generation
cd internal/ebpf
go generate

# Or use Makefile
make generate-ebpf
```

Generated files (gitignored):
- `claudemonitor_bpfeb.go` - Big endian bindings
- `claudemonitor_bpfel.go` - Little endian bindings  
- `claudemonitor_bpfeb.o` - Big endian object
- `claudemonitor_bpfel.o` - Little endian object

## Implementation Notes

### Event Structure

Events are defined to match between C and Go:

```c
// C structure
struct claude_event {
    __u64 timestamp;
    __u32 pid;
    // ... other fields
} __attribute__((packed));
```

```go
// Go structure
type claudeEvent struct {
    Timestamp uint64
    PID       uint32
    // ... other fields  
}
```

### IP Address Filtering

Anthropic API endpoints are filtered by IP ranges:
- CloudFront ranges commonly used by Anthropic
- Configurable in `anthropicIPRanges` variable
- Efficient CIDR matching using `net.IPNet`

### Resource Management

- Programs auto-detach on daemon shutdown
- Maps are automatically cleaned up
- Ring buffer has overflow protection
- PID tracking prevents memory leaks

## References

- [eBPF Documentation](https://ebpf.io/)
- [Cilium eBPF Library](https://github.com/cilium/ebpf)
- [BPF Type Format (BTF)](https://www.kernel.org/doc/html/latest/bpf/btf.html)
- [Linux eBPF Syscalls](https://man7.org/linux/man-pages/man2/bpf.2.html)