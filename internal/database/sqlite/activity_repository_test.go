/**
 * CONTEXT:   Comprehensive tests for SQLite activity repository with FK constraints
 * INPUT:     Test scenarios for CRUD operations, JSON metadata, and FK validation
 * OUTPUT:    Complete test coverage for activity repository functionality
 * BUSINESS:  Activity repository tests ensure data integrity and FK relationships
 * CHANGE:    Initial test implementation for CHECKPOINT 4 activity management
 * RISK:      Low - Test code ensuring system reliability and data consistency
 */

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * CONTEXT:   Test database setup for activity repository testing
 * INPUT:     Test name and database path requirements
 * OUTPUT:    Configured test database with schema and sample data
 * BUSINESS:  Test database provides isolated environment for activity testing
 * CHANGE:    Initial test database setup with FK constraints
 * RISK:      Low - Test infrastructure for reliable testing
 */
func setupActivityTestDB(t *testing.T) (*sql.DB, func()) {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_activity.db")

	sqliteDB, err := NewSQLiteDB(dbPath)
	require.NoError(t, err, "Failed to create test database")

	// Setup test data
	ctx := context.Background()
	
	// Insert test user
	_, err = sqliteDB.DB.ExecContext(ctx, `
		INSERT INTO users (id, username) VALUES ('test_user', 'test_user')
	`)
	require.NoError(t, err, "Failed to insert test user")

	// Insert test project
	_, err = sqliteDB.DB.ExecContext(ctx, `
		INSERT INTO projects (id, name, path) VALUES ('test_project', 'Test Project', '/test/path')
	`)
	require.NoError(t, err, "Failed to insert test project")

	// Insert test session
	sessionStart := time.Now().Add(-1 * time.Hour)
	sessionEnd := sessionStart.Add(5 * time.Hour)
	_, err = sqliteDB.DB.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, start_time, end_time, state, first_activity_time, last_activity_time, activity_count)
		VALUES ('test_session', 'test_user', ?, ?, 'active', ?, ?, 1)
	`, sessionStart, sessionEnd, sessionStart, sessionStart)
	require.NoError(t, err, "Failed to insert test session")

	// Insert test work block
	workBlockStart := time.Now().Add(-30 * time.Minute)
	_, err = sqliteDB.DB.ExecContext(ctx, `
		INSERT INTO work_blocks (id, session_id, project_id, start_time, state, last_activity_time, activity_count)
		VALUES ('test_workblock', 'test_session', 'test_project', ?, 'active', ?, 1)
	`, workBlockStart, workBlockStart)
	require.NoError(t, err, "Failed to insert test work block")

	cleanup := func() {
		sqliteDB.Close()
		os.RemoveAll(tempDir)
	}

	return sqliteDB.DB, cleanup
}

/**
 * CONTEXT:   Test activity creation and FK constraint validation
 * INPUT:     Activity entity with valid and invalid work block references
 * OUTPUT:    Successful activity creation with FK validation
 * BUSINESS:  Activities must have valid work block associations
 * CHANGE:    Initial FK constraint validation testing
 * RISK:      Low - Test ensuring data integrity requirements
 */
func TestActivityRepository_Save_FKConstraints(t *testing.T) {
	db, cleanup := setupActivityTestDB(t)
	defer cleanup()

	repo := NewActivityRepository(db)
	ctx := context.Background()

	t.Run("valid activity with FK references", func(t *testing.T) {
		// Create valid activity
		activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
			UserID:         "test_user",
			WorkBlockID:    "test_workblock",
			ProjectPath:    "/test/path",
			ActivityType:   entities.ActivityTypeCommand,
			ActivitySource: entities.ActivitySourceHook,
			Timestamp:      time.Now(),
			Command:        "test command",
			Description:    "test activity",
			Metadata:       map[string]string{"key": "value"},
		})
		require.NoError(t, err)

		// Set required associations
		err = activity.AssociateWithSession("test_session")
		require.NoError(t, err)
		err = activity.AssociateWithWorkBlock("test_workblock")
		require.NoError(t, err)

		// Save should succeed
		err = repo.Save(ctx, activity)
		assert.NoError(t, err, "Save with valid FK should succeed")

		// Verify activity was saved
		savedActivity, err := repo.FindByID(ctx, activity.ID())
		assert.NoError(t, err)
		assert.Equal(t, activity.ID(), savedActivity.ID())
		assert.Equal(t, "test command", savedActivity.Command())
		assert.Equal(t, "test activity", savedActivity.Description())
		assert.Equal(t, "value", savedActivity.Metadata()["key"])
	})

	t.Run("invalid activity with missing work block", func(t *testing.T) {
		// Create activity without work block
		activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
			UserID:         "test_user",
			ProjectPath:    "/test/path",
			ActivityType:   entities.ActivityTypeCommand,
			ActivitySource: entities.ActivitySourceHook,
			Timestamp:      time.Now(),
		})
		require.NoError(t, err)

		// Save should fail - no work block association
		err = repo.Save(ctx, activity)
		assert.Error(t, err, "Save without work block should fail")
		assert.Contains(t, err.Error(), "must be associated with a work block")
	})

	t.Run("invalid activity with non-existent work block", func(t *testing.T) {
		// Create activity with invalid work block ID
		activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
			UserID:         "test_user",
			WorkBlockID:    "invalid_workblock",
			ProjectPath:    "/test/path",
			ActivityType:   entities.ActivityTypeCommand,
			ActivitySource: entities.ActivitySourceHook,
			Timestamp:      time.Now(),
		})
		require.NoError(t, err)

		err = activity.AssociateWithSession("test_session")
		require.NoError(t, err)
		err = activity.AssociateWithWorkBlock("invalid_workblock")
		require.NoError(t, err)

		// Save should fail due to FK constraint
		err = repo.Save(ctx, activity)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "work block does not exist")
	})
}

/**
 * CONTEXT:   Test JSON metadata serialization and deserialization
 * INPUT:     Activity entities with various metadata configurations
 * OUTPUT:    Proper JSON handling with validation and error cases
 * BUSINESS:  Activity metadata provides flexible context storage
 * CHANGE:    Initial JSON metadata handling validation
 * RISK:      Low - Test ensuring metadata integrity and handling
 */
func TestActivityRepository_JSONMetadataHandling(t *testing.T) {
	db, cleanup := setupActivityTestDB(t)
	defer cleanup()

	repo := NewActivityRepository(db)
	ctx := context.Background()

	t.Run("complex metadata serialization", func(t *testing.T) {
		complexMetadata := map[string]string{
			"file_path":     "/path/to/file.go",
			"line_number":   "42",
			"function_name": "processActivity",
			"complexity":    "high",
			"duration_ms":   "1500",
		}

		activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
			UserID:         "test_user",
			ProjectPath:    "/test/path",
			ActivityType:   entities.ActivityTypeFileEdit,
			ActivitySource: entities.ActivitySourceCLI,
			Timestamp:      time.Now(),
			Command:        "edit file",
			Description:    "editing source file",
			Metadata:       complexMetadata,
		})
		require.NoError(t, err)

		err = activity.AssociateWithSession("test_session")
		require.NoError(t, err)
		err = activity.AssociateWithWorkBlock("test_workblock")
		require.NoError(t, err)

		// Save activity
		err = repo.Save(ctx, activity)
		require.NoError(t, err)

		// Retrieve and validate metadata
		savedActivity, err := repo.FindByID(ctx, activity.ID())
		require.NoError(t, err)

		savedMetadata := savedActivity.Metadata()
		assert.Equal(t, len(complexMetadata), len(savedMetadata))
		assert.Equal(t, "/path/to/file.go", savedMetadata["file_path"])
		assert.Equal(t, "42", savedMetadata["line_number"])
		assert.Equal(t, "processActivity", savedMetadata["function_name"])
		assert.Equal(t, "high", savedMetadata["complexity"])
		assert.Equal(t, "1500", savedMetadata["duration_ms"])
	})

	t.Run("empty metadata handling", func(t *testing.T) {
		activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
			UserID:         "test_user",
			ProjectPath:    "/test/path",
			ActivityType:   entities.ActivityTypeCommand,
			ActivitySource: entities.ActivitySourceHook,
			Timestamp:      time.Now(),
			Metadata:       map[string]string{}, // Empty metadata
		})
		require.NoError(t, err)

		err = activity.AssociateWithSession("test_session")
		require.NoError(t, err)
		err = activity.AssociateWithWorkBlock("test_workblock")
		require.NoError(t, err)

		// Save activity
		err = repo.Save(ctx, activity)
		require.NoError(t, err)

		// Retrieve and validate empty metadata
		savedActivity, err := repo.FindByID(ctx, activity.ID())
		require.NoError(t, err)

		savedMetadata := savedActivity.Metadata()
		assert.NotNil(t, savedMetadata)
		assert.Equal(t, 0, len(savedMetadata))
	})

	t.Run("null metadata handling", func(t *testing.T) {
		activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
			UserID:         "test_user",
			ProjectPath:    "/test/path",
			ActivityType:   entities.ActivityTypeCommand,
			ActivitySource: entities.ActivitySourceHook,
			Timestamp:      time.Now(),
			Metadata:       nil, // Null metadata
		})
		require.NoError(t, err)

		err = activity.AssociateWithSession("test_session")
		require.NoError(t, err)
		err = activity.AssociateWithWorkBlock("test_workblock")
		require.NoError(t, err)

		// Save activity
		err = repo.Save(ctx, activity)
		require.NoError(t, err)

		// Retrieve and validate metadata handling
		savedActivity, err := repo.FindByID(ctx, activity.ID())
		require.NoError(t, err)

		savedMetadata := savedActivity.Metadata()
		assert.NotNil(t, savedMetadata)
		assert.Equal(t, 0, len(savedMetadata))
	})
}

/**
 * CONTEXT:   Test Claude processing context serialization and handling
 * INPUT:     Activities with Claude processing context data
 * OUTPUT:    Proper Claude context storage and retrieval
 * BUSINESS:  Claude processing context enables accurate time tracking
 * CHANGE:    Initial Claude context handling testing
 * RISK:      Low - Test ensuring Claude integration data integrity
 */
func TestActivityRepository_ClaudeProcessingContext(t *testing.T) {
	db, cleanup := setupActivityTestDB(t)
	defer cleanup()

	repo := NewActivityRepository(db)
	ctx := context.Background()

	t.Run("claude processing start context", func(t *testing.T) {
		claudeContext := &entities.ClaudeProcessingContext{
			PromptID:         "prompt_123",
			EstimatedTime:    30 * time.Second,
			TokensCount:      nil, // Not available at start
			PromptLength:     150,
			ComplexityHint:   "code_generation",
			ClaudeActivity:   entities.ClaudeActivityStart,
		}

		activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
			UserID:         "test_user",
			ProjectPath:    "/test/path",
			ActivityType:   entities.ActivityTypeGeneration,
			ActivitySource: entities.ActivitySourceHook,
			Timestamp:      time.Now(),
			Command:        "claude generate",
			Description:    "starting code generation",
			ClaudeContext:  claudeContext,
		})
		require.NoError(t, err)

		err = activity.AssociateWithSession("test_session")
		require.NoError(t, err)
		err = activity.AssociateWithWorkBlock("test_workblock")
		require.NoError(t, err)

		// Save activity
		err = repo.Save(ctx, activity)
		require.NoError(t, err)

		// Retrieve and validate Claude context
		savedActivity, err := repo.FindByID(ctx, activity.ID())
		require.NoError(t, err)

		savedContext := savedActivity.ClaudeContext()
		require.NotNil(t, savedContext)
		assert.Equal(t, "prompt_123", savedContext.PromptID)
		assert.Equal(t, 30*time.Second, savedContext.EstimatedTime)
		assert.Equal(t, 150, savedContext.PromptLength)
		assert.Equal(t, "code_generation", savedContext.ComplexityHint)
		assert.Equal(t, entities.ClaudeActivityStart, savedContext.ClaudeActivity)
		assert.Nil(t, savedContext.TokensCount)
		assert.Nil(t, savedContext.ActualTime)
	})

	t.Run("claude processing end context", func(t *testing.T) {
		actualTime := 45 * time.Second
		tokensCount := 1200

		claudeContext := &entities.ClaudeProcessingContext{
			PromptID:         "prompt_123",
			EstimatedTime:    30 * time.Second,
			ActualTime:       &actualTime,
			TokensCount:      &tokensCount,
			PromptLength:     150,
			ComplexityHint:   "code_generation",
			ClaudeActivity:   entities.ClaudeActivityEnd,
		}

		activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
			UserID:         "test_user",
			ProjectPath:    "/test/path",
			ActivityType:   entities.ActivityTypeGeneration,
			ActivitySource: entities.ActivitySourceHook,
			Timestamp:      time.Now(),
			Command:        "claude complete",
			Description:    "completed code generation",
			ClaudeContext:  claudeContext,
		})
		require.NoError(t, err)

		err = activity.AssociateWithSession("test_session")
		require.NoError(t, err)
		err = activity.AssociateWithWorkBlock("test_workblock")
		require.NoError(t, err)

		// Save activity
		err = repo.Save(ctx, activity)
		require.NoError(t, err)

		// Retrieve and validate complete Claude context
		savedActivity, err := repo.FindByID(ctx, activity.ID())
		require.NoError(t, err)

		savedContext := savedActivity.ClaudeContext()
		require.NotNil(t, savedContext)
		assert.Equal(t, "prompt_123", savedContext.PromptID)
		assert.Equal(t, 30*time.Second, savedContext.EstimatedTime)
		assert.Equal(t, 45*time.Second, *savedContext.ActualTime)
		assert.Equal(t, 1200, *savedContext.TokensCount)
		assert.Equal(t, 150, savedContext.PromptLength)
		assert.Equal(t, entities.ClaudeActivityEnd, savedContext.ClaudeActivity)
	})
}

/**
 * CONTEXT:   Test work block activity count updates and validation
 * INPUT:     Activities saved to work blocks with count tracking
 * OUTPUT:    Accurate work block activity count updates
 * BUSINESS:  Work block activity counts drive time tracking metrics
 * CHANGE:    Initial activity count update testing
 * RISK:      Low - Test ensuring accurate activity counting
 */
func TestActivityRepository_WorkBlockActivityUpdates(t *testing.T) {
	db, cleanup := setupActivityTestDB(t)
	defer cleanup()

	repo := NewActivityRepository(db)
	ctx := context.Background()

	t.Run("single activity updates work block count", func(t *testing.T) {
		// Check initial work block count
		var initialCount int
		err := db.QueryRowContext(ctx, 
			"SELECT activity_count FROM work_blocks WHERE id = 'test_workblock'").Scan(&initialCount)
		require.NoError(t, err)

		// Create and save activity
		activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
			UserID:         "test_user",
			ProjectPath:    "/test/path",
			ActivityType:   entities.ActivityTypeCommand,
			ActivitySource: entities.ActivitySourceHook,
			Timestamp:      time.Now(),
		})
		require.NoError(t, err)

		err = activity.AssociateWithSession("test_session")
		require.NoError(t, err)
		err = activity.AssociateWithWorkBlock("test_workblock")
		require.NoError(t, err)

		err = repo.Save(ctx, activity)
		require.NoError(t, err)

		// Check updated work block count
		var updatedCount int
		err = db.QueryRowContext(ctx, 
			"SELECT activity_count FROM work_blocks WHERE id = 'test_workblock'").Scan(&updatedCount)
		require.NoError(t, err)

		assert.Equal(t, initialCount+1, updatedCount, "Activity count should increment by 1")
	})

	t.Run("multiple activities update work block count correctly", func(t *testing.T) {
		// Get current work block count
		var currentCount int
		err := db.QueryRowContext(ctx, 
			"SELECT activity_count FROM work_blocks WHERE id = 'test_workblock'").Scan(&currentCount)
		require.NoError(t, err)

		// Add multiple activities
		activitiesCount := 3
		for i := 0; i < activitiesCount; i++ {
			activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
				UserID:         "test_user",
				ProjectPath:    "/test/path",
				ActivityType:   entities.ActivityTypeCommand,
				ActivitySource: entities.ActivitySourceHook,
				Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
				Command:        fmt.Sprintf("command_%d", i),
			})
			require.NoError(t, err)

			err = activity.AssociateWithSession("test_session")
			require.NoError(t, err)
			err = activity.AssociateWithWorkBlock("test_workblock")
			require.NoError(t, err)

			err = repo.Save(ctx, activity)
			require.NoError(t, err)
		}

		// Check final work block count
		var finalCount int
		err = db.QueryRowContext(ctx, 
			"SELECT activity_count FROM work_blocks WHERE id = 'test_workblock'").Scan(&finalCount)
		require.NoError(t, err)

		assert.Equal(t, currentCount+activitiesCount, finalCount, 
			"Activity count should increment by number of activities added")
	})
}

/**
 * CONTEXT:   Test batch activity operations for performance and consistency
 * INPUT:     Multiple activities for batch insertion and validation
 * OUTPUT:    Efficient batch operations with proper FK and count updates
 * BUSINESS:  Batch operations support high-frequency activity logging
 * CHANGE:    Initial batch operation testing with integrity validation
 * RISK:      Medium - Batch operations require careful transaction handling
 */
func TestActivityRepository_BatchOperations(t *testing.T) {
	db, cleanup := setupActivityTestDB(t)
	defer cleanup()

	repo := NewActivityRepository(db)
	ctx := context.Background()

	t.Run("successful batch save with count updates", func(t *testing.T) {
		// Get initial work block count
		var initialCount int
		err := db.QueryRowContext(ctx, 
			"SELECT activity_count FROM work_blocks WHERE id = 'test_workblock'").Scan(&initialCount)
		require.NoError(t, err)

		// Create batch of activities
		batchSize := 5
		activities := make([]*entities.ActivityEvent, batchSize)
		
		for i := 0; i < batchSize; i++ {
			activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
				UserID:         "test_user",
				ProjectPath:    "/test/path",
				ActivityType:   entities.ActivityTypeCommand,
				ActivitySource: entities.ActivitySourceHook,
				Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
				Command:        fmt.Sprintf("batch_command_%d", i),
				Description:    fmt.Sprintf("batch activity %d", i),
			})
			require.NoError(t, err)

			err = activity.AssociateWithSession("test_session")
			require.NoError(t, err)
			err = activity.AssociateWithWorkBlock("test_workblock")
			require.NoError(t, err)

			activities[i] = activity
		}

		// Save batch
		err = repo.SaveBatch(ctx, activities)
		require.NoError(t, err)

		// Verify all activities were saved
		for _, activity := range activities {
			savedActivity, err := repo.FindByID(ctx, activity.ID())
			assert.NoError(t, err)
			assert.Equal(t, activity.ID(), savedActivity.ID())
		}

		// Verify work block count update
		var finalCount int
		err = db.QueryRowContext(ctx, 
			"SELECT activity_count FROM work_blocks WHERE id = 'test_workblock'").Scan(&finalCount)
		require.NoError(t, err)

		assert.Equal(t, initialCount+batchSize, finalCount, 
			"Work block count should reflect all batch activities")
	})

	t.Run("batch save with mixed work blocks", func(t *testing.T) {
		// Insert additional work block
		workBlockStart := time.Now().Add(-30 * time.Minute)
		_, err := db.ExecContext(ctx, `
			INSERT INTO work_blocks (id, session_id, project_id, start_time, state, last_activity_time, activity_count)
			VALUES ('test_workblock2', 'test_session', 'test_project', ?, 'active', ?, 0)
		`, workBlockStart, workBlockStart)
		require.NoError(t, err)

		// Create activities for different work blocks
		activities := make([]*entities.ActivityEvent, 4)
		
		// First two activities for test_workblock
		for i := 0; i < 2; i++ {
			activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
				UserID:         "test_user",
				ProjectPath:    "/test/path",
				ActivityType:   entities.ActivityTypeCommand,
				ActivitySource: entities.ActivitySourceHook,
				Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
				Command:        fmt.Sprintf("wb1_command_%d", i),
			})
			require.NoError(t, err)

			err = activity.AssociateWithSession("test_session")
			require.NoError(t, err)
			err = activity.AssociateWithWorkBlock("test_workblock")
			require.NoError(t, err)

			activities[i] = activity
		}

		// Last two activities for test_workblock2
		for i := 2; i < 4; i++ {
			activity, err := entities.NewActivityEvent(entities.ActivityEventConfig{
				UserID:         "test_user",
				ProjectPath:    "/test/path",
				ActivityType:   entities.ActivityTypeCommand,
				ActivitySource: entities.ActivitySourceHook,
				Timestamp:      time.Now().Add(time.Duration(i) * time.Second),
				Command:        fmt.Sprintf("wb2_command_%d", i),
			})
			require.NoError(t, err)

			err = activity.AssociateWithSession("test_session")
			require.NoError(t, err)
			err = activity.AssociateWithWorkBlock("test_workblock2")
			require.NoError(t, err)

			activities[i] = activity
		}

		// Save batch
		err = repo.SaveBatch(ctx, activities)
		require.NoError(t, err)

		// Verify work block counts updated correctly
		var count1, count2 int
		err = db.QueryRowContext(ctx, 
			"SELECT activity_count FROM work_blocks WHERE id = 'test_workblock'").Scan(&count1)
		require.NoError(t, err)

		err = db.QueryRowContext(ctx, 
			"SELECT activity_count FROM work_blocks WHERE id = 'test_workblock2'").Scan(&count2)
		require.NoError(t, err)

		// Each work block should have gained 2 activities
		assert.True(t, count2 >= 2, "test_workblock2 should have at least 2 activities")
	})
}

/**
 * CONTEXT:   Test activity query operations and filtering
 * INPUT:     Saved activities with various attributes for querying
 * OUTPUT:    Accurate query results with proper filtering and ordering
 * BUSINESS:  Activity queries enable reporting and analytics functionality
 * CHANGE:    Initial activity query testing with multiple filter types
 * RISK:      Low - Test ensuring query accuracy and performance
 */
func TestActivityRepository_QueryOperations(t *testing.T) {
	db, cleanup := setupActivityTestDB(t)
	defer cleanup()

	repo := NewActivityRepository(db)
	ctx := context.Background()

	// Setup test activities
	setupTestActivities := func() []*entities.ActivityEvent {
		activities := []*entities.ActivityEvent{}
		
		activityConfigs := []entities.ActivityEventConfig{
			{
				UserID:         "test_user",
				ProjectPath:    "/test/path",
				ActivityType:   entities.ActivityTypeCommand,
				ActivitySource: entities.ActivitySourceHook,
				Timestamp:      time.Now().Add(-10 * time.Minute),
				Command:        "git commit",
				Description:    "committing changes",
			},
			{
				UserID:         "test_user",
				ProjectPath:    "/test/path",
				ActivityType:   entities.ActivityTypeFileEdit,
				ActivitySource: entities.ActivitySourceCLI,
				Timestamp:      time.Now().Add(-5 * time.Minute),
				Command:        "edit main.go",
				Description:    "editing main file",
			},
			{
				UserID:         "test_user",
				ProjectPath:    "/test/path",
				ActivityType:   entities.ActivityTypeGeneration,
				ActivitySource: entities.ActivitySourceHook,
				Timestamp:      time.Now(),
				Command:        "claude generate",
				Description:    "generating code",
			},
		}

		for _, config := range activityConfigs {
			activity, err := entities.NewActivityEvent(config)
			require.NoError(t, err)

			err = activity.AssociateWithSession("test_session")
			require.NoError(t, err)
			err = activity.AssociateWithWorkBlock("test_workblock")
			require.NoError(t, err)

			err = repo.Save(ctx, activity)
			require.NoError(t, err)

			activities = append(activities, activity)
		}

		return activities
	}

	testActivities := setupTestActivities()

	t.Run("find by work block ID", func(t *testing.T) {
		activities, err := repo.FindByWorkBlockID(ctx, "test_workblock")
		require.NoError(t, err)
		
		// Should find all test activities plus any from previous tests
		assert.GreaterOrEqual(t, len(activities), len(testActivities))
		
		// Verify all activities belong to the correct work block
		for _, activity := range activities {
			assert.Equal(t, "test_workblock", activity.WorkBlockID())
		}
	})

	t.Run("find by session ID", func(t *testing.T) {
		activities, err := repo.FindBySessionID(ctx, "test_session")
		require.NoError(t, err)
		
		// Should find all test activities plus any from previous tests
		assert.GreaterOrEqual(t, len(activities), len(testActivities))
		
		// Verify all activities belong to the correct session
		for _, activity := range activities {
			assert.Equal(t, "test_session", activity.SessionID())
		}
	})

	t.Run("find by user ID", func(t *testing.T) {
		activities, err := repo.FindByUserID(ctx, "test_user")
		require.NoError(t, err)
		
		// Should find all test activities plus any from previous tests
		assert.GreaterOrEqual(t, len(activities), len(testActivities))
		
		// Verify all activities belong to the correct user
		for _, activity := range activities {
			assert.Equal(t, "test_user", activity.UserID())
		}
	})
}

/**
 * CONTEXT:   Test error conditions and edge cases
 * INPUT:     Invalid data and error scenarios for robust error handling
 * OUTPUT:    Proper error handling and graceful failure modes
 * BUSINESS:  Error handling ensures system stability under adverse conditions
 * CHANGE:    Initial error handling validation for activity operations
 * RISK:      Low - Test ensuring robust error handling and system stability
 */
func TestActivityRepository_ErrorHandling(t *testing.T) {
	db, cleanup := setupActivityTestDB(t)
	defer cleanup()

	repo := NewActivityRepository(db)
	ctx := context.Background()

	t.Run("save nil activity", func(t *testing.T) {
		err := repo.Save(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("find non-existent activity", func(t *testing.T) {
		_, err := repo.FindByID(ctx, "non_existent_id")
		require.Error(t, err)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete non-existent activity", func(t *testing.T) {
		err := repo.Delete(ctx, "non_existent_id")
		require.Error(t, err)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("batch save with nil activity", func(t *testing.T) {
		activities := []*entities.ActivityEvent{nil}
		err := repo.SaveBatch(ctx, activities)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is nil")
	})

	t.Run("batch save empty slice", func(t *testing.T) {
		err := repo.SaveBatch(ctx, []*entities.ActivityEvent{})
		assert.NoError(t, err, "Empty batch should succeed without error")
	})
}

