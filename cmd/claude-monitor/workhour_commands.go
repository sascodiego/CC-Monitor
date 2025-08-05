package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/claude-monitor/claude-monitor/internal/cli"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-002
 * CONTEXT:   Work day command implementations with comprehensive daily tracking capabilities
 * REASON:    Users need detailed daily work hour analysis with real-time status and comprehensive reporting
 * CHANGE:    Initial implementation of work day CLI commands.
 * PREVENTION:Validate date formats and handle timezone considerations properly
 * RISK:      Low - Daily tracking commands are informational and don't affect daemon operation
 */

func createWorkDayCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	workDayCmd := &cobra.Command{
		Use:   "workday",
		Short: "Daily work hour tracking and reporting",
		Long:  "Track and analyze daily work hours with detailed breakdowns and insights",
	}

	// Work day status command
	var (
		statusDate string
		statusDetailed bool
		statusBreaks bool
		statusPattern bool
		statusLive bool
		statusInterval time.Duration
	)

	statusCmd := &cobra.Command{
		Use:   "status [date]",
		Short: "Show current work day status",
		Long: `Display current work day status with real-time updates.

Examples:
  claude-monitor workhour workday status                # Today's status  
  claude-monitor workhour workday status 2024-01-15    # Specific date
  claude-monitor workhour workday status --live        # Live updates
  claude-monitor workhour workday status --detailed    # Include breakdowns`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				statusDate = args[0]
			}
			
			// Convert CLI manager to work hour manager
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.WorkDayStatusConfig{
				Date:           statusDate,
				Detailed:       statusDetailed,
				ShowBreaks:     statusBreaks,
				ShowPattern:    statusPattern,
				LiveUpdate:     statusLive,
				UpdateInterval: statusInterval,
				Format:         format,
				Verbose:        verbose,
			}
			
			return whManager.ExecuteWorkDayStatus(config)
		},
	}

	statusCmd.Flags().BoolVar(&statusDetailed, "detailed", false, "include detailed breakdown")
	statusCmd.Flags().BoolVar(&statusBreaks, "breaks", false, "show break analysis")
	statusCmd.Flags().BoolVar(&statusPattern, "pattern", false, "show work pattern")
	statusCmd.Flags().BoolVar(&statusLive, "live", false, "live updates")
	statusCmd.Flags().DurationVar(&statusInterval, "interval", 30*time.Second, "update interval for live mode")

	// Work day report command
	var (
		reportDate string
		reportOutput string
		reportTemplate string
		reportCharts bool
		reportGoals bool
		reportTrends bool
		reportComparison []string
	)

	reportCmd := &cobra.Command{
		Use:   "report [date]",
		Short: "Generate detailed work day report",
		Long: `Generate comprehensive daily work report with optional comparisons.

Examples:
  claude-monitor workhour workday report                    # Today's report
  claude-monitor workhour workday report 2024-01-15        # Specific date  
  claude-monitor workhour workday report --charts          # Include charts
  claude-monitor workhour workday report --output=daily.pdf # Save to file`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				reportDate = args[0]
			}
			
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.WorkDayReportConfig{
				Date:            reportDate,
				OutputFile:      reportOutput,
				Template:        reportTemplate,
				IncludeCharts:   reportCharts,
				IncludeGoals:    reportGoals,
				IncludeTrends:   reportTrends,
				ComparisonDays:  reportComparison,
				Format:          format,
				Verbose:         verbose,
			}
			
			return whManager.ExecuteWorkDayReport(config)
		},
	}

	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "output file path")
	reportCmd.Flags().StringVar(&reportTemplate, "template", "standard", "report template")
	reportCmd.Flags().BoolVar(&reportCharts, "charts", false, "include visual charts")
	reportCmd.Flags().BoolVar(&reportGoals, "goals", false, "include goal progress")
	reportCmd.Flags().BoolVar(&reportTrends, "trends", false, "include trend comparison")
	reportCmd.Flags().StringSliceVar(&reportComparison, "compare", nil, "days to compare (YYYY-MM-DD)")

	// Work day export command
	var (
		exportStart string
		exportEnd string
		exportOutput string
		exportRaw bool
		exportAggregate bool
		exportCompression bool
	)

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export work day data",
		Long: `Export work day data in various formats for external analysis.

Examples:
  claude-monitor workhour workday export --output=days.csv          # Export to CSV
  claude-monitor workhour workday export --start=2024-01-01 --end=2024-01-31 # Date range
  claude-monitor workhour workday export --raw                     # Include raw data`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.WorkDayExportConfig{
				StartDate:    exportStart,
				EndDate:      exportEnd,
				OutputFile:   exportOutput,
				Format:       format,
				IncludeRaw:   exportRaw,
				Aggregate:    exportAggregate,
				Compression:  exportCompression,
				Verbose:      verbose,
			}
			
			return whManager.ExecuteWorkDayExport(config)
		},
	}

	exportCmd.Flags().StringVar(&exportStart, "start", "", "start date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportEnd, "end", "", "end date (YYYY-MM-DD)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file (required)")
	exportCmd.Flags().BoolVar(&exportRaw, "raw", false, "include raw session/block data")
	exportCmd.Flags().BoolVar(&exportAggregate, "aggregate", false, "aggregate multiple days")
	exportCmd.Flags().BoolVar(&exportCompression, "compress", false, "compress output")
	exportCmd.MarkFlagRequired("output")

	workDayCmd.AddCommand(statusCmd, reportCmd, exportCmd)
	return workDayCmd
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-003
 * CONTEXT:   Work week command implementations for weekly analysis and pattern recognition
 * REASON:    Users need weekly productivity analysis with overtime tracking and pattern insights
 * CHANGE:    Initial implementation of work week CLI commands.
 * PREVENTION:Handle week boundary calculations properly, validate standard hours formats
 * RISK:      Low - Weekly analysis commands provide reporting functionality without system impact
 */

func createWorkWeekCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	workWeekCmd := &cobra.Command{
		Use:   "workweek",
		Short: "Weekly work analysis and reporting",
		Long:  "Analyze weekly work patterns and generate comprehensive reports",
	}

	// Work week report command
	var (
		weekStart string
		weekOutput string
		weekOvertime bool
		weekPattern bool
		weekGoals bool
		weekComparison []string
		weekStandardHours string
		weekDaily bool
	)

	reportCmd := &cobra.Command{
		Use:   "report [week-start]",
		Short: "Generate weekly work report",
		Long: `Generate comprehensive weekly work analysis report.

Examples:
  claude-monitor workhour workweek report                     # Current week
  claude-monitor workhour workweek report 2024-01-15         # Week of Jan 15
  claude-monitor workhour workweek report --overtime         # Include overtime analysis
  claude-monitor workhour workweek report --daily            # Daily breakdown`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				weekStart = args[0]
			}
			
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.WorkWeekReportConfig{
				WeekStart:          weekStart,
				OutputFile:         weekOutput,
				IncludeOvertime:    weekOvertime,
				IncludePattern:     weekPattern,
				IncludeGoals:       weekGoals,
				ComparisonWeeks:    weekComparison,
				StandardHours:      weekStandardHours,
				ShowDailyBreakdown: weekDaily,
				Format:             format,
				Verbose:            verbose,
			}
			
			return whManager.ExecuteWorkWeekReport(config)
		},
	}

	reportCmd.Flags().StringVarP(&weekOutput, "output", "o", "", "output file path")
	reportCmd.Flags().BoolVar(&weekOvertime, "overtime", false, "include overtime analysis")
	reportCmd.Flags().BoolVar(&weekPattern, "pattern", false, "include work pattern analysis")
	reportCmd.Flags().BoolVar(&weekGoals, "goals", false, "include goal tracking")
	reportCmd.Flags().StringSliceVar(&weekComparison, "compare", nil, "previous weeks for comparison")
	reportCmd.Flags().StringVar(&weekStandardHours, "standard-hours", "40h", "standard work hours")
	reportCmd.Flags().BoolVar(&weekDaily, "daily", false, "show daily breakdown")

	// Work week analysis command
	var (
		analysisWeekStart string
		analysisDepth string
		analysisProductivity bool
		analysisEfficiency bool
		analysisRecommendations bool
		analysisAverage bool
		analysisOutput string
	)

	analysisCmd := &cobra.Command{
		Use:   "analysis [week-start]",
		Short: "Perform advanced weekly work analysis",
		Long: `Analyze weekly work patterns with insights and recommendations.

Examples:
  claude-monitor workhour workweek analysis                      # Current week
  claude-monitor workhour workweek analysis --depth=comprehensive # Detailed analysis
  claude-monitor workhour workweek analysis --recommendations     # Include suggestions`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				analysisWeekStart = args[0]
			}
			
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.WorkWeekAnalysisConfig{
				WeekStart:              analysisWeekStart,
				AnalysisDepth:          analysisDepth,
				IncludeProductivity:    analysisProductivity,
				IncludeEfficiency:      analysisEfficiency,
				IncludeRecommendations: analysisRecommendations,
				CompareToAverage:       analysisAverage,
				OutputFile:             analysisOutput,
				Format:                 format,
				Verbose:                verbose,
			}
			
			return whManager.ExecuteWorkWeekAnalysis(config)
		},
	}

	analysisCmd.Flags().StringVar(&analysisDepth, "depth", "basic", "analysis depth (basic, detailed, comprehensive)")
	analysisCmd.Flags().BoolVar(&analysisProductivity, "productivity", false, "include productivity metrics")
	analysisCmd.Flags().BoolVar(&analysisEfficiency, "efficiency", false, "include efficiency analysis")
	analysisCmd.Flags().BoolVar(&analysisRecommendations, "recommendations", false, "include improvement suggestions")
	analysisCmd.Flags().BoolVar(&analysisAverage, "compare-average", false, "compare to historical average")
	analysisCmd.Flags().StringVarP(&analysisOutput, "output", "o", "", "output file path")

	workWeekCmd.AddCommand(reportCmd, analysisCmd)
	return workWeekCmd
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-004
 * CONTEXT:   Timesheet management commands for formal time tracking and HR integration
 * REASON:    Users need comprehensive timesheet generation, management, and export for billing and compliance
 * CHANGE:    Initial implementation of timesheet CLI commands.
 * PREVENTION:Validate timesheet policies and rounding rules, handle submission workflows properly
 * RISK:      Medium - Timesheet operations affect billing and compliance requirements
 */

func createTimesheetCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	timesheetCmd := &cobra.Command{
		Use:   "timesheet",
		Short: "Timesheet generation and management",
		Long:  "Generate, view, submit, and export timesheets for billing and HR purposes",
	}

	// Timesheet generate command
	var (
		generateEmployee string
		generatePeriod string
		generateStart string
		generateTemplate string
		generateRounding string
		generateRoundingMethod string
		generateOvertime string
		generateBreaks string
		generateOutput string
		generateAutoSubmit bool
	)

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate timesheet for period",
		Long: `Generate timesheet with policy-based time calculations.

Examples:
  claude-monitor workhour timesheet generate                    # Current period
  claude-monitor workhour timesheet generate --period=weekly   # Weekly timesheet
  claude-monitor workhour timesheet generate --rounding=15min  # 15-minute rounding`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.TimesheetGenerateConfig{
				EmployeeID:     generateEmployee,
				Period:         generatePeriod,
				StartDate:      generateStart,
				Template:       generateTemplate,
				RoundingRule:   generateRounding,
				RoundingMethod: generateRoundingMethod,
				OvertimeRules:  generateOvertime,
				BreakDeduction: generateBreaks,
				OutputFile:     generateOutput,
				AutoSubmit:     generateAutoSubmit,
				Format:         format,
				Verbose:        verbose,
			}
			
			return whManager.ExecuteTimesheetGenerate(config)
		},
	}

	generateCmd.Flags().StringVar(&generateEmployee, "employee", "", "employee ID")
	generateCmd.Flags().StringVar(&generatePeriod, "period", "weekly", "timesheet period (weekly, biweekly, monthly)")
	generateCmd.Flags().StringVar(&generateStart, "start", "", "period start date (YYYY-MM-DD)")
	generateCmd.Flags().StringVar(&generateTemplate, "template", "standard", "timesheet template")
	generateCmd.Flags().StringVar(&generateRounding, "rounding", "15min", "time rounding (15min, 30min, 1h)")
	generateCmd.Flags().StringVar(&generateRoundingMethod, "rounding-method", "nearest", "rounding method (up, down, nearest)")
	generateCmd.Flags().StringVar(&generateOvertime, "overtime", "8h", "overtime threshold")
	generateCmd.Flags().StringVar(&generateBreaks, "breaks", "30min", "break deduction")
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "output file path")
	generateCmd.Flags().BoolVar(&generateAutoSubmit, "auto-submit", false, "automatically submit timesheet")

	// Timesheet view command
	var (
		viewTimesheetID string
		viewEmployee string
		viewStart string
		viewEnd string
		viewStatus string
		viewDetails bool
		viewTotals bool
		viewGroupBy string
		viewSortBy string
	)

	viewCmd := &cobra.Command{
		Use:   "view [timesheet-id]",
		Short: "View timesheet details",
		Long: `View existing timesheets with filtering and sorting options.

Examples:
  claude-monitor workhour timesheet view                       # List recent timesheets
  claude-monitor workhour timesheet view TS-2024-001          # Specific timesheet
  claude-monitor workhour timesheet view --status=draft       # Filter by status
  claude-monitor workhour timesheet view --details            # Show detailed entries`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				viewTimesheetID = args[0]
			}
			
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.TimesheetViewConfig{
				TimesheetID: viewTimesheetID,
				EmployeeID:  viewEmployee,
				StartDate:   viewStart,
				EndDate:     viewEnd,
				Status:      viewStatus,
				ShowDetails: viewDetails,
				ShowTotals:  viewTotals,
				GroupBy:     viewGroupBy,
				SortBy:      viewSortBy,
				Format:      format,
				Verbose:     verbose,
			}
			
			return whManager.ExecuteTimesheetView(config)
		},
	}

	viewCmd.Flags().StringVar(&viewEmployee, "employee", "", "filter by employee ID")
	viewCmd.Flags().StringVar(&viewStart, "start", "", "start date filter (YYYY-MM-DD)")
	viewCmd.Flags().StringVar(&viewEnd, "end", "", "end date filter (YYYY-MM-DD)")
	viewCmd.Flags().StringVar(&viewStatus, "status", "", "filter by status (draft, submitted, approved)")
	viewCmd.Flags().BoolVar(&viewDetails, "details", false, "show detailed entries")
	viewCmd.Flags().BoolVar(&viewTotals, "totals", false, "show summary totals")
	viewCmd.Flags().StringVar(&viewGroupBy, "group-by", "", "group by (period, status, employee)")
	viewCmd.Flags().StringVar(&viewSortBy, "sort-by", "date", "sort by (date, duration, status)")

	// Timesheet submit command
	var (
		submitTimesheetID string
		submitForce bool
		submitComments string
		submitNotify bool
	)

	submitCmd := &cobra.Command{
		Use:   "submit <timesheet-id>",
		Short: "Submit timesheet for approval",
		Long: `Submit timesheet for approval with optional comments.

Examples:
  claude-monitor workhour timesheet submit TS-2024-001         # Submit timesheet
  claude-monitor workhour timesheet submit TS-2024-001 --force # Force submit
  claude-monitor workhour timesheet submit TS-2024-001 --comments="Overtime approved"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			submitTimesheetID = args[0]
			
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.TimesheetSubmitConfig{
				TimesheetID: submitTimesheetID,
				Force:       submitForce,
				Comments:    submitComments,
				Notify:      submitNotify,
				Verbose:     verbose,
			}
			
			return whManager.ExecuteTimesheetSubmit(config)
		},
	}

	submitCmd.Flags().BoolVar(&submitForce, "force", false, "force submit without validation")
	submitCmd.Flags().StringVar(&submitComments, "comments", "", "submission comments")
	submitCmd.Flags().BoolVar(&submitNotify, "notify", false, "send notification")

	// Timesheet export command
	var (
		exportTimesheetID string
		exportEmployee string
		exportStart string
		exportEnd string
		exportOutput string
		exportTemplate string
		exportSummary bool
		exportSignature bool
		exportCompress bool
	)

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export timesheet data",
		Long: `Export timesheets in various formats for HR and billing systems.

Examples:
  claude-monitor workhour timesheet export --timesheet=TS-2024-001 --output=timesheet.pdf
  claude-monitor workhour timesheet export --employee=EMP001 --format=excel
  claude-monitor workhour timesheet export --start=2024-01-01 --end=2024-01-31`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.TimesheetExportConfig{
				TimesheetID:      exportTimesheetID,
				EmployeeID:       exportEmployee,
				StartDate:        exportStart,
				EndDate:          exportEnd,
				OutputFile:       exportOutput,
				Format:           format,
				Template:         exportTemplate,
				IncludeSummary:   exportSummary,
				DigitalSignature: exportSignature,
				Compress:         exportCompress,
				Verbose:          verbose,
			}
			
			return whManager.ExecuteTimesheetExport(config)
		},
	}

	exportCmd.Flags().StringVar(&exportTimesheetID, "timesheet", "", "specific timesheet ID")
	exportCmd.Flags().StringVar(&exportEmployee, "employee", "", "employee ID for bulk export")
	exportCmd.Flags().StringVar(&exportStart, "start", "", "start date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportEnd, "end", "", "end date (YYYY-MM-DD)")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file (required)")
	exportCmd.Flags().StringVar(&exportTemplate, "template", "standard", "export template")
	exportCmd.Flags().BoolVar(&exportSummary, "summary", false, "include summary page")
	exportCmd.Flags().BoolVar(&exportSignature, "signature", false, "add digital signature")
	exportCmd.Flags().BoolVar(&exportCompress, "compress", false, "compress output")
	exportCmd.MarkFlagRequired("output")

	timesheetCmd.AddCommand(generateCmd, viewCmd, submitCmd, exportCmd)
	return timesheetCmd
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-005
 * CONTEXT:   Analytics commands for productivity insights and optimization recommendations
 * REASON:    Users need sophisticated analytics to understand work patterns and optimize productivity
 * CHANGE:    Initial implementation of analytics CLI commands.
 * PREVENTION:Validate analysis parameters and handle large datasets efficiently for performance
 * RISK:      Medium - Complex analytics could be resource intensive for large date ranges
 */

func createAnalyticsCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	analyticsCmd := &cobra.Command{
		Use:   "analytics",
		Short: "Work pattern analytics and insights",
		Long:  "Advanced analytics for productivity optimization and work pattern insights",
	}

	// Productivity analysis command
	var (
		productivityStart string
		productivityEnd string
		productivityGranularity string
		productivityPatterns bool
		productivityTrends bool
		productivityRecommendations bool
		productivityBaseline bool
		productivityMetrics []string
		productivityOutput string
		productivityCharts bool
	)

	productivityCmd := &cobra.Command{
		Use:   "productivity",
		Short: "Analyze productivity metrics",
		Long: `Analyze productivity patterns and generate optimization insights.

Examples:
  claude-monitor workhour analytics productivity                    # Default analysis
  claude-monitor workhour analytics productivity --trends          # Include trend analysis
  claude-monitor workhour analytics productivity --recommendations # Include suggestions
  claude-monitor workhour analytics productivity --granularity=hour # Hourly analysis`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.ProductivityAnalysisConfig{
				StartDate:              productivityStart,
				EndDate:                productivityEnd,
				Granularity:            productivityGranularity,
				IncludePatterns:        productivityPatterns,
				IncludeTrends:          productivityTrends,
				IncludeRecommendations: productivityRecommendations,
				CompareToBaseline:      productivityBaseline,
				MetricTypes:            productivityMetrics,
				OutputFile:             productivityOutput,
				IncludeCharts:          productivityCharts,
				Format:                 format,
				Verbose:                verbose,
			}
			
			return whManager.ExecuteProductivityAnalysis(config)
		},
	}

	productivityCmd.Flags().StringVar(&productivityStart, "start", "", "analysis start date (YYYY-MM-DD)")
	productivityCmd.Flags().StringVar(&productivityEnd, "end", "", "analysis end date (YYYY-MM-DD)")
	productivityCmd.Flags().StringVar(&productivityGranularity, "granularity", "day", "analysis granularity (hour, day, week)")
	productivityCmd.Flags().BoolVar(&productivityPatterns, "patterns", false, "include pattern analysis")
	productivityCmd.Flags().BoolVar(&productivityTrends, "trends", false, "include trend analysis")
	productivityCmd.Flags().BoolVar(&productivityRecommendations, "recommendations", false, "include optimization suggestions")
	productivityCmd.Flags().BoolVar(&productivityBaseline, "baseline", false, "compare to baseline performance")
	productivityCmd.Flags().StringSliceVar(&productivityMetrics, "metrics", nil, "specific metrics to analyze")
	productivityCmd.Flags().StringVarP(&productivityOutput, "output", "o", "", "output file path")
	productivityCmd.Flags().BoolVar(&productivityCharts, "charts", false, "include visual charts")

	// Work pattern analysis command
	var (
		patternStart string
		patternEnd string
		patternTypes []string
		patternMinData int
		patternBreaks bool
		patternPeakHours bool
		patternRecommendations bool
		patternIdeal bool
		patternOutput string
		patternVisualization string
	)

	patternCmd := &cobra.Command{
		Use:   "patterns",
		Short: "Analyze work patterns",
		Long: `Analyze work patterns to identify peak productivity periods and habits.

Examples:
  claude-monitor workhour analytics patterns                       # Basic pattern analysis
  claude-monitor workhour analytics patterns --peak-hours         # Identify peak hours
  claude-monitor workhour analytics patterns --breaks             # Include break patterns
  claude-monitor workhour analytics patterns --recommendations    # Include suggestions`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.WorkPatternAnalysisConfig{
				StartDate:              patternStart,
				EndDate:                patternEnd,
				PatternTypes:           patternTypes,
				MinDataPoints:          patternMinData,
				IncludeBreaks:          patternBreaks,
				IncludePeakHours:       patternPeakHours,
				IncludeRecommendations: patternRecommendations,
				CompareToIdeal:         patternIdeal,
				OutputFile:             patternOutput,
				VisualizationType:      patternVisualization,
				Format:                 format,
				Verbose:                verbose,
			}
			
			return whManager.ExecuteWorkPatternAnalysis(config)
		},
	}

	patternCmd.Flags().StringVar(&patternStart, "start", "", "analysis start date (YYYY-MM-DD)")
	patternCmd.Flags().StringVar(&patternEnd, "end", "", "analysis end date (YYYY-MM-DD)")
	patternCmd.Flags().StringSliceVar(&patternTypes, "types", nil, "pattern types to analyze")
	patternCmd.Flags().IntVar(&patternMinData, "min-data", 7, "minimum data points for pattern")
	patternCmd.Flags().BoolVar(&patternBreaks, "breaks", false, "analyze break patterns")
	patternCmd.Flags().BoolVar(&patternPeakHours, "peak-hours", false, "identify peak productivity hours")
	patternCmd.Flags().BoolVar(&patternRecommendations, "recommendations", false, "optimization recommendations")
	patternCmd.Flags().BoolVar(&patternIdeal, "compare-ideal", false, "compare to ideal work patterns")
	patternCmd.Flags().StringVarP(&patternOutput, "output", "o", "", "output file path")
	patternCmd.Flags().StringVar(&patternVisualization, "visualization", "chart", "visualization type")

	// Trend analysis command
	var (
		trendStart string
		trendEnd string
		trendMetrics []string
		trendPeriod string
		trendBaseline string
		trendForecasting bool
		trendSeasonality bool
		trendConfidence float64
		trendOutput string
		trendCharts bool
	)

	trendCmd := &cobra.Command{
		Use:   "trends",
		Short: "Analyze work time trends",
		Long: `Analyze long-term trends in work patterns and productivity.

Examples:
  claude-monitor workhour analytics trends                        # Default trend analysis
  claude-monitor workhour analytics trends --forecasting         # Include forecasting
  claude-monitor workhour analytics trends --seasonality         # Analyze seasonal patterns
  claude-monitor workhour analytics trends --period=monthly      # Monthly trends`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.TrendAnalysisConfig{
				StartDate:           trendStart,
				EndDate:             trendEnd,
				TrendMetrics:        trendMetrics,
				TrendPeriod:         trendPeriod,
				BaselinePeriod:      trendBaseline,
				IncludeForecasting:  trendForecasting,
				IncludeSeasonality:  trendSeasonality,
				ConfidenceLevel:     trendConfidence,
				OutputFile:          trendOutput,
				IncludeCharts:       trendCharts,
				Format:              format,
				Verbose:             verbose,
			}
			
			return whManager.ExecuteTrendAnalysis(config)
		},
	}

	trendCmd.Flags().StringVar(&trendStart, "start", "", "analysis start date (YYYY-MM-DD)")
	trendCmd.Flags().StringVar(&trendEnd, "end", "", "analysis end date (YYYY-MM-DD)")
	trendCmd.Flags().StringSliceVar(&trendMetrics, "metrics", nil, "metrics to analyze trends for")
	trendCmd.Flags().StringVar(&trendPeriod, "period", "weekly", "trend period (daily, weekly, monthly)")
	trendCmd.Flags().StringVar(&trendBaseline, "baseline", "", "baseline period for comparison")
	trendCmd.Flags().BoolVar(&trendForecasting, "forecasting", false, "include trend forecasting")
	trendCmd.Flags().BoolVar(&trendSeasonality, "seasonality", false, "analyze seasonal patterns")
	trendCmd.Flags().Float64Var(&trendConfidence, "confidence", 0.95, "statistical confidence level")
	trendCmd.Flags().StringVarP(&trendOutput, "output", "o", "", "output file path")
	trendCmd.Flags().BoolVar(&trendCharts, "charts", false, "include trend charts")

	analyticsCmd.AddCommand(productivityCmd, patternCmd, trendCmd)
	return analyticsCmd
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-006
 * CONTEXT:   Goals and policy management commands for work hour system configuration
 * REASON:    Users need to configure work hour goals and policies for proper time tracking and reporting
 * CHANGE:    Initial implementation of goals and policy CLI commands.
 * PREVENTION:Validate goal and policy parameters to ensure system consistency and prevent conflicts
 * RISK:      Medium - Policy changes affect timesheet calculations and goal tracking accuracy
 */

func createGoalsCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	goalsCmd := &cobra.Command{
		Use:   "goals",
		Short: "Work hour goals management",
		Long:  "Set and track work hour goals for productivity monitoring",
	}

	// Goals view command
	var (
		viewGoalsEmployee string
		viewGoalsPeriod string
		viewGoalsProgress bool
		viewGoalsHistory bool
		viewGoalsCharts bool
	)

	viewCmd := &cobra.Command{
		Use:   "view",
		Short: "View current work hour goals",
		Long: `Display current work hour goals and progress tracking.

Examples:
  claude-monitor workhour goals view                    # View all goals
  claude-monitor workhour goals view --progress        # Include progress
  claude-monitor workhour goals view --period=weekly   # Weekly goals only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.GoalsViewConfig{
				EmployeeID:    viewGoalsEmployee,
				Period:        viewGoalsPeriod,
				ShowProgress:  viewGoalsProgress,
				ShowHistory:   viewGoalsHistory,
				IncludeCharts: viewGoalsCharts,
				Format:        format,
				Verbose:       verbose,
			}
			
			return whManager.ExecuteGoalsView(config)
		},
	}

	viewCmd.Flags().StringVar(&viewGoalsEmployee, "employee", "", "specific employee")
	viewCmd.Flags().StringVar(&viewGoalsPeriod, "period", "", "goal period (daily, weekly, monthly)")
	viewCmd.Flags().BoolVar(&viewGoalsProgress, "progress", false, "show current progress")
	viewCmd.Flags().BoolVar(&viewGoalsHistory, "history", false, "show historical performance")
	viewCmd.Flags().BoolVar(&viewGoalsCharts, "charts", false, "include progress charts")

	// Goals set command
	var (
		setGoalsEmployee string
		setGoalsType string
		setGoalsTarget string
		setGoalsStart string
		setGoalsEnd string
		setGoalsDescription string
		setGoalsAutoReset bool
		setGoalsNotifications bool
	)

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set work hour goals",
		Long: `Set work hour goals for tracking and motivation.

Examples:
  claude-monitor workhour goals set --type=daily --target=8h      # Daily 8-hour goal
  claude-monitor workhour goals set --type=weekly --target=40h    # Weekly 40-hour goal
  claude-monitor workhour goals set --auto-reset                 # Auto-reset periods`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.GoalsSetConfig{
				EmployeeID:    setGoalsEmployee,
				GoalType:      setGoalsType,
				TargetHours:   setGoalsTarget,
				StartDate:     setGoalsStart,
				EndDate:       setGoalsEnd,
				Description:   setGoalsDescription,
				AutoReset:     setGoalsAutoReset,
				Notifications: setGoalsNotifications,
				Verbose:       verbose,
			}
			
			return whManager.ExecuteGoalsSet(config)
		},
	}

	setCmd.Flags().StringVar(&setGoalsEmployee, "employee", "", "target employee")
	setCmd.Flags().StringVar(&setGoalsType, "type", "daily", "goal type (daily, weekly, monthly)")
	setCmd.Flags().StringVar(&setGoalsTarget, "target", "", "target hours (e.g., 8h, 40h)")
	setCmd.Flags().StringVar(&setGoalsStart, "start", "", "goal start date (YYYY-MM-DD)")
	setCmd.Flags().StringVar(&setGoalsEnd, "end", "", "goal end date (YYYY-MM-DD)")
	setCmd.Flags().StringVar(&setGoalsDescription, "description", "", "goal description")
	setCmd.Flags().BoolVar(&setGoalsAutoReset, "auto-reset", false, "automatically reset goal periods")
	setCmd.Flags().BoolVar(&setGoalsNotifications, "notifications", false, "enable goal notifications")
	setCmd.MarkFlagRequired("target")

	goalsCmd.AddCommand(viewCmd, setCmd)
	return goalsCmd
}

func createPolicyCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	policyCmd := &cobra.Command{
		Use:   "policy",
		Short: "Work hour policy management",
		Long:  "Configure work hour policies for timesheet calculations and overtime rules",
	}

	// Policy view command
	var (
		viewPolicyType string
		viewPolicyDefaults bool
		viewPolicyHistory bool
		viewPolicyEmployee string
	)

	viewCmd := &cobra.Command{
		Use:   "view",
		Short: "View current work hour policies",
		Long: `Display current work hour policies and configurations.

Examples:
  claude-monitor workhour policy view                    # View all policies
  claude-monitor workhour policy view --type=overtime   # Overtime policies only
  claude-monitor workhour policy view --defaults        # Include default values`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.PolicyViewConfig{
				PolicyType:   viewPolicyType,
				ShowDefaults: viewPolicyDefaults,
				ShowHistory:  viewPolicyHistory,
				EmployeeID:   viewPolicyEmployee,
				Format:       format,
				Verbose:      verbose,
			}
			
			return whManager.ExecutePolicyView(config)
		},
	}

	viewCmd.Flags().StringVar(&viewPolicyType, "type", "", "policy type (timesheet, overtime, rounding)")
	viewCmd.Flags().BoolVar(&viewPolicyDefaults, "defaults", false, "show default policy values")
	viewCmd.Flags().BoolVar(&viewPolicyHistory, "history", false, "show policy change history")
	viewCmd.Flags().StringVar(&viewPolicyEmployee, "employee", "", "employee-specific policies")

	// Policy update command
	var (
		updatePolicyType string
		updateRoundingInterval string
		updateRoundingMethod string
		updateOvertimeThreshold string
		updateWeeklyThreshold string
		updateBreakDeduction string
		updateEffectiveDate string
		updateEmployee string
		updateReason string
	)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update work hour policies",
		Long: `Update work hour policies with new rules and thresholds.

Examples:
  claude-monitor workhour policy update --rounding-interval=15min     # 15-minute rounding
  claude-monitor workhour policy update --overtime-threshold=8h       # 8-hour overtime
  claude-monitor workhour policy update --break-deduction=30min       # 30-minute breaks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.PolicyUpdateConfig{
				PolicyType:        updatePolicyType,
				RoundingInterval:  updateRoundingInterval,
				RoundingMethod:    updateRoundingMethod,
				OvertimeThreshold: updateOvertimeThreshold,
				WeeklyThreshold:   updateWeeklyThreshold,
				BreakDeduction:    updateBreakDeduction,
				EffectiveDate:     updateEffectiveDate,
				EmployeeID:        updateEmployee,
				Reason:            updateReason,
				Verbose:           verbose,
			}
			
			return whManager.ExecutePolicyUpdate(config)
		},
	}

	updateCmd.Flags().StringVar(&updatePolicyType, "type", "timesheet", "policy type to update")
	updateCmd.Flags().StringVar(&updateRoundingInterval, "rounding-interval", "", "time rounding interval (15min, 30min, 1h)")
	updateCmd.Flags().StringVar(&updateRoundingMethod, "rounding-method", "", "rounding method (up, down, nearest)")
	updateCmd.Flags().StringVar(&updateOvertimeThreshold, "overtime-threshold", "", "daily overtime threshold")
	updateCmd.Flags().StringVar(&updateWeeklyThreshold, "weekly-threshold", "", "weekly overtime threshold")
	updateCmd.Flags().StringVar(&updateBreakDeduction, "break-deduction", "", "automatic break deduction")
	updateCmd.Flags().StringVar(&updateEffectiveDate, "effective", "", "effective date (YYYY-MM-DD)")
	updateCmd.Flags().StringVar(&updateEmployee, "employee", "", "employee-specific policy")
	updateCmd.Flags().StringVar(&updateReason, "reason", "", "reason for policy change")

	policyCmd.AddCommand(viewCmd, updateCmd)
	return policyCmd
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-WORKHOUR-007
 * CONTEXT:   Bulk operations commands for large-scale data export and system management
 * REASON:    Users need efficient bulk operations for data migration, backup, and large-scale analysis
 * CHANGE:    Initial implementation of bulk operations CLI commands.
 * PREVENTION:Implement progress indicators and proper error handling for long-running operations
 * RISK:      High - Bulk operations could impact system performance and consume significant resources
 */

func createBulkCommands(cliManager cli.EnhancedCLIManager) *cobra.Command {
	bulkCmd := &cobra.Command{
		Use:   "bulk",
		Short: "Bulk operations for work hour data",
		Long:  "Large-scale operations for data export, migration, and system management",
	}

	// Bulk export command
	var (
		exportBulkStart string
		exportBulkEnd string
		exportBulkDir string
		exportBulkTypes []string
		exportBulkFormats []string
		exportBulkCompression string
		exportBulkSplit string
		exportBulkMetadata bool
		exportBulkEmployees []string
		exportBulkParallel bool
		exportBulkConcurrency int
		exportBulkResume string
		exportBulkProgress bool
	)

	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Bulk export work hour data",
		Long: `Export large amounts of work hour data with parallel processing.

