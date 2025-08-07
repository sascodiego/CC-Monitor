/**
 * CONTEXT:   Repository interface for project persistence following Dependency Inversion Principle
 * INPUT:     Project entities and query parameters for CRUD operations
 * OUTPUT:    Interface contract for project storage implementations
 * BUSINESS:  Abstract project persistence to support multiple storage backends
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

// ProjectFilter provides filtering options for project queries
type ProjectFilter struct {
	Name         string
	ProjectType  entities.ProjectType
	IsActive     *bool
	CreatedAfter time.Time
	CreatedBefore time.Time
	UpdatedAfter time.Time
	UpdatedBefore time.Time
	MinHours     float64
	MaxHours     float64
	Limit        int
	Offset       int
}

// ProjectSortBy defines available sorting options
type ProjectSortBy string

const (
	ProjectSortByName        ProjectSortBy = "name"
	ProjectSortByCreated     ProjectSortBy = "created_at"
	ProjectSortByUpdated     ProjectSortBy = "updated_at"
	ProjectSortByLastActive  ProjectSortBy = "last_active_time"
	ProjectSortByTotalHours  ProjectSortBy = "total_hours"
	ProjectSortByWorkBlocks  ProjectSortBy = "total_work_blocks"
	ProjectSortByType        ProjectSortBy = "project_type"
)

// ProjectSortOrder defines sort direction
type ProjectSortOrder string

const (
	ProjectSortAsc  ProjectSortOrder = "asc"
	ProjectSortDesc ProjectSortOrder = "desc"
)

/**
 * CONTEXT:   Primary repository interface for project entity persistence
 * INPUT:     Project entities and query specifications for data operations
 * OUTPUT:    Repository contract enabling multiple storage backend implementations
 * BUSINESS:  Projects require persistent storage with efficient path-based querying
 * CHANGE:    Initial repository interface with comprehensive CRUD and analytics operations
 * RISK:      Low - Interface contract following dependency inversion principle
 */
type ProjectRepository interface {
	// Basic CRUD operations
	Save(ctx context.Context, project *entities.Project) error
	FindByID(ctx context.Context, projectID string) (*entities.Project, error)
	FindByPath(ctx context.Context, projectPath string) (*entities.Project, error)
	FindByName(ctx context.Context, projectName string) (*entities.Project, error)
	Update(ctx context.Context, project *entities.Project) error
	Delete(ctx context.Context, projectID string) error

	// Query operations
	FindAll(ctx context.Context) ([]*entities.Project, error)
	FindActive(ctx context.Context) ([]*entities.Project, error)
	FindInactive(ctx context.Context) ([]*entities.Project, error)
	FindByType(ctx context.Context, projectType entities.ProjectType) ([]*entities.Project, error)
	FindByFilter(ctx context.Context, filter ProjectFilter) ([]*entities.Project, error)
	FindWithSort(ctx context.Context, filter ProjectFilter, sortBy ProjectSortBy, order ProjectSortOrder) ([]*entities.Project, error)

	// Business logic queries
	FindOrCreateByPath(ctx context.Context, projectPath string) (*entities.Project, error)
	FindRecentlyActive(ctx context.Context, limit int) ([]*entities.Project, error)
	FindInactiveProjects(ctx context.Context, inactiveThreshold time.Duration) ([]*entities.Project, error)
	FindProjectsWithMinimumHours(ctx context.Context, minHours float64) ([]*entities.Project, error)

	// Analytics queries
	CountProjects(ctx context.Context) (int64, error)
	CountActiveProjects(ctx context.Context) (int64, error)
	CountProjectsByType(ctx context.Context) (map[entities.ProjectType]int64, error)
	GetProjectStatistics(ctx context.Context, projectID string, start, end time.Time) (*ProjectStatistics, error)
	GetAllProjectStatistics(ctx context.Context) (*ProjectStatistics, error)
	GetTopProjectsByHours(ctx context.Context, limit int) ([]*entities.Project, error)
	GetTopProjectsByWorkBlocks(ctx context.Context, limit int) ([]*entities.Project, error)

	// Time-based queries
	FindProjectsInTimeRange(ctx context.Context, start, end time.Time) ([]*entities.Project, error)
	GetProjectActivityByDay(ctx context.Context, projectID string, start, end time.Time) ([]*ProjectDailyActivity, error)
	GetProjectTrends(ctx context.Context, projectID string, days int) (*ProjectTrends, error)

	// Maintenance operations
	MarkInactiveProjects(ctx context.Context, inactiveThreshold time.Duration) (int64, error)
	UpdateProjectStatistics(ctx context.Context, projectID string) error
	CleanupEmptyProjects(ctx context.Context) (int64, error)

	// Batch operations
	SaveBatch(ctx context.Context, projects []*entities.Project) error
	UpdateBatch(ctx context.Context, projects []*entities.Project) error

	// Transaction support
	WithTransaction(ctx context.Context, fn func(repo ProjectRepository) error) error
}

