/**
 * CONTEXT:   Work block statistics and calculations - metrics, counts, and analytics
 * INPUT:     Session IDs, date ranges, work block entities for statistical analysis
 * OUTPUT:    Calculated metrics, counts, and analytics data for reporting
 * BUSINESS:  Statistical operations supporting productivity reporting and system monitoring
 * CHANGE:    Split from workblock_manager.go following Single Responsibility Principle
 * RISK:      Low - Read-only statistical calculations with no state modifications
 */

package business

import (
	"context"
	"fmt"
	"log"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

// WorkBlockStatistics handles statistical operations on work blocks
type WorkBlockStatistics struct {
	workBlockRepo *sqlite.WorkBlockRepository
	projectRepo   *sqlite.ProjectRepository
	activityRepo  *sqlite.ActivityRepository
}

/**
 * CONTEXT:   Create new work block statistics calculator
 * INPUT:     SQLite repositories for work blocks, projects, and activities
 * OUTPUT:    Configured work block statistics calculator
 * BUSINESS:  Statistical operations interface for work block analytics
 * CHANGE:    Extracted from WorkBlockManager for focused statistical operations
 * RISK:      Low - Clean constructor with dependency injection
 */
func NewWorkBlockStatistics(workBlockRepo *sqlite.WorkBlockRepository, projectRepo *sqlite.ProjectRepository, activityRepo *sqlite.ActivityRepository) *WorkBlockStatistics {
	return &WorkBlockStatistics{
		workBlockRepo: workBlockRepo,
		projectRepo:   projectRepo,
		activityRepo:  activityRepo,
	}
}

/**
 * CONTEXT:   Calculate total work time for session from work block durations
 * INPUT:     Session ID for work time calculation
 * OUTPUT:    Total work hours across all work blocks in session
 * BUSINESS:  Session work time calculated from individual work block durations
 * CHANGE:    Initial session work time calculation
 * RISK:      Low - Read-only calculation for reporting
 */
func (wbs *WorkBlockStatistics) CalculateSessionWorkTime(ctx context.Context, sessionID string) (float64, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("session ID cannot be empty")
	}

	workBlocks, err := wbs.workBlockRepo.GetBySession(ctx, sessionID, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to get work blocks for calculation: %w", err)
	}

	totalHours := 0.0
	for _, workBlock := range workBlocks {
		totalHours += workBlock.DurationHours
	}

	return totalHours, nil
}

/**
 * CONTEXT:   Get total work block count for health monitoring and statistics
 * INPUT:     Context for database operations
 * OUTPUT:    Total count of all work blocks in the system
 * BUSINESS:  Work block count metrics for system health monitoring
 * CHANGE:    CHECKPOINT 6 - Added work block count for health endpoint
 * RISK:      Low - Count query for monitoring
 */
func (wbs *WorkBlockStatistics) GetTotalWorkBlockCount(ctx context.Context) (int, error) {
	allWorkBlocks, err := wbs.workBlockRepo.GetAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get total work block count: %w", err)
	}
	return len(allWorkBlocks), nil
}

/**
 * CONTEXT:   Get active work blocks count for a specific session
 * INPUT:     Context and session ID for active work block lookup
 * OUTPUT:    Count of active work blocks for the session
 * BUSINESS:  Active work block counting supports session monitoring and cleanup
 * CHANGE:    CHECKPOINT 6 - Added active work block count for session monitoring
 * RISK:      Low - Count query for session management
 */
func (wbs *WorkBlockStatistics) GetActiveWorkBlocksForSession(ctx context.Context, sessionID string) (int, error) {
	workBlocks, err := wbs.workBlockRepo.GetBySession(ctx, sessionID, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to get work blocks for session: %w", err)
	}
	
	activeCount := 0
	for _, wb := range workBlocks {
		if wb.State == "active" && wb.EndTime == nil {
			activeCount++
		}
	}
	
	return activeCount, nil
}

/**
 * CONTEXT:   Get all active work blocks across all sessions for cleanup operations
 * INPUT:     Context for database operations
 * OUTPUT:    Slice of all currently active work blocks in the system
 * BUSINESS:  Active work blocks query supports system cleanup and monitoring
 * CHANGE:    CHECKPOINT 6 - Added active work blocks query for cleanup endpoint
 * RISK:      Low - Read-only query for cleanup operations
 */
