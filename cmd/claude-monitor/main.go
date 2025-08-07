/**
 * CONTEXT:   Claude Monitor single self-installing binary with multiple operation modes
 * INPUT:     Command line arguments determining operation mode (install, daemon, hook, CLI)
 * OUTPUT:    Self-contained work hour tracking system with zero external dependencies
 * BUSINESS:  Provide seamless work tracking with beautiful reports and AI-optimized setup
 * CHANGE:    Complete rewrite as single binary replacing Make-based installation
 * RISK:      Medium - Core application rewrite affecting all user interactions
 */

package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

//go:embed assets/*
var assets embed.FS

// Build information (set by build process)
var (
	Version   = "1.0.0"
	BuildTime = "development"
	GitCommit = "unknown"
)

/**
 * CONTEXT:   Color definitions for consistent beautiful CLI output
 * INPUT:     Terminal color capability detection
 * OUTPUT:    Themed color scheme for different message types
 * BUSINESS:  Professional appearance enhances user confidence and adoption
 * CHANGE:    Initial color theme optimized for readability and visual hierarchy
 * RISK:      Low - Colors with fallback for no-color terminals
 */
var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warningColor = color.New(color.FgYellow, color.Bold)
	infoColor    = color.New(color.FgCyan)
	headerColor  = color.New(color.FgMagenta, color.Bold)
	dimColor     = color.New(color.FgBlack, color.Bold)
)

/**
 * CONTEXT:   Application configuration loaded from embedded template
 * INPUT:     User configuration files and embedded defaults
 * OUTPUT:    Runtime configuration for all application modes
 * BUSINESS:  Centralized configuration enables consistent behavior across modes
 * CHANGE:    Initial configuration structure with embedded defaults
 * RISK:      Low - Configuration with safe defaults and validation
 */
type AppConfig struct {
	Daemon struct {
		ListenAddr             string `json:"listen_addr"`
		DatabasePath           string `json:"database_path"`
		LogLevel               string `json:"log_level"`
		EnableCORS             bool   `json:"enable_cors"`
		MaxConcurrentRequests  int    `json:"max_concurrent_requests"`
	} `json:"daemon"`
	
	Session struct {
		DurationHours   int `json:"duration_hours"`
		MaxIdleMinutes  int `json:"max_idle_minutes"`
	} `json:"session"`
	
	Hook struct {
		Enabled         bool     `json:"enabled"`
		TimeoutMS       int      `json:"timeout_ms"`
		FallbackLog     bool     `json:"fallback_log"`
		IgnorePatterns  []string `json:"ignore_patterns"`
	} `json:"hook"`
	
	Reporting struct {
		DefaultOutputFormat  string `json:"default_output_format"`
		EnableColors         bool   `json:"enable_colors"`
		MaxProjectsDisplay   int    `json:"max_projects_display"`
		TimeFormat           string `json:"time_format"`
	} `json:"reporting"`
	
	Projects struct {
		AutoDetect        bool              `json:"auto_detect"`
		CustomNames       map[string]string `json:"custom_names"`
		TrackGitBranches  bool              `json:"track_git_branches"`
	} `json:"projects"`
}

// Global flags
var (
	configFile   string
	verbose      bool
	outputFormat string
	noColor      bool
)

/**
 * CONTEXT:   Root command defining the single binary interface
 * INPUT:     Command line arguments and subcommand routing
 * OUTPUT:    Appropriate mode execution based on subcommand
 * BUSINESS:  Single entry point simplifies user interaction and deployment
 * CHANGE:    Initial root command with comprehensive help and examples
 * RISK:      Low - Command routing with clear error messages
 */
