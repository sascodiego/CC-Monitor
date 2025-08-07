/**
 * CONTEXT:   ProcessingTimeEstimator for intelligent Claude processing time estimation
 * INPUT:     Prompt content, historical data, and complexity hints for analysis
 * OUTPUT:    Estimated processing time based on prompt characteristics and patterns
 * BUSINESS:  Prevent false idle detection by accurately predicting Claude processing duration
 * CHANGE:    Initial implementation with prompt analysis and historical learning
 * RISK:      Medium - Estimation accuracy affects work block timing, uses conservative defaults
 */

package entities

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// ComplexityLevel represents different levels of Claude task complexity
type ComplexityLevel string

const (
	ComplexityLevelSimple    ComplexityLevel = "simple"    // Quick questions, explanations
	ComplexityLevelModerate  ComplexityLevel = "moderate"  // Analysis, code review
	ComplexityLevelComplex   ComplexityLevel = "complex"   // Code generation, refactoring
	ComplexityLevelExtensive ComplexityLevel = "extensive" // Large system design, multi-file changes
)

// ProcessingTimeEstimator provides intelligent time estimation for Claude processing
type ProcessingTimeEstimator struct {
	mu                sync.RWMutex
	historicalData    map[ComplexityLevel][]time.Duration // Historical processing times by complexity
	complexityRules   map[ComplexityLevel]time.Duration   // Base time estimates per complexity level
	keywordPatterns   map[ComplexityLevel][]*regexp.Regexp // Regex patterns for complexity detection
	promptLengthBase  time.Duration                       // Base time per prompt length unit
	contextSizeBase   time.Duration                       // Additional time for large context
}

// EstimationRequest holds parameters for processing time estimation
type EstimationRequest struct {
	Prompt          string            // The user prompt to analyze
	PromptLength    int               // Character count of prompt
	ContextSize     int               // Number of files/tokens in context
	ProjectType     string            // Type of project (e.g., "go", "python", "web")
	UserMetadata    map[string]string // Additional user-provided context
	PreviousPrompts []string          // Recent prompts for pattern analysis
}

/**
 * CONTEXT:   Factory for creating new ProcessingTimeEstimator with sensible defaults
 * INPUT:     No parameters, creates estimator with built-in rules and patterns
 * OUTPUT:    Configured ProcessingTimeEstimator ready for use
 * BUSINESS:  Provide conservative estimates to prevent false idle detection
 * CHANGE:    Initial implementation with keyword patterns and base estimates
 * RISK:      Low - Conservative defaults ensure adequate timeout buffers
 */
func NewProcessingTimeEstimator() *ProcessingTimeEstimator {
	estimator := &ProcessingTimeEstimator{
		historicalData:   make(map[ComplexityLevel][]time.Duration),
		complexityRules:  make(map[ComplexityLevel]time.Duration),
		keywordPatterns:  make(map[ComplexityLevel][]*regexp.Regexp),
		promptLengthBase: 200 * time.Millisecond, // ~200ms per 1000 characters
		contextSizeBase:  100 * time.Millisecond, // ~100ms per file in context
	}

	// Set base time estimates per complexity level
	estimator.complexityRules[ComplexityLevelSimple] = 15 * time.Second    // Quick responses
	estimator.complexityRules[ComplexityLevelModerate] = 45 * time.Second  // Analysis tasks
	estimator.complexityRules[ComplexityLevelComplex] = 2 * time.Minute    // Code generation
	estimator.complexityRules[ComplexityLevelExtensive] = 5 * time.Minute  // Large refactoring

	// Initialize keyword patterns for complexity detection
	estimator.initializeKeywordPatterns()

	return estimator
}

/**
 * CONTEXT:   Initialize keyword patterns for automatic complexity detection
 * INPUT:     No parameters, configures internal pattern matching rules
 * OUTPUT:    No return value, sets up internal keyword pattern matching
 * BUSINESS:  Automatically classify prompts by complexity for accurate time estimation
 * CHANGE:    Initial pattern rules based on common Claude Code usage patterns
 * RISK:      Low - Pattern matching provides hints, with fallback to moderate complexity
 */
