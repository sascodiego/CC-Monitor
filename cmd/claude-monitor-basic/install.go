package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-INSTALL-001
 * CONTEXT:   Self-contained installation system within the CLI binary
 * REASON:    Provide zero-configuration installation without external scripts
 * CHANGE:    Built-in installation capability for seamless setup.
 * PREVENTION:Validate all paths and permissions before creating files/directories
 * RISK:      Medium - Installation operations require system changes and permissions
 */

const (
	configTemplate = `# Claude Monitor Configuration - Auto-generated
daemon:
  log_level: INFO
  database_path: %s/claude.db
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
  rounding_method: nearest    # Time rounding method
  rounding_interval: 15m      # Rounding interval

export:
  default_format: json        # Default export format
  include_charts: true        # Include charts in reports
  template: professional     # Report template
`

	serviceTemplate = `[Unit]
Description=Claude Monitor Enhanced Daemon
Documentation=https://github.com/sascodiego/CC-Monitor
After=network.target
Wants=network.target

[Service]
Type=simple
User=root
Group=root
ExecStart=%s
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=claude-monitor

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=%s %s /tmp
PrivateTmp=false

[Install]
WantedBy=multi-user.target
`
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and configure Claude Monitor system",
	Long: `Automatically install and configure Claude Monitor with zero manual setup required.
This command will:
- Create all necessary directories
- Generate configuration files
- Set up systemd service (Linux)
- Initialize the database
- Start monitoring

Requires sudo privileges for system installation.`,
	Run: runInstall,
}

var (
	installForce    bool
	installUserOnly bool
	installDir      string
)

func init() {
	installCmd.Flags().BoolVar(&installForce, "force", false, "Force reinstallation even if already installed")
	installCmd.Flags().BoolVar(&installUserOnly, "user", false, "Install for current user only (no system service)")
	installCmd.Flags().StringVar(&installDir, "dir", "", "Custom installation directory (default: /usr/local/bin)")
}

