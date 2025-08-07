/**
 * CONTEXT:   Repository interface for work block persistence following Dependency Inversion Principle
 * INPUT:     WorkBlock entities and query parameters for CRUD operations
 * OUTPUT:    Interface contract for work block storage implementations
 * BUSINESS:  Abstract work block persistence to support multiple storage backends
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

// WorkBlockFilter provides filtering options for work block queries
type WorkBlockFilter struct {
	SessionID     string
	ProjectID     string
	ProjectName   string
	State         entities.WorkBlockState
	StartAfter    time.Time
	StartBefore   time.Time
	EndAfter      time.Time
	EndBefore     time.Time
	MinDuration   time.Duration
	MaxDuration   time.Duration
	IsActive      *bool
	Limit         int
	Offset        int
}

// WorkBlockSortBy defines available sorting options
type WorkBlockSortBy string

const (
	WorkBlockSortByStartTime  WorkBlockSortBy = "start_time"
	WorkBlockSortByEndTime    WorkBlockSortBy = "end_time"
	WorkBlockSortByDuration   WorkBlockSortBy = "duration"
	WorkBlockSortByActivity   WorkBlockSortBy = "activity_count"
	WorkBlockSortByUpdated    WorkBlockSortBy = "updated_at"
	WorkBlockSortByProject    WorkBlockSortBy = "project_name"
)

// WorkBlockSortOrder defines sort direction
type WorkBlockSortOrder string

const (
	WorkBlockSortAsc  WorkBlockSortOrder = "asc"
	WorkBlockSortDesc WorkBlockSortOrder = "desc"
)

/**
 * CONTEXT:   Primary repository interface for work block entity persistence
 * INPUT:     WorkBlock entities and query specifications for data operations
 * OUTPUT:    Repository contract enabling multiple storage backend implementations
 * BUSINESS:  Work blocks require persistent storage with efficient project-based querying
 * CHANGE:    Initial repository interface with comprehensive CRUD and analytics operations
 * RISK:      Low - Interface contract following dependency inversion principle
 */
type WorkBlockRepository interface {
	// Basic CRUD operations
	Save(ctx context.Context, workBlock *entities.WorkBlock) error
	FindByID(ctx context.Context, workBlockID string) (*entities.WorkBlock, error)
	Update(ctx context.Context, workBlock *entities.WorkBlock) error
	Delete(ctx context.Context, workBlockID string) error

	// Query operations
	FindBySessionID(ctx context.Context, sessionID string) ([]*entities.WorkBlock, error)
	FindByProjectID(ctx context.Context, projectID string) ([]*entities.WorkBlock, error)
	FindByProjectName(ctx context.Context, projectName string) ([]*entities.WorkBlock, error)
	FindActiveWorkBlocks(ctx context.Context) ([]*entities.WorkBlock, error)
	FindByFilter(ctx context.Context, filter WorkBlockFilter) ([]*entities.WorkBlock, error)
	FindWithSort(ctx context.Context, filter WorkBlockFilter, sortBy WorkBlockSortBy, order WorkBlockSortOrder) ([]*entities.WorkBlock, error)

	// Business logic queries
	FindIdleWorkBlocks(ctx context.Context, idleThreshold time.Duration) ([]*entities.WorkBlock, error)
	FindWorkBlocksInTimeRange(ctx context.Context, start, end time.Time) ([]*entities.WorkBlock, error)
	FindRecentWorkBlocks(ctx context.Context, limit int) ([]*entities.WorkBlock, error)
	FindLongRunningWorkBlocks(ctx context.Context, threshold time.Duration) ([]*entities.WorkBlock, error)

	// Project-based queries
	FindWorkBlocksByProject(ctx context.Context, projectID string, start, end time.Time) ([]*entities.WorkBlock, error)
	GetProjectWorkSummary(ctx context.Context, projectID string, start, end time.Time) (*ProjectWorkSummary, error)
	GetTopProjectsByHours(ctx context.Context, start, end time.Time, limit int) ([]*ProjectWorkSummary, error)

	// Analytics queries
	CountWorkBlocksByProject(ctx context.Context, projectID string) (int64, error)
	CountWorkBlocksByTimeRange(ctx context.Context, start, end time.Time) (int64, error)
	GetWorkBlockStatistics(ctx context.Context, start, end time.Time) (*WorkBlockStatistics, error)
	GetDailyWorkSummary(ctx context.Context, date time.Time) ([]*DailyWorkSummary, error)

	// Time-based aggregations
	GetHourlyDistribution(ctx context.Context, start, end time.Time) ([]*HourlyDistribution, error)
	GetWeeklyDistribution(ctx context.Context, start, end time.Time) ([]*WeeklyDistribution, error)
	GetProjectTimeDistribution(ctx context.Context, start, end time.Time) ([]*ProjectTimeDistribution, error)

	// Batch operations
	SaveBatch(ctx context.Context, workBlocks []*entities.WorkBlock) error
	FinishIdleWorkBlocks(ctx context.Context, idleThreshold time.Duration) (int64, error)
	DeleteBySessionID(ctx context.Context, sessionID string) (int64, error)

	// Transaction support
	WithTransaction(ctx context.Context, fn func(repo WorkBlockRepository) error) error
}

