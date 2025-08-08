/**
 * CONTEXT:   Claude Monitor single self-installing binary for work hour tracking
 * INPUT:     Command line arguments determining operation mode (install, daemon, CLI)
 * OUTPUT:    Self-contained work hour tracking system with zero external dependencies
 * BUSINESS:  Provide seamless work tracking with beautiful reports and direct data input
 * CHANGE:    Simplified binary removing hook dependencies for manual usage workflow
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
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	cfg "github.com/claude-monitor/system/internal/config"
	"github.com/claude-monitor/system/internal/daemon"
	"github.com/claude-monitor/system/internal/database/sqlite"
	"github.com/claude-monitor/system/internal/reporting"
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

// Unified reporting system
var (
	unifiedDB           *sqlite.SQLiteDB
	unifiedReportingSvc *reporting.SQLiteReportingService
	unifiedAnalytics    *reporting.WorkAnalyticsEngine
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
	Long: `Claude Monitor is a unified self-contained work hour tracking system.
	
Single binary providing both CLI and daemon functionality:
- Complete work hour tracking with beautiful analytics
- Built-in HTTP daemon with production features
- Zero external dependencies or separate processes
- SQLite database with automatic schema management

INSTALLATION:
  claude-monitor install              # Self-install the unified system
  claude-monitor service install      # Install as system service
  
UNIFIED OPERATION:
  claude-monitor daemon               # Run built-in daemon service
  
SERVICE MANAGEMENT:
  claude-monitor service start        # Start unified service
  claude-monitor service stop         # Stop unified service
  claude-monitor service status       # Check service health
  
DAILY USE:
  claude-monitor today                # Show today's work summary
  claude-monitor week                 # Show weekly analytics
  
SETUP:
  claude-monitor config               # Show configuration guide`,
	
	Example: `  # Complete unified setup
  claude-monitor install              # Single binary installation
  claude-monitor daemon &             # Start built-in daemon
  
  # Or as system service
  claude-monitor service install      # Install unified service
  claude-monitor service start        # Start service
  
  # Daily usage (single binary)
  claude-monitor today                # Today's summary
  claude-monitor week --output=json   # Weekly analytics
  claude-monitor service status       # Check unified service`,
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

	// Basic validation - assets embedded
	if _, err := assets.ReadFile("assets/config-template.json"); err != nil {
		errorColor.Fprintf(os.Stderr, "❌ Missing embedded assets: %v\n", err)
		os.Exit(1)
	}

	// Add global flags to root command
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default: ~/.claude-monitor/config.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, csv)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")

	// Add core subcommands
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(versionCmd)

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

/**
 * CONTEXT:   Initialize unified reporting system with SQLite database
 * INPUT:     Database file path for SQLite connection
 * OUTPUT:    Initialized reporting service and analytics engine
 * BUSINESS:  Unified reporting eliminates daemon dependency for CLI reports
 * CHANGE:    New unified reporting initialization replacing daemon HTTP calls
 * RISK:      Medium - Database connection affecting all reporting functionality
 */
