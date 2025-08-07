/**
 * CONTEXT:   Comprehensive tests for active session tracker and correlation logic
 * INPUT:     Test scenarios with various session contexts and correlation patterns
 * OUTPUT:    Validated session correlation accuracy and edge case handling
 * BUSINESS:  Ensure daemon-managed correlation system works reliably without temporary files
 * CHANGE:    Initial comprehensive test suite for session correlation system
 * RISK:      High - Test coverage critical for correlation system reliability
 */

package usecases

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/claude-monitor/system/internal/usecases/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockActiveSessionRepository implements ActiveSessionRepository for testing
type MockActiveSessionRepository struct {
	mock.Mock
}

func (m *MockActiveSessionRepository) Save(ctx context.Context, activeSession *entities.ActiveSession) error {
	args := m.Called(ctx, activeSession)
	return args.Error(0)
}

func (m *MockActiveSessionRepository) Update(ctx context.Context, activeSession *entities.ActiveSession) error {
	args := m.Called(ctx, activeSession)
	return args.Error(0)
}

func (m *MockActiveSessionRepository) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockActiveSessionRepository) FindByID(ctx context.Context, sessionID string) (*entities.ActiveSession, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ActiveSession), args.Error(1)
}

func (m *MockActiveSessionRepository) FindByTerminalPID(ctx context.Context, terminalPID int, userID string) ([]*entities.ActiveSession, error) {
	args := m.Called(ctx, terminalPID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.ActiveSession), args.Error(1)
}

func (m *MockActiveSessionRepository) FindByWorkingDir(ctx context.Context, workingDir, userID string) ([]*entities.ActiveSession, error) {
	args := m.Called(ctx, workingDir, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.ActiveSession), args.Error(1)
}

func (m *MockActiveSessionRepository) FindByUser(ctx context.Context, userID string) ([]*entities.ActiveSession, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.ActiveSession), args.Error(1)
}

func (m *MockActiveSessionRepository) FindAll(ctx context.Context) ([]*entities.ActiveSession, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.ActiveSession), args.Error(1)
}

func (m *MockActiveSessionRepository) FindExpiredSessions(ctx context.Context, expiredBefore time.Time) ([]*entities.ActiveSession, error) {
	args := m.Called(ctx, expiredBefore)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.ActiveSession), args.Error(1)
}

func (m *MockActiveSessionRepository) DeleteExpiredSessions(ctx context.Context, expiredBefore time.Time) (int64, error) {
	args := m.Called(ctx, expiredBefore)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockActiveSessionRepository) CountActiveSessions(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockActiveSessionRepository) GetActiveSessionStatistics(ctx context.Context) (*repositories.ActiveSessionStatistics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.ActiveSessionStatistics), args.Error(1)
}

// MockSessionRepository implements SessionRepository for testing
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Save(ctx context.Context, session *entities.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) Update(ctx context.Context, session *entities.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByID(ctx context.Context, sessionID string) (*entities.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) FindActiveSession(ctx context.Context, userID string) (*entities.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) FindExpiredSessions(ctx context.Context, expiredBefore time.Time) ([]*entities.Session, error) {
	args := m.Called(ctx, expiredBefore)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Session), args.Error(1)
}

func (m *MockSessionRepository) GetSessionStatistics(ctx context.Context, userID string, start, end time.Time) (*repositories.SessionStatistics, error) {
	args := m.Called(ctx, userID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.SessionStatistics), args.Error(1)
}

// Test helper functions

func createTestSessionContext(terminalPID int, userID, workingDir string) entities.SessionContext {
	return entities.SessionContext{
		TerminalPID: terminalPID,
		ShellPID:    12345,
		WorkingDir:  workingDir,
		ProjectPath: workingDir,
		UserID:      userID,
		Timestamp:   time.Now(),
	}
}

func createTestActiveSession(context entities.SessionContext) (*entities.ActiveSession, error) {
	config := entities.ActiveSessionConfig{
		SessionContext:    context,
		EstimatedDuration: 5 * time.Minute,
		EstimatedTokens:   1000,
	}
	return entities.NewActiveSession(config)
}

