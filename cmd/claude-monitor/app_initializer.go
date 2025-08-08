/**
 * CONTEXT:   Application initialization and dependency injection for unified binary
 * INPUT:     System environment, configuration, global state requirements
 * OUTPUT:    Fully initialized application with all dependencies configured
 * BUSINESS:  Proper initialization ensures reliable application startup
 * CHANGE:    Extracted from main.go to separate initialization concerns
 * RISK:      Medium - Application bootstrap affecting all subsequent operations
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
	"github.com/claude-monitor/system/internal/reporting"
	"github.com/fatih/color"
)

// Global variables for unified reporting system
var (
	unifiedDB           *sqlite.SQLiteDB
	unifiedReportingSvc *reporting.SQLiteReportingService
	unifiedAnalytics    *reporting.WorkAnalyticsEngine
)

// Global flags
var (
	configFile   string
	verbose      bool
	outputFormat string
	noColor      bool
)

// Build information (set by build process)
var (
	Version   = "1.0.0"
	BuildTime = "development"
	GitCommit = "unknown"
)

// Color definitions for consistent beautiful CLI output
var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warningColor = color.New(color.FgYellow, color.Bold)
	infoColor    = color.New(color.FgCyan)
	headerColor  = color.New(color.FgMagenta, color.Bold)
	dimColor     = color.New(color.FgBlack, color.Bold)
)

/**
 * CONTEXT:   Initialize reporting system for unified CLI access
 * INPUT:     Database path and configuration requirements
 * OUTPUT:    Configured reporting system ready for CLI commands
 * BUSINESS:  Unified reporting enables consistent data access across commands
 * CHANGE:    Extracted reporting initialization for better organization
 * RISK:      Medium - Database initialization affecting all reporting features
 */
func initializeReporting(dbPath string) error {
	// Ensure database directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}
	
	// Initialize database connection
	var err error
	unifiedDB, err = sqlite.NewSQLiteDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	// Initialize reporting services
	unifiedReportingSvc = reporting.NewSQLiteReportingService(unifiedDB)
	unifiedAnalytics = reporting.NewWorkAnalyticsEngine(unifiedDB)
	
	return nil
}

/**
 * CONTEXT:   Close reporting system and cleanup resources
 * INPUT:     Active database connections and reporting services
 * OUTPUT:    Properly closed resources preventing resource leaks
 * BUSINESS:  Resource cleanup ensures clean application shutdown
 * CHANGE:    Extracted cleanup logic for consistent resource management
 * RISK:      Low - Resource cleanup with error handling
 */
func closeReporting() {
	if unifiedDB != nil {
		if err := unifiedDB.Close(); err != nil {
			errorColor.Printf("Warning: Failed to close database: %v\n", err)
		}
	}
}

/**
 * CONTEXT:   Get current user ID for work tracking
 * INPUT:     System environment and user detection
 * OUTPUT:    User identifier for activity correlation
 * BUSINESS:  User identification enables multi-user work tracking
 * CHANGE:    Extracted user ID logic for reusability
 * RISK:      Low - Environment variable access with fallback
 */
func getCurrentUserID() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "default"
}

/**
 * CONTEXT:   Generate unified daily report for specified user and date
 * INPUT:     User ID and target date for report generation
 * OUTPUT:    Comprehensive daily report with work analytics
 * BUSINESS:  Daily reports provide essential work tracking insights
 * CHANGE:    Extracted report generation for command reusability
 * RISK:      Medium - Report generation with database queries and formatting
 */
func generateUnifiedDailyReport(userID string, date time.Time) error {
	if unifiedReportingSvc == nil {
		return fmt.Errorf("reporting system not initialized")
	}
	
	// Generate enhanced daily report
	report, err := unifiedReportingSvc.GetEnhancedDailyReport(userID, date)
	if err != nil {
		return fmt.Errorf("failed to generate daily report: %w", err)
	}
	
	return displayEnhancedDailyReport(report, date)
}

/**
 * CONTEXT:   Display enhanced daily report with professional formatting
 * INPUT:     Enhanced daily report data and display date
 * OUTPUT:    Beautiful formatted report with professional visual design
 * BUSINESS:  Professional reports improve user experience and adoption
 * CHANGE:    Extracted display logic for consistent formatting
 * RISK:      Low - Display function with enhanced visual appeal
 */
func displayEnhancedDailyReport(report *reporting.EnhancedDailyReport, date time.Time) error {
	// Use professional display system from reporting package
	reporting.DisplayProfessionalHeader("DAILY REPORT", date.Format("Monday, January 2, 2006"))
	
	if report.TotalWorkHours == 0 {
		reporting.DisplayProfessionalEmptyState("No work activity recorded for this date.")
		return nil
	}

	// Professional metrics dashboard
	activeWork := time.Duration(report.TotalWorkHours * float64(time.Hour))
	totalTime := time.Duration(report.ScheduleHours * float64(time.Hour))
	
	// Display comprehensive report using professional formatting
	return reporting.DisplayProfessionalDailyReport(report, activeWork, totalTime)
}

/**
 * CONTEXT:   Display enhanced monthly report with analytics
 * INPUT:     Enhanced monthly report data and formatting preferences
 * OUTPUT:    Comprehensive monthly summary with professional presentation
 * BUSINESS:  Monthly reports provide long-term productivity insights
 * CHANGE:    Extracted monthly display for consistent formatting
 * RISK:      Low - Display function with professional formatting
 */
func displayEnhancedMonthlyReport(report *reporting.EnhancedMonthlyReport) error {
	return reporting.DisplayProfessionalMonthlyReport(report)
}