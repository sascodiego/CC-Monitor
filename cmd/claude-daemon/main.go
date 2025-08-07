/**
 * CONTEXT:   Main entry point for Claude Monitor daemon with CLI interface
 * INPUT:     Command line arguments, configuration files, and system environment
 * OUTPUT:    Running Claude Monitor daemon providing HTTP API for activity tracking
 * BUSINESS:  Provide production-ready daemon for Claude Code integration and work tracking
 * CHANGE:    Initial main implementation with CLI interface and daemon orchestration
 * RISK:      High - Main entry point affecting daemon startup, configuration, and operation
 */

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/claude-monitor/system/internal/daemon"
)

// Build information (set by build process)
var (
	Version   = "1.0.0"
	BuildTime = "development"
	GitCommit = "unknown"
)

// Command line flags
var (
	configFile = flag.String("config", "", "Path to configuration file (JSON format)")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	logFormat  = flag.String("log-format", "json", "Log format (json, text)")
	version    = flag.Bool("version", false, "Show version information and exit")
	help       = flag.Bool("help", false, "Show help information and exit")
	
	// Daemon-specific flags
	listenAddr = flag.String("listen", "localhost:8080", "HTTP server listen address")
	dbPath     = flag.String("db", "./data/claude_monitor.kuzu", "Database file path")
	pidFile    = flag.String("pid", "", "PID file path (optional)")
)

/**
 * CONTEXT:   Main function orchestrating daemon startup and configuration
 * INPUT:     Command line arguments and system environment
 * OUTPUT:    Running daemon or appropriate error messages and exit codes
 * BUSINESS:  Provide reliable daemon startup with proper error handling and configuration
 * CHANGE:    Initial main function with complete daemon lifecycle management
 * RISK:      High - Main execution path affecting daemon availability and reliability
 */
func main() {
	// Parse command line flags
	flag.Parse()
	
	// Show version if requested
	if *version {
		showVersionInfo()
		os.Exit(0)
	}
	
	// Show help if requested
	if *help {
		showHelpInfo()
		os.Exit(0)
	}
	
	// Setup basic logger for startup
	logger := setupStartupLogger()
	
	// Log startup information
	logger.Info("Starting Claude Monitor daemon",
		"version", Version,
		"build_time", BuildTime,
		"git_commit", GitCommit,
		"config_file", *configFile,
		"listen_addr", *listenAddr,
		"database_path", *dbPath)
	
	// Create PID file if specified
	if *pidFile != "" {
		if err := createPIDFile(*pidFile); err != nil {
			logger.Error("Failed to create PID file", "path", *pidFile, "error", err)
			os.Exit(1)
		}
		defer removePIDFile(*pidFile)
	}
	
	// Create daemon orchestrator
	orchestrator, err := daemon.NewOrchestrator(daemon.OrchestratorConfig{
		ConfigPath: *configFile,
		Logger:     logger,
	})
	if err != nil {
		logger.Error("Failed to initialize daemon", "error", err)
		os.Exit(1)
	}
	
	// Run daemon
	logger.Info("Claude Monitor daemon starting")
	if err := orchestrator.Run(); err != nil {
		logger.Error("Daemon execution failed", "error", err)
		os.Exit(1)
	}
	
	logger.Info("Claude Monitor daemon stopped")
}

/**
 * CONTEXT:   Setup basic logger for daemon startup and initialization
 * INPUT:     Command line log level and format flags
 * OUTPUT:    Configured slog.Logger for startup logging
 * BUSINESS:  Provide logging during daemon initialization and configuration loading
 * CHANGE:    Initial startup logger with command line flag integration
 * RISK:      Low - Startup logging configuration with fallback defaults
 */
func setupStartupLogger() *slog.Logger {
	// Parse log level
	var level slog.Level
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	
	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
		AddSource: level == slog.LevelDebug,
	}
	
	// Create appropriate handler
	var handler slog.Handler
	if *logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	
	return slog.New(handler)
}

/**
 * CONTEXT:   Display version information for daemon identification and support
 * INPUT:     Build-time version information and system details
 * OUTPUT:     Formatted version information to stdout
 * BUSINESS:  Support deployment identification and troubleshooting
 * CHANGE:    Initial version display with complete build information
 * RISK:      Low - Information display with no system changes
 */
