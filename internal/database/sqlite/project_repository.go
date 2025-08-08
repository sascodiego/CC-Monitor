/**
 * CONTEXT:   Project repository for SQLite database operations
 * INPUT:     Project CRUD operations with automatic project detection
 * OUTPUT:    Database persistence for project management
 * BUSINESS:  Projects organize work blocks and activities by working directory
 * CHANGE:    CHECKPOINT 8 - Basic project repository for production deployment
 * RISK:      Low - Simple repository implementation for deployment verification
 */

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ProjectRepository handles database operations for projects
type ProjectRepository struct {
	db *sql.DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Project type is already defined in migration.go

/**
 * CONTEXT:   Create new project in database
 * INPUT:     Project entity with name and path
 * OUTPUT:    Persisted project with unique constraints validated
 * BUSINESS:  Projects are identified by unique paths
 * CHANGE:    CHECKPOINT 8 - Basic project creation
 * RISK:      Low - Standard database insertion with unique constraints
 */
func (pr *ProjectRepository) Create(ctx context.Context, project *Project) error {
	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}

	query := `
		INSERT INTO projects (id, name, path, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := pr.db.ExecContext(ctx, query,
		project.ID, project.Name, project.Path, project.Description,
		project.CreatedAt, project.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

/**
 * CONTEXT:   Get project by ID
 * INPUT:     Project ID for lookup
 * OUTPUT:    Project entity or error if not found
 * BUSINESS:  Project lookup for work block associations
 * CHANGE:    CHECKPOINT 8 - Basic project retrieval
 * RISK:      Low - Standard database query
 */
func (pr *ProjectRepository) GetByID(ctx context.Context, id string) (*Project, error) {
	if id == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	query := `
		SELECT id, name, path, description, created_at, updated_at
		FROM projects
		WHERE id = ?
	`

	var project Project
	err := pr.db.QueryRowContext(ctx, query, id).Scan(
		&project.ID, &project.Name, &project.Path, &project.Description,
		&project.CreatedAt, &project.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

/**
 * CONTEXT:   Get project by path with normalized path matching
 * INPUT:     Project path for lookup
 * OUTPUT:    Project entity or nil if not found
 * BUSINESS:  Project identification by working directory path
 * CHANGE:    CHECKPOINT 8 - Path-based project lookup
 * RISK:      Low - Query with path normalization
 */
func (pr *ProjectRepository) GetByPath(ctx context.Context, path string) (*Project, error) {
	if path == "" {
		return nil, fmt.Errorf("project path cannot be empty")
	}

	// Normalize path for consistent matching
	normalizedPath := normalizePath(path)

	query := `
		SELECT id, name, path, description, created_at, updated_at
		FROM projects
		WHERE path = ?
	`

	var project Project
	err := pr.db.QueryRowContext(ctx, query, normalizedPath).Scan(
		&project.ID, &project.Name, &project.Path, &project.Description,
		&project.CreatedAt, &project.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Project not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project by path: %w", err)
	}

	return &project, nil
}

/**
 * CONTEXT:   Get or create project by path with automatic detection
 * INPUT:     Project path for identification or creation
 * OUTPUT:    Existing or newly created project entity
 * BUSINESS:  Automatic project creation ensures work blocks have valid projects
 * CHANGE:    CHECKPOINT 8 - Auto-creation for seamless project management
 * RISK:      Low - Upsert operation with path normalization
 */
func (pr *ProjectRepository) GetOrCreate(ctx context.Context, projectPath string) (*Project, error) {
	if projectPath == "" {
		return nil, fmt.Errorf("project path cannot be empty")
	}

	// First, try to get existing project
	existingProject, err := pr.GetByPath(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing project: %w", err)
	}

	if existingProject != nil {
		return existingProject, nil
	}

	// Create new project
	normalizedPath := normalizePath(projectPath)
	projectName := generateProjectName(normalizedPath)
	projectID := generateProjectID(projectName, normalizedPath)

	newProject := &Project{
		ID:          projectID,
		Name:        projectName,
		Path:        normalizedPath,
		Description: fmt.Sprintf("Auto-detected project at %s", normalizedPath),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = pr.Create(ctx, newProject)
	if err != nil {
		return nil, fmt.Errorf("failed to create new project: %w", err)
	}

	return newProject, nil
}

/**
 * CONTEXT:   Get all projects for system monitoring
 * INPUT:     Context for database operations
 * OUTPUT:    All projects in the system
 * BUSINESS:  Project listing for reporting and administration
 * CHANGE:    CHECKPOINT 8 - All projects retrieval
 * RISK:      Low - System-wide query for monitoring
 */
func (pr *ProjectRepository) GetAll(ctx context.Context) ([]*Project, error) {
	query := `
		SELECT id, name, path, description, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC
	`

	rows, err := pr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var project Project
		err := rows.Scan(
			&project.ID, &project.Name, &project.Path, &project.Description,
			&project.CreatedAt, &project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, &project)
	}

	return projects, nil
}

/**
 * CONTEXT:   Update project in database
 * INPUT:     Updated project entity
 * OUTPUT:    Persisted changes to project
 * BUSINESS:  Project updates for name and description changes
 * CHANGE:    CHECKPOINT 8 - Project update operations
 * RISK:      Low - Standard update operation
 */
func (pr *ProjectRepository) Update(ctx context.Context, project *Project) error {
	if project == nil {
		return fmt.Errorf("project cannot be nil")
	}

	query := `
		UPDATE projects 
		SET name = ?, path = ?, description = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := pr.db.ExecContext(ctx, query,
		project.Name, project.Path, project.Description, project.UpdatedAt,
		project.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project %s not found", project.ID)
	}

	return nil
}

/**
 * CONTEXT:   Delete project from database
 * INPUT:     Project ID for deletion
 * OUTPUT:    Removed project with foreign key cascade
 * BUSINESS:  Project deletion removes associated work blocks
 * CHANGE:    CHECKPOINT 8 - Project deletion with cascade
 * RISK:      Medium - Deletion operation affecting associated records
 */
func (pr *ProjectRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("project ID cannot be empty")
	}

	query := `DELETE FROM projects WHERE id = ?`

	result, err := pr.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project %s not found", id)
	}

	return nil
}

// Helper functions for project path and name normalization

/**
 * CONTEXT:   Normalize project path for consistent database storage
 * INPUT:     Raw project path from working directory
 * OUTPUT:    Normalized path with consistent format
 * BUSINESS:  Path normalization ensures consistent project identification
 * CHANGE:    CHECKPOINT 8 - Path normalization utility
 * RISK:      Low - String processing utility
 */
func normalizePath(path string) string {
	// Clean the path to resolve . and .. elements
	cleaned := filepath.Clean(path)
	
	// Convert to absolute path format (forward slashes)
	normalized := filepath.ToSlash(cleaned)
	
	return normalized
}

/**
 * CONTEXT:   Generate project name from path
 * INPUT:     Normalized project path
 * OUTPUT:    Human-readable project name
 * BUSINESS:  Project names derived from directory structure
 * CHANGE:    CHECKPOINT 8 - Project name generation
 * RISK:      Low - Name generation utility
 */
func generateProjectName(path string) string {
	// Get the base directory name
	baseName := filepath.Base(path)
	
	// Handle special cases
	if baseName == "." || baseName == "/" || baseName == "\\" {
		return "Root Project"
	}
	
	// Clean up the name
	name := strings.ReplaceAll(baseName, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.Title(strings.ToLower(name))
	
	return name
}

// generateProjectID function is already defined in migration.go