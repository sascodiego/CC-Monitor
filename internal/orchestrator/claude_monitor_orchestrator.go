/**
 * CONTEXT:   Claude Monitor orchestrator for complete work hour tracking system integration
 * INPUT:     System configuration and component coordination requirements
 * OUTPUT:    Fully integrated Claude work hour tracking with activity generation
 * BUSINESS:  Orchestrator provides complete work hour tracking like the "hook" system
 * CHANGE:    Initial orchestrator for activity-driven work hour tracking system
 * RISK:      High - Core system integration affecting all monitoring and tracking
 */

package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/your-org/claude-monitor/internal/generator"
	"github.com/your-org/claude-monitor/internal/monitor"
	"github.com/your-org/claude-monitor/internal/reporting"
)

/**
 * CONTEXT:   Claude Monitor orchestrator configuration
 * INPUT:     Complete system configuration parameters
 * OUTPUT:    System behavior configuration
 * BUSINESS:  Configuration enables customizable Claude monitoring
 * CHANGE:    Initial orchestrator configuration
 * RISK:      Medium - Configuration affecting system behavior
 */
type OrchestratorConfig struct {
	// Monitor configurations
	ProcessMonitor monitor.ProcessMonitorConfig `json:"process_monitor"`
	FileIOMonitor  monitor.FileIOMonitorConfig  `json:"file_io_monitor"`
	HTTPMonitor    monitor.HTTPMonitorConfig    `json:"http_monitor"`
	NetworkMonitor monitor.NetworkMonitorConfig `json:"network_monitor"`
	
	// Activity generation
	ActivityGeneration ActivityGenerationConfig `json:"activity_generation"`
	
	// Reporting
	ReportingConfig ReportingConfig `json:"reporting"`
	
	// System settings
	VerboseLogging    bool          `json:"verbose_logging"`
	RealtimeReporting bool          `json:"realtime_reporting"`
	OutputFile        string        `json:"output_file"`
	StatisticsInterval time.Duration `json:"statistics_interval"`
}

/**
 * CONTEXT:   Activity generation configuration
 * INPUT:     Activity generator behavior parameters
 * OUTPUT:    Activity generation configuration
 * BUSINESS:  Activity configuration enables precise work tracking
 * CHANGE:    Initial activity generation configuration
 * RISK:      Medium - Activity generation affecting work hour accuracy
 */
type ActivityGenerationConfig struct {
	SessionDuration      time.Duration `json:"session_duration"`       // 5 hours
	IdleTimeout         time.Duration `json:"idle_timeout"`           // 5 minutes
	MinActivityDuration time.Duration `json:"min_activity_duration"`  // 1 second
	WorkFilePatterns    []string      `json:"work_file_patterns"`     // File patterns for work detection
	ProjectDetection    bool          `json:"project_detection"`      // Enable project detection
}

/**
 * CONTEXT:   Reporting configuration
 * INPUT:     Report generation parameters
 * OUTPUT:    Reporting system configuration
 * BUSINESS:  Reporting configuration enables customizable work hour reports
 * CHANGE:    Initial reporting configuration
 * RISK:      Low - Reporting configuration for work hour analysis
 */
type ReportingConfig struct {
	EnableRealtimeReports bool          `json:"enable_realtime_reports"`
	ReportInterval       time.Duration `json:"report_interval"`        // How often to generate reports
	ReportFormats        []string      `json:"report_formats"`         // json, text, html
	HistoryRetention     time.Duration `json:"history_retention"`      // How long to keep history
	DailyReports         bool          `json:"daily_reports"`          // Generate daily reports
	WeeklyReports        bool          `json:"weekly_reports"`         // Generate weekly reports
}

/**
 * CONTEXT:   Complete Claude Monitor orchestrator system
 * INPUT:     All monitoring components and activity generation
 * OUTPUT:    Integrated work hour tracking system
 * BUSINESS:  Orchestrator provides complete Claude work hour tracking solution
 * CHANGE:    Initial orchestrator for complete system integration
 * RISK:      High - Core system orchestrating all monitoring and activity generation
 */
