/**
 * CONTEXT:   Active session tracker for daemon-managed context-based session correlation
 * INPUT:     Session contexts from hook events, no manual ID management required
 * OUTPUT:    Correlated sessions using terminal/project context matching algorithms
 * BUSINESS:  Eliminate temporary files and environment variables for hook correlation
 * CHANGE:    Initial implementation replacing file-based correlation with smart matching
 * RISK:      High - Core correlation system that affects all session tracking accuracy
 */

package usecases

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/claude-monitor/system/internal/usecases/repositories"
)

// ActiveSessionTracker manages active sessions with context-based correlation
type ActiveSessionTracker struct {
	activeSessions     map[string]*entities.ActiveSession // sessionID -> active session
	sessionsByTerminal map[string][]string                 // "terminalPID:userID" -> sessionIDs
	sessionsByProject  map[string][]string                 // "workingDir:userID" -> sessionIDs
	activeSessionRepo  repositories.ActiveSessionRepository
	sessionRepo        repositories.SessionRepository
	mu                 sync.RWMutex
	logger             *log.Logger

	// Configuration
	maxConcurrentSessions int
	sessionTimeoutDuration time.Duration
	cleanupInterval       time.Duration
}

// ActiveSessionTrackerConfig holds configuration for the tracker
type ActiveSessionTrackerConfig struct {
	ActiveSessionRepo     repositories.ActiveSessionRepository
	SessionRepo           repositories.SessionRepository
	Logger                *log.Logger
	MaxConcurrentSessions int
	SessionTimeout        time.Duration
	CleanupInterval       time.Duration
}

/**
 * CONTEXT:   Factory method for creating new active session tracker
 * INPUT:     ActiveSessionTrackerConfig with dependencies and configuration
 * OUTPUT:    Configured ActiveSessionTracker ready for session correlation
 * BUSINESS:  Initialize tracker with proper defaults for Claude session management
 * CHANGE:    Initial implementation with comprehensive configuration
 * RISK:      Low - Initialization with sensible defaults and validation
 */
func NewActiveSessionTracker(config ActiveSessionTrackerConfig) *ActiveSessionTracker {
	// Set default values
	if config.MaxConcurrentSessions == 0 {
		config.MaxConcurrentSessions = 100 // Reasonable limit for concurrent sessions
	}
	if config.SessionTimeout == 0 {
		config.SessionTimeout = 30 * time.Minute // Default session timeout
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute // Default cleanup interval
	}

	return &ActiveSessionTracker{
		activeSessions:         make(map[string]*entities.ActiveSession),
		sessionsByTerminal:     make(map[string][]string),
		sessionsByProject:      make(map[string][]string),
		activeSessionRepo:      config.ActiveSessionRepo,
		sessionRepo:            config.SessionRepo,
		logger:                 config.Logger,
		maxConcurrentSessions:  config.MaxConcurrentSessions,
		sessionTimeoutDuration: config.SessionTimeout,
		cleanupInterval:        config.CleanupInterval,
	}
}

/**
 * CONTEXT:   Create new active session from hook start event
 * INPUT:     SessionContext from hook start event with terminal and project info
 * OUTPUT:    New ActiveSession entity tracked in memory and persisted to database
 * BUSINESS:  Start session tracking without requiring ID passing between hook processes
 * CHANGE:    Initial session creation with context-based tracking
 * RISK:      Medium - Session creation affects all subsequent correlation attempts
 */
