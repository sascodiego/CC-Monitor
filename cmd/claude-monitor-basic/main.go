package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"github.com/spf13/cobra"
)

/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-BASIC-001
 * CONTEXT:   Basic CLI for testing daemon functionality without complex dependencies
 * REASON:    Need simple CLI to test core daemon features without import cycle issues
 * CHANGE:    Basic implementation for immediate testing.
 * PREVENTION:Keep minimal dependencies, focus on core functionality
 * RISK:      Low - Simple implementation for testing purposes
 */

type DaemonStatus struct {
	ActivityHistory struct {
		CurrentProcesses      int    `json:"currentProcesses"`
		CurrentConnections    int    `json:"currentConnections"`
		CurrentAPIActivity    bool   `json:"currentAPIActivity"`
		TrafficPattern        string `json:"trafficPattern"`
		DataTransferBytes     int64  `json:"dataTransferBytes"`
		HTTPEventsDetected    bool   `json:"httpEventsDetected"`
		UserInteractions      int    `json:"userInteractions"`
		BackgroundOps         int    `json:"backgroundOps"`
		MinActivityThreshold  int64  `json:"minActivityThreshold"`
	} `json:"activityHistory"`
	CurrentSession struct {
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
		LastActivity    time.Time `json:"lastActivity"`
		IsActive        bool      `json:"isActive"`
	} `json:"currentWorkBlock"`
	DaemonRunning      bool      `json:"daemonRunning"`
	MonitoringActive   bool      `json:"monitoringActive"`
	LastRealActivity   time.Time `json:"lastRealActivity"`
	InactiveTimeout    bool      `json:"inactiveTimeout"`
	TimeSinceActivity  int64     `json:"timeSinceActivity"`
	Timestamp          time.Time `json:"timestamp"`
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "claude-monitor",
		Short: "Claude Monitor - Work Hour Tracking System",
		Long:  `Claude Monitor Alpha - Advanced work hour tracking and productivity analysis for Claude usage.`,
	}

	// Status command
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show current monitoring status",
		Run: func(cmd *cobra.Command, args []string) {
			showStatus()
		},
	}

	// Work hour commands
	var workhourCmd = &cobra.Command{
		Use:   "workhour",
		Short: "Work hour tracking and analysis",
		Long:  `Work hour tracking commands for productivity analysis and time management.`,
	}

	var workdayCmd = &cobra.Command{
		Use:   "workday",
		Short: "Daily work hour operations",
	}

	var workdayStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show current work day status",
		Run: func(cmd *cobra.Command, args []string) {
			showWorkdayStatus()
		},
	}

	// Build command tree
	workdayCmd.AddCommand(workdayStatusCmd)
	workhourCmd.AddCommand(workdayCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(workhourCmd)
	rootCmd.AddCommand(installCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showStatus() {
	fmt.Println("🚀 Claude Monitor System Status")
	fmt.Println("═══════════════════════════════════")
	
	status, err := readDaemonStatus()
	if err != nil {
		fmt.Printf("❌ Error reading daemon status: %v\n", err)
		return
	}

	// Daemon status
	if status.DaemonRunning {
		fmt.Println("✅ Enhanced Daemon: RUNNING")
	} else {
		fmt.Println("❌ Enhanced Daemon: STOPPED")
	}

	// Current activity
	fmt.Printf("📊 Current Activity: %d processes, %d connections\n", 
		status.ActivityHistory.CurrentProcesses, 
		status.ActivityHistory.CurrentConnections)
	
	if status.ActivityHistory.CurrentAPIActivity {
		fmt.Println("🔥 API Activity: DETECTED")
	} else {
		fmt.Println("💤 API Activity: IDLE")
	}

	// Session info
	if status.CurrentSession.IsActive {
		duration := time.Since(status.CurrentSession.StartTime)
		fmt.Printf("📈 Current Session: %s (running %v)\n", 
			status.CurrentSession.SessionID[:8], 
			duration.Round(time.Minute))
		fmt.Printf("⏰ Session Window: %s → %s\n", 
			status.CurrentSession.StartTime.Format("15:04"), 
			status.CurrentSession.EndTime.Format("15:04"))
	}

	// Work block info
	if status.CurrentWorkBlock.IsActive {
		workDuration := time.Since(status.CurrentWorkBlock.StartTime)
		fmt.Printf("🔥 Current Work Block: %s (active %v)\n", 
			status.CurrentWorkBlock.BlockID[:8], 
			workDuration.Round(time.Minute))
		
		timeSinceActivity := time.Duration(status.TimeSinceActivity)
		fmt.Printf("⏱️  Last Activity: %v ago\n", timeSinceActivity.Round(time.Second))
		
		if status.InactiveTimeout {
			fmt.Println("⚠️  Inactivity Timeout: APPROACHING (>5 minutes)")
		}
	}

	// Enhanced monitoring features
	fmt.Printf("🎯 Traffic Pattern: %s\n", status.ActivityHistory.TrafficPattern)
	fmt.Printf("📊 Data Transfer: %d bytes\n", status.ActivityHistory.DataTransferBytes)
	
	if status.ActivityHistory.HTTPEventsDetected {
		fmt.Printf("🌐 HTTP Detection: %d user interactions, %d background ops\n", 
			status.ActivityHistory.UserInteractions, 
			status.ActivityHistory.BackgroundOps)
	} else {
		fmt.Println("🌐 HTTP Detection: Not available (requires eBPF)")
	}

	fmt.Printf("\n📅 Status Updated: %s\n", status.Timestamp.Format("15:04:05"))
}

func showWorkdayStatus() {
	fmt.Println("📊 Work Day Status - " + time.Now().Format("January 2, 2006"))
	fmt.Println("═══════════════════════════════════════════════")
	
	status, err := readDaemonStatus()
	if err != nil {
		fmt.Printf("❌ Error reading status: %v\n", err)
		return
	}

	// Calculate work time
	if status.CurrentSession.IsActive && status.CurrentWorkBlock.IsActive {
		workStart := status.CurrentWorkBlock.StartTime
		workDuration := time.Since(workStart)
		
		fmt.Printf("⏰ Work Period: %s → ACTIVE (%v)\n", 
			workStart.Format("15:04"), 
			workDuration.Round(time.Minute))
		
		// Productivity analysis
		efficiency := calculateEfficiency(status)
		fmt.Printf("📈 Productivity: %.1f%% efficiency\n", efficiency)
		
		// Goal progress (assume 8h target)
		targetHours := 8.0
		currentHours := workDuration.Hours()
		progress := (currentHours / targetHours) * 100
		
		fmt.Printf("🎯 Goal Progress: %.1f%% of %.0fh target", progress, targetHours)
		if progress >= 100 {
			fmt.Printf(" ✅\n")
		} else {
			fmt.Printf("\n")
		}
		
		// Activity summary
		fmt.Printf("📋 Session: %s\n", status.CurrentSession.SessionID[:8])
		fmt.Printf("🔥 Work Block: %s (active)\n", status.CurrentWorkBlock.BlockID[:8])
		
		// Recent activity
		lastActivity := time.Since(status.LastRealActivity)
		fmt.Printf("⚡ Last Activity: %v ago\n", lastActivity.Round(time.Second))
		
		// Enhanced metrics
		if status.ActivityHistory.HTTPEventsDetected {
			fmt.Printf("💻 HTTP Activity: %d interactions detected\n", 
				status.ActivityHistory.UserInteractions)
		}
		
		fmt.Printf("🌐 Network: %d connections, %s pattern\n", 
			status.ActivityHistory.CurrentConnections, 
			status.ActivityHistory.TrafficPattern)
	} else {
		fmt.Println("💤 No active work session detected")
		fmt.Println("   Start using Claude to begin time tracking")
	}
}

func calculateEfficiency(status *DaemonStatus) float64 {
	// Simple efficiency calculation based on activity vs idle time
	totalTime := time.Since(status.CurrentWorkBlock.StartTime).Seconds()
	idleTime := float64(status.TimeSinceActivity) / 1e9 // nanoseconds to seconds
	
	if totalTime <= 0 {
		return 0
	}
	
	activeTime := totalTime - idleTime
	if activeTime < 0 {
		activeTime = 0
	}
	
	efficiency := (activeTime / totalTime) * 100
	if efficiency > 100 {
		efficiency = 100
	}
	
	return efficiency
}

func readDaemonStatus() (*DaemonStatus, error) {
	statusFile := "/tmp/claude-monitor-status.json"
	
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read status file: %v", err)
	}
	
	var status DaemonStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status JSON: %v", err)
	}
	
	return &status, nil
}