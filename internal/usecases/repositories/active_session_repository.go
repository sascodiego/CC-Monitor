/**
 * CONTEXT:   Repository interface for active session persistence in daemon-managed correlation
 * INPUT:     ActiveSession entities for database operations and query criteria
 * OUTPUT:    Persisted active sessions with context-based lookup capabilities
 * BUSINESS:  Provide database persistence for active session correlation without temp files
 * CHANGE:    Initial repository interface for active session management
 * RISK:      Medium - Database operations affect session correlation reliability
 */

package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/claude-monitor/system/internal/entities"
)

// ActiveSessionRepository defines the interface for active session persistence
type ActiveSessionRepository interface {
	// Create and update operations
	Save(ctx context.Context, activeSession *entities.ActiveSession) error
	Update(ctx context.Context, activeSession *entities.ActiveSession) error
	Delete(ctx context.Context, sessionID string) error

	// Query operations for context-based correlation
	FindByID(ctx context.Context, sessionID string) (*entities.ActiveSession, error)
	FindByTerminalPID(ctx context.Context, terminalPID int, userID string) ([]*entities.ActiveSession, error)
	FindByWorkingDir(ctx context.Context, workingDir, userID string) ([]*entities.ActiveSession, error)
	FindByUser(ctx context.Context, userID string) ([]*entities.ActiveSession, error)
	FindAll(ctx context.Context) ([]*entities.ActiveSession, error)

	// Cleanup and maintenance operations
	FindExpiredSessions(ctx context.Context, expiredBefore time.Time) ([]*entities.ActiveSession, error)
	DeleteExpiredSessions(ctx context.Context, expiredBefore time.Time) (int64, error)
	CountActiveSessions(ctx context.Context) (int64, error)

	// Statistical operations
	GetActiveSessionStatistics(ctx context.Context) (*ActiveSessionStatistics, error)
}

// ActiveSessionStatistics holds statistical data about active sessions
type ActiveSessionStatistics struct {
	TotalActiveSessions   int64                                         `json:"total_active_sessions"`
	SessionsByStatus      map[entities.ActiveSessionStatus]int64       `json:"sessions_by_status"`
	SessionsByUser        map[string]int64                              `json:"sessions_by_user"`
	AverageSessionAge     time.Duration                                 `json:"average_session_age"`
	OldestSessionAge      time.Duration                                 `json:"oldest_session_age"`
	NewestSessionAge      time.Duration                                 `json:"newest_session_age"`
	GeneratedAt           time.Time                                     `json:"generated_at"`
}

// Repository errors
var (
	ErrActiveSessionNotFound     = fmt.Errorf("active session not found")
	ErrActiveSessionExists       = fmt.Errorf("active session already exists")
	ErrActiveSessionInvalidID    = fmt.Errorf("invalid active session ID")
	ErrActiveSessionInvalidData  = fmt.Errorf("invalid active session data")
)