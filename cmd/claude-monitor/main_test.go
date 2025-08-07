/**
 * CONTEXT:   Comprehensive test suite for Claude Code hook integration
 * INPUT:     Various project structures and execution scenarios
 * OUTPUT:    Test coverage for hook reliability and performance requirements
 * BUSINESS:  Ensure hook works reliably across different project types and conditions
 * CHANGE:    Initial test implementation with performance and integration testing
 * RISK:      Low - Testing framework to prevent regressions and ensure reliability
 */

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * CONTEXT:   Test project detection logic with various directory structures
 * INPUT:     Different project types and directory layouts
 * OUTPUT:    Validation that project detection works correctly
 * BUSINESS:  Accurate project detection is critical for work organization
 * CHANGE:    Initial project detection tests covering common scenarios
 * RISK:      Medium - Project detection affects work tracking accuracy
 */
func TestProjectDetection(t *testing.T) {
	tests := []struct {
		name           string
		structure      map[string]string // file -> content
		workingDir     string
		expectedName   string
		expectedType   entities.ProjectType
		expectGitBranch bool
	}{
		{
			name: "go_project_with_modules",
			structure: map[string]string{
				"go.mod":    "module github.com/example/myproject\n\ngo 1.21\n",
				"go.sum":    "// go.sum content",
				"main.go":   "package main\n\nfunc main() {}",
			},
			workingDir:   "myproject",
			expectedName: "myproject",
			expectedType: entities.ProjectTypeGo,
		},
		{
			name: "rust_project_with_cargo",
			structure: map[string]string{
				"Cargo.toml": "[package]\nname = \"awesome-tool\"\nversion = \"0.1.0\"",
				"Cargo.lock": "# Cargo.lock content",
				"src/main.rs": "fn main() {}",
			},
			workingDir:   "awesome-tool",
			expectedName: "awesome-tool",
			expectedType: entities.ProjectTypeRust,
		},
		{
			name: "npm_javascript_project",
			structure: map[string]string{
				"package.json": `{"name": "my-webapp", "version": "1.0.0"}`,
				"index.js":     "console.log('hello world');",
				"node_modules/react/package.json": `{"name": "react"}`,
			},
			workingDir:   "my-webapp",
			expectedName: "my-webapp",
			expectedType: entities.ProjectTypeJavaScript,
		},
		{
			name: "typescript_project",
			structure: map[string]string{
				"package.json":   `{"name": "ts-project", "version": "1.0.0"}`,
				"tsconfig.json":  `{"compilerOptions": {"target": "es2020"}}`,
				"src/index.ts":   "const greeting: string = 'Hello World';",
			},
			workingDir:   "ts-project",
			expectedName: "ts-project",
			expectedType: entities.ProjectTypeTypeScript,
		},
		{
			name: "python_project_with_requirements",
			structure: map[string]string{
				"requirements.txt": "fastapi==0.68.0\nuvicorn==0.15.0",
				"main.py":          "from fastapi import FastAPI\napp = FastAPI()",
				"setup.py":         "from setuptools import setup\nsetup(name='myapi')",
			},
			workingDir:   "myapi",
			expectedName: "myapi",
			expectedType: entities.ProjectTypePython,
		},
		{
			name: "web_project_with_html",
			structure: map[string]string{
				"index.html":       "<!DOCTYPE html><html><head><title>Test</title></head></html>",
				"style.css":        "body { margin: 0; }",
				"webpack.config.js": "module.exports = {};",
			},
			workingDir:   "my-website",
			expectedName: "my-website",
			expectedType: entities.ProjectTypeWeb,
		},
		{
			name: "git_project_with_branch",
			structure: map[string]string{
				".git/HEAD":   "ref: refs/heads/feature/awesome-feature",
				".git/config": "[core]\n\trepositoryformatversion = 0",
				"README.md":   "# My Project",
			},
			workingDir:     "my-repo",
			expectedName:   "my-repo",
			expectedType:   entities.ProjectTypeGeneral,
			expectGitBranch: true,
		},
		{
			name: "nested_src_directory",
			structure: map[string]string{
				"../go.mod": "module github.com/example/parent-project\n",
				"main.go":   "package main",
			},
			workingDir:   "parent-project", // Should detect parent, not "src"
			expectedName: "parent-project",
			expectedType: entities.ProjectTypeGo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tempDir := setupTestDir(t, tt.structure, tt.workingDir)
			defer os.RemoveAll(tempDir)

			// Change to the working directory
			workDir := filepath.Join(tempDir, tt.workingDir)
			originalWd, _ := os.Getwd()
			defer os.Chdir(originalWd)

			if err := os.Chdir(workDir); err != nil {
				t.Fatalf("Failed to change to working directory: %v", err)
			}

			// Test project detection
			projectInfo := detectProjectInfo(workDir)

			assert.Equal(t, tt.expectedName, projectInfo.Name, "Project name mismatch")
			assert.Equal(t, tt.expectedType, projectInfo.Type, "Project type mismatch")

			if tt.expectGitBranch {
				assert.NotEmpty(t, projectInfo.GitBranch, "Expected git branch to be detected")
			}

			// Verify path is absolute
			assert.True(t, filepath.IsAbs(projectInfo.Path), "Project path should be absolute")
		})
	}
}

