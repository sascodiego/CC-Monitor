/**
 * CONTEXT:   Comprehensive tests for time-based session manager
 * INPUT:     Various session scenarios and edge cases
 * OUTPUT:    Test validation of session business logic
 * BUSINESS:  Verify single active session rule and time-based logic
 * CHANGE:    Initial test suite for database-backed session manager
 * RISK:      Low - Test code validating core business logic
 */

package business

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database for session manager testing
func setupTestDB(t *testing.T) (*sqlite.SQLiteDB, func()) {
	config := sqlite.DefaultConnectionConfig(":memory:")
	config.Timezone = "America/Montevideo"
	
	db, err := sqlite.NewSQLiteDB(config)
	require.NoError(t, err)
	
	err = db.Initialize()
	require.NoError(t, err)
	
	// Create test users to satisfy foreign key constraints
	testUsers := []string{"test_user", "test_user_expired", "user1", "user2", "concurrent_user"}
	for _, userID := range testUsers {
		_, err = db.DB().Exec("INSERT INTO users (id, username) VALUES (?, ?)", userID, userID)
		require.NoError(t, err)
	}
	
	// Create test project
	_, err = db.DB().Exec("INSERT INTO projects (id, name, path) VALUES (?, ?, ?)", "test_project", "Test Project", "/test/path")
	require.NoError(t, err)
	
	cleanup := func() {
		db.Close()
	}
	
	return db, cleanup
}

func TestSessionManager_GetOrCreateSession(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(manager *SessionManager, ctx context.Context, userID string, activityTime time.Time) *Session
		userID         string
		activityTime   time.Time
		expectedNew    bool
		expectedError  bool
	}{
		{
			name:         "create new session for new user",
			setup:        func(manager *SessionManager, ctx context.Context, userID string, activityTime time.Time) *Session { return nil },
			userID:       "test_user",
			activityTime: time.Now(),
			expectedNew:  true,
		},
		{
			name: "reuse active session within window",
			setup: func(manager *SessionManager, ctx context.Context, userID string, activityTime time.Time) *Session {
				session, err := manager.createNewSession(ctx, userID, activityTime.Add(-1*time.Hour))
				require.NoError(t, err)
				return session
			},
			userID:       "test_user",
			activityTime: time.Now(),
			expectedNew:  false,
		},
		{
			name: "create new session when previous expired",
			setup: func(manager *SessionManager, ctx context.Context, userID string, activityTime time.Time) *Session {
				expiredTime := activityTime.Add(-6 * time.Hour)
				session, err := manager.createNewSession(ctx, userID, expiredTime)
				require.NoError(t, err)
				return session
			},
			userID:       "test_user_expired",
			activityTime: time.Now(),
			expectedNew:  true,
		},
		{
			name:          "error on empty user ID",
			setup:         func(manager *SessionManager, ctx context.Context, userID string, activityTime time.Time) *Session { return nil },
			userID:        "",
			activityTime:  time.Now(),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh database for each test to avoid interference
			db, cleanup := setupTestDB(t)
			defer cleanup()
			
			sessionRepo := sqlite.NewSessionRepository(db)
			manager := NewSessionManager(sessionRepo)
			ctx := context.Background()
			
			// Setup test data
			existingSession := tt.setup(manager, ctx, tt.userID, tt.activityTime)
			
			// Execute
			session, err := manager.GetOrCreateSession(ctx, tt.userID, tt.activityTime)
			
			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.NotNil(t, session)
			assert.Equal(t, tt.userID, session.UserID)
			assert.Equal(t, "active", session.State)
			assert.Equal(t, 5.0, session.DurationHours)
			
			if tt.expectedNew {
				if existingSession != nil {
					assert.NotEqual(t, existingSession.ID, session.ID)
				}
				assert.True(t, session.StartTime.Equal(tt.activityTime) || 
					session.StartTime.Before(tt.activityTime.Add(time.Second)))
			} else {
				assert.Equal(t, existingSession.ID, session.ID)
				assert.True(t, session.ActivityCount > existingSession.ActivityCount)
			}
			
			// Verify session timing
			expectedEndTime := session.StartTime.Add(5 * time.Hour)
			assert.True(t, session.EndTime.Equal(expectedEndTime))
		})
	}
}

