/**
 * ⚠️  DEPRECATED - DANGEROUS STRACE-BASED MONITOR - DO NOT USE ⚠️
 * 
 * This file contains FileIOMonitor which uses invasive strace monitoring that:
 * - Causes zombie processes that cannot be killed
 * - Locks TCP ports permanently 
 * - Uses ptrace() which blocks process termination
 * - Interferes with Claude process execution
 * 
 * ✅ USE INSTEAD: NonInvasiveFileMonitor in non_invasive_file_monitor.go
 * 
 * CONTEXT:   DEPRECATED - Invasive strace-based file monitoring (CAUSES ZOMBIES)
 * INPUT:     File system operations via dangerous strace attachment
 * OUTPUT:    Work-related events BUT CREATES UNKILLABLE ZOMBIE PROCESSES
 * BUSINESS:  DO NOT USE - Replaced by NonInvasiveFileMonitor
 * CHANGE:    DEPRECATED - Marked dangerous to prevent accidental usage
 * RISK:      CRITICAL - CAUSES ZOMBIE PROCESSES AND RESOURCE LEAKS
 */

package monitor

import (
	"bufio"
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
 * CONTEXT:   File I/O event types for work activity classification
 * INPUT:     File operation events from system monitoring
 * OUTPUT:    Classified file events for work tracking
 * BUSINESS:  Event classification enables work productivity analysis
 * CHANGE:    Initial file I/O event types for Claude monitoring
 * RISK:      Low - Event type definitions for file operations
 */
type FileIOEventType string

const (
	FileIORead   FileIOEventType = "FILE_READ"
	FileIOWrite  FileIOEventType = "FILE_WRITE" 
	FileIOOpen   FileIOEventType = "FILE_OPEN"
	FileIOCreate FileIOEventType = "FILE_CREATE"
	FileIODelete FileIOEventType = "FILE_DELETE"
	FileIOModify FileIOEventType = "FILE_MODIFY"
)

type FileIOEvent struct {
	Type        FileIOEventType        `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	PID         int                    `json:"pid"`
	ProcessName string                 `json:"process_name"`
	FilePath    string                 `json:"file_path"`
	ProjectPath string                 `json:"project_path"`
	ProjectName string                 `json:"project_name"`
	FileType    string                 `json:"file_type"`
	Size        int64                  `json:"size"`
	IsWorkFile  bool                   `json:"is_work_file"`
	Details     map[string]interface{} `json:"details"`
}

/**
 * CONTEXT:   Selective file I/O monitor for Claude work detection
 * INPUT:     Claude process list and file system monitoring
 * OUTPUT:    Work-related file I/O events
 * BUSINESS:  Selective monitoring focuses on productive work activities
 * CHANGE:    Initial selective file monitor with work pattern detection
 * RISK:      High - File system monitoring affecting system performance
 */
type FileIOMonitor struct {
	ctx              context.Context
	cancel           context.CancelFunc
	eventCallback    func(FileIOEvent)
	trackedProcesses map[int]*ProcessInfo
	workFilePatterns []*regexp.Regexp
	excludePatterns  []*regexp.Regexp
	mu               sync.RWMutex
	running          bool
	stats            FileIOStats
}

/**
 * CONTEXT:   File I/O monitoring statistics
 * INPUT:     File operation monitoring metrics
 * OUTPUT:    Performance and activity statistics
 * BUSINESS:  Statistics enable monitoring optimization and work insights
 * CHANGE:    Initial file I/O statistics structure
 * RISK:      Low - Statistics structure for monitoring metrics
 */
type FileIOStats struct {
	TotalEvents      uint64    `json:"total_events"`
	WorkFileEvents   uint64    `json:"work_file_events"`
	EventsByType     map[string]uint64 `json:"events_by_type"`
	EventsByProject  map[string]uint64 `json:"events_by_project"`
	LastEventTime    time.Time `json:"last_event_time"`
	StartTime        time.Time `json:"start_time"`
	ErrorCount       uint64    `json:"error_count"`
}

/**
 * CONTEXT:   Create new file I/O monitor for Claude processes
 * INPUT:     Event callback and monitored processes
 * OUTPUT:    Configured file I/O monitor ready for activation
 * BUSINESS:  Monitor creation sets up selective file monitoring for work tracking
 * CHANGE:    Initial file I/O monitor constructor with work pattern detection
 * RISK:      Medium - Monitor initialization with file system access
 */
func NewFileIOMonitor(callback func(FileIOEvent)) *FileIOMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Work file patterns - files that indicate productive work
	workPatterns := []*regexp.Regexp{
		// Source code files
		regexp.MustCompile(`.*\.(go|py|js|ts|jsx|tsx|java|cpp|c|h|cs|rb|php|swift|kt|rs)$`),
		
		// Configuration files
		regexp.MustCompile(`.*\.(json|yaml|yml|toml|ini|conf|config)$`),
		
		// Documentation files
		regexp.MustCompile(`.*\.(md|txt|rst|adoc|tex|doc|docx)$`),
		
		// Data files
		regexp.MustCompile(`.*\.(sql|csv|xml|html|css|scss|less)$`),
		
		// Build and project files
		regexp.MustCompile(`.*(Makefile|Dockerfile|docker-compose|package\.json|requirements\.txt|go\.mod|Cargo\.toml|pom\.xml)$`),
		
		// Claude-specific files
		regexp.MustCompile(`.*(CLAUDE\.md|claude\.md|\.claude/.*)$`),
	}
	
	// Exclude patterns - files to ignore for work tracking
	excludePatterns := []*regexp.Regexp{
		// System and cache files
		regexp.MustCompile(`.*/\.git/.*`),
		regexp.MustCompile(`.*/node_modules/.*`),
		regexp.MustCompile(`.*/\.cache/.*`),
		regexp.MustCompile(`.*/__pycache__/.*`),
		regexp.MustCompile(`.*/target/.*`),
		regexp.MustCompile(`.*/build/.*`),
		regexp.MustCompile(`.*/dist/.*`),
		
		// Log and temporary files
		regexp.MustCompile(`.*\.(log|tmp|temp|bak|swp|~)$`),
		regexp.MustCompile(`.*/\..*\.swp$`),
		
		// Binary and media files
		regexp.MustCompile(`.*\.(exe|bin|so|dll|dylib|a|o)$`),
		regexp.MustCompile(`.*\.(jpg|jpeg|png|gif|bmp|ico|svg|mp3|mp4|avi|mov|wav)$`),
		regexp.MustCompile(`.*\.(zip|tar|gz|rar|7z|bz2|xz)$`),
	}
	
	monitor := &FileIOMonitor{
		ctx:              ctx,
		cancel:           cancel,
		eventCallback:    callback,
		trackedProcesses: make(map[int]*ProcessInfo),
		workFilePatterns: workPatterns,
		excludePatterns:  excludePatterns,
		stats: FileIOStats{
			EventsByType:    make(map[string]uint64),
			EventsByProject: make(map[string]uint64),
			StartTime:       time.Now(),
		},
	}
	
	return monitor
}

/**
 * CONTEXT:   Start file I/O monitoring for tracked Claude processes
 * INPUT:     Monitor activation request
 * OUTPUT:    Active file I/O monitoring with work event generation
 * BUSINESS:  Monitor start enables real-time work activity detection
 * CHANGE:    Initial monitor start with strace-based file monitoring
 * RISK:      High - File system monitoring affecting system performance
 */
func (fio *FileIOMonitor) Start() error {
	fio.mu.Lock()
	defer fio.mu.Unlock()
	
	if fio.running {
		return fmt.Errorf("file I/O monitor is already running")
	}
	
	fio.running = true
	
	// Start multiple monitoring methods
	go fio.monitorLoop()        // Process-based monitoring
	go fio.monitorWithStrace()  // Advanced strace monitoring
	go fio.monitorOpenFiles()   // Open files tracking
	
	log.Printf("File I/O monitor started with %d work patterns (strace: %t)", 
		len(fio.workFilePatterns), fio.canUseStrace())
	return nil
}

/**
 * CONTEXT:   Stop file I/O monitoring system
 * INPUT:     Monitor shutdown request
 * OUTPUT:    Cleanly stopped file monitoring
 * BUSINESS:  Monitor stop enables graceful shutdown
 * CHANGE:    Initial stop implementation with cleanup
 * RISK:      Medium - Monitor shutdown affecting tracked processes
 */
func (fio *FileIOMonitor) Stop() error {
	fio.mu.Lock()
	defer fio.mu.Unlock()
	
	if !fio.running {
		return nil
	}
	
	fio.cancel()
	fio.running = false
	
	log.Printf("File I/O monitor stopped")
	return nil
}

/**
 * CONTEXT:   Dynamically attach File I/O monitoring to specific process
 * INPUT:     Process ID to monitor
 * OUTPUT:    Process-specific file monitoring attachment
 * BUSINESS:  Dynamic attachment enables per-process work file tracking
 * CHANGE:    Event-driven process attachment for real-time monitoring
 * RISK:      High - Dynamic strace attachment affecting system performance
 */
func (fio *FileIOMonitor) AttachToProcess(pid int) error {
	fio.mu.Lock()
	defer fio.mu.Unlock()
	
	if !fio.running {
		return fmt.Errorf("File I/O monitor not running")
	}
	
	// Check if already attached
	if _, exists := fio.trackedProcesses[pid]; exists {
		return nil // Already attached
	}
	
	// Get process information
	processInfo, err := fio.getProcessInfo(pid)
	if err != nil {
		return fmt.Errorf("failed to get process info for PID %d: %w", pid, err)
	}
	
	// Add to tracked processes
	fio.trackedProcesses[pid] = processInfo
	
	// Start strace monitoring for this specific process
	if fio.canUseStrace() {
		go fio.startStraceForProcess(pid, processInfo)
	}
	
	log.Printf("Attached File I/O monitoring to PID %d (%s)", pid, processInfo.Command)
	return nil
}

/**
 * CONTEXT:   Dynamically detach File I/O monitoring from specific process
 * INPUT:     Process ID to stop monitoring
 * OUTPUT:    Process-specific file monitoring detachment and cleanup
 * BUSINESS:  Dynamic detachment prevents resource leaks from stopped processes
 * CHANGE:    Event-driven process detachment for resource cleanup
 * RISK:      Medium - Process detachment affecting monitoring coverage
 */
func (fio *FileIOMonitor) DetachFromProcess(pid int) error {
	fio.mu.Lock()
	defer fio.mu.Unlock()
	
	// Check if attached
	processInfo, exists := fio.trackedProcesses[pid]
	if !exists {
		return nil // Not attached
	}
	
	// Remove from tracked processes
	delete(fio.trackedProcesses, pid)
	
	// Clean up any strace processes for this PID
	fio.stopStraceForProcess(pid)
	
	log.Printf("Detached File I/O monitoring from PID %d (%s)", pid, processInfo.Command)
	return nil
}

/**
 * CONTEXT:   Update tracked processes for file I/O monitoring
 * INPUT:     Map of Claude processes to monitor
 * OUTPUT:    Updated monitoring targets
 * BUSINESS:  Process tracking enables selective file monitoring
 * CHANGE:    Initial process tracking update for file monitoring
 * RISK:      Low - Process list update with synchronization
 */
func (fio *FileIOMonitor) UpdateTrackedProcesses(processes map[int]*ProcessInfo) {
	fio.mu.Lock()
	defer fio.mu.Unlock()
	
	fio.trackedProcesses = make(map[int]*ProcessInfo)
	for pid, info := range processes {
		fio.trackedProcesses[pid] = info
	}
	
	log.Printf("File I/O monitor tracking %d Claude processes", len(fio.trackedProcesses))
}

/**
 * CONTEXT:   Main file I/O monitoring loop
 * INPUT:     Continuous file system monitoring for tracked processes
 * OUTPUT:    File I/O events sent to callback
 * BUSINESS:  Monitor loop provides continuous work activity detection
 * CHANGE:    Initial monitoring loop with selective file tracking
 * RISK:      High - Continuous monitoring affecting system performance
 */
func (fio *FileIOMonitor) monitorLoop() {
	ticker := time.NewTicker(2 * time.Second) // Check for new processes to monitor
	defer ticker.Stop()
	
	monitoredPIDs := make(map[int]*exec.Cmd)
	
	for {
		select {
		case <-fio.ctx.Done():
			// Stop all running monitors
			for _, cmd := range monitoredPIDs {
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
			}
			return
		case <-ticker.C:
			fio.updateProcessMonitoring(monitoredPIDs)
		}
	}
}

/**
 * CONTEXT:   Update process-specific file monitoring
 * INPUT:     Current monitored processes and new process list
 * OUTPUT:    Updated process monitoring with new strace instances
 * BUSINESS:  Process monitoring update maintains work activity tracking
 * CHANGE:    Initial process monitoring update with strace management
 * RISK:      Medium - Process monitoring affecting system resources
 */
func (fio *FileIOMonitor) updateProcessMonitoring(monitoredPIDs map[int]*exec.Cmd) {
	fio.mu.RLock()
	currentPIDs := make(map[int]*ProcessInfo)
	for pid, info := range fio.trackedProcesses {
		currentPIDs[pid] = info
	}
	fio.mu.RUnlock()
	
	// Stop monitoring processes that are no longer tracked
	for pid, cmd := range monitoredPIDs {
		if _, exists := currentPIDs[pid]; !exists {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			delete(monitoredPIDs, pid)
			log.Printf("Stopped file I/O monitoring for PID %d", pid)
		}
	}
	
	// Start monitoring new processes
	for pid, processInfo := range currentPIDs {
		if _, exists := monitoredPIDs[pid]; !exists {
			if fio.canUseStrace() {
				if cmd := fio.startStraceMonitoring(pid, processInfo); cmd != nil {
					monitoredPIDs[pid] = cmd
					log.Printf("Started file I/O monitoring for %s (PID: %d)", processInfo.Command, pid)
				}
			}
		}
	}
}

/**
 * CONTEXT:   Start strace monitoring for specific Claude process
 * INPUT:     Process ID and process information
 * OUTPUT:    Running strace command for file monitoring
 * BUSINESS:  Strace monitoring provides detailed file operation tracking
 * CHANGE:    Initial strace-based file monitoring for work detection
 * RISK:      High - Strace monitoring affecting process performance
 */
func (fio *FileIOMonitor) startStraceMonitoring(pid int, processInfo *ProcessInfo) *exec.Cmd {
	// Use strace to monitor file operations
	cmd := exec.Command("strace", "-p", strconv.Itoa(pid),
		"-e", "trace=openat,read,write,close,unlink,rename,mkdir,rmdir",
		"-f", "-q", "-s", "200", "-o", "/dev/stderr")
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to create strace stderr pipe for PID %d: %v", pid, err)
		return nil
	}
	
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start strace for PID %d: %v", pid, err)
		return nil
	}
	
	// Process strace output in goroutine
	go func() {
		defer stderr.Close()
		scanner := bufio.NewScanner(stderr)
		
		for scanner.Scan() {
			line := scanner.Text()
			if event := fio.parseStraceOutput(line, pid, processInfo); event != nil {
				fio.processFileEvent(*event)
			}
		}
	}()
	
	return cmd
}

/**
 * CONTEXT:   Parse strace output to extract file operations
 * INPUT:     Strace output line and process information
 * OUTPUT:    Parsed file I/O event or nil if not relevant
 * BUSINESS:  Strace parsing extracts meaningful file operations for work tracking
 * CHANGE:    Initial strace output parsing with work file detection
 * RISK:      Medium - Parsing accuracy affecting event quality
 */
func (fio *FileIOMonitor) parseStraceOutput(line string, pid int, processInfo *ProcessInfo) *FileIOEvent {
	// Parse different strace patterns
	
	// openat pattern: openat(AT_FDCWD, "/path/to/file", O_RDONLY) = 3
	openPattern := regexp.MustCompile(`openat\([^,]+,\s*"([^"]+)".*\)\s*=\s*(\d+|[^\d].*)`)
	if matches := openPattern.FindStringSubmatch(line); len(matches) >= 3 {
		filePath := matches[1]
		if fio.isWorkFile(filePath) {
			return &FileIOEvent{
				Type:        FileIOOpen,
				Timestamp:   time.Now(),
				PID:         pid,
				ProcessName: processInfo.Command,
				FilePath:    filePath,
				ProjectPath: fio.extractProjectPath(filePath, processInfo.WorkingDir),
				ProjectName: fio.extractProjectName(filePath, processInfo.WorkingDir),
				FileType:    filepath.Ext(filePath),
				IsWorkFile:  true,
				Details: map[string]interface{}{
					"operation": "open",
					"result":    matches[2],
				},
			}
		}
	}
	
	// write pattern: write(3, "content", 7) = 7
	writePattern := regexp.MustCompile(`write\((\d+),.*\)\s*=\s*(\d+)`)
	if matches := writePattern.FindStringSubmatch(line); len(matches) >= 3 {
		size, _ := strconv.ParseInt(matches[2], 10, 64)
		if size > 0 {
			return &FileIOEvent{
				Type:        FileIOWrite,
				Timestamp:   time.Now(),
				PID:         pid,
				ProcessName: processInfo.Command,
				Size:        size,
				IsWorkFile:  true, // Assume writes are work-related
				Details: map[string]interface{}{
					"operation": "write",
					"fd":        matches[1],
					"size":      size,
				},
			}
		}
	}
	
	// read pattern: read(3, "content", 1024) = 7
	readPattern := regexp.MustCompile(`read\((\d+),.*\)\s*=\s*(\d+)`)
	if matches := readPattern.FindStringSubmatch(line); len(matches) >= 3 {
		size, _ := strconv.ParseInt(matches[2], 10, 64)
		if size > 0 {
			return &FileIOEvent{
				Type:        FileIORead,
				Timestamp:   time.Now(),
				PID:         pid,
				ProcessName: processInfo.Command,
				Size:        size,
				IsWorkFile:  true, // Assume reads are work-related
				Details: map[string]interface{}{
					"operation": "read",
					"fd":        matches[1],
					"size":      size,
				},
			}
		}
	}
	
	return nil
}

