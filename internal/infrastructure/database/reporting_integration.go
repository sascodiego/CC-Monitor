/**
 * CONTEXT:   Integration layer for comprehensive KuzuDB reporting system
 * INPUT:     Reporting requests, user configuration, and system parameters
 * OUTPUT:    Unified reporting interface with caching, optimization, and monitoring
 * BUSINESS:  Integrated reporting system provides fast, comprehensive work hour analytics
 * CHANGE:    Initial implementation tying together all reporting components
 * RISK:      Medium - Integration complexity requires careful component coordination
 */

package database

import (
	"context"
	"fmt"
	"time"
)

/**
 * CONTEXT:   Comprehensive reporting service with all optimization features
 * INPUT:     Reporting requests with user context and time periods
 * OUTPUT:    Fast, cached, optimized reports with performance monitoring
 * BUSINESS:  Integrated service provides complete work hour analytics solution
 * CHANGE:    Initial implementation combining all reporting components
 * RISK:      Medium - Service integration requires proper component coordination
 */
type ComprehensiveReportingService struct {
	connectionManager   *KuzuConnectionManager
	reportingQueries    *ReportingQueries
	queryCache         *QueryCache
	performanceMonitor *PerformanceMonitor
	queryOptimizer     *QueryOptimizer
}

/**
 * CONTEXT:   Create comprehensive reporting service with all components
 * INPUT:     Database configuration and monitoring setup
 * OUTPUT:    Fully integrated reporting service ready for production use
 * BUSINESS:  Service initialization with all performance optimization features
 * CHANGE:    Initial service constructor with complete component integration
 * RISK:      Medium - Complex initialization requires proper component setup
 */
func NewComprehensiveReportingService(config KuzuConnectionConfig) (*ComprehensiveReportingService, error) {
	// Initialize database connection manager
	connectionManager, err := NewKuzuConnectionManagerWithValidation(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize connection manager: %w", err)
	}

	// Initialize query cache
	queryCache := NewQueryCache()

	// Initialize performance monitor with alert callback
	performanceMonitor := NewPerformanceMonitor(func(metrics QueryMetrics) {
		// Alert callback for slow queries
		fmt.Printf("SLOW QUERY ALERT: %s took %v (User: %s)\n", 
			metrics.QueryType, metrics.ExecutionTime, metrics.UserID)
	})

	// Initialize reporting queries with cache
	reportingQueries := NewReportingQueries(connectionManager)
	reportingQueries.queryCache = queryCache

	// Initialize query optimizer
	queryOptimizer := NewQueryOptimizer(connectionManager, performanceMonitor)

	service := &ComprehensiveReportingService{
		connectionManager:   connectionManager,
		reportingQueries:    reportingQueries,
		queryCache:         queryCache,
		performanceMonitor: performanceMonitor,
		queryOptimizer:     queryOptimizer,
	}

	// Perform initial setup and optimization
	if err := service.initializeOptimizations(context.Background()); err != nil {
		// Log warning but don't fail service creation
		fmt.Printf("Warning: Initial optimizations failed: %v\n", err)
	}

	return service, nil
}

/**
 * CONTEXT:   Initialize database optimizations for best performance
 * INPUT:     Context for timeout control during setup
 * OUTPUT:    Applied optimizations and initial performance setup
 * BUSINESS:  Initial optimization setup ensures optimal performance from start
 * CHANGE:    Initial optimization setup with essential indexes and configuration
 * RISK:      Medium - Optimization setup may take time but is non-critical
 */
func (crs *ComprehensiveReportingService) initializeOptimizations(ctx context.Context) error {
	// Apply critical indexes for reporting performance
	criticalIndexes := []string{
		"session_user_time",      // Essential for daily/weekly reports
		"workblock_project_time", // Essential for project reports
		"project_path",           // Essential for project lookups
	}

	result, err := crs.queryOptimizer.ApplyIndexRecommendations(ctx, criticalIndexes)
	if err != nil {
		return fmt.Errorf("failed to apply critical indexes: %w", err)
	}

	// Log optimization results
	fmt.Printf("Applied %d critical indexes in %v\n", 
		len(result.AppliedIndexes), result.TotalDuration)

	if len(result.FailedIndexes) > 0 {
		fmt.Printf("Warning: Failed to apply %d indexes: %v\n", 
			len(result.FailedIndexes), result.FailedIndexes)
	}

	return nil
}

/**
 * CONTEXT:   Get daily work report with full optimization pipeline
 * INPUT:     Date, user ID, and context for timeout control
 * OUTPUT:    Optimized daily report with caching and performance monitoring
 * BUSINESS:  Daily reports are most frequently requested, must be fastest
 * CHANGE:    Initial implementation with complete optimization pipeline
 * RISK:      Low - Well-tested daily report generation with full optimization
 */
