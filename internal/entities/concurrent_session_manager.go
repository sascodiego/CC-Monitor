/**
 * CONTEXT:   Concurrent session management for tracking multiple simultaneous Claude processes
 * INPUT:     Start/end events, terminal context, and correlation data for multiple Claude sessions
 * OUTPUT:    Session correlation, state management, and accurate time tracking across processes
 * BUSINESS:  Solve the critical problem of matching start/end events for concurrent Claude sessions
 * CHANGE:    Initial implementation of comprehensive concurrent session correlation system
 * RISK:      High - Core logic that affects accuracy of all concurrent session tracking
 */

package entities

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionCorrelationState represents the state of session correlation tracking
type SessionCorrelationState string

const (
	CorrelationStateActive    SessionCorrelationState = "active"
	CorrelationStateMatched   SessionCorrelationState = "matched"
	CorrelationStateOrphaned  SessionCorrelationState = "orphaned"
	CorrelationStateTimedOut  SessionCorrelationState = "timed_out"
)

// CorrelationConfidenceLevel indicates how confident we are about session correlation
type CorrelationConfidenceLevel float64

const (
	ConfidenceVeryHigh CorrelationConfidenceLevel = 0.95 // Direct ID match
	ConfidenceHigh     CorrelationConfidenceLevel = 0.85 // Strong terminal + timing match
	ConfidenceMedium   CorrelationConfidenceLevel = 0.70 // Good timing + project match
	ConfidenceLow      CorrelationConfidenceLevel = 0.50 // Weak correlation
	ConfidenceNone     CorrelationConfidenceLevel = 0.00 // No correlation found
)

/**
 * CONTEXT:   Terminal and process context for session correlation
 * INPUT:     Process IDs, terminal session data, and working directory information
 * OUTPUT:    Complete terminal context for correlation matching
 * BUSINESS:  Enable precise correlation by tracking terminal and process hierarchy
 * CHANGE:    Initial terminal context structure for multi-factor correlation
 * RISK:      Medium - Terminal detection reliability varies across systems
 */
type TerminalContext struct {
	PID           int               `json:"pid"`            // Parent process ID
	ShellPID      int               `json:"shell_pid"`      // Shell process ID
	SessionID     string            `json:"session_id"`     // Terminal session identifier
	WorkingDir    string            `json:"working_dir"`    // Current directory
	Environment   map[string]string `json:"environment"`    // Relevant env vars
	DetectedAt    time.Time         `json:"detected_at"`    // When context was captured
	HostName      string            `json:"host_name"`      // Machine hostname
	TerminalType  string            `json:"terminal_type"`  // Terminal type (xterm, etc)
	WindowID      string            `json:"window_id"`      // Window/tab identifier if available
}

/**
 * CONTEXT:   Active Claude session being tracked for correlation
 * INPUT:     Session start data, terminal context, and processing estimation
 * OUTPUT:    Complete session tracking with correlation capabilities
 * BUSINESS:  Track concurrent Claude sessions with precise correlation data
 * CHANGE:    Initial active session structure with enhanced correlation fields
 * RISK:      High - Session tracking accuracy directly affects time calculations
 */
type ActiveClaudeSession struct {
	ID                    string                      `json:"id"`
	PromptID              string                      `json:"prompt_id"`              // Correlation ID for matching
	StartTime             time.Time                   `json:"start_time"`
	EstimatedEndTime      time.Time                   `json:"estimated_end_time"`
	ProjectPath           string                      `json:"project_path"`
	ProjectName           string                      `json:"project_name"`
	TerminalContext       *TerminalContext            `json:"terminal_context"`
	PromptHash            string                      `json:"prompt_hash"`            // Hash of prompt content
	UserID                string                      `json:"user_id"`
	WorkBlockID           string                      `json:"work_block_id"`
	EstimatedDuration     time.Duration               `json:"estimated_duration"`
	ActualDuration        *time.Duration              `json:"actual_duration"`
	EndTime               *time.Time                  `json:"end_time"`
	State                 SessionCorrelationState     `json:"state"`
	CorrelationAttempts   int                         `json:"correlation_attempts"`
	LastCorrelationUpdate time.Time                   `json:"last_correlation_update"`
	CreatedAt             time.Time                   `json:"created_at"`
	
	// Correlation scoring factors
	TerminalWeight        float64                     `json:"terminal_weight"`        // Weight for terminal matching
	TimingWeight          float64                     `json:"timing_weight"`          // Weight for timing accuracy
	ProjectWeight         float64                     `json:"project_weight"`         // Weight for project matching
	PromptWeight          float64                     `json:"prompt_weight"`          // Weight for prompt similarity
}

