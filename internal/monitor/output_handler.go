/**
 * CONTEXT:   Advanced output handler for Claude monitoring events and analytics
 * INPUT:     Events from all monitoring subsystems and comprehensive statistics
 * OUTPUT:    Formatted output, logging, and reporting for Claude monitoring
 * BUSINESS:  Output handler provides user-friendly monitoring information display
 * CHANGE:    Initial output handler based on prototype with enhanced formatting
 * RISK:      Low - Output formatting utility with no system impact
 */

package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

/**
 * CONTEXT:   Comprehensive output handler for monitoring events and statistics
 * INPUT:     All types of monitoring events and analytics data
 * OUTPUT:    Formatted console output, file logging, and reports
 * BUSINESS:  Output handler provides comprehensive monitoring information display
 * CHANGE:    Initial comprehensive output handler
 * RISK:      Low - Output handling utility with file operations
 */
type OutputHandler struct {
	config          OutputConfig
	logFile         *os.File
	jsonEncoder     *json.Encoder
	eventLog        []EventLogEntry
	statistics      *ComprehensiveStats
	mu              sync.RWMutex
	startTime       time.Time
	outputFormatters map[string]EventFormatter
}

/**
 * CONTEXT:   Output handler configuration
 * INPUT:     Output formatting and logging parameters
 * OUTPUT:    Output handler behavior configuration
 * BUSINESS:  Configuration enables customizable output formatting
 * CHANGE:    Initial output configuration structure
 * RISK:      Low - Configuration structure for output handling
 */
type OutputConfig struct {
	LogToFile        bool   `json:"log_to_file"`        // Enable file logging
	LogFile          string `json:"log_file"`           // Log file path
	LogFormat        string `json:"log_format"`         // json, text, csv
	ConsoleOutput    bool   `json:"console_output"`     // Enable console output
	VerboseMode      bool   `json:"verbose_mode"`       // Verbose output
	ColorOutput      bool   `json:"color_output"`       // Colored console output
	TimestampFormat  string `json:"timestamp_format"`   // Timestamp format
	MaxLogEntries    int    `json:"max_log_entries"`    // Max log entries to keep
	ReportGeneration bool   `json:"report_generation"`  // Enable report generation
	ReportInterval   time.Duration `json:"report_interval"` // Report generation interval
}

/**
 * CONTEXT:   Event log entry for structured logging
 * INPUT:     Event data with metadata
 * OUTPUT:    Structured log entry
 * BUSINESS:  Event logging enables comprehensive monitoring audit trail
 * CHANGE:    Initial event log entry structure
 * RISK:      Low - Log entry data structure
 */
type EventLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	Source      string                 `json:"source"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	ProjectName string                 `json:"project_name"`
	ProcessPID  int                    `json:"process_pid"`
	Details     map[string]interface{} `json:"details"`
}

/**
 * CONTEXT:   Event formatter function type
 * INPUT:     Event data for formatting
 * OUTPUT:    Formatted string representation
 * BUSINESS:  Event formatters enable customizable event display
 * CHANGE:    Initial event formatter function type
 * RISK:      Low - Function type definition for event formatting
 */
type EventFormatter func(interface{}) string

/**
 * CONTEXT:   Console color codes for enhanced display
 * INPUT:     Color formatting requirements
 * OUTPUT:    ANSI color codes
 * BUSINESS:  Color codes enhance console output readability
 * CHANGE:    Initial console color code definitions
 * RISK:      Low - Console color constants
 */
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

/**
 * CONTEXT:   Create new output handler with configuration
 * INPUT:     Output configuration and statistics system
 * OUTPUT:    Configured output handler ready for event processing
 * BUSINESS:  Output handler creation enables comprehensive monitoring display
 * CHANGE:    Initial output handler constructor
 * RISK:      Medium - File operations for logging setup
 */
func NewOutputHandler(config OutputConfig, stats *ComprehensiveStats) (*OutputHandler, error) {
	// Set default configuration
	if config.TimestampFormat == "" {
		config.TimestampFormat = "15:04:05.000"
	}
	if config.MaxLogEntries == 0 {
		config.MaxLogEntries = 10000
	}
	if config.ReportInterval == 0 {
		config.ReportInterval = 5 * time.Minute
	}
	
	handler := &OutputHandler{
		config:           config,
		eventLog:         make([]EventLogEntry, 0),
		statistics:       stats,
		startTime:        time.Now(),
		outputFormatters: make(map[string]EventFormatter),
	}
	
	// Setup file logging if enabled
	if config.LogToFile && config.LogFile != "" {
		if err := handler.setupFileLogging(); err != nil {
			return nil, fmt.Errorf("failed to setup file logging: %w", err)
		}
	}
	
	// Setup event formatters
	handler.setupEventFormatters()
	
	// Start report generation if enabled
	if config.ReportGeneration {
		go handler.reportGenerationLoop()
	}
	
	return handler, nil
}

/**
 * CONTEXT:   Setup file logging system
 * INPUT:     Log file configuration
 * OUTPUT:    Configured file logging
 * BUSINESS:  File logging enables persistent monitoring audit trail
 * CHANGE:    Initial file logging setup
 * RISK:      Medium - File system operations for logging
 */
func (oh *OutputHandler) setupFileLogging() error {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(oh.config.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	
	// Open log file
	file, err := os.OpenFile(oh.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	
	oh.logFile = file
	
	// Setup JSON encoder if using JSON format
	if oh.config.LogFormat == "json" {
		oh.jsonEncoder = json.NewEncoder(file)
	}
	
	return nil
}

/**
 * CONTEXT:   Setup event formatters for different event types
 * INPUT:     Event formatter configuration
 * OUTPUT:    Configured event formatters
 * BUSINESS:  Event formatters enable customized event display
 * CHANGE:    Initial event formatter setup
 * RISK:      Low - Event formatter function setup
 */
func (oh *OutputHandler) setupEventFormatters() {
	oh.outputFormatters["process"] = oh.formatProcessEvent
	oh.outputFormatters["file_io"] = oh.formatFileIOEvent
	oh.outputFormatters["http"] = oh.formatHTTPEvent
	oh.outputFormatters["network"] = oh.formatNetworkEvent
	oh.outputFormatters["activity"] = oh.formatActivityEvent
}

/**
 * CONTEXT:   Handle process event output
 * INPUT:     Process event data
 * OUTPUT:    Formatted process event output
 * BUSINESS:  Process event handling enables process lifecycle tracking display
 * CHANGE:    Initial process event output handling
 * RISK:      Low - Event output formatting
 */
func (oh *OutputHandler) HandleProcessEvent(event ProcessEvent) {
	entry := EventLogEntry{
		Timestamp:   event.Timestamp,
		EventType:   "process",
		Source:      "process_monitor",
		Severity:    oh.getEventSeverity("process", string(event.Type)),
		Message:     oh.formatProcessEvent(event),
		ProjectName: extractProjectNameFromPath(event.WorkingDir),
		ProcessPID:  event.PID,
		Details: map[string]interface{}{
			"command":     event.Command,
			"working_dir": event.WorkingDir,
			"event_type":  event.Type,
		},
	}
	
	oh.logEvent(entry)
	
	if oh.config.ConsoleOutput && (oh.config.VerboseMode || event.Type == ProcessStarted || event.Type == ProcessStopped) {
		oh.printToConsole(entry)
	}
}

/**
 * CONTEXT:   Handle file I/O event output
 * INPUT:     File I/O event data
 * OUTPUT:    Formatted file I/O event output
 * BUSINESS:  File I/O event handling enables work activity tracking display
 * CHANGE:    Initial file I/O event output handling
 * RISK:      Low - Event output formatting
 */
func (oh *OutputHandler) HandleFileIOEvent(event FileIOEvent) {
	entry := EventLogEntry{
		Timestamp:   event.Timestamp,
		EventType:   "file_io",
		Source:      "file_io_monitor",
		Severity:    oh.getEventSeverity("file_io", string(event.Type)),
		Message:     oh.formatFileIOEvent(event),
		ProjectName: event.ProjectName,
		ProcessPID:  event.PID,
		Details: map[string]interface{}{
			"file_path":    event.FilePath,
			"file_type":    event.FileType,
			"size":         event.Size,
			"is_work_file": event.IsWorkFile,
			"io_type":      event.Type,
		},
	}
	
	oh.logEvent(entry)
	
	// Only show work file events in console unless verbose mode
	if oh.config.ConsoleOutput && (oh.config.VerboseMode || event.IsWorkFile) {
		oh.printToConsole(entry)
	}
}

/**
 * CONTEXT:   Handle HTTP event output
 * INPUT:     HTTP event data
 * OUTPUT:    Formatted HTTP event output
 * BUSINESS:  HTTP event handling enables API usage tracking display
 * CHANGE:    Initial HTTP event output handling
 * RISK:      Low - Event output formatting
 */
func (oh *OutputHandler) HandleHTTPEvent(event HTTPEvent) {
	entry := EventLogEntry{
		Timestamp:   event.Timestamp,
		EventType:   "http",
		Source:      "http_monitor",
		Severity:    oh.getEventSeverity("http", string(event.Type)),
		Message:     oh.formatHTTPEvent(event),
		ProjectName: event.ProjectName,
		ProcessPID:  event.PID,
		Details: map[string]interface{}{
			"method":       event.Method,
			"url":          event.URL,
			"host":         event.Host,
			"status_code":  event.StatusCode,
			"is_claude_api": event.IsClaudeAPI,
		},
	}
	
	oh.logEvent(entry)
	
	// Show Claude API events and errors in console
	if oh.config.ConsoleOutput && (oh.config.VerboseMode || event.IsClaudeAPI || event.Type == HTTPError) {
		oh.printToConsole(entry)
	}
}

/**
 * CONTEXT:   Handle network event output
 * INPUT:     Network event data
 * OUTPUT:    Formatted network event output
 * BUSINESS:  Network event handling enables connectivity tracking display
 * CHANGE:    Initial network event output handling
 * RISK:      Low - Event output formatting
 */
func (oh *OutputHandler) HandleNetworkEvent(event NetworkEvent) {
	entry := EventLogEntry{
		Timestamp:   event.Timestamp,
		EventType:   "network",
		Source:      "network_monitor",
		Severity:    oh.getEventSeverity("network", string(event.Type)),
		Message:     oh.formatNetworkEvent(event),
		ProjectName: event.ProjectName,
		ProcessPID:  event.PID,
		Details: map[string]interface{}{
			"protocol":      event.Protocol,
			"local_addr":    event.LocalAddr,
			"remote_addr":   event.RemoteAddr,
			"remote_host":   event.RemoteHost,
			"is_claude_api": event.IsClaudeAPI,
		},
	}
	
	oh.logEvent(entry)
	
	// Show Claude API connections in console
	if oh.config.ConsoleOutput && (oh.config.VerboseMode || event.IsClaudeAPI) {
		oh.printToConsole(entry)
	}
}

/**
 * CONTEXT:   Handle activity event output
 * INPUT:     Activity event data
 * OUTPUT:    Formatted activity event output
 * BUSINESS:  Activity event handling enables work tracking display
 * CHANGE:    Initial activity event output handling
 * RISK:      Low - Event output formatting
 */
func (oh *OutputHandler) HandleActivityEvent(event ActivityEvent) {
	entry := EventLogEntry{
		Timestamp:   event.Timestamp,
		EventType:   "activity",
		Source:      "activity_generator",
		Severity:    "info",
		Message:     oh.formatActivityEvent(event),
		ProjectName: event.ProjectName,
		Details: map[string]interface{}{
			"activity_type":   event.ActivityType,
			"activity_source": event.ActivitySource,
			"description":     event.Description,
		},
	}
	
	oh.logEvent(entry)
	
	// Always show work activities in console
	if oh.config.ConsoleOutput {
		oh.printToConsole(entry)
	}
}

/**
 * CONTEXT:   Log event to internal log and file
 * INPUT:     Event log entry
 * OUTPUT:    Logged event entry
 * BUSINESS:  Event logging enables comprehensive audit trail
 * CHANGE:    Initial event logging implementation
 * RISK:      Low - Event logging utility
 */
func (oh *OutputHandler) logEvent(entry EventLogEntry) {
	oh.mu.Lock()
	defer oh.mu.Unlock()
	
	// Add to internal log
	oh.eventLog = append(oh.eventLog, entry)
	
	// Limit log size
	if len(oh.eventLog) > oh.config.MaxLogEntries {
		oh.eventLog = oh.eventLog[1:]
	}
	
	// Log to file if enabled
	if oh.logFile != nil {
		oh.logToFile(entry)
	}
}

/**
 * CONTEXT:   Log event to file
 * INPUT:     Event log entry
 * OUTPUT:    File-logged event
 * BUSINESS:  File logging enables persistent event storage
 * CHANGE:    Initial file logging implementation
 * RISK:      Medium - File system operations
 */
func (oh *OutputHandler) logToFile(entry EventLogEntry) {
	switch oh.config.LogFormat {
	case "json":
		if oh.jsonEncoder != nil {
			oh.jsonEncoder.Encode(entry)
		}
	case "csv":
		oh.logCSV(entry)
	default: // text format
		oh.logText(entry)
	}
}

/**
 * CONTEXT:   Log event in text format
 * INPUT:     Event log entry
 * OUTPUT:    Text-formatted log entry
 * BUSINESS:  Text logging provides human-readable event logs
 * CHANGE:    Initial text logging implementation
 * RISK:      Low - Text formatting utility
 */
func (oh *OutputHandler) logText(entry EventLogEntry) {
	timestamp := entry.Timestamp.Format(oh.config.TimestampFormat)
	logLine := fmt.Sprintf("%s [%s] [%s] %s\n", 
		timestamp, strings.ToUpper(entry.Severity), entry.EventType, entry.Message)
	
	oh.logFile.WriteString(logLine)
}

/**
 * CONTEXT:   Log event in CSV format
 * INPUT:     Event log entry
 * OUTPUT:    CSV-formatted log entry
 * BUSINESS:  CSV logging enables spreadsheet analysis of events
 * CHANGE:    Initial CSV logging implementation
 * RISK:      Low - CSV formatting utility
 */
func (oh *OutputHandler) logCSV(entry EventLogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	csvLine := fmt.Sprintf("%s,%s,%s,%s,%s,%d,%s\n",
		timestamp, entry.EventType, entry.Source, entry.Severity, 
		oh.escapeCSV(entry.Message), entry.ProcessPID, entry.ProjectName)
	
	oh.logFile.WriteString(csvLine)
}

/**
 * CONTEXT:   Print event to console with formatting
 * INPUT:     Event log entry
 * OUTPUT:    Formatted console output
 * BUSINESS:  Console output provides real-time monitoring feedback
 * CHANGE:    Initial console output implementation
 * RISK:      Low - Console output utility
 */
func (oh *OutputHandler) printToConsole(entry EventLogEntry) {
	timestamp := entry.Timestamp.Format(oh.config.TimestampFormat)
	
	// Get color based on event type and severity
	color := oh.getEventColor(entry.EventType, entry.Severity)
	emoji := oh.getEventEmoji(entry.EventType, entry.Severity)
	
	if oh.config.ColorOutput {
		fmt.Printf("%s%s %s [%s] %s%s\n", 
			color, emoji, timestamp, strings.ToUpper(entry.EventType), entry.Message, ColorReset)
	} else {
		fmt.Printf("%s %s [%s] %s\n", 
			emoji, timestamp, strings.ToUpper(entry.EventType), entry.Message)
	}
}

// Event formatters

/**
 * CONTEXT:   Format process event for display
 * INPUT:     Process event data
 * OUTPUT:    Formatted process event string
 * BUSINESS:  Process event formatting provides clear process lifecycle display
 * CHANGE:    Initial process event formatting
 * RISK:      Low - Event formatting utility
 */
func (oh *OutputHandler) formatProcessEvent(event interface{}) string {
	if pe, ok := event.(ProcessEvent); ok {
		projectName := extractProjectNameFromPath(pe.WorkingDir)
		return fmt.Sprintf("Claude %s: %s (PID: %d) in project '%s'", 
			pe.Type, pe.Command, pe.PID, projectName)
	}
	return fmt.Sprintf("Process event: %+v", event)
}

/**
 * CONTEXT:   Format file I/O event for display
 * INPUT:     File I/O event data
 * OUTPUT:    Formatted file I/O event string
 * BUSINESS:  File I/O event formatting provides clear work activity display
 * CHANGE:    Initial file I/O event formatting
 * RISK:      Low - Event formatting utility
 */
func (oh *OutputHandler) formatFileIOEvent(event interface{}) string {
	if fe, ok := event.(FileIOEvent); ok {
		fileName := filepath.Base(fe.FilePath)
		sizeStr := ""
		if fe.Size > 0 {
			sizeStr = fmt.Sprintf(" (%s)", oh.formatBytes(fe.Size))
		}
		return fmt.Sprintf("File %s: %s%s in %s", 
			fe.Type, fileName, sizeStr, fe.ProjectName)
	}
	return fmt.Sprintf("File I/O event: %+v", event)
}

/**
 * CONTEXT:   Format HTTP event for display
 * INPUT:     HTTP event data
 * OUTPUT:    Formatted HTTP event string
 * BUSINESS:  HTTP event formatting provides clear API usage display
 * CHANGE:    Initial HTTP event formatting
 * RISK:      Low - Event formatting utility
 */
func (oh *OutputHandler) formatHTTPEvent(event interface{}) string {
	if he, ok := event.(HTTPEvent); ok {
		apiIndicator := ""
		if he.IsClaudeAPI {
			apiIndicator = " [CLAUDE API]"
		}
		
		if he.StatusCode > 0 {
			return fmt.Sprintf("HTTP %s %s -> %d%s", 
				he.Method, he.Host, he.StatusCode, apiIndicator)
		}
		return fmt.Sprintf("HTTP %s %s%s", 
			he.Method, he.Host, apiIndicator)
	}
	return fmt.Sprintf("HTTP event: %+v", event)
}

/**
 * CONTEXT:   Format network event for display
 * INPUT:     Network event data
 * OUTPUT:    Formatted network event string
 * BUSINESS:  Network event formatting provides clear connectivity display
 * CHANGE:    Initial network event formatting
 * RISK:      Low - Event formatting utility
 */
func (oh *OutputHandler) formatNetworkEvent(event interface{}) string {
	if ne, ok := event.(NetworkEvent); ok {
		apiIndicator := ""
		if ne.IsClaudeAPI {
			apiIndicator = " [CLAUDE API]"
		}
		return fmt.Sprintf("Network %s: %s %s -> %s%s", 
			ne.Type, ne.Protocol, ne.LocalAddr, ne.RemoteAddr, apiIndicator)
	}
	return fmt.Sprintf("Network event: %+v", event)
}

/**
 * CONTEXT:   Format activity event for display
 * INPUT:     Activity event data
 * OUTPUT:    Formatted activity event string
 * BUSINESS:  Activity event formatting provides clear work activity display
 * CHANGE:    Initial activity event formatting
 * RISK:      Low - Event formatting utility
 */
func (oh *OutputHandler) formatActivityEvent(event interface{}) string {
	if ae, ok := event.(ActivityEvent); ok {
		return fmt.Sprintf("Work Activity: %s in %s - %s", 
			ae.ActivityType, ae.ProjectName, ae.Description)
	}
	return fmt.Sprintf("Activity event: %+v", event)
}

// Helper methods

func (oh *OutputHandler) getEventSeverity(eventType, subType string) string {
	switch eventType {
	case "process":
		if subType == "started" || subType == "stopped" {
			return "info"
		}
		return "debug"
	case "file_io":
		return "debug"
	case "http":
		if subType == "error" {
			return "error"
		} else if subType == "claude_api" {
			return "info"
		}
		return "debug"
	case "network":
		if subType == "claude_api" {
			return "info"
		}
		return "debug"
	case "activity":
		return "info"
	default:
		return "debug"
	}
}

func (oh *OutputHandler) getEventColor(eventType, severity string) string {
	if !oh.config.ColorOutput {
		return ""
	}
	
	switch severity {
	case "error":
		return ColorRed
	case "warning":
		return ColorYellow
	case "info":
		switch eventType {
		case "process":
			return ColorGreen
		case "activity":
			return ColorCyan
		case "http":
			return ColorBlue
		case "network":
			return ColorPurple
		default:
			return ColorWhite
		}
	default:
		return ColorWhite
	}
}

func (oh *OutputHandler) getEventEmoji(eventType, severity string) string {
	switch eventType {
	case "process":
		return "🔄"
	case "file_io":
		return "📁"
	case "http":
		return "🌐"
	case "network":
		return "🔗"
	case "activity":
		return "⚡"
	default:
		return "📋"
	}
}

func (oh *OutputHandler) formatBytes(bytes int64) string {
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

func (oh *OutputHandler) escapeCSV(text string) string {
	if strings.Contains(text, ",") || strings.Contains(text, "\"") || strings.Contains(text, "\n") {
		text = strings.ReplaceAll(text, "\"", "\"\"")
		return "\"" + text + "\""
	}
	return text
}

/**
 * CONTEXT:   Report generation loop
 * INPUT:     Periodic report generation timing
 * OUTPUT:    Generated reports at configured intervals
 * BUSINESS:  Report generation provides periodic monitoring summaries
 * CHANGE:    Initial report generation loop
 * RISK:      Low - Report generation utility
 */
func (oh *OutputHandler) reportGenerationLoop() {
	ticker := time.NewTicker(oh.config.ReportInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			oh.generatePeriodicReport()
		}
	}
}

/**
 * CONTEXT:   Generate periodic monitoring report
 * INPUT:     Current monitoring statistics
 * OUTPUT:    Formatted monitoring report
 * BUSINESS:  Periodic reports provide regular monitoring insights
 * CHANGE:    Initial periodic report generation
 * RISK:      Low - Report generation utility
 */
func (oh *OutputHandler) generatePeriodicReport() {
	if oh.statistics == nil {
		return
	}
	
	oh.mu.RLock()
	recentEvents := oh.getRecentEvents(oh.config.ReportInterval)
	oh.mu.RUnlock()
	
	if len(recentEvents) == 0 {
		return // No activity to report
	}
	
	fmt.Printf("\n========== MONITORING REPORT (%s) ==========\n", 
		time.Now().Format("15:04:05"))
	
	// Event summary
	eventCounts := make(map[string]int)
	for _, event := range recentEvents {
		eventCounts[event.EventType]++
	}
	
	fmt.Printf("Activity Summary (last %v):\n", oh.config.ReportInterval)
	for eventType, count := range eventCounts {
		fmt.Printf("  %s: %d events\n", eventType, count)
	}
	
	// Project activity
	projectActivity := make(map[string]int)
	for _, event := range recentEvents {
		if event.ProjectName != "" {
			projectActivity[event.ProjectName]++
		}
	}
	
	if len(projectActivity) > 0 {
		fmt.Printf("Project Activity:\n")
		// Sort projects by activity
		type projectCount struct {
			name  string
			count int
		}
		var projects []projectCount
		for name, count := range projectActivity {
			projects = append(projects, projectCount{name, count})
		}
		sort.Slice(projects, func(i, j int) bool {
			return projects[i].count > projects[j].count
		})
		
		for i, proj := range projects {
			if i >= 5 { // Show top 5 projects
				break
			}
			fmt.Printf("  %s: %d events\n", proj.name, proj.count)
		}
	}
	
	fmt.Printf("===============================================\n")
}

/**
 * CONTEXT:   Get recent events within time period
 * INPUT:     Time duration for recent event filtering
 * OUTPUT:    Recent event log entries
 * BUSINESS:  Recent event filtering enables time-based analysis
 * CHANGE:    Initial recent event filtering
 * RISK:      Low - Event filtering utility
 */
func (oh *OutputHandler) getRecentEvents(duration time.Duration) []EventLogEntry {
	cutoff := time.Now().Add(-duration)
	var recentEvents []EventLogEntry
	
	for _, event := range oh.eventLog {
		if event.Timestamp.After(cutoff) {
			recentEvents = append(recentEvents, event)
		}
	}
	
	return recentEvents
}

/**
 * CONTEXT:   Generate comprehensive monitoring report
 * INPUT:     Report generation request
 * OUTPUT:    Comprehensive monitoring report
 * BUSINESS:  Comprehensive reports provide detailed monitoring analysis
 * CHANGE:    Initial comprehensive report generation
 * RISK:      Low - Report generation utility
 */
func (oh *OutputHandler) GenerateReport() string {
	oh.mu.RLock()
	defer oh.mu.RUnlock()
	
	var report strings.Builder
	
	report.WriteString(fmt.Sprintf("========== CLAUDE MONITORING REPORT ==========\n"))
	report.WriteString(fmt.Sprintf("Report Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Monitoring Started: %s\n", oh.startTime.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Uptime: %s\n", time.Since(oh.startTime).Round(time.Second)))
	report.WriteString(fmt.Sprintf("Total Events Logged: %d\n\n", len(oh.eventLog)))
	
	// Event type breakdown
	eventCounts := make(map[string]int)
	severityCounts := make(map[string]int)
	projectCounts := make(map[string]int)
	
	for _, event := range oh.eventLog {
		eventCounts[event.EventType]++
		severityCounts[event.Severity]++
		if event.ProjectName != "" {
			projectCounts[event.ProjectName]++
		}
	}
	
	report.WriteString("Event Type Breakdown:\n")
	for eventType, count := range eventCounts {
		percentage := float64(count) / float64(len(oh.eventLog)) * 100
		report.WriteString(fmt.Sprintf("  %s: %d (%.1f%%)\n", eventType, count, percentage))
	}
	
	report.WriteString("\nSeverity Breakdown:\n")
	for severity, count := range severityCounts {
		percentage := float64(count) / float64(len(oh.eventLog)) * 100
		report.WriteString(fmt.Sprintf("  %s: %d (%.1f%%)\n", severity, count, percentage))
	}
	
	if len(projectCounts) > 0 {
		report.WriteString("\nProject Activity:\n")
		// Sort projects by activity
		type projectCount struct {
			name  string
			count int
		}
		var projects []projectCount
		for name, count := range projectCounts {
			projects = append(projects, projectCount{name, count})
		}
		sort.Slice(projects, func(i, j int) bool {
			return projects[i].count > projects[j].count
		})
		
		for i, proj := range projects {
			if i >= 10 { // Show top 10 projects
				break
			}
			percentage := float64(proj.count) / float64(len(oh.eventLog)) * 100
			report.WriteString(fmt.Sprintf("  %s: %d (%.1f%%)\n", proj.name, proj.count, percentage))
		}
	}
	
	// Recent activity (last hour)
	recentEvents := oh.getRecentEvents(time.Hour)
	if len(recentEvents) > 0 {
		report.WriteString(fmt.Sprintf("\nRecent Activity (last hour): %d events\n", len(recentEvents)))
		
		recentEventCounts := make(map[string]int)
		for _, event := range recentEvents {
			recentEventCounts[event.EventType]++
		}
		
		for eventType, count := range recentEventCounts {
			report.WriteString(fmt.Sprintf("  %s: %d\n", eventType, count))
		}
	}
	
	report.WriteString("==============================================\n")
	
	return report.String()
}

/**
 * CONTEXT:   Close output handler and cleanup resources
 * INPUT:     Cleanup request
 * OUTPUT:    Closed output handler with cleaned resources
 * BUSINESS:  Cleanup ensures proper resource management
 * CHANGE:    Initial cleanup implementation
 * RISK:      Low - Resource cleanup utility
 */
func (oh *OutputHandler) Close() {
	if oh.logFile != nil {
		oh.logFile.Close()
	}
}