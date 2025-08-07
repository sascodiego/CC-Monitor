/**
 * CONTEXT:   Activity event generator for Claude work hour tracking
 * INPUT:     Process and file I/O events from monitoring systems
 * OUTPUT:    Activity events for daemon consumption and work tracking
 * BUSINESS:  Activity generation converts monitoring data to trackable work events
 * CHANGE:    Initial activity generator for enhanced monitoring integration
 * RISK:      High - Activity generation affecting work time calculations
 */

package monitor

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

/**
 * CONTEXT:   Activity event structure for daemon consumption
 * INPUT:     Generated activity data from monitoring events
 * OUTPUT:    Structured activity event for work tracking
 * BUSINESS:  Activity events enable accurate work hour calculation
 * CHANGE:    Initial activity event structure compatible with daemon
 * RISK:      Medium - Event structure affecting daemon integration
 */
type ActivityEvent struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	ProjectPath    string                 `json:"project_path"`
	ProjectName    string                 `json:"project_name"`
	ActivityType   string                 `json:"activity_type"`
	ActivitySource string                 `json:"activity_source"`
	Timestamp      time.Time              `json:"timestamp"`
	Command        string                 `json:"command"`
	Description    string                 `json:"description"`
	Metadata       map[string]interface{} `json:"metadata"`
}

/**
 * CONTEXT:   Activity generator for converting monitoring events to work activities
 * INPUT:     Process events and file I/O events from monitors
 * OUTPUT:    Activity events sent to daemon for work tracking
 * BUSINESS:  Activity generator enables comprehensive work hour tracking
 * CHANGE:    Initial activity generator with event correlation
 * RISK:      High - Activity generation affecting work tracking accuracy
 */
type ActivityGenerator struct {
	activityCallback    func(ActivityEvent)
	recentProcessEvents []ProcessEvent
	recentFileIOEvents  []FileIOEvent
	eventBuffer         map[string]*ActivitySession // Buffer events by project
	mu                  sync.RWMutex
	stats              ActivityGeneratorStats
	config             ActivityGeneratorConfig
}

/**
 * CONTEXT:   Activity session for correlating related events
 * INPUT:     Related process and file events for same project
 * OUTPUT:    Correlated activity session data
 * BUSINESS:  Event correlation improves activity detection accuracy
 * CHANGE:    Initial activity session for event correlation
 * RISK:      Medium - Event correlation affecting activity accuracy
 */
type ActivitySession struct {
	ProjectPath     string
	ProjectName     string
	LastProcessPID  int
	LastActivity    time.Time
	ProcessEvents   []ProcessEvent
	FileIOEvents    []FileIOEvent
	ActivityCount   int
	WorkFileCount   int
}

/**
 * CONTEXT:   Activity generator configuration for POST-based work detection
 * INPUT:     Configuration parameters for POST request work detection
 * OUTPUT:    Generator behavior settings for work/idle detection
 * BUSINESS:  Configuration enables POST request based work activity detection
 * CHANGE:    Simplified configuration focused on POST request work detection
 * RISK:      Low - Simplified configuration for work-focused activity generation
 */
type ActivityGeneratorConfig struct {
	EventBufferDuration  time.Duration `json:"event_buffer_duration"`   // How long to buffer events
	ActivityTimeout      time.Duration `json:"activity_timeout"`        // Time before generating activity from POST
	PostRequestOnly      bool          `json:"post_request_only"`       // Only generate activities from POST requests
	IdleTimeoutMinutes   int           `json:"idle_timeout_minutes"`    // Minutes without POST = idle
	VerboseLogging       bool          `json:"verbose_logging"`         // Enable detailed logging
}

/**
 * CONTEXT:   Activity generator statistics
 * INPUT:     Activity generation metrics
 * OUTPUT:    Performance and generation statistics
 * BUSINESS:  Statistics enable activity generation optimization
 * CHANGE:    Initial statistics structure for activity generator
 * RISK:      Low - Statistics structure for monitoring metrics
 */
