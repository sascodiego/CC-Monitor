/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-015
 * CONTEXT:   Work hour analytics implementation for pattern analysis and productivity metrics
 * REASON:    Business requirement for advanced analytics, trend analysis, and productivity insights
 * CHANGE:    Initial implementation.
 * PREVENTION:Optimize complex analytical queries, implement proper caching for expensive calculations
 * RISK:      Medium - Complex analytics could impact database performance under load
 */
package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-016
 * CONTEXT:   GetWorkMonthData implements monthly work hour aggregation
 * REASON:    Monthly reporting requires aggregation of daily work data for comprehensive analysis
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate month boundaries, handle partial months correctly, optimize for large datasets
 * RISK:      Medium - Month calculations with large datasets could cause performance issues
 */
func (whm *WorkHourManager) GetWorkMonthData(year int, month time.Month) ([]*domain.WorkDay, error) {
	// Calculate month boundaries
	startDate := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1) // Last day of month

	stmt := whm.prepared["getWorkDayRange"]
	rows, err := stmt.Query(
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly work data: %w", err)
	}
	defer rows.Close()

	var workDays []*domain.WorkDay

	for rows.Next() {
		var id, dateKey string
		var startTime, endTime sql.NullTime
		var totalSeconds, breakSeconds, sessionCount, blockCount int64
		var isComplete bool
		var efficiencyRatio float64

		err := rows.Scan(
			&id, &dateKey, &startTime, &endTime,
			&totalSeconds, &breakSeconds, &sessionCount,
			&blockCount, &isComplete, &efficiencyRatio,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monthly work day: %w", err)
		}

		// Parse date
		date, err := time.Parse("2006-01-02", dateKey)
		if err != nil {
			whm.logger.Warn("Failed to parse work day date", "dateKey", dateKey, "error", err)
			continue
		}

		workDay := &domain.WorkDay{
			ID:           id,
			Date:         date,
			TotalTime:    time.Duration(totalSeconds) * time.Second,
			BreakTime:    time.Duration(breakSeconds) * time.Second,
			SessionCount: int(sessionCount),
			BlockCount:   int(blockCount),
			IsComplete:   isComplete,
		}

		if startTime.Valid {
			workDay.StartTime = &startTime.Time
		}
		if endTime.Valid {
			workDay.EndTime = &endTime.Time
		}

		workDays = append(workDays, workDay)
	}

	return workDays, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-017
 * CONTEXT:   GetWorkPatternData implements pattern analysis data retrieval
 * REASON:    Pattern analysis requires detailed activity data points for productivity insights
 * CHANGE:    Initial implementation.
 * PREVENTION:Limit data range to prevent memory exhaustion, implement result pagination for large datasets
 * RISK:      High - Large date ranges could cause memory issues or slow query performance
 */