func (e *ProcessingTimeEstimator) initializeKeywordPatterns() {
	// Simple tasks - quick responses
	simplePatterns := []string{
		`(?i)\b(what|how|why|explain|describe|define)\b.*\?`,
		`(?i)\b(help|assistance|guide|tutorial)\b`,
		`(?i)\b(status|check|verify|validate)\b`,
		`(?i)\b(simple|quick|briefly)\b`,
	}
	
	// Moderate tasks - analysis and review
	moderatePatterns := []string{
		`(?i)\b(analyze|review|examine|investigate|debug)\b`,
		`(?i)\b(optimize|improve|enhance|refactor)\b.*\bfile\b`,
		`(?i)\b(compare|contrast|evaluate)\b`,
		`(?i)\b(test|testing|unit test|integration)\b`,
		`(?i)\b(read|parse|understand)\b.*\bcode\b`,
	}
	
	// Complex tasks - code generation and significant changes
	complexPatterns := []string{
		`(?i)\b(write|create|implement|build|generate)\b.*\bcode\b`,
		`(?i)\b(function|class|method|component|module)\b`,
		`(?i)\b(design|architecture|structure|framework)\b`,
		`(?i)\b(add|insert|modify|update)\b.*\b(feature|functionality)\b`,
		`(?i)\b(fix|solve|resolve)\b.*\bbug\b`,
		`(?i)\bmultiple files?\b`,
	}
	
	// Extensive tasks - large-scale changes
	extensivePatterns := []string{
		`(?i)\b(entire|whole|complete|full)\b.*(system|application|project|codebase)\b`,
		`(?i)\b(migrate|convert|transform|restructure)\b`,
		`(?i)\b(new\s+)(project|application|system|service)\b`,
		`(?i)\b(integration|api|database|backend|frontend)\b.*\b(implementation|development)\b`,
		`(?i)\bmany files?\b`,
		`(?i)\bcomplex\s+(logic|algorithm|system)\b`,
	}

	// Compile patterns for each complexity level
	e.keywordPatterns[ComplexityLevelSimple] = e.compilePatterns(simplePatterns)
	e.keywordPatterns[ComplexityLevelModerate] = e.compilePatterns(moderatePatterns)
	e.keywordPatterns[ComplexityLevelComplex] = e.compilePatterns(complexPatterns)
	e.keywordPatterns[ComplexityLevelExtensive] = e.compilePatterns(extensivePatterns)
}

func (e *ProcessingTimeEstimator) compilePatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, regex)
		}
	}
	return compiled
}

/**
 * CONTEXT:   Main estimation method that analyzes prompts and returns predicted processing time
 * INPUT:     EstimationRequest with prompt content and context information
 * OUTPUT:    Estimated processing time duration with confidence level
 * BUSINESS:  Core estimation logic that prevents false idle detection during Claude processing
 * CHANGE:    Initial implementation with multi-factor analysis
 * RISK:      Medium - Estimation accuracy directly affects work tracking precision
 */
func (e *ProcessingTimeEstimator) EstimateProcessingTime(request EstimationRequest) time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Detect complexity level from prompt content
	complexity := e.detectComplexity(request.Prompt)
	
	// Get base time for detected complexity
	baseTime := e.complexityRules[complexity]
	
	// Adjust for prompt length
	lengthAdjustment := time.Duration(request.PromptLength/1000) * e.promptLengthBase
	
	// Adjust for context size (number of files being worked with)
	contextAdjustment := time.Duration(request.ContextSize) * e.contextSizeBase
	
	// Apply historical data if available
	historicalAdjustment := e.calculateHistoricalAdjustment(complexity)
	
	// Calculate total estimated time
	totalTime := baseTime + lengthAdjustment + contextAdjustment + historicalAdjustment
	
	// Apply project-specific adjustments
	projectAdjustment := e.calculateProjectAdjustment(request.ProjectType, complexity)
	totalTime += projectAdjustment
	
	// Add buffer for safety (15% buffer to prevent false timeouts)
	safetyBuffer := time.Duration(float64(totalTime) * 0.15)
	totalTime += safetyBuffer
	
	// Ensure minimum and maximum bounds
	minTime := 10 * time.Second  // Minimum processing time
	maxTime := 15 * time.Minute  // Maximum reasonable processing time
	
	if totalTime < minTime {
		totalTime = minTime
	}
	if totalTime > maxTime {
		totalTime = maxTime
	}
	
	return totalTime
}

