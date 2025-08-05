package daemon

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-017
 * CONTEXT:   Enhanced daemon manager with sophisticated goroutine coordination and health monitoring
 * REASON:    Need robust orchestration with graceful error handling and monitoring capabilities
 * CHANGE:    Enhanced with health monitoring, timeout management, and improved error recovery.
 * PREVENTION:Implement circuit breakers, health checks, and proper resource cleanup patterns
 * RISK:      High - Daemon manager failure affects entire system operation
 */

// DefaultDaemonManager implements enhanced daemon management with monitoring and health checks
type DefaultDaemonManager struct {
	container       *arch.ServiceContainer
	logger          arch.Logger
	ebpfManager     arch.EBPFManager
	sessionManager  arch.SessionManager
	workBlockMgr    arch.WorkBlockManager
	eventProcessor  arch.EventProcessor
	dbManager       arch.DatabaseManager
	
	// Daemon state with atomic access
	startTime       time.Time
	running         int32 // atomic bool
	healthy         int32 // atomic bool
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	
	// Health monitoring
	healthCheckInterval time.Duration
	timeoutCheckInterval time.Duration
	lastHealthCheck     int64 // atomic Unix timestamp
	
	// Statistics
	restartCount        int64 // atomic counter
	healthCheckCount    int64 // atomic counter
	errorCount          int64 // atomic counter
}

// NewDaemonManager creates a new daemon manager with enhanced monitoring capabilities
func NewDaemonManager(container *arch.ServiceContainer, logger arch.Logger) (*DefaultDaemonManager, error) {
	if container == nil {
		return nil, fmt.Errorf("container cannot be nil")
	}
	
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	
	return &DefaultDaemonManager{
		container:            container,
		logger:               logger,
		running:              0,
		healthy:              0,
		healthCheckInterval:  30 * time.Second,
		timeoutCheckInterval: 60 * time.Second,
		lastHealthCheck:      0,
		restartCount:         0,
		healthCheckCount:     0,
		errorCount:           0,
	}, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-018
 * CONTEXT:   Enhanced daemon startup with health monitoring and robust error handling
 * REASON:    Need coordinated startup with comprehensive monitoring and graceful failure handling
 * CHANGE:    Enhanced with health monitoring goroutines and improved error recovery.
 * PREVENTION:Validate each startup step, implement rollback on failures, start monitoring early
 * RISK:      High - Startup failures could leave system in inconsistent state
 */

// Start initializes and starts all daemon components with enhanced monitoring
func (ddm *DefaultDaemonManager) Start() error {
	ddm.mu.Lock()
	defer ddm.mu.Unlock()
	
	if atomic.LoadInt32(&ddm.running) == 1 {
		return fmt.Errorf("daemon already running")
	}
	
	ddm.logger.Info("Starting enhanced daemon manager", 
		"healthCheckInterval", ddm.healthCheckInterval,
		"timeoutCheckInterval", ddm.timeoutCheckInterval)
	
	// Create context for daemon lifecycle
	ddm.ctx, ddm.cancel = context.WithCancel(context.Background())
	ddm.startTime = time.Now()
	
	// Initialize all services from container
	if err := ddm.initializeServices(); err != nil {
		ddm.incrementErrorCount()
		return fmt.Errorf("failed to initialize services: %w", err)
	}
	
	// Start database manager with health check
	if err := ddm.dbManager.Initialize(); err != nil {
		ddm.incrementErrorCount()
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	// Verify database health immediately
	if err := ddm.dbManager.HealthCheck(); err != nil {
		ddm.incrementErrorCount()
		ddm.logger.Warn("Database health check failed during startup", "error", err)
	}
	
	// Start eBPF monitoring
	if err := ddm.ebpfManager.LoadPrograms(); err != nil {
		ddm.incrementErrorCount()
		return fmt.Errorf("failed to load eBPF programs: %w", err)
	}
	
	// Create and start event processor
	eventCh := ddm.ebpfManager.GetEventChannel()
	ddm.eventProcessor = arch.NewDefaultEventProcessor(eventCh, ddm.logger)
	
	// Register event handlers
	if err := ddm.registerEventHandlers(); err != nil {
		ddm.incrementErrorCount()
		return fmt.Errorf("failed to register event handlers: %w", err)
	}
	
	// Start event processing
	if err := ddm.eventProcessor.Start(ddm.ctx); err != nil {
		ddm.incrementErrorCount()
		return fmt.Errorf("failed to start event processor: %w", err)
	}
	
	// Start eBPF event capture
	if err := ddm.ebpfManager.StartEventProcessing(ddm.ctx); err != nil {
		ddm.incrementErrorCount()
		return fmt.Errorf("failed to start eBPF event processing: %w", err)
	}
	
	// Start monitoring goroutines
	ddm.startMonitoringGoroutines()
	
	// Mark as running and healthy
	atomic.StoreInt32(&ddm.running, 1)
	atomic.StoreInt32(&ddm.healthy, 1)
	atomic.StoreInt64(&ddm.lastHealthCheck, time.Now().Unix())
	
	ddm.logger.Info("Daemon manager started successfully with monitoring", 
		"uptime", time.Since(ddm.startTime),
		"monitoringGoroutines", 3)
	
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-019
 * CONTEXT:   Background monitoring goroutines for health checks and timeout management
 * REASON:    Need continuous monitoring of system health and automatic timeout handling
 * CHANGE:    Added health monitoring, timeout checking, and session monitoring goroutines.
 * PREVENTION:Handle goroutine panics, implement proper shutdown coordination, avoid resource leaks
 * RISK:      Medium - Monitoring goroutine failures could affect system reliability
 */

// startMonitoringGoroutines starts background monitoring tasks
func (ddm *DefaultDaemonManager) startMonitoringGoroutines() {
	// Health monitoring goroutine
	ddm.wg.Add(1)
	go ddm.healthMonitor()
	
	// Timeout checking goroutine
	ddm.wg.Add(1)
	go ddm.timeoutMonitor()
	
	// Session expiry monitoring goroutine
	ddm.wg.Add(1)
	go ddm.sessionExpiryMonitor()
	
	ddm.logger.Debug("Started monitoring goroutines", 
		"healthCheckInterval", ddm.healthCheckInterval,
		"timeoutCheckInterval", ddm.timeoutCheckInterval)
}

// healthMonitor continuously monitors system component health
func (ddm *DefaultDaemonManager) healthMonitor() {
	defer ddm.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			ddm.logger.Error("Health monitor panic recovered", "panic", r)
			ddm.incrementErrorCount()
		}
	}()
	
	ddm.logger.Debug("Health monitor started")
	ticker := time.NewTicker(ddm.healthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ddm.ctx.Done():
			ddm.logger.Debug("Health monitor stopping")
			return
			
		case <-ticker.C:
			ddm.performHealthCheck()
		}
	}
}