/**
 * CONTEXT:   Project statistics aggregation for project analytics and insights
 * INPUT:     Statistical calculations from project repository queries
 * OUTPUT:    Aggregated project metrics for productivity analysis
 * BUSINESS:  Provide project-level insights for portfolio management and planning
 * CHANGE:    Initial statistics structure for project analytics
 * RISK:      Low - Data structure for project statistical reporting
 */
type ProjectStatistics struct {
	TotalProjects           int64                              `json:"total_projects"`
	ActiveProjects          int64                              `json:"active_projects"`
	InactiveProjects        int64                              `json:"inactive_projects"`
	ProjectsByType          map[entities.ProjectType]int64     `json:"projects_by_type"`
	TotalHours              float64                            `json:"total_hours"`
	TotalWorkBlocks         int64                              `json:"total_work_blocks"`
	AverageHoursPerProject  float64                            `json:"average_hours_per_project"`
	AverageBlocksPerProject float64                            `json:"average_blocks_per_project"`
	MostActiveProject       *entities.Project                  `json:"most_active_project"`
	RecentlyCreatedCount    int64                              `json:"recently_created_count"`
	LongRunningCount        int64                              `json:"long_running_count"`
	ProjectTypeDistribution []*ProjectTypeDistribution         `json:"project_type_distribution"`
	TopProjects             []*ProjectRanking                  `json:"top_projects"`
	CalculatedAt            time.Time                          `json:"calculated_at"`
}

/**
 * CONTEXT:   Daily project activity tracking for detailed project analysis
 * INPUT:     Daily aggregation of project work activity
 * OUTPUT:    Day-by-day project activity metrics
 * BUSINESS:  Track daily project engagement and identify work patterns
 * CHANGE:    Initial daily activity structure for project analytics
 * RISK:      Low - Data structure for project activity tracking
 */
type ProjectDailyActivity struct {
	Date            time.Time `json:"date"`
	ProjectID       string    `json:"project_id"`
	ProjectName     string    `json:"project_name"`
	TotalHours      float64   `json:"total_hours"`
	TotalWorkBlocks int64     `json:"total_work_blocks"`
	FirstActivity   time.Time `json:"first_activity"`
	LastActivity    time.Time `json:"last_activity"`
	ScheduleHours   float64   `json:"schedule_hours"`
	EfficiencyPct   float64   `json:"efficiency_percentage"`
	IsMainProject   bool      `json:"is_main_project"`
}

/**
 * CONTEXT:   Project trends analysis for long-term project insights
 * INPUT:     Trend calculations over specified time periods
 * OUTPUT:    Project trend metrics and growth patterns
 * BUSINESS:  Identify project growth trends and productivity changes over time
 * CHANGE:    Initial trends structure for project analysis
 * RISK:      Low - Data structure for trend analysis
 */
