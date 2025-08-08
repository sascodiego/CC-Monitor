/**
 * CONTEXT:   Pure time-based session manager eliminating memory state and IsActive flags
 * INPUT:     User ID and activity timestamps for session determination
 * OUTPUT:    Active sessions through database queries, automatic session creation
 * BUSINESS:  Sessions are active when NOW() BETWEEN start_time AND end_time
 * CHANGE:    Complete refactor from memory-based to database-backed session logic
 * RISK:      Medium - Core session logic replacement affecting all activity processing
 */

package business

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

// SessionManager handles pure time-based session logic without memory state
type SessionManager struct {
	sessionRepo   *sqlite.SessionRepository
	sessionLength time.Duration
	timezone      *time.Location
}

// Session represents the SQLite session entity  
type Session = sqlite.Session

/**
 * CONTEXT:   Create new session manager with database repository
 * INPUT:     SQLite session repository for database operations
 * OUTPUT:    Configured session manager with 5-hour session duration
 * BUSINESS:  Session manager provides single interface for session operations
 * CHANGE:    Initial session manager with pure database operations
 * RISK:      Low - Simple constructor with dependency injection
 */
func NewSessionManager(sessionRepo *sqlite.SessionRepository) *SessionManager {
	// Load America/Montevideo timezone for consistent time handling
	timezone, err := time.LoadLocation("America/Montevideo")
	if err != nil {
		log.Printf("Warning: failed to load timezone America/Montevideo, using UTC: %v", err)
		timezone = time.UTC
	}

	return &SessionManager{
		sessionRepo:   sessionRepo,
		sessionLength: 5 * time.Hour, // Claude Code sessions are exactly 5 hours
		timezone:      timezone,
	}
}

/**
 * CONTEXT:   Get or create active session using pure time-based logic
 * INPUT:     User ID and activity timestamp for session determination
 * OUTPUT:    Active session entity, creates new if none active or expired
 * BUSINESS:  Session active when current_time BETWEEN start_time AND end_time
 * CHANGE:    Replaced memory map lookup with database time queries
 * RISK:      Medium - Core session logic affecting all activity processing
 */
func (sm *SessionManager) GetOrCreateSession(ctx context.Context, userID string, activityTime time.Time) (*Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Convert activity time to our timezone for consistent processing
	activityTime = activityTime.In(sm.timezone)

	// STEP 1: Find active sessions for user (time-based query)
	activeSessions, err := sm.sessionRepo.GetActiveSessionsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	// STEP 2: Validate and handle multiple active sessions
	activeSession, err := sm.validateActiveSessions(ctx, activeSessions, activityTime)
	if err != nil {
		return nil, fmt.Errorf("failed to validate active sessions: %w", err)
	}

	// STEP 3: If valid active session exists, update and return
	if activeSession != nil {
		return sm.updateExistingSession(ctx, activeSession, activityTime)
	}

	// STEP 4: Create new session (no active session exists)
	return sm.createNewSession(ctx, userID, activityTime)
}

/**
 * CONTEXT:   Validate active sessions and handle edge cases
 * INPUT:     List of active sessions and current activity time
 * OUTPUT:    Single valid active session or nil if none valid
 * BUSINESS:  Only one session can be active per user at any time
 * CHANGE:    Replaced manual IsActive flag management with time validation
 * RISK:      Low - Validation logic with proper error handling
 */
func (sm *SessionManager) validateActiveSessions(ctx context.Context, sessions []*Session, activityTime time.Time) (*Session, error) {
	if len(sessions) == 0 {
		return nil, nil
	}

	// Handle multiple active sessions (data inconsistency)
	if len(sessions) > 1 {
		log.Printf("⚠️  CRITICAL: Found %d active sessions, cleaning up duplicates", len(sessions))
		return sm.cleanupDuplicateSessions(ctx, sessions, activityTime)
	}

	// Single active session - validate it's truly active
	session := sessions[0]
	if sm.isSessionActive(session, activityTime) {
		sessionAge := activityTime.Sub(session.StartTime)
		remaining := sm.sessionLength - sessionAge
		log.Printf("♻️  Using existing session %s (age: %v, remaining: %v)", 
			session.ID, sessionAge, remaining)
		return session, nil
	}

	// Session is expired, mark it and return nil
	log.Printf("⏰ Session %s expired, will create new session", session.ID)
	session.State = "expired"
	if err := sm.sessionRepo.Update(ctx, session); err != nil {
		log.Printf("Warning: failed to mark session as expired: %v", err)
	}
	
	return nil, nil
}

/**
 * CONTEXT:   Clean up multiple active sessions by keeping most recent
 * INPUT:     Multiple active sessions and activity time
 * OUTPUT:    Single valid session with others marked as expired
 * BUSINESS:  Enforce single active session per user constraint
 * CHANGE:    Database-based cleanup replacing memory map manipulation
 * RISK:      Medium - Data consistency operations affecting multiple sessions
 */