func (crs *ComprehensiveReportingService) GetDailyReport(ctx context.Context, date time.Time, userID string) (*DailyReport, error) {
	startTime := time.Now()
	queryType := "daily"
	
	// Try cache first
	cacheKey := fmt.Sprintf("daily_report_%s_%s", userID, date.Format("2006-01-02"))
	if cached := crs.queryCache.Get(cacheKey); cached != nil {
		if report, ok := cached.(*DailyReport); ok {
			// Record cache hit
			crs.performanceMonitor.RecordQuery(queryType, time.Since(startTime), true, 0, userID, date.String(), nil)
			return report, nil
		}
	}

	// Generate report with optimization
	report, err := crs.reportingQueries.GetDailyReport(ctx, date, userID)
	executionTime := time.Since(startTime)

	// Record performance metrics
	resultSize := int64(0)
	if report != nil {
		resultSize = crs.estimateReportSize(report)
	}
	
	crs.performanceMonitor.RecordQuery(queryType, executionTime, false, resultSize, userID, date.String(), err)

	return report, err
}

/**
 * CONTEXT:   Get weekly work report with trend analysis and optimization
 * INPUT:     Week start date, user ID, and context for timeout control
 * OUTPUT:    Comprehensive weekly report with trend insights and optimization
 * BUSINESS:  Weekly reports show productivity trends and week-over-week analysis
 * CHANGE:    Initial implementation with full weekly analytics
 * RISK:      Low - Weekly report with comprehensive trend analysis
 */
func (crs *ComprehensiveReportingService) GetWeeklyReport(ctx context.Context, weekStart time.Time, userID string) (*WeeklyReport, error) {
	startTime := time.Now()
	queryType := "weekly"
	
	// Try cache first
	cacheKey := fmt.Sprintf("weekly_report_%s_%s", userID, weekStart.Format("2006-01-02"))
	if cached := crs.queryCache.Get(cacheKey); cached != nil {
		if report, ok := cached.(*WeeklyReport); ok {
			crs.performanceMonitor.RecordQuery(queryType, time.Since(startTime), true, 0, userID, weekStart.String(), nil)
			return report, nil
		}
	}

	// Generate report
	report, err := crs.reportingQueries.GetWeeklyReport(ctx, weekStart, userID)
	executionTime := time.Since(startTime)

	resultSize := int64(0)
	if report != nil {
		resultSize = crs.estimateReportSize(report)
	}

	crs.performanceMonitor.RecordQuery(queryType, executionTime, false, resultSize, userID, weekStart.String(), err)

	return report, err
}

/**
 * CONTEXT:   Get monthly work report with calendar heatmap and comprehensive analytics
 * INPUT:     Month date, user ID, and context for timeout control
 * OUTPUT:    Complete monthly report with heatmap data and goal tracking
 * BUSINESS:  Monthly reports provide long-term insights and productivity analysis
 * CHANGE:    Initial implementation with calendar heatmap and comprehensive analytics
 * RISK:      Medium - Monthly reports are most complex, require optimization
 */
func (crs *ComprehensiveReportingService) GetMonthlyReport(ctx context.Context, month time.Time, userID string) (*MonthlyReport, error) {
	startTime := time.Now()
	queryType := "monthly"
	
	// Try cache first  
	cacheKey := fmt.Sprintf("monthly_report_%s_%s", userID, month.Format("2006-01"))
	if cached := crs.queryCache.Get(cacheKey); cached != nil {
		if report, ok := cached.(*MonthlyReport); ok {
			crs.performanceMonitor.RecordQuery(queryType, time.Since(startTime), true, 0, userID, month.String(), nil)
			return report, nil
		}
	}

	// Generate report
	report, err := crs.reportingQueries.GetMonthlyReport(ctx, month, userID)
	executionTime := time.Since(startTime)

	resultSize := int64(0)
	if report != nil {
		resultSize = crs.estimateReportSize(report)
	}

	crs.performanceMonitor.RecordQuery(queryType, executionTime, false, resultSize, userID, month.String(), err)

	return report, err
}

/**
 * CONTEXT:   Get project-specific analytics with deep-dive insights
 * INPUT:     Project ID, time period, user ID, and context
 * OUTPUT:    Detailed project report with work patterns and efficiency metrics
 * BUSINESS:  Project reports show time allocation and productivity per project
 * CHANGE:    Initial implementation with comprehensive project analytics
 * RISK:      Low - Project-scoped reporting with efficient queries
 */
