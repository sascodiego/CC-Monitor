/**
 * CONTEXT:   Query optimization utilities for KuzuDB reporting performance
 * INPUT:     Query patterns, execution plans, index usage, and performance metrics
 * OUTPUT:    Optimized queries, index recommendations, and performance improvements
 * BUSINESS:  Query optimization ensures all reports load in under 100ms for excellent UX
 * CHANGE:    Initial implementation of query optimization and tuning utilities
 * RISK:      Medium - Query optimization requires careful testing to avoid breaking changes
 */

package database

import (
	"context"
	"fmt"
	"strings"
	"time"
)

/**
 * CONTEXT:   Query optimizer with pattern recognition and automatic tuning
 * INPUT:     Query execution patterns, performance metrics, and optimization rules
 * OUTPUT:    Optimized queries, index suggestions, and performance improvements
 * BUSINESS:  Automated query optimization maintains fast report generation
 * CHANGE:    Initial implementation with rule-based optimization
 * RISK:      Medium - Optimization changes must be validated to ensure correctness
 */
type QueryOptimizer struct {
	connectionManager *KuzuConnectionManager
	performanceMonitor *PerformanceMonitor
	optimizationRules []OptimizationRule
	indexRecommendations map[string]IndexRecommendation
}

/**
 * CONTEXT:   Optimization rule for automatic query improvement
 * INPUT:     Query pattern matching and optimization transformation
 * OUTPUT:    Rule-based query transformation for better performance
 * BUSINESS:  Optimization rules capture proven performance improvements
 * CHANGE:    Initial rule structure for query optimization patterns
 * RISK:      Low - Data structure for optimization rule definition
 */
type OptimizationRule struct {
	Name        string
	Pattern     string
	Replacement string
	Condition   func(QueryMetrics) bool
	Improvement string
	RiskLevel   string
}

/**
 * CONTEXT:   Index recommendation for query performance improvement
 * INPUT:     Query patterns requiring indexed access
 * OUTPUT:    Index creation recommendations with performance impact estimates
 * BUSINESS:  Index recommendations guide database optimization for faster queries
 * CHANGE:    Initial index recommendation structure
 * RISK:      Low - Data structure for index optimization suggestions
 */
type IndexRecommendation struct {
	TableName     string  `json:"table_name"`
	ColumnNames   []string `json:"column_names"`
	IndexType     string  `json:"index_type"`
	Priority      int     `json:"priority"`
	EstimatedGain string  `json:"estimated_gain"`
	QueryTypes    []string `json:"query_types"`
	CreationSQL   string  `json:"creation_sql"`
}

// Pre-built optimization rules for common query patterns
var DefaultOptimizationRules = []OptimizationRule{
	{
		Name:        "TimeRangeOptimization",
		Pattern:     "WHERE.*start_time.*AND.*end_time",
		Replacement: "WHERE start_time >= $start AND start_time < $end",
		Condition:   func(m QueryMetrics) bool { return m.ExecutionTime > 50*time.Millisecond },
		Improvement: "Use single column range for better index utilization",
		RiskLevel:   "Low",
	},
	{
		Name:        "ProjectFilterOptimization", 
		Pattern:     "MATCH.*Project.*WHERE.*name",
		Replacement: "MATCH (p:Project {name: $project_name})",
		Condition:   func(m QueryMetrics) bool { return strings.Contains(m.QueryParams, "project_name") },
		Improvement: "Direct property match is faster than WHERE filter",
		RiskLevel:   "Low",
	},
	{
		Name:        "SessionUserOptimization",
		Pattern:     "MATCH.*User.*Session.*WHERE.*user_id",
		Replacement: "MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)",
		Condition:   func(m QueryMetrics) bool { return m.QueryType == "daily" || m.QueryType == "weekly" },
		Improvement: "Start traversal from specific user for better performance",
		RiskLevel:   "Low",
	},
	{
		Name:        "AggregationOptimization",
		Pattern:     "SUM.*duration_hours.*COUNT.*",
		Replacement: "SUM(duration_hours) as hours, COUNT(*) as count",
		Condition:   func(m QueryMetrics) bool { return m.ResultSize > 1000 },
		Improvement: "Combine aggregations to reduce query complexity",
		RiskLevel:   "Low",
	},
}

