# Claude Monitor - Specialized Agents

This directory contains specialized agents for developing and maintaining the Claude Monitor system. Each agent is an expert in a specific area of the project, designed to provide highly specialized and contextual assistance.

## 🎯 Available Agents

### **🏗️ Architecture Designer**
**File**: `architecture-designer.md`  
**Specialty**: System architecture analysis and design  
**Use when**: You need to evaluate system architecture, design new components, plan integrations, or make strategic architectural decisions.

**Usage examples**:
- Current architectural pattern analysis
- Module integration design
- Technical decision evaluation
- System evolution planning

### **💻 Software Engineer**
**File**: `software-engineer.md`  
**Specialty**: Software implementation and technical development  
**Use when**: You need to implement new features, debug, optimize performance, or solve technical problems.

**Usage examples**:
- New feature development
- Code and performance optimization
- Debugging and issue resolution
- Complex algorithm implementation

### **📈 Productivity Specialist**
**File**: `productivity-specialist.md`  
**Specialty**: Productivity analysis and UX optimization  
**Use when**: You need to analyze work patterns, improve workflows, design user experiences, or generate productivity insights.

**Usage examples**:
- Developer work pattern analysis
- More efficient CLI interface design
- Actionable insights generation
- Workflow and process optimization

### **🧹 Clean Code Analyst**
**File**: `clean-code-analyst.md`  
**Specialty**: Code quality analysis and clean code  
**Use when**: You need to evaluate code quality, identify refactoring opportunities, manage technical debt, or improve maintainability.

**Usage examples**:
- Complexity and code smell analysis
- Refactoring planning
- Technical debt management
- Quality standards establishment

### **🗄️ KuzuDB Specialist**
**File**: `kuzudb-specialist.md`  
**Specialty**: Graph database operations and Cypher queries  
**Use when**: You need to work with KuzuDB schema design, write Cypher queries, optimize database performance, or integrate KuzuDB with Go.

**Usage examples**:
- Graph schema design for work tracking
- Efficient reporting query optimization
- Database performance tuning
- Go driver integration patterns

### **🔄 Go Concurrency Specialist**
**File**: `go-concurrency-specialist.md`  
**Specialty**: Goroutines, channels, and concurrent systems  
**Use when**: You need to design concurrent systems, fix race conditions, implement worker pools, or optimize concurrent performance.

**Usage examples**:
- Concurrent event processing design
- Race condition detection and fixes
- Worker pool implementation
- Thread-safe data structure design

### **🪝 Hook Integration Specialist**
**File**: `hook-integration-specialist.md`  
**Specialty**: Claude Code hook system and activity detection  
**Use when**: You need to implement hook commands, ensure activity detection accuracy, design fallback mechanisms, or optimize hook performance.

**Usage examples**:
- Hook command implementation
- Activity detection patterns
- Fallback mechanism design
- Context detection optimization

### **🧪 Testing Specialist**
**File**: `testing-specialist.md`  
**Specialty**: Go testing, TDD, and quality assurance  
**Use when**: You need to write tests, implement mocks, design test strategies, or improve test coverage.

**Usage examples**:
- Table-driven test implementation
- Integration test design
- Mock and stub creation
- Test coverage improvement

### **🌐 HTTP API Specialist**
**File**: `http-api-specialist.md`  
**Specialty**: RESTful APIs and gorilla/mux  
**Use when**: You need to design API endpoints, implement HTTP servers, create middleware, or optimize API performance.

**Usage examples**:
- RESTful endpoint design
- Middleware implementation
- CORS and security configuration
- API performance optimization

### **👹 Daemon Service Specialist**
**File**: `daemon-service-specialist.md`  
**Specialty**: Background services and systemd integration  
**Use when**: You need to implement daemon processes, handle signals, integrate with systemd, or ensure service reliability.

**Usage examples**:
- Daemon lifecycle management
- Systemd service unit creation
- Signal handling implementation
- Health check and monitoring setup

### **🎨 CLI UX Specialist**
**File**: `cli-ux-specialist.md`  
**Specialty**: Cobra CLI and beautiful terminal output  
**Use when**: You need to design CLI commands, create beautiful output, implement interactive features, or improve user experience.

**Usage examples**:
- Command structure design
- Beautiful table and color formatting
- Progress indicator implementation
- Shell completion scripts

## 🚀 How to Use the Agents

### **Direct Invocation**
```bash
# Use the architecture designer
claude "Use the architecture-designer agent to evaluate the current system architecture"

# Use the go concurrency specialist
claude "Use the go-concurrency-specialist to fix race conditions in the event processor"

# Use the hook integration specialist
claude "Use the hook-integration-specialist to implement reliable activity detection"

# Use the testing specialist
claude "Use the testing-specialist to improve test coverage for the session manager"

# Use the HTTP API specialist
claude "Use the http-api-specialist to design RESTful endpoints for reporting"

# Use the daemon service specialist
claude "Use the daemon-service-specialist to implement graceful shutdown"

# Use the CLI UX specialist
claude "Use the cli-ux-specialist to create beautiful report output"
```

### **Workflow Integration**

Agents activate automatically when:

1. **Context Detection**: The system automatically recognizes when an agent is most appropriate for the task
2. **Keywords**: Certain words trigger specific agent activation (marked with "Use PROACTIVELY")
3. **Problem Type**: The nature of the problem determines which agent to use

## 📋 Specialization Matrix

