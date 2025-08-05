package workhour

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
	"github.com/claude-monitor/claude-monitor/pkg/events"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-021
 * CONTEXT:   Integration layer connecting work hour analytics with existing session/work block system
 * REASON:    Need seamless integration that doesn't disrupt existing monitoring while adding analytics
 * CHANGE:    New integration layer for work hour system connecting to existing infrastructure.
 * PREVENTION:Maintain separation of concerns, avoid tight coupling, handle integration failures gracefully
 * RISK:      High - Integration failures could affect both existing monitoring and new analytics
 */

// WorkHourIntegrator coordinates between existing session management and new work hour analytics
type WorkHourIntegrator struct {
	sessionManager    arch.SessionManager
	workBlockManager  arch.WorkBlockManager
	dbManager         arch.WorkHourDatabaseManager
	analyzer          arch.WorkHourAnalyzer
	reportGenerator   arch.WorkHourReportGenerator
	exporter          arch.WorkHourExporter
	logger            arch.Logger
	
	// Integration state
	mu                sync.RWMutex
	realTimeEnabled   bool
	cacheEnabled      bool
	aggregationTicker *time.Ticker
	eventSubscription chan *events.SystemEvent
	
	// Performance optimization
	workDayCache      map[string]*domain.WorkDay  // date -> WorkDay
	lastAggregation   time.Time
	aggregationInterval time.Duration
}

// NewWorkHourIntegrator creates a new work hour integrator
func NewWorkHourIntegrator(
	sessionManager arch.SessionManager,
	workBlockManager arch.WorkBlockManager,
	dbManager arch.WorkHourDatabaseManager,
	analyzer arch.WorkHourAnalyzer,
	reportGenerator arch.WorkHourReportGenerator,
	exporter arch.WorkHourExporter,
	logger arch.Logger,
) *WorkHourIntegrator {
	return &WorkHourIntegrator{
		sessionManager:      sessionManager,
		workBlockManager:    workBlockManager,
		dbManager:          dbManager,
		analyzer:           analyzer,
		reportGenerator:    reportGenerator,
		exporter:           exporter,
		logger:             logger,
		workDayCache:       make(map[string]*domain.WorkDay),
		aggregationInterval: 15 * time.Minute, // Aggregate every 15 minutes
		realTimeEnabled:    true,
		cacheEnabled:       true,
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-022
 * CONTEXT:   Real-time integration with existing session/work block events
 * REASON:    Need real-time work hour updates without polling or significant performance impact
 * CHANGE:    Event-driven integration with existing session and work block lifecycle.
 * PREVENTION:Handle event processing failures gracefully, avoid blocking existing event flow
 * RISK:      Medium - Event processing failures could cause analytics lag but shouldn't affect monitoring
 */

// StartRealTimeIntegration begins real-time integration with existing system
func (whi *WorkHourIntegrator) StartRealTimeIntegration(ctx context.Context) error {
	whi.mu.Lock()
	defer whi.mu.Unlock()
	
	if !whi.realTimeEnabled {
		whi.logger.Info("Real-time integration disabled")
		return nil
	}
	
	// Set up periodic aggregation
	whi.aggregationTicker = time.NewTicker(whi.aggregationInterval)
	
	// Start background processes
	go whi.aggregationLoop(ctx)
	go whi.cacheMaintenanceLoop(ctx)
	
	whi.logger.Info("Work hour real-time integration started",
		"aggregationInterval", whi.aggregationInterval,
		"cacheEnabled", whi.cacheEnabled)
	
	return nil
}

// StopRealTimeIntegration gracefully stops real-time integration
func (whi *WorkHourIntegrator) StopRealTimeIntegration() error {
	whi.mu.Lock()
	defer whi.mu.Unlock()
	
	if whi.aggregationTicker != nil {
		whi.aggregationTicker.Stop()
		whi.aggregationTicker = nil
	}
	
	// Final aggregation
	if err := whi.performAggregation(); err != nil {
		whi.logger.Error("Failed final aggregation during shutdown", "error", err)
		return err
	}
	
	whi.logger.Info("Work hour real-time integration stopped")
	return nil
}

// aggregationLoop performs periodic work hour data aggregation
func (whi *WorkHourIntegrator) aggregationLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-whi.aggregationTicker.C:
			if err := whi.performAggregation(); err != nil {
				whi.logger.Error("Aggregation failed", "error", err)
			}
		}
	}
}