/**
 * CONTEXT:   Create new query optimizer with default rules and monitoring
 * INPUT:     Database connection manager and performance monitor
 * OUTPUT:    Initialized query optimizer ready for automatic tuning
 * BUSINESS:  Query optimizer setup enables automatic performance improvements
 * CHANGE:    Initial optimizer constructor with default optimization rules
 * RISK:      Low - Standard optimizer initialization with proven rules
 */
func NewQueryOptimizer(connectionManager *KuzuConnectionManager, performanceMonitor *PerformanceMonitor) *QueryOptimizer {
	optimizer := &QueryOptimizer{
		connectionManager: connectionManager,
		performanceMonitor: performanceMonitor,
		optimizationRules: make([]OptimizationRule, len(DefaultOptimizationRules)),
		indexRecommendations: make(map[string]IndexRecommendation),
	}

	// Copy default rules
	copy(optimizer.optimizationRules, DefaultOptimizationRules)

	// Generate initial index recommendations
	optimizer.generateIndexRecommendations()

	return optimizer
}

/**
 * CONTEXT:   Optimize query for better performance using pattern matching
 * INPUT:     Original Cypher query and execution context
 * OUTPUT:    Optimized query with performance improvements
 * BUSINESS:  Query optimization ensures fast report generation
 * CHANGE:    Initial implementation with pattern-based optimization
 * RISK:      Medium - Query transformations must preserve correctness
 */
func (qo *QueryOptimizer) OptimizeQuery(originalQuery string, queryType string, params map[string]interface{}) (string, error) {
	optimizedQuery := originalQuery
	appliedOptimizations := make([]string, 0)

	// Apply optimization rules
	for _, rule := range qo.optimizationRules {
		if qo.shouldApplyRule(rule, queryType, params) {
			if strings.Contains(optimizedQuery, rule.Pattern) {
				optimizedQuery = strings.ReplaceAll(optimizedQuery, rule.Pattern, rule.Replacement)
				appliedOptimizations = append(appliedOptimizations, rule.Name)
			}
		}
	}

	// Add query hints for better performance
	optimizedQuery = qo.addQueryHints(optimizedQuery, queryType)

	// Log optimization if changes were made
	if len(appliedOptimizations) > 0 {
		fmt.Printf("Query optimized with rules: %v\n", appliedOptimizations)
	}

	return optimizedQuery, nil
}

/**
 * CONTEXT:   Check if optimization rule should be applied to query
 * INPUT:     Optimization rule, query type, and parameters
 * OUTPUT:    Boolean indicating if rule should be applied
 * BUSINESS:  Rule application logic ensures safe and beneficial optimizations
 * CHANGE:    Initial implementation with condition checking
 * RISK:      Low - Rule condition evaluation for safe optimization
 */
func (qo *QueryOptimizer) shouldApplyRule(rule OptimizationRule, queryType string, params map[string]interface{}) bool {
	// Check if rule has specific conditions
	if rule.Condition != nil {
		// Create dummy metrics for condition check
		dummyMetrics := QueryMetrics{
			QueryType:   queryType,
			QueryParams: fmt.Sprintf("%v", params),
		}
		return rule.Condition(dummyMetrics)
	}
	
	return true // Apply rule by default if no condition
}

/**
 * CONTEXT:   Add query hints for better KuzuDB query execution
 * INPUT:     Query string and query type for hint selection
 * OUTPUT:    Query with performance hints added
 * BUSINESS:  Query hints guide database optimizer for better performance
 * CHANGE:    Initial implementation with KuzuDB-specific hints
 * RISK:      Low - Query hints improve performance without breaking functionality
 */
func (qo *QueryOptimizer) addQueryHints(query string, queryType string) string {
	hints := make([]string, 0)

	// Add hints based on query type
	switch queryType {
	case "daily":
		// Daily reports typically scan recent data
		hints = append(hints, "// HINT: Use time-based index for recent data")
		
	case "weekly":
		// Weekly reports need efficient date range scanning
		hints = append(hints, "// HINT: Use composite index on (user_id, start_time)")
		
	case "monthly":
		// Monthly reports aggregate larger datasets
		hints = append(hints, "// HINT: Consider parallel aggregation")
		
	case "project":
		// Project reports focus on specific projects
		hints = append(hints, "// HINT: Use project-based index")
	}

	// Add general performance hints
	if strings.Contains(query, "ORDER BY") {
		hints = append(hints, "// HINT: Consider index on ORDER BY columns")
	}

	if strings.Contains(query, "SUM") || strings.Contains(query, "COUNT") {
		hints = append(hints, "// HINT: Aggregation may benefit from pre-computed summaries")
	}

	// Prepend hints to query
	if len(hints) > 0 {
		hintComment := strings.Join(hints, "\n") + "\n"
		return hintComment + query
	}

	return query
}

