/**
 * CONTEXT:   Work block management use case implementing 5-minute idle detection business logic
 * INPUT:     Session contexts, project information, and activity events for work block tracking
 * OUTPUT:    Work block entities with precise timing, idle detection, and project associations
 * BUSINESS:  Work blocks track active work periods, ending after 5 minutes of inactivity
 * CHANGE:    Initial implementation following Clean Architecture and SOLID principles
 * RISK:      High - Core work tracking logic that directly affects time calculation accuracy
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
 * CONTEXT:   Work block manager coordinating work period tracking and idle detection
 * INPUT:     Work block repository, project repository for persistence and business rules
 * OUTPUT:    Work block management operations with 5-minute idle detection
 * BUSINESS:  Manages active work periods within sessions with automatic idle detection
 * CHANGE:    Enhanced with Claude processing time tracking and smart idle detection
 * RISK:      High - Central component for work time tracking with repository dependencies
 */
type WorkBlockManager struct {
	workBlockRepo      repositories.WorkBlockRepository
	projectRepo        repositories.ProjectRepository
	logger             *slog.Logger
	mu                 sync.RWMutex
	activeBlocks       map[string]*entities.WorkBlock // sessionID -> active work block cache
	idleTimeout        time.Duration                   // 5 minutes
	cleanupInterval    time.Duration                   // idle cleanup interval
	// Enhanced: Claude processing support
	processingEstimator *entities.ProcessingTimeEstimator // Time estimation for Claude processing
}

// WorkBlockManagerConfig holds configuration for work block manager
type WorkBlockManagerConfig struct {
	WorkBlockRepo   repositories.WorkBlockRepository
	ProjectRepo     repositories.ProjectRepository
	Logger          *slog.Logger
	IdleTimeout     time.Duration
	CleanupInterval time.Duration
}

/**
 * CONTEXT:   Factory function for creating new work block manager with proper configuration
 * INPUT:     WorkBlockManagerConfig with repositories and operational parameters
 * OUTPUT:    Configured WorkBlockManager instance ready for work tracking
 * BUSINESS:  Work block manager requires repositories for persistence and project management
 * CHANGE:    Initial factory implementation with configuration validation
 * RISK:      Low - Factory function with validation and default value handling
 */
func NewWorkBlockManager(config WorkBlockManagerConfig) (*WorkBlockManager, error) {
	if config.WorkBlockRepo == nil {
		return nil, fmt.Errorf("work block repository cannot be nil")
	}

	if config.ProjectRepo == nil {
		return nil, fmt.Errorf("project repository cannot be nil")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	idleTimeout := config.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = entities.IdleTimeout // 5 minutes default
	}

	cleanupInterval := config.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 2 * time.Minute // default cleanup interval
	}

	wbm := &WorkBlockManager{
		workBlockRepo:       config.WorkBlockRepo,
		projectRepo:         config.ProjectRepo,
		logger:              logger,
		activeBlocks:        make(map[string]*entities.WorkBlock),
		idleTimeout:         idleTimeout,
		cleanupInterval:     cleanupInterval,
		// Enhanced: Initialize processing time estimator
		processingEstimator: entities.NewProcessingTimeEstimator(),
	}

	return wbm, nil
}

/**
 * CONTEXT:   Core business logic for starting work blocks with idle detection rules
 * INPUT:     Session ID, project ID, and start timestamp for work block creation
 * OUTPUT:    New work block entity or updated existing block based on idle detection
 * BUSINESS:  New work block starts if no active block OR idle timeout exceeded (>5 minutes)
 * CHANGE:    Initial implementation of work block business rules with idle detection
 * RISK:      High - Core work tracking logic that determines work time boundaries
 */
func (wbm *WorkBlockManager) StartWorkBlock(ctx context.Context, sessionID, projectID string, timestamp time.Time) (*entities.WorkBlock, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	if timestamp.IsZero() {
		return nil, fmt.Errorf("timestamp cannot be zero")
	}

	wbm.logger.Debug("Starting work block", "sessionID", sessionID, "projectID", projectID, "timestamp", timestamp)

	// Get project information
	project, err := wbm.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to find project: %w", err)
	}

	// Check for active work block
	wbm.mu.RLock()
	activeBlock, hasActive := wbm.activeBlocks[sessionID]
	wbm.mu.RUnlock()

	// If we have an active block, check if we should continue or start new
	if hasActive {
		// Check if block should start new due to idle timeout
		if activeBlock.ShouldStartNewBlock(timestamp) {
			// Finalize the current block
			if err := wbm.finalizeWorkBlock(ctx, activeBlock, timestamp); err != nil {
				wbm.logger.Warn("Failed to finalize idle work block", "error", err)
			}
			// Continue to create new block
		} else {
			// Update existing block with activity
			return wbm.UpdateActivity(ctx, activeBlock.ID(), timestamp)
		}
	}

	// Create new work block
	return wbm.createNewWorkBlock(ctx, sessionID, project, timestamp)
}

