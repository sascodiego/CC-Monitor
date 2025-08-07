/**
 * CONTEXT:   Non-invasive file monitoring for Claude processes without strace interference
 * INPUT:     Process information and working directories for file activity detection
 * OUTPUT:    File activity events without interfering with process execution
 * BUSINESS:  Non-invasive monitoring provides file activity detection without Claude process interference
 * CHANGE:    Complete replacement for invasive strace-based file monitoring
 * RISK:      Low - Read-only monitoring with no process interference
 */

package monitor

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// File I/O event types are defined in file_io_monitor.go

/**
 * CONTEXT:   Non-invasive file monitor structure
 * INPUT:     File monitoring configuration and process tracking
 * OUTPUT:    File activity events from multiple non-invasive sources
 * BUSINESS:  Non-invasive file monitoring enables safe Claude process activity detection
 * CHANGE:    Non-invasive file monitor replacing strace-based monitoring
 * RISK:      Low - Safe monitoring without process interference
 */
type NonInvasiveFileMonitor struct {
	ctx              context.Context
	cancel           context.CancelFunc
	eventCallback    func(FileIOEvent)
	trackedProcesses map[int]*ProcessInfo
	workingDirs      map[string]bool // Tracked working directories
	mu               sync.RWMutex
	running          bool
	stats            FileIOStats
}

/**
 * CONTEXT:   Create new non-invasive file monitor
 * INPUT:     Event callback for file activity notifications
 * OUTPUT:    Configured non-invasive file monitor ready for safe monitoring
 * BUSINESS:  Non-invasive monitor creation enables safe file activity tracking
 * CHANGE:    Non-invasive monitor creation replacing strace-based approach
 * RISK:      Low - Safe monitor initialization
 */
func NewNonInvasiveFileMonitor(callback func(FileIOEvent)) *NonInvasiveFileMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &NonInvasiveFileMonitor{
		ctx:              ctx,
		cancel:           cancel,
		eventCallback:    callback,
		trackedProcesses: make(map[int]*ProcessInfo),
		workingDirs:      make(map[string]bool),
		stats: FileIOStats{
			EventsByType:    make(map[string]uint64),
			EventsByProject: make(map[string]uint64),
			StartTime:       time.Now(),
		},
	}
}

/**
 * CONTEXT:   Start non-invasive file monitoring
 * INPUT:     Monitor activation request
 * OUTPUT:    Active non-invasive file monitoring with multiple detection methods
 * BUSINESS:  Non-invasive monitor start enables safe file activity tracking
 * CHANGE:    Multi-method non-invasive monitoring start
 * RISK:      Low - Safe monitoring activation
 */
func (nfm *NonInvasiveFileMonitor) Start() error {
	nfm.mu.Lock()
	defer nfm.mu.Unlock()
	
	if nfm.running {
		return fmt.Errorf("non-invasive file monitor is already running")
	}
	
	nfm.running = true
	
	// Method 1: Poll open files via /proc/PID/fd
	go nfm.pollOpenFilesWorker()
	
	// Method 2: Monitor working directories with inotify-like polling
	go nfm.monitorWorkingDirectories()
	
	// Method 3: Periodic lsof check for work files
	go nfm.periodicWorkFileCheck()
	
	log.Printf("📁 Non-invasive file monitor started - NO strace, NO process interference")
	return nil
}

/**
 * CONTEXT:   Stop non-invasive file monitoring
 * INPUT:     Monitor shutdown request
 * OUTPUT:    Cleanly stopped non-invasive monitoring
 * BUSINESS:  Safe monitor shutdown with cleanup
 * CHANGE:    Non-invasive monitor stop implementation
 * RISK:      Low - Safe monitoring shutdown
 */
func (nfm *NonInvasiveFileMonitor) Stop() error {
	nfm.mu.Lock()
	defer nfm.mu.Unlock()
	
	if !nfm.running {
		return nil
	}
	
	nfm.cancel()
	nfm.running = false
	
	log.Printf("📁 Non-invasive file monitor stopped")
	return nil
}

/**
 * CONTEXT:   Attach monitoring to process (non-invasive)
 * INPUT:     Process ID to monitor
 * OUTPUT:    Process added to tracking without interference
 * BUSINESS:  Process attachment enables targeted file monitoring
 * CHANGE:    Non-invasive process attachment
 * RISK:      Low - Safe process tracking addition
 */
