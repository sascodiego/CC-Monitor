//go:build linux

/**
 * CONTEXT:   Linux systemd service integration for Claude Monitor
 * INPUT:     Service configuration, systemctl commands, systemd unit management
 * OUTPUT:    Professional Linux service with systemd integration and journal logging
 * BUSINESS:  Linux service integration enables enterprise deployment on Linux systems
 * CHANGE:    Applied Dependency Inversion Principle with injected interfaces
 * RISK:      High - systemd integration requiring proper unit file management
 */

package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

/**
 * CONTEXT:   Linux systemd service manager implementation with dependency injection
 * INPUT:     Service configuration, systemctl operations, unit file management
 * OUTPUT:    Complete Linux service lifecycle management with systemd
 * BUSINESS:  LinuxServiceManager provides professional Linux service integration
 * CHANGE:    Applied dependency injection for testability and flexibility
 * RISK:      High - systemd integration requires careful unit file management
 */
type LinuxServiceManager struct {
	serviceName   string
	isUserService bool
	unitFilePath  string
	commandExec   CommandExecutor
	fileSystem    FileSystemProvider
}

// Constructor with dependency injection
func NewLinuxServiceManagerWithDeps(cmdExec CommandExecutor, fs FileSystemProvider) (*LinuxServiceManager, error) {
	serviceName := "claude-monitor"
	isUserService := serviceUserLevel
	
	var unitFilePath string
	if isUserService {
		homeDir, err := fs.GetUserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		unitFilePath = filepath.Join(homeDir, ".config/systemd/user", serviceName+".service")
	} else {
		unitFilePath = filepath.Join("/etc/systemd/system", serviceName+".service")
	}
	
	return &LinuxServiceManager{
		serviceName:   serviceName,
		isUserService: isUserService,
		unitFilePath:  unitFilePath,
		commandExec:   cmdExec,
		fileSystem:    fs,
	}, nil
}

// Legacy constructor for backward compatibility
func NewLinuxServiceManager() (*LinuxServiceManager, error) {
	// Use default implementations for backward compatibility
	return NewLinuxServiceManagerWithDeps(&DefaultCommandExecutor{}, &DefaultFileSystemProvider{})
}

// Original constructor logic preserved for reference
func newLinuxServiceManagerLegacy() (*LinuxServiceManager, error) {
	serviceName := "claude-monitor"
	isUserService := serviceUserLevel
	
	var unitFilePath string
	if isUserService {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		
		// Create user systemd directory if needed
		userSystemdDir := filepath.Join(homeDir, ".config", "systemd", "user")
		if err := os.MkdirAll(userSystemdDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create user systemd directory: %w", err)
		}
		
		unitFilePath = filepath.Join(userSystemdDir, serviceName+".service")
	} else {
		unitFilePath = filepath.Join("/etc/systemd/system", serviceName+".service")
	}
	
	return &LinuxServiceManager{
		serviceName:   serviceName,
		isUserService: isUserService,
		unitFilePath:  unitFilePath,
	}, nil
}

/**
 * CONTEXT:   Linux service installation with systemd unit file creation
 * INPUT:     Service configuration with Linux-specific systemd settings
 * OUTPUT:    Installed systemd service with proper unit file and permissions
 * BUSINESS:  Service installation enables professional Linux deployment
 * CHANGE:    Initial service installation with systemd unit generation
 * RISK:      High - systemd unit file creation requires proper syntax and permissions
 */
