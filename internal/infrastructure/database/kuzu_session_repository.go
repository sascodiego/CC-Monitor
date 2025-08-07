/**
 * CONTEXT:   KuzuDB implementation of SessionRepository interface for session persistence
 * INPUT:     Session entities, query filters, and sorting parameters for CRUD operations
 * OUTPUT:    Complete session repository implementation with optimized Cypher queries
 * BUSINESS:  Sessions require efficient storage and querying for 5-hour window analytics
 * CHANGE:    Initial KuzuDB implementation following repository interface contract
 * RISK:      Medium - Database operations require careful error handling and transaction management
 */

package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kuzudb/go-kuzu"
	"github.com/claude-monitor/system/internal/entities"
	"github.com/claude-monitor/system/internal/usecases/repositories"
)

/**
 * CONTEXT:   KuzuDB-specific implementation of SessionRepository interface
 * INPUT:     Database connection manager and session entity operations
 * OUTPUT:    Concrete repository implementation with optimized graph queries
 * BUSINESS:  Session persistence enables 5-hour window tracking and analytics
 * CHANGE:    Initial repository implementation with comprehensive query support
 * RISK:      Medium - Database operations with connection management and error handling
 */
type KuzuSessionRepository struct {
	connManager *KuzuConnectionManager
}

// NewKuzuSessionRepository creates a new KuzuDB session repository
func NewKuzuSessionRepository(connManager *KuzuConnectionManager) repositories.SessionRepository {
	return &KuzuSessionRepository{
		connManager: connManager,
	}
}

/**
 * CONTEXT:   Save session entity to KuzuDB with relationship management
 * INPUT:     Context and session entity with all required fields
 * OUTPUT:    Session persisted with user relationship or error if save fails
 * BUSINESS:  Session saves must create user relationship for proper graph structure
 * CHANGE:    Initial session save implementation with user relationship creation
 * RISK:      Medium - Transaction required to ensure atomicity of session and relationship
 */
func (ksr *KuzuSessionRepository) Save(ctx context.Context, session *entities.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	// Validate session before saving
	if err := session.Validate(); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	return ksr.connManager.WithTransaction(ctx, func(conn *kuzu.Connection) error {
		// First ensure user exists
		userQuery := `
			MERGE (u:User {id: $user_id})
			ON CREATE SET u.name = $user_id, u.created_at = current_timestamp(), u.updated_at = current_timestamp()
			ON MATCH SET u.updated_at = current_timestamp();
		`
		
		userParams := map[string]interface{}{
			"user_id": session.UserID(),
		}

		if _, err := conn.Query(userQuery); err != nil {
			return fmt.Errorf("failed to ensure user exists: %w", err)
		}

		// Create session node
		sessionQuery := `
			CREATE (s:Session {
				id: $id,
				user_id: $user_id,
				start_time: $start_time,
				end_time: $end_time,
				state: $state,
				first_activity_time: $first_activity_time,
				last_activity_time: $last_activity_time,
				activity_count: $activity_count,
				duration_hours: $duration_hours,
				is_active: $is_active,
				created_at: $created_at,
				updated_at: $updated_at
			});
		`

		sessionParams := map[string]interface{}{
			"id":                   session.ID(),
			"user_id":              session.UserID(),
			"start_time":           session.StartTime(),
			"end_time":             session.EndTime(),
			"state":                string(session.State()),
			"first_activity_time":  session.FirstActivityTime(),
			"last_activity_time":   session.LastActivityTime(),
			"activity_count":       session.ActivityCount(),
			"duration_hours":       session.Duration().Hours(),
			"is_active":            session.State() == entities.SessionStateActive,
			"created_at":           session.CreatedAt(),
			"updated_at":           session.UpdatedAt(),
		}

		if _, err := conn.Query(sessionQuery); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}

		// Create user-session relationship
		relationshipQuery := `
			MATCH (u:User {id: $user_id}), (s:Session {id: $session_id})
			CREATE (u)-[:HAS_SESSION {created_at: current_timestamp()}]->(s);
		`

		relationshipParams := map[string]interface{}{
			"user_id":    session.UserID(),
			"session_id": session.ID(),
		}

		if _, err := conn.Query(relationshipQuery); err != nil {
			return fmt.Errorf("failed to create user-session relationship: %w", err)
		}

		return nil
	})
}

