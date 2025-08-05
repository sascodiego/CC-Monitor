package cli

import (
	"fmt"
	"os"
	"strings"
	"time"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-010
 * CONTEXT:   Professional output formatting utilities for consistent CLI presentation
 * REASON:    Need consistent, beautiful terminal output with colors, formatting, and proper spacing
 * CHANGE:    New implementation of comprehensive formatting utilities for professional CLI interface.
 * PREVENTION:Handle terminal width detection and gracefully fallback to basic formatting if colors not supported
 * RISK:      Low - Formatting issues affect presentation but not functionality
 */

// ANSI color codes for terminal formatting
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
	ColorItalic = "\033[3m"
	
	// Foreground colors
	ColorBlack   = "\033[30m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	
	// Bright foreground colors
	ColorBrightBlack   = "\033[90m"
	ColorBrightRed     = "\033[91m"
	ColorBrightGreen   = "\033[92m"
	ColorBrightYellow  = "\033[93m"
	ColorBrightBlue    = "\033[94m"
	ColorBrightMagenta = "\033[95m"
	ColorBrightCyan    = "\033[96m"
	ColorBrightWhite   = "\033[97m"
	
	// Background colors
	ColorBgBlack   = "\033[40m"
	ColorBgRed     = "\033[41m"
	ColorBgGreen   = "\033[42m"
	ColorBgYellow  = "\033[43m"
	ColorBgBlue    = "\033[44m"
	ColorBgMagenta = "\033[45m"
	ColorBgCyan    = "\033[46m"
	ColorBgWhite   = "\033[47m"
)

// Unicode symbols for enhanced visual presentation
const (
	SymbolSuccess   = "✓"
	SymbolError     = "✗"
	SymbolWarning   = "⚠"
	SymbolInfo      = "ℹ"
	SymbolArrow     = "→"
	SymbolBullet    = "•"
	SymbolClock     = "⏱"
	SymbolGear      = "⚙"
	SymbolChart     = "📊"
	SymbolServer    = "🖥"
	SymbolDatabase  = "💾"
	SymbolNetwork   = "🌐"
	SymbolProcess   = "⚡"
)

// OutputFormatter provides formatting utilities for consistent CLI output
type OutputFormatter struct {
	colorEnabled bool
	terminalWidth int
}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter() *OutputFormatter {
	return &OutputFormatter{
		colorEnabled:  isColorSupported(),
		terminalWidth: getTerminalWidth(),
	}
}

// Color formatting methods
func (f *OutputFormatter) Colorize(text, color string) string {
	if !f.colorEnabled {
		return text
	}
	
	var colorCode string
	switch strings.ToLower(color) {
	case "red", "error":
		colorCode = ColorRed
	case "green", "success":
		colorCode = ColorGreen
	case "yellow", "warning":
		colorCode = ColorYellow
	case "blue", "info":
		colorCode = ColorBlue
	case "cyan":
		colorCode = ColorCyan
	case "magenta":
		colorCode = ColorMagenta
	case "white":
		colorCode = ColorWhite
	case "bright-red":
		colorCode = ColorBrightRed
	case "bright-green":
		colorCode = ColorBrightGreen
	case "bright-yellow":
		colorCode = ColorBrightYellow
	case "bright-blue":
		colorCode = ColorBrightBlue
	case "bright-cyan":
		colorCode = ColorBrightCyan
	case "dim":
		colorCode = ColorDim
	case "bold":
		colorCode = ColorBold
	default:
		return text
	}
	
	return colorCode + text + ColorReset
}

// Bold text formatting
func (f *OutputFormatter) Bold(text string) string {
	if !f.colorEnabled {
		return text
	}
	return ColorBold + text + ColorReset
}

// Dim text formatting
func (f *OutputFormatter) Dim(text string) string {
	if !f.colorEnabled {
		return text
	}
	return ColorDim + text + ColorReset
}

// Section headers and dividers
func (f *OutputFormatter) PrintSectionHeader(title string) {
	fmt.Printf("%s %s\n", f.Colorize(SymbolChart, "bright-blue"), f.Bold(title))
	fmt.Println(f.Colorize(strings.Repeat("═", len(title)+2), "blue"))
}

func (f *OutputFormatter) PrintSubHeader(title string) {
	fmt.Printf("%s %s\n", f.Colorize(SymbolBullet, "cyan"), f.Bold(title))
	fmt.Println(f.Dim(strings.Repeat("─", len(title)+2)))
}

