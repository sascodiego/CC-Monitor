# Claude Monitor - Installation Guide

[![Alpha Version](https://img.shields.io/badge/Status-Alpha-orange?style=flat-square)](https://github.com/sascodiego/CC-Monitor)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue?style=flat-square)](https://golang.org/)

## 🚀 One-Command Installation

### Zero-Configuration Setup ⚡

```bash
# Clone, build, and install everything automatically
git clone https://github.com/sascodiego/CC-Monitor.git
cd CC-Monitor
make build
sudo ./bin/claude-monitor-basic install
```

**That's it!** The system is now fully installed and running. 🎉

### What Happens Automatically:
1. ✅ **System directories created** (`/etc/claude-monitor`, `/var/lib/claude-monitor`, `/var/log/claude-monitor`)
2. ✅ **Configuration generated** with sensible defaults
3. ✅ **Binaries installed** to `/usr/local/bin`
4. ✅ **Systemd service** created and enabled
5. ✅ **Database initialized** and ready
6. ✅ **Monitoring started** automatically
7. ✅ **Convenience scripts** created (`claude-monitor-start`, `claude-monitor-stop`)

### Prerequisites

- **Linux System** (Ubuntu 20.04+, Debian 11+, CentOS 8+, or WSL2)
- **Go 1.21+** ([Install Go](https://golang.org/doc/install))
- **sudo access** (for system installation)

### Alternative Installation Options

#### User-Only Installation (No System Service)

```bash
# Install for current user only
./bin/claude-monitor-basic install --user

# Installs to:
# - Binaries: ~/.local/bin/
# - Config: ~/.config/claude-monitor/
# - Data: ~/.local/share/claude-monitor/
```

#### Custom Installation Directory

```bash
# Install to custom location
sudo ./bin/claude-monitor-basic install --dir=/opt/claude-monitor
```

#### Force Reinstall

```bash
# Reinstall even if already installed
sudo ./bin/claude-monitor-basic install --force
```

#### Option 2: Quick Build

```bash
# Clone and build in one step
git clone https://github.com/sascodiego/CC-Monitor.git
cd CC-Monitor
go build -o bin/claude-monitor-basic ./cmd/claude-monitor-basic
go build -o bin/claude-daemon-enhanced ./cmd/claude-daemon-enhanced
```

#### Option 3: Development Build

```bash
# Full development environment with eBPF support
git clone https://github.com/sascodiego/CC-Monitor.git
cd CC-Monitor

# Install development dependencies (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install -y build-essential clang llvm

# Build with eBPF support
make build-ebpf

# Run tests
make test
```

## 🔧 System Setup

### 1. Directory Structure

Claude Monitor uses these directories:

```
/etc/claude-monitor/          # Configuration files
/var/lib/claude-monitor/      # Database and data storage
/var/log/claude-monitor/      # Log files
/var/run/claude-monitor.pid   # Daemon PID file
/tmp/claude-monitor-status.json # Real-time status
```

### 2. Create System Directories

```bash
# Create required directories
sudo mkdir -p /etc/claude-monitor
sudo mkdir -p /var/lib/claude-monitor
sudo mkdir -p /var/log/claude-monitor

# Set permissions
sudo chown -R $USER:$USER /var/lib/claude-monitor
sudo chmod 755 /var/lib/claude-monitor
```

### 3. Configuration

Create basic configuration file:

```bash
sudo tee /etc/claude-monitor/config.yaml > /dev/null <<EOF
# Claude Monitor Configuration
daemon:
  log_level: INFO
  database_path: /var/lib/claude-monitor/claude.db
  status_file: /tmp/claude-monitor-status.json
  pid_file: /var/run/claude-monitor.pid

monitoring:
  session_duration: 5h        # 5-hour session windows
  inactivity_timeout: 5m      # 5-minute work block timeout
  update_interval: 5s         # Status update frequency

work_hours:
  daily_target: 8h            # Daily work hour goal
  weekly_target: 40h          # Weekly work hour goal
  overtime_threshold: 8h      # Daily overtime threshold
  rounding_method: nearest    # Time rounding (nearest, up, down)
  rounding_interval: 15m      # Rounding interval (15m, 30m, 1h)
EOF
```

## 🚀 First Run

### 1. Start the Enhanced Daemon

The enhanced daemon provides the most accurate monitoring:

```bash
# Start the enhanced daemon (requires sudo)
sudo ./bin/claude-daemon-enhanced

# Or run in background
sudo nohup ./bin/claude-daemon-enhanced > /var/log/claude-monitor/daemon.log 2>&1 &
```

### 2. Verify Installation

Check that the system is running:

```bash
# Check daemon status
./bin/claude-monitor-basic status

# Expected output:
# 🚀 Claude Monitor System Status
# ═══════════════════════════════════
# ✅ Enhanced Daemon: RUNNING
# 📊 Current Activity: X processes, Y connections
# ...
```

### 3. Test Work Hour Tracking

Start using Claude and check work tracking:

```bash
# Check current work day
./bin/claude-monitor-basic workhour workday status

# Expected output:
# 📊 Work Day Status - January 15, 2024
# ═══════════════════════════════════════════════
# ⏰ Work Period: 09:15 AM → ACTIVE (2h 30m)
# 📈 Productivity: 87% efficiency
# ...
```

## ⚙️ Service Installation (Optional)

### Create Systemd Service

For production use, create a systemd service:

```bash
sudo tee /etc/systemd/system/claude-monitor.service > /dev/null <<EOF
[Unit]
Description=Claude Monitor Enhanced Daemon
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/claude-daemon-enhanced
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable claude-monitor
sudo systemctl start claude-monitor

# Check service status
sudo systemctl status claude-monitor
```

## 🔍 Verification

### 1. Health Check

Run the health check script:

```bash
#!/bin/bash
echo "🏥 Claude Monitor Health Check"
echo "═══════════════════════════════"

# Check binaries
if [ -f "./bin/claude-daemon-enhanced" ]; then
    echo "✅ Enhanced daemon binary: OK"
else
    echo "❌ Enhanced daemon binary: MISSING"
fi

if [ -f "./bin/claude-monitor-basic" ]; then
    echo "✅ CLI binary: OK"
else
    echo "❌ CLI binary: MISSING"
fi

# Check daemon running
if pgrep -f claude-daemon > /dev/null; then
    echo "✅ Daemon process: RUNNING"
else
    echo "❌ Daemon process: NOT RUNNING"
fi

# Check status file
if [ -f "/tmp/claude-monitor-status.json" ]; then
    echo "✅ Status file: OK"
    PROCESSES=$(jq -r '.activityHistory.currentProcesses' /tmp/claude-monitor-status.json 2>/dev/null || echo "0")
    echo "📊 Claude processes: $PROCESSES"
else
    echo "❌ Status file: MISSING"
fi

# Check directories
for dir in "/var/lib/claude-monitor" "/var/log/claude-monitor"; do
    if [ -d "$dir" ]; then
        echo "✅ Directory $dir: OK"
    else
        echo "❌ Directory $dir: MISSING"
    fi
done
```

### 2. Test All Features

```bash
# Test basic commands
./bin/claude-monitor-basic status
./bin/claude-monitor-basic workhour workday status

# Start using Claude to generate activity
# The system should automatically detect sessions and work blocks
```

## 🔧 Troubleshooting

### Common Issues

#### 1. Permission Denied
```bash
# Problem: Permission denied when starting daemon
sudo chown root:root ./bin/claude-daemon-enhanced
sudo chmod +x ./bin/claude-daemon-enhanced
```

#### 2. Port Already in Use
```bash
# Problem: Address already in use
sudo pkill -f claude-daemon
# Wait 5 seconds, then restart
```

#### 3. Status File Not Found
```bash
# Problem: Cannot read status file
sudo chmod 644 /tmp/claude-monitor-status.json
# Or restart daemon to recreate
```

#### 4. Build Errors
```bash
# Problem: Compilation errors
go mod tidy
go mod download
make clean
make build
```

### Log Files

Check logs for debugging:

```bash
# Daemon logs (if using systemd)
sudo journalctl -u claude-monitor -f

# Manual daemon logs
tail -f /var/log/claude-monitor/daemon.log

# System logs
dmesg | grep claude
```

### Reset Installation

Complete reset if needed:

```bash
# Stop all processes
sudo pkill -f claude-daemon
sudo systemctl stop claude-monitor 2>/dev/null

# Remove files
sudo rm -rf /var/lib/claude-monitor
sudo rm -rf /var/log/claude-monitor
sudo rm -f /tmp/claude-monitor-status.json
sudo rm -f /var/run/claude-monitor.pid

# Rebuild
make clean
make build
```

## 📋 Next Steps

After successful installation:

1. **Read the User Guide**: Check `USER_GUIDE.md` for detailed usage
2. **Configure Goals**: Set your daily/weekly work hour targets
3. **Explore CLI**: Try all available commands
4. **Generate Reports**: Start creating work hour reports
5. **Set up Automation**: Consider systemd service for production

## 🆘 Support

If you encounter issues:

1. **Check Prerequisites**: Ensure all requirements are met
2. **Review Logs**: Check daemon and system logs
3. **Run Health Check**: Use the verification script above
4. **GitHub Issues**: Report bugs at [Issues](https://github.com/sascodiego/CC-Monitor/issues)
5. **Documentation**: Read the complete documentation in `docs/`

## ⚡ Performance Tips

1. **SSD Storage**: Use SSD for database storage
2. **Memory**: Allocate at least 1GB RAM for large datasets
3. **Clean Data**: Regularly clean old data using retention policies
4. **Monitor Resources**: Watch CPU/memory usage during heavy analysis

---

**🎉 Congratulations! Claude Monitor is now installed and ready to track your productivity!**