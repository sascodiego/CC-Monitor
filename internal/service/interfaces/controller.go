/**
 * CONTEXT:   Service controller interface for runtime operations only
 * INPUT:     Runtime control commands for service lifecycle management
 * OUTPUT:    Service start, stop, restart, and running status operations
 * BUSINESS:  Focused interface for service runtime control concerns only
 * CHANGE:    Interface Segregation Principle - extracted from fat ServiceManager interface
 * RISK:      Low - Interface segregation improving maintainability and testability
 */

package interfaces

/**
 * CONTEXT:   Service controller interface following Interface Segregation Principle
 * INPUT:     Runtime control commands without installation or monitoring dependencies
 * OUTPUT:    Service lifecycle operations (start, stop, restart, running status)
 * BUSINESS:  Clients needing only runtime control depend on minimal interface
 * CHANGE:    Initial ISP-compliant interface extracted from ServiceManager
 * RISK:      Low - Interface segregation reduces coupling and improves testability
 */
type ServiceController interface {
	// Start the service
	Start() error
	
	// Stop the service
	Stop() error
	
	// Restart the service (stop then start)
	Restart() error
	
	// Check if service is currently running
	IsRunning() bool
}