/**
 * CONTEXT:   Update activity on existing work block with idle detection validation
 * INPUT:     Work block ID and activity timestamp for work block state update
 * OUTPUT:    Updated work block entity or error if update invalid
 * BUSINESS:  Activity updates reset idle state and extend work block duration
 * CHANGE:    Initial activity update implementation with idle state management
 * RISK:      Medium - Work block state changes affect work time accuracy
 */
func (wbm *WorkBlockManager) UpdateActivity(ctx context.Context, workBlockID string, timestamp time.Time) (*entities.WorkBlock, error) {
	if workBlockID == "" {
		return nil, fmt.Errorf("work block ID cannot be empty")
	}

	if timestamp.IsZero() {
		return nil, fmt.Errorf("timestamp cannot be zero")
	}

	// Find work block
	workBlock, err := wbm.workBlockRepo.FindByID(ctx, workBlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to find work block: %w", err)
	}

	// Check if work block should start new due to idle gap
	if workBlock.ShouldStartNewBlock(timestamp) {
		return nil, fmt.Errorf("work block has been idle too long, new block should be started")
	}

	// Record activity
	err = workBlock.RecordActivity(timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to record activity: %w", err)
	}

	// Update in repository
	err = wbm.workBlockRepo.Update(ctx, workBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to update work block: %w", err)
	}

	// Update cache
	wbm.mu.Lock()
	wbm.activeBlocks[workBlock.SessionID()] = workBlock
	wbm.mu.Unlock()

	wbm.logger.Debug("Updated work block activity", "workBlockID", workBlockID, "timestamp", timestamp)
	return workBlock, nil
}

/**
 * CONTEXT:   Cleanup process for idle work blocks with automatic finalization
 * INPUT:     Idle threshold duration for determining which blocks to close
 * OUTPUT:    Number of work blocks closed due to idle timeout
 * BUSINESS:  Work blocks idle for more than threshold should be finalized automatically
 * CHANGE:    Initial idle cleanup implementation with batch processing
 * RISK:      Medium - Cleanup affects work time calculations and system performance
 */
func (wbm *WorkBlockManager) CloseIdleWorkBlocks(ctx context.Context, idleThreshold time.Duration) (int, error) {
	if idleThreshold <= 0 {
		idleThreshold = wbm.idleTimeout
	}

	now := time.Now()
	wbm.logger.Debug("Starting idle work block cleanup", "threshold", idleThreshold, "currentTime", now)

	// Find idle work blocks
	idleBlocks, err := wbm.workBlockRepo.FindIdleBlocks(ctx, now.Add(-idleThreshold))
	if err != nil {
		return 0, fmt.Errorf("failed to find idle work blocks: %w", err)
	}

	if len(idleBlocks) == 0 {
		wbm.logger.Debug("No idle work blocks found")
		return 0, nil
	}

	closedCount := 0
	for _, workBlock := range idleBlocks {
		// Calculate end time (last activity + some grace period)
		endTime := workBlock.LastActivityTime().Add(1 * time.Minute)

		// Finalize work block
		if err := workBlock.Finish(endTime); err != nil {
			wbm.logger.Warn("Failed to finalize idle work block", "workBlockID", workBlock.ID(), "error", err)
			continue
		}

		// Update in repository
		if err := wbm.workBlockRepo.Update(ctx, workBlock); err != nil {
			wbm.logger.Warn("Failed to update finalized work block", "workBlockID", workBlock.ID(), "error", err)
			continue
		}

		// Remove from cache
		wbm.mu.Lock()
		delete(wbm.activeBlocks, workBlock.SessionID())
		wbm.mu.Unlock()

		closedCount++
		wbm.logger.Debug("Closed idle work block", "workBlockID", workBlock.ID(), "sessionID", workBlock.SessionID())
	}

	wbm.logger.Info("Idle work block cleanup completed", "closedCount", closedCount)
	return closedCount, nil
}