func (nfm *NonInvasiveFileMonitor) AttachToProcess(pid int) error {
	nfm.mu.Lock()
	defer nfm.mu.Unlock()
	
	if !nfm.running {
		return fmt.Errorf("non-invasive file monitor not running")
	}
	
	// Get process information
	processInfo, err := nfm.getProcessInfo(pid)
	if err != nil {
		return fmt.Errorf("failed to get process info for PID %d: %w", pid, err)
	}
	
	// Add to tracked processes
	nfm.trackedProcesses[pid] = processInfo
	
	// Add working directory to monitoring
	if processInfo.WorkingDir != "" {
		nfm.workingDirs[processInfo.WorkingDir] = true
	}
	
	log.Printf("📂 Non-invasive monitoring attached to PID %d (%s) in %s", 
		pid, processInfo.Command, processInfo.WorkingDir)
	return nil
}

/**
 * CONTEXT:   Detach monitoring from process
 * INPUT:     Process ID to stop monitoring
 * OUTPUT:    Process removed from tracking
 * BUSINESS:  Process detachment prevents resource leaks
 * CHANGE:    Non-invasive process detachment
 * RISK:      Low - Safe process tracking removal
 */
func (nfm *NonInvasiveFileMonitor) DetachFromProcess(pid int) error {
	nfm.mu.Lock()
	defer nfm.mu.Unlock()
	
	// Remove from tracked processes
	if processInfo, exists := nfm.trackedProcesses[pid]; exists {
		delete(nfm.trackedProcesses, pid)
		log.Printf("📂 Non-invasive monitoring detached from PID %d (%s)", 
			pid, processInfo.Command)
	}
	
	return nil
}

/**
 * CONTEXT:   Worker for polling open files via /proc/PID/fd
 * INPUT:     Continuous monitoring of tracked processes
 * OUTPUT:    File open/close events from /proc filesystem
 * BUSINESS:  /proc polling provides file activity without process interference
 * CHANGE:    /proc-based file monitoring implementation
 * RISK:      Low - Read-only /proc filesystem access
 */
func (nfm *NonInvasiveFileMonitor) pollOpenFilesWorker() {
	ticker := time.NewTicker(3 * time.Second) // Poll every 3 seconds
	defer ticker.Stop()
	
	lastOpenFiles := make(map[int]map[string]bool)
	
	for {
		select {
		case <-nfm.ctx.Done():
			return
		case <-ticker.C:
			nfm.mu.RLock()
			processes := make(map[int]*ProcessInfo)
			for pid, info := range nfm.trackedProcesses {
				processes[pid] = info
			}
			nfm.mu.RUnlock()
			
			for pid, processInfo := range processes {
				// Check if process still exists
				if !nfm.processExists(pid) {
					nfm.DetachFromProcess(pid)
					continue
				}
				
				// Get current open files
				currentFiles := nfm.getProcessOpenFiles(pid)
				
				// Compare with last check
				lastFiles := lastOpenFiles[pid]
				if lastFiles == nil {
					lastFiles = make(map[string]bool)
				}
				
				// Detect newly opened files
				for file := range currentFiles {
					if !lastFiles[file] && nfm.isWorkRelatedFile(file) {
						nfm.generateFileEvent(FileIOOpen, file, pid, processInfo)
					}
				}
				
				// Detect closed files (file no longer open)
				for file := range lastFiles {
					if !currentFiles[file] && nfm.isWorkRelatedFile(file) {
						// Use FileIORead to indicate file was accessed (closed implies it was read)
						nfm.generateFileEvent(FileIORead, file, pid, processInfo)
					}
				}
				
				lastOpenFiles[pid] = currentFiles
			}
		}
	}
}

/**
 * CONTEXT:   Monitor working directories for file changes
 * INPUT:     Working directory monitoring for file activity
 * OUTPUT:    File modification events from directory polling
 * BUSINESS:  Directory monitoring provides file change detection
 * CHANGE:    Directory-based file change monitoring
 * RISK:      Low - Directory polling without process interference
 */
