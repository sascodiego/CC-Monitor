/**
 * CONTEXT:   Main daemon orchestrator coordinating all system components and lifecycle
 * INPUT:     System configuration, signal handling, and component coordination requirements
 * OUTPUT:    Running HTTP daemon with coordinated business logic and graceful lifecycle management
 * BUSINESS:  Central orchestration point ensuring reliable operation of Claude Monitor daemon
 * CHANGE:    Initial daemon orchestrator implementation with complete component coordination
 * RISK:      High - Central orchestration point affecting entire system reliability and operation
 */

package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/claude-monitor/system/internal/config"
	"github.com/claude-monitor/system/internal/infrastructure/database"
	httpInfra "github.com/claude-monitor/system/internal/infrastructure/http"
	"github.com/claude-monitor/system/internal/usecases"
)

/**
 * CONTEXT:   Main daemon orchestrator managing complete system lifecycle
 * INPUT:     Configuration, dependencies, and system resources for daemon operation
 * OUTPUT:    Coordinated daemon operation with HTTP server, business logic, and cleanup
 * BUSINESS:  Ensure reliable Claude Monitor operation with proper component coordination
 * CHANGE:    Initial orchestrator implementation with comprehensive lifecycle management
 * RISK:      High - Central coordination point affecting system reliability and data integrity
 */
type Orchestrator struct {
	// Configuration
	config *config.DaemonConfig
	logger *slog.Logger
	
	// Infrastructure
	dbFactory  *database.Factory
	httpServer *http.Server
	
	// Use Cases
	sessionManager   *usecases.SessionManager
	workBlockManager *usecases.WorkBlockManager
	projectManager   *usecases.ProjectManager
	eventProcessor   *usecases.EventProcessor
	
	// HTTP Infrastructure
	handlers *httpInfra.Handlers
	router   *mux.Router
	
	// Lifecycle management
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	startTime  time.Time
	isRunning  bool
	mu         sync.RWMutex
}

// OrchestratorConfig holds configuration for orchestrator initialization
type OrchestratorConfig struct {
	ConfigPath string
	Logger     *slog.Logger
}

/**
 * CONTEXT:   Factory function for creating daemon orchestrator with complete initialization
 * INPUT:     OrchestratorConfig with configuration path and logger
 * OUTPUT:    Fully initialized Orchestrator ready for daemon operation
 * BUSINESS:  Orchestrator requires complete component initialization and dependency injection
 * CHANGE:    Initial factory implementation with comprehensive component setup
 * RISK:      High - Complex initialization affecting all system components
 */
func NewOrchestrator(config OrchestratorConfig) (*Orchestrator, error) {
	// Load daemon configuration
	daemonConfig, err := loadConfiguration(config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Setup logger
	logger := config.Logger
	if logger == nil {
		logger = setupLogger(daemonConfig.Logging)
	}
	
	logger.Info("Initializing Claude Monitor daemon",
		"config_path", config.ConfigPath,
		"listen_addr", daemonConfig.Server.ListenAddr,
		"database_path", daemonConfig.Database.Path)
	
	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())
	
	orchestrator := &Orchestrator{
		config:    daemonConfig,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}
	
	// Initialize all components
	if err := orchestrator.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}
	
	logger.Info("Claude Monitor daemon initialized successfully")
	return orchestrator, nil
}

/**
 * CONTEXT:   Main daemon execution with signal handling and graceful shutdown
 * INPUT:     System signals and operational context for daemon lifecycle
 * OUTPUT:    Running daemon with HTTP server and background processes
 * BUSINESS:  Provide reliable Claude Monitor service with proper error handling and shutdown
 * CHANGE:    Initial daemon run implementation with signal handling and lifecycle management
 * RISK:      High - Main execution loop affecting system availability and reliability
 */
