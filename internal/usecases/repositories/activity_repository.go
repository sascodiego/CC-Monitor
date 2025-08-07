/**
 * CONTEXT:   Repository interface for activity event persistence following Dependency Inversion Principle
 * INPUT:     ActivityEvent entities and query parameters for CRUD operations
 * OUTPUT:    Interface contract for activity event storage implementations
 * BUSINESS:  Abstract activity event persistence to support multiple storage backends
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

// ActivityFilter provides filtering options for activity event queries
type ActivityFilter struct {
	UserID         string
	SessionID      string
	WorkBlockID    string
	ProjectID      string
	ProjectName    string
	ActivityType   entities.ActivityType
	ActivitySource entities.ActivitySource
	TimestampAfter time.Time
	TimestampBefore time.Time
	Command        string
	Limit          int
	Offset         int
}

// ActivitySortBy defines available sorting options
type ActivitySortBy string

const (
	ActivitySortByTimestamp ActivitySortBy = "timestamp"
	ActivitySortByCreated   ActivitySortBy = "created_at"
	ActivitySortByCommand   ActivitySortBy = "command"
	ActivitySortByProject   ActivitySortBy = "project_name"
	ActivitySortByType      ActivitySortBy = "activity_type"
)

// ActivitySortOrder defines sort direction
type ActivitySortOrder string

const (
	ActivitySortAsc  ActivitySortOrder = "asc"
	ActivitySortDesc ActivitySortOrder = "desc"
)

/**
 * CONTEXT:   Primary repository interface for activity event entity persistence
 * INPUT:     ActivityEvent entities and query specifications for data operations
 * OUTPUT:    Repository contract enabling multiple storage backend implementations
 * BUSINESS:  Activity events require efficient storage and querying for audit and analytics
 * CHANGE:    Initial repository interface with comprehensive CRUD and analytics operations
 * RISK:      Low - Interface contract following dependency inversion principle
 */
type ActivityRepository interface {
	// Basic CRUD operations
	Save(ctx context.Context, activity *entities.ActivityEvent) error
	FindByID(ctx context.Context, activityID string) (*entities.ActivityEvent, error)
	Update(ctx context.Context, activity *entities.ActivityEvent) error
	Delete(ctx context.Context, activityID string) error

	// Query operations
	FindByUserID(ctx context.Context, userID string) ([]*entities.ActivityEvent, error)
	FindBySessionID(ctx context.Context, sessionID string) ([]*entities.ActivityEvent, error)
	FindByWorkBlockID(ctx context.Context, workBlockID string) ([]*entities.ActivityEvent, error)
	FindByProjectID(ctx context.Context, projectID string) ([]*entities.ActivityEvent, error)
	FindByFilter(ctx context.Context, filter ActivityFilter) ([]*entities.ActivityEvent, error)
	FindWithSort(ctx context.Context, filter ActivityFilter, sortBy ActivitySortBy, order ActivitySortOrder) ([]*entities.ActivityEvent, error)

	// Time-based queries
	FindInTimeRange(ctx context.Context, start, end time.Time) ([]*entities.ActivityEvent, error)
	FindRecentActivities(ctx context.Context, limit int) ([]*entities.ActivityEvent, error)
	FindActivitiesByDay(ctx context.Context, date time.Time) ([]*entities.ActivityEvent, error)
	FindActivitiesByWeek(ctx context.Context, weekStart time.Time) ([]*entities.ActivityEvent, error)

	// Business logic queries
	FindLastActivityByUser(ctx context.Context, userID string) (*entities.ActivityEvent, error)
	FindLastActivityByProject(ctx context.Context, projectID string) (*entities.ActivityEvent, error)
	FindActivitiesAfterTimestamp(ctx context.Context, timestamp time.Time) ([]*entities.ActivityEvent, error)
	FindActivitiesByType(ctx context.Context, activityType entities.ActivityType) ([]*entities.ActivityEvent, error)

	// Analytics queries
	CountActivitiesByUser(ctx context.Context, userID string) (int64, error)
	CountActivitiesByProject(ctx context.Context, projectID string) (int64, error)
	CountActivitiesByTimeRange(ctx context.Context, start, end time.Time) (int64, error)
	CountActivitiesByType(ctx context.Context) (map[entities.ActivityType]int64, error)
	GetActivityStatistics(ctx context.Context, start, end time.Time) (*ActivityStatistics, error)

	// Pattern analysis
	GetActivityPatterns(ctx context.Context, userID string, days int) (*ActivityPatterns, error)
	GetHourlyActivityDistribution(ctx context.Context, start, end time.Time) ([]*HourlyActivityDistribution, error)
	GetCommandFrequency(ctx context.Context, start, end time.Time, limit int) ([]*CommandFrequency, error)
	GetProjectActivityBreakdown(ctx context.Context, start, end time.Time) ([]*ProjectActivityBreakdown, error)

	// Maintenance operations
	DeleteOldActivities(ctx context.Context, beforeTime time.Time) (int64, error)
	ArchiveActivities(ctx context.Context, beforeTime time.Time) (int64, error)

	// Batch operations
	SaveBatch(ctx context.Context, activities []*entities.ActivityEvent) error
	DeleteBySessionID(ctx context.Context, sessionID string) (int64, error)
	DeleteByWorkBlockID(ctx context.Context, workBlockID string) (int64, error)

	// Transaction support
	WithTransaction(ctx context.Context, fn func(repo ActivityRepository) error) error
}

