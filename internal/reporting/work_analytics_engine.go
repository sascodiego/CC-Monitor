/**
 * CONTEXT:   Work block analytics engine for deep work insights and productivity analysis
 * INPUT:     Work blocks from SQLite with activities and timing information
 * OUTPUT:    Comprehensive work analytics including focus metrics, patterns, and recommendations
 * BUSINESS:  Analytics engine provides actionable insights for productivity optimization
 * CHANGE:    Initial implementation of advanced work pattern analysis
 * RISK:      Medium - Complex analytics affecting user insights and recommendations
 */

package reporting

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

// WorkAnalyticsEngine provides advanced work pattern analysis
type WorkAnalyticsEngine struct {
	workBlockRepo *sqlite.WorkBlockRepository
	activityRepo  *sqlite.ActivityRepository
	projectRepo   *sqlite.ProjectRepository
}

// NewWorkAnalyticsEngine creates a new work analytics engine
func NewWorkAnalyticsEngine(
	workBlockRepo *sqlite.WorkBlockRepository,
	activityRepo *sqlite.ActivityRepository,
	projectRepo *sqlite.ProjectRepository,
) *WorkAnalyticsEngine {
	return &WorkAnalyticsEngine{
		workBlockRepo: workBlockRepo,
		activityRepo:  activityRepo,
		projectRepo:   projectRepo,
	}
}

/**
 * CONTEXT:   Deep work analysis for focus and productivity insights
 * INPUT:     Work blocks array requiring deep work pattern analysis
 * OUTPUT:    Deep work metrics with focus score, flow state detection, and optimization recommendations
 * BUSINESS:  Deep work analysis helps users optimize focus and productivity patterns
 * CHANGE:    Initial deep work analysis with flow state detection
 * RISK:      Low - Analytics function with pattern recognition and scoring
 */
func (wae *WorkAnalyticsEngine) AnalyzeDeepWork(ctx context.Context, workBlocks []*sqlite.WorkBlock) *DeepWorkAnalysis {
	analysis := &DeepWorkAnalysis{
		FocusBlocks:        make([]FocusBlock, 0),
		ContextSwitches:    0,
		DeepWorkTime:       time.Duration(0),
		ShallowWorkTime:    time.Duration(0),
		FragmentationScore: 0.0,
		FlowSessions:       make([]FlowSession, 0),
		Recommendations:    make([]string, 0),
	}

	if len(workBlocks) == 0 {
		return analysis
	}

	// Analyze each work block for deep work characteristics
	var totalDuration time.Duration
	lastProject := ""
	
	for _, wb := range workBlocks {
		// Calculate work block duration
		var duration time.Duration
		if wb.EndTime != nil {
			duration = wb.EndTime.Sub(wb.StartTime)
		} else {
			duration = time.Since(wb.StartTime)
		}

		totalDuration += duration

		// Get project for context switching analysis
		project, _ := wae.projectRepo.GetByID(ctx, wb.ProjectID)
		projectName := "Unknown"
		if project != nil {
			projectName = project.Name
		}

		// Track context switches
		if lastProject != "" && lastProject != projectName {
			analysis.ContextSwitches++
		}
		lastProject = projectName

		// Classify work blocks by focus level
		focusLevel := wae.classifyFocusLevel(duration, wb.ActivityCount)
		
		focusBlock := FocusBlock{
			StartTime:    wb.StartTime,
			Duration:     duration,
			ProjectName:  projectName,
			ActivityRate: float64(wb.ActivityCount) / duration.Minutes(),
			FocusLevel:   focusLevel,
			FlowState:    wae.detectFlowState(wb, duration),
		}

		// Handle nil end time
		if wb.EndTime != nil {
			focusBlock.EndTime = *wb.EndTime
		} else {
			focusBlock.EndTime = wb.StartTime.Add(duration)
		}

		analysis.FocusBlocks = append(analysis.FocusBlocks, focusBlock)

		// Categorize into deep vs shallow work
		if focusLevel >= FocusLevelDeep {
			analysis.DeepWorkTime += duration
		} else {
			analysis.ShallowWorkTime += duration
		}
	}

	// Calculate deep work percentage
	if totalDuration > 0 {
		analysis.DeepWorkPercentage = (analysis.DeepWorkTime.Hours() / totalDuration.Hours()) * 100
	}

	// Calculate focus score (0-100)
	analysis.FocusScore = wae.calculateFocusScore(analysis)

	// Calculate fragmentation score
	analysis.FragmentationScore = wae.calculateFragmentationScore(analysis.FocusBlocks)

	// Detect flow sessions (sequences of high-focus blocks)
	analysis.FlowSessions = wae.detectFlowSessions(analysis.FocusBlocks)

	// Generate recommendations
	analysis.Recommendations = wae.generateDeepWorkRecommendations(analysis)

	return analysis
}

