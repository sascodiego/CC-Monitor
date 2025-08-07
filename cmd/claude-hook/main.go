/**
 * CONTEXT:   Simplified hook command for daemon-managed session correlation
 * INPUT:     Command-line arguments with hook type and optional processing metrics
 * OUTPUT:    HTTP requests to daemon with automatically detected session context
 * BUSINESS:  Eliminate manual ID management by using automatic context detection for hook correlation
 * CHANGE:    Initial simplified hook command replacing file-based correlation system
 * RISK:      High - Hook command accuracy affects all session tracking and work time calculations
 */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/claude-monitor/system/internal/utils"
)

// Configuration constants
const (
	DefaultDaemonURL        = "http://localhost:9193"
	DefaultRequestTimeout   = 10 * time.Second
	DefaultRetryAttempts    = 3
	DefaultRetryDelay       = 1 * time.Second
)

// Command-line configuration
type HookConfig struct {
	DaemonURL             string
	HookType              string
	ProcessingDuration    int64  // seconds
	TokenCount            int64
	Success               bool
	ErrorMessage          string
	Command               string
	Debug                 bool
	Timeout               time.Duration
	RetryAttempts         int
	SkipOnDaemonFailure   bool
}

/**
 * CONTEXT:   Main entry point for simplified hook command
 * INPUT:     Command-line arguments with hook configuration
 * OUTPUT:    Exit code (0 for success, non-zero for errors)
 * BUSINESS:  Process Claude Code hook events without manual session ID management
 * CHANGE:    Initial main function with automatic context detection and daemon communication
 * RISK:      Medium - Main function orchestrates all hook processing logic
 */
func main() {
	config, err := parseCommandLineArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing command line: %v\n", err)
		os.Exit(1)
	}

	if config.Debug {
		fmt.Printf("Hook configuration: %+v\n", config)
	}

	// Detect session context automatically
	contextDetector := utils.NewDefaultContextDetector()
	sessionContext, err := contextDetector.DetectSessionContext()
	if err != nil {
		handleError(config, fmt.Sprintf("Failed to detect session context: %v", err))
		return
	}

	if config.Debug {
		fmt.Printf("Detected context: %+v\n", sessionContext)
	}

	// Validate context
	if err := contextDetector.ValidateContext(sessionContext); err != nil {
		handleError(config, fmt.Sprintf("Invalid session context: %v", err))
		return
	}

	// Process hook based on type
	switch config.HookType {
	case "start", "pre_action":
		err = processStartHook(config, sessionContext)
	case "end", "post_action":
		err = processEndHook(config, sessionContext)
	case "activity":
		err = processActivityHook(config, sessionContext)
	default:
		handleError(config, fmt.Sprintf("Unknown hook type: %s", config.HookType))
		return
	}

	if err != nil {
		handleError(config, fmt.Sprintf("Hook processing failed: %v", err))
		return
	}

	if config.Debug {
		fmt.Printf("Hook processed successfully: type=%s\n", config.HookType)
	}
}

/**
 * CONTEXT:   Parse command-line arguments into hook configuration
 * INPUT:     Command-line arguments from os.Args
 * OUTPUT:    HookConfig struct with parsed configuration or error
 * BUSINESS:  Support flexible hook configuration while maintaining simplicity
 * CHANGE:    Initial command-line parsing with sensible defaults
 * RISK:      Low - Configuration parsing with validation and defaults
 */
func parseCommandLineArgs() (*HookConfig, error) {
	config := &HookConfig{
		DaemonURL:           DefaultDaemonURL,
		Timeout:             DefaultRequestTimeout,
		RetryAttempts:       DefaultRetryAttempts,
		SkipOnDaemonFailure: false,
		Success:             true, // Default to success unless specified
	}

	flag.StringVar(&config.DaemonURL, "daemon-url", config.DaemonURL, "Daemon HTTP endpoint URL")
	flag.StringVar(&config.HookType, "type", "activity", "Hook type: start, end, activity")
	flag.Int64Var(&config.ProcessingDuration, "duration", 0, "Processing duration in seconds (for end hooks)")
	flag.Int64Var(&config.TokenCount, "tokens", 0, "Token count (for end hooks)")
	flag.BoolVar(&config.Success, "success", config.Success, "Whether the operation was successful")
	flag.StringVar(&config.ErrorMessage, "error", "", "Error message (if success=false)")
	flag.StringVar(&config.Command, "command", "", "Command that triggered the hook")
	flag.BoolVar(&config.Debug, "debug", false, "Enable debug output")
	flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "Request timeout duration")
	flag.IntVar(&config.RetryAttempts, "retry", config.RetryAttempts, "Number of retry attempts")
	flag.BoolVar(&config.SkipOnDaemonFailure, "skip-on-failure", config.SkipOnDaemonFailure, "Skip hook on daemon failure")

	flag.Parse()

	// Validate required fields
	if config.HookType == "" {
		return nil, fmt.Errorf("hook type is required")
	}

	// Validate hook type
	validTypes := map[string]bool{
		"start":      true,
		"pre_action": true,
		"end":        true,
		"post_action": true,
		"activity":   true,
	}
	if !validTypes[config.HookType] {
		return nil, fmt.Errorf("invalid hook type: %s", config.HookType)
	}

	// Parse environment variables as fallbacks
	if envURL := os.Getenv("CLAUDE_DAEMON_URL"); envURL != "" {
		config.DaemonURL = envURL
	}

	if envDebug := os.Getenv("CLAUDE_HOOK_DEBUG"); envDebug != "" {
		if debug, err := strconv.ParseBool(envDebug); err == nil {
			config.Debug = debug
		}
	}

	return config, nil
}