func (crs *ComprehensiveReportingService) GetProjectReport(ctx context.Context, projectID string, period TimePeriod, userID string) (*ProjectReport, error) {
	startTime := time.Now()
	queryType := "project"
	
	// Cache key includes project and period
	cacheKey := fmt.Sprintf("project_report_%s_%s_%s_%s", userID, projectID, 
		period.Start.Format("2006-01-02"), period.End.Format("2006-01-02"))
	
	if cached := crs.queryCache.Get(cacheKey); cached != nil {
		if report, ok := cached.(*ProjectReport); ok {
			crs.performanceMonitor.RecordQuery(queryType, time.Since(startTime), true, 0, userID, projectID, nil)
			return report, nil
		}
	}

	// Generate report
	report, err := crs.reportingQueries.GetProjectReport(ctx, projectID, period, userID)
	executionTime := time.Since(startTime)

	resultSize := int64(0)
	if report != nil {
		resultSize = crs.estimateReportSize(report)
	}

	crs.performanceMonitor.RecordQuery(queryType, executionTime, false, resultSize, userID, projectID, err)

	return report, err
}

/**
 * CONTEXT:   Get project rankings by time investment and activity
 * INPUT:     Time period, user ID, and ranking criteria
 * OUTPUT:    Ranked list of projects with time allocation metrics
 * BUSINESS:  Project rankings show time allocation priorities and focus areas
 * CHANGE:    Initial implementation with flexible ranking and filtering
 * RISK:      Low - Project ranking with efficient aggregation queries
 */
func (crs *ComprehensiveReportingService) GetProjectRankings(ctx context.Context, period TimePeriod, userID string) ([]ProjectTime, error) {
	startTime := time.Now()
	queryType := "project_rankings"
	
	// Generate rankings
	rankings, err := crs.reportingQueries.GetProjectRankings(ctx, period, userID)
	executionTime := time.Since(startTime)

	resultSize := int64(len(rankings) * 100) // Estimate
	crs.performanceMonitor.RecordQuery(queryType, executionTime, false, resultSize, userID, period.Type, err)

	return rankings, err
}

/**
 * CONTEXT:   Get comprehensive performance analytics for the reporting system
 * INPUT:     Time period for performance analysis
 * OUTPUT:    System performance metrics, optimization recommendations, and health status
 * BUSINESS:  Performance analytics enable proactive system optimization
 * CHANGE:    Initial implementation with comprehensive performance analysis
 * RISK:      Low - Performance analytics for system monitoring and optimization
 */
func (crs *ComprehensiveReportingService) GetPerformanceAnalytics(period time.Duration) *SystemPerformanceReport {
	// Get performance statistics
	perfStats := crs.performanceMonitor.GetStats(period)
	
	// Get cache statistics  
	cacheStats := crs.queryCache.GetStats()
	
	// Get optimization analysis
	optimizationReport := crs.queryOptimizer.GenerateOptimizationReport(period)
	
	// Create comprehensive performance report
	report := &SystemPerformanceReport{
		Period:               period,
		GeneratedAt:          time.Now(),
		PerformanceStats:     perfStats,
		CacheStats:          cacheStats,
		OptimizationReport:  *optimizationReport,
		SystemHealthScore:   crs.calculateSystemHealthScore(perfStats, cacheStats),
		RecommendedActions:  crs.generateSystemRecommendations(perfStats, cacheStats),
	}

	return report
}

/**
 * CONTEXT:   Comprehensive system performance report with all metrics
 * INPUT:     Performance data from all system components
 * OUTPUT:    Unified performance report for system monitoring
 * BUSINESS:  System performance report enables comprehensive performance management
 * CHANGE:    Initial system performance report structure
 * RISK:      Low - Data structure for system performance monitoring
 */
type SystemPerformanceReport struct {
	Period              time.Duration        `json:"period"`
	GeneratedAt         time.Time           `json:"generated_at"`
	PerformanceStats    PerformanceStats    `json:"performance_stats"`
	CacheStats          CacheStats          `json:"cache_stats"`
	OptimizationReport  OptimizationReport  `json:"optimization_report"`
	SystemHealthScore   float64             `json:"system_health_score"`
	RecommendedActions  []string            `json:"recommended_actions"`
	AlertsAndWarnings   []string            `json:"alerts_and_warnings"`
}

/**
 * CONTEXT:   Calculate overall system health score from component metrics
 * INPUT:     Performance and cache statistics
 * OUTPUT:    Composite health score percentage (0-100)
 * BUSINESS:  System health score provides single metric for system performance
 * CHANGE:    Initial implementation with weighted component scoring
 * RISK:      Low - Health scoring for system performance assessment
 */
