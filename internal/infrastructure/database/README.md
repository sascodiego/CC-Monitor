# KuzuDB Comprehensive Reporting System

## Overview

This is a complete, production-ready KuzuDB reporting system for Claude Monitor work hour tracking. The system provides:

- **Sub-100ms Query Performance**: Optimized queries with intelligent caching
- **Comprehensive Analytics**: Daily, weekly, monthly, and historical reports
- **Automatic Optimization**: Self-tuning queries and index recommendations
- **Performance Monitoring**: Real-time performance tracking and alerting
- **Smart Caching**: Intelligent cache management with TTL optimization
- **Graph Analytics**: Rich relationship queries for work pattern analysis

## Architecture Components

### Core Components

1. **ComprehensiveReportingService** - Main service interface
2. **ReportingQueries** - Optimized Cypher queries for all report types
3. **QueryCache** - High-performance LRU cache with TTL management
4. **PerformanceMonitor** - Real-time query performance tracking
5. **QueryOptimizer** - Automatic query optimization and index management
6. **KuzuConnectionManager** - Connection pooling and resource management

### Key Features

- **Fast Reports**: < 50ms daily, < 100ms weekly, < 300ms monthly
- **Smart Caching**: Automatic cache invalidation and optimization
- **Real-time Monitoring**: Performance metrics and slow query detection
- **Auto-optimization**: Query pattern recognition and improvement
- **Graph Analytics**: Rich relationship queries for work insights
- **Resource Management**: Proper connection pooling and cleanup

## Quick Start

### 1. Initialize the Service

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "your-project/internal/infrastructure/database"
)

