/**
 * CONTEXT:   Main display functions for Claude Monitor professional reports
 * INPUT:     Report data structures requiring comprehensive visual presentation
 * OUTPUT:    Complete professional CLI interface with coordinated sections
 * BUSINESS:  Main display orchestration provides cohesive user experience
 * CHANGE:    Refactored from monolithic file to focus on display coordination
 * RISK:      Low - Display orchestration with delegated formatting responsibilities
 */

package reporting

import (
	"fmt"
	"strings"
	"time"
)

// Data structures for display components

/**
 * CONTEXT:   Project data structure for display formatting
 * INPUT:     Project information with time allocation and activity metrics
 * OUTPUT:    Structured data for consistent project display
 * BUSINESS:  Standardized project data enables consistent reporting
 * CHANGE:    Kept as part of main display interface
 * RISK:      Low - Data structure for display coordination
 */
type ProjectData struct {
	Name     string
	Duration time.Duration
	Percent  float64
	Sessions int
	Color    string
}

/**
 * CONTEXT:   Work block data structure for timeline display
 * INPUT:     Individual work session information with timing and project context
 * OUTPUT:    Structured data for timeline visualization
 * BUSINESS:  Detailed work block data enables timeline analysis
 * CHANGE:    Kept as part of main display interface
 * RISK:      Low - Data structure for timeline coordination
 */
type WorkBlockData struct {
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	ProjectName   string
	ActivityCount int
}

/**
 * CONTEXT:   Professional header for Claude Monitor reports
 * INPUT:     Report title and date information
 * OUTPUT:    Elegant bordered header with consistent branding
 * BUSINESS:  Professional appearance builds user confidence
 * CHANGE:    Kept as main display function using extracted constants
 * RISK:      Low - Visual enhancement only
 */
func DisplayProfessionalHeader(title, date string) {
	headerWidth := DefaultHeaderWidth
	titleLine := fmt.Sprintf("    %s CLAUDE MONITOR %s %s", SymbolWork, SymbolWork, title)
	dateLine := fmt.Sprintf("        %s", date)
	
	// Top border
	fmt.Printf("%s%s", ColorBrightCyan, BoxTopLeft)
	fmt.Print(strings.Repeat(BoxHorizontal, headerWidth-2))
	fmt.Printf("%s%s\n", BoxTopRight, ColorReset)
	
	// Title line
	fmt.Printf("%s%s%s%-*s%s%s\n", 
		ColorBrightCyan, BoxVertical, ColorBold, headerWidth-2, titleLine, ColorReset, ColorBrightCyan + BoxVertical + ColorReset)
	
	// Separator
	fmt.Printf("%s%s", ColorBrightCyan, BoxTeeRight)
	fmt.Print(strings.Repeat(BoxHorizontal, headerWidth-2))
	fmt.Printf("%s%s\n", BoxTeeLeft, ColorReset)
	
	// Date line
	fmt.Printf("%s%s%s%-*s%s%s\n", 
		ColorBrightCyan, BoxVertical, ColorDim, headerWidth-2, dateLine, ColorReset, ColorBrightCyan + BoxVertical + ColorReset)
	
	// Bottom border
	fmt.Printf("%s%s", ColorBrightCyan, BoxBottomLeft)
	fmt.Print(strings.Repeat(BoxHorizontal, headerWidth-2))
	fmt.Printf("%s%s\n\n", BoxBottomRight, ColorReset)
}

/**
 * CONTEXT:   Professional metrics dashboard for key statistics
 * INPUT:     Daily metrics including work time, efficiency, sessions
 * OUTPUT:    Dashboard-style metrics display with contextual coloring
 * BUSINESS:  Quick visual overview of key productivity metrics
 * CHANGE:    Dashboard design using extracted utility functions
 * RISK:      Low - Metrics display enhancement
 */