var rootCmd = &cobra.Command{
	Use:   "claude-monitor",
	Short: "Claude Monitor - Self-Installing Work Hour Tracking",
	Long: `Claude Monitor is a self-contained work hour tracking system for Claude Code users.
	
It automatically tracks your coding activity, detects projects, and provides
beautiful analytics about your work patterns. Zero configuration required!

INSTALLATION:
  claude-monitor install              # Self-install the system
  claude-monitor service install      # Install as system service
  
OPERATION MODES:
  claude-monitor daemon               # Run as background service
  claude-monitor hook                 # Process activity (called by Claude Code)
  
SERVICE MANAGEMENT:
  claude-monitor service start        # Start system service
  claude-monitor service stop         # Stop system service
  claude-monitor service status       # Check service health
  
DAILY USE:
  claude-monitor today                # Show today's work summary
  claude-monitor week                 # Show weekly analytics
  claude-monitor status               # Check system health
  
SETUP:
  claude-monitor config               # Show Claude Code configuration guide`,
	
	Example: `  # Complete setup with system service
  claude-monitor install
  claude-monitor service install --auto-start
  claude-monitor config
  
  # Alternative manual setup
  claude-monitor install
  claude-monitor daemon &
  
  # Daily usage
  claude-monitor today
  claude-monitor week --output=json
  claude-monitor service status`,
}

/**
 * CONTEXT:   Main entry point with embedded asset initialization
 * INPUT:     Operating system environment and command line arguments
 * OUTPUT:    Executed command with appropriate exit code
 * BUSINESS:  Single binary deployment eliminates installation complexity
 * CHANGE:    Initial main function with asset validation and command execution
 * RISK:      Medium - Core entry point affecting all application functionality
 */
func main() {
	// Initialize color handling
	if noColor || os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
	}

	// Verify embedded assets
	if err := validateEmbeddedAssets(); err != nil {
		errorColor.Fprintf(os.Stderr, "❌ Invalid binary: %v\n", err)
		os.Exit(1)
	}

	// Add global flags to root command
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default: ~/.claude-monitor/config.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, csv)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	// Add all subcommands
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(hookCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(weekCmd)
	rootCmd.AddCommand(monthCmd)
	rootCmd.AddCommand(lastmonthCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(closeAllHooksCmd)

	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		errorColor.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}

/**
 * CONTEXT:   Self-installation command for system setup
 * INPUT:     Target installation directory and user permissions
 * OUTPUT:    Installed binary, configuration, and system integration
 * BUSINESS:  Self-installation eliminates complex setup procedures
 * CHANGE:    Initial installation command with cross-platform support
 * RISK:      Medium - System modification requiring careful error handling
 */
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Self-install Claude Monitor system",
	Long: `Install Claude Monitor to your system with zero external dependencies.

This command will:
- Copy the binary to /usr/local/bin (or equivalent)
- Create configuration directories
- Generate default configuration files
- Display Claude Code integration instructions

Installation requires write permissions to the target directory.`,
	
	RunE: runInstallCommand,
}

var (
	installPath   string
	installUser   bool
	installSystem bool
	installService bool
)

func init() {
	installCmd.Flags().StringVar(&installPath, "path", "", "custom installation path")
	installCmd.Flags().BoolVar(&installUser, "user", false, "install to user directory only")
	installCmd.Flags().BoolVar(&installSystem, "system", true, "install to system directory")
	installCmd.Flags().BoolVar(&installService, "service", false, "install as system service")
}

