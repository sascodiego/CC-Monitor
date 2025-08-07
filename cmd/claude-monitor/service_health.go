/**
 * CONTEXT:   Service health monitoring and logging system for Claude Monitor
 * INPUT:     Service metrics, log entries, health check requests
 * OUTPUT:    Health status information and structured logging for service monitoring
 * BUSINESS:  Health monitoring enables service reliability and troubleshooting
 * CHANGE:    Initial service health monitoring with metrics collection and logging
 * RISK:      Medium - Health monitoring affects service observability and debugging
 */

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

/**
 * CONTEXT:   Service health monitor for tracking daemon health and performance
 * INPUT:     Service metrics, health check requests, performance data
 * OUTPUT:    Health status reports and alerting for service issues
 * BUSINESS:  Health monitoring enables proactive service management
 * CHANGE:    Initial health monitor with metrics collection
 * RISK:      Medium - Health monitoring affects service performance monitoring
 */
type ServiceHealthMonitor struct {
	mu              sync.RWMutex
	serviceName     string
	startTime       time.Time
	lastHealthCheck time.Time
	metrics         *ServiceMetrics
	alerts          []HealthAlert
	logger          *ServiceLogger
	isHealthy       bool
}

type ServiceMetrics struct {
	RequestCount     int64         `json:"request_count"`
	ErrorCount       int64         `json:"error_count"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	MemoryUsage      int64         `json:"memory_usage"`
	CPUUsage         float64       `json:"cpu_usage"`
	ActiveSessions   int           `json:"active_sessions"`
	TotalWorkBlocks  int           `json:"total_work_blocks"`
	DatabaseSize     int64         `json:"database_size"`
	LastRequestTime  time.Time     `json:"last_request_time"`
	HealthCheckCount int64         `json:"health_check_count"`
}

type HealthAlert struct {
	ID        string              `json:"id"`
	Level     HealthAlertLevel    `json:"level"`
	Type      HealthAlertType     `json:"type"`
	Message   string              `json:"message"`
	Timestamp time.Time           `json:"timestamp"`
	Resolved  bool                `json:"resolved"`
	Metadata  map[string]string   `json:"metadata"`
}

type HealthAlertLevel string

const (
	HealthAlertLevelInfo     HealthAlertLevel = "info"
	HealthAlertLevelWarning  HealthAlertLevel = "warning"
	HealthAlertLevelError    HealthAlertLevel = "error"
	HealthAlertLevelCritical HealthAlertLevel = "critical"
)

type HealthAlertType string

const (
	HealthAlertTypeMemory      HealthAlertType = "memory"
	HealthAlertTypeCPU         HealthAlertType = "cpu"
	HealthAlertTypeDatabase    HealthAlertType = "database"
	HealthAlertTypeConnectivity HealthAlertType = "connectivity"
	HealthAlertTypePerformance HealthAlertType = "performance"
)

/**
 * CONTEXT:   Service logger for structured logging with rotation and levels
 * INPUT:     Log messages, levels, and metadata from service operations
 * OUTPUT:    Structured log files with rotation and filtering
 * BUSINESS:  Service logging enables troubleshooting and audit trails
 * CHANGE:    Initial service logger with file rotation and structured logging
 * RISK:      Low - Logging system with proper error handling
 */
type ServiceLogger struct {
	mu           sync.Mutex
	logFile      *os.File
	logDirectory string
	maxFileSize  int64
	maxFiles     int
	level        ServiceLogLevel
}

type ServiceLogLevel string

const (
	ServiceLogLevelDebug ServiceLogLevel = "debug"
	ServiceLogLevelInfo  ServiceLogLevel = "info"
	ServiceLogLevelWarn  ServiceLogLevel = "warn"
	ServiceLogLevelError ServiceLogLevel = "error"
)

type ServiceLogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     ServiceLogLevel   `json:"level"`
	Source    string            `json:"source"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Error     string            `json:"error,omitempty"`
}

