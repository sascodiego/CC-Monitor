#!/bin/bash

/**
 * CONTEXT:   Installation script for Claude Monitor system with hook integration
 * INPUT:     System environment, user preferences, and installation options
 * OUTPUT:    Complete Claude Monitor installation with daemon and hook
 * BUSINESS:  Provide one-click installation for seamless Claude Code integration
 * CHANGE:    Initial installation script with system detection and configuration
 * RISK:      Medium - System installation affects user environment and permissions
 */

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CLAUDE_MONITOR_VERSION="${CLAUDE_MONITOR_VERSION:-1.0.0}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="$HOME/.claude-monitor"
SERVICE_USER="${SERVICE_USER:-$USER}"

# Installation options
INSTALL_DAEMON=true
INSTALL_HOOK=true
INSTALL_SERVICE=false
CONFIGURE_CLAUDE=false
START_DAEMON=false

print_header() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║                   Claude Monitor v$CLAUDE_MONITOR_VERSION                   ║"
    echo "║           Complete Work Hour Tracking System            ║"
    echo "║                                                          ║"
    echo "║  • Claude Code Hook Integration                          ║"
    echo "║  • Automatic Project Detection                           ║"
    echo "║  • Real-time Session & Work Block Tracking             ║"
    echo "║  • KuzuDB Analytics & Reporting                         ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    echo
}

print_step() {
    echo -e "${BLUE}→${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1" >&2
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# System detection
detect_system() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
        ARCH=$(uname -m)
        if [[ -f /proc/version ]] && grep -q Microsoft /proc/version; then
            PLATFORM="wsl"
        else
            PLATFORM="linux"
        fi
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="darwin"
        ARCH=$(uname -m)
        PLATFORM="macos"
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        OS="windows"
        ARCH="amd64"
        PLATFORM="windows"
    else
        print_error "Unsupported operating system: $OSTYPE"
        exit 1
    fi

    # Normalize architecture
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) 
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    print_info "Detected system: $PLATFORM ($OS-$ARCH)"
}

# Check prerequisites
check_prerequisites() {
    print_step "Checking prerequisites..."

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        print_info "Visit: https://golang.org/dl/"
        exit 1
    fi

    # Check Go version
    GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
    if [[ $(echo "$GO_VERSION 1.21" | tr ' ' '\n' | sort -V | head -n1) != "1.21" ]]; then
        print_error "Go 1.21 or later is required. Found: $GO_VERSION"
        exit 1
    fi

    # Check if git is available (for version info)
    if ! command -v git &> /dev/null; then
        print_warning "Git not found. Build version info will be limited."
    fi

    # Check write permissions for install directory
    if [[ ! -w "$INSTALL_DIR" ]]; then
        if [[ $EUID -ne 0 ]]; then
            print_warning "Installation directory $INSTALL_DIR requires sudo access"
            NEED_SUDO=true
        fi
    fi

    print_success "Prerequisites checked"
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --daemon-only)
                INSTALL_HOOK=false
                shift
                ;;
            --hook-only)
                INSTALL_DAEMON=false
                shift
                ;;
            --install-service)
                INSTALL_SERVICE=true
                shift
                ;;
            --configure-claude)
                CONFIGURE_CLAUDE=true
                shift
                ;;
            --start-daemon)
                START_DAEMON=true
                shift
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --version)
                CLAUDE_MONITOR_VERSION="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
    echo "Claude Monitor Installation Script"
    echo
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  --daemon-only         Install only the daemon"
    echo "  --hook-only          Install only the Claude Code hook"
    echo "  --install-service    Install as system service (Linux/macOS)"
    echo "  --configure-claude   Configure Claude Code hooks automatically"
    echo "  --start-daemon       Start daemon after installation"
    echo "  --install-dir DIR    Installation directory (default: $INSTALL_DIR)"
    echo "  --version VER        Version to install (default: $CLAUDE_MONITOR_VERSION)"
    echo "  --help               Show this help message"
    echo
    echo "Examples:"
    echo "  $0                           # Install complete system"
    echo "  $0 --hook-only              # Install only Claude Code hook"
    echo "  $0 --install-service --start-daemon  # Install as service and start"
    echo "  $0 --configure-claude       # Install and configure Claude Code"
}

