/**
 * CONTEXT:   Claude Code hook integration for accurate Claude processing time tracking
 * INPUT:     Hook configuration and activity detection for pre/post action events
 * OUTPUT:    Integration utilities and configuration for enhanced time tracking
 * BUSINESS:  Prevent false idle detection during Claude processing by using dual hooks
 * CHANGE:    Initial implementation of enhanced hook integration with processing awareness
 * RISK:      Medium - Hook integration affects all activity detection, requires proper configuration
 */

package entities

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// HookType represents different types of Claude Code hooks
type HookType string

const (
	HookTypePreAction  HookType = "pre_action"  // Before Claude starts processing
	HookTypePostAction HookType = "post_action" // After Claude finishes processing
)

// HookConfig represents Claude Code hook configuration
type HookConfig struct {
	PreActionCommand  string `json:"pre_action"`
	PostActionCommand string `json:"post_action"`
	Enabled           bool   `json:"enabled"`
}

// ClaudeCodeIntegration provides utilities for enhanced Claude Code hook integration
type ClaudeCodeIntegration struct {
	monitorEndpoint string
	timeout         time.Duration
}

/**
 * CONTEXT:   Factory for creating Claude Code integration with monitor endpoint
 * INPUT:     Monitor daemon endpoint URL for activity reporting
 * OUTPUT:    Configured ClaudeCodeIntegration instance
 * BUSINESS:  Centralize hook integration configuration and utilities
 * CHANGE:    Initial integration utility factory
 * RISK:      Low - Utility factory for hook integration management
 */
func NewClaudeCodeIntegration(monitorEndpoint string) *ClaudeCodeIntegration {
	if monitorEndpoint == "" {
		monitorEndpoint = "http://localhost:9193" // Default local daemon
	}
	
	return &ClaudeCodeIntegration{
		monitorEndpoint: monitorEndpoint,
		timeout:         10 * time.Second, // HTTP timeout
	}
}

/**
 * CONTEXT:   Generate Claude Code hook configuration for enhanced processing time tracking
 * INPUT:     Monitor endpoint and optional hook customization parameters
 * OUTPUT:    Complete hook configuration JSON for Claude Code integration
 * BUSINESS:  Provide ready-to-use hook configuration that supports processing time tracking
 * CHANGE:    Initial hook configuration generation with dual hook support
 * RISK:      Low - Configuration generation utility with no side effects
 */
func (cci *ClaudeCodeIntegration) GenerateHookConfig() HookConfig {
	// Generate pre-action hook command
	preActionCommand := fmt.Sprintf(
		`/usr/local/bin/claude-monitor hook --type=start --endpoint=%s --timeout=%ds`,
		cci.monitorEndpoint,
		int(cci.timeout.Seconds()),
	)
	
	// Generate post-action hook command  
	postActionCommand := fmt.Sprintf(
		`/usr/local/bin/claude-monitor hook --type=end --endpoint=%s --timeout=%ds --tokens="${RESPONSE_TOKENS}" --processing-time="${PROCESSING_TIME}"`,
		cci.monitorEndpoint,
		int(cci.timeout.Seconds()),
	)
	
	return HookConfig{
		PreActionCommand:  preActionCommand,
		PostActionCommand: postActionCommand,
		Enabled:           true,
	}
}

/**
 * CONTEXT:   Generate Claude Code configuration JSON with enhanced hooks
 * INPUT:     Optional base configuration to extend with enhanced hooks
 * OUTPUT:    Complete Claude Code configuration JSON string
 * BUSINESS:  Provide complete configuration for Claude Code with enhanced time tracking
 * CHANGE:    Initial configuration generation with hook integration
 * RISK:      Low - Configuration generation utility for user setup
 */
