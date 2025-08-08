/**
 * CONTEXT:   Centralized port configuration for Claude Monitor system
 * INPUT:     Global constants for all network ports used in the system
 * OUTPUT:    Single source of truth for port configuration across all components
 * BUSINESS:  Centralized configuration prevents port conflicts and configuration errors
 * CHANGE:    New centralized port configuration replacing hardcoded values
 * RISK:      Low - Configuration constants with clear documentation
 */

package config

const (
	// DefaultDaemonPort is the default port for the Claude Monitor daemon HTTP server
	DefaultDaemonPort = "9193"
	
	// DefaultDaemonHost is the default host for the Claude Monitor daemon
	DefaultDaemonHost = "localhost"
	
	// DefaultListenAddr combines host and port for HTTP server binding
	DefaultListenAddr = DefaultDaemonHost + ":" + DefaultDaemonPort
	
	// DefaultDaemonURL is the full URL for daemon connections
	DefaultDaemonURL = "http://" + DefaultListenAddr
)

// GetDaemonURL returns the daemon URL with optional host/port overrides
func GetDaemonURL(host, port string) string {
	if host == "" {
		host = DefaultDaemonHost
	}
	if port == "" {
		port = DefaultDaemonPort
	}
	return "http://" + host + ":" + port
}

// GetListenAddr returns the listen address with optional host/port overrides
func GetListenAddr(host, port string) string {
	if host == "" {
		host = DefaultDaemonHost
	}
	if port == "" {
		port = DefaultDaemonPort
	}
	return host + ":" + port
}