/**
 * CONTEXT:   SQLite database connection and management for Claude Monitor
 * INPUT:     Database path, connection configuration, and transaction management
 * OUTPUT:    Production-ready SQLite database operations with connection pooling
 * BUSINESS:  Single-source SQLite persistence replacing gob-based system
 * CHANGE:    Complete rewrite from gob to SQLite with proper SQL operations
 * RISK:      Low - Standard database/sql package with SQLite, proper error handling
 */

package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

// SQLiteDB represents the SQLite database connection and operations
type SQLiteDB struct {
	db       *sql.DB
	dbPath   string
	mu       sync.RWMutex
	timezone *time.Location
}

// ConnectionConfig holds configuration for database connections
type ConnectionConfig struct {
	DBPath          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Timezone        string // Default: "America/Montevideo"
}

// DefaultConnectionConfig returns sensible defaults for SQLite connections
func DefaultConnectionConfig(dbPath string) *ConnectionConfig {
	return &ConnectionConfig{
		DBPath:          dbPath,
		MaxOpenConns:    25,  // SQLite handles concurrent reads well
		MaxIdleConns:    5,   // Keep some connections idle for quick access
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,
		Timezone:        "America/Montevideo",
	}
}

/**
 * CONTEXT:   Create new SQLite database connection with proper configuration
 * INPUT:     Connection configuration with paths and connection pooling settings
 * OUTPUT:    Configured SQLite database connection or error
 * BUSINESS:  Database serves as single source of truth for all work tracking data
 * CHANGE:    Initial SQLite connection implementation with production settings
 * RISK:      Medium - Database initialization critical for system operation
 */
func NewSQLiteDB(config *ConnectionConfig) (*SQLiteDB, error) {
	if config == nil {
		return nil, fmt.Errorf("connection config cannot be nil")
	}

	if config.DBPath == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}

	// Create directory if it doesn't exist
	dbDir := filepath.Dir(config.DBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Load timezone
	timezone, err := time.LoadLocation(config.Timezone)
	if err != nil {
		log.Printf("Warning: failed to load timezone %s, using UTC: %v", config.Timezone, err)
		timezone = time.UTC
	}

	// SQLite connection string with optimizations
	connectionString := config.DBPath + 
		"?_foreign_keys=on" +           // Enable foreign key constraints
		"&_journal_mode=WAL" +          // Write-Ahead Logging for better concurrent access
		"&_synchronous=NORMAL" +        // Balance between safety and performance
		"&_cache_size=10000" +          // 10MB cache
		"&_temp_store=memory" +         // Use memory for temp storage
		"&_timeout=5000"                // 5 second timeout

	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	sqliteDB := &SQLiteDB{
		db:       db,
		dbPath:   config.DBPath,
		timezone: timezone,
	}

	// Test connection and initialize schema
	if err := sqliteDB.Initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	log.Printf("🗄️  Initialized SQLite database at: %s (timezone: %s)", 
		config.DBPath, timezone.String())

	return sqliteDB, nil
}

/**
 * CONTEXT:   Initialize database schema and verify connection health
 * INPUT:     No parameters, uses embedded schema and connection settings
 * OUTPUT:    Error if initialization fails, nil on success
 * BUSINESS:  Ensure database is ready for work tracking operations
 * CHANGE:    Initial schema application with version tracking
 * RISK:      Medium - Schema changes must be backward compatible
 */
func (db *SQLiteDB) Initialize() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database connection test failed: %w", err)
	}

	// Apply schema
	schemaSQL, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute schema in a transaction for atomicity
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin schema transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit schema transaction: %w", err)
	}

	// Verify schema version
	var version int
	var description string
	query := "SELECT version, description FROM schema_version ORDER BY version DESC LIMIT 1"
	err = db.db.QueryRowContext(ctx, query).Scan(&version, &description)
	if err != nil {
		return fmt.Errorf("failed to verify schema version: %w", err)
	}

	log.Printf("🗄️  Database schema version %d: %s", version, description)

	return nil
}

/**
 * CONTEXT:   Execute database operations within a transaction for consistency
 * INPUT:     Transaction function that performs multiple database operations
 * OUTPUT:    Transaction result or rollback error
 * BUSINESS:  Ensure data consistency across related operations
 * CHANGE:    Initial transaction management implementation
 * RISK:      Low - Standard transaction pattern with automatic rollback
 */
func (db *SQLiteDB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safe to call even after commit

	if err := fn(tx); err != nil {
		return err // Transaction will be rolled back by defer
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Get database connection for direct SQL operations with context
 * INPUT:     Context for operation timeout and cancellation
 * OUTPUT:    Database connection ready for queries
 * BUSINESS:  Provide direct access for complex queries and operations
 * CHANGE:    Initial connection accessor for repository implementations
 * RISK:      Low - Read-only access to underlying connection
 */
func (db *SQLiteDB) DB() *sql.DB {
	return db.db
}

// SetDB sets the database connection (used primarily for testing)
func (db *SQLiteDB) SetDB(sqlDB *sql.DB) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.db = sqlDB
}

func (db *SQLiteDB) DBPath() string {
	return db.dbPath
}

func (db *SQLiteDB) Timezone() *time.Location {
	return db.timezone
}

/**
 * CONTEXT:   Simple database ping for basic connectivity check
 * INPUT:     Context for timeout control
 * OUTPUT:    Error if connection failed, nil if healthy
 * BUSINESS:  Quick connectivity check for health endpoints
 * CHANGE:    Simple ping method for health checks
 * RISK:      Low - Basic connectivity test
 */
func (db *SQLiteDB) Ping(ctx context.Context) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if err := db.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Health check for database connection and basic operations
 * INPUT:     Context for timeout control
 * OUTPUT:    Error if unhealthy, nil if healthy
 * BUSINESS:  Verify database is available for work tracking operations
 * CHANGE:    Initial health check implementation
 * RISK:      Low - Simple connectivity and table existence check
 */
func (db *SQLiteDB) HealthCheck(ctx context.Context) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Test connection
	if err := db.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test basic query
	var count int
	query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table'"
	if err := db.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	if count < 4 { // Should have at least users, projects, sessions, work_blocks tables
		return fmt.Errorf("database schema incomplete: only %d tables found", count)
	}

	return nil
}

