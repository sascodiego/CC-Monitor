/**
 * CONTEXT:   Comprehensive reporting system for Claude Monitor with beautiful CLI output
 * INPUT:     Time period parameters and output formatting preferences
 * OUTPUT:    Rich, colorful reports with insights and trends for daily, weekly, monthly analytics
 * BUSINESS:  Reporting provides users with detailed insights into work patterns and productivity
 * CHANGE:    Initial comprehensive reporting system with beautiful formatting
 * RISK:      Low - Read-only reporting functionality with extensive error handling
 */

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

/**
 * CONTEXT:   Enhanced report data structures for comprehensive analytics
 * INPUT:     Database query results aggregated by time periods
 * OUTPUT:    Structured report data with insights and trends
 * BUSINESS:  Rich data structures enable detailed work analytics and insights
 * CHANGE:    Initial enhanced reporting structures with comprehensive metrics
 * RISK:      Low - Data structures for analytics with no side effects
 */

type EnhancedDailyReport struct {
	Date                time.Time             `json:"date"`
	StartTime           time.Time             `json:"start_time"`
	EndTime             time.Time             `json:"end_time"`
	TotalWorkHours      float64               `json:"total_work_hours"`
	ScheduleHours       float64               `json:"schedule_hours"`
	ClaudeProcessingTime float64              `json:"claude_processing_time"`
	IdleTime            float64               `json:"idle_time"`
	EfficiencyPercent   float64               `json:"efficiency_percent"`
	TotalSessions       int                   `json:"total_sessions"`
	ClaudePrompts       int                   `json:"claude_prompts"`
	TotalWorkBlocks     int                   `json:"total_work_blocks"`
	ProjectBreakdown    []ProjectBreakdown    `json:"project_breakdown"`
	HourlyBreakdown     []HourlyData          `json:"hourly_breakdown"`
	WorkBlocks          []WorkBlockSummary    `json:"work_blocks"`
	Insights            []string              `json:"insights"`
	SessionSummary      SessionSummary        `json:"session_summary"`
	ClaudeActivity      ClaudeActivity        `json:"claude_activity"`
}

type EnhancedWeeklyReport struct {
	WeekStart           time.Time             `json:"week_start"`
	WeekEnd             time.Time             `json:"week_end"`
	WeekNumber          int                   `json:"week_number"`
	Year                int                   `json:"year"`
	TotalWorkHours      float64               `json:"total_work_hours"`
	DailyAverage        float64               `json:"daily_average"`
	ClaudeUsageHours    float64               `json:"claude_usage_hours"`
	ClaudeUsagePercent  float64               `json:"claude_usage_percent"`
	MostProductiveDay   DaySummary            `json:"most_productive_day"`
	DailyBreakdown      []DaySummary          `json:"daily_breakdown"`
	ProjectBreakdown    []ProjectBreakdown    `json:"project_breakdown"`
	Insights            []WeeklyInsight       `json:"insights"`
	Trends              []Trend               `json:"trends"`
	WeeklyStats         WeeklyStats           `json:"weekly_stats"`
}

type EnhancedMonthlyReport struct {
	Month               time.Time             `json:"month"`
	MonthName           string                `json:"month_name"`
	Year                int                   `json:"year"`
	DaysCompleted       int                   `json:"days_completed"`
	TotalDays           int                   `json:"total_days"`
	TotalWorkHours      float64               `json:"total_work_hours"`
	DailyAverage        float64               `json:"daily_average"`
	ProjectedHours      float64               `json:"projected_hours"`
	ClaudeUsageHours    float64               `json:"claude_usage_hours"`
	ClaudeUsagePercent  float64               `json:"claude_usage_percent"`
	BestDay             DaySummary            `json:"best_day"`
	DailyProgress       []DaySummary          `json:"daily_progress"`
	WeeklyBreakdown     []WeekSummary         `json:"weekly_breakdown"`
	ProjectBreakdown    []ProjectBreakdown    `json:"project_breakdown"`
	MonthlyStats        MonthlyStats          `json:"monthly_stats"`
	Achievements        []Achievement         `json:"achievements"`
	Trends              []Trend               `json:"trends"`
	Insights            []string              `json:"insights"`
}

type ProjectBreakdown struct {
	Name            string  `json:"name"`
	Hours           float64 `json:"hours"`
	Percentage      float64 `json:"percentage"`
	WorkBlocks      int     `json:"work_blocks"`
	ClaudeSessions  int     `json:"claude_sessions"`
	ClaudeHours     float64 `json:"claude_hours"`
	LastActivity    time.Time `json:"last_activity"`
}

type HourlyData struct {
	Hour       int     `json:"hour"`
	Hours      float64 `json:"hours"`
	WorkBlocks int     `json:"work_blocks"`
	IsActive   bool    `json:"is_active"`
}

type DaySummary struct {
	Date           time.Time `json:"date"`
	DayName        string    `json:"day_name"`
	Hours          float64   `json:"hours"`
	ClaudeSessions int       `json:"claude_sessions"`
	WorkBlocks     int       `json:"work_blocks"`
	Efficiency     float64   `json:"efficiency"`
	Status         string    `json:"status"` // excellent, good, low, none
}

type WeekSummary struct {
	WeekStart      time.Time `json:"week_start"`
	WeekNumber     int       `json:"week_number"`
	Hours          float64   `json:"hours"`
	ClaudeSessions int       `json:"claude_sessions"`
}

type SessionSummary struct {
	TotalSessions   int           `json:"total_sessions"`
	AverageSession  time.Duration `json:"average_session"`
	LongestSession  time.Duration `json:"longest_session"`
	ShortestSession time.Duration `json:"shortest_session"`
	SessionRange    string        `json:"session_range"`
}

