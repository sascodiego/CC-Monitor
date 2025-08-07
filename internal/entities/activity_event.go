/**
 * CONTEXT:   Domain entity representing activity events from Claude Code hooks
 * INPUT:     Hook event data including timestamps, project info, and user context
 * OUTPUT:    ActivityEvent entity with validation, normalization, and business rules
 * BUSINESS:  Activity events trigger session and work block updates in the system
 * CHANGE:    Initial implementation following Clean Architecture and SOLID principles
 * RISK:      Low - Domain entity for event data with validation and no external dependencies
 */

package entities

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ActivityType represents different types of user activities
type ActivityType string

const (
	ActivityTypeCommand     ActivityType = "command"
	ActivityTypeFileEdit    ActivityType = "file_edit"
	ActivityTypeFileRead    ActivityType = "file_read"
	ActivityTypeNavigation  ActivityType = "navigation"
	ActivityTypeSearch      ActivityType = "search"
	ActivityTypeGeneration  ActivityType = "generation"
	ActivityTypeOther       ActivityType = "other"
)

// ClaudeActivityType represents Claude processing states
type ClaudeActivityType string

const (
	ClaudeActivityUserAction  ClaudeActivityType = "user_action"    // Normal user activity
	ClaudeActivityStart       ClaudeActivityType = "claude_start"   // Claude begins processing
	ClaudeActivityEnd         ClaudeActivityType = "claude_end"     // Claude finishes processing
	ClaudeActivityProgress    ClaudeActivityType = "claude_progress" // Periodic progress updates
)

// ActivitySource indicates where the activity event originated
type ActivitySource string

const (
	ActivitySourceHook   ActivitySource = "hook"
	ActivitySourceCLI    ActivitySource = "cli"
	ActivitySourceDaemon ActivitySource = "daemon"
	ActivitySourceManual ActivitySource = "manual"
)

/**
 * CONTEXT:   Core activity event entity representing user interactions with Claude Code
 * INPUT:     Activity data from hooks including timing, project, and user information
 * OUTPUT:    Immutable activity event entity with validation and business logic
 * BUSINESS:  Activity events are the primary input for work tracking and session management
 * CHANGE:    Enhanced with Claude processing context for accurate time tracking
 * RISK:      Low - Pure domain logic with validation and no side effects
 */
type ActivityEvent struct {
	id            string
	userID        string
	sessionID     string
	workBlockID   string
	projectPath   string
	projectName   string
	activityType  ActivityType
	activitySource ActivitySource
	timestamp     time.Time
	command       string
	description   string
	metadata      map[string]string
	claudeContext *ClaudeProcessingContext // Enhanced: Claude processing context
	createdAt     time.Time
}

// ClaudeProcessingContext holds context for Claude processing activities
type ClaudeProcessingContext struct {
	PromptID         string        `json:"prompt_id"`          // Unique identifier for request
	EstimatedTime    time.Duration `json:"estimated_time"`     // How long we expect processing
	ActualTime       *time.Duration `json:"actual_time"`       // Actual time when completed
	TokensCount      *int          `json:"tokens_count"`       // Response size if available
	PromptLength     int           `json:"prompt_length"`      // Input prompt character count
	ComplexityHint   string        `json:"complexity_hint"`    // "code_generation", "analysis", etc.
	ClaudeActivity   ClaudeActivityType `json:"claude_activity"` // Type of Claude activity
}

// ActivityEventConfig holds configuration for creating new activity events
type ActivityEventConfig struct {
	UserID            string
	SessionID         string
	WorkBlockID       string
	ProjectPath       string
	ProjectName       string
	ActivityType      ActivityType
	ActivitySource    ActivitySource
	Timestamp         time.Time
	Command           string
	Description       string
	Metadata          map[string]string
	ClaudeContext     *ClaudeProcessingContext // Optional Claude processing context
}

/**
 * CONTEXT:   Factory method for creating new activity event entities with validation
 * INPUT:     ActivityEventConfig with all event data and context
 * OUTPUT:    Valid ActivityEvent entity or validation error
 * BUSINESS:  Activity events must have valid timestamps and user context
 * CHANGE:    Initial implementation with comprehensive validation
 * RISK:      Low - Validation prevents invalid event creation
 */
