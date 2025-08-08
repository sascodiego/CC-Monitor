/**
 * CONTEXT:   SQLite schema for Claude Monitor with proper relational design
 * INPUT:     Database initialization and migration requirements
 * OUTPUT:    Production-ready SQLite schema with indexes and constraints
 * BUSINESS:  Sessions track 5-hour windows, work blocks track active periods, activities track events
 * CHANGE:    Initial SQLite schema replacing gob-based persistence
 * RISK:      Low - Standard relational design with foreign key constraints
 */

-- Enable foreign key constraints (required for SQLite)
PRAGMA foreign_keys = ON;

-- Users table (normalized user information)
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Projects table (normalized project information) 
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table (5-hour Claude sessions)
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL, -- Always start_time + 5 hours (no IsActive flag needed)
    state TEXT NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'expired', 'finished')),
    first_activity_time DATETIME NOT NULL,
    last_activity_time DATETIME NOT NULL,
    activity_count INTEGER NOT NULL DEFAULT 1,
    duration_hours REAL NOT NULL DEFAULT 5.0, -- Always 5.0 for sessions
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key constraints
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Business rule constraints  
    CHECK (end_time > start_time),
    CHECK (last_activity_time >= first_activity_time),
    CHECK (first_activity_time >= start_time),
    CHECK (last_activity_time <= end_time),
    CHECK (activity_count >= 1),
    CHECK (duration_hours = 5.0) -- Enforce 5-hour session rule
);

-- Work blocks table (active work periods within sessions)
CREATE TABLE IF NOT EXISTS work_blocks (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME, -- NULL if still active
    state TEXT NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'idle', 'processing', 'finished')),
    last_activity_time DATETIME NOT NULL,
    activity_count INTEGER NOT NULL DEFAULT 1,
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    duration_hours REAL NOT NULL DEFAULT 0.0,
    -- Claude processing tracking
    claude_processing_seconds INTEGER NOT NULL DEFAULT 0,
    claude_processing_hours REAL NOT NULL DEFAULT 0.0,
    estimated_end_time DATETIME,
    last_claude_activity DATETIME,
    active_prompt_id TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key constraints
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    
    -- Business rule constraints
    CHECK (last_activity_time >= start_time),
    CHECK (end_time IS NULL OR end_time >= start_time),
    CHECK (end_time IS NULL OR last_activity_time <= end_time),
    CHECK (activity_count >= 1),
    CHECK (duration_seconds >= 0),
    CHECK (duration_hours >= 0.0),
    CHECK (claude_processing_seconds >= 0),
    CHECK (claude_processing_hours >= 0.0),
    CHECK ((state = 'finished' AND end_time IS NOT NULL) OR (state != 'finished'))
);

-- Activity events table (individual activity records)
CREATE TABLE IF NOT EXISTS activity_events (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    session_id TEXT,
    work_block_id TEXT,
    project_id TEXT,
    activity_type TEXT NOT NULL DEFAULT 'other' CHECK (activity_type IN ('command', 'file_edit', 'file_read', 'navigation', 'search', 'generation', 'other')),
    activity_source TEXT NOT NULL DEFAULT 'hook' CHECK (activity_source IN ('hook', 'cli', 'daemon', 'manual')),
    timestamp DATETIME NOT NULL,
    command TEXT,
    description TEXT,
    metadata TEXT, -- JSON string for flexible metadata storage
    -- Claude processing context
    claude_activity_type TEXT CHECK (claude_activity_type IN ('user_action', 'claude_start', 'claude_end', 'claude_progress')),
    prompt_id TEXT,
    estimated_processing_time INTEGER, -- milliseconds
    actual_processing_time INTEGER,    -- milliseconds
    tokens_count INTEGER,
    prompt_length INTEGER,
    complexity_hint TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key constraints
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE SET NULL,
    FOREIGN KEY (work_block_id) REFERENCES work_blocks(id) ON DELETE SET NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL,
    
    -- Business rule constraints
    CHECK (timestamp <= created_at),
    CHECK (estimated_processing_time IS NULL OR estimated_processing_time >= 0),
    CHECK (actual_processing_time IS NULL OR actual_processing_time >= 0),
    CHECK (tokens_count IS NULL OR tokens_count >= 0),
    CHECK (prompt_length IS NULL OR prompt_length >= 0)
);

-- Indexes for query performance (critical for reporting)

