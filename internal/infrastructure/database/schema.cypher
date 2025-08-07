/**
 * CONTEXT:   KuzuDB graph schema for Claude Monitor work hour tracking system
 * INPUT:     Activity events from Claude Code hooks, session data, work blocks
 * OUTPUT:    Optimized graph schema for work analytics and reporting queries
 * BUSINESS:  Track 5-hour sessions, 5-minute idle work blocks, and project analytics
 * CHANGE:    Initial schema design optimized for work hour analytics and reporting
 * RISK:      Low - Schema focused on core work tracking with efficient query patterns
 */

-- =============================================================================
-- NODE TABLES: Core entities for work tracking
-- =============================================================================

/**
 * CONTEXT:   User node table for identifying work session owners
 * INPUT:     User identification data from environment and activity events
 * OUTPUT:    User nodes with consistent identification across sessions
 * BUSINESS:  Users own sessions and work blocks, enabling user-specific analytics
 * CHANGE:    Initial user node design with basic identification fields
 * RISK:      Low - Simple user identification with no sensitive data
 */
CREATE NODE TABLE User(
    id STRING,
    name STRING,
    email STRING DEFAULT '',
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

/**
 * CONTEXT:   Project node table for organizing work by project/directory
 * INPUT:     Project information auto-detected from working directory
 * OUTPUT:    Project nodes with path normalization and type detection
 * BUSINESS:  Projects provide organizational structure for work analytics
 * CHANGE:    Initial project node design with path handling and type classification
 * RISK:      Low - Project metadata with path validation
 */
CREATE NODE TABLE Project(
    id STRING,
    name STRING,
    path STRING,
    normalized_path STRING,
    project_type STRING DEFAULT 'general',
    description STRING DEFAULT '',
    last_active_time TIMESTAMP,
    total_work_blocks INT64 DEFAULT 0,
    total_hours DOUBLE DEFAULT 0.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

/**
 * CONTEXT:   Session node table implementing 5-hour window business logic
 * INPUT:     Session timing data with exact 5-hour duration enforcement
 * OUTPUT:    Session nodes with state management and activity tracking
 * BUSINESS:  Sessions are exactly 5 hours long, starting from first interaction
 * CHANGE:    Initial session node design with business rule enforcement
 * RISK:      Low - Time-based session management with validation
 */
CREATE NODE TABLE Session(
    id STRING,
    user_id STRING,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    state STRING DEFAULT 'active', -- active, expired, finished
    first_activity_time TIMESTAMP,
    last_activity_time TIMESTAMP,
    activity_count INT64 DEFAULT 1,
    duration_hours DOUBLE DEFAULT 5.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

/**
 * CONTEXT:   WorkBlock node table implementing 5-minute idle detection logic
 * INPUT:     Work block timing data with project association and activity tracking
 * OUTPUT:    Work block nodes with idle detection and duration calculation
 * BUSINESS:  Work blocks track active work periods, ending after 5 minutes of inactivity
 * CHANGE:    Initial work block node design with idle detection business rules
 * RISK:      Low - Work period tracking with time-based state management
 */
CREATE NODE TABLE WorkBlock(
    id STRING,
    session_id STRING,
    project_id STRING,
    project_name STRING,
    project_path STRING,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    state STRING DEFAULT 'active', -- active, idle, finished
    last_activity_time TIMESTAMP,
    activity_count INT64 DEFAULT 1,
    duration_seconds INT64 DEFAULT 0,
    duration_hours DOUBLE DEFAULT 0.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

/**
 * CONTEXT:   ActivityEvent node table for storing hook events and activity tracking
 * INPUT:     Activity events from Claude Code hooks with timing and context
 * OUTPUT:    Activity event nodes for audit trail and pattern analysis
 * BUSINESS:  Activity events are primary input for session and work block updates
 * CHANGE:    Initial activity event node design for comprehensive event logging
 * RISK:      Low - Event logging with metadata support for analytics
 */
CREATE NODE TABLE ActivityEvent(
    id STRING,
    user_id STRING,
    session_id STRING DEFAULT '',
    work_block_id STRING DEFAULT '',
    project_path STRING DEFAULT '',
    project_name STRING DEFAULT '',
    activity_type STRING DEFAULT 'other',
    activity_source STRING DEFAULT 'hook',
    timestamp TIMESTAMP,
    command STRING DEFAULT '',
    description STRING DEFAULT '',
    metadata STRING DEFAULT '{}', -- JSON string for additional data
    created_at TIMESTAMP,
    PRIMARY KEY (id)
);

-- =============================================================================
-- RELATIONSHIP TABLES: Connections between entities for graph analytics
-- =============================================================================

/**
 * CONTEXT:   User to Session relationship for session ownership tracking
 * INPUT:     User-session associations with creation timestamps
 * OUTPUT:    Ownership relationships enabling user-specific session queries
 * BUSINESS:  Users own sessions for analytics and access control
 * CHANGE:    Initial user-session relationship with temporal tracking
 * RISK:      Low - Simple ownership relationship with timestamp
 */
CREATE REL TABLE HAS_SESSION(
    FROM User TO Session,
    created_at TIMESTAMP DEFAULT current_timestamp()
);

/**
 * CONTEXT:   Session to WorkBlock relationship for containment hierarchy
 * INPUT:     Session-work block associations with sequence tracking
 * OUTPUT:    Containment relationships enabling session-based work analytics
 * BUSINESS:  Sessions contain work blocks in chronological order
 * CHANGE:    Initial session-work block relationship with sequence numbers
 * RISK:      Low - Hierarchical relationship with ordering support
 */
CREATE REL TABLE CONTAINS_WORK(
    FROM Session TO WorkBlock,
    sequence_number INT64 DEFAULT 0,
    created_at TIMESTAMP DEFAULT current_timestamp()
);

/**
 * CONTEXT:   WorkBlock to Project relationship for project-based analytics
 * INPUT:     Work block-project associations with activity type classification
 * OUTPUT:    Project relationships enabling project-level time tracking
 * BUSINESS:  Work blocks are performed within specific projects
 * CHANGE:    Initial work block-project relationship with activity classification
 * RISK:      Low - Project association with activity type metadata
 */
CREATE REL TABLE WORK_IN_PROJECT(
    FROM WorkBlock TO Project,
    activity_type STRING DEFAULT 'claude_action',
    created_at TIMESTAMP DEFAULT current_timestamp()
);

/**
 * CONTEXT:   User to Project relationship for user-project work history
 * INPUT:     User-project associations with cumulative work metrics
 * OUTPUT:    Work history relationships enabling user-project analytics
 * BUSINESS:  Track total user engagement with specific projects over time
 * CHANGE:    Initial user-project relationship with cumulative metrics
 * RISK:      Low - Historical relationship with aggregated work data
 */
CREATE REL TABLE WORKS_ON(
    FROM User TO Project,
    first_activity TIMESTAMP,
    last_activity TIMESTAMP,
    total_hours DOUBLE DEFAULT 0.0,
    total_work_blocks INT64 DEFAULT 0
);

/**
 * CONTEXT:   Session to ActivityEvent relationship for event tracking
 * INPUT:     Session-activity associations for audit and analysis
 * OUTPUT:    Event relationships enabling session-based activity queries
 * BUSINESS:  Sessions are triggered and updated by activity events
 * CHANGE:    Initial session-activity relationship for event correlation
 * RISK:      Low - Event tracking relationship for audit purposes
 */
CREATE REL TABLE TRIGGERED_BY(
    FROM Session TO ActivityEvent,
    event_type STRING DEFAULT 'activity',
    created_at TIMESTAMP DEFAULT current_timestamp()
);

/**
 * CONTEXT:   WorkBlock to ActivityEvent relationship for detailed activity tracking
 * INPUT:     Work block-activity associations for granular analysis
 * OUTPUT:    Activity relationships enabling work block-level event queries
 * BUSINESS:  Work blocks are updated by individual activity events
 * CHANGE:    Initial work block-activity relationship for detailed tracking
 * RISK:      Low - Detailed event tracking for work block analysis
 */
CREATE REL TABLE GENERATED_BY(
    FROM WorkBlock TO ActivityEvent,
    activity_sequence INT64 DEFAULT 0,
    created_at TIMESTAMP DEFAULT current_timestamp()
);

-- =============================================================================
-- INDEXES: Performance optimization for reporting queries
-- =============================================================================

/**
 * CONTEXT:   Database indexes for optimizing work tracking and reporting queries
 * INPUT:     Query patterns for sessions, work blocks, and project analytics
 * OUTPUT:    Optimized indexes for < 100ms query response times
 * BUSINESS:  Fast reporting queries are essential for good user experience
 * CHANGE:    Initial index design focused on time-based and project-based queries
 * RISK:      Low - Standard indexes for common query patterns
 */

-- Time-based indexes for session queries
CREATE INDEX idx_session_start_time ON Session(start_time);
CREATE INDEX idx_session_end_time ON Session(end_time);
CREATE INDEX idx_session_user_time ON Session(user_id, start_time);
CREATE INDEX idx_session_active ON Session(is_active, state);

-- Work block indexes for project and time analytics
CREATE INDEX idx_workblock_start_time ON WorkBlock(start_time);
CREATE INDEX idx_workblock_end_time ON WorkBlock(end_time);
CREATE INDEX idx_workblock_project_time ON WorkBlock(project_id, start_time);
CREATE INDEX idx_workblock_session ON WorkBlock(session_id);
CREATE INDEX idx_workblock_active ON WorkBlock(is_active, state);

-- Project indexes for work allocation queries
CREATE INDEX idx_project_name ON Project(name);
CREATE INDEX idx_project_path ON Project(normalized_path);
CREATE INDEX idx_project_active_time ON Project(is_active, last_active_time);
CREATE INDEX idx_project_hours ON Project(total_hours);

-- Activity event indexes for pattern analysis
CREATE INDEX idx_activity_timestamp ON ActivityEvent(timestamp);
CREATE INDEX idx_activity_user_time ON ActivityEvent(user_id, timestamp);
CREATE INDEX idx_activity_project ON ActivityEvent(project_name);
CREATE INDEX idx_activity_type ON ActivityEvent(activity_type);

-- User indexes for user-specific queries
CREATE INDEX idx_user_id ON User(id);
CREATE INDEX idx_user_name ON User(name);

-- =============================================================================
-- SCHEMA VALIDATION AND CONSTRAINTS
-- =============================================================================

/**
 * CONTEXT:   Schema validation rules and business constraints
 * INPUT:     Data integrity requirements from business logic
 * OUTPUT:    Database constraints ensuring data consistency
 * BUSINESS:  Enforce business rules at database level for data integrity
 * CHANGE:    Initial constraint design for core business rule enforcement
 * RISK:      Medium - Constraints prevent invalid data but may impact performance
 */

-- Session duration constraint (must be exactly 5 hours = 18000 seconds)
-- Note: KuzuDB constraints are enforced through application logic

-- Work block duration constraints (duration must be positive)
-- Note: Validated in application layer with business logic

-- Project path uniqueness (normalized paths should be unique per user)
-- Note: Enforced through application logic and unique project ID generation

-- Activity event timestamp validation (cannot be too far in past/future)
-- Note: Validated in entity layer before persistence

-- =============================================================================
-- SAMPLE QUERIES: Examples of optimized queries for work analytics
-- =============================================================================

/**
 * CONTEXT:   Sample queries demonstrating schema usage for work analytics
 * INPUT:     Various reporting and analytics query patterns
 * OUTPUT:    Example Cypher queries optimized for performance
 * BUSINESS:  Provide examples of efficient queries for common use cases
 * CHANGE:    Initial query examples for development and optimization reference
 * RISK:      Low - Documentation queries for reference only
 */

/*
-- Get daily work report for specific user
MATCH (u:User {id: $user_id})-[:HAS_SESSION]->(s:Session)
WHERE DATE(s.start_time) = DATE($date)
OPTIONAL MATCH (s)-[:CONTAINS_WORK]->(w:WorkBlock)-[:WORK_IN_PROJECT]->(p:Project)
RETURN 
    s.id as session_id,
    s.start_time,
    s.end_time,
    p.name as project_name,
    SUM(w.duration_hours) as total_work_hours,
    COUNT(w) as work_blocks,
    MIN(w.start_time) as first_activity,
    MAX(w.end_time) as last_activity
ORDER BY s.start_time;

-- Get project time breakdown for time range
MATCH (u:User {id: $user_id})-[:WORKS_ON]->(p:Project)<-[:WORK_IN_PROJECT]-(w:WorkBlock)
WHERE w.start_time >= $start_date AND w.end_time <= $end_date
RETURN 
    p.name as project_name,
    p.path as project_path,
    SUM(w.duration_hours) as total_hours,
    COUNT(w) as work_blocks,
    COUNT(DISTINCT DATE(w.start_time)) as active_days,
    MIN(w.start_time) as first_activity,
    MAX(w.end_time) as last_activity
ORDER BY total_hours DESC;

-- Get hourly productivity distribution
MATCH (w:WorkBlock)
WHERE w.start_time >= $start_date AND w.end_time <= $end_date
RETURN 
    HOUR(w.start_time) as hour,
    SUM(w.duration_hours) as total_hours,
    COUNT(w) as work_blocks,
    AVG(w.duration_hours) as avg_block_size
ORDER BY hour;

-- Get weekly work trends
MATCH (s:Session)-[:CONTAINS_WORK]->(w:WorkBlock)
WHERE w.start_time >= $start_date AND w.end_time <= $end_date
WITH DATE_TRUNC('week', w.start_time) as week_start, w
RETURN 
    week_start,
    SUM(w.duration_hours) as weekly_hours,
    COUNT(DISTINCT s) as sessions,
    COUNT(w) as work_blocks,
    COUNT(DISTINCT w.project_name) as projects
ORDER BY week_start;
*/