/**
 * CONTEXT:   Work time calculator for precise time tracking and reporting
 * INPUT:     Work activities, sessions, and work blocks from activity generator
 * OUTPUT:    Calculated work times, session reports, and productivity analytics
 * BUSINESS:  Time calculation enables accurate work hour tracking vs schedule analysis
 * CHANGE:    Initial work time calculator for three-tier time tracking system
 * RISK:      High - Core component for accurate work time calculation and reporting
 */

package reporting

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/your-org/claude-monitor/internal/generator"
)

/**
 * CONTEXT:   Time tracking report types
 * INPUT:     Report type classification requirements
 * OUTPUT:    Structured report type definitions
 * BUSINESS:  Report types enable different perspectives on work time
 * CHANGE:    Initial report type definitions for comprehensive time tracking
 * RISK:      Low - Report type constants
 */
type ReportType string

const (
	ReportActiveWork  ReportType = "ACTIVE_WORK"   // Actual work time (excludes idle)
	ReportSessions    ReportType = "SESSIONS"      // 5-hour Claude usage windows
	ReportWorkDay     ReportType = "WORK_DAY"      // Full work day analysis
	ReportProductivity ReportType = "PRODUCTIVITY" // Work efficiency analysis
)

/**
 * CONTEXT:   Work time calculator for comprehensive time tracking
 * INPUT:     Activities, sessions, and work blocks from activity generator
 * OUTPUT:    Calculated work times and detailed reports
 * BUSINESS:  Calculator provides three-tier time tracking analysis
 * CHANGE:    Initial work time calculator with multi-dimensional analysis
 * RISK:      High - Core time calculation affecting accuracy of work tracking
 */
type WorkTimeCalculator struct {
	mu                sync.RWMutex
	activities        []generator.WorkActivity
	sessions          []generator.WorkSession
	workBlocks        []generator.WorkBlock
	calculationCache  map[string]interface{}
	lastCalculation   time.Time
	cacheExpiry       time.Duration
}

/**
 * CONTEXT:   Comprehensive work time report
 * INPUT:     Calculated work time data
 * OUTPUT:    Complete work time analysis report
 * BUSINESS:  Report provides comprehensive work time insights
 * CHANGE:    Initial work time report structure
 * RISK:      Low - Report data structure for work time analysis
 */
type WorkTimeReport struct {
	ReportType        ReportType               `json:"report_type"`
	GeneratedAt       time.Time                `json:"generated_at"`
	TimeRange         TimeRange                `json:"time_range"`
	
	// Core Time Metrics (3 types)
	ActiveWorkTime    time.Duration            `json:"active_work_time"`    // Real work time (excludes idle)
	SessionTime       time.Duration            `json:"session_time"`        // Claude session time (5-hour windows)
	WorkDayTime       time.Duration            `json:"work_day_time"`       // Total work day time
	
	// Detailed Breakdowns
	ProjectBreakdown  []ProjectTimeBreakdown   `json:"project_breakdown"`
	ActivityBreakdown []ActivityTimeBreakdown  `json:"activity_breakdown"`
	SessionBreakdown  []SessionTimeBreakdown   `json:"session_breakdown"`
	
	// Productivity Metrics
	ProductivityScore float64                  `json:"productivity_score"`  // 0.0 to 1.0
	EfficiencyRatio   float64                  `json:"efficiency_ratio"`    // Active work / Session time
	FocusScore        float64                  `json:"focus_score"`         // Continuous work periods
	
	// Comparative Analysis
	Comparisons       WorkTimeComparisons      `json:"comparisons"`
	
	// Statistics
	TotalActivities   int                      `json:"total_activities"`
	TotalSessions     int                      `json:"total_sessions"`
	TotalWorkBlocks   int                      `json:"total_work_blocks"`
	AverageBlockDuration time.Duration         `json:"average_block_duration"`
	
	// Insights
	Insights          []WorkTimeInsight        `json:"insights"`
}

/**
 * CONTEXT:   Time range for report filtering
 * INPUT:     Report time boundaries
 * OUTPUT:    Time range specification
 * BUSINESS:  Time range enables period-specific analysis
 * CHANGE:    Initial time range structure
 * RISK:      Low - Time range data structure
 */
