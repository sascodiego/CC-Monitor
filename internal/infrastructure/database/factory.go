/**
 * CONTEXT:   Database factory for initializing KuzuDB infrastructure layer
 * INPUT:     Configuration parameters and dependency injection requirements
 * OUTPUT:    Initialized repository implementations with connection management
 * BUSINESS:  Factory pattern provides clean initialization and dependency management
 * CHANGE:    Initial factory implementation for KuzuDB infrastructure setup
 * RISK:      Medium - Factory coordinates complex initialization with error handling
 */

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/claude-monitor/system/internal/usecases/repositories"
)

/**
 * CONTEXT:   Database infrastructure container with all repository implementations
 * INPUT:     Connection manager and repository interface implementations
 * OUTPUT:    Complete database infrastructure ready for dependency injection
 * BUSINESS:  Infrastructure container enables clean architecture dependency inversion
 * CHANGE:    Initial infrastructure container with all repository implementations
 * RISK:      Low - Simple container pattern with interface implementations
 */
type KuzuDatabaseInfrastructure struct {
	connManager       *KuzuConnectionManager
	migrationManager  *KuzuMigrationManager
	sessionRepo       repositories.SessionRepository
	workBlockRepo     repositories.WorkBlockRepository
	projectRepo       repositories.ProjectRepository
	activityRepo      repositories.ActivityRepository
}

// Getter methods for repository access
func (kdi *KuzuDatabaseInfrastructure) SessionRepository() repositories.SessionRepository {
	return kdi.sessionRepo
}

func (kdi *KuzuDatabaseInfrastructure) WorkBlockRepository() repositories.WorkBlockRepository {
	return kdi.workBlockRepo
}

func (kdi *KuzuDatabaseInfrastructure) ProjectRepository() repositories.ProjectRepository {
	return kdi.projectRepo
}

func (kdi *KuzuDatabaseInfrastructure) ActivityRepository() repositories.ActivityRepository {
	return kdi.activityRepo
}

func (kdi *KuzuDatabaseInfrastructure) ConnectionManager() *KuzuConnectionManager {
	return kdi.connManager
}

func (kdi *KuzuDatabaseInfrastructure) MigrationManager() *KuzuMigrationManager {
	return kdi.migrationManager
}

/**
 * CONTEXT:   Database infrastructure factory with complete initialization
 * INPUT:     Database configuration and initialization parameters
 * OUTPUT:    Fully initialized database infrastructure with all repositories
 * BUSINESS:  Factory provides one-stop initialization for database layer
 * CHANGE:    Initial factory implementation with migration and repository setup
 * RISK:      Medium - Complex initialization requires careful error handling
 */