/**
 * CONTEXT:   Test HTTP communication with daemon including timeout behavior
 * INPUT:     Mock HTTP server responses and network conditions
 * OUTPUT:    Validation of daemon communication reliability and error handling
 * BUSINESS:  Daemon communication is primary method for real-time tracking
 * CHANGE:    Initial daemon communication tests with various scenarios
 * RISK:      High - Network issues could affect hook performance
 */
func TestDaemonCommunication(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
		expectedError  string
	}{
		{
			name: "successful_communication",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Validate request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/activity", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Contains(t, r.Header.Get("User-Agent"), "claude-monitor-hook")

				// Parse and validate request body
				var req ActivityEventRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)

				assert.NotEmpty(t, req.UserID)
				assert.NotEmpty(t, req.ProjectName)
				assert.Equal(t, "command", req.ActivityType)
				assert.Equal(t, "hook", req.ActivitySource)
				assert.False(t, req.Timestamp.IsZero())

				// Send success response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":       "processed",
					"activity_id":  "test-id",
					"timestamp":    time.Now(),
					"processing_ms": 15,
				})
			},
			expectError: false,
		},
		{
			name: "daemon_returns_error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "internal server error",
				})
			},
			expectError:   true,
			expectedError: "daemon returned status 500",
		},
		{
			name: "daemon_returns_bad_request",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid request format",
				})
			},
			expectError:   true,
			expectedError: "daemon returned status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			// Create test activity event
			event, err := entities.NewActivityEventFromEnvironment("test-command", "test description")
			require.NoError(t, err)

			// Override daemon URL for testing
			originalURL := *daemonURL
			*daemonURL = server.URL
			defer func() { *daemonURL = originalURL }()

			// Test daemon communication
			config := getDefaultConfig()
			err = sendToDaemon(event, config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

/**
 * CONTEXT:   Test timeout behavior to ensure hook execution stays under 50ms
 * INPUT:     Slow daemon responses and network delays
 * OUTPUT:    Validation that timeouts work correctly and hook remains fast
 * BUSINESS:  Hook must never slow down Claude Code user experience
 * CHANGE:    Initial timeout testing for performance requirements
 * RISK:      High - Slow hooks could make Claude Code unusable
 */
func TestTimeoutBehavior(t *testing.T) {
	// Create slow server that exceeds timeout
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Longer than our 100ms timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	// Create test event
	event, err := entities.NewActivityEventFromEnvironment("test-command", "test description")
	require.NoError(t, err)

	// Set short timeout for testing
	originalTimeout := *timeout
	*timeout = 50 * time.Millisecond
	defer func() { *timeout = originalTimeout }()

	// Override daemon URL
	originalURL := *daemonURL
	*daemonURL = slowServer.URL
	defer func() { *daemonURL = originalURL }()

	// Test that request times out
	start := time.Now()
	config := getDefaultConfig()
	err = sendToDaemon(event, config)
	duration := time.Since(start)

	// Should fail due to timeout
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	
	// Should not take much longer than timeout
	assert.Less(t, duration, 100*time.Millisecond, "Request should timeout quickly")
}

/**
 * CONTEXT:   Test fallback logging when daemon is unavailable
 * INPUT:     Daemon unavailability scenarios and file system conditions
 * OUTPUT:    Validation that activities are always captured via fallback
 * BUSINESS:  Zero data loss requirement when daemon is down
 * CHANGE:    Initial fallback logging tests with file system validation
 * RISK:      Medium - File system issues could cause data loss
 */
func TestFallbackLogging(t *testing.T) {
	// Create temporary directory for log file
	tempDir, err := os.MkdirTemp("", "claude-monitor-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logFile := filepath.Join(tempDir, "activity.log")

	// Create test event
	event, err := entities.NewActivityEventFromEnvironment("test-command", "test description")
	require.NoError(t, err)

	// Override log file path
	originalLogFile := *logFile
	*logFile = logFile
	defer func() { *logFile = originalLogFile }()

	// Test fallback logging
	config := getDefaultConfig()
	writeToLocalFile(event, config)

	// Verify log file was created and contains data
	assert.FileExists(t, logFile)

	// Read and validate log content
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	lines := strings.Split(string(content), "\n")
	assert.GreaterOrEqual(t, len(lines), 1, "Should have at least one log line")

	// Parse the log entry
	var logEntry ActivityLogEntry
	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "hook_fallback", logEntry.Source)
	assert.Equal(t, Version, logEntry.Version)
	assert.Equal(t, event.ID(), logEntry.Event.ID)
	assert.Equal(t, event.UserID(), logEntry.Event.UserID)
}

/**
 * CONTEXT:   Test configuration loading and customization
 * INPUT:     Various configuration file contents and missing file scenarios
 * OUTPUT:    Validation that configuration works correctly with proper defaults
 * BUSINESS:  Configuration allows users to customize hook behavior
 * CHANGE:    Initial configuration testing with validation
 * RISK:      Low - Configuration affects behavior but has safe defaults
 */
func TestConfigurationLoading(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectEnabled  bool
		expectTimeout  int
		expectDaemonURL string
	}{
		{
			name: "default_config",
			configContent: "",
			expectEnabled: true,
			expectTimeout: 100,
			expectDaemonURL: "http://localhost:8080",
		},
		{
			name: "custom_config",
			configContent: `{
				"enabled": true,
				"daemon_url": "http://custom:9090",
				"timeout_ms": 200,
				"log_level": "debug",
				"project_names": {
					"/home/user/project": "MyProject"
				}
			}`,
			expectEnabled: true,
			expectTimeout: 200,
			expectDaemonURL: "http://custom:9090",
		},
		{
			name: "disabled_config",
			configContent: `{
				"enabled": false
			}`,
			expectEnabled: false,
		},
		{
			name: "invalid_json",
			configContent: `{invalid json`,
			expectEnabled: true, // Should fall back to defaults
			expectTimeout: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config directory
			tempDir, err := os.MkdirTemp("", "claude-monitor-config-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			configDir := filepath.Join(tempDir, ".claude-monitor")
			err = os.MkdirAll(configDir, 0755)
			require.NoError(t, err)

			if tt.configContent != "" {
				configFile := filepath.Join(configDir, "config.json")
				err = os.WriteFile(configFile, []byte(tt.configContent), 0644)
				require.NoError(t, err)
			}

			// Override home directory for testing
			originalHome := os.Getenv("HOME")
			os.Setenv("HOME", tempDir)
			defer os.Setenv("HOME", originalHome)

			// Load configuration
			config := loadConfig()

			// Validate configuration
			assert.Equal(t, tt.expectEnabled, config.Enabled)
			if tt.expectTimeout > 0 {
				assert.Equal(t, tt.expectTimeout, config.TimeoutMS)
			}
			if tt.expectDaemonURL != "" {
				assert.Equal(t, tt.expectDaemonURL, config.DaemonURL)
			}
		})
	}
}

