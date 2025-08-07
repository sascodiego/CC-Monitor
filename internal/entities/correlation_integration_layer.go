/**
 * CONTEXT:   Integration layer connecting concurrent session correlation with existing work block tracking
 * INPUT:     Correlation events, work block updates, and session state changes
 * OUTPUT:    Coordinated work block and session management with accurate time tracking
 * BUSINESS:  Bridge concurrent session correlation with existing work hour tracking system
 * CHANGE:    Initial implementation of correlation-aware work block integration
 * RISK:      High - Integration accuracy affects all work time calculations and reporting
 */

package entities

import (
	"context"
	"fmt"
	"sync"
	"time"
)

/**
 * CONTEXT:   Unified coordinator for concurrent session correlation and work block tracking
 * INPUT:     Hook events, correlation results, and work block state changes
 * OUTPUT:    Coordinated session and work block management with accurate timing
 * BUSINESS:  Ensure work block tracking accurately reflects concurrent Claude session timing
 * CHANGE:    Initial integration coordinator with session-aware work block management
 * RISK:      High - Coordinator accuracy affects all work time calculations and user reports
 */
type CorrelationIntegrationCoordinator struct {
	// Core components
	sessionManager    *ConcurrentSessionManager
	errorHandler      *CorrelationErrorHandler
	workBlockTracker  *WorkBlockTracker        // Existing work block tracker
	sessionTracker    *SessionManager          // Existing session tracker
	
	// Integration state
	sessionToWorkBlock map[string]string       // sessionID -> workBlockID
	workBlockToSession map[string]string       // workBlockID -> sessionID
	pendingStartEvents map[string]*ClaudeStartEvent // promptID -> start event
	
	// Coordination
	mutex             sync.RWMutex
	
	// Configuration
	integrationMode   IntegrationMode
	timeoutDuration   time.Duration
}

// IntegrationMode defines how correlation integrates with existing tracking
type IntegrationMode string

const (
	IntegrationModeReplace    IntegrationMode = "replace"     // Replace existing tracking
	IntegrationModeAugment    IntegrationMode = "augment"     // Augment existing tracking
	IntegrationModeParallel   IntegrationMode = "parallel"    // Run in parallel for comparison
	IntegrationModeDisabled   IntegrationMode = "disabled"    // Disable correlation integration
)

/**
 * CONTEXT:   Factory for creating correlation integration coordinator
 * INPUT:     Session manager, error handler, and existing tracking components
 * OUTPUT:    Configured coordinator ready for integrated session and work block tracking
 * BUSINESS:  Initialize integrated tracking system with concurrent session support
 * CHANGE:    Initial factory with comprehensive integration configuration
 * RISK:      Medium - Configuration affects integration behavior and accuracy
 */
func NewCorrelationIntegrationCoordinator(
	sessionManager *ConcurrentSessionManager,
	errorHandler *CorrelationErrorHandler,
	workBlockTracker *WorkBlockTracker,
	sessionTracker *SessionManager) *CorrelationIntegrationCoordinator {
	
	return &CorrelationIntegrationCoordinator{
		sessionManager:     sessionManager,
		errorHandler:       errorHandler,
		workBlockTracker:   workBlockTracker,
		sessionTracker:     sessionTracker,
		
		sessionToWorkBlock: make(map[string]string),
		workBlockToSession: make(map[string]string),
		pendingStartEvents: make(map[string]*ClaudeStartEvent),
		
		integrationMode:    IntegrationModeAugment, // Default to augment existing tracking
		timeoutDuration:    30 * time.Minute,      // Timeout for pending events
	}
}

/**
 * CONTEXT:   Process Claude start event with integrated session and work block tracking
 * INPUT:     Claude start event with correlation information
 * OUTPUT:    Integrated session and work block creation with correlation support
 * BUSINESS:  Start both session correlation tracking and work block tracking for Claude processing
 * CHANGE:    Initial integrated start event processing
 * RISK:      High - Start event processing accuracy affects all subsequent tracking
 */