/**
 * CONTEXT:   Analyze prompt content to determine complexity level using keyword patterns
 * INPUT:     Prompt string to analyze for complexity indicators
 * OUTPUT:    ComplexityLevel enum representing detected task complexity
 * BUSINESS:  Complexity detection drives accurate time estimation for different task types
 * CHANGE:    Initial implementation with pattern matching and scoring
 * RISK:      Low - Pattern matching provides hints with reasonable fallback
 */
func (e *ProcessingTimeEstimator) detectComplexity(prompt string) ComplexityLevel {
	// Score each complexity level based on pattern matches
	scores := make(map[ComplexityLevel]int)
	
	for level, patterns := range e.keywordPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(prompt) {
				scores[level]++
			}
		}
	}
	
	// Find the complexity level with highest score
	maxScore := 0
	detectedLevel := ComplexityLevelModerate // Default fallback
	
	// Check in order from most complex to least complex (prioritize higher complexity)
	complexityOrder := []ComplexityLevel{
		ComplexityLevelExtensive,
		ComplexityLevelComplex,
		ComplexityLevelModerate,
		ComplexityLevelSimple,
	}
	
	for _, level := range complexityOrder {
		if score := scores[level]; score > maxScore {
			maxScore = score
			detectedLevel = level
		}
	}
	
	// Additional heuristics based on prompt characteristics
	promptWords := strings.Fields(strings.ToLower(prompt))
	wordCount := len(promptWords)
	
	// Very short prompts are likely simple
	if wordCount < 10 && detectedLevel > ComplexityLevelSimple {
		detectedLevel = ComplexityLevelSimple
	}
	
	// Very long prompts suggest complexity
	if wordCount > 100 && detectedLevel < ComplexityLevelComplex {
		detectedLevel = ComplexityLevelComplex
	}
	
	return detectedLevel
}

/**
 * CONTEXT:   Calculate adjustment based on historical processing time data
 * INPUT:     ComplexityLevel to look up historical performance data
 * OUTPUT:    Time duration adjustment based on past processing times
 * BUSINESS:  Learn from actual processing times to improve estimation accuracy
 * CHANGE:    Initial implementation with simple averaging of historical data
 * RISK:      Low - Historical data improves accuracy over time, no negative impact
 */
func (e *ProcessingTimeEstimator) calculateHistoricalAdjustment(complexity ComplexityLevel) time.Duration {
	history := e.historicalData[complexity]
	if len(history) < 3 {
		return 0 // Not enough data for adjustment
	}
	
	// Calculate average of recent processing times
	var total time.Duration
	recentCount := 5 // Use last 5 data points
	if len(history) < recentCount {
		recentCount = len(history)
	}
	
	for i := len(history) - recentCount; i < len(history); i++ {
		total += history[i]
	}
	
	averageHistorical := total / time.Duration(recentCount)
	baseTime := e.complexityRules[complexity]
	
	// Calculate adjustment (positive or negative)
	adjustment := averageHistorical - baseTime
	
	// Limit adjustment to ±50% of base time
	maxAdjustment := time.Duration(float64(baseTime) * 0.5)
	if adjustment > maxAdjustment {
		adjustment = maxAdjustment
	} else if adjustment < -maxAdjustment {
		adjustment = -maxAdjustment
	}
	
	return adjustment
}

/**
 * CONTEXT:   Calculate processing time adjustments based on project type
 * INPUT:     Project type string and complexity level for context-aware adjustment
 * OUTPUT:    Time duration adjustment for specific project characteristics
 * BUSINESS:  Different project types have different processing characteristics
 * CHANGE:    Initial implementation with common project type patterns
 * RISK:      Low - Project-specific adjustments improve accuracy without major impact
 */