/**
 * CONTEXT:   Performance test to ensure hook execution stays under 50ms
 * INPUT:     Normal hook execution scenarios with timing measurement
 * OUTPUT:    Validation that performance requirements are met
 * BUSINESS:  Performance is critical for user experience with Claude Code
 * CHANGE:    Initial performance testing with timing validation
 * RISK:      High - Slow performance would make Claude Code unusable
 */
func TestPerformanceRequirements(t *testing.T) {
	// Create fast mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	// Override daemon URL
	originalURL := *daemonURL
	*daemonURL = server.URL
	defer func() { *daemonURL = originalURL }()

	// Run multiple iterations to get average performance
	iterations := 10
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		// Create and send activity event
		event, err := entities.NewActivityEventFromEnvironment("test-command", "performance test")
		require.NoError(t, err)

		config := getDefaultConfig()
		err = sendToDaemon(event, config)
		require.NoError(t, err)

		duration := time.Since(start)
		totalDuration += duration

		// Each individual execution should be under 50ms
		assert.Less(t, duration, 50*time.Millisecond, 
			"Hook execution took %v, must be under 50ms", duration)
	}

	averageDuration := totalDuration / time.Duration(iterations)
	t.Logf("Average hook execution time: %v", averageDuration)
	
	// Average should also be well under 50ms
	assert.Less(t, averageDuration, 30*time.Millisecond, 
		"Average hook execution time should be well under 50ms")
}