func NewKuzuDatabaseInfrastructure(config KuzuConnectionConfig) (*KuzuDatabaseInfrastructure, error) {
	// Initialize connection manager
	connManager, err := NewKuzuConnectionManagerWithValidation(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	// Initialize migration manager
	migrationManager := NewKuzuMigrationManager(connManager, "")

	// Create repository implementations
	sessionRepo := NewKuzuSessionRepository(connManager)
	workBlockRepo := NewKuzuWorkBlockRepository(connManager)
	projectRepo := NewKuzuProjectRepository(connManager)
	activityRepo := NewKuzuActivityRepository(connManager)

	infrastructure := &KuzuDatabaseInfrastructure{
		connManager:      connManager,
		migrationManager: migrationManager,
		sessionRepo:      sessionRepo,
		workBlockRepo:    workBlockRepo,
		projectRepo:      projectRepo,
		activityRepo:     activityRepo,
	}

	return infrastructure, nil
}

/**
 * CONTEXT:   Initialize database with schema migration and health checks
 * INPUT:     Context for initialization operations and timeout control
 * OUTPUT:    Database fully initialized and ready for work tracking operations
 * BUSINESS:  Initialization ensures database is ready for production work tracking
 * CHANGE:    Initial initialization with migration and health check
 * RISK:      Medium - Database initialization can fail due to permissions or schema issues
 */
func (kdi *KuzuDatabaseInfrastructure) Initialize(ctx context.Context) error {
	// Initialize migration tracking
	if err := kdi.migrationManager.InitializeMigrationTracking(ctx); err != nil {
		return fmt.Errorf("failed to initialize migration tracking: %w", err)
	}

	// Run migrations to ensure schema is up to date
	if err := kdi.migrationManager.Migrate(ctx, 0); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Validate schema is correctly set up
	if err := kdi.migrationManager.ValidateSchema(ctx); err != nil {
		return fmt.Errorf("database schema validation failed: %w", err)
	}

	// Health check to ensure everything is working
	if err := kdi.HealthCheck(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Comprehensive health check for database infrastructure
 * INPUT:     Context for health check operations and timeout control
 * OUTPUT:    Health status of all database components and connections
 * BUSINESS:  Health monitoring ensures reliable work tracking service
 * CHANGE:    Initial health check implementation for all components
 * RISK:      Low - Read-only health validation with minimal system impact
 */
func (kdi *KuzuDatabaseInfrastructure) HealthCheck(ctx context.Context) error {
	// Check connection manager health
	if err := kdi.connManager.HealthCheck(ctx); err != nil {
		return fmt.Errorf("connection manager health check failed: %w", err)
	}

	// Test basic repository operations
	healthCheckCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test session repository
	if _, err := kdi.sessionRepo.CountSessionsByTimeRange(healthCheckCtx, time.Now().Add(-24*time.Hour), time.Now()); err != nil {
		return fmt.Errorf("session repository health check failed: %w", err)
	}

	// Test project repository  
	if _, err := kdi.projectRepo.CountProjects(healthCheckCtx); err != nil {
		return fmt.Errorf("project repository health check failed: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Get comprehensive database statistics for monitoring
 * INPUT:     Context for statistics queries
 * OUTPUT:    Database statistics including connection pool and repository metrics
 * BUSINESS:  Database monitoring enables performance optimization and capacity planning
 * CHANGE:    Initial statistics collection for monitoring dashboard
 * RISK:      Low - Read-only statistics gathering with minimal performance impact
 */
type DatabaseStatistics struct {
	ConnectionPool    KuzuPoolStats                     `json:"connection_pool"`
	SessionStats      *repositories.SessionStatistics   `json:"session_stats"`
	WorkBlockStats    *repositories.WorkBlockStatistics `json:"work_block_stats"`
	ProjectStats      *repositories.ProjectStatistics   `json:"project_stats"`
	ActivityStats     *repositories.ActivityStatistics  `json:"activity_stats"`
	MigrationHistory  []Migration                       `json:"migration_history"`
	HealthStatus      string                           `json:"health_status"`
	LastUpdated       time.Time                        `json:"last_updated"`
}

func (kdi *KuzuDatabaseInfrastructure) GetStatistics(ctx context.Context) (*DatabaseStatistics, error) {
	statsCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stats := &DatabaseStatistics{
		ConnectionPool: kdi.connManager.GetStats(),
		LastUpdated:    time.Now(),
	}

	// Get health status
	if err := kdi.HealthCheck(statsCtx); err != nil {
		stats.HealthStatus = fmt.Sprintf("Unhealthy: %v", err)
	} else {
		stats.HealthStatus = "Healthy"
	}

	// Get migration history
	migrationHistory, err := kdi.migrationManager.GetMigrationHistory(statsCtx)
	if err == nil {
		stats.MigrationHistory = migrationHistory
	}

	// Get repository statistics (with timeout protection)
	now := time.Now()
	last24Hours := now.Add(-24 * time.Hour)

	if sessionStats, err := kdi.sessionRepo.GetSessionStatistics(statsCtx, "all", last24Hours, now); err == nil {
		stats.SessionStats = sessionStats
	}

	if workBlockStats, err := kdi.workBlockRepo.GetWorkBlockStatistics(statsCtx, last24Hours, now); err == nil {
		stats.WorkBlockStats = workBlockStats
	}

	if projectStats, err := kdi.projectRepo.GetProjectStatistics(statsCtx); err == nil {
		stats.ProjectStats = projectStats
	}

	if activityStats, err := kdi.activityRepo.GetActivityStatistics(statsCtx, last24Hours, now); err == nil {
		stats.ActivityStats = activityStats
	}

	return stats, nil
}

/**
 * CONTEXT:   Cleanup and shutdown database infrastructure safely
 * INPUT:     No parameters, initiates complete shutdown
 * OUTPUT:    All resources cleaned up and connections closed
 * BUSINESS:  Proper shutdown prevents resource leaks and data corruption
 * CHANGE:    Initial shutdown implementation with resource cleanup
 * RISK:      Low - Safe resource cleanup with error collection
 */
func (kdi *KuzuDatabaseInfrastructure) Close() error {
	var errors []error

	// Close connection manager
	if err := kdi.connManager.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close connection manager: %w", err))
	}

	// Repository cleanup is handled by connection manager closure
	// No explicit cleanup needed for repositories

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("errors during database infrastructure closure: %v", errors)
	}

	return nil
}

/**
 * CONTEXT:   Factory function for default development configuration
 * INPUT:     Database path for development setup
 * OUTPUT:    Database infrastructure configured for development use
 * BUSINESS:  Development factory enables quick setup for development and testing
 * CHANGE:    Initial development factory with sensible defaults
 * RISK:      Low - Development configuration with reasonable defaults
 */
func NewDevelopmentKuzuInfrastructure(dbPath string) (*KuzuDatabaseInfrastructure, error) {
	config := DefaultKuzuConfig()
	if dbPath != "" {
		config.DatabasePath = dbPath
	}

	// Development-specific settings
	config.EnableLogging = true
	config.QueryTimeout = 10 * time.Second

	return NewKuzuDatabaseInfrastructure(config)
}

/**
 * CONTEXT:   Factory function for production configuration with optimizations
 * INPUT:     Database path and production configuration parameters
 * OUTPUT:    Database infrastructure optimized for production workloads
 * BUSINESS:  Production factory provides optimized configuration for production deployment
 * CHANGE:    Initial production factory with performance optimizations
 * RISK:      Medium - Production configuration requires careful tuning for workload
 */
func NewProductionKuzuInfrastructure(dbPath string, maxConnections int, bufferSizeMB uint64) (*KuzuDatabaseInfrastructure, error) {
	config := DefaultKuzuConfig()
	config.DatabasePath = dbPath

	// Production optimizations
	if maxConnections > 0 {
		config.MaxConnections = maxConnections
	} else {
		config.MaxConnections = 20 // Higher for production
	}

	if bufferSizeMB > 0 {
		config.BufferPoolSize = bufferSizeMB
	} else {
		config.BufferPoolSize = 1024 // 1GB for production
	}

	config.EnableLogging = false // Disable debug logging in production
	config.QueryTimeout = 30 * time.Second
	config.ConnTimeout = 10 * time.Second

	return NewKuzuDatabaseInfrastructure(config)
}

/**
 * CONTEXT:   Factory function for testing configuration with isolation
 * INPUT:     Test database path and testing configuration
 * OUTPUT:    Database infrastructure configured for testing isolation
 * BUSINESS:  Test factory enables isolated testing with clean database state
 * CHANGE:    Initial test factory with testing-specific configuration
 * RISK:      Low - Test configuration with isolation and cleanup support
 */
func NewTestKuzuInfrastructure(testDbPath string) (*KuzuDatabaseInfrastructure, error) {
	config := DefaultKuzuConfig()
	config.DatabasePath = testDbPath

	// Testing-specific settings
	config.MaxConnections = 5 // Lower for tests
	config.QueryTimeout = 5 * time.Second
	config.ConnTimeout = 3 * time.Second
	config.EnableLogging = true // Enable for test debugging

	return NewKuzuDatabaseInfrastructure(config)
}