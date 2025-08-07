/**
 * CONTEXT:   Activity generator for converting monitoring events into work activities
 * INPUT:     Events from all monitoring subsystems (process, file I/O, HTTP, network)
 * OUTPUT:    Work activities, sessions, and work blocks for time tracking
 * BUSINESS:  Activity generation enables precise work hour tracking and productivity analysis
 * CHANGE:    Initial activity generator for work time calculation and session management
 * RISK:      High - Core component for accurate work time tracking and session management
 */

package generator

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/claude-monitor/system/internal/monitor"
)

/**
 * CONTEXT:   Activity types for work classification
 * INPUT:     Work activity classification requirements
 * OUTPUT:    Structured activity type definitions
 * BUSINESS:  Activity types enable precise work categorization
 * CHANGE:    Initial activity type definitions for work tracking
 * RISK:      Low - Activity type constants
 */
type ActivityType string

const (
	ActivityFileWork    ActivityType = "FILE_WORK"      // File operations on work files
	ActivityClaudeAPI   ActivityType = "CLAUDE_API"     // Claude API interactions
	ActivityCodeWork    ActivityType = "CODE_WORK"      // Coding/development work
	ActivityDocWork     ActivityType = "DOC_WORK"       // Documentation work
	ActivityProjectWork ActivityType = "PROJECT_WORK"   // General project work
	ActivityIdle        ActivityType = "IDLE"           // Idle time (>5 minutes)
)

/**
 * CONTEXT:   Work activity event structure
 * INPUT:     Generated work activity data
 * OUTPUT:    Structured work activity for time tracking
 * BUSINESS:  Work activities enable precise time tracking and productivity analysis
 * CHANGE:    Initial work activity structure based on monitoring events
 * RISK:      Low - Core data structure for work activity tracking
 */
type WorkActivity struct {
	ID             string                 `json:"id"`
	Type           ActivityType           `json:"type"`
	Timestamp      time.Time              `json:"timestamp"`
	Duration       time.Duration          `json:"duration"`
	ProjectName    string                 `json:"project_name"`
	ProjectPath    string                 `json:"project_path"`
	ProcessPID     int                    `json:"process_pid"`
	ProcessName    string                 `json:"process_name"`
	Description    string                 `json:"description"`
	Source         string                 `json:"source"`        // file_io, http, process, network
	SourceEvent    interface{}            `json:"source_event"`  // Original monitoring event
	WorkIndicator  bool                   `json:"work_indicator"` // True if this indicates active work
	Details        map[string]interface{} `json:"details"`
}

/**
 * CONTEXT:   Work session representing 5-hour Claude usage window
 * INPUT:     Session management for Claude usage tracking
 * OUTPUT:    Session structure for 5-hour work windows
 * BUSINESS:  Sessions enable Claude usage pattern tracking
 * CHANGE:    Initial session structure for 5-hour Claude windows
 * RISK:      Low - Session data structure for time management
 */
type WorkSession struct {
	ID           string        `json:"id"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`     // Always 5 hours
	ActiveTime   time.Duration `json:"active_time"`  // Actual work time within session
	WorkBlocks   []WorkBlock   `json:"work_blocks"`
	Activities   []WorkActivity `json:"activities"`
	ProjectStats map[string]ProjectSessionStats `json:"project_stats"`
	IsActive     bool          `json:"is_active"`
}

/**
 * CONTEXT:   Work block representing continuous work period
 * INPUT:     Work block management for continuous work tracking
 * OUTPUT:    Work block structure for continuous work periods
 * BUSINESS:  Work blocks enable precise active work time calculation
 * CHANGE:    Initial work block structure with idle detection
 * RISK:      Low - Work block data structure for time tracking
 */
