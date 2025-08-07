/**
 * CONTEXT:   Domain entity representing work blocks with 5-minute idle detection business logic
 * INPUT:     Session ID, project information, and timing data for work tracking
 * OUTPUT:    WorkBlock entity with validation, duration calculation, and state management
 * BUSINESS:  Work blocks track active work periods, ending after 5 minutes of inactivity
 * CHANGE:    Initial implementation following Clean Architecture and SOLID principles
 * RISK:      Low - Domain entity with pure business logic, no external dependencies
 */

package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WorkBlockState represents the current state of a work block
type WorkBlockState string

const (
	WorkBlockStateActive     WorkBlockState = "active"
	WorkBlockStateIdle       WorkBlockState = "idle"
	WorkBlockStateProcessing WorkBlockState = "processing" // Enhanced: Claude is working
	WorkBlockStateFinished   WorkBlockState = "finished"
)

// IdleTimeout defines when a work block is considered idle (5 minutes)
const IdleTimeout = 5 * time.Minute

/**
 * CONTEXT:   Core work block entity implementing idle detection and time tracking
 * INPUT:     Work block data with session reference, project info, and timing
 * OUTPUT:    Immutable work block entity with business logic methods
 * BUSINESS:  Work blocks represent continuous work periods within sessions
 * CHANGE:    Enhanced with Claude processing state and time tracking
 * RISK:      Low - Pure domain logic with validation and no side effects
 */
type WorkBlock struct {
	id                   string
	sessionID            string
	projectID            string
	projectName          string
	projectPath          string
	startTime            time.Time
	endTime              time.Time
	state                WorkBlockState
	lastActivityTime     time.Time
	activityCount        int64
	createdAt            time.Time
	updatedAt            time.Time
	// Enhanced: Claude processing tracking
	claudeProcessingTime time.Duration // Time Claude was actively processing
	estimatedEndTime     *time.Time    // When we expect Claude to finish processing
	lastClaudeActivity   *time.Time    // Last Claude processing event
	activePromptID       string        // Current processing prompt ID
}

// WorkBlockConfig holds configuration for creating new work blocks
type WorkBlockConfig struct {
	SessionID   string
	ProjectID   string
	ProjectName string
	ProjectPath string
	StartTime   time.Time
}

/**
 * CONTEXT:   Factory method for creating new work block entities with validation
 * INPUT:     WorkBlockConfig with session, project, and timing information
 * OUTPUT:    Valid WorkBlock entity or validation error
 * BUSINESS:  New work blocks start active and track time within session boundaries
 * CHANGE:    Initial implementation with comprehensive validation
 * RISK:      Low - Validation prevents invalid work block creation
 */
func NewWorkBlock(config WorkBlockConfig) (*WorkBlock, error) {
	// Validate required fields
	if config.SessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	if config.ProjectName == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	if config.StartTime.IsZero() {
		return nil, fmt.Errorf("start time cannot be zero")
	}

	// Validate start time is reasonable
	maxFutureTime := time.Now().Add(5 * time.Minute)
	if config.StartTime.After(maxFutureTime) {
		return nil, fmt.Errorf("start time cannot be more than 5 minutes in the future")
	}

	minPastTime := time.Now().Add(-24 * time.Hour)
	if config.StartTime.Before(minPastTime) {
		return nil, fmt.Errorf("start time cannot be more than 24 hours in the past")
	}

	now := time.Now()
	workBlockID := uuid.New().String()

	// Generate project ID if not provided
	projectID := config.ProjectID
	if projectID == "" {
		projectID = generateProjectID(config.ProjectName)
	}

	workBlock := &WorkBlock{
		id:               workBlockID,
		sessionID:        config.SessionID,
		projectID:        projectID,
		projectName:      config.ProjectName,
		projectPath:      config.ProjectPath,
		startTime:        config.StartTime,
		endTime:          time.Time{}, // Will be set when finished
		state:            WorkBlockStateActive,
		lastActivityTime: config.StartTime,
		activityCount:    1,
		createdAt:        now,
		updatedAt:        now,
	}

	return workBlock, nil
}