func (ast *ActiveSessionTracker) CreateSession(ctx context.Context, sessionContext entities.SessionContext) (*entities.ActiveSession, error) {
	ast.mu.Lock()
	defer ast.mu.Unlock()

	// Check if we're at the concurrent session limit
	if len(ast.activeSessions) >= ast.maxConcurrentSessions {
		ast.logger.Printf("Warning: Maximum concurrent sessions (%d) reached, cleaning up expired sessions",
			ast.maxConcurrentSessions)
		
		// Try to clean up expired sessions to make room
		cleaned := ast.cleanupExpiredSessionsUnsafe(time.Now())
		if cleaned == 0 {
			return nil, fmt.Errorf("maximum concurrent sessions reached (%d) and no expired sessions to clean",
				ast.maxConcurrentSessions)
		}
	}

	// Create new active session
	sessionConfig := entities.ActiveSessionConfig{
		SessionContext:    sessionContext,
		EstimatedDuration: 5 * time.Minute, // Default Claude processing time
		EstimatedTokens:   1000,             // Default token estimate
	}

	activeSession, err := entities.NewActiveSession(sessionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create active session entity: %w", err)
	}

	// Store in memory
	ast.activeSessions[activeSession.ID()] = activeSession

	// Index by terminal context
	terminalKey := fmt.Sprintf("%d:%s", sessionContext.TerminalPID, sessionContext.UserID)
	ast.sessionsByTerminal[terminalKey] = append(ast.sessionsByTerminal[terminalKey], activeSession.ID())

	// Index by project context
	projectKey := fmt.Sprintf("%s:%s", sessionContext.WorkingDir, sessionContext.UserID)
	ast.sessionsByProject[projectKey] = append(ast.sessionsByProject[projectKey], activeSession.ID())

	// Persist to database
	err = ast.activeSessionRepo.Save(ctx, activeSession)
	if err != nil {
		// Remove from memory if database save failed
		delete(ast.activeSessions, activeSession.ID())
		ast.removeFromIndexes(activeSession)
		return nil, fmt.Errorf("failed to save active session to database: %w", err)
	}

	ast.logger.Printf("Created active session %s for terminal PID %d, user %s, project %s",
		activeSession.ID(), sessionContext.TerminalPID, sessionContext.UserID, sessionContext.WorkingDir)

	return activeSession, nil
}

/**
 * CONTEXT:   Find active session for end event using smart correlation algorithms
 * INPUT:     SessionContext from hook end event for session matching
 * OUTPUT:    Best matching ActiveSession or error if no confident match found
 * BUSINESS:  Correlate end events with start events using terminal and project context
 * CHANGE:    Initial implementation with multiple matching strategies and confidence scoring
 * RISK:      High - Matching accuracy directly affects session tracking reliability
 */
func (ast *ActiveSessionTracker) FindSessionForEndEvent(ctx context.Context, sessionContext entities.SessionContext) (*entities.ActiveSession, error) {
	ast.mu.RLock()
	defer ast.mu.RUnlock()

	// Strategy 1: Exact terminal match (highest confidence)
	terminalKey := fmt.Sprintf("%d:%s", sessionContext.TerminalPID, sessionContext.UserID)
	terminalSessionIDs := ast.sessionsByTerminal[terminalKey]
	
	if len(terminalSessionIDs) > 0 {
		sessions := ast.getSessionsByIDs(terminalSessionIDs)
		
		if len(sessions) == 1 {
			// Perfect single match
			ast.logger.Printf("Found exact terminal match for session %s (terminal PID %d)",
				sessions[0].ID(), sessionContext.TerminalPID)
			return sessions[0], nil
		}
		
		if len(sessions) > 1 {
			// Multiple terminal matches - use scoring
			bestMatch := ast.findBestMatchByScoring(sessions, sessionContext)
			if bestMatch != nil {
				ast.logger.Printf("Found best terminal match for session %s (score-based)",
					bestMatch.ID())
				return bestMatch, nil
			}
		}
	}

	// Strategy 2: Project context match (lower confidence)
	projectKey := fmt.Sprintf("%s:%s", sessionContext.WorkingDir, sessionContext.UserID)
	projectSessionIDs := ast.sessionsByProject[projectKey]
	
	if len(projectSessionIDs) > 0 {
		sessions := ast.getSessionsByIDs(projectSessionIDs)
		
		if len(sessions) == 1 {
			// Single project match
			ast.logger.Printf("Found project match for session %s (working dir %s)",
				sessions[0].ID(), sessionContext.WorkingDir)
			return sessions[0], nil
		}
		
		if len(sessions) > 1 {
			// Multiple project matches - use scoring
			bestMatch := ast.findBestMatchByScoring(sessions, sessionContext)
			if bestMatch != nil {
				ast.logger.Printf("Found best project match for session %s (score-based)",
					bestMatch.ID())
				return bestMatch, nil
			}
		}
	}

	// Strategy 3: Fallback to all active sessions with user match
	userSessions := ast.getActiveSessionsByUser(sessionContext.UserID)
	if len(userSessions) > 0 {
		bestMatch := ast.findBestMatchByScoring(userSessions, sessionContext)
		if bestMatch != nil {
			ast.logger.Printf("Found fallback user match for session %s", bestMatch.ID())
			return bestMatch, nil
		}
	}

	return nil, fmt.Errorf("no matching active session found for terminal PID %d, user %s, directory %s",
		sessionContext.TerminalPID, sessionContext.UserID, sessionContext.WorkingDir)
}