// Status and message printing
func (f *OutputFormatter) PrintSuccess(message string) {
	fmt.Printf("%s %s\n", f.Colorize(SymbolSuccess, "green"), f.Colorize(message, "green"))
}

func (f *OutputFormatter) PrintError(message string, err error) {
	fmt.Printf("%s %s", f.Colorize(SymbolError, "red"), f.Colorize(message, "red"))
	if err != nil {
		fmt.Printf(": %s", f.Dim(err.Error()))
	}
	fmt.Println()
}

func (f *OutputFormatter) PrintWarning(message string) {
	fmt.Printf("%s %s\n", f.Colorize(SymbolWarning, "yellow"), f.Colorize(message, "yellow"))
}

func (f *OutputFormatter) PrintInfo(message string) {
	fmt.Printf("%s %s\n", f.Colorize(SymbolInfo, "blue"), message)
}

func (f *OutputFormatter) PrintStep(message string) {
	fmt.Printf("%s %s... ", f.Colorize(SymbolArrow, "cyan"), message)
}

// Status item formatting for key-value pairs
func (f *OutputFormatter) PrintStatusItem(key, value, statusType string) {
	maxKeyWidth := 20
	keyFormatted := fmt.Sprintf("%-*s", maxKeyWidth, key+":")
	
	var valueFormatted string
	switch statusType {
	case "success":
		valueFormatted = f.Colorize(value, "green")
	case "error":
		valueFormatted = f.Colorize(value, "red")
	case "warning":
		valueFormatted = f.Colorize(value, "yellow")
	case "info":
		valueFormatted = f.Colorize(value, "blue")
	default:
		valueFormatted = value
	}
	
	fmt.Printf("  %s %s\n", f.Dim(keyFormatted), valueFormatted)
}

// Progress and loading indicators
func (f *OutputFormatter) PrintProgressBar(current, total int, prefix string) {
	if total == 0 {
		return
	}
	
	percentage := float64(current) / float64(total)
	barWidth := 40
	filled := int(percentage * float64(barWidth))
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	
	fmt.Printf("\r%s [%s] %3.0f%% (%d/%d)", 
		prefix, 
		f.Colorize(bar, "cyan"), 
		percentage*100, 
		current, 
		total)
	
	if current >= total {
		fmt.Println()
	}
}

func (f *OutputFormatter) PrintSpinner(message string) {
	// Simple spinner implementation
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for i := 0; i < 10; i++ {
		fmt.Printf("\r%s %s", f.Colorize(spinners[i%len(spinners)], "cyan"), message)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Print("\r")
}

// Table formatting utilities
type TableFormatter struct {
	headers []string
	rows    [][]string
	widths  []int
	formatter *OutputFormatter
}

func (f *OutputFormatter) NewTable(headers []string) *TableFormatter {
	tf := &TableFormatter{
		headers:   headers,
		widths:    make([]int, len(headers)),
		formatter: f,
	}
	
	// Initialize widths with header lengths
	for i, header := range headers {
		tf.widths[i] = len(header)
	}
	
	return tf
}

func (tf *TableFormatter) AddRow(row []string) {
	if len(row) != len(tf.headers) {
		return // Skip invalid rows
	}
	
	// Update column widths
	for i, cell := range row {
		if len(cell) > tf.widths[i] {
			tf.widths[i] = len(cell)
		}
	}
	
	tf.rows = append(tf.rows, row)
}

func (tf *TableFormatter) Print() {
	if len(tf.rows) == 0 {
		tf.formatter.PrintWarning("No data to display")
		return
	}
	
	// Header
	for i, header := range tf.headers {
		headerFormatted := tf.formatter.Bold(header)
		fmt.Printf("%-*s", tf.widths[i]+10, headerFormatted) // +10 for ANSI codes
		if i < len(tf.headers)-1 {
			fmt.Print(" │ ")
		}
	}
	fmt.Println()
	
	// Separator
	for i, width := range tf.widths {
		fmt.Print(tf.formatter.Dim(strings.Repeat("─", width)))
		if i < len(tf.widths)-1 {
			fmt.Print(tf.formatter.Dim("─┼─"))
		}
	}
	fmt.Println()
	
	// Rows
	for _, row := range tf.rows {
		for i, cell := range row {
			fmt.Printf("%-*s", tf.widths[i], cell)
			if i < len(row)-1 {
				fmt.Print(tf.formatter.Dim(" │ "))
			}
		}
		fmt.Println()
	}
}

// Utility formatting functions
func (f *OutputFormatter) FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1e6)
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}

