/**
 * CONTEXT:   Migration utility to convert gob backup data to SQLite format
 * INPUT:     Gob backup file path and target SQLite database
 * OUTPUT:    Complete data migration with data integrity validation
 * BUSINESS:  Preserve all historical work tracking data during system upgrade
 * CHANGE:    One-time migration from gob binary format to SQLite relational storage
 * RISK:      Medium - Critical data migration must preserve all data without loss
 */

package sqlite

import (
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"time"
)

// Legacy data structures from gob backup (for migration only)
type LegacyDatabaseData struct {
	Sessions    map[string]*LegacySession      `gob:"sessions"`
	WorkBlocks  map[string]*LegacyWorkBlock    `gob:"work_blocks"`
	Activities  []*LegacyActivityEvent         `gob:"activities"`
	LastUpdated time.Time                      `gob:"last_updated"`
	Version     string                         `gob:"version"`
}

type LegacySession struct {
	ID                string    `gob:"id" json:"id"`
	UserID            string    `gob:"user_id" json:"user_id"`
	StartTime         time.Time `gob:"start_time" json:"start_time"`
	EndTime           time.Time `gob:"end_time" json:"end_time"`
	State             string    `gob:"state" json:"state"`
	FirstActivityTime time.Time `gob:"first_activity_time" json:"first_activity_time"`
	LastActivityTime  time.Time `gob:"last_activity_time" json:"last_activity_time"`
	ActivityCount     int64     `gob:"activity_count" json:"activity_count"`
	DurationHours     float64   `gob:"duration_hours" json:"duration_hours"`
	IsActive          bool      `gob:"is_active" json:"is_active"`
	CreatedAt         time.Time `gob:"created_at" json:"created_at"`
	UpdatedAt         time.Time `gob:"updated_at" json:"updated_at"`
}

type LegacyWorkBlock struct {
	ID                     string            `gob:"id" json:"id"`
	SessionID              string            `gob:"session_id" json:"session_id"`
	ProjectName            string            `gob:"project_name" json:"project_name"`
	ProjectPath            string            `gob:"project_path" json:"project_path"`
	StartTime              time.Time         `gob:"start_time" json:"start_time"`
	EndTime                time.Time         `gob:"end_time" json:"end_time"`
	State                  string            `gob:"state" json:"state"`
	LastActivityTime       time.Time         `gob:"last_activity_time" json:"last_activity_time"`
	ActivityCount          int64             `gob:"activity_count" json:"activity_count"`
	ActivityTypeCounters   map[string]int64  `gob:"activity_type_counters" json:"activity_type_counters"`
	DurationSeconds        int64             `gob:"duration_seconds" json:"duration_seconds"`
	DurationHours          float64           `gob:"duration_hours" json:"duration_hours"`
	IsActive               bool              `gob:"is_active" json:"is_active"`
	CreatedAt              time.Time         `gob:"created_at" json:"created_at"`
	UpdatedAt              time.Time         `gob:"updated_at" json:"updated_at"`
}

type LegacyActivityEvent struct {
	ID             string    `gob:"id"`
	UserID         string    `gob:"user_id"`
	SessionID      string    `gob:"session_id"`
	WorkBlockID    string    `gob:"work_block_id"`
	ProjectPath    string    `gob:"project_path"`
	ProjectName    string    `gob:"project_name"`
	ActivityType   string    `gob:"activity_type"`
	ActivitySource string    `gob:"activity_source"`
	Timestamp      time.Time `gob:"timestamp"`
	Command        string    `gob:"command"`
	Description    string    `gob:"description"`
	Metadata       string    `gob:"metadata"`
	CreatedAt      time.Time `gob:"created_at"`
}

// MigrationResult contains statistics about the migration process
type MigrationResult struct {
	UsersCreated      int           `json:"users_created"`
	ProjectsCreated   int           `json:"projects_created"`  
	SessionsMigrated  int           `json:"sessions_migrated"`
	WorkBlocksMigrated int          `json:"work_blocks_migrated"`
	ActivitiesMigrated int          `json:"activities_migrated"`
	DataIntegrityValid bool         `json:"data_integrity_valid"`
	MigrationDuration  time.Duration `json:"migration_duration"`
	Errors            []string       `json:"errors"`
}

