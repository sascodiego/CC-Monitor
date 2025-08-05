/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-026
 * CONTEXT:   Database migration utility for work hour schema extensions
 * REASON:    Need safe migration path from basic schema to work hour analytics without data loss
 * CHANGE:    Initial implementation.
 * PREVENTION:Always backup data before migration, implement rollback procedures, validate migration steps
 * RISK:      High - Migration failures could cause data loss or system instability
 */
package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
)

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-027
 * CONTEXT:   DatabaseMigrator handles safe schema upgrades and data migrations
 * REASON:    Need controlled migration process to upgrade existing installations to work hour functionality
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement transaction-based migrations with rollback capability, validate each step
 * RISK:      High - Improper migration could corrupt existing data or prevent system startup
 */
type DatabaseMigrator struct {
	db     *sql.DB
	logger arch.Logger
}

type MigrationStep struct {
	Version     int
	Name        string
	Description string
	UpQuery     string
	DownQuery   string
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-028
 * CONTEXT:   NewDatabaseMigrator creates migrator with database connection
 * REASON:    Factory pattern for migration management with proper logging and error handling
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate database connection before creating migrator, ensure proper permissions
 * RISK:      Medium - Migration without proper permissions could fail partially
 */
func NewDatabaseMigrator(db *sql.DB, logger arch.Logger) *DatabaseMigrator {
	return &DatabaseMigrator{
		db:     db,
		logger: logger,
	}
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-029
 * CONTEXT:   Migration steps for work hour schema extensions
 * REASON:    Define controlled migration path from basic monitoring to comprehensive work hour analytics
 * CHANGE:    Initial implementation.
 * PREVENTION:Test each migration step thoroughly, ensure backward compatibility where possible
 * RISK:      High - Incorrect migration steps could break existing functionality
 */
func (dm *DatabaseMigrator) GetWorkHourMigrations() []MigrationStep {
	return []MigrationStep{
		{
			Version:     1,
			Name:        "create_schema_version_table",
			Description: "Create schema version tracking table",
			UpQuery: `
				CREATE TABLE IF NOT EXISTS schema_versions (
					version INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					description TEXT,
					applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
				INSERT OR IGNORE INTO schema_versions (version, name, description) 
				VALUES (0, 'baseline', 'Initial schema with sessions, work_blocks, and processes');
			`,
			DownQuery: `DROP TABLE IF EXISTS schema_versions;`,
		},
		{
			Version:     2,
			Name:        "add_work_days_table",
			Description: "Add work days aggregation table",
			UpQuery: `
				CREATE TABLE IF NOT EXISTS work_days (
					id TEXT PRIMARY KEY,
					date_key TEXT NOT NULL UNIQUE,
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
				CREATE INDEX IF NOT EXISTS idx_work_days_date ON work_days(date_key);
				CREATE INDEX IF NOT EXISTS idx_work_days_complete ON work_days(is_complete);
			`,
			DownQuery: `
				DROP INDEX IF EXISTS idx_work_days_complete;
				DROP INDEX IF EXISTS idx_work_days_date;
				DROP TABLE IF EXISTS work_days;
			`,
		},
		{
			Version:     3,
			Name:        "add_work_weeks_table",
			Description: "Add work weeks aggregation table",
			UpQuery: `
				CREATE TABLE IF NOT EXISTS work_weeks (
					id TEXT PRIMARY KEY,
					week_start TEXT NOT NULL,
					week_end TEXT NOT NULL,
					total_time_seconds INTEGER DEFAULT 0,
					overtime_seconds INTEGER DEFAULT 0,
					average_day_seconds INTEGER DEFAULT 0,
					standard_hours_seconds INTEGER DEFAULT 144000,
					is_complete BOOLEAN DEFAULT FALSE,
					work_days_count INTEGER DEFAULT 0,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);
				CREATE INDEX IF NOT EXISTS idx_work_weeks_start ON work_weeks(week_start);
			`,
			DownQuery: `
				DROP INDEX IF EXISTS idx_work_weeks_start;
				DROP TABLE IF EXISTS work_weeks;
			`,
		},
		{
			Version:     4,
			Name:        "add_timesheets_tables",
			Description: "Add timesheet management tables",
			UpQuery: `
				CREATE TABLE IF NOT EXISTS timesheets (
					id TEXT PRIMARY KEY,
					employee_id TEXT NOT NULL DEFAULT 'default',
					period TEXT NOT NULL,
					start_date TEXT NOT NULL,
					end_date TEXT NOT NULL,
					total_hours_seconds INTEGER DEFAULT 0,
					regular_hours_seconds INTEGER DEFAULT 0,
					overtime_hours_seconds INTEGER DEFAULT 0,
					status TEXT DEFAULT 'draft',
					policy_data TEXT,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					submitted_at TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS timesheet_entries (
					id TEXT PRIMARY KEY,
					timesheet_id TEXT NOT NULL,
					date_key TEXT NOT NULL,
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

				CREATE INDEX IF NOT EXISTS idx_timesheets_employee_period ON timesheets(employee_id, period, start_date);
				CREATE INDEX IF NOT EXISTS idx_timesheet_entries_timesheet ON timesheet_entries(timesheet_id);
				CREATE INDEX IF NOT EXISTS idx_timesheet_entries_date ON timesheet_entries(date_key);
			`,
			DownQuery: `
				DROP INDEX IF EXISTS idx_timesheet_entries_date;
				DROP INDEX IF EXISTS idx_timesheet_entries_timesheet;
				DROP INDEX IF EXISTS idx_timesheets_employee_period;
				DROP TABLE IF EXISTS timesheet_entries;
				DROP TABLE IF EXISTS timesheets;
			`,
		},
		{
			Version:     5,
			Name:        "add_analytics_cache_tables",
			Description: "Add analytics and pattern caching tables",
			UpQuery: `
				CREATE TABLE IF NOT EXISTS work_patterns (
					id TEXT PRIMARY KEY,
					start_date TEXT NOT NULL,
					end_date TEXT NOT NULL,
					peak_hours TEXT,
					productivity_curve TEXT,
					work_day_type TEXT DEFAULT 'standard',
					consistency_score REAL DEFAULT 0.0,
					break_patterns TEXT,
					weekly_pattern TEXT,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					expires_at TIMESTAMP DEFAULT (datetime('now', '+24 hours'))
				);

				CREATE TABLE IF NOT EXISTS activity_summaries (
					id TEXT PRIMARY KEY,
					period TEXT NOT NULL,
					start_date TEXT NOT NULL,
					end_date TEXT NOT NULL,
					total_work_time_seconds INTEGER DEFAULT 0,
					total_sessions INTEGER DEFAULT 0,
					total_work_blocks INTEGER DEFAULT 0,
					average_session_seconds INTEGER DEFAULT 0,
					average_work_block_seconds INTEGER DEFAULT 0,
					daily_average_seconds INTEGER DEFAULT 0,
					trends_data TEXT,
					goals_data TEXT,
					efficiency_data TEXT,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					expires_at TIMESTAMP DEFAULT (datetime('now', '+1 hour'))
				);

				CREATE INDEX IF NOT EXISTS idx_work_patterns_dates ON work_patterns(start_date, end_date);
				CREATE INDEX IF NOT EXISTS idx_activity_summaries_period ON activity_summaries(period, start_date, end_date);
				CREATE INDEX IF NOT EXISTS idx_work_patterns_expires ON work_patterns(expires_at);
				CREATE INDEX IF NOT EXISTS idx_activity_summaries_expires ON activity_summaries(expires_at);
			`,
			DownQuery: `
				DROP INDEX IF EXISTS idx_activity_summaries_expires;
				DROP INDEX IF EXISTS idx_work_patterns_expires;
				DROP INDEX IF EXISTS idx_activity_summaries_period;
				DROP INDEX IF EXISTS idx_work_patterns_dates;
				DROP TABLE IF EXISTS activity_summaries;
				DROP TABLE IF EXISTS work_patterns;
			`,
		},
		{
			Version:     6,
			Name:        "add_cache_cleanup_triggers",
			Description: "Add automatic cache cleanup triggers",
			UpQuery: `
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
			`,
			DownQuery: `
				DROP TRIGGER IF EXISTS cleanup_expired_summaries;
				DROP TRIGGER IF EXISTS cleanup_expired_patterns;
			`,
		},
	}
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-030
 * CONTEXT:   MigrateToWorkHour performs complete migration to work hour analytics
 * REASON:    Need safe, transactional migration process that preserves existing data
 * CHANGE:    Initial implementation.
 * PREVENTION:Use transactions for each step, validate data integrity after each migration
 * RISK:      High - Migration failure could corrupt database or cause data loss
 */
func (dm *DatabaseMigrator) MigrateToWorkHour() error {
	dm.logger.Info("Starting work hour migration")

	// Get current schema version
	currentVersion, err := dm.getCurrentSchemaVersion()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	dm.logger.Info("Current schema version", "version", currentVersion)

	migrations := dm.GetWorkHourMigrations()
	targetVersion := len(migrations)

	if currentVersion >= targetVersion {
		dm.logger.Info("Schema is already up to date", "currentVersion", currentVersion, "targetVersion", targetVersion)
		return nil
	}

	// Create backup before migration
	if err := dm.createBackup(); err != nil {
		dm.logger.Warn("Failed to create backup", "error", err)
		// Continue with migration but log the warning
	}

	// Apply migrations sequentially
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue // Skip already applied migrations
		}

		dm.logger.Info("Applying migration", 
			"version", migration.Version,
			"name", migration.Name,
			"description", migration.Description)

		if err := dm.applyMigration(migration); err != nil {
			dm.logger.Error("Migration failed", 
				"version", migration.Version,
				"name", migration.Name,
				"error", err)
			return fmt.Errorf("migration %d failed: %w", migration.Version, err)
		}

		dm.logger.Info("Migration completed successfully", "version", migration.Version)
	}

	// Validate migration success
	if err := dm.validateMigration(); err != nil {
		return fmt.Errorf("migration validation failed: %w", err)
	}

	dm.logger.Info("Work hour migration completed successfully", 
		"fromVersion", currentVersion,
		"toVersion", targetVersion)

	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-031
 * CONTEXT:   getCurrentSchemaVersion determines current database schema version
 * REASON:    Need to know current schema state to determine which migrations to apply
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle case where schema_versions table doesn't exist (new installation)
 * RISK:      Low - Version detection failure only affects migration planning
 */
func (dm *DatabaseMigrator) getCurrentSchemaVersion() (int, error) {
	var version int
	err := dm.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_versions").Scan(&version)
	if err != nil {
		// If table doesn't exist, assume version 0 (baseline)
		if err == sql.ErrNoRows || err.Error() == "no such table: schema_versions" {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to query schema version: %w", err)
	}
	return version, nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-032
 * CONTEXT:   applyMigration executes single migration step within transaction
 * REASON:    Each migration step must be atomic to ensure database consistency
 * CHANGE:    Initial implementation.
 * PREVENTION:Use transactions for rollback capability, validate queries before execution
 * RISK:      High - Failed migration step without proper rollback could corrupt database
 */
func (dm *DatabaseMigrator) applyMigration(migration MigrationStep) error {
	tx, err := dm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration query
	_, err = tx.Exec(migration.UpQuery)
	if err != nil {
		return fmt.Errorf("failed to execute migration query: %w", err)
	}

	// Update schema version
	_, err = tx.Exec(`
		INSERT INTO schema_versions (version, name, description, applied_at) 
		VALUES (?, ?, ?, ?)`,
		migration.Version, migration.Name, migration.Description, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-033
 * CONTEXT:   createBackup creates database backup before migration
 * REASON:    Safety measure to allow recovery if migration fails
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure backup location has sufficient space, validate backup integrity
 * RISK:      Medium - Backup failure doesn't prevent migration but reduces recovery options
 */
func (dm *DatabaseMigrator) createBackup() error {
	// Simple backup by executing VACUUM INTO
	backupPath := fmt.Sprintf("/tmp/claude-monitor-backup-%d.db", time.Now().Unix())
	
	_, err := dm.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	dm.logger.Info("Database backup created", "backupPath", backupPath)
	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-034
 * CONTEXT:   validateMigration verifies migration completed successfully
 * REASON:    Post-migration validation ensures schema integrity and functionality
 * CHANGE:    Initial implementation.
 * PREVENTION:Check all required tables exist and have expected structure
 * RISK:      Low - Validation failure indicates migration problems but doesn't cause corruption
 */
func (dm *DatabaseMigrator) validateMigration() error {
	// Check that all expected tables exist
	expectedTables := []string{
		"sessions", "work_blocks", "processes",
		"work_days", "work_weeks", "timesheets", "timesheet_entries",
		"work_patterns", "activity_summaries", "schema_versions",
	}

	for _, table := range expectedTables {
		var count int
		err := dm.db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
			table).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		if count == 0 {
			return fmt.Errorf("table %s not found after migration", table)
		}
	}

	// Verify schema version
	currentVersion, err := dm.getCurrentSchemaVersion()
	if err != nil {
		return fmt.Errorf("failed to verify schema version: %w", err)
	}

	expectedVersion := len(dm.GetWorkHourMigrations())
	if currentVersion != expectedVersion {
		return fmt.Errorf("schema version mismatch: expected %d, got %d", expectedVersion, currentVersion)
	}

	dm.logger.Info("Migration validation successful", "schemaVersion", currentVersion)
	return nil
}

/**
 * AGENT:     database-manager
 * TRACE:     CLAUDE-WH-035
 * CONTEXT:   RollbackMigration rolls back migration to previous version
 * REASON:    Provide rollback capability if migration causes issues
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate rollback queries thoroughly, ensure data preservation
 * RISK:      High - Rollback operations could cause data loss if not properly implemented
 */
func (dm *DatabaseMigrator) RollbackMigration(targetVersion int) error {
	dm.logger.Info("Starting migration rollback", "targetVersion", targetVersion)

	currentVersion, err := dm.getCurrentSchemaVersion()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	if targetVersion >= currentVersion {
		return fmt.Errorf("target version %d is not less than current version %d", targetVersion, currentVersion)
	}

	migrations := dm.GetWorkHourMigrations()

	// Apply rollback migrations in reverse order
	for i := len(migrations) - 1; i >= 0; i-- {
		migration := migrations[i]
		if migration.Version <= targetVersion {
			break
		}

		if migration.Version > currentVersion {
			continue // Skip migrations that weren't applied
		}

		dm.logger.Info("Rolling back migration", 
			"version", migration.Version,
			"name", migration.Name)

		if err := dm.rollbackMigration(migration); err != nil {
			return fmt.Errorf("rollback of migration %d failed: %w", migration.Version, err)
		}
	}

	dm.logger.Info("Migration rollback completed", 
		"fromVersion", currentVersion,
		"toVersion", targetVersion)

	return nil
}

func (dm *DatabaseMigrator) rollbackMigration(migration MigrationStep) error {
	tx, err := dm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin rollback transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute rollback query
	_, err = tx.Exec(migration.DownQuery)
	if err != nil {
		return fmt.Errorf("failed to execute rollback query: %w", err)
	}

	// Remove from schema versions
	_, err = tx.Exec("DELETE FROM schema_versions WHERE version = ?", migration.Version)
	if err != nil {
		return fmt.Errorf("failed to remove schema version: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	return nil
}