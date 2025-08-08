/**
 * CONTEXT:   Work block repository for SQLite database operations
 * INPUT:     Work block CRUD operations with project associations
 * OUTPUT:    Database persistence for work block lifecycle management
 * BUSINESS:  Work blocks track active work periods within sessions
 * CHANGE:    CHECKPOINT 8 - Basic work block repository for production deployment
 * RISK:      Low - Simple repository implementation for deployment verification
 */

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// WorkBlockRepository handles database operations for work blocks
type WorkBlockRepository struct {
	db *sql.DB
}

// NewWorkBlockRepository creates a new work block repository
func NewWorkBlockRepository(db *sql.DB) *WorkBlockRepository {
	return &WorkBlockRepository{db: db}
}

// WorkBlock type is already defined in migration.go

/**
 * CONTEXT:   Create new work block in database
 * INPUT:     Work block entity with session and project associations
 * OUTPUT:    Persisted work block with database constraints validated
 * BUSINESS:  Work blocks must belong to sessions and projects
 * CHANGE:    CHECKPOINT 8 - Basic work block creation
 * RISK:      Low - Standard database insertion with FK constraints
 */
func (wr *WorkBlockRepository) Create(ctx context.Context, workBlock *WorkBlock) error {
	if workBlock == nil {
		return fmt.Errorf("work block cannot be nil")
	}

	query := `
		INSERT INTO work_blocks (
			id, session_id, project_id, start_time, end_time, state,
			last_activity_time, activity_count, duration_seconds, 
			duration_hours, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := wr.db.ExecContext(ctx, query,
		workBlock.ID, workBlock.SessionID, workBlock.ProjectID,
		workBlock.StartTime, workBlock.EndTime, workBlock.State,
		workBlock.LastActivityTime, workBlock.ActivityCount,
		workBlock.DurationSeconds, workBlock.DurationHours,
		workBlock.CreatedAt, workBlock.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create work block: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Get work block by ID
 * INPUT:     Work block ID for lookup
 * OUTPUT:    Work block entity or error if not found
 * BUSINESS:  Work block lookup for activity processing
 * CHANGE:    CHECKPOINT 8 - Basic work block retrieval
 * RISK:      Low - Standard database query
 */
func (wr *WorkBlockRepository) GetByID(ctx context.Context, id string) (*WorkBlock, error) {
	if id == "" {
		return nil, fmt.Errorf("work block ID cannot be empty")
	}

	query := `
		SELECT wb.id, wb.session_id, wb.project_id, wb.start_time, wb.end_time,
		       wb.state, wb.last_activity_time, wb.activity_count,
		       wb.duration_seconds, wb.duration_hours, wb.created_at, wb.updated_at
		FROM work_blocks wb
		WHERE wb.id = ?
	`

	var wb WorkBlock
	var endTime sql.NullTime

	err := wr.db.QueryRowContext(ctx, query, id).Scan(
		&wb.ID, &wb.SessionID, &wb.ProjectID, &wb.StartTime, &endTime,
		&wb.State, &wb.LastActivityTime, &wb.ActivityCount,
		&wb.DurationSeconds, &wb.DurationHours, &wb.CreatedAt, &wb.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("work block %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get work block: %w", err)
	}

	if endTime.Valid {
		wb.EndTime = &endTime.Time
	}

	return &wb, nil
}

/**
 * CONTEXT:   Get active work block for session and project
 * INPUT:     Session ID and project ID
 * OUTPUT:    Active work block or nil if none active
 * BUSINESS:  Only one active work block per session-project combination
 * CHANGE:    CHECKPOINT 8 - Active work block lookup
 * RISK:      Low - Query with session and project constraints
 */
func (wr *WorkBlockRepository) GetActiveBySessionAndProject(ctx context.Context, sessionID, projectID string) (*WorkBlock, error) {
	if sessionID == "" || projectID == "" {
		return nil, fmt.Errorf("session ID and project ID cannot be empty")
	}

	query := `
		SELECT wb.id, wb.session_id, wb.project_id, wb.start_time, wb.end_time,
		       wb.state, wb.last_activity_time, wb.activity_count,
		       wb.duration_seconds, wb.duration_hours, wb.created_at, wb.updated_at
		FROM work_blocks wb
		WHERE wb.session_id = ? AND wb.project_id = ? AND wb.end_time IS NULL
		ORDER BY wb.last_activity_time DESC
		LIMIT 1
	`

	var wb WorkBlock
	var endTime sql.NullTime

	err := wr.db.QueryRowContext(ctx, query, sessionID, projectID).Scan(
		&wb.ID, &wb.SessionID, &wb.ProjectID, &wb.StartTime, &endTime,
		&wb.State, &wb.LastActivityTime, &wb.ActivityCount,
		&wb.DurationSeconds, &wb.DurationHours, &wb.CreatedAt, &wb.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No active work block found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active work block: %w", err)
	}

	return &wb, nil
}

/**
 * CONTEXT:   Check if work block is idle based on last activity time
 * INPUT:     Work block and current time for idle detection
 * OUTPUT:    Boolean indicating if work block is idle (>5 minutes)
 * BUSINESS:  Idle work blocks should be finalized and new ones created
 * CHANGE:    CHECKPOINT 8 - Idle detection logic
 * RISK:      Low - Time comparison for idle detection
 */
func (wr *WorkBlockRepository) IsWorkBlockIdle(workBlock *WorkBlock, currentTime time.Time) bool {
	if workBlock == nil {
		return true
	}
	
	idleThreshold := 5 * time.Minute
	return currentTime.Sub(workBlock.LastActivityTime) > idleThreshold
}

/**
 * CONTEXT:   Record activity in work block
 * INPUT:     Work block ID and activity timestamp
 * OUTPUT:    Updated work block with new activity time and count
 * BUSINESS:  Activities update work block last activity time
 * CHANGE:    CHECKPOINT 8 - Activity recording in work blocks
 * RISK:      Low - Update operation with activity tracking
 */
func (wr *WorkBlockRepository) RecordActivity(ctx context.Context, workBlockID string, activityTime time.Time) error {
	if workBlockID == "" {
		return fmt.Errorf("work block ID cannot be empty")
	}

	query := `
		UPDATE work_blocks 
		SET last_activity_time = ?, activity_count = activity_count + 1, updated_at = ?
		WHERE id = ?
	`

	result, err := wr.db.ExecContext(ctx, query, activityTime, time.Now(), workBlockID)
	if err != nil {
		return fmt.Errorf("failed to record activity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("work block %s not found", workBlockID)
	}

	return nil
}

/**
 * CONTEXT:   Finish work block with end time and duration calculation
 * INPUT:     Work block ID and end timestamp
 * OUTPUT:    Finished work block with calculated duration
 * BUSINESS:  Work blocks are finished when sessions end or idle timeout
 * CHANGE:    CHECKPOINT 8 - Work block finalization
 * RISK:      Low - Update operation with duration calculation
 */
func (wr *WorkBlockRepository) FinishWorkBlock(ctx context.Context, workBlockID string, endTime time.Time) error {
	if workBlockID == "" {
		return fmt.Errorf("work block ID cannot be empty")
	}

	// Get current work block to calculate duration
	workBlock, err := wr.GetByID(ctx, workBlockID)
	if err != nil {
		return fmt.Errorf("failed to get work block for finishing: %w", err)
	}

	duration := endTime.Sub(workBlock.StartTime)
	durationSeconds := int(duration.Seconds())
	durationHours := duration.Hours()

	query := `
		UPDATE work_blocks 
		SET end_time = ?, duration_seconds = ?, duration_hours = ?, 
		    state = 'finished', updated_at = ?
		WHERE id = ?
	`

	result, err := wr.db.ExecContext(ctx, query, endTime, durationSeconds, durationHours, time.Now(), workBlockID)
	if err != nil {
		return fmt.Errorf("failed to finish work block: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("work block %s not found", workBlockID)
	}

	return nil
}

/**
 * CONTEXT:   Get work blocks by session for reporting
 * INPUT:     Session ID and optional limit
 * OUTPUT:    Array of work blocks for the session
 * BUSINESS:  Session work block analysis for reporting
 * CHANGE:    CHECKPOINT 8 - Session work block retrieval
 * RISK:      Low - Query with session filter
 */
func (wr *WorkBlockRepository) GetBySession(ctx context.Context, sessionID string, limit int) ([]*WorkBlock, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	query := `
		SELECT wb.id, wb.session_id, wb.project_id, wb.start_time, wb.end_time,
		       wb.state, wb.last_activity_time, wb.activity_count,
		       wb.duration_seconds, wb.duration_hours, wb.created_at, wb.updated_at
		FROM work_blocks wb
		WHERE wb.session_id = ?
		ORDER BY wb.start_time ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := wr.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get work blocks by session: %w", err)
	}
	defer rows.Close()

	var workBlocks []*WorkBlock
	for rows.Next() {
		var wb WorkBlock
		var endTime sql.NullTime

		err := rows.Scan(
			&wb.ID, &wb.SessionID, &wb.ProjectID, &wb.StartTime, &endTime,
			&wb.State, &wb.LastActivityTime, &wb.ActivityCount,
			&wb.DurationSeconds, &wb.DurationHours, &wb.CreatedAt, &wb.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan work block: %w", err)
		}

		if endTime.Valid {
			wb.EndTime = &endTime.Time
		}

		workBlocks = append(workBlocks, &wb)
	}

	return workBlocks, nil
}

