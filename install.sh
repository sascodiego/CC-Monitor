#!/bin/bash

# Claude Monitor - One-Click Installation Script
# This script automatically sets up Claude Monitor with zero configuration required

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/claude-monitor"
DATA_DIR="/var/lib/claude-monitor"
LOG_DIR="/var/log/claude-monitor"
SERVICE_NAME="claude-monitor"

echo -e "${BLUE}🚀 Claude Monitor - One-Click Installation${NC}"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo ""

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}❌ This script must be run as root${NC}"
   echo "Please run: sudo ./install.sh"
   exit 1
fi

# Detect system
if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS=$NAME
    VER=$VERSION_ID
else
    echo -e "${RED}❌ Cannot detect operating system${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Detected OS: $OS $VER${NC}"

# Check Go installation
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    echo "Please install Go 1.21+ first: https://golang.org/doc/install"
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
echo -e "${GREEN}✅ Go version: $GO_VERSION${NC}"

echo ""
echo -e "${YELLOW}📦 Installing Claude Monitor...${NC}"

# Step 1: Create all required directories
echo -e "${BLUE}1. Creating system directories...${NC}"
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"
mkdir -p "$LOG_DIR"
mkdir -p "$INSTALL_DIR"

# Set proper permissions
chown -R root:root "$CONFIG_DIR"
chown -R $SUDO_USER:$SUDO_USER "$DATA_DIR" 2>/dev/null || chown -R root:root "$DATA_DIR"
chown -R root:root "$LOG_DIR"
chmod 755 "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"

echo -e "${GREEN}   ✅ Directories created${NC}"

# Step 2: Build binaries
echo -e "${BLUE}2. Building Claude Monitor binaries...${NC}"
echo "   Building enhanced daemon..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o "$INSTALL_DIR/claude-daemon-enhanced" ./cmd/claude-daemon-enhanced
chmod +x "$INSTALL_DIR/claude-daemon-enhanced"

echo "   Building CLI interface..."
go build -ldflags="-s -w" -o "$INSTALL_DIR/claude-monitor" ./cmd/claude-monitor-basic
chmod +x "$INSTALL_DIR/claude-monitor"

echo "   Building simple daemon (backup)..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o "$INSTALL_DIR/claude-daemon-simple" ./cmd/claude-daemon-simple
chmod +x "$INSTALL_DIR/claude-daemon-simple"

echo -e "${GREEN}   ✅ Binaries built and installed${NC}"

# Step 3: Create default configuration
echo -e "${BLUE}3. Creating default configuration...${NC}"
cat > "$CONFIG_DIR/config.yaml" << EOF
# Claude Monitor Configuration - Auto-generated
daemon:
  log_level: INFO
  database_path: $DATA_DIR/claude.db
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

export:
  default_format: json        # Default export format
  include_charts: true        # Include charts in reports
  template: professional     # Report template
EOF

chmod 644 "$CONFIG_DIR/config.yaml"
echo -e "${GREEN}   ✅ Configuration created${NC}"

# Step 4: Create systemd service
echo -e "${BLUE}4. Setting up system service...${NC}"
cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOF
[Unit]
Description=Claude Monitor Enhanced Daemon
Documentation=https://github.com/sascodiego/CC-Monitor
After=network.target
Wants=network.target

[Service]
Type=simple
User=root
Group=root
ExecStart=$INSTALL_DIR/claude-daemon-enhanced
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=claude-monitor

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR $LOG_DIR /tmp
PrivateTmp=false

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd and enable service
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"

echo -e "${GREEN}   ✅ System service configured${NC}"

# Step 5: Create convenience scripts
echo -e "${BLUE}5. Creating convenience scripts...${NC}"

# Start script
cat > "$INSTALL_DIR/claude-monitor-start" << 'EOF'
#!/bin/bash
echo "🚀 Starting Claude Monitor..."
sudo systemctl start claude-monitor
sleep 2
if systemctl is-active --quiet claude-monitor; then
    echo "✅ Claude Monitor started successfully"
    claude-monitor status
else
    echo "❌ Failed to start Claude Monitor"
    sudo systemctl status claude-monitor
fi
EOF

# Stop script
cat > "$INSTALL_DIR/claude-monitor-stop" << 'EOF'
#!/bin/bash
echo "🛑 Stopping Claude Monitor..."
sudo systemctl stop claude-monitor
echo "✅ Claude Monitor stopped"
EOF

