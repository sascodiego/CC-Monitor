/**
 * CONTEXT:   Comprehensive KuzuDB reporting queries for work hour analytics
 * INPUT:     Time periods, user IDs, project filters, and query parameters
 * OUTPUT:    Optimized Cypher queries for daily/weekly/monthly/historical reports
 * BUSINESS:  Generate fast, comprehensive work analytics with < 100ms response times
 * CHANGE:    Initial implementation of optimized reporting query system
 * RISK:      Low - Read-only queries with proper parameterization and caching
 */

package database

import (
	"context"
	"fmt"
	"time"
)

/**
 * CONTEXT:   Reporting query manager with performance optimization and caching
 * INPUT:     Database connection manager and caching configuration
 * OUTPUT:    High-performance reporting queries with sub-100ms response times
 * BUSINESS:  Efficient work analytics queries for beautiful CLI reports
 * CHANGE:    Initial implementation with query optimization and smart caching
 * RISK:      Medium - Query complexity requires careful optimization
 */
type ReportingQueries struct {
	connectionManager *KuzuConnectionManager
	queryCache       *QueryCache
}

// NewReportingQueries creates a new reporting query manager
func NewReportingQueries(connectionManager *KuzuConnectionManager) *ReportingQueries {
	return &ReportingQueries{
		connectionManager: connectionManager,
		queryCache:       NewQueryCache(),
	}
}

