/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-SIMPLE-001
 * CONTEXT:   Simplified daemon for testing and demonstration purposes
 * REASON:    Need working daemon that can detect Claude processes and write status for CLI testing
 * CHANGE:    Simplified implementation for immediate testing.
 * PREVENTION:This is a simplified version - full IPC should be implemented for production
 * RISK:      Low - This is for testing only, production needs full implementation
 */
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/domain"
	"github.com/google/uuid"
)

type SimplifiedDaemon struct {
	logger        arch.Logger
	statusFile    string
	pidFile       string
	currentSession *domain.Session
	currentWorkBlock *domain.WorkBlock
	running       bool
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewSimplifiedDaemon(log arch.Logger) *SimplifiedDaemon {
	ctx, cancel := context.WithCancel(context.Background())
	return &SimplifiedDaemon{
		logger:     log,
		statusFile: "/tmp/claude-monitor-status.json",
		pidFile:    "/tmp/claude-monitor.pid",
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (sd *SimplifiedDaemon) Start() error {
	sd.running = true
	
	// Write PID file
	if err := sd.writePidFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	sd.logger.Info("Simplified daemon started")
	
	// Start monitoring loop
	go sd.monitoringLoop()
	
	return nil
}

func (sd *SimplifiedDaemon) Stop() error {
	sd.running = false
	if sd.cancel != nil {
		sd.cancel()
	}
	
	// Remove PID file
	os.Remove(sd.pidFile)
	
	sd.logger.Info("Simplified daemon stopped")
	return nil
}

func (sd *SimplifiedDaemon) writePidFile() error {
	pid := os.Getpid()
	return ioutil.WriteFile(sd.pidFile, []byte(strconv.Itoa(pid)), 0644)
}

func (sd *SimplifiedDaemon) monitoringLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sd.ctx.Done():
			return
		case <-ticker.C:
			if !sd.running {
				return
			}
			sd.checkClaudeProcesses()
			sd.writeStatus()
		}
	}
}

func (sd *SimplifiedDaemon) checkClaudeProcesses() {
	// Check for running Claude processes
	cmd := exec.Command("pgrep", "-f", "claude")
	output, err := cmd.Output()
	if err != nil {
		// No Claude processes found
		if sd.currentSession != nil {
			sd.finalizeCurrentSession()
		}
		return
	}

	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(pids) == 0 || (len(pids) == 1 && pids[0] == "") {
		// No Claude processes
		if sd.currentSession != nil {
			sd.finalizeCurrentSession()
		}
		return
	}

	sd.logger.Info("Claude processes detected", "count", len(pids))

	// If no current session, start one
	if sd.currentSession == nil {
		sd.startNewSession()
	}

	// Update work block activity
	sd.updateWorkBlockActivity()
}

func (sd *SimplifiedDaemon) startNewSession() {
	now := time.Now()
	sd.currentSession = &domain.Session{
		ID:        uuid.New().String(),
		StartTime: now,
		EndTime:   now.Add(5 * time.Hour), // 5-hour window
		IsActive:  true,
	}

	sd.currentWorkBlock = &domain.WorkBlock{
		ID:           uuid.New().String(),
		SessionID:    sd.currentSession.ID,
		StartTime:    now,
		LastActivity: now,
		IsActive:     true,
	}

	sd.logger.Info("New session started", 
		"sessionID", sd.currentSession.ID,
		"endTime", sd.currentSession.EndTime)
}

func (sd *SimplifiedDaemon) updateWorkBlockActivity() {
	now := time.Now()
	
	if sd.currentWorkBlock == nil {
		// Start new work block
		sd.currentWorkBlock = &domain.WorkBlock{
			ID:           uuid.New().String(),
			SessionID:    sd.currentSession.ID,
			StartTime:    now,
			LastActivity: now,
			IsActive:     true,
		}
		return
	}

	// Check if work block timed out (5 minutes)
	if now.Sub(sd.currentWorkBlock.LastActivity) > 5*time.Minute {
		// Finalize current work block
		sd.finalizeCurrentWorkBlock()
		
		// Start new work block
		sd.currentWorkBlock = &domain.WorkBlock{
			ID:           uuid.New().String(),
			SessionID:    sd.currentSession.ID,
			StartTime:    now,
			LastActivity: now,
			IsActive:     true,
		}
	} else {
		// Update activity
		sd.currentWorkBlock.LastActivity = now
	}
}

func (sd *SimplifiedDaemon) finalizeCurrentWorkBlock() {
	if sd.currentWorkBlock != nil {
		now := time.Now()
		sd.currentWorkBlock.Finalize(now)
		sd.logger.Info("Work block finalized", 
			"blockID", sd.currentWorkBlock.ID,
			"duration", sd.currentWorkBlock.Duration())
	}
}

func (sd *SimplifiedDaemon) finalizeCurrentSession() {
	if sd.currentSession != nil {
		sd.finalizeCurrentWorkBlock()
		sd.currentSession.IsActive = false
		sd.logger.Info("Session finalized", "sessionID", sd.currentSession.ID)
		sd.currentSession = nil
		sd.currentWorkBlock = nil
	}
}

func (sd *SimplifiedDaemon) writeStatus() {
	status := map[string]interface{}{
		"daemonRunning": sd.running,
		"timestamp":     time.Now(),
		"currentSession": sd.currentSession,
		"currentWorkBlock": sd.currentWorkBlock,
		"monitoringActive": sd.running,
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		sd.logger.Error("Failed to marshal status", "error", err)
		return
	}

	if err := ioutil.WriteFile(sd.statusFile, data, 0644); err != nil {
		sd.logger.Error("Failed to write status file", "error", err)
	}
}