/**
 * CONTEXT:   Claude end event for correlation with active sessions
 * INPUT:     End event data including timing, terminal context, and optional correlation ID
 * OUTPUT:    Complete end event structure for correlation matching
 * BUSINESS:  Provide all necessary data for accurate session correlation
 * CHANGE:    Initial end event structure for correlation system
 * RISK:      Medium - End event data quality affects correlation accuracy
 */
type ClaudeEndEvent struct {
	PromptID          string           `json:"prompt_id"`           // Optional correlation ID
	Timestamp         time.Time        `json:"timestamp"`
	ActualDuration    time.Duration    `json:"actual_duration"`
	EstimatedDuration time.Duration    `json:"estimated_duration"`   // From processing time estimator
	ProjectPath       string           `json:"project_path"`
	ProjectName       string           `json:"project_name"`
	TerminalContext   *TerminalContext `json:"terminal_context"`
	ResponseTokens    *int             `json:"response_tokens"`      // If available
	ProcessingMetrics map[string]interface{} `json:"processing_metrics"` // Additional metrics
	UserID            string           `json:"user_id"`
}

/**
 * CONTEXT:   Core concurrent session manager handling multiple Claude processes
 * INPUT:     Start/end events, terminal contexts, and correlation configuration
 * OUTPUT:    Session management, correlation, and accurate time tracking
 * BUSINESS:  Solve concurrent session correlation problem with multi-factor matching
 * CHANGE:    Initial implementation of concurrent session correlation system
 * RISK:      High - Core business logic affecting all concurrent session tracking
 */
type ConcurrentSessionManager struct {
	activeSessions        map[string]*ActiveClaudeSession // sessionID -> session
	sessionsByPromptID    map[string]*ActiveClaudeSession // promptID -> session  
	sessionsByTerminal    map[string][]*ActiveClaudeSession // terminalID -> sessions
	sessionHistory        []*ActiveClaudeSession           // Completed sessions
	orphanedEndEvents     []*ClaudeEndEvent                // Unmatched end events
	
	mutex                 sync.RWMutex
	sessionDuration       time.Duration                    // Max session age
	maxSessionAge         time.Duration                    // Cleanup threshold
	cleanupInterval       time.Duration                    // Background cleanup interval
	correlationTimeout    time.Duration                    // Max time to wait for correlation
	
	// Correlation weights (sum should equal 1.0)
	terminalWeight        float64                          // 40% - Terminal PID matching weight
	timingWeight          float64                          // 30% - Timing accuracy weight
	projectWeight         float64                          // 20% - Project path matching weight  
	promptWeight          float64                          // 10% - Prompt similarity weight
	
	// Background cleanup
	stopCleanup          chan bool
	cleanupRunning       bool
}

/**
 * CONTEXT:   Factory for creating concurrent session manager with proper configuration
 * INPUT:     Session duration, correlation weights, and cleanup settings
 * OUTPUT:    Configured ConcurrentSessionManager ready for session tracking
 * BUSINESS:  Initialize concurrent session manager with business rule compliance
 * CHANGE:    Initial factory with comprehensive configuration options
 * RISK:      Medium - Configuration affects correlation accuracy and performance
 */