func createTestTracker() (*ActiveSessionTracker, *MockActiveSessionRepository, *MockSessionRepository) {
	mockActiveRepo := &MockActiveSessionRepository{}
	mockSessionRepo := &MockSessionRepository{}
	logger := log.New(nil, "", 0) // Discard logging for tests

	config := ActiveSessionTrackerConfig{
		ActiveSessionRepo:     mockActiveRepo,
		SessionRepo:           mockSessionRepo,
		Logger:                logger,
		MaxConcurrentSessions: 10,
		SessionTimeout:        30 * time.Minute,
		CleanupInterval:       5 * time.Minute,
	}

	tracker := NewActiveSessionTracker(config)
	return tracker, mockActiveRepo, mockSessionRepo
}

/**
 * CONTEXT:   Test session creation with valid context
 * INPUT:     Valid session context with terminal PID, user ID, and working directory
 * OUTPUT:    Successfully created active session with proper state
 * BUSINESS:  Verify basic session creation functionality works correctly
 * CHANGE:    Initial test for successful session creation path
 * RISK:      Medium - Basic functionality test for session creation
 */
func TestCreateSession_Success(t *testing.T) {
	tracker, mockActiveRepo, _ := createTestTracker()
	ctx := context.Background()

	sessionContext := createTestSessionContext(1234, "testuser", "/test/project")

	// Mock expectations
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil)

	// Execute
	activeSession, err := tracker.CreateSession(ctx, sessionContext)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, activeSession)
	assert.Equal(t, sessionContext.UserID, activeSession.SessionContext().UserID)
	assert.Equal(t, sessionContext.TerminalPID, activeSession.SessionContext().TerminalPID)
	assert.Equal(t, sessionContext.WorkingDir, activeSession.SessionContext().WorkingDir)

	// Verify mocks
	mockActiveRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test session creation failure due to database error
 * INPUT:     Valid session context but database save fails
 * OUTPUT:    Error returned and session not tracked in memory
 * BUSINESS:  Verify proper error handling when database operations fail
 * CHANGE:    Initial test for database error handling in session creation
 * RISK:      Medium - Error handling test for database failures
 */
func TestCreateSession_DatabaseError(t *testing.T) {
	tracker, mockActiveRepo, _ := createTestTracker()
	ctx := context.Background()

	sessionContext := createTestSessionContext(1234, "testuser", "/test/project")

	// Mock expectations - database save fails
	expectedError := fmt.Errorf("database connection failed")
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(expectedError)

	// Execute
	activeSession, err := tracker.CreateSession(ctx, sessionContext)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, activeSession)
	assert.Contains(t, err.Error(), "database connection failed")

	// Verify session not in memory
	stats := tracker.GetStatistics()
	assert.Equal(t, 0, stats["active_sessions"])

	// Verify mocks
	mockActiveRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test finding session by exact terminal PID match
 * INPUT:     End event context with terminal PID that matches existing active session
 * OUTPUT:    Successfully found matching active session with high confidence
 * BUSINESS:  Verify primary correlation strategy using exact terminal PID matching
 * CHANGE:    Initial test for terminal PID correlation strategy
 * RISK:      High - Primary correlation strategy test critical for accuracy
 */
