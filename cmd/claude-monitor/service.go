/**
 * CONTEXT:   Cross-platform service management for Claude Monitor system service installation
 * INPUT:     Service configuration, platform detection, system service commands
 * OUTPUT:    System service integration with Windows SCM and Linux systemd
 * BUSINESS:  Professional daemon installation with system service management
 * CHANGE:    Initial service management implementation with cross-platform support
 * RISK:      High - System-level service installation requiring admin privileges
 */

package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

/**
 * CONTEXT:   Cross-platform service manager interface
 * INPUT:     Service configuration and platform-specific parameters
 * OUTPUT:    Service management operations (install, start, stop, status)
 * BUSINESS:  Abstract interface enabling Windows and Linux service support
 * CHANGE:    Initial interface design with comprehensive service operations
 * RISK:      Medium - Interface design affects all service implementations
 */
type ServiceManager interface {
	Install(config ServiceConfig) error
	Uninstall() error
	Start() error
	Stop() error
	Restart() error
	Status() (ServiceStatus, error)
	IsInstalled() bool
	IsRunning() bool
	GetLogs(lines int) ([]LogEntry, error)
}

/**
 * CONTEXT:   Service configuration structure for cross-platform deployment
 * INPUT:     User preferences, system requirements, security settings
 * OUTPUT:    Complete service configuration with platform adaptations
 * BUSINESS:  Standardized configuration enables consistent service behavior
 * CHANGE:    Initial configuration structure with security and reliability options
 * RISK:      Medium - Configuration affects service security and reliability
 */
type ServiceConfig struct {
	Name            string            `json:"name"`
	DisplayName     string            `json:"display_name"`
	Description     string            `json:"description"`
	ExecutablePath  string            `json:"executable_path"`
	Arguments       []string          `json:"arguments"`
	WorkingDir      string            `json:"working_dir"`
	User            string            `json:"user,omitempty"`
	Group           string            `json:"group,omitempty"`
	StartMode       ServiceStartMode  `json:"start_mode"`
	RestartOnFailure bool             `json:"restart_on_failure"`
	RestartDelay    time.Duration     `json:"restart_delay"`
	LogLevel        string            `json:"log_level"`
	Environment     map[string]string `json:"environment"`
	
	// Platform-specific settings
	WindowsService WindowsServiceConfig `json:"windows,omitempty"`
	LinuxService   LinuxServiceConfig   `json:"linux,omitempty"`
}

type ServiceStartMode string

const (
	StartModeAuto     ServiceStartMode = "auto"
	StartModeManual   ServiceStartMode = "manual" 
	StartModeDisabled ServiceStartMode = "disabled"
)

type WindowsServiceConfig struct {
	ServiceAccount string `json:"service_account,omitempty"`
	Password       string `json:"password,omitempty"`
	Dependencies   []string `json:"dependencies,omitempty"`
}

type LinuxServiceConfig struct {
	SystemdUnit     bool     `json:"systemd_unit"`
	UserService     bool     `json:"user_service"`
	SystemService   bool     `json:"system_service"`
	WantedBy        []string `json:"wanted_by,omitempty"`
	RequiredBy      []string `json:"required_by,omitempty"`
	After           []string `json:"after,omitempty"`
	Before          []string `json:"before,omitempty"`
	Capabilities    []string `json:"capabilities,omitempty"`
}

/**
 * CONTEXT:   Service status information for monitoring and troubleshooting
 * INPUT:     Platform service manager status queries
 * OUTPUT:    Unified service status across Windows and Linux platforms
 * BUSINESS:  Service status enables monitoring and automated management
 * CHANGE:    Initial status structure with comprehensive state information
 * RISK:      Low - Status structure with platform-specific adaptations
 */
type ServiceStatus struct {
	Name        string        `json:"name"`
	DisplayName string        `json:"display_name"`
	State       ServiceState  `json:"state"`
	PID         int           `json:"pid,omitempty"`
	Uptime      time.Duration `json:"uptime,omitempty"`
	StartTime   time.Time     `json:"start_time,omitempty"`
	Memory      int64         `json:"memory_usage,omitempty"`
	CPU         float64       `json:"cpu_usage,omitempty"`
	LastError   string        `json:"last_error,omitempty"`
}

type ServiceState string

const (
	ServiceStateUnknown  ServiceState = "unknown"
	ServiceStateRunning  ServiceState = "running"
	ServiceStateStopped  ServiceState = "stopped"
	ServiceStateStarting ServiceState = "starting"
	ServiceStateStopping ServiceState = "stopping"
	ServiceStateFailed   ServiceState = "failed"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}