func (l *LinuxServiceManager) Install(config ServiceConfig) error {
	// Check if service already exists
	if l.IsInstalled() {
		return fmt.Errorf("service '%s' already exists at %s", config.Name, l.unitFilePath)
	}
	
	// Generate systemd unit file content
	unitContent, err := l.generateSystemdUnit(config)
	if err != nil {
		return fmt.Errorf("failed to generate systemd unit: %w", err)
	}
	
	// Write unit file using injected file system
	if err := l.fileSystem.WriteFile(l.unitFilePath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}
	
	// Reload systemd daemon
	if err := l.systemctlCommand("daemon-reload"); err != nil {
		// Clean up on failure using injected file system
		l.fileSystem.RemoveFile(l.unitFilePath)
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	
	// Enable service if auto-start configured
	if config.StartMode == StartModeAuto {
		if err := l.systemctlCommand("enable", config.Name); err != nil {
			return fmt.Errorf("failed to enable service: %w", err)
		}
	}
	
	// Create log directory if needed
	if config.WorkingDir != "" {
		logsDir := filepath.Join(config.WorkingDir, "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			return fmt.Errorf("failed to create logs directory: %w", err)
		}
		
		// Set ownership if service runs as different user
		if config.User != "" && !l.isUserService {
			if err := l.setDirectoryOwnership(logsDir, config.User, config.Group); err != nil {
				// Log warning but don't fail
				fmt.Printf("Warning: failed to set log directory ownership: %v\n", err)
			}
		}
	}
	
	return nil
}

func (l *LinuxServiceManager) Uninstall() error {
	if !l.IsInstalled() {
		return fmt.Errorf("service is not installed")
	}
	
	// Stop service if running
	if l.IsRunning() {
		if err := l.systemctlCommand("stop", l.serviceName); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
	}
	
	// Disable service
	if err := l.systemctlCommand("disable", l.serviceName); err != nil {
		// Continue even if disable fails
		fmt.Printf("Warning: failed to disable service: %v\n", err)
	}
	
	// Remove unit file using injected file system
	if err := l.fileSystem.RemoveFile(l.unitFilePath); err != nil {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}
	
	// Reload systemd daemon
	if err := l.systemctlCommand("daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	
	// Reset failed state
	l.systemctlCommand("reset-failed", l.serviceName)
	
	return nil
}

func (l *LinuxServiceManager) Start() error {
	if !l.IsInstalled() {
		return fmt.Errorf("service is not installed")
	}
	
	if err := l.systemctlCommand("start", l.serviceName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	
	// Wait for service to be active
	if err := l.waitForServiceState("active", 30*time.Second); err != nil {
		return fmt.Errorf("service failed to start: %w", err)
	}
	
	return nil
}

func (l *LinuxServiceManager) Stop() error {
	if !l.IsInstalled() {
		return fmt.Errorf("service is not installed")
	}
	
	if err := l.systemctlCommand("stop", l.serviceName); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	
	// Wait for service to be inactive
	if err := l.waitForServiceState("inactive", 30*time.Second); err != nil {
		return fmt.Errorf("service failed to stop: %w", err)
	}
	
	return nil
}

func (l *LinuxServiceManager) Restart() error {
	if !l.IsInstalled() {
		return fmt.Errorf("service is not installed")
	}
	
	if err := l.systemctlCommand("restart", l.serviceName); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}
	
	// Wait for service to be active
	if err := l.waitForServiceState("active", 30*time.Second); err != nil {
		return fmt.Errorf("service failed to restart: %w", err)
	}
	
	return nil
}

/**
 * CONTEXT:   Linux service status query with systemd integration
 * INPUT:     systemctl status commands and service information parsing
 * OUTPUT:    Comprehensive service status with Linux-specific metrics
 * BUSINESS:  Service status enables monitoring and troubleshooting
 * CHANGE:    Initial status implementation with systemd status parsing
 * RISK:      Medium - Status parsing requires proper systemctl output handling
 */
func (l *LinuxServiceManager) Status() (ServiceStatus, error) {
	if !l.IsInstalled() {
		return ServiceStatus{}, fmt.Errorf("service is not installed")
	}
	
	// Get service status
	output, err := l.systemctlOutput("status", l.serviceName)
	if err != nil {
		// Service might be in failed state - still parse output
		if strings.Contains(err.Error(), "failed") {
			output = err.Error()
		} else {
			return ServiceStatus{}, fmt.Errorf("failed to get service status: %w", err)
		}
	}
	
	status := ServiceStatus{
		Name:        l.serviceName,
		DisplayName: "Claude Monitor Work Tracking Service",
	}
	
	// Parse systemctl status output
	status.State = l.parseServiceState(output)
	status.PID = l.parsePID(output)
	status.StartTime = l.parseStartTime(output)
	
	if !status.StartTime.IsZero() {
		status.Uptime = time.Since(status.StartTime)
	}
	
	// Get additional metrics if service is running
	if status.State == ServiceStateRunning && status.PID > 0 {
		if metrics, err := l.getProcessMetrics(status.PID); err == nil {
			status.Memory = metrics.Memory
			status.CPU = metrics.CPU
		}
	}
	
	// Get last error if any
	status.LastError = l.parseLastError(output)
	
	return status, nil
}

func (l *LinuxServiceManager) IsInstalled() bool {
	return l.fileSystem.FileExists(l.unitFilePath)
}

func (l *LinuxServiceManager) IsRunning() bool {
	output, err := l.systemctlOutput("is-active", l.serviceName)
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) == "active"
}

/**
 * CONTEXT:   Linux journal log retrieval for service logging
 * INPUT:     journalctl commands and service log filtering
 * OUTPUT:    Structured log entries from systemd journal
 * BUSINESS:  Log retrieval enables troubleshooting and monitoring
 * CHANGE:    Initial log retrieval with journalctl integration
 * RISK:      Medium - Journal parsing requires proper command handling
 */
func (l *LinuxServiceManager) GetLogs(lines int) ([]LogEntry, error) {
	if !l.IsInstalled() {
		return nil, fmt.Errorf("service is not installed")
	}
	
	// Build journalctl command
	args := []string{
		"journalctl",
		"--unit=" + l.serviceName,
		"--lines=" + strconv.Itoa(lines),
		"--output=json",
		"--no-pager",
	}
	
	if l.isUserService {
		args = append(args, "--user")
	}
	
	// Execute journalctl command using injected executor
	output, err := l.commandExec.Execute(args[0], args[1:]...)
	if err != nil {
		return nil, fmt.Errorf("failed to get journal logs: %w", err)
	}
	
	// Parse journal output
	logs, err := l.parseJournalOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse journal output: %w", err)
	}
	
	return logs, nil
}

