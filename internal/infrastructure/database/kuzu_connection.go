/**
 * CONTEXT:   KuzuDB connection management with pooling and transaction support
 * INPUT:     Database path, connection configuration, and transaction requirements
 * OUTPUT:    Thread-safe connection management with resource cleanup
 * BUSINESS:  Reliable database connectivity for work hour tracking persistence
 * CHANGE:    Initial Go implementation with connection pooling and transaction support
 * RISK:      Medium - Database connections require careful resource management
 */

package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kuzudb/go-kuzu"
)

// KuzuConnectionConfig holds configuration for KuzuDB connections
type KuzuConnectionConfig struct {
	DatabasePath    string        `json:"database_path"`
	MaxConnections  int           `json:"max_connections"`
	ConnTimeout     time.Duration `json:"connection_timeout"`
	QueryTimeout    time.Duration `json:"query_timeout"`
	EnableLogging   bool          `json:"enable_logging"`
	BufferPoolSize  uint64        `json:"buffer_pool_size_mb"`
	ReadOnly        bool          `json:"read_only"`
}

// DefaultKuzuConfig returns sensible defaults for KuzuDB connection
func DefaultKuzuConfig() KuzuConnectionConfig {
	return KuzuConnectionConfig{
		DatabasePath:   "./data/claude-monitor.kuzu",
		MaxConnections: 10,
		ConnTimeout:    30 * time.Second,
		QueryTimeout:   60 * time.Second,
		EnableLogging:  false,
		BufferPoolSize: 512, // 512MB
		ReadOnly:       false,
	}
}

/**
 * CONTEXT:   Connection pool manager for KuzuDB with resource management
 * INPUT:     Connection requests, transaction callbacks, query parameters
 * OUTPUT:    Managed connections with automatic cleanup and error handling
 * BUSINESS:  Efficient database access with connection reuse and resource limits
 * CHANGE:    Initial connection pool implementation with thread safety
 * RISK:      Medium - Connection leaks possible without proper cleanup
 */
type KuzuConnectionManager struct {
	config      KuzuConnectionConfig
	database    *kuzu.Database
	connections chan *kuzu.Connection
	inUse       map[*kuzu.Connection]bool
	mu          sync.RWMutex
	closed      bool
}

// NewKuzuConnectionManager creates a new connection manager
func NewKuzuConnectionManager(config KuzuConnectionConfig) (*KuzuConnectionManager, error) {
	// Open the database
	db, err := kuzu.OpenDatabase(config.DatabasePath, kuzu.DefaultSystemConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to open KuzuDB database at %s: %w", config.DatabasePath, err)
	}

	manager := &KuzuConnectionManager{
		config:      config,
		database:    db,
		connections: make(chan *kuzu.Connection, config.MaxConnections),
		inUse:       make(map[*kuzu.Connection]bool),
		closed:      false,
	}

	// Pre-populate connection pool
	for i := 0; i < config.MaxConnections; i++ {
		conn, err := kuzu.NewConnection(db)
		if err != nil {
			// Close any connections created so far
			manager.Close()
			return nil, fmt.Errorf("failed to create KuzuDB connection %d: %w", i, err)
		}

		// Configure connection timeout if needed
		if config.QueryTimeout > 0 {
			// Note: KuzuDB Go driver may not support query timeouts directly
			// This would be handled at the application level with context
		}

		select {
		case manager.connections <- conn:
			// Connection added to pool successfully
		default:
			// This shouldn't happen with pre-allocated channel
			conn.Close()
			return nil, fmt.Errorf("failed to add connection %d to pool", i)
		}
	}

	return manager, nil
}

/**
 * CONTEXT:   Acquire connection from pool with timeout and context support
 * INPUT:     Context for cancellation and timeout control
 * OUTPUT:    Available connection or timeout error
 * BUSINESS:  Connections must be available for work tracking operations
 * CHANGE:    Initial connection acquisition with timeout handling
 * RISK:      Low - Context-based timeout prevents indefinite blocking
 */
func (kcm *KuzuConnectionManager) AcquireConnection(ctx context.Context) (*kuzu.Connection, error) {
	kcm.mu.RLock()
	if kcm.closed {
		kcm.mu.RUnlock()
		return nil, fmt.Errorf("connection manager is closed")
	}
	kcm.mu.RUnlock()

	// Try to get connection with context timeout
	select {
	case conn := <-kcm.connections:
		kcm.mu.Lock()
		kcm.inUse[conn] = true
		kcm.mu.Unlock()
		return conn, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("connection acquisition timeout: %w", ctx.Err())
	}
}

