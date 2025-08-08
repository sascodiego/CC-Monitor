/**
 * CONTEXT:   Analytics calculator for Claude Monitor reporting system
 * INPUT:     Report data structures requiring mathematical analysis and insights
 * OUTPUT:    Enhanced reports with calculated metrics, trends, and insights
 * BUSINESS:  Analytics calculations provide valuable productivity insights and patterns
 * CHANGE:    Extracted from sqlite_reporting_service.go for Single Responsibility Principle
 * RISK:      Low - Pure mathematical calculations with no side effects
 */

package reporting

import (
	"fmt"
	"math"
	"sort"
	"time"
)

/**
 * CONTEXT:   Analytics calculator with focused mathematical responsibility
 * INPUT:     No dependencies - pure calculation logic
 * OUTPUT:    Analytics calculation capability
 * BUSINESS:  Focused calculator enables clean analytics computation
 * CHANGE:    Initial analytics calculator extracted for SRP compliance
 * RISK:      Low - Pure calculation logic with no external dependencies
 */
type DefaultAnalyticsCalculator struct{}

/**
 * CONTEXT:   Constructor for analytics calculator
 * INPUT:     No dependencies required for pure calculations
 * OUTPUT:    Configured analytics calculator ready for use
 * BUSINESS:  Constructor enables clean instantiation
 * CHANGE:    Initial constructor with no dependencies
 * RISK:      Low - Simple constructor for stateless calculator
 */
func NewDefaultAnalyticsCalculator() *DefaultAnalyticsCalculator {
	return &DefaultAnalyticsCalculator{}
}

/**
 * CONTEXT:   Calculate project breakdown percentages and statistics
 * INPUT:     Enhanced daily report with project data
 * OUTPUT:    Updated report with project breakdown calculations
 * BUSINESS:  Project breakdown shows time distribution across projects
 * CHANGE:    Extracted project breakdown calculation
 * RISK:      Low - Mathematical calculation with percentage computation
 */
func (calc *DefaultAnalyticsCalculator) CalculateProjectBreakdown(report *EnhancedDailyReport) {
	if report.TotalWorkHours == 0 {
		return
	}

	// Calculate percentages for each project
	for i := range report.ProjectBreakdown {
		project := &report.ProjectBreakdown[i]
		project.Percentage = (project.WorkHours / report.TotalWorkHours) * 100
	}

	// Sort projects by work hours
	sort.Slice(report.ProjectBreakdown, func(i, j int) bool {
		return report.ProjectBreakdown[i].WorkHours > report.ProjectBreakdown[j].WorkHours
	})
}

/**
 * CONTEXT:   Calculate hourly breakdown for daily productivity patterns
 * INPUT:     Enhanced daily report with work blocks
 * OUTPUT:    Updated report with hourly activity distribution
 * BUSINESS:  Hourly breakdown reveals productivity patterns throughout the day
 * CHANGE:    Extracted hourly breakdown calculation
 * RISK:      Low - Time-based aggregation with hour buckets
 */
func (calc *DefaultAnalyticsCalculator) CalculateHourlyBreakdown(report *EnhancedDailyReport) {
	hourlyMap := make(map[int]float64)
	
	// Process each work block
	for _, workBlock := range report.WorkBlocks {
		startHour := workBlock.StartTime.Hour()
		endHour := workBlock.EndTime.Hour()
		
		// Handle work blocks that span multiple hours
		for hour := startHour; hour <= endHour; hour++ {
			var duration float64
			
			if hour == startHour && hour == endHour {
				duration = workBlock.Duration.Hours()
			} else if hour == startHour {
				nextHour := time.Date(workBlock.StartTime.Year(), workBlock.StartTime.Month(), 
					workBlock.StartTime.Day(), hour+1, 0, 0, 0, workBlock.StartTime.Location())
				duration = nextHour.Sub(workBlock.StartTime).Hours()
			} else if hour == endHour {
				hourStart := time.Date(workBlock.EndTime.Year(), workBlock.EndTime.Month(),
					workBlock.EndTime.Day(), hour, 0, 0, 0, workBlock.EndTime.Location())
				duration = workBlock.EndTime.Sub(hourStart).Hours()
			} else {
				duration = 1.0
			}
			
			hourlyMap[hour] += duration
		}
	}
	
	// Convert to slice and sort
	report.HourlyBreakdown = make([]HourlyData, 0, len(hourlyMap))
	for hour, hours := range hourlyMap {
		report.HourlyBreakdown = append(report.HourlyBreakdown, HourlyData{
			Hour:  hour,
			Hours: hours,
		})
	}
	
	sort.Slice(report.HourlyBreakdown, func(i, j int) bool {
		return report.HourlyBreakdown[i].Hour < report.HourlyBreakdown[j].Hour
	})
}

/**
 * CONTEXT:   Generate insights for daily productivity patterns
 * INPUT:     Enhanced daily report with calculated data
 * OUTPUT:    Updated report with generated insights
 * BUSINESS:  Daily insights provide actionable productivity feedback
 * CHANGE:    Extracted daily insight generation
 * RISK:      Low - Text generation based on calculated data
 */