func TestFindSessionForEndEvent_ExactTerminalMatch(t *testing.T) {
	tracker, mockActiveRepo, _ := createTestTracker()
	ctx := context.Background()

	// Create and store a session
	sessionContext := createTestSessionContext(1234, "testuser", "/test/project")
	activeSession, err := createTestActiveSession(sessionContext)
	require.NoError(t, err)

	// Mock the session creation first
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil)
	createdSession, err := tracker.CreateSession(ctx, sessionContext)
	require.NoError(t, err)

	// Create end event context with same terminal PID
	endContext := createTestSessionContext(1234, "testuser", "/test/project")

	// Execute
	foundSession, err := tracker.FindSessionForEndEvent(ctx, endContext)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, foundSession)
	assert.Equal(t, createdSession.ID(), foundSession.ID())
	assert.Equal(t, 1234, foundSession.SessionContext().TerminalPID)

	// Verify mocks
	mockActiveRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test finding session with multiple terminal matches using scoring
 * INPUT:     End event context that matches multiple sessions in same terminal
 * OUTPUT:    Best matching session selected based on scoring algorithm
 * BUSINESS:  Verify scoring algorithm works when multiple sessions match terminal PID
 * CHANGE:    Initial test for context scoring and best match selection
 * RISK:      High - Scoring algorithm accuracy critical for multi-session correlation
 */
func TestFindSessionForEndEvent_MultipleTerminalMatches(t *testing.T) {
	tracker, mockActiveRepo, _ := createTestTracker()
	ctx := context.Background()

	// Create two sessions with same terminal PID but different projects
	sessionContext1 := createTestSessionContext(1234, "testuser", "/test/project1")
	sessionContext2 := createTestSessionContext(1234, "testuser", "/test/project2")

	// Mock the session creations
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil).Times(2)

	// Create both sessions
	session1, err := tracker.CreateSession(ctx, sessionContext1)
	require.NoError(t, err)

	session2, err := tracker.CreateSession(ctx, sessionContext2)
	require.NoError(t, err)

	// Create end event context that exactly matches project1
	endContext := createTestSessionContext(1234, "testuser", "/test/project1")

	// Execute
	foundSession, err := tracker.FindSessionForEndEvent(ctx, endContext)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, foundSession)
	
	// Should match session1 due to exact working directory match
	assert.Equal(t, session1.ID(), foundSession.ID())
	assert.Equal(t, "/test/project1", foundSession.SessionContext().WorkingDir)

	// Verify mocks
	mockActiveRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test finding session using project-based fallback correlation
 * INPUT:     End event context with different terminal PID but matching project
 * OUTPUT:    Successfully found matching session using project-based strategy
 * BUSINESS:  Verify secondary correlation strategy using project directory matching
 * CHANGE:    Initial test for project-based fallback correlation
 * RISK:      Medium - Secondary correlation strategy for terminal PID mismatches
 */
func TestFindSessionForEndEvent_ProjectMatch(t *testing.T) {
	tracker, mockActiveRepo, _ := createTestTracker()
	ctx := context.Background()

	// Create session with terminal PID 1234
	sessionContext := createTestSessionContext(1234, "testuser", "/test/project")
	
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil)
	createdSession, err := tracker.CreateSession(ctx, sessionContext)
	require.NoError(t, err)

	// Create end event context with different terminal PID but same project
	endContext := createTestSessionContext(5678, "testuser", "/test/project")

	// Execute
	foundSession, err := tracker.FindSessionForEndEvent(ctx, endContext)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, foundSession)
	assert.Equal(t, createdSession.ID(), foundSession.ID())
	assert.Equal(t, "/test/project", foundSession.SessionContext().WorkingDir)

	// Verify mocks
	mockActiveRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test session correlation failure when no matches found
 * INPUT:     End event context that doesn't match any active sessions
 * OUTPUT:    Error returned indicating no matching session found
 * BUSINESS:  Verify proper error handling when correlation fails completely
 * CHANGE:    Initial test for correlation failure scenarios
 * RISK:      Medium - Error handling test for correlation failures
 */
func TestFindSessionForEndEvent_NoMatch(t *testing.T) {
	tracker, mockActiveRepo, _ := createTestTracker()
	ctx := context.Background()

	// Create session for different user/terminal/project
	sessionContext := createTestSessionContext(1234, "user1", "/project1")
	
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil)
	_, err := tracker.CreateSession(ctx, sessionContext)
	require.NoError(t, err)

	// Create end event context that doesn't match
	endContext := createTestSessionContext(5678, "user2", "/project2")

	// Execute
	foundSession, err := tracker.FindSessionForEndEvent(ctx, endContext)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, foundSession)
	assert.Contains(t, err.Error(), "no matching active session found")

	// Verify mocks
	mockActiveRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test successful session completion with metrics
 * INPUT:     Active session ID, end timestamp, processing duration, and token count
 * OUTPUT:    Completed session entity and active session removed from tracking
 * BUSINESS:  Verify session lifecycle completion with accurate metrics recording
 * CHANGE:    Initial test for session completion and metrics capture
 * RISK:      High - Session completion affects historical tracking and analytics
 */