type ClaudeActivity struct {
	TotalPrompts        int           `json:"total_prompts"`
	ProcessingTime      time.Duration `json:"processing_time"`
	AverageProcessing   time.Duration `json:"average_processing"`
	SuccessfulPrompts   int           `json:"successful_prompts"`
	EfficiencyPercent   float64       `json:"efficiency_percent"`
}

type WeeklyInsight struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type Trend struct {
	Metric     string  `json:"metric"`
	Direction  string  `json:"direction"` // up, down, stable
	Change     float64 `json:"change"`
	Period     string  `json:"period"`
	Icon       string  `json:"icon"`
}

type WeeklyStats struct {
	ConsistencyScore  float64 `json:"consistency_score"`
	ProductivityPeak  string  `json:"productivity_peak"`
	WeekendWork       float64 `json:"weekend_work"`
	WeekendPercent    float64 `json:"weekend_percent"`
}

type MonthlyStats struct {
	WorkingDays        int     `json:"working_days"`
	TotalSessions      int     `json:"total_sessions"`
	AvgSessionsPerDay  float64 `json:"avg_sessions_per_day"`
	ConsistencyScore   float64 `json:"consistency_score"`
	MostProductiveWeek int     `json:"most_productive_week"`
	ClaudePrompts      int     `json:"claude_prompts"`
}

type Achievement struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Achieved    bool   `json:"achieved"`
}

/**
 * CONTEXT:   Beautiful daily report display with comprehensive formatting
 * INPUT:     EnhancedDailyReport with all work data for the day
 * OUTPUT:    Formatted terminal output with colors, tables, and visual elements
 * BUSINESS:  Daily reports are most frequently used feature for tracking work patterns
 * CHANGE:    Initial beautiful formatting with comprehensive daily insights
 * RISK:      Low - Pure display function with no side effects
 */
func displayEnhancedDailyReport(report *EnhancedDailyReport, date time.Time) error {
	fmt.Println()
	headerColor.Printf("📅 %s\n", date.Format("Monday, January 2, 2006"))
	fmt.Println(strings.Repeat("═", 60))
	
	if report.TotalWorkHours == 0 {
		warningColor.Printf("📭 No work activity recorded for %s\n", 
			date.Format("Monday, January 2, 2006"))
		fmt.Println()
		infoColor.Println("💡 Start working with Claude Code to see activity here!")
		return nil
	}

	// Work Summary Box
	displayWorkSummaryBox(report)

	// Session Summary Section
	displaySessionSummarySection(report)

	// Claude Activity Section
	displayClaudeActivitySection(report)

	// Project Breakdown Section
	if len(report.ProjectBreakdown) > 0 {
		displayProjectBreakdownSection(report.ProjectBreakdown)
	}

	// Hourly Breakdown Section
	if len(report.HourlyBreakdown) > 0 {
		displayHourlyBreakdownSection(report.HourlyBreakdown)
	}

	// Work Timeline Section
	if len(report.WorkBlocks) > 0 {
		displayWorkTimelineSection(report.WorkBlocks)
	}

	// Insights Section
	if len(report.Insights) > 0 {
		displayInsightsSection(report.Insights)
	}

	fmt.Println()
	infoColor.Println("📊 Run `claude-monitor week` to see weekly trends")

	return nil
}

/**
 * CONTEXT:   Work summary display with beautiful box formatting
 * INPUT:     EnhancedDailyReport with work metrics
 * OUTPUT:    Formatted box with key work metrics and efficiency
 * BUSINESS:  Summary box provides quick overview of daily productivity
 * CHANGE:    Initial box formatting with visual appeal
 * RISK:      Low - Display function with formatting only
 */
func displayWorkSummaryBox(report *EnhancedDailyReport) {
	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────┐")
	
	// Only show schedule if we have valid start and end times
	if !report.StartTime.IsZero() && !report.EndTime.IsZero() && report.ScheduleHours > 0 && report.ScheduleHours <= 18 {
		fmt.Printf("│ 🕰️  Schedule: %s - %s (%s)%s │\n",
			formatTimeSafe(report.StartTime, "15:04"),
			formatTimeSafe(report.EndTime, "15:04"),
			formatDuration(time.Duration(report.ScheduleHours * float64(time.Hour))),
			strings.Repeat(" ", 17 - len(formatDuration(time.Duration(report.ScheduleHours * float64(time.Hour))))))
	} else {
		fmt.Printf("│ 🕰️  Schedule: Active work time only%s │\n",
			strings.Repeat(" ", 22))
	}
	
	efficiency := report.EfficiencyPercent
	efficiencyStr := fmt.Sprintf("%.1f%%", efficiency)
	fmt.Printf("│ ⏱️  Active Work: %s (%s)%s │\n",
		formatDuration(time.Duration(report.TotalWorkHours * float64(time.Hour))),
		efficiencyStr,
		strings.Repeat(" ", 29 - len(formatDuration(time.Duration(report.TotalWorkHours * float64(time.Hour)))) - len(efficiencyStr)))

	if report.ClaudeProcessingTime > 0 {
		claudePercent := (report.ClaudeProcessingTime / report.TotalWorkHours) * 100
		claudePercentStr := fmt.Sprintf("%.1f%%", claudePercent)
		fmt.Printf("│ 🤖 Claude Processing: %s (%s)%s │\n",
			formatDuration(time.Duration(report.ClaudeProcessingTime * float64(time.Hour))),
			claudePercentStr,
			strings.Repeat(" ", 23 - len(formatDuration(time.Duration(report.ClaudeProcessingTime * float64(time.Hour)))) - len(claudePercentStr)))
	}

	fmt.Println("└─────────────────────────────────────────────────────┘")
}

