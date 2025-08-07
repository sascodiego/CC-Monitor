/**
 * CONTEXT:   Activity event processing pipeline coordinating all business logic components
 * INPUT:     Activity events from HTTP handlers requiring full business logic processing
 * OUTPUT:    Coordinated session, work block, and project updates with proper state management
 * BUSINESS:  Central orchestrator for all Claude Monitor business rules and entity coordination
 * CHANGE:    Initial event processor implementation following Clean Architecture principles
 * RISK:      High - Core business logic coordinator affecting all work tracking functionality
 */

package usecases

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/claude-monitor/system/internal/usecases/repositories"
)

/**
 * CONTEXT:   Main event processor coordinating session, work block, and project management
 * INPUT:     Use case managers and repositories for complete business logic coordination
 * OUTPUT:    Event processing operations with full entity lifecycle management
 * BUSINESS:  Process activity events through complete business logic pipeline
 * CHANGE:    Initial event processor implementation with dependency injection
 * RISK:      High - Central coordination point for all business logic operations
 */
type EventProcessor struct {
	sessionManager  *SessionManager
	workBlockManager *WorkBlockManager
	projectManager  *ProjectManager
	activityRepo    repositories.ActivityRepository
	logger          *slog.Logger
	
	// Statistics and state
	mu                sync.RWMutex
	processedEvents   int64
	lastActivity      time.Time
	activeSessions    map[string]*entities.Session // userID -> session
	activeWorkBlocks  map[string]*entities.WorkBlock // sessionID -> work block
	startTime         time.Time
}

// EventProcessorConfig holds configuration for event processor
type EventProcessorConfig struct {
	SessionManager   *SessionManager
	WorkBlockManager *WorkBlockManager
	ProjectManager   *ProjectManager
	ActivityRepo     repositories.ActivityRepository
	Logger           *slog.Logger
}

// SystemStatus represents overall system health and statistics
type SystemStatus struct {
	Status           string            `json:"status"`
	Uptime           time.Duration     `json:"uptime"`
	Version          string            `json:"version"`
	ActiveSessions   int               `json:"active_sessions"`
	ActiveWorkBlocks int               `json:"active_work_blocks"`
	TotalActivities  int64             `json:"total_activities"`
	DatabaseStatus   string            `json:"database_status"`
	LastActivity     time.Time         `json:"last_activity"`
	Metrics          map[string]string `json:"metrics"`
}

/**
 * CONTEXT:   Factory function for creating event processor with all dependencies
 * INPUT:     EventProcessorConfig with all required use case managers and repositories
 * OUTPUT:    Configured EventProcessor instance ready for activity processing
 * BUSINESS:  Event processor requires all business logic components for complete processing
 * CHANGE:    Initial factory implementation with dependency injection validation
 * RISK:      Medium - Factory function with extensive dependency validation
 */
func NewEventProcessor(config EventProcessorConfig) (*EventProcessor, error) {
	if config.SessionManager == nil {
		return nil, fmt.Errorf("session manager cannot be nil")
	}
	
	if config.WorkBlockManager == nil {
		return nil, fmt.Errorf("work block manager cannot be nil")
	}
	
	if config.ProjectManager == nil {
		return nil, fmt.Errorf("project manager cannot be nil")
	}
	
	if config.ActivityRepo == nil {
		return nil, fmt.Errorf("activity repository cannot be nil")
	}
	
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	
	ep := &EventProcessor{
		sessionManager:   config.SessionManager,
		workBlockManager: config.WorkBlockManager,
		projectManager:   config.ProjectManager,
		activityRepo:     config.ActivityRepo,
		logger:           logger,
		activeSessions:   make(map[string]*entities.Session),
		activeWorkBlocks: make(map[string]*entities.WorkBlock),
		startTime:        time.Now(),
	}
	
	return ep, nil
}

/**
 * CONTEXT:   Main activity event processing pipeline with complete business logic
 * INPUT:     Activity event requiring session, work block, and project coordination
 * OUTPUT:    Fully processed activity with updated sessions, work blocks, and projects
 * BUSINESS:  Coordinate all business rules: 5-hour sessions, 5-minute idle, project detection
 * CHANGE:    Initial processing pipeline with complete business logic integration
 * RISK:      High - Core processing logic affecting all work tracking accuracy
 */
