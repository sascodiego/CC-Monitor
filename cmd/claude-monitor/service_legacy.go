/**
 * CONTEXT:   DEPRECATED service management functions using fat ServiceManager interface
 * INPUT:     Legacy service operations and fat interface implementations
 * OUTPUT:    Service operations using deprecated monolithic interface
 * BUSINESS:  DEPRECATED legacy functions - maintained for backwards compatibility only
 * CHANGE:    DEPRECATED - Use segregated interfaces (ServiceInstaller, ServiceController, ServiceMonitor) instead
 * RISK:      Medium - Fat interface coupling violating Interface Segregation Principle
 */

package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

/**
 * CONTEXT:   DEPRECATED service command execution using fat ServiceManager interface
 * INPUT:     Operation name and service manager action function
 * OUTPUT:    Service operation result using deprecated interface
 * BUSINESS:  Legacy service command execution - replaced by segregated interface functions
 * CHANGE:    DEPRECATED - Use executeServiceControllerCommand instead
 * RISK:      Medium - Fat interface coupling - will be removed after migration
 */
// DEPRECATED: Use executeServiceControllerCommand instead
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

/**
 * CONTEXT:   DEPRECATED service manager factory for platform-specific implementations
 * INPUT:     Runtime platform detection and service configuration
 * OUTPUT:    Platform-appropriate service manager implementation using fat interface
 * BUSINESS:  DEPRECATED factory pattern - replaced by composite service manager with segregated interfaces
 * CHANGE:    DEPRECATED - Use NewCompositeServiceManager instead
 * RISK:      Medium - Platform detection affecting service functionality with fat interface
 */
// DEPRECATED: Use NewCompositeServiceManager instead
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

// DEPRECATED: NewLinuxServiceManager is now implemented in service_linux.go
// This legacy function has been removed to prevent conflicts