/**
 * CONTEXT:   Activity pattern analysis for work behavior insights
 * INPUT:     Time range and user context for activity pattern analysis
 * OUTPUT:    Activity patterns with peak times, rhythm analysis, and behavioral insights
 * BUSINESS:  Activity patterns help users understand work rhythms and optimize scheduling
 * CHANGE:    Initial activity pattern analysis with behavioral insights
 * RISK:      Medium - Complex pattern analysis with statistical calculations
 */
func (wae *WorkAnalyticsEngine) AnalyzeActivityPatterns(ctx context.Context, userID string, startTime, endTime time.Time) (*ActivityPatternAnalysis, error) {
	// Get activities for the time period
	activities, err := wae.activityRepo.GetActivitiesByTimeRange(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get activities for pattern analysis: %w", err)
	}

	analysis := &ActivityPatternAnalysis{
		TotalActivities:   len(activities),
		HourlyDistribution: make([]HourlyActivityData, 24),
		PeakHours:         make([]int, 0),
		ActivityTypes:     make(map[string]int),
		WorkRhythm:        "unknown",
		ConsistencyScore:  0.0,
		Recommendations:   make([]string, 0),
	}

	if len(activities) == 0 {
		return analysis, nil
	}

	// Initialize hourly distribution
	for hour := 0; hour < 24; hour++ {
		analysis.HourlyDistribution[hour] = HourlyActivityData{
			Hour:        hour,
			Activities:  0,
			Intensity:   0.0,
			Consistency: 0.0,
		}
	}

	// Analyze activity distribution
	hourCounts := make(map[int]int)
	dayActivityCounts := make(map[string]int) // Track daily activity for consistency
	
	for _, activity := range activities {
		hour := activity.Timestamp.Hour()
		hourCounts[hour]++
		
		// Count by activity type
		analysis.ActivityTypes[activity.ActivityType]++
		
		// Track daily consistency
		dayKey := activity.Timestamp.Format("2006-01-02")
		dayActivityCounts[dayKey]++
	}

	// Calculate hourly metrics
	maxHourlyActivities := 0
	_ = 0.0 // totalHourlyVariance for future variance calculations
	
	for hour, count := range hourCounts {
		analysis.HourlyDistribution[hour].Activities = count
		analysis.HourlyDistribution[hour].Intensity = float64(count) / float64(analysis.TotalActivities)
		
		if count > maxHourlyActivities {
			maxHourlyActivities = count
		}
	}

	// Identify peak hours (top 25% of activity)
	threshold := float64(maxHourlyActivities) * 0.75
	for hour, count := range hourCounts {
		if float64(count) >= threshold {
			analysis.PeakHours = append(analysis.PeakHours, hour)
		}
	}
	
	sort.Ints(analysis.PeakHours)

	// Determine work rhythm based on peak hours
	analysis.WorkRhythm = wae.determineWorkRhythm(analysis.PeakHours)

	// Calculate consistency score
	analysis.ConsistencyScore = wae.calculateConsistencyScore(dayActivityCounts)

	// Generate activity pattern recommendations
	analysis.Recommendations = wae.generateActivityPatternRecommendations(analysis)

	return analysis, nil
}

/**
 * CONTEXT:   Project focus analysis for context switching and concentration insights
 * INPUT:     Work blocks grouped by projects for focus analysis
 * OUTPUT:    Project focus metrics with switching costs and concentration recommendations
 * BUSINESS:  Project focus analysis helps users minimize context switching costs
 * CHANGE:    Initial project focus analysis with switching cost calculation
 * RISK:      Low - Focus analysis with project-based metrics
 */