/**
 * CONTEXT:   Activity statistics aggregation for activity analytics and insights
 * INPUT:     Statistical calculations from activity repository queries
 * OUTPUT:    Aggregated activity metrics for productivity analysis
 * BUSINESS:  Provide activity-level insights for user behavior and productivity patterns
 * CHANGE:    Initial statistics structure for activity analytics
 * RISK:      Low - Data structure for activity statistical reporting
 */
type ActivityStatistics struct {
	TotalActivities        int64                              `json:"total_activities"`
	ActivitiesByType       map[entities.ActivityType]int64    `json:"activities_by_type"`
	ActivitiesBySource     map[entities.ActivitySource]int64  `json:"activities_by_source"`
	UniqueUsers            int64                              `json:"unique_users"`
	UniqueSessions         int64                              `json:"unique_sessions"`
	UniqueProjects         int64                              `json:"unique_projects"`
	AverageActivitiesPerSession float64                       `json:"average_activities_per_session"`
	AverageActivitiesPerDay     float64                       `json:"average_activities_per_day"`
	MostActiveUser         string                             `json:"most_active_user"`
	MostActiveProject      string                             `json:"most_active_project"`
	MostCommonType         entities.ActivityType              `json:"most_common_type"`
	MostCommonCommand      string                             `json:"most_common_command"`
	FirstActivityTime      time.Time                          `json:"first_activity_time"`
	LastActivityTime       time.Time                          `json:"last_activity_time"`
	PeakActivityHour       int                                `json:"peak_activity_hour"`
	CalculatedAt           time.Time                          `json:"calculated_at"`
}

/**
 * CONTEXT:   Activity patterns analysis for user behavior insights
 * INPUT:     Pattern recognition calculations from activity data
 * OUTPUT:    User activity patterns and behavioral metrics
 * BUSINESS:  Understand user work patterns to optimize productivity and UX
 * CHANGE:    Initial patterns structure for behavioral analytics
 * RISK:      Low - Data structure for pattern analysis
 */
type ActivityPatterns struct {
	UserID                  string                           `json:"user_id"`
	AnalysisPeriodDays      int                              `json:"analysis_period_days"`
	TotalActivities         int64                            `json:"total_activities"`
	DailyActivityAverage    float64                          `json:"daily_activity_average"`
	MostActiveHour          int                              `json:"most_active_hour"`
	MostActiveDayOfWeek     int                              `json:"most_active_day_of_week"`
	WorkingHoursStart       int                              `json:"working_hours_start"`
	WorkingHoursEnd         int                              `json:"working_hours_end"`
	AverageSessionLength    float64                          `json:"average_session_length_hours"`
	PreferredActivityType   entities.ActivityType            `json:"preferred_activity_type"`
	MostUsedCommands        []string                         `json:"most_used_commands"`
	ProjectSwitchFrequency  float64                          `json:"project_switch_frequency"`
	ConsistencyScore        float64                          `json:"consistency_score"` // 0-100
	ProductivityScore       float64                          `json:"productivity_score"` // 0-100
	HourlyDistribution      []*HourlyActivityDistribution    `json:"hourly_distribution"`
	DailyPatterns           []*DailyActivityPattern          `json:"daily_patterns"`
}

/**
 * CONTEXT:   Hourly activity distribution for time pattern analysis
 * INPUT:     Hourly aggregation of activity data
 * OUTPUT:    Hour-by-hour activity distribution metrics
 * BUSINESS:  Identify peak activity hours and optimize scheduling
 * CHANGE:    Initial hourly distribution structure for temporal analytics
 * RISK:      Low - Data structure for time-based analysis
 */
