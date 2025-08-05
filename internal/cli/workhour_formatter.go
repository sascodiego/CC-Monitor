package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-015
 * CONTEXT:   Professional work hour output formatter with rich text formatting and charts
 * REASON:    Users need beautiful, readable output for work hour data with proper visual hierarchy
 * CHANGE:    Initial implementation of work hour output formatter.
 * PREVENTION:Handle nil values gracefully and provide consistent formatting across all output types
 * RISK:      Low - Formatter only handles display formatting and doesn't affect data integrity
 */

type WorkHourFormatter struct {
	baseFormatter *OutputFormatter
}

func NewWorkHourFormatter() *WorkHourFormatter {
	return &WorkHourFormatter{
		baseFormatter: NewOutputFormatter(),
	}
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-016
 * CONTEXT:   Work day status formatting with professional layout and color coding
 * REASON:    Daily status display needs clear visual hierarchy and easy-to-scan information
 * CHANGE:    Initial implementation of work day formatting functions.
 * PREVENTION:Handle edge cases like zero durations and missing data gracefully
 * RISK:      Low - Display formatting doesn't affect business logic
 */

func (whf *WorkHourFormatter) PrintWorkDayHeader(date time.Time) {
	dayName := date.Format("Monday")
	dateStr := date.Format("January 2, 2006")
	
	fmt.Printf("\n%s Work Day Status - %s, %s\n", 
		whf.baseFormatter.Colorize("📅", "blue"),
		whf.baseFormatter.Colorize(dayName, "bold"),
		whf.baseFormatter.Colorize(dateStr, "cyan"))
	fmt.Println(strings.Repeat("═", 60))
}

func (whf *WorkHourFormatter) PrintWorkDayBasicStatus(workDay *domain.WorkDay) {
	if workDay == nil {
		whf.baseFormatter.PrintWarning("No work day data available")
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📊 Daily Summary", "bold"))
	fmt.Println(strings.Repeat("─", 20))

	// Total work time
	totalHours := workDay.TotalTime.Hours()
	var timeColor string
	if totalHours >= 8 {
		timeColor = "green"
	} else if totalHours >= 6 {
		timeColor = "yellow"
	} else {
		timeColor = "red"
	}

	fmt.Printf("Total Work Time:    %s\n", 
		whf.baseFormatter.Colorize(whf.formatDuration(workDay.TotalTime), timeColor))
	
	// Sessions
	fmt.Printf("Sessions:           %s\n", 
		whf.baseFormatter.Colorize(fmt.Sprintf("%d", workDay.SessionCount), "cyan"))
	
	// Work blocks
	fmt.Printf("Work Blocks:        %s\n", 
		whf.baseFormatter.Colorize(fmt.Sprintf("%d", workDay.WorkBlockCount), "cyan"))

	// Average block time
	if workDay.WorkBlockCount > 0 {
		avgBlockTime := workDay.TotalTime / time.Duration(workDay.WorkBlockCount)
		fmt.Printf("Avg Block Time:     %s\n", 
			whf.baseFormatter.Colorize(whf.formatDuration(avgBlockTime), "blue"))
	}

	// Productivity score (if available)
	if workDay.ProductivityScore > 0 {
		var scoreColor string
		if workDay.ProductivityScore >= 80 {
			scoreColor = "green"
		} else if workDay.ProductivityScore >= 60 {
			scoreColor = "yellow"
		} else {
			scoreColor = "red"
		}
		
		fmt.Printf("Productivity Score: %s\n", 
			whf.baseFormatter.Colorize(fmt.Sprintf("%.1f%%", workDay.ProductivityScore), scoreColor))
	}

	// Current status
	fmt.Printf("Status:             %s\n", whf.getWorkDayStatusIndicator(workDay))
}

func (whf *WorkHourFormatter) PrintWorkDayDetailedStatus(workDay *domain.WorkDay) {
	if workDay == nil {
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📋 Detailed Breakdown", "bold"))
	fmt.Println(strings.Repeat("─", 25))

	// Work periods breakdown
	if len(workDay.WorkBlocks) > 0 {
		fmt.Println("\nWork Blocks:")
		for i, block := range workDay.WorkBlocks {
			duration := block.EndTime.Sub(block.StartTime)
			status := "Completed"
			statusColor := "green"
			
			if block.EndTime.IsZero() {
				status = "Active"
				statusColor = "yellow"
				duration = time.Since(block.StartTime)
			}

			fmt.Printf("  %d. %s - %s (%s) [%s]\n",
				i+1,
				block.StartTime.Format("15:04"),
				whf.formatEndTime(block.EndTime),
				whf.formatDuration(duration),
				whf.baseFormatter.Colorize(status, statusColor))
		}
	}

	// Activity timeline (simplified)
	if workDay.FirstActivity != nil && workDay.LastActivity != nil {
		fmt.Printf("\nActivity Period:    %s to %s\n",
			workDay.FirstActivity.Format("15:04"),
			workDay.LastActivity.Format("15:04"))
		
		totalSpan := workDay.LastActivity.Sub(*workDay.FirstActivity)
		if totalSpan > 0 && workDay.TotalTime > 0 {
			efficiency := float64(workDay.TotalTime) / float64(totalSpan) * 100
			fmt.Printf("Work Efficiency:    %.1f%% (active/total time)\n", efficiency)
		}
	}
}

func (whf *WorkHourFormatter) PrintWorkDayBreakAnalysis(workDay *domain.WorkDay) {
	if workDay == nil {
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("☕ Break Analysis", "bold"))
	fmt.Println(strings.Repeat("─", 20))

	// Calculate breaks between work blocks
	if len(workDay.WorkBlocks) > 1 {
		totalBreakTime := time.Duration(0)
		longestBreak := time.Duration(0)
		breakCount := 0

		for i := 1; i < len(workDay.WorkBlocks); i++ {
			if !workDay.WorkBlocks[i-1].EndTime.IsZero() {
				breakDuration := workDay.WorkBlocks[i].StartTime.Sub(workDay.WorkBlocks[i-1].EndTime)
				if breakDuration > 5*time.Minute { // Only count breaks > 5 minutes
					totalBreakTime += breakDuration
					breakCount++
					if breakDuration > longestBreak {
						longestBreak = breakDuration
					}
				}
			}
		}

		if breakCount > 0 {
			fmt.Printf("Total Break Time:   %s\n", whf.formatDuration(totalBreakTime))
			fmt.Printf("Number of Breaks:   %d\n", breakCount)
			fmt.Printf("Average Break:      %s\n", whf.formatDuration(totalBreakTime/time.Duration(breakCount)))
			fmt.Printf("Longest Break:      %s\n", whf.formatDuration(longestBreak))
		} else {
			fmt.Println("No significant breaks detected")
		}
	} else {
		fmt.Println("Insufficient data for break analysis")
	}
}

func (whf *WorkHourFormatter) PrintWorkDayPattern(workDay *domain.WorkDay) {
	if workDay == nil {
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📈 Work Pattern", "bold"))
	fmt.Println(strings.Repeat("─", 18))

	// Peak hours analysis (simplified)
	if len(workDay.WorkBlocks) > 0 {
		hourActivity := make(map[int]time.Duration)
		
		for _, block := range workDay.WorkBlocks {
			startHour := block.StartTime.Hour()
			endHour := startHour
			if !block.EndTime.IsZero() {
				endHour = block.EndTime.Hour()
			}
			
			for hour := startHour; hour <= endHour; hour++ {
				hourActivity[hour] += time.Hour // Simplified - should calculate actual time in hour
			}
		}

		// Find peak hour
		maxActivity := time.Duration(0)
		peakHour := 0
		for hour, activity := range hourActivity {
			if activity > maxActivity {
				maxActivity = activity
				peakHour = hour
			}
		}

		if maxActivity > 0 {
			fmt.Printf("Peak Activity Hour: %s\n", 
				whf.baseFormatter.Colorize(fmt.Sprintf("%02d:00", peakHour), "yellow"))
		}

		// Work pattern type
		pattern := whf.determineWorkPattern(workDay)
		fmt.Printf("Work Pattern:       %s\n", 
			whf.baseFormatter.Colorize(pattern, "blue"))
	}
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-017
 * CONTEXT:   Work week and timesheet formatting with comprehensive layout and visual elements
 * REASON:    Weekly reports and timesheets need professional formatting for business use
 * CHANGE:    Initial implementation of work week and timesheet formatting functions.
 * PREVENTION:Handle complex data structures and provide consistent formatting for all data types
 * RISK:      Low - Formatting functions are display-only and don't affect data processing
 */

func (whf *WorkHourFormatter) PrintWorkWeekSummary(workWeek *domain.WorkWeek) {
	if workWeek == nil {
		whf.baseFormatter.PrintWarning("No work week data available")
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📅 Weekly Summary", "bold"))
	fmt.Println(strings.Repeat("─", 20))

	// Total hours
	totalHours := workWeek.TotalTime.Hours()
	var timeColor string
	if totalHours >= 40 {
		timeColor = "green"
	} else if totalHours >= 35 {
		timeColor = "yellow"
	} else {
		timeColor = "red"
	}

	fmt.Printf("Total Work Time:    %s\n", 
		whf.baseFormatter.Colorize(whf.formatDuration(workWeek.TotalTime), timeColor))

	// Standard vs actual
	if workWeek.StandardHours > 0 {
		diff := workWeek.TotalTime - workWeek.StandardHours
		diffColor := "green"
		diffSymbol := "+"
		if diff < 0 {
			diffColor = "red"
			diffSymbol = ""
		}
		
		fmt.Printf("Standard Hours:     %s (%s%s)\n", 
			whf.formatDuration(workWeek.StandardHours),
			whf.baseFormatter.Colorize(diffSymbol+whf.formatDuration(diff), diffColor))
	}

	// Overtime
	if workWeek.OvertimeHours > 0 {
		fmt.Printf("Overtime Hours:     %s\n", 
			whf.baseFormatter.Colorize(whf.formatDuration(workWeek.OvertimeHours), "yellow"))
	}

	// Work days
	fmt.Printf("Work Days:          %d of %d\n", 
		len(workWeek.WorkDays), 5) // Assuming 5-day work week

	// Average daily hours
	if len(workWeek.WorkDays) > 0 {
		avgDaily := workWeek.TotalTime / time.Duration(len(workWeek.WorkDays))
		fmt.Printf("Avg Daily Hours:    %s\n", 
			whf.baseFormatter.Colorize(whf.formatDuration(avgDaily), "blue"))
	}
}

func (whf *WorkHourFormatter) PrintOvertimeAnalysis(workWeek *domain.WorkWeek) {
	if workWeek == nil || workWeek.OvertimeHours == 0 {
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("⏰ Overtime Analysis", "bold"))
	fmt.Println(strings.Repeat("─", 22))

	fmt.Printf("Total Overtime:     %s\n", 
		whf.baseFormatter.Colorize(whf.formatDuration(workWeek.OvertimeHours), "yellow"))

	// Overtime by day
	fmt.Println("\nDaily Overtime Breakdown:")
	for _, workDay := range workWeek.WorkDays {
		if workDay.OvertimeHours > 0 {
			fmt.Printf("  %s: %s\n", 
				workDay.Date.Format("Monday"),
				whf.formatDuration(workDay.OvertimeHours))
		}
	}
}

func (whf *WorkHourFormatter) PrintWeeklyDailyBreakdown(workWeek *domain.WorkWeek) {
	if workWeek == nil {
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📊 Daily Breakdown", "bold"))
	fmt.Println(strings.Repeat("─", 21))

	// Table header
	fmt.Printf("%-10s | %-12s | %-8s | %-8s\n", "Day", "Work Time", "Sessions", "Blocks")
	fmt.Println(strings.Repeat("─", 50))

	// Daily rows
	for _, workDay := range workWeek.WorkDays {
		dayName := workDay.Date.Format("Monday")
		workTime := whf.formatDuration(workDay.TotalTime)
		
		var timeColor string
		hours := workDay.TotalTime.Hours()
		if hours >= 8 {
			timeColor = "green"
		} else if hours >= 6 {
			timeColor = "yellow"
		} else if hours > 0 {
			timeColor = "red"
		} else {
			timeColor = "gray"
		}

		fmt.Printf("%-10s | %s | %-8d | %-8d\n",
			dayName,
			whf.baseFormatter.Colorize(fmt.Sprintf("%-12s", workTime), timeColor),
			workDay.SessionCount,
			workDay.WorkBlockCount)
	}
}

func (whf *WorkHourFormatter) PrintTimesheetHeader(timesheet *domain.Timesheet) {
	if timesheet == nil {
		return
	}

	fmt.Printf("\n%s Timesheet: %s\n", 
		whf.baseFormatter.Colorize("📄", "blue"),
		whf.baseFormatter.Colorize(timesheet.ID, "bold"))
	fmt.Println(strings.Repeat("═", 50))

	fmt.Printf("Employee:     %s\n", timesheet.EmployeeID)
	fmt.Printf("Period:       %s\n", timesheet.Period)
	fmt.Printf("Start Date:   %s\n", timesheet.StartDate.Format("2006-01-02"))
	fmt.Printf("End Date:     %s\n", timesheet.EndDate.Format("2006-01-02"))
	
	statusColor := "yellow"
	if timesheet.Status == domain.TimesheetStatusApproved {
		statusColor = "green"
	} else if timesheet.Status == domain.TimesheetStatusRejected {
		statusColor = "red"
	}
	
	fmt.Printf("Status:       %s\n", 
		whf.baseFormatter.Colorize(string(timesheet.Status), statusColor))
}

func (whf *WorkHourFormatter) PrintTimesheetEntries(timesheet *domain.Timesheet) {
	if timesheet == nil || len(timesheet.Entries) == 0 {
		fmt.Println("\nNo timesheet entries found")
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📋 Time Entries", "bold"))
	fmt.Println(strings.Repeat("─", 18))

	// Table header
	fmt.Printf("%-12s | %-10s | %-10s | %-12s | %-10s\n", 
		"Date", "Start", "End", "Duration", "Overtime")
	fmt.Println(strings.Repeat("─", 70))

	// Entry rows
	for _, entry := range timesheet.Entries {
		overtimeStr := "-"
		if entry.OvertimeHours > 0 {
			overtimeStr = whf.formatDuration(entry.OvertimeHours)
		}

		fmt.Printf("%-12s | %-10s | %-10s | %-12s | %-10s\n",
			entry.Date.Format("2006-01-02"),
			entry.StartTime.Format("15:04"),
			entry.EndTime.Format("15:04"),
			whf.formatDuration(entry.Duration),
			overtimeStr)
	}
}

func (whf *WorkHourFormatter) PrintTimesheetSummary(timesheet *domain.Timesheet) {
	if timesheet == nil {
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📊 Timesheet Summary", "bold"))
	fmt.Println(strings.Repeat("─", 23))

	fmt.Printf("Total Hours:        %s\n", 
		whf.baseFormatter.Colorize(whf.formatDuration(timesheet.TotalHours), "green"))
	
	if timesheet.OvertimeHours > 0 {
		fmt.Printf("Overtime Hours:     %s\n", 
			whf.baseFormatter.Colorize(whf.formatDuration(timesheet.OvertimeHours), "yellow"))
	}

	fmt.Printf("Regular Hours:      %s\n", 
		whf.formatDuration(timesheet.TotalHours-timesheet.OvertimeHours))

	if timesheet.CreatedAt != nil {
		fmt.Printf("Generated:          %s\n", 
			timesheet.CreatedAt.Format("2006-01-02 15:04"))
	}
	
	if timesheet.SubmittedAt != nil {
		fmt.Printf("Submitted:          %s\n", 
			timesheet.SubmittedAt.Format("2006-01-02 15:04"))
	}
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-018
 * CONTEXT:   Analytics and insights formatting with visual elements and recommendations
 * REASON:    Analytics output needs clear visual presentation of metrics and actionable insights
 * CHANGE:    Initial implementation of analytics formatting functions.
 * PREVENTION:Handle missing or incomplete analytics data gracefully with fallback displays
 * RISK:      Low - Analytics formatting is display-only and doesn't affect calculation accuracy
 */

func (whf *WorkHourFormatter) PrintAnalysisHeader(title, period string) {
	fmt.Printf("\n%s %s\n", 
		whf.baseFormatter.Colorize("📊", "blue"),
		whf.baseFormatter.Colorize(title, "bold"))
	
	if period != "" {
		fmt.Printf("Period: %s\n", whf.baseFormatter.Colorize(period, "cyan"))
	}
	
	fmt.Println(strings.Repeat("═", 50))
}

func (whf *WorkHourFormatter) PrintEfficiencyMetrics(metrics *domain.EfficiencyMetrics) {
	if metrics == nil {
		whf.baseFormatter.PrintWarning("No efficiency metrics available")
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("⚡ Efficiency Metrics", "bold"))
	fmt.Println(strings.Repeat("─", 25))

	// Active ratio
	activeColor := whf.getPercentageColor(metrics.ActiveRatio)
	fmt.Printf("Active Time Ratio:  %s\n", 
		whf.baseFormatter.Colorize(fmt.Sprintf("%.1f%%", metrics.ActiveRatio), activeColor))

	// Focus score
	focusColor := whf.getPercentageColor(metrics.FocusScore)
	fmt.Printf("Focus Score:        %s\n", 
		whf.baseFormatter.Colorize(fmt.Sprintf("%.1f%%", metrics.FocusScore), focusColor))

	// Productivity score
	if metrics.ProductivityScore > 0 {
		productivityColor := whf.getPercentageColor(metrics.ProductivityScore)
		fmt.Printf("Productivity Score: %s\n", 
			whf.baseFormatter.Colorize(fmt.Sprintf("%.1f%%", metrics.ProductivityScore), productivityColor))
	}

	// Session efficiency
	if metrics.SessionEfficiency > 0 {
		sessionColor := whf.getPercentageColor(metrics.SessionEfficiency)
		fmt.Printf("Session Efficiency: %s\n", 
			whf.baseFormatter.Colorize(fmt.Sprintf("%.1f%%", metrics.SessionEfficiency), sessionColor))
	}

	// Break frequency
	if metrics.BreakFrequency > 0 {
		fmt.Printf("Break Frequency:    %.1f breaks/hour\n", metrics.BreakFrequency)
	}
}

func (whf *WorkHourFormatter) PrintWorkPattern(pattern *domain.WorkPattern) {
	if pattern == nil {
		whf.baseFormatter.PrintWarning("No work pattern data available")
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📈 Work Pattern Analysis", "bold"))
	fmt.Println(strings.Repeat("─", 28))

	fmt.Printf("Work Day Type:      %s\n", 
		whf.baseFormatter.Colorize(string(pattern.WorkDayType), "cyan"))

	// Peak hours
	if len(pattern.PeakHours) > 0 {
		peakHoursStr := ""
		for i, hour := range pattern.PeakHours {
			if i > 0 {
				peakHoursStr += ", "
			}
			peakHoursStr += fmt.Sprintf("%02d:00", hour)
		}
		fmt.Printf("Peak Hours:         %s\n", 
			whf.baseFormatter.Colorize(peakHoursStr, "yellow"))
	}

	// Average session length
	if pattern.AvgSessionLength > 0 {
		fmt.Printf("Avg Session Length: %s\n", 
			whf.formatDuration(pattern.AvgSessionLength))
	}

	// Consistency score
	if pattern.ConsistencyScore > 0 {
		consistencyColor := whf.getPercentageColor(pattern.ConsistencyScore)
		fmt.Printf("Consistency Score:  %s\n", 
			whf.baseFormatter.Colorize(fmt.Sprintf("%.1f%%", pattern.ConsistencyScore), consistencyColor))
	}
}

func (whf *WorkHourFormatter) PrintTrendAnalysis(trends *domain.TrendAnalysis) {
	if trends == nil {
		whf.baseFormatter.PrintWarning("No trend analysis data available")
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📈 Trend Analysis", "bold"))
	fmt.Println(strings.Repeat("─", 20))

	// Trend direction
	var trendIcon, trendColor string
	switch trends.TrendDirection {
	case domain.TrendUp:
		trendIcon = "📈"
		trendColor = "green"
	case domain.TrendDown:
		trendIcon = "📉"
		trendColor = "red"
	case domain.TrendStable:
		trendIcon = "➡️"
		trendColor = "yellow"
	default:
		trendIcon = "❓"
		trendColor = "gray"
	}

	fmt.Printf("Trend Direction:    %s %s\n", 
		trendIcon,
		whf.baseFormatter.Colorize(string(trends.TrendDirection), trendColor))

	// Work time change
	if trends.WorkTimeChange != 0 {
		changeSign := "+"
		changeColor := "green"
		if trends.WorkTimeChange < 0 {
			changeSign = ""
			changeColor = "red"
		}
		
		fmt.Printf("Work Time Change:   %s\n", 
			whf.baseFormatter.Colorize(changeSign+whf.formatDuration(trends.WorkTimeChange), changeColor))
	}

	// Productivity change
	if trends.ProductivityChange != 0 {
		changeSign := "+"
		changeColor := "green"
		if trends.ProductivityChange < 0 {
			changeSign = ""
			changeColor = "red"
		}
		
		fmt.Printf("Productivity Change: %s\n", 
			whf.baseFormatter.Colorize(fmt.Sprintf("%s%.1f%%", changeSign, trends.ProductivityChange), changeColor))
	}

	// Confidence level
	if trends.ConfidenceLevel > 0 {
		confColor := whf.getPercentageColor(trends.ConfidenceLevel)
		fmt.Printf("Confidence Level:   %s\n", 
			whf.baseFormatter.Colorize(fmt.Sprintf("%.1f%%", trends.ConfidenceLevel), confColor))
	}
}

func (whf *WorkHourFormatter) PrintReportHeader(report *arch.WorkHourReport) {
	if report == nil {
		return
	}

	fmt.Printf("\n%s %s\n", 
		whf.baseFormatter.Colorize("📊", "blue"),
		whf.baseFormatter.Colorize(report.Title, "bold"))
	
	fmt.Printf("Report ID:    %s\n", report.ID)
	fmt.Printf("Period:       %s\n", report.Period)
	fmt.Printf("Generated:    %s\n", report.GeneratedAt.Format("2006-01-02 15:04"))
	
	fmt.Println(strings.Repeat("═", 60))
}

func (whf *WorkHourFormatter) PrintActivitySummary(summary *domain.ActivitySummary) {
	if summary == nil {
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("📋 Activity Summary", "bold"))
	fmt.Println(strings.Repeat("─", 22))

	fmt.Printf("Total Activities:   %d\n", summary.TotalActivities)
	fmt.Printf("Unique Sessions:    %d\n", summary.UniqueSessions)
	fmt.Printf("Average per Day:    %.1f\n", summary.AvgActivitiesPerDay)
	
	if summary.PeakActivityHour > 0 {
		fmt.Printf("Peak Hour:          %02d:00\n", summary.PeakActivityHour)
	}
}

func (whf *WorkHourFormatter) PrintGoalProgress(report *arch.WorkHourReport) {
	fmt.Println("\n" + whf.baseFormatter.Colorize("🎯 Goal Progress", "bold"))
	fmt.Println(strings.Repeat("─", 18))
	
	// This would display actual goal progress if implemented
	whf.baseFormatter.PrintWarning("Goal progress tracking not yet implemented")
}

func (whf *WorkHourFormatter) PrintWorkDayCharts(report *arch.WorkHourReport) {
	fmt.Println("\n" + whf.baseFormatter.Colorize("📊 Visual Charts", "bold"))
	fmt.Println(strings.Repeat("─", 20))
	
	// ASCII charts would be implemented here
	whf.baseFormatter.PrintWarning("ASCII charts not yet implemented")
}

func (whf *WorkHourFormatter) PrintPolicyConfiguration(config *arch.WorkHourConfiguration) {
	if config == nil {
		whf.baseFormatter.PrintWarning("No configuration available")
		return
	}

	fmt.Println("\n" + whf.baseFormatter.Colorize("⚙️ Work Hour Policies", "bold"))
	fmt.Println(strings.Repeat("─", 25))

	fmt.Printf("Rounding Interval:    %s\n", whf.formatDuration(config.DefaultPolicy.RoundingInterval))
	fmt.Printf("Rounding Method:      %s\n", config.DefaultPolicy.RoundingMethod)
	fmt.Printf("Overtime Threshold:   %s\n", whf.formatDuration(config.DefaultPolicy.OvertimeThreshold))
	fmt.Printf("Weekly Threshold:     %s\n", whf.formatDuration(config.DefaultPolicy.WeeklyThreshold))
	fmt.Printf("Break Deduction:      %s\n", whf.formatDuration(config.DefaultPolicy.BreakDeduction))
	fmt.Printf("Standard Work Hours:  %s\n", whf.formatDuration(config.StandardWorkHours))
	fmt.Printf("Work Week Start:      %s\n", config.WorkWeekStart)
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-019
 * CONTEXT:   Recommendation and insight formatting for actionable user guidance
 * REASON:    Users need clear, actionable recommendations based on their work patterns and analytics
 * CHANGE:    Initial implementation of recommendation formatting functions.
 * PREVENTION:Provide meaningful recommendations even with limited data, avoid generic advice
 * RISK:      Low - Recommendations are advisory and don't affect system operation
 */

func (whf *WorkHourFormatter) PrintRecommendations(pattern *domain.WorkPattern, efficiency *domain.EfficiencyMetrics, trends *domain.TrendAnalysis) {
	fmt.Println("\n" + whf.baseFormatter.Colorize("💡 Recommendations", "bold"))
	fmt.Println(strings.Repeat("─", 22))

	recommendations := whf.generateRecommendations(pattern, efficiency, trends)
	
	if len(recommendations) == 0 {
		fmt.Println("No specific recommendations available with current data")
		return
	}

	for i, rec := range recommendations {
		fmt.Printf("%d. %s\n", i+1, rec)
	}
}

func (whf *WorkHourFormatter) PrintProductivityRecommendations(metrics *domain.EfficiencyMetrics) {
	fmt.Println("\n" + whf.baseFormatter.Colorize("💡 Productivity Recommendations", "bold"))
	fmt.Println(strings.Repeat("─", 32))

	recommendations := whf.generateProductivityRecommendations(metrics)
	
	for i, rec := range recommendations {
		fmt.Printf("%d. %s\n", i+1, rec)
	}
}

func (whf *WorkHourFormatter) PrintPatternRecommendations(pattern *domain.WorkPattern) {
	fmt.Println("\n" + whf.baseFormatter.Colorize("💡 Pattern Recommendations", "bold"))
	fmt.Println(strings.Repeat("─", 28))

	recommendations := whf.generatePatternRecommendations(pattern)
	
	for i, rec := range recommendations {
		fmt.Printf("%d. %s\n", i+1, rec)
	}
}

func (whf *WorkHourFormatter) PrintWorkDayDetailedReport(workDay *domain.WorkDay) {
	if workDay == nil {
		return
	}

	whf.PrintWorkDayHeader(workDay.Date)
	whf.PrintWorkDayBasicStatus(workDay)
	whf.PrintWorkDayDetailedStatus(workDay)
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-020
 * CONTEXT:   Helper functions for formatting, color coding, and data presentation
 * REASON:    Need consistent helper functions for professional data presentation and user experience
 * CHANGE:    Initial implementation of work hour formatter helper functions.
 * PREVENTION:Handle edge cases in formatting and provide consistent color schemes
 * RISK:      Low - Helper functions support main formatting but don't affect core functionality
 */

// Helper functions for formatting and presentation
func (whf *WorkHourFormatter) formatDuration(d time.Duration) string {
	if d == 0 {
		return "0h 0m"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	} else if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	} else {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

func (whf *WorkHourFormatter) formatEndTime(t time.Time) string {
	if t.IsZero() {
		return "now"
	}
	return t.Format("15:04")
}

func (whf *WorkHourFormatter) getPercentageColor(percentage float64) string {
	if percentage >= 80 {
		return "green"
	} else if percentage >= 60 {
		return "yellow"
	} else {
		return "red"
	}
}

func (whf *WorkHourFormatter) getWorkDayStatusIndicator(workDay *domain.WorkDay) string {
	if workDay == nil {
		return whf.baseFormatter.Colorize("Unknown", "gray")
	}

	hours := workDay.TotalTime.Hours()
	if hours >= 8 {
		return whf.baseFormatter.Colorize("✓ Full Day", "green")
	} else if hours >= 6 {
		return whf.baseFormatter.Colorize("~ Partial Day", "yellow")
	} else if hours > 0 {
		return whf.baseFormatter.Colorize("⚠ Short Day", "red")
	} else {
		return whf.baseFormatter.Colorize("○ No Work", "gray")
	}
}

func (whf *WorkHourFormatter) determineWorkPattern(workDay *domain.WorkDay) string {
	if workDay == nil || len(workDay.WorkBlocks) == 0 {
		return "Unknown"
	}

	blockCount := len(workDay.WorkBlocks)
	if blockCount == 1 {
		return "Concentrated"
	} else if blockCount <= 3 {
		return "Focused"
	} else if blockCount <= 6 {
		return "Fragmented"
	} else {
		return "Highly Fragmented"
	}
}

func (whf *WorkHourFormatter) generateRecommendations(pattern *domain.WorkPattern, efficiency *domain.EfficiencyMetrics, trends *domain.TrendAnalysis) []string {
	recommendations := []string{}

	// Efficiency-based recommendations
	if efficiency != nil {
		if efficiency.FocusScore < 60 {
			recommendations = append(recommendations, "Consider reducing distractions to improve focus score")
		}
		if efficiency.ActiveRatio < 70 {
			recommendations = append(recommendations, "Try to maintain more consistent work periods")
		}
	}

	// Pattern-based recommendations
	if pattern != nil {
		if pattern.ConsistencyScore < 50 {
			recommendations = append(recommendations, "Establish more consistent daily work patterns")
		}
	}

	// Trend-based recommendations
	if trends != nil {
		if trends.TrendDirection == domain.TrendDown {
			recommendations = append(recommendations, "Consider reviewing recent changes that may have affected productivity")
		}
	}

	return recommendations
}

func (whf *WorkHourFormatter) generateProductivityRecommendations(metrics *domain.EfficiencyMetrics) []string {
	if metrics == nil {
		return []string{"Insufficient data for productivity recommendations"}
	}

	recommendations := []string{}

	if metrics.FocusScore < 70 {
		recommendations = append(recommendations, "Try time-blocking techniques to improve focus")
		recommendations = append(recommendations, "Consider using a distraction-blocking app during work hours")
	}

	if metrics.ActiveRatio < 75 {
		recommendations = append(recommendations, "Take shorter, more frequent breaks to maintain energy")
		recommendations = append(recommendations, "Review your work environment for potential improvements")
	}

	if metrics.SessionEfficiency < 60 {
		recommendations = append(recommendations, "Plan your work sessions with clear objectives")
		recommendations = append(recommendations, "Consider using the Pomodoro Technique for better session management")
	}

	return recommendations
}

func (whf *WorkHourFormatter) generatePatternRecommendations(pattern *domain.WorkPattern) []string {
	if pattern == nil {
		return []string{"Insufficient data for pattern recommendations"}
	}

	recommendations := []string{}

	if pattern.ConsistencyScore < 50 {
		recommendations = append(recommendations, "Try to start work at the same time each day")
		recommendations = append(recommendations, "Establish a regular work routine")
	}

	if len(pattern.PeakHours) > 0 {
		peakHour := pattern.PeakHours[0]
		recommendations = append(recommendations, 
			fmt.Sprintf("Schedule your most important tasks around %02d:00 (your peak hour)", peakHour))
	}

	if pattern.AvgSessionLength < 30*time.Minute {
		recommendations = append(recommendations, "Try to work in longer, uninterrupted blocks for better productivity")
	}

	return recommendations
}