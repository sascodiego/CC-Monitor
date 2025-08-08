/**
 * CONTEXT:   Data grouping and processing functions for work analytics
 * INPUT:     Raw work blocks and activities requiring grouping and processing
 * OUTPUT:    Organized data structures for analysis and pattern recognition
 * BUSINESS:  Data processing enables accurate analysis by organizing work patterns
 * CHANGE:    Extracted processing logic from monolithic work_analytics_engine.go
 * RISK:      Low - Data organization functions with clear transformation logic
 */

package reporting

import (
	"context"
	"sort"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

/**
 * CONTEXT:   Group work blocks by project continuity for context switching analysis
 * INPUT:     Array of work blocks requiring project grouping
 * OUTPUT:    Array of project sessions representing continuous work on same project
 * BUSINESS:  Project sessions help analyze context switching costs and focus efficiency
 * CHANGE:    Extracted from core engine, maintains grouping algorithm
 * RISK:      Low - Data grouping with project identification and time continuity
 */
func (wae *WorkAnalyticsEngine) groupWorkBlocksByProject(ctx context.Context, workBlocks []*sqlite.WorkBlock) []ProjectSession {
	if len(workBlocks) == 0 {
		return []ProjectSession{}
	}
	
	// Sort blocks by start time
	sortedBlocks := make([]*sqlite.WorkBlock, len(workBlocks))
	copy(sortedBlocks, workBlocks)
	sort.Slice(sortedBlocks, func(i, j int) bool {
		return sortedBlocks[i].StartTime.Before(sortedBlocks[j].StartTime)
	})
	
	sessions := make([]ProjectSession, 0)
	currentSession := ProjectSession{
		ProjectID:   sortedBlocks[0].ProjectID,
		StartTime:   sortedBlocks[0].StartTime,
		WorkBlocks:  []*sqlite.WorkBlock{sortedBlocks[0]},
	}
	
	// Get project name
	if project, err := wae.projectRepo.GetByID(ctx, sortedBlocks[0].ProjectID); err == nil && project != nil {
		currentSession.ProjectName = project.Name
	}
	
	for i := 1; i < len(sortedBlocks); i++ {
		block := sortedBlocks[i]
		lastBlock := currentSession.WorkBlocks[len(currentSession.WorkBlocks)-1]
		
		// Check if same project and continuous (gap < 30 minutes)
		var gap time.Duration
		if lastBlock.EndTime != nil {
			gap = block.StartTime.Sub(*lastBlock.EndTime)
		} else {
			gap = block.StartTime.Sub(lastBlock.LastActivityTime)
		}
		
		if block.ProjectID == currentSession.ProjectID && gap < 30*time.Minute {
			// Continue current session
			currentSession.WorkBlocks = append(currentSession.WorkBlocks, block)
		} else {
			// Finalize current session
			wae.finalizeProjectSession(&currentSession)
			sessions = append(sessions, currentSession)
			
			// Start new session
			currentSession = ProjectSession{
				ProjectID:  block.ProjectID,
				StartTime:  block.StartTime,
				WorkBlocks: []*sqlite.WorkBlock{block},
			}
			
			// Get project name
			if project, err := wae.projectRepo.GetByID(ctx, block.ProjectID); err == nil && project != nil {
				currentSession.ProjectName = project.Name
			}
		}
	}
	
	// Finalize last session
	wae.finalizeProjectSession(&currentSession)
	sessions = append(sessions, currentSession)
	
	return sessions
}