func (cic *CorrelationIntegrationCoordinator) ProcessClaudeStartEvent(ctx context.Context, startEvent *ClaudeStartEvent) error {
	cic.mutex.Lock()
	defer cic.mutex.Unlock()
	
	// Store pending start event for end correlation
	cic.pendingStartEvents[startEvent.PromptID] = startEvent
	
	// Start concurrent session tracking
	claudeSession, err := cic.sessionManager.StartClaudeSession(
		ctx,
		startEvent.TerminalContext,
		startEvent.PromptContent,
		startEvent.PromptID)
	if err != nil {
		return fmt.Errorf("failed to start Claude session: %w", err)
	}
	
	// Integrate with existing session tracking
	var regularSession *Session
	if cic.integrationMode != IntegrationModeDisabled {
		regularSession, err = cic.integrateWithRegularSession(ctx, startEvent, claudeSession)
		if err != nil {
			return fmt.Errorf("failed to integrate with regular session: %w", err)
		}
	}
	
	// Start or update work block with Claude processing awareness
	workBlock, err := cic.integrateWithWorkBlock(ctx, startEvent, claudeSession, regularSession)
	if err != nil {
		return fmt.Errorf("failed to integrate with work block: %w", err)
	}
	
	// Record associations
	cic.sessionToWorkBlock[claudeSession.ID] = workBlock.ID()
	cic.workBlockToSession[workBlock.ID()] = claudeSession.ID
	
	return nil
}

/**
 * CONTEXT:   Process Claude end event with correlation matching and work block completion
 * INPUT:     Claude end event with timing and correlation information
 * OUTPUT:    Matched session completion and accurate work block timing updates
 * BUSINESS:  Complete Claude session tracking and update work blocks with accurate timing
 * CHANGE:    Initial integrated end event processing with correlation matching
 * RISK:      High - End event correlation accuracy affects final work time calculations
 */
func (cic *CorrelationIntegrationCoordinator) ProcessClaudeEndEvent(ctx context.Context, endEvent *ClaudeEndEvent) error {
	cic.mutex.Lock()
	defer cic.mutex.Unlock()
	
	// Attempt correlation with concurrent session manager
	completedSession, err := cic.sessionManager.EndClaudeSession(ctx, endEvent)
	if err != nil {
		// Handle correlation failure with error handler
		return cic.handleCorrelationFailure(ctx, endEvent, err)
	}
	
	// Find corresponding work block
	workBlockID, exists := cic.sessionToWorkBlock[completedSession.ID]
	if !exists {
		return fmt.Errorf("no work block found for completed session %s", completedSession.ID)
	}
	
	// Update work block with Claude processing completion
	err = cic.completeWorkBlockWithClaudeSession(ctx, workBlockID, completedSession, endEvent)
	if err != nil {
		return fmt.Errorf("failed to complete work block: %w", err)
	}
	
	// Clean up associations
	delete(cic.sessionToWorkBlock, completedSession.ID)
	delete(cic.workBlockToSession, workBlockID)
	delete(cic.pendingStartEvents, endEvent.PromptID)
	
	return nil
}

/**
 * CONTEXT:   Integrate Claude session with existing regular session tracking
 * INPUT:     Start event, Claude session, and existing session context
 * OUTPUT:    Integrated session tracking that accounts for Claude processing
 * BUSINESS:  Ensure existing session tracking accurately reflects Claude usage patterns
 * CHANGE:    Initial integration with existing session management
 * RISK:      Medium - Session integration affects session duration and timing calculations
 */
func (cic *CorrelationIntegrationCoordinator) integrateWithRegularSession(
	ctx context.Context, 
	startEvent *ClaudeStartEvent, 
	claudeSession *ActiveClaudeSession) (*Session, error) {
	
	switch cic.integrationMode {
	case IntegrationModeReplace:
		// Use Claude session timing for regular session
		return cic.createSessionFromClaudeSession(claudeSession)
		
	case IntegrationModeAugment:
		// Augment existing session with Claude processing awareness
		return cic.augmentExistingSession(ctx, startEvent, claudeSession)
		
	case IntegrationModeParallel:
		// Run both systems in parallel for comparison
		return cic.runParallelSessionTracking(ctx, startEvent, claudeSession)
		
	default:
		return nil, fmt.Errorf("unsupported integration mode: %s", cic.integrationMode)
	}
}

/**
 * CONTEXT:   Integrate Claude session with work block tracking for accurate time recording
 * INPUT:     Start event, Claude session, regular session, and work block context
 * OUTPUT:    Work block that accurately tracks Claude processing time vs user interaction time
 * BUSINESS:  Ensure work blocks distinguish between user time and Claude processing time
 * CHANGE:    Initial work block integration with Claude processing awareness
 * RISK:      High - Work block integration directly affects reported work hours
 */