// timeoutMonitor continuously checks for work block timeouts
func (ddm *DefaultDaemonManager) timeoutMonitor() {
	defer ddm.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			ddm.logger.Error("Timeout monitor panic recovered", "panic", r)
			ddm.incrementErrorCount()
		}
	}()
	
	ddm.logger.Debug("Timeout monitor started")
	ticker := time.NewTicker(ddm.timeoutCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ddm.ctx.Done():
			ddm.logger.Debug("Timeout monitor stopping")
			return
			
		case <-ticker.C:
			ddm.checkWorkBlockTimeouts()
		}
	}
}

// sessionExpiryMonitor continuously monitors for session expiry
func (ddm *DefaultDaemonManager) sessionExpiryMonitor() {
	defer ddm.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			ddm.logger.Error("Session expiry monitor panic recovered", "panic", r)
			ddm.incrementErrorCount()
		}
	}()
	
	ddm.logger.Debug("Session expiry monitor started")
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ddm.ctx.Done():
			ddm.logger.Debug("Session expiry monitor stopping")
			return
			
		case <-ticker.C:
			ddm.checkSessionExpiry()
		}
	}
}

// initializeServices retrieves all services from the container
func (ddm *DefaultDaemonManager) initializeServices() error {
	var err error
	
	// Get database manager
	ddm.dbManager, err = ddm.container.GetDatabaseManager()
	if err != nil {
		return fmt.Errorf("failed to get database manager: %w", err)
	}
	
	// Get eBPF manager
	ddm.ebpfManager, err = ddm.container.GetEBPFManager()
	if err != nil {
		return fmt.Errorf("failed to get eBPF manager: %w", err)
	}
	
	// Get session manager
	ddm.sessionManager, err = ddm.container.GetSessionManager()
	if err != nil {
		return fmt.Errorf("failed to get session manager: %w", err)
	}
	
	// Get work block manager
	service, err := ddm.container.Get((*arch.WorkBlockManager)(nil))
	if err != nil {
		return fmt.Errorf("failed to get work block manager: %w", err)
	}
	
	var ok bool
	ddm.workBlockMgr, ok = service.(arch.WorkBlockManager)
	if !ok {
		return fmt.Errorf("service is not a WorkBlockManager")
	}
	
	return nil
}

