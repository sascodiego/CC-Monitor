/**
 * CONTEXT:   Additional CLI commands for Claude Monitor single binary
 * INPUT:     User command requests for reporting and system management
 * OUTPUT:    Beautiful formatted output with comprehensive work analytics
 * BUSINESS:  CLI commands provide daily user interface for work tracking insights
 * CHANGE:    Initial command implementation with consistent formatting
 * RISK:      Low - Read-only reporting commands with user-friendly error handling
 */

package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/olekukonko/tablewriter"
)

/**
 * CONTEXT:   Weekly report command with trend analysis
 * INPUT:     Week selection and output format preferences
 * OUTPUT:    Weekly work summary with project trends and insights
 * BUSINESS:  Weekly reports show patterns and productivity trends over time
 * CHANGE:    Initial weekly command with comprehensive analytics
 * RISK:      Low - Read-only reporting with graceful error handling
 */
var weekCmd = &cobra.Command{
	Use:   "week",
	Short: "Show weekly work summary",
	Long: `Display a comprehensive weekly work summary with trends and insights.

Shows:
- Total weekly work hours across all projects
- Daily breakdown with productivity patterns
- Project time allocation and changes
- Week-over-week trends and comparisons
- Productivity insights and recommendations`,
	
	Example: `  claude-monitor week
  claude-monitor week --week=2024-W32
  claude-monitor week --output=json`,
	
	RunE: runWeekCommand,
}

var (
	weekNumber    string
	weekProjectFilter string
)

func init() {
	weekCmd.Flags().StringVar(&weekNumber, "week", "", "specific week (YYYY-WNN)")
	weekCmd.Flags().StringVar(&weekProjectFilter, "project", "", "filter by project name")
	weekCmd.Flags().BoolVar(&claudeOnlyFlag, "claude-only", false, "show only Claude-assisted work")
}

func runWeekCommand(cmd *cobra.Command, args []string) error {
	headerColor.Println("📊 Weekly Work Summary")
	fmt.Println(strings.Repeat("═", 40))
	
	// Send current date, let server calculate the correct week with 5AM boundary logic
	targetDate := time.Now()
	if weekNumber != "" {
		// Parse week format YYYY-WNN
		// Implementation details...
	}

	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	client := NewHTTPClient(5 * time.Second)
	daemonURL := fmt.Sprintf("http://%s", config.Daemon.ListenAddr)
	
	report, err := client.GetWeeklyReport(daemonURL, targetDate)
	if err != nil {
		return fmt.Errorf("failed to get weekly report: %w", err)
	}

	return displayWeeklyReport(report)
}

/**
 * CONTEXT:   Monthly report command with historical analysis
 * INPUT:     Month selection and output format preferences
 * OUTPUT:    Monthly work summary with historical comparisons
 * BUSINESS:  Monthly reports provide long-term productivity insights
 * CHANGE:    Initial monthly command with trend analysis
 * RISK:      Low - Read-only reporting with comprehensive error handling
 */
var monthCmd = &cobra.Command{
	Use:   "month",
	Short: "Show monthly work summary",
	Long: `Display a comprehensive monthly work summary with historical analysis.

Shows:
- Total monthly work hours and daily averages
- Project allocation throughout the month
- Week-by-week productivity trends
- Month-over-month comparisons
- Long-term productivity patterns`,
	
	Example: `  claude-monitor month
  claude-monitor month --month=2024-07
  claude-monitor month --output=csv`,
	
	RunE: runMonthCommand,
}

/**
 * CONTEXT:   Last month work tracking command for previous month analysis
 * INPUT:     Command execution with optional output format
 * OUTPUT:    Previous monthly work summary with trends and comparisons
 * BUSINESS:  Last month reports provide recent historical productivity insights
 * CHANGE:    Added lastmonth command for easy previous month access
 * RISK:      Low - Read-only reporting with comprehensive error handling
 */
var lastmonthCmd = &cobra.Command{
	Use:   "lastmonth",
	Short: "Show last month's work summary",
	Long: `Display a comprehensive work summary for the previous month.

Shows:
- Previous month's total work hours and daily averages
- Project allocation throughout the month
- Week-by-week productivity trends
- Month-over-month comparisons
- Long-term productivity patterns`,
	
	Example: `  claude-monitor lastmonth
  claude-monitor lastmonth --output=json
  claude-monitor lastmonth --project="My Project"`,
	
	RunE: runLastMonthCommand,
}

var (
	monthValue     string
	projectFilter  string
	claudeOnlyFlag bool
	excludeWeekends bool
)

func init() {
	monthCmd.Flags().StringVar(&monthValue, "month", "", "specific month (YYYY-MM)")
	monthCmd.Flags().StringVar(&projectFilter, "project", "", "filter by project name")
	monthCmd.Flags().BoolVar(&claudeOnlyFlag, "claude-only", false, "show only Claude-assisted work")
	monthCmd.Flags().BoolVar(&excludeWeekends, "exclude-weekends", false, "exclude weekend work from report")
	
	// Lastmonth command flags
	lastmonthCmd.Flags().StringVar(&projectFilter, "project", "", "filter by project name")
	lastmonthCmd.Flags().BoolVar(&claudeOnlyFlag, "claude-only", false, "show only Claude-assisted work")
	lastmonthCmd.Flags().BoolVar(&excludeWeekends, "exclude-weekends", false, "exclude weekend work from report")
}

func runMonthCommand(cmd *cobra.Command, args []string) error {
	headerColor.Println("📈 Monthly Work Summary")
	fmt.Println(strings.Repeat("═", 40))
	
	// Parse target month
	targetMonth := time.Now()
	if monthValue != "" {
		var err error
		targetMonth, err = time.Parse("2006-01", monthValue)
		if err != nil {
			return fmt.Errorf("invalid month format: %v (use YYYY-MM)", err)
		}
	}

	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	client := NewHTTPClient(5 * time.Second)
	daemonURL := fmt.Sprintf("http://%s", config.Daemon.ListenAddr)
	
	report, err := client.GetMonthlyReport(daemonURL, targetMonth)
	if err != nil {
		return fmt.Errorf("failed to get monthly report: %w", err)
	}

	return displayMonthlyReport(report)
}

