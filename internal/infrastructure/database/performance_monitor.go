/**
 * CONTEXT:   Performance monitoring and optimization for KuzuDB reporting queries
 * INPUT:     Query execution times, cache hit rates, memory usage, and error metrics
 * OUTPUT:    Performance insights, slow query alerts, and optimization recommendations
 * BUSINESS:  Sub-100ms query response times are critical for excellent user experience
 * CHANGE:    Initial implementation with comprehensive performance tracking
 * RISK:      Low - Monitoring has minimal overhead and provides valuable insights
 */

package database

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

/**
 * CONTEXT:   Query performance metrics for monitoring and optimization
 * INPUT:     Individual query execution data and system performance metrics
 * OUTPUT:    Structured performance data for analysis and alerting
 * BUSINESS:  Performance monitoring enables proactive optimization and issue detection
 * CHANGE:    Initial performance metrics structure with comprehensive tracking
 * RISK:      Low - Data structure for performance monitoring
 */
type QueryMetrics struct {
	QueryType      string        `json:"query_type"`
	ExecutionTime  time.Duration `json:"execution_time"`
	CacheHit       bool          `json:"cache_hit"`
	ResultSize     int64         `json:"result_size"`
	UserID         string        `json:"user_id"`
	QueryParams    string        `json:"query_params"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	Timestamp      time.Time     `json:"timestamp"`
}

/**
 * CONTEXT:   Performance statistics aggregated over time periods
 * INPUT:     Aggregated metrics from multiple query executions
 * OUTPUT:    Statistical analysis of query performance patterns
 * BUSINESS:  Performance stats help identify trends and optimization opportunities
 * CHANGE:    Initial performance statistics structure for analysis
 * RISK:      Low - Aggregated statistics for performance insights
 */
type PerformanceStats struct {
	TotalQueries        int64         `json:"total_queries"`
	SuccessfulQueries   int64         `json:"successful_queries"`
	FailedQueries       int64         `json:"failed_queries"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	MedianExecutionTime  time.Duration `json:"median_execution_time"`
	P95ExecutionTime     time.Duration `json:"p95_execution_time"`
	P99ExecutionTime     time.Duration `json:"p99_execution_time"`
	SlowQueryCount      int64         `json:"slow_query_count"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
	ErrorRate           float64       `json:"error_rate"`
	QueriesPerSecond    float64       `json:"queries_per_second"`
	Period              time.Duration `json:"period"`
	LastUpdated         time.Time     `json:"last_updated"`
}

/**
 * CONTEXT:   Performance monitor with real-time metrics collection and analysis
 * INPUT:     Query execution events, performance thresholds, and monitoring configuration
 * OUTPUT:    Real-time performance insights, alerts, and optimization recommendations
 * BUSINESS:  Performance monitoring ensures reporting system meets speed requirements
 * CHANGE:    Initial implementation with real-time monitoring and alerting
 * RISK:      Medium - Monitoring overhead must be minimal to avoid affecting performance
 */
type PerformanceMonitor struct {
	recentMetrics    []QueryMetrics
	metricsBuffer    []QueryMetrics
	slowQueryLimit   time.Duration
	maxMetricsBuffer int
	alertThreshold   time.Duration
	alertCallback    func(QueryMetrics)
	mu               sync.RWMutex
	startTime        time.Time
	
	// Performance counters
	totalQueries      int64
	successfulQueries int64
	failedQueries     int64
	cacheHits         int64
	slowQueries       int64
}

// Performance monitoring configuration
const (
	DefaultSlowQueryThreshold = 100 * time.Millisecond
	DefaultAlertThreshold     = 500 * time.Millisecond
	MaxMetricsBufferSize      = 10000
	MetricsRetentionPeriod    = 1 * time.Hour
	StatsUpdateInterval       = 30 * time.Second
)

/**
 * CONTEXT:   Create new performance monitor with optimized configuration
 * INPUT:     Monitoring configuration and alert callback function
 * OUTPUT:    Initialized performance monitor ready for query tracking
 * BUSINESS:  Performance monitoring setup with sensible defaults for reporting queries
 * CHANGE:    Initial performance monitor constructor with background processing
 * RISK:      Low - Standard monitoring initialization with safe defaults
 */
func NewPerformanceMonitor(alertCallback func(QueryMetrics)) *PerformanceMonitor {
	monitor := &PerformanceMonitor{
		recentMetrics:    make([]QueryMetrics, 0),
		metricsBuffer:    make([]QueryMetrics, 0, MaxMetricsBufferSize),
		slowQueryLimit:   DefaultSlowQueryThreshold,
		maxMetricsBuffer: MaxMetricsBufferSize,
		alertThreshold:   DefaultAlertThreshold,
		alertCallback:    alertCallback,
		startTime:        time.Now(),
		
		totalQueries:      0,
		successfulQueries: 0,
		failedQueries:     0,
		cacheHits:         0,
		slowQueries:       0,
	}

	// Start background processing
	go monitor.backgroundProcessing()

	return monitor
}

/**
 * CONTEXT:   Record query execution metrics with performance analysis
 * INPUT:     Query execution details, timing, cache status, and results
 * OUTPUT:     Updated performance metrics and potential alerts
 * BUSINESS:  Accurate performance recording enables optimization and problem detection
 * CHANGE:    Initial implementation with comprehensive metric capture
 * RISK:      Low - Metric recording with minimal performance impact
 */
func (pm *PerformanceMonitor) RecordQuery(queryType string, executionTime time.Duration, cacheHit bool, resultSize int64, userID string, params string, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Create metrics entry
	metrics := QueryMetrics{
		QueryType:     queryType,
		ExecutionTime: executionTime,
		CacheHit:      cacheHit,
		ResultSize:    resultSize,
		UserID:        userID,
		QueryParams:   params,
		Timestamp:     time.Now(),
	}

	if err != nil {
		metrics.ErrorMessage = err.Error()
		pm.failedQueries++
	} else {
		pm.successfulQueries++
	}

	// Update counters
	pm.totalQueries++
	if cacheHit {
		pm.cacheHits++
	}
	if executionTime > pm.slowQueryLimit {
		pm.slowQueries++
	}

	// Add to buffer
	if len(pm.metricsBuffer) >= pm.maxMetricsBuffer {
		// Remove oldest entry
		pm.metricsBuffer = pm.metricsBuffer[1:]
	}
	pm.metricsBuffer = append(pm.metricsBuffer, metrics)

	// Check for alerts
	if executionTime > pm.alertThreshold && pm.alertCallback != nil {
		go pm.alertCallback(metrics) // Non-blocking alert
	}

	// Log slow queries for debugging
	if executionTime > pm.slowQueryLimit {
		pm.logSlowQuery(metrics)
	}
}

/**
 * CONTEXT:   Log slow query details for performance optimization
 * INPUT:     Query metrics with slow execution time
 * OUTPUT:    Detailed logging for slow query analysis
 * BUSINESS:  Slow query logging helps identify optimization opportunities
 * CHANGE:    Initial implementation with structured slow query logging
 * RISK:      Low - Logging for performance debugging
 */
func (pm *PerformanceMonitor) logSlowQuery(metrics QueryMetrics) {
	// Log slow query details
	fmt.Printf("SLOW QUERY [%s]: %v - Type: %s, User: %s, Cache: %v, Size: %d bytes, Params: %s\n",
		metrics.Timestamp.Format("2006-01-02 15:04:05"),
		metrics.ExecutionTime,
		metrics.QueryType,
		metrics.UserID,
		metrics.CacheHit,
		metrics.ResultSize,
		metrics.QueryParams)
}

/**
 * CONTEXT:   Generate comprehensive performance statistics for monitoring
 * INPUT:     Time period for statistics aggregation
 * OUTPUT:    Detailed performance stats with percentiles and trends
 * BUSINESS:  Performance statistics enable data-driven optimization decisions
 * CHANGE:    Initial implementation with comprehensive statistical analysis
 * RISK:      Medium - Statistical calculations require efficient algorithms
 */
func (pm *PerformanceMonitor) GetStats(period time.Duration) PerformanceStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-period)

	// Filter metrics for the requested period
	relevantMetrics := make([]QueryMetrics, 0)
	for _, metric := range pm.metricsBuffer {
		if metric.Timestamp.After(cutoff) {
			relevantMetrics = append(relevantMetrics, metric)
		}
	}

	if len(relevantMetrics) == 0 {
		return PerformanceStats{
			Period:      period,
			LastUpdated: now,
		}
	}

	// Calculate statistics
	stats := PerformanceStats{
		TotalQueries:  int64(len(relevantMetrics)),
		Period:        period,
		LastUpdated:   now,
	}

	// Count successes, failures, and cache hits
	cacheHits := int64(0)
	executionTimes := make([]time.Duration, 0, len(relevantMetrics))
	
	for _, metric := range relevantMetrics {
		executionTimes = append(executionTimes, metric.ExecutionTime)
		
		if metric.ErrorMessage == "" {
			stats.SuccessfulQueries++
		} else {
			stats.FailedQueries++
		}
		
		if metric.CacheHit {
			cacheHits++
		}
		
		if metric.ExecutionTime > pm.slowQueryLimit {
			stats.SlowQueryCount++
		}
	}

	// Calculate rates
	if stats.TotalQueries > 0 {
		stats.CacheHitRate = float64(cacheHits) / float64(stats.TotalQueries) * 100.0
		stats.ErrorRate = float64(stats.FailedQueries) / float64(stats.TotalQueries) * 100.0
		stats.QueriesPerSecond = float64(stats.TotalQueries) / period.Seconds()
	}

	// Calculate execution time statistics
	if len(executionTimes) > 0 {
		sort.Slice(executionTimes, func(i, j int) bool {
			return executionTimes[i] < executionTimes[j]
		})

		// Average
		total := time.Duration(0)
		for _, duration := range executionTimes {
			total += duration
		}
		stats.AverageExecutionTime = total / time.Duration(len(executionTimes))

		// Percentiles
		stats.MedianExecutionTime = pm.calculatePercentile(executionTimes, 50)
		stats.P95ExecutionTime = pm.calculatePercentile(executionTimes, 95)
		stats.P99ExecutionTime = pm.calculatePercentile(executionTimes, 99)
	}

	return stats
}

/**
 * CONTEXT:   Calculate execution time percentiles for performance analysis
 * INPUT:     Sorted array of execution times and desired percentile
 * OUTPUT:    Execution time at the specified percentile
 * BUSINESS:  Percentile analysis reveals query performance distribution
 * CHANGE:    Initial implementation with standard percentile calculation
 * RISK:      Low - Standard statistical calculation for performance analysis
 */
func (pm *PerformanceMonitor) calculatePercentile(sortedTimes []time.Duration, percentile int) time.Duration {
	if len(sortedTimes) == 0 {
		return 0
	}

	index := (percentile * len(sortedTimes)) / 100
	if index >= len(sortedTimes) {
		index = len(sortedTimes) - 1
	}
	if index < 0 {
		index = 0
	}

	return sortedTimes[index]
}

/**
 * CONTEXT:   Get recent query metrics for real-time monitoring
 * INPUT:     Number of recent queries to retrieve
 * OUTPUT:    Array of most recent query metrics
 * BUSINESS:  Recent query analysis helps identify immediate performance issues
 * CHANGE:    Initial implementation with configurable query count
 * RISK:      Low - Simple recent metrics retrieval
 */
func (pm *PerformanceMonitor) GetRecentMetrics(count int) []QueryMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if count <= 0 || len(pm.metricsBuffer) == 0 {
		return []QueryMetrics{}
	}

	startIndex := len(pm.metricsBuffer) - count
	if startIndex < 0 {
		startIndex = 0
	}

	recent := make([]QueryMetrics, len(pm.metricsBuffer[startIndex:]))
	copy(recent, pm.metricsBuffer[startIndex:])

	return recent
}

/**
 * CONTEXT:   Get slow queries for optimization analysis
 * INPUT:     Time period and count limit for slow query retrieval
 * OUTPUT:    Array of slow queries with execution details
 * BUSINESS:  Slow query analysis guides performance optimization efforts
 * CHANGE:    Initial implementation with time-based filtering
 * RISK:      Low - Slow query retrieval for performance analysis
 */
func (pm *PerformanceMonitor) GetSlowQueries(period time.Duration, limit int) []QueryMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	cutoff := time.Now().Add(-period)
	slowQueries := make([]QueryMetrics, 0)

	for _, metric := range pm.metricsBuffer {
		if metric.Timestamp.After(cutoff) && metric.ExecutionTime > pm.slowQueryLimit {
			slowQueries = append(slowQueries, metric)
		}
	}

	// Sort by execution time (slowest first)
	sort.Slice(slowQueries, func(i, j int) bool {
		return slowQueries[i].ExecutionTime > slowQueries[j].ExecutionTime
	})

	// Limit results
	if limit > 0 && len(slowQueries) > limit {
		slowQueries = slowQueries[:limit]
	}

	return slowQueries
}

/**
 * CONTEXT:   Analyze query patterns for optimization recommendations
 * INPUT:     Time period for pattern analysis
 * OUTPUT:    Performance insights and optimization suggestions
 * BUSINESS:  Pattern analysis enables proactive performance improvements
 * CHANGE:    Initial implementation with basic pattern detection
 * RISK:      Medium - Pattern analysis algorithms must be efficient
 */
func (pm *PerformanceMonitor) AnalyzePatterns(period time.Duration) *PerformanceAnalysis {
	stats := pm.GetStats(period)
	recentMetrics := pm.GetRecentMetrics(100)

	analysis := &PerformanceAnalysis{
		Period:      period,
		GeneratedAt: time.Now(),
		Stats:       stats,
	}

	// Analyze query types
	queryTypeStats := make(map[string]*QueryTypeStats)
	for _, metric := range recentMetrics {
		if stat, exists := queryTypeStats[metric.QueryType]; exists {
			stat.Count++
			stat.TotalTime += metric.ExecutionTime
			if metric.ExecutionTime > stat.MaxTime {
				stat.MaxTime = metric.ExecutionTime
			}
			if metric.ExecutionTime < stat.MinTime {
				stat.MinTime = metric.ExecutionTime
			}
		} else {
			queryTypeStats[metric.QueryType] = &QueryTypeStats{
				QueryType: metric.QueryType,
				Count:     1,
				TotalTime: metric.ExecutionTime,
				MinTime:   metric.ExecutionTime,
				MaxTime:   metric.ExecutionTime,
			}
		}
	}

	// Calculate averages and generate recommendations
	for _, stat := range queryTypeStats {
		stat.AvgTime = stat.TotalTime / time.Duration(stat.Count)
		analysis.QueryTypeStats = append(analysis.QueryTypeStats, *stat)
		
		// Generate recommendations
		if stat.AvgTime > DefaultSlowQueryThreshold {
			analysis.Recommendations = append(analysis.Recommendations, 
				fmt.Sprintf("Query type '%s' averaging %v - consider optimization", 
					stat.QueryType, stat.AvgTime))
		}
	}

	// Cache analysis
	if stats.CacheHitRate < 70.0 {
		analysis.Recommendations = append(analysis.Recommendations,
			fmt.Sprintf("Cache hit rate is %.1f%% - consider increasing cache TTL or size", 
				stats.CacheHitRate))
	}

	// Error analysis
	if stats.ErrorRate > 5.0 {
		analysis.Recommendations = append(analysis.Recommendations,
			fmt.Sprintf("Error rate is %.1f%% - investigate query failures", 
				stats.ErrorRate))
	}

	return analysis
}

/**
 * CONTEXT:   Performance analysis results with recommendations
 * INPUT:     Analyzed performance data and patterns
 * OUTPUT:    Structured analysis with actionable recommendations
 * BUSINESS:  Performance analysis guides optimization priorities and actions
 * CHANGE:    Initial analysis structure with recommendations
 * RISK:      Low - Data structure for performance analysis results
 */
type PerformanceAnalysis struct {
	Period           time.Duration     `json:"period"`
	GeneratedAt      time.Time         `json:"generated_at"`
	Stats            PerformanceStats  `json:"stats"`
	QueryTypeStats   []QueryTypeStats  `json:"query_type_stats"`
	Recommendations  []string          `json:"recommendations"`
	HealthScore      float64           `json:"health_score"`
}

type QueryTypeStats struct {
	QueryType string        `json:"query_type"`
	Count     int64         `json:"count"`
	TotalTime time.Duration `json:"total_time"`
	AvgTime   time.Duration `json:"avg_time"`
	MinTime   time.Duration `json:"min_time"`
	MaxTime   time.Duration `json:"max_time"`
}

/**
 * CONTEXT:   Background processing for metrics cleanup and analysis
 * INPUT:     No parameters, runs continuously for housekeeping
 * OUTPUT:    Automatic metrics cleanup and periodic analysis
 * BUSINESS:  Background processing maintains performance monitoring efficiency
 * CHANGE:    Initial implementation with periodic cleanup and analysis
 * RISK:      Low - Background processing with minimal resource usage
 */
func (pm *PerformanceMonitor) backgroundProcessing() {
	ticker := time.NewTicker(StatsUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.cleanupOldMetrics()
			pm.performPeriodicAnalysis()
		}
	}
}

/**
 * CONTEXT:   Remove old metrics to prevent memory growth
 * INPUT:     No parameters, uses retention period configuration
 * OUTPUT:    Cleaned metrics buffer within memory limits
 * BUSINESS:  Metrics cleanup prevents memory leaks in long-running systems
 * CHANGE:    Initial implementation with time-based cleanup
 * RISK:      Low - Memory management for metrics buffer
 */
func (pm *PerformanceMonitor) cleanupOldMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	cutoff := time.Now().Add(-MetricsRetentionPeriod)
	newBuffer := make([]QueryMetrics, 0)

	for _, metric := range pm.metricsBuffer {
		if metric.Timestamp.After(cutoff) {
			newBuffer = append(newBuffer, metric)
		}
	}

	pm.metricsBuffer = newBuffer
}

/**
 * CONTEXT:   Perform periodic performance analysis and health checks
 * INPUT:     No parameters, analyzes current performance state
 * OUTPUT:    Performance health assessment and optimization alerts
 * BUSINESS:  Periodic analysis enables proactive performance management
 * CHANGE:    Initial implementation with health scoring
 * RISK:      Low - Periodic analysis for performance insights
 */
func (pm *PerformanceMonitor) performPeriodicAnalysis() {
	// Get recent performance stats
	stats := pm.GetStats(10 * time.Minute)
	
	// Calculate health score
	healthScore := pm.calculateHealthScore(stats)
	
	// Log health status
	if healthScore < 70.0 {
		fmt.Printf("PERFORMANCE WARNING: Health score %.1f%% - Average: %v, P95: %v, Cache: %.1f%%, Errors: %.1f%%\n",
			healthScore, stats.AverageExecutionTime, stats.P95ExecutionTime, 
			stats.CacheHitRate, stats.ErrorRate)
	}
}

/**
 * CONTEXT:   Calculate overall performance health score
 * INPUT:     Performance statistics for health assessment
 * OUTPUT:    Health score percentage (0-100)
 * BUSINESS:  Health scoring provides simple performance overview
 * CHANGE:    Initial implementation with weighted scoring
 * RISK:      Low - Health scoring algorithm for performance assessment
 */
func (pm *PerformanceMonitor) calculateHealthScore(stats PerformanceStats) float64 {
	if stats.TotalQueries == 0 {
		return 100.0 // No queries = perfect health
	}

	score := 100.0

	// Execution time scoring (40% weight)
	if stats.AverageExecutionTime > DefaultSlowQueryThreshold {
		timeScore := 40.0 * (1.0 - float64(stats.AverageExecutionTime-DefaultSlowQueryThreshold)/float64(time.Second))
		if timeScore < 0 {
			timeScore = 0
		}
		score -= (40.0 - timeScore)
	}

	// Cache hit rate scoring (30% weight)
	cacheScore := stats.CacheHitRate * 0.3
	score -= (30.0 - cacheScore)

	// Error rate scoring (20% weight)
	errorScore := (100.0 - stats.ErrorRate) * 0.2
	score -= (20.0 - errorScore)

	// Slow query ratio scoring (10% weight)
	slowQueryRatio := float64(stats.SlowQueryCount) / float64(stats.TotalQueries) * 100.0
	slowQueryScore := (100.0 - slowQueryRatio) * 0.1
	score -= (10.0 - slowQueryScore)

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

/**
 * CONTEXT:   Set custom performance thresholds for monitoring
 * INPUT:     Slow query threshold and alert threshold durations
 * OUTPUT:    Updated monitoring thresholds
 * BUSINESS:  Customizable thresholds adapt monitoring to different performance requirements
 * CHANGE:    Initial implementation with threshold configuration
 * RISK:      Low - Configuration update for monitoring parameters
 */
func (pm *PerformanceMonitor) SetThresholds(slowQueryThreshold, alertThreshold time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.slowQueryLimit = slowQueryThreshold
	pm.alertThreshold = alertThreshold
}

/**
 * CONTEXT:   Get current performance monitoring configuration
 * INPUT:     No parameters, returns current monitoring settings
 * OUTPUT:    Current monitoring thresholds and configuration
 * BUSINESS:  Configuration visibility helps understand monitoring behavior
 * CHANGE:    Initial implementation with configuration export
 * RISK:      Low - Configuration query for monitoring settings
 */
type MonitorConfig struct {
	SlowQueryThreshold time.Duration `json:"slow_query_threshold"`
	AlertThreshold     time.Duration `json:"alert_threshold"`
	MaxMetricsBuffer   int           `json:"max_metrics_buffer"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	UpdateInterval     time.Duration `json:"update_interval"`
}

func (pm *PerformanceMonitor) GetConfig() MonitorConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return MonitorConfig{
		SlowQueryThreshold: pm.slowQueryLimit,
		AlertThreshold:     pm.alertThreshold,
		MaxMetricsBuffer:   pm.maxMetricsBuffer,
		RetentionPeriod:    MetricsRetentionPeriod,
		UpdateInterval:     StatsUpdateInterval,
	}
}