/**
 * CONTEXT:   Check if file path represents work-related file
 * INPUT:     File path string
 * OUTPUT:    Boolean indicating if file is work-related
 * BUSINESS:  Work file detection focuses monitoring on productive activities
 * CHANGE:    Initial work file detection with pattern matching
 * RISK:      Medium - File classification accuracy affecting work tracking
 */
func (fio *FileIOMonitor) isWorkFile(filePath string) bool {
	// Skip files that should be excluded
	for _, pattern := range fio.excludePatterns {
		if pattern.MatchString(filePath) {
			return false
		}
	}
	
	// Check if file matches work patterns
	for _, pattern := range fio.workFilePatterns {
		if pattern.MatchString(filePath) {
			return true
		}
	}
	
	return false
}

/**
 * CONTEXT:   Extract project path from file path
 * INPUT:     File path and process working directory
 * OUTPUT:    Project root path
 * BUSINESS:  Project path extraction enables work organization
 * CHANGE:    Initial project path extraction from file operations
 * RISK:      Low - Path analysis for project identification
 */
func (fio *FileIOMonitor) extractProjectPath(filePath, workingDir string) string {
	if workingDir != "" && strings.HasPrefix(filePath, workingDir) {
		return workingDir
	}
	
	// Try to find project root by looking for common project indicators
	dir := filepath.Dir(filePath)
	for dir != "/" && dir != "." {
		// Check for project indicators
		indicators := []string{".git", ".gitignore", "package.json", "go.mod", "Cargo.toml", 
			"requirements.txt", "pom.xml", "Makefile", "CLAUDE.md"}
		
		for _, indicator := range indicators {
			if _, err := os.Stat(filepath.Join(dir, indicator)); err == nil {
				return dir
			}
		}
		
		dir = filepath.Dir(dir)
	}
	
	return workingDir // Fallback to working directory
}