/**
 * CONTEXT:   Generate index recommendations based on query patterns
 * INPUT:     No parameters, analyzes common query patterns
 * OUTPUT:    Updated index recommendations for better performance
 * BUSINESS:  Index recommendations guide database optimization
 * CHANGE:    Initial implementation with common index patterns
 * RISK:      Low - Index recommendations don't change existing schema
 */
func (qo *QueryOptimizer) generateIndexRecommendations() {
	// Time-based indexes for session queries
	qo.indexRecommendations["session_user_time"] = IndexRecommendation{
		TableName:     "Session",
		ColumnNames:   []string{"user_id", "start_time"},
		IndexType:     "composite",
		Priority:      1,
		EstimatedGain: "50-80% improvement for daily/weekly reports",
		QueryTypes:    []string{"daily", "weekly", "monthly"},
		CreationSQL:   "CREATE INDEX idx_session_user_time ON Session(user_id, start_time);",
	}

	// Work block project index
	qo.indexRecommendations["workblock_project_time"] = IndexRecommendation{
		TableName:     "WorkBlock",
		ColumnNames:   []string{"project_id", "start_time"},
		IndexType:     "composite",
		Priority:      2,
		EstimatedGain: "30-60% improvement for project reports",
		QueryTypes:    []string{"project", "monthly"},
		CreationSQL:   "CREATE INDEX idx_workblock_project_time ON WorkBlock(project_id, start_time);",
	}

	// Activity event timestamp index
	qo.indexRecommendations["activity_timestamp"] = IndexRecommendation{
		TableName:     "ActivityEvent",
		ColumnNames:   []string{"timestamp"},
		IndexType:     "btree",
		Priority:      3,
		EstimatedGain: "20-40% improvement for activity analysis",
		QueryTypes:    []string{"activity", "audit"},
		CreationSQL:   "CREATE INDEX idx_activity_timestamp ON ActivityEvent(timestamp);",
	}

	// Project path index for quick lookups
	qo.indexRecommendations["project_path"] = IndexRecommendation{
		TableName:     "Project",
		ColumnNames:   []string{"normalized_path"},
		IndexType:     "btree",
		Priority:      4,
		EstimatedGain: "80-95% improvement for project lookups",
		QueryTypes:    []string{"project", "lookup"},
		CreationSQL:   "CREATE INDEX idx_project_path ON Project(normalized_path);",
	}
}

/**
 * CONTEXT:   Get index recommendations prioritized by performance impact
 * INPUT:     Query type filter for relevant recommendations
 * OUTPUT:    Sorted list of index recommendations
 * BUSINESS:  Prioritized recommendations guide optimization efforts
 * CHANGE:    Initial implementation with priority-based sorting
 * RISK:      Low - Read-only recommendation retrieval
 */
