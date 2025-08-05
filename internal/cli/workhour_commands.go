package cli

import (
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-014
 * CONTEXT:   CLI command configurations for work hour reporting system
 * REASON:    Need comprehensive CLI interface for work hour analytics, reporting, and export
 * CHANGE:    New CLI command structures for work hour management extending existing CLI.
 * PREVENTION:Validate all user inputs, provide clear error messages, support help documentation
 * RISK:      Low - CLI errors are user-facing and don't affect daemon operation
 */

// WorkHourCLIManager extends the existing CLI manager with work hour commands
type WorkHourCLIManager interface {
	EnhancedCLIManager // Embed existing CLI interface
	
	// Work Day Commands
	ExecuteWorkDayStatus(config *WorkDayStatusConfig) error
	ExecuteWorkDayReport(config *WorkDayReportConfig) error
	ExecuteWorkDayExport(config *WorkDayExportConfig) error
	
	// Work Week Commands
	ExecuteWorkWeekReport(config *WorkWeekReportConfig) error
	ExecuteWorkWeekAnalysis(config *WorkWeekAnalysisConfig) error
	
	// Timesheet Commands
	ExecuteTimesheetGenerate(config *TimesheetGenerateConfig) error
	ExecuteTimesheetView(config *TimesheetViewConfig) error
	ExecuteTimesheetSubmit(config *TimesheetSubmitConfig) error
	ExecuteTimesheetExport(config *TimesheetExportConfig) error
	
	// Analytics Commands
	ExecuteProductivityAnalysis(config *ProductivityAnalysisConfig) error
	ExecuteWorkPatternAnalysis(config *WorkPatternAnalysisConfig) error
	ExecuteTrendAnalysis(config *TrendAnalysisConfig) error
	
	// Goal and Policy Commands
	ExecuteGoalsView(config *GoalsViewConfig) error
	ExecuteGoalsSet(config *GoalsSetConfig) error
	ExecutePolicyView(config *PolicyViewConfig) error
	ExecutePolicyUpdate(config *PolicyUpdateConfig) error
	
	// Bulk Export Commands
	ExecuteBulkExport(config *BulkExportConfig) error
	ExecuteDataMigration(config *DataMigrationConfig) error
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-015
 * CONTEXT:   Configuration structures for work day related CLI commands
 * REASON:    Need structured configuration for daily work hour analysis and reporting
 * CHANGE:    New configuration structures for work day CLI commands.
 * PREVENTION:Validate date formats and timezone handling, provide sensible defaults
 * RISK:      Low - Configuration validation catches most errors at CLI level
 */

// WorkDayStatusConfig configures the work day status command
type WorkDayStatusConfig struct {
	Date         string `json:"date"`         // Date in YYYY-MM-DD format (default: today)
	Detailed     bool   `json:"detailed"`     // Include detailed breakdown
	ShowBreaks   bool   `json:"showBreaks"`   // Show break analysis
	ShowPattern  bool   `json:"showPattern"`  // Show work pattern for day
	LiveUpdate   bool   `json:"liveUpdate"`   // Update in real-time
	UpdateInterval time.Duration `json:"updateInterval"` // Update frequency for live mode
	Timezone     string `json:"timezone"`     // Display timezone
	Format       string `json:"format"`       // Output format (table, json, summary)
	Verbose      bool   `json:"verbose"`      // Verbose output
}

// WorkDayReportConfig configures the work day report command
type WorkDayReportConfig struct {
	Date            string   `json:"date"`            // Date in YYYY-MM-DD format
	OutputFile      string   `json:"outputFile"`      // Output file path
	Template        string   `json:"template"`        // Report template
	IncludeCharts   bool     `json:"includeCharts"`   // Include visual charts
	IncludeGoals    bool     `json:"includeGoals"`    // Include goal progress
	IncludeTrends   bool     `json:"includeTrends"`   // Include trend comparison
	ComparisonDays  []string `json:"comparisonDays"`  // Days to compare against
	CustomFields    []string `json:"customFields"`    // Additional fields
	Format          string   `json:"format"`          // Output format
	Verbose         bool     `json:"verbose"`         // Verbose output
}

// WorkDayExportConfig configures the work day export command
type WorkDayExportConfig struct {
	StartDate    string `json:"startDate"`    // Start date for range export
	EndDate      string `json:"endDate"`      // End date for range export
	OutputFile   string `json:"outputFile"`   // Output file path (required)
	Format       string `json:"format"`       // Export format (csv, json, excel, pdf)
	IncludeRaw   bool   `json:"includeRaw"`   // Include raw session/block data
	Aggregate    bool   `json:"aggregate"`    // Aggregate multiple days
	Timezone     string `json:"timezone"`     // Export timezone
	Compression  bool   `json:"compression"`  // Compress output file
	Verbose      bool   `json:"verbose"`      // Verbose output
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-016
 * CONTEXT:   Configuration structures for work week and timesheet CLI commands
 * REASON:    Need structured configuration for weekly analysis and formal timesheet management
 * CHANGE:    New configuration structures for work week and timesheet CLI commands.
 * PREVENTION:Validate week boundaries and timesheet policies, handle multi-week periods
 * RISK:      Medium - Week boundary calculations and timesheet policies need careful validation
 */

// WorkWeekReportConfig configures the work week report command
type WorkWeekReportConfig struct {
	WeekStart       string   `json:"weekStart"`       // Week start date (Monday)
	OutputFile      string   `json:"outputFile"`      // Output file path
	IncludeOvertime bool     `json:"includeOvertime"` // Include overtime analysis
	IncludePattern  bool     `json:"includePattern"`  // Include work pattern analysis
	IncludeGoals    bool     `json:"includeGoals"`    // Include goal tracking
	ComparisonWeeks []string `json:"comparisonWeeks"` // Previous weeks for comparison
	StandardHours   string   `json:"standardHours"`   // Standard work hours (e.g., "40h")
	ShowDailyBreakdown bool  `json:"showDailyBreakdown"` // Show daily details
	Format          string   `json:"format"`          // Output format
	Verbose         bool     `json:"verbose"`         // Verbose output
}

// WorkWeekAnalysisConfig configures the work week analysis command
type WorkWeekAnalysisConfig struct {
	WeekStart        string `json:"weekStart"`        // Week start date
	AnalysisDepth    string `json:"analysisDepth"`    // "basic", "detailed", "comprehensive"
	IncludeProductivity bool `json:"includeProductivity"` // Productivity metrics
	IncludeEfficiency   bool `json:"includeEfficiency"`   // Efficiency analysis
	IncludeRecommendations bool `json:"includeRecommendations"` // Improvement suggestions
	CompareToAverage    bool `json:"compareToAverage"`    // Compare to historical average
	OutputFile          string `json:"outputFile"`        // Output file path
	Format              string `json:"format"`            // Output format
	Verbose             bool   `json:"verbose"`           // Verbose output
}

// TimesheetGenerateConfig configures the timesheet generation command
type TimesheetGenerateConfig struct {
	EmployeeID    string `json:"employeeId"`    // Employee identifier
	Period        string `json:"period"`        // "weekly", "biweekly", "monthly"
	StartDate     string `json:"startDate"`     // Period start date
	Template      string `json:"template"`      // Timesheet template
	RoundingRule  string `json:"roundingRule"`  // "15min", "30min", "1h"
	RoundingMethod string `json:"roundingMethod"` // "up", "down", "nearest"
	OvertimeRules string `json:"overtimeRules"` // Overtime calculation rules
	BreakDeduction string `json:"breakDeduction"` // Automatic break deduction
	OutputFile    string `json:"outputFile"`    // Output file path
	AutoSubmit    bool   `json:"autoSubmit"`    // Automatically submit timesheet
	Format        string `json:"format"`        // Output format
	Verbose       bool   `json:"verbose"`       // Verbose output
}

// TimesheetViewConfig configures the timesheet view command
type TimesheetViewConfig struct {
	TimesheetID   string `json:"timesheetId"`   // Specific timesheet ID
	EmployeeID    string `json:"employeeId"`    // Employee ID for listing
	StartDate     string `json:"startDate"`     // Date range start
	EndDate       string `json:"endDate"`       // Date range end
	Status        string `json:"status"`        // Filter by status (draft, submitted, approved)
	ShowDetails   bool   `json:"showDetails"`   // Show detailed entries
	ShowTotals    bool   `json:"showTotals"`    // Show summary totals
	GroupBy       string `json:"groupBy"`       // Group by period, status, etc.
	SortBy        string `json:"sortBy"`        // Sort by date, duration, etc.
	Format        string `json:"format"`        // Output format
	Verbose       bool   `json:"verbose"`       // Verbose output
}

// TimesheetSubmitConfig configures the timesheet submit command
type TimesheetSubmitConfig struct {
	TimesheetID string `json:"timesheetId"` // Timesheet ID to submit
	Force       bool   `json:"force"`       // Force submit without validation
	Comments    string `json:"comments"`    // Submission comments
	Notify      bool   `json:"notify"`      // Send notification
	Verbose     bool   `json:"verbose"`     // Verbose output
}

// TimesheetExportConfig configures the timesheet export command
type TimesheetExportConfig struct {
	TimesheetID string `json:"timesheetId"` // Specific timesheet to export
	EmployeeID  string `json:"employeeId"`  // Export all timesheets for employee
	StartDate   string `json:"startDate"`   // Date range start
	EndDate     string `json:"endDate"`     // Date range end
	OutputFile  string `json:"outputFile"`  // Output file path (required)
	Format      string `json:"format"`      // Export format (pdf, excel, csv)
	Template    string `json:"template"`    // Export template
	IncludeSummary bool `json:"includeSummary"` // Include summary page
	DigitalSignature bool `json:"digitalSignature"` // Add digital signature
	Compress    bool   `json:"compress"`    // Compress output
	Verbose     bool   `json:"verbose"`     // Verbose output
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-017
 * CONTEXT:   Configuration structures for analytics and pattern analysis CLI commands
 * REASON:    Need sophisticated analytics capabilities for productivity insights and optimization
 * CHANGE:    New configuration structures for analytics CLI commands.
 * PREVENTION:Validate analysis parameters and date ranges, handle large datasets efficiently
 * RISK:      Medium - Complex analytics could be resource intensive for large date ranges
 */

// ProductivityAnalysisConfig configures the productivity analysis command
type ProductivityAnalysisConfig struct {
	StartDate         string   `json:"startDate"`         // Analysis start date
	EndDate           string   `json:"endDate"`           // Analysis end date
	Granularity       string   `json:"granularity"`       // "hour", "day", "week"
	IncludePatterns   bool     `json:"includePatterns"`   // Include pattern analysis
	IncludeTrends     bool     `json:"includeTrends"`     // Include trend analysis
	IncludeRecommendations bool `json:"includeRecommendations"` // Include suggestions
	CompareToBaseline bool     `json:"compareToBaseline"` // Compare to baseline performance
	MetricTypes       []string `json:"metricTypes"`       // Specific metrics to analyze
	OutputFile        string   `json:"outputFile"`        // Output file path
	IncludeCharts     bool     `json:"includeCharts"`     // Include visual charts
	Format            string   `json:"format"`            // Output format
	Verbose           bool     `json:"verbose"`           // Verbose output
}

// WorkPatternAnalysisConfig configures the work pattern analysis command
type WorkPatternAnalysisConfig struct {
	StartDate       string   `json:"startDate"`       // Analysis start date
	EndDate         string   `json:"endDate"`         // Analysis end date
	PatternTypes    []string `json:"patternTypes"`    // Types of patterns to analyze
	MinDataPoints   int      `json:"minDataPoints"`   // Minimum data points for pattern
	IncludeBreaks   bool     `json:"includeBreaks"`   // Analyze break patterns
	IncludePeakHours bool    `json:"includePeakHours"` // Identify peak productivity hours
	IncludeRecommendations bool `json:"includeRecommendations"` // Optimization recommendations
	CompareToIdeal  bool     `json:"compareToIdeal"`  // Compare to ideal work patterns
	OutputFile      string   `json:"outputFile"`      // Output file path
	VisualizationType string `json:"visualizationType"` // Chart/graph type
	Format          string   `json:"format"`          // Output format
	Verbose         bool     `json:"verbose"`         // Verbose output
}

// TrendAnalysisConfig configures the trend analysis command
type TrendAnalysisConfig struct {
	StartDate      string   `json:"startDate"`      // Analysis start date
	EndDate        string   `json:"endDate"`        // Analysis end date
	TrendMetrics   []string `json:"trendMetrics"`   // Metrics to analyze trends for
	TrendPeriod    string   `json:"trendPeriod"`    // "daily", "weekly", "monthly"
	BaselinePeriod string   `json:"baselinePeriod"` // Period for baseline calculation
	IncludeForecasting bool `json:"includeForecasting"` // Include trend forecasting
	IncludeSeasonality bool `json:"includeSeasonality"` // Analyze seasonal patterns
	ConfidenceLevel    float64 `json:"confidenceLevel"`    // Statistical confidence level
	OutputFile         string  `json:"outputFile"`         // Output file path
	IncludeCharts      bool    `json:"includeCharts"`      // Include trend charts
	Format             string  `json:"format"`             // Output format
	Verbose            bool    `json:"verbose"`            // Verbose output
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-018
 * CONTEXT:   Configuration structures for goals, policies, and system management CLI commands
 * REASON:    Need management capabilities for work hour policies, goals, and system configuration
 * CHANGE:    New configuration structures for management CLI commands.
 * PREVENTION:Validate policy rules and goal settings, ensure backwards compatibility
 * RISK:      Medium - Policy changes affect timesheet calculations and reporting accuracy
 */

// GoalsViewConfig configures the goals view command
type GoalsViewConfig struct {
	EmployeeID    string `json:"employeeId"`    // Specific employee (for multi-user)
	Period        string `json:"period"`        // "daily", "weekly", "monthly", "yearly"
	ShowProgress  bool   `json:"showProgress"`  // Show current progress
	ShowHistory   bool   `json:"showHistory"`   // Show historical goal performance
	IncludeCharts bool   `json:"includeCharts"` // Include progress charts
	Format        string `json:"format"`        // Output format
	Verbose       bool   `json:"verbose"`       // Verbose output
}

// GoalsSetConfig configures the goals set command
type GoalsSetConfig struct {
	EmployeeID    string `json:"employeeId"`    // Target employee
	GoalType      string `json:"goalType"`      // "daily", "weekly", "monthly"
	TargetHours   string `json:"targetHours"`   // Target work hours (e.g., "8h", "40h")
	StartDate     string `json:"startDate"`     // When goal takes effect
	EndDate       string `json:"endDate"`       // Goal end date (optional)
	Description   string `json:"description"`   // Goal description
	AutoReset     bool   `json:"autoReset"`     // Automatically reset goal periods
	Notifications bool   `json:"notifications"` // Enable goal notifications
	Verbose       bool   `json:"verbose"`       // Verbose output
}

// PolicyViewConfig configures the policy view command
type PolicyViewConfig struct {
	PolicyType   string `json:"policyType"`   // "timesheet", "overtime", "rounding"
	ShowDefaults bool   `json:"showDefaults"` // Show default policy values
	ShowHistory  bool   `json:"showHistory"`  // Show policy change history
	EmployeeID   string `json:"employeeId"`   // Employee-specific policies
	Format       string `json:"format"`       // Output format
	Verbose      bool   `json:"verbose"`      // Verbose output
}

// PolicyUpdateConfig configures the policy update command
type PolicyUpdateConfig struct {
	PolicyType        string `json:"policyType"`        // Type of policy to update
	RoundingInterval  string `json:"roundingInterval"`  // "15min", "30min", "1h"
	RoundingMethod    string `json:"roundingMethod"`    // "up", "down", "nearest"
	OvertimeThreshold string `json:"overtimeThreshold"` // Daily overtime threshold
	WeeklyThreshold   string `json:"weeklyThreshold"`   // Weekly overtime threshold
	BreakDeduction    string `json:"breakDeduction"`    // Automatic break deduction
	EffectiveDate     string `json:"effectiveDate"`     // When policy takes effect
	EmployeeID        string `json:"employeeId"`        // Employee-specific policy
	Reason            string `json:"reason"`            // Reason for policy change
	Verbose           bool   `json:"verbose"`           // Verbose output
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-019
 * CONTEXT:   Configuration structures for bulk operations and data management CLI commands
 * REASON:    Need efficient bulk operations for large-scale data export and system management
 * CHANGE:    New configuration structures for bulk operations CLI commands.
 * PREVENTION:Validate large operations parameters, provide progress feedback, handle interruption gracefully
 * RISK:      High - Bulk operations could impact system performance and consume significant resources
 */

// BulkExportConfig configures the bulk export command
type BulkExportConfig struct {
	StartDate       string   `json:"startDate"`       // Export start date
	EndDate         string   `json:"endDate"`         // Export end date
	OutputDirectory string   `json:"outputDirectory"` // Output directory path
	DataTypes       []string `json:"dataTypes"`       // Types of data to export
	Formats         []string `json:"formats"`         // Export formats
	Compression     string   `json:"compression"`     // Compression type ("zip", "tar.gz")
	SplitByPeriod   string   `json:"splitByPeriod"`   // Split files by period
	IncludeMetadata bool     `json:"includeMetadata"` // Include export metadata
	EmployeeFilter  []string `json:"employeeFilter"`  // Filter by employee IDs
	Parallel        bool     `json:"parallel"`        // Parallel processing
	MaxConcurrency  int      `json:"maxConcurrency"`  // Maximum concurrent operations
	ResumeFile      string   `json:"resumeFile"`      // Resume interrupted export
	Verbose         bool     `json:"verbose"`         // Verbose output
	Progress        bool     `json:"progress"`        // Show progress bar
}

// DataMigrationConfig configures the data migration command
type DataMigrationConfig struct {
	SourceFormat    string `json:"sourceFormat"`    // Source data format
	SourcePath      string `json:"sourcePath"`      // Source data path
	TargetFormat    string `json:"targetFormat"`    // Target format
	TargetPath      string `json:"targetPath"`      // Target path
	MigrationType   string `json:"migrationType"`   // "import", "export", "transform"
	ValidationLevel string `json:"validationLevel"` // "basic", "thorough", "strict"
	BackupBefore    bool   `json:"backupBefore"`    // Create backup before migration
	DryRun          bool   `json:"dryRun"`          // Dry run without actual changes
	BatchSize       int    `json:"batchSize"`       // Processing batch size
	ErrorHandling   string `json:"errorHandling"`   // "skip", "stop", "retry"
	LogFile         string `json:"logFile"`         // Migration log file
	Verbose         bool   `json:"verbose"`         // Verbose output
	Progress        bool   `json:"progress"`        // Show progress
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-020
 * CONTEXT:   Command execution result structures for work hour CLI operations
 * REASON:    Need structured results for CLI operations that can be formatted and processed
 * CHANGE:    New result structures for work hour CLI command responses.
 * PREVENTION:Ensure result structures support all output formats and error handling
 * RISK:      Low - Result structures are data containers with minimal business logic
 */

// WorkHourCommandResult represents the result of a work hour CLI command
type WorkHourCommandResult struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	Data         interface{}            `json:"data,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
	Errors       []string               `json:"errors,omitempty"`
	ExecutionTime time.Duration         `json:"executionTime"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// WorkDayResult represents the result of work day operations
type WorkDayResult struct {
	WorkDay         *domain.WorkDay         `json:"workDay"`
	Summary         *domain.ActivitySummary `json:"summary,omitempty"`
	GoalProgress    *domain.GoalProgress    `json:"goalProgress,omitempty"`
	Recommendations []string                `json:"recommendations,omitempty"`
}

// WorkWeekResult represents the result of work week operations
type WorkWeekResult struct {
	WorkWeek        *domain.WorkWeek        `json:"workWeek"`
	Summary         *domain.ActivitySummary `json:"summary,omitempty"`
	Pattern         *domain.WorkPattern     `json:"pattern,omitempty"`
	Trends          *domain.TrendAnalysis   `json:"trends,omitempty"`
	Recommendations []string                `json:"recommendations,omitempty"`
}

// TimesheetResult represents the result of timesheet operations
type TimesheetResult struct {
	Timesheet    *domain.Timesheet     `json:"timesheet"`
	Validation   *TimesheetValidation  `json:"validation,omitempty"`
	Policy       *domain.TimesheetPolicy `json:"policy,omitempty"`
	ExportPath   string                `json:"exportPath,omitempty"`
}

// TimesheetValidation represents timesheet validation results
type TimesheetValidation struct {
	IsValid      bool     `json:"isValid"`
	Errors       []string `json:"errors,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
	PolicyApplied bool    `json:"policyApplied"`
}

// AnalyticsResult represents the result of analytics operations
type AnalyticsResult struct {
	AnalysisType    string                  `json:"analysisType"`
	Period          string                  `json:"period"`
	StartDate       time.Time               `json:"startDate"`
	EndDate         time.Time               `json:"endDate"`
	Metrics         interface{}             `json:"metrics"`
	Insights        []string                `json:"insights,omitempty"`
	Recommendations []string                `json:"recommendations,omitempty"`
	Charts          []arch.ChartData        `json:"charts,omitempty"`
}

// ExportResult represents the result of export operations
type ExportResult struct {
	OutputPath     string    `json:"outputPath"`
	Format         string    `json:"format"`
	RecordCount    int       `json:"recordCount"`
	FileSize       int64     `json:"fileSize"`
	Compression    string    `json:"compression,omitempty"`
	Checksum       string    `json:"checksum,omitempty"`
	ExportedAt     time.Time `json:"exportedAt"`
}