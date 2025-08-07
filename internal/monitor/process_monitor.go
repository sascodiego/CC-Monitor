/**
 * CONTEXT:   Process monitor for detecting Claude instances startup and shutdown
 * INPUT:     System process list polling and process event detection
 * OUTPUT:    Process lifecycle events for Claude work hour tracking
 * BUSINESS:  Process monitoring provides reliable Claude activity detection
 * CHANGE:    Initial process monitor implementation for Claude detection
 * RISK:      Medium - Process monitoring affecting system performance
 */

package monitor

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
 * CONTEXT:   Process event types for Claude lifecycle tracking
 * INPUT:     Process start and stop events
 * OUTPUT:    Structured event data for activity processing
 * BUSINESS:  Process events enable work session correlation
 * CHANGE:    Initial process event structure
 * RISK:      Low - Event data structure for process tracking
 */
type ProcessEventType string

const (
	ProcessStarted ProcessEventType = "started"
	ProcessStopped ProcessEventType = "stopped"
)

type ProcessEvent struct {
	Type      ProcessEventType `json:"type"`
	PID       int              `json:"pid"`
	PPID      int              `json:"ppid"`
	Command   string           `json:"command"`
	Args      []string         `json:"args"`
	WorkingDir string          `json:"working_dir"`
	StartTime time.Time        `json:"start_time"`
	UserID    string           `json:"user_id"`
	Timestamp time.Time        `json:"timestamp"`
}

/**
 * CONTEXT:   Process information tracking for Claude instances
 * INPUT:     Process metadata from system process table
 * OUTPUT:    Structured process information for lifecycle tracking
 * BUSINESS:  Process info enables session correlation and work tracking
 * CHANGE:    Initial process information structure
 * RISK:      Low - Process metadata structure
 */
type ProcessInfo struct {
	PID        int       `json:"pid"`
	PPID       int       `json:"ppid"`
	Command    string    `json:"command"`
	Args       []string  `json:"args"`
	WorkingDir string    `json:"working_dir"`
	StartTime  time.Time `json:"start_time"`
	UserID     string    `json:"user_id"`
	CPUPercent float64   `json:"cpu_percent"`
	MemoryMB   int       `json:"memory_mb"`
}

/**
 * CONTEXT:   Enhanced process monitor with file I/O tracking for Claude processes
 * INPUT:     System process monitoring and Claude process identification
 * OUTPUT:    Process events, lifecycle management, and file I/O activity
 * BUSINESS:  Monitor enables comprehensive Claude activity tracking with work detection
 * CHANGE:    Enhanced monitor with integrated file I/O monitoring for work activity
 * RISK:      High - System monitoring affecting performance and resource usage
 */
type ProcessMonitor struct {
	ctx              context.Context
	cancel           context.CancelFunc
	eventCallback    func(ProcessEvent)
	pollingInterval  time.Duration
	trackedProcesses map[int]*ProcessInfo
	mu               sync.RWMutex
	running          bool
	stats            MonitorStats
	claudePatterns   []*regexp.Regexp
	fileIOMonitor    *NonInvasiveFileMonitor
	fileIOCallback   func(FileIOEvent)
}

/**
 * CONTEXT:   Monitor statistics for performance tracking
 * INPUT:     Process scanning and event generation metrics
 * OUTPUT:    Performance statistics for monitoring optimization
 * BUSINESS:  Statistics enable monitor performance optimization
 * CHANGE:    Initial statistics structure
 * RISK:      Low - Performance metrics structure
 */
type MonitorStats struct {
	ScansPerformed     uint64        `json:"scans_performed"`
	ProcessesDetected  uint64        `json:"processes_detected"`
	EventsGenerated    uint64        `json:"events_generated"`
	LastScanTime       time.Time     `json:"last_scan_time"`
	StartTime          time.Time     `json:"start_time"`
	AverageScanTime    time.Duration `json:"average_scan_time"`
	ErrorCount         uint64        `json:"error_count"`
}

