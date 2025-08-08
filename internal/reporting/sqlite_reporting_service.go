/**
 * CONTEXT:   Pure SQLite reporting service orchestrating all repositories for comprehensive analytics
 * INPUT:     SQLite repositories for sessions, work blocks, activities, and projects
 * OUTPUT:    Complete reporting functionality with time-based analytics and insights
 * BUSINESS:  Reporting service provides daily, weekly, monthly reports using only SQLite data
 * CHANGE:    Initial implementation replacing hybrid data sources with pure SQLite approach
 * RISK:      Medium - Core reporting functionality affecting user-facing analytics
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

// SQLiteReportingService orchestrates SQLite repositories for comprehensive reporting
type SQLiteReportingService struct {
	sessionRepo   *sqlite.SessionRepository
	workBlockRepo *sqlite.WorkBlockRepository
	activityRepo  *sqlite.ActivityRepository
	projectRepo   *sqlite.ProjectRepository
}

// NewSQLiteReportingService creates a new SQLite-based reporting service
func NewSQLiteReportingService(
	sessionRepo *sqlite.SessionRepository,
	workBlockRepo *sqlite.WorkBlockRepository,
	activityRepo *sqlite.ActivityRepository,
	projectRepo *sqlite.ProjectRepository,
) *SQLiteReportingService {
	return &SQLiteReportingService{
		sessionRepo:   sessionRepo,
		workBlockRepo: workBlockRepo,
		activityRepo:  activityRepo,
		projectRepo:   projectRepo,
	}
}

/**
 * CONTEXT:   Generate comprehensive daily report from SQLite data sources
 * INPUT:     User ID, date for report generation, timezone context
 * OUTPUT:    Enhanced daily report with work blocks, sessions, projects, and insights
 * BUSINESS:  Daily reports are primary user interface for work tracking analytics
 * CHANGE:    Initial SQLite-only daily report generation with comprehensive metrics
 * RISK:      Medium - Core reporting functionality with multiple data source orchestration
 */
func (srs *SQLiteReportingService) GenerateDailyReport(ctx context.Context, userID string, date time.Time) (*EnhancedDailyReport, error) {
	// Set up date boundaries (start of day to end of day)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-1 * time.Nanosecond)

	// Get sessions for the day
	sessions, err := srs.sessionRepo.FindByUserAndTimeRange(ctx, userID, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions for daily report: %w", err)
	}

	report := &EnhancedDailyReport{
		Date:             date,
		ProjectBreakdown: make([]ProjectBreakdown, 0),
		HourlyBreakdown:  make([]HourlyData, 0),
		WorkBlocks:       make([]WorkBlockSummary, 0),
		Insights:         make([]string, 0),
	}

	// Process sessions for the day
	totalSessions := len(sessions)
	if totalSessions == 0 {
		return report, nil
	}

	// Calculate session boundaries and active periods
	var firstActivity, lastActivity time.Time
	var sessionDurations []time.Duration
	activeSessions := 0

	for _, session := range sessions {
		// Check if session is active using time-based logic
		currentTime := time.Now()
		isActive := currentTime.After(session.StartTime) && currentTime.Before(session.EndTime)

		if isActive {
			activeSessions++
		}

		// Track session timing
		if !session.FirstActivityTime.IsZero() {
			if firstActivity.IsZero() || session.FirstActivityTime.Before(firstActivity) {
				firstActivity = session.FirstActivityTime
			}
		}

		if !session.LastActivityTime.IsZero() {
			if lastActivity.IsZero() || session.LastActivityTime.After(lastActivity) {
				lastActivity = session.LastActivityTime
			}
		}

		// Calculate session duration based on actual activity
		if !session.FirstActivityTime.IsZero() && !session.LastActivityTime.IsZero() {
			sessionDuration := session.LastActivityTime.Sub(session.FirstActivityTime)
			if sessionDuration > 0 {
				sessionDurations = append(sessionDurations, sessionDuration)
			}
		}

		// Get work blocks for this session
		workBlocks, err := srs.workBlockRepo.GetBySession(ctx, session.ID, 0)
		if err != nil {
			continue // Skip this session on work block error
		}

		// Process work blocks for comprehensive analytics
		for _, workBlock := range workBlocks {
			if err := srs.processWorkBlockForReport(ctx, workBlock, report); err != nil {
				continue // Skip problematic work blocks
			}
		}
	}

	// Set report session summary
	report.TotalSessions = totalSessions
	report.SessionSummary = srs.calculateSessionSummary(sessionDurations)

	// Set schedule timing if we have activity data
	if !firstActivity.IsZero() && !lastActivity.IsZero() {
		report.StartTime = firstActivity
		report.EndTime = lastActivity
		report.ScheduleHours = lastActivity.Sub(firstActivity).Hours()
	}

	// Calculate project breakdown and insights
	srs.calculateProjectBreakdown(report)
	srs.calculateHourlyBreakdown(report)
	srs.generateDailyInsights(report)

	return report, nil
}

