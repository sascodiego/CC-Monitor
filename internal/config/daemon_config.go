/**
 * CONTEXT:   Daemon configuration management for Claude Monitor HTTP server
 * INPUT:     Configuration files, environment variables, and default settings
 * OUTPUT:    Validated daemon configuration with all operational parameters
 * BUSINESS:  Centralized configuration management for daemon startup and operation
 * CHANGE:    Initial configuration implementation with validation and defaults
 * RISK:      Low - Configuration management with comprehensive validation and defaults
 */

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

/**
 * CONTEXT:   Main daemon configuration structure with all operational parameters
 * INPUT:     Configuration values from files, environment, and defaults
 * OUTPUT:    Complete daemon configuration ready for service initialization
 * BUSINESS:  Configure all aspects of daemon operation including network, database, and timing
 * CHANGE:    Initial configuration structure with comprehensive daemon settings
 * RISK:      Low - Configuration data structure with validation methods
 */
type DaemonConfig struct {
	// Server configuration
	Server ServerConfig `json:"server"`
	
	// Database configuration
	Database DatabaseConfig `json:"database"`
	
	// Logging configuration
	Logging LoggingConfig `json:"logging"`
	
	// Work tracking configuration
	WorkTracking WorkTrackingConfig `json:"work_tracking"`
	
	// Performance configuration
	Performance PerformanceConfig `json:"performance"`
	
	// Health and monitoring
	Health HealthConfig `json:"health"`
}

type ServerConfig struct {
	ListenAddr      string        `json:"listen_addr"`
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	TLSEnabled      bool          `json:"tls_enabled"`
	TLSCertFile     string        `json:"tls_cert_file"`
	TLSKeyFile      string        `json:"tls_key_file"`
}

type DatabaseConfig struct {
	Path                string        `json:"path"`
	ConnectionTimeout   time.Duration `json:"connection_timeout"`
	QueryTimeout        time.Duration `json:"query_timeout"`
	BackupEnabled       bool          `json:"backup_enabled"`
	BackupInterval      time.Duration `json:"backup_interval"`
	BackupPath          string        `json:"backup_path"`
	MaxConnections      int           `json:"max_connections"`
	MaxIdleConnections  int           `json:"max_idle_connections"`
	ConnectTimeout      time.Duration `json:"connect_timeout"`
}

type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	OutputFile string `json:"output_file"`
	MaxSizeMB  int    `json:"max_size_mb"`
	MaxBackups int    `json:"max_backups"`
	MaxAgeDays int    `json:"max_age_days"`
}

