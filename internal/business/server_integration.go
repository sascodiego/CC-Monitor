/**
 * CONTEXT:   Server integration layer for CHECKPOINT 3 with work block and project management
 * INPUT:     HTTP requests and activity events requiring session and work block management
 * OUTPUT:    Processed activities with complete SQLite persistence for sessions, work blocks, and projects
 * BUSINESS:  Complete work tracking with session management, project auto-creation, and idle detection
 * CHANGE:    Enhanced integration layer adding work block management to session processing
 * RISK:      Medium - Core server functionality integration affecting complete activity processing
 */

package business

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

// ServerIntegration provides complete work tracking integration for HTTP server
type ServerIntegration struct {
	sessionManager   *SessionManager
	workBlockManager *WorkBlockManager
	sqliteDB         *sqlite.SQLiteDB
	sessionRepo      *sqlite.SessionRepository
	projectRepo      *sqlite.ProjectRepository
	workBlockRepo    *sqlite.WorkBlockRepository
	timezone         *time.Location
}


/**
 * CONTEXT:   Create complete server integration with SQLite database for work tracking
 * INPUT:     SQLite database path for complete work tracking storage
 * OUTPUT:    Configured integration layer ready for complete activity processing
 * BUSINESS:  Single integration point for all work tracking operations including sessions, work blocks, and projects
 * CHANGE:    Enhanced integration layer with work block and project management
 * RISK:      Low - Clean integration layer with proper error handling and dependency injection
 */
func NewServerIntegration(dbPath string) (*ServerIntegration, error) {
	// Setup SQLite database
	config := sqlite.DefaultConnectionConfig(dbPath)
	config.Timezone = "America/Montevideo"
	
	db, err := sqlite.NewSQLiteDB(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQLite database: %w", err)
	}
	
	if err := db.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize SQLite database: %w", err)
	}
	
	// Create all repositories
	sessionRepo := sqlite.NewSessionRepository(db)
	projectRepo := sqlite.NewProjectRepository(db.DB())
	workBlockRepo := sqlite.NewWorkBlockRepository(db.DB())
	activityRepo := sqlite.NewActivityRepository(db.DB())
	
	// Create managers
	sessionManager := NewSessionManager(sessionRepo)
	workBlockManager := NewWorkBlockManager(workBlockRepo, projectRepo, activityRepo)
	
	// Load timezone
	timezone, err := time.LoadLocation("America/Montevideo")
	if err != nil {
		log.Printf("Warning: failed to load timezone America/Montevideo, using UTC: %v", err)
		timezone = time.UTC
	}
	
	return &ServerIntegration{
		sessionManager:   sessionManager,
		workBlockManager: workBlockManager,
		sqliteDB:         db,
		sessionRepo:      sessionRepo,
		projectRepo:      projectRepo,
		workBlockRepo:    workBlockRepo,
		timezone:         timezone,
	}, nil
}

/**
 * CONTEXT:   Process activity event with complete work tracking including sessions and work blocks
 * INPUT:     Activity event from HTTP request with user, project, and timing information
 * OUTPUT:    Complete work tracking updated in SQLite database including session, work block, and project
 * BUSINESS:  Core activity processing with session management, work block creation/update, and project auto-creation
 * CHANGE:    Enhanced activity processing with work block management and idle detection
 * RISK:      Medium - Core activity processing affecting complete user work tracking and time calculations
 */
func (si *ServerIntegration) ProcessActivityEvent(ctx context.Context, event *ActivityEvent) error {
	if event == nil {
		return fmt.Errorf("activity event cannot be nil")
	}
	
	// Validate required fields
	if event.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	
	// Use ProjectPath if available, otherwise use ProjectName
	projectPath := event.ProjectPath
	if projectPath == "" && event.ProjectName != "" {
		projectPath = fmt.Sprintf("/unknown/%s", event.ProjectName)
	}
	if projectPath == "" {
		return fmt.Errorf("project path or project name is required")
	}
	
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().In(si.timezone)
	}
	
	// Convert timestamp to timezone
	event.Timestamp = event.Timestamp.In(si.timezone)
	
	log.Printf("📝 Processing complete activity: user=%s, path=%s, time=%s", 
		event.UserID, projectPath, event.Timestamp.Format("2006-01-02 15:04:05"))
	
	// STEP 1: Create/ensure user exists in database
	if err := si.ensureUserExists(ctx, event.UserID); err != nil {
		return fmt.Errorf("failed to ensure user exists: %w", err)
	}
	
	// STEP 2: Get or create session using session manager
	session, err := si.sessionManager.GetOrCreateSession(ctx, event.UserID, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to get or create session: %w", err)
	}
	
	// STEP 3: Process work block activity (includes project auto-creation)
	workBlock, err := si.workBlockManager.ProcessActivity(ctx, session.ID, projectPath, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to process work block activity: %w", err)
	}
	
	log.Printf("✅ Complete activity processed: session=%s, work_block=%s, activities=%d, work_hours=%.2f", 
		session.ID, workBlock.ID, session.ActivityCount, workBlock.DurationHours)
	
	return nil
}