func main() {
    // Configure database connection
    config := database.KuzuConnectionConfig{
        DatabasePath:   "./data/claude-monitor.kuzu",
        MaxConnections: 10,
        QueryTimeout:   60 * time.Second,
        EnableLogging:  true,
    }

    // Create comprehensive reporting service
    service, err := database.NewComprehensiveReportingService(config)
    if err != nil {
        panic(fmt.Sprintf("Failed to initialize reporting service: %v", err))
    }
    defer service.Shutdown(context.Background())

    // Service is ready for use!
    fmt.Println("Comprehensive reporting service initialized successfully")
}
```

### 2. Generate Daily Reports

```go
func generateDailyReport(service *database.ComprehensiveReportingService) {
    ctx := context.Background()
    userID := "user123"
    today := time.Now()

    // Get daily report (cached, optimized, monitored)
    report, err := service.GetDailyReport(ctx, today, userID)
    if err != nil {
        fmt.Printf("Error generating daily report: %v\n", err)
        return
    }

    // Display report data
    fmt.Printf("Daily Report for %s\n", report.Date.Format("2006-01-02"))
    fmt.Printf("Total Work Hours: %.2f\n", report.TotalWorkHours)
    fmt.Printf("Efficiency: %.1f%%\n", report.Efficiency*100)
    fmt.Printf("Projects Worked On: %d\n", len(report.ProjectBreakdown))
    
    // Project breakdown
    fmt.Println("\nProject Time Allocation:")
    for _, project := range report.ProjectBreakdown {
        fmt.Printf("  %s: %.2f hours (%.1f%%)\n", 
            project.Name, project.Hours, project.Percentage)
    }
    
    // Hourly productivity
    fmt.Println("\nHourly Productivity:")
    for _, hourly := range report.HourlyBreakdown {
        fmt.Printf("  %02d:00 - %d minutes (%s)\n", 
            hourly.Hour, hourly.WorkMinutes, hourly.ProductivityLevel)
    }
}
```

### 3. Generate Weekly Reports with Trends

```go
func generateWeeklyReport(service *database.ComprehensiveReportingService) {
    ctx := context.Background()
    userID := "user123"
    thisWeek := time.Now().Add(-time.Duration(int(time.Now().Weekday())) * 24 * time.Hour)

    report, err := service.GetWeeklyReport(ctx, thisWeek, userID)
    if err != nil {
        fmt.Printf("Error generating weekly report: %v\n", err)
        return
    }

    fmt.Printf("Weekly Report: %s to %s\n", 
        report.WeekStart.Format("Jan 2"), report.WeekEnd.Format("Jan 2, 2006"))
    fmt.Printf("Total Work Hours: %.2f\n", report.TotalWorkHours)
    fmt.Printf("Weekly Trend: %s\n", report.WeeklyTrend)
    fmt.Printf("Most Productive Day: %s\n", report.MostProductiveDay)
    fmt.Printf("Compared to Last Week: %.1f%%\n", report.ComparedToLastWeek)

    // Daily breakdown
    fmt.Println("\nDaily Breakdown:")
    for _, daily := range report.DailyBreakdown {
        fmt.Printf("  %s: %.2f hours\n", 
            daily.Date.Format("Mon Jan 2"), daily.TotalWorkHours)
    }

    // Top projects
    fmt.Println("\nTop Projects This Week:")
    for i, project := range report.TopProjects {
        if i >= 5 { break } // Top 5
        fmt.Printf("  %d. %s: %.2f hours (%.1f%%)\n", 
            i+1, project.Name, project.Hours, project.Percentage)
    }
}
```

### 4. Generate Monthly Reports with Heatmap

```go
func generateMonthlyReport(service *database.ComprehensiveReportingService) {
    ctx := context.Background()
    userID := "user123"
    thisMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)

    report, err := service.GetMonthlyReport(ctx, thisMonth, userID)
    if err != nil {
        fmt.Printf("Error generating monthly report: %v\n", err)
        return
    }

    fmt.Printf("Monthly Report: %s %d\n", thisMonth.Format("January"), report.Year)
    fmt.Printf("Total Work Hours: %.2f\n", report.TotalWorkHours)
    fmt.Printf("Working Days: %d\n", report.WorkingDays)
    fmt.Printf("Average Hours/Day: %.2f\n", report.AverageHoursPerDay)
    fmt.Printf("Monthly Trend: %s\n", report.MonthlyTrend)
    fmt.Printf("Goal Progress: %.1f%%\n", report.GoalProgress)

    // Calendar heatmap
    fmt.Println("\nDaily Activity Heatmap:")
    for _, day := range report.CalendarHeatmap {
        productivity := "🟢" // High
        if day.ProductivityLevel == "medium" {
            productivity = "🟡"
        } else if day.ProductivityLevel == "low" {
            productivity = "🔴"
        }
        
        fmt.Printf("  %s %s %.1fh\n", 
            day.Date.Format("Jan 2"), productivity, day.WorkHours)
    }

    // Weekly breakdown
    fmt.Println("\nWeekly Breakdown:")
    for i, weekly := range report.WeeklyBreakdown {
        fmt.Printf("  Week %d: %.2f hours (%s trend)\n", 
            i+1, weekly.TotalWorkHours, weekly.WeeklyTrend)
    }
}
```

### 5. Project Deep-Dive Analysis

```go
func generateProjectReport(service *database.ComprehensiveReportingService) {
    ctx := context.Background()
    userID := "user123"
    projectID := "project_claude_monitor"
    
    // Last 30 days
    period := database.TimePeriod{
        Type:  "custom",
        Start: time.Now().Add(-30 * 24 * time.Hour),
        End:   time.Now(),
    }

    report, err := service.GetProjectReport(ctx, projectID, period, userID)
    if err != nil {
        fmt.Printf("Error generating project report: %v\n", err)
        return
    }

    fmt.Printf("Project Report: %s\n", report.ProjectName)
    fmt.Printf("Path: %s\n", report.ProjectPath)
    fmt.Printf("Period: %s to %s\n", 
        report.Period.Start.Format("Jan 2"), report.Period.End.Format("Jan 2, 2006"))
    fmt.Printf("Total Hours: %.2f\n", report.TotalHours)
    fmt.Printf("Work Blocks: %d\n", report.WorkBlocks)
    fmt.Printf("Active Days: %d\n", report.ActiveDays)
    fmt.Printf("Efficiency: %.1f%%\n", report.Efficiency*100)
    fmt.Printf("Avg Block Size: %v\n", report.AvgBlockSize)
    fmt.Printf("Productivity Trend: %s\n", report.ProductivityTrend)

    // Daily breakdown
    fmt.Println("\nDaily Work Pattern:")
    for _, day := range report.DailyBreakdown {
        fmt.Printf("  %s: %.2f hours (%s)\n", 
            day.Date.Format("Jan 2"), day.WorkHours, day.ProductivityLevel)
    }

    // Hourly pattern
    fmt.Println("\nHourly Work Pattern:")
    for _, hour := range report.HourlyPattern {
        if hour.WorkMinutes > 0 {
            fmt.Printf("  %02d:00: %d minutes\n", hour.Hour, hour.WorkMinutes)
        }
    }
}
```

### 6. Performance Monitoring and Optimization

```go
func monitorSystemPerformance(service *database.ComprehensiveReportingService) {
    // Get comprehensive performance analytics
    perfReport := service.GetPerformanceAnalytics(1 * time.Hour)

    fmt.Printf("System Performance Report\n")
    fmt.Printf("Period: %v\n", perfReport.Period)
    fmt.Printf("Health Score: %.1f%%\n", perfReport.SystemHealthScore)

    // Query performance
    stats := perfReport.PerformanceStats
    fmt.Printf("Total Queries: %d\n", stats.TotalQueries)
    fmt.Printf("Average Response: %v\n", stats.AverageExecutionTime)
    fmt.Printf("95th Percentile: %v\n", stats.P95ExecutionTime)
    fmt.Printf("Cache Hit Rate: %.1f%%\n", stats.CacheHitRate)
    fmt.Printf("Error Rate: %.1f%%\n", stats.ErrorRate)
    fmt.Printf("Slow Queries: %d\n", stats.SlowQueryCount)

    // Cache performance
    cache := perfReport.CacheStats
    fmt.Printf("Cache Entries: %d\n", cache.EntryCount)
    fmt.Printf("Memory Usage: %.1f%%\n", cache.MemoryUsage)
    fmt.Printf("Cache Hits: %d\n", cache.HitCount)
    fmt.Printf("Cache Misses: %d\n", cache.MissCount)

    // Recommendations
    fmt.Println("\nRecommendations:")
    for i, rec := range perfReport.RecommendedActions {
        fmt.Printf("  %d. %s\n", i+1, rec)
    }

    // Optimization opportunities
    fmt.Println("\nIndex Recommendations:")
    for i, idx := range perfReport.OptimizationReport.IndexRecommendations {
        fmt.Printf("  %d. %s: %s\n", i+1, idx.TableName, idx.EstimatedGain)
    }
}
```

### 7. Health Checking and Monitoring

```go
func healthCheck(service *database.ComprehensiveReportingService) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    health, err := service.HealthCheck(ctx)
    if err != nil {
        fmt.Printf("Health check failed: %v\n", err)
        return
    }

    fmt.Printf("Service: %s\n", health.Service)
    fmt.Printf("Status: %s\n", health.Status)
    fmt.Printf("Checked At: %s\n", health.CheckedAt.Format("2006-01-02 15:04:05"))

    fmt.Println("\nHealth Details:")
    for key, value := range health.Details {
        fmt.Printf("  %s: %v\n", key, value)
    }

    // Alert on unhealthy status
    if health.Status != "healthy" {
        fmt.Printf("⚠️  System is %s - investigation required!\n", health.Status)
    } else {
        fmt.Println("✅ System is healthy")
    }
}
```

## Performance Targets

The system is designed to meet these performance requirements:

- **Daily Reports**: < 50ms response time
- **Weekly Reports**: < 100ms response time  
- **Monthly Reports**: < 300ms response time
- **Project Reports**: < 200ms response time
- **Cache Hit Rate**: > 70% for optimal performance
- **Error Rate**: < 1% for production stability
- **Memory Usage**: < 100MB for reasonable resource usage

## Query Optimization Features

### Automatic Index Creation

The system automatically creates optimized indexes:

```sql
-- Session queries optimization
CREATE INDEX idx_session_user_time ON Session(user_id, start_time);