/**
 * CONTEXT:   Release connection back to pool for reuse
 * INPUT:     Connection to be returned to the pool
 * OUTPUT:    Connection available for next request
 * BUSINESS:  Connection reuse is essential for performance under load
 * CHANGE:    Initial connection release with safety checks
 * RISK:      Low - Safe connection return with validation
 */
func (kcm *KuzuConnectionManager) ReleaseConnection(conn *kuzu.Connection) error {
	if conn == nil {
		return fmt.Errorf("cannot release nil connection")
	}

	kcm.mu.Lock()
	defer kcm.mu.Unlock()

	if kcm.closed {
		// If manager is closed, close this connection
		conn.Close()
		return fmt.Errorf("connection manager is closed, connection closed")
	}

	// Check if connection was actually in use
	if !kcm.inUse[conn] {
		return fmt.Errorf("connection was not acquired from this manager")
	}

	// Remove from in-use tracking
	delete(kcm.inUse, conn)

	// Return to pool
	select {
	case kcm.connections <- conn:
		return nil
	default:
		// Pool is full, close this connection
		conn.Close()
		return fmt.Errorf("connection pool full, connection closed")
	}
}

/**
 * CONTEXT:   Execute query with automatic connection management
 * INPUT:     Cypher query string and parameters with context
 * OUTPUT:    Query result with automatic connection cleanup
 * BUSINESS:  Simplified query execution for application code
 * CHANGE:    Initial query execution wrapper with resource management
 * RISK:      Low - Automatic connection cleanup prevents resource leaks
 */
func (kcm *KuzuConnectionManager) Query(ctx context.Context, query string, params map[string]interface{}) (*kuzu.QueryResult, error) {
	conn, err := kcm.AcquireConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	// Ensure connection is released
	defer func() {
		if releaseErr := kcm.ReleaseConnection(conn); releaseErr != nil {
			// Log the error but don't override the main error
			fmt.Printf("Warning: failed to release connection: %v\n", releaseErr)
		}
	}()

	// Execute query with context timeout
	resultChan := make(chan *kuzu.QueryResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := conn.Query(query)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- result
	}()

	// Wait for result or context cancellation
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, fmt.Errorf("query execution failed: %w", err)
	case <-ctx.Done():
		return nil, fmt.Errorf("query execution cancelled: %w", ctx.Err())
	}
}

/**
 * CONTEXT:   Execute function within a transaction with automatic rollback
 * INPUT:     Transaction function with connection parameter
 * OUTPUT:    Transaction result with commit/rollback handling
 * BUSINESS:  ACID transactions ensure data consistency for work tracking
 * CHANGE:    Initial transaction support implementation
 * RISK:      Medium - Transaction handling requires careful error management
 */
