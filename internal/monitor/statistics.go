/**
 * CONTEXT:   Advanced statistics system for comprehensive Claude monitoring analytics
 * INPUT:     Events from all monitoring subsystems (process, file I/O, HTTP, network)
 * OUTPUT:    Comprehensive analytics and insights for Claude usage patterns
 * BUSINESS:  Advanced statistics enable detailed Claude usage analysis and optimization
 * CHANGE:    Initial advanced statistics based on prototype with multi-dimensional analysis
 * RISK:      Medium - Statistics collection affecting monitoring performance
 */

package monitor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

/**
 * CONTEXT:   Comprehensive statistics aggregator for all monitoring systems
 * INPUT:     Events from process, file I/O, HTTP, and network monitors
 * OUTPUT:    Aggregated statistics and analytics insights
 * BUSINESS:  Statistics aggregation provides holistic Claude usage analytics
 * CHANGE:    Initial comprehensive statistics system
 * RISK:      Medium - Statistics aggregation affecting system performance
 */
type ComprehensiveStats struct {
	mu                    sync.RWMutex
	startTime            time.Time
	processStats         ProcessMonitorStats
	fileIOStats          FileIOStats
	httpStats            HTTPStats
	networkStats         NetworkStats
	activityStats        ActivityGeneratorStats
	eventCounts          map[string]uint64
	projectStats         map[string]*ProjectStats
	timelineStats        []TimelineEntry
	performanceMetrics   PerformanceMetrics
	prototypeStats       PrototypeCompatStats
	insightEngine       *InsightEngine
	config              StatsConfig
}

/**
 * CONTEXT:   Project-specific statistics tracking
 * INPUT:     Project-based event aggregation
 * OUTPUT:    Per-project analytics and insights
 * BUSINESS:  Project statistics enable project-specific productivity analysis
 * CHANGE:    Initial project statistics tracking
 * RISK:      Low - Project statistics data structure
 */
type ProjectStats struct {
	ProjectName        string            `json:"project_name"`
	ProjectPath        string            `json:"project_path"`
	ProcessCount       int               `json:"process_count"`
	FileOperations     uint64            `json:"file_operations"`
	HTTPRequests       uint64            `json:"http_requests"`
	NetworkConnections uint64            `json:"network_connections"`
	WorkActivities     uint64            `json:"work_activities"`
	TotalWorkTime      time.Duration     `json:"total_work_time"`
	LastActivity       time.Time         `json:"last_activity"`
	FileTypes          map[string]uint64 `json:"file_types"`
	APIEndpoints       map[string]uint64 `json:"api_endpoints"`
	ProductivityScore  float64           `json:"productivity_score"`
}

/**
 * CONTEXT:   Timeline entry for temporal analysis
 * INPUT:     Time-based event tracking
 * OUTPUT:    Temporal analytics data point
 * BUSINESS:  Timeline tracking enables temporal usage pattern analysis
 * CHANGE:    Initial timeline statistics tracking
 * RISK:      Low - Timeline data structure
 */
type TimelineEntry struct {
	Timestamp      time.Time `json:"timestamp"`
	ProcessEvents  uint64    `json:"process_events"`
	FileIOEvents   uint64    `json:"file_io_events"`
	HTTPEvents     uint64    `json:"http_events"`
	NetworkEvents  uint64    `json:"network_events"`
	WorkActivities uint64    `json:"work_activities"`
	ActiveProjects int       `json:"active_projects"`
}

/**
 * CONTEXT:   Performance metrics for monitoring system health
 * INPUT:     System performance monitoring data
 * OUTPUT:    Performance analytics and optimization insights
 * BUSINESS:  Performance metrics enable monitoring system optimization
 * CHANGE:    Initial performance metrics tracking
 * RISK:      Low - Performance metrics data structure
 */
type PerformanceMetrics struct {
	CPUUsage           float64           `json:"cpu_usage"`
	MemoryUsage        float64           `json:"memory_usage"`
	EventProcessingRate float64          `json:"event_processing_rate"`
	AverageEventLatency time.Duration    `json:"average_event_latency"`
	SystemLoad         float64           `json:"system_load"`
	ErrorRate          float64           `json:"error_rate"`
	Bottlenecks        []string          `json:"bottlenecks"`
}

/**
 * CONTEXT:   Statistics configuration
 * INPUT:     Statistics collection and analysis parameters
 * OUTPUT:    Statistics system behavior configuration
 * BUSINESS:  Configuration enables customizable statistics collection
 * CHANGE:    Initial statistics configuration
 * RISK:      Low - Configuration structure for statistics system
 */
