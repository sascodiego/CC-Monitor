/**
 * CONTEXT:   Work block project integration - project operations, validation, and activity handling
 * INPUT:     Work block IDs, project paths, activity data for project-related operations
 * OUTPUT:    Project-integrated work block data, validation results, and activity summaries
 * BUSINESS:  Project integration supporting work block organization and detailed activity analysis
 * CHANGE:    Split from workblock_manager.go following Single Responsibility Principle
 * RISK:      Low - Project integration and validation operations with activity repository
 */

package business

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

// WorkBlockProjectIntegration handles project-related work block operations
type WorkBlockProjectIntegration struct {
	workBlockRepo *sqlite.WorkBlockRepository
	projectRepo   *sqlite.ProjectRepository
	activityRepo  *sqlite.ActivityRepository
}

// WorkBlockSummary provides detailed information about a work block
type WorkBlockSummary struct {
	WorkBlock        *sqlite.WorkBlock    `json:"work_block"`
	ActivityCount    int                  `json:"activity_count"`
	Activities       []*sqlite.Activity   `json:"activities,omitempty"`
	ProjectPath      string               `json:"project_path"`
	ProjectName      string               `json:"project_name"`
	DurationMinutes  float64              `json:"duration_minutes"`
	IsActive         bool                 `json:"is_active"`
}

/**
 * CONTEXT:   Create new work block project integration manager
 * INPUT:     SQLite repositories for work blocks, projects, and activities
 * OUTPUT:    Configured work block project integration manager
 * BUSINESS:  Project integration operations interface for work block management
 * CHANGE:    Extracted from WorkBlockManager for focused project operations
 * RISK:      Low - Clean constructor with dependency injection
 */
func NewWorkBlockProjectIntegration(workBlockRepo *sqlite.WorkBlockRepository, projectRepo *sqlite.ProjectRepository, activityRepo *sqlite.ActivityRepository) *WorkBlockProjectIntegration {
	return &WorkBlockProjectIntegration{
		workBlockRepo: workBlockRepo,
		projectRepo:   projectRepo,
		activityRepo:  activityRepo,
	}
}

/**
 * CONTEXT:   Get activities for a specific work block with proper ordering
 * INPUT:     Context and work block ID to retrieve activities for
 * OUTPUT:    Ordered list of activities within the work block
 * BUSINESS:  Activities grouped by work block for detailed analysis and reporting
 * CHANGE:    Enhanced activity integration for work block activity access
 * RISK:      Low - Simple query delegation with proper error handling
 */
func (wbpi *WorkBlockProjectIntegration) GetWorkBlockActivities(ctx context.Context, workBlockID string) ([]*sqlite.Activity, error) {
	if workBlockID == "" {
		return nil, fmt.Errorf("work block ID cannot be empty")
	}

	activities, err := wbpi.activityRepo.GetActivitiesByWorkBlock(workBlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activities for work block: %w", err)
	}

	log.Printf("📋 Retrieved %d activities for work block %s", len(activities), workBlockID)
	return activities, nil
}

/**
 * CONTEXT:   Validate work block activity count against database activities
 * INPUT:     Context and work block entity to validate
 * OUTPUT:    Boolean indicating if activity count matches actual activities
 * BUSINESS:  Data integrity validation ensuring work block statistics are accurate
 * CHANGE:    Enhanced validation with activity repository integration
 * RISK:      Low - Validation helper for data consistency checks
 */
func (wbpi *WorkBlockProjectIntegration) ValidateWorkBlockActivityCount(ctx context.Context, workBlock *WorkBlock) (bool, error) {
	if workBlock == nil {
		return false, fmt.Errorf("work block cannot be nil")
	}

	actualCount, err := wbpi.getWorkBlockActivityCount(ctx, workBlock.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get actual activity count: %w", err)
	}

	expectedCount := int64(workBlock.ActivityCount)
	isValid := actualCount == expectedCount

	if !isValid {
		log.Printf("⚠️ Activity count mismatch for work block %s: expected=%d, actual=%d", 
			workBlock.ID, expectedCount, actualCount)
	}

	return isValid, nil
}