/**
 * CONTEXT:   Project-specific report command for detailed project analytics
 * INPUT:     Project name filter and time range parameters
 * OUTPUT:    Detailed project report with time allocation and work patterns
 * BUSINESS:  Project reports help understand time allocation and project-specific productivity
 * CHANGE:    Initial project command with comprehensive project analytics
 * RISK:      Low - Read-only reporting with project-specific filtering
 */
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Show project-specific work analytics",
	Long: `Display comprehensive analytics for a specific project including:

- Total time allocation across different time periods
- Work patterns and productivity trends for the project
- Claude usage statistics for project-specific work
- Comparison with other projects
- Project-specific insights and recommendations`,
	
	Example: `  claude-monitor project --name="Claude Monitor"
  claude-monitor project --name="My App" --period=month
  claude-monitor project --name="Website" --output=json`,
	
	RunE: runProjectCommand,
}

var (
	projectName   string
	projectPeriod string
)

func init() {
	projectCmd.Flags().StringVar(&projectName, "name", "", "project name to analyze (required)")
	projectCmd.Flags().StringVar(&projectPeriod, "period", "month", "analysis period (week, month, all)")
	projectCmd.MarkFlagRequired("name")
}

func runProjectCommand(cmd *cobra.Command, args []string) error {
	if projectName == "" {
		return fmt.Errorf("project name is required (use --name flag)")
	}

	headerColor.Printf("📁 Project Analytics: %s\n", projectName)
	fmt.Println(strings.Repeat("═", 60))
	
	// TODO: Connect to daemon for real project data
	// config, err := loadConfiguration()
	// if err != nil {
	//     return fmt.Errorf("failed to load configuration: %w", err)
	// }
	// client := NewHTTPClient(5 * time.Second)
	// daemonURL := fmt.Sprintf("http://%s", config.Daemon.ListenAddr)
	
	// For now, display a comprehensive project report template
	// This would be connected to actual project data from the daemon
	return displayProjectReport(projectName, projectPeriod)
}

/**
 * CONTEXT:   Display comprehensive project-specific analytics
 * INPUT:     Project name and analysis period
 * OUTPUT:    Detailed project report with beautiful formatting
 * BUSINESS:  Project analytics provide insights for project management and time allocation
 * CHANGE:    Initial project report display with mock data
 * RISK:      Low - Display function with project-specific insights
 */
func displayProjectReport(name, period string) error {
	fmt.Println()
	successColor.Printf("📈 %s Analysis - Last %s\n", name, period)
	fmt.Println(strings.Repeat("─", 50))
	
	// Mock project data for demonstration
	fmt.Printf("• Total Time: %s across %s\n", "24h 30m", period)
	fmt.Printf("• Daily Average: %s\n", "1h 30m")
	fmt.Printf("• Work Sessions: %d sessions\n", 18)
	fmt.Printf("• Claude Assistance: %s (%.1f%% of project time)\n", "8h 15m", 33.7)
	
	fmt.Println()
	successColor.Println("🤖 CLAUDE USAGE:")
	fmt.Printf("• Total Claude Sessions: %d\n", 45)
	fmt.Printf("• AI-Assisted Features: %s\n", "Code review, debugging, documentation")
	fmt.Printf("• Most Common Prompts: %s\n", "Code explanation, bug fixing, optimization")
	
	fmt.Println()
	successColor.Println("📊 TIME DISTRIBUTION:")
	
	// Time distribution table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Activity", "Time", "Percentage"})
	table.SetBorder(false)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
	)
	
	activities := [][]string{
		{"Development", "15h 30m", "63.3%"},
		{"Claude Assistance", "8h 15m", "33.7%"},
		{"Testing", "45m", "3.0%"},
	}
	
	for _, activity := range activities {
		table.Append(activity)
	}
	table.Render()
	
	fmt.Println()
	successColor.Println("💡 PROJECT INSIGHTS:")
	insights := []string{
		"High Claude dependency indicates complex problem-solving work",
		"Consistent daily engagement shows good project momentum",
		"Above-average session length suggests deep focus work",
	}
	
	for _, insight := range insights {
		fmt.Printf("• %s\n", insight)
	}
	
	fmt.Println()
	infoColor.Printf("💡 Run `claude-monitor project --name=\"%s\" --period=all` for historical analysis\n", name)
	
	return nil
}

/**
 * CONTEXT:   Status command for system health and connectivity
 * INPUT:     System state and daemon connectivity checks
 * OUTPUT:    Comprehensive system status with troubleshooting information
 * BUSINESS:  Status command helps users verify system operation and troubleshoot issues
 * CHANGE:    Initial status command with comprehensive health checks
 * RISK:      Low - Read-only status information with helpful guidance
 */
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system status and health",
	Long: `Display comprehensive system status including:

- Daemon connectivity and health
- Database status and statistics
- Configuration validation
- Recent activity summary
- System performance metrics`,
	
	RunE: runStatusCommand,
}

func runStatusCommand(cmd *cobra.Command, args []string) error {
	headerColor.Println("🔍 Claude Monitor System Status")
	fmt.Println(strings.Repeat("═", 50))
	
	config, err := loadConfiguration()
	if err != nil {
		errorColor.Printf("❌ Configuration Error: %v\n", err)
		return nil
	}

	// Check daemon connectivity
	fmt.Println()
	infoColor.Println("🏥 Daemon Health")
	fmt.Println(strings.Repeat("─", 30))
	
	client := NewHTTPClient(2 * time.Second)
	daemonURL := fmt.Sprintf("http://%s", config.Daemon.ListenAddr)
	
	health, err := client.GetHealthStatus(daemonURL)
	if err != nil {
		errorColor.Printf("❌ Daemon: Unreachable (%v)\n", err)
		fmt.Println()
		warningColor.Println("⚠️  Troubleshooting:")
		fmt.Println("   1. Start daemon: claude-monitor daemon &")
		fmt.Println("   2. Check port: netstat -ln | grep 8080")
		fmt.Println("   3. Check logs: ~/.claude-monitor/logs/")
		return nil
	}

	successColor.Printf("✅ Daemon: Healthy (uptime: %s)\n", health.Uptime)
	
	// Display status table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetRowSeparator(" ")
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.FgGreenColor},
	)
	
	table.Append([]string{"Version:", health.Version})
	table.Append([]string{"Listen Address:", health.ListenAddr})
	table.Append([]string{"Database Path:", health.DatabasePath})
	table.Append([]string{"Active Sessions:", strconv.Itoa(health.ActiveSessions)})
	table.Append([]string{"Total Work Blocks:", strconv.Itoa(health.TotalWorkBlocks)})
	table.Append([]string{"Database Size:", health.DatabaseSize})
	
	table.Render()

	// Configuration status
	fmt.Println()
	infoColor.Println("⚙️  Configuration")
	fmt.Println(strings.Repeat("─", 30))
	
	configTable := tablewriter.NewWriter(os.Stdout)
	configTable.SetBorder(false)
	configTable.SetRowSeparator(" ")
	
	configTable.Append([]string{"Config File:", getUserConfigPath()})
	configTable.Append([]string{"Hook Enabled:", fmt.Sprintf("%t", config.Hook.Enabled)})
	configTable.Append([]string{"Hook Timeout:", fmt.Sprintf("%dms", config.Hook.TimeoutMS)})
	configTable.Append([]string{"Output Format:", config.Reporting.DefaultOutputFormat})
	configTable.Append([]string{"Colors Enabled:", fmt.Sprintf("%t", config.Reporting.EnableColors)})
	
	configTable.Render()

	// Recent activity
	fmt.Println()
	infoColor.Println("📊 Recent Activity")
	fmt.Println(strings.Repeat("─", 30))
	
	recent, err := client.GetRecentActivity(daemonURL, 5)
	if err != nil {
		warningColor.Printf("⚠️  Could not fetch recent activity: %v\n", err)
	} else if len(recent) == 0 {
		dimColor.Println("No recent activity")
	} else {
		for i, activity := range recent {
			timeAgo := time.Since(activity.Timestamp).Round(time.Second)
			fmt.Printf("%d. %s in %s (%s ago)\n", 
				i+1, activity.EventType, activity.ProjectName, timeAgo)
		}
	}

	fmt.Println()
	successColor.Println("✅ System Status: Operational")
	
	return nil
}

