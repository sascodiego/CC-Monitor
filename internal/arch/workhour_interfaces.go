package arch

import (
	"time"

	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-007
 * CONTEXT:   Work hour analytics interfaces for reporting and business intelligence
 * REASON:    Need clean abstractions for work hour analysis, reporting, and export capabilities
 * CHANGE:    New interfaces for work hour analytics system extending existing architecture.
 * PREVENTION:Keep interfaces focused, validate date ranges, handle timezone conversions properly
 * RISK:      Medium - Complex analytics could impact performance if not properly cached and optimized
 */

// WorkHourAnalyzer defines the contract for work hour analysis and reporting
type WorkHourAnalyzer interface {
	// AnalyzeWorkDay generates a work day analysis from raw session/work block data
	AnalyzeWorkDay(date time.Time) (*domain.WorkDay, error)
	
	// AnalyzeWorkWeek generates a work week analysis for the week containing the given date
	AnalyzeWorkWeek(date time.Time) (*domain.WorkWeek, error)
	
	// AnalyzeWorkPattern identifies work patterns from historical data
	AnalyzeWorkPattern(startDate, endDate time.Time) (*domain.WorkPattern, error)
	
	// GenerateActivitySummary creates comprehensive activity summary for period
	GenerateActivitySummary(period string, startDate, endDate time.Time) (*domain.ActivitySummary, error)
	
	// CalculateProductivityMetrics computes productivity and efficiency metrics
	CalculateProductivityMetrics(startDate, endDate time.Time) (*domain.EfficiencyMetrics, error)
	
	// GetWorkDayTrends analyzes trends in work patterns over time
	GetWorkDayTrends(startDate, endDate time.Time) (*domain.TrendAnalysis, error)
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-008
 * CONTEXT:   Timesheet management interface for formal time tracking and export
 * REASON:    Business requirement for professional timesheet generation with configurable policies
 * CHANGE:    New interface for timesheet management with business rule support.
 * PREVENTION:Validate timesheet policies and ensure rounding rules are correctly applied
 * RISK:      High - Timesheet accuracy is critical for billing and compliance requirements
 */

// TimesheetManager defines the contract for timesheet generation and management
type TimesheetManager interface {
	// GenerateTimesheet creates a formal timesheet for the specified period
	GenerateTimesheet(employeeID string, period domain.TimesheetPeriod, startDate time.Time, policy domain.TimesheetPolicy) (*domain.Timesheet, error)
	
	// ApplyTimesheetPolicy applies rounding and overtime rules to timesheet entries
	ApplyTimesheetPolicy(timesheet *domain.Timesheet) error
	
	// ValidateTimesheet checks timesheet for consistency and business rule compliance
	ValidateTimesheet(timesheet *domain.Timesheet) error
	
	// SaveTimesheet persists a timesheet to storage
	SaveTimesheet(timesheet *domain.Timesheet) error
	
	// GetTimesheet retrieves a previously saved timesheet
	GetTimesheet(timesheetID string) (*domain.Timesheet, error)
	
	// GetTimesheetsByPeriod retrieves all timesheets for an employee in a period
	GetTimesheetsByPeriod(employeeID string, startDate, endDate time.Time) ([]*domain.Timesheet, error)
	
	// SubmitTimesheet changes timesheet status to submitted
	SubmitTimesheet(timesheetID string) error
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-009
 * CONTEXT:   Report generation interface for comprehensive work hour reporting
 * REASON:    Need flexible report generation with multiple formats and customizable content
 * CHANGE:    New interface for report generation engine with export capabilities.
 * PREVENTION:Validate report parameters and handle large datasets efficiently
 * RISK:      Medium - Large report generation could impact system performance
 */

// WorkHourReportGenerator defines the contract for generating work hour reports
type WorkHourReportGenerator interface {
	// GenerateDailyReport creates a detailed daily work report
	GenerateDailyReport(date time.Time, config ReportConfig) (*WorkHourReport, error)
	
	// GenerateWeeklyReport creates a comprehensive weekly work report
	GenerateWeeklyReport(weekStart time.Time, config ReportConfig) (*WorkHourReport, error)
	
	// GenerateMonthlyReport creates a monthly work summary report
	GenerateMonthlyReport(year int, month time.Month, config ReportConfig) (*WorkHourReport, error)
	
	// GenerateCustomReport creates a report for custom date range
	GenerateCustomReport(startDate, endDate time.Time, config ReportConfig) (*WorkHourReport, error)
	
	// GenerateTimesheetReport creates a formal timesheet report
	GenerateTimesheetReport(timesheetID string, config ReportConfig) (*WorkHourReport, error)
	
	// GenerateProductivityReport creates a productivity analysis report
	GenerateProductivityReport(startDate, endDate time.Time, config ReportConfig) (*WorkHourReport, error)
	
	// GenerateTrendReport creates a trend analysis report
	GenerateTrendReport(startDate, endDate time.Time, config ReportConfig) (*WorkHourReport, error)
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-010
 * CONTEXT:   Export interface for multiple output formats and data integration
 * REASON:    Need flexible export capabilities for integration with external systems
 * CHANGE:    New interface for export engine with multiple format support.
 * PREVENTION:Validate export formats and handle file I/O errors gracefully
 * RISK:      Low - Export failures don't affect core monitoring functionality
 */

// WorkHourExporter defines the contract for exporting work hour data
type WorkHourExporter interface {
	// ExportToJSON exports report data in JSON format
	ExportToJSON(report *WorkHourReport, outputPath string) error
	
	// ExportToCSV exports report data in CSV format for spreadsheet applications
	ExportToCSV(report *WorkHourReport, outputPath string) error
	
	// ExportToPDF generates a formatted PDF report
	ExportToPDF(report *WorkHourReport, outputPath string) error
	
	// ExportToHTML generates an HTML report with interactive charts
	ExportToHTML(report *WorkHourReport, outputPath string) error
	
	// ExportToExcel exports to Excel format with multiple worksheets
	ExportToExcel(report *WorkHourReport, outputPath string) error
	
	// ExportRawData exports raw session/work block data for external analysis
	ExportRawData(startDate, endDate time.Time, format ExportFormat, outputPath string) error
	
	// ExportTimesheet exports timesheet in standard formats (PDF, Excel)
	ExportTimesheet(timesheet *domain.Timesheet, format ExportFormat, outputPath string) error
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-011
 * CONTEXT:   Work hour database interface extending existing database operations
 * REASON:    Need optimized queries for work hour analytics while maintaining data consistency
 * CHANGE:    Extension of DatabaseManager interface for work hour specific operations.
 * PREVENTION:Use proper indexes, cache frequently accessed aggregations, validate query parameters
 * RISK:      High - Poor query performance could impact overall system responsiveness
 */

// WorkHourDatabaseManager extends DatabaseManager with work hour specific operations
type WorkHourDatabaseManager interface {
	DatabaseManager // Embed existing database interface
	
	// Aggregation Queries
	GetWorkDayData(date time.Time) (*domain.WorkDay, error)
	GetWorkWeekData(weekStart time.Time) (*domain.WorkWeek, error)
	GetWorkMonthData(year int, month time.Month) ([]*domain.WorkDay, error)
	
	// Pattern Analysis Queries
	GetWorkPatternData(startDate, endDate time.Time) ([]WorkPatternDataPoint, error)
	GetProductivityMetrics(startDate, endDate time.Time) (*domain.EfficiencyMetrics, error)
	GetBreakPatterns(startDate, endDate time.Time) ([]domain.BreakPattern, error)
	
	// Trend Analysis Queries
	GetWorkTimeTrends(startDate, endDate time.Time, granularity Granularity) ([]TrendDataPoint, error)
	GetSessionCountTrends(startDate, endDate time.Time, granularity Granularity) ([]TrendDataPoint, error)
	GetEfficiencyTrends(startDate, endDate time.Time, granularity Granularity) ([]TrendDataPoint, error)
	
	// Timesheet Operations
	SaveWorkDay(workDay *domain.WorkDay) error
	SaveTimesheet(timesheet *domain.Timesheet) error
	GetTimesheetData(startDate, endDate time.Time) ([]*domain.TimesheetEntry, error)
	
	// Performance Optimizations
	RefreshWorkDayCache(date time.Time) error
	GetCachedWorkDayStats(date time.Time) (*domain.WorkDay, bool)
	InvalidateWorkHourCache(startDate, endDate time.Time) error
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-012
 * CONTEXT:   Configuration and data structures for work hour system
 * REASON:    Need well-defined configuration options and data transfer objects
 * CHANGE:    New configuration structures for work hour reporting system.
 * PREVENTION:Validate all configuration values and provide sensible defaults
 * RISK:      Low - Configuration errors are typically caught at initialization
 */

// ReportConfig defines configuration options for report generation
type ReportConfig struct {
	IncludeCharts     bool          `json:"includeCharts"`     // Include visual charts in reports
	IncludeBreakdown  bool          `json:"includeBreakdown"`  // Include detailed breakdowns
	IncludePatterns   bool          `json:"includePatterns"`   // Include pattern analysis
	IncludeTrends     bool          `json:"includeTrends"`     // Include trend analysis
	Timezone          string        `json:"timezone"`          // Report timezone
	Format            ReportFormat  `json:"format"`            // Output format
	Template          string        `json:"template"`          // Report template
	CustomFields      []string      `json:"customFields"`      // Additional fields to include
	GroupBy           []GroupByField `json:"groupBy"`          // Grouping options
	SortBy            SortField     `json:"sortBy"`            // Sorting options
	FilterCriteria    FilterCriteria `json:"filterCriteria"`   // Data filtering
}

// ReportFormat defines available report output formats
type ReportFormat string

const (
	ReportFormatJSON  ReportFormat = "json"
	ReportFormatCSV   ReportFormat = "csv"
	ReportFormatPDF   ReportFormat = "pdf"
	ReportFormatHTML  ReportFormat = "html"
	ReportFormatExcel ReportFormat = "excel"
	ReportFormatText  ReportFormat = "text"
)

// ExportFormat defines available export formats
type ExportFormat string

const (
	ExportJSON  ExportFormat = "json"
	ExportCSV   ExportFormat = "csv"
	ExportPDF   ExportFormat = "pdf"
	ExportExcel ExportFormat = "excel"
	ExportXML   ExportFormat = "xml"
)

// GroupByField defines available grouping options for reports
type GroupByField string

const (
	GroupByDay     GroupByField = "day"
	GroupByWeek    GroupByField = "week"
	GroupByMonth   GroupByField = "month"
	GroupBySession GroupByField = "session"
	GroupByHour    GroupByField = "hour"
)

// SortField defines available sorting options
type SortField string

const (
	SortByDate     SortField = "date"
	SortByDuration SortField = "duration"
	SortByActivity SortField = "activity"
	SortByName     SortField = "name"
)

// FilterCriteria defines data filtering options
type FilterCriteria struct {
	MinDuration    time.Duration `json:"minDuration"`    // Minimum work block duration
	MaxDuration    time.Duration `json:"maxDuration"`    // Maximum work block duration
	MinSessionTime time.Duration `json:"minSessionTime"` // Minimum session duration
	ExcludeWeekends bool         `json:"excludeWeekends"` // Exclude weekend data
	ExcludeHolidays bool         `json:"excludeHolidays"` // Exclude holiday data
	TimeOfDay      TimeRange     `json:"timeOfDay"`      // Filter by time of day
}

// TimeRange defines a time of day range
type TimeRange struct {
	StartHour int `json:"startHour"` // 0-23
	EndHour   int `json:"endHour"`   // 0-23
}

// Granularity defines the time granularity for trend analysis
type Granularity string

const (
	GranularityHour  Granularity = "hour"
	GranularityDay   Granularity = "day"
	GranularityWeek  Granularity = "week"
	GranularityMonth Granularity = "month"
)

// WorkHourReport represents a generated work hour report
type WorkHourReport struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	ReportType   string                 `json:"reportType"`
	Period       string                 `json:"period"`
	StartDate    time.Time              `json:"startDate"`
	EndDate      time.Time              `json:"endDate"`
	GeneratedAt  time.Time              `json:"generatedAt"`
	Summary      *domain.ActivitySummary `json:"summary"`
	WorkDays     []*domain.WorkDay      `json:"workDays,omitempty"`
	WorkWeeks    []*domain.WorkWeek     `json:"workWeeks,omitempty"`
	Patterns     *domain.WorkPattern    `json:"patterns,omitempty"`
	Trends       *domain.TrendAnalysis  `json:"trends,omitempty"`
	Charts       []ChartData            `json:"charts,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ChartData represents chart information for visual reports
type ChartData struct {
	Type     string      `json:"type"`     // "line", "bar", "pie", etc.
	Title    string      `json:"title"`
	Data     interface{} `json:"data"`     // Chart-specific data structure
	Options  interface{} `json:"options"`  // Chart-specific options
}

// WorkPatternDataPoint represents a single data point for pattern analysis
type WorkPatternDataPoint struct {
	Timestamp   time.Time     `json:"timestamp"`
	Duration    time.Duration `json:"duration"`
	ActivityType string       `json:"activityType"`
	Intensity   float64       `json:"intensity"`    // 0-1 activity intensity
	Productivity float64      `json:"productivity"` // 0-1 productivity score
}

// TrendDataPoint represents a single data point for trend analysis
type TrendDataPoint struct {
	Date     time.Time `json:"date"`
	Value    float64   `json:"value"`
	Baseline float64   `json:"baseline"` // Moving average or baseline value
	Change   float64   `json:"change"`   // Percentage change from previous
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-013
 * CONTEXT:   Work hour service coordinator interface for high-level operations
 * REASON:    Need coordinated service that orchestrates analyzer, reporter, and exporter components
 * CHANGE:    New coordinator interface for work hour services integration.
 * PREVENTION:Handle service failures gracefully and maintain service boundaries
 * RISK:      Medium - Service coordination failures could affect reporting functionality
 */

// WorkHourService defines the high-level contract for work hour operations
type WorkHourService interface {
	// Core Analysis Operations
	GetDailyWorkSummary(date time.Time) (*domain.WorkDay, error)
	GetWeeklyWorkSummary(weekStart time.Time) (*domain.WorkWeek, error)
	GetMonthlyWorkSummary(year int, month time.Month) ([]*domain.WorkDay, error)
	
	// Report Generation
	GenerateReport(reportType string, startDate, endDate time.Time, config ReportConfig) (*WorkHourReport, error)
	ExportReport(report *WorkHourReport, format ExportFormat, outputPath string) error
	
	// Timesheet Operations
	CreateTimesheet(employeeID string, period domain.TimesheetPeriod, startDate time.Time) (*domain.Timesheet, error)
	FinalizeTimesheet(timesheetID string) error
	
	// Analytics and Insights
	GetProductivityInsights(startDate, endDate time.Time) (*domain.EfficiencyMetrics, error)
	GetWorkPatternAnalysis(startDate, endDate time.Time) (*domain.WorkPattern, error)
	GetTrendAnalysis(startDate, endDate time.Time) (*domain.TrendAnalysis, error)
	
	// Configuration and Management
	UpdateWorkHourPolicy(policy domain.TimesheetPolicy) error
	GetWorkHourConfiguration() (*WorkHourConfiguration, error)
	RefreshCache() error
}

// WorkHourConfiguration defines configuration for the work hour system
type WorkHourConfiguration struct {
	DefaultPolicy       domain.TimesheetPolicy `json:"defaultPolicy"`
	ReportTimezone      string                 `json:"reportTimezone"`
	WorkWeekStart       time.Weekday           `json:"workWeekStart"`
	StandardWorkHours   time.Duration          `json:"standardWorkHours"`
	OvertimeThreshold   time.Duration          `json:"overtimeThreshold"`
	BreakDeduction      time.Duration          `json:"breakDeduction"`
	CacheRefreshInterval time.Duration         `json:"cacheRefreshInterval"`
	EnableTrendAnalysis bool                   `json:"enableTrendAnalysis"`
	EnablePatternAnalysis bool                 `json:"enablePatternAnalysis"`
}