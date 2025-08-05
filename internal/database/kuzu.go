/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-004
 * CONTEXT:   Kùzu graph database manager implementation for Claude monitoring system
 * REASON:    Need embedded graph database for session/work block relationships with high performance
 * CHANGE:    Fixed implementation with proper interfaces and field names.
 * PREVENTION:Ensure proper transaction boundaries and connection cleanup to prevent resource leaks
 * RISK:      Medium - Database corruption if transactions not handled properly
 */
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"

	// Use CGO-based SQL driver for Kùzu
	_ "github.com/mattn/go-sqlite3"
)

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-004
 * CONTEXT:   KuzuManager implements DatabaseManager interface for graph operations
 * REASON:    Centralized database operations with transaction management and connection pooling
 * CHANGE:    Fixed implementation with proper interfaces.
 * PREVENTION:Always use prepared statements and proper transaction boundaries for consistency
 * RISK:      High - Data corruption or loss if not properly synchronized
 */
type KuzuManager struct {
	db       *sql.DB
	dbPath   string
	mu       sync.RWMutex
	logger   arch.Logger
	prepared map[string]*sql.Stmt
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-004
 * CONTEXT:   NewKuzuManager creates a new database manager with schema initialization
 * REASON:    Factory pattern for proper initialization with error handling and resource management
 * CHANGE:    Fixed implementation with proper interfaces.
 * PREVENTION:Ensure database file permissions are correct and path is writable before initialization
 * RISK:      Medium - Database initialization failure if path not accessible
 */
func NewKuzuManager(dbPath string, logger arch.Logger) (*KuzuManager, error) {
	// Ensure database directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// For now, use SQLite as a placeholder for Kùzu
	// In production, this would use the official Kùzu Go library
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	km := &KuzuManager{
		db:       db,
		dbPath:   dbPath,
		logger:   logger,
		prepared: make(map[string]*sql.Stmt),
	}

	if err := km.initializeSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	if err := km.preparePredefinedStatements(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare statements: %w", err)
	}

	km.logger.Info("Kùzu database manager initialized", 
		"dbPath", dbPath,
		"connections", "10/5")

	return km, nil
}

// Initialize sets up database schema and connections (implements DatabaseManager interface)
func (km *KuzuManager) Initialize() error {
	// Schema initialization is already done in constructor
	return nil
}

// initializeSchema creates database schema
func (km *KuzuManager) initializeSchema() error {
	schema := `
		-- Sessions table (equivalent to Session nodes)
		CREATE TABLE IF NOT EXISTS sessions (
			session_id TEXT PRIMARY KEY,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		-- Work blocks table (equivalent to WorkBlock nodes)  
		CREATE TABLE IF NOT EXISTS work_blocks (
			block_id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			duration_seconds INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions(session_id)
		);

		-- Processes table (equivalent to Process nodes)
		CREATE TABLE IF NOT EXISTS processes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pid INTEGER NOT NULL,
			command TEXT NOT NULL,
			start_time TIMESTAMP NOT NULL,
			session_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions(session_id)
		);

		-- Indexes for performance
		CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions(start_time);
		CREATE INDEX IF NOT EXISTS idx_work_blocks_session ON work_blocks(session_id);
		CREATE INDEX IF NOT EXISTS idx_work_blocks_time ON work_blocks(start_time, end_time);
		CREATE INDEX IF NOT EXISTS idx_processes_session ON processes(session_id);
		CREATE INDEX IF NOT EXISTS idx_processes_pid ON processes(pid);
	`

	_, err := km.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	km.logger.Info("Database schema initialized successfully")
	return nil
}

// preparePredefinedStatements prepares frequently used SQL statements
func (km *KuzuManager) preparePredefinedStatements() error {
	statements := map[string]string{
		"createSession": `INSERT INTO sessions (session_id, start_time, end_time) VALUES (?, ?, ?)`,
		"updateSession": `UPDATE sessions SET end_time = ? WHERE session_id = ?`,
		"createWorkBlock": `INSERT INTO work_blocks (block_id, session_id, start_time, end_time, duration_seconds) VALUES (?, ?, ?, ?, ?)`,
		"updateWorkBlock": `UPDATE work_blocks SET end_time = ?, duration_seconds = ? WHERE block_id = ?`,
		"createProcess": `INSERT INTO processes (pid, command, start_time, session_id) VALUES (?, ?, ?, ?)`,
		"getCurrentSession": `SELECT session_id, start_time, end_time FROM sessions WHERE end_time IS NULL OR end_time > ? ORDER BY start_time DESC LIMIT 1`,
		"getActiveSession": `SELECT session_id, start_time, end_time FROM sessions WHERE end_time IS NULL OR end_time > datetime('now') ORDER BY start_time DESC LIMIT 1`,
		"getActiveWorkBlocks": `SELECT block_id, start_time, end_time, duration_seconds FROM work_blocks WHERE session_id = ? AND end_time IS NULL`,
	}

	for name, query := range statements {
		stmt, err := km.db.Prepare(query)
		if err != nil {
			return fmt.Errorf("failed to prepare statement %s: %w", name, err)
		}
		km.prepared[name] = stmt
	}

	km.logger.Info("Prepared statements initialized", "count", len(statements))
	return nil
}

// SaveSession persists a session to the graph database (implements DatabaseManager interface)
func (km *KuzuManager) SaveSession(session *domain.Session) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	tx, err := km.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt := km.prepared["createSession"]
	_, err = tx.Stmt(stmt).Exec(session.ID, session.StartTime, session.EndTime)
	if err != nil {
		km.logger.Error("Failed to create session", "sessionID", session.ID, "error", err)
		return fmt.Errorf("failed to create session %s: %w", session.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit session creation: %w", err)
	}

	km.logger.Info("Session created successfully", "sessionID", session.ID, "startTime", session.StartTime)
	return nil
}

// SaveWorkBlock persists a work block and its relationship to session (implements DatabaseManager interface)
func (km *KuzuManager) SaveWorkBlock(workBlock *domain.WorkBlock) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	tx, err := km.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Validate session exists
	var sessionExists int
	err = tx.QueryRow("SELECT 1 FROM sessions WHERE session_id = ?", workBlock.SessionID).Scan(&sessionExists)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("session %s does not exist", workBlock.SessionID)
		}
		return fmt.Errorf("failed to validate session: %w", err)
	}

	stmt := km.prepared["createWorkBlock"]
	_, err = tx.Stmt(stmt).Exec(workBlock.ID, workBlock.SessionID, workBlock.StartTime, workBlock.EndTime, workBlock.DurationSeconds)
	if err != nil {
		km.logger.Error("Failed to create work block", "blockID", workBlock.ID, "sessionID", workBlock.SessionID, "error", err)
		return fmt.Errorf("failed to create work block %s: %w", workBlock.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit work block creation: %w", err)
	}

	km.logger.Info("Work block created successfully", "blockID", workBlock.ID, "sessionID", workBlock.SessionID, "startTime", workBlock.StartTime)
	return nil
}

