/**
 * CONTEXT:   Core work block operations - creation, processing, and lifecycle management
 * INPUT:     Activity events, session IDs, project paths for work block management
 * OUTPUT:    Active work blocks with proper state transitions and idle detection
 * BUSINESS:  Core work block operations with 5-minute idle timeout and automatic project creation
 * CHANGE:    Split from workblock_manager.go following Single Responsibility Principle
 * RISK:      Medium - Core work tracking logic affecting time calculations and user reports
 */

package business

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

// WorkBlockManagerCore handles core work block operations
type WorkBlockManagerCore struct {
	workBlockRepo *sqlite.WorkBlockRepository
	projectRepo   *sqlite.ProjectRepository
	activityRepo  *sqlite.ActivityRepository
	timezone      *time.Location
	idleThreshold time.Duration // Time threshold for idle detection (5 minutes)
}

// WorkBlock type alias for consistency
type WorkBlock = sqlite.WorkBlock

// Project type alias for consistency  
type Project = sqlite.Project

/**
 * CONTEXT:   Create new work block manager core with activity integration
 * INPUT:     SQLite repositories for work blocks, projects, and activities
 * OUTPUT:    Configured work block manager core with activity coordination capabilities
 * BUSINESS:  Core operations interface for work block management with activity integration
 * CHANGE:    Extracted from WorkBlockManager for focused core operations
 * RISK:      Low - Clean constructor with dependency injection
 */
func NewWorkBlockManagerCore(workBlockRepo *sqlite.WorkBlockRepository, projectRepo *sqlite.ProjectRepository, activityRepo *sqlite.ActivityRepository) *WorkBlockManagerCore {
	// Load timezone for consistent time handling
	timezone, err := time.LoadLocation("America/Montevideo")
	if err != nil {
		log.Printf("Warning: failed to load timezone America/Montevideo, using UTC: %v", err)
		timezone = time.UTC
	}

	return &WorkBlockManagerCore{
		workBlockRepo: workBlockRepo,
		projectRepo:   projectRepo,
		activityRepo:  activityRepo,
		timezone:      timezone,
		idleThreshold: 5 * time.Minute, // Enhanced: Configurable idle detection
	}
}

/**
 * CONTEXT:   Process activity event with work block creation and idle detection
 * INPUT:     Session ID, project path, activity timestamp for work block management
 * OUTPUT:    Active work block for the session-project combination with idle handling
 * BUSINESS:  Create new work block if idle timeout exceeded, otherwise update existing
 * CHANGE:    Core activity processing with automatic project creation and idle detection
 * RISK:      Medium - Critical path for work time tracking affecting user reports
 */
func (wbmc *WorkBlockManagerCore) ProcessActivity(ctx context.Context, sessionID, projectPath string, activityTime time.Time) (*WorkBlock, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	if projectPath == "" {
		return nil, fmt.Errorf("project path cannot be empty")
	}

	if activityTime.IsZero() {
		activityTime = time.Now().In(wbmc.timezone)
	}

	// Convert activity time to timezone
	activityTime = activityTime.In(wbmc.timezone)

	log.Printf("🔨 Processing work block activity: session=%s, path=%s, time=%s", 
		sessionID, projectPath, activityTime.Format("2006-01-02 15:04:05"))

	// STEP 1: Get or create project from path
	project, err := wbmc.projectRepo.GetOrCreate(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create project: %w", err)
	}

	// STEP 2: Find existing active work block for session-project
	activeWorkBlock, err := wbmc.workBlockRepo.GetActiveBySessionAndProject(ctx, sessionID, project.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active work block: %w", err)
	}

	// STEP 3: Check if we need to create new work block or update existing
	if activeWorkBlock == nil {
		// No active work block, create new one
		return wbmc.createNewWorkBlock(ctx, sessionID, project, activityTime)
	}

	// STEP 4: Check if existing work block is idle (>5 minutes since last activity)
	if wbmc.workBlockRepo.IsWorkBlockIdle(activeWorkBlock, activityTime) {
		log.Printf("💤 Work block %s is idle (last activity: %v), creating new work block",
			activeWorkBlock.ID, activeWorkBlock.LastActivityTime.Format("15:04:05"))
		
		// Finish the idle work block
		if err := wbmc.finishIdleWorkBlock(ctx, activeWorkBlock, activityTime); err != nil {
			log.Printf("Warning: failed to finish idle work block: %v", err)
		}
		
		// Create new work block
		return wbmc.createNewWorkBlock(ctx, sessionID, project, activityTime)
	}

	// STEP 5: Update existing active work block with new activity
	if err := wbmc.workBlockRepo.RecordActivity(ctx, activeWorkBlock.ID, activityTime); err != nil {
		return nil, fmt.Errorf("failed to record activity in work block: %w", err)
	}

	// Refresh work block data after update
	updatedWorkBlock, err := wbmc.workBlockRepo.GetByID(ctx, activeWorkBlock.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated work block: %w", err)
	}

	log.Printf("✅ Updated work block: id=%s, activities=%d, duration=%.2f hours",
		updatedWorkBlock.ID, updatedWorkBlock.ActivityCount, updatedWorkBlock.DurationHours)

	return updatedWorkBlock, nil
}

