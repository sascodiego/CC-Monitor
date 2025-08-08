/**
 * CONTEXT:   Mock implementations for dependency injection testing
 * INPUT:     Test scenarios requiring mocked external dependencies
 * OUTPUT:    Mock implementations enabling isolated unit testing
 * BUSINESS:  Mock dependencies enable reliable testing of business logic
 * CHANGE:    Initial mock implementations for SOLID testing architecture
 * RISK:      Low - Test-only mock implementations with no production impact
 */

package main

import (
	"fmt"
	"os"
)

/**
 * CONTEXT:   Mock command executor for testing system commands
 * INPUT:     Predefined command responses and error scenarios
 * OUTPUT:    Controlled command execution results for testing
 * BUSINESS:  Mock executor enables testing without actual system commands
 * CHANGE:    Initial mock implementation for command execution testing
 * RISK:      Low - Test-only mock with configurable responses
 */
type MockCommandExecutor struct {
	// Commands that should return success
	SuccessCommands map[string][]byte
	// Commands that should return error
	ErrorCommands map[string]error
	// Track executed commands for verification
	ExecutedCommands []string
}

func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		SuccessCommands:  make(map[string][]byte),
		ErrorCommands:    make(map[string]error),
		ExecutedCommands: make([]string, 0),
	}
}

func (m *MockCommandExecutor) Execute(name string, args ...string) ([]byte, error) {
	cmdKey := name
	if len(args) > 0 {
		cmdKey = fmt.Sprintf("%s %s", name, args[0])
	}
	
	m.ExecutedCommands = append(m.ExecutedCommands, cmdKey)
	
	if err, exists := m.ErrorCommands[cmdKey]; exists {
		return nil, err
	}
	
	if output, exists := m.SuccessCommands[cmdKey]; exists {
		return output, nil
	}
	
	// Default success response
	return []byte("mock success"), nil
}

func (m *MockCommandExecutor) ExecuteWithOutput(name string, args ...string) ([]byte, []byte, error) {
	stdout, err := m.Execute(name, args...)
	return stdout, nil, err
}

func (m *MockCommandExecutor) ExecuteBackground(name string, args ...string) (*os.Process, error) {
	_, err := m.Execute(name, args...)
	if err != nil {
		return nil, err
	}
	// Return a mock process (in real tests, this would be more sophisticated)
	return &os.Process{Pid: 12345}, nil
}

func (m *MockCommandExecutor) CommandExists(name string) bool {
	// Configurable command existence for testing
	return name != "nonexistent-command"
}

/**
 * CONTEXT:   Mock file system provider for testing file operations
 * INPUT:     Simulated file system state and operations
 * OUTPUT:    Controlled file system responses for testing
 * BUSINESS:  Mock file system enables testing without actual file operations
 * CHANGE:    Initial mock implementation for file system testing
 * RISK:      Low - Test-only mock with in-memory file system simulation
 */
type MockFileSystemProvider struct {
	// Simulated files (path -> content)
	Files map[string][]byte
	// Simulated directories
	Directories map[string]bool
	// Track operations for verification
	Operations []string
}

func NewMockFileSystemProvider() *MockFileSystemProvider {
	return &MockFileSystemProvider{
		Files:       make(map[string][]byte),
		Directories: make(map[string]bool),
		Operations:  make([]string, 0),
	}
}

func (m *MockFileSystemProvider) FileExists(path string) bool {
	m.Operations = append(m.Operations, fmt.Sprintf("FileExists:%s", path))
	_, exists := m.Files[path]
	return exists
}

func (m *MockFileSystemProvider) IsExecutable(path string) bool {
	m.Operations = append(m.Operations, fmt.Sprintf("IsExecutable:%s", path))
	return m.FileExists(path)
}

func (m *MockFileSystemProvider) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.Operations = append(m.Operations, fmt.Sprintf("WriteFile:%s", path))
	m.Files[path] = data
	return nil
}