Examples:
  claude-monitor workhour bulk export --start=2024-01-01 --end=2024-12-31 --dir=/exports
  claude-monitor workhour bulk export --parallel --concurrency=4
  claude-monitor workhour bulk export --split=monthly --compression=zip`,
		RunE: func(cmd *cobra.Command, args []string) error {
			whManager, ok := cliManager.(cli.WorkHourCLIManager)
			if !ok {
				return fmt.Errorf("work hour functionality not available")
			}
			
			config := &cli.BulkExportConfig{
				StartDate:       exportBulkStart,
				EndDate:         exportBulkEnd,
				OutputDirectory: exportBulkDir,
				DataTypes:       exportBulkTypes,
				Formats:         exportBulkFormats,
				Compression:     exportBulkCompression,
				SplitByPeriod:   exportBulkSplit,
				IncludeMetadata: exportBulkMetadata,
				EmployeeFilter:  exportBulkEmployees,
				Parallel:        exportBulkParallel,
				MaxConcurrency:  exportBulkConcurrency,
				ResumeFile:      exportBulkResume,
				Verbose:         verbose,
				Progress:        exportBulkProgress,
			}
			
			return whManager.ExecuteBulkExport(config)
		},
	}

	exportCmd.Flags().StringVar(&exportBulkStart, "start", "", "export start date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportBulkEnd, "end", "", "export end date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportBulkDir, "dir", "", "output directory (required)")
	exportCmd.Flags().StringSliceVar(&exportBulkTypes, "types", []string{"workdays", "timesheets"}, "data types to export")
	exportCmd.Flags().StringSliceVar(&exportBulkFormats, "formats", []string{"csv"}, "export formats")
	exportCmd.Flags().StringVar(&exportBulkCompression, "compression", "", "compression type (zip, tar.gz)")
	exportCmd.Flags().StringVar(&exportBulkSplit, "split", "", "split files by period (daily, weekly, monthly)")
	exportCmd.Flags().BoolVar(&exportBulkMetadata, "metadata", false, "include export metadata")
	exportCmd.Flags().StringSliceVar(&exportBulkEmployees, "employees", nil, "filter by employee IDs")
	exportCmd.Flags().BoolVar(&exportBulkParallel, "parallel", false, "enable parallel processing")
	exportCmd.Flags().IntVar(&exportBulkConcurrency, "concurrency", 2, "maximum concurrent operations")
	exportCmd.Flags().StringVar(&exportBulkResume, "resume", "", "resume interrupted export")
	exportCmd.Flags().BoolVar(&exportBulkProgress, "progress", true, "show progress bar")
	exportCmd.MarkFlagRequired("dir")

	bulkCmd.AddCommand(exportCmd)
	return bulkCmd
}