func runInstallCommand(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	
	headerColor.Println("🚀 Claude Monitor Self-Installation")
	fmt.Println(strings.Repeat("═", 50))
	
	// Detect current binary location
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine current binary location: %w", err)
	}
	
	infoColor.Printf("📍 Current binary: %s\n", currentBinary)
	
	// Determine target installation path
	targetPath, err := getInstallationPath()
	if err != nil {
		return fmt.Errorf("cannot determine installation path: %w", err)
	}
	
	infoColor.Printf("📂 Target location: %s\n", targetPath)
	
	// Copy binary to target location
	if err := copyBinaryToTarget(currentBinary, targetPath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	successColor.Printf("✅ Binary installed successfully\n")
	
	// Create configuration directory structure
	configDir, err := createConfigurationDirectory()
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}
	successColor.Printf("✅ Configuration directory: %s\n", configDir)
	
	// Generate default configuration files
	if err := generateConfigurationFiles(configDir); err != nil {
		return fmt.Errorf("failed to generate configuration: %w", err)
	}
	successColor.Printf("✅ Configuration files generated\n")
	
	// Create database directory
	dbDir := filepath.Join(configDir, "data")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}
	successColor.Printf("✅ Database directory: %s\n", dbDir)
	
	// Install system service if requested
	if installService {
		if err := installSystemService(targetPath); err != nil {
			warningColor.Printf("⚠️  Service installation failed: %v\n", err)
		} else {
			successColor.Printf("✅ System service installed\n")
		}
	}
	
	duration := time.Since(startTime)
	fmt.Println()
	successColor.Printf("🎉 Installation completed in %v\n", duration.Round(time.Millisecond))
	
	// Display next steps
	fmt.Println()
	headerColor.Println("📋 Next Steps:")
	fmt.Println("1. Start the daemon:")
	infoColor.Printf("   %s daemon &\n", targetPath)
	fmt.Println()
	fmt.Println("2. Configure Claude Code integration:")
	infoColor.Printf("   %s config\n", targetPath)
	fmt.Println()
	fmt.Println("3. Verify installation:")
	infoColor.Printf("   %s status\n", targetPath)
	
	return nil
}

/**
 * CONTEXT:   Daemon mode for background service operation
 * INPUT:     Configuration files and database connection parameters
 * OUTPUT:    HTTP server processing activity events and serving reports
 * BUSINESS:  Daemon provides real-time activity processing and data persistence
 * CHANGE:    Initial daemon implementation with embedded database and HTTP server
 * RISK:      High - Background service affecting system performance and stability
 */
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run Claude Monitor daemon",
	Long: `Run Claude Monitor as a background daemon service.

The daemon provides:
- HTTP server on localhost:9193 for activity events
- KuzuDB database for work hour analytics  
- Session and work block management
- Fallback log processing
- Health monitoring endpoint

The daemon must be running for hook integration to work.`,
	
	RunE: runDaemonCommand,
}

var (
	daemonPort     string
	daemonHost     string
	daemonLogLevel string
	daemonService  bool
)

func init() {
	daemonCmd.Flags().StringVar(&daemonPort, "port", "9193", "HTTP server port")
	daemonCmd.Flags().StringVar(&daemonHost, "host", "localhost", "HTTP server host")
	daemonCmd.Flags().StringVar(&daemonLogLevel, "log-level", "info", "log level (debug, info, warn, error)")
	daemonCmd.Flags().BoolVar(&daemonService, "service", false, "run as system service (auto-detected)")
}