func DisplayMetricsDashboard(activeWork time.Duration, totalTime time.Duration, sessions int, efficiency float64, claudeTime time.Duration) {
	sectionWidth := DefaultSectionWidth
	claudePercent := 0.0
	if activeWork > 0 {
		claudePercent = float64(claudeTime) / float64(activeWork) * 100
	}
	
	// Section header
	fmt.Printf("%s%s%s %s TODAY'S METRICS %s", 
		ColorBrightBlue, BoxTopLeft, BoxHorizontal, SymbolSession, strings.Repeat(BoxHorizontal, sectionWidth-20))
	fmt.Printf("%s%s\n", BoxTopRight, ColorReset)
	
	// Metrics line 1
	activeWorkStr := formatDurationPro(activeWork)
	efficiencyStr := fmt.Sprintf("%.1f%%", efficiency)
	efficiencyColor := getEfficiencyColor(efficiency)
	
	line1 := fmt.Sprintf("  %s Active Work: %s%s%s     %s Efficiency: %s%s%s", 
		SymbolWork, ColorBrightGreen, activeWorkStr, ColorReset,
		SymbolEfficiency, efficiencyColor, efficiencyStr, ColorReset)
	
	fmt.Printf("%s%s%-*s%s%s\n", 
		ColorBrightBlue, BoxVertical, sectionWidth, line1, BoxVertical, ColorReset)
	
	// Metrics line 2
	sessionsStr := fmt.Sprintf("%d sessions", sessions)
	focusScore := calculateFocusScore(efficiency, sessions)
	focusColor := getFocusColor(focusScore)
	
	line2 := fmt.Sprintf("  %s Sessions: %s%s%s           %s Focus: %s%d/100%s", 
		SymbolTimeline, ColorBrightCyan, sessionsStr, ColorReset,
		SymbolFocus, focusColor, focusScore, ColorReset)
	
	fmt.Printf("%s%s%-*s%s%s\n", 
		ColorBrightBlue, BoxVertical, sectionWidth, line2, BoxVertical, ColorReset)
	
	// Claude processing line
	if claudeTime > 0 {
		claudeStr := fmt.Sprintf("%s (%.1f%%)", formatDurationPro(claudeTime), claudePercent)
		line3 := fmt.Sprintf("  %s Claude Processing: %s%s%s", 
			SymbolClaude, ColorBrightMagenta, claudeStr, ColorReset)
		
		fmt.Printf("%s%s%-*s%s%s\n", 
			ColorBrightBlue, BoxVertical, sectionWidth, line3, BoxVertical, ColorReset)
	}
	
	// Bottom border
	fmt.Printf("%s%s", ColorBrightBlue, BoxBottomLeft)
	fmt.Print(strings.Repeat(BoxHorizontal, sectionWidth))
	fmt.Printf("%s%s\n\n", BoxBottomRight, ColorReset)
}

/**
 * CONTEXT:   Professional project breakdown table
 * INPUT:     Project data with time allocation and activity counts
 * OUTPUT:    Modern table with visual progress indicators
 * BUSINESS:  Clear project time allocation for productivity analysis
 * CHANGE:    Table design using extracted utility functions for formatting
 * RISK:      Low - Project display enhancement
 */
