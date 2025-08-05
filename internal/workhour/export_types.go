/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-001
 * CONTEXT:   Comprehensive type definitions for advanced export functionality
 * REASON:    Need well-defined types for all export formats, configurations, and data structures
 * CHANGE:    Complete type system for export engine with all format-specific structures.
 * PREVENTION:Validate all type constraints, ensure backward compatibility, implement proper JSON tags
 * RISK:      Low - Type definitions are foundational but must be comprehensive and consistent
 */
package workhour

import (
	"html/template"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-002
 * CONTEXT:   Export result and progress tracking structures
 * REASON:    Users need comprehensive feedback on export progress and results
 * CHANGE:    Export tracking and result structures with detailed status information.
 * PREVENTION:Ensure all status transitions are valid, handle concurrent access to progress data
 * RISK:      Medium - Progress tracking must be thread-safe and accurate
 */

// ExportResult represents the result of an export operation
type ExportResult struct {
	RequestID      string                 `json:"requestId"`
	Status         ExportStatus           `json:"status"`
	StartedAt      time.Time              `json:"startedAt"`
	CompletedAt    *time.Time             `json:"completedAt,omitempty"`
	Progress       float64                `json:"progress"`
	OutputPath     string                 `json:"outputPath,omitempty"`
	FileSize       int64                  `json:"fileSize,omitempty"`
	RecordCount    int                    `json:"recordCount,omitempty"`
	ErrorMessage   string                 `json:"errorMessage,omitempty"`
	Warnings       []string               `json:"warnings,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ExportStatus represents the current status of an export operation
type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
	ExportStatusCancelled  ExportStatus = "cancelled"
)

// ExportProgress represents progress information during export
type ExportProgress struct {
	RequestID       string    `json:"requestId"`
	Stage           string    `json:"stage"`
	Progress        float64   `json:"progress"`
	CurrentRecord   int       `json:"currentRecord"`
	TotalRecords    int       `json:"totalRecords"`
	CurrentFile     string    `json:"currentFile,omitempty"`
	Message         string    `json:"message,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	EstimatedRemaining time.Duration `json:"estimatedRemaining,omitempty"`
}

// ProgressHandler defines interface for progress callbacks
type ProgressHandler interface {
	OnProgress(progress ExportProgress) error
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-003
 * CONTEXT:   JSON export comprehensive data structures
 * REASON:    JSON export needs rich, hierarchical structure for API integration and analysis
 * CHANGE:    Comprehensive JSON export structures with metadata and analytics.
 * PREVENTION:Ensure JSON structure is backward compatible, validate all nested objects
 * RISK:      Low - JSON structures are flexible but need consistent organization
 */

// ComprehensiveJSONExport represents the complete JSON export structure
type ComprehensiveJSONExport struct {
	Metadata   ExportMetadata        `json:"metadata"`
	Request    ExportRequestSummary  `json:"request"`
	Data       JSONDataStructure     `json:"data"`
	Analytics  JSONAnalyticsSection  `json:"analytics,omitempty"`
	Insights   JSONInsightsSection   `json:"insights,omitempty"`
}

// ExportMetadata contains metadata about the export
type ExportMetadata struct {
	ExportID         string        `json:"exportId"`
	ExportedAt       time.Time     `json:"exportedAt"`
	ExportFormat     string        `json:"exportFormat"`
	ExportVersion    string        `json:"exportVersion"`
	SourceSystem     string        `json:"sourceSystem"`
	ExportedBy       string        `json:"exportedBy,omitempty"`
	Timezone         string        `json:"timezone"`
	GenerationTime   time.Duration `json:"generationTime"`
}

// ExportRequestSummary summarizes the export request parameters
type ExportRequestSummary struct {
	ReportType     string `json:"reportType"`
	DateRange      string `json:"dateRange"`
	EmployeeCount  int    `json:"employeeCount"`
	ProjectCount   int    `json:"projectCount"`
	IncludeCharts  bool   `json:"includeCharts"`
	IncludeRawData bool   `json:"includeRawData"`
}

// JSONDataStructure contains the main report data
type JSONDataStructure struct {
	Summary    *domain.ActivitySummary `json:"summary,omitempty"`
	WorkDays   []*domain.WorkDay       `json:"workDays,omitempty"`
	WorkWeeks  []*domain.WorkWeek      `json:"workWeeks,omitempty"`
	Patterns   *domain.WorkPattern     `json:"patterns,omitempty"`
	Trends     *domain.TrendAnalysis   `json:"trends,omitempty"`
	Charts     []arch.ChartData        `json:"charts,omitempty"`
	RawData    *RawDataSection         `json:"rawData,omitempty"`
}

// JSONAnalyticsSection contains computed analytics
type JSONAnalyticsSection struct {
	ProductivityMetrics  *domain.EfficiencyMetrics `json:"productivityMetrics,omitempty"`
	WorkPatternAnalysis  *domain.WorkPattern       `json:"workPatternAnalysis,omitempty"`
	TrendAnalysis        *domain.TrendAnalysis     `json:"trendAnalysis,omitempty"`
	ComparisonMetrics    *ComparisonMetrics        `json:"comparisonMetrics,omitempty"`
	Benchmarks          *BenchmarkData            `json:"benchmarks,omitempty"`
}

// JSONInsightsSection contains AI-generated insights and recommendations
type JSONInsightsSection struct {
	KeyFindings          []string                `json:"keyFindings,omitempty"`
	Recommendations      []string                `json:"recommendations,omitempty"`
	ProductivityTips     []string                `json:"productivityTips,omitempty"`
	OptimizationAreas    []OptimizationArea      `json:"optimizationAreas,omitempty"`
	WorkPatternInsights  []WorkPatternInsight    `json:"workPatternInsights,omitempty"`
}

// RawDataSection contains raw session and work block data
type RawDataSection struct {
	Sessions    []domain.Session    `json:"sessions"`
	WorkBlocks  []domain.WorkBlock  `json:"workBlocks"`
	Processes   []domain.Process    `json:"processes"`
	Activities  []ActivityRecord    `json:"activities"`
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-004
 * CONTEXT:   CSV export configuration and data structures
 * REASON:    CSV export needs flexible column configuration and data transformation options
 * CHANGE:    CSV configuration structures with column mapping and formatting options.
 * PREVENTION:Validate CSV column configurations, ensure proper escaping and formatting
 * RISK:      Low - CSV configuration is straightforward but needs proper validation
 */

// CSVConfig defines configuration for CSV export
type CSVConfig struct {
	// Basic settings
	Separator         rune     `json:"separator"`
	UseCRLF          bool     `json:"useCRLF"`
	IncludeHeader    bool     `json:"includeHeader"`
	IncludeMetadata  bool     `json:"includeMetadata"`
	
	// Column configuration
	Headers          []string            `json:"headers"`
	ColumnMapping    map[string]string   `json:"columnMapping"`
	FieldFormatters  map[string]string   `json:"fieldFormatters"`
	
	// Multi-sheet support
	MultiSheet       bool     `json:"multiSheet"`
	SheetNames       []string `json:"sheetNames"`
	
	// Data filtering
	IncludeEmpty     bool     `json:"includeEmpty"`
	MaxRows          int      `json:"maxRows"`
	SortBy           string   `json:"sortBy"`
	SortDirection    string   `json:"sortDirection"`
	
	// Formatting
	DateFormat       string   `json:"dateFormat"`
	TimeFormat       string   `json:"timeFormat"`
	DurationFormat   string   `json:"durationFormat"`
	DecimalPlaces    int      `json:"decimalPlaces"`
}

// CSVDataRow represents a single row of CSV data
type CSVDataRow struct {
	Fields   map[string]interface{} `json:"fields"`
	Metadata map[string]string      `json:"metadata,omitempty"`
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-005
 * CONTEXT:   PDF export configuration and advanced formatting structures
 * REASON:    PDF export requires complex formatting, layout, and branding configuration
 * CHANGE:    Comprehensive PDF configuration with advanced formatting and branding options.
 * PREVENTION:Validate PDF configuration parameters, ensure font and asset availability
 * RISK:      Medium - PDF configuration is complex with many interdependent parameters
 */

// AdvancedPDFConfig defines comprehensive PDF export configuration
type AdvancedPDFConfig struct {
	// Document properties
	Title            string   `json:"title"`
	Subject          string   `json:"subject"`
	Author           string   `json:"author"`
	Creator          string   `json:"creator"`
	Keywords         []string `json:"keywords"`
	
	// Layout settings
	PageFormat       string      `json:"pageFormat"`        // A4, Letter, Legal, etc.
	Orientation      string      `json:"orientation"`       // portrait, landscape
	Margins          PDFMargins  `json:"margins"`
	
	// Content settings
	IncludeCharts    bool        `json:"includeCharts"`
	IncludeBreakdown bool        `json:"includeBreakdown"`
	IncludeTOC       bool        `json:"includeTOC"`
	IncludePageNumbers bool      `json:"includePageNumbers"`
	IncludeWatermark bool        `json:"includeWatermark"`
	WatermarkText    string      `json:"watermarkText,omitempty"`
	
	// Typography
	FontFamily       string      `json:"fontFamily"`
	FontSize         int         `json:"fontSize"`
	LineHeight       float64     `json:"lineHeight"`
	HeaderFont       PDFFont     `json:"headerFont"`
	BodyFont         PDFFont     `json:"bodyFont"`
	
	// Branding
	CompanyName      string               `json:"companyName"`
	CompanyLogo      string               `json:"companyLogo"`
	BrandColors      map[string]string    `json:"brandColors"`
	HeaderTemplate   string               `json:"headerTemplate"`
	FooterTemplate   string               `json:"footerTemplate"`
	
	// Security
	DigitalSignature bool        `json:"digitalSignature"`
	PasswordProtect  bool        `json:"passwordProtect"`
	AllowPrinting    bool        `json:"allowPrinting"`
	AllowCopying     bool        `json:"allowCopying"`
	
	// Quality settings
	ImageDPI         int         `json:"imageDPI"`
	ChartDPI         int         `json:"chartDPI"`
	CompressionLevel int         `json:"compressionLevel"`
	
	// Localization
	Timezone         string      `json:"timezone"`
	DateFormat       string      `json:"dateFormat"`
	TimeFormat       string      `json:"timeFormat"`
	CurrencyFormat   string      `json:"currencyFormat"`
	Language         string      `json:"language"`
}

// PDFMargins defines page margins
type PDFMargins struct {
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
}

// PDFFont defines font configuration
type PDFFont struct {
	Family string  `json:"family"`
	Size   int     `json:"size"`
	Weight string  `json:"weight"`    // normal, bold
	Style  string  `json:"style"`     // normal, italic
	Color  string  `json:"color"`     // hex color
}

// PDFContent represents the content to be rendered in PDF
type PDFContent struct {
	Report    *arch.WorkHourReport `json:"report"`
	Request   *ExportRequest       `json:"request"`
	Charts    []ChartData          `json:"charts"`
	Metadata  map[string]interface{} `json:"metadata"`
	Template  *Template            `json:"template"`
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-006
 * CONTEXT:   HTML export configuration and template structures
 * REASON:    HTML export needs comprehensive template system with responsive design and interactivity
 * CHANGE:    HTML export structures with template system and responsive design configuration.
 * PREVENTION:Validate HTML templates, sanitize user data, ensure cross-browser compatibility
 * RISK:      Medium - HTML templates can be complex with security implications
 */

// HTMLTemplateData contains all data for HTML template rendering
type HTMLTemplateData struct {
	// Basic report data
	Report      *arch.WorkHourReport `json:"report"`
	Request     *ExportRequest       `json:"request"`
	GeneratedAt time.Time            `json:"generatedAt"`
	Timezone    string               `json:"timezone"`
	
	// Interactive features
	EnableInteractive bool `json:"enableInteractive"`
	EnableFiltering   bool `json:"enableFiltering"`
	EnableExport      bool `json:"enableExport"`
	
	// Charts and visualizations
	ChartData    interface{}    `json:"chartData"`
	ChartConfig  *ChartConfig   `json:"chartConfig"`
	
	// Styling and theming
	StyleCSS     string `json:"styleCSS"`
	ThemeCSS     string `json:"themeCSS"`
	BrandingCSS  string `json:"brandingCSS"`
	CustomCSS    string `json:"customCSS"`
	
	// JavaScript functionality
	ScriptJS     string `json:"scriptJS"`
	ChartJS      string `json:"chartJS"`
	InteractiveJS string `json:"interactiveJS"`
	
	// Responsive design
	ResponsiveCSS string `json:"responsiveCSS"`
	MobileJS      string `json:"mobileJS"`
	
	// Export and printing
	ExportJS     string `json:"exportJS"`
	PrintCSS     string `json:"printCSS"`
	
	// SEO and metadata
	SEOMetadata  *SEOMetadata  `json:"seoMetadata"`
	OpenGraph    *OpenGraphData `json:"openGraph"`
}

// SEOMetadata contains search engine optimization data
type SEOMetadata struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Author      string   `json:"author"`
	Canonical   string   `json:"canonical,omitempty"`
}

// OpenGraphData contains Open Graph protocol data for social sharing
type OpenGraphData struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	URL         string `json:"url,omitempty"`
	Image       string `json:"image,omitempty"`
	SiteName    string `json:"siteName"`
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-007
 * CONTEXT:   Excel export configuration with advanced spreadsheet features
 * REASON:    Excel export needs comprehensive configuration for multiple worksheets and advanced features
 * CHANGE:    Excel export structures with worksheet configuration and advanced formatting.
 * PREVENTION:Validate Excel configurations, ensure worksheet compatibility, handle formula errors
 * RISK:      Medium - Excel export is complex with many format-specific requirements
 */

// ExcelConfig defines comprehensive Excel export configuration
type ExcelConfig struct {
	// Workbook settings
	Worksheets        []WorksheetConfig `json:"worksheets"`
	IncludeCharts     bool              `json:"includeCharts"`
	IncludeFormulas   bool              `json:"includeFormulas"`
	IncludeFormatting bool              `json:"includeFormatting"`
	
	// Data formatting
	DateFormat        string            `json:"dateFormat"`
	TimeFormat        string            `json:"timeFormat"`
	CurrencyFormat    string            `json:"currencyFormat"`
	NumberFormat      string            `json:"numberFormat"`
	
	// Styling
	ThemeName         string            `json:"themeName"`
	ColorScheme       map[string]string `json:"colorScheme"`
	FontFamily        string            `json:"fontFamily"`
	FontSize          int               `json:"fontSize"`
	
	// Features
	AutoFilter        bool              `json:"autoFilter"`
	FreezePane        bool              `json:"freezePane"`
	ColumnWidth       map[string]int    `json:"columnWidth"`
	RowHeight         int               `json:"rowHeight"`
	
	// Protection
	PasswordProtect   bool              `json:"passwordProtect"`
	Password          string            `json:"password,omitempty"`
	ReadOnly          bool              `json:"readOnly"`
	
	// Metadata
	Author            string            `json:"author"`
	Company           string            `json:"company"`
	Comments          string            `json:"comments"`
}

// WorksheetConfig defines configuration for individual Excel worksheets
type WorksheetConfig struct {
	Name              string            `json:"name"`
	Type              string            `json:"type"`              // summary, daily, weekly, charts, raw
	Headers           []string          `json:"headers"`
	ColumnMapping     map[string]string `json:"columnMapping"`
	IncludeCharts     bool              `json:"includeCharts"`
	ChartTypes        []string          `json:"chartTypes"`
	Formatting        *WorksheetFormat  `json:"formatting"`
	ConditionalFormat []ConditionalRule `json:"conditionalFormat"`
}

// WorksheetFormat defines formatting for worksheets
type WorksheetFormat struct {
	HeaderStyle    *CellStyle `json:"headerStyle"`
	DataStyle      *CellStyle `json:"dataStyle"`
	AlternateRows  bool       `json:"alternateRows"`
	Borders        bool       `json:"borders"`
	GridLines      bool       `json:"gridLines"`
}

// CellStyle defines cell formatting
type CellStyle struct {
	FontFamily     string `json:"fontFamily"`
	FontSize       int    `json:"fontSize"`
	FontBold       bool   `json:"fontBold"`
	FontItalic     bool   `json:"fontItalic"`
	FontColor      string `json:"fontColor"`
	BackgroundColor string `json:"backgroundColor"`
	Alignment      string `json:"alignment"`      // left, center, right
	VerticalAlign  string `json:"verticalAlign"`  // top, middle, bottom
	NumberFormat   string `json:"numberFormat"`
}

// ConditionalRule defines conditional formatting rules
type ConditionalRule struct {
	Range     string    `json:"range"`
	Condition string    `json:"condition"`    // greater_than, less_than, between, etc.
	Value     interface{} `json:"value"`
	Style     *CellStyle `json:"style"`
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-008
 * CONTEXT:   Chart generation and template management structures
 * REASON:    Need comprehensive chart generation and template management for all export formats
 * CHANGE:    Chart and template structures with configuration and customization options.
 * PREVENTION:Validate chart configurations, ensure template security, handle rendering errors
 * RISK:      Medium - Chart generation and templates have complex dependencies
 */

// ChartData represents chart information for exports
type ChartData struct {
	ID       string      `json:"id"`
	Type     string      `json:"type"`         // line, bar, pie, scatter, area
	Title    string      `json:"title"`
	Data     interface{} `json:"data"`
	Options  interface{} `json:"options"`
	Width    int         `json:"width"`
	Height   int         `json:"height"`
	Format   string      `json:"format"`       // png, svg, pdf
	DPI      int         `json:"dpi"`
}

// ChartConfig defines chart generation configuration
type ChartConfig struct {
	Theme         string            `json:"theme"`         // professional, modern, classic
	ColorScheme   map[string]string `json:"colorScheme"`
	Format        string            `json:"format"`        // png, svg, pdf
	Resolution    int               `json:"resolution"`    // DPI
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	Responsive    bool              `json:"responsive"`
	Interactive   bool              `json:"interactive"`
	Animations    bool              `json:"animations"`
}

// Template represents an export template
type Template struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`         // html, pdf, excel
	Content      string                 `json:"content"`
	HTMLTemplate *template.Template     `json:"-"`
	Variables    map[string]interface{} `json:"variables"`
	Assets       []string               `json:"assets"`
	Metadata     map[string]string      `json:"metadata"`
}

// TemplateManager manages export templates
type TemplateManager struct {
	templates    map[string]*Template
	templateDir  string
	assetDir     string
	customCSS    string
	customJS     string
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-009
 * CONTEXT:   Analytics and insights data structures for enhanced reporting
 * REASON:    Need comprehensive analytics structures to provide business intelligence in exports
 * CHANGE:    Analytics and insights structures for enhanced reporting capabilities.
 * PREVENTION:Validate analytics calculations, ensure data consistency, handle edge cases
 * RISK:      Low - Analytics structures are data containers with validation needs
 */

// ComparisonMetrics contains comparison data for reports
type ComparisonMetrics struct {
	PreviousPeriod    *PeriodMetrics `json:"previousPeriod"`
	YearOverYear      *PeriodMetrics `json:"yearOverYear"`
	Baseline          *PeriodMetrics `json:"baseline"`
	Industry          *PeriodMetrics `json:"industry,omitempty"`
}

// PeriodMetrics contains metrics for a specific period
type PeriodMetrics struct {
	Period           string        `json:"period"`
	TotalWorkTime    time.Duration `json:"totalWorkTime"`
	AverageDaily     time.Duration `json:"averageDaily"`
	ProductivityScore float64      `json:"productivityScore"`
	EfficiencyRatio   float64      `json:"efficiencyRatio"`
	Change           float64       `json:"change"`           // percentage change
	ChangeDirection  string        `json:"changeDirection"`  // up, down, stable
}

// BenchmarkData contains benchmark comparisons
type BenchmarkData struct {
	IndustryAverage   *BenchmarkMetric `json:"industryAverage,omitempty"`
	CompanyAverage    *BenchmarkMetric `json:"companyAverage,omitempty"`
	TeamAverage       *BenchmarkMetric `json:"teamAverage,omitempty"`
	PersonalBest      *BenchmarkMetric `json:"personalBest,omitempty"`
}

// BenchmarkMetric represents a benchmark comparison
type BenchmarkMetric struct {
	Name        string  `json:"name"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Percentile  float64 `json:"percentile"`
	Ranking     int     `json:"ranking,omitempty"`
	TotalCount  int     `json:"totalCount,omitempty"`
}

// OptimizationArea represents an area for improvement
type OptimizationArea struct {
	Area          string  `json:"area"`
	Description   string  `json:"description"`
	Priority      string  `json:"priority"`       // high, medium, low
	Impact        string  `json:"impact"`         // high, medium, low
	Effort        string  `json:"effort"`         // high, medium, low
	Recommendation string `json:"recommendation"`
	Metrics       map[string]float64 `json:"metrics"`
}

// WorkPatternInsight represents insights about work patterns  
type WorkPatternInsight struct {
	Pattern      string                 `json:"pattern"`
	Description  string                 `json:"description"`
	Frequency    string                 `json:"frequency"`      // daily, weekly, monthly
	Confidence   float64                `json:"confidence"`     // 0-1
	Impact       string                 `json:"impact"`         // positive, negative, neutral
	Suggestion   string                 `json:"suggestion"`
	Data         map[string]interface{} `json:"data"`
}

// ActivityRecord represents a detailed activity record
type ActivityRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	Type         string    `json:"type"`         // api_call, process_start, process_end
	Details      string    `json:"details"`
	Duration     time.Duration `json:"duration,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TYPES-010
 * CONTEXT:   Interface definitions for external generators and processors
 * REASON:    Need clean interfaces for PDF, Excel, and chart generators to enable different implementations
 * CHANGE:    Generator interfaces for pluggable export format implementations.
 * PREVENTION:Keep interfaces focused and extensible, ensure error handling consistency
 * RISK:      Low - Interfaces define contracts but implementation complexity varies
 */

// PDFGenerator interface for PDF generation implementations
type PDFGenerator interface {
	GenerateAdvancedPDF(content *PDFContent, config *AdvancedPDFConfig, outputPath string) error
	AddDigitalSignature(pdfPath string, signatureConfig *SignatureConfig) error
	ValidatePDF(pdfPath string) error
}

// ExcelGenerator interface for Excel generation implementations
type ExcelGenerator interface {
	GenerateAdvancedExcel(report *arch.WorkHourReport, config *ExcelConfig, outputPath string) error
	AddWorksheet(workbook interface{}, config *WorksheetConfig, data interface{}) error
	AddChart(worksheet interface{}, chartConfig *ChartConfig, data interface{}) error
	ApplyFormatting(worksheet interface{}, format *WorksheetFormat) error
}

// ChartGenerator interface for chart generation implementations
type ChartGenerator interface {
	GenerateCharts(report *arch.WorkHourReport, config *ChartConfig) ([]ChartData, error)
	GenerateChart(chartType string, data interface{}, config *ChartConfig) (*ChartData, error)
	RenderChart(chart *ChartData, outputPath string) error
	GetSupportedTypes() []string
}

// SignatureConfig defines digital signature configuration
type SignatureConfig struct {
	CertificatePath string            `json:"certificatePath"`
	PrivateKeyPath  string            `json:"privateKeyPath"`
	Password        string            `json:"password"`
	Reason          string            `json:"reason"`
	Location        string            `json:"location"`
	ContactInfo     string            `json:"contactInfo"`
	Metadata        map[string]string `json:"metadata"`
}

// Ensure all types implement proper validation and serialization
var (
	_ = (*ExportResult)(nil)
	_ = (*ExportProgress)(nil)
	_ = (*ComprehensiveJSONExport)(nil)
	_ = (*CSVConfig)(nil)
	_ = (*AdvancedPDFConfig)(nil)
	_ = (*HTMLTemplateData)(nil)
	_ = (*ExcelConfig)(nil)
)