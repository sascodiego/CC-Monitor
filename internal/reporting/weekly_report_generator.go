/**
 * CONTEXT:   Weekly report generator for Claude Monitor work tracking system
 * INPUT:     SQLite repositories, user ID, week start date for report generation
 * OUTPUT:    Enhanced weekly reports with daily breakdown, trends, and analytics
 * BUSINESS:  Weekly reports show work patterns and consistency over time periods
 * CHANGE:    Extracted from sqlite_reporting_service.go for Single Responsibility Principle
 * RISK:      Medium - Multi-day aggregation with trend calculation logic
 */

package reporting

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

/**
 * CONTEXT:   Weekly report generator with focused responsibility
 * INPUT:     SQLite repositories for data access and daily report dependency
 * OUTPUT:    Weekly report generation capability with daily aggregation
 * BUSINESS:  Focused generator enables clean weekly report creation with trend analysis
 * CHANGE:    Initial weekly generator extracted for SRP compliance
 * RISK:      Medium - Depends on daily report generation for aggregation
 */
type WeeklyReportGenerator struct {
	sessionRepo          *sqlite.SessionRepository
	workBlockRepo        *sqlite.WorkBlockRepository
	activityRepo         *sqlite.ActivityRepository
	projectRepo          *sqlite.ProjectRepository
	dailyReportGenerator *DailyReportGenerator
}

/**
 * CONTEXT:   Constructor for weekly report generator
 * INPUT:     SQLite repositories and daily report generator dependency
 * OUTPUT:    Configured weekly report generator ready for use
 * BUSINESS:  Constructor enables dependency injection for clean architecture
 * CHANGE:    Initial constructor with daily generator dependency
 * RISK:      Low - Constructor with proper dependency injection
 */
func NewWeeklyReportGenerator(
	sessionRepo *sqlite.SessionRepository,
	workBlockRepo *sqlite.WorkBlockRepository,
	activityRepo *sqlite.ActivityRepository,
	projectRepo *sqlite.ProjectRepository,
	dailyGenerator *DailyReportGenerator,
) *WeeklyReportGenerator {
	return &WeeklyReportGenerator{
		sessionRepo:          sessionRepo,
		workBlockRepo:        workBlockRepo,
		activityRepo:         activityRepo,
		projectRepo:          projectRepo,
		dailyReportGenerator: dailyGenerator,
	}
}

/**
 * CONTEXT:   Generate comprehensive weekly report with daily breakdown
 * INPUT:     User ID, week start date for 7-day period analysis
 * OUTPUT:    Enhanced weekly report with trends, insights, and project breakdown
 * BUSINESS:  Weekly reports provide work pattern analysis and productivity trends
 * CHANGE:    Extracted weekly report generation for focused responsibility
 * RISK:      Medium - Multi-day aggregation with trend calculation logic
 */
func (wrg *WeeklyReportGenerator) GenerateWeekly(ctx context.Context, userID string, weekStart time.Time) (*EnhancedWeeklyReport, error) {
	// Ensure weekStart is beginning of day
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	weekEnd := weekStart.Add(7 * 24 * time.Hour).Add(-1 * time.Nanosecond)

	report := &EnhancedWeeklyReport{
		WeekStart:        weekStart,
		WeekEnd:          weekEnd,
		WeekNumber:       wrg.getWeekNumber(weekStart),
		Year:             weekStart.Year(),
		DailyBreakdown:   make([]DaySummary, 7),
		ProjectBreakdown: make([]ProjectBreakdown, 0),
		Insights:         make([]WeeklyInsight, 0),
		Trends:           make([]Trend, 0),
	}

	// Initialize daily breakdown with all 7 days
	for i := 0; i < 7; i++ {
		day := weekStart.Add(time.Duration(i) * 24 * time.Hour)
		report.DailyBreakdown[i] = DaySummary{
			Date:    day,
			DayName: day.Format("Mon"),
			Hours:   0.0,
		}
	}

	// Process each day's data using daily generator
	projectTotals := make(map[string]*ProjectBreakdown)
	totalWorkHours := 0.0
	bestDayHours := 0.0

	for i, day := range report.DailyBreakdown {
		dailyReport, err := wrg.dailyReportGenerator.GenerateDaily(ctx, userID, day.Date)
		if err != nil {
			continue // Skip days with errors
		}

		// Update daily summary
		report.DailyBreakdown[i].Hours = dailyReport.TotalWorkHours
		report.DailyBreakdown[i].ClaudeSessions = dailyReport.TotalSessions
		report.DailyBreakdown[i].WorkBlocks = len(dailyReport.WorkBlocks)

		// Track productivity status
		report.DailyBreakdown[i].Status = wrg.calculateProductivityStatus(dailyReport.TotalWorkHours)

		totalWorkHours += dailyReport.TotalWorkHours

		// Track most productive day
		if dailyReport.TotalWorkHours > bestDayHours {
			bestDayHours = dailyReport.TotalWorkHours
			report.MostProductiveDay = report.DailyBreakdown[i]
		}

		// Aggregate project data
		wrg.aggregateProjectData(dailyReport.ProjectBreakdown, projectTotals)
	}

	// Set weekly totals and averages
	report.TotalWorkHours = totalWorkHours
	report.DailyAverage = totalWorkHours / 7.0

	// Convert project totals to slice and calculate percentages
	wrg.finalizeProjectBreakdown(projectTotals, totalWorkHours, report)

	return report, nil
}