/**
 * CONTEXT:   Generate weekly report with daily breakdown and trend analysis
 * INPUT:     User ID, start of week date for 7-day analysis period
 * OUTPUT:    Enhanced weekly report with daily patterns, trends, and productivity insights
 * BUSINESS:  Weekly reports show work patterns and consistency over time
 * CHANGE:    Initial SQLite-only weekly report with comprehensive trend analysis
 * RISK:      Medium - Multi-day aggregation with trend calculation logic
 */
func (srs *SQLiteReportingService) GenerateWeeklyReport(ctx context.Context, userID string, weekStart time.Time) (*EnhancedWeeklyReport, error) {
	// Ensure weekStart is beginning of day
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	weekEnd := weekStart.Add(7 * 24 * time.Hour).Add(-1 * time.Nanosecond)

	// Get all sessions for the week
	_, err := srs.sessionRepo.FindByUserAndTimeRange(ctx, userID, weekStart, weekEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions for weekly report: %w", err)
	}

	report := &EnhancedWeeklyReport{
		WeekStart:        weekStart,
		WeekEnd:          weekEnd,
		WeekNumber:       srs.getWeekNumber(weekStart),
		Year:             weekStart.Year(),
		DailyBreakdown:   make([]DaySummary, 7),
		ProjectBreakdown: make([]ProjectBreakdown, 0),
		Insights:         make([]WeeklyInsight, 0),
		Trends:           make([]Trend, 0),
	}

	// Initialize daily breakdown with all 7 days
	for i := 0; i < 7; i++ {
		day := weekStart.Add(time.Duration(i) * 24 * time.Hour)
		report.DailyBreakdown[i] = DaySummary{
			Date:    day,
			DayName: day.Format("Mon"),
			Hours:   0.0,
		}
	}

	// Process each day's data
	projectTotals := make(map[string]*ProjectBreakdown)
	totalWorkHours := 0.0
	bestDayHours := 0.0

	for i, day := range report.DailyBreakdown {
		dailyReport, err := srs.GenerateDailyReport(ctx, userID, day.Date)
		if err != nil {
			continue // Skip days with errors
		}

		// Update daily summary
		report.DailyBreakdown[i].Hours = dailyReport.TotalWorkHours
		report.DailyBreakdown[i].ClaudeSessions = dailyReport.TotalSessions
		report.DailyBreakdown[i].WorkBlocks = dailyReport.TotalWorkBlocks

		// Track productivity status
		if dailyReport.TotalWorkHours >= 8 {
			report.DailyBreakdown[i].Status = "excellent"
		} else if dailyReport.TotalWorkHours >= 6 {
			report.DailyBreakdown[i].Status = "good"
		} else if dailyReport.TotalWorkHours >= 3 {
			report.DailyBreakdown[i].Status = "low"
		} else {
			report.DailyBreakdown[i].Status = "none"
		}

		totalWorkHours += dailyReport.TotalWorkHours

		// Track best day
		if dailyReport.TotalWorkHours > bestDayHours {
			bestDayHours = dailyReport.TotalWorkHours
			report.MostProductiveDay = report.DailyBreakdown[i]
		}

		// Aggregate project data
		for _, project := range dailyReport.ProjectBreakdown {
			if existing, exists := projectTotals[project.Name]; exists {
				existing.Hours += project.Hours
				existing.WorkBlocks += project.WorkBlocks
				existing.ClaudeSessions += project.ClaudeSessions
			} else {
				projectTotals[project.Name] = &ProjectBreakdown{
					Name:           project.Name,
					Hours:          project.Hours,
					WorkBlocks:     project.WorkBlocks,
					ClaudeSessions: project.ClaudeSessions,
				}
			}
		}
	}

	// Set weekly totals
	report.TotalWorkHours = totalWorkHours
	report.DailyAverage = totalWorkHours / 7.0

	// Convert project totals to slice and calculate percentages
	for _, project := range projectTotals {
		if totalWorkHours > 0 {
			project.Percentage = (project.Hours / totalWorkHours) * 100
		}
		report.ProjectBreakdown = append(report.ProjectBreakdown, *project)
	}

	// Sort projects by hours
	sort.Slice(report.ProjectBreakdown, func(i, j int) bool {
		return report.ProjectBreakdown[i].Hours > report.ProjectBreakdown[j].Hours
	})

	// Generate insights and trends
	srs.generateWeeklyInsights(report)
	srs.generateWeeklyTrends(report)

	return report, nil
}

/**
 * CONTEXT:   Generate monthly report with heatmap, achievements, and comprehensive statistics
 * INPUT:     User ID, month start date for full month analysis
 * OUTPUT:    Enhanced monthly report with daily progress, achievements, and long-term trends
 * BUSINESS:  Monthly reports provide long-term productivity insights and goal tracking
 * CHANGE:    Initial SQLite-only monthly report with comprehensive analytics
 * RISK:      Medium - Month-long data aggregation with complex achievement calculation
 */