func (qo *QueryOptimizer) GetIndexRecommendations(queryTypeFilter string) []IndexRecommendation {
	recommendations := make([]IndexRecommendation, 0)

	for _, rec := range qo.indexRecommendations {
		// Filter by query type if specified
		if queryTypeFilter != "" {
			found := false
			for _, qt := range rec.QueryTypes {
				if qt == queryTypeFilter {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		recommendations = append(recommendations, rec)
	}

	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(recommendations); i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[i].Priority > recommendations[j].Priority {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}

	return recommendations
}

/**
 * CONTEXT:   Apply index recommendations to improve query performance
 * INPUT:     Context for timeout control and specific recommendations to apply
 * OUTPUT:    Results of index creation with success/failure status
 * BUSINESS:  Index application provides actual performance improvements
 * CHANGE:    Initial implementation with safe index creation
 * RISK:      Medium - Index creation affects database schema and may take time
 */
func (qo *QueryOptimizer) ApplyIndexRecommendations(ctx context.Context, recommendationNames []string) (*IndexApplicationResult, error) {
	result := &IndexApplicationResult{
		AppliedIndexes:  make([]string, 0),
		FailedIndexes:   make([]string, 0),
		ExecutionTimes:  make(map[string]time.Duration),
		StartTime:       time.Now(),
	}

	for _, name := range recommendationNames {
		recommendation, exists := qo.indexRecommendations[name]
		if !exists {
			result.FailedIndexes = append(result.FailedIndexes, name)
			continue
		}

		// Apply the index
		startTime := time.Now()
		err := qo.applyIndex(ctx, recommendation)
		executionTime := time.Since(startTime)

		result.ExecutionTimes[name] = executionTime

		if err != nil {
			result.FailedIndexes = append(result.FailedIndexes, name)
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", name, err))
		} else {
			result.AppliedIndexes = append(result.AppliedIndexes, name)
		}
	}

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

/**
 * CONTEXT:   Apply single index recommendation to database
 * INPUT:     Context and index recommendation details
 * OUTPUT:    Success or error from index creation
 * BUSINESS:  Individual index creation for performance improvement
 * CHANGE:    Initial implementation with error handling
 * RISK:      Medium - Index creation may fail or take significant time
 */
func (qo *QueryOptimizer) applyIndex(ctx context.Context, recommendation IndexRecommendation) error {
	// Check if index already exists (basic check)
	checkQuery := fmt.Sprintf("SHOW INDEXES ON %s;", recommendation.TableName)
	result, err := qo.connectionManager.Query(ctx, checkQuery, nil)
	if err != nil {
		// If SHOW INDEXES fails, try to create anyway
		fmt.Printf("Warning: Could not check existing indexes: %v\n", err)
	} else {
		defer result.Close()
		// TODO: Parse result to check if index exists
		// For now, proceed with creation
	}

	// Create the index
	_, err = qo.connectionManager.Query(ctx, recommendation.CreationSQL, nil)
	if err != nil {
		return fmt.Errorf("failed to create index %s: %w", 
			strings.Join(recommendation.ColumnNames, "_"), err)
	}

	fmt.Printf("Successfully created index: %s\n", recommendation.CreationSQL)
	return nil
}

/**
 * CONTEXT:   Results of index application process
 * INPUT:     Index creation results, timing, and error information
 * OUTPUT:    Comprehensive results of index optimization
 * BUSINESS:  Application results show optimization progress and issues
 * CHANGE:    Initial result structure for index application
 * RISK:      Low - Data structure for index creation results
 */
type IndexApplicationResult struct {
	AppliedIndexes  []string                 `json:"applied_indexes"`
	FailedIndexes   []string                 `json:"failed_indexes"`
	ExecutionTimes  map[string]time.Duration `json:"execution_times"`
	Errors          []string                 `json:"errors"`
	StartTime       time.Time                `json:"start_time"`
	EndTime         time.Time                `json:"end_time"`
	TotalDuration   time.Duration            `json:"total_duration"`
}

/**
 * CONTEXT:   Analyze query execution plan for optimization opportunities
 * INPUT:     Query string and execution context
 * OUTPUT:    Execution plan analysis with optimization suggestions
 * BUSINESS:  Execution plan analysis reveals performance bottlenecks
 * CHANGE:    Initial implementation with basic plan analysis
 * RISK:      Low - Read-only execution plan analysis
 */
func (qo *QueryOptimizer) AnalyzeExecutionPlan(ctx context.Context, query string, params map[string]interface{}) (*ExecutionPlanAnalysis, error) {
	// Get execution plan (KuzuDB specific)
	explainQuery := "EXPLAIN " + query
	result, err := qo.connectionManager.Query(ctx, explainQuery, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution plan: %w", err)
	}
	defer result.Close()

	analysis := &ExecutionPlanAnalysis{
		OriginalQuery: query,
		Timestamp:     time.Now(),
	}

	// Parse execution plan
	planSteps := make([]string, 0)
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to read execution plan: %w", err)
		}
		
		// Extract plan information (implementation depends on KuzuDB format)
		if len(row) > 0 {
			if stepStr, ok := row[0].(string); ok {
				planSteps = append(planSteps, stepStr)
			}
		}
	}

	analysis.ExecutionSteps = planSteps
	
	// Analyze plan for optimization opportunities
	analysis.Recommendations = qo.analyzeExecutionSteps(planSteps)
	analysis.EstimatedCost = qo.estimateQueryCost(planSteps)
	analysis.IndexUsage = qo.detectIndexUsage(planSteps)

	return analysis, nil
}

/**
 * CONTEXT:   Execution plan analysis results with optimization insights
 * INPUT:     Query execution plan data and analysis results
 * OUTPUT:    Structured analysis with actionable recommendations
 * BUSINESS:  Plan analysis guides specific optimization actions
 * CHANGE:    Initial analysis structure for execution plans
 * RISK:      Low - Data structure for execution plan analysis
 */
type ExecutionPlanAnalysis struct {
	OriginalQuery   string    `json:"original_query"`
	ExecutionSteps  []string  `json:"execution_steps"`
	EstimatedCost   float64   `json:"estimated_cost"`
	IndexUsage      []string  `json:"index_usage"`
	Recommendations []string  `json:"recommendations"`
	Bottlenecks     []string  `json:"bottlenecks"`
	Timestamp       time.Time `json:"timestamp"`
}

/**
 * CONTEXT:   Analyze execution plan steps for optimization opportunities
 * INPUT:     Array of execution plan steps
 * OUTPUT:    Optimization recommendations based on plan analysis
 * BUSINESS:  Step analysis identifies specific performance improvements
 * CHANGE:    Initial implementation with common pattern recognition
 * RISK:      Low - Pattern analysis for optimization suggestions
 */
func (qo *QueryOptimizer) analyzeExecutionSteps(steps []string) []string {
	recommendations := make([]string, 0)

	for _, step := range steps {
		stepLower := strings.ToLower(step)
		
		// Check for table scans
		if strings.Contains(stepLower, "table scan") || strings.Contains(stepLower, "seq scan") {
			recommendations = append(recommendations, 
				"Consider adding index to avoid table scan: " + step)
		}
		
		// Check for sorts
		if strings.Contains(stepLower, "sort") {
			recommendations = append(recommendations, 
				"Consider index on sort columns to avoid sorting: " + step)
		}
		
		// Check for nested loops
		if strings.Contains(stepLower, "nested loop") {
			recommendations = append(recommendations, 
				"Consider index on join columns to improve nested loop: " + step)
		}
		
		// Check for aggregations
		if strings.Contains(stepLower, "aggregate") || strings.Contains(stepLower, "group") {
			recommendations = append(recommendations, 
				"Consider pre-computed aggregates for better performance: " + step)
		}
	}

	return recommendations
}

/**
 * CONTEXT:   Estimate query cost based on execution plan
 * INPUT:     Execution plan steps for cost calculation
 * OUTPUT:    Estimated relative cost score
 * BUSINESS:  Cost estimation helps prioritize optimization efforts
 * CHANGE:    Initial implementation with basic cost modeling
 * RISK:      Low - Cost estimation for optimization prioritization
 */
func (qo *QueryOptimizer) estimateQueryCost(steps []string) float64 {
	cost := 0.0
	
	for _, step := range steps {
		stepLower := strings.ToLower(step)
		
		// Assign costs based on operation types
		switch {
		case strings.Contains(stepLower, "table scan"):
			cost += 100.0 // High cost for table scans
		case strings.Contains(stepLower, "index scan"):
			cost += 10.0 // Low cost for index scans
		case strings.Contains(stepLower, "sort"):
			cost += 50.0 // Medium cost for sorting
		case strings.Contains(stepLower, "nested loop"):
			cost += 75.0 // High cost for nested loops
		case strings.Contains(stepLower, "hash join"):
			cost += 25.0 // Medium cost for hash joins
		case strings.Contains(stepLower, "aggregate"):
			cost += 30.0 // Medium cost for aggregation
		default:
			cost += 5.0 // Base cost for other operations
		}
	}
	
	return cost
}

/**
 * CONTEXT:   Detect index usage in execution plan
 * INPUT:     Execution plan steps for index detection
 * OUTPUT:    List of indexes used in query execution
 * BUSINESS:  Index usage detection validates optimization effectiveness
 * CHANGE:    Initial implementation with pattern matching
 * RISK:      Low - Index usage detection for optimization validation
 */
func (qo *QueryOptimizer) detectIndexUsage(steps []string) []string {
	indexUsage := make([]string, 0)
	
	for _, step := range steps {
		stepLower := strings.ToLower(step)
		
		// Look for index references in execution plan
		if strings.Contains(stepLower, "index") {
			// Extract index name if possible
			if strings.Contains(stepLower, "idx_") {
				// Simple pattern matching for index names
				parts := strings.Split(step, " ")
				for _, part := range parts {
					if strings.HasPrefix(strings.ToLower(part), "idx_") {
						indexUsage = append(indexUsage, part)
					}
				}
			} else {
				indexUsage = append(indexUsage, "Unknown index used")
			}
		}
	}
	
	return indexUsage
}

/**
 * CONTEXT:   Benchmark query performance before and after optimization
 * INPUT:     Original and optimized queries with test parameters
 * OUTPUT:    Performance comparison results
 * BUSINESS:  Benchmarking validates optimization effectiveness
 * CHANGE:    Initial implementation with performance comparison
 * RISK:      Low - Performance benchmarking for optimization validation
 */
func (qo *QueryOptimizer) BenchmarkOptimization(ctx context.Context, originalQuery, optimizedQuery string, params map[string]interface{}, iterations int) (*OptimizationBenchmark, error) {
	benchmark := &OptimizationBenchmark{
		OriginalQuery:   originalQuery,
		OptimizedQuery:  optimizedQuery,
		Iterations:      iterations,
		StartTime:       time.Now(),
	}

	// Benchmark original query
	originalTimes := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		result, err := qo.connectionManager.Query(ctx, originalQuery, params)
		duration := time.Since(start)
		
		if err != nil {
			benchmark.OriginalErrors++
		} else {
			result.Close()
			originalTimes = append(originalTimes, duration)
		}
	}

	// Benchmark optimized query
	optimizedTimes := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		result, err := qo.connectionManager.Query(ctx, optimizedQuery, params)
		duration := time.Since(start)
		
		if err != nil {
			benchmark.OptimizedErrors++
		} else {
			result.Close()
			optimizedTimes = append(optimizedTimes, duration)
		}
	}

	// Calculate statistics
	benchmark.OriginalAvg = qo.calculateAverage(originalTimes)
	benchmark.OptimizedAvg = qo.calculateAverage(optimizedTimes)
	benchmark.OriginalMin = qo.calculateMin(originalTimes)
	benchmark.OptimizedMin = qo.calculateMin(optimizedTimes)
	benchmark.OriginalMax = qo.calculateMax(originalTimes)
	benchmark.OptimizedMax = qo.calculateMax(optimizedTimes)

	// Calculate improvement
	if benchmark.OriginalAvg > 0 {
		improvement := float64(benchmark.OriginalAvg-benchmark.OptimizedAvg) / float64(benchmark.OriginalAvg) * 100.0
		benchmark.ImprovementPercent = improvement
	}

	benchmark.EndTime = time.Now()
	return benchmark, nil
}