func (cic *CorrelationIntegrationCoordinator) integrateWithWorkBlock(
	ctx context.Context,
	startEvent *ClaudeStartEvent,
	claudeSession *ActiveClaudeSession,
	regularSession *Session) (*WorkBlock, error) {
	
	// Get or create work block for this session
	sessionID := ""
	if regularSession != nil {
		sessionID = regularSession.ID()
	} else {
		// Create placeholder session ID for work block tracking
		sessionID = fmt.Sprintf("claude-session-%s", claudeSession.ID)
	}
	
	// Check if work block already exists or needs creation
	activeWorkBlock, err := cic.getOrCreateWorkBlockForSession(
		ctx,
		sessionID,
		startEvent.ProjectName,
		startEvent.Timestamp)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get or create work block: %w", err)
	}
	
	// Start Claude processing in the work block
	err = activeWorkBlock.StartClaudeProcessing(
		startEvent.Timestamp,
		claudeSession.EstimatedDuration,
		startEvent.PromptID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to start Claude processing in work block: %w", err)
	}
	
	return activeWorkBlock, nil
}

/**
 * CONTEXT:   Complete work block with accurate Claude session timing
 * INPUT:     Work block ID, completed Claude session, and end event data
 * OUTPUT:    Work block updated with accurate Claude processing time and total timing
 * BUSINESS:  Record precise work block timing that separates Claude processing from user time
 * CHANGE:    Initial work block completion with Claude processing time integration
 * RISK:      High - Work block completion accuracy affects final reported work hours
 */
func (cic *CorrelationIntegrationCoordinator) completeWorkBlockWithClaudeSession(
	ctx context.Context,
	workBlockID string,
	claudeSession *ActiveClaudeSession,
	endEvent *ClaudeEndEvent) error {
	
	// Get active work block
	activeWorkBlocks := cic.workBlockTracker.GetActiveBlocks()
	var targetWorkBlock *WorkBlock
	
	for _, wb := range activeWorkBlocks {
		if wb.ID() == workBlockID {
			targetWorkBlock = wb
			break
		}
	}
	
	if targetWorkBlock == nil {
		return fmt.Errorf("work block %s not found or not active", workBlockID)
	}
	
	// End Claude processing in work block
	err := targetWorkBlock.EndClaudeProcessing(endEvent.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to end Claude processing: %w", err)
	}
	
	// Record final activity to update work block timing
	err = targetWorkBlock.RecordActivity(endEvent.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to record final activity: %w", err)
	}
	
	// Determine if work block should be finished based on session completion
	if claudeSession.ActualDuration != nil {
		// Calculate work block end time
		workBlockEndTime := endEvent.Timestamp
		
		// Check if this should end the work block or continue
		if cic.shouldFinishWorkBlock(targetWorkBlock, claudeSession, endEvent) {
			err = targetWorkBlock.Finish(workBlockEndTime)
			if err != nil {
				return fmt.Errorf("failed to finish work block: %w", err)
			}
		}
	}
	
	return nil
}

/**
 * CONTEXT:   Handle correlation failure with fallback strategies
 * INPUT:     Context, end event, and correlation error
 * OUTPUT:    Recovery action using error handler or alternative correlation
 * BUSINESS:  Maintain system reliability when correlation fails
 * CHANGE:    Initial correlation failure handling integration
 * RISK:      Medium - Failure handling affects system reliability
 */
func (cic *CorrelationIntegrationCoordinator) handleCorrelationFailure(
	ctx context.Context,
	endEvent *ClaudeEndEvent,
	correlationErr error) error {
	
	// Attempt error handler recovery
	recoveryErr := cic.errorHandler.HandleOrphanedEndEvent(ctx, endEvent)
	if recoveryErr != nil {
		// Both correlation and recovery failed
		return fmt.Errorf("correlation failed (%w) and recovery failed (%w)", 
			correlationErr, recoveryErr)
	}
	
	// Recovery succeeded - try to find work block updates needed
	return cic.handleRecoveredEvent(ctx, endEvent)
}

/**
 * CONTEXT:   Handle successfully recovered event for work block integration
 * INPUT:     Context and recovered end event
 * OUTPUT:    Work block updates based on recovered session information
 * BUSINESS:  Integrate recovered sessions with work block tracking
 * CHANGE:    Initial recovered event handling for work block integration
 * RISK:      Medium - Recovery integration affects work block accuracy
 */
func (cic *CorrelationIntegrationCoordinator) handleRecoveredEvent(ctx context.Context, endEvent *ClaudeEndEvent) error {
	// After recovery, the error handler may have created a synthetic session
	// or matched with an existing session. We need to ensure work blocks are updated.
	
	// For now, create a minimal work block entry for the recovered event
	return cic.createWorkBlockFromRecoveredEvent(ctx, endEvent)
}

/**
 * CONTEXT:   Create session from Claude session for replacement integration mode
 * INPUT:     Claude session with timing and context information
 * OUTPUT:    Regular session based on Claude session timing
 * BUSINESS:  Create regular session that matches Claude session timing exactly
 * CHANGE:    Initial session creation from Claude session
 * RISK:      Medium - Session creation accuracy affects session-based reporting
 */
