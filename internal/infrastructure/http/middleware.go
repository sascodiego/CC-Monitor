/**
 * CONTEXT:   HTTP middleware for Claude Monitor daemon with logging, metrics, and validation
 * INPUT:     HTTP requests requiring cross-cutting concerns like logging and rate limiting
 * OUTPUT:    Enhanced request processing with observability and error handling
 * BUSINESS:  Support production operation with proper logging, metrics, and security
 * CHANGE:    Initial middleware implementation with comprehensive request processing
 * RISK:      Medium - Middleware affects all HTTP requests and system observability
 */

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

/**
 * CONTEXT:   Request logging middleware for HTTP request/response observability
 * INPUT:     HTTP requests and responses flowing through the system
 * OUTPUT:    Structured logs with request details, timing, and response status
 * BUSINESS:  Provide request visibility for debugging, monitoring, and performance analysis
 * CHANGE:    Initial logging middleware with comprehensive request information
 * RISK:      Low - Logging middleware with no request modification
 */
func LoggingMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			
			// Create response writer wrapper to capture status and size
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     200, // Default status code
			}
			
			// Extract request ID if present
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}
			
			// Add request ID to response headers
			wrapper.Header().Set("X-Request-ID", requestID)
			
			// Log request start
			logger.Info("HTTP request started",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.Header.Get("User-Agent"),
				"content_length", r.ContentLength,
				"request_id", requestID,
			)
			
			// Process request
			next.ServeHTTP(wrapper, r)
			
			// Calculate processing time
			duration := time.Since(startTime)
			
			// Log request completion
			logLevel := slog.LevelInfo
			if wrapper.statusCode >= 400 {
				logLevel = slog.LevelWarn
			}
			if wrapper.statusCode >= 500 {
				logLevel = slog.LevelError
			}
			
			logger.Log(r.Context(), logLevel, "HTTP request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapper.statusCode,
				"duration_ms", duration.Milliseconds(),
				"response_size", wrapper.size,
				"request_id", requestID,
			)
		})
	}
}

/**
 * CONTEXT:   Metrics collection middleware for HTTP request statistics
 * INPUT:     HTTP requests and responses requiring performance measurement
 * OUTPUT:    Request metrics including counts, durations, and status distributions
 * BUSINESS:  Support performance monitoring and capacity planning decisions
 * CHANGE:    Initial metrics middleware with request timing and counting
 * RISK:      Low - Metrics collection with minimal performance overhead
 */
func MetricsMiddleware() mux.MiddlewareFunc {
	var (
		requestCount    int64
		requestDuration time.Duration
		statusCounts    = make(map[int]int64)
		mu             sync.RWMutex
	)
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     200,
			}
			
			// Process request
			next.ServeHTTP(wrapper, r)
			
			// Update metrics
			duration := time.Since(startTime)
			mu.Lock()
			requestCount++
			requestDuration += duration
			statusCounts[wrapper.statusCode]++
			mu.Unlock()
			
			// Add metrics headers
			w.Header().Set("X-Response-Time", fmt.Sprintf("%d", duration.Milliseconds()))
		})
	}
}

/**
 * CONTEXT:   Rate limiting middleware to protect against excessive requests
 * INPUT:     HTTP requests requiring rate limit enforcement
 * OUTPUT:    Rate limit enforcement with HTTP 429 responses when limits exceeded
 * BUSINESS:  Protect system resources from abuse and ensure fair usage
 * CHANGE:    Initial rate limiting with simple token bucket algorithm
 * RISK:      Medium - Rate limiting affects client request processing
 */
func RateLimitMiddleware(requestsPerSecond int) mux.MiddlewareFunc {
	type client struct {
		tokens    int
		lastRefill time.Time
	}
	
	clients := make(map[string]*client)
	mu := sync.RWMutex{}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client identifier (IP address)
			clientIP := getClientIP(r)
			
			mu.Lock()
			c, exists := clients[clientIP]
			if !exists {
				c = &client{
					tokens:    requestsPerSecond,
					lastRefill: time.Now(),
				}
				clients[clientIP] = c
			}
			
			// Refill tokens based on time passed
			now := time.Now()
			timePassed := now.Sub(c.lastRefill)
			tokensToAdd := int(timePassed.Seconds()) * requestsPerSecond
			if tokensToAdd > 0 {
				c.tokens = min(requestsPerSecond, c.tokens+tokensToAdd)
				c.lastRefill = now
			}
			
			// Check if request is allowed
			if c.tokens <= 0 {
				mu.Unlock()
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			// Consume token
			c.tokens--
			mu.Unlock()
			
			next.ServeHTTP(w, r)
		})
	}
}

