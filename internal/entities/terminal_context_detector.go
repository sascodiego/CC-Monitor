/**
 * CONTEXT:   Terminal and process context detection for accurate session correlation
 * INPUT:     Process environment, terminal information, and system context
 * OUTPUT:    Complete terminal context for multi-factor correlation matching
 * BUSINESS:  Enable precise correlation by detecting terminal sessions, PIDs, and environment
 * CHANGE:    Initial implementation of comprehensive terminal context detection
 * RISK:      Medium - Terminal detection reliability varies across different systems and terminals
 */

package entities

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TerminalType represents different terminal applications
type TerminalType string

const (
	TerminalTypeXterm     TerminalType = "xterm"
	TerminalTypeGnome     TerminalType = "gnome-terminal" 
	TerminalTypeKonsole   TerminalType = "konsole"
	TerminalTypeTerminal  TerminalType = "terminal"     // macOS Terminal
	TerminalTypeITerm     TerminalType = "iterm"       // macOS iTerm
	TerminalTypeWSL       TerminalType = "wsl"         // Windows Subsystem for Linux
	TerminalTypeCmd       TerminalType = "cmd"         // Windows Command Prompt
	TerminalTypePowerShell TerminalType = "powershell" // Windows PowerShell
	TerminalTypeUnknown   TerminalType = "unknown"
)

/**
 * CONTEXT:   Terminal context detector with cross-platform support
 * INPUT:     System environment and process information for terminal detection
 * OUTPUT:    Terminal context detector with platform-specific detection methods
 * BUSINESS:  Provide reliable terminal context detection across different systems
 * CHANGE:    Initial detector structure with platform detection capabilities
 * RISK:      Medium - Platform detection methods have varying reliability
 */
type TerminalContextDetector struct {
	hostName      string
	platform      string
	detectionTime time.Time
}

/**
 * CONTEXT:   Factory for creating terminal context detector with platform detection
 * INPUT:     No parameters, detects platform and system automatically
 * OUTPUT:    Configured TerminalContextDetector ready for context detection
 * BUSINESS:  Initialize detector with appropriate platform-specific methods
 * CHANGE:    Initial factory with platform detection and configuration
 * RISK:      Low - Factory initialization with platform detection
 */
func NewTerminalContextDetector() *TerminalContextDetector {
	hostname, _ := os.Hostname()
	
	// Detect platform
	var platform string
	if _, err := os.Stat("/proc/version"); err == nil {
		// Linux or WSL
		if content, err := os.ReadFile("/proc/version"); err == nil {
			if strings.Contains(strings.ToLower(string(content)), "microsoft") {
				platform = "wsl"
			} else {
				platform = "linux"
			}
		} else {
			platform = "linux"
		}
	} else if _, err := os.Stat("/System/Library"); err == nil {
		platform = "darwin" // macOS
	} else {
		platform = "windows"
	}
	
	return &TerminalContextDetector{
		hostName:      hostname,
		platform:      platform,
		detectionTime: time.Now(),
	}
}

/**
 * CONTEXT:   Detect complete terminal context with process hierarchy and environment
 * INPUT:     No parameters, detects current process context automatically
 * OUTPUT:    Complete TerminalContext with PIDs, session info, and environment
 * BUSINESS:  Provide all necessary context data for accurate session correlation
 * CHANGE:    Initial comprehensive context detection implementation
 * RISK:      Medium - Detection methods may fail on some systems, fallbacks needed
 */
func (tcd *TerminalContextDetector) DetectContext() (*TerminalContext, error) {
	context := &TerminalContext{
		DetectedAt:   tcd.detectionTime,
		HostName:     tcd.hostName,
		Environment:  make(map[string]string),
		TerminalType: string(TerminalTypeUnknown),
	}
	
	// Detect PIDs
	if err := tcd.detectProcessInfo(context); err != nil {
		// Continue with detection even if PID detection fails
	}
	
	// Detect working directory
	if workingDir, err := os.Getwd(); err == nil {
		context.WorkingDir = workingDir
	}
	
	// Detect terminal session information
	if err := tcd.detectSessionInfo(context); err != nil {
		// Continue with detection even if session detection fails
	}
	
	// Detect terminal type
	context.TerminalType = string(tcd.detectTerminalType())
	
	// Collect relevant environment variables
	tcd.collectEnvironmentVars(context)
	
	// Platform-specific enhancements
	if err := tcd.enhanceWithPlatformSpecific(context); err != nil {
		// Continue even if platform-specific detection fails
	}
	
	return context, nil
}

