/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TEMPLATE-001
 * CONTEXT:   Template management system for professional export formatting and customization
 * REASON:    Need comprehensive template system to support custom branding and professional formatting
 * CHANGE:    Template management implementation with customizable templates and asset handling.
 * PREVENTION:Validate template syntax, sanitize user templates, handle asset dependencies properly
 * RISK:      Medium - Template system has security implications and complex dependency management
 */
package workhour

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/claude-monitor/claude-monitor/internal/arch"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TEMPLATE-002
 * CONTEXT:   HTML template generation and management for professional report presentation
 * REASON:    HTML reports need professional templates with responsive design and interactive features
 * CHANGE:    HTML template management with responsive design and interactive capabilities.
 * PREVENTION:Validate HTML templates for security, ensure cross-browser compatibility, handle template errors
 * RISK:      Medium - HTML templates can have security vulnerabilities and browser compatibility issues
 */

// getHTMLTemplate retrieves an HTML template by name
func (ee *ExportEngine) getHTMLTemplate(templateName string) *template.Template {
	ee.mu.RLock()
	defer ee.mu.RUnlock()
	
	if templateName == "" {
		templateName = "default_html"
	}
	
	if tmpl, exists := ee.templates[templateName]; exists && tmpl.HTMLTemplate != nil {
		return tmpl.HTMLTemplate
	}
	
	// Return default template if specific template not found
	if defaultTmpl, exists := ee.templates["default_html"]; exists && defaultTmpl.HTMLTemplate != nil {
		return defaultTmpl.HTMLTemplate
	}
	
	return nil
}

// prepareHTMLChartData prepares chart data for HTML rendering
func (ee *ExportEngine) prepareHTMLChartData(report *arch.WorkHourReport, request *ExportRequest) interface{} {
	if !request.IncludeCharts || ee.chartGenerator == nil {
		return nil
	}
	
	chartData := make(map[string]interface{})
	
	// Prepare productivity chart data
	if len(report.WorkDays) > 0 {
		productivityData := make([]map[string]interface{}, 0)
		for _, day := range report.WorkDays {
			productivityData = append(productivityData, map[string]interface{}{
				"date":        day.Date.Format("2006-01-02"),
				"hours":       day.TotalTime.Hours(),
				"efficiency":  day.GetEfficiencyRatio(),
				"sessions":    day.SessionCount,
				"blocks":      day.BlockCount,
			})
		}
		chartData["productivity"] = productivityData
	}
	
	// Prepare weekly trend data
	if len(report.WorkWeeks) > 0 {
		weeklyData := make([]map[string]interface{}, 0)
		for _, week := range report.WorkWeeks {
			weeklyData = append(weeklyData, map[string]interface{}{
				"week":        week.WeekStart.Format("2006-01-02"),
				"hours":       week.TotalTime.Hours(),
				"overtime":    week.OvertimeHours.Hours(),
				"efficiency":  week.GetEfficiencyScore(),
				"workDays":    len(week.WorkDays),
			})
		}
		chartData["weekly"] = weeklyData
	}
	
	// Prepare time distribution data
	if report.Summary != nil {
		chartData["summary"] = map[string]interface{}{
			"totalHours":    report.Summary.TotalWorkTime.Hours(),
			"totalSessions": report.Summary.TotalSessions,
			"totalBlocks":   report.Summary.TotalWorkBlocks,
			"dailyAverage":  report.Summary.DailyAverage.Hours(),
		}
	}
	
	return chartData
}

// getHTMLChartConfig returns configuration for HTML charts
func (ee *ExportEngine) getHTMLChartConfig() *ChartConfig {
	return &ChartConfig{
		Theme:       "professional",
		ColorScheme: ee.config.BrandColors,
		Format:      "svg", // SVG for web
		Width:       800,
		Height:      400,
		Responsive:  true,
		Interactive: true,
		Animations:  true,
	}
}