/**
 * CONTEXT:   Optimization benchmark results with performance comparison
 * INPUT:     Benchmark data from original vs optimized query performance
 * OUTPUT:    Comprehensive performance comparison metrics
 * BUSINESS:  Benchmark results validate optimization effectiveness
 * CHANGE:    Initial benchmark result structure
 * RISK:      Low - Data structure for optimization benchmark results
 */
type OptimizationBenchmark struct {
	OriginalQuery       string        `json:"original_query"`
	OptimizedQuery      string        `json:"optimized_query"`
	Iterations          int           `json:"iterations"`
	OriginalAvg         time.Duration `json:"original_avg"`
	OptimizedAvg        time.Duration `json:"optimized_avg"`
	OriginalMin         time.Duration `json:"original_min"`
	OptimizedMin        time.Duration `json:"optimized_min"`
	OriginalMax         time.Duration `json:"original_max"`
	OptimizedMax        time.Duration `json:"optimized_max"`
	ImprovementPercent  float64       `json:"improvement_percent"`
	OriginalErrors      int           `json:"original_errors"`
	OptimizedErrors     int           `json:"optimized_errors"`
	StartTime           time.Time     `json:"start_time"`
	EndTime             time.Time     `json:"end_time"`
}

// Helper functions for benchmark calculations
func (qo *QueryOptimizer) calculateAverage(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}
	
	total := time.Duration(0)
	for _, t := range times {
		total += t
	}
	
	return total / time.Duration(len(times))
}