/**
 * CONTEXT:   Detect process IDs including parent and shell processes
 * INPUT:     Terminal context to populate with process information
 * OUTPUT:    Updated context with PID and shell PID information
 * BUSINESS:  Process IDs provide the most reliable correlation factor
 * CHANGE:    Initial process information detection with hierarchy traversal
 * RISK:      Medium - Process detection methods vary by platform and may fail
 */
func (tcd *TerminalContextDetector) detectProcessInfo(context *TerminalContext) error {
	// Get current process PID
	currentPID := os.Getpid()
	context.ShellPID = currentPID
	
	// Get parent PID
	parentPID := os.Getppid()
	context.PID = parentPID
	
	// Try to find terminal process by walking up the process tree
	terminalPID, err := tcd.findTerminalProcess(currentPID)
	if err == nil {
		context.PID = terminalPID
	}
	
	return nil
}

/**
 * CONTEXT:   Find terminal process by walking up the process hierarchy
 * INPUT:     Starting PID to begin process tree traversal
 * OUTPUT:    Terminal process PID or error if not found
 * BUSINESS:  Identify actual terminal process for precise correlation
 * CHANGE:    Initial process tree traversal with terminal detection
 * RISK:      Medium - Process tree traversal complexity varies by system
 */
func (tcd *TerminalContextDetector) findTerminalProcess(startPID int) (int, error) {
	switch tcd.platform {
	case "linux", "wsl":
		return tcd.findTerminalProcessLinux(startPID)
	case "darwin":
		return tcd.findTerminalProcessMacOS(startPID)
	case "windows":
		return tcd.findTerminalProcessWindows(startPID)
	default:
		return startPID, fmt.Errorf("unsupported platform for terminal detection: %s", tcd.platform)
	}
}

/**
 * CONTEXT:   Linux-specific terminal process detection using /proc filesystem
 * INPUT:     Starting PID for process tree traversal
 * OUTPUT:    Terminal process PID or error if detection fails
 * BUSINESS:  Linux process detection using /proc for reliable PID information
 * CHANGE:    Initial Linux terminal detection using /proc/PID/stat
 * RISK:      Low - Linux /proc filesystem provides reliable process information
 */
func (tcd *TerminalContextDetector) findTerminalProcessLinux(startPID int) (int, error) {
	currentPID := startPID
	maxDepth := 10 // Prevent infinite loops
	
	for depth := 0; depth < maxDepth; depth++ {
		// Read process information from /proc
		statPath := fmt.Sprintf("/proc/%d/stat", currentPID)
		statData, err := os.ReadFile(statPath)
		if err != nil {
			break
		}
		
		cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", currentPID)
		cmdlineData, err := os.ReadFile(cmdlinePath)
		if err != nil {
			break
		}
		
		cmdline := strings.ReplaceAll(string(cmdlineData), "\x00", " ")
		cmdline = strings.TrimSpace(cmdline)
		
		// Check if this looks like a terminal process
		if tcd.isTerminalProcess(cmdline) {
			return currentPID, nil
		}
		
		// Get parent PID from stat
		statFields := strings.Fields(string(statData))
		if len(statFields) < 4 {
			break
		}
		
		parentPID, err := strconv.Atoi(statFields[3])
		if err != nil || parentPID <= 1 {
			break
		}
		
		if parentPID == currentPID {
			break // Prevent infinite loop
		}
		
		currentPID = parentPID
	}
	
	return startPID, fmt.Errorf("terminal process not found")
}

/**
 * CONTEXT:   macOS-specific terminal process detection using ps command
 * INPUT:     Starting PID for process tree traversal
 * OUTPUT:    Terminal process PID or error if detection fails
 * BUSINESS:  macOS process detection using system commands
 * CHANGE:    Initial macOS terminal detection using ps
 * RISK:      Medium - Command execution may fail or vary across macOS versions
 */
func (tcd *TerminalContextDetector) findTerminalProcessMacOS(startPID int) (int, error) {
	cmd := exec.Command("ps", "-o", "pid,ppid,comm", "-p", strconv.Itoa(startPID))
	output, err := cmd.Output()
	if err != nil {
		return startPID, err
	}
	
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return startPID, fmt.Errorf("no process information found")
	}
	
	// Parse ps output to find terminal
	currentPID := startPID
	maxDepth := 10
	
	for depth := 0; depth < maxDepth; depth++ {
		cmd := exec.Command("ps", "-o", "pid,ppid,comm", "-p", strconv.Itoa(currentPID))
		output, err := cmd.Output()
		if err != nil {
			break
		}
		
		lines := strings.Split(string(output), "\n")
		if len(lines) < 2 {
			break
		}
		
		fields := strings.Fields(lines[1])
		if len(fields) < 3 {
			break
		}
		
		command := fields[2]
		if strings.Contains(command, "Terminal") || strings.Contains(command, "iTerm") {
			return currentPID, nil
		}
		
		parentPID, err := strconv.Atoi(fields[1])
		if err != nil || parentPID <= 1 {
			break
		}
		
		if parentPID == currentPID {
			break
		}
		
		currentPID = parentPID
	}
	
	return startPID, fmt.Errorf("terminal process not found")
}

