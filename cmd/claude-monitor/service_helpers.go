/**
 * CONTEXT:   Service utility functions for display formatting and configuration conversion
 * INPUT:     Service status data, configuration objects, and system information
 * OUTPUT:    Formatted display output, configuration conversions, and utility operations
 * BUSINESS:  Helper functions provide consistent formatting and reduce code duplication
 * CHANGE:    Extracted from service.go - focused utility functions with single responsibility
 * RISK:      Low - Utility functions with no side effects, focused on display and conversion
 */

package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/claude-monitor/system/internal/service/interfaces"
	"github.com/fatih/color"
)

/**
 * CONTEXT:   Service controller command execution using ServiceController interface
 * INPUT:     Operation name and controller action function
 * OUTPUT:    Service operation result using focused interface
 * BUSINESS:  Controller commands use only controller interface (Interface Segregation Principle)
 * CHANGE:    Interface segregation - controller operations only
 * RISK:      Medium - Service operations affecting system behavior
 */
func executeServiceControllerCommand(operation string, action func(interfaces.ServiceController) error) error {
	// Create composite service manager
	compositeManager, err := NewCompositeServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Use ServiceInstaller interface for installation check
	var installer interfaces.ServiceInstaller = compositeManager
	// Use ServiceController interface for runtime operations
	var controller interfaces.ServiceController = compositeManager
	
	if !installer.IsInstalled() {
		return fmt.Errorf("service is not installed - run 'claude-monitor service install' first")
	}
	
	infoColor.Printf("⏳ %sing service...\n", strings.Title(operation))
	
	if err := action(controller); err != nil {
		return fmt.Errorf("failed to %s service: %w", operation, err)
	}
	
	successColor.Printf("✅ Service %sed successfully\n", operation)
	
	// Wait a moment and show status for start/restart
	if operation == "start" || operation == "restart" {
		time.Sleep(time.Second)
		if controller.IsRunning() {
			successColor.Println("✅ Service is running and healthy")
		} else {
			warningColor.Println("⚠️  Service may still be starting...")
		}
	}
	
	return nil
}

/**
 * CONTEXT:   Service status display using interfaces.ServiceStatus structure
 * INPUT:     interfaces.ServiceStatus with comprehensive status information
 * OUTPUT:    Formatted status display with color coding and metrics
 * BUSINESS:  Status display enables monitoring using segregated interface data
 * CHANGE:    Updated to use interfaces.ServiceStatus instead of legacy ServiceStatus
 * RISK:      Low - Display function using interface-segregated status information
 */
func displayInterfacesServiceStatus(status interfaces.ServiceStatus) {
	// Service state with color coding
	var stateColor *color.Color
	var stateIcon string
	
	switch status.State {
	case interfaces.ServiceStateRunning:
		stateColor = successColor
		stateIcon = "✅"
	case interfaces.ServiceStateStopped:
		stateColor = warningColor
		stateIcon = "⏹️"
	case interfaces.ServiceStateStarting:
		stateColor = infoColor
		stateIcon = "⏳"
	case interfaces.ServiceStateStopping:
		stateColor = warningColor
		stateIcon = "⏹️"
	case interfaces.ServiceStateFailed:
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

/**
 * CONTEXT:   Legacy service status display function (DEPRECATED)
 * INPUT:     Legacy ServiceStatus structure
 * OUTPUT:    Formatted status display with color coding and metrics
 * BUSINESS:  DEPRECATED status display for backwards compatibility
 * CHANGE:    DEPRECATED - Use displayInterfacesServiceStatus instead
 * RISK:      Low - Legacy display function, will be removed after migration
 */
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

/**
 * CONTEXT:   User display name formatting for service configuration
 * INPUT:     User string (may be empty)
 * OUTPUT:    Platform-appropriate user display name
 * BUSINESS:  Consistent user display across platforms for service configuration
 * CHANGE:    Extracted utility function with platform-specific defaults
 * RISK:      Low - Simple string formatting utility with platform detection
 */
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

/**
 * CONTEXT:   Byte size formatting utility for memory display
 * INPUT:     Byte count as int64
 * OUTPUT:    Human-readable byte size string (B, KB, MB, GB, etc.)
 * BUSINESS:  Consistent memory usage display in service status
 * CHANGE:    Extracted utility function for byte formatting
 * RISK:      Low - Pure formatting function with no side effects
 */
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

/**
 * CONTEXT:   Configuration conversion from legacy ServiceConfig to interfaces.ServiceConfig
 * INPUT:     Legacy ServiceConfig structure with platform-specific settings
 * OUTPUT:    interfaces.ServiceConfig structure for segregated interface use
 * BUSINESS:  Configuration conversion enables Interface Segregation Principle adoption
 * CHANGE:    Initial configuration conversion supporting interface migration
 * RISK:      Low - Configuration mapping preserving all settings
 */
func convertToInterfacesConfig(config ServiceConfig) interfaces.ServiceConfig {
	return interfaces.ServiceConfig{
		Name:             config.Name,
		DisplayName:      config.DisplayName,
		Description:      config.Description,
		ExecutablePath:   config.ExecutablePath,
		Arguments:        config.Arguments,
		WorkingDir:       config.WorkingDir,
		User:             config.User,
		Group:            config.Group,
		StartMode:        interfaces.ServiceStartMode(config.StartMode),
		RestartOnFailure: config.RestartOnFailure,
		LogLevel:         config.LogLevel,
		Environment:      config.Environment,
	}
}