/**
 * CONTEXT:   Process hook start event by creating active session
 * INPUT:     Hook configuration and detected session context
 * OUTPUT:    HTTP request to daemon session start endpoint, error if failed
 * BUSINESS:  Create active session for daemon-managed correlation without ID passing
 * CHANGE:    Initial start hook processing with session creation
 * RISK:      High - Start hook accuracy affects all subsequent correlation attempts
 */
func processStartHook(config *HookConfig, sessionContext *entities.SessionContext) error {
	startRequest := SessionStartRequest{
		TerminalPID:       sessionContext.TerminalPID,
		ShellPID:          sessionContext.ShellPID,
		WorkingDir:        sessionContext.WorkingDir,
		ProjectPath:       sessionContext.ProjectPath,
		UserID:            sessionContext.UserID,
		Timestamp:         sessionContext.Timestamp,
		EstimatedDuration: 300, // 5 minutes default
		EstimatedTokens:   1000,
		Command:           config.Command,
		Metadata:          map[string]string{
			"hook_type": config.HookType,
			"debug":     strconv.FormatBool(config.Debug),
		},
	}

	endpoint := fmt.Sprintf("%s/api/session/start", config.DaemonURL)
	
	return sendRequestWithRetry(config, "POST", endpoint, startRequest, func(responseBody []byte) error {
		var response SessionStartResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return fmt.Errorf("failed to parse start response: %w", err)
		}

		if config.Debug {
			fmt.Printf("Session started: ID=%s, TerminalPID=%d, Duration=%dms\n", 
				response.SessionID, response.TerminalPID, response.ProcessingMS)
		}

		return nil
	})
}

/**
 * CONTEXT:   Process hook end event by correlating with active session
 * INPUT:     Hook configuration with processing metrics and detected session context
 * OUTPUT:    HTTP request to daemon session end endpoint, error if correlation failed
 * BUSINESS:  Correlate end event with active session using context matching
 * CHANGE:    Initial end hook processing with automatic correlation
 * RISK:      High - End hook correlation accuracy critical for session completion
 */
func processEndHook(config *HookConfig, sessionContext *entities.SessionContext) error {
	endRequest := SessionEndRequest{
		TerminalPID:               sessionContext.TerminalPID,
		ShellPID:                  sessionContext.ShellPID,
		WorkingDir:                sessionContext.WorkingDir,
		ProjectPath:               sessionContext.ProjectPath,
		UserID:                    sessionContext.UserID,
		Timestamp:                 sessionContext.Timestamp,
		ProcessingDurationSeconds: config.ProcessingDuration,
		TokenCount:                config.TokenCount,
		Success:                   config.Success,
		ErrorMessage:              config.ErrorMessage,
		Metadata: map[string]string{
			"hook_type": config.HookType,
			"command":   config.Command,
			"debug":     strconv.FormatBool(config.Debug),
		},
	}

	endpoint := fmt.Sprintf("%s/api/session/end", config.DaemonURL)
	
	return sendRequestWithRetry(config, "POST", endpoint, endRequest, func(responseBody []byte) error {
		var response SessionEndResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return fmt.Errorf("failed to parse end response: %w", err)
		}

		if config.Debug {
			fmt.Printf("Session completed: ActiveID=%s, CompletedID=%s, Duration=%v, Tokens=%d\n", 
				response.ActiveSessionID, response.CompletedSessionID, 
				response.Duration, response.TokenCount)
		}

		return nil
	})
}

/**
 * CONTEXT:   Process activity hook event for general activity tracking
 * INPUT:     Hook configuration and detected session context
 * OUTPUT:    HTTP request to daemon activity endpoint, error if processing failed
 * BUSINESS:  Track general activity events within session context
 * CHANGE:    Initial activity hook processing for general event tracking
 * RISK:      Medium - Activity events used for session activity tracking
 */