/**
 * CONTEXT:   Calculate productivity status based on work hours
 * INPUT:     Daily work hours for status determination
 * OUTPUT:    Productivity status string for day classification
 * BUSINESS:  Status classification helps users understand productivity levels
 * CHANGE:    Extracted status calculation for reusability
 * RISK:      Low - Simple classification logic with clear thresholds
 */
func (wrg *WeeklyReportGenerator) calculateProductivityStatus(hours float64) string {
	if hours >= 8 {
		return "excellent"
	} else if hours >= 6 {
		return "good"
	} else if hours >= 3 {
		return "low"
	}
	return "none"
}

/**
 * CONTEXT:   Aggregate project data from daily reports
 * INPUT:     Daily project breakdown and project totals map
 * OUTPUT:    Updated project totals with accumulated data
 * BUSINESS:  Project aggregation enables weekly project analysis
 * CHANGE:    Extracted aggregation logic for better organization
 * RISK:      Low - Data aggregation with map operations
 */
func (wrg *WeeklyReportGenerator) aggregateProjectData(dailyProjects []ProjectBreakdown, projectTotals map[string]*ProjectBreakdown) {
	for _, project := range dailyProjects {
		if existing, exists := projectTotals[project.ProjectName]; exists {
			existing.WorkHours += project.WorkHours
			existing.Sessions += project.Sessions
		} else {
			projectTotals[project.ProjectName] = &ProjectBreakdown{
				ProjectName: project.ProjectName,
				ProjectPath: project.ProjectPath,
				WorkHours:   project.WorkHours,
				Sessions:    project.Sessions,
			}
		}
	}
}

/**
 * CONTEXT:   Finalize project breakdown with percentages and sorting
 * INPUT:     Project totals, total work hours, and target report
 * OUTPUT:    Sorted project breakdown with calculated percentages
 * BUSINESS:  Project breakdown shows time distribution across projects
 * CHANGE:    Extracted finalization logic for clean organization
 * RISK:      Low - Data processing with percentage calculation
 */
func (wrg *WeeklyReportGenerator) finalizeProjectBreakdown(projectTotals map[string]*ProjectBreakdown, totalWorkHours float64, report *EnhancedWeeklyReport) {
	for _, project := range projectTotals {
		if totalWorkHours > 0 {
			project.Percentage = (project.WorkHours / totalWorkHours) * 100
		}
		report.ProjectBreakdown = append(report.ProjectBreakdown, *project)
	}

	// Sort projects by work hours
	sort.Slice(report.ProjectBreakdown, func(i, j int) bool {
		return report.ProjectBreakdown[i].WorkHours > report.ProjectBreakdown[j].WorkHours
	})
}

/**
 * CONTEXT:   Calculate week number for given date
 * INPUT:     Date for week number calculation
 * OUTPUT:    Week number within the year
 * BUSINESS:  Week number enables consistent weekly reporting periods
 * CHANGE:    Extracted week calculation utility
 * RISK:      Low - Date calculation utility function
 */
func (wrg *WeeklyReportGenerator) getWeekNumber(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}

/**
 * CONTEXT:   Interface compliance for ReportGenerator
 * INPUT:     Context, user ID, date parameters for different report types
 * OUTPUT:    Appropriate report type or error for unsupported operations
 * BUSINESS:  Interface compliance enables polymorphic report generation
 * CHANGE:    Interface implementation for clean generator abstraction
 * RISK:      Low - Interface compliance with clear error messages
 */
func (wrg *WeeklyReportGenerator) GenerateDaily(ctx context.Context, userID string, date time.Time) (*EnhancedDailyReport, error) {
	return nil, fmt.Errorf("daily report generation not supported by weekly generator")
}

func (wrg *WeeklyReportGenerator) GenerateMonthly(ctx context.Context, userID string, monthStart time.Time) (*EnhancedMonthlyReport, error) {
	return nil, fmt.Errorf("monthly report generation not supported by weekly generator")
}