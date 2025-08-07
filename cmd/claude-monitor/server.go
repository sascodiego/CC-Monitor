/**
 * CONTEXT:   Embedded HTTP server for Claude Monitor single binary
 * INPUT:     HTTP requests for activity events and reporting endpoints
 * OUTPUT:    HTTP responses with work tracking data and health information
 * BUSINESS:  Embedded server enables daemon functionality without external dependencies
 * CHANGE:    Initial embedded server implementation with minimal dependencies
 * RISK:      Medium - HTTP server providing core daemon functionality
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/claude-monitor/system/internal/monitor"
	"github.com/claude-monitor/system/internal/database"
)

// FileMonitorInterface defines the common interface for file monitors
type FileMonitorInterface interface {
	Start() error
	Stop() error
	AttachToProcess(pid int) error
	DetachFromProcess(pid int) error
}

/**
 * CONTEXT:   Embedded server configuration for single binary operation
 * INPUT:     Server configuration parameters and application settings
 * OUTPUT:    Configured server instance with all necessary components
 * BUSINESS:  Server configuration provides customizable daemon behavior
 * CHANGE:    Initial server configuration with session and database settings
 * RISK:      Low - Configuration structure with validation
 */
type EmbeddedServerConfig struct {
	ListenAddr     string
	DatabasePath   string
	LogLevel       string
	DurationHours  int
	MaxIdleMinutes int
}

type EmbeddedServer struct {
	config           EmbeddedServerConfig
	router           *mux.Router
	server           *http.Server
	startTime        time.Time
	sessions         map[string]*Session
	workBlocks       map[string]*WorkBlock
	activities       []*ActivityEvent
	mu               sync.RWMutex
	dataFile         string
	db               *database.KuzuDBConnection
	healthMonitor    *ServiceHealthMonitor
	processMonitor   *monitor.ProcessMonitor
	httpMonitor      *monitor.HTTPMonitor
	fileIOMonitor    FileMonitorInterface
	activityGenerator *monitor.ActivityGenerator
}

/**
 * CONTEXT:   Create new embedded server instance with configuration
 * INPUT:     Server configuration and initialization parameters
 * OUTPUT:    Configured server ready for HTTP request handling
 * BUSINESS:  Server initialization sets up all components for daemon operation
 * CHANGE:    Initial server constructor with in-memory storage
 * RISK:      Medium - Server initialization affecting system functionality
 */
/**
 * CONTEXT:   Enhanced server constructor with KuzuDB integration
 * INPUT:     Server configuration with KuzuDB database path
 * OUTPUT:    Configured server with KuzuDB persistence
 * BUSINESS:  KuzuDB integration provides robust work tracking data persistence
 * CHANGE:    Migrated from JSON to KuzuDB with backward compatibility
 * RISK:      Medium - Database integration affecting data persistence reliability
 */
func NewEmbeddedServer(config EmbeddedServerConfig) (*EmbeddedServer, error) {
	server := &EmbeddedServer{
		config:        config,
		startTime:     time.Now(),
		sessions:      make(map[string]*Session),
		workBlocks:    make(map[string]*WorkBlock),
		activities:    make([]*ActivityEvent, 0),
		dataFile:      filepath.Join(filepath.Dir(config.DatabasePath), "data.json"),
		healthMonitor: nil,
	}

	// Initialize KuzuDB connection
	server.db = database.NewKuzuDBConnection(config.DatabasePath)
	if err := server.db.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize KuzuDB: %w", err)
	}
	log.Printf("📊 KuzuDB initialized successfully at: %s", config.DatabasePath)

	// Setup HTTP routes
	server.setupRoutes()

	// Load existing data from KuzuDB (NO JSON FALLBACK)
	if err := server.loadDataFromKuzu(); err != nil {
		log.Printf("Warning: failed to load existing data from KuzuDB: %v", err)
	}

	// Initialize activity generator with POST-focused configuration
	activityConfig := monitor.ActivityGeneratorConfig{
		EventBufferDuration:  5 * time.Second,  // Shorter buffer for POST requests
		ActivityTimeout:      15 * time.Second, // Quick response to POST requests
		PostRequestOnly:      true,             // Only generate activities from POST requests
		IdleTimeoutMinutes:   5,                // 5 minutes without POST = idle
		VerboseLogging:       true,
	}
	
	activityGenerator := monitor.NewActivityGenerator(server.handleGeneratedActivity, activityConfig)
	server.activityGenerator = activityGenerator
	
	// Initialize HTTP monitor for comprehensive HTTP traffic work detection
	httpConfig := monitor.HTTPMonitorConfig{
		MonitorPorts:     []int{443, 80, 8080, 3000, 8000}, // Common development ports
		UseTcpdump:       true,           // Enable tcpdump for real traffic monitoring
		UseSSMonitor:     true,           // Enable ss socket statistics
		UseProcNet:       true,           // Enable /proc/net monitoring (primary)
		VerboseLogging:   true,
		MethodInference:  true,           // Enable intelligent GET/POST inference
		TrackConnections: true,           // Enable connection lifecycle tracking
	}
	httpMonitor := monitor.NewHTTPMonitor(server.handleHTTPEvent, httpConfig)
	server.httpMonitor = httpMonitor
	
	// Initialize NON-INVASIVE File Monitor - no strace, no process interference
	fileIOMonitor := monitor.NewNonInvasiveFileMonitor(server.handleFileIOEvent)
	server.fileIOMonitor = fileIOMonitor
	
	// CRITICAL: Validate no strace processes are running (prevents zombie processes)
	if err := server.validateNoStraceProcesses(); err != nil {
		log.Printf("⚠️  STRACE VALIDATION WARNING: %v", err)
		log.Printf("🔧 RECOMMENDATION: Kill strace processes before starting daemon")
	}
	
	// Initialize process monitor with activity generation
	processMonitor := monitor.NewProcessMonitor(server.handleProcessEvent)
	processMonitor.SetFileIOCallback(server.activityGenerator.HandleFileIOEvent)
	server.processMonitor = processMonitor
	log.Printf("Process monitor with HTTP monitoring, File I/O monitoring, and activity generation initialized successfully")

	return server, nil
}

/**
 * CONTEXT:   Set health monitor for service monitoring integration
 * INPUT:     Service health monitor instance
 * OUTPUT:    Server configured with health monitoring
 * BUSINESS:  Health monitoring enables service reliability and troubleshooting
 * CHANGE:    Initial health monitor integration
 * RISK:      Low - Health monitor integration with optional functionality
 */
func (s *EmbeddedServer) SetHealthMonitor(monitor *ServiceHealthMonitor) {
	s.healthMonitor = monitor
}

/**
 * CONTEXT:   HTTP route setup for API endpoints
 * INPUT:     HTTP request routing configuration
 * OUTPUT:    Configured router with all necessary endpoints
 * BUSINESS:  API endpoints provide interface for activity tracking and reporting
 * CHANGE:    Initial route configuration with core endpoints
 * RISK:      Low - HTTP routing setup with standard patterns
 */
func (s *EmbeddedServer) setupRoutes() {
	s.router = mux.NewRouter()

	// Health and status endpoints
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/status", s.handleStatus).Methods("GET")

	// Activity endpoints
	s.router.HandleFunc("/activity", s.handleActivity).Methods("POST")
	s.router.HandleFunc("/activity/recent", s.handleRecentActivity).Methods("GET")

	// Reporting endpoints
	s.router.HandleFunc("/reports/daily", s.handleDailyReport).Methods("GET")
	s.router.HandleFunc("/reports/weekly", s.handleWeeklyReport).Methods("GET")
	s.router.HandleFunc("/reports/monthly", s.handleMonthlyReport).Methods("GET")

	// Session management endpoints
	s.router.HandleFunc("/sessions/pending", s.handlePendingSessions).Methods("GET")
	s.router.HandleFunc("/sessions/close-all", s.handleCloseAllSessions).Methods("POST")

	// Process monitoring endpoints
	s.router.HandleFunc("/monitor/stats", s.handleMonitorStats).Methods("GET")
	s.router.HandleFunc("/monitor/processes", s.handleTrackedProcesses).Methods("GET")
	s.router.HandleFunc("/monitor/file-io", s.handleFileIOStats).Methods("GET")
	s.router.HandleFunc("/monitor/activities", s.handleActivityStats).Methods("GET")

	// Database query endpoint
	s.router.HandleFunc("/db/query", s.handleDatabaseQuery).Methods("GET")

	// Static content and web interface (basic)
	s.router.HandleFunc("/", s.handleRoot).Methods("GET")
}

/**
 * CONTEXT:   Start HTTP server and begin listening for requests
 * INPUT:     Server configuration and HTTP listening address
 * OUTPUT:    Running HTTP server processing requests
 * BUSINESS:  Server startup enables daemon functionality for work tracking
 * CHANGE:    Initial server startup with graceful error handling
 * RISK:      High - Server startup affecting daemon availability
 */