type TimeRange struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Duration    time.Duration `json:"duration"`
	Description string    `json:"description"`
}

/**
 * CONTEXT:   Project-specific time breakdown
 * INPUT:     Per-project time calculation
 * OUTPUT:    Project time analysis
 * BUSINESS:  Project breakdown enables project-specific productivity analysis
 * CHANGE:    Initial project time breakdown
 * RISK:      Low - Project time breakdown structure
 */
type ProjectTimeBreakdown struct {
	ProjectName     string        `json:"project_name"`
	ProjectPath     string        `json:"project_path"`
	ActiveWorkTime  time.Duration `json:"active_work_time"`
	SessionTime     time.Duration `json:"session_time"`
	WorkBlocks      int           `json:"work_blocks"`
	Activities      int           `json:"activities"`
	ClaudeAPICalls  int           `json:"claude_api_calls"`
	FileOperations  int           `json:"file_operations"`
	LastActivity    time.Time     `json:"last_activity"`
	ProductivityScore float64     `json:"productivity_score"`
}

/**
 * CONTEXT:   Activity type time breakdown
 * INPUT:     Per-activity-type time calculation
 * OUTPUT:    Activity type time analysis
 * BUSINESS:  Activity breakdown enables work type analysis
 * CHANGE:    Initial activity type breakdown
 * RISK:      Low - Activity time breakdown structure
 */
type ActivityTimeBreakdown struct {
	ActivityType   generator.ActivityType `json:"activity_type"`
	Count          int                    `json:"count"`
	TotalTime      time.Duration          `json:"total_time"`
	AverageTime    time.Duration          `json:"average_time"`
	Percentage     float64                `json:"percentage"`
	Description    string                 `json:"description"`
}

/**
 * CONTEXT:   Session time breakdown
 * INPUT:     Per-session time calculation
 * OUTPUT:    Session time analysis
 * BUSINESS:  Session breakdown enables 5-hour window analysis
 * CHANGE:    Initial session time breakdown
 * RISK:      Low - Session time breakdown structure
 */
type SessionTimeBreakdown struct {
	SessionID       string        `json:"session_id"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	TotalDuration   time.Duration `json:"total_duration"`   // Always 5 hours
	ActiveDuration  time.Duration `json:"active_duration"`  // Actual work time
	IdleDuration    time.Duration `json:"idle_duration"`    // Idle time within session
	WorkBlocks      int           `json:"work_blocks"`
	Activities      int           `json:"activities"`
	ProjectCount    int           `json:"project_count"`
	EfficiencyRatio float64       `json:"efficiency_ratio"` // Active / Total
}

/**
 * CONTEXT:   Work time comparisons
 * INPUT:     Comparative time analysis
 * OUTPUT:    Time comparison metrics
 * BUSINESS:  Comparisons enable productivity trend analysis
 * CHANGE:    Initial work time comparisons
 * RISK:      Low - Time comparison structure
 */
type WorkTimeComparisons struct {
	PreviousPeriod     WorkTimeSummary `json:"previous_period"`
	PercentageChange   float64         `json:"percentage_change"`
	Trend              string          `json:"trend"`              // "improving", "declining", "stable"
	DailyAverage       time.Duration   `json:"daily_average"`
	WeeklyAverage      time.Duration   `json:"weekly_average"`
	BestDay            DayPerformance  `json:"best_day"`
	WorstDay           DayPerformance  `json:"worst_day"`
}

/**
 * CONTEXT:   Work time summary
 * INPUT:     Summarized time data
 * OUTPUT:    Condensed time metrics
 * BUSINESS:  Summary enables quick time overview
 * CHANGE:    Initial work time summary
 * RISK:      Low - Time summary structure
 */
type WorkTimeSummary struct {
	ActiveWorkTime time.Duration `json:"active_work_time"`
	SessionTime    time.Duration `json:"session_time"`
	WorkDayTime    time.Duration `json:"work_day_time"`
	Activities     int           `json:"activities"`
	Sessions       int           `json:"sessions"`
}

/**
 * CONTEXT:   Daily performance metrics
 * INPUT:     Day-specific performance data
 * OUTPUT:    Daily performance analysis
 * BUSINESS:  Daily metrics enable day-to-day performance tracking
 * CHANGE:    Initial daily performance structure
 * RISK:      Low - Daily performance structure
 */
type DayPerformance struct {
	Date           time.Time     `json:"date"`
	ActiveWorkTime time.Duration `json:"active_work_time"`
	ProductivityScore float64    `json:"productivity_score"`
	Activities     int           `json:"activities"`
	Sessions       int           `json:"sessions"`
}

/**
 * CONTEXT:   Work time insight
 * INPUT:     Analytical insight data
 * OUTPUT:    Actionable work time insight
 * BUSINESS:  Insights provide actionable work time analysis
 * CHANGE:    Initial work time insight structure
 * RISK:      Low - Insight structure for work time analysis
 */
type WorkTimeInsight struct {
	Type        string                 `json:"type"`        // "efficiency", "pattern", "recommendation"
	Category    string                 `json:"category"`    // "positive", "warning", "neutral"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`      // "high", "medium", "low"
	Suggestion  string                 `json:"suggestion"`
	Data        map[string]interface{} `json:"data"`
}

