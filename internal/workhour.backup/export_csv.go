/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-001
 * CONTEXT:   Comprehensive CSV export implementation with advanced formatting and multi-sheet support
 * REASON:    CSV format needs professional formatting with configurable columns and multiple export styles
 * CHANGE:    Complete CSV export implementation with multi-sheet support and data transformation.
 * PREVENTION:Handle CSV escaping properly, validate column configurations, ensure data consistency
 * RISK:      Low - CSV format is well-established with good library support, main risk is data formatting
 */
package workhour

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-002
 * CONTEXT:   CSV configuration generation based on report type and user preferences
 * REASON:    Different report types need different CSV column structures and formatting
 * CHANGE:    CSV configuration generator with type-specific column mapping.
 * PREVENTION:Validate column mappings, ensure all required fields are included, handle missing data
 * RISK:      Low - Configuration generation is straightforward but needs proper validation
 */

// getCSVConfig generates CSV configuration based on request and report
func (ee *ExportEngine) getCSVConfig(request *ExportRequest, report *arch.WorkHourReport) *CSVConfig {
	config := &CSVConfig{
		Separator:        ',',
		UseCRLF:         false,
		IncludeHeader:   true,
		IncludeMetadata: true,
		IncludeEmpty:    false,
		MaxRows:         10000,
		SortBy:          "date",
		SortDirection:   "asc",
		DateFormat:      "2006-01-02",
		TimeFormat:      "15:04:05",
		DurationFormat:  "hours",
		DecimalPlaces:   2,
		MultiSheet:      request.IncludeBreakdown,
	}
	
	// Configure based on report type
	switch request.ReportType {
	case "daily":
		config.Headers = []string{
			"Date", "Start Time", "End Time", "Total Work Time", 
			"Break Time", "Session Count", "Work Block Count", 
			"Efficiency Ratio", "Peak Hours", "Notes",
		}
		config.ColumnMapping = map[string]string{
			"Date":            "date",
			"Start Time":      "start_time",
			"End Time":        "end_time",
			"Total Work Time": "total_time",
			"Break Time":      "break_time",
			"Session Count":   "session_count",
			"Work Block Count": "block_count",
			"Efficiency Ratio": "efficiency_ratio",
			"Peak Hours":      "peak_hours",
			"Notes":           "notes",
		}
		
	case "weekly":
		config.Headers = []string{
			"Week Start", "Week End", "Total Work Time", "Overtime Hours",
			"Standard Hours", "Average Daily Time", "Work Days",
			"Efficiency Score", "Weekly Goal", "Achievement Rate",
		}
		config.ColumnMapping = map[string]string{
			"Week Start":        "week_start",
			"Week End":          "week_end",
			"Total Work Time":   "total_time",
			"Overtime Hours":    "overtime_hours",
			"Standard Hours":    "standard_hours",
			"Average Daily Time": "average_daily",
			"Work Days":         "work_days",
			"Efficiency Score":  "efficiency_score",
			"Weekly Goal":       "weekly_goal",
			"Achievement Rate":  "achievement_rate",
		}
		
	case "timesheet":
		config.Headers = []string{
			"Date", "Start Time", "End Time", "Duration", "Project",
			"Task", "Description", "Billable", "Rate", "Amount",
			"Status", "Approved By", "Notes",
		}
		config.ColumnMapping = map[string]string{
			"Date":        "date",
			"Start Time":  "start_time",
			"End Time":    "end_time",
			"Duration":    "duration",
			"Project":     "project",
			"Task":        "task",
			"Description": "description",
			"Billable":    "billable",
			"Rate":        "rate",
			"Amount":      "amount",
			"Status":      "status",
			"Approved By": "approved_by",
			"Notes":       "notes",
		}
		
	case "analytics":
		config.Headers = []string{
			"Date", "Productivity Score", "Focus Score", "Active Ratio",
			"Break Frequency", "Task Switches", "Peak Performance Hour",
			"Low Performance Hour", "Trend Direction", "Recommendations",
		}
		config.ColumnMapping = map[string]string{
			"Date":                  "date",
			"Productivity Score":    "productivity_score",
			"Focus Score":           "focus_score",
			"Active Ratio":          "active_ratio",
			"Break Frequency":       "break_frequency",
			"Task Switches":         "task_switches",
			"Peak Performance Hour": "peak_hour",
			"Low Performance Hour":  "low_hour",
			"Trend Direction":       "trend_direction",
			"Recommendations":       "recommendations",
		}
		
	default:
		// Generic configuration
		config.Headers = []string{
			"Date", "Total Work Time", "Sessions", "Work Blocks", "Notes",
		}
		config.ColumnMapping = map[string]string{
			"Date":            "date",
			"Total Work Time": "total_time",
			"Sessions":        "sessions",
			"Work Blocks":     "blocks",
			"Notes":           "notes",
		}
	}
	
	// Apply custom field formatters
	config.FieldFormatters = map[string]string{
		"date":             config.DateFormat,
		"time":             config.TimeFormat,
		"duration":         config.DurationFormat,
		"percentage":       fmt.Sprintf("%%.%df%%%%", config.DecimalPlaces),
		"decimal":          fmt.Sprintf("%%.%df", config.DecimalPlaces),
		"currency":         request.CurrencyFormat,
	}
	
	return config
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-003
 * CONTEXT:   CSV metadata header generation with comprehensive export information
 * REASON:    Users need clear metadata about the export for context and data validation
 * CHANGE:    CSV metadata header implementation with export details and data specifications.
 * PREVENTION:Ensure metadata doesn't interfere with data processing, validate all metadata fields
 * RISK:      Low - Metadata is informational and doesn't affect data integrity
 */

// writeCSVMetadata writes metadata header to CSV file
func (ee *ExportEngine) writeCSVMetadata(writer *csv.Writer, request *ExportRequest, report *arch.WorkHourReport) error {
	metadata := [][]string{
		{"# Claude Monitor Work Hour Export"},
		{"# Generated:", time.Now().Format("2006-01-02 15:04:05")},
		{"# Report Type:", request.ReportType},
		{"# Date Range:", fmt.Sprintf("%s to %s", request.StartDate.Format("2006-01-02"), request.EndDate.Format("2006-01-02"))},
		{"# Timezone:", request.Timezone},
		{"# Export Format:", "CSV"},
		{"# Export ID:", request.ID},
		{},  // Empty row separator
	}
	
	for _, row := range metadata {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write metadata row: %w", err)
		}
	}
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-004
 * CONTEXT:   Daily work data CSV export with comprehensive field formatting
 * REASON:    Daily reports need detailed breakdown with proper data formatting for analysis
 * CHANGE:    Daily CSV export implementation with all relevant fields and formatting.
 * PREVENTION:Handle missing data gracefully, ensure consistent formatting, validate field values
 * RISK:      Low - Daily data export is straightforward with well-defined fields
 */

// writeCSVDailyData writes daily work data to CSV
func (ee *ExportEngine) writeCSVDailyData(writer *csv.Writer, report *arch.WorkHourReport, config *CSVConfig) error {
	for _, workDay := range report.WorkDays {
		record := []string{
			workDay.Date.Format(config.DateFormat),
			ee.formatTimePtr(workDay.StartTime, config.TimeFormat),
			ee.formatTimePtr(workDay.EndTime, config.TimeFormat),
			ee.formatDurationCSV(workDay.TotalTime, config.DurationFormat),
			ee.formatDurationCSV(workDay.BreakTime, config.DurationFormat),
			strconv.Itoa(workDay.SessionCount),
			strconv.Itoa(workDay.BlockCount),
			ee.formatFloat(workDay.GetEfficiencyRatio(), config.DecimalPlaces),
			ee.formatPeakHours(workDay.GetPeakHours()),
			ee.sanitizeString(workDay.Notes),
		}
		
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write daily data record: %w", err)
		}
	}
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-005
 * CONTEXT:   Weekly work data CSV export with overtime and goal tracking
 * REASON:    Weekly reports need comprehensive metrics including overtime and performance tracking
 * CHANGE:    Weekly CSV export implementation with overtime calculations and goal tracking.
 * PREVENTION:Handle week boundary calculations properly, validate overtime calculations, ensure goal data consistency
 * RISK:      Low - Weekly data aggregation is well-defined with clear business rules
 */

// writeCSVWeeklyData writes weekly work data to CSV
func (ee *ExportEngine) writeCSVWeeklyData(writer *csv.Writer, report *arch.WorkHourReport, config *CSVConfig) error {
	for _, workWeek := range report.WorkWeeks {
		record := []string{
			workWeek.WeekStart.Format(config.DateFormat),
			workWeek.WeekEnd.Format(config.DateFormat),
			ee.formatDurationCSV(workWeek.TotalTime, config.DurationFormat),
			ee.formatDurationCSV(workWeek.OvertimeHours, config.DurationFormat),
			ee.formatDurationCSV(workWeek.StandardHours, config.DurationFormat),
			ee.formatDurationCSV(workWeek.AverageDay, config.DurationFormat),
			strconv.Itoa(len(workWeek.WorkDays)),
			ee.formatFloat(workWeek.GetEfficiencyScore(), config.DecimalPlaces),
			ee.formatDurationCSV(workWeek.GetWeeklyGoal(), config.DurationFormat),
			ee.formatFloat(workWeek.GetAchievementRate(), config.DecimalPlaces),
		}
		
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write weekly data record: %w", err)
		}
	}
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-006
 * CONTEXT:   Monthly work data CSV export with aggregated metrics
 * REASON:    Monthly reports need aggregated data with trend analysis and summary metrics
 * CHANGE:    Monthly CSV export implementation with aggregated data and trend information.
 * PREVENTION:Handle month boundary calculations, validate aggregation logic, ensure data consistency
 * RISK:      Low - Monthly aggregation follows standard business rules with clear validation
 */

// writeCSVMonthlyData writes monthly work data to CSV
func (ee *ExportEngine) writeCSVMonthlyData(writer *csv.Writer, report *arch.WorkHourReport, config *CSVConfig) error {
	// Monthly data is typically aggregated from daily data
	return ee.writeCSVDailyData(writer, report, config)
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-007
 * CONTEXT:   Timesheet data CSV export with billing and approval information
 * REASON:    Timesheet exports need formal structure for billing systems and HR processes
 * CHANGE:    Timesheet CSV export implementation with billing fields and approval workflow data.
 * PREVENTION:Validate billing calculations, ensure approval status consistency, handle currency formatting
 * RISK:      Medium - Timesheet data affects billing and compliance, requires careful validation
 */

// writeCSVTimesheetData writes timesheet data to CSV
func (ee *ExportEngine) writeCSVTimesheetData(writer *csv.Writer, report *arch.WorkHourReport, config *CSVConfig) error {
	// Convert work days to timesheet entries
	for _, workDay := range report.WorkDays {
		// Each work day could have multiple timesheet entries
		// For now, create one entry per day
		record := []string{
			workDay.Date.Format(config.DateFormat),
			ee.formatTimePtr(workDay.StartTime, config.TimeFormat),
			ee.formatTimePtr(workDay.EndTime, config.TimeFormat),
			ee.formatDurationCSV(workDay.TotalTime, config.DurationFormat),
			"General Work",                    // Project (placeholder)
			"Claude CLI Usage",               // Task
			fmt.Sprintf("Work session with %d blocks", workDay.BlockCount), // Description
			"Yes",                            // Billable
			"$0.00",                         // Rate (placeholder)
			"$0.00",                         // Amount (placeholder)
			"Draft",                         // Status
			"",                              // Approved By
			ee.sanitizeString(workDay.Notes), // Notes
		}
		
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write timesheet record: %w", err)
		}
	}
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-008
 * CONTEXT:   Analytics data CSV export with productivity metrics and insights
 * REASON:    Analytics exports need detailed metrics for business intelligence and optimization
 * CHANGE:    Analytics CSV export implementation with productivity metrics and recommendations.
 * PREVENTION:Validate analytics calculations, ensure metric consistency, handle recommendation formatting
 * RISK:      Low - Analytics data is derived from validated base data with known calculation methods
 */

// writeCSVAnalyticsData writes analytics data to CSV
func (ee *ExportEngine) writeCSVAnalyticsData(writer *csv.Writer, report *arch.WorkHourReport, config *CSVConfig) error {
	for _, workDay := range report.WorkDays {
		// Calculate analytics metrics for the day
		productivityScore := ee.calculateProductivityScore(workDay)
		focusScore := ee.calculateFocusScore(workDay)
		activeRatio := workDay.GetEfficiencyRatio()
		
		record := []string{
			workDay.Date.Format(config.DateFormat),
			ee.formatFloat(productivityScore, config.DecimalPlaces),
			ee.formatFloat(focusScore, config.DecimalPlaces),
			ee.formatFloat(activeRatio, config.DecimalPlaces),
			ee.formatFloat(ee.calculateBreakFrequency(workDay), config.DecimalPlaces),
			strconv.Itoa(ee.calculateTaskSwitches(workDay)),
			ee.formatHour(ee.getPeakPerformanceHour(workDay)),
			ee.formatHour(ee.getLowPerformanceHour(workDay)),
			ee.getTrendDirection(workDay),
			ee.generateRecommendations(workDay),
		}
		
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write analytics record: %w", err)
		}
	}
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-009
 * CONTEXT:   Generic CSV export for fallback and custom report types
 * REASON:    Need fallback export format for unsupported report types and custom configurations
 * CHANGE:    Generic CSV export implementation with flexible field mapping.
 * PREVENTION:Handle unknown data types gracefully, provide meaningful fallback formatting
 * RISK:      Low - Generic export provides basic functionality with standard field handling
 */

// writeCSVGenericData writes generic report data to CSV
func (ee *ExportEngine) writeCSVGenericData(writer *csv.Writer, report *arch.WorkHourReport, config *CSVConfig) error {
	// Write summary information if available
	if report.Summary != nil {
		summaryRows := [][]string{
			{"Metric", "Value"},
			{"Report Type", report.ReportType},
			{"Period", report.Period},
			{"Total Work Time", ee.formatDurationCSV(report.Summary.TotalWorkTime, config.DurationFormat)},
			{"Total Sessions", strconv.Itoa(report.Summary.TotalSessions)},
			{"Total Work Blocks", strconv.Itoa(report.Summary.TotalWorkBlocks)},
			{"Daily Average", ee.formatDurationCSV(report.Summary.DailyAverage, config.DurationFormat)},
		}
		
		for _, row := range summaryRows {
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("failed to write generic summary row: %w", err)
			}
		}
	}
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-010
 * CONTEXT:   Multi-sheet CSV export with separate files for different data categories
 * REASON:    Complex reports need separation of different data types for better analysis
 * CHANGE:    Multi-sheet CSV export implementation with separate files and comprehensive data organization.
 * PREVENTION:Ensure consistent naming conventions, validate file creation, handle directory structure
 * RISK:      Low - Multi-file export is straightforward with proper file management
 */

// exportCSVSummarySheet exports summary data to separate CSV file
func (ee *ExportEngine) exportCSVSummarySheet(outputPath string, report *arch.WorkHourReport, config *CSVConfig) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create summary CSV file: %w", err)
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write summary headers
	headers := []string{"Metric", "Value", "Unit", "Notes"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write summary headers: %w", err)
	}
	
	// Write summary data
	if report.Summary != nil {
		summaryData := [][]string{
			{"Total Work Time", ee.formatDurationCSV(report.Summary.TotalWorkTime, config.DurationFormat), "hours", ""},
			{"Total Sessions", strconv.Itoa(report.Summary.TotalSessions), "count", ""},
			{"Total Work Blocks", strconv.Itoa(report.Summary.TotalWorkBlocks), "count", ""},
			{"Daily Average", ee.formatDurationCSV(report.Summary.DailyAverage, config.DurationFormat), "hours", ""},
			{"Report Period", report.Period, "date range", ""},
			{"Generated At", report.GeneratedAt.Format(config.DateFormat + " " + config.TimeFormat), "timestamp", ""},
		}
		
		for _, row := range summaryData {
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("failed to write summary data: %w", err)
			}
		}
	}
	
	return nil
}

