/**
 * CONTEXT:   HTTP handlers for daemon-managed session correlation endpoints
 * INPUT:     HTTP requests from hook events with session context for correlation
 * OUTPUT:    JSON responses with session creation, correlation, and completion results
 * BUSINESS:  Handle hook start/end events without temporary files or environment variables
 * CHANGE:    Initial session correlation handlers replacing file-based hook correlation
 * RISK:      High - Session correlation endpoints affect all hook integration accuracy
 */

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/claude-monitor/system/internal/usecases"
)

// SessionCorrelationProcessor defines the interface for session correlation operations
type SessionCorrelationProcessor interface {
	CreateSession(ctx context.Context, sessionContext entities.SessionContext) (*entities.ActiveSession, error)
	FindSessionForEndEvent(ctx context.Context, sessionContext entities.SessionContext) (*entities.ActiveSession, error)
	EndSession(ctx context.Context, sessionID string, endTime time.Time, processingDuration time.Duration, tokenCount int64) (*entities.Session, error)
	GetActiveSessionStatistics(ctx context.Context) (map[string]interface{}, error)
}

// ExtendedHandlers extends the base handlers with session correlation functionality
type ExtendedHandlers struct {
	*Handlers
	sessionTracker usecases.ActiveSessionTracker
}

// ExtendedHandlerConfig holds configuration for extended handlers
type ExtendedHandlerConfig struct {
	HandlerConfig  HandlerConfig
	SessionTracker usecases.ActiveSessionTracker
}

/**
 * CONTEXT:   Factory function for creating extended handlers with session correlation
 * INPUT:     ExtendedHandlerConfig with base handler config and session tracker
 * OUTPUT:    Extended handlers with session correlation endpoints
 * BUSINESS:  Add session correlation capabilities to existing HTTP handlers
 * CHANGE:    Initial extended handler factory with session tracker integration
 * RISK:      Medium - Extended handlers must maintain compatibility with base handlers
 */
func NewExtendedHandlers(config ExtendedHandlerConfig) (*ExtendedHandlers, error) {
	baseHandlers, err := NewHandlers(config.HandlerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create base handlers: %w", err)
	}

	return &ExtendedHandlers{
		Handlers:       baseHandlers,
		sessionTracker: config.SessionTracker,
	}, nil
}

/**
 * CONTEXT:   Hook start event endpoint for creating active sessions
 * INPUT:     HTTP POST request with SessionContext from hook start event
 * OUTPUT:    HTTP response with created active session information
 * BUSINESS:  Create active session from hook start without requiring ID passing
 * CHANGE:    Initial session start endpoint for daemon-managed correlation
 * RISK:      High - Session start accuracy affects all subsequent correlation attempts
 */
func (eh *ExtendedHandlers) HandleSessionStart(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	eh.logger.Info("Processing session start event", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)

	// Parse request body
	var startRequest SessionStartRequest
	if err := json.NewDecoder(r.Body).Decode(&startRequest); err != nil {
		eh.logger.Error("Failed to parse session start request", "error", err)
		eh.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format", err)
		return
	}

	// Validate required fields
	if err := eh.validateSessionStartRequest(&startRequest); err != nil {
		eh.logger.Error("Invalid session start request", "error", err)
		eh.writeErrorResponse(w, http.StatusBadRequest, "Invalid session start data", err)
		return
	}

	// Create session context
	sessionContext := entities.SessionContext{
		TerminalPID: startRequest.TerminalPID,
		ShellPID:    startRequest.ShellPID,
		WorkingDir:  startRequest.WorkingDir,
		ProjectPath: startRequest.ProjectPath,
		UserID:      startRequest.UserID,
		Timestamp:   startRequest.Timestamp,
	}

	// Create active session
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	activeSession, err := eh.sessionTracker.CreateSession(ctx, sessionContext)
	if err != nil {
		eh.logger.Error("Failed to create active session", 
			"userID", startRequest.UserID,
			"terminalPID", startRequest.TerminalPID,
			"workingDir", startRequest.WorkingDir,
			"error", err)
		eh.writeErrorResponse(w, http.StatusInternalServerError, "Session creation failed", err)
		return
	}

	// Success response
	processingTime := time.Since(startTime)
	response := SessionStartResponse{
		Status:               "created",
		SessionID:            activeSession.ID(),
		TerminalPID:          startRequest.TerminalPID,
		UserID:               startRequest.UserID,
		WorkingDir:           startRequest.WorkingDir,
		ProjectPath:          startRequest.ProjectPath,
		StartTime:            activeSession.StartTime(),
		EstimatedEndTime:     activeSession.EstimatedEndTime(),
		Timestamp:            time.Now().UTC(),
		ProcessingMS:         processingTime.Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		eh.logger.Error("Failed to encode session start response", "error", err)
	}

	eh.logger.Info("Session started successfully", 
		"sessionID", activeSession.ID(),
		"processingTime", processingTime,
		"userID", startRequest.UserID,
		"terminalPID", startRequest.TerminalPID)
}