func NewActivityEvent(config ActivityEventConfig) (*ActivityEvent, error) {
	// Validate required fields
	if config.UserID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	if config.Timestamp.IsZero() {
		return nil, fmt.Errorf("timestamp cannot be zero")
	}

	// Validate timestamp is reasonable
	now := time.Now()
	maxFutureTime := now.Add(5 * time.Minute)
	if config.Timestamp.After(maxFutureTime) {
		return nil, fmt.Errorf("timestamp cannot be more than 5 minutes in future")
	}

	minPastTime := now.Add(-24 * time.Hour)
	if config.Timestamp.Before(minPastTime) {
		return nil, fmt.Errorf("timestamp cannot be more than 24 hours in past")
	}

	// Set defaults for optional fields
	activityType := config.ActivityType
	if activityType == "" {
		activityType = ActivityTypeOther
	}

	activitySource := config.ActivitySource
	if activitySource == "" {
		activitySource = ActivitySourceHook
	}

	// Normalize project path if provided
	normalizedPath := config.ProjectPath
	if normalizedPath != "" {
		var err error
		normalizedPath, err = normalizePath(normalizedPath)
		if err != nil {
			return nil, fmt.Errorf("invalid project path: %w", err)
		}
	}

	// Generate project name if not provided
	projectName := config.ProjectName
	if projectName == "" && normalizedPath != "" {
		projectName = filepath.Base(normalizedPath)
	}

	// Copy metadata to avoid external mutations
	metadata := make(map[string]string)
	for k, v := range config.Metadata {
		metadata[k] = v
	}

	eventID := uuid.New().String()

	event := &ActivityEvent{
		id:             eventID,
		userID:         config.UserID,
		sessionID:      config.SessionID,
		workBlockID:    config.WorkBlockID,
		projectPath:    normalizedPath,
		projectName:    projectName,
		activityType:   activityType,
		activitySource: activitySource,
		timestamp:      config.Timestamp,
		command:        strings.TrimSpace(config.Command),
		description:    strings.TrimSpace(config.Description),
		metadata:       metadata,
		claudeContext:  config.ClaudeContext, // Enhanced: Store Claude context
		createdAt:      now,
	}

	return event, nil
}

/**
 * CONTEXT:   Create activity event from current environment automatically
 * INPUT:     Optional command and description for context
 * OUTPUT:    ActivityEvent with auto-detected user and project information
 * BUSINESS:  Simplify event creation from Claude Code hooks with auto-detection
 * CHANGE:    Initial auto-detection implementation for seamless integration
 * RISK:      Low - Environment-based detection with fallback values
 */
func NewActivityEventFromEnvironment(command, description string) (*ActivityEvent, error) {
	// Auto-detect user ID
	userID := os.Getenv("USER")
	if userID == "" {
		userID = os.Getenv("USERNAME")
	}
	if userID == "" {
		userID = "unknown-user"
	}

	// Auto-detect project path
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = ""
	}

	return NewActivityEvent(ActivityEventConfig{
		UserID:         userID,
		ProjectPath:    workingDir,
		ActivitySource: ActivitySourceHook,
		Timestamp:      time.Now(),
		Command:        command,
		Description:    description,
		Metadata:       make(map[string]string),
	})
}

// Getter methods (immutable access)
func (ae *ActivityEvent) ID() string                     { return ae.id }
func (ae *ActivityEvent) UserID() string                 { return ae.userID }
func (ae *ActivityEvent) SessionID() string              { return ae.sessionID }
func (ae *ActivityEvent) WorkBlockID() string            { return ae.workBlockID }
func (ae *ActivityEvent) ProjectPath() string            { return ae.projectPath }
func (ae *ActivityEvent) ProjectName() string            { return ae.projectName }
func (ae *ActivityEvent) ActivityType() ActivityType     { return ae.activityType }
func (ae *ActivityEvent) ActivitySource() ActivitySource { return ae.activitySource }
func (ae *ActivityEvent) Timestamp() time.Time           { return ae.timestamp }
func (ae *ActivityEvent) Command() string                { return ae.command }
func (ae *ActivityEvent) Description() string            { return ae.description }
func (ae *ActivityEvent) CreatedAt() time.Time           { return ae.createdAt }

