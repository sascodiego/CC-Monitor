/**
 * CONTEXT:   Enhanced hook commands with concurrent session correlation support
 * INPUT:     Hook command events with correlation IDs, terminal context, and event types
 * OUTPUT:    Hook command implementations for start/end/activity events with correlation
 * BUSINESS:  Solve concurrent session correlation problem with enhanced hook command support
 * CHANGE:    Initial implementation of correlation-aware hook commands
 * RISK:      High - Hook command accuracy directly affects all concurrent session tracking
 */

package entities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// HookEventType represents different types of hook events for correlation
type HookEventType string

const (
	HookEventStart    HookEventType = "start"     // Claude processing starts
	HookEventEnd      HookEventType = "end"       // Claude processing ends  
	HookEventActivity HookEventType = "activity"  // Regular user activity
	HookEventProgress HookEventType = "progress"  // Claude processing progress
	HookEventHeartbeat HookEventType = "heartbeat" // Keep-alive for long operations
)

/**
 * CONTEXT:   Enhanced hook command configuration with correlation support
 * INPUT:     Hook command parameters including correlation IDs and terminal context
 * OUTPUT:    Hook command configuration for correlation-aware event processing
 * BUSINESS:  Enable precise correlation between start and end events for concurrent sessions
 * CHANGE:    Initial hook command structure with correlation enhancement
 * RISK:      Medium - Hook configuration affects all event processing accuracy
 */
type EnhancedHookCommand struct {
	// Basic hook parameters
	EventType          HookEventType `json:"event_type"`
	DaemonEndpoint     string        `json:"daemon_endpoint"`
	TimeoutSeconds     int           `json:"timeout_seconds"`
	
	// Correlation parameters
	PromptID           string        `json:"prompt_id"`            // Correlation ID
	PromptFile         string        `json:"prompt_file"`          // Temporary prompt file
	AutoCorrelate      bool          `json:"auto_correlate"`       // Enable automatic correlation
	CorrelationTimeout int           `json:"correlation_timeout"`  // Max correlation wait time
	
	// Processing parameters
	EstimatedDuration  string        `json:"estimated_duration"`   // Expected processing time
	ActualDuration     string        `json:"actual_duration"`      // Actual processing time (end events)
	ResponseTokens     int           `json:"response_tokens"`      // Response token count
	ProcessingMetrics  map[string]interface{} `json:"processing_metrics"` // Additional metrics
	
	// Context detection
	DetectTerminal     bool          `json:"detect_terminal"`      // Enable terminal context detection
	ProjectPath        string        `json:"project_path"`         // Override project path
	ForceProjectName   string        `json:"force_project_name"`   // Override project name
	
	// Output and debugging
	Verbose            bool          `json:"verbose"`              // Enable verbose output
	OutputFormat       string        `json:"output_format"`        // json, text, silent
	LogLevel           string        `json:"log_level"`            // debug, info, warn, error
	
	// Internal fields
	terminalDetector   *TerminalContextDetector
	httpClient         *http.Client
}

/**
 * CONTEXT:   Factory for creating enhanced hook command with proper configuration
 * INPUT:     Hook command parameters from command line or configuration
 * OUTPUT:    Configured EnhancedHookCommand ready for event processing
 * BUSINESS:  Initialize hook command with correlation and terminal detection capabilities
 * CHANGE:    Initial factory with comprehensive configuration options
 * RISK:      Medium - Configuration affects hook reliability and correlation accuracy
 */
func NewEnhancedHookCommand(endpoint string, timeoutSeconds int) *EnhancedHookCommand {
	if endpoint == "" {
		endpoint = "http://localhost:9193" // Default daemon endpoint
	}
	
	if timeoutSeconds <= 0 {
		timeoutSeconds = 10 // Default timeout
	}
	
	return &EnhancedHookCommand{
		DaemonEndpoint:     endpoint,
		TimeoutSeconds:     timeoutSeconds,
		AutoCorrelate:      true,
		CorrelationTimeout: 30, // 30 seconds
		DetectTerminal:     true,
		OutputFormat:       "text",
		LogLevel:           "info",
		Verbose:            false,
		
		terminalDetector:   NewTerminalContextDetector(),
		httpClient:         &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
	}
}