func (cci *ClaudeCodeIntegration) GenerateClaudeCodeConfigJSON(baseConfig map[string]interface{}) (string, error) {
	// Create base configuration if not provided
	if baseConfig == nil {
		baseConfig = make(map[string]interface{})
	}
	
	// Generate hook configuration
	hookConfig := cci.GenerateHookConfig()
	
	// Add hooks to configuration
	baseConfig["hooks"] = map[string]string{
		"pre_action":  hookConfig.PreActionCommand,
		"post_action": hookConfig.PostActionCommand,
	}
	
	// Ensure other important settings
	if _, exists := baseConfig["activity_tracking"]; !exists {
		baseConfig["activity_tracking"] = map[string]interface{}{
			"enabled":           true,
			"detailed_logging":  true,
			"processing_aware":  true, // Enhanced: Processing-aware tracking
		}
	}
	
	// Convert to JSON
	configJSON, err := json.MarshalIndent(baseConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	return string(configJSON), nil
}

/**
 * CONTEXT:   Create activity event for pre-action hook with Claude processing context
 * INPUT:     Prompt content and hook execution context
 * OUTPUT:    ActivityEvent configured for Claude processing start
 * BUSINESS:  Generate proper activity event for Claude processing start detection
 * CHANGE:    Initial pre-action hook event creation with processing context
 * RISK:      Medium - Event creation affects processing state detection
 */
func (cci *ClaudeCodeIntegration) CreatePreActionEvent(prompt string, metadata map[string]string) (*ActivityEvent, error) {
	// Create processing time estimator for duration estimation
	estimator := NewProcessingTimeEstimator()
	
	// Generate processing context
	promptID := fmt.Sprintf("prompt_%d", time.Now().Unix())
	processingContext := estimator.CreateProcessingContext(prompt, promptID)
	processingContext.ClaudeActivity = ClaudeActivityStart
	
	// Create activity event
	return NewActivityEvent(ActivityEventConfig{
		UserID:         getUserID(),
		ProjectPath:    getCurrentProjectPath(),
		ActivityType:   ActivityTypeGeneration,
		ActivitySource: ActivitySourceHook,
		Timestamp:      time.Now(),
		Command:        extractCommandFromPrompt(prompt),
		Description:    fmt.Sprintf("Claude processing started: %s", truncatePrompt(prompt, 100)),
		Metadata:       metadata,
		ClaudeContext:  processingContext,
	})
}

/**
 * CONTEXT:   Create activity event for post-action hook with processing completion
 * INPUT:     Response metrics and processing completion context
 * OUTPUT:    ActivityEvent configured for Claude processing end
 * BUSINESS:  Generate proper activity event for Claude processing completion detection
 * CHANGE:    Initial post-action hook event creation with completion tracking
 * RISK:      Medium - Event creation affects processing time calculation accuracy
 */
func (cci *ClaudeCodeIntegration) CreatePostActionEvent(responseTokens int, processingTimeSeconds int, metadata map[string]string) (*ActivityEvent, error) {
	// Create processing context for completion
	processingTime := time.Duration(processingTimeSeconds) * time.Second
	processingContext := &ClaudeProcessingContext{
		PromptID:       metadata["prompt_id"], // Should be passed from pre-action
		ActualTime:     &processingTime,
		TokensCount:    &responseTokens,
		ClaudeActivity: ClaudeActivityEnd,
	}
	
	// Enhanced metadata with response metrics
	enhancedMetadata := make(map[string]string)
	for k, v := range metadata {
		enhancedMetadata[k] = v
	}
	enhancedMetadata["response_tokens"] = fmt.Sprintf("%d", responseTokens)
	enhancedMetadata["processing_seconds"] = fmt.Sprintf("%d", processingTimeSeconds)
	
	// Create activity event
	return NewActivityEvent(ActivityEventConfig{
		UserID:         getUserID(),
		ProjectPath:    getCurrentProjectPath(),
		ActivityType:   ActivityTypeGeneration,
		ActivitySource: ActivitySourceHook,
		Timestamp:      time.Now(),
		Command:        "claude_processing_complete",
		Description:    fmt.Sprintf("Claude processing completed: %d tokens, %ds", responseTokens, processingTimeSeconds),
		Metadata:       enhancedMetadata,
		ClaudeContext:  processingContext,
	})
}

/**
 * CONTEXT:   Generate installation instructions for enhanced Claude Code integration
 * INPUT:     Target system and installation preferences
 * OUTPUT:    Complete installation and configuration instructions
 * BUSINESS:  Provide clear setup instructions for users to enable enhanced time tracking
 * CHANGE:    Initial installation guide generation
 * RISK:      Low - Documentation generation utility
 */
func (cci *ClaudeCodeIntegration) GenerateInstallationInstructions(targetSystem string) string {
	hookConfig := cci.GenerateHookConfig()
	configJSON, _ := cci.GenerateClaudeCodeConfigJSON(nil)
	
	instructions := fmt.Sprintf(`
# Enhanced Claude Code Integration Setup

## Overview
This setup enables accurate Claude processing time tracking by using both pre and post action hooks.

**Benefits:**
- ✅ No false idle detection during Claude processing
- ✅ Separate tracking of user time vs Claude processing time  
- ✅ Intelligent processing time estimation
- ✅ Accurate productivity metrics

## Installation Steps

### 1. Install Claude Monitor
%s

### 2. Configure Claude Code Hooks

Create or update your Claude Code configuration file:

**Location:** ~/.claude/config.json

**Configuration:**
```json
%s
```

### 3. Test Integration

Run these commands to test the integration:

```bash
# Test pre-action hook
%s

# Test post-action hook  
%s
```

### 4. Verify Setup

Check that both hooks are working:

```bash
# Check daemon is running
claude-monitor status

# Check recent activity
claude-monitor show --recent

# Verify processing time tracking
claude-monitor show --details
```

## Advanced Configuration

### Custom Processing Time Estimation

You can customize processing time estimation by setting environment variables:

```bash
export CLAUDE_MONITOR_BASE_PROCESSING_TIME="30s"
export CLAUDE_MONITOR_COMPLEXITY_MULTIPLIER="1.5"
export CLAUDE_MONITOR_MAX_PROCESSING_TIME="10m"
```

### Progress Updates (Optional)

For very long Claude operations, you can enable progress updates:

```json
{
  "hooks": {
    "pre_action": "%s",
    "post_action": "%s", 
    "progress_action": "/usr/local/bin/claude-monitor hook --type=progress --endpoint=%s"
  },
  "progress_interval": "30s"
}
```

## Troubleshooting

### Hook Not Executing
- Check Claude Code configuration path
- Verify claude-monitor binary is in PATH
- Check daemon is running: claude-monitor status

### Incorrect Processing Time
- Verify post-action hook receives PROCESSING_TIME variable
- Check daemon logs: claude-monitor logs
- Test estimation accuracy: claude-monitor stats --estimator

### False Idle Detection Still Occurring
- Ensure both pre and post hooks are configured
- Check that processing state is being set: claude-monitor show --processing
- Verify hook execution order and timing

## Support

For additional help:
- Check documentation: claude-monitor help
- View system status: claude-monitor diagnose
- Enable debug logging: claude-monitor --debug

`,
		generateSystemSpecificInstallation(targetSystem),
		configJSON,
		hookConfig.PreActionCommand,
		hookConfig.PostActionCommand,
		hookConfig.PreActionCommand,
		hookConfig.PostActionCommand,
		cci.monitorEndpoint,
	)
	
	return instructions
}

/**
 * CONTEXT:   Utility functions for hook integration and event creation
 * INPUT:     Various context and environment information
 * OUTPUT:    Utility values for activity event creation
 * BUSINESS:  Support hook integration with proper context detection
 * CHANGE:    Initial utility functions for hook support
 * RISK:      Low - Utility functions with environment detection
 */

func getUserID() string {
	userID := os.Getenv("USER")
	if userID == "" {
		userID = os.Getenv("USERNAME")
	}
	if userID == "" {
		userID = "unknown-user"
	}
	return userID
}

func getCurrentProjectPath() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return workingDir
}

