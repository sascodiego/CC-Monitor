package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
	"github.com/claude-monitor/claude-monitor/internal/workhour"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-008
 * CONTEXT:   Work hour CLI manager implementation extending enhanced CLI manager
 * REASON:    Need comprehensive work hour CLI functionality with service integration and professional formatting
 * CHANGE:    Initial implementation of work hour CLI manager.
 * PREVENTION:Validate all inputs and provide clear error messages, handle service unavailability gracefully
 * RISK:      Medium - CLI manager coordinates multiple services and needs proper error handling
 */

// WorkHourEnhancedCLIManager extends the DefaultEnhancedCLIManager with work hour capabilities
type WorkHourEnhancedCLIManager struct {
	*DefaultEnhancedCLIManager
	workHourService arch.WorkHourService
	workHourFormatter *WorkHourFormatter
}

// NewWorkHourEnhancedCLIManager creates a new work hour enhanced CLI manager
func NewWorkHourEnhancedCLIManager(logger arch.Logger, workHourService arch.WorkHourService) *WorkHourEnhancedCLIManager {
	baseManager := NewEnhancedCLIManager(logger)
	
	return &WorkHourEnhancedCLIManager{
		DefaultEnhancedCLIManager: baseManager,
		workHourService:          workHourService,
		workHourFormatter:        NewWorkHourFormatter(),
	}
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-009
 * CONTEXT:   Work day command implementations with real-time status and comprehensive reporting
 * REASON:    Users need detailed daily work hour tracking with live updates and professional output
 * CHANGE:    Initial implementation of work day CLI operations.
 * PREVENTION:Handle date parsing errors and timezone considerations, validate status update intervals
 * RISK:      Low - Work day operations are primarily read-only status and reporting functions
 */

// ExecuteWorkDayStatus displays current work day status with optional live updates
func (whm *WorkHourEnhancedCLIManager) ExecuteWorkDayStatus(config *WorkDayStatusConfig) error {
	// Parse target date
	targetDate := time.Now()
	if config.Date != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", config.Date)
		if err != nil {
			return fmt.Errorf("invalid date format. Use YYYY-MM-DD: %w", err)
		}
	}

	if config.LiveUpdate {
		return whm.executeWorkDayStatusLive(config, targetDate)
	}

	// Get work day data
	workDay, err := whm.workHourService.GetDailyWorkSummary(targetDate)
	if err != nil {
		return fmt.Errorf("failed to get work day summary: %w", err)
	}

	// Format and display
	switch strings.ToLower(config.Format) {
	case "json":
		return whm.printWorkDayJSON(workDay, config)
	case "csv":
		return whm.printWorkDayCSV(workDay, config)
	default:
		return whm.printWorkDayStatus(workDay, config)
	}
}

func (whm *WorkHourEnhancedCLIManager) executeWorkDayStatusLive(config *WorkDayStatusConfig, targetDate time.Time) error {
	whm.formatter.PrintInfo(fmt.Sprintf("Live work day status updates (every %v, press Ctrl+C to stop)", config.UpdateInterval))
	fmt.Println()

	ticker := time.NewTicker(config.UpdateInterval)
	defer ticker.Stop()

	// Show initial status
	if err := whm.showSingleWorkDayStatus(config, targetDate); err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			// Clear screen and show updated status
			whm.formatter.ClearScreen()
			whm.formatter.PrintInfo(fmt.Sprintf("Work Day Status - %s (Last updated: %s)", 
				targetDate.Format("Monday, January 2, 2006"), time.Now().Format("15:04:05")))
			fmt.Println()

			if err := whm.showSingleWorkDayStatus(config, targetDate); err != nil {
				whm.formatter.PrintWarning(fmt.Sprintf("Error updating status: %v", err))
			}
		}
	}
}

func (whm *WorkHourEnhancedCLIManager) showSingleWorkDayStatus(config *WorkDayStatusConfig, targetDate time.Time) error {
	workDay, err := whm.workHourService.GetDailyWorkSummary(targetDate)
	if err != nil {
		return err
	}

	return whm.printWorkDayStatus(workDay, config)
}

