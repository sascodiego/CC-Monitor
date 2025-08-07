# Claude Code Integration Guide

## Automated AI-Optimized Configuration

To configure Claude Code for work hour tracking, copy and paste the following instructions to Claude:

### Configuration Prompt for AI Assistant

```
I need to configure Claude Code hooks for automatic work hour tracking. Please help me set up the following configuration:

**Objective**: Add a pre-action hook that calls `claude-monitor hook` before each Claude action to track work hours automatically.

**Platform**: [Specify your platform: Windows, macOS, Linux]

**Claude Code Configuration**: 
Add this to your Claude Code settings (usually in ~/.claude-code/config.json or equivalent):

```json
{
  "hooks": {
    "pre_action": "/usr/local/bin/claude-monitor hook"
  }
}
```

**Requirements**:
1. The hook must execute in <50ms to not impact Claude Code performance
2. It should capture project information from the current working directory
3. Failed hook calls should not prevent Claude Code from working
4. The daemon should be running on localhost:8080

**Installation Steps**:
1. Install claude-monitor: `claude-monitor install`
2. Start daemon: `claude-monitor daemon &`
3. Add hook configuration to Claude Code
4. Verify: `claude-monitor status`

**Troubleshooting**:
- Test hook directly: `claude-monitor hook --debug`
- Check daemon health: `curl localhost:8080/health`
- View logs: `claude-monitor daemon --log-level=debug`

Please provide platform-specific instructions and help me verify the configuration is working correctly.
```

### Manual Configuration Steps

#### 1. Install Claude Monitor
```bash
# Download and install the binary
curl -L https://github.com/claude-monitor/releases/latest/download/claude-monitor-$(uname -s)-$(uname -m) -o claude-monitor
chmod +x claude-monitor
sudo mv claude-monitor /usr/local/bin/
```

#### 2. Initialize System
```bash
# Self-install and create configuration
claude-monitor install

# Start the daemon
claude-monitor daemon &

# Verify installation
claude-monitor status
```

#### 3. Configure Claude Code Hooks

**Linux/macOS** - Add to `~/.claude-code/config.json`:
```json
{
  "hooks": {
    "pre_action": "/usr/local/bin/claude-monitor hook"
  }
}
```

**Windows** - Add to `%USERPROFILE%\.claude-code\config.json`:
```json
{
  "hooks": {
    "pre_action": "C:\\Program Files\\claude-monitor\\claude-monitor.exe hook"
  }
}
```

#### 4. Test Configuration
```bash
# Test the hook directly
claude-monitor hook --debug

# Check daemon connectivity
curl http://localhost:8080/health

# View today's activity
claude-monitor today
```

### Advanced Configuration

#### Custom Project Names
Edit `~/.claude-monitor/config.json`:
```json
{
  "projects": {
    "custom_names": {
      "/home/user/work/complex-app": "MyMainProject",
      "/home/user/experiments": "Learning"
    }
  }
}
```

#### Hook Performance Tuning
```json
{
  "hook": {
    "timeout_ms": 25,
    "ignore_patterns": [
      "*/node_modules/*",
      "*/target/*", 
      "*/.git/*"
    ]
  }
}
```

#### Reporting Customization
```json
{
  "reporting": {
    "default_output_format": "json",
    "enable_colors": false,
    "time_format": "15:04:05"
  }
}
```

### Verification Commands

```bash
# Check system status
claude-monitor status

# View today's work
claude-monitor today

# View this week's summary  
claude-monitor week

# Debug hook execution
claude-monitor hook --debug --verbose

# Test daemon health
curl http://localhost:8080/health | jq
```

### Common Issues and Solutions

1. **Hook timeout**: Reduce timeout_ms to 25ms
2. **Permission denied**: Ensure claude-monitor is executable
3. **Daemon not responding**: Check if port 8080 is available
4. **No project detected**: Verify working directory has project indicators
5. **Missing activities**: Check ~/.claude-monitor/activity.log for fallback entries

### Platform-Specific Notes

#### Windows (WSL)
- Place binary in `/usr/local/bin/` inside WSL
- Configure Claude Code inside WSL environment
- Daemon runs on WSL localhost:8080

#### macOS
- Binary works with Apple Silicon and Intel
- May need to bypass Gatekeeper: `xattr -d com.apple.quarantine claude-monitor`
- Use `brew` for installation if available

#### Linux
- Standard installation to `/usr/local/bin/`
- May need `sudo` for installation
- SystemD service available with `claude-monitor install --service`