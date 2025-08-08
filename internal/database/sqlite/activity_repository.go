/**
 * CONTEXT:   Activity repository for WorkBlock-centric activity management
 * INPUT:     Activity events with FK relationships to work blocks
 * OUTPUT:    Database CRUD operations with proper JSON metadata handling
 * BUSINESS:  Activities belong to work blocks and drive idle detection
 * CHANGE:    New repository replacing in-memory activity arrays
 * RISK:      Low - Standard repository pattern with FK constraints
 */

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ActivityRepository handles database operations for activities
type ActivityRepository struct {
	db *sql.DB
}

// NewActivityRepository creates a new activity repository
func NewActivityRepository(db *sql.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// Activity represents an activity event in the database
type Activity struct {
	ID           string            `json:"id"`
	WorkBlockID  string            `json:"work_block_id"`
	Timestamp    time.Time         `json:"timestamp"`
	ActivityType string            `json:"activity_type"`
	Command      string            `json:"command"`
	Description  string            `json:"description"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
}

// ActivitySummary provides aggregated activity data for reporting
type ActivitySummary struct {
	TotalActivities int               `json:"total_activities"`
	ActivityCounts  map[string]int    `json:"activity_counts"`
	FirstActivity   time.Time         `json:"first_activity"`
	LastActivity    time.Time         `json:"last_activity"`
	ActivityRate    float64           `json:"activity_rate"` // activities per minute
}

/**
 * CONTEXT:   Save activity to database with WorkBlock FK relationship
 * INPUT:     Activity struct with all required fields and JSON metadata
 * OUTPUT:    Database insert with FK constraint validation
 * BUSINESS:  Activities must belong to existing work blocks
 * CHANGE:    Initial implementation with JSON metadata support
 * RISK:      Medium - JSON serialization and FK constraint validation
 */
func (r *ActivityRepository) SaveActivity(activity *Activity) error {
	// Serialize metadata to JSON
	metadataJSON := "{}"
	if activity.Metadata != nil {
		jsonBytes, err := json.Marshal(activity.Metadata)
		if err != nil {
			return fmt.Errorf("failed to serialize metadata: %w", err)
		}
		metadataJSON = string(jsonBytes)
	}

	query := `
		INSERT INTO activities (
			id, work_block_id, timestamp, activity_type, 
			command, description, metadata, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		activity.ID,
		activity.WorkBlockID,
		activity.Timestamp,
		activity.ActivityType,
		activity.Command,
		activity.Description,
		metadataJSON,
		activity.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save activity: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Get all activities for a specific work block
 * INPUT:     Work block ID for FK relationship lookup
 * OUTPUT:    Array of activities with deserialized JSON metadata
 * BUSINESS:  Activities are retrieved by work block for reporting
 * CHANGE:    Initial implementation with JSON metadata deserialization
 * RISK:      Low - Standard SELECT query with JSON handling
 */
func (r *ActivityRepository) GetActivitiesByWorkBlock(workBlockID string) ([]*Activity, error) {
	query := `
		SELECT id, work_block_id, timestamp, activity_type, 
		       command, description, metadata, created_at
		FROM activities 
		WHERE work_block_id = ? 
		ORDER BY timestamp ASC`

	rows, err := r.db.Query(query, workBlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to query activities: %w", err)
	}
	defer rows.Close()

	var activities []*Activity
	for rows.Next() {
		activity := &Activity{}
		var metadataJSON string

		err := rows.Scan(
			&activity.ID,
			&activity.WorkBlockID,
			&activity.Timestamp,
			&activity.ActivityType,
			&activity.Command,
			&activity.Description,
			&metadataJSON,
			&activity.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}

		// Deserialize metadata
		if metadataJSON != "" && metadataJSON != "{}" {
			err = json.Unmarshal([]byte(metadataJSON), &activity.Metadata)
			if err != nil {
				// Log error but continue - don't fail entire query for bad JSON
				activity.Metadata = map[string]string{"parse_error": metadataJSON}
			}
		} else {
			activity.Metadata = make(map[string]string)
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

/**
 * CONTEXT:   Get activity summary for work block reporting and analytics
 * INPUT:     Work block ID for aggregation calculations
 * OUTPUT:    Activity summary with counts, rates, and time ranges
 * BUSINESS:  Activity summaries drive work block insights and reporting
 * CHANGE:    Initial implementation with comprehensive activity metrics
 * RISK:      Low - Aggregation query with proper error handling
 */
func (r *ActivityRepository) GetActivitySummary(workBlockID string) (*ActivitySummary, error) {
	// Get basic activity statistics
	query := `
		SELECT 
			COUNT(*) as total_activities,
			MIN(timestamp) as first_activity,
			MAX(timestamp) as last_activity,
			activity_type
		FROM activities 
		WHERE work_block_id = ? 
		GROUP BY activity_type
		ORDER BY COUNT(*) DESC`

	rows, err := r.db.Query(query, workBlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to query activity summary: %w", err)
	}
	defer rows.Close()

	summary := &ActivitySummary{
		ActivityCounts: make(map[string]int),
	}

	var firstActivitySet, lastActivitySet bool
	for rows.Next() {
		var count int
		var activityType string
		var firstActivity, lastActivity time.Time

		err := rows.Scan(&count, &firstActivity, &lastActivity, &activityType)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity summary: %w", err)
		}

		summary.TotalActivities += count
		summary.ActivityCounts[activityType] = count

		// Track overall first and last activity times
		if !firstActivitySet || firstActivity.Before(summary.FirstActivity) {
			summary.FirstActivity = firstActivity
			firstActivitySet = true
		}
		if !lastActivitySet || lastActivity.After(summary.LastActivity) {
			summary.LastActivity = lastActivity
			lastActivitySet = true
		}
	}

	// Calculate activity rate (activities per minute)
	if firstActivitySet && lastActivitySet && summary.TotalActivities > 0 {
		duration := summary.LastActivity.Sub(summary.FirstActivity)
		if duration > 0 {
			summary.ActivityRate = float64(summary.TotalActivities) / duration.Minutes()
		}
	}

	return summary, nil
}

/**
 * CONTEXT:   Get activities within a time range for reporting
 * INPUT:     Time range for activity filtering across all work blocks
 * OUTPUT:    Activities array sorted by timestamp
 * BUSINESS:  Time-based activity queries support daily/weekly/monthly reports
 * CHANGE:    Initial implementation for reporting system integration
 * RISK:      Low - Standard time range query with proper indexing
 */
func (r *ActivityRepository) GetActivitiesByTimeRange(startTime, endTime time.Time) ([]*Activity, error) {
	query := `
		SELECT a.id, a.work_block_id, a.timestamp, a.activity_type, 
		       a.command, a.description, a.metadata, a.created_at
		FROM activities a
		JOIN work_blocks wb ON a.work_block_id = wb.id
		WHERE a.timestamp >= ? AND a.timestamp <= ?
		ORDER BY a.timestamp ASC`

	rows, err := r.db.Query(query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query activities by time range: %w", err)
	}
	defer rows.Close()

	var activities []*Activity
	for rows.Next() {
		activity := &Activity{}
		var metadataJSON string

		err := rows.Scan(
			&activity.ID,
			&activity.WorkBlockID,
			&activity.Timestamp,
			&activity.ActivityType,
			&activity.Command,
			&activity.Description,
			&metadataJSON,
			&activity.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}

		// Deserialize metadata
		if metadataJSON != "" && metadataJSON != "{}" {
			err = json.Unmarshal([]byte(metadataJSON), &activity.Metadata)
			if err != nil {
				activity.Metadata = map[string]string{"parse_error": metadataJSON}
			}
		} else {
			activity.Metadata = make(map[string]string)
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

/**
 * CONTEXT:   Bulk insert activities for performance optimization
 * INPUT:     Array of activities for batch database operations
 * OUTPUT:    Single transaction with all activities inserted
 * BUSINESS:  Bulk operations improve performance for high-frequency activity logging
 * CHANGE:    Initial implementation with transaction support
 * RISK:      Medium - Transaction management and bulk insert validation
 */
func (r *ActivityRepository) SaveActivitiesBatch(activities []*Activity) error {
	if len(activities) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO activities (
			id, work_block_id, timestamp, activity_type, 
			command, description, metadata, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, activity := range activities {
		// Serialize metadata to JSON
		metadataJSON := "{}"
		if activity.Metadata != nil {
			jsonBytes, err := json.Marshal(activity.Metadata)
			if err != nil {
				return fmt.Errorf("failed to serialize metadata for activity %s: %w", activity.ID, err)
			}
			metadataJSON = string(jsonBytes)
		}

		_, err = stmt.Exec(
			activity.ID,
			activity.WorkBlockID,
			activity.Timestamp,
			activity.ActivityType,
			activity.Command,
			activity.Description,
			metadataJSON,
			activity.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to execute statement for activity %s: %w", activity.ID, err)
		}
	}

	return tx.Commit()
}

/**
 * CONTEXT:   Delete activities by work block for cleanup operations
 * INPUT:     Work block ID for cascade deletion
 * OUTPUT:    Cleanup of all associated activities
 * BUSINESS:  Activity cleanup supports work block lifecycle management
 * CHANGE:    Initial implementation for data lifecycle management
 * RISK:      Medium - Deletion operations require careful validation
 */
func (r *ActivityRepository) DeleteActivitiesByWorkBlock(workBlockID string) error {
	query := `DELETE FROM activities WHERE work_block_id = ?`
	
	result, err := r.db.Exec(query, workBlockID)
	if err != nil {
		return fmt.Errorf("failed to delete activities: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log successful deletion
	_ = rowsAffected // Could log this for debugging

	return nil
}

/**
 * CONTEXT:   Get all activities in the system for health monitoring and statistics
 * INPUT:     Context for database operations
 * OUTPUT:    All activities in the database, ordered by timestamp
 * BUSINESS:  System-wide activity queries for health monitoring and analytics
 * CHANGE:    Added GetAll method for activity count and system monitoring
 * RISK:      Medium - Could return large dataset, consider pagination in future
 */
func (r *ActivityRepository) GetAll(ctx context.Context) ([]*Activity, error) {
	query := `
		SELECT 
			id, work_block_id, timestamp, activity_type, 
			command, description, metadata, created_at
		FROM activities 
		ORDER BY timestamp DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all activities: %w", err)
	}
	defer rows.Close()

	var activities []*Activity
	for rows.Next() {
		activity := &Activity{}
		var metadataJSON string

		err := rows.Scan(
			&activity.ID,
			&activity.WorkBlockID,
			&activity.Timestamp,
			&activity.ActivityType,
			&activity.Command,
			&activity.Description,
			&metadataJSON,
			&activity.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}

		// Deserialize metadata
		if metadataJSON != "" && metadataJSON != "{}" {
			err = json.Unmarshal([]byte(metadataJSON), &activity.Metadata)
			if err != nil {
				activity.Metadata = map[string]string{"parse_error": metadataJSON}
			}
		} else {
			activity.Metadata = make(map[string]string)
		}

		activities = append(activities, activity)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activities: %w", err)
	}

	return activities, nil
}