/**
 * CONTEXT:   Load legacy gob backup data for migration to SQLite
 * INPUT:     Path to gob backup file created by legacy system
 * OUTPUT:    Legacy database data structure or error if load fails
 * BUSINESS:  Preserve complete work history during system architecture change
 * CHANGE:    One-time data loading from binary gob format
 * RISK:      Medium - Must handle potentially corrupted backup files gracefully
 */
func LoadLegacyData(gobFilePath string) (*LegacyDatabaseData, error) {
	if _, err := os.Stat(gobFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("backup file does not exist: %s", gobFilePath)
	}

	file, err := os.Open(gobFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	var data LegacyDatabaseData
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode backup data: %w", err)
	}

	// Initialize maps if nil (defensive programming)
	if data.Sessions == nil {
		data.Sessions = make(map[string]*LegacySession)
	}
	if data.WorkBlocks == nil {
		data.WorkBlocks = make(map[string]*LegacyWorkBlock)
	}
	if data.Activities == nil {
		data.Activities = make([]*LegacyActivityEvent, 0)
	}

	log.Printf("📦 Loaded legacy data: %d sessions, %d work blocks, %d activities", 
		len(data.Sessions), len(data.WorkBlocks), len(data.Activities))

	return &data, nil
}

/**
 * CONTEXT:   Complete migration from gob backup to SQLite with data validation
 * INPUT:     SQLite database connection and path to gob backup file
 * OUTPUT:    Migration result with statistics and validation status
 * BUSINESS:  Seamless transition to new architecture without data loss
 * CHANGE:    One-time complete system data migration with integrity checks
 * RISK:      High - Critical operation that must succeed without data corruption
 */
func MigrateFromGobBackup(db *SQLiteDB, gobFilePath string) (*MigrationResult, error) {
	startTime := time.Now()
	result := &MigrationResult{
		Errors: make([]string, 0),
	}

	log.Printf("🔄 Starting migration from gob backup: %s", gobFilePath)

	// Load legacy data
	legacyData, err := LoadLegacyData(gobFilePath)
	if err != nil {
		return result, fmt.Errorf("failed to load legacy data: %w", err)
	}

	// Perform migration in a single transaction for atomicity
	ctx := context.Background()
	err = db.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Step 1: Extract and create users
		users, err := extractUsers(legacyData)
		if err != nil {
			return fmt.Errorf("failed to extract users: %w", err)
		}

		for _, user := range users {
			if err := insertUser(tx, user); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert user %s: %v", user.ID, err))
				return err
			}
		}
		result.UsersCreated = len(users)
		log.Printf("✅ Created %d users", result.UsersCreated)

		// Step 2: Extract and create projects
		projects, err := extractProjects(legacyData)
		if err != nil {
			return fmt.Errorf("failed to extract projects: %w", err)
		}

		for _, project := range projects {
			if err := insertProject(tx, project); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert project %s: %v", project.ID, err))
				return err
			}
		}
		result.ProjectsCreated = len(projects)
		log.Printf("✅ Created %d projects", result.ProjectsCreated)

		// Step 3: Migrate sessions with proper end_time calculation
		for _, legacySession := range legacyData.Sessions {
			session := convertSession(legacySession, db.timezone)
			if err := insertSession(tx, session); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert session %s: %v", session.ID, err))
				return err
			}
			result.SessionsMigrated++
		}
		log.Printf("✅ Migrated %d sessions", result.SessionsMigrated)

		// Step 4: Migrate work blocks with project references
		projectLookup := createProjectLookup(projects)
		for _, legacyWorkBlock := range legacyData.WorkBlocks {
			workBlock := convertWorkBlock(legacyWorkBlock, projectLookup, db.timezone)
			if err := insertWorkBlock(tx, workBlock); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert work block %s: %v", workBlock.ID, err))
				return err
			}
			result.WorkBlocksMigrated++
		}
		log.Printf("✅ Migrated %d work blocks", result.WorkBlocksMigrated)

		// Step 5: Migrate activity events (if needed for full history)
		for _, legacyActivity := range legacyData.Activities {
			activity := convertActivityEvent(legacyActivity, projectLookup, db.timezone)
			if err := insertActivityEvent(tx, activity); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert activity %s: %v", activity.ID, err))
				return err
			}
			result.ActivitiesMigrated++
		}
		log.Printf("✅ Migrated %d activity events", result.ActivitiesMigrated)

		return nil
	})

	if err != nil {
		result.MigrationDuration = time.Since(startTime)
		return result, fmt.Errorf("migration transaction failed: %w", err)
	}

	result.MigrationDuration = time.Since(startTime)

	// Step 6: Validate data integrity
	log.Printf("🔍 Validating data integrity...")
	if err := validateMigrationIntegrity(db, legacyData, result); err != nil {
		result.DataIntegrityValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Data integrity validation failed: %v", err))
		return result, fmt.Errorf("migration integrity validation failed: %w", err)
	}

	result.DataIntegrityValid = true
	log.Printf("✅ Migration completed successfully in %v", result.MigrationDuration)
	log.Printf("📊 Migration Summary: %d users, %d projects, %d sessions, %d work blocks, %d activities", 
		result.UsersCreated, result.ProjectsCreated, result.SessionsMigrated, 
		result.WorkBlocksMigrated, result.ActivitiesMigrated)

	return result, nil
}