// Getter methods (immutable access)
func (wb *WorkBlock) ID() string                    { return wb.id }
func (wb *WorkBlock) SessionID() string             { return wb.sessionID }
func (wb *WorkBlock) ProjectID() string             { return wb.projectID }
func (wb *WorkBlock) ProjectName() string           { return wb.projectName }
func (wb *WorkBlock) ProjectPath() string           { return wb.projectPath }
func (wb *WorkBlock) StartTime() time.Time          { return wb.startTime }
func (wb *WorkBlock) EndTime() time.Time            { return wb.endTime }
func (wb *WorkBlock) State() WorkBlockState         { return wb.state }
func (wb *WorkBlock) LastActivityTime() time.Time  { return wb.lastActivityTime }
func (wb *WorkBlock) ActivityCount() int64          { return wb.activityCount }
func (wb *WorkBlock) CreatedAt() time.Time          { return wb.createdAt }
func (wb *WorkBlock) UpdatedAt() time.Time          { return wb.updatedAt }

// Enhanced: Claude processing getter methods
func (wb *WorkBlock) ClaudeProcessingTime() time.Duration { return wb.claudeProcessingTime }
func (wb *WorkBlock) EstimatedEndTime() *time.Time {
	if wb.estimatedEndTime == nil {
		return nil
	}
	endTime := *wb.estimatedEndTime
	return &endTime
}
func (wb *WorkBlock) LastClaudeActivity() *time.Time {
	if wb.lastClaudeActivity == nil {
		return nil
	}
	claudeTime := *wb.lastClaudeActivity
	return &claudeTime
}
func (wb *WorkBlock) ActivePromptID() string { return wb.activePromptID }

/**
 * CONTEXT:   Enhanced idle detection that considers Claude processing states
 * INPUT:     Current timestamp for comparison with last activity and processing state
 * OUTPUT:    Boolean indicating if work block should be considered idle
 * BUSINESS:  Work block is NOT idle if Claude is actively processing, even without user activity
 * CHANGE:    Enhanced logic to prevent false idle detection during Claude processing
 * RISK:      Medium - Critical logic that affects work time accuracy, handles processing timeouts
 */
func (wb *WorkBlock) IsIdle(currentTime time.Time) bool {
	// Non-active states return their idle status directly
	if wb.state == WorkBlockStateFinished {
		return false
	}
	if wb.state == WorkBlockStateIdle {
		return true
	}
	
	// If Claude is currently processing, NOT idle
	if wb.state == WorkBlockStateProcessing {
		return wb.isClaudeProcessingTimedOut(currentTime)
	}
	
	// Regular idle detection for active state (5 minutes)
	if wb.state == WorkBlockStateActive {
		timeSinceLastActivity := currentTime.Sub(wb.lastActivityTime)
		return timeSinceLastActivity > IdleTimeout
	}
	
	return true
}

/**
 * CONTEXT:   Check if Claude processing has timed out beyond reasonable expectations
 * INPUT:     Current timestamp to compare with estimated completion time
 * OUTPUT:    Boolean indicating if Claude processing should be considered timed out
 * BUSINESS:  Detect hung Claude processing that might indicate a problem
 * CHANGE:    Initial timeout detection for Claude processing states
 * RISK:      Medium - Timeout logic prevents infinite processing states
 */