// generateCustomCSS generates custom CSS based on request and branding
func (ee *ExportEngine) generateCustomCSS(request *ExportRequest) string {
	css := ee.getBaseCSS()
	
	// Add branding colors
	if ee.config.BrandColors != nil {
		css += ee.generateBrandingCSS(ee.config.BrandColors)
	}
	
	// Add custom CSS if provided
	if ee.config.CustomCSS != "" {
		css += "\n/* Custom CSS */\n" + ee.config.CustomCSS
	}
	
	return css
}

// getThemeCSS returns theme-specific CSS
func (ee *ExportEngine) getThemeCSS(themeName string) string {
	switch themeName {
	case "modern":
		return ee.getModernThemeCSS()
	case "classic":
		return ee.getClassicThemeCSS()
	case "minimal":
		return ee.getMinimalThemeCSS()
	default:
		return ee.getProfessionalThemeCSS()
	}
}

// getBrandingCSS returns branding-specific CSS
func (ee *ExportEngine) getBrandingCSS() string {
	if ee.config.CompanyName == "" {
		return ""
	}
	
	return fmt.Sprintf(`
/* Company Branding */
.company-header::before {
    content: "%s";
    font-weight: bold;
    color: %s;
}
.company-logo {
    background-image: url('%s');
    background-size: contain;
    background-repeat: no-repeat;
}
`, ee.config.CompanyName, 
   ee.config.BrandColors["primary"], 
   ee.config.CompanyLogo)
}

// generateInteractiveJS generates JavaScript for interactive features
func (ee *ExportEngine) generateInteractiveJS(report *arch.WorkHourReport, request *ExportRequest) string {
	js := ee.getBaseJavaScript()
	
	// Add chart initialization if charts are enabled
	if request.IncludeCharts {
		js += ee.getChartInitializationJS()
	}
	
	// Add filtering functionality
	if request.IncludeBreakdown {
		js += ee.getFilteringJS()
	}
	
	// Add export functionality
	js += ee.getClientExportJS()
	
	return js
}

// getChartLibraryJS returns the chart library JavaScript
func (ee *ExportEngine) getChartLibraryJS() string {
	// This would typically load from CDN or include Chart.js/D3.js
	return `
/* Chart.js library would be included here or loaded from CDN */
<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script src="https://cdn.jsdelivr.net/npm/date-fns@latest/index.min.js"></script>
`
}

// getResponsiveCSS returns responsive design CSS
func (ee *ExportEngine) getResponsiveCSS() string {
	return `
/* Responsive Design */
@media (max-width: 768px) {
    .container { padding: 15px; }
    .metrics { grid-template-columns: 1fr; }
    table { font-size: 14px; }
    .chart-container { height: 300px; }
}

@media (max-width: 480px) {
    .container { padding: 10px; }
    h1 { font-size: 1.5em; }
    h2 { font-size: 1.2em; }
    .metric { padding: 15px; }
    .chart-container { height: 250px; }
}

@media print {
    .no-print { display: none !important; }
    .container { box-shadow: none; }
    body { background: white; }
    .chart-container { break-inside: avoid; }
}
`
}

// getMobileJS returns mobile-specific JavaScript
func (ee *ExportEngine) getMobileJS() string {
	return `
/* Mobile optimizations */
if (window.innerWidth <= 768) {
    document.body.classList.add('mobile');
    
    // Optimize touch interactions
    document.addEventListener('touchstart', function() {}, {passive: true});
    
    // Adjust chart sizes for mobile
    if (typeof adjustChartsForMobile === 'function') {
        adjustChartsForMobile();
    }
}
`
}

// getExportJS returns client-side export functionality
func (ee *ExportEngine) getExportJS() string {
	return `
/* Client-side export functionality */
function exportToPDF() {
    window.print();
}

function exportToCSV() {
    const tables = document.querySelectorAll('table');
    if (tables.length === 0) return;
    
    let csv = '';
    const table = tables[0];
    
    for (let row of table.rows) {
        const cells = Array.from(row.cells);
        csv += cells.map(cell => '"' + cell.textContent.replace(/"/g, '""') + '"').join(',') + '\\n';
    }
    
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'work_hour_report.csv';
    a.click();
    window.URL.revokeObjectURL(url);
}
`
}