func (s *EmbeddedServer) Start() error {
	log.Printf("Starting embedded server on %s", s.config.ListenAddr)

	// Start process monitor
	if s.processMonitor != nil {
		if err := s.processMonitor.Start(); err != nil {
			log.Printf("Warning: failed to start process monitor: %v", err)
			log.Printf("Continuing without process monitoring")
		} else {
			log.Printf("Process monitor started successfully")
		}
	}
	
	// Start HTTP monitor for POST request work detection
	if s.httpMonitor != nil {
		if err := s.httpMonitor.Start(); err != nil {
			log.Printf("Warning: failed to start HTTP monitor: %v", err)
			log.Printf("Continuing without HTTP POST monitoring")
		} else {
			log.Printf("HTTP monitor started successfully for POST request work detection")
		}
	}
	
	// Start File I/O monitor for real-time file activity detection
	if s.fileIOMonitor != nil {
		if err := s.fileIOMonitor.Start(); err != nil {
			log.Printf("Warning: failed to start File I/O monitor: %v", err)
			log.Printf("Continuing without File I/O monitoring")
		} else {
			log.Printf("File I/O monitor started successfully for real-time work detection")
		}
	}
	
	// Start activity generator cleanup routine
	if s.activityGenerator != nil {
		go func() {
			ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
			defer ticker.Stop()
			
			for {
				select {
				case <-ticker.C:
					s.activityGenerator.CleanupSessions()
				}
			}
		}()
		log.Printf("Activity generator cleanup routine started")
	}

	// Create HTTP server
	s.server = &http.Server{
		Addr:    s.config.ListenAddr,
		Handler: s.router,
	}

	return s.server.ListenAndServe()
}

/**
 * CONTEXT:   Graceful server shutdown with context timeout
 * INPUT:     Context with timeout for shutdown operations
 * OUTPUT:    Graceful shutdown or timeout error
 * BUSINESS:  Graceful shutdown ensures data integrity and service reliability
 * CHANGE:    Initial shutdown implementation with health monitor integration
 * RISK:      Medium - Shutdown affects service availability
 */
func (s *EmbeddedServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	
	// Stop process monitor
	if s.processMonitor != nil {
		if err := s.processMonitor.Stop(); err != nil {
			log.Printf("Warning: failed to stop process monitor: %v", err)
		}
	}
	
	// Stop HTTP monitor
	if s.httpMonitor != nil {
		if err := s.httpMonitor.Stop(); err != nil {
			log.Printf("Warning: failed to stop HTTP monitor: %v", err)
		}
	}
	
	// Stop file I/O monitor (CRITICAL FIX for zombie processes)
	if s.fileIOMonitor != nil {
		if err := s.fileIOMonitor.Stop(); err != nil {
			log.Printf("Warning: failed to stop file I/O monitor: %v", err)
		}
	}
	
	// Note: ActivityGenerator in monitor package doesn't have Stop method
	// This is handled by stopping the input monitors (HTTP, FileIO, Process)
	
	// Save data before shutdown to KuzuDB (NO JSON FALLBACK)
	if err := s.saveDataToKuzu(); err != nil {
		log.Printf("Warning: failed to save data to KuzuDB during shutdown: %v", err)
	}

	// Close KuzuDB connection
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Printf("Warning: failed to close KuzuDB connection: %v", err)
		}
	}
	
	// Close health monitor
	if s.healthMonitor != nil {
		if s.healthMonitor.logger != nil {
			s.healthMonitor.logger.Info("ServerShutdown", "Server shutting down gracefully", nil)
			s.healthMonitor.logger.Close()
		}
	}
	
	return s.server.Shutdown(ctx)
}

/**
 * CONTEXT:   Immediate server stop without graceful shutdown
 * INPUT:     Force stop request
 * OUTPUT:    Immediate server termination
 * BUSINESS:  Force stop enables emergency service termination
 * CHANGE:    Initial force stop implementation
 * RISK:      High - Force stop may cause data loss
 */
func (s *EmbeddedServer) Stop() error {
	if s.server == nil {
		return nil
	}
	
	return s.server.Close()
}

/**
 * CONTEXT:   Health check endpoint for system monitoring
 * INPUT:     HTTP GET request for health status
 * OUTPUT:    JSON response with server health and uptime information
 * BUSINESS:  Health endpoint enables monitoring and troubleshooting
 * CHANGE:    Initial health check with basic system information
 * RISK:      Low - Read-only health information endpoint
 */
func (s *EmbeddedServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	health := HealthStatus{
		Version:         Version,
		Uptime:          time.Since(s.startTime),
		ListenAddr:      s.config.ListenAddr,
		DatabasePath:    s.config.DatabasePath,
		DatabaseSize:    s.getDatabaseSize(),
		ActiveSessions:  len(s.sessions),
		TotalWorkBlocks: len(s.workBlocks),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

/**
 * CONTEXT:   Activity event processing endpoint
 * INPUT:     HTTP POST request with activity event JSON payload
 * OUTPUT:    HTTP response confirming activity processing
 * BUSINESS:  Activity endpoint is core input for work hour tracking
 * CHANGE:    Initial activity processing with session and work block management
 * RISK:      High - Core functionality affecting all work tracking
 */
func (s *EmbeddedServer) handleActivity(w http.ResponseWriter, r *http.Request) {
	var eventReq ActivityEventRequest
	if err := json.NewDecoder(r.Body).Decode(&eventReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Convert to activity event
	event := &ActivityEvent{
		ID:             generateEventID(),
		UserID:         eventReq.UserID,
		ProjectPath:    eventReq.ProjectPath,
		ProjectName:    eventReq.ProjectName,
		ActivityType:   eventReq.ActivityType,
		ActivitySource: eventReq.ActivitySource,
		Timestamp:      eventReq.Timestamp,
		Command:        eventReq.Command,
		Description:    eventReq.Description,
		Metadata:       eventReq.Metadata,
		CreatedAt:      time.Now(),
	}

	// Process activity event
	if err := s.processActivity(event); err != nil {
		log.Printf("Error processing activity: %v", err)
		http.Error(w, "Processing failed", http.StatusInternalServerError)
		return
	}

	// Save data periodically to KuzuDB
	go s.saveDataToKuzu()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "processed"})
}

/**
 * CONTEXT:   Activity event processing with session and work block logic
 * INPUT:     Activity event with timing and project information
 * OUTPUT:    Updated sessions and work blocks with new activity
 * BUSINESS:  Activity processing implements core work tracking business logic
 * CHANGE:    Initial processing with 5-hour sessions and 5-minute idle detection
 * RISK:      High - Core business logic affecting all work calculations
 */
func (s *EmbeddedServer) processActivity(event *ActivityEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create session (5-hour windows) - use single global user
	session := s.getOrCreateSession("claude_user", event.Timestamp)
	if session == nil {
		return fmt.Errorf("failed to create session")
	}

	// Get or create work block (5-minute idle detection)
	workBlock := s.getOrCreateWorkBlock(session.ID, event.ProjectName, event.ProjectPath, event.Timestamp)
	if workBlock == nil {
		return fmt.Errorf("failed to create work block")
	}

	// Set activity relationships
	event.SessionID = session.ID
	event.WorkBlockID = workBlock.ID

	// NO LONGER STORE INDIVIDUAL ACTIVITIES - only count operations
	// s.activities = append(s.activities, event)

	// Update session activity
	session.LastActivityTime = event.Timestamp
	session.ActivityCount++
	session.UpdatedAt = time.Now()

	// Update work block with activity counter
	workBlock.LastActivityTime = event.Timestamp
	workBlock.ActivityCount++
	workBlock.EndTime = event.Timestamp
	workBlock.DurationSeconds = int(workBlock.EndTime.Sub(workBlock.StartTime).Seconds())
	workBlock.DurationHours = float64(workBlock.DurationSeconds) / 3600.0
	workBlock.UpdatedAt = time.Now()

	// Update activity type counters in work block
	if workBlock.ActivityTypeCounters == nil {
		workBlock.ActivityTypeCounters = make(map[string]int)
	}
	workBlock.ActivityTypeCounters[event.ActivityType]++

	// Update database with work block activity counter
	if s.db != nil {
		if err := s.db.UpdateWorkBlockActivity(workBlock.ID, event.ActivityType); err != nil {
			log.Printf("Warning: failed to update work block activity in KuzuDB: %v", err)
		}
	}

	return nil
}

/**
 * CONTEXT:   Session management with 5-hour window logic
 * INPUT:     User ID and activity timestamp
 * OUTPUT:    Active session or new session if needed
 * BUSINESS:  Sessions implement 5-hour window business rule
 * CHANGE:    Initial session management with time-based creation
 * RISK:      Medium - Session logic affecting work time calculations
 */
func (s *EmbeddedServer) getOrCreateSession(userID string, timestamp time.Time) *Session {
	// Find active session for user
	for _, session := range s.sessions {
		if session.UserID == userID && session.IsActive {
			// Check if still within 5-hour window
			if timestamp.Sub(session.StartTime) <= time.Duration(s.config.DurationHours)*time.Hour {
				return session
			}
			// Session expired, mark inactive
			session.IsActive = false
			session.State = "expired"
		}
	}

	// Create new session
	session := NewSession(userID, timestamp)
	s.sessions[session.ID] = session
	return session
}

/**
 * CONTEXT:   Work block management with idle detection logic  
 * INPUT:     Session ID, project info, and activity timestamp
 * OUTPUT:    Active work block or new work block if idle timeout exceeded
 * BUSINESS:  Work blocks implement 5-minute idle detection business rule
 * CHANGE:    Initial work block management with idle timeout logic
 * RISK:      Medium - Work block logic affecting active time calculations
 */
func (s *EmbeddedServer) getOrCreateWorkBlock(sessionID, projectName, projectPath string, timestamp time.Time) *WorkBlock {
	// Find active work block for session and project
	for _, wb := range s.workBlocks {
		if wb.SessionID == sessionID && wb.ProjectName == projectName && wb.IsActive {
			// Check if activity is within 5-minute window
			if timestamp.Sub(wb.LastActivityTime) <= time.Duration(s.config.MaxIdleMinutes)*time.Minute {
				return wb
			}
			// Work block went idle, finalize it
			wb.IsActive = false
			wb.State = "idle"
			wb.EndTime = wb.LastActivityTime.Add(time.Duration(s.config.MaxIdleMinutes) * time.Minute)
		}
	}

	// Create new work block
	workBlock := NewWorkBlock(sessionID, projectName, projectPath, timestamp)
	s.workBlocks[workBlock.ID] = workBlock
	return workBlock
}

/**
 * CONTEXT:   Daily report generation endpoint
 * INPUT:     HTTP GET request with date parameter
 * OUTPUT:    JSON daily report with work summary and insights
 * BUSINESS:  Daily reports provide primary user interface for work tracking
 * CHANGE:    Initial daily report with project breakdown and statistics
 * RISK:      Low - Read-only reporting endpoint with data aggregation
 */
func (s *EmbeddedServer) handleDailyReport(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	targetDate := time.Now()
	if dateStr != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}
	}

	report := s.generateDailyReport(targetDate)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

