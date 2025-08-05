/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-009
 * CONTEXT:   Go eBPF manager implementing the EBPFManager interface with ring buffer processing
 * REASON:    Need production eBPF implementation to replace placeholder for kernel-level monitoring
 * CHANGE:    Initial implementation.
 * PREVENTION:Always cleanup eBPF resources on shutdown, handle attach/detach errors properly
 * RISK:      High - Resource leaks could crash system or exhaust kernel memory
 */

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall -Werror" claudeMonitor ./claude_monitor.c

package ebpf

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"golang.org/x/sys/unix"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/pkg/events"
)

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-010
 * CONTEXT:   Event types and structures matching the eBPF C program
 * REASON:    Need Go representations of eBPF event structures for ring buffer parsing
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep structures synchronized with C definitions, validate sizes
 * RISK:      High - Structure mismatch causes data corruption and parsing errors
 */

const (
	EventExec    = 1
	EventConnect = 2
	EventExit    = 3
	EventHTTPRequest = 4
)

// claudeEvent matches the C structure in claude_monitor.c
type claudeEvent struct {
	Timestamp     uint64
	PID           uint32
	PPID          uint32
	UID           uint32
	EventType     uint32
	TargetAddr    uint32
	TargetPort    uint16
	ExitCode      int32
	Comm          [16]byte
	Path          [256]byte
	// HTTP request data
	HTTPMethod    [8]byte
	HTTPURI       [128]byte
	ContentLength uint32
	SocketFD      uint32
}

// Anthropic API IP ranges for connection filtering
var anthropicIPRanges = []string{
	"52.84.0.0/15",     // CloudFront ranges commonly used by Anthropic
	"54.230.0.0/16",    // CloudFront
	"99.86.0.0/16",     // CloudFront
	"13.32.0.0/15",     // CloudFront
	"204.246.164.0/22", // CloudFront
	"54.192.0.0/12",    // CloudFront
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-011
 * CONTEXT:   EBPFManager implementation with lifecycle management and event processing
 * REASON:    Implement the arch.EBPFManager interface for production kernel monitoring
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure proper resource cleanup, handle concurrent access safely
 * RISK:      High - Improper cleanup causes kernel resource leaks and system instability
 */

type Manager struct {
	objs          *claudeMonitorObjects
	links         []link.Link
	reader        *ringbuf.Reader
	eventCh       chan *events.SystemEvent
	stopCh        chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	running       bool
	logger        arch.Logger
	
	// Statistics counters
	eventsProcessed int64
	droppedEvents   int64
	processingErrors int64
	
	// IP filtering
	anthropicNets []*net.IPNet
}

func NewManager(logger arch.Logger) *Manager {
	manager := &Manager{
		eventCh:     make(chan *events.SystemEvent, 1000),
		stopCh:      make(chan struct{}),
		logger:      logger,
		anthropicNets: make([]*net.IPNet, 0, len(anthropicIPRanges)),
	}
	
	// Parse Anthropic IP ranges for connection filtering
	for _, cidr := range anthropicIPRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Warn("Failed to parse Anthropic IP range", "cidr", cidr, "error", err)
			continue
		}
		manager.anthropicNets = append(manager.anthropicNets, network)
	}
	
	return manager
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-012
 * CONTEXT:   eBPF program loading and kernel attachment
 * REASON:    Need to load compiled eBPF programs and attach to syscall tracepoints
 * CHANGE:    Initial implementation.
 * PREVENTION:Check for root privileges, validate program loading, handle errors gracefully
 * RISK:      Critical - Program loading failures prevent monitoring functionality
 */

func (m *Manager) LoadPrograms() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check for root privileges
	if unix.Geteuid() != 0 {
		return fmt.Errorf("eBPF manager requires root privileges")
	}
	
	// Remove memory limits for eBPF
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("failed to remove memlock limit: %w", err)
	}
	
	// Load programs using internal method (may be stubbed)
	return m.loadProgramsInternal()
}

// loadProgramsInternal is implemented in either ebpf_impl.go or stubs.go
// depending on build tags

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-013
 * CONTEXT:   Ring buffer event processing with filtering and validation
 * REASON:    Need efficient processing of kernel events with proper filtering
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle ring buffer overflow, validate all event data, implement backpressure
 * RISK:      Medium - Ring buffer overflow could cause event loss and monitoring gaps
 */

