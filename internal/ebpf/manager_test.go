/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-018
 * CONTEXT:   Unit tests for eBPF manager functionality and event processing
 * REASON:    Need comprehensive testing of eBPF manager without requiring root privileges
 * CHANGE:    Initial implementation.
 * PREVENTION:Mock kernel interactions for testing, validate event parsing logic independently
 * RISK:      Medium - Test failures could indicate production issues with event processing
 */

package ebpf

import (
	"net"
	"testing"
	"time"
	"unsafe"

	"github.com/claude-monitor/claude-monitor/pkg/events"
	"github.com/claude-monitor/claude-monitor/pkg/logger"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	messages []string
}

func (ml *MockLogger) Debug(msg string, fields ...interface{}) {
	ml.messages = append(ml.messages, msg)
}

func (ml *MockLogger) Info(msg string, fields ...interface{}) {
	ml.messages = append(ml.messages, msg)
}

func (ml *MockLogger) Warn(msg string, fields ...interface{}) {
	ml.messages = append(ml.messages, msg)
}

func (ml *MockLogger) Error(msg string, fields ...interface{}) {
	ml.messages = append(ml.messages, msg)
}

func (ml *MockLogger) Fatal(msg string, fields ...interface{}) {
	ml.messages = append(ml.messages, msg)
}

func TestNewManager(t *testing.T) {
	logger := &MockLogger{}
	manager := NewManager(logger)
	
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	
	if manager.logger != logger {
		t.Error("Logger not set correctly")
	}
	
	if len(manager.anthropicNets) == 0 {
		t.Error("Anthropic IP networks not parsed")
	}
	
	if cap(manager.eventCh) != 1000 {
		t.Error("Event channel capacity not set correctly")
	}
}

func TestParseEvent(t *testing.T) {
	logger := &MockLogger{}
	manager := NewManager(logger)
	
	// Test execve event parsing
	t.Run("ExecveEvent", func(t *testing.T) {
		rawEvent := claudeEvent{
			Timestamp: uint64(time.Now().UnixNano()),
			PID:       12345,
			PPID:      12344,
			UID:       1000,
			EventType: EventExec,
		}
		copy(rawEvent.Comm[:], "claude")
		copy(rawEvent.Path[:], "/usr/local/bin/claude")
		
		// Convert to byte slice
		data := (*[unsafe.Sizeof(claudeEvent{})]byte)(unsafe.Pointer(&rawEvent))[:]
		
		event := manager.parseEvent(data)
		if event == nil {
			t.Fatal("parseEvent returned nil")
		}
		
		if event.Type != events.EventExecve {
			t.Errorf("Expected EventExecve, got %v", event.Type)
		}
		
		if event.PID != 12345 {
			t.Errorf("Expected PID 12345, got %d", event.PID)
		}
		
		if event.Command != "claude" {
			t.Errorf("Expected command 'claude', got '%s'", event.Command)
		}
		
		path, ok := event.GetMetadata("path")
		if !ok {
			t.Error("Path metadata not found")
		}
		
		if path != "/usr/local/bin/claude" {
			t.Errorf("Expected path '/usr/local/bin/claude', got '%s'", path)
		}
	})
	
	// Test connect event parsing
	t.Run("ConnectEvent", func(t *testing.T) {
		rawEvent := claudeEvent{
			Timestamp:  uint64(time.Now().UnixNano()),
			PID:        12345,
			UID:        1000,
			EventType:  EventConnect,
			TargetAddr: 0x08080808, // 8.8.8.8 in little endian
			TargetPort: 443,
		}
		copy(rawEvent.Comm[:], "claude")
		
		data := (*[unsafe.Sizeof(claudeEvent{})]byte)(unsafe.Pointer(&rawEvent))[:]
		
		event := manager.parseEvent(data)
		if event == nil {
			t.Fatal("parseEvent returned nil")
		}
		
		if event.Type != events.EventConnect {
			t.Errorf("Expected EventConnect, got %v", event.Type)
		}
		
		targetIP, ok := event.GetMetadata("target_ip")
		if !ok {
			t.Error("target_ip metadata not found")
		}
		
		// The IP should be parsed correctly
		if targetIP != "8.8.8.8" {
			t.Errorf("Expected IP '8.8.8.8', got '%s'", targetIP)
		}
		
		targetPort, ok := event.GetMetadata("target_port")
		if !ok {
			t.Error("target_port metadata not found")
		}
		
		if targetPort != uint16(443) {
			t.Errorf("Expected port 443, got %v", targetPort)
		}
	})
}

func TestIsClaudeProcess(t *testing.T) {
	logger := &MockLogger{}
	manager := NewManager(logger)
	
	testCases := []struct {
		command  string
		expected bool
	}{
		{"claude", true},
		{"/usr/local/bin/claude", true},
		{"claude.exe", true},
		{"some-claude-tool", true},
		{"python", false},
		{"bash", false},
		{"", false},
	}
	
	for _, tc := range testCases {
		result := manager.isClaudeProcess(tc.command)
		if result != tc.expected {
			t.Errorf("isClaudeProcess(%q) = %v, expected %v", tc.command, result, tc.expected)
		}
	}
}

