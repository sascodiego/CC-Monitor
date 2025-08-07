/**
 * CONTEXT:   KuzuDB-compatible database connection for Claude Monitor
 * INPUT:     Activity events, sessions, work blocks for persistent storage
 * OUTPUT:    Database operations with proper persistence to ~/.claude/monitor/monitor.db
 * BUSINESS:  Work hour tracking with persistent database storage (NO JSON)
 * CHANGE:    Simple binary format persistence instead of JSON, KuzuDB-ready structure
 * RISK:      Low - Simple binary persistence with KuzuDB-compatible data structure
 */

package database

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// KuzuDBConnection manages the database connection and operations
type KuzuDBConnection struct {
	dbPath     string
	mu         sync.RWMutex
	initialized bool
	data       *DatabaseData
}

// DatabaseData represents the in-memory database structure
type DatabaseData struct {
	Sessions    map[string]*Session      `gob:"sessions"`
	WorkBlocks  map[string]*WorkBlock    `gob:"work_blocks"`
	Activities  []*ActivityEvent         `gob:"activities"`
	LastUpdated time.Time                `gob:"last_updated"`
	Version     string                   `gob:"version"`
}

// NewKuzuDBConnection creates a new KuzuDB connection
func NewKuzuDBConnection(dbPath string) *KuzuDBConnection {
	return &KuzuDBConnection{
		dbPath: dbPath,
		data: &DatabaseData{
			Sessions:   make(map[string]*Session),
			WorkBlocks: make(map[string]*WorkBlock),
			Activities: make([]*ActivityEvent, 0),
			Version:    "1.0.0",
		},
	}
}

// Initialize sets up the database
func (k *KuzuDBConnection) Initialize() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Create directory if it doesn't exist
	dbDir := filepath.Dir(k.dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	log.Printf("📊 Initializing KuzuDB-compatible database at: %s", k.dbPath)

	// Load existing data from binary format
	if err := k.loadDataFromBinary(); err != nil {
		// Initialize empty database if load fails
		k.data = &DatabaseData{
			Sessions:    make(map[string]*Session),
			WorkBlocks:  make(map[string]*WorkBlock),
			Activities:  make([]*ActivityEvent, 0),
			Version:     "1.0.0",
			LastUpdated: time.Now(),
		}
		log.Printf("📊 Created new KuzuDB-compatible database")
	} else {
		log.Printf("📊 Loaded existing KuzuDB-compatible database with %d sessions", len(k.data.Sessions))
	}

	k.initialized = true
	return nil
}

// SaveSession persists a session to the database
func (k *KuzuDBConnection) SaveSession(session *Session) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.initialized {
		return fmt.Errorf("database not initialized")
	}

	k.data.Sessions[session.ID] = session
	k.data.LastUpdated = time.Now()

	return k.saveDataToBinary()
}

// SaveWorkBlock persists a work block to the database
func (k *KuzuDBConnection) SaveWorkBlock(workBlock *WorkBlock) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.initialized {
		return fmt.Errorf("database not initialized")
	}

	k.data.WorkBlocks[workBlock.ID] = workBlock
	k.data.LastUpdated = time.Now()

	return k.saveDataToBinary()
}

// SaveActivity NO LONGER STORES INDIVIDUAL ACTIVITIES - ONLY COUNTS OPERATIONS
// This method is deprecated - use UpdateWorkBlockActivity instead
func (k *KuzuDBConnection) SaveActivity(activity *ActivityEvent) error {
	// NO-OP: No almacenamos actividades individuales, solo contamos operaciones
	return nil
}

