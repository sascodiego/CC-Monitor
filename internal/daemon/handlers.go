/**
 * CONTEXT:   Production HTTP handlers for Claude Monitor daemon API
 * INPUT:     HTTP requests for health, status, and metrics endpoints
 * OUTPUT:    JSON responses with daemon health, status, and performance metrics
 * BUSINESS:  Production API endpoints for monitoring and health checking
 * CHANGE:    CRITICAL FIX - Complete production handlers for minimal API
 * RISK:      Low - Read-only endpoints with proper error handling
 */

package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

/**
 * CONTEXT:   Health check endpoint with database connectivity verification
 * INPUT:     HTTP GET request for health status
 * OUTPUT:    JSON response with health status and database connectivity
 * BUSINESS:  Health endpoint required for load balancer and monitoring systems
 * CHANGE:    Production health check with database verification
 * RISK:      Low - Health check with database ping
 */
func (o *Orchestrator) handleHealth(w http.ResponseWriter, r *http.Request) {
	healthData := map[string]interface{}{
		"status":    "ok",
		"service":   "claude-monitor",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
	}
	
	// Check database connectivity
	if o.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		
		if err := o.db.Ping(ctx); err != nil {
			healthData["status"] = "degraded"
			healthData["database"] = "unhealthy"
			healthData["error"] = err.Error()
			
			o.logger.Warn("Health check - database unhealthy", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			healthData["database"] = "healthy"
		}
	} else {
		healthData["database"] = "not_configured"
	}
	
	w.Header().Set("Content-Type", "application/json")
	if healthData["status"] == "ok" {
		w.WriteHeader(http.StatusOK)
	}
	
	json.NewEncoder(w).Encode(healthData)
}

/**
 * CONTEXT:   Comprehensive status endpoint with daemon runtime information
 * INPUT:     HTTP GET request for detailed daemon status
 * OUTPUT:    JSON response with uptime, configuration, and runtime metrics
 * BUSINESS:  Status endpoint provides detailed information for monitoring and debugging
 * CHANGE:    Production status endpoint with comprehensive daemon information
 * RISK:      Low - Status information endpoint with non-sensitive data
 */
func (o *Orchestrator) handleStatus(w http.ResponseWriter, r *http.Request) {
	uptime := o.GetUptime()
	
	statusData := map[string]interface{}{
		"status":      o.healthStatus,
		"uptime":      uptime.String(),
		"uptime_seconds": int64(uptime.Seconds()),
		"version":     "1.0.0",
		"pid":         os.Getpid(),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"start_time":  o.startTime.UTC().Format(time.RFC3339),
	}
	
	// Runtime metrics
	statusData["metrics"] = map[string]interface{}{
		"total_requests":     atomic.LoadInt64(&o.requestCount),
		"active_connections": atomic.LoadInt32(&o.connectionCount),
		"last_request_time":  o.lastRequestTime.Format(time.RFC3339),
	}
	
	// Configuration (non-sensitive)
	if o.config != nil {
		statusData["config"] = map[string]interface{}{
			"listen_addr":        fmt.Sprintf("%s:%d", o.config.Server.Host, o.config.Server.Port),
			"rate_limit_rps":     o.config.Performance.RateLimitRPS,
			"max_connections":    o.config.Database.MaxConnections,
			"tls_enabled":        o.config.Server.TLSEnabled,
		}
	}
	
	// Database status
	if o.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()
		
		if err := o.db.Ping(ctx); err != nil {
			statusData["database"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			statusData["database"] = map[string]interface{}{
				"status": "healthy",
				"path":   o.config.Database.Path,
			}
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(statusData)
}

/**
 * CONTEXT:   Basic metrics endpoint for monitoring systems
 * INPUT:     HTTP GET request for daemon metrics
 * OUTPUT:    JSON response with performance and operational metrics
 * BUSINESS:  Metrics endpoint provides data for monitoring and alerting systems
 * CHANGE:    Basic metrics implementation for production monitoring
 * RISK:      Low - Basic metrics endpoint with performance data
 */
func (o *Orchestrator) handleMetrics(w http.ResponseWriter, r *http.Request) {
	uptime := o.GetUptime()
	
	metricsData := map[string]interface{}{
		"claude_monitor_uptime_seconds":        int64(uptime.Seconds()),
		"claude_monitor_requests_total":        atomic.LoadInt64(&o.requestCount),
		"claude_monitor_active_connections":    atomic.LoadInt32(&o.connectionCount),
		"claude_monitor_rate_limit_rps":        o.config.Performance.RateLimitRPS,
		"claude_monitor_health_status":         o.healthStatus,
		"claude_monitor_version_info":          "1.0.0",
	}
	
	// Add database metrics if available
	if o.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		
		if err := o.db.Ping(ctx); err != nil {
			metricsData["claude_monitor_database_up"] = 0
		} else {
			metricsData["claude_monitor_database_up"] = 1
		}
	}
	
	// Add timestamp
	metricsData["claude_monitor_metrics_timestamp"] = time.Now().Unix()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metricsData)
}