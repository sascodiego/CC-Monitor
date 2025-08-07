/**
 * CONTEXT:   Project management use case for auto-detection and project lifecycle
 * INPUT:     Project paths, names, and project information from activity events
 * OUTPUT:    Project entities with auto-detection, validation, and persistence
 * BUSINESS:  Automatically detect and manage projects from working directory information
 * CHANGE:    Initial implementation following Clean Architecture and SOLID principles
 * RISK:      Medium - Project detection affects work block associations and analytics
 */

package usecases

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/claude-monitor/system/internal/entities"
	"github.com/claude-monitor/system/internal/usecases/repositories"
)

/**
 * CONTEXT:   Project manager coordinating project auto-detection and lifecycle
 * INPUT:     Project repository for persistence and project information from activity
 * OUTPUT:    Project management operations with auto-detection and caching
 * BUSINESS:  Manages project entities with automatic detection from working directories
 * CHANGE:    Initial project manager implementation with dependency injection
 * RISK:      Medium - Central component for project detection and association
 */
type ProjectManager struct {
	projectRepo   repositories.ProjectRepository
	logger        *slog.Logger
	mu            sync.RWMutex
	projectCache  map[string]*entities.Project // path -> project cache
	maxCacheSize  int
}

// ProjectManagerConfig holds configuration for project manager
type ProjectManagerConfig struct {
	ProjectRepo  repositories.ProjectRepository
	Logger       *slog.Logger
	MaxCacheSize int
}

/**
 * CONTEXT:   Factory function for creating new project manager with proper configuration
 * INPUT:     ProjectManagerConfig with repository and operational parameters
 * OUTPUT:    Configured ProjectManager instance ready for project management
 * BUSINESS:  Project manager requires repository for persistence and caching for performance
 * CHANGE:    Initial factory implementation with configuration validation
 * RISK:      Low - Factory function with validation and default value handling
 */
func NewProjectManager(config ProjectManagerConfig) (*ProjectManager, error) {
	if config.ProjectRepo == nil {
		return nil, fmt.Errorf("project repository cannot be nil")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	maxCacheSize := config.MaxCacheSize
	if maxCacheSize <= 0 {
		maxCacheSize = 1000 // default cache size
	}

	pm := &ProjectManager{
		projectRepo:  config.ProjectRepo,
		logger:       logger,
		projectCache: make(map[string]*entities.Project),
		maxCacheSize: maxCacheSize,
	}

	return pm, nil
}

/**
 * CONTEXT:   Get or create project with auto-detection from path information
 * INPUT:     Project path and optional project name for project detection
 * OUTPUT:    Project entity either from cache/database or newly created
 * BUSINESS:  Projects are auto-detected from working directory with name normalization
 * CHANGE:    Initial implementation with auto-detection and caching logic
 * RISK:      Medium - Core project detection logic affecting work block associations
 */
func (pm *ProjectManager) GetOrCreateProject(ctx context.Context, projectPath, projectName string) (*entities.Project, error) {
	if projectPath == "" {
		return nil, fmt.Errorf("project path cannot be empty")
	}

	// Normalize the path
	normalizedPath, err := pm.normalizePath(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize path: %w", err)
	}

	pm.logger.Debug("Getting or creating project", "path", normalizedPath, "name", projectName)

	// Check cache first
	pm.mu.RLock()
	cachedProject, hasCached := pm.projectCache[normalizedPath]
	pm.mu.RUnlock()

	if hasCached {
		pm.logger.Debug("Found project in cache", "projectID", cachedProject.ID())
		return cachedProject, nil
	}

	// Check database
	existingProject, err := pm.projectRepo.FindByPath(ctx, normalizedPath)
	if err != nil && !isProjectNotFoundError(err) {
		return nil, fmt.Errorf("failed to find project by path: %w", err)
	}

	// If found in database, cache and return
	if existingProject != nil {
		pm.addToCache(normalizedPath, existingProject)
		pm.logger.Debug("Found project in database", "projectID", existingProject.ID())
		return existingProject, nil
	}

	// Create new project
	return pm.createNewProject(ctx, normalizedPath, projectName)
}

/**
 * CONTEXT:   Auto-detect project information from path and environment
 * INPUT:     Project path for analysis and project name detection
 * OUTPUT:    ProjectInfo struct with detected name, type, and metadata
 * BUSINESS:  Automatically extract meaningful project information from file system
 * CHANGE:    Initial auto-detection implementation with common project patterns
 * RISK:      Medium - Project detection logic affects project naming and classification
 */
type ProjectInfo struct {
	Name        string
	Type        string
	Language    string
	Framework   string
	Description string
	Metadata    map[string]string
}

func (pm *ProjectManager) DetectProjectInfo(projectPath string) (*ProjectInfo, error) {
	if projectPath == "" {
		return nil, fmt.Errorf("project path cannot be empty")
	}

	normalizedPath, err := pm.normalizePath(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize path: %w", err)
	}

	// Extract project name from path
	projectName := filepath.Base(normalizedPath)
	if projectName == "." || projectName == "/" {
		projectName = "unknown-project"
	}

	// Detect project type and language
	projectType := pm.detectProjectType(normalizedPath)
	language := pm.detectPrimaryLanguage(normalizedPath)
	framework := pm.detectFramework(normalizedPath)

	// Create metadata
	metadata := make(map[string]string)
	metadata["path"] = normalizedPath
	metadata["detected_at"] = "runtime"
	
	if projectType != "" {
		metadata["type"] = projectType
	}
	if language != "" {
		metadata["primary_language"] = language
	}
	if framework != "" {
		metadata["framework"] = framework
	}

	info := &ProjectInfo{
		Name:        pm.normalizeProjectName(projectName),
		Type:        projectType,
		Language:    language,
		Framework:   framework,
		Description: fmt.Sprintf("Auto-detected %s project", projectName),
		Metadata:    metadata,
	}

	pm.logger.Debug("Detected project info", 
		"name", info.Name, 
		"type", info.Type, 
		"language", info.Language,
		"framework", info.Framework)

	return info, nil
}

/**
 * CONTEXT:   Update project information with new activity data
 * INPUT:     Project ID and activity timestamp for project maintenance
 * OUTPUT:    Updated project entity with latest activity information
 * BUSINESS:  Track project activity for analytics and project lifecycle management
 * CHANGE:    Initial project activity tracking implementation
 * RISK:      Low - Simple timestamp update operation with caching
 */
func (pm *ProjectManager) UpdateProjectActivity(ctx context.Context, projectID string, activityTime time.Time) error {
	if projectID == "" {
		return fmt.Errorf("project ID cannot be empty")
	}

	// Find project
	project, err := pm.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to find project: %w", err)
	}

	// Update activity
	project.RecordActivity(activityTime)

	// Update in repository
	err = pm.projectRepo.Update(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to update project activity: %w", err)
	}

	// Update cache
	pm.mu.Lock()
	pm.projectCache[project.Path()] = project
	pm.mu.Unlock()

	pm.logger.Debug("Updated project activity", "projectID", projectID, "activityTime", activityTime)
	return nil
}