/**
 * CONTEXT:   Get all work blocks for system monitoring
 * INPUT:     Context for database operations
 * OUTPUT:    All work blocks in the system
 * BUSINESS:  System monitoring and cleanup operations
 * CHANGE:    CHECKPOINT 8 - All work blocks retrieval
 * RISK:      Low - System-wide query for monitoring
 */
func (wr *WorkBlockRepository) GetAll(ctx context.Context) ([]*WorkBlock, error) {
	query := `
		SELECT wb.id, wb.session_id, wb.project_id, wb.start_time, wb.end_time,
		       wb.state, wb.last_activity_time, wb.activity_count,
		       wb.duration_seconds, wb.duration_hours, wb.created_at, wb.updated_at
		FROM work_blocks wb
		ORDER BY wb.created_at DESC
	`

	rows, err := wr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all work blocks: %w", err)
	}
	defer rows.Close()

	var workBlocks []*WorkBlock
	for rows.Next() {
		var wb WorkBlock
		var endTime sql.NullTime

		err := rows.Scan(
			&wb.ID, &wb.SessionID, &wb.ProjectID, &wb.StartTime, &endTime,
			&wb.State, &wb.LastActivityTime, &wb.ActivityCount,
			&wb.DurationSeconds, &wb.DurationHours, &wb.CreatedAt, &wb.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan work block: %w", err)
		}

		if endTime.Valid {
			wb.EndTime = &endTime.Time
		}

		workBlocks = append(workBlocks, &wb)
	}

	return workBlocks, nil
}