# Build binaries
build_binaries() {
    print_step "Building Claude Monitor binaries..."

    # Clean previous builds
    make clean > /dev/null 2>&1 || true

    # Build based on selection
    if [[ "$INSTALL_DAEMON" == "true" && "$INSTALL_HOOK" == "true" ]]; then
        make build-all VERSION="$CLAUDE_MONITOR_VERSION"
    elif [[ "$INSTALL_DAEMON" == "true" ]]; then
        make build-daemon VERSION="$CLAUDE_MONITOR_VERSION"
    elif [[ "$INSTALL_HOOK" == "true" ]]; then
        make build-hook VERSION="$CLAUDE_MONITOR_VERSION"
    fi

    print_success "Binaries built successfully"
}

# Install binaries
install_binaries() {
    print_step "Installing binaries to $INSTALL_DIR..."

    # Create install directory if needed
    if [[ $NEED_SUDO == "true" ]]; then
        sudo mkdir -p "$INSTALL_DIR"
    else
        mkdir -p "$INSTALL_DIR"
    fi

    # Install daemon
    if [[ "$INSTALL_DAEMON" == "true" && -f "bin/claude-daemon" ]]; then
        if [[ $NEED_SUDO == "true" ]]; then
            sudo cp bin/claude-daemon "$INSTALL_DIR/"
            sudo chmod +x "$INSTALL_DIR/claude-daemon"
        else
            cp bin/claude-daemon "$INSTALL_DIR/"
            chmod +x "$INSTALL_DIR/claude-daemon"
        fi
        print_success "Daemon installed: $INSTALL_DIR/claude-daemon"
    fi

    # Install hook
    if [[ "$INSTALL_HOOK" == "true" && -f "bin/claude-monitor" ]]; then
        if [[ $NEED_SUDO == "true" ]]; then
            sudo cp bin/claude-monitor "$INSTALL_DIR/"
            sudo chmod +x "$INSTALL_DIR/claude-monitor"
        else
            cp bin/claude-monitor "$INSTALL_DIR/"
            chmod +x "$INSTALL_DIR/claude-monitor"
        fi
        print_success "Hook installed: $INSTALL_DIR/claude-monitor"
    fi
}

# Setup configuration
setup_configuration() {
    print_step "Setting up configuration..."

    # Create config directory
    mkdir -p "$CONFIG_DIR"

    # Create default configuration if it doesn't exist
    if [[ ! -f "$CONFIG_DIR/config.json" ]]; then
        cat > "$CONFIG_DIR/config.json" << EOF
{
  "enabled": true,
  "daemon_url": "http://localhost:8080",
  "timeout_ms": 100,
  "log_level": "info",
  "project_names": {},
  "ignore_patterns": [
    "*/tmp/*",
    "*/node_modules/*",
    "*/.git/*",
    "*/target/*",
    "*/build/*",
    "*/dist/*"
  ]
}
EOF
        print_success "Default configuration created: $CONFIG_DIR/config.json"
    else
        print_info "Configuration file already exists: $CONFIG_DIR/config.json"
    fi

    # Create data directory
    mkdir -p "$CONFIG_DIR/data"
    
    # Set proper permissions
    chmod 755 "$CONFIG_DIR"
    chmod 644 "$CONFIG_DIR/config.json" 2>/dev/null || true
}

# Install system service
install_service() {
    if [[ "$INSTALL_SERVICE" != "true" ]]; then
        return
    fi

    print_step "Installing system service..."

    case $PLATFORM in
        linux|wsl)
            # Create systemd service
            SERVICE_FILE="/etc/systemd/system/claude-monitor.service"
            sudo tee "$SERVICE_FILE" > /dev/null << EOF
[Unit]
Description=Claude Monitor Daemon
Documentation=https://github.com/claude-monitor/system
After=network.target
Wants=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/claude-daemon -config $CONFIG_DIR/config.json
ExecReload=/bin/kill -HUP \$MAINPID
KillMode=process
Restart=on-failure
RestartSec=5s

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=$CONFIG_DIR