/**
 * CONTEXT:   Hook end event endpoint for correlating and completing sessions
 * INPUT:     HTTP POST request with SessionContext and processing metrics from hook end
 * OUTPUT:    HTTP response with correlated session completion information
 * BUSINESS:  Correlate end events with active sessions using context matching
 * CHANGE:    Initial session end endpoint with context-based correlation
 * RISK:      High - Session correlation accuracy critical for work tracking reliability
 */
func (eh *ExtendedHandlers) HandleSessionEnd(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	eh.logger.Info("Processing session end event", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)

	// Parse request body
	var endRequest SessionEndRequest
	if err := json.NewDecoder(r.Body).Decode(&endRequest); err != nil {
		eh.logger.Error("Failed to parse session end request", "error", err)
		eh.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format", err)
		return
	}

	// Validate required fields
	if err := eh.validateSessionEndRequest(&endRequest); err != nil {
		eh.logger.Error("Invalid session end request", "error", err)
		eh.writeErrorResponse(w, http.StatusBadRequest, "Invalid session end data", err)
		return
	}

	// Create session context for correlation
	sessionContext := entities.SessionContext{
		TerminalPID: endRequest.TerminalPID,
		ShellPID:    endRequest.ShellPID,
		WorkingDir:  endRequest.WorkingDir,
		ProjectPath: endRequest.ProjectPath,
		UserID:      endRequest.UserID,
		Timestamp:   endRequest.Timestamp,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Find matching active session
	activeSession, err := eh.sessionTracker.FindSessionForEndEvent(ctx, sessionContext)
	if err != nil {
		eh.logger.Error("Failed to find matching session for end event", 
			"userID", endRequest.UserID,
			"terminalPID", endRequest.TerminalPID,
			"workingDir", endRequest.WorkingDir,
			"error", err)
		eh.writeErrorResponse(w, http.StatusNotFound, "Matching session not found", err)
		return
	}

	// End the session with processing metrics
	processingDuration := time.Duration(endRequest.ProcessingDurationSeconds) * time.Second
	completedSession, err := eh.sessionTracker.EndSession(ctx, activeSession.ID(), 
		endRequest.Timestamp, processingDuration, endRequest.TokenCount)
	if err != nil {
		eh.logger.Error("Failed to end session", 
			"sessionID", activeSession.ID(),
			"error", err)
		eh.writeErrorResponse(w, http.StatusInternalServerError, "Session completion failed", err)
		return
	}

	// Success response
	requestProcessingTime := time.Since(startTime)
	response := SessionEndResponse{
		Status:                   "completed",
		ActiveSessionID:          activeSession.ID(),
		CompletedSessionID:       completedSession.ID(),
		TerminalPID:              endRequest.TerminalPID,
		UserID:                   endRequest.UserID,
		WorkingDir:               endRequest.WorkingDir,
		Duration:                 completedSession.Duration(),
		ProcessingDurationActual: processingDuration,
		TokenCount:               endRequest.TokenCount,
		Timestamp:                time.Now().UTC(),
		ProcessingMS:             requestProcessingTime.Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		eh.logger.Error("Failed to encode session end response", "error", err)
	}

	eh.logger.Info("Session ended successfully", 
		"activeSessionID", activeSession.ID(),
		"completedSessionID", completedSession.ID(),
		"correlationTime", requestProcessingTime,
		"sessionDuration", completedSession.Duration(),
		"userID", endRequest.UserID)
}

/**
 * CONTEXT:   Active session status endpoint for debugging and monitoring
 * INPUT:     HTTP GET request with optional filtering parameters
 * OUTPUT:    HTTP response with current active sessions and correlation statistics
 * BUSINESS:  Provide visibility into active session correlation system state
 * CHANGE:    Initial active session status endpoint for system monitoring
 * RISK:      Low - Read-only monitoring endpoint for debugging and operations
 */
func (eh *ExtendedHandlers) HandleActiveSessionStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get active session statistics
	stats, err := eh.sessionTracker.GetStatistics()
	if err != nil {
		eh.logger.Error("Failed to get active session statistics", "error", err)
		eh.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get statistics", err)
		return
	}

	// Get optional user filter
	userFilter := r.URL.Query().Get("user_id")

	response := ActiveSessionStatusResponse{
		Timestamp:         time.Now().UTC(),
		Statistics:        stats,
		UserFilter:        userFilter,
		CorrelationStatus: "operational",
	}

	// Add user-specific information if requested
	if userFilter != "" {
		// This would require extending the session tracker to get user-specific active sessions
		// For now, we'll include a placeholder
		response.UserActiveSessions = 0 // Placeholder
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		eh.logger.Error("Failed to encode active session status response", "error", err)
	}
}