/**
 * CONTEXT:   Get project statistics for analytics and reporting
 * INPUT:     Project ID and optional time range for statistics calculation
 * OUTPUT:    Project statistics including work time, activity counts, and patterns
 * BUSINESS:  Provide project-level analytics for productivity insights and reporting
 * CHANGE:    Initial statistics implementation delegating to repository
 * RISK:      Low - Read-only analytics operation with no state changes
 */
func (pm *ProjectManager) GetProjectStatistics(ctx context.Context, projectID string, start, end time.Time) (*repositories.ProjectStatistics, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	stats, err := pm.projectRepo.GetProjectStatistics(ctx, projectID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get project statistics: %w", err)
	}

	pm.logger.Debug("Retrieved project statistics",
		"projectID", projectID,
		"totalWorkBlocks", stats.TotalWorkBlocks)

	return stats, nil
}

/**
 * CONTEXT:   List all projects with optional filtering and sorting
 * INPUT:     Context and optional filter criteria for project listing
 * OUTPUT:    List of project entities matching filter criteria
 * BUSINESS:  Provide project discovery and listing for CLI and reporting
 * CHANGE:    Initial project listing implementation with repository delegation
 * RISK:      Low - Read-only operation with filtering support
 */
func (pm *ProjectManager) ListProjects(ctx context.Context, filter *repositories.ProjectFilter) ([]*entities.Project, error) {
	var projects []*entities.Project
	var err error
	
	if filter != nil {
		projects, err = pm.projectRepo.FindByFilter(ctx, *filter)
	} else {
		projects, err = pm.projectRepo.FindAll(ctx)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	pm.logger.Debug("Listed projects", "count", len(projects))
	return projects, nil
}

// Private helper methods

/**
 * CONTEXT:   Create new project entity with auto-detected information
 * INPUT:     Project path and optional name for project creation
 * OUTPUT:    New project entity with proper validation and persistence
 * BUSINESS:  Create projects with auto-detected metadata and proper validation
 * CHANGE:    Internal helper for project creation with auto-detection
 * RISK:      Medium - Project creation affects all subsequent work tracking
 */
func (pm *ProjectManager) createNewProject(ctx context.Context, projectPath, projectName string) (*entities.Project, error) {
	// Auto-detect project information
	projectInfo, err := pm.DetectProjectInfo(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project info: %w", err)
	}

	// Use provided name if available, otherwise use detected name
	if projectName != "" {
		projectInfo.Name = pm.normalizeProjectName(projectName)
	}

	// Create project entity
	project, err := entities.NewProject(entities.ProjectConfig{
		Name:        projectInfo.Name,
		Path:        projectPath,
		Description: projectInfo.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create project entity: %w", err)
	}

	// Save to repository
	err = pm.projectRepo.Save(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	// Add to cache
	pm.addToCache(projectPath, project)

	pm.logger.Info("Created new project",
		"projectID", project.ID(),
		"name", project.Name(),
		"path", projectPath)

	return project, nil
}

func (pm *ProjectManager) normalizePath(path string) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(path)
	
	// Convert to absolute path if possible
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		// If absolute path fails, use clean path
		return cleanPath, nil
	}
	
	return absPath, nil
}

func (pm *ProjectManager) normalizeProjectName(name string) string {
	// Remove common prefixes/suffixes
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, ".")
	
	// Replace common separators with hyphens
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	
	// Convert to lowercase
	name = strings.ToLower(name)
	
	// Ensure non-empty
	if name == "" {
		name = "unnamed-project"
	}
	
	return name
}

