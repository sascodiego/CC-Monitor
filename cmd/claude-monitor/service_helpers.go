/**
 * CONTEXT:   Service helper functions for platform detection and integration
 * INPUT:     Platform detection, service state checking, utility functions
 * OUTPUT:    Service management utilities and cross-platform compatibility
 * BUSINESS:  Service helpers enable seamless service integration across platforms
 * CHANGE:    Initial service helper functions with platform detection
 * RISK:      Low - Utility functions with proper error handling
 */

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

/**
 * CONTEXT:   Windows service detection helper
 * INPUT:     Runtime environment and Windows service API
 * OUTPUT:    Boolean indicating if running as Windows service
 * BUSINESS:  Service detection enables automatic service mode activation
 * CHANGE:    Initial Windows service detection implementation
 * RISK:      Low - Platform detection with safe fallback
 */
func isRunningAsWindowsService() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, nil
	}
	
	// Check if we're running in a service context
	// This is a simplified check - in production, you'd use golang.org/x/sys/windows/svc
	// For now, check if no console is attached (common for services)
	if os.Getenv("CLAUDE_MONITOR_MODE") == "service" {
		return true, nil
	}
	
	// Additional checks could be added here
	// - Check if running under service control manager
	// - Check for service-specific environment variables
	// - Check process parent information
	
	return false, nil
}

/**
 * CONTEXT:   Service configuration validation
 * INPUT:     Service configuration parameters and validation rules
 * OUTPUT:    Validated configuration or error details
 * BUSINESS:  Configuration validation ensures reliable service operation
 * CHANGE:    Initial configuration validation with comprehensive checks
 * RISK:      Medium - Configuration validation affects service reliability
 */
func validateServiceConfiguration(config ServiceConfig) error {
	if config.Name == "" {
		return fmt.Errorf("service name is required")
	}
	
	if config.DisplayName == "" {
		return fmt.Errorf("service display name is required")
	}
	
	if config.ExecutablePath == "" {
		return fmt.Errorf("executable path is required")
	}
	
	// Check if executable exists
	if _, err := os.Stat(config.ExecutablePath); err != nil {
		return fmt.Errorf("executable not found: %s", config.ExecutablePath)
	}
	
	// Validate working directory
	if config.WorkingDir != "" {
		if info, err := os.Stat(config.WorkingDir); err != nil {
			return fmt.Errorf("working directory not found: %s", config.WorkingDir)
		} else if !info.IsDir() {
			return fmt.Errorf("working directory is not a directory: %s", config.WorkingDir)
		}
	}
	
	// Validate start mode
	switch config.StartMode {
	case StartModeAuto, StartModeManual, StartModeDisabled:
		// Valid start modes
	default:
		return fmt.Errorf("invalid start mode: %s", config.StartMode)
	}
	
	return nil
}

/**
 * CONTEXT:   Cross-platform service capability detection
 * INPUT:     Platform information and service system availability
 * OUTPUT:    Service capabilities and limitations for current platform
 * BUSINESS:  Capability detection enables appropriate service feature availability
 * CHANGE:    Initial capability detection with Windows and Linux support
 * RISK:      Low - Feature detection with safe defaults
 */
type ServiceCapabilities struct {
	CanInstall      bool   `json:"can_install"`
	CanStart        bool   `json:"can_start"`
	CanStop         bool   `json:"can_stop"`
	CanRestart      bool   `json:"can_restart"`
	CanGetStatus    bool   `json:"can_get_status"`
	CanGetLogs      bool   `json:"can_get_logs"`
	RequiresAdmin   bool   `json:"requires_admin"`
	ServiceType     string `json:"service_type"`
	ConfigLocation  string `json:"config_location"`
	LogLocation     string `json:"log_location"`
}

func GetServiceCapabilities() ServiceCapabilities {
	switch runtime.GOOS {
	case "windows":
		return ServiceCapabilities{
			CanInstall:     true,
			CanStart:       true,
			CanStop:        true,
			CanRestart:     true,
			CanGetStatus:   true,
			CanGetLogs:     true,
			RequiresAdmin:  true,
			ServiceType:    "Windows Service (SCM)",
			ConfigLocation: "Registry",
			LogLocation:    "Windows Event Log",
		}
		
	case "linux":
		return ServiceCapabilities{
			CanInstall:     true,
			CanStart:       true,
			CanStop:        true,
			CanRestart:     true,
			CanGetStatus:   true,
			CanGetLogs:     true,
			RequiresAdmin:  !serviceUserLevel,
			ServiceType:    "systemd",
			ConfigLocation: getLinuxConfigLocation(),
			LogLocation:    "systemd journal",
		}
		
	default:
		return ServiceCapabilities{
			CanInstall:     false,
			CanStart:       false,
			CanStop:        false,
			CanRestart:     false,
			CanGetStatus:   false,
			CanGetLogs:     false,
			RequiresAdmin:  false,
			ServiceType:    "Unsupported",
			ConfigLocation: "N/A",
			LogLocation:    "N/A",
		}
	}
}

func getLinuxConfigLocation() string {
	if serviceUserLevel {
		homeDir, _ := os.UserHomeDir()
		return homeDir + "/.config/systemd/user/"
	}
	return "/etc/systemd/system/"
}


/**
 * CONTEXT:   Service status display helpers
 * INPUT:     Service status information and formatting preferences
 * OUTPUT:    Formatted service status for CLI display
 * BUSINESS:  Status helpers provide consistent service information presentation
 * CHANGE:    Initial status helpers with color coding and formatting
 * RISK:      Low - Display formatting with no system impact
 */
func formatServiceStatusSummary(status ServiceStatus) string {
	var summary strings.Builder
	
	// Service state
	switch status.State {
	case ServiceStateRunning:
		summary.WriteString("✅ Running")
	case ServiceStateStopped:
		summary.WriteString("⏹️ Stopped")
	case ServiceStateStarting:
		summary.WriteString("⏳ Starting")
	case ServiceStateStopping:
		summary.WriteString("⏹️ Stopping")
	case ServiceStateFailed:
		summary.WriteString("❌ Failed")
	default:
		summary.WriteString("❓ Unknown")
	}
	
	// Uptime if available
	if status.Uptime > 0 {
		summary.WriteString(fmt.Sprintf(" (uptime: %s)", formatDuration(status.Uptime)))
	}
	
	// PID if available
	if status.PID > 0 {
		summary.WriteString(fmt.Sprintf(" [PID: %d]", status.PID))
	}
	
	return summary.String()
}

func formatServiceInfo(status ServiceStatus) [][]string {
	var info [][]string
	
	info = append(info, []string{"Service Name:", status.Name})
	info = append(info, []string{"Display Name:", status.DisplayName})
	info = append(info, []string{"State:", string(status.State)})
	
	if status.PID > 0 {
		info = append(info, []string{"Process ID:", fmt.Sprintf("%d", status.PID)})
	}
	
	if !status.StartTime.IsZero() {
		info = append(info, []string{"Started:", status.StartTime.Format("2006-01-02 15:04:05")})
		info = append(info, []string{"Uptime:", formatDuration(status.Uptime)})
	}
	
	if status.Memory > 0 {
		info = append(info, []string{"Memory:", formatBytes(status.Memory)})
	}
	
	if status.CPU > 0 {
		info = append(info, []string{"CPU:", fmt.Sprintf("%.1f%%", status.CPU)})
	}
	
	if status.LastError != "" {
		info = append(info, []string{"Last Error:", status.LastError})
	}
	
	return info
}