/**
 * CONTEXT:   Start Claude processing within work block to prevent false idle detection
 * INPUT:     Session ID, prompt content, and processing start time
 * OUTPUT:    Updated work block with Claude processing state
 * BUSINESS:  Claude processing prevents work block from being marked idle during AI operations
 * CHANGE:    Enhanced work block management with Claude processing state tracking
 * RISK:      High - Processing state affects idle detection and work time accuracy
 */
func (wbm *WorkBlockManager) StartClaudeProcessing(ctx context.Context, sessionID, prompt string, startTime time.Time) (*entities.WorkBlock, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}
	
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}
	
	if startTime.IsZero() {
		return nil, fmt.Errorf("start time cannot be zero")
	}
	
	// Get or create active work block
	workBlock, err := wbm.GetActiveWorkBlock(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active work block: %w", err)
	}
	
	if workBlock == nil {
		return nil, fmt.Errorf("no active work block found for session")
	}
	
	// Create processing context with estimated time
	estimationRequest := entities.EstimationRequest{
		Prompt:       prompt,
		PromptLength: len(prompt),
		ContextSize:  1, // Default context size
	}
	estimatedDuration := wbm.processingEstimator.EstimateProcessingTime(estimationRequest)
	
	// Generate prompt ID for tracking
	promptID := fmt.Sprintf("prompt_%d_%s", startTime.Unix(), sessionID)
	
	// Start Claude processing in work block
	err = workBlock.StartClaudeProcessing(startTime, estimatedDuration, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to start Claude processing: %w", err)
	}
	
	// Update in repository
	err = wbm.workBlockRepo.Update(ctx, workBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to update work block with processing state: %w", err)
	}
	
	// Update cache
	wbm.mu.Lock()
	wbm.activeBlocks[sessionID] = workBlock
	wbm.mu.Unlock()
	
	wbm.logger.Info("Started Claude processing",
		"sessionID", sessionID,
		"workBlockID", workBlock.ID(),
		"promptID", promptID,
		"estimatedDuration", estimatedDuration)
	
	return workBlock, nil
}

/**
 * CONTEXT:   End Claude processing and record actual processing time for analytics
 * INPUT:     Session ID, processing end time, and optional response metrics
 * OUTPUT:    Updated work block with recorded Claude processing time
 * BUSINESS:  Record Claude processing time separately from user interaction time
 * CHANGE:    Enhanced work block management with processing time tracking
 * RISK:      Medium - Processing time calculations affect work analytics accuracy
 */
func (wbm *WorkBlockManager) EndClaudeProcessing(ctx context.Context, sessionID string, endTime time.Time, responseMetrics map[string]interface{}) (*entities.WorkBlock, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}
	
	if endTime.IsZero() {
		return nil, fmt.Errorf("end time cannot be zero")
	}
	
	// Get active work block
	workBlock, err := wbm.GetActiveWorkBlock(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active work block: %w", err)
	}
	
	if workBlock == nil {
		return nil, fmt.Errorf("no active work block found for session")
	}
	
	if workBlock.State() != entities.WorkBlockStateProcessing {
		wbm.logger.Warn("Attempted to end Claude processing on non-processing work block",
			"sessionID", sessionID,
			"workBlockState", workBlock.State())
		return workBlock, nil
	}
	
	// End Claude processing in work block
	err = workBlock.EndClaudeProcessing(endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to end Claude processing: %w", err)
	}
	
	// Record actual processing time for learning
	if workBlock.LastClaudeActivity() != nil {
		actualProcessingTime := endTime.Sub(*workBlock.LastClaudeActivity())
		
		// Determine complexity for learning (simplified approach)
		complexity := entities.ComplexityLevelModerate // Default
		if actualProcessingTime > 2*time.Minute {
			complexity = entities.ComplexityLevelComplex
		} else if actualProcessingTime < 30*time.Second {
			complexity = entities.ComplexityLevelSimple
		}
		
		wbm.processingEstimator.RecordActualTime(complexity, actualProcessingTime)
	}
	
	// Update in repository
	err = wbm.workBlockRepo.Update(ctx, workBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to update work block after processing end: %w", err)
	}
	
	// Update cache
	wbm.mu.Lock()
	wbm.activeBlocks[sessionID] = workBlock
	wbm.mu.Unlock()
	
	wbm.logger.Info("Ended Claude processing",
		"sessionID", sessionID,
		"workBlockID", workBlock.ID(),
		"claudeProcessingTime", workBlock.ClaudeProcessingTime())
	
	return workBlock, nil
}