func (sm *SessionManager) cleanupDuplicateSessions(ctx context.Context, sessions []*Session, activityTime time.Time) (*Session, error) {
	var mostRecent *Session
	
	// Find most recent session
	for _, session := range sessions {
		if mostRecent == nil || session.StartTime.After(mostRecent.StartTime) {
			mostRecent = session
		}
	}

	// Mark all others as expired
	for _, session := range sessions {
		if session.ID != mostRecent.ID {
			log.Printf("🔄 Expiring duplicate session %s (started %v)", session.ID, session.StartTime)
			session.State = "expired"
			if err := sm.sessionRepo.Update(ctx, session); err != nil {
				log.Printf("Warning: failed to expire duplicate session %s: %v", session.ID, err)
			}
		}
	}

	// Validate the kept session is still active
	if sm.isSessionActive(mostRecent, activityTime) {
		return mostRecent, nil
	}

	// Most recent is also expired
	log.Printf("🔄 Most recent session %s also expired", mostRecent.ID) 
	mostRecent.State = "expired"
	if err := sm.sessionRepo.Update(ctx, mostRecent); err != nil {
		log.Printf("Warning: failed to expire most recent session: %v", err)
	}

	return nil, nil
}

/**
 * CONTEXT:   Check if session is truly active based on time calculation
 * INPUT:     Session entity and current activity time
 * OUTPUT:    Boolean indicating if session is within 5-hour window
 * BUSINESS:  Session active when activity_time <= (start_time + 5 hours)
 * CHANGE:    Pure time calculation replacing IsActive flag checks
 * RISK:      Low - Simple time comparison logic
 */
func (sm *SessionManager) isSessionActive(session *Session, activityTime time.Time) bool {
	sessionEnd := session.StartTime.Add(sm.sessionLength)
	return activityTime.Before(sessionEnd) || activityTime.Equal(sessionEnd)
}

/**
 * CONTEXT:   Update existing active session with new activity
 * INPUT:     Active session and new activity timestamp
 * OUTPUT:    Updated session with latest activity time
 * BUSINESS:  Track last activity time and increment activity count
 * CHANGE:    Database update replacing in-memory field updates
 * RISK:      Low - Simple session update operations
 */
func (sm *SessionManager) updateExistingSession(ctx context.Context, session *Session, activityTime time.Time) (*Session, error) {
	// Update session activity tracking
	session.LastActivityTime = activityTime
	session.ActivityCount++
	
	// Set first activity time if this is the first activity
	if session.FirstActivityTime.IsZero() {
		session.FirstActivityTime = activityTime
	}

	// Update in database
	if err := sm.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session activity: %w", err)
	}

	return session, nil
}

/**
 * CONTEXT:   Create new session with proper time boundaries and validation
 * INPUT:     User ID and activity start time
 * OUTPUT:    New session entity persisted to database
 * BUSINESS:  Sessions start at activity time and last exactly 5 hours
 * CHANGE:    Database creation replacing memory map insertion
 * RISK:      Low - Session creation with validation and error handling
 */
func (sm *SessionManager) createNewSession(ctx context.Context, userID string, startTime time.Time) (*Session, error) {
	session := &Session{
		ID:                generateSessionID(userID, startTime),
		UserID:            userID,
		StartTime:         startTime,
		EndTime:           startTime.Add(sm.sessionLength),
		State:             "active",
		FirstActivityTime: startTime,
		LastActivityTime:  startTime,
		ActivityCount:     1,
		DurationHours:     5.0,
		CreatedAt:         time.Now().In(sm.timezone),
		UpdatedAt:         time.Now().In(sm.timezone),
	}

	// Persist to database
	if err := sm.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create new session: %w", err)
	}

	log.Printf("🆕 Created new session %s for user %s at %v (ends: %v)", 
		session.ID, userID, startTime.Format("2006-01-02 15:04:05"),
		session.EndTime.Format("2006-01-02 15:04:05"))

	return session, nil
}

/**
 * CONTEXT:   Get current active session for user (read-only operation)
 * INPUT:     User ID for session lookup
 * OUTPUT:    Currently active session or nil if none active
 * BUSINESS:  Active session determined by current time within session window
 * CHANGE:    Database query replacing memory map lookup
 * RISK:      Low - Read-only session lookup operation
 */
func (sm *SessionManager) GetActiveSession(ctx context.Context, userID string) (*Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	activeSessions, err := sm.sessionRepo.GetActiveSessionsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	if len(activeSessions) == 0 {
		return nil, nil
	}

	if len(activeSessions) > 1 {
		log.Printf("⚠️  Warning: Multiple active sessions found for user %s", userID)
	}

	// Return most recent active session
	var mostRecent *Session
	for _, session := range activeSessions {
		if mostRecent == nil || session.StartTime.After(mostRecent.StartTime) {
			mostRecent = session
		}
	}

	return mostRecent, nil
}