// registerEventHandlers registers business logic event handlers
func (ddm *DefaultDaemonManager) registerEventHandlers() error {
	// Register session interaction handler
	sessionHandler := NewSessionEventHandler(ddm.sessionManager, ddm.workBlockMgr, ddm.logger)
	if err := ddm.eventProcessor.RegisterHandler(sessionHandler); err != nil {
		return fmt.Errorf("failed to register session handler: %w", err)
	}
	
	// Register process tracking handler
	processHandler := NewProcessEventHandler(ddm.dbManager, ddm.sessionManager, ddm.logger)
	if err := ddm.eventProcessor.RegisterHandler(processHandler); err != nil {
		return fmt.Errorf("failed to register process handler: %w", err)
	}
	
	return nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-035
 * CONTEXT:   Daemon shutdown with proper resource cleanup and coordination
 * REASON:    Need graceful shutdown that ensures all resources are properly released
 * CHANGE:    Initial implementation.
 * PREVENTION:Always stop services in reverse dependency order and handle errors
 * RISK:      Medium - Improper shutdown could cause resource leaks or data corruption
 */

// Stop gracefully shuts down all daemon components with enhanced coordination
func (ddm *DefaultDaemonManager) Stop() error {
	ddm.mu.Lock()
	defer ddm.mu.Unlock()
	
	if atomic.LoadInt32(&ddm.running) == 0 {
		ddm.logger.Debug("Daemon already stopped")
		return nil
	}
	
	uptime := time.Since(ddm.startTime)
	ddm.logger.Info("Stopping daemon manager", 
		"uptime", uptime,
		"errorCount", atomic.LoadInt64(&ddm.errorCount),
		"healthCheckCount", atomic.LoadInt64(&ddm.healthCheckCount))
	
	var errors []error
	
	// Mark as unhealthy and not running
	atomic.StoreInt32(&ddm.healthy, 0)
	atomic.StoreInt32(&ddm.running, 0)
	
	// Cancel context to signal shutdown to all goroutines
	if ddm.cancel != nil {
		ddm.cancel()
	}
	
	// Stop event processor first to prevent new events
	if ddm.eventProcessor != nil {
		ddm.logger.Debug("Stopping event processor")
		if err := ddm.eventProcessor.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("event processor stop error: %w", err))
			ddm.incrementErrorCount()
		}
	}
	
	// Stop eBPF manager to stop event generation
	if ddm.ebpfManager != nil {
		ddm.logger.Debug("Stopping eBPF manager")
		if err := ddm.ebpfManager.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("eBPF manager stop error: %w", err))
			ddm.incrementErrorCount()
		}
	}
	
	// Wait for monitoring goroutines to stop with timeout
	ddm.logger.Debug("Waiting for monitoring goroutines to stop")
	done := make(chan struct{})
	go func() {
		ddm.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		ddm.logger.Debug("All monitoring goroutines stopped")
	case <-time.After(10 * time.Second):
		ddm.logger.Warn("Timeout waiting for monitoring goroutines")
		errors = append(errors, fmt.Errorf("monitoring goroutine shutdown timeout"))
	}
	
	// Finalize current session and work block
	ddm.logger.Debug("Finalizing current session and work blocks")
	if ddm.sessionManager != nil {
		if err := ddm.sessionManager.FinalizeCurrentSession(); err != nil {
			errors = append(errors, fmt.Errorf("session finalization error: %w", err))
			ddm.incrementErrorCount()
		}
	}
	
	if ddm.workBlockMgr != nil {
		if err := ddm.workBlockMgr.FinalizeCurrentBlock(); err != nil {
			errors = append(errors, fmt.Errorf("work block finalization error: %w", err))
			ddm.incrementErrorCount()
		}
	}
	
	// Close database last
	if ddm.dbManager != nil {
		ddm.logger.Debug("Closing database")
		if err := ddm.dbManager.Close(); err != nil {
			errors = append(errors, fmt.Errorf("database close error: %w", err))
			ddm.incrementErrorCount()
		}
	}
	
	if len(errors) > 0 {
		ddm.logger.Error("Errors during daemon shutdown", 
			"errorCount", len(errors),
			"errors", errors,
			"uptime", uptime)
		return fmt.Errorf("shutdown errors: %v", errors)
	}
	
	ddm.logger.Info("Daemon manager stopped successfully", 
		"uptime", uptime,
		"totalErrorCount", atomic.LoadInt64(&ddm.errorCount),
		"totalHealthChecks", atomic.LoadInt64(&ddm.healthCheckCount))
	return nil
}