/**
 * CONTEXT:   Extract project name from file path
 * INPUT:     File path and process working directory
 * OUTPUT:    Project name
 * BUSINESS:  Project name extraction enables work categorization
 * CHANGE:    Initial project name extraction from paths
 * RISK:      Low - Name extraction utility for project identification
 */
func (fio *FileIOMonitor) extractProjectName(filePath, workingDir string) string {
	projectPath := fio.extractProjectPath(filePath, workingDir)
	if projectPath != "" {
		return filepath.Base(projectPath)
	}
	return "unknown"
}

/**
 * CONTEXT:   Process file I/O event and send to callback
 * INPUT:     File I/O event with work classification
 * OUTPUT:    Processed event sent to callback
 * BUSINESS:  Event processing enables work activity tracking
 * CHANGE:    Initial event processing with statistics update
 * RISK:      Low - Event processing with statistics tracking
 */
func (fio *FileIOMonitor) processFileEvent(event FileIOEvent) {
	fio.mu.Lock()
	fio.stats.TotalEvents++
	if event.IsWorkFile {
		fio.stats.WorkFileEvents++
	}
	fio.stats.EventsByType[string(event.Type)]++
	if event.ProjectName != "" {
		fio.stats.EventsByProject[event.ProjectName]++
	}
	fio.stats.LastEventTime = time.Now()
	fio.mu.Unlock()
	
	// Send event to callback
	if fio.eventCallback != nil {
		go fio.eventCallback(event)
	}
}