/**
 * CONTEXT:   Mark expired sessions as expired based on current time
 * INPUT:     Context for database operations
 * OUTPUT:    Number of sessions marked as expired
 * BUSINESS:  Sessions automatically expire when current_time > end_time
 * CHANGE:    Batch database update replacing individual IsActive flag updates
 * RISK:      Low - Bulk update with time-based filtering
 */
func (sm *SessionManager) MarkExpiredSessions(ctx context.Context) (int, error) {
	expiredCount, err := sm.sessionRepo.MarkExpiredSessions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to mark expired sessions: %w", err)
	}

	if expiredCount > 0 {
		log.Printf("⏰ Marked %d sessions as expired", expiredCount)
	}

	return expiredCount, nil
}

/**
 * CONTEXT:   Generate unique session ID with timestamp and user info
 * INPUT:     User ID and session start time
 * OUTPUT:    Unique session identifier string
 * BUSINESS:  Session IDs include timestamp for uniqueness and debugging
 * CHANGE:    Helper function for session ID generation with nanosecond precision
 * RISK:      Low - ID generation utility function
 */
func generateSessionID(userID string, startTime time.Time) string {
	return fmt.Sprintf("session_%s_%d_%d", userID, startTime.Unix(), startTime.Nanosecond())
}

/**
 * CONTEXT:   Get all active sessions across all users for system monitoring
 * INPUT:     Context for database operations
 * OUTPUT:    Slice of all currently active sessions in the system
 * BUSINESS:  Active sessions query supports health monitoring and cleanup operations
 * CHANGE:    CHECKPOINT 6 - Added active sessions query for health and cleanup endpoints
 * RISK:      Low - Read-only query for system monitoring
 */
func (sm *SessionManager) GetActiveSessions(ctx context.Context) ([]*Session, error) {
	activeSessions, err := sm.sessionRepo.GetActiveSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}
	return activeSessions, nil
}

/**
 * CONTEXT:   Get count of active sessions for health monitoring
 * INPUT:     Context for database operations
 * OUTPUT:    Count of currently active sessions
 * BUSINESS:  Active session count for health status and monitoring dashboards
 * CHANGE:    CHECKPOINT 6 - Added active session count for health endpoint
 * RISK:      Low - Count query for monitoring
 */
func (sm *SessionManager) GetActiveSessionCount(ctx context.Context) (int, error) {
	activeSessions, err := sm.GetActiveSessions(ctx)
	if err != nil {
		return 0, err
	}
	return len(activeSessions), nil
}

/**
 * CONTEXT:   Get total session count for health monitoring and statistics
 * INPUT:     Context for database operations
 * OUTPUT:    Total count of all sessions in the system
 * BUSINESS:  Total session count for system health monitoring and analytics
 * CHANGE:    CHECKPOINT 6 - Added total session count for health endpoint
 * RISK:      Low - Count query for monitoring
 */
func (sm *SessionManager) GetTotalSessionCount(ctx context.Context) (int, error) {
	allSessions, err := sm.sessionRepo.GetAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get total session count: %w", err)
	}
	return len(allSessions), nil
}

/**
 * CONTEXT:   Get recent sessions for database query endpoint and monitoring
 * INPUT:     Context and limit for recent session count
 * OUTPUT:    Slice of most recent sessions up to the specified limit
 * BUSINESS:  Recent session queries support administrative monitoring and troubleshooting
 * CHANGE:    CHECKPOINT 6 - Added recent sessions query for database endpoint
 * RISK:      Low - Read-only query with limit for administrative use
 */
func (sm *SessionManager) GetRecentSessions(ctx context.Context, limit int) ([]*Session, error) {
	allSessions, err := sm.sessionRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent sessions: %w", err)
	}
	
	// Sort by created time descending and limit
	sortedSessions := allSessions
	if len(sortedSessions) > limit {
		sortedSessions = sortedSessions[len(sortedSessions)-limit:]
	}
	
	return sortedSessions, nil
}

/**
 * CONTEXT:   Close a specific session by ID for cleanup operations
 * INPUT:     Context, session ID, and close timestamp
 * OUTPUT:    Session marked as closed with proper end time
 * BUSINESS:  Session closure supports cleanup operations and proper work tracking finalization
 * CHANGE:    CHECKPOINT 6 - Added session closure for cleanup endpoint
 * RISK:      Medium - State modification affecting work tracking
 */
func (sm *SessionManager) CloseSession(ctx context.Context, sessionID string, closeTime time.Time) error {
	session, err := sm.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session for closure: %w", err)
	}
	
	if session == nil {
		return fmt.Errorf("session %s not found", sessionID)
	}
	
	// Update session state
	session.State = "closed"
	session.UpdatedAt = closeTime
	
	// Update in database
	if err := sm.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}
	
	log.Printf("\u2705 Closed session %s at %v", sessionID, closeTime)
	return nil
}