/**
 * CONTEXT:   Create new process monitor instance
 * INPUT:     Event callback function and polling configuration
 * OUTPUT:    Configured process monitor ready for activation
 * BUSINESS:  Monitor creation sets up Claude process detection infrastructure
 * CHANGE:    Enhanced Claude detection with specific patterns and confidence scoring
 * RISK:      Medium - Monitor initialization affecting system resources
 */
func NewProcessMonitor(callback func(ProcessEvent)) *ProcessMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Enhanced Claude detection patterns - more specific and accurate
	patterns := []*regexp.Regexp{
		// High priority Claude processes
		regexp.MustCompile(`^claude$`),                    // Claude CLI binary
		regexp.MustCompile(`^claude-code$`),               // Claude Code CLI
		regexp.MustCompile(`^claude-desktop$`),            // Claude Desktop app
		regexp.MustCompile(`^anthropic-claude$`),          // Official Anthropic Claude
		
		// Node.js Claude processes
		regexp.MustCompile(`node.*claude-code`),           // Node Claude Code
		regexp.MustCompile(`node.*@anthropic/claude`),     // Node Anthropic SDK
		regexp.MustCompile(`npm.*claude`),                 // NPM Claude packages
		
		// Python Claude processes  
		regexp.MustCompile(`python.*anthropic`),           // Python Anthropic SDK
		regexp.MustCompile(`python.*claude`),              // Python Claude scripts
		
		// Development processes
		regexp.MustCompile(`.*claude-monitor.*`),          // Our own monitor (for testing)
		regexp.MustCompile(`.*claude-daemon.*`),           // Our daemon processes
		
		// Browser processes running Claude
		regexp.MustCompile(`.*--app.*claude\.ai.*`),       // Browser app mode for Claude
		regexp.MustCompile(`.*--app.*anthropic.*`),        // Browser app mode for Anthropic
	}
	
	monitor := &ProcessMonitor{
		ctx:              ctx,
		cancel:           cancel,
		eventCallback:    callback,
		pollingInterval:  5 * time.Second, // Poll every 5 seconds
		trackedProcesses: make(map[int]*ProcessInfo),
		claudePatterns:   patterns,
		stats: MonitorStats{
			StartTime: time.Now(),
		},
	}
	
	return monitor
}

/**
 * CONTEXT:   Set file I/O event callback for work activity tracking
 * INPUT:     File I/O event callback function
 * OUTPUT:    Process monitor configured with file I/O monitoring
 * BUSINESS:  File I/O callback enables work activity event processing
 * CHANGE:    Initial file I/O callback configuration
 * RISK:      Low - Callback configuration for enhanced monitoring
 */
func (pm *ProcessMonitor) SetFileIOCallback(callback func(FileIOEvent)) {
	pm.fileIOCallback = callback
}

/**
 * CONTEXT:   Start process monitoring system
 * INPUT:     Monitor activation request
 * OUTPUT:    Active process monitoring with event generation
 * BUSINESS:  Start enables real-time Claude process detection
 * CHANGE:    Initial start implementation with polling loop
 * RISK:      High - System monitoring affecting performance
 */
func (pm *ProcessMonitor) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.running {
		return fmt.Errorf("process monitor is already running")
	}
	
	pm.running = true
	
	// Initialize NON-INVASIVE file I/O monitor if callback is configured (CRITICAL FIX)
	if pm.fileIOCallback != nil {
		pm.fileIOMonitor = NewNonInvasiveFileMonitor(pm.fileIOCallback)
		if err := pm.fileIOMonitor.Start(); err != nil {
			log.Printf("Warning: failed to start file I/O monitor: %v", err)
		} else {
			log.Printf("File I/O monitor started successfully")
		}
	}
	
	// Initial scan to populate current processes (async)
	go func() {
		if err := pm.scanProcesses(); err != nil {
			log.Printf("Warning: initial process scan failed: %v", err)
		}
	}()
	
	// Start monitoring goroutine
	go pm.monitorLoop()
	
	// Start process state monitoring from prototype
	go pm.startProcessStateMonitor()
	
	log.Printf("Process monitor started with %d detection patterns", len(pm.claudePatterns))
	return nil
}

