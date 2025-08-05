package daemon

import (
	"context"
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
 * CONTEXT:   State coordinator for thread-safe coordination between session and work block managers
 * REASON:    Multiple goroutines need consistent view of session and work block state
 * CHANGE:    Initial implementation with ordered locking and state change notifications.
 * PREVENTION:Always acquire locks in consistent order to prevent deadlocks
 * RISK:      High - Deadlocks could cause daemon to hang indefinitely
 */

type StateChangeType int32

const (
	SessionStarted StateChangeType = iota
	SessionExpired
	WorkBlockStarted
	WorkBlockFinished
	SystemShutdown
)

// StateChange represents a state transition in the system
type StateChange struct {
	Type      StateChangeType `json:"type"`
	SessionID string          `json:"sessionID,omitempty"`
	BlockID   string          `json:"blockID,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Data      interface{}     `json:"data,omitempty"`
}

// StateSubscriber defines the interface for state change listeners
type StateSubscriber interface {
	OnStateChange(change StateChange) error
	GetSubscriberID() string
}

// StateCoordinator manages coordination between session and work block managers
type StateCoordinator struct {
	sessionManager   arch.SessionManager
	workBlockManager arch.WorkBlockManager
	logger           arch.Logger
	
	// Ordered locking to prevent deadlocks (always acquire in this order)
	sessionLock      sync.RWMutex
	workBlockLock    sync.RWMutex
	subscriberLock   sync.RWMutex
	
	// State change notifications
	stateChangeCh    chan StateChange
	subscribers      map[string]StateSubscriber
	
	// Coordination state
	running          int32 // atomic bool
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	
	// Metrics
	stateChanges     int64 // atomic counter
	lastStateChange  int64 // atomic timestamp
}

// NewStateCoordinator creates a new state coordinator
func NewStateCoordinator(
	sessionMgr arch.SessionManager,
	workBlockMgr arch.WorkBlockManager,
	logger arch.Logger,
) *StateCoordinator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &StateCoordinator{
		sessionManager:   sessionMgr,
		workBlockManager: workBlockMgr,
		logger:           logger,
		stateChangeCh:    make(chan StateChange, 100),
		subscribers:      make(map[string]StateSubscriber),
		ctx:              ctx,
		cancel:           cancel,
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-012
 * CONTEXT:   State coordinator startup with notification processing
 * REASON:    Need coordinated state change processing and subscriber notifications
 * CHANGE:    Initial implementation with goroutine-based state processing.
 * PREVENTION:Ensure proper cleanup of notification goroutines on shutdown
 * RISK:      Medium - Goroutine leaks if not properly cancelled
 */

// Start begins state coordination and notification processing
func (sc *StateCoordinator) Start() error {
	if atomic.LoadInt32(&sc.running) == 1 {
		return fmt.Errorf("state coordinator already running")
	}
	
	sc.logger.Info("Starting state coordinator")
	
	// Start state change processor
	sc.wg.Add(1)
	go sc.processStateChanges()
	
	// Start health monitor
	sc.wg.Add(1)
	go sc.healthMonitor()
	
	atomic.StoreInt32(&sc.running, 1)
	sc.logger.Info("State coordinator started successfully")
	
	return nil
}

// Stop gracefully shuts down state coordination
func (sc *StateCoordinator) Stop() error {
	if atomic.LoadInt32(&sc.running) == 0 {
		return nil
	}
	
	sc.logger.Info("Stopping state coordinator")
	
	// Send shutdown notification
	sc.NotifyStateChange(StateChange{
		Type:      SystemShutdown,
		Timestamp: time.Now(),
	})
	
	// Signal shutdown
	sc.cancel()
	
	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		sc.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		sc.logger.Info("State coordinator stopped gracefully")
	case <-time.After(10 * time.Second):
		sc.logger.Warn("State coordinator shutdown timeout")
	}
	
	atomic.StoreInt32(&sc.running, 0)
	close(sc.stateChangeCh)
	
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-013
 * CONTEXT:   Consistent state access with deadlock prevention
 * REASON:    Need atomic view of session and work block state across components
 * CHANGE:    Initial implementation with ordered locking pattern.
 * PREVENTION:Always acquire locks in same order (session first, then work block)
 * RISK:      High - Lock ordering violations could cause deadlocks
 */

// GetConsistentState returns a consistent view of session and work block state
func (sc *StateCoordinator) GetConsistentState() (*domain.Session, *domain.WorkBlock, error) {
	// Always acquire locks in same order to prevent deadlock
	sc.sessionLock.RLock()
	defer sc.sessionLock.RUnlock()
	
	sc.workBlockLock.RLock()
	defer sc.workBlockLock.RUnlock()
	
	// Get current session
	session, sessionExists := sc.sessionManager.GetCurrentSession()
	
	// Get current work block
	workBlock, workBlockExists := sc.workBlockManager.GetActiveBlock()
	
	// Validate state consistency
	if sessionExists && workBlockExists {
		if workBlock.SessionID != session.ID {
			return nil, nil, fmt.Errorf("state inconsistency: work block %s not in session %s", 
				workBlock.ID, session.ID)
		}
	}
	
	// Return consistent state
	if sessionExists {
		if workBlockExists {
			return session, workBlock, nil
		}
		return session, nil, nil
	}
	
	return nil, nil, nil
}

// ValidateStateConsistency checks for state inconsistencies between managers
func (sc *StateCoordinator) ValidateStateConsistency() error {
	session, workBlock, err := sc.GetConsistentState()
	if err != nil {
		return err
	}
	
	// Additional consistency checks
	if session != nil && workBlock != nil {
		// Work block must be within session time boundaries
		if workBlock.StartTime.Before(session.StartTime) {
			return fmt.Errorf("work block starts before session: block=%v, session=%v", 
				workBlock.StartTime, session.StartTime)
		}
		
		if workBlock.EndTime != nil && workBlock.EndTime.After(session.EndTime) {
			return fmt.Errorf("work block ends after session: block=%v, session=%v", 
				workBlock.EndTime, session.EndTime)
		}
	}
	
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-014
 * CONTEXT:   State change notification system for component coordination
 * REASON:    Components need to be notified of state changes for proper coordination
 * CHANGE:    Initial implementation with subscriber pattern.
 * PREVENTION:Handle subscriber notification failures gracefully, prevent blocking
 * RISK:      Medium - Slow subscribers could block state change processing
 */

// NotifyStateChange sends a state change notification to all subscribers
func (sc *StateCoordinator) NotifyStateChange(change StateChange) {
	atomic.AddInt64(&sc.stateChanges, 1)
	atomic.StoreInt64(&sc.lastStateChange, time.Now().Unix())
	
	select {
	case sc.stateChangeCh <- change:
		sc.logger.Debug("State change notification sent", 
			"type", change.Type,
			"sessionID", change.SessionID,
			"blockID", change.BlockID)
	default:
		sc.logger.Warn("State change channel full, dropping notification",
			"type", change.Type)
	}
}

// Subscribe adds a state change subscriber
func (sc *StateCoordinator) Subscribe(subscriber StateSubscriber) error {
	if subscriber == nil {
		return fmt.Errorf("subscriber cannot be nil")
	}
	
	sc.subscriberLock.Lock()
	defer sc.subscriberLock.Unlock()
	
	subscriberID := subscriber.GetSubscriberID()
	if _, exists := sc.subscribers[subscriberID]; exists {
		return fmt.Errorf("subscriber %s already registered", subscriberID)
	}
	
	sc.subscribers[subscriberID] = subscriber
	sc.logger.Debug("State change subscriber registered", 
		"subscriberID", subscriberID,
		"totalSubscribers", len(sc.subscribers))
	
	return nil
}

// Unsubscribe removes a state change subscriber
func (sc *StateCoordinator) Unsubscribe(subscriberID string) error {
	sc.subscriberLock.Lock()
	defer sc.subscriberLock.Unlock()
	
	if _, exists := sc.subscribers[subscriberID]; !exists {
		return fmt.Errorf("subscriber %s not found", subscriberID)
	}
	
	delete(sc.subscribers, subscriberID)
	sc.logger.Debug("State change subscriber removed", 
		"subscriberID", subscriberID,
		"remainingSubscribers", len(sc.subscribers))
	
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-015
 * CONTEXT:   Background goroutines for state processing and health monitoring
 * REASON:    Need asynchronous processing of state changes and system health checks
 * CHANGE:    Initial implementation with concurrent processing.
 * PREVENTION:Handle subscriber errors gracefully, implement circuit breaker for failing subscribers
 * RISK:      Medium - Subscriber failures could affect overall system stability
 */

// processStateChanges processes state change notifications in background
func (sc *StateCoordinator) processStateChanges() {
	defer sc.wg.Done()
	
	sc.logger.Debug("State change processor started")
	
	for {
		select {
		case <-sc.ctx.Done():
			sc.logger.Debug("State change processor stopping")
			return
			
		case change := <-sc.stateChangeCh:
			sc.handleStateChange(change)
		}
	}
}

// handleStateChange processes a single state change and notifies subscribers
func (sc *StateCoordinator) handleStateChange(change StateChange) {
	sc.logger.Debug("Processing state change", 
		"type", change.Type,
		"sessionID", change.SessionID,
		"blockID", change.BlockID)
	
	// Get current subscribers
	sc.subscriberLock.RLock()
	subscribers := make([]StateSubscriber, 0, len(sc.subscribers))
	for _, subscriber := range sc.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	sc.subscriberLock.RUnlock()
	
	// Notify each subscriber (with timeout to prevent blocking)
	for _, subscriber := range subscribers {
		go sc.notifySubscriber(subscriber, change)
	}
}

// notifySubscriber notifies a single subscriber with timeout protection
func (sc *StateCoordinator) notifySubscriber(subscriber StateSubscriber, change StateChange) {
	defer func() {
		if r := recover(); r != nil {
			sc.logger.Error("Subscriber notification panic",
				"subscriberID", subscriber.GetSubscriberID(),
				"panic", r)
		}
	}()
	
	// Create timeout context for subscriber notification
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- subscriber.OnStateChange(change)
	}()
	
	select {
	case err := <-done:
		if err != nil {
			sc.logger.Error("Subscriber notification failed",
				"subscriberID", subscriber.GetSubscriberID(),
				"error", err)
		}
	case <-ctx.Done():
		sc.logger.Warn("Subscriber notification timeout",
			"subscriberID", subscriber.GetSubscriberID(),
			"timeout", "5s")
	}
}

// healthMonitor performs periodic health checks and consistency validation
func (sc *StateCoordinator) healthMonitor() {
	defer sc.wg.Done()
	
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	
	sc.logger.Debug("Health monitor started")
	
	for {
		select {
		case <-sc.ctx.Done():
			sc.logger.Debug("Health monitor stopping")
			return
			
		case <-ticker.C:
			sc.performHealthCheck()
		}
	}
}

// performHealthCheck validates system state consistency
func (sc *StateCoordinator) performHealthCheck() {
	// Validate state consistency
	if err := sc.ValidateStateConsistency(); err != nil {
		sc.logger.Error("State consistency validation failed", "error", err)
		
		// Notify about consistency issue
		sc.NotifyStateChange(StateChange{
			Type:      SystemShutdown, // Use as error signal
			Timestamp: time.Now(),
			Data:      fmt.Sprintf("Consistency error: %v", err),
		})
	}
	
	// Log health metrics
	stateChanges := atomic.LoadInt64(&sc.stateChanges)
	lastChange := atomic.LoadInt64(&sc.lastStateChange)
	
	sc.logger.Debug("State coordinator health check",
		"stateChanges", stateChanges,
		"lastStateChange", time.Unix(lastChange, 0),
		"subscribers", len(sc.subscribers),
		"running", atomic.LoadInt32(&sc.running))
}

// GetMetrics returns state coordinator metrics
func (sc *StateCoordinator) GetMetrics() map[string]interface{} {
	sc.subscriberLock.RLock()
	subscriberCount := len(sc.subscribers)
	sc.subscriberLock.RUnlock()
	
	return map[string]interface{}{
		"running":         atomic.LoadInt32(&sc.running) == 1,
		"stateChanges":    atomic.LoadInt64(&sc.stateChanges),
		"lastStateChange": time.Unix(atomic.LoadInt64(&sc.lastStateChange), 0),
		"subscribers":     subscriberCount,
		"channelDepth":    len(sc.stateChangeCh),
	}
}

// IsRunning returns true if the coordinator is currently running
func (sc *StateCoordinator) IsRunning() bool {
	return atomic.LoadInt32(&sc.running) == 1
}