// getPrintCSS returns print-specific CSS
func (ee *ExportEngine) getPrintCSS() string {
	return `
/* Print Styles */
@media print {
    @page {
        margin: 0.5in;
        size: A4;
    }
    
    body {
        font-size: 12pt;
        line-height: 1.4;
        color: black;
        background: white;
    }
    
    .container {
        box-shadow: none;
        border: none;
        background: white;
    }
    
    .no-print,
    .interactive-controls,
    .export-buttons {
        display: none !important;
    }
    
    h1, h2, h3 {
        page-break-after: avoid;
    }
    
    table {
        page-break-inside: avoid;
    }
    
    .chart-container {
        page-break-inside: avoid;
        max-height: 4in;
    }
    
    .metric {
        border: 1px solid #ccc;
    }
}
`
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TEMPLATE-003
 * CONTEXT:   SEO and social media metadata generation for HTML exports
 * REASON:    HTML reports shared online need proper metadata for search engines and social platforms
 * CHANGE:    SEO and social media metadata generation for professional presentation.
 * PREVENTION:Validate metadata content, ensure proper escaping, handle missing data gracefully
 * RISK:      Low - Metadata generation is informational and doesn't affect core functionality
 */

// generateSEOMetadata generates SEO metadata for HTML reports
func (ee *ExportEngine) generateSEOMetadata(report *arch.WorkHourReport, request *ExportRequest) *SEOMetadata {
	title := fmt.Sprintf("%s - %s", report.Title, ee.config.CompanyName)
	description := fmt.Sprintf("Work hour report for %s generated by Claude Monitor", report.Period)
	
	keywords := []string{
		"work hours",
		"productivity",
		"timesheet",
		"analytics",
		request.ReportType,
		"Claude Monitor",
	}
	
	return &SEOMetadata{
		Title:       title,
		Description: description,
		Keywords:    keywords,
		Author:      ee.config.CompanyName,
	}
}

// generateOpenGraphData generates Open Graph data for social sharing
func (ee *ExportEngine) generateOpenGraphData(report *arch.WorkHourReport, request *ExportRequest) *OpenGraphData {
	return &OpenGraphData{
		Title:       report.Title,
		Description: fmt.Sprintf("Work hour analytics for %s", report.Period),
		Type:        "article",
		SiteName:    ee.config.CompanyName + " Work Hour Reports",
	}
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TEMPLATE-004
 * CONTEXT:   Asset management for HTML exports including images, CSS, and JavaScript
 * REASON:    HTML reports need proper asset management for standalone and packaged distribution
 * CHANGE:    Asset management implementation with copying and packaging capabilities.
 * PREVENTION:Validate asset paths, handle missing assets gracefully, implement proper cleanup
 * RISK:      Low - Asset management is straightforward file operations with proper error handling
 */

// copyHTMLAssets copies necessary assets for HTML reports
func (ee *ExportEngine) copyHTMLAssets(outputDir string, request *ExportRequest) error {
	assetsDir := filepath.Join(outputDir, "assets")
	
	// Create assets directory
	if err := ee.createDirectory(assetsDir); err != nil {
		return fmt.Errorf("failed to create assets directory: %w", err)
	}
	
	// Copy CSS files
	if err := ee.copyAsset("default.css", filepath.Join(assetsDir, "styles.css")); err != nil {
		ee.logger.Warn("Failed to copy CSS asset", "error", err)
	}
	
	// Copy JavaScript files
	if err := ee.copyAsset("default.js", filepath.Join(assetsDir, "scripts.js")); err != nil {
		ee.logger.Warn("Failed to copy JavaScript asset", "error", err)
	}
	
	// Copy company logo if available
	if ee.config.CompanyLogo != "" {
		logoPath := filepath.Join(assetsDir, "logo.png")
		if err := ee.copyFile(ee.config.CompanyLogo, logoPath); err != nil {
			ee.logger.Warn("Failed to copy company logo", "error", err)
		}
	}
	
	return nil
}

// createHTMLPackage creates a standalone HTML package
func (ee *ExportEngine) createHTMLPackage(htmlPath string) error {
	packageDir := strings.TrimSuffix(htmlPath, ".html") + "_package"
	
	// Create package directory
	if err := ee.createDirectory(packageDir); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}
	
	// Copy HTML file
	packageHTMLPath := filepath.Join(packageDir, "index.html")
	if err := ee.copyFile(htmlPath, packageHTMLPath); err != nil {
		return fmt.Errorf("failed to copy HTML file: %w", err)
	}
	
	// Copy assets
	if err := ee.copyHTMLAssets(packageDir, &ExportRequest{}); err != nil {
		return fmt.Errorf("failed to copy assets: %w", err)
	}
	
	// Create README file
	readmePath := filepath.Join(packageDir, "README.txt")
	readmeContent := ee.generatePackageReadme()
	if err := ee.writeFile(readmePath, readmeContent); err != nil {
		ee.logger.Warn("Failed to create README file", "error", err)
	}
	
	// Compress package if enabled
	if ee.config.EnableCompression {
		zipPath := packageDir + ".zip"
		if err := ee.compressDirectory(packageDir, zipPath); err != nil {
			return fmt.Errorf("failed to compress package: %w", err)
		}
	}
	
	return nil
}

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-EXPORT-TEMPLATE-005
 * CONTEXT:   CSS theme implementations for different visual styles
 * REASON:    Different organizations need different visual themes for their reports
 * CHANGE:    Multiple CSS theme implementations with professional styling options.
 * PREVENTION:Ensure cross-browser compatibility, validate CSS syntax, maintain consistent styling
 * RISK:      Low - CSS themes are visual only and don't affect functionality
 */

// getBaseCSS returns the base CSS for all themes
func (ee *ExportEngine) getBaseCSS() string {
	return `
/* Base Styles */
* {
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    line-height: 1.6;
    color: #333;
    margin: 0;
    padding: 0;
    background-color: #f8f9fa;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 30px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

/* Typography */
h1, h2, h3, h4, h5, h6 {
    margin: 0 0 1rem;
    font-weight: 600;
    line-height: 1.2;
}

h1 { font-size: 2.5rem; }
h2 { font-size: 2rem; }
h3 { font-size: 1.5rem; }

/* Layout */
header {
    text-align: center;
    margin-bottom: 2rem;
    padding-bottom: 1rem;
    border-bottom: 2px solid #e9ecef;
}

main {
    margin: 2rem 0;
}

section {
    margin: 2rem 0;
}

/* Metrics Grid */
.metrics {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 1.5rem;
    margin: 1.5rem 0;
}

.metric {
    text-align: center;
    padding: 1.5rem;
    background: #f8f9fa;
    border-radius: 8px;
    border: 1px solid #e9ecef;
    transition: transform 0.2s, box-shadow 0.2s;
}

.metric:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 15px rgba(0,0,0,0.1);
}

.metric-value {
    display: block;
    font-size: 2.5rem;
    font-weight: 700;
    color: #007bff;
    margin-bottom: 0.5rem;
}

.metric-label {
    color: #6c757d;
    font-size: 0.9rem;
    font-weight: 500;
}

/* Tables */
table {
    width: 100%;
    border-collapse: collapse;
    margin: 1.5rem 0;
    background: white;
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
}

th, td {
    padding: 1rem;
    text-align: left;
    border-bottom: 1px solid #e9ecef;
}

th {
    background-color: #f8f9fa;
    font-weight: 600;
    color: #495057;
    font-size: 0.9rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
}

tr:hover {
    background-color: #f8f9fa;
}

tr:last-child td {
    border-bottom: none;
}

/* Charts */
.chart-container {
    margin: 2rem 0;
    padding: 1rem;
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    position: relative;
    height: 400px;
}

.chart-title {
    text-align: center;
    margin-bottom: 1rem;
    font-size: 1.2rem;
    font-weight: 600;
    color: #495057;
}

/* Footer */
footer {
    text-align: center;
    margin-top: 3rem;
    padding-top: 2rem;
    border-top: 1px solid #e9ecef;
    color: #6c757d;
    font-size: 0.9rem;
}

/* Utilities */
.text-center { text-align: center; }
.text-right { text-align: right; }
.text-muted { color: #6c757d; }
.text-primary { color: #007bff; }
.text-success { color: #28a745; }
.text-warning { color: #ffc107; }
.text-danger { color: #dc3545; }

.mb-0 { margin-bottom: 0; }
.mb-1 { margin-bottom: 0.5rem; }
.mb-2 { margin-bottom: 1rem; }
.mb-3 { margin-bottom: 1.5rem; }
.mb-4 { margin-bottom: 2rem; }

.mt-0 { margin-top: 0; }
.mt-1 { margin-top: 0.5rem; }
.mt-2 { margin-top: 1rem; }
.mt-3 { margin-top: 1.5rem; }
.mt-4 { margin-top: 2rem; }
`
}

// generateBrandingCSS generates CSS for brand colors
func (ee *ExportEngine) generateBrandingCSS(colors map[string]string) string {
	css := "\n/* Brand Colors */\n"
	
	if primary, ok := colors["primary"]; ok {
		css += fmt.Sprintf(`
.text-primary, .metric-value { color: %s !important; }
.bg-primary { background-color: %s !important; }
.border-primary { border-color: %s !important; }
`, primary, primary, primary)
	}
	
	if secondary, ok := colors["secondary"]; ok {
		css += fmt.Sprintf(`
.text-secondary { color: %s !important; }
.bg-secondary { background-color: %s !important; }
`, secondary, secondary)
	}
	
	return css
}

// getProfessionalThemeCSS returns professional theme CSS
func (ee *ExportEngine) getProfessionalThemeCSS() string {
	return `
/* Professional Theme */
body {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
    padding: 2rem 0;
}

.container {
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.2);
}

.metric {
    background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
    border: 1px solid rgba(0, 123, 255, 0.1);
}

.metric:hover {
    background: linear-gradient(135deg, #e9ecef 0%, #dee2e6 100%);
}

h1, h2 {
    background: linear-gradient(135deg, #007bff, #6f42c1);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
}
`
}

// getModernThemeCSS returns modern theme CSS
func (ee *ExportEngine) getModernThemeCSS() string {
	return `
/* Modern Theme */
body {
    background: #1a1a1a;
    color: #e0e0e0;
}

.container {
    background: #2d2d2d;
    border: 1px solid #404040;
    color: #e0e0e0;
}

.metric {
    background: #3a3a3a;
    border: 1px solid #505050;
    color: #e0e0e0;
}

.metric-value {
    color: #00d4aa;
}

th {
    background-color: #404040;
    color: #e0e0e0;
}

tr:hover {
    background-color: #404040;
}

table {
    background: #2d2d2d;
}
`
}

// getClassicThemeCSS returns classic theme CSS
func (ee *ExportEngine) getClassicThemeCSS() string {
	return `
/* Classic Theme */
body {
    font-family: Georgia, 'Times New Roman', serif;
    background: #f9f7f4;
}

.container {
    background: #ffffff;
    border: 2px solid #8b4513;
    border-radius: 0;
}

h1, h2, h3 {
    color: #8b4513;
    font-family: Georgia, serif;
}

.metric {
    background: #f9f7f4;
    border: 1px solid #8b4513;
    border-radius: 0;
}

.metric-value {
    color: #8b4513;
}

th {
    background-color: #8b4513;
    color: white;
}
`
}

// getMinimalThemeCSS returns minimal theme CSS
func (ee *ExportEngine) getMinimalThemeCSS() string {
	return `
/* Minimal Theme */
body {
    background: white;
    font-family: 'Helvetica Neue', Arial, sans-serif;
}

.container {
    background: white;
    box-shadow: none;
    border: none;
    border-top: 4px solid #007bff;
}

.metric {
    background: white;
    border: 1px solid #e0e0e0;
    box-shadow: none;
}

.metric:hover {
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

table {
    box-shadow: none;
    border-radius: 0;
}

th {
    background: white;
    border-bottom: 2px solid #007bff;
}
`
}

// Utility methods for asset management

// createDirectory creates a directory if it doesn't exist
func (ee *ExportEngine) createDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// copyAsset copies an asset from memory to file
func (ee *ExportEngine) copyAsset(assetName, destPath string) error {
	ee.mu.RLock()
	asset, exists := ee.assets[assetName]
	ee.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("asset not found: %s", assetName)
	}
	
	return ioutil.WriteFile(destPath, asset, 0644)
}