func (qo *QueryOptimizer) calculateMin(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}
	
	min := times[0]
	for _, t := range times[1:] {
		if t < min {
			min = t
		}
	}
	
	return min
}

func (qo *QueryOptimizer) calculateMax(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 0
	}
	
	max := times[0]
	for _, t := range times[1:] {
		if t > max {
			max = t
		}
	}
	
	return max
}

/**
 * CONTEXT:   Generate comprehensive optimization report
 * INPUT:     Time period for analysis and optimization recommendations
 * OUTPUT:    Complete optimization report with actionable recommendations
 * BUSINESS:  Optimization reports guide database performance improvements
 * CHANGE:    Initial implementation with comprehensive analysis
 * RISK:      Low - Report generation for optimization insights
 */
func (qo *QueryOptimizer) GenerateOptimizationReport(period time.Duration) *OptimizationReport {
	report := &OptimizationReport{
		Period:      period,
		GeneratedAt: time.Now(),
	}

	// Get performance statistics
	stats := qo.performanceMonitor.GetStats(period)
	report.PerformanceStats = stats

	// Get index recommendations
	report.IndexRecommendations = qo.GetIndexRecommendations("")

	// Get slow queries
	report.SlowQueries = qo.performanceMonitor.GetSlowQueries(period, 10)

	// Generate optimization summary
	report.OptimizationSummary = qo.generateOptimizationSummary(stats, report.SlowQueries)

	// Calculate optimization priority
	report.OptimizationPriority = qo.calculateOptimizationPriority(stats)

	return report
}