func (f *OutputFormatter) FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (f *OutputFormatter) TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	
	if maxLen <= 3 {
		return s[:maxLen]
	}
	
	return s[:maxLen-3] + "..."
}

func (f *OutputFormatter) PadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func (f *OutputFormatter) PadLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

func (f *OutputFormatter) Center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	
	padding := width - len(s)
	leftPad := padding / 2
	rightPad := padding - leftPad
	
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

// Banners and special formatting
func (f *OutputFormatter) PrintStartupBanner() {
	banner := []string{
		"┌─────────────────────────────────────────┐",
		"│          Claude Monitor v1.0.0         │",
		"│     Session & Work Hours Tracking      │",
		"└─────────────────────────────────────────┘",
	}
	
	fmt.Println()
	for _, line := range banner {
		fmt.Println(f.Colorize(line, "cyan"))
	}
	fmt.Println()
}

func (f *OutputFormatter) PrintDivider() {
	width := f.terminalWidth
	if width <= 0 {
		width = 80
	}
	fmt.Println(f.Dim(strings.Repeat("─", width)))
}

func (f *OutputFormatter) ClearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (f *OutputFormatter) ClearLine() {
	fmt.Print("\033[2K\r")
}

// Terminal capability detection
func isColorSupported() bool {
	// Check for common environment variables that indicate color support
	term := os.Getenv("TERM")
	if term == "" {
		return false
	}
	
	// Most modern terminals support color
	colorTerms := []string{"xterm", "xterm-256color", "screen", "tmux", "rxvt", "konsole", "gnome", "linux"}
	for _, colorTerm := range colorTerms {
		if strings.Contains(term, colorTerm) {
			return true
		}
	}
	
	// Check for NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	
	// Check for FORCE_COLOR environment variable
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	
	return false
}

func getTerminalWidth() int {
	// Try to get terminal width from environment
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if width, err := fmt.Sscanf(cols, "%d", new(int)); err == nil && width > 0 {
			return *new(int)
		}
	}
	
	// Default fallback width
	return 80
}

// Percentage and statistics formatting
func (f *OutputFormatter) FormatPercentage(value, total float64) string {
	if total == 0 {
		return "0%"
	}
	
	percentage := (value / total) * 100
	return fmt.Sprintf("%.1f%%", percentage)
}

func (f *OutputFormatter) FormatStatistic(label, value string, trend string) {
	var trendSymbol string
	var trendColor string
	
	switch trend {
	case "up":
		trendSymbol = "↗"
		trendColor = "green"
	case "down":
		trendSymbol = "↘"
		trendColor = "red"
	case "stable":
		trendSymbol = "→"
		trendColor = "blue"
	default:
		trendSymbol = ""
		trendColor = ""
	}
	
	fmt.Printf("  %s: %s", f.Dim(label), f.Bold(value))
	if trendSymbol != "" {
		fmt.Printf(" %s", f.Colorize(trendSymbol, trendColor))
	}
	fmt.Println()
}

// Interactive prompts and confirmations
func (f *OutputFormatter) PromptConfirmation(message string, defaultYes bool) bool {
	var prompt string
	if defaultYes {
		prompt = fmt.Sprintf("%s %s [Y/n]: ", f.Colorize("?", "blue"), message)
	} else {
		prompt = fmt.Sprintf("%s %s [y/N]: ", f.Colorize("?", "blue"), message)
	}
	
	fmt.Print(prompt)
	
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return defaultYes
	}
	
	response = strings.ToLower(strings.TrimSpace(response))
	
	switch response {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	case "":
		return defaultYes
	default:
		f.PrintWarning("Please answer 'y' or 'n'")
		return f.PromptConfirmation(message, defaultYes)
	}
}

func (f *OutputFormatter) PromptInput(message, defaultValue string) string {
	var prompt string
	if defaultValue != "" {
		prompt = fmt.Sprintf("%s %s [%s]: ", f.Colorize("?", "blue"), message, f.Dim(defaultValue))
	} else {
		prompt = fmt.Sprintf("%s %s: ", f.Colorize("?", "blue"), message)
	}
	
	fmt.Print(prompt)
	
	var response string
	if _, err := fmt.Scanln(&response); err != nil || response == "" {
		return defaultValue
	}
	
	return strings.TrimSpace(response)
}