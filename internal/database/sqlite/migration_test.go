/**
 * CONTEXT:   Test suite for data migration from gob backup to SQLite
 * INPUT:     Test scenarios covering legacy data loading and conversion to SQLite
 * OUTPUT:    Validation of migration process and data integrity preservation
 * BUSINESS:  Ensure complete historical work data is preserved during system upgrade
 * CHANGE:    Initial migration test suite with synthetic legacy data
 * RISK:      Medium - Critical migration testing ensures no data loss during transition
 */

package sqlite

import (
	"context"
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadLegacyData(t *testing.T) {
	t.Run("Load valid gob backup should succeed", func(t *testing.T) {
		// Create test gob file
		gobFile := createTestGobFile(t)
		
		data, err := LoadLegacyData(gobFile)
		require.NoError(t, err)
		require.NotNil(t, data)
		
		assert.Len(t, data.Sessions, 2)
		assert.Len(t, data.WorkBlocks, 3)
		assert.Len(t, data.Activities, 5)
		assert.Equal(t, "1.0.0", data.Version)
	})

	t.Run("Load non-existent file should fail", func(t *testing.T) {
		data, err := LoadLegacyData("/non/existent/path.db")
		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "backup file does not exist")
	})

	t.Run("Load invalid gob file should fail", func(t *testing.T) {
		// Create invalid gob file
		tempDir := t.TempDir()
		invalidFile := filepath.Join(tempDir, "invalid.db")
		
		file, err := os.Create(invalidFile)
		require.NoError(t, err)
		file.WriteString("not a valid gob file")
		file.Close()
		
		data, err := LoadLegacyData(invalidFile)
		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Contains(t, err.Error(), "failed to decode backup data")
	})
}

func TestMigrateFromGobBackup(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	t.Run("Complete migration should succeed", func(t *testing.T) {
		// Create test gob file
		gobFile := createTestGobFile(t)
		
		result, err := MigrateFromGobBackup(db, gobFile)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		// Verify migration results
		assert.Equal(t, 1, result.UsersCreated, "Should create 1 unique user")
		assert.Equal(t, 2, result.ProjectsCreated, "Should create 2 unique projects")
		assert.Equal(t, 2, result.SessionsMigrated, "Should migrate 2 sessions")
		assert.Equal(t, 3, result.WorkBlocksMigrated, "Should migrate 3 work blocks")
		assert.Equal(t, 5, result.ActivitiesMigrated, "Should migrate 5 activities")
		assert.True(t, result.DataIntegrityValid, "Data integrity should be valid")
		assert.Empty(t, result.Errors, "Should have no errors")
		assert.Greater(t, result.MigrationDuration, time.Duration(0))

		// Verify data was actually inserted
		ctx := context.Background()
		
		// Check users
		var userCount int
		err = db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
		require.NoError(t, err)
		assert.Equal(t, 1, userCount)

		// Check projects
		var projectCount int
		err = db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM projects").Scan(&projectCount)
		require.NoError(t, err)
		assert.Equal(t, 2, projectCount)

		// Check sessions
		var sessionCount int
		err = db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&sessionCount)
		require.NoError(t, err)
		assert.Equal(t, 2, sessionCount)

		// Check work blocks
		var workBlockCount int
		err = db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM work_blocks").Scan(&workBlockCount)
		require.NoError(t, err)
		assert.Equal(t, 3, workBlockCount)

		// Check activities
		var activityCount int
		err = db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM activity_events").Scan(&activityCount)
		require.NoError(t, err)
		assert.Equal(t, 5, activityCount)
	})

	t.Run("Migration with non-existent file should fail", func(t *testing.T) {
		result, err := MigrateFromGobBackup(db, "/non/existent/file.db")
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, err.Error(), "failed to load legacy data")
	})
}