type ClaudeMonitorOrchestrator struct {
	ctx    context.Context
	cancel context.CancelFunc
	config OrchestratorConfig
	mu     sync.RWMutex
	
	// Core monitoring components
	processMonitor *monitor.ProcessMonitor
	fileIOMonitor  *monitor.FileIOMonitor
	httpMonitor    *monitor.HTTPMonitor
	networkMonitor *monitor.NetworkMonitor
	
	// Activity generation and reporting
	activityGenerator   *generator.ActivityGenerator
	workTimeCalculator  *reporting.WorkTimeCalculator
	comprehensiveStats  *monitor.ComprehensiveStats
	
	// System state
	running          bool
	startTime        time.Time
	lastReportTime   time.Time
	
	// Data storage (in-memory for now)
	activities       []generator.WorkActivity
	sessions         []generator.WorkSession
	workBlocks       []generator.WorkBlock
	
	// Callbacks for external integration
	onActivityGenerated func(generator.WorkActivity)
	onSessionComplete   func(generator.WorkSession)
	onReportGenerated   func(*reporting.WorkTimeReport)
}

/**
 * CONTEXT:   Create new Claude Monitor orchestrator
 * INPUT:     Orchestrator configuration
 * OUTPUT:    Configured complete monitoring system
 * BUSINESS:  Orchestrator creation enables complete Claude work hour tracking
 * CHANGE:    Initial orchestrator constructor with complete system setup
 * RISK:      High - System initialization affecting all monitoring components
 */
func NewClaudeMonitorOrchestrator(config OrchestratorConfig) *ClaudeMonitorOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set default configuration
	if config.ActivityGeneration.SessionDuration == 0 {
		config.ActivityGeneration.SessionDuration = 5 * time.Hour
	}
	if config.ActivityGeneration.IdleTimeout == 0 {
		config.ActivityGeneration.IdleTimeout = 5 * time.Minute
	}
	if config.StatisticsInterval == 0 {
		config.StatisticsInterval = 30 * time.Second
	}
	if config.ReportingConfig.ReportInterval == 0 {
		config.ReportingConfig.ReportInterval = 15 * time.Minute
	}
	
	orchestrator := &ClaudeMonitorOrchestrator{
		ctx:          ctx,
		cancel:       cancel,
		config:       config,
		activities:   make([]generator.WorkActivity, 0),
		sessions:     make([]generator.WorkSession, 0),
		workBlocks:   make([]generator.WorkBlock, 0),
		startTime:    time.Now(),
	}
	
	// Initialize components
	orchestrator.initializeComponents()
	
	return orchestrator
}

/**
 * CONTEXT:   Initialize all monitoring and generation components
 * INPUT:     Component initialization requirements
 * OUTPUT:    Initialized and connected monitoring system
 * BUSINESS:  Component initialization creates complete monitoring pipeline
 * CHANGE:    Initial component initialization and connection
 * RISK:      High - Component initialization affecting system functionality
 */
func (cmo *ClaudeMonitorOrchestrator) initializeComponents() {
	// Initialize activity generator
	cmo.activityGenerator = generator.NewActivityGenerator(
		cmo.handleActivityGenerated,
		cmo.handleSessionComplete,
	)
	
	// Initialize work time calculator
	cmo.workTimeCalculator = reporting.NewWorkTimeCalculator()
	
	// Initialize comprehensive statistics
	statsConfig := monitor.StatsConfig{
		CollectionInterval: cmo.config.StatisticsInterval,
		EnableInsights:     true,
		EnablePerformance:  true,
		VerboseLogging:     cmo.config.VerboseLogging,
	}
	cmo.comprehensiveStats = monitor.NewComprehensiveStats(statsConfig)
	
	// Initialize process monitor
	cmo.processMonitor = monitor.NewProcessMonitor(cmo.handleProcessEvent)
	cmo.processMonitor.SetFileIOCallback(cmo.handleFileIOEvent)
	
	// Initialize HTTP monitor
	cmo.httpMonitor = monitor.NewHTTPMonitor(cmo.handleHTTPEvent, cmo.config.HTTPMonitor)
	
	// Initialize network monitor  
	cmo.networkMonitor = monitor.NewNetworkMonitor(cmo.handleNetworkEvent)
	
	log.Printf("🚀 Claude Monitor Orchestrator initialized with all components")
}

