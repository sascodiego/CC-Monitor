package daemon

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-011
 * CONTEXT:   Enhanced session manager with precise 5-hour window logic and concurrent safety
 * REASON:    Business requirement for exact session boundaries independent of activity level
 * CHANGE:    Enhanced implementation with atomic operations and improved state management.
 * PREVENTION:Always use atomic operations for timestamp access, ensure session boundaries are never violated
 * RISK:      High - Race conditions could cause session overlap or incorrect billing logic
 */

// DefaultSessionManager implements the SessionManager interface with enhanced concurrent safety
type DefaultSessionManager struct {
	dbManager    arch.DatabaseManager
	timeProvider arch.TimeProvider
	logger       arch.Logger
	
	// Session state with atomic access for precise timing
	mu                    sync.RWMutex
	currentSession        *domain.Session
	currentSessionEndTime int64 // atomic Unix timestamp for concurrent access
	lastInteractionTime   int64 // atomic Unix timestamp
	sessionStartCount     int64 // atomic counter for session statistics
}

// NewSessionManager creates a new session manager with enhanced state tracking
func NewSessionManager(dbManager arch.DatabaseManager, timeProvider arch.TimeProvider, logger arch.Logger) *DefaultSessionManager {
	return &DefaultSessionManager{
		dbManager:             dbManager,
		timeProvider:          timeProvider,
		logger:                logger,
		currentSessionEndTime: 0, // No active session initially
		lastInteractionTime:   0,
		sessionStartCount:     0,
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-012
 * CONTEXT:   Enhanced session interaction handling with precise 5-hour window business logic
 * REASON:    Business requirement: "New session starts when: now > currentSessionEndTime"
 * CHANGE:    Enhanced with atomic timestamp operations and precise session boundary detection.
 * PREVENTION:Use atomic operations for all time comparisons, validate session state transitions
 * RISK:      High - Incorrect session boundaries could cause billing inaccuracies
 */

// HandleInteraction processes user interaction events with precise 5-hour session window logic
func (dsm *DefaultSessionManager) HandleInteraction(timestamp time.Time) (*domain.Session, error) {
	// Use atomic load for concurrent-safe session end time check
	currentEndTime := atomic.LoadInt64(&dsm.currentSessionEndTime)
	timestampUnix := timestamp.Unix()
	
	// Fast path: check if we need a new session without lock
	if currentEndTime == 0 || timestampUnix > currentEndTime {
		// Need new session - acquire write lock
		dsm.mu.Lock()
		defer dsm.mu.Unlock()
		
		// Double-check after acquiring lock (avoid race condition)
		currentEndTime = atomic.LoadInt64(&dsm.currentSessionEndTime)
		if currentEndTime == 0 || timestampUnix > currentEndTime {
			if err := dsm.startNewSession(timestamp); err != nil {
				return nil, fmt.Errorf("failed to start new session: %w", err)
			}
			
			dsm.logger.Info("Started new session", 
				"sessionID", dsm.currentSession.ID,
				"startTime", dsm.currentSession.StartTime,
				"endTime", dsm.currentSession.EndTime,
				"sessionCount", atomic.LoadInt64(&dsm.sessionStartCount))
		}
	} else {
		// Within existing session - acquire read lock for access
		dsm.mu.RLock()
		defer dsm.mu.RUnlock()
		
		dsm.logger.Debug("Interaction within current session", 
			"sessionID", dsm.currentSession.ID,
			"interactionTime", timestamp,
			"sessionEndTime", time.Unix(currentEndTime, 0))
	}
	
	// Update last interaction time atomically
	atomic.StoreInt64(&dsm.lastInteractionTime, timestampUnix)
	
	return dsm.currentSession, nil
}

// startNewSession creates and persists a new session with enhanced state management
func (dsm *DefaultSessionManager) startNewSession(startTime time.Time) error {
	// Finalize previous session if it exists
	if dsm.currentSession != nil {
		dsm.currentSession.IsActive = false
		if err := dsm.dbManager.SaveSession(dsm.currentSession); err != nil {
			dsm.logger.Error("Failed to finalize previous session", 
				"sessionID", dsm.currentSession.ID,
				"error", err)
			// Continue with new session creation despite finalization error
		}
		dsm.logger.Info("Finalized previous session", 
			"sessionID", dsm.currentSession.ID,
			"duration", dsm.currentSession.Duration())
	}
	
	// Create new session with exactly 5-hour window
	dsm.currentSession = domain.NewSession(startTime)
	
	// Atomically update session end time for concurrent access
	atomic.StoreInt64(&dsm.currentSessionEndTime, dsm.currentSession.EndTime.Unix())
	
	// Increment session counter
	atomic.AddInt64(&dsm.sessionStartCount, 1)
	
	// Persist new session
	if err := dsm.dbManager.SaveSession(dsm.currentSession); err != nil {
		return fmt.Errorf("failed to save new session: %w", err)
	}
	
	dsm.logger.Debug("New session created", 
		"sessionID", dsm.currentSession.ID,
		"startTime", dsm.currentSession.StartTime,
		"endTime", dsm.currentSession.EndTime,
		"duration", "5h0m0s")
	
	return nil
}

// GetCurrentSession returns the active session with concurrent-safe timing check
func (dsm *DefaultSessionManager) GetCurrentSession() (*domain.Session, bool) {
	// Fast path: atomic check of session validity
	currentEndTime := atomic.LoadInt64(&dsm.currentSessionEndTime)
	if currentEndTime == 0 {
		return nil, false // No session active
	}
	
	now := dsm.timeProvider.Now().Unix()
	if now > currentEndTime {
		return nil, false // Session expired
	}
	
	// Session is valid - acquire read lock for safe access
	dsm.mu.RLock()
	defer dsm.mu.RUnlock()
	
	// Double-check session state after acquiring lock
	if dsm.currentSession != nil && dsm.currentSession.IsActive {
		return dsm.currentSession, true
	}
	
	return nil, false
}

// IsSessionActive checks if there is currently an active session
func (dsm *DefaultSessionManager) IsSessionActive() bool {
	_, active := dsm.GetCurrentSession()
	return active
}

// FinalizeCurrentSession ends the current session with enhanced state cleanup
func (dsm *DefaultSessionManager) FinalizeCurrentSession() error {
	dsm.mu.Lock()
	defer dsm.mu.Unlock()
	
	if dsm.currentSession == nil {
		dsm.logger.Debug("No current session to finalize")
		return nil
	}
	
	sessionID := dsm.currentSession.ID
	sessionDuration := dsm.currentSession.Duration()
	
	dsm.logger.Info("Finalizing current session", 
		"sessionID", sessionID,
		"duration", sessionDuration,
		"startTime", dsm.currentSession.StartTime)
	
	// Mark session as inactive
	dsm.currentSession.IsActive = false
	
	// Persist final session state
	if err := dsm.dbManager.SaveSession(dsm.currentSession); err != nil {
		return fmt.Errorf("failed to finalize session %s: %w", sessionID, err)
	}
	
	// Clear current session state atomically
	dsm.currentSession = nil
	atomic.StoreInt64(&dsm.currentSessionEndTime, 0)
	atomic.StoreInt64(&dsm.lastInteractionTime, 0)
	
	dsm.logger.Info("Session finalized successfully", 
		"sessionID", sessionID,
		"finalDuration", sessionDuration)
	
	return nil
}

// GetSessionStats returns statistics for the specified time period
func (dsm *DefaultSessionManager) GetSessionStats(period arch.TimePeriod) (*domain.SessionStats, error) {
	return dsm.dbManager.GetSessionStats(period)
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-013
 * CONTEXT:   Enhanced work block manager with precise 5-minute inactivity timeout logic
 * REASON:    Business requirement: "New work block starts when: now - lastActivityTime > 5 minutes"
 * CHANGE:    Enhanced with atomic timestamp operations and precise timeout handling.
 * PREVENTION:Always ensure work blocks never extend beyond their containing session boundaries
 * RISK:      Medium - Work block timing errors could cause inaccurate billing
 */

// DefaultWorkBlockManager implements enhanced work block management with precise timeout logic
type DefaultWorkBlockManager struct {
	dbManager    arch.DatabaseManager
	timeProvider arch.TimeProvider
	logger       arch.Logger
	
	// Work block state with atomic access for concurrent safety
	mu                     sync.RWMutex
	currentBlock           *domain.WorkBlock
	lastActivityTime       int64 // atomic Unix timestamp
	currentBlockStartTime  int64 // atomic Unix timestamp
	workBlockCount         int64 // atomic counter for statistics
	
	// Timeout configuration (5 minutes as per business rules)
	inactivityTimeout      time.Duration
}

// NewWorkBlockManager creates a new work block manager with enhanced timeout handling
func NewWorkBlockManager(dbManager arch.DatabaseManager, timeProvider arch.TimeProvider, logger arch.Logger) *DefaultWorkBlockManager {
	return &DefaultWorkBlockManager{
		dbManager:             dbManager,
		timeProvider:          timeProvider,
		logger:                logger,
		lastActivityTime:      0, // No activity initially
		currentBlockStartTime: 0,
		workBlockCount:        0,
		inactivityTimeout:     5 * time.Minute, // Business rule: 5-minute timeout
	}
}

// RecordActivity processes activity events with precise 5-minute timeout logic
func (dwm *DefaultWorkBlockManager) RecordActivity(sessionID string, timestamp time.Time) (*domain.WorkBlock, error) {
	timestampUnix := timestamp.Unix()
	lastActivity := atomic.LoadInt64(&dwm.lastActivityTime)
	
	// Fast path: check if we need a new work block without lock
	needNewBlock := dwm.currentBlock == nil || 
					 dwm.currentBlock.SessionID != sessionID ||
					 (lastActivity > 0 && timestampUnix-lastActivity > int64(dwm.inactivityTimeout.Seconds()))
	
	if needNewBlock {
		// Need new work block - acquire write lock
		dwm.mu.Lock()
		defer dwm.mu.Unlock()
		
		// Double-check conditions after acquiring lock
		lastActivity = atomic.LoadInt64(&dwm.lastActivityTime)
		inactivityExceeded := lastActivity > 0 && timestampUnix-lastActivity > int64(dwm.inactivityTimeout.Seconds())
		
		if dwm.currentBlock == nil {
			dwm.logger.Debug("No current work block, starting new one")
		} else if dwm.currentBlock.SessionID != sessionID {
			dwm.logger.Info("Session changed, finalizing current work block and starting new one",
				"oldSessionID", dwm.currentBlock.SessionID,
				"newSessionID", sessionID)
			if err := dwm.finalizeCurrentBlock(); err != nil {
				dwm.logger.Error("Failed to finalize work block on session change", "error", err)
			}
		} else if inactivityExceeded {
			inactiveMinutes := float64(timestampUnix-lastActivity) / 60.0
			dwm.logger.Info("Inactivity timeout exceeded, finalizing current work block",
				"blockID", dwm.currentBlock.ID,
				"inactiveMinutes", inactiveMinutes,
				"timeoutMinutes", dwm.inactivityTimeout.Minutes())
			if err := dwm.finalizeCurrentBlock(); err != nil {
				dwm.logger.Error("Failed to finalize inactive work block", "error", err)
			}
		}
		
		// Start new work block
		if err := dwm.startNewWorkBlock(sessionID, timestamp); err != nil {
			return nil, fmt.Errorf("failed to start new work block: %w", err)
		}
		
		dwm.logger.Info("Started new work block", 
			"blockID", dwm.currentBlock.ID,
			"sessionID", sessionID,
			"startTime", dwm.currentBlock.StartTime,
			"workBlockCount", atomic.LoadInt64(&dwm.workBlockCount))
	} else {
		// Within existing work block - acquire read lock for safe access
		dwm.mu.RLock()
		defer dwm.mu.RUnlock()
		
		dwm.logger.Debug("Activity within current work block",
			"blockID", dwm.currentBlock.ID,
			"activityTime", timestamp)
	}
	
	// Update activity timestamp atomically and in work block
	atomic.StoreInt64(&dwm.lastActivityTime, timestampUnix)
	if dwm.currentBlock != nil {
		dwm.currentBlock.UpdateActivity(timestamp)
	}
	
	return dwm.currentBlock, nil
}

// startNewWorkBlock creates and persists a new work block with enhanced state management
func (dwm *DefaultWorkBlockManager) startNewWorkBlock(sessionID string, startTime time.Time) error {
	// Create new work block
	dwm.currentBlock = domain.NewWorkBlock(sessionID, startTime)
	
	// Update atomic state
	atomic.StoreInt64(&dwm.currentBlockStartTime, startTime.Unix())
	atomic.StoreInt64(&dwm.lastActivityTime, startTime.Unix())
	atomic.AddInt64(&dwm.workBlockCount, 1)
	
	// Persist new work block
	if err := dwm.dbManager.SaveWorkBlock(dwm.currentBlock); err != nil {
		return fmt.Errorf("failed to save new work block: %w", err)
	}
	
	dwm.logger.Debug("New work block created", 
		"blockID", dwm.currentBlock.ID,
		"sessionID", sessionID,
		"startTime", startTime)
	
	return nil
}

// GetActiveBlock returns the current active work block with timeout validation
func (dwm *DefaultWorkBlockManager) GetActiveBlock() (*domain.WorkBlock, bool) {
	// Fast path: check if block exists and hasn't timed out
	lastActivity := atomic.LoadInt64(&dwm.lastActivityTime)
	if lastActivity == 0 {
		return nil, false // No active block
	}
	
	// Check for timeout
	now := dwm.timeProvider.Now().Unix()
	if now-lastActivity > int64(dwm.inactivityTimeout.Seconds()) {
		return nil, false // Block has timed out
	}
	
	// Block is valid - acquire read lock for safe access
	dwm.mu.RLock()
	defer dwm.mu.RUnlock()
	
	if dwm.currentBlock != nil && dwm.currentBlock.IsActive {
		return dwm.currentBlock, true
	}
	
	return nil, false
}

// finalizeCurrentBlock ends the current work block with enhanced duration calculation
func (dwm *DefaultWorkBlockManager) finalizeCurrentBlock() error {
	if dwm.currentBlock == nil {
		dwm.logger.Debug("No current work block to finalize")
		return nil
	}
	
	blockID := dwm.currentBlock.ID
	sessionID := dwm.currentBlock.SessionID
	
	// Use last activity time as end time for accurate duration
	lastActivity := atomic.LoadInt64(&dwm.lastActivityTime)
	var endTime time.Time
	if lastActivity > 0 {
		endTime = time.Unix(lastActivity, 0)
	} else {
		endTime = dwm.timeProvider.Now()
	}
	
	// Finalize the work block
	dwm.currentBlock.Finalize(endTime)
	duration := dwm.currentBlock.Duration()
	
	// Persist finalized work block
	if err := dwm.dbManager.SaveWorkBlock(dwm.currentBlock); err != nil {
		return fmt.Errorf("failed to save finalized work block %s: %w", blockID, err)
	}
	
	dwm.logger.Info("Work block finalized", 
		"blockID", blockID,
		"sessionID", sessionID,
		"duration", duration,
		"durationSeconds", int64(duration.Seconds()),
		"startTime", dwm.currentBlock.StartTime,
		"endTime", endTime)
	
	return nil
}

// FinalizeCurrentBlock ends the current work block with enhanced state cleanup
func (dwm *DefaultWorkBlockManager) FinalizeCurrentBlock() error {
	dwm.mu.Lock()
	defer dwm.mu.Unlock()
	
	if dwm.currentBlock == nil {
		dwm.logger.Debug("No current work block to finalize")
		return nil
	}
	
	blockID := dwm.currentBlock.ID
	if err := dwm.finalizeCurrentBlock(); err != nil {
		return fmt.Errorf("failed to finalize work block %s: %w", blockID, err)
	}
	
	// Clear current block state atomically
	dwm.currentBlock = nil
	atomic.StoreInt64(&dwm.currentBlockStartTime, 0)
	atomic.StoreInt64(&dwm.lastActivityTime, 0)
	
	dwm.logger.Debug("Work block state cleared", "blockID", blockID)
	return nil
}

// CheckInactivityTimeout checks if current block should be finalized due to precise 5-minute timeout
func (dwm *DefaultWorkBlockManager) CheckInactivityTimeout(currentTime time.Time) error {
	// Fast path: atomic check for timeout without lock
	lastActivity := atomic.LoadInt64(&dwm.lastActivityTime)
	if lastActivity == 0 {
		return nil // No active block
	}
	
	currentTimeUnix := currentTime.Unix()
	inactivityDuration := currentTimeUnix - lastActivity
	
	// Check if timeout exceeded
	if inactivityDuration <= int64(dwm.inactivityTimeout.Seconds()) {
		return nil // Still within timeout window
	}
	
	// Timeout exceeded - acquire write lock for finalization
	dwm.mu.Lock()
	defer dwm.mu.Unlock()
	
	// Double-check after acquiring lock
	lastActivity = atomic.LoadInt64(&dwm.lastActivityTime)
	if lastActivity == 0 || dwm.currentBlock == nil {
		return nil // Block already finalized
	}
	
	inactivityDuration = currentTimeUnix - lastActivity
	if inactivityDuration <= int64(dwm.inactivityTimeout.Seconds()) {
		return nil // Timeout window changed while acquiring lock
	}
	
	// Finalize due to timeout
	inactiveMinutes := float64(inactivityDuration) / 60.0
	dwm.logger.Info("Work block inactivity timeout reached", 
		"blockID", dwm.currentBlock.ID,
		"lastActivity", time.Unix(lastActivity, 0),
		"inactiveMinutes", inactiveMinutes,
		"timeoutMinutes", dwm.inactivityTimeout.Minutes())
	
	if err := dwm.finalizeCurrentBlock(); err != nil {
		return fmt.Errorf("failed to finalize inactive work block: %w", err)
	}
	
	// Clear state
	dwm.currentBlock = nil
	atomic.StoreInt64(&dwm.currentBlockStartTime, 0)
	atomic.StoreInt64(&dwm.lastActivityTime, 0)
	
	return nil
}