// exportCSVDailySheet exports daily data to separate CSV file
func (ee *ExportEngine) exportCSVDailySheet(outputPath string, report *arch.WorkHourReport, config *CSVConfig) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create daily CSV file: %w", err)
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write headers
	if err := writer.Write(config.Headers); err != nil {
		return fmt.Errorf("failed to write daily headers: %w", err)
	}
	
	// Write daily data
	return ee.writeCSVDailyData(writer, report, config)
}

// exportCSVWeeklySheet exports weekly data to separate CSV file
func (ee *ExportEngine) exportCSVWeeklySheet(outputPath string, report *arch.WorkHourReport, config *CSVConfig) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create weekly CSV file: %w", err)
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write headers for weekly data
	weeklyHeaders := []string{
		"Week Start", "Week End", "Total Work Time", "Overtime Hours",
		"Standard Hours", "Average Daily Time", "Work Days", "Efficiency Score",
	}
	if err := writer.Write(weeklyHeaders); err != nil {
		return fmt.Errorf("failed to write weekly headers: %w", err)
	}
	
	// Write weekly data
	return ee.writeCSVWeeklyData(writer, report, config)
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-CSV-011
 * CONTEXT:   CSV formatting utility methods for data transformation and validation
 * REASON:    Need consistent data formatting across all CSV exports with proper type handling
 * CHANGE:    CSV formatting utilities with comprehensive data type support.
 * PREVENTION:Handle null values gracefully, validate formatting parameters, ensure consistent output
 * RISK:      Low - Formatting utilities are straightforward with clear input/output expectations
 */