func (ae *ActivityEvent) Metadata() map[string]string {
	// Return copy to prevent external mutations
	result := make(map[string]string)
	for k, v := range ae.metadata {
		result[k] = v
	}
	return result
}

func (ae *ActivityEvent) ClaudeContext() *ClaudeProcessingContext {
	if ae.claudeContext == nil {
		return nil
	}
	// Return copy to prevent external mutations
	contextCopy := *ae.claudeContext
	return &contextCopy
}

/**
 * CONTEXT:   Associate activity event with session and work block after processing
 * INPUT:     Session ID and work block ID from business logic processing
 * OUTPUT:    Updated activity event with associations
 * BUSINESS:  Events get associated with sessions and work blocks during processing
 * CHANGE:    Initial association logic for event lifecycle management
 * RISK:      Low - Simple field updates with validation
 */
func (ae *ActivityEvent) AssociateWithSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	ae.sessionID = sessionID
	return nil
}

func (ae *ActivityEvent) AssociateWithWorkBlock(workBlockID string) error {
	if workBlockID == "" {
		return fmt.Errorf("work block ID cannot be empty")
	}

	ae.workBlockID = workBlockID
	return nil
}

/**
 * CONTEXT:   Add metadata to activity event for additional context
 * INPUT:     Key-value metadata pairs for event enrichment
 * OUTPUT:    Error if metadata invalid, nil on success
 * BUSINESS:  Metadata provides additional context for analytics and debugging
 * CHANGE:    Initial metadata management implementation
 * RISK:      Low - Simple key-value storage with validation
 */
func (ae *ActivityEvent) AddMetadata(key, value string) error {
	if key == "" {
		return fmt.Errorf("metadata key cannot be empty")
	}

	if ae.metadata == nil {
		ae.metadata = make(map[string]string)
	}

	ae.metadata[key] = value
	return nil
}

func (ae *ActivityEvent) GetMetadata(key string) (string, bool) {
	if ae.metadata == nil {
		return "", false
	}
	value, exists := ae.metadata[key]
	return value, exists
}

/**
 * CONTEXT:   Calculate time since activity for idle detection and analysis
 * INPUT:     Current timestamp for comparison
 * OUTPUT:    Duration since activity occurred
 * BUSINESS:  Time calculations used for idle detection and work pattern analysis
 * CHANGE:    Initial time calculation utilities
 * RISK:      Low - Simple time arithmetic with no side effects
 */
func (ae *ActivityEvent) Age(currentTime time.Time) time.Duration {
	return currentTime.Sub(ae.timestamp)
}

func (ae *ActivityEvent) IsRecent(currentTime time.Time, threshold time.Duration) bool {
	return ae.Age(currentTime) <= threshold
}

/**
 * CONTEXT:   Check if activity event has required project information
 * INPUT:     No parameters, checks internal project data
 * OUTPUT:    Boolean indicating if project information is available
 * BUSINESS:  Project information required for work block and analytics
 * CHANGE:    Initial project information validation
 * RISK:      Low - Simple field validation with no side effects
 */
func (ae *ActivityEvent) HasProjectInfo() bool {
	return ae.projectPath != "" && ae.projectName != ""
}

func (ae *ActivityEvent) HasSessionInfo() bool {
	return ae.sessionID != ""
}

func (ae *ActivityEvent) HasWorkBlockInfo() bool {
	return ae.workBlockID != ""
}

/**
 * CONTEXT:   Check if activity event contains Claude processing context
 * INPUT:     No parameters, checks internal Claude context data
 * OUTPUT:    Boolean indicating if Claude processing information is available
 * BUSINESS:  Claude context enables smart idle detection during processing
 * CHANGE:    Enhanced Claude processing detection utilities
 * RISK:      Low - Simple field validation with no side effects
 */
func (ae *ActivityEvent) HasClaudeContext() bool {
	return ae.claudeContext != nil
}

func (ae *ActivityEvent) IsClaudeProcessingStart() bool {
	return ae.claudeContext != nil && ae.claudeContext.ClaudeActivity == ClaudeActivityStart
}

func (ae *ActivityEvent) IsClaudeProcessingEnd() bool {
	return ae.claudeContext != nil && ae.claudeContext.ClaudeActivity == ClaudeActivityEnd
}

