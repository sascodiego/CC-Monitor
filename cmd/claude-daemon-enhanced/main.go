package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/claude-monitor/claude-monitor/internal/daemon"
	"github.com/claude-monitor/claude-monitor/pkg/logger"
)

func main() {
	log := logger.NewDefaultLogger("claude-daemon-enhanced", "INFO")
	
	log.Info("Claude Monitor Enhanced Daemon starting with activity detection")
	
	// Create enhanced daemon with real activity monitoring
	d := daemon.NewEnhancedDaemon(log)
	
	// Start daemon
	if err := d.Start(); err != nil {
		log.Fatal("Failed to start daemon", "error", err)
		os.Exit(1)
	}
	
	// Wait for signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	log.Info("Enhanced daemon running with inactivity detection, waiting for signal...")
	sig := <-sigCh
	
	log.Info("Received signal, shutting down", "signal", sig)
	
	if err := d.Stop(); err != nil {
		log.Error("Error stopping daemon", "error", err)
		os.Exit(1)
	}
	
	fmt.Println("Enhanced daemon stopped successfully")
}