func TestSessionConversion(t *testing.T) {
	montevideoTZ, err := time.LoadLocation("America/Montevideo")
	require.NoError(t, err)

	t.Run("Convert legacy session with IsActive=true", func(t *testing.T) {
		startTime := time.Now().Add(-2 * time.Hour)
		legacy := &LegacySession{
			ID:                "test-session",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour),
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime.Add(time.Hour),
			ActivityCount:     10,
			DurationHours:     5.0,
			IsActive:          true,
			CreatedAt:         startTime,
			UpdatedAt:         startTime.Add(time.Hour),
		}

		converted := convertSession(legacy, montevideoTZ)
		
		assert.Equal(t, legacy.ID, converted.ID)
		assert.Equal(t, legacy.UserID, converted.UserID)
		assert.WithinDuration(t, legacy.StartTime, converted.StartTime, time.Second)
		assert.WithinDuration(t, legacy.StartTime.Add(5*time.Hour), converted.EndTime, time.Second)
		assert.Equal(t, "active", converted.State) // Still active since end time is in future
		assert.Equal(t, legacy.ActivityCount, converted.ActivityCount)
		assert.Equal(t, 5.0, converted.DurationHours) // Always 5.0 for sessions
		assert.Equal(t, montevideoTZ, converted.StartTime.Location())
	})

	t.Run("Convert legacy session with IsActive=true but expired", func(t *testing.T) {
		startTime := time.Now().Add(-6 * time.Hour)
		legacy := &LegacySession{
			ID:                "expired-session",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour), // Expired 1 hour ago
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime.Add(2 * time.Hour),
			ActivityCount:     5,
			DurationHours:     5.0,
			IsActive:          true, // Legacy flag doesn't matter
			CreatedAt:         startTime,
			UpdatedAt:         startTime.Add(2 * time.Hour),
		}

		converted := convertSession(legacy, montevideoTZ)
		
		assert.Equal(t, "expired", converted.State) // Should be expired based on time
		assert.Equal(t, 5.0, converted.DurationHours)
	})

	t.Run("Convert legacy session with IsActive=false", func(t *testing.T) {
		startTime := time.Now().Add(-3 * time.Hour)
		legacy := &LegacySession{
			ID:                "finished-session",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour),
			State:             "finished",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime.Add(time.Hour),
			ActivityCount:     8,
			DurationHours:     5.0,
			IsActive:          false,
			CreatedAt:         startTime,
			UpdatedAt:         startTime.Add(time.Hour),
		}

		converted := convertSession(legacy, montevideoTZ)
		
		assert.Equal(t, "finished", converted.State)
		assert.Equal(t, 5.0, converted.DurationHours)
	})
}

func TestWorkBlockConversion(t *testing.T) {
	montevideoTZ, err := time.LoadLocation("America/Montevideo")
	require.NoError(t, err)

	projectLookup := map[string]string{
		"TestProject|/path/to/test": "proj_test123",
	}

	t.Run("Convert legacy work block with IsActive=true", func(t *testing.T) {
		startTime := time.Now().Add(-time.Hour)
		legacy := &LegacyWorkBlock{
			ID:                   "test-workblock",
			SessionID:            "test-session",
			ProjectName:          "TestProject",
			ProjectPath:          "/path/to/test",
			StartTime:            startTime,
			EndTime:              time.Time{}, // Zero time for active block
			State:                "active",
			LastActivityTime:     startTime.Add(30 * time.Minute),
			ActivityCount:        5,
			ActivityTypeCounters: map[string]int64{"command": 3, "edit": 2},
			DurationSeconds:      1800, // 30 minutes
			DurationHours:        0.5,
			IsActive:             true,
			CreatedAt:            startTime,
			UpdatedAt:            startTime.Add(30 * time.Minute),
		}

		converted := convertWorkBlock(legacy, projectLookup, montevideoTZ)
		
		assert.Equal(t, legacy.ID, converted.ID)
		assert.Equal(t, legacy.SessionID, converted.SessionID)
		assert.Equal(t, "proj_test123", converted.ProjectID)
		assert.WithinDuration(t, legacy.StartTime, converted.StartTime, time.Second)
		assert.Nil(t, converted.EndTime) // Should be nil for active blocks
		assert.Equal(t, "active", converted.State)
		assert.Equal(t, legacy.ActivityCount, converted.ActivityCount)
		assert.Equal(t, legacy.DurationSeconds, converted.DurationSeconds)
		assert.Equal(t, legacy.DurationHours, converted.DurationHours)
		assert.Equal(t, montevideoTZ, converted.StartTime.Location())
	})

	t.Run("Convert legacy work block with IsActive=false", func(t *testing.T) {
		startTime := time.Now().Add(-2 * time.Hour)
		endTime := startTime.Add(time.Hour)
		legacy := &LegacyWorkBlock{
			ID:               "finished-workblock",
			SessionID:        "test-session",
			ProjectName:      "TestProject",
			ProjectPath:      "/path/to/test",
			StartTime:        startTime,
			EndTime:          endTime,
			State:            "finished",
			LastActivityTime: endTime,
			ActivityCount:    10,
			DurationSeconds:  3600, // 1 hour
			DurationHours:    1.0,
			IsActive:         false,
			CreatedAt:        startTime,
			UpdatedAt:        endTime,
		}

		converted := convertWorkBlock(legacy, projectLookup, montevideoTZ)
		
		assert.Equal(t, "finished", converted.State)
		assert.NotNil(t, converted.EndTime)
		assert.WithinDuration(t, endTime, *converted.EndTime, time.Second)
	})

	t.Run("Convert work block with unknown project", func(t *testing.T) {
		startTime := time.Now()
		legacy := &LegacyWorkBlock{
			ID:               "unknown-project-workblock",
			SessionID:        "test-session",
			ProjectName:      "UnknownProject",
			ProjectPath:      "/unknown/path",
			StartTime:        startTime,
			EndTime:          time.Time{},
			State:            "active",
			LastActivityTime: startTime,
			ActivityCount:    1,
			DurationSeconds:  0,
			DurationHours:    0.0,
			IsActive:         true,
			CreatedAt:        startTime,
			UpdatedAt:        startTime,
		}

		converted := convertWorkBlock(legacy, projectLookup, montevideoTZ)
		
		// Should generate new project ID
		assert.Contains(t, converted.ProjectID, "proj_")
		assert.NotEqual(t, "proj_test123", converted.ProjectID)
	})
}