func (srs *SQLiteReportingService) GenerateMonthlyReport(ctx context.Context, userID string, monthStart time.Time) (*EnhancedMonthlyReport, error) {
	// Set up month boundaries
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, monthStart.Location())
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-1 * time.Nanosecond)

	report := &EnhancedMonthlyReport{
		Month:            monthStart,
		MonthName:        monthStart.Format("January"),
		Year:             monthStart.Year(),
		TotalDays:        srs.getDaysInMonth(monthStart),
		DailyProgress:    make([]DaySummary, 0),
		WeeklyBreakdown:  make([]WeekSummary, 0),
		ProjectBreakdown: make([]ProjectBreakdown, 0),
		Achievements:     make([]Achievement, 0),
		Trends:           make([]Trend, 0),
		Insights:         make([]string, 0),
	}

	// Calculate days completed
	now := time.Now()
	if now.Before(monthEnd) {
		report.DaysCompleted = now.Day()
	} else {
		report.DaysCompleted = report.TotalDays
	}

	// Generate daily progress for heatmap
	projectTotals := make(map[string]*ProjectBreakdown)
	totalWorkHours := 0.0
	workingDays := 0
	totalSessions := 0
	bestDayHours := 0.0

	for day := monthStart; day.Before(monthEnd); day = day.Add(24 * time.Hour) {
		if day.After(now) {
			break // Don't process future days
		}

		dailyReport, err := srs.GenerateDailyReport(ctx, userID, day)
		if err != nil {
			continue
		}

		daySummary := DaySummary{
			Date:           day,
			DayName:        day.Format("Mon"),
			Hours:          dailyReport.TotalWorkHours,
			ClaudeSessions: dailyReport.TotalSessions,
			WorkBlocks:     dailyReport.TotalWorkBlocks,
		}

		// Set status based on hours
		if dailyReport.TotalWorkHours >= 8 {
			daySummary.Status = "excellent"
		} else if dailyReport.TotalWorkHours >= 6 {
			daySummary.Status = "good"
		} else if dailyReport.TotalWorkHours >= 3 {
			daySummary.Status = "low"
		} else {
			daySummary.Status = "none"
		}

		report.DailyProgress = append(report.DailyProgress, daySummary)

		// Update totals
		totalWorkHours += dailyReport.TotalWorkHours
		totalSessions += dailyReport.TotalSessions

		if dailyReport.TotalWorkHours > 1 {
			workingDays++
		}

		if dailyReport.TotalWorkHours > bestDayHours {
			bestDayHours = dailyReport.TotalWorkHours
			report.BestDay = daySummary
		}

		// Aggregate project data
		for _, project := range dailyReport.ProjectBreakdown {
			if existing, exists := projectTotals[project.Name]; exists {
				existing.Hours += project.Hours
				existing.WorkBlocks += project.WorkBlocks
				existing.ClaudeSessions += project.ClaudeSessions
			} else {
				projectTotals[project.Name] = &ProjectBreakdown{
					Name:           project.Name,
					Hours:          project.Hours,
					WorkBlocks:     project.WorkBlocks,
					ClaudeSessions: project.ClaudeSessions,
				}
			}
		}
	}

	// Set monthly statistics
	report.TotalWorkHours = totalWorkHours
	if report.DaysCompleted > 0 {
		report.DailyAverage = totalWorkHours / float64(report.DaysCompleted)
	}

	// Project future hours if month is not complete
	if report.DaysCompleted < report.TotalDays && report.DailyAverage > 0 {
		remainingDays := float64(report.TotalDays - report.DaysCompleted)
		report.ProjectedHours = totalWorkHours + (report.DailyAverage * remainingDays)
	}

	// Process project breakdown
	for _, project := range projectTotals {
		if totalWorkHours > 0 {
			project.Percentage = (project.Hours / totalWorkHours) * 100
		}
		report.ProjectBreakdown = append(report.ProjectBreakdown, *project)
	}

	// Sort projects by hours
	sort.Slice(report.ProjectBreakdown, func(i, j int) bool {
		return report.ProjectBreakdown[i].Hours > report.ProjectBreakdown[j].Hours
	})

	// Set monthly stats
	report.MonthlyStats = MonthlyStats{
		WorkingDays:       workingDays,
		TotalSessions:     totalSessions,
		AvgSessionsPerDay: float64(totalSessions) / float64(report.DaysCompleted),
		ConsistencyScore:  srs.calculateConsistencyScore(report.DailyProgress),
		ClaudePrompts:     totalSessions, // Approximation
	}

	// Generate achievements, trends, and insights
	srs.generateMonthlyAchievements(report)
	srs.generateMonthlyTrends(report)
	srs.generateMonthlyInsights(report)

	return report, nil
}

// Helper methods for report generation