/**
 * CONTEXT:   Complete optimization report with analysis and recommendations
 * INPUT:     Optimization analysis data and performance metrics
 * OUTPUT:    Structured report for database optimization guidance
 * BUSINESS:  Optimization reports enable systematic performance improvements
 * CHANGE:    Initial report structure for optimization guidance
 * RISK:      Low - Data structure for optimization reporting
 */
type OptimizationReport struct {
	Period                time.Duration           `json:"period"`
	GeneratedAt           time.Time               `json:"generated_at"`
	PerformanceStats      PerformanceStats        `json:"performance_stats"`
	IndexRecommendations  []IndexRecommendation   `json:"index_recommendations"`
	SlowQueries          []QueryMetrics          `json:"slow_queries"`
	OptimizationSummary   string                  `json:"optimization_summary"`
	OptimizationPriority  string                  `json:"optimization_priority"`
	RecommendedActions    []string                `json:"recommended_actions"`
}

func (qo *QueryOptimizer) generateOptimizationSummary(stats PerformanceStats, slowQueries []QueryMetrics) string {
	if stats.TotalQueries == 0 {
		return "No queries to analyze"
	}

	summary := fmt.Sprintf("Analyzed %d queries over %v. ", stats.TotalQueries, stats.Period)
	
	if stats.AverageExecutionTime > DefaultSlowQueryThreshold {
		summary += fmt.Sprintf("Average execution time of %v exceeds recommended %v. ", 
			stats.AverageExecutionTime, DefaultSlowQueryThreshold)
	} else {
		summary += "Query performance is within acceptable limits. "
	}

	if stats.CacheHitRate < 70.0 {
		summary += fmt.Sprintf("Cache hit rate of %.1f%% could be improved. ", stats.CacheHitRate)
	}

	if len(slowQueries) > 0 {
		summary += fmt.Sprintf("Found %d slow queries requiring optimization. ", len(slowQueries))
	}

	return summary
}

func (qo *QueryOptimizer) calculateOptimizationPriority(stats PerformanceStats) string {
	if stats.AverageExecutionTime > 200*time.Millisecond || stats.ErrorRate > 10.0 {
		return "HIGH"
	} else if stats.AverageExecutionTime > 100*time.Millisecond || stats.CacheHitRate < 50.0 {
		return "MEDIUM"
	}
	return "LOW"
}