func initializeReporting(dbPath string) error {
	// Create database connection
	dbConfig := sqlite.DefaultConnectionConfig(dbPath)
	db, err := sqlite.NewSQLiteDB(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	unifiedDB = db

	// Initialize repositories
	sessionRepo := sqlite.NewSessionRepository(db)
	workBlockRepo := sqlite.NewWorkBlockRepository(db.DB())
	activityRepo := sqlite.NewActivityRepository(db.DB())
	projectRepo := sqlite.NewProjectRepository(db.DB())

	// Initialize unified reporting service
	unifiedReportingSvc = reporting.NewSQLiteReportingService(
		sessionRepo,
		workBlockRepo,
		activityRepo,
		projectRepo,
	)

	// Initialize unified analytics engine
	unifiedAnalytics = reporting.NewWorkAnalyticsEngine(
		workBlockRepo,
		activityRepo,
		projectRepo,
	)

	return nil
}

/**
 * CONTEXT:   Close unified reporting system and database connections
 * INPUT:     None - cleanup of initialized reporting resources
 * OUTPUT:    Proper cleanup of database connections and reporting services
 * BUSINESS:  Resource cleanup prevents connection leaks in CLI operations
 * CHANGE:    New cleanup function for unified reporting system
 * RISK:      Low - Resource cleanup with error logging
 */
func closeReporting() {
	if unifiedDB != nil {
		if err := unifiedDB.Close(); err != nil {
			log.Printf("Warning: failed to close database: %v", err)
		}
		unifiedDB = nil
	}
	unifiedReportingSvc = nil
	unifiedAnalytics = nil
}

/**
 * CONTEXT:   Generate daily report using unified SQLite reporting system
 * INPUT:     User ID and target date for report generation
 * OUTPUT:    Professional daily report with comprehensive analytics
 * BUSINESS:  Unified daily reports provide same functionality as daemon without HTTP dependency
 * CHANGE:    Direct database reporting replacing daemon HTTP API calls
 * RISK:      Low - Read-only reporting functionality with professional display
 */
func generateUnifiedDailyReport(userID string, date time.Time) error {
	if unifiedReportingSvc == nil {
		return fmt.Errorf("unified reporting service not initialized")
	}

	ctx := context.Background()
	report, err := unifiedReportingSvc.GenerateDailyReport(ctx, userID, date)
	if err != nil {
		return fmt.Errorf("failed to generate daily report: %w", err)
	}

	return displayEnhancedDailyReport(report, date)
}

/**
 * CONTEXT:   Get current user ID for unified reporting operations
 * INPUT:     System environment variables for user identification
 * OUTPUT:    User ID string for reporting queries
 * BUSINESS:  User identification required for all unified reporting operations
 * CHANGE:    Utility function for consistent user ID retrieval in unified system
 * RISK:      Low - Simple user identification from environment variables
 */
func getCurrentUserID() string {
	// Try multiple environment variables for user identification
	if userID := os.Getenv("USER"); userID != "" {
		return userID
	}
	if userID := os.Getenv("USERNAME"); userID != "" {
		return userID
	}
	if userID := os.Getenv("LOGNAME"); userID != "" {
		return userID
	}
	// Fallback to default user
	return "default_user"
}

/**
 * CONTEXT:   Load application configuration with fallback to defaults
 * INPUT:     Configuration files, environment variables, and default settings
 * OUTPUT:    Complete application configuration ready for daemon and CLI operation
 * BUSINESS:  Centralized configuration enables consistent behavior across all operation modes
 * CHANGE:    Initial configuration loading with integration of daemon config system
 * RISK:      Low - Configuration loading with comprehensive defaults and validation
 */
func loadConfiguration() (*AppConfig, error) {
	// Load daemon configuration using existing config system
	daemonConfig, err := cfg.LoadDaemonConfig("")
	if err != nil {
		// Fall back to default config if loading fails
		daemonConfig = cfg.NewDefaultConfig()
	}
	
	// Create AppConfig with default values
	appConfig := &AppConfig{
		Daemon: struct {
			ListenAddr             string `json:"listen_addr"`
			DatabasePath           string `json:"database_path"`
			LogLevel               string `json:"log_level"`
			EnableCORS             bool   `json:"enable_cors"`
			MaxConcurrentRequests  int    `json:"max_concurrent_requests"`
		}{
			ListenAddr:             daemonConfig.GetServerAddr(),
			DatabasePath:           daemonConfig.GetDatabasePath(),
			LogLevel:               daemonConfig.Logging.Level,
			EnableCORS:             true,
			MaxConcurrentRequests:  daemonConfig.Performance.MaxConcurrentRequests,
		},
		Session: struct {
			DurationHours   int `json:"duration_hours"`
			MaxIdleMinutes  int `json:"max_idle_minutes"`
		}{
			DurationHours:   5, // Claude session duration
			MaxIdleMinutes:  5, // Work block idle timeout
		},
		Reporting: struct {
			DefaultOutputFormat  string `json:"default_output_format"`
			EnableColors         bool   `json:"enable_colors"`
			MaxProjectsDisplay   int    `json:"max_projects_display"`
			TimeFormat           string `json:"time_format"`
		}{
			DefaultOutputFormat:  "professional",
			EnableColors:         true,
			MaxProjectsDisplay:   10,
			TimeFormat:           "15:04",
		},
		Projects: struct {
			AutoDetect        bool              `json:"auto_detect"`
			CustomNames       map[string]string `json:"custom_names"`
			TrackGitBranches  bool              `json:"track_git_branches"`
		}{
			AutoDetect:        true,
			CustomNames:       make(map[string]string),
			TrackGitBranches:  false,
		},
	}
	
	// Try to load custom configuration file if it exists
	configPath := filepath.Join(os.Getenv("HOME"), ".claude", "config.json")
	if configPath == "" {
		// Windows fallback
		if homeDir := os.Getenv("USERPROFILE"); homeDir != "" {
			configPath = filepath.Join(homeDir, ".claude", "config.json")
		}
	}
	
	if _, err := os.Stat(configPath); err == nil {
		// Configuration file exists, try to load it
		data, err := os.ReadFile(configPath)
		if err == nil {
			// Unmarshal into existing config to preserve defaults
			if err := json.Unmarshal(data, appConfig); err != nil {
				fmt.Printf("Warning: Failed to parse config file %s: %v\n", configPath, err)
				// Continue with default config
			}
		}
	}
	
	return appConfig, nil
}

/**
 * CONTEXT:   Enhanced daily report display using professional reporting system
 * INPUT:     Enhanced daily report with comprehensive work analytics
 * OUTPUT:    Beautiful daily report with professional visual design
 * BUSINESS:  Professional daily reports improve user experience and tool adoption
 * CHANGE:    Direct integration with professional display system
 * RISK:      Low - Display function with enhanced visual appeal
 */
func displayEnhancedDailyReport(report *reporting.EnhancedDailyReport, date time.Time) error {
	// Use professional display system from reporting package
	reporting.DisplayProfessionalHeader("DAILY REPORT", date.Format("Monday, January 2, 2006"))
	
	if report.TotalWorkHours == 0 {
		reporting.DisplayProfessionalEmptyState("No work activity recorded for this date. Start tracking by adding work sessions manually or via API.")
		return nil
	}

	// Professional metrics dashboard
	activeWork := time.Duration(report.TotalWorkHours * float64(time.Hour))
	totalTime := time.Duration(report.ScheduleHours * float64(time.Hour))
	claudeTime := time.Duration(report.ClaudeProcessingTime * float64(time.Hour))
	reporting.DisplayMetricsDashboard(activeWork, totalTime, report.TotalSessions, report.EfficiencyPercent, claudeTime)

	// Professional project breakdown
	if len(report.ProjectBreakdown) > 0 {
		projectData := make([]reporting.ProjectData, len(report.ProjectBreakdown))
		for i, project := range report.ProjectBreakdown {
			projectData[i] = reporting.ProjectData{
				Name:     project.Name,
				Duration: time.Duration(project.Hours * float64(time.Hour)),
				Percent:  project.Percentage,
				Sessions: project.ClaudeSessions,
			}
		}
		reporting.DisplayProfessionalProjectBreakdown(projectData)
	}

	// Professional work timeline
	if len(report.WorkBlocks) > 0 {
		workBlockData := make([]reporting.WorkBlockData, len(report.WorkBlocks))
		for i, wb := range report.WorkBlocks {
			workBlockData[i] = reporting.WorkBlockData{
				StartTime:     wb.StartTime,
				EndTime:       wb.EndTime,
				Duration:      wb.Duration,
				ProjectName:   wb.ProjectName,
				ActivityCount: wb.Activities,
			}
		}
		reporting.DisplayProfessionalWorkTimeline(workBlockData)
	}

	// Professional insights
	if len(report.Insights) > 0 {
		reporting.DisplayProfessionalInsights(report.Insights)
	}

	// Professional footer with actionable next steps
	reporting.DisplayProfessionalFooter()

	return nil
}

/**
 * CONTEXT:   Enhanced monthly report display using professional reporting system
 * INPUT:     Enhanced monthly report with comprehensive work analytics
 * OUTPUT:    Beautiful monthly report with professional visual design
 * BUSINESS:  Professional monthly reports improve long-term work tracking insights
 * CHANGE:    Initial implementation for monthly reporting display
 * RISK:      Low - Display function with no side effects
 */
func displayEnhancedMonthlyReport(report *reporting.EnhancedMonthlyReport) error {
	// Use professional display system from reporting package
	reporting.DisplayProfessionalHeader("MONTHLY REPORT", fmt.Sprintf("%s %d", report.MonthName, report.Year))
	
	if report.TotalWorkHours == 0 {
		reporting.DisplayProfessionalEmptyState("No work activity recorded for this month. Start tracking by adding work sessions manually or via API.")
		return nil
	}
	
	// Professional metrics dashboard for monthly data
	activeWork := time.Duration(report.TotalWorkHours * float64(time.Hour))
	totalTime := time.Duration(report.TotalWorkHours * float64(time.Hour)) // For monthly, use same value
	claudeTime := time.Duration(report.ClaudeUsageHours * float64(time.Hour))
	efficiency := (report.TotalWorkHours / float64(report.DaysCompleted*8)) * 100 // Assuming 8-hour work days
	
	reporting.DisplayMetricsDashboard(activeWork, totalTime, report.MonthlyStats.TotalSessions, efficiency, claudeTime)
	
	// Project breakdown if available
	if len(report.ProjectBreakdown) > 0 {
		projectData := make([]reporting.ProjectData, len(report.ProjectBreakdown))
		for i, project := range report.ProjectBreakdown {
			projectData[i] = reporting.ProjectData{
				Name:     project.Name,
				Duration: time.Duration(project.Hours * float64(time.Hour)),
				Percent:  project.Percentage,
				Sessions: 1, // Default sessions count
			}
		}
		reporting.DisplayProfessionalProjectBreakdown(projectData)
	}
	
	// Professional insights
	if len(report.Insights) > 0 {
		reporting.DisplayProfessionalInsights(report.Insights)
	}
	
	// Professional footer with actionable next steps
	reporting.DisplayProfessionalFooter()
	
	return nil
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
	targetPath := "/usr/local/bin/claude-monitor"
	if installUser {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot get user home directory: %w", err)
		}
		targetPath = filepath.Join(homeDir, ".local", "bin", "claude-monitor")
	}
	if installPath != "" {
		targetPath = installPath
	}
	
	infoColor.Printf("📂 Target location: %s\n", targetPath)
	
	// Copy binary to target location
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	input, err := os.Open(currentBinary)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer input.Close()
	output, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target binary: %w", err)
	}
	defer output.Close()
	if _, err := output.ReadFrom(input); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
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
	Short: "Run Claude Monitor daemon (unified binary mode)",
	Long: `Run Claude Monitor as a background daemon service.

This single binary provides complete daemon functionality:
- HTTP server on localhost:9193 for data input and API access
- SQLite database for work hour analytics and reporting
- Session and work block management with 5-hour windows
- Rate limiting and production middleware
- Health monitoring endpoints (/health, /status, /metrics)
- Graceful shutdown and proper resource cleanup

No separate daemon binary needed - this unified approach simplifies deployment.`,
	
	RunE: runDaemonCommand,
}

