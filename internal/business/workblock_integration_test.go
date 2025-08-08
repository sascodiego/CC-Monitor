/**
 * CONTEXT:   Integration test for CHECKPOINT 3 work block and project management
 * INPUT:     Test scenarios validating work block lifecycle and project auto-creation
 * OUTPUT:    Comprehensive test coverage for work block manager and server integration
 * BUSINESS:  Ensure work block idle detection, project relationships, and time calculations work correctly
 * CHANGE:    Initial integration test suite for CHECKPOINT 3 validation
 * RISK:      Low - Test code for validating system behavior with realistic scenarios
 */

package business

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
)

/**
 * CONTEXT:   Test complete work block lifecycle with idle detection and project auto-creation
 * INPUT:     Test database and simulated activity events over time
 * OUTPUT:    Validated work block creation, idle detection, and project relationships
 * BUSINESS:  Work blocks should auto-create projects, track activity, and handle idle timeouts correctly
 * CHANGE:    Initial comprehensive work block lifecycle test
 * RISK:      Low - Test validates critical work tracking business logic
 */
func TestWorkBlockLifecycleIntegration(t *testing.T) {
	// Create temporary test database
	testDB := filepath.Join(os.TempDir(), "test_workblock_checkpoint3.db")
	defer os.Remove(testDB)

	// Initialize server integration
	integration, err := NewServerIntegration(testDB)
	if err != nil {
		t.Fatalf("Failed to create server integration: %v", err)
	}
	defer integration.Close()

	ctx := context.Background()
	userID := "test_user"
	projectPath := "/mnt/c/src/CC-Monitor"

	// SCENARIO 1: First activity creates session and work block
	t.Run("FirstActivityCreatesSessionAndWorkBlock", func(t *testing.T) {
		startTime := time.Now()
		
		event := &ActivityEvent{
			UserID:      userID,
			ProjectPath: projectPath,
			Timestamp:   startTime,
			Command:     "claude edit file.go",
			Description: "First activity in new session",
		}

		err := integration.ProcessActivityEvent(ctx, event)
		if err != nil {
			t.Errorf("Failed to process first activity: %v", err)
		}

		// Verify session was created
		session, err := integration.sessionManager.GetActiveSession(ctx, userID)
		if err != nil {
			t.Errorf("Failed to get active session: %v", err)
		}
		if session == nil {
			t.Error("Expected active session to be created")
		}

		// Verify work block was created
		workBlock, err := integration.workBlockManager.GetActiveWorkBlock(ctx, session.ID, projectPath)
		if err != nil {
			t.Errorf("Failed to get active work block: %v", err)
		}
		if workBlock == nil {
			t.Error("Expected active work block to be created")
		}

		// Verify project was auto-created
		project, err := integration.workBlockManager.GetOrCreateProject(ctx, projectPath)
		if err != nil {
			t.Errorf("Failed to get auto-created project: %v", err)
		}
		if project == nil {
			t.Error("Expected project to be auto-created")
		}
		if project.Name != "CC-Monitor" {
			t.Errorf("Expected project name 'CC-Monitor', got '%s'", project.Name)
		}

		t.Logf("✅ First activity: session=%s, work_block=%s, project=%s",
			session.ID, workBlock.ID, project.Name)
	})

	// SCENARIO 2: Additional activities within 5 minutes update existing work block
	t.Run("RecentActivitiesUpdateExistingWorkBlock", func(t *testing.T) {
		// Get current active session and work block
		session, _ := integration.sessionManager.GetActiveSession(ctx, userID)
		workBlockBefore, _ := integration.workBlockManager.GetActiveWorkBlock(ctx, session.ID, projectPath)
		
		// Activity within 5 minutes (2 minutes later)
		recentTime := time.Now().Add(2 * time.Minute)
		
		event := &ActivityEvent{
			UserID:      userID,
			ProjectPath: projectPath,
			Timestamp:   recentTime,
			Command:     "claude read other.go",
			Description: "Recent activity in same work block",
		}

		err := integration.ProcessActivityEvent(ctx, event)
		if err != nil {
			t.Errorf("Failed to process recent activity: %v", err)
		}

		// Verify same work block was updated
		workBlockAfter, err := integration.workBlockManager.GetActiveWorkBlock(ctx, session.ID, projectPath)
		if err != nil {
			t.Errorf("Failed to get updated work block: %v", err)
		}

		if workBlockAfter.ID != workBlockBefore.ID {
			t.Error("Expected same work block to be updated, but got different work block")
		}

		if workBlockAfter.ActivityCount <= workBlockBefore.ActivityCount {
			t.Errorf("Expected activity count to increase, before=%d, after=%d",
				workBlockBefore.ActivityCount, workBlockAfter.ActivityCount)
		}

		t.Logf("✅ Recent activity: work_block=%s, activities=%d",
			workBlockAfter.ID, workBlockAfter.ActivityCount)
	})

	// SCENARIO 3: Activity after 5+ minutes creates new work block
	t.Run("IdleTimeoutCreatesNewWorkBlock", func(t *testing.T) {
		// Get current work block
		session, _ := integration.sessionManager.GetActiveSession(ctx, userID)
		workBlockBefore, _ := integration.workBlockManager.GetActiveWorkBlock(ctx, session.ID, projectPath)
		
		// Activity after idle timeout (6 minutes later)
		idleTime := time.Now().Add(6 * time.Minute)
		
		event := &ActivityEvent{
			UserID:      userID,
			ProjectPath: projectPath,
			Timestamp:   idleTime,
			Command:     "claude generate function",
			Description: "Activity after idle timeout",
		}

		err := integration.ProcessActivityEvent(ctx, event)
		if err != nil {
			t.Errorf("Failed to process activity after idle timeout: %v", err)
		}

		// Verify new work block was created
		workBlockAfter, err := integration.workBlockManager.GetActiveWorkBlock(ctx, session.ID, projectPath)
		if err != nil {
			t.Errorf("Failed to get new work block: %v", err)
		}

		if workBlockAfter.ID == workBlockBefore.ID {
			t.Error("Expected new work block to be created after idle timeout")
		}

		if workBlockAfter.ActivityCount != 1 {
			t.Errorf("Expected new work block to have 1 activity, got %d", workBlockAfter.ActivityCount)
		}

		t.Logf("✅ Idle timeout: old_work_block=%s, new_work_block=%s",
			workBlockBefore.ID, workBlockAfter.ID)
	})

	// SCENARIO 4: Different project creates separate work block
	t.Run("DifferentProjectCreatesSeparateWorkBlock", func(t *testing.T) {
		session, _ := integration.sessionManager.GetActiveSession(ctx, userID)
		workBlockProject1, _ := integration.workBlockManager.GetActiveWorkBlock(ctx, session.ID, projectPath)
		
		// Activity in different project
		differentProjectPath := "/mnt/c/src/OtherProject"
		
		event := &ActivityEvent{
			UserID:      userID,
			ProjectPath: differentProjectPath,
			Timestamp:   time.Now(),
			Command:     "claude edit main.go",
			Description: "Activity in different project",
		}

		err := integration.ProcessActivityEvent(ctx, event)
		if err != nil {
			t.Errorf("Failed to process activity in different project: %v", err)
		}

		// Verify different work block for different project
		workBlockProject2, err := integration.workBlockManager.GetActiveWorkBlock(ctx, session.ID, differentProjectPath)
		if err != nil {
			t.Errorf("Failed to get work block for different project: %v", err)
		}

		if workBlockProject2.ID == workBlockProject1.ID {
			t.Error("Expected different work blocks for different projects")
		}

		// Verify both work blocks exist
		if workBlockProject1.ProjectID == workBlockProject2.ProjectID {
			t.Error("Expected different project IDs for different work blocks")
		}

		// Verify both projects were auto-created
		project1, _ := integration.workBlockManager.GetOrCreateProject(ctx, projectPath)
		project2, _ := integration.workBlockManager.GetOrCreateProject(ctx, differentProjectPath)

		if project1.ID == project2.ID {
			t.Error("Expected different projects to have different IDs")
		}

		t.Logf("✅ Different projects: project1=%s (%s), project2=%s (%s)",
			project1.Name, project1.Path, project2.Name, project2.Path)
	})

	// SCENARIO 5: Session work block summary
	t.Run("SessionWorkBlockSummary", func(t *testing.T) {
		session, _ := integration.sessionManager.GetActiveSession(ctx, userID)
		
		// Get all work blocks for session
		workBlocks, err := integration.workBlockManager.GetWorkBlocksBySession(ctx, session.ID, 0)
		if err != nil {
			t.Errorf("Failed to get session work blocks: %v", err)
		}

		if len(workBlocks) < 2 {
			t.Errorf("Expected at least 2 work blocks in session, got %d", len(workBlocks))
		}

		// Calculate total work time
		totalWorkTime, err := integration.workBlockManager.CalculateSessionWorkTime(ctx, session.ID)
		if err != nil {
			t.Errorf("Failed to calculate session work time: %v", err)
		}

		if totalWorkTime <= 0 {
			t.Errorf("Expected positive work time, got %.2f hours", totalWorkTime)
		}

		t.Logf("✅ Session summary: work_blocks=%d, total_work_time=%.2f hours",
			len(workBlocks), totalWorkTime)

		// Log work block details
		for i, wb := range workBlocks {
			t.Logf("   Work block %d: id=%s, project=%s, duration=%.2f hours, activities=%d",
				i+1, wb.ID, wb.ProjectID, wb.DurationHours, wb.ActivityCount)
		}
	})
}