// GetStatus returns current daemon status
func (ddm *DefaultDaemonManager) GetStatus() (*domain.SystemStatus, error) {
	ddm.mu.RLock()
	defer ddm.mu.RUnlock()
	
	status := &domain.SystemStatus{
		DaemonRunning:    atomic.LoadInt32(&ddm.running) == 1,
		MonitoringActive: atomic.LoadInt32(&ddm.running) == 1,
		Uptime:          time.Since(ddm.startTime),
	}
	
	// Get current session
	if ddm.sessionManager != nil && atomic.LoadInt32(&ddm.running) == 1 {
		if session, exists := ddm.sessionManager.GetCurrentSession(); exists {
			status.CurrentSession = session
		}
	}
	
	// Get current work block
	if ddm.workBlockMgr != nil && atomic.LoadInt32(&ddm.running) == 1 {
		if block, exists := ddm.workBlockMgr.GetActiveBlock(); exists {
			status.CurrentWorkBlock = block
			status.LastActivity = &block.LastActivity
		}
	}
	
	// Get event processing stats
	if ddm.eventProcessor != nil {
		if stats := ddm.eventProcessor.GetStats(); stats != nil {
			status.EventsProcessed = stats.EventsProcessed
		}
	}
	
	return status, nil
}

// IsRunning returns true if daemon is currently running
func (ddm *DefaultDaemonManager) IsRunning() bool {
	return atomic.LoadInt32(&ddm.running) == 1
}

