# Daemon-Managed Session Correlation System

## Overview

This implementation provides a robust daemon-managed approach for Claude hook correlation that eliminates the need for temporary files and environment variables. Instead, it uses intelligent context-based matching with database persistence for reliable session tracking.

## 🚀 Key Benefits

- **No Temporary Files**: Eliminates file system dependencies and cleanup issues
- **No Environment Variables**: Removes complexity of passing data between processes  
- **Context-Based Correlation**: Uses terminal PID, working directory, and project information for matching
- **Database Persistence**: Survives daemon restarts and provides reliable state management
- **Smart Matching**: Multiple correlation strategies with confidence scoring
- **Thread-Safe**: Handles concurrent sessions across multiple terminals safely

## 🏗️ Architecture Components

### 1. Active Session Entity (`/internal/entities/active_session.go`)
- Represents active sessions awaiting correlation
- Contains session context (terminal PID, working directory, user ID)
- Provides context matching algorithms with confidence scoring
- Validates session timing and state transitions

### 2. ActiveSessionTracker (`/internal/usecases/active_session_tracker.go`)
- Core business logic for session correlation
- Manages in-memory session tracking with database persistence
- Implements smart correlation algorithms:
  - **Primary**: Exact terminal PID + user match
  - **Secondary**: Working directory + user match  
  - **Tertiary**: User-only match with timing validation
- Handles concurrent sessions and cleanup of expired sessions

### 3. Context Detection (`/internal/utils/context_detector.go`)
- Automatically detects session context without manual input
- Cross-platform user ID detection (USER, USERNAME, LOGNAME)
- Terminal PID detection from parent process
- Working directory and project path detection
- Git repository and project indicator file detection

### 4. Database Repository (`/internal/infrastructure/database/kuzu_active_session_repository.go`)
- KuzuDB implementation for active session persistence
- Context-based queries for correlation matching
- Cleanup operations for expired sessions
- Statistical reporting for monitoring

### 5. HTTP Endpoints (`/internal/infrastructure/http/session_handlers.go`)
- `/api/session/start` - Create active session from hook start
- `/api/session/end` - Correlate and complete session from hook end  
- `/api/session/status` - Monitor active session state
- `/api/correlation/test` - Test correlation algorithms

### 6. Simplified Hook Command (`/cmd/claude-hook/main.go`)
- Single binary for all hook operations
- Automatic context detection - no manual configuration
- Retry logic and error handling
- Support for start, end, and activity events

### 7. Integration Script (`/examples/claude-code-hook-integration.sh`)
- Bash script for Claude Code integration
- Health checks and fallback handling
- Environment variable support
- Debug and monitoring capabilities

## 🔄 Session Lifecycle

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Hook Start    │───▶│  Active Session  │───▶│ Completed Session │
│                 │    │                  │    │                   │
│ • Detect Context│    │ • Track State    │    │ • Historical Record│
│ • Create Session│    │ • Wait for End   │    │ • Final Metrics   │
│ • Store in DB   │    │ • Handle Activity│    │ • Remove from Active│
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Start Event Processing
1. Hook detects context (terminal PID, working dir, user)
2. Sends context to daemon via `/api/session/start`
3. Daemon creates ActiveSession entity
4. Session stored in database and tracked in memory
5. Response confirms session creation

### End Event Processing  
1. Hook detects same context information
2. Sends context + metrics to daemon via `/api/session/end`
3. Daemon finds matching active session using:
   - Exact terminal PID match (preferred)
   - Working directory match (fallback)
   - User match with timing validation (last resort)
4. Session completed with actual metrics
5. Historical session record created
6. Active session removed from tracking

## 🎯 Correlation Strategies

### Primary Strategy: Terminal PID Matching
```go
// Exact terminal match (highest confidence)
if terminalPID == session.TerminalPID && userID == session.UserID {
    confidence = 0.8 + bonuses // Very high confidence
}
```

### Secondary Strategy: Project Directory Matching  
```go
// Working directory match (medium confidence)  
if workingDir == session.WorkingDir && userID == session.UserID {
    confidence = 0.5 + bonuses // Medium confidence
}
```