type ActivityGeneratorStats struct {
	TotalProcessEvents   uint64 `json:"total_process_events"`
	TotalFileIOEvents    uint64 `json:"total_file_io_events"`
	GeneratedActivities  uint64 `json:"generated_activities"`
	WorkActivities       uint64 `json:"work_activities"`
	ProcessActivities    uint64 `json:"process_activities"`
	CorrelatedSessions   uint64 `json:"correlated_sessions"`
	StartTime           time.Time `json:"start_time"`
	LastActivityTime    time.Time `json:"last_activity_time"`
}

/**
 * CONTEXT:   Create new activity generator for POST-based work detection
 * INPUT:     Activity callback function and POST-focused configuration
 * OUTPUT:    Configured activity generator ready for POST request work tracking
 * BUSINESS:  Generator creation sets up POST request work activity detection pipeline
 * CHANGE:    Simplified activity generator focused on POST request work detection
 * RISK:      Medium - Generator initialization for POST-focused work tracking
 */
func NewActivityGenerator(callback func(ActivityEvent), config ActivityGeneratorConfig) *ActivityGenerator {
	// Set default configuration values for POST-based work detection
	if config.EventBufferDuration == 0 {
		config.EventBufferDuration = 5 * time.Second // Shorter buffer for POST requests
	}
	if config.ActivityTimeout == 0 {
		config.ActivityTimeout = 15 * time.Second // Quick response to POST requests
	}
	if config.IdleTimeoutMinutes == 0 {
		config.IdleTimeoutMinutes = 5 // 5 minutes without POST = idle
	}
	
	// Enable POST-only mode by default
	config.PostRequestOnly = true
	
	return &ActivityGenerator{
		activityCallback: callback,
		eventBuffer:     make(map[string]*ActivitySession),
		config:          config,
		stats: ActivityGeneratorStats{
			StartTime: time.Now(),
		},
	}
}

/**
 * CONTEXT:   Process incoming process events for activity generation
 * INPUT:     Process event from process monitor
 * OUTPUT:    Processed event with potential activity generation
 * BUSINESS:  Process event handling enables work session correlation
 * CHANGE:    Initial process event handling with session correlation
 * RISK:      Medium - Process event handling affecting activity generation
 */
func (ag *ActivityGenerator) HandleProcessEvent(event ProcessEvent) {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	
	ag.stats.TotalProcessEvents++
	ag.recentProcessEvents = append(ag.recentProcessEvents, event)
	
	// Keep only recent events (last 100)
	if len(ag.recentProcessEvents) > 100 {
		ag.recentProcessEvents = ag.recentProcessEvents[1:]
	}
	
	if ag.config.VerboseLogging {
		log.Printf("📥 Processing process event: %s for %s (PID: %d)", 
			event.Type, event.Command, event.PID)
	}
	
	// Extract project information
	projectPath := event.WorkingDir
	projectName := extractProjectNameFromPath(projectPath)
	sessionKey := projectPath
	
	// Get or create activity session
	session := ag.getOrCreateSession(sessionKey, projectPath, projectName)
	session.ProcessEvents = append(session.ProcessEvents, event)
	session.LastActivity = event.Timestamp
	session.LastProcessPID = event.PID
	session.ActivityCount++
	
	// Skip process event generation in POST-only mode
	if ag.config.PostRequestOnly {
		return
	}
	
	// Generate activity based on event type (deprecated in POST-only mode)
	switch event.Type {
	case ProcessStarted:
		ag.generateProcessActivity(event, session, "claude_process_started")
	case ProcessStopped:
		ag.generateProcessActivity(event, session, "claude_process_stopped")
	}
	
	// Check if we should generate a work activity from accumulated events
	ag.checkAndGenerateWorkActivity(session)
}