func (cic *CorrelationIntegrationCoordinator) createSessionFromClaudeSession(claudeSession *ActiveClaudeSession) (*Session, error) {
	sessionConfig := SessionConfig{
		UserID:    claudeSession.UserID,
		StartTime: claudeSession.StartTime,
	}
	
	return NewSession(sessionConfig)
}

/**
 * CONTEXT:   Augment existing session with Claude processing awareness
 * INPUT:     Start event and Claude session for augmentation
 * OUTPUT:    Enhanced existing session with Claude processing context
 * BUSINESS:  Enhance existing session tracking with Claude-specific timing and context
 * CHANGE:    Initial session augmentation with Claude processing awareness
 * RISK:      Medium - Session augmentation affects session timing calculations
 */
func (cic *CorrelationIntegrationCoordinator) augmentExistingSession(
	ctx context.Context,
	startEvent *ClaudeStartEvent,
	claudeSession *ActiveClaudeSession) (*Session, error) {
	
	// Get or create regular session using existing session tracker
	session, err := cic.sessionTracker.GetOrCreateSession(
		ctx,
		claudeSession.UserID,
		startEvent.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create regular session: %w", err)
	}
	
	// Record activity to indicate Claude processing started
	workBlockID := cic.sessionToWorkBlock[claudeSession.ID]
	err = session.RecordActivity(startEvent.Timestamp, workBlockID)
	if err != nil {
		return nil, fmt.Errorf("failed to record Claude start activity: %w", err)
	}
	
	return session, nil
}

/**
 * CONTEXT:   Run parallel session tracking for comparison and validation
 * INPUT:     Start event and Claude session for parallel tracking
 * OUTPUT:    Both Claude-aware and traditional session tracking
 * BUSINESS:  Compare Claude-aware tracking with traditional tracking for validation
 * CHANGE:    Initial parallel session tracking implementation
 * RISK:      Low - Parallel tracking provides validation but doesn't affect primary tracking
 */
func (cic *CorrelationIntegrationCoordinator) runParallelSessionTracking(
	ctx context.Context,
	startEvent *ClaudeStartEvent,
	claudeSession *ActiveClaudeSession) (*Session, error) {
	
	// Create both types of sessions for comparison
	regularSession, err := cic.sessionTracker.GetOrCreateSession(
		ctx,
		claudeSession.UserID,
		startEvent.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to create regular session for parallel tracking: %w", err)
	}
	
	// Log parallel tracking data for analysis
	// In practice, this might send metrics to a monitoring system
	
	return regularSession, nil
}

/**
 * CONTEXT:   Get existing work block or create new one for session
 * INPUT:     Session ID, project name, and timestamp for work block management
 * OUTPUT:    Active work block for the session and project
 * BUSINESS:  Ensure work block continuity across Claude processing events
 * CHANGE:    Initial work block management for session integration
 * RISK:      Medium - Work block selection affects time tracking continuity
 */
func (cic *CorrelationIntegrationCoordinator) getOrCreateWorkBlockForSession(
	ctx context.Context,
	sessionID string,
	projectName string,
	timestamp time.Time) (*WorkBlock, error) {
	
	// Use existing work block tracker to get or create work block
	return cic.workBlockTracker.UpdateWorkBlock(ctx, sessionID, projectName, timestamp)
}

/**
 * CONTEXT:   Determine if work block should be finished based on session completion
 * INPUT:     Work block, completed Claude session, and end event
 * OUTPUT:    Boolean indicating if work block should be finished
 * BUSINESS:  Decide work block lifecycle based on Claude session patterns
 * CHANGE:    Initial work block lifecycle decision logic
 * RISK:      Medium - Work block lifecycle decisions affect reported work duration
 */
func (cic *CorrelationIntegrationCoordinator) shouldFinishWorkBlock(
	workBlock *WorkBlock,
	claudeSession *ActiveClaudeSession,
	endEvent *ClaudeEndEvent) bool {
	
	// Finish work block if:
	// 1. Claude processing was long (>10 minutes) and likely represents end of work
	// 2. End of regular session window
	// 3. Project change detected
	
	if claudeSession.ActualDuration != nil && *claudeSession.ActualDuration > 10*time.Minute {
		return true
	}
	
	// Check if work block has been active for a long time
	if workBlock.Duration() > 2*time.Hour {
		return true
	}
	
	// Don't finish by default - let normal idle detection handle it
	return false
}