func (nfm *NonInvasiveFileMonitor) monitorWorkingDirectories() {
	ticker := time.NewTicker(5 * time.Second) // Poll directories every 5 seconds
	defer ticker.Stop()
	
	lastModTimes := make(map[string]time.Time)
	
	for {
		select {
		case <-nfm.ctx.Done():
			return
		case <-ticker.C:
			nfm.mu.RLock()
			dirs := make(map[string]bool)
			for dir := range nfm.workingDirs {
				dirs[dir] = true
			}
			nfm.mu.RUnlock()
			
			for dir := range dirs {
				nfm.checkDirectoryForChanges(dir, lastModTimes)
			}
		}
	}
}

/**
 * CONTEXT:   Check directory for file changes
 * INPUT:     Directory path and last modification times
 * OUTPUT:    File modification events for changed files
 * BUSINESS:  Directory change detection enables file activity tracking
 * CHANGE:    Directory change detection implementation
 * RISK:      Low - File system read operations
 */
func (nfm *NonInvasiveFileMonitor) checkDirectoryForChanges(dir string, lastModTimes map[string]time.Time) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, don't fail the walk
		}
		
		// Only check work-related files
		if info.IsDir() || !nfm.isWorkRelatedFile(path) {
			return nil
		}
		
		// Check if file was modified
		modTime := info.ModTime()
		if lastTime, exists := lastModTimes[path]; !exists || modTime.After(lastTime) {
			lastModTimes[path] = modTime
			
			// Generate file modification event
			if exists { // Don't generate events for initial scan
				nfm.generateFileEventFromPath(FileIOModify, path, dir)
			}
		}
		
		return nil
	})
	
	if err != nil && nfm.isVerboseLogging() {
		log.Printf("📁 Error walking directory %s: %v", dir, err)
	}
}

/**
 * CONTEXT:   Periodic lsof check for work files
 * INPUT:     Periodic lsof execution for tracked processes
 * OUTPUT:    Work file access events from lsof
 * BUSINESS:  lsof provides additional file access detection
 * CHANGE:    lsof-based file access monitoring
 * RISK:      Low - External command execution without process interference
 */
func (nfm *NonInvasiveFileMonitor) periodicWorkFileCheck() {
	ticker := time.NewTicker(10 * time.Second) // lsof check every 10 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-nfm.ctx.Done():
			return
		case <-ticker.C:
			nfm.mu.RLock()
			processes := make(map[int]*ProcessInfo)
			for pid, info := range nfm.trackedProcesses {
				processes[pid] = info
			}
			nfm.mu.RUnlock()
			
			for pid, processInfo := range processes {
				if !nfm.processExists(pid) {
					continue
				}
				
				workFiles := nfm.getLsofWorkFiles(pid)
				for _, file := range workFiles {
					nfm.generateFileEvent(FileIORead, file, pid, processInfo)
				}
			}
		}
	}
}

// Helper methods