type WorkBlock struct {
	ID          string         `json:"id"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
	Duration    time.Duration  `json:"duration"`
	ProjectName string         `json:"project_name"`
	ProjectPath string         `json:"project_path"`
	Activities  []WorkActivity `json:"activities"`
	IsActive    bool           `json:"is_active"`   // Still ongoing
	IdleTimeout time.Duration  `json:"idle_timeout"` // 5 minutes
}

/**
 * CONTEXT:   Project statistics within a session
 * INPUT:     Per-project session analytics
 * OUTPUT:    Project session statistics
 * BUSINESS:  Project session stats enable project-specific productivity analysis
 * CHANGE:    Initial project session statistics
 * RISK:      Low - Project statistics data structure
 */
type ProjectSessionStats struct {
	ProjectName    string        `json:"project_name"`
	ActiveTime     time.Duration `json:"active_time"`
	FileOperations int           `json:"file_operations"`
	ClaudeAPICalls int           `json:"claude_api_calls"`
	WorkBlocks     int           `json:"work_blocks"`
	LastActivity   time.Time     `json:"last_activity"`
}

/**
 * CONTEXT:   Activity generator for work time tracking
 * INPUT:     Monitoring events from all subsystems
 * OUTPUT:    Work activities, sessions, and time tracking
 * BUSINESS:  Activity generator provides core work time tracking functionality
 * CHANGE:    Initial activity generator implementation
 * RISK:      High - Core component for work time tracking accuracy
 */
type ActivityGenerator struct {
	ctx              context.Context
	cancel           context.CancelFunc
	activityCallback func(WorkActivity)
	sessionCallback  func(WorkSession)
	mu               sync.RWMutex
	
	// Current state
	currentSession   *WorkSession
	currentWorkBlock *WorkBlock
	lastActivityTime time.Time
	
	// Configuration
	sessionDuration time.Duration // 5 hours
	idleTimeout     time.Duration // 5 minutes
	
	// Storage
	activities   []WorkActivity
	sessions     []WorkSession
	workBlocks   []WorkBlock
	
	// Statistics
	stats ActivityGeneratorStats
}

/**
 * CONTEXT:   Activity generator statistics
 * INPUT:     Activity generation metrics
 * OUTPUT:    Generator performance and activity statistics
 * BUSINESS:  Statistics enable activity generation optimization
 * CHANGE:    Initial activity generator statistics
 * RISK:      Low - Statistics structure for generator metrics
 */
type ActivityGeneratorStats struct {
	TotalActivities      uint64            `json:"total_activities"`
	WorkActivities       uint64            `json:"work_activities"`
	ActivitiesByType     map[string]uint64 `json:"activities_by_type"`
	ActivitiesByProject  map[string]uint64 `json:"activities_by_project"`
	TotalSessions        uint64            `json:"total_sessions"`
	TotalWorkBlocks      uint64            `json:"total_work_blocks"`
	TotalActiveTime      time.Duration     `json:"total_active_time"`
	AverageSessionTime   time.Duration     `json:"average_session_time"`
	CurrentSessionActive bool              `json:"current_session_active"`
	LastActivityTime     time.Time         `json:"last_activity_time"`
	StartTime            time.Time         `json:"start_time"`
}

/**
 * CONTEXT:   Create new activity generator
 * INPUT:     Activity and session callback functions
 * OUTPUT:    Configured activity generator ready for event processing
 * BUSINESS:  Generator creation enables work time tracking system
 * CHANGE:    Initial activity generator constructor
 * RISK:      Medium - Generator initialization for work tracking
 */
func NewActivityGenerator(activityCallback func(WorkActivity), sessionCallback func(WorkSession)) *ActivityGenerator {
	ctx, cancel := context.WithCancel(context.Background())
	
	generator := &ActivityGenerator{
		ctx:              ctx,
		cancel:           cancel,
		activityCallback: activityCallback,
		sessionCallback:  sessionCallback,
		sessionDuration:  5 * time.Hour,  // 5-hour Claude sessions
		idleTimeout:      5 * time.Minute, // 5-minute idle timeout
		activities:       make([]WorkActivity, 0),
		sessions:         make([]WorkSession, 0),
		workBlocks:       make([]WorkBlock, 0),
		stats: ActivityGeneratorStats{
			ActivitiesByType:    make(map[string]uint64),
			ActivitiesByProject: make(map[string]uint64),
			StartTime:           time.Now(),
		},
	}
	
	// Start cleanup goroutine for idle detection
	go generator.idleDetectionLoop()
	
	return generator
}

/**
 * CONTEXT:   Process file I/O event to generate work activity
 * INPUT:     File I/O event from file monitor
 * OUTPUT:    Generated work activity if event indicates work
 * BUSINESS:  File I/O processing enables file-based work detection
 * CHANGE:    Initial file I/O event processing for work activity generation
 * RISK:      Medium - File I/O activity generation accuracy
 */
func (ag *ActivityGenerator) ProcessFileIOEvent(event monitor.FileIOEvent) {
	if !event.IsWorkFile {
		return // Skip non-work files
	}
	
	activityType := ag.classifyFileActivity(event)
	
	activity := WorkActivity{
		ID:            ag.generateActivityID(),
		Type:          activityType,
		Timestamp:     event.Timestamp,
		Duration:      time.Second, // Instantaneous file operation
		ProjectName:   event.ProjectName,
		ProjectPath:   event.ProjectPath,
		ProcessPID:    event.PID,
		ProcessName:   event.ProcessName,
		Description:   ag.generateFileActivityDescription(event),
		Source:        "file_io",
		SourceEvent:   event,
		WorkIndicator: true,
		Details: map[string]interface{}{
			"file_path": event.FilePath,
			"file_type": event.FileType,
			"io_type":   event.Type,
			"size":      event.Size,
		},
	}
	
	ag.processWorkActivity(activity)
}

/**
 * CONTEXT:   Process HTTP event to generate work activity
 * INPUT:     HTTP event from HTTP monitor
 * OUTPUT:    Generated work activity if event indicates Claude API usage
 * BUSINESS:  HTTP processing enables Claude API work detection
 * CHANGE:    Initial HTTP event processing for work activity generation
 * RISK:      Medium - HTTP activity generation accuracy
 */
func (ag *ActivityGenerator) ProcessHTTPEvent(event monitor.HTTPEvent) {
	if !event.IsClaudeAPI {
		return // Only process Claude API events
	}
	
	activity := WorkActivity{
		ID:            ag.generateActivityID(),
		Type:          ActivityClaudeAPI,
		Timestamp:     event.Timestamp,
		Duration:      time.Second, // API call duration
		ProjectName:   event.ProjectName,
		ProjectPath:   event.ProjectPath,
		ProcessPID:    event.PID,
		ProcessName:   event.ProcessName,
		Description:   fmt.Sprintf("Claude API %s to %s", event.Method, event.Host),
		Source:        "http",
		SourceEvent:   event,
		WorkIndicator: true,
		Details: map[string]interface{}{
			"method":         event.Method,
			"host":           event.Host,
			"url":            event.URL,
			"status_code":    event.StatusCode,
			"content_length": event.ContentLength,
		},
	}
	
	ag.processWorkActivity(activity)
}

/**
 * CONTEXT:   Process network event to generate work activity
 * INPUT:     Network event from network monitor
 * OUTPUT:    Generated work activity if event indicates Claude connection
 * BUSINESS:  Network processing enables connection-based work detection
 * CHANGE:    Initial network event processing for work activity generation
 * RISK:      Low - Network activity generation
 */
func (ag *ActivityGenerator) ProcessNetworkEvent(event monitor.NetworkEvent) {
	if !event.IsClaudeAPI {
		return // Only process Claude API connections
	}
	
	activity := WorkActivity{
		ID:            ag.generateActivityID(),
		Type:          ActivityClaudeAPI,
		Timestamp:     event.Timestamp,
		Duration:      time.Second,
		ProjectName:   event.ProjectName,
		ProjectPath:   event.ProjectPath,
		ProcessPID:    event.PID,
		ProcessName:   event.ProcessName,
		Description:   fmt.Sprintf("Claude API connection to %s", event.RemoteHost),
		Source:        "network",
		SourceEvent:   event,
		WorkIndicator: true,
		Details: map[string]interface{}{
			"protocol":     event.Protocol,
			"local_addr":   event.LocalAddr,
			"remote_addr":  event.RemoteAddr,
			"remote_host":  event.RemoteHost,
		},
	}
	
	ag.processWorkActivity(activity)
}

/**
 * CONTEXT:   Process process event to generate work activity
 * INPUT:     Process event from process monitor
 * OUTPUT:    Generated work activity for process lifecycle
 * BUSINESS:  Process processing enables process-based work session tracking
 * CHANGE:    Initial process event processing for work activity generation
 * RISK:      Low - Process activity generation
 */
func (ag *ActivityGenerator) ProcessProcessEvent(event monitor.ProcessEvent) {
	var activityType ActivityType
	var description string
	
	switch event.Type {
	case monitor.ProcessStarted:
		activityType = ActivityProjectWork
		description = fmt.Sprintf("Started Claude process: %s", event.Command)
	case monitor.ProcessStopped:
		activityType = ActivityProjectWork
		description = fmt.Sprintf("Stopped Claude process: %s", event.Command)
	default:
		return // Skip unknown process events
	}
	
	activity := WorkActivity{
		ID:            ag.generateActivityID(),
		Type:          activityType,
		Timestamp:     event.Timestamp,
		Duration:      time.Second,
		ProjectName:   ag.extractProjectNameFromPath(event.WorkingDir),
		ProjectPath:   event.WorkingDir,
		ProcessPID:    event.PID,
		ProcessName:   event.Command,
		Description:   description,
		Source:        "process",
		SourceEvent:   event,
		WorkIndicator: true,
		Details: map[string]interface{}{
			"command":     event.Command,
			"working_dir": event.WorkingDir,
			"user_id":     event.UserID,
			"args":        event.Args,
		},
	}
	
	ag.processWorkActivity(activity)
}

/**
 * CONTEXT:   Process work activity and update sessions/work blocks
 * INPUT:     Generated work activity
 * OUTPUT:    Updated sessions, work blocks, and activity tracking
 * BUSINESS:  Activity processing maintains work time tracking state
 * CHANGE:    Initial work activity processing with session management
 * RISK:      High - Core work time tracking logic
 */
func (ag *ActivityGenerator) processWorkActivity(activity WorkActivity) {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	
	now := activity.Timestamp
	ag.lastActivityTime = now
	
	// Update statistics
	ag.stats.TotalActivities++
	if activity.WorkIndicator {
		ag.stats.WorkActivities++
	}
	ag.stats.ActivitiesByType[string(activity.Type)]++
	if activity.ProjectName != "" {
		ag.stats.ActivitiesByProject[activity.ProjectName]++
	}
	ag.stats.LastActivityTime = now
	
	// Manage session
	ag.ensureActiveSession(now)
	
	// Manage work block
	ag.ensureActiveWorkBlock(activity, now)
	
	// Add activity to current work block and session
	if ag.currentWorkBlock != nil {
		ag.currentWorkBlock.Activities = append(ag.currentWorkBlock.Activities, activity)
	}
	
	if ag.currentSession != nil {
		ag.currentSession.Activities = append(ag.currentSession.Activities, activity)
		ag.updateSessionStats(activity)
	}
	
	// Store activity
	ag.activities = append(ag.activities, activity)
	
	// Send activity callback
	if ag.activityCallback != nil {
		go ag.activityCallback(activity)
	}
	
	log.Printf("⚡ Work Activity: %s in %s - %s", activity.Type, activity.ProjectName, activity.Description)
}

/**
 * CONTEXT:   Ensure active session exists (5-hour window management)
 * INPUT:     Current timestamp
 * OUTPUT:    Active session for current time period
 * BUSINESS:  Session management enables 5-hour Claude usage tracking
 * CHANGE:    Initial session management for 5-hour windows
 * RISK:      Medium - Session timing accuracy affecting work tracking
 */
func (ag *ActivityGenerator) ensureActiveSession(timestamp time.Time) {
	// Check if current session is still valid (within 5 hours)
	if ag.currentSession != nil {
		sessionEnd := ag.currentSession.StartTime.Add(ag.sessionDuration)
		if timestamp.Before(sessionEnd) {
			return // Current session still valid
		}
		
		// Close current session
		ag.closeCurrentSession()
	}
	
	// Create new session
	sessionID := fmt.Sprintf("session_%d", timestamp.Unix())
	ag.currentSession = &WorkSession{
		ID:           sessionID,
		StartTime:    timestamp,
		EndTime:      timestamp.Add(ag.sessionDuration),
		Duration:     ag.sessionDuration,
		WorkBlocks:   make([]WorkBlock, 0),
		Activities:   make([]WorkActivity, 0),
		ProjectStats: make(map[string]ProjectSessionStats),
		IsActive:     true,
	}
	
	ag.stats.TotalSessions++
	ag.stats.CurrentSessionActive = true
	
	log.Printf("🎯 New Claude Session Started: %s (5-hour window)", sessionID)
}

/**
 * CONTEXT:   Ensure active work block exists (continuous work tracking)
 * INPUT:     Work activity and timestamp
 * OUTPUT:    Active work block for continuous work
 * BUSINESS:  Work block management enables continuous work time calculation
 * CHANGE:    Initial work block management with idle detection
 * RISK:      High - Work block timing accuracy affecting work hour calculation
 */
func (ag *ActivityGenerator) ensureActiveWorkBlock(activity WorkActivity, timestamp time.Time) {
	// Check if we need to start a new work block (idle timeout or different project)
	needNewBlock := ag.currentWorkBlock == nil ||
		timestamp.Sub(ag.lastActivityTime) > ag.idleTimeout ||
		ag.currentWorkBlock.ProjectName != activity.ProjectName
	
	if needNewBlock {
		// Close current work block if exists
		if ag.currentWorkBlock != nil {
			ag.closeCurrentWorkBlock(timestamp)
		}
		
		// Create new work block
		blockID := fmt.Sprintf("block_%s_%d", activity.ProjectName, timestamp.Unix())
		ag.currentWorkBlock = &WorkBlock{
			ID:          blockID,
			StartTime:   timestamp,
			ProjectName: activity.ProjectName,
			ProjectPath: activity.ProjectPath,
			Activities:  make([]WorkActivity, 0),
			IsActive:    true,
			IdleTimeout: ag.idleTimeout,
		}
		
		ag.stats.TotalWorkBlocks++
		
		log.Printf("📊 New Work Block Started: %s in %s", blockID, activity.ProjectName)
	}
}

/**
 * CONTEXT:   Close current session and update statistics
 * INPUT:     Session closure trigger
 * OUTPUT:    Closed session with calculated statistics
 * BUSINESS:  Session closure enables session analytics and archiving
 * CHANGE:    Initial session closure with statistics calculation
 * RISK:      Medium - Session closure accuracy affecting analytics
 */
func (ag *ActivityGenerator) closeCurrentSession() {
	if ag.currentSession == nil {
		return
	}
	
	// Close any active work block
	if ag.currentWorkBlock != nil {
		ag.closeCurrentWorkBlock(time.Now())
	}
	
	// Calculate session statistics
	ag.currentSession.IsActive = false
	
	// Calculate active time from work blocks
	var totalActiveTime time.Duration
	for _, block := range ag.currentSession.WorkBlocks {
		totalActiveTime += block.Duration
	}
	ag.currentSession.ActiveTime = totalActiveTime
	ag.stats.TotalActiveTime += totalActiveTime
	
	// Store completed session
	ag.sessions = append(ag.sessions, *ag.currentSession)
	
	// Send session callback
	if ag.sessionCallback != nil {
		go ag.sessionCallback(*ag.currentSession)
	}
	
	log.Printf("✅ Session Completed: %s - Active Time: %s", 
		ag.currentSession.ID, ag.currentSession.ActiveTime.Round(time.Minute))
	
	ag.currentSession = nil
	ag.stats.CurrentSessionActive = false
}

/**
 * CONTEXT:   Close current work block and update duration
 * INPUT:     Work block closure timestamp
 * OUTPUT:    Closed work block with calculated duration
 * BUSINESS:  Work block closure enables precise work time calculation
 * CHANGE:    Initial work block closure with duration calculation
 * RISK:      High - Work block duration accuracy affecting work hour tracking
 */
func (ag *ActivityGenerator) closeCurrentWorkBlock(endTime time.Time) {
	if ag.currentWorkBlock == nil {
		return
	}
	
	// Calculate duration
	ag.currentWorkBlock.EndTime = endTime
	ag.currentWorkBlock.Duration = endTime.Sub(ag.currentWorkBlock.StartTime)
	ag.currentWorkBlock.IsActive = false
	
	// Add to current session
	if ag.currentSession != nil {
		ag.currentSession.WorkBlocks = append(ag.currentSession.WorkBlocks, *ag.currentWorkBlock)
	}
	
	// Store completed work block
	ag.workBlocks = append(ag.workBlocks, *ag.currentWorkBlock)
	
	log.Printf("⏱️  Work Block Completed: %s - Duration: %s", 
		ag.currentWorkBlock.ID, ag.currentWorkBlock.Duration.Round(time.Second))
	
	ag.currentWorkBlock = nil
}

/**
 * CONTEXT:   Idle detection loop for automatic work block closure
 * INPUT:     Continuous idle time monitoring
 * OUTPUT:    Automatic work block closure on idle timeout
 * BUSINESS:  Idle detection enables accurate work time calculation
 * CHANGE:    Initial idle detection for work block management
 * RISK:      Medium - Idle detection accuracy affecting work time calculation
 */
func (ag *ActivityGenerator) idleDetectionLoop() {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()
	
	for {
		select {
		case <-ag.ctx.Done():
			return
		case <-ticker.C:
			ag.checkForIdleTimeout()
		}
	}
}

/**
 * CONTEXT:   Check for idle timeout and close work blocks
 * INPUT:     Idle timeout check
 * OUTPUT:    Work block closure if idle timeout exceeded
 * BUSINESS:  Idle timeout ensures accurate work time tracking
 * CHANGE:    Initial idle timeout implementation
 * RISK:      Medium - Idle timeout accuracy affecting work hour calculation
 */
func (ag *ActivityGenerator) checkForIdleTimeout() {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	
	if ag.currentWorkBlock == nil {
		return
	}
	
	now := time.Now()
	timeSinceLastActivity := now.Sub(ag.lastActivityTime)
	
	if timeSinceLastActivity > ag.idleTimeout {
		log.Printf("💤 Idle timeout detected (%s), closing work block", timeSinceLastActivity.Round(time.Second))
		ag.closeCurrentWorkBlock(ag.lastActivityTime.Add(ag.idleTimeout))
	}
}

// Helper methods

func (ag *ActivityGenerator) classifyFileActivity(event monitor.FileIOEvent) ActivityType {
	switch event.FileType {
	case ".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".rs":
		return ActivityCodeWork
	case ".md", ".txt", ".doc", ".docx":
		return ActivityDocWork
	default:
		return ActivityFileWork
	}
}

func (ag *ActivityGenerator) generateFileActivityDescription(event monitor.FileIOEvent) string {
	return fmt.Sprintf("File %s: %s", event.Type, event.FilePath)
}

func (ag *ActivityGenerator) generateActivityID() string {
	return fmt.Sprintf("activity_%d", time.Now().UnixNano())
}

func (ag *ActivityGenerator) extractProjectNameFromPath(path string) string {
	if path == "" {
		return "unknown"
	}
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return "unknown"
}

func (ag *ActivityGenerator) updateSessionStats(activity WorkActivity) {
	if ag.currentSession == nil {
		return
	}
	
	projectName := activity.ProjectName
	if projectName == "" {
		projectName = "unknown"
	}
	
	stats, exists := ag.currentSession.ProjectStats[projectName]
	if !exists {
		stats = ProjectSessionStats{
			ProjectName: projectName,
		}
	}
	
	stats.LastActivity = activity.Timestamp
	
	switch activity.Type {
	case ActivityClaudeAPI:
		stats.ClaudeAPICalls++
	case ActivityFileWork, ActivityCodeWork, ActivityDocWork:
		stats.FileOperations++
	}
	
	ag.currentSession.ProjectStats[projectName] = stats
}

/**
 * CONTEXT:   Get activity generator statistics
 * INPUT:     Statistics request
 * OUTPUT:    Current activity generator metrics
 * BUSINESS:  Statistics provide activity generation insights
 * CHANGE:    Initial statistics getter for activity generator
 * RISK:      Low - Read-only statistics access
 */
func (ag *ActivityGenerator) GetStats() ActivityGeneratorStats {
	ag.mu.RLock()
	defer ag.mu.RUnlock()
	
	stats := ag.stats
	stats.ActivitiesByType = make(map[string]uint64)
	stats.ActivitiesByProject = make(map[string]uint64)
	
	for k, v := range ag.stats.ActivitiesByType {
		stats.ActivitiesByType[k] = v
	}
	for k, v := range ag.stats.ActivitiesByProject {
		stats.ActivitiesByProject[k] = v
	}
	
	return stats
}

/**
 * CONTEXT:   Stop activity generator and cleanup
 * INPUT:     Generator shutdown request
 * OUTPUT:    Stopped generator with cleanup
 * BUSINESS:  Generator stop enables graceful shutdown
 * CHANGE:    Initial stop implementation with cleanup
 * RISK:      Medium - Generator shutdown affecting active tracking
 */
func (ag *ActivityGenerator) Stop() {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	
	// Close current work block and session
	if ag.currentWorkBlock != nil {
		ag.closeCurrentWorkBlock(time.Now())
	}
	
	if ag.currentSession != nil {
		ag.closeCurrentSession()
	}
	
	ag.cancel()
	
	log.Printf("Activity generator stopped - Total activities: %d, Total sessions: %d", 
		ag.stats.TotalActivities, ag.stats.TotalSessions)
}