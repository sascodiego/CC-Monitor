# Claude Monitor Hook Integration

This is the Claude Code hook integration for the Claude Monitor work tracking system. It provides seamless activity tracking by executing before each Claude Code action.

## Overview

The `claude-monitor` command is designed to be executed as a Claude Code hook. It:

- Executes in <50ms to not impact user experience
- Automatically detects the current project from working directory
- Sends activity events to the Claude Monitor daemon
- Falls back to local file logging if daemon is unavailable
- Works across Windows, macOS, and Linux (WSL optimized)

## Quick Start

### 1. Build the Hook Command

```bash
# Build the claude-monitor hook command
make build-hook
# or
go build -o claude-monitor ./cmd/claude-monitor
```

### 2. Start the Daemon

```bash
# Start the Claude Monitor daemon
./claude-daemon
```

### 3. Configure Claude Code Hooks

Add the hook to your Claude Code configuration:

```json
{
  "hooks": {
    "pre_action": "/path/to/claude-monitor"
  }
}
```

### 4. Verify Integration

```bash
# Test hook execution
./claude-monitor -debug

# Check daemon status
curl http://localhost:8080/health
```

## Project Detection

The hook automatically detects projects using multiple strategies:

### Detection Methods
1. **Git repositories** - Uses `.git` directory and branch info
2. **Package managers** - `package.json`, `go.mod`, `Cargo.toml`, etc.
3. **Build files** - `Makefile`, `CMakeLists.txt`, `pom.xml`
4. **Directory structure** - Avoids generic names like `src`, `lib`

### Supported Project Types
- **Go** - `go.mod`, `go.sum` files
- **Rust** - `Cargo.toml`, `Cargo.lock` files
- **JavaScript** - `package.json` with Node.js ecosystem
- **TypeScript** - `tsconfig.json` with `package.json`
- **Python** - `requirements.txt`, `setup.py`, `pyproject.toml`
- **Web** - `index.html`, `webpack.config.js`, etc.
- **General** - Any directory with reasonable project structure

## Configuration

Configuration file: `~/.claude-monitor/config.json`

### Example Configuration

```json
{
  "enabled": true,
  "daemon_url": "http://localhost:8080",
  "timeout_ms": 100,
  "log_level": "info",
  "project_names": {
    "/home/user/projects/my-app": "MyApp",
    "/home/user/work/client-project": "ClientProject"
  },
  "ignore_patterns": [
    "*/tmp/*",
    "*/node_modules/*",
    "*/.git/*"
  ]
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable the hook |
| `daemon_url` | string | `"http://localhost:8080"` | Daemon endpoint URL |
| `timeout_ms` | number | `100` | HTTP timeout in milliseconds |
| `log_level` | string | `"info"` | Log level (debug, info, warn, error) |
| `project_names` | object | `{}` | Custom project name mappings |
| `ignore_patterns` | array | `[]` | Path patterns to ignore |

## Command Line Options

```bash
# Show version information
claude-monitor -version

# Show help and usage
claude-monitor -help

# Enable debug logging
claude-monitor -debug

# Use custom daemon URL
claude-monitor -daemon http://custom:9090

# Set custom timeout
claude-monitor -timeout 200ms

# Use custom log file for fallback
claude-monitor -log-file /tmp/claude-activity.log

# Add command context
claude-monitor -command "file edit" -description "Editing main.go"
```

## Performance Requirements

The hook is designed for minimal impact on Claude Code:

- **Target execution time**: <50ms
- **HTTP timeout**: 100ms (configurable)
- **Memory usage**: <10MB
- **Silent failures**: Never interrupts Claude Code

### Performance Testing

```bash
# Run performance tests
go test -run TestPerformanceRequirements ./cmd/claude-monitor