/**
 * CONTEXT:   Stop process monitoring system
 * INPUT:     Monitor shutdown request
 * OUTPUT:    Cleanly stopped monitoring system
 * BUSINESS:  Stop enables graceful monitor shutdown
 * CHANGE:    Initial stop implementation
 * RISK:      Medium - Monitor shutdown affecting tracked processes
 */
func (pm *ProcessMonitor) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if !pm.running {
		return nil
	}
	
	// Stop file I/O monitor if running
	if pm.fileIOMonitor != nil {
		if err := pm.fileIOMonitor.Stop(); err != nil {
			log.Printf("Warning: failed to stop file I/O monitor: %v", err)
		}
	}
	
	pm.cancel()
	pm.running = false
	
	log.Printf("Process monitor stopped")
	return nil
}

/**
 * CONTEXT:   Main monitoring loop for process detection
 * INPUT:     Continuous process scanning and event detection
 * OUTPUT:    Process lifecycle events sent to callback
 * BUSINESS:  Monitor loop provides continuous Claude process tracking
 * CHANGE:    Initial monitoring loop with error handling
 * RISK:      High - Continuous monitoring affecting system performance
 */
func (pm *ProcessMonitor) monitorLoop() {
	ticker := time.NewTicker(pm.pollingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			if err := pm.scanProcesses(); err != nil {
				pm.mu.Lock()
				pm.stats.ErrorCount++
				pm.mu.Unlock()
				log.Printf("Process scan error: %v", err)
			}
		}
	}
}

/**
 * CONTEXT:   Scan system processes for Claude instances
 * INPUT:     System process table via ps command
 * OUTPUT:    Updated tracked processes and generated events
 * BUSINESS:  Process scanning detects Claude lifecycle changes
 * CHANGE:    Initial process scanning with ps command parsing
 * RISK:      Medium - System command execution affecting performance
 */
func (pm *ProcessMonitor) scanProcesses() error {
	scanStart := time.Now()
	
	// Get process list using ps command (simplified for speed)
	cmd := exec.Command("ps", "axo", "pid,ppid,comm,args,user")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute ps command: %w", err)
	}
	
	currentProcesses := make(map[int]*ProcessInfo)
	lines := strings.Split(string(output), "\n")
	
	// Skip header line
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		
		processInfo, err := pm.parseProcessLine(line)
		if err != nil {
			continue // Skip invalid lines
		}
		
		// Check if this is a Claude process
		if pm.isClaudeProcess(processInfo) {
			currentProcesses[processInfo.PID] = processInfo
		}
	}
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Detect new processes (started)
	for pid, processInfo := range currentProcesses {
		if _, exists := pm.trackedProcesses[pid]; !exists {
			// New Claude process detected
			event := ProcessEvent{
				Type:       ProcessStarted,
				PID:        processInfo.PID,
				PPID:       processInfo.PPID,
				Command:    processInfo.Command,
				Args:       processInfo.Args,
				WorkingDir: processInfo.WorkingDir,
				StartTime:  processInfo.StartTime,
				UserID:     processInfo.UserID,
				Timestamp:  time.Now(),
			}
			
			pm.stats.EventsGenerated++
			if pm.eventCallback != nil {
				go pm.eventCallback(event)
			}
		}
	}
	
	// Detect stopped processes
	for pid, processInfo := range pm.trackedProcesses {
		if _, exists := currentProcesses[pid]; !exists {
			// Claude process stopped
			event := ProcessEvent{
				Type:       ProcessStopped,
				PID:        processInfo.PID,
				PPID:       processInfo.PPID,
				Command:    processInfo.Command,
				Args:       processInfo.Args,
				WorkingDir: processInfo.WorkingDir,
				StartTime:  processInfo.StartTime,
				UserID:     processInfo.UserID,
				Timestamp:  time.Now(),
			}
			
			pm.stats.EventsGenerated++
			if pm.eventCallback != nil {
				go pm.eventCallback(event)
			}
		}
	}
	
	// Update tracked processes
	pm.trackedProcesses = currentProcesses
	pm.stats.ProcessesDetected = uint64(len(currentProcesses))
	pm.stats.ScansPerformed++
	pm.stats.LastScanTime = time.Now()
	
	// Update non-invasive file I/O monitor with new process list
	if pm.fileIOMonitor != nil {
		// NonInvasiveFileMonitor uses AttachToProcess/DetachFromProcess dynamically
		// This will be handled automatically by the process event callbacks
		log.Printf("File I/O monitor tracking %d Claude processes", len(currentProcesses))
	}
	
	scanDuration := time.Since(scanStart)
	pm.stats.AverageScanTime = time.Duration(
		(int64(pm.stats.AverageScanTime)*int64(pm.stats.ScansPerformed-1) + int64(scanDuration)) / 
		int64(pm.stats.ScansPerformed))
	
	return nil
}