/**
 * CONTEXT:   Create new work time calculator
 * INPUT:     Calculator configuration
 * OUTPUT:    Configured work time calculator
 * BUSINESS:  Calculator creation enables comprehensive time tracking
 * CHANGE:    Initial work time calculator constructor
 * RISK:      Medium - Calculator initialization for time tracking
 */
func NewWorkTimeCalculator() *WorkTimeCalculator {
	return &WorkTimeCalculator{
		activities:       make([]generator.WorkActivity, 0),
		sessions:         make([]generator.WorkSession, 0),
		workBlocks:       make([]generator.WorkBlock, 0),
		calculationCache: make(map[string]interface{}),
		cacheExpiry:      5 * time.Minute,
	}
}

/**
 * CONTEXT:   Update calculator with new activities
 * INPUT:     Work activities from activity generator
 * OUTPUT:    Updated calculator state
 * BUSINESS:  Activity updates enable real-time time calculation
 * CHANGE:    Initial activity update for time calculator
 * RISK:      Medium - Activity update accuracy affecting calculations
 */
func (wtc *WorkTimeCalculator) UpdateActivities(activities []generator.WorkActivity) {
	wtc.mu.Lock()
	defer wtc.mu.Unlock()
	
	wtc.activities = activities
	wtc.invalidateCache()
}

/**
 * CONTEXT:   Update calculator with new sessions
 * INPUT:     Work sessions from activity generator
 * OUTPUT:    Updated calculator state
 * BUSINESS:  Session updates enable session-based time calculation
 * CHANGE:    Initial session update for time calculator
 * RISK:      Medium - Session update accuracy affecting calculations
 */
func (wtc *WorkTimeCalculator) UpdateSessions(sessions []generator.WorkSession) {
	wtc.mu.Lock()
	defer wtc.mu.Unlock()
	
	wtc.sessions = sessions
	wtc.invalidateCache()
}

/**
 * CONTEXT:   Update calculator with new work blocks
 * INPUT:     Work blocks from activity generator
 * OUTPUT:    Updated calculator state
 * BUSINESS:  Work block updates enable precise time calculation
 * CHANGE:    Initial work block update for time calculator
 * RISK:      High - Work block accuracy affecting precise time calculations
 */
func (wtc *WorkTimeCalculator) UpdateWorkBlocks(workBlocks []generator.WorkBlock) {
	wtc.mu.Lock()
	defer wtc.mu.Unlock()
	
	wtc.workBlocks = workBlocks
	wtc.invalidateCache()
}

/**
 * CONTEXT:   Generate comprehensive work time report
 * INPUT:     Report type and time range
 * OUTPUT:    Complete work time analysis report
 * BUSINESS:  Report generation provides comprehensive work time insights
 * CHANGE:    Initial work time report generation
 * RISK:      Medium - Report accuracy affecting work time analysis
 */