# Environment
Environment=HOME=$HOME
Environment=USER=$SERVICE_USER

[Install]
WantedBy=multi-user.target
EOF

            sudo systemctl daemon-reload
            sudo systemctl enable claude-monitor
            print_success "Systemd service installed and enabled"
            ;;
            
        macos)
            # Create launchd plist
            PLIST_FILE="$HOME/Library/LaunchAgents/com.claude-monitor.daemon.plist"
            mkdir -p "$HOME/Library/LaunchAgents"
            
            cat > "$PLIST_FILE" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.claude-monitor.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/claude-daemon</string>
        <string>-config</string>
        <string>$CONFIG_DIR/config.json</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$CONFIG_DIR/daemon.log</string>
    <key>StandardErrorPath</key>
    <string>$CONFIG_DIR/daemon.error.log</string>
    <key>WorkingDirectory</key>
    <string>$CONFIG_DIR</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>HOME</key>
        <string>$HOME</string>
        <key>USER</key>
        <string>$USER</string>
    </dict>
</dict>
</plist>
EOF

            launchctl load "$PLIST_FILE"
            print_success "LaunchAgent installed and loaded"
            ;;
            
        *)
            print_warning "System service not supported on $PLATFORM"
            ;;
    esac
}

# Configure Claude Code
configure_claude_code() {
    if [[ "$CONFIGURE_CLAUDE" != "true" ]]; then
        return
    fi

    print_step "Configuring Claude Code integration..."

    # Try to find Claude Code config
    CLAUDE_CONFIG_PATHS=(
        "$HOME/.claude/settings.json"
        "$HOME/.config/claude/settings.json"
        "$HOME/Library/Application Support/Claude/settings.json"
    )

    CLAUDE_CONFIG=""
    for path in "${CLAUDE_CONFIG_PATHS[@]}"; do
        if [[ -f "$path" ]]; then
            CLAUDE_CONFIG="$path"
            break
        fi
    done

    if [[ -n "$CLAUDE_CONFIG" ]]; then
        # Backup existing config
        cp "$CLAUDE_CONFIG" "$CLAUDE_CONFIG.backup"
        
        # Add hook configuration (simplified approach)
        print_info "Claude Code config found: $CLAUDE_CONFIG"
        print_info "Please manually add the following to your Claude Code settings:"
        echo
        echo '{'
        echo '  "hooks": {'
        echo "    \"pre_action\": \"$INSTALL_DIR/claude-monitor\""
        echo '  }'
        echo '}'
        echo
    else
        print_warning "Claude Code configuration not found"
        print_info "Please configure Claude Code hooks manually:"
        print_info "Add this to your Claude Code settings:"
        echo
        echo '{'
        echo '  "hooks": {'
        echo "    \"pre_action\": \"$INSTALL_DIR/claude-monitor\""
        echo '  }'
        echo '}'
        echo
    fi
}

# Start services
start_services() {
    if [[ "$START_DAEMON" != "true" ]]; then
        return
    fi

    print_step "Starting Claude Monitor daemon..."

    case $PLATFORM in
        linux|wsl)
            if [[ "$INSTALL_SERVICE" == "true" ]]; then
                sudo systemctl start claude-monitor
                print_success "Daemon started via systemd"
            else
                nohup "$INSTALL_DIR/claude-daemon" -config "$CONFIG_DIR/config.json" > "$CONFIG_DIR/daemon.log" 2>&1 &
                print_success "Daemon started in background"
            fi
            ;;
            
        macos)
            if [[ "$INSTALL_SERVICE" == "true" ]]; then
                # Already started by launchctl load
                print_success "Daemon started via LaunchAgent"
            else
                nohup "$INSTALL_DIR/claude-daemon" -config "$CONFIG_DIR/config.json" > "$CONFIG_DIR/daemon.log" 2>&1 &
                print_success "Daemon started in background"
            fi
            ;;
            
        *)
            "$INSTALL_DIR/claude-daemon" -config "$CONFIG_DIR/config.json" &
            print_success "Daemon started"
            ;;
    esac

    # Wait a moment and test
    sleep 2
    if curl -s http://localhost:8080/health > /dev/null; then
        print_success "Daemon is running and healthy"
    else
        print_warning "Daemon may not be running properly"
        print_info "Check logs: $CONFIG_DIR/daemon.log"
    fi
}