/**
 * CONTEXT:   Service command group for comprehensive service management
 * INPUT:     User service management requests and system administration
 * OUTPUT:    Professional service installation and management interface
 * BUSINESS:  Service commands enable enterprise-grade daemon deployment
 * CHANGE:    Initial service command group with install/start/stop/status operations
 * RISK:      High - Service commands modify system configuration
 */
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage Claude Monitor system service",
	Long: `Professional system service management for Claude Monitor.

Provides enterprise-grade daemon installation with:
- Automatic system service registration
- Cross-platform support (Windows/Linux)
- Service lifecycle management
- Health monitoring and logging
- Security best practices

Supports both Windows Service Control Manager and Linux systemd.`,
	
	Example: `  # Install and start system service
  claude-monitor service install --auto-start
  claude-monitor service status
  
  # Manual service management
  claude-monitor service stop
  claude-monitor service start
  claude-monitor service restart
  
  # View service logs
  claude-monitor service logs --lines=50
  
  # Remove service
  claude-monitor service uninstall`,
}

var (
	serviceAutoStart  bool
	serviceUser       string
	serviceUserLevel  bool
	serviceSystemMode bool
	serviceLogLines   int
	serviceFollow     bool
)

func init() {
	// Service install subcommand
	serviceInstallCmd := &cobra.Command{
		Use:   "install",
		Short: "Install Claude Monitor as system service",
		Long: `Install Claude Monitor as a system service with automatic startup.

This will:
- Register service with system service manager
- Configure automatic startup on boot
- Set appropriate security permissions
- Create service logs directory
- Verify service installation`,
		RunE: runServiceInstall,
	}
	
	serviceInstallCmd.Flags().BoolVar(&serviceAutoStart, "auto-start", true, "start service automatically after install")
	serviceInstallCmd.Flags().StringVar(&serviceUser, "user", "", "run service as specific user (Linux only)")
	serviceInstallCmd.Flags().BoolVar(&serviceUserLevel, "user-service", false, "install as user-level service")
	serviceInstallCmd.Flags().BoolVar(&serviceSystemMode, "system", true, "install as system-level service")
	
	// Service management subcommands
	serviceStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start Claude Monitor service", 
		RunE:  runServiceStart,
	}
	
	serviceStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop Claude Monitor service",
		RunE:  runServiceStop,
	}
	
	serviceRestartCmd := &cobra.Command{
		Use:   "restart", 
		Short: "Restart Claude Monitor service",
		RunE:  runServiceRestart,
	}
	
	serviceStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show service status and health",
		RunE:  runServiceStatus,
	}
	
	serviceLogsCmd := &cobra.Command{
		Use:   "logs",
		Short: "View service logs",
		RunE:  runServiceLogs,
	}
	
	serviceLogsCmd.Flags().IntVar(&serviceLogLines, "lines", 20, "number of log lines to show")
	serviceLogsCmd.Flags().BoolVarP(&serviceFollow, "follow", "f", false, "follow log output")
	
	serviceUninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Claude Monitor service",
		RunE:  runServiceUninstall,
	}
	
	// Add all subcommands
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	serviceCmd.AddCommand(serviceLogsCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
}

/**
 * CONTEXT:   Service manager factory for platform-specific implementations
 * INPUT:     Runtime platform detection and service configuration
 * OUTPUT:    Platform-appropriate service manager implementation
 * BUSINESS:  Factory pattern enables clean cross-platform service support
 * CHANGE:    Initial factory implementation with Windows and Linux support
 * RISK:      Medium - Platform detection affects service functionality
 */