/**
 * CONTEXT:   Enhanced daily report generation with fixed time calculations
 * INPUT:     Target date for daily report generation
 * OUTPUT:    Comprehensive daily report with proper time validation and schedule calculation
 * BUSINESS:  Daily reports provide primary user interface with accurate time analytics
 * CHANGE:    Fixed schedule duration calculations and time validation to prevent impossible displays
 * RISK:      Low - Improved time calculations prevent user confusion and mathematical errors
 */
func (s *EmbeddedServer) generateDailyReport(date time.Time) *DailyReport {
	// Work day starts at 5:00 AM and ends at 5:00 AM next day (America/Montevideo)
	montevideoTZ, _ := time.LoadLocation("America/Montevideo")
	startOfWorkDay := time.Date(date.Year(), date.Month(), date.Day(), 5, 0, 0, 0, montevideoTZ)
	endOfWorkDay := startOfWorkDay.Add(24 * time.Hour)

	// Try KuzuDB first for comprehensive reporting
	if s.db != nil {
		if kuzuReport, err := s.db.GetDailyReport(date); err == nil {
			// Convert KuzuDB report to server report format with proper time initialization
			serverReport := &DailyReport{
				Date:             kuzuReport.Date,
				StartTime:        time.Time{},     // Will be set to actual first activity or stay zero
				EndTime:          time.Time{},     // Will be set to actual last activity or stay zero
				TotalWorkHours:   kuzuReport.TotalWorkHours,
				TotalSessions:    kuzuReport.TotalSessions,
				TotalWorkBlocks:  kuzuReport.TotalWorkBlocks,
				ProjectBreakdown: make([]ProjectSummary, len(kuzuReport.ProjectBreakdown)),
				WorkBlocks:       make([]WorkBlockSummary, 0),
				Insights:         kuzuReport.Insights,
				ActivitySummary:  ActivitySummary{
					CommandCount: kuzuReport.ActivitySummary.CommandCount,
					EditCount:    kuzuReport.ActivitySummary.EditCount,
					QueryCount:   kuzuReport.ActivitySummary.QueryCount,
					OtherCount:   kuzuReport.ActivitySummary.OtherCount,
				},
			}
			
			// Convert project breakdown
			for i, pt := range kuzuReport.ProjectBreakdown {
				serverReport.ProjectBreakdown[i] = ProjectSummary{
					Name:       pt.Name,
					Hours:      pt.Hours,
					Percentage: pt.Percentage,
					WorkBlocks: pt.WorkBlocks,
				}
			}
			
			// Get work blocks from database for timeline (work day 5AM-5AM America/Montevideo)
			montevideoTZ, _ := time.LoadLocation("America/Montevideo")
			startOfWorkDay := time.Date(date.Year(), date.Month(), date.Day(), 5, 0, 0, 0, montevideoTZ)
			endOfWorkDay := startOfWorkDay.Add(24 * time.Hour)
			
			if workBlocksFromDB, dbErr := s.db.GetWorkBlocksWithCounters(); dbErr == nil {
				// Convert work blocks to summary format for timeline
				for _, wb := range workBlocksFromDB {
					if wb.StartTime.After(startOfWorkDay.Add(-time.Second)) && wb.StartTime.Before(endOfWorkDay.Add(time.Second)) {
						duration := time.Duration(wb.DurationSeconds) * time.Second
						if duration == 0 && !wb.EndTime.IsZero() {
							duration = wb.EndTime.Sub(wb.StartTime)
						}
						
						// Track earliest and latest times for actual work schedule (validate times)
						if !wb.StartTime.IsZero() && isValidTime(wb.StartTime) {
							if serverReport.StartTime.IsZero() || wb.StartTime.Before(serverReport.StartTime) {
								serverReport.StartTime = wb.StartTime
							}
						}
						if !wb.EndTime.IsZero() && isValidTime(wb.EndTime) {
							// Validate end time is after start time and reasonable (max 12 hours per work block)
							if wb.EndTime.After(wb.StartTime) && wb.EndTime.Sub(wb.StartTime) <= 12*time.Hour {
								if serverReport.EndTime.IsZero() || wb.EndTime.After(serverReport.EndTime) {
									serverReport.EndTime = wb.EndTime
								}
							}
						}
						
						serverReport.WorkBlocks = append(serverReport.WorkBlocks, WorkBlockSummary{
							StartTime:   wb.StartTime,
							EndTime:     wb.EndTime,
							Duration:    duration,
							ProjectName: wb.ProjectName,
							Activities:  int(wb.ActivityCount),
						})
					}
				}
			}
			
			log.Printf("📊 Generated daily report from KuzuDB: %.1f hours across %d projects, %d work blocks", 
				serverReport.TotalWorkHours, len(serverReport.ProjectBreakdown), len(serverReport.WorkBlocks))
			return serverReport
		} else {
			log.Printf("Warning: failed to get daily report from KuzuDB: %v", err)
		}
	}

	// Fallback to in-memory data
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &DailyReport{
		Date:             date,
		StartTime:        time.Time{},     // Will be set to actual first activity
		EndTime:          time.Time{},     // Will be set to actual last activity
		TotalWorkHours:   0,
		TotalSessions:    0,
		TotalWorkBlocks:  0,
		ProjectBreakdown: make([]ProjectSummary, 0),
		WorkBlocks:       make([]WorkBlockSummary, 0),
		Insights:         make([]string, 0),
		ActivitySummary:  ActivitySummary{},
	}

	projectHours := make(map[string]float64)
	projectBlocks := make(map[string]int)

	// Calculate work blocks for the work day (5AM to 5AM)
	for _, wb := range s.workBlocks {
		if wb.StartTime.After(startOfWorkDay) && wb.StartTime.Before(endOfWorkDay) {
			report.TotalWorkHours += wb.DurationHours
			projectHours[wb.ProjectName] += wb.DurationHours
			projectBlocks[wb.ProjectName]++
			report.TotalWorkBlocks++

			// Track earliest and latest times for actual work schedule (validate times)
			if !wb.StartTime.IsZero() && isValidTime(wb.StartTime) {
				if report.StartTime.IsZero() || wb.StartTime.Before(report.StartTime) {
					report.StartTime = wb.StartTime
				}
			}
			if !wb.EndTime.IsZero() && isValidTime(wb.EndTime) {
				// Validate end time is after start time and reasonable (max 12 hours per work block)
				if wb.EndTime.After(wb.StartTime) && wb.EndTime.Sub(wb.StartTime) <= 12*time.Hour {
					if report.EndTime.IsZero() || wb.EndTime.After(report.EndTime) {
						report.EndTime = wb.EndTime
					}
				}
			}
			
			// Add work block to timeline
			duration := time.Duration(wb.DurationHours * float64(time.Hour))
			if duration == 0 && !wb.EndTime.IsZero() {
				duration = wb.EndTime.Sub(wb.StartTime)
			}
			
			report.WorkBlocks = append(report.WorkBlocks, WorkBlockSummary{
				StartTime:   wb.StartTime,
				EndTime:     wb.EndTime,
				Duration:    duration,
				ProjectName: wb.ProjectName,
				Activities:  wb.ActivityCount,
			})
		}
	}

	// Generate project breakdown
	for projectName, hours := range projectHours {
		percentage := (hours / report.TotalWorkHours) * 100
		report.ProjectBreakdown = append(report.ProjectBreakdown, ProjectSummary{
			Name:       projectName,
			Hours:      hours,
			Percentage: percentage,
			WorkBlocks: projectBlocks[projectName],
		})
	}

	// Generate insights with proper schedule calculation
	if report.TotalWorkHours > 0 {
		var scheduleDuration time.Duration
		var efficiency float64
		
		// Calculate schedule duration only if we have valid start and end times
		if !report.StartTime.IsZero() && !report.EndTime.IsZero() && report.EndTime.After(report.StartTime) {
			scheduleDuration = report.EndTime.Sub(report.StartTime)
			// Validate schedule duration is reasonable (max 18 hours in a work day)
			if scheduleDuration > 0 && scheduleDuration <= 18*time.Hour {
				scheduleHours := scheduleDuration.Hours()
				if scheduleHours > 0 {
					efficiency = (report.TotalWorkHours / scheduleHours) * 100
				}
			} else {
				// Schedule duration is invalid, treat efficiency as 100% (active work only)
				efficiency = 100.0
			}
		} else {
			// No valid schedule times, assume 100% efficiency for active work
			efficiency = 100.0
		}
		
		if efficiency >= 80 {
			report.Insights = append(report.Insights, "Excellent focus! High efficiency today")
		} else if efficiency >= 60 {
			report.Insights = append(report.Insights, "Great productivity with good work focus")
		} else {
			report.Insights = append(report.Insights, "Consider longer focused work sessions")
		}

		if len(report.ProjectBreakdown) == 1 {
			report.Insights = append(report.Insights, "Single project focus - great for deep work")
		} else if len(report.ProjectBreakdown) > 3 {
			report.Insights = append(report.Insights, "Multiple project context switching")
		}
	}

	log.Printf("📊 Generated daily report from memory: %.1f hours across %d projects", 
		report.TotalWorkHours, len(report.ProjectBreakdown))
	return report
}