type StatsConfig struct {
	CollectionInterval  time.Duration `json:"collection_interval"`  // How often to collect stats
	RetentionPeriod    time.Duration `json:"retention_period"`     // How long to keep detailed stats
	TimelineResolution time.Duration `json:"timeline_resolution"`  // Timeline granularity
	EnableInsights     bool          `json:"enable_insights"`      // Enable insight generation
	EnablePerformance  bool          `json:"enable_performance"`   // Enable performance monitoring
	VerboseLogging     bool          `json:"verbose_logging"`      // Enable verbose logging
}

/**
 * CONTEXT:   Insight engine for generating analytics insights
 * INPUT:     Aggregated statistics data
 * OUTPUT:    Generated insights and recommendations
 * BUSINESS:  Insight engine provides actionable analytics insights
 * CHANGE:    Initial insight engine implementation
 * RISK:      Medium - Insight generation affecting system performance
 */
type InsightEngine struct {
	insights           []Insight
	insightGenerators []InsightGenerator
	mu                sync.RWMutex
}

type Insight struct {
	Type        string                 `json:"type"`        // productivity, performance, usage, etc.
	Category    string                 `json:"category"`    // warning, info, recommendation
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Impact      string                 `json:"impact"`      // high, medium, low
	Suggestions []string               `json:"suggestions"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
}

type InsightGenerator func(*ComprehensiveStats) []Insight

/**
 * CONTEXT:   Prototype-compatible statistics for backward compatibility
 * INPUT:     Statistics from prototype-style event processing
 * OUTPUT:    Statistics matching prototype format and calculations
 * BUSINESS:  Prototype compatibility enables migration and comparison
 * CHANGE:    Prototype statistics compatibility layer from process-check-prototype.go
 * RISK:      Low - Compatibility layer for statistics
 */
type PrototypeCompatStats struct {
	EventCountsByType    map[string]int `json:"event_counts_by_type"`    // File read/write/open, HTTP POST, etc.
	TotalBytesRead       int64          `json:"total_bytes_read"`        // Total bytes read from files
	TotalBytesWrite      int64          `json:"total_bytes_write"`       // Total bytes written to files
	HTTPRequestCount     int            `json:"http_request_count"`      // Total HTTP requests
	HTTPPostCount        int            `json:"http_post_count"`         // HTTP POST requests (work indicators)
	ClaudeAPIPostCount   int            `json:"claude_api_post_count"`   // Claude API POST requests
	ProcessStartCount    int            `json:"process_start_count"`     // Process start events
	ProcessStopCount     int            `json:"process_stop_count"`      // Process stop events
	NetworkConnections   int            `json:"network_connections"`     // Network connection count
	WorkFileOperations   int            `json:"work_file_operations"`    // Work-related file operations
	MonitoringDuration   time.Duration  `json:"monitoring_duration"`     // Total monitoring time
	StartTime            time.Time      `json:"start_time"`              // Statistics start time
}

/**
 * CONTEXT:   Create new comprehensive statistics system
 * INPUT:     Statistics configuration
 * OUTPUT:    Configured comprehensive statistics system
 * BUSINESS:  Statistics system creation enables comprehensive Claude analytics
 * CHANGE:    Initial comprehensive statistics constructor
 * RISK:      Medium - Statistics system initialization affecting monitoring
 */
func NewComprehensiveStats(config StatsConfig) *ComprehensiveStats {
	// Set default configuration
	if config.CollectionInterval == 0 {
		config.CollectionInterval = 30 * time.Second
	}
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 24 * time.Hour
	}
	if config.TimelineResolution == 0 {
		config.TimelineResolution = 5 * time.Minute
	}
	
	stats := &ComprehensiveStats{
		startTime:     time.Now(),
		eventCounts:   make(map[string]uint64),
		projectStats:  make(map[string]*ProjectStats),
		timelineStats: make([]TimelineEntry, 0),
		config:        config,
		prototypeStats: PrototypeCompatStats{
			EventCountsByType: make(map[string]int),
			StartTime:        time.Now(),
		},
	}
	
	// Initialize insight engine if enabled
	if config.EnableInsights {
		stats.insightEngine = &InsightEngine{
			insights:          make([]Insight, 0),
			insightGenerators: stats.createInsightGenerators(),
		}
	}
	
	return stats
}

/**
 * CONTEXT:   Update statistics with process monitor data
 * INPUT:     Process monitor statistics
 * OUTPUT:    Updated comprehensive statistics
 * BUSINESS:  Process statistics update enables process-based analytics
 * CHANGE:    Initial process statistics integration
 * RISK:      Low - Statistics update for process data
 */
func (cs *ComprehensiveStats) UpdateProcessStats(stats MonitorStats) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Convert MonitorStats to ProcessMonitorStats
	cs.processStats = ProcessMonitorStats{
		ScansPerformed:     stats.ScansPerformed,
		ProcessesDetected:  stats.ProcessesDetected,
		EventsGenerated:    stats.EventsGenerated,
		LastScanTime:      stats.LastScanTime,
		StartTime:         stats.StartTime,
		AverageScanTime:   stats.AverageScanTime,
		ErrorCount:        stats.ErrorCount,
	}
	
	cs.eventCounts["process_events"] = stats.EventsGenerated
	cs.updateTimeline()
}

/**
 * CONTEXT:   Update statistics with file I/O monitor data
 * INPUT:     File I/O monitor statistics
 * OUTPUT:    Updated comprehensive statistics
 * BUSINESS:  File I/O statistics update enables file-based analytics
 * CHANGE:    Initial file I/O statistics integration
 * RISK:      Low - Statistics update for file I/O data
 */
func (cs *ComprehensiveStats) UpdateFileIOStats(stats FileIOStats) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.fileIOStats = stats
	cs.eventCounts["file_io_events"] = stats.TotalEvents
	
	// Update project statistics with file I/O data
	for project, count := range stats.EventsByProject {
		cs.updateProjectStats(project, "file_operations", count)
	}
	
	cs.updateTimeline()
}

/**
 * CONTEXT:   Update statistics with HTTP monitor data
 * INPUT:     HTTP monitor statistics
 * OUTPUT:    Updated comprehensive statistics
 * BUSINESS:  HTTP statistics update enables HTTP/API analytics
 * CHANGE:    Initial HTTP statistics integration
 * RISK:      Low - Statistics update for HTTP data
 */
func (cs *ComprehensiveStats) UpdateHTTPStats(stats HTTPStats) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.httpStats = stats
	cs.eventCounts["http_events"] = stats.TotalRequests
	
	// Update project statistics with HTTP data
	for host, count := range stats.RequestsByHost {
		if cs.isClaudeEndpoint(host) {
			cs.eventCounts["claude_api_requests"] += count
		}
	}
	
	cs.updateTimeline()
}

/**
 * CONTEXT:   Update statistics with network monitor data
 * INPUT:     Network monitor statistics
 * OUTPUT:    Updated comprehensive statistics
 * BUSINESS:  Network statistics update enables connectivity analytics
 * CHANGE:    Initial network statistics integration
 * RISK:      Low - Statistics update for network data
 */
func (cs *ComprehensiveStats) UpdateNetworkStats(stats NetworkStats) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.networkStats = stats
	cs.eventCounts["network_events"] = stats.TotalConnections
	
	cs.updateTimeline()
}

/**
 * CONTEXT:   Update statistics with activity generator data
 * INPUT:     Activity generator statistics
 * OUTPUT:    Updated comprehensive statistics
 * BUSINESS:  Activity statistics update enables work productivity analytics
 * CHANGE:    Initial activity statistics integration
 * RISK:      Low - Statistics update for activity data
 */
func (cs *ComprehensiveStats) UpdateActivityStats(stats ActivityGeneratorStats) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	cs.activityStats = stats
	cs.eventCounts["work_activities"] = stats.WorkActivities
	
	cs.updateTimeline()
}

/**
 * CONTEXT:   Update prototype-compatible statistics with event data
 * INPUT:     Event type and event details for prototype statistics update
 * OUTPUT:    Updated prototype-compatible statistics
 * BUSINESS:  Prototype statistics update maintains compatibility with original monitoring
 * CHANGE:    Prototype statistics integration from process-check-prototype.go
 * RISK:      Low - Compatibility statistics update
 */
func (cs *ComprehensiveStats) UpdatePrototypeEvent(eventType string, details map[string]interface{}) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Update event counts by type (prototype-style)
	cs.prototypeStats.EventCountsByType[eventType]++
	cs.prototypeStats.MonitoringDuration = time.Since(cs.prototypeStats.StartTime)
	
	switch eventType {
	case "FILE_READ":
		if bytesRead, ok := details["size"].(int64); ok {
			cs.prototypeStats.TotalBytesRead += bytesRead
		} else if bytesRead, ok := details["bytes_delta"].(int64); ok {
			cs.prototypeStats.TotalBytesRead += bytesRead
		}
		if isWorkFile, ok := details["is_work_file"].(bool); ok && isWorkFile {
			cs.prototypeStats.WorkFileOperations++
		}
		
	case "FILE_WRITE":
		if bytesWrite, ok := details["size"].(int64); ok {
			cs.prototypeStats.TotalBytesWrite += bytesWrite
		} else if bytesWrite, ok := details["bytes_delta"].(int64); ok {
			cs.prototypeStats.TotalBytesWrite += bytesWrite
		}
		if isWorkFile, ok := details["is_work_file"].(bool); ok && isWorkFile {
			cs.prototypeStats.WorkFileOperations++
		}
		
	case "FILE_OPEN":
		if isWorkFile, ok := details["is_work_file"].(bool); ok && isWorkFile {
			cs.prototypeStats.WorkFileOperations++
		}
		
	case "HTTP_POST":
		cs.prototypeStats.HTTPPostCount++
		cs.prototypeStats.HTTPRequestCount++
		if isClaudeAPI, ok := details["is_claude_api"].(bool); ok && isClaudeAPI {
			cs.prototypeStats.ClaudeAPIPostCount++
		}
		
	case "HTTP_REQUEST":
		cs.prototypeStats.HTTPRequestCount++
		if method, ok := details["method"].(string); ok && method == "POST" {
			cs.prototypeStats.HTTPPostCount++
			if isClaudeAPI, ok := details["is_claude_api"].(bool); ok && isClaudeAPI {
				cs.prototypeStats.ClaudeAPIPostCount++
			}
		}
		
	case "PROCESS_STARTED", "started":
		cs.prototypeStats.ProcessStartCount++
		
	case "PROCESS_STOPPED", "stopped":
		cs.prototypeStats.ProcessStopCount++
		
	case "NET_CONNECT", "NETWORK_CONNECTION":
		cs.prototypeStats.NetworkConnections++
	}
}

/**
 * CONTEXT:   Update project-specific statistics
 * INPUT:     Project name, metric name, and count
 * OUTPUT:    Updated project statistics
 * BUSINESS:  Project statistics enable per-project analytics
 * CHANGE:    Initial project statistics update
 * RISK:      Low - Project statistics update
 */
func (cs *ComprehensiveStats) updateProjectStats(projectName, metric string, count uint64) {
	if _, exists := cs.projectStats[projectName]; !exists {
		cs.projectStats[projectName] = &ProjectStats{
			ProjectName:    projectName,
			FileTypes:      make(map[string]uint64),
			APIEndpoints:   make(map[string]uint64),
			LastActivity:   time.Now(),
		}
	}
	
	project := cs.projectStats[projectName]
	project.LastActivity = time.Now()
	
	switch metric {
	case "file_operations":
		project.FileOperations = count
	case "http_requests":
		project.HTTPRequests = count
	case "network_connections":
		project.NetworkConnections = count
	case "work_activities":
		project.WorkActivities = count
	}
	
	// Update productivity score
	project.ProductivityScore = cs.calculateProductivityScore(project)
}

/**
 * CONTEXT:   Update timeline statistics
 * INPUT:     Current statistics state
 * OUTPUT:    Updated timeline with current data point
 * BUSINESS:  Timeline update enables temporal analysis
 * CHANGE:    Initial timeline statistics update
 * RISK:      Low - Timeline statistics update
 */
func (cs *ComprehensiveStats) updateTimeline() {
	now := time.Now()
	entry := TimelineEntry{
		Timestamp:      now,
		ProcessEvents:  cs.eventCounts["process_events"],
		FileIOEvents:   cs.eventCounts["file_io_events"],
		HTTPEvents:     cs.eventCounts["http_events"],
		NetworkEvents:  cs.eventCounts["network_events"],
		WorkActivities: cs.eventCounts["work_activities"],
		ActiveProjects: len(cs.projectStats),
	}
	
	// Add to timeline
	cs.timelineStats = append(cs.timelineStats, entry)
	
	// Limit timeline size based on retention period
	cutoff := now.Add(-cs.config.RetentionPeriod)
	for i, e := range cs.timelineStats {
		if e.Timestamp.After(cutoff) {
			cs.timelineStats = cs.timelineStats[i:]
			break
		}
	}
}

/**
 * CONTEXT:   Calculate productivity score for project
 * INPUT:     Project statistics data
 * OUTPUT:    Productivity score (0.0 to 1.0)
 * BUSINESS:  Productivity scoring enables project productivity assessment
 * CHANGE:    Initial productivity scoring algorithm
 * RISK:      Low - Productivity calculation utility
 */
func (cs *ComprehensiveStats) calculateProductivityScore(project *ProjectStats) float64 {
	score := 0.0
	
	// File operations contribute to productivity
	if project.FileOperations > 0 {
		score += 0.3 * (float64(project.FileOperations) / 100.0)
	}
	
	// Work activities are most important
	if project.WorkActivities > 0 {
		score += 0.5 * (float64(project.WorkActivities) / 50.0)
	}
	
	// HTTP requests to Claude APIs indicate active usage
	if project.HTTPRequests > 0 {
		score += 0.2 * (float64(project.HTTPRequests) / 20.0)
	}
	
	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

/**
 * CONTEXT:   Generate comprehensive analytics report
 * INPUT:     Statistics aggregation request
 * OUTPUT:    Comprehensive analytics report
 * BUSINESS:  Analytics report provides holistic Claude usage insights
 * CHANGE:    Initial comprehensive analytics report generation
 * RISK:      Low - Report generation utility
 */
func (cs *ComprehensiveStats) GenerateReport() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	report := map[string]interface{}{
		"summary": map[string]interface{}{
			"start_time":         cs.startTime,
			"uptime":            time.Since(cs.startTime),
			"total_events":      cs.getTotalEvents(),
			"active_projects":   len(cs.projectStats),
			"monitoring_health": cs.getMonitoringHealth(),
		},
		"process_monitoring": cs.processStats,
		"file_io_monitoring": cs.fileIOStats,
		"http_monitoring":    cs.httpStats,
		"network_monitoring": cs.networkStats,
		"activity_generation": cs.activityStats,
		"project_analytics":  cs.getTopProjects(10),
		"timeline_data":      cs.getTimelineData(24 * time.Hour),
	}
	
	// Add performance metrics if enabled
	if cs.config.EnablePerformance {
		report["performance_metrics"] = cs.performanceMetrics
	}
	
	// Add insights if enabled
	if cs.config.EnableInsights && cs.insightEngine != nil {
		cs.generateInsights()
		report["insights"] = cs.insightEngine.insights
	}
	
	// Add prototype-compatible statistics
	report["prototype_statistics"] = cs.prototypeStats
	
	return report
}

/**
 * CONTEXT:   Generate analytics insights
 * INPUT:     Comprehensive statistics data
 * OUTPUT:    Generated insights and recommendations
 * BUSINESS:  Insight generation provides actionable analytics recommendations
 * CHANGE:    Initial insight generation implementation
 * RISK:      Low - Insight generation utility
 */
func (cs *ComprehensiveStats) generateInsights() {
	if cs.insightEngine == nil {
		return
	}
	
	cs.insightEngine.mu.Lock()
	defer cs.insightEngine.mu.Unlock()
	
	// Clear old insights
	cs.insightEngine.insights = make([]Insight, 0)
	
	// Run insight generators
	for _, generator := range cs.insightEngine.insightGenerators {
		insights := generator(cs)
		cs.insightEngine.insights = append(cs.insightEngine.insights, insights...)
	}
	
	// Sort insights by impact
	sort.Slice(cs.insightEngine.insights, func(i, j int) bool {
		return cs.insightEngine.insights[i].Impact > cs.insightEngine.insights[j].Impact
	})
}

/**
 * CONTEXT:   Create insight generators for analytics
 * INPUT:     Insight generation configuration
 * OUTPUT:    Array of insight generator functions
 * BUSINESS:  Insight generators provide specialized analytics insights
 * CHANGE:    Initial insight generators creation
 * RISK:      Low - Insight generator functions
 */
func (cs *ComprehensiveStats) createInsightGenerators() []InsightGenerator {
	generators := []InsightGenerator{
		cs.generateProductivityInsights,
		cs.generatePerformanceInsights,
		cs.generateUsageInsights,
		cs.generateOptimizationInsights,
	}
	
	return generators
}

/**
 * CONTEXT:   Generate productivity insights
 * INPUT:     Comprehensive statistics data
 * OUTPUT:    Productivity-related insights
 * BUSINESS:  Productivity insights enable work effectiveness assessment
 * CHANGE:    Initial productivity insight generation
 * RISK:      Low - Productivity insight generation
 */
func (cs *ComprehensiveStats) generateProductivityInsights(stats *ComprehensiveStats) []Insight {
	var insights []Insight
	
	// High productivity project insight
	topProject := cs.getTopProject()
	if topProject != nil && topProject.ProductivityScore > 0.8 {
		insights = append(insights, Insight{
			Type:        "productivity",
			Category:    "info",
			Title:       "High Productivity Project Detected",
			Description: fmt.Sprintf("Project '%s' shows exceptional productivity with score %.2f", topProject.ProjectName, topProject.ProductivityScore),
			Impact:      "high",
			Suggestions: []string{"Continue current workflow", "Consider applying similar patterns to other projects"},
			Data:        map[string]interface{}{"project": topProject.ProjectName, "score": topProject.ProductivityScore},
			Timestamp:   time.Now(),
		})
	}
	
	// Low productivity warning
	if cs.activityStats.WorkActivities == 0 && time.Since(cs.startTime) > time.Hour {
		insights = append(insights, Insight{
			Type:        "productivity",
			Category:    "warning",
			Title:       "No Work Activities Detected",
			Description: "No work activities have been generated despite monitoring activity",
			Impact:      "high",
			Suggestions: []string{"Check if Claude processes are working on files", "Verify file I/O monitoring is active"},
			Data:        map[string]interface{}{"monitoring_time": time.Since(cs.startTime)},
			Timestamp:   time.Now(),
		})
	}
	
	return insights
}

/**
 * CONTEXT:   Generate performance insights
 * INPUT:     Comprehensive statistics data
 * OUTPUT:    Performance-related insights
 * BUSINESS:  Performance insights enable monitoring system optimization
 * CHANGE:    Initial performance insight generation
 * RISK:      Low - Performance insight generation
 */
func (cs *ComprehensiveStats) generatePerformanceInsights(stats *ComprehensiveStats) []Insight {
	var insights []Insight
	
	// High error rate warning
	if cs.processStats.ErrorCount > 10 {
		errorRate := float64(cs.processStats.ErrorCount) / float64(cs.processStats.ScansPerformed) * 100
		insights = append(insights, Insight{
			Type:        "performance",
			Category:    "warning",
			Title:       "High Error Rate Detected",
			Description: fmt.Sprintf("Process monitoring shows %.2f%% error rate", errorRate),
			Impact:      "medium",
			Suggestions: []string{"Check system permissions", "Review process monitoring configuration"},
			Data:        map[string]interface{}{"error_rate": errorRate, "error_count": cs.processStats.ErrorCount},
			Timestamp:   time.Now(),
		})
	}
	
	return insights
}

/**
 * CONTEXT:   Generate usage insights
 * INPUT:     Comprehensive statistics data
 * OUTPUT:    Usage pattern insights
 * BUSINESS:  Usage insights enable Claude usage pattern analysis
 * CHANGE:    Initial usage insight generation
 * RISK:      Low - Usage insight generation
 */
func (cs *ComprehensiveStats) generateUsageInsights(stats *ComprehensiveStats) []Insight {
	var insights []Insight
	
	// Multi-project usage insight
	if len(cs.projectStats) > 3 {
		insights = append(insights, Insight{
			Type:        "usage",
			Category:    "info",
			Title:       "Multi-Project Development Detected",
			Description: fmt.Sprintf("Active development across %d projects", len(cs.projectStats)),
			Impact:      "medium",
			Suggestions: []string{"Consider project-specific productivity tracking", "Monitor context switching patterns"},
			Data:        map[string]interface{}{"project_count": len(cs.projectStats)},
			Timestamp:   time.Now(),
		})
	}
	
	// Claude API usage insight
	if cs.httpStats.ClaudeAPIRequests > 100 {
		insights = append(insights, Insight{
			Type:        "usage",
			Category:    "info",
			Title:       "Heavy Claude API Usage",
			Description: fmt.Sprintf("Detected %d Claude API requests", cs.httpStats.ClaudeAPIRequests),
			Impact:      "medium",
			Suggestions: []string{"Monitor API usage patterns", "Consider API cost optimization"},
			Data:        map[string]interface{}{"api_requests": cs.httpStats.ClaudeAPIRequests},
			Timestamp:   time.Now(),
		})
	}
	
	return insights
}

/**
 * CONTEXT:   Generate optimization insights
 * INPUT:     Comprehensive statistics data
 * OUTPUT:    Optimization recommendation insights
 * BUSINESS:  Optimization insights enable system and workflow improvements
 * CHANGE:    Initial optimization insight generation
 * RISK:      Low - Optimization insight generation
 */
func (cs *ComprehensiveStats) generateOptimizationInsights(stats *ComprehensiveStats) []Insight {
	var insights []Insight
	
	// File I/O optimization
	if cs.fileIOStats.TotalEvents > 1000 && float64(cs.fileIOStats.WorkFileEvents)/float64(cs.fileIOStats.TotalEvents) < 0.5 {
		insights = append(insights, Insight{
			Type:        "optimization",
			Category:    "recommendation",
			Title:       "File I/O Pattern Optimization",
			Description: "Many non-work file operations detected, consider filtering optimization",
			Impact:      "low",
			Suggestions: []string{"Review file pattern filters", "Exclude more system directories"},
			Data:        map[string]interface{}{"total_events": cs.fileIOStats.TotalEvents, "work_ratio": float64(cs.fileIOStats.WorkFileEvents)/float64(cs.fileIOStats.TotalEvents)},
			Timestamp:   time.Now(),
		})
	}
	
	return insights
}

// Helper methods

func (cs *ComprehensiveStats) getTotalEvents() uint64 {
	total := uint64(0)
	for _, count := range cs.eventCounts {
		total += count
	}
	return total
}

func (cs *ComprehensiveStats) getMonitoringHealth() string {
	totalErrors := cs.processStats.ErrorCount + cs.fileIOStats.ErrorCount + 
	               cs.httpStats.ErrorCount + cs.networkStats.ErrorCount
	
	if totalErrors == 0 {
		return "excellent"
	} else if totalErrors < 10 {
		return "good"
	} else if totalErrors < 50 {
		return "fair"
	} else {
		return "poor"
	}
}

func (cs *ComprehensiveStats) getTopProjects(limit int) []ProjectStats {
	projects := make([]ProjectStats, 0, len(cs.projectStats))
	for _, proj := range cs.projectStats {
		projects = append(projects, *proj)
	}
	
	// Sort by productivity score
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].ProductivityScore > projects[j].ProductivityScore
	})
	
	if len(projects) > limit {
		projects = projects[:limit]
	}
	
	return projects
}

func (cs *ComprehensiveStats) getTopProject() *ProjectStats {
	var topProject *ProjectStats
	maxScore := 0.0
	
	for _, proj := range cs.projectStats {
		if proj.ProductivityScore > maxScore {
			maxScore = proj.ProductivityScore
			topProject = proj
		}
	}
	
	return topProject
}

func (cs *ComprehensiveStats) getTimelineData(duration time.Duration) []TimelineEntry {
	cutoff := time.Now().Add(-duration)
	var timelineData []TimelineEntry
	
	for _, entry := range cs.timelineStats {
		if entry.Timestamp.After(cutoff) {
			timelineData = append(timelineData, entry)
		}
	}
	
	return timelineData
}

func (cs *ComprehensiveStats) isClaudeEndpoint(host string) bool {
	claudeHosts := []string{"api.anthropic.com", "claude.ai", "anthropic.com"}
	for _, claudeHost := range claudeHosts {
		if strings.Contains(host, claudeHost) {
			return true
		}
	}
	return false
}

/**
 * CONTEXT:   Print comprehensive statistics summary
 * INPUT:     Statistics display request
 * OUTPUT:    Formatted statistics summary
 * BUSINESS:  Statistics summary provides quick system overview
 * CHANGE:    Initial statistics summary display
 * RISK:      Low - Display utility function
 */
func (cs *ComprehensiveStats) PrintSummary() {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	fmt.Printf("\n========== CLAUDE MONITORING STATISTICS ==========\n")
	fmt.Printf("Uptime: %s\n", time.Since(cs.startTime).Round(time.Second))
	fmt.Printf("Total Events: %d\n", cs.getTotalEvents())
	fmt.Printf("Active Projects: %d\n", len(cs.projectStats))
	fmt.Printf("Monitoring Health: %s\n", cs.getMonitoringHealth())
	
	fmt.Printf("\nEvent Breakdown:\n")
	for eventType, count := range cs.eventCounts {
		fmt.Printf("  %s: %d\n", eventType, count)
	}
	
	fmt.Printf("\nTop Projects:\n")
	topProjects := cs.getTopProjects(5)
	for i, proj := range topProjects {
		fmt.Printf("  %d. %s (Score: %.2f)\n", i+1, proj.ProjectName, proj.ProductivityScore)
	}
	
	if cs.config.EnableInsights && cs.insightEngine != nil {
		cs.generateInsights()
		if len(cs.insightEngine.insights) > 0 {
			fmt.Printf("\nKey Insights:\n")
			for i, insight := range cs.insightEngine.insights {
				if i >= 3 { // Show top 3 insights
					break
				}
				fmt.Printf("  %s: %s\n", insight.Category, insight.Title)
			}
		}
	}
	
	fmt.Printf("================================================\n")
}

/**
 * CONTEXT:   Print prototype-compatible statistics summary from process-check-prototype.go
 * INPUT:     Prototype statistics display request
 * OUTPUT:    Formatted prototype-style statistics summary
 * BUSINESS:  Prototype summary provides familiar monitoring output format
 * CHANGE:    Prototype statistics summary from process-check-prototype.go
 * RISK:      Low - Display utility for prototype compatibility
 */
func (cs *ComprehensiveStats) PrintPrototypeSummary() {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	duration := time.Since(cs.prototypeStats.StartTime)
	
	fmt.Println("\n========== RESUMEN DE ACTIVIDAD (Prototype Style) ==========")
	fmt.Printf("Tiempo de monitoreo: %s\n", duration.Round(time.Second))
	fmt.Println("\nEventos por tipo:")
	
	for eventType, count := range cs.prototypeStats.EventCountsByType {
		fmt.Printf("  %s: %d\n", eventType, count)
	}
	
	fmt.Printf("\nI/O Total:\n")
	fmt.Printf("  Lectura: %s\n", cs.formatBytes(cs.prototypeStats.TotalBytesRead))
	fmt.Printf("  Escritura: %s\n", cs.formatBytes(cs.prototypeStats.TotalBytesWrite))
	
	fmt.Printf("\nHTTP Requests:\n")
	fmt.Printf("  Total HTTP Requests: %d\n", cs.prototypeStats.HTTPRequestCount)
	fmt.Printf("  POST Requests: %d\n", cs.prototypeStats.HTTPPostCount)
	fmt.Printf("  Claude API POST Requests: %d\n", cs.prototypeStats.ClaudeAPIPostCount)
	
	fmt.Printf("\nProcess Events:\n")
	fmt.Printf("  Process Started: %d\n", cs.prototypeStats.ProcessStartCount)
	fmt.Printf("  Process Stopped: %d\n", cs.prototypeStats.ProcessStopCount)
	
	fmt.Printf("\nActivity Summary:\n")
	fmt.Printf("  Work File Operations: %d\n", cs.prototypeStats.WorkFileOperations)
	fmt.Printf("  Network Connections: %d\n", cs.prototypeStats.NetworkConnections)
	
	fmt.Printf("============================================================\n")
}

/**
 * CONTEXT:   Format byte counts for display (prototype compatibility)
 * INPUT:     Byte count as int64
 * OUTPUT:    Formatted byte string with appropriate units
 * BUSINESS:  Byte formatting provides readable data size display
 * CHANGE:    Byte formatting utility from prototype
 * RISK:      Low - Display utility function
 */
func (cs *ComprehensiveStats) formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

/**
 * CONTEXT:   Export statistics to JSON
 * INPUT:     JSON export request
 * OUTPUT:    JSON representation of comprehensive statistics
 * BUSINESS:  JSON export enables external analytics integration
 * CHANGE:    Initial JSON export functionality
 * RISK:      Low - JSON export utility
 */
func (cs *ComprehensiveStats) ExportJSON() (string, error) {
	report := cs.GenerateReport()
	
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal statistics to JSON: %w", err)
	}
	
	return string(jsonData), nil
}

// ProcessMonitorStats is needed to match the interface
type ProcessMonitorStats struct {
	ScansPerformed     uint64        `json:"scans_performed"`
	ProcessesDetected  uint64        `json:"processes_detected"`
	EventsGenerated    uint64        `json:"events_generated"`
	LastScanTime       time.Time     `json:"last_scan_time"`
	StartTime          time.Time     `json:"start_time"`
	AverageScanTime    time.Duration `json:"average_scan_time"`
	ErrorCount         uint64        `json:"error_count"`
}