/**
 * CONTEXT:   Update Claude processing progress to prevent timeout during long operations
 * INPUT:     Session ID and progress timestamp for extending processing timeout
 * OUTPUT:    Updated work block with extended processing timeout
 * BUSINESS:  Prevent processing timeout during legitimate long Claude operations
 * CHANGE:    Enhanced processing state management with progress tracking
 * RISK:      Low - Progress updates maintain accurate processing state without affecting timing
 */
func (wbm *WorkBlockManager) UpdateClaudeProcessingProgress(ctx context.Context, sessionID string, progressTime time.Time) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	
	if progressTime.IsZero() {
		return fmt.Errorf("progress time cannot be zero")
	}
	
	// Get active work block
	workBlock, err := wbm.GetActiveWorkBlock(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get active work block: %w", err)
	}
	
	if workBlock == nil {
		return fmt.Errorf("no active work block found for session")
	}
	
	if workBlock.State() != entities.WorkBlockStateProcessing {
		// Not in processing state, ignore progress update
		return nil
	}
	
	// Update processing progress
	err = workBlock.UpdateClaudeProcessingProgress(progressTime)
	if err != nil {
		return fmt.Errorf("failed to update processing progress: %w", err)
	}
	
	// Update in repository
	err = wbm.workBlockRepo.Update(ctx, workBlock)
	if err != nil {
		return fmt.Errorf("failed to update work block progress: %w", err)
	}
	
	// Update cache
	wbm.mu.Lock()
	wbm.activeBlocks[sessionID] = workBlock
	wbm.mu.Unlock()
	
	wbm.logger.Debug("Updated Claude processing progress",
		"sessionID", sessionID,
		"workBlockID", workBlock.ID(),
		"progressTime", progressTime)
	
	return nil
}

/**
 * CONTEXT:   Process activity event with Claude processing context awareness
 * INPUT:     Activity event with optional Claude processing context
 * OUTPUT:    Updated work block with appropriate state based on activity type
 * BUSINESS:  Handle different types of activity events including Claude processing states
 * CHANGE:    Enhanced activity processing with Claude context awareness
 * RISK:      High - Core activity processing logic that affects all work tracking
 */
func (wbm *WorkBlockManager) ProcessActivityWithClaudeContext(ctx context.Context, activityEvent *entities.ActivityEvent) (*entities.WorkBlock, error) {
	if activityEvent == nil {
		return nil, fmt.Errorf("activity event cannot be nil")
	}
	
	sessionID := activityEvent.SessionID()
	if sessionID == "" {
		return nil, fmt.Errorf("activity event must have session ID")
	}
	
	// Check if this is a Claude processing activity
	claudeContext := activityEvent.ClaudeContext()
	if claudeContext != nil {
		switch claudeContext.ClaudeActivity {
		case entities.ClaudeActivityStart:
			return wbm.handleClaudeProcessingStart(ctx, activityEvent, claudeContext)
		case entities.ClaudeActivityEnd:
			return wbm.handleClaudeProcessingEnd(ctx, activityEvent, claudeContext)
		case entities.ClaudeActivityProgress:
			return wbm.handleClaudeProcessingProgress(ctx, activityEvent, claudeContext)
		default:
			// Handle as regular user activity
			return wbm.handleRegularActivity(ctx, activityEvent)
		}
	} else {
		// Handle as regular user activity
		return wbm.handleRegularActivity(ctx, activityEvent)
	}
}

/**
 * CONTEXT:   Get active work block for session without creating new block
 * INPUT:     Session ID for work block lookup
 * OUTPUT:    Active work block entity or nil if no active block exists
 * BUSINESS:  Provide read-only access to active work block for status checks
 * CHANGE:    Enhanced with Claude processing state awareness in idle detection
 * RISK:      Low - Read-only operation for work block status queries
 */
