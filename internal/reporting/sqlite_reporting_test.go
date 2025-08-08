/**
 * CONTEXT:   Comprehensive test suite for SQLite reporting system validation
 * INPUT:     Test database with sample work blocks, sessions, and activities
 * OUTPUT:    Complete validation of SQLite-only reporting functionality
 * BUSINESS:  Testing ensures reliable reporting system for production use
 * CHANGE:    Initial comprehensive test suite for SQLite reporting validation
 * RISK:      Low - Testing infrastructure with no production impact
 */

package reporting

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/claude-monitor/system/internal/database/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

// Test database setup and teardown
func setupTestDatabase(t *testing.T) (*sqlite.SQLiteDB, func()) {
	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "test_claude_monitor_*.db")
	if err != nil {
		t.Fatalf("Failed to create temporary database file: %v", err)
	}
	tmpFile.Close()

	// Open database connection
	db, err := sql.Open("sqlite3", tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create SQLiteDB wrapper
	sqliteDB := &sqlite.SQLiteDB{}
	sqliteDB.SetDB(db)

	// Create tables
	if err := createTestTables(db); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return sqliteDB, cleanup
}

func createTestTables(db *sql.DB) error {
	// Create sessions table
	sessionsSQL := `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			start_time TEXT NOT NULL,
			end_time TEXT NOT NULL,
			state TEXT NOT NULL,
			first_activity_time TEXT,
			last_activity_time TEXT,
			activity_count INTEGER DEFAULT 0,
			duration_hours REAL DEFAULT 5.0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
	`

	// Create projects table
	projectsSQL := `
		CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			normalized_path TEXT NOT NULL,
			project_type TEXT DEFAULT 'general',
			description TEXT DEFAULT '',
			last_active_time TEXT NOT NULL,
			total_work_blocks INTEGER DEFAULT 0,
			total_hours REAL DEFAULT 0.0,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
	`

	// Create work_blocks table
	workBlocksSQL := `
		CREATE TABLE IF NOT EXISTS work_blocks (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			start_time TEXT NOT NULL,
			end_time TEXT,
			state TEXT NOT NULL,
			last_activity_time TEXT NOT NULL,
			activity_count INTEGER DEFAULT 0,
			duration_seconds INTEGER DEFAULT 0,
			duration_hours REAL DEFAULT 0.0,
			claude_processing_seconds INTEGER DEFAULT 0,
			claude_processing_hours REAL DEFAULT 0.0,
			estimated_end_time TEXT,
			last_claude_activity TEXT,
			active_prompt_id TEXT DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(id),
			FOREIGN KEY (project_id) REFERENCES projects(id)
		);
	`

	// Create activities table
	activitiesSQL := `
		CREATE TABLE IF NOT EXISTS activities (
			id TEXT PRIMARY KEY,
			work_block_id TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			command TEXT DEFAULT '',
			description TEXT DEFAULT '',
			metadata TEXT DEFAULT '{}',
			created_at TEXT NOT NULL,
			FOREIGN KEY (work_block_id) REFERENCES work_blocks(id)
		);
	`

	tables := []string{sessionsSQL, projectsSQL, workBlocksSQL, activitiesSQL}
	for _, tableSQL := range tables {
		if _, err := db.Exec(tableSQL); err != nil {
			return err
		}
	}

	return nil
}