// TimePeriod represents different time period types for historical analysis
type TimePeriod struct {
	Type  string    `json:"type"`  // "day", "week", "month", "custom"
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

/**
 * CONTEXT:   Daily report data structure with comprehensive work metrics
 * INPUT:     Daily work data aggregated from sessions and work blocks
 * OUTPUT:    Complete daily analytics suitable for CLI display
 * BUSINESS:  Daily reports show work time, efficiency, and project breakdown
 * CHANGE:    Initial daily report structure with rich analytics
 * RISK:      Low - Data structure for read-only reporting
 */
type DailyReport struct {
	Date                 time.Time         `json:"date"`
	TotalWorkHours       float64           `json:"total_work_hours"`
	TotalScheduleHours   float64           `json:"total_schedule_hours"`
	StartTime            time.Time         `json:"start_time"`
	EndTime              time.Time         `json:"end_time"`
	Efficiency           float64           `json:"efficiency"`
	SessionCount         int64             `json:"session_count"`
	WorkBlockCount       int64             `json:"work_block_count"`
	ActivityCount        int64             `json:"activity_count"`
	ProjectBreakdown     []ProjectTime     `json:"project_breakdown"`
	HourlyBreakdown      []HourlyStats     `json:"hourly_breakdown"`
	MostProductiveHour   int               `json:"most_productive_hour"`
	LongestWorkBlock     time.Duration     `json:"longest_work_block"`
	AverageBlockSize     time.Duration     `json:"average_block_size"`
	IdleTime             time.Duration     `json:"idle_time"`
}

/**
 * CONTEXT:   Weekly report data structure with trend analysis
 * INPUT:     Weekly work data with daily breakdowns and comparisons
 * OUTPUT:    Comprehensive weekly analytics with trend insights
 * BUSINESS:  Weekly reports show patterns, trends, and weekly goals progress
 * CHANGE:    Initial weekly report structure with trend analysis
 * RISK:      Low - Data structure for analytics and reporting
 */
type WeeklyReport struct {
	WeekStart          time.Time       `json:"week_start"`
	WeekEnd            time.Time       `json:"week_end"`
	TotalWorkHours     float64         `json:"total_work_hours"`
	TotalScheduleHours float64         `json:"total_schedule_hours"`
	Efficiency         float64         `json:"efficiency"`
	DailyBreakdown     []DailyReport   `json:"daily_breakdown"`
	TopProjects        []ProjectTime   `json:"top_projects"`
	MostProductiveDay  string          `json:"most_productive_day"`
	WeeklyTrend        string          `json:"weekly_trend"` // "increasing", "decreasing", "stable"
	ComparedToLastWeek float64         `json:"compared_to_last_week"` // percentage change
}

/**
 * CONTEXT:   Monthly report data structure with comprehensive analytics
 * INPUT:     Monthly work data with weekly and daily granularity
 * OUTPUT:    Complete monthly analytics with calendar heatmap data
 * BUSINESS:  Monthly reports show long-term patterns and monthly goals
 * CHANGE:    Initial monthly report structure with rich analytics
 * RISK:      Low - Data structure for comprehensive monthly reporting
 */
type MonthlyReport struct {
	Month              time.Time       `json:"month"`
	Year               int             `json:"year"`
	TotalWorkHours     float64         `json:"total_work_hours"`
	TotalScheduleHours float64         `json:"total_schedule_hours"`
	Efficiency         float64         `json:"efficiency"`
	WorkingDays        int             `json:"working_days"`
	AverageHoursPerDay float64         `json:"average_hours_per_day"`
	WeeklyBreakdown    []WeeklyReport  `json:"weekly_breakdown"`
	CalendarHeatmap    []DayStats      `json:"calendar_heatmap"`
	TopProjects        []ProjectTime   `json:"top_projects"`
	MonthlyTrend       string          `json:"monthly_trend"`
	ComparedToLastMonth float64        `json:"compared_to_last_month"`
	GoalProgress       float64         `json:"goal_progress"` // percentage of monthly goal
}

/**
 * CONTEXT:   Project time allocation data structure
 * INPUT:     Project-specific work metrics and analytics
 * OUTPUT:    Project time data suitable for rankings and breakdowns
 * BUSINESS:  Project breakdown shows time allocation and work distribution
 * CHANGE:    Initial project time structure for analytics
 * RISK:      Low - Data structure for project analytics
 */
type ProjectTime struct {
	Name            string        `json:"name"`
	Path            string        `json:"path"`
	Hours           float64       `json:"hours"`
	Percentage      float64       `json:"percentage"`
	WorkBlocks      int64         `json:"work_blocks"`
	ActiveDays      int64         `json:"active_days"`
	AvgBlockSize    time.Duration `json:"avg_block_size"`
	FirstActivity   time.Time     `json:"first_activity"`
	LastActivity    time.Time     `json:"last_activity"`
	ProductivityTrend string      `json:"productivity_trend"`
}

/**
 * CONTEXT:   Hourly statistics for productivity heatmap
 * INPUT:     Hour-by-hour work activity data
 * OUTPUT:    Hourly productivity metrics for visualization
 * BUSINESS:  Hourly breakdown shows daily work patterns and peak hours
 * CHANGE:    Initial hourly statistics structure
 * RISK:      Low - Data structure for hourly analytics
 */
type HourlyStats struct {
	Hour            int           `json:"hour"`
	WorkMinutes     int64         `json:"work_minutes"`
	WorkBlocks      int64         `json:"work_blocks"`
	ActivityCount   int64         `json:"activity_count"`
	Efficiency      float64       `json:"efficiency"`
	ProductivityLevel string      `json:"productivity_level"` // "high", "medium", "low"
}

/**
 * CONTEXT:   Daily statistics for calendar heatmap visualization
 * INPUT:     Day-level work metrics for heatmap generation
 * OUTPUT:    Daily stats suitable for calendar heatmap visualization
 * BUSINESS:  Calendar heatmap shows work patterns across time periods
 * CHANGE:    Initial daily statistics structure for heatmap
 * RISK:      Low - Data structure for calendar visualization
 */
type DayStats struct {
	Date              time.Time `json:"date"`
	WorkHours         float64   `json:"work_hours"`
	Efficiency        float64   `json:"efficiency"`
	ProductivityLevel string    `json:"productivity_level"`
	SessionCount      int64     `json:"session_count"`
	WorkBlockCount    int64     `json:"work_block_count"`
}

/**
 * CONTEXT:   Get comprehensive daily report with optimized query performance
 * INPUT:     Date, user ID for filtering, context for timeout control
 * OUTPUT:    Complete daily report with project breakdown and hourly stats
 * BUSINESS:  Daily reports are most frequently requested, must be fast (< 50ms)
 * CHANGE:    Initial implementation with caching and optimization
 * RISK:      Low - Read-only query with proper parameterization
 */
func (rq *ReportingQueries) GetDailyReport(ctx context.Context, date time.Time, userID string) (*DailyReport, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("daily_report_%s_%s", userID, date.Format("2006-01-02"))
	if cached := rq.queryCache.Get(cacheKey); cached != nil {
		if report, ok := cached.(*DailyReport); ok {
			return report, nil
		}
	}

	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
		WHERE DATE(s.start_time) = DATE($date) AND s.is_active = true
		OPTIONAL MATCH (s)-[:CONTAINS_WORK]->(w:WorkBlock)-[:WORK_IN_PROJECT]->(p:Project)
		WHERE w.is_active = true
		WITH s, p, w,
		     CASE 
		         WHEN w IS NOT NULL THEN w.duration_hours 
		         ELSE 0.0 
		     END as work_hours,
		     CASE 
		         WHEN w IS NOT NULL THEN w.duration_seconds 
		         ELSE 0 
		     END as work_seconds
		WITH s, p,
		     SUM(work_hours) as project_hours,
		     SUM(work_seconds) as project_seconds,
		     COUNT(w) as project_blocks,
		     MIN(CASE WHEN w IS NOT NULL THEN w.start_time ELSE NULL END) as first_work,
		     MAX(CASE WHEN w IS NOT NULL THEN w.end_time ELSE NULL END) as last_work,
		     MAX(work_seconds) as longest_block_seconds
		RETURN 
		    DATE(s.start_time) as date,
		    SUM(project_hours) as total_work_hours,
		    COUNT(DISTINCT s) as session_count,
		    SUM(project_blocks) as work_block_count,
		    SUM(s.activity_count) as activity_count,
		    MIN(first_work) as start_time,
		    MAX(last_work) as end_time,
		    MAX(longest_block_seconds) as longest_block_seconds,
		    CASE 
		        WHEN SUM(project_blocks) > 0 
		        THEN AVG(project_seconds) 
		        ELSE 0 
		    END as avg_block_seconds,
		    COLLECT({
		        name: CASE WHEN p IS NOT NULL THEN p.name ELSE 'Unknown' END,
		        path: CASE WHEN p IS NOT NULL THEN p.path ELSE '' END,
		        hours: project_hours,
		        work_blocks: project_blocks,
		        first_activity: first_work,
		        last_activity: last_work
		    }) as project_breakdown
		ORDER BY date
	`

	params := map[string]interface{}{
		"user_id": userID,
		"date":    date,
	}

	result, err := rq.connectionManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("daily report query failed: %w", err)
	}
	defer result.Close()

	report, err := rq.parseDailyReport(result, date)
	if err != nil {
		return nil, fmt.Errorf("failed to parse daily report: %w", err)
	}

	// Get hourly breakdown
	hourlyStats, err := rq.getHourlyBreakdown(ctx, date, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly breakdown: %w", err)
	}
	report.HourlyBreakdown = hourlyStats

	// Calculate derived metrics
	report.Efficiency = rq.calculateEfficiency(report.TotalWorkHours, report.TotalScheduleHours)
	report.MostProductiveHour = rq.findMostProductiveHour(hourlyStats)
	
	// Cache the result (5 minute TTL for current day, 1 hour for past days)
	ttl := 1 * time.Hour
	if date.Format("2006-01-02") == time.Now().Format("2006-01-02") {
		ttl = 5 * time.Minute
	}
	rq.queryCache.Set(cacheKey, report, ttl)

	return report, nil
}

/**
 * CONTEXT:   Get hourly breakdown for daily productivity analysis
 * INPUT:     Date, user ID for filtering work blocks by hour
 * OUTPUT:    Hour-by-hour productivity statistics
 * BUSINESS:  Hourly breakdown shows peak productivity hours and work patterns
 * CHANGE:    Initial implementation with hour-based aggregation
 * RISK:      Low - Read-only aggregation query with proper indexing
 */
func (rq *ReportingQueries) getHourlyBreakdown(ctx context.Context, date time.Time, userID string) ([]HourlyStats, error) {
	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)-[:CONTAINS_WORK]->(w:WorkBlock)
		WHERE DATE(w.start_time) = DATE($date) AND w.is_active = true
		WITH HOUR(w.start_time) as hour, 
		     w.duration_seconds as work_seconds,
		     w.activity_count as activities
		WITH hour,
		     SUM(work_seconds) as total_seconds,
		     COUNT(*) as work_blocks,
		     SUM(activities) as total_activities
		RETURN 
		    hour,
		    total_seconds / 60 as work_minutes,
		    work_blocks,
		    total_activities,
		    CASE 
		        WHEN work_blocks > 0 
		        THEN (total_seconds / 3600.0) / work_blocks 
		        ELSE 0.0 
		    END as efficiency
		ORDER BY hour
	`

	params := map[string]interface{}{
		"user_id": userID,
		"date":    date,
	}

	result, err := rq.connectionManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("hourly breakdown query failed: %w", err)
	}
	defer result.Close()

	return rq.parseHourlyStats(result)
}