-- Work block project queries
CREATE INDEX idx_workblock_project_time ON WorkBlock(project_id, start_time);

-- Activity event analysis
CREATE INDEX idx_activity_timestamp ON ActivityEvent(timestamp);

-- Project path lookups
CREATE INDEX idx_project_path ON Project(normalized_path);
```

### Query Pattern Optimization

- **Time Range Queries**: Optimized for date-based filtering
- **User-Scoped Queries**: Efficient user-specific data access
- **Project Aggregations**: Fast project-level analytics
- **Relationship Traversals**: Optimized graph relationship queries

### Smart Caching Strategy

- **Current Day**: 5-minute TTL (data changes frequently)
- **Current Week**: 15-minute TTL (moderate updates)
- **Current Month**: 30-minute TTL (less frequent changes)
- **Historical Data**: 4-hour TTL (static data)

## Monitoring and Alerting

### Performance Monitoring

The system continuously monitors:

- Query execution times
- Cache hit rates
- Error rates and patterns
- Memory usage
- Database connection health

### Automatic Alerting

Alerts are generated for:

- Queries taking > 500ms
- Cache hit rate < 70%
- Error rate > 5%
- Memory usage > 80%
- Database connectivity issues

### Health Scoring

System health is scored based on:

- **Query Performance** (40% weight)
- **Cache Efficiency** (30% weight)  
- **Error Rate** (20% weight)
- **Memory Usage** (10% weight)

## Production Configuration

### Recommended Settings

```go
config := database.KuzuConnectionConfig{
    DatabasePath:    "/var/lib/claude-monitor/data/reports.kuzu",
    MaxConnections:  20,  // Adjust based on load
    QueryTimeout:    30 * time.Second,
    BufferPoolSize:  1024, // 1GB for better performance
    EnableLogging:   false, // Disable in production
}
```

### Memory Management

- **Cache Size**: 200MB for high-traffic systems
- **Connection Pool**: 10-20 connections depending on load
- **Buffer Pool**: 1GB+ for large datasets
- **Cleanup Interval**: 5 minutes for cache maintenance

## Troubleshooting

### Common Issues

1. **Slow Query Performance**
   - Check index recommendations
   - Review query patterns
   - Increase buffer pool size

2. **Low Cache Hit Rate**
   - Increase cache size
   - Adjust TTL values
   - Review cache invalidation patterns

3. **High Memory Usage**
   - Reduce cache size
   - Decrease connection pool
   - Implement result pagination

4. **Connection Issues**
   - Check database path
   - Verify file permissions
   - Review connection pool settings

### Debug Commands

```go
// Get performance analytics
perfReport := service.GetPerformanceAnalytics(1 * time.Hour)

