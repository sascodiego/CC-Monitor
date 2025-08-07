/**
 * CONTEXT:   Domain entity representing project information with automatic detection logic
 * INPUT:     Project path, name, and metadata for work tracking organization
 * OUTPUT:    Project entity with validation, path normalization, and business rules
 * BUSINESS:  Projects automatically detected from working directory, track work organization
 * CHANGE:    Initial implementation following Clean Architecture and SOLID principles
 * RISK:      Low - Domain entity with path handling and validation logic
 */

package entities

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ProjectType represents different types of projects
type ProjectType string

const (
	ProjectTypeGeneral    ProjectType = "general"
	ProjectTypeGo         ProjectType = "go"
	ProjectTypeRust       ProjectType = "rust"
	ProjectTypePython     ProjectType = "python"
	ProjectTypeJavaScript ProjectType = "javascript"
	ProjectTypeTypeScript ProjectType = "typescript"
	ProjectTypeWeb        ProjectType = "web"
	ProjectTypeOther      ProjectType = "other"
)

/**
 * CONTEXT:   Core project entity for organizing and tracking work by project
 * INPUT:     Project metadata including path, name, type, and work statistics
 * OUTPUT:    Immutable project entity with path handling and business logic
 * BUSINESS:  Projects provide organizational structure for work blocks and sessions
 * CHANGE:    Initial domain entity with complete project management logic
 * RISK:      Low - Pure domain logic with path validation and no external dependencies
 */
type Project struct {
	id               string
	name             string
	path             string
	normalizedPath   string
	projectType      ProjectType
	description      string
	lastActiveTime   time.Time
	totalWorkBlocks  int64
	totalHours       float64
	isActive         bool
	createdAt        time.Time
	updatedAt        time.Time
}

// ProjectConfig holds configuration for creating new projects
type ProjectConfig struct {
	Name        string
	Path        string
	Description string
}

/**
 * CONTEXT:   Factory method for creating new project entities with path validation
 * INPUT:     ProjectConfig with name, path, and optional description
 * OUTPUT:    Valid Project entity or validation error
 * BUSINESS:  Projects created from working directory with automatic type detection
 * CHANGE:    Initial implementation with path normalization and validation
 * RISK:      Low - Validation prevents invalid project creation
 */
func NewProject(config ProjectConfig) (*Project, error) {
	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	if config.Path == "" {
		return nil, fmt.Errorf("project path cannot be empty")
	}

	// Normalize and validate path
	normalizedPath, err := normalizePath(config.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid project path: %w", err)
	}

	// Generate consistent project ID
	projectID := generateProjectIDFromPath(normalizedPath)

	// Detect project type
	projectType := detectProjectType(normalizedPath)

	now := time.Now()

	project := &Project{
		id:             projectID,
		name:           strings.TrimSpace(config.Name),
		path:           config.Path,
		normalizedPath: normalizedPath,
		projectType:    projectType,
		description:    strings.TrimSpace(config.Description),
		lastActiveTime: now,
		totalWorkBlocks: 0,
		totalHours:     0,
		isActive:       true,
		createdAt:      now,
		updatedAt:      now,
	}

	return project, nil
}

/**
 * CONTEXT:   Create project from current working directory automatically
 * INPUT:     Current working directory path
 * OUTPUT:    Project entity detected from directory structure
 * BUSINESS:  Automatic project detection from working directory for seamless UX
 * CHANGE:    Initial auto-detection implementation
 * RISK:      Low - Directory-based detection with fallback to directory name
 */
func NewProjectFromWorkingDir(workingDir string) (*Project, error) {
	if workingDir == "" {
		return nil, fmt.Errorf("working directory cannot be empty")
	}

	// Extract project name from directory
	projectName := filepath.Base(workingDir)
	if projectName == "" || projectName == "." || projectName == "/" {
		projectName = "unknown-project"
	}

	return NewProject(ProjectConfig{
		Name: projectName,
		Path: workingDir,
		Description: fmt.Sprintf("Auto-detected project from %s", workingDir),
	})
}