func NewServiceManager() (ServiceManager, error) {
	switch runtime.GOOS {
	case "windows":
		// WindowsServiceManager is only available on Windows builds
		// This will be handled by build tags
		return nil, fmt.Errorf("Windows service manager not available in this build")
	case "linux":
		return NewLinuxServiceManager()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

/**
 * CONTEXT:   Default service configuration generation
 * INPUT:     Binary path, system information, user preferences
 * OUTPUT:    Complete service configuration with platform optimizations
 * BUSINESS:  Default configuration ensures reliable service operation
 * CHANGE:    Initial configuration with security and performance defaults
 * RISK:      Medium - Default configuration affects service security and reliability
 */
func getDefaultServiceConfig() (ServiceConfig, error) {
	executable, err := os.Executable()
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("cannot determine executable path: %w", err)
	}
	
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".claude-monitor")
	
	config := ServiceConfig{
		Name:             "claude-monitor",
		DisplayName:      "Claude Monitor Work Tracking Service",
		Description:      "Work hour tracking daemon for Claude Code users with project analytics and session management",
		ExecutablePath:   executable,
		Arguments:        []string{"daemon", "--log-level=info"},
		WorkingDir:       configDir,
		StartMode:        StartModeAuto,
		RestartOnFailure: true,
		RestartDelay:     5 * time.Second,
		LogLevel:         "info",
		Environment: map[string]string{
			"CLAUDE_MONITOR_CONFIG": filepath.Join(configDir, "config.json"),
			"CLAUDE_MONITOR_MODE":   "service",
		},
	}
	
	// Platform-specific configuration
	switch runtime.GOOS {
	case "windows":
		config.WindowsService = WindowsServiceConfig{
			ServiceAccount: "LocalSystem",
			Dependencies:   []string{"Tcpip"},
		}
		
	case "linux":
		config.LinuxService = LinuxServiceConfig{
			SystemdUnit:   true,
			UserService:   serviceUserLevel,
			SystemService: serviceSystemMode,
			WantedBy:      []string{"multi-user.target"},
			After:         []string{"network.target"},
		}
		
		if serviceUser != "" {
			config.User = serviceUser
		} else if serviceUserLevel {
			config.User = os.Getenv("USER")
		}
	}
	
	return config, nil
}

/**
 * CONTEXT:   Service installation command implementation
 * INPUT:     Installation flags, system permissions, target configuration
 * OUTPUT:    Installed and configured system service ready for operation
 * BUSINESS:  Service installation enables professional daemon deployment
 * CHANGE:    Initial installation with comprehensive system integration
 * RISK:      High - System modification requiring careful validation and rollback
 */