/**
 * CONTEXT:   Session correlation test endpoint for validating correlation logic
 * INPUT:     HTTP POST request with test session context for correlation simulation
 * OUTPUT:    HTTP response with correlation test results and matching scores
 * BUSINESS:  Support testing and debugging of session correlation algorithms
 * CHANGE:    Initial correlation test endpoint for development and debugging
 * RISK:      Low - Testing endpoint with no production data modification
 */
func (eh *ExtendedHandlers) HandleCorrelationTest(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var testRequest CorrelationTestRequest
	if err := json.NewDecoder(r.Body).Decode(&testRequest); err != nil {
		eh.logger.Error("Failed to parse correlation test request", "error", err)
		eh.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format", err)
		return
	}

	// This endpoint would simulate correlation without actually creating sessions
	// It's useful for testing and debugging the correlation algorithms
	response := CorrelationTestResponse{
		Status:              "simulated",
		Timestamp:           time.Now().UTC(),
		TestContext:         testRequest.TestContext,
		PotentialMatches:    0, // Would calculate based on current active sessions
		BestMatchScore:      0.0, // Would calculate correlation score
		CorrelationStrategy: "terminal_pid", // Would determine best strategy
		Message:             "Correlation test completed (simulation mode)",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Request and Response types for session correlation

type SessionStartRequest struct {
	TerminalPID         int                   `json:"terminal_pid"`
	ShellPID            int                   `json:"shell_pid"`
	WorkingDir          string                `json:"working_dir"`
	ProjectPath         string                `json:"project_path"`
	UserID              string                `json:"user_id"`
	Timestamp           time.Time             `json:"timestamp"`
	EstimatedDuration   int                   `json:"estimated_duration_seconds,omitempty"`
	EstimatedTokens     int64                 `json:"estimated_tokens,omitempty"`
	Command             string                `json:"command,omitempty"`
	Metadata            map[string]string     `json:"metadata,omitempty"`
}

type SessionStartResponse struct {
	Status               string                `json:"status"`
	SessionID            string                `json:"session_id"`
	TerminalPID          int                   `json:"terminal_pid"`
	UserID               string                `json:"user_id"`
	WorkingDir           string                `json:"working_dir"`
	ProjectPath          string                `json:"project_path"`
	StartTime            time.Time             `json:"start_time"`
	EstimatedEndTime     time.Time             `json:"estimated_end_time"`
	Timestamp            time.Time             `json:"timestamp"`
	ProcessingMS         int64                 `json:"processing_ms"`
}

type SessionEndRequest struct {
	TerminalPID                int                   `json:"terminal_pid"`
	ShellPID                   int                   `json:"shell_pid"`
	WorkingDir                 string                `json:"working_dir"`
	ProjectPath                string                `json:"project_path"`
	UserID                     string                `json:"user_id"`
	Timestamp                  time.Time             `json:"timestamp"`
	ProcessingDurationSeconds  int64                 `json:"processing_duration_seconds"`
	TokenCount                 int64                 `json:"token_count"`
	Success                    bool                  `json:"success"`
	ErrorMessage               string                `json:"error_message,omitempty"`
	ResponseTokens             int64                 `json:"response_tokens,omitempty"`
	Metadata                   map[string]string     `json:"metadata,omitempty"`
}

type SessionEndResponse struct {
	Status                   string        `json:"status"`
	ActiveSessionID          string        `json:"active_session_id"`
	CompletedSessionID       string        `json:"completed_session_id"`
	TerminalPID              int           `json:"terminal_pid"`
	UserID                   string        `json:"user_id"`
	WorkingDir               string        `json:"working_dir"`
	Duration                 time.Duration `json:"duration"`
	ProcessingDurationActual time.Duration `json:"processing_duration_actual"`
	TokenCount               int64         `json:"token_count"`
	Timestamp                time.Time     `json:"timestamp"`
	ProcessingMS             int64         `json:"processing_ms"`
}

type ActiveSessionStatusResponse struct {
	Timestamp            time.Time              `json:"timestamp"`
	Statistics           map[string]interface{} `json:"statistics"`
	UserFilter           string                 `json:"user_filter,omitempty"`
	UserActiveSessions   int                    `json:"user_active_sessions,omitempty"`
	CorrelationStatus    string                 `json:"correlation_status"`
}

type CorrelationTestRequest struct {
	TestContext entities.SessionContext `json:"test_context"`
	TestMode    string                   `json:"test_mode"` // "find_match", "simulate_create"
}

type CorrelationTestResponse struct {
	Status              string                `json:"status"`
	Timestamp           time.Time             `json:"timestamp"`
	TestContext         entities.SessionContext `json:"test_context"`
	PotentialMatches    int                   `json:"potential_matches"`
	BestMatchScore      float64               `json:"best_match_score"`
	CorrelationStrategy string                `json:"correlation_strategy"`
	Message             string                `json:"message"`
}

// Validation helper methods

/**
 * CONTEXT:   Validate session start request data for completeness and correctness
 * INPUT:     SessionStartRequest with hook start event data
 * OUTPUT:    Error if validation fails, nil if valid
 * BUSINESS:  Ensure session start requests contain all necessary correlation data
 * CHANGE:    Initial validation logic for session start requests
 * RISK:      Medium - Invalid requests affect session creation accuracy
 */
func (eh *ExtendedHandlers) validateSessionStartRequest(req *SessionStartRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if req.WorkingDir == "" {
		return fmt.Errorf("working_dir is required")
	}

	if req.TerminalPID <= 0 {
		return fmt.Errorf("terminal_pid must be positive, got %d", req.TerminalPID)
	}

	if req.ShellPID <= 0 {
		return fmt.Errorf("shell_pid must be positive, got %d", req.ShellPID)
	}

	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now() // Set default timestamp
	}

	// Validate timestamp is reasonable
	maxFutureTime := time.Now().Add(1 * time.Hour)
	if req.Timestamp.After(maxFutureTime) {
		return fmt.Errorf("timestamp cannot be more than 1 hour in the future")
	}

	minPastTime := time.Now().Add(-24 * time.Hour)
	if req.Timestamp.Before(minPastTime) {
		return fmt.Errorf("timestamp cannot be more than 24 hours in the past")
	}

	return nil
}

/**
 * CONTEXT:   Validate session end request data for correlation and completion
 * INPUT:     SessionEndRequest with hook end event data and processing metrics
 * OUTPUT:    Error if validation fails, nil if valid
 * BUSINESS:  Ensure session end requests contain necessary data for correlation and completion
 * CHANGE:    Initial validation logic for session end requests
 * RISK:      Medium - Invalid requests affect session correlation and completion accuracy
 */
func (eh *ExtendedHandlers) validateSessionEndRequest(req *SessionEndRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if req.WorkingDir == "" {
		return fmt.Errorf("working_dir is required")
	}

	if req.TerminalPID <= 0 {
		return fmt.Errorf("terminal_pid must be positive, got %d", req.TerminalPID)
	}

	if req.ShellPID <= 0 {
		return fmt.Errorf("shell_pid must be positive, got %d", req.ShellPID)
	}

	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now() // Set default timestamp
	}

	if req.ProcessingDurationSeconds < 0 {
		return fmt.Errorf("processing_duration_seconds cannot be negative, got %d", req.ProcessingDurationSeconds)
	}

	if req.TokenCount < 0 {
		return fmt.Errorf("token_count cannot be negative, got %d", req.TokenCount)
	}

	// Validate reasonable processing duration (max 30 minutes)
	maxProcessingDuration := int64(30 * 60) // 30 minutes in seconds
	if req.ProcessingDurationSeconds > maxProcessingDuration {
		return fmt.Errorf("processing_duration_seconds too large, max %d seconds", maxProcessingDuration)
	}

	return nil
}