/**
 * CONTEXT:   Process individual work block for daily report aggregation
 * INPUT:     Work block entity from SQLite and report object for data aggregation
 * OUTPUT:    Updated report with work block metrics, duration, and activity data
 * BUSINESS:  Work blocks are core building blocks of work time calculation
 * CHANGE:    Initial work block processing with comprehensive metrics extraction
 * RISK:      Low - Individual work block processing with validation
 */
func (srs *SQLiteReportingService) processWorkBlockForReport(ctx context.Context, workBlock *sqlite.WorkBlock, report *EnhancedDailyReport) error {
	// Calculate work block duration
	var duration time.Duration
	if workBlock.EndTime != nil {
		duration = workBlock.EndTime.Sub(workBlock.StartTime)
	} else {
		// Active work block - calculate duration to now
		duration = time.Now().Sub(workBlock.StartTime)
	}

	// Add to total work hours
	report.TotalWorkHours += duration.Hours()

	// Get project information
	project, err := srs.projectRepo.GetByID(ctx, workBlock.ProjectID)
	if err == nil && project != nil {
		// Create work block summary
		workBlockSummary := WorkBlockSummary{
			StartTime:   workBlock.StartTime,
			EndTime:     workBlock.StartTime.Add(duration),
			Duration:    duration,
			ProjectName: project.Name,
			Activities:  int(workBlock.ActivityCount),
		}

		// Handle nil end time
		if workBlock.EndTime != nil {
			workBlockSummary.EndTime = *workBlock.EndTime
		}

		report.WorkBlocks = append(report.WorkBlocks, workBlockSummary)
	}

	report.TotalWorkBlocks++

	return nil
}

/**
 * CONTEXT:   Calculate project breakdown from work blocks in daily report
 * INPUT:     Report object with work blocks requiring project aggregation
 * OUTPUT:    Project breakdown with hours, percentages, and work block counts
 * BUSINESS:  Project breakdown shows time allocation across different projects
 * CHANGE:    Initial project aggregation with percentage calculation
 * RISK:      Low - Data aggregation and percentage calculation logic
 */
func (srs *SQLiteReportingService) calculateProjectBreakdown(report *EnhancedDailyReport) {
	projectMap := make(map[string]*ProjectBreakdown)

	// Aggregate by project
	for _, wb := range report.WorkBlocks {
		if existing, exists := projectMap[wb.ProjectName]; exists {
			existing.Hours += wb.Duration.Hours()
			existing.WorkBlocks++
		} else {
			projectMap[wb.ProjectName] = &ProjectBreakdown{
				Name:       wb.ProjectName,
				Hours:      wb.Duration.Hours(),
				WorkBlocks: 1,
			}
		}
	}

	// Convert to slice and calculate percentages
	for _, project := range projectMap {
		if report.TotalWorkHours > 0 {
			project.Percentage = (project.Hours / report.TotalWorkHours) * 100
		}
		report.ProjectBreakdown = append(report.ProjectBreakdown, *project)
	}

	// Sort by hours descending
	sort.Slice(report.ProjectBreakdown, func(i, j int) bool {
		return report.ProjectBreakdown[i].Hours > report.ProjectBreakdown[j].Hours
	})
}

/**
 * CONTEXT:   Calculate hourly work distribution for daily timeline visualization
 * INPUT:     Report object with work blocks requiring hourly breakdown
 * OUTPUT:    Hourly data array with work hours per hour of the day
 * BUSINESS:  Hourly breakdown shows productivity patterns throughout the day
 * CHANGE:    Initial hourly aggregation for visual timeline display
 * RISK:      Low - Time-based aggregation with hour extraction
 */
func (srs *SQLiteReportingService) calculateHourlyBreakdown(report *EnhancedDailyReport) {
	hourlyMap := make(map[int]*HourlyData)

	// Initialize all 24 hours
	for hour := 0; hour < 24; hour++ {
		hourlyMap[hour] = &HourlyData{
			Hour:       hour,
			Hours:      0.0,
			WorkBlocks: 0,
			IsActive:   false,
		}
	}

	// Aggregate work blocks by hour
	for _, wb := range report.WorkBlocks {
		startHour := wb.StartTime.Hour()
		endHour := wb.EndTime.Hour()

		// Handle work blocks that span multiple hours
		if startHour == endHour {
			// Work block within single hour
			hourlyMap[startHour].Hours += wb.Duration.Hours()
			hourlyMap[startHour].WorkBlocks++
			hourlyMap[startHour].IsActive = true
		} else {
			// Work block spans multiple hours - distribute proportionally  
			_ = wb.Duration.Minutes() // Total duration for proportional distribution
			for hour := startHour; hour <= endHour; hour++ {
				var minutesInHour float64
				if hour == startHour {
					// First hour - from start time to end of hour
					minutesInHour = 60 - float64(wb.StartTime.Minute())
				} else if hour == endHour {
					// Last hour - from start of hour to end time
					minutesInHour = float64(wb.EndTime.Minute())
				} else {
					// Full hour
					minutesInHour = 60
				}

				hoursInThisHour := minutesInHour / 60.0
				if hoursInThisHour > 0 {
					hourlyMap[hour].Hours += hoursInThisHour
					hourlyMap[hour].WorkBlocks++
					hourlyMap[hour].IsActive = true
				}
			}
		}
	}

	// Convert to slice, only including hours with activity
	for hour := 0; hour < 24; hour++ {
		if hourlyMap[hour].IsActive {
			report.HourlyBreakdown = append(report.HourlyBreakdown, *hourlyMap[hour])
		}
	}

	// Sort by hour
	sort.Slice(report.HourlyBreakdown, func(i, j int) bool {
		return report.HourlyBreakdown[i].Hour < report.HourlyBreakdown[j].Hour
	})
}

