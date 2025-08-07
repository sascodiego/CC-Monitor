/**
 * CONTEXT:   Helper functions for Claude Monitor single binary implementation
 * INPUT:     Various system operations and data processing requirements
 * OUTPUT:    Utility functions supporting installation, HTTP, and formatting
 * BUSINESS:  Support functions enable clean separation of concerns
 * CHANGE:    Initial helper implementation with comprehensive functionality
 * RISK:      Low - Utility functions with focused responsibilities
 */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

/**
 * CONTEXT:   HTTP client for daemon communication with timeout management
 * INPUT:     HTTP requests with strict timeout requirements
 * OUTPUT:    HTTP responses with proper error handling and timeout
 * BUSINESS:  HTTP communication provides real-time data exchange with daemon
 * CHANGE:    Initial HTTP client with configurable timeout and error handling
 * RISK:      Medium - Network communication affecting performance and reliability
 */
type HTTPClient struct {
	client  *http.Client
	timeout time.Duration
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

func (c *HTTPClient) SendActivityEvent(url string, event *ActivityEvent) error {
	// Convert to request format
	data := ActivityEventRequest{
		UserID:         event.UserID,
		ProjectPath:    event.ProjectPath,
		ProjectName:    event.ProjectName,
		ActivityType:   event.ActivityType,
		ActivitySource: event.ActivitySource,
		Timestamp:      event.Timestamp,
		Command:        event.Command,
		Description:    event.Description,
		Metadata:       event.Metadata,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("claude-monitor/%s", Version))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *HTTPClient) GetDailyReport(baseURL string, date time.Time) (*DailyReport, error) {
	url := fmt.Sprintf("%s/reports/daily?date=%s", baseURL, date.Format("2006-01-02"))
	
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	var report DailyReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("response decode error: %w", err)
	}

	return &report, nil
}

func (c *HTTPClient) GetWeeklyReport(baseURL string, date time.Time) (*WeeklyReport, error) {
	url := fmt.Sprintf("%s/reports/weekly?date=%s", baseURL, date.Format("2006-01-02"))
	
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	var report WeeklyReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("response decode error: %w", err)
	}

	
	return &report, nil
}

func (c *HTTPClient) GetMonthlyReport(baseURL string, month time.Time) (*MonthlyReport, error) {
	url := fmt.Sprintf("%s/reports/monthly?month=%s", baseURL, month.Format("2006-01"))
	
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	var report MonthlyReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return nil, fmt.Errorf("response decode error: %w", err)
	}

	return &report, nil
}

func (c *HTTPClient) GetHealthStatus(baseURL string) (*HealthStatus, error) {
	url := fmt.Sprintf("%s/health", baseURL)
	
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	var health HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("response decode error: %w", err)
	}

	return &health, nil
}

func (c *HTTPClient) GetRecentActivity(baseURL string, limit int) ([]RecentActivity, error) {
	url := fmt.Sprintf("%s/activity/recent?limit=%d", baseURL, limit)
	
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	var activities []RecentActivity
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		return nil, fmt.Errorf("response decode error: %w", err)
	}

	return activities, nil
}

func (c *HTTPClient) GetPendingSessions(baseURL string) ([]PendingSessionSummary, error) {
	url := fmt.Sprintf("%s/sessions/pending", baseURL)
	
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	var sessions []PendingSessionSummary
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("response decode error: %w", err)
	}

	return sessions, nil
}

func (c *HTTPClient) CloseAllPendingSessions(baseURL string) (*CloseSessionsResult, error) {
	url := fmt.Sprintf("%s/sessions/close-all", baseURL)
	
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("daemon returned status %d", resp.StatusCode)
	}

	var result CloseSessionsResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("response decode error: %w", err)
	}

	return &result, nil
}

/**
 * CONTEXT:   Binary installation with cross-platform file operations
 * INPUT:     Source binary path and target installation location
 * OUTPUT:    Copied binary with proper permissions and validation
 * BUSINESS:  Binary installation enables system integration and PATH access
 * CHANGE:    Initial binary copy implementation with permission handling
 * RISK:      High - File system operations with potential permission issues
 */
func copyBinaryToTarget(source, target string) error {
	// Create target directory if needed
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Open source file
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer sourceFile.Close()

	// Create target file
	targetFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create target: %w", err)
	}
	defer targetFile.Close()

	// Copy data
	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Set executable permissions
	if runtime.GOOS != "windows" {
		if err := os.Chmod(target, 0755); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	return nil
}