// copyFile copies a file from source to destination
func (ee *ExportEngine) copyFile(srcPath, destPath string) error {
	data, err := ioutil.ReadFile(srcPath)
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(destPath, data, 0644)
}

// writeFile writes content to a file
func (ee *ExportEngine) writeFile(path, content string) error {
	return ioutil.WriteFile(path, []byte(content), 0644)
}

// generatePackageReadme generates a README for HTML packages
func (ee *ExportEngine) generatePackageReadme() string {
	return fmt.Sprintf(`Claude Monitor Work Hour Report Package
=====================================

This package contains a standalone HTML report generated by Claude Monitor.

Contents:
- index.html: Main report file
- assets/: Supporting files (CSS, JavaScript, images)

To view the report:
1. Open index.html in any modern web browser
2. All dependencies are included in this package

Generated: %s
Export Engine: Claude Monitor v2.0

For support, visit: https://claude-monitor.example.com
`, time.Now().Format("2006-01-02 15:04:05"));
}

// JavaScript implementations

// getBaseJavaScript returns base JavaScript functionality
func (ee *ExportEngine) getBaseJavaScript() string {
	return `
/* Base JavaScript Functionality */
document.addEventListener('DOMContentLoaded', function() {
    console.log('Claude Monitor Report initialized');
    
    // Add export buttons
    addExportButtons();
    
    // Initialize tooltips
    initializeTooltips();
    
    // Add responsive table handling
    handleResponsiveTables();
});

function addExportButtons() {
    const header = document.querySelector('header');
    if (!header) return;
    
    const buttonContainer = document.createElement('div');
    buttonContainer.className = 'export-buttons no-print';
    buttonContainer.style.cssText = 'margin-top: 1rem; text-align: center;';
    
    const pdfButton = document.createElement('button');
    pdfButton.textContent = 'Export to PDF';
    pdfButton.onclick = exportToPDF;
    pdfButton.style.cssText = 'margin: 0 0.5rem; padding: 0.5rem 1rem; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;';
    
    const csvButton = document.createElement('button');
    csvButton.textContent = 'Export to CSV';
    csvButton.onclick = exportToCSV;
    csvButton.style.cssText = 'margin: 0 0.5rem; padding: 0.5rem 1rem; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer;';
    
    buttonContainer.appendChild(pdfButton);
    buttonContainer.appendChild(csvButton);
    header.appendChild(buttonContainer);
}

function initializeTooltips() {
    const elements = document.querySelectorAll('[data-tooltip]');
    elements.forEach(element => {
        element.addEventListener('mouseenter', showTooltip);
        element.addEventListener('mouseleave', hideTooltip);
    });
}

function handleResponsiveTables() {
    const tables = document.querySelectorAll('table');
    tables.forEach(table => {
        const wrapper = document.createElement('div');
        wrapper.style.cssText = 'overflow-x: auto; margin: 1rem 0;';
        table.parentNode.insertBefore(wrapper, table);
        wrapper.appendChild(table);
    });
}
`
}

