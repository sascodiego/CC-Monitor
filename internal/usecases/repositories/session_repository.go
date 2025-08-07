/**
 * CONTEXT:   Repository interface for session persistence following Dependency Inversion Principle
 * INPUT:     Session entities and query parameters for CRUD operations
 * OUTPUT:    Interface contract for session storage implementations
 * BUSINESS:  Abstract session persistence to allow multiple storage backends
 * CHANGE:    Initial repository interface following Clean Architecture principles
 * RISK:      Low - Interface definition with no implementation dependencies
 */

package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/claude-monitor/system/internal/entities"
)

// SessionFilter provides filtering options for session queries
type SessionFilter struct {
	UserID      string
	State       entities.SessionState
	StartAfter  time.Time
	StartBefore time.Time
	EndAfter    time.Time
	EndBefore   time.Time
	IsActive    *bool
	Limit       int
	Offset      int
}

// SessionSortBy defines available sorting options
type SessionSortBy string

const (
	SessionSortByStartTime SessionSortBy = "start_time"
	SessionSortByEndTime   SessionSortBy = "end_time"
	SessionSortByUpdated   SessionSortBy = "updated_at"
	SessionSortByActivity  SessionSortBy = "activity_count"
)

// SessionSortOrder defines sort direction
type SessionSortOrder string

const (
	SessionSortAsc  SessionSortOrder = "asc"
	SessionSortDesc SessionSortOrder = "desc"
)

/**
 * CONTEXT:   Primary repository interface for session entity persistence
 * INPUT:     Session entities and query specifications for data operations
 * OUTPUT:    Repository contract enabling multiple storage backend implementations
 * BUSINESS:  Sessions require persistent storage with efficient querying for analytics
 * CHANGE:    Initial repository interface with comprehensive CRUD operations
 * RISK:      Low - Interface contract following dependency inversion principle
 */
type SessionRepository interface {
	// Basic CRUD operations
	Save(ctx context.Context, session *entities.Session) error
	FindByID(ctx context.Context, sessionID string) (*entities.Session, error)
	Update(ctx context.Context, session *entities.Session) error
	Delete(ctx context.Context, sessionID string) error

	// Query operations
	FindByUserID(ctx context.Context, userID string) ([]*entities.Session, error)
	FindActiveSession(ctx context.Context, userID string) (*entities.Session, error)
	FindByFilter(ctx context.Context, filter SessionFilter) ([]*entities.Session, error)
	FindWithSort(ctx context.Context, filter SessionFilter, sortBy SessionSortBy, order SessionSortOrder) ([]*entities.Session, error)

	// Business logic queries
	FindExpiredSessions(ctx context.Context, beforeTime time.Time) ([]*entities.Session, error)
	FindSessionsInTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*entities.Session, error)
	FindRecentSessions(ctx context.Context, userID string, limit int) ([]*entities.Session, error)

	// Analytics queries
	CountSessionsByUser(ctx context.Context, userID string) (int64, error)
	CountSessionsByTimeRange(ctx context.Context, start, end time.Time) (int64, error)
	GetSessionStatistics(ctx context.Context, userID string, start, end time.Time) (*SessionStatistics, error)

	// Batch operations
	SaveBatch(ctx context.Context, sessions []*entities.Session) error
	DeleteExpired(ctx context.Context, beforeTime time.Time) (int64, error)

	// Transaction support
	WithTransaction(ctx context.Context, fn func(repo SessionRepository) error) error
}

/**
 * CONTEXT:   Session statistics aggregation for analytics and reporting
 * INPUT:     Statistical calculations from session repository queries
 * OUTPUT:    Aggregated metrics for session analysis and insights
 * BUSINESS:  Provide session-level analytics for productivity insights
 * CHANGE:    Initial statistics structure for session analytics
 * RISK:      Low - Data structure for statistical reporting
 */
type SessionStatistics struct {
	TotalSessions       int64         `json:"total_sessions"`
	ActiveSessions      int64         `json:"active_sessions"`
	ExpiredSessions     int64         `json:"expired_sessions"`
	FinishedSessions    int64         `json:"finished_sessions"`
	TotalHours          float64       `json:"total_hours"`
	AverageSessionHours float64       `json:"average_session_hours"`
	TotalActivities     int64         `json:"total_activities"`
	AverageActivities   float64       `json:"average_activities_per_session"`
	FirstSessionTime    time.Time     `json:"first_session_time"`
	LastSessionTime     time.Time     `json:"last_session_time"`
	SessionDuration     time.Duration `json:"session_duration"`
	PeriodStart         time.Time     `json:"period_start"`
	PeriodEnd           time.Time     `json:"period_end"`
}

/**
 * CONTEXT:   Session repository error types for specific error handling
 * INPUT:     Repository operation context and error conditions
 * OUTPUT:    Typed errors for application-specific error handling
 * BUSINESS:  Enable proper error handling and user feedback for session operations
 * CHANGE:    Initial error types for session repository operations
 * RISK:      Low - Error type definitions for better error handling
 */
type SessionRepositoryError struct {
	Op      string // Operation that failed
	Message string // Error message
	Err     error  // Underlying error
}

func (e *SessionRepositoryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("session repository %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("session repository %s: %s", e.Op, e.Message)
}

func (e *SessionRepositoryError) Unwrap() error {
	return e.Err
}

// Common session repository errors
var (
	ErrSessionNotFound      = &SessionRepositoryError{Op: "find", Message: "session not found"}
	ErrSessionAlreadyExists = &SessionRepositoryError{Op: "save", Message: "session already exists"}
	ErrSessionInvalidState  = &SessionRepositoryError{Op: "update", Message: "invalid session state"}
	ErrSessionConcurrency   = &SessionRepositoryError{Op: "update", Message: "concurrent modification detected"}
)