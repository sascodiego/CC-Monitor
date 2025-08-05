package domain

import (
	"time"

	"github.com/google/uuid"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-001
 * CONTEXT:   Core domain entities representing the business objects in the Claude Monitor system
 * REASON:    Need well-defined domain objects that represent sessions, work blocks, and processes
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep entities focused on data and behavior, avoid infrastructure concerns
 * RISK:      Low - Domain entities are stable and well-defined by business requirements
 */

// Session represents a 5-hour tracking window that starts with first user interaction
type Session struct {
	ID        string    `json:"sessionID"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	IsActive  bool      `json:"isActive"`
}

// NewSession creates a new session starting at the given time
func NewSession(startTime time.Time) *Session {
	return &Session{
		ID:        uuid.New().String(),
		StartTime: startTime,
		EndTime:   startTime.Add(5 * time.Hour), // 5-hour window
		IsActive:  true,
	}
}

// IsExpired checks if the session has exceeded its 5-hour window
func (s *Session) IsExpired(currentTime time.Time) bool {
	return currentTime.After(s.EndTime)
}

// Duration returns the total duration of the session
func (s *Session) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-002
 * CONTEXT:   WorkBlock entity representing continuous activity periods within sessions
 * REASON:    Business requirement to track actual work time with 5-minute inactivity timeout
 * CHANGE:    Initial implementation.
 * PREVENTION:Always ensure work blocks are contained within session boundaries
 * RISK:      Medium - Work blocks extending beyond sessions could corrupt billing logic
 */

// WorkBlock represents a continuous period of activity within a session
type WorkBlock struct {
	ID               string    `json:"blockID"`
	SessionID        string    `json:"sessionID"`
	StartTime        time.Time `json:"startTime"`
	EndTime          *time.Time `json:"endTime,omitempty"`
	DurationSeconds  int64     `json:"durationSeconds"`
	LastActivity     time.Time `json:"lastActivity"`
	IsActive         bool      `json:"isActive"`
}

// NewWorkBlock creates a new work block for the given session
func NewWorkBlock(sessionID string, startTime time.Time) *WorkBlock {
	return &WorkBlock{
		ID:           uuid.New().String(),
		SessionID:    sessionID,
		StartTime:    startTime,
		LastActivity: startTime,
		IsActive:     true,
	}
}

// UpdateActivity updates the last activity time for the work block
func (wb *WorkBlock) UpdateActivity(activityTime time.Time) {
	wb.LastActivity = activityTime
}

// IsInactive checks if the work block has been inactive for more than 5 minutes
func (wb *WorkBlock) IsInactive(currentTime time.Time) bool {
	return currentTime.Sub(wb.LastActivity) > 5*time.Minute
}

// Finalize completes the work block and calculates duration
func (wb *WorkBlock) Finalize(endTime time.Time) {
	wb.EndTime = &endTime
	wb.IsActive = false
	wb.DurationSeconds = int64(endTime.Sub(wb.StartTime).Seconds())
}

// Duration returns the current duration of the work block
func (wb *WorkBlock) Duration() time.Duration {
	if wb.EndTime != nil {
		return wb.EndTime.Sub(wb.StartTime)
	}
	return wb.LastActivity.Sub(wb.StartTime)
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-003
 * CONTEXT:   Process entity representing system processes monitored by eBPF
 * REASON:    Need to track process information from eBPF events for session correlation
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate PID ranges and ensure command strings are sanitized
 * RISK:      Low - Process data is read-only from kernel events
 */

// Process represents a system process detected by eBPF monitoring
type Process struct {
	PID       uint32    `json:"pid"`
	Command   string    `json:"command"`
	StartTime time.Time `json:"startTime"`
}

// NewProcess creates a new process record
func NewProcess(pid uint32, command string, startTime time.Time) *Process {
	return &Process{
		PID:       pid,
		Command:   command,
		StartTime: startTime,
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-004
 * CONTEXT:   SystemStatus entity representing current daemon and monitoring state
 * REASON:    Need structured status information for CLI status command and health monitoring
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep status lightweight and avoid expensive calculations during status queries
 * RISK:      Low - Status is read-only aggregation of system state
 */

// SystemStatus represents the current state of the monitoring system
type SystemStatus struct {
	DaemonRunning     bool            `json:"daemonRunning"`
	CurrentSession    *Session        `json:"currentSession,omitempty"`
	CurrentWorkBlock  *WorkBlock      `json:"currentWorkBlock,omitempty"`
	LastActivity      *time.Time      `json:"lastActivity,omitempty"`
	MonitoringActive  bool            `json:"monitoringActive"`
	EventsProcessed   int64           `json:"eventsProcessed"`
	Uptime            time.Duration   `json:"uptime"`
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-005
 * CONTEXT:   SessionStats entity for aggregated reporting data
 * REASON:    CLI reporting commands need structured statistics for different time periods
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure statistics calculations are efficient and cached when possible
 * RISK:      Low - Statistics are derived data and can be recalculated if needed
 */

// SessionStats represents aggregated statistics for reporting
type SessionStats struct {
	Period            string        `json:"period"`
	TotalSessions     int           `json:"totalSessions"`
	TotalWorkTime     time.Duration `json:"totalWorkTime"`
	AverageWorkTime   time.Duration `json:"averageWorkTime"`
	SessionCount      int           `json:"sessionCount"`
	WorkBlockCount    int           `json:"workBlockCount"`
	StartDate         time.Time     `json:"startDate"`
	EndDate           time.Time     `json:"endDate"`
}