/**
 * CONTEXT:   Windows-specific terminal process detection
 * INPUT:     Starting PID for process tree traversal
 * OUTPUT:    Terminal process PID or error if detection fails
 * BUSINESS:  Windows process detection for command prompt and PowerShell
 * CHANGE:    Initial Windows terminal detection
 * RISK:      High - Windows process detection is more complex and variable
 */
func (tcd *TerminalContextDetector) findTerminalProcessWindows(startPID int) (int, error) {
	// Windows terminal detection is more complex
	// For now, return the parent PID
	return os.Getppid(), nil
}

/**
 * CONTEXT:   Check if process command line indicates a terminal application
 * INPUT:     Process command line string from process information
 * OUTPUT:    Boolean indicating if process appears to be a terminal
 * BUSINESS:  Identify terminal processes for accurate correlation
 * CHANGE:    Initial terminal process identification heuristics
 * RISK:      Low - Heuristic matching with known terminal patterns
 */
func (tcd *TerminalContextDetector) isTerminalProcess(cmdline string) bool {
	cmdlineLower := strings.ToLower(cmdline)
	
	terminalIndicators := []string{
		"gnome-terminal",
		"konsole",
		"xterm",
		"terminal",
		"iterm",
		"cmd.exe",
		"powershell",
		"wsl.exe",
		"bash.exe",
	}
	
	for _, indicator := range terminalIndicators {
		if strings.Contains(cmdlineLower, indicator) {
			return true
		}
	}
	
	return false
}

/**
 * CONTEXT:   Detect terminal session information and identifiers
 * INPUT:     Terminal context to populate with session information
 * OUTPUT:    Updated context with session ID and terminal-specific data
 * BUSINESS:  Session identifiers help correlate multiple Claude sessions in same terminal
 * CHANGE:    Initial session information detection with multiple methods
 * RISK:      Medium - Session detection methods vary significantly by terminal type
 */
func (tcd *TerminalContextDetector) detectSessionInfo(context *TerminalContext) error {
	// Method 1: Environment-based session detection
	sessionEnvVars := []string{
		"TERM_SESSION_ID",
		"WINDOWID",
		"ITERM_SESSION_ID",
		"TERMINAL_SESSION_ID",
		"TMUX",
		"STY", // GNU Screen
	}
	
	for _, envVar := range sessionEnvVars {
		if value := os.Getenv(envVar); value != "" {
			context.SessionID = fmt.Sprintf("%s=%s", envVar, value)
			break
		}
	}
	
	// Method 2: Terminal-specific detection
	if context.SessionID == "" {
		context.SessionID = tcd.detectTerminalSpecificSessionID()
	}
	
	// Method 3: Generate fallback session ID
	if context.SessionID == "" {
		context.SessionID = fmt.Sprintf("term_%d_%d_%s", 
			context.PID, 
			context.ShellPID, 
			tcd.hostName)
	}
	
	return nil
}

/**
 * CONTEXT:   Detect terminal type from environment and process information
 * INPUT:     No parameters, uses environment variables and process data
 * OUTPUT:    TerminalType indicating the specific terminal application
 * BUSINESS:  Terminal type information helps with session correlation accuracy
 * CHANGE:    Initial terminal type detection with environment analysis
 * RISK:      Low - Terminal type detection is supplementary information
 */
func (tcd *TerminalContextDetector) detectTerminalType() TerminalType {
	// Check environment variables first
	if term := os.Getenv("TERM_PROGRAM"); term != "" {
		switch strings.ToLower(term) {
		case "iterm.app":
			return TerminalTypeITerm
		case "apple_terminal":
			return TerminalTypeTerminal
		case "gnome-terminal":
			return TerminalTypeGnome
		}
	}
	
	// Check TERM variable
	if term := os.Getenv("TERM"); term != "" {
		if strings.Contains(term, "xterm") {
			return TerminalTypeXterm
		}
	}
	
	// Platform-specific defaults
	switch tcd.platform {
	case "wsl":
		return TerminalTypeWSL
	case "windows":
		if os.Getenv("PSModulePath") != "" {
			return TerminalTypePowerShell
		}
		return TerminalTypeCmd
	case "darwin":
		return TerminalTypeTerminal
	case "linux":
		return TerminalTypeXterm
	}
	
	return TerminalTypeUnknown
}