/**
 * CONTEXT:   End active session with processing metrics and convert to regular session
 * INPUT:     Active session ID, end timestamp, processing duration, and token count
 * OUTPUT:    Completed session entity with accurate timing and metrics
 * BUSINESS:  Complete session lifecycle from active tracking to historical record
 * CHANGE:    Initial session completion with metrics capture and state transition
 * RISK:      Medium - Session completion affects historical tracking and analytics
 */
func (ast *ActiveSessionTracker) EndSession(ctx context.Context, sessionID string, endTime time.Time, processingDuration time.Duration, tokenCount int64) (*entities.Session, error) {
	ast.mu.Lock()
	defer ast.mu.Unlock()

	// Find active session
	activeSession, exists := ast.activeSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("active session %s not found", sessionID)
	}

	// End the active session
	err := activeSession.EndSession(endTime, processingDuration, tokenCount)
	if err != nil {
		return nil, fmt.Errorf("failed to end active session: %w", err)
	}

	// Update in database
	err = ast.activeSessionRepo.Update(ctx, activeSession)
	if err != nil {
		ast.logger.Printf("Warning: failed to update active session in database: %v", err)
	}

	// Create historical session record
	sessionConfig := entities.SessionConfig{
		UserID:    activeSession.SessionContext().UserID,
		StartTime: activeSession.StartTime(),
	}

	session, err := entities.NewSession(sessionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create session entity: %w", err)
	}

	// Update session with end metrics
	err = session.RecordActivity(endTime, "") // Record final activity
	if err != nil {
		return nil, fmt.Errorf("failed to record final activity: %w", err)
	}

	err = session.Finalize(endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize session: %w", err)
	}

	// Save historical session
	err = ast.sessionRepo.Save(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to save historical session: %w", err)
	}

	// Remove from active tracking
	delete(ast.activeSessions, sessionID)
	ast.removeFromIndexes(activeSession)

	// Remove from active session database
	err = ast.activeSessionRepo.Delete(ctx, sessionID)
	if err != nil {
		ast.logger.Printf("Warning: failed to delete active session from database: %v", err)
	}

	ast.logger.Printf("Completed session %s -> %s (duration: %v, tokens: %d)",
		activeSession.ID(), session.ID(), processingDuration, tokenCount)

	return session, nil
}

// Private helper methods