func (crs *ComprehensiveReportingService) calculateSystemHealthScore(perfStats PerformanceStats, cacheStats CacheStats) float64 {
	score := 100.0
	
	// Query performance (40% weight)
	if perfStats.AverageExecutionTime > DefaultSlowQueryThreshold {
		perfPenalty := 40.0 * (float64(perfStats.AverageExecutionTime) / float64(500*time.Millisecond))
		if perfPenalty > 40.0 {
			perfPenalty = 40.0
		}
		score -= perfPenalty
	}
	
	// Cache performance (30% weight)
	if cacheStats.HitRate < 70.0 {
		cachePenalty := 30.0 * ((70.0 - cacheStats.HitRate) / 70.0)
		score -= cachePenalty
	}
	
	// Error rate (20% weight)
	if perfStats.ErrorRate > 0 {
		errorPenalty := 20.0 * (perfStats.ErrorRate / 10.0) // Max 10% error rate
		if errorPenalty > 20.0 {
			errorPenalty = 20.0
		}
		score -= errorPenalty
	}
	
	// Memory usage (10% weight)
	if cacheStats.MemoryUsage > 80.0 {
		memoryPenalty := 10.0 * ((cacheStats.MemoryUsage - 80.0) / 20.0)
		score -= memoryPenalty
	}
	
	if score < 0 {
		score = 0
	}
	
	return score
}

/**
 * CONTEXT:   Generate system optimization recommendations
 * INPUT:     Performance and cache statistics for analysis
 * OUTPUT:    Actionable recommendations for system improvement
 * BUSINESS:  System recommendations guide optimization and maintenance actions
 * CHANGE:    Initial implementation with comprehensive recommendation logic
 * RISK:      Low - Recommendation generation for system optimization
 */
func (crs *ComprehensiveReportingService) generateSystemRecommendations(perfStats PerformanceStats, cacheStats CacheStats) []string {
	recommendations := make([]string, 0)
	
	// Performance recommendations
	if perfStats.AverageExecutionTime > DefaultSlowQueryThreshold {
		recommendations = append(recommendations, 
			fmt.Sprintf("Average query time is %v, consider query optimization", perfStats.AverageExecutionTime))
	}
	
	if perfStats.P95ExecutionTime > 200*time.Millisecond {
		recommendations = append(recommendations, 
			fmt.Sprintf("95th percentile query time is %v, investigate slow queries", perfStats.P95ExecutionTime))
	}
	
	// Cache recommendations
	if cacheStats.HitRate < 70.0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Cache hit rate is %.1f%%, consider increasing cache size or TTL", cacheStats.HitRate))
	}
	
	if cacheStats.MemoryUsage > 80.0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Cache memory usage is %.1f%%, consider increasing max cache size", cacheStats.MemoryUsage))
	}
	
	// Index recommendations
	if perfStats.SlowQueryCount > perfStats.TotalQueries/10 {
		recommendations = append(recommendations, 
			"High number of slow queries detected, review index recommendations")
	}
	
	// Error recommendations
	if perfStats.ErrorRate > 1.0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Error rate is %.1f%%, investigate query failures", perfStats.ErrorRate))
	}
	
	return recommendations
}

/**
 * CONTEXT:   Invalidate cached reports for user when new data arrives
 * INPUT:     User ID and optional specific report types to invalidate
 * OUTPUT:    Number of cache entries invalidated
 * BUSINESS:  Cache invalidation ensures users see updated data after activities
 * CHANGE:    Initial implementation with selective cache invalidation
 * RISK:      Low - Cache invalidation for data consistency
 */
func (crs *ComprehensiveReportingService) InvalidateUserReports(userID string, reportTypes []string) int {
	totalInvalidated := 0
	
	if len(reportTypes) == 0 {
		// Invalidate all user reports
		pattern := fmt.Sprintf("*_%s_*", userID)
		totalInvalidated = crs.queryCache.InvalidatePattern(pattern)
	} else {
		// Invalidate specific report types
		for _, reportType := range reportTypes {
			pattern := fmt.Sprintf("%s_%s_*", reportType, userID)
			totalInvalidated += crs.queryCache.InvalidatePattern(pattern)
		}
	}
	
	return totalInvalidated
}

/**
 * CONTEXT:   Estimate memory size of report structures for caching and monitoring
 * INPUT:     Report interface (various report types)
 * OUTPUT:    Estimated memory size in bytes
 * BUSINESS:  Size estimation enables efficient cache memory management
 * CHANGE:    Initial implementation with type-specific size calculations
 * RISK:      Low - Memory estimation for cache optimization
 */