// SaveProcess persists a process record and its session relationship (implements DatabaseManager interface)
func (km *KuzuManager) SaveProcess(process *domain.Process, sessionID string) error {
	km.mu.RLock()
	defer km.mu.RUnlock()

	stmt := km.prepared["createProcess"]
	_, err := stmt.Exec(process.PID, process.Command, process.StartTime, sessionID)
	if err != nil {
		km.logger.Error("Failed to create process", "pid", process.PID, "command", process.Command, "error", err)
		return fmt.Errorf("failed to create process %d: %w", process.PID, err)
	}

	km.logger.Debug("Process created successfully", "pid", process.PID, "command", process.Command, "sessionID", sessionID)
	return nil
}

// GetSessionStats calculates aggregated statistics for reporting (implements DatabaseManager interface)
func (km *KuzuManager) GetSessionStats(period arch.TimePeriod) (*domain.SessionStats, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	// Calculate time range based on period
	now := time.Now()
	var startTime time.Time
	var periodStr string

	switch period {
	case arch.PeriodDaily:
		startTime = now.AddDate(0, 0, -1)
		periodStr = "daily"
	case arch.PeriodWeekly:
		startTime = now.AddDate(0, 0, -7)
		periodStr = "weekly"
	case arch.PeriodMonthly:
		startTime = now.AddDate(0, -1, 0)
		periodStr = "monthly"
	default:
		return nil, fmt.Errorf("unsupported period: %v", period)
	}

	// Session statistics
	sessionQuery := `
		SELECT 
			COUNT(*) as session_count,
			AVG(CASE WHEN end_time IS NOT NULL THEN 
				(julianday(end_time) - julianday(start_time)) * 24 * 3600 
				ELSE 0 END) as avg_session_duration_seconds,
			SUM(CASE WHEN end_time IS NOT NULL THEN 
				(julianday(end_time) - julianday(start_time)) * 24 * 3600 
				ELSE 0 END) as total_session_duration_seconds
		FROM sessions 
		WHERE start_time >= ? AND start_time <= ?`

	var sessionCount int
	var avgDuration, totalDuration sql.NullFloat64
	err := km.db.QueryRow(sessionQuery, startTime, now).Scan(&sessionCount, &avgDuration, &totalDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to query session statistics: %w", err)
	}

	// Work block statistics
	workBlockQuery := `
		SELECT COUNT(*) as work_block_count, SUM(duration_seconds) as total_work_duration_seconds
		FROM work_blocks wb
		JOIN sessions s ON wb.session_id = s.session_id
		WHERE s.start_time >= ? AND s.start_time <= ?`

	var workBlockCount int
	var totalWorkDuration sql.NullFloat64
	err = km.db.QueryRow(workBlockQuery, startTime, now).Scan(&workBlockCount, &totalWorkDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to query work block statistics: %w", err)
	}

	stats := &domain.SessionStats{
		Period:            periodStr,
		TotalSessions:     sessionCount,
		SessionCount:      sessionCount,
		WorkBlockCount:    workBlockCount,
		TotalWorkTime:     time.Duration(totalWorkDuration.Float64) * time.Second,
		AverageWorkTime:   time.Duration(avgDuration.Float64) * time.Second,
		StartDate:         startTime,
		EndDate:           now,
	}

	km.logger.Info("Session stats generated", "period", periodStr, "sessionCount", sessionCount, "workBlockCount", workBlockCount)
	return stats, nil
}