/**
 * CONTEXT:   Parse process information from ps command output line
 * INPUT:     Single line of ps command output
 * OUTPUT:    Structured process information
 * BUSINESS:  Process parsing enables structured process tracking
 * CHANGE:    Initial process line parsing with field extraction
 * RISK:      Medium - Parsing robustness affecting process detection
 */
func (pm *ProcessMonitor) parseProcessLine(line string) (*ProcessInfo, error) {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return nil, fmt.Errorf("insufficient fields in process line")
	}
	
	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return nil, fmt.Errorf("invalid PID: %s", fields[0])
	}
	
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("invalid PPID: %s", fields[1])
	}
	
	command := fields[2]
	userID := fields[len(fields)-1]
	
	// Parse arguments (everything between command and user)
	var args []string
	if len(fields) > 5 {
		argsText := strings.Join(fields[3:len(fields)-1], " ")
		args = strings.Fields(argsText)
	}
	
	// Get working directory (best effort)
	workingDir := pm.getProcessWorkingDir(pid)
	
	processInfo := &ProcessInfo{
		PID:        pid,
		PPID:       ppid,
		Command:    command,
		Args:       args,
		WorkingDir: workingDir,
		StartTime:  time.Now(), // Approximate start time
		UserID:     userID,
	}
	
	return processInfo, nil
}

/**
 * CONTEXT:   Get working directory for a process
 * INPUT:     Process ID
 * OUTPUT:    Working directory path or empty string
 * BUSINESS:  Working directory provides project context for work tracking
 * CHANGE:    Initial working directory detection via /proc filesystem
 * RISK:      Low - Working directory detection with error handling
 */
func (pm *ProcessMonitor) getProcessWorkingDir(pid int) string {
	// Try to read working directory from /proc
	cwdPath := fmt.Sprintf("/proc/%d/cwd", pid)
	if target, err := os.Readlink(cwdPath); err == nil {
		return target
	}
	
	// Fallback: try to get from environment
	environPath := fmt.Sprintf("/proc/%d/environ", pid)
	if data, err := os.ReadFile(environPath); err == nil {
		environ := strings.Split(string(data), "\x00")
		for _, env := range environ {
			if strings.HasPrefix(env, "PWD=") {
				return strings.TrimPrefix(env, "PWD=")
			}
		}
	}
	
	return ""
}

/**
 * CONTEXT:   Multi-factor Claude process identification with confidence scoring
 * INPUT:     Process information structure
 * OUTPUT:    Boolean indicating if process is Claude with confidence data
 * BUSINESS:  Enhanced process identification prevents false positives in monitoring
 * CHANGE:    Multi-factor analysis with confidence scoring and detailed logging
 * RISK:      Medium - Process identification accuracy affecting monitoring quality
 */