-- Session indexes
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions(start_time);
CREATE INDEX IF NOT EXISTS idx_sessions_end_time ON sessions(end_time); 
CREATE INDEX IF NOT EXISTS idx_sessions_state ON sessions(state);
CREATE INDEX IF NOT EXISTS idx_sessions_time_range ON sessions(start_time, end_time);

-- Work block indexes
CREATE INDEX IF NOT EXISTS idx_work_blocks_session_id ON work_blocks(session_id);
CREATE INDEX IF NOT EXISTS idx_work_blocks_project_id ON work_blocks(project_id);
CREATE INDEX IF NOT EXISTS idx_work_blocks_start_time ON work_blocks(start_time);
CREATE INDEX IF NOT EXISTS idx_work_blocks_state ON work_blocks(state);
CREATE INDEX IF NOT EXISTS idx_work_blocks_time_range ON work_blocks(start_time, end_time);

-- Activity event indexes
CREATE INDEX IF NOT EXISTS idx_activity_events_user_id ON activity_events(user_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_session_id ON activity_events(session_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_work_block_id ON activity_events(work_block_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_project_id ON activity_events(project_id);
CREATE INDEX IF NOT EXISTS idx_activity_events_timestamp ON activity_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_activity_events_activity_type ON activity_events(activity_type);
CREATE INDEX IF NOT EXISTS idx_activity_events_claude_activity ON activity_events(claude_activity_type);

-- Project indexes
CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
CREATE INDEX IF NOT EXISTS idx_projects_path ON projects(path);

-- User indexes
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Composite indexes for common reporting queries

-- Daily/weekly/monthly reporting by user and time range
CREATE INDEX IF NOT EXISTS idx_sessions_user_time ON sessions(user_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_work_blocks_user_time ON work_blocks(session_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_activity_events_user_time ON activity_events(user_id, timestamp);

-- Project analytics
CREATE INDEX IF NOT EXISTS idx_work_blocks_project_time ON work_blocks(project_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_activity_events_project_time ON activity_events(project_id, timestamp);

-- Claude processing analytics
CREATE INDEX IF NOT EXISTS idx_activity_events_claude_processing ON activity_events(claude_activity_type, timestamp) WHERE claude_activity_type IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_work_blocks_claude_processing ON work_blocks(claude_processing_seconds, start_time) WHERE claude_processing_seconds > 0;

-- Database metadata and versioning
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    description TEXT NOT NULL
);

-- Insert initial schema version
INSERT OR IGNORE INTO schema_version (version, description) 
VALUES (1, 'Initial SQLite schema with sessions, work blocks, projects, and activity events');

-- Views for common queries (optional, for reporting convenience)

-- Active sessions view
CREATE VIEW IF NOT EXISTS active_sessions AS
SELECT 
    s.*,
    u.username,
    COUNT(wb.id) as work_block_count,
    COALESCE(SUM(wb.duration_hours), 0) as total_work_hours
FROM sessions s
JOIN users u ON s.user_id = u.id
LEFT JOIN work_blocks wb ON s.id = wb.session_id
WHERE s.state = 'active' 
  AND datetime('now') <= s.end_time
GROUP BY s.id, u.username;

-- Daily work summary view
CREATE VIEW IF NOT EXISTS daily_work_summary AS
SELECT 
    DATE(wb.start_time, 'localtime') as work_date,
    wb.project_id,
    p.name as project_name,
    COUNT(wb.id) as work_blocks,
    SUM(wb.duration_hours) as total_hours,
    SUM(wb.claude_processing_hours) as claude_hours,
    SUM(wb.activity_count) as total_activities
FROM work_blocks wb
JOIN projects p ON wb.project_id = p.id
WHERE wb.end_time IS NOT NULL -- Only finished work blocks
GROUP BY work_date, wb.project_id, p.name
ORDER BY work_date DESC, total_hours DESC;

-- Project activity summary view
CREATE VIEW IF NOT EXISTS project_activity_summary AS
SELECT 
    p.id as project_id,
    p.name as project_name,
    p.path as project_path,
    COUNT(DISTINCT wb.session_id) as unique_sessions,
    COUNT(wb.id) as total_work_blocks,
    SUM(wb.duration_hours) as total_work_hours,
    SUM(wb.claude_processing_hours) as total_claude_hours,
    MIN(wb.start_time) as first_work_time,
    MAX(wb.last_activity_time) as last_work_time,
    SUM(wb.activity_count) as total_activities
FROM projects p
LEFT JOIN work_blocks wb ON p.id = wb.project_id
GROUP BY p.id, p.name, p.path
ORDER BY total_work_hours DESC;