/**
 * CONTEXT:   Process incoming HTTP POST events for work activity generation
 * INPUT:     HTTP POST event from HTTP monitor (Claude API calls)
 * OUTPUT:    Work activity generated from POST request
 * BUSINESS:  HTTP POST event handling enables precise work activity detection
 * CHANGE:    New HTTP POST event handling for definitive work activity tracking
 * RISK:      High - POST event handling directly affecting work activity detection
 */
func (ag *ActivityGenerator) HandleHTTPPostEvent(event HTTPEvent) {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	
	// Only process POST requests to Claude API
	if (event.Method != "POST" && event.Method != "POST/PUT" && event.Method != "PUT") || !event.IsClaudeAPI {
		return
	}
	
	ag.stats.TotalFileIOEvents++ // Using file IO stats for now
	
	if ag.config.VerboseLogging {
		log.Printf("🔥 Processing POST request for work activity: %s", event.Host)
	}
	
	// Extract project information from process context
	projectPath := event.ProjectPath
	projectName := event.ProjectName
	if projectName == "" {
		projectName = "claude_work"
	}
	
	sessionKey := fmt.Sprintf("post_%s", projectPath)
	if sessionKey == "post_" {
		sessionKey = fmt.Sprintf("post_pid_%d", event.PID)
	}
	
	// Get or create activity session
	session := ag.getOrCreateSession(sessionKey, projectPath, projectName)
	
	// Store POST event details
	session.LastActivity = event.Timestamp
	session.ActivityCount++
	session.WorkFileCount++ // Count POST as work activity
	
	// Generate immediate work activity from POST request
	ag.generatePOSTWorkActivity(event, session)
}

/**
 * CONTEXT:   Generate work activity from Claude API POST request
 * INPUT:     HTTP POST event to Claude API and activity session
 * OUTPUT:    Work activity event sent to daemon for work tracking
 * BUSINESS:  POST work activity generation enables definitive work activity detection
 * CHANGE:    New POST-based work activity generation for precise work tracking
 * RISK:      High - POST work activity generation directly affecting work time calculations
 */
func (ag *ActivityGenerator) generatePOSTWorkActivity(event HTTPEvent, session *ActivitySession) {
	// Create work activity from POST request
	activity := ActivityEvent{
		ID:             generateActivityID(),
		UserID:         "claude_user", // Single global user for all Claude sessions
		ProjectPath:    session.ProjectPath,
		ProjectName:    session.ProjectName,
		ActivityType:   "claude_post_work",
		ActivitySource: "http_post_monitor",
		Timestamp:      event.Timestamp,
		Command:        "claude_api_post",
		Description:    fmt.Sprintf("Claude API work activity: POST to %s", event.Host),
		Metadata: map[string]interface{}{
			"post_url":        event.URL,
			"post_host":       event.Host,
			"content_length":  event.ContentLength,
			"work_indicator":  true,
			"idle_reset":      true, // This POST resets idle detection
			"api_endpoint":    event.Host,
		},
	}

	ag.sendActivity(activity)
	ag.stats.WorkActivities++

	if ag.config.VerboseLogging {
		log.Printf("🔥 Generated POST work activity: %s for %s", 
			activity.ActivityType, session.ProjectName)
	}
}

/**
 * CONTEXT:   Process incoming file I/O events for activity generation (deprecated)
 * INPUT:     File I/O event from file monitor
 * OUTPUT:    Processed event with potential activity generation
 * BUSINESS:  File I/O event handling (now secondary to POST requests)
 * CHANGE:    Deprecated file I/O handling in favor of POST request focus
 * RISK:      Low - Secondary file I/O handling for POST-focused system
 */
