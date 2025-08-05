/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-001
 * CONTEXT:   Timesheet manager implementing formal time tracking with configurable business rules
 * REASON:    Business requirement for professional timesheet generation with rounding rules and overtime calculation
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate all timesheet policies, ensure rounding rules are correctly applied, handle edge cases
 * RISK:      High - Timesheet accuracy is critical for billing and compliance requirements
 */
package workhour

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
	"github.com/google/uuid"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-002
 * CONTEXT:   TimesheetManager implements formal timesheet generation and management
 * REASON:    Core business logic for converting work hour data into formal timesheets with business rules
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate all timesheet data, implement proper concurrency controls, handle policy changes gracefully
 * RISK:      High - Timesheet calculations must be accurate for billing and compliance
 */
type TimesheetManager struct {
	dbManager        arch.WorkHourDatabaseManager
	workHourAnalyzer arch.WorkHourAnalyzer
	logger           arch.Logger
	defaultPolicy    domain.TimesheetPolicy
	mu               sync.RWMutex
	timesheetCache   map[string]*domain.Timesheet
	validationRules  TimesheetValidationRules
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-003
 * CONTEXT:   TimesheetValidationRules defines business rules for timesheet validation
 * REASON:    Need configurable validation rules to ensure timesheet compliance with business policies
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep validation rules reasonable and configurable for different business requirements
 * RISK:      Medium - Overly strict validation could prevent legitimate timesheet submissions
 */
type TimesheetValidationRules struct {
	MaxDailyHours     time.Duration `json:"maxDailyHours"`     // Maximum allowed daily hours
	MaxWeeklyHours    time.Duration `json:"maxWeeklyHours"`    // Maximum allowed weekly hours
	MinEntryDuration  time.Duration `json:"minEntryDuration"`  // Minimum meaningful time entry
	MaxEntryDuration  time.Duration `json:"maxEntryDuration"`  // Maximum single entry duration
	RequireBreaks     bool          `json:"requireBreaks"`     // Require breaks for long days
	BreakThreshold    time.Duration `json:"breakThreshold"`    // Hours after which breaks are required
	MinBreakDuration  time.Duration `json:"minBreakDuration"`  // Minimum break duration
}

func NewTimesheetManager(
	dbManager arch.WorkHourDatabaseManager,
	analyzer arch.WorkHourAnalyzer,
	logger arch.Logger,
) *TimesheetManager {
	return &TimesheetManager{
		dbManager:        dbManager,
		workHourAnalyzer: analyzer,
		logger:           logger,
		defaultPolicy: domain.TimesheetPolicy{
			RoundingInterval:  15 * time.Minute,
			RoundingMethod:    domain.RoundNearest,
			OvertimeThreshold: 8 * time.Hour,
			WeeklyThreshold:   40 * time.Hour,
			BreakDeduction:    30 * time.Minute,
		},
		timesheetCache: make(map[string]*domain.Timesheet),
		validationRules: TimesheetValidationRules{
			MaxDailyHours:    12 * time.Hour,
			MaxWeeklyHours:   60 * time.Hour,
			MinEntryDuration: 5 * time.Minute,
			MaxEntryDuration: 12 * time.Hour,
			RequireBreaks:    true,
			BreakThreshold:   6 * time.Hour,
			MinBreakDuration: 15 * time.Minute,
		},
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-004
 * CONTEXT:   GenerateTimesheet creates formal timesheet from work hour data with business rule application
 * REASON:    Core business requirement for converting monitoring data into professional timesheet format
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate input parameters, handle edge cases like partial periods, ensure data consistency
 * RISK:      High - Timesheet generation is critical for billing and compliance workflows
 */
func (tm *TimesheetManager) GenerateTimesheet(
	employeeID string,
	period domain.TimesheetPeriod,
	startDate time.Time,
	policy domain.TimesheetPolicy,
) (*domain.Timesheet, error) {
	
	if employeeID == "" {
		employeeID = "default"
	}

	// Calculate end date based on period
	endDate, err := tm.calculatePeriodEndDate(period, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate period end date: %w", err)
	}

	// Generate timesheet ID
	timesheetID := uuid.New().String()

	// Create timesheet structure
	timesheet := &domain.Timesheet{
		ID:         timesheetID,
		EmployeeID: employeeID,
		Period:     period,
		StartDate:  startDate,
		EndDate:    endDate,
		Entries:    []domain.TimesheetEntry{},
		Policy:     policy,
		Status:     domain.TimesheetDraft,
		CreatedAt:  time.Now(),
	}

	// Generate timesheet entries from work hour data
	entries, err := tm.generateTimesheetEntries(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate timesheet entries: %w", err)
	}

	timesheet.Entries = entries

	// Apply timesheet policy (rounding, overtime, etc.)
	if err := tm.ApplyTimesheetPolicy(timesheet); err != nil {
		return nil, fmt.Errorf("failed to apply timesheet policy: %w", err)
	}

	// Validate timesheet
	if err := tm.ValidateTimesheet(timesheet); err != nil {
		return nil, fmt.Errorf("timesheet validation failed: %w", err)
	}

	// Cache the timesheet
	tm.mu.Lock()
	tm.timesheetCache[timesheetID] = timesheet
	tm.mu.Unlock()

	tm.logger.Info("Timesheet generated successfully",
		"timesheetID", timesheetID,
		"employeeID", employeeID,
		"period", period,
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"entries", len(entries),
		"totalHours", timesheet.TotalHours)

	return timesheet, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-005
 * CONTEXT:   generateTimesheetEntries converts work hour data into timesheet entries
 * REASON:    Need to transform work blocks and sessions into standard timesheet entry format
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle overlapping work blocks correctly, ensure proper time allocation
 * RISK:      High - Incorrect entry generation affects timesheet accuracy
 */
func (tm *TimesheetManager) generateTimesheetEntries(startDate, endDate time.Time) ([]domain.TimesheetEntry, error) {
	var entries []domain.TimesheetEntry

	// Get timesheet data from database
	timesheetData, err := tm.dbManager.GetTimesheetData(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get timesheet data: %w", err)
	}

	// Convert to timesheet entries
	for _, data := range timesheetData {
		entry := domain.TimesheetEntry{
			ID:          uuid.New().String(),
			Date:        data.Date,
			StartTime:   data.StartTime,
			EndTime:     data.EndTime,
			Duration:    data.Duration,
			Project:     data.Project,
			Task:        data.Task,
			Description: data.Description,
			Billable:    data.Billable,
		}
		entries = append(entries, entry)
	}

	// If no database entries, generate from work day analysis
	if len(entries) == 0 {
		entries, err = tm.generateEntriesFromWorkDays(startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("failed to generate entries from work days: %w", err)
		}
	}

	// Sort entries by date and start time
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Date.Equal(entries[j].Date) {
			return entries[i].StartTime.Before(entries[j].StartTime)
		}
		return entries[i].Date.Before(entries[j].Date)
	})

	tm.logger.Debug("Timesheet entries generated",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"entryCount", len(entries))

	return entries, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-006
 * CONTEXT:   generateEntriesFromWorkDays creates timesheet entries from work day analysis
 * REASON:    Fallback method to generate entries when explicit timesheet data is not available
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure generated entries are reasonable and don't create artificial precision
 * RISK:      Medium - Generated entries should reflect actual work patterns accurately
 */
func (tm *TimesheetManager) generateEntriesFromWorkDays(startDate, endDate time.Time) ([]domain.TimesheetEntry, error) {
	var entries []domain.TimesheetEntry

	// Iterate through each day in the period
	current := startDate
	for current.Before(endDate) || current.Equal(endDate) {
		workDay, err := tm.workHourAnalyzer.AnalyzeWorkDay(current)
		if err != nil {
			tm.logger.Warn("Failed to analyze work day for timesheet entry generation",
				"date", current.Format("2006-01-02"), "error", err)
			current = current.AddDate(0, 0, 1)
			continue
		}

		// Create entry if there was work activity
		if workDay.TotalTime > 0 && workDay.StartTime != nil && workDay.EndTime != nil {
			entry := domain.TimesheetEntry{
				ID:          uuid.New().String(),
				Date:        current,
				StartTime:   *workDay.StartTime,
				EndTime:     *workDay.EndTime,
				Duration:    workDay.TotalTime,
				Project:     "Claude CLI Usage",
				Task:        "Development",
				Description: fmt.Sprintf("Work session - %d blocks, %d sessions", workDay.BlockCount, workDay.SessionCount),
				Billable:    true,
			}
			entries = append(entries, entry)
		}

		current = current.AddDate(0, 0, 1)
	}

	return entries, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-007
 * CONTEXT:   ApplyTimesheetPolicy implements business rules including rounding and overtime calculations
 * REASON:    Business requirement for applying configurable policies to timesheet entries
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate policy parameters, ensure rounding maintains data integrity
 * RISK:      High - Policy application affects billing calculations and compliance
 */
func (tm *TimesheetManager) ApplyTimesheetPolicy(timesheet *domain.Timesheet) error {
	if timesheet == nil {
		return fmt.Errorf("timesheet cannot be nil")
	}

	// Apply rounding to each entry
	for i := range timesheet.Entries {
		entry := &timesheet.Entries[i]
		originalDuration := entry.Duration
		
		// Apply rounding policy
		entry.Duration = timesheet.Policy.RoundDuration(entry.Duration)
		
		// Log significant rounding changes
		if originalDuration != entry.Duration {
			tm.logger.Debug("Timesheet entry rounded",
				"entryID", entry.ID,
				"originalDuration", originalDuration,
				"roundedDuration", entry.Duration,
				"roundingMethod", timesheet.Policy.RoundingMethod)
		}

		// Apply break deduction for long entries
		if entry.Duration > timesheet.Policy.BreakThreshold && timesheet.Policy.BreakDeduction > 0 {
			entry.Duration -= timesheet.Policy.BreakDeduction
			tm.logger.Debug("Break deduction applied",
				"entryID", entry.ID,
				"breakDeduction", timesheet.Policy.BreakDeduction)
		}
	}

	// Calculate totals
	timesheet.ApplyPolicy()

	tm.logger.Info("Timesheet policy applied",
		"timesheetID", timesheet.ID,
		"totalHours", timesheet.TotalHours,
		"regularHours", timesheet.RegularHours,
		"overtimeHours", timesheet.OvertimeHours,
		"entries", len(timesheet.Entries))

	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-008
 * CONTEXT:   ValidateTimesheet implements comprehensive timesheet validation rules
 * REASON:    Business requirement for ensuring timesheet compliance before submission
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep validation rules reasonable and provide clear error messages
 * RISK:      Medium - Validation must catch errors without being overly restrictive
 */
func (tm *TimesheetManager) ValidateTimesheet(timesheet *domain.Timesheet) error {
	if timesheet == nil {
		return fmt.Errorf("timesheet cannot be nil")
	}

	var validationErrors []string

	// Validate timesheet period
	if timesheet.EndDate.Before(timesheet.StartDate) {
		validationErrors = append(validationErrors, "end date cannot be before start date")
	}

	// Validate total hours
	if timesheet.TotalHours > tm.validationRules.MaxWeeklyHours {
		validationErrors = append(validationErrors,
			fmt.Sprintf("total hours (%v) exceeds maximum weekly hours (%v)",
				timesheet.TotalHours, tm.validationRules.MaxWeeklyHours))
	}

	// Validate individual entries
	dailyTotals := make(map[string]time.Duration)
	
	for _, entry := range timesheet.Entries {
		// Validate entry duration
		if entry.Duration < tm.validationRules.MinEntryDuration {
			validationErrors = append(validationErrors,
				fmt.Sprintf("entry %s duration (%v) below minimum (%v)",
					entry.ID, entry.Duration, tm.validationRules.MinEntryDuration))
		}

		if entry.Duration > tm.validationRules.MaxEntryDuration {
			validationErrors = append(validationErrors,
				fmt.Sprintf("entry %s duration (%v) exceeds maximum (%v)",
					entry.ID, entry.Duration, tm.validationRules.MaxEntryDuration))
		}

		// Validate entry times
		if entry.EndTime.Before(entry.StartTime) {
			validationErrors = append(validationErrors,
				fmt.Sprintf("entry %s end time before start time", entry.ID))
		}

		// Track daily totals
		dateKey := entry.Date.Format("2006-01-02")
		dailyTotals[dateKey] += entry.Duration
	}

	// Validate daily hours
	for dateKey, dailyTotal := range dailyTotals {
		if dailyTotal > tm.validationRules.MaxDailyHours {
			validationErrors = append(validationErrors,
				fmt.Sprintf("daily total for %s (%v) exceeds maximum (%v)",
					dateKey, dailyTotal, tm.validationRules.MaxDailyHours))
		}

		// Check break requirements for long days
		if tm.validationRules.RequireBreaks && dailyTotal > tm.validationRules.BreakThreshold {
			if !tm.hasAdequateBreaks(timesheet.Entries, dateKey) {
				validationErrors = append(validationErrors,
					fmt.Sprintf("day %s requires breaks for %v+ hours", dateKey, tm.validationRules.BreakThreshold))
			}
		}
	}

	// Return validation errors if any
	if len(validationErrors) > 0 {
		return fmt.Errorf("timesheet validation failed: %v", validationErrors)
	}

	tm.logger.Debug("Timesheet validation passed", "timesheetID", timesheet.ID)
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-009
 * CONTEXT:   Helper method to check if adequate breaks are present for long work days
 * REASON:    Business compliance requirement for break validation in long work periods
 * CHANGE:    Initial implementation.
 * PREVENTION:Use reasonable break detection logic that accounts for real work patterns
 * RISK:      Low - Break validation is advisory and doesn't block core functionality
 */
func (tm *TimesheetManager) hasAdequateBreaks(entries []domain.TimesheetEntry, dateKey string) bool {
	// Find entries for the specific date
	var dayEntries []domain.TimesheetEntry
	for _, entry := range entries {
		if entry.Date.Format("2006-01-02") == dateKey {
			dayEntries = append(dayEntries, entry)
		}
	}

	if len(dayEntries) < 2 {
		// Single entry - no breaks detected
		return false
	}

	// Sort entries by start time
	sort.Slice(dayEntries, func(i, j int) bool {
		return dayEntries[i].StartTime.Before(dayEntries[j].StartTime)
	})

	// Check for gaps between entries that could be breaks
	for i := 1; i < len(dayEntries); i++ {
		prevEnd := dayEntries[i-1].EndTime
		currentStart := dayEntries[i].StartTime
		
		if currentStart.After(prevEnd) {
			breakDuration := currentStart.Sub(prevEnd)
			if breakDuration >= tm.validationRules.MinBreakDuration {
				return true
			}
		}
	}

	return false
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-010
 * CONTEXT:   Timesheet persistence and retrieval methods
 * REASON:    Need to save and retrieve timesheets for workflow management and audit trails
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle database errors gracefully, maintain cache consistency
 * RISK:      Medium - Persistence failures could cause data loss or workflow disruption
 */
func (tm *TimesheetManager) SaveTimesheet(timesheet *domain.Timesheet) error {
	if timesheet == nil {
		return fmt.Errorf("timesheet cannot be nil")
	}

	// Save to database
	err := tm.dbManager.SaveTimesheet(timesheet)
	if err != nil {
		return fmt.Errorf("failed to save timesheet to database: %w", err)
	}

	// Update cache
	tm.mu.Lock()
	tm.timesheetCache[timesheet.ID] = timesheet
	tm.mu.Unlock()

	tm.logger.Info("Timesheet saved successfully",
		"timesheetID", timesheet.ID,
		"employeeID", timesheet.EmployeeID,
		"status", timesheet.Status)

	return nil
}

func (tm *TimesheetManager) GetTimesheet(timesheetID string) (*domain.Timesheet, error) {
	// Check cache first
	tm.mu.RLock()
	if cached, exists := tm.timesheetCache[timesheetID]; exists {
		tm.mu.RUnlock()
		return cached, nil
	}
	tm.mu.RUnlock()

	// Load from database
	timesheet, err := tm.dbManager.GetTimesheet(timesheetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get timesheet from database: %w", err)
	}

	// Update cache
	tm.mu.Lock()
	tm.timesheetCache[timesheetID] = timesheet
	tm.mu.Unlock()

	return timesheet, nil
}

func (tm *TimesheetManager) GetTimesheetsByPeriod(employeeID string, startDate, endDate time.Time) ([]*domain.Timesheet, error) {
	// For now, implement a simple approach
	// In a production system, this would be optimized with database queries
	
	timesheets, err := tm.dbManager.GetTimesheetsByPeriod(employeeID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get timesheets by period: %w", err)
	}

	tm.logger.Debug("Retrieved timesheets by period",
		"employeeID", employeeID,
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"count", len(timesheets))

	return timesheets, nil
}

func (tm *TimesheetManager) SubmitTimesheet(timesheetID string) error {
	timesheet, err := tm.GetTimesheet(timesheetID)
	if err != nil {
		return fmt.Errorf("failed to get timesheet for submission: %w", err)
	}

	// Final validation before submission
	if err := tm.ValidateTimesheet(timesheet); err != nil {
		return fmt.Errorf("timesheet validation failed before submission: %w", err)
	}

	// Update status
	timesheet.Status = domain.TimesheetSubmitted
	now := time.Now()
	timesheet.SubmittedAt = &now

	// Save updated timesheet
	if err := tm.SaveTimesheet(timesheet); err != nil {
		return fmt.Errorf("failed to save submitted timesheet: %w", err)
	}

	tm.logger.Info("Timesheet submitted successfully",
		"timesheetID", timesheetID,
		"submittedAt", now)

	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-TIMESHEET-011
 * CONTEXT:   Helper methods for timesheet period calculations and utilities
 * REASON:    Need utility functions for period calculations and timesheet management
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle edge cases in date calculations, validate period types
 * RISK:      Low - Utility functions support main timesheet operations
 */
func (tm *TimesheetManager) calculatePeriodEndDate(period domain.TimesheetPeriod, startDate time.Time) (time.Time, error) {
	switch period {
	case domain.TimesheetWeekly:
		return startDate.AddDate(0, 0, 6), nil
	case domain.TimesheetBiweekly:
		return startDate.AddDate(0, 0, 13), nil
	case domain.TimesheetMonthly:
		return startDate.AddDate(0, 1, -1), nil
	case domain.TimesheetCustom:
		// For custom periods, caller should specify end date separately
		return startDate.AddDate(0, 0, 6), nil // Default to weekly
	default:
		return time.Time{}, fmt.Errorf("unsupported timesheet period: %s", period)
	}
}

// SetValidationRules allows runtime configuration of validation rules
func (tm *TimesheetManager) SetValidationRules(rules TimesheetValidationRules) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.validationRules = rules
	tm.logger.Info("Timesheet validation rules updated",
		"maxDailyHours", rules.MaxDailyHours,
		"maxWeeklyHours", rules.MaxWeeklyHours,
		"requireBreaks", rules.RequireBreaks)
}

// GetValidationRules returns current validation rules
func (tm *TimesheetManager) GetValidationRules() TimesheetValidationRules {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	return tm.validationRules
}

// Ensure TimesheetManager implements TimesheetManager interface
var _ arch.TimesheetManager = (*TimesheetManager)(nil)