/**
 * CONTEXT:   Check if strace is available for file monitoring
 * INPUT:     System command availability check
 * OUTPUT:    Boolean indicating strace availability
 * BUSINESS:  Strace availability determines monitoring method
 * CHANGE:    Initial strace availability check
 * RISK:      Low - System command availability check
 */
func (fio *FileIOMonitor) canUseStrace() bool {
	_, err := exec.LookPath("strace")
	return err == nil
}

/**
 * CONTEXT:   Get file I/O monitoring statistics
 * INPUT:     Statistics request
 * OUTPUT:    Current file I/O monitoring metrics
 * BUSINESS:  Statistics provide monitoring system health information
 * CHANGE:    Initial statistics getter for file I/O monitor
 * RISK:      Low - Read-only statistics access
 */
func (fio *FileIOMonitor) GetStats() FileIOStats {
	fio.mu.RLock()
	defer fio.mu.RUnlock()
	
	// Create a copy to prevent concurrent access issues
	stats := fio.stats
	stats.EventsByType = make(map[string]uint64)
	stats.EventsByProject = make(map[string]uint64)
	
	for k, v := range fio.stats.EventsByType {
		stats.EventsByType[k] = v
	}
	for k, v := range fio.stats.EventsByProject {
		stats.EventsByProject[k] = v
	}
	
	return stats
}

