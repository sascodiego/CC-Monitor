/**
 * CONTEXT:   Active session entity for daemon-managed context-based correlation system
 * INPUT:     Terminal context, user ID, and activity timestamps for session correlation
 * OUTPUT:    Active session with context matching and lifecycle management
 * BUSINESS:  Track active Claude sessions without temporary files or environment variables
 * CHANGE:    Initial implementation replacing file-based correlation with context-based matching
 * RISK:      Medium - Core correlation logic that affects all session tracking accuracy
 */

package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionContext represents the environment context for session correlation
type SessionContext struct {
	TerminalPID   int       `json:"terminal_pid"`
	ShellPID      int       `json:"shell_pid"`
	WorkingDir    string    `json:"working_dir"`
	ProjectPath   string    `json:"project_path"`
	UserID        string    `json:"user_id"`
	Timestamp     time.Time `json:"timestamp"`
}

// ActiveSessionStatus represents the current status of an active session
type ActiveSessionStatus string

const (
	ActiveSessionStatusActive     ActiveSessionStatus = "active"
	ActiveSessionStatusProcessing ActiveSessionStatus = "processing"
	ActiveSessionStatusEnding     ActiveSessionStatus = "ending"
	ActiveSessionStatusEnded      ActiveSessionStatus = "ended"
)

/**
 * CONTEXT:   Active session entity for real-time session tracking and correlation
 * INPUT:     Session context with terminal/project information for matching
 * OUTPUT:    Active session with context-based identification and state management
 * BUSINESS:  Correlate hook start/end events using terminal and project context
 * CHANGE:    Initial implementation of context-based session correlation
 * RISK:      Medium - Session correlation accuracy critical for work tracking
 */
type ActiveSession struct {
	id               string
	sessionContext   SessionContext
	startTime        time.Time
	estimatedEndTime time.Time
	lastActivity     time.Time
	status           ActiveSessionStatus
	activityCount    int64
	createdAt        time.Time
	updatedAt        time.Time

	// Processing metrics (optional)
	processingDuration time.Duration
	tokenCount         int64
	estimatedTokens    int64
}

// ActiveSessionConfig holds configuration for creating new active sessions
type ActiveSessionConfig struct {
	SessionContext     SessionContext
	EstimatedDuration  time.Duration
	EstimatedTokens    int64
}

/**
 * CONTEXT:   Factory method for creating new active session with context validation
 * INPUT:     ActiveSessionConfig with session context and estimation parameters
 * OUTPUT:    Valid ActiveSession entity or validation error
 * BUSINESS:  Create active sessions for daemon-managed correlation without ID passing
 * CHANGE:    Initial implementation with comprehensive context validation
 * RISK:      Medium - Context validation critical for accurate session matching
 */
func NewActiveSession(config ActiveSessionConfig) (*ActiveSession, error) {
	// Validate session context
	if err := validateSessionContext(config.SessionContext); err != nil {
		return nil, fmt.Errorf("invalid session context: %w", err)
	}

	// Set default estimated duration if not provided
	estimatedDuration := config.EstimatedDuration
	if estimatedDuration == 0 {
		estimatedDuration = 5 * time.Minute // Default Claude processing time
	}

	now := time.Now()
	sessionID := uuid.New().String()

	session := &ActiveSession{
		id:               sessionID,
		sessionContext:   config.SessionContext,
		startTime:        config.SessionContext.Timestamp,
		estimatedEndTime: config.SessionContext.Timestamp.Add(estimatedDuration),
		lastActivity:     config.SessionContext.Timestamp,
		status:           ActiveSessionStatusActive,
		activityCount:    1,
		estimatedTokens:  config.EstimatedTokens,
		createdAt:        now,
		updatedAt:        now,
	}

	return session, nil
}

func validateSessionContext(ctx SessionContext) error {
	if ctx.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if ctx.WorkingDir == "" {
		return fmt.Errorf("working directory cannot be empty")
	}

	if ctx.TerminalPID <= 0 {
		return fmt.Errorf("terminal PID must be positive, got %d", ctx.TerminalPID)
	}

	if ctx.ShellPID <= 0 {
		return fmt.Errorf("shell PID must be positive, got %d", ctx.ShellPID)
	}

	if ctx.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}

	// Validate timestamp is reasonable
	maxFutureTime := time.Now().Add(1 * time.Hour)
	if ctx.Timestamp.After(maxFutureTime) {
		return fmt.Errorf("timestamp cannot be more than 1 hour in the future")
	}

	minPastTime := time.Now().Add(-24 * time.Hour)
	if ctx.Timestamp.Before(minPastTime) {
		return fmt.Errorf("timestamp cannot be more than 24 hours in the past")
	}

	return nil
}