/**
 * CONTEXT:   Calculate session summary statistics from session durations
 * INPUT:     Array of session durations for statistical analysis
 * OUTPUT:    Session summary with average, longest, shortest, and range information
 * BUSINESS:  Session summaries provide insights into work session patterns
 * CHANGE:    Initial session statistics calculation with duration analysis
 * RISK:      Low - Statistical calculation with proper validation
 */
func (srs *SQLiteReportingService) calculateSessionSummary(sessionDurations []time.Duration) SessionSummary {
	summary := SessionSummary{
		TotalSessions: len(sessionDurations),
	}

	if len(sessionDurations) == 0 {
		return summary
	}

	var totalDuration time.Duration
	longestDuration := time.Duration(0)
	shortestDuration := time.Duration(math.MaxInt64)

	for _, duration := range sessionDurations {
		totalDuration += duration
		if duration > longestDuration {
			longestDuration = duration
		}
		if duration < shortestDuration {
			shortestDuration = duration
		}
	}

	summary.AverageSession = totalDuration / time.Duration(len(sessionDurations))
	summary.LongestSession = longestDuration
	summary.ShortestSession = shortestDuration

	// Create session range description
	if longestDuration > 0 && shortestDuration < time.Duration(math.MaxInt64) {
		summary.SessionRange = fmt.Sprintf("%s - %s", 
			formatDuration(shortestDuration), 
			formatDuration(longestDuration))
	}

	return summary
}

/**
 * CONTEXT:   Generate daily insights from work patterns and productivity metrics
 * INPUT:     Enhanced daily report with work data requiring analysis
 * OUTPUT:    Array of actionable insights and recommendations for productivity improvement
 * BUSINESS:  Daily insights provide personalized recommendations for work optimization
 * CHANGE:    Initial insight generation with pattern recognition and recommendations
 * RISK:      Low - Pattern analysis with insight generation logic
 */
func (srs *SQLiteReportingService) generateDailyInsights(report *EnhancedDailyReport) {
	insights := make([]string, 0)

	// Work duration insights
	if report.TotalWorkHours >= 8 {
		insights = append(insights, "🔥 Excellent productivity with 8+ hours of focused work!")
	} else if report.TotalWorkHours >= 6 {
		insights = append(insights, "👍 Solid work day with good productivity levels")
	} else if report.TotalWorkHours >= 3 {
		insights = append(insights, "⚡ Moderate work session - consider extending focus blocks")
	} else if report.TotalWorkHours > 0 {
		insights = append(insights, "📈 Light work day - great for planning and preparation")
	}

	// Deep work analysis from work blocks
	if len(report.WorkBlocks) > 0 {
		longBlocks := 0
		shortBlocks := 0
		totalDuration := time.Duration(0)

		for _, wb := range report.WorkBlocks {
			totalDuration += wb.Duration
			if wb.Duration >= 25*time.Minute {
				longBlocks++
			} else if wb.Duration < 5*time.Minute {
				shortBlocks++
			}
		}

		deepWorkPercent := 0.0
		if report.TotalWorkHours > 0 {
			deepWorkTime := 0.0
			for _, wb := range report.WorkBlocks {
				if wb.Duration >= 25*time.Minute {
					deepWorkTime += wb.Duration.Hours()
				}
			}
			deepWorkPercent = (deepWorkTime / report.TotalWorkHours) * 100
		}

		if deepWorkPercent >= 70 {
			insights = append(insights, fmt.Sprintf("🧠 Excellent deep work focus: %.1f%% in 25+ minute blocks", deepWorkPercent))
		} else if deepWorkPercent >= 40 {
			insights = append(insights, fmt.Sprintf("🔍 Good focus with %.1f%% deep work - try extending blocks", deepWorkPercent))
		} else {
			insights = append(insights, "🎯 Consider longer focus blocks (25+ minutes) for deep work")
		}

		// Context switching analysis
		if len(report.ProjectBreakdown) == 1 {
			insights = append(insights, "🎯 Single project focus - excellent for deep work flow")
		} else if len(report.ProjectBreakdown) <= 3 {
			insights = append(insights, "👍 Good project focus with minimal context switching")
		} else {
			insights = append(insights, fmt.Sprintf("🔄 High context switching (%d projects) may reduce focus efficiency", len(report.ProjectBreakdown)))
		}
	}

	// Project insights
	if len(report.ProjectBreakdown) > 0 {
		topProject := report.ProjectBreakdown[0]
		insights = append(insights, fmt.Sprintf("📊 Primary focus: %s (%.1f%% of work time)", topProject.Name, topProject.Percentage))
	}

	// Schedule insights
	if !report.StartTime.IsZero() && !report.EndTime.IsZero() {
		scheduleHours := report.EndTime.Sub(report.StartTime).Hours()
		efficiency := (report.TotalWorkHours / scheduleHours) * 100

		if efficiency >= 60 {
			insights = append(insights, fmt.Sprintf("⚡ Great time efficiency: %.1f%% active work", efficiency))
		} else if efficiency >= 30 {
			insights = append(insights, fmt.Sprintf("📈 Moderate efficiency: %.1f%% active work - consider time blocking", efficiency))
		} else if efficiency > 0 {
			insights = append(insights, "⏰ Low efficiency detected - focus on longer active blocks")
		}

		// Early bird / night owl insights
		startHour := report.StartTime.Hour()
		if startHour < 7 {
			insights = append(insights, "🌅 Early bird schedule - excellent for deep focus work")
		} else if startHour > 10 {
			insights = append(insights, "😴 Late start today - consider morning sessions for peak productivity")
		}
	}

	report.Insights = insights
}