func runDaemonCommand(cmd *cobra.Command, args []string) error {
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Check if running as Windows service
	isWindowsService := false
	serviceConfig := ServiceConfig{}
	
	if runtime.GOOS == "windows" {
		if running, err := isRunningAsWindowsService(); err == nil && running {
			isWindowsService = true
			daemonService = true
			
			// Load service configuration
			serviceConfig, err = getDefaultServiceConfig()
			if err != nil {
				return fmt.Errorf("failed to load service configuration: %w", err)
			}
		}
	}
	
	// Handle service mode
	if isWindowsService {
		return RunAsWindowsService(serviceConfig)
	}
	
	// Regular daemon mode
	if !daemonService {
		headerColor.Println("🔄 Starting Claude Monitor Daemon")
		fmt.Println(strings.Repeat("═", 40))
	}
	
	// Override config with command line flags
	if daemonPort != "9193" {
		config.Daemon.ListenAddr = fmt.Sprintf("%s:%s", daemonHost, daemonPort)
	}
	
	// Initialize database
	dbPath := expandPath(config.Daemon.DatabasePath)
	if err := initializeDatabase(dbPath); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	if !daemonService {
		successColor.Printf("✅ Database initialized: %s\n", dbPath)
	}
	
	// Initialize service health monitor
	healthMonitor, err := NewServiceHealthMonitor("claude-monitor")
	if err != nil {
		log.Printf("Warning: failed to initialize health monitor: %v", err)
	}
	
	// Create embedded HTTP server
	server, err := NewEmbeddedServer(EmbeddedServerConfig{
		ListenAddr:     config.Daemon.ListenAddr,
		DatabasePath:   dbPath,
		LogLevel:       config.Daemon.LogLevel,
		DurationHours:  config.Session.DurationHours,
		MaxIdleMinutes: config.Session.MaxIdleMinutes,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	
	// Set health monitor in server
	if healthMonitor != nil {
		server.SetHealthMonitor(healthMonitor)
	}
	
	if !daemonService {
		successColor.Printf("✅ Daemon listening on %s\n", config.Daemon.ListenAddr)
		infoColor.Printf("📊 Web interface: http://%s/\n", config.Daemon.ListenAddr)
		infoColor.Printf("🔍 Health check: http://%s/health\n", config.Daemon.ListenAddr)
	}
	
	// Start server in background
	if err := server.Start(); err != nil {
		return fmt.Errorf("daemon failed: %w", err)
	}
	
	// Setup graceful shutdown with signal handling (CRITICAL FIX)
	return waitForShutdownSignal(server)
}

/**
 * CONTEXT:   Signal handling for graceful daemon shutdown
 * INPUT:     SIGTERM, SIGINT signals and server instance
 * OUTPUT:    Graceful shutdown with proper resource cleanup
 * BUSINESS:  Prevents zombie processes by ensuring all monitors are stopped
 * CHANGE:    Critical fix for zombie process prevention with timeout
 * RISK:      Low - Essential signal handling for daemon reliability
 */
func waitForShutdownSignal(server *EmbeddedServer) error {
	// Create signal channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, starting graceful shutdown...", sig)
	
	// Create shutdown context with timeout (prevents hanging)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Perform graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
		// Force shutdown if graceful fails
		return server.Stop()
	}
	
	log.Printf("Daemon shutdown completed successfully")
	return nil
}

/**
 * CONTEXT:   Hook mode for ultra-fast activity event processing
 * INPUT:     Current working directory and environment context from Claude Code
 * OUTPUT:    Activity event sent to daemon in <10ms execution time
 * BUSINESS:  Hook captures every Claude action for accurate work tracking
 * CHANGE:    Optimized hook implementation with sub-10ms performance target
 * RISK:      High - Hook performance directly impacts Claude Code user experience
 */
var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Process activity event (called by Claude Code)",
	Long: `Process a single activity event from Claude Code.

This command is designed to be called as a Claude Code hook and must execute
in under 10ms to avoid impacting user experience.

The hook will:
- Auto-detect the current project from working directory
- Send activity event to daemon via HTTP
- Fall back to local file if daemon unavailable
- Exit silently on any errors to not disrupt Claude Code`,
	
	RunE: runHookCommand,
}

var (
	hookTimeout  int
	hookDebug    bool
	hookFallback bool
)

func init() {
	hookCmd.Flags().IntVar(&hookTimeout, "timeout", 50, "HTTP timeout in milliseconds")
	hookCmd.Flags().BoolVar(&hookDebug, "debug", false, "enable debug output")
	hookCmd.Flags().BoolVar(&hookFallback, "fallback", true, "enable fallback logging")
}