func (ag *ActivityGenerator) HandleFileIOEvent(event FileIOEvent) {
	// Skip file I/O processing if in POST-only mode
	if ag.config.PostRequestOnly {
		return
	}
	ag.mu.Lock()
	defer ag.mu.Unlock()
	
	ag.stats.TotalFileIOEvents++
	ag.recentFileIOEvents = append(ag.recentFileIOEvents, event)
	
	// Keep only recent events (last 200)
	if len(ag.recentFileIOEvents) > 200 {
		ag.recentFileIOEvents = ag.recentFileIOEvents[1:]
	}
	
	if ag.config.VerboseLogging {
		log.Printf("📁 Processing file I/O event: %s for %s (PID: %d)", 
			event.Type, event.FilePath, event.PID)
	}
	
	// Only process work-related file events
	if !event.IsWorkFile {
		return
	}
	
	// Extract project information
	projectPath := event.ProjectPath
	projectName := event.ProjectName
	sessionKey := projectPath
	
	if sessionKey == "" {
		sessionKey = fmt.Sprintf("pid_%d", event.PID)
		projectName = "unknown"
	}
	
	// Get or create activity session
	session := ag.getOrCreateSession(sessionKey, projectPath, projectName)
	session.FileIOEvents = append(session.FileIOEvents, event)
	session.LastActivity = event.Timestamp
	session.ActivityCount++
	session.WorkFileCount++
	
	// Generate file I/O activity (deprecated in POST-only mode)
	ag.generateFileIOActivity(event, session)
	
	// Check if we should generate a work activity from accumulated events
	ag.checkAndGenerateWorkActivity(session)
}

/**
 * CONTEXT:   Get or create activity session for event correlation
 * INPUT:     Session key, project path, and project name
 * OUTPUT:    Activity session for event correlation
 * BUSINESS:  Session management enables event correlation and work tracking
 * CHANGE:    Initial session management for event correlation
 * RISK:      Medium - Session management affecting event correlation
 */
func (ag *ActivityGenerator) getOrCreateSession(sessionKey, projectPath, projectName string) *ActivitySession {
	session, exists := ag.eventBuffer[sessionKey]
	if !exists {
		session = &ActivitySession{
			ProjectPath:   projectPath,
			ProjectName:   projectName,
			LastActivity:  time.Now(),
			ProcessEvents: make([]ProcessEvent, 0),
			FileIOEvents:  make([]FileIOEvent, 0),
		}
		ag.eventBuffer[sessionKey] = session
		ag.stats.CorrelatedSessions++
	}
	
	return session
}

/**
 * CONTEXT:   Check if accumulated events warrant generating work activity
 * INPUT:     Activity session with accumulated events
 * OUTPUT:    Potential work activity generation
 * BUSINESS:  Work activity generation enables accurate work time tracking
 * CHANGE:    Initial work activity generation logic
 * RISK:      High - Work activity generation affecting time tracking accuracy
 */
func (ag *ActivityGenerator) checkAndGenerateWorkActivity(session *ActivitySession) {
	now := time.Now()
	
	// Check if enough activity has accumulated or timeout reached
	shouldGenerate := false
	activityType := "claude_work"
	
	// Skip traditional work activity generation in POST-only mode
	if ag.config.PostRequestOnly {
		return // POST requests generate immediate activities
	}
	
	// Generate if enough work file activity (deprecated in POST-only mode)
	if session.WorkFileCount >= 3 { // Hardcoded for legacy compatibility
		shouldGenerate = true
		activityType = "claude_file_work"
	}
	
	// Generate if session has been active for a while
	if now.Sub(session.LastActivity) >= ag.config.ActivityTimeout && session.ActivityCount > 0 {
		shouldGenerate = true
	}
	
	// Generate if mix of process and file events indicates work
	if len(session.ProcessEvents) > 0 && len(session.FileIOEvents) > 0 {
		shouldGenerate = true
		activityType = "claude_interactive_work"
	}
	
	if shouldGenerate {
		ag.generateWorkActivity(session, activityType)
		
		// Reset session counters but keep recent events
		session.ActivityCount = 0
		session.WorkFileCount = 0
		
		// Keep only recent events
		if len(session.ProcessEvents) > 5 {
			session.ProcessEvents = session.ProcessEvents[len(session.ProcessEvents)-5:]
		}
		if len(session.FileIOEvents) > 10 {
			session.FileIOEvents = session.FileIOEvents[len(session.FileIOEvents)-10:]
		}
	}
}