// Helper functions for data extraction and conversion

func extractUsers(data *LegacyDatabaseData) ([]*User, error) {
	userMap := make(map[string]*User)
	
	// Extract users from sessions
	for _, session := range data.Sessions {
		if _, exists := userMap[session.UserID]; !exists {
			userMap[session.UserID] = &User{
				ID:       session.UserID,
				Username: session.UserID, // Use ID as username for legacy data
			}
		}
	}

	users := make([]*User, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}

	return users, nil
}

func extractProjects(data *LegacyDatabaseData) ([]*Project, error) {
	projectMap := make(map[string]*Project)

	// Extract projects from work blocks
	for _, workBlock := range data.WorkBlocks {
		if workBlock.ProjectName != "" && workBlock.ProjectPath != "" {
			projectID := generateProjectID(workBlock.ProjectName, workBlock.ProjectPath)
			if _, exists := projectMap[projectID]; !exists {
				projectMap[projectID] = &Project{
					ID:   projectID,
					Name: workBlock.ProjectName,
					Path: workBlock.ProjectPath,
				}
			}
		}
	}

	projects := make([]*Project, 0, len(projectMap))
	for _, project := range projectMap {
		projects = append(projects, project)
	}

	return projects, nil
}

func convertSession(legacy *LegacySession, tz *time.Location) *Session {
	// Ensure end_time is exactly start_time + 5 hours (no IsActive flag needed)
	endTime := legacy.StartTime.Add(5 * time.Hour)
	
	// Convert IsActive flag to state using time-based logic
	state := "finished"
	if legacy.IsActive && time.Now().Before(endTime) {
		state = "active"
	} else if time.Now().After(endTime) {
		state = "expired"
	}

	return &Session{
		ID:                legacy.ID,
		UserID:            legacy.UserID,
		StartTime:         legacy.StartTime.In(tz),
		EndTime:           endTime.In(tz),
		State:             state,
		FirstActivityTime: legacy.FirstActivityTime.In(tz),
		LastActivityTime:  legacy.LastActivityTime.In(tz),
		ActivityCount:     legacy.ActivityCount,
		DurationHours:     5.0, // Always 5 hours for sessions
		CreatedAt:         legacy.CreatedAt.In(tz),
		UpdatedAt:         legacy.UpdatedAt.In(tz),
	}
}

func convertWorkBlock(legacy *LegacyWorkBlock, projectLookup map[string]string, tz *time.Location) *WorkBlock {
	projectID := projectLookup[legacy.ProjectName+"|"+legacy.ProjectPath]
	if projectID == "" {
		projectID = generateProjectID(legacy.ProjectName, legacy.ProjectPath)
	}

	// Convert IsActive flag to state
	state := "finished"
	if legacy.IsActive {
		state = "active"
	}

	// Handle potentially zero end time
	var endTime *time.Time
	if !legacy.EndTime.IsZero() {
		converted := legacy.EndTime.In(tz)
		endTime = &converted
	}

	return &WorkBlock{
		ID:               legacy.ID,
		SessionID:        legacy.SessionID,
		ProjectID:        projectID,
		StartTime:        legacy.StartTime.In(tz),
		EndTime:          endTime,
		State:            state,
		LastActivityTime: legacy.LastActivityTime.In(tz),
		ActivityCount:    legacy.ActivityCount,
		DurationSeconds:  legacy.DurationSeconds,
		DurationHours:    legacy.DurationHours,
		CreatedAt:        legacy.CreatedAt.In(tz),
		UpdatedAt:        legacy.UpdatedAt.In(tz),
	}
}

