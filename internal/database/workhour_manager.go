/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-001
 * CONTEXT:   Work hour database manager extending Kùzu schema for comprehensive work hour analytics
 * REASON:    Need specialized database operations for work hour tracking, reporting, and analytics beyond basic session/work block storage
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate all time calculations, ensure timezone consistency, implement proper caching for performance
 * RISK:      High - Work hour calculations must be accurate for billing and compliance requirements
 */
package database

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-002
 * CONTEXT:   WorkHourManager implements WorkHourDatabaseManager interface with optimized queries
 * REASON:    Need dedicated manager for work hour analytics with caching and performance optimizations
 * CHANGE:    Initial implementation.
 * PREVENTION:Monitor query performance, implement proper cache invalidation, validate aggregation logic
 * RISK:      Medium - Poor query performance could impact system responsiveness
 */
type WorkHourManager struct {
	*KuzuManager                     // Embed base database manager
	cache        *WorkHourCache      // Cached work hour data
	cacheMu      sync.RWMutex        // Cache access mutex
	logger       arch.Logger         // Logger instance
	prepared     map[string]*sql.Stmt // Work hour specific prepared statements
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-003
 * CONTEXT:   WorkHourCache provides in-memory caching for frequently accessed work hour data
 * REASON:    Work hour calculations can be expensive, caching improves performance for reporting
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement TTL expiration and cache size limits to prevent memory exhaustion
 * RISK:      Low - Cache misses don't affect functionality, only performance
 */
type WorkHourCache struct {
	workDays       map[string]*domain.WorkDay    // Date-keyed work day cache
	workWeeks      map[string]*domain.WorkWeek   // Week-keyed work week cache
	patterns       map[string]*domain.WorkPattern // Date range keyed patterns
	timesheets     map[string]*domain.Timesheet  // Timesheet cache
	lastUpdated    map[string]time.Time          // Cache entry timestamps
	maxSize        int                           // Maximum cache entries
	ttl            time.Duration                 // Time to live for cache entries
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-004
 * CONTEXT:   NewWorkHourManager creates work hour manager with extended schema and caching
 * REASON:    Factory pattern for proper initialization with work hour specific schema and optimizations
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure schema migration is backward compatible, validate cache configuration
 * RISK:      Medium - Schema initialization failure could prevent work hour functionality
 */
func NewWorkHourManager(dbPath string, logger arch.Logger) (*WorkHourManager, error) {
	// Initialize base Kuzu manager
	baseManager, err := NewKuzuManager(dbPath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize base manager: %w", err)
	}

	whm := &WorkHourManager{
		KuzuManager: baseManager,
		cache: &WorkHourCache{
			workDays:    make(map[string]*domain.WorkDay),
			workWeeks:   make(map[string]*domain.WorkWeek),
			patterns:    make(map[string]*domain.WorkPattern),
			timesheets:  make(map[string]*domain.Timesheet),
			lastUpdated: make(map[string]time.Time),
			maxSize:     1000,
			ttl:         1 * time.Hour,
		},
		logger:   logger,
		prepared: make(map[string]*sql.Stmt),
	}

	// Initialize work hour schema extensions
	if err := whm.initializeWorkHourSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize work hour schema: %w", err)
	}

	// Prepare work hour specific statements
	if err := whm.prepareWorkHourStatements(); err != nil {
		return nil, fmt.Errorf("failed to prepare work hour statements: %w", err)
	}

	logger.Info("Work hour database manager initialized", 
		"cacheSize", whm.cache.maxSize,
		"cacheTTL", whm.cache.ttl)

	return whm, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-005
 * CONTEXT:   initializeWorkHourSchema extends existing schema with work hour tracking tables
 * REASON:    Need additional tables for work days, work weeks, timesheets, and analytics
 * CHANGE:    Initial implementation.
 * PREVENTION:Use IF NOT EXISTS to avoid conflicts, ensure foreign key relationships are correct
 * RISK:      High - Schema errors could corrupt existing data or prevent new functionality
 */
func (whm *WorkHourManager) initializeWorkHourSchema() error {
	workHourSchema := `
		-- Work days table (daily aggregations)
		CREATE TABLE IF NOT EXISTS work_days (
			id TEXT PRIMARY KEY,
			date_key TEXT NOT NULL UNIQUE, -- YYYY-MM-DD format for indexing
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			total_time_seconds INTEGER DEFAULT 0,
			break_time_seconds INTEGER DEFAULT 0,
			session_count INTEGER DEFAULT 0,
			block_count INTEGER DEFAULT 0,
			is_complete BOOLEAN DEFAULT FALSE,
			efficiency_ratio REAL DEFAULT 0.0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		-- Work weeks table (weekly aggregations)
		CREATE TABLE IF NOT EXISTS work_weeks (
			id TEXT PRIMARY KEY,
			week_start TEXT NOT NULL, -- YYYY-MM-DD format for week start (Monday)
			week_end TEXT NOT NULL,
			total_time_seconds INTEGER DEFAULT 0,
			overtime_seconds INTEGER DEFAULT 0,
			average_day_seconds INTEGER DEFAULT 0,
			standard_hours_seconds INTEGER DEFAULT 144000, -- 40 hours
			is_complete BOOLEAN DEFAULT FALSE,
			work_days_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		-- Timesheets table (formal timesheet entries)
		CREATE TABLE IF NOT EXISTS timesheets (
			id TEXT PRIMARY KEY,
			employee_id TEXT NOT NULL DEFAULT 'default',
			period TEXT NOT NULL, -- 'weekly', 'biweekly', 'monthly'
			start_date TEXT NOT NULL, -- YYYY-MM-DD
			end_date TEXT NOT NULL, -- YYYY-MM-DD
			total_hours_seconds INTEGER DEFAULT 0,
			regular_hours_seconds INTEGER DEFAULT 0,
			overtime_hours_seconds INTEGER DEFAULT 0,
			status TEXT DEFAULT 'draft', -- 'draft', 'submitted', 'approved'
			policy_data TEXT, -- JSON serialized TimesheetPolicy
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			submitted_at TIMESTAMP
		);

		-- Timesheet entries table (individual time entries)
		CREATE TABLE IF NOT EXISTS timesheet_entries (
			id TEXT PRIMARY KEY,
			timesheet_id TEXT NOT NULL,
			date_key TEXT NOT NULL, -- YYYY-MM-DD
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP NOT NULL,
			duration_seconds INTEGER NOT NULL,
			project TEXT DEFAULT 'Claude CLI Usage',
			task TEXT DEFAULT 'Development',
			description TEXT,
			billable BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (timesheet_id) REFERENCES timesheets(id)
		);

		-- Work patterns cache table (pattern analysis results)
		CREATE TABLE IF NOT EXISTS work_patterns (
			id TEXT PRIMARY KEY,
			start_date TEXT NOT NULL, -- YYYY-MM-DD
			end_date TEXT NOT NULL, -- YYYY-MM-DD
			peak_hours TEXT, -- JSON array of peak hours
			productivity_curve TEXT, -- JSON array of hourly productivity scores
			work_day_type TEXT DEFAULT 'standard',
			consistency_score REAL DEFAULT 0.0,
			break_patterns TEXT, -- JSON array of BreakPattern objects
			weekly_pattern TEXT, -- JSON array of weekly durations
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP DEFAULT (datetime('now', '+24 hours'))
		);

		-- Activity summaries cache table (comprehensive metrics)
		CREATE TABLE IF NOT EXISTS activity_summaries (
			id TEXT PRIMARY KEY,
			period TEXT NOT NULL, -- 'daily', 'weekly', 'monthly'
			start_date TEXT NOT NULL, -- YYYY-MM-DD
			end_date TEXT NOT NULL, -- YYYY-MM-DD
			total_work_time_seconds INTEGER DEFAULT 0,
			total_sessions INTEGER DEFAULT 0,
			total_work_blocks INTEGER DEFAULT 0,
			average_session_seconds INTEGER DEFAULT 0,
			average_work_block_seconds INTEGER DEFAULT 0,
			daily_average_seconds INTEGER DEFAULT 0,
			trends_data TEXT, -- JSON serialized TrendAnalysis
			goals_data TEXT, -- JSON serialized GoalProgress
			efficiency_data TEXT, -- JSON serialized EfficiencyMetrics
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP DEFAULT (datetime('now', '+1 hour'))
		);

		-- Work hour indexes for optimal query performance
		CREATE INDEX IF NOT EXISTS idx_work_days_date ON work_days(date_key);
		CREATE INDEX IF NOT EXISTS idx_work_days_complete ON work_days(is_complete);
		CREATE INDEX IF NOT EXISTS idx_work_weeks_start ON work_weeks(week_start);
		CREATE INDEX IF NOT EXISTS idx_timesheets_employee_period ON timesheets(employee_id, period, start_date);
		CREATE INDEX IF NOT EXISTS idx_timesheet_entries_timesheet ON timesheet_entries(timesheet_id);
		CREATE INDEX IF NOT EXISTS idx_timesheet_entries_date ON timesheet_entries(date_key);
		CREATE INDEX IF NOT EXISTS idx_work_patterns_dates ON work_patterns(start_date, end_date);
		CREATE INDEX IF NOT EXISTS idx_activity_summaries_period ON activity_summaries(period, start_date, end_date);
		CREATE INDEX IF NOT EXISTS idx_work_patterns_expires ON work_patterns(expires_at);
		CREATE INDEX IF NOT EXISTS idx_activity_summaries_expires ON activity_summaries(expires_at);

		-- Triggers for cache cleanup
		CREATE TRIGGER IF NOT EXISTS cleanup_expired_patterns
		AFTER INSERT ON work_patterns
		BEGIN
			DELETE FROM work_patterns WHERE expires_at < datetime('now');
		END;

		CREATE TRIGGER IF NOT EXISTS cleanup_expired_summaries
		AFTER INSERT ON activity_summaries
		BEGIN
			DELETE FROM activity_summaries WHERE expires_at < datetime('now');
		END;
	`

	_, err := whm.db.Exec(workHourSchema)
	if err != nil {
		return fmt.Errorf("failed to create work hour schema: %w", err)
	}

	whm.logger.Info("Work hour schema extensions initialized successfully")
	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-006
 * CONTEXT:   prepareWorkHourStatements prepares optimized SQL statements for work hour operations
 * REASON:    Prepared statements improve performance and security for frequently used queries
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate all SQL statements for syntax and security, use parameterized queries
 * RISK:      Medium - SQL errors could cause query failures or security vulnerabilities
 */
func (whm *WorkHourManager) prepareWorkHourStatements() error {
	statements := map[string]string{
		// Work day operations
		"upsertWorkDay": `
			INSERT OR REPLACE INTO work_days (
				id, date_key, start_time, end_time, total_time_seconds, 
				break_time_seconds, session_count, block_count, is_complete, 
				efficiency_ratio, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,

		"getWorkDay": `
			SELECT id, date_key, start_time, end_time, total_time_seconds,
			       break_time_seconds, session_count, block_count, is_complete,
			       efficiency_ratio, created_at, updated_at
			FROM work_days WHERE date_key = ?`,

		"getWorkDayRange": `
			SELECT id, date_key, start_time, end_time, total_time_seconds,
			       break_time_seconds, session_count, block_count, is_complete,
			       efficiency_ratio
			FROM work_days 
			WHERE date_key >= ? AND date_key <= ?
			ORDER BY date_key`,

		// Work week operations
		"upsertWorkWeek": `
			INSERT OR REPLACE INTO work_weeks (
				id, week_start, week_end, total_time_seconds, overtime_seconds,
				average_day_seconds, standard_hours_seconds, is_complete,
				work_days_count, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,

		"getWorkWeek": `
			SELECT id, week_start, week_end, total_time_seconds, overtime_seconds,
			       average_day_seconds, standard_hours_seconds, is_complete,
			       work_days_count
			FROM work_weeks WHERE week_start = ?`,

		// Timesheet operations
		"createTimesheet": `
			INSERT INTO timesheets (
				id, employee_id, period, start_date, end_date,
				policy_data, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,

		"updateTimesheet": `
			UPDATE timesheets SET 
				total_hours_seconds = ?, regular_hours_seconds = ?,
				overtime_hours_seconds = ?, status = ?, updated_at = ?
			WHERE id = ?`,

		"getTimesheet": `
			SELECT id, employee_id, period, start_date, end_date,
			       total_hours_seconds, regular_hours_seconds, overtime_hours_seconds,
			       status, policy_data, created_at, submitted_at
			FROM timesheets WHERE id = ?`,

		// Aggregation queries for work hour analytics
		"getDailyWorkData": `
			SELECT 
				s.session_id,
				s.start_time,
				s.end_time,
				wb.block_id,
				wb.start_time,
				wb.end_time,
				wb.duration_seconds
			FROM sessions s
			LEFT JOIN work_blocks wb ON s.session_id = wb.session_id
			WHERE date(s.start_time) = ?
			ORDER BY s.start_time, wb.start_time`,

		"getWorkTimeByDateRange": `
			SELECT 
				date(s.start_time) as work_date,
				COUNT(DISTINCT s.session_id) as session_count,
				COUNT(wb.block_id) as block_count,
				SUM(wb.duration_seconds) as total_work_seconds,
				MIN(s.start_time) as first_activity,
				MAX(COALESCE(wb.end_time, s.end_time)) as last_activity
			FROM sessions s
			LEFT JOIN work_blocks wb ON s.session_id = wb.session_id
			WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
			GROUP BY date(s.start_time)
			ORDER BY work_date`,

		"getHourlyProductivity": `
			SELECT 
				CAST(strftime('%H', wb.start_time) AS INTEGER) as hour_of_day,
				COUNT(*) as block_count,
				SUM(wb.duration_seconds) as total_seconds,
				AVG(wb.duration_seconds) as avg_duration
			FROM work_blocks wb
			JOIN sessions s ON wb.session_id = s.session_id
			WHERE date(s.start_time) >= ? AND date(s.start_time) <= ?
			GROUP BY hour_of_day
			ORDER BY hour_of_day`,
	}

	for name, query := range statements {
		stmt, err := whm.db.Prepare(query)
		if err != nil {
			return fmt.Errorf("failed to prepare statement %s: %w", name, err)
		}
		whm.prepared[name] = stmt
	}

	whm.logger.Info("Work hour prepared statements initialized", "count", len(statements))
	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-007
 * CONTEXT:   GetWorkDayData implements WorkHourDatabaseManager interface for daily aggregation
 * REASON:    Business requirement for comprehensive daily work hour analysis and reporting
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate date format, handle timezone consistently, cache results for performance
 * RISK:      Medium - Incorrect aggregations could affect work hour reporting accuracy
 */
func (whm *WorkHourManager) GetWorkDayData(date time.Time) (*domain.WorkDay, error) {
	dateKey := date.Format("2006-01-02")
	
	// Check cache first
	whm.cacheMu.RLock()
	if cached, exists := whm.cache.workDays[dateKey]; exists {
		if lastUpdate, ok := whm.cache.lastUpdated[dateKey]; ok {
			if time.Since(lastUpdate) < whm.cache.ttl {
				whm.cacheMu.RUnlock()
				return cached, nil
			}
		}
	}
	whm.cacheMu.RUnlock()

	// Check if already aggregated in database
	stmt := whm.prepared["getWorkDay"]
	var workDay *domain.WorkDay
	
	err := stmt.QueryRow(dateKey).Scan(
		&workDay.ID, &workDay.Date, &workDay.StartTime, &workDay.EndTime,
		&workDay.TotalTime, &workDay.BreakTime, &workDay.SessionCount,
		&workDay.BlockCount, &workDay.IsComplete, new(interface{}), // efficiency_ratio
		new(interface{}), new(interface{}), // created_at, updated_at
	)
	
	if err == nil {
		// Found in database, update cache
		whm.updateWorkDayCache(dateKey, workDay)
		return workDay, nil
	}
	
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query work day: %w", err)
	}

	// Not found, calculate from raw data
	workDay, err = whm.calculateWorkDayFromRaw(date)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate work day: %w", err)
	}

	// Save calculated work day
	if err := whm.SaveWorkDay(workDay); err != nil {
		whm.logger.Warn("Failed to save calculated work day", "date", dateKey, "error", err)
	}

	// Update cache
	whm.updateWorkDayCache(dateKey, workDay)

	return workDay, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-008
 * CONTEXT:   calculateWorkDayFromRaw computes work day metrics from session and work block data
 * REASON:    Need to aggregate raw monitoring data into business-relevant work day metrics
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate time calculations, handle edge cases like spans across midnight
 * RISK:      High - Incorrect calculations affect billing and reporting accuracy
 */
func (whm *WorkHourManager) calculateWorkDayFromRaw(date time.Time) (*domain.WorkDay, error) {
	dateKey := date.Format("2006-01-02")
	
	stmt := whm.prepared["getDailyWorkData"]
	rows, err := stmt.Query(dateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily work data: %w", err)
	}
	defer rows.Close()

	workDay := domain.NewWorkDay(date)
	sessionMap := make(map[string]bool)
	var lastBlockEnd *time.Time
	var firstBlockStart *time.Time
	totalBreakTime := time.Duration(0)

	for rows.Next() {
		var sessionID, blockID sql.NullString
		var sessionStart, sessionEnd, blockStart, blockEnd sql.NullTime
		var blockDuration sql.NullInt64

		err := rows.Scan(
			&sessionID, &sessionStart, &sessionEnd,
			&blockID, &blockStart, &blockEnd, &blockDuration,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan work data row: %w", err)
		}

		// Count unique sessions
		if sessionID.Valid {
			sessionMap[sessionID.String] = true
		}

		// Process work blocks
		if blockID.Valid && blockStart.Valid && blockDuration.Valid {
			workDay.BlockCount++
			
			// Track total work time
			workDay.TotalTime += time.Duration(blockDuration.Int64) * time.Second
			
			// Track day boundaries
			workDay.UpdateActivity(blockStart.Time)
			if blockEnd.Valid {
				workDay.UpdateActivity(blockEnd.Time)
			}

			// Calculate break time between consecutive blocks
			if firstBlockStart == nil {
				firstBlockStart = &blockStart.Time
			}
			
			if lastBlockEnd != nil && blockStart.Valid {
				breakDuration := blockStart.Time.Sub(*lastBlockEnd)
				if breakDuration > 0 && breakDuration < 4*time.Hour { // Reasonable break threshold
					totalBreakTime += breakDuration
				}
			}
			
			if blockEnd.Valid {
				lastBlockEnd = &blockEnd.Time
			}
		}
	}

	workDay.SessionCount = len(sessionMap)
	workDay.BreakTime = totalBreakTime
	
	// Mark as complete if day is past
	now := time.Now()
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, date.Location())
	workDay.IsComplete = now.After(dayEnd)

	return workDay, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-009
 * CONTEXT:   SaveWorkDay persists calculated work day data with proper validation
 * REASON:    Need to store aggregated work day data for efficient reporting and caching
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate work day data consistency, handle concurrent updates properly
 * RISK:      Medium - Data corruption if concurrent updates not handled properly
 */
func (whm *WorkHourManager) SaveWorkDay(workDay *domain.WorkDay) error {
	whm.mu.Lock()
	defer whm.mu.Unlock()

	stmt := whm.prepared["upsertWorkDay"]
	dateKey := workDay.Date.Format("2006-01-02")
	
	_, err := stmt.Exec(
		workDay.ID,
		dateKey,
		workDay.StartTime,
		workDay.EndTime,
		int64(workDay.TotalTime.Seconds()),
		int64(workDay.BreakTime.Seconds()),
		workDay.SessionCount,
		workDay.BlockCount,
		workDay.IsComplete,
		workDay.GetEfficiencyRatio(),
		time.Now(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to save work day: %w", err)
	}

	whm.logger.Debug("Work day saved", "date", dateKey, "totalTime", workDay.TotalTime)
	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-010
 * CONTEXT:   GetWorkWeekData implements WorkHourDatabaseManager interface for weekly aggregation
 * REASON:    Business requirement for weekly work hour analysis and overtime calculation
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure week boundaries are correctly calculated, handle partial weeks properly
 * RISK:      Medium - Incorrect week calculations could affect overtime and scheduling analysis
 */
func (whm *WorkHourManager) GetWorkWeekData(weekStart time.Time) (*domain.WorkWeek, error) {
	// Normalize to Monday
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	
	weekKey := weekStart.Format("2006-01-02")
	
	// Check cache first
	whm.cacheMu.RLock()
	if cached, exists := whm.cache.workWeeks[weekKey]; exists {
		if lastUpdate, ok := whm.cache.lastUpdated["week_"+weekKey]; ok {
			if time.Since(lastUpdate) < whm.cache.ttl {
				whm.cacheMu.RUnlock()
				return cached, nil
			}
		}
	}
	whm.cacheMu.RUnlock()

	// Try to get from database
	stmt := whm.prepared["getWorkWeek"]
	var workWeek *domain.WorkWeek
	
	err := stmt.QueryRow(weekKey).Scan(
		&workWeek.ID, &workWeek.WeekStart, &workWeek.WeekEnd,
		&workWeek.TotalTime, &workWeek.OvertimeHours, &workWeek.AverageDay,
		&workWeek.StandardHours, &workWeek.IsComplete, new(interface{}),
	)
	
	if err == nil {
		whm.updateWorkWeekCache(weekKey, workWeek)
		return workWeek, nil
	}
	
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query work week: %w", err)
	}

	// Calculate from work days
	workWeek, err = whm.calculateWorkWeekFromDays(weekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate work week: %w", err)
	}

	// Save calculated work week
	if err := whm.saveWorkWeek(workWeek); err != nil {
		whm.logger.Warn("Failed to save calculated work week", "weekStart", weekKey, "error", err)
	}

	whm.updateWorkWeekCache(weekKey, workWeek)
	return workWeek, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-011
 * CONTEXT:   calculateWorkWeekFromDays aggregates work days into weekly metrics
 * REASON:    Weekly reports require aggregation of daily work data with overtime calculations
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate week boundary calculations, ensure overtime logic matches business rules
 * RISK:      High - Overtime calculations must be accurate for compliance and billing
 */
func (whm *WorkHourManager) calculateWorkWeekFromDays(weekStart time.Time) (*domain.WorkWeek, error) {
	workWeek := domain.NewWorkWeek(weekStart, 40*time.Hour)
	
	// Get work days for the week
	stmt := whm.prepared["getWorkDayRange"]
	weekEnd := weekStart.AddDate(0, 0, 6)
	
	rows, err := stmt.Query(
		weekStart.Format("2006-01-02"),
		weekEnd.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query work days for week: %w", err)
	}
	defer rows.Close()

	var totalTime time.Duration
	workDayCount := 0

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
			return nil, fmt.Errorf("failed to scan work day: %w", err)
		}

		// Parse date
		date, err := time.Parse("2006-01-02", dateKey)
		if err != nil {
			continue
		}

		// Create work day object
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

		workWeek.WorkDays = append(workWeek.WorkDays, *workDay)
		totalTime += workDay.TotalTime
		workDayCount++
	}

	workWeek.TotalTime = totalTime
	if workDayCount > 0 {
		workWeek.AverageDay = totalTime / time.Duration(workDayCount)
	}
	
	// Calculate overtime
	workWeek.CalculateOvertime()
	
	// Mark as complete if week is past
	now := time.Now()
	workWeek.IsComplete = now.After(workWeek.WeekEnd)

	return workWeek, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-012
 * CONTEXT:   Cache management methods for work hour data optimization
 * REASON:    Work hour calculations can be expensive, caching improves response times
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement cache size limits and TTL expiration to prevent memory issues
 * RISK:      Low - Cache management is not critical path, failures only affect performance
 */
func (whm *WorkHourManager) updateWorkDayCache(dateKey string, workDay *domain.WorkDay) {
	whm.cacheMu.Lock()
	defer whm.cacheMu.Unlock()
	
	// Implement LRU eviction if cache is full
	if len(whm.cache.workDays) >= whm.cache.maxSize {
		whm.evictOldestCacheEntries()
	}
	
	whm.cache.workDays[dateKey] = workDay
	whm.cache.lastUpdated[dateKey] = time.Now()
}

func (whm *WorkHourManager) updateWorkWeekCache(weekKey string, workWeek *domain.WorkWeek) {
	whm.cacheMu.Lock()
	defer whm.cacheMu.Unlock()
	
	cacheKey := "week_" + weekKey
	whm.cache.workWeeks[weekKey] = workWeek
	whm.cache.lastUpdated[cacheKey] = time.Now()
}

func (whm *WorkHourManager) evictOldestCacheEntries() {
	// Simple LRU implementation - remove 10% of entries
	toRemove := whm.cache.maxSize / 10
	if toRemove == 0 {
		toRemove = 1
	}
	
	// Find oldest entries
	type cacheEntry struct {
		key  string
		time time.Time
	}
	
	var entries []cacheEntry
	for key, updateTime := range whm.cache.lastUpdated {
		entries = append(entries, cacheEntry{key, updateTime})
	}
	
	// Sort by time and remove oldest
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].time.After(entries[j].time) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	
	for i := 0; i < toRemove && i < len(entries); i++ {
		key := entries[i].key
		delete(whm.cache.lastUpdated, key)
		
		if strings.HasPrefix(key, "week_") {
			delete(whm.cache.workWeeks, strings.TrimPrefix(key, "week_"))
		} else {
			delete(whm.cache.workDays, key)
		}
	}
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-013
 * CONTEXT:   saveWorkWeek persists calculated work week data
 * REASON:    Need to store weekly aggregations for efficient reporting
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle concurrent access properly, validate week data consistency
 * RISK:      Medium - Concurrent updates could cause data inconsistency
 */
func (whm *WorkHourManager) saveWorkWeek(workWeek *domain.WorkWeek) error {
	whm.mu.Lock()
	defer whm.mu.Unlock()

	stmt := whm.prepared["upsertWorkWeek"]
	
	_, err := stmt.Exec(
		workWeek.ID,
		workWeek.WeekStart.Format("2006-01-02"),
		workWeek.WeekEnd.Format("2006-01-02"),
		int64(workWeek.TotalTime.Seconds()),
		int64(workWeek.OvertimeHours.Seconds()),
		int64(workWeek.AverageDay.Seconds()),
		int64(workWeek.StandardHours.Seconds()),
		workWeek.IsComplete,
		len(workWeek.WorkDays),
		time.Now(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to save work week: %w", err)
	}

	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-014
 * CONTEXT:   Close extends base Close method with work hour specific cleanup
 * REASON:    Need to properly cleanup work hour resources and cache on shutdown
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure all prepared statements are closed properly
 * RISK:      Low - Resource cleanup failure doesn't affect data integrity
 */
func (whm *WorkHourManager) Close() error {
	// Close work hour specific prepared statements
	for name, stmt := range whm.prepared {
		if err := stmt.Close(); err != nil {
			whm.logger.Warn("Failed to close work hour statement", "name", name, "error", err)
		}
	}

	// Clear cache
	whm.cacheMu.Lock()
	whm.cache.workDays = make(map[string]*domain.WorkDay)
	whm.cache.workWeeks = make(map[string]*domain.WorkWeek)
	whm.cache.patterns = make(map[string]*domain.WorkPattern)
	whm.cache.timesheets = make(map[string]*domain.Timesheet)
	whm.cache.lastUpdated = make(map[string]time.Time)
	whm.cacheMu.Unlock()

	// Call base Close method
	return whm.KuzuManager.Close()
}

// Ensure WorkHourManager implements WorkHourDatabaseManager interface
var _ arch.WorkHourDatabaseManager = (*WorkHourManager)(nil)