/**
 * CONTEXT:   Advanced strace monitoring for all tracked processes
 * INPUT:     Continuous strace monitoring across all Claude processes
 * OUTPUT:    Detailed file I/O events from strace analysis
 * BUSINESS:  Strace monitoring provides comprehensive file operation tracking
 * CHANGE:    Advanced strace integration from prototype for precise monitoring
 * RISK:      High - Intensive strace monitoring affecting system performance
 */
func (fio *FileIOMonitor) monitorWithStrace() {
	if !fio.canUseStrace() {
		log.Printf("strace not available, skipping advanced file monitoring")
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	activeStrace := make(map[int]*exec.Cmd)
	
	for {
		select {
		case <-fio.ctx.Done():
			// Stop all strace processes
			for _, cmd := range activeStrace {
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
			}
			return
		case <-ticker.C:
			fio.updateStraceMonitoring(activeStrace)
		}
	}
}

/**
 * CONTEXT:   Monitor open files for tracked processes
 * INPUT:     Process file descriptor monitoring
 * OUTPUT:    File open/close events from /proc/PID/fd analysis
 * BUSINESS:  Open files monitoring tracks file access patterns
 * CHANGE:    Open files monitoring from prototype for file tracking
 * RISK:      Medium - File descriptor monitoring affecting performance
 */
func (fio *FileIOMonitor) monitorOpenFiles() {
	ticker := time.NewTicker(3 * time.Second) // Check open files every 3 seconds
	defer ticker.Stop()
	
	knownFiles := make(map[int]map[string]bool) // PID -> file paths
	
	for {
		select {
		case <-fio.ctx.Done():
			return
		case <-ticker.C:
			fio.scanOpenFiles(knownFiles)
		}
	}
}

/**
 * CONTEXT:   Update strace monitoring for current processes
 * INPUT:     Active strace commands and current process list
 * OUTPUT:    Updated strace monitoring with new process coverage
 * BUSINESS:  Strace monitoring update ensures comprehensive coverage
 * CHANGE:    Dynamic strace management from prototype
 * RISK:      Medium - Strace process management affecting resources
 */
func (fio *FileIOMonitor) updateStraceMonitoring(activeStrace map[int]*exec.Cmd) {
	fio.mu.RLock()
	currentProcesses := make(map[int]*ProcessInfo)
	for pid, info := range fio.trackedProcesses {
		currentProcesses[pid] = info
	}
	fio.mu.RUnlock()
	
	// Stop strace for processes no longer tracked
	for pid, cmd := range activeStrace {
		if _, exists := currentProcesses[pid]; !exists {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			delete(activeStrace, pid)
			log.Printf("Stopped strace monitoring for PID %d", pid)
		}
	}
	
	// Start strace for new processes
	for pid, processInfo := range currentProcesses {
		if _, exists := activeStrace[pid]; !exists {
			if cmd := fio.startAdvancedStrace(pid, processInfo); cmd != nil {
				activeStrace[pid] = cmd
				log.Printf("Started advanced strace for %s (PID: %d)", processInfo.Command, pid)
			}
		}
	}
}

/**
 * CONTEXT:   Start advanced strace monitoring for specific process
 * INPUT:     Process ID and process information
 * OUTPUT:    Running strace command with comprehensive file tracking
 * BUSINESS:  Advanced strace provides detailed file operation analysis
 * CHANGE:    Enhanced strace monitoring from prototype with comprehensive tracking
 * RISK:      High - Advanced strace affecting process and system performance
 */
func (fio *FileIOMonitor) startAdvancedStrace(pid int, processInfo *ProcessInfo) *exec.Cmd {
	// Advanced strace with comprehensive file operations
	cmd := exec.Command("strace", "-p", strconv.Itoa(pid),
		"-e", "trace=open,openat,read,write,close,unlink,rename,mkdir,rmdir,creat,truncate",
		"-f", "-q", "-s", "100", "-xx")
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create strace stdout pipe for PID %d: %v", pid, err)
		return nil
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdout.Close()
		log.Printf("Failed to create strace stderr pipe for PID %d: %v", pid, err)
		return nil
	}
	
	if err := cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		log.Printf("Failed to start advanced strace for PID %d: %v", pid, err)
		return nil
	}
	
	// Process strace output (stderr typically contains the trace)
	go func() {
		defer stderr.Close()
		scanner := bufio.NewScanner(stderr)
		
		for scanner.Scan() {
			line := scanner.Text()
			if event := fio.parseAdvancedStraceOutput(line, pid, processInfo); event != nil {
				fio.processFileEvent(*event)
			}
		}
	}()
	
	return cmd
}

