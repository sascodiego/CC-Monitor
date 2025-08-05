/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-ACTIVITY-001
 * CONTEXT:   Enhanced activity monitoring with real inactivity detection and optimized thresholds
 * REASON:    Need to detect real user inactivity vs just process existence for accurate work block tracking
 * CHANGE:    Updated thresholds for better detection of small user interactions (200B min, 100B keepalive, 2KB burst).
 * PREVENTION:Monitor multiple signals of activity and implement proper timeout logic
 * RISK:      Medium - False positives on inactivity could interrupt active work sessions
 */
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
	"github.com/claude-monitor/claude-monitor/pkg/events"
	"github.com/google/uuid"
)

type ActivityIndicator struct {
	Timestamp          time.Time              `json:"timestamp"`
	ProcessCount       int                   `json:"processCount"`
	NetworkConnections int                   `json:"networkConnections"`
	APIActivity        bool                  `json:"apiActivity"`
	ConnectionDetails  []ConnectionInfo      `json:"connectionDetails"`
	DataTransferBytes  int64                 `json:"dataTransferBytes"`
	TrafficPattern     TrafficPatternType    `json:"trafficPattern"`
	// HTTP method detection from eBPF events
	HTTPMethodCounts   HTTPMethodStats       `json:"httpMethodCounts"`
	UserInteractions   int                   `json:"userInteractions"`
	BackgroundOps      int                   `json:"backgroundOps"`
	HTTPEventsDetected bool                  `json:"httpEventsDetected"`
}

type ConnectionInfo struct {
	LocalAddr    string    `json:"localAddr"`
	RemoteAddr   string    `json:"remoteAddr"`
	State        string    `json:"state"`
	Duration     time.Duration `json:"duration"`
	RxBytes      int64     `json:"rxBytes"`
	TxBytes      int64     `json:"txBytes"`
	LastSeen     time.Time `json:"lastSeen"`
}

type TrafficPatternType int