func (calc *DefaultAnalyticsCalculator) GenerateDailyInsights(report *EnhancedDailyReport) {
	report.Insights = make([]string, 0)

	// Work hours insights
	report.Insights = append(report.Insights, calc.getWorkHoursInsight(report.TotalWorkHours))

	// Project focus insights
	if len(report.ProjectBreakdown) > 3 {
		report.Insights = append(report.Insights, "🔀 Consider focusing on fewer projects for deeper progress")
	} else if len(report.ProjectBreakdown) == 1 {
		report.Insights = append(report.Insights, "🎯 Excellent focus on a single project today!")
	}

	// Peak productivity insights
	if len(report.HourlyBreakdown) > 0 {
		peakHour := calc.findPeakHour(report.HourlyBreakdown)
		if peakHour >= 0 {
			report.Insights = append(report.Insights, 
				fmt.Sprintf("⏰ Peak productivity at %d:00", peakHour))
		}
	}
}

/**
 * CONTEXT:   Generate insights for weekly productivity patterns
 * INPUT:     Enhanced weekly report with aggregated data
 * OUTPUT:    Updated report with weekly insights
 * BUSINESS:  Weekly insights provide pattern recognition and improvement suggestions
 * CHANGE:    Extracted weekly insight generation
 * RISK:      Low - Analysis of weekly patterns with text generation
 */
func (calc *DefaultAnalyticsCalculator) GenerateWeeklyInsights(report *EnhancedWeeklyReport) {
	report.Insights = make([]WeeklyInsight, 0)

	// Overall productivity insight
	avgHours := report.DailyAverage
	insightType := "productivity"
	message := fmt.Sprintf("Weekly average: %.1f hours/day", avgHours)
	
	if avgHours < 4 {
		insightType = "improvement"
		message += " - aim for consistency"
	}

	report.Insights = append(report.Insights, WeeklyInsight{
		Type:    insightType,
		Message: message,
	})

	// Consistency analysis
	consistency := calc.calculateConsistency(report)
	if consistency < 0.6 {
		report.Insights = append(report.Insights, WeeklyInsight{
			Type:    "pattern",
			Message: "Work pattern varies - consider establishing routine",
		})
	}
}

/**
 * CONTEXT:   Generate trend analysis for weekly reports
 * INPUT:     Enhanced weekly report for trend calculation
 * OUTPUT:    Updated report with trend data
 * BUSINESS:  Weekly trends show productivity patterns and momentum
 * CHANGE:    Extracted weekly trend generation
 * RISK:      Low - Trend calculation based on daily data
 */
func (calc *DefaultAnalyticsCalculator) GenerateWeeklyTrends(report *EnhancedWeeklyReport) {
	report.Trends = make([]Trend, 0)

	if len(report.DailyBreakdown) >= 6 {
		firstHalf := calc.avgHours(report.DailyBreakdown[:3])
		secondHalf := calc.avgHours(report.DailyBreakdown[3:6])
		
		trendType := "stable"
		description := "Consistent work pattern"
		value := 0.0
		
		if secondHalf > firstHalf*1.2 {
			trendType = "increasing"
			description = "Work hours increasing"
			value = secondHalf - firstHalf
		} else if firstHalf > secondHalf*1.2 {
			trendType = "decreasing"
			description = "Work hours decreasing"
			value = firstHalf - secondHalf
		}

		report.Trends = append(report.Trends, Trend{
			Type:        trendType,
			Description: description,
			Value:       value,
		})
	}
}

// Interface compliance methods (placeholder implementations)
func (calc *DefaultAnalyticsCalculator) GenerateMonthlyAchievements(report *EnhancedMonthlyReport) {}
func (calc *DefaultAnalyticsCalculator) GenerateMonthlyTrends(report *EnhancedMonthlyReport)        {}
func (calc *DefaultAnalyticsCalculator) GenerateMonthlyInsights(report *EnhancedMonthlyReport)     {}

// Helper functions
func (calc *DefaultAnalyticsCalculator) getWorkHoursInsight(hours float64) string {
	if hours >= 8 {
		return "🌟 Excellent productivity with 8+ hours"
	} else if hours >= 6 {
		return "✅ Good productivity day"
	} else if hours >= 3 {
		return "⚡ Moderate work day"
	} else if hours > 0 {
		return "🚀 Getting started"
	}
	return "📋 No tracked work today"
}

func (calc *DefaultAnalyticsCalculator) findPeakHour(hourlyData []HourlyData) int {
	if len(hourlyData) == 0 {
		return -1
	}
	
	peakHour := hourlyData[0].Hour
	maxHours := hourlyData[0].Hours
	
	for _, data := range hourlyData {
		if data.Hours > maxHours {
			maxHours = data.Hours
			peakHour = data.Hour
		}
	}
	
	return peakHour
}

func (calc *DefaultAnalyticsCalculator) calculateConsistency(report *EnhancedWeeklyReport) float64 {
	if len(report.DailyBreakdown) == 0 {
		return 0
	}
	
	mean := report.DailyAverage
	variance := 0.0
	
	for _, day := range report.DailyBreakdown {
		diff := day.Hours - mean
		variance += diff * diff
	}
	
	variance /= float64(len(report.DailyBreakdown))
	stdDev := math.Sqrt(variance)
	
	if mean == 0 {
		return 0
	}
	
	cv := stdDev / mean
	return math.Max(0, 1-cv)
}

func (calc *DefaultAnalyticsCalculator) avgHours(days []DaySummary) float64 {
	if len(days) == 0 {
		return 0
	}
	
	total := 0.0
	for _, day := range days {
		total += day.Hours
	}
	
	return total / float64(len(days))
}