func (wbm *WorkBlockManager) GetActiveWorkBlock(ctx context.Context, sessionID string) (*entities.WorkBlock, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	// Check cache first
	wbm.mu.RLock()
	cachedBlock, hasCached := wbm.activeBlocks[sessionID]
	wbm.mu.RUnlock()

	if hasCached && !cachedBlock.IsIdle(time.Now()) {
		return cachedBlock, nil
	}

	// Check database
	activeBlock, err := wbm.workBlockRepo.FindActiveBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Check if block is still active
	if activeBlock.IsIdle(time.Now()) {
		return nil, repositories.ErrWorkBlockNotFound
	}

	// Update cache
	wbm.mu.Lock()
	wbm.activeBlocks[sessionID] = activeBlock
	wbm.mu.Unlock()

	return activeBlock, nil
}

/**
 * CONTEXT:   Get work block statistics for analytics and reporting
 * INPUT:     Session ID and optional time range for statistics calculation
 * OUTPUT:    Work block statistics including durations, counts, and efficiency metrics
 * BUSINESS:  Provide work block analytics for productivity insights and reporting
 * CHANGE:    Initial statistics implementation delegating to repository
 * RISK:      Low - Read-only analytics operation with no state changes
 */
func (wbm *WorkBlockManager) GetWorkBlockStatistics(ctx context.Context, sessionID string, start, end time.Time) (*repositories.WorkBlockStatistics, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	stats, err := wbm.workBlockRepo.GetStatisticsBySession(ctx, sessionID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get work block statistics: %w", err)
	}

	wbm.logger.Debug("Retrieved work block statistics",
		"sessionID", sessionID,
		"totalBlocks", stats.TotalWorkBlocks)

	return stats, nil
}

/**
 * CONTEXT:   Finalize work block when ending due to idle timeout or session end
 * INPUT:     Work block entity and end timestamp for finalization
 * OUTPUT:    Finalized work block with proper end time and state
 * BUSINESS:  Work blocks must be finalized to calculate accurate work durations
 * CHANGE:    Initial finalization implementation with state management
 * RISK:      Medium - Work block finalization affects work time calculations
 */
func (wbm *WorkBlockManager) FinalizeWorkBlock(ctx context.Context, workBlockID string, endTime time.Time) (*entities.WorkBlock, error) {
	if workBlockID == "" {
		return nil, fmt.Errorf("work block ID cannot be empty")
	}

	if endTime.IsZero() {
		return nil, fmt.Errorf("end time cannot be zero")
	}

	// Find work block
	workBlock, err := wbm.workBlockRepo.FindByID(ctx, workBlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to find work block: %w", err)
	}

	// Finalize work block
	err = workBlock.Finish(endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to finish work block: %w", err)
	}

	// Update in repository
	err = wbm.workBlockRepo.Update(ctx, workBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to update finalized work block: %w", err)
	}

	// Remove from cache
	wbm.mu.Lock()
	delete(wbm.activeBlocks, workBlock.SessionID())
	wbm.mu.Unlock()

	wbm.logger.Info("Finalized work block", "workBlockID", workBlockID, "duration", workBlock.Duration())
	return workBlock, nil
}

/**
 * CONTEXT:   Create new work block with proper validation and persistence
 * INPUT:     Session ID, project entity, and start timestamp for work block creation
 * OUTPUT:    New work block entity with proper project association and timing
 * BUSINESS:  New work blocks represent continuous work periods within sessions
 * CHANGE:    Internal helper method for work block creation with full lifecycle
 * RISK:      High - Work block creation affects all subsequent work time tracking
 */