// Check system health
health, _ := service.HealthCheck(context.Background())

// Review slow queries
optimizer := service.GetQueryOptimizer()
slowQueries := optimizer.GetSlowQueries(30*time.Minute, 10)

// Analyze execution plans
analysis, _ := optimizer.AnalyzeExecutionPlan(ctx, query, params)
```

## Integration Examples

### CLI Integration

```go
func dailyCommand(service *database.ComprehensiveReportingService) {
    report, err := service.GetDailyReport(context.Background(), time.Now(), getCurrentUser())
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    // Format for beautiful CLI output
    formatDailyReport(report)
}
```

### Web API Integration

```go
func dailyReportHandler(service *database.ComprehensiveReportingService) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userID := getUserFromContext(r.Context())
        date := getDateFromQuery(r)
        
        report, err := service.GetDailyReport(r.Context(), date, userID)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(report)
    }
}
```

### Scheduled Reports

```go
func scheduleReports(service *database.ComprehensiveReportingService) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        // Generate and cache reports for active users
        users := getActiveUsers()
        for _, userID := range users {
            // Pre-generate today's report
            service.GetDailyReport(context.Background(), time.Now(), userID)
            
            // Pre-generate current week report
            service.GetWeeklyReport(context.Background(), getCurrentWeekStart(), userID)
        }
    }
}
```

This comprehensive reporting system provides enterprise-grade work hour analytics with optimal performance, automatic optimization, and comprehensive monitoring. It's production-ready and designed to scale with your Claude Monitor usage.