// Getter methods (immutable access)
func (as *ActiveSession) ID() string                          { return as.id }
func (as *ActiveSession) SessionContext() SessionContext      { return as.sessionContext }
func (as *ActiveSession) StartTime() time.Time                { return as.startTime }
func (as *ActiveSession) EstimatedEndTime() time.Time         { return as.estimatedEndTime }
func (as *ActiveSession) LastActivity() time.Time             { return as.lastActivity }
func (as *ActiveSession) Status() ActiveSessionStatus         { return as.status }
func (as *ActiveSession) ActivityCount() int64                { return as.activityCount }
func (as *ActiveSession) ProcessingDuration() time.Duration   { return as.processingDuration }
func (as *ActiveSession) TokenCount() int64                   { return as.tokenCount }
func (as *ActiveSession) EstimatedTokens() int64              { return as.estimatedTokens }
func (as *ActiveSession) CreatedAt() time.Time                { return as.createdAt }
func (as *ActiveSession) UpdatedAt() time.Time                { return as.updatedAt }

/**
 * CONTEXT:   Check if this active session matches the provided context for correlation
 * INPUT:     SessionContext to compare against this session's context
 * OUTPUT:    Match score (0.0 to 1.0) indicating correlation confidence
 * BUSINESS:  Higher scores for exact terminal matches, lower for project-only matches
 * CHANGE:    Initial context matching algorithm for session correlation
 * RISK:      High - Matching accuracy directly affects correlation reliability
 */
func (as *ActiveSession) ContextMatchScore(ctx SessionContext) float64 {
	var score float64

	// Exact terminal PID match (highest confidence)
	if as.sessionContext.TerminalPID == ctx.TerminalPID &&
		as.sessionContext.UserID == ctx.UserID {
		score += 0.6 // Base score for terminal match
		
		// Bonus for shell PID match
		if as.sessionContext.ShellPID == ctx.ShellPID {
			score += 0.2
		}
		
		// Bonus for exact directory match
		if as.sessionContext.WorkingDir == ctx.WorkingDir {
			score += 0.1
		}
		
		// Bonus for project path match
		if as.sessionContext.ProjectPath == ctx.ProjectPath {
			score += 0.1
		}
	} else if as.sessionContext.WorkingDir == ctx.WorkingDir &&
		as.sessionContext.UserID == ctx.UserID {
		// Project-only match (lower confidence)
		score += 0.4
		
		// Bonus for project path match
		if as.sessionContext.ProjectPath == ctx.ProjectPath {
			score += 0.2
		}
	} else if as.sessionContext.UserID == ctx.UserID {
		// User-only match (lowest confidence)
		score += 0.1
	}

	return score
}

/**
 * CONTEXT:   Check if session matches timing expectations for end event correlation
 * INPUT:     End event timestamp to validate against session timing
 * OUTPUT:    Boolean indicating if timing is reasonable for this session
 * BUSINESS:  End events should occur close to estimated end time
 * CHANGE:    Initial timing validation for end event correlation
 * RISK:      Medium - Timing validation prevents incorrect session ending
 */
func (as *ActiveSession) IsTimingReasonableForEnd(endTime time.Time) bool {
	// Check if end time is after start time
	if endTime.Before(as.startTime) {
		return false
	}

	// Check if session has been running for reasonable time (at least 10 seconds)
	minRunTime := 10 * time.Second
	if endTime.Sub(as.startTime) < minRunTime {
		return false
	}

	// Check if session hasn't been running too long (max 30 minutes)
	maxRunTime := 30 * time.Minute
	if endTime.Sub(as.startTime) > maxRunTime {
		return false
	}

	return true
}

/**
 * CONTEXT:   Update session activity with new timestamp
 * INPUT:     Activity timestamp for session update
 * OUTPUT:    Error if activity is invalid, nil on success
 * BUSINESS:  Keep session active with latest activity time
 * CHANGE:    Initial activity update implementation
 * RISK:      Low - Simple state update with validation
 */
func (as *ActiveSession) UpdateActivity(activityTime time.Time) error {
	// Validate activity time is not before start
	if activityTime.Before(as.startTime) {
		return fmt.Errorf("activity time %v cannot be before session start %v",
			activityTime, as.startTime)
	}

	// Validate activity time is reasonable
	maxFutureTime := time.Now().Add(5 * time.Minute)
	if activityTime.After(maxFutureTime) {
		return fmt.Errorf("activity time %v is too far in the future", activityTime)
	}

	// Update session state
	as.lastActivity = activityTime
	as.activityCount++
	as.updatedAt = time.Now()

	return nil
}

/**
 * CONTEXT:   Mark session as processing (between start and end hooks)
 * INPUT:     Processing start timestamp
 * OUTPUT:    Error if session cannot transition to processing, nil on success
 * BUSINESS:  Session enters processing state after start hook, before end hook
 * CHANGE:    Initial processing state transition logic
 * RISK:      Low - State transition with validation
 */
