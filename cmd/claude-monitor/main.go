package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/claude-monitor/claude-monitor/internal/cli"
	"github.com/claude-monitor/claude-monitor/pkg/logger"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-001
 * CONTEXT:   Enhanced CLI entry point with comprehensive command structure and user experience features
 * REASON:    Need professional CLI interface with rich formatting, interactive features, and comprehensive options
 * CHANGE:    Enhanced from basic implementation with global flags, help system, and professional formatting.
 * PREVENTION:Validate all flag combinations and provide clear usage examples in help text
 * RISK:      Low - CLI errors are user-facing and recoverable, won't affect daemon operation
 */

var (
	version = "1.0.0"
	log     = logger.NewDefaultLogger("claude-monitor-cli", "INFO")

	// Global flags
	verbose    bool
	configFile string
	logLevel   string
	format     string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "claude-monitor",
		Short:   "Claude CLI session and work hours monitoring system",
		Long: `Claude Monitor tracks your Claude CLI usage patterns, measuring both 
session windows (5-hour periods) and active work time blocks.

The system runs as a background daemon and provides detailed reporting
of your Claude usage for productivity tracking and session management.

Examples:
  claude-monitor start                    # Start monitoring daemon
  claude-monitor status --watch           # Live status updates
  claude-monitor report --period=weekly   # Weekly usage report
  claude-monitor report --format=json     # JSON format report`,
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set up logging based on flags
			if verbose {
				logLevel = "DEBUG"
			}
			if logLevel != "" {
				log = logger.NewDefaultLogger("claude-monitor-cli", logLevel)
			}
		},
	}
	
	// Global persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output with detailed information")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.claude-monitor.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "INFO", "logging level (DEBUG, INFO, WARN, ERROR)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "table", "output format (table, json, csv, summary)")
	
	// Create enhanced CLI manager with work hour capabilities
	// TODO: Initialize work hour service when available
	cliManager := cli.NewEnhancedCLIManagerWithWorkHour(log, nil)
	
	// Add all commands
	rootCmd.AddCommand(createDaemonCommands(cliManager))
	rootCmd.AddCommand(createStatusCommand(cliManager))
	rootCmd.AddCommand(createReportCommands(cliManager))
	rootCmd.AddCommand(createConfigCommands(cliManager))
	rootCmd.AddCommand(createLogsCommand(cliManager))
	rootCmd.AddCommand(createHealthCommand(cliManager))
	rootCmd.AddCommand(createExportCommand(cliManager))
	
	// Add work hour commands
	rootCmd.AddCommand(createWorkHourCommands(cliManager))
	
	if err := rootCmd.Execute(); err != nil {
		log.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-002
 * CONTEXT:   Enhanced daemon control commands with comprehensive options and user feedback
 * REASON:    Need professional daemon lifecycle management with proper status feedback and configuration
 * CHANGE:    Enhanced from basic implementation with more options, better error handling, and user feedback.
 * PREVENTION:Always validate configuration and provide clear status updates during operations
 * RISK:      Medium - Daemon control failures could leave system in inconsistent state
 */

func createDaemonCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Daemon lifecycle management",
		Long:  "Start, stop, restart, and manage the Claude Monitor daemon",
	}
	
	// Start command
	var (
		dbPath      string
		daemonLogLevel string
		pidFile     string
		foreground  bool
	)
	
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Claude Monitor daemon",
		Long: `Start the monitoring daemon in the background. Requires root privileges 
to load eBPF programs and attach to kernel tracepoints.

Examples:
  sudo claude-monitor daemon start                    # Start with defaults
  sudo claude-monitor daemon start --foreground       # Run in foreground
  sudo claude-monitor daemon start --db-path=/custom  # Custom database path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("daemon start requires root privileges for eBPF operations")
			}
			
			config := &cli.DaemonConfig{
				DatabasePath: dbPath,
				LogLevel:     daemonLogLevel,
				PidFile:      pidFile,
				Foreground:   foreground,
				Verbose:      verbose,
				Format:       format,
			}
			
			return cliManager.ExecuteStart(config)
		},
	}
	
	startCmd.Flags().StringVar(&dbPath, "db-path", "/var/lib/claude-monitor/db", "database storage path")
	startCmd.Flags().StringVar(&daemonLogLevel, "daemon-log-level", "INFO", "daemon log level (DEBUG, INFO, WARN, ERROR)")
	startCmd.Flags().StringVar(&pidFile, "pid-file", "/var/run/claude-monitor.pid", "daemon PID file location")
	startCmd.Flags().BoolVarP(&foreground, "foreground", "F", false, "run daemon in foreground (don't daemonize)")
	
	// Stop command
	var (
		timeout time.Duration
		force   bool
	)
	
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the Claude Monitor daemon",
		Long: `Gracefully stop the monitoring daemon and finalize any active work blocks.