/**
 * CONTEXT:   Start complete Claude monitoring system
 * INPUT:     System start request
 * OUTPUT:    Running integrated monitoring system
 * BUSINESS:  System start enables complete Claude work hour tracking
 * CHANGE:    Initial system start with all components
 * RISK:      High - System startup affecting monitoring reliability
 */
func (cmo *ClaudeMonitorOrchestrator) Start() error {
	cmo.mu.Lock()
	defer cmo.mu.Unlock()
	
	if cmo.running {
		return fmt.Errorf("Claude Monitor orchestrator is already running")
	}
	
	// Start all monitoring components
	if err := cmo.processMonitor.Start(); err != nil {
		return fmt.Errorf("failed to start process monitor: %w", err)
	}
	
	if err := cmo.httpMonitor.Start(); err != nil {
		return fmt.Errorf("failed to start HTTP monitor: %w", err)
	}
	
	if err := cmo.networkMonitor.Start(); err != nil {
		return fmt.Errorf("failed to start network monitor: %w", err)
	}
	
	// Start system loops
	go cmo.statisticsUpdateLoop()
	go cmo.reportingLoop()
	go cmo.systemHealthLoop()
	
	cmo.running = true
	cmo.startTime = time.Now()
	
	log.Printf("✅ Claude Monitor System Started - Complete work hour tracking active")
	log.Printf("📊 Session Duration: %s, Idle Timeout: %s", 
		cmo.config.ActivityGeneration.SessionDuration,
		cmo.config.ActivityGeneration.IdleTimeout)
	
	return nil
}

/**
 * CONTEXT:   Stop complete Claude monitoring system
 * INPUT:     System stop request
 * OUTPUT:    Cleanly stopped monitoring system
 * BUSINESS:  System stop enables graceful shutdown with final reports
 * CHANGE:    Initial system stop with cleanup and final reporting
 * RISK:      Medium - System shutdown affecting data integrity
 */
func (cmo *ClaudeMonitorOrchestrator) Stop() error {
	cmo.mu.Lock()
	defer cmo.mu.Unlock()
	
	if !cmo.running {
		return nil
	}
	
	log.Printf("🛑 Stopping Claude Monitor System...")
	
	// Generate final report
	cmo.generateFinalReport()
	
	// Stop activity generator (closes active sessions and blocks)
	if cmo.activityGenerator != nil {
		cmo.activityGenerator.Stop()
	}
	
	// Stop all monitoring components
	if cmo.processMonitor != nil {
		cmo.processMonitor.Stop()
	}
	if cmo.httpMonitor != nil {
		cmo.httpMonitor.Stop()
	}
	if cmo.networkMonitor != nil {
		cmo.networkMonitor.Stop()
	}
	
	// Cancel context to stop all loops
	cmo.cancel()
	
	cmo.running = false
	
	// Print final statistics
	cmo.printFinalStatistics()
	
	log.Printf("✅ Claude Monitor System stopped cleanly")
	return nil
}

// Event handlers - Convert monitoring events to activities

/**
 * CONTEXT:   Handle process events and convert to activities
 * INPUT:     Process event from process monitor
 * OUTPUT:    Activity generation from process event
 * BUSINESS:  Process event handling enables process-based work tracking
 * CHANGE:    Initial process event to activity conversion
 * RISK:      Medium - Process event handling affecting work tracking
 */
func (cmo *ClaudeMonitorOrchestrator) handleProcessEvent(event monitor.ProcessEvent) {
	if cmo.config.VerboseLogging {
		log.Printf("📋 Process Event: %s %s (PID: %d)", event.Type, event.Command, event.PID)
	}
	
	// Convert process event to activity via activity generator
	cmo.activityGenerator.ProcessProcessEvent(event)
	
	// Update comprehensive statistics
	cmo.comprehensiveStats.UpdatePrototypeEvent(string(event.Type), map[string]interface{}{
		"command":     event.Command,
		"working_dir": event.WorkingDir,
		"pid":         event.PID,
	})
}