func (o *Orchestrator) Run() error {
	o.mu.Lock()
	o.isRunning = true
	o.mu.Unlock()
	
	o.logger.Info("Starting Claude Monitor daemon",
		"version", "1.0.0",
		"listen_addr", o.config.Server.ListenAddr,
		"pid", os.Getpid())
	
	// Start background processes
	if err := o.startBackgroundProcesses(); err != nil {
		return fmt.Errorf("failed to start background processes: %w", err)
	}
	
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	
	// Start HTTP server
	serverErrChan := make(chan error, 1)
	go o.startHTTPServer(serverErrChan)
	
	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		o.logger.Info("Received shutdown signal", "signal", sig)
		return o.gracefulShutdown()
	case err := <-serverErrChan:
		o.logger.Error("HTTP server error", "error", err)
		return fmt.Errorf("HTTP server failed: %w", err)
	}
}

/**
 * CONTEXT:   Graceful shutdown with proper resource cleanup and data finalization
 * INPUT:     Shutdown context and timeout constraints
 * OUTPUT:    Clean shutdown with finalized work blocks and closed resources
 * BUSINESS:  Ensure data integrity during shutdown with proper work time finalization
 * CHANGE:    Initial graceful shutdown implementation with comprehensive cleanup
 * RISK:      High - Shutdown process affects data integrity and system reliability
 */
func (o *Orchestrator) gracefulShutdown() error {
	o.mu.Lock()
	o.isRunning = false
	o.mu.Unlock()
	
	o.logger.Info("Starting graceful shutdown")
	
	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		o.config.Server.ShutdownTimeout,
	)
	defer shutdownCancel()
	
	var shutdownErrors []error
	
	// Shutdown HTTP server
	if o.httpServer != nil {
		o.logger.Info("Shutting down HTTP server")
		if err := o.httpServer.Shutdown(shutdownCtx); err != nil {
			o.logger.Error("HTTP server shutdown error", "error", err)
			shutdownErrors = append(shutdownErrors, fmt.Errorf("HTTP server shutdown: %w", err))
		}
	}
	
	// Stop event processor (finalizes active work blocks)
	if o.eventProcessor != nil {
		o.logger.Info("Stopping event processor")
		if err := o.eventProcessor.Stop(shutdownCtx); err != nil {
			o.logger.Error("Event processor stop error", "error", err)
			shutdownErrors = append(shutdownErrors, fmt.Errorf("event processor stop: %w", err))
		}
	}
	
	// Cancel context to stop background processes
	o.cancel()
	
	// Wait for background processes to complete
	done := make(chan struct{})
	go func() {
		o.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		o.logger.Info("All background processes stopped")
	case <-shutdownCtx.Done():
		o.logger.Warn("Shutdown timeout exceeded, forcing stop")
		shutdownErrors = append(shutdownErrors, fmt.Errorf("shutdown timeout exceeded"))
	}
	
	// Close database connections
	if o.dbFactory != nil {
		o.logger.Info("Closing database connections")
		if err := o.dbFactory.Close(); err != nil {
			o.logger.Error("Database close error", "error", err)
			shutdownErrors = append(shutdownErrors, fmt.Errorf("database close: %w", err))
		}
	}
	
	// Log shutdown completion
	shutdownDuration := time.Since(o.startTime)
	if len(shutdownErrors) > 0 {
		o.logger.Error("Graceful shutdown completed with errors",
			"duration", shutdownDuration,
			"errors", len(shutdownErrors))
		return fmt.Errorf("shutdown completed with %d errors", len(shutdownErrors))
	}
	
	o.logger.Info("Graceful shutdown completed successfully",
		"duration", shutdownDuration)
	return nil
}

/**
 * CONTEXT:   Check if daemon is currently running
 * INPUT:     No parameters, checks internal running state
 * OUTPUT:    Boolean indicating if daemon is running
 * BUSINESS:  Support status queries and operational decisions
 * CHANGE:    Initial running state check implementation
 * RISK:      Low - Simple state check with no side effects
 */
func (o *Orchestrator) IsRunning() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.isRunning
}

/**
 * CONTEXT:   Get daemon uptime for monitoring and status reporting
 * INPUT:     No parameters, calculates uptime from start time
 * OUTPUT:    Duration representing daemon uptime
 * BUSINESS:  Support monitoring and operational visibility
 * CHANGE:    Initial uptime calculation implementation
 * RISK:      Low - Simple time calculation with no side effects
 */