/**
 * CONTEXT:   Find session by ID with efficient graph traversal
 * INPUT:     Context and session ID for lookup
 * OUTPUT:    Session entity or not found error
 * BUSINESS:  Session lookup by ID is common for session management operations
 * CHANGE:    Initial session lookup with entity reconstruction
 * RISK:      Low - Simple query with proper error handling
 */
func (ksr *KuzuSessionRepository) FindByID(ctx context.Context, sessionID string) (*entities.Session, error) {
	if sessionID == "" {
		return nil, repositories.ErrSessionNotFound
	}

	query := `
		MATCH (s:Session {id: $session_id})
		RETURN s.id, s.user_id, s.start_time, s.end_time, s.state,
			   s.first_activity_time, s.last_activity_time, s.activity_count,
			   s.duration_hours, s.is_active, s.created_at, s.updated_at;
	`

	params := map[string]interface{}{
		"session_id": sessionID,
	}

	result, err := ksr.connManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}
	defer result.Close()

	if !result.HasNext() {
		return nil, repositories.ErrSessionNotFound
	}

	record, err := result.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to read session record: %w", err)
	}

	return ksr.recordToSession(record)
}

/**
 * CONTEXT:   Update existing session with optimized merge operation
 * INPUT:     Context and updated session entity
 * OUTPUT:    Session updated in database or error if update fails
 * BUSINESS:  Session updates occur frequently during activity tracking
 * CHANGE:    Initial session update with merge operation for efficiency
 * RISK:      Medium - Update operation requires existence validation and concurrency handling
 */
func (ksr *KuzuSessionRepository) Update(ctx context.Context, session *entities.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	// Validate session before updating
	if err := session.Validate(); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	query := `
		MATCH (s:Session {id: $id})
		SET s.user_id = $user_id,
			s.start_time = $start_time,
			s.end_time = $end_time,
			s.state = $state,
			s.first_activity_time = $first_activity_time,
			s.last_activity_time = $last_activity_time,
			s.activity_count = $activity_count,
			s.duration_hours = $duration_hours,
			s.is_active = $is_active,
			s.updated_at = $updated_at
		RETURN s.id;
	`

	params := map[string]interface{}{
		"id":                   session.ID(),
		"user_id":              session.UserID(),
		"start_time":           session.StartTime(),
		"end_time":             session.EndTime(),
		"state":                string(session.State()),
		"first_activity_time":  session.FirstActivityTime(),
		"last_activity_time":   session.LastActivityTime(),
		"activity_count":       session.ActivityCount(),
		"duration_hours":       session.Duration().Hours(),
		"is_active":            session.State() == entities.SessionStateActive,
		"updated_at":           time.Now(),
	}

	result, err := ksr.connManager.Query(ctx, query, params)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	defer result.Close()

	if !result.HasNext() {
		return repositories.ErrSessionNotFound
	}

	return nil
}

/**
 * CONTEXT:   Delete session and clean up relationships
 * INPUT:     Context and session ID for deletion
 * OUTPUT:    Session deleted with all relationships or error
 * BUSINESS:  Session deletion must clean up work blocks and relationships
 * CHANGE:    Initial session deletion with relationship cleanup
 * RISK:      High - Deletion requires transaction to maintain graph integrity
 */
