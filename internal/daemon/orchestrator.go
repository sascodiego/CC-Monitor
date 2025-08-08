/**
 * CONTEXT:   Simplified daemon orchestrator for production deployment
 * INPUT:     Basic configuration and simple HTTP server startup
 * OUTPUT:    Running daemon with simplified architecture
 * BUSINESS:  Minimal orchestrator for production deployment cleanup
 * CHANGE:    CHECKPOINT 8 - Simplified orchestrator removing complex references
 * RISK:      Low - Simplified implementation during production deployment
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
	"golang.org/x/time/rate"
	
	"github.com/gorilla/mux"
	cfg "github.com/claude-monitor/system/internal/config"
	"github.com/claude-monitor/system/internal/database/sqlite"
)

/**
 * CONTEXT:   Production HTTP daemon orchestrator with rate limiting and monitoring
 * INPUT:     Complete configuration and production dependencies
 * OUTPUT:    Production-ready daemon with HTTP server, database, and monitoring
 * BUSINESS:  Production deployment requires complete, reliable HTTP server
 * CHANGE:    CRITICAL FIX - Complete production HTTP server implementation
 * RISK:      Low - Production-grade implementation with proper error handling
 */
type Orchestrator struct {
	// Configuration
	config *cfg.DaemonConfig
	logger *slog.Logger
	
	// Infrastructure  
	db         *sqlite.SQLiteDB
	httpServer *http.Server
	
	// HTTP Server 
	router      *mux.Router
	rateLimiter *rate.Limiter
	requestMux  sync.Mutex
	
	// Monitoring
	requestCount     int64
	lastRequestTime  time.Time
	connectionCount  int32
	healthStatus     string
	
	// Lifecycle management
	ctx       context.Context
	cancel    context.CancelFunc
	startTime time.Time
	isRunning bool
}

// OrchestratorConfig holds configuration for orchestrator initialization
type OrchestratorConfig struct {
	ConfigPath   string
	Logger       *slog.Logger
	DaemonConfig *cfg.DaemonConfig
}

/**
 * CONTEXT:   Production factory for daemon orchestrator with complete configuration
 * INPUT:     Complete daemon configuration with database and server settings
 * OUTPUT:    Production orchestrator with database, rate limiting, and monitoring
 * BUSINESS:  Production deployment requires complete, reliable daemon initialization
 * CHANGE:    CRITICAL FIX - Complete production initialization with all components
 * RISK:      Medium - Complete initialization affecting all daemon functionality
 */
func NewOrchestrator(config OrchestratorConfig) (*Orchestrator, error) {
	// Create logger if none provided
	logger := config.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}
	
	logger.Info("Initializing production Claude Monitor daemon")
	
	// Use provided daemon config or create default
	daemonConfig := config.DaemonConfig
	if daemonConfig == nil {
		daemonConfig = cfg.NewDefaultConfig()
	}
	
	// Validate configuration
	if err := daemonConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid daemon configuration: %w", err)
	}
	
	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create rate limiter from configuration
	rateLimiter := rate.NewLimiter(rate.Limit(daemonConfig.Performance.RateLimitRPS), daemonConfig.Performance.RateLimitRPS)
	
	// Initialize database connection
	dbConfig := sqlite.DefaultConnectionConfig(daemonConfig.Database.Path)
	dbConfig.MaxOpenConns = daemonConfig.Database.MaxConnections
	dbConfig.MaxIdleConns = daemonConfig.Database.MaxIdleConnections
	dbConfig.ConnMaxLifetime = daemonConfig.Database.ConnectTimeout
	
	db, err := sqlite.NewSQLiteDB(dbConfig)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	
	orchestrator := &Orchestrator{
		config:       daemonConfig,
		logger:       logger,
		db:           db,
		rateLimiter:  rateLimiter,
		ctx:          ctx,
		cancel:       cancel,
		startTime:    time.Now(),
		healthStatus: "initializing",
	}
	
	logger.Info("Production daemon initialized successfully",
		"database", daemonConfig.Database.Path,
		"rate_limit", daemonConfig.Performance.RateLimitRPS)
	return orchestrator, nil
}

/**
 * CONTEXT:   Production daemon execution with complete monitoring and error handling
 * INPUT:     System signals for shutdown and runtime management
 * OUTPUT:    Running production daemon with HTTP server, database, and monitoring
 * BUSINESS:  Provide complete production service for HTTP API and monitoring
 * CHANGE:    CRITICAL FIX - Complete production daemon execution
 * RISK:      Medium - Production daemon affecting all system functionality
 */
func (o *Orchestrator) Run() error {
	o.isRunning = true
	o.healthStatus = "starting"
	
	o.logger.Info("Starting production Claude Monitor daemon",
		"version", "1.0.0",
		"pid", os.Getpid(),
		"database", o.config.Database.Path,
		"listen_addr", fmt.Sprintf("%s:%d", o.config.Server.Host, o.config.Server.Port))
	
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Setup production HTTP server
	if err := o.setupProductionServer(); err != nil {
		return fmt.Errorf("failed to setup HTTP server: %w", err)
	}
	
	// Start HTTP server
	serverErrChan := make(chan error, 1)
	go o.startHTTPServer(serverErrChan)
	
	// Mark as healthy after successful start
	o.healthStatus = "healthy"
	o.logger.Info("Production daemon started successfully",
		"endpoints", []string{"/health", "/status", "/metrics"})
	
	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		o.logger.Info("Received shutdown signal", "signal", sig)
		return o.gracefulShutdown()
	case err := <-serverErrChan:
		o.logger.Error("HTTP server error", "error", err)
		o.healthStatus = "unhealthy"
		return fmt.Errorf("HTTP server failed: %w", err)
	}
}

