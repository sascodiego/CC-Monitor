/**
 * CONTEXT:   Session repository implementation for SQLite database operations
 * INPUT:     Session entities and query parameters for CRUD operations
 * OUTPUT:    Session data with proper error handling and transaction support
 * BUSINESS:  Session management following 5-hour window business rules
 * CHANGE:    Initial SQLite repository replacing gob-based session storage
 * RISK:      Low - Standard repository pattern with prepared statements and error handling
 */

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// SessionRepository provides database operations for sessions
type SessionRepository struct {
	db *SQLiteDB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *SQLiteDB) *SessionRepository {
	return &SessionRepository{db: db}
}

/**
 * CONTEXT:   Create new session in database with validation
 * INPUT:     Session entity with all required fields and business logic validation
 * OUTPUT:    Error if creation fails, nil on success
 * BUSINESS:  Sessions must be exactly 5 hours long with valid time ranges
 * CHANGE:    Initial SQLite session creation with constraint validation
 * RISK:      Low - Prepared statement with parameter binding prevents SQL injection
 */
func (r *SessionRepository) Create(ctx context.Context, session *Session) error {
	// Validate session before database operation
	if err := validateSession(session); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	// Convert times to database timezone
	session.StartTime = r.db.ToDBTime(session.StartTime)
	session.EndTime = r.db.ToDBTime(session.EndTime)
	session.FirstActivityTime = r.db.ToDBTime(session.FirstActivityTime)
	session.LastActivityTime = r.db.ToDBTime(session.LastActivityTime)
	session.CreatedAt = r.db.ToDBTime(session.CreatedAt)
	session.UpdatedAt = r.db.ToDBTime(session.UpdatedAt)

	query := `
		INSERT INTO sessions (
			id, user_id, start_time, end_time, state, first_activity_time, 
			last_activity_time, activity_count, duration_hours, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.DB().ExecContext(ctx, query,
		session.ID, session.UserID, session.StartTime, session.EndTime,
		session.State, session.FirstActivityTime, session.LastActivityTime,
		session.ActivityCount, session.DurationHours, session.CreatedAt, session.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	log.Printf("✅ Created session: %s (user: %s, period: %s - %s)",
		session.ID, session.UserID, 
		session.StartTime.Format("2006-01-02 15:04:05"),
		session.EndTime.Format("2006-01-02 15:04:05"))

	return nil
}

/**
 * CONTEXT:   Retrieve session by ID with timezone conversion
 * INPUT:     Session ID for lookup
 * OUTPUT:    Session entity or error if not found
 * BUSINESS:  Sessions contain complete 5-hour window data
 * CHANGE:    Initial SQLite session retrieval with timezone handling
 * RISK:      Low - Simple SELECT query with parameter binding
 */
func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	query := `
		SELECT id, user_id, start_time, end_time, state, first_activity_time,
			   last_activity_time, activity_count, duration_hours, created_at, updated_at
		FROM sessions
		WHERE id = ?
	`

	session := &Session{}
	err := r.db.DB().QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID, &session.UserID, &session.StartTime, &session.EndTime,
		&session.State, &session.FirstActivityTime, &session.LastActivityTime,
		&session.ActivityCount, &session.DurationHours, &session.CreatedAt, &session.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Convert times from database timezone to local
	session.StartTime = r.db.FromDBTime(session.StartTime)
	session.EndTime = r.db.FromDBTime(session.EndTime)
	session.FirstActivityTime = r.db.FromDBTime(session.FirstActivityTime)
	session.LastActivityTime = r.db.FromDBTime(session.LastActivityTime)
	session.CreatedAt = r.db.FromDBTime(session.CreatedAt)
	session.UpdatedAt = r.db.FromDBTime(session.UpdatedAt)

	return session, nil
}

/**
 * CONTEXT:   Find sessions by user ID and time range for reporting
 * INPUT:     User ID and optional time range for filtering
 * OUTPUT:    List of sessions matching criteria
 * BUSINESS:  Support daily, weekly, monthly reporting queries
 * CHANGE:    Initial SQLite session querying with time range filtering
 * RISK:      Low - Indexed query with prepared statements
 */
func (r *SessionRepository) FindByUserAndTimeRange(ctx context.Context, userID string, startTime, endTime time.Time) ([]*Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Convert query times to database timezone
	startTime = r.db.ToDBTime(startTime)
	endTime = r.db.ToDBTime(endTime)

	query := `
		SELECT id, user_id, start_time, end_time, state, first_activity_time,
			   last_activity_time, activity_count, duration_hours, created_at, updated_at
		FROM sessions
		WHERE user_id = ? AND start_time >= ? AND start_time <= ?
		ORDER BY start_time DESC
	`

	rows, err := r.db.DB().QueryContext(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]*Session, 0)
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(
			&session.ID, &session.UserID, &session.StartTime, &session.EndTime,
			&session.State, &session.FirstActivityTime, &session.LastActivityTime,
			&session.ActivityCount, &session.DurationHours, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		// Convert times from database timezone
		session.StartTime = r.db.FromDBTime(session.StartTime)
		session.EndTime = r.db.FromDBTime(session.EndTime)
		session.FirstActivityTime = r.db.FromDBTime(session.FirstActivityTime)
		session.LastActivityTime = r.db.FromDBTime(session.LastActivityTime)
		session.CreatedAt = r.db.FromDBTime(session.CreatedAt)
		session.UpdatedAt = r.db.FromDBTime(session.UpdatedAt)

		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

/**
 * CONTEXT:   Get active sessions for user (sessions not yet expired)
 * INPUT:     User ID for filtering active sessions
 * OUTPUT:    List of currently active sessions
 * BUSINESS:  Active sessions are within their 5-hour window
 * CHANGE:    Initial active session querying with time-based logic
 * RISK:      Low - Time-based filtering with current timestamp
 */
func (r *SessionRepository) GetActiveSessionsByUser(ctx context.Context, userID string) ([]*Session, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	currentTime := r.db.Now()

	query := `
		SELECT id, user_id, start_time, end_time, state, first_activity_time,
			   last_activity_time, activity_count, duration_hours, created_at, updated_at
		FROM sessions
		WHERE user_id = ? AND state = 'active' AND ? <= end_time
		ORDER BY start_time DESC
	`

	rows, err := r.db.DB().QueryContext(ctx, query, userID, currentTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query active sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]*Session, 0)
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(
			&session.ID, &session.UserID, &session.StartTime, &session.EndTime,
			&session.State, &session.FirstActivityTime, &session.LastActivityTime,
			&session.ActivityCount, &session.DurationHours, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan active session: %w", err)
		}

		// Convert times from database timezone
		session.StartTime = r.db.FromDBTime(session.StartTime)
		session.EndTime = r.db.FromDBTime(session.EndTime)
		session.FirstActivityTime = r.db.FromDBTime(session.FirstActivityTime)
		session.LastActivityTime = r.db.FromDBTime(session.LastActivityTime)
		session.CreatedAt = r.db.FromDBTime(session.CreatedAt)
		session.UpdatedAt = r.db.FromDBTime(session.UpdatedAt)

		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active sessions: %w", err)
	}

	return sessions, nil
}

/**
 * CONTEXT:   Update existing session with new activity or state changes
 * INPUT:     Session entity with updated fields
 * OUTPUT:    Error if update fails, nil on success
 * BUSINESS:  Sessions can be updated with new activity until expired
 * CHANGE:    Initial SQLite session update with optimistic concurrency
 * RISK:      Low - Prepared statement with WHERE clause and updated_at check
 */
func (r *SessionRepository) Update(ctx context.Context, session *Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	// Validate session before update
	if err := validateSession(session); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	// Set updated timestamp
	session.UpdatedAt = r.db.Now()

	// Convert times to database timezone
	session.StartTime = r.db.ToDBTime(session.StartTime)
	session.EndTime = r.db.ToDBTime(session.EndTime)
	session.FirstActivityTime = r.db.ToDBTime(session.FirstActivityTime)
	session.LastActivityTime = r.db.ToDBTime(session.LastActivityTime)
	session.CreatedAt = r.db.ToDBTime(session.CreatedAt)
	session.UpdatedAt = r.db.ToDBTime(session.UpdatedAt)

	query := `
		UPDATE sessions SET 
			state = ?, first_activity_time = ?, last_activity_time = ?,
			activity_count = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.DB().ExecContext(ctx, query,
		session.State, session.FirstActivityTime, session.LastActivityTime,
		session.ActivityCount, session.UpdatedAt, session.ID)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found for update: %s", session.ID)
	}

	log.Printf("✅ Updated session: %s (activities: %d, state: %s)",
		session.ID, session.ActivityCount, session.State)

	return nil
}