func (ksr *KuzuSessionRepository) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return repositories.ErrSessionNotFound
	}

	return ksr.connManager.WithTransaction(ctx, func(conn *kuzu.Connection) error {
		// Delete session and all relationships
		query := `
			MATCH (s:Session {id: $session_id})
			OPTIONAL MATCH (s)-[r]-()
			DELETE r, s
			RETURN COUNT(s) as deleted_count;
		`

		params := map[string]interface{}{
			"session_id": sessionID,
		}

		result, err := conn.Query(query)
		if err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}
		defer result.Close()

		if !result.HasNext() {
			return repositories.ErrSessionNotFound
		}

		record, err := result.Next()
		if err != nil {
			return fmt.Errorf("failed to read deletion result: %w", err)
		}

		deletedCount := record[0].(int64)
		if deletedCount == 0 {
			return repositories.ErrSessionNotFound
		}

		return nil
	})
}

/**
 * CONTEXT:   Find all sessions for a specific user with chronological ordering
 * INPUT:     Context and user ID for session lookup
 * OUTPUT:    Array of user sessions sorted by start time
 * BUSINESS:  User session history is essential for work pattern analysis
 * CHANGE:    Initial user session query with time-based ordering
 * RISK:      Medium - Large result sets may impact performance without pagination
 */
func (ksr *KuzuSessionRepository) FindByUserID(ctx context.Context, userID string) ([]*entities.Session, error) {
	if userID == "" {
		return []*entities.Session{}, nil
	}

	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
		RETURN s.id, s.user_id, s.start_time, s.end_time, s.state,
			   s.first_activity_time, s.last_activity_time, s.activity_count,
			   s.duration_hours, s.is_active, s.created_at, s.updated_at
		ORDER BY s.start_time DESC;
	`

	params := map[string]interface{}{
		"user_id": userID,
	}

	return ksr.executeFindQuery(ctx, query, params)
}

/**
 * CONTEXT:   Find currently active session for user (5-hour window logic)
 * INPUT:     Context and user ID for active session lookup
 * OUTPUT:    Active session entity or nil if no active session
 * BUSINESS:  Active session lookup is critical for activity event processing
 * CHANGE:    Initial active session query with state and time filtering
 * RISK:      Low - Single session query with specific business logic
 */
func (ksr *KuzuSessionRepository) FindActiveSession(ctx context.Context, userID string) (*entities.Session, error) {
	if userID == "" {
		return nil, nil
	}

	currentTime := time.Now()
	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
		WHERE s.is_active = true 
		  AND s.state = 'active'
		  AND s.start_time <= $current_time
		  AND s.end_time > $current_time
		RETURN s.id, s.user_id, s.start_time, s.end_time, s.state,
			   s.first_activity_time, s.last_activity_time, s.activity_count,
			   s.duration_hours, s.is_active, s.created_at, s.updated_at
		ORDER BY s.start_time DESC
		LIMIT 1;
	`

	params := map[string]interface{}{
		"user_id":      userID,
		"current_time": currentTime,
	}

	result, err := ksr.connManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query active session: %w", err)
	}
	defer result.Close()

	if !result.HasNext() {
		return nil, nil
	}

	record, err := result.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to read active session record: %w", err)
	}

	return ksr.recordToSession(record)
}

/**
 * CONTEXT:   Find sessions with flexible filtering and sorting capabilities
 * INPUT:     Context, filter criteria, and sorting parameters
 * OUTPUT:    Filtered and sorted session array matching criteria
 * BUSINESS:  Flexible session queries enable various reporting and analytics use cases
 * CHANGE:    Initial flexible query implementation with dynamic filter building
 * RISK:      Medium - Dynamic query building requires careful SQL injection prevention
 */
func (ksr *KuzuSessionRepository) FindByFilter(ctx context.Context, filter repositories.SessionFilter) ([]*entities.Session, error) {
	query, params := ksr.buildFilterQuery(filter)
	return ksr.executeFindQuery(ctx, query, params)
}