func runHookCommand(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		if hookDebug && duration > 10*time.Millisecond {
			warningColor.Printf("⚠️  Hook took %v (target: <10ms)\n", duration)
		}
	}()

	// Load configuration quickly
	config, err := loadConfiguration()
	if err != nil || !config.Hook.Enabled {
		if hookDebug {
			infoColor.Printf("ℹ️  Hook disabled or config unavailable\n")
		}
		return nil
	}

	// Create activity event from environment
	activityEvent, err := createActivityEventFromEnvironment()
	if err != nil {
		if hookDebug {
			warningColor.Printf("⚠️  Failed to create activity event: %v\n", err)
		}
		return nil // Silent failure for hooks
	}

	// Send to daemon with tight timeout
	client := NewHTTPClient(time.Duration(hookTimeout) * time.Millisecond)
	
	daemonURL := fmt.Sprintf("http://%s/activity", config.Daemon.ListenAddr)
	if err := client.SendActivityEvent(daemonURL, activityEvent); err != nil {
		if hookDebug {
			warningColor.Printf("⚠️  Daemon unavailable: %v\n", err)
		}
		
		// Fallback to local file
		if hookFallback && config.Hook.FallbackLog {
			if err := writeActivityToFallbackLog(activityEvent); err != nil && hookDebug {
				warningColor.Printf("⚠️  Fallback failed: %v\n", err)
			}
		}
		return nil // Silent failure
	}

	if hookDebug {
		successColor.Printf("✅ Activity processed in %v\n", time.Since(startTime).Round(time.Microsecond))
	}
	
	return nil
}

/**
 * CONTEXT:   Today command showing daily work summary with beautiful formatting
 * INPUT:     Date parameter and output format preferences
 * OUTPUT:     Formatted daily report with project breakdown and insights
 * BUSINESS:  Daily reports are the most used feature for work tracking
 * CHANGE:    Initial today command with comprehensive daily analytics
 * RISK:      Low - Read-only reporting command with user-friendly error handling
 */
var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's work summary",
	Long: `Display a comprehensive summary of today's work activity.

Shows:
- Total work hours and schedule time
- Project breakdown with time allocation
- Work blocks and productivity patterns  
- Session information and efficiency metrics
- Insights and recommendations`,
	
	Example: `  claude-monitor today
  claude-monitor today --date=2024-08-06
  claude-monitor today --output=json`,
	
	RunE: runTodayCommand,
}

var todayDate string

func init() {
	todayCmd.Flags().StringVar(&todayDate, "date", "", "specific date (YYYY-MM-DD)")
}

func runTodayCommand(cmd *cobra.Command, args []string) error {
	// Parse target date - adjust for work day boundary (5:00 AM)
	targetDate := time.Now()
	if todayDate != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", todayDate)
		if err != nil {
			return fmt.Errorf("invalid date format: %v (use YYYY-MM-DD)", err)
		}
	} else {
		// If current time is before 5:00 AM, use previous day for work day calculation
		now := time.Now()
		if now.Hour() < 5 {
			targetDate = now.AddDate(0, 0, -1) // Previous day
		}
	}

	// Load configuration for daemon connection
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Connect to daemon for report data
	client := NewHTTPClient(5 * time.Second)
	daemonURL := fmt.Sprintf("http://%s", config.Daemon.ListenAddr)
	
	// Get daily report data
	report, err := client.GetDailyReport(daemonURL, targetDate)
	if err != nil {
		return fmt.Errorf("failed to get daily report: %w", err)
	}

	// Display report based on output format
	switch outputFormat {
	case "json":
		return outputJSON(report)
	case "csv":
		return outputCSV(report)
	default:
		return displayDailyReport(report, targetDate)
	}
}

/**
 * CONTEXT:   Close-all-hooks command for cleaning up pending hook sessions
 * INPUT:     Optional flags for dry-run and force close
 * OUTPUT:    Summary of closed sessions and work blocks
 * BUSINESS:  Cleanup command prevents orphaned sessions from hook interruptions
 * CHANGE:    Initial implementation with safe defaults and user feedback
 * RISK:      Low - Read/write operations with confirmation and dry-run support
 */
