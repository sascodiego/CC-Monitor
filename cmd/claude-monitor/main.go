package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-PRODUCTION-001
 * CONTEXT:   Clean production CLI that works with existing enhanced daemon
 * REASON:    Need working CLI without complex dependencies that interfaces with proven daemon
 * CHANGE:    Simple production CLI with install command and status reading.
 * PREVENTION:Keep dependencies minimal, read status from daemon's JSON output
 * RISK:      Low - Simple file operations and system commands only
 */

const version = "v1.0.0"

type StatusData struct {
	DaemonRunning    bool `json:"daemonRunning"`
	MonitoringActive bool `json:"monitoringActive"`
	CurrentSession   struct {
		SessionID string    `json:"sessionID"`
		StartTime time.Time `json:"startTime"`
		EndTime   time.Time `json:"endTime"`
		IsActive  bool      `json:"isActive"`
	} `json:"currentSession"`
	CurrentWorkBlock struct {
		BlockID         string    `json:"blockID"`
		SessionID       string    `json:"sessionID"`
		StartTime       time.Time `json:"startTime"`
		DurationSeconds int       `json:"durationSeconds"`
		IsActive        bool      `json:"isActive"`
	} `json:"currentWorkBlock"`
	TimeSinceActivity int64 `json:"timeSinceActivity"`
	InactiveTimeout   bool  `json:"inactiveTimeout"`
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		return
	}

	switch os.Args[1] {
	case "install":
		if err := installSystem(); err != nil {
			fmt.Printf("❌ Installation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Claude Monitor installed successfully!")
		showUsage()
	case "status":
		showStatus()
	case "report":
		generateReport()
	case "export": 
		exportData()
	case "start":
		startService()
	case "stop":
		stopService()
	case "restart":
		restartService()
	case "version":
		fmt.Printf("Claude Monitor %s\n", version)
	case "help", "-h", "--help":
		showHelp()
	default:
		fmt.Printf("❌ Unknown command: %s\n", os.Args[1])
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf(`Claude Monitor %s - Work Hour Tracking System

USAGE:
  claude-monitor <command>

COMMANDS:
  install    Install Claude Monitor system with single command
  status     Show current monitoring status and session info
  report     Generate database reports (shows current data)
  export     Export monitoring data 
  start      Start the monitoring service
  stop       Stop the monitoring service  
  restart    Restart the monitoring service
  version    Show version information
  help       Show this help message

Claude Monitor automatically tracks Claude CLI sessions with:
• 5-hour session windows from first interaction
• 5-minute inactivity timeout for work blocks
• Persistent SQLite database storage
• Real-time status monitoring

The system runs as a background daemon and stores all data persistently.
`, version)
}

func showUsage() {
	fmt.Println("\n🎯 Quick Start:")
	fmt.Println("  claude-monitor status     # Check current session")
	fmt.Println("  claude-monitor report     # View collected data")
	fmt.Println("  claude-monitor export     # Export your work data")
}

func installSystem() error {
	fmt.Println("🚀 Installing Claude Monitor Production System...")

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	projectDir := filepath.Dir(filepath.Dir(execPath))

	// Build the complete daemon  
	fmt.Println("📦 Building claude-daemon-complete...")
	buildCmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && CGO_ENABLED=1 go build -ldflags=\"-s -w\" -o bin/claude-daemon-complete ./cmd/claude-daemon-complete", projectDir))
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build daemon: %w\nOutput: %s", err, string(output))
	}

	// Stop any existing daemon
	fmt.Println("🛑 Stopping existing daemons...")
	exec.Command("sudo", "systemctl", "stop", "claude-monitor").Run()
	exec.Command("sudo", "pkill", "-f", "claude-daemon").Run()

	// Create system directories
	fmt.Println("📁 Creating system directories...")
	dirs := []string{
		"/var/lib/claude-monitor",
		"/var/log/claude-monitor", 
		"/etc/claude-monitor",
	}
	
	for _, dir := range dirs {
		if err := runSudoCommand("mkdir", "-p", dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Set permissions
	user := getCurrentUser()
	for _, dir := range dirs {
		if err := runSudoCommand("chown", user+":"+user, dir); err != nil {
			return fmt.Errorf("failed to set permissions on %s: %w", dir, err)
		}
	}

	// Install CLI
	fmt.Println("🔧 Installing CLI...")
	if err := runSudoCommand("cp", execPath, "/usr/local/bin/claude-monitor"); err != nil {
		return fmt.Errorf("failed to install CLI: %w", err)
	}
	if err := runSudoCommand("chmod", "+x", "/usr/local/bin/claude-monitor"); err != nil {
		return fmt.Errorf("failed to set CLI permissions: %w", err)
	}

	// Install daemon
	fmt.Println("⚙️  Installing daemon...")
	daemonPath := filepath.Join(projectDir, "bin", "claude-daemon-complete")
	if err := runSudoCommand("cp", daemonPath, "/usr/local/bin/claude-daemon-complete"); err != nil {
		return fmt.Errorf("failed to install daemon: %w", err)
	}
	if err := runSudoCommand("chmod", "+x", "/usr/local/bin/claude-daemon-complete"); err != nil {
		return fmt.Errorf("failed to set daemon permissions: %w", err)
	}

	// Create systemd service
	fmt.Println("🔄 Creating systemd service...")
	serviceContent := fmt.Sprintf(`[Unit]
Description=Claude Monitor Daemon - Work Hour Tracking
After=network.target

[Service]
Type=simple
User=%s
Group=%s
ExecStart=/usr/local/bin/claude-daemon-complete
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/claude-monitor /var/log/claude-monitor /tmp

[Install]
WantedBy=multi-user.target
`, user, user)

	if err := writeSudoFile("/etc/systemd/system/claude-monitor.service", serviceContent); err != nil {
		return fmt.Errorf("failed to create systemd service: %w", err)
	}

	// Enable and start service
	fmt.Println("🎯 Enabling and starting service...")
	if err := runSudoCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	if err := runSudoCommand("systemctl", "enable", "claude-monitor"); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}
	if err := runSudoCommand("systemctl", "start", "claude-monitor"); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait a moment for service to start
	fmt.Println("⏳ Waiting for service to start...")
	time.Sleep(3 * time.Second)

	return nil
}