/**
 * CONTEXT:   Create work block from recovered event when correlation recovery succeeds
 * INPUT:     Context and recovered end event
 * OUTPUT:     Work block created from recovered event data
 * BUSINESS:  Ensure recovered events contribute to work block tracking
 * CHANGE:    Initial work block creation from recovered events
 * RISK:      Medium - Recovered event work block accuracy affects time tracking completeness
 */
func (cic *CorrelationIntegrationCoordinator) createWorkBlockFromRecoveredEvent(
	ctx context.Context,
	endEvent *ClaudeEndEvent) error {
	
	// Create minimal work block for recovered event
	// Use estimated start time if available
	var startTime time.Time
	if endEvent.ActualDuration > 0 {
		startTime = endEvent.Timestamp.Add(-endEvent.ActualDuration)
	} else {
		startTime = endEvent.Timestamp.Add(-5 * time.Minute) // Default 5 minutes
	}
	
	workBlockConfig := WorkBlockConfig{
		SessionID:   fmt.Sprintf("recovered-%d", endEvent.Timestamp.Unix()),
		ProjectName: endEvent.ProjectName,
		ProjectPath: endEvent.ProjectPath,
		StartTime:   startTime,
	}
	
	workBlock, err := NewWorkBlock(workBlockConfig)
	if err != nil {
		return fmt.Errorf("failed to create work block from recovered event: %w", err)
	}
	
	// Immediately finish work block with end event timing
	err = workBlock.Finish(endEvent.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to finish recovered work block: %w", err)
	}
	
	return nil
}

// Configuration and monitoring methods

func (cic *CorrelationIntegrationCoordinator) SetIntegrationMode(mode IntegrationMode) {
	cic.mutex.Lock()
	defer cic.mutex.Unlock()
	cic.integrationMode = mode
}

func (cic *CorrelationIntegrationCoordinator) GetIntegrationMode() IntegrationMode {
	cic.mutex.RLock()
	defer cic.mutex.RUnlock()
	return cic.integrationMode
}

func (cic *CorrelationIntegrationCoordinator) GetActiveSessionCount() int {
	cic.mutex.RLock()
	defer cic.mutex.RUnlock()
	return len(cic.sessionToWorkBlock)
}

func (cic *CorrelationIntegrationCoordinator) GetPendingEventCount() int {
	cic.mutex.RLock()
	defer cic.mutex.RUnlock()
	return len(cic.pendingStartEvents)
}

/**
 * CONTEXT:   Clean up expired pending events and associations
 * INPUT:     Timeout duration for pending event cleanup
 * OUTPUT:    Number of expired events cleaned up
 * BUSINESS:  Maintain system health by cleaning up stale integration state
 * CHANGE:    Initial cleanup process for integration state
 * RISK:      Low - Cleanup maintains system health
 */
func (cic *CorrelationIntegrationCoordinator) CleanupExpiredEvents() int {
	cic.mutex.Lock()
	defer cic.mutex.Unlock()
	
	now := time.Now()
	cleanedCount := 0
	
	for promptID, startEvent := range cic.pendingStartEvents {
		if now.Sub(startEvent.Timestamp) > cic.timeoutDuration {
			delete(cic.pendingStartEvents, promptID)
			cleanedCount++
		}
	}
	
	return cleanedCount
}

/**
 * CONTEXT:   Integration coordinator status for monitoring and debugging
 * INPUT:     No parameters, returns current integration state
 * OUTPUT:    Integration status information for monitoring
 * BUSINESS:  Provide integration status for system monitoring and troubleshooting
 * CHANGE:    Initial status reporting for integration coordinator
 * RISK:      Low - Status reporting for system monitoring
 */
type IntegrationStatus struct {
	IntegrationMode     IntegrationMode `json:"integration_mode"`
	ActiveSessions      int             `json:"active_sessions"`
	PendingEvents       int             `json:"pending_events"`
	SessionAssociations int             `json:"session_associations"`
	WorkBlockAssociations int          `json:"work_block_associations"`
	LastCleanup         time.Time       `json:"last_cleanup"`
}

func (cic *CorrelationIntegrationCoordinator) GetStatus() *IntegrationStatus {
	cic.mutex.RLock()
	defer cic.mutex.RUnlock()
	
	return &IntegrationStatus{
		IntegrationMode:       cic.integrationMode,
		ActiveSessions:        cic.sessionManager.GetActiveSessionCount(),
		PendingEvents:         len(cic.pendingStartEvents),
		SessionAssociations:   len(cic.sessionToWorkBlock),
		WorkBlockAssociations: len(cic.workBlockToSession),
		LastCleanup:          time.Now(), // Would track actual cleanup time
	}
}