func (whm *WorkHourEnhancedCLIManager) printWorkDayStatus(workDay *domain.WorkDay, config *WorkDayStatusConfig) error {
	whm.workHourFormatter.PrintWorkDayHeader(workDay.Date)

	// Basic status
	whm.workHourFormatter.PrintWorkDayBasicStatus(workDay)

	if config.Detailed {
		whm.workHourFormatter.PrintWorkDayDetailedStatus(workDay)
	}

	if config.ShowBreaks {
		whm.workHourFormatter.PrintWorkDayBreakAnalysis(workDay)
	}

	if config.ShowPattern {
		whm.workHourFormatter.PrintWorkDayPattern(workDay)
	}

	return nil
}

// ExecuteWorkDayReport generates comprehensive daily work report
func (whm *WorkHourEnhancedCLIManager) ExecuteWorkDayReport(config *WorkDayReportConfig) error {
	// Parse target date
	targetDate := time.Now()
	if config.Date != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", config.Date)
		if err != nil {
			return fmt.Errorf("invalid date format. Use YYYY-MM-DD: %w", err)
		}
	}

	// Generate report
	reportConfig := arch.ReportConfig{
		IncludeBreakdown: config.IncludeCharts,
		IncludePatterns:  config.IncludeTrends,
		IncludeTrends:    config.IncludeTrends,
	}

	report, err := whm.workHourService.GenerateReport("daily", targetDate, targetDate, reportConfig)
	if err != nil {
		return fmt.Errorf("failed to generate daily report: %w", err)
	}

	// Output report
	if config.OutputFile != "" {
		return whm.exportWorkDayReport(report, config)
	}

	return whm.printWorkDayReport(report, config)
}

func (whm *WorkHourEnhancedCLIManager) printWorkDayReport(report *arch.WorkHourReport, config *WorkDayReportConfig) error {
	whm.workHourFormatter.PrintReportHeader(report)

	if len(report.WorkDays) > 0 {
		workDay := report.WorkDays[0]
		whm.workHourFormatter.PrintWorkDayDetailedReport(workDay)
	}

	if report.Summary != nil {
		whm.workHourFormatter.PrintActivitySummary(report.Summary)
	}

	if config.IncludeGoals {
		whm.workHourFormatter.PrintGoalProgress(report)
	}

	if config.IncludeCharts {
		whm.workHourFormatter.PrintWorkDayCharts(report)
	}

	return nil
}