### Tertiary Strategy: User + Timing Validation
```go
// User match with timing check (low confidence)
if userID == session.UserID && timingReasonable(session, endTime) {
    confidence = 0.2 + bonuses // Low confidence
}
```

### Confidence Scoring
- **Base Match**: Terminal (0.6), Project (0.4), User (0.1)
- **Shell PID Bonus**: +0.2 for exact shell process match
- **Directory Bonus**: +0.1 for exact directory match  
- **Project Bonus**: +0.1 for exact project path match
- **Timing Bonus**: +0.3 for sessions close to estimated duration
- **Minimum Threshold**: 0.6 required for confident correlation

## 🗄️ Database Schema

```sql
-- Active sessions table for correlation
CREATE TABLE active_claude_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    terminal_pid INTEGER NOT NULL,
    shell_pid INTEGER NOT NULL,
    working_dir TEXT NOT NULL,  
    project_path TEXT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    estimated_end_time TIMESTAMP NOT NULL,
    last_activity TIMESTAMP NOT NULL,
    status TEXT NOT NULL, -- 'active', 'processing', 'ended'
    activity_count INTEGER NOT NULL,
    processing_duration_ms INTEGER DEFAULT 0,
    token_count INTEGER DEFAULT 0,
    estimated_tokens INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Indexes for fast correlation lookups
CREATE INDEX idx_active_sessions_terminal ON active_claude_sessions(terminal_pid, user_id);
CREATE INDEX idx_active_sessions_project ON active_claude_sessions(working_dir, user_id);  
CREATE INDEX idx_active_sessions_user ON active_claude_sessions(user_id);
CREATE INDEX idx_active_sessions_status ON active_claude_sessions(status);
```

## 🧪 Testing Coverage

### Unit Tests (`*_test.go`)
- **ActiveSessionTracker**: Correlation logic, concurrent operations, error handling
- **ContextDetector**: Cross-platform context detection, validation, edge cases
- **ActiveSession Entity**: State transitions, validation, scoring algorithms
- **HTTP Handlers**: Request validation, error responses, integration flows

### Integration Tests
- **Complete Lifecycle**: Start → Correlation → Completion with real database
- **Concurrent Sessions**: Multiple sessions across different terminals
- **Error Recovery**: Database failures, timeout handling, cleanup operations
- **Performance**: Benchmarks for correlation speed under load

### Test Scenarios Covered
- ✅ Exact terminal PID matches
- ✅ Multiple sessions in same terminal (scoring)
- ✅ Project-based fallback correlation
- ✅ User-only correlation with timing
- ✅ No match scenarios (error handling)
- ✅ Concurrent session creation and correlation
- ✅ Database failure recovery
- ✅ Session timeout and cleanup
- ✅ Context detection across platforms
- ✅ Invalid input validation

## 🚀 Usage Examples

### Basic Hook Integration
```bash
# Start hook (pre_action)
claude-hook --type=start --daemon-url=http://localhost:8080

# End hook (post_action)  
claude-hook --type=end --duration=30 --tokens=1500 --daemon-url=http://localhost:8080
```

### Claude Code Integration
```bash
# In Claude Code hook configuration
export CLAUDE_DAEMON_URL="http://localhost:8080"
export CLAUDE_HOOK_DEBUG="true"

# Pre-action hook
./examples/claude-code-hook-integration.sh pre_action

# Post-action hook with metrics
CLAUDE_DURATION=45 CLAUDE_TOKENS=2000 \
./examples/claude-code-hook-integration.sh post_action
```