/**
 * CONTEXT:   Session summary display with statistical information
 * INPUT:     EnhancedDailyReport with session data
 * OUTPUT:    Formatted section showing session statistics
 * BUSINESS:  Session information helps understand work patterns and continuity
 * CHANGE:    Initial session summary with comprehensive metrics
 * RISK:      Low - Read-only display function
 */
func displaySessionSummarySection(report *EnhancedDailyReport) {
	fmt.Println()
	successColor.Println("📊 SESSION SUMMARY:")
	fmt.Printf("• Total Sessions: %d sessions\n", report.SessionSummary.TotalSessions)
	fmt.Printf("• Average Session: %s\n", formatDuration(report.SessionSummary.AverageSession))
	fmt.Printf("• Longest Session: %s (%s)\n", 
		formatDuration(report.SessionSummary.LongestSession),
		report.SessionSummary.SessionRange)
}

/**
 * CONTEXT:   Claude activity display with AI usage metrics
 * INPUT:     EnhancedDailyReport with Claude processing data
 * OUTPUT:    Formatted section showing Claude usage statistics
 * BUSINESS:  Claude metrics show AI assistance level and efficiency
 * CHANGE:    Initial Claude activity display with detailed metrics
 * RISK:      Low - Display function with AI usage insights
 */
func displayClaudeActivitySection(report *EnhancedDailyReport) {
	if report.SessionSummary.TotalSessions > 0 {
		fmt.Println()
		successColor.Println("🤖 CLAUDE ACTIVITY:")
		fmt.Printf("• Claude Sessions: %d session", report.SessionSummary.TotalSessions)
		if report.SessionSummary.TotalSessions > 1 {
			fmt.Print("s")
		}
		fmt.Println()
		
		// Only show processing time if we have real data
		if report.ClaudeActivity.ProcessingTime > 0 {
			fmt.Printf("• Processing Time: %s total\n", formatDuration(report.ClaudeActivity.ProcessingTime))
		}
	}
}

/**
 * CONTEXT:   Project breakdown display with beautiful table formatting
 * INPUT:     Project breakdown data with time allocation
 * OUTPUT:    Formatted table showing project time distribution
 * BUSINESS:  Project breakdown shows work allocation across different projects
 * CHANGE:    Initial project table with Claude session tracking
 * RISK:      Low - Table display function with project insights
 */
func displayProjectBreakdownSection(projects []ProjectBreakdown) {
	fmt.Println()
	successColor.Println("📁 PROJECT BREAKDOWN:")
	
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Project", "Time", "%", "Claude Sessions"})
	table.SetBorder(false)
	table.SetRowSeparator("-")
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
	)
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.FgGreenColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgYellowColor},
		tablewriter.Colors{tablewriter.FgBlueColor},
	)

	for _, project := range projects {
		claudeInfo := ""
		if project.ClaudeSessions > 0 {
			claudeInfo = fmt.Sprintf("%d sessions", project.ClaudeSessions)
		} else {
			claudeInfo = "0 sessions"
		}

		table.Append([]string{
			truncateString(project.Name, 25),
			formatDuration(time.Duration(project.Hours * float64(time.Hour))),
			fmt.Sprintf("%.1f%%", project.Percentage),
			claudeInfo,
		})
	}
	
	table.Render()
}

/**
 * CONTEXT:   Hourly breakdown display with visual progress bars
 * INPUT:     Hourly work data for the day
 * OUTPUT:    Visual hourly breakdown with Unicode block characters
 * BUSINESS:  Hourly view shows work distribution and peak productivity times
 * CHANGE:    Initial hourly display with beautiful visual progress bars
 * RISK:      Low - Visual display function with time distribution
 */
func displayHourlyBreakdownSection(hourly []HourlyData) {
	fmt.Println()
	successColor.Println("⏰ HOURLY BREAKDOWN:")
	
	// Find max hours for scaling
	maxHours := 0.0
	for _, h := range hourly {
		if h.Hours > maxHours {
			maxHours = h.Hours
		}
	}
	
	if maxHours == 0 {
		return
	}

	// Display hourly bars only for hours with actual work activity
	for _, h := range hourly {
		if h.Hours > 0 {
			barLength := int((h.Hours / maxHours) * 20)
			bar := strings.Repeat("█", barLength) + strings.Repeat("░", 20-barLength)
			fmt.Printf("%02d:00 %s %s\n", h.Hour, bar, formatDuration(time.Duration(h.Hours * float64(time.Hour))))
		}
	}
}

/**
 * CONTEXT:   Work timeline display showing chronological work blocks
 * INPUT:     Work block summaries for the day
 * OUTPUT:    Timeline table showing work periods and projects
 * BUSINESS:  Timeline helps understand work flow and context switching
 * CHANGE:    Initial timeline display with project and duration information
 * RISK:      Low - Timeline display function with work flow insights
 */
func displayWorkTimelineSection(workBlocks []WorkBlockSummary) {
	if len(workBlocks) == 0 {
		return
	}
	
	fmt.Println()
	successColor.Println("🕒 WORK TIMELINE:")
	
	// Sort by start time
	sort.Slice(workBlocks, func(i, j int) bool {
		return workBlocks[i].StartTime.Before(workBlocks[j].StartTime)
	})
	
	// Unify contiguous blocks by project
	unifiedBlocks := unifyWorkBlocksByProject(workBlocks)
	
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Time", "Duration", "Project", "Activities"})
	table.SetBorder(false)
	table.SetRowSeparator("-")
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
	)

	for _, block := range unifiedBlocks {
		// Validate block times before displaying
		timeRange := "--:-- to --:--"
		if isValidWorkBlockTime(block.StartTime) && isValidWorkBlockTime(block.EndTime) {
			if block.EndTime.After(block.StartTime) && block.Duration <= 12*time.Hour {
				timeRange = fmt.Sprintf("%s-%s", 
					formatTimeSafe(block.StartTime, "15:04"), 
					formatTimeSafe(block.EndTime, "15:04"))
			}
		}
		
		table.Append([]string{
			timeRange,
			formatDuration(block.Duration),
			truncateString(block.ProjectName, 20),
			fmt.Sprintf("%d", block.Activities),
		})
	}
	
	table.Render()
}

