/**
 * CONTEXT:   Test suite for session repository CRUD operations and business logic
 * INPUT:     Test scenarios covering session lifecycle and database operations
 * OUTPUT:    Validation of session repository functionality and error handling
 * BUSINESS:  Ensure session repository correctly implements 5-hour window business rules
 * CHANGE:    Initial test coverage for SQLite session repository
 * RISK:      Low - Comprehensive test coverage ensures repository correctness
 */

package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionRepositoryCreate(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	repo := NewSessionRepository(db)
	ctx := context.Background()

	// Create test user first
	_, err := db.DB().ExecContext(ctx,
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"test-user", "testuser", time.Now(), time.Now())
	require.NoError(t, err)

	t.Run("Create valid session should succeed", func(t *testing.T) {
		startTime := time.Now().Truncate(time.Second)
		session := &Session{
			ID:                "test-session-1",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour),
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     1,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.Create(ctx, session)
		assert.NoError(t, err)

		// Verify session was created
		retrieved, err := repo.GetByID(ctx, "test-session-1")
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrieved.ID)
		assert.Equal(t, session.UserID, retrieved.UserID)
		assert.Equal(t, session.State, retrieved.State)
		assert.Equal(t, session.DurationHours, retrieved.DurationHours)
		assert.WithinDuration(t, session.StartTime, retrieved.StartTime, time.Second)
		assert.WithinDuration(t, session.EndTime, retrieved.EndTime, time.Second)
	})

	t.Run("Create session with invalid duration should fail", func(t *testing.T) {
		startTime := time.Now()
		session := &Session{
			ID:                "test-session-2",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(3 * time.Hour), // Wrong duration
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     1,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.Create(ctx, session)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session duration must be exactly 5 hours")
	})

	t.Run("Create session with invalid state should fail", func(t *testing.T) {
		startTime := time.Now()
		session := &Session{
			ID:                "test-session-3",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour),
			State:             "invalid-state",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     1,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.Create(ctx, session)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid session state")
	})

	t.Run("Create session with non-existent user should fail", func(t *testing.T) {
		startTime := time.Now()
		session := &Session{
			ID:                "test-session-4",
			UserID:            "non-existent-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour),
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     1,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.Create(ctx, session)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "FOREIGN KEY constraint failed")
	})

	t.Run("Create session with zero activity count should fail", func(t *testing.T) {
		startTime := time.Now()
		session := &Session{
			ID:                "test-session-5",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour),
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     0, // Invalid
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.Create(ctx, session)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "activity count must be at least 1")
	})
}

func TestSessionRepositoryGetByID(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	repo := NewSessionRepository(db)
	ctx := context.Background()

	// Create test user and session
	_, err := db.DB().ExecContext(ctx,
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"test-user", "testuser", time.Now(), time.Now())
	require.NoError(t, err)

	startTime := time.Now().Truncate(time.Second)
	session := &Session{
		ID:                "test-session-get",
		UserID:            "test-user",
		StartTime:         startTime,
		EndTime:           startTime.Add(5 * time.Hour),
		State:             "active",
		FirstActivityTime: startTime,
		LastActivityTime:  startTime,
		ActivityCount:     1,
		DurationHours:     5.0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err = repo.Create(ctx, session)
	require.NoError(t, err)

	t.Run("Get existing session should succeed", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, "test-session-get")
		require.NoError(t, err)
		
		assert.Equal(t, session.ID, retrieved.ID)
		assert.Equal(t, session.UserID, retrieved.UserID)
		assert.Equal(t, session.State, retrieved.State)
		assert.WithinDuration(t, session.StartTime, retrieved.StartTime, time.Second)
		assert.WithinDuration(t, session.EndTime, retrieved.EndTime, time.Second)
	})

	t.Run("Get non-existent session should fail", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, "non-existent-session")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("Get with empty ID should fail", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})
}