func (crs *ComprehensiveReportingService) estimateReportSize(report interface{}) int64 {
	switch r := report.(type) {
	case *DailyReport:
		size := int64(300) // Base size
		size += int64(len(r.ProjectBreakdown)) * 150
		size += int64(len(r.HourlyBreakdown)) * 80
		return size
		
	case *WeeklyReport:
		size := int64(400) // Base size
		size += int64(len(r.DailyBreakdown)) * 300
		size += int64(len(r.TopProjects)) * 150
		return size
		
	case *MonthlyReport:
		size := int64(600) // Base size
		size += int64(len(r.WeeklyBreakdown)) * 400
		size += int64(len(r.CalendarHeatmap)) * 100
		size += int64(len(r.TopProjects)) * 150
		return size
		
	case *ProjectReport:
		size := int64(350) // Base size
		size += int64(len(r.DailyBreakdown)) * 100
		size += int64(len(r.HourlyPattern)) * 80
		return size
		
	default:
		return 1000 // Default estimate
	}
}

/**
 * CONTEXT:   Graceful shutdown of reporting service with resource cleanup
 * INPUT:     Context for timeout control during shutdown
 * OUTPUT:    Clean shutdown with all resources properly released
 * BUSINESS:  Proper shutdown prevents resource leaks and data corruption
 * CHANGE:    Initial implementation with comprehensive resource cleanup
 * RISK:      Low - Standard resource cleanup for graceful shutdown
 */
func (crs *ComprehensiveReportingService) Shutdown(ctx context.Context) error {
	fmt.Println("Shutting down comprehensive reporting service...")
	
	// Clear cache to free memory
	clearedEntries := crs.queryCache.Clear()
	fmt.Printf("Cleared %d cache entries\n", clearedEntries)
	
	// Get final performance stats
	finalStats := crs.performanceMonitor.GetStats(1 * time.Hour)
	fmt.Printf("Final performance: %d queries, avg %v, cache hit rate %.1f%%\n", 
		finalStats.TotalQueries, finalStats.AverageExecutionTime, finalStats.CacheHitRate)
	
	// Close database connections
	if err := crs.connectionManager.Close(); err != nil {
		return fmt.Errorf("failed to close database connections: %w", err)
	}
	
	fmt.Println("Comprehensive reporting service shutdown complete")
	return nil
}

/**
 * CONTEXT:   Health check for the comprehensive reporting service
 * INPUT:     Context for timeout control
 * OUTPUT:    Health status and system metrics
 * BUSINESS:  Health checks enable monitoring and alerting for service availability
 * CHANGE:    Initial implementation with comprehensive health validation
 * RISK:      Low - Non-destructive health check with timeout protection
 */
func (crs *ComprehensiveReportingService) HealthCheck(ctx context.Context) (*ServiceHealthStatus, error) {
	health := &ServiceHealthStatus{
		Service:     "ComprehensiveReportingService",
		CheckedAt:   time.Now(),
		Status:      "healthy",
		Details:     make(map[string]interface{}),
	}
	
	// Check database connectivity
	if err := crs.connectionManager.HealthCheck(ctx); err != nil {
		health.Status = "unhealthy"
		health.Details["database_error"] = err.Error()
		return health, err
	}
	
	// Check cache status
	cacheStats := crs.queryCache.GetStats()
	health.Details["cache_entries"] = cacheStats.EntryCount
	health.Details["cache_hit_rate"] = cacheStats.HitRate
	
	// Check recent performance
	perfStats := crs.performanceMonitor.GetStats(5 * time.Minute)
	health.Details["recent_queries"] = perfStats.TotalQueries
	health.Details["average_response_time"] = perfStats.AverageExecutionTime.String()
	
	// Calculate overall health score
	healthScore := crs.calculateSystemHealthScore(perfStats, cacheStats)
	health.Details["health_score"] = healthScore
	
	if healthScore < 70.0 {
		health.Status = "degraded"
	}
	
	if healthScore < 50.0 {
		health.Status = "unhealthy"
	}
	
	return health, nil
}

/**
 * CONTEXT:   Service health status for monitoring and alerting
 * INPUT:     Health check results and system metrics
 * OUTPUT:    Structured health status for external monitoring
 * BUSINESS:  Health status enables monitoring and automated alerting
 * CHANGE:    Initial health status structure
 * RISK:      Low - Data structure for service health reporting
 */
type ServiceHealthStatus struct {
	Service   string                 `json:"service"`
	Status    string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	CheckedAt time.Time             `json:"checked_at"`
	Details   map[string]interface{} `json:"details"`
}