var (
	daemonPort     string
	daemonHost     string
	daemonLogLevel string
	daemonService  bool
)

func init() {
	daemonCmd.Flags().StringVar(&daemonPort, "port", cfg.DefaultDaemonPort, "HTTP server port")
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
	if daemonPort != cfg.DefaultDaemonPort {
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
	
	// Skip health monitor initialization for now
	
	// Initialize production daemon using orchestrator
	log.Printf("Starting Claude Monitor daemon on %s", config.Daemon.ListenAddr)
	log.Printf("Database: %s", dbPath)
	log.Printf("Log level: %s", config.Daemon.LogLevel)
	
	if !daemonService {
		successColor.Printf("✅ Daemon starting on %s\n", config.Daemon.ListenAddr)
		infoColor.Printf("📊 Database: %s\n", dbPath)
		infoColor.Printf("🔒 Rate limit: %d req/s\n", config.Daemon.MaxConcurrentRequests)
	}
	
	// Create daemon config from app config
	daemonConfig, err := cfg.LoadDaemonConfig("")
	if err != nil {
		return fmt.Errorf("failed to load daemon config: %w", err)
	}
	
	// Override with values from app config
	daemonConfig.Server.ListenAddr = config.Daemon.ListenAddr
	daemonConfig.Database.Path = config.Daemon.DatabasePath
	daemonConfig.Logging.Level = config.Daemon.LogLevel
	
	// Create and run production orchestrator
	orchestratorConfig := daemon.OrchestratorConfig{
		DaemonConfig: daemonConfig,
		Logger:       nil, // Will create default structured logger
	}
	
	orchestrator, err := daemon.NewOrchestrator(orchestratorConfig)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}
	
	// Run daemon (blocks until shutdown)
	if err := orchestrator.Run(); err != nil {
		return fmt.Errorf("daemon failed: %w", err)
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

	// Load configuration for database connection
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Connect directly to database for reporting (unified approach)
	dbPath := expandPath(config.Daemon.DatabasePath)
	if err := initializeReporting(dbPath); err != nil {
		return fmt.Errorf("failed to initialize reporting: %w", err)
	}
	defer closeReporting()

	// Generate report from unified reporting system
	userID := getCurrentUserID()
	return generateUnifiedDailyReport(userID, targetDate)
}


/**
 * CONTEXT:   Weekly report command using unified reporting system
 * INPUT:     Optional week start date and output format
 * OUTPUT:    Weekly report with trends and productivity patterns
 * BUSINESS:  Weekly reports show productivity trends and work patterns
 * CHANGE:    Unified weekly reporting replacing daemon HTTP dependency
 * RISK:      Low - Read-only reporting with professional display
 */
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Claude Monitor v%s\n", Version)
		fmt.Printf("Build time: %s\n", BuildTime)  
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

/**
 * CONTEXT:   Create configuration directory structure for Claude Monitor
 * INPUT:     No parameters, uses standard configuration paths
 * OUTPUT:    Configuration directory path or error
 * BUSINESS:  Ensure configuration directory exists for system setup
 * CHANGE:    Initial configuration directory creation
 * RISK:      Low - Directory creation with proper permissions
 */
func createConfigurationDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	
	return configDir, nil
}