var closeAllHooksCmd = &cobra.Command{
	Use:   "close-all-hooks",
	Short: "Close all pending hook sessions",
	Long: `Close all pending hook sessions that may not have received proper end events.
	
This command is useful for cleaning up orphaned sessions that may occur when:
- Claude processes are interrupted or killed
- System shutdowns interrupt hook processing  
- Network issues prevent hook completion
- Development testing leaves open sessions

The command will:
- Find all active/pending sessions
- Mark them as closed with current timestamp
- Close any open work blocks
- Provide summary of actions taken`,
	
	Example: `  # Preview what would be closed (dry-run)
  claude-monitor close-all-hooks --dry-run
  
  # Close all pending hooks
  claude-monitor close-all-hooks
  
  # Force close without confirmation
  claude-monitor close-all-hooks --force`,
	
	RunE: runCloseAllHooksCommand,
}

var (
	closeHooksDryRun bool
	closeHooksForce  bool
)

func init() {
	closeAllHooksCmd.Flags().BoolVar(&closeHooksDryRun, "dry-run", false, "show what would be closed without making changes")
	closeAllHooksCmd.Flags().BoolVar(&closeHooksForce, "force", false, "skip confirmation prompt")
}

func runCloseAllHooksCommand(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	
	headerColor.Println("🧹 Close All Pending Hooks")
	fmt.Println(strings.Repeat("═", 50))
	
	// Load configuration for daemon connection
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Connect to daemon
	client := NewHTTPClient(10 * time.Second)
	daemonURL := fmt.Sprintf("http://%s", config.Daemon.ListenAddr)
	
	// Get pending sessions information
	pendingSessions, err := client.GetPendingSessions(daemonURL)
	if err != nil {
		return fmt.Errorf("failed to get pending sessions: %w", err)
	}
	
	if len(pendingSessions) == 0 {
		successColor.Println("✅ No pending hook sessions found")
		return nil
	}
	
	// Display what will be closed
	fmt.Printf("Found %d pending hook sessions:\n\n", len(pendingSessions))
	
	for i, session := range pendingSessions {
		fmt.Printf("  %d. Session ID: %s\n", i+1, session.ID)
		fmt.Printf("     Started: %s\n", session.StartTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("     Duration: %v\n", time.Since(session.StartTime).Round(time.Second))
		fmt.Printf("     Project: %s\n", session.ProjectName)
		fmt.Printf("     Work Blocks: %d active\n", session.ActiveWorkBlocks)
		fmt.Println()
	}
	
	// Dry-run mode - show what would happen
	if closeHooksDryRun {
		infoColor.Println("🔍 DRY-RUN MODE - No changes will be made")
		
		totalWorkBlocks := 0
		for _, session := range pendingSessions {
			totalWorkBlocks += session.ActiveWorkBlocks
		}
		
		fmt.Printf("Would close:\n")
		fmt.Printf("  • %d pending sessions\n", len(pendingSessions))
		fmt.Printf("  • %d active work blocks\n", totalWorkBlocks)
		fmt.Printf("  • All sessions marked as closed at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		
		infoColor.Printf("\nTo execute these changes, run without --dry-run flag\n")
		return nil
	}
	
	// Confirmation prompt (unless force flag is used)
	if !closeHooksForce {
		warningColor.Printf("⚠️  This will close %d pending sessions. Continue? [y/N]: ", len(pendingSessions))
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" && response != "yes" && response != "YES" {
			infoColor.Println("Operation cancelled by user")
			return nil
		}
	}
	
	// Execute the close operation
	fmt.Println()
	infoColor.Println("🔄 Closing pending sessions...")
	
	closeResult, err := client.CloseAllPendingSessions(daemonURL)
	if err != nil {
		return fmt.Errorf("failed to close pending sessions: %w", err)
	}
	
	duration := time.Since(startTime)
	
	// Display results
	fmt.Println()
	successColor.Printf("✅ Successfully closed pending hooks in %v\n", duration.Round(time.Millisecond))
	
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  • Sessions closed: %d\n", closeResult.ClosedSessions)
	fmt.Printf("  • Work blocks closed: %d\n", closeResult.ClosedWorkBlocks)
	fmt.Printf("  • Total work time recovered: %v\n", closeResult.TotalWorkTime.Round(time.Second))
	
	if closeResult.Errors > 0 {
		warningColor.Printf("  • Errors encountered: %d\n", closeResult.Errors)
		warningColor.Printf("  • Check daemon logs for details\n")
	}
	
	fmt.Println()
	infoColor.Printf("💡 Tip: Use 'claude-monitor status' to verify system health\n")
	
	return nil
}

// Additional commands (week, month, status, config, version) follow similar patterns...

/**
 * CONTEXT:   Configuration validation and embedded asset verification
 * INPUT:     Embedded file system and asset integrity checks
 * OUTPUT:    Validation results ensuring binary completeness
 * BUSINESS:  Asset validation prevents corrupted binary deployment issues
 * CHANGE:    Initial asset validation with comprehensive checks
 * RISK:      Low - Validation function with clear error reporting
 */
func validateEmbeddedAssets() error {
	requiredAssets := []string{
		"assets/config-template.json",
		"assets/schema.cypher", 
		"assets/claude-code-integration.md",
	}

	for _, asset := range requiredAssets {
		if _, err := assets.ReadFile(asset); err != nil {
			return fmt.Errorf("missing embedded asset: %s", asset)
		}
	}

	return nil
}

/**
 * CONTEXT:   Cross-platform installation path detection
 * INPUT:     Operating system and user permission context
 * OUTPUT:    Appropriate installation directory for the current platform
 * BUSINESS:  Correct installation paths ensure system integration
 * CHANGE:    Initial path detection with cross-platform support
 * RISK:      Medium - Path detection affects installation reliability
 */
func getInstallationPath() (string, error) {
	if installPath != "" {
		return installPath, nil
	}
	
	if installUser {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".local", "bin", "claude-monitor"), nil
	}
	
	switch runtime.GOOS {
	case "windows":
		return filepath.Join("C:", "Program Files", "claude-monitor", "claude-monitor.exe"), nil
	case "darwin", "linux":
		return "/usr/local/bin/claude-monitor", nil
	default:
		return "/usr/local/bin/claude-monitor", nil
	}
}