func TestIsAnthropicIP(t *testing.T) {
	logger := &MockLogger{}
	manager := NewManager(logger)
	
	// Test with known CloudFront IP (should match anthropic ranges)
	cloudFrontIP := net.ParseIP("52.84.1.1")
	if !manager.isAnthropicIP(cloudFrontIP) {
		t.Error("CloudFront IP should be detected as Anthropic")
	}
	
	// Test with random IP (should not match)
	randomIP := net.ParseIP("192.168.1.1")
	if manager.isAnthropicIP(randomIP) {
		t.Error("Random IP should not be detected as Anthropic")
	}
}

func TestEventFiltering(t *testing.T) {
	logger := &MockLogger{}
	manager := NewManager(logger)
	
	// Test execve event filtering
	t.Run("ExecveFiltering", func(t *testing.T) {
		claudeEvent := &events.SystemEvent{
			Type:    events.EventExecve,
			Command: "claude",
		}
		
		if !manager.isRelevantEvent(claudeEvent) {
			t.Error("Claude execve event should be relevant")
		}
		
		pythonEvent := &events.SystemEvent{
			Type:    events.EventExecve,
			Command: "python",
		}
		
		if manager.isRelevantEvent(pythonEvent) {
			t.Error("Python execve event should not be relevant")
		}
	})
	
	// Test connect event filtering
	t.Run("ConnectFiltering", func(t *testing.T) {
		claudeConnectEvent := &events.SystemEvent{
			Type:     events.EventConnect,
			Command:  "claude",
			Metadata: map[string]interface{}{
				"target_ip": "52.84.1.1", // CloudFront IP
			},
		}
		
		if !manager.isRelevantEvent(claudeConnectEvent) {
			t.Error("Claude connect to Anthropic should be relevant")
		}
		
		pythonConnectEvent := &events.SystemEvent{
			Type:     events.EventConnect,
			Command:  "python",
			Metadata: map[string]interface{}{
				"target_ip": "52.84.1.1",
			},
		}
		
		if manager.isRelevantEvent(pythonConnectEvent) {
			t.Error("Python connect should not be relevant")
		}
		
		claudeRandomConnect := &events.SystemEvent{
			Type:     events.EventConnect,
			Command:  "claude",
			Metadata: map[string]interface{}{
				"target_ip": "192.168.1.1",
			},
		}
		
		if manager.isRelevantEvent(claudeRandomConnect) {
			t.Error("Claude connect to random IP should not be relevant")
		}
	})
}

func TestGetStats(t *testing.T) {
	logger := &MockLogger{}
	manager := NewManager(logger)
	
	stats, err := manager.GetStats()
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	
	if stats == nil {
		t.Fatal("GetStats returned nil stats")
	}
	
	if stats.RingBufferSize != 256*1024 {
		t.Errorf("Expected ring buffer size %d, got %d", 256*1024, stats.RingBufferSize)
	}
	
	// Initially, no programs should be attached
	if stats.ProgramsAttached != 0 {
		t.Errorf("Expected 0 programs attached, got %d", stats.ProgramsAttached)
	}
}

func TestExtractString(t *testing.T) {
	logger := &MockLogger{}
	manager := NewManager(logger)
	
	testCases := []struct {
		input    []byte
		expected string
	}{
		{[]byte{'c', 'l', 'a', 'u', 'd', 'e', 0, 0, 0}, "claude"},
		{[]byte{'t', 'e', 's', 't', 0}, "test"},
		{[]byte{0, 0, 0, 0}, ""},
		{[]byte{'n', 'o', 'n', 'u', 'l', 'l'}, "nonull"}, // no null terminator
	}
	
	for _, tc := range testCases {
		result := manager.extractString(tc.input)
		if result != tc.expected {
			t.Errorf("extractString(%v) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// Benchmark event parsing performance
func BenchmarkParseEvent(b *testing.B) {
	logger := logger.NewDefaultLogger("test", "ERROR") // Reduce logging overhead
	manager := NewManager(logger)
	
	rawEvent := claudeEvent{
		Timestamp: uint64(time.Now().UnixNano()),
		PID:       12345,
		PPID:      12344,
		UID:       1000,
		EventType: EventExec,
	}
	copy(rawEvent.Comm[:], "claude")
	copy(rawEvent.Path[:], "/usr/local/bin/claude")
	
	data := (*[unsafe.Sizeof(claudeEvent{})]byte)(unsafe.Pointer(&rawEvent))[:]
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := manager.parseEvent(data)
		if event == nil {
			b.Fatal("parseEvent returned nil")
		}
	}
}