/**
 * CONTEXT:   HTTP handler for activity events with complete work tracking integration
 * INPUT:     HTTP POST request with JSON activity event
 * OUTPUT:    HTTP response with complete activity processing status including work blocks
 * BUSINESS:  Maintain API compatibility while using complete work tracking integration
 * CHANGE:    Enhanced HTTP handler using complete work tracking integration layer
 * RISK:      Low - HTTP handler with proper error handling and validation
 */
func (si *ServerIntegration) HandleActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var event ActivityEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("❌ Invalid JSON in activity request: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	
	// Use request context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Process activity event
	if err := si.ProcessActivityEvent(ctx, &event); err != nil {
		log.Printf("❌ Failed to process activity: %v", err)
		http.Error(w, "Failed to process activity", http.StatusInternalServerError)
		return
	}
	
	// Return success response
	response := map[string]interface{}{
		"status":    "success",
		"timestamp": time.Now().In(si.timezone).Format(time.RFC3339),
		"processed": true,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

/**
 * CONTEXT:   Get current active session for user (API endpoint)
 * INPUT:     HTTP GET request with user ID parameter
 * OUTPUT:    JSON response with active session information
 * BUSINESS:  Provide session status API for monitoring and debugging
 * CHANGE:    Session status API using new time-based session logic
 * RISK:      Low - Read-only API endpoint for session information
 */
func (si *ServerIntegration) HandleGetActiveSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id parameter required", http.StatusBadRequest)
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Get active session
	session, err := si.sessionManager.GetActiveSession(ctx, userID)
	if err != nil {
		log.Printf("❌ Failed to get active session: %v", err)
		http.Error(w, "Failed to get active session", http.StatusInternalServerError)
		return
	}
	
	var response map[string]interface{}
	if session == nil {
		response = map[string]interface{}{
			"status":         "success",
			"has_active_session": false,
			"message":        "No active session found",
		}
	} else {
		response = map[string]interface{}{
			"status":         "success",
			"has_active_session": true,
			"session": map[string]interface{}{
				"id":              session.ID,
				"user_id":         session.UserID,
				"start_time":      session.StartTime.Format(time.RFC3339),
				"end_time":        session.EndTime.Format(time.RFC3339),
				"activity_count":  session.ActivityCount,
				"state":           session.State,
				"duration_hours":  session.DurationHours,
			},
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

/**
 * CONTEXT:   Cleanup expired sessions (maintenance endpoint)
 * INPUT:     HTTP POST request to trigger session cleanup
 * OUTPUT:    JSON response with cleanup results
 * BUSINESS:  Manual trigger for session cleanup and maintenance
 * CHANGE:    Session cleanup API using new session manager
 * RISK:      Low - Maintenance operation with proper error handling
 */
func (si *ServerIntegration) HandleCleanupExpiredSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Mark expired sessions
	expiredCount, err := si.sessionManager.MarkExpiredSessions(ctx)
	if err != nil {
		log.Printf("❌ Failed to mark expired sessions: %v", err)
		http.Error(w, "Failed to cleanup expired sessions", http.StatusInternalServerError)
		return
	}
	
	// Also mark idle work blocks
	idleCount, err := si.workBlockManager.MarkIdleWorkBlocks(ctx)
	if err != nil {
		log.Printf("Warning: failed to mark idle work blocks: %v", err)
	}
	
	response := map[string]interface{}{
		"status":           "success",
		"expired_sessions": expiredCount,
		"idle_work_blocks": idleCount,
		"timestamp":        time.Now().In(si.timezone).Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	
	log.Printf("🧹 Session cleanup completed: %d sessions marked as expired", expiredCount)
}

/**
 * CONTEXT:   Database connection closure for cleanup
 * INPUT:     Shutdown signal requiring clean resource cleanup
 * OUTPUT:    Closed database connections and released resources
 * BUSINESS:  Proper resource cleanup preventing database connection leaks
 * CHANGE:    Resource cleanup for SQLite integration layer
 * RISK:      Low - Standard resource cleanup operation
 */
func (si *ServerIntegration) Close() error {
	if si.sqliteDB != nil {
		return si.sqliteDB.Close()
	}
	return nil
}

// Helper functions for database operations

/**
 * CONTEXT:   Get work block status for session and project combination
 * INPUT:     HTTP GET request with session_id and project_path parameters
 * OUTPUT:    JSON response with active work block information if exists
 * BUSINESS:  Work block status API for monitoring active work periods
 * CHANGE:    New work block status endpoint for debugging and monitoring
 * RISK:      Low - Read-only API endpoint for work block information
 */
func (si *ServerIntegration) HandleGetWorkBlockStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	sessionID := r.URL.Query().Get("session_id")
	projectPath := r.URL.Query().Get("project_path")
	
	if sessionID == "" {
		http.Error(w, "session_id parameter required", http.StatusBadRequest)
		return
	}
	
	if projectPath == "" {
		http.Error(w, "project_path parameter required", http.StatusBadRequest)
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Get active work block
	workBlock, err := si.workBlockManager.GetActiveWorkBlock(ctx, sessionID, projectPath)
	if err != nil {
		log.Printf("❌ Failed to get work block status: %v", err)
		http.Error(w, "Failed to get work block status", http.StatusInternalServerError)
		return
	}
	
	var response map[string]interface{}
	if workBlock == nil {
		response = map[string]interface{}{
			"status":               "success",
			"has_active_work_block": false,
			"message":              "No active work block found",
		}
	} else {
		response = map[string]interface{}{
			"status":               "success",
			"has_active_work_block": true,
			"work_block": map[string]interface{}{
				"id":                    workBlock.ID,
				"session_id":           workBlock.SessionID,
				"project_id":           workBlock.ProjectID,
				"start_time":           workBlock.StartTime.Format(time.RFC3339),
				"state":                workBlock.State,
				"last_activity_time":   workBlock.LastActivityTime.Format(time.RFC3339),
				"activity_count":       workBlock.ActivityCount,
				"duration_hours":       workBlock.DurationHours,
				"duration_seconds":     workBlock.DurationSeconds,
			},
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

/**
 * CONTEXT:   List all work blocks for a session for reporting and analysis
 * INPUT:     HTTP GET request with session_id parameter and optional limit
 * OUTPUT:    JSON response with array of work blocks for the session
 * BUSINESS:  Session work block listing for productivity analysis and debugging
 * CHANGE:    New work block listing endpoint for session analysis
 * RISK:      Low - Read-only API endpoint for work block reporting
 */
func (si *ServerIntegration) HandleGetSessionWorkBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id parameter required", http.StatusBadRequest)
		return
	}
	
	// Optional limit parameter
	limit := 50 // Default limit
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Get work blocks for session
	workBlocks, err := si.workBlockManager.GetWorkBlocksBySession(ctx, sessionID, limit)
	if err != nil {
		log.Printf("❌ Failed to get session work blocks: %v", err)
		http.Error(w, "Failed to get session work blocks", http.StatusInternalServerError)
		return
	}
	
	// Calculate total work time
	totalWorkTime, err := si.workBlockManager.CalculateSessionWorkTime(ctx, sessionID)
	if err != nil {
		log.Printf("Warning: failed to calculate session work time: %v", err)
		totalWorkTime = 0
	}
	
	response := map[string]interface{}{
		"status":           "success",
		"session_id":       sessionID,
		"work_block_count": len(workBlocks),
		"total_work_hours": totalWorkTime,
		"work_blocks":      workBlocks,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Helper functions for database operations

func (si *ServerIntegration) ensureUserExists(ctx context.Context, userID string) error {
	_, err := si.sqliteDB.DB().ExecContext(ctx, 
		"INSERT OR IGNORE INTO users (id, username) VALUES (?, ?)", 
		userID, userID)
	return err
}