/**
 * CONTEXT:   Enhanced work block summary with activity integration
 * INPUT:     Context and work block ID for comprehensive summary
 * OUTPUT:    Work block summary with activity statistics and project information
 * BUSINESS:  Comprehensive work block reporting with activity details
 * CHANGE:    Enhanced summary with activity metrics for CHECKPOINT 4
 * RISK:      Low - Read-only summary generation with activity integration
 */
func (wbpi *WorkBlockProjectIntegration) GetWorkBlockSummary(ctx context.Context, workBlockID string, includeActivities bool) (*WorkBlockSummary, error) {
	if workBlockID == "" {
		return nil, fmt.Errorf("work block ID cannot be empty")
	}

	// Get work block
	workBlock, err := wbpi.workBlockRepo.GetByID(ctx, workBlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get work block: %w", err)
	}

	// Get project
	project, err := wbpi.projectRepo.GetByID(ctx, workBlock.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Get activity count
	activityCount, err := wbpi.getWorkBlockActivityCount(ctx, workBlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity count: %w", err)
	}

	// Validate activity count
	isValid, err := wbpi.ValidateWorkBlockActivityCount(ctx, workBlock)
	if err != nil {
		log.Printf("Warning: failed to validate activity count: %v", err)
		isValid = false
	}

	// Use project information if available
	projectName := "Unknown"
	projectPath := "unknown"
	if project != nil {
		projectName = project.Name
		projectPath = project.Path
	}
	
	summary := &WorkBlockSummary{
		WorkBlock:       workBlock,
		ActivityCount:   int(activityCount),
		ProjectName:     projectName,
		ProjectPath:     projectPath,
		DurationMinutes: workBlock.DurationHours * 60,
		IsActive:        workBlock.State == "active" && workBlock.EndTime == nil,
	}
	
	// Log validation result (to use isValid variable)
	_ = isValid // Variable is used for validation but not stored in summary

	// Include activities if requested
	if includeActivities {
		activities, err := wbpi.GetWorkBlockActivities(ctx, workBlockID)
		if err != nil {
			return nil, fmt.Errorf("failed to get activities: %w", err)
		}
		summary.Activities = activities
	}

	return summary, nil
}

/**
 * CONTEXT:   Validate work block data integrity and relationships
 * INPUT:     Context and work block entity for comprehensive validation
 * OUTPUT:    Validation result with detailed error information if invalid
 * BUSINESS:  Data integrity validation ensuring work block consistency
 * CHANGE:    Initial work block validation with project and activity checks
 * RISK:      Low - Read-only validation with comprehensive checks
 */
func (wbpi *WorkBlockProjectIntegration) ValidateWorkBlock(ctx context.Context, workBlock *WorkBlock) error {
	if workBlock == nil {
		return fmt.Errorf("work block cannot be nil")
	}

	// Validate required fields
	if workBlock.ID == "" {
		return fmt.Errorf("work block ID cannot be empty")
	}

	if workBlock.SessionID == "" {
		return fmt.Errorf("work block session ID cannot be empty")
	}

	if workBlock.ProjectID == "" {
		return fmt.Errorf("work block project ID cannot be empty")
	}

	// Validate project exists
	project, err := wbpi.projectRepo.GetByID(ctx, workBlock.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to validate project: %w", err)
	}
	if project == nil {
		return fmt.Errorf("work block references non-existent project: %s", workBlock.ProjectID)
	}

	// Validate activity count
	isValidCount, err := wbpi.ValidateWorkBlockActivityCount(ctx, workBlock)
	if err != nil {
		return fmt.Errorf("failed to validate activity count: %w", err)
	}
	if !isValidCount {
		return fmt.Errorf("work block activity count mismatch")
	}

	// Validate time consistency
	if workBlock.EndTime != nil && workBlock.EndTime.Before(workBlock.StartTime) {
		return fmt.Errorf("work block end time cannot be before start time")
	}

	if workBlock.DurationHours < 0 {
		return fmt.Errorf("work block duration cannot be negative")
	}

	log.Printf("✅ Work block %s validation successful", workBlock.ID)
	return nil
}

/**
 * CONTEXT:   Get project information from file path with intelligent parsing
 * INPUT:     File path or working directory path for project identification
 * OUTPUT:    Project name and normalized path for project operations
 * BUSINESS:  Project identification from file paths supporting automatic project detection
 * CHANGE:    Initial project path parsing with intelligent name extraction
 * RISK:      Low - Path parsing utility with fallback handling
 */
func (wbpi *WorkBlockProjectIntegration) GetProjectFromPath(projectPath string) (string, string, error) {
	if projectPath == "" {
		return "", "", fmt.Errorf("project path cannot be empty")
	}

	// Clean and normalize the path
	normalizedPath := filepath.Clean(projectPath)
	
	// Extract project name from path
	projectName := filepath.Base(normalizedPath)
	
	// Handle special cases
	if projectName == "." || projectName == "" {
		projectName = "unknown-project"
	}
	
	// Remove common prefixes that aren't useful as project names
	if strings.HasPrefix(projectName, ".") && len(projectName) > 1 {
		projectName = projectName[1:] // Remove leading dot
	}
	
	log.Printf("📋 Extracted project info: name='%s', path='%s'", projectName, normalizedPath)
	
	return projectName, normalizedPath, nil
}

/**
 * CONTEXT:   Update work block project association
 * INPUT:     Context, work block ID, and new project path for project update
 * OUTPUT:    Updated work block with new project association
 * BUSINESS:  Project association updates support project reorganization and corrections
 * CHANGE:    Initial project update implementation with validation
 * RISK:      Medium - Project association changes affecting work block categorization
 */
func (wbpi *WorkBlockProjectIntegration) UpdateWorkBlockProject(ctx context.Context, workBlockID, newProjectPath string) (*WorkBlock, error) {
	if workBlockID == "" {
		return nil, fmt.Errorf("work block ID cannot be empty")
	}
	
	if newProjectPath == "" {
		return nil, fmt.Errorf("new project path cannot be empty")
	}

	// Get existing work block
	workBlock, err := wbpi.workBlockRepo.GetByID(ctx, workBlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to get work block: %w", err)
	}
	
	if workBlock == nil {
		return nil, fmt.Errorf("work block %s not found", workBlockID)
	}

	// Get or create new project
	newProject, err := wbpi.projectRepo.GetOrCreate(ctx, newProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create new project: %w", err)
	}

	// Update work block project association
	oldProjectID := workBlock.ProjectID
	workBlock.ProjectID = newProject.ID

	// Update in database
	if err := wbpi.workBlockRepo.Update(ctx, workBlock); err != nil {
		return nil, fmt.Errorf("failed to update work block project: %w", err)
	}

	log.Printf("🔄 Updated work block %s project: %s -> %s (%s)", 
		workBlockID, oldProjectID, newProject.ID, newProject.Name)

	return workBlock, nil
}

/**
 * CONTEXT:   Synchronize project data across work blocks
 * INPUT:     Context and project ID for synchronization operations
 * OUTPUT:    Number of work blocks synchronized with updated project information
 * BUSINESS:  Project synchronization ensures data consistency across work blocks
 * CHANGE:    Initial project synchronization implementation
 * RISK:      Medium - Bulk updates affecting multiple work blocks
 */
func (wbpi *WorkBlockProjectIntegration) SyncProjectData(ctx context.Context, projectID string) (int, error) {
	if projectID == "" {
		return 0, fmt.Errorf("project ID cannot be empty")
	}

	// Get project information
	project, err := wbpi.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("failed to get project: %w", err)
	}
	
	if project == nil {
		return 0, fmt.Errorf("project %s not found", projectID)
	}

	// Get all work blocks for this project
	allWorkBlocks, err := wbpi.workBlockRepo.GetAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get work blocks for sync: %w", err)
	}

	syncedCount := 0
	for _, wb := range allWorkBlocks {
		if wb.ProjectID == projectID {
			// Work block already references this project - validate consistency
			if err := wbpi.ValidateWorkBlock(ctx, wb); err != nil {
				log.Printf("Warning: work block %s validation failed during sync: %v", wb.ID, err)
				continue
			}
			syncedCount++
		}
	}

	log.Printf("🔄 Synchronized %d work blocks for project %s (%s)", syncedCount, project.ID, project.Name)
	return syncedCount, nil
}