type WorkTrackingConfig struct {
	SessionDuration time.Duration `json:"session_duration"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	AutoFinalize    bool          `json:"auto_finalize"`
}

type PerformanceConfig struct {
	MaxConcurrentRequests int           `json:"max_concurrent_requests"`
	RequestQueueSize      int           `json:"request_queue_size"`
	WorkerPoolSize        int           `json:"worker_pool_size"`
	CacheSize             int           `json:"cache_size"`
	CacheTTL              time.Duration `json:"cache_ttl"`
	RateLimitRPS          int           `json:"rate_limit_rps"`
}

type HealthConfig struct {
	EnableHealthCheck bool          `json:"enable_health_check"`
	HealthCheckPath   string        `json:"health_check_path"`
	ReadyCheckPath    string        `json:"ready_check_path"`
	MetricsPath       string        `json:"metrics_path"`
	CheckInterval     time.Duration `json:"check_interval"`
}

/**
 * CONTEXT:   Default configuration values for Claude Monitor daemon
 * INPUT:     No parameters, provides sensible defaults for all configuration options
 * OUTPUT:    DaemonConfig instance with production-ready default values
 * BUSINESS:  Enable zero-configuration startup while allowing customization
 * CHANGE:    Initial default configuration with Claude Monitor specific values
 * RISK:      Low - Default values based on Claude Monitor requirements
 */
func NewDefaultConfig() *DaemonConfig {
	return &DaemonConfig{
		Server: ServerConfig{
			ListenAddr:      DefaultListenAddr,
			Host:            "localhost",
			Port:            9193,
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			TLSEnabled:      false,
		},
		Database: DatabaseConfig{
			Path:               "./data/claude_monitor.db",
			ConnectionTimeout:  10 * time.Second,
			QueryTimeout:       30 * time.Second,
			BackupEnabled:      true,
			BackupInterval:     24 * time.Hour,
			BackupPath:         "./data/backups",
			MaxConnections:     25,
			MaxIdleConnections: 5,
			ConnectTimeout:     10 * time.Second,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputFile: "",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
		},
		WorkTracking: WorkTrackingConfig{
			SessionDuration: 5 * time.Hour,  // Claude session duration
			IdleTimeout:     5 * time.Minute, // Work block idle timeout
			CleanupInterval: 2 * time.Minute, // Cleanup frequency
			AutoFinalize:    true,
		},
		Performance: PerformanceConfig{
			MaxConcurrentRequests: 1000,
			RequestQueueSize:      10000,
			WorkerPoolSize:        10,
			CacheSize:             1000,
			CacheTTL:              10 * time.Minute,
			RateLimitRPS:          1000,
		},
		Health: HealthConfig{
			EnableHealthCheck: true,
			HealthCheckPath:   "/health",
			ReadyCheckPath:    "/ready",
			MetricsPath:       "/metrics",
			CheckInterval:     30 * time.Second,
		},
	}
}

/**
 * CONTEXT:   Load daemon configuration from file with fallback to defaults
 * INPUT:     Configuration file path (JSON format)
 * OUTPUT:    Loaded and validated daemon configuration or error
 * BUSINESS:  Allow file-based configuration while maintaining defaults
 * CHANGE:    Initial configuration loading with JSON support
 * RISK:      Medium - File I/O and JSON parsing with validation
 */
func LoadDaemonConfig(configPath string) (*DaemonConfig, error) {
	// Start with defaults
	config := NewDefaultConfig()
	
	// If no config file specified, return defaults
	if configPath == "" {
		return config, nil
	}
	
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// File doesn't exist, return defaults with warning
		fmt.Printf("Warning: Configuration file %s not found, using defaults\n", configPath)
		return config, nil
	}
	
	// Read configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	
	// Parse JSON configuration
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}
	
	// Validate configuration
	err = config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	fmt.Printf("Loaded configuration from %s\n", configPath)
	return config, nil
}

/**
 * CONTEXT:   Create daemon configuration from environment variables
 * INPUT:     Environment variables with CLAUDE_MONITOR_ prefix
 * OUTPUT:    Daemon configuration with environment overrides applied
 * BUSINESS:  Support container and deployment environments with env var configuration
 * CHANGE:    Initial environment variable configuration support
 * RISK:      Medium - Environment variable parsing with type conversion
 */
func LoadFromEnvironment() *DaemonConfig {
	config := NewDefaultConfig()
	
	// Server configuration from environment
	if addr := os.Getenv("CLAUDE_MONITOR_LISTEN_ADDR"); addr != "" {
		config.Server.ListenAddr = addr
	}
	
	if dbPath := os.Getenv("CLAUDE_MONITOR_DB_PATH"); dbPath != "" {
		config.Database.Path = dbPath
	}
	
	if logLevel := os.Getenv("CLAUDE_MONITOR_LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}
	
	if logFormat := os.Getenv("CLAUDE_MONITOR_LOG_FORMAT"); logFormat != "" {
		config.Logging.Format = logFormat
	}
	
	if logFile := os.Getenv("CLAUDE_MONITOR_LOG_FILE"); logFile != "" {
		config.Logging.OutputFile = logFile
	}
	
	// Parse duration environment variables
	if sessionDuration := os.Getenv("CLAUDE_MONITOR_SESSION_DURATION"); sessionDuration != "" {
		if dur, err := time.ParseDuration(sessionDuration); err == nil {
			config.WorkTracking.SessionDuration = dur
		}
	}
	
	if idleTimeout := os.Getenv("CLAUDE_MONITOR_IDLE_TIMEOUT"); idleTimeout != "" {
		if dur, err := time.ParseDuration(idleTimeout); err == nil {
			config.WorkTracking.IdleTimeout = dur
		}
	}
	
	return config
}

/**
 * CONTEXT:   Validate daemon configuration for consistency and operational requirements
 * INPUT:     No parameters, validates internal configuration state
 * OUTPUT:    Error if configuration invalid, nil if valid
 * BUSINESS:  Ensure daemon can start successfully with provided configuration
 * CHANGE:    Initial validation implementation with comprehensive checks
 * RISK:      Low - Validation only operation with no side effects
 */
func (dc *DaemonConfig) Validate() error {
	// Validate server configuration
	if dc.Server.Port <= 0 || dc.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", dc.Server.Port)
	}
	
	if dc.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server read timeout must be positive, got %v", dc.Server.ReadTimeout)
	}
	
	if dc.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server write timeout must be positive, got %v", dc.Server.WriteTimeout)
	}
	
	if dc.Server.ShutdownTimeout <= 0 {
		return fmt.Errorf("server shutdown timeout must be positive, got %v", dc.Server.ShutdownTimeout)
	}
	
	// Validate TLS configuration
	if dc.Server.TLSEnabled {
		if dc.Server.TLSCertFile == "" {
			return fmt.Errorf("TLS cert file required when TLS enabled")
		}
		if dc.Server.TLSKeyFile == "" {
			return fmt.Errorf("TLS key file required when TLS enabled")
		}
	}
	
	// Validate database configuration
	if dc.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}
	
	// Ensure database directory exists
	dbDir := filepath.Dir(dc.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory %s: %w", dbDir, err)
	}
	
	if dc.Database.ConnectionTimeout <= 0 {
		return fmt.Errorf("database connection timeout must be positive, got %v", dc.Database.ConnectionTimeout)
	}
	
	if dc.Database.QueryTimeout <= 0 {
		return fmt.Errorf("database query timeout must be positive, got %v", dc.Database.QueryTimeout)
	}
	
	// Validate backup configuration
	if dc.Database.BackupEnabled {
		if dc.Database.BackupPath == "" {
			return fmt.Errorf("backup path required when backup enabled")
		}
		if dc.Database.BackupInterval <= 0 {
			return fmt.Errorf("backup interval must be positive when backup enabled, got %v", dc.Database.BackupInterval)
		}
		
		// Ensure backup directory exists
		if err := os.MkdirAll(dc.Database.BackupPath, 0755); err != nil {
			return fmt.Errorf("failed to create backup directory %s: %w", dc.Database.BackupPath, err)
		}
	}
	
	// Validate logging configuration
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[dc.Logging.Level] {
		return fmt.Errorf("invalid log level %s, must be one of: debug, info, warn, error", dc.Logging.Level)
	}
	
	validFormats := map[string]bool{
		"json": true, "text": true,
	}
	if !validFormats[dc.Logging.Format] {
		return fmt.Errorf("invalid log format %s, must be one of: json, text", dc.Logging.Format)
	}
	
	// Create log file directory if specified
	if dc.Logging.OutputFile != "" {
		logDir := filepath.Dir(dc.Logging.OutputFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory %s: %w", logDir, err)
		}
	}
	
	// Validate work tracking configuration
	if dc.WorkTracking.SessionDuration != 5*time.Hour {
		return fmt.Errorf("session duration must be exactly 5 hours for Claude compatibility, got %v", dc.WorkTracking.SessionDuration)
	}
	
	if dc.WorkTracking.IdleTimeout != 5*time.Minute {
		return fmt.Errorf("idle timeout must be exactly 5 minutes for work block logic, got %v", dc.WorkTracking.IdleTimeout)
	}
	
	if dc.WorkTracking.CleanupInterval <= 0 {
		return fmt.Errorf("cleanup interval must be positive, got %v", dc.WorkTracking.CleanupInterval)
	}
	
	// Validate performance configuration
	if dc.Performance.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("max concurrent requests must be positive, got %d", dc.Performance.MaxConcurrentRequests)
	}
	
	if dc.Performance.WorkerPoolSize <= 0 {
		return fmt.Errorf("worker pool size must be positive, got %d", dc.Performance.WorkerPoolSize)
	}
	
	if dc.Performance.CacheSize <= 0 {
		return fmt.Errorf("cache size must be positive, got %d", dc.Performance.CacheSize)
	}
	
	if dc.Performance.RateLimitRPS <= 0 {
		return fmt.Errorf("rate limit RPS must be positive, got %d", dc.Performance.RateLimitRPS)
	}
	
	return nil
}

/**
 * CONTEXT:   Save daemon configuration to file for persistence
 * INPUT:     File path for saving configuration in JSON format
 * OUTPUT:    Error if save fails, nil on success
 * BUSINESS:  Allow configuration persistence and sharing across environments
 * CHANGE:    Initial configuration save implementation with JSON serialization
 * RISK:      Medium - File I/O with JSON serialization
 */
func (dc *DaemonConfig) SaveToFile(configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}
	
	// Marshal configuration to JSON
	data, err := json.MarshalIndent(dc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	// Write to file
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}
	
	return nil
}

/**
 * CONTEXT:   Get database connection string from configuration
 * INPUT:     No parameters, uses internal database configuration
 * OUTPUT:    Database path ready for SQLite connection
 * BUSINESS:  Provide consistent database connection parameters
 * CHANGE:    Updated for SQLite database configuration
 * RISK:      Low - Simple path construction for database connection
 */
func (dc *DaemonConfig) GetDatabasePath() string {
	return dc.Database.Path
}

/**
 * CONTEXT:   Get server address from configuration for HTTP server binding
 * INPUT:     No parameters, uses internal server configuration
 * OUTPUT:    Server address string ready for HTTP server Listen
 * BUSINESS:  Provide consistent server binding configuration
 * CHANGE:    Initial server address configuration
 * RISK:      Low - Simple address string construction
 */
func (dc *DaemonConfig) GetServerAddr() string {
	return dc.Server.ListenAddr
}