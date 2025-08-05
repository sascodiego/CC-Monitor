package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/arch"
	"github.com/claude-monitor/claude-monitor/internal/daemon"
	"github.com/claude-monitor/claude-monitor/internal/database"
	"github.com/claude-monitor/claude-monitor/internal/ebpf"
	"github.com/claude-monitor/claude-monitor/pkg/logger"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-029
 * CONTEXT:   Main daemon entry point with proper initialization and shutdown handling
 * REASON:    Need single entry point for daemon process with clean startup/shutdown lifecycle
 * CHANGE:    Initial implementation.
 * PREVENTION:Always handle signals gracefully and ensure proper resource cleanup
 * RISK:      High - Improper daemon lifecycle could cause resource leaks or data corruption
 */

func main() {
	// Initialize logger first
	log := logger.NewDefaultLogger("claude-daemon", "INFO")
	
	log.Info("Claude Monitor Daemon starting", "version", "1.0.0")
	
	// Create service container
	container := arch.NewServiceContainer()
	
	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	// Initialize daemon with dependency injection
	daemonSvc, err := initializeDaemon(container, log)
	if err != nil {
		log.Fatal("Failed to initialize daemon", "error", err)
		os.Exit(1)
	}
	
	// Start daemon services
	if err := daemonSvc.Start(); err != nil {
		log.Fatal("Failed to start daemon", "error", err)
		os.Exit(1)
	}
	
	log.Info("Claude Monitor Daemon started successfully")
	
	// Wait for shutdown signal
	select {
	case sig := <-sigCh:
		log.Info("Received shutdown signal", "signal", sig)
	case <-ctx.Done():
		log.Info("Context cancelled")
	}
	
	// Graceful shutdown
	log.Info("Shutting down daemon...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	
	if err := shutdownDaemon(shutdownCtx, daemonSvc, container, log); err != nil {
		log.Error("Error during shutdown", "error", err)
		os.Exit(1)
	}
	
	log.Info("Claude Monitor Daemon stopped")
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-030
 * CONTEXT:   Daemon initialization with dependency injection setup
 * REASON:    Need clean separation of service registration and daemon construction
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate all service dependencies before starting daemon to catch errors early
 * RISK:      Medium - Missing dependencies could cause runtime failures
 */

func initializeDaemon(container *arch.ServiceContainer, log arch.Logger) (arch.DaemonManager, error) {
	// Register logger service
	if err := container.RegisterInstance(log); err != nil {
		return nil, fmt.Errorf("failed to register logger: %w", err)
	}
	
	// Register core services using factories
	if err := registerServices(container, log); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}
	
	// Initialize container (validates all dependencies)
	if err := container.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize container: %w", err)
	}
	
	// Create daemon manager
	daemonSvc, err := daemon.NewDaemonManager(container, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create daemon manager: %w", err)
	}
	
	return daemonSvc, nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-031
 * CONTEXT:   Service registration with factory functions for proper dependency injection
 * REASON:    Need to register all system services with their dependencies in correct order
 * CHANGE:    Initial implementation.
 * PREVENTION:Register services in dependency order to avoid resolution failures
 * RISK:      Medium - Incorrect service registration could cause startup failures
 */

func registerServices(container *arch.ServiceContainer, log arch.Logger) error {
	// Register utility services
	if err := registerUtilityServices(container, log); err != nil {
		return fmt.Errorf("failed to register utility services: %w", err)
	}
	
	// Register database manager
	if err := container.RegisterSingleton((*arch.DatabaseManager)(nil), database.NewKuzuManagerFactory); err != nil {
		return fmt.Errorf("failed to register database manager: %w", err)
	}
	
	// Register eBPF manager (production implementation)
	if err := container.RegisterSingleton((*arch.EBPFManager)(nil), func(c *arch.ServiceContainer) (interface{}, error) {
		logger, err := c.GetLogger()
		if err != nil {
			return nil, err
		}
		
		// Use production eBPF manager with kernel-level monitoring
		return ebpf.NewManager(logger), nil
	}); err != nil {
		return fmt.Errorf("failed to register eBPF manager: %w", err)
	}
	
	// Register session manager
	if err := container.RegisterSingleton((*arch.SessionManager)(nil), func(c *arch.ServiceContainer) (interface{}, error) {
		logger, err := c.GetLogger()
		if err != nil {
			return nil, err
		}
		
		dbManager, err := c.GetDatabaseManager()
		if err != nil {
			return nil, err
		}
		
		timeProvider, err := c.Get((*arch.TimeProvider)(nil))
		if err != nil {
			return nil, err
		}
		
		// Session manager will be implemented by daemon-core agent
		return daemon.NewSessionManager(dbManager, timeProvider.(arch.TimeProvider), logger), nil
	}); err != nil {
		return fmt.Errorf("failed to register session manager: %w", err)
	}
	
	// Register work block manager
	if err := container.RegisterSingleton((*arch.WorkBlockManager)(nil), func(c *arch.ServiceContainer) (interface{}, error) {
		logger, err := c.GetLogger()
		if err != nil {
			return nil, err
		}
		
		dbManager, err := c.GetDatabaseManager()
		if err != nil {
			return nil, err
		}
		
		timeProvider, err := c.Get((*arch.TimeProvider)(nil))
		if err != nil {
			return nil, err
		}
		
		// Work block manager will be implemented by daemon-core agent
		return daemon.NewWorkBlockManager(dbManager, timeProvider.(arch.TimeProvider), logger), nil
	}); err != nil {
		return fmt.Errorf("failed to register work block manager: %w", err)
	}
	
	return nil
}

func registerUtilityServices(container *arch.ServiceContainer, log arch.Logger) error {
	// Register time provider
	if err := container.RegisterInstance(arch.TimeProvider(daemon.NewDefaultTimeProvider())); err != nil {
		return fmt.Errorf("failed to register time provider: %w", err)
	}
	
	// Register system provider
	if err := container.RegisterInstance(arch.SystemProvider(daemon.NewDefaultSystemProvider())); err != nil {
		return fmt.Errorf("failed to register system provider: %w", err)
	}
	
	return nil
}

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-032
 * CONTEXT:   Graceful shutdown with proper resource cleanup and timeout handling
 * REASON:    Need clean shutdown process that ensures all resources are properly released
 * CHANGE:    Initial implementation.
 * PREVENTION:Always use timeouts for shutdown operations to prevent hanging processes
 * RISK:      Medium - Improper shutdown could cause resource leaks or data corruption
 */

func shutdownDaemon(ctx context.Context, daemonSvc arch.DaemonManager, container *arch.ServiceContainer, log arch.Logger) error {
	// Create shutdown channel for coordination
	shutdownComplete := make(chan error, 1)
	
	go func() {
		// Stop daemon services
		if err := daemonSvc.Stop(); err != nil {
			shutdownComplete <- fmt.Errorf("daemon stop error: %w", err)
			return
		}
		
		// Dispose of container resources
		if err := container.Dispose(); err != nil {
			shutdownComplete <- fmt.Errorf("container disposal error: %w", err)
			return
		}
		
		shutdownComplete <- nil
	}()
	
	// Wait for shutdown completion or timeout
	select {
	case err := <-shutdownComplete:
		if err != nil {
			return err
		}
		log.Info("Graceful shutdown completed")
		return nil
		
	case <-ctx.Done():
		log.Warn("Shutdown timeout exceeded, forcing exit")
		return fmt.Errorf("shutdown timeout exceeded")
	}
}