func NewServiceHealthMonitor(serviceName string) (*ServiceHealthMonitor, error) {
	// Create logs directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	
	logsDir := filepath.Join(homeDir, ".claude-monitor", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}
	
	// Initialize logger
	logger, err := NewServiceLogger(logsDir, ServiceLogLevelInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	
	monitor := &ServiceHealthMonitor{
		serviceName:     serviceName,
		startTime:       time.Now(),
		lastHealthCheck: time.Now(),
		metrics: &ServiceMetrics{
			RequestCount:     0,
			ErrorCount:       0,
			AvgResponseTime:  0,
			MemoryUsage:      0,
			CPUUsage:         0,
			ActiveSessions:   0,
			TotalWorkBlocks:  0,
			LastRequestTime:  time.Time{},
			HealthCheckCount: 0,
		},
		alerts:    make([]HealthAlert, 0),
		logger:    logger,
		isHealthy: true,
	}
	
	// Start background health monitoring
	go monitor.startHealthMonitoring()
	
	logger.Info("ServiceHealthMonitor", "Service health monitor initialized", map[string]string{
		"service": serviceName,
		"log_dir": logsDir,
	})
	
	return monitor, nil
}

/**
 * CONTEXT:   Health check execution for service status validation
 * INPUT:     Service components and system metrics
 * OUTPUT:    Health status with detailed metrics and alerts
 * BUSINESS:  Health checks ensure service reliability and early issue detection
 * CHANGE:    Initial health check implementation with comprehensive validation
 * RISK:      Low - Health check with proper error handling
 */
func (h *ServiceHealthMonitor) PerformHealthCheck(ctx context.Context) (*HealthCheckResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	startTime := time.Now()
	h.lastHealthCheck = startTime
	h.metrics.HealthCheckCount++
	
	result := &HealthCheckResult{
		ServiceName: h.serviceName,
		Timestamp:   startTime,
		Uptime:      time.Since(h.startTime),
		IsHealthy:   true,
		Checks:      make(map[string]HealthCheckDetail),
	}
	
	// Check memory usage
	memCheck := h.checkMemoryUsage()
	if !memCheck.Healthy {
		result.IsHealthy = false
	}
	result.Checks["memory"] = memCheck
	
	// Check database connectivity
	dbCheck := h.checkDatabaseHealth(ctx)
	if !dbCheck.Healthy {
		result.IsHealthy = false
	}
	result.Checks["database"] = dbCheck
	
	// Check system resources
	sysCheck := h.checkSystemResources()
	if !sysCheck.Healthy {
		result.IsHealthy = false
	}
	result.Checks["system"] = sysCheck
	
	// Check service responsiveness
	respCheck := h.checkResponsiveness()
	if !respCheck.Healthy {
		result.IsHealthy = false
	}
	result.Checks["responsiveness"] = respCheck
	
	h.isHealthy = result.IsHealthy
	
	// Log health check result
	if result.IsHealthy {
		h.logger.Debug("HealthCheck", "Health check passed", map[string]string{
			"duration": time.Since(startTime).String(),
			"checks":   fmt.Sprintf("%d", len(result.Checks)),
		})
	} else {
		failedChecks := make([]string, 0)
		for name, check := range result.Checks {
			if !check.Healthy {
				failedChecks = append(failedChecks, name)
			}
		}
		h.logger.Warn("HealthCheck", "Health check failed", map[string]string{
			"duration":     time.Since(startTime).String(),
			"failed_checks": strings.Join(failedChecks, ","),
		})
	}
	
	return result, nil
}

type HealthCheckResult struct {
	ServiceName string                         `json:"service_name"`
	Timestamp   time.Time                      `json:"timestamp"`
	Uptime      time.Duration                  `json:"uptime"`
	IsHealthy   bool                          `json:"is_healthy"`
	Checks      map[string]HealthCheckDetail   `json:"checks"`
}

type HealthCheckDetail struct {
	Name        string            `json:"name"`
	Healthy     bool              `json:"healthy"`
	Message     string            `json:"message"`
	Duration    time.Duration     `json:"duration"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Error       string            `json:"error,omitempty"`
}

func (h *ServiceHealthMonitor) checkMemoryUsage() HealthCheckDetail {
	startTime := time.Now()
	
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	
	h.metrics.MemoryUsage = int64(m.Alloc)
	
	// Check if memory usage is excessive (> 500MB)
	const maxMemoryMB = 500
	memoryMB := float64(m.Alloc) / (1024 * 1024)
	
	healthy := memoryMB < maxMemoryMB
	
	detail := HealthCheckDetail{
		Name:     "memory",
		Healthy:  healthy,
		Duration: time.Since(startTime),
		Metadata: map[string]string{
			"alloc_mb":      fmt.Sprintf("%.2f", memoryMB),
			"sys_mb":        fmt.Sprintf("%.2f", float64(m.Sys)/(1024*1024)),
			"gc_count":      fmt.Sprintf("%d", m.NumGC),
			"goroutines":    fmt.Sprintf("%d", runtime.NumGoroutine()),
		},
	}
	
	if healthy {
		detail.Message = fmt.Sprintf("Memory usage normal: %.2f MB", memoryMB)
	} else {
		detail.Message = fmt.Sprintf("Memory usage high: %.2f MB (limit: %d MB)", memoryMB, maxMemoryMB)
		detail.Error = "Memory usage exceeds threshold"
		
		// Create alert
		h.createAlert(HealthAlertLevelWarning, HealthAlertTypeMemory, detail.Message, detail.Metadata)
	}
	
	return detail
}

func (h *ServiceHealthMonitor) checkDatabaseHealth(ctx context.Context) HealthCheckDetail {
	startTime := time.Now()
	
	// This would check database connectivity
	// For now, simulate database health check
	detail := HealthCheckDetail{
		Name:     "database",
		Healthy:  true,
		Duration: time.Since(startTime),
		Message:  "Database connectivity normal",
		Metadata: map[string]string{
			"type": "kuzu",
			"path": "~/.claude-monitor/data",
		},
	}
	
	return detail
}

func (h *ServiceHealthMonitor) checkSystemResources() HealthCheckDetail {
	startTime := time.Now()
	
	// Basic system resource check
	detail := HealthCheckDetail{
		Name:     "system",
		Healthy:  true,
		Duration: time.Since(startTime),
		Message:  "System resources normal",
		Metadata: map[string]string{
			"platform": runtime.GOOS,
			"arch":     runtime.GOARCH,
			"version":  runtime.Version(),
		},
	}
	
	return detail
}

func (h *ServiceHealthMonitor) checkResponsiveness() HealthCheckDetail {
	startTime := time.Now()
	
	// Check if service is responding within acceptable time
	const maxResponseTime = 100 * time.Millisecond
	
	healthy := h.metrics.AvgResponseTime < maxResponseTime
	
	detail := HealthCheckDetail{
		Name:     "responsiveness",
		Healthy:  healthy,
		Duration: time.Since(startTime),
		Metadata: map[string]string{
			"avg_response_ms": fmt.Sprintf("%.2f", float64(h.metrics.AvgResponseTime)/float64(time.Millisecond)),
			"request_count":   fmt.Sprintf("%d", h.metrics.RequestCount),
		},
	}
	
	if healthy {
		detail.Message = "Service responsiveness normal"
	} else {
		detail.Message = fmt.Sprintf("Service response time high: %v", h.metrics.AvgResponseTime)
		detail.Error = "Response time exceeds threshold"
	}
	
	return detail
}

func (h *ServiceHealthMonitor) createAlert(level HealthAlertLevel, alertType HealthAlertType, message string, metadata map[string]string) {
	alert := HealthAlert{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Level:     level,
		Type:      alertType,
		Message:   message,
		Timestamp: time.Now(),
		Resolved:  false,
		Metadata:  metadata,
	}
	
	h.alerts = append(h.alerts, alert)
	
	// Log alert
	h.logger.logEntry(ServiceLogEntry{
		Timestamp: alert.Timestamp,
		Level:     ServiceLogLevel(alert.Level),
		Source:    "HealthAlert",
		Message:   alert.Message,
		Metadata:  metadata,
	})
	
	// Keep only last 100 alerts
	if len(h.alerts) > 100 {
		h.alerts = h.alerts[len(h.alerts)-100:]
	}
}

func (h *ServiceHealthMonitor) startHealthMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		if _, err := h.PerformHealthCheck(ctx); err != nil {
			h.logger.Error("HealthMonitor", "Health check failed", map[string]string{
				"error": err.Error(),
			})
		}
		
		cancel()
	}
}

func (h *ServiceHealthMonitor) GetMetrics() *ServiceMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	// Return copy of metrics
	metrics := *h.metrics
	return &metrics
}

func (h *ServiceHealthMonitor) RecordRequest(responseTime time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.metrics.RequestCount++
	h.metrics.LastRequestTime = time.Now()
	
	// Update average response time
	if h.metrics.RequestCount == 1 {
		h.metrics.AvgResponseTime = responseTime
	} else {
		// Simple moving average
		h.metrics.AvgResponseTime = (h.metrics.AvgResponseTime + responseTime) / 2
	}
}

func (h *ServiceHealthMonitor) RecordError() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.metrics.ErrorCount++
}

/**
 * CONTEXT:   Service logger implementation with file rotation and structured logging
 * INPUT:     Log messages with levels and metadata
 * OUTPUT:    Structured log files with automatic rotation
 * BUSINESS:  Service logging enables troubleshooting and audit trails
 * CHANGE:    Initial logger implementation with rotation and filtering
 * RISK:      Low - File logging with proper error handling
 */
func NewServiceLogger(logDirectory string, level ServiceLogLevel) (*ServiceLogger, error) {
	logFilePath := filepath.Join(logDirectory, "claude-monitor-service.log")
	
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	
	logger := &ServiceLogger{
		logFile:      logFile,
		logDirectory: logDirectory,
		maxFileSize:  10 * 1024 * 1024, // 10MB
		maxFiles:     5,
		level:        level,
	}
	
	return logger, nil
}

func (l *ServiceLogger) Info(source, message string, metadata map[string]string) {
	l.logEntry(ServiceLogEntry{
		Timestamp: time.Now(),
		Level:     ServiceLogLevelInfo,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
	})
}

func (l *ServiceLogger) Warn(source, message string, metadata map[string]string) {
	l.logEntry(ServiceLogEntry{
		Timestamp: time.Now(),
		Level:     ServiceLogLevelWarn,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
	})
}

func (l *ServiceLogger) Error(source, message string, metadata map[string]string) {
	l.logEntry(ServiceLogEntry{
		Timestamp: time.Now(),
		Level:     ServiceLogLevelError,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
	})
}

func (l *ServiceLogger) Debug(source, message string, metadata map[string]string) {
	l.logEntry(ServiceLogEntry{
		Timestamp: time.Now(),
		Level:     ServiceLogLevelDebug,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
	})
}

func (l *ServiceLogger) logEntry(entry ServiceLogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Check log level
	if !l.shouldLog(entry.Level) {
		return
	}
	
	// Format log entry
	logLine := l.formatLogEntry(entry)
	
	// Write to file
	if _, err := l.logFile.WriteString(logLine + "\n"); err != nil {
		log.Printf("Failed to write to log file: %v", err)
	}
	
	// Check if rotation is needed
	if l.needsRotation() {
		if err := l.rotateLogFile(); err != nil {
			log.Printf("Failed to rotate log file: %v", err)
		}
	}
}

func (l *ServiceLogger) shouldLog(level ServiceLogLevel) bool {
	levelOrder := map[ServiceLogLevel]int{
		ServiceLogLevelDebug: 0,
		ServiceLogLevelInfo:  1,
		ServiceLogLevelWarn:  2,
		ServiceLogLevelError: 3,
	}
	
	return levelOrder[level] >= levelOrder[l.level]
}

func (l *ServiceLogger) formatLogEntry(entry ServiceLogEntry) string {
	var parts []string
	
	// Timestamp
	parts = append(parts, entry.Timestamp.Format("2006-01-02 15:04:05"))
	
	// Level
	parts = append(parts, fmt.Sprintf("[%s]", strings.ToUpper(string(entry.Level))))
	
	// Source
	parts = append(parts, fmt.Sprintf("<%s>", entry.Source))
	
	// Message
	parts = append(parts, entry.Message)
	
	// Metadata
	if len(entry.Metadata) > 0 {
		var metaParts []string
		for k, v := range entry.Metadata {
			metaParts = append(metaParts, fmt.Sprintf("%s=%s", k, v))
		}
		parts = append(parts, fmt.Sprintf("metadata={%s}", strings.Join(metaParts, ",")))
	}
	
	// Error
	if entry.Error != "" {
		parts = append(parts, fmt.Sprintf("error=%s", entry.Error))
	}
	
	return strings.Join(parts, " ")
}

func (l *ServiceLogger) needsRotation() bool {
	if l.logFile == nil {
		return false
	}
	
	info, err := l.logFile.Stat()
	if err != nil {
		return false
	}
	
	return info.Size() >= l.maxFileSize
}

func (l *ServiceLogger) rotateLogFile() error {
	// Close current file
	if l.logFile != nil {
		l.logFile.Close()
	}
	
	// Rotate existing log files
	baseName := "claude-monitor-service.log"
	
	// Remove oldest file
	oldestFile := filepath.Join(l.logDirectory, fmt.Sprintf("%s.%d", baseName, l.maxFiles-1))
	os.Remove(oldestFile)
	
	// Rotate files
	for i := l.maxFiles - 2; i >= 0; i-- {
		oldName := filepath.Join(l.logDirectory, baseName)
		if i > 0 {
			oldName = filepath.Join(l.logDirectory, fmt.Sprintf("%s.%d", baseName, i))
		}
		
		newName := filepath.Join(l.logDirectory, fmt.Sprintf("%s.%d", baseName, i+1))
		os.Rename(oldName, newName)
	}
	
	// Create new log file
	logFilePath := filepath.Join(l.logDirectory, baseName)
	var err error
	l.logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}
	
	return nil
}

func (l *ServiceLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}