/**
 * CONTEXT:   Generate default configuration files from embedded templates
 * INPUT:     Configuration directory path for file creation
 * OUTPUT:    Error if generation fails, nil on success
 * BUSINESS:  Provide default configuration for zero-configuration startup
 * CHANGE:    Initial configuration file generation
 * RISK:      Low - File creation with embedded templates
 */
func generateConfigurationFiles(configDir string) error {
	// Create basic config.json if it doesn't exist
	configPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := `{
  "daemon": {
    "listen_addr": "localhost:9193",
    "database_path": "~/.claude/data/claude_monitor.db",
    "log_level": "info"
  },
  "session": {
    "duration_hours": 5,
    "max_idle_minutes": 5
  },
  "reporting": {
    "default_output_format": "professional",
    "enable_colors": true,
    "time_format": "15:04"
  }
}`
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}
	
	return nil
}

/**
 * CONTEXT:   Install Claude Monitor as system service
 * INPUT:     Binary path for service installation
 * OUTPUT:    Error if installation fails, nil on success
 * BUSINESS:  Enable background service operation for automatic startup
 * CHANGE:    Basic service installation (placeholder for full implementation)
 * RISK:      Medium - System service installation requiring elevated permissions
 */
func installSystemService(binaryPath string) error {
	// This is a placeholder - full service installation would be platform-specific
	return fmt.Errorf("system service installation not yet implemented")
}