func convertActivityEvent(legacy *LegacyActivityEvent, projectLookup map[string]string, tz *time.Location) *ActivityEvent {
	projectID := projectLookup[legacy.ProjectName+"|"+legacy.ProjectPath]
	if projectID == "" && legacy.ProjectName != "" {
		projectID = generateProjectID(legacy.ProjectName, legacy.ProjectPath)
	}

	return &ActivityEvent{
		ID:             legacy.ID,
		UserID:         legacy.UserID,
		SessionID:      legacy.SessionID,
		WorkBlockID:    legacy.WorkBlockID,
		ProjectID:      projectID,
		ActivityType:   legacy.ActivityType,
		ActivitySource: legacy.ActivitySource,
		Timestamp:      legacy.Timestamp.In(tz),
		Command:        legacy.Command,
		Description:    legacy.Description,
		Metadata:       legacy.Metadata,
		CreatedAt:      legacy.CreatedAt.In(tz),
	}
}

func createProjectLookup(projects []*Project) map[string]string {
	lookup := make(map[string]string)
	for _, project := range projects {
		key := project.Name + "|" + project.Path
		lookup[key] = project.ID
	}
	return lookup
}

func generateProjectID(name, path string) string {
	// Simple deterministic ID generation
	if name == "" {
		return "unknown-project"
	}
	hash := fmt.Sprintf("%x", []byte(name+path))
	if len(hash) > 12 {
		hash = hash[:12]
	}
	return fmt.Sprintf("proj_%s", hash)
}

func validateMigrationIntegrity(db *SQLiteDB, legacyData *LegacyDatabaseData, result *MigrationResult) error {
	ctx := context.Background()
	
	// Validate session count
	var sessionCount int
	if err := db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&sessionCount); err != nil {
		return fmt.Errorf("failed to count sessions: %w", err)
	}
	if sessionCount != len(legacyData.Sessions) {
		return fmt.Errorf("session count mismatch: expected %d, got %d", len(legacyData.Sessions), sessionCount)
	}

	// Validate work block count
	var workBlockCount int
	if err := db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM work_blocks").Scan(&workBlockCount); err != nil {
		return fmt.Errorf("failed to count work blocks: %w", err)
	}
	if workBlockCount != len(legacyData.WorkBlocks) {
		return fmt.Errorf("work block count mismatch: expected %d, got %d", len(legacyData.WorkBlocks), workBlockCount)
	}

	// Validate activity count (if activities were stored)
	var activityCount int
	if err := db.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM activity_events").Scan(&activityCount); err != nil {
		return fmt.Errorf("failed to count activities: %w", err)
	}
	if activityCount != len(legacyData.Activities) {
		return fmt.Errorf("activity count mismatch: expected %d, got %d", len(legacyData.Activities), activityCount)
	}

	// Validate foreign key relationships
	var orphanedWorkBlocks int
	query := "SELECT COUNT(*) FROM work_blocks wb WHERE NOT EXISTS (SELECT 1 FROM sessions s WHERE s.id = wb.session_id)"
	if err := db.DB().QueryRowContext(ctx, query).Scan(&orphanedWorkBlocks); err != nil {
		return fmt.Errorf("failed to check orphaned work blocks: %w", err)
	}
	if orphanedWorkBlocks > 0 {
		return fmt.Errorf("found %d orphaned work blocks", orphanedWorkBlocks)
	}

	log.Printf("✅ Data integrity validation passed")
	return nil
}