/**
 * CONTEXT:   systemd unit file generation from service configuration
 * INPUT:     Service configuration with Linux-specific settings
 * OUTPUT:    Complete systemd unit file with proper syntax and options
 * BUSINESS:  Unit file generation enables reliable systemd service operation
 * CHANGE:    Initial unit file generation with security and reliability options
 * RISK:      High - Unit file syntax errors can prevent service operation
 */
func (l *LinuxServiceManager) generateSystemdUnit(config ServiceConfig) (string, error) {
	var unit strings.Builder
	
	// [Unit] section
	unit.WriteString("[Unit]\n")
	unit.WriteString(fmt.Sprintf("Description=%s\n", config.Description))
	unit.WriteString("Documentation=https://github.com/claude-monitor/system/system\n")
	
	// Dependencies
	if len(config.LinuxService.After) > 0 {
		unit.WriteString(fmt.Sprintf("After=%s\n", strings.Join(config.LinuxService.After, " ")))
	} else {
		unit.WriteString("After=network.target\n")
	}
	
	if len(config.LinuxService.Before) > 0 {
		unit.WriteString(fmt.Sprintf("Before=%s\n", strings.Join(config.LinuxService.Before, " ")))
	}
	
	if len(config.LinuxService.RequiredBy) > 0 {
		unit.WriteString(fmt.Sprintf("RequiredBy=%s\n", strings.Join(config.LinuxService.RequiredBy, " ")))
	} else {
		unit.WriteString("Wants=network.target\n")
	}
	
	unit.WriteString("\n")
	
	// [Service] section
	unit.WriteString("[Service]\n")
	unit.WriteString("Type=simple\n")
	
	// User and group
	if config.User != "" && !l.isUserService {
		unit.WriteString(fmt.Sprintf("User=%s\n", config.User))
	}
	if config.Group != "" && !l.isUserService {
		unit.WriteString(fmt.Sprintf("Group=%s\n", config.Group))
	}
	
	// Executable and arguments
	execStart := config.ExecutablePath
	if len(config.Arguments) > 0 {
		execStart += " " + strings.Join(config.Arguments, " ")
	}
	unit.WriteString(fmt.Sprintf("ExecStart=%s\n", execStart))
	
	// Working directory
	if config.WorkingDir != "" {
		unit.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", config.WorkingDir))
	}
	
	// Restart configuration
	if config.RestartOnFailure {
		unit.WriteString("Restart=on-failure\n")
		unit.WriteString(fmt.Sprintf("RestartSec=%d\n", int(config.RestartDelay.Seconds())))
	} else {
		unit.WriteString("Restart=no\n")
	}
	
	// Environment variables
	for key, value := range config.Environment {
		unit.WriteString(fmt.Sprintf("Environment=\"%s=%s\"\n", key, value))
	}
	
	// Process management
	unit.WriteString("ExecReload=/bin/kill -HUP $MAINPID\n")
	unit.WriteString("KillMode=mixed\n")
	unit.WriteString("TimeoutStopSec=30\n")
	
	// Security settings (for system services)
	if !l.isUserService {
		unit.WriteString("\n# Security settings\n")
		unit.WriteString("NoNewPrivileges=yes\n")
		unit.WriteString("PrivateTmp=yes\n")
		unit.WriteString("ProtectSystem=strict\n")
		unit.WriteString("ProtectHome=read-only\n")
		
		// Read/write paths
		if config.WorkingDir != "" {
			unit.WriteString(fmt.Sprintf("ReadWritePaths=%s\n", config.WorkingDir))
		}
		
		// Add database directory to read-write paths
		dbPath := filepath.Join(config.WorkingDir, "data")
		unit.WriteString(fmt.Sprintf("ReadWritePaths=%s\n", dbPath))
		
		// Capabilities if specified
		if len(config.LinuxService.Capabilities) > 0 {
			unit.WriteString(fmt.Sprintf("CapabilityBoundingSet=%s\n", strings.Join(config.LinuxService.Capabilities, " ")))
			unit.WriteString(fmt.Sprintf("AmbientCapabilities=%s\n", strings.Join(config.LinuxService.Capabilities, " ")))
		}
		
		// Additional hardening
		unit.WriteString("ProtectKernelTunables=yes\n")
		unit.WriteString("ProtectKernelModules=yes\n")
		unit.WriteString("ProtectControlGroups=yes\n")
		unit.WriteString("RestrictRealtime=yes\n")
		unit.WriteString("RestrictNamespaces=yes\n")
	}
	
	unit.WriteString("\n")
	
	// [Install] section
	unit.WriteString("[Install]\n")
	if len(config.LinuxService.WantedBy) > 0 {
		unit.WriteString(fmt.Sprintf("WantedBy=%s\n", strings.Join(config.LinuxService.WantedBy, " ")))
	} else {
		if l.isUserService {
			unit.WriteString("WantedBy=default.target\n")
		} else {
			unit.WriteString("WantedBy=multi-user.target\n")
		}
	}
	
	return unit.String(), nil
}