/**
 * CONTEXT:   Generate weekly insights from daily patterns and work consistency
 * INPUT:     Enhanced weekly report with daily breakdown requiring pattern analysis
 * OUTPUT:    Array of weekly insights with productivity patterns and recommendations
 * BUSINESS:  Weekly insights identify productivity patterns and suggest improvements
 * CHANGE:    Initial weekly insight generation with consistency and pattern analysis
 * RISK:      Low - Pattern analysis with weekly trend recognition
 */
func (srs *SQLiteReportingService) generateWeeklyInsights(report *EnhancedWeeklyReport) {
	insights := make([]WeeklyInsight, 0)

	// Consistency analysis
	workingDays := 0
	totalHours := 0.0
	weekendHours := 0.0

	for _, day := range report.DailyBreakdown {
		if day.Hours > 1 {
			workingDays++
		}
		totalHours += day.Hours

		// Weekend work analysis
		if day.Date.Weekday() == time.Saturday || day.Date.Weekday() == time.Sunday {
			weekendHours += day.Hours
		}
	}

	// Consistency insights
	if workingDays >= 6 {
		insights = append(insights, WeeklyInsight{
			Type:        "consistency",
			Title:       "Outstanding consistency",
			Description: fmt.Sprintf("%d-day work streak shows excellent dedication", workingDays),
			Icon:        "🔥",
		})
	} else if workingDays >= 4 {
		insights = append(insights, WeeklyInsight{
			Type:        "consistency",
			Title:       "Good work rhythm",
			Description: fmt.Sprintf("%d working days with solid productivity", workingDays),
			Icon:        "📈",
		})
	} else {
		insights = append(insights, WeeklyInsight{
			Type:        "improvement",
			Title:       "Consistency opportunity",
			Description: "Consider establishing a more regular work schedule",
			Icon:        "🎯",
		})
	}

	// Weekend work analysis
	if weekendHours > 0 {
		weekendPercent := (weekendHours / totalHours) * 100
		if weekendPercent > 25 {
			insights = append(insights, WeeklyInsight{
				Type:        "balance",
				Title:       "High weekend activity",
				Description: fmt.Sprintf("%.1f%% of work on weekends - ensure work-life balance", weekendPercent),
				Icon:        "⚖️",
			})
		} else {
			insights = append(insights, WeeklyInsight{
				Type:        "balance",
				Title:       "Balanced schedule",
				Description: fmt.Sprintf("%.1f%% weekend work shows healthy balance", weekendPercent),
				Icon:        "✅",
			})
		}
	}

	// Productivity pattern analysis
	bestDay := report.MostProductiveDay
	if bestDay.Hours > 0 {
		insights = append(insights, WeeklyInsight{
			Type:        "performance",
			Title:       fmt.Sprintf("%s was your peak day", bestDay.DayName),
			Description: fmt.Sprintf("%.1f hours of focused work", bestDay.Hours),
			Icon:        "🏆",
		})
	}

	report.Insights = insights
}

/**
 * CONTEXT:   Generate weekly trends comparing current week to patterns
 * INPUT:     Enhanced weekly report with metrics requiring trend analysis
 * OUTPUT:    Array of trends showing directional changes in productivity metrics
 * BUSINESS:  Weekly trends help identify productivity improvements or declines
 * CHANGE:    Initial trend analysis with directional indicators
 * RISK:      Low - Trend calculation with directional analysis
 */
