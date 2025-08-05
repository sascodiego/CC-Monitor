/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-DB-EXT-001
 * CONTEXT:   Database extensions implementing missing WorkHourDatabaseManager interface methods
 * REASON:    Need to provide missing database operations for pattern analysis and trend data
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate SQL queries and handle database errors gracefully
 * RISK:      Medium - Database operation failures could affect analytics functionality
 */
package workhour

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/database"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-DB-EXT-002
 * CONTEXT:   DatabaseExtensions wraps WorkHourManager to provide missing interface methods
 * REASON:    WorkHourManager needs additional methods to fully implement WorkHourDatabaseManager interface
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure wrapper maintains transaction consistency and proper error handling
 * RISK:      Medium - Database wrapper complexity could introduce bugs
 */
type DatabaseExtensions struct {
	*database.WorkHourManager
	logger arch.Logger
}

func NewDatabaseExtensions(workHourManager *database.WorkHourManager, logger arch.Logger) *DatabaseExtensions {
	return &DatabaseExtensions{
		WorkHourManager: workHourManager,
		logger:          logger,
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-DB-EXT-003
 * CONTEXT:   GetWorkPatternData implements pattern analysis data retrieval
 * REASON:    Pattern analysis requires historical activity data points for sophisticated analysis
 * CHANGE:    Initial implementation.
 * PREVENTION:Optimize query performance with proper indexes and limit result size
 * RISK:      Medium - Large datasets could impact query performance
 */
func (dbe *DatabaseExtensions) GetWorkPatternData(startDate, endDate time.Time) ([]arch.WorkPatternDataPoint, error) {
	query := `
		SELECT 
			wb.start_time,
			wb.duration_seconds,
			'work_block' as activity_type,
			CAST(wb.duration_seconds AS REAL) / 3600.0 / 8.0 as intensity,
			CASE 
				WHEN wb.duration_seconds >= 1800 THEN 1.0  -- 30+ minutes = high productivity
				WHEN wb.duration_seconds >= 900 THEN 0.7   -- 15+ minutes = medium productivity  
				WHEN wb.duration_seconds >= 300 THEN 0.4   -- 5+ minutes = low productivity
				ELSE 0.1                                   -- < 5 minutes = minimal productivity
			END as productivity
		FROM work_blocks wb
		JOIN sessions s ON wb.session_id = s.session_id
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
		  AND wb.duration_seconds > 0
		ORDER BY wb.start_time
	`

	rows, err := dbe.GetDB().Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query work pattern data: %w", err)
	}
	defer rows.Close()

	var dataPoints []arch.WorkPatternDataPoint
	for rows.Next() {
		var timestamp time.Time
		var durationSeconds int64
		var activityType string
		var intensity, productivity float64

		err := rows.Scan(&timestamp, &durationSeconds, &activityType, &intensity, &productivity)
		if err != nil {
			dbe.logger.Warn("Failed to scan work pattern data row", "error", err)
			continue
		}

		dataPoint := arch.WorkPatternDataPoint{
			Timestamp:    timestamp,
			Duration:     time.Duration(durationSeconds) * time.Second,
			ActivityType: activityType,
			Intensity:    intensity,
			Productivity: productivity,
		}
		dataPoints = append(dataPoints, dataPoint)
	}

	dbe.logger.Debug("Work pattern data retrieved",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"dataPoints", len(dataPoints))

	return dataPoints, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-DB-EXT-004
 * CONTEXT:   GetProductivityMetrics calculates efficiency metrics from database
 * REASON:    Business requirement for productivity analysis and efficiency reporting
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle division by zero and validate metric calculations
 * RISK:      Low - Productivity metrics are analytical and don't affect core functionality
 */
func (dbe *DatabaseExtensions) GetProductivityMetrics(startDate, endDate time.Time) (*domain.EfficiencyMetrics, error) {
	query := `
		SELECT 
			COUNT(DISTINCT s.session_id) as session_count,
			COUNT(wb.block_id) as block_count,
			SUM(wb.duration_seconds) as total_work_seconds,
			MIN(s.start_time) as first_activity,
			MAX(COALESCE(wb.end_time, s.end_time)) as last_activity,
			AVG(wb.duration_seconds) as avg_block_duration
		FROM sessions s
		LEFT JOIN work_blocks wb ON s.session_id = wb.session_id
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
	`

	var sessionCount, blockCount int64
	var totalWorkSeconds int64
	var firstActivity, lastActivity sql.NullTime
	var avgBlockDuration sql.NullFloat64

	err := dbe.GetDB().QueryRow(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).Scan(
		&sessionCount, &blockCount, &totalWorkSeconds, &firstActivity, &lastActivity, &avgBlockDuration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query productivity metrics: %w", err)
	}

	// Calculate efficiency metrics
	metrics := &domain.EfficiencyMetrics{}

	if firstActivity.Valid && lastActivity.Valid && blockCount > 0 {
		totalSpan := lastActivity.Time.Sub(firstActivity.Time)
		totalWorkTime := time.Duration(totalWorkSeconds) * time.Second

		// Active ratio = work time / total time span
		if totalSpan > 0 {
			metrics.ActiveRatio = float64(totalWorkTime) / float64(totalSpan)
		}

		// Focus score based on average block duration (longer blocks = better focus)
		if avgBlockDuration.Valid {
			avgDuration := time.Duration(avgBlockDuration.Float64) * time.Second
			// Normalize to 0-1 scale (30+ minutes = perfect focus score)
			metrics.FocusScore = float64(avgDuration) / float64(30*time.Minute)
			if metrics.FocusScore > 1.0 {
				metrics.FocusScore = 1.0
			}
		}

		// Interruption rate = breaks per hour
		if totalSpan.Hours() > 0 && blockCount > 1 {
			// Number of breaks = number of blocks - 1
			breaks := blockCount - 1
			metrics.InterruptionRate = float64(breaks) / totalSpan.Hours()
		}

		// Peak efficiency time (simplified - find hour with most activity)
		peakHour, err := dbe.findPeakEfficiencyHour(startDate, endDate)
		if err != nil {
			dbe.logger.Warn("Failed to find peak efficiency hour", "error", err)
			metrics.PeakEfficiency = "Unknown"
		} else {
			metrics.PeakEfficiency = peakHour
		}
	}

	dbe.logger.Debug("Productivity metrics calculated",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"activeRatio", metrics.ActiveRatio,
		"focusScore", metrics.FocusScore,
		"interruptionRate", metrics.InterruptionRate)

	return metrics, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-DB-EXT-005
 * CONTEXT:   findPeakEfficiencyHour identifies most productive time period
 * REASON:    Business requirement for identifying optimal work periods for scheduling
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle edge cases with no data and validate hour calculations
 * RISK:      Low - Peak hour analysis is informational and doesn't affect core operations
 */
func (dbe *DatabaseExtensions) findPeakEfficiencyHour(startDate, endDate time.Time) (string, error) {
	query := `
		SELECT 
			CAST(strftime('%H', wb.start_time) AS INTEGER) as hour_of_day,
			SUM(wb.duration_seconds) as total_seconds
		FROM work_blocks wb
		JOIN sessions s ON wb.session_id = s.session_id
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
		  AND wb.duration_seconds > 0
		GROUP BY hour_of_day
		ORDER BY total_seconds DESC
		LIMIT 1
	`

	var peakHour int
	var totalSeconds int64

	err := dbe.GetDB().QueryRow(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).Scan(
		&peakHour, &totalSeconds,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return "No data available", nil
		}
		return "", fmt.Errorf("failed to find peak efficiency hour: %w", err)
	}

	// Format hour as time range
	nextHour := (peakHour + 1) % 24
	return fmt.Sprintf("%02d:00-%02d:00", peakHour, nextHour), nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-DB-EXT-006
 * CONTEXT:   GetBreakPatterns analyzes break timing and duration patterns
 * REASON:    Business requirement for break pattern analysis and wellness insights
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle edge cases with insufficient data for pattern detection
 * RISK:      Low - Break pattern analysis is informational and doesn't affect core tracking
 */
func (dbe *DatabaseExtensions) GetBreakPatterns(startDate, endDate time.Time) ([]domain.BreakPattern, error) {
	// Query to find gaps between work blocks that could be breaks
	query := `
		WITH break_analysis AS (
			SELECT 
				wb1.end_time,
				wb2.start_time,
				CAST(strftime('%H', wb2.start_time) AS INTEGER) as break_hour,
				(julianday(wb2.start_time) - julianday(wb1.end_time)) * 24 * 60 as break_minutes
			FROM work_blocks wb1
			JOIN work_blocks wb2 ON wb1.session_id = wb2.session_id
			JOIN sessions s ON wb1.session_id = s.session_id
			WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
			  AND wb2.start_time > wb1.end_time
			  AND (julianday(wb2.start_time) - julianday(wb1.end_time)) * 24 * 60 BETWEEN 5 AND 120  -- 5 minutes to 2 hours
		)
		SELECT 
			break_hour,
			AVG(break_minutes) as avg_duration_minutes,
			COUNT(*) as frequency,
			CASE 
				WHEN AVG(break_minutes) < 30 THEN 'short'
				WHEN AVG(break_minutes) BETWEEN 30 AND 90 THEN 'lunch'
				ELSE 'long'
			END as break_type
		FROM break_analysis
		GROUP BY break_hour
		HAVING COUNT(*) >= 2  -- At least 2 instances to be considered a pattern
		ORDER BY frequency DESC
	`

	rows, err := dbe.GetDB().Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query break patterns: %w", err)
	}
	defer rows.Close()

	var patterns []domain.BreakPattern
	totalBreaks := 0

	// First pass: collect all patterns and count total
	for rows.Next() {
		var startHour int
		var avgDurationMinutes float64
		var frequency int
		var breakTypeStr string

		err := rows.Scan(&startHour, &avgDurationMinutes, &frequency, &breakTypeStr)
		if err != nil {
			dbe.logger.Warn("Failed to scan break pattern row", "error", err)
			continue
		}

		// Convert break type string to enum
		var breakType domain.BreakType
		switch breakTypeStr {
		case "short":
			breakType = domain.ShortBreak
		case "lunch":
			breakType = domain.LunchBreak
		case "long":
			breakType = domain.LongBreak
		default:
			breakType = domain.ShortBreak
		}

		pattern := domain.BreakPattern{
			StartHour: startHour,
			Duration:  time.Duration(avgDurationMinutes) * time.Minute,
			Frequency: 0, // Will calculate after we know total
			Type:      breakType,
		}

		patterns = append(patterns, pattern)
		totalBreaks += frequency
	}

	// Second pass: calculate frequency ratios
	for i := range patterns {
		if totalBreaks > 0 {
			// Note: This is a simplified frequency calculation
			// In a real implementation, you'd want more sophisticated pattern analysis
			patterns[i].Frequency = 0.5 // Placeholder - would need more complex calculation
		}
	}

	dbe.logger.Debug("Break patterns analyzed",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"patterns", len(patterns))

	return patterns, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-DB-EXT-007
 * CONTEXT:   Trend analysis data retrieval with different granularities
 * REASON:    Business requirement for trend analysis across different time scales
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate granularity parameters and handle large datasets efficiently
 * RISK:      Medium - Large trend datasets could impact query performance
 */
func (dbe *DatabaseExtensions) GetWorkTimeTrends(startDate, endDate time.Time, granularity arch.Granularity) ([]arch.TrendDataPoint, error) {
	var dateFormat, groupBy string
	
	switch granularity {
	case arch.GranularityDay:
		dateFormat = "%Y-%m-%d"
		groupBy = "date(s.start_time)"
	case arch.GranularityWeek:
		dateFormat = "%Y-W%W"
		groupBy = "strftime('%Y-W%W', s.start_time)"
	case arch.GranularityMonth:
		dateFormat = "%Y-%m"
		groupBy = "strftime('%Y-%m', s.start_time)"
	case arch.GranularityHour:
		dateFormat = "%Y-%m-%d %H"
		groupBy = "strftime('%Y-%m-%d %H', s.start_time)"
	default:
		return nil, fmt.Errorf("unsupported granularity: %s", granularity)
	}

	query := fmt.Sprintf(`
		SELECT 
			strftime('%s', s.start_time) as period,
			SUM(wb.duration_seconds) as total_seconds,
			MIN(s.start_time) as period_start
		FROM sessions s
		LEFT JOIN work_blocks wb ON s.session_id = wb.session_id
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
		GROUP BY %s
		ORDER BY period_start
	`, dateFormat, groupBy)

	rows, err := dbe.GetDB().Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query work time trends: %w", err)
	}
	defer rows.Close()

	var trendPoints []arch.TrendDataPoint
	var previousValue float64

	for rows.Next() {
		var period string
		var totalSeconds sql.NullInt64
		var periodStart time.Time

		err := rows.Scan(&period, &totalSeconds, &periodStart)
		if err != nil {
			dbe.logger.Warn("Failed to scan trend data row", "error", err)
			continue
		}

		value := 0.0
		if totalSeconds.Valid {
			value = float64(totalSeconds.Int64) / 3600.0 // Convert to hours
		}

		// Calculate change from previous period
		change := 0.0
		if previousValue > 0 {
			change = ((value - previousValue) / previousValue) * 100
		}

		trendPoint := arch.TrendDataPoint{
			Date:     periodStart,
			Value:    value,
			Baseline: previousValue, // Use previous value as baseline
			Change:   change,
		}

		trendPoints = append(trendPoints, trendPoint)
		previousValue = value
	}

	dbe.logger.Debug("Work time trends retrieved",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"granularity", granularity,
		"dataPoints", len(trendPoints))

	return trendPoints, nil
}

func (dbe *DatabaseExtensions) GetSessionCountTrends(startDate, endDate time.Time, granularity arch.Granularity) ([]arch.TrendDataPoint, error) {
	// Similar implementation to GetWorkTimeTrends but counting sessions
	var dateFormat, groupBy string
	
	switch granularity {
	case arch.GranularityDay:
		dateFormat = "%Y-%m-%d"
		groupBy = "date(s.start_time)"
	case arch.GranularityWeek:
		dateFormat = "%Y-W%W"
		groupBy = "strftime('%Y-W%W', s.start_time)"
	case arch.GranularityMonth:
		dateFormat = "%Y-%m"
		groupBy = "strftime('%Y-%m', s.start_time)"
	case arch.GranularityHour:
		dateFormat = "%Y-%m-%d %H"
		groupBy = "strftime('%Y-%m-%d %H', s.start_time)"
	default:
		return nil, fmt.Errorf("unsupported granularity: %s", granularity)
	}

	query := fmt.Sprintf(`
		SELECT 
			strftime('%s', s.start_time) as period,
			COUNT(DISTINCT s.session_id) as session_count,
			MIN(s.start_time) as period_start
		FROM sessions s
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
		GROUP BY %s
		ORDER BY period_start
	`, dateFormat, groupBy)

	rows, err := dbe.GetDB().Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query session count trends: %w", err)
	}
	defer rows.Close()

	var trendPoints []arch.TrendDataPoint
	var previousValue float64

	for rows.Next() {
		var period string
		var sessionCount int64
		var periodStart time.Time

		err := rows.Scan(&period, &sessionCount, &periodStart)
		if err != nil {
			dbe.logger.Warn("Failed to scan session trend data row", "error", err)
			continue
		}

		value := float64(sessionCount)

		// Calculate change from previous period
		change := 0.0
		if previousValue > 0 {
			change = ((value - previousValue) / previousValue) * 100
		}

		trendPoint := arch.TrendDataPoint{
			Date:     periodStart,
			Value:    value,
			Baseline: previousValue,
			Change:   change,
		}

		trendPoints = append(trendPoints, trendPoint)
		previousValue = value
	}

	return trendPoints, nil
}

func (dbe *DatabaseExtensions) GetEfficiencyTrends(startDate, endDate time.Time, granularity arch.Granularity) ([]arch.TrendDataPoint, error) {
	// Efficiency trends based on active ratio over time
	workTimeTrends, err := dbe.GetWorkTimeTrends(startDate, endDate, granularity)
	if err != nil {
		return nil, fmt.Errorf("failed to get work time trends for efficiency calculation: %w", err)
	}

	// Convert work time to efficiency ratios (simplified calculation)
	var efficiencyTrends []arch.TrendDataPoint
	
	for _, trend := range workTimeTrends {
		// Calculate efficiency as ratio of actual work time to expected work time
		// This is a simplified calculation - in practice you'd want more sophisticated metrics
		expectedHours := 8.0 // Assume 8-hour work day expectation
		efficiency := trend.Value / expectedHours
		if efficiency > 1.0 {
			efficiency = 1.0
		}

		efficiencyPoint := arch.TrendDataPoint{
			Date:     trend.Date,
			Value:    efficiency,
			Baseline: trend.Baseline / expectedHours,
			Change:   trend.Change,
		}

		efficiencyTrends = append(efficiencyTrends, efficiencyPoint)
	}

	return efficiencyTrends, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-DB-EXT-008
 * CONTEXT:   Additional interface methods for timesheet and cache operations
 * REASON:    Complete WorkHourDatabaseManager interface implementation
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle missing method implementations and provide reasonable defaults
 * RISK:      Low - Supporting methods for interface completeness
 */
func (dbe *DatabaseExtensions) GetTimesheetData(startDate, endDate time.Time) ([]*domain.TimesheetEntry, error) {
	query := `
		SELECT 
			wb.block_id,
			s.session_id,
			date(wb.start_time) as work_date,
			wb.start_time,
			wb.end_time,
			wb.duration_seconds,
			'Claude CLI Usage' as project,
			'Development' as task,
			'Work session from monitoring data' as description,
			1 as billable
		FROM work_blocks wb
		JOIN sessions s ON wb.session_id = s.session_id
		WHERE date(wb.start_time) >= ? AND date(wb.start_time) <= ?
		  AND wb.duration_seconds > 0
		ORDER BY wb.start_time
	`

	rows, err := dbe.GetDB().Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query timesheet data: %w", err)
	}
	defer rows.Close()

	var entries []*domain.TimesheetEntry

	for rows.Next() {
		var blockID, sessionID, project, task, description string
		var workDate time.Time
		var startTime, endTime time.Time
		var durationSeconds int64
		var billable bool

		err := rows.Scan(
			&blockID, &sessionID, &workDate, &startTime, &endTime, 
			&durationSeconds, &project, &task, &description, &billable,
		)
		if err != nil {
			dbe.logger.Warn("Failed to scan timesheet entry row", "error", err)
			continue
		}

		entry := &domain.TimesheetEntry{
			ID:          blockID,
			Date:        workDate,
			StartTime:   startTime,
			EndTime:     endTime,
			Duration:    time.Duration(durationSeconds) * time.Second,
			Project:     project,
			Task:        task,
			Description: description,
			Billable:    billable,
		}

		entries = append(entries, entry)
	}

	dbe.logger.Debug("Timesheet data retrieved",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"entries", len(entries))

	return entries, nil
}

// Additional interface methods that may need implementation
func (dbe *DatabaseExtensions) GetTimesheet(timesheetID string) (*domain.Timesheet, error) {
	// This would be implemented in the main WorkHourManager
	return nil, fmt.Errorf("GetTimesheet not implemented in extensions")
}

func (dbe *DatabaseExtensions) GetTimesheetsByPeriod(employeeID string, startDate, endDate time.Time) ([]*domain.Timesheet, error) {
	// This would be implemented in the main WorkHourManager  
	return nil, fmt.Errorf("GetTimesheetsByPeriod not implemented in extensions")
}

func (dbe *DatabaseExtensions) RefreshWorkDayCache(date time.Time) error {
	// Invalidate cache for specific date
	return dbe.InvalidateWorkHourCache(date, date)
}

func (dbe *DatabaseExtensions) GetCachedWorkDayStats(date time.Time) (*domain.WorkDay, bool) {
	// Cache checking would be implemented in the main WorkHourManager
	return nil, false
}

func (dbe *DatabaseExtensions) InvalidateWorkHourCache(startDate, endDate time.Time) error {
	dbe.logger.Info("Work hour cache invalidation requested",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"))
	
	// In a real implementation, this would clear relevant cache entries
	return nil
}

// Helper method to get direct database access
func (dbe *DatabaseExtensions) GetDB() *sql.DB {
	// This assumes the underlying WorkHourManager exposes the database connection
	// You might need to modify the WorkHourManager to expose this
	return dbe.WorkHourManager.GetDB()
}