/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-001
 * CONTEXT:   Advanced export engine for professional work hour reporting with multi-format support
 * REASON:    Need comprehensive export functionality with professional formatting, templates, and batch processing
 * CHANGE:    Complete export engine implementation with advanced features.
 * PREVENTION:Handle large datasets efficiently, validate export parameters, implement proper error recovery
 * RISK:      Medium - Complex export operations could impact performance, require careful memory management
 */
package workhour

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-002
 * CONTEXT:   Enhanced export engine with template system and professional formatting capabilities
 * REASON:    Users need professional export capabilities with customizable templates and advanced features
 * CHANGE:    Enhanced export engine with template system, progress tracking, and batch processing.
 * PREVENTION:Validate templates and configurations, handle resource cleanup, implement concurrent safety
 * RISK:      High - Export engine coordinates multiple complex operations that could fail in various ways
 */

// ExportEngine provides comprehensive export functionality for work hour data
type ExportEngine struct {
	logger           arch.Logger
	templateManager  *TemplateManager
	pdfGenerator     PDFGenerator
	excelGenerator   ExcelGenerator
	chartGenerator   ChartGenerator
	
	// Configuration
	config           *ExportConfig
	outputDirectory  string
	tempDirectory    string
	
	// Progress tracking
	progressChannel  chan ExportProgress
	progressHandlers []ProgressHandler
	
	// Concurrency control
	maxConcurrency   int
	semaphore        chan struct{}
	
	// Template and asset management
	templates        map[string]*Template
	assets           map[string][]byte
	
	mu               sync.RWMutex
}

// ExportConfig defines configuration for the export engine
type ExportConfig struct {
	// Default settings
	DefaultTimezone      string                 `json:"defaultTimezone"`
	DefaultDateFormat    string                 `json:"defaultDateFormat"`
	DefaultTimeFormat    string                 `json:"defaultTimeFormat"`
	
	// Template settings
	TemplateDirectory    string                 `json:"templateDirectory"`
	AssetDirectory       string                 `json:"assetDirectory"`
	CustomTemplates      map[string]string      `json:"customTemplates"`
	
	// Output settings
	OutputDirectory      string                 `json:"outputDirectory"`
	TempDirectory        string                 `json:"tempDirectory"`
	CreateSubdirectories bool                   `json:"createSubdirectories"`
	CleanupTemp         bool                   `json:"cleanupTemp"`
	
	// Performance settings
	MaxConcurrency      int                    `json:"maxConcurrency"`
	ChunkSize           int                    `json:"chunkSize"`
	MemoryLimit         int64                  `json:"memoryLimit"`
	
	// Feature flags
	EnableCharts        bool                   `json:"enableCharts"`
	EnableCompression   bool                   `json:"enableCompression"`
	EnableWatermarks    bool                   `json:"enableWatermarks"`
	EnableDigitalSign   bool                   `json:"enableDigitalSignature"`
	
	// Branding
	CompanyName         string                 `json:"companyName"`
	CompanyLogo         string                 `json:"companyLogo"`
	BrandColors         map[string]string      `json:"brandColors"`
	CustomCSS           string                 `json:"customCSS"`
}