| Area | Architecture | Software | Productivity | Clean Code | KuzuDB | Concurrency | Hooks | Testing | HTTP API | Daemon | CLI UX |
|------|-------------|----------|--------------|------------|---------|-------------|-------|---------|----------|---------|--------|
| **System Design** | ✅ | 🔹 | ⚪ | 🔹 | 🔹 | 🔹 | ⚪ | ⚪ | 🔹 | 🔹 | ⚪ |
| **Go Implementation** | 🔹 | ✅ | ⚪ | 🔹 | 🔹 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Database/KuzuDB** | 🔹 | 🔹 | ⚪ | ⚪ | ✅ | ⚪ | ⚪ | 🔹 | ⚪ | ⚪ | ⚪ |
| **Concurrency** | 🔹 | 🔹 | ⚪ | ⚪ | ⚪ | ✅ | 🔹 | 🔹 | 🔹 | 🔹 | ⚪ |
| **Hook System** | ⚪ | 🔹 | ⚪ | ⚪ | ⚪ | ⚪ | ✅ | 🔹 | ⚪ | 🔹 | ⚪ |
| **Testing** | 🔹 | 🔹 | ⚪ | 🔹 | 🔹 | 🔹 | 🔹 | ✅ | 🔹 | 🔹 | 🔹 |
| **HTTP/API** | 🔹 | 🔹 | ⚪ | ⚪ | ⚪ | 🔹 | 🔹 | 🔹 | ✅ | 🔹 | ⚪ |
| **Services/Daemon** | 🔹 | 🔹 | ⚪ | ⚪ | ⚪ | 🔹 | ⚪ | 🔹 | 🔹 | ✅ | ⚪ |
| **CLI/UX** | ⚪ | 🔹 | ✅ | ⚪ | ⚪ | ⚪ | ⚪ | 🔹 | ⚪ | ⚪ | ✅ |
| **Performance** | 🔹 | ✅ | 🔹 | 🔹 | ✅ | ✅ | 🔹 | 🔹 | ✅ | 🔹 | ⚪ |

**Legend**: ✅ Expert | 🔹 Advanced | ⚪ Basic

## 🔄 Agent Collaboration

Agents are designed to work collaboratively:

### **Typical Workflows**

#### **New Feature Development**
1. **Architecture Designer**: Design system integration
2. **Hook Integration Specialist**: Implement activity detection
3. **Go Concurrency Specialist**: Design concurrent processing
4. **Software Engineer**: Implement core functionality
5. **Testing Specialist**: Create comprehensive tests
6. **CLI UX Specialist**: Design user interface

#### **Database and Reporting**
1. **KuzuDB Specialist**: Design graph schema
2. **HTTP API Specialist**: Create reporting endpoints
3. **Software Engineer**: Implement business logic
4. **CLI UX Specialist**: Create beautiful reports
5. **Testing Specialist**: Integration testing

#### **Service Implementation**
1. **Daemon Service Specialist**: Design service architecture
2. **Go Concurrency Specialist**: Implement concurrent processing
3. **HTTP API Specialist**: Create API endpoints
4. **Hook Integration Specialist**: Integrate with Claude Code
5. **Testing Specialist**: End-to-end testing

## 📊 Effectiveness Metrics

### **KPIs by Agent**

#### **Core Development Agents**
- **Software Engineer**: Test coverage > 80%, implementation time ±20% of estimate
- **Go Concurrency**: Zero race conditions, < 10% CPU usage
- **Testing Specialist**: > 80% coverage, zero flaky tests
- **Clean Code Analyst**: Code quality score > 85/100

#### **Infrastructure Agents**
- **KuzuDB Specialist**: Query response < 100ms, zero data inconsistencies
- **HTTP API Specialist**: API response < 100ms for 95th percentile
- **Daemon Service**: Zero downtime, graceful shutdown < 30s
- **Hook Integration**: 100% activity capture, < 50ms overhead

#### **User Experience Agents**
- **CLI UX Specialist**: Command response < 100ms, beautiful output
- **Architecture Designer**: 100% documented decisions
- **Productivity Specialist**: > 3 improvements per analysis

## 🛠️ Agent Maintenance

### **Agent Updates**
Agents are updated based on:
- User feedback
- New best practices
- Project evolution
- Effectiveness metrics

### **Creating New Agents**
To create new agents:

1. **Identify Need**: What specialized area is missing?
2. **Define Scope**: What are the specific responsibilities?
3. **Create Template**: Follow existing agent format
4. **Define Triggers**: When should it activate automatically?
5. **Testing**: Validate effectiveness with real cases

### **Agent Structure**
```markdown
---
name: agent-name
description: Use PROACTIVELY for [triggers]. Specializes in [areas]. This agent [what it does].
tools: Read, MultiEdit, Write, Grep, Glob, Bash
model: sonnet
---

You are a [role] specializing in [expertise].

## Core Expertise
[Technical expertise description]

## Primary Responsibilities
1. [Responsibility 1]
2. [Responsibility 2]

## Technical Specialization
### [Area 1]
- [Specific expertise]

## Working Methodology
[How the agent approaches tasks]

## Quality Standards
- [Metric 1]
- [Metric 2]

## Integration Points
You work closely with:
- [Other agent]: [How you collaborate]
```

## 🔗 Useful Links

- **Main Documentation**: [CLAUDE.md](../../CLAUDE.md)
- **Project Architecture**: [PROJECT.md](../../PROJECT.md)
- **Problems and Solutions**: [PRB.md](../../PRB.md) and [PRP.md](../../PRP.md)

---

## Agent Summary

**Total Agents**: 11 specialized agents covering all aspects of the Claude Monitor system

**Core Technologies Covered**:
- Go language and concurrency
- KuzuDB graph database
- Claude Code hook system
- HTTP APIs with gorilla/mux
- Daemon services and systemd
- Cobra CLI framework
- Testing and quality assurance

**Note**: These agents are constantly evolving. If you find improvement opportunities or need a new type of specialized agent, document it for future iterations.