/**
 * CONTEXT:   Expand file paths with home directory and environment variables
 * INPUT:     File path potentially containing ~ or environment variables
 * OUTPUT:    Fully expanded absolute path
 * BUSINESS:  Support flexible path configuration in user environments
 * CHANGE:    Path expansion utility for configuration and database paths
 * RISK:      Low - Path expansion with fallback to original path
 */
func expandPath(path string) string {
	// Handle home directory expansion
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}
	
	// Handle environment variable expansion
	path = os.ExpandEnv(path)
	
	return path
}

// Stub functions for missing service helpers
func formatDuration(d time.Duration) string {
	return d.String()
}

type HTTPClient struct{}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{}
}

type HealthStatus struct {
	Uptime string
	ActiveSessions int
	TotalWorkBlocks int
}

func (c *HTTPClient) GetHealthStatus(url string) (*HealthStatus, error) {
	return &HealthStatus{Uptime: "0s", ActiveSessions: 0, TotalWorkBlocks: 0}, fmt.Errorf("daemon not implemented")
}

/**
 * CONTEXT:   Initialize database schema and ensure data directory exists
 * INPUT:     Database file path for SQLite database
 * OUTPUT:    Error if initialization fails, nil on success
 * BUSINESS:  Ensure database is ready for work tracking operations
 * CHANGE:    Updated database initialization using proper SQLite configuration
 * RISK:      Medium - Database initialization critical for system operation
 */
func initializeDatabase(dbPath string) error {
	// Expand path (handle ~ and environment variables)
	expandedPath := expandPath(dbPath)
	
	// Create database directory
	dbDir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}
	
	// Initialize database connection and schema
	dbConfig := sqlite.DefaultConnectionConfig(expandedPath)
	db, err := sqlite.NewSQLiteDB(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()
	
	return nil
}