func (wae *WorkAnalyticsEngine) AnalyzeProjectFocus(ctx context.Context, workBlocks []*sqlite.WorkBlock) *ProjectFocusAnalysis {
	analysis := &ProjectFocusAnalysis{
		ProjectSessions:   make([]ProjectSession, 0),
		ContextSwitches:   0,
		SwitchingCost:     time.Duration(0),
		FocusEfficiency:   0.0,
		Recommendations:   make([]string, 0),
	}

	if len(workBlocks) == 0 {
		return analysis
	}

	// Group work blocks by project and time continuity
	projectSessions := wae.groupWorkBlocksByProject(ctx, workBlocks)
	
	// Analyze each project session
	var totalWorkTime time.Duration
	var totalSwitchingTime time.Duration
	lastEndTime := time.Time{}
	
	for _, session := range projectSessions {
		totalWorkTime += session.Duration
		
		// Calculate switching cost (time gaps between project sessions)
		if !lastEndTime.IsZero() && session.StartTime.Sub(lastEndTime) < 30*time.Minute {
			switchGap := session.StartTime.Sub(lastEndTime)
			totalSwitchingTime += switchGap
			analysis.ContextSwitches++
		}
		
		lastEndTime = session.EndTime
		analysis.ProjectSessions = append(analysis.ProjectSessions, session)
	}

	analysis.SwitchingCost = totalSwitchingTime

	// Calculate focus efficiency (work time vs total time including switching)
	totalTimeSpent := totalWorkTime + totalSwitchingTime
	if totalTimeSpent > 0 {
		analysis.FocusEfficiency = (totalWorkTime.Hours() / totalTimeSpent.Hours()) * 100
	}

	// Generate project focus recommendations
	analysis.Recommendations = wae.generateProjectFocusRecommendations(analysis)

	return analysis
}

// Helper methods for work block classification and analysis

/**
 * CONTEXT:   Classify work block focus level based on duration and activity patterns
 * INPUT:     Work block duration and activity count for focus level calculation
 * OUTPUT:    Focus level classification (distracted, focused, deep, or flow)
 * BUSINESS:  Focus level classification drives productivity insights and recommendations
 * CHANGE:    Initial focus level classification with activity-based scoring
 * RISK:      Low - Classification logic with business rule validation
 */
func (wae *WorkAnalyticsEngine) classifyFocusLevel(duration time.Duration, activityCount int64) FocusLevel {
	minutes := duration.Minutes()
	activityRate := float64(activityCount) / minutes

	// Classification based on duration and activity rate
	switch {
	case minutes < 5:
		return FocusLevelDistracted
	case minutes < 15:
		return FocusLevelFocused
	case minutes >= 25 && activityRate > 0.5: // Pomodoro+ with good activity
		return FocusLevelDeep
	case minutes >= 45 && activityRate > 0.3: // Extended focus with sustained activity
		return FocusLevelFlow
	case minutes >= 25:
		return FocusLevelDeep
	default:
		return FocusLevelFocused
	}
}

/**
 * CONTEXT:   Detect flow state indicators in work blocks
 * INPUT:     Work block entity and duration for flow state analysis
 * OUTPUT:    Boolean indicating potential flow state based on duration and activity patterns
 * BUSINESS:  Flow state detection helps users identify optimal work conditions
 * CHANGE:    Initial flow state detection with duration and activity criteria
 * RISK:      Low - Pattern detection with statistical analysis
 */
func (wae *WorkAnalyticsEngine) detectFlowState(workBlock *sqlite.WorkBlock, duration time.Duration) bool {
	// Flow state indicators:
	// 1. Duration >= 45 minutes (extended focus)
	// 2. Consistent activity rate (0.2-1.0 activities per minute)
	// 3. Minimal state changes (not frequently idle/active)
	
	minutes := duration.Minutes()
	if minutes < 45 {
		return false
	}
	
	activityRate := float64(workBlock.ActivityCount) / minutes
	return activityRate >= 0.2 && activityRate <= 1.0
}

/**
 * CONTEXT:   Calculate overall focus score from deep work analysis
 * INPUT:     Deep work analysis with focus blocks and work patterns
 * OUTPUT:    Focus score from 0-100 based on deep work percentage and consistency
 * BUSINESS:  Focus score provides single metric for work focus quality
 * CHANGE:    Initial focus score calculation with weighted metrics
 * RISK:      Low - Scoring algorithm with business rule validation
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
 * CHANGE:    Initial fragmentation calculation with block size variance
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
 * CONTEXT:   Detect flow sessions from sequences of high-focus work blocks
 * INPUT:     Array of focus blocks for flow session pattern detection
 * OUTPUT:    Array of flow sessions with duration and quality metrics
 * BUSINESS:  Flow sessions represent optimal work periods for productivity insights
 * CHANGE:    Initial flow session detection with continuity analysis
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

/**
 * CONTEXT:   Group work blocks by project continuity for context switching analysis
 * INPUT:     Array of work blocks requiring project grouping
 * OUTPUT:    Array of project sessions representing continuous work on same project
 * BUSINESS:  Project sessions help analyze context switching costs and focus efficiency
 * CHANGE:    Initial project grouping with time continuity validation
 * RISK:      Low - Data grouping with project identification
 */