func insertTestData(t *testing.T, db *sqlite.SQLiteDB) {
	// Insert test session
	session := &sqlite.Session{
		ID:                "test-session-1",
		UserID:           "test-user",
		StartTime:        time.Now().Add(-4 * time.Hour),
		EndTime:          time.Now().Add(1 * time.Hour),
		State:            "active",
		FirstActivityTime: time.Now().Add(-3 * time.Hour),
		LastActivityTime:  time.Now().Add(-30 * time.Minute),
		ActivityCount:    25,
		DurationHours:    5.0,
		CreatedAt:        time.Now().Add(-4 * time.Hour),
		UpdatedAt:        time.Now().Add(-30 * time.Minute),
	}

	sessionRepo := sqlite.NewSessionRepository(db)
	if err := sessionRepo.Create(context.Background(), session); err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Insert test project
	project := &sqlite.Project{
		ID:             "test-project-1",
		Name:           "Test Project",
		Path:           "/test/project",
		NormalizedPath: "/test/project",
		ProjectType:    "general",
		Description:    "Test project for reporting",
		LastActiveTime: time.Now().Add(-30 * time.Minute),
		TotalWorkBlocks: 3,
		TotalHours:     2.5,
		IsActive:       true,
		CreatedAt:      time.Now().Add(-4 * time.Hour),
		UpdatedAt:      time.Now().Add(-30 * time.Minute),
	}

	projectRepo := sqlite.NewProjectRepository(db)
	if err := projectRepo.Create(context.Background(), project); err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	// Insert test work blocks
	workBlockRepo := sqlite.NewWorkBlockRepository(db)
	
	workBlocks := []*sqlite.WorkBlock{
		{
			ID:               "test-wb-1",
			SessionID:        session.ID,
			ProjectID:        project.ID,
			StartTime:        time.Now().Add(-3 * time.Hour),
			EndTime:          timePtr(time.Now().Add(-2*time.Hour + -30*time.Minute)),
			State:            "finished",
			LastActivityTime: time.Now().Add(-2*time.Hour + -30*time.Minute),
			ActivityCount:    10,
			DurationSeconds:  1800, // 30 minutes
			DurationHours:    0.5,
			CreatedAt:        time.Now().Add(-3 * time.Hour),
			UpdatedAt:        time.Now().Add(-2*time.Hour + -30*time.Minute),
		},
		{
			ID:               "test-wb-2",
			SessionID:        session.ID,
			ProjectID:        project.ID,
			StartTime:        time.Now().Add(-2 * time.Hour),
			EndTime:          timePtr(time.Now().Add(-1 * time.Hour)),
			State:            "finished",
			LastActivityTime: time.Now().Add(-1 * time.Hour),
			ActivityCount:    8,
			DurationSeconds:  3600, // 60 minutes
			DurationHours:    1.0,
			CreatedAt:        time.Now().Add(-2 * time.Hour),
			UpdatedAt:        time.Now().Add(-1 * time.Hour),
		},
		{
			ID:               "test-wb-3",
			SessionID:        session.ID,
			ProjectID:        project.ID,
			StartTime:        time.Now().Add(-1 * time.Hour),
			EndTime:          nil, // Active work block
			State:            "active",
			LastActivityTime: time.Now().Add(-30 * time.Minute),
			ActivityCount:    7,
			DurationSeconds:  1800, // 30 minutes so far
			DurationHours:    0.5,
			CreatedAt:        time.Now().Add(-1 * time.Hour),
			UpdatedAt:        time.Now().Add(-30 * time.Minute),
		},
	}

	for _, wb := range workBlocks {
		if err := workBlockRepo.Create(context.Background(), wb); err != nil {
			t.Fatalf("Failed to create test work block %s: %v", wb.ID, err)
		}
	}

	// Insert test activities
	activityRepo := sqlite.NewActivityRepository(db.DB())
	
	activities := []*sqlite.Activity{
		{
			ID:           "test-activity-1",
			WorkBlockID:  "test-wb-1",
			Timestamp:    time.Now().Add(-3 * time.Hour),
			ActivityType: "command",
			Command:      "claude-code",
			Description:  "Started work session",
			Metadata:     map[string]string{"type": "session_start"},
			CreatedAt:    time.Now().Add(-3 * time.Hour),
		},
		{
			ID:           "test-activity-2",
			WorkBlockID:  "test-wb-1",
			Timestamp:    time.Now().Add(-2*time.Hour + -45*time.Minute),
			ActivityType: "edit",
			Command:      "file-edit",
			Description:  "Edited source code",
			Metadata:     map[string]string{"file": "main.go", "lines": "25"},
			CreatedAt:    time.Now().Add(-2*time.Hour + -45*time.Minute),
		},
		{
			ID:           "test-activity-3",
			WorkBlockID:  "test-wb-2",
			Timestamp:    time.Now().Add(-2 * time.Hour),
			ActivityType: "query",
			Command:      "claude-query",
			Description:  "Asked Claude for help",
			Metadata:     map[string]string{"topic": "debugging", "response_time": "15s"},
			CreatedAt:    time.Now().Add(-2 * time.Hour),
		},
	}

	for _, activity := range activities {
		if err := activityRepo.SaveActivity(activity); err != nil {
			t.Fatalf("Failed to create test activity %s: %v", activity.ID, err)
		}
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

/**
 * CONTEXT:   Test daily report generation with SQLite data sources
 * INPUT:     Test database with sample work data
 * OUTPUT:    Validation of daily report accuracy and completeness
 * BUSINESS:  Daily reports are primary user interface requiring reliable functionality
 * CHANGE:    Initial test for daily report generation
 * RISK:      Low - Test validation with no production impact
 */
func TestSQLiteReportingService_GenerateDailyReport(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Insert test data
	insertTestData(t, db)

	// Initialize repositories
	sessionRepo := sqlite.NewSessionRepository(db)
	workBlockRepo := sqlite.NewWorkBlockRepository(db)
	activityRepo := sqlite.NewActivityRepository(db.DB())
	projectRepo := sqlite.NewProjectRepository(db)

	// Create reporting service
	reportingService := NewSQLiteReportingService(
		sessionRepo,
		workBlockRepo,
		activityRepo,
		projectRepo,
	)

	// Generate daily report
	ctx := context.Background()
	today := time.Now()
	report, err := reportingService.GenerateDailyReport(ctx, "test-user", today)

	// Validate report
	if err != nil {
		t.Fatalf("Failed to generate daily report: %v", err)
	}

	if report == nil {
		t.Fatal("Daily report is nil")
	}

	// Validate basic metrics
	if report.TotalSessions != 1 {
		t.Errorf("Expected 1 session, got %d", report.TotalSessions)
	}

	if report.TotalWorkBlocks != 3 {
		t.Errorf("Expected 3 work blocks, got %d", report.TotalWorkBlocks)
	}

	// Validate work hours calculation (2 hours from finished blocks)
	expectedHours := 2.0 // 0.5 + 1.0 + 0.5 (active block calculated to now)
	if report.TotalWorkHours < expectedHours-0.1 || report.TotalWorkHours > expectedHours+1.0 {
		t.Errorf("Expected approximately %.1f work hours, got %.1f", expectedHours, report.TotalWorkHours)
	}

	// Validate project breakdown
	if len(report.ProjectBreakdown) != 1 {
		t.Errorf("Expected 1 project, got %d", len(report.ProjectBreakdown))
	} else {
		project := report.ProjectBreakdown[0]
		if project.Name != "Test Project" {
			t.Errorf("Expected project name 'Test Project', got '%s'", project.Name)
		}
		if project.WorkBlocks != 3 {
			t.Errorf("Expected 3 work blocks for project, got %d", project.WorkBlocks)
		}
	}

	// Validate work blocks
	if len(report.WorkBlocks) != 3 {
		t.Errorf("Expected 3 work blocks in report, got %d", len(report.WorkBlocks))
	}

	t.Logf("Daily report generated successfully with %.1f hours across %d work blocks", 
		report.TotalWorkHours, report.TotalWorkBlocks)
}

/**
 * CONTEXT:   Test weekly report generation with SQLite data sources
 * INPUT:     Test database with sample work data across multiple days
 * OUTPUT:    Validation of weekly report aggregation and trends
 * BUSINESS:  Weekly reports show productivity patterns requiring accurate aggregation
 * CHANGE:    Initial test for weekly report generation
 * RISK:      Low - Test validation with no production impact
 */
func TestSQLiteReportingService_GenerateWeeklyReport(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Insert test data
	insertTestData(t, db)

	// Initialize repositories
	sessionRepo := sqlite.NewSessionRepository(db)
	workBlockRepo := sqlite.NewWorkBlockRepository(db)
	activityRepo := sqlite.NewActivityRepository(db.DB())
	projectRepo := sqlite.NewProjectRepository(db)

	// Create reporting service
	reportingService := NewSQLiteReportingService(
		sessionRepo,
		workBlockRepo,
		activityRepo,
		projectRepo,
	)

	// Generate weekly report (start of current week)
	ctx := context.Background()
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	
	report, err := reportingService.GenerateWeeklyReport(ctx, "test-user", weekStart)

	// Validate report
	if err != nil {
		t.Fatalf("Failed to generate weekly report: %v", err)
	}

	if report == nil {
		t.Fatal("Weekly report is nil")
	}

	// Validate week metadata
	if report.WeekNumber == 0 {
		t.Error("Week number should be set")
	}

	if report.Year != now.Year() {
		t.Errorf("Expected year %d, got %d", now.Year(), report.Year)
	}

	// Validate daily breakdown (7 days)
	if len(report.DailyBreakdown) != 7 {
		t.Errorf("Expected 7 days in breakdown, got %d", len(report.DailyBreakdown))
	}

	// Validate project breakdown
	if len(report.ProjectBreakdown) > 0 {
		if report.ProjectBreakdown[0].Name != "Test Project" {
			t.Errorf("Expected project 'Test Project', got '%s'", report.ProjectBreakdown[0].Name)
		}
	}

	t.Logf("Weekly report generated successfully for week %d with %.1f total hours", 
		report.WeekNumber, report.TotalWorkHours)
}

/**
 * CONTEXT:   Test monthly report generation with SQLite data sources
 * INPUT:     Test database with sample work data across a month
 * OUTPUT:    Validation of monthly report heatmap and achievements
 * BUSINESS:  Monthly reports provide long-term insights requiring comprehensive data
 * CHANGE:    Initial test for monthly report generation
 * RISK:      Low - Test validation with no production impact
 */
func TestSQLiteReportingService_GenerateMonthlyReport(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Insert test data
	insertTestData(t, db)

	// Initialize repositories
	sessionRepo := sqlite.NewSessionRepository(db)
	workBlockRepo := sqlite.NewWorkBlockRepository(db)
	activityRepo := sqlite.NewActivityRepository(db.DB())
	projectRepo := sqlite.NewProjectRepository(db)

	// Create reporting service
	reportingService := NewSQLiteReportingService(
		sessionRepo,
		workBlockRepo,
		activityRepo,
		projectRepo,
	)

	// Generate monthly report
	ctx := context.Background()
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	
	report, err := reportingService.GenerateMonthlyReport(ctx, "test-user", monthStart)

	// Validate report
	if err != nil {
		t.Fatalf("Failed to generate monthly report: %v", err)
	}

	if report == nil {
		t.Fatal("Monthly report is nil")
	}

	// Validate month metadata
	if report.Year != now.Year() {
		t.Errorf("Expected year %d, got %d", now.Year(), report.Year)
	}

	if report.MonthName == "" {
		t.Error("Month name should be set")
	}

	if report.TotalDays == 0 {
		t.Error("Total days should be greater than 0")
	}

	// Validate daily progress exists
	if len(report.DailyProgress) == 0 {
		t.Error("Daily progress should contain at least today's data")
	}

	// Validate monthly stats
	if report.MonthlyStats.WorkingDays < 0 {
		t.Error("Working days should not be negative")
	}

	t.Logf("Monthly report generated successfully for %s %d with %.1f total hours over %d days", 
		report.MonthName, report.Year, report.TotalWorkHours, report.DaysCompleted)
}

/**
 * CONTEXT:   Test work analytics engine integration with SQLite data
 * INPUT:     Test work blocks for deep work analysis
 * OUTPUT:    Validation of analytics engine functionality
 * BUSINESS:  Analytics engine provides advanced insights for productivity optimization
 * CHANGE:    Initial test for work analytics engine
 * RISK:      Low - Test validation with comprehensive analytics coverage
 */
func TestWorkAnalyticsEngine_Integration(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Insert test data
	insertTestData(t, db)

	// Initialize repositories
	workBlockRepo := sqlite.NewWorkBlockRepository(db)
	activityRepo := sqlite.NewActivityRepository(db.DB())
	projectRepo := sqlite.NewProjectRepository(db)

	// Create analytics engine
	analyticsEngine := NewWorkAnalyticsEngine(
		workBlockRepo,
		activityRepo,
		projectRepo,
	)

	// Get test work blocks
	ctx := context.Background()
	workBlocks, err := workBlockRepo.GetBySession(ctx, "test-session-1", 0)
	if err != nil {
		t.Fatalf("Failed to get work blocks: %v", err)
	}

	if len(workBlocks) == 0 {
		t.Fatal("No work blocks found for testing")
	}

	// Test deep work analysis
	deepWorkAnalysis := analyticsEngine.AnalyzeDeepWork(ctx, workBlocks)
	if deepWorkAnalysis == nil {
		t.Fatal("Deep work analysis is nil")
	}

	// Validate deep work metrics
	if deepWorkAnalysis.FocusScore < 0 || deepWorkAnalysis.FocusScore > 100 {
		t.Errorf("Focus score should be 0-100, got %.1f", deepWorkAnalysis.FocusScore)
	}

	if len(deepWorkAnalysis.FocusBlocks) != len(workBlocks) {
		t.Errorf("Expected %d focus blocks, got %d", len(workBlocks), len(deepWorkAnalysis.FocusBlocks))
	}

	// Test activity pattern analysis
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-1 * time.Nanosecond)

	activityAnalysis, err := analyticsEngine.AnalyzeActivityPatterns(ctx, "test-user", startOfDay, endOfDay)
	if err != nil {
		t.Fatalf("Failed to analyze activity patterns: %v", err)
	}

	if activityAnalysis == nil {
		t.Fatal("Activity analysis is nil")
	}

	// Validate activity analysis
	if activityAnalysis.TotalActivities < 0 {
		t.Error("Total activities should not be negative")
	}

	if len(activityAnalysis.HourlyDistribution) != 24 {
		t.Errorf("Expected 24 hourly entries, got %d", len(activityAnalysis.HourlyDistribution))
	}

	// Test project focus analysis
	projectAnalysis := analyticsEngine.AnalyzeProjectFocus(ctx, workBlocks)
	if projectAnalysis == nil {
		t.Fatal("Project focus analysis is nil")
	}

	// Validate project analysis
	if projectAnalysis.FocusEfficiency < 0 || projectAnalysis.FocusEfficiency > 100 {
		t.Errorf("Focus efficiency should be 0-100, got %.1f", projectAnalysis.FocusEfficiency)
	}

	t.Logf("Analytics engine integration successful - Focus Score: %.1f, Activity Count: %d, Project Efficiency: %.1f%%",
		deepWorkAnalysis.FocusScore, 
		activityAnalysis.TotalActivities,
		projectAnalysis.FocusEfficiency)
}

/**
 * CONTEXT:   Test error handling and edge cases in reporting system
 * INPUT:     Various error conditions and edge case scenarios
 * OUTPUT:    Validation of robust error handling and graceful degradation
 * BUSINESS:  Robust error handling ensures reliable reporting under all conditions
 * CHANGE:    Initial test for error handling and edge cases
 * RISK:      Low - Error handling test with comprehensive coverage
 */
func TestSQLiteReporting_ErrorHandling(t *testing.T) {
	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Initialize repositories
	sessionRepo := sqlite.NewSessionRepository(db)
	workBlockRepo := sqlite.NewWorkBlockRepository(db)
	activityRepo := sqlite.NewActivityRepository(db.DB())
	projectRepo := sqlite.NewProjectRepository(db)

	// Create reporting service
	reportingService := NewSQLiteReportingService(
		sessionRepo,
		workBlockRepo,
		activityRepo,
		projectRepo,
	)

	ctx := context.Background()

	// Test with non-existent user
	report, err := reportingService.GenerateDailyReport(ctx, "non-existent-user", time.Now())
	if err != nil {
		t.Errorf("Should handle non-existent user gracefully, got error: %v", err)
	}
	if report == nil {
		t.Error("Should return empty report for non-existent user")
	}
	if report.TotalWorkHours != 0 {
		t.Errorf("Expected 0 work hours for non-existent user, got %.1f", report.TotalWorkHours)
	}

	// Test with future date
	futureDate := time.Now().AddDate(1, 0, 0)
	report, err = reportingService.GenerateDailyReport(ctx, "test-user", futureDate)
	if err != nil {
		t.Errorf("Should handle future dates gracefully, got error: %v", err)
	}
	if report.TotalWorkHours != 0 {
		t.Errorf("Expected 0 work hours for future date, got %.1f", report.TotalWorkHours)
	}

	// Test with very old date
	oldDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	report, err = reportingService.GenerateDailyReport(ctx, "test-user", oldDate)
	if err != nil {
		t.Errorf("Should handle old dates gracefully, got error: %v", err)
	}

	t.Log("Error handling tests completed successfully")
}

/**
 * CONTEXT:   Performance test for SQLite reporting with larger datasets
 * INPUT:     Larger test dataset for performance validation
 * OUTPUT:    Performance metrics and validation of system scalability
 * BUSINESS:  Performance testing ensures system scales with real-world data volumes
 * CHANGE:    Initial performance test for reporting system
 * RISK:      Low - Performance test with controlled dataset
 */
func TestSQLiteReporting_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Insert larger test dataset
	insertLargeTestDataset(t, db)

	// Initialize repositories
	sessionRepo := sqlite.NewSessionRepository(db)
	workBlockRepo := sqlite.NewWorkBlockRepository(db)
	activityRepo := sqlite.NewActivityRepository(db.DB())
	projectRepo := sqlite.NewProjectRepository(db)

	// Create reporting service
	reportingService := NewSQLiteReportingService(
		sessionRepo,
		workBlockRepo,
		activityRepo,
		projectRepo,
	)

	ctx := context.Background()

	// Test daily report performance
	startTime := time.Now()
	report, err := reportingService.GenerateDailyReport(ctx, "test-user", time.Now())
	dailyDuration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Daily report failed: %v", err)
	}

	// Test weekly report performance
	startTime = time.Now()
	weekStart := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
	_, err = reportingService.GenerateWeeklyReport(ctx, "test-user", weekStart)
	weeklyDuration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Weekly report failed: %v", err)
	}

	// Performance assertions (should complete within reasonable time)
	if dailyDuration > 5*time.Second {
		t.Errorf("Daily report too slow: %v", dailyDuration)
	}

	if weeklyDuration > 10*time.Second {
		t.Errorf("Weekly report too slow: %v", weeklyDuration)
	}

	t.Logf("Performance test completed - Daily: %v, Weekly: %v, Work blocks: %d",
		dailyDuration, weeklyDuration, report.TotalWorkBlocks)
}