/**
 * CONTEXT:   Terminal-specific session ID detection for different terminal types
 * INPUT:     No parameters, uses detected terminal type and environment
 * OUTPUT:    Terminal-specific session identifier string
 * BUSINESS:  Specialized session detection for better correlation accuracy
 * CHANGE:    Initial terminal-specific session ID detection
 * RISK:      Medium - Terminal-specific methods may not work on all versions
 */
func (tcd *TerminalContextDetector) detectTerminalSpecificSessionID() string {
	terminalType := tcd.detectTerminalType()
	
	switch terminalType {
	case TerminalTypeITerm:
		// iTerm provides session ID in environment
		if sessionID := os.Getenv("ITERM_SESSION_ID"); sessionID != "" {
			return fmt.Sprintf("iterm_%s", sessionID)
		}
		
	case TerminalTypeGnome:
		// GNOME Terminal uses WINDOWID
		if windowID := os.Getenv("WINDOWID"); windowID != "" {
			return fmt.Sprintf("gnome_%s", windowID)
		}
		
	case TerminalTypeWSL:
		// WSL might have Windows-specific identifiers
		if wslDistro := os.Getenv("WSL_DISTRO_NAME"); wslDistro != "" {
			return fmt.Sprintf("wsl_%s_%d", wslDistro, os.Getpid())
		}
	}
	
	return ""
}

/**
 * CONTEXT:   Collect relevant environment variables for correlation context
 * INPUT:     Terminal context to populate with environment data
 * OUTPUT:    Updated context with relevant environment variables
 * BUSINESS:  Environment variables provide additional correlation context
 * CHANGE:    Initial environment variable collection for correlation
 * RISK:      Low - Environment variable collection for context enhancement
 */
func (tcd *TerminalContextDetector) collectEnvironmentVars(context *TerminalContext) {
	relevantVars := []string{
		"USER", "USERNAME", "LOGNAME",
		"HOME", "USERPROFILE",
		"SHELL",
		"TERM", "TERM_PROGRAM",
		"PWD", "OLDPWD",
		"SSH_CLIENT", "SSH_CONNECTION", // For remote sessions
		"DISPLAY", "WAYLAND_DISPLAY",   // For X11/Wayland
		"WINDOWID",
		"ITERM_SESSION_ID",
		"TERMINAL_SESSION_ID",
		"TMUX", "TMUX_PANE",
		"STY", // GNU Screen
		"WSL_DISTRO_NAME",
		"WT_SESSION", // Windows Terminal
	}
	
	for _, varName := range relevantVars {
		if value := os.Getenv(varName); value != "" {
			context.Environment[varName] = value
		}
	}
}

/**
 * CONTEXT:   Platform-specific context enhancements with additional detection methods
 * INPUT:     Terminal context to enhance with platform-specific information
 * OUTPUT:    Enhanced context with platform-specific data and identifiers
 * BUSINESS:  Platform-specific enhancements improve correlation accuracy
 * CHANGE:    Initial platform-specific enhancement implementation
 * RISK:      Medium - Platform-specific methods may fail on some configurations
 */
func (tcd *TerminalContextDetector) enhanceWithPlatformSpecific(context *TerminalContext) error {
	switch tcd.platform {
	case "linux":
		return tcd.enhanceLinuxContext(context)
	case "darwin":
		return tcd.enhanceMacOSContext(context)
	case "windows", "wsl":
		return tcd.enhanceWindowsContext(context)
	default:
		return nil
	}
}

/**
 * CONTEXT:   Linux-specific context enhancement using system information
 * INPUT:     Terminal context to enhance with Linux-specific data
 * OUTPUT:    Enhanced context with Linux system information
 * BUSINESS:  Linux enhancements for better correlation on Linux systems
 * CHANGE:    Initial Linux-specific context enhancement
 * RISK:      Low - Linux system information is generally reliable
 */
func (tcd *TerminalContextDetector) enhanceLinuxContext(context *TerminalContext) error {
	// Add Linux-specific information
	
	// Check if running under X11
	if display := os.Getenv("DISPLAY"); display != "" {
		context.Environment["X11_DISPLAY"] = display
		
		// Try to get window ID for X11
		if windowID := os.Getenv("WINDOWID"); windowID != "" {
			context.WindowID = windowID
		}
	}
	
	// Check for Wayland
	if waylandDisplay := os.Getenv("WAYLAND_DISPLAY"); waylandDisplay != "" {
		context.Environment["WAYLAND_DISPLAY"] = waylandDisplay
	}
	
	// Read additional system information
	if loginInfo, err := tcd.getLinuxLoginInfo(); err == nil {
		context.Environment["LOGIN_INFO"] = loginInfo
	}
	
	return nil
}