// UpdateWorkBlockActivity updates activity count and type counters for a work block
func (k *KuzuDBConnection) UpdateWorkBlockActivity(workBlockID string, activityType string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.initialized {
		return fmt.Errorf("database not initialized")
	}

	workBlock := k.data.WorkBlocks[workBlockID]
	if workBlock == nil {
		return fmt.Errorf("work block not found: %s", workBlockID)
	}

	// Update activity counters based on type
	workBlock.ActivityCount++
	workBlock.LastActivityTime = time.Now()
	workBlock.UpdatedAt = time.Now()

	// Update activity type counters
	if workBlock.ActivityTypeCounters == nil {
		workBlock.ActivityTypeCounters = make(map[string]int64)
	}
	workBlock.ActivityTypeCounters[activityType]++

	k.data.LastUpdated = time.Now()
	return k.saveDataToBinary()
}

// ConvertServerWorkBlockToDatabase converts from server types to database types
func ConvertServerWorkBlockToDatabase(serverWB interface{}) *WorkBlock {
	// Accept the server WorkBlock struct directly
	switch wb := serverWB.(type) {
	case map[string]interface{}:
		// Handle JSON-like interface format
		dbWB := &WorkBlock{
			ID:               getStringFromInterface(wb["id"]),
			SessionID:        getStringFromInterface(wb["session_id"]),
			ProjectName:      getStringFromInterface(wb["project_name"]),
			ProjectPath:      getStringFromInterface(wb["project_path"]),
			StartTime:        getTimeFromInterface(wb["start_time"]),
			EndTime:          getTimeFromInterface(wb["end_time"]),
			State:            getStringFromInterface(wb["state"]),
			LastActivityTime: getTimeFromInterface(wb["last_activity_time"]),
			ActivityCount:    int64(getIntFromInterface(wb["activity_count"])),
			DurationSeconds:  int64(getIntFromInterface(wb["duration_seconds"])),
			DurationHours:    getFloat64FromInterface(wb["duration_hours"]),
			IsActive:         getBoolFromInterface(wb["is_active"]),
			CreatedAt:        getTimeFromInterface(wb["created_at"]),
			UpdatedAt:        getTimeFromInterface(wb["updated_at"]),
		}
		
		// Convert activity type counters
		if counters, ok := wb["activity_type_counters"].(map[string]int); ok {
			dbWB.ActivityTypeCounters = make(map[string]int64)
			for k, v := range counters {
				dbWB.ActivityTypeCounters[k] = int64(v)
			}
		}
		return dbWB
	default:
		return nil
	}
}