func (wb *WorkBlock) isClaudeProcessingTimedOut(currentTime time.Time) bool {
	// If no estimated end time, use last Claude activity + reasonable buffer
	if wb.estimatedEndTime == nil {
		if wb.lastClaudeActivity == nil {
			// No Claude activity recorded, shouldn't be in processing state
			return true
		}
		// Use 10 minutes as maximum processing time without estimate
		maxProcessingTime := 10 * time.Minute
		return currentTime.Sub(*wb.lastClaudeActivity) > maxProcessingTime
	}
	
	// Check if we're past estimated completion time
	if currentTime.Before(*wb.estimatedEndTime) {
		return false // Still within estimated time
	}
	
	// We're past estimated time - add grace period (50% of original estimate)
	originalEstimate := wb.estimatedEndTime.Sub(*wb.lastClaudeActivity)
	graceTime := time.Duration(float64(originalEstimate) * 0.5)
	graceEndTime := wb.estimatedEndTime.Add(graceTime)
	
	// Considered timed out if past grace period
	return currentTime.After(graceEndTime)
}

/**
 * CONTEXT:   Check if work block should start a new block due to idle timeout
 * INPUT:     Activity timestamp to check against last activity
 * OUTPUT:    Boolean indicating if new work block should be created
 * BUSINESS:  New work block starts if gap between activities > 5 minutes
 * CHANGE:    Initial new block detection logic
 * RISK:      Low - Time gap calculation with validation
 */
func (wb *WorkBlock) ShouldStartNewBlock(activityTime time.Time) bool {
	if wb.state != WorkBlockStateActive {
		return true
	}

	gap := activityTime.Sub(wb.lastActivityTime)
	return gap > IdleTimeout
}

/**
 * CONTEXT:   Calculate work block duration and timing metrics
 * INPUT:     Optional current time for duration calculation
 * OUTPUT:    Duration metrics for active work time
 * BUSINESS:  Duration calculated from start to end (or current time if active)
 * CHANGE:    Initial duration calculation implementation
 * RISK:      Low - Pure calculation based on timestamps
 */
func (wb *WorkBlock) Duration() time.Duration {
	if wb.endTime.IsZero() {
		// Work block is still active, calculate duration to now
		return time.Since(wb.startTime)
	}
	return wb.endTime.Sub(wb.startTime)
}

func (wb *WorkBlock) DurationAt(currentTime time.Time) time.Duration {
	if wb.endTime.IsZero() {
		// Work block is still active
		if currentTime.Before(wb.startTime) {
			return 0
		}
		return currentTime.Sub(wb.startTime)
	}
	return wb.endTime.Sub(wb.startTime)
}

func (wb *WorkBlock) DurationHours() float64 {
	return wb.Duration().Hours()
}

func (wb *WorkBlock) DurationSeconds() int64 {
	return int64(wb.Duration().Seconds())
}

/**
 * CONTEXT:   Check if work block is currently active
 * INPUT:     No parameters, uses current state and time
 * OUTPUT:    Boolean indicating if work block is active
 * BUSINESS:  Work block is active if state is active and not idle
 * CHANGE:    Added IsActive method for compatibility
 * RISK:      Low - Simple state checking
 */
func (wb *WorkBlock) IsActive() bool {
	return wb.state == WorkBlockStateActive && !wb.IsIdle(time.Now())
}

/**
 * CONTEXT:   Start Claude processing within the work block
 * INPUT:     Processing start time, estimated duration, and prompt ID
 * OUTPUT:    Error if state transition invalid, nil on success
 * BUSINESS:  Claude processing prevents work block from being marked as idle
 * CHANGE:    Enhanced Claude processing state management
 * RISK:      Medium - Processing state affects idle detection and work timing
 */
func (wb *WorkBlock) StartClaudeProcessing(startTime time.Time, estimatedDuration time.Duration, promptID string) error {
	// Validate processing can be started
	if wb.state == WorkBlockStateFinished {
		return fmt.Errorf("cannot start Claude processing on finished work block")
	}
	
	// Validate start time
	if startTime.Before(wb.startTime) {
		return fmt.Errorf("Claude processing start time %v cannot be before work block start %v",
			startTime, wb.startTime)
	}
	
	maxFutureTime := time.Now().Add(5 * time.Minute)
	if startTime.After(maxFutureTime) {
		return fmt.Errorf("Claude processing start time %v cannot be more than 5 minutes in future",
			startTime)
	}
	
	// Update Claude processing state
	wb.state = WorkBlockStateProcessing
	wb.lastClaudeActivity = &startTime
	wb.activePromptID = promptID
	
	// Set estimated end time
	estimatedEnd := startTime.Add(estimatedDuration)
	wb.estimatedEndTime = &estimatedEnd
	
	// Also update last activity time (user initiated this)
	wb.lastActivityTime = startTime
	wb.activityCount++
	wb.updatedAt = time.Now()
	
	return nil
}