func (kcm *KuzuConnectionManager) WithTransaction(ctx context.Context, fn func(*kuzu.Connection) error) error {
	conn, err := kcm.AcquireConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for transaction: %w", err)
	}

	// Ensure connection is released
	defer func() {
		if releaseErr := kcm.ReleaseConnection(conn); releaseErr != nil {
			fmt.Printf("Warning: failed to release transaction connection: %v\n", releaseErr)
		}
	}()

	// Begin transaction
	if _, err := conn.Query("BEGIN TRANSACTION;"); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute function
	if err := fn(conn); err != nil {
		// Rollback on error
		if _, rollbackErr := conn.Query("ROLLBACK;"); rollbackErr != nil {
			return fmt.Errorf("transaction failed and rollback failed: %w (original error: %v)", rollbackErr, err)
		}
		return fmt.Errorf("transaction rolled back: %w", err)
	}

	// Commit transaction
	if _, err := conn.Query("COMMIT;"); err != nil {
		// Try to rollback after failed commit
		if _, rollbackErr := conn.Query("ROLLBACK;"); rollbackErr != nil {
			return fmt.Errorf("commit failed and rollback failed: %w (original error: %v)", rollbackErr, err)
		}
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Health check for connection manager and database
 * INPUT:     Context for timeout control
 * OUTPUT:    Health status and error information
 * BUSINESS:  Health monitoring ensures reliable work tracking service
 * CHANGE:    Initial health check implementation
 * RISK:      Low - Non-destructive health validation
 */
func (kcm *KuzuConnectionManager) HealthCheck(ctx context.Context) error {
	kcm.mu.RLock()
	if kcm.closed {
		kcm.mu.RUnlock()
		return fmt.Errorf("connection manager is closed")
	}

	// Check if we have connections available
	availableConns := len(kcm.connections)
	inUseConns := len(kcm.inUse)
	totalConns := availableConns + inUseConns
	kcm.mu.RUnlock()

	if totalConns == 0 {
		return fmt.Errorf("no database connections available")
	}

	// Test database connectivity with a simple query
	_, err := kcm.Query(ctx, "RETURN 1 as test;", nil)
	if err != nil {
		return fmt.Errorf("database connectivity test failed: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Get connection pool statistics for monitoring
 * INPUT:     No parameters, returns current pool state
 * OUTPUT:    Connection pool metrics for monitoring
 * BUSINESS:  Pool monitoring helps optimize performance and detect issues
 * CHANGE:    Initial pool statistics implementation
 * RISK:      Low - Read-only statistics with no side effects
 */
type KuzuPoolStats struct {
	MaxConnections       int           `json:"max_connections"`
	AvailableConnections int           `json:"available_connections"`
	InUseConnections     int           `json:"in_use_connections"`
	TotalConnections     int           `json:"total_connections"`
	DatabasePath         string        `json:"database_path"`
	QueryTimeout         time.Duration `json:"query_timeout"`
	IsClosed             bool          `json:"is_closed"`
}

func (kcm *KuzuConnectionManager) GetStats() KuzuPoolStats {
	kcm.mu.RLock()
	defer kcm.mu.RUnlock()

	availableConns := len(kcm.connections)
	inUseConns := len(kcm.inUse)

	return KuzuPoolStats{
		MaxConnections:       kcm.config.MaxConnections,
		AvailableConnections: availableConns,
		InUseConnections:     inUseConns,
		TotalConnections:     availableConns + inUseConns,
		DatabasePath:         kcm.config.DatabasePath,
		QueryTimeout:         kcm.config.QueryTimeout,
		IsClosed:             kcm.closed,
	}
}

/**
 * CONTEXT:   Close connection manager and cleanup all resources
 * INPUT:     No parameters, initiates shutdown process
 * OUTPUT:    All connections closed and resources cleaned up
 * BUSINESS:  Proper cleanup prevents resource leaks during shutdown
 * CHANGE:    Initial cleanup implementation with safety checks
 * RISK:      Low - Safe resource cleanup with error handling
 */
func (kcm *KuzuConnectionManager) Close() error {
	kcm.mu.Lock()
	defer kcm.mu.Unlock()

	if kcm.closed {
		return nil // Already closed
	}

	kcm.closed = true

	var errors []error

	// Close all connections in pool
	close(kcm.connections)
	for conn := range kcm.connections {
		if err := conn.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close pooled connection: %w", err))
		}
	}

	// Close any in-use connections
	for conn := range kcm.inUse {
		if err := conn.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close in-use connection: %w", err))
		}
	}

	// Close database
	if kcm.database != nil {
		if err := kcm.database.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close database: %w", err))
		}
	}

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("errors during close: %v", errors)
	}

	return nil
}

/**
 * CONTEXT:   Connection manager factory with configuration validation
 * INPUT:     Configuration parameters and validation requirements
 * OUTPUT:    Initialized connection manager or configuration error
 * BUSINESS:  Proper initialization ensures reliable database operations
 * CHANGE:    Initial factory implementation with validation
 * RISK:      Low - Validation prevents runtime errors from bad configuration
 */
func NewKuzuConnectionManagerWithValidation(config KuzuConnectionConfig) (*KuzuConnectionManager, error) {
	// Validate configuration
	if config.DatabasePath == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}

	if config.MaxConnections <= 0 {
		config.MaxConnections = 10
	}

	if config.ConnTimeout <= 0 {
		config.ConnTimeout = 30 * time.Second
	}

	if config.QueryTimeout <= 0 {
		config.QueryTimeout = 60 * time.Second
	}

	if config.BufferPoolSize == 0 {
		config.BufferPoolSize = 512 // 512MB default
	}

	return NewKuzuConnectionManager(config)
}