/**
 * CONTEXT:   Handle file I/O events and convert to activities
 * INPUT:     File I/O event from file monitor
 * OUTPUT:    Activity generation from file operations
 * BUSINESS:  File I/O handling enables file-based work tracking
 * CHANGE:    Initial file I/O event to activity conversion
 * RISK:      High - File I/O handling affecting work hour accuracy
 */
func (cmo *ClaudeMonitorOrchestrator) handleFileIOEvent(event monitor.FileIOEvent) {
	if cmo.config.VerboseLogging && event.IsWorkFile {
		log.Printf("📁 File I/O Event: %s %s in %s", event.Type, event.FilePath, event.ProjectName)
	}
	
	// Convert file I/O event to activity
	cmo.activityGenerator.ProcessFileIOEvent(event)
	
	// Update comprehensive statistics
	cmo.comprehensiveStats.UpdatePrototypeEvent(string(event.Type), map[string]interface{}{
		"file_path":   event.FilePath,
		"size":        event.Size,
		"is_work_file": event.IsWorkFile,
		"project_name": event.ProjectName,
	})
}

/**
 * CONTEXT:   Handle HTTP events and convert to activities
 * INPUT:     HTTP event from HTTP monitor
 * OUTPUT:    Activity generation from HTTP operations
 * BUSINESS:  HTTP handling enables Claude API work tracking
 * CHANGE:    Initial HTTP event to activity conversion
 * RISK:      High - HTTP handling affecting Claude API work detection
 */
func (cmo *ClaudeMonitorOrchestrator) handleHTTPEvent(event monitor.HTTPEvent) {
	if cmo.config.VerboseLogging && event.IsClaudeAPI {
		log.Printf("🌐 HTTP Event: %s %s %s (Claude API: %t)", event.Method, event.Host, event.URL, event.IsClaudeAPI)
	}
	
	// Convert HTTP event to activity
	cmo.activityGenerator.ProcessHTTPEvent(event)
	
	// Update comprehensive statistics
	cmo.comprehensiveStats.UpdatePrototypeEvent(string(event.Type), map[string]interface{}{
		"method":        event.Method,
		"host":          event.Host,
		"is_claude_api": event.IsClaudeAPI,
		"status_code":   event.StatusCode,
	})
}

/**
 * CONTEXT:   Handle network events and convert to activities
 * INPUT:     Network event from network monitor
 * OUTPUT:    Activity generation from network operations
 * BUSINESS:  Network handling enables connection-based work tracking
 * CHANGE:    Initial network event to activity conversion
 * RISK:      Medium - Network handling affecting connectivity-based work tracking
 */
func (cmo *ClaudeMonitorOrchestrator) handleNetworkEvent(event monitor.NetworkEvent) {
	if cmo.config.VerboseLogging && event.IsClaudeAPI {
		log.Printf("🔗 Network Event: %s %s -> %s (Claude API: %t)", 
			event.Protocol, event.LocalAddr, event.RemoteAddr, event.IsClaudeAPI)
	}
	
	// Convert network event to activity
	cmo.activityGenerator.ProcessNetworkEvent(event)
	
	// Update comprehensive statistics
	cmo.comprehensiveStats.UpdatePrototypeEvent("NET_CONNECT", map[string]interface{}{
		"protocol":      event.Protocol,
		"remote_addr":   event.RemoteAddr,
		"is_claude_api": event.IsClaudeAPI,
	})
}

// Activity and session handlers

/**
 * CONTEXT:   Handle generated work activities
 * INPUT:     Work activity from activity generator
 * OUTPUT:    Stored activity and real-time reporting
 * BUSINESS:  Activity handling enables work hour tracking and real-time reporting
 * CHANGE:    Initial work activity handling with storage and reporting
 * RISK:      High - Activity handling affecting work hour tracking accuracy
 */
