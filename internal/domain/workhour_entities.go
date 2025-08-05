package domain

import (
	"time"

	"github.com/google/uuid"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-002
 * CONTEXT:   Extended domain entities for work hour tracking and analytics
 * REASON:    Need business-oriented entities for labor time tracking beyond raw session/workblock data
 * CHANGE:    New domain entities for comprehensive work hour management.
 * PREVENTION:Keep entities focused on work hour business rules, validate all time calculations
 * RISK:      Low - Domain entities are well-defined by business requirements
 */

// WorkDay represents a calendar day's work activity aggregated from sessions/work blocks
type WorkDay struct {
	ID           string        `json:"id"`
	Date         time.Time     `json:"date"`          // Date (without time component)
	StartTime    *time.Time    `json:"startTime"`     // First activity of the day
	EndTime      *time.Time    `json:"endTime"`       // Last activity of the day
	TotalTime    time.Duration `json:"totalTime"`     // Total active work time
	BreakTime    time.Duration `json:"breakTime"`     // Time between work blocks
	SessionCount int           `json:"sessionCount"`  // Number of sessions in day
	BlockCount   int           `json:"blockCount"`    // Number of work blocks in day
	IsComplete   bool          `json:"isComplete"`    // True if day is finished (past midnight)
}

// NewWorkDay creates a new work day for the given date
func NewWorkDay(date time.Time) *WorkDay {
	// Normalize to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	
	return &WorkDay{
		ID:           uuid.New().String(),
		Date:         normalizedDate,
		TotalTime:    0,
		BreakTime:    0,
		SessionCount: 0,
		BlockCount:   0,
		IsComplete:   time.Now().After(normalizedDate.Add(24 * time.Hour)),
	}
}

// UpdateActivity updates the work day with new activity information
func (wd *WorkDay) UpdateActivity(activityTime time.Time) {
	if wd.StartTime == nil || activityTime.Before(*wd.StartTime) {
		wd.StartTime = &activityTime
	}
	
	if wd.EndTime == nil || activityTime.After(*wd.EndTime) {
		wd.EndTime = &activityTime
	}
}

// GetWorkDuration returns the total span from first to last activity
func (wd *WorkDay) GetWorkDuration() time.Duration {
	if wd.StartTime == nil || wd.EndTime == nil {
		return 0
	}
	return wd.EndTime.Sub(*wd.StartTime)
}

// GetEfficiencyRatio returns the ratio of active time to total time span
func (wd *WorkDay) GetEfficiencyRatio() float64 {
	totalSpan := wd.GetWorkDuration()
	if totalSpan == 0 {
		return 0
	}
	return float64(wd.TotalTime) / float64(totalSpan)
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-003
 * CONTEXT:   Work week aggregation entity for weekly reporting and analytics
 * REASON:    Business requirement for weekly work pattern analysis and overtime tracking
 * CHANGE:    New entity for weekly work hour aggregation and analysis.
 * PREVENTION:Ensure week boundaries are correctly calculated across different locales and DST
 * RISK:      Low - Week calculations are well-defined but need timezone handling
 */

// WorkWeek represents a week's work activity aggregated from work days
type WorkWeek struct {
	ID              string        `json:"id"`
	WeekStart       time.Time     `json:"weekStart"`       // Start of week (Monday)
	WeekEnd         time.Time     `json:"weekEnd"`         // End of week (Sunday)
	TotalTime       time.Duration `json:"totalTime"`       // Total work time for week
	OvertimeHours   time.Duration `json:"overtimeHours"`   // Hours over standard work week
	WorkDays        []WorkDay     `json:"workDays"`        // Individual work days
	AverageDay      time.Duration `json:"averageDay"`      // Average daily work time
	StandardHours   time.Duration `json:"standardHours"`   // Expected hours per week (40h default)
	IsComplete      bool          `json:"isComplete"`      // True if week is finished
}

// NewWorkWeek creates a new work week starting from the given Monday
func NewWorkWeek(weekStart time.Time, standardHours time.Duration) *WorkWeek {
	if standardHours == 0 {
		standardHours = 40 * time.Hour // Default 40-hour work week
	}
	
	weekEnd := weekStart.Add(7 * 24 * time.Hour).Add(-time.Second)
	
	return &WorkWeek{
		ID:            uuid.New().String(),
		WeekStart:     weekStart,
		WeekEnd:       weekEnd,
		StandardHours: standardHours,
		WorkDays:      make([]WorkDay, 0, 7),
		IsComplete:    time.Now().After(weekEnd),
	}
}

// CalculateOvertime determines overtime hours based on standard work week
func (ww *WorkWeek) CalculateOvertime() {
	if ww.TotalTime > ww.StandardHours {
		ww.OvertimeHours = ww.TotalTime - ww.StandardHours
	} else {
		ww.OvertimeHours = 0
	}
}

// GetWorkPattern returns a pattern analysis of the work week
func (ww *WorkWeek) GetWorkPattern() WorkPattern {
	return AnalyzeWorkPattern(ww.WorkDays)
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-004
 * CONTEXT:   Timesheet entity for formal time tracking and export
 * REASON:    Business requirement for professional timesheet export and billing integration
 * CHANGE:    New entity for timesheet-style reporting with configurable policies.
 * PREVENTION:Validate all time entries and ensure rounding policies are correctly applied
 * RISK:      Medium - Timesheet accuracy is critical for billing and compliance
 */

// Timesheet represents a formal timesheet for a specific period
type Timesheet struct {
	ID           string            `json:"id"`
	EmployeeID   string            `json:"employeeId"`    // Future multi-user support
	Period       TimesheetPeriod   `json:"period"`        // Weekly, bi-weekly, monthly
	StartDate    time.Time         `json:"startDate"`
	EndDate      time.Time         `json:"endDate"`
	Entries      []TimesheetEntry  `json:"entries"`       // Individual time entries
	TotalHours   time.Duration     `json:"totalHours"`
	RegularHours time.Duration     `json:"regularHours"`
	OvertimeHours time.Duration    `json:"overtimeHours"`
	Policy       TimesheetPolicy   `json:"policy"`        // Rounding, overtime rules
	Status       TimesheetStatus   `json:"status"`
	CreatedAt    time.Time         `json:"createdAt"`
	SubmittedAt  *time.Time        `json:"submittedAt,omitempty"`
}

// TimesheetEntry represents a single time entry in a timesheet
type TimesheetEntry struct {
	ID          string        `json:"id"`
	Date        time.Time     `json:"date"`
	StartTime   time.Time     `json:"startTime"`
	EndTime     time.Time     `json:"endTime"`
	Duration    time.Duration `json:"duration"`
	Project     string        `json:"project"`      // "Claude CLI Usage"
	Task        string        `json:"task"`         // "Development", "Research", etc.
	Description string        `json:"description"`  // Optional notes
	Billable    bool          `json:"billable"`     // For billing systems
}

// TimesheetPeriod defines the timesheet reporting period
type TimesheetPeriod string

const (
	TimesheetWeekly    TimesheetPeriod = "weekly"
	TimesheetBiweekly  TimesheetPeriod = "biweekly"
	TimesheetMonthly   TimesheetPeriod = "monthly"
	TimesheetCustom    TimesheetPeriod = "custom"
)

// TimesheetStatus defines the timesheet workflow status
type TimesheetStatus string

const (
	TimesheetDraft     TimesheetStatus = "draft"
	TimesheetSubmitted TimesheetStatus = "submitted"
	TimesheetApproved  TimesheetStatus = "approved"
	TimesheetRejected  TimesheetStatus = "rejected"
)

// TimesheetPolicy defines rounding and overtime calculation rules
type TimesheetPolicy struct {
	RoundingInterval  time.Duration `json:"roundingInterval"`  // 15min, 30min, 1h
	RoundingMethod    RoundingMethod `json:"roundingMethod"`   // Up, down, nearest
	OvertimeThreshold time.Duration `json:"overtimeThreshold"` // Daily overtime threshold
	WeeklyThreshold   time.Duration `json:"weeklyThreshold"`   // Weekly overtime threshold
	BreakDeduction    time.Duration `json:"breakDeduction"`    // Automatic break deduction
}

// RoundingMethod defines how time should be rounded
type RoundingMethod string

const (
	RoundUp      RoundingMethod = "up"
	RoundDown    RoundingMethod = "down"
	RoundNearest RoundingMethod = "nearest"
)

// ApplyPolicy applies the timesheet policy to calculate final hours
func (ts *Timesheet) ApplyPolicy() {
	for i := range ts.Entries {
		entry := &ts.Entries[i]
		entry.Duration = ts.Policy.RoundDuration(entry.Duration)
	}
	
	ts.calculateTotals()
}

// RoundDuration applies rounding rules to a duration
func (tp *TimesheetPolicy) RoundDuration(duration time.Duration) time.Duration {
	if tp.RoundingInterval == 0 {
		return duration
	}
	
	switch tp.RoundingMethod {
	case RoundUp:
		return time.Duration((duration + tp.RoundingInterval - 1) / tp.RoundingInterval * tp.RoundingInterval)
	case RoundDown:
		return time.Duration(duration / tp.RoundingInterval * tp.RoundingInterval)
	case RoundNearest:
		halfInterval := tp.RoundingInterval / 2
		return time.Duration((duration + halfInterval) / tp.RoundingInterval * tp.RoundingInterval)
	default:
		return duration
	}
}

// calculateTotals calculates total, regular, and overtime hours
func (ts *Timesheet) calculateTotals() {
	var total time.Duration
	for _, entry := range ts.Entries {
		total += entry.Duration
	}
	
	ts.TotalHours = total
	
	// Calculate overtime based on weekly threshold
	if total > ts.Policy.WeeklyThreshold {
		ts.OvertimeHours = total - ts.Policy.WeeklyThreshold
		ts.RegularHours = ts.Policy.WeeklyThreshold
	} else {
		ts.RegularHours = total
		ts.OvertimeHours = 0
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-005
 * CONTEXT:   Work pattern analysis entities for productivity insights
 * REASON:    Need to identify work patterns, peak productivity times, and optimization opportunities
 * CHANGE:    New entities for work pattern analysis and productivity metrics.
 * PREVENTION:Ensure pattern analysis doesn't expose sensitive personal data inappropriately
 * RISK:      Low - Pattern analysis is statistical and privacy-preserving
 */

// WorkPattern represents analyzed work patterns and productivity insights
type WorkPattern struct {
	PeakHours        []int            `json:"peakHours"`        // Hours of day with most activity (0-23)
	ProductivityCurve []float64       `json:"productivityCurve"` // Hourly productivity scores
	WorkDayType      WorkDayType      `json:"workDayType"`      // Early bird, night owl, etc.
	ConsistencyScore float64          `json:"consistencyScore"` // 0-1 consistency rating
	BreakPatterns    []BreakPattern   `json:"breakPatterns"`    // Common break times and durations
	WeeklyPattern    []time.Duration  `json:"weeklyPattern"`    // Monday-Sunday work durations
}

// WorkDayType categorizes work schedule preferences
type WorkDayType string

const (
	EarlyBird    WorkDayType = "early_bird"    // Peak before 10 AM
	StandardDay  WorkDayType = "standard"      // Peak 10 AM - 4 PM
	NightOwl     WorkDayType = "night_owl"     // Peak after 4 PM
	FlexibleDay  WorkDayType = "flexible"      // Multiple peaks
)

// BreakPattern represents common break timing and duration patterns
type BreakPattern struct {
	StartHour int           `json:"startHour"`  // Hour when break typically starts
	Duration  time.Duration `json:"duration"`   // Typical break duration
	Frequency float64       `json:"frequency"`  // How often this pattern occurs (0-1)
	Type      BreakType     `json:"type"`       // Lunch, coffee, etc.
}

// BreakType categorizes different types of breaks
type BreakType string

const (
	ShortBreak  BreakType = "short"   // < 30 minutes
	LunchBreak  BreakType = "lunch"   // 30-90 minutes
	LongBreak   BreakType = "long"    // > 90 minutes
	EndOfDay    BreakType = "end"     // End of work day
)

// AnalyzeWorkPattern analyzes work days to identify patterns
func AnalyzeWorkPattern(workDays []WorkDay) WorkPattern {
	if len(workDays) == 0 {
		return WorkPattern{}
	}
	
	// This would contain sophisticated pattern analysis logic
	// For now, return a basic structure
	return WorkPattern{
		PeakHours:        []int{9, 10, 14, 15}, // Default business hours
		ProductivityCurve: make([]float64, 24), // Hourly productivity
		WorkDayType:      StandardDay,
		ConsistencyScore: 0.8,
		BreakPatterns:    []BreakPattern{},
		WeeklyPattern:    make([]time.Duration, 7),
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-006
 * CONTEXT:   Activity summary entities for comprehensive reporting
 * REASON:    Need structured summary data for dashboards and executive reporting
 * CHANGE:    New entities for activity summaries and trend analysis.
 * PREVENTION:Cache summary calculations and update incrementally when possible
 * RISK:      Low - Summary data is derived and can be recalculated
 */

// ActivitySummary provides high-level work activity metrics
type ActivitySummary struct {
	Period          string            `json:"period"`          // "daily", "weekly", "monthly"
	StartDate       time.Time         `json:"startDate"`
	EndDate         time.Time         `json:"endDate"`
	TotalWorkTime   time.Duration     `json:"totalWorkTime"`
	TotalSessions   int               `json:"totalSessions"`
	TotalWorkBlocks int               `json:"totalWorkBlocks"`
	AverageSession  time.Duration     `json:"averageSession"`
	AverageWorkBlock time.Duration    `json:"averageWorkBlock"`
	DailyAverage    time.Duration     `json:"dailyAverage"`
	Trends          TrendAnalysis     `json:"trends"`
	Goals           GoalProgress      `json:"goals"`
	Efficiency      EfficiencyMetrics `json:"efficiency"`
}

// TrendAnalysis shows growth/decline trends
type TrendAnalysis struct {
	WorkTimeChange    float64 `json:"workTimeChange"`    // Percentage change from previous period
	SessionChange     float64 `json:"sessionChange"`     // Percentage change in sessions
	EfficiencyChange  float64 `json:"efficiencyChange"`  // Percentage change in efficiency
	TrendDirection    string  `json:"trendDirection"`    // "up", "down", "stable"
}

// GoalProgress tracks progress against configurable work goals
type GoalProgress struct {
	DailyGoal      time.Duration `json:"dailyGoal"`      // Target daily work time
	WeeklyGoal     time.Duration `json:"weeklyGoal"`     // Target weekly work time
	MonthlyGoal    time.Duration `json:"monthlyGoal"`    // Target monthly work time
	DailyProgress  float64       `json:"dailyProgress"`  // 0-1 progress toward daily goal
	WeeklyProgress float64       `json:"weeklyProgress"` // 0-1 progress toward weekly goal
	MonthlyProgress float64      `json:"monthlyProgress"` // 0-1 progress toward monthly goal
	OnTrack        bool          `json:"onTrack"`        // Whether goals are achievable
}

// EfficiencyMetrics provides productivity and efficiency measurements
type EfficiencyMetrics struct {
	ActiveRatio      float64 `json:"activeRatio"`      // Active time / Total time span
	FocusScore       float64 `json:"focusScore"`       // Continuity of work blocks (0-1)
	InterruptionRate float64 `json:"interruptionRate"` // Breaks per hour
	PeakEfficiency   string  `json:"peakEfficiency"`   // Best performing time period
}

// NewActivitySummary creates a new activity summary for the given period
func NewActivitySummary(period string, startDate, endDate time.Time) *ActivitySummary {
	return &ActivitySummary{
		Period:    period,
		StartDate: startDate,
		EndDate:   endDate,
		Trends:    TrendAnalysis{TrendDirection: "stable"},
		Goals:     GoalProgress{DailyGoal: 8 * time.Hour, WeeklyGoal: 40 * time.Hour},
		Efficiency: EfficiencyMetrics{},
	}
}