/**
 * CONTEXT:   Get comprehensive weekly report with trend analysis
 * INPUT:     Week start date, user ID for filtering
 * OUTPUT:    Complete weekly report with daily breakdown and trends
 * BUSINESS:  Weekly reports show work patterns and productivity trends
 * CHANGE:    Initial implementation with trend calculation
 * RISK:      Low - Read-only query with efficient weekly aggregation
 */
func (rq *ReportingQueries) GetWeeklyReport(ctx context.Context, weekStart time.Time, userID string) (*WeeklyReport, error) {
	// Normalize to Monday start of week
	weekStart = rq.getWeekStart(weekStart)
	weekEnd := weekStart.Add(7 * 24 * time.Hour)

	// Check cache first
	cacheKey := fmt.Sprintf("weekly_report_%s_%s", userID, weekStart.Format("2006-01-02"))
	if cached := rq.queryCache.Get(cacheKey); cached != nil {
		if report, ok := cached.(*WeeklyReport); ok {
			return report, nil
		}
	}

	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
		WHERE s.start_time >= $week_start AND s.start_time < $week_end AND s.is_active = true
		OPTIONAL MATCH (s)-[:CONTAINS_WORK]->(w:WorkBlock)-[:WORK_IN_PROJECT]->(p:Project)
		WHERE w.is_active = true
		WITH DATE(s.start_time) as work_date,
		     p,
		     SUM(CASE WHEN w IS NOT NULL THEN w.duration_hours ELSE 0.0 END) as daily_hours,
		     SUM(CASE WHEN w IS NOT NULL THEN w.duration_seconds ELSE 0 END) as daily_seconds,
		     COUNT(DISTINCT s) as daily_sessions,
		     COUNT(w) as daily_blocks,
		     MIN(CASE WHEN w IS NOT NULL THEN w.start_time ELSE NULL END) as first_work,
		     MAX(CASE WHEN w IS NOT NULL THEN w.end_time ELSE NULL END) as last_work
		WITH work_date, p,
		     daily_hours,
		     daily_sessions,
		     daily_blocks,
		     first_work,
		     last_work,
		     CASE 
		         WHEN first_work IS NOT NULL AND last_work IS NOT NULL 
		         THEN (last_work - first_work) / 3600.0 
		         ELSE 0.0 
		     END as schedule_hours
		RETURN 
		    work_date,
		    SUM(daily_hours) as total_work_hours,
		    SUM(schedule_hours) as total_schedule_hours,
		    SUM(daily_sessions) as session_count,
		    SUM(daily_blocks) as work_block_count,
		    MIN(first_work) as week_start_time,
		    MAX(last_work) as week_end_time,
		    COLLECT({
		        name: CASE WHEN p IS NOT NULL THEN p.name ELSE 'Unknown' END,
		        path: CASE WHEN p IS NOT NULL THEN p.path ELSE '' END,
		        hours: daily_hours,
		        work_blocks: daily_blocks
		    }) as project_breakdown
		ORDER BY work_date
	`

	params := map[string]interface{}{
		"user_id":    userID,
		"week_start": weekStart,
		"week_end":   weekEnd,
	}

	result, err := rq.connectionManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("weekly report query failed: %w", err)
	}
	defer result.Close()

	report, err := rq.parseWeeklyReport(result, weekStart, weekEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse weekly report: %w", err)
	}

	// Get daily breakdown for the week
	dailyReports := make([]DailyReport, 0, 7)
	for i := 0; i < 7; i++ {
		day := weekStart.Add(time.Duration(i) * 24 * time.Hour)
		dailyReport, err := rq.GetDailyReport(ctx, day, userID)
		if err != nil {
			// Don't fail the whole report for one day
			continue
		}
		dailyReports = append(dailyReports, *dailyReport)
	}
	report.DailyBreakdown = dailyReports

	// Calculate trends and comparisons
	report.MostProductiveDay = rq.findMostProductiveDay(dailyReports)
	report.WeeklyTrend = rq.calculateWeeklyTrend(dailyReports)
	
	// Compare with last week
	lastWeekReport, err := rq.GetWeeklyReport(ctx, weekStart.Add(-7*24*time.Hour), userID)
	if err == nil {
		report.ComparedToLastWeek = rq.calculatePercentageChange(lastWeekReport.TotalWorkHours, report.TotalWorkHours)
	}

	// Cache the result (15 minutes TTL for current week, 2 hours for past weeks)
	ttl := 2 * time.Hour
	if rq.isCurrentWeek(weekStart) {
		ttl = 15 * time.Minute
	}
	rq.queryCache.Set(cacheKey, report, ttl)

	return report, nil
}

/**
 * CONTEXT:   Get comprehensive monthly report with calendar heatmap data
 * INPUT:     Month and year, user ID for filtering
 * OUTPUT:    Complete monthly report with weekly breakdown and heatmap
 * BUSINESS:  Monthly reports show long-term patterns and goal progress
 * CHANGE:    Initial implementation with calendar heatmap generation
 * RISK:      Medium - Complex monthly aggregation requires optimization
 */
func (rq *ReportingQueries) GetMonthlyReport(ctx context.Context, month time.Time, userID string) (*MonthlyReport, error) {
	// Normalize to first day of month
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)

	// Check cache first
	cacheKey := fmt.Sprintf("monthly_report_%s_%s", userID, monthStart.Format("2006-01"))
	if cached := rq.queryCache.Get(cacheKey); cached != nil {
		if report, ok := cached.(*MonthlyReport); ok {
			return report, nil
		}
	}

	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
		WHERE s.start_time >= $month_start AND s.start_time < $month_end AND s.is_active = true
		OPTIONAL MATCH (s)-[:CONTAINS_WORK]->(w:WorkBlock)-[:WORK_IN_PROJECT]->(p:Project)
		WHERE w.is_active = true
		WITH DATE(s.start_time) as work_date,
		     p,
		     SUM(CASE WHEN w IS NOT NULL THEN w.duration_hours ELSE 0.0 END) as daily_hours,
		     COUNT(DISTINCT s) as daily_sessions,
		     COUNT(w) as daily_blocks,
		     MIN(CASE WHEN w IS NOT NULL THEN w.start_time ELSE NULL END) as first_work,
		     MAX(CASE WHEN w IS NOT NULL THEN w.end_time ELSE NULL END) as last_work
		RETURN 
		    work_date,
		    SUM(daily_hours) as total_work_hours,
		    SUM(daily_sessions) as session_count,
		    SUM(daily_blocks) as work_block_count,
		    MIN(first_work) as day_start_time,
		    MAX(last_work) as day_end_time,
		    COUNT(DISTINCT work_date) as working_days,
		    COLLECT({
		        date: work_date,
		        hours: daily_hours,
		        sessions: daily_sessions,
		        blocks: daily_blocks,
		        efficiency: CASE 
		            WHEN first_work IS NOT NULL AND last_work IS NOT NULL AND (last_work - first_work) > 0
		            THEN daily_hours / ((last_work - first_work) / 3600.0)
		            ELSE 0.0
		        END
		    }) as daily_breakdown,
		    COLLECT({
		        name: CASE WHEN p IS NOT NULL THEN p.name ELSE 'Unknown' END,
		        path: CASE WHEN p IS NOT NULL THEN p.path ELSE '' END,
		        hours: daily_hours,
		        work_blocks: daily_blocks
		    }) as project_breakdown
		ORDER BY work_date
	`

	params := map[string]interface{}{
		"user_id":     userID,
		"month_start": monthStart,
		"month_end":   monthEnd,
	}

	result, err := rq.connectionManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("monthly report query failed: %w", err)
	}
	defer result.Close()

	report, err := rq.parseMonthlyReport(result, monthStart)
	if err != nil {
		return nil, fmt.Errorf("failed to parse monthly report: %w", err)
	}

	// Generate calendar heatmap data
	heatmapData, err := rq.generateCalendarHeatmap(ctx, monthStart, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate calendar heatmap: %w", err)
	}
	report.CalendarHeatmap = heatmapData

	// Get weekly breakdowns for the month
	weeklyReports := make([]WeeklyReport, 0)
	currentWeek := rq.getWeekStart(monthStart)
	for currentWeek.Before(monthEnd) {
		if currentWeek.Month() == monthStart.Month() {
			weeklyReport, err := rq.GetWeeklyReport(ctx, currentWeek, userID)
			if err == nil {
				weeklyReports = append(weeklyReports, *weeklyReport)
			}
		}
		currentWeek = currentWeek.Add(7 * 24 * time.Hour)
	}
	report.WeeklyBreakdown = weeklyReports

	// Calculate monthly trends and comparisons
	lastMonthReport, err := rq.GetMonthlyReport(ctx, monthStart.AddDate(0, -1, 0), userID)
	if err == nil {
		report.ComparedToLastMonth = rq.calculatePercentageChange(lastMonthReport.TotalWorkHours, report.TotalWorkHours)
	}
	
	report.MonthlyTrend = rq.calculateMonthlyTrend(weeklyReports)

	// Cache the result (30 minutes TTL for current month, 4 hours for past months)
	ttl := 4 * time.Hour
	if rq.isCurrentMonth(monthStart) {
		ttl = 30 * time.Minute
	}
	rq.queryCache.Set(cacheKey, report, ttl)

	return report, nil
}