/**
 * CONTEXT:   Execute start hook event with correlation ID generation
 * INPUT:     Optional prompt file path and correlation parameters
 * OUTPUT:    Correlation ID for matching with end event, or error
 * BUSINESS:  Begin Claude session tracking with correlation ID for concurrent session support
 * CHANGE:    Initial start hook implementation with correlation generation
 * RISK:      High - Start hook accuracy affects all subsequent session correlation
 */
func (ehc *EnhancedHookCommand) ExecuteStartHook(ctx context.Context, promptFile string) (string, error) {
	if ehc.Verbose {
		fmt.Printf("🚀 Starting Claude session correlation tracking...\n")
	}
	
	// Detect terminal context
	var terminalCtx *TerminalContext
	var err error
	
	if ehc.DetectTerminal {
		terminalCtx, err = ehc.terminalDetector.DetectContext()
		if err != nil {
			if ehc.Verbose {
				fmt.Printf("⚠️  Terminal context detection failed: %v\n", err)
			}
			// Continue without terminal context but log warning
		} else if ehc.Verbose {
			fmt.Printf("📡 Terminal context: PID=%d, Session=%s\n", 
				terminalCtx.PID, terminalCtx.SessionID)
		}
	}
	
	// Read prompt content for correlation
	var promptContent string
	if promptFile != "" {
		content, err := os.ReadFile(promptFile)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file %s: %w", promptFile, err)
		}
		promptContent = string(content)
	}
	
	// Generate correlation ID
	correlationID := ehc.generateCorrelationID(terminalCtx, promptContent)
	
	// Estimate processing time
	estimator := NewProcessingTimeEstimator()
	estimatedDuration := estimator.EstimateProcessingTime(promptContent)
	
	// Create start event
	startEvent := &ClaudeStartEvent{
		PromptID:          correlationID,
		Timestamp:         time.Now(),
		ProjectPath:       ehc.getProjectPath(),
		ProjectName:       ehc.getProjectName(),
		TerminalContext:   terminalCtx,
		PromptContent:     promptContent,
		EstimatedDuration: estimatedDuration,
		UserID:            ehc.getUserID(),
		ProcessingMetrics: ehc.ProcessingMetrics,
	}
	
	// Send to daemon
	err = ehc.sendStartEventToDaemon(ctx, startEvent)
	if err != nil {
		return "", fmt.Errorf("failed to send start event to daemon: %w", err)
	}
	
	if ehc.Verbose {
		fmt.Printf("✅ Start event sent: ID=%s, Estimated=%v\n", 
			correlationID, estimatedDuration)
	}
	
	// Store correlation ID for potential use by end event
	ehc.storeCorrelationID(correlationID)
	
	return correlationID, nil
}

/**
 * CONTEXT:   Execute end hook event with correlation matching
 * INPUT:     Correlation ID (optional), actual duration, and response metrics
 * OUTPUT:    Matched session information or correlation error
 * BUSINESS:  End Claude session with accurate timing using correlation matching
 * CHANGE:    Initial end hook implementation with intelligent correlation
 * RISK:      High - End hook correlation accuracy affects time tracking precision
 */