// getChartInitializationJS returns chart initialization JavaScript
func (ee *ExportEngine) getChartInitializationJS() string {
	return `
/* Chart Initialization */
function initializeCharts() {
    if (typeof Chart === 'undefined') {
        console.warn('Chart.js library not loaded');
        return;
    }
    
    // Initialize productivity chart
    initProductivityChart();
    
    // Initialize weekly trend chart
    initWeeklyTrendChart();
    
    // Initialize summary pie chart
    initSummaryChart();
}

function initProductivityChart() {
    const canvas = document.getElementById('productivityChart');
    if (!canvas || !window.chartData?.productivity) return;
    
    const ctx = canvas.getContext('2d');
    new Chart(ctx, {
        type: 'line',
        data: {
            labels: window.chartData.productivity.map(d => d.date),
            datasets: [{
                label: 'Work Hours',
                data: window.chartData.productivity.map(d => d.hours),
                borderColor: '#007bff',
                backgroundColor: 'rgba(0, 123, 255, 0.1)',
                tension: 0.4
            }, {
                label: 'Efficiency',
                data: window.chartData.productivity.map(d => d.efficiency * 10),
                borderColor: '#28a745',
                backgroundColor: 'rgba(40, 167, 69, 0.1)',
                yAxisID: 'y1'
            }]
        },
        options: {
            responsive: true,
            scales: {
                y: {
                    beginAtZero: true,
                    title: { display: true, text: 'Hours' }
                },
                y1: {
                    type: 'linear',
                    display: true,
                    position: 'right',
                    title: { display: true, text: 'Efficiency (x10)' },
                    grid: { drawOnChartArea: false }
                }
            }
        }
    });
}

function adjustChartsForMobile() {
    const charts = document.querySelectorAll('.chart-container');
    charts.forEach(chart => {
        chart.style.height = '250px';
    });
}

// Call chart initialization after DOM load
document.addEventListener('DOMContentLoaded', function() {
    setTimeout(initializeCharts, 100);
});
`
}