// Getter methods (immutable access)
func (p *Project) ID() string               { return p.id }
func (p *Project) Name() string             { return p.name }
func (p *Project) Path() string             { return p.path }
func (p *Project) NormalizedPath() string   { return p.normalizedPath }
func (p *Project) ProjectType() ProjectType { return p.projectType }
func (p *Project) Description() string      { return p.description }
func (p *Project) LastActiveTime() time.Time { return p.lastActiveTime }
func (p *Project) TotalWorkBlocks() int64   { return p.totalWorkBlocks }
func (p *Project) TotalHours() float64      { return p.totalHours }
func (p *Project) IsActive() bool           { return p.isActive }
func (p *Project) CreatedAt() time.Time     { return p.createdAt }
func (p *Project) UpdatedAt() time.Time     { return p.updatedAt }

/**
 * CONTEXT:   Update project statistics when work blocks are added
 * INPUT:     Work block hours to add to project totals
 * OUTPUT:    Error if update invalid, nil on success
 * BUSINESS:  Projects track total work time and block count for analytics
 * CHANGE:    Initial statistics update implementation
 * RISK:      Low - Simple numerical updates with validation
 */
func (p *Project) AddWorkBlock(hours float64, activityTime time.Time) error {
	if hours < 0 {
		return fmt.Errorf("hours cannot be negative: %f", hours)
	}

	p.totalWorkBlocks++
	p.totalHours += hours
	p.lastActiveTime = activityTime
	p.isActive = true
	p.updatedAt = time.Now()

	return nil
}

/**
 * CONTEXT:   Record activity on project to update last active time
 * INPUT:     Activity timestamp for project activity tracking
 * OUTPUT:    Error if timestamp invalid, nil on success
 * BUSINESS:  Track project activity for lifecycle and analytics
 * CHANGE:    Added method for activity recording
 * RISK:      Low - Simple timestamp update
 */
func (p *Project) RecordActivity(activityTime time.Time) error {
	if activityTime.IsZero() {
		return fmt.Errorf("activity time cannot be zero")
	}

	p.lastActiveTime = activityTime
	p.isActive = true
	p.updatedAt = time.Now()

	return nil
}

/**
 * CONTEXT:   Mark project as inactive when no recent work detected
 * INPUT:     Timestamp when project became inactive
 * OUTPUT:    Error if state change invalid, nil on success
 * BUSINESS:  Projects become inactive to optimize queries and reporting
 * CHANGE:    Initial project lifecycle management
 * RISK:      Low - State change with timestamp validation
 */
func (p *Project) MarkInactive(inactiveTime time.Time) error {
	if inactiveTime.Before(p.lastActiveTime) {
		return fmt.Errorf("inactive time %v cannot be before last active time %v",
			inactiveTime, p.lastActiveTime)
	}

	p.isActive = false
	p.updatedAt = time.Now()

	return nil
}

/**
 * CONTEXT:   Calculate project activity metrics and patterns
 * INPUT:     Optional time range for calculation scope
 * OUTPUT:    Activity metrics including work density and patterns
 * BUSINESS:  Provide insights into project work patterns and productivity
 * CHANGE:    Initial metrics calculation implementation
 * RISK:      Low - Read-only calculation based on stored data
 */
func (p *Project) AverageHoursPerBlock() float64 {
	if p.totalWorkBlocks == 0 {
		return 0
	}
	return p.totalHours / float64(p.totalWorkBlocks)
}

func (p *Project) ProjectAge() time.Duration {
	return time.Since(p.createdAt)
}

func (p *Project) InactiveDuration() time.Duration {
	if p.isActive {
		return 0
	}
	return time.Since(p.lastActiveTime)
}

/**
 * CONTEXT:   Check if two projects represent the same entity
 * INPUT:     Another project entity for comparison
 * OUTPUT:    Boolean indicating if projects are equivalent
 * BUSINESS:  Project equality based on normalized path for deduplication
 * CHANGE:    Initial project equality implementation
 * RISK:      Low - Comparison logic with no side effects
 */
func (p *Project) IsSameProject(other *Project) bool {
	if other == nil {
		return false
	}
	return p.normalizedPath == other.normalizedPath
}

/**
 * CONTEXT:   Export project data for serialization or reporting
 * INPUT:     No parameters, uses internal project state
 * OUTPUT:    ProjectData struct suitable for JSON serialization
 * BUSINESS:  Provide read-only view of project data for external use
 * CHANGE:    Initial data export implementation
 * RISK:      Low - Read-only operation with no state changes
 */