func (wbs *WorkBlockStatistics) GetAllActiveWorkBlocks(ctx context.Context) ([]*WorkBlock, error) {
	allWorkBlocks, err := wbs.workBlockRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all work blocks: %w", err)
	}
	
	activeWorkBlocks := make([]*sqlite.WorkBlock, 0)
	for _, wb := range allWorkBlocks {
		if wb.State == "active" && wb.EndTime == nil {
			activeWorkBlocks = append(activeWorkBlocks, wb)
		}
	}
	
	return activeWorkBlocks, nil
}

/**
 * CONTEXT:   Get project statistics from work blocks for reporting and analytics
 * INPUT:     Context for database operations
 * OUTPUT:    Map of project names to statistics including hours, work blocks, and activities
 * BUSINESS:  Project statistics support reporting endpoints and project analytics
 * CHANGE:    CHECKPOINT 6 - Added project statistics for database query endpoint
 * RISK:      Low - Read-only aggregation for reporting
 */
func (wbs *WorkBlockStatistics) GetProjectStatistics(ctx context.Context) (map[string]map[string]interface{}, error) {
	allWorkBlocks, err := wbs.workBlockRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get work blocks for project statistics: %w", err)
	}
	
	projectStats := make(map[string]map[string]interface{})
	
	for _, wb := range allWorkBlocks {
		// TODO: Implement project name lookup from ProjectID
		projectName := wb.ProjectID // Using ProjectID as temporary project name
		if projectName == "" {
			projectName = "Unknown"
		}
		
		if _, exists := projectStats[projectName]; !exists {
			projectStats[projectName] = map[string]interface{}{
				"work_blocks":      0,
				"total_hours":      0.0,
				"total_activities": 0,
				"project_path":     "unknown", // TODO: Implement project path lookup
			}
		}
		
		stats := projectStats[projectName]
		stats["work_blocks"] = stats["work_blocks"].(int) + 1
		stats["total_hours"] = stats["total_hours"].(float64) + wb.DurationHours
	}
	
	// Add activity counts (if available from activity repository)
	if wbs.activityRepo != nil {
		allActivities, err := wbs.activityRepo.GetAll(ctx)
		if err == nil {
			for _, activity := range allActivities {
				// TODO: Implement project name lookup from WorkBlock
				projectName := "Unknown" // Temporary placeholder
				_ = activity // Avoid unused variable error
				
				if stats, exists := projectStats[projectName]; exists {
					stats["total_activities"] = stats["total_activities"].(int) + 1
				}
			}
		}
	}
	
	return projectStats, nil
}

/**
 * CONTEXT:   Get activity count for work block validation and metrics
 * INPUT:     Context and work block ID to count activities for
 * OUTPUT:    Number of activities associated with the work block
 * BUSINESS:  Activity counting validates work block statistics and relationships
 * CHANGE:    Enhanced activity integration for work block metrics
 * RISK:      Low - Read-only activity counting for validation
 */
func (wbs *WorkBlockStatistics) GetWorkBlockActivityCount(ctx context.Context, workBlockID string) (int64, error) {
	if workBlockID == "" {
		return 0, fmt.Errorf("work block ID cannot be empty")
	}

	activities, err := wbs.activityRepo.GetActivitiesByWorkBlock(workBlockID)
	if err != nil {
		return 0, fmt.Errorf("failed to get activities for work block: %w", err)
	}

	return int64(len(activities)), nil
}

/**
 * CONTEXT:   Calculate average session duration across all sessions
 * INPUT:     Context for database operations and optional date range
 * OUTPUT:    Average session duration in hours
 * BUSINESS:  Session duration analytics for productivity insights
 * CHANGE:    Initial average session duration calculation
 * RISK:      Low - Read-only statistical calculation
 */
func (wbs *WorkBlockStatistics) CalculateAverageSessionDuration(ctx context.Context) (float64, error) {
	allWorkBlocks, err := wbs.workBlockRepo.GetAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get work blocks for average calculation: %w", err)
	}

	// Group work blocks by session and calculate session durations
	sessionDurations := make(map[string]float64)
	for _, wb := range allWorkBlocks {
		sessionDurations[wb.SessionID] += wb.DurationHours
	}

	if len(sessionDurations) == 0 {
		return 0, nil
	}

	// Calculate average
	totalDuration := 0.0
	for _, duration := range sessionDurations {
		totalDuration += duration
	}

	average := totalDuration / float64(len(sessionDurations))
	log.Printf("📊 Calculated average session duration: %.2f hours across %d sessions", average, len(sessionDurations))

	return average, nil
}