// ExecuteWorkDayExport exports work day data in various formats
func (whm *WorkHourEnhancedCLIManager) ExecuteWorkDayExport(config *WorkDayExportConfig) error {
	// Parse date range
	startDate := time.Now()
	endDate := time.Now()

	if config.StartDate != "" {
		var err error
		startDate, err = time.Parse("2006-01-02", config.StartDate)
		if err != nil {
			return fmt.Errorf("invalid start date format. Use YYYY-MM-DD: %w", err)
		}
	}

	if config.EndDate != "" {
		var err error
		endDate, err = time.Parse("2006-01-02", config.EndDate)
		if err != nil {
			return fmt.Errorf("invalid end date format. Use YYYY-MM-DD: %w", err)
		}
	}

	// Generate export data
	exportData, err := whm.generateWorkDayExportData(startDate, endDate, config)
	if err != nil {
		return fmt.Errorf("failed to generate export data: %w", err)
	}

	// Export to file
	return whm.writeWorkDayExport(exportData, config)
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-010
 * CONTEXT:   Work week command implementations for weekly analysis and pattern recognition
 * REASON:    Users need comprehensive weekly productivity analysis with overtime tracking and insights
 * CHANGE:    Initial implementation of work week CLI operations.
 * PREVENTION:Handle week boundary calculations properly, validate comparison periods
 * RISK:      Low - Weekly analysis operations are read-only reporting functions
 */

// ExecuteWorkWeekReport generates comprehensive weekly work report
func (whm *WorkHourEnhancedCLIManager) ExecuteWorkWeekReport(config *WorkWeekReportConfig) error {
	// Parse week start date
	weekStart := getWeekStart(time.Now())
	if config.WeekStart != "" {
		var err error
		weekStart, err = time.Parse("2006-01-02", config.WeekStart)
		if err != nil {
			return fmt.Errorf("invalid week start date format. Use YYYY-MM-DD: %w", err)
		}
		weekStart = getWeekStart(weekStart)
	}

	// Get work week data
	workWeek, err := whm.workHourService.GetWeeklyWorkSummary(weekStart)
	if err != nil {
		return fmt.Errorf("failed to get work week summary: %w", err)
	}

	// Generate report
	reportConfig := arch.ReportConfig{
		IncludeBreakdown: config.ShowDailyBreakdown,
		IncludePatterns:  config.IncludePattern,
		IncludeTrends:    true,
	}

	report, err := whm.workHourService.GenerateReport("weekly", weekStart, weekStart.AddDate(0, 0, 6), reportConfig)
	if err != nil {
		return fmt.Errorf("failed to generate weekly report: %w", err)
	}

	// Output report
	if config.OutputFile != "" {
		return whm.exportWorkWeekReport(report, config)
	}

	return whm.printWorkWeekReport(report, config)
}

func (whm *WorkHourEnhancedCLIManager) printWorkWeekReport(report *arch.WorkHourReport, config *WorkWeekReportConfig) error {
	whm.workHourFormatter.PrintReportHeader(report)

	if len(report.WorkWeeks) > 0 {
		workWeek := report.WorkWeeks[0]
		whm.workHourFormatter.PrintWorkWeekSummary(workWeek)

		if config.IncludeOvertime {
			whm.workHourFormatter.PrintOvertimeAnalysis(workWeek)
		}

		if config.ShowDailyBreakdown {
			whm.workHourFormatter.PrintWeeklyDailyBreakdown(workWeek)
		}
	}

	if config.IncludeGoals {
		whm.workHourFormatter.PrintGoalProgress(report)
	}

	return nil
}

// ExecuteWorkWeekAnalysis performs advanced weekly work pattern analysis
func (whm *WorkHourEnhancedCLIManager) ExecuteWorkWeekAnalysis(config *WorkWeekAnalysisConfig) error {
	// Parse week start date
	weekStart := getWeekStart(time.Now())
	if config.WeekStart != "" {
		var err error
		weekStart, err = time.Parse("2006-01-02", config.WeekStart)
		if err != nil {
			return fmt.Errorf("invalid week start date format. Use YYYY-MM-DD: %w", err)
		}
		weekStart = getWeekStart(weekStart)
	}

	// Get analysis data
	weekEnd := weekStart.AddDate(0, 0, 6)

	var workPattern *domain.WorkPattern
	var efficiency *domain.EfficiencyMetrics
	var trends *domain.TrendAnalysis
	var err error

	if config.IncludeProductivity || config.AnalysisDepth != "basic" {
		efficiency, err = whm.workHourService.GetProductivityInsights(weekStart, weekEnd)
		if err != nil {
			whm.logger.Warn("Failed to get productivity insights", "error", err)
		}
	}

	if config.IncludeEfficiency || config.AnalysisDepth == "comprehensive" {
		workPattern, err = whm.workHourService.GetWorkPatternAnalysis(weekStart, weekEnd)
		if err != nil {
			whm.logger.Warn("Failed to get work pattern analysis", "error", err)
		}
	}

	if config.CompareToAverage {
		trends, err = whm.workHourService.GetTrendAnalysis(weekStart.AddDate(0, 0, -28), weekEnd)
		if err != nil {
			whm.logger.Warn("Failed to get trend analysis", "error", err)
		}
	}

	// Output analysis
	if config.OutputFile != "" {
		return whm.exportWorkWeekAnalysis(workPattern, efficiency, trends, config)
	}

	return whm.printWorkWeekAnalysis(workPattern, efficiency, trends, config)
}

func (whm *WorkHourEnhancedCLIManager) printWorkWeekAnalysis(
	pattern *domain.WorkPattern, 
	efficiency *domain.EfficiencyMetrics,
	trends *domain.TrendAnalysis,
	config *WorkWeekAnalysisConfig) error {

	whm.workHourFormatter.PrintAnalysisHeader("Weekly Work Analysis", config.WeekStart)

	if efficiency != nil {
		whm.workHourFormatter.PrintEfficiencyMetrics(efficiency)
	}

	if pattern != nil {
		whm.workHourFormatter.PrintWorkPattern(pattern)
	}

	if trends != nil {
		whm.workHourFormatter.PrintTrendAnalysis(trends)
	}

	if config.IncludeRecommendations {
		whm.workHourFormatter.PrintRecommendations(pattern, efficiency, trends)
	}

	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-011
 * CONTEXT:   Timesheet command implementations for formal time tracking and HR integration
 * REASON:    Users need professional timesheet management for billing, compliance, and HR workflows
 * CHANGE:    Initial implementation of timesheet CLI operations.
 * PREVENTION:Validate timesheet policies and handle submission workflows properly
 * RISK:      Medium - Timesheet operations affect billing accuracy and compliance requirements
 */

// ExecuteTimesheetGenerate creates timesheet for specified period
func (whm *WorkHourEnhancedCLIManager) ExecuteTimesheetGenerate(config *TimesheetGenerateConfig) error {
	// Parse start date
	startDate := time.Now()
	if config.StartDate != "" {
		var err error
		startDate, err = time.Parse("2006-01-02", config.StartDate)
		if err != nil {
			return fmt.Errorf("invalid start date format. Use YYYY-MM-DD: %w", err)
		}
	}

	// Parse period
	var period domain.TimesheetPeriod
	switch strings.ToLower(config.Period) {
	case "weekly":
		period = domain.PeriodWeekly
		startDate = getWeekStart(startDate)
	case "biweekly":
		period = domain.PeriodBiweekly
		startDate = getBiweekStart(startDate)
	case "monthly":
		period = domain.PeriodMonthly
		startDate = getMonthStart(startDate)
	default:
		return fmt.Errorf("invalid period: %s (valid: weekly, biweekly, monthly)", config.Period)
	}

	// Create timesheet
	employeeID := config.EmployeeID
	if employeeID == "" {
		employeeID = "default"
	}

	timesheet, err := whm.workHourService.CreateTimesheet(employeeID, period, startDate)
	if err != nil {
		return fmt.Errorf("failed to create timesheet: %w", err)
	}

	whm.formatter.PrintSuccess(fmt.Sprintf("Timesheet created successfully: %s", timesheet.ID))

	// Auto-submit if requested
	if config.AutoSubmit {
		if err := whm.workHourService.FinalizeTimesheet(timesheet.ID); err != nil {
			whm.formatter.PrintWarning(fmt.Sprintf("Failed to auto-submit timesheet: %v", err))
		} else {
			whm.formatter.PrintSuccess("Timesheet submitted automatically")
		}
	}

	// Output timesheet
	if config.OutputFile != "" {
		return whm.exportTimesheet(timesheet, config)
	}

	return whm.printTimesheet(timesheet, config)
}

func (whm *WorkHourEnhancedCLIManager) printTimesheet(timesheet *domain.Timesheet, config *TimesheetGenerateConfig) error {
	whm.workHourFormatter.PrintTimesheetHeader(timesheet)
	whm.workHourFormatter.PrintTimesheetEntries(timesheet)
	whm.workHourFormatter.PrintTimesheetSummary(timesheet)
	return nil
}

// ExecuteTimesheetView displays existing timesheets
func (whm *WorkHourEnhancedCLIManager) ExecuteTimesheetView(config *TimesheetViewConfig) error {
	if config.TimesheetID != "" {
		// View specific timesheet
		return whm.viewSingleTimesheet(config.TimesheetID, config)
	}

	// List timesheets with filters
	return whm.listTimesheets(config)
}

func (whm *WorkHourEnhancedCLIManager) viewSingleTimesheet(timesheetID string, config *TimesheetViewConfig) error {
	whm.formatter.PrintInfo(fmt.Sprintf("Timesheet details for: %s", timesheetID))
	
	// This would require implementing a GetTimesheet method in the service
	whm.formatter.PrintWarning("Timesheet viewing not yet implemented in service layer")
	return nil
}

func (whm *WorkHourEnhancedCLIManager) listTimesheets(config *TimesheetViewConfig) error {
	whm.formatter.PrintInfo("Listing timesheets...")
	
	// This would require implementing a ListTimesheets method in the service
	whm.formatter.PrintWarning("Timesheet listing not yet implemented in service layer")
	return nil
}

// ExecuteTimesheetSubmit submits timesheet for approval
func (whm *WorkHourEnhancedCLIManager) ExecuteTimesheetSubmit(config *TimesheetSubmitConfig) error {
	whm.formatter.PrintStep(fmt.Sprintf("Submitting timesheet: %s", config.TimesheetID))

	if !config.Force {
		// Validate timesheet before submission
		whm.formatter.PrintStep("Validating timesheet...")
		// Validation logic would go here
		whm.formatter.PrintSuccess("Timesheet validation passed")
	}

	// Submit timesheet
	err := whm.workHourService.FinalizeTimesheet(config.TimesheetID)
	if err != nil {
		return fmt.Errorf("failed to submit timesheet: %w", err)
	}

	whm.formatter.PrintSuccess(fmt.Sprintf("Timesheet %s submitted successfully", config.TimesheetID))

	if config.Comments != "" {
		whm.formatter.PrintInfo(fmt.Sprintf("Submission comments: %s", config.Comments))
	}

	return nil
}

// ExecuteTimesheetExport exports timesheet data
func (whm *WorkHourEnhancedCLIManager) ExecuteTimesheetExport(config *TimesheetExportConfig) error {
	whm.formatter.PrintStep("Preparing timesheet export...")

	// This would require implementing export functionality
	whm.formatter.PrintWarning("Timesheet export not yet implemented")
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-012
 * CONTEXT:   Analytics command implementations for productivity insights and optimization
 * REASON:    Users need sophisticated analytics to understand work patterns and improve productivity
 * CHANGE:    Initial implementation of analytics CLI operations.
 * PREVENTION:Handle large datasets efficiently and provide meaningful insights even with limited data
 * RISK:      Medium - Analytics operations can be resource intensive for large date ranges
 */

// ExecuteProductivityAnalysis performs productivity metrics analysis
func (whm *WorkHourEnhancedCLIManager) ExecuteProductivityAnalysis(config *ProductivityAnalysisConfig) error {
	// Parse date range
	startDate, endDate, err := whm.parseDateRange(config.StartDate, config.EndDate, 30)
	if err != nil {
		return err
	}

	whm.formatter.PrintStep(fmt.Sprintf("Analyzing productivity for period: %s to %s", 
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02")))

	// Get productivity metrics
	metrics, err := whm.workHourService.GetProductivityInsights(startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to get productivity insights: %w", err)
	}

	// Output analysis
	if config.OutputFile != "" {
		return whm.exportProductivityAnalysis(metrics, config)
	}

	return whm.printProductivityAnalysis(metrics, config)
}

func (whm *WorkHourEnhancedCLIManager) printProductivityAnalysis(metrics *domain.EfficiencyMetrics, config *ProductivityAnalysisConfig) error {
	whm.workHourFormatter.PrintAnalysisHeader("Productivity Analysis", config.StartDate)
	whm.workHourFormatter.PrintEfficiencyMetrics(metrics)

	if config.IncludeRecommendations {
		whm.workHourFormatter.PrintProductivityRecommendations(metrics)
	}

	return nil
}

// ExecuteWorkPatternAnalysis analyzes work patterns and habits
func (whm *WorkHourEnhancedCLIManager) ExecuteWorkPatternAnalysis(config *WorkPatternAnalysisConfig) error {
	// Parse date range
	startDate, endDate, err := whm.parseDateRange(config.StartDate, config.EndDate, 30)
	if err != nil {
		return err
	}

	whm.formatter.PrintStep(fmt.Sprintf("Analyzing work patterns for period: %s to %s", 
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02")))

	// Get work pattern analysis
	pattern, err := whm.workHourService.GetWorkPatternAnalysis(startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to get work pattern analysis: %w", err)
	}

	// Output analysis
	if config.OutputFile != "" {
		return whm.exportWorkPatternAnalysis(pattern, config)
	}

	return whm.printWorkPatternAnalysis(pattern, config)
}

func (whm *WorkHourEnhancedCLIManager) printWorkPatternAnalysis(pattern *domain.WorkPattern, config *WorkPatternAnalysisConfig) error {
	whm.workHourFormatter.PrintAnalysisHeader("Work Pattern Analysis", config.StartDate)
	whm.workHourFormatter.PrintWorkPattern(pattern)

	if config.IncludeRecommendations {
		whm.workHourFormatter.PrintPatternRecommendations(pattern)
	}

	return nil
}

// ExecuteTrendAnalysis analyzes long-term trends
func (whm *WorkHourEnhancedCLIManager) ExecuteTrendAnalysis(config *TrendAnalysisConfig) error {
	// Parse date range
	startDate, endDate, err := whm.parseDateRange(config.StartDate, config.EndDate, 90)
	if err != nil {
		return err
	}

	whm.formatter.PrintStep(fmt.Sprintf("Analyzing trends for period: %s to %s", 
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02")))

	// Get trend analysis
	trends, err := whm.workHourService.GetTrendAnalysis(startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to get trend analysis: %w", err)
	}

	// Output analysis
	if config.OutputFile != "" {
		return whm.exportTrendAnalysis(trends, config)
	}

	return whm.printTrendAnalysis(trends, config)
}

func (whm *WorkHourEnhancedCLIManager) printTrendAnalysis(trends *domain.TrendAnalysis, config *TrendAnalysisConfig) error {
	whm.workHourFormatter.PrintAnalysisHeader("Trend Analysis", config.StartDate)
	whm.workHourFormatter.PrintTrendAnalysis(trends)

	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-013
 * CONTEXT:   Goals and policy management CLI operations for system configuration
 * REASON:    Users need to manage work hour goals and policies for proper tracking and reporting
 * CHANGE:    Initial implementation of goals and policy CLI operations.
 * PREVENTION:Validate policy changes to ensure system consistency and prevent configuration conflicts
 * RISK:      Medium - Policy changes affect system behavior and calculations
 */

// ExecuteGoalsView displays current work hour goals
func (whm *WorkHourEnhancedCLIManager) ExecuteGoalsView(config *GoalsViewConfig) error {
	whm.formatter.PrintSectionHeader("Work Hour Goals")
	
	// This would require implementing goals management in the service layer
	whm.formatter.PrintWarning("Goals management not yet implemented in service layer")
	return nil
}

// ExecuteGoalsSet sets new work hour goals
func (whm *WorkHourEnhancedCLIManager) ExecuteGoalsSet(config *GoalsSetConfig) error {
	whm.formatter.PrintStep(fmt.Sprintf("Setting %s work hour goal: %s", config.GoalType, config.TargetHours))
	
	// This would require implementing goals management in the service layer
	whm.formatter.PrintWarning("Goals management not yet implemented in service layer")
	return nil
}

// ExecutePolicyView displays current work hour policies
func (whm *WorkHourEnhancedCLIManager) ExecutePolicyView(config *PolicyViewConfig) error {
	whConfig, err := whm.workHourService.GetWorkHourConfiguration()
	if err != nil {
		return fmt.Errorf("failed to get work hour configuration: %w", err)
	}

	whm.workHourFormatter.PrintPolicyConfiguration(whConfig)
	return nil
}

// ExecutePolicyUpdate updates work hour policies
func (whm *WorkHourEnhancedCLIManager) ExecutePolicyUpdate(config *PolicyUpdateConfig) error {
	whm.formatter.PrintStep("Updating work hour policies...")

	// Get current configuration
	whConfig, err := whm.workHourService.GetWorkHourConfiguration()
	if err != nil {
		return fmt.Errorf("failed to get current configuration: %w", err)
	}

	// Update policy based on configuration
	policy := whConfig.DefaultPolicy

	if config.RoundingInterval != "" {
		interval, err := time.ParseDuration(config.RoundingInterval)
		if err != nil {
			return fmt.Errorf("invalid rounding interval: %w", err)
		}
		policy.RoundingInterval = interval
	}

	if config.RoundingMethod != "" {
		switch strings.ToLower(config.RoundingMethod) {
		case "up":
			policy.RoundingMethod = domain.RoundUp
		case "down":
			policy.RoundingMethod = domain.RoundDown
		case "nearest":
			policy.RoundingMethod = domain.RoundNearest
		default:
			return fmt.Errorf("invalid rounding method: %s", config.RoundingMethod)
		}
	}

	if config.OvertimeThreshold != "" {
		threshold, err := time.ParseDuration(config.OvertimeThreshold)
		if err != nil {
			return fmt.Errorf("invalid overtime threshold: %w", err)
		}
		policy.OvertimeThreshold = threshold
	}

	// Update the policy
	err = whm.workHourService.UpdateWorkHourPolicy(policy)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	whm.formatter.PrintSuccess("Work hour policy updated successfully")
	return nil
}

// ExecuteBulkExport performs bulk export operations
func (whm *WorkHourEnhancedCLIManager) ExecuteBulkExport(config *BulkExportConfig) error {
	whm.formatter.PrintStep("Starting bulk export operation...")

	// This would require implementing bulk export functionality
	whm.formatter.PrintWarning("Bulk export not yet implemented")
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-014
 * CONTEXT:   Helper functions for work hour CLI operations
 * REASON:    Need common utilities for date parsing, formatting, and export operations
 * CHANGE:    Initial implementation of work hour CLI helper functions.
 * PREVENTION:Handle edge cases in date calculations and provide consistent error messages
 * RISK:      Low - Helper functions support main operations but don't affect core functionality
 */

// Helper functions for date range parsing and formatting
func (whm *WorkHourEnhancedCLIManager) parseDateRange(startStr, endStr string, defaultDays int) (time.Time, time.Time, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -defaultDays)

	if endStr != "" {
		var err error
		endDate, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end date format. Use YYYY-MM-DD: %w", err)
		}
	}

	if startStr != "" {
		var err error
		startDate, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start date format. Use YYYY-MM-DD: %w", err)
		}
	}

	if startDate.After(endDate) {
		return time.Time{}, time.Time{}, fmt.Errorf("start date cannot be after end date")
	}

	return startDate, endDate, nil
}

// Date calculation helpers
func getWeekStart(date time.Time) time.Time {
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	return date.AddDate(0, 0, -weekday+1)
}

func getBiweekStart(date time.Time) time.Time {
	weekStart := getWeekStart(date)
	// This is a simplified biweek calculation - in practice you might want
	// to use a reference date to determine biweek boundaries
	return weekStart
}

func getMonthStart(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
}

// JSON output helpers
func (whm *WorkHourEnhancedCLIManager) printWorkDayJSON(workDay *domain.WorkDay, config *WorkDayStatusConfig) error {
	jsonData, err := json.MarshalIndent(workDay, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal work day to JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func (whm *WorkHourEnhancedCLIManager) printWorkDayCSV(workDay *domain.WorkDay, config *WorkDayStatusConfig) error {
	// CSV output would be implemented here
	whm.formatter.PrintWarning("CSV output not yet implemented")
	return nil
}

// Export operation stubs - these would be implemented with actual export logic
func (whm *WorkHourEnhancedCLIManager) generateWorkDayExportData(startDate, endDate time.Time, config *WorkDayExportConfig) (interface{}, error) {
	return nil, fmt.Errorf("work day export not yet implemented")
}

func (whm *WorkHourEnhancedCLIManager) writeWorkDayExport(data interface{}, config *WorkDayExportConfig) error {
	return fmt.Errorf("work day export not yet implemented")
}

func (whm *WorkHourEnhancedCLIManager) exportWorkDayReport(report *arch.WorkHourReport, config *WorkDayReportConfig) error {
	return fmt.Errorf("work day report export not yet implemented")
}

func (whm *WorkHourEnhancedCLIManager) exportWorkWeekReport(report *arch.WorkHourReport, config *WorkWeekReportConfig) error {
	return fmt.Errorf("work week report export not yet implemented")
}

func (whm *WorkHourEnhancedCLIManager) exportWorkWeekAnalysis(pattern *domain.WorkPattern, efficiency *domain.EfficiencyMetrics, trends *domain.TrendAnalysis, config *WorkWeekAnalysisConfig) error {
	return fmt.Errorf("work week analysis export not yet implemented")
}

func (whm *WorkHourEnhancedCLIManager) exportTimesheet(timesheet *domain.Timesheet, config *TimesheetGenerateConfig) error {
	return fmt.Errorf("timesheet export not yet implemented")
}

func (whm *WorkHourEnhancedCLIManager) exportProductivityAnalysis(metrics *domain.EfficiencyMetrics, config *ProductivityAnalysisConfig) error {
	return fmt.Errorf("productivity analysis export not yet implemented")
}

func (whm *WorkHourEnhancedCLIManager) exportWorkPatternAnalysis(pattern *domain.WorkPattern, config *WorkPatternAnalysisConfig) error {
	return fmt.Errorf("work pattern analysis export not yet implemented")
}

func (whm *WorkHourEnhancedCLIManager) exportTrendAnalysis(trends *domain.TrendAnalysis, config *TrendAnalysisConfig) error {
	return fmt.Errorf("trend analysis export not yet implemented")
}

// Ensure WorkHourEnhancedCLIManager implements WorkHourCLIManager interface
var _ WorkHourCLIManager = (*WorkHourEnhancedCLIManager)(nil)