func NewConcurrentSessionManager(sessionDuration time.Duration) *ConcurrentSessionManager {
	if sessionDuration == 0 {
		sessionDuration = 5 * time.Hour // Default Claude session duration
	}
	
	csm := &ConcurrentSessionManager{
		activeSessions:     make(map[string]*ActiveClaudeSession),
		sessionsByPromptID: make(map[string]*ActiveClaudeSession),
		sessionsByTerminal: make(map[string][]*ActiveClaudeSession),
		sessionHistory:     make([]*ActiveClaudeSession, 0),
		orphanedEndEvents:  make([]*ClaudeEndEvent, 0),
		
		sessionDuration:    sessionDuration,
		maxSessionAge:      24 * time.Hour,      // Clean up old sessions after 24h
		cleanupInterval:    5 * time.Minute,     // Run cleanup every 5 minutes
		correlationTimeout: 30 * time.Minute,    // Max time to wait for correlation
		
		// Correlation weights (total = 1.0)
		terminalWeight:     0.40, // Terminal matching is most reliable
		timingWeight:       0.30, // Timing accuracy is very important
		projectWeight:      0.20, // Project context helps
		promptWeight:       0.10, // Prompt similarity is helpful but variable
		
		stopCleanup:        make(chan bool),
		cleanupRunning:     false,
	}
	
	// Start background cleanup
	go csm.runBackgroundCleanup()
	
	return csm
}

/**
 * CONTEXT:   Start tracking a new Claude session with correlation support
 * INPUT:     Terminal context, prompt content, and session configuration
 * OUTPUT:    Active Claude session with correlation ID and tracking data
 * BUSINESS:  Begin tracking new Claude session with proper correlation setup
 * CHANGE:    Initial session start implementation with correlation support
 * RISK:      High - Session start accuracy affects all subsequent correlation
 */
func (csm *ConcurrentSessionManager) StartClaudeSession(ctx context.Context, terminalCtx *TerminalContext, prompt string, promptID string) (*ActiveClaudeSession, error) {
	csm.mutex.Lock()
	defer csm.mutex.Unlock()
	
	// Generate session ID and prompt ID if needed
	sessionID := uuid.New().String()
	if promptID == "" {
		promptID = csm.generatePromptID(terminalCtx, prompt)
	}
	
	// Create processing time estimator
	estimator := NewProcessingTimeEstimator()
	estimatedDuration := estimator.EstimateProcessingTime(prompt)
	
	now := time.Now()
	
	session := &ActiveClaudeSession{
		ID:                    sessionID,
		PromptID:              promptID,
		StartTime:             now,
		EstimatedEndTime:      now.Add(estimatedDuration),
		ProjectPath:           terminalCtx.WorkingDir,
		ProjectName:           filepath.Base(terminalCtx.WorkingDir),
		TerminalContext:       terminalCtx,
		PromptHash:            csm.hashPrompt(prompt),
		UserID:                terminalCtx.Environment["USER"],
		EstimatedDuration:     estimatedDuration,
		State:                 CorrelationStateActive,
		CorrelationAttempts:   0,
		LastCorrelationUpdate: now,
		CreatedAt:             now,
		
		// Set correlation weights from manager
		TerminalWeight:        csm.terminalWeight,
		TimingWeight:          csm.timingWeight,
		ProjectWeight:         csm.projectWeight,
		PromptWeight:          csm.promptWeight,
	}
	
	// Store in all tracking maps
	csm.activeSessions[sessionID] = session
	csm.sessionsByPromptID[promptID] = session
	
	// Add to terminal tracking
	terminalID := csm.getTerminalID(terminalCtx)
	if sessions, exists := csm.sessionsByTerminal[terminalID]; exists {
		csm.sessionsByTerminal[terminalID] = append(sessions, session)
	} else {
		csm.sessionsByTerminal[terminalID] = []*ActiveClaudeSession{session}
	}
	
	return session, nil
}

/**
 * CONTEXT:   End Claude session with correlation matching for concurrent processes
 * INPUT:     Claude end event with timing and optional correlation ID
 * OUTPUT:    Matched session with accurate timing, or correlation error
 * BUSINESS:  Match end events to correct Claude sessions using multi-factor correlation
 * CHANGE:    Initial end session implementation with intelligent correlation
 * RISK:      High - Correlation accuracy directly affects time tracking precision
 */
