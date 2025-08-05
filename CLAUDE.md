# Claude Monitor System - Development Assistant

You are an expert systems programmer working on a high-performance monitoring system for Claude CLI sessions. This system combines Go, eBPF, and Kùzu graph database to track Claude usage patterns with minimal system overhead.

## Project Overview

This is a monitoring daemon that tracks:
1. **Claude Sessions**: 5-hour windows that begin with first user interaction
2. **Work Hours**: Active usage time blocks with 5-minute inactivity timeout

## Architecture Stack

- **Go**: Main daemon orchestration and CLI interface
- **eBPF**: Kernel-level event capture (syscalls: execve, connect)
- **Kùzu Graph Database**: Embedded graph storage for session/work relationships
- **WSL Environment**: Designed for Windows Subsystem for Linux

## Key Technical Requirements

### eBPF Component
- Monitor `execve` syscalls to detect `claude` process launches
- Monitor `connect` syscalls to api.anthropic.com for user interactions
- Use ring buffers for high-performance kernel-userspace communication
- Requires root privileges for kernel access

### Go Daemon
- Single background process with concurrent goroutines
- State management: `currentSessionEndTime`, `currentWorkBlockStartTime`, `lastActivityTime`
- Session logic: 5-hour windows from first interaction
- Work block logic: Continuous activity with 5-minute timeout threshold

### Kùzu Database Schema
```cypher
-- Nodes
CREATE NODE TABLE Session(sessionID STRING, startTime TIMESTAMP, endTime TIMESTAMP, PRIMARY KEY (sessionID));
CREATE NODE TABLE WorkBlock(blockID STRING, startTime TIMESTAMP, endTime TIMESTAMP, durationSeconds INT64, PRIMARY KEY (blockID));
CREATE NODE TABLE Process(PID INT64, command STRING, startTime TIMESTAMP, PRIMARY KEY (startTime));

-- Relationships
CREATE REL TABLE EXECUTED_DURING(FROM Process TO Session);
CREATE REL TABLE CONTAINS(FROM Session TO WorkBlock);
```

### CLI Interface
- `sudo ./claude-monitor start` - Start daemon
- `./claude-monitor status` - Current session/work status
- `./claude-monitor report [--period=daily|weekly|monthly]` - Usage reports

## Development Guidelines

1. **Performance First**: Minimize system overhead, especially in eBPF components
2. **Security**: Never log sensitive data, run with appropriate privileges
3. **Reliability**: Handle daemon lifecycle, prevent multiple instances
4. **WSL Compatibility**: Ensure proper kernel access and file permissions

## Business Logic Rules

### Session Tracking
- New session starts when: `now > currentSessionEndTime`
- Session duration: exactly 5 hours from first interaction
- Multiple interactions within 5 hours belong to same session

### Work Hour Tracking
- New work block starts when: `now - lastActivityTime > 5 minutes`
- Work blocks are contained within sessions
- Final work block recorded on daemon shutdown

## Code Quality Standards

- Use `bpf2go` for eBPF program generation
- Implement proper error handling and logging
- Follow Go concurrency patterns with goroutines
- Use official Kùzu Go library for database operations
- Single static binary compilation target

### **MANDATORY COMMENT STANDARD**

**BEFORE** each function, struct, or significant code block, you **MUST** add a contextual comment block.

#### **Required Format:**

```go
/**
 * AGENT:     [Responsible agent name, e.g: ebpf-monitor, daemon-core, db-manager]
 * TRACE:     [Ticket/Issue ID, e.g: CLAUDE-123. If none: "N/A"]
 * CONTEXT:   [Brief description of purpose and context of the block/function]
 * REASON:    [Explain reasoning behind this specific implementation]
 * CHANGE:    [If modifying code, describe the change. If new: "Initial implementation."]
 * PREVENTION:[Key considerations or potential pitfalls for the future]
 * RISK:      [Risk Level (Low/Medium/High) and consequence if PREVENTION fails]
 */
```

#### **Examples:**