/**
 * CONTEXT:   Update work block in database
 * INPUT:     Updated work block entity
 * OUTPUT:    Persisted changes to work block
 * BUSINESS:  Work block updates for state changes and duration updates
 * CHANGE:    CHECKPOINT 8 - Work block update operations
 * RISK:      Low - Standard update operation
 */
func (wr *WorkBlockRepository) Update(ctx context.Context, workBlock *WorkBlock) error {
	if workBlock == nil {
		return fmt.Errorf("work block cannot be nil")
	}

	query := `
		UPDATE work_blocks 
		SET session_id = ?, project_id = ?, start_time = ?, end_time = ?, 
		    state = ?, last_activity_time = ?, activity_count = ?,
		    duration_seconds = ?, duration_hours = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := wr.db.ExecContext(ctx, query,
		workBlock.SessionID, workBlock.ProjectID, workBlock.StartTime, workBlock.EndTime,
		workBlock.State, workBlock.LastActivityTime, workBlock.ActivityCount,
		workBlock.DurationSeconds, workBlock.DurationHours, workBlock.UpdatedAt,
		workBlock.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update work block: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("work block %s not found", workBlock.ID)
	}

	return nil
}

/**
 * CONTEXT:   Mark idle work blocks as finished
 * INPUT:     Current timestamp for idle detection
 * OUTPUT:    Count of work blocks marked as idle
 * BUSINESS:  Cleanup operation for idle work blocks
 * CHANGE:    CHECKPOINT 8 - Idle work block cleanup
 * RISK:      Medium - Bulk update operation affecting multiple work blocks
 */
func (wr *WorkBlockRepository) MarkIdleWorkBlocks(ctx context.Context, currentTime time.Time) (int, error) {
	idleThreshold := currentTime.Add(-5 * time.Minute)
	
	query := `
		UPDATE work_blocks 
		SET end_time = datetime(last_activity_time, '+5 minutes'),
		    duration_seconds = CAST((julianday(datetime(last_activity_time, '+5 minutes')) - julianday(start_time)) * 86400 AS INTEGER),
		    duration_hours = (julianday(datetime(last_activity_time, '+5 minutes')) - julianday(start_time)) * 24,
		    state = 'idle',
		    updated_at = ?
		WHERE end_time IS NULL 
		  AND last_activity_time < ?
	`

	result, err := wr.db.ExecContext(ctx, query, currentTime, idleThreshold)
	if err != nil {
		return 0, fmt.Errorf("failed to mark idle work blocks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}