/**
 * CONTEXT:   Comprehensive test suite for SQLite database connection and operations
 * INPUT:     Test scenarios covering connection, schema, migration, and CRUD operations
 * OUTPUT:    Complete validation of SQLite foundation functionality
 * BUSINESS:  Ensure database operations support work tracking business requirements
 * CHANGE:    Initial test suite for SQLite foundation validation
 * RISK:      Low - Test coverage ensures production readiness and data integrity
 */

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteDBConnection(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	t.Run("NewSQLiteDB with default config", func(t *testing.T) {
		config := DefaultConnectionConfig(dbPath)
		db, err := NewSQLiteDB(config)
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		assert.Equal(t, dbPath, db.DBPath())
		assert.NotNil(t, db.DB())
		assert.Equal(t, "America/Montevideo", db.Timezone().String())
	})

	t.Run("Connection with invalid path should fail", func(t *testing.T) {
		config := DefaultConnectionConfig("/invalid/path/test.db")
		db, err := NewSQLiteDB(config)
		assert.Error(t, err)
		assert.Nil(t, db)
	})

	t.Run("Connection with nil config should fail", func(t *testing.T) {
		db, err := NewSQLiteDB(nil)
		assert.Error(t, err)
		assert.Nil(t, db)
	})

	t.Run("Connection with empty path should fail", func(t *testing.T) {
		config := DefaultConnectionConfig("")
		db, err := NewSQLiteDB(config)
		assert.Error(t, err)
		assert.Nil(t, db)
	})
}

func TestSQLiteDBInitialization(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_init.db")
	
	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Schema should be initialized", func(t *testing.T) {
		ctx := context.Background()
		
		// Check that all required tables exist
		expectedTables := []string{"users", "projects", "sessions", "work_blocks", "activity_events", "schema_version"}
		for _, table := range expectedTables {
			var count int
			query := fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s'", table)
			err := db.DB().QueryRowContext(ctx, query).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count, "Table %s should exist", table)
		}
	})

	t.Run("Schema version should be recorded", func(t *testing.T) {
		ctx := context.Background()
		
		var version int
		var description string
		query := "SELECT version, description FROM schema_version ORDER BY version DESC LIMIT 1"
		err := db.DB().QueryRowContext(ctx, query).Scan(&version, &description)
		require.NoError(t, err)
		assert.Equal(t, 1, version)
		assert.Contains(t, description, "Initial SQLite schema")
	})

	t.Run("Foreign key constraints should be enabled", func(t *testing.T) {
		ctx := context.Background()
		
		var foreignKeys int
		err := db.DB().QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys)
		require.NoError(t, err)
		assert.Equal(t, 1, foreignKeys, "Foreign keys should be enabled")
	})
}

func TestSQLiteDBHealthCheck(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_health.db")
	
	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Health check should pass", func(t *testing.T) {
		ctx := context.Background()
		err := db.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("Health check with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		
		err := db.HealthCheck(ctx)
		assert.NoError(t, err)
	})
}

func TestSQLiteDBTransactions(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_transactions.db")
	
	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Successful transaction should commit", func(t *testing.T) {
		ctx := context.Background()
		
		err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, 
				"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
				"test-user", "testuser", time.Now(), time.Now())
			return err
		})
		require.NoError(t, err)
		
		// Verify the insert was committed
		var count int
		err = db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id = 'test-user'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Failed transaction should rollback", func(t *testing.T) {
		ctx := context.Background()
		
		err := db.WithTransaction(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx,
				"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
				"test-user-2", "testuser2", time.Now(), time.Now())
			if err != nil {
				return err
			}
			
			// Force an error to trigger rollback
			return fmt.Errorf("forced error")
		})
		assert.Error(t, err)
		
		// Verify the insert was rolled back
		var count int
		err = db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE id = 'test-user-2'").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestSQLiteDBStats(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_stats.db")
	
	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Stats should return expected structure", func(t *testing.T) {
		ctx := context.Background()
		stats, err := db.GetStats(ctx)
		require.NoError(t, err)
		
		// Check for expected stat keys
		expectedKeys := []string{
			"users_count", "projects_count", "sessions_count", 
			"work_blocks_count", "activity_events_count",
			"active_sessions", "active_work_blocks",
			"database_size_bytes", "timezone", "last_updated",
		}
		
		for _, key := range expectedKeys {
			assert.Contains(t, stats, key, "Stats should contain %s", key)
		}
		
		assert.Equal(t, "America/Montevideo", stats["timezone"])
		assert.Greater(t, stats["database_size_bytes"].(int64), int64(0))
	})
}

