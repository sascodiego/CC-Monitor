/**
 * CONTEXT:   Service management orchestration - DEPRECATED file being replaced by focused modules
 * INPUT:     Service management functionality split into focused files
 * OUTPUT:    This file is being deprecated in favor of service_*.go files
 * BUSINESS:  DEPRECATED - Service functionality moved to focused files following Single Responsibility Principle
 * CHANGE:    File split into service_commands.go, service_helpers.go, service_permissions.go, service_legacy.go, service_types.go
 * RISK:      Low - This file will be removed after ensuring all functionality is preserved in focused modules
 */

// DEPRECATED: This file has been split into focused modules:
// - service_commands.go: Command handlers (runService* functions)
// - service_helpers.go: Display, format, convert utilities
// - service_permissions.go: Permission checking logic
// - service_legacy.go: Deprecated functions
// - service_types.go: Types, constants, initialization
//
// This file should be removed after verifying all functionality is preserved.

package main

// All service functionality has been moved to:
// - /cmd/claude-monitor/service_commands.go
// - /cmd/claude-monitor/service_helpers.go
// - /cmd/claude-monitor/service_permissions.go
// - /cmd/claude-monitor/service_legacy.go
// - /cmd/claude-monitor/service_types.go