/**
 * CONTEXT:   Parse advanced strace output with comprehensive pattern matching
 * INPUT:     Advanced strace output line and process information
 * OUTPUT:    Detailed file I/O event or nil if not relevant
 * BUSINESS:  Advanced parsing extracts comprehensive file operations
 * CHANGE:    Enhanced strace parsing from prototype with multiple patterns
 * RISK:      Medium - Complex parsing affecting event accuracy
 */
func (fio *FileIOMonitor) parseAdvancedStraceOutput(line string, pid int, processInfo *ProcessInfo) *FileIOEvent {
	// Enhanced patterns for comprehensive file operation detection
	
	// Enhanced openat pattern with flags and modes
	openatPattern := regexp.MustCompile(`openat\(([^,]+),\s*"([^"]+)"(?:,\s*([^,)]+))?(?:,\s*([^)]+))?\)\s*=\s*(-?\d+)`)
	if matches := openatPattern.FindStringSubmatch(line); len(matches) >= 6 {
		filePath := matches[2]
		result := matches[5]
		
		if fio.isWorkFile(filePath) && result != "-1" { // Successful open
			eventType := FileIOOpen
			if strings.Contains(matches[3], "O_CREAT") {
				eventType = FileIOCreate
			}
			
			return &FileIOEvent{
				Type:        eventType,
				Timestamp:   time.Now(),
				PID:         pid,
				ProcessName: processInfo.Command,
				FilePath:    filePath,
				ProjectPath: fio.extractProjectPath(filePath, processInfo.WorkingDir),
				ProjectName: fio.extractProjectName(filePath, processInfo.WorkingDir),
				FileType:    filepath.Ext(filePath),
				IsWorkFile:  true,
				Details: map[string]interface{}{
					"operation": "openat",
					"flags":     matches[3],
					"mode":      matches[4],
					"fd":        result,
				},
			}
		}
	}
	
	// Enhanced write pattern with buffer content analysis
	writePattern := regexp.MustCompile(`write\((\d+),\s*"([^"]*)".*,\s*(\d+)\)\s*=\s*(\d+)`)
	if matches := writePattern.FindStringSubmatch(line); len(matches) >= 5 {
		size, _ := strconv.ParseInt(matches[4], 10, 64)
		requestedSize, _ := strconv.ParseInt(matches[3], 10, 64)
		
		if size > 0 {
			return &FileIOEvent{
				Type:        FileIOWrite,
				Timestamp:   time.Now(),
				PID:         pid,
				ProcessName: processInfo.Command,
				Size:        size,
				IsWorkFile:  true,
				Details: map[string]interface{}{
					"operation":      "write",
					"fd":             matches[1],
					"size":           size,
					"requested_size": requestedSize,
					"content_preview": matches[2][:min(len(matches[2]), 50)], // First 50 chars
				},
			}
		}
	}
	
	// Enhanced read pattern with size analysis
	readPattern := regexp.MustCompile(`read\((\d+),\s*.*,\s*(\d+)\)\s*=\s*(\d+)`)
	if matches := readPattern.FindStringSubmatch(line); len(matches) >= 4 {
		size, _ := strconv.ParseInt(matches[3], 10, 64)
		requestedSize, _ := strconv.ParseInt(matches[2], 10, 64)
		
		if size > 0 {
			return &FileIOEvent{
				Type:        FileIORead,
				Timestamp:   time.Now(),
				PID:         pid,
				ProcessName: processInfo.Command,
				Size:        size,
				IsWorkFile:  true,
				Details: map[string]interface{}{
					"operation":      "read",
					"fd":             matches[1],
					"size":           size,
					"requested_size": requestedSize,
				},
			}
		}
	}
	
	// Unlink/delete pattern
	unlinkPattern := regexp.MustCompile(`unlink(?:at)?\([^"]*"([^"]+)"\)\s*=\s*(\d+)`)
	if matches := unlinkPattern.FindStringSubmatch(line); len(matches) >= 3 {
		filePath := matches[1]
		if fio.isWorkFile(filePath) && matches[2] == "0" { // Successful delete
			return &FileIOEvent{
				Type:        FileIODelete,
				Timestamp:   time.Now(),
				PID:         pid,
				ProcessName: processInfo.Command,
				FilePath:    filePath,
				ProjectPath: fio.extractProjectPath(filePath, processInfo.WorkingDir),
				ProjectName: fio.extractProjectName(filePath, processInfo.WorkingDir),
				FileType:    filepath.Ext(filePath),
				IsWorkFile:  true,
				Details: map[string]interface{}{
					"operation": "unlink",
				},
			}
		}
	}
	
	// Rename pattern
	renamePattern := regexp.MustCompile(`rename(?:at)?\([^"]*"([^"]+)"[^"]*"([^"]+)"\)\s*=\s*(\d+)`)
	if matches := renamePattern.FindStringSubmatch(line); len(matches) >= 4 {
		oldPath := matches[1]
		newPath := matches[2]
		
		if (fio.isWorkFile(oldPath) || fio.isWorkFile(newPath)) && matches[3] == "0" {
			return &FileIOEvent{
				Type:        FileIOModify,
				Timestamp:   time.Now(),
				PID:         pid,
				ProcessName: processInfo.Command,
				FilePath:    newPath,
				ProjectPath: fio.extractProjectPath(newPath, processInfo.WorkingDir),
				ProjectName: fio.extractProjectName(newPath, processInfo.WorkingDir),
				FileType:    filepath.Ext(newPath),
				IsWorkFile:  true,
				Details: map[string]interface{}{
					"operation": "rename",
					"old_path":  oldPath,
					"new_path":  newPath,
				},
			}
		}
	}
	
	return nil
}