func DisplayProfessionalProjectBreakdown(projects []ProjectData) {
	if len(projects) == 0 {
		return
	}
	
	sectionWidth := DefaultSectionWidth
	
	// Section header
	fmt.Printf("%s%s%s %s PROJECT BREAKDOWN %s", 
		ColorBrightGreen, BoxTopLeft, BoxHorizontal, SymbolProject, strings.Repeat(BoxHorizontal, sectionWidth-22))
	fmt.Printf("%s%s\n", BoxTopRight, ColorReset)
	
	// Table header
	headerLine := fmt.Sprintf(" %-25s │ %-8s │ %-5s │ Sessions ", "Project", "Time", "%")
	fmt.Printf("%s%s%s%s%s\n", 
		ColorBrightGreen, BoxVertical, ColorBold, headerLine, ColorReset + ColorBrightGreen + BoxVertical + ColorReset)
	
	// Header separator
	fmt.Printf("%s%s", ColorBrightGreen, BoxTeeRight)
	fmt.Print(strings.Repeat(BoxHorizontal, 25) + BoxTeeDown + strings.Repeat(BoxHorizontal, 10) + BoxTeeDown + strings.Repeat(BoxHorizontal, 7) + BoxTeeDown + strings.Repeat(BoxHorizontal, 10))
	fmt.Printf("%s%s\n", BoxTeeLeft, ColorReset)
	
	// Project rows
	for i, project := range projects {
		projectName := truncateStringPro(project.Name, MaxProjectNameLen)
		timeStr := formatDurationPro(project.Duration)
		percentStr := fmt.Sprintf("%.1f%%", project.Percent)
		sessionsStr := fmt.Sprintf("%d", project.Sessions)
		
		// Color based on time allocation
		projectColor := getProjectColor(project.Percent)
		
		row := fmt.Sprintf(" %s%-25s%s │ %-8s │ %-5s │ %-8s ", 
			projectColor, projectName, ColorReset, timeStr, percentStr, sessionsStr)
		
		fmt.Printf("%s%s%s%s%s\n", 
			ColorBrightGreen, BoxVertical, row, BoxVertical, ColorReset)
		
		// Row separator (except last)
		if i < len(projects)-1 {
			fmt.Printf("%s%s", ColorBrightGreen, BoxTeeRight)
			fmt.Print(strings.Repeat(BoxHorizontal, 25) + BoxCross + strings.Repeat(BoxHorizontal, 10) + BoxCross + strings.Repeat(BoxHorizontal, 7) + BoxCross + strings.Repeat(BoxHorizontal, 10))
			fmt.Printf("%s%s\n", BoxTeeLeft, ColorReset)
		}
	}
	
	// Bottom border
	fmt.Printf("%s%s", ColorBrightGreen, BoxBottomLeft)
	fmt.Print(strings.Repeat(BoxHorizontal, 25) + BoxTeeUp + strings.Repeat(BoxHorizontal, 10) + BoxTeeUp + strings.Repeat(BoxHorizontal, 7) + BoxTeeUp + strings.Repeat(BoxHorizontal, 10))
	fmt.Printf("%s%s\n\n", BoxBottomRight, ColorReset)
}

/**
 * CONTEXT:   Professional work timeline with visual connectors
 * INPUT:     Work block data with time ranges and activities
 * OUTPUT:    Timeline visualization with professional styling
 * BUSINESS:  Visual work timeline helps identify productivity patterns
 * CHANGE:    Timeline with extracted utility functions for formatting
 * RISK:      Low - Timeline display enhancement
 */
func DisplayProfessionalWorkTimeline(workBlocks []WorkBlockData) {
	if len(workBlocks) == 0 {
		return
	}
	
	sectionWidth := DefaultSectionWidth
	
	// Section header
	fmt.Printf("%s%s%s %s WORK TIMELINE %s", 
		ColorBrightYellow, BoxTopLeft, BoxHorizontal, SymbolTimeline, strings.Repeat(BoxHorizontal, sectionWidth-18))
	fmt.Printf("%s%s\n", BoxTopRight, ColorReset)
	
	// Timeline entries
	for i, block := range workBlocks {
		timeRange := fmt.Sprintf("%s-%s", 
			block.StartTime.Format("15:04"), 
			block.EndTime.Format("15:04"))
		
		durationStr := formatDurationPro(block.Duration)
		projectName := truncateStringPro(block.ProjectName, 15)
		activityStr := fmt.Sprintf("%d activities", block.ActivityCount)
		
		// Visual connector
		connector := "├──"
		if i == len(workBlocks)-1 {
			connector = "└──"
		}
		
		// Color based on duration
		durationColor := getDurationColor(block.Duration)
		
		timeline := fmt.Sprintf(" %s%s%s %s %s%-8s%s %-15s %s%s%s", 
			ColorDim, connector, ColorReset,
			timeRange, durationColor, durationStr, ColorReset,
			projectName, ColorDim, activityStr, ColorReset)
		
		fmt.Printf("%s%s%-*s%s%s\n", 
			ColorBrightYellow, BoxVertical, sectionWidth, timeline, BoxVertical, ColorReset)
	}
	
	// Bottom border
	fmt.Printf("%s%s", ColorBrightYellow, BoxBottomLeft)
	fmt.Print(strings.Repeat(BoxHorizontal, sectionWidth))
	fmt.Printf("%s%s\n\n", BoxBottomRight, ColorReset)
}

