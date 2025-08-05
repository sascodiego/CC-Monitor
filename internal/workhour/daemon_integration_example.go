/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-INTEGRATION-001
 * CONTEXT:   Example integration showing how enhanced daemon connects with work hour analytics system
 * REASON:    Need clear example of how to integrate work hour system with existing enhanced activity monitoring
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep integration simple and ensure it doesn't disrupt existing daemon functionality
 * RISK:      Low - Example code for demonstration and integration guidance
 */
package workhour

import (
	"fmt"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/daemon"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-INTEGRATION-002
 * CONTEXT:   DaemonWorkHourIntegration shows how to integrate work hour system with enhanced daemon
 * REASON:    Business requirement for seamless integration that doesn't affect existing monitoring
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure integration is optional and fails gracefully if work hour system unavailable
 * RISK:      Medium - Integration must not affect core daemon functionality
 */
type DaemonWorkHourIntegration struct {
	enhancedDaemon  *daemon.EnhancedDaemon
	workHourSystem  *WorkHourComponents
	integrationHandlers EventIntegrationHandlers
	logger          arch.Logger
	isIntegrated    bool
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-INTEGRATION-003
 * CONTEXT:   Example of how enhanced daemon would initialize work hour integration
 * REASON:    Need clear pattern for optional work hour integration that doesn't break existing functionality
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle integration failures gracefully and log appropriate messages
 * RISK:      Low - Example code demonstrating integration patterns
 */
func ExampleDaemonWithWorkHourIntegration(logger arch.Logger, dbPath string) error {
	logger.Info("Starting enhanced daemon with work hour analytics integration")
	
	// Initialize enhanced daemon (existing functionality)
	enhancedDaemon := daemon.NewEnhancedDaemon(logger)
	
	// Initialize work hour system (new functionality)
	workHourFactory := NewWorkHourFactory(logger)
	workHourSystem, err := workHourFactory.CreateWorkHourSystem(dbPath)
	if err != nil {
		logger.Error("Failed to initialize work hour system", "error", err)
		logger.Warn("Continuing without work hour analytics")
		
		// Start daemon without work hour integration
		return enhancedDaemon.Start()
	}
	
	// Create integration wrapper
	integration := &DaemonWorkHourIntegration{
		enhancedDaemon: enhancedDaemon,
		workHourSystem: workHourSystem,
		logger:         logger,
	}
	
	// Set up integration
	if err := integration.SetupIntegration(); err != nil {
		logger.Error("Failed to setup work hour integration", "error", err)
		logger.Warn("Starting daemon without work hour analytics")
		return enhancedDaemon.Start()
	}
	
	// Start integrated system
	return integration.Start()
}

func (dwhi *DaemonWorkHourIntegration) SetupIntegration() error {
	dwhi.logger.Info("Setting up work hour integration with enhanced daemon")
	
	// Get integration handlers from work hour system
	dwhi.integrationHandlers = dwhi.workHourSystem.GetEventIntegrationHandlers()
	
	// Connect work hour system with daemon
	if err := dwhi.workHourSystem.IntegrateWithEnhancedDaemon(dwhi.enhancedDaemon); err != nil {
		return fmt.Errorf("failed to integrate systems: %w", err)
	}
	
	dwhi.isIntegrated = true
	dwhi.logger.Info("Work hour integration setup completed")
	return nil
}

func (dwhi *DaemonWorkHourIntegration) Start() error {
	dwhi.logger.Info("Starting integrated daemon with work hour analytics")
	
	// Start work hour system first
	if err := dwhi.workHourSystem.Start(); err != nil {
		return fmt.Errorf("failed to start work hour system: %w", err)
	}
	
	// Start enhanced daemon with integration hooks
	if err := dwhi.startEnhancedDaemonWithHooks(); err != nil {
		// Try to stop work hour system if daemon fails
		dwhi.workHourSystem.Stop()
		return fmt.Errorf("failed to start enhanced daemon: %w", err)
	}
	
	dwhi.logger.Info("Integrated system started successfully")
	return nil
}

func (dwhi *DaemonWorkHourIntegration) Stop() error {
	dwhi.logger.Info("Stopping integrated daemon system")
	
	var stopErrors []error
	
	// Stop enhanced daemon first
	if err := dwhi.enhancedDaemon.Stop(); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("enhanced daemon stop error: %w", err))
	}
	
	// Stop work hour system
	if err := dwhi.workHourSystem.Stop(); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("work hour system stop error: %w", err))
	}
	
	if len(stopErrors) > 0 {
		return fmt.Errorf("stop errors: %v", stopErrors)
	}
	
	dwhi.logger.Info("Integrated system stopped successfully")
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-INTEGRATION-004
 * CONTEXT:   Enhanced daemon startup with work hour event hooks
 * REASON:    Need to show how existing daemon code would be modified to emit work hour events
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure event emission doesn't impact daemon performance or reliability
 * RISK:      Medium - Event hooks must not interfere with core daemon operations
 */