/**
 * CONTEXT:   Project work summary aggregation for project-based analytics
 * INPUT:     Work block aggregation calculations for specific projects
 * OUTPUT:    Project-level work metrics and statistics
 * BUSINESS:  Provide project-based insights for productivity and time allocation
 * CHANGE:    Initial project summary structure for analytics
 * RISK:      Low - Data structure for project-level reporting
 */
type ProjectWorkSummary struct {
	ProjectID         string    `json:"project_id"`
	ProjectName       string    `json:"project_name"`
	TotalWorkBlocks   int64     `json:"total_work_blocks"`
	TotalHours        float64   `json:"total_hours"`
	AverageBlockHours float64   `json:"average_block_hours"`
	FirstWorkTime     time.Time `json:"first_work_time"`
	LastWorkTime      time.Time `json:"last_work_time"`
	ActiveDays        int       `json:"active_days"`
	HoursPerDay       float64   `json:"hours_per_day"`
	EfficiencyRating  float64   `json:"efficiency_rating"`
}

/**
 * CONTEXT:   Work block statistics aggregation for comprehensive analytics
 * INPUT:     Statistical calculations from work block repository queries
 * OUTPUT:    Aggregated metrics for work block analysis and insights
 * BUSINESS:  Provide work block-level analytics for productivity optimization
 * CHANGE:    Initial statistics structure for work block analytics
 * RISK:      Low - Data structure for statistical reporting
 */
type WorkBlockStatistics struct {
	TotalWorkBlocks     int64         `json:"total_work_blocks"`
	ActiveWorkBlocks    int64         `json:"active_work_blocks"`
	IdleWorkBlocks      int64         `json:"idle_work_blocks"`
	FinishedWorkBlocks  int64         `json:"finished_work_blocks"`
	TotalHours          float64       `json:"total_hours"`
	AverageBlockHours   float64       `json:"average_block_hours"`
	MedianBlockHours    float64       `json:"median_block_hours"`
	TotalActivities     int64         `json:"total_activities"`
	AverageActivities   float64       `json:"average_activities_per_block"`
	UniqueProjects      int           `json:"unique_projects"`
	MostActiveProject   string        `json:"most_active_project"`
	LongestBlockHours   float64       `json:"longest_block_hours"`
	ShortestBlockHours  float64       `json:"shortest_block_hours"`
	PeriodStart         time.Time     `json:"period_start"`
	PeriodEnd           time.Time     `json:"period_end"`
	AnalysisDuration    time.Duration `json:"analysis_duration"`
}

/**
 * CONTEXT:   Daily work summary for day-by-day productivity tracking
 * INPUT:     Daily aggregation of work blocks and time metrics
 * OUTPUT:    Daily productivity metrics and project breakdown
 * BUSINESS:  Enable daily work pattern analysis and productivity tracking
 * CHANGE:    Initial daily summary structure for calendar-based analytics
 * RISK:      Low - Data structure for daily reporting
 */