/**
 * CONTEXT:   Test work block idle detection and cleanup functionality
 * INPUT:     Work blocks in various states requiring idle detection
 * OUTPUT:    Validated idle detection logic and bulk cleanup operations
 * BUSINESS:  Work blocks should be automatically marked as idle after 5 minutes without activity
 * CHANGE:    Initial idle detection test suite
 * RISK:      Low - Test validates critical idle detection business logic
 */
func TestWorkBlockIdleDetection(t *testing.T) {
	// Create temporary test database
	testDB := filepath.Join(os.TempDir(), "test_idle_detection.db")
	defer os.Remove(testDB)

	// Initialize integration
	integration, err := NewServerIntegration(testDB)
	if err != nil {
		t.Fatalf("Failed to create server integration: %v", err)
	}
	defer integration.Close()

	ctx := context.Background()
	userID := "idle_test_user"
	projectPath := "/test/idle/project"

	// Create initial work block
	event := &ActivityEvent{
		UserID:      userID,
		ProjectPath: projectPath,
		Timestamp:   time.Now().Add(-10 * time.Minute), // 10 minutes ago
		Command:     "initial command",
		Description: "Initial activity for idle test",
	}

	err = integration.ProcessActivityEvent(ctx, event)
	if err != nil {
		t.Fatalf("Failed to create initial work block: %v", err)
	}

	session, _ := integration.sessionManager.GetActiveSession(ctx, userID)

	// Test idle detection
	t.Run("MarkIdleWorkBlocks", func(t *testing.T) {
		// Mark idle work blocks
		idleCount, err := integration.workBlockManager.MarkIdleWorkBlocks(ctx)
		if err != nil {
			t.Errorf("Failed to mark idle work blocks: %v", err)
		}

		if idleCount == 0 {
			t.Error("Expected at least one work block to be marked as idle")
		}

		t.Logf("✅ Marked %d work blocks as idle", idleCount)
	})

	// Test that new activity after idle creates new work block
	t.Run("NewActivityAfterIdleCreatesNewBlock", func(t *testing.T) {
		// New activity now
		newEvent := &ActivityEvent{
			UserID:      userID,
			ProjectPath: projectPath,
			Timestamp:   time.Now(),
			Command:     "new command after idle",
			Description: "Activity after idle period",
		}

		err := integration.ProcessActivityEvent(ctx, newEvent)
		if err != nil {
			t.Errorf("Failed to process activity after idle: %v", err)
		}

		// Should have multiple work blocks now
		workBlocks, err := integration.workBlockManager.GetWorkBlocksBySession(ctx, session.ID, 0)
		if err != nil {
			t.Errorf("Failed to get work blocks: %v", err)
		}

		if len(workBlocks) < 2 {
			t.Errorf("Expected at least 2 work blocks after idle detection, got %d", len(workBlocks))
		}

		t.Logf("✅ Created new work block after idle: total_work_blocks=%d", len(workBlocks))
	})
}