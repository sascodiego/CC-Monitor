package daemon

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/pkg/events"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-006
 * CONTEXT:   Event processor with concurrent goroutine coordination and backpressure handling
 * REASON:    Need high-performance event processing pipeline that can handle eBPF event bursts
 * CHANGE:    Initial implementation with producer-consumer pattern.
 * PREVENTION:Monitor channel fill levels, implement event dropping or batching under load
 * RISK:      Medium - Unbounded event queues could cause memory exhaustion
 */

// DefaultEventProcessor implements high-performance event processing with backpressure
type DefaultEventProcessor struct {
	eventCh     <-chan *events.SystemEvent
	handlers    []arch.EventHandler
	logger      arch.Logger
	
	// Worker pool configuration
	workerCount int
	bufferSize  int
	
	// Processing state
	running     int32 // atomic bool
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	
	// Metrics and monitoring
	metrics     *ProcessingMetrics
	mu          sync.RWMutex
}

// ProcessingMetrics tracks event processing performance
type ProcessingMetrics struct {
	EventsProcessed   int64     `json:"eventsProcessed"`
	EventsDropped     int64     `json:"eventsDropped"`
	ProcessingErrors  int64     `json:"processingErrors"`
	HandlersRegistered int      `json:"handlersRegistered"`
	WorkersActive     int      `json:"workersActive"`
	QueueDepth        int      `json:"queueDepth"`
	LastEventTime     time.Time `json:"lastEventTime"`
	AverageLatency    time.Duration `json:"averageLatency"`
}

// NewDefaultEventProcessor creates a new event processor with optimal defaults
func NewDefaultEventProcessor(eventCh <-chan *events.SystemEvent, logger arch.Logger) *DefaultEventProcessor {
	return &DefaultEventProcessor{
		eventCh:     eventCh,
		logger:      logger,
		workerCount: 4,  // Optimal for most systems
		bufferSize:  1000, // Buffer for burst handling
		handlers:    make([]arch.EventHandler, 0),
		metrics:     &ProcessingMetrics{},
	}
}

