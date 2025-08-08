/**
 * CONTEXT:   Production HTTP middleware for Claude Monitor daemon
 * INPUT:     HTTP requests requiring rate limiting, logging, and metrics collection
 * OUTPUT:    Processed requests with rate limiting, logging, and performance tracking
 * BUSINESS:  Production middleware essential for API reliability and monitoring
 * CHANGE:    CRITICAL FIX - Complete production middleware implementation
 * RISK:      Medium - Middleware affecting all HTTP requests and API performance
 */

package daemon

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

/**
 * CONTEXT:   Rate limiting middleware using token bucket algorithm
 * INPUT:     HTTP requests requiring rate limiting based on configuration
 * OUTPUT:    HTTP 429 responses for rate limited requests, allowed requests passed through
 * BUSINESS:  Rate limiting prevents API abuse and ensures fair resource usage
 * CHANGE:    Production rate limiting using golang.org/x/time/rate package
 * RISK:      Medium - Rate limiting affecting API availability under high load
 */
func (o *Orchestrator) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check rate limit
		if !o.rateLimiter.Allow() {
			o.logger.Warn("Rate limit exceeded", 
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"endpoint", r.URL.Path)
			
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", o.config.Performance.RateLimitRPS))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", "1")
			
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate_limit_exceeded", "message": "Too many requests"}`))
			return
		}
		
		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", o.config.Performance.RateLimitRPS))
		
		next.ServeHTTP(w, r)
	})
}

/**
 * CONTEXT:   HTTP request logging middleware for audit and debugging
 * INPUT:     HTTP requests requiring structured logging
 * OUTPUT:    Structured log entries with request details and response status
 * BUSINESS:  Request logging essential for debugging, monitoring, and security audit
 * CHANGE:    Production request logging with structured slog integration
 * RISK:      Low - Logging middleware with minimal performance impact
 */
func (o *Orchestrator) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create response wrapper to capture status code
		wrapped := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Process request
		next.ServeHTTP(wrapped, r)
		
		// Log request with structured data
		duration := time.Since(start)
		
		o.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"status_code", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"content_length", r.ContentLength)
		
		// Update last request time for status endpoint
		o.requestMux.Lock()
		o.lastRequestTime = time.Now()
		o.requestMux.Unlock()
	})
}

/**
 * CONTEXT:   Metrics collection middleware for performance monitoring
 * INPUT:     HTTP requests requiring metrics collection
 * OUTPUT:    Updated request counters and connection tracking
 * BUSINESS:  Metrics collection enables monitoring, alerting, and performance analysis
 * CHANGE:    Basic metrics collection for production monitoring
 * RISK:      Low - Minimal overhead metrics collection
 */
func (o *Orchestrator) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment request counter
		atomic.AddInt64(&o.requestCount, 1)
		
		// Track active connections
		atomic.AddInt32(&o.connectionCount, 1)
		defer atomic.AddInt32(&o.connectionCount, -1)
		
		next.ServeHTTP(w, r)
	})
}

/**
 * CONTEXT:   Response wrapper for capturing HTTP status codes in middleware
 * INPUT:     HTTP responses requiring status code capture
 * OUTPUT:    Response with captured status code for logging
 * BUSINESS:  Status code capture essential for proper request logging and metrics
 * CHANGE:    Standard response wrapper pattern for middleware
 * RISK:      Low - Simple response wrapper with minimal overhead
 */
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}