func (csm *ConcurrentSessionManager) EndClaudeSession(ctx context.Context, endEvent *ClaudeEndEvent) (*ActiveClaudeSession, error) {
	csm.mutex.Lock()
	defer csm.mutex.Unlock()
	
	// Method 1: Direct prompt ID correlation (preferred)
	if endEvent.PromptID != "" {
		if session, exists := csm.sessionsByPromptID[endEvent.PromptID]; exists {
			return csm.finalizeSession(session, endEvent)
		}
	}
	
	// Method 2: Multi-factor correlation
	bestMatch, confidence := csm.findBestSessionMatch(endEvent)
	
	if bestMatch == nil || confidence < ConfidenceLow {
		// Store as orphaned event for potential later correlation
		csm.orphanedEndEvents = append(csm.orphanedEndEvents, endEvent)
		return nil, fmt.Errorf("no confident correlation found for end event (confidence: %.2f)", confidence)
	}
	
	// Log correlation confidence for monitoring
	if confidence < ConfidenceHigh {
		// Could log warning about lower confidence correlation
	}
	
	return csm.finalizeSession(bestMatch, endEvent)
}

/**
 * CONTEXT:   Multi-factor correlation algorithm for matching end events to active sessions
 * INPUT:     Claude end event with all available correlation data
 * OUTPUT:    Best matching session and confidence level
 * BUSINESS:  Intelligent correlation using terminal, timing, project, and prompt data
 * CHANGE:    Initial multi-factor correlation algorithm implementation
 * RISK:      High - Correlation algorithm accuracy affects session matching reliability
 */
func (csm *ConcurrentSessionManager) findBestSessionMatch(endEvent *ClaudeEndEvent) (*ActiveClaudeSession, CorrelationConfidenceLevel) {
	var bestMatch *ActiveClaudeSession
	var bestScore CorrelationConfidenceLevel = ConfidenceNone
	
	for _, session := range csm.activeSessions {
		if session.State != CorrelationStateActive {
			continue // Skip non-active sessions
		}
		
		score := csm.calculateCorrelationScore(session, endEvent)
		if score > bestScore {
			bestScore = score
			bestMatch = session
		}
	}
	
	return bestMatch, bestScore
}

/**
 * CONTEXT:   Calculate correlation score between session and end event
 * INPUT:     Active session and end event for correlation scoring
 * OUTPUT:    Confidence score (0.0 to 1.0) indicating correlation strength
 * BUSINESS:  Weighted scoring algorithm using terminal, timing, project, and prompt factors
 * CHANGE:    Initial correlation scoring algorithm with configurable weights
 * RISK:      High - Scoring accuracy determines correlation reliability
 */
func (csm *ConcurrentSessionManager) calculateCorrelationScore(session *ActiveClaudeSession, endEvent *ClaudeEndEvent) CorrelationConfidenceLevel {
	var totalScore float64 = 0.0
	
	// Factor 1: Terminal context matching (40% weight)
	terminalScore := csm.calculateTerminalScore(session.TerminalContext, endEvent.TerminalContext)
	totalScore += terminalScore * session.TerminalWeight
	
	// Factor 2: Timing accuracy (30% weight)
	timingScore := csm.calculateTimingScore(session, endEvent)
	totalScore += timingScore * session.TimingWeight
	
	// Factor 3: Project context matching (20% weight)
	projectScore := csm.calculateProjectScore(session, endEvent)
	totalScore += projectScore * session.ProjectWeight
	
	// Factor 4: Prompt similarity (10% weight)
	promptScore := csm.calculatePromptScore(session, endEvent)
	totalScore += promptScore * session.PromptWeight
	
	return CorrelationConfidenceLevel(totalScore)
}

/**
 * CONTEXT:   Calculate terminal context matching score
 * INPUT:     Session terminal context and end event terminal context
 * OUTPUT:    Score (0.0 to 1.0) indicating terminal context similarity
 * BUSINESS:  Terminal matching provides highest confidence correlation factor
 * CHANGE:    Initial terminal context scoring implementation
 * RISK:      Medium - Terminal detection reliability varies across systems
 */