// Helper functions for various operations...

/**
 * CONTEXT:   Configuration loading with embedded defaults and user overrides
 * INPUT:     User configuration files and embedded template
 * OUTPUT:    Runtime configuration with proper fallbacks
 * BUSINESS:  Configuration system provides customization while maintaining defaults
 * CHANGE:    Initial configuration system with JSON template support
 * RISK:      Low - Configuration with safe defaults and validation
 */
func loadConfiguration() (*AppConfig, error) {
	var config AppConfig
	
	// Load embedded default configuration
	defaultConfigData, err := assets.ReadFile("assets/config-template.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load default configuration: %w", err)
	}
	
	if err := json.Unmarshal(defaultConfigData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse default configuration: %w", err)
	}
	
	// Load user configuration if available
	userConfigPath := getUserConfigPath()
	if userConfigData, err := os.ReadFile(userConfigPath); err == nil {
		// Merge user configuration over defaults
		if err := json.Unmarshal(userConfigData, &config); err != nil {
			log.Printf("Warning: Invalid user configuration, using defaults: %v", err)
		}
	}
	
	// Expand paths
	config.Daemon.DatabasePath = expandPath(config.Daemon.DatabasePath)
	
	return &config, nil
}

func getUserConfigPath() string {
	if configFile != "" {
		return configFile
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	
	return filepath.Join(homeDir, ".claude-monitor", "config.json")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// Additional helper functions for HTTP client, database initialization, etc...