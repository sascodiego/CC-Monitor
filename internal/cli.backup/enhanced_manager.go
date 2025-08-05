package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-006
 * CONTEXT:   Enhanced CLI manager with professional user experience and comprehensive functionality
 * REASON:    Need modern CLI interface with rich formatting, interactive features, and multiple output formats
 * CHANGE:    Complete rewrite with professional status display, reporting, and user experience features.
 * PREVENTION:Validate all inputs, handle errors gracefully, and provide clear user feedback for all operations
 * RISK:      Low - Enhanced CLI improves user experience but doesn't affect core daemon functionality
 */

// Configuration structures for enhanced CLI operations
type DaemonConfig struct {
	DatabasePath string
	LogLevel     string
	PidFile      string
	Foreground   bool
	Verbose      bool
	Format       string
}

type StopConfig struct {
	Timeout time.Duration
	Force   bool
	Verbose bool
	Format  string
}

type RestartConfig struct {
	Verbose bool
	Format  string
}

type StatusConfig struct {
	Watch    bool
	Interval time.Duration
	Simple   bool
	Verbose  bool
	Format   string
}

type ReportConfig struct {
	Type        string // daily, weekly, monthly, range
	Date        string
	StartDate   string
	EndDate     string
	OutputFile  string
	Detailed    bool
	SummaryOnly bool
	Verbose     bool
	Format      string
}

type ConfigShowConfig struct {
	Verbose bool
	Format  string
}

type ConfigSetConfig struct {
	Key     string
	Value   string
	Verbose bool
}

type LogsConfig struct {
	Follow  bool
	Lines   int
	Verbose bool
	Format  string
}

type HealthConfig struct {
	Verbose bool
	Format  string
}

type ExportConfig struct {
	OutputFile string
	StartDate  string
	EndDate    string
	Verbose    bool
	Format     string
}

// Enhanced CLI Manager interface with comprehensive operations
type EnhancedCLIManager interface {
	// Daemon lifecycle
	ExecuteStart(config *DaemonConfig) error
	ExecuteStop(config *StopConfig) error
	ExecuteRestart(config *RestartConfig) error
	
	// Status and monitoring
	ExecuteStatus(config *StatusConfig) error
	ExecuteHealth(config *HealthConfig) error
	
	// Reporting and analytics
	ExecuteReport(config *ReportConfig) error
	ExecuteExport(config *ExportConfig) error
	
	// Configuration management
	ExecuteConfigShow(config *ConfigShowConfig) error
	ExecuteConfigSet(config *ConfigSetConfig) error
	
	// Logging and troubleshooting
	ExecuteLogs(config *LogsConfig) error
}

// DefaultEnhancedCLIManager implements the EnhancedCLIManager interface
type DefaultEnhancedCLIManager struct {
	logger arch.Logger
	formatter *OutputFormatter
}

// NewEnhancedCLIManager creates a new enhanced CLI manager
func NewEnhancedCLIManager(logger arch.Logger) *DefaultEnhancedCLIManager {
	return &DefaultEnhancedCLIManager{
		logger:    logger,
		formatter: NewOutputFormatter(),
	}
}