/**
 * CONTEXT:   Professional insights display with enhanced formatting
 * INPUT:     Generated insights and recommendations
 * OUTPUT:    Visually appealing insights with proper formatting
 * BUSINESS:  Professional insights encourage user engagement
 * CHANGE:    Insights formatting using extracted text wrapping utility
 * RISK:      Low - Insights display improvement
 */
func DisplayProfessionalInsights(insights []string) {
	if len(insights) == 0 {
		return
	}
	
	sectionWidth := DefaultSectionWidth
	
	// Section header
	fmt.Printf("%s%s%s %s INSIGHTS & RECOMMENDATIONS %s", 
		ColorBrightMagenta, BoxTopLeft, BoxHorizontal, SymbolInsight, strings.Repeat(BoxHorizontal, sectionWidth-28))
	fmt.Printf("%s%s\n", BoxTopRight, ColorReset)
	
	// Display insights
	for _, insight := range insights {
		wrappedInsight := wrapText(insight, sectionWidth-4)
		lines := strings.Split(wrappedInsight, "\n")
		
		for j, line := range lines {
			if j == 0 {
				fmt.Printf("%s%s %s• %s%s%-*s%s%s\n", 
					ColorBrightMagenta, BoxVertical, ColorBrightWhite, ColorReset, 
					line, sectionWidth-len(line)-3, "", BoxVertical, ColorReset)
			} else {
				fmt.Printf("%s%s   %-*s%s%s\n", 
					ColorBrightMagenta, BoxVertical, sectionWidth-3, line, BoxVertical, ColorReset)
			}
		}
	}
	
	// Bottom border
	fmt.Printf("%s%s", ColorBrightMagenta, BoxBottomLeft)
	fmt.Print(strings.Repeat(BoxHorizontal, sectionWidth))
	fmt.Printf("%s%s\n\n", BoxBottomRight, ColorReset)
}

/**
 * CONTEXT:   Professional footer with navigation suggestions
 * INPUT:     Available commands and next steps
 * OUTPUT:    Actionable footer with command suggestions
 * BUSINESS:  Footer guidance improves user experience and tool adoption
 * CHANGE:    Footer with extracted constants for consistent formatting
 * RISK:      Low - Footer enhancement
 */
func DisplayProfessionalFooter() {
	fmt.Printf("%s%s %s NEXT STEPS%s\n", 
		ColorDim, SymbolTrend, "Quick Commands", ColorReset)
	
	fmt.Printf("%s• %sWeekly overview:%s claude-monitor week\n", ColorDim, ColorCyan, ColorReset)
	fmt.Printf("%s• %sMonthly analysis:%s claude-monitor month\n", ColorDim, ColorCyan, ColorReset)  
	fmt.Printf("%s• %sProject deep dive:%s claude-monitor project --name=\"ProjectName\"\n", ColorDim, ColorCyan, ColorReset)
	fmt.Printf("%s• %sSystem status:%s claude-monitor status\n\n", ColorDim, ColorCyan, ColorReset)
}

/**
 * CONTEXT:   Professional empty state display
 * INPUT:     Empty state context and guidance
 * OUTPUT:    Professional empty state with clear next steps
 * BUSINESS:  Professional empty states improve user onboarding
 * CHANGE:    Empty state using extracted text wrapping utility
 * RISK:      Low - Empty state improvement
 */
