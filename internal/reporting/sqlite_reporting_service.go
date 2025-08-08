/**
 * CONTEXT:   SQLite reporting service coordinator orchestrating specialized report generators
 * INPUT:     Report generators and analytics calculator for focused responsibility
 * OUTPUT:    Complete reporting functionality with delegated generation logic
 * BUSINESS:  Reporting service coordinates specialized generators following SRP
 * CHANGE:    Refactored from 1,097-line god object to lightweight coordinator pattern
 * RISK:      Low - Coordinator pattern with focused generators reduces complexity
 */

package reporting

import (
	"context"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

/**
 * CONTEXT:   Reporting service coordinator with specialized generators
 * INPUT:     Dedicated generators for each report type and analytics calculator
 * OUTPUT:    Orchestrated reporting capability with clean separation of concerns
 * BUSINESS:  Coordinator pattern enables focused generators while maintaining unified interface
 * CHANGE:    Refactored to use extracted generators following Single Responsibility Principle
 * RISK:      Low - Delegation pattern with clear generator responsibilities
 */
type SQLiteReportingService struct {
	dailyGenerator    *DailyReportGenerator
	weeklyGenerator   *WeeklyReportGenerator
	monthlyGenerator  *MonthlyReportGenerator
	analyticsCalculator AnalyticsCalculator
}

/**
 * CONTEXT:   Constructor for reporting service coordinator
 * INPUT:     SQLite repositories for generator initialization and analytics calculator
 * OUTPUT:    Configured reporting service with specialized generators ready for use
 * BUSINESS:  Constructor creates focused generators and injects dependencies cleanly
 * CHANGE:    Constructor now creates generators instead of storing repositories directly
 * RISK:      Low - Constructor with generator creation and dependency injection
 */
func NewSQLiteReportingService(
	sessionRepo *sqlite.SessionRepository,
	workBlockRepo *sqlite.WorkBlockRepository,
	activityRepo *sqlite.ActivityRepository,
	projectRepo *sqlite.ProjectRepository,
) *SQLiteReportingService {
	// Create specialized generators with repository dependencies
	dailyGen := NewDailyReportGenerator(sessionRepo, workBlockRepo, activityRepo, projectRepo)
	weeklyGen := NewWeeklyReportGenerator(sessionRepo, workBlockRepo, activityRepo, projectRepo, dailyGen)
	monthlyGen := NewMonthlyReportGenerator(sessionRepo, workBlockRepo, activityRepo, projectRepo, dailyGen)
	
	// Create analytics calculator for enhanced insights
	calculator := NewDefaultAnalyticsCalculator()
	
	return &SQLiteReportingService{
		dailyGenerator:      dailyGen,
		weeklyGenerator:     weeklyGen,
		monthlyGenerator:    monthlyGen,
		analyticsCalculator: calculator,
	}
}

/**
 * CONTEXT:   Generate enhanced daily report using dedicated daily generator
 * INPUT:     User ID, date for report generation with timezone context
 * OUTPUT:    Enhanced daily report with analytics calculations and insights
 * BUSINESS:  Daily reports provide core work tracking analytics through specialized generator
 * CHANGE:    Refactored to delegate to DailyReportGenerator with analytics enhancement
 * RISK:      Low - Clean delegation to focused generator with analytics calculation
 */
func (srs *SQLiteReportingService) GenerateDailyReport(ctx context.Context, userID string, date time.Time) (*EnhancedDailyReport, error) {
	// Generate base daily report using specialized generator
	report, err := srs.dailyGenerator.GenerateDaily(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	
	// Enhance report with analytics calculations
	srs.analyticsCalculator.CalculateProjectBreakdown(report)
	srs.analyticsCalculator.CalculateHourlyBreakdown(report)
	srs.analyticsCalculator.GenerateDailyInsights(report)
	
	return report, nil
}

/**
 * CONTEXT:   Generate enhanced weekly report using dedicated weekly generator
 * INPUT:     User ID, week start date for 7-day analysis period
 * OUTPUT:    Enhanced weekly report with analytics calculations and insights
 * BUSINESS:  Weekly reports provide work pattern analysis through specialized generator
 * CHANGE:    Refactored to delegate to WeeklyReportGenerator with analytics enhancement
 * RISK:      Low - Clean delegation to focused generator with analytics calculation
 */
func (srs *SQLiteReportingService) GenerateWeeklyReport(ctx context.Context, userID string, weekStart time.Time) (*EnhancedWeeklyReport, error) {
	// Generate base weekly report using specialized generator
	report, err := srs.weeklyGenerator.GenerateWeekly(ctx, userID, weekStart)
	if err != nil {
		return nil, err
	}
	
	// Enhance report with analytics calculations
	srs.analyticsCalculator.GenerateWeeklyInsights(report)
	srs.analyticsCalculator.GenerateWeeklyTrends(report)
	
	return report, nil
}

/**
 * CONTEXT:   Generate enhanced monthly report using dedicated monthly generator
 * INPUT:     User ID, month start date for full month analysis
 * OUTPUT:    Enhanced monthly report with analytics calculations and insights
 * BUSINESS:  Monthly reports provide long-term insights through specialized generator
 * CHANGE:    Refactored to delegate to MonthlyReportGenerator with analytics enhancement
 * RISK:      Low - Clean delegation to focused generator with analytics calculation
 */
func (srs *SQLiteReportingService) GenerateMonthlyReport(ctx context.Context, userID string, monthStart time.Time) (*EnhancedMonthlyReport, error) {
	// Generate base monthly report using specialized generator
	report, err := srs.monthlyGenerator.GenerateMonthly(ctx, userID, monthStart)
	if err != nil {
		return nil, err
	}
	
	// Enhance report with analytics calculations
	srs.analyticsCalculator.GenerateMonthlyAchievements(report)
	srs.analyticsCalculator.GenerateMonthlyTrends(report)
	srs.analyticsCalculator.GenerateMonthlyInsights(report)
	
	return report, nil
}

// Coordinator interface compliance ensures consistent service contract
var _ ReportingService = (*SQLiteReportingService)(nil)


/**
 * CONTEXT:   Reporting service interface for consistent report generation contract
 * INPUT:     Context, user ID, and time parameters for different report types
 * OUTPUT:    Enhanced reports with complete analytics and insights
 * BUSINESS:  Service interface enables clean abstraction and testing
 * CHANGE:    Interface extracted for reporting service abstraction
 * RISK:      Low - Interface abstraction with clear service contract
 */
type ReportingService interface {
	GenerateDailyReport(ctx context.Context, userID string, date time.Time) (*EnhancedDailyReport, error)
	GenerateWeeklyReport(ctx context.Context, userID string, weekStart time.Time) (*EnhancedWeeklyReport, error)
	GenerateMonthlyReport(ctx context.Context, userID string, monthStart time.Time) (*EnhancedMonthlyReport, error)
}