/**
 * CONTEXT:   Get comprehensive database statistics for monitoring
 * INPUT:     Context for query timeout
 * OUTPUT:    Database statistics including table counts and sizes
 * BUSINESS:  Monitor database health and growth for system administration
 * CHANGE:    Initial statistics implementation for monitoring
 * RISK:      Low - Read-only statistics queries
 */
func (db *SQLiteDB) GetStats(ctx context.Context) (map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	stats := make(map[string]interface{})

	// Table row counts
	tables := []string{"users", "projects", "sessions", "work_blocks", "activity_events"}
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := db.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			log.Printf("Warning: failed to count %s table: %v", table, err)
			stats[table+"_count"] = 0
		} else {
			stats[table+"_count"] = count
		}
	}

	// Active sessions and work blocks
	var activeSessions, activeWorkBlocks int

	query := "SELECT COUNT(*) FROM sessions WHERE state = 'active' AND datetime('now') <= end_time"
	if err := db.db.QueryRowContext(ctx, query).Scan(&activeSessions); err != nil {
		log.Printf("Warning: failed to count active sessions: %v", err)
	}
	stats["active_sessions"] = activeSessions

	query = "SELECT COUNT(*) FROM work_blocks WHERE state IN ('active', 'processing')"
	if err := db.db.QueryRowContext(ctx, query).Scan(&activeWorkBlocks); err != nil {
		log.Printf("Warning: failed to count active work blocks: %v", err)
	}
	stats["active_work_blocks"] = activeWorkBlocks

	// Database file info
	if fileInfo, err := os.Stat(db.dbPath); err == nil {
		stats["database_size_bytes"] = fileInfo.Size()
		stats["database_modified"] = fileInfo.ModTime()
	}

	// Connection pool stats
	dbStats := db.db.Stats()
	stats["max_open_connections"] = dbStats.MaxOpenConnections
	stats["open_connections"] = dbStats.OpenConnections
	stats["in_use_connections"] = dbStats.InUse
	stats["idle_connections"] = dbStats.Idle

	stats["timezone"] = db.timezone.String()
	stats["last_updated"] = time.Now().In(db.timezone)

	return stats, nil
}

/**
 * CONTEXT:   Backup database to specified location for data safety
 * INPUT:     Backup file path for database copy
 * OUTPUT:    Error if backup fails, nil on success
 * BUSINESS:  Ensure work tracking data is protected against loss
 * CHANGE:    Initial backup implementation for data protection
 * RISK:      Low - File copy operation with proper error handling
 */
func (db *SQLiteDB) Backup(backupPath string) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Create backup directory if needed
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Use SQLite VACUUM INTO for consistent backup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := "VACUUM INTO ?"
	if _, err := db.db.ExecContext(ctx, query, backupPath); err != nil {
		return fmt.Errorf("failed to create database backup: %w", err)
	}

	log.Printf("🗄️  Database backed up to: %s", backupPath)
	return nil
}

/**
 * CONTEXT:   Close database connection and clean up resources
 * INPUT:     No parameters, closes all connections
 * OUTPUT:    Error if close fails, nil on success
 * BUSINESS:  Ensure proper cleanup when system shuts down
 * CHANGE:    Initial cleanup implementation
 * RISK:      Low - Standard database close with error handling
 */
func (db *SQLiteDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.db == nil {
		return nil
	}

	if err := db.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	log.Printf("🗄️  Closed SQLite database connection")
	db.db = nil
	return nil
}

/**
 * CONTEXT:   Convert time to database timezone for consistent storage
 * INPUT:     Time value in any timezone
 * OUTPUT:    Time converted to database timezone
 * BUSINESS:  Ensure all stored times use consistent timezone (America/Montevideo)
 * CHANGE:    Initial timezone conversion utility
 * RISK:      Low - Standard time conversion with timezone handling
 */
func (db *SQLiteDB) ToDBTime(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.In(db.timezone)
}

/**
 * CONTEXT:   Convert database time to local timezone for display
 * INPUT:     Time value from database in database timezone
 * OUTPUT:    Time converted to local timezone
 * BUSINESS:  Display times in user's local timezone for better UX
 * CHANGE:    Initial timezone conversion for user display
 * RISK:      Low - Standard time conversion for user interface
 */
func (db *SQLiteDB) FromDBTime(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.In(time.Local)
}

/**
 * CONTEXT:   Get current time in database timezone
 * INPUT:     No parameters, uses current system time
 * OUTPUT:    Current time in database timezone
 * BUSINESS:  Consistent timestamp generation for database operations
 * CHANGE:    Initial current time utility for database operations
 * RISK:      Low - Simple time generation with timezone conversion
 */
func (db *SQLiteDB) Now() time.Time {
	return time.Now().In(db.timezone)
}