func (wtc *WorkTimeCalculator) GenerateReport(reportType ReportType, timeRange TimeRange) *WorkTimeReport {
	wtc.mu.Lock()
	defer wtc.mu.Unlock()
	
	// Filter data by time range
	filteredActivities := wtc.filterActivitiesByTimeRange(timeRange)
	filteredSessions := wtc.filterSessionsByTimeRange(timeRange)
	filteredWorkBlocks := wtc.filterWorkBlocksByTimeRange(timeRange)
	
	// Calculate core time metrics
	activeWorkTime := wtc.calculateActiveWorkTime(filteredWorkBlocks)
	sessionTime := wtc.calculateSessionTime(filteredSessions)
	workDayTime := wtc.calculateWorkDayTime(timeRange, filteredActivities)
	
	// Generate breakdowns
	projectBreakdown := wtc.generateProjectBreakdown(filteredActivities, filteredWorkBlocks)
	activityBreakdown := wtc.generateActivityBreakdown(filteredActivities)
	sessionBreakdown := wtc.generateSessionBreakdown(filteredSessions)
	
	// Calculate productivity metrics
	productivityScore := wtc.calculateProductivityScore(activeWorkTime, sessionTime, len(filteredActivities))
	efficiencyRatio := wtc.calculateEfficiencyRatio(activeWorkTime, sessionTime)
	focusScore := wtc.calculateFocusScore(filteredWorkBlocks)
	
	// Generate comparisons
	comparisons := wtc.generateComparisons(timeRange, activeWorkTime)
	
	// Calculate statistics
	var totalBlockDuration time.Duration
	for _, block := range filteredWorkBlocks {
		totalBlockDuration += block.Duration
	}
	avgBlockDuration := time.Duration(0)
	if len(filteredWorkBlocks) > 0 {
		avgBlockDuration = totalBlockDuration / time.Duration(len(filteredWorkBlocks))
	}
	
	// Generate insights
	insights := wtc.generateInsights(activeWorkTime, sessionTime, efficiencyRatio, focusScore)
	
	report := &WorkTimeReport{
		ReportType:           reportType,
		GeneratedAt:          time.Now(),
		TimeRange:            timeRange,
		ActiveWorkTime:       activeWorkTime,
		SessionTime:          sessionTime,
		WorkDayTime:          workDayTime,
		ProjectBreakdown:     projectBreakdown,
		ActivityBreakdown:    activityBreakdown,
		SessionBreakdown:     sessionBreakdown,
		ProductivityScore:    productivityScore,
		EfficiencyRatio:      efficiencyRatio,
		FocusScore:           focusScore,
		Comparisons:          comparisons,
		TotalActivities:      len(filteredActivities),
		TotalSessions:        len(filteredSessions),
		TotalWorkBlocks:      len(filteredWorkBlocks),
		AverageBlockDuration: avgBlockDuration,
		Insights:             insights,
	}
	
	return report
}

/**
 * CONTEXT:   Calculate active work time (excludes idle periods)
 * INPUT:     Filtered work blocks
 * OUTPUT:    Total active work time
 * BUSINESS:  Active work time represents actual productive time
 * CHANGE:    Initial active work time calculation
 * RISK:      High - Active work time accuracy affecting productivity metrics
 */
func (wtc *WorkTimeCalculator) calculateActiveWorkTime(workBlocks []generator.WorkBlock) time.Duration {
	var totalActiveTime time.Duration
	
	for _, block := range workBlocks {
		if !block.IsActive { // Only count completed blocks
			totalActiveTime += block.Duration
		}
	}
	
	return totalActiveTime
}

/**
 * CONTEXT:   Calculate session time (5-hour Claude windows)
 * INPUT:     Filtered sessions
 * OUTPUT:    Total session time
 * BUSINESS:  Session time represents Claude usage windows
 * CHANGE:    Initial session time calculation
 * RISK:      Medium - Session time accuracy affecting usage analysis
 */
