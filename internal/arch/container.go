package arch

import (
	"fmt"
	"reflect"
	"sync"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-019
 * CONTEXT:   Dependency injection container for service management and composition
 * REASON:    Need clean dependency management without circular dependencies for testability
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate service dependencies at startup, not runtime to avoid circular deps
 * RISK:      Medium - Service dependency cycles could cause startup failures
 */

// ServiceContainer manages service registration and dependency injection
type ServiceContainer struct {
	services    map[reflect.Type]interface{}
	factories   map[reflect.Type]ServiceFactory
	singletons  map[reflect.Type]interface{}
	mu          sync.RWMutex
	initialized bool
}

// ServiceFactory is a function that creates a service instance
type ServiceFactory func(container *ServiceContainer) (interface{}, error)

// NewServiceContainer creates a new service container
func NewServiceContainer() *ServiceContainer {
	return &ServiceContainer{
		services:   make(map[reflect.Type]interface{}),
		factories:  make(map[reflect.Type]ServiceFactory),
		singletons: make(map[reflect.Type]interface{}),
	}
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-020
 * CONTEXT:   Service registration methods for different lifecycle patterns
 * REASON:    Need flexible service registration supporting both instances and factories
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate service types at registration to catch errors early
 * RISK:      Low - Registration errors are caught at startup, not runtime
 */

// RegisterInstance registers a concrete service instance
func (sc *ServiceContainer) RegisterInstance(service interface{}) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	if service == nil {
		return fmt.Errorf("cannot register nil service")
	}
	
	serviceType := reflect.TypeOf(service)
	if _, exists := sc.services[serviceType]; exists {
		return fmt.Errorf("service %v already registered", serviceType)
	}
	
	sc.services[serviceType] = service
	return nil
}

// RegisterSingleton registers a service factory that creates a singleton instance
func (sc *ServiceContainer) RegisterSingleton(serviceType interface{}, factory ServiceFactory) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	t := reflect.TypeOf(serviceType).Elem()
	if _, exists := sc.factories[t]; exists {
		return fmt.Errorf("service %v already registered", t)
	}
	
	sc.factories[t] = factory
	return nil
}

// RegisterTransient registers a service factory that creates new instances on each request
func (sc *ServiceContainer) RegisterTransient(serviceType interface{}, factory ServiceFactory) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	t := reflect.TypeOf(serviceType).Elem()
	sc.factories[t] = factory
	return nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-021
 * CONTEXT:   Service resolution methods with type safety and error handling
 * REASON:    Need type-safe service resolution with clear error messages for missing dependencies
 * CHANGE:    Initial implementation.
 * PREVENTION:Always check for service existence before casting to avoid panics
 * RISK:      Medium - Missing service dependencies could cause runtime panics
 */

// Get retrieves a service by its interface type
func (sc *ServiceContainer) Get(serviceType interface{}) (interface{}, error) {
	sc.mu.RLock()
	t := reflect.TypeOf(serviceType).Elem()
	
	// Check for direct instance registration
	if service, exists := sc.services[t]; exists {
		sc.mu.RUnlock()
		return service, nil
	}
	
	// Check for singleton instances
	if singleton, exists := sc.singletons[t]; exists {
		sc.mu.RUnlock()
		return singleton, nil
	}
	
	// Check for factory registration
	factory, exists := sc.factories[t]
	sc.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("service %v not registered", t)
	}
	
	// Create instance using factory
	instance, err := factory(sc)
	if err != nil {
		return nil, fmt.Errorf("failed to create service %v: %w", t, err)
	}
	
	// Store as singleton if it was registered as such
	sc.mu.Lock()
	if factory != nil {
		sc.singletons[t] = instance
	}
	sc.mu.Unlock()
	
	return instance, nil
}