func (pm *ProcessMonitor) isClaudeProcess(processInfo *ProcessInfo) bool {
	confidence := 0.0
	reasons := make([]string, 0)
	
	// Factor 1: Command name patterns (40% weight)
	for _, pattern := range pm.claudePatterns {
		if pattern.MatchString(processInfo.Command) {
			confidence += 0.4
			reasons = append(reasons, fmt.Sprintf("command_match:%s", processInfo.Command))
			break
		}
	}
	
	// Factor 2: Arguments analysis (30% weight)
	for _, arg := range processInfo.Args {
		for _, pattern := range pm.claudePatterns {
			if pattern.MatchString(arg) {
				confidence += 0.3
				reasons = append(reasons, fmt.Sprintf("args_match:%s", arg))
				goto args_done
			}
		}
	}
args_done:
	
	// Factor 3: Working directory analysis (20% weight)
	if processInfo.WorkingDir != "" {
		// Check if working directory contains Claude-related files
		if pm.hasClaudeProjectStructure(processInfo.WorkingDir) {
			confidence += 0.2
			reasons = append(reasons, fmt.Sprintf("workdir_structure:%s", processInfo.WorkingDir))
		}
		
		// Check working directory path patterns
		for _, pattern := range pm.claudePatterns {
			if pattern.MatchString(processInfo.WorkingDir) {
				confidence += 0.1
				reasons = append(reasons, fmt.Sprintf("workdir_pattern:%s", processInfo.WorkingDir))
				break
			}
		}
	}
	
	// Factor 4: Process tree analysis (10% weight)
	if pm.hasClaudeParentProcess(processInfo.PPID) {
		confidence += 0.1
		reasons = append(reasons, fmt.Sprintf("parent_claude:%d", processInfo.PPID))
	}
	
	// Confidence threshold for Claude process identification
	isClaudeProc := confidence >= 0.5
	
	// Log detailed analysis for debugging
	if isClaudeProc {
		log.Printf("🎯 Claude process detected: %s (PID:%d) - Confidence: %.2f - Reasons: %v", 
			processInfo.Command, processInfo.PID, confidence, reasons)
	} else if confidence > 0.2 {
		log.Printf("🤔 Potential Claude process (low confidence): %s (PID:%d) - Confidence: %.2f - Reasons: %v", 
			processInfo.Command, processInfo.PID, confidence, reasons)
	}
	
	return isClaudeProc
}

/**
 * CONTEXT:   Get current monitor statistics
 * INPUT:     Statistics request
 * OUTPUT:    Current monitoring performance metrics
 * BUSINESS:  Statistics provide monitoring system health information
 * CHANGE:    Initial statistics getter
 * RISK:      Low - Read-only statistics access
 */
func (pm *ProcessMonitor) GetStats() MonitorStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.stats
}

/**
 * CONTEXT:   Get currently tracked Claude processes
 * INPUT:     Process tracking request
 * OUTPUT:    Map of active Claude processes
 * BUSINESS:  Process tracking enables multi-instance monitoring
 * CHANGE:    Initial process tracking getter
 * RISK:      Low - Read-only process information access
 */
func (pm *ProcessMonitor) GetTrackedProcesses() map[int]*ProcessInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	// Return a copy to prevent concurrent access issues
	result := make(map[int]*ProcessInfo)
	for pid, info := range pm.trackedProcesses {
		result[pid] = info
	}
	
	return result
}

/**
 * CONTEXT:   Get file I/O monitoring statistics
 * INPUT:     File I/O statistics request
 * OUTPUT:    Current file I/O monitoring metrics or nil if not available
 * BUSINESS:  File I/O statistics provide work activity insights
 * CHANGE:    Initial file I/O statistics getter
 * RISK:      Low - Read-only statistics access
 */
func (pm *ProcessMonitor) GetFileIOStats() *FileIOStats {
	if pm.fileIOMonitor == nil {
		return nil
	}
	
	stats := pm.fileIOMonitor.GetStats()
	return &stats
}