// NewEventProcessorWithConfig creates an event processor with custom configuration
func NewEventProcessorWithConfig(eventCh <-chan *events.SystemEvent, logger arch.Logger, workerCount, bufferSize int) *DefaultEventProcessor {
	return &DefaultEventProcessor{
		eventCh:     eventCh,
		logger:      logger,
		workerCount: workerCount,
		bufferSize:  bufferSize,
		handlers:    make([]arch.EventHandler, 0),
		metrics:     &ProcessingMetrics{},
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-007
 * CONTEXT:   Event processor startup with worker pool initialization
 * REASON:    Need coordinated startup of worker goroutines with proper error handling
 * CHANGE:    Initial implementation with worker pool pattern.
 * PREVENTION:Ensure all workers start successfully before marking processor as running
 * RISK:      Medium - Failed worker startup could cause event processing delays
 */

// Start begins event processing with worker pool
func (dep *DefaultEventProcessor) Start(ctx context.Context) error {
	dep.mu.Lock()
	defer dep.mu.Unlock()
	
	if atomic.LoadInt32(&dep.running) == 1 {
		return fmt.Errorf("event processor already running")
	}
	
	dep.logger.Info("Starting event processor", 
		"workers", dep.workerCount,
		"bufferSize", dep.bufferSize,
		"handlers", len(dep.handlers))
	
	// Create processor context
	dep.ctx, dep.cancel = context.WithCancel(ctx)
	
	// Sort handlers by priority (higher priority first)
	dep.sortHandlersByPriority()
	
	// Start worker pool
	for i := 0; i < dep.workerCount; i++ {
		dep.wg.Add(1)
		go dep.worker(i)
	}
	
	// Start metrics reporter
	dep.wg.Add(1)
	go dep.metricsReporter()
	
	// Start main event distributor
	dep.wg.Add(1)
	go dep.eventDistributor()
	
	atomic.StoreInt32(&dep.running, 1)
	dep.logger.Info("Event processor started successfully")
	
	return nil
}

// Stop gracefully shuts down event processing
func (dep *DefaultEventProcessor) Stop() error {
	dep.mu.Lock()
	defer dep.mu.Unlock()
	
	if atomic.LoadInt32(&dep.running) == 0 {
		return nil
	}
	
	dep.logger.Info("Stopping event processor")
	
	// Signal shutdown
	if dep.cancel != nil {
		dep.cancel()
	}
	
	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		dep.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		dep.logger.Info("Event processor stopped gracefully")
	case <-time.After(30 * time.Second):
		dep.logger.Warn("Event processor shutdown timeout, forcing exit")
	}
	
	atomic.StoreInt32(&dep.running, 0)
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-008
 * CONTEXT:   Event handler registration with priority ordering
 * REASON:    Need dynamic handler registration with proper ordering for event processing
 * CHANGE:    Initial implementation with priority-based ordering.
 * PREVENTION:Validate handler interfaces and prevent duplicate registrations
 * RISK:      Low - Handler registration errors are caught at startup
 */

// RegisterHandler adds an event handler with priority ordering
func (dep *DefaultEventProcessor) RegisterHandler(handler arch.EventHandler) error {
	dep.mu.Lock()
	defer dep.mu.Unlock()
	
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}
	
	// Check for duplicate handlers (by type)
	for _, existing := range dep.handlers {
		if fmt.Sprintf("%T", existing) == fmt.Sprintf("%T", handler) {
			return fmt.Errorf("handler of type %T already registered", handler)
		}
	}
	
	dep.handlers = append(dep.handlers, handler)
	dep.metrics.HandlersRegistered = len(dep.handlers)
	
	// Re-sort handlers if processor is running
	if atomic.LoadInt32(&dep.running) == 1 {
		dep.sortHandlersByPriority()
	}
	
	dep.logger.Debug("Registered event handler", 
		"handlerType", fmt.Sprintf("%T", handler),
		"priority", handler.Priority(),
		"totalHandlers", len(dep.handlers))
	
	return nil
}

// ProcessEvent processes a single event through all applicable handlers
func (dep *DefaultEventProcessor) ProcessEvent(event *events.SystemEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}
	
	start := time.Now()
	defer func() {
		atomic.AddInt64(&dep.metrics.EventsProcessed, 1)
		dep.updateLatencyMetrics(time.Since(start))
	}()
	
	// Process event through all applicable handlers
	var errors []error
	for _, handler := range dep.handlers {
		if handler.CanHandle(event.Type) {
			if err := handler.Handle(event); err != nil {
				atomic.AddInt64(&dep.metrics.ProcessingErrors, 1)
				errors = append(errors, fmt.Errorf("handler %T error: %w", handler, err))
				dep.logger.Error("Handler error", 
					"handlerType", fmt.Sprintf("%T", handler),
					"eventType", event.Type,
					"error", err)
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("processing errors: %v", errors)
	}
	
	return nil
}

// GetStats returns current processing statistics
func (dep *DefaultEventProcessor) GetStats() *arch.EventProcessorStats {
	dep.mu.RLock()
	defer dep.mu.RUnlock()
	
	return &arch.EventProcessorStats{
		EventsProcessed:    atomic.LoadInt64(&dep.metrics.EventsProcessed),
		EventsDropped:      atomic.LoadInt64(&dep.metrics.EventsDropped),
		ProcessingErrors:   atomic.LoadInt64(&dep.metrics.ProcessingErrors),
		HandlersRegistered: dep.metrics.HandlersRegistered,
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-009
 * CONTEXT:   Worker goroutines for concurrent event processing
 * REASON:    Need concurrent event processing to handle high-frequency eBPF events
 * CHANGE:    Initial implementation with worker pool pattern.
 * PREVENTION:Handle panics in worker goroutines to prevent process crashes
 * RISK:      High - Worker panic could bring down entire event processing
 */

// worker processes events from the event channel
func (dep *DefaultEventProcessor) worker(workerID int) {
	defer dep.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			dep.logger.Error("Worker panic recovered", 
				"workerID", workerID,
				"panic", r)
		}
	}()
	
	dep.logger.Debug("Event worker started", "workerID", workerID)
	
	for {
		select {
		case <-dep.ctx.Done():
			dep.logger.Debug("Event worker stopping", "workerID", workerID)
			return
			
		case event := <-dep.eventCh:
			if event != nil {
				dep.metrics.LastEventTime = time.Now()
				if err := dep.ProcessEvent(event); err != nil {
					dep.logger.Debug("Event processing error", 
						"workerID", workerID,
						"error", err)
				}
			}
		}
	}
}

// eventDistributor monitors the event channel and provides backpressure handling
func (dep *DefaultEventProcessor) eventDistributor() {
	defer dep.wg.Done()
	
	dep.logger.Debug("Event distributor started")
	
	// This goroutine could implement more sophisticated distribution logic
	// For now, it just monitors and reports on queue depth
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-dep.ctx.Done():
			dep.logger.Debug("Event distributor stopping")
			return
			
		case <-ticker.C:
			// Monitor queue depth for backpressure detection
			queueDepth := len(dep.eventCh)
			dep.metrics.QueueDepth = queueDepth
			
			if queueDepth > (dep.bufferSize * 8 / 10) { // 80% threshold
				dep.logger.Warn("Event queue depth high", 
					"queueDepth", queueDepth,
					"threshold", dep.bufferSize*8/10)
			}
		}
	}
}