type ProjectData struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Path             string      `json:"path"`
	NormalizedPath   string      `json:"normalized_path"`
	ProjectType      ProjectType `json:"project_type"`
	Description      string      `json:"description"`
	LastActiveTime   time.Time   `json:"last_active_time"`
	TotalWorkBlocks  int64       `json:"total_work_blocks"`
	TotalHours       float64     `json:"total_hours"`
	AverageHoursPerBlock float64 `json:"average_hours_per_block"`
	IsActive         bool        `json:"is_active"`
	ProjectAge       float64     `json:"project_age_hours"`
	InactiveDuration float64     `json:"inactive_duration_hours"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

func (p *Project) ToData() ProjectData {
	return ProjectData{
		ID:                   p.id,
		Name:                 p.name,
		Path:                 p.path,
		NormalizedPath:       p.normalizedPath,
		ProjectType:          p.projectType,
		Description:          p.description,
		LastActiveTime:       p.lastActiveTime,
		TotalWorkBlocks:      p.totalWorkBlocks,
		TotalHours:           p.totalHours,
		AverageHoursPerBlock: p.AverageHoursPerBlock(),
		IsActive:             p.isActive,
		ProjectAge:           p.ProjectAge().Hours(),
		InactiveDuration:     p.InactiveDuration().Hours(),
		CreatedAt:            p.createdAt,
		UpdatedAt:            p.updatedAt,
	}
}

/**
 * CONTEXT:   Validate project internal state consistency
 * INPUT:     No parameters, validates internal state
 * OUTPUT:    Error if state is inconsistent, nil if valid
 * BUSINESS:  Ensure project data integrity and business rule compliance
 * CHANGE:    Initial state validation implementation
 * RISK:      Low - Validation only, no state changes
 */
func (p *Project) Validate() error {
	// Check required fields
	if p.id == "" {
		return fmt.Errorf("project ID cannot be empty")
	}
	if p.name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if p.path == "" {
		return fmt.Errorf("project path cannot be empty")
	}
	if p.normalizedPath == "" {
		return fmt.Errorf("normalized path cannot be empty")
	}

	// Check numerical constraints
	if p.totalWorkBlocks < 0 {
		return fmt.Errorf("total work blocks cannot be negative")
	}
	if p.totalHours < 0 {
		return fmt.Errorf("total hours cannot be negative")
	}

	// Check time relationships
	if p.lastActiveTime.Before(p.createdAt) {
		return fmt.Errorf("last active time cannot be before created time")
	}

	return nil
}

/**
 * CONTEXT:   Utility functions for path handling and project type detection
 * INPUT:     File system paths and project structure information
 * OUTPUT:    Normalized paths and detected project types
 * BUSINESS:  Consistent path handling and automatic project classification
 * CHANGE:    Initial utility functions for project management
 * RISK:      Low - File path manipulation with error handling
 */

func normalizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Clean and resolve the path
	cleanPath := filepath.Clean(path)
	
	// Convert to absolute path if possible
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		// If absolute path fails, use clean path
		return cleanPath, nil
	}

	return absPath, nil
}

func detectProjectType(projectPath string) ProjectType {
	// Check for common project files and directories
	pathLower := strings.ToLower(projectPath)
	
	// Go projects
	if strings.Contains(pathLower, "go.mod") || strings.Contains(pathLower, "go.sum") {
		return ProjectTypeGo
	}
	
	// Rust projects
	if strings.Contains(pathLower, "cargo.toml") || strings.Contains(pathLower, "cargo.lock") {
		return ProjectTypeRust
	}
	
	// Python projects
	if strings.Contains(pathLower, "requirements.txt") || strings.Contains(pathLower, "setup.py") ||
		strings.Contains(pathLower, "pyproject.toml") {
		return ProjectTypePython
	}
	
	// JavaScript/TypeScript projects
	if strings.Contains(pathLower, "package.json") || strings.Contains(pathLower, "node_modules") {
		if strings.Contains(pathLower, "tsconfig.json") {
			return ProjectTypeTypeScript
		}
		return ProjectTypeJavaScript
	}
	
	// Web projects
	if strings.Contains(pathLower, "index.html") || strings.Contains(pathLower, "webpack") {
		return ProjectTypeWeb
	}
	
	return ProjectTypeGeneral
}

func generateProjectIDFromPath(normalizedPath string) string {
	// Generate a consistent ID based on path
	hash := fmt.Sprintf("%x", []byte(normalizedPath))
	if len(hash) > 12 {
		hash = hash[:12]
	}
	return fmt.Sprintf("proj_%s", hash)
}