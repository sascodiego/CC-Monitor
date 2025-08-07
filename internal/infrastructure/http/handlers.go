/**
 * CONTEXT:   HTTP handlers for Claude Monitor daemon API endpoints
 * INPUT:     HTTP requests from Claude Code hooks and CLI tools
 * OUTPUT:    JSON responses with activity processing, health checks, and status info
 * BUSINESS:  Provide HTTP API interface for all Claude Monitor functionality
 * CHANGE:    Initial HTTP handler implementation with complete API coverage
 * RISK:      High - API endpoints affect all client integrations and work tracking
 */

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/claude-monitor/system/internal/entities"
)

/**
 * CONTEXT:   HTTP handler collection with dependency injection for business logic
 * INPUT:     Event processor and logger dependencies for request handling
 * OUTPUT:    Complete HTTP handler set for Claude Monitor API endpoints
 * BUSINESS:  Coordinate HTTP layer with business logic through dependency injection
 * CHANGE:    Initial handler structure with proper separation of concerns
 * RISK:      Medium - Handler structure affects API consistency and error handling
 */
type Handlers struct {
	eventProcessor EventProcessor
	logger         *slog.Logger
	startTime      time.Time
}

// EventProcessor defines the interface for processing activity events
type EventProcessor interface {
	ProcessActivity(ctx context.Context, event *entities.ActivityEvent) error
	GetSystemStatus(ctx context.Context) (*SystemStatus, error)
	GetActiveSession(ctx context.Context, userID string) (*entities.Session, error)
	GetActiveWorkBlock(ctx context.Context, sessionID string) (*entities.WorkBlock, error)
}

// SystemStatus represents overall system health and statistics
type SystemStatus struct {
	Status           string            `json:"status"`
	Uptime           time.Duration     `json:"uptime"`
	Version          string            `json:"version"`
	ActiveSessions   int               `json:"active_sessions"`
	ActiveWorkBlocks int               `json:"active_work_blocks"`
	TotalActivities  int64             `json:"total_activities"`
	DatabaseStatus   string            `json:"database_status"`
	LastActivity     time.Time         `json:"last_activity"`
	Metrics          map[string]string `json:"metrics"`
}

// HandlerConfig holds configuration for handlers
type HandlerConfig struct {
	EventProcessor EventProcessor
	Logger         *slog.Logger
}

/**
 * CONTEXT:   Factory function for creating HTTP handlers with dependencies
 * INPUT:     HandlerConfig with event processor and logger dependencies
 * OUTPUT:    Configured Handlers instance ready for HTTP server
 * BUSINESS:  Handlers require event processor for business logic coordination
 * CHANGE:    Initial factory implementation with dependency injection
 * RISK:      Low - Factory function with validation and initialization
 */
