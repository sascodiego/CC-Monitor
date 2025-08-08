/**
 * CONTEXT:   Service command handlers implementing service lifecycle operations
 * INPUT:     Cobra commands for install, start, stop, restart, status, logs, uninstall
 * OUTPUT:    Service management operations using segregated interfaces (Interface Segregation Principle)
 * BUSINESS:  Service commands provide professional daemon management with cross-platform support
 * CHANGE:    Extracted from service.go - focused command handlers using segregated interfaces
 * RISK:      Medium - Service commands modify system configuration and affect daemon lifecycle
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/claude-monitor/system/internal/service/interfaces"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

/**
 * CONTEXT:   Service installation command implementation
 * INPUT:     Installation flags, system permissions, target configuration
 * OUTPUT:    Installed and configured system service ready for operation
 * BUSINESS:  Service installation enables professional daemon deployment
 * CHANGE:    Updated to use segregated interfaces instead of fat ServiceManager
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
	
	// Create composite service manager (implementing all segregated interfaces)
	compositeManager, err := NewCompositeServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Use ServiceInstaller interface for installation operations
	var installer interfaces.ServiceInstaller = compositeManager
	
	// Check if already installed
	if installer.IsInstalled() {
		warningColor.Println("⚠️  Service is already installed")
		
		// Check if running (use ServiceController interface for runtime operations)
		var controller interfaces.ServiceController = compositeManager
		if controller.IsRunning() {
			infoColor.Println("✅ Service is currently running")
		} else {
			infoColor.Println("🔄 Service is installed but not running")
			
			if serviceAutoStart {
				infoColor.Println("⏳ Starting existing service...")
				if err := controller.Start(); err != nil {
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
	
	// Convert to interfaces config
	interfacesConfig := convertToInterfacesConfig(config)
	
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
	if err := installer.Install(interfacesConfig); err != nil {
		return fmt.Errorf("service installation failed: %w", err)
	}
	successColor.Println("✅ Service installed successfully")
	
	// Start service if requested
	if serviceAutoStart {
		infoColor.Println("⏳ Starting service...")
		var controller interfaces.ServiceController = compositeManager
		if err := controller.Start(); err != nil {
			warningColor.Printf("⚠️  Service installed but failed to start: %v\n", err)
			fmt.Println()
			infoColor.Println("💡 You can start the service manually with:")
			infoColor.Printf("   claude-monitor service start\n")
		} else {
			successColor.Println("✅ Service started successfully")
			
			// Wait a moment and verify
			time.Sleep(2 * time.Second)
			if controller.IsRunning() {
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

/**
 * CONTEXT:   Service start command using ServiceController interface
 * INPUT:     Start command parameters
 * OUTPUT:    Started service using focused interface
 * BUSINESS:  Start command uses only controller interface (Interface Segregation Principle)
 * CHANGE:    Updated to use ServiceController interface instead of fat ServiceManager
 * RISK:      Medium - Service start operation affecting system behavior
 */
func runServiceStart(cmd *cobra.Command, args []string) error {
	return executeServiceControllerCommand("start", func(controller interfaces.ServiceController) error {
		return controller.Start()
	})
}

/**
 * CONTEXT:   Service stop command using ServiceController interface
 * INPUT:     Stop command parameters
 * OUTPUT:    Stopped service using focused interface
 * BUSINESS:  Stop command uses only controller interface (Interface Segregation Principle)
 * CHANGE:    Updated to use ServiceController interface instead of fat ServiceManager
 * RISK:      Medium - Service stop operation affecting system behavior
 */
func runServiceStop(cmd *cobra.Command, args []string) error {
	return executeServiceControllerCommand("stop", func(controller interfaces.ServiceController) error {
		return controller.Stop()
	})
}

/**
 * CONTEXT:   Service restart command using ServiceController interface
 * INPUT:     Restart command parameters
 * OUTPUT:    Restarted service using focused interface
 * BUSINESS:  Restart command uses only controller interface (Interface Segregation Principle)
 * CHANGE:    Updated to use ServiceController interface instead of fat ServiceManager
 * RISK:      Medium - Service restart operation affecting system behavior
 */
