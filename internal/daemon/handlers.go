package daemon

import (
	"fmt"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
	"github.com/claude-monitor/claude-monitor/pkg/events"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-014
 * CONTEXT:   Enhanced event handlers implementing precise business logic for Claude process detection
 * REASON:    Need sophisticated event filtering and state coordination for accurate session tracking
 * CHANGE:    Enhanced with improved Claude process detection and error handling.
 * PREVENTION:Validate all event data before processing, implement circuit breaker for handler failures
 * RISK:      High - Handler errors could cause missed interactions or incorrect state tracking
 */

// SessionEventHandler handles events that affect session and work block state with enhanced filtering
type SessionEventHandler struct {
	*arch.BaseEventHandler
	sessionManager  arch.SessionManager
	workBlockMgr    arch.WorkBlockManager
	eventValidator  *events.EventValidator
	
	// Statistics for monitoring
	eventsProcessed int64
	eventsFiltered  int64
	errorCount      int64
}

// NewSessionEventHandler creates a new session event handler with enhanced validation
func NewSessionEventHandler(sessionMgr arch.SessionManager, workBlockMgr arch.WorkBlockManager, logger arch.Logger) *SessionEventHandler {
	eventTypes := []events.EventType{
		events.EventExecve,
		events.EventConnect,
	}
	
	baseHandler := arch.NewBaseEventHandler("session-handler", 100, eventTypes, logger)
	
	return &SessionEventHandler{
		BaseEventHandler: baseHandler,
		sessionManager:   sessionMgr,
		workBlockMgr:     workBlockMgr,
		eventValidator:   &events.EventValidator{},
		eventsProcessed:  0,
		eventsFiltered:   0,
		errorCount:       0,
	}
}

// Handle processes events with enhanced validation and business logic
func (seh *SessionEventHandler) Handle(event *events.SystemEvent) error {
	// Validate event structure
	if err := seh.eventValidator.Validate(event); err != nil {
		seh.errorCount++
		seh.LogError("Invalid event received", 
			"error", err,
			"eventType", event.Type,
			"pid", event.PID)
		return fmt.Errorf("event validation failed: %w", err)
	}
	
	// Check if event is relevant for Claude monitoring
	if !seh.eventValidator.IsRelevant(event) {
		seh.eventsFiltered++
		seh.LogDebug("Event filtered out", 
			"eventType", event.Type,
			"pid", event.PID,
			"command", event.Command,
			"reason", "not Claude-related")
		return nil
	}
	
	seh.eventsProcessed++
	
	// Determine interaction type
	var interactionType string
	switch {
	case event.IsClaudeProcess() && event.Type == events.EventExecve:
		interactionType = "claude_process_start"
	case event.IsAnthropicConnection() && event.Type == events.EventConnect:
		interactionType = "api_connection"
	default:
		interactionType = "unknown"
	}
	
	seh.LogDebug("Processing Claude interaction", 
		"interactionType", interactionType,
		"eventType", event.Type,
		"pid", event.PID,
		"command", event.Command,
		"timestamp", event.Timestamp)
	
	// Handle session logic - this may create a new 5-hour session
	session, err := seh.sessionManager.HandleInteraction(event.Timestamp)
	if err != nil {
		seh.errorCount++
		seh.LogError("Session handling failed", 
			"error", err,
			"interactionType", interactionType)
		return fmt.Errorf("session handling error: %w", err)
	}
	
	// Handle work block logic - this may create a new work block or extend existing
	if session != nil {
		workBlock, err := seh.workBlockMgr.RecordActivity(session.ID, event.Timestamp)
		if err != nil {
			seh.errorCount++
			seh.LogError("Work block handling failed", 
				"error", err,
				"sessionID", session.ID,
				"interactionType", interactionType)
			return fmt.Errorf("work block handling error: %w", err)
		}
		
		seh.LogInfo("Claude interaction processed successfully", 
			"interactionType", interactionType,
			"sessionID", session.ID,
			"workBlockID", workBlock.ID,
			"eventsProcessed", seh.eventsProcessed)
	} else {
		seh.LogError("No session available for work block recording", 
			"interactionType", interactionType)
		return fmt.Errorf("no active session for work block")
	}
	
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-015
 * CONTEXT:   Enhanced process event handler with sophisticated process lifecycle tracking
 * REASON:    Need accurate process correlation with sessions for complete activity monitoring
 * CHANGE:    Enhanced with process validation, PID reuse handling, and improved error recovery.
 * PREVENTION:Always validate PID ranges, handle process state transitions, implement process cleanup
 * RISK:      Medium - Process tracking errors could affect session correlation accuracy
 */

// ProcessEventHandler handles process lifecycle events with enhanced tracking
type ProcessEventHandler struct {
	*arch.BaseEventHandler
	dbManager       arch.DatabaseManager
	sessionManager  arch.SessionManager
	eventValidator  *events.EventValidator
	
	// Process tracking state
	activeProcesses map[uint32]*domain.Process // PID -> Process
	processCount    int64
	errorCount      int64
}

// NewProcessEventHandler creates a new process event handler with enhanced tracking
func NewProcessEventHandler(dbMgr arch.DatabaseManager, sessionMgr arch.SessionManager, logger arch.Logger) *ProcessEventHandler {
	eventTypes := []events.EventType{
		events.EventExecve,
		events.EventExit,
	}
	
	baseHandler := arch.NewBaseEventHandler("process-handler", 50, eventTypes, logger)
	
	return &ProcessEventHandler{
		BaseEventHandler: baseHandler,
		dbManager:        dbMgr,
		sessionManager:   sessionMgr,
		eventValidator:   &events.EventValidator{},
		activeProcesses:  make(map[uint32]*domain.Process),
		processCount:     0,
		errorCount:       0,
	}
}

// Handle processes process lifecycle events with enhanced validation
func (peh *ProcessEventHandler) Handle(event *events.SystemEvent) error {
	// Validate event structure
	if err := peh.eventValidator.Validate(event); err != nil {
		peh.errorCount++
		peh.LogError("Invalid process event", 
			"error", err,
			"eventType", event.Type,
			"pid", event.PID)
		return fmt.Errorf("process event validation failed: %w", err)
	}
	
	// Only handle Claude process events
	if !event.IsClaudeProcess() {
		peh.LogDebug("Non-Claude process event filtered", 
			"eventType", event.Type,
			"pid", event.PID,
			"command", event.Command)
		return nil
	}
	
	peh.LogDebug("Processing Claude process event", 
		"eventType", event.Type,
		"pid", event.PID,
		"command", event.Command,
		"timestamp", event.Timestamp)
	
	switch event.Type {
	case events.EventExecve:
		return peh.handleProcessStart(event)
	case events.EventExit:
		return peh.handleProcessExit(event)
	default:
		peh.LogDebug("Unhandled process event type", 
			"eventType", event.Type,
			"pid", event.PID)
		return nil
	}
}

func (peh *ProcessEventHandler) handleProcessStart(event *events.SystemEvent) error {
	peh.LogInfo("Claude process started", 
		"pid", event.PID,
		"command", event.Command,
		"timestamp", event.Timestamp)
	
	// Check for PID reuse
	if existingProcess, exists := peh.activeProcesses[event.PID]; exists {
		peh.LogWarn("PID reuse detected", 
			"pid", event.PID,
			"oldCommand", existingProcess.Command,
			"newCommand", event.Command,
			"oldStartTime", existingProcess.StartTime)
		
		// Remove old process from tracking
		delete(peh.activeProcesses, event.PID)
	}
	
	// Create process record
	process := domain.NewProcess(event.PID, event.Command, event.Timestamp)
	
	// Get current session to link process
	var sessionID string
	if session, exists := peh.sessionManager.GetCurrentSession(); exists {
		sessionID = session.ID
		peh.LogDebug("Linking process to active session", 
			"pid", event.PID,
			"sessionID", sessionID)
	} else {
		peh.LogDebug("No active session for process linking", 
			"pid", event.PID)
	}
	
	// Save process record to database
	if err := peh.dbManager.SaveProcess(process, sessionID); err != nil {
		peh.errorCount++
		peh.LogError("Failed to save process record", 
			"error", err,
			"pid", event.PID,
			"command", event.Command)
		return fmt.Errorf("failed to save process: %w", err)
	}
	
	// Track active process
	peh.activeProcesses[event.PID] = process
	peh.processCount++
	
	peh.LogInfo("Claude process tracked successfully", 
		"pid", event.PID,
		"sessionID", sessionID,
		"totalProcesses", peh.processCount)
	
	return nil
}

func (peh *ProcessEventHandler) handleProcessExit(event *events.SystemEvent) error {
	peh.LogInfo("Claude process exited", 
		"pid", event.PID,
		"command", event.Command,
		"timestamp", event.Timestamp)
	
	// Check if we were tracking this process
	process, wasTracking := peh.activeProcesses[event.PID]
	if !wasTracking {
		peh.LogDebug("Process exit for untracked PID", 
			"pid", event.PID,
			"command", event.Command)
		return nil
	}
	
	// Calculate process lifetime
	lifetime := event.Timestamp.Sub(process.StartTime)
	
	peh.LogInfo("Tracked Claude process ended", 
		"pid", event.PID,
		"command", process.Command,
		"startTime", process.StartTime,
		"endTime", event.Timestamp,
		"lifetime", lifetime)
	
	// Remove from active tracking
	delete(peh.activeProcesses, event.PID)
	
	// Note: Process end time could be persisted to database if needed
	// This would require extending the domain.Process model
	
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-016
 * CONTEXT:   Handler utility methods for statistics and monitoring
 * REASON:    Need observability into handler performance and error rates
 * CHANGE:    Added statistics methods for monitoring handler health.
 * PREVENTION:Keep statistics lightweight and avoid expensive calculations
 * RISK:      Low - Statistics are for monitoring only and don't affect core logic
 */

// GetSessionHandlerStats returns statistics for the session event handler
func (seh *SessionEventHandler) GetSessionHandlerStats() map[string]interface{} {
	return map[string]interface{}{
		"eventsProcessed": seh.eventsProcessed,
		"eventsFiltered":  seh.eventsFiltered,
		"errorCount":      seh.errorCount,
		"handlerType":     "session-handler",
	}
}

// GetProcessHandlerStats returns statistics for the process event handler
func (peh *ProcessEventHandler) GetProcessHandlerStats() map[string]interface{} {
	return map[string]interface{}{
		"processCount":     peh.processCount,
		"activeProcesses": len(peh.activeProcesses),
		"errorCount":      peh.errorCount,
		"handlerType":     "process-handler",
	}
}

// GetActiveProcesses returns a list of currently tracked processes
func (peh *ProcessEventHandler) GetActiveProcesses() []*domain.Process {
	processes := make([]*domain.Process, 0, len(peh.activeProcesses))
	for _, process := range peh.activeProcesses {
		processes = append(processes, process)
	}
	return processes
}

// ResetStats resets handler statistics (useful for testing or monitoring resets)
func (seh *SessionEventHandler) ResetStats() {
	seh.eventsProcessed = 0
	seh.eventsFiltered = 0
	seh.errorCount = 0
}

// ResetStats resets handler statistics (useful for testing or monitoring resets)
func (peh *ProcessEventHandler) ResetStats() {
	peh.processCount = 0
	peh.errorCount = 0
	// Note: We don't clear activeProcesses as they represent actual running processes
}