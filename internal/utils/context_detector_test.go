/**
 * CONTEXT:   Comprehensive tests for automatic context detection system
 * INPUT:     Various environment configurations and process states for testing
 * OUTPUT:    Validated context detection accuracy across different scenarios
 * BUSINESS:  Ensure automatic context detection works reliably for hook correlation
 * CHANGE:    Initial comprehensive test suite for context detection utilities
 * RISK:      High - Context detection accuracy critical for correlation system reliability
 */

package utils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * CONTEXT:   Test basic context detection with valid environment
 * INPUT:     Standard environment with USER variable and valid working directory
 * OUTPUT:    Successfully detected session context with all required fields
 * BUSINESS:  Verify basic context detection works in normal environments
 * CHANGE:    Initial test for successful context detection path
 * RISK:      Medium - Basic functionality test for context detection
 */
func TestDetectSessionContext_Success(t *testing.T) {
	// Set up test environment
	originalUser := os.Getenv("USER")
	defer func() {
		if originalUser != "" {
			os.Setenv("USER", originalUser)
		} else {
			os.Unsetenv("USER")
		}
	}()
	
	os.Setenv("USER", "testuser")

	detector := NewDefaultContextDetector()
	
	// Execute
	context, err := detector.DetectSessionContext()
	
	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, context)
	
	// Check required fields
	assert.Equal(t, "testuser", context.UserID)
	assert.Greater(t, context.TerminalPID, 0)
	assert.Greater(t, context.ShellPID, 0)
	assert.NotEmpty(t, context.WorkingDir)
	assert.NotEmpty(t, context.ProjectPath)
	assert.False(t, context.Timestamp.IsZero())
	
	// Check timestamp is recent
	assert.WithinDuration(t, time.Now(), context.Timestamp, 1*time.Second)
}

/**
 * CONTEXT:   Test user ID detection with various environment variable configurations
 * INPUT:     Different combinations of USER, USERNAME, and LOGNAME environment variables
 * OUTPUT:    Correct user ID detected with proper fallback behavior
 * BUSINESS:  Verify user ID detection works across different operating systems and configurations
 * CHANGE:    Initial test for cross-platform user ID detection
 * RISK:      Medium - User ID detection affects session isolation and correlation
 */
func TestDetectUserID_EnvironmentVariables(t *testing.T) {
	// Save original environment
	originalVars := map[string]string{
		"USER":     os.Getenv("USER"),
		"USERNAME": os.Getenv("USERNAME"),
		"LOGNAME":  os.Getenv("LOGNAME"),
	}
	defer func() {
		for key, value := range originalVars {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	tests := []struct {
		name           string
		userVar        string
		usernameVar    string
		lognameVar     string
		fallbackUserID string
		expectedUserID string
		shouldError    bool
	}{
		{
			name:           "USER variable set",
			userVar:        "testuser",
			usernameVar:    "",
			lognameVar:     "",
			expectedUserID: "testuser",
			shouldError:    false,
		},
		{
			name:           "USERNAME fallback",
			userVar:        "",
			usernameVar:    "windowsuser",
			lognameVar:     "",
			expectedUserID: "windowsuser",
			shouldError:    false,
		},
		{
			name:           "LOGNAME fallback",
			userVar:        "",
			usernameVar:    "",
			lognameVar:     "unixuser",
			expectedUserID: "unixuser",
			shouldError:    false,
		},
		{
			name:           "Fallback user ID",
			userVar:        "",
			usernameVar:    "",
			lognameVar:     "",
			fallbackUserID: "fallbackuser",
			expectedUserID: "fallbackuser",
			shouldError:    false,
		},
		{
			name:        "No user ID available",
			userVar:     "",
			usernameVar: "",
			lognameVar:  "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all environment variables
			os.Unsetenv("USER")
			os.Unsetenv("USERNAME")
			os.Unsetenv("LOGNAME")

			// Set test variables
			if tt.userVar != "" {
				os.Setenv("USER", tt.userVar)
			}
			if tt.usernameVar != "" {
				os.Setenv("USERNAME", tt.usernameVar)
			}
			if tt.lognameVar != "" {
				os.Setenv("LOGNAME", tt.lognameVar)
			}

			// Create detector with fallback
			config := ContextDetectorConfig{
				DetectProjectName: true,
				UseGitRepository:  true,
				FallbackUserID:    tt.fallbackUserID,
			}
			detector := NewContextDetector(config)

			// Execute
			userID, err := detector.detectUserID()

			// Assertions
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedUserID, userID)
			}
		})
	}
}

