#!/bin/bash
#
# Claude Monitor - System Daemon Installation Script
# Installs Claude Monitor as a system service with automatic startup
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Get the actual user (not root when using sudo)
get_real_user() {
    if [[ -n "$SUDO_USER" ]]; then
        echo "$SUDO_USER"
    else
        echo "$USER"
    fi
}

# Main installation function
install_claude_monitor() {
    local real_user=$(get_real_user)
    local user_home="/home/$real_user"
    
    log_info "Installing Claude Monitor daemon for user: $real_user"
    
    # 1. Build the latest version
    log_info "Building Claude Monitor..."
    if [[ -f "go.mod" ]]; then
        go build -o bin/claude-monitor ./cmd/claude-monitor
        log_success "Claude Monitor built successfully"
    else
        log_error "go.mod not found. Please run this script from the CC-Monitor directory"
        exit 1
    fi
    
    # 2. Create system directories
    log_info "Creating system directories..."
    mkdir -p /usr/local/bin
    mkdir -p /etc/systemd/system
    
    # 3. Install binary
    log_info "Installing binary to /usr/local/bin..."
    cp bin/claude-monitor /usr/local/bin/
    chmod +x /usr/local/bin/claude-monitor
    log_success "Binary installed"
    
    # 4. Create user configuration directory
    log_info "Setting up user configuration..."
    sudo -u "$real_user" mkdir -p "$user_home/.claude/monitor"
    
    # 5. Create default configuration if it doesn't exist
    if [[ ! -f "$user_home/.claude/monitor/config.json" ]]; then
        log_info "Creating default configuration..."
        sudo -u "$real_user" tee "$user_home/.claude/monitor/config.json" > /dev/null << 'EOF'
{
  "listen_addr": "localhost:9193",
  "database_path": "~/.claude/monitor/monitor.db",
  "log_level": "info",
  "duration_hours": 5,
  "idle_timeout_minutes": 5,
  "web_interface_enabled": true,
  "auto_start_enabled": true
}
EOF
        log_success "Default configuration created"
    else
        log_info "Using existing configuration"
    fi
    
    # 6. Install systemd service (per-user service)
    log_info "Installing systemd service..."
    cp claude-monitor.service /etc/systemd/system/claude-monitor@.service
    systemctl daemon-reload
    log_success "Systemd service installed"
    
    # 7. Enable and start the service for the user
    log_info "Enabling service for user $real_user..."
    systemctl enable claude-monitor@"$real_user".service
    systemctl start claude-monitor@"$real_user".service
    
    # 8. Check service status
    sleep 2
    if systemctl is-active --quiet claude-monitor@"$real_user".service; then
        log_success "Claude Monitor daemon is running!"
    else
        log_warning "Service may not have started correctly. Check status with:"
        echo "    sudo systemctl status claude-monitor@$real_user.service"
    fi
    
    # 9. Display status and usage information
    echo ""
    echo "═══════════════════════════════════════════════════════"
    echo "🎉 Claude Monitor System Daemon Installation Complete"
    echo "═══════════════════════════════════════════════════════"
    echo ""
    echo "Service Status:"
    systemctl status claude-monitor@"$real_user".service --no-pager -l
    echo ""
    echo "Useful Commands:"
    echo "  Start:   sudo systemctl start claude-monitor@$real_user.service"
    echo "  Stop:    sudo systemctl stop claude-monitor@$real_user.service"
    echo "  Restart: sudo systemctl restart claude-monitor@$real_user.service"
    echo "  Status:  sudo systemctl status claude-monitor@$real_user.service"
    echo "  Logs:    sudo journalctl -u claude-monitor@$real_user.service -f"
    echo ""
    echo "Web Interface: http://localhost:9193/"
    echo "Database: $user_home/.claude/monitor/monitor.db"
    echo "Config: $user_home/.claude/monitor/config.json"
    echo ""
    echo "🚀 The daemon will now start automatically on system boot!"
}

# Uninstall function
uninstall_claude_monitor() {
    local real_user=$(get_real_user)
    
    log_info "Uninstalling Claude Monitor daemon for user: $real_user"
    
    # Stop and disable service
    if systemctl is-enabled --quiet claude-monitor@"$real_user".service; then
        systemctl stop claude-monitor@"$real_user".service
        systemctl disable claude-monitor@"$real_user".service
        log_success "Service stopped and disabled"
    fi
    
    # Remove systemd service file
    if [[ -f "/etc/systemd/system/claude-monitor@.service" ]]; then
        rm -f /etc/systemd/system/claude-monitor@.service
        systemctl daemon-reload
        log_success "Systemd service removed"
    fi
    
    # Remove binary
    if [[ -f "/usr/local/bin/claude-monitor" ]]; then
        rm -f /usr/local/bin/claude-monitor
        log_success "Binary removed"
    fi
    
    log_success "Claude Monitor daemon uninstalled"
    log_info "User data preserved in /home/$real_user/.claude/monitor/"
}

# Show usage
show_usage() {
    echo "Claude Monitor - System Daemon Installation"
    echo ""
    echo "Usage: $0 [install|uninstall|status]"
    echo ""
    echo "Commands:"
    echo "  install    - Install Claude Monitor as system daemon"
    echo "  uninstall  - Remove Claude Monitor system daemon"
    echo "  status     - Show daemon status"
    echo ""
    echo "Example:"
    echo "  sudo $0 install"
}

# Show status
show_status() {
    local real_user=$(get_real_user)
    
    echo "Claude Monitor System Daemon Status"
    echo "═══════════════════════════════════════"
    
    if systemctl is-active --quiet claude-monitor@"$real_user".service; then
        log_success "Daemon is running"
    else
        log_warning "Daemon is not running"
    fi
    
    echo ""
    systemctl status claude-monitor@"$real_user".service --no-pager -l
}

# Main script logic
case "${1:-install}" in
    "install")
        check_root
        install_claude_monitor
        ;;
    "uninstall")
        check_root
        uninstall_claude_monitor
        ;;
    "status")
        show_status
        ;;
    *)
        show_usage
        exit 1
        ;;
esac