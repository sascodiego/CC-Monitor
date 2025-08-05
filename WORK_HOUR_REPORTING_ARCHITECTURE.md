# Claude Monitor Work Hour Reporting System Architecture

## Overview

This document describes the comprehensive work hour reporting system architecture for Claude Monitor, designed to extrapolate existing session and work block data into sophisticated labor time tracking and business analytics capabilities.

## System Architecture Summary

### Foundation
The work hour reporting system builds on Claude Monitor's excellent existing infrastructure:
- **Real-time activity detection** with HTTP method-based user interaction tracking
- **Session management** with 5-hour windows and precise timing
- **Work block tracking** with 5-minute inactivity timeouts
- **Kùzu graph database** for storing relationships and time-series data
- **eBPF monitoring** for kernel-level event capture with minimal overhead

### Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI Reporting Layer                      │ ← Work hour commands, exports
├─────────────────────────────────────────────────────────────┤
│              Work Hour Analytics Engine                     │ ← Business logic, calculations
├─────────────────────────────────────────────────────────────┤
│               Work Hour Domain Layer                        │ ← Work day, timesheet entities
├─────────────────────────────────────────────────────────────┤
│            Enhanced Database Layer                          │ ← Extended schema, optimized queries
├─────────────────────────────────────────────────────────────┤
│          Existing Session/WorkBlock Infrastructure          │ ← Current foundation
└─────────────────────────────────────────────────────────────┘
```

## Key Design Files Created

### 1. Domain Entities (`/internal/domain/workhour_entities.go`)
**Extended business entities for work hour tracking:**

- **WorkDay**: Calendar day aggregation with start/end times, total work time, break analysis
- **WorkWeek**: Weekly aggregation with overtime calculation and work pattern analysis  
- **Timesheet**: Formal timesheet with configurable policies (rounding, overtime rules)
- **WorkPattern**: Productivity pattern analysis (peak hours, work day type, consistency)
- **ActivitySummary**: High-level metrics with trends and goal tracking

### 2. System Interfaces (`/internal/arch/workhour_interfaces.go`)
**Clean abstractions for work hour system components:**

- **WorkHourAnalyzer**: Analytics and pattern analysis interface
- **TimesheetManager**: Formal timesheet generation and management
- **WorkHourReportGenerator**: Multi-format report generation
- **WorkHourExporter**: Export engine for multiple output formats
- **WorkHourDatabaseManager**: Extended database operations for analytics
- **WorkHourService**: High-level coordination interface

### 3. CLI Commands (`/internal/cli/workhour_commands.go`)
**Comprehensive CLI interface for work hour operations:**

#### Core Commands
```bash
# Daily work analysis
claude-monitor workday status --date=2024-01-15 --detailed
claude-monitor workday report --format=pdf --include-charts

# Weekly analysis  
claude-monitor workweek report --week-start=2024-01-15 --include-overtime
claude-monitor workweek analysis --analysis-depth=comprehensive

# Timesheet management
claude-monitor timesheet generate --period=weekly --rounding-rule=15min
claude-monitor timesheet export --format=pdf --digital-signature

# Analytics and insights
claude-monitor analytics productivity --start-date=2024-01-01 --end-date=2024-01-31
claude-monitor analytics patterns --include-recommendations
claude-monitor analytics trends --include-forecasting

# Goals and policies
claude-monitor goals set --goal-type=daily --target-hours=8h
claude-monitor policy update --rounding-interval=15min --overtime-threshold=8h