func DisplayProfessionalEmptyState(message string) {
	sectionWidth := DefaultSectionWidth
	
	// Empty state header
	fmt.Printf("%s%s%s %s NO DATA FOUND %s", 
		ColorYellow, BoxTopLeft, BoxHorizontal, "⚠️", strings.Repeat(BoxHorizontal, sectionWidth-18))
	fmt.Printf("%s%s\n", BoxTopRight, ColorReset)
	
	// Message
	wrappedMessage := wrapText(message, sectionWidth-4)
	lines := strings.Split(wrappedMessage, "\n")
	
	for _, line := range lines {
		fmt.Printf("%s%s %-*s %s%s\n", 
			ColorYellow, BoxVertical, sectionWidth-2, line, BoxVertical, ColorReset)
	}
	
	// Empty line
	fmt.Printf("%s%s %-*s %s%s\n", 
		ColorYellow, BoxVertical, sectionWidth-2, "", BoxVertical, ColorReset)
	
	// Guidance
	guidance := "💡 Start tracking: Ensure daemon is running with 'claude-monitor daemon'"
	fmt.Printf("%s%s %s%-*s %s%s\n", 
		ColorYellow, BoxVertical, ColorBrightWhite, sectionWidth-2-len(guidance), guidance, BoxVertical, ColorReset)
	
	// Bottom border
	fmt.Printf("%s%s", ColorYellow, BoxBottomLeft)
	fmt.Print(strings.Repeat(BoxHorizontal, sectionWidth))
	fmt.Printf("%s%s\n\n", BoxBottomRight, ColorReset)
}

/**
 * CONTEXT:   Display comprehensive daily report with professional formatting
 * INPUT:     Enhanced daily report with work data and time metrics
 * OUTPUT:    Complete daily report display with all sections
 * BUSINESS:  Professional daily reports are core user interface
 * CHANGE:    Main display orchestration using separated utility functions
 * RISK:      Low - Display coordination with enhanced visual appeal
 */
func DisplayProfessionalDailyReport(report *EnhancedDailyReport, activeWork, totalTime time.Duration) error {
	// Display metrics dashboard
	DisplayMetricsDashboard(
		activeWork,
		totalTime,
		report.TotalSessions,
		report.EfficiencyPercent,
		time.Duration(report.ClaudeProcessingTime*float64(time.Hour)),
	)
	
	// Display project breakdown if available
	if len(report.ProjectBreakdown) > 0 {
		projects := make([]ProjectData, len(report.ProjectBreakdown))
		for i, proj := range report.ProjectBreakdown {
			projects[i] = ProjectData{
				Name:     proj.ProjectName,
				Duration: time.Duration(proj.WorkHours * float64(time.Hour)),
				Percent:  proj.Percentage,
				Sessions: proj.Sessions,
			}
		}
		DisplayProfessionalProjectBreakdown(projects)
	}
	
	// Display work timeline if available
	if len(report.WorkBlocks) > 0 {
		workBlocks := make([]WorkBlockData, len(report.WorkBlocks))
		for i, block := range report.WorkBlocks {
			workBlocks[i] = WorkBlockData{
				StartTime:     block.StartTime,
				EndTime:       block.EndTime,
				Duration:      block.Duration,
				ProjectName:   block.ProjectName,
				ActivityCount: 1, // Simplified for display
			}
		}
		DisplayProfessionalWorkTimeline(workBlocks)
	}
	
	// Display insights
	if len(report.Insights) > 0 {
		DisplayProfessionalInsights(report.Insights)
	}
	
	// Display footer with next steps
	DisplayProfessionalFooter()
	
	return nil
}

/**
 * CONTEXT:   Display comprehensive monthly report with professional formatting
 * INPUT:     Enhanced monthly report with heatmap and analytics
 * OUTPUT:    Complete monthly report display with all sections
 * BUSINESS:  Professional monthly reports provide long-term insights
 * CHANGE:    Monthly display using extracted utilities for consistent formatting
 * RISK:      Low - Monthly report formatting with visual enhancements
 */