/**
 * CONTEXT:   Get historical analysis for any time period with comparison
 * INPUT:     Historical period specification, user ID
 * OUTPUT:    Historical report with trends and period comparisons
 * BUSINESS:  Historical analysis enables long-term productivity insights
 * CHANGE:    Initial implementation with flexible period handling
 * RISK:      Medium - Large date ranges require query optimization
 */
func (rq *ReportingQueries) GetHistoricalAnalysis(ctx context.Context, period TimePeriod, userID string) (*MonthlyReport, error) {
	// For now, delegate to monthly report for month periods
	if period.Type == "month" {
		return rq.GetMonthlyReport(ctx, period.Start, userID)
	}

	// For custom periods, create a generalized report
	return rq.getCustomPeriodReport(ctx, period, userID)
}

/**
 * CONTEXT:   Get project-specific deep dive analysis
 * INPUT:     Project ID, time period, user ID for filtering
 * OUTPUT:    Detailed project analytics with work patterns and efficiency
 * BUSINESS:  Project analysis shows detailed time allocation and productivity per project
 * CHANGE:    Initial implementation with project-focused analytics
 * RISK:      Low - Project-scoped query with proper indexing
 */
func (rq *ReportingQueries) GetProjectReport(ctx context.Context, projectID string, period TimePeriod, userID string) (*ProjectReport, error) {
	query := `
		MATCH (u:User {id: $user_id})-[:WORKS_ON]->(p:Project {id: $project_id})
		MATCH (p)<-[:WORK_IN_PROJECT]-(w:WorkBlock)
		WHERE w.start_time >= $start_date AND w.end_time <= $end_date AND w.is_active = true
		OPTIONAL MATCH (w)<-[:CONTAINS_WORK]-(s:Session)
		WITH p, w, s,
		     DATE(w.start_time) as work_date,
		     HOUR(w.start_time) as work_hour,
		     w.duration_hours as hours,
		     w.duration_seconds as seconds,
		     w.activity_count as activities
		RETURN 
		    p.name as project_name,
		    p.path as project_path,
		    SUM(hours) as total_hours,
		    COUNT(w) as work_blocks,
		    COUNT(DISTINCT work_date) as active_days,
		    COUNT(DISTINCT s) as sessions,
		    SUM(activities) as total_activities,
		    MIN(w.start_time) as first_activity,
		    MAX(w.end_time) as last_activity,
		    AVG(seconds) as avg_block_seconds,
		    MAX(seconds) as longest_block_seconds,
		    COLLECT(DISTINCT work_date) as active_dates,
		    COLLECT({
		        date: work_date,
		        hour: work_hour,
		        hours: hours,
		        blocks: COUNT(w),
		        activities: SUM(activities)
		    }) as daily_breakdown
	`

	params := map[string]interface{}{
		"user_id":    userID,
		"project_id": projectID,
		"start_date": period.Start,
		"end_date":   period.End,
	}

	result, err := rq.connectionManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("project report query failed: %w", err)
	}
	defer result.Close()

	return rq.parseProjectReport(result, period)
}