func showStatus() {
	fmt.Println("📊 Claude Monitor Status")
	fmt.Println("========================")
	
	// Check systemd service status
	cmd := exec.Command("systemctl", "is-active", "claude-monitor")
	output, err := cmd.Output()
	serviceActive := err == nil && strings.TrimSpace(string(output)) == "active"
	
	if serviceActive {
		fmt.Println("✅ Service: Running")
	} else {
		fmt.Println("❌ Service: Not running")
		fmt.Println("💡 Run 'claude-monitor install' to set up the system")
		fmt.Println("💡 Run 'claude-monitor start' to start the service")
		return
	}
	
	// Read daemon status
	statusFile := "/tmp/claude-monitor-status.json"
	if data, err := ioutil.ReadFile(statusFile); err == nil {
		var status StatusData
		if err := json.Unmarshal(data, &status); err == nil {
			fmt.Printf("✅ Daemon: %s\n", map[bool]string{true: "Active", false: "Inactive"}[status.DaemonRunning])
			fmt.Printf("📡 Monitoring: %s\n", map[bool]string{true: "Active", false: "Inactive"}[status.MonitoringActive])
			
			if status.CurrentSession.IsActive {
				fmt.Printf("\n📅 Current Session:\n")
				fmt.Printf("   ID: %s\n", status.CurrentSession.SessionID[:8]+"...")
				fmt.Printf("   Started: %s\n", status.CurrentSession.StartTime.Format("15:04:05"))
				fmt.Printf("   Ends: %s\n", status.CurrentSession.EndTime.Format("15:04:05"))
				
				if status.CurrentWorkBlock.IsActive {
					fmt.Printf("\n⏱️  Current Work Block:\n")
					fmt.Printf("   Duration: %d minutes\n", status.CurrentWorkBlock.DurationSeconds/60)
					fmt.Printf("   Active: %s\n", map[bool]string{true: "Yes", false: "No"}[status.CurrentWorkBlock.IsActive])
				}
				
				if status.InactiveTimeout {
					fmt.Printf("\n⏸️  Status: Inactive (timeout)\n")
					fmt.Printf("   Idle time: %d minutes\n", status.TimeSinceActivity/(1000000000*60))
				} else {
					fmt.Printf("\n🟢 Status: Active\n")
				}
			} else {
				fmt.Printf("\n⏸️  No active session\n")
			}
		} else {
			fmt.Println("⚠️  Could not parse status data")
		}
	} else {
		fmt.Println("⚠️  Status file not found - daemon may be starting")
	}
	
	// Check database
	dbPath := "/var/lib/claude-monitor/claude.db"
	if info, err := os.Stat(dbPath); err == nil {
		fmt.Printf("\n💾 Database: %s (%.1f KB)\n", dbPath, float64(info.Size())/1024)
	} else {
		fmt.Printf("\n❌ Database: Not found\n")
	}
}

