/**
 * CONTEXT:   Self-contained data types for Claude Monitor single binary
 * INPUT:     Activity events, sessions, work blocks, and reporting data
 * OUTPUT:    Complete type system embedded in single binary
 * BUSINESS:  Self-contained types eliminate external dependencies 
 * CHANGE:    Initial type definitions extracted from internal packages
 * RISK:      Low - Data structure definitions with no external dependencies
 */

package main

import (
	"fmt"
	"strings"
	"time"
)

/**
 * CONTEXT:   Activity event representation for work hour tracking
 * INPUT:     User activity data from Claude Code hooks
 * OUTPUT:    Structured activity event with metadata
 * BUSINESS:  Activity events are primary input for work tracking system
 * CHANGE:    Self-contained activity event without external entity dependencies
 * RISK:      Low - Simple data structure with validation
 */
type ActivityEvent struct {
	ID             string            `json:"id"`
	UserID         string            `json:"user_id"`
	SessionID      string            `json:"session_id"`
	WorkBlockID    string            `json:"work_block_id"`
	ProjectPath    string            `json:"project_path"`
	ProjectName    string            `json:"project_name"`
	ActivityType   string            `json:"activity_type"`
	ActivitySource string            `json:"activity_source"`
	Timestamp      time.Time         `json:"timestamp"`
	Command        string            `json:"command"`
	Description    string            `json:"description"`
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      time.Time         `json:"created_at"`
}

func NewActivityEvent(config ActivityEventConfig) *ActivityEvent {
	return &ActivityEvent{
		ID:             generateEventID(),
		UserID:         config.UserID,
		SessionID:      "",  // Will be set during processing
		WorkBlockID:    "",  // Will be set during processing
		ProjectPath:    config.ProjectPath,
		ProjectName:    config.ProjectName,
		ActivityType:   config.ActivityType,
		ActivitySource: config.ActivitySource,
		Timestamp:      config.Timestamp,
		Command:        config.Command,
		Description:    config.Description,
		Metadata:       config.Metadata,
		CreatedAt:      time.Now(),
	}
}

type ActivityEventConfig struct {
	UserID         string
	ProjectPath    string
	ProjectName    string
	ActivityType   string
	ActivitySource string
	Timestamp      time.Time
	Command        string
	Description    string
	Metadata       map[string]string
}

/**
 * CONTEXT:   Project type enumeration for categorizing work
 * INPUT:     File system analysis and project structure detection
 * OUTPUT:    Project type classification for analytics
 * BUSINESS:  Project types enable specialized reporting and insights
 * CHANGE:    Self-contained project type system
 * RISK:      Low - Simple enumeration with string constants
 */
type ProjectType string

const (
	ProjectTypeGeneral    ProjectType = "general"
	ProjectTypeGo         ProjectType = "go"
	ProjectTypeRust       ProjectType = "rust"
	ProjectTypeJavaScript ProjectType = "javascript"
	ProjectTypeTypeScript ProjectType = "typescript"
	ProjectTypePython     ProjectType = "python"
	ProjectTypeWeb        ProjectType = "web"
)

/**
 * CONTEXT:   Activity type enumeration for categorizing user actions
 * INPUT:     Different types of user interactions with Claude Code
 * OUTPUT:    Activity classification for detailed analytics
 * BUSINESS:  Activity types enable granular work pattern analysis
 * CHANGE:    Self-contained activity type system
 * RISK:      Low - Simple enumeration with validation
 */
type ActivityType string

const (
	ActivityTypeCommand ActivityType = "command"
	ActivityTypeEdit    ActivityType = "edit"
	ActivityTypeQuery   ActivityType = "query"
	ActivityTypeOther   ActivityType = "other"
)

type ActivitySource string

const (
	ActivitySourceHook   ActivitySource = "hook"
	ActivitySourceManual ActivitySource = "manual"
	ActivitySourceSystem ActivitySource = "system"
)

/**
 * CONTEXT:   Session management for 5-hour work windows
 * INPUT:     Session timing and activity tracking data
 * OUTPUT:    Session object with state management
 * BUSINESS:  Sessions implement 5-hour window business logic
 * CHANGE:    Self-contained session management
 * RISK:      Low - Session state management with validation
 */
type Session struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	State             string    `json:"state"`
	FirstActivityTime time.Time `json:"first_activity_time"`
	LastActivityTime  time.Time `json:"last_activity_time"`
	ActivityCount     int       `json:"activity_count"`
	DurationHours     float64   `json:"duration_hours"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewSession(userID string, startTime time.Time) *Session {
	return &Session{
		ID:                generateSessionID(startTime),
		UserID:            userID,
		StartTime:         startTime,
		EndTime:           startTime.Add(5 * time.Hour), // 5-hour window
		State:             "active",
		FirstActivityTime: startTime,
		LastActivityTime:  startTime,
		ActivityCount:     1,
		DurationHours:     5.0,
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

/**
 * CONTEXT:   Work block management for active work period tracking
 * INPUT:     Work block timing with idle detection logic
 * OUTPUT:    Work block object with duration calculation
 * BUSINESS:  Work blocks track active work periods with 5-minute idle timeout
 * CHANGE:    Self-contained work block management
 * RISK:      Low - Work period tracking with time calculations
 */
type WorkBlock struct {
	ID                   string            `json:"id"`
	SessionID            string            `json:"session_id"`
	ProjectID            string            `json:"project_id"`
	ProjectName          string            `json:"project_name"`
	ProjectPath          string            `json:"project_path"`
	StartTime            time.Time         `json:"start_time"`
	EndTime              time.Time         `json:"end_time"`
	State                string            `json:"state"`
	LastActivityTime     time.Time         `json:"last_activity_time"`
	ActivityCount        int               `json:"activity_count"`
	ActivityTypeCounters map[string]int    `json:"activity_type_counters"`
	DurationSeconds      int               `json:"duration_seconds"`
	DurationHours        float64           `json:"duration_hours"`
	IsActive             bool              `json:"is_active"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