// Insert larger test dataset for performance testing
func insertLargeTestDataset(t *testing.T, db *sqlite.SQLiteDB) {
	// Create multiple sessions over the past week
	sessionRepo := sqlite.NewSessionRepository(db)
	projectRepo := sqlite.NewProjectRepository(db)
	workBlockRepo := sqlite.NewWorkBlockRepository(db)

	// Create test projects
	projects := []*sqlite.Project{
		{
			ID:             "perf-project-1",
			Name:           "Performance Project 1",
			Path:           "/perf/project1",
			NormalizedPath: "/perf/project1",
			ProjectType:    "go",
			LastActiveTime: time.Now(),
			TotalHours:     10.0,
			IsActive:       true,
			CreatedAt:      time.Now().AddDate(0, 0, -7),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             "perf-project-2",
			Name:           "Performance Project 2",
			Path:           "/perf/project2",
			NormalizedPath: "/perf/project2",
			ProjectType:    "javascript",
			LastActiveTime: time.Now(),
			TotalHours:     8.0,
			IsActive:       true,
			CreatedAt:      time.Now().AddDate(0, 0, -7),
			UpdatedAt:      time.Now(),
		},
	}

	for _, project := range projects {
		if err := projectRepo.Create(context.Background(), project); err != nil {
			t.Fatalf("Failed to create performance test project: %v", err)
		}
	}

	// Create sessions and work blocks for the past 7 days
	ctx := context.Background()
	for day := 0; day < 7; day++ {
		sessionDate := time.Now().AddDate(0, 0, -day)
		sessionID := fmt.Sprintf("perf-session-%d", day)

		session := &sqlite.Session{
			ID:                sessionID,
			UserID:           "test-user",
			StartTime:        sessionDate.Add(-4 * time.Hour),
			EndTime:          sessionDate.Add(1 * time.Hour),
			State:            "active",
			FirstActivityTime: sessionDate.Add(-3 * time.Hour),
			LastActivityTime:  sessionDate.Add(-1 * time.Hour),
			ActivityCount:    50,
			DurationHours:    5.0,
			CreatedAt:        sessionDate.Add(-4 * time.Hour),
			UpdatedAt:        sessionDate.Add(-1 * time.Hour),
		}

		if err := sessionRepo.Create(ctx, session); err != nil {
			t.Fatalf("Failed to create performance test session: %v", err)
		}

		// Create multiple work blocks per session
		for wb := 0; wb < 5; wb++ {
			workBlockID := fmt.Sprintf("perf-wb-%d-%d", day, wb)
			projectID := projects[wb%2].ID // Alternate between projects

			workBlock := &sqlite.WorkBlock{
				ID:               workBlockID,
				SessionID:        sessionID,
				ProjectID:        projectID,
				StartTime:        sessionDate.Add(time.Duration(-3+wb) * time.Hour),
				EndTime:          timePtr(sessionDate.Add(time.Duration(-3+wb+1) * time.Hour)),
				State:            "finished",
				LastActivityTime: sessionDate.Add(time.Duration(-3+wb+1) * time.Hour),
				ActivityCount:    int64(10 + wb*2),
				DurationSeconds:  3600, // 1 hour
				DurationHours:    1.0,
				CreatedAt:        sessionDate.Add(time.Duration(-3+wb) * time.Hour),
				UpdatedAt:        sessionDate.Add(time.Duration(-3+wb+1) * time.Hour),
			}

			if err := workBlockRepo.Create(ctx, workBlock); err != nil {
				t.Fatalf("Failed to create performance test work block: %v", err)
			}
		}
	}
}