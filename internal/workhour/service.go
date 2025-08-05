/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-001
 * CONTEXT:   Work hour service coordinator implementing high-level business operations
 * REASON:    Need centralized service to orchestrate analyzer, timesheet manager, and reporting components
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle service coordination failures gracefully, maintain clear service boundaries
 * RISK:      Medium - Service coordination failures could affect reporting functionality
 */
package workhour

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-002
 * CONTEXT:   WorkHourService implements high-level coordination of work hour analytics and reporting
 * REASON:    Business requirement for unified service API coordinating all work hour functionality
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement proper error handling and service isolation, monitor service health
 * RISK:      High - Service coordination affects all work hour functionality
 */
type WorkHourService struct {
	analyzer         arch.WorkHourAnalyzer
	timesheetManager arch.TimesheetManager
	dbManager        arch.WorkHourDatabaseManager
	logger           arch.Logger
	configuration    *arch.WorkHourConfiguration
	mu               sync.RWMutex
	
	// Event integration for real-time updates
	eventChannel     chan WorkHourEvent
	eventSubscribers []WorkHourEventSubscriber
	ctx              context.Context
	cancel           context.CancelFunc
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-003
 * CONTEXT:   WorkHourEvent defines events for real-time work hour system integration
 * REASON:    Need event-driven updates to maintain real-time synchronization with enhanced daemon
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep event payloads lightweight, handle event processing failures gracefully
 * RISK:      Medium - Event processing failures could cause data inconsistencies
 */
type WorkHourEvent struct {
	Type      WorkHourEventType `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      interface{}       `json:"data"`
}

type WorkHourEventType string

const (
	EventWorkDayUpdated    WorkHourEventType = "work_day_updated"
	EventWorkWeekUpdated   WorkHourEventType = "work_week_updated"
	EventTimesheetCreated  WorkHourEventType = "timesheet_created"
	EventTimesheetSubmitted WorkHourEventType = "timesheet_submitted"
	EventCacheInvalidated  WorkHourEventType = "cache_invalidated"
)

type WorkHourEventSubscriber interface {
	OnWorkHourEvent(event WorkHourEvent) error
}

func NewWorkHourService(
	analyzer arch.WorkHourAnalyzer,
	timesheetManager arch.TimesheetManager,
	dbManager arch.WorkHourDatabaseManager,
	logger arch.Logger,
) *WorkHourService {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkHourService{
		analyzer:         analyzer,
		timesheetManager: timesheetManager,
		dbManager:        dbManager,
		logger:           logger,
		configuration: &arch.WorkHourConfiguration{
			DefaultPolicy: domain.TimesheetPolicy{
				RoundingInterval:  15 * time.Minute,
				RoundingMethod:    domain.RoundNearest,
				OvertimeThreshold: 8 * time.Hour,
				WeeklyThreshold:   40 * time.Hour,
				BreakDeduction:    30 * time.Minute,
			},
			ReportTimezone:        "Local",
			WorkWeekStart:         time.Monday,
			StandardWorkHours:     40 * time.Hour,
			OvertimeThreshold:     40 * time.Hour,
			BreakDeduction:        30 * time.Minute,
			CacheRefreshInterval:  1 * time.Hour,
			EnableTrendAnalysis:   true,
			EnablePatternAnalysis: true,
		},
		eventChannel:     make(chan WorkHourEvent, 100),
		eventSubscribers: make([]WorkHourEventSubscriber, 0),
		ctx:              ctx,
		cancel:           cancel,
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-004
 * CONTEXT:   Start initializes work hour service with background processing
 * REASON:    Need background processing for event handling and cache management
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure proper goroutine lifecycle management and resource cleanup
 * RISK:      Medium - Background processing failures could affect service responsiveness
 */
func (whs *WorkHourService) Start() error {
	whs.logger.Info("Starting work hour service")
	
	// Start event processing goroutine
	go whs.processEvents()
	
	// Start cache refresh goroutine
	go whs.cacheRefreshLoop()
	
	whs.logger.Info("Work hour service started successfully")
	return nil
}

func (whs *WorkHourService) Stop() error {
	whs.logger.Info("Stopping work hour service")
	
	whs.cancel()
	
	whs.logger.Info("Work hour service stopped")
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-005
 * CONTEXT:   Core analysis operations implementing WorkHourService interface
 * REASON:    Business requirement for unified access to work hour analysis functionality
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate input parameters and handle service dependencies gracefully
 * RISK:      High - Analysis operations are core to work hour functionality
 */
func (whs *WorkHourService) GetDailyWorkSummary(date time.Time) (*domain.WorkDay, error) {
	workDay, err := whs.analyzer.AnalyzeWorkDay(date)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze work day: %w", err)
	}

	// Emit event for real-time updates
	whs.emitEvent(WorkHourEvent{
		Type:      EventWorkDayUpdated,
		Timestamp: time.Now(),
		Data:      workDay,
	})

	whs.logger.Debug("Daily work summary retrieved",
		"date", date.Format("2006-01-02"),
		"totalTime", workDay.TotalTime,
		"sessionCount", workDay.SessionCount)

	return workDay, nil
}

func (whs *WorkHourService) GetWeeklyWorkSummary(weekStart time.Time) (*domain.WorkWeek, error) {
	workWeek, err := whs.analyzer.AnalyzeWorkWeek(weekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze work week: %w", err)
	}

	// Emit event for real-time updates
	whs.emitEvent(WorkHourEvent{
		Type:      EventWorkWeekUpdated,
		Timestamp: time.Now(),
		Data:      workWeek,
	})

	whs.logger.Debug("Weekly work summary retrieved",
		"weekStart", weekStart.Format("2006-01-02"),
		"totalTime", workWeek.TotalTime,
		"overtimeHours", workWeek.OvertimeHours)

	return workWeek, nil
}

func (whs *WorkHourService) GetMonthlyWorkSummary(year int, month time.Month) ([]*domain.WorkDay, error) {
	workDays, err := whs.dbManager.GetWorkMonthData(year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly work data: %w", err)
	}

	whs.logger.Debug("Monthly work summary retrieved",
		"year", year,
		"month", month,
		"workDays", len(workDays))

	return workDays, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-006
 * CONTEXT:   Report generation coordinating analyzer and formatter components
 * REASON:    Business requirement for comprehensive report generation with multiple formats
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate report parameters and handle large report generation efficiently
 * RISK:      Medium - Report generation could impact system performance with large datasets
 */
func (whs *WorkHourService) GenerateReport(
	reportType string,
	startDate, endDate time.Time,
	config arch.ReportConfig,
) (*arch.WorkHourReport, error) {
	
	whs.logger.Info("Generating work hour report",
		"reportType", reportType,
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"))

	// Create report structure
	report := &arch.WorkHourReport{
		ID:          fmt.Sprintf("report_%d", time.Now().Unix()),
		Title:       fmt.Sprintf("%s Work Hour Report", reportType),
		ReportType:  reportType,
		Period:      fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		StartDate:   startDate,
		EndDate:     endDate,
		GeneratedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Generate report content based on type
	switch reportType {
	case "daily":
		err := whs.generateDailyReportContent(report, startDate, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate daily report content: %w", err)
		}
		
	case "weekly":
		err := whs.generateWeeklyReportContent(report, startDate, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate weekly report content: %w", err)
		}
		
	case "monthly":
		err := whs.generateMonthlyReportContent(report, startDate.Year(), startDate.Month(), config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate monthly report content: %w", err)
		}
		
	case "custom":
		err := whs.generateCustomReportContent(report, startDate, endDate, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate custom report content: %w", err)
		}
		
	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}

	whs.logger.Info("Work hour report generated successfully",
		"reportID", report.ID,
		"reportType", reportType)

	return report, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-007
 * CONTEXT:   Report content generation methods for different report types
 * REASON:    Need specialized content generation for different reporting requirements
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle edge cases in data aggregation, validate report content consistency
 * RISK:      Medium - Report content accuracy affects business decision making
 */
func (whs *WorkHourService) generateDailyReportContent(
	report *arch.WorkHourReport,
	date time.Time,
	config arch.ReportConfig,
) error {
	// Get daily work summary
	workDay, err := whs.GetDailyWorkSummary(date)
	if err != nil {
		return fmt.Errorf("failed to get daily work summary: %w", err)
	}

	report.WorkDays = []*domain.WorkDay{workDay}

	// Generate activity summary
	if config.IncludeBreakdown {
		summary, err := whs.analyzer.GenerateActivitySummary("daily", date, date)
		if err != nil {
			whs.logger.Warn("Failed to generate activity summary for daily report", "error", err)
		} else {
			report.Summary = summary
		}
	}

	// Include pattern analysis if requested
	if config.IncludePatterns {
		// Use a week of data for pattern analysis
		weekStart := date.AddDate(0, 0, -7)
		patterns, err := whs.analyzer.AnalyzeWorkPattern(weekStart, date)
		if err != nil {
			whs.logger.Warn("Failed to analyze work patterns for daily report", "error", err)
		} else {
			report.Patterns = patterns
		}
	}

	return nil
}

func (whs *WorkHourService) generateWeeklyReportContent(
	report *arch.WorkHourReport,
	weekStart time.Time,
	config arch.ReportConfig,
) error {
	// Get weekly work summary
	workWeek, err := whs.GetWeeklyWorkSummary(weekStart)
	if err != nil {
		return fmt.Errorf("failed to get weekly work summary: %w", err)
	}

	report.WorkWeeks = []*domain.WorkWeek{workWeek}
	report.WorkDays = make([]*domain.WorkDay, len(workWeek.WorkDays))
	for i := range workWeek.WorkDays {
		report.WorkDays[i] = &workWeek.WorkDays[i]
	}

	// Generate activity summary
	if config.IncludeBreakdown {
		weekEnd := weekStart.AddDate(0, 0, 6)
		summary, err := whs.analyzer.GenerateActivitySummary("weekly", weekStart, weekEnd)
		if err != nil {
			whs.logger.Warn("Failed to generate activity summary for weekly report", "error", err)
		} else {
			report.Summary = summary
		}
	}

	// Include trend analysis if requested
	if config.IncludeTrends {
		// Use 4 weeks of data for trend analysis
		trendStart := weekStart.AddDate(0, 0, -21)
		trends, err := whs.analyzer.GetWorkDayTrends(trendStart, weekStart.AddDate(0, 0, 6))
		if err != nil {
			whs.logger.Warn("Failed to analyze trends for weekly report", "error", err)
		} else {
			report.Trends = trends
		}
	}

	return nil
}

func (whs *WorkHourService) generateMonthlyReportContent(
	report *arch.WorkHourReport,
	year int,
	month time.Month,
	config arch.ReportConfig,
) error {
	// Get monthly work data
	workDays, err := whs.GetMonthlyWorkSummary(year, month)
	if err != nil {
		return fmt.Errorf("failed to get monthly work summary: %w", err)
	}

	report.WorkDays = workDays

	// Calculate monthly metrics
	startDate := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, -1)

	// Generate activity summary
	if config.IncludeBreakdown {
		summary, err := whs.analyzer.GenerateActivitySummary("monthly", startDate, endDate)
		if err != nil {
			whs.logger.Warn("Failed to generate activity summary for monthly report", "error", err)
		} else {
			report.Summary = summary
		}
	}

	// Include pattern analysis if requested
	if config.IncludePatterns {
		patterns, err := whs.analyzer.AnalyzeWorkPattern(startDate, endDate)
		if err != nil {
			whs.logger.Warn("Failed to analyze work patterns for monthly report", "error", err)
		} else {
			report.Patterns = patterns
		}
	}

	return nil
}

func (whs *WorkHourService) generateCustomReportContent(
	report *arch.WorkHourReport,
	startDate, endDate time.Time,
	config arch.ReportConfig,
) error {
	// Generate activity summary for custom period
	summary, err := whs.analyzer.GenerateActivitySummary("custom", startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to generate activity summary: %w", err)
	}
	report.Summary = summary

	// Include detailed analysis if requested
	if config.IncludeBreakdown {
		// Get work days for the period
		current := startDate
		for current.Before(endDate) || current.Equal(endDate) {
			workDay, err := whs.analyzer.AnalyzeWorkDay(current)
			if err != nil {
				whs.logger.Warn("Failed to analyze work day for custom report",
					"date", current.Format("2006-01-02"), "error", err)
			} else if workDay.TotalTime > 0 {
				report.WorkDays = append(report.WorkDays, workDay)
			}
			current = current.AddDate(0, 0, 1)
		}
	}

	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-008
 * CONTEXT:   Timesheet operations coordinating with timesheet manager
 * REASON:    Business requirement for timesheet creation and management through service interface
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate timesheet parameters and handle workflow state transitions properly
 * RISK:      High - Timesheet operations are critical for billing and compliance
 */
func (whs *WorkHourService) CreateTimesheet(
	employeeID string,
	period domain.TimesheetPeriod,
	startDate time.Time,
) (*domain.Timesheet, error) {
	
	// Use default policy from configuration
	policy := whs.configuration.DefaultPolicy
	
	timesheet, err := whs.timesheetManager.GenerateTimesheet(employeeID, period, startDate, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate timesheet: %w", err)
	}

	// Save the timesheet
	if err := whs.timesheetManager.SaveTimesheet(timesheet); err != nil {
		return nil, fmt.Errorf("failed to save timesheet: %w", err)
	}

	// Emit event
	whs.emitEvent(WorkHourEvent{
		Type:      EventTimesheetCreated,
		Timestamp: time.Now(),
		Data:      timesheet,
	})

	whs.logger.Info("Timesheet created successfully",
		"timesheetID", timesheet.ID,
		"employeeID", employeeID,
		"period", period,
		"startDate", startDate.Format("2006-01-02"))

	return timesheet, nil
}

func (whs *WorkHourService) FinalizeTimesheet(timesheetID string) error {
	err := whs.timesheetManager.SubmitTimesheet(timesheetID)
	if err != nil {
		return fmt.Errorf("failed to submit timesheet: %w", err)
	}

	// Emit event
	whs.emitEvent(WorkHourEvent{
		Type:      EventTimesheetSubmitted,
		Timestamp: time.Now(),
		Data:      map[string]string{"timesheetID": timesheetID},
	})

	whs.logger.Info("Timesheet finalized successfully", "timesheetID", timesheetID)
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-009
 * CONTEXT:   Analytics and insights operations for business intelligence
 * REASON:    Business requirement for productivity insights and optimization recommendations
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle analytics failures gracefully, provide meaningful insights even with limited data
 * RISK:      Medium - Analytics quality affects business decision making
 */
func (whs *WorkHourService) GetProductivityInsights(startDate, endDate time.Time) (*domain.EfficiencyMetrics, error) {
	metrics, err := whs.analyzer.CalculateProductivityMetrics(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate productivity metrics: %w", err)
	}

	whs.logger.Debug("Productivity insights retrieved",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"activeRatio", metrics.ActiveRatio,
		"focusScore", metrics.FocusScore)

	return metrics, nil
}

func (whs *WorkHourService) GetWorkPatternAnalysis(startDate, endDate time.Time) (*domain.WorkPattern, error) {
	pattern, err := whs.analyzer.AnalyzeWorkPattern(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze work pattern: %w", err)
	}

	whs.logger.Debug("Work pattern analysis retrieved",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"workDayType", pattern.WorkDayType,
		"peakHours", pattern.PeakHours)

	return pattern, nil
}

func (whs *WorkHourService) GetTrendAnalysis(startDate, endDate time.Time) (*domain.TrendAnalysis, error) {
	trends, err := whs.analyzer.GetWorkDayTrends(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze trends: %w", err)
	}

	whs.logger.Debug("Trend analysis retrieved",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"trendDirection", trends.TrendDirection,
		"workTimeChange", trends.WorkTimeChange)

	return trends, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-010
 * CONTEXT:   Configuration and management operations
 * REASON:    Need runtime configuration management and system maintenance operations
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate configuration changes and handle cache refresh failures gracefully
 * RISK:      Low - Configuration management supports core functionality
 */
func (whs *WorkHourService) UpdateWorkHourPolicy(policy domain.TimesheetPolicy) error {
	whs.mu.Lock()
	defer whs.mu.Unlock()
	
	whs.configuration.DefaultPolicy = policy
	
	whs.logger.Info("Work hour policy updated",
		"roundingInterval", policy.RoundingInterval,
		"roundingMethod", policy.RoundingMethod,
		"overtimeThreshold", policy.OvertimeThreshold)
	
	return nil
}

func (whs *WorkHourService) GetWorkHourConfiguration() (*arch.WorkHourConfiguration, error) {
	whs.mu.RLock()
	defer whs.mu.RUnlock()
	
	// Return a copy to prevent external modification
	config := *whs.configuration
	return &config, nil
}

func (whs *WorkHourService) RefreshCache() error {
	// Invalidate database cache
	err := whs.dbManager.InvalidateWorkHourCache(time.Now().AddDate(0, 0, -30), time.Now())
	if err != nil {
		whs.logger.Warn("Failed to invalidate database cache", "error", err)
	}

	// Emit cache invalidation event
	whs.emitEvent(WorkHourEvent{
		Type:      EventCacheInvalidated,
		Timestamp: time.Now(),
		Data:      nil,
	})

	whs.logger.Info("Work hour cache refreshed")
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-SERVICE-011
 * CONTEXT:   Event processing and background maintenance operations
 * REASON:    Need event-driven updates and background maintenance for optimal performance
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle event processing failures gracefully, implement proper goroutine lifecycle
 * RISK:      Medium - Background processing failures could affect service responsiveness
 */
func (whs *WorkHourService) processEvents() {
	whs.logger.Info("Work hour service event processing started")
	
	for {
		select {
		case <-whs.ctx.Done():
			whs.logger.Info("Work hour service event processing stopped")
			return
			
		case event := <-whs.eventChannel:
			whs.handleEvent(event)
		}
	}
}

func (whs *WorkHourService) handleEvent(event WorkHourEvent) {
	whs.logger.Debug("Processing work hour event", "type", event.Type)
	
	// Notify subscribers
	for _, subscriber := range whs.eventSubscribers {
		if err := subscriber.OnWorkHourEvent(event); err != nil {
			whs.logger.Warn("Event subscriber error",
				"eventType", event.Type,
				"error", err)
		}
	}
}

func (whs *WorkHourService) emitEvent(event WorkHourEvent) {
	select {
	case whs.eventChannel <- event:
		// Event sent successfully
	default:
		// Channel full, log warning but don't block
		whs.logger.Warn("Work hour event channel full, dropping event", "type", event.Type)
	}
}

func (whs *WorkHourService) cacheRefreshLoop() {
	ticker := time.NewTicker(whs.configuration.CacheRefreshInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-whs.ctx.Done():
			return
			
		case <-ticker.C:
			if err := whs.RefreshCache(); err != nil {
				whs.logger.Warn("Scheduled cache refresh failed", "error", err)
			}
		}
	}
}

// Subscribe allows components to receive work hour events
func (whs *WorkHourService) Subscribe(subscriber WorkHourEventSubscriber) {
	whs.eventSubscribers = append(whs.eventSubscribers, subscriber)
}

// Export method stub (implementation would depend on export requirements)
func (whs *WorkHourService) ExportReport(report *arch.WorkHourReport, format arch.ExportFormat, outputPath string) error {
	// This would be implemented by a separate exporter component
	whs.logger.Info("Report export requested",
		"reportID", report.ID,
		"format", format,
		"outputPath", outputPath)
	
	return fmt.Errorf("export functionality not yet implemented")
}

// Ensure WorkHourService implements WorkHourService interface
var _ arch.WorkHourService = (*WorkHourService)(nil)