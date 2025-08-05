/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-029
 * CONTEXT:   Test file for HTTP method detection functionality
 * REASON:    Need comprehensive tests to validate HTTP parsing and event classification
 * CHANGE:    Initial implementation.
 * PREVENTION:Test edge cases, malformed HTTP, and various method/URI combinations
 * RISK:      Low - Test file doesn't affect production code, helps prevent bugs
 */

package ebpf

import (
	"testing"

	"github.com/claude-monitor/claude-monitor/pkg/events"
)

func TestHTTPMethodDetection(t *testing.T) {
	tests := []struct {
		name           string
		httpMethod     string
		httpURI        string
		contentLength  uint32
		expectedUser   bool
		expectedBg     bool
		description    string
	}{
		{
			name:           "POST to messages endpoint",
			httpMethod:     "POST",
			httpURI:        "/v1/messages",
			contentLength:  1024,
			expectedUser:   true,
			expectedBg:     false,
			description:    "POST requests to /v1/messages are user interactions",
		},
		{
			name:           "POST to complete endpoint",
			httpMethod:     "POST", 
			httpURI:        "/v1/complete",
			contentLength:  512,
			expectedUser:   true,
			expectedBg:     false,
			description:    "POST requests to /v1/complete are user interactions",
		},
		{
			name:           "GET to health endpoint",
			httpMethod:     "GET",
			httpURI:        "/health",
			contentLength:  0,
			expectedUser:   false,
			expectedBg:     true,
			description:    "GET requests to /health are background operations",
		},
		{
			name:           "GET to status endpoint",
			httpMethod:     "GET",
			httpURI:        "/v1/status",
			contentLength:  0,
			expectedUser:   false,
			expectedBg:     true,
			description:    "GET requests to /v1/status are background operations",
		},
		{
			name:           "OPTIONS preflight",
			httpMethod:     "OPTIONS",
			httpURI:        "/v1/messages",
			contentLength:  0,
			expectedUser:   false,
			expectedBg:     true,
			description:    "OPTIONS requests are background preflight checks",
		},
		{
			name:           "HEAD connection check",
			httpMethod:     "HEAD",
			httpURI:        "/",
			contentLength:  0,
			expectedUser:   false,
			expectedBg:     true,
			description:    "HEAD requests are background connection checks",
		},
		{
			name:           "PUT request",
			httpMethod:     "PUT",
			httpURI:        "/v1/data",
			contentLength:  256,
			expectedUser:   true,
			expectedBg:     false,
			description:    "PUT requests are typically user-initiated",
		},
		{
			name:           "PATCH request",
			httpMethod:     "PATCH",
			httpURI:        "/v1/settings",
			contentLength:  128,
			expectedUser:   true,
			expectedBg:     false,
			description:    "PATCH requests are typically user-initiated",
		},
		{
			name:           "GET to interactive endpoint",
			httpMethod:     "GET",
			httpURI:        "/v1/messages",
			contentLength:  0,
			expectedUser:   true,
			expectedBg:     false,
			description:    "GET requests to /v1/messages are user interactions",
		},
		{
			name:           "Unknown method",
			httpMethod:     "CUSTOM",
			httpURI:        "/api/test",
			contentLength:  0,
			expectedUser:   false,
			expectedBg:     true,
			description:    "Unknown methods default to background operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTP request event
			event := &events.SystemEvent{
				Type:     events.EventHTTPRequest,
				PID:      12345,
				Command:  "claude",
				Metadata: make(map[string]interface{}),
			}
			
			// Set HTTP metadata
			event.SetMetadata("http_method", tt.httpMethod)
			event.SetMetadata("http_uri", tt.httpURI)
			event.SetMetadata("content_length", tt.contentLength)
			event.SetMetadata("target_ip", "52.84.1.1")
			event.SetMetadata("target_port", uint16(443))
			event.SetMetadata("host", "api.anthropic.com")

			// Test user interaction detection
			isUser := event.IsUserInteraction()
			if isUser != tt.expectedUser {
				t.Errorf("IsUserInteraction() = %v, expected %v for %s", 
					isUser, tt.expectedUser, tt.description)
			}

			// Test background operation detection  
			isBg := event.IsBackgroundOperation()
			if isBg != tt.expectedBg {
				t.Errorf("IsBackgroundOperation() = %v, expected %v for %s",
					isBg, tt.expectedBg, tt.description)
			}

			// Verify they are mutually exclusive (except for edge cases)
			if isUser && isBg {
				t.Errorf("Event cannot be both user interaction and background operation: %s", 
					tt.description)
			}
		})
	}
}