func (ep *EventProcessor) ProcessActivity(ctx context.Context, event *entities.ActivityEvent) error {
	startTime := time.Now()
	
	ep.logger.Info("Processing activity event",
		"activityID", event.ID(),
		"userID", event.UserID(),
		"projectName", event.ProjectName(),
		"timestamp", event.Timestamp())
	
	// Step 1: Get or create project
	project, err := ep.projectManager.GetOrCreateProject(ctx, event.ProjectPath(), event.ProjectName())
	if err != nil {
		return fmt.Errorf("failed to get or create project: %w", err)
	}
	
	ep.logger.Debug("Project resolved",
		"projectID", project.ID(),
		"projectName", project.Name(),
		"projectPath", project.Path())
	
	// Step 2: Get or create session (5-hour window logic)
	session, err := ep.sessionManager.GetOrCreateSession(ctx, event.UserID(), event.Timestamp())
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}
	
	// Associate event with session
	if err := event.AssociateWithSession(session.ID()); err != nil {
		return fmt.Errorf("failed to associate event with session: %w", err)
	}
	
	ep.logger.Debug("Session resolved",
		"sessionID", session.ID(),
		"userID", event.UserID(),
		"sessionStart", session.StartTime())
	
	// Step 3: Get or create work block (5-minute idle logic)
	workBlock, err := ep.workBlockManager.StartWorkBlock(ctx, session.ID(), project.ID(), event.Timestamp())
	if err != nil {
		return fmt.Errorf("failed to start work block: %w", err)
	}
	
	// Associate event with work block
	if err := event.AssociateWithWorkBlock(workBlock.ID()); err != nil {
		return fmt.Errorf("failed to associate event with work block: %w", err)
	}
	
	ep.logger.Debug("Work block resolved",
		"workBlockID", workBlock.ID(),
		"sessionID", session.ID(),
		"projectID", project.ID())
	
	// Step 4: Update project activity
	if err := ep.projectManager.UpdateProjectActivity(ctx, project.ID(), event.Timestamp()); err != nil {
		ep.logger.Warn("Failed to update project activity", "error", err)
		// Don't fail the whole request for project activity update
	}
	
	// Step 5: Save activity event
	if err := ep.activityRepo.Save(ctx, event); err != nil {
		return fmt.Errorf("failed to save activity event: %w", err)
	}
	
	// Step 6: Update internal state and statistics
	ep.updateProcessingStatistics(event.UserID(), session, workBlock)
	
	processingTime := time.Since(startTime)
	ep.logger.Info("Activity event processed successfully",
		"activityID", event.ID(),
		"sessionID", session.ID(),
		"workBlockID", workBlock.ID(),
		"projectID", project.ID(),
		"processingTime", processingTime)
	
	return nil
}

/**
 * CONTEXT:   Get current system status for health checks and monitoring
 * INPUT:     Context for status collection operations
 * OUTPUT:    SystemStatus with current statistics and health information
 * BUSINESS:  Provide system visibility for monitoring and operational decisions
 * CHANGE:    Initial system status implementation with comprehensive metrics
 * RISK:      Low - Read-only status collection with performance metrics
 */
func (ep *EventProcessor) GetSystemStatus(ctx context.Context) (*SystemStatus, error) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()
	
	// Calculate uptime
	uptime := time.Since(ep.startTime)
	
	// Get active counts
	activeSessions := len(ep.activeSessions)
	activeWorkBlocks := len(ep.activeWorkBlocks)
	
	// Create metrics map
	metrics := make(map[string]string)
	metrics["processed_events"] = fmt.Sprintf("%d", ep.processedEvents)
	metrics["uptime_seconds"] = fmt.Sprintf("%.0f", uptime.Seconds())
	
	if ep.processedEvents > 0 {
		avgProcessingTime := uptime.Nanoseconds() / ep.processedEvents
		metrics["avg_processing_ns"] = fmt.Sprintf("%d", avgProcessingTime)
	}
	
	// Determine overall status
	status := "healthy"
	dbStatus := "connected" // This would check actual DB connection
	
	systemStatus := &SystemStatus{
		Status:           status,
		Uptime:           uptime,
		Version:          "1.0.0", // This would come from build info
		ActiveSessions:   activeSessions,
		ActiveWorkBlocks: activeWorkBlocks,
		TotalActivities:  ep.processedEvents,
		DatabaseStatus:   dbStatus,
		LastActivity:     ep.lastActivity,
		Metrics:          metrics,
	}
	
	ep.logger.Debug("System status collected",
		"status", status,
		"activeSessions", activeSessions,
		"activeWorkBlocks", activeWorkBlocks,
		"totalActivities", ep.processedEvents)
	
	return systemStatus, nil
}

/**
 * CONTEXT:   Get active session for specific user
 * INPUT:     User ID for session lookup
 * OUTPUT:    Active session entity or nil if no active session exists
 * BUSINESS:  Support status queries and session management operations
 * CHANGE:    Initial active session lookup with cache and database fallback
 * RISK:      Low - Read-only session lookup operation
 */
