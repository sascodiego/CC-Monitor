//go:build ebpf
// +build ebpf

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-020
 * CONTEXT:   eBPF implementation for environments with clang/eBPF support
 * REASON:    Need actual eBPF implementation separate from stubs for production use
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure this file only compiles when eBPF build tag is used
 * RISK:      High - eBPF programs require careful resource management and error handling
 */

package ebpf

import (
	"fmt"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

// loadProgramsInternal implements the actual eBPF program loading
func (m *Manager) loadProgramsInternal() error {
	// Load eBPF spec and objects
	spec, err := loadClaudeMonitor()
	if err != nil {
		return fmt.Errorf("failed to load eBPF spec: %w", err)
	}
	
	objs := &claudeMonitorObjects{}
	if err := spec.LoadAndAssign(objs, nil); err != nil {
		return fmt.Errorf("failed to load eBPF objects: %w", err)
	}
	m.objs = objs
	
	// Attach tracepoints
	var links []link.Link
	
	// Attach execve tracepoint
	execveLink, err := link.Tracepoint(link.TracepointOptions{
		Group:   "syscalls",
		Name:    "sys_enter_execve",
		Program: objs.TraceExecve,
	})
	if err != nil {
		objs.Close()
		return fmt.Errorf("failed to attach execve tracepoint: %w", err)
	}
	links = append(links, execveLink)
	
	// Attach connect tracepoint
	connectLink, err := link.Tracepoint(link.TracepointOptions{
		Group:   "syscalls",
		Name:    "sys_enter_connect",
		Program: objs.TraceConnect,
	})
	if err != nil {
		execveLink.Close()
		objs.Close()
		return fmt.Errorf("failed to attach connect tracepoint: %w", err)
	}
	links = append(links, connectLink)
	
	// Attach exit tracepoint
	exitLink, err := link.Tracepoint(link.TracepointOptions{
		Group:   "sched",
		Name:    "sched_process_exit",
		Program: objs.TraceExit,
	})
	if err != nil {
		execveLink.Close()
		connectLink.Close()
		objs.Close()
		return fmt.Errorf("failed to attach exit tracepoint: %w", err)
	}
	links = append(links, exitLink)
	
	// Attach write tracepoint for HTTP request monitoring
	writeLink, err := link.Tracepoint(link.TracepointOptions{
		Group:   "syscalls",
		Name:    "sys_enter_write",
		Program: objs.TraceWrite,
	})
	if err != nil {
		execveLink.Close()
		connectLink.Close()
		exitLink.Close()
		objs.Close()
		return fmt.Errorf("failed to attach write tracepoint: %w", err)
	}
	links = append(links, writeLink)
	
	m.links = links
	
	// Create ring buffer reader
	reader, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		m.cleanup()
		return fmt.Errorf("failed to create ring buffer reader: %w", err)
	}
	m.reader = reader
	
	m.logger.Info("eBPF programs loaded successfully", "programs", len(links))
	return nil
}