/**
 * CONTEXT:   Configuration directory creation with proper structure
 * INPUT:     User home directory and configuration requirements
 * OUTPUT:    Created configuration directory with proper permissions
 * BUSINESS:  Configuration directory provides persistent storage for user settings
 * CHANGE:    Initial directory creation with standard structure
 * RISK:      Low - Directory creation with standard permissions
 */
func createConfigurationDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".claude-monitor")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"data", "logs", "cache"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(configDir, subdir), 0755); err != nil {
			return "", fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}

	return configDir, nil
}

/**
 * CONTEXT:   Configuration file generation from embedded templates
 * INPUT:     Configuration directory and embedded template assets
 * OUTPUT:    Generated configuration files with user-editable defaults
 * BUSINESS:  Configuration files enable user customization while providing defaults
 * CHANGE:    Initial file generation with template expansion
 * RISK:      Low - File creation with error handling and validation
 */
func generateConfigurationFiles(configDir string) error {
	// Generate main configuration file
	configPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(configPath); err == nil {
		// Config file already exists, don't overwrite
		return nil
	}

	configData, err := assets.ReadFile("assets/config-template.json")
	if err != nil {
		return fmt.Errorf("failed to read config template: %w", err)
	}

	// Expand placeholders in config template
	configContent := string(configData)
	configContent = strings.ReplaceAll(configContent, "~/.claude-monitor/data/claude.db", 
		filepath.Join(configDir, "data", "claude.db"))

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Copy Claude Code integration guide
	guidePath := filepath.Join(configDir, "claude-code-integration.md")
	guideData, err := assets.ReadFile("assets/claude-code-integration.md")
	if err != nil {
		return fmt.Errorf("failed to read integration guide: %w", err)
	}

	if err := os.WriteFile(guidePath, guideData, 0644); err != nil {
		return fmt.Errorf("failed to write integration guide: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Database initialization with embedded schema
 * INPUT:     Database path and embedded schema file
 * OUTPUT:    Initialized KuzuDB database with complete schema
 * BUSINESS:  Database provides persistent storage for work hour analytics
 * CHANGE:    Initial database setup with schema application
 * RISK:      Medium - Database operations affecting data persistence
 */
func initializeDatabase(dbPath string) error {
	// Create database directory
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// For single binary, we'll use a simple file-based approach initially
	// TODO: Implement KuzuDB integration for full graph database functionality
	
	// Create a marker file to indicate database is initialized
	markerFile := filepath.Join(dbDir, ".initialized")
	if _, err := os.Stat(markerFile); err == nil {
		return nil // Already initialized
	}

	// Read embedded schema for reference
	schemaData, err := assets.ReadFile("assets/schema.cypher")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	// Write schema to database directory for reference
	schemaFile := filepath.Join(dbDir, "schema.cypher")
	if err := os.WriteFile(schemaFile, schemaData, 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	// Create initialization marker
	if err := os.WriteFile(markerFile, []byte(fmt.Sprintf("initialized_%d", time.Now().Unix())), 0644); err != nil {
		return fmt.Errorf("failed to create initialization marker: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Activity event creation from current environment
 * INPUT:     Working directory, environment variables, and user context
 * OUTPUT:    ActivityEvent entity with project detection and metadata
 * BUSINESS:  Activity events provide the primary data input for work tracking
 * CHANGE:    Initial environment-based event creation with project detection
 * RISK:      Medium - Environment detection affects tracking accuracy
 */
func createActivityEventFromEnvironment() (*ActivityEvent, error) {
	// Get user ID
	userID := getUserID()
	if userID == "" {
		return nil, fmt.Errorf("unable to determine user ID")
	}

	// Get working directory
	workingDir, err := os.Getwd()
	if err != nil {
		if fallbackDir := os.Getenv("PWD"); fallbackDir != "" {
			workingDir = fallbackDir
		} else {
			return nil, fmt.Errorf("unable to determine working directory: %w", err)
		}
	}

	// Detect project information
	projectInfo := detectProjectInfo(workingDir)

	// Create metadata
	metadata := make(map[string]string)
	metadata["working_dir"] = workingDir
	metadata["project_type"] = string(projectInfo.Type)
	metadata["git_branch"] = projectInfo.GitBranch
	metadata["claude_code_version"] = os.Getenv("CLAUDE_CODE_VERSION")
	metadata["platform"] = runtime.GOOS
	metadata["architecture"] = runtime.GOARCH
	
	if hostname, err := os.Hostname(); err == nil {
		metadata["hostname"] = hostname
	}

	// Create activity event
	return NewActivityEvent(ActivityEventConfig{
		UserID:         userID,
		ProjectPath:    projectInfo.Path,
		ProjectName:    projectInfo.Name,
		ActivityType:   string(ActivityTypeCommand),
		ActivitySource: string(ActivitySourceHook),
		Timestamp:      time.Now(),
		Command:        "claude-action",
		Description:    "Claude Code activity",
		Metadata:       metadata,
	}), nil
}

func getUserID() string {
	userIDSources := []string{"USER", "USERNAME", "LOGNAME"}
	for _, envVar := range userIDSources {
		if userID := os.Getenv(envVar); userID != "" {
			return userID
		}
	}
	return "unknown-user"
}

/**
 * CONTEXT:   Project information detection with multiple strategies
 * INPUT:     Working directory path for analysis
 * OUTPUT:    ProjectInfo with name, path, type, and git information
 * BUSINESS:  Project detection enables work organization and tracking
 * CHANGE:    Initial project detection with git and package file analysis
 * RISK:      Low - Multiple fallback strategies ensure project detection
 */
type ProjectInfo struct {
	Name      string
	Path      string
	Type      ProjectType
	GitBranch string
}

func detectProjectInfo(workingDir string) ProjectInfo {
	info := ProjectInfo{
		Path: workingDir,
		Type: ProjectTypeGeneral,
	}

	// Find project root
	projectRoot := findProjectRoot(workingDir)
	if projectRoot != "" {
		info.Path = projectRoot
		info.Name = filepath.Base(projectRoot)
	} else {
		info.Name = filepath.Base(workingDir)
	}

	// Avoid generic names
	if isGenericDirName(info.Name) {
		parent := filepath.Dir(info.Path)
		parentName := filepath.Base(parent)
		if !isGenericDirName(parentName) {
			info.Name = parentName
			info.Path = parent
		}
	}

	// Detect project type
	info.Type = detectProjectTypeFromPath(info.Path)

	// Get git branch
	info.GitBranch = getGitBranch(info.Path)

	return info
}

func findProjectRoot(startDir string) string {
	current := startDir
	for {
		// Check for git repository
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}

		// Check for project files
		projectFiles := []string{
			"package.json", "go.mod", "Cargo.toml", "pyproject.toml",
			"setup.py", "requirements.txt", "pom.xml", "build.gradle",
			"Makefile", "CMakeLists.txt",
		}

		for _, file := range projectFiles {
			if _, err := os.Stat(filepath.Join(current, file)); err == nil {
				return current
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return ""
}

func detectProjectTypeFromPath(projectPath string) ProjectType {
	typeIndicators := map[string]ProjectType{
		"go.mod":           ProjectTypeGo,
		"Cargo.toml":       ProjectTypeRust,
		"package.json":     ProjectTypeJavaScript,
		"requirements.txt": ProjectTypePython,
		"setup.py":         ProjectTypePython,
		"pyproject.toml":   ProjectTypePython,
	}

	for filename, projectType := range typeIndicators {
		if _, err := os.Stat(filepath.Join(projectPath, filename)); err == nil {
			if projectType == ProjectTypeJavaScript {
				if _, err := os.Stat(filepath.Join(projectPath, "tsconfig.json")); err == nil {
					return ProjectTypeTypeScript
				}
			}
			return projectType
		}
	}

	// Check for web projects
	webFiles := []string{"index.html", "webpack.config.js", "vite.config.js", "next.config.js"}
	for _, file := range webFiles {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			return ProjectTypeWeb
		}
	}

	return ProjectTypeGeneral
}

func getGitBranch(projectPath string) string {
	headFile := filepath.Join(projectPath, ".git", "HEAD")
	content, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}

	headContent := strings.TrimSpace(string(content))
	if strings.HasPrefix(headContent, "ref: refs/heads/") {
		return strings.TrimPrefix(headContent, "ref: refs/heads/")
	}

	return ""
}

func isGenericDirName(name string) bool {
	generic := []string{
		"src", "lib", "app", "code", "work", "projects", "dev", "tmp", "test", "tests",
		"client", "server", "frontend", "backend", "api", "web", "scripts", "tools",
		"bin", "build", "dist", "out", "target", "node_modules", ".git", "vendor",
	}
	
	lowerName := strings.ToLower(name)
	for _, g := range generic {
		if lowerName == g {
			return true
		}
	}
	return false
}

/**
 * CONTEXT:   Fallback logging when daemon is unavailable
 * INPUT:     ActivityEvent that couldn't be sent to daemon
 * OUTPUT:    Event written to local log file for later processing
 * BUSINESS:  Fallback ensures no work activity is lost when daemon is down
 * CHANGE:    Initial fallback implementation with atomic file operations
 * RISK:      Low - Simple file append operations with error handling
 */
func writeActivityToFallbackLog(event *ActivityEvent) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	logDir := filepath.Join(homeDir, ".claude-monitor", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile := filepath.Join(logDir, "fallback-activities.log")
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	logEntry := FallbackLogEntry{
		Timestamp: time.Now().UTC(),
		Event:     *event,
		Source:    "hook_fallback",
		Version:   Version,
	}

	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(file, "%s\n", jsonData)
	return err
}

/**
 * CONTEXT:   System service installation for automatic daemon startup
 * INPUT:     Binary path and operating system service management
 * OUTPUT:    Installed system service with auto-start configuration
 * BUSINESS:  Service installation provides hands-free daemon management
 * CHANGE:    Initial service installation with cross-platform support
 * RISK:      High - System service modification affecting system startup
 */
func installSystemService(binaryPath string) error {
	switch runtime.GOOS {
	case "linux":
		return installSystemdService(binaryPath)
	case "darwin":
		return installLaunchdService(binaryPath)
	case "windows":
		return installWindowsService(binaryPath)
	default:
		return fmt.Errorf("system service not supported on %s", runtime.GOOS)
	}
}

func installSystemdService(binaryPath string) error {
	// Get the real user (not root when using sudo)
	user := os.Getenv("USER")
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		user = sudoUser
	}
	
	userHome := os.Getenv("HOME")
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		userHome = "/home/" + sudoUser
	}
	
	serviceContent := fmt.Sprintf(`[Unit]
Description=Claude Monitor - Work Hour Tracking Daemon
Documentation=https://github.com/claude-monitor/system
After=network.target
Wants=network.target

[Service]
Type=simple
User=%s
Group=%s
WorkingDirectory=%s/.claude/monitor
ExecStart=%s daemon --config %s/.claude/monitor/config.json
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=claude-monitor

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=%s/.claude
ProtectHome=tmpfs
BindReadOnlyPaths=%s

# Resource limits
LimitNOFILE=8192
MemoryMax=512M
TasksMax=256

[Install]
WantedBy=multi-user.target
`, user, user, userHome, binaryPath, userHome, userHome, userHome)

	servicePath := "/etc/systemd/system/claude-monitor.service"
	
	// Check if running as root (required for system service installation)
	if os.Geteuid() != 0 {
		return fmt.Errorf("system service installation requires root privileges (use sudo)")
	}
	
	// Write service file
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}
	
	// Reload systemd and enable service
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	
	if err := runCommand("systemctl", "enable", "claude-monitor.service"); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	
	// Start the service
	if err := runCommand("systemctl", "start", "claude-monitor.service"); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	
	return nil
}

// runCommand executes a system command
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command '%s %s' failed: %v\nOutput: %s", name, strings.Join(args, " "), err, string(output))
	}
	return nil
}

func installLaunchdService(binaryPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.claude-monitor.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
`, binaryPath)

	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.claude-monitor.daemon.plist")
	return os.WriteFile(plistPath, []byte(plistContent), 0644)
}

func installWindowsService(binaryPath string) error {
	// Windows service installation would require additional dependencies
	// For now, return unsupported
	return fmt.Errorf("Windows service installation not implemented in self-contained binary")
}

// Data structures for HTTP communication and reporting
type ActivityEventRequest struct {
	UserID         string            `json:"user_id"`
	ProjectPath    string            `json:"project_path"`
	ProjectName    string            `json:"project_name"`
	ActivityType   string            `json:"activity_type"`
	ActivitySource string            `json:"activity_source"`
	Timestamp      time.Time         `json:"timestamp"`
	Command        string            `json:"command"`
	Description    string            `json:"description"`
	Metadata       map[string]string `json:"metadata"`
}

type FallbackLogEntry struct {
	Timestamp time.Time     `json:"timestamp"`
	Event     ActivityEvent `json:"event"`
	Source    string        `json:"source"`
	Version   string        `json:"version"`
}


type PendingSessionSummary struct {
	ID               string    `json:"id"`
	StartTime        time.Time `json:"start_time"`
	ProjectName      string    `json:"project_name"`
	ProjectPath      string    `json:"project_path"`
	ActiveWorkBlocks int       `json:"active_work_blocks"`
	LastActivity     time.Time `json:"last_activity"`
	UserID           string    `json:"user_id"`
}

type CloseSessionsResult struct {
	ClosedSessions    int           `json:"closed_sessions"`
	ClosedWorkBlocks  int           `json:"closed_work_blocks"`
	TotalWorkTime     time.Duration `json:"total_work_time"`
	Errors            int           `json:"errors"`
	ClosedAt          time.Time     `json:"closed_at"`
}