/**
 * CONTEXT:   Test terminal PID detection accuracy
 * INPUT:     Current process environment for parent PID detection
 * OUTPUT:    Valid terminal PID that represents parent process
 * BUSINESS:  Verify terminal PID detection provides reliable process identification
 * CHANGE:    Initial test for terminal PID detection logic
 * RISK:      High - Terminal PID accuracy critical for primary correlation strategy
 */
func TestDetectTerminalPID_Success(t *testing.T) {
	detector := NewDefaultContextDetector()
	
	// Execute
	terminalPID, err := detector.detectTerminalPID()
	
	// Assertions
	require.NoError(t, err)
	assert.Greater(t, terminalPID, 1, "Terminal PID should be greater than 1")
	
	// Should be different from current process PID
	currentPID := os.Getpid()
	assert.NotEqual(t, currentPID, terminalPID, "Terminal PID should not be current process PID")
}

/**
 * CONTEXT:   Test working directory detection with various directory states
 * INPUT:     Different working directory configurations and permissions
 * OUTPUT:    Valid absolute working directory path
 * BUSINESS:  Verify working directory detection provides absolute paths for project identification
 * CHANGE:    Initial test for working directory detection logic
 * RISK:      Medium - Working directory accuracy affects project-based correlation
 */
func TestDetectWorkingDirectory_Success(t *testing.T) {
	detector := NewDefaultContextDetector()
	
	// Execute
	workingDir, err := detector.detectWorkingDirectory()
	
	// Assertions
	require.NoError(t, err)
	assert.NotEmpty(t, workingDir)
	
	// Should be absolute path
	assert.True(t, filepath.IsAbs(workingDir), "Working directory should be absolute path")
	
	// Directory should exist
	_, statErr := os.Stat(workingDir)
	assert.NoError(t, statErr, "Working directory should exist")
}

/**
 * CONTEXT:   Test project path detection with git repository
 * INPUT:     Directory structure with git repository for project detection
 * OUTPUT:    Correctly identified git repository root as project path
 * BUSINESS:  Verify git repository detection provides accurate project boundaries
 * CHANGE:    Initial test for git-based project path detection
 * RISK:      Low - Project path used for enhanced correlation accuracy
 */
func TestDetectProjectPath_GitRepository(t *testing.T) {
	// Create temporary directory structure with git repository
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	err := os.Mkdir(gitDir, 0755)
	require.NoError(t, err)
	
	// Create subdirectory
	subDir := filepath.Join(tempDir, "subproject")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	detector := NewContextDetector(ContextDetectorConfig{
		UseGitRepository: true,
	})
	
	// Execute from subdirectory
	projectPath, err := detector.detectProjectPath(subDir)
	
	// Assertions
	require.NoError(t, err)
	assert.Equal(t, tempDir, projectPath, "Should detect git repository root")
}

/**
 * CONTEXT:   Test project path detection with project indicator files
 * INPUT:     Directory structure with common project files (go.mod, package.json, etc.)
 * OUTPUT:    Correctly identified project root based on indicator files
 * BUSINESS:  Verify project indicator detection works across different project types
 * CHANGE:    Initial test for project indicator file detection
 * RISK:      Low - Project indicators enhance correlation but not critical
 */
func TestDetectProjectPath_ProjectIndicators(t *testing.T) {
	tests := []struct {
		name          string
		indicatorFile string
	}{
		{"Go module", "go.mod"},
		{"Node.js project", "package.json"},
		{"Python project", "requirements.txt"},
		{"Rust project", "Cargo.toml"},
		{"Makefile project", "Makefile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tempDir := t.TempDir()
			subDir := filepath.Join(tempDir, "subdir")
			err := os.Mkdir(subDir, 0755)
			require.NoError(t, err)

			// Create indicator file in root
			indicatorPath := filepath.Join(tempDir, tt.indicatorFile)
			file, err := os.Create(indicatorPath)
			require.NoError(t, err)
			file.Close()

			detector := NewContextDetector(ContextDetectorConfig{
				UseGitRepository: false, // Use project indicators instead
			})
			
			// Execute from subdirectory
			projectPath, err := detector.detectProjectPath(subDir)
			
			// Assertions
			require.NoError(t, err)
			assert.Equal(t, tempDir, projectPath, "Should detect project root with %s", tt.indicatorFile)
		})
	}
}