/**
 * CONTEXT:   Scan open files for all tracked processes
 * INPUT:     Known files map and current process tracking
 * OUTPUT:    New file open events from /proc/PID/fd analysis
 * BUSINESS:  Open files scanning detects file access patterns
 * CHANGE:    Open files scanning from prototype for file tracking
 * RISK:      Medium - File descriptor scanning affecting performance
 */
func (fio *FileIOMonitor) scanOpenFiles(knownFiles map[int]map[string]bool) {
	fio.mu.RLock()
	processes := make(map[int]*ProcessInfo)
	for pid, info := range fio.trackedProcesses {
		processes[pid] = info
	}
	fio.mu.RUnlock()
	
	for pid, processInfo := range processes {
		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		
		// Initialize known files for this PID if not exists
		if knownFiles[pid] == nil {
			knownFiles[pid] = make(map[string]bool)
		}
		
		files, err := os.ReadDir(fdDir)
		if err != nil {
			continue // Process might have ended
		}
		
		currentFiles := make(map[string]bool)
		
		for _, file := range files {
			fdPath := filepath.Join(fdDir, file.Name())
			target, err := os.Readlink(fdPath)
			if err != nil {
				continue
			}
			
			// Skip pipes, sockets, and other non-file descriptors
			if strings.Contains(target, "pipe:") || 
			   strings.Contains(target, "socket:") || 
			   strings.Contains(target, "anon_inode:") {
				continue
			}
			
			currentFiles[target] = true
			
			// New file opened
			if !knownFiles[pid][target] && fio.isWorkFile(target) {
				event := &FileIOEvent{
					Type:        FileIOOpen,
					Timestamp:   time.Now(),
					PID:         pid,
					ProcessName: processInfo.Command,
					FilePath:    target,
					ProjectPath: fio.extractProjectPath(target, processInfo.WorkingDir),
					ProjectName: fio.extractProjectName(target, processInfo.WorkingDir),
					FileType:    filepath.Ext(target),
					IsWorkFile:  true,
					Details: map[string]interface{}{
						"operation": "fd_open",
						"fd":        file.Name(),
						"method":    "/proc/fd",
					},
				}
				
				fio.processFileEvent(*event)
			}
		}
		
		// Update known files for this PID
		knownFiles[pid] = currentFiles
	}
	
	// Clean up known files for processes that no longer exist
	for pid := range knownFiles {
		if _, exists := processes[pid]; !exists {
			delete(knownFiles, pid)
		}
	}
}

// Helper methods for dynamic process attachment

/**
 * CONTEXT:   Get process information for monitoring
 * INPUT:     Process ID
 * OUTPUT:    Process information structure
 * BUSINESS:  Process info enables targeted monitoring
 * CHANGE:    Process information extraction for dynamic monitoring
 * RISK:      Low - Process information utility
 */
func (fio *FileIOMonitor) getProcessInfo(pid int) (*ProcessInfo, error) {
	// Read process command
	cmdlineFile := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmdlineBytes, err := os.ReadFile(cmdlineFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cmdline for PID %d: %w", pid, err)
	}
	
	// Parse command line (null-separated)
	cmdline := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
	cmdline = strings.TrimSpace(cmdline)
	
	parts := strings.Fields(cmdline)
	var command string
	var args []string
	
	if len(parts) > 0 {
		command = filepath.Base(parts[0])
		args = parts[1:]
	}
	
	// Read working directory
	cwdFile := fmt.Sprintf("/proc/%d/cwd", pid)
	workingDir, err := os.Readlink(cwdFile)
	if err != nil {
		workingDir = "" // Non-fatal
	}
	
	return &ProcessInfo{
		PID:        pid,
		Command:    command,
		Args:       args,
		WorkingDir: workingDir,
	}, nil
}