### Daemon Configuration
```go
// Start daemon with active session tracking
tracker := usecases.NewActiveSessionTracker(usecases.ActiveSessionTrackerConfig{
    ActiveSessionRepo:     kuzuActiveRepo,
    SessionRepo:           kuzuSessionRepo,
    Logger:                logger,
    MaxConcurrentSessions: 100,
    SessionTimeout:        30 * time.Minute,
    CleanupInterval:       5 * time.Minute,
})

// Add session endpoints to HTTP server
extendedHandlers := http.NewExtendedHandlers(http.ExtendedHandlerConfig{
    HandlerConfig:  baseConfig,
    SessionTracker: tracker,
})

router.HandleFunc("/api/session/start", extendedHandlers.HandleSessionStart).Methods("POST")
router.HandleFunc("/api/session/end", extendedHandlers.HandleSessionEnd).Methods("POST")
```

## 🔧 Configuration Options

### Context Detector Configuration
```go
config := ContextDetectorConfig{
    DetectProjectName: true,     // Use directory name as project name
    UseGitRepository:  true,     // Detect git repo roots as project boundaries  
    FallbackUserID:    "unknown", // Fallback when user detection fails
}
```

### Session Tracker Configuration
```go
config := ActiveSessionTrackerConfig{
    MaxConcurrentSessions: 100,           // Prevent memory exhaustion
    SessionTimeout:        30 * time.Minute, // Auto-cleanup threshold
    CleanupInterval:       5 * time.Minute,  // Background cleanup frequency
}
```

### Hook Command Configuration
```bash
# Environment variables
export CLAUDE_DAEMON_URL="http://localhost:8080"    # Daemon endpoint
export CLAUDE_HOOK_DEBUG="true"                     # Debug output
export CLAUDE_HOOK_TIMEOUT="10s"                    # Request timeout
export CLAUDE_HOOK_RETRY="3"                        # Retry attempts
```

## 📊 Monitoring and Debugging

### Health Endpoints
```bash
# Check daemon health
curl http://localhost:8080/api/health

# Check active session status
curl http://localhost:8080/api/session/status

# Test correlation logic
curl -X POST http://localhost:8080/api/correlation/test \
  -H "Content-Type: application/json" \
  -d '{"test_context": {...}, "test_mode": "find_match"}'
```

### Debug Output
```bash
# Enable debug logging
claude-hook --type=start --debug --daemon-url=http://localhost:8080

# Output includes:
# - Detected context information
# - HTTP request/response details  
# - Correlation matching process
# - Performance timing
```

### Metrics and Statistics
```go
// Get tracker statistics
stats := tracker.GetStatistics()
// Returns: active_sessions, terminal_indexes, project_indexes, status_breakdown
```

## 🏆 Advantages Over Previous Approach

| Aspect | File-Based | Daemon-Managed |
|--------|------------|----------------|
| **Reliability** | File I/O failures, permission issues | Database persistence, transaction safety |
| **Performance** | File creation/deletion overhead | In-memory tracking with database backup |
| **Concurrency** | File locking contention | Thread-safe data structures |
| **Debugging** | Limited visibility into correlation | Full logging and monitoring capabilities |
| **Recovery** | Lost files = lost sessions | Database survives daemon restarts |
| **Security** | Temp files readable by others | Memory-only with secure database |
| **Maintenance** | Manual cleanup required | Automatic garbage collection |
| **Scalability** | File system limitations | Database can handle thousands of sessions |

## 🔮 Future Enhancements

1. **Machine Learning Correlation**: Train models on successful correlations to improve matching accuracy
2. **Distributed Sessions**: Support for multiple daemon instances with shared correlation state  
3. **Advanced Analytics**: Pattern recognition for user productivity insights
4. **Real-time Monitoring**: WebSocket endpoints for live session tracking
5. **Performance Optimization**: Caching strategies and connection pooling for high-load scenarios

## 🎯 Success Metrics

- **Correlation Accuracy**: >99% successful hook start/end matching
- **Performance**: <10ms average correlation time under normal load
- **Reliability**: Zero data loss due to correlation failures
- **Concurrency**: Support for 100+ concurrent sessions per user
- **Recovery**: Full session state recovery after daemon restart

This implementation provides a production-ready, scalable, and reliable foundation for Claude hook correlation that eliminates the complexity and fragility of temporary file-based approaches while providing superior performance and debugging capabilities.