/**
 * CONTEXT:   Test project path detection fallback behavior
 * INPUT:     Directory without git repository or project indicators
 * OUTPUT:    Working directory used as fallback project path
 * BUSINESS:  Verify project path detection provides reliable fallback when indicators not found
 * CHANGE:    Initial test for project path detection fallback logic
 * RISK:      Low - Fallback behavior for edge cases in project detection
 */
func TestDetectProjectPath_Fallback(t *testing.T) {
	// Create temporary directory without any project indicators
	tempDir := t.TempDir()

	detector := NewDefaultContextDetector()
	
	// Execute
	projectPath, err := detector.detectProjectPath(tempDir)
	
	// Assertions
	require.NoError(t, err)
	assert.Equal(t, tempDir, projectPath, "Should fallback to working directory")
}

/**
 * CONTEXT:   Test session context validation with various invalid inputs
 * INPUT:     SessionContext with missing or invalid fields
 * OUTPUT:    Appropriate validation errors for each invalid case
 * BUSINESS:  Verify context validation prevents invalid contexts from causing correlation issues
 * CHANGE:    Initial test for comprehensive context validation
 * RISK:      Medium - Context validation prevents correlation failures
 */
func TestValidateContext_InvalidInputs(t *testing.T) {
	detector := NewDefaultContextDetector()

	tests := []struct {
		name           string
		context        *entities.SessionContext
		expectedError  string
	}{
		{
			name:          "Nil context",
			context:       nil,
			expectedError: "context cannot be nil",
		},
		{
			name: "Empty user ID",
			context: &entities.SessionContext{
				UserID:      "",
				WorkingDir:  "/test",
				TerminalPID: 1234,
				ShellPID:    5678,
				Timestamp:   time.Now(),
			},
			expectedError: "user ID cannot be empty",
		},
		{
			name: "Empty working directory",
			context: &entities.SessionContext{
				UserID:      "testuser",
				WorkingDir:  "",
				TerminalPID: 1234,
				ShellPID:    5678,
				Timestamp:   time.Now(),
			},
			expectedError: "working directory cannot be empty",
		},
		{
			name: "Invalid terminal PID",
			context: &entities.SessionContext{
				UserID:      "testuser",
				WorkingDir:  "/test",
				TerminalPID: -1,
				ShellPID:    5678,
				Timestamp:   time.Now(),
			},
			expectedError: "terminal PID must be positive",
		},
		{
			name: "Invalid shell PID",
			context: &entities.SessionContext{
				UserID:      "testuser",
				WorkingDir:  "/test",
				TerminalPID: 1234,
				ShellPID:    0,
				Timestamp:   time.Now(),
			},
			expectedError: "shell PID must be positive",
		},
		{
			name: "Zero timestamp",
			context: &entities.SessionContext{
				UserID:      "testuser",
				WorkingDir:  "/test",
				TerminalPID: 1234,
				ShellPID:    5678,
				Timestamp:   time.Time{},
			},
			expectedError: "timestamp cannot be zero",
		},
		{
			name: "Future timestamp",
			context: &entities.SessionContext{
				UserID:      "testuser",
				WorkingDir:  "/test",
				TerminalPID: 1234,
				ShellPID:    5678,
				Timestamp:   time.Now().Add(2 * time.Hour),
			},
			expectedError: "timestamp cannot be more than 1 hour in the future",
		},
		{
			name: "Old timestamp",
			context: &entities.SessionContext{
				UserID:      "testuser",
				WorkingDir:  "/test",
				TerminalPID: 1234,
				ShellPID:    5678,
				Timestamp:   time.Now().Add(-25 * time.Hour),
			},
			expectedError: "timestamp cannot be more than 24 hours in the past",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			err := detector.ValidateContext(tt.context)
			
			// Assertions
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

/**
 * CONTEXT:   Test valid session context validation
 * INPUT:     SessionContext with all valid fields and values
 * OUTPUT:    Successful validation with no errors
 * BUSINESS:  Verify context validation accepts valid contexts for correlation
 * CHANGE:    Initial test for successful context validation
 * RISK:      Low - Positive validation test case
 */
func TestValidateContext_ValidInput(t *testing.T) {
	detector := NewDefaultContextDetector()
	
	// Create temporary directory for working directory validation
	tempDir := t.TempDir()

	context := &entities.SessionContext{
		UserID:      "testuser",
		WorkingDir:  tempDir,
		ProjectPath: tempDir,
		TerminalPID: 1234,
		ShellPID:    5678,
		Timestamp:   time.Now(),
	}

	// Execute
	err := detector.ValidateContext(context)
	
	// Assertions
	assert.NoError(t, err)
}

/**
 * CONTEXT:   Test context detector configuration variations
 * INPUT:     Different ContextDetectorConfig settings
 * OUTPUT:    Context detection behavior changes based on configuration
 * BUSINESS:  Verify context detector can be configured for different environments
 * CHANGE:    Initial test for configurable context detection behavior
 * RISK:      Low - Configuration validation for flexibility
 */
func TestContextDetectorConfig_Variations(t *testing.T) {
	tests := []struct {
		name           string
		config         ContextDetectorConfig
		expectGitCheck bool
		expectFallback string
	}{
		{
			name: "Git repository enabled",
			config: ContextDetectorConfig{
				UseGitRepository: true,
				FallbackUserID:   "fallback",
			},
			expectGitCheck: true,
			expectFallback: "fallback",
		},
		{
			name: "Git repository disabled",
			config: ContextDetectorConfig{
				UseGitRepository: false,
				FallbackUserID:   "different",
			},
			expectGitCheck: false,
			expectFallback: "different",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewContextDetector(tt.config)
			
			// Test configuration is applied
			assert.Equal(t, tt.config.UseGitRepository, detector.useGitRepository)
			assert.Equal(t, tt.expectFallback, detector.fallbackUserID)
		})
	}
}

/**
 * CONTEXT:   Test default context detector configuration
 * INPUT:     No configuration provided, uses defaults
 * OUTPUT:    Context detector with sensible default configuration
 * BUSINESS:  Verify default detector configuration works for common use cases
 * CHANGE:    Initial test for default configuration behavior
 * RISK:      Low - Default configuration test for convenience
 */
func TestNewDefaultContextDetector(t *testing.T) {
	detector := NewDefaultContextDetector()
	
	// Test default configuration values
	assert.True(t, detector.detectProjectName)
	assert.True(t, detector.useGitRepository)
	assert.Equal(t, "unknown", detector.fallbackUserID)
}

/**
 * CONTEXT:   Integration test for complete context detection flow
 * INPUT:     Real environment with actual working directory and process state
 * OUTPUT:    Complete session context ready for correlation
 * BUSINESS:  Verify end-to-end context detection works in realistic conditions
 * CHANGE:    Initial integration test for complete context detection
 * RISK:      Medium - Integration test validates entire context detection system
 */
func TestCompleteContextDetectionFlow(t *testing.T) {
	// Set up environment
	originalUser := os.Getenv("USER")
	defer func() {
		if originalUser != "" {
			os.Setenv("USER", originalUser)
		} else {
			os.Unsetenv("USER")
		}
	}()
	os.Setenv("USER", "integrationtest")

	detector := NewDefaultContextDetector()
	
	// Execute complete detection
	sessionContext, err := detector.DetectSessionContext()
	require.NoError(t, err)
	
	// Validate complete context
	err = detector.ValidateContext(sessionContext)
	require.NoError(t, err)
	
	// Test context fields are reasonable
	assert.Equal(t, "integrationtest", sessionContext.UserID)
	assert.Greater(t, sessionContext.TerminalPID, 1)
	assert.Greater(t, sessionContext.ShellPID, 1)
	assert.True(t, filepath.IsAbs(sessionContext.WorkingDir))
	assert.True(t, filepath.IsAbs(sessionContext.ProjectPath))
	assert.WithinDuration(t, time.Now(), sessionContext.Timestamp, 1*time.Second)
	
	// Test that context can be used for correlation
	// (This would typically be done in the tracker tests, but we can verify basic usability)
	assert.NotEmpty(t, sessionContext.UserID)
	assert.NotEmpty(t, sessionContext.WorkingDir)
	assert.Greater(t, sessionContext.TerminalPID, 0)
	assert.Greater(t, sessionContext.ShellPID, 0)
}