# Benchmark hook execution
go test -bench=. ./cmd/claude-monitor
```

## Fallback Logging

When the daemon is unavailable, activities are logged to:
- **Default location**: `~/.claude-monitor/activity.log`
- **Custom location**: Via `-log-file` flag or config
- **Format**: JSON lines for easy processing
- **Recovery**: Daemon processes logs when available

### Fallback Log Format

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "event": {
    "id": "activity-uuid",
    "user_id": "username",
    "project_name": "my-project",
    "project_path": "/path/to/project",
    "activity_type": "command",
    "timestamp": "2024-01-15T10:30:00Z"
  },
  "source": "hook_fallback",
  "version": "1.0.0"
}
```

## Integration Examples

### Claude Code Settings

```json
{
  "hooks": {
    "pre_action": "/usr/local/bin/claude-monitor",
    "post_action": "/usr/local/bin/claude-monitor -description 'Action completed'"
  },
  "hook_timeout": 1000
}
```

### Systemd Service (Linux)

```ini
[Unit]
Description=Claude Monitor Hook
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/claude-daemon
User=claude
Restart=always
Environment=CLAUDE_MONITOR_LOG_LEVEL=info

[Install]
WantedBy=multi-user.target
```

### Development Workflow

```bash
# Start daemon in development
./claude-daemon -log-level debug -log-format text

# Test hook manually
./claude-monitor -debug -command "test" -description "Manual test"

# Monitor activity logs
tail -f ~/.claude-monitor/activity.log
```

## Troubleshooting

### Common Issues

1. **Hook not executing**
   ```bash
   # Check Claude Code hook configuration
   # Verify file permissions and path
   chmod +x /path/to/claude-monitor
   ```

2. **Slow execution**
   ```bash
   # Test hook performance
   time ./claude-monitor -debug
   # Should complete in <50ms
   ```

3. **Daemon connection failed**
   ```bash
   # Check daemon status
   curl http://localhost:8080/health
   
   # Check fallback logs
   tail ~/.claude-monitor/activity.log
   ```

4. **Wrong project detection**
   ```bash
   # Check current directory
   ./claude-monitor -debug
   
   # Add custom mapping in config.json
   {
     "project_names": {
       "/full/path/to/project": "Custom Name"
     }
   }
   ```

### Debug Mode

Enable debug mode to see detailed execution information:

```bash
./claude-monitor -debug
```

Debug output includes:
- Execution timing
- Project detection details
- HTTP request/response information
- Fallback logging decisions
- Configuration values used

### Health Checks

```bash
# Check daemon health
curl http://localhost:8080/health

# Check user status
curl "http://localhost:8080/status?user_id=$(whoami)"

# Test hook with daemon
./claude-monitor -debug && echo "Hook executed successfully"
```

## Architecture

The hook integrates with the Claude Monitor system:

```
Claude Code → Hook → Daemon → KuzuDB
             ↓
       Fallback File
```

1. **Claude Code** executes hook before each action
2. **Hook** detects project and creates activity event
3. **Daemon** processes event into sessions and work blocks  
4. **KuzuDB** stores data for analytics and reporting
5. **Fallback** ensures no data loss when daemon unavailable

## Development

### Building from Source

```bash
# Build hook command
go build -o claude-monitor ./cmd/claude-monitor

# Run tests
go test ./cmd/claude-monitor

# Run with coverage
go test -cover ./cmd/claude-monitor

# Build with version info
go build -ldflags "-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o claude-monitor ./cmd/claude-monitor
```

### Testing

```bash
# Unit tests
go test ./cmd/claude-monitor

# Integration tests (requires running daemon)
go test -tags=integration ./cmd/claude-monitor

# Performance tests
go test -run TestPerformanceRequirements ./cmd/claude-monitor

# Test with race detector
go test -race ./cmd/claude-monitor
```

## Security Considerations

- Hook executes with user permissions
- No sensitive data in command line arguments
- Project paths normalized to prevent traversal
- HTTP client uses reasonable timeouts
- Fallback logs written with user permissions only

## Contributing

When modifying the hook:

1. Maintain <50ms execution time requirement
2. Add tests for new project detection methods
3. Ensure cross-platform compatibility
4. Update documentation for new configuration options
5. Test with real Claude Code integration