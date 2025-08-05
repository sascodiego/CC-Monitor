/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-EXPORT-001
 * CONTEXT:   CLI command configuration structures for comprehensive export functionality
 * REASON:    Need well-defined command configuration structures for all export operations
 * CHANGE:    Export command configuration types with comprehensive parameter support.
 * PREVENTION:Validate all configuration fields, provide sensible defaults, ensure type safety
 * RISK:      Low - Configuration structures are data containers but need proper validation
 */
package cli

import (
	"fmt"
	"time"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-EXPORT-002
 * CONTEXT:   Main export command configuration with comprehensive parameter support
 * REASON:    Export commands need flexible configuration to support all use cases and formats
 * CHANGE:    Main export command configuration with all necessary parameters and options.
 * PREVENTION:Ensure all required fields are validated, provide clear field documentation, handle optional parameters properly
 * RISK:      Medium - Configuration structure affects all export operations and needs to be comprehensive
 */

// ExportCommandConfig defines configuration for export commands
type ExportCommandConfig struct {
	// Basic parameters
	ReportType    string `json:"reportType"`    // daily, weekly, monthly, timesheet, analytics, custom
	Format        string `json:"format"`        // json, csv, pdf, html, excel
	StartDate     string `json:"startDate"`     // YYYY-MM-DD format
	EndDate       string `json:"endDate"`       // YYYY-MM-DD format
	
	// Output configuration
	OutputPath    string `json:"outputPath"`    // Full path to output file
	Filename      string `json:"filename"`      // Filename only (used with default output directory)
	Template      string `json:"template"`      // Template name for formatting
	
	// Content options
	IncludeCharts    bool `json:"includeCharts"`    // Include visual charts
	IncludeBreakdown bool `json:"includeBreakdown"` // Include detailed breakdowns
	IncludePatterns  bool `json:"includePatterns"`  // Include pattern analysis
	IncludeTrends    bool `json:"includeTrends"`    // Include trend analysis
	IncludeRawData   bool `json:"includeRawData"`   // Include raw session/block data
	
	// Filtering options
	EmployeeFilter []string `json:"employeeFilter"` // Filter by employee IDs
	ProjectFilter  []string `json:"projectFilter"`  // Filter by project names
	
	// Formatting options
	Timezone       string `json:"timezone"`       // Timezone for date/time formatting
	DateFormat     string `json:"dateFormat"`     // Date format string
	TimeFormat     string `json:"timeFormat"`     // Time format string
	CurrencyFormat string `json:"currencyFormat"` // Currency format for billing
	
	// Advanced options
	CompressOutput   bool                   `json:"compressOutput"`   // Compress output files
	DigitalSignature bool                   `json:"digitalSignature"` // Add digital signature (PDF)
	Watermark        string                 `json:"watermark"`        // Watermark text
	CustomFields     map[string]interface{} `json:"customFields"`     // Custom fields to include
	
	// Processing options
	BatchSize     int  `json:"batchSize"`     // Batch size for large datasets
	Async         bool `json:"async"`         // Process asynchronously
	MaxConcurrency int  `json:"maxConcurrency"` // Maximum concurrent operations
	
	// Metadata
	RequestedBy string `json:"requestedBy"` // Who requested the export
	Description string `json:"description"` // Export description
	Tags        []string `json:"tags"`      // Tags for organization
	
	// CLI options
	Verbose bool `json:"verbose"` // Verbose output
	Quiet   bool `json:"quiet"`   // Suppress non-error output
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-EXPORT-003
 * CONTEXT:   Batch export configuration for processing multiple exports efficiently
 * REASON:    Users need ability to process multiple export requests in batch for efficiency
 * CHANGE:    Batch export configuration with concurrency control and progress tracking.
 * PREVENTION:Control resource usage with concurrency limits, handle timeouts properly, provide progress feedback
 * RISK:      High - Batch operations can consume significant resources and need careful management
 */

// BatchExportConfig defines configuration for batch export operations
type BatchExportConfig struct {
	// Export requests
	ExportRequests []*ExportCommandConfig `json:"exportRequests"` // List of export requests
	
	// Processing options
	MaxConcurrency  int           `json:"maxConcurrency"`  // Maximum concurrent exports
	TimeoutMinutes  int           `json:"timeoutMinutes"`  // Overall timeout in minutes
	RetryAttempts   int           `json:"retryAttempts"`   // Number of retry attempts for failed exports
	RetryDelay      time.Duration `json:"retryDelay"`      // Delay between retry attempts
	
	// Output options
	OutputDirectory string `json:"outputDirectory"` // Base directory for all outputs
	CreateManifest  bool   `json:"createManifest"`  // Create manifest file listing all exports
	CompressAll     bool   `json:"compressAll"`     // Compress all outputs into single archive
	
	// Progress and feedback
	ShowProgress    bool   `json:"showProgress"`    // Show progress indicator
	ProgressFormat  string `json:"progressFormat"`  // Progress format (bar, percentage, detailed)
	EmailReport     bool   `json:"emailReport"`     // Email completion report
	EmailRecipients []string `json:"emailRecipients"` // Email recipients
	
	// Error handling
	StopOnError     bool `json:"stopOnError"`     // Stop batch on first error
	ContinueOnError bool `json:"continueOnError"` // Continue processing despite errors
	SavePartial     bool `json:"savePartial"`     // Save partial results on failure
	
	// Metadata
	BatchName       string    `json:"batchName"`       // Name for this batch operation
	BatchDescription string   `json:"batchDescription"` // Description of batch operation
	RequestedBy     string    `json:"requestedBy"`     // Who requested the batch
	RequestedAt     time.Time `json:"requestedAt"`     // When batch was requested
	
	// CLI options
	Verbose bool `json:"verbose"` // Verbose output
	Quiet   bool `json:"quiet"`   // Suppress non-error output
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-EXPORT-004
 * CONTEXT:   Template management command configuration for customizing export appearance
 * REASON:    Users need comprehensive template management for branding and customization
 * CHANGE:    Template management configuration with all CRUD operations and validation.
 * PREVENTION:Validate template syntax and dependencies, backup existing templates, handle version conflicts
 * RISK:      Medium - Template management affects export appearance and needs proper validation
 */

// TemplateCommandConfig defines configuration for template management commands
type TemplateCommandConfig struct {
	// Action to perform
	Action string `json:"action"` // list, create, update, delete, validate, export, import
	
	// Template identification
	TemplateName    string `json:"templateName"`    // Name of the template
	TemplateType    string `json:"templateType"`    // Type: html, pdf, excel, etc.
	TemplateVersion string `json:"templateVersion"` // Template version
	
	// Template content
	TemplateFile    string   `json:"templateFile"`    // Path to template file
	TemplateContent string   `json:"templateContent"` // Template content (for inline creation)
	TemplateAssets  []string `json:"templateAssets"`  // Associated asset files
	
	// Template metadata
	Description     string            `json:"description"`     // Template description
	Author          string            `json:"author"`          // Template author
	Tags            []string          `json:"tags"`            // Template tags
	Variables       map[string]string `json:"variables"`       // Template variables and their descriptions
	
	// Template settings
	DefaultVariables map[string]interface{} `json:"defaultVariables"` // Default variable values
	RequiredAssets   []string               `json:"requiredAssets"`   // Required asset files
	Permissions      []string               `json:"permissions"`      // Template permissions
	
	// Export/Import options
	ExportPath      string   `json:"exportPath"`      // Path for template export
	ImportPath      string   `json:"importPath"`      // Path for template import
	IncludeAssets   bool     `json:"includeAssets"`   // Include assets in export/import
	OverwriteExisting bool   `json:"overwriteExisting"` // Overwrite existing templates
	
	// Validation options
	ValidateOnly    bool     `json:"validateOnly"`    // Only validate, don't save
	StrictValidation bool    `json:"strictValidation"` // Use strict validation rules
	CheckDependencies bool   `json:"checkDependencies"` // Check template dependencies
	
	// CLI options
	OutputFormat    string   `json:"outputFormat"`    // Output format for list/show commands
	Verbose         bool     `json:"verbose"`         // Verbose output
	Quiet           bool     `json:"quiet"`           // Suppress non-error output
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-EXPORT-005
 * CONTEXT:   Export status and monitoring command configuration for tracking operations
 * REASON:    Users need comprehensive monitoring of export operations with detailed status information
 * CHANGE:    Export status command configuration with monitoring and history capabilities.
 * PREVENTION:Handle status queries efficiently, validate operation IDs, provide meaningful status information
 * RISK:      Low - Status commands are read-only but need proper parameter validation
 */

// ExportStatusConfig defines configuration for export status commands
type ExportStatusConfig struct {
	// Action to perform
	Action string `json:"action"` // list, show, cancel, history, stats
	
	// Export identification
	ExportID    string `json:"exportId"`    // Specific export ID to query
	BatchID     string `json:"batchId"`     // Batch operation ID
	RequestedBy string `json:"requestedBy"` // Filter by who requested
	
	// Filtering options
	Status      []string  `json:"status"`      // Filter by status (pending, processing, completed, failed)
	DateFrom    time.Time `json:"dateFrom"`    // Filter from date
	DateTo      time.Time `json:"dateTo"`      // Filter to date
	ReportType  []string  `json:"reportType"`  // Filter by report type
	Format      []string  `json:"format"`      // Filter by export format
	
	// Display options
	ShowDetails    bool     `json:"showDetails"`    // Show detailed information
	ShowProgress   bool     `json:"showProgress"`   // Show progress information
	ShowLogs       bool     `json:"showLogs"`       // Show operation logs
	ShowStats      bool     `json:"showStats"`      // Show statistics
	GroupBy        string   `json:"groupBy"`        // Group results by field
	SortBy         string   `json:"sortBy"`         // Sort results by field
	SortDirection  string   `json:"sortDirection"`  // Sort direction (asc, desc)
	
	// Pagination
	Limit          int      `json:"limit"`          // Limit number of results
	Offset         int      `json:"offset"`         // Offset for pagination
	Page           int      `json:"page"`           // Page number
	PageSize       int      `json:"pageSize"`       // Results per page
	
	// History options
	HistoryDays    int      `json:"historyDays"`    // Number of days of history
	IncludeErrors  bool     `json:"includeErrors"`  // Include failed operations
	IncludeWarnings bool    `json:"includeWarnings"` // Include operations with warnings
	
	// Cancellation options
	Force          bool     `json:"force"`          // Force cancellation
	Reason         string   `json:"reason"`         // Cancellation reason
	
	// CLI options
	OutputFormat   string   `json:"outputFormat"`   // Output format (table, json, csv)
	Watch          bool     `json:"watch"`          // Watch mode for real-time updates
	WatchInterval  time.Duration `json:"watchInterval"` // Watch update interval
	Verbose        bool     `json:"verbose"`        // Verbose output
	Quiet          bool     `json:"quiet"`          // Suppress non-error output
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-EXPORT-006
 * CONTEXT:   Export schedule configuration for automated export operations
 * REASON:    Users need ability to schedule regular exports for automated reporting
 * CHANGE:    Export schedule configuration with cron-like scheduling and automation features.
 * PREVENTION:Validate schedule expressions, handle timezone considerations, ensure proper error handling for scheduled operations
 * RISK:      Medium - Scheduled operations need careful error handling and resource management
 */

// ExportScheduleConfig defines configuration for scheduled export operations
type ExportScheduleConfig struct {
	// Schedule identification
	ScheduleName    string `json:"scheduleName"`    // Name for this schedule
	ScheduleID      string `json:"scheduleId"`      // Unique schedule identifier
	Description     string `json:"description"`     // Schedule description
	
	// Schedule timing
	CronExpression  string        `json:"cronExpression"`  // Cron expression for scheduling
	Timezone        string        `json:"timezone"`        // Timezone for schedule
	StartDate       time.Time     `json:"startDate"`       // When to start the schedule
	EndDate         *time.Time    `json:"endDate"`         // When to end the schedule (optional)
	MaxRuns         int           `json:"maxRuns"`         // Maximum number of runs (optional)
	
	// Export configuration
	ExportTemplate  *ExportCommandConfig `json:"exportTemplate"` // Template for export configuration
	DynamicDates    bool                 `json:"dynamicDates"`   // Use dynamic date ranges (e.g., "last week")
	DatePattern     string               `json:"datePattern"`    // Pattern for dynamic dates
	
	// Output configuration
	OutputPattern   string `json:"outputPattern"`   // Pattern for output filenames (with date/time variables)
	OutputDirectory string `json:"outputDirectory"` // Base directory for scheduled exports
	ArchiveOld      bool   `json:"archiveOld"`      // Archive old exports
	CleanupAfter    int    `json:"cleanupAfter"`    // Days after which to cleanup old exports
	
	// Notification configuration
	NotifyOnSuccess bool     `json:"notifyOnSuccess"` // Send notification on successful export
	NotifyOnFailure bool     `json:"notifyOnFailure"` // Send notification on export failure
	EmailRecipients []string `json:"emailRecipients"` // Email recipients for notifications
	SlackWebhook    string   `json:"slackWebhook"`    // Slack webhook for notifications
	
	// Error handling
	RetryOnFailure  bool          `json:"retryOnFailure"`  // Retry failed exports
	MaxRetries      int           `json:"maxRetries"`      // Maximum retry attempts
	RetryDelay      time.Duration `json:"retryDelay"`      // Delay between retries
	
	// Metadata
	CreatedBy       string    `json:"createdBy"`       // Who created the schedule
	CreatedAt       time.Time `json:"createdAt"`       // When schedule was created
	LastRun         *time.Time `json:"lastRun"`        // Last execution time
	NextRun         *time.Time `json:"nextRun"`        // Next scheduled execution
	RunCount        int       `json:"runCount"`        // Number of times executed
	
	// Status
	Enabled         bool   `json:"enabled"`         // Whether schedule is enabled
	Status          string `json:"status"`          // Current status (active, paused, error)
	LastError       string `json:"lastError"`       // Last error message
	
	// CLI options
	Verbose         bool   `json:"verbose"`         // Verbose output
	Quiet           bool   `json:"quiet"`           // Suppress non-error output
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-EXPORT-007
 * CONTEXT:   Export validation configuration for ensuring data quality and consistency
 * REASON:    Users need ability to validate export configurations and data before processing
 * CHANGE:    Export validation configuration with comprehensive validation options.
 * PREVENTION:Implement thorough validation rules, provide clear validation messages, handle edge cases
 * RISK:      Low - Validation improves data quality but doesn't affect core functionality
 */

// ExportValidationConfig defines configuration for export validation
type ExportValidationConfig struct {
	// Validation scope
	ValidationType  string   `json:"validationType"`  // config, data, template, output
	ValidationRules []string `json:"validationRules"` // Specific validation rules to apply
	
	// Configuration validation
	ConfigFile      string `json:"configFile"`      // Configuration file to validate
	ConfigData      string `json:"configData"`      // Configuration data to validate
	StrictMode      bool   `json:"strictMode"`      // Use strict validation rules
	
	// Data validation
	StartDate       string `json:"startDate"`       // Start date for data validation
	EndDate         string `json:"endDate"`         // End date for data validation
	SampleSize      int    `json:"sampleSize"`      // Sample size for data validation
	CheckIntegrity  bool   `json:"checkIntegrity"`  // Check data integrity
	CheckConsistency bool  `json:"checkConsistency"` // Check data consistency
	
	// Template validation
	TemplateName    string `json:"templateName"`    // Template to validate
	TemplateFile    string `json:"templateFile"`    // Template file to validate
	CheckSyntax     bool   `json:"checkSyntax"`     // Check template syntax
	CheckVariables  bool   `json:"checkVariables"`  // Check template variables
	CheckAssets     bool   `json:"checkAssets"`     // Check template assets
	
	// Output validation
	OutputFile      string `json:"outputFile"`      // Output file to validate
	CheckFormat     bool   `json:"checkFormat"`     // Check output format
	CheckContent    bool   `json:"checkContent"`    // Check output content
	CompareBaseline string `json:"compareBaseline"` // Baseline file for comparison
	
	// Validation options
	FailFast        bool   `json:"failFast"`        // Stop on first validation error
	ShowWarnings    bool   `json:"showWarnings"`    // Show validation warnings
	GenerateReport  bool   `json:"generateReport"`  // Generate validation report
	ReportFormat    string `json:"reportFormat"`    // Validation report format
	
	// CLI options
	Verbose         bool   `json:"verbose"`         // Verbose output
	Quiet           bool   `json:"quiet"`           // Suppress non-error output
	OutputPath      string `json:"outputPath"`      // Path for validation output
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-EXPORT-008
 * CONTEXT:   Helper functions and utilities for export command configurations
 * REASON:    Need utility functions to support command configuration operations and validation
 * CHANGE:    Configuration utility functions with validation and default value support.
 * PREVENTION:Validate all configuration parameters, provide sensible defaults, handle edge cases
 * RISK:      Low - Utility functions support configuration but need proper validation
 */

// GetDefaultExportConfig returns default export command configuration
func GetDefaultExportConfig() *ExportCommandConfig {
	return &ExportCommandConfig{
		ReportType:       "daily",
		Format:           "json",
		IncludeCharts:    false,
		IncludeBreakdown: false,
		IncludePatterns:  false,
		IncludeTrends:    false,
		IncludeRawData:   false,
		Timezone:         "Local",
		DateFormat:       "2006-01-02",
		TimeFormat:       "15:04:05",
		CompressOutput:   false,
		DigitalSignature: false,
		BatchSize:        1000,
		Async:            false,
		MaxConcurrency:   4,
		CustomFields:     make(map[string]interface{}),
		Verbose:          false,
		Quiet:            false,
	}
}

// GetDefaultBatchExportConfig returns default batch export configuration
func GetDefaultBatchExportConfig() *BatchExportConfig {
	return &BatchExportConfig{
		ExportRequests:   make([]*ExportCommandConfig, 0),
		MaxConcurrency:   2,
		TimeoutMinutes:   60,
		RetryAttempts:    3,
		RetryDelay:       5 * time.Second,
		CreateManifest:   true,
		CompressAll:      false,
		ShowProgress:     true,
		ProgressFormat:   "bar",
		StopOnError:      false,
		ContinueOnError:  true,
		SavePartial:      true,
		RequestedAt:      time.Now(),
		Verbose:          false,
		Quiet:            false,
	}
}

// GetDefaultTemplateConfig returns default template command configuration
func GetDefaultTemplateConfig() *TemplateCommandConfig {
	return &TemplateCommandConfig{
		TemplateType:      "html",
		Variables:         make(map[string]string),
		DefaultVariables:  make(map[string]interface{}),
		IncludeAssets:     true,
		OverwriteExisting: false,
		ValidateOnly:      false,
		StrictValidation:  false,
		CheckDependencies: true,
		OutputFormat:      "table",
		Verbose:           false,
		Quiet:             false,
	}
}

// ValidateExportConfig validates export command configuration
func ValidateExportConfig(config *ExportCommandConfig) error {
	if config == nil {
		return fmt.Errorf("export configuration is nil")
	}
	
	// Validate report type
	validReportTypes := []string{"daily", "weekly", "monthly", "timesheet", "analytics", "custom"}
	if !contains(validReportTypes, config.ReportType) {
		return fmt.Errorf("invalid report type: %s", config.ReportType)
	}
	
	// Validate format
	validFormats := []string{"json", "csv", "pdf", "html", "excel"}
	if !contains(validFormats, config.Format) {
		return fmt.Errorf("invalid format: %s", config.Format)
	}
	
	// Validate date formats if provided
	if config.StartDate != "" {
		if _, err := time.Parse("2006-01-02", config.StartDate); err != nil {
			return fmt.Errorf("invalid start date format: %s", config.StartDate)
		}
	}
	
	if config.EndDate != "" {
		if _, err := time.Parse("2006-01-02", config.EndDate); err != nil {
			return fmt.Errorf("invalid end date format: %s", config.EndDate)
		}
	}
	
	// Validate timezone if provided
	if config.Timezone != "" && config.Timezone != "Local" {
		if _, err := time.LoadLocation(config.Timezone); err != nil {
			return fmt.Errorf("invalid timezone: %s", config.Timezone)
		}
	}
	
	// Validate concurrency settings
	if config.MaxConcurrency < 1 {
		return fmt.Errorf("max concurrency must be at least 1")
	}
	
	if config.BatchSize < 1 {
		return fmt.Errorf("batch size must be at least 1")
	}
	
	return nil
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// MergeExportConfigs merges two export configurations with the second taking precedence
func MergeExportConfigs(base, override *ExportCommandConfig) *ExportCommandConfig {
	if base == nil {
		base = GetDefaultExportConfig()
	}
	
	if override == nil {
		return base
	}
	
	merged := *base // Copy base configuration
	
	// Override non-empty values
	if override.ReportType != "" {
		merged.ReportType = override.ReportType
	}
	if override.Format != "" {
		merged.Format = override.Format
	}
	if override.StartDate != "" {
		merged.StartDate = override.StartDate
	}
	if override.EndDate != "" {
		merged.EndDate = override.EndDate
	}
	if override.OutputPath != "" {
		merged.OutputPath = override.OutputPath
	}
	if override.Filename != "" {
		merged.Filename = override.Filename
	}
	if override.Template != "" {
		merged.Template = override.Template
	}
	
	// Override boolean values (always take from override)
	merged.IncludeCharts = override.IncludeCharts
	merged.IncludeBreakdown = override.IncludeBreakdown
	merged.IncludePatterns = override.IncludePatterns
	merged.IncludeTrends = override.IncludeTrends
	merged.IncludeRawData = override.IncludeRawData
	merged.CompressOutput = override.CompressOutput
	merged.DigitalSignature = override.DigitalSignature
	merged.Async = override.Async
	merged.Verbose = override.Verbose
	merged.Quiet = override.Quiet
	
	// Override slices and maps if provided
	if len(override.EmployeeFilter) > 0 {
		merged.EmployeeFilter = override.EmployeeFilter
	}
	if len(override.ProjectFilter) > 0 {
		merged.ProjectFilter = override.ProjectFilter
	}
	if len(override.CustomFields) > 0 {
		merged.CustomFields = override.CustomFields
	}
	if len(override.Tags) > 0 {
		merged.Tags = override.Tags
	}
	
	// Override numeric values if non-zero
	if override.BatchSize > 0 {
		merged.BatchSize = override.BatchSize
	}
	if override.MaxConcurrency > 0 {
		merged.MaxConcurrency = override.MaxConcurrency
	}
	
	return &merged
}