func (nfm *NonInvasiveFileMonitor) processExists(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

func (nfm *NonInvasiveFileMonitor) getProcessOpenFiles(pid int) map[string]bool {
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

func (nfm *NonInvasiveFileMonitor) isWorkRelatedFile(filePath string) bool {
	// Enhanced work file detection
	workExtensions := []string{
		".go", ".js", ".ts", ".jsx", ".tsx", ".vue", ".py", ".java", ".cpp", ".c", ".h", ".hpp",
		".rs", ".rb", ".php", ".cs", ".swift", ".kt", ".scala", ".clj", ".hs", ".elm", ".dart",
		".r", ".m", ".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd", ".md", ".txt", 
		".json", ".yaml", ".yml", ".toml", ".ini", ".conf", ".cfg", ".xml", ".html", ".css", 
		".scss", ".sass", ".less", ".sql", ".graphql", ".proto", ".thrift",
	}
	
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, workExt := range workExtensions {
		if ext == workExt {
			return true
		}
	}
	
	// Check for configuration files and common work files
	fileName := strings.ToLower(filepath.Base(filePath))
	workFiles := []string{
		"makefile", "dockerfile", "docker-compose", "package.json", "requirements.txt", 
		"cargo.toml", "go.mod", "go.sum", "pom.xml", "build.gradle", "composer.json", 
		"gemfile", ".env", ".gitignore", ".gitconfig", "readme", "license", "changelog",
		"tsconfig.json", "webpack.config", "rollup.config", "vite.config",
	}
	
	for _, workFile := range workFiles {
		if strings.Contains(fileName, workFile) {
			return true
		}
	}
	
	return false
}

func (nfm *NonInvasiveFileMonitor) getLsofWorkFiles(pid int) []string {
	// Implementation of lsof-based work file detection
	// This is a placeholder - can be implemented if needed
	return []string{}
}

func (nfm *NonInvasiveFileMonitor) generateFileEvent(eventType FileIOEventType, filePath string, pid int, processInfo *ProcessInfo) {
	event := FileIOEvent{
		Type:        eventType,
		Timestamp:   time.Now(),
		PID:         pid,
		ProcessName: processInfo.Command,
		FilePath:    filePath,
		ProjectPath: processInfo.WorkingDir,
		ProjectName: nfm.extractProjectName(filePath, processInfo.WorkingDir),
		Details: map[string]interface{}{
			"monitoring_method": "non_invasive",
			"safe_monitoring":   true,
			"source":           "proc_fs",
		},
	}
	
	nfm.processFileEvent(event)
}

func (nfm *NonInvasiveFileMonitor) generateFileEventFromPath(eventType FileIOEventType, filePath, projectPath string) {
	event := FileIOEvent{
		Type:        eventType,
		Timestamp:   time.Now(),
		FilePath:    filePath,
		ProjectPath: projectPath,
		ProjectName: nfm.extractProjectName(filePath, projectPath),
		Details: map[string]interface{}{
			"monitoring_method": "directory_polling",
			"safe_monitoring":   true,
			"source":           "directory_watch",
		},
	}
	
	nfm.processFileEvent(event)
}

func (nfm *NonInvasiveFileMonitor) processFileEvent(event FileIOEvent) {
	// Update statistics
	nfm.mu.Lock()
	nfm.stats.TotalEvents++
	nfm.stats.WorkFileEvents++
	nfm.stats.EventsByType[string(event.Type)]++
	if event.ProjectName != "" {
		nfm.stats.EventsByProject[event.ProjectName]++
	}
	nfm.stats.LastEventTime = event.Timestamp
	nfm.mu.Unlock()
	
	// Send to callback
	if nfm.eventCallback != nil {
		nfm.eventCallback(event)
	}
	
	if nfm.isVerboseLogging() {
		log.Printf("📄 File Event: %s %s in %s (PID: %d)", 
			event.Type, filepath.Base(event.FilePath), event.ProjectName, event.PID)
	}
}

func (nfm *NonInvasiveFileMonitor) extractProjectName(filePath, workingDir string) string {
	if workingDir != "" {
		return filepath.Base(workingDir)
	}
	
	// Try to extract project name from file path
	parts := strings.Split(filepath.Dir(filePath), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" && parts[i] != "." {
			return parts[i]
		}
	}
	
	return "unknown"
}

func (nfm *NonInvasiveFileMonitor) getProcessInfo(pid int) (*ProcessInfo, error) {
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

func (nfm *NonInvasiveFileMonitor) isVerboseLogging() bool {
	// Simple verbose logging check - can be enhanced with configuration
	return false
}

/**
 * CONTEXT:   Get non-invasive file monitoring statistics
 * INPUT:     Statistics request
 * OUTPUT:    Current monitoring metrics and performance data
 * BUSINESS:  Statistics provide monitoring system health and activity insights
 * CHANGE:    Non-invasive monitoring statistics getter
 * RISK:      Low - Read-only statistics access
 */
func (nfm *NonInvasiveFileMonitor) GetStats() FileIOStats {
	nfm.mu.RLock()
	defer nfm.mu.RUnlock()
	
	// Create a copy to prevent concurrent access issues
	stats := nfm.stats
	stats.EventsByType = make(map[string]uint64)
	stats.EventsByProject = make(map[string]uint64)
	
	for k, v := range nfm.stats.EventsByType {
		stats.EventsByType[k] = v
	}
	for k, v := range nfm.stats.EventsByProject {
		stats.EventsByProject[k] = v
	}
	
	return stats
}