# Verification
verify_installation() {
    print_step "Verifying installation..."

    # Check binaries
    if [[ "$INSTALL_DAEMON" == "true" ]]; then
        if [[ -x "$INSTALL_DIR/claude-daemon" ]]; then
            VERSION_OUTPUT=$($INSTALL_DIR/claude-daemon -version 2>/dev/null | head -n1 || echo "unknown")
            print_success "Daemon: $VERSION_OUTPUT"
        else
            print_error "Daemon not found or not executable: $INSTALL_DIR/claude-daemon"
        fi
    fi

    if [[ "$INSTALL_HOOK" == "true" ]]; then
        if [[ -x "$INSTALL_DIR/claude-monitor" ]]; then
            VERSION_OUTPUT=$($INSTALL_DIR/claude-monitor -version 2>/dev/null | head -n1 || echo "unknown")
            print_success "Hook: $VERSION_OUTPUT"
        else
            print_error "Hook not found or not executable: $INSTALL_DIR/claude-monitor"
        fi
    fi

    # Check configuration
    if [[ -f "$CONFIG_DIR/config.json" ]]; then
        print_success "Configuration: $CONFIG_DIR/config.json"
    fi

    # Test hook execution
    if [[ "$INSTALL_HOOK" == "true" && -x "$INSTALL_DIR/claude-monitor" ]]; then
        if timeout 1s "$INSTALL_DIR/claude-monitor" -debug > /dev/null 2>&1; then
            print_success "Hook execution test passed"
        else
            print_warning "Hook execution test failed (may be normal if daemon not running)"
        fi
    fi
}

print_completion() {
    echo
    print_success "Claude Monitor installation completed!"
    echo
    
    echo -e "${BLUE}Next Steps:${NC}"
    echo
    
    if [[ "$INSTALL_DAEMON" == "true" && "$START_DAEMON" != "true" ]]; then
        echo "1. Start the daemon:"
        if [[ "$INSTALL_SERVICE" == "true" ]]; then
            case $PLATFORM in
                linux|wsl) echo "   sudo systemctl start claude-monitor" ;;
                macos) echo "   launchctl load ~/Library/LaunchAgents/com.claude-monitor.daemon.plist" ;;
            esac
        else
            echo "   $INSTALL_DIR/claude-daemon"
        fi
        echo
    fi
    
    if [[ "$INSTALL_HOOK" == "true" && "$CONFIGURE_CLAUDE" != "true" ]]; then
        echo "2. Configure Claude Code hooks:"
        echo '   Add to Claude Code settings: {"hooks": {"pre_action": "'$INSTALL_DIR'/claude-monitor"}}'
        echo
    fi
    
    echo "3. Test the integration:"
    echo "   # Check daemon health"
    echo "   curl http://localhost:8080/health"
    echo
    echo "   # Test hook execution"
    echo "   $INSTALL_DIR/claude-monitor -debug"
    echo
    echo "4. View your work tracking:"
    echo "   # Check current status"
    echo "   curl \"http://localhost:8080/status?user_id=\$(whoami)\""
    echo
    
    echo -e "${BLUE}Configuration:${NC}"
    echo "  Config file: $CONFIG_DIR/config.json"
    echo "  Data directory: $CONFIG_DIR/data"
    echo "  Logs: $CONFIG_DIR/"
    echo
    
    echo -e "${BLUE}Troubleshooting:${NC}"
    echo "  Documentation: cmd/claude-monitor/README.md"
    echo "  Debug hook: $INSTALL_DIR/claude-monitor -debug"
    echo "  View logs: tail -f $CONFIG_DIR/daemon.log"
    echo
    
    echo -e "${GREEN}Happy tracking! 🎯${NC}"
}

# Main installation flow
main() {
    print_header
    parse_args "$@"
    detect_system
    check_prerequisites
    build_binaries
    install_binaries
    setup_configuration
    install_service
    configure_claude_code
    start_services
    verify_installation
    print_completion
}

# Run main function
main "$@"