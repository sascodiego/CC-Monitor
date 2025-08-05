//go:build !ebpf
// +build !ebpf

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-019
 * CONTEXT:   eBPF stubs for environments without clang/eBPF support
 * REASON:    Need compilation support for CI/development environments without eBPF toolchain
 * CHANGE:    Initial implementation.
 * PREVENTION:Mark clearly as stubs, ensure LoadPrograms fails gracefully with informative error
 * RISK:      Low - Stubs only used in development/testing environments without eBPF support
 */

package ebpf

import (
	"fmt"
)

// claudeMonitorObjects represents the eBPF objects (stub version)
type claudeMonitorObjects struct{}

func (objs *claudeMonitorObjects) Close() error {
	return nil
}

// loadClaudeMonitor loads the eBPF spec (stub version)
func loadClaudeMonitor() (interface{}, error) {
	return nil, fmt.Errorf("eBPF support not compiled in - build with 'go build -tags ebpf' and ensure clang is available")
}

// Stub implementations for when eBPF is not available
func (m *Manager) loadProgramsInternal() error {
	return fmt.Errorf("eBPF programs not available - this build was compiled without eBPF support. Please ensure:\n" +
		"1. clang compiler is installed\n" +
		"2. Build with: make generate-ebpf && go build -tags ebpf\n" +
		"3. Run as root on Linux kernel 5.4+")
}