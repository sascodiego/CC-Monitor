/**
 * CONTEXT:   Core analytics engine with main analysis functions and struct definition
 * INPUT:     Work blocks, activities, and time ranges for comprehensive analysis
 * OUTPUT:    Complete analytics results with deep work, patterns, and focus insights
 * BUSINESS:  Primary interface for work analytics providing productivity insights
 * CHANGE:    Split from monolithic work_analytics_engine.go following SRP
 * RISK:      Low - Core structure with clear separation of concerns
 */

package reporting

import (
	"context"
	"fmt"
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
 * CHANGE:    Extracted from monolithic file, maintains all original functionality
 * RISK:      Low - Core analysis function with established business logic
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
 * CHANGE:    Extracted from monolithic file, maintains all original functionality
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
 * CHANGE:    Extracted from monolithic file, maintains all original functionality
 * RISK:      Low - Focus analysis with established project-based metrics
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