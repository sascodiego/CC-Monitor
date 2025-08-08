/**
 * CONTEXT:   Service configuration types, constants, and command initialization
 * INPUT:     Service configuration structures, enums, and CLI command definitions
 * OUTPUT:    Type definitions and command initialization for service management
 * BUSINESS:  Service types provide structured configuration and CLI interface
 * CHANGE:    Extracted from service.go - focused type definitions and command setup
 * RISK:      Low - Type definitions and command structure with no side effects
 */

package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/claude-monitor/system/internal/service"
	"github.com/spf13/cobra"
)

// =============================================================================
// Service Configuration Types
// =============================================================================

/**
 * CONTEXT:   Legacy ServiceManager interface - DEPRECATED in favor of segregated interfaces
 * INPUT:     Service configuration and platform-specific parameters
 * OUTPUT:    Service management operations (install, start, stop, status)
 * BUSINESS:  Fat interface violating Interface Segregation Principle - being replaced
 * CHANGE:    DEPRECATED - Use interfaces.ServiceInstaller, ServiceController, ServiceMonitor instead
 * RISK:      Medium - Fat interface coupling - will be removed after migration
 */
// DEPRECATED: Use segregated interfaces instead
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

// =============================================================================
// CLI Command Variables
// =============================================================================

var (
	serviceAutoStart  bool
	serviceUser       string
	serviceUserLevel  bool
	serviceSystemMode bool
	serviceLogLines   int
	serviceFollow     bool
)

//go:embed service_templates/*
var serviceTemplates embed.FS

// =============================================================================
// Command Definition and Initialization
// =============================================================================

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

/**
 * CONTEXT:   Service command initialization with subcommands and flags
 * INPUT:     Cobra command definitions and flag configurations
 * OUTPUT:    Complete service command structure with all subcommands
 * BUSINESS:  Command initialization enables user-friendly service management
 * CHANGE:    Initial command setup with install/start/stop/restart/status/logs/uninstall
 * RISK:      Low - Command structure initialization with no side effects
 */
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

// =============================================================================
// Service Factory and Configuration Functions
// =============================================================================

/**
 * CONTEXT:   Create composite service manager implementing all segregated interfaces
 * INPUT:     Platform detection and service configuration requirements
 * OUTPUT:    Composite service manager with all interface implementations
 * BUSINESS:  Factory provides access to segregated interfaces while maintaining compatibility
 * CHANGE:    Updated to use Interface Segregation Principle with composite pattern
 * RISK:      Medium - Service manager creation affecting all service functionality
 */
func NewCompositeServiceManager() (*service.CompositeServiceManager, error) {
	return service.NewCompositeServiceManager()
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