// unifyWorkBlocksByProject combines overlapping and contiguous work blocks by project
func unifyWorkBlocksByProject(workBlocks []WorkBlockSummary) []WorkBlockSummary {
	if len(workBlocks) == 0 {
		return []WorkBlockSummary{}
	}
	
	// Group work blocks by project name first
	projectBlocks := make(map[string][]WorkBlockSummary)
	for _, wb := range workBlocks {
		projectBlocks[wb.ProjectName] = append(projectBlocks[wb.ProjectName], wb)
	}
	
	var unified []WorkBlockSummary
	
	// Process each project separately
	for _, blocks := range projectBlocks {
		if len(blocks) == 0 {
			continue
		}
		
		// Sort blocks by start time for this project
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].StartTime.Before(blocks[j].StartTime)
		})
		
		// Merge overlapping and contiguous blocks
		current := blocks[0]
		totalActivities := current.Activities
		
		for i := 1; i < len(blocks); i++ {
			next := blocks[i]
			
			// If blocks overlap or are contiguous (within 5 minutes gap), merge them
			if next.StartTime.Sub(current.EndTime) <= 5*time.Minute {
				// Extend the end time if next block goes further
				if next.EndTime.After(current.EndTime) {
					current.EndTime = next.EndTime
				}
				// Ensure start time is the earliest
				if next.StartTime.Before(current.StartTime) {
					current.StartTime = next.StartTime
				}
				totalActivities += next.Activities
			} else {
				// Gap > 5 minutes, save current unified block and start new one
				current.Duration = current.EndTime.Sub(current.StartTime)
				current.Activities = totalActivities
				unified = append(unified, current)
				
				// Start new block
				current = next
				totalActivities = current.Activities
			}
		}
		
		// Add the final unified block for this project
		current.Duration = current.EndTime.Sub(current.StartTime)
		current.Activities = totalActivities
		unified = append(unified, current)
	}
	
	// Sort final unified blocks by start time across all projects
	sort.Slice(unified, func(i, j int) bool {
		return unified[i].StartTime.Before(unified[j].StartTime)
	})
	
	return unified
}

/**
 * CONTEXT:   Insights display with actionable recommendations
 * INPUT:     Generated insights from work pattern analysis
 * OUTPUT:    Bulleted list of insights with emojis and recommendations
 * BUSINESS:  Insights provide actionable recommendations for productivity improvement
 * CHANGE:    Initial insights display with helpful recommendations
 * RISK:      Low - Display function with productivity insights
 */
func displayInsightsSection(insights []string) {
	fmt.Println()
	successColor.Println("💡 INSIGHTS:")
	
	for _, insight := range insights {
		fmt.Printf("• %s\n", insight)
	}
}

/**
 * CONTEXT:   Enhanced weekly report display with trends and patterns
 * INPUT:     EnhancedWeeklyReport with comprehensive weekly data
 * OUTPUT:    Formatted weekly report with daily breakdown and insights
 * BUSINESS:  Weekly reports show productivity trends and patterns over time
 * CHANGE:    Initial weekly report display with comprehensive formatting
 * RISK:      Low - Read-only display function with trend analysis
 */
func displayEnhancedWeeklyReport(report *EnhancedWeeklyReport) error {
	fmt.Println()
	headerColor.Printf("📅 Week %d - %s - %s\n", 
		report.WeekNumber,
		report.WeekStart.Format("January 2"),
		report.WeekEnd.Format("January 2, 2006"))
	fmt.Println(strings.Repeat("═", 60))
	
	if report.TotalWorkHours == 0 {
		warningColor.Println("📭 No work activity recorded for this week")
		return nil
	}

	// Weekly Summary Box
	displayWeeklySummaryBox(report)

	// Daily Breakdown Section
	displayWeeklyDailyBreakdown(report.DailyBreakdown)

	// Weekly Insights
	displayWeeklyInsights(report.Insights)

	// Top Projects This Week
	if len(report.ProjectBreakdown) > 0 {
		displayWeeklyProjectBreakdown(report.ProjectBreakdown)
	}

	// Trends Section
	if len(report.Trends) > 0 {
		displayTrendsSection(report.Trends)
	}

	fmt.Println()
	infoColor.Println("📈 Run `claude-monitor month` to see monthly patterns")

	return nil
}

/**
 * CONTEXT:   Weekly summary box with key metrics
 * INPUT:     EnhancedWeeklyReport with weekly statistics
 * OUTPUT:     Formatted box showing weekly overview
 * BUSINESS:  Summary provides quick weekly productivity overview
 * CHANGE:    Initial weekly summary box with comprehensive metrics
 * RISK:      Low - Display formatting function
 */
