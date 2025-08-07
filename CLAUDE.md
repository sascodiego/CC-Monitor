# Claude Monitor System - Development Assistant
# Go + KuzuDB + Hook Integration Architecture

You are an expert Go developer working on Claude Monitor, a user-friendly work hour tracking system for Claude Code users. This system uses Claude Code's hook system for accurate activity detection, Go for reliable backend processing, and KuzuDB for rich data analytics.

## Project Overview

This is a work hour tracking system that provides:
1. **Claude Sessions**: 5-hour windows that begin with first user interaction
2. **Work Hours**: Active usage time blocks with 5-minute inactivity timeout
3. **Project Detection**: Automatic identification of the current project
4. **Dual Time Metrics**: Both active work time and total schedule tracking
5. **Rich Reporting**: Daily, weekly, monthly, and historical analytics

## Architecture Stack - Go + KuzuDB

- **Go Language**: Reliable, fast development with excellent tooling
- **Claude Code Hooks**: "claude-code action" command for precise activity detection
- **KuzuDB Graph Database**: Complex relational queries for work pattern analysis
- **Gorilla/mux**: HTTP server for receiving activity events
- **Cobra CLI**: User-friendly command-line interface
- **WSL Environment**: Optimized for Windows Subsystem for Linux

## System Specifications

| Component | Specification | Benefit |
|-----------|--------------|----------|
| Activity Detection | 100% accuracy via hooks | Perfect work tracking |
| Project Detection | Automatic from working dir | No manual configuration |
| Time Tracking | Dual metrics (active + total) | Complete work insights |
| Session Management | 5-hour windows | Matches Claude usage |
| Reporting | Daily/Weekly/Monthly | Comprehensive analytics |

## Key System Components

### Claude Code Hook Command
```go
// Hook command executed before each Claude action
package main

import (
    "encoding/json"
    "net/http"
    "os"
    "path/filepath"
    "time"
)

type ActivityEvent struct {
    Timestamp   time.Time `json:"timestamp"`
    ProjectPath string    `json:"project_path"`
    ProjectName string    `json:"project_name"`
    UserID      string    `json:"user_id"`
}

func main() {
    // Detect current project automatically
    workingDir, _ := os.Getwd()
    projectName := filepath.Base(workingDir)
    
    event := ActivityEvent{
        Timestamp:   time.Now(),
        ProjectPath: workingDir,
        ProjectName: projectName,
        UserID:      os.Getenv("USER"),
    }
    
    // Send to daemon via HTTP
    sendToDaemon(event)
}
```

### Go Daemon (HTTP Server)
```go
// Main daemon processing activity events
package main

import (
    "encoding/json"
    "net/http"
    "time"
    
    "github.com/gorilla/mux"
)

type ClaudeMonitor struct {
    sessionManager *SessionManager
    workTracker    *WorkBlockTracker
    database       *KuzuDBConnection
}

func (cm *ClaudeMonitor) handleActivity(w http.ResponseWriter, r *http.Request) {
    var event ActivityEvent
    json.NewDecoder(r.Body).Decode(&event)
    
    // Process the activity event
    session := cm.sessionManager.GetOrCreateSession(event.Timestamp)
    workBlock := cm.workTracker.UpdateWorkBlock(session.ID, event)
    
    // Save to database
    cm.database.SaveActivity(session, workBlock, event)
}
```

### KuzuDB Integration
```go
// Direct Go integration with KuzuDB
package database

import "github.com/kuzudb/kuzu-go"

type KuzuDBConnection struct {
    db *kuzu.Database
    conn *kuzu.Connection
}

func (k *KuzuDBConnection) SaveActivity(session *Session, workBlock *WorkBlock, event ActivityEvent) error {
    query := `
        MERGE (u:User {id: $user_id})
        MERGE (p:Project {name: $project_name, path: $project_path})
        MERGE (s:Session {id: $session_id, start_time: $session_start})
        MERGE (w:WorkBlock {id: $work_id, start_time: $work_start})
        MERGE (u)-[:HAS_SESSION]->(s)
        MERGE (s)-[:CONTAINS_WORK]->(w)
        MERGE (w)-[:WORK_IN_PROJECT]->(p)
    `
    
    return k.conn.Query(query, params)
}
```

## Business Logic Rules

### Session Tracking
- New session starts when: `now > currentSessionEndTime`
- Session duration: exactly 5 hours from first interaction
- Multiple interactions within 5 hours belong to same session