func TestSessionRepositoryFindByUserAndTimeRange(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	repo := NewSessionRepository(db)
	ctx := context.Background()

	// Create test user
	_, err := db.DB().ExecContext(ctx,
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"test-user", "testuser", time.Now(), time.Now())
	require.NoError(t, err)

	// Create multiple sessions at different times
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	sessions := []*Session{
		{
			ID:                "session-1",
			UserID:            "test-user",
			StartTime:         baseTime,
			EndTime:           baseTime.Add(5 * time.Hour),
			State:             "finished",
			FirstActivityTime: baseTime,
			LastActivityTime:  baseTime.Add(time.Hour),
			ActivityCount:     10,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                "session-2",
			UserID:            "test-user",
			StartTime:         baseTime.Add(24 * time.Hour),
			EndTime:           baseTime.Add(24*time.Hour + 5*time.Hour),
			State:             "finished",
			FirstActivityTime: baseTime.Add(24 * time.Hour),
			LastActivityTime:  baseTime.Add(24*time.Hour + time.Hour),
			ActivityCount:     15,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                "session-3",
			UserID:            "test-user",
			StartTime:         baseTime.Add(48 * time.Hour),
			EndTime:           baseTime.Add(48*time.Hour + 5*time.Hour),
			State:             "active",
			FirstActivityTime: baseTime.Add(48 * time.Hour),
			LastActivityTime:  baseTime.Add(48*time.Hour + time.Hour),
			ActivityCount:     5,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
	}

	for _, session := range sessions {
		err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	t.Run("Find sessions in time range should return correct sessions", func(t *testing.T) {
		// Query for sessions in first 2 days
		startTime := baseTime.Add(-time.Hour)
		endTime := baseTime.Add(25 * time.Hour)

		found, err := repo.FindByUserAndTimeRange(ctx, "test-user", startTime, endTime)
		require.NoError(t, err)
		
		assert.Len(t, found, 2, "Should find 2 sessions in time range")
		
		// Sessions should be ordered by start_time DESC
		assert.Equal(t, "session-2", found[0].ID)
		assert.Equal(t, "session-1", found[1].ID)
	})

	t.Run("Find sessions with narrow time range", func(t *testing.T) {
		// Query for only the first session
		startTime := baseTime.Add(-time.Hour)
		endTime := baseTime.Add(time.Hour)

		found, err := repo.FindByUserAndTimeRange(ctx, "test-user", startTime, endTime)
		require.NoError(t, err)
		
		assert.Len(t, found, 1)
		assert.Equal(t, "session-1", found[0].ID)
	})

	t.Run("Find sessions with no matches", func(t *testing.T) {
		// Query for time range with no sessions
		startTime := baseTime.Add(-48 * time.Hour)
		endTime := baseTime.Add(-24 * time.Hour)

		found, err := repo.FindByUserAndTimeRange(ctx, "test-user", startTime, endTime)
		require.NoError(t, err)
		
		assert.Len(t, found, 0)
	})

	t.Run("Find with empty user ID should fail", func(t *testing.T) {
		found, err := repo.FindByUserAndTimeRange(ctx, "", baseTime, baseTime.Add(time.Hour))
		assert.Error(t, err)
		assert.Nil(t, found)
		assert.Contains(t, err.Error(), "user ID cannot be empty")
	})
}

func TestSessionRepositoryGetActiveSessionsByUser(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	repo := NewSessionRepository(db)
	ctx := context.Background()

	// Create test user
	_, err := db.DB().ExecContext(ctx,
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"test-user", "testuser", time.Now(), time.Now())
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	// Create sessions with different states and times
	sessions := []*Session{
		{
			ID:                "active-session-1",
			UserID:            "test-user",
			StartTime:         now.Add(-time.Hour),
			EndTime:           now.Add(4 * time.Hour), // Still active
			State:             "active",
			FirstActivityTime: now.Add(-time.Hour),
			LastActivityTime:  now.Add(-30 * time.Minute),
			ActivityCount:     5,
			DurationHours:     5.0,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "expired-session",
			UserID:            "test-user",
			StartTime:         now.Add(-10 * time.Hour),
			EndTime:           now.Add(-5 * time.Hour), // Expired
			State:             "active", // Still marked active but expired by time
			FirstActivityTime: now.Add(-10 * time.Hour),
			LastActivityTime:  now.Add(-6 * time.Hour),
			ActivityCount:     3,
			DurationHours:     5.0,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "finished-session",
			UserID:            "test-user",
			StartTime:         now.Add(-2 * time.Hour),
			EndTime:           now.Add(3 * time.Hour),
			State:             "finished",
			FirstActivityTime: now.Add(-2 * time.Hour),
			LastActivityTime:  now.Add(-time.Hour),
			ActivityCount:     8,
			DurationHours:     5.0,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}

	for _, session := range sessions {
		err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	t.Run("Get active sessions should return only truly active sessions", func(t *testing.T) {
		activeSessions, err := repo.GetActiveSessionsByUser(ctx, "test-user")
		require.NoError(t, err)
		
		assert.Len(t, activeSessions, 1, "Should find only 1 truly active session")
		assert.Equal(t, "active-session-1", activeSessions[0].ID)
		assert.Equal(t, "active", activeSessions[0].State)
	})

	t.Run("Get active sessions with empty user ID should fail", func(t *testing.T) {
		activeSessions, err := repo.GetActiveSessionsByUser(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, activeSessions)
		assert.Contains(t, err.Error(), "user ID cannot be empty")
	})
}

func TestSessionRepositoryUpdate(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	repo := NewSessionRepository(db)
	ctx := context.Background()

	// Create test user and session
	_, err := db.DB().ExecContext(ctx,
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"test-user", "testuser", time.Now(), time.Now())
	require.NoError(t, err)

	startTime := time.Now().Truncate(time.Second)
	session := &Session{
		ID:                "test-session-update",
		UserID:            "test-user",
		StartTime:         startTime,
		EndTime:           startTime.Add(5 * time.Hour),
		State:             "active",
		FirstActivityTime: startTime,
		LastActivityTime:  startTime,
		ActivityCount:     1,
		DurationHours:     5.0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err = repo.Create(ctx, session)
	require.NoError(t, err)

	t.Run("Update session should succeed", func(t *testing.T) {
		// Update session with new activity
		session.LastActivityTime = startTime.Add(time.Hour)
		session.ActivityCount = 10
		session.State = "finished"

		err := repo.Update(ctx, session)
		assert.NoError(t, err)

		// Verify updates
		updated, err := repo.GetByID(ctx, session.ID)
		require.NoError(t, err)
		
		assert.Equal(t, int64(10), updated.ActivityCount)
		assert.Equal(t, "finished", updated.State)
		assert.WithinDuration(t, session.LastActivityTime, updated.LastActivityTime, time.Second)
	})

	t.Run("Update non-existent session should fail", func(t *testing.T) {
		nonExistentSession := &Session{
			ID:                "non-existent",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour),
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     1,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.Update(ctx, nonExistentSession)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found for update")
	})

	t.Run("Update with invalid data should fail", func(t *testing.T) {
		invalidSession := &Session{
			ID:                "test-session-update",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(3 * time.Hour), // Invalid duration
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     1,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.Update(ctx, invalidSession)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session duration must be exactly 5 hours")
	})
}

func TestSessionRepositoryDelete(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	repo := NewSessionRepository(db)
	ctx := context.Background()

	// Create test user and session
	_, err := db.DB().ExecContext(ctx,
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"test-user", "testuser", time.Now(), time.Now())
	require.NoError(t, err)

	startTime := time.Now().Truncate(time.Second)
	session := &Session{
		ID:                "test-session-delete",
		UserID:            "test-user",
		StartTime:         startTime,
		EndTime:           startTime.Add(5 * time.Hour),
		State:             "active",
		FirstActivityTime: startTime,
		LastActivityTime:  startTime,
		ActivityCount:     1,
		DurationHours:     5.0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err = repo.Create(ctx, session)
	require.NoError(t, err)

	t.Run("Delete existing session should succeed", func(t *testing.T) {
		err := repo.Delete(ctx, "test-session-delete")
		assert.NoError(t, err)

		// Verify session was deleted
		deleted, err := repo.GetByID(ctx, "test-session-delete")
		assert.Error(t, err)
		assert.Nil(t, deleted)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("Delete non-existent session should fail", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existent-session")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found for deletion")
	})

	t.Run("Delete with empty ID should fail", func(t *testing.T) {
		err := repo.Delete(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})
}

func TestSessionRepositoryMarkExpiredSessions(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	repo := NewSessionRepository(db)
	ctx := context.Background()

	// Create test user
	_, err := db.DB().ExecContext(ctx,
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"test-user", "testuser", time.Now(), time.Now())
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	// Create sessions with different expiration states
	sessions := []*Session{
		{
			ID:                "expired-session-1",
			UserID:            "test-user",
			StartTime:         now.Add(-10 * time.Hour),
			EndTime:           now.Add(-5 * time.Hour), // Expired 5 hours ago
			State:             "active",
			FirstActivityTime: now.Add(-10 * time.Hour),
			LastActivityTime:  now.Add(-6 * time.Hour),
			ActivityCount:     3,
			DurationHours:     5.0,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "active-session",
			UserID:            "test-user",
			StartTime:         now.Add(-time.Hour),
			EndTime:           now.Add(4 * time.Hour), // Still active
			State:             "active",
			FirstActivityTime: now.Add(-time.Hour),
			LastActivityTime:  now.Add(-30 * time.Minute),
			ActivityCount:     5,
			DurationHours:     5.0,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "finished-session",
			UserID:            "test-user",
			StartTime:         now.Add(-8 * time.Hour),
			EndTime:           now.Add(-3 * time.Hour),
			State:             "finished", // Already finished
			FirstActivityTime: now.Add(-8 * time.Hour),
			LastActivityTime:  now.Add(-4 * time.Hour),
			ActivityCount:     8,
			DurationHours:     5.0,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}

	for _, session := range sessions {
		err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	t.Run("Mark expired sessions should update only expired active sessions", func(t *testing.T) {
		expiredCount, err := repo.MarkExpiredSessions(ctx)
		require.NoError(t, err)
		
		assert.Equal(t, 1, expiredCount, "Should mark 1 session as expired")

		// Verify the expired session was updated
		expiredSession, err := repo.GetByID(ctx, "expired-session-1")
		require.NoError(t, err)
		assert.Equal(t, "expired", expiredSession.State)

		// Verify active session remains active
		activeSession, err := repo.GetByID(ctx, "active-session")
		require.NoError(t, err)
		assert.Equal(t, "active", activeSession.State)

		// Verify finished session remains finished
		finishedSession, err := repo.GetByID(ctx, "finished-session")
		require.NoError(t, err)
		assert.Equal(t, "finished", finishedSession.State)
	})
}

func TestSessionRepositoryGetStats(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()
	
	repo := NewSessionRepository(db)
	ctx := context.Background()

	// Create test user
	_, err := db.DB().ExecContext(ctx,
		"INSERT INTO users (id, username, created_at, updated_at) VALUES (?, ?, ?, ?)",
		"test-user", "testuser", time.Now(), time.Now())
	require.NoError(t, err)

	now := time.Now().Truncate(time.Second)

	// Create sessions with different states
	sessions := []*Session{
		{
			ID:                "active-session",
			UserID:            "test-user",
			StartTime:         now.Add(-time.Hour),
			EndTime:           now.Add(4 * time.Hour),
			State:             "active",
			FirstActivityTime: now.Add(-time.Hour),
			LastActivityTime:  now.Add(-30 * time.Minute),
			ActivityCount:     5,
			DurationHours:     5.0,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "expired-session",
			UserID:            "test-user",
			StartTime:         now.Add(-10 * time.Hour),
			EndTime:           now.Add(-5 * time.Hour),
			State:             "expired",
			FirstActivityTime: now.Add(-10 * time.Hour),
			LastActivityTime:  now.Add(-6 * time.Hour),
			ActivityCount:     8,
			DurationHours:     5.0,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "finished-session",
			UserID:            "test-user",
			StartTime:         now.Add(-8 * time.Hour),
			EndTime:           now.Add(-3 * time.Hour),
			State:             "finished",
			FirstActivityTime: now.Add(-8 * time.Hour),
			LastActivityTime:  now.Add(-4 * time.Hour),
			ActivityCount:     12,
			DurationHours:     5.0,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}

	for _, session := range sessions {
		err := repo.Create(ctx, session)
		require.NoError(t, err)
	}

	t.Run("Get stats should return correct counts", func(t *testing.T) {
		stats, err := repo.GetStats(ctx, "test-user")
		require.NoError(t, err)
		
		assert.Equal(t, 3, stats["total_sessions"])
		assert.Equal(t, 1, stats["active_sessions"])
		assert.Equal(t, 1, stats["expired_sessions"])
		assert.Equal(t, 1, stats["finished_sessions"])
		
		// Average activity count: (5 + 8 + 12) / 3 = 8.33...
		avgActivity := stats["avg_activity_count"].(float64)
		assert.InDelta(t, 8.33, avgActivity, 0.1)
		
		assert.Contains(t, stats, "most_recent_session")
	})

	t.Run("Get stats for all users", func(t *testing.T) {
		stats, err := repo.GetStats(ctx, "")
		require.NoError(t, err)
		
		assert.Equal(t, 3, stats["total_sessions"])
		assert.Contains(t, stats, "avg_activity_count")
	})
}

// Test session validation function
func TestSessionValidation(t *testing.T) {
	t.Run("Valid session should pass validation", func(t *testing.T) {
		startTime := time.Now()
		session := &Session{
			ID:                "valid-session",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour),
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     1,
			DurationHours:     5.0,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := validateSession(session)
		assert.NoError(t, err)
	})

	t.Run("Nil session should fail validation", func(t *testing.T) {
		err := validateSession(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session cannot be nil")
	})

	t.Run("Empty ID should fail validation", func(t *testing.T) {
		startTime := time.Now()
		session := &Session{
			ID:                "", // Empty
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(5 * time.Hour),
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     1,
			DurationHours:     5.0,
		}

		err := validateSession(session)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})

	t.Run("Invalid duration should fail validation", func(t *testing.T) {
		startTime := time.Now()
		session := &Session{
			ID:                "test-session",
			UserID:            "test-user",
			StartTime:         startTime,
			EndTime:           startTime.Add(3 * time.Hour), // Wrong duration
			State:             "active",
			FirstActivityTime: startTime,
			LastActivityTime:  startTime,
			ActivityCount:     1,
			DurationHours:     5.0,
		}

		err := validateSession(session)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session duration must be exactly 5 hours")
	})
}