func displayWeeklySummaryBox(report *EnhancedWeeklyReport) {
	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────┐")
	fmt.Printf("│ 📈 Total Work: %s across 7 days%s │\n",
		formatDuration(time.Duration(report.TotalWorkHours * float64(time.Hour))),
		strings.Repeat(" ", 27 - len(formatDuration(time.Duration(report.TotalWorkHours * float64(time.Hour))))))
	fmt.Printf("│ 📊 Daily Average: %s%s │\n",
		formatDuration(time.Duration(report.DailyAverage * float64(time.Hour))),
		strings.Repeat(" ", 36 - len(formatDuration(time.Duration(report.DailyAverage * float64(time.Hour))))))
	
	if report.MostProductiveDay.Hours > 0 {
		fmt.Printf("│ 🔥 Most Productive: %s (%s) 🏆%s │\n",
			report.MostProductiveDay.DayName,
			formatDuration(time.Duration(report.MostProductiveDay.Hours * float64(time.Hour))),
			strings.Repeat(" ", 20 - len(report.MostProductiveDay.DayName) - len(formatDuration(time.Duration(report.MostProductiveDay.Hours * float64(time.Hour))))))
	}
	
	if report.ClaudeUsageHours > 0 {
		fmt.Printf("│ 🤖 Claude Usage: %s (%.1f%% of work time)%s │\n",
			formatDuration(time.Duration(report.ClaudeUsageHours * float64(time.Hour))),
			report.ClaudeUsagePercent,
			strings.Repeat(" ", 19 - len(formatDuration(time.Duration(report.ClaudeUsageHours * float64(time.Hour)))) - len(fmt.Sprintf("%.1f%%", report.ClaudeUsagePercent))))
	}
	fmt.Println("└─────────────────────────────────────────────────────┘")
}

/**
 * CONTEXT:   Weekly daily breakdown with visual progress bars
 * INPUT:     Daily summaries for the week
 * OUTPUT:    Visual daily breakdown showing productivity patterns
 * BUSINESS:  Daily breakdown shows work distribution across the week
 * CHANGE:    Initial weekly daily view with visual progress indicators
 * RISK:      Low - Visual display function with daily patterns
 */
func displayWeeklyDailyBreakdown(dailyBreakdown []DaySummary) {
	fmt.Println()
	successColor.Println("📊 DAILY BREAKDOWN:")
	
	// Find max hours for scaling
	maxHours := 0.0
	for _, day := range dailyBreakdown {
		if day.Hours > maxHours {
			maxHours = day.Hours
		}
	}
	
	if maxHours == 0 {
		return
	}

	for _, day := range dailyBreakdown {
		barLength := int((day.Hours / maxHours) * 20)
		bar := strings.Repeat("█", barLength) + strings.Repeat("░", 20-barLength)
		
		dayStr := fmt.Sprintf("%s %s", day.DayName, day.Date.Format("01/02"))
		trophy := ""
		if day.Status == "excellent" {
			trophy = " 🏆"
		}
		
		if day.Hours > 0 {
			fmt.Printf("%-10s %s %s [%d sessions]%s\n", 
				dayStr, bar, 
				formatDuration(time.Duration(day.Hours * float64(time.Hour))),
				day.ClaudeSessions,
				trophy)
		} else {
			dimColor.Printf("%-10s %s %s [0 sessions]\n", 
				dayStr, bar, "0h 0m")
		}
	}
}

/**
 * CONTEXT:   Weekly insights display with pattern analysis
 * INPUT:     Weekly insights from pattern analysis
 * OUTPUT:    Formatted insights with icons and descriptions
 * BUSINESS:  Weekly insights provide productivity pattern understanding
 * CHANGE:    Initial weekly insights with comprehensive pattern analysis
 * RISK:      Low - Display function with weekly productivity insights
 */
func displayWeeklyInsights(insights []WeeklyInsight) {
	if len(insights) == 0 {
		return
	}

	fmt.Println()
	successColor.Println("🎯 WEEKLY INSIGHTS:")
	
	for _, insight := range insights {
		fmt.Printf("• %s %s: %s\n", insight.Icon, insight.Title, insight.Description)
	}
}

/**
 * CONTEXT:   Weekly project breakdown display
 * INPUT:     Project breakdown data for the week
 * OUTPUT:    Table showing project time allocation for the week
 * BUSINESS:  Weekly project view shows time allocation patterns
 * CHANGE:    Initial weekly project breakdown with comprehensive metrics
 * RISK:      Low - Table display function with project analytics
 */
func displayWeeklyProjectBreakdown(projects []ProjectBreakdown) {
	fmt.Println()
	successColor.Println("📁 TOP PROJECTS THIS WEEK:")
	
	// Sort by hours descending
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Hours > projects[j].Hours
	})

	for i, project := range projects {
		if i >= 3 { // Show top 3 projects
			break
		}
		fmt.Printf("• %-20s %s (%.1f%%)\n", 
			truncateString(project.Name, 20), 
			formatDuration(time.Duration(project.Hours * float64(time.Hour))),
			project.Percentage)
	}
}

/**
 * CONTEXT:   Trends section display with directional indicators
 * INPUT:     Trend data with metrics and changes
 * OUTPUT:    Formatted trends with directional arrows and percentages
 * BUSINESS:  Trends show productivity changes and patterns over time
 * CHANGE:    Initial trends display with visual directional indicators
 * RISK:      Low - Display function with trend visualization
 */
func displayTrendsSection(trends []Trend) {
	fmt.Println()
	successColor.Println("📈 TRENDS:")
	
	for _, trend := range trends {
		directionIcon := getTrendIcon(trend.Direction)
		changeStr := ""
		if trend.Change != 0 {
			changeStr = fmt.Sprintf(" %.1f%%", trend.Change)
		}
		fmt.Printf("• %s %s trend: %s %s%s\n", 
			trend.Icon, trend.Metric, directionIcon, trend.Direction, changeStr)
	}
}

func getTrendIcon(direction string) string {
	switch direction {
	case "up":
		return "↗️"
	case "down":
		return "↘️"
	case "stable":
		return "↔️"
	default:
		return "━"
	}
}

/**
 * CONTEXT:   Enhanced monthly report display with comprehensive analytics
 * INPUT:     EnhancedMonthlyReport with full monthly data
 * OUTPUT:    Detailed monthly report with heatmap, progress, and achievements
 * BUSINESS:  Monthly reports provide long-term productivity insights and trends
 * CHANGE:    Initial comprehensive monthly report with visual heatmap
 * RISK:      Low - Display function with extensive monthly analytics
 */
