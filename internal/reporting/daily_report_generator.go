/**
 * CONTEXT:   Daily report generator for Claude Monitor work tracking system
 * INPUT:     SQLite repositories, user ID, date specification for report generation
 * OUTPUT:    Enhanced daily reports with work blocks, sessions, and analytics
 * BUSINESS:  Daily reports provide essential work tracking insights for productivity analysis
 * CHANGE:    Extracted from sqlite_reporting_service.go for Single Responsibility Principle
 * RISK:      Medium - Core reporting functionality affecting daily user analytics
 */

package reporting

import (
	"context"
	"fmt"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

/**
 * CONTEXT:   Daily report generator with focused responsibility
 * INPUT:     SQLite repositories for data access
 * OUTPUT:    Daily report generation capability
 * BUSINESS:  Focused generator enables clean daily report creation
 * CHANGE:    Initial daily generator extracted for SRP compliance
 * RISK:      Low - Focused generator with clear dependencies
 */
type DailyReportGenerator struct {
	sessionRepo   *sqlite.SessionRepository
	workBlockRepo *sqlite.WorkBlockRepository
	activityRepo  *sqlite.ActivityRepository
	projectRepo   *sqlite.ProjectRepository
}

/**
 * CONTEXT:   Constructor for daily report generator
 * INPUT:     SQLite repositories for complete data access
 * OUTPUT:    Configured daily report generator ready for use
 * BUSINESS:  Constructor enables dependency injection for clean architecture
 * CHANGE:    Initial constructor with repository dependencies
 * RISK:      Low - Simple constructor with dependency injection
 */
func NewDailyReportGenerator(
	sessionRepo *sqlite.SessionRepository,
	workBlockRepo *sqlite.WorkBlockRepository,
	activityRepo *sqlite.ActivityRepository,
	projectRepo *sqlite.ProjectRepository,
) *DailyReportGenerator {
	return &DailyReportGenerator{
		sessionRepo:   sessionRepo,
		workBlockRepo: workBlockRepo,
		activityRepo:  activityRepo,
		projectRepo:   projectRepo,
	}
}

/**
 * CONTEXT:   Generate comprehensive daily report from SQLite data sources
 * INPUT:     User ID, date for report generation, timezone context
 * OUTPUT:    Enhanced daily report with work blocks, sessions, projects, and insights
 * BUSINESS:  Daily reports are primary user interface for work tracking analytics
 * CHANGE:    Extracted daily report generation for focused responsibility
 * RISK:      Medium - Core reporting functionality with multiple data source orchestration
 */
func (drg *DailyReportGenerator) GenerateDaily(ctx context.Context, userID string, date time.Time) (*EnhancedDailyReport, error) {
	// Set up date boundaries (start of day to end of day)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-1 * time.Nanosecond)

	// Get sessions for the day
	sessions, err := drg.sessionRepo.FindByUserAndTimeRange(ctx, userID, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions for daily report: %w", err)
	}

	report := &EnhancedDailyReport{
		Date:             date,
		ProjectBreakdown: make([]ProjectBreakdown, 0),
		HourlyBreakdown:  make([]HourlyData, 0),
		WorkBlocks:       make([]WorkBlockSummary, 0),
		Insights:         make([]string, 0),
	}

	// Process sessions for the day
	totalSessions := len(sessions)
	if totalSessions == 0 {
		return report, nil
	}

	// Calculate session boundaries and active periods
	var firstActivity, lastActivity time.Time
	var sessionDurations []time.Duration
	
	for _, session := range sessions {
		if firstActivity.IsZero() || session.StartTime.Before(firstActivity) {
			firstActivity = session.StartTime
		}
		if session.EndTime.After(lastActivity) {
			lastActivity = session.EndTime
		}
		sessionDurations = append(sessionDurations, session.EndTime.Sub(session.StartTime))
	}

	// Get work blocks for all sessions
	var allWorkBlocks []*sqlite.WorkBlock
	for _, session := range sessions {
		workBlocks, err := drg.workBlockRepo.GetBySession(ctx, session.ID, 0)
		if err != nil {
			continue // Skip failed sessions but continue processing
		}
		allWorkBlocks = append(allWorkBlocks, workBlocks...)
	}

	// Process work blocks for the report
	for _, workBlock := range allWorkBlocks {
		if err := drg.processWorkBlockForReport(ctx, workBlock, report); err != nil {
			// Log error but continue processing other blocks
			continue
		}
	}

	// Set report totals
	report.TotalSessions = totalSessions
	if !firstActivity.IsZero() && !lastActivity.IsZero() {
		report.ScheduleHours = lastActivity.Sub(firstActivity).Hours()
	}

	return report, nil
}

/**
 * CONTEXT:   Process individual work block for daily report integration
 * INPUT:     Work block data and target report for aggregation
 * OUTPUT:    Updated report with work block data incorporated
 * BUSINESS:  Work block processing aggregates detailed work tracking data
 * CHANGE:    Extracted work block processing for focused daily report logic
 * RISK:      Low - Data processing with error handling for individual blocks
 */
func (drg *DailyReportGenerator) processWorkBlockForReport(ctx context.Context, workBlock *sqlite.WorkBlock, report *EnhancedDailyReport) error {
	if workBlock.EndTime == nil {
		// Work block is still active, use current time
		now := time.Now()
		workBlock.EndTime = &now
	}

	var duration time.Duration
	if workBlock.EndTime != nil {
		duration = workBlock.EndTime.Sub(workBlock.StartTime)
	} else {
		duration = time.Now().Sub(workBlock.StartTime)
	}
	report.TotalWorkHours += duration.Hours()

	// Get project information
	project, err := drg.projectRepo.GetByID(ctx, workBlock.ProjectID)
	if err == nil && project != nil {
		// Find or create project breakdown entry
		var projectBreakdown *ProjectBreakdown
		for i := range report.ProjectBreakdown {
			if report.ProjectBreakdown[i].ProjectName == project.Name {
				projectBreakdown = &report.ProjectBreakdown[i]
				break
			}
		}
		
		if projectBreakdown == nil {
			report.ProjectBreakdown = append(report.ProjectBreakdown, ProjectBreakdown{
				ProjectName: project.Name,
				ProjectPath: project.Path,
			})
			projectBreakdown = &report.ProjectBreakdown[len(report.ProjectBreakdown)-1]
		}
		
		projectBreakdown.WorkHours += duration.Hours()
		projectBreakdown.Sessions++
	}

	// Create work block summary
	summary := WorkBlockSummary{
		StartTime:   workBlock.StartTime,
		EndTime:     workBlock.StartTime.Add(duration),
		Duration:    duration,
		ProjectName: "Unknown Project", // Default value
	}

	// Set project name if available
	if project, err := drg.projectRepo.GetByID(ctx, workBlock.ProjectID); err == nil && project != nil {
		summary.ProjectName = project.Name
	}

	report.WorkBlocks = append(report.WorkBlocks, summary)

	return nil
}

/**
 * CONTEXT:   Interface compliance for ReportGenerator
 * INPUT:     Context, user ID, date parameters for different report types
 * OUTPUT:    Appropriate report type or error for unsupported operations
 * BUSINESS:  Interface compliance enables polymorphic report generation
 * CHANGE:    Interface implementation for clean generator abstraction
 * RISK:      Low - Interface compliance with clear error messages
 */
func (drg *DailyReportGenerator) GenerateWeekly(ctx context.Context, userID string, weekStart time.Time) (*EnhancedWeeklyReport, error) {
	return nil, fmt.Errorf("weekly report generation not supported by daily generator")
}

func (drg *DailyReportGenerator) GenerateMonthly(ctx context.Context, userID string, monthStart time.Time) (*EnhancedMonthlyReport, error) {
	return nil, fmt.Errorf("monthly report generation not supported by daily generator")
}