/**
 * CONTEXT:   Generate process-based activity event
 * INPUT:     Process event, session, and activity type
 * OUTPUT:    Activity event sent to daemon
 * BUSINESS:  Process activity generation tracks Claude lifecycle events
 * CHANGE:    Initial process activity generation
 * RISK:      Medium - Process activity generation affecting event tracking
 */
func (ag *ActivityGenerator) generateProcessActivity(event ProcessEvent, session *ActivitySession, activityType string) {
	activity := ActivityEvent{
		ID:             generateActivityID(),
		UserID:         event.UserID,
		ProjectPath:    session.ProjectPath,
		ProjectName:    session.ProjectName,
		ActivityType:   activityType,
		ActivitySource: "enhanced_process_monitor",
		Timestamp:      event.Timestamp,
		Command:        event.Command,
		Description:    fmt.Sprintf("Claude process %s: %s (PID: %d)", event.Type, event.Command, event.PID),
		Metadata: map[string]interface{}{
			"process_pid":  event.PID,
			"process_ppid": event.PPID,
			"process_cmd":  event.Command,
			"working_dir":  event.WorkingDir,
			"event_type":   event.Type,
		},
	}
	
	ag.sendActivity(activity)
	ag.stats.ProcessActivities++
}

/**
 * CONTEXT:   Generate file I/O based activity event
 * INPUT:     File I/O event and session
 * OUTPUT:    Activity event sent to daemon
 * BUSINESS:  File I/O activity generation tracks work file operations
 * CHANGE:    Initial file I/O activity generation
 * RISK:      Medium - File I/O activity generation affecting work tracking
 */
func (ag *ActivityGenerator) generateFileIOActivity(event FileIOEvent, session *ActivitySession) {
	activity := ActivityEvent{
		ID:             generateActivityID(),
		UserID:         "claude_user", // Single global user for all Claude sessions
		ProjectPath:    session.ProjectPath,
		ProjectName:    session.ProjectName,
		ActivityType:   "claude_file_operation",
		ActivitySource: "enhanced_file_monitor",
		Timestamp:      event.Timestamp,
		Command:        event.ProcessName,
		Description:    fmt.Sprintf("File %s: %s", event.Type, filepath.Base(event.FilePath)),
		Metadata: map[string]interface{}{
			"file_path":     event.FilePath,
			"file_type":     event.FileType,
			"file_size":     event.Size,
			"io_type":       event.Type,
			"process_pid":   event.PID,
			"process_name":  event.ProcessName,
			"is_work_file":  event.IsWorkFile,
		},
	}
	
	ag.sendActivity(activity)
}

/**
 * CONTEXT:   Generate comprehensive work activity from accumulated events
 * INPUT:     Activity session with accumulated events and activity type
 * OUTPUT:    Work activity event sent to daemon
 * BUSINESS:  Work activity generation provides core work time tracking
 * CHANGE:    Initial comprehensive work activity generation
 * RISK:      High - Work activity generation directly affecting work time calculations
 */
