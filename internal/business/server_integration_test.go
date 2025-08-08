/**
 * CONTEXT:   Integration tests for server session integration
 * INPUT:     HTTP requests and activity events for session processing
 * OUTPUT:    Test validation of server integration functionality
 * BUSINESS:  Verify server integration maintains API compatibility with new session logic
 * CHANGE:    Integration tests for new server session integration layer
 * RISK:      Low - Test code validating integration functionality
 */

package business

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerSessionIntegration_ProcessActivityEvent(t *testing.T) {
	// Create temporary database for testing
	dbPath := "/tmp/test_server_integration.db"
	defer os.Remove(dbPath)
	
	integration, err := NewServerSessionIntegration(dbPath)
	require.NoError(t, err)
	defer integration.Close()
	
	ctx := context.Background()
	userID := "test_user"
	projectName := "test_project"
	
	// Test activity event processing
	event := &ActivityEvent{
		UserID:      userID,
		ProjectName: projectName,
		ProjectPath: "/test/path",
		Timestamp:   time.Now(),
		Command:     "test command",
		Description: "test activity",
	}
	
	// Process activity event
	err = integration.ProcessActivityEvent(ctx, event)
	require.NoError(t, err)
	
	// Verify session was created
	session, err := integration.sessionManager.GetActiveSession(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, "active", session.State)
	assert.Equal(t, int64(1), session.ActivityCount)
}

func TestServerSessionIntegration_HandleActivity(t *testing.T) {
	// Create temporary database for testing
	dbPath := "/tmp/test_server_http.db"
	defer os.Remove(dbPath)
	
	integration, err := NewServerSessionIntegration(dbPath)
	require.NoError(t, err)
	defer integration.Close()
	
	// Test valid activity request
	event := ActivityEvent{
		UserID:      "test_user",
		ProjectName: "test_project", 
		ProjectPath: "/test/path",
		Timestamp:   time.Now(),
		Command:     "test command",
	}
	
	body, err := json.Marshal(event)
	require.NoError(t, err)
	
	req := httptest.NewRequest("POST", "/activity", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	// Handle request
	integration.HandleActivity(w, req)
	
	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	
	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["status"])
	assert.Equal(t, true, response["processed"])
}

func TestServerSessionIntegration_HandleGetActiveSession(t *testing.T) {
	// Create temporary database for testing
	dbPath := "/tmp/test_server_session_api.db"
	defer os.Remove(dbPath)
	
	integration, err := NewServerSessionIntegration(dbPath)
	require.NoError(t, err)
	defer integration.Close()
	
	ctx := context.Background()
	userID := "test_user"
	
	// Test when no session exists
	req := httptest.NewRequest("GET", "/active-session?user_id="+userID, nil)
	w := httptest.NewRecorder()
	
	integration.HandleGetActiveSession(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["status"])
	assert.Equal(t, false, response["has_active_session"])
	
	// Create a session
	event := &ActivityEvent{
		UserID:      userID,
		ProjectName: "test_project",
		ProjectPath: "/test/path",
		Timestamp:   time.Now(),
	}
	
	err = integration.ProcessActivityEvent(ctx, event)
	require.NoError(t, err)
	
	// Test when session exists
	req = httptest.NewRequest("GET", "/active-session?user_id="+userID, nil)
	w = httptest.NewRecorder()
	
	integration.HandleGetActiveSession(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["status"])
	assert.Equal(t, true, response["has_active_session"])
	
	sessionData := response["session"].(map[string]interface{})
	assert.Equal(t, userID, sessionData["user_id"])
	assert.Equal(t, "active", sessionData["state"])
	assert.Equal(t, float64(1), sessionData["activity_count"])
}

func TestServerSessionIntegration_HandleCleanupExpiredSessions(t *testing.T) {
	// Create temporary database for testing
	dbPath := "/tmp/test_server_cleanup.db"
	defer os.Remove(dbPath)
	
	integration, err := NewServerSessionIntegration(dbPath)
	require.NoError(t, err)
	defer integration.Close()
	
	// Create an expired session manually
	ctx := context.Background()
	expiredTime := time.Now().Add(-6 * time.Hour)
	
	// First create user to avoid foreign key constraint
	err = integration.ensureUserExists(ctx, "expired_user")
	require.NoError(t, err)
	
	_, err = integration.sqliteDB.DB().ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, start_time, end_time, state, first_activity_time, last_activity_time, activity_count, duration_hours, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?, 1, 5.0, ?, ?)
	`, "expired_session", "expired_user", expiredTime, expiredTime.Add(5*time.Hour), expiredTime, expiredTime, expiredTime, expiredTime)
	require.NoError(t, err)
	
	// Test cleanup endpoint
	req := httptest.NewRequest("POST", "/cleanup-expired-sessions", nil)
	w := httptest.NewRecorder()
	
	integration.HandleCleanupExpiredSessions(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "success", response["status"])
	
	// Should have marked 1 session as expired
	expiredCount := response["expired_sessions"].(float64)
	assert.Equal(t, float64(1), expiredCount)
}

func TestServerSessionIntegration_ValidationErrors(t *testing.T) {
	// Create temporary database for testing
	dbPath := "/tmp/test_server_validation.db"
	defer os.Remove(dbPath)
	
	integration, err := NewServerSessionIntegration(dbPath)
	require.NoError(t, err)
	defer integration.Close()
	
	ctx := context.Background()
	
	// Test nil event
	err = integration.ProcessActivityEvent(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "activity event cannot be nil")
	
	// Test missing user ID
	event := &ActivityEvent{
		ProjectName: "test_project",
	}
	err = integration.ProcessActivityEvent(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is required")
	
	// Test missing project name
	event = &ActivityEvent{
		UserID: "test_user",
	}
	err = integration.ProcessActivityEvent(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project name is required")
}

func TestServerSessionIntegration_HTTPErrorHandling(t *testing.T) {
	// Create temporary database for testing
	dbPath := "/tmp/test_server_http_errors.db"
	defer os.Remove(dbPath)
	
	integration, err := NewServerSessionIntegration(dbPath)
	require.NoError(t, err)
	defer integration.Close()
	
	// Test invalid JSON
	req := httptest.NewRequest("POST", "/activity", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()
	
	integration.HandleActivity(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// Test wrong HTTP method
	req = httptest.NewRequest("GET", "/activity", nil)
	w = httptest.NewRecorder()
	
	integration.HandleActivity(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	
	// Test missing user_id parameter in session endpoint
	req = httptest.NewRequest("GET", "/active-session", nil)
	w = httptest.NewRecorder()
	
	integration.HandleGetActiveSession(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}