func (csm *ConcurrentSessionManager) calculateTerminalScore(sessionTerminal, eventTerminal *TerminalContext) float64 {
	if sessionTerminal == nil || eventTerminal == nil {
		return 0.0
	}
	
	score := 0.0
	factors := 0
	
	// PID matching (most reliable)
	if sessionTerminal.PID != 0 && eventTerminal.PID != 0 {
		if sessionTerminal.PID == eventTerminal.PID {
			score += 0.4 // 40% for exact PID match
		}
		factors++
	}
	
	// Shell PID matching
	if sessionTerminal.ShellPID != 0 && eventTerminal.ShellPID != 0 {
		if sessionTerminal.ShellPID == eventTerminal.ShellPID {
			score += 0.3 // 30% for shell PID match
		}
		factors++
	}
	
	// Session ID matching
	if sessionTerminal.SessionID != "" && eventTerminal.SessionID != "" {
		if sessionTerminal.SessionID == eventTerminal.SessionID {
			score += 0.2 // 20% for session ID match
		}
		factors++
	}
	
	// Hostname matching
	if sessionTerminal.HostName != "" && eventTerminal.HostName != "" {
		if sessionTerminal.HostName == eventTerminal.HostName {
			score += 0.1 // 10% for hostname match
		}
		factors++
	}
	
	// Normalize by number of factors checked
	if factors > 0 {
		return score
	}
	
	return 0.0
}

/**
 * CONTEXT:   Calculate timing accuracy score for session correlation
 * INPUT:     Active session with timing estimates and end event with actual timing
 * OUTPUT:    Score (0.0 to 1.0) indicating timing prediction accuracy
 * BUSINESS:  Timing accuracy helps distinguish between concurrent sessions
 * CHANGE:    Initial timing scoring with estimation accuracy evaluation
 * RISK:      Medium - Timing estimates may vary significantly based on prompt complexity
 */
func (csm *ConcurrentSessionManager) calculateTimingScore(session *ActiveClaudeSession, endEvent *ClaudeEndEvent) float64 {
	// Calculate timing accuracy
	estimatedDuration := session.EstimatedDuration
	actualDuration := endEvent.ActualDuration
	
	if estimatedDuration == 0 || actualDuration == 0 {
		return 0.0
	}
	
	// Calculate accuracy as 1 - (relative error)
	var accuracy float64
	if estimatedDuration > actualDuration {
		accuracy = float64(actualDuration) / float64(estimatedDuration)
	} else {
		accuracy = float64(estimatedDuration) / float64(actualDuration)
	}
	
	// Timing must be reasonably close (within 3x) to be considered valid
	if accuracy < 0.33 { // More than 3x off
		return 0.0
	}
	
	// Additional factor: session age appropriateness
	sessionAge := endEvent.Timestamp.Sub(session.StartTime)
	expectedAge := session.EstimatedDuration
	
	var ageAccuracy float64
	if sessionAge > expectedAge {
		ageAccuracy = float64(expectedAge) / float64(sessionAge)
	} else {
		ageAccuracy = float64(sessionAge) / float64(expectedAge)
	}
	
	// Combine timing accuracy and age appropriateness
	return (accuracy * 0.7) + (ageAccuracy * 0.3)
}

/**
 * CONTEXT:   Calculate project context matching score
 * INPUT:     Session project info and end event project info
 * OUTPUT:    Score (0.0 to 1.0) indicating project context similarity
 * BUSINESS:  Project matching helps correlation when multiple Claude sessions in different projects
 * CHANGE:    Initial project context scoring implementation
 * RISK:      Low - Project matching is supplementary correlation factor
 */
func (csm *ConcurrentSessionManager) calculateProjectScore(session *ActiveClaudeSession, endEvent *ClaudeEndEvent) float64 {
	// Exact project path match
	if session.ProjectPath != "" && endEvent.ProjectPath != "" {
		if session.ProjectPath == endEvent.ProjectPath {
			return 1.0 // Perfect match
		}
	}
	
	// Project name match (less precise)
	if session.ProjectName != "" && endEvent.ProjectName != "" {
		if session.ProjectName == endEvent.ProjectName {
			return 0.8 // Good match
		}
	}
	
	// Directory similarity (parent directories)
	if session.ProjectPath != "" && endEvent.ProjectPath != "" {
		sessionDir := filepath.Dir(session.ProjectPath)
		eventDir := filepath.Dir(endEvent.ProjectPath)
		if sessionDir == eventDir {
			return 0.6 // Same parent directory
		}
	}
	
	return 0.0
}

