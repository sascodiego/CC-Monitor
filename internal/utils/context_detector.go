/**
 * CONTEXT:   Context detection utilities for automatic session correlation without ID passing
 * INPUT:     Current process environment and system state
 * OUTPUT:    SessionContext with terminal PID, working directory, and project information
 * BUSINESS:  Eliminate need for manual ID management by detecting environment context
 * CHANGE:    Initial implementation of automatic context detection for hook correlation
 * RISK:      Medium - Context detection accuracy critical for correlation system reliability
 */

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

)

// ContextDetector provides utilities for detecting session context automatically
type ContextDetector struct {
	// Configuration for context detection behavior
	detectProjectName bool
	useGitRepository  bool
	fallbackUserID    string
}

// ContextDetectorConfig holds configuration for context detection
type ContextDetectorConfig struct {
	DetectProjectName bool   // Whether to detect project name from directory
	UseGitRepository  bool   // Whether to use git repository root as project path
	FallbackUserID    string // Fallback user ID if environment detection fails
}

/**
 * CONTEXT:   Factory method for creating context detector with configuration
 * INPUT:     ContextDetectorConfig with detection behavior settings
 * OUTPUT:    Configured ContextDetector ready for context detection
 * BUSINESS:  Initialize detector with appropriate settings for different environments
 * CHANGE:    Initial implementation with configurable detection behavior
 * RISK:      Low - Configuration initialization with sensible defaults
 */
func NewContextDetector(config ContextDetectorConfig) *ContextDetector {
	return &ContextDetector{
		detectProjectName: config.DetectProjectName,
		useGitRepository:  config.UseGitRepository,
		fallbackUserID:    config.FallbackUserID,
	}
}

/**
 * CONTEXT:   Detect complete session context from current process environment
 * INPUT:     No parameters, detects context from current process and environment
 * OUTPUT:    SessionContext with all necessary information for session correlation
 * BUSINESS:  Provide complete context for daemon-managed correlation without manual setup
 * CHANGE:    Initial comprehensive context detection implementation
 * RISK:      High - Context detection accuracy directly affects all correlation attempts
 */
func (cd *ContextDetector) DetectSessionContext() (*entities.SessionContext, error) {
	// Detect current timestamp
	timestamp := time.Now()

	// Detect user ID
	userID, err := cd.detectUserID()
	if err != nil {
		return nil, fmt.Errorf("failed to detect user ID: %w", err)
	}

	// Detect terminal PID (parent process)
	terminalPID, err := cd.detectTerminalPID()
	if err != nil {
		return nil, fmt.Errorf("failed to detect terminal PID: %w", err)
	}

	// Detect shell PID (current process)
	shellPID := os.Getpid()

	// Detect working directory
	workingDir, err := cd.detectWorkingDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to detect working directory: %w", err)
	}

	// Detect project path
	projectPath, err := cd.detectProjectPath(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project path: %w", err)
	}

	context := &entities.SessionContext{
		TerminalPID: terminalPID,
		ShellPID:    shellPID,
		WorkingDir:  workingDir,
		ProjectPath: projectPath,
		UserID:      userID,
		Timestamp:   timestamp,
	}

	return context, nil
}

/**
 * CONTEXT:   Detect user ID from environment variables with fallback strategies
 * INPUT:     No parameters, checks environment variables
 * OUTPUT:    User ID string or error if detection fails
 * BUSINESS:  Identify user for session scoping and correlation
 * CHANGE:    Initial user ID detection with multiple fallback strategies
 * RISK:      Medium - Incorrect user ID affects session isolation
 */
