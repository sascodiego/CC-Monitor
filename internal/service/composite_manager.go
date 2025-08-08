/**
 * CONTEXT:   Composite service manager implementing all segregated interfaces
 * INPUT:     Platform-specific service management implementations
 * OUTPUT:    Unified service manager providing all functionality through focused interfaces
 * BUSINESS:  Composite pattern enables clients to use specific interfaces while maintaining full functionality
 * CHANGE:    Initial composite manager supporting Interface Segregation Principle
 * RISK:      Medium - Composite coordination affecting all service functionality
 */

package service

import (
	"fmt"
	"runtime"
	"time"

	"github.com/claude-monitor/system/internal/service/interfaces"
)

/**
 * CONTEXT:   Composite service manager implementing all segregated interfaces
 * INPUT:     Platform detection and service management requirements
 * OUTPUT:    Service manager implementing ServiceInstaller, ServiceController, ServiceMonitor
 * BUSINESS:  Composite enables interface segregation while maintaining complete functionality
 * CHANGE:    Initial composite manager with platform-specific delegation
 * RISK:      Medium - Composite pattern coordination affecting service operations
 */
type CompositeServiceManager struct {
	platformManager PlatformServiceManager
}

/**
 * CONTEXT:   Platform-specific service manager interface for internal delegation
 * INPUT:     Platform service operations and system-specific requirements
 * OUTPUT:    Complete service management functionality for specific platform
 * BUSINESS:  Platform abstraction enables cross-platform service support
 * CHANGE:    Initial platform abstraction for composite delegation
 * RISK:      Medium - Platform abstraction affecting service reliability
 */
type PlatformServiceManager interface {
	// Installation operations
	Install(config interfaces.ServiceConfig) error
	Uninstall() error
	IsInstalled() bool
	
	// Runtime operations
	Start() error
	Stop() error
	Restart() error
	IsRunning() bool
	
	// Monitoring operations
	Status() (interfaces.ServiceStatus, error)
	GetLogs(lines int) ([]interfaces.LogEntry, error)
	HealthCheck() error
	GetUptime() (time.Duration, error)
}

/**
 * CONTEXT:   Factory function for creating platform-appropriate composite manager
 * INPUT:     Runtime platform detection and service configuration
 * OUTPUT:    Composite service manager with platform-specific implementation
 * BUSINESS:  Factory pattern enables clean platform abstraction and testing
 * CHANGE:    Initial factory with Windows and Linux platform support
 * RISK:      Medium - Platform detection affecting service functionality
 */