func (wbm *WorkBlockManager) createNewWorkBlock(ctx context.Context, sessionID string, project *entities.Project, startTime time.Time) (*entities.WorkBlock, error) {
	// Create new work block entity
	workBlock, err := entities.NewWorkBlock(entities.WorkBlockConfig{
		SessionID:   sessionID,
		ProjectID:   project.ID(),
		ProjectName: project.Name(),
		ProjectPath: project.Path(),
		StartTime:   startTime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create work block entity: %w", err)
	}

	// Save to repository
	err = wbm.workBlockRepo.Save(ctx, workBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to save work block: %w", err)
	}

	// Update cache
	wbm.mu.Lock()
	wbm.activeBlocks[sessionID] = workBlock
	wbm.mu.Unlock()

	wbm.logger.Info("Created new work block",
		"workBlockID", workBlock.ID(),
		"sessionID", sessionID,
		"projectName", project.Name(),
		"startTime", startTime)

	return workBlock, nil
}

/**
 * CONTEXT:   Internal helper for finalizing work blocks with proper error handling
 * INPUT:     Work block entity and finalization timestamp
 * OUTPUT:    Error if finalization fails, nil on success
 * BUSINESS:  Ensure work blocks are properly finalized with accurate timing
 * CHANGE:    Internal finalization helper with error handling and logging
 * RISK:      Medium - Work block finalization affects timing accuracy
 */
func (wbm *WorkBlockManager) finalizeWorkBlock(ctx context.Context, workBlock *entities.WorkBlock, endTime time.Time) error {
	// Calculate appropriate end time (last activity + grace period)
	appropriateEndTime := workBlock.LastActivityTime().Add(1 * time.Minute)
	if endTime.Before(appropriateEndTime) {
		endTime = appropriateEndTime
	}

	// Finalize work block
	err := workBlock.Finish(endTime)
	if err != nil {
		return fmt.Errorf("failed to finish work block: %w", err)
	}

	// Update in repository
	err = wbm.workBlockRepo.Update(ctx, workBlock)
	if err != nil {
		return fmt.Errorf("failed to update finalized work block: %w", err)
	}

	// Remove from cache
	wbm.mu.Lock()
	delete(wbm.activeBlocks, workBlock.SessionID())
	wbm.mu.Unlock()

	wbm.logger.Debug("Finalized work block", "workBlockID", workBlock.ID(), "duration", workBlock.Duration())
	return nil
}

/**
 * CONTEXT:   Start background cleanup process for automatic work block maintenance
 * INPUT:     Context for cleanup lifecycle control
 * OUTPUT:    Cleanup process running in background with periodic idle detection
 * BUSINESS:  Automatic cleanup maintains work time accuracy and system performance
 * CHANGE:    Initial background cleanup implementation with configurable interval
 * RISK:      Low - Background process with proper context handling and error recovery
 */
func (wbm *WorkBlockManager) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(wbm.cleanupInterval)
	defer ticker.Stop()

	wbm.logger.Info("Starting work block cleanup process", "interval", wbm.cleanupInterval)

	for {
		select {
		case <-ctx.Done():
			wbm.logger.Info("Work block cleanup process stopped")
			return
		case <-ticker.C:
			closedCount, err := wbm.CloseIdleWorkBlocks(ctx, wbm.idleTimeout)
			if err != nil {
				wbm.logger.Error("Work block cleanup failed", "error", err)
			} else if closedCount > 0 {
				wbm.logger.Info("Work block cleanup completed", "closedBlocks", closedCount)
			}
		}
	}
}

/**
 * CONTEXT:   Handle Claude processing start activity event
 * INPUT:     Activity event and Claude processing context for processing start
 * OUTPUT:    Work block updated with Claude processing state
 * BUSINESS:  Start Claude processing state to prevent false idle detection
 * CHANGE:    Enhanced activity handling with Claude processing awareness
 * RISK:      High - Processing state management affects work time accuracy
 */
func (wbm *WorkBlockManager) handleClaudeProcessingStart(ctx context.Context, activityEvent *entities.ActivityEvent, claudeContext *entities.ClaudeProcessingContext) (*entities.WorkBlock, error) {
	sessionID := activityEvent.SessionID()
	
	// Extract prompt information (from command or description)
	prompt := activityEvent.Command()
	if prompt == "" {
		prompt = activityEvent.Description()
	}
	if prompt == "" {
		prompt = "Claude processing" // Fallback
	}
	
	return wbm.StartClaudeProcessing(ctx, sessionID, prompt, activityEvent.Timestamp())
}

/**
 * CONTEXT:   Handle Claude processing end activity event
 * INPUT:     Activity event and Claude processing context for processing completion
 * OUTPUT:    Work block updated with recorded Claude processing time
 * BUSINESS:  End Claude processing and record actual processing duration
 * CHANGE:    Enhanced activity handling with processing completion tracking
 * RISK:      Medium - Processing completion affects work time calculations
 */
func (wbm *WorkBlockManager) handleClaudeProcessingEnd(ctx context.Context, activityEvent *entities.ActivityEvent, claudeContext *entities.ClaudeProcessingContext) (*entities.WorkBlock, error) {
	sessionID := activityEvent.SessionID()
	
	// Extract response metrics from activity metadata
	responseMetrics := make(map[string]interface{})
	metadata := activityEvent.Metadata()
	for key, value := range metadata {
		responseMetrics[key] = value
	}
	
	return wbm.EndClaudeProcessing(ctx, sessionID, activityEvent.Timestamp(), responseMetrics)
}