func TestActivityEventConversion(t *testing.T) {
	montevideoTZ, err := time.LoadLocation("America/Montevideo")
	require.NoError(t, err)

	projectLookup := map[string]string{
		"TestProject|/path/to/test": "proj_test123",
	}

	t.Run("Convert legacy activity event", func(t *testing.T) {
		timestamp := time.Now().Add(-time.Hour)
		legacy := &LegacyActivityEvent{
			ID:             "test-activity",
			UserID:         "test-user",
			SessionID:      "test-session",
			WorkBlockID:    "test-workblock",
			ProjectPath:    "/path/to/test",
			ProjectName:    "TestProject",
			ActivityType:   "command",
			ActivitySource: "hook",
			Timestamp:      timestamp,
			Command:        "echo test",
			Description:    "Test command",
			Metadata:       `{"key": "value"}`,
			CreatedAt:      timestamp,
		}

		converted := convertActivityEvent(legacy, projectLookup, montevideoTZ)
		
		assert.Equal(t, legacy.ID, converted.ID)
		assert.Equal(t, legacy.UserID, converted.UserID)
		assert.Equal(t, legacy.SessionID, converted.SessionID)
		assert.Equal(t, legacy.WorkBlockID, converted.WorkBlockID)
		assert.Equal(t, "proj_test123", converted.ProjectID)
		assert.Equal(t, legacy.ActivityType, converted.ActivityType)
		assert.Equal(t, legacy.ActivitySource, converted.ActivitySource)
		assert.WithinDuration(t, legacy.Timestamp, converted.Timestamp, time.Second)
		assert.Equal(t, legacy.Command, converted.Command)
		assert.Equal(t, legacy.Description, converted.Description)
		assert.Equal(t, legacy.Metadata, converted.Metadata)
		assert.Equal(t, montevideoTZ, converted.Timestamp.Location())
	})

	t.Run("Convert activity with empty project", func(t *testing.T) {
		timestamp := time.Now()
		legacy := &LegacyActivityEvent{
			ID:             "no-project-activity",
			UserID:         "test-user",
			SessionID:      "test-session",
			WorkBlockID:    "test-workblock",
			ProjectPath:    "",
			ProjectName:    "",
			ActivityType:   "other",
			ActivitySource: "cli",
			Timestamp:      timestamp,
			Command:        "",
			Description:    "Activity without project",
			Metadata:       "",
			CreatedAt:      timestamp,
		}

		converted := convertActivityEvent(legacy, projectLookup, montevideoTZ)
		
		assert.Equal(t, "", converted.ProjectID) // Should be empty for activities without project
	})
}