// Helper functions for testing

func setupTestDir(t *testing.T, structure map[string]string, workingDir string) string {
	tempDir, err := os.MkdirTemp("", "claude-monitor-test-*")
	require.NoError(t, err)

	baseDir := filepath.Join(tempDir, workingDir)
	err = os.MkdirAll(baseDir, 0755)
	require.NoError(t, err)

	for filePath, content := range structure {
		fullPath := filepath.Join(tempDir, filePath)
		
		// Create directory if needed
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	return tempDir
}

/**
 * CONTEXT:   Integration test with real daemon for end-to-end validation
 * INPUT:     Real HTTP daemon and activity event processing
 * OUTPUT:    Validation that hook integrates correctly with complete system
 * BUSINESS:  End-to-end integration ensures real-world functionality
 * CHANGE:    Initial integration test requiring running daemon
 * RISK:      Medium - Integration test requires external daemon process
 */
func TestIntegrationWithDaemon(t *testing.T) {
	// Skip if running in CI or if daemon not available
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Check if daemon is running by attempting health check
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		t.Skip("Daemon not running, skipping integration test")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skip("Daemon not healthy, skipping integration test")
	}

	// Create test event
	event, err := entities.NewActivityEventFromEnvironment("integration-test", "testing hook integration")
	require.NoError(t, err)

	// Send to real daemon
	config := getDefaultConfig()
	err = sendToDaemon(event, config)
	assert.NoError(t, err, "Should successfully send to running daemon")

	// Verify daemon received the event by checking status
	statusResp, err := client.Get(fmt.Sprintf("http://localhost:8080/status?user_id=%s", event.UserID()))
	require.NoError(t, err)
	defer statusResp.Body.Close()

	assert.Equal(t, http.StatusOK, statusResp.StatusCode, "Status endpoint should return OK")

	// Parse status response
	var statusData map[string]interface{}
	body, err := io.ReadAll(statusResp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(body, &statusData)
	require.NoError(t, err)

	// Verify we have a user status response
	assert.Equal(t, event.UserID(), statusData["user_id"])
	assert.Contains(t, statusData, "timestamp")
}