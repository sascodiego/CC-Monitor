//go:build !windows

/**
 * CONTEXT:   Non-Windows platform stubs for Windows-specific service functions
 * INPUT:     Service configuration and function calls on non-Windows platforms
 * OUTPUT:    Error responses indicating Windows-only functionality
 * BUSINESS:  Platform stubs enable cross-platform compilation
 * CHANGE:    Initial stub implementation for non-Windows platforms
 * RISK:      Low - Stub functions with clear error messages
 */

package main

import "fmt"

/**
 * CONTEXT:   Windows service detection stub for non-Windows platforms
 * INPUT:     Function call attempting Windows service detection
 * OUTPUT:    Always returns false (not Windows service) with no error
 * BUSINESS:  Platform stub enables cross-platform compilation
 * CHANGE:    Added missing function stub for daemon manager compatibility
 * RISK:      Low - Simple stub returning appropriate values for non-Windows platforms
 */
func isRunningAsWindowsService() (bool, error) {
	return false, nil // Never running as Windows service on non-Windows platforms
}

// RunAsWindowsService is only available on Windows
func RunAsWindowsService(config ServiceConfig) error {
	return fmt.Errorf("Windows service mode is only supported on Windows")
}