/**
 * CONTEXT:   End Claude processing and record actual processing time
 * INPUT:     Processing end time and actual processing duration
 * OUTPUT:    Error if state transition invalid, nil on success  
 * BUSINESS:  Record Claude processing time separately from user interaction time
 * CHANGE:    Enhanced processing completion with time tracking
 * RISK:      Medium - Processing time calculations affect work metrics
 */
func (wb *WorkBlock) EndClaudeProcessing(endTime time.Time) error {
	// Validate processing can be ended
	if wb.state != WorkBlockStateProcessing {
		return fmt.Errorf("work block is not in processing state, current state: %s", wb.state)
	}
	
	if wb.lastClaudeActivity == nil {
		return fmt.Errorf("no Claude processing start time recorded")
	}
	
	// Validate end time
	if endTime.Before(*wb.lastClaudeActivity) {
		return fmt.Errorf("Claude processing end time %v cannot be before start time %v",
			endTime, *wb.lastClaudeActivity)
	}
	
	maxFutureTime := time.Now().Add(5 * time.Minute)
	if endTime.After(maxFutureTime) {
		return fmt.Errorf("Claude processing end time %v cannot be more than 5 minutes in future", endTime)
	}
	
	// Calculate and record actual processing time
	actualProcessingTime := endTime.Sub(*wb.lastClaudeActivity)
	wb.claudeProcessingTime += actualProcessingTime
	
	// Reset processing state
	wb.state = WorkBlockStateActive
	wb.estimatedEndTime = nil
	wb.activePromptID = ""
	
	// Update last activity time (Claude finished, user can see response)
	wb.lastActivityTime = endTime
	wb.activityCount++
	wb.updatedAt = time.Now()
	
	return nil
}

/**
 * CONTEXT:   Update Claude processing progress with periodic heartbeat
 * INPUT:     Progress timestamp for keeping processing state alive
 * OUTPUT:    Error if update invalid, nil on success
 * BUSINESS:  Prevent processing timeout during long Claude operations
 * CHANGE:    Enhanced processing progress tracking
 * RISK:      Low - Progress updates maintain accurate processing state
 */
func (wb *WorkBlock) UpdateClaudeProcessingProgress(progressTime time.Time) error {
	if wb.state != WorkBlockStateProcessing {
		return fmt.Errorf("work block is not in processing state")
	}
	
	if wb.lastClaudeActivity == nil {
		return fmt.Errorf("no Claude processing start time recorded")
	}
	
	// Validate progress time
	if progressTime.Before(*wb.lastClaudeActivity) {
		return fmt.Errorf("Claude processing progress time cannot be before start time")
	}
	
	maxFutureTime := time.Now().Add(5 * time.Minute)
	if progressTime.After(maxFutureTime) {
		return fmt.Errorf("Claude processing progress time cannot be more than 5 minutes in future")
	}
	
	// Update last Claude activity to extend timeout
	wb.lastClaudeActivity = &progressTime
	wb.updatedAt = time.Now()
	
	return nil
}

/**
 * CONTEXT:   Record new activity within the work block with validation
 * INPUT:     Activity timestamp for updating work block state
 * OUTPUT:    Error if activity is invalid, nil on success
 * BUSINESS:  Activities update last activity time and reset idle state
 * CHANGE:    Enhanced activity recording with Claude processing state handling
 * RISK:      Low - Validation prevents invalid state transitions
 */