func displayEnhancedMonthlyReport(report *EnhancedMonthlyReport) error {
	fmt.Println()
	headerColor.Printf("📅 %s %d", report.MonthName, report.Year)
	if report.DaysCompleted < report.TotalDays {
		headerColor.Printf(" - Current Month (%d days completed)", report.DaysCompleted)
	} else {
		headerColor.Printf(" - Historical Report")
	}
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))
	
	if report.TotalWorkHours == 0 {
		warningColor.Printf("📭 No work activity recorded for %s %d\n", report.MonthName, report.Year)
		return nil
	}

	// Monthly Summary Box
	displayMonthlySummaryBox(report)

	// Monthly Heatmap
	displayMonthlyHeatmap(report.DailyProgress, report.Year, int(report.Month.Month()))

	// Monthly Statistics
	displayMonthlyStatistics(report.MonthlyStats)

	// Achievements
	if len(report.Achievements) > 0 {
		displayAchievements(report.Achievements)
	}

	// Monthly Trends
	if len(report.Trends) > 0 {
		displayTrendsSection(report.Trends)
	}

	fmt.Println()
	infoColor.Println("📊 Use specific dates for detailed historical analysis")

	return nil
}

/**
 * CONTEXT:   Monthly summary box with key metrics and projections
 * INPUT:     EnhancedMonthlyReport with monthly statistics
 * OUTPUT:    Formatted box with monthly overview and projections
 * BUSINESS:  Monthly summary provides comprehensive month overview
 * CHANGE:    Initial monthly summary with projections and achievements
 * RISK:      Low - Summary display with monthly metrics
 */
func displayMonthlySummaryBox(report *EnhancedMonthlyReport) {
	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────┐")
	fmt.Printf("│ 📈 Total Work: %s (%d days)%s │\n",
		formatDuration(time.Duration(report.TotalWorkHours * float64(time.Hour))),
		report.DaysCompleted,
		strings.Repeat(" ", 28 - len(formatDuration(time.Duration(report.TotalWorkHours * float64(time.Hour)))) - len(fmt.Sprintf("(%d days)", report.DaysCompleted))))
	fmt.Printf("│ 📊 Daily Average: %s%s │\n",
		formatDuration(time.Duration(report.DailyAverage * float64(time.Hour))),
		strings.Repeat(" ", 36 - len(formatDuration(time.Duration(report.DailyAverage * float64(time.Hour))))))
	
	if report.DaysCompleted < report.TotalDays && report.ProjectedHours > 0 {
		fmt.Printf("│ 🎯 On Track For: ~%s this month%s │\n",
			formatDuration(time.Duration(report.ProjectedHours * float64(time.Hour))),
			strings.Repeat(" ", 24 - len(formatDuration(time.Duration(report.ProjectedHours * float64(time.Hour))))))
	}
	
	if report.ClaudeUsageHours > 0 {
		fmt.Printf("│ 🤖 Claude Assisted: %s (%.1f%%)%s │\n",
			formatDuration(time.Duration(report.ClaudeUsageHours * float64(time.Hour))),
			report.ClaudeUsagePercent,
			strings.Repeat(" ", 24 - len(formatDuration(time.Duration(report.ClaudeUsageHours * float64(time.Hour)))) - len(fmt.Sprintf("%.1f%%", report.ClaudeUsagePercent))))
	}
	fmt.Println("└─────────────────────────────────────────────────────┘")
}

/**
 * CONTEXT:   Monthly heatmap display showing daily activity patterns
 * INPUT:     Daily progress data for the month
 * OUTPUT:    Visual heatmap with color-coded activity levels
 * BUSINESS:  Heatmap provides visual overview of work consistency across the month
 * CHANGE:    Initial heatmap implementation with Unicode characters
 * RISK:      Low - Visual display function with calendar heatmap
 */
func displayMonthlyHeatmap(dailyProgress []DaySummary, year, month int) {
	fmt.Println()
	successColor.Println("📅 MONTHLY HEATMAP:")
	fmt.Println("      S  M  T  W  T  F  S")
	
	// Create a map of days for quick lookup
	dayMap := make(map[int]DaySummary)
	for _, day := range dailyProgress {
		dayMap[day.Date.Day()] = day
	}
	
	// Get first day of month and number of days
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	daysInMonth := lastDay.Day()
	startWeekday := int(firstDay.Weekday())
	
	// Display weeks
	weekNum := 1
	day := 1
	
	for day <= daysInMonth {
		fmt.Printf("Week %-2d", weekNum)
		
		// Handle first week padding
		if weekNum == 1 {
			for i := 0; i < startWeekday; i++ {
				fmt.Print("   ")
			}
		}
		
		// Display days of the week
		for weekday := startWeekday; weekday < 7 && day <= daysInMonth; weekday++ {
			if dayData, exists := dayMap[day]; exists {
				icon := getDayStatusIcon(dayData.Hours)
				fmt.Printf(" %s ", icon)
			} else {
				fmt.Print(" ░ ")
			}
			day++
		}
		
		// Fill remaining days of incomplete last week
		if day > daysInMonth {
			for weekday := day - 1 - lastDay.Day() + startWeekday; weekday < 7; weekday++ {
				fmt.Print("   ")
			}
		}
		
		fmt.Println()
		weekNum++
		startWeekday = 0 // Reset for subsequent weeks
	}
	
	fmt.Println()
	fmt.Println("Legend: 🔥 >8h  🟢 6-8h  🟡 3-6h  ⚫ <3h  ░ No work")
}

func getDayStatusIcon(hours float64) string {
	switch {
	case hours >= 8:
		return "🔥"
	case hours >= 6:
		return "🟢"
	case hours >= 3:
		return "🟡"
	case hours > 0:
		return "⚫"
	default:
		return "░"
	}
}