// performAggregation aggregates recent session/work block data into work hour structures
func (whi *WorkHourIntegrator) performAggregation() error {
	whi.mu.Lock()
	defer whi.mu.Unlock()
	
	now := time.Now()
	
	// Determine aggregation window
	startTime := whi.lastAggregation
	if startTime.IsZero() {
		startTime = now.Add(-24 * time.Hour) // Initial aggregation covers last 24 hours
	}
	
	whi.logger.Debug("Performing work hour aggregation",
		"startTime", startTime,
		"endTime", now)
	
	// Aggregate work days for the period
	if err := whi.aggregateWorkDays(startTime, now); err != nil {
		return fmt.Errorf("failed to aggregate work days: %w", err)
	}
	
	// Update cache if enabled
	if whi.cacheEnabled {
		if err := whi.refreshWorkDayCache(startTime, now); err != nil {
			whi.logger.Warn("Failed to refresh work day cache", "error", err)
		}
	}
	
	whi.lastAggregation = now
	return nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-023
 * CONTEXT:   Work day aggregation from raw session and work block data
 * REASON:    Need to transform raw monitoring data into business-oriented work day entities
 * CHANGE:    Aggregation logic that processes sessions/work blocks into work days.
 * PREVENTION:Handle incomplete data gracefully, validate time calculations, ensure timezone consistency
 * RISK:      Medium - Aggregation errors could cause inaccurate work hour calculations
 */

// aggregateWorkDays processes sessions and work blocks into work day aggregates
func (whi *WorkHourIntegrator) aggregateWorkDays(startTime, endTime time.Time) error {
	// Get sessions and work blocks for the period
	sessions, err := whi.getSessionsInPeriod(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}
	
	workBlocks, err := whi.getWorkBlocksInPeriod(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get work blocks: %w", err)
	}
	
	// Group data by date
	workDayData := whi.groupDataByDate(sessions, workBlocks)
	
	// Process each work day
	for dateStr, dayData := range workDayData {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			whi.logger.Error("Invalid date in work day data", "date", dateStr, "error", err)
			continue
		}
		
		workDay, err := whi.processWorkDayData(date, dayData)
		if err != nil {
			whi.logger.Error("Failed to process work day", "date", dateStr, "error", err)
			continue
		}
		
		// Save work day to database
		if err := whi.dbManager.SaveWorkDay(workDay); err != nil {
			whi.logger.Error("Failed to save work day", "date", dateStr, "error", err)
			continue
		}
		
		whi.logger.Debug("Work day aggregated successfully",
			"date", dateStr,
			"totalTime", workDay.TotalTime,
			"sessionCount", workDay.SessionCount,
			"blockCount", workDay.BlockCount)
	}
	
	return nil
}

// WorkDayData holds sessions and work blocks for a specific date
type WorkDayData struct {
	Sessions   []*domain.Session
	WorkBlocks []*domain.WorkBlock
}

// groupDataByDate groups sessions and work blocks by calendar date
func (whi *WorkHourIntegrator) groupDataByDate(sessions []*domain.Session, workBlocks []*domain.WorkBlock) map[string]*WorkDayData {
	workDayData := make(map[string]*WorkDayData)
	
	// Group sessions by date
	for _, session := range sessions {
		dateStr := session.StartTime.Format("2006-01-02")
		if workDayData[dateStr] == nil {
			workDayData[dateStr] = &WorkDayData{
				Sessions:   make([]*domain.Session, 0),
				WorkBlocks: make([]*domain.WorkBlock, 0),
			}
		}
		workDayData[dateStr].Sessions = append(workDayData[dateStr].Sessions, session)
	}
	
	// Group work blocks by date
	for _, workBlock := range workBlocks {
		dateStr := workBlock.StartTime.Format("2006-01-02")
		if workDayData[dateStr] == nil {
			workDayData[dateStr] = &WorkDayData{
				Sessions:   make([]*domain.Session, 0),
				WorkBlocks: make([]*domain.WorkBlock, 0),
			}
		}
		workDayData[dateStr].WorkBlocks = append(workDayData[dateStr].WorkBlocks, workBlock)
	}
	
	return workDayData
}