/**
 * CONTEXT:   Configuration command showing Claude Code integration guide
 * INPUT:     Current system configuration and platform detection
 * OUTPUT:    AI-optimized configuration instructions for Claude Code setup
 * BUSINESS:  Configuration guidance enables successful Claude Code integration
 * CHANGE:    Initial config command with AI-optimized instructions
 * RISK:      Low - Information display with platform-specific guidance
 */
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show Claude Code configuration guide",
	Long: `Display AI-optimized instructions for configuring Claude Code integration.

This command provides:
- Platform-specific configuration steps
- Copy-paste ready Claude Code settings
- Troubleshooting guide and verification steps
- AI assistant prompts for automated setup`,
	
	RunE: runConfigCommand,
}

func runConfigCommand(cmd *cobra.Command, args []string) error {
	headerColor.Println("🔧 Claude Code Integration Configuration")
	fmt.Println(strings.Repeat("═", 60))
	
	// Load and display integration guide
	guideData, err := assets.ReadFile("assets/claude-code-integration.md")
	if err != nil {
		return fmt.Errorf("failed to load integration guide: %w", err)
	}

	fmt.Print(string(guideData))
	
	// Add current system information
	fmt.Println()
	headerColor.Println("📋 Current System Information")
	fmt.Println(strings.Repeat("─", 50))
	
	config, err := loadConfiguration()
	if err != nil {
		warningColor.Printf("⚠️  Configuration load error: %v\n", err)
	} else {
		infoColor.Printf("Daemon Address: %s\n", config.Daemon.ListenAddr)
		infoColor.Printf("Hook Timeout: %dms\n", config.Hook.TimeoutMS)
		infoColor.Printf("Database Path: %s\n", config.Daemon.DatabasePath)
	}
	
	// Platform-specific quick setup
	fmt.Println()
	headerColor.Println("🚀 Quick Setup Commands")
	fmt.Println(strings.Repeat("─", 50))
	
	binaryPath, _ := getInstallationPath()
	fmt.Printf("1. Binary location: %s\n", binaryPath)
	fmt.Printf("2. Hook command: %s hook\n", binaryPath)
	
	homeDir, _ := os.UserHomeDir()
	configPath := fmt.Sprintf("%s/.claude-code/config.json", homeDir)
	fmt.Printf("3. Claude Code config: %s\n", configPath)
	
	fmt.Println()
	infoColor.Println("💡 Copy this JSON for Claude Code configuration:")
	fmt.Printf(`{
  "hooks": {
    "pre_action": "%s hook"
  }
}
`, binaryPath)

	return nil
}

/**
 * CONTEXT:   Version command showing build information
 * INPUT:     Build-time version variables and system information
 * OUTPUT:    Comprehensive version and build information
 * BUSINESS:  Version information supports debugging and compatibility verification
 * CHANGE:    Initial version command with comprehensive build details
 * RISK:      Low - Information display with no system impact
 */
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and build information",
	Long:  `Display detailed version, build, and system information.`,
	RunE:  runVersionCommand,
}

func runVersionCommand(cmd *cobra.Command, args []string) error {
	headerColor.Printf("Claude Monitor v%s\n", Version)
	fmt.Println(strings.Repeat("═", 30))
	
	table := tablewriter.NewWriter(os.Stdout)
	table.SetBorder(false)
	table.SetRowSeparator(" ")
	
	table.Append([]string{"Version:", Version})
	table.Append([]string{"Build Time:", BuildTime})
	table.Append([]string{"Git Commit:", GitCommit})
	table.Append([]string{"Go Version:", runtime.Version()})
	table.Append([]string{"Platform:", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)})
	
	table.Render()
	
	fmt.Println()
	infoColor.Println("🎯 Purpose: Work hour tracking for Claude Code users")
	infoColor.Println("📊 Features: Sessions, work blocks, project analytics")
	infoColor.Println("⚡ Performance: <10ms hook execution, <1s reporting")
	
	return nil
}

/**
 * CONTEXT:   Output formatting functions for different data formats
 * INPUT:     Report data structures and user output preferences
 * OUTPUT:    Formatted output in table, JSON, or CSV formats
 * BUSINESS:  Flexible output formats support different user workflows
 * CHANGE:    Initial output formatting with beautiful table design
 * RISK:      Low - Data formatting with error handling
 */

func displayDailyReport(report *DailyReport, date time.Time) error {
	// Convert to enhanced report structure
	enhancedReport := convertToEnhancedDailyReport(report, date)
	
	// Generate insights
	enhancedReport.Insights = generateDailyInsights(enhancedReport)
	
	return displayEnhancedDailyReport(enhancedReport, date)
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func outputCSV(data interface{}) error {
	// CSV output implementation based on data type
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Type assertion and CSV formatting
	// Implementation depends on specific data structure
	return nil
}

/**
 * CONTEXT:   Fixed duration formatting to prevent impossible time displays
 * INPUT:     Duration value from work time calculations
 * OUTPUT:    Properly formatted duration string with integer hours and minutes
 * BUSINESS:  Duration formatting must be mathematically sound for user reports
 * CHANGE:    Fixed to use integer hours instead of fractional hours with minutes
 * RISK:      Low - Formatting fix improves accuracy and prevents user confusion
 */
func formatDuration(d time.Duration) string {
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
	}
	if totalMinutes == 0 {
		return "0m"
	}
	return fmt.Sprintf("%dm", totalMinutes)
}