func (e *ProcessingTimeEstimator) calculateProjectAdjustment(projectType string, complexity ComplexityLevel) time.Duration {
	if projectType == "" {
		return 0
	}
	
	projectType = strings.ToLower(projectType)
	baseTime := e.complexityRules[complexity]
	
	// Project-specific multipliers
	switch {
	case strings.Contains(projectType, "rust"):
		// Rust code generation tends to take longer due to complexity
		return time.Duration(float64(baseTime) * 0.2)
	case strings.Contains(projectType, "cpp") || strings.Contains(projectType, "c++"):
		// C++ also tends to be more complex
		return time.Duration(float64(baseTime) * 0.15)
	case strings.Contains(projectType, "javascript") || strings.Contains(projectType, "typescript"):
		// Web development often involves multiple files
		return time.Duration(float64(baseTime) * 0.1)
	case strings.Contains(projectType, "python"):
		// Python tends to be slightly faster
		return time.Duration(float64(baseTime) * -0.05)
	case strings.Contains(projectType, "go"):
		// Go is generally straightforward
		return time.Duration(float64(baseTime) * -0.1)
	default:
		return 0
	}
}

/**
 * CONTEXT:   Record actual processing time for learning and improving estimates
 * INPUT:     Complexity level and actual processing time duration
 * OUTPUT:    No return value, updates internal historical data
 * BUSINESS:  Learn from actual processing times to improve future estimates
 * CHANGE:    Initial implementation with simple historical data storage
 * RISK:      Low - Historical data collection improves system over time
 */
func (e *ProcessingTimeEstimator) RecordActualTime(complexity ComplexityLevel, actualTime time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if e.historicalData[complexity] == nil {
		e.historicalData[complexity] = make([]time.Duration, 0)
	}
	
	// Add new data point
	e.historicalData[complexity] = append(e.historicalData[complexity], actualTime)
	
	// Keep only recent data (last 50 data points per complexity level)
	maxHistorySize := 50
	if len(e.historicalData[complexity]) > maxHistorySize {
		// Remove oldest data points
		e.historicalData[complexity] = e.historicalData[complexity][len(e.historicalData[complexity])-maxHistorySize:]
	}
}

/**
 * CONTEXT:   Get estimation statistics for monitoring and debugging
 * INPUT:     No parameters, analyzes current estimator state
 * OUTPUT:    EstimatorStats with accuracy metrics and data points
 * BUSINESS:  Provide insights into estimator performance for system monitoring
 * CHANGE:    Initial implementation with basic statistics
 * RISK:      Low - Read-only statistics for monitoring and improvement
 */
type EstimatorStats struct {
	ComplexityDistribution map[ComplexityLevel]int     `json:"complexity_distribution"`
	AverageEstimates       map[ComplexityLevel]float64 `json:"average_estimates"`
	HistoricalDataPoints   map[ComplexityLevel]int     `json:"historical_data_points"`
	TotalEstimations       int                         `json:"total_estimations"`
}

func (e *ProcessingTimeEstimator) GetStats() EstimatorStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	stats := EstimatorStats{
		ComplexityDistribution: make(map[ComplexityLevel]int),
		AverageEstimates:       make(map[ComplexityLevel]float64),
		HistoricalDataPoints:   make(map[ComplexityLevel]int),
	}
	
	totalEstimations := 0
	for level, history := range e.historicalData {
		count := len(history)
		stats.HistoricalDataPoints[level] = count
		totalEstimations += count
		
		if count > 0 {
			var total time.Duration
			for _, duration := range history {
				total += duration
			}
			stats.AverageEstimates[level] = total.Seconds() / float64(count)
		}
	}
	
	stats.TotalEstimations = totalEstimations
	return stats
}

/**
 * CONTEXT:   Create Claude processing context for activity events
 * INPUT:     Prompt content and request parameters for context creation
 * OUTPUT:    ClaudeProcessingContext ready for use in activity events
 * BUSINESS:  Standardized context creation for consistent processing time tracking
 * CHANGE:    Initial implementation with prompt analysis and ID generation
 * RISK:      Low - Context creation utility with no side effects
 */
func (e *ProcessingTimeEstimator) CreateProcessingContext(prompt string, promptID string) *ClaudeProcessingContext {
	estimatedTime := e.EstimateProcessingTime(EstimationRequest{
		Prompt:       prompt,
		PromptLength: len(prompt),
		ContextSize:  1, // Default context size
	})
	
	complexity := e.detectComplexity(prompt)
	complexityHint := string(complexity)
	
	return &ClaudeProcessingContext{
		PromptID:       promptID,
		EstimatedTime:  estimatedTime,
		PromptLength:   len(prompt),
		ComplexityHint: complexityHint,
		ClaudeActivity: ClaudeActivityStart,
	}
}