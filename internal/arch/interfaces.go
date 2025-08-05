package arch

import (
	"context"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/domain"
	"github.com/claude-monitor/claude-monitor/pkg/events"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-006
 * CONTEXT:   Core system interfaces defining contracts between components
 * REASON:    Need clean abstractions for dependency injection and testing isolation
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep interfaces focused and avoid kitchen-sink patterns
 * RISK:      High - Poor interface design affects entire system maintainability
 */

// EBPFManager defines the contract for eBPF program management and event capture
type EBPFManager interface {
	// LoadPrograms loads and attaches all eBPF programs to kernel
	LoadPrograms() error
	
	// StartEventProcessing begins reading events from eBPF ring buffers
	StartEventProcessing(ctx context.Context) error
	
	// GetEventChannel returns the channel for receiving system events
	GetEventChannel() <-chan *events.SystemEvent
	
	// Stop cleanly detaches eBPF programs and releases resources
	Stop() error
	
	// GetStats returns eBPF-specific monitoring statistics
	GetStats() (*EBPFStats, error)
}

// EBPFStats represents eBPF monitoring statistics
type EBPFStats struct {
	EventsProcessed   int64 `json:"eventsProcessed"`
	DroppedEvents     int64 `json:"droppedEvents"`
	ProgramsAttached  int   `json:"programsAttached"`
	RingBufferSize    int   `json:"ringBufferSize"`
}

// EventProcessorStats represents event processing statistics
type EventProcessorStats struct {
	EventsProcessed   int64 `json:"eventsProcessed"`
	EventsDropped     int64 `json:"eventsDropped"`
	ProcessingErrors  int64 `json:"processingErrors"`
	HandlersRegistered int   `json:"handlersRegistered"`
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-007
 * CONTEXT:   Session management interface for business logic coordination
 * REASON:    Session logic is complex with 5-hour windows and needs clean abstraction
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate session state transitions and handle concurrent access safely
 * RISK:      High - Session logic errors could cause incorrect time tracking
 */

// SessionManager defines the contract for session lifecycle management
type SessionManager interface {
	// HandleInteraction processes user interaction events and manages session state
	HandleInteraction(timestamp time.Time) (*domain.Session, error)
	
	// GetCurrentSession returns the active session if one exists
	GetCurrentSession() (*domain.Session, bool)
	
	// IsSessionActive checks if there is currently an active session
	IsSessionActive() bool
	
	// FinalizeCurrentSession ends the current session and persists final state
	FinalizeCurrentSession() error
	
	// GetSessionStats returns statistics for the specified time period
	GetSessionStats(period TimePeriod) (*domain.SessionStats, error)
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-008
 * CONTEXT:   Work block management interface for activity tracking within sessions
 * REASON:    Work blocks have complex timeout logic and need separate management from sessions
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure work blocks never extend beyond their containing session boundaries
 * RISK:      Medium - Work block timing errors could cause inaccurate billing
 */

// WorkBlockManager defines the contract for work block lifecycle management
type WorkBlockManager interface {
	// RecordActivity processes activity events and manages work block state
	RecordActivity(sessionID string, timestamp time.Time) (*domain.WorkBlock, error)
	
	// GetActiveBlock returns the current active work block if one exists
	GetActiveBlock() (*domain.WorkBlock, bool)
	
	// FinalizeCurrentBlock ends the current work block due to inactivity timeout
	FinalizeCurrentBlock() error
	
	// CheckInactivityTimeout checks if current block should be finalized due to timeout
	CheckInactivityTimeout(currentTime time.Time) error
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-009
 * CONTEXT:   Database persistence interface abstracting Kùzu graph operations
 * REASON:    Need clean abstraction over graph database operations for testability
 * CHANGE:    Initial implementation.
 * PREVENTION:Always use transactions for multi-node operations and handle connection failures
 * RISK:      Medium - Data persistence failures could cause data loss
 */

// DatabaseManager defines the contract for data persistence operations
type DatabaseManager interface {
	// Initialize sets up database schema and connections
	Initialize() error
	
	// SaveSession persists a session to the graph database
	SaveSession(session *domain.Session) error
	
	// SaveWorkBlock persists a work block and its relationship to session
	SaveWorkBlock(block *domain.WorkBlock) error
	
	// SaveProcess persists a process record and its session relationship
	SaveProcess(process *domain.Process, sessionID string) error
	
	// GetSessionStats calculates aggregated statistics for reporting
	GetSessionStats(period TimePeriod) (*domain.SessionStats, error)
	
	// GetActiveSession retrieves the currently active session from database
	GetActiveSession() (*domain.Session, error)
	
	// Close cleanly shuts down database connections
	Close() error
	
	// HealthCheck verifies database connectivity and schema integrity
	HealthCheck() error
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-010
 * CONTEXT:   Event processing interface for handling eBPF events with business logic
 * REASON:    Need clean separation between event capture and business logic processing
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure event handlers are stateless and can handle events in any order
 * RISK:      High - Event processing errors could cause missed interactions or incorrect state
 */

// EventProcessor defines the contract for processing system events
type EventProcessor interface {
	// ProcessEvent handles a single system event and updates system state
	ProcessEvent(event *events.SystemEvent) error
	
	// Start begins event processing loop
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down event processing
	Stop() error
	
	// RegisterHandler adds an event handler for specific event types
	RegisterHandler(handler EventHandler) error
	
	// GetStats returns event processing statistics
	GetStats() *EventProcessorStats
}

// EventHandler defines the contract for handling specific types of events
type EventHandler interface {
	// CanHandle returns true if this handler can process the given event type
	CanHandle(eventType events.EventType) bool
	
	// Handle processes the event and returns any error
	Handle(event *events.SystemEvent) error
	
	// Priority returns the handler priority (higher values processed first)
	Priority() int
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-011
 * CONTEXT:   Daemon lifecycle management interface for system service coordination
 * REASON:    Need clean abstraction for daemon startup, shutdown, and health monitoring
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure proper cleanup of all resources on daemon shutdown
 * RISK:      Medium - Resource leaks if daemon doesn't shut down cleanly
 */

// DaemonManager defines the contract for daemon lifecycle management
type DaemonManager interface {
	// Start initializes all components and begins monitoring
	Start() error
	
	// Stop gracefully shuts down all components
	Stop() error
	
	// GetStatus returns current daemon and monitoring status
	GetStatus() (*domain.SystemStatus, error)
	
	// IsRunning returns true if daemon is currently active
	IsRunning() bool
	
	// Restart performs a graceful restart of the daemon
	Restart() error
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-012
 * CONTEXT:   CLI command interface for user interaction and system control
 * REASON:    Need clean abstraction for CLI operations separate from business logic
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate all user inputs and provide clear error messages
 * RISK:      Low - CLI errors are user-facing and recoverable
 */

// CLIManager defines the contract for command-line interface operations
type CLIManager interface {
	// ExecuteStart starts the daemon with optional configuration
	ExecuteStart(config *StartConfig) error
	
	// ExecuteStatus returns formatted status information
	ExecuteStatus() (string, error)
	
	// ExecuteReport generates usage reports for specified period
	ExecuteReport(period TimePeriod) (string, error)
	
	// ExecuteStop stops the running daemon
	ExecuteStop() error
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-013
 * CONTEXT:   Configuration and enumeration types for system operation
 * REASON:    Need well-defined configuration structures and enums for type safety
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate configuration values and provide sensible defaults
 * RISK:      Low - Configuration errors are typically caught at startup
 */

// TimePeriod represents different reporting time periods
type TimePeriod string

const (
	PeriodDaily   TimePeriod = "daily"
	PeriodWeekly  TimePeriod = "weekly"
	PeriodMonthly TimePeriod = "monthly"
)

// StartConfig contains configuration options for daemon startup
type StartConfig struct {
	DatabasePath string `json:"databasePath"`
	LogLevel     string `json:"logLevel"`
	PidFile      string `json:"pidFile"`
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-014
 * CONTEXT:   Utility interfaces for cross-cutting concerns
 * REASON:    Need clean abstractions for logging, timing, and system utilities
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep utility interfaces minimal and focused on single responsibilities
 * RISK:      Low - Utility interfaces are stable and well-understood patterns
 */

// Logger defines the contract for structured logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
}

// TimeProvider defines the contract for time operations (useful for testing)
type TimeProvider interface {
	Now() time.Time
	Since(t time.Time) time.Duration
}

// SystemProvider defines the contract for system-level operations
type SystemProvider interface {
	GetPID() int
	WritePidFile(path string) error
	RemovePidFile(path string) error
	CheckPidFile(path string) (bool, error)
}