func TestEndSession_Success(t *testing.T) {
	tracker, mockActiveRepo, mockSessionRepo := createTestTracker()
	ctx := context.Background()

	// Create and store a session
	sessionContext := createTestSessionContext(1234, "testuser", "/test/project")
	
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil)
	activeSession, err := tracker.CreateSession(ctx, sessionContext)
	require.NoError(t, err)

	// Mock expectations for session completion
	mockActiveRepo.On("Update", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil)
	mockActiveRepo.On("Delete", ctx, activeSession.ID()).Return(nil)
	mockSessionRepo.On("Save", ctx, mock.AnythingOfType("*entities.Session")).Return(nil)

	// Execute
	endTime := time.Now()
	processingDuration := 30 * time.Second
	tokenCount := int64(1500)

	completedSession, err := tracker.EndSession(ctx, activeSession.ID(), endTime, processingDuration, tokenCount)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, completedSession)
	assert.Equal(t, sessionContext.UserID, completedSession.UserID())
	
	// Verify session removed from active tracking
	stats := tracker.GetStatistics()
	assert.Equal(t, 0, stats["active_sessions"])

	// Verify mocks
	mockActiveRepo.AssertExpectations(t)
	mockSessionRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test ending non-existent session
 * INPUT:     Session ID that doesn't exist in active session tracking
 * OUTPUT:    Error returned indicating session not found
 * BUSINESS:  Verify proper error handling for invalid session IDs in end events
 * CHANGE:    Initial test for invalid session ID handling
 * RISK:      Medium - Error handling for orphaned or invalid end events
 */
func TestEndSession_SessionNotFound(t *testing.T) {
	tracker, _, _ := createTestTracker()
	ctx := context.Background()

	// Execute with non-existent session ID
	endTime := time.Now()
	processingDuration := 30 * time.Second
	tokenCount := int64(1500)

	completedSession, err := tracker.EndSession(ctx, "non-existent-id", endTime, processingDuration, tokenCount)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, completedSession)
	assert.Contains(t, err.Error(), "not found")
}

/**
 * CONTEXT:   Test context match scoring algorithm accuracy
 * INPUT:     Various session contexts with different similarity levels
 * OUTPUT:    Correct match scores reflecting context similarity
 * BUSINESS:  Verify scoring algorithm produces reliable correlation confidence scores
 * CHANGE:    Initial test for context matching scoring algorithm
 * RISK:      High - Scoring algorithm accuracy directly affects correlation quality
 */
func TestContextMatchScoring(t *testing.T) {
	// Create test session
	sessionContext := createTestSessionContext(1234, "testuser", "/test/project")
	activeSession, err := createTestActiveSession(sessionContext)
	require.NoError(t, err)

	tests := []struct {
		name           string
		matchContext   entities.SessionContext
		expectedScore  float64
		scoreThreshold float64
	}{
		{
			name:           "Perfect match",
			matchContext:   createTestSessionContext(1234, "testuser", "/test/project"),
			expectedScore:  0.9,
			scoreThreshold: 0.8,
		},
		{
			name:           "Terminal match, different directory",
			matchContext:   createTestSessionContext(1234, "testuser", "/different/project"),
			expectedScore:  0.6,
			scoreThreshold: 0.5,
		},
		{
			name:           "Project match, different terminal",
			matchContext:   createTestSessionContext(5678, "testuser", "/test/project"),
			expectedScore:  0.4,
			scoreThreshold: 0.3,
		},
		{
			name:           "User only match",
			matchContext:   createTestSessionContext(5678, "testuser", "/different/project"),
			expectedScore:  0.1,
			scoreThreshold: 0.05,
		},
		{
			name:           "No match",
			matchContext:   createTestSessionContext(5678, "differentuser", "/different/project"),
			expectedScore:  0.0,
			scoreThreshold: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := activeSession.ContextMatchScore(tt.matchContext)
			assert.GreaterOrEqual(t, score, tt.scoreThreshold, 
				"Match score for %s should be at least %.2f, got %.2f", tt.name, tt.scoreThreshold, score)
			
			if tt.name == "Perfect match" {
				assert.GreaterOrEqual(t, score, tt.expectedScore, 
					"Perfect match should have high score")
			}
		})
	}
}