func (wtc *WorkTimeCalculator) calculateSessionTime(sessions []generator.WorkSession) time.Duration {
	var totalSessionTime time.Duration
	
	for _, session := range sessions {
		if !session.IsActive { // Only count completed sessions
			totalSessionTime += session.ActiveTime // Use active time within session
		}
	}
	
	return totalSessionTime
}

/**
 * CONTEXT:   Calculate work day time (total time from first to last activity)
 * INPUT:     Time range and activities
 * OUTPUT:    Total work day duration
 * BUSINESS:  Work day time represents total work schedule time
 * CHANGE:    Initial work day time calculation
 * RISK:      Medium - Work day time accuracy affecting schedule analysis
 */
func (wtc *WorkTimeCalculator) calculateWorkDayTime(timeRange TimeRange, activities []generator.WorkActivity) time.Duration {
	if len(activities) == 0 {
		return 0
	}
	
	// Find first and last activity
	var firstActivity, lastActivity time.Time
	for i, activity := range activities {
		if i == 0 {
			firstActivity = activity.Timestamp
			lastActivity = activity.Timestamp
		} else {
			if activity.Timestamp.Before(firstActivity) {
				firstActivity = activity.Timestamp
			}
			if activity.Timestamp.After(lastActivity) {
				lastActivity = activity.Timestamp
			}
		}
	}
	
	return lastActivity.Sub(firstActivity)
}

/**
 * CONTEXT:   Generate project time breakdown
 * INPUT:     Activities and work blocks filtered by project
 * OUTPUT:    Per-project time analysis
 * BUSINESS:  Project breakdown enables project-specific productivity analysis
 * CHANGE:    Initial project breakdown generation
 * RISK:      Medium - Project breakdown accuracy affecting project analysis
 */
func (wtc *WorkTimeCalculator) generateProjectBreakdown(activities []generator.WorkActivity, workBlocks []generator.WorkBlock) []ProjectTimeBreakdown {
	projectData := make(map[string]*ProjectTimeBreakdown)
	
	// Process work blocks for active work time
	for _, block := range workBlocks {
		projectName := block.ProjectName
		if projectName == "" {
			projectName = "unknown"
		}
		
		if _, exists := projectData[projectName]; !exists {
			projectData[projectName] = &ProjectTimeBreakdown{
				ProjectName: projectName,
				ProjectPath: block.ProjectPath,
			}
		}
		
		if !block.IsActive { // Only count completed blocks
			projectData[projectName].ActiveWorkTime += block.Duration
		}
		projectData[projectName].WorkBlocks++
	}
	
	// Process activities for additional metrics
	for _, activity := range activities {
		projectName := activity.ProjectName
		if projectName == "" {
			projectName = "unknown"
		}
		
		if _, exists := projectData[projectName]; !exists {
			projectData[projectName] = &ProjectTimeBreakdown{
				ProjectName: projectName,
				ProjectPath: activity.ProjectPath,
			}
		}
		
		data := projectData[projectName]
		data.Activities++
		
		if activity.Timestamp.After(data.LastActivity) {
			data.LastActivity = activity.Timestamp
		}
		
		switch activity.Type {
		case generator.ActivityClaudeAPI:
			data.ClaudeAPICalls++
		case generator.ActivityFileWork, generator.ActivityCodeWork, generator.ActivityDocWork:
			data.FileOperations++
		}
	}
	
	// Calculate productivity scores and convert to slice
	var breakdown []ProjectTimeBreakdown
	for _, data := range projectData {
		data.ProductivityScore = wtc.calculateProjectProductivityScore(data)
		breakdown = append(breakdown, *data)
	}
	
	// Sort by active work time
	sort.Slice(breakdown, func(i, j int) bool {
		return breakdown[i].ActiveWorkTime > breakdown[j].ActiveWorkTime
	})
	
	return breakdown
}

/**
 * CONTEXT:   Generate activity type breakdown
 * INPUT:     Activities grouped by type
 * OUTPUT:    Per-activity-type time analysis
 * BUSINESS:  Activity breakdown enables work type analysis
 * CHANGE:    Initial activity breakdown generation
 * RISK:      Low - Activity breakdown for work type analysis
 */
