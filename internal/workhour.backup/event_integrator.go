/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-001
 * CONTEXT:   Event integrator connecting enhanced daemon monitoring with work hour analytics system
 * REASON:    Need real-time integration between activity monitoring and work hour analysis for immediate updates
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle event processing failures gracefully, maintain loose coupling between systems
 * RISK:      Medium - Integration failures could cause delayed or inconsistent work hour updates
 */
package workhour

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-002
 * CONTEXT:   WorkHourEventIntegrator implements real-time integration with enhanced daemon
 * REASON:    Business requirement for immediate work hour updates as activity occurs
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement proper error recovery and event deduplication, monitor integration health
 * RISK:      High - Integration quality affects real-time accuracy of work hour tracking
 */
type WorkHourEventIntegrator struct {
	workHourService *WorkHourService
	logger          arch.Logger
	
	// Event processing
	sessionEvents   chan SessionEvent
	workBlockEvents chan WorkBlockEvent
	activityEvents  chan ActivityEvent
	
	// State tracking for optimization
	lastProcessedSession   string
	lastProcessedWorkBlock string
	mu                     sync.RWMutex
	
	// Integration control
	ctx                    context.Context
	cancel                 context.CancelFunc
	isRunning              bool
	eventStats             EventIntegrationStats
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-003
 * CONTEXT:   Event types for different daemon lifecycle events
 * REASON:    Need structured event types to handle different aspects of activity monitoring
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep event structures lightweight and ensure proper serialization
 * RISK:      Low - Event structure changes affect integration but not core functionality
 */
type SessionEvent struct {
	Type        SessionEventType `json:"type"`
	SessionID   string          `json:"sessionId"`
	Timestamp   time.Time       `json:"timestamp"`
	Session     *domain.Session `json:"session,omitempty"`
}

type SessionEventType string

const (
	SessionStarted SessionEventType = "session_started"
	SessionEnded   SessionEventType = "session_ended"
	SessionUpdated SessionEventType = "session_updated"
)

type WorkBlockEvent struct {
	Type        WorkBlockEventType  `json:"type"`
	BlockID     string             `json:"blockId"`
	SessionID   string             `json:"sessionId"`
	Timestamp   time.Time          `json:"timestamp"`
	WorkBlock   *domain.WorkBlock  `json:"workBlock,omitempty"`
}

type WorkBlockEventType string

const (
	WorkBlockStarted   WorkBlockEventType = "work_block_started"
	WorkBlockFinalized WorkBlockEventType = "work_block_finalized"
	WorkBlockUpdated   WorkBlockEventType = "work_block_updated"
)

type ActivityEvent struct {
	Type         ActivityEventType `json:"type"`
	Timestamp    time.Time        `json:"timestamp"`
	ActivityType string           `json:"activityType"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type ActivityEventType string

const (
	UserInteractionDetected ActivityEventType = "user_interaction"
	BackgroundActivity      ActivityEventType = "background_activity"
	InactivityDetected      ActivityEventType = "inactivity_detected"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-004
 * CONTEXT:   EventIntegrationStats tracks integration performance and health metrics
 * REASON:    Need monitoring of integration performance to detect and resolve issues
 * CHANGE:    Initial implementation.
 * PREVENTION:Monitor stats regularly and alert on anomalies
 * RISK:      Low - Statistics collection for monitoring and debugging
 */
type EventIntegrationStats struct {
	SessionEventsProcessed   int64     `json:"sessionEventsProcessed"`
	WorkBlockEventsProcessed int64     `json:"workBlockEventsProcessed"`
	ActivityEventsProcessed  int64     `json:"activityEventsProcessed"`
	ProcessingErrors         int64     `json:"processingErrors"`
	LastEventProcessed       time.Time `json:"lastEventProcessed"`
	IntegrationStartTime     time.Time `json:"integrationStartTime"`
	WorkDaysUpdated          int64     `json:"workDaysUpdated"`
	mu                       sync.RWMutex
}

func NewWorkHourEventIntegrator(workHourService *WorkHourService, logger arch.Logger) *WorkHourEventIntegrator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkHourEventIntegrator{
		workHourService: workHourService,
		logger:          logger,
		sessionEvents:   make(chan SessionEvent, 100),
		workBlockEvents: make(chan WorkBlockEvent, 200),
		activityEvents:  make(chan ActivityEvent, 500),
		ctx:             ctx,
		cancel:          cancel,
		eventStats: EventIntegrationStats{
			IntegrationStartTime: time.Now(),
		},
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-005
 * CONTEXT:   Start initializes event integration with background processing
 * REASON:    Need background processing to handle events asynchronously without blocking daemon
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure proper goroutine lifecycle management and resource cleanup
 * RISK:      Medium - Background processing failures could cause event loss
 */
func (whei *WorkHourEventIntegrator) Start() error {
	whei.mu.Lock()
	defer whei.mu.Unlock()
	
	if whei.isRunning {
		return fmt.Errorf("work hour event integrator already running")
	}
	
	whei.logger.Info("Starting work hour event integrator")
	
	// Start event processing goroutines
	go whei.processSessionEvents()
	go whei.processWorkBlockEvents()
	go whei.processActivityEvents()
	go whei.processPeriodicUpdates()
	
	whei.isRunning = true
	whei.logger.Info("Work hour event integrator started successfully")
	
	return nil
}

func (whei *WorkHourEventIntegrator) Stop() error {
	whei.mu.Lock()
	defer whei.mu.Unlock()
	
	if !whei.isRunning {
		return nil
	}
	
	whei.logger.Info("Stopping work hour event integrator")
	whei.cancel()
	whei.isRunning = false
	
	whei.logger.Info("Work hour event integrator stopped")
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-006
 * CONTEXT:   Event injection methods for daemon integration
 * REASON:    Enhanced daemon needs to inject events for real-time work hour updates
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate event data and handle channel full conditions gracefully
 * RISK:      Medium - Event injection failures could cause inconsistent work hour data
 */
func (whei *WorkHourEventIntegrator) OnSessionStarted(session *domain.Session) {
	event := SessionEvent{
		Type:      SessionStarted,
		SessionID: session.ID,
		Timestamp: session.StartTime,
		Session:   session,
	}
	
	whei.injectSessionEvent(event)
}

func (whei *WorkHourEventIntegrator) OnSessionEnded(session *domain.Session) {
	event := SessionEvent{
		Type:      SessionEnded,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Session:   session,
	}
	
	whei.injectSessionEvent(event)
}

func (whei *WorkHourEventIntegrator) OnWorkBlockStarted(workBlock *domain.WorkBlock) {
	event := WorkBlockEvent{
		Type:      WorkBlockStarted,
		BlockID:   workBlock.ID,
		SessionID: workBlock.SessionID,
		Timestamp: workBlock.StartTime,
		WorkBlock: workBlock,
	}
	
	whei.injectWorkBlockEvent(event)
}

func (whei *WorkHourEventIntegrator) OnWorkBlockFinalized(workBlock *domain.WorkBlock) {
	event := WorkBlockEvent{
		Type:      WorkBlockFinalized,
		BlockID:   workBlock.ID,
		SessionID: workBlock.SessionID,
		Timestamp: time.Now(),
		WorkBlock: workBlock,
	}
	
	whei.injectWorkBlockEvent(event)
}

func (whei *WorkHourEventIntegrator) OnUserInteraction(timestamp time.Time, activityType string, metadata map[string]interface{}) {
	event := ActivityEvent{
		Type:         UserInteractionDetected,
		Timestamp:    timestamp,
		ActivityType: activityType,
		Metadata:     metadata,
	}
	
	whei.injectActivityEvent(event)
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-007
 * CONTEXT:   Event injection helper methods with error handling
 * REASON:    Need robust event injection that handles channel capacity and errors gracefully
 * CHANGE:    Initial implementation.
 * PREVENTION:Monitor channel capacity and implement backpressure handling
 * RISK:      Medium - Event loss could cause work hour data inconsistencies
 */
func (whei *WorkHourEventIntegrator) injectSessionEvent(event SessionEvent) {
	select {
	case whei.sessionEvents <- event:
		whei.logger.Debug("Session event injected", "type", event.Type, "sessionID", event.SessionID)
	default:
		whei.incrementErrorStats()
		whei.logger.Warn("Session event channel full, dropping event", 
			"type", event.Type, "sessionID", event.SessionID)
	}
}

func (whei *WorkHourEventIntegrator) injectWorkBlockEvent(event WorkBlockEvent) {
	select {
	case whei.workBlockEvents <- event:
		whei.logger.Debug("Work block event injected", "type", event.Type, "blockID", event.BlockID)
	default:
		whei.incrementErrorStats()
		whei.logger.Warn("Work block event channel full, dropping event", 
			"type", event.Type, "blockID", event.BlockID)
	}
}

func (whei *WorkHourEventIntegrator) injectActivityEvent(event ActivityEvent) {
	select {
	case whei.activityEvents <- event:
		whei.logger.Debug("Activity event injected", "type", event.Type, "activityType", event.ActivityType)
	default:
		whei.incrementErrorStats()
		whei.logger.Warn("Activity event channel full, dropping event", 
			"type", event.Type, "activityType", event.ActivityType)
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-008
 * CONTEXT:   Session event processing for real-time work hour updates
 * REASON:    Session lifecycle events trigger work day and week recalculations
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle processing errors gracefully and implement event deduplication
 * RISK:      High - Session processing affects work hour calculations accuracy
 */
func (whei *WorkHourEventIntegrator) processSessionEvents() {
	whei.logger.Info("Session event processing started")
	
	for {
		select {
		case <-whei.ctx.Done():
			whei.logger.Info("Session event processing stopped")
			return
			
		case event := <-whei.sessionEvents:
			if err := whei.handleSessionEvent(event); err != nil {
				whei.incrementErrorStats()
				whei.logger.Error("Failed to process session event", 
					"type", event.Type, "sessionID", event.SessionID, "error", err)
			} else {
				whei.incrementSessionStats()
			}
		}
	}
}

func (whei *WorkHourEventIntegrator) handleSessionEvent(event SessionEvent) error {
	whei.logger.Debug("Processing session event", "type", event.Type, "sessionID", event.SessionID)
	
	// Prevent duplicate processing
	if whei.lastProcessedSession == event.SessionID && event.Type != SessionEnded {
		whei.logger.Debug("Skipping duplicate session event", "sessionID", event.SessionID)
		return nil
	}
	
	switch event.Type {
	case SessionStarted:
		return whei.handleSessionStarted(event)
	case SessionEnded:
		return whei.handleSessionEnded(event)
	case SessionUpdated:
		return whei.handleSessionUpdated(event)
	default:
		return fmt.Errorf("unknown session event type: %s", event.Type)
	}
}

func (whei *WorkHourEventIntegrator) handleSessionStarted(event SessionEvent) error {
	whei.lastProcessedSession = event.SessionID
	
	// Session started - this will trigger work day analysis when needed
	whei.logger.Info("Session started, work hour tracking activated", 
		"sessionID", event.SessionID,
		"startTime", event.Timestamp)
	
	// Trigger work day update for the session start date
	sessionDate := event.Timestamp
	return whei.triggerWorkDayUpdate(sessionDate)
}

func (whei *WorkHourEventIntegrator) handleSessionEnded(event SessionEvent) error {
	if event.Session == nil {
		return fmt.Errorf("session data missing for session ended event")
	}
	
	// Session ended - finalize work day calculation
	whei.logger.Info("Session ended, finalizing work hour calculations", 
		"sessionID", event.SessionID,
		"duration", event.Timestamp.Sub(event.Session.StartTime))
	
	// Trigger work day update for the session date
	sessionDate := event.Session.StartTime
	return whei.triggerWorkDayUpdate(sessionDate)
}

func (whei *WorkHourEventIntegrator) handleSessionUpdated(event SessionEvent) error {
	// Session updated - may need to recalculate work day
	if event.Session != nil {
		sessionDate := event.Session.StartTime
		return whei.triggerWorkDayUpdate(sessionDate)
	}
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-009
 * CONTEXT:   Work block event processing for real-time work time tracking
 * REASON:    Work block lifecycle events provide precise work time measurements
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle overlapping work blocks and ensure accurate time calculations
 * RISK:      High - Work block processing directly affects billable time accuracy
 */
func (whei *WorkHourEventIntegrator) processWorkBlockEvents() {
	whei.logger.Info("Work block event processing started")
	
	for {
		select {
		case <-whei.ctx.Done():
			whei.logger.Info("Work block event processing stopped")
			return
			
		case event := <-whei.workBlockEvents:
			if err := whei.handleWorkBlockEvent(event); err != nil {
				whei.incrementErrorStats()
				whei.logger.Error("Failed to process work block event", 
					"type", event.Type, "blockID", event.BlockID, "error", err)
			} else {
				whei.incrementWorkBlockStats()
			}
		}
	}
}

func (whei *WorkHourEventIntegrator) handleWorkBlockEvent(event WorkBlockEvent) error {
	whei.logger.Debug("Processing work block event", "type", event.Type, "blockID", event.BlockID)
	
	// Prevent duplicate processing
	if whei.lastProcessedWorkBlock == event.BlockID && event.Type != WorkBlockFinalized {
		whei.logger.Debug("Skipping duplicate work block event", "blockID", event.BlockID)
		return nil
	}
	
	switch event.Type {
	case WorkBlockStarted:
		return whei.handleWorkBlockStarted(event)
	case WorkBlockFinalized:
		return whei.handleWorkBlockFinalized(event)
	case WorkBlockUpdated:
		return whei.handleWorkBlockUpdated(event)
	default:
		return fmt.Errorf("unknown work block event type: %s", event.Type)
	}
}

func (whei *WorkHourEventIntegrator) handleWorkBlockStarted(event WorkBlockEvent) error {
	whei.lastProcessedWorkBlock = event.BlockID
	
	whei.logger.Debug("Work block started", 
		"blockID", event.BlockID,
		"sessionID", event.SessionID,
		"startTime", event.Timestamp)
	
	// Work block started - prepare for time tracking
	return nil
}

func (whei *WorkHourEventIntegrator) handleWorkBlockFinalized(event WorkBlockEvent) error {
	if event.WorkBlock == nil {
		return fmt.Errorf("work block data missing for finalized event")
	}
	
	duration := event.WorkBlock.Duration()
	whei.logger.Info("Work block finalized", 
		"blockID", event.BlockID,
		"sessionID", event.SessionID,
		"duration", duration)
	
	// Work block finalized - trigger work day recalculation
	workBlockDate := event.WorkBlock.StartTime
	return whei.triggerWorkDayUpdate(workBlockDate)
}

func (whei *WorkHourEventIntegrator) handleWorkBlockUpdated(event WorkBlockEvent) error {
	// Work block updated - may need recalculation
	if event.WorkBlock != nil {
		workBlockDate := event.WorkBlock.StartTime
		return whei.triggerWorkDayUpdate(workBlockDate)
	}
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-010
 * CONTEXT:   Activity event processing for enhanced work pattern analysis
 * REASON:    Activity events provide fine-grained data for productivity and pattern analysis
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle high-frequency activity events efficiently without impacting performance
 * RISK:      Medium - Activity processing volume could impact system performance
 */
func (whei *WorkHourEventIntegrator) processActivityEvents() {
	whei.logger.Info("Activity event processing started")
	
	// Batch activity events for efficiency
	batchSize := 50
	batchTimeout := 5 * time.Second
	
	activityBatch := make([]ActivityEvent, 0, batchSize)
	batchTimer := time.NewTimer(batchTimeout)
	
	for {
		select {
		case <-whei.ctx.Done():
			// Process remaining batch
			if len(activityBatch) > 0 {
				whei.processActivityBatch(activityBatch)
			}
			whei.logger.Info("Activity event processing stopped")
			return
			
		case event := <-whei.activityEvents:
			activityBatch = append(activityBatch, event)
			
			// Process batch when full
			if len(activityBatch) >= batchSize {
				whei.processActivityBatch(activityBatch)
				activityBatch = activityBatch[:0]
				batchTimer.Reset(batchTimeout)
			}
			
		case <-batchTimer.C:
			// Process batch on timeout
			if len(activityBatch) > 0 {
				whei.processActivityBatch(activityBatch)
				activityBatch = activityBatch[:0]
			}
			batchTimer.Reset(batchTimeout)
		}
	}
}

func (whei *WorkHourEventIntegrator) processActivityBatch(events []ActivityEvent) {
	whei.logger.Debug("Processing activity event batch", "size", len(events))
	
	for _, event := range events {
		if err := whei.handleActivityEvent(event); err != nil {
			whei.incrementErrorStats()
			whei.logger.Error("Failed to process activity event", 
				"type", event.Type, "error", err)
		} else {
			whei.incrementActivityStats()
		}
	}
}

func (whei *WorkHourEventIntegrator) handleActivityEvent(event ActivityEvent) error {
	whei.logger.Debug("Processing activity event", "type", event.Type, "activityType", event.ActivityType)
	
	switch event.Type {
	case UserInteractionDetected:
		// User interaction - this indicates active work
		return whei.handleUserInteraction(event)
	case BackgroundActivity:
		// Background activity - log for pattern analysis
		return whei.handleBackgroundActivity(event)
	case InactivityDetected:
		// Inactivity - may trigger work block finalization
		return whei.handleInactivityDetected(event)
	default:
		return fmt.Errorf("unknown activity event type: %s", event.Type)
	}
}

func (whei *WorkHourEventIntegrator) handleUserInteraction(event ActivityEvent) error {
	// User interaction detected - ensure work day tracking is active
	eventDate := event.Timestamp
	
	// Note: We could trigger immediate work day updates here, but for efficiency
	// we let the periodic update process handle it unless it's critical
	whei.logger.Debug("User interaction detected", 
		"timestamp", event.Timestamp,
		"activityType", event.ActivityType)
	
	return nil
}

func (whei *WorkHourEventIntegrator) handleBackgroundActivity(event ActivityEvent) error {
	// Background activity - useful for pattern analysis but not immediate action
	return nil
}

func (whei *WorkHourEventIntegrator) handleInactivityDetected(event ActivityEvent) error {
	// Inactivity detected - this might indicate end of work period
	whei.logger.Debug("Inactivity detected", "timestamp", event.Timestamp)
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-011
 * CONTEXT:   Work day update triggering and periodic maintenance
 * REASON:    Need efficient work day updates that don't overwhelm the system
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement update throttling and error recovery for work day calculations
 * RISK:      Medium - Excessive updates could impact system performance
 */
func (whei *WorkHourEventIntegrator) triggerWorkDayUpdate(date time.Time) error {
	// Trigger work day analysis which will update cache and database
	_, err := whei.workHourService.GetDailyWorkSummary(date)
	if err != nil {
		return fmt.Errorf("failed to update work day: %w", err)
	}
	
	whei.incrementWorkDayStats()
	return nil
}

func (whei *WorkHourEventIntegrator) processPeriodicUpdates() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-whei.ctx.Done():
			return
			
		case <-ticker.C:
			whei.performPeriodicMaintenance()
		}
	}
}

func (whei *WorkHourEventIntegrator) performPeriodicMaintenance() {
	whei.logger.Debug("Performing periodic work hour maintenance")
	
	// Update current day to ensure real-time accuracy
	today := time.Now()
	if _, err := whei.workHourService.GetDailyWorkSummary(today); err != nil {
		whei.logger.Warn("Failed to update current day during maintenance", "error", err)
	}
	
	// Log integration statistics
	stats := whei.GetIntegrationStats()
	whei.logger.Info("Work hour integration statistics",
		"sessionEvents", stats.SessionEventsProcessed,
		"workBlockEvents", stats.WorkBlockEventsProcessed,
		"activityEvents", stats.ActivityEventsProcessed,
		"errors", stats.ProcessingErrors,
		"workDaysUpdated", stats.WorkDaysUpdated)
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-EVENT-012
 * CONTEXT:   Statistics tracking and monitoring methods
 * REASON:    Need monitoring of integration health and performance metrics
 * CHANGE:    Initial implementation.
 * PREVENTION:Monitor statistics for anomalies and performance issues
 * RISK:      Low - Statistics tracking for operational monitoring
 */
func (whei *WorkHourEventIntegrator) incrementSessionStats() {
	whei.eventStats.mu.Lock()
	defer whei.eventStats.mu.Unlock()
	
	whei.eventStats.SessionEventsProcessed++
	whei.eventStats.LastEventProcessed = time.Now()
}

func (whei *WorkHourEventIntegrator) incrementWorkBlockStats() {
	whei.eventStats.mu.Lock()
	defer whei.eventStats.mu.Unlock()
	
	whei.eventStats.WorkBlockEventsProcessed++
	whei.eventStats.LastEventProcessed = time.Now()
}

func (whei *WorkHourEventIntegrator) incrementActivityStats() {
	whei.eventStats.mu.Lock()
	defer whei.eventStats.mu.Unlock()
	
	whei.eventStats.ActivityEventsProcessed++
	whei.eventStats.LastEventProcessed = time.Now()
}

func (whei *WorkHourEventIntegrator) incrementErrorStats() {
	whei.eventStats.mu.Lock()
	defer whei.eventStats.mu.Unlock()
	
	whei.eventStats.ProcessingErrors++
}

func (whei *WorkHourEventIntegrator) incrementWorkDayStats() {
	whei.eventStats.mu.Lock()
	defer whei.eventStats.mu.Unlock()
	
	whei.eventStats.WorkDaysUpdated++
}

func (whei *WorkHourEventIntegrator) GetIntegrationStats() EventIntegrationStats {
	whei.eventStats.mu.RLock()
	defer whei.eventStats.mu.RUnlock()
	
	// Return a copy to prevent external modification
	return whei.eventStats
}

// IsHealthy returns true if the integration is functioning properly
func (whei *WorkHourEventIntegrator) IsHealthy() bool {
	stats := whei.GetIntegrationStats()
	
	// Consider healthy if processing events within last hour
	timeSinceLastEvent := time.Since(stats.LastEventProcessed)
	return timeSinceLastEvent < 1*time.Hour
}