func TestHTTPEventMetadata(t *testing.T) {
	event := &events.SystemEvent{
		Type:     events.EventHTTPRequest,
		PID:      12345,
		Command:  "claude",
		Metadata: make(map[string]interface{}),
	}
	
	// Set test metadata
	event.SetMetadata("http_method", "POST")
	event.SetMetadata("http_uri", "/v1/messages")
	event.SetMetadata("content_length", uint32(1024))
	
	// Test getter methods
	if method := event.GetHTTPMethod(); method != "POST" {
		t.Errorf("GetHTTPMethod() = %q, expected %q", method, "POST")
	}
	
	if uri := event.GetHTTPURI(); uri != "/v1/messages" {
		t.Errorf("GetHTTPURI() = %q, expected %q", uri, "/v1/messages")
	}
	
	if length := event.GetContentLength(); length != 1024 {
		t.Errorf("GetContentLength() = %d, expected %d", length, 1024)
	}
	
	if !event.IsHTTPRequest() {
		t.Error("IsHTTPRequest() = false, expected true")
	}
}

func TestNonHTTPEventMethods(t *testing.T) {
	event := &events.SystemEvent{
		Type:     events.EventConnect,
		PID:      12345,
		Command:  "claude",
		Metadata: make(map[string]interface{}),
	}
	
	// Test that HTTP methods return defaults for non-HTTP events
	if method := event.GetHTTPMethod(); method != "" {
		t.Errorf("GetHTTPMethod() for non-HTTP event = %q, expected empty string", method)
	}
	
	if uri := event.GetHTTPURI(); uri != "" {
		t.Errorf("GetHTTPURI() for non-HTTP event = %q, expected empty string", uri)
	}
	
	if length := event.GetContentLength(); length != 0 {
		t.Errorf("GetContentLength() for non-HTTP event = %d, expected 0", length)
	}
	
	if event.IsHTTPRequest() {
		t.Error("IsHTTPRequest() for non-HTTP event = true, expected false")
	}
	
	if event.IsUserInteraction() {
		t.Error("IsUserInteraction() for non-HTTP event = true, expected false")
	}
	
	if event.IsBackgroundOperation() {
		t.Error("IsBackgroundOperation() for non-HTTP event = true, expected false")
	}
}

func TestEventValidation(t *testing.T) {
	validator := &events.EventValidator{}
	
	// Test HTTP request event validation
	httpEvent := &events.SystemEvent{
		Type:     events.EventHTTPRequest,
		PID:      12345,
		Command:  "claude",
		Metadata: make(map[string]interface{}),
	}
	httpEvent.SetMetadata("http_method", "POST")
	httpEvent.SetMetadata("http_uri", "/v1/messages")
	
	if !validator.IsRelevant(httpEvent) {
		t.Error("HTTP request event should be relevant")
	}
	
	// Test non-Claude process
	nonClaudeEvent := &events.SystemEvent{
		Type:     events.EventHTTPRequest,
		PID:      12345,
		Command:  "curl",
		Metadata: make(map[string]interface{}),
	}
	nonClaudeEvent.SetMetadata("http_method", "POST")
	nonClaudeEvent.SetMetadata("http_uri", "/v1/messages")
	
	if validator.IsRelevant(nonClaudeEvent) {
		t.Error("Non-Claude HTTP request event should not be relevant")
	}
}