# Status script
cat > "$INSTALL_DIR/claude-monitor-status" << 'EOF'
#!/bin/bash
if systemctl is-active --quiet claude-monitor; then
    claude-monitor status
else
    echo "❌ Claude Monitor is not running"
    echo "Start with: claude-monitor-start"
fi
EOF

chmod +x "$INSTALL_DIR/claude-monitor-start"
chmod +x "$INSTALL_DIR/claude-monitor-stop"
chmod +x "$INSTALL_DIR/claude-monitor-status"

echo -e "${GREEN}   ✅ Convenience scripts created${NC}"

# Step 6: Initialize database and test
echo -e "${BLUE}6. Initializing system...${NC}"

# Create empty database file with proper permissions
touch "$DATA_DIR/claude.db"
chown $SUDO_USER:$SUDO_USER "$DATA_DIR/claude.db" 2>/dev/null || chown root:root "$DATA_DIR/claude.db"
chmod 644 "$DATA_DIR/claude.db"

# Test CLI
echo "   Testing CLI interface..."
if "$INSTALL_DIR/claude-monitor" --help > /dev/null 2>&1; then
    echo -e "${GREEN}   ✅ CLI interface working${NC}"
else
    echo -e "${YELLOW}   ⚠️  CLI may need dependencies${NC}"
fi

echo -e "${GREEN}   ✅ System initialized${NC}"

# Step 7: Final setup and start
echo -e "${BLUE}7. Starting Claude Monitor service...${NC}"

# Start the service
systemctl start "$SERVICE_NAME"
sleep 3

# Check if service started successfully
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo -e "${GREEN}   ✅ Service started successfully${NC}"
else
    echo -e "${YELLOW}   ⚠️  Service may need manual start${NC}"
fi

echo ""
echo -e "${GREEN}🎉 INSTALLATION COMPLETE!${NC}"
echo -e "${GREEN}═══════════════════════════${NC}"
echo ""
echo -e "${BLUE}📋 Quick Start:${NC}"
echo ""
echo -e "${YELLOW}  # Check status${NC}"
echo "  claude-monitor status"
echo ""
echo -e "${YELLOW}  # View work day${NC}"
echo "  claude-monitor workhour workday status"
echo ""
echo -e "${YELLOW}  # Control service${NC}"
echo "  claude-monitor-start    # Start monitoring"
echo "  claude-monitor-stop     # Stop monitoring"
echo "  claude-monitor-status   # Quick status check"
echo ""
echo -e "${BLUE}📁 Important Paths:${NC}"
echo "  Binaries: $INSTALL_DIR/claude-*"
echo "  Config:   $CONFIG_DIR/config.yaml"
echo "  Data:     $DATA_DIR/"
echo "  Logs:     $LOG_DIR/ (or journalctl -u claude-monitor)"
echo ""
echo -e "${BLUE}🔧 Service Management:${NC}"
echo "  sudo systemctl start claude-monitor    # Start"
echo "  sudo systemctl stop claude-monitor     # Stop"
echo "  sudo systemctl restart claude-monitor  # Restart"
echo "  sudo systemctl status claude-monitor   # Status"
echo "  journalctl -u claude-monitor -f        # View logs"
echo ""
echo -e "${GREEN}✨ Claude Monitor is now ready to track your productivity!${NC}"
echo -e "${GREEN}Just start using Claude CLI and the system will automatically begin monitoring.${NC}"
echo ""

# Final status check
echo -e "${BLUE}📊 Current Status:${NC}"
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo -e "${GREEN}✅ Service: RUNNING${NC}"
    # Give the daemon a moment to initialize
    sleep 2
    if [[ -f "/tmp/claude-monitor-status.json" ]]; then
        echo -e "${GREEN}✅ Status file: CREATED${NC}"
        # Try to show current status
        if "$INSTALL_DIR/claude-monitor" status > /dev/null 2>&1; then
            echo -e "${GREEN}✅ Monitoring: ACTIVE${NC}"
            echo ""
            echo -e "${YELLOW}Current system status:${NC}"
            "$INSTALL_DIR/claude-monitor" status 2>/dev/null || echo "Status will be available once Claude processes are detected"
        fi
    else
        echo -e "${YELLOW}⚠️  Status file: Will be created on first activity${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Service: Not running (may need manual start)${NC}"
    echo "   Try: sudo systemctl start claude-monitor"
fi
echo ""
echo -e "${BLUE}🚀 Installation completed successfully!${NC}"
echo -e "${BLUE}Start using Claude CLI to begin automatic time tracking.${NC}"