func (wae *WorkAnalyticsEngine) groupWorkBlocksByProject(ctx context.Context, workBlocks []*sqlite.WorkBlock) []ProjectSession {
	if len(workBlocks) == 0 {
		return []ProjectSession{}
	}
	
	// Sort blocks by start time
	sortedBlocks := make([]*sqlite.WorkBlock, len(workBlocks))
	copy(sortedBlocks, workBlocks)
	sort.Slice(sortedBlocks, func(i, j int) bool {
		return sortedBlocks[i].StartTime.Before(sortedBlocks[j].StartTime)
	})
	
	sessions := make([]ProjectSession, 0)
	currentSession := ProjectSession{
		ProjectID:   sortedBlocks[0].ProjectID,
		StartTime:   sortedBlocks[0].StartTime,
		WorkBlocks:  []*sqlite.WorkBlock{sortedBlocks[0]},
	}
	
	// Get project name
	if project, err := wae.projectRepo.GetByID(ctx, sortedBlocks[0].ProjectID); err == nil && project != nil {
		currentSession.ProjectName = project.Name
	}
	
	for i := 1; i < len(sortedBlocks); i++ {
		block := sortedBlocks[i]
		lastBlock := currentSession.WorkBlocks[len(currentSession.WorkBlocks)-1]
		
		// Check if same project and continuous (gap < 30 minutes)
		var gap time.Duration
		if lastBlock.EndTime != nil {
			gap = block.StartTime.Sub(*lastBlock.EndTime)
		} else {
			gap = block.StartTime.Sub(lastBlock.LastActivityTime)
		}
		
		if block.ProjectID == currentSession.ProjectID && gap < 30*time.Minute {
			// Continue current session
			currentSession.WorkBlocks = append(currentSession.WorkBlocks, block)
		} else {
			// Finalize current session
			wae.finalizeProjectSession(&currentSession)
			sessions = append(sessions, currentSession)
			
			// Start new session
			currentSession = ProjectSession{
				ProjectID:  block.ProjectID,
				StartTime:  block.StartTime,
				WorkBlocks: []*sqlite.WorkBlock{block},
			}
			
			// Get project name
			if project, err := wae.projectRepo.GetByID(ctx, block.ProjectID); err == nil && project != nil {
				currentSession.ProjectName = project.Name
			}
		}
	}
	
	// Finalize last session
	wae.finalizeProjectSession(&currentSession)
	sessions = append(sessions, currentSession)
	
	return sessions
}

// Helper methods for analysis calculations

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

func (wae *WorkAnalyticsEngine) determineWorkRhythm(peakHours []int) string {
	if len(peakHours) == 0 {
		return "irregular"
	}
	
	// Analyze peak hour distribution
	earlyHours := 0  // 6-10 AM
	midHours := 0    // 10 AM - 2 PM
	lateHours := 0   // 2-6 PM
	eveningHours := 0 // 6-10 PM
	
	for _, hour := range peakHours {
		switch {
		case hour >= 6 && hour < 10:
			earlyHours++
		case hour >= 10 && hour < 14:
			midHours++
		case hour >= 14 && hour < 18:
			lateHours++
		case hour >= 18 && hour < 22:
			eveningHours++
		}
	}
	
	// Determine predominant rhythm
	max := math.Max(math.Max(float64(earlyHours), float64(midHours)), 
					math.Max(float64(lateHours), float64(eveningHours)))
	
	switch max {
	case float64(earlyHours):
		return "early_bird"
	case float64(midHours):
		return "mid_day"
	case float64(lateHours):
		return "afternoon"
	case float64(eveningHours):
		return "night_owl"
	default:
		return "distributed"
	}
}

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

// Recommendation generators

func (wae *WorkAnalyticsEngine) generateDeepWorkRecommendations(analysis *DeepWorkAnalysis) []string {
	recommendations := make([]string, 0)
	
	if analysis.DeepWorkPercentage < 30 {
		recommendations = append(recommendations, "🎯 Focus on 25+ minute work blocks for deep work")
		recommendations = append(recommendations, "📱 Consider reducing distractions during work sessions")
	}
	
	if analysis.ContextSwitches > 8 {
		recommendations = append(recommendations, "🔄 High context switching detected - try time blocking by project")
	}
	
	if analysis.FragmentationScore > 0.5 {
		recommendations = append(recommendations, "🧩 Work is highly fragmented - consider longer uninterrupted sessions")
	}
	
	if len(analysis.FlowSessions) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("✨ Great! %d flow sessions detected - replicate these conditions", len(analysis.FlowSessions)))
	}
	
	return recommendations
}