func (ehc *EnhancedHookCommand) ExecuteEndHook(ctx context.Context, promptID string, actualDuration time.Duration, responseTokens int) error {
	if ehc.Verbose {
		fmt.Printf("🏁 Ending Claude session with correlation...\n")
	}
	
	// Use provided prompt ID or try to detect from stored correlation
	correlationID := promptID
	if correlationID == "" && ehc.AutoCorrelate {
		storedID, err := ehc.retrieveStoredCorrelationID()
		if err == nil {
			correlationID = storedID
			if ehc.Verbose {
				fmt.Printf("📎 Using stored correlation ID: %s\n", correlationID)
			}
		}
	}
	
	// Detect terminal context for correlation fallback
	var terminalCtx *TerminalContext
	var err error
	
	if ehc.DetectTerminal {
		terminalCtx, err = ehc.terminalDetector.DetectContext()
		if err != nil && ehc.Verbose {
			fmt.Printf("⚠️  Terminal context detection failed: %v\n", err)
		}
	}
	
	// Create end event
	endEvent := &ClaudeEndEvent{
		PromptID:          correlationID,
		Timestamp:         time.Now(),
		ActualDuration:    actualDuration,
		ProjectPath:       ehc.getProjectPath(),
		ProjectName:       ehc.getProjectName(),
		TerminalContext:   terminalCtx,
		ResponseTokens:    &responseTokens,
		ProcessingMetrics: ehc.ProcessingMetrics,
		UserID:            ehc.getUserID(),
	}
	
	// Send to daemon
	err = ehc.sendEndEventToDaemon(ctx, endEvent)
	if err != nil {
		return fmt.Errorf("failed to send end event to daemon: %w", err)
	}
	
	if ehc.Verbose {
		if correlationID != "" {
			fmt.Printf("✅ End event sent: ID=%s, Duration=%v, Tokens=%d\n", 
				correlationID, actualDuration, responseTokens)
		} else {
			fmt.Printf("✅ End event sent: Duration=%v, Tokens=%d (auto-correlation)\n", 
				actualDuration, responseTokens)
		}
	}
	
	// Clean up stored correlation ID
	ehc.cleanupStoredCorrelationID()
	
	return nil
}

/**
 * CONTEXT:   Execute regular activity hook for non-Claude activity
 * INPUT:     Activity context and optional project information
 * OUTPUT:    Activity event processing result
 * BUSINESS:  Track regular user activity that doesn't involve Claude processing
 * CHANGE:    Initial activity hook implementation
 * RISK:      Low - Activity hook provides supplementary activity tracking
 */
