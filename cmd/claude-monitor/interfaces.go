/**
 * CONTEXT:   Dependency Inversion Principle interfaces for external dependencies
 * INPUT:     Abstract interfaces for testing and dependency injection
 * OUTPUT:    Mockable interfaces enabling SOLID architecture and testability  
 * BUSINESS:  Interfaces decouple concrete implementations from business logic
 * CHANGE:    Initial implementation of DIP interfaces for SOLID refactoring
 * RISK:      Low - Interface definitions with no implementation dependencies
 */

package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	
	"github.com/claude-monitor/system/internal/database/sqlite"
)

/**
 * CONTEXT:   Command execution abstraction for system operations
 * INPUT:     Command name, arguments, and execution context
 * OUTPUT:    Command results, stdout/stderr, and error handling
 * BUSINESS:  Abstracts system command execution for service management
 * CHANGE:    Initial interface for os/exec dependency inversion
 * RISK:      Low - Interface abstraction with no side effects
 */
type CommandExecutor interface {
	// Execute command with arguments and return combined output
	Execute(name string, args ...string) ([]byte, error)
	
	// Execute command with separate stdout/stderr handling  
	ExecuteWithOutput(name string, args ...string) (stdout []byte, stderr []byte, err error)
	
	// Execute command in background and return process handle
	ExecuteBackground(name string, args ...string) (*os.Process, error)
	
	// Check if command/binary exists in system PATH
	CommandExists(name string) bool
}

/**
 * CONTEXT:   File system operations abstraction for service management
 * INPUT:     File paths, permissions, and file system operations
 * OUTPUT:    File operations results and error handling
 * BUSINESS:  Abstracts file system operations for cross-platform compatibility
 * CHANGE:    Initial interface for os package dependency inversion
 * RISK:      Low - Interface abstraction for file operations
 */
type FileSystemProvider interface {
	// File existence and permission checks
	FileExists(path string) bool
	IsExecutable(path string) bool
	
	// File operations
	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadFile(path string) ([]byte, error)
	RemoveFile(path string) error
	
	// Directory operations
	CreateDir(path string, perm os.FileMode) error
	RemoveDir(path string) error
	
	// Path operations
	GetWorkingDir() (string, error)
	GetUserHomeDir() (string, error)
}

/**
 * CONTEXT:   Database operations abstraction for data persistence
 * INPUT:     Database queries, connections, and transaction management
 * OUTPUT:    Query results, connection status, and error handling
 * BUSINESS:  Abstracts database operations for testability and flexibility
 * CHANGE:    Initial interface for database dependency inversion
 * RISK:      Medium - Database abstraction affecting data consistency
 */
type DatabaseProvider interface {
	// Connection management
	Connect(databasePath string) error
	Close() error
	Ping() error
	
	// Query operations
	Query(query string, args ...interface{}) ([]map[string]interface{}, error)
	Execute(query string, args ...interface{}) error
	
	// Transaction support
	BeginTransaction() (Transaction, error)
}

/**
 * CONTEXT:   Database transaction abstraction for atomic operations
 * INPUT:     Transaction queries and rollback/commit operations
 * OUTPUT:    Transaction results and error handling
 * BUSINESS:  Ensures data consistency through atomic operations
 * CHANGE:    Initial interface for transaction dependency inversion
 * RISK:      High - Transaction management affecting data integrity
 */
type Transaction interface {
	Query(query string, args ...interface{}) ([]map[string]interface{}, error)
	Execute(query string, args ...interface{}) error
	Commit() error
	Rollback() error
}

// Default implementations using concrete dependencies
/**
 * CONTEXT:   Default command executor using os/exec package
 * INPUT:     System commands and arguments
 * OUTPUT:    Command execution results using standard library
 * BUSINESS:  Provides real system command execution for production use
 * CHANGE:    Initial implementation of CommandExecutor interface
 * RISK:      Medium - Direct system command execution with security implications
 */
type DefaultCommandExecutor struct{}

func (d *DefaultCommandExecutor) Execute(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

func (d *DefaultCommandExecutor) ExecuteWithOutput(name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.Command(name, args...)
	stdout, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return stdout, exitError.Stderr, err
		}
		return stdout, nil, err
	}
	return stdout, nil, nil
}