func NewCompositeServiceManager() (*CompositeServiceManager, error) {
	var platformManager PlatformServiceManager
	var err error
	
	switch runtime.GOOS {
	case "windows":
		// Windows platform manager will be implemented in platform-specific files
		return nil, fmt.Errorf("Windows service manager not yet implemented")
		
	case "linux":
		// Linux platform manager will be implemented in platform-specific files
		platformManager, err = NewLinuxServiceManager()
		if err != nil {
			return nil, fmt.Errorf("failed to create Linux service manager: %w", err)
		}
		
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	
	return &CompositeServiceManager{
		platformManager: platformManager,
	}, nil
}

// =============================================================================
// ServiceInstaller Interface Implementation
// =============================================================================

/**
 * CONTEXT:   Service installation implementation through platform manager
 * INPUT:     Service configuration and installation parameters
 * OUTPUT:    Service installed on system with appropriate configuration
 * BUSINESS:  Installation enables professional daemon deployment
 * CHANGE:    Interface segregation - installation operations only
 * RISK:      High - System modification requiring careful validation
 */
func (csm *CompositeServiceManager) Install(config interfaces.ServiceConfig) error {
	return csm.platformManager.Install(config)
}

/**
 * CONTEXT:   Service uninstallation implementation through platform manager
 * INPUT:     No parameters, operates on currently installed service
 * OUTPUT:    Service removed from system service manager
 * BUSINESS:  Uninstallation enables clean service removal
 * CHANGE:    Interface segregation - installation operations only
 * RISK:      High - System modification with service removal
 */
func (csm *CompositeServiceManager) Uninstall() error {
	return csm.platformManager.Uninstall()
}

/**
 * CONTEXT:   Installation status check through platform manager
 * INPUT:     No parameters, checks current system state
 * OUTPUT:    Boolean indicating if service is installed
 * BUSINESS:  Installation status enables conditional operations
 * CHANGE:    Interface segregation - installation operations only
 * RISK:      Low - Read-only system state check
 */
func (csm *CompositeServiceManager) IsInstalled() bool {
	return csm.platformManager.IsInstalled()
}

// =============================================================================
// ServiceController Interface Implementation
// =============================================================================

/**
 * CONTEXT:   Service start implementation through platform manager
 * INPUT:     No parameters, starts currently installed service
 * OUTPUT:    Service started and running on system
 * BUSINESS:  Start operation enables service activation
 * CHANGE:    Interface segregation - runtime operations only
 * RISK:      Medium - Service startup affecting system behavior
 */
func (csm *CompositeServiceManager) Start() error {
	return csm.platformManager.Start()
}

/**
 * CONTEXT:   Service stop implementation through platform manager
 * INPUT:     No parameters, stops currently running service
 * OUTPUT:    Service stopped and no longer running
 * BUSINESS:  Stop operation enables service deactivation
 * CHANGE:    Interface segregation - runtime operations only
 * RISK:      Medium - Service shutdown affecting system behavior
 */
func (csm *CompositeServiceManager) Stop() error {
	return csm.platformManager.Stop()
}

/**
 * CONTEXT:   Service restart implementation through platform manager
 * INPUT:     No parameters, restarts currently installed service
 * OUTPUT:    Service stopped then started with fresh state
 * BUSINESS:  Restart operation enables service state reset
 * CHANGE:    Interface segregation - runtime operations only
 * RISK:      Medium - Service restart affecting system behavior
 */
func (csm *CompositeServiceManager) Restart() error {
	return csm.platformManager.Restart()
}

/**
 * CONTEXT:   Running status check through platform manager
 * INPUT:     No parameters, checks current service state
 * OUTPUT:    Boolean indicating if service is currently running
 * BUSINESS:  Running status enables conditional runtime operations
 * CHANGE:    Interface segregation - runtime operations only
 * RISK:      Low - Read-only service state check
 */
func (csm *CompositeServiceManager) IsRunning() bool {
	return csm.platformManager.IsRunning()
}

// =============================================================================
// ServiceMonitor Interface Implementation
// =============================================================================

/**
 * CONTEXT:   Service status retrieval through platform manager
 * INPUT:     No parameters, retrieves comprehensive service status
 * OUTPUT:    Complete service status with metrics and diagnostic information
 * BUSINESS:  Status information enables monitoring and troubleshooting
 * CHANGE:    Interface segregation - monitoring operations only
 * RISK:      Low - Read-only service status information
 */
func (csm *CompositeServiceManager) Status() (interfaces.ServiceStatus, error) {
	return csm.platformManager.Status()
}

/**
 * CONTEXT:   Service log retrieval through platform manager
 * INPUT:     Number of log lines to retrieve
 * OUTPUT:    Service log entries with timestamps and diagnostic information
 * BUSINESS:  Log information enables troubleshooting and behavior analysis
 * CHANGE:    Interface segregation - monitoring operations only
 * RISK:      Low - Read-only log information retrieval
 */
func (csm *CompositeServiceManager) GetLogs(lines int) ([]interfaces.LogEntry, error) {
	return csm.platformManager.GetLogs(lines)
}

/**
 * CONTEXT:   Service health check through platform manager
 * INPUT:     No parameters, performs comprehensive health validation
 * OUTPUT:    Error if service is unhealthy, nil if healthy
 * BUSINESS:  Health check enables proactive monitoring and alerting
 * CHANGE:    Interface segregation - monitoring operations only
 * RISK:      Low - Read-only health status validation
 */
func (csm *CompositeServiceManager) HealthCheck() error {
	return csm.platformManager.HealthCheck()
}

/**
 * CONTEXT:   Service uptime retrieval through platform manager
 * INPUT:     No parameters, calculates service uptime duration
 * OUTPUT:    Duration since service start time
 * BUSINESS:  Uptime information enables performance monitoring
 * CHANGE:    Interface segregation - monitoring operations only
 * RISK:      Low - Read-only uptime calculation
 */
func (csm *CompositeServiceManager) GetUptime() (time.Duration, error) {
	return csm.platformManager.GetUptime()
}

// =============================================================================
// Interface Compliance Verification (Compile-time checks)
// =============================================================================

// Verify CompositeServiceManager implements all segregated interfaces
var _ interfaces.ServiceInstaller = (*CompositeServiceManager)(nil)
var _ interfaces.ServiceController = (*CompositeServiceManager)(nil)
var _ interfaces.ServiceMonitor = (*CompositeServiceManager)(nil)