func TestSQLiteDBBackup(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_backup.db")
	backupPath := filepath.Join(tempDir, "backup.db")
	
	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Backup should create valid database file", func(t *testing.T) {
		err := db.Backup(backupPath)
		require.NoError(t, err)
		
		// Verify backup file exists
		_, err = os.Stat(backupPath)
		assert.NoError(t, err)
		
		// Verify backup file is valid SQLite database
		backupConfig := DefaultConnectionConfig(backupPath)
		backupDB, err := NewSQLiteDB(backupConfig)
		require.NoError(t, err)
		defer backupDB.Close()
		
		// Verify schema exists in backup
		ctx := context.Background()
		err = backupDB.HealthCheck(ctx)
		assert.NoError(t, err)
	})
}

func TestSQLiteDBTimezoneHandling(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_timezone.db")
	
	config := DefaultConnectionConfig(dbPath)
	config.Timezone = "UTC"
	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Timezone conversion should work correctly", func(t *testing.T) {
		// Test time in different timezone
		localTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.Local)
		
		// Convert to database timezone
		dbTime := db.ToDBTime(localTime)
		assert.Equal(t, time.UTC, dbTime.Location())
		
		// Convert back from database timezone
		backToLocal := db.FromDBTime(dbTime)
		assert.Equal(t, time.Local, backToLocal.Location())
		
		// Times should represent the same instant
		assert.True(t, localTime.Equal(backToLocal))
	})

	t.Run("Now() should return time in database timezone", func(t *testing.T) {
		now := db.Now()
		assert.Equal(t, time.UTC, now.Location())
	})
}

func TestSQLiteDBConnectionPooling(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_pool.db")
	
	config := DefaultConnectionConfig(dbPath)
	config.MaxOpenConns = 5
	config.MaxIdleConns = 2
	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Connection pool stats should be available", func(t *testing.T) {
		ctx := context.Background()
		stats, err := db.GetStats(ctx)
		require.NoError(t, err)
		
		assert.Contains(t, stats, "max_open_connections")
		assert.Contains(t, stats, "open_connections")
		assert.Contains(t, stats, "in_use_connections")
		assert.Contains(t, stats, "idle_connections")
		
		// Verify pool configuration
		assert.Equal(t, 5, stats["max_open_connections"])
	})
}

func TestSQLiteDBErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_errors.db")
	
	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Invalid SQL should return descriptive error", func(t *testing.T) {
		ctx := context.Background()
		_, err := db.DB().ExecContext(ctx, "INVALID SQL STATEMENT")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "syntax error")
	})

	t.Run("Foreign key constraint violation should be handled", func(t *testing.T) {
		ctx := context.Background()
		
		// Try to insert session with non-existent user
		_, err := db.DB().ExecContext(ctx, `
			INSERT INTO sessions (id, user_id, start_time, end_time, state, 
								  first_activity_time, last_activity_time, 
								  activity_count, duration_hours, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "test-session", "non-existent-user", time.Now(), time.Now().Add(5*time.Hour),
			"active", time.Now(), time.Now(), 1, 5.0, time.Now(), time.Now())
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "FOREIGN KEY constraint failed")
	})
}

// Helper function for creating test database
func createTestDB(t *testing.T) *SQLiteDB {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	
	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	
	return db
}

// Benchmark tests for performance validation
func BenchmarkSQLiteDBConnection(b *testing.B) {
	tempDir := b.TempDir()
	
	b.Run("NewSQLiteDB", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			dbPath := filepath.Join(tempDir, fmt.Sprintf("bench_%d.db", i))
			config := DefaultConnectionConfig(dbPath)
			db, err := NewSQLiteDB(config)
			if err != nil {
				b.Fatal(err)
			}
			db.Close()
		}
	})
}

func BenchmarkSQLiteDBQuery(b *testing.B) {
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, "bench.db")
	
	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteDB(config)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	
	ctx := context.Background()
	
	// Insert test data
	_, err = db.DB().ExecContext(ctx, 
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"bench-user", "benchuser", time.Now(), time.Now())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	
	b.Run("Simple SELECT", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var count int
			err := db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}