func processActivityHook(config *HookConfig, sessionContext *entities.SessionContext) error {
	activityRequest := ActivityEventRequest{
		UserID:         sessionContext.UserID,
		ProjectPath:    sessionContext.ProjectPath,
		ProjectName:    extractProjectName(sessionContext.ProjectPath),
		ActivityType:   "command",
		ActivitySource: "hook",
		Timestamp:      sessionContext.Timestamp,
		Command:        config.Command,
		Description:    fmt.Sprintf("Hook activity: %s", config.HookType),
		Metadata: map[string]string{
			"hook_type":    config.HookType,
			"terminal_pid": strconv.Itoa(sessionContext.TerminalPID),
			"working_dir":  sessionContext.WorkingDir,
		},
	}

	endpoint := fmt.Sprintf("%s/api/activity", config.DaemonURL)
	
	return sendRequestWithRetry(config, "POST", endpoint, activityRequest, func(responseBody []byte) error {
		var response ActivityEventResponse
		if err := json.Unmarshal(responseBody, &response); err != nil {
			return fmt.Errorf("failed to parse activity response: %w", err)
		}

		if config.Debug {
			fmt.Printf("Activity processed: ID=%s, Project=%s, Duration=%dms\n", 
				response.ActivityID, response.ProjectName, response.ProcessingMS)
		}

		return nil
	})
}

/**
 * CONTEXT:   Send HTTP request to daemon with retry logic and error handling
 * INPUT:     HTTP method, endpoint URL, request payload, and response handler
 * OUTPUT:    Error if all retry attempts fail, nil on success
 * BUSINESS:  Provide reliable communication with daemon despite network issues
 * CHANGE:    Initial HTTP client with retry logic and timeout handling
 * RISK:      Medium - Network communication reliability affects hook processing
 */
func sendRequestWithRetry(config *HookConfig, method, endpoint string, payload interface{}, responseHandler func([]byte) error) error {
	// Marshall request payload
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	var lastError error
	for attempt := 0; attempt < config.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(DefaultRetryDelay * time.Duration(attempt)) // Exponential backoff
			if config.Debug {
				fmt.Printf("Retrying request (attempt %d/%d)\n", attempt+1, config.RetryAttempts)
			}
		}

		// Create HTTP client with timeout
		client := &http.Client{
			Timeout: config.Timeout,
		}

		// Create request
		req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(jsonPayload))
		if err != nil {
			lastError = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "claude-hook/1.0")

		// Add context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
		req = req.WithContext(ctx)

		// Send request
		resp, err := client.Do(req)
		cancel()

		if err != nil {
			lastError = fmt.Errorf("HTTP request failed: %w", err)
			continue
		}

		// Read response body
		defer resp.Body.Close()
		responseBody := make([]byte, 0)
		buf := make([]byte, 1024)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				responseBody = append(responseBody, buf[:n]...)
			}
			if readErr != nil {
				break
			}
		}

		// Check HTTP status
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastError = fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(responseBody))
			continue
		}

		// Process response
		if responseHandler != nil {
			err = responseHandler(responseBody)
			if err != nil {
				lastError = fmt.Errorf("response processing failed: %w", err)
				continue
			}
		}

		// Success
		return nil
	}

	return fmt.Errorf("all retry attempts failed, last error: %w", lastError)
}

// Helper functions

func extractProjectName(projectPath string) string {
	if projectPath == "" {
		return "unknown"
	}
	
	// Extract the last component of the path as project name
	parts := strings.Split(projectPath, string(os.PathSeparator))
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	
	return "unknown"
}

func handleError(config *HookConfig, message string) {
	if config.SkipOnDaemonFailure {
		if config.Debug {
			fmt.Printf("Warning (skipped): %s\n", message)
		}
		os.Exit(0) // Exit successfully when skipping on failure
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
		os.Exit(1)
	}
}

// Request/Response types (duplicated for standalone binary)

type SessionStartRequest struct {
	TerminalPID       int               `json:"terminal_pid"`
	ShellPID          int               `json:"shell_pid"`
	WorkingDir        string            `json:"working_dir"`
	ProjectPath       string            `json:"project_path"`
	UserID            string            `json:"user_id"`
	Timestamp         time.Time         `json:"timestamp"`
	EstimatedDuration int               `json:"estimated_duration_seconds,omitempty"`
	EstimatedTokens   int64             `json:"estimated_tokens,omitempty"`
	Command           string            `json:"command,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

type SessionStartResponse struct {
	Status           string    `json:"status"`
	SessionID        string    `json:"session_id"`
	TerminalPID      int       `json:"terminal_pid"`
	UserID           string    `json:"user_id"`
	WorkingDir       string    `json:"working_dir"`
	ProjectPath      string    `json:"project_path"`
	StartTime        time.Time `json:"start_time"`
	EstimatedEndTime time.Time `json:"estimated_end_time"`
	Timestamp        time.Time `json:"timestamp"`
	ProcessingMS     int64     `json:"processing_ms"`
}

type SessionEndRequest struct {
	TerminalPID               int               `json:"terminal_pid"`
	ShellPID                  int               `json:"shell_pid"`
	WorkingDir                string            `json:"working_dir"`
	ProjectPath               string            `json:"project_path"`
	UserID                    string            `json:"user_id"`
	Timestamp                 time.Time         `json:"timestamp"`
	ProcessingDurationSeconds int64             `json:"processing_duration_seconds"`
	TokenCount                int64             `json:"token_count"`
	Success                   bool              `json:"success"`
	ErrorMessage              string            `json:"error_message,omitempty"`
	Metadata                  map[string]string `json:"metadata,omitempty"`
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