/**
 * CONTEXT:   Get project rankings by time investment for time period
 * INPUT:     Time period, user ID, ranking criteria
 * OUTPUT:    Ranked list of projects by work time and other metrics
 * BUSINESS:  Project rankings show time allocation priorities and focus areas
 * CHANGE:    Initial implementation with flexible ranking criteria
 * RISK:      Low - Aggregation query with efficient project indexing
 */
func (rq *ReportingQueries) GetProjectRankings(ctx context.Context, period TimePeriod, userID string) ([]ProjectTime, error) {
	query := `
		MATCH (u:User {id: $user_id})-[:WORKS_ON]->(p:Project)
		MATCH (p)<-[:WORK_IN_PROJECT]-(w:WorkBlock)
		WHERE w.start_time >= $start_date AND w.end_time <= $end_date AND w.is_active = true
		WITH p,
		     SUM(w.duration_hours) as total_hours,
		     COUNT(w) as work_blocks,
		     COUNT(DISTINCT DATE(w.start_time)) as active_days,
		     MIN(w.start_time) as first_activity,
		     MAX(w.end_time) as last_activity,
		     AVG(w.duration_seconds) as avg_block_seconds
		WITH p, total_hours, work_blocks, active_days, first_activity, last_activity, avg_block_seconds,
		     SUM(total_hours) OVER () as grand_total
		RETURN 
		    p.name as project_name,
		    p.path as project_path,
		    total_hours as hours,
		    (total_hours / grand_total * 100) as percentage,
		    work_blocks,
		    active_days,
		    avg_block_seconds,
		    first_activity,
		    last_activity
		ORDER BY total_hours DESC
		LIMIT 20
	`

	params := map[string]interface{}{
		"user_id":    userID,
		"start_date": period.Start,
		"end_date":   period.End,
	}

	result, err := rq.connectionManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("project rankings query failed: %w", err)
	}
	defer result.Close()

	return rq.parseProjectRankings(result)
}

// ProjectReport represents detailed project analysis
type ProjectReport struct {
	ProjectName       string        `json:"project_name"`
	ProjectPath       string        `json:"project_path"`
	Period           TimePeriod    `json:"period"`
	TotalHours       float64       `json:"total_hours"`
	WorkBlocks       int64         `json:"work_blocks"`
	ActiveDays       int64         `json:"active_days"`
	Sessions         int64         `json:"sessions"`
	TotalActivities  int64         `json:"total_activities"`
	FirstActivity    time.Time     `json:"first_activity"`
	LastActivity     time.Time     `json:"last_activity"`
	AvgBlockSize     time.Duration `json:"avg_block_size"`
	LongestBlock     time.Duration `json:"longest_block"`
	DailyBreakdown   []DayStats    `json:"daily_breakdown"`
	HourlyPattern    []HourlyStats `json:"hourly_pattern"`
	ProductivityTrend string       `json:"productivity_trend"`
	Efficiency       float64       `json:"efficiency"`
}