func (cmo *ClaudeMonitorOrchestrator) handleActivityGenerated(activity generator.WorkActivity) {
	cmo.mu.Lock()
	cmo.activities = append(cmo.activities, activity)
	cmo.mu.Unlock()
	
	// Update work time calculator
	cmo.workTimeCalculator.UpdateActivities(cmo.activities)
	
	// Real-time activity reporting
	if cmo.config.RealtimeReporting {
		log.Printf("⚡ Work Activity Generated: %s in %s - %s", 
			activity.Type, activity.ProjectName, activity.Description)
	}
	
	// External callback
	if cmo.onActivityGenerated != nil {
		go cmo.onActivityGenerated(activity)
	}
}

/**
 * CONTEXT:   Handle completed work sessions
 * INPUT:     Completed work session from activity generator
 * OUTPUT:    Stored session and session reporting
 * BUSINESS:  Session handling enables 5-hour window tracking and analysis
 * CHANGE:    Initial session handling with storage and reporting
 * RISK:      Medium - Session handling affecting session-based analytics
 */
func (cmo *ClaudeMonitorOrchestrator) handleSessionComplete(session generator.WorkSession) {
	cmo.mu.Lock()
	cmo.sessions = append(cmo.sessions, session)
	cmo.mu.Unlock()
	
	// Update work time calculator
	cmo.workTimeCalculator.UpdateSessions(cmo.sessions)
	
	log.Printf("✅ Work Session Completed: %s - Duration: %s, Active Time: %s", 
		session.ID, 
		session.Duration.Round(time.Minute),
		session.ActiveTime.Round(time.Minute))
	
	// Generate session report
	cmo.generateSessionReport(session)
	
	// External callback
	if cmo.onSessionComplete != nil {
		go cmo.onSessionComplete(session)
	}
}

// System loops

/**
 * CONTEXT:   Statistics update loop for comprehensive analytics
 * INPUT:     Periodic statistics collection
 * OUTPUT:    Updated comprehensive statistics
 * BUSINESS:  Statistics updates enable comprehensive system analytics
 * CHANGE:    Initial statistics update loop
 * RISK:      Low - Statistics collection affecting system insights
 */
func (cmo *ClaudeMonitorOrchestrator) statisticsUpdateLoop() {
	ticker := time.NewTicker(cmo.config.StatisticsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cmo.ctx.Done():
			return
		case <-ticker.C:
			cmo.updateComprehensiveStatistics()
		}
	}
}

/**
 * CONTEXT:   Reporting loop for periodic work hour reports
 * INPUT:     Periodic report generation
 * OUTPUT:    Generated work hour reports
 * BUSINESS:  Reporting loop enables regular work hour analysis
 * CHANGE:    Initial reporting loop for work hour tracking
 * RISK:      Low - Report generation for work hour insights
 */
func (cmo *ClaudeMonitorOrchestrator) reportingLoop() {
	if !cmo.config.ReportingConfig.EnableRealtimeReports {
		return
	}
	
	ticker := time.NewTicker(cmo.config.ReportingConfig.ReportInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-cmo.ctx.Done():
			return
		case <-ticker.C:
			cmo.generatePeriodicReport()
		}
	}
}

/**
 * CONTEXT:   System health monitoring loop
 * INPUT:     System health monitoring
 * OUTPUT:    System health updates and alerts
 * BUSINESS:  Health monitoring ensures reliable work tracking
 * CHANGE:    Initial system health monitoring
 * RISK:      Medium - Health monitoring affecting system reliability
 */
func (cmo *ClaudeMonitorOrchestrator) systemHealthLoop() {
	ticker := time.NewTicker(1 * time.Minute) // Check health every minute
	defer ticker.Stop()
	
	for {
		select {
		case <-cmo.ctx.Done():
			return
		case <-ticker.C:
			cmo.checkSystemHealth()
		}
	}
}

// Report generation

/**
 * CONTEXT:   Generate periodic work time report
 * INPUT:     Current system state
 * OUTPUT:    Comprehensive work time report
 * BUSINESS:  Periodic reporting enables regular work hour analysis
 * CHANGE:    Initial periodic report generation
 * RISK:      Low - Report generation for work hour insights
 */