func DisplayProfessionalMonthlyReport(report *EnhancedMonthlyReport) error {
	// Display month header
	monthName := report.Month.Format("January 2006")
	DisplayProfessionalHeader("MONTHLY REPORT", monthName)
	
	if report.TotalWorkHours == 0 {
		DisplayProfessionalEmptyState("No work activity recorded for this month.")
		return nil
	}
	
	// Display monthly metrics
	sectionWidth := DefaultSectionWidth
	
	// Monthly summary section
	fmt.Printf("%s%s%s %s MONTHLY SUMMARY %s", 
		ColorBrightCyan, BoxTopLeft, BoxHorizontal, SymbolSession, strings.Repeat(BoxHorizontal, sectionWidth-21))
	fmt.Printf("%s%s\n", BoxTopRight, ColorReset)
	
	// Total work hours line
	totalHoursStr := formatDurationPro(time.Duration(report.TotalWorkHours * float64(time.Hour)))
	line1 := fmt.Sprintf("  %s Total Work: %s%s%s     %s Working Days: %s%d%s", 
		SymbolWork, ColorBrightGreen, totalHoursStr, ColorReset,
		SymbolTimeline, ColorBrightCyan, report.WorkingDays, ColorReset)
	
	fmt.Printf("%s%s%-*s%s%s\n", 
		ColorBrightCyan, BoxVertical, sectionWidth, line1, BoxVertical, ColorReset)
	
	// Average hours line
	avgDailyStr := fmt.Sprintf("%.1fh", report.AverageHoursPerDay)
	avgWorkingStr := fmt.Sprintf("%.1fh", report.AverageHoursPerWorkingDay)
	line2 := fmt.Sprintf("  %s Daily Avg: %s%s%s        %s Working Avg: %s%s%s", 
		SymbolEfficiency, ColorBrightYellow, avgDailyStr, ColorReset,
		SymbolFocus, ColorBrightGreen, avgWorkingStr, ColorReset)
	
	fmt.Printf("%s%s%-*s%s%s\n", 
		ColorBrightCyan, BoxVertical, sectionWidth, line2, BoxVertical, ColorReset)
	
	// Best day and streak
	if !report.BestDay.Date.IsZero() {
		bestDayStr := report.BestDay.Date.Format("Jan 2")
		bestHoursStr := fmt.Sprintf("%.1fh", report.BestDay.Hours)
		line3 := fmt.Sprintf("  %s Best Day: %s%s (%s)%s   %s Longest Streak: %s%d days%s", 
			SymbolTrend, ColorBrightMagenta, bestDayStr, bestHoursStr, ColorReset,
			SymbolFocus, ColorBrightGreen, report.LongestWorkStreak, ColorReset)
		
		fmt.Printf("%s%s%-*s%s%s\n", 
			ColorBrightCyan, BoxVertical, sectionWidth, line3, BoxVertical, ColorReset)
	}
	
	// Bottom border
	fmt.Printf("%s%s", ColorBrightCyan, BoxBottomLeft)
	fmt.Print(strings.Repeat(BoxHorizontal, sectionWidth))
	fmt.Printf("%s%s\n\n", BoxBottomRight, ColorReset)
	
	// Display project breakdown if available
	if len(report.ProjectBreakdown) > 0 {
		projects := make([]ProjectData, len(report.ProjectBreakdown))
		for i, proj := range report.ProjectBreakdown {
			projects[i] = ProjectData{
				Name:     proj.ProjectName,
				Duration: time.Duration(proj.WorkHours * float64(time.Hour)),
				Percent:  proj.Percentage,
				Sessions: proj.Sessions,
			}
		}
		DisplayProfessionalProjectBreakdown(projects)
	}
	
	// Display insights if available
	if len(report.Insights) > 0 {
		DisplayProfessionalInsights(report.Insights)
	}
	
	// Display footer
	DisplayProfessionalFooter()
	
	return nil
}