func (m *MockFileSystemProvider) ReadFile(path string) ([]byte, error) {
	m.Operations = append(m.Operations, fmt.Sprintf("ReadFile:%s", path))
	if content, exists := m.Files[path]; exists {
		return content, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

func (m *MockFileSystemProvider) RemoveFile(path string) error {
	m.Operations = append(m.Operations, fmt.Sprintf("RemoveFile:%s", path))
	delete(m.Files, path)
	return nil
}

func (m *MockFileSystemProvider) CreateDir(path string, perm os.FileMode) error {
	m.Operations = append(m.Operations, fmt.Sprintf("CreateDir:%s", path))
	m.Directories[path] = true
	return nil
}

func (m *MockFileSystemProvider) RemoveDir(path string) error {
	m.Operations = append(m.Operations, fmt.Sprintf("RemoveDir:%s", path))
	delete(m.Directories, path)
	return nil
}

func (m *MockFileSystemProvider) GetWorkingDir() (string, error) {
	m.Operations = append(m.Operations, "GetWorkingDir")
	return "/mock/working/dir", nil
}

func (m *MockFileSystemProvider) GetUserHomeDir() (string, error) {
	m.Operations = append(m.Operations, "GetUserHomeDir")
	return "/mock/home/user", nil
}

/**
 * CONTEXT:   Mock database provider for testing database operations
 * INPUT:     Simulated database queries and responses
 * OUTPUT:    Controlled database responses for testing
 * BUSINESS:  Mock database enables testing without actual database operations
 * CHANGE:    Initial mock implementation for database testing
 * RISK:      Low - Test-only mock with in-memory data simulation
 */
type MockDatabaseProvider struct {
	// Track queries for verification
	Queries []string
	// Simulated query responses
	QueryResponses map[string][]map[string]interface{}
	// Control connection state
	IsConnected bool
	ShouldFailConnection bool
}

func NewMockDatabaseProvider() *MockDatabaseProvider {
	return &MockDatabaseProvider{
		Queries:        make([]string, 0),
		QueryResponses: make(map[string][]map[string]interface{}),
		IsConnected:    false,
	}
}

func (m *MockDatabaseProvider) Connect(databasePath string) error {
	if m.ShouldFailConnection {
		return fmt.Errorf("mock connection failure")
	}
	m.IsConnected = true
	return nil
}

func (m *MockDatabaseProvider) Close() error {
	m.IsConnected = false
	return nil
}

func (m *MockDatabaseProvider) Ping() error {
	if !m.IsConnected {
		return fmt.Errorf("not connected")
	}
	return nil
}

func (m *MockDatabaseProvider) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	m.Queries = append(m.Queries, query)
	
	if response, exists := m.QueryResponses[query]; exists {
		return response, nil
	}
	
	// Default empty response
	return []map[string]interface{}{}, nil
}

func (m *MockDatabaseProvider) Execute(query string, args ...interface{}) error {
	m.Queries = append(m.Queries, query)
	return nil
}

func (m *MockDatabaseProvider) BeginTransaction() (Transaction, error) {
	return &MockTransaction{}, nil
}

/**
 * CONTEXT:   Mock transaction for testing database transaction operations
 * INPUT:     Transaction queries and commit/rollback operations
 * OUTPUT:    Controlled transaction responses for testing
 * BUSINESS:  Mock transaction enables testing atomic operations
 * CHANGE:    Initial mock implementation for transaction testing
 * RISK:      Low - Test-only mock with transaction state simulation
 */
type MockTransaction struct {
	Queries []string
	IsCommitted bool
	IsRolledBack bool
}

func (m *MockTransaction) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	m.Queries = append(m.Queries, query)
	return []map[string]interface{}{}, nil
}

func (m *MockTransaction) Execute(query string, args ...interface{}) error {
	m.Queries = append(m.Queries, query)
	return nil
}

func (m *MockTransaction) Commit() error {
	m.IsCommitted = true
	return nil
}

func (m *MockTransaction) Rollback() error {
	m.IsRolledBack = true
	return nil
}