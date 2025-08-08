/**
 * CONTEXT:   Recommendation generation functions for work analytics insights
 * INPUT:     Analysis results requiring actionable productivity recommendations
 * OUTPUT:    Tailored recommendations for improving work patterns and productivity
 * BUSINESS:  Recommendations translate analytical insights into actionable user guidance
 * CHANGE:    Extracted recommendation logic from monolithic work_analytics_engine.go
 * RISK:      Low - Recommendation generation with established business rules
 */

package reporting

import (
	"fmt"
	"time"
)

/**
 * CONTEXT:   Generate deep work recommendations based on analysis results
 * INPUT:     Deep work analysis with focus metrics and work patterns
 * OUTPUT:    Array of actionable recommendations for improving deep work quality
 * BUSINESS:  Deep work recommendations help users optimize focus and minimize distractions
 * CHANGE:    Extracted from core engine, maintains recommendation logic
 * RISK:      Low - Recommendation generation with focus improvement strategies
 */
func (wae *WorkAnalyticsEngine) generateDeepWorkRecommendations(analysis *DeepWorkAnalysis) []string {
	recommendations := make([]string, 0)
	
	if analysis.DeepWorkPercentage < 30 {
		recommendations = append(recommendations, "Focus on 25+ minute work blocks for deep work")
		recommendations = append(recommendations, "Consider reducing distractions during work sessions")
	}
	
	if analysis.ContextSwitches > 8 {
		recommendations = append(recommendations, "High context switching detected - try time blocking by project")
	}
	
	if analysis.FragmentationScore > 0.5 {
		recommendations = append(recommendations, "Work is highly fragmented - consider longer uninterrupted sessions")
	}
	
	if len(analysis.FlowSessions) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Great! %d flow sessions detected - replicate these conditions", len(analysis.FlowSessions)))
	} else if analysis.DeepWorkTime > 2*time.Hour {
		recommendations = append(recommendations, "You have good deep work time - try extending sessions to achieve flow state")
	}
	
	if analysis.FocusScore < 50 {
		recommendations = append(recommendations, "Focus score is low - consider implementing the Pomodoro Technique")
		recommendations = append(recommendations, "Create a dedicated workspace to minimize interruptions")
	} else if analysis.FocusScore >= 80 {
		recommendations = append(recommendations, "Excellent focus score! Maintain your current work patterns")
	}
	
	return recommendations
}

/**
 * CONTEXT:   Generate activity pattern recommendations based on behavioral analysis
 * INPUT:     Activity pattern analysis with work rhythms and consistency metrics
 * OUTPUT:    Array of recommendations for optimizing work scheduling and habits
 * BUSINESS:  Activity pattern recommendations help users align work with natural rhythms
 * CHANGE:    Extracted from core engine, maintains pattern-based recommendation logic
 * RISK:      Low - Recommendation generation with scheduling optimization strategies
 */
func (wae *WorkAnalyticsEngine) generateActivityPatternRecommendations(analysis *ActivityPatternAnalysis) []string {
	recommendations := make([]string, 0)
	
	switch analysis.WorkRhythm {
	case "early_bird":
		recommendations = append(recommendations, "Your peak is morning - schedule important work before 10 AM")
		recommendations = append(recommendations, "Protect your morning hours for deep work and complex tasks")
	case "night_owl":
		recommendations = append(recommendations, "Evening productivity detected - protect your late-day focus time")
		recommendations = append(recommendations, "Schedule meetings and administrative tasks earlier in the day")
	case "mid_day":
		recommendations = append(recommendations, "Mid-day productivity peak - use morning for preparation and afternoon for execution")
	case "afternoon":
		recommendations = append(recommendations, "Afternoon peak detected - save complex work for post-lunch hours")
	case "distributed":
		recommendations = append(recommendations, "Distributed work pattern - good for varied task scheduling")
		recommendations = append(recommendations, "Consider time-blocking different types of work throughout the day")
	case "irregular":
		recommendations = append(recommendations, "Irregular work pattern - try establishing consistent work hours")
		recommendations = append(recommendations, "Experiment with different times to find your natural peak hours")
	}
	
	if analysis.ConsistencyScore < 50 {
		recommendations = append(recommendations, "Inconsistent work pattern - try establishing regular work hours")
		recommendations = append(recommendations, "Consider creating a daily work routine to build consistency")
	} else if analysis.ConsistencyScore >= 80 {
		recommendations = append(recommendations, "Excellent work consistency! Your routine is well-established")
	}
	
	if len(analysis.PeakHours) > 6 {
		recommendations = append(recommendations, "Work scattered across many hours - consider focusing on 3-4 peak hours")
		recommendations = append(recommendations, "Try batching similar activities within your most productive hours")
	}
	
	if analysis.TotalActivities < 20 {
		recommendations = append(recommendations, "Low activity detected - consider if you're capturing all work sessions")
	} else if analysis.TotalActivities > 200 {
		recommendations = append(recommendations, "High activity volume - ensure you're taking adequate breaks")
	}
	
	return recommendations
}

/**
 * CONTEXT:   Generate project focus recommendations for context switching optimization
 * INPUT:     Project focus analysis with switching costs and efficiency metrics
 * OUTPUT:    Array of recommendations for minimizing context switching and improving focus
 * BUSINESS:  Project focus recommendations help reduce cognitive switching costs
 * CHANGE:    Extracted from core engine, maintains focus efficiency recommendation logic
 * RISK:      Low - Recommendation generation with context switching mitigation strategies
 */
func (wae *WorkAnalyticsEngine) generateProjectFocusRecommendations(analysis *ProjectFocusAnalysis) []string {
	recommendations := make([]string, 0)
	
	if analysis.ContextSwitches > 5 {
		recommendations = append(recommendations, "High project switching - consider batching similar work")
		recommendations = append(recommendations, "Try dedicating specific days or time blocks to individual projects")
	}
	
	if analysis.FocusEfficiency < 70 {
		recommendations = append(recommendations, "Focus efficiency below 70% - reduce interruptions between projects")
		recommendations = append(recommendations, "Consider longer work blocks to minimize switching overhead")
	} else if analysis.FocusEfficiency >= 85 {
		recommendations = append(recommendations, "Excellent focus efficiency! Your project management is working well")
	}
	
	if analysis.SwitchingCost > 30*time.Minute {
		recommendations = append(recommendations, "High switching cost detected - minimize project transitions")
		recommendations = append(recommendations, "Plan project switches during natural break points")
	}
	
	if len(analysis.ProjectSessions) > 10 {
		recommendations = append(recommendations, "Many short project sessions - consider consolidating related work")
	}
	
	// Analyze project session patterns
	if len(analysis.ProjectSessions) > 0 {
		totalSessions := len(analysis.ProjectSessions)
		var totalDuration time.Duration
		
		for _, session := range analysis.ProjectSessions {
			totalDuration += session.Duration
		}
		
		avgSessionDuration := totalDuration / time.Duration(totalSessions)
		
		if avgSessionDuration < 30*time.Minute {
			recommendations = append(recommendations, "Average project sessions are short - try extending focus periods")
		} else if avgSessionDuration >= 2*time.Hour {
			recommendations = append(recommendations, "Great! Long project sessions indicate strong focus capability")
		}
	}
	
	return recommendations
}