func (m *Manager) StartEventProcessing(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.running {
		return fmt.Errorf("event processing already running")
	}
	
	if m.reader == nil {
		return fmt.Errorf("eBPF programs not loaded")
	}
	
	m.running = true
	m.wg.Add(1)
	
	go m.processEvents(ctx)
	
	m.logger.Info("eBPF event processing started")
	return nil
}

func (m *Manager) processEvents(ctx context.Context) {
	defer m.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			m.logger.Debug("Context cancelled, stopping event processing")
			return
		case <-m.stopCh:
			m.logger.Debug("Stop signal received, stopping event processing")
			return
		default:
			// Read event from ring buffer with timeout
			record, err := m.reader.Read()
			if err != nil {
				if err == ringbuf.ErrClosed {
					m.logger.Info("Ring buffer closed, stopping event processing")
					return
				}
				atomic.AddInt64(&m.processingErrors, 1)
				m.logger.Warn("Ring buffer read error", "error", err)
				continue
			}
			
			// Parse and validate event
			event := m.parseEvent(record.RawSample)
			if event == nil {
				atomic.AddInt64(&m.droppedEvents, 1)
				continue
			}
			
			// Apply filtering
			if !m.isRelevantEvent(event) {
				atomic.AddInt64(&m.droppedEvents, 1)
				continue
			}
			
			// Send to event channel with non-blocking write
			select {
			case m.eventCh <- event:
				atomic.AddInt64(&m.eventsProcessed, 1)
			default:
				atomic.AddInt64(&m.droppedEvents, 1)
				m.logger.Warn("Event channel full, dropping event")
			}
		}
	}
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-014
 * CONTEXT:   Zero-copy event parsing from ring buffer data
 * REASON:    Minimize memory allocations and copying for high-performance event processing
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate data length before casting, check bounds for all field access
 * RISK:      High - Memory corruption if ring buffer data is malformed or truncated
 */

func (m *Manager) parseEvent(data []byte) *events.SystemEvent {
	if len(data) < int(unsafe.Sizeof(claudeEvent{})) {
		m.logger.Warn("Ring buffer data too short", "length", len(data), "expected", unsafe.Sizeof(claudeEvent{}))
		return nil
	}
	
	// Zero-copy cast to C struct
	rawEvent := (*claudeEvent)(unsafe.Pointer(&data[0]))
	
	event := &events.SystemEvent{
		PID:       rawEvent.PID,
		UID:       rawEvent.UID,
		Timestamp: time.Unix(0, int64(rawEvent.Timestamp)),
		Metadata:  make(map[string]interface{}),
	}
	
	// Safely extract command string
	event.Command = m.extractString(rawEvent.Comm[:])
	
	// Parse event type and specific data
	switch rawEvent.EventType {
	case EventExec:
		event.Type = events.EventExecve
		path := m.extractString(rawEvent.Path[:])
		if path != "" {
			event.SetMetadata("path", path)
		}
		event.SetMetadata("ppid", rawEvent.PPID)
		
	case EventConnect:
		event.Type = events.EventConnect 
		
		// Convert IP address to string
		addr := make(net.IP, 4)
		binary.LittleEndian.PutUint32(addr, rawEvent.TargetAddr)
		event.SetMetadata("target_ip", addr.String())
		event.SetMetadata("target_port", rawEvent.TargetPort)
		
		// Try to resolve hostname for known IPs
		if hostname := m.resolveAnthropicHost(addr); hostname != "" {
			event.SetMetadata("host", hostname)
		}
		
	case EventExit:
		event.Type = events.EventExit
		event.SetMetadata("exit_code", rawEvent.ExitCode)
		
	case EventHTTPRequest:
		event.Type = events.EventHTTPRequest
		
		// Extract HTTP method and URI
		httpMethod := m.extractString(rawEvent.HTTPMethod[:])
		httpURI := m.extractString(rawEvent.HTTPURI[:])
		
		// Convert IP address to string for correlation
		addr := make(net.IP, 4)
		binary.LittleEndian.PutUint32(addr, rawEvent.TargetAddr)
		
		// Set HTTP metadata
		event.SetMetadata("http_method", httpMethod)
		event.SetMetadata("http_uri", httpURI)
		event.SetMetadata("content_length", rawEvent.ContentLength)
		event.SetMetadata("socket_fd", rawEvent.SocketFD)
		event.SetMetadata("target_ip", addr.String())
		event.SetMetadata("target_port", rawEvent.TargetPort)
		
		// Try to resolve hostname for known IPs
		if hostname := m.resolveAnthropicHost(addr); hostname != "" {
			event.SetMetadata("host", hostname)
		}
		
	default:
		m.logger.Warn("Unknown event type", "type", rawEvent.EventType)
		return nil
	}
	
	return event
}

