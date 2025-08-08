/**
 * CONTEXT:   Display constants and configuration for Claude Monitor professional UI
 * INPUT:     Color schemes, symbols, and box drawing characters for consistent theming
 * OUTPUT:    Centralized display configuration for professional CLI interface
 * BUSINESS:  Consistent visual identity enhances user confidence and adoption
 * CHANGE:    Extracted from professional_display.go to follow Single Responsibility Principle
 * RISK:      Low - Constants-only file with no behavioral logic
 */

package reporting

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
	SymbolWork       = "⚡"
	SymbolTime       = "🕰️"
	SymbolEfficiency = "🎯"
	SymbolFocus      = "🧠"
	SymbolClaude     = "🤖"
	SymbolProject    = "📁"
	SymbolSession    = "📊"
	SymbolTimeline   = "⏱️"
	SymbolInsight    = "💡"
	SymbolTrend      = "📈"
)

// Display configuration constants
const (
	DefaultSectionWidth = 66
	DefaultHeaderWidth  = 68
	MaxProjectNameLen   = 25
	MaxTextWrapWidth    = 66
)

// Efficiency thresholds for color coding
const (
	HighEfficiencyThreshold   = 80.0
	MediumEfficiencyThreshold = 60.0
)

// Focus score calculation constants
const (
	EfficiencyWeight    = 0.7
	MaxSessionBonus     = 30.0
	SessionBonusPerUnit = 10.0
	MaxFocusScore       = 100.0
)

// Duration thresholds for color coding (in hours)
const (
	LongDurationThreshold   = 2.0
	MediumDurationThreshold = 1.0
)

// Project color thresholds (percentage)
const (
	HighProjectThreshold   = 40.0
	MediumProjectThreshold = 20.0
)