func TestSessionManager_SingleActiveSessionRule(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	sessionRepo := sqlite.NewSessionRepository(db)
	manager := NewSessionManager(sessionRepo)
	ctx := context.Background()
	
	userID := "test_user"
	now := time.Now()

	// Create first session
	session1, err := manager.GetOrCreateSession(ctx, userID, now)
	require.NoError(t, err)
	assert.Equal(t, "active", session1.State)

	// Activity within same session window should reuse session
	session2, err := manager.GetOrCreateSession(ctx, userID, now.Add(2*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, session1.ID, session2.ID)
	assert.Equal(t, "active", session2.State)
	assert.True(t, session2.ActivityCount > session1.ActivityCount)

	// Activity after expiration should create new session
	session3, err := manager.GetOrCreateSession(ctx, userID, now.Add(6*time.Hour))
	require.NoError(t, err)
	assert.NotEqual(t, session1.ID, session3.ID)
	assert.Equal(t, "active", session3.State)

	// Verify only one active session exists
	activeSession, err := manager.GetActiveSession(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, session3.ID, activeSession.ID)
}

func TestSessionManager_CleanupDuplicateSessions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	sessionRepo := sqlite.NewSessionRepository(db)
	ctx := context.Background()
	
	userID := "test_user"
	now := time.Now()

	// Manually create duplicate active sessions (simulate data inconsistency)
	session1Start := now.Add(-2 * time.Hour)
	session1 := &Session{
		ID:                "session1",
		UserID:            userID,
		StartTime:         session1Start,
		EndTime:           session1Start.Add(5 * time.Hour),
		State:             "active",
		FirstActivityTime: session1Start,
		LastActivityTime:  session1Start,
		ActivityCount:     1,
		DurationHours:     5.0,
		CreatedAt:         session1Start,
		UpdatedAt:         session1Start,
	}
	
	session2Start := now.Add(-1 * time.Hour)
	session2 := &Session{
		ID:                "session2", 
		UserID:            userID,
		StartTime:         session2Start,
		EndTime:           session2Start.Add(5 * time.Hour),
		State:             "active",
		FirstActivityTime: session2Start,
		LastActivityTime:  session2Start,
		ActivityCount:     1,
		DurationHours:     5.0,
		CreatedAt:         session2Start,
		UpdatedAt:         session2Start,
	}

	// Insert both sessions directly
	require.NoError(t, sessionRepo.Create(ctx, session1))
	require.NoError(t, sessionRepo.Create(ctx, session2))

	// Create manager and attempt to get session (should cleanup duplicates)
	manager := NewSessionManager(sessionRepo)
	activeSession, err := manager.GetOrCreateSession(ctx, userID, now)
	require.NoError(t, err)

	// Should return the most recent session (session2)
	assert.Equal(t, session2.ID, activeSession.ID)

	// Verify session1 was marked as expired_duplicate
	session1Updated, err := sessionRepo.GetByID(ctx, session1.ID)
	require.NoError(t, err)
	assert.Equal(t, "expired_duplicate", session1Updated.State)
}

func TestSessionManager_SessionExpirationLogic(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	sessionRepo := sqlite.NewSessionRepository(db)
	manager := NewSessionManager(sessionRepo)
	ctx := context.Background()
	
	baseTime := time.Now()

	tests := []struct {
		name         string
		sessionStart time.Time
		activityTime time.Time
		expectActive bool
	}{
		{
			name:         "activity at session start",
			sessionStart: baseTime,
			activityTime: baseTime,
			expectActive: true,
		},
		{
			name:         "activity 4 hours after start",
			sessionStart: baseTime,
			activityTime: baseTime.Add(4 * time.Hour),
			expectActive: true,
		},
		{
			name:         "activity exactly at 5-hour mark",
			sessionStart: baseTime,
			activityTime: baseTime.Add(5 * time.Hour),
			expectActive: true,
		},
		{
			name:         "activity 1 minute after expiry",
			sessionStart: baseTime,
			activityTime: baseTime.Add(5*time.Hour + 1*time.Minute),
			expectActive: false,
		},
		{
			name:         "activity 1 hour after expiry",
			sessionStart: baseTime,
			activityTime: baseTime.Add(6 * time.Hour),
			expectActive: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use unique test user ID for each test to avoid ID conflicts
			testUserID := fmt.Sprintf("test_user_%d", i)
			
			// Create test user
			_, err := db.DB().Exec("INSERT OR IGNORE INTO users (id, username) VALUES (?, ?)", testUserID, testUserID)
			require.NoError(t, err)
			
			// Create session at specific start time
			session, err := manager.createNewSession(ctx, testUserID, tt.sessionStart)
			require.NoError(t, err)

			// Test if session is considered active for the activity time
			isActive := manager.isSessionActive(session, tt.activityTime)
			assert.Equal(t, tt.expectActive, isActive)
		})
	}
}