func (whm *WorkHourManager) GetWorkPatternData(startDate, endDate time.Time) ([]arch.WorkPatternDataPoint, error) {
	// Check if we have cached pattern data
	cacheKey := fmt.Sprintf("%s_%s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	whm.cacheMu.RLock()
	if cached, exists := whm.cache.patterns[cacheKey]; exists {
		if lastUpdate, ok := whm.cache.lastUpdated["pattern_"+cacheKey]; ok {
			if time.Since(lastUpdate) < whm.cache.ttl {
				whm.cacheMu.RUnlock()
				return whm.convertWorkPatternToDataPoints(cached), nil
			}
		}
	}
	whm.cacheMu.RUnlock()

	// Limit date range to prevent performance issues
	maxDays := 90
	if endDate.Sub(startDate) > time.Duration(maxDays)*24*time.Hour {
		endDate = startDate.AddDate(0, 0, maxDays)
		whm.logger.Warn("Limited pattern analysis date range", "maxDays", maxDays)
	}

	query := `
		SELECT 
			wb.start_time,
			wb.duration_seconds,
			CASE 
				WHEN wb.duration_seconds > 3600 THEN 'focused'
				WHEN wb.duration_seconds > 1800 THEN 'productive' 
				WHEN wb.duration_seconds > 900 THEN 'active'
				ELSE 'brief'
			END as activity_type,
			CAST(LEAST(wb.duration_seconds / 3600.0, 1.0) AS REAL) as intensity,
			CAST(CASE 
				WHEN wb.duration_seconds > 2400 THEN 0.9
				WHEN wb.duration_seconds > 1200 THEN 0.7
				WHEN wb.duration_seconds > 600 THEN 0.5
				ELSE 0.3
			END AS REAL) as productivity
		FROM work_blocks wb
		JOIN sessions s ON wb.session_id = s.session_id
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
		ORDER BY wb.start_time
	`

	rows, err := whm.db.Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
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
			return nil, fmt.Errorf("failed to scan pattern data point: %w", err)
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

	return dataPoints, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-018
 * CONTEXT:   GetProductivityMetrics calculates comprehensive productivity metrics
 * REASON:    Business requirement for efficiency analysis and productivity optimization insights
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate metric calculations, handle edge cases like zero work time properly
 * RISK:      Medium - Incorrect productivity calculations could mislead optimization efforts
 */
func (whm *WorkHourManager) GetProductivityMetrics(startDate, endDate time.Time) (*domain.EfficiencyMetrics, error) {
	// Query for productivity calculation data
	query := `
		SELECT 
			SUM(wb.duration_seconds) as total_work_seconds,
			COUNT(wb.block_id) as total_blocks,
			COUNT(DISTINCT s.session_id) as total_sessions,
			MIN(s.start_time) as first_activity,
			MAX(COALESCE(wb.end_time, s.end_time)) as last_activity,
			AVG(wb.duration_seconds) as avg_block_duration,
			-- Calculate breaks as gaps between consecutive blocks
			SUM(CASE WHEN wb.duration_seconds < 300 THEN 1 ELSE 0 END) as short_blocks,
			COUNT(CASE WHEN wb.duration_seconds > 1800 THEN 1 END) as focused_blocks
		FROM work_blocks wb
		JOIN sessions s ON wb.session_id = s.session_id
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
	`

	var totalWorkSeconds, totalBlocks, totalSessions, shortBlocks, focusedBlocks int64
	var firstActivity, lastActivity sql.NullTime
	var avgBlockDuration float64

	err := whm.db.QueryRow(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).Scan(
		&totalWorkSeconds, &totalBlocks, &totalSessions,
		&firstActivity, &lastActivity, &avgBlockDuration,
		&shortBlocks, &focusedBlocks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query productivity metrics: %w", err)
	}

	metrics := &domain.EfficiencyMetrics{}

	// Calculate active ratio (active time / total time span)
	if firstActivity.Valid && lastActivity.Valid {
		totalSpan := lastActivity.Time.Sub(firstActivity.Time)
		if totalSpan > 0 {
			activeTime := time.Duration(totalWorkSeconds) * time.Second
			metrics.ActiveRatio = float64(activeTime) / float64(totalSpan)
		}
	}

	// Calculate focus score (percentage of long blocks)
	if totalBlocks > 0 {
		metrics.FocusScore = float64(focusedBlocks) / float64(totalBlocks)
	}

	// Calculate interruption rate (short blocks per hour)
	if totalWorkSeconds > 0 {
		totalWorkHours := float64(totalWorkSeconds) / 3600.0
		metrics.InterruptionRate = float64(shortBlocks) / totalWorkHours
	}

	// Determine peak efficiency period
	peakHour, err := whm.findPeakEfficiencyHour(startDate, endDate)
	if err != nil {
		whm.logger.Warn("Failed to determine peak efficiency hour", "error", err)
		metrics.PeakEfficiency = "Unknown"
	} else {
		metrics.PeakEfficiency = peakHour
	}

	return metrics, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-019
 * CONTEXT:   findPeakEfficiencyHour identifies the most productive hour of day
 * REASON:    Peak efficiency analysis helps optimize work scheduling and productivity
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle cases with insufficient data, validate hour calculations
 * RISK:      Low - Peak hour calculation failure doesn't affect core functionality
 */
func (whm *WorkHourManager) findPeakEfficiencyHour(startDate, endDate time.Time) (string, error) {
	stmt := whm.prepared["getHourlyProductivity"]
	rows, err := stmt.Query(startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return "", fmt.Errorf("failed to query hourly productivity: %w", err)
	}
	defer rows.Close()

	maxProductivity := 0.0
	peakHour := 9 // Default to 9 AM

	for rows.Next() {
		var hour, blockCount, totalSeconds int64
		var avgDuration float64

		err := rows.Scan(&hour, &blockCount, &totalSeconds, &avgDuration)
		if err != nil {
			continue
		}

		// Calculate productivity score: average duration * block count
		productivity := avgDuration * float64(blockCount)
		
		if productivity > maxProductivity {
			maxProductivity = productivity
			peakHour = int(hour)
		}
	}

	// Format peak hour as readable string
	return fmt.Sprintf("%02d:00-%02d:00", peakHour, (peakHour+1)%24), nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-020
 * CONTEXT:   GetBreakPatterns analyzes break timing and duration patterns
 * REASON:    Break pattern analysis helps optimize work schedules and productivity
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate break calculations, handle edge cases with consecutive blocks
 * RISK:      Low - Break pattern analysis is informational and doesn't affect core operations
 */
func (whm *WorkHourManager) GetBreakPatterns(startDate, endDate time.Time) ([]domain.BreakPattern, error) {
	// Query to find gaps between work blocks (breaks)
	query := `
		WITH ordered_blocks AS (
			SELECT 
				wb.start_time,
				wb.end_time,
				LAG(wb.end_time) OVER (ORDER BY wb.start_time) as prev_end_time,
				CAST(strftime('%H', wb.start_time) AS INTEGER) as start_hour
			FROM work_blocks wb
			JOIN sessions s ON wb.session_id = s.session_id
			WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
			  AND wb.end_time IS NOT NULL
		),
		breaks AS (
			SELECT 
				start_hour,
				(julianday(start_time) - julianday(prev_end_time)) * 24 * 3600 as break_duration_seconds
			FROM ordered_blocks
			WHERE prev_end_time IS NOT NULL
			  AND (julianday(start_time) - julianday(prev_end_time)) * 24 * 3600 BETWEEN 300 AND 14400 -- 5 min to 4 hours
		)
		SELECT 
			start_hour,
			AVG(break_duration_seconds) as avg_duration,
			COUNT(*) as frequency_count,
			CASE 
				WHEN AVG(break_duration_seconds) < 1800 THEN 'short'
				WHEN AVG(break_duration_seconds) < 5400 THEN 'lunch'
				ELSE 'long'
			END as break_type
		FROM breaks
		GROUP BY start_hour
		HAVING COUNT(*) >= 2  -- Only include patterns that occur at least twice
		ORDER BY frequency_count DESC
	`

	rows, err := whm.db.Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query break patterns: %w", err)
	}
	defer rows.Close()

	var patterns []domain.BreakPattern
	totalBreaks := 0

	// First pass: count total breaks for frequency calculation
	tempPatterns := make([]struct {
		hour        int
		duration    time.Duration
		count       int
		breakType   string
	}, 0)

	for rows.Next() {
		var hour, frequencyCount int64
		var avgDuration float64
		var breakType string

		err := rows.Scan(&hour, &avgDuration, &frequencyCount, &breakType)
		if err != nil {
			continue
		}

		tempPatterns = append(tempPatterns, struct {
			hour        int
			duration    time.Duration
			count       int
			breakType   string
		}{
			hour:      int(hour),
			duration:  time.Duration(avgDuration) * time.Second,
			count:     int(frequencyCount),
			breakType: breakType,
		})

		totalBreaks += int(frequencyCount)
	}

	// Second pass: calculate frequencies and create final patterns
	for _, temp := range tempPatterns {
		frequency := float64(temp.count) / float64(totalBreaks)
		
		pattern := domain.BreakPattern{
			StartHour: temp.hour,
			Duration:  temp.duration,
			Frequency: frequency,
			Type:      domain.BreakType(temp.breakType),
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-021
 * CONTEXT:   Trend analysis methods for work time, sessions, and efficiency trends
 * REASON:    Business requirement for trend analysis to identify patterns and optimization opportunities
 * CHANGE:    Initial implementation.
 * PREVENTION:Optimize trend queries for large datasets, implement proper aggregation logic
 * RISK:      Medium - Trend calculations with large datasets could impact performance
 */
func (whm *WorkHourManager) GetWorkTimeTrends(startDate, endDate time.Time, granularity arch.Granularity) ([]arch.TrendDataPoint, error) {
	var groupBy string
	
	switch granularity {
	case arch.GranularityDay:
		groupBy = "date(s.start_time)"
	case arch.GranularityWeek:
		groupBy = "date(s.start_time, 'weekday 1', '-6 days')" // Monday of week
	case arch.GranularityMonth:
		groupBy = "strftime('%Y-%m', s.start_time)"
	default:
		return nil, fmt.Errorf("unsupported granularity: %v", granularity)
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as period,
			SUM(wb.duration_seconds) as total_work_seconds
		FROM sessions s
		LEFT JOIN work_blocks wb ON s.session_id = wb.session_id
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
		GROUP BY %s
		ORDER BY period
	`, groupBy, groupBy)

	rows, err := whm.db.Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query work time trends: %w", err)
	}
	defer rows.Close()

	var dataPoints []arch.TrendDataPoint
	var previousValue float64
	baseline := 0.0
	count := 0

	for rows.Next() {
		var period string
		var totalWorkSeconds sql.NullInt64

		err := rows.Scan(&period, &totalWorkSeconds)
		if err != nil {
			continue
		}

		// Parse date based on granularity
		var date time.Time
		switch granularity {
		case arch.GranularityDay:
			date, _ = time.Parse("2006-01-02", period)
		case arch.GranularityWeek:
			date, _ = time.Parse("2006-01-02", period)
		case arch.GranularityMonth:
			date, _ = time.Parse("2006-01", period)
		}

		value := 0.0
		if totalWorkSeconds.Valid {
			value = float64(totalWorkSeconds.Int64) / 3600.0 // Convert to hours
		}

		// Calculate moving baseline (running average)
		count++
		baseline = ((baseline * float64(count-1)) + value) / float64(count)

		// Calculate percentage change from previous period
		change := 0.0
		if previousValue > 0 {
			change = ((value - previousValue) / previousValue) * 100
		}

		dataPoint := arch.TrendDataPoint{
			Date:     date,
			Value:    value,
			Baseline: baseline,
			Change:   change,
		}

		dataPoints = append(dataPoints, dataPoint)
		previousValue = value
	}

	return dataPoints, nil
}

func (whm *WorkHourManager) GetSessionCountTrends(startDate, endDate time.Time, granularity arch.Granularity) ([]arch.TrendDataPoint, error) {
	// Similar implementation to GetWorkTimeTrends but counting sessions
	var groupBy string
	
	switch granularity {
	case arch.GranularityDay:
		groupBy = "date(s.start_time)"
	case arch.GranularityWeek:
		groupBy = "date(s.start_time, 'weekday 1', '-6 days')"
	case arch.GranularityMonth:
		groupBy = "strftime('%Y-%m', s.start_time)"
	default:
		return nil, fmt.Errorf("unsupported granularity: %v", granularity)
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as period,
			COUNT(DISTINCT s.session_id) as session_count
		FROM sessions s
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
		GROUP BY %s
		ORDER BY period
	`, groupBy, groupBy)

	rows, err := whm.db.Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query session count trends: %w", err)
	}
	defer rows.Close()

	var dataPoints []arch.TrendDataPoint
	var previousValue float64
	baseline := 0.0
	count := 0

	for rows.Next() {
		var period string
		var sessionCount int64

		err := rows.Scan(&period, &sessionCount)
		if err != nil {
			continue
		}

		// Parse date based on granularity
		var date time.Time
		switch granularity {
		case arch.GranularityDay:
			date, _ = time.Parse("2006-01-02", period)
		case arch.GranularityWeek:
			date, _ = time.Parse("2006-01-02", period)
		case arch.GranularityMonth:
			date, _ = time.Parse("2006-01", period)
		}

		value := float64(sessionCount)

		// Calculate moving baseline
		count++
		baseline = ((baseline * float64(count-1)) + value) / float64(count)

		// Calculate percentage change
		change := 0.0
		if previousValue > 0 {
			change = ((value - previousValue) / previousValue) * 100
		}

		dataPoint := arch.TrendDataPoint{
			Date:     date,
			Value:    value,
			Baseline: baseline,
			Change:   change,
		}

		dataPoints = append(dataPoints, dataPoint)
		previousValue = value
	}

	return dataPoints, nil
}

func (whm *WorkHourManager) GetEfficiencyTrends(startDate, endDate time.Time, granularity arch.Granularity) ([]arch.TrendDataPoint, error) {
	// Efficiency trends based on active ratio calculations
	var groupBy string
	
	switch granularity {
	case arch.GranularityDay:
		groupBy = "date(s.start_time)"
	case arch.GranularityWeek:
		groupBy = "date(s.start_time, 'weekday 1', '-6 days')"
	case arch.GranularityMonth:
		groupBy = "strftime('%Y-%m', s.start_time)"
	default:
		return nil, fmt.Errorf("unsupported granularity: %v", granularity)
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as period,
			SUM(wb.duration_seconds) as total_work_seconds,
			MIN(s.start_time) as first_activity,
			MAX(COALESCE(wb.end_time, s.end_time)) as last_activity
		FROM sessions s
		LEFT JOIN work_blocks wb ON s.session_id = wb.session_id
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
		GROUP BY %s
		ORDER BY period
	`, groupBy, groupBy)

	rows, err := whm.db.Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query efficiency trends: %w", err)
	}
	defer rows.Close()

	var dataPoints []arch.TrendDataPoint
	var previousValue float64
	baseline := 0.0
	count := 0

	for rows.Next() {
		var period string
		var totalWorkSeconds sql.NullInt64
		var firstActivity, lastActivity sql.NullTime

		err := rows.Scan(&period, &totalWorkSeconds, &firstActivity, &lastActivity)
		if err != nil {
			continue
		}

		// Parse date based on granularity
		var date time.Time
		switch granularity {
		case arch.GranularityDay:
			date, _ = time.Parse("2006-01-02", period)
		case arch.GranularityWeek:
			date, _ = time.Parse("2006-01-02", period)
		case arch.GranularityMonth:
			date, _ = time.Parse("2006-01", period)
		}

		// Calculate efficiency ratio
		value := 0.0
		if totalWorkSeconds.Valid && firstActivity.Valid && lastActivity.Valid {
			workTime := time.Duration(totalWorkSeconds.Int64) * time.Second
			totalSpan := lastActivity.Time.Sub(firstActivity.Time)
			if totalSpan > 0 {
				value = float64(workTime) / float64(totalSpan)
			}
		}

		// Calculate moving baseline
		count++
		baseline = ((baseline * float64(count-1)) + value) / float64(count)

		// Calculate percentage change
		change := 0.0
		if previousValue > 0 {
			change = ((value - previousValue) / previousValue) * 100
		}

		dataPoint := arch.TrendDataPoint{
			Date:     date,
			Value:    value,
			Baseline: baseline,
			Change:   change,
		}

		dataPoints = append(dataPoints, dataPoint)
		previousValue = value
	}

	return dataPoints, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-022
 * CONTEXT:   Timesheet management methods for formal time tracking
 * REASON:    Business requirement for professional timesheet generation and management
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate timesheet data integrity, ensure policy compliance, handle concurrency properly
 * RISK:      High - Timesheet accuracy is critical for billing and compliance requirements
 */
func (whm *WorkHourManager) SaveTimesheet(timesheet *domain.Timesheet) error {
	whm.mu.Lock()
	defer whm.mu.Unlock()

	tx, err := whm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin timesheet transaction: %w", err)
	}
	defer tx.Rollback()

	// Serialize policy data to JSON
	policyData, err := json.Marshal(timesheet.Policy)
	if err != nil {
		return fmt.Errorf("failed to serialize timesheet policy: %w", err)
	}

	// Insert or update timesheet
	stmt := whm.prepared["createTimesheet"]
	_, err = tx.Stmt(stmt).Exec(
		timesheet.ID,
		timesheet.EmployeeID,
		string(timesheet.Period),
		timesheet.StartDate.Format("2006-01-02"),
		timesheet.EndDate.Format("2006-01-02"),
		string(policyData),
		timesheet.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save timesheet: %w", err)
	}

	// Save timesheet entries
	for _, entry := range timesheet.Entries {
		_, err = tx.Exec(`
			INSERT OR REPLACE INTO timesheet_entries (
				id, timesheet_id, date_key, start_time, end_time,
				duration_seconds, project, task, description, billable
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID, timesheet.ID, entry.Date.Format("2006-01-02"),
			entry.StartTime, entry.EndTime, int64(entry.Duration.Seconds()),
			entry.Project, entry.Task, entry.Description, entry.Billable,
		)
		if err != nil {
			return fmt.Errorf("failed to save timesheet entry: %w", err)
		}
	}

	// Update timesheet totals
	updateStmt := whm.prepared["updateTimesheet"]
	_, err = tx.Stmt(updateStmt).Exec(
		int64(timesheet.TotalHours.Seconds()),
		int64(timesheet.RegularHours.Seconds()),
		int64(timesheet.OvertimeHours.Seconds()),
		string(timesheet.Status),
		time.Now(),
		timesheet.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update timesheet totals: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit timesheet transaction: %w", err)
	}

	whm.logger.Info("Timesheet saved", "timesheetID", timesheet.ID, "period", timesheet.Period)
	return nil
}

func (whm *WorkHourManager) GetTimesheetData(startDate, endDate time.Time) ([]*domain.TimesheetEntry, error) {
	query := `
		SELECT 
			wb.block_id as id,
			wb.session_id,
			date(s.start_time) as date_key,
			wb.start_time,
			COALESCE(wb.end_time, datetime(wb.start_time, '+' || wb.duration_seconds || ' seconds')) as end_time,
			wb.duration_seconds,
			'Claude CLI Usage' as project,
			'Development' as task,
			'Automated time tracking' as description,
			1 as billable
		FROM work_blocks wb
		JOIN sessions s ON wb.session_id = s.session_id
		WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
		ORDER BY wb.start_time
	`

	rows, err := whm.db.Query(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query timesheet data: %w", err)
	}
	defer rows.Close()

	var entries []*domain.TimesheetEntry

	for rows.Next() {
		var id, sessionID, dateKey, project, task, description string
		var startTime, endTime time.Time
		var durationSeconds int64
		var billable bool

		err := rows.Scan(
			&id, &sessionID, &dateKey, &startTime, &endTime,
			&durationSeconds, &project, &task, &description, &billable,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timesheet entry: %w", err)
		}

		// Parse date
		date, err := time.Parse("2006-01-02", dateKey)
		if err != nil {
			continue
		}

		entry := &domain.TimesheetEntry{
			ID:          id,
			Date:        date,
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

	return entries, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-023
 * CONTEXT:   Cache management methods for work hour performance optimization
 * REASON:    Work hour calculations can be expensive, proper cache management improves system performance
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement cache size limits and proper TTL management to prevent memory issues
 * RISK:      Low - Cache management failures only affect performance, not functionality
 */
func (whm *WorkHourManager) RefreshWorkDayCache(date time.Time) error {
	dateKey := date.Format("2006-01-02")
	
	// Remove from cache to force recalculation
	whm.cacheMu.Lock()
	delete(whm.cache.workDays, dateKey)
	delete(whm.cache.lastUpdated, dateKey)
	whm.cacheMu.Unlock()

	// Recalculate and cache
	_, err := whm.GetWorkDayData(date)
	return err
}

func (whm *WorkHourManager) GetCachedWorkDayStats(date time.Time) (*domain.WorkDay, bool) {
	dateKey := date.Format("2006-01-02")
	
	whm.cacheMu.RLock()
	defer whm.cacheMu.RUnlock()
	
	if cached, exists := whm.cache.workDays[dateKey]; exists {
		if lastUpdate, ok := whm.cache.lastUpdated[dateKey]; ok {
			if time.Since(lastUpdate) < whm.cache.ttl {
				return cached, true
			}
		}
	}
	
	return nil, false
}

func (whm *WorkHourManager) InvalidateWorkHourCache(startDate, endDate time.Time) error {
	whm.cacheMu.Lock()
	defer whm.cacheMu.Unlock()

	// Iterate through date range and remove cached entries
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		delete(whm.cache.workDays, dateKey)
		delete(whm.cache.lastUpdated, dateKey)
		
		// Also invalidate week cache if this date falls in a week boundary
		weekKey := d.Format("2006-01-02")
		for d.Weekday() != time.Monday {
			d = d.AddDate(0, 0, -1)
		}
		weekKey = d.Format("2006-01-02")
		delete(whm.cache.workWeeks, weekKey)
		delete(whm.cache.lastUpdated, "week_"+weekKey)
	}

	whm.logger.Debug("Work hour cache invalidated", "startDate", startDate, "endDate", endDate)
	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-024
 * CONTEXT:   Helper method to convert WorkPattern to WorkPatternDataPoint slice
 * REASON:    Need to convert cached pattern data to interface format for consistency
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle nil patterns gracefully, validate data conversion
 * RISK:      Low - Data conversion errors don't affect core functionality
 */
func (whm *WorkHourManager) convertWorkPatternToDataPoints(pattern *domain.WorkPattern) []arch.WorkPatternDataPoint {
	var dataPoints []arch.WorkPatternDataPoint
	
	// Convert break patterns to data points
	for _, breakPattern := range pattern.BreakPatterns {
		dataPoint := arch.WorkPatternDataPoint{
			Timestamp:    time.Date(2023, 1, 1, breakPattern.StartHour, 0, 0, 0, time.UTC), // Use fixed date for hour-based pattern
			Duration:     breakPattern.Duration,
			ActivityType: string(breakPattern.Type),
			Intensity:    breakPattern.Frequency,
			Productivity: 0.5, // Default productivity for breaks
		}
		dataPoints = append(dataPoints, dataPoint)
	}
	
	return dataPoints
}