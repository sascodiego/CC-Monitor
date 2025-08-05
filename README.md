# Claude Monitor v1.0.0 - Work Hour Tracking System

**🎯 Automatically track your Claude CLI usage sessions and work hours**

Claude Monitor is a production-ready system that tracks your Claude CLI sessions with 5-hour session windows and 5-minute activity timeouts, storing all data persistently in a SQLite database.

## ✨ Features

- **📊 Automatic Session Tracking**: 5-hour windows from first Claude interaction
- **⏱️ Work Block Detection**: 5-minute inactivity timeout for precise work measurement  
- **💾 Persistent Database**: SQLite storage that survives system reboots
- **🔄 System Service**: Runs automatically as background daemon
- **📈 Real-time Status**: Live monitoring of current sessions and work blocks
- **🛠️ Single Command Install**: Complete system setup with one command

## 🚀 Quick Installation

### Prerequisites
- Go 1.21 or later
- Linux with systemd
- sudo privileges

### Install Claude Monitor

```bash
# 1. Build the CLI
CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/claude-monitor ./cmd/claude-monitor

# 2. Install the complete system (requires sudo password)
sudo -E ./bin/claude-monitor install
```

This single command will:
- ✅ Build the complete daemon with database persistence
- ✅ Install CLI to `/usr/local/bin/claude-monitor`
- ✅ Install daemon to `/usr/local/bin/claude-daemon-complete`
- ✅ Create systemd service for automatic startup
- ✅ Initialize database and directories
- ✅ Start monitoring immediately

## 📊 Usage

### Check Status
```bash
claude-monitor status
```
Shows current session, work blocks, and database information.

### Generate Reports
```bash
claude-monitor report
```
View collected data and database statistics.

### Export Data
```bash
claude-monitor export
```
Information about accessing your persistent work data.

### Service Management
```bash
claude-monitor start     # Start the service
claude-monitor stop      # Stop the service  
claude-monitor restart   # Restart the service
```

## 📁 System Architecture

```
Claude Monitor System
├── claude-monitor                    # Production CLI
├── claude-daemon-complete           # Enhanced daemon with persistence
├── /var/lib/claude-monitor/         # Database storage
│   └── claude.db                    # SQLite database (persistent)
├── /tmp/claude-monitor-status.json  # Real-time status
└── systemd service                  # Auto-start daemon
```

## 🗄️ Database Schema

The system uses SQLite with the following tables:

```sql
-- Sessions: 5-hour windows
CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    is_active BOOLEAN NOT NULL
);

-- Work Blocks: Activity periods within sessions
CREATE TABLE work_blocks (
    block_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    duration_seconds INTEGER DEFAULT 0,
    is_active BOOLEAN NOT NULL
);

-- Work Days: Daily summaries
CREATE TABLE work_days (
    date TEXT PRIMARY KEY,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    total_seconds INTEGER DEFAULT 0,
    session_count INTEGER DEFAULT 0,
    block_count INTEGER DEFAULT 0,
    efficiency REAL DEFAULT 0.0
);
```

## 🔍 Status Example

```bash
$ claude-monitor status

📊 Claude Monitor Status
========================
✅ Service: Running
✅ Daemon: Active
📡 Monitoring: Active

📅 Current Session:
   ID: 3df8d624...
   Started: 00:03:08
   Ends: 05:03:08

🟢 Status: Active

💾 Database: /var/lib/claude-monitor/claude.db (40.0 KB)
```

## 🏗️ Development Architecture

This system is built with specialized components:

- **Enhanced Daemon**: Real-time monitoring with database persistence
- **Simple CLI**: Production interface without complex dependencies
- **SQLite Database**: Persistent storage with proper schema
- **Systemd Integration**: Professional service management

## 🔧 Manual Database Access

You can directly query the SQLite database:

```bash
sqlite3 /var/lib/claude-monitor/claude.db

.tables
SELECT * FROM sessions ORDER BY start_time DESC LIMIT 5;
SELECT * FROM work_blocks ORDER BY start_time DESC LIMIT 10;
SELECT date, total_seconds/3600.0 as hours FROM work_days ORDER BY date DESC;
```

## 🚨 Troubleshooting

### Service not starting?
```bash
sudo systemctl status claude-monitor
sudo journalctl -u claude-monitor -f
```

### Database issues?
```bash
ls -la /var/lib/claude-monitor/
sqlite3 /var/lib/claude-monitor/claude.db ".tables"
```

### Permission problems?
```bash
sudo chown -R $USER:$USER /var/lib/claude-monitor
```

## 📈 Performance

- **Memory Usage**: ~10MB daemon footprint
- **CPU Usage**: Minimal (monitoring only)
- **Database Size**: Grows slowly (~1KB per day typical usage)
- **Status Updates**: Every 5 seconds

## 🛡️ Security

- **No sensitive data**: Only process names and timestamps stored
- **Local storage**: All data stays on your machine
- **Minimal privileges**: Daemon runs as user, not root
- **SQLite**: Simple, file-based database with no network exposure

## 📋 System Requirements

- **OS**: Linux with systemd
- **Go**: 1.21+ (for building)
- **SQLite**: CGO enabled (included in build)
- **Disk**: ~1MB for binaries, minimal for data
- **Memory**: ~10MB for daemon

## 🔄 Data Persistence

**✅ Your data survives:**
- System reboots
- Service restarts  
- Daemon crashes
- System updates

All session and work block data is immediately written to the SQLite database for maximum reliability.

---

**Built with Go, eBPF, and SQLite for maximum performance and reliability.**