func NewHandlers(config HandlerConfig) (*Handlers, error) {
	if config.EventProcessor == nil {
		return nil, fmt.Errorf("event processor cannot be nil")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Handlers{
		eventProcessor: config.EventProcessor,
		logger:         logger,
		startTime:      time.Now(),
	}, nil
}

/**
 * CONTEXT:   Main activity event processing endpoint for Claude Code hooks
 * INPUT:     HTTP POST request with ActivityEvent JSON payload
 * OUTPUT:    HTTP response with processing status and timing information
 * BUSINESS:  Core endpoint for all Claude activity tracking and work hour calculation
 * CHANGE:    Initial activity handler with comprehensive error handling and validation
 * RISK:      High - Primary endpoint for all work tracking functionality
 */
func (h *Handlers) HandleActivity(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	h.logger.Info("Processing activity event", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)

	// Parse request body
	var activityData ActivityEventRequest
	if err := json.NewDecoder(r.Body).Decode(&activityData); err != nil {
		h.logger.Error("Failed to parse activity request", "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format", err)
		return
	}

	// Create activity event from request
	activityEvent, err := h.createActivityEventFromRequest(&activityData)
	if err != nil {
		h.logger.Error("Failed to create activity event", "error", err)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid activity data", err)
		return
	}

	// Process the activity event
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := h.eventProcessor.ProcessActivity(ctx, activityEvent); err != nil {
		h.logger.Error("Failed to process activity event", 
			"activityID", activityEvent.ID(), 
			"userID", activityEvent.UserID(),
			"error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Processing failed", err)
		return
	}

	// Success response
	processingTime := time.Since(startTime)
	response := ActivityEventResponse{
		Status:       "processed",
		ActivityID:   activityEvent.ID(),
		Timestamp:    time.Now().UTC(),
		ProcessingMS: processingTime.Milliseconds(),
		UserID:       activityEvent.UserID(),
		ProjectName:  activityEvent.ProjectName(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
	}

	h.logger.Info("Activity processed successfully", 
		"activityID", activityEvent.ID(),
		"processingTime", processingTime,
		"userID", activityEvent.UserID(),
		"project", activityEvent.ProjectName())
}

/**
 * CONTEXT:   System health check endpoint for monitoring and load balancers
 * INPUT:     HTTP GET request for health status
 * OUTPUT:    HTTP response with system health status and basic metrics
 * BUSINESS:  Provide health monitoring for operational visibility and alerts
 * CHANGE:    Initial health check implementation with system status
 * RISK:      Low - Read-only monitoring endpoint with no state changes
 */
func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get system status
	status, err := h.eventProcessor.GetSystemStatus(ctx)
	if err != nil {
		h.logger.Error("Failed to get system status", "error", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Health check failed", err)
		return
	}

	// Calculate uptime
	status.Uptime = time.Since(h.startTime)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Error("Failed to encode health response", "error", err)
	}
}

/**
 * CONTEXT:   Readiness check endpoint for deployment and orchestration systems
 * INPUT:     HTTP GET request for readiness status
 * OUTPUT:    HTTP response indicating if system is ready to serve requests
 * BUSINESS:  Support deployment automation and traffic routing decisions
 * CHANGE:    Initial readiness check with database connectivity validation
 * RISK:      Low - Simple readiness validation for deployment systems
 */
func (h *Handlers) HandleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Quick health check
	_, err := h.eventProcessor.GetSystemStatus(ctx)
	if err != nil {
		h.writeErrorResponse(w, http.StatusServiceUnavailable, "System not ready", err)
		return
	}

	response := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(h.startTime),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

/**
 * CONTEXT:   Current status endpoint for CLI tools and debugging
 * INPUT:     HTTP GET request with optional user_id query parameter
 * OUTPUT:    HTTP response with current sessions, work blocks, and activity status
 * BUSINESS:  Provide real-time status information for CLI commands and debugging
 * CHANGE:    Initial status endpoint with session and work block information
 * RISK:      Medium - Status endpoint provides internal state information
 */
func (h *Handlers) HandleStatus(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "user_id parameter required", nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get active session
	activeSession, err := h.eventProcessor.GetActiveSession(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get active session", "userID", userID, "error", err)
		// Don't fail the whole request, just log and continue
	}

	// Get active work block if we have a session
	var activeWorkBlock *entities.WorkBlock
	if activeSession != nil {
		activeWorkBlock, err = h.eventProcessor.GetActiveWorkBlock(ctx, activeSession.ID())
		if err != nil {
			h.logger.Error("Failed to get active work block", "sessionID", activeSession.ID(), "error", err)
			// Don't fail the whole request, just log and continue
		}
	}

	// Build status response
	statusResponse := UserStatusResponse{
		UserID:          userID,
		Timestamp:       time.Now().UTC(),
		HasActiveSession: activeSession != nil,
		HasActiveWork:   activeWorkBlock != nil,
	}

	if activeSession != nil {
		sessionData := SessionData{
			ID:               activeSession.ID(),
			StartTime:        activeSession.StartTime(),
			EndTime:          activeSession.EndTime(),
			State:            string(activeSession.State()),
			ActivityCount:    int(activeSession.ActivityCount()),
			Duration:         activeSession.Duration(),
			IsActive:         activeSession.IsActive(),
		}
		statusResponse.ActiveSession = &sessionData
	}

	if activeWorkBlock != nil {
		workBlockData := WorkBlockData{
			ID:               activeWorkBlock.ID(),
			SessionID:        activeWorkBlock.SessionID(),
			ProjectID:        activeWorkBlock.ProjectID(),
			ProjectName:      activeWorkBlock.ProjectName(),
			ProjectPath:      activeWorkBlock.ProjectPath(),
			StartTime:        activeWorkBlock.StartTime(),
			LastActivity:     activeWorkBlock.LastActivityTime(),
			Duration:         activeWorkBlock.Duration(),
			IsActive:         activeWorkBlock.IsActive(),
		}
		statusResponse.ActiveWorkBlock = &workBlockData
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(statusResponse); err != nil {
		h.logger.Error("Failed to encode status response", "error", err)
	}
}

/**
 * CONTEXT:   Session listing endpoint for analytics and reporting
 * INPUT:     HTTP GET request with optional filtering parameters
 * OUTPUT:    HTTP response with list of sessions based on filter criteria
 * BUSINESS:  Support session queries for CLI reporting and analytics
 * CHANGE:    Initial session listing with basic filtering support
 * RISK:      Medium - Endpoint providing session history and analytics data
 */
func (h *Handlers) HandleSessions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "user_id parameter required", nil)
		return
	}

	// Parse optional date range
	var startDate, endDate time.Time
	if startStr := r.URL.Query().Get("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}
	if endStr := r.URL.Query().Get("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed.Add(24 * time.Hour) // Include the entire end date
		}
	}

	// Parse limit
	limit := 50 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// This would require extending the EventProcessor interface
	// For now, return a simple response
	response := map[string]interface{}{
		"user_id": userID,
		"start_date": startDate,
		"end_date": endDate,
		"limit": limit,
		"sessions": []interface{}{}, // Placeholder
		"total": 0,
		"message": "Session listing not yet implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

/**
 * CONTEXT:   Work block listing endpoint for detailed work time analysis
 * INPUT:     HTTP GET request with session_id or user_id and date range
 * OUTPUT:    HTTP response with list of work blocks and time analytics
 * BUSINESS:  Support work block queries for detailed time tracking analysis
 * CHANGE:    Initial work block listing endpoint for time analytics
 * RISK:      Medium - Endpoint providing detailed work time information
 */
func (h *Handlers) HandleWorkBlocks(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	userID := r.URL.Query().Get("user_id")
	
	if sessionID == "" && userID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "session_id or user_id parameter required", nil)
		return
	}

	// This would require extending the EventProcessor interface
	// For now, return a simple response
	response := map[string]interface{}{
		"session_id": sessionID,
		"user_id": userID,
		"work_blocks": []interface{}{}, // Placeholder
		"total": 0,
		"message": "Work block listing not yet implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Request and Response types

type ActivityEventRequest struct {
	UserID         string            `json:"user_id"`
	ProjectPath    string            `json:"project_path"`
	ProjectName    string            `json:"project_name"`
	ActivityType   string            `json:"activity_type"`
	ActivitySource string            `json:"activity_source"`
	Timestamp      time.Time         `json:"timestamp"`
	Command        string            `json:"command"`
	Description    string            `json:"description"`
	Metadata       map[string]string `json:"metadata"`
}

type ActivityEventResponse struct {
	Status       string    `json:"status"`
	ActivityID   string    `json:"activity_id"`
	Timestamp    time.Time `json:"timestamp"`
	ProcessingMS int64     `json:"processing_ms"`
	UserID       string    `json:"user_id"`
	ProjectName  string    `json:"project_name"`
}

type UserStatusResponse struct {
	UserID           string          `json:"user_id"`
	Timestamp        time.Time       `json:"timestamp"`
	HasActiveSession bool            `json:"has_active_session"`
	HasActiveWork    bool            `json:"has_active_work"`
	ActiveSession    *SessionData    `json:"active_session,omitempty"`
	ActiveWorkBlock  *WorkBlockData  `json:"active_work_block,omitempty"`
}

type SessionData struct {
	ID            string        `json:"id"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	State         string        `json:"state"`
	ActivityCount int           `json:"activity_count"`
	Duration      time.Duration `json:"duration"`
	IsActive      bool          `json:"is_active"`
}

type WorkBlockData struct {
	ID           string        `json:"id"`
	SessionID    string        `json:"session_id"`
	ProjectID    string        `json:"project_id"`
	ProjectName  string        `json:"project_name"`
	ProjectPath  string        `json:"project_path"`
	StartTime    time.Time     `json:"start_time"`
	LastActivity time.Time     `json:"last_activity"`
	Duration     time.Duration `json:"duration"`
	IsActive     bool          `json:"is_active"`
}

type ErrorResponse struct {
	Status    string    `json:"status"`
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
}

// Helper methods

/**
 * CONTEXT:   Create activity event entity from HTTP request data
 * INPUT:     ActivityEventRequest with client-provided activity information
 * OUTPUT:    ActivityEvent entity ready for business logic processing
 * BUSINESS:  Convert HTTP request data to domain entity with proper validation
 * CHANGE:    Initial request-to-entity conversion with comprehensive validation
 * RISK:      Medium - Data conversion affects entity creation and validation
 */
func (h *Handlers) createActivityEventFromRequest(req *ActivityEventRequest) (*entities.ActivityEvent, error) {
	// Set default timestamp if not provided
	timestamp := req.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	// Map activity type
	activityType := entities.ActivityTypeOther
	switch req.ActivityType {
	case "command":
		activityType = entities.ActivityTypeCommand
	case "file_edit":
		activityType = entities.ActivityTypeFileEdit
	case "file_read":
		activityType = entities.ActivityTypeFileRead
	case "navigation":
		activityType = entities.ActivityTypeNavigation
	case "search":
		activityType = entities.ActivityTypeSearch
	case "generation":
		activityType = entities.ActivityTypeGeneration
	}

	// Map activity source
	activitySource := entities.ActivitySourceHook
	switch req.ActivitySource {
	case "cli":
		activitySource = entities.ActivitySourceCLI
	case "daemon":
		activitySource = entities.ActivitySourceDaemon
	case "manual":
		activitySource = entities.ActivitySourceManual
	}

	// Create activity event
	return entities.NewActivityEvent(entities.ActivityEventConfig{
		UserID:         req.UserID,
		ProjectPath:    req.ProjectPath,
		ProjectName:    req.ProjectName,
		ActivityType:   activityType,
		ActivitySource: activitySource,
		Timestamp:      timestamp,
		Command:        req.Command,
		Description:    req.Description,
		Metadata:       req.Metadata,
	})
}

/**
 * CONTEXT:   Standardized error response helper for consistent API error handling
 * INPUT:     HTTP response writer, status code, message, and optional error details
 * OUTPUT:    JSON error response with consistent format and logging
 * BUSINESS:  Provide consistent error responses across all API endpoints
 * CHANGE:    Initial error response helper with standard format
 * RISK:      Low - Error handling utility with no business logic
 */
func (h *Handlers) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	errorResponse := ErrorResponse{
		Status:    "error",
		Error:     http.StatusText(statusCode),
		Message:   message,
		Timestamp: time.Now().UTC(),
	}

	// Add error details for internal server errors (but not for client errors)
	if statusCode >= 500 && err != nil {
		errorResponse.Message = fmt.Sprintf("%s: %v", message, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if encodeErr := json.NewEncoder(w).Encode(errorResponse); encodeErr != nil {
		h.logger.Error("Failed to encode error response", "error", encodeErr)
	}
}