/**
 * CONTEXT:   Start strace monitoring for specific process
 * INPUT:     Process ID and process information
 * OUTPUT:    Running strace command monitoring file operations
 * BUSINESS:  Per-process strace enables precise file activity detection
 * CHANGE:    Dynamic strace attachment for event-driven monitoring
 * RISK:      High - Strace process affecting system performance
 */
func (fio *FileIOMonitor) startStraceForProcess(pid int, processInfo *ProcessInfo) {
	// DISABLED: strace causes Claude processes to hang/freeze
	// This is a known issue when attaching debuggers to running processes
	log.Printf("⚠️  Strace monitoring DISABLED for PID %d (%s) - prevents Claude interference", 
		pid, processInfo.Command)
	
	// Alternative: Use non-invasive /proc filesystem monitoring
	// This provides file activity without process interference
	go fio.monitorProcessFilesProcFS(pid, processInfo)
}

/**
 * CONTEXT:   Non-invasive /proc filesystem monitoring for file activity
 * INPUT:     Process ID and process information
 * OUTPUT:    File activity events without interfering with process execution
 * BUSINESS:  /proc monitoring provides file activity detection without process interference
 * CHANGE:    Non-invasive monitoring alternative to prevent Claude process hanging
 * RISK:      Low - Read-only /proc filesystem access with no process interference
 */
func (fio *FileIOMonitor) monitorProcessFilesProcFS(pid int, processInfo *ProcessInfo) {
	ticker := time.NewTicker(2 * time.Second) // Poll every 2 seconds
	defer ticker.Stop()
	
	lastOpenFiles := make(map[string]bool)
	
	for {
		select {
		case <-fio.ctx.Done():
			return
		case <-ticker.C:
			// Check if process still exists
			if !fio.processExists(pid) {
				log.Printf("📄 Process %d no longer exists, stopping /proc monitoring", pid)
				return
			}
			
			// Read open files from /proc/PID/fd
			currentFiles := fio.getProcessOpenFiles(pid)
			
			// Detect new files opened
			for file := range currentFiles {
				if !lastOpenFiles[file] && fio.isWorkRelatedFile(file) {
					// Generate file open event
					event := FileIOEvent{
						Type:        FileIOOpen,
						Timestamp:   time.Now(),
						PID:         pid,
						ProcessName: processInfo.Command,
						FilePath:    file,
						ProjectPath: processInfo.WorkingDir,
						ProjectName: fio.extractProjectName(file, processInfo.WorkingDir),
						Details: map[string]interface{}{
							"monitoring_method": "proc_fs",
							"non_invasive":      true,
						},
					}
					
					fio.processFileEvent(event)
				}
			}
			
			lastOpenFiles = currentFiles
		}
	}
}

func (fio *FileIOMonitor) processExists(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

func (fio *FileIOMonitor) getProcessOpenFiles(pid int) map[string]bool {
	files := make(map[string]bool)
	
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return files
	}
	
	for _, entry := range entries {
		fdPath := filepath.Join(fdDir, entry.Name())
		target, err := os.Readlink(fdPath)
		if err != nil {
			continue
		}
		
		// Only track regular files (skip sockets, pipes, etc.)
		if strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "/dev/") && 
		   !strings.HasPrefix(target, "/proc/") && !strings.HasPrefix(target, "/sys/") {
			files[target] = true
		}
	}
	
	return files
}

func (fio *FileIOMonitor) isWorkRelatedFile(filePath string) bool {
	// Simple work file detection - can be enhanced later
	workExtensions := []string{".go", ".js", ".py", ".ts", ".jsx", ".tsx", ".vue", ".java", ".cpp", ".c", ".h", 
		".rs", ".rb", ".php", ".cs", ".swift", ".kt", ".scala", ".clj", ".hs", ".elm", ".dart", ".r", ".m", 
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd", ".md", ".txt", ".json", ".yaml", ".yml", 
		".toml", ".ini", ".conf", ".cfg", ".xml", ".html", ".css", ".scss", ".sass", ".less", ".sql"}
	
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, workExt := range workExtensions {
		if ext == workExt {
			return true
		}
	}
	
	// Check for configuration files and common work files
	fileName := strings.ToLower(filepath.Base(filePath))
	workFiles := []string{"makefile", "dockerfile", "docker-compose", "package.json", "requirements.txt", 
		"cargo.toml", "go.mod", "pom.xml", "build.gradle", "composer.json", "gemfile", ".env", ".gitignore"}
	
	for _, workFile := range workFiles {
		if strings.Contains(fileName, workFile) {
			return true
		}
	}
	
	return false
}

/**
 * CONTEXT:   Stop strace monitoring for specific process
 * INPUT:     Process ID
 * OUTPUT:    Stopped strace process and cleanup
 * BUSINESS:  Process cleanup prevents resource leaks
 * CHANGE:    Dynamic strace cleanup for event-driven monitoring
 * RISK:      Low - Process cleanup utility
 */
func (fio *FileIOMonitor) stopStraceForProcess(pid int) {
	// Kill any strace processes monitoring this PID
	// This is a simple approach - in production, you'd want to track
	// the strace process PIDs and kill them specifically
	exec.Command("pkill", "-f", fmt.Sprintf("strace.*-p.*%d", pid)).Run()
}


// Utility function for minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}