func (m *Manager) extractString(data []byte) string {
	// Find null terminator
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-015
 * CONTEXT:   Event filtering logic for Claude processes and Anthropic API connections
 * REASON:    Need efficient filtering to reduce noise and focus on relevant events
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep filtering logic fast, validate IP addresses before network operations
 * RISK:      Medium - Over-aggressive filtering could miss legitimate interactions
 */

func (m *Manager) isRelevantEvent(event *events.SystemEvent) bool {
	switch event.Type {
	case events.EventExecve:
		// Filter for Claude processes
		return m.isClaudeProcess(event.Command)
		
	case events.EventConnect:
		// Must be from Claude process AND to Anthropic API
		if !m.isClaudeProcess(event.Command) {
			return false
		}
		
		targetIP, _ := event.GetMetadata("target_ip")
		if ipStr, ok := targetIP.(string); ok {
			ip := net.ParseIP(ipStr)
			if ip != nil {
				return m.isAnthropicIP(ip)
			}
		}
		return false
		
	case events.EventExit:
		// Only care about Claude process exits
		return m.isClaudeProcess(event.Command)
		
	case events.EventHTTPRequest:
		// Must be from Claude process AND either user interaction or background operation
		if !m.isClaudeProcess(event.Command) {
			return false
		}
		
		// Verify this is to Anthropic API
		targetIP, _ := event.GetMetadata("target_ip")
		if ipStr, ok := targetIP.(string); ok {
			ip := net.ParseIP(ipStr)
			if ip != nil && m.isAnthropicIP(ip) {
				// All HTTP requests to Anthropic API are relevant
				// The business logic will determine if it's user interaction vs background
				return true
			}
		}
		return false
		
	default:
		return false
	}
}

func (m *Manager) isClaudeProcess(command string) bool {
	// Check for various Claude command formats
	return command == "claude" ||
		   strings.HasSuffix(command, "/claude") ||
		   strings.HasSuffix(command, "claude.exe") ||
		   strings.Contains(command, "claude")
}

func (m *Manager) isAnthropicIP(ip net.IP) bool {
	for _, network := range m.anthropicNets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (m *Manager) resolveAnthropicHost(ip net.IP) string {
	// Simple hostname mapping for known Anthropic endpoints
	if m.isAnthropicIP(ip) {
		return "api.anthropic.com"
	}
	return ""
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-016
 * CONTEXT:   Interface implementation for event channel access and statistics
 * REASON:    Implement required EBPFManager interface methods for daemon integration
 * CHANGE:    Initial implementation.
 * PREVENTION:Return read-only channels to prevent external manipulation
 * RISK:      Low - Interface methods are straightforward data access
 */

func (m *Manager) GetEventChannel() <-chan *events.SystemEvent {
	return m.eventCh
}

func (m *Manager) GetStats() (*arch.EBPFStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := &arch.EBPFStats{
		EventsProcessed:  atomic.LoadInt64(&m.eventsProcessed),
		DroppedEvents:    atomic.LoadInt64(&m.droppedEvents),
		ProgramsAttached: len(m.links),
		RingBufferSize:   256 * 1024, // Match MAX_EVENTS from C program
	}
	
	return stats, nil
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-017
 * CONTEXT:   Clean shutdown with proper resource cleanup
 * REASON:    Need graceful shutdown to prevent kernel resource leaks
 * CHANGE:    Initial implementation.
 * PREVENTION:Always cleanup in reverse order of initialization, handle cleanup errors
 * RISK:      High - Resource leaks can crash system or exhaust kernel memory
 */

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.running {
		return nil
	}
	
	m.logger.Info("Stopping eBPF manager")
	
	// Signal event processing to stop
	close(m.stopCh)
	
	// Wait for event processing to complete
	m.wg.Wait()
	
	// Cleanup eBPF resources
	m.cleanup()
	
	// Close event channel
	close(m.eventCh)
	
	m.running = false
	m.logger.Info("eBPF manager stopped successfully")
	
	return nil
}

func (m *Manager) cleanup() {
	// Close ring buffer reader
	if m.reader != nil {
		m.reader.Close()
		m.reader = nil
	}
	
	// Detach and close all links
	for _, l := range m.links {
		if err := l.Close(); err != nil {
			m.logger.Warn("Failed to close eBPF link", "error", err)
		}
	}
	m.links = nil
	
	// Close eBPF objects
	if m.objs != nil {
		m.objs.Close()
		m.objs = nil
	}
}