func (dwhi *DaemonWorkHourIntegration) startEnhancedDaemonWithHooks() error {
	// This demonstrates how the enhanced daemon would be modified to emit events
	// In practice, these hooks would be added to the actual enhanced daemon code
	
	dwhi.logger.Info("Starting enhanced daemon with work hour event hooks")
	
	// Start the enhanced daemon normally
	if err := dwhi.enhancedDaemon.Start(); err != nil {
		return err
	}
	
	// If integration is available, set up event monitoring
	if dwhi.isIntegrated {
		go dwhi.monitorDaemonEvents()
	}
	
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-INTEGRATION-005
 * CONTEXT:   Example event monitoring showing how daemon events trigger work hour updates
 * REASON:    Need to demonstrate the event flow from daemon monitoring to work hour analytics
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle monitoring errors gracefully and don't impact daemon performance
 * RISK:      Low - Example monitoring code for demonstration
 */
func (dwhi *DaemonWorkHourIntegration) monitorDaemonEvents() {
	dwhi.logger.Info("Starting daemon event monitoring for work hour integration")
	
	// This is example code showing how you would monitor daemon state
	// In practice, the enhanced daemon would emit events directly
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	var lastSession *domain.Session
	var lastWorkBlock *domain.WorkBlock
	
	for {
		select {
		case <-ticker.C:
			// Example: Check if daemon has a current session
			if currentSession := dwhi.getCurrentSessionFromDaemon(); currentSession != nil {
				if lastSession == nil || lastSession.ID != currentSession.ID {
					// New session started
					dwhi.integrationHandlers.OnSessionStarted(currentSession)
					lastSession = currentSession
					dwhi.logger.Debug("Work hour event: session started", "sessionID", currentSession.ID)
				}
			} else if lastSession != nil {
				// Session ended
				dwhi.integrationHandlers.OnSessionEnded(lastSession)
				dwhi.logger.Debug("Work hour event: session ended", "sessionID", lastSession.ID)
				lastSession = nil
			}
			
			// Example: Check if daemon has a current work block
			if currentWorkBlock := dwhi.getCurrentWorkBlockFromDaemon(); currentWorkBlock != nil {
				if lastWorkBlock == nil || lastWorkBlock.ID != currentWorkBlock.ID {
					// New work block started
					dwhi.integrationHandlers.OnWorkBlockStarted(currentWorkBlock)
					lastWorkBlock = currentWorkBlock
					dwhi.logger.Debug("Work hour event: work block started", "blockID", currentWorkBlock.ID)
				}
			} else if lastWorkBlock != nil {
				// Work block finalized
				dwhi.integrationHandlers.OnWorkBlockFinalized(lastWorkBlock)
				dwhi.logger.Debug("Work hour event: work block finalized", "blockID", lastWorkBlock.ID)
				lastWorkBlock = nil
			}
		}
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-INTEGRATION-006
 * CONTEXT:   Helper methods showing how to extract daemon state for work hour events
 * REASON:    Need to show how existing daemon state maps to work hour event data
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle cases where daemon state is unavailable or incomplete
 * RISK:      Low - Example helper methods for state extraction
 */
func (dwhi *DaemonWorkHourIntegration) getCurrentSessionFromDaemon() *domain.Session {
	// This is example code - in practice you would access the enhanced daemon's current session
	// The enhanced daemon would need to expose its current session state
	
	// Example: If daemon exposes GetCurrentSession method
	// return dwhi.enhancedDaemon.GetCurrentSession()
	
	// For demonstration, return nil (no current session)
	return nil
}

func (dwhi *DaemonWorkHourIntegration) getCurrentWorkBlockFromDaemon() *domain.WorkBlock {
	// This is example code - in practice you would access the enhanced daemon's current work block
	// The enhanced daemon would need to expose its current work block state
	
	// Example: If daemon exposes GetCurrentWorkBlock method
	// return dwhi.enhancedDaemon.GetCurrentWorkBlock()
	
	// For demonstration, return nil (no current work block)
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-INTEGRATION-007
 * CONTEXT:   Example of how enhanced daemon would be modified to emit work hour events directly
 * REASON:    Show the ideal integration where daemon emits events as they occur
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure event emission is lightweight and doesn't block daemon operations
 * RISK:      Low - Example code showing ideal integration pattern
 */

// ExampleEnhancedDaemonModifications shows how the enhanced daemon would be modified
// to emit work hour events directly (pseudo-code)
func ExampleEnhancedDaemonModifications() {
	/*
	// Example modifications to enhanced daemon:
	
	type EnhancedDaemon struct {
		// ... existing fields ...
		workHourEventHandlers *EventIntegrationHandlers  // Add this field
	}
	
	// Add method to set work hour integration
	func (ed *EnhancedDaemon) SetWorkHourIntegration(handlers EventIntegrationHandlers) {
		ed.workHourEventHandlers = &handlers
	}
	
	// Modify startNewSession to emit event
	func (ed *EnhancedDaemon) startNewSession() {
		// ... existing session creation code ...
		
		// Emit work hour event if integration is available
		if ed.workHourEventHandlers != nil {
			ed.workHourEventHandlers.OnSessionStarted(ed.currentSession)
		}
	}
	
	// Modify finalizeCurrentWorkBlock to emit event
	func (ed *EnhancedDaemon) finalizeCurrentWorkBlock() {
		if ed.currentWorkBlock != nil {
			// ... existing finalization code ...
			
			// Emit work hour event if integration is available
			if ed.workHourEventHandlers != nil {
				ed.workHourEventHandlers.OnWorkBlockFinalized(ed.currentWorkBlock)
			}
		}
	}
	
	// Add method to emit user interaction events
	func (ed *EnhancedDaemon) emitUserInteractionEvent(timestamp time.Time, activityType string) {
		if ed.workHourEventHandlers != nil {
			metadata := map[string]interface{}{
				"processCount": ed.getClaudeProcessCount(),
				"connections": len(ed.connectionTracker),
			}
			ed.workHourEventHandlers.OnUserInteraction(timestamp, activityType, metadata)
		}
	}
	*/
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-INTEGRATION-008
 * CONTEXT:   Integration health monitoring and status reporting
 * REASON:    Need to monitor integration health and provide status information
 * CHANGE:    Initial implementation.
 * PREVENTION:Monitor integration health regularly and alert on failures
 * RISK:      Low - Monitoring code for operational support
 */
func (dwhi *DaemonWorkHourIntegration) GetIntegrationStatus() map[string]interface{} {
	status := map[string]interface{}{
		"integrated":     dwhi.isIntegrated,
		"daemonRunning":  true, // Would check actual daemon status
	}
	
	if dwhi.isIntegrated {
		workHourStatus := dwhi.workHourSystem.GetSystemStatus()
		status["workHourSystem"] = workHourStatus
	}
	
	return status
}

// Example of how to use the integrated system
func ExampleUsage() {
	/*
	// Initialize integrated daemon
	logger := // ... get logger
	dbPath := "/path/to/claude-monitor.db"
	
	if err := ExampleDaemonWithWorkHourIntegration(logger, dbPath); err != nil {
		logger.Error("Failed to start integrated daemon", "error", err)
		return
	}
	
	// Access work hour functionality
	// (this would be available through the integration)
	workHourFactory := NewWorkHourFactory(logger)
	workHourSystem, _ := workHourFactory.CreateWorkHourSystem(dbPath)
	
	// Get today's work summary
	todaySummary, _ := workHourSystem.GetTodaysSummary()
	fmt.Printf("Today's work time: %v\n", todaySummary.TotalTime)
	
	// Generate weekly report
	report, _ := workHourSystem.GenerateQuickReport("weekly", 
		time.Now().AddDate(0, 0, -7), time.Now())
	fmt.Printf("Weekly report generated: %s\n", report.Title)
	
	// Create timesheet
	timesheet, _ := workHourSystem.CreateQuickTimesheet("default", 
		time.Now().AddDate(0, 0, -7))
	fmt.Printf("Timesheet created: %s\n", timesheet.ID)
	*/
}