func (wae *WorkAnalyticsEngine) generateActivityPatternRecommendations(analysis *ActivityPatternAnalysis) []string {
	recommendations := make([]string, 0)
	
	switch analysis.WorkRhythm {
	case "early_bird":
		recommendations = append(recommendations, "🌅 Your peak is morning - schedule important work before 10 AM")
	case "night_owl":
		recommendations = append(recommendations, "🦉 Evening productivity detected - protect your late-day focus time")
	case "distributed":
		recommendations = append(recommendations, "⚖️ Distributed work pattern - good for varied task scheduling")
	}
	
	if analysis.ConsistencyScore < 50 {
		recommendations = append(recommendations, "📅 Inconsistent work pattern - try establishing regular work hours")
	}
	
	if len(analysis.PeakHours) > 6 {
		recommendations = append(recommendations, "⏰ Work scattered across many hours - consider focusing on 3-4 peak hours")
	}
	
	return recommendations
}

func (wae *WorkAnalyticsEngine) generateProjectFocusRecommendations(analysis *ProjectFocusAnalysis) []string {
	recommendations := make([]string, 0)
	
	if analysis.ContextSwitches > 5 {
		recommendations = append(recommendations, "🔄 High project switching - consider batching similar work")
	}
	
	if analysis.FocusEfficiency < 70 {
		recommendations = append(recommendations, "⚡ Focus efficiency below 70% - reduce interruptions between projects")
	}
	
	if analysis.SwitchingCost > 30*time.Minute {
		recommendations = append(recommendations, "⏱️ High switching cost detected - minimize project transitions")
	}
	
	return recommendations
}

// Data structures for analytics results

type DeepWorkAnalysis struct {
	DeepWorkTime       time.Duration  `json:"deep_work_time"`
	ShallowWorkTime    time.Duration  `json:"shallow_work_time"`
	DeepWorkPercentage float64        `json:"deep_work_percentage"`
	FocusScore         float64        `json:"focus_score"`
	ContextSwitches    int            `json:"context_switches"`
	FragmentationScore float64        `json:"fragmentation_score"`
	FocusBlocks        []FocusBlock   `json:"focus_blocks"`
	FlowSessions       []FlowSession  `json:"flow_sessions"`
	Recommendations    []string       `json:"recommendations"`
}

type FocusBlock struct {
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	ProjectName  string        `json:"project_name"`
	ActivityRate float64       `json:"activity_rate"`
	FocusLevel   FocusLevel    `json:"focus_level"`
	FlowState    bool          `json:"flow_state"`
}

type FlowSession struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Quality   float64       `json:"quality"`
	Blocks    []FocusBlock  `json:"blocks"`
}

type FocusLevel int

const (
	FocusLevelDistracted FocusLevel = iota
	FocusLevelFocused
	FocusLevelDeep
	FocusLevelFlow
)

func (fl FocusLevel) String() string {
	switch fl {
	case FocusLevelDistracted:
		return "distracted"
	case FocusLevelFocused:
		return "focused"
	case FocusLevelDeep:
		return "deep"
	case FocusLevelFlow:
		return "flow"
	default:
		return "unknown"
	}
}

type ActivityPatternAnalysis struct {
	TotalActivities    int                   `json:"total_activities"`
	HourlyDistribution []HourlyActivityData  `json:"hourly_distribution"`
	PeakHours          []int                 `json:"peak_hours"`
	ActivityTypes      map[string]int        `json:"activity_types"`
	WorkRhythm         string                `json:"work_rhythm"`
	ConsistencyScore   float64               `json:"consistency_score"`
	Recommendations    []string              `json:"recommendations"`
}

type HourlyActivityData struct {
	Hour        int     `json:"hour"`
	Activities  int     `json:"activities"`
	Intensity   float64 `json:"intensity"`
	Consistency float64 `json:"consistency"`
}

type ProjectFocusAnalysis struct {
	ProjectSessions []ProjectSession `json:"project_sessions"`
	ContextSwitches int              `json:"context_switches"`
	SwitchingCost   time.Duration    `json:"switching_cost"`
	FocusEfficiency float64          `json:"focus_efficiency"`
	Recommendations []string         `json:"recommendations"`
}

type ProjectSession struct {
	ProjectID   string                `json:"project_id"`
	ProjectName string                `json:"project_name"`
	StartTime   time.Time             `json:"start_time"`
	EndTime     time.Time             `json:"end_time"`
	Duration    time.Duration         `json:"duration"`
	WorkBlocks  []*sqlite.WorkBlock   `json:"work_blocks"`
}