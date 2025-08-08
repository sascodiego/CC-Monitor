/**
 * CONTEXT:   Service monitor interface for monitoring and diagnostics only
 * INPUT:     Monitoring requests and log retrieval parameters
 * OUTPUT:    Service status, logs, health checks, and uptime information
 * BUSINESS:  Focused interface for service monitoring concerns only
 * CHANGE:    Interface Segregation Principle - extracted from fat ServiceManager interface
 * RISK:      Low - Interface segregation improving maintainability and testability
 */

package interfaces

import "time"

/**
 * CONTEXT:   Service monitor interface following Interface Segregation Principle
 * INPUT:     Monitoring and diagnostic requests without installation or runtime dependencies
 * OUTPUT:    Service status, logs, health information, and uptime metrics
 * BUSINESS:  Clients needing only monitoring depend on minimal interface
 * CHANGE:    Initial ISP-compliant interface extracted from ServiceManager
 * RISK:      Low - Interface segregation reduces coupling and improves testability
 */
type ServiceMonitor interface {
	// Get comprehensive service status information
	Status() (ServiceStatus, error)
	
	// Retrieve service log entries
	GetLogs(lines int) ([]LogEntry, error)
	
	// Perform health check on service
	HealthCheck() error
	
	// Get service uptime duration
	GetUptime() (time.Duration, error)
}

/**
 * CONTEXT:   Service status information for monitoring and troubleshooting
 * INPUT:     Platform service manager status queries and system metrics
 * OUTPUT:    Unified service status with performance and diagnostic information
 * BUSINESS:  Service status enables monitoring, alerting, and problem diagnosis
 * CHANGE:    Shared status structure used by monitor interface
 * RISK:      Low - Status structure providing read-only diagnostic information
 */
type ServiceStatus struct {
	Name        string        `json:"name"`
	DisplayName string        `json:"display_name"`
	State       ServiceState  `json:"state"`
	PID         int           `json:"pid,omitempty"`
	Uptime      time.Duration `json:"uptime,omitempty"`
	StartTime   time.Time     `json:"start_time,omitempty"`
	Memory      int64         `json:"memory_usage,omitempty"`
	CPU         float64       `json:"cpu_usage,omitempty"`
	LastError   string        `json:"last_error,omitempty"`
}

type ServiceState string

const (
	ServiceStateUnknown  ServiceState = "unknown"
	ServiceStateRunning  ServiceState = "running"
	ServiceStateStopped  ServiceState = "stopped"
	ServiceStateStarting ServiceState = "starting"
	ServiceStateStopping ServiceState = "stopping"
	ServiceStateFailed   ServiceState = "failed"
)

/**
 * CONTEXT:   Log entry structure for service log retrieval
 * INPUT:     Service logging system entries with timestamps and metadata
 * OUTPUT:    Structured log entry with timestamp, level, message, and source
 * BUSINESS:  Log entries enable troubleshooting and service behavior analysis
 * CHANGE:    Shared log entry structure used by monitor interface
 * RISK:      Low - Log structure providing read-only diagnostic information
 */
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}