// GetActiveSessions retrieves all active sessions
func (k *KuzuDBConnection) GetActiveSessions() ([]*Session, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if !k.initialized {
		return []*Session{}, fmt.Errorf("database not initialized")
	}

	var activeSessions []*Session
	for _, session := range k.data.Sessions {
		if session.IsActive {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions, nil
}

// GetWorkBlocksByProject retrieves work blocks for a specific project
func (k *KuzuDBConnection) GetWorkBlocksByProject(projectName string, startTime, endTime time.Time) ([]*WorkBlock, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if !k.initialized {
		return []*WorkBlock{}, fmt.Errorf("database not initialized")
	}

	var projectWorkBlocks []*WorkBlock
	for _, workBlock := range k.data.WorkBlocks {
		if workBlock.ProjectName == projectName &&
			workBlock.StartTime.After(startTime.Add(-time.Second)) &&
			workBlock.StartTime.Before(endTime.Add(time.Second)) {
			projectWorkBlocks = append(projectWorkBlocks, workBlock)
		}
	}

	return projectWorkBlocks, nil
}

// GetDailyReport generates a daily work report
func (k *KuzuDBConnection) GetDailyReport(date time.Time) (*DailyReport, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if !k.initialized {
		return nil, fmt.Errorf("database not initialized")
	}

	// Work day starts at 5:00 AM and ends at 5:00 AM next day (America/Montevideo)
	montevideoTZ, _ := time.LoadLocation("America/Montevideo")
	startOfWorkDay := time.Date(date.Year(), date.Month(), date.Day(), 5, 0, 0, 0, montevideoTZ)
	endOfWorkDay := startOfWorkDay.Add(24 * time.Hour)

	// Calculate project breakdown
	projectTimes := make(map[string]*ProjectTime)
	totalWorkHours := 0.0
	totalWorkBlocks := 0

	for _, workBlock := range k.data.WorkBlocks {
		if workBlock.StartTime.After(startOfWorkDay.Add(-time.Second)) && 
		   workBlock.StartTime.Before(endOfWorkDay.Add(time.Second)) {
			projectName := workBlock.ProjectName
			if projectName == "" {
				projectName = "Unknown"
			}

			if projectTimes[projectName] == nil {
				projectTimes[projectName] = &ProjectTime{
					Name:       projectName,
					Hours:      0.0,
					WorkBlocks: 0,
				}
			}

			projectTimes[projectName].Hours += workBlock.DurationHours
			projectTimes[projectName].WorkBlocks++
			totalWorkHours += workBlock.DurationHours
			totalWorkBlocks++
		}
	}

	// Convert to slice and calculate percentages
	var projectBreakdown []*ProjectTime
	for _, pt := range projectTimes {
		if totalWorkHours > 0 {
			pt.Percentage = (pt.Hours / totalWorkHours) * 100
		}
		projectBreakdown = append(projectBreakdown, pt)
	}

	// Count activities from work block counters
	activitySummary := ActivitySummary{}
	for _, workBlock := range k.data.WorkBlocks {
		if workBlock.StartTime.After(startOfWorkDay.Add(-time.Second)) && 
		   workBlock.StartTime.Before(endOfWorkDay.Add(time.Second)) {
			
			if workBlock.ActivityTypeCounters != nil {
				activitySummary.CommandCount += int(workBlock.ActivityTypeCounters["command"])
				activitySummary.EditCount += int(workBlock.ActivityTypeCounters["edit"])
				activitySummary.QueryCount += int(workBlock.ActivityTypeCounters["query"])
				activitySummary.OtherCount += int(workBlock.ActivityTypeCounters["other"])
				activitySummary.PostCount += int(workBlock.ActivityTypeCounters["post"])
				activitySummary.HttpCount += int(workBlock.ActivityTypeCounters["http"])
			}
		}
	}

	insights := []string{}
	if totalWorkHours > 0 {
		if totalWorkHours >= 4 {
			insights = append(insights, "Excellent focus! Good work session today")
		} else if totalWorkHours >= 1 {
			insights = append(insights, "Good productivity session")
		} else {
			insights = append(insights, "Short work session - consider longer focus periods")
		}

		if len(projectBreakdown) == 1 {
			insights = append(insights, "Single project focus - excellent for deep work")
		} else if len(projectBreakdown) > 3 {
			insights = append(insights, "Multiple projects - consider reducing context switching")
		}
	}

	return &DailyReport{
		Date:             date,
		TotalWorkHours:   totalWorkHours,
		TotalSessions:    len(k.data.Sessions),
		TotalWorkBlocks:  totalWorkBlocks,
		ProjectBreakdown: projectBreakdown,
		Insights:         insights,
		ActivitySummary:  activitySummary,
	}, nil
}

// GetWorkBlocksWithCounters returns work blocks with their activity type counters
func (k *KuzuDBConnection) GetWorkBlocksWithCounters() ([]*WorkBlock, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	
	if !k.initialized || k.data == nil {
		return []*WorkBlock{}, nil
	}
	
	var workBlocks []*WorkBlock
	for _, wb := range k.data.WorkBlocks {
		workBlocks = append(workBlocks, wb)
	}
	
	return workBlocks, nil
}

// GetDatabaseStats returns comprehensive database statistics
func (k *KuzuDBConnection) GetDatabaseStats() (map[string]interface{}, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	
	if !k.initialized || k.data == nil {
		return map[string]interface{}{
			"total_sessions":    0,
			"total_work_blocks": 0,
			"total_activities":  0,
			"active_sessions":   0,
			"active_work_blocks": 0,
		}, nil
	}
	
	// Count active sessions and work blocks
	activeSessions := 0
	activeWorkBlocks := 0
	
	for _, session := range k.data.Sessions {
		if session.IsActive {
			activeSessions++
		}
	}
	for _, workBlock := range k.data.WorkBlocks {
		if workBlock.IsActive {
			activeWorkBlocks++
		}
	}
	
	return map[string]interface{}{
		"total_sessions":    len(k.data.Sessions),
		"total_work_blocks": len(k.data.WorkBlocks),
		"total_activities":  len(k.data.Activities),
		"active_sessions":   activeSessions,
		"active_work_blocks": activeWorkBlocks,
		"last_updated":      k.data.LastUpdated,
		"version":          k.data.Version,
	}, nil
}

// Close closes the database connection
func (k *KuzuDBConnection) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.initialized {
		// Save final data
		if err := k.saveDataToBinary(); err != nil {
			log.Printf("Warning: failed to save data on close: %v", err)
		}
	}

	log.Printf("📊 Closed KuzuDB-compatible database connection")
	k.initialized = false
	return nil
}

// Internal binary persistence methods (NO JSON)

func (k *KuzuDBConnection) loadDataFromBinary() error {
	if _, err := os.Stat(k.dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database file does not exist")
	}

	file, err := os.Open(k.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database file: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&k.data); err != nil {
		return fmt.Errorf("failed to decode database: %w", err)
	}

	// Initialize maps if nil
	if k.data.Sessions == nil {
		k.data.Sessions = make(map[string]*Session)
	}
	if k.data.WorkBlocks == nil {
		k.data.WorkBlocks = make(map[string]*WorkBlock)
	}
	if k.data.Activities == nil {
		k.data.Activities = make([]*ActivityEvent, 0)
	}

	return nil
}

func (k *KuzuDBConnection) saveDataToBinary() error {
	// Write to temporary file first for atomic operation
	tempFile := k.dbPath + ".tmp"
	
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp database file: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(k.data); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to encode database: %w", err)
	}

	file.Close()

	// Atomic rename
	if err := os.Rename(tempFile, k.dbPath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp database: %w", err)
	}

	return nil
}