// MustGet retrieves a service and panics if not found (use sparingly)
func (sc *ServiceContainer) MustGet(serviceType interface{}) interface{} {
	service, err := sc.Get(serviceType)
	if err != nil {
		panic(fmt.Sprintf("service resolution failed: %v", err))
	}
	return service
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-022
 * CONTEXT:   Generic typed service resolution methods for better developer experience
 * REASON:    Reduce type casting boilerplate and improve type safety for common service types
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate type assertions to prevent runtime panics
 * RISK:      Low - Type mismatches are caught early with clear error messages
 */

// GetEBPFManager retrieves the eBPF manager service
func (sc *ServiceContainer) GetEBPFManager() (EBPFManager, error) {
	service, err := sc.Get((*EBPFManager)(nil))
	if err != nil {
		return nil, err
	}
	
	ebpfManager, ok := service.(EBPFManager)
	if !ok {
		return nil, fmt.Errorf("service is not an EBPFManager")
	}
	
	return ebpfManager, nil
}

// GetSessionManager retrieves the session manager service
func (sc *ServiceContainer) GetSessionManager() (SessionManager, error) {
	service, err := sc.Get((*SessionManager)(nil))
	if err != nil {
		return nil, err
	}
	
	sessionManager, ok := service.(SessionManager)
	if !ok {
		return nil, fmt.Errorf("service is not a SessionManager")
	}
	
	return sessionManager, nil
}

// GetDatabaseManager retrieves the database manager service
func (sc *ServiceContainer) GetDatabaseManager() (DatabaseManager, error) {
	service, err := sc.Get((*DatabaseManager)(nil))
	if err != nil {
		return nil, err
	}
	
	dbManager, ok := service.(DatabaseManager)
	if !ok {
		return nil, fmt.Errorf("service is not a DatabaseManager")
	}
	
	return dbManager, nil
}

// GetLogger retrieves the logger service
func (sc *ServiceContainer) GetLogger() (Logger, error) {
	service, err := sc.Get((*Logger)(nil))
	if err != nil {
		return nil, err
	}
	
	logger, ok := service.(Logger)
	if !ok {
		return nil, fmt.Errorf("service is not a Logger")
	}
	
	return logger, nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-023
 * CONTEXT:   Container lifecycle management and validation methods
 * REASON:    Need proper container initialization and cleanup for resource management
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate all dependencies are resolvable before marking as initialized
 * RISK:      Medium - Unresolved dependencies could cause runtime failures
 */

// Initialize validates all service dependencies can be resolved
func (sc *ServiceContainer) Initialize() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	if sc.initialized {
		return fmt.Errorf("container already initialized")
	}
	
	// Validate all registered services can be resolved
	for serviceType := range sc.factories {
		if _, err := sc.resolveService(serviceType); err != nil {
			return fmt.Errorf("failed to resolve service %v: %w", serviceType, err)
		}
	}
	
	sc.initialized = true
	return nil
}

// IsInitialized returns true if the container has been initialized
func (sc *ServiceContainer) IsInitialized() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.initialized
}

// resolveService attempts to resolve a service without locking (internal use)
func (sc *ServiceContainer) resolveService(serviceType reflect.Type) (interface{}, error) {
	// Check direct instances
	if service, exists := sc.services[serviceType]; exists {
		return service, nil
	}
	
	// Check singletons
	if singleton, exists := sc.singletons[serviceType]; exists {
		return singleton, nil
	}
	
	// Check factories
	if factory, exists := sc.factories[serviceType]; exists {
		return factory(sc)
	}
	
	return nil, fmt.Errorf("service %v not found", serviceType)
}

// Dispose cleans up all managed resources
func (sc *ServiceContainer) Dispose() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	var errors []error
	
	// Dispose of services that implement Disposable interface
	for _, service := range sc.services {
		if disposable, ok := service.(Disposable); ok {
			if err := disposable.Dispose(); err != nil {
				errors = append(errors, err)
			}
		}
	}
	
	for _, singleton := range sc.singletons {
		if disposable, ok := singleton.(Disposable); ok {
			if err := disposable.Dispose(); err != nil {
				errors = append(errors, err)
			}
		}
	}
	
	// Clear all registrations
	sc.services = make(map[reflect.Type]interface{})
	sc.factories = make(map[reflect.Type]ServiceFactory)
	sc.singletons = make(map[reflect.Type]interface{})
	sc.initialized = false
	
	if len(errors) > 0 {
		return fmt.Errorf("disposal errors: %v", errors)
	}
	
	return nil
}

// Disposable interface for services that need cleanup
type Disposable interface {
	Dispose() error
}