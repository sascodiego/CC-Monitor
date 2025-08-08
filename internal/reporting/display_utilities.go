/**
 * CONTEXT:   Display utility functions for Claude Monitor professional CLI formatting
 * INPUT:     Raw data values (durations, percentages, strings) requiring formatting
 * OUTPUT:    Formatted strings with appropriate colors and truncation for display
 * BUSINESS:  Consistent formatting creates professional user experience
 * CHANGE:    Extracted from professional_display.go to follow Single Responsibility Principle
 * RISK:      Low - Pure utility functions with no side effects
 */

package reporting

import (
	"fmt"
	"math"
	"strings"
	"time"
)

/**
 * CONTEXT:   Format duration values for professional display
 * INPUT:     Time duration value to format
 * OUTPUT:    Human-readable duration string (e.g., "2h 30m", "45m", "0m")
 * BUSINESS:  Consistent time formatting improves report readability
 * CHANGE:    Extracted utility function for duration formatting
 * RISK:      Low - Pure formatting function with consistent output
 */
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

/**
 * CONTEXT:   Truncate strings to fit within display constraints
 * INPUT:     String to truncate and maximum length allowed
 * OUTPUT:    Truncated string with ellipsis if needed, preserving readability
 * BUSINESS:  Consistent truncation prevents UI layout issues
 * CHANGE:    Extracted utility function for string truncation
 * RISK:      Low - Safe string manipulation with length checks
 */
func truncateStringPro(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "…"
}

/**
 * CONTEXT:   Color coding for efficiency percentages
 * INPUT:     Efficiency percentage (0-100)
 * OUTPUT:    ANSI color code based on efficiency thresholds
 * BUSINESS:  Color coding provides instant visual feedback on performance
 * CHANGE:    Extracted utility function for efficiency color mapping
 * RISK:      Low - Pure color mapping with defined thresholds
 */
func getEfficiencyColor(efficiency float64) string {
	if efficiency >= HighEfficiencyThreshold {
		return ColorBrightGreen
	}
	if efficiency >= MediumEfficiencyThreshold {
		return ColorBrightYellow
	}
	return ColorBrightRed
}

/**
 * CONTEXT:   Color coding for focus scores
 * INPUT:     Focus score (0-100 calculated value)
 * OUTPUT:    ANSI color code based on focus score thresholds
 * BUSINESS:  Visual focus indicators help users understand productivity patterns
 * CHANGE:    Extracted utility function for focus score color mapping
 * RISK:      Low - Pure color mapping with consistent thresholds
 */
func getFocusColor(focusScore int) string {
	if focusScore >= int(HighEfficiencyThreshold) {
		return ColorBrightGreen
	}
	if focusScore >= int(MediumEfficiencyThreshold) {
		return ColorBrightYellow
	}
	return ColorBrightRed
}

/**
 * CONTEXT:   Color coding for work block durations
 * INPUT:     Duration of work block
 * OUTPUT:    ANSI color code based on duration length thresholds
 * BUSINESS:  Duration colors help identify long vs short work sessions
 * CHANGE:    Extracted utility function for duration color mapping
 * RISK:      Low - Pure duration analysis with predefined thresholds
 */
func getDurationColor(d time.Duration) string {
	hours := d.Hours()
	if hours >= LongDurationThreshold {
		return ColorBrightGreen
	}
	if hours >= MediumDurationThreshold {
		return ColorBrightYellow
	}
	return ColorBrightRed
}

/**
 * CONTEXT:   Color coding for project time allocation percentages
 * INPUT:     Project time percentage (0-100)
 * OUTPUT:    ANSI color code based on project allocation thresholds
 * BUSINESS:  Project colors highlight time distribution across projects
 * CHANGE:    Extracted utility function for project color mapping
 * RISK:      Low - Pure percentage analysis with consistent thresholds
 */
func getProjectColor(percent float64) string {
	if percent >= HighProjectThreshold {
		return ColorBrightCyan
	}
	if percent >= MediumProjectThreshold {
		return ColorCyan
	}
	return ColorDim
}

/**
 * CONTEXT:   Calculate focus score based on efficiency and session metrics
 * INPUT:     Efficiency percentage and number of sessions
 * OUTPUT:    Focus score (0-100) combining efficiency with session activity
 * BUSINESS:  Focus score provides comprehensive productivity assessment
 * CHANGE:    Extracted utility function for focus score calculation
 * RISK:      Low - Mathematical calculation with bounded output (0-100)
 */
func calculateFocusScore(efficiency float64, sessions int) int {
	// Calculate focus score based on efficiency and session count
	baseScore := efficiency * EfficiencyWeight
	sessionBonus := math.Min(float64(sessions)*SessionBonusPerUnit, MaxSessionBonus)
	return int(math.Min(baseScore+sessionBonus, MaxFocusScore))
}

/**
 * CONTEXT:   Text wrapping utility for multi-line display formatting
 * INPUT:     Text string and desired width for wrapping
 * OUTPUT:    Multi-line string with proper word breaks at specified width
 * BUSINESS:  Consistent text wrapping maintains professional layout
 * CHANGE:    Extracted utility function for text formatting
 * RISK:      Low - String manipulation with word boundary preservation
 */
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