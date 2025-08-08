/**
 * CONTEXT:   Classification and detection logic for work analytics patterns
 * INPUT:     Work blocks, activity patterns, and behavioral data for pattern detection
 * OUTPUT:    Classified focus levels, detected states, and work rhythm patterns
 * BUSINESS:  Pattern detection drives productivity insights and behavioral analysis
 * CHANGE:    Extracted detection logic from monolithic work_analytics_engine.go
 * RISK:      Low - Classification algorithms with established business rules
 */

package reporting

import (
	"math"
	"sort"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

/**
 * CONTEXT:   Classify work block focus level based on duration and activity patterns
 * INPUT:     Work block duration and activity count for focus level calculation
 * OUTPUT:    Focus level classification (distracted, focused, deep, or flow)
 * BUSINESS:  Focus level classification drives productivity insights and recommendations
 * CHANGE:    Extracted from core engine, maintains classification logic
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
 * CHANGE:    Extracted from core engine, maintains flow detection criteria
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
 * CONTEXT:   Determine work rhythm pattern from peak activity hours
 * INPUT:     Array of peak hours for work rhythm classification
 * OUTPUT:    Work rhythm classification string (early_bird, night_owl, etc.)
 * BUSINESS:  Work rhythm helps users optimize their scheduling for peak productivity
 * CHANGE:    Extracted from core engine, maintains rhythm classification logic
 * RISK:      Low - Pattern classification with time-based business rules
 */
func (wae *WorkAnalyticsEngine) determineWorkRhythm(peakHours []int) string {
	if len(peakHours) == 0 {
		return "irregular"
	}
	
	// Sort peak hours for analysis
	sortedPeaks := make([]int, len(peakHours))
	copy(sortedPeaks, peakHours)
	sort.Ints(sortedPeaks)
	
	// Analyze peak hour distribution
	earlyHours := 0  // 6-10 AM
	midHours := 0    // 10 AM - 2 PM
	lateHours := 0   // 2-6 PM
	eveningHours := 0 // 6-10 PM
	
	for _, hour := range sortedPeaks {
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