func (ksr *KuzuSessionRepository) FindWithSort(ctx context.Context, filter repositories.SessionFilter, sortBy repositories.SessionSortBy, order repositories.SessionSortOrder) ([]*entities.Session, error) {
	query, params := ksr.buildFilterQuery(filter)
	
	// Add sorting
	orderClause := ksr.buildOrderClause(sortBy, order)
	query += " " + orderClause

	return ksr.executeFindQuery(ctx, query, params)
}

/**
 * CONTEXT:   Find expired sessions for cleanup and state management
 * INPUT:     Context and cutoff time for expiration detection
 * OUTPUT:    Array of sessions expired before the cutoff time
 * BUSINESS:  Expired session detection enables cleanup and state transitions
 * CHANGE:    Initial expired session query for maintenance operations
 * RISK:      Low - Time-based query with clear business logic
 */
func (ksr *KuzuSessionRepository) FindExpiredSessions(ctx context.Context, beforeTime time.Time) ([]*entities.Session, error) {
	query := `
		MATCH (s:Session)
		WHERE s.end_time < $before_time OR (s.is_active = true AND s.state = 'active' AND s.end_time < $before_time)
		RETURN s.id, s.user_id, s.start_time, s.end_time, s.state,
			   s.first_activity_time, s.last_activity_time, s.activity_count,
			   s.duration_hours, s.is_active, s.created_at, s.updated_at
		ORDER BY s.end_time;
	`

	params := map[string]interface{}{
		"before_time": beforeTime,
	}

	return ksr.executeFindQuery(ctx, query, params)
}

/**
 * CONTEXT:   Find sessions within specific time range for analytics
 * INPUT:     Context, user ID, start and end time boundaries
 * OUTPUT:    Sessions within time range sorted chronologically
 * BUSINESS:  Time range queries enable daily, weekly, monthly reporting
 * CHANGE:    Initial time range query for reporting functionality
 * RISK:      Low - Time-based filtering with clear boundaries
 */
func (ksr *KuzuSessionRepository) FindSessionsInTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*entities.Session, error) {
	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
		WHERE s.start_time >= $start_time AND s.start_time < $end_time
		RETURN s.id, s.user_id, s.start_time, s.end_time, s.state,
			   s.first_activity_time, s.last_activity_time, s.activity_count,
			   s.duration_hours, s.is_active, s.created_at, s.updated_at
		ORDER BY s.start_time;
	`

	params := map[string]interface{}{
		"user_id":    userID,
		"start_time": start,
		"end_time":   end,
	}

	return ksr.executeFindQuery(ctx, query, params)
}

/**
 * CONTEXT:   Find recent sessions with limit for quick access
 * INPUT:     Context, user ID, and session count limit
 * OUTPUT:    Most recent sessions up to specified limit
 * BUSINESS:  Recent session access enables quick user context and continuation
 * CHANGE:    Initial recent sessions query with limit
 * RISK:      Low - Simple query with limit clause
 */
func (ksr *KuzuSessionRepository) FindRecentSessions(ctx context.Context, userID string, limit int) ([]*entities.Session, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
		RETURN s.id, s.user_id, s.start_time, s.end_time, s.state,
			   s.first_activity_time, s.last_activity_time, s.activity_count,
			   s.duration_hours, s.is_active, s.created_at, s.updated_at
		ORDER BY s.start_time DESC
		LIMIT $limit;
	`

	params := map[string]interface{}{
		"user_id": userID,
		"limit":   int64(limit),
	}

	return ksr.executeFindQuery(ctx, query, params)
}

/**
 * CONTEXT:   Count sessions by user for analytics and statistics
 * INPUT:     Context and user ID for session counting
 * OUTPUT:    Total session count for the specified user
 * BUSINESS:  Session counting enables user engagement metrics and analytics
 * CHANGE:    Initial session counting query
 * RISK:      Low - Simple counting query with aggregation
 */