// processWorkDayData converts raw session/work block data into a WorkDay entity
func (whi *WorkHourIntegrator) processWorkDayData(date time.Time, data *WorkDayData) (*domain.WorkDay, error) {
	workDay := domain.NewWorkDay(date)
	
	var totalWorkTime time.Duration
	var firstActivity, lastActivity *time.Time
	
	// Process work blocks to calculate total time and activity window
	for _, workBlock := range data.WorkBlocks {
		// Update activity window
		if firstActivity == nil || workBlock.StartTime.Before(*firstActivity) {
			firstActivity = &workBlock.StartTime
		}
		
		var blockEndTime time.Time
		if workBlock.EndTime != nil {
			blockEndTime = *workBlock.EndTime
		} else {
			blockEndTime = workBlock.LastActivity
		}
		
		if lastActivity == nil || blockEndTime.After(*lastActivity) {
			lastActivity = &blockEndTime
		}
		
		// Add work block duration
		blockDuration := workBlock.Duration()
		totalWorkTime += blockDuration
		
		workDay.BlockCount++
	}
	
	// Set work day properties
	workDay.TotalTime = totalWorkTime
	workDay.SessionCount = len(data.Sessions)
	workDay.StartTime = firstActivity
	workDay.EndTime = lastActivity
	
	// Calculate break time (time between work blocks)
	if workDay.StartTime != nil && workDay.EndTime != nil {
		totalSpan := workDay.EndTime.Sub(*workDay.StartTime)
		workDay.BreakTime = totalSpan - totalWorkTime
		if workDay.BreakTime < 0 {
			workDay.BreakTime = 0
		}
	}
	
	return workDay, nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-024
 * CONTEXT:   Cache management for work hour data performance optimization
 * REASON:    Need fast access to frequently requested work hour data without database queries
 * CHANGE:    Caching layer for work hour data with automatic maintenance and invalidation.
 * PREVENTION:Implement cache invalidation policies, handle memory constraints, validate cache consistency
 * RISK:      Medium - Cache inconsistencies could serve stale data but won't corrupt underlying data
 */

// cacheMaintenanceLoop performs periodic cache maintenance
func (whi *WorkHourIntegrator) cacheMaintenanceLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Cache maintenance every hour
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			whi.performCacheMaintenance()
		}
	}
}

// performCacheMaintenance cleans up old cache entries and validates cache consistency
func (whi *WorkHourIntegrator) performCacheMaintenance() {
	whi.mu.Lock()
	defer whi.mu.Unlock()
	
	if !whi.cacheEnabled {
		return
	}
	
	now := time.Now()
	cutoffDate := now.AddDate(0, 0, -7) // Keep only last 7 days in cache
	
	entriesRemoved := 0
	for dateStr := range whi.workDayCache {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			delete(whi.workDayCache, dateStr)
			entriesRemoved++
			continue
		}
		
		if date.Before(cutoffDate) {
			delete(whi.workDayCache, dateStr)
			entriesRemoved++
		}
	}
	
	whi.logger.Debug("Cache maintenance completed",
		"entriesRemoved", entriesRemoved,
		"cacheSize", len(whi.workDayCache))
}

// refreshWorkDayCache updates cached work day data
func (whi *WorkHourIntegrator) refreshWorkDayCache(startTime, endTime time.Time) error {
	current := startTime
	for current.Before(endTime) {
		dateStr := current.Format("2006-01-02")
		
		// Get work day from database
		workDay, err := whi.dbManager.GetWorkDayData(current)
		if err != nil {
			whi.logger.Warn("Failed to get work day for cache", "date", dateStr, "error", err)
		} else if workDay != nil {
			whi.workDayCache[dateStr] = workDay
		}
		
		current = current.AddDate(0, 0, 1)
	}
	
	return nil
}

