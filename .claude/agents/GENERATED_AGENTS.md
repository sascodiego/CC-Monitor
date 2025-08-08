# 🤖 Generated Specialist Agents for Claude Monitor

**Generated**: 2025-08-08  
**Project**: Claude Monitor System v1.0.0  
**Analyzer**: context-engineering-specialist

## Overview

Based on comprehensive architecture analysis, the following specialist agents have been generated to address specific needs identified in the Claude Monitor codebase. These agents are tailored to resolve current issues and optimize system performance.

## Generated Agents

### 1. 🗄️ sqlite-database-specialist
**Purpose**: Database optimization and query performance  
**Priority**: HIGH - Critical for resolving timeout issues  
**Key Responsibilities**:
- Optimize slow reporting queries causing timeouts
- Design efficient indexes for time-series data
- Manage database migrations safely
- Tune SQLite configuration for embedded use

**Immediate Actions**:
- Add covering indexes for report queries
- Optimize date-based aggregations
- Implement query result caching

---

### 2. 🔧 daemon-reliability-specialist
**Purpose**: Service stability and reliability  
**Priority**: HIGH - Essential for system availability  
**Key Responsibilities**:
- Fix graceful shutdown issues
- Implement health check endpoints
- Prevent resource leaks and zombie processes
- Design error recovery mechanisms

**Immediate Actions**:
- Fix /health endpoint timeout
- Implement proper signal handling
- Add connection pool monitoring

---

### 3. 🔍 debugging-diagnostics-specialist
**Purpose**: Root cause analysis and troubleshooting  
**Priority**: CRITICAL - Needed to resolve current issues  
**Key Responsibilities**:
- Diagnose endpoint timeout problems
- Trace data flow disconnections
- Profile performance bottlenecks
- Create diagnostic tools and runbooks

**Immediate Actions**:
- Investigate why activities aren't appearing in reports
- Add request tracing
- Create diagnostic endpoints

---

### 4. 🧪 integration-testing-specialist
**Purpose**: Comprehensive test coverage and regression prevention  
**Priority**: MEDIUM - Important for long-term stability  
**Key Responsibilities**:
- Create end-to-end test scenarios
- Build regression test suite
- Design test fixtures and data generators
- Establish CI/CD testing pipeline

**Immediate Actions**:
- Test complete activity lifecycle
- Add tests for timeout scenarios
- Create data flow validation tests

---

### 5. ⚡ performance-optimization-specialist
**Purpose**: System performance and efficiency  
**Priority**: MEDIUM - Required for production readiness  
**Key Responsibilities**:
- Achieve <10ms hook execution time
- Optimize database query performance
- Implement intelligent caching
- Reduce memory footprint

**Immediate Actions**:
- Optimize hook HTTP calls
- Add connection pooling
- Implement activity batching

## Recommended Workflow

### Phase 1: Critical Issue Resolution (Immediate)
1. Deploy **debugging-diagnostics-specialist** to identify root causes
2. Use **sqlite-database-specialist** to fix query performance
3. Apply **daemon-reliability-specialist** for service stability

### Phase 2: Stabilization (Short-term)
1. **performance-optimization-specialist** optimizes critical paths
2. **integration-testing-specialist** creates regression tests
3. **daemon-reliability-specialist** implements monitoring

### Phase 3: Optimization (Medium-term)
1. **sqlite-database-specialist** implements advanced optimizations
2. **performance-optimization-specialist** achieves performance targets
3. **integration-testing-specialist** establishes full test coverage

## Agent Collaboration Matrix

| Agent | Works With | Collaboration Focus |
|-------|------------|-------------------|
| sqlite-database | daemon-reliability | Database operation safety |
| sqlite-database | performance-optimization | Query optimization |
| debugging-diagnostics | All agents | Issue identification |
| daemon-reliability | integration-testing | Failure scenario testing |
| performance-optimization | integration-testing | Performance regression tests |

## Usage Examples

```bash
# Use debugging specialist to diagnose current issues
claude code "Use debugging-diagnostics-specialist to investigate why /health endpoint is timing out"

# Fix database performance issues
claude code "Use sqlite-database-specialist to add indexes and optimize the daily report query"

# Improve daemon reliability
claude code "Use daemon-reliability-specialist to implement proper graceful shutdown"

# Create comprehensive tests
claude code "Use integration-testing-specialist to create end-to-end tests for activity processing"

# Optimize performance
claude code "Use performance-optimization-specialist to achieve <10ms hook execution"
```

## Success Metrics

- ✅ **Issue Resolution**: All timeout issues resolved
- ✅ **Data Consistency**: 100% of activities appear in reports
- ✅ **Performance**: <10ms hook, <100ms queries
- ✅ **Reliability**: 99.9% uptime
- ✅ **Test Coverage**: >80% for critical paths

## Notes

1. These agents are specifically tailored to the current state of Claude Monitor after the SQLite refactoring
2. Each agent has deep knowledge of the specific issues identified in DIAG.md
3. Agents are designed to work together for comprehensive system improvement
4. Priority levels are based on current critical issues affecting production

## Maintenance

These agents should be updated as the system evolves:
- When major refactoring occurs
- When new technologies are introduced
- When performance requirements change
- When new issues are identified

---

**Remember**: These specialists work best when used for their specific domains. Don't hesitate to engage multiple specialists for complex issues that span multiple areas.