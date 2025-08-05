package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/claude-monitor/claude-monitor/internal/daemon"
	"github.com/claude-monitor/claude-monitor/pkg/logger"
)

/**
 * AGENT:     daemon-core
 * TRACE:     CLAUDE-COMPLETE-DAEMON-002
 * CONTEXT:   Simplified complete daemon using enhanced daemon with database persistence
 * REASON:    Need production daemon with persistence without complex workhour dependencies
 * CHANGE:    Simplified to use EnhancedDaemon with persistence integration.
 * PREVENTION:Ensure proper cleanup of daemon and database connections on shutdown
 * RISK:      Low - Using proven enhanced daemon with added persistence layer
 */

func main() {
	logger := logger.NewDefaultLogger("claude-daemon-complete", "INFO")
	logger.Info("Claude Monitor Complete Daemon starting with database persistence")

	// Create enhanced daemon with persistence
	enhancedDaemon := daemon.NewEnhancedDaemon(logger)
	
	// Start daemon
	if err := enhancedDaemon.Start(); err != nil {
		logger.Fatal("Failed to start enhanced daemon", "error", err)
		os.Exit(1)
	}

	// Wait for signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	logger.Info("Complete daemon running with database persistence, waiting for signal...")
	sig := <-sigCh
	
	logger.Info("Received signal, shutting down", "signal", sig)
	
	if err := enhancedDaemon.Stop(); err != nil {
		logger.Error("Error stopping daemon", "error", err)
		os.Exit(1)
	}
	
	fmt.Println("Complete daemon stopped successfully")
}