// Data structures for KuzuDB compatibility

type Session struct {
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

type WorkBlock struct {
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

type ActivityEvent struct {
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

type DailyReport struct {
	Date             time.Time      `gob:"date"`
	TotalWorkHours   float64        `gob:"total_work_hours"`
	TotalSessions    int            `gob:"total_sessions"`
	TotalWorkBlocks  int            `gob:"total_work_blocks"`
	ProjectBreakdown []*ProjectTime `gob:"project_breakdown"`
	Insights         []string       `gob:"insights"`
	ActivitySummary  ActivitySummary `gob:"activity_summary"`
}

type ProjectTime struct {
	Name       string  `gob:"name"`
	Hours      float64 `gob:"hours"`
	Percentage float64 `gob:"percentage"`
	WorkBlocks int     `gob:"work_blocks"`
}

type ActivitySummary struct {
	CommandCount int `gob:"command_count"`
	EditCount    int `gob:"edit_count"`
	QueryCount   int `gob:"query_count"`
	PostCount    int `gob:"post_count"`
	HttpCount    int `gob:"http_count"`
	OtherCount   int `gob:"other_count"`
}

// Helper functions for type conversion between server and database types

func getStringFromInterface(i interface{}) string {
	if s, ok := i.(string); ok {
		return s
	}
	return ""
}

func getIntFromInterface(i interface{}) int {
	if n, ok := i.(int); ok {
		return n
	}
	if n, ok := i.(float64); ok {
		return int(n)
	}
	return 0
}

func getFloat64FromInterface(i interface{}) float64 {
	if f, ok := i.(float64); ok {
		return f
	}
	if f, ok := i.(int); ok {
		return float64(f)
	}
	return 0.0
}

func getBoolFromInterface(i interface{}) bool {
	if b, ok := i.(bool); ok {
		return b
	}
	return false
}

func getTimeFromInterface(i interface{}) time.Time {
	if t, ok := i.(time.Time); ok {
		return t
	}
	return time.Time{}
}