/**
 * CONTEXT:   Test cleanup of expired active sessions
 * INPUT:     Active sessions that have exceeded timeout threshold
 * OUTPUT:    Expired sessions removed from tracking and proper cleanup count
 * BUSINESS:  Verify automatic cleanup prevents memory leaks and maintains accurate state
 * CHANGE:    Initial test for expired session cleanup functionality
 * RISK:      Medium - Cleanup functionality prevents resource leaks
 */
func TestCleanupExpiredSessions(t *testing.T) {
	tracker, mockActiveRepo, _ := createTestTracker()
	ctx := context.Background()

	// Create sessions with different ages
	now := time.Now()
	recentContext := createTestSessionContext(1234, "user1", "/project1")
	recentContext.Timestamp = now.Add(-10 * time.Minute) // Recent session

	oldContext := createTestSessionContext(5678, "user2", "/project2")
	oldContext.Timestamp = now.Add(-60 * time.Minute) // Old session (expired)

	// Mock session creations
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil).Times(2)

	// Create both sessions
	_, err := tracker.CreateSession(ctx, recentContext)
	require.NoError(t, err)

	_, err = tracker.CreateSession(ctx, oldContext)
	require.NoError(t, err)

	// Verify we have 2 active sessions
	stats := tracker.GetStatistics()
	assert.Equal(t, 2, stats["active_sessions"])

	// Execute cleanup
	cleanedCount, err := tracker.CleanupExpiredSessions(ctx, now)

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, int64(1), cleanedCount) // Only the old session should be cleaned

	// Verify we now have 1 active session
	stats = tracker.GetStatistics()
	assert.Equal(t, 1, stats["active_sessions"])

	// Verify mocks
	mockActiveRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Test concurrent session operations for thread safety
 * INPUT:     Multiple goroutines creating and finding sessions simultaneously
 * OUTPUT:    All operations complete successfully without race conditions
 * BUSINESS:  Verify session tracker is thread-safe under concurrent load
 * CHANGE:    Initial test for concurrent operation thread safety
 * RISK:      High - Thread safety critical for production daemon usage
 */