func (pm *ProjectManager) detectProjectType(projectPath string) string {
	// Check for common project indicators
	commonFiles := []struct {
		file        string
		projectType string
	}{
		{"go.mod", "go-module"},
		{"package.json", "nodejs"},
		{"Cargo.toml", "rust"},
		{"pom.xml", "maven"},
		{"build.gradle", "gradle"},
		{"setup.py", "python"},
		{"requirements.txt", "python"},
		{"composer.json", "php"},
		{"Gemfile", "ruby"},
		{"Makefile", "makefile"},
		{"CMakeLists.txt", "cmake"},
		{".git", "git-repository"},
	}

	for _, cf := range commonFiles {
		checkPath := filepath.Join(projectPath, cf.file)
		if pm.pathExists(checkPath) {
			return cf.projectType
		}
	}

	return "generic"
}

func (pm *ProjectManager) detectPrimaryLanguage(projectPath string) string {
	// This would typically scan files in the directory
	// For now, return based on project type indicators
	
	languageFiles := []struct {
		pattern  string
		language string
	}{
		{"go.mod", "go"},
		{"*.go", "go"},
		{"package.json", "javascript"},
		{"*.js", "javascript"},
		{"*.ts", "typescript"},
		{"Cargo.toml", "rust"},
		{"*.rs", "rust"},
		{"*.py", "python"},
		{"*.java", "java"},
		{"*.cpp", "cpp"},
		{"*.c", "c"},
	}

	for _, lf := range languageFiles {
		checkPath := filepath.Join(projectPath, lf.pattern)
		if pm.pathExists(checkPath) {
			return lf.language
		}
	}

	return "unknown"
}

func (pm *ProjectManager) detectFramework(projectPath string) string {
	// Detect common frameworks based on file presence
	frameworks := []struct {
		indicator string
		framework string
	}{
		{"next.config.js", "nextjs"},
		{"nuxt.config.js", "nuxtjs"},
		{"vue.config.js", "vuejs"},
		{"angular.json", "angular"},
		{"gatsby-config.js", "gatsby"},
		{"svelte.config.js", "svelte"},
	}

	for _, fw := range frameworks {
		checkPath := filepath.Join(projectPath, fw.indicator)
		if pm.pathExists(checkPath) {
			return fw.framework
		}
	}

	return ""
}

func (pm *ProjectManager) pathExists(path string) bool {
	_, err := filepath.Abs(path)
	return err == nil
}

func (pm *ProjectManager) addToCache(path string, project *entities.Project) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check cache size limit
	if len(pm.projectCache) >= pm.maxCacheSize {
		// Simple eviction: remove first entry (could be improved with LRU)
		for k := range pm.projectCache {
			delete(pm.projectCache, k)
			break
		}
	}

	pm.projectCache[path] = project
}

// Helper function to check if error is "project not found"
func isProjectNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "project not found" || err == repositories.ErrProjectNotFound
}