func (as *ActiveSession) StartProcessing(timestamp time.Time) error {
	if as.status != ActiveSessionStatusActive {
		return fmt.Errorf("cannot start processing on session with status %s", as.status)
	}

	as.status = ActiveSessionStatusProcessing
	as.lastActivity = timestamp
	as.updatedAt = time.Now()

	return nil
}

/**
 * CONTEXT:   End session with processing metrics and duration
 * INPUT:     End timestamp, processing duration, and token count
 * OUTPUT:    Error if session cannot be ended, nil on success
 * BUSINESS:  Session completes with actual processing metrics recorded
 * CHANGE:    Initial session ending with metrics capture
 * RISK:      Low - Final state transition with metrics
 */
func (as *ActiveSession) EndSession(endTime time.Time, processingDuration time.Duration, tokenCount int64) error {
	if as.status != ActiveSessionStatusActive && as.status != ActiveSessionStatusProcessing {
		return fmt.Errorf("cannot end session with status %s", as.status)
	}

	// Validate end time
	if !as.IsTimingReasonableForEnd(endTime) {
		return fmt.Errorf("end time %v is not reasonable for session started at %v",
			endTime, as.startTime)
	}

	as.status = ActiveSessionStatusEnded
	as.lastActivity = endTime
	as.processingDuration = processingDuration
	as.tokenCount = tokenCount
	as.updatedAt = time.Now()

	return nil
}

/**
 * CONTEXT:   Export active session data for serialization and persistence
 * INPUT:     No parameters, uses internal session state
 * OUTPUT:    ActiveSessionData struct suitable for JSON serialization and database storage
 * BUSINESS:  Provide read-only view of active session data for external use
 * CHANGE:    Initial data export implementation for persistence layer
 * RISK:      Low - Read-only operation with no state changes
 */
type ActiveSessionData struct {
	ID                 string                `json:"id"`
	TerminalPID        int                   `json:"terminal_pid"`
	ShellPID           int                   `json:"shell_pid"`
	WorkingDir         string                `json:"working_dir"`
	ProjectPath        string                `json:"project_path"`
	UserID             string                `json:"user_id"`
	StartTime          time.Time             `json:"start_time"`
	EstimatedEndTime   time.Time             `json:"estimated_end_time"`
	LastActivity       time.Time             `json:"last_activity"`
	Status             ActiveSessionStatus   `json:"status"`
	ActivityCount      int64                 `json:"activity_count"`
	ProcessingDuration time.Duration         `json:"processing_duration"`
	TokenCount         int64                 `json:"token_count"`
	EstimatedTokens    int64                 `json:"estimated_tokens"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
}

func (as *ActiveSession) ToData() ActiveSessionData {
	return ActiveSessionData{
		ID:                 as.id,
		TerminalPID:        as.sessionContext.TerminalPID,
		ShellPID:           as.sessionContext.ShellPID,
		WorkingDir:         as.sessionContext.WorkingDir,
		ProjectPath:        as.sessionContext.ProjectPath,
		UserID:             as.sessionContext.UserID,
		StartTime:          as.startTime,
		EstimatedEndTime:   as.estimatedEndTime,
		LastActivity:       as.lastActivity,
		Status:             as.status,
		ActivityCount:      as.activityCount,
		ProcessingDuration: as.processingDuration,
		TokenCount:         as.tokenCount,
		EstimatedTokens:    as.estimatedTokens,
		CreatedAt:          as.createdAt,
		UpdatedAt:          as.updatedAt,
	}
}

/**
 * CONTEXT:   Validate active session internal state consistency
 * INPUT:     No parameters, validates internal state
 * OUTPUT:    Error if state is inconsistent, nil if valid
 * BUSINESS:  Ensure active session data integrity and business rule compliance
 * CHANGE:    Initial state validation implementation
 * RISK:      Low - Validation only, no state changes
 */
func (as *ActiveSession) Validate() error {
	// Check required fields
	if as.id == "" {
		return fmt.Errorf("active session ID cannot be empty")
	}

	if err := validateSessionContext(as.sessionContext); err != nil {
		return fmt.Errorf("invalid session context: %w", err)
	}

	// Check time relationships
	if as.lastActivity.Before(as.startTime) {
		return fmt.Errorf("last activity time cannot be before start time")
	}

	if as.estimatedEndTime.Before(as.startTime) {
		return fmt.Errorf("estimated end time cannot be before start time")
	}

	// Check activity count
	if as.activityCount < 1 {
		return fmt.Errorf("active session must have at least 1 activity")
	}

	// Check processing duration consistency
	if as.status == ActiveSessionStatusEnded && as.processingDuration < 0 {
		return fmt.Errorf("ended session cannot have negative processing duration")
	}

	return nil
}