func (wb *WorkBlock) RecordActivity(activityTime time.Time) error {
	// Validate activity time
	if activityTime.Before(wb.startTime) {
		return fmt.Errorf("activity time %v cannot be before work block start %v",
			activityTime, wb.startTime)
	}

	// Validate activity time is reasonable
	maxFutureTime := time.Now().Add(5 * time.Minute)
	if activityTime.After(maxFutureTime) {
		return fmt.Errorf("activity time %v cannot be more than 5 minutes in future",
			activityTime)
	}

	// If work block is finished, cannot add more activities
	if wb.state == WorkBlockStateFinished {
		return fmt.Errorf("cannot record activity on finished work block")
	}

	// Update activity state
	wb.lastActivityTime = activityTime
	wb.activityCount++
	wb.updatedAt = time.Now()

	// Reset state to active if was idle (but preserve processing state)
	if wb.state == WorkBlockStateIdle {
		wb.state = WorkBlockStateActive
	}
	// Note: Don't change state if currently processing - Claude might still be working

	return nil
}

/**
 * CONTEXT:   Mark work block as idle when no recent activity detected
 * INPUT:     Timestamp when idle state was detected
 * OUTPUT:    Error if transition invalid, nil on success
 * BUSINESS:  Work blocks become idle after 5 minutes without activity
 * CHANGE:    Initial idle state transition logic
 * RISK:      Low - State transition with validation
 */
func (wb *WorkBlock) MarkIdle(idleTime time.Time) error {
	if wb.state == WorkBlockStateFinished {
		return fmt.Errorf("cannot mark finished work block as idle")
	}

	if wb.state == WorkBlockStateIdle {
		return nil // Already idle
	}

	// Validate idle time is after last activity + timeout
	expectedIdleTime := wb.lastActivityTime.Add(IdleTimeout)
	if idleTime.Before(expectedIdleTime) {
		return fmt.Errorf("idle time %v is before expected idle time %v",
			idleTime, expectedIdleTime)
	}

	wb.state = WorkBlockStateIdle
	wb.updatedAt = time.Now()

	return nil
}

/**
 * CONTEXT:   Finalize work block when session ends or new block starts
 * INPUT:     End timestamp for work block completion
 * OUTPUT:    Error if finalization invalid, nil on success
 * BUSINESS:  Work blocks finish with defined end time and cannot be modified
 * CHANGE:    Initial work block finalization logic
 * RISK:      Low - Final state transition with validation
 */
func (wb *WorkBlock) Finish(endTime time.Time) error {
	if wb.state == WorkBlockStateFinished {
		return fmt.Errorf("work block is already finished")
	}

	// Validate end time
	if endTime.Before(wb.startTime) {
		return fmt.Errorf("end time %v cannot be before start time %v",
			endTime, wb.startTime)
	}

	if endTime.After(time.Now().Add(5*time.Minute)) {
		return fmt.Errorf("end time %v cannot be more than 5 minutes in future", endTime)
	}

	wb.endTime = endTime
	wb.state = WorkBlockStateFinished
	wb.updatedAt = time.Now()

	return nil
}

/**
 * CONTEXT:   Export work block data for serialization or reporting
 * INPUT:     No parameters, uses internal work block state
 * OUTPUT:    WorkBlockData struct suitable for JSON serialization
 * BUSINESS:  Provide read-only view of work block data for external use
 * CHANGE:    Initial data export implementation
 * RISK:      Low - Read-only operation with no state changes
 */
