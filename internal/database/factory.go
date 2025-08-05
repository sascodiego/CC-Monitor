/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-004
 * CONTEXT:   Database service factory for dependency injection container
 * REASON:    Clean factory pattern for database manager instantiation with proper configuration
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate database path and permissions before creating manager instance
 * RISK:      Medium - Database initialization failure if path or permissions invalid
 */
package database

import (
	"os"
	"path/filepath"

	"github.com/claude-monitor/claude-monitor/internal/arch"
)

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-004
 * CONTEXT:   Factory function for creating KuzuManager through service container
 * REASON:    Dependency injection pattern for proper service instantiation with logging
 * CHANGE:    Initial implementation.
 * PREVENTION:Use environment variables or config files for database path to avoid hardcoding
 * RISK:      Low - Factory creates instance with proper error handling
 */
func NewKuzuManagerFactory(container *arch.ServiceContainer) (interface{}, error) {
	// Get logger from container
	log, err := container.GetLogger()
	if err != nil {
		return nil, err
	}

	// Default database path (can be overridden by environment variable)
	dbPath := os.Getenv("CLAUDE_MONITOR_DB_PATH")
	if dbPath == "" {
		// Default to user's home directory under .claude-monitor
		homeDir, err := os.UserHomeDir()
		if err != nil {
			dbPath = "/tmp/claude-monitor/claude-monitor.db"
		} else {
			dbPath = filepath.Join(homeDir, ".claude-monitor", "claude-monitor.db")
		}
	}

	log.Info("Initializing database manager", 
		"dbPath", dbPath)

	return NewKuzuManager(dbPath, log)
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-025
 * CONTEXT:   Factory function for creating WorkHourManager with extended capabilities
 * REASON:    Need factory for work hour manager with comprehensive analytics and reporting features
 * CHANGE:    New factory for work hour management.
 * PREVENTION:Validate database path and ensure work hour schema is properly initialized
 * RISK:      Medium - Work hour schema initialization failure could prevent analytics functionality
 */
func NewWorkHourManagerFactory(container *arch.ServiceContainer) (interface{}, error) {
	// Get logger from container
	log, err := container.GetLogger()
	if err != nil {
		return nil, err
	}

	// Default database path (can be overridden by environment variable)
	dbPath := os.Getenv("CLAUDE_MONITOR_DB_PATH")
	if dbPath == "" {
		// Default to user's home directory under .claude-monitor
		homeDir, err := os.UserHomeDir()
		if err != nil {
			dbPath = "/tmp/claude-monitor/claude-monitor.db"
		} else {
			dbPath = filepath.Join(homeDir, ".claude-monitor", "claude-monitor.db")
		}
	}

	log.Info("Initializing work hour database manager", 
		"dbPath", dbPath,
		"features", []string{"analytics", "reporting", "timesheets", "caching"})

	return NewWorkHourManager(dbPath, log)
}