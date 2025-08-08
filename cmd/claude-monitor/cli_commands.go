/**
 * CONTEXT:   CLI command definitions for Claude Monitor unified binary
 * INPUT:     Command line arguments, subcommand routing, user interaction
 * OUTPUT:    Cobra command tree with comprehensive help and validation
 * BUSINESS:  Single entry point with focused commands for work tracking operations
 * CHANGE:    Extracted from main.go to achieve Single Responsibility Principle compliance
 * RISK:      Low - Pure command definition with proper validation and error handling
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

/**
 * CONTEXT:   Root command defining the single binary interface
 * INPUT:     Command line arguments and subcommand routing
 * OUTPUT:    Appropriate mode execution based on subcommand
 * BUSINESS:  Single entry point simplifies user interaction and deployment
 * CHANGE:    Extracted from main.go for better separation of concerns
 * RISK:      Low - Command routing with clear error messages
 */
var rootCmd = &cobra.Command{
	Use:     "claude-monitor",
	Short:   "Claude Monitor - Self-Installing Work Hour Tracking",
	Long:    `Claude Monitor is a unified self-contained work hour tracking system.`,
	Example: `  claude-monitor install && claude-monitor daemon &`,
	Version: fmt.Sprintf("%s (built %s, commit %s)", Version, BuildTime, GitCommit),
}

/**
 * CONTEXT:   Installation command for zero-dependency system setup
 * INPUT:     Installation flags and system configuration
 * OUTPUT:    Fully configured Claude Monitor system ready for use
 * BUSINESS:  Self-installation eliminates deployment complexity and configuration errors
 * CHANGE:    Extracted installation command with comprehensive validation
 * RISK:      Medium - System modification requiring careful error handling
 */
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Self-install Claude Monitor system",
	Long:  `Install Claude Monitor with zero external dependencies.`,
	RunE:  runInstallCommand,
}

/**
 * CONTEXT:   Daemon command for running background HTTP server
 * INPUT:     Daemon configuration and runtime parameters
 * OUTPUT:    HTTP server with work tracking API and health monitoring
 * BUSINESS:  Background service enables continuous work tracking and data collection
 * CHANGE:    Extracted daemon command with production-grade configuration
 * RISK:      High - Background service affecting system performance and stability
 */
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run Claude Monitor daemon (unified binary mode)",
	Long:  `Run Claude Monitor as a background daemon service with HTTP API.`,
	RunE:  runDaemonCommand,
}

/**
 * CONTEXT:   Daily report command for work tracking analytics
 * INPUT:     Date specification and display preferences
 * OUTPUT:    Beautiful formatted daily work summary with analytics
 * BUSINESS:  Daily reports enable work pattern awareness and productivity insights
 * CHANGE:    Extracted reporting command with enhanced formatting
 * RISK:      Low - Read-only reporting command with user-friendly error handling
 */
var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Display today's work tracking report",
	Long:  `Generate and display a comprehensive daily work report with analytics.`,
	RunE:  runTodayCommand,
}

/**
 * CONTEXT:   Version command for build and system information
 * INPUT:     Version request and system environment
 * OUTPUT:    Version details, build information, and system diagnostics
 * BUSINESS:  Version information enables support and troubleshooting
 * CHANGE:    Extracted version command with system diagnostics
 * RISK:      Low - Read-only information command with system details
 */
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and build information",
	Long:  `Display Claude Monitor version, build details, and system information.`,
	Run:   runVersionCommand,
}

/**
 * CONTEXT:   Version command handler with system diagnostics
 * INPUT:     Command arguments and verbose flag
 * OUTPUT:    Version information and optional system details
 * BUSINESS:  Version information enables support and troubleshooting
 * CHANGE:    Extracted version handler for better organization
 * RISK:      Low - Read-only information display
 */
func runVersionCommand(cmd *cobra.Command, args []string) {
	headerColor.Printf("Claude Monitor v%s\n", Version)
	fmt.Printf("Built: %s\n", BuildTime)
	fmt.Printf("Commit: %s\n", GitCommit)
	fmt.Printf("Go: %s\n", runtime.Version())
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	
	if verbose {
		fmt.Println("\nSystem Information:")
		homeDir, _ := os.UserHomeDir()
		configDir := filepath.Join(homeDir, ".claude-monitor")
		fmt.Printf("Config Directory: %s\n", configDir)
		fmt.Printf("Database Path: %s\n", filepath.Join(configDir, "monitor.db"))
		
		if _, err := os.Stat(configDir); err == nil {
			successColor.Println("✅ Installation: Found")
		} else {
			warningColor.Println("⚠️  Installation: Not found")
		}
	}
}

/**
 * CONTEXT:   Command initialization and flag configuration
 * INPUT:     Global flags and command hierarchy setup
 * OUTPUT:    Fully configured command tree with validation
 * BUSINESS:  Proper command structure enables intuitive user experience
 * CHANGE:    Extracted command initialization from main function
 * RISK:      Low - Command setup with proper flag validation
 */
func initializeCommands() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "table", "output format")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color output")
	
	// Install command flags
	installCmd.Flags().String("db-path", "", "custom database directory path")
	installCmd.Flags().Bool("force", false, "force reinstallation")
	installCmd.Flags().Bool("skip-service", false, "skip system service installation")
	
	// Daemon command flags  
	daemonCmd.Flags().String("listen", "localhost:9193", "HTTP server listen address")
	daemonCmd.Flags().String("log-level", "info", "logging level")
	daemonCmd.Flags().Bool("cors", false, "enable CORS")
	daemonCmd.Flags().Int("max-requests", 100, "maximum concurrent requests")
	
	// Today command flags
	todayCmd.Flags().String("date", "", "specific date (YYYY-MM-DD)")
	todayCmd.Flags().Bool("json", false, "output as JSON")
	todayCmd.Flags().Bool("csv", false, "output as CSV")
	
	// Version command flags
	versionCmd.Flags().BoolVar(&verbose, "verbose", false, "show detailed system information")
	
	// Add all commands to root
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(daemonCmd) 
	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(serviceCmd) // Will be imported from service.go
	
	// Configure colors
	if noColor || os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
	}
}

/**
 * CONTEXT:   Execute root command with proper error handling
 * INPUT:     Parsed command line arguments and environment
 * OUTPUT:    Command execution with appropriate exit codes
 * BUSINESS:  Proper command execution enables reliable automation
 * CHANGE:    Extracted command execution from main function
 * RISK:      Low - Command execution with proper error handling
 */
func executeRootCommand() error {
	initializeCommands()
	return rootCmd.Execute()
}