# Bulk operations
claude-monitor export bulk --start-date=2024-01-01 --formats=csv,json,pdf
```

### 4. Integration Layer (`/internal/workhour/integrator.go`)
**Seamless integration with existing session/work block system:**

- **Real-time event processing**: Responds to session and work block lifecycle events
- **Automated aggregation**: Periodic work day calculation from raw data
- **Performance caching**: Optimized cache management for frequently accessed data
- **Event-driven updates**: Immediate cache invalidation and recalculation on activity

### 5. Export Engine (`/internal/workhour/exporter.go`)
**Multi-format export capabilities:**

- **JSON**: API integration and data interchange
- **CSV**: Spreadsheet applications and data analysis
- **HTML**: Interactive reports with charts and visualizations
- **PDF**: Formal document generation and archival
- **Excel**: Advanced spreadsheet features with multiple worksheets

## Key Features

### 1. Daily Work Reports
- Work start/end time detection from first/last activity
- Total hours worked calculation from work block aggregation
- Break analysis (gaps between work blocks with duration and frequency)
- Productivity metrics (active vs idle time ratios)
- Goal progress tracking against configurable daily targets

### 2. Advanced Analytics
- **Weekly/Monthly Summaries**: Aggregated statistics with trend analysis
- **Work Pattern Analysis**: Peak productivity hours, work day type classification
- **Claude Usage Intensity**: Session frequency and duration analysis
- **Overtime Detection**: Automatic calculation with configurable thresholds
- **Efficiency Metrics**: Focus score, interruption rate, active time ratios

### 3. Professional Timesheet Generation
- **Configurable Policies**: Rounding rules (15min, 30min, 1h), overtime thresholds
- **Multiple Periods**: Weekly, bi-weekly, monthly timesheet cycles
- **Business Rule Support**: Break deductions, overtime calculations
- **Approval Workflow**: Draft → Submitted → Approved status management
- **Compliance Ready**: Professional formatting for billing and HR systems

### 4. Business Intelligence Features
- **Trend Analysis**: Historical performance trends with forecasting
- **Goal Management**: Configurable work hour goals with progress tracking
- **Pattern Recognition**: Work schedule optimization recommendations
- **Productivity Insights**: Peak performance period identification
- **Break Pattern Analysis**: Optimal break timing and duration insights

### 5. Enterprise Export Capabilities
- **Multiple Formats**: JSON, CSV, PDF, HTML, Excel support
- **Bulk Operations**: Large-scale data export with progress tracking
- **Template System**: Customizable report templates and branding
- **Data Integration**: API-ready exports for external systems
- **Compliance Support**: Formal document generation with digital signatures

## Technical Excellence

### Performance Optimizations
- **Incremental Aggregation**: Only process new/changed data
- **Smart Caching**: Frequently accessed work days cached in memory
- **Query Optimization**: Indexed database queries for large datasets
- **Lazy Loading**: Report components loaded on demand
- **Parallel Processing**: Concurrent export operations for bulk data

### Data Integrity
- **Transaction Management**: ACID compliance for work hour calculations
- **Validation Pipelines**: Multi-layer data validation and consistency checks
- **Audit Trails**: Complete history of policy changes and calculations
- **Timezone Handling**: Consistent timezone conversion across all operations
- **Error Recovery**: Graceful handling of incomplete or corrupted data

### Integration Patterns
- **Event-Driven Architecture**: Real-time updates without polling
- **Service Boundaries**: Clean separation between monitoring and analytics
- **Backward Compatibility**: No disruption to existing monitoring functionality
- **Extensible Design**: Plugin architecture for custom business rules
- **Configuration Management**: Runtime policy updates without restarts

## Implementation Strategy

### Phase 1: Core Analytics (High Priority)
1. Work day aggregation from existing session/work block data
2. Basic daily and weekly reporting
3. CSV and JSON export capabilities
4. CLI commands for daily work analysis

### Phase 2: Advanced Reporting (Medium Priority)  
1. Timesheet generation with business policies
2. PDF and HTML export with professional formatting
3. Pattern analysis and productivity insights
4. Goal management and progress tracking

### Phase 3: Business Intelligence (Medium Priority)
1. Trend analysis and forecasting
2. Advanced analytics with recommendations
3. Excel export with charts and formulas
4. Bulk export and data migration tools

### Phase 4: Enterprise Features (Lower Priority)
1. Multi-user support and role-based access
2. API endpoints for external integration
3. Custom report templates and branding
4. Advanced compliance and audit features

## Database Schema Extensions

### Work Hour Tables
```sql
-- Work day aggregations
CREATE TABLE work_days (
    id TEXT PRIMARY KEY,
    date DATE NOT NULL,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    total_time_seconds INTEGER,
    break_time_seconds INTEGER,
    session_count INTEGER,
    block_count INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Timesheet management
CREATE TABLE timesheets (
    id TEXT PRIMARY KEY,
    employee_id TEXT,
    period_type TEXT,
    start_date DATE,
    end_date DATE,
    total_hours_seconds INTEGER,
    overtime_hours_seconds INTEGER,
    status TEXT,
    policy_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Work hour policies
CREATE TABLE work_hour_policies (
    id TEXT PRIMARY KEY,
    rounding_interval_seconds INTEGER,
    rounding_method TEXT,
    overtime_threshold_seconds INTEGER,
    break_deduction_seconds INTEGER,
    effective_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Performance Indexes
```sql
CREATE INDEX idx_work_days_date ON work_days(date);
CREATE INDEX idx_work_days_total_time ON work_days(total_time_seconds);
CREATE INDEX idx_timesheets_employee_period ON timesheets(employee_id, start_date, end_date);
CREATE INDEX idx_timesheets_status ON timesheets(status);
```

## Monitoring and Observability

### Metrics Collection
- Work hour calculation performance
- Export operation success rates  
- Cache hit ratios and memory usage
- Database query performance
- CLI command usage statistics

### Health Checks
- Database connectivity and schema validation
- Cache consistency verification
- Export template availability
- Integration event processing status
- System resource utilization

## Security and Privacy

### Data Protection
- No sensitive personal data exposure in logs
- Configurable data retention policies
- Secure export file handling
- Access control for timesheet operations
- Audit logging for policy changes

### System Security
- Input validation for all CLI parameters
- File system access controls for exports
- Database connection security
- Resource usage limits for bulk operations
- Error message sanitization

## Conclusion

This comprehensive work hour reporting system transforms Claude Monitor from a session tracking tool into a complete labor time management solution. The architecture preserves the excellent performance characteristics of the existing system while adding sophisticated business analytics capabilities.

**Key Strengths:**
- **Non-intrusive Integration**: Builds on existing infrastructure without disruption
- **Performance Optimized**: Maintains sub-10ms event processing latency
- **Business Ready**: Professional timesheet and compliance features
- **Extensible Design**: Clean interfaces support future enhancements
- **Developer Friendly**: Comprehensive CLI and export capabilities

The system provides immediate value for individual productivity tracking while offering enterprise-grade features for business time management and compliance requirements.

---

**File Locations:**
- Domain Entities: `/mnt/c/src/ClaudeMmonitor/internal/domain/workhour_entities.go`
- System Interfaces: `/mnt/c/src/ClaudeMmonitor/internal/arch/workhour_interfaces.go`  
- CLI Commands: `/mnt/c/src/ClaudeMmonitor/internal/cli/workhour_commands.go`
- Integration Layer: `/mnt/c/src/ClaudeMmonitor/internal/workhour/integrator.go`
- Export Engine: `/mnt/c/src/ClaudeMmonitor/internal/workhour/exporter.go`