/**
 * CONTEXT:   Create new work block for session-project combination
 * INPUT:     Session ID, project entity, and start time for work block creation
 * OUTPUT:    New active work block persisted to database
 * BUSINESS:  Work blocks start active and track time within session boundaries
 * CHANGE:    Initial work block creation with project relationship
 * RISK:      Low - Work block creation with validation and error handling
 */
func (wbmc *WorkBlockManagerCore) createNewWorkBlock(ctx context.Context, sessionID string, project *Project, startTime time.Time) (*WorkBlock, error) {
	workBlockID := fmt.Sprintf("wb_%s_%s_%d_%d", sessionID, project.ID, startTime.Unix(), startTime.Nanosecond())

	workBlock := &WorkBlock{
		ID:               workBlockID,
		SessionID:        sessionID,
		ProjectID:        project.ID,
		StartTime:        startTime,
		EndTime:          nil, // Will be set when finished
		State:            "active",
		LastActivityTime: startTime,
		ActivityCount:    1,
		DurationSeconds:  0,
		DurationHours:    0.0,
		CreatedAt:        time.Now().In(wbmc.timezone),
		UpdatedAt:        time.Now().In(wbmc.timezone),
	}

	if err := wbmc.workBlockRepo.Create(ctx, workBlock); err != nil {
		return nil, fmt.Errorf("failed to create new work block: %w", err)
	}

	log.Printf("🆕 Created new work block: id=%s, project=%s (%s), session=%s",
		workBlock.ID, project.Name, project.Path, sessionID)

	return workBlock, nil
}

/**
 * CONTEXT:   Finish idle work block with calculated end time
 * INPUT:     Idle work block and current activity time for finalization
 * OUTPUT:    Finished work block with end time set to last activity + idle timeout
 * BUSINESS:  Idle work blocks end 5 minutes after last activity for accurate time tracking
 * CHANGE:    Initial idle work block finalization with time calculation
 * RISK:      Low - Work block finalization with time validation
 */
func (wbmc *WorkBlockManagerCore) finishIdleWorkBlock(ctx context.Context, workBlock *WorkBlock, currentTime time.Time) error {
	// Calculate end time as last activity + idle timeout (5 minutes)
	endTime := workBlock.LastActivityTime.Add(5 * time.Minute)
	
	// Ensure end time is not in the future
	if endTime.After(currentTime) {
		endTime = currentTime
	}

	if err := wbmc.workBlockRepo.FinishWorkBlock(ctx, workBlock.ID, endTime); err != nil {
		return fmt.Errorf("failed to finish idle work block: %w", err)
	}

	log.Printf("🏁 Finished idle work block: id=%s, duration=%.2f hours (ended at %s)",
		workBlock.ID, endTime.Sub(workBlock.StartTime).Hours(), endTime.Format("15:04:05"))

	return nil
}

/**
 * CONTEXT:   Get active work block for session and project if exists
 * INPUT:     Session ID and project path for work block lookup
 * OUTPUT:    Currently active work block or nil if none active
 * BUSINESS:  Check work block status without creating new blocks
 * CHANGE:    Initial work block status query
 * RISK:      Low - Read-only work block lookup
 */