/**
 * CONTEXT:   Calculate prompt similarity score for correlation
 * INPUT:     Session prompt hash and end event context
 * OUTPUT:    Score (0.0 to 1.0) indicating prompt similarity
 * BUSINESS:  Prompt similarity provides additional correlation confidence
 * CHANGE:    Initial prompt similarity scoring implementation
 * RISK:      Low - Prompt similarity is lowest weight correlation factor
 */
func (csm *ConcurrentSessionManager) calculatePromptScore(session *ActiveClaudeSession, endEvent *ClaudeEndEvent) float64 {
	// This is a simplified implementation
	// In practice, more sophisticated similarity measures could be used
	
	// For now, we can't easily compare prompts since end event might not have the original prompt
	// This could be enhanced by storing prompt hashes or using other indicators
	
	// Placeholder implementation - could use response token count correlation, etc.
	if session.PromptHash != "" {
		// Could implement prompt similarity here if we had access to both prompts
		return 0.5 // Neutral score when we can't determine similarity
	}
	
	return 0.0
}

/**
 * CONTEXT:   Finalize matched session with end event data
 * INPUT:     Matched active session and end event with completion data
 * OUTPUT:    Completed session with accurate timing and state updates
 * BUSINESS:  Complete session tracking with final timing and state management
 * CHANGE:    Initial session finalization with state cleanup
 * RISK:      Medium - Session finalization affects final time calculations
 */
func (csm *ConcurrentSessionManager) finalizeSession(session *ActiveClaudeSession, endEvent *ClaudeEndEvent) (*ActiveClaudeSession, error) {
	// Update session with final data
	now := time.Now()
	session.EndTime = &endEvent.Timestamp
	session.ActualDuration = &endEvent.ActualDuration
	session.State = CorrelationStateMatched
	session.LastCorrelationUpdate = now
	
	// Remove from active tracking
	delete(csm.activeSessions, session.ID)
	delete(csm.sessionsByPromptID, session.PromptID)
	
	// Remove from terminal tracking
	terminalID := csm.getTerminalID(session.TerminalContext)
	if sessions, exists := csm.sessionsByTerminal[terminalID]; exists {
		// Remove session from terminal list
		for i, s := range sessions {
			if s.ID == session.ID {
				csm.sessionsByTerminal[terminalID] = append(sessions[:i], sessions[i+1:]...)
				break
			}
		}
		// Clean up empty terminal lists
		if len(csm.sessionsByTerminal[terminalID]) == 0 {
			delete(csm.sessionsByTerminal, terminalID)
		}
	}
	
	// Add to history
	csm.sessionHistory = append(csm.sessionHistory, session)
	
	return session, nil
}

/**
 * CONTEXT:   Generate unique prompt ID for correlation tracking
 * INPUT:     Terminal context and prompt content for ID generation
 * OUTPUT:    Unique prompt ID for correlation matching
 * BUSINESS:  Create deterministic prompt IDs for reliable correlation
 * CHANGE:    Initial prompt ID generation with terminal and content factors
 * RISK:      Low - ID generation utility with collision avoidance
 */
func (csm *ConcurrentSessionManager) generatePromptID(terminalCtx *TerminalContext, prompt string) string {
	// Create prompt ID based on terminal context + timestamp + prompt hash
	timestamp := time.Now().Unix()
	contextData := fmt.Sprintf("%d_%d_%s_%d", 
		terminalCtx.PID, 
		terminalCtx.ShellPID, 
		terminalCtx.WorkingDir, 
		timestamp)
	
	promptHash := csm.hashPrompt(prompt)
	
	return fmt.Sprintf("prompt_%x_%s", 
		sha256.Sum256([]byte(contextData)), 
		promptHash[:8])
}

/**
 * CONTEXT:   Generate consistent hash for prompt content
 * INPUT:     Prompt text content
 * OUTPUT:    Consistent hash string for prompt identification
 * BUSINESS:  Create prompt fingerprints for correlation and similarity detection
 * CHANGE:    Initial prompt hashing implementation
 * RISK:      Low - Hashing utility for prompt identification
 */
func (csm *ConcurrentSessionManager) hashPrompt(prompt string) string {
	// Normalize prompt for consistent hashing
	normalized := strings.TrimSpace(strings.ToLower(prompt))
	if len(normalized) > 500 {
		normalized = normalized[:500] // Limit length for consistency
	}
	
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", hash)
}