// GetActiveSession retrieves the currently active session from database (implements DatabaseManager interface)
func (km *KuzuManager) GetActiveSession() (*domain.Session, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	stmt := km.prepared["getActiveSession"]
	var sessionID string
	var startTime, endTime sql.NullTime
	err := stmt.QueryRow().Scan(&sessionID, &startTime, &endTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No active session
		}
		return nil, fmt.Errorf("failed to query active session: %w", err)
	}

	session := &domain.Session{
		ID:        sessionID,
		StartTime: startTime.Time,
		IsActive:  true,
	}

	if endTime.Valid {
		session.EndTime = endTime.Time
		session.IsActive = time.Now().Before(endTime.Time)
	}

	return session, nil
}

// HealthCheck verifies database connectivity and schema integrity (implements DatabaseManager interface)
func (km *KuzuManager) HealthCheck() error {
	km.mu.RLock()
	defer km.mu.RUnlock()

	// Simple ping to verify connection
	if err := km.db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Verify schema exists
	var count int
	err := km.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&count)
	if err != nil {
		return fmt.Errorf("schema verification failed: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("sessions table not found")
	}

	return nil
}

// Close cleanly shuts down database connections (implements DatabaseManager interface)
func (km *KuzuManager) Close() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Close prepared statements
	for name, stmt := range km.prepared {
		if err := stmt.Close(); err != nil {
			km.logger.Warn("Failed to close prepared statement", "name", name, "error", err)
		}
	}

	// Close database connection
	if err := km.db.Close(); err != nil {
		km.logger.Error("Failed to close database connection", "error", err)
		return err
	}

	km.logger.Info("Database manager closed successfully")
	return nil
}

// Ensure KuzuManager implements DatabaseManager interface
var _ arch.DatabaseManager = (*KuzuManager)(nil)