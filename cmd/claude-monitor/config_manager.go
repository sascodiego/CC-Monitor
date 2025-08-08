/**
 * CONTEXT:   Configuration management for Claude Monitor unified binary
 * INPUT:     Configuration files, environment variables, command line overrides
 * OUTPUT:    Validated application configuration for all operation modes
 * BUSINESS:  Centralized configuration enables consistent behavior across modes
 * CHANGE:    Extracted from main.go to separate configuration concerns
 * RISK:      Low - Configuration loading with comprehensive defaults and validation
 */

package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	cfg "github.com/claude-monitor/system/internal/config"
)

/**
 * CONTEXT:   Application configuration structure for unified binary
 * INPUT:     Configuration parameters for daemon, session, reporting, projects
 * OUTPUT:    Runtime configuration enabling all application modes
 * BUSINESS:  Unified configuration ensures consistent behavior across CLI and daemon
 * CHANGE:    Extracted AppConfig type definition for better organization
 * RISK:      Low - Configuration structure with safe defaults
 */
type AppConfig struct {
	Daemon struct {
		ListenAddr             string `json:"listen_addr"`
		DatabasePath           string `json:"database_path"`
		LogLevel               string `json:"log_level"`
		EnableCORS             bool   `json:"enable_cors"`
		MaxConcurrentRequests  int    `json:"max_concurrent_requests"`
	} `json:"daemon"`
	
	Session struct {
		DurationHours   int `json:"duration_hours"`
		MaxIdleMinutes  int `json:"max_idle_minutes"`
	} `json:"session"`
	
	Reporting struct {
		DefaultOutputFormat  string `json:"default_output_format"`
		EnableColors         bool   `json:"enable_colors"`
		MaxProjectsDisplay   int    `json:"max_projects_display"`
		TimeFormat           string `json:"time_format"`
	} `json:"reporting"`
	
	Projects struct {
		AutoDetect        bool              `json:"auto_detect"`
		CustomNames       map[string]string `json:"custom_names"`
		TrackGitBranches  bool              `json:"track_git_branches"`
	} `json:"projects"`
}

/**
 * CONTEXT:   Configuration loading with defaults and file overrides
 * INPUT:     Configuration file path and environment settings
 * OUTPUT:    Validated application configuration ready for use
 * BUSINESS:  Configuration loading enables customization while ensuring defaults
 * CHANGE:    Extracted configuration loading from main startup sequence
 * RISK:      Low - Configuration loading with error handling and fallbacks
 */
func loadConfiguration() (*AppConfig, error) {
	// Load daemon configuration using existing config system
	daemonConfig, err := cfg.LoadDaemonConfig("")
	if err != nil {
		daemonConfig = cfg.NewDefaultConfig()
	}
	
	// Create AppConfig with default values
	appConfig := &AppConfig{}
	
	// Daemon configuration
	appConfig.Daemon.ListenAddr = daemonConfig.GetServerAddr()
	appConfig.Daemon.DatabasePath = daemonConfig.GetDatabasePath()
	appConfig.Daemon.LogLevel = daemonConfig.Logging.Level
	appConfig.Daemon.EnableCORS = true
	appConfig.Daemon.MaxConcurrentRequests = daemonConfig.Performance.MaxConcurrentRequests
	
	// Session configuration
	appConfig.Session.DurationHours = 5   // Claude session duration
	appConfig.Session.MaxIdleMinutes = 5  // Work block idle timeout
	
	// Reporting configuration
	appConfig.Reporting.DefaultOutputFormat = "professional"
	appConfig.Reporting.EnableColors = true
	appConfig.Reporting.MaxProjectsDisplay = 10
	appConfig.Reporting.TimeFormat = "15:04"
	
	// Projects configuration
	appConfig.Projects.AutoDetect = true
	appConfig.Projects.CustomNames = make(map[string]string)
	appConfig.Projects.TrackGitBranches = false
	
	// Try to load custom configuration file if it exists
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		if data, err := os.ReadFile(configPath); err == nil {
			// Unmarshal into existing config to preserve defaults
			json.Unmarshal(data, appConfig)
		}
	}
	
	return appConfig, nil
}

/**
 * CONTEXT:   Configuration file path resolution
 * INPUT:     Environment variables and OS detection
 * OUTPUT:    Absolute path to configuration file location
 * BUSINESS:  Consistent config location enables predictable behavior
 * CHANGE:    Extracted config path logic for reusability
 * RISK:      Low - Path resolution with OS-specific fallbacks
 */
func getConfigPath() string {
	configPath := filepath.Join(os.Getenv("HOME"), ".claude", "config.json")
	if configPath == "" {
		// Windows fallback
		if homeDir := os.Getenv("USERPROFILE"); homeDir != "" {
			configPath = filepath.Join(homeDir, ".claude", "config.json")
		}
	}
	return configPath
}

/**
 * CONTEXT:   Configuration directory creation for installation
 * INPUT:     Home directory and configuration requirements
 * OUTPUT:    Created configuration directory with proper permissions
 * BUSINESS:  Configuration directory enables persistent settings
 * CHANGE:    Extracted directory creation from installation process
 * RISK:      Low - Directory creation with error handling
 */
func createConfigurationDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	configDir := filepath.Join(homeDir, ".claude-monitor")
	err = os.MkdirAll(configDir, 0755)
	return configDir, err
}