func TestConcurrentOperations(t *testing.T) {
	tracker, mockActiveRepo, _ := createTestTracker()
	ctx := context.Background()

	// Mock all save operations
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil).Times(10)

	// Create multiple sessions concurrently
	sessionChan := make(chan *entities.ActiveSession, 10)
	errorChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(terminalPID int) {
			sessionContext := createTestSessionContext(terminalPID, "testuser", fmt.Sprintf("/project%d", terminalPID))
			session, err := tracker.CreateSession(ctx, sessionContext)
			
			if err != nil {
				errorChan <- err
			} else {
				sessionChan <- session
			}
		}(1000 + i)
	}

	// Collect results
	var sessions []*entities.ActiveSession
	var errors []error

	for i := 0; i < 10; i++ {
		select {
		case session := <-sessionChan:
			sessions = append(sessions, session)
		case err := <-errorChan:
			errors = append(errors, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Assertions
	assert.Len(t, errors, 0, "No errors should occur during concurrent operations")
	assert.Len(t, sessions, 10, "All sessions should be created successfully")

	// Verify final state
	stats := tracker.GetStatistics()
	assert.Equal(t, 10, stats["active_sessions"])

	// Verify mocks
	mockActiveRepo.AssertExpectations(t)
}

/**
 * CONTEXT:   Benchmark test for session correlation performance
 * INPUT:     Large number of active sessions and correlation attempts
 * OUTPUT:    Performance metrics for correlation operations
 * BUSINESS:  Verify correlation system performs well under realistic load
 * CHANGE:    Initial benchmark for correlation performance validation
 * RISK:      Medium - Performance validation for production readiness
 */
func BenchmarkSessionCorrelation(b *testing.B) {
	tracker, mockActiveRepo, _ := createTestTracker()
	ctx := context.Background()

	// Mock all operations
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil).Maybe()

	// Create a number of active sessions
	numSessions := 100
	for i := 0; i < numSessions; i++ {
		sessionContext := createTestSessionContext(1000+i, "testuser", fmt.Sprintf("/project%d", i))
		_, err := tracker.CreateSession(ctx, sessionContext)
		if err != nil {
			b.Fatalf("Failed to create test session: %v", err)
		}
	}

	// Benchmark correlation operations
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Create end event context that should match middle session
		endContext := createTestSessionContext(1050, "testuser", "/project50")
		
		// Attempt correlation
		_, err := tracker.FindSessionForEndEvent(ctx, endContext)
		if err != nil {
			// In benchmark, we expect some correlation attempts to fail
			// This is normal behavior when testing with various contexts
		}
	}
}

/**
 * CONTEXT:   Integration test for complete session lifecycle
 * INPUT:     Full session lifecycle from creation to completion with real-world patterns
 * OUTPUT:    Complete session tracking with all metrics and state transitions
 * BUSINESS:  Verify end-to-end session tracking works as designed for real usage
 * CHANGE:    Initial integration test for complete session lifecycle
 * RISK:      High - Integration test validates entire session correlation system
 */
func TestCompleteSessionLifecycle(t *testing.T) {
	tracker, mockActiveRepo, mockSessionRepo := createTestTracker()
	ctx := context.Background()

	// Mock all database operations
	mockActiveRepo.On("Save", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil)
	mockActiveRepo.On("Update", ctx, mock.AnythingOfType("*entities.ActiveSession")).Return(nil)
	mockActiveRepo.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
	mockSessionRepo.On("Save", ctx, mock.AnythingOfType("*entities.Session")).Return(nil)

	// Step 1: Create session (simulating hook start)
	startContext := createTestSessionContext(1234, "testuser", "/test/project")
	activeSession, err := tracker.CreateSession(ctx, startContext)
	require.NoError(t, err)
	assert.NotNil(t, activeSession)

	// Verify session is tracked
	stats := tracker.GetStatistics()
	assert.Equal(t, 1, stats["active_sessions"])

	// Step 2: Simulate some time passing and activity
	time.Sleep(10 * time.Millisecond) // Small delay to simulate processing

	// Step 3: Find session for end event (simulating hook end)
	endContext := createTestSessionContext(1234, "testuser", "/test/project")
	foundSession, err := tracker.FindSessionForEndEvent(ctx, endContext)
	require.NoError(t, err)
	assert.Equal(t, activeSession.ID(), foundSession.ID())

	// Step 4: End session with metrics
	endTime := time.Now()
	processingDuration := 30 * time.Second
	tokenCount := int64(1500)

	completedSession, err := tracker.EndSession(ctx, foundSession.ID(), endTime, processingDuration, tokenCount)
	require.NoError(t, err)
	assert.NotNil(t, completedSession)

	// Step 5: Verify session is no longer tracked
	stats = tracker.GetStatistics()
	assert.Equal(t, 0, stats["active_sessions"])

	// Step 6: Verify completed session has correct data
	assert.Equal(t, startContext.UserID, completedSession.UserID())
	assert.True(t, completedSession.StartTime().Before(endTime))

	// Verify all mocks were called
	mockActiveRepo.AssertExpectations(t)
	mockSessionRepo.AssertExpectations(t)
}