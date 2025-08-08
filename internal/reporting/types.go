/**
 * CONTEXT:   Data structures and types for Claude Monitor reporting system
 * INPUT:     No input - pure type definitions for report data structures
 * OUTPUT:    Comprehensive type definitions for all report formats and analytics
 * BUSINESS:  Type definitions enable strong typing for all reporting components
 * CHANGE:    Extracted from sqlite_reporting_service.go for better organization
 * RISK:      Low - Pure type definitions with JSON serialization tags
 */

package reporting

import (
	"time"
)

/**
 * CONTEXT:   Enhanced daily report structure with comprehensive work tracking data
 * INPUT:     No input - data structure definition
 * OUTPUT:    Daily report structure with work blocks, projects, and insights
 * BUSINESS:  Daily reports are primary interface for work tracking analytics
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type EnhancedDailyReport struct {
	Date                     time.Time             `json:"date"`
	StartTime                time.Time             `json:"start_time"`
	EndTime                  time.Time             `json:"end_time"`
	TotalWorkHours           float64               `json:"total_work_hours"`
	DeepWorkHours            float64               `json:"deep_work_hours"`
	FocusScore               float64               `json:"focus_score"`
	ScheduleHours            float64               `json:"schedule_hours"`
	ClaudeProcessingTime     float64               `json:"claude_processing_time"`
	IdleTime                 float64               `json:"idle_time"`
	EfficiencyPercent        float64               `json:"efficiency_percent"`
	TotalSessions            int                   `json:"total_sessions"`
	ClaudePrompts            int                   `json:"claude_prompts"`
	TotalWorkBlocks          int                   `json:"total_work_blocks"`
	ProjectBreakdown         []ProjectBreakdown    `json:"project_breakdown"`
	HourlyBreakdown          []HourlyData          `json:"hourly_breakdown"`
	WorkBlocks               []WorkBlockSummary    `json:"work_blocks"`
	Insights                 []string              `json:"insights"`
	SessionSummary           SessionSummary        `json:"session_summary"`
	ClaudeActivity           ClaudeActivity        `json:"claude_activity"`
}

/**
 * CONTEXT:   Enhanced weekly report structure with daily breakdown and trends
 * INPUT:     No input - data structure definition
 * OUTPUT:    Weekly report structure with daily patterns and productivity analysis
 * BUSINESS:  Weekly reports show work patterns and consistency over time
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type EnhancedWeeklyReport struct {
	WeekStart           time.Time             `json:"week_start"`
	WeekEnd             time.Time             `json:"week_end"`
	WeekNumber          int                   `json:"week_number"`
	Year                int                   `json:"year"`
	TotalWorkHours      float64               `json:"total_work_hours"`
	DailyAverage        float64               `json:"daily_average"`
	ClaudeUsageHours    float64               `json:"claude_usage_hours"`
	ClaudeUsagePercent  float64               `json:"claude_usage_percent"`
	MostProductiveDay   DaySummary            `json:"most_productive_day"`
	DailyBreakdown      []DaySummary          `json:"daily_breakdown"`
	ProjectBreakdown    []ProjectBreakdown    `json:"project_breakdown"`
	Insights            []WeeklyInsight       `json:"insights"`
	Trends              []Trend               `json:"trends"`
	WeeklyStats         WeeklyStats           `json:"weekly_stats"`
}

/**
 * CONTEXT:   Enhanced monthly report structure with heatmap and achievements
 * INPUT:     No input - data structure definition
 * OUTPUT:    Monthly report structure with daily progress and long-term insights
 * BUSINESS:  Monthly reports provide comprehensive productivity tracking and goals
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type EnhancedMonthlyReport struct {
	Month            time.Time         `json:"month"`
	Year             int               `json:"year"`
	MonthStart       time.Time         `json:"month_start"`
	MonthEnd         time.Time         `json:"month_end"`
	TotalWorkHours   float64           `json:"total_work_hours"`
	WorkingDays      int               `json:"working_days"`
	AverageHoursPerDay float64         `json:"average_hours_per_day"`
	AverageHoursPerWorkingDay float64  `json:"average_hours_per_working_day"`
	LongestWorkStreak int              `json:"longest_work_streak"`
	BestDay          DayData           `json:"best_day"`
	DailyHeatmap     []DayData         `json:"daily_heatmap"`
	ProjectBreakdown []ProjectBreakdown `json:"project_breakdown"`
	Achievements     []Achievement      `json:"achievements"`
	Trends           []Trend           `json:"trends"`
	Insights         []string          `json:"insights"`
}

/**
 * CONTEXT:   Project breakdown structure for time allocation analysis
 * INPUT:     No input - data structure definition
 * OUTPUT:    Project data with hours, percentages, and activity metrics
 * BUSINESS:  Project breakdown shows time distribution across projects
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type ProjectBreakdown struct {
	ProjectName string    `json:"project_name"`
	ProjectPath string    `json:"project_path"`
	WorkHours   float64   `json:"work_hours"`
	Percentage  float64   `json:"percentage"`
	Sessions    int       `json:"sessions"`
}

/**
 * CONTEXT:   Hourly data structure for daily productivity timeline
 * INPUT:     No input - data structure definition
 * OUTPUT:    Hourly work distribution for visualization
 * BUSINESS:  Hourly breakdown reveals productivity patterns throughout the day
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type HourlyData struct {
	Hour  int     `json:"hour"`
	Hours float64 `json:"hours"`
}

/**
 * CONTEXT:   Day summary structure for multi-day reports
 * INPUT:     No input - data structure definition
 * OUTPUT:    Daily summary with hours, sessions, and status
 * BUSINESS:  Day summaries enable weekly and monthly aggregation
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type DaySummary struct {
	Date           time.Time `json:"date"`
	DayName        string    `json:"day_name"`
	Hours          float64   `json:"hours"`
	ClaudeSessions int       `json:"claude_sessions"`
	WorkBlocks     int       `json:"work_blocks"`
	Status         string    `json:"status"`
}

/**
 * CONTEXT:   Day data structure for monthly heatmap visualization
 * INPUT:     No input - data structure definition
 * OUTPUT:    Daily data with hours and intensity level for heatmap
 * BUSINESS:  Day data enables monthly heatmap and progress tracking
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type DayData struct {
	Date  time.Time `json:"date"`
	Hours float64   `json:"hours"`
	Level int       `json:"level"`
}

/**
 * CONTEXT:   Work block summary structure for detailed work tracking
 * INPUT:     No input - data structure definition
 * OUTPUT:    Work block data with timing and project information
 * BUSINESS:  Work blocks are fundamental units of work time tracking
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type WorkBlockSummary struct {
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	ProjectName string        `json:"project_name"`
}

/**
 * CONTEXT:   Session summary structure for session analytics
 * INPUT:     No input - data structure definition
 * OUTPUT:    Session statistics with durations and patterns
 * BUSINESS:  Session summaries provide insights into work session patterns
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type SessionSummary struct {
	TotalSessions   int           `json:"total_sessions"`
	AverageSession  time.Duration `json:"average_session"`
	LongestSession  time.Duration `json:"longest_session"`
	ShortestSession time.Duration `json:"shortest_session"`
	SessionRange    string        `json:"session_range"`
}

/**
 * CONTEXT:   Claude activity structure for AI usage tracking
 * INPUT:     No input - data structure definition
 * OUTPUT:    Claude usage metrics and processing statistics
 * BUSINESS:  Claude activity tracking shows AI tool usage patterns
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type ClaudeActivity struct {
	TotalPrompts      int           `json:"total_prompts"`
	ProcessingTime    time.Duration `json:"processing_time"`
	AverageProcessing time.Duration `json:"average_processing"`
	SuccessfulPrompts int           `json:"successful_prompts"`
	EfficiencyPercent float64       `json:"efficiency_percent"`
}

/**
 * CONTEXT:   Weekly insight structure for pattern recognition
 * INPUT:     No input - data structure definition
 * OUTPUT:    Weekly insights with types, titles, and recommendations
 * BUSINESS:  Weekly insights provide actionable productivity feedback
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type WeeklyInsight struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

/**
 * CONTEXT:   Trend structure for productivity pattern tracking
 * INPUT:     No input - data structure definition
 * OUTPUT:    Trend data with metrics and directional changes
 * BUSINESS:  Trends help identify productivity improvements or declines
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type Trend struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Value       float64 `json:"value"`
}

/**
 * CONTEXT:   Weekly stats structure for comprehensive weekly analysis
 * INPUT:     No input - data structure definition
 * OUTPUT:    Weekly statistics with consistency and productivity metrics
 * BUSINESS:  Weekly stats provide detailed productivity analysis
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type WeeklyStats struct {
	ConsistencyScore  float64 `json:"consistency_score"`
	ProductivityPeak  string  `json:"productivity_peak"`
	WeekendWork       float64 `json:"weekend_work"`
	WeekendPercent    float64 `json:"weekend_percent"`
}

/**
 * CONTEXT:   Achievement structure for milestone tracking
 * INPUT:     No input - data structure definition
 * OUTPUT:    Achievement data with status and descriptions
 * BUSINESS:  Achievements provide motivation and goal tracking
 * CHANGE:    Extracted type definition for better code organization
 * RISK:      Low - Data structure with JSON serialization support
 */
type Achievement struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Achieved    bool   `json:"achieved"`
}