func NewWorkBlock(sessionID, projectName, projectPath string, startTime time.Time) *WorkBlock {
	return &WorkBlock{
		ID:                   generateWorkBlockID(sessionID, startTime),
		SessionID:            sessionID,
		ProjectID:            generateProjectID(projectPath),
		ProjectName:          projectName,
		ProjectPath:          projectPath,
		StartTime:            startTime,
		EndTime:              startTime, // Will be updated
		State:                "active",
		LastActivityTime:     startTime,
		ActivityCount:        1,
		ActivityTypeCounters: make(map[string]int),
		DurationSeconds:      0,
		DurationHours:        0.0,
		IsActive:             true,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

/**
 * CONTEXT:   Project representation for work organization
 * INPUT:     Project information from directory analysis
 * OUTPUT:    Project object with metadata and statistics
 * BUSINESS:  Projects provide organizational structure for work analytics
 * CHANGE:    Self-contained project management
 * RISK:      Low - Project metadata with path handling
 */
type Project struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Path             string      `json:"path"`
	NormalizedPath   string      `json:"normalized_path"`
	ProjectType      ProjectType `json:"project_type"`
	Description      string      `json:"description"`
	LastActiveTime   time.Time   `json:"last_active_time"`
	TotalWorkBlocks  int         `json:"total_work_blocks"`
	TotalHours       float64     `json:"total_hours"`
	IsActive         bool        `json:"is_active"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

func NewProject(name, path string, projectType ProjectType) *Project {
	return &Project{
		ID:              generateProjectID(path),
		Name:            name,
		Path:            path,
		NormalizedPath:  normalizePath(path),
		ProjectType:     projectType,
		Description:     "",
		LastActiveTime:  time.Now(),
		TotalWorkBlocks: 0,
		TotalHours:      0.0,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

/**
 * CONTEXT:   User representation for work tracking ownership
 * INPUT:     User identification from environment
 * OUTPUT:    User object for session and work ownership
 * BUSINESS:  Users own sessions and work blocks for access control
 * CHANGE:    Self-contained user management
 * RISK:      Low - Simple user identification
 */
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUser(id, name string) *User {
	return &User{
		ID:        id,
		Name:      name,
		Email:     "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Utility functions for ID generation and path handling

func generateEventID() string {
	return fmt.Sprintf("event_%d_%s", time.Now().UnixNano(), generateRandomString(8))
}

func generateSessionID(startTime time.Time) string {
	return fmt.Sprintf("session_%s_%s", startTime.Format("20060102_150405"), generateRandomString(6))
}

func generateWorkBlockID(sessionID string, startTime time.Time) string {
	return fmt.Sprintf("wb_%s_%s", sessionID[8:], startTime.Format("150405"))
}

func generateProjectID(path string) string {
	// Simple hash-based project ID
	return fmt.Sprintf("project_%x", hashString(path))
}

func normalizePath(path string) string {
	// Normalize path for consistent comparison
	return strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func hashString(s string) uint32 {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

/**
 * CONTEXT:   Reporting data structures for daily, weekly, and monthly analytics
 * INPUT:     Aggregated work data from database queries
 * OUTPUT:    Structured reports for CLI display and JSON export
 * BUSINESS:  Reporting types enable comprehensive work analytics and insights
 * CHANGE:    Initial reporting structures for analytics features
 * RISK:      Low - Data structure definitions for reporting functionality
 */

type DailyReport struct {
	Date             time.Time         `json:"date"`
	StartTime        time.Time         `json:"start_time"`
	EndTime          time.Time         `json:"end_time"`
	TotalWorkHours   float64           `json:"total_work_hours"`
	TotalSessions    int               `json:"total_sessions"`
	TotalWorkBlocks  int               `json:"total_work_blocks"`
	ProjectBreakdown []ProjectSummary  `json:"project_breakdown"`
	WorkBlocks       []WorkBlockSummary `json:"work_blocks"`
	Insights         []string          `json:"insights"`
	ActivitySummary  ActivitySummary   `json:"activity_summary"`
}

type ProjectSummary struct {
	Name       string  `json:"name"`
	Hours      float64 `json:"hours"`
	Percentage float64 `json:"percentage"`
	WorkBlocks int     `json:"work_blocks"`
}

type ActivitySummary struct {
	CommandCount int `json:"command_count"`
	EditCount    int `json:"edit_count"`
	QueryCount   int `json:"query_count"`
	OtherCount   int `json:"other_count"`
}

type WorkBlockSummary struct {
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	ProjectName string        `json:"project_name"`
	Activities  int           `json:"activities"`
}

type DayActivity struct {
	Date              time.Time `json:"date"`
	WorkHours         float64   `json:"work_hours"`
	ProductivityLevel string    `json:"productivity_level"`
	WorkBlocks        int       `json:"work_blocks"`
}