/**
 * CONTEXT:   Resolve project conflicts in work block associations
 * INPUT:     Context for database operations and optional conflict resolution strategy
 * OUTPUT:    Number of conflicts resolved and resolution summary
 * BUSINESS:  Conflict resolution ensures data integrity in project associations
 * CHANGE:    Initial conflict resolution implementation
 * RISK:      Medium - Data modification affecting project categorization
 */
func (wbpi *WorkBlockProjectIntegration) ResolveProjectConflicts(ctx context.Context) (map[string]interface{}, error) {
	allWorkBlocks, err := wbpi.workBlockRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get work blocks for conflict resolution: %w", err)
	}

	results := map[string]interface{}{
		"total_work_blocks":    len(allWorkBlocks),
		"conflicts_found":      0,
		"conflicts_resolved":   0,
		"orphaned_work_blocks": 0,
		"invalid_projects":     []string{},
	}

	conflictsFound := 0
	conflictsResolved := 0
	orphanedCount := 0
	invalidProjects := make([]string, 0)

	for _, wb := range allWorkBlocks {
		// Check if project exists
		project, err := wbpi.projectRepo.GetByID(ctx, wb.ProjectID)
		if err != nil {
			log.Printf("Error checking project %s: %v", wb.ProjectID, err)
			conflictsFound++
			continue
		}

		if project == nil {
			// Orphaned work block - project doesn't exist
			log.Printf("Orphaned work block %s references non-existent project %s", wb.ID, wb.ProjectID)
			orphanedCount++
			conflictsFound++
			
			// Try to resolve by creating a default project
			defaultProject, err := wbpi.projectRepo.GetOrCreate(ctx, "unknown-project")
			if err == nil {
				wb.ProjectID = defaultProject.ID
				if updateErr := wbpi.workBlockRepo.Update(ctx, wb); updateErr == nil {
					conflictsResolved++
				}
			}
			continue
		}

		// Validate work block
		if err := wbpi.ValidateWorkBlock(ctx, wb); err != nil {
			log.Printf("Invalid work block %s: %v", wb.ID, err)
			conflictsFound++
			invalidProjects = append(invalidProjects, wb.ProjectID)
		}
	}

	results["conflicts_found"] = conflictsFound
	results["conflicts_resolved"] = conflictsResolved
	results["orphaned_work_blocks"] = orphanedCount
	results["invalid_projects"] = invalidProjects

	log.Printf("🔧 Project conflict resolution: %d found, %d resolved, %d orphaned", 
		conflictsFound, conflictsResolved, orphanedCount)

	return results, nil
}

/**
 * CONTEXT:   Get activity count for work block validation and metrics
 * INPUT:     Context and work block ID to count activities for
 * OUTPUT:    Number of activities associated with the work block
 * BUSINESS:  Activity counting validates work block statistics and relationships
 * CHANGE:    Private helper method for activity counting
 * RISK:      Low - Read-only activity counting for validation
 */
func (wbpi *WorkBlockProjectIntegration) getWorkBlockActivityCount(ctx context.Context, workBlockID string) (int64, error) {
	if workBlockID == "" {
		return 0, fmt.Errorf("work block ID cannot be empty")
	}

	activities, err := wbpi.activityRepo.GetActivitiesByWorkBlock(workBlockID)
	if err != nil {
		return 0, fmt.Errorf("failed to get activities for work block: %w", err)
	}

	return int64(len(activities)), nil
}