// Additional helper methods for server operation

func (s *EmbeddedServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Redirect to health endpoint
	s.handleHealth(w, r)
}

func (s *EmbeddedServer) handleRecentActivity(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 5
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	activities := make([]RecentActivity, 0, limit)
	count := 0
	for i := len(s.activities) - 1; i >= 0 && count < limit; i-- {
		act := s.activities[i]
		activities = append(activities, RecentActivity{
			Timestamp:   act.Timestamp,
			EventType:   act.ActivityType,
			ProjectName: act.ProjectName,
		})
		count++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activities)
}

func (s *EmbeddedServer) handleWeeklyReport(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	targetDate := time.Now()
	if dateStr != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}
	}

	report := s.generateWeeklyReport(targetDate)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

/**
 * CONTEXT:   Generate weekly report with 5AM work day boundaries
 * INPUT:     Target week start date
 * OUTPUT:    WeeklyReport with daily breakdowns using correct work day logic
 * BUSINESS:  Weekly reports aggregate daily reports respecting 5AM-5AM work day boundaries
 * CHANGE:    Initial implementation replacing placeholder with real weekly logic
 * RISK:      Low - Aggregates existing daily report logic
 */
func (s *EmbeddedServer) generateWeeklyReport(currentDate time.Time) *WeeklyReport {
	// Calculate week start from current date with 5AM work day logic
	montevideoTZ, _ := time.LoadLocation("America/Montevideo")
	
	// If current time is before 5AM, adjust to previous day for work day calculation
	adjustedDate := currentDate
	if currentDate.Hour() < 5 {
		adjustedDate = currentDate.AddDate(0, 0, -1)
	}
	
	// Find Monday of the week
	daysSinceMonday := int(adjustedDate.Weekday()) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6 // Sunday
	}
	
	mondayStart := adjustedDate.AddDate(0, 0, -daysSinceMonday)
	weekStartWorkDay := time.Date(mondayStart.Year(), mondayStart.Month(), mondayStart.Day(), 5, 0, 0, 0, montevideoTZ)
	
	// Aggregate data from 7 daily reports with better logging
	totalHours := 0.0
	dailyReports := make([]*DailyReport, 7)
	
	for i := 0; i < 7; i++ {
		dayDate := weekStartWorkDay.AddDate(0, 0, i)
		dailyReport := s.generateDailyReport(dayDate)
		dailyReports[i] = dailyReport
		
		if dailyReport.TotalWorkHours > 0 {
			log.Printf("📊 Week aggregation: %s has %.1f work hours", 
				dayDate.Format("2006-01-02"), dailyReport.TotalWorkHours)
		}
		
		totalHours += dailyReport.TotalWorkHours
	}
	
	log.Printf("📊 Weekly report total: %.1f hours across 7 days", totalHours)
	
	// Build trends (day-by-day comparison)
	trends := make([]Trend, 0)
	for i, daily := range dailyReports {
		if daily.TotalWorkHours > 0 {
			dayName := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}[i]
			trends = append(trends, Trend{
				Metric:    "Hours",
				Direction: "stable", // Could calculate direction by comparing with previous day
				Change:    daily.TotalWorkHours,
				Period:    dayName,
				Icon:      "⏰",
			})
		}
	}
	
	// Build daily breakdown summaries
	dailyBreakdown := make([]DaySummary, 0)
	projectSummaries := make(map[string]*ProjectSummary)
	
	for i, daily := range dailyReports {
		if daily.TotalWorkHours > 0 {
			dayName := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}[i]
			dailyBreakdown = append(dailyBreakdown, DaySummary{
				Date:           daily.Date,
				DayName:        dayName,
				Hours:          daily.TotalWorkHours,
				ClaudeSessions: daily.TotalSessions,
				WorkBlocks:     daily.TotalWorkBlocks,
			})
			
			// Aggregate project summaries
			for _, project := range daily.ProjectBreakdown {
				if existing, ok := projectSummaries[project.Name]; ok {
					existing.Hours += project.Hours
					existing.WorkBlocks += project.WorkBlocks
				} else {
					projectSummaries[project.Name] = &ProjectSummary{
						Name:       project.Name,
						Hours:      project.Hours,
						Percentage: 0, // Will calculate after
						WorkBlocks: project.WorkBlocks,
					}
				}
			}
		}
	}
	
	// Calculate project percentages
	projectSummariesSlice := make([]ProjectSummary, 0)
	for _, project := range projectSummaries {
		if totalHours > 0 {
			project.Percentage = (project.Hours / totalHours) * 100
		}
		projectSummariesSlice = append(projectSummariesSlice, *project)
	}
	
	return &WeeklyReport{
		Week:             weekStartWorkDay,
		TotalHours:       totalHours,
		DailyBreakdown:   dailyBreakdown,
		ProjectSummaries: projectSummariesSlice,
		Trends:           trends,
	}
}

/**
 * CONTEXT:   Generate comprehensive monthly report with aggregated data
 * INPUT:     Target month for monthly report generation
 * OUTPUT:    MonthlyReport with complete monthly analytics and comparisons
 * BUSINESS:  Monthly reports show long-term productivity patterns and month-over-month trends
 * CHANGE:    Implemented real monthly report generation with daily aggregation
 * RISK:      Medium - Monthly aggregation requires processing multiple daily reports
 */
func (s *EmbeddedServer) generateMonthlyReport(month time.Time) *MonthlyReport {
	// Normalize to first day of month with work day logic (5AM America/Montevideo)
	montevideoTZ, _ := time.LoadLocation("America/Montevideo")
	monthStart := time.Date(month.Year(), month.Month(), 1, 5, 0, 0, 0, montevideoTZ)
	
	// Get last day of month
	nextMonth := monthStart.AddDate(0, 1, 0)
	
	// Aggregate data from daily reports for the entire month
	totalHours := 0.0
	daysWithWork := 0
	projectTotals := make(map[string]float64)
	dailyReports := make([]*DailyReport, 0)
	
	// Process each day of the month with better logging
	for current := monthStart; current.Before(nextMonth); current = current.Add(24 * time.Hour) {
		dailyReport := s.generateDailyReport(current)
		dailyReports = append(dailyReports, dailyReport)
		
		if dailyReport.TotalWorkHours > 0 {
			log.Printf("📊 Month aggregation: %s has %.1f work hours", 
				current.Format("2006-01-02"), dailyReport.TotalWorkHours)
			
			totalHours += dailyReport.TotalWorkHours
			daysWithWork++
			
			// Aggregate project data
			for _, project := range dailyReport.ProjectBreakdown {
				projectTotals[project.Name] += project.Hours
			}
		}
	}
	
	log.Printf("📊 Monthly report total: %.1f hours across %d working days", totalHours, daysWithWork)
	
	// Calculate averages
	dailyAverage := 0.0
	if daysWithWork > 0 {
		dailyAverage = totalHours / float64(daysWithWork)
	}
	
	// Create project comparisons
	comparisons := make([]Comparison, 0)
	for projectName, hours := range projectTotals {
		percentage := 0.0
		if totalHours > 0 {
			percentage = (hours / totalHours) * 100
		}
		
		comparisons = append(comparisons, Comparison{
			Period:    "Project: " + projectName + " in " + month.Format("January 2006"),
			Change:    percentage,
		})
	}
	
	// Add total hours comparison
	comparisons = append(comparisons, Comparison{
		Period:    "Total Work Hours in " + month.Format("January 2006"),
		Change:    totalHours,
	})
	
	// Add working days comparison
	comparisons = append(comparisons, Comparison{
		Period:    "Working Days in " + month.Format("January 2006"),
		Change:    float64(daysWithWork),
	})
	
	log.Printf("📅 Generated monthly report for %s: %.1f hours across %d working days (%.1f hours/day average)", 
		month.Format("January 2006"), totalHours, daysWithWork, dailyAverage)
	
	return &MonthlyReport{
		Month:       monthStart,
		TotalHours:  totalHours,
		Comparisons: comparisons,
	}
}

/**
 * CONTEXT:   Handle monthly report HTTP endpoint with real data aggregation
 * INPUT:     HTTP GET request with optional month parameter
 * OUTPUT:    JSON monthly report with comprehensive monthly analytics
 * BUSINESS:  Monthly reports provide long-term productivity insights and trends
 * CHANGE:    Implemented real monthly report generation replacing placeholder
 * RISK:      Medium - Monthly aggregation requires careful data processing
 */