func runServiceInstall(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	
	headerColor.Println("🚀 Claude Monitor Service Installation")
	fmt.Println(strings.Repeat("═", 60))
	
	// Check for required permissions
	if err := checkServicePermissions(); err != nil {
		return fmt.Errorf("insufficient permissions: %w", err)
	}
	
	// Create service manager
	manager, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Check if already installed
	if manager.IsInstalled() {
		warningColor.Println("⚠️  Service is already installed")
		
		// Check if running
		if manager.IsRunning() {
			infoColor.Println("✅ Service is currently running")
		} else {
			infoColor.Println("🔄 Service is installed but not running")
			
			if serviceAutoStart {
				infoColor.Println("⏳ Starting existing service...")
				if err := manager.Start(); err != nil {
					return fmt.Errorf("failed to start existing service: %w", err)
				}
				successColor.Println("✅ Service started successfully")
			}
		}
		return nil
	}
	
	// Generate service configuration
	config, err := getDefaultServiceConfig()
	if err != nil {
		return fmt.Errorf("failed to generate service configuration: %w", err)
	}
	
	infoColor.Printf("📍 Service Name: %s\n", config.Name)
	infoColor.Printf("📂 Executable: %s\n", config.ExecutablePath)
	infoColor.Printf("🏠 Working Dir: %s\n", config.WorkingDir)
	infoColor.Printf("👤 User: %s\n", getUserDisplayName(config.User))
	
	// Create working directory if needed
	if err := os.MkdirAll(config.WorkingDir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}
	
	// Create logs directory
	logsDir := filepath.Join(config.WorkingDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	
	// Install the service
	infoColor.Println("⏳ Installing service...")
	if err := manager.Install(config); err != nil {
		return fmt.Errorf("service installation failed: %w", err)
	}
	successColor.Println("✅ Service installed successfully")
	
	// Start service if requested
	if serviceAutoStart {
		infoColor.Println("⏳ Starting service...")
		if err := manager.Start(); err != nil {
			warningColor.Printf("⚠️  Service installed but failed to start: %v\n", err)
			fmt.Println()
			infoColor.Println("💡 You can start the service manually with:")
			infoColor.Printf("   claude-monitor service start\n")
		} else {
			successColor.Println("✅ Service started successfully")
			
			// Wait a moment and verify
			time.Sleep(2 * time.Second)
			if manager.IsRunning() {
				successColor.Println("✅ Service is running and healthy")
			}
		}
	}
	
	duration := time.Since(startTime)
	fmt.Println()
	successColor.Printf("🎉 Service installation completed in %v\n", duration.Round(time.Millisecond))
	
	// Display next steps
	fmt.Println()
	headerColor.Println("📋 Service Management Commands:")
	fmt.Printf("  Status:     claude-monitor service status\n")
	fmt.Printf("  Start:      claude-monitor service start\n")
	fmt.Printf("  Stop:       claude-monitor service stop\n")
	fmt.Printf("  Restart:    claude-monitor service restart\n")
	fmt.Printf("  Logs:       claude-monitor service logs\n")
	fmt.Printf("  Uninstall:  claude-monitor service uninstall\n")
	
	return nil
}

func runServiceStart(cmd *cobra.Command, args []string) error {
	return executeServiceCommand("start", func(manager ServiceManager) error {
		return manager.Start()
	})
}

func runServiceStop(cmd *cobra.Command, args []string) error {
	return executeServiceCommand("stop", func(manager ServiceManager) error {
		return manager.Stop()
	})
}

func runServiceRestart(cmd *cobra.Command, args []string) error {
	return executeServiceCommand("restart", func(manager ServiceManager) error {
		return manager.Restart()
	})
}

/**
 * CONTEXT:   Service status display with comprehensive health information
 * INPUT:     Service manager status queries and system metrics
 * OUTPUT:    Detailed service status with troubleshooting information
 * BUSINESS:  Service status enables monitoring and problem diagnosis
 * CHANGE:    Initial status display with health metrics and guidance
 * RISK:      Low - Read-only status information with helpful guidance
 */
func runServiceStatus(cmd *cobra.Command, args []string) error {
	headerColor.Println("🔍 Claude Monitor Service Status")
	fmt.Println(strings.Repeat("═", 50))
	
	manager, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Check installation status
	if !manager.IsInstalled() {
		warningColor.Println("❌ Service is not installed")
		fmt.Println()
		infoColor.Println("💡 Install the service with:")
		infoColor.Println("   claude-monitor service install")
		return nil
	}
	
	// Get detailed status
	status, err := manager.Status()
	if err != nil {
		errorColor.Printf("❌ Failed to get service status: %v\n", err)
		return nil
	}
	
	// Display status
	fmt.Println()
	displayServiceStatus(status)
	
	// Check daemon connectivity
	fmt.Println()
	infoColor.Println("🏥 Daemon Health Check")
	fmt.Println(strings.Repeat("─", 30))
	
	config, err := loadConfiguration()
	if err == nil {
		client := NewHTTPClient(2 * time.Second)
		daemonURL := fmt.Sprintf("http://%s", config.Daemon.ListenAddr)
		
		if health, err := client.GetHealthStatus(daemonURL); err != nil {
			warningColor.Printf("⚠️  Daemon API: Unreachable (%v)\n", err)
		} else {
			successColor.Printf("✅ Daemon API: Healthy (uptime: %s)\n", health.Uptime)
			infoColor.Printf("📊 Active Sessions: %d\n", health.ActiveSessions)
			infoColor.Printf("🔄 Work Blocks: %d\n", health.TotalWorkBlocks)
		}
	}
	
	return nil
}

func runServiceLogs(cmd *cobra.Command, args []string) error {
	manager, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	if !manager.IsInstalled() {
		return fmt.Errorf("service is not installed")
	}
	
	logs, err := manager.GetLogs(serviceLogLines)
	if err != nil {
		return fmt.Errorf("failed to retrieve logs: %w", err)
	}
	
	headerColor.Printf("📋 Service Logs (last %d entries)\n", len(logs))
	fmt.Println(strings.Repeat("═", 50))
	
	for _, entry := range logs {
		var logColor *color.Color
		switch strings.ToLower(entry.Level) {
		case "error":
			logColor = errorColor
		case "warn", "warning":
			logColor = warningColor
		case "info":
			logColor = infoColor
		default:
			logColor = dimColor
		}
		
		logColor.Printf("%s [%s] %s: %s\n",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Level,
			entry.Source,
			entry.Message)
	}
	
	return nil
}

func runServiceUninstall(cmd *cobra.Command, args []string) error {
	headerColor.Println("🗑️  Claude Monitor Service Uninstallation")
	fmt.Println(strings.Repeat("═", 60))
	
	manager, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	if !manager.IsInstalled() {
		warningColor.Println("⚠️  Service is not installed")
		return nil
	}
	
	// Stop service if running
	if manager.IsRunning() {
		infoColor.Println("⏳ Stopping service...")
		if err := manager.Stop(); err != nil {
			warningColor.Printf("⚠️  Failed to stop service: %v\n", err)
		} else {
			successColor.Println("✅ Service stopped")
		}
	}
	
	// Uninstall service
	infoColor.Println("⏳ Uninstalling service...")
	if err := manager.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}
	
	successColor.Println("✅ Service uninstalled successfully")
	infoColor.Println("ℹ️  Configuration and data files were preserved")
	
	return nil
}