// getFilteringJS returns filtering functionality JavaScript
func (ee *ExportEngine) getFilteringJS() string {
	return `
/* Filtering Functionality */
function addFilterControls() {
    const main = document.querySelector('main');
    if (!main) return;
    
    const filterContainer = document.createElement('div');
    filterContainer.className = 'filter-controls no-print';
    filterContainer.style.cssText = 'margin: 1rem 0; padding: 1rem; background: #f8f9fa; border-radius: 4px;';
    
    // Date range filter
    const dateFilter = document.createElement('input');
    dateFilter.type = 'date';
    dateFilter.addEventListener('change', filterByDate);
    
    // Efficiency filter
    const efficiencyFilter = document.createElement('select');
    efficiencyFilter.innerHTML = '<option value="">All Efficiency Levels</option><option value="high">High (>80%)</option><option value="medium">Medium (60-80%)</option><option value="low">Low (<60%)</option>';
    efficiencyFilter.addEventListener('change', filterByEfficiency);
    
    filterContainer.appendChild(document.createTextNode('Filter by date: '));
    filterContainer.appendChild(dateFilter);
    filterContainer.appendChild(document.createTextNode(' Efficiency: '));
    filterContainer.appendChild(efficiencyFilter);
    
    main.insertBefore(filterContainer, main.firstChild);
}

function filterByDate(event) {
    const selectedDate = event.target.value;
    const rows = document.querySelectorAll('table tbody tr');
    
    rows.forEach(row => {
        const dateCell = row.cells[0];
        if (dateCell && (selectedDate === '' || dateCell.textContent.includes(selectedDate))) {
            row.style.display = '';
        } else {
            row.style.display = 'none';
        }
    });
}

function filterByEfficiency(event) {
    const filterValue = event.target.value;
    const rows = document.querySelectorAll('table tbody tr');
    
    rows.forEach(row => {
        if (row.cells.length < 8) return;
        
        const efficiencyCell = row.cells[7];
        const efficiency = parseFloat(efficiencyCell.textContent);
        
        let show = true;
        if (filterValue === 'high' && efficiency <= 0.8) show = false;
        if (filterValue === 'medium' && (efficiency < 0.6 || efficiency > 0.8)) show = false;
        if (filterValue === 'low' && efficiency >= 0.6) show = false;
        
        row.style.display = show ? '' : 'none';
    });
}

// Add filter controls after DOM load
document.addEventListener('DOMContentLoaded', function() {
    addFilterControls();
});
`
}

// getClientExportJS returns client-side export functionality
func (ee *ExportEngine) getClientExportJS() string {
	return ee.getExportJS() // Already implemented above
}