func (o *Orchestrator) GetUptime() time.Duration {
	return time.Since(o.startTime)
}

// Private initialization and helper methods

/**
 * CONTEXT:   Initialize all system components with proper dependency injection
 * INPUT:     Configuration and logger for component initialization
 * OUTPUT:    Fully initialized system components ready for operation
 * BUSINESS:  Ensure all components are properly configured and connected
 * CHANGE:    Initial component initialization with dependency wiring
 * RISK:      High - Component initialization affects entire system functionality
 */
func (o *Orchestrator) initializeComponents() error {
	// Initialize database factory
	if err := o.initializeDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	
	// Initialize use case managers
	if err := o.initializeUseCases(); err != nil {
		return fmt.Errorf("failed to initialize use cases: %w", err)
	}
	
	// Initialize HTTP infrastructure
	if err := o.initializeHTTPInfrastructure(); err != nil {
		return fmt.Errorf("failed to initialize HTTP infrastructure: %w", err)
	}
	
	return nil
}

func (o *Orchestrator) initializeDatabase() error {
	o.logger.Info("Initializing database", "path", o.config.Database.Path)
	
	factory, err := database.NewFactory(database.FactoryConfig{
		DatabasePath:      o.config.Database.Path,
		ConnectionTimeout: o.config.Database.ConnectionTimeout,
		QueryTimeout:      o.config.Database.QueryTimeout,
		Logger:           o.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create database factory: %w", err)
	}
	
	o.dbFactory = factory
	
	// Test database connection
	if err := factory.TestConnection(); err != nil {
		return fmt.Errorf("database connection test failed: %w", err)
	}
	
	o.logger.Info("Database initialized successfully")
	return nil
}

func (o *Orchestrator) initializeUseCases() error {
	o.logger.Info("Initializing use case managers")
	
	// Create repositories
	sessionRepo, err := o.dbFactory.CreateSessionRepository()
	if err != nil {
		return fmt.Errorf("failed to create session repository: %w", err)
	}
	
	workBlockRepo, err := o.dbFactory.CreateWorkBlockRepository()
	if err != nil {
		return fmt.Errorf("failed to create work block repository: %w", err)
	}
	
	projectRepo, err := o.dbFactory.CreateProjectRepository()
	if err != nil {
		return fmt.Errorf("failed to create project repository: %w", err)
	}
	
	activityRepo, err := o.dbFactory.CreateActivityRepository()
	if err != nil {
		return fmt.Errorf("failed to create activity repository: %w", err)
	}
	
	// Create session manager
	o.sessionManager = usecases.NewSessionManager(sessionRepo, o.logger)
	
	// Create work block manager
	o.workBlockManager, err = usecases.NewWorkBlockManager(usecases.WorkBlockManagerConfig{
		WorkBlockRepo:   workBlockRepo,
		ProjectRepo:     projectRepo,
		Logger:          o.logger,
		IdleTimeout:     o.config.WorkTracking.IdleTimeout,
		CleanupInterval: o.config.WorkTracking.CleanupInterval,
	})
	if err != nil {
		return fmt.Errorf("failed to create work block manager: %w", err)
	}
	
	// Create project manager
	o.projectManager, err = usecases.NewProjectManager(usecases.ProjectManagerConfig{
		ProjectRepo:  projectRepo,
		Logger:       o.logger,
		MaxCacheSize: o.config.Performance.CacheSize,
	})
	if err != nil {
		return fmt.Errorf("failed to create project manager: %w", err)
	}
	
	// Create event processor
	o.eventProcessor, err = usecases.NewEventProcessor(usecases.EventProcessorConfig{
		SessionManager:   o.sessionManager,
		WorkBlockManager: o.workBlockManager,
		ProjectManager:   o.projectManager,
		ActivityRepo:     activityRepo,
		Logger:           o.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create event processor: %w", err)
	}
	
	o.logger.Info("Use case managers initialized successfully")
	return nil
}

func (o *Orchestrator) initializeHTTPInfrastructure() error {
	o.logger.Info("Initializing HTTP infrastructure")
	
	// Create handlers
	handlers, err := httpInfra.NewHandlers(httpInfra.HandlerConfig{
		EventProcessor: o.eventProcessor,
		Logger:         o.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create HTTP handlers: %w", err)
	}
	
	o.handlers = handlers
	
	// Setup router with middleware
	o.setupRouter()
	
	// Create HTTP server
	o.httpServer = &http.Server{
		Addr:           o.config.Server.ListenAddr,
		Handler:        o.router,
		ReadTimeout:    o.config.Server.ReadTimeout,
		WriteTimeout:   o.config.Server.WriteTimeout,
		IdleTimeout:    o.config.Server.IdleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB
	}
	
	o.logger.Info("HTTP infrastructure initialized successfully")
	return nil
}

func (o *Orchestrator) setupRouter() {
	o.router = mux.NewRouter()
	
	// Add middleware
	o.router.Use(httpInfra.RecoveryMiddleware(o.logger))
	o.router.Use(httpInfra.LoggingMiddleware(o.logger))
	o.router.Use(httpInfra.MetricsMiddleware())
	o.router.Use(httpInfra.ValidationMiddleware())
	o.router.Use(httpInfra.CORSMiddleware())
	o.router.Use(httpInfra.RateLimitMiddleware(o.config.Performance.RateLimitRPS))
	o.router.Use(httpInfra.TimeoutMiddleware(30 * time.Second))
	
	// API routes
	api := o.router.PathPrefix("/api/v1").Subrouter()
	
	// Core activity endpoint
	api.HandleFunc("/activity", o.handlers.HandleActivity).Methods("POST")
	
	// Status and health endpoints
	api.HandleFunc("/status", o.handlers.HandleStatus).Methods("GET")
	api.HandleFunc("/sessions", o.handlers.HandleSessions).Methods("GET")
	api.HandleFunc("/work-blocks", o.handlers.HandleWorkBlocks).Methods("GET")
	
	// Health endpoints (also available at root level)
	o.router.HandleFunc("/health", o.handlers.HandleHealth).Methods("GET")
	o.router.HandleFunc("/ready", o.handlers.HandleReady).Methods("GET")
	api.HandleFunc("/health", o.handlers.HandleHealth).Methods("GET")
	api.HandleFunc("/ready", o.handlers.HandleReady).Methods("GET")
	
	// Legacy activity endpoint for hook compatibility
	o.router.HandleFunc("/activity", o.handlers.HandleActivity).Methods("POST")
}

func (o *Orchestrator) startBackgroundProcesses() error {
	o.logger.Info("Starting background processes")
	
	// Start event processor cleanup
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		o.eventProcessor.StartBackgroundCleanup(o.ctx)
	}()
	
	// Start work block manager cleanup
	o.wg.Add(1)
	go func() {
		defer o.wg.Done()
		o.workBlockManager.StartCleanup(o.ctx)
	}()
	
	o.logger.Info("Background processes started")
	return nil
}

func (o *Orchestrator) startHTTPServer(errChan chan<- error) {
	o.logger.Info("Starting HTTP server", "addr", o.config.Server.ListenAddr)
	
	if o.config.Server.TLSEnabled {
		err := o.httpServer.ListenAndServeTLS(
			o.config.Server.TLSCertFile,
			o.config.Server.TLSKeyFile,
		)
		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTPS server failed: %w", err)
		}
	} else {
		err := o.httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}
}

// Configuration and logging helpers

func loadConfiguration(configPath string) (*config.DaemonConfig, error) {
	if configPath != "" {
		return config.LoadDaemonConfig(configPath)
	}
	
	// Try environment variables
	envConfig := config.LoadFromEnvironment()
	
	// Validate configuration
	if err := envConfig.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return envConfig, nil
}

func setupLogger(logConfig config.LoggingConfig) *slog.Logger {
	var level slog.Level
	switch logConfig.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	
	opts := &slog.HandlerOptions{
		Level: level,
	}
	
	var handler slog.Handler
	if logConfig.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	
	return slog.New(handler)
}