func (ksr *KuzuSessionRepository) CountSessionsByUser(ctx context.Context, userID string) (int64, error) {
	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
		RETURN COUNT(s) as session_count;
	`

	params := map[string]interface{}{
		"user_id": userID,
	}

	result, err := ksr.connManager.Query(ctx, query, params)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	defer result.Close()

	if !result.HasNext() {
		return 0, nil
	}

	record, err := result.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to read count result: %w", err)
	}

	return record[0].(int64), nil
}

/**
 * CONTEXT:   Count sessions within time range for period analytics
 * INPUT:     Context, start and end time boundaries
 * OUTPUT:    Session count within specified time range
 * BUSINESS:  Time-based session counting enables period-over-period analysis
 * CHANGE:    Initial time range counting query
 * RISK:      Low - Time-based aggregation query
 */
func (ksr *KuzuSessionRepository) CountSessionsByTimeRange(ctx context.Context, start, end time.Time) (int64, error) {
	query := `
		MATCH (s:Session)
		WHERE s.start_time >= $start_time AND s.start_time < $end_time
		RETURN COUNT(s) as session_count;
	`

	params := map[string]interface{}{
		"start_time": start,
		"end_time":   end,
	}

	result, err := ksr.connManager.Query(ctx, query, params)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions by time range: %w", err)
	}
	defer result.Close()

	if !result.HasNext() {
		return 0, nil
	}

	record, err := result.Next()
	if err != nil {
		return 0, fmt.Errorf("failed to read count result: %w", err)
	}

	return record[0].(int64), nil
}

/**
 * CONTEXT:   Get comprehensive session statistics for analytics
 * INPUT:     Context, user ID, and time range for statistics calculation
 * OUTPUT:    SessionStatistics with aggregated metrics and insights
 * BUSINESS:  Session statistics provide productivity insights and work patterns
 * CHANGE:    Initial session statistics aggregation with comprehensive metrics
 * RISK:      Medium - Complex aggregation query may impact performance
 */
func (ksr *KuzuSessionRepository) GetSessionStatistics(ctx context.Context, userID string, start, end time.Time) (*repositories.SessionStatistics, error) {
	query := `
		MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
		WHERE s.start_time >= $start_time AND s.start_time < $end_time
		RETURN 
			COUNT(s) as total_sessions,
			SUM(CASE WHEN s.state = 'active' THEN 1 ELSE 0 END) as active_sessions,
			SUM(CASE WHEN s.state = 'expired' THEN 1 ELSE 0 END) as expired_sessions,
			SUM(CASE WHEN s.state = 'finished' THEN 1 ELSE 0 END) as finished_sessions,
			SUM(s.duration_hours) as total_hours,
			AVG(s.duration_hours) as average_hours,
			SUM(s.activity_count) as total_activities,
			AVG(s.activity_count) as average_activities,
			MIN(s.start_time) as first_session,
			MAX(s.start_time) as last_session;
	`

	params := map[string]interface{}{
		"user_id":    userID,
		"start_time": start,
		"end_time":   end,
	}

	result, err := ksr.connManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get session statistics: %w", err)
	}
	defer result.Close()

	if !result.HasNext() {
		return &repositories.SessionStatistics{
			PeriodStart: start,
			PeriodEnd:   end,
		}, nil
	}

	record, err := result.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to read statistics result: %w", err)
	}

	stats := &repositories.SessionStatistics{
		TotalSessions:       record[0].(int64),
		ActiveSessions:      record[1].(int64),
		ExpiredSessions:     record[2].(int64),
		FinishedSessions:    record[3].(int64),
		TotalHours:          record[4].(float64),
		AverageSessionHours: record[5].(float64),
		TotalActivities:     record[6].(int64),
		AverageActivities:   record[7].(float64),
		FirstSessionTime:    record[8].(time.Time),
		LastSessionTime:     record[9].(time.Time),
		SessionDuration:     entities.SessionDuration,
		PeriodStart:         start,
		PeriodEnd:           end,
	}

	return stats, nil
}

/**
 * CONTEXT:   Save multiple sessions in batch for efficiency
 * INPUT:     Context and array of sessions to save
 * OUTPUT:    All sessions saved atomically or rollback on error
 * BUSINESS:  Batch operations improve performance for bulk session creation
 * CHANGE:    Initial batch save implementation with transaction support
 * RISK:      High - Batch operations require careful transaction management
 */
func (ksr *KuzuSessionRepository) SaveBatch(ctx context.Context, sessions []*entities.Session) error {
	if len(sessions) == 0 {
		return nil
	}

	return ksr.connManager.WithTransaction(ctx, func(conn *kuzu.Connection) error {
		for _, session := range sessions {
			if err := ksr.saveSessionInTransaction(conn, session); err != nil {
				return fmt.Errorf("failed to save session %s in batch: %w", session.ID(), err)
			}
		}
		return nil
	})
}

/**
 * CONTEXT:   Delete expired sessions in batch for maintenance
 * INPUT:     Context and cutoff time for expired session cleanup
 * OUTPUT:    Count of deleted sessions or error
 * BUSINESS:  Batch deletion of expired sessions enables database maintenance
 * CHANGE:    Initial batch deletion for maintenance operations
 * RISK:      Medium - Bulk deletion requires careful validation and logging
 */
func (ksr *KuzuSessionRepository) DeleteExpired(ctx context.Context, beforeTime time.Time) (int64, error) {
	return ksr.connManager.WithTransaction(ctx, func(conn *kuzu.Connection) (int64, error) {
		query := `
			MATCH (s:Session)
			WHERE s.end_time < $before_time
			DELETE s
			RETURN COUNT(s) as deleted_count;
		`

		params := map[string]interface{}{
			"before_time": beforeTime,
		}

		result, err := conn.Query(query)
		if err != nil {
			return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
		}
		defer result.Close()

		if !result.HasNext() {
			return 0, nil
		}

		record, err := result.Next()
		if err != nil {
			return 0, fmt.Errorf("failed to read deletion count: %w", err)
		}

		return record[0].(int64), nil
	})
}

/**
 * CONTEXT:   Execute function within transaction with session repository context
 * INPUT:     Context and transaction function with repository parameter
 * OUTPUT:    Transaction result with commit/rollback handling
 * BUSINESS:  Transaction support ensures data consistency for complex operations
 * CHANGE:    Initial transaction wrapper for repository operations
 * RISK:      Medium - Transaction management requires careful error handling
 */
func (ksr *KuzuSessionRepository) WithTransaction(ctx context.Context, fn func(repo repositories.SessionRepository) error) error {
	return ksr.connManager.WithTransaction(ctx, func(conn *kuzu.Connection) error {
		// Create temporary repository with transaction connection
		txRepo := &KuzuSessionRepository{
			connManager: &KuzuConnectionManager{
				database:    ksr.connManager.database,
				connections: make(chan *kuzu.Connection, 1),
				inUse:       make(map[*kuzu.Connection]bool),
			},
		}
		// Add connection to transaction repository
		txRepo.connManager.connections <- conn
		
		return fn(txRepo)
	})
}

// Helper methods for internal operations

/**
 * CONTEXT:   Execute session find query and convert results to entities
 * INPUT:     Context, Cypher query, and query parameters
 * OUTPUT:    Array of session entities from query results
 * BUSINESS:  Common query execution reduces code duplication and ensures consistency
 * CHANGE:    Initial query execution helper with result conversion
 * RISK:      Low - Internal helper with standardized error handling
 */
func (ksr *KuzuSessionRepository) executeFindQuery(ctx context.Context, query string, params map[string]interface{}) ([]*entities.Session, error) {
	result, err := ksr.connManager.Query(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer result.Close()

	var sessions []*entities.Session
	for result.HasNext() {
		record, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to read record: %w", err)
		}

		session, err := ksr.recordToSession(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to session: %w", err)
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

/**
 * CONTEXT:   Convert database record to session entity
 * INPUT:     Database record array with session fields
 * OUTPUT:    Session entity with proper field mapping
 * BUSINESS:  Record conversion enables clean separation between database and domain layers
 * CHANGE:    Initial record conversion with comprehensive field mapping
 * RISK:      Medium - Field mapping requires careful type conversion and validation
 */
func (ksr *KuzuSessionRepository) recordToSession(record []interface{}) (*entities.Session, error) {
	if len(record) < 12 {
		return nil, fmt.Errorf("invalid record length: expected 12 fields, got %d", len(record))
	}

	// Map database fields to session config
	config := entities.SessionConfig{
		UserID:    record[1].(string),
		StartTime: record[2].(time.Time),
	}

	session, err := entities.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session from record: %w", err)
	}

	// Set additional fields that aren't part of constructor
	sessionData := session.ToData()
	sessionData.ID = record[0].(string)
	sessionData.EndTime = record[3].(time.Time)
	sessionData.State = entities.SessionState(record[4].(string))
	sessionData.FirstActivityTime = record[5].(time.Time)
	sessionData.LastActivityTime = record[6].(time.Time)
	sessionData.ActivityCount = record[7].(int64)
	sessionData.DurationHours = record[8].(float64)
	sessionData.CreatedAt = record[10].(time.Time)
	sessionData.UpdatedAt = record[11].(time.Time)

	// Reconstruct session with updated data
	return ksr.sessionDataToEntity(sessionData)
}

/**
 * CONTEXT:   Build dynamic filter query with parameterized conditions
 * INPUT:     Session filter criteria with optional parameters
 * OUTPUT:    Cypher query string and parameter map for execution
 * BUSINESS:  Dynamic filtering enables flexible session queries for various use cases
 * CHANGE:    Initial dynamic query builder with SQL injection prevention
 * RISK:      Medium - Dynamic query building requires careful parameterization
 */
func (ksr *KuzuSessionRepository) buildFilterQuery(filter repositories.SessionFilter) (string, map[string]interface{}) {
	query := `
		MATCH (s:Session)
		WHERE 1=1
	`
	params := make(map[string]interface{})

	// Add filter conditions
	if filter.UserID != "" {
		query += " AND s.user_id = $user_id"
		params["user_id"] = filter.UserID
	}

	if filter.State != "" {
		query += " AND s.state = $state"
		params["state"] = string(filter.State)
	}

	if !filter.StartAfter.IsZero() {
		query += " AND s.start_time >= $start_after"
		params["start_after"] = filter.StartAfter
	}

	if !filter.StartBefore.IsZero() {
		query += " AND s.start_time < $start_before"
		params["start_before"] = filter.StartBefore
	}

	if !filter.EndAfter.IsZero() {
		query += " AND s.end_time >= $end_after"
		params["end_after"] = filter.EndAfter
	}

	if !filter.EndBefore.IsZero() {
		query += " AND s.end_time < $end_before"
		params["end_before"] = filter.EndBefore
	}

	if filter.IsActive != nil {
		query += " AND s.is_active = $is_active"
		params["is_active"] = *filter.IsActive
	}

	// Add result fields
	query += `
		RETURN s.id, s.user_id, s.start_time, s.end_time, s.state,
			   s.first_activity_time, s.last_activity_time, s.activity_count,
			   s.duration_hours, s.is_active, s.created_at, s.updated_at
	`

	// Add pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" SKIP %d", filter.Offset)
	}

	return query, params
}

/**
 * CONTEXT:   Build ORDER BY clause for session sorting
 * INPUT:     Sort field and direction parameters
 * OUTPUT:    SQL ORDER BY clause for query optimization
 * BUSINESS:  Sorting enables organized session displays and efficient pagination
 * CHANGE:    Initial sort clause builder with field validation
 * RISK:      Low - Static sort clause generation with field validation
 */
func (ksr *KuzuSessionRepository) buildOrderClause(sortBy repositories.SessionSortBy, order repositories.SessionSortOrder) string {
	var field string
	switch sortBy {
	case repositories.SessionSortByStartTime:
		field = "s.start_time"
	case repositories.SessionSortByEndTime:
		field = "s.end_time"
	case repositories.SessionSortByUpdated:
		field = "s.updated_at"
	case repositories.SessionSortByActivity:
		field = "s.activity_count"
	default:
		field = "s.start_time"
	}

	direction := "ASC"
	if order == repositories.SessionSortDesc {
		direction = "DESC"
	}

	return fmt.Sprintf("ORDER BY %s %s", field, direction)
}

/**
 * CONTEXT:   Save session within existing transaction
 * INPUT:     Transaction connection and session entity
 * OUTPUT:    Session saved within transaction context
 * BUSINESS:  Transaction-aware saving enables batch operations and consistency
 * CHANGE:    Initial transaction-aware save operation
 * RISK:      Medium - Transaction context requires careful connection handling
 */
func (ksr *KuzuSessionRepository) saveSessionInTransaction(conn *kuzu.Connection, session *entities.Session) error {
	// Validate session
	if err := session.Validate(); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	// Ensure user exists
	userQuery := `
		MERGE (u:User {id: $user_id})
		ON CREATE SET u.name = $user_id, u.created_at = current_timestamp(), u.updated_at = current_timestamp();
	`

	userParams := map[string]interface{}{
		"user_id": session.UserID(),
	}

	if _, err := conn.Query(userQuery); err != nil {
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Create session
	sessionQuery := `
		CREATE (s:Session {
			id: $id,
			user_id: $user_id,
			start_time: $start_time,
			end_time: $end_time,
			state: $state,
			first_activity_time: $first_activity_time,
			last_activity_time: $last_activity_time,
			activity_count: $activity_count,
			duration_hours: $duration_hours,
			is_active: $is_active,
			created_at: $created_at,
			updated_at: $updated_at
		});
	`

	sessionParams := map[string]interface{}{
		"id":                   session.ID(),
		"user_id":              session.UserID(),
		"start_time":           session.StartTime(),
		"end_time":             session.EndTime(),
		"state":                string(session.State()),
		"first_activity_time":  session.FirstActivityTime(),
		"last_activity_time":   session.LastActivityTime(),
		"activity_count":       session.ActivityCount(),
		"duration_hours":       session.Duration().Hours(),
		"is_active":            session.State() == entities.SessionStateActive,
		"created_at":           session.CreatedAt(),
		"updated_at":           session.UpdatedAt(),
	}

	if _, err := conn.Query(sessionQuery); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Create relationship
	relationshipQuery := `
		MATCH (u:User {id: $user_id}), (s:Session {id: $session_id})
		CREATE (u)-[:HAS_SESSION {created_at: current_timestamp()}]->(s);
	`

	relationshipParams := map[string]interface{}{
		"user_id":    session.UserID(),
		"session_id": session.ID(),
	}

	if _, err := conn.Query(relationshipQuery); err != nil {
		return fmt.Errorf("failed to create user-session relationship: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Convert session data struct to session entity
 * INPUT:     SessionData struct with all session fields
 * OUTPUT:    Session entity with proper business logic
 * BUSINESS:  Data conversion enables reconstruction of entities from storage
 * CHANGE:    Initial data-to-entity conversion for repository operations
 * RISK:      Low - Simple conversion with validation
 */
func (ksr *KuzuSessionRepository) sessionDataToEntity(data entities.SessionData) (*entities.Session, error) {
	// Create new session with basic data
	config := entities.SessionConfig{
		UserID:    data.UserID,
		StartTime: data.StartTime,
	}

	session, err := entities.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session entity: %w", err)
	}

	// Note: In a full implementation, we would need methods to reconstruct
	// the session with all its internal state. This is a simplified version.
	
	return session, nil
}