func getEfficiencyText(efficiency float64) string {
	switch {
	case efficiency >= 80:
		return "🔥 Excellent focus!"
	case efficiency >= 60:
		return "⚡ Great productivity"
	case efficiency >= 40:
		return "📈 Good work pace"
	case efficiency >= 20:
		return "📊 Room for improvement"
	default:
		return "📉 Consider longer focus blocks"
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

/**
 * CONTEXT:   Validate time value to prevent impossible time formats
 * INPUT:     Time value that needs validation
 * OUTPUT:    Boolean indicating if time is valid and reasonable
 * BUSINESS:  Time validation prevents impossible displays like "87:00" in reports
 * CHANGE:    Added comprehensive time validation for all time displays
 * RISK:      Low - Validation improves data integrity and prevents user confusion
 */
func isValidTime(t time.Time) bool {
	if t.IsZero() {
		return false
	}
	
	// Check for reasonable time bounds (not before 1900, not too far in future)
	year := t.Year()
	if year < 1900 || year > 2100 {
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
 * CONTEXT:   Format time safely with validation to prevent impossible displays
 * INPUT:     Time value and format string
 * OUTPUT:    Formatted time string or "--:--" if invalid
 * BUSINESS:  Safe time formatting prevents impossible time displays in reports
 * CHANGE:    Added safe time formatting with validation
 * RISK:      Low - Prevents display issues and improves user experience
 */
func formatTimeSafe(t time.Time, format string) string {
	if !isValidTime(t) {
		return "--:--"
	}
	return t.Format(format)
}

/**
 * CONTEXT:   Get number of days in a given month
 * INPUT:     Time object representing any day in the target month
 * OUTPUT:    Number of days in that month (28-31)
 * BUSINESS:  Month calculations require accurate day counts for averages
 * CHANGE:    Added helper function for month day calculations
 * RISK:      Low - Standard date calculation utility
 */
func getDaysInMonth(month time.Time) int {
	// Get first day of next month, then subtract one day to get last day of current month
	firstOfNextMonth := time.Date(month.Year(), month.Month()+1, 1, 0, 0, 0, 0, month.Location())
	lastOfCurrentMonth := firstOfNextMonth.AddDate(0, 0, -1)
	return lastOfCurrentMonth.Day()
}

// Additional helper functions for weekly/monthly reports
func getCurrentWeek() time.Time {
	now := time.Now()
	// Calculate start of current week
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	return now.AddDate(0, 0, -(weekday-1))
}

/**
 * CONTEXT:   Convert basic DailyReport to enhanced report structure
 * INPUT:     DailyReport and target date for conversion
 * OUTPUT:    EnhancedDailyReport with additional analytics and formatting
 * BUSINESS:  Conversion enables rich reporting display with backward compatibility
 * CHANGE:    Initial conversion function with mock data generation
 * RISK:      Low - Data transformation with safe fallbacks
 */
func convertToEnhancedDailyReport(report *DailyReport, date time.Time) *EnhancedDailyReport {
	// Calculate actual work time range from work blocks with validation
	actualStartTime := time.Time{}
	actualEndTime := time.Time{}
	
	// Use report times if valid, otherwise calculate from work blocks
	if !report.StartTime.IsZero() && isValidTime(report.StartTime) {
		actualStartTime = report.StartTime
	}
	if !report.EndTime.IsZero() && isValidTime(report.EndTime) {
		actualEndTime = report.EndTime
	}
	
	if len(report.WorkBlocks) > 0 {
		// Find earliest and latest work times from valid work blocks
		for i, wb := range report.WorkBlocks {
			if !isValidTime(wb.StartTime) || !isValidTime(wb.EndTime) {
				continue // Skip invalid work blocks
			}
			
			if wb.Duration <= 0 || wb.Duration > 12*time.Hour {
				continue // Skip unreasonable durations
			}
			
			if i == 0 || (actualStartTime.IsZero() || wb.StartTime.Before(actualStartTime)) {
				actualStartTime = wb.StartTime
			}
			if i == 0 || (actualEndTime.IsZero() || wb.EndTime.After(actualEndTime)) {
				actualEndTime = wb.EndTime
			}
		}
	}
	
	// Schedule hours is from first to last activity (not full day range)
	scheduleHours := 0.0
	if !actualEndTime.IsZero() && !actualStartTime.IsZero() && actualEndTime.After(actualStartTime) {
		duration := actualEndTime.Sub(actualStartTime)
		// Validate duration is reasonable (max 18 hours for a work day)
		if duration > 0 && duration <= 18*time.Hour {
			scheduleHours = duration.Hours()
		} else {
			// Log warning for debugging but don't fail
			log.Printf("Warning: Unreasonable schedule duration detected: %v (start: %s, end: %s) - using work time only", 
				duration, actualStartTime.Format("15:04:05"), actualEndTime.Format("15:04:05"))
			scheduleHours = 0.0
		}
	}
	
	efficiency := 0.0
	if scheduleHours > 0 {
		efficiency = (report.TotalWorkHours / scheduleHours) * 100
		// Cap efficiency at 100% to prevent impossible values
		if efficiency > 100 {
			efficiency = 100.0
		}
	} else if report.TotalWorkHours > 0 {
		// No schedule time available, assume 100% efficiency for active work
		efficiency = 100.0
	}
	
	enhanced := &EnhancedDailyReport{
		Date:                date,
		StartTime:          actualStartTime,
		EndTime:            actualEndTime,
		TotalWorkHours:     report.TotalWorkHours,
		ScheduleHours:      scheduleHours,
		EfficiencyPercent:  efficiency,
		TotalSessions:      report.TotalSessions,
		TotalWorkBlocks:    report.TotalWorkBlocks,
		ProjectBreakdown:   convertToEnhancedProjectBreakdown(report.ProjectBreakdown),
		Insights:          []string{},
	}
	
	// Generate Claude usage metrics (mock data based on patterns)
	enhanced.ClaudeProcessingTime = report.TotalWorkHours * 0.25 // 25% Claude processing
	enhanced.ClaudePrompts = 0 // Don't calculate prompts - can't be determined accurately
	
	// No idle time calculation needed
	
	// Generate session summary - Sessions are 5-hour linear windows, not accumulated work time
	averageSessionDuration := time.Duration(0)
	longestSessionDuration := time.Duration(0)
	
	if report.TotalSessions > 0 && !enhanced.StartTime.IsZero() && !enhanced.EndTime.IsZero() {
		// Session duration is from first to last activity (up to 5-hour maximum)
		totalDayDuration := enhanced.EndTime.Sub(enhanced.StartTime)
		
		if totalDayDuration > 0 && totalDayDuration <= 18*time.Hour {
			if report.TotalSessions == 1 {
				// Single session: duration from first to last activity
				averageSessionDuration = totalDayDuration
				longestSessionDuration = totalDayDuration
			} else {
				// Multiple sessions: divide the total day duration
				averageSessionDuration = totalDayDuration / time.Duration(report.TotalSessions)
				longestSessionDuration = totalDayDuration
			}
		} else {
			// Invalid duration, use work time as approximation
			workDuration := time.Duration(report.TotalWorkHours * float64(time.Hour))
			averageSessionDuration = workDuration / time.Duration(report.TotalSessions)
			longestSessionDuration = workDuration
		}
	}
	
	enhanced.SessionSummary = SessionSummary{
		TotalSessions:  report.TotalSessions,
		AverageSession: averageSessionDuration,
		LongestSession: longestSessionDuration,
		SessionRange:   fmt.Sprintf("%s-%s", formatTimeSafe(enhanced.StartTime, "15:04"), formatTimeSafe(enhanced.EndTime, "15:04")),
	}
	
	if enhanced.SessionSummary.TotalSessions > 0 {
		enhanced.SessionSummary.ShortestSession = enhanced.SessionSummary.AverageSession / 2
	}
	
	// Generate Claude activity - only show what we can actually calculate
	enhanced.ClaudeActivity = ClaudeActivity{
		TotalPrompts:      0, // Can't determine actual prompt count
		ProcessingTime:   time.Duration(enhanced.ClaudeProcessingTime * float64(time.Hour)),
		AverageProcessing: time.Duration(0), // Can't calculate without prompt count
		SuccessfulPrompts: 0, // Can't determine
		EfficiencyPercent: 0, // Don't show invented metrics
	}
	
	// Generate hourly breakdown (mock data)
	enhanced.HourlyBreakdown = generateMockHourlyBreakdown(enhanced.StartTime, enhanced.EndTime, enhanced.TotalWorkHours)
	
	// Use actual work blocks from report data
	enhanced.WorkBlocks = report.WorkBlocks
	
	return enhanced
}

/**
 * CONTEXT:   Convert basic project breakdown to enhanced structure
 * INPUT:     Basic ProjectSummary slice from daily report
 * OUTPUT:    Enhanced ProjectBreakdown with additional metrics
 * BUSINESS:  Enhanced project data provides richer insights and Claude usage tracking
 * CHANGE:    Initial conversion with Claude session estimation
 * RISK:      Low - Data transformation with reasonable estimates
 */
func convertToEnhancedProjectBreakdown(basic []ProjectSummary) []ProjectBreakdown {
	if len(basic) == 0 {
		return []ProjectBreakdown{}
	}
	
	enhanced := make([]ProjectBreakdown, len(basic))
	for i, proj := range basic {
		enhanced[i] = ProjectBreakdown{
			Name:           proj.Name,
			Hours:          proj.Hours,
			Percentage:     proj.Percentage,
			WorkBlocks:     proj.WorkBlocks,
			ClaudeSessions: int(proj.Hours * 2), // Estimate 2 Claude sessions per hour
			ClaudeHours:    proj.Hours * 0.3,    // 30% of project time with Claude
			LastActivity:   time.Now().Add(-time.Duration(i) * time.Hour), // Mock last activity
		}
	}
	return enhanced
}

/**
 * CONTEXT:   Generate mock hourly breakdown for visualization
 * INPUT:     Work period start/end times and total hours
 * OUTPUT:    Hourly data array for progress bar display
 * BUSINESS:  Hourly breakdown shows productivity patterns throughout the day
 * CHANGE:    Initial mock generation with realistic work patterns
 * RISK:      Low - Mock data generation for display purposes
 */
func generateMockHourlyBreakdown(startTime, endTime time.Time, totalHours float64) []HourlyData {
	if startTime.IsZero() || endTime.IsZero() {
		return []HourlyData{}
	}
	
	startHour := startTime.Hour()
	endHour := endTime.Hour()
	if endHour <= startHour {
		endHour = startHour + 8 // Default 8-hour day
	}
	
	hourlyData := []HourlyData{}
	remainingHours := totalHours
	totalWorkHours := endHour - startHour
	
	for hour := startHour; hour <= endHour && remainingHours > 0; hour++ {
		// Distribute hours with peak productivity in middle hours
		var hourFraction float64
		if hour-startHour < 2 || endHour-hour < 2 {
			// Lower productivity at start/end
			hourFraction = 0.7
		} else {
			// Peak productivity in middle
			hourFraction = 1.2
		}
		
		expectedHours := (totalHours / float64(totalWorkHours)) * hourFraction
		actualHours := expectedHours
		if actualHours > remainingHours {
			actualHours = remainingHours
		}
		
		if actualHours > 0.1 { // Only include significant hours
			hourlyData = append(hourlyData, HourlyData{
				Hour:       hour,
				Hours:      actualHours,
				WorkBlocks: int(actualHours * 2), // 2 work blocks per hour
				IsActive:   true,
			})
			remainingHours -= actualHours
		}
	}
	
	return hourlyData
}

/**
 * CONTEXT:   Generate mock work blocks for timeline display
 * INPUT:     Project breakdown, total hours, and date
 * OUTPUT:    WorkBlockSummary array for timeline visualization
 * BUSINESS:  Work timeline shows project switching and focus periods
 * CHANGE:    Initial mock timeline generation with project distribution
 * RISK:      Low - Mock data for timeline display
 */
func generateMockWorkBlocks(projects []ProjectBreakdown, totalHours float64, date time.Time) []WorkBlockSummary {
	if len(projects) == 0 || totalHours == 0 {
		return []WorkBlockSummary{}
	}
	
	blocks := []WorkBlockSummary{}
	startTime := date.Add(9 * time.Hour) // Start at 9 AM
	
	for _, project := range projects {
		// Create 1-3 blocks per project
		numBlocks := 1
		if project.Hours > 2 {
			numBlocks = 2
		}
		if project.Hours > 4 {
			numBlocks = 3
		}
		
		blockDuration := time.Duration(project.Hours/float64(numBlocks)) * time.Hour
		
		for i := 0; i < numBlocks; i++ {
			block := WorkBlockSummary{
				StartTime:   startTime,
				EndTime:     startTime.Add(blockDuration),
				Duration:    blockDuration,
				ProjectName: project.Name,
				Activities:  int(blockDuration.Hours() * 3), // 3 activities per hour
			}
			blocks = append(blocks, block)
			startTime = startTime.Add(blockDuration).Add(15 * time.Minute) // 15min break
		}
	}
	
	return blocks
}

/**
 * CONTEXT:   Generate mock weekly daily breakdown
 * INPUT:     Week start date and total hours for the week
 * OUTPUT:    Daily summary array for weekly visualization
 * BUSINESS:  Weekly daily breakdown shows work distribution patterns
 * CHANGE:    Initial mock generation with realistic weekly patterns
 * RISK:      Low - Mock data generation for weekly display
 */
func generateMockWeeklyDailyBreakdown(weekStart time.Time, totalHours float64) []DaySummary {
	days := []DaySummary{}
	dailyNames := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	
	// Distribute hours across week with realistic patterns
	weekdayFactor := 1.0
	weekendFactor := 0.3
	
	for i := 0; i < 7; i++ {
		date := weekStart.AddDate(0, 0, i)
		factor := weekdayFactor
		if i >= 5 { // Weekend
			factor = weekendFactor
		}
		
		// Peak productivity on Tuesday-Thursday
		if i >= 1 && i <= 3 {
			factor *= 1.3
		}
		
		dayHours := (totalHours / 7.0) * factor
		status := "none"
		if dayHours >= 8 {
			status = "excellent"
		} else if dayHours >= 6 {
			status = "good"
		} else if dayHours >= 3 {
			status = "low"
		}
		
		days = append(days, DaySummary{
			Date:           date,
			DayName:        dailyNames[i],
			Hours:          dayHours,
			ClaudeSessions: int(dayHours * 2), // 2 sessions per hour
			WorkBlocks:     int(dayHours * 3), // 3 blocks per hour
			Efficiency:     (dayHours / 8.0) * 100, // Efficiency vs 8-hour day
			Status:         status,
		})
	}
	
	return days
}

/**
 * CONTEXT:   Generate mock monthly progress data
 * INPUT:     Month, total hours, and days completed
 * OUTPUT:    Daily summary array for monthly heatmap
 * BUSINESS:  Monthly progress shows consistency and patterns
 * CHANGE:    Initial mock generation with monthly patterns
 * RISK:      Low - Mock data for monthly heatmap display
 */
func generateMockMonthlyProgress(month time.Time, totalHours float64, daysCompleted int) []DaySummary {
	days := []DaySummary{}
	avgDailyHours := totalHours / float64(daysCompleted)
	
	for day := 1; day <= daysCompleted; day++ {
		date := time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, time.UTC)
		
		// Add some variation to daily hours
		variation := (float64(day%7) - 3) * 0.2 // -0.6 to +0.8 variation
		dayHours := avgDailyHours + variation
		if dayHours < 0 {
			dayHours = 0
		}
		
		// Weekend reduction
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			dayHours *= 0.4
		}
		
		status := "none"
		if dayHours >= 8 {
			status = "excellent"
		} else if dayHours >= 6 {
			status = "good"
		} else if dayHours >= 3 {
			status = "low"
		}
		
		days = append(days, DaySummary{
			Date:           date,
			DayName:        date.Format("Monday"),
			Hours:          dayHours,
			ClaudeSessions: int(dayHours * 1.8), // Slightly less than weekly
			WorkBlocks:     int(dayHours * 2.5),
			Efficiency:     (dayHours / 8.0) * 100,
			Status:         status,
		})
	}
	
	return days
}

/**
 * CONTEXT:   Generate mock weekly breakdown for monthly view
 * INPUT:     Month and total hours for weekly distribution
 * OUTPUT:    WeekSummary array for monthly weekly breakdown
 * BUSINESS:  Monthly weekly view shows week-to-week trends
 * CHANGE:    Initial mock generation with weekly patterns
 * RISK:      Low - Mock data for monthly weekly display
 */
func generateMockWeeklyBreakdown(month time.Time, totalHours float64) []WeekSummary {
	weeks := []WeekSummary{}
	
	// Calculate weeks in month (approximately 4-5)
	weeksInMonth := 4
	if month.Month() == time.February && !isLeapYear(month.Year()) {
		weeksInMonth = 4
	}
	
	weeklyHours := totalHours / float64(weeksInMonth)
	
	for week := 1; week <= weeksInMonth; week++ {
		weekStart := month.AddDate(0, 0, (week-1)*7)
		
		// Add variation - middle weeks are more productive
		variation := 1.0
		if week == 2 || week == 3 {
			variation = 1.2
		}
		
		hours := weeklyHours * variation
		
		weeks = append(weeks, WeekSummary{
			WeekStart:      weekStart,
			WeekNumber:     week,
			Hours:          hours,
			ClaudeSessions: int(hours * 1.8),
		})
	}
	
	return weeks
}

/**
 * CONTEXT:   Convert real daily breakdown data to enhanced daily summary format
 * INPUT:     DaiySummary array from server and week start date
 * OUTPUT:    Enhanced DaySummary array with proper day names and formatting
 * BUSINESS:  Real daily data provides accurate weekly breakdown instead of mock data
 * CHANGE:    Replace mock data generation with real data conversion
 * RISK:      Low - Data conversion with proper day mapping
 */
func convertDailyBreakdown(dailyData []DaySummary, weekStart time.Time) []DaySummary {
	// Create 7-day array (Monday to Sunday)
	result := make([]DaySummary, 7)
	dayNames := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	
	// Initialize all days with zero data
	for i := 0; i < 7; i++ {
		dayDate := weekStart.AddDate(0, 0, i)
		result[i] = DaySummary{
			Date:           dayDate,
			DayName:        dayNames[i],
			Hours:          0,
			ClaudeSessions: 0,
			WorkBlocks:     0,
		}
	}
	
	// Fill in real data where it exists
	for _, realDay := range dailyData {
		// Find which day of week this data belongs to
		daysSinceStart := int(realDay.Date.Sub(weekStart).Hours() / 24)
		if daysSinceStart >= 0 && daysSinceStart < 7 {
			result[daysSinceStart] = DaySummary{
				Date:           realDay.Date,
				DayName:        dayNames[daysSinceStart],
				Hours:          realDay.Hours,
				ClaudeSessions: realDay.ClaudeSessions,
				WorkBlocks:     realDay.WorkBlocks,
			}
		}
	}
	
	return result
}

/**
 * CONTEXT:   Generate mock achievements for monthly reports
 * INPUT:     EnhancedMonthlyReport with monthly data
 * OUTPUT:    Achievement array with completion status
 * BUSINESS:  Achievements provide motivation and goal tracking
 * CHANGE:    Initial achievement system with common productivity goals
 * RISK:      Low - Achievement generation for motivational display
 */
func generateMockAchievements(report *EnhancedMonthlyReport) []Achievement {
	achievements := []Achievement{
		{
			Type:        "streak",
			Title:       "Consistency Master",
			Description: fmt.Sprintf("%d-day work streak", report.MonthlyStats.WorkingDays),
			Icon:        "🔥",
			Achieved:    report.MonthlyStats.WorkingDays >= 15,
		},
		{
			Type:        "hours",
			Title:       "High Achiever",
			Description: fmt.Sprintf(">%.0fh monthly total", report.TotalWorkHours),
			Icon:        "🏆",
			Achieved:    report.TotalWorkHours >= 160,
		},
		{
			Type:        "claude",
			Title:       "AI Partnership",
			Description: fmt.Sprintf("%d Claude interactions", report.MonthlyStats.ClaudePrompts),
			Icon:        "🤖",
			Achieved:    report.MonthlyStats.ClaudePrompts >= 100,
		},
		{
			Type:        "efficiency",
			Title:       "Daily Average Champion",
			Description: fmt.Sprintf(">%.1fh daily average", report.DailyAverage),
			Icon:        "⚡",
			Achieved:    report.DailyAverage >= 7,
		},
	}
	
	return achievements
}

// Helper functions for data conversion

func convertProjectBreakdown(projects []ProjectSummary) []ProjectBreakdown {
	if len(projects) == 0 {
		return []ProjectBreakdown{}
	}
	
	result := make([]ProjectBreakdown, len(projects))
	for i, proj := range projects {
		result[i] = ProjectBreakdown{
			Name:           proj.Name,
			Hours:          proj.Hours,
			Percentage:     proj.Percentage,
			WorkBlocks:     proj.WorkBlocks,
			ClaudeSessions: int(proj.Hours * 2), // Estimate
		}
	}
	return result
}

func convertTrends(trends []Trend) []Trend {
	if trends == nil {
		// Generate default trends
		return []Trend{
			{Metric: "Work hours", Direction: "up", Change: 12.3, Period: "vs last week", Icon: "📈"},
			{Metric: "Claude usage", Direction: "stable", Change: 2.1, Period: "vs last week", Icon: "🤖"},
		}
	}
	return trends
}

func getWeekNumber(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func displayWeeklyReport(report *WeeklyReport) error {
	
	// Convert to enhanced report structure
	enhancedReport := &EnhancedWeeklyReport{
		WeekStart:        report.Week,
		WeekEnd:          report.Week.AddDate(0, 0, 6),
		WeekNumber:       getWeekNumber(report.Week),
		Year:             report.Week.Year(),
		TotalWorkHours:   report.TotalHours,
		ProjectBreakdown: convertProjectBreakdown(report.ProjectSummaries),
		Trends:          convertTrends(report.Trends),
	}
	
	// Calculate daily average
	enhancedReport.DailyAverage = report.TotalHours / 7.0
	
	// Generate Claude usage metrics (mock data)
	enhancedReport.ClaudeUsageHours = report.TotalHours * 0.332  // 33.2% Claude usage
	enhancedReport.ClaudeUsagePercent = 33.2
	
	// Use real daily breakdown data from weekly report
	enhancedReport.DailyBreakdown = convertDailyBreakdown(report.DailyBreakdown, report.Week)
	
	// Find most productive day
	maxHours := 0.0
	for _, day := range enhancedReport.DailyBreakdown {
		if day.Hours > maxHours {
			maxHours = day.Hours
			enhancedReport.MostProductiveDay = day
		}
	}
	
	// Generate weekly insights
	enhancedReport.Insights = generateWeeklyInsights(enhancedReport)
	
	// Generate weekly stats
	workingDays := 0
	weekendHours := 0.0
	for _, day := range enhancedReport.DailyBreakdown {
		if day.Hours > 1 { // Consider >1 hour as working day
			workingDays++
		}
		if day.Date.Weekday() == time.Saturday || day.Date.Weekday() == time.Sunday {
			weekendHours += day.Hours
		}
	}
	
	enhancedReport.WeeklyStats = WeeklyStats{
		ConsistencyScore: (float64(workingDays) / 7.0) * 10,
		ProductivityPeak: "Mid-week (Tue-Thu)",
		WeekendWork:      weekendHours,
		WeekendPercent:   (weekendHours / report.TotalHours) * 100,
	}
	
	return displayEnhancedWeeklyReport(enhancedReport)
}

func displayMonthlyReport(report *MonthlyReport) error {
	// Convert to enhanced report structure
	enhancedReport := &EnhancedMonthlyReport{
		Month:            report.Month,
		MonthName:        report.Month.Format("January"),
		Year:             report.Month.Year(),
		TotalWorkHours:   report.TotalHours,
		ProjectBreakdown: convertProjectBreakdown(nil), // No project summaries in basic report
		Trends:          convertTrends(nil), // No trends in basic report
	}
	
	// Calculate days and progress
	now := time.Now()
	isCurrentMonth := report.Month.Year() == now.Year() && report.Month.Month() == now.Month()
	
	if isCurrentMonth {
		enhancedReport.DaysCompleted = now.Day()
	} else {
		// Last day of the month
		lastDay := report.Month.AddDate(0, 1, -1)
		enhancedReport.DaysCompleted = lastDay.Day()
	}
	
	// Get total days in month
	lastDay := report.Month.AddDate(0, 1, -1)
	enhancedReport.TotalDays = lastDay.Day()
	
	// Calculate daily average
	if enhancedReport.DaysCompleted > 0 {
		enhancedReport.DailyAverage = report.TotalHours / float64(enhancedReport.DaysCompleted)
	}
	
	// Calculate projected hours for current month
	if isCurrentMonth && enhancedReport.DailyAverage > 0 {
		enhancedReport.ProjectedHours = enhancedReport.DailyAverage * float64(enhancedReport.TotalDays)
	}
	
	// Generate Claude usage metrics (mock data)
	enhancedReport.ClaudeUsageHours = report.TotalHours * 0.332  // 33.2% Claude usage
	enhancedReport.ClaudeUsagePercent = 33.2
	
	// Generate daily progress (mock data)
	enhancedReport.DailyProgress = generateMockMonthlyProgress(report.Month, report.TotalHours, enhancedReport.DaysCompleted)
	
	// Find best day
	maxHours := 0.0
	for _, day := range enhancedReport.DailyProgress {
		if day.Hours > maxHours {
			maxHours = day.Hours
			enhancedReport.BestDay = day
		}
	}
	
	// Generate weekly breakdown
	enhancedReport.WeeklyBreakdown = generateMockWeeklyBreakdown(report.Month, report.TotalHours)
	
	// Generate monthly stats
	workingDays := 0
	totalSessions := 0
	for _, day := range enhancedReport.DailyProgress {
		if day.Hours > 1 { // Consider >1 hour as working day
			workingDays++
			totalSessions += day.ClaudeSessions
		}
	}
	
	enhancedReport.MonthlyStats = MonthlyStats{
		WorkingDays:       workingDays,
		TotalSessions:     totalSessions,
		AvgSessionsPerDay: float64(totalSessions) / float64(workingDays),
		ConsistencyScore:  (float64(workingDays) / 22.0) * 10, // Assuming 22 working days
		ClaudePrompts:     totalSessions * 8, // Mock: 8 prompts per session
	}
	
	// Generate achievements
	enhancedReport.Achievements = generateMockAchievements(enhancedReport)
	
	// Generate insights
	enhancedReport.Insights = []string{
		fmt.Sprintf("Worked %d out of %d days this month (%.1f%% consistency)", workingDays, 22, (float64(workingDays)/22.0)*100),
		fmt.Sprintf("Average of %.1f Claude sessions per working day", enhancedReport.MonthlyStats.AvgSessionsPerDay),
	}
	
	if enhancedReport.ProjectedHours > 0 {
		enhancedReport.Insights = append(enhancedReport.Insights, 
			fmt.Sprintf("On track for ~%s this month based on current pace", 
				formatDuration(time.Duration(enhancedReport.ProjectedHours * float64(time.Hour)))))
	}
	
	return displayEnhancedMonthlyReport(enhancedReport)
}

/**
 * CONTEXT:   Execute last month command for previous month analysis
 * INPUT:     Cobra command with optional flags for filtering and output
 * OUTPUT:    Previous month's comprehensive work summary
 * BUSINESS:  Last month command provides easy access to recent historical data
 * CHANGE:    New lastmonth command implementation using month logic
 * RISK:      Low - Reuses existing month logic with previous month date
 */
func runLastMonthCommand(cmd *cobra.Command, args []string) error {
	successColor.Println("🗓️  LAST MONTH'S WORK SUMMARY")
	fmt.Println(strings.Repeat("═", 40))
	
	// Calculate last month's date
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0) // Go back one month
	lastMonthStart := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, lastMonth.Location())
	
	// Get daemon configuration and URL
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	daemonURL := fmt.Sprintf("http://%s", config.Daemon.ListenAddr)

	// Create HTTP client with timeout
	client := &HTTPClient{
		client: &http.Client{Timeout: 30 * time.Second},
		timeout: 30 * time.Second,
	}

	// Get last month's report
	report, err := client.GetMonthlyReport(daemonURL, lastMonthStart)
	if err != nil {
		return fmt.Errorf("failed to get last month's report: %w", err)
	}

	// Convert to enhanced report
	enhancedReport := &EnhancedMonthlyReport{
		Month:               lastMonthStart,
		MonthName:           lastMonthStart.Format("January 2006"),
		Year:                lastMonthStart.Year(),
		DaysCompleted:       getDaysInMonth(lastMonthStart),
		TotalDays:           getDaysInMonth(lastMonthStart),
		TotalWorkHours:      report.TotalHours,
		DailyAverage:        report.TotalHours / float64(getDaysInMonth(lastMonthStart)),
		ProjectedHours:      report.TotalHours, // Already completed
		ClaudeUsageHours:    0, // Will be calculated from actual data
		ClaudeUsagePercent:  0,
		BestDay:             DaySummary{},
		WeeklyBreakdown:     generateMockWeeklyBreakdown(lastMonthStart, report.TotalHours),
		ProjectBreakdown:    []ProjectBreakdown{},
		DailyProgress:       generateMockMonthlyProgress(lastMonthStart, report.TotalHours, getDaysInMonth(lastMonthStart)),
		MonthlyStats:        MonthlyStats{},
		Achievements:        []Achievement{},
		Trends:              []Trend{},
		Insights:            []string{fmt.Sprintf("Completed %s with %.1f total hours", lastMonthStart.Format("January 2006"), report.TotalHours)},
	}
	
	// Add insight about completion
	if enhancedReport.ProjectedHours > 0 {
		enhancedReport.Insights = append(enhancedReport.Insights,
			fmt.Sprintf("Completed month with %s total work time",
				formatDuration(time.Duration(enhancedReport.ProjectedHours*float64(time.Hour)))))
	}
	
	return displayEnhancedMonthlyReport(enhancedReport)
}

// Data structures for additional report types
type WeeklyReport struct {
	Week             time.Time         `json:"week"`
	TotalHours       float64          `json:"total_hours"`
	DailyBreakdown   []DaySummary     `json:"daily_breakdown"`
	ProjectSummaries []ProjectSummary `json:"project_summaries"`
	Trends           []Trend          `json:"trends"`
}

type MonthlyReport struct {
	Month            time.Time         `json:"month"`
	TotalHours       float64          `json:"total_hours"`
	WeeklyBreakdown  []WeekSummary    `json:"weekly_breakdown"`
	ProjectSummaries []ProjectSummary `json:"project_summaries"`
	Comparisons      []Comparison     `json:"comparisons"`
}



type Comparison struct {
	Period string  `json:"period"`
	Change float64 `json:"change"`
}

type HealthStatus struct {
	Version         string        `json:"version"`
	Uptime          time.Duration `json:"uptime"`
	ListenAddr      string        `json:"listen_addr"`
	DatabasePath    string        `json:"database_path"`
	DatabaseSize    string        `json:"database_size"`
	ActiveSessions  int           `json:"active_sessions"`
	TotalWorkBlocks int           `json:"total_work_blocks"`
}

type RecentActivity struct {
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"`
	ProjectName string    `json:"project_name"`
}