func (ae *ActivityEvent) IsClaudeProcessingActivity() bool {
	return ae.claudeContext != nil && 
		(ae.claudeContext.ClaudeActivity == ClaudeActivityStart ||
		 ae.claudeContext.ClaudeActivity == ClaudeActivityEnd ||
		 ae.claudeContext.ClaudeActivity == ClaudeActivityProgress)
}

func (ae *ActivityEvent) GetEstimatedProcessingTime() time.Duration {
	if ae.claudeContext == nil {
		return 0
	}
	return ae.claudeContext.EstimatedTime
}

func (ae *ActivityEvent) GetPromptID() string {
	if ae.claudeContext == nil {
		return ""
	}
	return ae.claudeContext.PromptID
}

/**
 * CONTEXT:   Export activity event data for serialization or reporting
 * INPUT:     No parameters, uses internal event state
 * OUTPUT:    ActivityEventData struct suitable for JSON serialization
 * BUSINESS:  Provide read-only view of event data for external use
 * CHANGE:    Initial data export implementation
 * RISK:      Low - Read-only operation with no state changes
 */
type ActivityEventData struct {
	ID               string                    `json:"id"`
	UserID           string                    `json:"user_id"`
	SessionID        string                    `json:"session_id"`
	WorkBlockID      string                    `json:"work_block_id"`
	ProjectPath      string                    `json:"project_path"`
	ProjectName      string                    `json:"project_name"`
	ActivityType     ActivityType              `json:"activity_type"`
	ActivitySource   ActivitySource            `json:"activity_source"`
	Timestamp        time.Time                 `json:"timestamp"`
	Command          string                    `json:"command"`
	Description      string                    `json:"description"`
	Metadata         map[string]string         `json:"metadata"`
	ClaudeContext    *ClaudeProcessingContext  `json:"claude_context"`    // Enhanced: Claude processing context
	HasProjectInfo   bool                      `json:"has_project_info"`
	HasSessionInfo   bool                      `json:"has_session_info"`
	HasWorkBlockInfo bool                      `json:"has_work_block_info"`
	HasClaudeContext bool                      `json:"has_claude_context"` // Enhanced: Claude context flag
	CreatedAt        time.Time                 `json:"created_at"`
}

func (ae *ActivityEvent) ToData() ActivityEventData {
	return ActivityEventData{
		ID:               ae.id,
		UserID:           ae.userID,
		SessionID:        ae.sessionID,
		WorkBlockID:      ae.workBlockID,
		ProjectPath:      ae.projectPath,
		ProjectName:      ae.projectName,
		ActivityType:     ae.activityType,
		ActivitySource:   ae.activitySource,
		Timestamp:        ae.timestamp,
		Command:          ae.command,
		Description:      ae.description,
		Metadata:         ae.Metadata(),     // Use getter for safe copy
		ClaudeContext:    ae.ClaudeContext(), // Enhanced: Use getter for safe copy
		HasProjectInfo:   ae.HasProjectInfo(),
		HasSessionInfo:   ae.HasSessionInfo(),
		HasWorkBlockInfo: ae.HasWorkBlockInfo(),
		HasClaudeContext: ae.HasClaudeContext(), // Enhanced: Claude context flag
		CreatedAt:        ae.createdAt,
	}
}

/**
 * CONTEXT:   Validate activity event internal state consistency
 * INPUT:     No parameters, validates internal state
 * OUTPUT:    Error if state is inconsistent, nil if valid
 * BUSINESS:  Ensure activity event data integrity and business rule compliance
 * CHANGE:    Initial state validation implementation
 * RISK:      Low - Validation only, no state changes
 */
func (ae *ActivityEvent) Validate() error {
	// Check required fields
	if ae.id == "" {
		return fmt.Errorf("activity event ID cannot be empty")
	}
	if ae.userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if ae.timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}

	// Check timestamp validity
	now := time.Now()
	if ae.timestamp.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("timestamp cannot be more than 5 minutes in future")
	}
	if ae.timestamp.Before(now.Add(-24 * time.Hour)) {
		return fmt.Errorf("timestamp cannot be more than 24 hours in past")
	}

	// Check created time
	if ae.createdAt.Before(ae.timestamp) {
		return fmt.Errorf("created time cannot be before activity timestamp")
	}

	return nil
}