func (cmo *ClaudeMonitorOrchestrator) generatePeriodicReport() {
	now := time.Now()
	timeRange := reporting.TimeRange{
		Start:       cmo.lastReportTime,
		End:         now,
		Duration:    now.Sub(cmo.lastReportTime),
		Description: fmt.Sprintf("Period from %s to %s", 
			cmo.lastReportTime.Format("15:04"), now.Format("15:04")),
	}
	
	if cmo.lastReportTime.IsZero() {
		timeRange.Start = cmo.startTime
		timeRange.Duration = now.Sub(cmo.startTime)
		timeRange.Description = fmt.Sprintf("Session from %s to %s", 
			cmo.startTime.Format("15:04"), now.Format("15:04"))
	}
	
	report := cmo.workTimeCalculator.GenerateReport(reporting.ReportActiveWork, timeRange)
	
	if cmo.config.VerboseLogging {
		cmo.workTimeCalculator.PrintReport(report)
	} else {
		log.Printf("📊 Work Time Report: Active Work: %s, Sessions: %d, Activities: %d",
			report.ActiveWorkTime.Round(time.Minute),
			report.TotalSessions,
			report.TotalActivities)
	}
	
	cmo.lastReportTime = now
	
	// External callback
	if cmo.onReportGenerated != nil {
		go cmo.onReportGenerated(report)
	}
}

/**
 * CONTEXT:   Generate session-specific report
 * INPUT:     Completed work session
 * OUTPUT:    Session analysis report
 * BUSINESS:  Session reporting enables individual session analysis
 * CHANGE:    Initial session report generation
 * RISK:      Low - Session report generation for session insights
 */
func (cmo *ClaudeMonitorOrchestrator) generateSessionReport(session generator.WorkSession) {
	timeRange := reporting.TimeRange{
		Start:       session.StartTime,
		End:         session.EndTime,
		Duration:    session.Duration,
		Description: fmt.Sprintf("Session %s", session.ID),
	}
	
	report := cmo.workTimeCalculator.GenerateReport(reporting.ReportSessions, timeRange)
	
	log.Printf("📋 Session Report: %s - Efficiency: %.1f%%, Projects: %d",
		session.ID,
		report.EfficiencyRatio*100,
		len(session.ProjectStats))
}

/**
 * CONTEXT:   Generate final system report on shutdown
 * INPUT:     Complete system data
 * OUTPUT:    Final comprehensive report
 * BUSINESS:  Final reporting provides complete session analysis
 * CHANGE:    Initial final report generation
 * RISK:      Low - Final report generation for complete analysis
 */
func (cmo *ClaudeMonitorOrchestrator) generateFinalReport() {
	now := time.Now()
	timeRange := reporting.TimeRange{
		Start:       cmo.startTime,
		End:         now,
		Duration:    now.Sub(cmo.startTime),
		Description: fmt.Sprintf("Complete session from %s to %s", 
			cmo.startTime.Format("15:04:05"), now.Format("15:04:05")),
	}
	
	report := cmo.workTimeCalculator.GenerateReport(reporting.ReportWorkDay, timeRange)
	
	fmt.Printf("\n🎯 FINAL WORK SESSION REPORT\n")
	cmo.workTimeCalculator.PrintReport(report)
	
	// Also print prototype-style summary
	if cmo.comprehensiveStats != nil {
		cmo.comprehensiveStats.PrintPrototypeSummary()
	}
}

// Utility methods

func (cmo *ClaudeMonitorOrchestrator) updateComprehensiveStatistics() {
	// Update statistics from all monitoring components
	if cmo.processMonitor != nil {
		processStats := cmo.processMonitor.GetStats()
		cmo.comprehensiveStats.UpdateProcessStats(processStats)
	}
	
	if cmo.httpMonitor != nil {
		httpStats := cmo.httpMonitor.GetStats()
		cmo.comprehensiveStats.UpdateHTTPStats(httpStats)
	}
	
	if cmo.activityGenerator != nil {
		activityStats := cmo.activityGenerator.GetStats()
		cmo.comprehensiveStats.UpdateActivityStats(activityStats)
	}
}

