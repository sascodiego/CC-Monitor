/**
 * CONTEXT:   KuzuDB embedded schema for Claude Monitor single binary
 * INPUT:     Activity events from hook calls and daemon processing
 * OUTPUT:    Optimized graph schema for work hour analytics and reporting
 * BUSINESS:  Track 5-hour sessions, 5-minute idle work blocks, and project analytics
 * CHANGE:    Embedded schema copy optimized for single binary distribution
 * RISK:      Low - Read-only schema definition embedded in binary
 */

-- =============================================================================
-- NODE TABLES: Core entities for work tracking
-- =============================================================================

CREATE NODE TABLE User(
    id STRING,
    name STRING,
    email STRING DEFAULT '',
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

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

CREATE NODE TABLE Session(
    id STRING,
    user_id STRING,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    state STRING DEFAULT 'active',
    first_activity_time TIMESTAMP,
    last_activity_time TIMESTAMP,
    activity_count INT64 DEFAULT 1,
    duration_hours DOUBLE DEFAULT 5.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE NODE TABLE WorkBlock(
    id STRING,
    session_id STRING,
    project_id STRING,
    project_name STRING,
    project_path STRING,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    state STRING DEFAULT 'active',
    last_activity_time TIMESTAMP,
    activity_count INT64 DEFAULT 1,
    duration_seconds INT64 DEFAULT 0,
    duration_hours DOUBLE DEFAULT 0.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (id)
);

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
    metadata STRING DEFAULT '{}',
    created_at TIMESTAMP,
    PRIMARY KEY (id)
);

-- =============================================================================
-- RELATIONSHIP TABLES: Connections between entities
-- =============================================================================

CREATE REL TABLE HAS_SESSION(
    FROM User TO Session,
    created_at TIMESTAMP DEFAULT current_timestamp()
);

CREATE REL TABLE CONTAINS_WORK(
    FROM Session TO WorkBlock,
    sequence_number INT64 DEFAULT 0,
    created_at TIMESTAMP DEFAULT current_timestamp()
);

CREATE REL TABLE WORK_IN_PROJECT(
    FROM WorkBlock TO Project,
    activity_type STRING DEFAULT 'claude_action',
    created_at TIMESTAMP DEFAULT current_timestamp()
);

CREATE REL TABLE WORKS_ON(
    FROM User TO Project,
    first_activity TIMESTAMP,
    last_activity TIMESTAMP,
    total_hours DOUBLE DEFAULT 0.0,
    total_work_blocks INT64 DEFAULT 0
);

CREATE REL TABLE TRIGGERED_BY(
    FROM Session TO ActivityEvent,
    event_type STRING DEFAULT 'activity',
    created_at TIMESTAMP DEFAULT current_timestamp()
);

CREATE REL TABLE GENERATED_BY(
    FROM WorkBlock TO ActivityEvent,
    activity_sequence INT64 DEFAULT 0,
    created_at TIMESTAMP DEFAULT current_timestamp()
);

-- =============================================================================
-- INDEXES: Performance optimization
-- =============================================================================

CREATE INDEX idx_session_start_time ON Session(start_time);
CREATE INDEX idx_session_end_time ON Session(end_time);
CREATE INDEX idx_session_user_time ON Session(user_id, start_time);
CREATE INDEX idx_session_active ON Session(is_active, state);

CREATE INDEX idx_workblock_start_time ON WorkBlock(start_time);
CREATE INDEX idx_workblock_end_time ON WorkBlock(end_time);
CREATE INDEX idx_workblock_project_time ON WorkBlock(project_id, start_time);
CREATE INDEX idx_workblock_session ON WorkBlock(session_id);
CREATE INDEX idx_workblock_active ON WorkBlock(is_active, state);

CREATE INDEX idx_project_name ON Project(name);
CREATE INDEX idx_project_path ON Project(normalized_path);
CREATE INDEX idx_project_active_time ON Project(is_active, last_active_time);
CREATE INDEX idx_project_hours ON Project(total_hours);

CREATE INDEX idx_activity_timestamp ON ActivityEvent(timestamp);
CREATE INDEX idx_activity_user_time ON ActivityEvent(user_id, timestamp);
CREATE INDEX idx_activity_project ON ActivityEvent(project_name);
CREATE INDEX idx_activity_type ON ActivityEvent(activity_type);

CREATE INDEX idx_user_id ON User(id);
CREATE INDEX idx_user_name ON User(name);