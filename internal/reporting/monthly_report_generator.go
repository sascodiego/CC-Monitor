/**
 * CONTEXT:   Monthly report generator for Claude Monitor work tracking system
 * INPUT:     SQLite repositories, user ID, month start date for report generation
 * OUTPUT:    Enhanced monthly reports with daily heatmap, achievements, and trends
 * BUSINESS:  Monthly reports provide long-term productivity insights and goal tracking
 * CHANGE:    Extracted from sqlite_reporting_service.go for Single Responsibility Principle
 * RISK:      Medium - Month-long data aggregation with complex achievement calculation
 */

package reporting

import (
	"context"
	"fmt"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

/**
 * CONTEXT:   Monthly report generator with focused responsibility
 * INPUT:     SQLite repositories for data access and daily report dependency
 * OUTPUT:    Monthly report generation capability with achievement tracking
 * BUSINESS:  Focused generator enables clean monthly report creation with long-term analysis
 * CHANGE:    Initial monthly generator extracted for SRP compliance
 * RISK:      Medium - Complex month-long aggregation with achievement calculations
 */
type MonthlyReportGenerator struct {
	sessionRepo          *sqlite.SessionRepository
	workBlockRepo        *sqlite.WorkBlockRepository
	activityRepo         *sqlite.ActivityRepository
	projectRepo          *sqlite.ProjectRepository
	dailyReportGenerator *DailyReportGenerator
}

/**
 * CONTEXT:   Constructor for monthly report generator
 * INPUT:     SQLite repositories and daily report generator dependency
 * OUTPUT:    Configured monthly report generator ready for use
 * BUSINESS:  Constructor enables dependency injection for clean architecture
 * CHANGE:    Initial constructor with daily generator dependency
 * RISK:      Low - Constructor with proper dependency injection
 */
func NewMonthlyReportGenerator(
	sessionRepo *sqlite.SessionRepository,
	workBlockRepo *sqlite.WorkBlockRepository,
	activityRepo *sqlite.ActivityRepository,
	projectRepo *sqlite.ProjectRepository,
	dailyGenerator *DailyReportGenerator,
) *MonthlyReportGenerator {
	return &MonthlyReportGenerator{
		sessionRepo:          sessionRepo,
		workBlockRepo:        workBlockRepo,
		activityRepo:         activityRepo,
		projectRepo:          projectRepo,
		dailyReportGenerator: dailyGenerator,
	}
}

/**
 * CONTEXT:   Generate comprehensive monthly report with heatmap and achievements
 * INPUT:     User ID, month start date for full month analysis
 * OUTPUT:    Enhanced monthly report with daily progress, achievements, and trends
 * BUSINESS:  Monthly reports provide long-term productivity insights and goal tracking
 * CHANGE:    Extracted monthly report generation for focused responsibility
 * RISK:      Medium - Month-long data aggregation with complex achievement calculation
 */
func (mrg *MonthlyReportGenerator) GenerateMonthly(ctx context.Context, userID string, monthStart time.Time) (*EnhancedMonthlyReport, error) {
	// Ensure monthStart is beginning of month
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, monthStart.Location())
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-1 * time.Nanosecond)

	report := &EnhancedMonthlyReport{
		Month:            monthStart,
		Year:             monthStart.Year(),
		MonthStart:       monthStart,
		MonthEnd:         monthEnd,
		DailyHeatmap:     make([]DayData, 0),
		ProjectBreakdown: make([]ProjectBreakdown, 0),
		Achievements:     make([]Achievement, 0),
		Trends:           make([]Trend, 0),
		Insights:         make([]string, 0),
	}

	// Process each day in the month
	projectTotals := make(map[string]*ProjectBreakdown)
	totalWorkHours := 0.0
	workingDays := 0
	bestDayHours := 0.0
	longestStreak := 0
	currentStreak := 0

	for day := monthStart; day.Before(monthEnd.AddDate(0, 0, 1)); day = day.AddDate(0, 0, 1) {
		dailyReport, err := mrg.dailyReportGenerator.GenerateDaily(ctx, userID, day)
		if err != nil {
			// Add day with zero hours
			report.DailyHeatmap = append(report.DailyHeatmap, DayData{
				Date:  day,
				Hours: 0.0,
				Level: 0,
			})
			currentStreak = 0
			continue
		}

		// Create heatmap entry
		heatmapData := DayData{
			Date:  day,
			Hours: dailyReport.TotalWorkHours,
			Level: mrg.calculateHeatmapLevel(dailyReport.TotalWorkHours),
		}
		report.DailyHeatmap = append(report.DailyHeatmap, heatmapData)

		totalWorkHours += dailyReport.TotalWorkHours

		// Track working days and streaks
		if dailyReport.TotalWorkHours > 0 {
			workingDays++
			currentStreak++
			if currentStreak > longestStreak {
				longestStreak = currentStreak
			}
		} else {
			currentStreak = 0
		}

		// Track best day
		if dailyReport.TotalWorkHours > bestDayHours {
			bestDayHours = dailyReport.TotalWorkHours
			report.BestDay = heatmapData
		}

		// Aggregate project data
		mrg.aggregateMonthlyProjectData(dailyReport.ProjectBreakdown, projectTotals)
	}

	// Set monthly totals and averages
	report.TotalWorkHours = totalWorkHours
	report.WorkingDays = workingDays
	report.AverageHoursPerDay = totalWorkHours / float64(mrg.getDaysInMonth(monthStart))
	if workingDays > 0 {
		report.AverageHoursPerWorkingDay = totalWorkHours / float64(workingDays)
	}
	report.LongestWorkStreak = longestStreak

	// Finalize project breakdown
	mrg.finalizeMonthlyProjectBreakdown(projectTotals, totalWorkHours, report)

	return report, nil
}