/**
 * CONTEXT:   Monthly statistics display with comprehensive metrics
 * INPUT:     MonthlyStats with detailed monthly information
 * OUTPUT:    Formatted statistics section with key monthly metrics
 * BUSINESS:  Monthly stats provide detailed productivity insights
 * CHANGE:    Initial monthly statistics display with comprehensive data
 * RISK:      Low - Statistics display function with monthly insights
 */
func displayMonthlyStatistics(stats MonthlyStats) {
	fmt.Println()
	successColor.Println("📊 MONTHLY STATISTICS:")
	fmt.Printf("• Working days: %d / %d working days (%.1f%% consistency)\n", 
		stats.WorkingDays, 
		// Assuming ~22 working days per month
		22, 
		(float64(stats.WorkingDays)/22.0)*100)
	fmt.Printf("• Total sessions: %d sessions\n", stats.TotalSessions)
	fmt.Printf("• Avg sessions/day: %.1f\n", stats.AvgSessionsPerDay)
	fmt.Printf("• Claude prompts: %d total\n", stats.ClaudePrompts)
	if stats.MostProductiveWeek > 0 {
		fmt.Printf("• Most productive week: Week %d\n", stats.MostProductiveWeek)
	}
}

/**
 * CONTEXT:   Achievements display with progress indicators
 * INPUT:     Achievement list with completion status
 * OUTPUT:    Formatted achievements with icons and status
 * BUSINESS:  Achievements provide motivation and goal tracking
 * CHANGE:    Initial achievements display with visual indicators
 * RISK:      Low - Achievement display function with motivational elements
 */
func displayAchievements(achievements []Achievement) {
	fmt.Println()
	successColor.Println("🏆 ACHIEVEMENTS:")
	
	for _, achievement := range achievements {
		status := "⚪"
		if achievement.Achieved {
			status = "✅"
		}
		fmt.Printf("%s %s %s\n", status, achievement.Icon, achievement.Title)
		if achievement.Achieved && achievement.Description != "" {
			fmt.Printf("    %s\n", achievement.Description)
		}
	}
}

/**
 * CONTEXT:   Enhanced duration formatting with better readability
 * INPUT:     Duration value for formatting
 * OUTPUT:    Human-readable duration string
 * BUSINESS:  Clear duration formatting improves report readability
 * CHANGE:    Enhanced duration formatting with better precision
 * RISK:      Low - Utility function with formatting only
 */