func (cmo *ClaudeMonitorOrchestrator) checkSystemHealth() {
	// Basic health checks
	if cmo.activityGenerator != nil {
		stats := cmo.activityGenerator.GetStats()
		
		// Check if we're generating activities
		if time.Since(stats.LastActivityTime) > 30*time.Minute && cmo.running {
			if cmo.config.VerboseLogging {
				log.Printf("⚠️  No activities generated in last 30 minutes")
			}
		}
	}
}

func (cmo *ClaudeMonitorOrchestrator) printFinalStatistics() {
	uptime := time.Since(cmo.startTime)
	
	fmt.Printf("\n========== CLAUDE MONITOR FINAL STATISTICS ==========\n")
	fmt.Printf("Total Runtime: %s\n", uptime.Round(time.Second))
	fmt.Printf("Total Activities Generated: %d\n", len(cmo.activities))
	fmt.Printf("Total Sessions: %d\n", len(cmo.sessions))
	
	if cmo.activityGenerator != nil {
		stats := cmo.activityGenerator.GetStats()
		fmt.Printf("Work Activities: %d\n", stats.WorkActivities)
		fmt.Printf("Total Active Time: %s\n", stats.TotalActiveTime.Round(time.Minute))
	}
	
	fmt.Printf("====================================================\n")
}

// External integration methods

/**
 * CONTEXT:   Set callback for activity generation events
 * INPUT:     Activity callback function
 * OUTPUT:    Configured activity callback
 * BUSINESS:  Activity callback enables external integration
 * CHANGE:    Initial activity callback configuration
 * RISK:      Low - External callback configuration
 */
func (cmo *ClaudeMonitorOrchestrator) SetActivityCallback(callback func(generator.WorkActivity)) {
	cmo.onActivityGenerated = callback
}

/**
 * CONTEXT:   Set callback for session completion events
 * INPUT:     Session callback function
 * OUTPUT:    Configured session callback
 * BUSINESS:  Session callback enables external session tracking
 * CHANGE:    Initial session callback configuration
 * RISK:      Low - External callback configuration
 */
func (cmo *ClaudeMonitorOrchestrator) SetSessionCallback(callback func(generator.WorkSession)) {
	cmo.onSessionComplete = callback
}

/**
 * CONTEXT:   Set callback for report generation events
 * INPUT:     Report callback function
 * OUTPUT:    Configured report callback
 * BUSINESS:  Report callback enables external reporting integration
 * CHANGE:    Initial report callback configuration
 * RISK:      Low - External callback configuration
 */
func (cmo *ClaudeMonitorOrchestrator) SetReportCallback(callback func(*reporting.WorkTimeReport)) {
	cmo.onReportGenerated = callback
}

/**
 * CONTEXT:   Get current system statistics
 * INPUT:     Statistics request
 * OUTPUT:    Current comprehensive system statistics
 * BUSINESS:  Statistics access enables external monitoring
 * CHANGE:    Initial statistics access method
 * RISK:      Low - Statistics access for external monitoring
 */
func (cmo *ClaudeMonitorOrchestrator) GetCurrentStatistics() (*monitor.ComprehensiveStats, error) {
	if !cmo.running {
		return nil, fmt.Errorf("orchestrator is not running")
	}
	
	cmo.updateComprehensiveStatistics()
	return cmo.comprehensiveStats, nil
}

/**
 * CONTEXT:   Generate current work time report
 * INPUT:     Report type and time range
 * OUTPUT:    Current work time report
 * BUSINESS:  Report access enables external reporting
 * CHANGE:    Initial report access method
 * RISK:      Low - Report access for external reporting
 */
func (cmo *ClaudeMonitorOrchestrator) GenerateCurrentReport(reportType reporting.ReportType) (*reporting.WorkTimeReport, error) {
	if !cmo.running {
		return nil, fmt.Errorf("orchestrator is not running")
	}
	
	now := time.Now()
	timeRange := reporting.TimeRange{
		Start:       cmo.startTime,
		End:         now,
		Duration:    now.Sub(cmo.startTime),
		Description: "Current session",
	}
	
	return cmo.workTimeCalculator.GenerateReport(reportType, timeRange), nil
}