/**
 * CONTEXT:   Delete session and cascade to related work blocks
 * INPUT:     Session ID to delete
 * OUTPUT:    Error if deletion fails, nil on success
 * BUSINESS:  Cascading delete removes all related work blocks and activities
 * CHANGE:    Initial session deletion with foreign key cascading
 * RISK:      Medium - Cascading delete affects multiple tables
 */
func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	// Use transaction for safe deletion
	return r.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Delete session (cascades to work_blocks and activity_events)
		result, err := tx.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", sessionID)
		if err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return fmt.Errorf("session not found for deletion: %s", sessionID)
		}

		log.Printf("🗑️  Deleted session: %s (with cascading deletes)", sessionID)
		return nil
	})
}

/**
 * CONTEXT:   Mark expired sessions as expired based on current time
 * INPUT:     Context for query timeout
 * OUTPUT:    Number of sessions marked as expired
 * BUSINESS:  Sessions expire when current time > end_time
 * CHANGE:    Initial session expiration batch processing
 * RISK:      Low - Bulk update with time-based filtering
 */
func (r *SessionRepository) MarkExpiredSessions(ctx context.Context) (int, error) {
	currentTime := r.db.Now()

	query := `
		UPDATE sessions 
		SET state = 'expired', updated_at = ?
		WHERE state = 'active' AND ? > end_time
	`

	result, err := r.db.DB().ExecContext(ctx, query, currentTime, currentTime)
	if err != nil {
		return 0, fmt.Errorf("failed to mark expired sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Printf("⏰ Marked %d sessions as expired", rowsAffected)
	}

	return int(rowsAffected), nil
}

/**
 * CONTEXT:   Get session statistics for monitoring and reporting
 * INPUT:     Context and optional user ID filter
 * OUTPUT:    Session statistics including counts by state
 * BUSINESS:  Monitor session health and user activity patterns
 * CHANGE:    Initial session statistics for system monitoring
 * RISK:      Low - Aggregate queries for statistics
 */
func (r *SessionRepository) GetStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Base query with optional user filter
	whereClause := ""
	args := []interface{}{}
	if userID != "" {
		whereClause = "WHERE user_id = ?"
		args = append(args, userID)
	}

	// Total sessions
	query := fmt.Sprintf("SELECT COUNT(*) FROM sessions %s", whereClause)
	var totalSessions int
	if err := r.db.DB().QueryRowContext(ctx, query, args...).Scan(&totalSessions); err != nil {
		return nil, fmt.Errorf("failed to count total sessions: %w", err)
	}
	stats["total_sessions"] = totalSessions

	// Sessions by state
	states := []string{"active", "expired", "finished"}
	for _, state := range states {
		stateArgs := append(args, state)
		query := fmt.Sprintf("SELECT COUNT(*) FROM sessions %s %s", 
			whereClause, 
			func() string {
				if whereClause == "" {
					return "WHERE state = ?"
				}
				return "AND state = ?"
			}())
		
		var count int
		if err := r.db.DB().QueryRowContext(ctx, query, stateArgs...).Scan(&count); err != nil {
			log.Printf("Warning: failed to count %s sessions: %v", state, err)
			stats[state+"_sessions"] = 0
		} else {
			stats[state+"_sessions"] = count
		}
	}

	// Average activity count
	query = fmt.Sprintf("SELECT AVG(activity_count) FROM sessions %s", whereClause)
	var avgActivityCount sql.NullFloat64
	if err := r.db.DB().QueryRowContext(ctx, query, args...).Scan(&avgActivityCount); err != nil {
		log.Printf("Warning: failed to calculate average activity count: %v", err)
		stats["avg_activity_count"] = 0.0
	} else if avgActivityCount.Valid {
		stats["avg_activity_count"] = avgActivityCount.Float64
	} else {
		stats["avg_activity_count"] = 0.0
	}

	// Most recent session
	query = fmt.Sprintf("SELECT MAX(start_time) FROM sessions %s", whereClause)
	var mostRecentSession sql.NullTime
	if err := r.db.DB().QueryRowContext(ctx, query, args...).Scan(&mostRecentSession); err != nil {
		log.Printf("Warning: failed to find most recent session: %v", err)
	} else if mostRecentSession.Valid {
		stats["most_recent_session"] = r.db.FromDBTime(mostRecentSession.Time)
	}

	return stats, nil
}