type DailyWorkSummary struct {
	Date            time.Time `json:"date"`
	TotalHours      float64   `json:"total_hours"`
	TotalWorkBlocks int64     `json:"total_work_blocks"`
	UniqueProjects  int       `json:"unique_projects"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	ScheduleHours   float64   `json:"schedule_hours"`
	EfficiencyPct   float64   `json:"efficiency_percentage"`
	TopProject      string    `json:"top_project"`
	TopProjectHours float64   `json:"top_project_hours"`
}

/**
 * CONTEXT:   Hourly distribution for time pattern analysis
 * INPUT:     Hourly aggregation of work activity across time periods
 * OUTPUT:    Hour-by-hour work distribution for pattern identification
 * BUSINESS:  Identify peak productivity hours and work patterns
 * CHANGE:    Initial hourly distribution structure for temporal analytics
 * RISK:      Low - Data structure for time pattern analysis
 */
type HourlyDistribution struct {
	Hour        int     `json:"hour"`         // 0-23
	TotalHours  float64 `json:"total_hours"`
	WorkBlocks  int64   `json:"work_blocks"`
	Percentage  float64 `json:"percentage"`
	DayOfWeek   int     `json:"day_of_week"` // 0=Sunday, 1=Monday, etc.
}

/**
 * CONTEXT:   Weekly distribution for week-over-week productivity analysis
 * INPUT:     Weekly aggregation of work metrics and patterns
 * OUTPUT:    Week-level productivity trends and comparisons
 * BUSINESS:  Track weekly productivity trends and identify patterns
 * CHANGE:    Initial weekly distribution structure for trend analysis
 * RISK:      Low - Data structure for weekly reporting
 */
type WeeklyDistribution struct {
	WeekStart   time.Time `json:"week_start"`
	WeekEnd     time.Time `json:"week_end"`
	TotalHours  float64   `json:"total_hours"`
	WorkBlocks  int64     `json:"work_blocks"`
	ActiveDays  int       `json:"active_days"`
	Percentage  float64   `json:"percentage"`
	WeekNumber  int       `json:"week_number"`
	Year        int       `json:"year"`
}

/**
 * CONTEXT:   Project time distribution for portfolio analysis
 * INPUT:     Project-based time allocation across specified periods
 * OUTPUT:    Project time allocation percentages and rankings
 * BUSINESS:  Understand time allocation across different projects and priorities
 * CHANGE:    Initial project distribution structure for portfolio analytics
 * RISK:      Low - Data structure for project allocation analysis
 */
type ProjectTimeDistribution struct {
	ProjectID   string  `json:"project_id"`
	ProjectName string  `json:"project_name"`
	TotalHours  float64 `json:"total_hours"`
	Percentage  float64 `json:"percentage"`
	WorkBlocks  int64   `json:"work_blocks"`
	Rank        int     `json:"rank"`
}

/**
 * CONTEXT:   Work block repository error types for specific error handling
 * INPUT:     Repository operation context and error conditions
 * OUTPUT:    Typed errors for application-specific error handling
 * BUSINESS:  Enable proper error handling and user feedback for work block operations
 * CHANGE:    Initial error types for work block repository operations
 * RISK:      Low - Error type definitions for better error handling
 */
type WorkBlockRepositoryError struct {
	Op      string // Operation that failed
	Message string // Error message
	Err     error  // Underlying error
}

func (e *WorkBlockRepositoryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("work block repository %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("work block repository %s: %s", e.Op, e.Message)
}

func (e *WorkBlockRepositoryError) Unwrap() error {
	return e.Err
}

// Common work block repository errors
var (
	ErrWorkBlockNotFound      = &WorkBlockRepositoryError{Op: "find", Message: "work block not found"}
	ErrWorkBlockAlreadyExists = &WorkBlockRepositoryError{Op: "save", Message: "work block already exists"}
	ErrWorkBlockInvalidState  = &WorkBlockRepositoryError{Op: "update", Message: "invalid work block state"}
	ErrWorkBlockConcurrency   = &WorkBlockRepositoryError{Op: "update", Message: "concurrent modification detected"}
)