type WorkBlockData struct {
	ID                     string           `json:"id"`
	SessionID              string           `json:"session_id"`
	ProjectID              string           `json:"project_id"`
	ProjectName            string           `json:"project_name"`
	ProjectPath            string           `json:"project_path"`
	StartTime              time.Time        `json:"start_time"`
	EndTime                time.Time        `json:"end_time"`
	State                  WorkBlockState   `json:"state"`
	LastActivityTime       time.Time        `json:"last_activity_time"`
	ActivityCount          int64            `json:"activity_count"`
	DurationSeconds        int64            `json:"duration_seconds"`
	DurationHours          float64          `json:"duration_hours"`
	IsActive               bool             `json:"is_active"`
	CreatedAt              time.Time        `json:"created_at"`
	UpdatedAt              time.Time        `json:"updated_at"`
	// Enhanced: Claude processing fields
	ClaudeProcessingSeconds int64      `json:"claude_processing_seconds"`
	ClaudeProcessingHours   float64    `json:"claude_processing_hours"`
	EstimatedEndTime        *time.Time `json:"estimated_end_time"`
	LastClaudeActivity      *time.Time `json:"last_claude_activity"`
	ActivePromptID          string     `json:"active_prompt_id"`
	IsProcessing            bool       `json:"is_processing"`
}

func (wb *WorkBlock) ToData() WorkBlockData {
	return WorkBlockData{
		ID:                      wb.id,
		SessionID:               wb.sessionID,
		ProjectID:               wb.projectID,
		ProjectName:             wb.projectName,
		ProjectPath:             wb.projectPath,
		StartTime:               wb.startTime,
		EndTime:                 wb.endTime,
		State:                   wb.state,
		LastActivityTime:        wb.lastActivityTime,
		ActivityCount:           wb.activityCount,
		DurationSeconds:         wb.DurationSeconds(),
		DurationHours:           wb.DurationHours(),
		IsActive:                wb.state == WorkBlockStateActive,
		CreatedAt:               wb.createdAt,
		UpdatedAt:               wb.updatedAt,
		// Enhanced: Claude processing data
		ClaudeProcessingSeconds: int64(wb.claudeProcessingTime.Seconds()),
		ClaudeProcessingHours:   wb.claudeProcessingTime.Hours(),
		EstimatedEndTime:        wb.EstimatedEndTime(),      // Use getter for safe copy
		LastClaudeActivity:      wb.LastClaudeActivity(),    // Use getter for safe copy
		ActivePromptID:          wb.activePromptID,
		IsProcessing:            wb.state == WorkBlockStateProcessing,
	}
}

/**
 * CONTEXT:   Validate work block internal state consistency
 * INPUT:     No parameters, validates internal state
 * OUTPUT:    Error if state is inconsistent, nil if valid
 * BUSINESS:  Ensure work block data integrity and business rule compliance
 * CHANGE:    Initial state validation implementation
 * RISK:      Low - Validation only, no state changes
 */
func (wb *WorkBlock) Validate() error {
	// Check required fields
	if wb.id == "" {
		return fmt.Errorf("work block ID cannot be empty")
	}
	if wb.sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if wb.projectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Check time relationships
	if wb.lastActivityTime.Before(wb.startTime) {
		return fmt.Errorf("last activity time cannot be before start time")
	}

	if !wb.endTime.IsZero() && wb.endTime.Before(wb.startTime) {
		return fmt.Errorf("end time cannot be before start time")
	}

	if !wb.endTime.IsZero() && wb.lastActivityTime.After(wb.endTime) {
		return fmt.Errorf("last activity time cannot be after end time")
	}

	// Check activity count
	if wb.activityCount < 1 {
		return fmt.Errorf("work block must have at least 1 activity")
	}

	// Check state consistency
	if wb.state == WorkBlockStateFinished && wb.endTime.IsZero() {
		return fmt.Errorf("finished work block must have end time")
	}

	return nil
}

/**
 * CONTEXT:   Utility function for generating consistent project IDs
 * INPUT:     Project name string for ID generation
 * OUTPUT:    Consistent project ID based on project name
 * BUSINESS:  Project IDs should be deterministic for same project names
 * CHANGE:    Initial project ID generation utility
 * RISK:      Low - Simple string manipulation utility
 */
func generateProjectID(projectName string) string {
	// Simple hash-based ID generation
	// In production, this could use a proper hash function
	hash := fmt.Sprintf("%x", []byte(projectName))
	if len(hash) > 8 {
		hash = hash[:8]
	}
	return fmt.Sprintf("proj_%s", hash)
}