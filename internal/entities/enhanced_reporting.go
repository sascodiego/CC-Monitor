/**
 * CONTEXT:   Enhanced reporting with Claude processing time separation and analytics
 * INPUT:     Work blocks, sessions, and activity data with Claude processing context
 * OUTPUT:    Detailed reports showing user time vs Claude time vs total schedule time
 * BUSINESS:  Provide accurate productivity insights by separating different types of work time
 * CHANGE:    Enhanced reporting system with Claude processing time breakdown
 * RISK:      Medium - Reporting accuracy affects user productivity insights and system value
 */

package entities

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// EnhancedWorkMetrics provides detailed breakdown of work time including Claude processing
type EnhancedWorkMetrics struct {
	// Time breakdown
	UserInteractionTime  time.Duration `json:"user_interaction_time"`   // Time user was actively working
	ClaudeProcessingTime time.Duration `json:"claude_processing_time"`  // Time Claude was processing
	IdleTime            time.Duration `json:"idle_time"`               // Time user was actually idle
	TotalScheduleTime   time.Duration `json:"total_schedule_time"`     // Total time from first to last activity
	
	// Calculated metrics
	EfficiencyPercent    float64       `json:"efficiency_percent"`     // (User + Claude) / Total Schedule * 100
	UserActivityPercent  float64       `json:"user_activity_percent"`  // User Time / Total Schedule * 100
	ClaudeActivityPercent float64      `json:"claude_activity_percent"` // Claude Time / Total Schedule * 100
	IdlePercent         float64       `json:"idle_percent"`            // Idle Time / Total Schedule * 100
	
	// Session info
	WorkStartTime       time.Time     `json:"work_start_time"`
	WorkEndTime         time.Time     `json:"work_end_time"`
	
	// Activity counts
	TotalSessions       int           `json:"total_sessions"`
	TotalWorkBlocks     int           `json:"total_work_blocks"`
	ProcessingBlocks    int           `json:"processing_blocks"`      // Blocks that had Claude processing
	AverageBlockSize    time.Duration `json:"average_block_size"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	
	// Project distribution
	ProjectCount        int                        `json:"project_count"`
	ProjectBreakdown    []EnhancedProjectMetrics   `json:"project_breakdown"`
	
	// Processing insights
	ProcessingEstimatorStats EstimatorStats        `json:"processing_estimator_stats"`
}

// EnhancedProjectMetrics provides detailed project-level time breakdown
type EnhancedProjectMetrics struct {
	ProjectName           string        `json:"project_name"`
	UserInteractionTime   time.Duration `json:"user_interaction_time"`
	ClaudeProcessingTime  time.Duration `json:"claude_processing_time"`
	TotalTime            time.Duration `json:"total_time"`
	
	// Percentages of total work time
	UserTimePercent      float64       `json:"user_time_percent"`
	ClaudeTimePercent    float64       `json:"claude_time_percent"`
	TotalTimePercent     float64       `json:"total_time_percent"`
	
	// Activity metrics
	WorkBlocks           int           `json:"work_blocks"`
	ProcessingBlocks     int           `json:"processing_blocks"`
	FirstActivity        time.Time     `json:"first_activity"`
	LastActivity         time.Time     `json:"last_activity"`
	
	// Processing characteristics
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	MaxProcessingTime    time.Duration `json:"max_processing_time"`
	TotalProcessingSessions int         `json:"total_processing_sessions"`
}

// ProcessingSessionMetrics tracks individual Claude processing sessions
type ProcessingSessionMetrics struct {
	PromptID             string        `json:"prompt_id"`
	ProjectName          string        `json:"project_name"`
	EstimatedTime        time.Duration `json:"estimated_time"`
	ActualTime           time.Duration `json:"actual_time"`
	EstimationAccuracy   float64       `json:"estimation_accuracy"` // Actual / Estimated
	StartTime           time.Time     `json:"start_time"`
	EndTime             time.Time     `json:"end_time"`
	ComplexityHint      string        `json:"complexity_hint"`
	TokensCount         *int          `json:"tokens_count"`
	PromptLength        int           `json:"prompt_length"`
}

// EnhancedReportGenerator creates detailed reports with Claude processing insights
type EnhancedReportGenerator struct {
	timezone *time.Location
}

/**
 * CONTEXT:   Factory for creating enhanced report generator with timezone support
 * INPUT:     Timezone string for proper time calculations and display
 * OUTPUT:    Configured EnhancedReportGenerator instance
 * BUSINESS:  Generate accurate reports with proper timezone handling for global users
 * CHANGE:    Initial enhanced report generator with timezone awareness
 * RISK:      Low - Report generator factory with timezone configuration
 */
func NewEnhancedReportGenerator(timezone string) (*EnhancedReportGenerator, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC // fallback
	}
	
	return &EnhancedReportGenerator{
		timezone: loc,
	}, nil
}

/**
 * CONTEXT:   Generate comprehensive daily report with Claude processing breakdown
 * INPUT:     Work blocks and sessions data for a specific day
 * OUTPUT:    Enhanced metrics with detailed time breakdown and processing insights
 * BUSINESS:  Provide users with accurate view of their productivity including AI assistance time
 * CHANGE:    Enhanced daily reporting with Claude processing time separation
 * RISK:      Medium - Report accuracy affects user productivity insights
 */
func (erg *EnhancedReportGenerator) GenerateDailyReport(workBlocks []*WorkBlock, sessions []*Session, date time.Time) *EnhancedWorkMetrics {
	if len(workBlocks) == 0 {
		return &EnhancedWorkMetrics{} // Empty day
	}
	
	// Initialize metrics
	metrics := &EnhancedWorkMetrics{
		ProjectBreakdown: make([]EnhancedProjectMetrics, 0),
	}
	
	// Calculate time boundaries
	var workStart, workEnd time.Time
	for i, block := range workBlocks {
		blockData := block.ToData()
		if i == 0 || blockData.StartTime.Before(workStart) {
			workStart = blockData.StartTime
		}
		blockEndTime := blockData.EndTime
		if blockEndTime.IsZero() {
			blockEndTime = blockData.LastActivityTime
		}
		if i == 0 || blockEndTime.After(workEnd) {
			workEnd = blockEndTime
		}
	}
	
	metrics.WorkStartTime = workStart
	metrics.WorkEndTime = workEnd
	metrics.TotalScheduleTime = workEnd.Sub(workStart)
	
	// Calculate time breakdowns
	var totalUserTime, totalClaudeTime time.Duration
	processingBlockCount := 0
	var processingTimes []time.Duration
	
	projectData := make(map[string]*EnhancedProjectMetrics)
	
	for _, block := range workBlocks {
		blockData := block.ToData()
		
		// Calculate user interaction time (total block time - Claude processing time)
		userInteractionTime := time.Duration(blockData.DurationSeconds)*time.Second - 
							   time.Duration(blockData.ClaudeProcessingSeconds)*time.Second
		claudeProcessingTime := time.Duration(blockData.ClaudeProcessingSeconds) * time.Second
		
		totalUserTime += userInteractionTime
		totalClaudeTime += claudeProcessingTime
		
		// Track processing blocks
		if claudeProcessingTime > 0 {
			processingBlockCount++
			processingTimes = append(processingTimes, claudeProcessingTime)
		}
		
		// Aggregate by project
		projectName := blockData.ProjectName
		if existing, found := projectData[projectName]; found {
			existing.UserInteractionTime += userInteractionTime
			existing.ClaudeProcessingTime += claudeProcessingTime
			existing.TotalTime += time.Duration(blockData.DurationSeconds) * time.Second
			existing.WorkBlocks++
			
			if claudeProcessingTime > 0 {
				existing.ProcessingBlocks++
				if claudeProcessingTime > existing.MaxProcessingTime {
					existing.MaxProcessingTime = claudeProcessingTime
				}
			}
			
			if blockData.StartTime.Before(existing.FirstActivity) {
				existing.FirstActivity = blockData.StartTime
			}
			if blockData.LastActivityTime.After(existing.LastActivity) {
				existing.LastActivity = blockData.LastActivityTime
			}
		} else {
			processingBlocks := 0
			if claudeProcessingTime > 0 {
				processingBlocks = 1
			}
			
			projectData[projectName] = &EnhancedProjectMetrics{
				ProjectName:           projectName,
				UserInteractionTime:   userInteractionTime,
				ClaudeProcessingTime:  claudeProcessingTime,
				TotalTime:            time.Duration(blockData.DurationSeconds) * time.Second,
				WorkBlocks:           1,
				ProcessingBlocks:     processingBlocks,
				FirstActivity:        blockData.StartTime,
				LastActivity:         blockData.LastActivityTime,
				MaxProcessingTime:    claudeProcessingTime,
			}
		}
	}
	
	// Calculate idle time
	totalActiveTime := totalUserTime + totalClaudeTime
	metrics.IdleTime = metrics.TotalScheduleTime - totalActiveTime
	if metrics.IdleTime < 0 {
		metrics.IdleTime = 0
	}
	
	// Set time metrics
	metrics.UserInteractionTime = totalUserTime
	metrics.ClaudeProcessingTime = totalClaudeTime
	
	// Calculate percentages
	if metrics.TotalScheduleTime > 0 {
		totalSeconds := metrics.TotalScheduleTime.Seconds()
		metrics.EfficiencyPercent = (totalActiveTime.Seconds() / totalSeconds) * 100
		metrics.UserActivityPercent = (totalUserTime.Seconds() / totalSeconds) * 100
		metrics.ClaudeActivityPercent = (totalClaudeTime.Seconds() / totalSeconds) * 100
		metrics.IdlePercent = (metrics.IdleTime.Seconds() / totalSeconds) * 100
	}
	
	// Set activity counts
	metrics.TotalSessions = len(sessions)
	metrics.TotalWorkBlocks = len(workBlocks)
	metrics.ProcessingBlocks = processingBlockCount
	
	// Calculate averages
	if len(workBlocks) > 0 {
		metrics.AverageBlockSize = time.Duration(int64(totalActiveTime) / int64(len(workBlocks)))
	}
	if len(processingTimes) > 0 {
		var totalProcessingTime time.Duration
		for _, pt := range processingTimes {
			totalProcessingTime += pt
		}
		metrics.AverageProcessingTime = time.Duration(int64(totalProcessingTime) / int64(len(processingTimes)))
	}
	
	// Calculate project percentages and sort
	totalProjectTime := totalUserTime + totalClaudeTime
	projectMetrics := make([]EnhancedProjectMetrics, 0, len(projectData))
	for _, pm := range projectData {
		if totalProjectTime > 0 {
			pm.UserTimePercent = (pm.UserInteractionTime.Seconds() / totalProjectTime.Seconds()) * 100
			pm.ClaudeTimePercent = (pm.ClaudeProcessingTime.Seconds() / totalProjectTime.Seconds()) * 100
			pm.TotalTimePercent = (pm.TotalTime.Seconds() / totalProjectTime.Seconds()) * 100
		}
		
		// Calculate average processing time for project
		if pm.ProcessingBlocks > 0 {
			pm.AverageProcessingTime = time.Duration(int64(pm.ClaudeProcessingTime) / int64(pm.ProcessingBlocks))
		}
		
		projectMetrics = append(projectMetrics, *pm)
	}
	
	// Sort by total time (descending)
	sort.Slice(projectMetrics, func(i, j int) bool {
		return projectMetrics[i].TotalTime > projectMetrics[j].TotalTime
	})
	
	metrics.ProjectBreakdown = projectMetrics
	metrics.ProjectCount = len(projectMetrics)
	
	return metrics
}

/**
 * CONTEXT:   Generate formatted text report with enhanced Claude processing insights
 * INPUT:     Enhanced work metrics with Claude processing breakdown
 * OUTPUT:    Human-readable report string with detailed time analysis
 * BUSINESS:  Provide users with clear, actionable productivity insights
 * CHANGE:    Enhanced text reporting with Claude processing visualization
 * RISK:      Low - Text formatting utility with no business logic
 */
func (erg *EnhancedReportGenerator) FormatEnhancedReport(metrics *EnhancedWorkMetrics, title string) string {
	if metrics.TotalScheduleTime == 0 {
		return fmt.Sprintf("\n%s\n%s\n\nNo activity recorded.\n", 
			title, 
			strings.Repeat("=", len(title)))
	}
	
	report := fmt.Sprintf(`
%s
%s

⏰ TIME BREAKDOWN
├── User Interaction Time:  %s (%.1f%%)
├── Claude Processing Time: %s (%.1f%%)
├── Idle Time:             %s (%.1f%%)
└── Total Schedule Time:    %s

📊 PRODUCTIVITY METRICS  
├── Overall Efficiency:     %.1f%% (Active Time / Schedule Time)
├── AI Assistance Ratio:    %.1f%% (Claude Time / Active Time)
├── Average Work Block:     %s
├── Average Processing:     %s
└── Schedule: %s - %s

🔢 ACTIVITY SUMMARY
├── Work Sessions:         %d
├── Work Blocks:           %d
├── Processing Blocks:     %d (%.1f%% of blocks used Claude)
└── Projects Worked On:    %d

`,
		title,
		strings.Repeat("=", len(title)),
		formatDuration(metrics.UserInteractionTime), metrics.UserActivityPercent,
		formatDuration(metrics.ClaudeProcessingTime), metrics.ClaudeActivityPercent,
		formatDuration(metrics.IdleTime), metrics.IdlePercent,
		formatDuration(metrics.TotalScheduleTime),
		metrics.EfficiencyPercent,
		calculateAIAssistanceRatio(metrics),
		formatDuration(metrics.AverageBlockSize),
		formatDuration(metrics.AverageProcessingTime),
		metrics.WorkStartTime.In(erg.timezone).Format("15:04"),
		metrics.WorkEndTime.In(erg.timezone).Format("15:04"),
		metrics.TotalSessions,
		metrics.TotalWorkBlocks,
		metrics.ProcessingBlocks,
		calculateProcessingBlockPercent(metrics),
		metrics.ProjectCount,
	)
	
	// Add project breakdown
	if len(metrics.ProjectBreakdown) > 0 {
		report += "📁 PROJECT BREAKDOWN\n"
		for i, project := range metrics.ProjectBreakdown {
			if i >= 5 { // Limit to top 5 projects
				remaining := len(metrics.ProjectBreakdown) - i
				report += fmt.Sprintf("└── ... and %d more projects\n\n", remaining)
				break
			}
			
			report += fmt.Sprintf("├── %s\n", project.ProjectName)
			report += fmt.Sprintf("│   ├── User Time:    %s (%.1f%%)\n", 
				formatDuration(project.UserInteractionTime), project.UserTimePercent)
			report += fmt.Sprintf("│   ├── Claude Time:  %s (%.1f%%)\n", 
				formatDuration(project.ClaudeProcessingTime), project.ClaudeTimePercent)
			report += fmt.Sprintf("│   ├── Work Blocks:  %d (%d with Claude processing)\n", 
				project.WorkBlocks, project.ProcessingBlocks)
			
			if project.ProcessingBlocks > 0 {
				report += fmt.Sprintf("│   └── Avg Processing: %s (Max: %s)\n", 
					formatDuration(project.AverageProcessingTime), 
					formatDuration(project.MaxProcessingTime))
			} else {
				report += "│   └── No Claude processing in this project\n"
			}
		}
		report += "\n"
	}
	
	// Add processing insights
	if metrics.ClaudeProcessingTime > 0 {
		report += "🤖 CLAUDE PROCESSING INSIGHTS\n"
		report += fmt.Sprintf("├── Total Processing Time: %s\n", formatDuration(metrics.ClaudeProcessingTime))
		report += fmt.Sprintf("├── Processing Efficiency: %.1f%% of total active time\n", 
			calculateProcessingEfficiency(metrics))
		report += fmt.Sprintf("├── Average Processing Session: %s\n", formatDuration(metrics.AverageProcessingTime))
		report += fmt.Sprintf("└── Processing Blocks: %d of %d total blocks (%.1f%%)\n\n",
			metrics.ProcessingBlocks, metrics.TotalWorkBlocks, calculateProcessingBlockPercent(metrics))
		
		// Add estimation accuracy if available
		if metrics.ProcessingEstimatorStats.TotalEstimations > 0 {
			report += "📈 PROCESSING TIME ESTIMATION\n"
			report += fmt.Sprintf("├── Total Estimations: %d\n", metrics.ProcessingEstimatorStats.TotalEstimations)
			report += "├── Average Estimates by Complexity:\n"
			for level, avg := range metrics.ProcessingEstimatorStats.AverageEstimates {
				report += fmt.Sprintf("│   ├── %s: %.1fs\n", level, avg)
			}
			report += "\n"
		}
	}
	
	return report
}

/**
 * CONTEXT:   Generate processing session analysis report
 * INPUT:     List of processing session metrics for detailed analysis
 * OUTPUT:    Formatted report showing processing patterns and accuracy
 * BUSINESS:  Help users understand Claude processing patterns and estimation accuracy
 * CHANGE:    Initial processing session analysis reporting
 * RISK:      Low - Analysis reporting utility for processing insights
 */
func (erg *EnhancedReportGenerator) GenerateProcessingAnalysisReport(sessions []ProcessingSessionMetrics) string {
	if len(sessions) == 0 {
		return "\nNo Claude processing sessions found.\n"
	}
	
	// Calculate analytics
	var totalEstimated, totalActual time.Duration
	var accuracySum float64
	complexityMap := make(map[string]int)
	projectMap := make(map[string]time.Duration)
	
	for _, session := range sessions {
		totalEstimated += session.EstimatedTime
		totalActual += session.ActualTime
		accuracySum += session.EstimationAccuracy
		
		complexityMap[session.ComplexityHint]++
		projectMap[session.ProjectName] += session.ActualTime
	}
	
	avgAccuracy := accuracySum / float64(len(sessions))
	overallAccuracy := (totalActual.Seconds() / totalEstimated.Seconds()) * 100
	
	report := fmt.Sprintf(`
🤖 CLAUDE PROCESSING ANALYSIS
=============================

📊 OVERALL STATISTICS
├── Total Processing Sessions: %d
├── Total Estimated Time:     %s
├── Total Actual Time:        %s
├── Overall Accuracy:         %.1f%% (Actual/Estimated ratio)
├── Average Session Accuracy: %.2f
└── Time Range: %s - %s

🧠 COMPLEXITY BREAKDOWN
`,
		len(sessions),
		formatDuration(totalEstimated),
		formatDuration(totalActual),
		overallAccuracy,
		avgAccuracy,
		sessions[0].StartTime.In(erg.timezone).Format("15:04"),
		sessions[len(sessions)-1].EndTime.In(erg.timezone).Format("15:04"),
	)
	
	// Add complexity breakdown
	for complexity, count := range complexityMap {
		percent := float64(count) / float64(len(sessions)) * 100
		report += fmt.Sprintf("├── %s: %d sessions (%.1f%%)\n", 
			strings.Title(complexity), count, percent)
	}
	
	report += "\n📁 PROJECT PROCESSING TIME\n"
	
	// Sort projects by processing time
	type projectTime struct {
		name string
		time time.Duration
	}
	var sortedProjects []projectTime
	for name, duration := range projectMap {
		sortedProjects = append(sortedProjects, projectTime{name, duration})
	}
	sort.Slice(sortedProjects, func(i, j int) bool {
		return sortedProjects[i].time > sortedProjects[j].time
	})
	
	for _, pt := range sortedProjects {
		percent := pt.time.Seconds() / totalActual.Seconds() * 100
		report += fmt.Sprintf("├── %s: %s (%.1f%%)\n", pt.name, formatDuration(pt.time), percent)
	}
	
	// Add recent sessions
	report += "\n⏱️ RECENT PROCESSING SESSIONS\n"
	
	// Show last 5 sessions
	start := len(sessions) - 5
	if start < 0 {
		start = 0
	}
	
	for i := start; i < len(sessions); i++ {
		session := sessions[i]
		report += fmt.Sprintf("├── %s | %s | Est: %s, Actual: %s (%.1f%% accuracy)\n",
			session.StartTime.In(erg.timezone).Format("15:04:05"),
			session.ProjectName,
			formatDuration(session.EstimatedTime),
			formatDuration(session.ActualTime),
			session.EstimationAccuracy*100,
		)
	}
	
	report += "\n"
	return report
}

// Helper functions for enhanced reporting

func calculateAIAssistanceRatio(metrics *EnhancedWorkMetrics) float64 {
	activeTime := metrics.UserInteractionTime + metrics.ClaudeProcessingTime
	if activeTime == 0 {
		return 0
	}
	return (metrics.ClaudeProcessingTime.Seconds() / activeTime.Seconds()) * 100
}

func calculateProcessingBlockPercent(metrics *EnhancedWorkMetrics) float64 {
	if metrics.TotalWorkBlocks == 0 {
		return 0
	}
	return float64(metrics.ProcessingBlocks) / float64(metrics.TotalWorkBlocks) * 100
}

func calculateProcessingEfficiency(metrics *EnhancedWorkMetrics) float64 {
	activeTime := metrics.UserInteractionTime + metrics.ClaudeProcessingTime
	if activeTime == 0 {
		return 0
	}
	return (metrics.ClaudeProcessingTime.Seconds() / activeTime.Seconds()) * 100
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%02ds", minutes, seconds)  
	}
	return fmt.Sprintf("%ds", seconds)
}