func (ag *ActivityGenerator) generateWorkActivity(session *ActivitySession, activityType string) {
	// Calculate work intensity based on accumulated events
	workIntensity := ag.calculateWorkIntensity(session)
	
	// Generate comprehensive metadata
	metadata := map[string]interface{}{
		"session_duration_seconds": time.Since(session.LastActivity).Seconds(),
		"process_events_count":     len(session.ProcessEvents),
		"file_io_events_count":     len(session.FileIOEvents),
		"work_file_count":          session.WorkFileCount,
		"work_intensity":           workIntensity,
		"last_process_pid":         session.LastProcessPID,
	}
	
	// Add file operation details
	if len(session.FileIOEvents) > 0 {
		fileTypes := make(map[string]int)
		var totalFileSize int64
		
		for _, fileEvent := range session.FileIOEvents {
			if fileEvent.IsWorkFile {
				fileTypes[fileEvent.FileType]++
				totalFileSize += fileEvent.Size
			}
		}
		
		metadata["file_types"] = fileTypes
		metadata["total_file_size"] = totalFileSize
	}
	
	// Add process details
	if len(session.ProcessEvents) > 0 {
		processNames := make(map[string]int)
		for _, procEvent := range session.ProcessEvents {
			processNames[procEvent.Command]++
		}
		metadata["process_names"] = processNames
	}
	
	activity := ActivityEvent{
		ID:             generateActivityID(),
		UserID:         ag.getUserFromSession(session),
		ProjectPath:    session.ProjectPath,
		ProjectName:    session.ProjectName,
		ActivityType:   activityType,
		ActivitySource: "enhanced_activity_monitor",
		Timestamp:      session.LastActivity,
		Command:        ag.getPrimaryCommandFromSession(session),
		Description:    ag.generateWorkDescription(session, workIntensity),
		Metadata:       metadata,
	}
	
	ag.sendActivity(activity)
	ag.stats.WorkActivities++
	
	if ag.config.VerboseLogging {
		log.Printf("🎯 Generated work activity: %s for %s (intensity: %.2f)", 
			activityType, session.ProjectName, workIntensity)
	}
}

/**
 * CONTEXT:   Calculate work intensity from session events
 * INPUT:     Activity session with accumulated events
 * OUTPUT:    Work intensity score (0.0 to 1.0)
 * BUSINESS:  Work intensity enables productivity assessment
 * CHANGE:    Initial work intensity calculation
 * RISK:      Medium - Work intensity calculation affecting productivity metrics
 */
func (ag *ActivityGenerator) calculateWorkIntensity(session *ActivitySession) float64 {
	intensity := 0.0
	
	// Base intensity from event count
	eventCount := float64(len(session.ProcessEvents) + len(session.FileIOEvents))
	intensity += (eventCount / 20.0) * 0.4 // Max 40% from event count
	
	// Intensity from work files
	workFileRatio := float64(session.WorkFileCount) / float64(len(session.FileIOEvents)+1)
	intensity += workFileRatio * 0.4 // Max 40% from work file ratio
	
	// Intensity from event diversity (process + file events)
	if len(session.ProcessEvents) > 0 && len(session.FileIOEvents) > 0 {
		intensity += 0.2 // 20% bonus for diverse activity
	}
	
	// Cap at 1.0
	if intensity > 1.0 {
		intensity = 1.0
	}
	
	return intensity
}

/**
 * CONTEXT:   Get user ID from session events
 * INPUT:     Activity session
 * OUTPUT:    User ID string
 * BUSINESS:  User identification enables per-user work tracking
 * CHANGE:    Initial user identification from session
 * RISK:      Low - User identification utility function
 */
func (ag *ActivityGenerator) getUserFromSession(session *ActivitySession) string {
	// Try to get user from process events
	for _, procEvent := range session.ProcessEvents {
		if procEvent.UserID != "" {
			return procEvent.UserID
		}
	}
	
	return "system" // Default fallback
}

/**
 * CONTEXT:   Get primary command from session events
 * INPUT:     Activity session
 * OUTPUT:    Primary command string
 * BUSINESS:  Command identification enables activity classification
 * CHANGE:    Initial command identification from session
 * RISK:      Low - Command identification utility function
 */
