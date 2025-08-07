/**
 * CONTEXT:   Session management use case implementing 5-hour Claude session windows
 * INPUT:     User activity events, timestamps, and session management requests
 * OUTPUT:    Session entities with proper state management and business rule enforcement
 * BUSINESS:  Manages 5-hour session windows starting from first interaction
 * CHANGE:    Initial implementation of session management business logic.
 * RISK:      Medium - Critical for accurate time tracking, requires precise timing logic
 */

package usecases

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/claude-monitor/system/internal/usecases/repositories"
)

// SessionManager handles session lifecycle and business rules
type SessionManager struct {
	sessionRepo     repositories.SessionRepository
	sessionDuration time.Duration
	logger          *log.Logger
}

// NewSessionManager creates a new session manager with dependencies
func NewSessionManager(sessionRepo repositories.SessionRepository, logger *log.Logger) *SessionManager {
	return &SessionManager{
		sessionRepo:     sessionRepo,
		sessionDuration: 5 * time.Hour, // Claude session duration
		logger:          logger,
	}
}

/**
 * CONTEXT:   Get or create session implementing 5-hour window business rule
 * INPUT:     User ID and activity timestamp for session determination
 * OUTPUT:    Active session entity, creates new if current expired or none exists
 * BUSINESS:  New session starts when > 5 hours from current session start time
 * CHANGE:    Initial implementation with precise 5-hour session logic.
 * RISK:      Medium - Critical timing logic for accurate session tracking
 */
func (sm *SessionManager) GetOrCreateSession(ctx context.Context, userID string, timestamp time.Time) (*entities.Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Get currently active session
	activeSession, err := sm.sessionRepo.FindActiveSession(ctx, userID)
	if err != nil && !isNotFoundError(err) {
		sm.logger.Printf("Error finding active session for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to find active session: %w", err)
	}

	// Check if we need a new session
	if activeSession == nil || sm.IsSessionExpired(activeSession, timestamp) {
		// Close expired session if exists
		if activeSession != nil && sm.IsSessionExpired(activeSession, timestamp) {
			sm.logger.Printf("Closing expired session %s for user %s", activeSession.ID, userID)
			err = sm.closeSession(ctx, activeSession, timestamp)
			if err != nil {
				sm.logger.Printf("Warning: failed to close expired session: %v", err)
			}
		}

		// Create new session
		newSession, err := sm.createNewSession(ctx, userID, timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to create new session: %w", err)
		}

		sm.logger.Printf("Created new session %s for user %s at %v", newSession.ID, userID, timestamp)
		return newSession, nil
	}

	// Update activity in existing session
	err = sm.UpdateSessionActivity(ctx, activeSession.ID(), timestamp)
	if err != nil {
		sm.logger.Printf("Warning: failed to update session activity: %v", err)
	}

	return activeSession, nil
}

/**
 * CONTEXT:   Check if session has expired based on 5-hour window rule
 * INPUT:     Session entity and current timestamp for expiration check
 * OUTPUT:    Boolean indicating if session has exceeded 5-hour window
 * BUSINESS:  Session expires exactly 5 hours after start time, regardless of activity
 * CHANGE:    Initial implementation with precise expiration logic.
 * RISK:      Low - Simple time comparison with well-defined business rule
 */
func (sm *SessionManager) IsSessionExpired(session *entities.Session, timestamp time.Time) bool {
	if session == nil {
		return true
	}

	// Session expires after exactly 5 hours from start time
	expirationTime := session.StartTime().Add(sm.sessionDuration)
	return timestamp.After(expirationTime)
}

/**
 * CONTEXT:   Update session with latest activity timestamp
 * INPUT:     Session ID and activity timestamp
 * OUTPUT:    Updated session with latest activity time
 * BUSINESS:  Track last activity for session analytics without affecting expiration
 * CHANGE:    Initial implementation of activity tracking.
 * RISK:      Low - Simple timestamp update operation
 */
func (sm *SessionManager) UpdateSessionActivity(ctx context.Context, sessionID string, timestamp time.Time) error {
	session, err := sm.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to find session %s: %w", sessionID, err)
	}

	// Update last activity time
	session.RecordActivity(timestamp)

	err = sm.sessionRepo.Update(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Close all expired sessions for cleanup and accurate state management
 * INPUT:     Current timestamp to determine which sessions have expired
 * OUTPUT:    Number of sessions closed and any errors encountered
 * BUSINESS:  Maintain accurate session states by closing expired sessions
 * CHANGE:    Initial implementation of session cleanup process.
 * RISK:      Medium - Batch operation affecting multiple sessions
 */
func (sm *SessionManager) CloseExpiredSessions(ctx context.Context, currentTime time.Time) (int64, error) {
	// Find sessions that expired before current time
	expiredBefore := currentTime.Add(-sm.sessionDuration)
	expiredSessions, err := sm.sessionRepo.FindExpiredSessions(ctx, expiredBefore)
	if err != nil {
		return 0, fmt.Errorf("failed to find expired sessions: %w", err)
	}

	var closedCount int64
	for _, session := range expiredSessions {
		err = sm.closeSession(ctx, session, currentTime)
		if err != nil {
			sm.logger.Printf("Failed to close expired session %s: %v", session.ID, err)
			continue
		}
		closedCount++
	}

	sm.logger.Printf("Closed %d expired sessions", closedCount)
	return closedCount, nil
}

/**
 * CONTEXT:   Get session statistics for analytics and reporting
 * INPUT:     User ID and date range for statistics calculation
 * OUTPUT:    Session statistics including total time, session count, etc.
 * BUSINESS:  Provide session analytics for productivity insights
 * CHANGE:    Initial implementation of session statistics.
 * RISK:      Low - Read-only statistical operations
 */
func (sm *SessionManager) GetSessionStatistics(ctx context.Context, userID string, start, end time.Time) (*repositories.SessionStatistics, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	stats, err := sm.sessionRepo.GetSessionStatistics(ctx, userID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get session statistics: %w", err)
	}

	return stats, nil
}

// Private helper methods

func (sm *SessionManager) createNewSession(ctx context.Context, userID string, startTime time.Time) (*entities.Session, error) {
	session, err := entities.NewSession(userID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to create session entity: %w", err)
	}

	err = sm.sessionRepo.Save(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to save new session: %w", err)
	}

	return session, nil
}

func (sm *SessionManager) closeSession(ctx context.Context, session *entities.Session, endTime time.Time) error {
	// Sessions automatically end after 5 hours, but we record the actual end time
	session.End(endTime)

	err := sm.sessionRepo.Update(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	return nil
}

// Helper function to check if error is "not found"
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// This would be implemented based on the specific repository error types
	return err.Error() == "session not found" || err == repositories.ErrSessionNotFound
}