func runServiceRestart(cmd *cobra.Command, args []string) error {
	return executeServiceControllerCommand("restart", func(controller interfaces.ServiceController) error {
		return controller.Restart()
	})
}

/**
 * CONTEXT:   Service status command using ServiceInstaller and ServiceMonitor interfaces
 * INPUT:     Status command parameters
 * OUTPUT:    Service status using focused interfaces
 * BUSINESS:  Status command uses installer and monitor interfaces (Interface Segregation Principle)
 * CHANGE:    Updated to use segregated interfaces instead of fat ServiceManager
 * RISK:      Low - Read-only status operations with improved interface segregation
 */
func runServiceStatus(cmd *cobra.Command, args []string) error {
	headerColor.Println("🔍 Claude Monitor Service Status")
	fmt.Println(strings.Repeat("═", 50))
	
	// Create composite service manager
	compositeManager, err := NewCompositeServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Use ServiceInstaller interface for installation status
	var installer interfaces.ServiceInstaller = compositeManager
	// Use ServiceMonitor interface for status information
	var monitor interfaces.ServiceMonitor = compositeManager
	
	// Check installation status
	if !installer.IsInstalled() {
		warningColor.Println("❌ Service is not installed")
		fmt.Println()
		infoColor.Println("💡 Install the service with:")
		infoColor.Println("   claude-monitor service install")
		return nil
	}
	
	// Get detailed status
	status, err := monitor.Status()
	if err != nil {
		errorColor.Printf("❌ Failed to get service status: %v\n", err)
		return nil
	}
	
	// Display status
	fmt.Println()
	displayInterfacesServiceStatus(status)
	
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

/**
 * CONTEXT:   Service logs command using ServiceInstaller and ServiceMonitor interfaces
 * INPUT:     Log retrieval parameters and line count
 * OUTPUT:    Service logs using focused interfaces
 * BUSINESS:  Logs command uses installer and monitor interfaces (Interface Segregation Principle)
 * CHANGE:    Updated to use segregated interfaces instead of fat ServiceManager
 * RISK:      Low - Read-only log retrieval with improved interface segregation
 */
func runServiceLogs(cmd *cobra.Command, args []string) error {
	// Create composite service manager
	compositeManager, err := NewCompositeServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Use ServiceInstaller interface for installation check
	var installer interfaces.ServiceInstaller = compositeManager
	// Use ServiceMonitor interface for log retrieval
	var monitor interfaces.ServiceMonitor = compositeManager
	
	if !installer.IsInstalled() {
		return fmt.Errorf("service is not installed")
	}
	
	logs, err := monitor.GetLogs(serviceLogLines)
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

/**
 * CONTEXT:   Service uninstall command using ServiceInstaller and ServiceController interfaces
 * INPUT:     Uninstall command parameters
 * OUTPUT:    Uninstalled service using focused interfaces
 * BUSINESS:  Uninstall command uses installer and controller interfaces (Interface Segregation Principle)
 * CHANGE:    Updated to use segregated interfaces instead of fat ServiceManager
 * RISK:      High - Service uninstallation affecting system configuration
 */
func runServiceUninstall(cmd *cobra.Command, args []string) error {
	headerColor.Println("🗑️  Claude Monitor Service Uninstallation")
	fmt.Println(strings.Repeat("═", 60))
	
	// Create composite service manager
	compositeManager, err := NewCompositeServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}
	
	// Use segregated interfaces
	var installer interfaces.ServiceInstaller = compositeManager
	var controller interfaces.ServiceController = compositeManager
	
	if !installer.IsInstalled() {
		warningColor.Println("⚠️  Service is not installed")
		return nil
	}
	
	// Stop service if running
	if controller.IsRunning() {
		infoColor.Println("⏳ Stopping service...")
		if err := controller.Stop(); err != nil {
			warningColor.Printf("⚠️  Failed to stop service: %v\n", err)
		} else {
			successColor.Println("✅ Service stopped")
		}
	}
	
	// Uninstall service
	infoColor.Println("⏳ Uninstalling service...")
	if err := installer.Uninstall(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}
	
	successColor.Println("✅ Service uninstalled successfully")
	infoColor.Println("ℹ️  Configuration and data files were preserved")
	
	return nil
}