func (ep *EventProcessor) GetActiveSession(ctx context.Context, userID string) (*entities.Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}
	
	// Check cache first
	ep.mu.RLock()
	cachedSession, hasCached := ep.activeSessions[userID]
	ep.mu.RUnlock()
	
	if hasCached {
		// Verify session is still active
		if cachedSession.IsActive() {
			return cachedSession, nil
		}
		
		// Remove expired session from cache
		ep.mu.Lock()
		delete(ep.activeSessions, userID)
		ep.mu.Unlock()
	}
	
	// Get from session manager
	session, err := ep.sessionManager.GetOrCreateSession(ctx, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}
	
	// Update cache
	ep.mu.Lock()
	ep.activeSessions[userID] = session
	ep.mu.Unlock()
	
	return session, nil
}

/**
 * CONTEXT:   Get active work block for specific session
 * INPUT:     Session ID for work block lookup
 * OUTPUT:    Active work block entity or nil if no active work block exists
 * BUSINESS:  Support status queries and work block management operations
 * CHANGE:    Initial active work block lookup with cache and database fallback
 * RISK:      Low - Read-only work block lookup operation
 */
func (ep *EventProcessor) GetActiveWorkBlock(ctx context.Context, sessionID string) (*entities.WorkBlock, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}
	
	// Check cache first
	ep.mu.RLock()
	cachedWorkBlock, hasCached := ep.activeWorkBlocks[sessionID]
	ep.mu.RUnlock()
	
	if hasCached {
		// Verify work block is still active
		if cachedWorkBlock.IsActive() && !cachedWorkBlock.IsIdle(time.Now()) {
			return cachedWorkBlock, nil
		}
		
		// Remove inactive work block from cache
		ep.mu.Lock()
		delete(ep.activeWorkBlocks, sessionID)
		ep.mu.Unlock()
	}
	
	// Get from work block manager
	workBlock, err := ep.workBlockManager.GetActiveWorkBlock(ctx, sessionID)
	if err != nil {
		return nil, err // This might be a "not found" error which is valid
	}
	
	// Update cache
	ep.mu.Lock()
	ep.activeWorkBlocks[sessionID] = workBlock
	ep.mu.Unlock()
	
	return workBlock, nil
}

/**
 * CONTEXT:   Start background cleanup processes for expired sessions and idle work blocks
 * INPUT:     Context for cleanup lifecycle control
 * OUTPUT:    Background cleanup processes running with proper coordination
 * BUSINESS:  Maintain system health through automatic cleanup of expired entities
 * CHANGE:    Initial background cleanup coordination with all managers
 * RISK:      Medium - Background processes affecting system state and performance
 */
func (ep *EventProcessor) StartBackgroundCleanup(ctx context.Context) {
	ep.logger.Info("Starting background cleanup processes")
	
	// Start session cleanup (every 5 minutes)
	go ep.sessionCleanupWorker(ctx)
	
	// Start work block cleanup (every 2 minutes)
	go ep.workBlockCleanupWorker(ctx)
	
	// Start cache cleanup (every 10 minutes)
	go ep.cacheCleanupWorker(ctx)
}

/**
 * CONTEXT:   Stop event processor gracefully with proper cleanup
 * INPUT:     Context for graceful shutdown operations
 * OUTPUT:    Clean shutdown with finalized work blocks and closed resources
 * BUSINESS:  Ensure data integrity during shutdown with proper work block finalization
 * CHANGE:    Initial graceful shutdown implementation with state finalization
 * RISK:      High - Shutdown logic affects data integrity and work time accuracy
 */
func (ep *EventProcessor) Stop(ctx context.Context) error {
	ep.logger.Info("Stopping event processor gracefully")
	
	// Finalize all active work blocks
	ep.mu.Lock()
	activeWorkBlocks := make([]*entities.WorkBlock, 0, len(ep.activeWorkBlocks))
	for _, workBlock := range ep.activeWorkBlocks {
		activeWorkBlocks = append(activeWorkBlocks, workBlock)
	}
	ep.mu.Unlock()
	
	// Finalize work blocks with appropriate end time
	finalizeTime := time.Now()
	finalizedCount := 0
	
	for _, workBlock := range activeWorkBlocks {
		if err := ep.workBlockManager.FinalizeWorkBlock(ctx, workBlock.ID(), finalizeTime); err != nil {
			ep.logger.Error("Failed to finalize work block during shutdown",
				"workBlockID", workBlock.ID(),
				"error", err)
		} else {
			finalizedCount++
		}
	}
	
	ep.logger.Info("Event processor stopped",
		"finalizedWorkBlocks", finalizedCount,
		"totalProcessedEvents", ep.processedEvents)
	
	return nil
}