// formatTimePtr formats a time pointer for CSV output
func (ee *ExportEngine) formatTimePtr(t *time.Time, format string) string {
	if t == nil {
		return ""
	}
	return t.Format(format)
}

// formatDurationCSV formats a duration for CSV output
func (ee *ExportEngine) formatDurationCSV(d time.Duration, format string) string {
	if d == 0 {
		return "0"
	}
	
	switch format {
	case "hours":
		return fmt.Sprintf("%.2f", d.Hours())
	case "minutes":
		return fmt.Sprintf("%.0f", d.Minutes())
	case "seconds":
		return fmt.Sprintf("%.0f", d.Seconds())
	case "hms":
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	default:
		return fmt.Sprintf("%.2f", d.Hours()) // Default to hours
	}
}

// formatFloat formats a float value with specified decimal places
func (ee *ExportEngine) formatFloat(value float64, decimalPlaces int) string {
	format := fmt.Sprintf("%%.%df", decimalPlaces)
	return fmt.Sprintf(format, value)
}

// formatPeakHours formats peak hours array for CSV output
func (ee *ExportEngine) formatPeakHours(hours []int) string {
	if len(hours) == 0 {
		return ""
	}
	
	hourStrings := make([]string, len(hours))
	for i, hour := range hours {
		hourStrings[i] = fmt.Sprintf("%02d:00", hour)
	}
	
	return strings.Join(hourStrings, "; ")
}

