/**
 * CONTEXT:   Domain entity representing a Claude session with 5-hour window business logic
 * INPUT:     User ID, start time, and session configuration parameters
 * OUTPUT:    Session entity with validation, state management, and business rule enforcement
 * BUSINESS:  Sessions are exactly 5 hours long, starting from first user interaction
 * CHANGE:    Initial implementation following Clean Architecture and SOLID principles
 * RISK:      Low - Domain entity with no external dependencies, pure business logic
 */

package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionState represents the current state of a session
type SessionState string

const (
	SessionStateActive   SessionState = "active"
	SessionStateExpired  SessionState = "expired"
	SessionStateFinished SessionState = "finished"
)

// SessionDuration defines the exact duration of a Claude session (5 hours)
const SessionDuration = 5 * time.Hour

/**
 * CONTEXT:   Core session entity implementing 5-hour window business rules
 * INPUT:     Session data with timestamps, user info, and work block references
 * OUTPUT:    Immutable session entity with validation and business logic methods
 * BUSINESS:  Sessions track user activity within exact 5-hour windows
 * CHANGE:    Initial domain entity with full business logic implementation
 * RISK:      Low - Pure domain logic with no side effects or external dependencies
 */
type Session struct {
	id                string
	userID            string
	startTime         time.Time
	endTime           time.Time
	state             SessionState
	firstActivityTime time.Time
	lastActivityTime  time.Time
	activityCount     int64
	workBlockIDs      []string
	createdAt         time.Time
	updatedAt         time.Time
}

// SessionConfig holds configuration for creating new sessions
type SessionConfig struct {
	UserID    string
	StartTime time.Time
}

/**
 * CONTEXT:   Factory method for creating new session entities with proper validation
 * INPUT:     SessionConfig with user ID and start time
 * OUTPUT:    Valid Session entity or validation error
 * BUSINESS:  New sessions start with provided timestamp and last exactly 5 hours
 * CHANGE:    Initial implementation with comprehensive validation
 * RISK:      Low - Validation prevents invalid session creation
 */
func NewSession(config SessionConfig) (*Session, error) {
	// Validate required fields
	if config.UserID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	if config.StartTime.IsZero() {
		return nil, fmt.Errorf("start time cannot be zero")
	}

	// Validate start time is not too far in the future
	maxFutureTime := time.Now().Add(1 * time.Hour)
	if config.StartTime.After(maxFutureTime) {
		return nil, fmt.Errorf("start time cannot be more than 1 hour in the future")
	}

	// Validate start time is not too old (more than 24 hours ago)
	minPastTime := time.Now().Add(-24 * time.Hour)
	if config.StartTime.Before(minPastTime) {
		return nil, fmt.Errorf("start time cannot be more than 24 hours in the past")
	}

	now := time.Now()
	sessionID := uuid.New().String()

	session := &Session{
		id:                sessionID,
		userID:            config.UserID,
		startTime:         config.StartTime,
		endTime:           config.StartTime.Add(SessionDuration),
		state:             SessionStateActive,
		firstActivityTime: config.StartTime,
		lastActivityTime:  config.StartTime,
		activityCount:     1,
		workBlockIDs:      make([]string, 0),
		createdAt:         now,
		updatedAt:         now,
	}

	return session, nil
}

// Getter methods (immutable access)
func (s *Session) ID() string                 { return s.id }
func (s *Session) UserID() string             { return s.userID }
func (s *Session) StartTime() time.Time       { return s.startTime }
func (s *Session) EndTime() time.Time         { return s.endTime }
func (s *Session) State() SessionState        { return s.state }
func (s *Session) FirstActivityTime() time.Time { return s.firstActivityTime }
func (s *Session) LastActivityTime() time.Time  { return s.lastActivityTime }
func (s *Session) ActivityCount() int64       { return s.activityCount }
func (s *Session) WorkBlockIDs() []string     { return append([]string(nil), s.workBlockIDs...) }
func (s *Session) CreatedAt() time.Time       { return s.createdAt }
func (s *Session) UpdatedAt() time.Time       { return s.updatedAt }

/**
 * CONTEXT:   Business logic for determining if activity timestamp is valid for this session
 * INPUT:     Activity timestamp to validate against session window
 * OUTPUT:    Boolean indicating if activity fits within 5-hour session window
 * BUSINESS:  Activities only valid within session start/end time boundaries
 * CHANGE:    Initial validation logic for session time boundaries
 * RISK:      Low - Pure validation logic with no side effects
 */
func (s *Session) IsActivityValid(activityTime time.Time) bool {
	// Activity must be within session time window
	if activityTime.Before(s.startTime) || activityTime.After(s.endTime) {
		return false
	}

	// Activity must not be too far in future (clock skew protection)
	maxFutureTime := time.Now().Add(5 * time.Minute)
	if activityTime.After(maxFutureTime) {
		return false
	}

	return true
}

/**
 * CONTEXT:   Check if session has expired based on current time and business rules
 * INPUT:     Current timestamp for comparison
 * OUTPUT:    Boolean indicating if session is expired
 * BUSINESS:  Session expires when current time > session end time
 * CHANGE:    Initial expiration logic implementation
 * RISK:      Low - Simple time comparison with no side effects
 */
func (s *Session) IsExpired(currentTime time.Time) bool {
	return currentTime.After(s.endTime)
}

/**
 * CONTEXT:   Check if session is currently active based on state and expiration
 * INPUT:     No parameters, uses current time and session state
 * OUTPUT:    Boolean indicating if session is active
 * BUSINESS:  Session is active if state is active and not expired
 * CHANGE:    Added IsActive method for compatibility
 * RISK:      Low - Simple state and time checking
 */
