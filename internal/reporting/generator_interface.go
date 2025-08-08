/**
 * CONTEXT:   Common interface for all report generators in Claude Monitor
 * INPUT:     Context, date parameters, and user identification
 * OUTPUT:    Generated reports with consistent structure and error handling
 * BUSINESS:  Unified interface enables polymorphic report generation and extensibility
 * CHANGE:    Initial interface definition for Single Responsibility Principle compliance
 * RISK:      Low - Interface definition with clear contracts for implementations
 */

package reporting

import (
	"context"
	"time"
)

/**
 * CONTEXT:   Report generator interface for polymorphic report creation
 * INPUT:     Context for cancellation, user ID, and date specification
 * OUTPUT:    Generated report data structure or error for failed generation
 * BUSINESS:  Common interface enables consistent report generation across time periods
 * CHANGE:    Initial interface design supporting daily/weekly/monthly generators
 * RISK:      Low - Interface contract with clear input/output expectations
 */
type ReportGenerator interface {
	GenerateDaily(ctx context.Context, userID string, date time.Time) (*EnhancedDailyReport, error)
	GenerateWeekly(ctx context.Context, userID string, weekStart time.Time) (*EnhancedWeeklyReport, error)
	GenerateMonthly(ctx context.Context, userID string, monthStart time.Time) (*EnhancedMonthlyReport, error)
}

/**
 * CONTEXT:   Analytics calculator interface for mathematical operations
 * INPUT:     Report data structures requiring analysis
 * OUTPUT:    Enhanced reports with calculated metrics and insights
 * BUSINESS:  Separates calculation logic from data generation and formatting
 * CHANGE:    Initial analytics interface for clean separation of concerns
 * RISK:      Low - Pure calculation interface with no side effects
 */
type AnalyticsCalculator interface {
	CalculateProjectBreakdown(report *EnhancedDailyReport)
	CalculateHourlyBreakdown(report *EnhancedDailyReport)
	GenerateDailyInsights(report *EnhancedDailyReport)
	GenerateWeeklyInsights(report *EnhancedWeeklyReport)
	GenerateWeeklyTrends(report *EnhancedWeeklyReport)
	GenerateMonthlyAchievements(report *EnhancedMonthlyReport)
	GenerateMonthlyTrends(report *EnhancedMonthlyReport)
	GenerateMonthlyInsights(report *EnhancedMonthlyReport)
}

/**
 * CONTEXT:   Report formatter interface for presentation layer
 * INPUT:     Generated report data and formatting preferences
 * OUTPUT:    Formatted output ready for display or export
 * BUSINESS:  Separates presentation concerns from data generation
 * CHANGE:    Initial formatter interface for display abstraction
 * RISK:      Low - Presentation interface with no business logic
 */
type ReportFormatter interface {
	FormatDaily(report *EnhancedDailyReport) (string, error)
	FormatWeekly(report *EnhancedWeeklyReport) (string, error)
	FormatMonthly(report *EnhancedMonthlyReport) (string, error)
	FormatJSON(report interface{}) (string, error)
	FormatCSV(report interface{}) (string, error)
}