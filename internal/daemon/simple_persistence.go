package daemon

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-PERSISTENCE-001
 * CONTEXT:   Simple persistence layer for enhanced daemon to save work hour data
 * REASON:    Need persistent storage without complex work hour system dependencies
 * CHANGE:    Simple database operations for session and work block persistence.
 * PREVENTION:Keep database operations simple and focused, handle connection errors gracefully
 * RISK:      Low - Simple SQLite operations with basic error handling
 */

type SimplePersistence struct {
	db *sql.DB
}

type SessionRecord struct {
	SessionID string    `json:"sessionId"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	IsActive  bool      `json:"isActive"`
}

type WorkBlockRecord struct {
	BlockID       string    `json:"blockId"`
	SessionID     string    `json:"sessionId"`
	StartTime     time.Time `json:"startTime"`
	EndTime       *time.Time `json:"endTime,omitempty"`
	DurationSecs  int       `json:"durationSeconds"`
	IsActive      bool      `json:"isActive"`
}

type WorkDayRecord struct {
	Date         string    `json:"date"`
	StartTime    time.Time `json:"startTime"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	TotalSeconds int       `json:"totalSeconds"`
	SessionCount int       `json:"sessionCount"`
	BlockCount   int       `json:"blockCount"`
	Efficiency   float64   `json:"efficiency"`
}

func NewSimplePersistence(dbPath string) (*SimplePersistence, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	sp := &SimplePersistence{db: db}
	if err := sp.createTables(); err != nil {
		return nil, err
	}

	return sp, nil
}

func (sp *SimplePersistence) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			session_id TEXT PRIMARY KEY,
			start_time DATETIME NOT NULL,
			end_time DATETIME NOT NULL,
			is_active BOOLEAN NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS work_blocks (
			block_id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			duration_seconds INTEGER DEFAULT 0,
			is_active BOOLEAN NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions (session_id)
		)`,
		`CREATE TABLE IF NOT EXISTS work_days (
			date TEXT PRIMARY KEY,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			total_seconds INTEGER DEFAULT 0,
			session_count INTEGER DEFAULT 0,
			block_count INTEGER DEFAULT 0,
			efficiency REAL DEFAULT 0.0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions (start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_work_blocks_session_id ON work_blocks (session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_work_days_date ON work_days (date)`,
	}

	for _, query := range queries {
		if _, err := sp.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (sp *SimplePersistence) SaveSession(session SessionRecord) error {
	query := `INSERT OR REPLACE INTO sessions (session_id, start_time, end_time, is_active) 
			  VALUES (?, ?, ?, ?)`
	_, err := sp.db.Exec(query, session.SessionID, session.StartTime, session.EndTime, session.IsActive)
	return err
}

func (sp *SimplePersistence) SaveWorkBlock(block WorkBlockRecord) error {
	query := `INSERT OR REPLACE INTO work_blocks 
			  (block_id, session_id, start_time, end_time, duration_seconds, is_active) 
			  VALUES (?, ?, ?, ?, ?, ?)`
	_, err := sp.db.Exec(query, block.BlockID, block.SessionID, block.StartTime, 
		block.EndTime, block.DurationSecs, block.IsActive)
	return err
}

func (sp *SimplePersistence) SaveWorkDay(day WorkDayRecord) error {
	query := `INSERT OR REPLACE INTO work_days 
			  (date, start_time, end_time, total_seconds, session_count, block_count, efficiency) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := sp.db.Exec(query, day.Date, day.StartTime, day.EndTime, 
		day.TotalSeconds, day.SessionCount, day.BlockCount, day.Efficiency)
	return err
}

func (sp *SimplePersistence) GetWorkDayHistory(days int) ([]WorkDayRecord, error) {
	query := `SELECT date, start_time, end_time, total_seconds, session_count, block_count, efficiency
			  FROM work_days 
			  ORDER BY date DESC 
			  LIMIT ?`
	
	rows, err := sp.db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workDays []WorkDayRecord
	for rows.Next() {
		var day WorkDayRecord
		err := rows.Scan(&day.Date, &day.StartTime, &day.EndTime, 
			&day.TotalSeconds, &day.SessionCount, &day.BlockCount, &day.Efficiency)
		if err != nil {
			continue
		}
		workDays = append(workDays, day)
	}

	return workDays, nil
}

func (sp *SimplePersistence) GetSessionHistory(days int) ([]SessionRecord, error) {
	query := `SELECT session_id, start_time, end_time, is_active
			  FROM sessions 
			  WHERE start_time >= datetime('now', '-' || ? || ' days')
			  ORDER BY start_time DESC`
	
	rows, err := sp.db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionRecord
	for rows.Next() {
		var session SessionRecord
		err := rows.Scan(&session.SessionID, &session.StartTime, &session.EndTime, &session.IsActive)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (sp *SimplePersistence) GetTotalWorkTime(days int) (time.Duration, error) {
	query := `SELECT COALESCE(SUM(total_seconds), 0) 
			  FROM work_days 
			  WHERE date >= date('now', '-' || ? || ' days')`
	
	var totalSeconds int
	err := sp.db.QueryRow(query, days).Scan(&totalSeconds)
	if err != nil {
		return 0, err
	}

	return time.Duration(totalSeconds) * time.Second, nil
}

func (sp *SimplePersistence) ExportData(format string) (string, error) {
	workDays, err := sp.GetWorkDayHistory(30)
	if err != nil {
		return "", err
	}

	sessions, err := sp.GetSessionHistory(30)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"workDays": workDays,
		"sessions": sessions,
		"exportTime": time.Now(),
		"format": format,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func (sp *SimplePersistence) Close() error {
	if sp.db != nil {
		return sp.db.Close()
	}
	return nil
}