func TestProjectIDGeneration(t *testing.T) {
	t.Run("Generate consistent project IDs", func(t *testing.T) {
		id1 := generateProjectID("TestProject", "/path/to/project")
		id2 := generateProjectID("TestProject", "/path/to/project")
		
		assert.Equal(t, id1, id2, "Same name+path should generate same ID")
		assert.Contains(t, id1, "proj_")
	})

	t.Run("Generate different IDs for different projects", func(t *testing.T) {
		id1 := generateProjectID("Project1", "/path/1")
		id2 := generateProjectID("Project2", "/path/2")
		
		assert.NotEqual(t, id1, id2)
	})

	t.Run("Handle empty project name", func(t *testing.T) {
		id := generateProjectID("", "/some/path")
		assert.Equal(t, "unknown-project", id)
	})
}

func TestMigrationValidation(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Insert test data
	ctx := context.Background()
	_, err := db.DB().ExecContext(ctx,
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"test-user", "testuser", time.Now(), time.Now())
	require.NoError(t, err)

	_, err = db.DB().ExecContext(ctx,
		"INSERT INTO projects (id, name, path, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		"proj_test", "TestProject", "/test/path", time.Now(), time.Now())
	require.NoError(t, err)

	now := time.Now()
	_, err = db.DB().ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, start_time, end_time, state, 
							  first_activity_time, last_activity_time, 
							  activity_count, duration_hours, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-session", "test-user", now, now.Add(5*time.Hour), "active",
		now, now.Add(time.Hour), 5, 5.0, now, now)
	require.NoError(t, err)

	_, err = db.DB().ExecContext(ctx, `
		INSERT INTO work_blocks (id, session_id, project_id, start_time, end_time, state,
								 last_activity_time, activity_count, duration_seconds, duration_hours,
								 claude_processing_seconds, claude_processing_hours, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-workblock", "test-session", "proj_test", now, now.Add(time.Hour), "finished",
		now.Add(time.Hour), 3, 3600, 1.0, 0, 0.0, now, now)
	require.NoError(t, err)

	t.Run("Validation should pass with matching data", func(t *testing.T) {
		legacyData := &LegacyDatabaseData{
			Sessions: map[string]*LegacySession{
				"test-session": {},
			},
			WorkBlocks: map[string]*LegacyWorkBlock{
				"test-workblock": {},
			},
			Activities: []*LegacyActivityEvent{},
		}

		result := &MigrationResult{}
		err := validateMigrationIntegrity(db, legacyData, result)
		assert.NoError(t, err)
	})

	t.Run("Validation should fail with mismatched counts", func(t *testing.T) {
		legacyData := &LegacyDatabaseData{
			Sessions: map[string]*LegacySession{
				"test-session": {},
				"extra-session": {}, // Extra session not in database
			},
			WorkBlocks: map[string]*LegacyWorkBlock{
				"test-workblock": {},
			},
			Activities: []*LegacyActivityEvent{},
		}

		result := &MigrationResult{}
		err := validateMigrationIntegrity(db, legacyData, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session count mismatch")
	})
}

// Helper function to create test gob backup file
func createTestGobFile(t *testing.T) string {
	tempDir := t.TempDir()
	gobFile := filepath.Join(tempDir, "test_backup.db")

	// Create test legacy data
	now := time.Now()
	legacyData := &LegacyDatabaseData{
		Sessions: map[string]*LegacySession{
			"session-1": {
				ID:                "session-1",
				UserID:            "test-user",
				StartTime:         now.Add(-5 * time.Hour),
				EndTime:           now,
				State:             "finished",
				FirstActivityTime: now.Add(-5 * time.Hour),
				LastActivityTime:  now.Add(-time.Hour),
				ActivityCount:     10,
				DurationHours:     5.0,
				IsActive:          false,
				CreatedAt:         now.Add(-5 * time.Hour),
				UpdatedAt:         now,
			},
			"session-2": {
				ID:                "session-2",
				UserID:            "test-user",
				StartTime:         now.Add(-2 * time.Hour),
				EndTime:           now.Add(3 * time.Hour),
				State:             "active",
				FirstActivityTime: now.Add(-2 * time.Hour),
				LastActivityTime:  now.Add(-30 * time.Minute),
				ActivityCount:     5,
				DurationHours:     5.0,
				IsActive:          true,
				CreatedAt:         now.Add(-2 * time.Hour),
				UpdatedAt:         now.Add(-30 * time.Minute),
			},
		},
		WorkBlocks: map[string]*LegacyWorkBlock{
			"wb-1": {
				ID:               "wb-1",
				SessionID:        "session-1",
				ProjectName:      "Project1",
				ProjectPath:      "/path/to/project1",
				StartTime:        now.Add(-5 * time.Hour),
				EndTime:          now.Add(-3 * time.Hour),
				State:            "finished",
				LastActivityTime: now.Add(-3 * time.Hour),
				ActivityCount:    5,
				DurationSeconds:  7200, // 2 hours
				DurationHours:    2.0,
				IsActive:         false,
				CreatedAt:        now.Add(-5 * time.Hour),
				UpdatedAt:        now.Add(-3 * time.Hour),
			},
			"wb-2": {
				ID:               "wb-2",
				SessionID:        "session-1",
				ProjectName:      "Project2",
				ProjectPath:      "/path/to/project2",
				StartTime:        now.Add(-2 * time.Hour),
				EndTime:          now,
				State:            "finished",
				LastActivityTime: now,
				ActivityCount:    3,
				DurationSeconds:  7200,
				DurationHours:    2.0,
				IsActive:         false,
				CreatedAt:        now.Add(-2 * time.Hour),
				UpdatedAt:        now,
			},
			"wb-3": {
				ID:               "wb-3",
				SessionID:        "session-2",
				ProjectName:      "Project1",
				ProjectPath:      "/path/to/project1",
				StartTime:        now.Add(-time.Hour),
				EndTime:          time.Time{}, // Active block
				State:            "active",
				LastActivityTime: now.Add(-30 * time.Minute),
				ActivityCount:    2,
				DurationSeconds:  1800, // 30 minutes
				DurationHours:    0.5,
				IsActive:         true,
				CreatedAt:        now.Add(-time.Hour),
				UpdatedAt:        now.Add(-30 * time.Minute),
			},
		},
		Activities: []*LegacyActivityEvent{
			{
				ID:             "act-1",
				UserID:         "test-user",
				SessionID:      "session-1",
				WorkBlockID:    "wb-1",
				ProjectPath:    "/path/to/project1",
				ProjectName:    "Project1",
				ActivityType:   "command",
				ActivitySource: "hook",
				Timestamp:      now.Add(-4 * time.Hour),
				Command:        "test command 1",
				Description:    "Test activity 1",
				CreatedAt:      now.Add(-4 * time.Hour),
			},
			{
				ID:             "act-2",
				UserID:         "test-user",
				SessionID:      "session-1",
				WorkBlockID:    "wb-1",
				ProjectPath:    "/path/to/project1",
				ProjectName:    "Project1",
				ActivityType:   "edit",
				ActivitySource: "hook",
				Timestamp:      now.Add(-3 * time.Hour),
				Command:        "edit file",
				Description:    "Test activity 2",
				CreatedAt:      now.Add(-3 * time.Hour),
			},
			{
				ID:             "act-3",
				UserID:         "test-user",
				SessionID:      "session-1",
				WorkBlockID:    "wb-2",
				ProjectPath:    "/path/to/project2",
				ProjectName:    "Project2",
				ActivityType:   "command",
				ActivitySource: "hook",
				Timestamp:      now.Add(-90 * time.Minute),
				Command:        "test command 2",
				Description:    "Test activity 3",
				CreatedAt:      now.Add(-90 * time.Minute),
			},
			{
				ID:             "act-4",
				UserID:         "test-user",
				SessionID:      "session-2",
				WorkBlockID:    "wb-3",
				ProjectPath:    "/path/to/project1",
				ProjectName:    "Project1",
				ActivityType:   "navigation",
				ActivitySource: "hook",
				Timestamp:      now.Add(-45 * time.Minute),
				Command:        "cd /path",
				Description:    "Test activity 4",
				CreatedAt:      now.Add(-45 * time.Minute),
			},
			{
				ID:             "act-5",
				UserID:         "test-user",
				SessionID:      "session-2",
				WorkBlockID:    "wb-3",
				ProjectPath:    "/path/to/project1",
				ProjectName:    "Project1",
				ActivityType:   "other",
				ActivitySource: "cli",
				Timestamp:      now.Add(-30 * time.Minute),
				Command:        "",
				Description:    "Test activity 5",
				CreatedAt:      now.Add(-30 * time.Minute),
			},
		},
		LastUpdated: now,
		Version:     "1.0.0",
	}

	// Write to gob file
	file, err := os.Create(gobFile)
	require.NoError(t, err)
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(legacyData)
	require.NoError(t, err)

	return gobFile
}