func (ehc *EnhancedHookCommand) ExecuteActivityHook(ctx context.Context) error {
	if ehc.Verbose {
		fmt.Printf("📊 Recording regular activity...\n")
	}
	
	// Detect terminal context
	var terminalCtx *TerminalContext
	if ehc.DetectTerminal {
		var err error
		terminalCtx, err = ehc.terminalDetector.DetectContext()
		if err != nil && ehc.Verbose {
			fmt.Printf("⚠️  Terminal context detection failed: %v\n", err)
		}
	}
	
	// Create activity event
	activityEvent, err := NewActivityEvent(ActivityEventConfig{
		UserID:         ehc.getUserID(),
		ProjectPath:    ehc.getProjectPath(),
		ActivityType:   ActivityTypeInteraction,
		ActivitySource: ActivitySourceHook,
		Timestamp:      time.Now(),
		Command:        "hook_activity",
		Description:    "Regular user activity",
		Metadata:       map[string]string{
			"hook_type":       string(HookEventActivity),
			"terminal_pid":    strconv.Itoa(terminalCtx.PID),
			"session_id":      terminalCtx.SessionID,
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to create activity event: %w", err)
	}
	
	// Send to daemon
	err = ehc.sendActivityEventToDaemon(ctx, activityEvent)
	if err != nil {
		return fmt.Errorf("failed to send activity event to daemon: %w", err)
	}
	
	if ehc.Verbose {
		fmt.Printf("✅ Activity event recorded\n")
	}
	
	return nil
}

/**
 * CONTEXT:   Generate unique correlation ID for session matching
 * INPUT:     Terminal context and prompt content for ID generation
 * OUTPUT:    Unique correlation ID for start/end event matching
 * BUSINESS:  Create deterministic correlation IDs for reliable session matching
 * CHANGE:    Initial correlation ID generation with collision avoidance
 * RISK:      Medium - Correlation ID uniqueness critical for accurate session matching
 */
func (ehc *EnhancedHookCommand) generateCorrelationID(terminalCtx *TerminalContext, promptContent string) string {
	// Create correlation ID based on terminal context + timestamp + prompt hash
	timestamp := time.Now().Unix()
	
	var contextData string
	if terminalCtx != nil {
		contextData = fmt.Sprintf("%d_%d_%s_%d", 
			terminalCtx.PID, 
			terminalCtx.ShellPID, 
			terminalCtx.SessionID, 
			timestamp)
	} else {
		// Fallback when terminal context unavailable
		contextData = fmt.Sprintf("%d_%s_%d", 
			os.Getpid(), 
			ehc.getProjectPath(), 
			timestamp)
	}
	
	// Add prompt hash for uniqueness
	promptHash := ehc.hashPrompt(promptContent)
	
	// Create deterministic but unique ID
	fullHash := sha256.Sum256([]byte(contextData + promptHash))
	shortHash := fmt.Sprintf("%x", fullHash)[:16] // First 16 chars
	
	return fmt.Sprintf("claude_%d_%s", timestamp, shortHash)
}

/**
 * CONTEXT:   Send start event to daemon with correlation data
 * INPUT:     Context and Claude start event with correlation information
 * OUTPUT:    HTTP response from daemon or error
 * BUSINESS:  Communicate start event to daemon for session correlation tracking
 * CHANGE:    Initial daemon communication for start events
 * RISK:      Medium - Network communication reliability affects correlation accuracy
 */
func (ehc *EnhancedHookCommand) sendStartEventToDaemon(ctx context.Context, startEvent *ClaudeStartEvent) error {
	endpoint := fmt.Sprintf("%s/api/v1/claude/start", ehc.DaemonEndpoint)
	
	eventData, err := json.Marshal(startEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal start event: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(eventData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "claude-monitor-hook/1.0")
	
	resp, err := ehc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned error %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

/**
 * CONTEXT:   Send end event to daemon for correlation matching
 * INPUT:     Context and Claude end event with correlation and timing data
 * OUTPUT:    HTTP response from daemon or error
 * BUSINESS:  Communicate end event to daemon for session correlation and completion
 * CHANGE:    Initial daemon communication for end events
 * RISK:      Medium - Network communication reliability affects session completion
 */
func (ehc *EnhancedHookCommand) sendEndEventToDaemon(ctx context.Context, endEvent *ClaudeEndEvent) error {
	endpoint := fmt.Sprintf("%s/api/v1/claude/end", ehc.DaemonEndpoint)
	
	eventData, err := json.Marshal(endEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal end event: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(eventData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "claude-monitor-hook/1.0")
	
	resp, err := ehc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned error %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

/**
 * CONTEXT:   Send regular activity event to daemon
 * INPUT:     Context and activity event for non-Claude activity tracking
 * OUTPUT:    HTTP response from daemon or error
 * BUSINESS:  Communicate regular activity to daemon for comprehensive work tracking
 * CHANGE:    Initial daemon communication for activity events
 * RISK:      Low - Activity event communication is supplementary
 */
func (ehc *EnhancedHookCommand) sendActivityEventToDaemon(ctx context.Context, activityEvent *ActivityEvent) error {
	endpoint := fmt.Sprintf("%s/api/v1/activity", ehc.DaemonEndpoint)
	
	eventData, err := json.Marshal(activityEvent.ToData())
	if err != nil {
		return fmt.Errorf("failed to marshal activity event: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(eventData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "claude-monitor-hook/1.0")
	
	resp, err := ehc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon returned error %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

/**
 * CONTEXT:   Store correlation ID for retrieval by end event
 * INPUT:     Correlation ID to store for later retrieval
 * OUTPUT:    No return value, stores ID in temporary location
 * BUSINESS:  Enable correlation ID sharing between start and end hook executions
 * CHANGE:    Initial correlation ID storage implementation
 * RISK:      Low - Temporary storage for correlation ID persistence
 */
func (ehc *EnhancedHookCommand) storeCorrelationID(correlationID string) {
	// Store in temporary file for retrieval by end hook
	tempDir := os.TempDir()
	correlationFile := filepath.Join(tempDir, fmt.Sprintf("claude-monitor-correlation-%d.txt", os.Getpid()))
	
	err := os.WriteFile(correlationFile, []byte(correlationID), 0600)
	if err != nil && ehc.Verbose {
		fmt.Printf("⚠️  Failed to store correlation ID: %v\n", err)
	}
}

/**
 * CONTEXT:   Retrieve stored correlation ID from temporary storage
 * INPUT:     No parameters, reads from temporary storage location
 * OUTPUT:    Stored correlation ID or error if not found
 * BUSINESS:  Retrieve correlation ID for end event matching when not explicitly provided
 * CHANGE:    Initial correlation ID retrieval implementation
 * RISK:      Low - Correlation ID retrieval for automatic matching
 */
func (ehc *EnhancedHookCommand) retrieveStoredCorrelationID() (string, error) {
	tempDir := os.TempDir()
	correlationFile := filepath.Join(tempDir, fmt.Sprintf("claude-monitor-correlation-%d.txt", os.Getpid()))
	
	data, err := os.ReadFile(correlationFile)
	if err != nil {
		return "", fmt.Errorf("no stored correlation ID found: %w", err)
	}
	
	return strings.TrimSpace(string(data)), nil
}

/**
 * CONTEXT:   Clean up stored correlation ID after successful correlation
 * INPUT:     No parameters, removes temporary correlation storage
 * OUTPUT:    No return value, cleans up temporary files
 * BUSINESS:  Clean up temporary correlation data to prevent conflicts
 * CHANGE:    Initial correlation cleanup implementation
 * RISK:      Low - Cleanup prevents correlation ID conflicts
 */
func (ehc *EnhancedHookCommand) cleanupStoredCorrelationID() {
	tempDir := os.TempDir()
	correlationFile := filepath.Join(tempDir, fmt.Sprintf("claude-monitor-correlation-%d.txt", os.Getpid()))
	
	err := os.Remove(correlationFile)
	if err != nil && ehc.Verbose {
		// Only log in verbose mode - cleanup failure is not critical
		fmt.Printf("ℹ️  Failed to cleanup correlation file: %v\n", err)
	}
}

// Helper methods for hook command support

func (ehc *EnhancedHookCommand) getProjectPath() string {
	if ehc.ProjectPath != "" {
		return ehc.ProjectPath
	}
	
	workingDir, err := os.Getwd()
	if err != nil {
		return ""
	}
	
	return workingDir
}

func (ehc *EnhancedHookCommand) getProjectName() string {
	if ehc.ForceProjectName != "" {
		return ehc.ForceProjectName
	}
	
	projectPath := ehc.getProjectPath()
	if projectPath == "" {
		return "unknown"
	}
	
	return filepath.Base(projectPath)
}

func (ehc *EnhancedHookCommand) getUserID() string {
	userID := os.Getenv("USER")
	if userID == "" {
		userID = os.Getenv("USERNAME")
	}
	if userID == "" {
		userID = "unknown-user"
	}
	return userID
}

func (ehc *EnhancedHookCommand) hashPrompt(prompt string) string {
	// Normalize prompt for consistent hashing
	normalized := strings.TrimSpace(strings.ToLower(prompt))
	if len(normalized) > 500 {
		normalized = normalized[:500] // Limit length for consistency
	}
	
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", hash)[:16] // First 16 characters
}

/**
 * CONTEXT:   Claude start event structure for daemon communication
 * INPUT:     Start event data with correlation and timing information
 * OUTPUT:    Complete start event for session tracking initialization
 * BUSINESS:  Provide all necessary data for starting concurrent session tracking
 * CHANGE:    Initial start event structure for correlation system
 * RISK:      Medium - Start event completeness affects session correlation accuracy
 */
type ClaudeStartEvent struct {
	PromptID          string                 `json:"prompt_id"`
	Timestamp         time.Time              `json:"timestamp"`
	ProjectPath       string                 `json:"project_path"`
	ProjectName       string                 `json:"project_name"`
	TerminalContext   *TerminalContext       `json:"terminal_context"`
	PromptContent     string                 `json:"prompt_content"`      // For processing estimation
	EstimatedDuration time.Duration          `json:"estimated_duration"`
	UserID            string                 `json:"user_id"`
	ProcessingMetrics map[string]interface{} `json:"processing_metrics"`
}