/**
 * CONTEXT:   Execute systemctl command with injected executor
 * INPUT:     systemctl command arguments
 * OUTPUT:    Command execution result with error handling
 * BUSINESS:  Systemctl operations using dependency injection for testability
 * CHANGE:    Applied dependency injection for command execution
 * RISK:      Medium - System service commands affecting service state
 */
func (l *LinuxServiceManager) systemctlCommand(args ...string) error {
	cmdArgs := l.buildSystemctlArgs(args...)
	_, err := l.commandExec.Execute(cmdArgs[0], cmdArgs[1:]...)
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w", strings.Join(args, " "), err)
	}
	return nil
}

/**
 * CONTEXT:   Execute systemctl command and capture output with injected executor
 * INPUT:     systemctl command arguments
 * OUTPUT:    Command output and error handling
 * BUSINESS:  Systemctl queries using dependency injection for testability
 * CHANGE:    Applied dependency injection for command output capture
 * RISK:      Medium - System service queries affecting status reporting
 */
func (l *LinuxServiceManager) systemctlOutput(args ...string) (string, error) {
	cmdArgs := l.buildSystemctlArgs(args...)
	output, err := l.commandExec.Execute(cmdArgs[0], cmdArgs[1:]...)
	if err != nil {
		return string(output), fmt.Errorf("systemctl %s failed: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}

/**
 * CONTEXT:   Build systemctl command arguments with user service handling
 * INPUT:     systemctl command arguments
 * OUTPUT:    Complete command arguments array ready for execution
 * BUSINESS:  Command argument building for both user and system services
 * CHANGE:    Extracted argument building from command construction
 * RISK:      Low - Command argument preparation without execution
 */
func (l *LinuxServiceManager) buildSystemctlArgs(args ...string) []string {
	cmdArgs := make([]string, 0, len(args)+2)
	cmdArgs = append(cmdArgs, "systemctl")
	
	if l.isUserService {
		cmdArgs = append(cmdArgs, "--user")
	}
	
	cmdArgs = append(cmdArgs, args...)
	return cmdArgs
}

// Legacy method for backward compatibility
func (l *LinuxServiceManager) buildSystemctlCommand(args ...string) *exec.Cmd {
	cmdArgs := l.buildSystemctlArgs(args...)
	// Note: This method returns *exec.Cmd for external use
	// For internal use, prefer l.commandExec.Execute() for testability
	return exec.Command(cmdArgs[0], cmdArgs[1:]...)
}

func (l *LinuxServiceManager) waitForServiceState(targetState string, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		output, err := l.systemctlOutput("is-active", l.serviceName)
		if err == nil && strings.TrimSpace(output) == targetState {
			return nil
		}
		
		time.Sleep(500 * time.Millisecond)
	}
	
	return fmt.Errorf("timeout waiting for service state %s", targetState)
}

func (l *LinuxServiceManager) parseServiceState(output string) ServiceState {
	if strings.Contains(output, "Active: active (running)") {
		return ServiceStateRunning
	}
	if strings.Contains(output, "Active: inactive (dead)") {
		return ServiceStateStopped
	}
	if strings.Contains(output, "Active: activating") {
		return ServiceStateStarting
	}
	if strings.Contains(output, "Active: deactivating") {
		return ServiceStateStopping
	}
	if strings.Contains(output, "Active: failed") {
		return ServiceStateFailed
	}
	return ServiceStateUnknown
}

func (l *LinuxServiceManager) parsePID(output string) int {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Main PID:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "PID:" && i+1 < len(parts) {
					if pid, err := strconv.Atoi(parts[i+1]); err == nil {
						return pid
					}
				}
			}
		}
	}
	return 0
}