/**
 * CONTEXT:   Generate terminal identifier for session grouping
 * INPUT:     Terminal context with process and session information
 * OUTPUT:    Unique terminal identifier for correlation tracking
 * BUSINESS:  Group sessions by terminal for correlation and cleanup
 * CHANGE:    Initial terminal ID generation implementation
 * RISK:      Low - Terminal identification utility
 */
func (csm *ConcurrentSessionManager) getTerminalID(terminalCtx *TerminalContext) string {
	if terminalCtx == nil {
		return "unknown"
	}
	
	// Use session ID if available, otherwise construct from PIDs
	if terminalCtx.SessionID != "" {
		return terminalCtx.SessionID
	}
	
	return fmt.Sprintf("term_%d_%d_%s", 
		terminalCtx.PID, 
		terminalCtx.ShellPID, 
		terminalCtx.HostName)
}

// Getter methods for monitoring and debugging
func (csm *ConcurrentSessionManager) GetActiveSessionCount() int {
	csm.mutex.RLock()
	defer csm.mutex.RUnlock()
	return len(csm.activeSessions)
}

func (csm *ConcurrentSessionManager) GetOrphanedEventCount() int {
	csm.mutex.RLock()
	defer csm.mutex.RUnlock()
	return len(csm.orphanedEndEvents)
}

func (csm *ConcurrentSessionManager) GetActiveSessions() []*ActiveClaudeSession {
	csm.mutex.RLock()
	defer csm.mutex.RUnlock()
	
	sessions := make([]*ActiveClaudeSession, 0, len(csm.activeSessions))
	for _, session := range csm.activeSessions {
		sessionCopy := *session
		sessions = append(sessions, &sessionCopy)
	}
	return sessions
}

/**
 * CONTEXT:   Background cleanup process for expired sessions and orphaned events
 * INPUT:     Cleanup interval and age thresholds for automated maintenance
 * OUTPUT:    Cleaned session state with removed expired and orphaned entries
 * BUSINESS:  Maintain system health by cleaning up stale correlation data
 * CHANGE:    Initial background cleanup implementation
 * RISK:      Medium - Cleanup timing affects memory usage and correlation accuracy
 */
func (csm *ConcurrentSessionManager) runBackgroundCleanup() {
	ticker := time.NewTicker(csm.cleanupInterval)
	defer ticker.Stop()
	
	csm.cleanupRunning = true
	defer func() { csm.cleanupRunning = false }()
	
	for {
		select {
		case <-ticker.C:
			csm.performCleanup()
		case <-csm.stopCleanup:
			return
		}
	}
}

func (csm *ConcurrentSessionManager) performCleanup() {
	csm.mutex.Lock()
	defer csm.mutex.Unlock()
	
	now := time.Now()
	cleanupCount := 0
	
	// Clean up expired active sessions
	for sessionID, session := range csm.activeSessions {
		sessionAge := now.Sub(session.CreatedAt)
		
		if sessionAge > csm.maxSessionAge {
			// Session too old - mark as timed out and move to history
			session.State = CorrelationStateTimedOut
			session.LastCorrelationUpdate = now
			
			csm.sessionHistory = append(csm.sessionHistory, session)
			delete(csm.activeSessions, sessionID)
			delete(csm.sessionsByPromptID, session.PromptID)
			cleanupCount++
		}
	}
	
	// Clean up old orphaned events
	remainingEvents := make([]*ClaudeEndEvent, 0)
	for _, event := range csm.orphanedEndEvents {
		eventAge := now.Sub(event.Timestamp)
		if eventAge < csm.correlationTimeout {
			remainingEvents = append(remainingEvents, event)
		}
	}
	csm.orphanedEndEvents = remainingEvents
	
	// Clean up old session history (keep only recent)
	if len(csm.sessionHistory) > 1000 {
		// Keep only last 1000 sessions
		csm.sessionHistory = csm.sessionHistory[len(csm.sessionHistory)-1000:]
	}
}

// Shutdown gracefully stops the concurrent session manager
func (csm *ConcurrentSessionManager) Shutdown() error {
	if csm.cleanupRunning {
		csm.stopCleanup <- true
	}
	
	// Final cleanup
	csm.performCleanup()
	
	return nil
}