func (wtc *WorkTimeCalculator) generateActivityBreakdown(activities []generator.WorkActivity) []ActivityTimeBreakdown {
	activityData := make(map[generator.ActivityType]*ActivityTimeBreakdown)
	totalTime := time.Duration(0)
	
	for _, activity := range activities {
		if _, exists := activityData[activity.Type]; !exists {
			activityData[activity.Type] = &ActivityTimeBreakdown{
				ActivityType: activity.Type,
				Description:  wtc.getActivityTypeDescription(activity.Type),
			}
		}
		
		data := activityData[activity.Type]
		data.Count++
		data.TotalTime += activity.Duration
		totalTime += activity.Duration
	}
	
	// Calculate averages and percentages
	var breakdown []ActivityTimeBreakdown
	for _, data := range activityData {
		if data.Count > 0 {
			data.AverageTime = data.TotalTime / time.Duration(data.Count)
		}
		if totalTime > 0 {
			data.Percentage = float64(data.TotalTime) / float64(totalTime) * 100
		}
		breakdown = append(breakdown, *data)
	}
	
	// Sort by total time
	sort.Slice(breakdown, func(i, j int) bool {
		return breakdown[i].TotalTime > breakdown[j].TotalTime
	})
	
	return breakdown
}

/**
 * CONTEXT:   Generate session breakdown
 * INPUT:     Sessions with timing data
 * OUTPUT:    Per-session time analysis
 * BUSINESS:  Session breakdown enables 5-hour window analysis
 * CHANGE:    Initial session breakdown generation
 * RISK:      Low - Session breakdown for window analysis
 */
func (wtc *WorkTimeCalculator) generateSessionBreakdown(sessions []generator.WorkSession) []SessionTimeBreakdown {
	var breakdown []SessionTimeBreakdown
	
	for _, session := range sessions {
		idleDuration := session.Duration - session.ActiveTime
		efficiencyRatio := 0.0
		if session.Duration > 0 {
			efficiencyRatio = float64(session.ActiveTime) / float64(session.Duration)
		}
		
		sessionBreakdown := SessionTimeBreakdown{
			SessionID:       session.ID,
			StartTime:       session.StartTime,
			EndTime:         session.EndTime,
			TotalDuration:   session.Duration,
			ActiveDuration:  session.ActiveTime,
			IdleDuration:    idleDuration,
			WorkBlocks:      len(session.WorkBlocks),
			Activities:      len(session.Activities),
			ProjectCount:    len(session.ProjectStats),
			EfficiencyRatio: efficiencyRatio,
		}
		
		breakdown = append(breakdown, sessionBreakdown)
	}
	
	// Sort by start time
	sort.Slice(breakdown, func(i, j int) bool {
		return breakdown[i].StartTime.Before(breakdown[j].StartTime)
	})
	
	return breakdown
}

// Helper methods and calculations