func (srs *SQLiteReportingService) generateWeeklyTrends(report *EnhancedWeeklyReport) {
	trends := make([]Trend, 0)

	// For now, generate static trends - in future, compare with previous weeks
	if report.TotalWorkHours > 0 {
		trends = append(trends, Trend{
			Metric:    "Total Work",
			Direction: "stable",
			Change:    0.0,
			Period:    "week",
			Icon:      "📊",
		})

		trends = append(trends, Trend{
			Metric:    "Consistency",
			Direction: "stable",
			Change:    0.0,
			Period:    "week",
			Icon:      "📈",
		})
	}

	report.Trends = trends
}

// Helper functions

func (srs *SQLiteReportingService) getWeekNumber(date time.Time) int {
	_, week := date.ISOWeek()
	return week
}

func (srs *SQLiteReportingService) getDaysInMonth(monthStart time.Time) int {
	return monthStart.AddDate(0, 1, -1).Day()
}

func (srs *SQLiteReportingService) calculateConsistencyScore(dailyProgress []DaySummary) float64 {
	if len(dailyProgress) == 0 {
		return 0.0
	}

	workingDays := 0
	for _, day := range dailyProgress {
		if day.Hours > 1 {
			workingDays++
		}
	}

	return (float64(workingDays) / float64(len(dailyProgress))) * 100
}

func (srs *SQLiteReportingService) generateMonthlyAchievements(report *EnhancedMonthlyReport) {
	achievements := make([]Achievement, 0)

	// Work hour achievements
	if report.TotalWorkHours >= 160 {
		achievements = append(achievements, Achievement{
			Type:        "hours",
			Title:       "160+ Hour Month",
			Description: "Excellent monthly productivity",
			Icon:        "🏆",
			Achieved:    true,
		})
	}

	// Consistency achievements
	if report.MonthlyStats.ConsistencyScore >= 80 {
		achievements = append(achievements, Achievement{
			Type:        "consistency",
			Title:       "Consistency Master",
			Description: "80%+ working day consistency",
			Icon:        "🔥",
			Achieved:    true,
		})
	}

	report.Achievements = achievements
}

func (srs *SQLiteReportingService) generateMonthlyTrends(report *EnhancedMonthlyReport) {
	trends := make([]Trend, 0)

	// Static trends for now - future: compare with previous months
	if report.TotalWorkHours > 0 {
		trends = append(trends, Trend{
			Metric:    "Monthly Progress",
			Direction: "stable",
			Change:    0.0,
			Period:    "month",
			Icon:      "📅",
		})
	}

	report.Trends = trends
}

func (srs *SQLiteReportingService) generateMonthlyInsights(report *EnhancedMonthlyReport) {
	insights := make([]string, 0)

	if report.TotalWorkHours >= 160 {
		insights = append(insights, "🏆 Outstanding month with 160+ hours of productive work")
	} else if report.TotalWorkHours >= 120 {
		insights = append(insights, "👍 Strong month with excellent productivity levels")
	} else if report.TotalWorkHours >= 80 {
		insights = append(insights, "📈 Good month with solid work output")
	}

	if report.MonthlyStats.ConsistencyScore >= 80 {
		insights = append(insights, "🔥 Excellent consistency with regular work schedule")
	}

	report.Insights = insights
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0m"
	}

	totalMinutes := int(d.Minutes())
	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	if hours >= 1 {
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if totalMinutes >= 1 {
		return fmt.Sprintf("%dm", totalMinutes)
	} else {
		seconds := int(d.Seconds())
		if seconds == 0 {
			return "0s"
		}
		return fmt.Sprintf("%ds", seconds)
	}
}

// Report data structures (matching the existing types from reporting.go)

type EnhancedDailyReport struct {
	Date                     time.Time             `json:"date"`
	StartTime                time.Time             `json:"start_time"`
	EndTime                  time.Time             `json:"end_time"`
	TotalWorkHours           float64               `json:"total_work_hours"`
	DeepWorkHours            float64               `json:"deep_work_hours"`
	FocusScore               float64               `json:"focus_score"`
	ScheduleHours            float64               `json:"schedule_hours"`
	ClaudeProcessingTime     float64               `json:"claude_processing_time"`
	IdleTime                 float64               `json:"idle_time"`
	EfficiencyPercent        float64               `json:"efficiency_percent"`
	TotalSessions            int                   `json:"total_sessions"`
	ClaudePrompts            int                   `json:"claude_prompts"`
	TotalWorkBlocks          int                   `json:"total_work_blocks"`
	ProjectBreakdown         []ProjectBreakdown    `json:"project_breakdown"`
	HourlyBreakdown          []HourlyData          `json:"hourly_breakdown"`
	WorkBlocks               []WorkBlockSummary    `json:"work_blocks"`
	Insights                 []string              `json:"insights"`
	SessionSummary           SessionSummary        `json:"session_summary"`
	ClaudeActivity           ClaudeActivity        `json:"claude_activity"`
}