/**
 * CONTEXT:   Production graceful shutdown with complete resource cleanup
 * INPUT:     Shutdown context with configurable timeout
 * OUTPUT:    Clean shutdown with database, server, and monitoring cleanup
 * BUSINESS:  Ensure complete cleanup during production shutdown
 * CHANGE:    CRITICAL FIX - Complete production shutdown with monitoring
 * RISK:      Low - Essential shutdown handling for production reliability
 */
func (o *Orchestrator) gracefulShutdown() error {
	o.isRunning = false
	o.healthStatus = "shutting_down"
	o.logger.Info("Starting graceful shutdown")
	
	// Create shutdown context with configurable timeout
	shutdownTimeout := o.config.Server.ShutdownTimeout
	if shutdownTimeout == 0 {
		shutdownTimeout = 30 * time.Second
	}
	
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer shutdownCancel()
	
	// Shutdown HTTP server
	if o.httpServer != nil {
		o.logger.Info("Shutting down HTTP server",
			"timeout", shutdownTimeout,
			"active_connections", o.connectionCount)
		
		if err := o.httpServer.Shutdown(shutdownCtx); err != nil {
			o.logger.Error("HTTP server shutdown error", "error", err)
		}
	}
	
	// Cancel context
	o.cancel()
	
	// Close database connections
	if o.db != nil {
		o.logger.Info("Closing database connections")
		if err := o.db.Close(); err != nil {
			o.logger.Error("Database close error", "error", err)
		}
	}
	
	// Log final statistics
	uptime := time.Since(o.startTime)
	o.healthStatus = "stopped"
	
	o.logger.Info("Graceful shutdown completed successfully",
		"uptime", uptime,
		"total_requests", o.requestCount,
		"final_status", o.healthStatus)
	
	return nil
}

/**
 * CONTEXT:   Check if daemon is running
 * INPUT:     No parameters
 * OUTPUT:    Boolean running state
 * BUSINESS:  Support status queries
 * CHANGE:    CHECKPOINT 8 - Basic state check
 * RISK:      Low - Simple state query
 */
func (o *Orchestrator) IsRunning() bool {
	return o.isRunning
}

/**
 * CONTEXT:   Get daemon uptime
 * INPUT:     No parameters  
 * OUTPUT:    Duration since startup
 * BUSINESS:  Support monitoring
 * CHANGE:    CHECKPOINT 8 - Basic uptime calculation
 * RISK:      Low - Simple time calculation
 */
func (o *Orchestrator) GetUptime() time.Duration {
	return time.Since(o.startTime)
}

/**
 * CONTEXT:   Setup production HTTP server with rate limiting and monitoring endpoints
 * INPUT:     Complete daemon configuration with performance and monitoring settings
 * OUTPUT:    Production HTTP server with rate limiting, health checks, and metrics
 * BUSINESS:  Production API requires rate limiting, monitoring, and proper error handling
 * CHANGE:    CRITICAL FIX - Complete production HTTP server with rate limiting
 * RISK:      Medium - Production HTTP server affecting all API functionality
 */
func (o *Orchestrator) setupProductionServer() error {
	o.router = mux.NewRouter()
	
	// Add rate limiting middleware
	o.router.Use(o.rateLimitMiddleware)
	o.router.Use(o.loggingMiddleware)
	o.router.Use(o.metricsMiddleware)
	
	// Health endpoint with database connectivity check
	o.router.HandleFunc("/health", o.handleHealth).Methods("GET")
	
	// Status endpoint with comprehensive daemon information
	o.router.HandleFunc("/status", o.handleStatus).Methods("GET")
	
	// Metrics endpoint for monitoring (basic)
	o.router.HandleFunc("/metrics", o.handleMetrics).Methods("GET")
	
	// Create production HTTP server with timeouts
	listenAddr := fmt.Sprintf("%s:%d", o.config.Server.Host, o.config.Server.Port)
	
	o.httpServer = &http.Server{
		Addr:           listenAddr,
		Handler:        o.router,
		ReadTimeout:    o.config.Server.ReadTimeout,
		WriteTimeout:   o.config.Server.WriteTimeout,
		IdleTimeout:    o.config.Server.IdleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB
	}
	
	return nil
}

/**
 * CONTEXT:   Start HTTP server in background
 * INPUT:     Error channel for server failures
 * OUTPUT:    Running HTTP server
 * BUSINESS:  HTTP server required for daemon operation
 * CHANGE:    CHECKPOINT 8 - Basic server startup
 * RISK:      Low - Simple server startup
 */
func (o *Orchestrator) startHTTPServer(errChan chan<- error) {
	o.logger.Info("Starting HTTP server", "addr", o.httpServer.Addr)
	
	err := o.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		errChan <- fmt.Errorf("HTTP server failed: %w", err)
	}
}