func runInstall(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Claude Monitor - Automatic Installation")
	fmt.Println("═════════════════════════════════════════")
	fmt.Println()

	// Check system compatibility
	if runtime.GOOS != "linux" {
		fmt.Printf("❌ Unsupported operating system: %s\n", runtime.GOOS)
		fmt.Println("Claude Monitor currently supports Linux and WSL2 only.")
		os.Exit(1)
	}

	// Check if running as root (unless user-only install)
	if !installUserOnly && os.Geteuid() != 0 {
		fmt.Println("❌ System installation requires root privileges")
		fmt.Println("Please run: sudo claude-monitor install")
		fmt.Println("Or use: claude-monitor install --user (for user-only installation)")
		os.Exit(1)
	}

	// Detect current user
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("❌ Cannot detect current user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ User: %s\n", currentUser.Username)
	fmt.Printf("✅ OS: Linux (%s)\n", runtime.GOARCH)

	// Set installation paths
	var (
		binDir    string
		configDir string
		dataDir   string
		logDir    string
	)

	if installUserOnly {
		homeDir := currentUser.HomeDir
		binDir = filepath.Join(homeDir, ".local", "bin")
		configDir = filepath.Join(homeDir, ".config", "claude-monitor")
		dataDir = filepath.Join(homeDir, ".local", "share", "claude-monitor")
		logDir = filepath.Join(homeDir, ".local", "share", "claude-monitor", "logs")
	} else {
		if installDir != "" {
			binDir = installDir
		} else {
			binDir = "/usr/local/bin"
		}
		configDir = "/etc/claude-monitor"
		dataDir = "/var/lib/claude-monitor"
		logDir = "/var/log/claude-monitor"
	}

	fmt.Println()
	fmt.Println("📦 Installation Configuration:")
	fmt.Printf("   Binaries: %s\n", binDir)
	fmt.Printf("   Config:   %s\n", configDir)
	fmt.Printf("   Data:     %s\n", dataDir)
	fmt.Printf("   Logs:     %s\n", logDir)
	fmt.Println()

	// Create directories
	fmt.Println("📁 Creating directories...")
	dirs := []string{binDir, configDir, dataDir, logDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("❌ Failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Printf("   ✅ %s\n", dir)
	}

	// Copy current binary to installation directory
	fmt.Println()
	fmt.Println("🔧 Installing binaries...")
	
	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		fmt.Printf("❌ Cannot determine current executable: %v\n", err)
		os.Exit(1)
	}

	// Copy CLI binary
	cliTarget := filepath.Join(binDir, "claude-monitor")
	if err := copyFile(currentExe, cliTarget); err != nil {
		fmt.Printf("❌ Failed to install CLI binary: %v\n", err)
		os.Exit(1)
	}
	if err := os.Chmod(cliTarget, 0755); err != nil {
		fmt.Printf("❌ Failed to set permissions on CLI binary: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✅ claude-monitor -> %s\n", cliTarget)

	// Check if daemon binaries exist and copy them
	possibleDaemons := []string{
		"claude-daemon-enhanced",
		"claude-daemon-simple",
		"claude-daemon",
	}

	var daemonPath string
	currentDir := filepath.Dir(currentExe)
	
	for _, daemon := range possibleDaemons {
		testPath := filepath.Join(currentDir, "..", "..", "bin", daemon)
		if _, err := os.Stat(testPath); err == nil {
			daemonTarget := filepath.Join(binDir, daemon)
			if err := copyFile(testPath, daemonTarget); err == nil {
				if err := os.Chmod(daemonTarget, 0755); err == nil {
					fmt.Printf("   ✅ %s -> %s\n", daemon, daemonTarget)
					if daemon == "claude-daemon-enhanced" {
						daemonPath = daemonTarget
					}
				}
			}
		}
	}

	if daemonPath == "" {
		fmt.Println("   ⚠️  No daemon binary found - you may need to build first")
		daemonPath = filepath.Join(binDir, "claude-daemon-enhanced")
	}

	// Create configuration file
	fmt.Println()
	fmt.Println("⚙️  Creating configuration...")
	configFile := filepath.Join(configDir, "config.yaml")
	configContent := fmt.Sprintf(configTemplate, dataDir)
	
	if err := writeFile(configFile, configContent, 0644); err != nil {
		fmt.Printf("❌ Failed to create config file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✅ Config: %s\n", configFile)

	// Initialize database
	fmt.Println()
	fmt.Println("🗄️  Initializing database...")
	dbFile := filepath.Join(dataDir, "claude.db")
	if err := writeFile(dbFile, "", 0644); err != nil {
		fmt.Printf("❌ Failed to create database file: %v\n", err)
		os.Exit(1)
	}

	// Set proper ownership for data directory
	if !installUserOnly && currentUser.Username != "root" {
		// Try to set ownership to the user who ran sudo
		if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
			if sudoUserInfo, err := user.Lookup(sudoUser); err == nil {
				if cmd := exec.Command("chown", "-R", sudoUser+":"+sudoUserInfo.Gid, dataDir); cmd.Run() == nil {
					fmt.Printf("   ✅ Set ownership to %s\n", sudoUser)
				}
			}
		}
	}
	fmt.Printf("   ✅ Database: %s\n", dbFile)

	// Create systemd service (if system install)
	if !installUserOnly {
		fmt.Println()
		fmt.Println("🔧 Setting up system service...")
		serviceFile := "/etc/systemd/system/claude-monitor.service"
		serviceContent := fmt.Sprintf(serviceTemplate, daemonPath, dataDir, logDir)
		
		if err := writeFile(serviceFile, serviceContent, 0644); err != nil {
			fmt.Printf("❌ Failed to create systemd service: %v\n", err)
			os.Exit(1)
		}

		// Reload systemd and enable service
		if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to reload systemd: %v\n", err)
		} else {
			fmt.Println("   ✅ Systemd reloaded")
		}

		if err := exec.Command("systemctl", "enable", "claude-monitor").Run(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to enable service: %v\n", err)
		} else {
			fmt.Println("   ✅ Service enabled")
		}

		// Start service
		fmt.Println("   🚀 Starting service...")
		if err := exec.Command("systemctl", "start", "claude-monitor").Run(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to start service: %v\n", err)
			fmt.Println("      You can start manually with: sudo systemctl start claude-monitor")
		} else {
			fmt.Println("   ✅ Service started")
		}
	}

	// Create convenience scripts
	fmt.Println()
	fmt.Println("🛠️  Creating convenience commands...")
	
	if !installUserOnly {
		createConvenienceScript(binDir, "claude-monitor-start", `#!/bin/bash
echo "🚀 Starting Claude Monitor..."
sudo systemctl start claude-monitor
sleep 2
systemctl is-active --quiet claude-monitor && echo "✅ Started" || echo "❌ Failed"
`)
		createConvenienceScript(binDir, "claude-monitor-stop", `#!/bin/bash
echo "🛑 Stopping Claude Monitor..."
sudo systemctl stop claude-monitor
echo "✅ Stopped"
`)
		createConvenienceScript(binDir, "claude-monitor-restart", `#!/bin/bash
echo "🔄 Restarting Claude Monitor..."
sudo systemctl restart claude-monitor
sleep 2
systemctl is-active --quiet claude-monitor && echo "✅ Restarted" || echo "❌ Failed"
`)
	}

	// Final verification
	fmt.Println()
	fmt.Println("🔍 Verifying installation...")
	
	// Check if CLI works
	if _, err := exec.Command(cliTarget, "--help").Output(); err != nil {
		fmt.Printf("⚠️  CLI verification failed: %v\n", err)
	} else {
		fmt.Println("   ✅ CLI working")
	}

	// Check service status (if system install)
	if !installUserOnly {
		if output, err := exec.Command("systemctl", "is-active", "claude-monitor").Output(); err == nil {
			status := strings.TrimSpace(string(output))
			if status == "active" {
				fmt.Println("   ✅ Service running")
			} else {
				fmt.Printf("   ⚠️  Service status: %s\n", status)
			}
		}
	}

	// Success message
	fmt.Println()
	fmt.Println("🎉 INSTALLATION COMPLETE!")
	fmt.Println("════════════════════════")
	fmt.Println()
	fmt.Println("📋 Quick Start:")
	fmt.Println()
	fmt.Println("  # Check status")
	fmt.Println("  claude-monitor status")
	fmt.Println()
	fmt.Println("  # View work day")
	fmt.Println("  claude-monitor workhour workday status")
	fmt.Println()
	
	if !installUserOnly {
		fmt.Println("  # Control service")
		fmt.Println("  sudo systemctl start claude-monitor    # Start")
		fmt.Println("  sudo systemctl stop claude-monitor     # Stop")
		fmt.Println("  sudo systemctl status claude-monitor   # Status")
		fmt.Println()
		fmt.Println("  # Or use convenience commands:")
		fmt.Println("  claude-monitor-start")
		fmt.Println("  claude-monitor-stop")
		fmt.Println("  claude-monitor-restart")
		fmt.Println()
	}
	
	fmt.Println("🚀 Claude Monitor is ready!")
	fmt.Println("Start using Claude CLI and the system will automatically track your work hours.")
	
	// Try to show current status
	if !installUserOnly {
		fmt.Println()
		fmt.Println("📊 Current Status:")
		if err := exec.Command(cliTarget, "status").Run(); err != nil {
			fmt.Println("Status will be available once the daemon starts and detects Claude activity.")
		}
	}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	buf := make([]byte, 64*1024)
	for {
		n, err := srcFile.Read(buf)
		if n == 0 {
			break
		}
		if err != nil {
			return err
		}
		if _, err := dstFile.Write(buf[:n]); err != nil {
			return err
		}
	}

	return os.Chmod(dst, srcInfo.Mode())
}

func writeFile(path, content string, perm os.FileMode) error {
	return os.WriteFile(path, []byte(content), perm)
}

func createConvenienceScript(dir, name, content string) {
	scriptPath := filepath.Join(dir, name)
	if err := writeFile(scriptPath, content, 0755); err == nil {
		fmt.Printf("   ✅ %s\n", name)
	}
}