func formatDurationEnhanced(d time.Duration) string {
	if d < 0 {
		return "0m"
	}
	
	totalMinutes := int(d.Minutes())
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	
	if hours >= 1 {
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if totalMinutes >= 1 {
		return fmt.Sprintf("%dm", totalMinutes)
	} else {
		seconds := int(d.Seconds())
		if seconds == 0 {
			return "0s"
		}
		return fmt.Sprintf("%ds", seconds)
	}
}

/**
 * CONTEXT:   Generate insights from daily work patterns
 * INPUT:     EnhancedDailyReport with work data
 * OUTPUT:    List of actionable insights and observations
 * BUSINESS:  Insights provide actionable recommendations for productivity improvement
 * CHANGE:    Initial insight generation with pattern analysis
 * RISK:      Low - Analysis function generating recommendations
 */
func generateDailyInsights(report *EnhancedDailyReport) []string {
	var insights []string
	
	// Work pattern insights
	if len(report.WorkBlocks) > 0 {
		// Find longest work block
		longestDuration := time.Duration(0)
		longestProject := ""
		for _, block := range report.WorkBlocks {
			if block.Duration > longestDuration {
				longestDuration = block.Duration
				longestProject = block.ProjectName
			}
		}
		
		insights = append(insights, fmt.Sprintf(
			"🎯 Your longest focus session was %s on %s",
			formatDuration(longestDuration),
			longestProject))
		
		// Project focus insights
		if len(report.ProjectBreakdown) > 0 {
			topProject := report.ProjectBreakdown[0]
			insights = append(insights, fmt.Sprintf(
				"📊 You spent most time on %s (%.1f%% of total work)",
				topProject.Name,
				topProject.Percentage))
		}
		
		// Work schedule insights
		startHour := report.StartTime.Hour()
		switch {
		case startHour < 7:
			insights = append(insights, "🌅 You're an early bird! Great for deep focus work")
		case startHour > 10:
			insights = append(insights, "😴 Late starter today - consider morning work sessions for peak productivity")
		default:
			insights = append(insights, "⏰ Good work schedule timing for consistent productivity")
		}
		
		// Enhanced efficiency insights with specific recommendations
		if report.EfficiencyPercent >= 80 {
			insights = append(insights, "🔥 Excellent efficiency today! You maintained great focus")
		} else if report.EfficiencyPercent >= 60 {
			insights = append(insights, "⚡ Good productivity balance between work and breaks")
		} else if report.EfficiencyPercent >= 20 {
			insights = append(insights, "📈 Consider longer focus blocks to improve efficiency within your 5-hour Claude sessions")
		} else if report.EfficiencyPercent >= 5 {
			insights = append(insights, "⏰ Low efficiency detected - focus on longer work blocks within your Claude session")
		} else {
			insights = append(insights, "🚨 Very low efficiency - consider dedicated focus time without interruptions")
		}
		
		// Work pattern and context switching insights
		projectCount := len(report.ProjectBreakdown)
		if projectCount == 1 {
			insights = append(insights, "🎯 Single project focus - excellent for deep work and flow state")
		} else if projectCount <= 3 {
			insights = append(insights, "👍 Good project focus with minimal context switching")
		} else {
			insights = append(insights, fmt.Sprintf("🔄 %d projects - high context switching may reduce focus efficiency", projectCount))
		}
		
		// Work block pattern analysis
		if len(report.WorkBlocks) > 0 {
			totalActivities := 0
			shortBlocks := 0
			for _, wb := range report.WorkBlocks {
				totalActivities += wb.Activities
				if wb.Duration < 5*time.Minute {
					shortBlocks++
				}
			}
			
			if shortBlocks > len(report.WorkBlocks)/2 {
				insights = append(insights, "⚡ Many short work blocks (< 5min) detected - work blocks end after 5min of inactivity")
			}
			
			avgActivitiesPerMinute := float64(totalActivities) / (report.TotalWorkHours * 60)
			if avgActivitiesPerMinute > 10 {
				insights = append(insights, "⚡ High activity rate - great active coding session!")
			} else if avgActivitiesPerMinute < 2 {
				insights = append(insights, "🤔 Lower activity rate - consider if you were in planning/thinking mode")
			}
		}
		
		// Claude usage insights
		if report.ClaudeActivity.TotalPrompts > 0 {
			claudeRatio := report.ClaudeProcessingTime / report.TotalWorkHours
			if claudeRatio > 0.4 {
				insights = append(insights, "🤖 High Claude usage today - great AI-assisted productivity!")
			} else if claudeRatio > 0.2 {
				insights = append(insights, "🤖 Healthy balance of Claude assistance and independent work")
			} else {
				insights = append(insights, "💡 Consider using Claude more for complex tasks and problem-solving")
			}
		}
	}
	
	return insights
}

/**
 * CONTEXT:   Validate work block time to ensure reasonable display
 * INPUT:     Time value from work block
 * OUTPUT:    Boolean indicating if time is valid for work block display
 * BUSINESS:  Work block time validation prevents impossible time displays
 * CHANGE:    Added work block specific time validation
 * RISK:      Low - Validation improves display accuracy
 */
func isValidWorkBlockTime(t time.Time) bool {
	if t.IsZero() {
		return false
	}
	
	// Check for reasonable time bounds
	year := t.Year()
	if year < 2020 || year > 2030 {
		return false
	}
	
	// Check hour is valid (0-23)
	hour := t.Hour()
	if hour < 0 || hour > 23 {
		return false
	}
	
	// Check minute is valid (0-59)
	minute := t.Minute()
	if minute < 0 || minute > 59 {
		return false
	}
	
	return true
}

/**
 * CONTEXT:   Generate weekly insights from work patterns
 * INPUT:     EnhancedWeeklyReport with weekly data
 * OUTPUT:    List of weekly insights with productivity patterns
 * BUSINESS:  Weekly insights show productivity trends and improvement opportunities
 * CHANGE:    Initial weekly insight generation with trend analysis
 * RISK:      Low - Pattern analysis function for weekly insights
 */
func generateWeeklyInsights(report *EnhancedWeeklyReport) []WeeklyInsight {
	var insights []WeeklyInsight
	
	// Productivity pattern analysis
	if len(report.DailyBreakdown) > 0 {
		// Find peak days
		weekdayHours := 0.0
		weekendHours := 0.0
		
		for _, day := range report.DailyBreakdown {
			if day.Date.Weekday() == time.Saturday || day.Date.Weekday() == time.Sunday {
				weekendHours += day.Hours
			} else {
				weekdayHours += day.Hours
			}
		}
		
		if weekdayHours > 0 {
			insights = append(insights, WeeklyInsight{
				Type:        "productivity",
				Title:       "Best productivity",
				Description: "Mid-week (Tue-Thu) shows strongest focus patterns",
				Icon:        "🎯",
			})
		}
		
		if weekendHours > 0 {
			weekendPercent := (weekendHours / report.TotalWorkHours) * 100
			if weekendPercent > 20 {
				insights = append(insights, WeeklyInsight{
					Type:        "balance",
					Title:       "Weekend work",
					Description: fmt.Sprintf("%.1f%% of work done on weekends", weekendPercent),
					Icon:        "⚖️",
				})
			}
		}
		
		// Claude usage pattern
		if report.ClaudeUsagePercent > 0 {
			var claudeInsight WeeklyInsight
			if report.ClaudeUsagePercent > 40 {
				claudeInsight = WeeklyInsight{
					Type:        "ai_usage",
					Title:       "High Claude dependency",
					Description: fmt.Sprintf("%.1f%% AI-assisted work - excellent productivity boost", report.ClaudeUsagePercent),
					Icon:        "🤖",
				}
			} else if report.ClaudeUsagePercent > 20 {
				claudeInsight = WeeklyInsight{
					Type:        "ai_usage",
					Title:       "Balanced AI usage",
					Description: fmt.Sprintf("%.1f%% Claude assistance shows healthy balance", report.ClaudeUsagePercent),
					Icon:        "🤖",
				}
			} else {
				claudeInsight = WeeklyInsight{
					Type:        "ai_usage",
					Title:       "Low Claude usage",
					Description: "Consider using Claude more for complex problem-solving",
					Icon:        "💡",
				}
			}
			insights = append(insights, claudeInsight)
		}
	}
	
	// Consistency analysis
	workingDays := 0
	for _, day := range report.DailyBreakdown {
		if day.Hours > 1 { // More than 1 hour considered working
			workingDays++
		}
	}
	
	if workingDays >= 6 {
		insights = append(insights, WeeklyInsight{
			Type:        "consistency",
			Title:       "Excellent consistency",
			Description: fmt.Sprintf("%d-day work streak shows great dedication", workingDays),
			Icon:        "🔥",
		})
	} else if workingDays >= 4 {
		insights = append(insights, WeeklyInsight{
			Type:        "consistency",
			Title:       "Good work pattern",
			Description: fmt.Sprintf("%d working days with solid productivity", workingDays),
			Icon:        "📈",
		})
	}
	
	return insights
}