/**
 * CONTEXT:   Work block session management - session lifecycle, cleanup, and health checks
 * INPUT:     Session IDs, timestamps, limits for session-related work block operations
 * OUTPUT:    Session work block management, cleanup results, and health status
 * BUSINESS:  Session-oriented work block operations supporting session lifecycle management
 * CHANGE:    Split from workblock_manager.go following Single Responsibility Principle
 * RISK:      Medium - Session state management affecting work block lifecycle
 */

package business

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

// WorkBlockSessionManager handles session-related work block operations
type WorkBlockSessionManager struct {
	workBlockRepo *sqlite.WorkBlockRepository
	projectRepo   *sqlite.ProjectRepository
	activityRepo  *sqlite.ActivityRepository
	timezone      *time.Location
}

/**
 * CONTEXT:   Create new work block session manager
 * INPUT:     SQLite repositories for work blocks, projects, and activities
 * OUTPUT:    Configured work block session manager
 * BUSINESS:  Session-oriented operations interface for work block management
 * CHANGE:    Extracted from WorkBlockManager for focused session operations
 * RISK:      Low - Clean constructor with dependency injection
 */
func NewWorkBlockSessionManager(workBlockRepo *sqlite.WorkBlockRepository, projectRepo *sqlite.ProjectRepository, activityRepo *sqlite.ActivityRepository) *WorkBlockSessionManager {
	// Load timezone for consistent time handling
	timezone, err := time.LoadLocation("America/Montevideo")
	if err != nil {
		log.Printf("Warning: failed to load timezone America/Montevideo, using UTC: %v", err)
		timezone = time.UTC
	}

	return &WorkBlockSessionManager{
		workBlockRepo: workBlockRepo,
		projectRepo:   projectRepo,
		activityRepo:  activityRepo,
		timezone:      timezone,
	}
}

/**
 * CONTEXT:   Get work blocks for a specific session for analysis and reporting
 * INPUT:     Context and session ID for work block lookup
 * OUTPUT:    Slice of all work blocks associated with the session
 * BUSINESS:  Session work block queries support reporting and session analysis
 * CHANGE:    CHECKPOINT 6 - Added session work block query for reporting
 * RISK:      Low - Read-only query for session analysis
 */
func (wbsm *WorkBlockSessionManager) GetWorkBlocksForSession(ctx context.Context, sessionID string) ([]*WorkBlock, error) {
	workBlocks, err := wbsm.workBlockRepo.GetBySession(ctx, sessionID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get work blocks for session: %w", err)
	}
	return workBlocks, nil
}

/**
 * CONTEXT:   Get recent work blocks for database query endpoint and monitoring
 * INPUT:     Context and limit for recent work block count
 * OUTPUT:    Slice of most recent work blocks up to the specified limit
 * BUSINESS:  Recent work block queries support administrative monitoring and troubleshooting
 * CHANGE:    CHECKPOINT 6 - Added recent work blocks query for database endpoint
 * RISK:      Low - Read-only query with limit for administrative use
 */
func (wbsm *WorkBlockSessionManager) GetRecentWorkBlocks(ctx context.Context, limit int) ([]*WorkBlock, error) {
	allWorkBlocks, err := wbsm.workBlockRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent work blocks: %w", err)
	}
	
	// Sort by created time descending and limit
	sortedWorkBlocks := allWorkBlocks
	if len(sortedWorkBlocks) > limit {
		sortedWorkBlocks = sortedWorkBlocks[len(sortedWorkBlocks)-limit:]
	}
	
	return sortedWorkBlocks, nil
}

/**
 * CONTEXT:   Close a specific work block by ID for cleanup operations
 * INPUT:     Context, work block ID, and close timestamp
 * OUTPUT:    Work block marked as closed with proper end time and duration
 * BUSINESS:  Work block closure supports cleanup operations and proper time tracking finalization
 * CHANGE:    CHECKPOINT 6 - Added work block closure for cleanup endpoint
 * RISK:      Medium - State modification affecting work tracking
 */
func (wbsm *WorkBlockSessionManager) CloseWorkBlock(ctx context.Context, workBlockID string, closeTime time.Time) error {
	workBlock, err := wbsm.workBlockRepo.GetByID(ctx, workBlockID)
	if err != nil {
		return fmt.Errorf("failed to get work block for closure: %w", err)
	}
	
	if workBlock == nil {
		return fmt.Errorf("work block %s not found", workBlockID)
	}
	
	// Calculate final duration
	duration := closeTime.Sub(workBlock.StartTime)
	workBlock.EndTime = &closeTime
	workBlock.DurationSeconds = int64(duration.Seconds())
	workBlock.DurationHours = duration.Hours()
	workBlock.State = "closed"
	workBlock.UpdatedAt = closeTime
	
	// Update in database
	if err := wbsm.workBlockRepo.Update(ctx, workBlock); err != nil {
		return fmt.Errorf("failed to close work block: %w", err)
	}
	
	log.Printf("✅ Closed work block %s at %v (duration: %.2f hours)", workBlockID, closeTime, duration.Hours())
	return nil
}