// NewExportEngine creates a new enhanced export engine
func NewExportEngine(logger arch.Logger, config *ExportConfig) (*ExportEngine, error) {
	if config == nil {
		config = getDefaultExportConfig()
	}
	
	// Ensure directories exist
	if err := os.MkdirAll(config.OutputDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	
	if err := os.MkdirAll(config.TempDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	
	engine := &ExportEngine{
		logger:           logger,
		config:           config,
		outputDirectory:  config.OutputDirectory,
		tempDirectory:    config.TempDirectory,
		maxConcurrency:   config.MaxConcurrency,
		semaphore:        make(chan struct{}, config.MaxConcurrency),
		progressChannel:  make(chan ExportProgress, 100),
		progressHandlers: make([]ProgressHandler, 0),
		templates:        make(map[string]*Template),
		assets:           make(map[string][]byte),
	}
	
	// Initialize components
	if err := engine.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize export engine: %w", err)
	}
	
	logger.Info("Export engine initialized successfully",
		"outputDir", config.OutputDirectory,
		"tempDir", config.TempDirectory,
		"maxConcurrency", config.MaxConcurrency)
	
	return engine, nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-003
 * CONTEXT:   Multi-format export request processing with unified interface
 * REASON:    Need single entry point for all export operations with consistent parameter handling
 * CHANGE:    Unified export request processor with validation and routing.
 * PREVENTION:Validate all export parameters before processing, handle format-specific requirements
 * RISK:      Medium - Export request routing errors could cause incorrect format generation
 */

// ExportRequest represents a comprehensive export request
type ExportRequest struct {
	// Basic request info
	ID              string                 `json:"id"`
	RequestedAt     time.Time              `json:"requestedAt"`
	RequestedBy     string                 `json:"requestedBy"`
	
	// Data selection
	ReportType      string                 `json:"reportType"`      // daily, weekly, monthly, timesheet, analytics
	StartDate       time.Time              `json:"startDate"`
	EndDate         time.Time              `json:"endDate"`
	EmployeeFilter  []string               `json:"employeeFilter"`
	ProjectFilter   []string               `json:"projectFilter"`
	
	// Output configuration
	Format          arch.ExportFormat      `json:"format"`
	OutputPath      string                 `json:"outputPath"`
	Filename        string                 `json:"filename"`
	Template        string                 `json:"template"`
	
	// Content options
	IncludeCharts   bool                   `json:"includeCharts"`
	IncludeBreakdown bool                  `json:"includeBreakdown"`
	IncludePatterns bool                   `json:"includePatterns"`
	IncludeTrends   bool                   `json:"includeTrends"`
	IncludeRawData  bool                   `json:"includeRawData"`
	
	// Formatting options
	Timezone        string                 `json:"timezone"`
	DateFormat      string                 `json:"dateFormat"`
	TimeFormat      string                 `json:"timeFormat"`
	CurrencyFormat  string                 `json:"currencyFormat"`
	
	// Advanced options
	CompressOutput  bool                   `json:"compressOutput"`
	DigitalSignature bool                  `json:"digitalSignature"`
	Watermark       string                 `json:"watermark"`
	CustomFields    map[string]interface{} `json:"customFields"`
	
	// Processing options
	BatchSize       int                    `json:"batchSize"`
	Priority        ExportPriority         `json:"priority"`
	Async           bool                   `json:"async"`
	
	// Callback configuration
	ProgressCallback func(ExportProgress)   `json:"-"`
	CompletionCallback func(ExportResult)  `json:"-"`
}

type ExportPriority int

const (
	PriorityLow ExportPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

// ProcessExportRequest processes a comprehensive export request
func (ee *ExportEngine) ProcessExportRequest(ctx context.Context, request *ExportRequest) (*ExportResult, error) {
	ee.logger.Info("Processing export request",
		"requestId", request.ID,
		"reportType", request.ReportType,
		"format", request.Format,
		"startDate", request.StartDate.Format("2006-01-02"),
		"endDate", request.EndDate.Format("2006-01-02"))
	
	// Validate request
	if err := ee.validateExportRequest(request); err != nil {
		return nil, fmt.Errorf("invalid export request: %w", err)
	}
	
	// Create result structure
	result := &ExportResult{
		RequestID:   request.ID,
		StartedAt:   time.Now(),
		Status:      ExportStatusProcessing,
		Progress:    0.0,
	}
	
	// Handle async processing
	if request.Async {
		go ee.processExportAsync(ctx, request, result)
		return result, nil
	}
	
	// Process synchronously
	return ee.processExportSync(ctx, request, result)
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-004
 * CONTEXT:   Professional JSON export with comprehensive metadata and structured output
 * REASON:    JSON format needs to be API-ready with complete metadata for external system integration
 * CHANGE:    Enhanced JSON export with rich metadata, structured hierarchy, and validation.
 * PREVENTION:Validate JSON structure before output, ensure all dates use consistent timezone, handle large objects
 * RISK:      Low - JSON export is well-supported with good error handling capabilities
 */

// ExportToJSONAdvanced exports comprehensive JSON with metadata and structured organization
func (ee *ExportEngine) ExportToJSONAdvanced(ctx context.Context, request *ExportRequest, report *arch.WorkHourReport) error {
	ee.logger.Info("Generating advanced JSON export", "requestId", request.ID)
	
	// Prepare comprehensive JSON structure
	jsonExport := &ComprehensiveJSONExport{
		Metadata: ExportMetadata{
			ExportID:         request.ID,
			ExportedAt:       time.Now(),
			ExportFormat:     "json",
			ExportVersion:    "2.0",
			SourceSystem:     "claude-monitor",
			ExportedBy:       request.RequestedBy,
			Timezone:         request.Timezone,
			GenerationTime:   time.Since(time.Now()),
		},
		Request: ExportRequestSummary{
			ReportType:    request.ReportType,
			DateRange:     fmt.Sprintf("%s to %s", request.StartDate.Format("2006-01-02"), request.EndDate.Format("2006-01-02")),
			EmployeeCount: len(request.EmployeeFilter),
			ProjectCount:  len(request.ProjectFilter),
			IncludeCharts: request.IncludeCharts,
		},
		Data: JSONDataStructure{
			Summary:     report.Summary,
			WorkDays:    report.WorkDays,
			WorkWeeks:   report.WorkWeeks,
			Patterns:    report.Patterns,
			Trends:      report.Trends,
			Charts:      report.Charts,
			RawData:     nil, // Populated if requested
		},
		Analytics: ee.generateAnalyticsSection(report),
		Insights:  ee.generateInsightsSection(report),
	}
	
	// Add raw data if requested
	if request.IncludeRawData {
		rawData, err := ee.fetchRawData(ctx, request)
		if err != nil {
			ee.logger.Warn("Failed to fetch raw data for JSON export", "error", err)
		} else {
			jsonExport.Data.RawData = rawData
		}
	}
	
	// Generate output file path
	outputPath := ee.generateOutputPath(request, "json")
	
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON output file: %w", err)
	}
	defer file.Close()
	
	// Configure JSON encoder
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	
	// Write JSON with error handling
	if err := encoder.Encode(jsonExport); err != nil {
		return fmt.Errorf("failed to encode JSON export: %w", err)
	}
	
	// Compress if requested
	if request.CompressOutput {
		if err := ee.compressFile(outputPath, "json"); err != nil {
			ee.logger.Warn("Failed to compress JSON export", "error", err)
		}
	}
	
	ee.logger.Info("JSON export completed successfully",
		"outputPath", outputPath,
		"fileSize", ee.getFileSize(outputPath))
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-005
 * CONTEXT:   Professional CSV export with configurable columns and data transformation
 * REASON:    CSV format needs to be optimized for spreadsheet applications with proper data formatting
 * CHANGE:    Enhanced CSV export with configurable columns, proper escaping, and multiple sheet support.
 * PREVENTION:Handle CSV field escaping properly, validate column configurations, ensure consistent formatting
 * RISK:      Low - CSV format is simple but requires proper escaping and field validation
 */

// ExportToCSVAdvanced exports data in CSV format with advanced configuration options
func (ee *ExportEngine) ExportToCSVAdvanced(ctx context.Context, request *ExportRequest, report *arch.WorkHourReport) error {
	ee.logger.Info("Generating advanced CSV export", "requestId", request.ID)
	
	// Determine CSV export type
	csvConfig := ee.getCSVConfig(request, report)
	
	// Generate output path
	outputPath := ee.generateOutputPath(request, "csv")
	
	// Handle multi-sheet CSV (separate files)
	if csvConfig.MultiSheet {
		return ee.exportMultiSheetCSV(ctx, request, report, csvConfig)
	}
	
	// Single CSV file export
	return ee.exportSingleCSV(ctx, request, report, csvConfig, outputPath)
}

// exportSingleCSV creates a single CSV file with configured columns
func (ee *ExportEngine) exportSingleCSV(ctx context.Context, request *ExportRequest, report *arch.WorkHourReport, config *CSVConfig, outputPath string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV output file: %w", err)
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Configure CSV writer
	writer.Comma = config.Separator
	writer.UseCRLF = config.UseCRLF
	
	// Write metadata header if requested
	if config.IncludeMetadata {
		if err := ee.writeCSVMetadata(writer, request, report); err != nil {
			return fmt.Errorf("failed to write CSV metadata: %w", err)
		}
	}
	
	// Write column headers
	if err := writer.Write(config.Headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}
	
	// Write data rows based on report type
	switch request.ReportType {
	case "daily":
		return ee.writeCSVDailyData(writer, report, config)
	case "weekly":
		return ee.writeCSVWeeklyData(writer, report, config)
	case "monthly":
		return ee.writeCSVMonthlyData(writer, report, config)
	case "timesheet":
		return ee.writeCSVTimesheetData(writer, report, config)
	case "analytics":
		return ee.writeCSVAnalyticsData(writer, report, config)
	default:
		return ee.writeCSVGenericData(writer, report, config)
	}
}

// exportMultiSheetCSV creates separate CSV files for different data categories
func (ee *ExportEngine) exportMultiSheetCSV(ctx context.Context, request *ExportRequest, report *arch.WorkHourReport, config *CSVConfig) error {
	baseDir := filepath.Dir(ee.generateOutputPath(request, "csv"))
	baseName := strings.TrimSuffix(request.Filename, ".csv")
	
	// Create subdirectory for multi-sheet export
	multiSheetDir := filepath.Join(baseDir, baseName+"_csv")
	if err := os.MkdirAll(multiSheetDir, 0755); err != nil {
		return fmt.Errorf("failed to create multi-sheet directory: %w", err)
	}
	
	// Export summary sheet
	summaryPath := filepath.Join(multiSheetDir, "summary.csv")
	if err := ee.exportCSVSummarySheet(summaryPath, report, config); err != nil {
		return fmt.Errorf("failed to export summary sheet: %w", err)
	}
	
	// Export daily data sheet
	if len(report.WorkDays) > 0 {
		dailyPath := filepath.Join(multiSheetDir, "daily_data.csv")
		if err := ee.exportCSVDailySheet(dailyPath, report, config); err != nil {
			return fmt.Errorf("failed to export daily sheet: %w", err)
		}
	}
	
	// Export weekly data sheet
	if len(report.WorkWeeks) > 0 {
		weeklyPath := filepath.Join(multiSheetDir, "weekly_data.csv")
		if err := ee.exportCSVWeeklySheet(weeklyPath, report, config); err != nil {
			return fmt.Errorf("failed to export weekly sheet: %w", err)
		}
	}
	
	// Compress multi-sheet directory if requested
	if request.CompressOutput {
		zipPath := multiSheetDir + ".zip"
		if err := ee.compressDirectory(multiSheetDir, zipPath); err != nil {
			ee.logger.Warn("Failed to compress multi-sheet CSV", "error", err)
		}
	}
	
	ee.logger.Info("Multi-sheet CSV export completed", "directory", multiSheetDir)
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-006
 * CONTEXT:   Professional PDF export with advanced formatting and branding capabilities
 * REASON:    PDF format requires professional presentation for formal reports and client deliverables
 * CHANGE:    Enhanced PDF export with advanced formatting, branding, charts, and digital signatures.
 * PREVENTION:Handle PDF generation errors gracefully, validate document structure, ensure font availability
 * RISK:      High - PDF generation is complex with external dependencies and potential formatting issues
 */

// ExportToPDFAdvanced generates professional PDF reports with advanced formatting
func (ee *ExportEngine) ExportToPDFAdvanced(ctx context.Context, request *ExportRequest, report *arch.WorkHourReport) error {
	ee.logger.Info("Generating advanced PDF export", "requestId", request.ID)
	
	// Check if PDF generator is available
	if ee.pdfGenerator == nil {
		return fmt.Errorf("PDF generator not available")
	}
	
	// Prepare PDF configuration
	pdfConfig := &AdvancedPDFConfig{
		// Document properties
		Title:       ee.generatePDFTitle(request, report),
		Subject:     ee.generatePDFSubject(request, report),
		Author:      ee.config.CompanyName,
		Creator:     "Claude Monitor Export Engine v2.0",
		Keywords:    ee.generatePDFKeywords(request, report),
		
		// Layout settings
		PageFormat:    "A4",
		Orientation:   "portrait",
		Margins:       PDFMargins{Top: 20, Bottom: 20, Left: 15, Right: 15},
		
		// Content settings
		IncludeCharts:     request.IncludeCharts,
		IncludeBreakdown:  request.IncludeBreakdown,
		IncludeTOC:        true,
		IncludePageNumbers: true,
		IncludeWatermark:  request.Watermark != "",
		WatermarkText:     request.Watermark,
		
		// Branding
		CompanyName:       ee.config.CompanyName,
		CompanyLogo:       ee.config.CompanyLogo,
		BrandColors:       ee.config.BrandColors,
		
		// Digital signature
		DigitalSignature:  request.DigitalSignature,
		
		// Timezone and formatting
		Timezone:          request.Timezone,
		DateFormat:        request.DateFormat,
		TimeFormat:        request.TimeFormat,
		CurrencyFormat:    request.CurrencyFormat,
	}
	
	// Generate charts if requested
	var chartData []ChartData
	if request.IncludeCharts && ee.chartGenerator != nil {
		charts, err := ee.chartGenerator.GenerateCharts(report, &ChartConfig{
			Theme:      "professional",
			ColorScheme: ee.config.BrandColors,
			Format:     "png",
			Resolution: 300, // DPI for print quality
		})
		if err != nil {
			ee.logger.Warn("Failed to generate charts for PDF", "error", err)
		} else {
			chartData = charts
		}
	}
	
	// Prepare PDF content
	pdfContent := &PDFContent{
		Report:    report,
		Request:   request,
		Charts:    chartData,
		Metadata:  ee.generatePDFMetadata(request, report),
		Template:  ee.getPDFTemplate(request.Template),
	}
	
	// Generate output path
	outputPath := ee.generateOutputPath(request, "pdf")
	
	// Generate PDF
	if err := ee.pdfGenerator.GenerateAdvancedPDF(pdfContent, pdfConfig, outputPath); err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}
	
	// Add digital signature if requested
	if request.DigitalSignature {
		if err := ee.addDigitalSignature(outputPath); err != nil {
			ee.logger.Warn("Failed to add digital signature to PDF", "error", err)
		}
	}
	
	ee.logger.Info("PDF export completed successfully",
		"outputPath", outputPath,
		"fileSize", ee.getFileSize(outputPath))
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-007
 * CONTEXT:   Interactive HTML export with responsive design and dynamic charts
 * REASON:    HTML format provides rich interactive experience for web sharing and dashboard embedding
 * CHANGE:    Enhanced HTML export with responsive design, interactive charts, and modern web features.
 * PREVENTION:Validate HTML structure, sanitize user data, ensure cross-browser compatibility
 * RISK:      Medium - HTML generation requires proper escaping and template validation
 */

// ExportToHTMLAdvanced generates interactive HTML reports with modern features
func (ee *ExportEngine) ExportToHTMLAdvanced(ctx context.Context, request *ExportRequest, report *arch.WorkHourReport) error {
	ee.logger.Info("Generating advanced HTML export", "requestId", request.ID)
	
	// Prepare template data
	templateData := &HTMLTemplateData{
		// Basic report data
		Report:      report,
		Request:     request,
		GeneratedAt: time.Now(),
		Timezone:    request.Timezone,
		
		// Interactive features
		EnableInteractive: true,
		EnableFiltering:   true,
		EnableExport:      true,
		
		// Charts and visualizations
		ChartData:    ee.prepareHTMLChartData(report, request),
		ChartConfig:  ee.getHTMLChartConfig(),
		
		// Styling and branding
		StyleCSS:     ee.generateCustomCSS(request),
		ThemeCSS:     ee.getThemeCSS(request.Template),
		BrandingCSS:  ee.getBrandingCSS(),
		
		// Interactive JavaScript
		ScriptJS:     ee.generateInteractiveJS(report, request),
		ChartJS:      ee.getChartLibraryJS(),
		
		// Responsive design
		ResponsiveCSS: ee.getResponsiveCSS(),
		MobileJS:      ee.getMobileJS(),
		
		// Export features
		ExportJS:     ee.getExportJS(),
		PrintCSS:     ee.getPrintCSS(),
		
		// Metadata
		SEOMetadata:  ee.generateSEOMetadata(report, request),
		OpenGraph:    ee.generateOpenGraphData(report, request),
	}
	
	// Get template
	template := ee.getHTMLTemplate(request.Template)
	if template == nil {
		return fmt.Errorf("HTML template not found: %s", request.Template)
	}
	
	// Generate output path
	outputPath := ee.generateOutputPath(request, "html")
	
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create HTML output file: %w", err)
	}
	defer file.Close()
	
	// Execute template
	if err := template.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute HTML template: %w", err)
	}
	
	// Copy assets if needed
	if err := ee.copyHTMLAssets(filepath.Dir(outputPath), request); err != nil {
		ee.logger.Warn("Failed to copy HTML assets", "error", err)
	}
	
	// Create standalone package if requested
	if request.CompressOutput {
		if err := ee.createHTMLPackage(outputPath); err != nil {
			ee.logger.Warn("Failed to create HTML package", "error", err)
		}
	}
	
	ee.logger.Info("HTML export completed successfully",
		"outputPath", outputPath,
		"fileSize", ee.getFileSize(outputPath))
	
	return nil
}

// Continue with the rest of the implementation...
// This file is getting quite long, so I'll create additional files for specific components

// Helper method implementations will be in separate files for better organization
func getDefaultExportConfig() *ExportConfig {
	return &ExportConfig{
		DefaultTimezone:      "UTC",
		DefaultDateFormat:    "2006-01-02",
		DefaultTimeFormat:    "15:04:05",
		OutputDirectory:      "./exports",
		TempDirectory:        "./temp",
		CreateSubdirectories: true,
		CleanupTemp:         true,
		MaxConcurrency:      4,
		ChunkSize:           1000,
		MemoryLimit:         500 * 1024 * 1024, // 500MB
		EnableCharts:        true,
		EnableCompression:   false,
		EnableWatermarks:    false,
		EnableDigitalSign:   false,
		CompanyName:         "Claude Monitor",
		BrandColors: map[string]string{
			"primary":   "#007bff",
			"secondary": "#6c757d",
			"success":   "#28a745",
			"info":      "#17a2b8",
			"warning":   "#ffc107",
			"danger":    "#dc3545",
		},
	}
}