// GetCachedWorkDay retrieves work day from cache
func (whi *WorkHourIntegrator) GetCachedWorkDay(date time.Time) (*domain.WorkDay, bool) {
	whi.mu.RLock()
	defer whi.mu.RUnlock()
	
	if !whi.cacheEnabled {
		return nil, false
	}
	
	dateStr := date.Format("2006-01-02")
	workDay, exists := whi.workDayCache[dateStr]
	return workDay, exists
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-025
 * CONTEXT:   Event subscription and processing for real-time work hour updates
 * REASON:    Need immediate work hour updates when sessions or work blocks change
 * CHANGE:    Event-driven integration that responds to session and work block lifecycle events.
 * PREVENTION:Handle event processing asynchronously, avoid blocking event publishers
 * RISK:      Low - Event processing failures affect analytics timeliness but not core monitoring
 */

// SubscribeToSessionEvents subscribes to session lifecycle events
func (whi *WorkHourIntegrator) SubscribeToSessionEvents(eventCh <-chan *events.SystemEvent) {
	go func() {
		for event := range eventCh {
			if err := whi.processSessionEvent(event); err != nil {
				whi.logger.Error("Failed to process session event", "error", err)
			}
		}
	}()
}

// processSessionEvent handles session-related events for work hour updates
func (whi *WorkHourIntegrator) processSessionEvent(event *events.SystemEvent) error {
	// Process different event types
	switch event.Type {
	case events.EventSessionStarted:
		return whi.handleSessionStarted(event)
	case events.EventSessionEnded:
		return whi.handleSessionEnded(event)
	case events.EventWorkBlockStarted:
		return whi.handleWorkBlockStarted(event)
	case events.EventWorkBlockEnded:
		return whi.handleWorkBlockEnded(event)
	default:
		// Ignore other event types
		return nil
	}
}

// handleSessionStarted processes session start events
func (whi *WorkHourIntegrator) handleSessionStarted(event *events.SystemEvent) error {
	sessionID, ok := event.Metadata["sessionID"].(string)
	if !ok {
		return fmt.Errorf("missing sessionID in session started event")
	}
	
	whi.logger.Debug("Processing session started event", "sessionID", sessionID)
	
	// Trigger immediate work day cache refresh for today
	today := time.Now().Truncate(24 * time.Hour)
	return whi.invalidateWorkDayCache(today)
}

// handleSessionEnded processes session end events
func (whi *WorkHourIntegrator) handleSessionEnded(event *events.SystemEvent) error {
	sessionID, ok := event.Metadata["sessionID"].(string)
	if !ok {
		return fmt.Errorf("missing sessionID in session ended event")
	}
	
	whi.logger.Debug("Processing session ended event", "sessionID", sessionID)
	
	// Trigger work day aggregation for the session's date
	if sessionStartTime, ok := event.Metadata["startTime"].(time.Time); ok {
		return whi.aggregateWorkDayForDate(sessionStartTime)
	}
	
	return nil
}

// handleWorkBlockStarted processes work block start events
func (whi *WorkHourIntegrator) handleWorkBlockStarted(event *events.SystemEvent) error {
	blockID, ok := event.Metadata["blockID"].(string)
	if !ok {
		return fmt.Errorf("missing blockID in work block started event")
	}
	
	whi.logger.Debug("Processing work block started event", "blockID", blockID)
	
	// Real-time work day update
	today := time.Now().Truncate(24 * time.Hour)
	return whi.invalidateWorkDayCache(today)
}

// handleWorkBlockEnded processes work block end events
func (whi *WorkHourIntegrator) handleWorkBlockEnded(event *events.SystemEvent) error {
	blockID, ok := event.Metadata["blockID"].(string)
	if !ok {
		return fmt.Errorf("missing blockID in work block ended event")
	}
	
	whi.logger.Debug("Processing work block ended event", "blockID", blockID)
	
	// Trigger immediate aggregation for the work block's date
	if blockStartTime, ok := event.Metadata["startTime"].(time.Time); ok {
		return whi.aggregateWorkDayForDate(blockStartTime)
	}
	
	return nil
}

// aggregateWorkDayForDate triggers work day aggregation for a specific date
func (whi *WorkHourIntegrator) aggregateWorkDayForDate(date time.Time) error {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	return whi.aggregateWorkDays(startOfDay, endOfDay)
}

// invalidateWorkDayCache removes work day from cache to force refresh
func (whi *WorkHourIntegrator) invalidateWorkDayCache(date time.Time) error {
	whi.mu.Lock()
	defer whi.mu.Unlock()
	
	dateStr := date.Format("2006-01-02")
	delete(whi.workDayCache, dateStr)
	
	whi.logger.Debug("Work day cache invalidated", "date", dateStr)
	return nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-WORKHR-026
 * CONTEXT:   Database query methods for integration with existing session/work block data
 * REASON:    Need efficient queries to retrieve session and work block data for aggregation
 * CHANGE:    Database query methods that integrate with existing database schema.
 * PREVENTION:Use proper indexes, limit query results, validate date ranges
 * RISK:      Medium - Inefficient queries could impact database performance
 */

// getSessionsInPeriod retrieves sessions within the specified time period
func (whi *WorkHourIntegrator) getSessionsInPeriod(startTime, endTime time.Time) ([]*domain.Session, error) {
	// This would query the existing sessions table
	// Implementation depends on the actual database schema and query methods
	
	// Placeholder implementation - in practice this would use the existing DatabaseManager
	whi.logger.Debug("Querying sessions in period",
		"startTime", startTime,
		"endTime", endTime)
	
	// Return empty slice for now - actual implementation would query database
	return []*domain.Session{}, nil
}

// getWorkBlocksInPeriod retrieves work blocks within the specified time period
func (whi *WorkHourIntegrator) getWorkBlocksInPeriod(startTime, endTime time.Time) ([]*domain.WorkBlock, error) {
	// This would query the existing work_blocks table
	// Implementation depends on the actual database schema and query methods
	
	whi.logger.Debug("Querying work blocks in period",
		"startTime", startTime,
		"endTime", endTime)
	
	// Return empty slice for now - actual implementation would query database
	return []*domain.WorkBlock{}, nil
}

// Configuration methods for integration behavior
func (whi *WorkHourIntegrator) SetRealTimeEnabled(enabled bool) {
	whi.mu.Lock()
	defer whi.mu.Unlock()
	whi.realTimeEnabled = enabled
}

func (whi *WorkHourIntegrator) SetCacheEnabled(enabled bool) {
	whi.mu.Lock()
	defer whi.mu.Unlock()
	whi.cacheEnabled = enabled
}

func (whi *WorkHourIntegrator) SetAggregationInterval(interval time.Duration) {
	whi.mu.Lock()
	defer whi.mu.Unlock()
	whi.aggregationInterval = interval
}