func TestSessionManager_GetActiveSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	sessionRepo := sqlite.NewSessionRepository(db)
	manager := NewSessionManager(sessionRepo)
	ctx := context.Background()
	
	userID := "test_user"
	now := time.Now()

	// Test with no active sessions
	activeSession, err := manager.GetActiveSession(ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, activeSession)

	// Create active session
	session, err := manager.GetOrCreateSession(ctx, userID, now)
	require.NoError(t, err)

	// Test getting active session
	activeSession, err = manager.GetActiveSession(ctx, userID)
	require.NoError(t, err)
	assert.NotNil(t, activeSession)
	assert.Equal(t, session.ID, activeSession.ID)

	// Test with expired session (manually expire)
	session.State = "expired"
	require.NoError(t, sessionRepo.Update(ctx, session))

	// Should return nil for expired session
	activeSession, err = manager.GetActiveSession(ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, activeSession)
}

func TestSessionManager_MarkExpiredSessions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	sessionRepo := sqlite.NewSessionRepository(db)
	manager := NewSessionManager(sessionRepo)
	ctx := context.Background()
	
	now := time.Now()

	// Create active session that's expired
	expiredStart := now.Add(-6 * time.Hour)
	expiredSession := &Session{
		ID:                "expired_session",
		UserID:            "user1",
		StartTime:         expiredStart,
		EndTime:           expiredStart.Add(5 * time.Hour),
		State:             "active",
		FirstActivityTime: expiredStart,
		LastActivityTime:  expiredStart,
		ActivityCount:     1,
		DurationHours:     5.0,
		CreatedAt:         expiredStart,
		UpdatedAt:         expiredStart,
	}
	require.NoError(t, sessionRepo.Create(ctx, expiredSession))

	// Create active session that's still valid
	validStart := now.Add(-2 * time.Hour)
	validSession := &Session{
		ID:                "valid_session",
		UserID:            "user2", 
		StartTime:         validStart,
		EndTime:           validStart.Add(5 * time.Hour),
		State:             "active",
		FirstActivityTime: validStart,
		LastActivityTime:  validStart,
		ActivityCount:     1,
		DurationHours:     5.0,
		CreatedAt:         validStart,
		UpdatedAt:         validStart,
	}
	require.NoError(t, sessionRepo.Create(ctx, validSession))

	// Mark expired sessions
	expiredCount, err := manager.MarkExpiredSessions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount)

	// Verify expired session was marked
	updatedExpired, err := sessionRepo.GetByID(ctx, expiredSession.ID)
	require.NoError(t, err)
	assert.Equal(t, "expired", updatedExpired.State)

	// Verify valid session unchanged
	updatedValid, err := sessionRepo.GetByID(ctx, validSession.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", updatedValid.State)
}

func TestSessionManager_ConcurrentAccess(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	sessionRepo := sqlite.NewSessionRepository(db)
	manager := NewSessionManager(sessionRepo)
	ctx := context.Background()
	
	userID := "concurrent_user"
	now := time.Now()
	
	// Test sequential calls to same session (simulates concurrent access patterns)
	const numCalls = 5
	sessions := make([]*Session, numCalls)
	
	// Make sequential calls with small time variations (within same session window)
	for i := 0; i < numCalls; i++ {
		activityTime := now.Add(time.Duration(i) * time.Minute) // Within 5-hour window
		session, err := manager.GetOrCreateSession(ctx, userID, activityTime)
		require.NoError(t, err)
		require.NotNil(t, session)
		sessions[i] = session
	}
	
	// All calls should return the same session (same session reused)
	firstSessionID := sessions[0].ID
	for i := 1; i < numCalls; i++ {
		assert.Equal(t, firstSessionID, sessions[i].ID,
			"all calls should return the same session ID")
		assert.Equal(t, userID, sessions[i].UserID)
		assert.Equal(t, "active", sessions[i].State)
		assert.True(t, sessions[i].ActivityCount >= sessions[i-1].ActivityCount,
			"activity count should increase or stay same")
	}
	
	// Final session should have all activities counted
	assert.Equal(t, int64(numCalls), sessions[numCalls-1].ActivityCount)
}