func (cd *ContextDetector) detectUserID() (string, error) {
	// Primary: USER environment variable (Unix systems)
	if userID := os.Getenv("USER"); userID != "" {
		return userID, nil
	}

	// Fallback 1: USERNAME environment variable (Windows systems)
	if userID := os.Getenv("USERNAME"); userID != "" {
		return userID, nil
	}

	// Fallback 2: LOGNAME environment variable
	if userID := os.Getenv("LOGNAME"); userID != "" {
		return userID, nil
	}

	// Fallback 3: Configuration fallback
	if cd.fallbackUserID != "" {
		return cd.fallbackUserID, nil
	}

	// Fallback 4: System-specific detection
	userID, err := cd.detectSystemUserID()
	if err == nil && userID != "" {
		return userID, nil
	}

	return "", fmt.Errorf("unable to detect user ID from environment")
}

/**
 * CONTEXT:   Detect terminal PID by examining parent process chain
 * INPUT:     No parameters, examines current process hierarchy
 * OUTPUT:    Terminal process PID or error if detection fails
 * BUSINESS:  Identify terminal for exact session correlation matching
 * CHANGE:    Initial terminal PID detection with cross-platform support
 * RISK:      High - Terminal PID accuracy critical for primary correlation strategy
 */
func (cd *ContextDetector) detectTerminalPID() (int, error) {
	// Get parent process PID
	ppid := os.Getppid()
	
	// Validate PPID
	if ppid <= 1 {
		return 0, fmt.Errorf("invalid parent process PID: %d", ppid)
	}

	// For most cases, the parent process is the terminal/shell
	// This works for:
	// - bash/zsh directly running the command
	// - Terminal applications spawning shells
	// - IDE terminals running commands
	return ppid, nil
}

/**
 * CONTEXT:   Detect current working directory with validation
 * INPUT:     No parameters, uses os.Getwd()
 * OUTPUT:    Absolute working directory path or error
 * BUSINESS:  Identify current project directory for project-based correlation
 * CHANGE:    Initial working directory detection with validation
 * RISK:      Medium - Working directory used for project matching
 */
func (cd *ContextDetector) detectWorkingDirectory() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(workingDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Validate directory exists
	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("working directory does not exist: %w", err)
	}

	return absPath, nil
}

/**
 * CONTEXT:   Detect project path using git repository or directory-based strategies
 * INPUT:     Working directory path as starting point for project detection
 * OUTPUT:    Project path (repository root or project directory) or error
 * BUSINESS:  Identify project boundaries for improved correlation accuracy
 * CHANGE:    Initial project path detection with git and directory strategies
 * RISK:      Low - Project path used for enhanced correlation but not critical
 */
func (cd *ContextDetector) detectProjectPath(workingDir string) (string, error) {
	// Strategy 1: Git repository root (if enabled)
	if cd.useGitRepository {
		if gitRoot, err := cd.findGitRepositoryRoot(workingDir); err == nil {
			return gitRoot, nil
		}
	}

	// Strategy 2: Look for common project indicators
	projectRoot, err := cd.findProjectRoot(workingDir)
	if err == nil {
		return projectRoot, nil
	}

	// Strategy 3: Use working directory as fallback
	return workingDir, nil
}

/**
 * CONTEXT:   Find git repository root by walking up directory tree
 * INPUT:     Starting directory path for git repository search
 * OUTPUT:    Git repository root path or error if not found
 * BUSINESS:  Use git repository boundaries to define project scope
 * CHANGE:    Initial git repository detection implementation
 * RISK:      Low - Git detection used for enhanced project identification
 */
func (cd *ContextDetector) findGitRepositoryRoot(startDir string) (string, error) {
	currentDir := startDir

	for {
		// Check if .git directory exists
		gitDir := filepath.Join(currentDir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return currentDir, nil
		}

		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		
		// Check if we've reached the root
		if parentDir == currentDir {
			break
		}
		
		currentDir = parentDir
	}

	return "", fmt.Errorf("not in a git repository")
}

/**
 * CONTEXT:   Find project root using common project indicator files
 * INPUT:     Starting directory path for project root search
 * OUTPUT:    Project root path or error if not found
 * BUSINESS:  Identify project boundaries using standard project files
 * CHANGE:    Initial project root detection with common indicators
 * RISK:      Low - Project root detection used for enhanced correlation
 */