func (ag *ActivityGenerator) getPrimaryCommandFromSession(session *ActivitySession) string {
	if session.LastProcessPID > 0 {
		// Find the most recent process command
		for i := len(session.ProcessEvents) - 1; i >= 0; i-- {
			if session.ProcessEvents[i].PID == session.LastProcessPID {
				return session.ProcessEvents[i].Command
			}
		}
	}
	
	// Fallback to first process command
	if len(session.ProcessEvents) > 0 {
		return session.ProcessEvents[0].Command
	}
	
	// Fallback to file I/O process name
	if len(session.FileIOEvents) > 0 {
		return session.FileIOEvents[len(session.FileIOEvents)-1].ProcessName
	}
	
	return "claude"
}

/**
 * CONTEXT:   Generate work description from session data
 * INPUT:     Activity session and work intensity
 * OUTPUT:    Human-readable work description
 * BUSINESS:  Work description provides meaningful activity context
 * CHANGE:    Initial work description generation
 * RISK:      Low - Description generation utility function
 */
func (ag *ActivityGenerator) generateWorkDescription(session *ActivitySession, intensity float64) string {
	projectName := session.ProjectName
	if projectName == "" || projectName == "unknown" {
		projectName = "Claude work"
	}
	
	intensityLevel := "moderate"
	if intensity >= 0.8 {
		intensityLevel = "high"
	} else if intensity <= 0.4 {
		intensityLevel = "light"
	}
	
	fileCount := len(session.FileIOEvents)
	processCount := len(session.ProcessEvents)
	
	return fmt.Sprintf("%s activity in %s (%s intensity, %d file ops, %d process events)", 
		strings.Title(intensityLevel), projectName, intensityLevel, fileCount, processCount)
}

/**
 * CONTEXT:   Send activity event to callback
 * INPUT:     Activity event to send
 * OUTPUT:    Activity event sent to daemon via callback
 * BUSINESS:  Activity sending enables daemon integration
 * CHANGE:    Initial activity sending with statistics
 * RISK:      Medium - Activity sending affecting daemon integration
 */
func (ag *ActivityGenerator) sendActivity(activity ActivityEvent) {
	ag.stats.GeneratedActivities++
	ag.stats.LastActivityTime = time.Now()
	
	if ag.activityCallback != nil {
		go ag.activityCallback(activity)
	}
	
	if ag.config.VerboseLogging {
		activityJSON, _ := json.Marshal(activity)
		log.Printf("📤 Sent activity: %s", string(activityJSON))
	}
}

/**
 * CONTEXT:   Get activity generator statistics
 * INPUT:     Statistics request
 * OUTPUT:    Current activity generation metrics
 * BUSINESS:  Statistics enable activity generation optimization
 * CHANGE:    Initial statistics getter
 * RISK:      Low - Read-only statistics access
 */
func (ag *ActivityGenerator) GetStats() ActivityGeneratorStats {
	ag.mu.RLock()
	defer ag.mu.RUnlock()
	return ag.stats
}

/**
 * CONTEXT:   Clean up old activity sessions
 * INPUT:     Session cleanup request
 * OUTPUT:    Cleaned activity session buffer
 * BUSINESS:  Session cleanup prevents memory leaks
 * CHANGE:    Initial session cleanup implementation
 * RISK:      Low - Memory management utility function
 */
func (ag *ActivityGenerator) CleanupSessions() {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	
	cutoff := time.Now().Add(-ag.config.EventBufferDuration * 2)
	cleaned := 0
	
	for key, session := range ag.eventBuffer {
		if session.LastActivity.Before(cutoff) {
			delete(ag.eventBuffer, key)
			cleaned++
		}
	}
	
	if cleaned > 0 && ag.config.VerboseLogging {
		log.Printf("🧹 Cleaned up %d inactive activity sessions", cleaned)
	}
}

// Utility functions

func generateActivityID() string {
	return fmt.Sprintf("activity_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

func extractProjectNameFromPath(path string) string {
	if path == "" {
		return "unknown"
	}
	
	// Extract project name from path (last directory component)
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	
	return "unknown"
}