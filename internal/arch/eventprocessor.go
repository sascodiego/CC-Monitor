package arch

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/claude-monitor/claude-monitor/pkg/events"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-024
 * CONTEXT:   Event-driven architecture implementation for processing eBPF events
 * REASON:    Decouple event capture from business logic processing with proper error handling
 * CHANGE:    Initial implementation.
 * PREVENTION:Limit event handlers per type to avoid performance bottlenecks, handle panics
 * RISK:      High - Event processing lag could cause session timing inaccuracies
 */

// DefaultEventProcessor implements the EventProcessor interface
type DefaultEventProcessor struct {
	handlers    []EventHandler
	eventCh     <-chan *events.SystemEvent
	stopCh      chan struct{}
	mu          sync.RWMutex
	running     bool
	stats       *EventProcessorStats
	logger      Logger
}

// NewDefaultEventProcessor creates a new event processor
func NewDefaultEventProcessor(eventCh <-chan *events.SystemEvent, logger Logger) *DefaultEventProcessor {
	return &DefaultEventProcessor{
		handlers: make([]EventHandler, 0),
		eventCh:  eventCh,
		stopCh:   make(chan struct{}),
		stats:    &EventProcessorStats{},
		logger:   logger,
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-025
 * CONTEXT:   Event handler registration with priority-based ordering
 * REASON:    Need deterministic handler execution order for business logic dependencies
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate handler interfaces and avoid duplicate registrations
 * RISK:      Medium - Handler registration errors could cause missed events
 */

// RegisterHandler adds an event handler with priority-based ordering
func (dep *DefaultEventProcessor) RegisterHandler(handler EventHandler) error {
	dep.mu.Lock()
	defer dep.mu.Unlock()
	
	if handler == nil {
		return fmt.Errorf("cannot register nil handler")
	}
	
	// Check for duplicate registration
	for _, existing := range dep.handlers {
		if existing == handler {
			return fmt.Errorf("handler already registered")
		}
	}
	
	dep.handlers = append(dep.handlers, handler)
	dep.stats.HandlersRegistered++
	
	// Sort handlers by priority (higher priority first)
	sort.Slice(dep.handlers, func(i, j int) bool {
		return dep.handlers[i].Priority() > dep.handlers[j].Priority()
	})
	
	dep.logger.Debug("Event handler registered", 
		"handler", fmt.Sprintf("%T", handler),
		"priority", handler.Priority(),
		"total_handlers", len(dep.handlers))
	
	return nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-026
 * CONTEXT:   Main event processing loop with error recovery and graceful shutdown
 * REASON:    Need robust event processing that can handle errors without stopping the daemon
 * CHANGE:    Initial implementation.
 * PREVENTION:Always recover from handler panics and log processing errors for debugging
 * RISK:      High - Event processing failures could cause system state corruption
 */

// Start begins the event processing loop
func (dep *DefaultEventProcessor) Start(ctx context.Context) error {
	dep.mu.Lock()
	if dep.running {
		dep.mu.Unlock()
		return fmt.Errorf("event processor already running")
	}
	dep.running = true
	dep.mu.Unlock()
	
	dep.logger.Info("Starting event processor", "handlers", len(dep.handlers))
	
	go func() {
		defer func() {
			dep.mu.Lock()
			dep.running = false
			dep.mu.Unlock()
		}()
		
		for {
			select {
			case event := <-dep.eventCh:
				if event != nil {
					dep.processEventSafely(event)
				}
				
			case <-dep.stopCh:
				dep.logger.Info("Event processor stopping")
				return
				
			case <-ctx.Done():
				dep.logger.Info("Event processor context cancelled")
				return
			}
		}
	}()
	
	return nil
}

// processEventSafely processes an event with panic recovery
func (dep *DefaultEventProcessor) processEventSafely(event *events.SystemEvent) {
	defer func() {
		if r := recover(); r != nil {
			dep.stats.ProcessingErrors++
			dep.logger.Error("Event processing panic", 
				"error", r,
				"event_type", event.Type,
				"event_pid", event.PID)
		}
	}()
	
	if err := dep.ProcessEvent(event); err != nil {
		dep.stats.ProcessingErrors++
		dep.logger.Error("Event processing error", 
			"error", err,
			"event_type", event.Type,
			"event_pid", event.PID)
	} else {
		dep.stats.EventsProcessed++
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-027
 * CONTEXT:   Event processing logic with handler dispatch and validation
 * REASON:    Core business logic that coordinates event handling across multiple handlers
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate events before processing and ensure handlers are called in priority order
 * RISK:      High - Event processing errors could cause missed interactions or state corruption
 */

// ProcessEvent processes a single system event through registered handlers
func (dep *DefaultEventProcessor) ProcessEvent(event *events.SystemEvent) error {
	if event == nil {
		return fmt.Errorf("cannot process nil event")
	}
	
	// Validate the event
	validator := &events.EventValidator{}
	if err := validator.Validate(event); err != nil {
		dep.stats.EventsDropped++
		return fmt.Errorf("invalid event: %w", err)
	}
	
	// Check if event is relevant for processing
	if !validator.IsRelevant(event) {
		dep.stats.EventsDropped++
		dep.logger.Debug("Dropping irrelevant event", 
			"event_type", event.Type,
			"command", event.Command)
		return nil
	}
	
	dep.logger.Debug("Processing event", 
		"event_type", event.Type,
		"pid", event.PID,
		"command", event.Command,
		"timestamp", event.Timestamp)
	
	// Process event through all capable handlers
	handledCount := 0
	dep.mu.RLock()
	handlers := make([]EventHandler, len(dep.handlers))
	copy(handlers, dep.handlers)
	dep.mu.RUnlock()
	
	for _, handler := range handlers {
		if handler.CanHandle(event.Type) {
			if err := handler.Handle(event); err != nil {
				dep.logger.Warn("Handler error", 
					"handler", fmt.Sprintf("%T", handler),
					"error", err,
					"event_type", event.Type)
				// Continue processing with other handlers
				continue
			}
			handledCount++
		}
	}
	
	if handledCount == 0 {
		dep.logger.Debug("No handlers found for event", "event_type", event.Type)
	}
	
	return nil
}

// Stop gracefully shuts down the event processor
func (dep *DefaultEventProcessor) Stop() error {
	dep.mu.RLock()
	running := dep.running
	dep.mu.RUnlock()
	
	if !running {
		return nil
	}
	
	dep.logger.Info("Stopping event processor")
	close(dep.stopCh)
	
	// Wait for processor to stop (with timeout could be added here)
	for {
		dep.mu.RLock()
		stillRunning := dep.running
		dep.mu.RUnlock()
		
		if !stillRunning {
			break
		}
		// Small sleep to avoid busy waiting
		// time.Sleep(10 * time.Millisecond)
	}
	
	dep.logger.Info("Event processor stopped", 
		"events_processed", dep.stats.EventsProcessed,
		"processing_errors", dep.stats.ProcessingErrors)
	
	return nil
}

// GetStats returns current processing statistics
func (dep *DefaultEventProcessor) GetStats() *EventProcessorStats {
	dep.mu.RLock()
	defer dep.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	return &EventProcessorStats{
		EventsProcessed:    dep.stats.EventsProcessed,
		EventsDropped:      dep.stats.EventsDropped,
		ProcessingErrors:   dep.stats.ProcessingErrors,
		HandlersRegistered: dep.stats.HandlersRegistered,
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-028
 * CONTEXT:   Base event handler implementation for common handler patterns
 * REASON:    Provide common functionality for event handlers to reduce boilerplate
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep base handler minimal and focused on common concerns only
 * RISK:      Low - Base handler is optional and handlers can implement interface directly
 */

// BaseEventHandler provides common functionality for event handlers
type BaseEventHandler struct {
	name        string
	priority    int
	eventTypes  []events.EventType
	logger      Logger
}

// NewBaseEventHandler creates a new base event handler
func NewBaseEventHandler(name string, priority int, eventTypes []events.EventType, logger Logger) *BaseEventHandler {
	return &BaseEventHandler{
		name:       name,
		priority:   priority,
		eventTypes: eventTypes,
		logger:     logger,
	}
}

// CanHandle returns true if this handler can process the given event type
func (beh *BaseEventHandler) CanHandle(eventType events.EventType) bool {
	for _, supportedType := range beh.eventTypes {
		if supportedType == eventType {
			return true
		}
	}
	return false
}

// Priority returns the handler priority
func (beh *BaseEventHandler) Priority() int {
	return beh.priority
}

// Name returns the handler name
func (beh *BaseEventHandler) Name() string {
	return beh.name
}

// LogDebug logs a debug message with handler context
func (beh *BaseEventHandler) LogDebug(msg string, fields ...interface{}) {
	if beh.logger != nil {
		allFields := append([]interface{}{"handler", beh.name}, fields...)
		beh.logger.Debug(msg, allFields...)
	}
}

// LogError logs an error message with handler context
func (beh *BaseEventHandler) LogError(msg string, fields ...interface{}) {
	if beh.logger != nil {
		allFields := append([]interface{}{"handler", beh.name}, fields...)
		beh.logger.Error(msg, allFields...)
	}
}

// LogInfo logs an info message with handler context
func (beh *BaseEventHandler) LogInfo(msg string, fields ...interface{}) {
	if beh.logger != nil {
		allFields := append([]interface{}{"handler", beh.name}, fields...)
		beh.logger.Info(msg, allFields...)
	}
}

// LogWarn logs a warning message with handler context
func (beh *BaseEventHandler) LogWarn(msg string, fields ...interface{}) {
	if beh.logger != nil {
		allFields := append([]interface{}{"handler", beh.name}, fields...)
		beh.logger.Warn(msg, allFields...)
	}
}