package events

import (
	"time"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-015
 * CONTEXT:   Event type definitions for eBPF to Go communication
 * REASON:    Need well-defined event structures for kernel-userspace communication
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep event structures lightweight and avoid complex nested data
 * RISK:      Medium - Event structure changes require coordination with eBPF programs
 */

// EventType represents the type of system event captured
type EventType uint32

const (
	// EventExecve represents process execution events (execve syscall)
	EventExecve EventType = iota
	
	// EventConnect represents network connection events (connect syscall)
	EventConnect
	
	// EventExit represents process exit events
	EventExit
	
	// EventHTTPRequest represents HTTP request events from socket writes
	EventHTTPRequest
	
	// EventUnknown represents unrecognized events
	EventUnknown
)

// String returns a human-readable representation of the event type
func (et EventType) String() string {
	switch et {
	case EventExecve:
		return "execve"
	case EventConnect:
		return "connect"
	case EventExit:
		return "exit"
	case EventHTTPRequest:
		return "http_request"
	default:
		return "unknown"
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-016
 * CONTEXT:   SystemEvent structure representing events from eBPF programs
 * REASON:    Need consistent event format for all eBPF to Go communication
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure all event fields are properly initialized and validated
 * RISK:      High - Malformed events could cause processing errors or security issues
 */

// SystemEvent represents a system event captured by eBPF programs
type SystemEvent struct {
	// Type identifies the kind of event
	Type EventType `json:"type"`
	
	// PID is the process ID that generated the event
	PID uint32 `json:"pid"`
	
	// Command is the command name or path
	Command string `json:"command"`
	
	// Timestamp is when the event occurred (kernel time)
	Timestamp time.Time `json:"timestamp"`
	
	// UID is the user ID that owns the process
	UID uint32 `json:"uid"`
	
	// Metadata contains event-specific additional data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewSystemEvent creates a new system event with basic validation
func NewSystemEvent(eventType EventType, pid uint32, command string) *SystemEvent {
	return &SystemEvent{
		Type:      eventType,
		PID:       pid,
		Command:   command,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// IsClaudeProcess returns true if this event is from a Claude CLI process
func (se *SystemEvent) IsClaudeProcess() bool {
	return se.Command == "claude" || 
		   se.Command == "/usr/local/bin/claude" ||
		   se.Command == "claude.exe"
}

// IsAnthropicConnection returns true if this is a connection to Anthropic API
func (se *SystemEvent) IsAnthropicConnection() bool {
	if se.Type != EventConnect {
		return false
	}
	
	if host, ok := se.Metadata["host"].(string); ok {
		return host == "api.anthropic.com" || host == "claude.ai"
	}
	
	return false
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-028
 * CONTEXT:   HTTP request event utility methods for method detection and classification
 * REASON:    Need helper methods to classify HTTP requests as user interactions vs background operations
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate metadata exists before accessing, provide safe defaults
 * RISK:      Low - Helper methods are defensive and don't modify state
 */

// IsHTTPRequest returns true if this is an HTTP request event
func (se *SystemEvent) IsHTTPRequest() bool {
	return se.Type == EventHTTPRequest
}

// GetHTTPMethod returns the HTTP method for HTTP request events
func (se *SystemEvent) GetHTTPMethod() string {
	if !se.IsHTTPRequest() {
		return ""
	}
	
	if method, ok := se.Metadata["http_method"].(string); ok {
		return method
	}
	
	return ""
}

// GetHTTPURI returns the HTTP URI for HTTP request events
func (se *SystemEvent) GetHTTPURI() string {
	if !se.IsHTTPRequest() {
		return ""
	}
	
	if uri, ok := se.Metadata["http_uri"].(string); ok {
		return uri
	}
	
	return ""
}

// GetContentLength returns the Content-Length header value for HTTP request events
func (se *SystemEvent) GetContentLength() uint32 {
	if !se.IsHTTPRequest() {
		return 0
	}
	
	if length, ok := se.Metadata["content_length"].(uint32); ok {
		return length
	}
	
	return 0
}

// IsUserInteraction returns true if this HTTP request represents real user interaction
func (se *SystemEvent) IsUserInteraction() bool {
	if !se.IsHTTPRequest() {
		return false
	}
	
	method := se.GetHTTPMethod()
	uri := se.GetHTTPURI()
	
	// POST requests to /v1/messages are user interactions (conversation/prompts)
	if method == "POST" && (uri == "/v1/messages" || uri == "/v1/complete") {
		return true
	}
	
	// PUT/PATCH requests are also typically user-initiated
	if method == "PUT" || method == "PATCH" {
		return true
	}
	
	// GET requests are typically background operations (health checks, etc.)
	// unless they're to specific interactive endpoints
	if method == "GET" && (uri == "/v1/messages" || uri == "/v1/conversation") {
		return true
	}
	
	return false
}

// IsBackgroundOperation returns true if this HTTP request represents background activity
func (se *SystemEvent) IsBackgroundOperation() bool {
	if !se.IsHTTPRequest() {
		return false
	}
	
	method := se.GetHTTPMethod()
	uri := se.GetHTTPURI()
	
	// GET requests to health/status endpoints are background operations
	if method == "GET" && (uri == "/health" || uri == "/status" || uri == "/v1/status") {
		return true
	}
	
	// OPTIONS requests are typically preflight checks
	if method == "OPTIONS" {
		return true
	}
	
	// HEAD requests are typically connection checks
	if method == "HEAD" {
		return true
	}
	
	// Default: if not clearly a user interaction, treat as background
	return !se.IsUserInteraction()
}

// SetMetadata adds metadata to the event
func (se *SystemEvent) SetMetadata(key string, value interface{}) {
	if se.Metadata == nil {
		se.Metadata = make(map[string]interface{})
	}
	se.Metadata[key] = value
}

// GetMetadata retrieves metadata from the event
func (se *SystemEvent) GetMetadata(key string) (interface{}, bool) {
	if se.Metadata == nil {
		return nil, false
	}
	value, exists := se.Metadata[key]
	return value, exists
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-017
 * CONTEXT:   Event validation and filtering utilities
 * REASON:    Need robust event validation to prevent processing of malformed or irrelevant events
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate events before processing and log validation failures
 * RISK:      Medium - Invalid events could cause system instability or incorrect tracking
 */

// EventValidator provides validation for system events
type EventValidator struct{}

// Validate checks if an event is properly formed and relevant
func (ev *EventValidator) Validate(event *SystemEvent) error {
	if event == nil {
		return ErrNilEvent
	}
	
	if event.PID == 0 {
		return ErrInvalidPID
	}
	
	if event.Command == "" {
		return ErrEmptyCommand
	}
	
	if event.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	
	return nil
}

// IsRelevant checks if an event should be processed by the monitor
func (ev *EventValidator) IsRelevant(event *SystemEvent) bool {
	switch event.Type {
	case EventExecve:
		return event.IsClaudeProcess()
	case EventConnect:
		return event.IsAnthropicConnection()
	case EventExit:
		return event.IsClaudeProcess()
	case EventHTTPRequest:
		return event.IsClaudeProcess() && (event.IsUserInteraction() || event.IsBackgroundOperation())
	default:
		return false
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-018
 * CONTEXT:   Event processing error definitions
 * REASON:    Need well-defined errors for event processing failure handling
 * CHANGE:    Initial implementation.
 * PREVENTION:Always wrap errors with context and ensure errors are actionable
 * RISK:      Low - Error handling is defensive and improves system reliability
 */

// Event processing errors
var (
	ErrNilEvent          = NewEventError("event is nil")
	ErrInvalidPID        = NewEventError("invalid PID")
	ErrEmptyCommand      = NewEventError("empty command")
	ErrInvalidTimestamp  = NewEventError("invalid timestamp")
	ErrUnknownEventType  = NewEventError("unknown event type")
)

// EventError represents an event processing error
type EventError struct {
	Message string
}

func (ee *EventError) Error() string {
	return ee.Message
}

func NewEventError(message string) *EventError {
	return &EventError{Message: message}
}