/**
 * CONTEXT:   Request validation middleware for input sanitization and security
 * INPUT:     HTTP requests requiring validation and sanitization
 * OUTPUT:    Validated requests with sanitized headers and proper content validation
 * BUSINESS:  Ensure request integrity and protect against malicious input
 * CHANGE:    Initial validation middleware with security-focused checks
 * RISK:      Medium - Request validation affects all API functionality
 */
func ValidationMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check Content-Type for POST/PUT requests
			if r.Method == "POST" || r.Method == "PUT" {
				contentType := r.Header.Get("Content-Type")
				if !strings.Contains(contentType, "application/json") {
					http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
					return
				}
			}
			
			// Check request size
			if r.ContentLength > 1024*1024 { // 1MB limit
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			
			// Sanitize headers
			r.Header.Set("X-Forwarded-Host", "") // Remove potential header injection
			
			next.ServeHTTP(w, r)
		})
	}
}

/**
 * CONTEXT:   CORS middleware for cross-origin request support
 * INPUT:     HTTP requests requiring CORS header handling
 * OUTPUT:    Proper CORS headers enabling cross-origin API access
 * BUSINESS:  Support web-based clients and development environments
 * CHANGE:    Initial CORS middleware with security-conscious defaults
 * RISK:      Medium - CORS configuration affects cross-origin security
 */
func CORSMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")
			
			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

/**
 * CONTEXT:   Panic recovery middleware for graceful error handling
 * INPUT:     HTTP requests that might cause panic conditions
 * OUTPUT:    Graceful error responses with panic recovery and logging
 * BUSINESS:  Ensure system stability and proper error reporting during failures
 * CHANGE:    Initial panic recovery with comprehensive error logging
 * RISK:      High - Panic recovery affects system stability and error handling
 */
func RecoveryMiddleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get stack trace
					stackTrace := string(debug.Stack())
					
					// Log the panic
					logger.Error("HTTP handler panic recovered",
						"error", err,
						"method", r.Method,
						"path", r.URL.Path,
						"remote_addr", r.RemoteAddr,
						"stack_trace", stackTrace,
					)
					
					// Return error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					
					errorResponse := map[string]interface{}{
						"status":    "error",
						"error":     "Internal Server Error",
						"message":   "An unexpected error occurred",
						"timestamp": time.Now().UTC(),
					}
					
					// Don't include panic details in response for security
					if json := tryMarshalJSON(errorResponse); json != nil {
						w.Write(json)
					} else {
						w.Write([]byte(`{"status":"error","message":"Internal server error"}`))
					}
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

/**
 * CONTEXT:   Request timeout middleware to prevent hung connections
 * INPUT:     HTTP requests requiring timeout enforcement
 * OUTPUT:    Timeout enforcement with proper error responses
 * BUSINESS:  Ensure system responsiveness and resource management
 * CHANGE:    Initial timeout middleware with configurable duration
 * RISK:      Medium - Timeout configuration affects request processing
 */
func TimeoutMiddleware(timeout time.Duration) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create timeout context
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			
			// Create new request with timeout context
			r = r.WithContext(ctx)
			
			// Channel to signal completion
			done := make(chan struct{})
			
			// Process request in goroutine
			go func() {
				defer close(done)
				next.ServeHTTP(w, r)
			}()
			
			// Wait for completion or timeout
			select {
			case <-done:
				// Request completed successfully
				return
			case <-ctx.Done():
				// Request timed out
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestTimeout)
				
				errorResponse := map[string]interface{}{
					"status":    "error",
					"error":     "Request Timeout",
					"message":   fmt.Sprintf("Request timed out after %v", timeout),
					"timestamp": time.Now().UTC(),
				}
				
				if json := tryMarshalJSON(errorResponse); json != nil {
					w.Write(json)
				}
				return
			}
		})
	}
}

// Helper types and functions

/**
 * CONTEXT:   Response writer wrapper for capturing HTTP response metrics
 * INPUT:     HTTP responses requiring status code and size tracking
 * OUTPUT:    Response metrics captured for logging and monitoring
 * BUSINESS:  Enable response monitoring and performance analysis
 * CHANGE:    Initial response wrapper with status and size tracking
 * RISK:      Low - Response wrapper with no functional changes
 */
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriterWrapper) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Utility functions

func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Use remote address
	return r.RemoteAddr
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func tryMarshalJSON(v interface{}) []byte {
	defer func() {
		recover() // Ignore any marshaling errors
	}()
	
	if data, err := json.Marshal(v); err == nil {
		return data
	}
	return nil
}