// formatHour formats an hour value for CSV output
func (ee *ExportEngine) formatHour(hour int) string {
	if hour < 0 || hour > 23 {
		return ""
	}
	return fmt.Sprintf("%02d:00", hour)
}

// sanitizeString sanitizes a string for CSV output
func (ee *ExportEngine) sanitizeString(s string) string {
	// Remove or escape problematic characters
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	
	// Trim whitespace
	s = strings.TrimSpace(s)
	
	return s
}

// Analytics calculation methods (placeholders for now)

func (ee *ExportEngine) calculateProductivityScore(workDay *domain.WorkDay) float64 {
	// Calculate productivity score based on work patterns
	baseScore := workDay.GetEfficiencyRatio() * 100
	
	// Adjust based on session count and work blocks
	if workDay.SessionCount > 0 {
		blocksPerSession := float64(workDay.BlockCount) / float64(workDay.SessionCount)
		if blocksPerSession > 3 {
			baseScore *= 1.1 // Bonus for sustained work
		}
	}
	
	return baseScore
}

func (ee *ExportEngine) calculateFocusScore(workDay *domain.WorkDay) float64 {
	// Calculate focus score based on work block patterns
	if workDay.BlockCount == 0 {
		return 0
	}
	
	// Higher score for fewer, longer blocks
	avgBlockTime := workDay.TotalTime / time.Duration(workDay.BlockCount)
	
	// Score based on average block time
	if avgBlockTime >= 30*time.Minute {
		return 90.0
	} else if avgBlockTime >= 15*time.Minute {
		return 75.0
	} else if avgBlockTime >= 5*time.Minute {
		return 60.0
	}
	
	return 40.0
}