/**
 * CONTEXT:   Handle Claude processing progress activity event
 * INPUT:     Activity event and Claude processing context for progress updates
 * OUTPUT:    Error if progress update fails, nil on success
 * BUSINESS:  Update processing progress to extend timeout during long operations
 * CHANGE:    Enhanced activity handling with progress tracking
 * RISK:      Low - Progress updates maintain processing state without timing impact
 */
func (wbm *WorkBlockManager) handleClaudeProcessingProgress(ctx context.Context, activityEvent *entities.ActivityEvent, claudeContext *entities.ClaudeProcessingContext) (*entities.WorkBlock, error) {
	sessionID := activityEvent.SessionID()
	
	err := wbm.UpdateClaudeProcessingProgress(ctx, sessionID, activityEvent.Timestamp())
	if err != nil {
		return nil, err
	}
	
	// Return current active work block
	return wbm.GetActiveWorkBlock(ctx, sessionID)
}

/**
 * CONTEXT:   Handle regular user activity event without Claude processing context
 * INPUT:     Activity event for regular user interaction
 * OUTPUT:    Work block updated with regular activity
 * BUSINESS:  Process regular user activities with standard work block logic
 * CHANGE:    Extracted regular activity handling for clarity and separation
 * RISK:      Medium - Regular activity processing affects work block state
 */
func (wbm *WorkBlockManager) handleRegularActivity(ctx context.Context, activityEvent *entities.ActivityEvent) (*entities.WorkBlock, error) {
	sessionID := activityEvent.SessionID()
	projectName := activityEvent.ProjectName()
	
	if projectName == "" {
		return nil, fmt.Errorf("activity event must have project name for regular activities")
	}
	
	// Find or create project
	project, err := wbm.projectRepo.FindByName(ctx, projectName)
	if err != nil {
		if err == repositories.ErrProjectNotFound {
			// Create new project
			project, err = entities.NewProject(entities.ProjectConfig{
				Name: projectName,
				Path: activityEvent.ProjectPath(),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create project entity: %w", err)
			}
			
			err = wbm.projectRepo.Save(ctx, project)
			if err != nil {
				return nil, fmt.Errorf("failed to save project: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to find project: %w", err)
		}
	}
	
	// Start or update work block
	return wbm.StartWorkBlock(ctx, sessionID, project.ID(), activityEvent.Timestamp())
}

/**
 * CONTEXT:   Get processing time estimation statistics for monitoring
 * INPUT:     No parameters, retrieves current estimator statistics
 * OUTPUT:    Processing time estimator statistics for analysis
 * BUSINESS:  Provide insights into processing time estimation accuracy
 * CHANGE:    Enhanced monitoring with processing time estimation statistics
 * RISK:      Low - Read-only statistics for monitoring and improvement
 */
func (wbm *WorkBlockManager) GetProcessingEstimatorStats() entities.EstimatorStats {
	return wbm.processingEstimator.GetStats()
}

/**
 * CONTEXT:   Validate work block manager internal state and configuration
 * INPUT:     No parameters, validates internal configuration and state
 * OUTPUT:    Error if configuration invalid, nil if valid
 * BUSINESS:  Ensure work block manager is properly configured for reliable operation
 * CHANGE:    Enhanced validation with Claude processing estimator validation
 * RISK:      Low - Validation only operation for configuration verification
 */
func (wbm *WorkBlockManager) Validate() error {
	if wbm.workBlockRepo == nil {
		return fmt.Errorf("work block repository cannot be nil")
	}

	if wbm.projectRepo == nil {
		return fmt.Errorf("project repository cannot be nil")
	}

	if wbm.logger == nil {
		return fmt.Errorf("logger cannot be nil")
	}

	if wbm.idleTimeout != entities.IdleTimeout {
		return fmt.Errorf("idle timeout must be exactly %v, got %v", entities.IdleTimeout, wbm.idleTimeout)
	}

	if wbm.cleanupInterval <= 0 {
		return fmt.Errorf("cleanup interval must be positive, got %v", wbm.cleanupInterval)
	}
	
	if wbm.processingEstimator == nil {
		return fmt.Errorf("processing estimator cannot be nil")
	}

	return nil
}