func (l *LinuxServiceManager) parseStartTime(output string) time.Time {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Active: active (running) since") {
			// Parse timestamp from line like: "Active: active (running) since Wed 2024-08-06 10:30:45 UTC; 2h 15min ago"
			parts := strings.Split(line, " since ")
			if len(parts) > 1 {
				timePart := strings.Split(parts[1], ";")[0]
				if startTime, err := time.Parse("Mon 2006-01-02 15:04:05 MST", timePart); err == nil {
					return startTime
				}
			}
		}
	}
	return time.Time{}
}

func (l *LinuxServiceManager) parseLastError(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "failed") && !strings.Contains(line, "Active: failed") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

type LinuxProcessMetrics struct {
	Memory int64
	CPU    float64
}

func (l *LinuxServiceManager) getProcessMetrics(pid int) (*LinuxProcessMetrics, error) {
	// Read process status from /proc/PID/status
	statusFile := filepath.Join("/proc", strconv.Itoa(pid), "status")
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read process status: %w", err)
	}
	
	metrics := &LinuxProcessMetrics{}
	
	// Parse memory usage
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if kb, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					metrics.Memory = kb * 1024 // Convert KB to bytes
				}
			}
		}
	}
	
	// CPU usage would require reading /proc/PID/stat and calculating over time
	// For simplicity, we'll skip it here
	
	return metrics, nil
}

func (l *LinuxServiceManager) parseJournalOutput(output string) ([]LogEntry, error) {
	var logs []LogEntry
	
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		// Simple journal parsing - real implementation would parse JSON
		// For now, create basic log entries
		entry := LogEntry{
			Timestamp: time.Now(), // Would parse from JSON
			Level:     "info",     // Would parse from JSON
			Message:   line,
			Source:    l.serviceName,
		}
		
		logs = append(logs, entry)
	}
	
	return logs, scanner.Err()
}

func (l *LinuxServiceManager) setDirectoryOwnership(path, username, groupname string) error {
	// Get user ID
	var uid int = -1
	if username != "" {
		u, err := user.Lookup(username)
		if err != nil {
			return fmt.Errorf("failed to lookup user %s: %w", username, err)
		}
		uid, err = strconv.Atoi(u.Uid)
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}
	}
	
	// Get group ID
	var gid int = -1
	if groupname != "" {
		g, err := user.LookupGroup(groupname)
		if err != nil {
			return fmt.Errorf("failed to lookup group %s: %w", groupname, err)
		}
		gid, err = strconv.Atoi(g.Gid)
		if err != nil {
			return fmt.Errorf("invalid group ID: %w", err)
		}
	}
	
	// Change ownership
	return filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return syscall.Chown(filePath, uid, gid)
	})
}