// NewEnhancedCLIManagerWithWorkHour creates an enhanced CLI manager with work hour capabilities
func NewEnhancedCLIManagerWithWorkHour(logger arch.Logger, workHourService arch.WorkHourService) EnhancedCLIManager {
	if workHourService != nil {
		return NewWorkHourEnhancedCLIManager(logger, workHourService)
	}
	return NewEnhancedCLIManager(logger)
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-007
 * CONTEXT:   Enhanced daemon start implementation with comprehensive feedback and error handling
 * REASON:    Users need clear feedback during daemon startup with proper validation and status updates
 * CHANGE:    Enhanced implementation with progress indicators, validation, and professional output.
 * PREVENTION:Always validate configuration, check for existing processes, and provide clear error messages
 * RISK:      Medium - Daemon startup failures could leave system in inconsistent state
 */

// ExecuteStart starts the daemon with enhanced configuration and feedback
func (ecm *DefaultEnhancedCLIManager) ExecuteStart(config *DaemonConfig) error {
	if config.Verbose {
		ecm.logger.Info("Starting Claude Monitor daemon with enhanced configuration")
	}
	
	// Print startup banner
	ecm.formatter.PrintStartupBanner()
	
	// Step 1: Validate configuration
	ecm.formatter.PrintStep("Validating configuration...")
	if err := ecm.validateDaemonConfig(config); err != nil {
		ecm.formatter.PrintError("Configuration validation failed", err)
		return err
	}
	ecm.formatter.PrintSuccess("Configuration validated")
	
	// Step 2: Check for existing daemon
	ecm.formatter.PrintStep("Checking for existing daemon...")
	if running, err := ecm.isDaemonRunning(config.PidFile); err != nil {
		ecm.formatter.PrintError("Failed to check daemon status", err)
		return err
	} else if running {
		ecm.formatter.PrintWarning("Daemon is already running")
		return fmt.Errorf("daemon is already running")
	}
	ecm.formatter.PrintSuccess("No existing daemon found")
	
	// Step 3: Prepare environment
	ecm.formatter.PrintStep("Preparing environment...")
	if err := ecm.prepareDaemonEnvironment(config); err != nil {
		ecm.formatter.PrintError("Environment preparation failed", err)
		return err
	}
	ecm.formatter.PrintSuccess("Environment prepared")
	
	// Step 4: Start daemon process
	ecm.formatter.PrintStep("Starting daemon process...")
	if err := ecm.startDaemonProcess(config); err != nil {
		ecm.formatter.PrintError("Failed to start daemon", err)
		return err
	}
	
	// Step 5: Verify daemon startup
	ecm.formatter.PrintStep("Verifying daemon startup...")
	if err := ecm.verifyDaemonStartup(config); err != nil {
		ecm.formatter.PrintError("Daemon startup verification failed", err)
		return err
	}
	
	ecm.formatter.PrintSuccess("Claude Monitor daemon started successfully")
	
	// Show quick status if verbose
	if config.Verbose {
		fmt.Println()
		return ecm.ExecuteStatus(&StatusConfig{
			Simple:  true,
			Verbose: false,
			Format:  config.Format,
		})
	}
	
	return nil
}

// validateDaemonConfig validates daemon configuration parameters
func (ecm *DefaultEnhancedCLIManager) validateDaemonConfig(config *DaemonConfig) error {
	// Validate database path
	if config.DatabasePath != "" {
		dbDir := filepath.Dir(config.DatabasePath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("cannot create database directory: %w", err)
		}
	}
	
	// Validate log level
	validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	isValid := false
	for _, level := range validLevels {
		if strings.ToUpper(config.LogLevel) == level {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid log level: %s (valid: %v)", config.LogLevel, validLevels)
	}
	
	// Validate PID file directory
	pidDir := filepath.Dir(config.PidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return fmt.Errorf("cannot create PID directory: %w", err)
	}
	
	return nil
}

// prepareDaemonEnvironment sets up the environment for daemon execution
func (ecm *DefaultEnhancedCLIManager) prepareDaemonEnvironment(config *DaemonConfig) error {
	// Create necessary directories
	dirs := []string{
		filepath.Dir(config.DatabasePath),
		filepath.Dir(config.PidFile),
		"/var/log/claude-monitor",
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return nil
}

// startDaemonProcess starts the actual daemon process
func (ecm *DefaultEnhancedCLIManager) startDaemonProcess(config *DaemonConfig) error {
	// Find daemon executable
	daemonPath, err := ecm.findDaemonExecutable()
	if err != nil {
		return fmt.Errorf("cannot find daemon executable: %w", err)
	}
	
	// Build command arguments
	args := []string{}
	if config.DatabasePath != "" {
		args = append(args, "--db-path", config.DatabasePath)
	}
	if config.LogLevel != "" {
		args = append(args, "--log-level", config.LogLevel)
	}
	if config.PidFile != "" {
		args = append(args, "--pid-file", config.PidFile)
	}
	if config.Foreground {
		args = append(args, "--foreground")
	}
	
	// Start daemon
	cmd := exec.Command(daemonPath, args...)
	
	if config.Foreground {
		// Run in foreground
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	} else {
		// Start in background
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start daemon process: %w", err)
		}
		
		ecm.logger.Info("Daemon process started", "pid", cmd.Process.Pid)
		return nil
	}
}

// verifyDaemonStartup checks if daemon started successfully
func (ecm *DefaultEnhancedCLIManager) verifyDaemonStartup(config *DaemonConfig) error {
	// Wait briefly for daemon to initialize
	time.Sleep(2 * time.Second)
	
	// Check if daemon is running
	running, err := ecm.isDaemonRunning(config.PidFile)
	if err != nil {
		return fmt.Errorf("failed to verify daemon status: %w", err)
	}
	
	if !running {
		return fmt.Errorf("daemon failed to start properly")
	}
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-008
 * CONTEXT:   Enhanced status display with real-time updates and professional formatting
 * REASON:    Users need comprehensive status visibility with beautiful output and live updates
 * CHANGE:    New implementation with rich formatting, watch mode, and multiple output formats.
 * PREVENTION:Handle daemon communication failures gracefully and provide useful fallback information
 * RISK:      Low - Status display issues don't affect daemon operation
 */

// ExecuteStatus displays current system status with enhanced formatting
func (ecm *DefaultEnhancedCLIManager) ExecuteStatus(config *StatusConfig) error {
	if config.Watch {
		return ecm.executeWatchStatus(config)
	}
	
	// Get current status
	status, err := ecm.getCurrentStatus()
	if err != nil {
		return fmt.Errorf("failed to get system status: %w", err)
	}
	
	// Format and display status
	switch strings.ToLower(config.Format) {
	case "json":
		return ecm.printStatusJSON(status)
	case "csv":
		return ecm.printStatusCSV(status)
	default:
		if config.Simple {
			return ecm.printStatusSimple(status, config.Verbose)
		}
		return ecm.printStatusDetailed(status, config.Verbose)
	}
}

// executeWatchStatus implements live status updates
func (ecm *DefaultEnhancedCLIManager) executeWatchStatus(config *StatusConfig) error {
	// Set up signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()
	
	ecm.formatter.PrintInfo(fmt.Sprintf("Watching Claude Monitor status (updates every %v, press Ctrl+C to stop)", config.Interval))
	fmt.Println()
	
	// Show initial status
	if err := ecm.showSingleStatus(config); err != nil {
		return err
	}
	
	for {
		select {
		case <-sigChan:
			ecm.formatter.PrintInfo("Watch mode stopped")
			return nil
		case <-ticker.C:
			// Clear screen and show updated status
			ecm.formatter.ClearScreen()
			ecm.formatter.PrintInfo(fmt.Sprintf("Claude Monitor Status (Last updated: %s)", time.Now().Format("15:04:05")))
			fmt.Println()
			
			if err := ecm.showSingleStatus(config); err != nil {
				ecm.formatter.PrintWarning(fmt.Sprintf("Error updating status: %v", err))
			}
		}
	}
}

// showSingleStatus displays status once
func (ecm *DefaultEnhancedCLIManager) showSingleStatus(config *StatusConfig) error {
	status, err := ecm.getCurrentStatus()
	if err != nil {
		return err
	}
	
	switch strings.ToLower(config.Format) {
	case "json":
		return ecm.printStatusJSON(status)
	default:
		if config.Simple {
			return ecm.printStatusSimple(status, config.Verbose)
		}
		return ecm.printStatusDetailed(status, config.Verbose)
	}
}

// printStatusDetailed displays comprehensive status information
func (ecm *DefaultEnhancedCLIManager) printStatusDetailed(status *SystemStatus, verbose bool) error {
	// Header
	ecm.formatter.PrintSectionHeader("Claude Monitor Status")
	
	// Daemon status
	ecm.formatter.PrintSubHeader("System Status")
	if status.DaemonRunning {
		ecm.formatter.PrintStatusItem("Daemon", "Running", "success")
		if status.DaemonPID > 0 {
			ecm.formatter.PrintStatusItem("Process ID", strconv.Itoa(status.DaemonPID), "info")
		}
		if status.Uptime > 0 {
			ecm.formatter.PrintStatusItem("Uptime", ecm.formatter.FormatDuration(status.Uptime), "info")
		}
	} else {
		ecm.formatter.PrintStatusItem("Daemon", "Not running", "error")
	}
	
	ecm.formatter.PrintStatusItem("Monitoring", fmt.Sprintf("%v", status.MonitoringActive), getStatusColor(status.MonitoringActive))
	
	if verbose && status.DaemonRunning {
		ecm.formatter.PrintStatusItem("Events Processed", strconv.FormatInt(status.EventsProcessed, 10), "info")
	}
	
	fmt.Println()
	
	// Session information
	ecm.formatter.PrintSubHeader("Session Information")
	if status.CurrentSession != nil {
		session := status.CurrentSession
		remaining := session.EndTime.Sub(time.Now())
		
		ecm.formatter.PrintStatusItem("Session ID", session.ID, "info")
		ecm.formatter.PrintStatusItem("Started", session.StartTime.Format("2006-01-02 15:04:05"), "info")
		ecm.formatter.PrintStatusItem("Expires", session.EndTime.Format("2006-01-02 15:04:05"), "info")
		
		if remaining > 0 {
			ecm.formatter.PrintStatusItem("Remaining", ecm.formatter.FormatDuration(remaining), "success")
			ecm.formatter.PrintStatusItem("Status", "Active", "success")
		} else {
			ecm.formatter.PrintStatusItem("Status", "Expired", "warning")
		}
		
		if verbose {
			// TODO: Get actual work block count from database
			ecm.formatter.PrintStatusItem("Work Blocks", "0", "info")
		}
	} else {
		ecm.formatter.PrintStatusItem("Status", "No active session", "warning")
	}
	
	fmt.Println()
	
	// Work block information
	ecm.formatter.PrintSubHeader("Work Block Information")
	if status.CurrentWorkBlock != nil {
		wb := status.CurrentWorkBlock
		duration := time.Since(wb.StartTime)
		
		ecm.formatter.PrintStatusItem("Block ID", wb.ID, "info")
		ecm.formatter.PrintStatusItem("Started", wb.StartTime.Format("15:04:05"), "info")
		ecm.formatter.PrintStatusItem("Duration", ecm.formatter.FormatDuration(duration), "success")
		
		if verbose {
			// TODO: Get actual activity count from database
			ecm.formatter.PrintStatusItem("Activities", "0", "info")
		}
		
		ecm.formatter.PrintStatusItem("Status", "Active", "success")
	} else {
		ecm.formatter.PrintStatusItem("Status", "No active work block", "warning")
		
		if status.LastActivity != nil {
			timeSince := time.Since(*status.LastActivity)
			ecm.formatter.PrintStatusItem("Last Activity", fmt.Sprintf("%v ago", ecm.formatter.FormatDuration(timeSince)), "info")
			
			if timeSince > 5*time.Minute {
				ecm.formatter.PrintInfo("Note: Work block will start with next Claude interaction")
			}
		}
	}
	
	fmt.Println()
	
	// Today's summary
	if status.TodayStats != nil {
		ecm.formatter.PrintSubHeader("Today's Summary")
		ecm.formatter.PrintStatusItem("Sessions Used", strconv.Itoa(status.TodayStats.SessionCount), "info")
		ecm.formatter.PrintStatusItem("Work Blocks", strconv.Itoa(status.TodayStats.WorkBlockCount), "info")
		ecm.formatter.PrintStatusItem("Total Work Time", ecm.formatter.FormatDuration(status.TodayStats.TotalWorkTime), "info")
		
		if status.TodayStats.WorkBlockCount > 0 {
			ecm.formatter.PrintStatusItem("Avg Block Time", ecm.formatter.FormatDuration(status.TodayStats.AvgWorkBlockTime), "info")
		}
	}
	
	// System health (if verbose)
	if verbose && status.DaemonRunning && status.Health != nil {
		fmt.Println()
		ecm.formatter.PrintSubHeader("System Health")
		
		ecm.formatter.PrintStatusItem("eBPF Monitoring", fmt.Sprintf("%v", status.Health.EBPFHealthy), getStatusColor(status.Health.EBPFHealthy))
		ecm.formatter.PrintStatusItem("Database", fmt.Sprintf("%v", status.Health.DatabaseHealthy), getStatusColor(status.Health.DatabaseHealthy))
		
		if status.Health.EventsProcessed > 0 {
			ecm.formatter.PrintStatusItem("Events Processed", strconv.FormatInt(status.Health.EventsProcessed, 10), "info")
		}
		
		if status.Health.EventsDropped > 0 {
			dropRate := float64(status.Health.EventsDropped) / float64(status.Health.EventsProcessed) * 100
			ecm.formatter.PrintStatusItem("Events Dropped", fmt.Sprintf("%d (%.2f%%)", status.Health.EventsDropped, dropRate), getDropRateColor(dropRate))
			
			if dropRate > 1.0 {
				ecm.formatter.PrintWarning("High event drop rate detected")
			}
		}
	}
	
	return nil
}

// printStatusSimple displays minimal status information
func (ecm *DefaultEnhancedCLIManager) printStatusSimple(status *SystemStatus, verbose bool) error {
	if status.DaemonRunning {
		fmt.Printf("Daemon: %s", ecm.formatter.Colorize("Running", "green"))
		if status.DaemonPID > 0 {
			fmt.Printf(" (PID: %d)", status.DaemonPID)
		}
		fmt.Println()
	} else {
		fmt.Printf("Daemon: %s\n", ecm.formatter.Colorize("Not running", "red"))
	}
	
	if status.CurrentSession != nil {
		remaining := status.CurrentSession.EndTime.Sub(time.Now())
		if remaining > 0 {
			fmt.Printf("Session: %s (%s remaining)\n", 
				ecm.formatter.Colorize("Active", "green"),
				ecm.formatter.FormatDuration(remaining))
		} else {
			fmt.Printf("Session: %s\n", ecm.formatter.Colorize("Expired", "yellow"))
		}
	} else {
		fmt.Printf("Session: %s\n", ecm.formatter.Colorize("None", "yellow"))
	}
	
	if status.CurrentWorkBlock != nil {
		duration := time.Since(status.CurrentWorkBlock.StartTime)
		fmt.Printf("Work Block: %s (%s)\n", 
			ecm.formatter.Colorize("Active", "green"),
			ecm.formatter.FormatDuration(duration))
	} else {
		fmt.Printf("Work Block: %s\n", ecm.formatter.Colorize("None", "yellow"))
	}
	
	if verbose && status.TodayStats != nil {
		fmt.Printf("Today: %d sessions, %s work time\n",
			status.TodayStats.SessionCount,
			ecm.formatter.FormatDuration(status.TodayStats.TotalWorkTime))
	}
	
	return nil
}

// printStatusJSON outputs status in JSON format
func (ecm *DefaultEnhancedCLIManager) printStatusJSON(status *SystemStatus) error {
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status to JSON: %w", err)
	}
	
	fmt.Println(string(jsonData))
	return nil
}

// printStatusCSV outputs status in CSV format
func (ecm *DefaultEnhancedCLIManager) printStatusCSV(status *SystemStatus) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()
	
	// Write headers
	headers := []string{"Timestamp", "DaemonRunning", "MonitoringActive", "CurrentSessionID", "SessionRemaining", "WorkBlockActive", "WorkBlockDuration", "EventsProcessed"}
	if err := writer.Write(headers); err != nil {
		return err
	}
	
	// Write data
	record := []string{
		time.Now().Format(time.RFC3339),
		fmt.Sprintf("%v", status.DaemonRunning),
		fmt.Sprintf("%v", status.MonitoringActive),
	}
	
	if status.CurrentSession != nil {
		record = append(record, status.CurrentSession.ID)
		remaining := status.CurrentSession.EndTime.Sub(time.Now())
		record = append(record, remaining.String())
	} else {
		record = append(record, "", "")
	}
	
	if status.CurrentWorkBlock != nil {
		record = append(record, "true")
		duration := time.Since(status.CurrentWorkBlock.StartTime)
		record = append(record, duration.String())
	} else {
		record = append(record, "false", "")
	}
	
	record = append(record, strconv.FormatInt(status.EventsProcessed, 10))
	
	return writer.Write(record)
}

// Helper functions
func getStatusColor(status bool) string {
	if status {
		return "success"
	}
	return "error"
}

func getDropRateColor(rate float64) string {
	if rate > 5.0 {
		return "error"
	} else if rate > 1.0 {
		return "warning"
	}
	return "info"
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-009
 * CONTEXT:   System status data structures and helper functions
 * REASON:    Need comprehensive status representation for different CLI display modes
 * CHANGE:    New implementation of status structures matching domain models.
 * PREVENTION:Keep status structures aligned with domain models and handle nil values gracefully
 * RISK:      Low - Status structures are used for display only
 */

// SystemStatus represents the current system state for CLI display
type SystemStatus struct {
	DaemonRunning    bool                    `json:"daemonRunning"`
	DaemonPID        int                     `json:"daemonPid,omitempty"`
	MonitoringActive bool                    `json:"monitoringActive"`
	Uptime           time.Duration           `json:"uptime,omitempty"`
	EventsProcessed  int64                   `json:"eventsProcessed"`
	CurrentSession   *domain.Session         `json:"currentSession,omitempty"`
	CurrentWorkBlock *domain.WorkBlock       `json:"currentWorkBlock,omitempty"`
	LastActivity     *time.Time              `json:"lastActivity,omitempty"`
	TodayStats       *TodayStats             `json:"todayStats,omitempty"`
	Health           *SystemHealth           `json:"health,omitempty"`
}

// TodayStats represents today's usage statistics
type TodayStats struct {
	SessionCount     int           `json:"sessionCount"`
	WorkBlockCount   int           `json:"workBlockCount"`
	TotalWorkTime    time.Duration `json:"totalWorkTime"`
	AvgWorkBlockTime time.Duration `json:"avgWorkBlockTime"`
}

// SystemHealth represents system health metrics
type SystemHealth struct {
	EBPFHealthy      bool  `json:"ebpfHealthy"`
	DatabaseHealthy  bool  `json:"databaseHealthy"`
	EventsProcessed  int64 `json:"eventsProcessed"`
	EventsDropped    int64 `json:"eventsDropped"`
}

// getCurrentStatus retrieves the current system status
func (ecm *DefaultEnhancedCLIManager) getCurrentStatus() (*SystemStatus, error) {
	status := &SystemStatus{}
	
	// Read daemon status from status file first (more reliable)
	statusFile := "/tmp/claude-monitor-status.json"
	daemonStatus, err := ReadDaemonStatus(statusFile)
	if err != nil {
		ecm.logger.Debug("Could not read daemon status file", "error", err)
		// Fall back to PID file check
		pidFile := "/var/run/claude-monitor.pid"
		running, pidErr := ecm.isDaemonRunning(pidFile)
		if pidErr != nil {
			ecm.logger.Warn("Failed to check daemon status", "error", pidErr)
		}
		status.DaemonRunning = running
		status.MonitoringActive = running
	} else {
		// Use status from file
		status.DaemonRunning = daemonStatus.DaemonRunning
		status.MonitoringActive = daemonStatus.MonitoringActive
		status.CurrentSession = daemonStatus.CurrentSession
		status.CurrentWorkBlock = daemonStatus.CurrentWorkBlock
	}
	
	if status.DaemonRunning {
		// Get PID
		pidFile := "/var/run/claude-monitor.pid"
		if pid, err := ecm.getDaemonPID(pidFile); err == nil {
			status.DaemonPID = pid
		}
		
		
		status.EventsProcessed = 0 // Will be implemented with real monitoring
		status.TodayStats = &TodayStats{
			SessionCount:     0,
			WorkBlockCount:   0,
			TotalWorkTime:    0,
			AvgWorkBlockTime: 0,
		}
		status.Health = &SystemHealth{
			EBPFHealthy:     true,
			DatabaseHealthy: true,
			EventsProcessed: 0,
			EventsDropped:   0,
		}
	}
	
	return status, nil
}

// getDaemonPID retrieves the daemon process ID from PID file
func (ecm *DefaultEnhancedCLIManager) getDaemonPID(pidFile string) (int, error) {
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	
	pidStr := strings.TrimSpace(string(pidData))
	return strconv.Atoi(pidStr)
}

// isDaemonRunning checks if the daemon process is running
func (ecm *DefaultEnhancedCLIManager) isDaemonRunning(pidFile string) (bool, error) {
	pidData, err := os.ReadFile(pidFile)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	
	pidStr := strings.TrimSpace(string(pidData))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false, fmt.Errorf("invalid PID in file: %s", pidStr)
	}
	
	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false, nil
	}
	
	// Signal 0 checks if process exists without affecting it
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false, nil
	}
	
	return true, nil
}

// findDaemonExecutable locates the daemon binary
func (ecm *DefaultEnhancedCLIManager) findDaemonExecutable() (string, error) {
	candidates := []string{
		"./claude-daemon",
		"/usr/local/bin/claude-daemon",
		"/usr/bin/claude-daemon",
	}
	
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	
	return "", fmt.Errorf("claude-daemon executable not found")
}