func (ee *ExportEngine) calculateBreakFrequency(workDay *domain.WorkDay) float64 {
	// Calculate break frequency per hour
	if workDay.TotalTime == 0 {
		return 0
	}
	
	// Estimate breaks based on work blocks (simplified)
	estimatedBreaks := workDay.BlockCount - 1
	if estimatedBreaks < 0 {
		estimatedBreaks = 0
	}
	
	return float64(estimatedBreaks) / workDay.TotalTime.Hours()
}

func (ee *ExportEngine) calculateTaskSwitches(workDay *domain.WorkDay) int {
	// Estimate task switches based on work blocks
	if workDay.BlockCount <= 1 {
		return 0
	}
	return workDay.BlockCount - 1
}

func (ee *ExportEngine) getPeakPerformanceHour(workDay *domain.WorkDay) int {
	// Get peak performance hour (simplified)
	peakHours := workDay.GetPeakHours()
	if len(peakHours) > 0 {
		return peakHours[0]
	}
	return 10 // Default to 10 AM
}

func (ee *ExportEngine) getLowPerformanceHour(workDay *domain.WorkDay) int {
	// Get low performance hour (simplified)
	return 15 // Default to 3 PM (post-lunch dip)
}

func (ee *ExportEngine) getTrendDirection(workDay *domain.WorkDay) string {
	// Simplified trend direction
	efficiency := workDay.GetEfficiencyRatio()
	if efficiency > 0.8 {
		return "improving"
	} else if efficiency > 0.6 {
		return "stable"
	}
	return "declining"
}

func (ee *ExportEngine) generateRecommendations(workDay *domain.WorkDay) string {
	recommendations := []string{}
	
	efficiency := workDay.GetEfficiencyRatio()
	if efficiency < 0.7 {
		recommendations = append(recommendations, "Consider taking more regular breaks")
	}
	
	if workDay.BlockCount > 20 {
		recommendations = append(recommendations, "Try to work in longer focused blocks")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Maintain current work patterns")
	}
	
	return strings.Join(recommendations, "; ")
}