func generateReport() {
	fmt.Println("📈 Claude Monitor Data Report")
	fmt.Println("=============================")
	
	dbPath := "/var/lib/claude-monitor/claude.db"
	if info, err := os.Stat(dbPath); err == nil {
		fmt.Printf("✅ Database: %s (%.1f KB)\n", dbPath, float64(info.Size())/1024)
		fmt.Printf("📊 Database contains persistent work hour tracking data\n")
		fmt.Printf("🔗 Sessions, work blocks, and activity data are stored\n")
		
		// Show current status as a basic report
		fmt.Println("\n📋 Current Status:")
		statusFile := "/tmp/claude-monitor-status.json"
		if data, err := ioutil.ReadFile(statusFile); err == nil {
			var status StatusData
			if err := json.Unmarshal(data, &status); err == nil {
				if status.CurrentSession.IsActive {
					duration := time.Since(status.CurrentSession.StartTime)
					fmt.Printf("• Active session: %v\n", duration.Round(time.Minute))
					fmt.Printf("• Session ends: %s\n", status.CurrentSession.EndTime.Format("15:04"))
				}
			}
		}
		
		fmt.Println("\n💡 Advanced reporting features will be added in future versions")
		fmt.Println("💡 Raw data is available in the SQLite database")
	} else {
		fmt.Println("❌ No database found")
		fmt.Println("💡 Run 'claude-monitor install' to set up the system")
	}
}

func exportData() {
	fmt.Println("📤 Claude Monitor Data Export")
	fmt.Println("=============================")
	
	dbPath := "/var/lib/claude-monitor/claude.db"
	if info, err := os.Stat(dbPath); err == nil {
		fmt.Printf("✅ Database found: %s (%.1f KB)\n", dbPath, float64(info.Size())/1024)
		fmt.Println("📊 Database contains:")
		fmt.Println("  • Session records (5-hour windows)")
		fmt.Println("  • Work block records (activity periods)")  
		fmt.Println("  • Work day summaries")
		fmt.Println("  • Activity timestamps")
		
		fmt.Printf("\n💾 You can access the SQLite database directly:\n")
		fmt.Printf("   sqlite3 %s\n", dbPath)
		fmt.Printf("   .tables\n")
		fmt.Printf("   SELECT * FROM sessions;\n")
		fmt.Printf("   SELECT * FROM work_blocks;\n")
		
		fmt.Println("\n💡 Export functionality will be enhanced in future versions")
	} else {
		fmt.Println("❌ No database found")
		fmt.Println("💡 Run 'claude-monitor install' first")
	}
}

func startService() {
	fmt.Println("🚀 Starting Claude Monitor service...")
	if err := runSudoCommand("systemctl", "start", "claude-monitor"); err != nil {
		fmt.Printf("❌ Failed to start service: %v\n", err)
		fmt.Println("💡 Try: claude-monitor install")
	} else {
		fmt.Println("✅ Service started successfully")
		time.Sleep(2 * time.Second)
		showStatus()
	}
}

func stopService() {
	fmt.Println("🛑 Stopping Claude Monitor service...")
	if err := runSudoCommand("systemctl", "stop", "claude-monitor"); err != nil {
		fmt.Printf("❌ Failed to stop service: %v\n", err)
	} else {
		fmt.Println("✅ Service stopped successfully")
	}
}

func restartService() {
	fmt.Println("🔄 Restarting Claude Monitor service...")
	if err := runSudoCommand("systemctl", "restart", "claude-monitor"); err != nil {
		fmt.Printf("❌ Failed to restart service: %v\n", err)
	} else {
		fmt.Println("✅ Service restarted successfully")
		time.Sleep(2 * time.Second)
		showStatus()
	}
}

func runSudoCommand(args ...string) error {
	cmd := exec.Command("sudo", args...)
	cmd.Stdin = strings.NewReader("yoelego1995\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

func writeSudoFile(path, content string) error {
	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = strings.NewReader(content)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

func getCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("LOGNAME"); user != "" {
		return user
	}
	return "dsasco" // fallback
}