func showVersionInfo() {
	fmt.Printf("Claude Monitor Daemon\n")
	fmt.Printf("Version:     %s\n", Version)
	fmt.Printf("Build Time:  %s\n", BuildTime)
	fmt.Printf("Git Commit:  %s\n", GitCommit)
	fmt.Printf("Go Version:  %s\n", runtime.Version())
	fmt.Printf("OS/Arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("\n")
	fmt.Printf("HTTP API:    Gorilla/mux with structured logging\n")
	fmt.Printf("Database:    KuzuDB graph database\n")
	fmt.Printf("Features:    5-hour sessions, 5-minute idle detection\n")
}

/**
 * CONTEXT:   Display help information for daemon usage and configuration
 * INPUT:     Command line flag definitions and usage information
 * OUTPUT:    Formatted help text to stdout with examples
 * BUSINESS:  Support daemon configuration and operational guidance
 * CHANGE:    Initial help display with comprehensive usage information
 * RISK:      Low - Help text display with no system changes
 */
func showHelpInfo() {
	fmt.Printf("Claude Monitor Daemon - Activity tracking for Claude Code\n\n")
	
	fmt.Printf("USAGE:\n")
	fmt.Printf("  claude-daemon [OPTIONS]\n\n")
	
	fmt.Printf("OPTIONS:\n")
	flag.PrintDefaults()
	
	fmt.Printf("\nEXAMPLES:\n")
	fmt.Printf("  # Start with default configuration\n")
	fmt.Printf("  claude-daemon\n\n")
	
	fmt.Printf("  # Start with custom configuration file\n")
	fmt.Printf("  claude-daemon -config /etc/claude-monitor/config.json\n\n")
	
	fmt.Printf("  # Start with custom listen address and database\n")
	fmt.Printf("  claude-daemon -listen localhost:8080 -db /var/lib/claude-monitor/data.kuzu\n\n")
	
	fmt.Printf("  # Start with debug logging\n")
	fmt.Printf("  claude-daemon -log-level debug -log-format text\n\n")
	
	fmt.Printf("  # Start as daemon with PID file\n")
	fmt.Printf("  claude-daemon -pid /var/run/claude-daemon.pid\n\n")
	
	fmt.Printf("ENVIRONMENT VARIABLES:\n")
	fmt.Printf("  CLAUDE_MONITOR_LISTEN_ADDR    HTTP server listen address\n")
	fmt.Printf("  CLAUDE_MONITOR_DB_PATH        Database file path\n")
	fmt.Printf("  CLAUDE_MONITOR_LOG_LEVEL      Log level (debug, info, warn, error)\n")
	fmt.Printf("  CLAUDE_MONITOR_LOG_FORMAT     Log format (json, text)\n")
	fmt.Printf("  CLAUDE_MONITOR_LOG_FILE       Log file path (optional)\n")
	
	fmt.Printf("\nCONFIGURATION:\n")
	fmt.Printf("  Configuration file format: JSON\n")
	fmt.Printf("  Default paths searched: ./config.json, ~/.claude-monitor/config.json\n")
	fmt.Printf("  Environment variables override config file values\n")
	fmt.Printf("  Command line flags override environment variables\n")
	
	fmt.Printf("\nAPI ENDPOINTS:\n")
	fmt.Printf("  POST /activity          Process Claude Code activity event\n")
	fmt.Printf("  GET  /health           System health check\n")
	fmt.Printf("  GET  /ready            Readiness check\n")
	fmt.Printf("  GET  /status           Current status (requires user_id param)\n")
	fmt.Printf("  GET  /api/v1/*         Versioned API endpoints\n")
	
	fmt.Printf("\nSIGNALS:\n")
	fmt.Printf("  SIGTERM, SIGINT        Graceful shutdown\n")
	fmt.Printf("  SIGHUP                 Reload configuration (planned)\n")
	
	fmt.Printf("\nFor more information, visit: https://github.com/claude-monitor/system\n")
}

/**
 * CONTEXT:   Create PID file for daemon process management
 * INPUT:     PID file path for daemon identification
 * OUTPUT:    PID file creation with current process ID
 * BUSINESS:  Support system service management and process monitoring
 * CHANGE:    Initial PID file creation with error handling
 * RISK:      Medium - File system operation affecting daemon management
 */
func createPIDFile(pidFilePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(pidFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create PID file directory: %w", err)
	}
	
	// Check if PID file already exists
	if _, err := os.Stat(pidFilePath); err == nil {
		// Check if process is still running
		existingPIDData, err := os.ReadFile(pidFilePath)
		if err == nil {
			return fmt.Errorf("PID file already exists: %s (contains: %s)", pidFilePath, string(existingPIDData))
		}
	}
	
	// Write current PID
	pid := os.Getpid()
	pidData := fmt.Sprintf("%d\n", pid)
	
	if err := os.WriteFile(pidFilePath, []byte(pidData), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	
	return nil
}

/**
 * CONTEXT:   Remove PID file during daemon shutdown
 * INPUT:     PID file path for cleanup
 * OUTPUT:    PID file removal with error logging
 * BUSINESS:  Clean up process management files during daemon shutdown
 * CHANGE:    Initial PID file cleanup implementation
 * RISK:      Low - File cleanup operation with error logging
 */
func removePIDFile(pidFilePath string) {
	if err := os.Remove(pidFilePath); err != nil {
		// Log error but don't fail shutdown for PID file removal
		fmt.Fprintf(os.Stderr, "Warning: failed to remove PID file %s: %v\n", pidFilePath, err)
	}
}