Examples:
  claude-monitor daemon stop                 # Graceful stop
  claude-monitor daemon stop --timeout=60s  # Custom timeout
  claude-monitor daemon stop --force        # Force stop if needed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &cli.StopConfig{
				Timeout: timeout,
				Force:   force,
				Verbose: verbose,
				Format:  format,
			}
			
			return cliManager.ExecuteStop(config)
		},
	}
	
	stopCmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "shutdown timeout duration")
	stopCmd.Flags().BoolVarP(&force, "force", "F", false, "force stop if graceful shutdown fails")
	
	// Restart command
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the Claude Monitor daemon",
		Long:  "Stop and start the daemon with the same configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliManager.ExecuteRestart(&cli.RestartConfig{
				Verbose: verbose,
				Format:  format,
			})
		},
	}
	
	daemonCmd.AddCommand(startCmd, stopCmd, restartCmd)
	return daemonCmd
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-003
 * CONTEXT:   Enhanced status command with real-time updates and rich formatting
 * REASON:    Users need comprehensive visibility into system state with professional output formatting
 * CHANGE:    Enhanced from basic implementation with watch mode, JSON output, and detailed metrics.
 * PREVENTION:Handle daemon unavailability gracefully and provide useful fallback information
 * RISK:      Low - Status command failures don't affect daemon operation
 */

func createStatusCommand(cliManager cli.EnhancedCLIManager) *cobra.Command {
	var (
		watch    bool
		interval time.Duration
		simple   bool
	)
	
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current monitoring status",
		Long: `Display current Claude Monitor status including:
- Active session information
- Current work block status  
- Today's usage summary
- System health metrics

Examples:
  claude-monitor status                    # Current status
  claude-monitor status --watch            # Live updates every 5 seconds
  claude-monitor status --format=json      # JSON output
  claude-monitor status --simple           # Minimal output`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &cli.StatusConfig{
				Watch:    watch,
				Interval: interval,
				Simple:   simple,
				Verbose:  verbose,
				Format:   format,
			}
			
			return cliManager.ExecuteStatus(config)
		},
	}
	
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch mode - update every interval")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "update interval for watch mode")
	cmd.Flags().BoolVar(&simple, "simple", false, "simple output format")
	
	return cmd
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-004
 * CONTEXT:   Enhanced reporting commands with comprehensive filtering and output options
 * REASON:    Users need flexible reporting capabilities with multiple time periods and export formats
 * CHANGE:    Enhanced from basic implementation with custom date ranges, export options, and detailed analysis.
 * PREVENTION:Validate date ranges and output parameters before processing large datasets
 * RISK:      Low - Report generation failures don't affect monitoring but impact user insights
 */

func createReportCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "Generate usage reports",
		Long: `Generate detailed usage reports for Claude Monitor data including:
- Daily, weekly, monthly summaries
- Session and work block analytics
- Export capabilities in multiple formats`,
	}
	
	// Common report flags
	var (
		outputFile string
		detailed   bool
		summaryOnly bool
	)
	
	// Daily report
	dailyCmd := &cobra.Command{
		Use:   "daily [date]",
		Short: "Generate daily usage report",
		Long: `Generate a detailed report for a specific day (default: today).

Examples:
  claude-monitor report daily                    # Today's report
  claude-monitor report daily 2024-01-15        # Specific date
  claude-monitor report daily --format=json     # JSON output
  claude-monitor report daily --output=report.csv # Save to file`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var date string
			if len(args) > 0 {
				date = args[0]
			}
			
			config := &cli.ReportConfig{
				Type:        "daily",
				Date:        date,
				OutputFile:  outputFile,
				Detailed:    detailed,
				SummaryOnly: summaryOnly,
				Verbose:     verbose,
				Format:      format,
			}
			
			return cliManager.ExecuteReport(config)
		},
	}
	
	// Weekly report
	weeklyCmd := &cobra.Command{
		Use:   "weekly [start-date]",
		Short: "Generate weekly usage report",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var startDate string
			if len(args) > 0 {
				startDate = args[0]
			}
			
			config := &cli.ReportConfig{
				Type:        "weekly",
				Date:        startDate,
				OutputFile:  outputFile,
				Detailed:    detailed,
				SummaryOnly: summaryOnly,
				Verbose:     verbose,
				Format:      format,
			}
			
			return cliManager.ExecuteReport(config)
		},
	}
	
	// Monthly report
	monthlyCmd := &cobra.Command{
		Use:   "monthly [year-month]",
		Short: "Generate monthly usage report",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var month string
			if len(args) > 0 {
				month = args[0]
			}
			
			config := &cli.ReportConfig{
				Type:        "monthly",
				Date:        month,
				OutputFile:  outputFile,
				Detailed:    detailed,
				SummaryOnly: summaryOnly,
				Verbose:     verbose,
				Format:      format,
			}
			
			return cliManager.ExecuteReport(config)
		},
	}
	
	// Custom range report
	rangeCmd := &cobra.Command{
		Use:   "range <start-date> <end-date>",
		Short: "Generate custom date range report",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &cli.ReportConfig{
				Type:        "range",
				StartDate:   args[0],
				EndDate:     args[1],
				OutputFile:  outputFile,
				Detailed:    detailed,
				SummaryOnly: summaryOnly,
				Verbose:     verbose,
				Format:      format,
			}
			
			return cliManager.ExecuteReport(config)
		},
	}
	
	// Add flags to all report commands
	for _, cmd := range []*cobra.Command{dailyCmd, weeklyCmd, monthlyCmd, rangeCmd} {
		cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (default: stdout)")
		cmd.Flags().BoolVar(&detailed, "detailed", false, "include detailed work block information")
		cmd.Flags().BoolVar(&summaryOnly, "summary-only", false, "show only summary statistics")
	}
	
	reportCmd.AddCommand(dailyCmd, weeklyCmd, monthlyCmd, rangeCmd)
	return reportCmd
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-005
 * CONTEXT:   Additional utility commands for system management and troubleshooting
 * REASON:    Users need comprehensive system management capabilities beyond basic daemon control
 * CHANGE:    New implementation of utility commands for logs, health, configuration, and export.
 * PREVENTION:Validate file permissions and system access for all utility operations
 * RISK:      Low - Utility commands are read-only or have minimal system impact
 */

func createConfigCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  "View and modify Claude Monitor configuration settings",
	}
	
	// Show configuration
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliManager.ExecuteConfigShow(&cli.ConfigShowConfig{
				Verbose: verbose,
				Format:  format,
			})
		},
	}
	
	// Set configuration
	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliManager.ExecuteConfigSet(&cli.ConfigSetConfig{
				Key:     args[0],
				Value:   args[1],
				Verbose: verbose,
			})
		},
	}
	
	configCmd.AddCommand(showCmd, setCmd)
	return configCmd
}

func createLogsCommand(cliManager cli.EnhancedCLIManager) *cobra.Command {
	var (
		follow bool
		lines  int
	)
	
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View daemon logs",
		Long:  "Display and follow Claude Monitor daemon logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &cli.LogsConfig{
				Follow:  follow,
				Lines:   lines,
				Verbose: verbose,
				Format:  format,
			}
			
			return cliManager.ExecuteLogs(config)
		},
	}
	
	cmd.Flags().BoolVar(&follow, "follow", false, "follow log output")
	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "number of lines to show")
	
	return cmd
}

func createHealthCommand(cliManager cli.EnhancedCLIManager) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "System health check",
		Long:  "Perform comprehensive system health checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &cli.HealthConfig{
				Verbose: verbose,
				Format:  format,
			}
			
			return cliManager.ExecuteHealth(config)
		},
	}
}

func createExportCommand(cliManager cli.EnhancedCLIManager) *cobra.Command {
	var (
		outputFile string
		startDate  string
		endDate    string
	)
	
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export monitoring data",
		Long:  "Export all monitoring data in various formats",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &cli.ExportConfig{
				OutputFile: outputFile,
				StartDate:  startDate,
				EndDate:    endDate,
				Verbose:    verbose,
				Format:     format,
			}
			
			return cliManager.ExecuteExport(config)
		},
	}
	
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file (required)")
	cmd.Flags().StringVar(&startDate, "start-date", "", "start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&endDate, "end-date", "", "end date (YYYY-MM-DD)")
	cmd.MarkFlagRequired("output")
	
	return cmd
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-001
 * CONTEXT:   Work hour command structure providing comprehensive work hour management
 * REASON:    Users need comprehensive work hour reporting, analytics, and management capabilities
 * CHANGE:    New work hour command suite with professional CLI interface.
 * PREVENTION:Validate all work hour command parameters and provide clear usage examples
 * RISK:      Low - Work hour CLI commands don't affect core daemon functionality
 */
func createWorkHourCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	workHourCmd := &cobra.Command{
		Use:   "workhour",
		Short: "Work hour analytics and reporting",
		Long: `Comprehensive work hour management including:
- Daily work tracking and analysis
- Weekly productivity reports  
- Timesheet generation and management
- Work pattern analytics and insights
- Goal tracking and overtime analysis

Examples:
  claude-monitor workhour status                    # Current work day status
  claude-monitor workhour daily report             # Today's detailed report
  claude-monitor workhour weekly analysis          # Weekly productivity patterns
  claude-monitor workhour timesheet generate       # Generate current timesheet`,
	}

	// Add work day commands
	workHourCmd.AddCommand(createWorkDayCommands(cliManager))
	
	// Add work week commands  
	workHourCmd.AddCommand(createWorkWeekCommands(cliManager))
	
	// Add timesheet commands
	workHourCmd.AddCommand(createTimesheetCommands(cliManager))
	
	// Add analytics commands
	workHourCmd.AddCommand(createAnalyticsCommands(cliManager))
	
	// Add goals and policy commands
	workHourCmd.AddCommand(createGoalsCommands(cliManager))
	workHourCmd.AddCommand(createPolicyCommands(cliManager))
	
	// Add bulk operations
	workHourCmd.AddCommand(createBulkCommands(cliManager))

	return workHourCmd
}