/**
 * CONTEXT:   Service permission validation for cross-platform service installation
 * INPUT:     System platform detection, user permissions, and service installation requirements
 * OUTPUT:    Permission validation ensuring secure service deployment
 * BUSINESS:  Permission checks prevent unauthorized service installation and ensure proper security
 * CHANGE:    Extracted from service.go - focused permission validation with platform-specific checks
 * RISK:      Medium - Permission validation affecting service installation security
 */

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

/**
 * CONTEXT:   Cross-platform service permission validation
 * INPUT:     Current platform and user context
 * OUTPUT:    Validation result for service installation permissions
 * BUSINESS:  Service installation requires appropriate system permissions based on platform
 * CHANGE:    Platform detection with Windows and Linux permission validation
 * RISK:      Medium - Permission validation affecting service security and installation success
 */
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

/**
 * CONTEXT:   Windows administrator rights validation
 * INPUT:     Current Windows session and user context
 * OUTPUT:    Validation result for administrator privileges
 * BUSINESS:  Windows service installation requires administrator privileges
 * CHANGE:    Windows-specific permission check using 'net session' command
 * RISK:      Medium - Administrator privilege validation for secure service installation
 */
func checkWindowsAdminRights() error {
	// Try to open a handle to the SCM with full access
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("administrator privileges required - please run as administrator")
	}
	return nil
}

/**
 * CONTEXT:   Linux service installation permission validation
 * INPUT:     Linux user context, systemd directories, and service installation mode
 * OUTPUT:    Validation result for systemd service installation permissions
 * BUSINESS:  Linux service installation requires either root privileges or user systemd access
 * CHANGE:    Linux-specific permission check with user and system service differentiation
 * RISK:      Medium - Systemd permission validation affecting service installation
 */
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