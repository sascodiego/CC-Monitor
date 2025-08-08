/**
 * CONTEXT:   Service installer interface for installation-only operations
 * INPUT:     Service configuration and installation parameters
 * OUTPUT:    Service installation, removal, and installation status checking
 * BUSINESS:  Focused interface for service installation concerns only
 * CHANGE:    Interface Segregation Principle - extracted from fat ServiceManager interface
 * RISK:      Low - Interface segregation improving maintainability and testability
 */

package interfaces

/**
 * CONTEXT:   Service installer interface following Interface Segregation Principle
 * INPUT:     Service configuration for installation operations
 * OUTPUT:    Installation operations without runtime or monitoring dependencies
 * BUSINESS:  Clients needing only installation operations depend on minimal interface
 * CHANGE:    Initial ISP-compliant interface extracted from ServiceManager
 * RISK:      Low - Interface segregation reduces coupling and improves testability
 */
type ServiceInstaller interface {
	// Install service with provided configuration
	Install(config ServiceConfig) error
	
	// Uninstall service from system
	Uninstall() error
	
	// Check if service is installed on system
	IsInstalled() bool
}

/**
 * CONTEXT:   Service configuration structure for installation operations
 * INPUT:     Installation parameters and service settings
 * OUTPUT:    Complete service configuration for cross-platform installation
 * BUSINESS:  Configuration enables consistent service installation across platforms
 * CHANGE:    Shared configuration structure used by installer interface
 * RISK:      Medium - Configuration structure affects installation reliability
 */
type ServiceConfig struct {
	Name            string            `json:"name"`
	DisplayName     string            `json:"display_name"`
	Description     string            `json:"description"`
	ExecutablePath  string            `json:"executable_path"`
	Arguments       []string          `json:"arguments"`
	WorkingDir      string            `json:"working_dir"`
	User            string            `json:"user,omitempty"`
	Group           string            `json:"group,omitempty"`
	StartMode       ServiceStartMode  `json:"start_mode"`
	RestartOnFailure bool             `json:"restart_on_failure"`
	LogLevel        string            `json:"log_level"`
	Environment     map[string]string `json:"environment"`
}

type ServiceStartMode string

const (
	StartModeAuto     ServiceStartMode = "auto"
	StartModeManual   ServiceStartMode = "manual" 
	StartModeDisabled ServiceStartMode = "disabled"
)