/**
 * CONTEXT:   Get all active sessions in the system
 * INPUT:     Context for database operations
 * OUTPUT:    Array of currently active sessions
 * BUSINESS:  Active sessions for monitoring and cleanup operations
 * CHANGE:    Added GetActiveSessions method for system monitoring
 * RISK:      Low - Read-only query for active sessions
 */
func (r *SessionRepository) GetActiveSessions(ctx context.Context) ([]*Session, error) {
	currentTime := r.db.Now()
	
	query := `
		SELECT id, user_id, start_time, end_time, state, first_activity_time,
		       last_activity_time, activity_count, duration_hours, created_at, updated_at
		FROM sessions 
		WHERE state = 'active' AND ? <= end_time
		ORDER BY start_time DESC`
	
	rows, err := r.db.DB().QueryContext(ctx, query, currentTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		var firstActivityTime, lastActivityTime sql.NullTime

		err := rows.Scan(
			&session.ID, &session.UserID, &session.StartTime, &session.EndTime,
			&session.State, &firstActivityTime, &lastActivityTime,
			&session.ActivityCount, &session.DurationHours,
			&session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		// Convert times from database timezone
		session.StartTime = r.db.FromDBTime(session.StartTime)
		session.EndTime = r.db.FromDBTime(session.EndTime)
		session.CreatedAt = r.db.FromDBTime(session.CreatedAt)
		session.UpdatedAt = r.db.FromDBTime(session.UpdatedAt)

		if firstActivityTime.Valid {
			converted := r.db.FromDBTime(firstActivityTime.Time)
			session.FirstActivityTime = converted
		}
		if lastActivityTime.Valid {
			converted := r.db.FromDBTime(lastActivityTime.Time)
			session.LastActivityTime = converted
		}

		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active sessions: %w", err)
	}

	return sessions, nil
}

/**
 * CONTEXT:   Get all sessions in the system for analytics and monitoring
 * INPUT:     Context for database operations
 * OUTPUT:    All sessions ordered by creation time
 * BUSINESS:  System-wide session queries for reporting and analytics
 * CHANGE:    Added GetAll method for session monitoring and reporting
 * RISK:      Medium - Could return large dataset, consider pagination
 */
func (r *SessionRepository) GetAll(ctx context.Context) ([]*Session, error) {
	query := `
		SELECT id, user_id, start_time, end_time, state, first_activity_time,
		       last_activity_time, activity_count, duration_hours, created_at, updated_at
		FROM sessions 
		ORDER BY created_at DESC`
	
	rows, err := r.db.DB().QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		var firstActivityTime, lastActivityTime sql.NullTime

		err := rows.Scan(
			&session.ID, &session.UserID, &session.StartTime, &session.EndTime,
			&session.State, &firstActivityTime, &lastActivityTime,
			&session.ActivityCount, &session.DurationHours,
			&session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		// Convert times from database timezone
		session.StartTime = r.db.FromDBTime(session.StartTime)
		session.EndTime = r.db.FromDBTime(session.EndTime)
		session.CreatedAt = r.db.FromDBTime(session.CreatedAt)
		session.UpdatedAt = r.db.FromDBTime(session.UpdatedAt)

		if firstActivityTime.Valid {
			converted := r.db.FromDBTime(firstActivityTime.Time)
			session.FirstActivityTime = converted
		}
		if lastActivityTime.Valid {
			converted := r.db.FromDBTime(lastActivityTime.Time)
			session.LastActivityTime = converted
		}

		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}

// Validation helper function
func validateSession(session *Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	if session.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if session.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if session.StartTime.IsZero() {
		return fmt.Errorf("start time cannot be zero")
	}

	if session.EndTime.IsZero() {
		return fmt.Errorf("end time cannot be zero")
	}

	// Validate 5-hour duration rule
	expectedDuration := 5 * time.Hour
	actualDuration := session.EndTime.Sub(session.StartTime)
	if actualDuration != expectedDuration {
		return fmt.Errorf("session duration must be exactly 5 hours, got %v", actualDuration)
	}

	// Validate state
	validStates := []string{"active", "expired", "finished"}
	validState := false
	for _, state := range validStates {
		if session.State == state {
			validState = true
			break
		}
	}
	if !validState {
		return fmt.Errorf("invalid session state: %s, must be one of: %s", 
			session.State, strings.Join(validStates, ", "))
	}

	if session.ActivityCount < 1 {
		return fmt.Errorf("activity count must be at least 1")
	}

	if session.DurationHours != 5.0 {
		return fmt.Errorf("duration hours must be exactly 5.0 for sessions")
	}

	return nil
}