/**
 * CONTEXT:   Professional CLI display utilities for Claude Monitor reports
 * INPUT:     Report data structures and formatting requirements
 * OUTPUT:    Premium, visually stunning CLI interface matching modern developer tools
 * BUSINESS:  Professional appearance enhances user confidence and adoption
 * CHANGE:    New professional display system for enhanced UX/UI
 * RISK:      Low - Display enhancement with no business logic changes
 */

package reporting

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Color constants for professional theming
const (
	ColorReset     = "\033[0m"
	ColorBold      = "\033[1m"
	ColorDim       = "\033[2m"
	ColorUnderline = "\033[4m"

	// Primary colors
	ColorBlue    = "\033[34m"
	ColorCyan    = "\033[36m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorRed     = "\033[31m"
	ColorMagenta = "\033[35m"
	ColorWhite   = "\033[37m"

	// Bright colors
	ColorBrightBlue    = "\033[94m"
	ColorBrightCyan    = "\033[96m"
	ColorBrightGreen   = "\033[92m"
	ColorBrightYellow  = "\033[93m"
	ColorBrightRed     = "\033[91m"
	ColorBrightWhite   = "\033[97m"
	ColorBrightMagenta = "\033[95m"

	// Background colors
	BgBlue   = "\033[44m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgRed    = "\033[41m"
)

// Professional box drawing characters
const (
	BoxTopLeft     = "╭"
	BoxTopRight    = "╮"
	BoxBottomLeft  = "╰"
	BoxBottomRight = "╯"
	BoxHorizontal  = "─"
	BoxVertical    = "│"
	BoxCross       = "┼"
	BoxTeeDown     = "┬"
	BoxTeeUp       = "┴"
	BoxTeeRight    = "├"
	BoxTeeLeft     = "┤"
)

// Professional symbols
const (
	SymbolWork      = "⚡"
	SymbolTime      = "🕰️"
	SymbolEfficiency = "🎯"
	SymbolFocus     = "🧠"
	SymbolClaude    = "🤖"
	SymbolProject   = "📁"
	SymbolSession   = "📊"
	SymbolTimeline  = "⏱️"
	SymbolInsight   = "💡"
	SymbolTrend     = "📈"
)

/**
 * CONTEXT:   Professional header for Claude Monitor reports
 * INPUT:     Report title and date information
 * OUTPUT:    Elegant bordered header with consistent branding
 * BUSINESS:  Professional appearance builds user confidence
 * CHANGE:    Professional header design with modern aesthetics
 * RISK:      Low - Visual enhancement only
 */
func DisplayProfessionalHeader(title, date string) {
	headerWidth := 68
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
 * CHANGE:    Professional dashboard design with visual hierarchy
 * RISK:      Low - Metrics display enhancement
 */
func DisplayMetricsDashboard(activeWork time.Duration, totalTime time.Duration, sessions int, efficiency float64, claudeTime time.Duration) {
	sectionWidth := 66
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
 * CHANGE:    Professional table design with enhanced readability
 * RISK:      Low - Project display enhancement
 */
type ProjectData struct {
	Name     string
	Duration time.Duration
	Percent  float64
	Sessions int
	Color    string
}

func DisplayProfessionalProjectBreakdown(projects []ProjectData) {
	if len(projects) == 0 {
		return
	}
	
	sectionWidth := 66
	
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
		projectName := truncateStringPro(project.Name, 25)
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
 * CHANGE:    Enhanced timeline with visual connectors and better formatting
 * RISK:      Low - Timeline display enhancement
 */
type WorkBlockData struct {
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	ProjectName   string
	ActivityCount int
}

func DisplayProfessionalWorkTimeline(workBlocks []WorkBlockData) {
	if len(workBlocks) == 0 {
		return
	}
	
	sectionWidth := 66
	
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
 * CHANGE:    Enhanced insight formatting with visual appeal
 * RISK:      Low - Insights display improvement
 */
func DisplayProfessionalInsights(insights []string) {
	if len(insights) == 0 {
		return
	}
	
	sectionWidth := 66
	
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
 * CHANGE:    Professional footer with clear next steps
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
 * CHANGE:    Enhanced empty state with actionable guidance
 * RISK:      Low - Empty state improvement
 */
func DisplayProfessionalEmptyState(message string) {
	sectionWidth := 66
	
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

// Professional utility functions

func formatDurationPro(d time.Duration) string {
	if d == 0 {
		return "0m"
	}
	
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	
	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}

func truncateStringPro(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "…"
}

func getEfficiencyColor(efficiency float64) string {
	if efficiency >= 80 {
		return ColorBrightGreen
	}
	if efficiency >= 60 {
		return ColorBrightYellow
	}
	return ColorBrightRed
}

func getFocusColor(focusScore int) string {
	if focusScore >= 80 {
		return ColorBrightGreen
	}
	if focusScore >= 60 {
		return ColorBrightYellow
	}
	return ColorBrightRed
}

func getDurationColor(d time.Duration) string {
	hours := d.Hours()
	if hours >= 2 {
		return ColorBrightGreen
	}
	if hours >= 1 {
		return ColorBrightYellow
	}
	return ColorBrightRed
}

func getProjectColor(percent float64) string {
	if percent >= 40 {
		return ColorBrightCyan
	}
	if percent >= 20 {
		return ColorCyan
	}
	return ColorDim
}

func calculateFocusScore(efficiency float64, sessions int) int {
	// Calculate focus score based on efficiency and session count
	baseScore := efficiency * 0.7
	sessionBonus := math.Min(float64(sessions)*10, 30)
	return int(math.Min(baseScore+sessionBonus, 100))
}

func wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}
	
	var wrapped []string
	words := strings.Fields(text)
	currentLine := ""
	
	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		} else {
			wrapped = append(wrapped, currentLine)
			currentLine = word
		}
	}
	
	if currentLine != "" {
		wrapped = append(wrapped, currentLine)
	}
	
	return strings.Join(wrapped, "\n")
}