func (cd *ContextDetector) findProjectRoot(startDir string) (string, error) {
	// Common project indicator files
	projectIndicators := []string{
		"package.json",    // Node.js projects
		"go.mod",          // Go modules
		"Cargo.toml",      // Rust projects
		"requirements.txt", // Python projects
		"pom.xml",         // Java Maven projects
		"build.gradle",    // Java Gradle projects
		"Makefile",        // Make-based projects
		"pyproject.toml",  // Python PEP 518 projects
		".project",        // Eclipse projects
		"composer.json",   // PHP Composer projects
	}

	currentDir := startDir

	for {
		// Check for project indicators
		for _, indicator := range projectIndicators {
			indicatorPath := filepath.Join(currentDir, indicator)
			if _, err := os.Stat(indicatorPath); err == nil {
				return currentDir, nil
			}
		}

		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		
		// Check if we've reached the root
		if parentDir == currentDir {
			break
		}
		
		currentDir = parentDir
	}

	return "", fmt.Errorf("project root not found")
}

// Platform-specific helper methods

func (cd *ContextDetector) detectSystemUserID() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return cd.detectWindowsUserID()
	case "linux", "darwin":
		return cd.detectUnixUserID()
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func (cd *ContextDetector) detectWindowsUserID() (string, error) {
	// Try USERPROFILE environment variable
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		// Extract username from path like C:\Users\username
		parts := strings.Split(userProfile, string(os.PathSeparator))
		if len(parts) >= 2 {
			return parts[len(parts)-1], nil
		}
	}

	return "", fmt.Errorf("unable to detect Windows user ID")
}

func (cd *ContextDetector) detectUnixUserID() (string, error) {
	// Try to read from /proc/self/stat (Linux)
	if runtime.GOOS == "linux" {
		// This is a more complex implementation that would read process info
		// For now, return error to fall back to environment variables
	}

	// Try HOME environment variable
	if home := os.Getenv("HOME"); home != "" {
		// Extract username from path like /home/username
		parts := strings.Split(home, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-1], nil
		}
	}

	return "", fmt.Errorf("unable to detect Unix user ID")
}

/**
 * CONTEXT:   Validate detected session context for consistency and completeness
 * INPUT:     SessionContext to validate
 * OUTPUT:    Error if context is invalid, nil if valid
 * BUSINESS:  Ensure detected context meets requirements for reliable correlation
 * CHANGE:    Initial context validation implementation
 * RISK:      Medium - Invalid context leads to correlation failures
 */
func (cd *ContextDetector) ValidateContext(context *entities.SessionContext) error {
	if context == nil {
		return fmt.Errorf("context cannot be nil")
	}

	if context.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if context.WorkingDir == "" {
		return fmt.Errorf("working directory cannot be empty")
	}

	if context.TerminalPID <= 0 {
		return fmt.Errorf("terminal PID must be positive, got %d", context.TerminalPID)
	}

	if context.ShellPID <= 0 {
		return fmt.Errorf("shell PID must be positive, got %d", context.ShellPID)
	}

	if context.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}

	// Validate working directory exists
	if _, err := os.Stat(context.WorkingDir); err != nil {
		return fmt.Errorf("working directory does not exist: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Create default context detector with sensible defaults
 * INPUT:     No parameters, uses default configuration
 * OUTPUT:    ContextDetector with default settings
 * BUSINESS:  Provide easy-to-use detector for common use cases
 * CHANGE:    Initial default configuration implementation
 * RISK:      Low - Default configuration for convenience
 */
func NewDefaultContextDetector() *ContextDetector {
	return NewContextDetector(ContextDetectorConfig{
		DetectProjectName: true,
		UseGitRepository:  true,
		FallbackUserID:    "unknown",
	})
}