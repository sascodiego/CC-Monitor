/**
 * CONTEXT:   Daemon lifecycle management for Claude Monitor unified binary
 * INPUT:     Daemon configuration, service flags, runtime parameters
 * OUTPUT:    HTTP server with work tracking API and graceful shutdown
 * BUSINESS:  Background daemon enables continuous work tracking and data collection
 * CHANGE:    Extracted from main.go to separate daemon concerns from CLI routing
 * RISK:      High - Background service management affecting system stability
 */

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	cfg "github.com/claude-monitor/system/internal/config"
	"github.com/claude-monitor/system/internal/daemon"
	"github.com/claude-monitor/system/internal/database/sqlite"
	"github.com/claude-monitor/system/internal/reporting"
)

// Daemon-specific flags
var (
	daemonHost    string
	daemonPort    string
	daemonService bool
)

/**
 * CONTEXT:   Daemon command execution with configuration and lifecycle management
 * INPUT:     Command arguments and daemon configuration flags
 * OUTPUT:    Running HTTP daemon with graceful shutdown capability
 * BUSINESS:  Daemon mode provides background service for continuous work tracking
 * CHANGE:    Extracted daemon execution logic from main command handler
 * RISK:      High - Service startup and lifecycle management
 */
func runDaemonCommand(cmd *cobra.Command, args []string) error {
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Check if running as Windows service
	if runtime.GOOS == "windows" {
		if running, err := isRunningAsWindowsService(); err == nil && running {
			daemonService = true
			serviceConfig, err := getDefaultServiceConfig()
			if err != nil {
				return fmt.Errorf("failed to load service configuration: %w", err)
			}
			return RunAsWindowsService(serviceConfig)
		}
	}
	
	return runStandardDaemon(config)
}

/**
 * CONTEXT:   Standard daemon execution for non-service mode
 * INPUT:     Application configuration and runtime parameters
 * OUTPUT:    HTTP server with graceful shutdown and proper cleanup
 * BUSINESS:  Standard daemon provides development and direct execution mode
 * CHANGE:    Extracted standard daemon logic for better organization
 * RISK:      Medium - HTTP server lifecycle and resource management
 */
func runStandardDaemon(config *AppConfig) error {
	if !daemonService {
		headerColor.Println("🔄 Starting Claude Monitor Daemon")
	}
	
	// Override config with command line flags
	if daemonPort != cfg.DefaultDaemonPort {
		config.Daemon.ListenAddr = fmt.Sprintf("%s:%s", daemonHost, daemonPort)
	}
	
	// Initialize database
	if err := initializeDatabase(expandPath(config.Daemon.DatabasePath)); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	// Create and start daemon orchestrator
	orchestrator, err := createDaemonOrchestrator(config)
	if err != nil {
		return fmt.Errorf("failed to create daemon orchestrator: %w", err)
	}
	
	return startDaemon(orchestrator, config)
}

/**
 * CONTEXT:   Create daemon orchestrator with configuration
 * INPUT:     Application configuration and database setup
 * OUTPUT:    Configured daemon orchestrator ready for startup
 * BUSINESS:  Orchestrator coordinates all daemon components and middleware
 * CHANGE:    Extracted orchestrator creation for better testing
 * RISK:      Medium - Component initialization and dependency setup
 */
func createDaemonOrchestrator(config *AppConfig) (*daemon.Orchestrator, error) {
	daemonConfig := &cfg.DaemonConfig{
		Server: cfg.ServerConfig{
			ListenAddr:             config.Daemon.ListenAddr,
			MaxConcurrentRequests:  config.Daemon.MaxConcurrentRequests,
			EnableCORS:             config.Daemon.EnableCORS,
		},
		Database: cfg.DatabaseConfig{
			Path: config.Daemon.DatabasePath,
		},
	}
	
	return daemon.NewOrchestrator(daemonConfig)
}

/**
 * CONTEXT:   Start daemon with graceful shutdown handling
 * INPUT:     Configured orchestrator and application settings
 * OUTPUT:    Running daemon with signal handling and cleanup
 * BUSINESS:  Proper daemon startup enables reliable background operation
 * CHANGE:    Extracted daemon startup with signal handling
 * RISK:      High - Signal handling and graceful shutdown coordination
 */
func startDaemon(orchestrator *daemon.Orchestrator, config *AppConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Start orchestrator in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- orchestrator.Start(ctx)
	}()
	
	if !daemonService {
		successColor.Printf("✅ Daemon started on %s\n", config.Daemon.ListenAddr)
		infoColor.Println("📊 Endpoints: /health, /status, /metrics")
		fmt.Println("Press Ctrl+C to stop...")
	}
	
	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		if !daemonService {
			warningColor.Printf("🛑 Received %s, shutting down...\n", sig)
		}
		cancel()
		
		// Graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		
		if err := orchestrator.Shutdown(shutdownCtx); err != nil {
			errorColor.Printf("❌ Shutdown error: %v\n", err)
		} else if !daemonService {
			successColor.Println("✅ Daemon stopped gracefully")
		}
		return nil
		
	case err := <-errChan:
		return err
	}
}

/**
 * CONTEXT:   Database initialization with proper error handling
 * INPUT:     Database path and configuration
 * OUTPUT:    Initialized database ready for daemon operations
 * BUSINESS:  Database setup enables work tracking data persistence
 * CHANGE:    Extracted database initialization for reusability
 * RISK:      Medium - Database file creation and schema setup
 */
func initializeDatabase(dbPath string) error {
	// Ensure database directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}
	
	// Initialize global database connection
	var err error
	unifiedDB, err = sqlite.NewSQLiteDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database connection: %w", err)
	}
	
	// Initialize reporting services
	unifiedReportingSvc = reporting.NewSQLiteReportingService(unifiedDB)
	unifiedAnalytics = reporting.NewWorkAnalyticsEngine(unifiedDB)
	
	return nil
}

/**
 * CONTEXT:   Path expansion for configuration values
 * INPUT:     Path string potentially containing ~ or environment variables
 * OUTPUT:    Expanded absolute path ready for file operations
 * BUSINESS:  Path expansion enables flexible configuration management
 * CHANGE:    Extracted path utility for reuse across components
 * RISK:      Low - String manipulation with error handling
 */
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[2:])
	}
	return path
}