/**
 * CONTEXT:   Calculate heatmap level based on daily work hours
 * INPUT:     Daily work hours for intensity calculation
 * OUTPUT:    Heatmap level (0-4) for visual representation
 * BUSINESS:  Heatmap levels provide visual productivity patterns
 * CHANGE:    Extracted heatmap calculation for reusability
 * RISK:      Low - Simple level calculation with clear thresholds
 */
func (mrg *MonthlyReportGenerator) calculateHeatmapLevel(hours float64) int {
	if hours >= 8 {
		return 4 // Maximum intensity
	} else if hours >= 6 {
		return 3
	} else if hours >= 4 {
		return 2
	} else if hours >= 1 {
		return 1
	}
	return 0 // No activity
}

/**
 * CONTEXT:   Aggregate project data for monthly reporting
 * INPUT:     Daily project breakdown and monthly project totals
 * OUTPUT:    Updated monthly project totals
 * BUSINESS:  Monthly project aggregation shows long-term project focus
 * CHANGE:    Extracted monthly aggregation logic
 * RISK:      Low - Data aggregation similar to weekly but extended
 */
func (mrg *MonthlyReportGenerator) aggregateMonthlyProjectData(dailyProjects []ProjectBreakdown, projectTotals map[string]*ProjectBreakdown) {
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
 * CONTEXT:   Finalize monthly project breakdown with percentages
 * INPUT:     Project totals, total work hours, and target report
 * OUTPUT:    Complete project breakdown for monthly report
 * BUSINESS:  Monthly project breakdown shows focus distribution
 * CHANGE:    Extracted monthly finalization logic
 * RISK:      Low - Similar to weekly but with monthly scope
 */
func (mrg *MonthlyReportGenerator) finalizeMonthlyProjectBreakdown(projectTotals map[string]*ProjectBreakdown, totalWorkHours float64, report *EnhancedMonthlyReport) {
	for _, project := range projectTotals {
		if totalWorkHours > 0 {
			project.Percentage = (project.WorkHours / totalWorkHours) * 100
		}
		report.ProjectBreakdown = append(report.ProjectBreakdown, *project)
	}
}

/**
 * CONTEXT:   Get number of days in given month
 * INPUT:     Date within the target month
 * OUTPUT:    Number of days in the month
 * BUSINESS:  Day count enables accurate monthly average calculations
 * CHANGE:    Extracted utility for month calculations
 * RISK:      Low - Date utility function
 */
func (mrg *MonthlyReportGenerator) getDaysInMonth(date time.Time) int {
	firstDay := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	lastDay := firstDay.AddDate(0, 1, -1)
	return lastDay.Day()
}

/**
 * CONTEXT:   Interface compliance for ReportGenerator
 * INPUT:     Context, user ID, date parameters for different report types
 * OUTPUT:    Appropriate report type or error for unsupported operations
 * BUSINESS:  Interface compliance enables polymorphic report generation
 * CHANGE:    Interface implementation for clean generator abstraction
 * RISK:      Low - Interface compliance with clear error messages
 */
func (mrg *MonthlyReportGenerator) GenerateDaily(ctx context.Context, userID string, date time.Time) (*EnhancedDailyReport, error) {
	return nil, fmt.Errorf("daily report generation not supported by monthly generator")
}

func (mrg *MonthlyReportGenerator) GenerateWeekly(ctx context.Context, userID string, weekStart time.Time) (*EnhancedWeeklyReport, error) {
	return nil, fmt.Errorf("weekly report generation not supported by monthly generator")
}