/**
 * CONTEXT:   Health check for work block manager dependencies
 * INPUT:     Context for timeout handling
 * OUTPUT:    Health status of repositories and database connections
 * BUSINESS:  System health monitoring for work block management
 * CHANGE:    Initial health check implementation
 * RISK:      Low - Read-only health check operations
 */
func (wbsm *WorkBlockSessionManager) HealthCheck(ctx context.Context) error {
	// Test work block repository connection
	_, err := wbsm.workBlockRepo.GetByID(ctx, "health-check-test")
	if err != nil && err.Error() != "work block health-check-test not found" {
		return fmt.Errorf("work block repository health check failed: %w", err)
	}

	// Test project repository connection
	_, err = wbsm.projectRepo.GetByID(ctx, "health-check-test")
	if err != nil && err.Error() != "project health-check-test not found" {
		return fmt.Errorf("project repository health check failed: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Handle session end with work block finalization
 * INPUT:     Session ID and end timestamp for session completion
 * OUTPUT:    Number of work blocks finalized for the ended session
 * BUSINESS:  Session end handling ensures proper work block closure and time tracking
 * CHANGE:    Initial session end handling with work block finalization
 * RISK:      Medium - Session state transition affecting multiple work blocks
 */
func (wbsm *WorkBlockSessionManager) HandleSessionEnd(ctx context.Context, sessionID string, endTime time.Time) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session ID cannot be empty")
	}

	if endTime.IsZero() {
		endTime = time.Now().In(wbsm.timezone)
	}

	// Get all work blocks for session
	workBlocks, err := wbsm.workBlockRepo.GetBySession(ctx, sessionID, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to get work blocks for session end: %w", err)
	}

	finishedCount := 0
	for _, workBlock := range workBlocks {
		if workBlock.EndTime == nil && workBlock.State == "active" {
			if err := wbsm.workBlockRepo.FinishWorkBlock(ctx, workBlock.ID, endTime); err != nil {
				log.Printf("Warning: failed to finish work block %s on session end: %v", workBlock.ID, err)
				continue
			}
			finishedCount++
		}
	}

	if finishedCount > 0 {
		log.Printf("🏁 Session end: finalized %d work blocks for session %s", finishedCount, sessionID)
	}

	return finishedCount, nil
}

/**
 * CONTEXT:   Clean up old work blocks based on retention policy
 * INPUT:     Context and retention duration for cleanup operations
 * OUTPUT:    Number of work blocks cleaned up
 * BUSINESS:  Cleanup operations maintain database performance and enforce retention policies
 * CHANGE:    Initial cleanup implementation with configurable retention
 * RISK:      Medium - Data deletion operation with retention policy enforcement
 */
func (wbsm *WorkBlockSessionManager) CleanupOldWorkBlocks(ctx context.Context, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		retentionDays = 90 // Default 90-day retention
	}

	cutoffTime := time.Now().In(wbsm.timezone).AddDate(0, 0, -retentionDays)
	
	allWorkBlocks, err := wbsm.workBlockRepo.GetAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get work blocks for cleanup: %w", err)
	}

	cleanedCount := 0
	for _, workBlock := range allWorkBlocks {
		// Only clean up finished work blocks older than cutoff
		if workBlock.EndTime != nil && workBlock.EndTime.Before(cutoffTime) {
			if err := wbsm.workBlockRepo.Delete(ctx, workBlock.ID); err != nil {
				log.Printf("Warning: failed to delete old work block %s: %v", workBlock.ID, err)
				continue
			}
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		log.Printf("🧹 Cleanup completed: removed %d work blocks older than %d days", cleanedCount, retentionDays)
	}

	return cleanedCount, nil
}

/**
 * CONTEXT:   Process session work blocks for analysis and reporting
 * INPUT:     Session ID and processing options for work block operations
 * OUTPUT:    Processed work block summary and statistics
 * BUSINESS:  Session work block processing supports analytical operations and reporting
 * CHANGE:    Initial session work block processing implementation
 * RISK:      Low - Read-only processing for analysis purposes
 */
func (wbsm *WorkBlockSessionManager) ProcessSessionWorkBlocks(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	workBlocks, err := wbsm.workBlockRepo.GetBySession(ctx, sessionID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get work blocks for processing: %w", err)
	}

	// Calculate session statistics
	totalHours := 0.0
	activeCount := 0
	finishedCount := 0
	totalActivities := 0

	for _, wb := range workBlocks {
		totalHours += wb.DurationHours
		totalActivities += wb.ActivityCount
		if wb.EndTime == nil {
			activeCount++
		} else {
			finishedCount++
		}
	}

	result := map[string]interface{}{
		"session_id":         sessionID,
		"total_work_blocks":  len(workBlocks),
		"active_work_blocks": activeCount,
		"finished_work_blocks": finishedCount,
		"total_hours":        totalHours,
		"total_activities":   totalActivities,
		"processed_at":       time.Now().In(wbsm.timezone),
	}

	log.Printf("📊 Processed session %s: %d work blocks, %.2f hours, %d activities", 
		sessionID, len(workBlocks), totalHours, totalActivities)

	return result, nil
}