func (wbmc *WorkBlockManagerCore) GetActiveWorkBlock(ctx context.Context, sessionID, projectPath string) (*WorkBlock, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	if projectPath == "" {
		return nil, fmt.Errorf("project path cannot be empty")
	}

	// Get project by path
	project, err := wbmc.projectRepo.GetByPath(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get project by path: %w", err)
	}

	if project == nil {
		return nil, nil // Project doesn't exist, so no active work block
	}

	// Get active work block for session-project
	workBlock, err := wbmc.workBlockRepo.GetActiveBySessionAndProject(ctx, sessionID, project.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active work block: %w", err)
	}

	return workBlock, nil
}

/**
 * CONTEXT:   Get all work blocks for session for reporting and analysis
 * INPUT:     Session ID and optional limit for pagination
 * OUTPUT:    Array of work blocks with project information for reporting
 * BUSINESS:  Session work block analysis for productivity reporting
 * CHANGE:    Initial session work block listing
 * RISK:      Low - Read-only query for reporting purposes
 */
func (wbmc *WorkBlockManagerCore) GetWorkBlocksBySession(ctx context.Context, sessionID string, limit int) ([]*WorkBlock, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	workBlocks, err := wbmc.workBlockRepo.GetBySession(ctx, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get work blocks by session: %w", err)
	}

	return workBlocks, nil
}

/**
 * CONTEXT:   Batch process to mark idle work blocks across all sessions
 * INPUT:     Current timestamp for idle detection comparison
 * OUTPUT:    Number of work blocks marked as idle with cleanup results
 * BUSINESS:  Periodic maintenance to detect and close idle work blocks
 * CHANGE:    Initial batch idle detection for system maintenance
 * RISK:      Medium - Bulk state changes affecting multiple work blocks
 */
func (wbmc *WorkBlockManagerCore) MarkIdleWorkBlocks(ctx context.Context) (int, error) {
	currentTime := time.Now().In(wbmc.timezone)
	
	idleCount, err := wbmc.workBlockRepo.MarkIdleWorkBlocks(ctx, currentTime)
	if err != nil {
		return 0, fmt.Errorf("failed to mark idle work blocks: %w", err)
	}

	if idleCount > 0 {
		log.Printf("🧹 Idle detection completed: marked %d work blocks as idle", idleCount)
	}

	return idleCount, nil
}

/**
 * CONTEXT:   Finish all active work blocks for session (when session ends)
 * INPUT:     Session ID and end timestamp for work block finalization
 * OUTPUT:    Number of work blocks finished with session completion
 * BUSINESS:  Session completion requires finishing all associated work blocks
 * CHANGE:    Initial session cleanup with work block finalization
 * RISK:      Medium - Bulk finalization affecting session work time calculations
 */
func (wbmc *WorkBlockManagerCore) FinishWorkBlocksForSession(ctx context.Context, sessionID string, endTime time.Time) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session ID cannot be empty")
	}

	if endTime.IsZero() {
		endTime = time.Now().In(wbmc.timezone)
	}

	// Get all active work blocks for session
	workBlocks, err := wbmc.workBlockRepo.GetBySession(ctx, sessionID, 0) // No limit
	if err != nil {
		return 0, fmt.Errorf("failed to get work blocks for session: %w", err)
	}

	finishedCount := 0
	for _, workBlock := range workBlocks {
		if workBlock.EndTime == nil { // Only finish active work blocks
			if err := wbmc.workBlockRepo.FinishWorkBlock(ctx, workBlock.ID, endTime); err != nil {
				log.Printf("Warning: failed to finish work block %s: %v", workBlock.ID, err)
				continue
			}
			finishedCount++
		}
	}

	if finishedCount > 0 {
		log.Printf("🏁 Session cleanup: finished %d work blocks for session %s", finishedCount, sessionID)
	}

	return finishedCount, nil
}

/**
 * CONTEXT:   Get project by path with auto-creation if needed
 * INPUT:     Working directory path for project lookup or creation
 * OUTPUT:    Project entity with consistent ID and path normalization
 * BUSINESS:  Seamless project management for work block organization
 * CHANGE:    Project access wrapper with auto-creation
 * RISK:      Low - Delegated to project repository
 */
func (wbmc *WorkBlockManagerCore) GetOrCreateProject(ctx context.Context, projectPath string) (*Project, error) {
	if projectPath == "" {
		return nil, fmt.Errorf("project path cannot be empty")
	}

	return wbmc.projectRepo.GetOrCreate(ctx, projectPath)
}