// Private helper methods

/**
 * CONTEXT:   Update internal processing statistics and cache state
 * INPUT:     User ID, session, and work block from successful processing
 * OUTPUT:    Updated internal statistics and cache state
 * BUSINESS:  Maintain accurate processing metrics and entity caches
 * CHANGE:    Internal statistics and cache management helper
 * RISK:      Low - Internal state management with proper synchronization
 */
func (ep *EventProcessor) updateProcessingStatistics(userID string, session *entities.Session, workBlock *entities.WorkBlock) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	
	// Update statistics
	ep.processedEvents++
	ep.lastActivity = time.Now()
	
	// Update caches
	ep.activeSessions[userID] = session
	ep.activeWorkBlocks[session.ID()] = workBlock
}

/**
 * CONTEXT:   Background worker for session cleanup and expiration handling
 * INPUT:     Context for worker lifecycle control
 * OUTPUT:    Periodic session cleanup with expired session removal
 * BUSINESS:  Maintain accurate session states through automatic cleanup
 * CHANGE:    Background session cleanup worker implementation
 * RISK:      Medium - Session cleanup affects system state and cache
 */
func (ep *EventProcessor) sessionCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	ep.logger.Info("Session cleanup worker started")
	
	for {
		select {
		case <-ctx.Done():
			ep.logger.Info("Session cleanup worker stopped")
			return
		case <-ticker.C:
			cleanedCount, err := ep.sessionManager.CloseExpiredSessions(ctx, time.Now())
			if err != nil {
				ep.logger.Error("Session cleanup failed", "error", err)
			} else if cleanedCount > 0 {
				ep.logger.Info("Session cleanup completed", "closedSessions", cleanedCount)
				
				// Clean up cache
				ep.cleanupSessionCache()
			}
		}
	}
}

/**
 * CONTEXT:   Background worker for work block cleanup and idle detection
 * INPUT:     Context for worker lifecycle control
 * OUTPUT:    Periodic work block cleanup with idle block finalization
 * BUSINESS:  Maintain accurate work block states through automatic cleanup
 * CHANGE:    Background work block cleanup worker implementation
 * RISK:      Medium - Work block cleanup affects work time calculations
 */
func (ep *EventProcessor) workBlockCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	
	ep.logger.Info("Work block cleanup worker started")
	
	for {
		select {
		case <-ctx.Done():
			ep.logger.Info("Work block cleanup worker stopped")
			return
		case <-ticker.C:
			closedCount, err := ep.workBlockManager.CloseIdleWorkBlocks(ctx, 5*time.Minute)
			if err != nil {
				ep.logger.Error("Work block cleanup failed", "error", err)
			} else if closedCount > 0 {
				ep.logger.Info("Work block cleanup completed", "closedWorkBlocks", closedCount)
				
				// Clean up cache
				ep.cleanupWorkBlockCache()
			}
		}
	}
}

/**
 * CONTEXT:   Background worker for cache cleanup and memory management
 * INPUT:     Context for worker lifecycle control
 * OUTPUT:    Periodic cache cleanup removing stale entries
 * BUSINESS:  Maintain memory efficiency through cache management
 * CHANGE:    Background cache cleanup worker implementation
 * RISK:      Low - Cache cleanup affects memory usage but not functionality
 */
func (ep *EventProcessor) cacheCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	ep.logger.Info("Cache cleanup worker started")
	
	for {
		select {
		case <-ctx.Done():
			ep.logger.Info("Cache cleanup worker stopped")
			return
		case <-ticker.C:
			ep.cleanupSessionCache()
			ep.cleanupWorkBlockCache()
		}
	}
}

func (ep *EventProcessor) cleanupSessionCache() {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	
	cleanedCount := 0
	for userID, session := range ep.activeSessions {
		if !session.IsActive() {
			delete(ep.activeSessions, userID)
			cleanedCount++
		}
	}
	
	if cleanedCount > 0 {
		ep.logger.Debug("Cleaned session cache", "removedSessions", cleanedCount)
	}
}

func (ep *EventProcessor) cleanupWorkBlockCache() {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	
	cleanedCount := 0
	now := time.Now()
	
	for sessionID, workBlock := range ep.activeWorkBlocks {
		if !workBlock.IsActive() || workBlock.IsIdle(now) {
			delete(ep.activeWorkBlocks, sessionID)
			cleanedCount++
		}
	}
	
	if cleanedCount > 0 {
		ep.logger.Debug("Cleaned work block cache", "removedWorkBlocks", cleanedCount)
	}
}