// Data structures for SQLite storage

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Session struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	State             string    `json:"state"`
	FirstActivityTime time.Time `json:"first_activity_time"`
	LastActivityTime  time.Time `json:"last_activity_time"`
	ActivityCount     int64     `json:"activity_count"`
	DurationHours     float64   `json:"duration_hours"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type WorkBlock struct {
	ID                      string     `json:"id"`
	SessionID               string     `json:"session_id"`
	ProjectID               string     `json:"project_id"`
	StartTime               time.Time  `json:"start_time"`
	EndTime                 *time.Time `json:"end_time"`
	State                   string     `json:"state"`
	LastActivityTime        time.Time  `json:"last_activity_time"`
	ActivityCount           int64      `json:"activity_count"`
	DurationSeconds         int64      `json:"duration_seconds"`
	DurationHours           float64    `json:"duration_hours"`
	ClaudeProcessingSeconds int64      `json:"claude_processing_seconds"`
	ClaudeProcessingHours   float64    `json:"claude_processing_hours"`
	EstimatedEndTime        *time.Time `json:"estimated_end_time"`
	LastClaudeActivity      *time.Time `json:"last_claude_activity"`
	ActivePromptID          string     `json:"active_prompt_id"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type ActivityEvent struct {
	ID                      string     `json:"id"`
	UserID                  string     `json:"user_id"`
	SessionID               string     `json:"session_id"`
	WorkBlockID             string     `json:"work_block_id"`
	ProjectID               string     `json:"project_id"`
	ActivityType            string     `json:"activity_type"`
	ActivitySource          string     `json:"activity_source"`
	Timestamp               time.Time  `json:"timestamp"`
	Command                 string     `json:"command"`
	Description             string     `json:"description"`
	Metadata                string     `json:"metadata"`
	ClaudeActivityType      *string    `json:"claude_activity_type"`
	PromptID                string     `json:"prompt_id"`
	EstimatedProcessingTime *int       `json:"estimated_processing_time"`
	ActualProcessingTime    *int       `json:"actual_processing_time"`
	TokensCount             *int       `json:"tokens_count"`
	PromptLength            *int       `json:"prompt_length"`
	ComplexityHint          string     `json:"complexity_hint"`
	CreatedAt               time.Time  `json:"created_at"`
}

// Database insert functions

func insertUser(tx *sql.Tx, user *User) error {
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	query := `
		INSERT OR IGNORE INTO users (id, username, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := tx.Exec(query, user.ID, user.Username, user.CreatedAt, user.UpdatedAt)
	return err
}

func insertProject(tx *sql.Tx, project *Project) error {
	now := time.Now()
	if project.CreatedAt.IsZero() {
		project.CreatedAt = now
	}
	if project.UpdatedAt.IsZero() {
		project.UpdatedAt = now
	}

	query := `
		INSERT OR IGNORE INTO projects (id, name, path, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := tx.Exec(query, project.ID, project.Name, project.Path, 
		project.Description, project.CreatedAt, project.UpdatedAt)
	return err
}

func insertSession(tx *sql.Tx, session *Session) error {
	query := `
		INSERT OR IGNORE INTO sessions 
		(id, user_id, start_time, end_time, state, first_activity_time, last_activity_time,
		 activity_count, duration_hours, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := tx.Exec(query, session.ID, session.UserID, session.StartTime, session.EndTime,
		session.State, session.FirstActivityTime, session.LastActivityTime,
		session.ActivityCount, session.DurationHours, session.CreatedAt, session.UpdatedAt)
	return err
}

func insertWorkBlock(tx *sql.Tx, workBlock *WorkBlock) error {
	query := `
		INSERT OR IGNORE INTO work_blocks 
		(id, session_id, project_id, start_time, end_time, state, last_activity_time,
		 activity_count, duration_seconds, duration_hours, claude_processing_seconds,
		 claude_processing_hours, estimated_end_time, last_claude_activity, active_prompt_id,
		 created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := tx.Exec(query, workBlock.ID, workBlock.SessionID, workBlock.ProjectID,
		workBlock.StartTime, workBlock.EndTime, workBlock.State, workBlock.LastActivityTime,
		workBlock.ActivityCount, workBlock.DurationSeconds, workBlock.DurationHours,
		workBlock.ClaudeProcessingSeconds, workBlock.ClaudeProcessingHours,
		workBlock.EstimatedEndTime, workBlock.LastClaudeActivity, workBlock.ActivePromptID,
		workBlock.CreatedAt, workBlock.UpdatedAt)
	return err
}

func insertActivityEvent(tx *sql.Tx, activity *ActivityEvent) error {
	query := `
		INSERT OR IGNORE INTO activity_events 
		(id, user_id, session_id, work_block_id, project_id, activity_type, activity_source,
		 timestamp, command, description, metadata, claude_activity_type, prompt_id,
		 estimated_processing_time, actual_processing_time, tokens_count, prompt_length,
		 complexity_hint, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := tx.Exec(query, activity.ID, activity.UserID, activity.SessionID,
		activity.WorkBlockID, activity.ProjectID, activity.ActivityType, activity.ActivitySource,
		activity.Timestamp, activity.Command, activity.Description, activity.Metadata,
		activity.ClaudeActivityType, activity.PromptID, activity.EstimatedProcessingTime,
		activity.ActualProcessingTime, activity.TokensCount, activity.PromptLength,
		activity.ComplexityHint, activity.CreatedAt)
	return err
}