func extractCommandFromPrompt(prompt string) string {
	// Extract likely command from prompt (simplified heuristic)
	prompt = strings.TrimSpace(prompt)
	if len(prompt) > 50 {
		// Look for command patterns
		lines := strings.Split(prompt, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "/") || 
			   strings.Contains(line, "claude") ||
			   strings.Contains(line, "help") ||
			   strings.Contains(line, "write") ||
			   strings.Contains(line, "create") {
				return line
			}
		}
		// Fallback to first line
		if len(lines) > 0 {
			return lines[0]
		}
	}
	return prompt
}

func truncatePrompt(prompt string, maxLength int) string {
	if len(prompt) <= maxLength {
		return prompt
	}
	return prompt[:maxLength-3] + "..."
}

func generateSystemSpecificInstallation(targetSystem string) string {
	switch strings.ToLower(targetSystem) {
	case "linux", "ubuntu", "debian":
		return `# Install Claude Monitor (Linux)
curl -fsSL https://raw.githubusercontent.com/claude-monitor/install/main/install.sh | bash

# Or manual installation:
wget https://github.com/claude-monitor/releases/latest/download/claude-monitor-linux-amd64.tar.gz
tar -xzf claude-monitor-linux-amd64.tar.gz
sudo mv claude-monitor /usr/local/bin/
sudo chmod +x /usr/local/bin/claude-monitor

# Start daemon
claude-monitor daemon --start`

	case "macos", "darwin":
		return `# Install Claude Monitor (macOS)
curl -fsSL https://raw.githubusercontent.com/claude-monitor/install/main/install.sh | bash

# Or with Homebrew:
brew tap claude-monitor/tap
brew install claude-monitor

# Start daemon
claude-monitor daemon --start`

	case "windows":
		return `# Install Claude Monitor (Windows)
# Download from: https://github.com/claude-monitor/releases/latest
# Extract claude-monitor.exe to C:\Program Files\Claude Monitor\

# Add to PATH and start daemon
claude-monitor.exe daemon --start

# Or use PowerShell installer:
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/claude-monitor/install/main/install.ps1" | Invoke-Expression`

	default:
		return `# Install Claude Monitor
# Visit: https://github.com/claude-monitor/releases/latest
# Download appropriate binary for your system
# Extract and add to PATH
# Run: claude-monitor daemon --start`
	}
}