/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-001
 * CONTEXT:   Work hour analyzer implementing real-time analysis of activity data into business metrics
 * REASON:    Need to transform enhanced daemon activity monitoring into comprehensive work hour analytics
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate all time calculations, handle timezone consistency, implement efficient caching
 * RISK:      High - Work hour calculations must be accurate for billing and compliance requirements
 */
package workhour

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-002
 * CONTEXT:   WorkHourAnalyzer implements comprehensive work hour analysis with pattern recognition
 * REASON:    Core business logic engine for transforming monitoring data into actionable work hour insights
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement proper concurrency controls and validate all calculations against business rules
 * RISK:      High - Analytics accuracy directly affects reporting and billing systems
 */
type WorkHourAnalyzer struct {
	dbManager        arch.WorkHourDatabaseManager
	logger           arch.Logger
	analysisCache    *AnalysisCache
	mu               sync.RWMutex
	analysisConfig   AnalysisConfig
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-003
 * CONTEXT:   AnalysisCache provides optimized caching for expensive calculations
 * REASON:    Pattern analysis and trend calculations can be expensive, caching improves performance
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement TTL expiration and cache size limits, monitor cache hit rates
 * RISK:      Medium - Cache misses affect performance but not functionality
 */
type AnalysisCache struct {
	workDays        map[string]*domain.WorkDay      // Date-keyed work day cache
	workWeeks       map[string]*domain.WorkWeek     // Week-keyed work week cache
	patterns        map[string]*domain.WorkPattern  // Pattern analysis cache
	summaries       map[string]*domain.ActivitySummary // Activity summary cache
	lastUpdated     map[string]time.Time            // Cache entry timestamps
	mu              sync.RWMutex
	maxSize         int
	ttl             time.Duration
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-004
 * CONTEXT:   AnalysisConfig defines configuration for work hour analysis behavior
 * REASON:    Need configurable thresholds and rules for different business requirements
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate configuration values and provide sensible defaults
 * RISK:      Low - Configuration errors are typically caught at initialization
 */
type AnalysisConfig struct {
	WorkDayStartHour   int           `json:"workDayStartHour"`   // Default work day start (8 AM)
	WorkDayEndHour     int           `json:"workDayEndHour"`     // Default work day end (6 PM)
	StandardWorkHours  time.Duration `json:"standardWorkHours"`  // Standard daily work hours
	MinWorkBlockDuration time.Duration `json:"minWorkBlockDuration"` // Minimum meaningful work block
	BreakThreshold     time.Duration `json:"breakThreshold"`     // Minimum gap to consider a break
	ProductivityWindow time.Duration `json:"productivityWindow"` // Window for productivity calculations
	PatternLookback    int           `json:"patternLookback"`    // Days to analyze for patterns
}

func NewWorkHourAnalyzer(dbManager arch.WorkHourDatabaseManager, logger arch.Logger) *WorkHourAnalyzer {
	return &WorkHourAnalyzer{
		dbManager: dbManager,
		logger:    logger,
		analysisCache: &AnalysisCache{
			workDays:    make(map[string]*domain.WorkDay),
			workWeeks:   make(map[string]*domain.WorkWeek),
			patterns:    make(map[string]*domain.WorkPattern),
			summaries:   make(map[string]*domain.ActivitySummary),
			lastUpdated: make(map[string]time.Time),
			maxSize:     500,
			ttl:         30 * time.Minute,
		},
		analysisConfig: AnalysisConfig{
			WorkDayStartHour:   8,
			WorkDayEndHour:     18,
			StandardWorkHours:  8 * time.Hour,
			MinWorkBlockDuration: 5 * time.Minute,
			BreakThreshold:     15 * time.Minute,
			ProductivityWindow: 1 * time.Hour,
			PatternLookback:    30,
		},
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-005
 * CONTEXT:   AnalyzeWorkDay implements real-time work day analysis from enhanced monitoring data
 * REASON:    Core business requirement for accurate daily work hour tracking and analysis
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle edge cases like work spanning midnight, validate time zones consistently
 * RISK:      High - Daily analysis forms the foundation for all higher-level reporting
 */
func (wha *WorkHourAnalyzer) AnalyzeWorkDay(date time.Time) (*domain.WorkDay, error) {
	dateKey := date.Format("2006-01-02")
	
	// Check cache first
	wha.analysisCache.mu.RLock()
	if cached, exists := wha.analysisCache.workDays[dateKey]; exists {
		if lastUpdate, ok := wha.analysisCache.lastUpdated[dateKey]; ok {
			if time.Since(lastUpdate) < wha.analysisCache.ttl {
				wha.analysisCache.mu.RUnlock()
				wha.logger.Debug("Work day analysis cache hit", "date", dateKey)
				return cached, nil
			}
		}
	}
	wha.analysisCache.mu.RUnlock()

	// Get from database (may trigger calculation)
	workDay, err := wha.dbManager.GetWorkDayData(date)
	if err != nil {
		return nil, fmt.Errorf("failed to get work day data: %w", err)
	}

	// Enhance with real-time analysis
	enhancedWorkDay, err := wha.enhanceWorkDayAnalysis(workDay, date)
	if err != nil {
		wha.logger.Warn("Failed to enhance work day analysis", "date", dateKey, "error", err)
		enhancedWorkDay = workDay
	}

	// Update cache
	wha.updateWorkDayCache(dateKey, enhancedWorkDay)

	wha.logger.Info("Work day analysis completed", 
		"date", dateKey, 
		"totalTime", enhancedWorkDay.TotalTime,
		"sessionCount", enhancedWorkDay.SessionCount,
		"blockCount", enhancedWorkDay.BlockCount)

	return enhancedWorkDay, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-006
 * CONTEXT:   enhanceWorkDayAnalysis adds sophisticated analysis to basic work day data
 * REASON:    Need to add efficiency metrics, pattern detection, and quality analysis beyond raw time tracking
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure analysis algorithms handle edge cases and incomplete data gracefully
 * RISK:      Medium - Analysis enhancements should not fail core tracking functionality
 */
func (wha *WorkHourAnalyzer) enhanceWorkDayAnalysis(workDay *domain.WorkDay, date time.Time) (*domain.WorkDay, error) {
	// Create enhanced copy
	enhanced := *workDay
	
	// Analyze efficiency and quality metrics
	if enhanced.TotalTime > 0 {
		workSpan := enhanced.GetWorkDuration()
		if workSpan > 0 {
			// Calculate more sophisticated efficiency metrics
			efficiencyRatio := float64(enhanced.TotalTime) / float64(workSpan)
			
			// Adjust for reasonable work day patterns
			if workSpan > 12*time.Hour {
				// Very long spans indicate possible system issues or overnight sessions
				wha.logger.Debug("Unusually long work span detected", 
					"date", date.Format("2006-01-02"), 
					"span", workSpan,
					"activeTime", enhanced.TotalTime)
			}
			
			// Analyze break patterns for quality insights
			if enhanced.BreakTime > 0 && enhanced.BlockCount > 1 {
				avgBreakTime := enhanced.BreakTime / time.Duration(enhanced.BlockCount-1)
				if avgBreakTime < wha.analysisConfig.BreakThreshold {
					wha.logger.Debug("Short breaks detected - high focus session", 
						"date", date.Format("2006-01-02"),
						"avgBreak", avgBreakTime)
				}
			}
		}
	}

	return &enhanced, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-007
 * CONTEXT:   AnalyzeWorkWeek implements weekly aggregation and overtime analysis
 * REASON:    Business requirement for weekly work hour reporting and overtime calculation
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure week boundaries are correctly calculated across DST changes and locales
 * RISK:      High - Overtime calculations must be accurate for compliance and billing
 */
func (wha *WorkHourAnalyzer) AnalyzeWorkWeek(date time.Time) (*domain.WorkWeek, error) {
	// Normalize to Monday
	weekStart := date
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}
	
	weekKey := weekStart.Format("2006-01-02")
	
	// Check cache
	wha.analysisCache.mu.RLock()
	if cached, exists := wha.analysisCache.workWeeks[weekKey]; exists {
		if lastUpdate, ok := wha.analysisCache.lastUpdated["week_"+weekKey]; ok {
			if time.Since(lastUpdate) < wha.analysisCache.ttl {
				wha.analysisCache.mu.RUnlock()
				wha.logger.Debug("Work week analysis cache hit", "weekStart", weekKey)
				return cached, nil
			}
		}
	}
	wha.analysisCache.mu.RUnlock()

	// Get from database
	workWeek, err := wha.dbManager.GetWorkWeekData(weekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to get work week data: %w", err)
	}

	// Enhance with analysis
	enhancedWorkWeek, err := wha.enhanceWorkWeekAnalysis(workWeek)
	if err != nil {
		wha.logger.Warn("Failed to enhance work week analysis", "weekStart", weekKey, "error", err)
		enhancedWorkWeek = workWeek
	}

	// Update cache
	wha.updateWorkWeekCache(weekKey, enhancedWorkWeek)

	wha.logger.Info("Work week analysis completed",
		"weekStart", weekKey,
		"totalTime", enhancedWorkWeek.TotalTime,
		"overtimeHours", enhancedWorkWeek.OvertimeHours,
		"workDays", len(enhancedWorkWeek.WorkDays))

	return enhancedWorkWeek, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-008
 * CONTEXT:   enhanceWorkWeekAnalysis adds weekly-specific analytics and insights
 * REASON:    Need weekly patterns, overtime analysis, and work-life balance insights
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle partial weeks correctly and validate overtime calculations
 * RISK:      Medium - Week-level analysis affects weekly reporting and overtime tracking
 */
func (wha *WorkHourAnalyzer) enhanceWorkWeekAnalysis(workWeek *domain.WorkWeek) (*domain.WorkWeek, error) {
	enhanced := *workWeek
	
	// Analyze weekly patterns
	if len(enhanced.WorkDays) > 0 {
		// Calculate work day consistency
		if len(enhanced.WorkDays) > 1 {
			workTimes := make([]float64, 0, len(enhanced.WorkDays))
			for _, day := range enhanced.WorkDays {
				if day.TotalTime > 0 {
					workTimes = append(workTimes, float64(day.TotalTime))
				}
			}
			
			if len(workTimes) > 1 {
				consistency := calculateConsistencyScore(workTimes)
				wha.logger.Debug("Weekly work consistency calculated",
					"weekStart", enhanced.WeekStart.Format("2006-01-02"),
					"consistency", consistency,
					"workDays", len(workTimes))
			}
		}
		
		// Identify peak work days
		var peakDay *domain.WorkDay
		var maxTime time.Duration
		for i := range enhanced.WorkDays {
			if enhanced.WorkDays[i].TotalTime > maxTime {
				maxTime = enhanced.WorkDays[i].TotalTime
				peakDay = &enhanced.WorkDays[i]
			}
		}
		
		if peakDay != nil {
			wha.logger.Debug("Peak work day identified",
				"weekStart", enhanced.WeekStart.Format("2006-01-02"),
				"peakDay", peakDay.Date.Format("2006-01-02"),
				"peakTime", maxTime)
		}
	}

	return &enhanced, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-009
 * CONTEXT:   AnalyzeWorkPattern implements sophisticated pattern recognition and insights
 * REASON:    Business requirement for work pattern analysis to optimize productivity and identify trends
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure pattern analysis doesn't expose sensitive personal data inappropriately
 * RISK:      Low - Pattern analysis is statistical and privacy-preserving
 */
func (wha *WorkHourAnalyzer) AnalyzeWorkPattern(startDate, endDate time.Time) (*domain.WorkPattern, error) {
	patternKey := fmt.Sprintf("%s_%s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	// Check cache
	wha.analysisCache.mu.RLock()
	if cached, exists := wha.analysisCache.patterns[patternKey]; exists {
		if lastUpdate, ok := wha.analysisCache.lastUpdated["pattern_"+patternKey]; ok {
			if time.Since(lastUpdate) < wha.analysisCache.ttl {
				wha.analysisCache.mu.RUnlock()
				return cached, nil
			}
		}
	}
	wha.analysisCache.mu.RUnlock()

	// Get pattern data points
	patternData, err := wha.dbManager.GetWorkPatternData(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern data: %w", err)
	}

	// Analyze patterns
	pattern := wha.analyzeActivityPatterns(patternData, startDate, endDate)
	
	// Update cache
	wha.updatePatternCache(patternKey, pattern)

	wha.logger.Info("Work pattern analysis completed",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"dataPoints", len(patternData),
		"workDayType", pattern.WorkDayType,
		"peakHours", pattern.PeakHours)

	return pattern, nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-010
 * CONTEXT:   analyzeActivityPatterns implements core pattern recognition algorithms
 * REASON:    Need sophisticated analysis to identify productivity patterns and optimization opportunities
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle edge cases with insufficient data, validate statistical calculations
 * RISK:      Medium - Pattern analysis affects insights and recommendations
 */
func (wha *WorkHourAnalyzer) analyzeActivityPatterns(patternData []arch.WorkPatternDataPoint, startDate, endDate time.Time) *domain.WorkPattern {
	// Initialize pattern structure
	pattern := &domain.WorkPattern{
		PeakHours:        []int{},
		ProductivityCurve: make([]float64, 24),
		WorkDayType:      domain.StandardDay,
		ConsistencyScore: 0.0,
		BreakPatterns:    []domain.BreakPattern{},
		WeeklyPattern:    make([]time.Duration, 7),
	}

	if len(patternData) == 0 {
		return pattern
	}

	// Analyze hourly activity distribution
	hourlyActivity := make([]float64, 24)
	hourlyCount := make([]int, 24)
	
	for _, dataPoint := range patternData {
		hour := dataPoint.Timestamp.Hour()
		hourlyActivity[hour] += dataPoint.Productivity
		hourlyCount[hour]++
	}

	// Calculate average productivity per hour
	var maxProductivity float64
	var peakHours []int
	
	for hour := 0; hour < 24; hour++ {
		if hourlyCount[hour] > 0 {
			pattern.ProductivityCurve[hour] = hourlyActivity[hour] / float64(hourlyCount[hour])
			
			// Identify peak hours (top 25% of productivity)
			if pattern.ProductivityCurve[hour] > maxProductivity {
				maxProductivity = pattern.ProductivityCurve[hour]
			}
		}
	}

	// Find peak hours (80% of max productivity or higher)
	threshold := maxProductivity * 0.8
	for hour, productivity := range pattern.ProductivityCurve {
		if productivity >= threshold && productivity > 0 {
			peakHours = append(peakHours, hour)
		}
	}
	pattern.PeakHours = peakHours

	// Determine work day type based on peak hours
	pattern.WorkDayType = wha.classifyWorkDayType(peakHours)

	// Calculate consistency score
	if len(patternData) > 1 {
		pattern.ConsistencyScore = wha.calculatePatternConsistency(patternData)
	}

	// Analyze weekly patterns
	weeklyDurations := make([][]time.Duration, 7)
	for _, dataPoint := range patternData {
		weekday := int(dataPoint.Timestamp.Weekday())
		if weekday == 0 { // Sunday = 0, convert to Monday = 0
			weekday = 6
		} else {
			weekday--
		}
		weeklyDurations[weekday] = append(weeklyDurations[weekday], dataPoint.Duration)
	}

	// Calculate average duration per weekday
	for day := 0; day < 7; day++ {
		if len(weeklyDurations[day]) > 0 {
			var total time.Duration
			for _, duration := range weeklyDurations[day] {
				total += duration
			}
			pattern.WeeklyPattern[day] = total / time.Duration(len(weeklyDurations[day]))
		}
	}

	return pattern
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-011
 * CONTEXT:   Helper methods for pattern classification and analysis
 * REASON:    Need supporting algorithms for pattern recognition and statistical analysis
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate statistical calculations and handle edge cases
 * RISK:      Low - Helper methods support main analysis functions
 */
func (wha *WorkHourAnalyzer) classifyWorkDayType(peakHours []int) domain.WorkDayType {
	if len(peakHours) == 0 {
		return domain.StandardDay
	}

	earlyCount := 0  // Before 10 AM
	standardCount := 0 // 10 AM - 4 PM  
	lateCount := 0   // After 4 PM

	for _, hour := range peakHours {
		switch {
		case hour < 10:
			earlyCount++
		case hour >= 10 && hour < 16:
			standardCount++
		case hour >= 16:
			lateCount++
		}
	}

	// Determine dominant pattern
	maxCount := max(earlyCount, standardCount, lateCount)
	
	switch maxCount {
	case earlyCount:
		if earlyCount > standardCount && earlyCount > lateCount {
			return domain.EarlyBird
		}
	case lateCount:
		if lateCount > standardCount && lateCount > earlyCount {
			return domain.NightOwl
		}
	case standardCount:
		return domain.StandardDay
	}

	// Multiple peaks indicate flexible schedule
	if len(peakHours) > 6 {
		return domain.FlexibleDay
	}

	return domain.StandardDay
}

func (wha *WorkHourAnalyzer) calculatePatternConsistency(patternData []arch.WorkPatternDataPoint) float64 {
	if len(patternData) < 2 {
		return 0.0
	}

	// Calculate coefficient of variation for productivity scores
	var sum, sumSquares float64
	count := 0

	for _, dataPoint := range patternData {
		if dataPoint.Productivity > 0 {
			sum += dataPoint.Productivity
			sumSquares += dataPoint.Productivity * dataPoint.Productivity
			count++
		}
	}

	if count < 2 {
		return 0.0
	}

	mean := sum / float64(count)
	variance := (sumSquares - sum*sum/float64(count)) / float64(count-1)
	
	if mean == 0 {
		return 0.0
	}
	
	stdDev := math.Sqrt(variance)
	coefficientOfVariation := stdDev / mean
	
	// Convert to consistency score (1 = perfect consistency, 0 = no consistency)
	consistencyScore := math.Max(0, 1.0-coefficientOfVariation)
	
	return consistencyScore
}

func calculateConsistencyScore(values []float64) float64 {
	if len(values) < 2 {
		return 1.0
	}

	// Calculate coefficient of variation
	var sum float64
	for _, value := range values {
		sum += value
	}
	mean := sum / float64(len(values))

	var variance float64
	for _, value := range values {
		diff := value - mean
		variance += diff * diff
	}
	variance /= float64(len(values) - 1)

	if mean == 0 {
		return 0.0
	}

	stdDev := math.Sqrt(variance)
	coefficientOfVariation := stdDev / mean
	
	// Convert to consistency score (1 = perfect consistency)
	return math.Max(0, 1.0-coefficientOfVariation)
}

func max(a, b, c int) int {
	if a >= b && a >= c {
		return a
	}
	if b >= c {
		return b
	}
	return c
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-012
 * CONTEXT:   Cache management methods for analysis optimization
 * REASON:    Analysis calculations can be expensive, proper caching improves system responsiveness
 * CHANGE:    Initial implementation.
 * PREVENTION:Implement cache size limits and TTL expiration to prevent memory issues
 * RISK:      Low - Cache management failures only affect performance, not functionality
 */
func (wha *WorkHourAnalyzer) updateWorkDayCache(dateKey string, workDay *domain.WorkDay) {
	wha.analysisCache.mu.Lock()
	defer wha.analysisCache.mu.Unlock()
	
	wha.analysisCache.workDays[dateKey] = workDay
	wha.analysisCache.lastUpdated[dateKey] = time.Now()
	
	// Simple LRU eviction if cache is full
	if len(wha.analysisCache.workDays) > wha.analysisCache.maxSize {
		wha.evictOldestEntries()
	}
}

func (wha *WorkHourAnalyzer) updateWorkWeekCache(weekKey string, workWeek *domain.WorkWeek) {
	wha.analysisCache.mu.Lock()
	defer wha.analysisCache.mu.Unlock()
	
	cacheKey := "week_" + weekKey
	wha.analysisCache.workWeeks[weekKey] = workWeek
	wha.analysisCache.lastUpdated[cacheKey] = time.Now()
}

func (wha *WorkHourAnalyzer) updatePatternCache(patternKey string, pattern *domain.WorkPattern) {
	wha.analysisCache.mu.Lock()
	defer wha.analysisCache.mu.Unlock()
	
	cacheKey := "pattern_" + patternKey
	wha.analysisCache.patterns[patternKey] = pattern
	wha.analysisCache.lastUpdated[cacheKey] = time.Now()
}

func (wha *WorkHourAnalyzer) evictOldestEntries() {
	// Remove 10% of entries when cache is full
	toRemove := max(1, wha.analysisCache.maxSize/10, 1)
	
	// Find oldest entries
	type cacheEntry struct {
		key  string
		time time.Time
	}
	
	var entries []cacheEntry
	for key, updateTime := range wha.analysisCache.lastUpdated {
		entries = append(entries, cacheEntry{key, updateTime})
	}
	
	// Sort by time (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].time.Before(entries[j].time)
	})
	
	// Remove oldest entries
	for i := 0; i < toRemove && i < len(entries); i++ {
		key := entries[i].key
		delete(wha.analysisCache.lastUpdated, key)
		
		// Remove from appropriate cache
		if key[:5] == "week_" {
			delete(wha.analysisCache.workWeeks, key[5:])
		} else if key[:8] == "pattern_" {
			delete(wha.analysisCache.patterns, key[8:])
		} else {
			delete(wha.analysisCache.workDays, key)
		}
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-WH-ANALYZER-013
 * CONTEXT:   Additional interface methods for comprehensive analysis capabilities
 * REASON:    Complete implementation of WorkHourAnalyzer interface for full reporting support
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure all methods handle edge cases and validate input parameters
 * RISK:      Medium - Interface completeness affects reporting and analytics capabilities
 */
func (wha *WorkHourAnalyzer) GenerateActivitySummary(period string, startDate, endDate time.Time) (*domain.ActivitySummary, error) {
	summaryKey := fmt.Sprintf("%s_%s_%s", period, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	// Check cache
	wha.analysisCache.mu.RLock()
	if cached, exists := wha.analysisCache.summaries[summaryKey]; exists {
		if lastUpdate, ok := wha.analysisCache.lastUpdated["summary_"+summaryKey]; ok {
			if time.Since(lastUpdate) < wha.analysisCache.ttl {
				wha.analysisCache.mu.RUnlock()
				return cached, nil
			}
		}
	}
	wha.analysisCache.mu.RUnlock()

	summary := domain.NewActivitySummary(period, startDate, endDate)
	
	// Calculate summary metrics (simplified implementation)
	days := int(endDate.Sub(startDate).Hours()/24) + 1
	
	var totalWorkTime time.Duration
	var totalSessions, totalWorkBlocks int
	
	// Iterate through each day in the period
	for d := 0; d < days; d++ {
		currentDate := startDate.AddDate(0, 0, d)
		if currentDate.After(endDate) {
			break
		}
		
		workDay, err := wha.AnalyzeWorkDay(currentDate)
		if err != nil {
			wha.logger.Warn("Failed to analyze work day for summary", "date", currentDate, "error", err)
			continue
		}
		
		totalWorkTime += workDay.TotalTime
		totalSessions += workDay.SessionCount
		totalWorkBlocks += workDay.BlockCount
	}
	
	summary.TotalWorkTime = totalWorkTime
	summary.TotalSessions = totalSessions
	summary.TotalWorkBlocks = totalWorkBlocks
	summary.DailyAverage = totalWorkTime / time.Duration(days)
	
	if totalSessions > 0 {
		summary.AverageSession = totalWorkTime / time.Duration(totalSessions)
	}
	if totalWorkBlocks > 0 {
		summary.AverageWorkBlock = totalWorkTime / time.Duration(totalWorkBlocks)
	}

	// Update cache
	wha.analysisCache.mu.Lock()
	wha.analysisCache.summaries[summaryKey] = summary
	wha.analysisCache.lastUpdated["summary_"+summaryKey] = time.Now()
	wha.analysisCache.mu.Unlock()

	wha.logger.Info("Activity summary generated",
		"period", period,
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"totalWorkTime", totalWorkTime,
		"totalSessions", totalSessions)

	return summary, nil
}

func (wha *WorkHourAnalyzer) CalculateProductivityMetrics(startDate, endDate time.Time) (*domain.EfficiencyMetrics, error) {
	// Get productivity metrics from database
	metrics, err := wha.dbManager.GetProductivityMetrics(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get productivity metrics: %w", err)
	}

	wha.logger.Info("Productivity metrics calculated",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"activeRatio", metrics.ActiveRatio,
		"focusScore", metrics.FocusScore)

	return metrics, nil
}

func (wha *WorkHourAnalyzer) GetWorkDayTrends(startDate, endDate time.Time) (*domain.TrendAnalysis, error) {
	// Get trend data from database
	trendData, err := wha.dbManager.GetWorkTimeTrends(startDate, endDate, arch.GranularityDay)
	if err != nil {
		return nil, fmt.Errorf("failed to get trend data: %w", err)
	}

	// Analyze trends
	trends := &domain.TrendAnalysis{
		TrendDirection: "stable",
	}

	if len(trendData) > 1 {
		firstValue := trendData[0].Value
		lastValue := trendData[len(trendData)-1].Value
		
		if lastValue > firstValue {
			trends.WorkTimeChange = ((lastValue - firstValue) / firstValue) * 100
			trends.TrendDirection = "up"
		} else if lastValue < firstValue {
			trends.WorkTimeChange = ((firstValue - lastValue) / firstValue) * -100
			trends.TrendDirection = "down"
		}
	}

	wha.logger.Info("Work day trends analyzed",
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"trendDirection", trends.TrendDirection,
		"workTimeChange", trends.WorkTimeChange)

	return trends, nil
}

// Ensure WorkHourAnalyzer implements WorkHourAnalyzer interface
var _ arch.WorkHourAnalyzer = (*WorkHourAnalyzer)(nil)