type HourlyActivityDistribution struct {
	Hour            int     `json:"hour"`             // 0-23
	ActivityCount   int64   `json:"activity_count"`
	Percentage      float64 `json:"percentage"`
	AveragePerDay   float64 `json:"average_per_day"`
	PeakDayCount    int64   `json:"peak_day_count"`
	IsWorkingHour   bool    `json:"is_working_hour"`
}

/**
 * CONTEXT:   Daily activity pattern for work schedule analysis
 * INPUT:     Daily activity pattern recognition
 * OUTPUT:    Daily work pattern classification and metrics
 * BUSINESS:  Classify daily work patterns to understand work styles
 * CHANGE:    Initial daily pattern structure for schedule analytics
 * RISK:      Low - Data structure for daily pattern analysis
 */
type DailyActivityPattern struct {
	DayOfWeek       int     `json:"day_of_week"`      // 0=Sunday, 1=Monday, etc.
	DayName         string  `json:"day_name"`
	ActivityCount   int64   `json:"activity_count"`
	AveragePerWeek  float64 `json:"average_per_week"`
	IsWorkingDay    bool    `json:"is_working_day"`
	PatternType     string  `json:"pattern_type"`     // "light", "normal", "heavy", "intense"
	StartHour       int     `json:"start_hour"`
	EndHour         int     `json:"end_hour"`
	PeakHour        int     `json:"peak_hour"`
}

/**
 * CONTEXT:   Command frequency analysis for command usage insights
 * INPUT:     Command usage frequency calculations
 * OUTPUT:    Most frequently used commands and usage patterns
 * BUSINESS:  Understand command usage to improve CLI design and help
 * CHANGE:    Initial command frequency structure for usage analytics
 * RISK:      Low - Data structure for command analysis
 */
type CommandFrequency struct {
	Command       string  `json:"command"`
	Count         int64   `json:"count"`
	Percentage    float64 `json:"percentage"`
	FirstUsed     time.Time `json:"first_used"`
	LastUsed      time.Time `json:"last_used"`
	UniqueUsers   int64   `json:"unique_users"`
	AveragePerDay float64 `json:"average_per_day"`
	Rank          int     `json:"rank"`
}

/**
 * CONTEXT:   Project activity breakdown for project engagement analysis
 * INPUT:     Project-based activity aggregation
 * OUTPUT:    Activity distribution across projects
 * BUSINESS:  Understand project engagement levels and activity distribution
 * CHANGE:    Initial project activity breakdown structure for project analytics
 * RISK:      Low - Data structure for project activity analysis
 */
type ProjectActivityBreakdown struct {
	ProjectID       string  `json:"project_id"`
	ProjectName     string  `json:"project_name"`
	ActivityCount   int64   `json:"activity_count"`
	Percentage      float64 `json:"percentage"`
	FirstActivity   time.Time `json:"first_activity"`
	LastActivity    time.Time `json:"last_activity"`
	ActiveDays      int     `json:"active_days"`
	AveragePerDay   float64 `json:"average_per_day"`
	MostCommonType  entities.ActivityType `json:"most_common_type"`
	TopCommands     []string `json:"top_commands"`
}

/**
 * CONTEXT:   Activity repository error types for specific error handling
 * INPUT:     Repository operation context and error conditions
 * OUTPUT:    Typed errors for application-specific error handling
 * BUSINESS:  Enable proper error handling and user feedback for activity operations
 * CHANGE:    Initial error types for activity repository operations
 * RISK:      Low - Error type definitions for better error handling
 */
type ActivityRepositoryError struct {
	Op      string // Operation that failed
	Message string // Error message
	Err     error  // Underlying error
}

func (e *ActivityRepositoryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("activity repository %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("activity repository %s: %s", e.Op, e.Message)
}

func (e *ActivityRepositoryError) Unwrap() error {
	return e.Err
}

// Common activity repository errors
var (
	ErrActivityNotFound      = &ActivityRepositoryError{Op: "find", Message: "activity not found"}
	ErrActivityAlreadyExists = &ActivityRepositoryError{Op: "save", Message: "activity already exists"}
	ErrActivityInvalidTime   = &ActivityRepositoryError{Op: "save", Message: "invalid activity timestamp"}
	ErrActivityConcurrency   = &ActivityRepositoryError{Op: "update", Message: "concurrent modification detected"}
)