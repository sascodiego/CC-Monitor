/**
 * CONTEXT:   Unit tests for SessionManager business logic and 5-hour session rules
 * INPUT:     Test scenarios covering session creation, expiration, and edge cases
 * OUTPUT:    Comprehensive test coverage validating session management behavior
 * BUSINESS:  Verify 5-hour session windows and proper session lifecycle management
 * CHANGE:    Initial test implementation for session management use case.
 * RISK:      Low - Test code with no side effects on production system
 */

package usecases

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/claude-monitor/system/internal/usecases/repositories"
)

// MockSessionRepository for testing
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Save(ctx context.Context, session *entities.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByID(ctx context.Context, sessionID string) (*entities.Session, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).(*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) Update(ctx context.Context, session *entities.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByUserID(ctx context.Context, userID string) ([]*entities.Session, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) FindActiveSession(ctx context.Context, userID string) (*entities.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) FindByFilter(ctx context.Context, filter repositories.SessionFilter) ([]*entities.Session, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) FindWithSort(ctx context.Context, filter repositories.SessionFilter, sortBy repositories.SessionSortBy, order repositories.SessionSortOrder) ([]*entities.Session, error) {
	args := m.Called(ctx, filter, sortBy, order)
	return args.Get(0).([]*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) FindExpiredSessions(ctx context.Context, beforeTime time.Time) ([]*entities.Session, error) {
	args := m.Called(ctx, beforeTime)
	return args.Get(0).([]*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) FindSessionsInTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*entities.Session, error) {
	args := m.Called(ctx, userID, start, end)
	return args.Get(0).([]*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) FindRecentSessions(ctx context.Context, userID string, limit int) ([]*entities.Session, error) {
	args := m.Called(ctx, userID, limit)
	return args.Get(0).([]*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) CountSessionsByUser(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSessionRepository) CountSessionsByTimeRange(ctx context.Context, start, end time.Time) (int64, error) {
	args := m.Called(ctx, start, end)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSessionRepository) GetSessionStatistics(ctx context.Context, userID string, start, end time.Time) (*repositories.SessionStatistics, error) {
	args := m.Called(ctx, userID, start, end)
	return args.Get(0).(*repositories.SessionStatistics), args.Error(1)
}

func (m *MockSessionRepository) SaveBatch(ctx context.Context, sessions []*entities.Session) error {
	args := m.Called(ctx, sessions)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteExpired(ctx context.Context, beforeTime time.Time) (int64, error) {
	args := m.Called(ctx, beforeTime)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSessionRepository) WithTransaction(ctx context.Context, fn func(repo repositories.SessionRepository) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

// Test setup helper
func setupSessionManagerTest() (*SessionManager, *MockSessionRepository, *log.Logger) {
	mockRepo := new(MockSessionRepository)
	logger := log.New(os.Stdout, "TEST: ", log.LstdFlags)
	sessionManager := NewSessionManager(mockRepo, logger)
	return sessionManager, mockRepo, logger
}

/**
 * CONTEXT:   Test GetOrCreateSession with no existing session
 * INPUT:     User ID and timestamp for new session creation
 * OUTPUT:    New session should be created and saved to repository
 * BUSINESS:  First activity should create new 5-hour session window
 * CHANGE:    Initial test for new session creation scenario.
 * RISK:      Low - Testing session creation with mocked dependencies
 */
func TestSessionManager_GetOrCreateSession_NewSession(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		timestamp time.Time
		wantErr   bool
	}{
		{
			name:      "create new session for valid user",
			userID:    "user123",
			timestamp: time.Now(),
			wantErr:   false,
		},
		{
			name:      "empty user ID should return error",
			userID:    "",
			timestamp: time.Now(),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionManager, mockRepo, _ := setupSessionManagerTest()
			ctx := context.Background()

			if !tt.wantErr {
				// Mock: no active session found
				mockRepo.On("FindActiveSession", ctx, tt.userID).Return((*entities.Session)(nil), repositories.ErrSessionNotFound)
				
				// Mock: save new session
				mockRepo.On("Save", ctx, mock.AnythingOfType("*entities.Session")).Return(nil)
			}

			session, err := sessionManager.GetOrCreateSession(ctx, tt.userID, tt.timestamp)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, session)
			} else {
				require.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, tt.userID, session.UserID)
				assert.Equal(t, tt.timestamp.Truncate(time.Second), session.StartTime.Truncate(time.Second))
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

/**
 * CONTEXT:   Test GetOrCreateSession with existing active session
 * INPUT:     User ID and timestamp with existing non-expired session
 * OUTPUT:    Should return existing session and update activity
 * BUSINESS:  Activities within 5-hour window should use same session
 * CHANGE:    Initial test for existing session reuse scenario.
 * RISK:      Low - Testing session reuse with mocked dependencies
 */
func TestSessionManager_GetOrCreateSession_ExistingSession(t *testing.T) {
	sessionManager, mockRepo, _ := setupSessionManagerTest()
	ctx := context.Background()
	userID := "user123"
	startTime := time.Now().Add(-2 * time.Hour) // 2 hours ago
	currentTime := time.Now()

	// Create existing session (not expired)
	existingSession, _ := entities.NewSession(userID, startTime)

	// Mock: return existing active session
	mockRepo.On("FindActiveSession", ctx, userID).Return(existingSession, nil)
	
	// Mock: find session by ID for activity update
	mockRepo.On("FindByID", ctx, existingSession.ID).Return(existingSession, nil)
	
	// Mock: update session with new activity
	mockRepo.On("Update", ctx, existingSession).Return(nil)

	session, err := sessionManager.GetOrCreateSession(ctx, userID, currentTime)

	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, existingSession.ID, session.ID)
	assert.Equal(t, existingSession.UserID, session.UserID)

	mockRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test GetOrCreateSession with expired session
 * INPUT:     User ID and timestamp with existing expired session (>5 hours old)
 * OUTPUT:    Should close expired session and create new session
 * BUSINESS:  Sessions expire after exactly 5 hours, new session starts after expiration
 * CHANGE:    Initial test for session expiration and renewal scenario.
 * RISK:      Low - Testing session expiration logic with mocked dependencies
 */
func TestSessionManager_GetOrCreateSession_ExpiredSession(t *testing.T) {
	sessionManager, mockRepo, _ := setupSessionManagerTest()
	ctx := context.Background()
	userID := "user123"
	expiredStartTime := time.Now().Add(-6 * time.Hour) // 6 hours ago (expired)
	currentTime := time.Now()

	// Create expired session
	expiredSession, _ := entities.NewSession(userID, expiredStartTime)

	// Mock: return expired session
	mockRepo.On("FindActiveSession", ctx, userID).Return(expiredSession, nil)
	
	// Mock: update expired session (close it)
	mockRepo.On("Update", ctx, expiredSession).Return(nil)
	
	// Mock: save new session
	mockRepo.On("Save", ctx, mock.AnythingOfType("*entities.Session")).Return(nil)

	session, err := sessionManager.GetOrCreateSession(ctx, userID, currentTime)

	require.NoError(t, err)
	require.NotNil(t, session)
	assert.NotEqual(t, expiredSession.ID, session.ID) // Should be different session
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, currentTime.Truncate(time.Second), session.StartTime.Truncate(time.Second))

	mockRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test IsSessionExpired with various time scenarios
 * INPUT:     Session entities and timestamps for expiration checking
 * OUTPUT:    Boolean indicating correct expiration status based on 5-hour rule
 * BUSINESS:  Sessions expire exactly 5 hours after start time
 * CHANGE:    Initial test for session expiration logic.
 * RISK:      Low - Testing time comparison logic
 */
func TestSessionManager_IsSessionExpired(t *testing.T) {
	sessionManager, _, _ := setupSessionManagerTest()
	baseTime := time.Now()

	tests := []struct {
		name          string
		session       *entities.Session
		checkTime     time.Time
		expectExpired bool
	}{
		{
			name:          "nil session should be expired",
			session:       nil,
			checkTime:     baseTime,
			expectExpired: true,
		},
		{
			name: "session within 5 hours should not be expired",
			session: func() *entities.Session {
				s, _ := entities.NewSession("user123", baseTime.Add(-4*time.Hour))
				return s
			}(),
			checkTime:     baseTime,
			expectExpired: false,
		},
		{
			name: "session exactly at 5 hours should not be expired",
			session: func() *entities.Session {
				s, _ := entities.NewSession("user123", baseTime.Add(-5*time.Hour))
				return s
			}(),
			checkTime:     baseTime,
			expectExpired: false,
		},
		{
			name: "session over 5 hours should be expired",
			session: func() *entities.Session {
				s, _ := entities.NewSession("user123", baseTime.Add(-5*time.Hour-1*time.Second))
				return s
			}(),
			checkTime:     baseTime,
			expectExpired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expired := sessionManager.IsSessionExpired(tt.session, tt.checkTime)
			assert.Equal(t, tt.expectExpired, expired)
		})
	}
}

/**
 * CONTEXT:   Test CloseExpiredSessions batch operation
 * INPUT:     Current time to determine which sessions should be closed
 * OUTPUT:    Number of closed sessions and proper repository interactions
 * BUSINESS:  Cleanup expired sessions to maintain accurate system state
 * CHANGE:    Initial test for batch session cleanup operation.
 * RISK:      Low - Testing batch operations with mocked dependencies
 */
func TestSessionManager_CloseExpiredSessions(t *testing.T) {
	sessionManager, mockRepo, _ := setupSessionManagerTest()
	ctx := context.Background()
	currentTime := time.Now()

	// Create expired sessions
	expiredSession1, _ := entities.NewSession("user1", currentTime.Add(-6*time.Hour))
	expiredSession2, _ := entities.NewSession("user2", currentTime.Add(-7*time.Hour))
	expiredSessions := []*entities.Session{expiredSession1, expiredSession2}

	// Mock: find expired sessions
	expiredBefore := currentTime.Add(-5 * time.Hour)
	mockRepo.On("FindExpiredSessions", ctx, expiredBefore).Return(expiredSessions, nil)
	
	// Mock: update each expired session
	mockRepo.On("Update", ctx, expiredSession1).Return(nil)
	mockRepo.On("Update", ctx, expiredSession2).Return(nil)

	closedCount, err := sessionManager.CloseExpiredSessions(ctx, currentTime)

	require.NoError(t, err)
	assert.Equal(t, int64(2), closedCount)

	mockRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test concurrent session creation scenario
 * INPUT:     Multiple goroutines attempting to create sessions simultaneously
 * OUTPUT:    Thread-safe session creation without race conditions
 * BUSINESS:  System should handle concurrent user activities safely
 * CHANGE:    Initial test for concurrent session access.
 * RISK:      Medium - Testing concurrency scenarios for thread safety
 */
func TestSessionManager_ConcurrentAccess(t *testing.T) {
	sessionManager, mockRepo, _ := setupSessionManagerTest()
	ctx := context.Background()
	userID := "user123"
	timestamp := time.Now()

	// Configure mock for concurrent access
	mockRepo.On("FindActiveSession", ctx, userID).Return((*entities.Session)(nil), repositories.ErrSessionNotFound)
	mockRepo.On("Save", ctx, mock.AnythingOfType("*entities.Session")).Return(nil)

	// Run concurrent session creation
	const numGoroutines = 10
	results := make(chan *entities.Session, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			session, err := sessionManager.GetOrCreateSession(ctx, userID, timestamp)
			if err != nil {
				errors <- err
				return
			}
			results <- session
		}()
	}

	// Collect results
	var sessions []*entities.Session
	var errs []error

	for i := 0; i < numGoroutines; i++ {
		select {
		case session := <-results:
			sessions = append(sessions, session)
		case err := <-errors:
			errs = append(errs, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out waiting for goroutines")
		}
	}

	// At least one session should be created successfully
	assert.True(t, len(sessions) > 0, "At least one session should be created")
	
	// All successful sessions should have same user ID
	for _, session := range sessions {
		assert.Equal(t, userID, session.UserID)
	}
}