### Work Hour Tracking
- New work block starts when: `now - lastActivityTime > 5 minutes`
- Work blocks are contained within sessions
- Final work block recorded on daemon shutdown

## Development Guidelines - Go Focus

1. **Simplicity**: Use Go's straightforward syntax for maintainable code
2. **Error Handling**: Explicit error checking with proper error propagation
3. **Goroutines**: Use lightweight goroutines for concurrent processing
4. **Standard Library**: Leverage Go's rich standard library for common tasks
5. **Testing**: Use Go's built-in testing framework with table-driven tests

## Code Quality Standards - Go Edition

### **MANDATORY COMMENT STANDARD**

**BEFORE** each function, struct, or significant code section, you **MUST** add:

```go
/**
 * CONTEXT:   [Purpose and context of the code block]
 * INPUT:     [Expected input parameters and their constraints]
 * OUTPUT:    [Expected output and possible error conditions]
 * BUSINESS:  [Business logic rules this code implements]
 * CHANGE:    [If modifying: describe change. If new: "Initial implementation."]
 * RISK:      [Risk Level (Low/Medium/High) and mitigation strategy]
 */
```

### Go-Specific Examples:

```go
/**
 * CONTEXT:   Session manager handling 5-hour window logic for Claude sessions
 * INPUT:     Activity events with timestamps and project information
 * OUTPUT:    Session object with start/end times, error if session creation fails
 * BUSINESS:  New session starts when current session expired (5 hours since start)
 * CHANGE:    Initial implementation with mutex for thread safety.
 * RISK:      Low - Mutex contention possible under high load, but activity is typically low
 */
type SessionManager struct {
    mu              sync.RWMutex
    currentSession  *Session
    sessionDuration time.Duration // 5 hours
}

func (sm *SessionManager) GetOrCreateSession(activityTime time.Time) *Session {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    if sm.currentSession == nil || activityTime.Sub(sm.currentSession.StartTime) > sm.sessionDuration {
        sm.currentSession = &Session{
            ID:        generateSessionID(activityTime),
            StartTime: activityTime,
            EndTime:   activityTime.Add(sm.sessionDuration),
        }
    }
    
    return sm.currentSession
}

/**
 * CONTEXT:   Work block tracker for detecting active work periods vs idle time
 * INPUT:     Session ID, current activity timestamp, project name
 * OUTPUT:    Updated work block with accurate timing, handles idle detection
 * BUSINESS:  New work block starts after 5+ minutes of inactivity
 * CHANGE:    Initial implementation with idle timeout logic.
 * RISK:      Medium - Incorrect idle detection could skew work time calculations
 */
func (wt *WorkBlockTracker) UpdateWorkBlock(sessionID string, timestamp time.Time, project string) *WorkBlock {
    wt.mu.Lock()
    defer wt.mu.Unlock()
    
    lastActivity, exists := wt.lastActivity[sessionID]
    
    // Start new work block if idle > 5 minutes or first activity
    if !exists || timestamp.Sub(lastActivity) > 5*time.Minute {
        workBlock := &WorkBlock{
            ID:          generateWorkBlockID(sessionID, timestamp),
            SessionID:   sessionID,
            ProjectName: project,
            StartTime:   timestamp,
        }
        wt.activeBlocks[sessionID] = workBlock
    }
    
    // Update last activity time
    wt.lastActivity[sessionID] = timestamp
    
    return wt.activeBlocks[sessionID]
}
```

## Testing Strategy

- **Table-driven tests** for business logic validation
- **HTTP integration tests** for daemon API endpoints
- **Database integration tests** with real KuzuDB instance
- **CLI integration tests** with end-to-end scenarios
- **Mock testing** for external dependencies
- **Time-based testing** for session and work block logic
- **Coverage target**: > 80% with go test -cover

## Go Development Best Practices

This project follows Go best practices and idioms for maintainable, reliable code.

### **Core Development Principles**

#### **Go Simplicity**
- **Philosophy**: "Simple, reliable, efficient"
- **Approach**: Prefer clear, explicit code over clever optimizations
- **Error Handling**: Always check and handle errors explicitly
- **Documentation**: Use godoc-style comments for all public functions
- **Example**: "Write straightforward Go code that any team member can understand"

