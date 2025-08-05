package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-048
 * CONTEXT:   CLI manager implementation for user command processing
 * REASON:    Need concrete implementation of CLI operations for user interaction
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate user inputs and provide clear error messages
 * RISK:      Low - CLI errors are user-facing and don't affect daemon operation
 */

// TimePeriod represents different reporting time periods (mirrors arch.TimePeriod)
type TimePeriod string

const (
	PeriodDaily   TimePeriod = "daily"
	PeriodWeekly  TimePeriod = "weekly"
	PeriodMonthly TimePeriod = "monthly"
)

// StartConfig contains configuration options for daemon startup (mirrors arch.StartConfig)
type StartConfig struct {
	DatabasePath string `json:"databasePath"`
	LogLevel     string `json:"logLevel"`
	PidFile      string `json:"pidFile"`
}

// CLIManager interface for command-line operations
type CLIManager interface {
	ExecuteStart(config *StartConfig) error
	ExecuteStatus() (string, error)
	ExecuteReport(period TimePeriod) (string, error)
	ExecuteStop() error
}

// DefaultCLIManager implements the CLIManager interface
type DefaultCLIManager struct {
	logger arch.Logger
}

// NewDefaultCLIManager creates a new CLI manager
func NewDefaultCLIManager(logger arch.Logger) *DefaultCLIManager {
	return &DefaultCLIManager{
		logger: logger,
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-049
 * CONTEXT:   Daemon start command implementation with process management
 * REASON:    Need to start daemon process and manage PID file for process control
 * CHANGE:    Initial implementation.
 * PREVENTION:Check for existing daemon process and validate configuration before starting
 * RISK:      Medium - Multiple daemon instances could cause resource conflicts
 */

// ExecuteStart starts the daemon with optional configuration
func (dcm *DefaultCLIManager) ExecuteStart(config *StartConfig) error {
	dcm.logger.Info("Starting Claude Monitor daemon")
	
	// Check if daemon is already running
	if running, err := dcm.isDaemonRunning(config.PidFile); err != nil {
		return fmt.Errorf("failed to check daemon status: %w", err)
	} else if running {
		return fmt.Errorf("daemon is already running")
	}
	
	// Validate configuration
	if err := dcm.validateStartConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Start daemon process
	if err := dcm.startDaemonProcess(config); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	
	// Wait briefly and check if daemon started successfully
	time.Sleep(2 * time.Second)
	
	if running, err := dcm.isDaemonRunning(config.PidFile); err != nil {
		return fmt.Errorf("failed to verify daemon startup: %w", err)
	} else if !running {
		return fmt.Errorf("daemon failed to start")
	}
	
	fmt.Println("Claude Monitor daemon started successfully")
	return nil
}

// validateStartConfig validates the start configuration
func (dcm *DefaultCLIManager) validateStartConfig(config *StartConfig) error {
	// Validate database path directory exists
	if config.DatabasePath != "" {
		if err := os.MkdirAll(config.DatabasePath, 0755); err != nil {
			return fmt.Errorf("cannot create database directory: %w", err)
		}
	}
	
	// Validate log level
	validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	isValidLevel := false
	for _, level := range validLevels {
		if strings.ToUpper(config.LogLevel) == level {
			isValidLevel = true
			break
		}
	}
	if !isValidLevel {
		return fmt.Errorf("invalid log level: %s", config.LogLevel)
	}
	
	return nil
}

// startDaemonProcess starts the daemon as a background process
func (dcm *DefaultCLIManager) startDaemonProcess(config *StartConfig) error {
	// Find the daemon executable
	daemonPath, err := dcm.findDaemonExecutable()
	if err != nil {
		return fmt.Errorf("cannot find daemon executable: %w", err)
	}
	
	// Create command with configuration
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
	
	cmd := exec.Command(daemonPath, args...)
	
	// Start daemon in background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}
	
	dcm.logger.Info("Daemon process started", "pid", cmd.Process.Pid)
	return nil
}

// findDaemonExecutable locates the daemon binary
func (dcm *DefaultCLIManager) findDaemonExecutable() (string, error) {
	// Try different possible locations
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

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-050
 * CONTEXT:   Status command implementation with daemon communication
 * REASON:    Users need to check current monitoring status and daemon health
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle cases where daemon is not running gracefully
 * RISK:      Low - Status command failures don't affect daemon operation
 */

// ExecuteStatus returns formatted status information
func (dcm *DefaultCLIManager) ExecuteStatus() (string, error) {
	// Check if daemon is running
	running, err := dcm.isDaemonRunning("/var/run/claude-monitor.pid")
	if err != nil {
		return "", fmt.Errorf("failed to check daemon status: %w", err)
	}
	
	if !running {
		return "Claude Monitor daemon is not running\n", nil
	}
	
	// For now, return basic status. This would be enhanced to communicate with daemon
	status := &domain.SystemStatus{
		DaemonRunning:    true,
		MonitoringActive: true,
		EventsProcessed:  0,
		Uptime:          0, // Would be calculated from daemon
	}
	
	return dcm.formatStatus(status), nil
}

// formatStatus formats system status for display
func (dcm *DefaultCLIManager) formatStatus(status *domain.SystemStatus) string {
	var output strings.Builder
	
	output.WriteString("Claude Monitor Status\n")
	output.WriteString("====================\n\n")
	
	// Daemon status
	output.WriteString(fmt.Sprintf("Daemon Running: %v\n", status.DaemonRunning))
	output.WriteString(fmt.Sprintf("Monitoring Active: %v\n", status.MonitoringActive))
	
	if status.Uptime > 0 {
		output.WriteString(fmt.Sprintf("Uptime: %v\n", status.Uptime))
	}
	
	output.WriteString(fmt.Sprintf("Events Processed: %d\n\n", status.EventsProcessed))
	
	// Current session
	if status.CurrentSession != nil {
		output.WriteString("Current Session:\n")
		output.WriteString(fmt.Sprintf("  ID: %s\n", status.CurrentSession.ID))
		output.WriteString(fmt.Sprintf("  Start Time: %s\n", status.CurrentSession.StartTime.Format(time.RFC3339)))
		output.WriteString(fmt.Sprintf("  End Time: %s\n", status.CurrentSession.EndTime.Format(time.RFC3339)))
		output.WriteString("\n")
	} else {
		output.WriteString("No active session\n\n")
	}
	
	// Current work block
	if status.CurrentWorkBlock != nil {
		output.WriteString("Current Work Block:\n")
		output.WriteString(fmt.Sprintf("  ID: %s\n", status.CurrentWorkBlock.ID))
		output.WriteString(fmt.Sprintf("  Start Time: %s\n", status.CurrentWorkBlock.StartTime.Format(time.RFC3339)))
		output.WriteString(fmt.Sprintf("  Duration: %v\n", status.CurrentWorkBlock.Duration()))
		
		if status.LastActivity != nil {
			output.WriteString(fmt.Sprintf("  Last Activity: %s\n", status.LastActivity.Format(time.RFC3339)))
		}
		output.WriteString("\n")
	} else {
		output.WriteString("No active work block\n\n")
	}
	
	return output.String()
}

// ExecuteReport generates usage reports for specified period
func (dcm *DefaultCLIManager) ExecuteReport(period TimePeriod) (string, error) {
	// For now, return placeholder report. This would be enhanced to communicate with daemon
	stats := &domain.SessionStats{
		Period:         string(period),
		TotalSessions:  0,
		TotalWorkTime:  0,
		SessionCount:   0,
		WorkBlockCount: 0,
		StartDate:      time.Now().AddDate(0, 0, -1),
		EndDate:        time.Now(),
	}
	
	return dcm.formatReport(stats), nil
}

// formatReport formats session statistics for display
func (dcm *DefaultCLIManager) formatReport(stats *domain.SessionStats) string {
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("Claude Monitor Report (%s)\n", strings.Title(stats.Period)))
	output.WriteString("=======================\n\n")
	
	output.WriteString(fmt.Sprintf("Period: %s to %s\n\n", 
		stats.StartDate.Format("2006-01-02"), 
		stats.EndDate.Format("2006-01-02")))
	
	output.WriteString(fmt.Sprintf("Total Sessions: %d\n", stats.TotalSessions))
	output.WriteString(fmt.Sprintf("Total Work Time: %v\n", stats.TotalWorkTime))
	output.WriteString(fmt.Sprintf("Average Work Time: %v\n", stats.AverageWorkTime))
	output.WriteString(fmt.Sprintf("Work Blocks: %d\n\n", stats.WorkBlockCount))
	
	if stats.TotalSessions > 0 {
		avgSessionTime := stats.TotalWorkTime / time.Duration(stats.TotalSessions)
		output.WriteString(fmt.Sprintf("Average Session Duration: %v\n", avgSessionTime))
	}
	
	return output.String()
}

// ExecuteStop stops the running daemon
func (dcm *DefaultCLIManager) ExecuteStop() error {
	pidFile := "/var/run/claude-monitor.pid"
	
	// Check if daemon is running
	running, err := dcm.isDaemonRunning(pidFile)
	if err != nil {
		return fmt.Errorf("failed to check daemon status: %w", err)
	}
	
	if !running {
		fmt.Println("Claude Monitor daemon is not running")
		return nil
	}
	
	// Read PID and terminate daemon
	if err := dcm.stopDaemonProcess(pidFile); err != nil {
		return fmt.Errorf("failed to stop daemon: %w", err)
	}
	
	fmt.Println("Claude Monitor daemon stopped")
	return nil
}

// isDaemonRunning checks if daemon is running by checking PID file
func (dcm *DefaultCLIManager) isDaemonRunning(pidFile string) (bool, error) {
	// Check if PID file exists
	pidData, err := os.ReadFile(pidFile)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	
	// Parse PID
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
	if err := process.Signal(os.Signal(nil)); err != nil {
		return false, nil
	}
	
	return true, nil
}

// stopDaemonProcess stops the daemon process
func (dcm *DefaultCLIManager) stopDaemonProcess(pidFile string) error {
	// Read PID
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}
	
	pidStr := strings.TrimSpace(string(pidData))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID: %s", pidStr)
	}
	
	// Find and terminate process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}
	
	// Send SIGTERM for graceful shutdown
	if err := process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to signal process: %w", err)
	}
	
	// Wait for process to exit
	time.Sleep(2 * time.Second)
	
	// Check if process still exists
	if err := process.Signal(os.Signal(nil)); err == nil {
		// Process still running, force kill
		dcm.logger.Warn("Daemon did not exit gracefully, forcing termination")
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}
	
	// Remove PID file
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		dcm.logger.Warn("Failed to remove PID file", "error", err)
	}
	
	return nil
}