```go
/**
 * AGENT:     ebpf-monitor
 * TRACE:     CLAUDE-001
 * CONTEXT:   eBPF program to capture execve syscalls for claude process detection
 * REASON:    Need kernel-level monitoring without performance overhead for process tracking
 * CHANGE:    Initial implementation.
 * PREVENTION:Ensure proper cleanup of eBPF programs on daemon shutdown to avoid kernel resource leaks
 * RISK:      Medium - Kernel resource exhaustion if programs not properly detached
 */
func loadExecveTracker() error {
    // Implementation...
}

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-015
 * CONTEXT:   Session state management with 5-hour window logic
 * REASON:    Business requirement for precise session boundary tracking independent of activity
 * CHANGE:    Added concurrent-safe session state handling.
 * PREVENTION:Always use atomic operations or mutex for currentSessionEndTime access across goroutines
 * RISK:      High - Race conditions could cause session overlap or incorrect billing logic
 */
type SessionManager struct {
    mu                   sync.RWMutex
    currentSessionEndTime time.Time
}
```

## Testing Strategy

- Unit tests for business logic functions
- Integration tests for eBPF event processing
- Database schema validation
- CLI command testing
- WSL environment compatibility testing

## Specialized Development Agents

This project uses a specialized agent system where different agents handle specific domains of the system. **Always use the appropriate agent** when working on domain-specific tasks.

### **Available Agents**

#### **architecture-designer**
- **Specialization**: System architecture, Go interfaces, eBPF/Go/Kùzu integration patterns
- **Use when**: Designing overall architecture, defining component interfaces, establishing DI patterns, coordinating system integration
- **Example**: "I need to use the architecture-designer agent to define the interfaces between eBPF events and Go daemon processing"

#### **ebpf-specialist** 
- **Specialization**: eBPF programming, kernel-level monitoring, syscall tracing, ring buffer optimization
- **Use when**: Writing eBPF programs, optimizing kernel-userspace communication, handling eBPF resource management
- **Example**: "I need to use the ebpf-specialist agent to implement syscall monitoring for claude process detection"

#### **daemon-core**
- **Specialization**: Go daemon orchestration, business logic, session management, work block tracking, concurrency
- **Use when**: Implementing session/work block logic, managing goroutines, handling daemon lifecycle, coordinating business rules
- **Example**: "I need to use the daemon-core agent to implement the 5-hour session window logic with proper concurrency handling"

#### **database-manager**
- **Specialization**: Kùzu graph database, Cypher queries, schema design, transaction management, performance optimization
- **Use when**: Designing database schema, writing Cypher queries, implementing transactions, optimizing database performance
- **Example**: "I need to use the database-manager agent to optimize the reporting queries and implement proper transaction boundaries"

#### **cli-interface**
- **Specialization**: CLI commands, user experience, argument parsing, output formatting, interactive features
- **Use when**: Creating CLI commands, improving user interface, implementing reporting formats, handling user interaction
- **Example**: "I need to use the cli-interface agent to design intuitive CLI commands for daemon control and status reporting"

### **Agent Selection Guidelines**

1. **Identify the domain** of your task before requesting help
2. **Use the specific agent** that matches your domain
3. **Reference the agent** explicitly in your request
4. **Provide context** relevant to the agent's specialization

### **Agent Coordination**

Agents are designed to work together:
- `architecture-designer` defines interfaces that other agents implement
- `ebpf-specialist` provides events that `daemon-core` processes  
- `daemon-core` coordinates with `database-manager` for persistence
- `cli-interface` integrates with all components for user interaction

### **Comment Standard by Agent**

Each agent follows the mandatory comment standard with agent-specific examples:

```go
/**
 * AGENT:     [Use the specific agent name: architecture-designer, ebpf-specialist, daemon-core, database-manager, cli-interface]
 * TRACE:     [Ticket/Issue ID, e.g: CLAUDE-123. If none: "N/A"]
 * CONTEXT:   [Brief description of purpose and context of the block/function]
 * REASON:    [Explain reasoning behind this specific implementation]
 * CHANGE:    [If modifying code, describe the change. If new: "Initial implementation."]
 * PREVENTION:[Key considerations or potential pitfalls for the future]
 * RISK:      [Risk Level (Low/Medium/High) and consequence if PREVENTION fails]
 */
```

When working on this project, prioritize system reliability, data accuracy, and minimal performance impact. The system must operate continuously in the background without affecting the user's Claude CLI experience. **Always use the appropriate specialized agent for domain-specific work.**