type ProjectTrends struct {
	ProjectID              string    `json:"project_id"`
	ProjectName            string    `json:"project_name"`
	AnalysisPeriodDays     int       `json:"analysis_period_days"`
	TrendStartDate         time.Time `json:"trend_start_date"`
	TrendEndDate           time.Time `json:"trend_end_date"`
	
	// Hours trends
	TotalHours             float64   `json:"total_hours"`
	DailyAverageHours      float64   `json:"daily_average_hours"`
	WeeklyAverageHours     float64   `json:"weekly_average_hours"`
	HoursTrend             string    `json:"hours_trend"` // "increasing", "decreasing", "stable"
	HoursTrendPercentage   float64   `json:"hours_trend_percentage"`
	
	// Activity trends
	TotalWorkBlocks        int64     `json:"total_work_blocks"`
	DailyAverageBlocks     float64   `json:"daily_average_blocks"`
	ActivityTrend          string    `json:"activity_trend"`
	ActivityTrendPercentage float64  `json:"activity_trend_percentage"`
	
	// Engagement metrics
	ActiveDays             int       `json:"active_days"`
	ActiveDaysPercentage   float64   `json:"active_days_percentage"`
	LongestStreakDays      int       `json:"longest_streak_days"`
	CurrentStreakDays      int       `json:"current_streak_days"`
	
	// Productivity metrics
	AverageBlockSize       float64   `json:"average_block_size_hours"`
	ProductivityScore      float64   `json:"productivity_score"` // 0-100
	ConsistencyScore       float64   `json:"consistency_score"` // 0-100
}

/**
 * CONTEXT:   Project type distribution for portfolio analysis
 * INPUT:     Project type aggregation across the portfolio
 * OUTPUT:    Distribution of projects by type with metrics
 * BUSINESS:  Understand project type diversity and allocation patterns
 * CHANGE:    Initial type distribution structure for portfolio analytics
 * RISK:      Low - Data structure for project type analysis
 */
type ProjectTypeDistribution struct {
	ProjectType     entities.ProjectType `json:"project_type"`
	Count           int64                `json:"count"`
	Percentage      float64              `json:"percentage"`
	TotalHours      float64              `json:"total_hours"`
	AverageHours    float64              `json:"average_hours"`
	HoursPercentage float64              `json:"hours_percentage"`
}

/**
 * CONTEXT:   Project ranking for top project identification
 * INPUT:     Project ranking calculations based on various metrics
 * OUTPUT:    Ranked list of projects by activity and productivity
 * BUSINESS:  Identify most important and active projects for prioritization
 * CHANGE:    Initial ranking structure for project prioritization
 * RISK:      Low - Data structure for project ranking
 */
type ProjectRanking struct {
	Rank            int     `json:"rank"`
	ProjectID       string  `json:"project_id"`
	ProjectName     string  `json:"project_name"`
	ProjectType     entities.ProjectType `json:"project_type"`
	TotalHours      float64 `json:"total_hours"`
	TotalWorkBlocks int64   `json:"total_work_blocks"`
	HoursPercentage float64 `json:"hours_percentage"`
	LastActiveTime  time.Time `json:"last_active_time"`
	DaysActive      int     `json:"days_active"`
	Score           float64 `json:"score"` // Composite ranking score
}

/**
 * CONTEXT:   Project repository error types for specific error handling
 * INPUT:     Repository operation context and error conditions
 * OUTPUT:    Typed errors for application-specific error handling
 * BUSINESS:  Enable proper error handling and user feedback for project operations
 * CHANGE:    Initial error types for project repository operations
 * RISK:      Low - Error type definitions for better error handling
 */
type ProjectRepositoryError struct {
	Op      string // Operation that failed
	Message string // Error message
	Err     error  // Underlying error
}

func (e *ProjectRepositoryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("project repository %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("project repository %s: %s", e.Op, e.Message)
}

func (e *ProjectRepositoryError) Unwrap() error {
	return e.Err
}

// Common project repository errors
var (
	ErrProjectNotFound      = &ProjectRepositoryError{Op: "find", Message: "project not found"}
	ErrProjectAlreadyExists = &ProjectRepositoryError{Op: "save", Message: "project already exists"}
	ErrProjectInvalidPath   = &ProjectRepositoryError{Op: "save", Message: "invalid project path"}
	ErrProjectConcurrency   = &ProjectRepositoryError{Op: "update", Message: "concurrent modification detected"}
)