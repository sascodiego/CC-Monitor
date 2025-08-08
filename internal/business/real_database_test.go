/**
 * CONTEXT:   Test new SessionManager with real migrated database from CHECKPOINT 1
 * INPUT:     Actual production data migrated to SQLite in test_migration.db
 * OUTPUT:    Verification that new session logic works with real data
 * BUSINESS:  Validate CHECKPOINT 2 session refactoring works with production data
 * CHANGE:    Integration test using real migrated data from CHECKPOINT 1
 * RISK:      Low - Test validates production data compatibility
 */

package business

import (
	"context"
	"testing"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionManager_WithRealMigratedDatabase(t *testing.T) {
	// Use the real migrated database from CHECKPOINT 1
	dbPath := "/mnt/c/src/CC-Monitor/test_migration.db"
	
	// Create connection to existing database
	config := sqlite.DefaultConnectionConfig(dbPath)
	config.Timezone = "America/Montevideo"
	
	db, err := sqlite.NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()
	
	// Create session repository and manager
	sessionRepo := sqlite.NewSessionRepository(db)
	sessionManager := NewSessionManager(sessionRepo)
	
	ctx := context.Background()
	
	// Test 1: Check if we can query existing sessions
	t.Run("QueryExistingSessions", func(t *testing.T) {
		// Get session stats
		stats, err := sessionRepo.GetStats(ctx, "")
		require.NoError(t, err)
		
		totalSessions := stats["total_sessions"].(int)
		t.Logf("Found %d sessions in migrated database", totalSessions)
		
		// Should have at least some sessions (from CHECKPOINT 1 report: 2 sessions)
		assert.GreaterOrEqual(t, totalSessions, 1, "should have migrated sessions")
		
		if activeSessions, ok := stats["active_sessions"].(int); ok {
			t.Logf("Active sessions: %d", activeSessions)
		}
	})
	
	// Test 2: Check GetActiveSession with existing data
	t.Run("GetActiveSessionWithExistingData", func(t *testing.T) {
		// Try to get active sessions for a user (may or may not exist)
		activeSession, err := sessionManager.GetActiveSession(ctx, "claude_user")
		require.NoError(t, err)
		
		if activeSession != nil {
			t.Logf("Found active session: %s (start: %s, activities: %d)",
				activeSession.ID,
				activeSession.StartTime.Format("2006-01-02 15:04:05"),
				activeSession.ActivityCount)
			
			assert.Equal(t, "claude_user", activeSession.UserID)
			assert.Equal(t, 5.0, activeSession.DurationHours)
		} else {
			t.Logf("No active session found for claude_user (expected if sessions are expired)")
		}
	})
	
	// Test 3: Create new session in migrated database
	t.Run("CreateNewSessionInMigratedDatabase", func(t *testing.T) {
		// Create new user for testing
		testUserID := "test_checkpoint2_user"
		_, err := db.DB().ExecContext(ctx, "INSERT OR IGNORE INTO users (id, username) VALUES (?, ?)", testUserID, testUserID)
		require.NoError(t, err)
		
		// Create new session using SessionManager
		now := time.Now()
		session, err := sessionManager.GetOrCreateSession(ctx, testUserID, now)
		require.NoError(t, err)
		require.NotNil(t, session)
		
		// Verify session properties
		assert.Equal(t, testUserID, session.UserID)
		assert.Equal(t, "active", session.State)
		assert.Equal(t, 5.0, session.DurationHours)
		assert.Equal(t, int64(1), session.ActivityCount)
		
		// Verify session timing
		expectedEndTime := session.StartTime.Add(5 * time.Hour)
		assert.True(t, session.EndTime.Equal(expectedEndTime))
		
		t.Logf("Created new session: %s (start: %s, end: %s)",
			session.ID,
			session.StartTime.Format("2006-01-02 15:04:05"),
			session.EndTime.Format("2006-01-02 15:04:05"))
	})
	
	// Test 4: Session reuse logic with new activity
	t.Run("SessionReuseInMigratedDatabase", func(t *testing.T) {
		testUserID := "test_reuse_user"
		_, err := db.DB().ExecContext(ctx, "INSERT OR IGNORE INTO users (id, username) VALUES (?, ?)", testUserID, testUserID)
		require.NoError(t, err)
		
		now := time.Now()
		
		// Create initial session
		session1, err := sessionManager.GetOrCreateSession(ctx, testUserID, now)
		require.NoError(t, err)
		
		// Create activity 30 minutes later (should reuse same session)
		session2, err := sessionManager.GetOrCreateSession(ctx, testUserID, now.Add(30*time.Minute))
		require.NoError(t, err)
		
		// Should be the same session
		assert.Equal(t, session1.ID, session2.ID)
		assert.Equal(t, int64(2), session2.ActivityCount)
		
		t.Logf("Session reuse verified: session %s reused with %d activities",
			session2.ID, session2.ActivityCount)
	})
	
	// Test 5: Mark expired sessions in migrated database
	t.Run("MarkExpiredSessionsInMigratedDatabase", func(t *testing.T) {
		expiredCount, err := sessionManager.MarkExpiredSessions(ctx)
		require.NoError(t, err)
		
		t.Logf("Marked %d sessions as expired in migrated database", expiredCount)
		
		// Should be non-negative
		assert.GreaterOrEqual(t, expiredCount, 0)
	})
}

func TestServerIntegration_WithRealMigratedDatabase(t *testing.T) {
	// Test server integration with real migrated database
	dbPath := "/mnt/c/src/CC-Monitor/test_migration.db"
	
	// Create server integration
	integration, err := NewServerSessionIntegration(dbPath)
	require.NoError(t, err)
	defer integration.Close()
	
	ctx := context.Background()
	
	// Test processing activity with real database
	t.Run("ProcessActivityWithMigratedDatabase", func(t *testing.T) {
		// Create test user
		testUserID := "integration_test_user"
		err := integration.ensureUserExists(ctx, testUserID)
		require.NoError(t, err)
		
		// Process activity event
		event := &ActivityEvent{
			UserID:      testUserID,
			ProjectName: "test_project_checkpoint2",
			ProjectPath: "/test/checkpoint2",
			Timestamp:   time.Now(),
			Command:     "test command for checkpoint2",
			Description: "testing new session manager with real database",
		}
		
		err = integration.ProcessActivityEvent(ctx, event)
		require.NoError(t, err)
		
		// Verify session was created
		session, err := integration.sessionManager.GetActiveSession(ctx, testUserID)
		require.NoError(t, err)
		require.NotNil(t, session)
		
		assert.Equal(t, testUserID, session.UserID)
		assert.Equal(t, "active", session.State)
		
		t.Logf("Successfully processed activity with real database: session %s created", session.ID)
	})
}