/**
 * CONTEXT:   macOS-specific context enhancement using system frameworks
 * INPUT:     Terminal context to enhance with macOS-specific data
 * OUTPUT:    Enhanced context with macOS system information
 * BUSINESS:  macOS enhancements for better correlation on macOS systems
 * CHANGE:    Initial macOS-specific context enhancement
 * RISK:      Medium - macOS system information access may vary by version
 */
func (tcd *TerminalContextDetector) enhanceMacOSContext(context *TerminalContext) error {
	// Add macOS-specific information
	
	// Try to get window information (requires additional tools)
	// For now, use available environment information
	
	if termProgram := os.Getenv("TERM_PROGRAM"); termProgram != "" {
		context.Environment["TERM_PROGRAM"] = termProgram
		
		if termVersion := os.Getenv("TERM_PROGRAM_VERSION"); termVersion != "" {
			context.Environment["TERM_PROGRAM_VERSION"] = termVersion
		}
	}
	
	return nil
}

/**
 * CONTEXT:   Windows/WSL-specific context enhancement
 * INPUT:     Terminal context to enhance with Windows-specific data
 * OUTPUT:    Enhanced context with Windows system information
 * BUSINESS:  Windows enhancements for better correlation on Windows systems
 * CHANGE:    Initial Windows-specific context enhancement
 * RISK:      High - Windows terminal detection is complex and variable
 */
func (tcd *TerminalContextDetector) enhanceWindowsContext(context *TerminalContext) error {
	// Add Windows-specific information
	
	if tcd.platform == "wsl" {
		// WSL-specific enhancements
		if wslDistro := os.Getenv("WSL_DISTRO_NAME"); wslDistro != "" {
			context.Environment["WSL_DISTRO"] = wslDistro
		}
		
		if wslInterop := os.Getenv("WSL_INTEROP"); wslInterop != "" {
			context.Environment["WSL_INTEROP"] = wslInterop
		}
	}
	
	// Windows Terminal detection
	if wtSession := os.Getenv("WT_SESSION"); wtSession != "" {
		context.SessionID = fmt.Sprintf("wt_%s", wtSession)
		context.Environment["WT_SESSION"] = wtSession
	}
	
	return nil
}

/**
 * CONTEXT:   Get Linux login session information for enhanced context
 * INPUT:     No parameters, reads system login information
 * OUTPUT:    Login session information string or error
 * BUSINESS:  Linux login information provides additional correlation context
 * CHANGE:    Initial Linux login information detection
 * RISK:      Low - Login information access is generally available
 */
func (tcd *TerminalContextDetector) getLinuxLoginInfo() (string, error) {
	// Try to get session information from loginctl or who
	cmd := exec.Command("who", "am", "i")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output)), nil
	}
	
	// Fallback to environment-based detection
	if user := os.Getenv("USER"); user != "" {
		return fmt.Sprintf("user=%s", user), nil
	}
	
	return "", fmt.Errorf("login information not available")
}

/**
 * CONTEXT:   Validate terminal context completeness and consistency
 * INPUT:     Terminal context to validate
 * OUTPUT:    Error if context is invalid or incomplete, nil if valid
 * BUSINESS:  Ensure terminal context has sufficient data for correlation
 * CHANGE:    Initial terminal context validation
 * RISK:      Low - Validation helps ensure correlation data quality
 */
func (tcd *TerminalContextDetector) ValidateContext(context *TerminalContext) error {
	if context == nil {
		return fmt.Errorf("terminal context is nil")
	}
	
	// Check required fields
	if context.PID == 0 && context.ShellPID == 0 {
		return fmt.Errorf("no process information available")
	}
	
	if context.WorkingDir == "" {
		return fmt.Errorf("working directory not detected")
	}
	
	if context.SessionID == "" {
		return fmt.Errorf("session ID not generated")
	}
	
	if context.HostName == "" {
		return fmt.Errorf("hostname not detected")
	}
	
	// Validate working directory exists
	if _, err := os.Stat(context.WorkingDir); err != nil {
		return fmt.Errorf("working directory does not exist: %s", context.WorkingDir)
	}
	
	// Validate working directory is absolute
	if !filepath.IsAbs(context.WorkingDir) {
		return fmt.Errorf("working directory must be absolute path: %s", context.WorkingDir)
	}
	
	return nil
}