func (ast *ActiveSessionTracker) getSessionsByIDs(sessionIDs []string) []*entities.ActiveSession {
	sessions := make([]*entities.ActiveSession, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if session, exists := ast.activeSessions[id]; exists {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func (ast *ActiveSessionTracker) getActiveSessionsByUser(userID string) []*entities.ActiveSession {
	sessions := make([]*entities.ActiveSession, 0)
	for _, session := range ast.activeSessions {
		if session.SessionContext().UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

/**
 * CONTEXT:   Find best matching session using confidence scoring algorithm
 * INPUT:     Candidate sessions and target session context for matching
 * OUTPUT:    Best matching session or nil if no confident match found
 * BUSINESS:  Use weighted scoring to find most likely session match
 * CHANGE:    Initial implementation of confidence-based session matching
 * RISK:      High - Scoring algorithm accuracy critical for correlation reliability
 */
func (ast *ActiveSessionTracker) findBestMatchByScoring(sessions []*entities.ActiveSession, targetContext entities.SessionContext) *entities.ActiveSession {
	var bestMatch *entities.ActiveSession
	var bestScore float64

	for _, session := range sessions {
		// Get context match score (0.0 to 1.0)
		contextScore := session.ContextMatchScore(targetContext)
		
		// Get timing score based on how close we are to estimated completion
		timingScore := ast.calculateTimingScore(session, targetContext.Timestamp)
		
		// Combined score with weighting
		combinedScore := (contextScore * 0.7) + (timingScore * 0.3)
		
		ast.logger.Printf("Session %s scores: context=%.2f, timing=%.2f, combined=%.2f",
			session.ID()[:8], contextScore, timingScore, combinedScore)

		if combinedScore > bestScore {
			bestMatch = session
			bestScore = combinedScore
		}
	}

	// Require minimum confidence threshold
	minConfidenceThreshold := 0.6
	if bestScore < minConfidenceThreshold {
		ast.logger.Printf("Best match score %.2f below threshold %.2f, rejecting match",
			bestScore, minConfidenceThreshold)
		return nil
	}

	return bestMatch
}

func (ast *ActiveSessionTracker) calculateTimingScore(session *entities.ActiveSession, endTime time.Time) float64 {
	// How long has the session been running?
	actualDuration := endTime.Sub(session.StartTime())
	expectedDuration := session.EstimatedEndTime().Sub(session.StartTime())

	// Perfect score if actual duration matches expected duration
	durationDiff := math.Abs(actualDuration.Seconds() - expectedDuration.Seconds())
	maxAcceptableDiff := expectedDuration.Seconds() * 2 // Allow 200% variance

	if durationDiff > maxAcceptableDiff {
		return 0.0 // Too far off
	}

	// Score decreases linearly with difference
	score := 1.0 - (durationDiff / maxAcceptableDiff)
	return math.Max(0.0, score)
}

func (ast *ActiveSessionTracker) removeFromIndexes(activeSession *entities.ActiveSession) {
	sessionContext := activeSession.SessionContext()
	sessionID := activeSession.ID()

	// Remove from terminal index
	terminalKey := fmt.Sprintf("%d:%s", sessionContext.TerminalPID, sessionContext.UserID)
	ast.sessionsByTerminal[terminalKey] = ast.removeFromSlice(ast.sessionsByTerminal[terminalKey], sessionID)
	if len(ast.sessionsByTerminal[terminalKey]) == 0 {
		delete(ast.sessionsByTerminal, terminalKey)
	}

	// Remove from project index
	projectKey := fmt.Sprintf("%s:%s", sessionContext.WorkingDir, sessionContext.UserID)
	ast.sessionsByProject[projectKey] = ast.removeFromSlice(ast.sessionsByProject[projectKey], sessionID)
	if len(ast.sessionsByProject[projectKey]) == 0 {
		delete(ast.sessionsByProject, projectKey)
	}
}

func (ast *ActiveSessionTracker) removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

/**
 * CONTEXT:   Clean up expired active sessions that haven't received end events
 * INPUT:     Current timestamp to determine which sessions have expired
 * OUTPUT:    Number of sessions cleaned up and any errors encountered
 * BUSINESS:  Prevent memory leaks from orphaned sessions and maintain accurate state
 * CHANGE:    Initial implementation of active session cleanup process
 * RISK:      Medium - Cleanup affects session state and must preserve valid sessions
 */
func (ast *ActiveSessionTracker) CleanupExpiredSessions(ctx context.Context, currentTime time.Time) (int64, error) {
	ast.mu.Lock()
	defer ast.mu.Unlock()

	return int64(ast.cleanupExpiredSessionsUnsafe(currentTime)), nil
}

// cleanupExpiredSessionsUnsafe performs cleanup without acquiring locks (caller must hold lock)
func (ast *ActiveSessionTracker) cleanupExpiredSessionsUnsafe(currentTime time.Time) int {
	cleaned := 0
	expiredSessions := make([]*entities.ActiveSession, 0)

	// Find expired sessions
	for _, session := range ast.activeSessions {
		timeSinceStart := currentTime.Sub(session.StartTime())
		if timeSinceStart > ast.sessionTimeoutDuration {
			expiredSessions = append(expiredSessions, session)
		}
	}

	// Clean up expired sessions
	for _, session := range expiredSessions {
		// Remove from tracking
		delete(ast.activeSessions, session.ID())
		ast.removeFromIndexes(session)
		cleaned++

		ast.logger.Printf("Cleaned up expired active session %s (started %v ago)",
			session.ID(), currentTime.Sub(session.StartTime()))

		// Note: In a production system, you might want to:
		// 1. Create a "timed out" historical session record
		// 2. Send notifications about orphaned sessions
		// 3. Delete from database in background to avoid blocking
	}

	return cleaned
}

/**
 * CONTEXT:   Get statistics about active session tracker state
 * INPUT:     No parameters, uses current tracker state
 * OUTPUT:    Statistics about active sessions, memory usage, and performance metrics
 * BUSINESS:  Provide insights into tracker performance and resource usage
 * CHANGE:    Initial implementation of tracker statistics
 * RISK:      Low - Read-only statistics gathering
 */
func (ast *ActiveSessionTracker) GetStatistics() map[string]interface{} {
	ast.mu.RLock()
	defer ast.mu.RUnlock()

	stats := map[string]interface{}{
		"active_sessions":        len(ast.activeSessions),
		"terminal_indexes":       len(ast.sessionsByTerminal),
		"project_indexes":        len(ast.sessionsByProject),
		"max_concurrent_limit":   ast.maxConcurrentSessions,
		"session_timeout_mins":   ast.sessionTimeoutDuration.Minutes(),
	}

	// Add session status breakdown
	statusCounts := make(map[entities.ActiveSessionStatus]int)
	for _, session := range ast.activeSessions {
		statusCounts[session.Status()]++
	}
	stats["status_breakdown"] = statusCounts

	return stats
}