const (
	TrafficUnknown TrafficPatternType = iota
	TrafficKeepalive     // Regular, small data transfers
	TrafficInteractive   // Larger data transfers, irregular timing
	TrafficBurst         // Sudden increase in data transfer
	TrafficIdle          // No significant data transfer
	TrafficHTTPUser      // HTTP POST/PUT requests - definitive user activity
	TrafficHTTPBackground // HTTP GET/OPTIONS requests - background operations
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-HTTP-001
 * CONTEXT:   HTTP method statistics tracking for enhanced activity classification
 * REASON:    Need to track HTTP method distribution to distinguish user activity from background operations
 * CHANGE:    Initial implementation.
 * PREVENTION:Initialize all counters properly and handle concurrent access safely
 * RISK:      Low - Statistics tracking for monitoring and debugging HTTP method detection
 */
type HTTPMethodStats struct {
	POST    int `json:"post"`
	GET     int `json:"get"`
	PUT     int `json:"put"`
	PATCH   int `json:"patch"`
	DELETE  int `json:"delete"`
	OPTIONS int `json:"options"`
	HEAD    int `json:"head"`
	Other   int `json:"other"`
}

type EnhancedDaemon struct {
	logger              arch.Logger
	statusFile          string
	pidFile             string
	currentSession      *domain.Session
	currentWorkBlock    *domain.WorkBlock
	running             bool
	ctx                 context.Context
	cancel              context.CancelFunc
	lastRealActivity    time.Time
	inactivityTimeout   time.Duration
	activityHistory     []ActivityIndicator
	connectionTracker   map[string]*ConnectionInfo
	persistence         *SimplePersistence  // Add persistence
	baselineConnections int
	minActivityThreshold int64
	patternAnalyzer     *TrafficPatternAnalyzer
	// HTTP method detection components
	eventChannel        <-chan *events.SystemEvent
	httpEventProcessor  *HTTPEventProcessor
	eventValidator      *events.EventValidator
}

type TrafficPatternAnalyzer struct {
	baselineDataRate    int64     // Bytes per second baseline
	lastMeasurement     time.Time
	connectionLifetimes map[string]time.Duration
	keepaliveThreshold  int64     // Minimum bytes for real activity
	burstThreshold      int64     // Sudden increase threshold
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-HTTP-003
 * CONTEXT:   HTTP event processor for real-time analysis of eBPF HTTP events
 * REASON:    Need dedicated processor to handle HTTP events from eBPF and classify user vs background activity
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle event processing errors gracefully and maintain thread safety
 * RISK:      Medium - Event processing errors could cause activity detection failures
 */
type HTTPEventProcessor struct {
	mu               sync.RWMutex
	logger           arch.Logger
	methodStats      HTTPMethodStats
	userInteractions int
	backgroundOps    int
	eventsProcessed  int64
	lastEventTime    time.Time
}

func NewHTTPEventProcessor(logger arch.Logger) *HTTPEventProcessor {
	return &HTTPEventProcessor{
		logger:        logger,
		methodStats:   HTTPMethodStats{},
		lastEventTime: time.Now(),
	}
}

// ProcessEvent processes a single HTTP event and returns classification results
func (hep *HTTPEventProcessor) ProcessEvent(event *events.SystemEvent) (isUserActivity, isBackground bool) {
	hep.mu.Lock()
	defer hep.mu.Unlock()

	if !event.IsHTTPRequest() {
		return false, false
	}

	// Update statistics
	hep.eventsProcessed++
	hep.lastEventTime = event.Timestamp
	
	method := event.GetHTTPMethod()
	hep.updateMethodStats(method)

	// Classify the event
	isUserActivity = event.IsUserInteraction()
	isBackground = event.IsBackgroundOperation()

	if isUserActivity {
		hep.userInteractions++
		hep.logger.Debug("HTTP user interaction detected", 
			"method", method, 
			"uri", event.GetHTTPURI(),
			"pid", event.PID)
	} else if isBackground {
		hep.backgroundOps++
		hep.logger.Debug("HTTP background operation detected", 
			"method", method, 
			"uri", event.GetHTTPURI(),
			"pid", event.PID)
	}

	return isUserActivity, isBackground
}

func (hep *HTTPEventProcessor) updateMethodStats(method string) {
	switch method {
	case "POST":
		hep.methodStats.POST++
	case "GET":
		hep.methodStats.GET++
	case "PUT":
		hep.methodStats.PUT++
	case "PATCH":
		hep.methodStats.PATCH++
	case "DELETE":
		hep.methodStats.DELETE++
	case "OPTIONS":
		hep.methodStats.OPTIONS++
	case "HEAD":
		hep.methodStats.HEAD++
	default:
		hep.methodStats.Other++
	}
}

// GetStats returns current HTTP processing statistics
func (hep *HTTPEventProcessor) GetStats() (HTTPMethodStats, int, int, bool) {
	hep.mu.RLock()
	defer hep.mu.RUnlock()
	
	hasEvents := hep.eventsProcessed > 0
	return hep.methodStats, hep.userInteractions, hep.backgroundOps, hasEvents
}

// Reset clears all statistics for a new measurement cycle
func (hep *HTTPEventProcessor) Reset() {
	hep.mu.Lock()
	defer hep.mu.Unlock()
	
	hep.methodStats = HTTPMethodStats{}
	hep.userInteractions = 0
	hep.backgroundOps = 0
}

func NewEnhancedDaemon(log arch.Logger) *EnhancedDaemon {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize persistence
	persistence, err := NewSimplePersistence("/var/lib/claude-monitor/claude.db")
	if err != nil {
		log.Error("Failed to initialize persistence, continuing without database", "error", err)
		persistence = nil
	}
	
	return &EnhancedDaemon{
		logger:            log,
		statusFile:        "/tmp/claude-monitor-status.json",
		pidFile:           "/tmp/claude-monitor.pid",
		ctx:               ctx,
		cancel:            cancel,
		inactivityTimeout: 5 * time.Minute, // 5-minute inactivity timeout
		lastRealActivity:  time.Now(),
		activityHistory:   make([]ActivityIndicator, 0),
		connectionTracker: make(map[string]*ConnectionInfo),
		baselineConnections: -1, // Initialize to detect baseline
		minActivityThreshold: 200, // Minimum 200 bytes for real activity
		persistence:       persistence, // Add persistence
		patternAnalyzer: &TrafficPatternAnalyzer{
			keepaliveThreshold: 100,  // 100 bytes threshold for keepalive
			burstThreshold:     2048, // 2KB threshold for burst activity
			connectionLifetimes: make(map[string]time.Duration),
		},
		httpEventProcessor: NewHTTPEventProcessor(log),
		eventValidator:     &events.EventValidator{},
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-HTTP-002
 * CONTEXT:   Event channel injection for eBPF HTTP event integration
 * REASON:    Need to connect eBPF event stream for HTTP method detection
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure event channel is properly connected and handle nil gracefully
 * RISK:      Medium - HTTP event integration must handle eBPF availability gracefully
 */
// SetEventChannel allows injection of eBPF event channel for HTTP method detection
func (ed *EnhancedDaemon) SetEventChannel(eventCh <-chan *events.SystemEvent) {
	ed.eventChannel = eventCh
	ed.logger.Info("eBPF event channel connected for HTTP method detection")
}

func (ed *EnhancedDaemon) Start() error {
	ed.running = true
	
	// Write PID file
	if err := ed.writePidFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	ed.logger.Info("Enhanced daemon started with activity monitoring")
	
	// Start monitoring loop
	go ed.monitoringLoop()
	
	// Start HTTP event processing if eBPF channel is available
	if ed.eventChannel != nil {
		go ed.httpEventProcessingLoop()
	}
	
	// Start connection cleanup routine
	go ed.connectionCleanupLoop()
	
	return nil
}

func (ed *EnhancedDaemon) Stop() error {
	ed.running = false
	if ed.cancel != nil {
		ed.cancel()
	}
	
	// Finalize current work block before stopping
	if ed.currentWorkBlock != nil && ed.currentWorkBlock.IsActive {
		ed.finalizeCurrentWorkBlock()
	}
	
	// Remove PID file
	os.Remove(ed.pidFile)
	
	ed.logger.Info("Enhanced daemon stopped")
	return nil
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-014
 * CONTEXT:   Background cleanup loop for connection tracking maintenance
 * REASON:    Prevent memory leaks from stale connection tracking data
 * CHANGE:    Added dedicated cleanup loop running every 5 minutes.
 * PREVENTION:Monitor memory usage and adjust cleanup frequency if needed
 * RISK:      Low - Resource cleanup routine for system health
 */
func (ed *EnhancedDaemon) connectionCleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ed.ctx.Done():
			return
		case <-ticker.C:
			if ed.running {
				ed.cleanupStaleConnections()
			}
		}
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-HTTP-004
 * CONTEXT:   HTTP event processing loop for real-time eBPF event consumption
 * REASON:    Need dedicated goroutine to process HTTP events from eBPF for superior activity detection
 * CHANGE:    Initial implementation.
 * PREVENTION:Handle event processing errors gracefully and maintain performance under load
 * RISK:      Medium - Event processing failures could cause activity detection to fall back to heuristics
 */
func (ed *EnhancedDaemon) httpEventProcessingLoop() {
	ed.logger.Info("HTTP event processing loop started")
	
	for {
		select {
		case <-ed.ctx.Done():
			ed.logger.Info("HTTP event processing loop stopped")
			return
			
		case event := <-ed.eventChannel:
			if event == nil {
				continue
			}
			
			// Validate event
			if err := ed.eventValidator.Validate(event); err != nil {
				ed.logger.Debug("Invalid HTTP event received", "error", err)
				continue
			}
			
			// Only process HTTP request events
			if !event.IsHTTPRequest() {
				continue
			}
			
			// Process the HTTP event
			isUserActivity, isBackground := ed.httpEventProcessor.ProcessEvent(event)
			
			// If it's a user interaction, update activity timestamp immediately
			if isUserActivity {
				ed.lastRealActivity = event.Timestamp
				ed.logger.Debug("HTTP user interaction updated activity timestamp", 
					"method", event.GetHTTPMethod(),
					"uri", event.GetHTTPURI(),
					"timestamp", event.Timestamp)
			}
			
			// Log background operations for debugging
			if isBackground {
				ed.logger.Debug("HTTP background operation detected", 
					"method", event.GetHTTPMethod(),
					"uri", event.GetHTTPURI())
			}
		}
	}
}

func (ed *EnhancedDaemon) writePidFile() error {
	pid := os.Getpid()
	return ioutil.WriteFile(ed.pidFile, []byte(strconv.Itoa(pid)), 0644)
}

func (ed *EnhancedDaemon) monitoringLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ed.ctx.Done():
			return
		case <-ticker.C:
			if !ed.running {
				return
			}
			ed.checkActivity()
			ed.processActivityState()
			ed.writeStatus()
		}
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-009
 * CONTEXT:   Enhanced activity detection with traffic pattern analysis
 * REASON:    Need to distinguish between real user interactions and automatic keepalive connections
 * CHANGE:    Implemented sophisticated traffic analysis with data volume and pattern recognition.
 * PREVENTION:Monitor baseline connection counts and data transfer patterns to avoid false positives
 * RISK:      Medium - Incorrect thresholds could miss real activity or create false positives
 */
func (ed *EnhancedDaemon) checkActivity() {
	now := time.Now()
	
	// Check for Claude processes
	processCount := ed.getClaudeProcessCount()
	
	// Get detailed connection information with data transfer metrics
	connectionDetails, totalDataTransfer := ed.getDetailedConnectionInfo()
	networkConnections := len(connectionDetails)
	
	// Get HTTP method statistics from event processor
	httpMethodCounts, userInteractions, backgroundOps, httpEventsDetected := ed.httpEventProcessor.GetStats()
	
	// Analyze traffic patterns with HTTP method information
	trafficPattern := ed.analyzeTrafficPatternWithHTTP(connectionDetails, totalDataTransfer, httpEventsDetected, userInteractions > 0)
	
	// Detect API activity using enhanced heuristics + HTTP detection
	apiActivity := ed.detectEnhancedAPIActivity(connectionDetails, trafficPattern)
	
	indicator := ActivityIndicator{
		Timestamp:          now,
		ProcessCount:       processCount,
		NetworkConnections: networkConnections,
		APIActivity:        apiActivity,
		ConnectionDetails:  connectionDetails,
		DataTransferBytes:  totalDataTransfer,
		TrafficPattern:     trafficPattern,
		// HTTP method detection data
		HTTPMethodCounts:   httpMethodCounts,
		UserInteractions:   userInteractions,
		BackgroundOps:      backgroundOps,
		HTTPEventsDetected: httpEventsDetected,
	}
	
	// Reset HTTP processor stats for next cycle
	ed.httpEventProcessor.Reset()
	
	// Keep last 12 indicators (1 minute of history)
	ed.activityHistory = append(ed.activityHistory, indicator)
	if len(ed.activityHistory) > 12 {
		ed.activityHistory = ed.activityHistory[1:]
	}
	
	// Determine if this represents "real" activity using enhanced detection
	if ed.isRealActivityEnhanced(indicator) {
		ed.lastRealActivity = now
		ed.logger.Debug("Real activity detected", 
			"processes", processCount,
			"connections", networkConnections,
			"dataTransfer", totalDataTransfer,
			"pattern", trafficPattern,
			"apiActivity", apiActivity)
	} else {
		ed.logger.Debug("Keepalive/background activity detected (not real activity)",
			"processes", processCount,
			"connections", networkConnections,
			"dataTransfer", totalDataTransfer,
			"pattern", trafficPattern)
	}
}

func (ed *EnhancedDaemon) getClaudeProcessCount() int {
	cmd := exec.Command("pgrep", "-f", "claude")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	
	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(pids) == 1 && pids[0] == "" {
		return 0
	}
	
	return len(pids)
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-010
 * CONTEXT:   Enhanced connection monitoring with detailed traffic analysis and realistic thresholds
 * REASON:    Need detailed connection info including data transfer metrics to distinguish activity types
 * CHANGE:    Updated thresholds to detect smaller interactions: 200B min activity, 100B keepalive, 2KB burst.
 * PREVENTION:Parse network statistics carefully and handle connection tracking state properly
 * RISK:      Medium - Network parsing errors could cause incorrect activity detection
 */
func (ed *EnhancedDaemon) getDetailedConnectionInfo() ([]ConnectionInfo, int64) {
	var connections []ConnectionInfo
	var totalDataTransfer int64
	
	// Get detailed connection info with data transfer statistics
	cmd := exec.Command("ss", "-i", "-n", "-t", "state", "established")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to netstat if ss is not available
		return ed.getConnectionInfoFallback()
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if ed.isAnthropicConnection(line) {
			connInfo := ed.parseConnectionInfo(line)
			if connInfo != nil {
				connections = append(connections, *connInfo)
				totalDataTransfer += connInfo.RxBytes + connInfo.TxBytes
				
				// Update connection tracker
				connKey := fmt.Sprintf("%s->%s", connInfo.LocalAddr, connInfo.RemoteAddr)
				if existing, exists := ed.connectionTracker[connKey]; exists {
					connInfo.Duration = time.Since(existing.LastSeen)
				}
				connInfo.LastSeen = time.Now()
				ed.connectionTracker[connKey] = connInfo
			}
		}
	}
	
	// Establish baseline if not set
	if ed.baselineConnections == -1 && len(connections) > 0 {
		ed.baselineConnections = len(connections)
		ed.logger.Info("Established connection baseline", "connections", ed.baselineConnections)
	}
	
	return connections, totalDataTransfer
}

func (ed *EnhancedDaemon) getConnectionInfoFallback() ([]ConnectionInfo, int64) {
	var connections []ConnectionInfo
	
	// Fallback to basic netstat if ss is not available
	cmd := exec.Command("netstat", "-tn")
	output, err := cmd.Output()
	if err != nil {
		return connections, 0
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "ESTABLISHED") && ed.isAnthropicConnection(line) {
			// Basic connection info without data transfer metrics
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				connInfo := &ConnectionInfo{
					LocalAddr:  parts[3],
					RemoteAddr: parts[4],
					State:      "ESTABLISHED",
					LastSeen:   time.Now(),
				}
				connections = append(connections, *connInfo)
			}
		}
	}
	
	return connections, 0
}

func (ed *EnhancedDaemon) isAnthropicConnection(line string) bool {
	return strings.Contains(line, "api.anthropic.com") || 
		   strings.Contains(line, ":443") ||
		   strings.Contains(line, "anthropic")
}

func (ed *EnhancedDaemon) parseConnectionInfo(line string) *ConnectionInfo {
	// Parse ss output format: "tcp ESTAB 0 0 local_addr remote_addr"
	parts := strings.Fields(line)
	if len(parts) < 5 {
		return nil
	}
	
	connInfo := &ConnectionInfo{
		LocalAddr:  parts[4],
		RemoteAddr: parts[5],
		State:      parts[1],
		LastSeen:   time.Now(),
	}
	
	// Try to extract byte counts from ss output
	if len(parts) >= 7 {
		if rxBytes, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
			connInfo.RxBytes = rxBytes
		}
		if txBytes, err := strconv.ParseInt(parts[3], 10, 64); err == nil {
			connInfo.TxBytes = txBytes
		}
	}
	
	return connInfo
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-HTTP-005
 * CONTEXT:   Enhanced traffic pattern analysis integrating HTTP method detection with network heuristics
 * REASON:    HTTP method information provides definitive user activity classification, superior to network heuristics
 * CHANGE:    Removed byte size restrictions for HTTP method detection - POST/PUT requests are user activity regardless of size.
 * PREVENTION:Prioritize HTTP method info when available, fallback gracefully to network analysis
 * RISK:      Low - HTTP detection enhances accuracy without breaking existing fallback logic
 */
func (ed *EnhancedDaemon) analyzeTrafficPatternWithHTTP(connections []ConnectionInfo, totalData int64, httpEventsDetected bool, hasUserInteractions bool) TrafficPatternType {
	// If we have HTTP events detected, prioritize HTTP method classification - IGNORE BYTE SIZE
	if httpEventsDetected {
		if hasUserInteractions {
			ed.logger.Debug("HTTP user interaction detected - definitive user activity (ignoring byte size)", 
				"dataBytes", totalData)
			return TrafficHTTPUser
		} else {
			ed.logger.Debug("HTTP events detected but no user interactions - background operations (ignoring byte size)",
				"dataBytes", totalData)
			return TrafficHTTPBackground
		}
	}
	
	// Fallback to traditional network analysis when HTTP detection is not available
	return ed.analyzeTrafficPattern(connections, totalData)
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-011
 * CONTEXT:   Traffic pattern analysis to distinguish interactive vs keepalive traffic - FALLBACK ONLY
 * REASON:    Need sophisticated pattern recognition to avoid false positives when HTTP method detection unavailable
 * CHANGE:    This is now FALLBACK logic only - HTTP method detection takes precedence over byte size analysis.
 * PREVENTION:Only use this when HTTP method detection is unavailable, calibrate thresholds appropriately
 * RISK:      Low - This is secondary to HTTP method detection which provides definitive classification
 */
func (ed *EnhancedDaemon) analyzeTrafficPattern(connections []ConnectionInfo, totalData int64) TrafficPatternType {
	if len(connections) == 0 {
		return TrafficIdle
	}
	
	// Analyze data volume patterns
	if totalData == 0 {
		return TrafficIdle
	}
	
	// Check for burst activity (sudden increase in data)
	if len(ed.activityHistory) >= 2 {
		previous := ed.activityHistory[len(ed.activityHistory)-1]
		dataIncrease := totalData - previous.DataTransferBytes
		
		if dataIncrease > ed.patternAnalyzer.burstThreshold {
			return TrafficBurst
		}
	}
	
	// Check for keepalive pattern (small, regular data transfers)
	if totalData <= ed.patternAnalyzer.keepaliveThreshold {
		// Small data transfer, likely keepalive
		if ed.isRegularPattern(connections) {
			return TrafficKeepalive
		}
	}
	
	// Check connection count vs baseline
	if ed.baselineConnections > 0 {
		connectionChange := len(connections) - ed.baselineConnections
		
		// Significant connection count change with data transfer
		if abs(connectionChange) > 2 && totalData > ed.minActivityThreshold {
			return TrafficInteractive
		}
	}
	
	// Default to interactive if we have substantial data transfer
	if totalData > ed.minActivityThreshold {
		return TrafficInteractive
	}
	
	return TrafficKeepalive
}

func (ed *EnhancedDaemon) isRegularPattern(connections []ConnectionInfo) bool {
	// Check if connections follow a regular keepalive pattern
	if len(ed.activityHistory) < 3 {
		return false
	}
	
	// Look for consistent connection count over time
	recentHistory := ed.activityHistory[len(ed.activityHistory)-3:]
	connCounts := make([]int, len(recentHistory))
	for i, hist := range recentHistory {
		connCounts[i] = hist.NetworkConnections
	}
	
	// Check for low variance in connection count (indicates stable keepalive)
	variance := calculateVariance(connCounts)
	return variance < 2.0 // Low variance threshold
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-HTTP-007
 * CONTEXT:   Enhanced API activity detection prioritizing HTTP method classification over byte size heuristics
 * REASON:    HTTP method detection provides definitive API activity classification
 * CHANGE:    Added HTTP traffic pattern handling, removed byte size restrictions for HTTP-detected activity.
 * PREVENTION:Prioritize HTTP method info, use byte thresholds only as fallback for non-HTTP patterns
 * RISK:      Low - HTTP detection provides superior accuracy over network heuristics
 */
func (ed *EnhancedDaemon) detectEnhancedAPIActivity(connections []ConnectionInfo, pattern TrafficPatternType) bool {
	// Enhanced API activity detection based on pattern analysis
	switch pattern {
	case TrafficHTTPUser:
		// Definitive user API activity from HTTP POST/PUT - IGNORE BYTE SIZE
		ed.logger.Debug("API activity confirmed by HTTP user interaction (ignoring byte size)")
		return true
		
	case TrafficHTTPBackground:
		// Background HTTP operations (GET/OPTIONS) - still API activity but not user-initiated
		ed.logger.Debug("Background API activity detected via HTTP methods")
		return true
		
	case TrafficInteractive, TrafficBurst:
		// Legacy network pattern detection - still valid
		return true
		
	case TrafficKeepalive:
		// Keepalive traffic is not considered real activity
		return false
		
	case TrafficIdle:
		// No network activity
		return false
		
	default:
		// Unknown pattern - fallback to data volume analysis (only when HTTP detection unavailable)
		if len(ed.activityHistory) > 0 {
			current := ed.activityHistory[len(ed.activityHistory)-1]
			// Only use byte threshold if no HTTP events were detected
			if !current.HTTPEventsDetected {
				return current.DataTransferBytes > ed.minActivityThreshold
			}
			// If HTTP events were detected but pattern is unknown, be conservative
			ed.logger.Debug("HTTP events detected but pattern unknown - being conservative")
			return false
		}
		return len(connections) > 0
	}
}

func calculateVariance(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	
	// Calculate mean
	sum := 0
	for _, v := range values {
		sum += v
	}
	mean := float64(sum) / float64(len(values))
	
	// Calculate variance
	varSum := 0.0
	for _, v := range values {
		diff := float64(v) - mean
		varSum += diff * diff
	}
	
	return varSum / float64(len(values))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-HTTP-006
 * CONTEXT:   Enhanced real activity detection prioritizing HTTP method information over network heuristics
 * REASON:    HTTP method detection provides definitive classification superior to network traffic analysis
 * CHANGE:    Removed byte size restrictions for POST detection - HTTP method information takes precedence.
 * PREVENTION:Always validate HTTP event data and maintain fallback to network analysis
 * RISK:      Low - HTTP detection enhances accuracy while preserving existing logic as fallback
 */
func (ed *EnhancedDaemon) isRealActivityEnhanced(indicator ActivityIndicator) bool {
	// Enhanced real activity detection with HTTP method prioritization:
	
	// 1. Must have Claude processes running
	if indicator.ProcessCount == 0 {
		return false
	}
	
	// 2. Prioritize HTTP method detection when available - NO BYTE SIZE RESTRICTIONS
	if indicator.HTTPEventsDetected {
		if indicator.UserInteractions > 0 {
			ed.logger.Debug("Real activity confirmed by HTTP user interactions (ignoring byte size)", 
				"userInteractions", indicator.UserInteractions,
				"httpMethods", indicator.HTTPMethodCounts,
				"dataBytes", indicator.DataTransferBytes)
			return true
		}
		
		// If we have HTTP events but no user interactions, it's background activity
		if indicator.BackgroundOps > 0 {
			ed.logger.Debug("Background activity detected via HTTP methods (ignoring byte size)",
				"backgroundOps", indicator.BackgroundOps,
				"httpMethods", indicator.HTTPMethodCounts,
				"dataBytes", indicator.DataTransferBytes)
			return false
		}
	}
	
	// 3. Traffic pattern analysis (enhanced with new HTTP patterns)
	switch indicator.TrafficPattern {
	case TrafficHTTPUser:
		// Definitive user activity from HTTP POST/PUT requests - IGNORE BYTE SIZE
		ed.logger.Debug("HTTP POST/PUT activity detected - definitive user interaction regardless of size")
		return true
		
	case TrafficHTTPBackground:
		// Definitive background activity from HTTP GET/OPTIONS requests - IGNORE BYTE SIZE
		ed.logger.Debug("HTTP GET/OPTIONS activity detected - definitive background operation regardless of size")
		return false
		
	case TrafficInteractive, TrafficBurst:
		// Clear indicators of user interaction (legacy network analysis)
		return true
		
	case TrafficKeepalive:
		// Keepalive traffic alone is not real activity
		// But check for additional signals (fallback only when HTTP detection unavailable)
		return ed.hasAdditionalActivitySignalsFallback(indicator)
		
	case TrafficIdle:
		// No network activity
		return false
		
	default:
		// Unknown pattern - fall back to data volume analysis ONLY when HTTP detection unavailable
		if !indicator.HTTPEventsDetected {
			return indicator.DataTransferBytes > ed.minActivityThreshold
		}
		// If HTTP events were detected but classified as unknown, be conservative
		return false
	}
}

func (ed *EnhancedDaemon) hasAdditionalActivitySignalsFallback(indicator ActivityIndicator) bool {
	// Look for additional signals that might indicate real activity even with keepalive traffic
	// NOTE: This is FALLBACK logic only when HTTP method detection is unavailable
	
	// If HTTP events were detected, trust the HTTP classification over heuristics
	if indicator.HTTPEventsDetected {
		// HTTP detection already classified this - don't override with heuristics
		ed.logger.Debug("HTTP events detected - trusting HTTP classification over heuristics")
		return false
	}
	
	// 1. Significant change in process count
	if len(ed.activityHistory) > 0 {
		previous := ed.activityHistory[len(ed.activityHistory)-1]
		processChange := abs(indicator.ProcessCount - previous.ProcessCount)
		if processChange > 1 {
			ed.logger.Debug("Process count change detected (fallback heuristic)", "change", processChange)
			return true
		}
	}
	
	// 2. New connections appearing (not just count changes)
	if ed.baselineConnections > 0 {
		connectionIncrease := indicator.NetworkConnections - ed.baselineConnections
		if connectionIncrease > 3 {
			ed.logger.Debug("Significant connection increase (fallback heuristic)", "increase", connectionIncrease)
			return true
		}
	}
	
	// 3. Data transfer above minimum threshold (fallback only when no HTTP detection)
	if indicator.DataTransferBytes > ed.minActivityThreshold {
		ed.logger.Debug("Data transfer above threshold (fallback heuristic)", "bytes", indicator.DataTransferBytes)
		return true
	}
	
	// 4. Check for sustained activity over time
	if ed.hasSustainedActivity() {
		ed.logger.Debug("Sustained activity pattern detected (fallback heuristic)")
		return true
	}
	
	return false
}

func (ed *EnhancedDaemon) hasSustainedActivity() bool {
	// Check if there has been consistent activity over the last few measurements
	if len(ed.activityHistory) < 3 {
		return false
	}
	
	recentHistory := ed.activityHistory[len(ed.activityHistory)-3:]
	activityCount := 0
	
	for _, hist := range recentHistory {
		if hist.ProcessCount > 0 && (hist.NetworkConnections > 0 || hist.APIActivity) {
			activityCount++
		}
	}
	
	// Sustained activity if active in most recent measurements
	return activityCount >= 2
}

func (ed *EnhancedDaemon) processActivityState() {
	now := time.Now()
	hasClaudeProcesses := ed.getClaudeProcessCount() > 0
	timeSinceActivity := now.Sub(ed.lastRealActivity)
	
	// Session management (5-hour windows)
	if hasClaudeProcesses && ed.currentSession == nil {
		// Start new session
		ed.startNewSession()
	} else if !hasClaudeProcesses && ed.currentSession != nil {
		// No Claude processes - end session
		ed.finalizeCurrentSession()
		return
	}
	
	// Work block management (5-minute inactivity timeout)
	if ed.currentSession != nil {
		if timeSinceActivity <= ed.inactivityTimeout {
			// Active period
			if ed.currentWorkBlock == nil {
				// Start new work block
				ed.startNewWorkBlock()
			} else {
				// Update existing work block activity with validation
				if ed.lastRealActivity.Before(ed.currentWorkBlock.StartTime) {
					ed.logger.Error("lastRealActivity is before work block start time, using start time",
						"lastRealActivity", ed.lastRealActivity,
						"workBlockStartTime", ed.currentWorkBlock.StartTime,
						"blockID", ed.currentWorkBlock.ID)
					ed.currentWorkBlock.UpdateActivity(ed.currentWorkBlock.StartTime)
				} else {
					ed.currentWorkBlock.UpdateActivity(ed.lastRealActivity)
				}
			}
		} else {
			// Inactive period (> 5 minutes since last real activity)
			if ed.currentWorkBlock != nil && ed.currentWorkBlock.IsActive {
				// Finalize current work block due to inactivity
				ed.finalizeCurrentWorkBlock()
				ed.logger.Info("Work block finalized due to inactivity", 
					"inactiveFor", timeSinceActivity,
					"blockID", ed.currentWorkBlock.ID)
			}
		}
	}
}

func (ed *EnhancedDaemon) startNewSession() {
	now := time.Now()
	ed.currentSession = &domain.Session{
		ID:        uuid.New().String(),
		StartTime: now,
		EndTime:   now.Add(5 * time.Hour), // 5-hour window
		IsActive:  true,
	}
	
	// Save session to database
	if ed.persistence != nil {
		sessionRecord := SessionRecord{
			SessionID: ed.currentSession.ID,
			StartTime: ed.currentSession.StartTime,
			EndTime:   ed.currentSession.EndTime,
			IsActive:  ed.currentSession.IsActive,
		}
		if err := ed.persistence.SaveSession(sessionRecord); err != nil {
			ed.logger.Error("Failed to save session to database", "error", err, "sessionID", ed.currentSession.ID)
		}
	}
	
	ed.logger.Info("New session started with enhanced monitoring", 
		"sessionID", ed.currentSession.ID,
		"endTime", ed.currentSession.EndTime)
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-015
 * CONTEXT:   Fixed work block creation timing bug where lastActivity could be before startTime
 * REASON:    lastActivity must never be before startTime to prevent negative duration calculations
 * CHANGE:    Ensure lastActivity is at least equal to startTime when creating new work blocks.
 * PREVENTION:Always validate timing relationships in work block creation and activity updates
 * RISK:      High - Incorrect timing could cause negative durations and billing calculation errors
 */
func (ed *EnhancedDaemon) startNewWorkBlock() {
	now := time.Now()
	
	// Ensure lastActivity is not before the work block start time
	lastActivity := ed.lastRealActivity
	if lastActivity.Before(now) {
		// If last activity was before now, use now as the start of this work block
		lastActivity = now
		ed.logger.Debug("Adjusted lastActivity to work block start time", 
			"originalLastActivity", ed.lastRealActivity,
			"adjustedLastActivity", lastActivity)
	}
	
	ed.currentWorkBlock = &domain.WorkBlock{
		ID:           uuid.New().String(),
		SessionID:    ed.currentSession.ID,
		StartTime:    now,
		LastActivity: lastActivity,
		IsActive:     true,
	}
	
	ed.logger.Info("New work block started", 
		"blockID", ed.currentWorkBlock.ID,
		"sessionID", ed.currentSession.ID,
		"startTime", now,
		"lastActivity", lastActivity)
}

func (ed *EnhancedDaemon) finalizeCurrentWorkBlock() {
	if ed.currentWorkBlock != nil {
		// Use last real activity time, not current time
		ed.currentWorkBlock.Finalize(ed.lastRealActivity)
		
		// Save work block to database
		if ed.persistence != nil {
			blockRecord := WorkBlockRecord{
				BlockID:      ed.currentWorkBlock.ID,
				SessionID:    ed.currentWorkBlock.SessionID,
				StartTime:    ed.currentWorkBlock.StartTime,
				EndTime:      ed.currentWorkBlock.EndTime,
				DurationSecs: int(ed.currentWorkBlock.Duration().Seconds()),
				IsActive:     false,
			}
			if err := ed.persistence.SaveWorkBlock(blockRecord); err != nil {
				ed.logger.Error("Failed to save work block to database", "error", err, "blockID", ed.currentWorkBlock.ID)
			}
		}
		
		ed.logger.Info("Work block finalized", 
			"blockID", ed.currentWorkBlock.ID,
			"duration", ed.currentWorkBlock.Duration(),
			"endedAt", ed.lastRealActivity)
	}
}

func (ed *EnhancedDaemon) finalizeCurrentSession() {
	if ed.currentSession != nil {
		ed.finalizeCurrentWorkBlock()
		ed.currentSession.IsActive = false
		
		// Save finalized session to database
		if ed.persistence != nil {
			sessionRecord := SessionRecord{
				SessionID: ed.currentSession.ID,
				StartTime: ed.currentSession.StartTime,
				EndTime:   ed.currentSession.EndTime,
				IsActive:  false,
			}
			if err := ed.persistence.SaveSession(sessionRecord); err != nil {
				ed.logger.Error("Failed to save finalized session to database", "error", err, "sessionID", ed.currentSession.ID)
			}
		}
		
		ed.logger.Info("Session finalized", "sessionID", ed.currentSession.ID)
		ed.currentSession = nil
		ed.currentWorkBlock = nil
	}
}

func (ed *EnhancedDaemon) writeStatus() {
	now := time.Now()
	timeSinceActivity := now.Sub(ed.lastRealActivity)
	
	status := map[string]interface{}{
		"daemonRunning":     ed.running,
		"timestamp":         now,
		"currentSession":    ed.currentSession,
		"currentWorkBlock":  ed.currentWorkBlock,
		"monitoringActive":  ed.running,
		"lastRealActivity":  ed.lastRealActivity,
		"timeSinceActivity": timeSinceActivity,
		"inactiveTimeout":   timeSinceActivity > ed.inactivityTimeout,
		"activityHistory":   ed.getRecentActivitySummary(),
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		ed.logger.Error("Failed to marshal status", "error", err)
		return
	}

	if err := ioutil.WriteFile(ed.statusFile, data, 0644); err != nil {
		ed.logger.Error("Failed to write status file", "error", err)
	}
}

func (ed *EnhancedDaemon) getRecentActivitySummary() map[string]interface{} {
	if len(ed.activityHistory) == 0 {
		return map[string]interface{}{"samples": 0}
	}
	
	recent := ed.activityHistory[len(ed.activityHistory)-1]
	return map[string]interface{}{
		"samples":            len(ed.activityHistory),
		"currentProcesses":   recent.ProcessCount,
		"currentConnections": recent.NetworkConnections,
		"currentAPIActivity": recent.APIActivity,
		"dataTransferBytes":  recent.DataTransferBytes,
		"trafficPattern":     ed.trafficPatternToString(recent.TrafficPattern),
		"baselineConnections": ed.baselineConnections,
		"activeConnections":  len(ed.connectionTracker),
		"minActivityThreshold": ed.minActivityThreshold,
		// HTTP method detection statistics
		"httpEventsDetected": recent.HTTPEventsDetected,
		"httpMethodCounts":   recent.HTTPMethodCounts,
		"userInteractions":   recent.UserInteractions,
		"backgroundOps":      recent.BackgroundOps,
	}
}

func (ed *EnhancedDaemon) trafficPatternToString(pattern TrafficPatternType) string {
	switch pattern {
	case TrafficKeepalive:
		return "keepalive"
	case TrafficInteractive:
		return "interactive"
	case TrafficBurst:
		return "burst"
	case TrafficIdle:
		return "idle"
	case TrafficHTTPUser:
		return "http_user"
	case TrafficHTTPBackground:
		return "http_background"
	default:
		return "unknown"
	}
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-CORE-013
 * CONTEXT:   Connection cleanup and maintenance functions
 * REASON:    Need to clean up stale connection tracking data to prevent memory leaks
 * CHANGE:    Added connection cleanup and pattern analysis maintenance functions.
 * PREVENTION:Run cleanup periodically to prevent unbounded memory growth
 * RISK:      Low - Memory leaks from connection tracking if cleanup fails
 */
func (ed *EnhancedDaemon) cleanupStaleConnections() {
	now := time.Now()
	staleThreshold := 10 * time.Minute
	
	for key, conn := range ed.connectionTracker {
		if now.Sub(conn.LastSeen) > staleThreshold {
			delete(ed.connectionTracker, key)
			delete(ed.patternAnalyzer.connectionLifetimes, key)
		}
	}
}

// GetActivityMetrics returns current activity detection metrics for monitoring
func (ed *EnhancedDaemon) GetActivityMetrics() map[string]interface{} {
	return map[string]interface{}{
		"baselineConnections":    ed.baselineConnections,
		"trackedConnections":     len(ed.connectionTracker),
		"minActivityThreshold":   ed.minActivityThreshold,
		"keepaliveThreshold":     ed.patternAnalyzer.keepaliveThreshold,
		"burstThreshold":         ed.patternAnalyzer.burstThreshold,
		"activityHistoryLength": len(ed.activityHistory),
		"lastRealActivity":      ed.lastRealActivity,
	}
}

// SetActivityThresholds allows runtime adjustment of activity detection parameters
func (ed *EnhancedDaemon) SetActivityThresholds(minActivity, keepalive, burst int64) {
	ed.minActivityThreshold = minActivity
	ed.patternAnalyzer.keepaliveThreshold = keepalive
	ed.patternAnalyzer.burstThreshold = burst
	ed.logger.Info("Activity thresholds updated",
		"minActivity", minActivity,
		"keepalive", keepalive,
		"burst", burst)
}