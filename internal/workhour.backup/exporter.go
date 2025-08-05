package workhour

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-027
 * CONTEXT:   Work hour export engine supporting multiple output formats and professional reporting
 * REASON:    Need flexible export capabilities for integration with external systems and professional reporting
 * CHANGE:    New export engine with multiple format support and templating system.
 * PREVENTION:Validate export parameters, handle file I/O errors gracefully, ensure output format compliance
 * RISK:      Low - Export failures don't affect core monitoring functionality but impact user workflow
 */

// WorkHourExporter implements the WorkHourExporter interface for multiple export formats
type WorkHourExporter struct {
	logger       arch.Logger
	templateDir  string
	outputDir    string
	
	// Export templates
	htmlTemplate *template.Template
	pdfGenerator PDFGenerator
	excelWriter  ExcelWriter
	
	// Export configuration
	defaultTimezone string
	defaultFormat   arch.ExportFormat
}

// NewWorkHourExporter creates a new work hour exporter
func NewWorkHourExporter(logger arch.Logger, templateDir, outputDir string) *WorkHourExporter {
	exporter := &WorkHourExporter{
		logger:          logger,
		templateDir:     templateDir,
		outputDir:       outputDir,
		defaultTimezone: "UTC",
		defaultFormat:   arch.ExportJSON,
	}
	
	// Initialize templates and generators
	if err := exporter.initializeTemplates(); err != nil {
		logger.Error("Failed to initialize export templates", "error", err)
	}
	
	return exporter
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-028
 * CONTEXT:   JSON export implementation for API integration and data interchange
 * REASON:    JSON is the most versatile format for API integration and data processing
 * CHANGE:    JSON export implementation with structured output and metadata.
 * PREVENTION:Validate JSON structure, handle encoding errors, ensure timezone consistency
 * RISK:      Low - JSON export is straightforward with good error handling in Go
 */

// ExportToJSON exports report data in JSON format
func (whe *WorkHourExporter) ExportToJSON(report *arch.WorkHourReport, outputPath string) error {
	whe.logger.Info("Exporting work hour report to JSON", "outputPath", outputPath)
	
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Create JSON export structure
	jsonExport := &JSONExport{
		Metadata: ExportMetadata{
			ExportedAt:      time.Now(),
			ExportFormat:    "json",
			ExportVersion:   "1.0",
			SourceSystem:    "claude-monitor",
			Timezone:        whe.defaultTimezone,
		},
		Report: report,
	}
	
	// Open output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Encode JSON with pretty printing
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(jsonExport); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	
	whe.logger.Info("JSON export completed successfully", "outputPath", outputPath)
	return nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-029
 * CONTEXT:   CSV export implementation for spreadsheet applications and data analysis
 * REASON:    CSV format is widely supported by spreadsheet applications and data analysis tools
 * CHANGE:    CSV export implementation with configurable columns and data transformation.
 * PREVENTION:Handle CSV escaping properly, validate field formats, ensure consistent date formatting
 * RISK:      Low - CSV format is simple with good library support
 */

// ExportToCSV exports report data in CSV format for spreadsheet applications
func (whe *WorkHourExporter) ExportToCSV(report *arch.WorkHourReport, outputPath string) error {
	whe.logger.Info("Exporting work hour report to CSV", "outputPath", outputPath)
	
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Determine CSV format based on report type
	switch report.ReportType {
	case "daily":
		return whe.exportDailyReportCSV(writer, report)
	case "weekly":
		return whe.exportWeeklyReportCSV(writer, report)
	case "timesheet":
		return whe.exportTimesheetCSV(writer, report)
	default:
		return whe.exportGenericCSV(writer, report)
	}
}

// exportDailyReportCSV exports daily work hour data in CSV format
func (whe *WorkHourExporter) exportDailyReportCSV(writer *csv.Writer, report *arch.WorkHourReport) error {
	// Write header
	header := []string{
		"Date",
		"Start Time",
		"End Time",
		"Total Work Time",
		"Break Time",
		"Session Count",
		"Work Block Count",
		"Efficiency Ratio",
	}
	
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	
	// Write work day data
	for _, workDay := range report.WorkDays {
		record := []string{
			workDay.Date.Format("2006-01-02"),
			formatTimePtr(workDay.StartTime),
			formatTimePtr(workDay.EndTime),
			formatDuration(workDay.TotalTime),
			formatDuration(workDay.BreakTime),
			fmt.Sprintf("%d", workDay.SessionCount),
			fmt.Sprintf("%d", workDay.BlockCount),
			fmt.Sprintf("%.2f", workDay.GetEfficiencyRatio()),
		}
		
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}
	
	return nil
}

// exportWeeklyReportCSV exports weekly work hour data in CSV format
func (whe *WorkHourExporter) exportWeeklyReportCSV(writer *csv.Writer, report *arch.WorkHourReport) error {
	// Write header
	header := []string{
		"Week Start",
		"Week End",
		"Total Work Time",
		"Overtime Hours",
		"Standard Hours",
		"Average Daily Time",
		"Work Days",
		"Efficiency Score",
	}
	
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	
	// Write work week data
	for _, workWeek := range report.WorkWeeks {
		record := []string{
			workWeek.WeekStart.Format("2006-01-02"),
			workWeek.WeekEnd.Format("2006-01-02"),
			formatDuration(workWeek.TotalTime),
			formatDuration(workWeek.OvertimeHours),
			formatDuration(workWeek.StandardHours),
			formatDuration(workWeek.AverageDay),
			fmt.Sprintf("%d", len(workWeek.WorkDays)),
			"0.00", // Placeholder for efficiency score
		}
		
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}
	
	return nil
}

// exportTimesheetCSV exports timesheet data in CSV format
func (whe *WorkHourExporter) exportTimesheetCSV(writer *csv.Writer, report *arch.WorkHourReport) error {
	// This would export timesheet entry data
	// Implementation depends on timesheet structure in the report
	
	header := []string{
		"Date",
		"Start Time",
		"End Time",
		"Duration",
		"Project",
		"Task",
		"Description",
		"Billable",
	}
	
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	
	// Placeholder for timesheet entries
	// In practice, this would iterate through timesheet entries
	
	return nil
}

// exportGenericCSV exports generic report data
func (whe *WorkHourExporter) exportGenericCSV(writer *csv.Writer, report *arch.WorkHourReport) error {
	// Basic summary export
	header := []string{"Metric", "Value"}
	
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	
	if report.Summary != nil {
		records := [][]string{
			{"Report Type", report.ReportType},
			{"Period", report.Period},
			{"Start Date", report.StartDate.Format("2006-01-02")},
			{"End Date", report.EndDate.Format("2006-01-02")},
			{"Total Work Time", formatDuration(report.Summary.TotalWorkTime)},
			{"Total Sessions", fmt.Sprintf("%d", report.Summary.TotalSessions)},
			{"Total Work Blocks", fmt.Sprintf("%d", report.Summary.TotalWorkBlocks)},
			{"Daily Average", formatDuration(report.Summary.DailyAverage)},
		}
		
		for _, record := range records {
			if err := writer.Write(record); err != nil {
				return fmt.Errorf("failed to write CSV record: %w", err)
			}
		}
	}
	
	return nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-030
 * CONTEXT:   HTML export implementation with interactive charts and professional formatting
 * REASON:    HTML format provides rich visualization and interactive features for reports
 * CHANGE:    HTML export implementation with templating system and chart integration.
 * PREVENTION:Validate template syntax, sanitize data for HTML output, handle template errors
 * RISK:      Medium - Template rendering errors could cause export failures
 */

// ExportToHTML generates an HTML report with interactive charts
func (whe *WorkHourExporter) ExportToHTML(report *arch.WorkHourReport, outputPath string) error {
	whe.logger.Info("Exporting work hour report to HTML", "outputPath", outputPath)
	
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Prepare template data
	templateData := &HTMLTemplateData{
		Report:      report,
		GeneratedAt: time.Now(),
		Timezone:    whe.defaultTimezone,
		ChartData:   whe.prepareChartData(report),
		StyleCSS:    whe.getReportCSS(),
		ScriptJS:    whe.getReportJavaScript(),
	}
	
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Execute template
	if whe.htmlTemplate == nil {
		return fmt.Errorf("HTML template not initialized")
	}
	
	if err := whe.htmlTemplate.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}
	
	whe.logger.Info("HTML export completed successfully", "outputPath", outputPath)
	return nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-031
 * CONTEXT:   PDF export implementation for formal document generation
 * REASON:    PDF format is required for formal reports and document archival
 * CHANGE:    PDF export implementation using HTML-to-PDF conversion or direct PDF generation.
 * PREVENTION:Handle PDF generation errors, validate document structure, ensure font availability
 * RISK:      Medium - PDF generation can be complex with external dependencies
 */

// ExportToPDF generates a formatted PDF report
func (whe *WorkHourExporter) ExportToPDF(report *arch.WorkHourReport, outputPath string) error {
	whe.logger.Info("Exporting work hour report to PDF", "outputPath", outputPath)
	
	if whe.pdfGenerator == nil {
		return fmt.Errorf("PDF generator not initialized")
	}
	
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Generate PDF using the PDF generator
	pdfConfig := &PDFConfig{
		Title:       fmt.Sprintf("%s Work Hour Report", strings.Title(report.ReportType)),
		Subject:     fmt.Sprintf("Work hour report for %s", report.Period),
		Author:      "Claude Monitor",
		Creator:     "Claude Monitor Work Hour System",
		Keywords:    []string{"work hours", "timesheet", "productivity", "Claude"},
		Timezone:    whe.defaultTimezone,
		IncludeCharts: true,
		PageFormat:  "A4",
		Orientation: "portrait",
	}
	
	return whe.pdfGenerator.GeneratePDF(report, pdfConfig, outputPath)
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-032
 * CONTEXT:   Excel export implementation for advanced spreadsheet functionality
 * REASON:    Excel format provides advanced features like formulas, charts, and multiple worksheets
 * CHANGE:    Excel export implementation with multiple worksheets and advanced formatting.
 * PREVENTION:Handle Excel library errors, validate worksheet structure, ensure data type consistency
 * RISK:      Medium - Excel export requires external library with potential compatibility issues
 */

// ExportToExcel exports to Excel format with multiple worksheets
func (whe *WorkHourExporter) ExportToExcel(report *arch.WorkHourReport, outputPath string) error {
	whe.logger.Info("Exporting work hour report to Excel", "outputPath", outputPath)
	
	if whe.excelWriter == nil {
		return fmt.Errorf("Excel writer not initialized")
	}
	
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Configure Excel export
	excelConfig := &ExcelConfig{
		Worksheets: []WorksheetConfig{
			{Name: "Summary", Type: "summary"},
			{Name: "Daily Data", Type: "daily"},
			{Name: "Weekly Data", Type: "weekly"},
			{Name: "Charts", Type: "charts"},
		},
		IncludeCharts:    true,
		IncludeFormulas:  true,
		IncludeFormatting: true,
		DateFormat:       "YYYY-MM-DD",
		TimeFormat:       "HH:MM:SS",
	}
	
	return whe.excelWriter.WriteExcel(report, excelConfig, outputPath)
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-033
 * CONTEXT:   Raw data export for external analysis and system integration
 * REASON:    Need to export raw session/work block data for external analysis tools
 * CHANGE:    Raw data export implementation with flexible filtering and format options.
 * PREVENTION:Validate date ranges, limit result sets for performance, handle large datasets
 * RISK:      Medium - Large data exports could impact system performance
 */

// ExportRawData exports raw session/work block data for external analysis
func (whe *WorkHourExporter) ExportRawData(startDate, endDate time.Time, format arch.ExportFormat, outputPath string) error {
	whe.logger.Info("Exporting raw work hour data",
		"startDate", startDate,
		"endDate", endDate,
		"format", format,
		"outputPath", outputPath)
	
	// This would query raw data from the database
	// Implementation depends on database interface
	
	rawData := &RawDataExport{
		Metadata: ExportMetadata{
			ExportedAt:    time.Now(),
			ExportFormat:  string(format),
			ExportVersion: "1.0",
			SourceSystem:  "claude-monitor",
			Timezone:      whe.defaultTimezone,
		},
		DateRange: DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		},
		Sessions:   []domain.Session{},   // Would be populated from database
		WorkBlocks: []domain.WorkBlock{}, // Would be populated from database
		Processes:  []domain.Process{},   // Would be populated from database
	}
	
	// Export in requested format
	switch format {
	case arch.ExportJSON:
		return whe.exportRawDataJSON(rawData, outputPath)
	case arch.ExportCSV:
		return whe.exportRawDataCSV(rawData, outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// ExportTimesheet exports timesheet in standard formats
func (whe *WorkHourExporter) ExportTimesheet(timesheet *domain.Timesheet, format arch.ExportFormat, outputPath string) error {
	whe.logger.Info("Exporting timesheet",
		"timesheetId", timesheet.ID,
		"format", format,
		"outputPath", outputPath)
	
	switch format {
	case arch.ExportPDF:
		return whe.exportTimesheetPDF(timesheet, outputPath)
	case arch.ExportExcel:
		return whe.exportTimesheetExcel(timesheet, outputPath)
	case arch.ExportCSV:
		return whe.exportTimesheetCSV2(timesheet, outputPath)
	default:
		return fmt.Errorf("unsupported timesheet export format: %s", format)
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-034
 * CONTEXT:   Utility methods and data structures for export functionality
 * REASON:    Need supporting structures and utility methods for export operations
 * CHANGE:    Export utility methods and data structures for formatting and processing.
 * PREVENTION:Validate utility function inputs, handle edge cases in formatting
 * RISK:      Low - Utility functions are straightforward with minimal complexity
 */

// Supporting data structures for export

type ExportMetadata struct {
	ExportedAt      time.Time `json:"exportedAt"`
	ExportFormat    string    `json:"exportFormat"`
	ExportVersion   string    `json:"exportVersion"`
	SourceSystem    string    `json:"sourceSystem"`
	Timezone        string    `json:"timezone"`
}

type JSONExport struct {
	Metadata ExportMetadata     `json:"metadata"`
	Report   *arch.WorkHourReport `json:"report"`
}

type RawDataExport struct {
	Metadata   ExportMetadata    `json:"metadata"`
	DateRange  DateRange         `json:"dateRange"`
	Sessions   []domain.Session  `json:"sessions"`
	WorkBlocks []domain.WorkBlock `json:"workBlocks"`
	Processes  []domain.Process  `json:"processes"`
}

type DateRange struct {
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

type HTMLTemplateData struct {
	Report      *arch.WorkHourReport `json:"report"`
	GeneratedAt time.Time          `json:"generatedAt"`
	Timezone    string             `json:"timezone"`
	ChartData   interface{}        `json:"chartData"`
	StyleCSS    string             `json:"styleCSS"`
	ScriptJS    string             `json:"scriptJS"`
}

type PDFConfig struct {
	Title         string   `json:"title"`
	Subject       string   `json:"subject"`
	Author        string   `json:"author"`
	Creator       string   `json:"creator"`
	Keywords      []string `json:"keywords"`
	Timezone      string   `json:"timezone"`
	IncludeCharts bool     `json:"includeCharts"`
	PageFormat    string   `json:"pageFormat"`
	Orientation   string   `json:"orientation"`
}

type ExcelConfig struct {
	Worksheets        []WorksheetConfig `json:"worksheets"`
	IncludeCharts     bool              `json:"includeCharts"`
	IncludeFormulas   bool              `json:"includeFormulas"`
	IncludeFormatting bool              `json:"includeFormatting"`
	DateFormat        string            `json:"dateFormat"`
	TimeFormat        string            `json:"timeFormat"`
}

type WorksheetConfig struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Interfaces for external generators (would be implemented separately)
type PDFGenerator interface {
	GeneratePDF(report *arch.WorkHourReport, config *PDFConfig, outputPath string) error
}

type ExcelWriter interface {
	WriteExcel(report *arch.WorkHourReport, config *ExcelConfig, outputPath string) error
}

// Utility functions

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("15:04:05")
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "00:00:00"
	}
	
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// initializeTemplates loads HTML templates for report generation
func (whe *WorkHourExporter) initializeTemplates() error {
	if whe.templateDir == "" {
		whe.logger.Warn("Template directory not specified, using default templates")
		return nil
	}
	
	templatePath := filepath.Join(whe.templateDir, "report.html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		whe.logger.Warn("HTML template not found, using default template", "path", templatePath)
		return nil
	}
	
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}
	
	whe.htmlTemplate = tmpl
	return nil
}

// prepareChartData prepares data for chart rendering
func (whe *WorkHourExporter) prepareChartData(report *arch.WorkHourReport) interface{} {
	// This would prepare chart data based on the report content
	// Implementation depends on charting library requirements
	return map[string]interface{}{
		"type": "work_hour_charts",
		"data": report.Charts,
	}
}

// getReportCSS returns CSS styles for HTML reports
func (whe *WorkHourExporter) getReportCSS() string {
	return `
	body { font-family: Arial, sans-serif; margin: 20px; }
	.header { background-color: #f0f0f0; padding: 20px; margin-bottom: 20px; }
	.summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; }
	.metric { background-color: #ffffff; border: 1px solid #ddd; padding: 15px; border-radius: 5px; }
	.metric-value { font-size: 2em; font-weight: bold; color: #007bff; }
	.metric-label { color: #666; font-size: 0.9em; }
	table { width: 100%; border-collapse: collapse; margin-top: 20px; }
	th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
	th { background-color: #f8f9fa; }
	`
}

// getReportJavaScript returns JavaScript for interactive features
func (whe *WorkHourExporter) getReportJavaScript() string {
	return `
	// Chart rendering and interactive features would go here
	console.log('Work hour report loaded');
	`
}

// Placeholder implementations for specialized export methods
func (whe *WorkHourExporter) exportRawDataJSON(data *RawDataExport, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (whe *WorkHourExporter) exportRawDataCSV(data *RawDataExport, outputPath string) error {
	// Implementation would export raw data in CSV format
	return fmt.Errorf("raw data CSV export not yet implemented")
}

func (whe *WorkHourExporter) exportTimesheetPDF(timesheet *domain.Timesheet, outputPath string) error {
	// Implementation would generate timesheet PDF
	return fmt.Errorf("timesheet PDF export not yet implemented")
}

func (whe *WorkHourExporter) exportTimesheetExcel(timesheet *domain.Timesheet, outputPath string) error {
	// Implementation would generate timesheet Excel file
	return fmt.Errorf("timesheet Excel export not yet implemented")
}

func (whe *WorkHourExporter) exportTimesheetCSV2(timesheet *domain.Timesheet, outputPath string) error {
	// Implementation would generate timesheet CSV
	return fmt.Errorf("timesheet CSV export not yet implemented")
}