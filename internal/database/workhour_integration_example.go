/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-036
 * CONTEXT:   Work hour integration example demonstrating comprehensive usage of extended database features
 * REASON:    Provide practical examples of using work hour database extensions for reporting and analytics
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep examples current with interface changes, validate example code periodically
 * RISK:      Low - Example code doesn't affect production functionality
 */
package database

import (
	"fmt"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-037
 * CONTEXT:   WorkHourIntegrationExample demonstrates comprehensive work hour database usage
 * REASON:    Provide practical examples for developers implementing work hour analytics and reporting
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep examples synchronized with interface changes, test examples regularly
 * RISK:      Low - Example failures don't affect production systems
 */
type WorkHourIntegrationExample struct {
	whm    *WorkHourManager
	logger arch.Logger
}

func NewWorkHourIntegrationExample(whm *WorkHourManager, logger arch.Logger) *WorkHourIntegrationExample {
	return &WorkHourIntegrationExample{
		whm:    whm,
		logger: logger,
	}
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-038
 * CONTEXT:   Example: Daily work hour analysis and reporting
 * REASON:    Demonstrate how to aggregate session/work block data into daily work summaries
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate date handling across timezones, test with various work patterns
 * RISK:      Low - Example code for demonstration purposes only
 */
func (ex *WorkHourIntegrationExample) DailyWorkHourAnalysisExample() error {
	ex.logger.Info("=== Daily Work Hour Analysis Example ===")

	// Analyze today's work
	today := time.Now()
	workDay, err := ex.whm.GetWorkDayData(today)
	if err != nil {
		return fmt.Errorf("failed to get work day data: %w", err)
	}

	ex.logger.Info("Daily Work Summary",
		"date", workDay.Date.Format("2006-01-02"),
		"totalTime", workDay.TotalTime,
		"sessionCount", workDay.SessionCount,
		"blockCount", workDay.BlockCount,
		"breakTime", workDay.BreakTime,
		"efficiency", fmt.Sprintf("%.2f%%", workDay.GetEfficiencyRatio()*100),
		"isComplete", workDay.IsComplete)

	// Get productivity metrics for the day
	productivity, err := ex.whm.GetProductivityMetrics(today, today)
	if err != nil {
		return fmt.Errorf("failed to get productivity metrics: %w", err)
	}

	ex.logger.Info("Daily Productivity Metrics",
		"activeRatio", fmt.Sprintf("%.2f%%", productivity.ActiveRatio*100),
		"focusScore", fmt.Sprintf("%.2f", productivity.FocusScore),
		"interruptionRate", fmt.Sprintf("%.2f/hour", productivity.InterruptionRate),
		"peakEfficiency", productivity.PeakEfficiency)

	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-039
 * CONTEXT:   Example: Weekly work pattern analysis and overtime calculation
 * REASON:    Demonstrate weekly aggregation with overtime tracking and pattern identification
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure week boundary calculations are consistent across different locales
 * RISK:      Low - Example code for demonstration purposes only
 */
func (ex *WorkHourIntegrationExample) WeeklyWorkPatternExample() error {
	ex.logger.Info("=== Weekly Work Pattern Analysis Example ===")

	// Get this week's data (Monday to Sunday)
	now := time.Now()
	weekStart := now
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}

	workWeek, err := ex.whm.GetWorkWeekData(weekStart)
	if err != nil {
		return fmt.Errorf("failed to get work week data: %w", err)
	}

	ex.logger.Info("Weekly Work Summary",
		"weekStart", workWeek.WeekStart.Format("2006-01-02"),
		"totalTime", workWeek.TotalTime,
		"overtimeHours", workWeek.OvertimeHours,
		"averageDay", workWeek.AverageDay,
		"workDaysCount", len(workWeek.WorkDays),
		"standardHours", workWeek.StandardHours,
		"isComplete", workWeek.IsComplete)

	// Analyze work patterns for the week
	patterns, err := ex.whm.GetWorkPatternData(workWeek.WeekStart, workWeek.WeekEnd)
	if err != nil {
		return fmt.Errorf("failed to get work pattern data: %w", err)
	}

	ex.logger.Info("Work Pattern Analysis",
		"dataPoints", len(patterns),
		"analysisRange", fmt.Sprintf("%s to %s", 
			workWeek.WeekStart.Format("2006-01-02"),
			workWeek.WeekEnd.Format("2006-01-02")))

	// Show pattern insights
	activityTypes := make(map[string]int)
	totalIntensity := 0.0
	totalProductivity := 0.0

	for _, pattern := range patterns {
		activityTypes[pattern.ActivityType]++
		totalIntensity += pattern.Intensity
		totalProductivity += pattern.Productivity
	}

	if len(patterns) > 0 {
		avgIntensity := totalIntensity / float64(len(patterns))
		avgProductivity := totalProductivity / float64(len(patterns))

		ex.logger.Info("Pattern Insights",
			"avgIntensity", fmt.Sprintf("%.2f", avgIntensity),
			"avgProductivity", fmt.Sprintf("%.2f", avgProductivity),
			"activityBreakdown", activityTypes)
	}

	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-040
 * CONTEXT:   Example: Comprehensive timesheet generation with business policies
 * REASON:    Demonstrate formal timesheet creation with rounding policies and export preparation
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate timesheet calculations against business rules, test with various policies
 * RISK:      Low - Example timesheet generation for demonstration only
 */
func (ex *WorkHourIntegrationExample) TimesheetGenerationExample() error {
	ex.logger.Info("=== Timesheet Generation Example ===")

	// Define timesheet period (last week)
	now := time.Now()
	endDate := now.AddDate(0, 0, -int(now.Weekday())+1) // Last Monday
	startDate := endDate.AddDate(0, 0, -7)

	// Get timesheet data for the period
	entries, err := ex.whm.GetTimesheetData(startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to get timesheet data: %w", err)
	}

	// Create timesheet with business policy
	timesheet := &domain.Timesheet{
		ID:         fmt.Sprintf("timesheet_%s", startDate.Format("2006W02")),
		EmployeeID: "default",
		Period:     domain.TimesheetWeekly,
		StartDate:  startDate,
		EndDate:    endDate,
		Entries:    make([]domain.TimesheetEntry, 0),
		Policy: domain.TimesheetPolicy{
			RoundingInterval:  15 * time.Minute,
			RoundingMethod:    domain.RoundNearest,
			OvertimeThreshold: 8 * time.Hour,   // Daily overtime threshold
			WeeklyThreshold:   40 * time.Hour,  // Weekly overtime threshold
			BreakDeduction:    30 * time.Minute, // Automatic lunch break
		},
		Status:    domain.TimesheetDraft,
		CreatedAt: now,
	}

	// Convert database entries to timesheet entries
	for _, entry := range entries {
		timesheetEntry := domain.TimesheetEntry{
			ID:          entry.ID,
			Date:        entry.Date,
			StartTime:   entry.StartTime,
			EndTime:     entry.EndTime,
			Duration:    entry.Duration,
			Project:     entry.Project,
			Task:        entry.Task,
			Description: entry.Description,
			Billable:    entry.Billable,
		}
		timesheet.Entries = append(timesheet.Entries, timesheetEntry)
	}

	// Apply business policies
	timesheet.ApplyPolicy()

	ex.logger.Info("Timesheet Generated",
		"timesheetID", timesheet.ID,
		"period", string(timesheet.Period),
		"startDate", timesheet.StartDate.Format("2006-01-02"),
		"endDate", timesheet.EndDate.Format("2006-01-02"),
		"entriesCount", len(timesheet.Entries),
		"totalHours", timesheet.TotalHours,
		"regularHours", timesheet.RegularHours,
		"overtimeHours", timesheet.OvertimeHours,
		"status", string(timesheet.Status))

	// Save timesheet to database
	if err := ex.whm.SaveTimesheet(timesheet); err != nil {
		return fmt.Errorf("failed to save timesheet: %w", err)
	}

	ex.logger.Info("Timesheet saved successfully")

	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-041
 * CONTEXT:   Example: Trend analysis and productivity insights over time
 * REASON:    Demonstrate advanced analytics capabilities for long-term productivity optimization
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle large datasets efficiently, implement proper pagination for trend analysis
 * RISK:      Low - Trend analysis examples for demonstration purposes
 */
func (ex *WorkHourIntegrationExample) ProductivityTrendAnalysisExample() error {
	ex.logger.Info("=== Productivity Trend Analysis Example ===")

	// Analyze trends over the last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	// Get work time trends
	workTimeTrends, err := ex.whm.GetWorkTimeTrends(startDate, endDate, arch.GranularityDay)
	if err != nil {
		return fmt.Errorf("failed to get work time trends: %w", err)
	}

	// Get session count trends
	sessionTrends, err := ex.whm.GetSessionCountTrends(startDate, endDate, arch.GranularityDay)
	if err != nil {
		return fmt.Errorf("failed to get session trends: %w", err)
	}

	// Get efficiency trends
	efficiencyTrends, err := ex.whm.GetEfficiencyTrends(startDate, endDate, arch.GranularityDay)
	if err != nil {
		return fmt.Errorf("failed to get efficiency trends: %w", err)
	}

	ex.logger.Info("Trend Analysis Results",
		"analysisRange", fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		"workTimeTrends", len(workTimeTrends),
		"sessionTrends", len(sessionTrends),
		"efficiencyTrends", len(efficiencyTrends))

	// Calculate summary statistics
	if len(workTimeTrends) > 0 {
		totalWorkHours := 0.0
		avgChange := 0.0
		maxWorkDay := 0.0
		minWorkDay := 999999.0

		for _, trend := range workTimeTrends {
			totalWorkHours += trend.Value
			avgChange += trend.Change
			if trend.Value > maxWorkDay {
				maxWorkDay = trend.Value
			}
			if trend.Value < minWorkDay && trend.Value > 0 {
				minWorkDay = trend.Value
			}
		}

		avgWorkHours := totalWorkHours / float64(len(workTimeTrends))
		avgChange = avgChange / float64(len(workTimeTrends))

		ex.logger.Info("Work Time Trend Summary",
			"totalHours", fmt.Sprintf("%.2f", totalWorkHours),
			"avgDailyHours", fmt.Sprintf("%.2f", avgWorkHours),
			"avgChange", fmt.Sprintf("%.2f%%", avgChange),
			"maxDay", fmt.Sprintf("%.2f hours", maxWorkDay),
			"minDay", fmt.Sprintf("%.2f hours", minWorkDay))
	}

	// Analyze break patterns
	breakPatterns, err := ex.whm.GetBreakPatterns(startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to get break patterns: %w", err)
	}

	ex.logger.Info("Break Pattern Analysis",
		"patterns", len(breakPatterns))

	for _, pattern := range breakPatterns {
		ex.logger.Info("Break Pattern",
			"startHour", fmt.Sprintf("%02d:00", pattern.StartHour),
			"duration", pattern.Duration,
			"frequency", fmt.Sprintf("%.2f%%", pattern.Frequency*100),
			"type", string(pattern.Type))
	}

	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-042
 * CONTEXT:   Example: Cache management and performance optimization
 * REASON:    Demonstrate proper cache usage and optimization techniques for work hour queries
 * CHANGE:    Initial implementation.
 * PREVENTION:Monitor cache hit rates, implement cache warming strategies for better performance
 * RISK:      Low - Cache management examples for optimization guidance
 */
func (ex *WorkHourIntegrationExample) CacheOptimizationExample() error {
	ex.logger.Info("=== Cache Optimization Example ===")

	// Example 1: Check cached work day stats
	today := time.Now()
	cachedWorkDay, found := ex.whm.GetCachedWorkDayStats(today)
	if found {
		ex.logger.Info("Found cached work day data",
			"date", today.Format("2006-01-02"),
			"totalTime", cachedWorkDay.TotalTime,
			"cacheHit", true)
	} else {
		ex.logger.Info("Work day data not in cache, will calculate from database",
			"date", today.Format("2006-01-02"),
			"cacheHit", false)

		// This will calculate and cache the result
		workDay, err := ex.whm.GetWorkDayData(today)
		if err != nil {
			return fmt.Errorf("failed to get work day data: %w", err)
		}

		ex.logger.Info("Work day calculated and cached",
			"date", today.Format("2006-01-02"),
			"totalTime", workDay.TotalTime)
	}

	// Example 2: Refresh cache for specific date
	yesterday := today.AddDate(0, 0, -1)
	if err := ex.whm.RefreshWorkDayCache(yesterday); err != nil {
		return fmt.Errorf("failed to refresh cache: %w", err)
	}

	ex.logger.Info("Cache refreshed for date", "date", yesterday.Format("2006-01-02"))

	// Example 3: Invalidate cache for date range
	weekAgo := today.AddDate(0, 0, -7)
	if err := ex.whm.InvalidateWorkHourCache(weekAgo, today); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	ex.logger.Info("Cache invalidated for date range",
		"startDate", weekAgo.Format("2006-01-02"),
		"endDate", today.Format("2006-01-02"))

	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-043
 * CONTEXT:   Example: Complete migration from basic to work hour analytics
 * REASON:    Demonstrate safe migration process for upgrading existing installations
 * CHANGE:    Initial implementation.
 * PREVENTION:Always backup before migration, validate migration results thoroughly
 * RISK:      Low - Migration example for demonstration, not production migration
 */
func (ex *WorkHourIntegrationExample) MigrationExample() error {
	ex.logger.Info("=== Database Migration Example ===")

	// Create migrator
	migrator := NewDatabaseMigrator(ex.whm.db, ex.logger)

	// Check current schema version
	currentVersion, err := migrator.getCurrentSchemaVersion()
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	ex.logger.Info("Current schema version", "version", currentVersion)

	// Perform migration (this is safe to run multiple times)
	if err := migrator.MigrateToWorkHour(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Verify migration
	newVersion, err := migrator.getCurrentSchemaVersion()
	if err != nil {
		return fmt.Errorf("failed to verify migration: %w", err)
	}

	ex.logger.Info("Migration completed successfully",
		"fromVersion", currentVersion,
		"toVersion", newVersion)

	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-044
 * CONTEXT:   RunAllExamples executes comprehensive work hour database demonstration
 * REASON:    Provide complete walkthrough of work hour database capabilities
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle example failures gracefully, log all operations for debugging
 * RISK:      Low - Examples are for demonstration and don't affect production data
 */
func (ex *WorkHourIntegrationExample) RunAllExamples() error {
	ex.logger.Info("Running comprehensive work hour database examples")

	examples := []struct {
		name string
		fn   func() error
	}{
		{"Migration", ex.MigrationExample},
		{"Daily Analysis", ex.DailyWorkHourAnalysisExample},
		{"Weekly Patterns", ex.WeeklyWorkPatternExample},
		{"Timesheet Generation", ex.TimesheetGenerationExample},
		{"Trend Analysis", ex.ProductivityTrendAnalysisExample},
		{"Cache Optimization", ex.CacheOptimizationExample},
	}

	for _, example := range examples {
		ex.logger.Info("Running example", "name", example.name)
		
		if err := example.fn(); err != nil {
			ex.logger.Error("Example failed", "name", example.name, "error", err)
			continue // Continue with other examples
		}
		
		ex.logger.Info("Example completed successfully", "name", example.name)
		fmt.Println() // Add spacing between examples
	}

	ex.logger.Info("All work hour database examples completed")
	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-045
 * CONTEXT:   Usage example for CLI integration
 * REASON:    Show how CLI commands can leverage work hour database extensions
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep CLI examples synchronized with actual CLI implementation
 * RISK:      Low - CLI examples for reference purposes
 */
func ExampleCLIUsage() {
	// This function shows how CLI commands might use the work hour database

	fmt.Println(`
Work Hour Database CLI Integration Examples:

1. Daily Work Summary:
   ./claude-monitor report --type=daily --date=2024-01-15

2. Weekly Work Analysis:
   ./claude-monitor report --type=weekly --week-start=2024-01-15

3. Generate Timesheet:
   ./claude-monitor timesheet generate --period=weekly --start=2024-01-15

4. Productivity Analysis:
   ./claude-monitor analyze productivity --range=30days

5. Export Data:
   ./claude-monitor export --format=csv --start=2024-01-01 --end=2024-01-31

6. Cache Management:
   ./claude-monitor cache refresh --date=2024-01-15
   ./claude-monitor cache clear --range=7days

7. Database Migration:
   ./claude-monitor migrate --to-work-hour

These commands would leverage the WorkHourManager for:
- Comprehensive work hour analytics
- Professional timesheet generation
- Advanced productivity insights
- Efficient data caching
- Safe schema migrations
`)
}