#### **Concurrent Safety**
- **Specialization**: Goroutines and channel-based communication
- **Approach**: Use sync.Mutex for shared state, channels for communication
- **Testing**: Test concurrent code with race detector (-race flag)
- **Monitoring**: Use context.Context for cancellation and timeouts
#### **Database Integration**
- **Specialization**: KuzuDB graph database with Go
- **Approach**: Use official Go driver for KuzuDB
- **Focus**: Cypher queries, schema design, transaction management
- **Testing**: Integration tests with real database instances
- **Example**: "Design graph schema for session-work-project relationships"

#### **CLI Development**
- **Specialization**: Command-line interface with Cobra
- **Approach**: User-friendly commands with clear help text
- **Focus**: Beautiful output formatting, progress indicators
- **Testing**: End-to-end CLI testing scenarios
- **Example**: "Create intuitive commands for daily/weekly/monthly reports"

#### **HTTP Server Design**
- **Specialization**: RESTful API design with gorilla/mux
- **Approach**: Simple, reliable HTTP endpoints
- **Focus**: Request validation, error handling, middleware
- **Testing**: HTTP integration tests with test servers
- **Example**: "Design /activity endpoint for receiving hook events"

### **Comment Examples for Go Code**

```go
/**
 * CONTEXT:   HTTP endpoint for receiving activity events from Claude Code hooks
 * INPUT:     JSON ActivityEvent with timestamp, project info, user ID
 * OUTPUT:    HTTP 200 on success, HTTP 400 on invalid JSON, HTTP 500 on processing error
 * BUSINESS:  Processes activity to update sessions and work blocks
 * CHANGE:    Initial implementation with basic validation and error handling.
 * RISK:      Low - Input validation prevents most issues, database errors logged
 */

/**
 * AGENT:     rust-async-engineer
 * TRACE:     CLAUDE-RS-200
 * CONTEXT:   Bounded channel for backpressure management
 * REASON:    Prevent memory exhaustion from unbounded event queue
 * CHANGE:    Initial Rust implementation with explicit backpressure.
 * PREVENTION:Monitor channel capacity and adjust based on load testing
 * RISK:      Medium - Event loss if channel fills during spike
 */

/**
 * AGENT:     rust-ffi-specialist
 * TRACE:     CLAUDE-RS-300
 * CONTEXT:   Safe wrapper for KuzuDB C API with RAII cleanup
 * REASON:    Ensure database resources are properly released
 * CHANGE:    Initial FFI implementation with Drop trait.
 * RISK:      Low - Input validation prevents most issues, database errors logged
 */
func (cm *ClaudeMonitor) handleActivity(w http.ResponseWriter, r *http.Request) {
    var event ActivityEvent
    
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid JSON format", http.StatusBadRequest)
        return
    }
    
    // Validate required fields
    if event.ProjectName == "" || event.Timestamp.IsZero() {
        http.Error(w, "Missing required fields", http.StatusBadRequest)
        return
    }
    
    // Process the activity event
    if err := cm.processActivity(event); err != nil {
        log.Printf("Error processing activity: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "processed"})
}
```

## System Goals

### Go Implementation Targets
- **Activity Detection**: 100% accuracy via Claude Code hooks
- **Response Time**: < 1 second for all CLI commands
- **Database Queries**: < 100ms for reporting queries
- **Hook Overhead**: < 50ms per Claude Code action
- **Memory Usage**: < 100MB resident set size
- **User Experience**: Intuitive CLI with beautiful output

## Implementation Checklist

- [ ] Claude Code hook command ("claude-code action")
- [ ] Go HTTP daemon for processing events
- [ ] KuzuDB integration with Go driver
- [ ] Session management (5-hour windows)
- [ ] Work block tracking (5-minute idle detection)
- [ ] CLI with Cobra framework
- [ ] Beautiful output formatting
- [ ] Historical reporting features
- [ ] Testing suite with real workflow scenarios
- [ ] Documentation and user guides

## Important Development Notes

1. **Hook Integration**: Ensure "claude-code action" executes reliably before each Claude action
2. **Project Detection**: Use working directory to automatically identify current project
3. **Time Accuracy**: Precise timestamp handling for session and work block calculations
4. **Error Recovery**: Graceful handling of daemon unavailability with local file fallback
5. **User Experience**: Prioritize clear, actionable CLI output over technical metrics

When working on this project, leverage Go's strengths: simplicity, reliability, and excellent tooling. The hook-based approach provides perfect accuracy while the graph database enables rich analytics and reporting.

**Focus on user value: accurate work tracking with beautiful, actionable reports.**