// metricsReporter periodically reports processing metrics
func (dep *DefaultEventProcessor) metricsReporter() {
	defer dep.wg.Done()
	
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	dep.logger.Debug("Metrics reporter started")
	
	for {
		select {
		case <-dep.ctx.Done():
			dep.logger.Debug("Metrics reporter stopping")
			return
			
		case <-ticker.C:
			dep.reportMetrics()
		}
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-010
 * CONTEXT:   Helper methods for event processor internal operations
 * REASON:    Need utility functions for handler management and metrics tracking
 * CHANGE:    Initial implementation with priority sorting and metrics.
 * PREVENTION:Keep helper methods simple and focused on single responsibilities
 * RISK:      Low - Helper methods are internal implementation details
 */

// sortHandlersByPriority sorts handlers by priority (higher priority first)
func (dep *DefaultEventProcessor) sortHandlersByPriority() {
	sort.Slice(dep.handlers, func(i, j int) bool {
		return dep.handlers[i].Priority() > dep.handlers[j].Priority()
	})
}

// updateLatencyMetrics updates average latency calculation
func (dep *DefaultEventProcessor) updateLatencyMetrics(latency time.Duration) {
	// Simple moving average (could be improved with more sophisticated metrics)
	if dep.metrics.AverageLatency == 0 {
		dep.metrics.AverageLatency = latency
	} else {
		dep.metrics.AverageLatency = (dep.metrics.AverageLatency + latency) / 2
	}
}

// reportMetrics logs current processing metrics
func (dep *DefaultEventProcessor) reportMetrics() {
	stats := dep.GetStats()
	
	dep.logger.Info("Event processing metrics",
		"eventsProcessed", stats.EventsProcessed,
		"eventsDropped", stats.EventsDropped,
		"processingErrors", stats.ProcessingErrors,
		"handlersRegistered", stats.HandlersRegistered,
		"queueDepth", dep.metrics.QueueDepth,
		"averageLatency", dep.metrics.AverageLatency)
}

// IsRunning returns true if the processor is currently running
func (dep *DefaultEventProcessor) IsRunning() bool {
	return atomic.LoadInt32(&dep.running) == 1
}

// GetHandlerCount returns the number of registered handlers
func (dep *DefaultEventProcessor) GetHandlerCount() int {
	dep.mu.RLock()
	defer dep.mu.RUnlock()
	return len(dep.handlers)
}