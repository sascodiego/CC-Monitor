/**
 * CONTEXT:   Data structures and types for work analytics system
 * INPUT:     Analytics results requiring structured data representation
 * OUTPUT:    Comprehensive type definitions for analysis results and metrics
 * BUSINESS:  Structured data types ensure consistent analytics result representation
 * CHANGE:    Extracted data structures from monolithic work_analytics_engine.go
 * RISK:      Low - Type definitions with clear field documentation
 */

package reporting

import (
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

// DeepWorkAnalysis contains comprehensive deep work metrics and insights
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

// FocusBlock represents a single work block with focus analysis
type FocusBlock struct {
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	ProjectName  string        `json:"project_name"`
	ActivityRate float64       `json:"activity_rate"`
	FocusLevel   FocusLevel    `json:"focus_level"`
	FlowState    bool          `json:"flow_state"`
}

// FlowSession represents a continuous period of high-focus work
type FlowSession struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Quality   float64       `json:"quality"`
	Blocks    []FocusBlock  `json:"blocks"`
}

// FocusLevel represents the intensity of focus during a work block
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

// ActivityPatternAnalysis contains work rhythm and behavioral insights
type ActivityPatternAnalysis struct {
	TotalActivities    int                   `json:"total_activities"`
	HourlyDistribution []HourlyActivityData  `json:"hourly_distribution"`
	PeakHours          []int                 `json:"peak_hours"`
	ActivityTypes      map[string]int        `json:"activity_types"`
	WorkRhythm         string                `json:"work_rhythm"`
	ConsistencyScore   float64               `json:"consistency_score"`
	Recommendations    []string              `json:"recommendations"`
}

// HourlyActivityData represents activity metrics for a specific hour
type HourlyActivityData struct {
	Hour        int     `json:"hour"`
	Activities  int     `json:"activities"`
	Intensity   float64 `json:"intensity"`
	Consistency float64 `json:"consistency"`
}

// ProjectFocusAnalysis contains context switching and project focus metrics
type ProjectFocusAnalysis struct {
	ProjectSessions []ProjectSession `json:"project_sessions"`
	ContextSwitches int              `json:"context_switches"`
	SwitchingCost   time.Duration    `json:"switching_cost"`
	FocusEfficiency float64          `json:"focus_efficiency"`
	Recommendations []string         `json:"recommendations"`
}

// ProjectSession represents continuous work on a single project
type ProjectSession struct {
	ProjectID   string                `json:"project_id"`
	ProjectName string                `json:"project_name"`
	StartTime   time.Time             `json:"start_time"`
	EndTime     time.Time             `json:"end_time"`
	Duration    time.Duration         `json:"duration"`
	WorkBlocks  []*sqlite.WorkBlock   `json:"work_blocks"`
}