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

// RunAsWindowsService is only available on Windows
func RunAsWindowsService(config ServiceConfig) error {
	return fmt.Errorf("Windows service mode is only supported on Windows")
}