/**
 * CONTEXT:   Set polling interval for process scanning
 * INPUT:     New polling interval duration
 * OUTPUT:    Updated monitor configuration
 * BUSINESS:  Polling configuration enables performance optimization
 * CHANGE:    Initial polling interval setter
 * RISK:      Low - Configuration update with validation
 */
func (pm *ProcessMonitor) SetPollingInterval(interval time.Duration) error {
	if interval < 1*time.Second {
		return fmt.Errorf("polling interval must be at least 1 second")
	}
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.pollingInterval = interval
	return nil
}

/**
 * CONTEXT:   Check if working directory has Claude project structure
 * INPUT:     Working directory path
 * OUTPUT:    Boolean indicating Claude project presence
 * BUSINESS:  Project structure analysis improves Claude process detection accuracy
 * CHANGE:    Initial project structure detection for Claude identification
 * RISK:      Low - File system analysis for process identification
 */
func (pm *ProcessMonitor) hasClaudeProjectStructure(workingDir string) bool {
	if workingDir == "" {
		return false
	}
	
	// Check for Claude-specific files and directories
	claudeIndicators := []string{
		".claude",           // Claude configuration directory
		"claude.md",         // Claude instructions file
		"CLAUDE.md",         // Common Claude instructions file
		".anthropic",        // Anthropic configuration
		"claude.toml",       // Claude configuration file
		"claude.json",       // Claude configuration file
	}
	
	for _, indicator := range claudeIndicators {
		if _, err := os.Stat(filepath.Join(workingDir, indicator)); err == nil {
			return true
		}
	}
	
	// Check for Claude-related package files
	packageFiles := []string{
		"package.json",     // Check for anthropic/claude packages
		"requirements.txt", // Check for anthropic packages
		"Pipfile",         // Check for anthropic packages
		"pyproject.toml",  // Check for anthropic packages
		"go.mod",          // Check for Claude-related Go modules
	}
	
	for _, packageFile := range packageFiles {
		if content, err := os.ReadFile(filepath.Join(workingDir, packageFile)); err == nil {
			contentStr := strings.ToLower(string(content))
			if strings.Contains(contentStr, "anthropic") || strings.Contains(contentStr, "claude") {
				return true
			}
		}
	}
	
	return false
}

/**
 * CONTEXT:   Check if parent process is Claude-related
 * INPUT:     Parent process ID
 * OUTPUT:    Boolean indicating if parent is Claude process
 * BUSINESS:  Parent process analysis improves Claude child process detection
 * CHANGE:    Initial parent process analysis for process tree detection
 * RISK:      Low - Process tree analysis for identification
 */
func (pm *ProcessMonitor) hasClaudeParentProcess(ppid int) bool {
	if ppid <= 1 {
		return false // Skip init and kernel processes
	}
	
	// Read parent process command
	cmdlineFile := fmt.Sprintf("/proc/%d/cmdline", ppid)
	if cmdline, err := os.ReadFile(cmdlineFile); err == nil {
		cmdlineStr := strings.ToLower(string(cmdline))
		
		// Check if parent command matches Claude patterns
		for _, pattern := range pm.claudePatterns {
			if pattern.MatchString(cmdlineStr) {
				return true
			}
		}
		
		// Check for common Claude parent processes
		claudeParents := []string{
			"claude", "claude-code", "claude-desktop",
			"node", "python", "npm", "anthropic",
		}
		
		for _, parent := range claudeParents {
			if strings.Contains(cmdlineStr, parent) {
				return true
			}
		}
	}
	
	return false
}

/**
 * CONTEXT:   Convert ProcessEvent to string representation
 * INPUT:     Process event structure
 * OUTPUT:    Human-readable event description
 * BUSINESS:  String conversion enables debugging and logging
 * CHANGE:    Initial string conversion for process events
 * RISK:      Low - Display utility function
 */