func (wtc *WorkTimeCalculator) calculateProductivityScore(activeTime, sessionTime time.Duration, activities int) float64 {
	score := 0.0
	
	// Active time component (50% weight)
	if sessionTime > 0 {
		score += 0.5 * (float64(activeTime) / float64(sessionTime))
	}
	
	// Activity density component (30% weight)
	if activeTime > 0 {
		activitiesPerHour := float64(activities) / (float64(activeTime) / float64(time.Hour))
		score += 0.3 * (activitiesPerHour / 10.0) // Normalize to activities per hour
	}
	
	// Consistency component (20% weight)
	score += 0.2 * 0.8 // Base consistency score
	
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

func (wtc *WorkTimeCalculator) calculateEfficiencyRatio(activeTime, sessionTime time.Duration) float64 {
	if sessionTime == 0 {
		return 0.0
	}
	return float64(activeTime) / float64(sessionTime)
}

func (wtc *WorkTimeCalculator) calculateFocusScore(workBlocks []generator.WorkBlock) float64 {
	if len(workBlocks) == 0 {
		return 0.0
	}
	
	// Calculate average work block duration
	var totalDuration time.Duration
	for _, block := range workBlocks {
		totalDuration += block.Duration
	}
	avgDuration := totalDuration / time.Duration(len(workBlocks))
	
	// Longer average blocks indicate better focus
	// Score from 0.0 to 1.0, where 1.0 = 1 hour average blocks
	focusScore := float64(avgDuration) / float64(time.Hour)
	if focusScore > 1.0 {
		focusScore = 1.0
	}
	
	return focusScore
}

func (wtc *WorkTimeCalculator) calculateProjectProductivityScore(project *ProjectTimeBreakdown) float64 {
	score := 0.0
	
	// Active work time (40% weight)
	if project.ActiveWorkTime > 0 {
		score += 0.4 * (float64(project.ActiveWorkTime) / float64(4*time.Hour))
	}
	
	// Claude API usage (30% weight)
	if project.ClaudeAPICalls > 0 {
		score += 0.3 * (float64(project.ClaudeAPICalls) / 20.0)
	}
	
	// File operations (20% weight)
	if project.FileOperations > 0 {
		score += 0.2 * (float64(project.FileOperations) / 50.0)
	}
	
	// Activity consistency (10% weight)
	if project.Activities > 0 {
		score += 0.1 * (float64(project.Activities) / 100.0)
	}
	
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

func (wtc *WorkTimeCalculator) getActivityTypeDescription(activityType generator.ActivityType) string {
	descriptions := map[generator.ActivityType]string{
		generator.ActivityFileWork:    "File Operations",
		generator.ActivityClaudeAPI:   "Claude API Usage",
		generator.ActivityCodeWork:    "Code Development",
		generator.ActivityDocWork:     "Documentation",
		generator.ActivityProjectWork: "Project Management",
		generator.ActivityIdle:        "Idle Time",
	}
	
	if desc, exists := descriptions[activityType]; exists {
		return desc
	}
	return "Unknown Activity"
}

// Filter methods

func (wtc *WorkTimeCalculator) filterActivitiesByTimeRange(timeRange TimeRange) []generator.WorkActivity {
	var filtered []generator.WorkActivity
	
	for _, activity := range wtc.activities {
		if activity.Timestamp.After(timeRange.Start) && activity.Timestamp.Before(timeRange.End) {
			filtered = append(filtered, activity)
		}
	}
	
	return filtered
}

func (wtc *WorkTimeCalculator) filterSessionsByTimeRange(timeRange TimeRange) []generator.WorkSession {
	var filtered []generator.WorkSession
	
	for _, session := range wtc.sessions {
		if session.StartTime.Before(timeRange.End) && session.EndTime.After(timeRange.Start) {
			filtered = append(filtered, session)
		}
	}
	
	return filtered
}

func (wtc *WorkTimeCalculator) filterWorkBlocksByTimeRange(timeRange TimeRange) []generator.WorkBlock {
	var filtered []generator.WorkBlock
	
	for _, block := range wtc.workBlocks {
		if block.StartTime.After(timeRange.Start) && block.StartTime.Before(timeRange.End) {
			filtered = append(filtered, block)
		}
	}
	
	return filtered
}

// Cache and utility methods

func (wtc *WorkTimeCalculator) invalidateCache() {
	wtc.calculationCache = make(map[string]interface{})
	wtc.lastCalculation = time.Time{}
}

func (wtc *WorkTimeCalculator) generateComparisons(timeRange TimeRange, activeWorkTime time.Duration) WorkTimeComparisons {
	// Placeholder implementation - would compare with previous periods
	return WorkTimeComparisons{
		Trend:        "stable",
		DailyAverage: activeWorkTime / time.Duration(timeRange.Duration.Hours()/24),
	}
}

func (wtc *WorkTimeCalculator) generateInsights(activeTime, sessionTime time.Duration, efficiencyRatio, focusScore float64) []WorkTimeInsight {
	var insights []WorkTimeInsight
	
	// Efficiency insight
	if efficiencyRatio > 0.8 {
		insights = append(insights, WorkTimeInsight{
			Type:        "efficiency",
			Category:    "positive",
			Title:       "High Work Efficiency",
			Description: fmt.Sprintf("You maintained %.1f%% efficiency during Claude sessions", efficiencyRatio*100),
			Impact:      "high",
			Suggestion:  "Continue this excellent work pattern",
		})
	} else if efficiencyRatio < 0.5 {
		insights = append(insights, WorkTimeInsight{
			Type:        "efficiency",
			Category:    "warning",
			Title:       "Low Work Efficiency",
			Description: fmt.Sprintf("Efficiency is %.1f%% - consider reducing idle time", efficiencyRatio*100),
			Impact:      "medium",
			Suggestion:  "Try working in shorter, focused sessions",
		})
	}
	
	// Focus insight
	if focusScore > 0.7 {
		insights = append(insights, WorkTimeInsight{
			Type:        "pattern",
			Category:    "positive",
			Title:       "Excellent Focus",
			Description: fmt.Sprintf("High focus score of %.1f indicates sustained work periods", focusScore),
			Impact:      "high",
			Suggestion:  "Maintain these focused work sessions",
		})
	}
	
	return insights
}

/**
 * CONTEXT:   Print formatted work time report
 * INPUT:     Work time report
 * OUTPUT:    Formatted console report
 * BUSINESS:  Report printing provides readable work time analysis
 * CHANGE:    Initial report printing for work time analysis
 * RISK:      Low - Display utility for work time reports
 */
func (wtc *WorkTimeCalculator) PrintReport(report *WorkTimeReport) {
	fmt.Printf("\n========== WORK TIME REPORT (%s) ==========\n", report.ReportType)
	fmt.Printf("Generated: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Period: %s to %s (%s)\n", 
		report.TimeRange.Start.Format("15:04"), 
		report.TimeRange.End.Format("15:04"),
		report.TimeRange.Duration.Round(time.Minute))
	
	fmt.Printf("\n🎯 CORE TIME METRICS:\n")
	fmt.Printf("  Active Work Time: %s\n", report.ActiveWorkTime.Round(time.Minute))
	fmt.Printf("  Session Time:     %s\n", report.SessionTime.Round(time.Minute))
	fmt.Printf("  Work Day Time:    %s\n", report.WorkDayTime.Round(time.Minute))
	
	fmt.Printf("\n📊 PRODUCTIVITY METRICS:\n")
	fmt.Printf("  Productivity Score: %.1f%%\n", report.ProductivityScore*100)
	fmt.Printf("  Efficiency Ratio:   %.1f%%\n", report.EfficiencyRatio*100)
	fmt.Printf("  Focus Score:        %.1f%%\n", report.FocusScore*100)
	
	fmt.Printf("\n🏗️  PROJECT BREAKDOWN:\n")
	for i, project := range report.ProjectBreakdown {
		if i >= 5 { // Show top 5 projects
			break
		}
		fmt.Printf("  %d. %s: %s (Score: %.1f%%)\n", 
			i+1, project.ProjectName, 
			project.ActiveWorkTime.Round(time.Minute),
			project.ProductivityScore*100)
	}
	
	fmt.Printf("\n⚡ ACTIVITY BREAKDOWN:\n")
	for _, activity := range report.ActivityBreakdown {
		fmt.Printf("  %s: %s (%.1f%% - %d activities)\n",
			activity.Description,
			activity.TotalTime.Round(time.Minute),
			activity.Percentage,
			activity.Count)
	}
	
	if len(report.Insights) > 0 {
		fmt.Printf("\n💡 KEY INSIGHTS:\n")
		for _, insight := range report.Insights {
			fmt.Printf("  %s: %s\n", insight.Category, insight.Title)
			fmt.Printf("    %s\n", insight.Description)
		}
	}
	
	fmt.Printf("\n📈 STATISTICS:\n")
	fmt.Printf("  Total Activities: %d\n", report.TotalActivities)
	fmt.Printf("  Total Sessions:   %d\n", report.TotalSessions)
	fmt.Printf("  Total Work Blocks: %d\n", report.TotalWorkBlocks)
	fmt.Printf("  Avg Block Duration: %s\n", report.AverageBlockDuration.Round(time.Second))
	
	fmt.Printf("==================================================\n")
}