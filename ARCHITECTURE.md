# Claude Monitor - System Architecture

This document describes the architectural foundation of the Claude Monitor system, designed for high-performance monitoring of Claude CLI sessions with minimal system overhead.

## Architecture Overview

The system follows a layered architecture with clean separation of concerns:

```
┌─────────────────────────────────────┐
│           CLI Layer                 │ ← User Commands (start, status, report, stop)
├─────────────────────────────────────┤
│        Application Layer            │ ← Business Logic (Session/WorkBlock Management)
├─────────────────────────────────────┤
│         Domain Layer                │ ← Core Entities (Session, WorkBlock, Process)
├─────────────────────────────────────┤
│       Infrastructure Layer          │ ← eBPF, Kùzu Database, System Services
└─────────────────────────────────────┘
```

## Project Structure

```
claude-monitor/
├── cmd/
│   ├── claude-daemon/          # Daemon entry point
│   └── claude-monitor/         # CLI entry point
├── internal/
│   ├── arch/                   # Architecture components
│   │   ├── interfaces.go       # Core system interfaces
│   │   ├── container.go        # Dependency injection
│   │   └── eventprocessor.go   # Event-driven architecture
│   ├── daemon/                 # Daemon implementation
│   ├── database/               # Database layer (to be implemented)
│   ├── ebpf/                   # eBPF programs (to be implemented)
│   ├── cli/                    # CLI interface
│   └── domain/                 # Domain entities
├── pkg/
│   ├── events/                 # Event system
│   └── logger/                 # Logging utilities
└── Makefile                    # Build system
```

## Core Components

### 1. Domain Layer (`internal/domain/`)

**Purpose**: Contains the core business entities and rules.

**Key Components**:
- `Session`: 5-hour tracking windows
- `WorkBlock`: Continuous activity periods within sessions  
- `Process`: System processes detected by eBPF
- `SystemStatus`: Current daemon state
- `SessionStats`: Aggregated reporting data

### 2. Architecture Layer (`internal/arch/`)

**Purpose**: Defines system interfaces and patterns for component coordination.

**Key Components**:
- `interfaces.go`: Core system contracts
- `container.go`: Dependency injection container
- `eventprocessor.go`: Event-driven processing pipeline

**Core Interfaces**:
- `EBPFManager`: eBPF program management
- `SessionManager`: Session lifecycle
- `WorkBlockManager`: Work block tracking
- `DatabaseManager`: Data persistence
- `EventProcessor`: Event handling coordination

### 3. Event System (`pkg/events/`)

**Purpose**: Kernel-userspace communication structures.

**Key Types**:
- `SystemEvent`: eBPF event representation
- `EventType`: Event classification (execve, connect, exit)
- `EventValidator`: Event validation and filtering

### 4. Dependency Injection

The system uses constructor-based dependency injection with a service container:

```go
// Service registration
container.RegisterSingleton((*SessionManager)(nil), func(c *ServiceContainer) (interface{}, error) {
    dbManager, _ := c.GetDatabaseManager()
    timeProvider, _ := c.Get((*TimeProvider)(nil))
    logger, _ := c.GetLogger()
    return NewSessionManager(dbManager, timeProvider, logger), nil
})

// Service resolution
sessionMgr, err := container.GetSessionManager()
```

### 5. Event-Driven Architecture

Events flow through a pipeline with prioritized handlers:

```go
// Event flow: eBPF → EventProcessor → BusinessLogicHandlers → Database
ebpfManager.GetEventChannel() → eventProcessor.ProcessEvent() → handlers.Handle()
```

## Service Lifecycle

### Daemon Startup
1. Initialize service container
2. Register all services with factories
3. Validate dependencies
4. Load eBPF programs
5. Start event processing
6. Register event handlers

### Event Processing
1. eBPF captures syscalls (execve, connect)
2. Events validated and filtered
3. Relevant events dispatched to handlers
4. Session/WorkBlock state updated
5. Changes persisted to database

### Graceful Shutdown
1. Stop event processing
2. Finalize current session/work block
3. Detach eBPF programs
4. Close database connections
5. Clean up resources

## Design Patterns

### 1. Repository Pattern
Database operations abstracted through repositories:
```go
type SessionRepository interface {
    Create(session *Session) error
    FindActive() (*Session, error)
    Update(session *Session) error
}
```

### 2. Observer Pattern
Event handlers register for specific event types:
```go
type EventHandler interface {
    CanHandle(eventType EventType) bool
    Handle(event *SystemEvent) error
    Priority() int
}
```

### 3. State Machine Pattern
Session lifecycle managed with clear state transitions:
```go
type SessionState int
const (
    SessionInactive SessionState = iota
    SessionActive
    SessionExpiring
)
```

## Concurrency Safety

All shared state is protected with appropriate synchronization:

- `sync.RWMutex` for read-heavy data structures
- Atomic operations for counters and flags
- Channel-based communication for event processing
- Context-based cancellation for graceful shutdown

## Error Handling

Structured error handling with context:

```go
type MonitorError struct {
    Component string
    Operation string
    Cause     error
    Timestamp time.Time
}
```

## Interface Contracts

### eBPF ↔ Go
- Events communicated through ring buffers
- Structured event format with validation
- Graceful program attachment/detachment

### Go ↔ Database
- Transaction-based operations
- Repository abstraction
- Connection pooling and health checks

### CLI ↔ Daemon
- Process-based communication
- PID file management
- Signal-based shutdown

## Extension Points

The architecture supports extension through:

1. **Event Handlers**: Add new business logic handlers
2. **Database Adapters**: Support different databases
3. **eBPF Programs**: Monitor additional syscalls
4. **CLI Commands**: Add new user operations
5. **Reporting Formats**: Custom output formats

## Specialized Agent Responsibilities

- **`ebpf-specialist`**: Implement eBPF programs and kernel integration
- **`daemon-core`**: Complete session/work block business logic
- **`database-manager`**: Implement Kùzu graph database operations
- **`cli-interface`**: Enhance CLI experience and reporting

## Performance Considerations

- eBPF ring buffers for high-throughput event capture
- Minimal heap allocations in event processing
- Database connection pooling
- Lazy initialization of expensive resources
- Event batching for database operations

## Security Considerations

- Root privilege requirement for eBPF
- PID file protection
- Input validation on all user inputs
- Secure temporary file handling
- Process isolation between CLI and daemon

This architecture provides a solid foundation for implementing a high-performance, maintainable monitoring system that can be extended by specialized agents while maintaining clean separation of concerns.