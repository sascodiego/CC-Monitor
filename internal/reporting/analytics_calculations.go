/**
 * CONTEXT:   Mathematical calculations and scoring functions for work analytics
 * INPUT:     Analysis data structures requiring mathematical computation and scoring
 * OUTPUT:    Calculated scores, statistical metrics, and mathematical analysis results
 * BUSINESS:  Provides quantitative foundation for productivity insights and recommendations
 * CHANGE:    Extracted calculation logic from monolithic work_analytics_engine.go
 * RISK:      Low - Pure mathematical functions with clear input/output contracts
 */

package reporting

import (
	"math"
	"sort"
	"time"
)

/**
 * CONTEXT:   Calculate overall focus score from deep work analysis
 * INPUT:     Deep work analysis with focus blocks and work patterns
 * OUTPUT:    Focus score from 0-100 based on deep work percentage and consistency
 * BUSINESS:  Focus score provides single metric for work focus quality
 * CHANGE:    Extracted from core engine, maintains calculation logic
 * RISK:      Low - Scoring algorithm with established business rule validation
 */
func (wae *WorkAnalyticsEngine) calculateFocusScore(analysis *DeepWorkAnalysis) float64 {
	score := 0.0
	
	// Deep work percentage (0-50 points)
	score += analysis.DeepWorkPercentage * 0.5
	
	// Flow sessions bonus (0-20 points)
	flowBonus := math.Min(float64(len(analysis.FlowSessions))*5, 20)
	score += flowBonus
	
	// Context switching penalty (0-20 points deduction)
	contextPenalty := math.Min(float64(analysis.ContextSwitches)*2, 20)
	score -= contextPenalty
	
	// Fragmentation penalty (0-10 points deduction)
	fragmentationPenalty := analysis.FragmentationScore * 10
	score -= fragmentationPenalty
	
	// Ensure score is within 0-100 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

/**
 * CONTEXT:   Calculate work fragmentation score from focus blocks
 * INPUT:     Array of focus blocks for fragmentation analysis
 * OUTPUT:    Fragmentation score from 0-1 indicating work interruption level
 * BUSINESS:  Fragmentation score identifies work interruption patterns
 * CHANGE:    Extracted from core engine, maintains statistical calculation
 * RISK:      Low - Statistical calculation with variance analysis
 */
func (wae *WorkAnalyticsEngine) calculateFragmentationScore(focusBlocks []FocusBlock) float64 {
	if len(focusBlocks) < 2 {
		return 0.0
	}
	
	// Calculate variance in block durations
	var totalDuration time.Duration
	durations := make([]float64, len(focusBlocks))
	
	for i, block := range focusBlocks {
		durations[i] = block.Duration.Minutes()
		totalDuration += block.Duration
	}
	
	mean := totalDuration.Minutes() / float64(len(focusBlocks))
	variance := 0.0
	
	for _, duration := range durations {
		diff := duration - mean
		variance += diff * diff
	}
	variance /= float64(len(focusBlocks))
	
	// Normalize fragmentation score (higher variance = more fragmentation)
	// Cap at 1.0 for extremely fragmented work
	fragmentationScore := math.Sqrt(variance) / (mean + 1) // +1 to avoid division by zero
	if fragmentationScore > 1.0 {
		fragmentationScore = 1.0
	}
	
	return fragmentationScore
}

/**
 * CONTEXT:   Calculate flow session quality based on focus block composition
 * INPUT:     Array of focus blocks within a flow session
 * OUTPUT:    Quality score from 0-1 indicating flow session effectiveness
 * BUSINESS:  Flow quality helps identify most productive work periods
 * CHANGE:    Extracted from core engine, maintains quality calculation
 * RISK:      Low - Quality scoring with weighted focus level assessment
 */
func (wae *WorkAnalyticsEngine) calculateFlowQuality(blocks []FocusBlock) float64 {
	if len(blocks) == 0 {
		return 0.0
	}
	
	qualitySum := 0.0
	for _, block := range blocks {
		switch block.FocusLevel {
		case FocusLevelFlow:
			qualitySum += 1.0
		case FocusLevelDeep:
			qualitySum += 0.8
		case FocusLevelFocused:
			qualitySum += 0.6
		case FocusLevelDistracted:
			qualitySum += 0.3
		}
	}
	
	return qualitySum / float64(len(blocks))
}

/**
 * CONTEXT:   Calculate work consistency score from daily activity distribution
 * INPUT:     Map of daily activity counts for consistency analysis
 * OUTPUT:    Consistency score from 0-100 indicating work pattern regularity
 * BUSINESS:  Consistency score measures work habit stability and routine quality
 * CHANGE:    Extracted from core engine, maintains coefficient of variation calculation
 * RISK:      Low - Statistical consistency measurement with variance analysis
 */
func (wae *WorkAnalyticsEngine) calculateConsistencyScore(dayActivityCounts map[string]int) float64 {
	if len(dayActivityCounts) == 0 {
		return 0.0
	}
	
	// Calculate coefficient of variation (CV) as consistency measure
	values := make([]float64, 0, len(dayActivityCounts))
	sum := 0.0
	
	for _, count := range dayActivityCounts {
		val := float64(count)
		values = append(values, val)
		sum += val
	}
	
	if sum == 0 {
		return 0.0
	}
	
	mean := sum / float64(len(values))
	variance := 0.0
	
	for _, val := range values {
		diff := val - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	
	if mean == 0 {
		return 0.0
	}
	
	cv := math.Sqrt(variance) / mean
	
	// Convert to consistency score (lower CV = higher consistency)
	// Scale from 0-100 where 100 is perfect consistency
	consistencyScore := math.Max(0, 100-cv*50) // Adjust scaling as needed
	
	return consistencyScore
}

/**
 * CONTEXT:   Finalize flow session with calculated metrics and timing
 * INPUT:     Flow session pointer requiring duration and quality calculation
 * OUTPUT:    Updated session with end time, duration, and quality metrics
 * BUSINESS:  Session finalization ensures accurate flow state measurement
 * CHANGE:    Extracted from core engine, maintains session calculation logic
 * RISK:      Low - Session metric calculation with time and quality assessment
 */
func (wae *WorkAnalyticsEngine) finalizeFlowSession(session *FlowSession) {
	if len(session.Blocks) == 0 {
		return
	}
	
	// Calculate session end time and duration
	var totalDuration time.Duration
	endTime := session.StartTime
	
	for _, block := range session.Blocks {
		totalDuration += block.Duration
		if block.EndTime.After(endTime) {
			endTime = block.EndTime
		}
	}
	
	session.EndTime = endTime
	session.Duration = totalDuration
	session.Quality = wae.calculateFlowQuality(session.Blocks)
}

/**
 * CONTEXT:   Finalize project session with calculated timing and duration metrics
 * INPUT:     Project session pointer requiring duration calculation from work blocks
 * OUTPUT:    Updated session with end time and total duration metrics
 * BUSINESS:  Project session finalization enables accurate focus efficiency calculation
 * CHANGE:    Extracted from core engine, maintains project session timing logic
 * RISK:      Low - Session timing calculation with work block duration aggregation
 */
func (wae *WorkAnalyticsEngine) finalizeProjectSession(session *ProjectSession) {
	if len(session.WorkBlocks) == 0 {
		return
	}
	
	// Calculate session end time and duration
	var totalDuration time.Duration
	endTime := session.StartTime
	
	for _, block := range session.WorkBlocks {
		var blockDuration time.Duration
		if block.EndTime != nil {
			blockDuration = block.EndTime.Sub(block.StartTime)
			if block.EndTime.After(endTime) {
				endTime = *block.EndTime
			}
		} else {
			blockDuration = time.Since(block.StartTime)
			if block.LastActivityTime.After(endTime) {
				endTime = block.LastActivityTime
			}
		}
		totalDuration += blockDuration
	}
	
	session.EndTime = endTime
	session.Duration = totalDuration
}

/**
 * CONTEXT:   Check if flow session qualifies based on duration and block count
 * INPUT:     Flow session with blocks requiring qualification assessment
 * OUTPUT:    Boolean indicating if session meets flow state criteria
 * BUSINESS:  Flow session qualification ensures only meaningful flow periods are identified
 * CHANGE:    Extracted from core engine, maintains qualification criteria
 * RISK:      Low - Qualification logic with established duration and continuity rules
 */
func (wae *WorkAnalyticsEngine) qualifiesAsFlowSession(session FlowSession) bool {
	if len(session.Blocks) < 2 {
		return false
	}
	
	totalDuration := time.Duration(0)
	for _, block := range session.Blocks {
		totalDuration += block.Duration
	}
	
	return totalDuration >= 45*time.Minute
}

/**
 * CONTEXT:   Detect flow sessions from sequences of high-focus work blocks
 * INPUT:     Array of focus blocks for flow session pattern detection
 * OUTPUT:    Array of flow sessions with duration and quality metrics
 * BUSINESS:  Flow sessions represent optimal work periods for productivity insights
 * CHANGE:    Extracted from core engine, maintains flow detection algorithm
 * RISK:      Low - Pattern detection with time continuity validation
 */
func (wae *WorkAnalyticsEngine) detectFlowSessions(focusBlocks []FocusBlock) []FlowSession {
	if len(focusBlocks) == 0 {
		return []FlowSession{}
	}
	
	// Sort blocks by start time
	sortedBlocks := make([]FocusBlock, len(focusBlocks))
	copy(sortedBlocks, focusBlocks)
	sort.Slice(sortedBlocks, func(i, j int) bool {
		return sortedBlocks[i].StartTime.Before(sortedBlocks[j].StartTime)
	})
	
	flowSessions := make([]FlowSession, 0)
	currentSession := FlowSession{
		StartTime: sortedBlocks[0].StartTime,
		Blocks:    []FocusBlock{sortedBlocks[0]},
	}
	
	for i := 1; i < len(sortedBlocks); i++ {
		block := sortedBlocks[i]
		lastBlock := currentSession.Blocks[len(currentSession.Blocks)-1]
		
		// Check if blocks are continuous (gap < 15 minutes) and both are deep/flow level
		gap := block.StartTime.Sub(lastBlock.EndTime)
		if gap < 15*time.Minute && 
		   lastBlock.FocusLevel >= FocusLevelDeep && 
		   block.FocusLevel >= FocusLevelDeep {
			// Continue current flow session
			currentSession.Blocks = append(currentSession.Blocks, block)
		} else {
			// End current session if it qualifies (2+ blocks, 45+ minutes)
			if wae.qualifiesAsFlowSession(currentSession) {
				wae.finalizeFlowSession(&currentSession)
				flowSessions = append(flowSessions, currentSession)
			}
			
			// Start new session
			currentSession = FlowSession{
				StartTime: block.StartTime,
				Blocks:    []FocusBlock{block},
			}
		}
	}
	
	// Check final session
	if wae.qualifiesAsFlowSession(currentSession) {
		wae.finalizeFlowSession(&currentSession)
		flowSessions = append(flowSessions, currentSession)
	}
	
	return flowSessions
}