// Restart performs a graceful restart with enhanced monitoring
func (ddm *DefaultDaemonManager) Restart() error {
	ddm.logger.Info("Restarting daemon manager")
	
	if err := ddm.Stop(); err != nil {
		ddm.incrementErrorCount()
		return fmt.Errorf("failed to stop daemon: %w", err)
	}
	
	// Wait briefly for cleanup
	time.Sleep(2 * time.Second)
	
	if err := ddm.Start(); err != nil {
		ddm.incrementErrorCount()
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	
	// Increment restart counter
	atomic.AddInt64(&ddm.restartCount, 1)
	
	ddm.logger.Info("Daemon restart completed", 
		"restartCount", atomic.LoadInt64(&ddm.restartCount))
	
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-021
 * CONTEXT:   Utility methods for health monitoring and statistics
 * REASON:    Need comprehensive monitoring capabilities for system health and performance
 * CHANGE:    Added health check methods, statistics tracking, and monitoring utilities.
 * PREVENTION:Keep monitoring lightweight, handle component failures gracefully
 * RISK:      Low - Monitoring utilities are defensive and improve system observability
 */

// performHealthCheck checks the health of all system components
func (ddm *DefaultDaemonManager) performHealthCheck() {
	atomic.AddInt64(&ddm.healthCheckCount, 1)
	atomic.StoreInt64(&ddm.lastHealthCheck, time.Now().Unix())
	
	healthy := true
	var issues []string
	
	// Check database health
	if ddm.dbManager != nil {
		if err := ddm.dbManager.HealthCheck(); err != nil {
			issues = append(issues, fmt.Sprintf("database: %v", err))
			healthy = false
			ddm.incrementErrorCount()
		}
	}
	
	// Check eBPF manager health
	if ddm.ebpfManager != nil {
		if stats, err := ddm.ebpfManager.GetStats(); err != nil {
			issues = append(issues, fmt.Sprintf("ebpf: %v", err))
			healthy = false
			ddm.incrementErrorCount()
		} else if stats.DroppedEvents > 0 {
			ddm.logger.Warn("eBPF events being dropped", 
				"droppedEvents", stats.DroppedEvents,
				"processedEvents", stats.EventsProcessed)
		}
	}
	
	// Check event processor health
	if ddm.eventProcessor != nil {
		if stats := ddm.eventProcessor.GetStats(); stats != nil {
			if stats.ProcessingErrors > 0 {
				ddm.logger.Warn("Event processing errors detected", 
					"processingErrors", stats.ProcessingErrors,
					"eventsProcessed", stats.EventsProcessed)
			}
		}
	}
	
	// Update health status
	if healthy {
		atomic.StoreInt32(&ddm.healthy, 1)
		ddm.logger.Debug("Health check passed", 
			"healthCheckCount", atomic.LoadInt64(&ddm.healthCheckCount))
	} else {
		atomic.StoreInt32(&ddm.healthy, 0)
		ddm.logger.Error("Health check failed", 
			"issues", issues,
			"healthCheckCount", atomic.LoadInt64(&ddm.healthCheckCount))
	}
}

// checkWorkBlockTimeouts checks for work block inactivity timeouts
func (ddm *DefaultDaemonManager) checkWorkBlockTimeouts() {
	if ddm.workBlockMgr == nil {
		return
	}
	
	currentTime := time.Now()
	if err := ddm.workBlockMgr.CheckInactivityTimeout(currentTime); err != nil {
		ddm.logger.Error("Work block timeout check failed", "error", err)
		ddm.incrementErrorCount()
	} else {
		ddm.logger.Debug("Work block timeout check completed", "time", currentTime)
	}
}

// checkSessionExpiry checks for session expiry
func (ddm *DefaultDaemonManager) checkSessionExpiry() {
	if ddm.sessionManager == nil {
		return
	}
	
	if session, exists := ddm.sessionManager.GetCurrentSession(); exists {
		if session.IsExpired(time.Now()) {
			ddm.logger.Info("Session expired, finalizing", 
				"sessionID", session.ID,
				"endTime", session.EndTime)
			
			if err := ddm.sessionManager.FinalizeCurrentSession(); err != nil {
				ddm.logger.Error("Failed to finalize expired session", "error", err)
				ddm.incrementErrorCount()
			}
			
			// Also finalize any active work blocks
			if err := ddm.workBlockMgr.FinalizeCurrentBlock(); err != nil {
				ddm.logger.Error("Failed to finalize work block on session expiry", "error", err)
				ddm.incrementErrorCount()
			}
		}
	}
}

// incrementErrorCount atomically increments the error counter
func (ddm *DefaultDaemonManager) incrementErrorCount() {
	atomic.AddInt64(&ddm.errorCount, 1)
}

// IsHealthy returns true if the daemon is currently healthy
func (ddm *DefaultDaemonManager) IsHealthy() bool {
	return atomic.LoadInt32(&ddm.healthy) == 1
}

// GetDaemonStats returns comprehensive daemon statistics
func (ddm *DefaultDaemonManager) GetDaemonStats() map[string]interface{} {
	stats := map[string]interface{}{
		"running":           ddm.IsRunning(),
		"healthy":           ddm.IsHealthy(),
		"uptime":            time.Since(ddm.startTime),
		"errorCount":        atomic.LoadInt64(&ddm.errorCount),
		"restartCount":      atomic.LoadInt64(&ddm.restartCount),
		"healthCheckCount":  atomic.LoadInt64(&ddm.healthCheckCount),
		"lastHealthCheck":   time.Unix(atomic.LoadInt64(&ddm.lastHealthCheck), 0),
	}
	
	// Add component-specific stats if available
	if ddm.eventProcessor != nil {
		if procStats := ddm.eventProcessor.GetStats(); procStats != nil {
			stats["eventProcessor"] = procStats
		}
	}
	
	if ddm.ebpfManager != nil {
		if ebpfStats, err := ddm.ebpfManager.GetStats(); err == nil {
			stats["ebpfManager"] = ebpfStats
		}
	}
	
	return stats
}