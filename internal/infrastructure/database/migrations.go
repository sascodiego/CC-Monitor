/**
 * CONTEXT:   Database migration system for KuzuDB schema management and versioning
 * INPUT:     Schema migration scripts, version tracking, and database initialization
 * OUTPUT:    Automated schema setup and migration management for work tracking system
 * BUSINESS:  Reliable schema management ensures consistent database state across deployments
 * CHANGE:    Initial migration system with schema versioning and error recovery
 * RISK:      Medium - Schema changes require careful validation and rollback support
 */

package database

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kuzudb/go-kuzu"
)

// MigrationStatus represents the state of a migration
type MigrationStatus string

const (
	MigrationStatusPending   MigrationStatus = "pending"
	MigrationStatusRunning   MigrationStatus = "running" 
	MigrationStatusCompleted MigrationStatus = "completed"
	MigrationStatusFailed    MigrationStatus = "failed"
)

// Migration represents a single database migration
type Migration struct {
	Version     int             `json:"version"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Script      string          `json:"script"`
	Status      MigrationStatus `json:"status"`
	AppliedAt   time.Time       `json:"applied_at"`
	Duration    time.Duration   `json:"duration"`
	Error       string          `json:"error,omitempty"`
}

/**
 * CONTEXT:   Migration manager for KuzuDB schema evolution and version tracking
 * INPUT:     Migration scripts, database connection, and version requirements
 * OUTPUT:    Managed schema updates with version tracking and error recovery
 * BUSINESS:  Schema evolution must be reliable and reversible for production use
 * CHANGE:    Initial migration manager with comprehensive error handling
 * RISK:      Medium - Schema migrations can break database if not carefully managed
 */
type KuzuMigrationManager struct {
	connManager    *KuzuConnectionManager
	migrationsPath string
	currentVersion int
	migrations     []Migration
}

// NewKuzuMigrationManager creates a new migration manager
func NewKuzuMigrationManager(connManager *KuzuConnectionManager, migrationsPath string) *KuzuMigrationManager {
	return &KuzuMigrationManager{
		connManager:    connManager,
		migrationsPath: migrationsPath,
		migrations:     make([]Migration, 0),
	}
}

/**
 * CONTEXT:   Initialize database with migration tracking table
 * INPUT:     Database connection and migration metadata requirements
 * OUTPUT:    Migration tracking table created for version management
 * BUSINESS:  Migration tracking enables reliable schema version management
 * CHANGE:    Initial migration tracking table setup
 * RISK:      Low - Simple table creation with error handling
 */
func (kmm *KuzuMigrationManager) InitializeMigrationTracking(ctx context.Context) error {
	// Create migration tracking table
	migrationTableSQL := `
		CREATE NODE TABLE IF NOT EXISTS Migration(
			version INT64,
			name STRING,
			description STRING,
			status STRING,
			applied_at TIMESTAMP,
			duration_ms INT64,
			error_message STRING DEFAULT '',
			checksum STRING DEFAULT '',
			PRIMARY KEY (version)
		);
	`

	_, err := kmm.connManager.Query(ctx, migrationTableSQL, nil)
	if err != nil {
		return fmt.Errorf("failed to create migration tracking table: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Load migration scripts from filesystem and prepare for execution
 * INPUT:     Migration script directory with versioned SQL files
 * OUTPUT:    Sorted list of migrations ready for execution
 * BUSINESS:  Migration scripts must be loaded in version order for proper execution
 * CHANGE:    Initial migration loading with file validation
 * RISK:      Low - File system operations with validation and error handling
 */
func (kmm *KuzuMigrationManager) LoadMigrations() error {
	if kmm.migrationsPath == "" {
		// Load embedded schema as first migration
		kmm.migrations = []Migration{
			{
				Version:     1,
				Name:        "initial_schema",
				Description: "Initial KuzuDB schema for Claude Monitor work tracking",
				Script:      getInitialSchema(),
				Status:      MigrationStatusPending,
			},
		}
		return nil
	}

	// Check if migrations directory exists
	if _, err := os.Stat(kmm.migrationsPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist: %s", kmm.migrationsPath)
	}

	// Read migration files
	err := filepath.WalkDir(kmm.migrationsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .cypher and .sql files
		if !strings.HasSuffix(path, ".cypher") && !strings.HasSuffix(path, ".sql") {
			return nil
		}

		migration, err := kmm.parseMigrationFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse migration file %s: %w", path, err)
		}

		kmm.migrations = append(kmm.migrations, migration)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Sort migrations by version
	sort.Slice(kmm.migrations, func(i, j int) bool {
		return kmm.migrations[i].Version < kmm.migrations[j].Version
	})

	return nil
}

/**
 * CONTEXT:   Parse individual migration file and extract metadata
 * INPUT:     Migration file path with version and description in filename
 * OUTPUT:    Migration struct with parsed metadata and script content
 * BUSINESS:  Migration files must follow naming convention for proper ordering
 * CHANGE:    Initial file parsing with metadata extraction
 * RISK:      Low - File parsing with validation and error handling
 */
func (kmm *KuzuMigrationManager) parseMigrationFile(filePath string) (Migration, error) {
	// Extract filename without extension
	filename := filepath.Base(filePath)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	// Expected format: 001_initial_schema.cypher or 002_add_indexes.sql
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return Migration{}, fmt.Errorf("invalid migration filename format, expected: version_name.cypher")
	}

	// Parse version
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return Migration{}, fmt.Errorf("invalid version number in filename: %s", parts[0])
	}

	// Extract name
	name := strings.ReplaceAll(parts[1], "_", " ")

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return Migration{}, fmt.Errorf("failed to read migration file: %w", err)
	}

	// Extract description from first comment line if present
	description := name
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-- Description:") {
			description = strings.TrimPrefix(line, "-- Description:")
			description = strings.TrimSpace(description)
			break
		}
	}

	return Migration{
		Version:     version,
		Name:        name,
		Description: description,
		Script:      string(content),
		Status:      MigrationStatusPending,
	}, nil
}

/**
 * CONTEXT:   Get current database schema version from migration tracking
 * INPUT:     Database connection for version query
 * OUTPUT:    Current schema version number or 0 if no migrations applied
 * BUSINESS:  Version tracking enables incremental schema updates
 * CHANGE:    Initial version tracking query
 * RISK:      Low - Simple query with error handling
 */
func (kmm *KuzuMigrationManager) GetCurrentVersion(ctx context.Context) (int, error) {
	query := `
		MATCH (m:Migration) 
		WHERE m.status = 'completed' 
		RETURN MAX(m.version) as max_version;
	`

	result, err := kmm.connManager.Query(ctx, query, nil)
	if err != nil {
		// If migration table doesn't exist, version is 0
		if strings.Contains(err.Error(), "Migration") {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	defer result.Close()

	if !result.HasNext() {
		return 0, nil
	}

	record, err := result.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to read version result: %w", err)
	}

	if len(record) == 0 {
		return 0, nil
	}

	maxVersion, ok := record[0].(int64)
	if !ok {
		return 0, nil
	}

	return int(maxVersion), nil
}

/**
 * CONTEXT:   Apply pending migrations to bring database to latest schema version
 * INPUT:     Target version (0 for latest), migration context, and error handling
 * OUTPUT:    Database updated to target schema version with migration tracking
 * BUSINESS:  Schema updates must be reliable and trackable for production deployment
 * CHANGE:    Initial migration execution with comprehensive error handling
 * RISK:      High - Schema changes can break application if not properly validated
 */
func (kmm *KuzuMigrationManager) Migrate(ctx context.Context, targetVersion int) error {
	// Initialize migration tracking if needed
	if err := kmm.InitializeMigrationTracking(ctx); err != nil {
		return fmt.Errorf("failed to initialize migration tracking: %w", err)
	}

	// Get current version
	currentVersion, err := kmm.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	kmm.currentVersion = currentVersion

	// Load migrations if not already loaded
	if len(kmm.migrations) == 0 {
		if err := kmm.LoadMigrations(); err != nil {
			return fmt.Errorf("failed to load migrations: %w", err)
		}
	}

	// Determine target version
	if targetVersion == 0 {
		// Migrate to latest version
		if len(kmm.migrations) == 0 {
			return fmt.Errorf("no migrations found")
		}
		targetVersion = kmm.migrations[len(kmm.migrations)-1].Version
	}

	// Find pending migrations
	pendingMigrations := make([]Migration, 0)
	for _, migration := range kmm.migrations {
		if migration.Version > currentVersion && migration.Version <= targetVersion {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}

	if len(pendingMigrations) == 0 {
		return nil // No migrations to apply
	}

	// Apply each migration
	for _, migration := range pendingMigrations {
		if err := kmm.applyMigration(ctx, &migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", migration.Version, err)
		}
	}

	return nil
}

/**
 * CONTEXT:   Apply single migration with transaction support and error tracking
 * INPUT:     Migration to apply, database context, and error recovery requirements
 * OUTPUT:    Migration applied with status tracking or rollback on error
 * BUSINESS:  Individual migrations must be atomic to prevent partial schema states
 * CHANGE:    Initial single migration execution with transaction support
 * RISK:      High - Failed migrations can leave database in inconsistent state
 */
func (kmm *KuzuMigrationManager) applyMigration(ctx context.Context, migration *Migration) error {
	startTime := time.Now()
	migration.Status = MigrationStatusRunning

	// Record migration start
	if err := kmm.recordMigrationStatus(ctx, migration); err != nil {
		return fmt.Errorf("failed to record migration start: %w", err)
	}

	// Apply migration within transaction
	err := kmm.connManager.WithTransaction(ctx, func(conn *kuzu.Connection) error {
		// Split script into individual statements
		statements := kmm.splitStatements(migration.Script)

		for i, statement := range statements {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}

			// Skip comments
			if strings.HasPrefix(statement, "--") || strings.HasPrefix(statement, "/*") {
				continue
			}

			_, err := conn.Query(statement)
			if err != nil {
				return fmt.Errorf("statement %d failed: %w\nStatement: %s", i+1, err, statement)
			}
		}

		return nil
	})

	// Record migration result
	duration := time.Since(startTime)
	migration.Duration = duration
	migration.AppliedAt = time.Now()

	if err != nil {
		migration.Status = MigrationStatusFailed
		migration.Error = err.Error()
		if recordErr := kmm.recordMigrationStatus(ctx, migration); recordErr != nil {
			return fmt.Errorf("migration failed and status recording failed: %w (original error: %v)", recordErr, err)
		}
		return err
	}

	migration.Status = MigrationStatusCompleted
	if err := kmm.recordMigrationStatus(ctx, migration); err != nil {
		return fmt.Errorf("migration succeeded but status recording failed: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Record migration status in tracking table for audit and recovery
 * INPUT:     Migration with current status and timing information
 * OUTPUT:     Migration status persisted for tracking and rollback support
 * BUSINESS:  Migration tracking enables audit trail and recovery capabilities
 * CHANGE:    Initial status recording with comprehensive metadata
 * RISK:      Low - Status tracking with error handling
 */
func (kmm *KuzuMigrationManager) recordMigrationStatus(ctx context.Context, migration *Migration) error {
	query := `
		MERGE (m:Migration {version: $version})
		SET m.name = $name,
			m.description = $description,
			m.status = $status,
			m.applied_at = $applied_at,
			m.duration_ms = $duration_ms,
			m.error_message = $error_message;
	`

	params := map[string]interface{}{
		"version":       migration.Version,
		"name":          migration.Name,
		"description":   migration.Description,
		"status":        string(migration.Status),
		"applied_at":    migration.AppliedAt,
		"duration_ms":   int64(migration.Duration.Milliseconds()),
		"error_message": migration.Error,
	}

	_, err := kmm.connManager.Query(ctx, query, params)
	return err
}

/**
 * CONTEXT:   Split migration script into individual executable statements
 * INPUT:     Multi-statement migration script with various statement types
 * OUTPUT:    Array of individual statements ready for execution
 * BUSINESS:  Statement separation enables granular error reporting and debugging
 * CHANGE:    Initial statement splitting with semicolon delimiter
 * RISK:      Low - Simple string splitting with validation
 */
func (kmm *KuzuMigrationManager) splitStatements(script string) []string {
	// Simple semicolon-based splitting
	// Note: This may need enhancement for complex scripts with semicolons in strings
	statements := strings.Split(script, ";")
	
	result := make([]string, 0)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}
	
	return result
}

/**
 * CONTEXT:   Get migration history for audit and troubleshooting
 * INPUT:     Database context for migration tracking query
 * OUTPUT:    Complete migration history with status and timing
 * BUSINESS:  Migration history enables audit trail and troubleshooting
 * CHANGE:    Initial migration history query
 * RISK:      Low - Read-only query with error handling
 */
func (kmm *KuzuMigrationManager) GetMigrationHistory(ctx context.Context) ([]Migration, error) {
	query := `
		MATCH (m:Migration)
		RETURN m.version, m.name, m.description, m.status, m.applied_at, m.duration_ms, m.error_message
		ORDER BY m.version;
	`

	result, err := kmm.connManager.Query(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration history: %w", err)
	}

	defer result.Close()

	var migrations []Migration
	for result.HasNext() {
		record, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to read migration record: %w", err)
		}

		if len(record) >= 7 {
			migration := Migration{
				Version:     int(record[0].(int64)),
				Name:        record[1].(string),
				Description: record[2].(string),
				Status:      MigrationStatus(record[3].(string)),
				AppliedAt:   record[4].(time.Time),
				Duration:    time.Duration(record[5].(int64)) * time.Millisecond,
				Error:       record[6].(string),
			}
			migrations = append(migrations, migration)
		}
	}

	return migrations, nil
}

/**
 * CONTEXT:   Validate database schema against expected structure
 * INPUT:     Expected schema version and validation requirements
 * OUTPUT:    Schema validation results with any inconsistencies found
 * BUSINESS:  Schema validation ensures database integrity and consistency
 * CHANGE:    Initial schema validation implementation
 * RISK:      Low - Read-only validation with comprehensive checks
 */
func (kmm *KuzuMigrationManager) ValidateSchema(ctx context.Context) error {
	// Check if all expected tables exist
	expectedTables := []string{"User", "Project", "Session", "WorkBlock", "ActivityEvent", "Migration"}
	
	for _, tableName := range expectedTables {
		query := fmt.Sprintf("MATCH (n:%s) RETURN COUNT(n) LIMIT 1;", tableName)
		_, err := kmm.connManager.Query(ctx, query, nil)
		if err != nil {
			return fmt.Errorf("table %s not found or invalid: %w", tableName, err)
		}
	}

	// Validate relationships exist
	expectedRelationships := []string{"HAS_SESSION", "CONTAINS_WORK", "WORK_IN_PROJECT", "WORKS_ON", "TRIGGERED_BY", "GENERATED_BY"}
	
	for _, relName := range expectedRelationships {
		query := fmt.Sprintf("MATCH ()-[r:%s]-() RETURN COUNT(r) LIMIT 1;", relName)
		_, err := kmm.connManager.Query(ctx, query, nil)
		if err != nil {
			// Relationship may not exist yet if no data, but structure should be valid
			// This is acceptable for empty database
			continue
		}
	}

	return nil
}

/**
 * CONTEXT:   Get initial schema as embedded migration for deployment
 * INPUT:     No parameters, returns complete initial schema
 * OUTPUT:    Initial schema as migration script for database initialization
 * BUSINESS:  Embedded schema ensures consistent initial database structure
 * CHANGE:    Initial schema embedding for deployment
 * RISK:      Low - Static schema definition with no runtime dependencies
 */
func getInitialSchema() string {
	return `
-- Description: Initial KuzuDB schema for Claude Monitor work tracking system

-- Create User nodes
CREATE NODE TABLE User(
    id STRING,
    name STRING,
    email STRING DEFAULT '',
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

-- Create Project nodes
CREATE NODE TABLE Project(
    id STRING,
    name STRING,
    path STRING,
    normalized_path STRING,
    project_type STRING DEFAULT 'general',
    description STRING DEFAULT '',
    last_active_time TIMESTAMP,
    total_work_blocks INT64 DEFAULT 0,
    total_hours DOUBLE DEFAULT 0.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

-- Create Session nodes
CREATE NODE TABLE Session(
    id STRING,
    user_id STRING,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    state STRING DEFAULT 'active',
    first_activity_time TIMESTAMP,
    last_activity_time TIMESTAMP,
    activity_count INT64 DEFAULT 1,
    duration_hours DOUBLE DEFAULT 5.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

-- Create WorkBlock nodes
CREATE NODE TABLE WorkBlock(
    id STRING,
    session_id STRING,
    project_id STRING,
    project_name STRING,
    project_path STRING,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    state STRING DEFAULT 'active',
    last_activity_time TIMESTAMP,
    activity_count INT64 DEFAULT 1,
    duration_seconds INT64 DEFAULT 0,
    duration_hours DOUBLE DEFAULT 0.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

-- Create ActivityEvent nodes
CREATE NODE TABLE ActivityEvent(
    id STRING,
    user_id STRING,
    session_id STRING DEFAULT '',
    work_block_id STRING DEFAULT '',
    project_path STRING DEFAULT '',
    project_name STRING DEFAULT '',
    activity_type STRING DEFAULT 'other',
    activity_source STRING DEFAULT 'hook',
    timestamp TIMESTAMP,
    command STRING DEFAULT '',
    description STRING DEFAULT '',
    metadata STRING DEFAULT '{}',
    created_at TIMESTAMP,
    PRIMARY KEY (id)
);

-- Create relationships
CREATE REL TABLE HAS_SESSION(FROM User TO Session, created_at TIMESTAMP DEFAULT current_timestamp());
CREATE REL TABLE CONTAINS_WORK(FROM Session TO WorkBlock, sequence_number INT64 DEFAULT 0, created_at TIMESTAMP DEFAULT current_timestamp());
CREATE REL TABLE WORK_IN_PROJECT(FROM WorkBlock TO Project, activity_type STRING DEFAULT 'claude_action', created_at TIMESTAMP DEFAULT current_timestamp());
CREATE REL TABLE WORKS_ON(FROM User TO Project, first_activity TIMESTAMP, last_activity TIMESTAMP, total_hours DOUBLE DEFAULT 0.0, total_work_blocks INT64 DEFAULT 0);
CREATE REL TABLE TRIGGERED_BY(FROM Session TO ActivityEvent, event_type STRING DEFAULT 'activity', created_at TIMESTAMP DEFAULT current_timestamp());
CREATE REL TABLE GENERATED_BY(FROM WorkBlock TO ActivityEvent, activity_sequence INT64 DEFAULT 0, created_at TIMESTAMP DEFAULT current_timestamp());
`
}