func (s *Session) IsActive() bool {
	return s.state == SessionStateActive && !s.IsExpired(time.Now())
}

/**
 * CONTEXT:   Calculate session duration and efficiency metrics
 * INPUT:     No parameters, uses internal session state
 * OUTPUT:    Duration metrics including total time and efficiency percentage
 * BUSINESS:  Sessions are always 5 hours total, efficiency based on activity density
 * CHANGE:    Initial metrics calculation implementation
 * RISK:      Low - Pure calculation based on timestamps
 */
func (s *Session) Duration() time.Duration {
	return s.endTime.Sub(s.startTime)
}

func (s *Session) ElapsedTime(currentTime time.Time) time.Duration {
	if currentTime.Before(s.startTime) {
		return 0
	}
	if currentTime.After(s.endTime) {
		return s.Duration()
	}
	return currentTime.Sub(s.startTime)
}

func (s *Session) RemainingTime(currentTime time.Time) time.Duration {
	if currentTime.After(s.endTime) {
		return 0
	}
	return s.endTime.Sub(currentTime)
}

/**
 * CONTEXT:   Record new activity within the session with timestamp validation
 * INPUT:     Activity timestamp and optional work block ID
 * OUTPUT:    Error if activity is invalid, nil on success
 * BUSINESS:  Activities update last activity time and increment activity count
 * CHANGE:    Initial activity recording with validation
 * RISK:      Low - Validation prevents invalid state transitions
 */
func (s *Session) RecordActivity(activityTime time.Time, workBlockID string) error {
	// Validate activity time
	if !s.IsActivityValid(activityTime) {
		return fmt.Errorf("activity time %v is not valid for session window %v to %v",
			activityTime, s.startTime, s.endTime)
	}

	// Validate session is still active
	if s.state != SessionStateActive {
		return fmt.Errorf("cannot record activity on %s session", s.state)
	}

	// Update session state
	s.lastActivityTime = activityTime
	s.activityCount++
	s.updatedAt = time.Now()

	// Add work block ID if provided
	if workBlockID != "" {
		// Check if work block ID already exists
		for _, id := range s.workBlockIDs {
			if id == workBlockID {
				return nil // Already recorded
			}
		}
		s.workBlockIDs = append(s.workBlockIDs, workBlockID)
	}

	return nil
}

/**
 * CONTEXT:   Finalize session when it expires or is manually closed
 * INPUT:     Final timestamp for session completion
 * OUTPUT:    Error if session cannot be finalized, nil on success
 * BUSINESS:  Sessions transition to finished state and cannot be modified further
 * CHANGE:    Initial session finalization logic
 * RISK:      Low - State transition with validation
 */
func (s *Session) Finalize(finalTime time.Time) error {
	if s.state == SessionStateFinished {
		return fmt.Errorf("session is already finalized")
	}

	// Set appropriate final state
	if finalTime.After(s.endTime) {
		s.state = SessionStateExpired
	} else {
		s.state = SessionStateFinished
	}

	s.updatedAt = time.Now()
	return nil
}

/**
 * CONTEXT:   Export session data for serialization or reporting
 * INPUT:     No parameters, uses internal session state
 * OUTPUT:    SessionData struct suitable for JSON serialization
 * BUSINESS:  Provide read-only view of session data for external use
 * CHANGE:    Initial data export implementation
 * RISK:      Low - Read-only operation with no state changes
 */
type SessionData struct {
	ID                string       `json:"id"`
	UserID            string       `json:"user_id"`
	StartTime         time.Time    `json:"start_time"`
	EndTime           time.Time    `json:"end_time"`
	State             SessionState `json:"state"`
	FirstActivityTime time.Time    `json:"first_activity_time"`
	LastActivityTime  time.Time    `json:"last_activity_time"`
	ActivityCount     int64        `json:"activity_count"`
	WorkBlockIDs      []string     `json:"work_block_ids"`
	DurationHours     float64      `json:"duration_hours"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
}

func (s *Session) ToData() SessionData {
	return SessionData{
		ID:                s.id,
		UserID:            s.userID,
		StartTime:         s.startTime,
		EndTime:           s.endTime,
		State:             s.state,
		FirstActivityTime: s.firstActivityTime,
		LastActivityTime:  s.lastActivityTime,
		ActivityCount:     s.activityCount,
		WorkBlockIDs:      s.WorkBlockIDs(), // Use getter for safe copy
		DurationHours:     s.Duration().Hours(),
		CreatedAt:         s.createdAt,
		UpdatedAt:         s.updatedAt,
	}
}

/**
 * CONTEXT:   Validate session internal state consistency
 * INPUT:     No parameters, validates internal state
 * OUTPUT:    Error if state is inconsistent, nil if valid
 * BUSINESS:  Ensure session data integrity and business rule compliance
 * CHANGE:    Initial state validation implementation
 * RISK:      Low - Validation only, no state changes
 */
func (s *Session) Validate() error {
	// Check required fields
	if s.id == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if s.userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	// Check time relationships
	if s.endTime.Sub(s.startTime) != SessionDuration {
		return fmt.Errorf("session duration must be exactly %v, got %v",
			SessionDuration, s.endTime.Sub(s.startTime))
	}

	if s.lastActivityTime.Before(s.firstActivityTime) {
		return fmt.Errorf("last activity time cannot be before first activity time")
	}

	if s.firstActivityTime.Before(s.startTime) {
		return fmt.Errorf("first activity time cannot be before session start time")
	}

	if s.lastActivityTime.After(s.endTime) {
		return fmt.Errorf("last activity time cannot be after session end time")
	}

	// Check activity count
	if s.activityCount < 1 {
		return fmt.Errorf("session must have at least 1 activity")
	}

	return nil
}