func (pe ProcessEvent) String() string {
	return fmt.Sprintf("ProcessEvent{Type:%s, PID:%d, Command:%s, WorkingDir:%s, Timestamp:%s}",
		pe.Type, pe.PID, pe.Command, pe.WorkingDir, pe.Timestamp.Format("15:04:05"))
}

/**
 * CONTEXT:   Process state monitoring from prototype for process health tracking
 * INPUT:     Continuous process existence monitoring for tracked processes
 * OUTPUT:    Process termination detection and cleanup events
 * BUSINESS:  Process state monitoring ensures reliable process lifecycle tracking
 * CHANGE:    Process state monitoring from prototype with enhanced tracking
 * RISK:      Low - Process existence monitoring with minimal system impact
 */
func (pm *ProcessMonitor) startProcessStateMonitor() {
	ticker := time.NewTicker(5 * time.Second) // Check process state every 5 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.checkProcessStates()
		}
	}
}

/**
 * CONTEXT:   Check state of all tracked processes
 * INPUT:     Current tracked process list
 * OUTPUT:    Process state validation and cleanup for terminated processes
 * BUSINESS:  Process state checking ensures accurate process lifecycle tracking
 * CHANGE:    Enhanced process state checking from prototype
 * RISK:      Low - Process existence checking with error handling
 */
func (pm *ProcessMonitor) checkProcessStates() {
	pm.mu.RLock()
	trackedPIDs := make([]int, 0, len(pm.trackedProcesses))
	for pid := range pm.trackedProcesses {
		trackedPIDs = append(trackedPIDs, pid)
	}
	pm.mu.RUnlock()
	
	var terminatedPIDs []int
	
	// Check each tracked process
	for _, pid := range trackedPIDs {
		if !pm.processExists(pid) {
			terminatedPIDs = append(terminatedPIDs, pid)
		}
	}
	
	// Handle terminated processes
	if len(terminatedPIDs) > 0 {
		pm.handleTerminatedProcesses(terminatedPIDs)
	}
}

/**
 * CONTEXT:   Check if process exists using /proc filesystem
 * INPUT:     Process ID
 * OUTPUT:    Boolean indicating if process exists
 * BUSINESS:  Process existence check enables reliable process lifecycle tracking
 * CHANGE:    Process existence check from prototype
 * RISK:      Low - File system check for process existence
 */
func (pm *ProcessMonitor) processExists(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

/**
 * CONTEXT:   Handle terminated processes cleanup and event generation
 * INPUT:     List of terminated process IDs
 * OUTPUT:    Process termination events and cleanup
 * BUSINESS:  Terminated process handling ensures proper cleanup and event tracking
 * CHANGE:    Enhanced terminated process handling with cleanup
 * RISK:      Low - Process cleanup and event generation
 */
func (pm *ProcessMonitor) handleTerminatedProcesses(terminatedPIDs []int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for _, pid := range terminatedPIDs {
		if processInfo, exists := pm.trackedProcesses[pid]; exists {
			// Generate process stopped event
			event := ProcessEvent{
				Type:       ProcessStopped,
				PID:        processInfo.PID,
				PPID:       processInfo.PPID,
				Command:    processInfo.Command,
				Args:       processInfo.Args,
				WorkingDir: processInfo.WorkingDir,
				StartTime:  processInfo.StartTime,
				UserID:     processInfo.UserID,
				Timestamp:  time.Now(),
			}
			
			// Remove from tracked processes
			delete(pm.trackedProcesses, pid)
			
			pm.stats.EventsGenerated++
			
			log.Printf("🔴 Claude process terminated: %s (PID: %d)", processInfo.Command, pid)
			
			// Send event to callback
			if pm.eventCallback != nil {
				go pm.eventCallback(event)
			}
		}
	}
	
	// Update statistics
	pm.stats.ProcessesDetected = uint64(len(pm.trackedProcesses))
	
	// NonInvasiveFileMonitor handles process tracking automatically via events
	// No manual process list updates needed
}