type EnhancedWeeklyReport struct {
	WeekStart           time.Time             `json:"week_start"`
	WeekEnd             time.Time             `json:"week_end"`
	WeekNumber          int                   `json:"week_number"`
	Year                int                   `json:"year"`
	TotalWorkHours      float64               `json:"total_work_hours"`
	DailyAverage        float64               `json:"daily_average"`
	ClaudeUsageHours    float64               `json:"claude_usage_hours"`
	ClaudeUsagePercent  float64               `json:"claude_usage_percent"`
	MostProductiveDay   DaySummary            `json:"most_productive_day"`
	DailyBreakdown      []DaySummary          `json:"daily_breakdown"`
	ProjectBreakdown    []ProjectBreakdown    `json:"project_breakdown"`
	Insights            []WeeklyInsight       `json:"insights"`
	Trends              []Trend               `json:"trends"`
	WeeklyStats         WeeklyStats           `json:"weekly_stats"`
}

type EnhancedMonthlyReport struct {
	Month               time.Time             `json:"month"`
	MonthName           string                `json:"month_name"`
	Year                int                   `json:"year"`
	DaysCompleted       int                   `json:"days_completed"`
	TotalDays           int                   `json:"total_days"`
	TotalWorkHours      float64               `json:"total_work_hours"`
	DailyAverage        float64               `json:"daily_average"`
	ProjectedHours      float64               `json:"projected_hours"`
	ClaudeUsageHours    float64               `json:"claude_usage_hours"`
	ClaudeUsagePercent  float64               `json:"claude_usage_percent"`
	BestDay             DaySummary            `json:"best_day"`
	DailyProgress       []DaySummary          `json:"daily_progress"`
	WeeklyBreakdown     []WeekSummary         `json:"weekly_breakdown"`
	ProjectBreakdown    []ProjectBreakdown    `json:"project_breakdown"`
	MonthlyStats        MonthlyStats          `json:"monthly_stats"`
	Achievements        []Achievement         `json:"achievements"`
	Trends              []Trend               `json:"trends"`
	Insights            []string              `json:"insights"`
}

type ProjectBreakdown struct {
	Name            string    `json:"name"`
	Hours           float64   `json:"hours"`
	Percentage      float64   `json:"percentage"`
	WorkBlocks      int       `json:"work_blocks"`
	ClaudeSessions  int       `json:"claude_sessions"`
	ClaudeHours     float64   `json:"claude_hours"`
	LastActivity    time.Time `json:"last_activity"`
}

type HourlyData struct {
	Hour       int     `json:"hour"`
	Hours      float64 `json:"hours"`
	WorkBlocks int     `json:"work_blocks"`
	IsActive   bool    `json:"is_active"`
}

type DaySummary struct {
	Date           time.Time `json:"date"`
	DayName        string    `json:"day_name"`
	Hours          float64   `json:"hours"`
	ClaudeSessions int       `json:"claude_sessions"`
	WorkBlocks     int       `json:"work_blocks"`
	Efficiency     float64   `json:"efficiency"`
	Status         string    `json:"status"`
}

type WeekSummary struct {
	WeekStart      time.Time `json:"week_start"`
	WeekNumber     int       `json:"week_number"`
	Hours          float64   `json:"hours"`
	ClaudeSessions int       `json:"claude_sessions"`
}

type SessionSummary struct {
	TotalSessions   int           `json:"total_sessions"`
	AverageSession  time.Duration `json:"average_session"`
	LongestSession  time.Duration `json:"longest_session"`
	ShortestSession time.Duration `json:"shortest_session"`
	SessionRange    string        `json:"session_range"`
}

type ClaudeActivity struct {
	TotalPrompts      int           `json:"total_prompts"`
	ProcessingTime    time.Duration `json:"processing_time"`
	AverageProcessing time.Duration `json:"average_processing"`
	SuccessfulPrompts int           `json:"successful_prompts"`
	EfficiencyPercent float64       `json:"efficiency_percent"`
}

type WeeklyInsight struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type Trend struct {
	Metric    string  `json:"metric"`
	Direction string  `json:"direction"`
	Change    float64 `json:"change"`
	Period    string  `json:"period"`
	Icon      string  `json:"icon"`
}

type WeeklyStats struct {
	ConsistencyScore  float64 `json:"consistency_score"`
	ProductivityPeak  string  `json:"productivity_peak"`
	WeekendWork       float64 `json:"weekend_work"`
	WeekendPercent    float64 `json:"weekend_percent"`
}

type MonthlyStats struct {
	WorkingDays        int     `json:"working_days"`
	TotalSessions      int     `json:"total_sessions"`
	AvgSessionsPerDay  float64 `json:"avg_sessions_per_day"`
	ConsistencyScore   float64 `json:"consistency_score"`
	MostProductiveWeek int     `json:"most_productive_week"`
	ClaudePrompts      int     `json:"claude_prompts"`
}

type Achievement struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Achieved    bool   `json:"achieved"`
}

type WorkBlockSummary struct {
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	ProjectName string        `json:"project_name"`
	Activities  int           `json:"activities"`
}