func (d *DefaultCommandExecutor) ExecuteBackground(name string, args ...string) (*os.Process, error) {
	cmd := exec.Command(name, args...)
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd.Process, nil
}

func (d *DefaultCommandExecutor) CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

/**
 * CONTEXT:   Default file system provider using os package
 * INPUT:     File paths and file system operations
 * OUTPUT:    File system operation results using standard library
 * BUSINESS:  Provides real file system access for production use
 * CHANGE:    Initial implementation of FileSystemProvider interface
 * RISK:      Medium - Direct file system access with security implications
 */
type DefaultFileSystemProvider struct{}

func (d *DefaultFileSystemProvider) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (d *DefaultFileSystemProvider) IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

func (d *DefaultFileSystemProvider) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (d *DefaultFileSystemProvider) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (d *DefaultFileSystemProvider) RemoveFile(path string) error {
	return os.Remove(path)
}

func (d *DefaultFileSystemProvider) CreateDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (d *DefaultFileSystemProvider) RemoveDir(path string) error {
	return os.RemoveAll(path)
}

func (d *DefaultFileSystemProvider) GetWorkingDir() (string, error) {
	return os.Getwd()
}

func (d *DefaultFileSystemProvider) GetUserHomeDir() (string, error) {
	return os.UserHomeDir()
}

/**
 * CONTEXT:   SQLite database provider adapter for dependency injection
 * INPUT:     SQLite database connection and query operations
 * OUTPUT:    DatabaseProvider interface implementation for SQLite
 * BUSINESS:  Adapts existing SQLite implementation to DatabaseProvider interface
 * CHANGE:    Initial adapter implementation for SQLite database integration
 * RISK:      Medium - Database adapter affecting data consistency and queries
 */
type SQLiteDatabaseProvider struct {
	db *sqlite.SQLiteDB
}

func NewSQLiteDatabaseProvider(db *sqlite.SQLiteDB) *SQLiteDatabaseProvider {
	return &SQLiteDatabaseProvider{db: db}
}

func (s *SQLiteDatabaseProvider) Connect(databasePath string) error {
	// Connection already established via constructor
	return nil
}

func (s *SQLiteDatabaseProvider) Close() error {
	return s.db.Close()
}

func (s *SQLiteDatabaseProvider) Ping() error {
	// SQLite doesn't need explicit ping - test with a simple query
	_, err := s.db.DB().Exec("SELECT 1")
	return err
}

func (s *SQLiteDatabaseProvider) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	// TODO: Implement generic query method for SQLite
	// This would require adapting SQLite result sets to generic map format
	return nil, fmt.Errorf("generic query method not implemented yet")
}

func (s *SQLiteDatabaseProvider) Execute(query string, args ...interface{}) error {
	_, err := s.db.DB().Exec(query, args...)
	return err
}

func (s *SQLiteDatabaseProvider) BeginTransaction() (Transaction, error) {
	tx, err := s.db.DB().Begin()
	if err != nil {
		return nil, err
	}
	return &SQLiteTransaction{tx: tx}, nil
}

/**
 * CONTEXT:   SQLite transaction adapter for dependency injection
 * INPUT:     SQLite transaction operations
 * OUTPUT:    Transaction interface implementation for SQLite
 * BUSINESS:  Provides atomic operations through Transaction interface
 * CHANGE:    Initial transaction adapter for SQLite integration
 * RISK:      High - Transaction management affecting data integrity
 */
type SQLiteTransaction struct {
	tx *sql.Tx
}

func (s *SQLiteTransaction) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	// TODO: Implement generic query method for transactions
	return nil, fmt.Errorf("transaction query method not implemented yet")
}

func (s *SQLiteTransaction) Execute(query string, args ...interface{}) error {
	_, err := s.tx.Exec(query, args...)
	return err
}

func (s *SQLiteTransaction) Commit() error {
	return s.tx.Commit()
}

func (s *SQLiteTransaction) Rollback() error {
	return s.tx.Rollback()
}