/**
 * CONTEXT:   Helper functions for service management operations
 * INPUT:     Service manager operations and system utilities
 * OUTPUT:    Common service management functionality and error handling
 * BUSINESS:  Helper functions reduce code duplication and improve reliability
 * CHANGE:    Initial helper functions with consistent error handling
 * RISK:      Low - Utility functions with proper error propagation
 */
func executeServiceCommand(operation string, action func(ServiceManager) error) error {
	manager, err := NewServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	if !manager.IsInstalled() {
		return fmt.Errorf("service is not installed - run 'claude-monitor service install' first")
	}
	
	infoColor.Printf("⏳ %sing service...\n", strings.Title(operation))
	
	if err := action(manager); err != nil {
		return fmt.Errorf("failed to %s service: %w", operation, err)
	}
	
	successColor.Printf("✅ Service %sed successfully\n", operation)
	
	// Wait a moment and show status for start/restart
	if operation == "start" || operation == "restart" {
		time.Sleep(time.Second)
		if manager.IsRunning() {
			successColor.Println("✅ Service is running and healthy")
		} else {
			warningColor.Println("⚠️  Service may still be starting...")
		}
	}
	
	return nil
}

func displayServiceStatus(status ServiceStatus) {
	// Service state with color coding
	var stateColor *color.Color
	var stateIcon string
	
	switch status.State {
	case ServiceStateRunning:
		stateColor = successColor
		stateIcon = "✅"
	case ServiceStateStopped:
		stateColor = warningColor
		stateIcon = "⏹️"
	case ServiceStateStarting:
		stateColor = infoColor
		stateIcon = "⏳"
	case ServiceStateStopping:
		stateColor = warningColor
		stateIcon = "⏹️"
	case ServiceStateFailed:
		stateColor = errorColor
		stateIcon = "❌"
	default:
		stateColor = dimColor
		stateIcon = "❓"
	}
	
	fmt.Printf("Service: %s\n", status.DisplayName)
	stateColor.Printf("%s State: %s\n", stateIcon, string(status.State))
	
	if status.PID > 0 {
		infoColor.Printf("🆔 PID: %d\n", status.PID)
	}
	
	if !status.StartTime.IsZero() {
		infoColor.Printf("⏰ Started: %s\n", status.StartTime.Format("2006-01-02 15:04:05"))
		infoColor.Printf("⏱️  Uptime: %s\n", status.Uptime.Round(time.Second))
	}
	
	if status.Memory > 0 {
		infoColor.Printf("💾 Memory: %s\n", formatBytes(status.Memory))
	}
	
	if status.CPU > 0 {
		infoColor.Printf("🔄 CPU: %.1f%%\n", status.CPU)
	}
	
	if status.LastError != "" {
		errorColor.Printf("⚠️  Last Error: %s\n", status.LastError)
	}
}

func checkServicePermissions() error {
	switch runtime.GOOS {
	case "windows":
		// Check if running as administrator
		return checkWindowsAdminRights()
	case "linux":
		// Check if can write to systemd directory or running as root
		return checkLinuxServicePermissions()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func checkWindowsAdminRights() error {
	// Try to open a handle to the SCM with full access
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("administrator privileges required - please run as administrator")
	}
	return nil
}

func checkLinuxServicePermissions() error {
	if serviceUserLevel {
		// User service - check if can write to user systemd directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		
		userSystemdDir := filepath.Join(homeDir, ".config", "systemd", "user")
		if err := os.MkdirAll(userSystemdDir, 0755); err != nil {
			return fmt.Errorf("cannot create user systemd directory: %w", err)
		}
		return nil
	}
	
	// System service - check if can write to system systemd directory
	systemdDir := "/etc/systemd/system"
	testFile := filepath.Join(systemdDir, ".claude-monitor-test")
	
	file, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("root privileges required for system service installation")
	}
	file.Close()
	os.Remove(testFile)
	
	return nil
}

func getUserDisplayName(user string) string {
	if user == "" {
		switch runtime.GOOS {
		case "windows":
			return "LocalSystem"
		default:
			return "root"
		}
	}
	return user
}

func formatBytes(bytes int64) string {
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

//go:embed service_templates/*
var serviceTemplates embed.FS