func (s *EmbeddedServer) handleMonthlyReport(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	targetMonth := time.Now()
	if monthStr != "" {
		var err error
		targetMonth, err = time.Parse("2006-01", monthStr)
		if err != nil {
			http.Error(w, "Invalid month format", http.StatusBadRequest)
			return
		}
	}

	report := s.generateMonthlyReport(targetMonth)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (s *EmbeddedServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	// Simple web interface
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Claude Monitor Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        .section { margin: 20px 0; }
        .endpoint { margin: 10px 0; }
        code { background: #f5f5f5; padding: 2px 4px; }
    </style>
</head>
<body>
    <h1>🎯 Claude Monitor Dashboard</h1>
    <p>Server started: %s</p>
    <p>Uptime: %v</p>
    
    <div class="section">
        <h2>API Endpoints</h2>
        <div class="endpoint">
            <strong>Health:</strong> <code>GET <a href="/health">/health</a></code>
        </div>
        <div class="endpoint">
            <strong>Daily Report:</strong> <code>GET <a href="/reports/daily">/reports/daily</a></code>
        </div>
        <div class="endpoint">
            <strong>Recent Activity:</strong> <code>GET <a href="/activity/recent">/activity/recent</a></code>
        </div>
    </div>
    
    <div class="section">
        <h2>CLI Commands</h2>
        <p><code>claude-monitor today</code> - Show today's work summary</p>
        <p><code>claude-monitor week</code> - Show weekly analytics</p>
        <p><code>claude-monitor status</code> - Check system health</p>
    </div>
</body>
</html>
`, s.startTime.Format("2006-01-02 15:04:05"), time.Since(s.startTime).Round(time.Second))

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// KuzuDB Data persistence methods

/**
 * CONTEXT:   Save server data to KuzuDB with session, work block, and activity persistence
 * INPUT:     Current server state (sessions, work blocks, activities)
 * OUTPUT:    Data persisted to KuzuDB database
 * BUSINESS:  KuzuDB persistence provides robust work tracking data storage
 * CHANGE:    New KuzuDB persistence replacing JSON-based storage
 * RISK:      High - Database operations affecting data integrity and persistence
 */
func (s *EmbeddedServer) saveDataToKuzu() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.db == nil {
		return fmt.Errorf("KuzuDB connection not available")
	}

	// Save sessions to KuzuDB
	for _, session := range s.sessions {
		kuzu_session := &database.Session{
			ID:                session.ID,
			UserID:            session.UserID,
			StartTime:         session.StartTime,
			EndTime:           session.EndTime,
			State:             session.State,
			FirstActivityTime: session.FirstActivityTime,
			LastActivityTime:  session.LastActivityTime,
			ActivityCount:     int64(session.ActivityCount),
			DurationHours:     session.DurationHours,
			IsActive:          session.IsActive,
			CreatedAt:         session.CreatedAt,
			UpdatedAt:         session.UpdatedAt,
		}
		
		if err := s.db.SaveSession(kuzu_session); err != nil {
			log.Printf("Warning: failed to save session %s to KuzuDB: %v", session.ID, err)
		}
	}

	// Save work blocks to KuzuDB
	for _, workBlock := range s.workBlocks {
		kuzu_workblock := &database.WorkBlock{
			ID:              workBlock.ID,
			SessionID:       workBlock.SessionID,
			ProjectName:     workBlock.ProjectName,
			ProjectPath:     workBlock.ProjectPath,
			StartTime:       workBlock.StartTime,
			EndTime:         workBlock.EndTime,
			State:           workBlock.State,
			LastActivityTime: workBlock.LastActivityTime,
			ActivityCount:   int64(workBlock.ActivityCount),
			DurationSeconds: int64(workBlock.DurationSeconds),
			DurationHours:   workBlock.DurationHours,
			IsActive:        workBlock.IsActive,
			CreatedAt:       workBlock.CreatedAt,
			UpdatedAt:       workBlock.UpdatedAt,
		}
		
		if err := s.db.SaveWorkBlock(kuzu_workblock); err != nil {
			log.Printf("Warning: failed to save work block %s to KuzuDB: %v", workBlock.ID, err)
		}
	}

	// Save activities to KuzuDB (last 100 for efficiency)
	activityCount := len(s.activities)
	startIndex := 0
	if activityCount > 100 {
		startIndex = activityCount - 100
	}
	
	for i := startIndex; i < activityCount; i++ {
		activity := s.activities[i]
		
		// Convert metadata map[string]string to string
		metadataJson := ""
		if len(activity.Metadata) > 0 {
			if jsonData, err := json.Marshal(activity.Metadata); err == nil {
				metadataJson = string(jsonData)
			}
		}
		
		kuzu_activity := &database.ActivityEvent{
			ID:             activity.ID,
			UserID:         activity.UserID,
			SessionID:      activity.SessionID,
			WorkBlockID:    activity.WorkBlockID,
			ProjectPath:    activity.ProjectPath,
			ProjectName:    activity.ProjectName,
			ActivityType:   activity.ActivityType,
			ActivitySource: activity.ActivitySource,
			Timestamp:      activity.Timestamp,
			Command:        activity.Command,
			Description:    activity.Description,
			Metadata:       metadataJson,
			CreatedAt:      activity.CreatedAt,
		}
		
		if err := s.db.SaveActivity(kuzu_activity); err != nil {
			log.Printf("Warning: failed to save activity %s to KuzuDB: %v", activity.ID, err)
		}
	}

	return nil
}

/**
 * CONTEXT:   Load server data from KuzuDB with session, work block, and activity loading
 * INPUT:     KuzuDB database connection and data queries
 * OUTPUT:    Server state populated from KuzuDB data
 * BUSINESS:  KuzuDB loading provides persistent work tracking data recovery
 * CHANGE:    New KuzuDB loading replacing JSON-based loading
 * RISK:      Medium - Database operations affecting server state initialization
 */
func (s *EmbeddedServer) loadDataFromKuzu() error {
	if s.db == nil {
		return fmt.Errorf("KuzuDB connection not available")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Load active sessions from KuzuDB
	sessions, err := s.db.GetActiveSessions()
	if err != nil {
		log.Printf("Warning: failed to load sessions from KuzuDB: %v", err)
	} else {
		for _, session := range sessions {
			s.sessions[session.ID] = &Session{
				ID:                session.ID,
				UserID:            session.UserID,
				StartTime:         session.StartTime,
				EndTime:           session.EndTime,
				State:             session.State,
				FirstActivityTime: session.FirstActivityTime,
				LastActivityTime:  session.LastActivityTime,
				ActivityCount:     int(session.ActivityCount),
				DurationHours:     session.DurationHours,
				IsActive:          session.IsActive,
				CreatedAt:         session.CreatedAt,
				UpdatedAt:         session.UpdatedAt,
			}
		}
		log.Printf("📊 Loaded %d active sessions from KuzuDB", len(sessions))
	}

	// Note: Work blocks and activities will be loaded on-demand through reporting queries
	// This keeps memory usage efficient while maintaining data persistence
	
	return nil
}

// Legacy JSON Data persistence methods (for backward compatibility)
func (s *EmbeddedServer) saveData() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := struct {
		Sessions   map[string]*Session   `json:"sessions"`
		WorkBlocks map[string]*WorkBlock `json:"work_blocks"`
		Activities []*ActivityEvent      `json:"activities"`
		Timestamp  time.Time             `json:"timestamp"`
	}{
		Sessions:   s.sessions,
		WorkBlocks: s.workBlocks,
		Activities: s.activities,
		Timestamp:  time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.dataFile, jsonData, 0644)
}

func (s *EmbeddedServer) loadData() error {
	if _, err := os.Stat(s.dataFile); os.IsNotExist(err) {
		return nil // No data file exists yet
	}

	jsonData, err := os.ReadFile(s.dataFile)
	if err != nil {
		return err
	}

	var data struct {
		Sessions   map[string]*Session   `json:"sessions"`
		WorkBlocks map[string]*WorkBlock `json:"work_blocks"`
		Activities []*ActivityEvent      `json:"activities"`
		Timestamp  time.Time             `json:"timestamp"`
	}

	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if data.Sessions != nil {
		s.sessions = data.Sessions
	}
	if data.WorkBlocks != nil {
		s.workBlocks = data.WorkBlocks
	}
	if data.Activities != nil {
		s.activities = data.Activities
	}

	return nil
}

func (s *EmbeddedServer) getDatabaseSize() string {
	if info, err := os.Stat(s.dataFile); err == nil {
		if info.Size() < 1024 {
			return fmt.Sprintf("%d bytes", info.Size())
		} else if info.Size() < 1024*1024 {
			return fmt.Sprintf("%.1f KB", float64(info.Size())/1024)
		} else {
			return fmt.Sprintf("%.1f MB", float64(info.Size())/(1024*1024))
		}
	}
	return "0 bytes"
}

/**
 * CONTEXT:   Get pending/active sessions endpoint for close-all-hooks command
 * INPUT:     HTTP GET request for pending session information
 * OUTPUT:    JSON array of pending sessions with details
 * BUSINESS:  Provides data for cleanup commands to identify orphaned sessions
 * CHANGE:    Initial implementation for session cleanup functionality
 * RISK:      Low - Read-only endpoint providing session status information
 */
func (s *EmbeddedServer) handlePendingSessions(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pendingSessions := make([]PendingSessionSummary, 0)
	
	for _, session := range s.sessions {
		if session.IsActive {
			// Count active work blocks for this session
			activeWorkBlocks := 0
			for _, wb := range s.workBlocks {
				if wb.SessionID == session.ID && wb.IsActive {
					activeWorkBlocks++
				}
			}
			
			summary := PendingSessionSummary{
				ID:               session.ID,
				StartTime:        session.StartTime,
				ProjectName:      s.getPrimaryProjectForSession(session.ID),
				ProjectPath:      s.getPrimaryProjectPathForSession(session.ID),
				ActiveWorkBlocks: activeWorkBlocks,
				LastActivity:     session.LastActivityTime,
				UserID:           session.UserID,
			}
			
			pendingSessions = append(pendingSessions, summary)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pendingSessions)
}

/**
 * CONTEXT:   Close all pending sessions endpoint for cleanup operations
 * INPUT:     HTTP POST request to close all active sessions
 * OUTPUT:    JSON response with summary of closed sessions and work blocks
 * BUSINESS:  Enables cleanup of orphaned sessions and work blocks
 * CHANGE:    Initial implementation with comprehensive session closure
 * RISK:      Medium - Modifies session state, affects work time calculations
 */
func (s *EmbeddedServer) handleCloseAllSessions(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	closeTime := time.Now()
	closedSessions := 0
	closedWorkBlocks := 0
	totalWorkTime := time.Duration(0)
	errors := 0

	// Close all active sessions
	for _, session := range s.sessions {
		if session.IsActive {
			session.IsActive = false
			session.State = "closed_by_cleanup"
			session.UpdatedAt = closeTime
			closedSessions++

			// Calculate session work time
			sessionDuration := session.LastActivityTime.Sub(session.StartTime)
			if sessionDuration > 0 {
				totalWorkTime += sessionDuration
			}
		}
	}

	// Close all active work blocks
	for _, workBlock := range s.workBlocks {
		if workBlock.IsActive {
			workBlock.IsActive = false
			workBlock.State = "closed_by_cleanup"
			workBlock.EndTime = workBlock.LastActivityTime
			workBlock.UpdatedAt = closeTime
			
			// Update duration calculations
			duration := workBlock.EndTime.Sub(workBlock.StartTime)
			workBlock.DurationSeconds = int(duration.Seconds())
			workBlock.DurationHours = duration.Hours()
			
			closedWorkBlocks++
		}
	}

	// Save data after changes to KuzuDB (NO JSON FALLBACK)
	if err := s.saveDataToKuzu(); err != nil {
		log.Printf("Error saving data to KuzuDB after closing sessions: %v", err)
		errors++
	}

	result := CloseSessionsResult{
		ClosedSessions:   closedSessions,
		ClosedWorkBlocks: closedWorkBlocks,
		TotalWorkTime:    totalWorkTime,
		Errors:           errors,
		ClosedAt:         closeTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
	
	log.Printf("Closed %d sessions and %d work blocks via close-all-hooks command", 
		closedSessions, closedWorkBlocks)
}

/**
 * CONTEXT:   Get primary project name for a session (most recent work block)
 * INPUT:     Session ID for project lookup
 * OUTPUT:    Primary project name or "Multiple Projects" if mixed
 * BUSINESS:  Provides meaningful project identification for session summaries
 * CHANGE:    Initial implementation finding most recent project activity
 * RISK:      Low - Helper function for session display information
 */
func (s *EmbeddedServer) getPrimaryProjectForSession(sessionID string) string {
	var latestProject string
	var latestTime time.Time
	
	projectCount := make(map[string]int)
	
	for _, wb := range s.workBlocks {
		if wb.SessionID == sessionID {
			projectCount[wb.ProjectName]++
			
			if wb.LastActivityTime.After(latestTime) {
				latestTime = wb.LastActivityTime
				latestProject = wb.ProjectName
			}
		}
	}
	
	if len(projectCount) > 1 {
		return fmt.Sprintf("%s (+%d others)", latestProject, len(projectCount)-1)
	}
	
	if latestProject == "" {
		return "Unknown Project"
	}
	
	return latestProject
}

/**
 * CONTEXT:   Get primary project path for a session (most recent work block)
 * INPUT:     Session ID for project path lookup
 * OUTPUT:    Primary project path or empty string
 * BUSINESS:  Provides project path information for session identification
 * CHANGE:    Initial implementation finding most recent project path
 * RISK:      Low - Helper function for session path information
 */
func (s *EmbeddedServer) getPrimaryProjectPathForSession(sessionID string) string {
	var latestPath string
	var latestTime time.Time
	
	for _, wb := range s.workBlocks {
		if wb.SessionID == sessionID {
			if wb.LastActivityTime.After(latestTime) {
				latestTime = wb.LastActivityTime
				latestPath = wb.ProjectPath
			}
		}
	}
	
	return latestPath
}

/**
 * CONTEXT:   Handle process lifecycle events for monitoring only
 * INPUT:     Process event with start/stop information and process details
 * OUTPUT:    Informational logging and activity generator notification
 * BUSINESS:  Process events provide visibility into Claude instances without affecting work tracking
 * CHANGE:    Enhanced with activity generator integration for comprehensive monitoring
 * RISK:      Low - Informational logging with activity correlation
 */
func (s *EmbeddedServer) handleProcessEvent(event monitor.ProcessEvent) {
	// Send to activity generator for correlation
	if s.activityGenerator != nil {
		s.activityGenerator.HandleProcessEvent(event)
	}
	
	// Update HTTP monitor with tracked processes for POST detection
	if s.httpMonitor != nil && s.processMonitor != nil {
		processes := s.processMonitor.GetTrackedProcesses()
		s.httpMonitor.UpdateTrackedProcesses(processes)
	}
	
	switch event.Type {
	case monitor.ProcessStarted:
		s.logProcessStarted(event)
		s.attachMonitorsToProcess(event)
	case monitor.ProcessStopped:
		s.logProcessStopped(event)
		s.detachMonitorsFromProcess(event)
	default:
		log.Printf("Unknown process event type: %s", event.Type)
	}
}

/**
 * CONTEXT:   Attach monitoring (File I/O, HTTP) to new Claude process
 * INPUT:     Process started event with Claude process information
 * OUTPUT:    Dynamic monitoring attachment for real-time activity detection
 * BUSINESS:  Dynamic attachment enables precise per-process work tracking
 * CHANGE:    Event-driven process monitoring attachment
 * RISK:      High - Dynamic monitoring attachment affecting system performance
 */
func (s *EmbeddedServer) attachMonitorsToProcess(event monitor.ProcessEvent) {
	// Skip our own daemon process
	if strings.Contains(event.Command, "claude-monitor") {
		return
	}
	
	// Skip non-Claude processes (but allow related processes like npm, git, etc.)
	if !s.isClaudeRelatedProcess(event.Command) {
		if s.config.LogLevel == "debug" {
			log.Printf("Skipping monitoring attachment for non-Claude process: %s", event.Command)
		}
		return
	}
	
	// Attach File I/O monitoring to the new process
	if s.fileIOMonitor != nil {
		if err := s.fileIOMonitor.AttachToProcess(event.PID); err != nil {
			log.Printf("Warning: failed to attach File I/O monitor to PID %d: %v", event.PID, err)
		} else {
			log.Printf("🔗 Attached File I/O monitoring to %s (PID: %d)", event.Command, event.PID)
		}
	}
	
	// Attach HTTP monitoring to the new process (for tcpdump filtering)
	if s.httpMonitor != nil {
		if err := s.httpMonitor.AttachToProcess(event.PID); err != nil {
			log.Printf("Warning: failed to attach HTTP monitor to PID %d: %v", event.PID, err)
		} else {
			log.Printf("🔗 Attached HTTP monitoring to %s (PID: %d)", event.Command, event.PID)
		}
	}
}

/**
 * CONTEXT:   Detach monitoring from stopped Claude process
 * INPUT:     Process stopped event with Claude process information
 * OUTPUT:    Dynamic monitoring detachment and resource cleanup
 * BUSINESS:  Dynamic detachment prevents resource leaks and stale monitoring
 * CHANGE:    Event-driven process monitoring detachment
 * RISK:      Medium - Dynamic monitoring detachment affecting resource cleanup
 */
func (s *EmbeddedServer) detachMonitorsFromProcess(event monitor.ProcessEvent) {
	// Skip our own daemon process
	if strings.Contains(event.Command, "claude-monitor") {
		return
	}
	
	// Skip non-Claude processes
	if !s.isClaudeRelatedProcess(event.Command) {
		return
	}
	
	// Detach File I/O monitoring from the stopped process
	if s.fileIOMonitor != nil {
		if err := s.fileIOMonitor.DetachFromProcess(event.PID); err != nil {
			log.Printf("Warning: failed to detach File I/O monitor from PID %d: %v", event.PID, err)
		} else {
			log.Printf("🔓 Detached File I/O monitoring from %s (PID: %d)", event.Command, event.PID)
		}
	}
	
	// Detach HTTP monitoring from the stopped process
	if s.httpMonitor != nil {
		if err := s.httpMonitor.DetachFromProcess(event.PID); err != nil {
			log.Printf("Warning: failed to detach HTTP monitor from PID %d: %v", event.PID, err)
		} else {
			log.Printf("🔓 Detached HTTP monitoring from %s (PID: %d)", event.Command, event.PID)
		}
	}
}

/**
 * CONTEXT:   Check if process is Claude-related for monitoring
 * INPUT:     Process command name
 * OUTPUT:    Boolean indicating if process should be monitored
 * BUSINESS:  Process filtering enables focused monitoring on relevant processes
 * CHANGE:    Initial Claude-related process detection
 * RISK:      Low - Process filtering logic
 */
func (s *EmbeddedServer) isClaudeRelatedProcess(command string) bool {
	claudeProcesses := []string{
		"claude",      // Main Claude binary
		"node",        // Node.js (Claude Code runs on Node)
		"npm",         // npm commands (often used with Claude)
		"git",         // Git commands (often used during development)
		"python",      // Python interpreters
		"go",          // Go commands
		"code",        // VS Code
		"vim",         // Vim editor
		"nvim",        // Neovim
		"emacs",       // Emacs editor
	}
	
	for _, process := range claudeProcesses {
		if strings.Contains(command, process) {
			return true
		}
	}
	
	return false
}

/**
 * CONTEXT:   Check if HTTP traffic is work-related for monitoring
 * INPUT:     HTTP host and URL
 * OUTPUT:    Boolean indicating if HTTP traffic should be tracked as work
 * BUSINESS:  HTTP filtering enables focused work activity detection
 * CHANGE:    Initial work-related HTTP traffic detection
 * RISK:      Low - HTTP traffic filtering logic
 */
func (s *EmbeddedServer) isWorkRelatedHTTPTraffic(host, url string) bool {
	workRelatedHosts := []string{
		"api.anthropic.com",   // Claude API
		"claude.ai",           // Claude web interface
		"github.com",          // GitHub
		"api.github.com",      // GitHub API
		"gitlab.com",          // GitLab
		"bitbucket.org",       // Bitbucket
		"stackoverflow.com",   // Stack Overflow
		"npmjs.org",           // NPM registry
		"pypi.org",            // Python package index
		"golang.org",          // Go packages
		"pkg.go.dev",          // Go package documentation
		"localhost",           // Local development
		"127.0.0.1",           // Local development
	}
	
	for _, workHost := range workRelatedHosts {
		if strings.Contains(host, workHost) {
			return true
		}
	}
	
	// Check for development-related URL patterns
	workRelatedPaths := []string{
		"/api/",
		"/v1/",
		"/v2/",
		"/graphql",
		"/webhook",
		"/auth",
		"/oauth",
	}
	
	for _, workPath := range workRelatedPaths {
		if strings.Contains(url, workPath) {
			return true
		}
	}
	
	return false
}

/**
 * CONTEXT:   Handle generated activity events from activity generator
 * INPUT:     Generated activity event from monitoring correlation
 * OUTPUT:    Activity processed through normal activity processing pipeline
 * BUSINESS:  Generated activities enable comprehensive work tracking from monitoring
 * CHANGE:    Initial generated activity handling for enhanced monitoring
 * RISK:      High - Generated activity handling affecting work time calculations
 */
func (s *EmbeddedServer) handleGeneratedActivity(activity monitor.ActivityEvent) {
	// Convert monitor.ActivityEvent to server ActivityEvent
	// Convert metadata from map[string]interface{} to map[string]string
	metadata := make(map[string]string)
	for key, value := range activity.Metadata {
		metadata[key] = fmt.Sprintf("%v", value)
	}
	
	serverActivity := &ActivityEvent{
		ID:             activity.ID,
		UserID:         activity.UserID,
		SessionID:      "", // Will be set during processing
		WorkBlockID:    "", // Will be set during processing
		ProjectPath:    activity.ProjectPath,
		ProjectName:    activity.ProjectName,
		ActivityType:   activity.ActivityType,
		ActivitySource: activity.ActivitySource,
		Timestamp:      activity.Timestamp,
		Command:        activity.Command,
		Description:    activity.Description,
		Metadata:       metadata,
		CreatedAt:      time.Now(),
	}
	
	// Process through normal activity processing pipeline
	if err := s.processActivity(serverActivity); err != nil {
		log.Printf("Error processing generated activity: %v", err)
	} else {
		log.Printf("✅ Processed generated activity: %s for %s", 
			activity.ActivityType, activity.ProjectName)
	}
	
	// Save data periodically to KuzuDB
	go s.saveDataToKuzu()
}

/**
 * CONTEXT:   Handle HTTP events from HTTP monitor for POST request work detection
 * INPUT:     HTTP event from HTTP monitor (POST requests to Claude API)
 * OUTPUT:    HTTP event processed through activity generator for work tracking
 * BUSINESS:  HTTP POST event handling enables definitive work activity detection
 * CHANGE:    New HTTP POST event handling for precise work activity tracking
 * RISK:      High - HTTP POST event handling directly affecting work activity detection
 */
func (s *EmbeddedServer) handleHTTPEvent(event monitor.HTTPEvent) {
	// Process all HTTP traffic for comprehensive work detection
	// Prioritize Claude API calls but also track development-related traffic
	
	isWorkRelatedTraffic := event.IsClaudeAPI || 
		s.isWorkRelatedHTTPTraffic(event.Host, event.URL)
	
	if !isWorkRelatedTraffic {
		// Skip non-work related traffic (reduce noise)
		return
	}
	
	// Send to activity generator for work activity generation
	if s.activityGenerator != nil {
		// Process HTTP event for work activity tracking  
		if event.Method == "POST" || event.Method == "POST/PUT" || event.Method == "PUT" {
			s.activityGenerator.HandleHTTPPostEvent(event)
		}
		
		if s.config.LogLevel == "debug" || s.config.LogLevel == "verbose" {
			emoji := "🔥"
			if event.IsClaudeAPI {
				emoji = "🤖"
			}
			log.Printf("%s HTTP work activity detected: %s %s to %s", 
				emoji, event.Method, event.URL, event.Host)
		}
	}
}

/**
 * CONTEXT:   Handle File I/O events from File I/O monitor for work detection
 * INPUT:     File I/O event from File I/O monitor (work file operations)
 * OUTPUT:    File I/O event processed through activity generator for work tracking
 * BUSINESS:  File I/O event handling enables precise work activity detection
 * CHANGE:    New File I/O event handling for real-time work activity tracking
 * RISK:      High - File I/O event handling directly affecting work activity detection
 */
func (s *EmbeddedServer) handleFileIOEvent(event monitor.FileIOEvent) {
	// Only process work file operations for activity detection
	if !event.IsWorkFile {
		return
	}
	
	// Send to activity generator for work activity generation
	if s.activityGenerator != nil {
		s.activityGenerator.HandleFileIOEvent(event)
		
		if s.config.LogLevel == "debug" || s.config.LogLevel == "verbose" {
			log.Printf("📁 File I/O work activity detected: %s on %s", 
				event.Type, event.FilePath)
		}
	}
}

/**
 * CONTEXT:   Log Claude process startup for monitoring visibility
 * INPUT:     Process started event with Claude instance information
 * OUTPUT:    Informational log entry only
 * BUSINESS:  Process logging provides visibility into Claude instances without affecting sessions
 * CHANGE:    Simplified to informational logging only
 * RISK:      Low - Pure logging with no system state changes
 */
func (s *EmbeddedServer) logProcessStarted(event monitor.ProcessEvent) {
	// Skip our own daemon process to avoid noise
	if strings.Contains(event.Command, "claude-monitor") {
		return
	}
	
	projectName := extractProjectNameFromPath(event.WorkingDir)
	log.Printf("✅ Claude process started: %s (PID: %d) in project '%s' (%s)", 
		event.Command, event.PID, projectName, event.WorkingDir)
}

/**
 * CONTEXT:   Log Claude process shutdown for monitoring visibility
 * INPUT:     Process stopped event with Claude instance information
 * OUTPUT:    Informational log entry only
 * BUSINESS:  Process logging provides visibility into Claude lifecycle without affecting sessions
 * CHANGE:    Simplified to informational logging only
 * RISK:      Low - Pure logging with no system state changes
 */
func (s *EmbeddedServer) logProcessStopped(event monitor.ProcessEvent) {
	// Skip our own daemon process to avoid noise
	if strings.Contains(event.Command, "claude-monitor") {
		return
	}
	
	projectName := extractProjectNameFromPath(event.WorkingDir)
	log.Printf("❌ Claude process stopped: %s (PID: %d) in project '%s' (%s)", 
		event.Command, event.PID, projectName, event.WorkingDir)
}

/**
 * CONTEXT:   Extract project name from working directory path
 * INPUT:     Working directory path from process event
 * OUTPUT:    Project name derived from directory structure
 * BUSINESS:  Project extraction enables work organization from process monitoring
 * CHANGE:    Initial project name extraction from path
 * RISK:      Low - Path parsing utility for project identification
 */
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

// isValidTime function is defined in commands.go - removed duplicate


/**
 * CONTEXT:   Process monitor statistics endpoint
 * INPUT:     HTTP GET request for monitor statistics
 * OUTPUT:    JSON response with process monitoring statistics
 * BUSINESS:  Statistics endpoint enables monitoring system health assessment
 * CHANGE:    Initial monitor statistics endpoint
 * RISK:      Low - Read-only statistics endpoint
 */
func (s *EmbeddedServer) handleMonitorStats(w http.ResponseWriter, r *http.Request) {
	if s.processMonitor == nil {
		http.Error(w, "Process monitor not available", http.StatusServiceUnavailable)
		return
	}
	
	stats := s.processMonitor.GetStats()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

/**
 * CONTEXT:   Tracked processes endpoint
 * INPUT:     HTTP GET request for currently tracked Claude processes
 * OUTPUT:    JSON response with active Claude processes
 * BUSINESS:  Processes endpoint enables real-time Claude process visibility
 * CHANGE:    Initial tracked processes endpoint
 * RISK:      Low - Read-only process information endpoint
 */
func (s *EmbeddedServer) handleTrackedProcesses(w http.ResponseWriter, r *http.Request) {
	if s.processMonitor == nil {
		http.Error(w, "Process monitor not available", http.StatusServiceUnavailable)
		return
	}
	
	processes := s.processMonitor.GetTrackedProcesses()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processes)
}

/**
 * CONTEXT:   File I/O monitoring statistics endpoint
 * INPUT:     HTTP GET request for file I/O monitoring statistics
 * OUTPUT:    JSON response with file I/O monitoring metrics
 * BUSINESS:  File I/O statistics enable work activity monitoring assessment
 * CHANGE:    Initial file I/O statistics endpoint for enhanced monitoring
 * RISK:      Low - Read-only statistics endpoint
 */
func (s *EmbeddedServer) handleFileIOStats(w http.ResponseWriter, r *http.Request) {
	if s.processMonitor == nil {
		http.Error(w, "Process monitor not available", http.StatusServiceUnavailable)
		return
	}
	
	stats := s.processMonitor.GetFileIOStats()
	if stats == nil {
		http.Error(w, "File I/O monitoring not enabled", http.StatusServiceUnavailable)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

/**
 * CONTEXT:   Activity generator statistics endpoint
 * INPUT:     HTTP GET request for activity generation statistics
 * OUTPUT:    JSON response with activity generation metrics
 * BUSINESS:  Activity statistics enable work tracking system assessment
 * CHANGE:    Initial activity generator statistics endpoint
 * RISK:      Low - Read-only statistics endpoint
 */
func (s *EmbeddedServer) handleActivityStats(w http.ResponseWriter, r *http.Request) {
	if s.activityGenerator == nil {
		http.Error(w, "Activity generator not available", http.StatusServiceUnavailable)
		return
	}
	
	stats := s.activityGenerator.GetStats()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

/**
 * CONTEXT:   Database query endpoint for direct data inspection
 * INPUT:     HTTP GET request with query type parameter
 * OUTPUT:    JSON response with database statistics and record counts
 * BUSINESS:  Database inspection enables monitoring and debugging
 * CHANGE:    New database query endpoint for record analysis
 * RISK:      Low - Read-only database inspection endpoint
 */
func (s *EmbeddedServer) handleDatabaseQuery(w http.ResponseWriter, r *http.Request) {
	queryType := r.URL.Query().Get("type")
	if queryType == "" {
		queryType = "stats"
	}

	if s.db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	var result interface{}

	switch queryType {
	case "stats":
		// Get comprehensive database statistics from KuzuDB
		dbStats, err := s.db.GetDatabaseStats()
		if err != nil {
			http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Add database path to stats
		dbStats["database_path"] = s.config.DatabasePath
		result = dbStats

	case "workblocks":
		// Get all work blocks with activity counters
		workBlocks, err := s.db.GetWorkBlocksWithCounters()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get work blocks: %v", err), http.StatusInternalServerError)
			return
		}
		result = workBlocks

	case "sessions":
		// Get all sessions
		s.mu.RLock()
		sessionList := make([]map[string]interface{}, 0, len(s.sessions))
		for _, session := range s.sessions {
			sessionList = append(sessionList, map[string]interface{}{
				"id":            session.ID,
				"user_id":       session.UserID,
				"start_time":    session.StartTime,
				"last_activity": session.LastActivityTime,
				"activity_count": session.ActivityCount,
				"duration_hours": session.DurationHours,
				"is_active":     session.IsActive,
				"state":         session.State,
			})
		}
		s.mu.RUnlock()
		result = map[string]interface{}{
			"sessions": sessionList,
			"count":    len(sessionList),
		}

	case "workblocks_old":
		// Get all work blocks (deprecated - using KuzuDB version instead)
		s.mu.RLock()
		workBlockList := make([]map[string]interface{}, 0, len(s.workBlocks))
		for _, wb := range s.workBlocks {
			workBlockList = append(workBlockList, map[string]interface{}{
				"id":              wb.ID,
				"session_id":      wb.SessionID,
				"project_name":    wb.ProjectName,
				"project_path":    wb.ProjectPath,
				"start_time":      wb.StartTime,
				"end_time":        wb.EndTime,
				"duration_hours":  wb.DurationHours,
				"duration_seconds": wb.DurationSeconds,
				"activity_count":  wb.ActivityCount,
				"is_active":       wb.IsActive,
				"state":           wb.State,
			})
		}
		s.mu.RUnlock()
		result = map[string]interface{}{
			"work_blocks": workBlockList,
			"count":       len(workBlockList),
		}

	case "activities":
		// Get recent activities with limit
		limitStr := r.URL.Query().Get("limit")
		limit := 20
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
				limit = l
			}
		}

		s.mu.RLock()
		activitiesCount := len(s.activities)
		startIndex := 0
		if activitiesCount > limit {
			startIndex = activitiesCount - limit
		}

		activityList := make([]map[string]interface{}, 0, limit)
		for i := startIndex; i < activitiesCount; i++ {
			activity := s.activities[i]
			activityList = append(activityList, map[string]interface{}{
				"id":             activity.ID,
				"user_id":        activity.UserID,
				"session_id":     activity.SessionID,
				"work_block_id":  activity.WorkBlockID,
				"project_name":   activity.ProjectName,
				"activity_type":  activity.ActivityType,
				"activity_source": activity.ActivitySource,
				"timestamp":      activity.Timestamp,
				"command":        activity.Command,
				"description":    activity.Description,
				"created_at":     activity.CreatedAt,
			})
		}
		s.mu.RUnlock()

		result = map[string]interface{}{
			"activities":      activityList,
			"count":          len(activityList),
			"total_activities": activitiesCount,
			"showing_recent":  limit,
		}

	case "projects":
		// Get project breakdown
		s.mu.RLock()
		projectStats := make(map[string]map[string]interface{})
		for _, wb := range s.workBlocks {
			projectName := wb.ProjectName
			if projectName == "" {
				projectName = "Unknown"
			}
			
			if projectStats[projectName] == nil {
				projectStats[projectName] = map[string]interface{}{
					"name":         projectName,
					"work_blocks":  0,
					"total_hours":  0.0,
					"total_activities": 0,
				}
			}
			
			stats := projectStats[projectName]
			stats["work_blocks"] = stats["work_blocks"].(int) + 1
			stats["total_hours"] = stats["total_hours"].(float64) + wb.DurationHours
		}

		// Count activities per project
		for _, activity := range s.activities {
			projectName := activity.ProjectName
			if projectName == "" {
				projectName = "Unknown"
			}
			
			if projectStats[projectName] != nil {
				stats := projectStats[projectName]
				stats["total_activities"] = stats["total_activities"].(int) + 1
			}
		}
		s.mu.RUnlock()

		projectList := make([]map[string]interface{}, 0, len(projectStats))
		for _, stats := range projectStats {
			projectList = append(projectList, stats)
		}

		result = map[string]interface{}{
			"projects": projectList,
			"count":    len(projectList),
		}

	default:
		http.Error(w, "Invalid query type. Use: stats, sessions, workblocks, activities, projects", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

/**
 * CONTEXT:   Get count of active sessions
 * INPUT:     Current session data
 * OUTPUT:    Count of active sessions
 * BUSINESS:  Helper function for database statistics
 * CHANGE:    New helper function for session counting
 * RISK:      Low - Simple counting function
 */
func (s *EmbeddedServer) getActiveSessionsCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	count := 0
	for _, session := range s.sessions {
		if session.IsActive {
			count++
		}
	}
	return count
}

/**
 * CONTEXT:   Get count of active work blocks
 * INPUT:     Current work block data
 * OUTPUT:    Count of active work blocks
 * BUSINESS:  Helper function for database statistics
 * CHANGE:    New helper function for work block counting
 * RISK:      Low - Simple counting function
 */
func (s *EmbeddedServer) getActiveWorkBlocksCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	count := 0
	for _, workBlock := range s.workBlocks {
		if workBlock.IsActive {
			count++
		}
	}
	return count
}

/**
 * CONTEXT:   Anti-strace validation to prevent zombie process creation
 * INPUT:     System process check for existing strace instances
 * OUTPUT:    Error if dangerous strace processes detected, nil if safe
 * BUSINESS:  Prevent zombie processes by detecting invasive strace usage
 * CHANGE:    Critical safety validation to prevent strace-induced zombie processes
 * RISK:      Low - Protective validation prevents system issues
 */
func (s *EmbeddedServer) validateNoStraceProcesses() error {
	// Check for any existing strace processes targeting claude-monitor
	cmd := exec.Command("pgrep", "-f", "strace.*claude-monitor")
	output, err := cmd.Output()
	
	if err == nil && len(output) > 0 {
		// Found dangerous strace processes
		straceOutput := strings.TrimSpace(string(output))
		return fmt.Errorf("found dangerous strace processes targeting claude-monitor: PIDs %s", straceOutput)
	}
	
	// Check for any strace processes with ptrace that could interfere
	cmd = exec.Command("pgrep", "-f", "strace.*-p.*[0-9]")
	output, err = cmd.Output()
	
	if err == nil && len(output) > 0 {
		// Found potentially interfering strace processes
		log.Printf("🔍 Found general strace processes (may not be related): %s", strings.TrimSpace(string(output)))
	}
	
	return nil
}

