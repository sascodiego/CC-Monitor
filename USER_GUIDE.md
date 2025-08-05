# Claude Monitor - User Guide

[![Alpha Version](https://img.shields.io/badge/Status-Alpha-orange?style=flat-square)](https://github.com/sascodiego/CC-Monitor)

## 📖 Complete User Guide

This guide covers everything you need to know to use Claude Monitor effectively for work hour tracking and productivity analysis.

## 🎯 Overview

Claude Monitor automatically tracks your Claude CLI usage and converts it into actionable work hour data. It provides:

- **Automatic Time Tracking**: No manual timers needed
- **Professional Reports**: Generate timesheets and analytics
- **Productivity Insights**: Understand your work patterns
- **Multiple Export Formats**: JSON, CSV, HTML, PDF, Excel

## 🚀 Basic Usage

### Starting the System

```bash
# Start the enhanced daemon (in terminal 1)
sudo ./bin/claude-daemon-enhanced

# Use Claude Monitor CLI (in terminal 2)
./bin/claude-monitor-basic status
```

### Core Commands

#### System Status
```bash
# Check if system is running
./bin/claude-monitor-basic status

# Output:
# 🚀 Claude Monitor System Status
# ═══════════════════════════════════
# ✅ Enhanced Daemon: RUNNING
# 📊 Current Activity: 8 processes, 3 connections
# 🔥 API Activity: DETECTED
# 📈 Current Session: a1b2c3d4 (running 2h 30m)
# ⏰ Session Window: 09:00 → 14:00
# 🔥 Current Work Block: e5f6g7h8 (active 45m)
# ⏱️  Last Activity: 2m ago
```

#### Work Day Status
```bash
# Today's work summary
./bin/claude-monitor-basic workhour workday status

# Output:
# 📊 Work Day Status - January 15, 2024
# ═══════════════════════════════════════════════
# ⏰ Work Period: 09:15 AM → 05:30 PM (8h 15m)
# 📈 Productivity: 87% efficiency (7h 12m active)
# 🎯 Goal Progress: 103% of 8h target ✅
# 📋 Sessions: 3 sessions, 12 work blocks
# 🔥 Peak Hours: 10:00-12:00 AM, 02:00-04:00 PM
```

## 📊 Understanding the System

### How It Works

1. **Session Detection**: 5-hour windows starting from first Claude activity
2. **Work Block Tracking**: Continuous work periods with 5-minute inactivity timeout
3. **Process Monitoring**: Detects Claude CLI processes automatically
4. **Network Analysis**: Monitors API connections for activity patterns

### Key Concepts

#### Sessions (5-Hour Windows)
- **Start**: Triggered by first Claude activity
- **Duration**: Exactly 5 hours from start time
- **Multiple Sessions**: Can have multiple sessions per day
- **Overlap**: Sessions can overlap if you work >5 hours continuously

#### Work Blocks (Activity Periods)
- **Start**: First activity within a session
- **End**: After 5 minutes of inactivity
- **Multiple Blocks**: Can have multiple work blocks per session
- **Automatic**: No manual start/stop required

### Status Indicators

| Indicator | Meaning |
|-----------|---------|
| ✅ RUNNING | System is active and monitoring |
| 🔥 DETECTED | Real user activity detected |
| 💤 IDLE | No activity, but monitoring |
| ⚠️ APPROACHING | Inactivity timeout approaching |
| ❌ STOPPED | System not running |

## ⏰ Work Hour Tracking

### Daily Tracking

```bash
# Current work day status
./bin/claude-monitor-basic workhour workday status

# Key metrics shown:
# - Work start/end times
# - Total time worked
# - Productivity percentage
# - Goal progress (vs 8h target)
# - Session and work block info
# - Peak productivity hours
```

### Understanding Metrics

#### Productivity Calculation
- **Active Time**: Time with Claude activity
- **Total Time**: Time since work started
- **Efficiency**: (Active Time / Total Time) × 100

#### Goal Progress
- **Default Target**: 8 hours per day
- **Progress**: (Hours Worked / Target) × 100
- **Status**: ✅ Complete | 🕐 In Progress | ⚠️ Behind

## 🎯 Advanced Features

### Time Patterns

The system recognizes different work patterns:

#### Work Schedule Types
- **Early Bird**: Start before 8 AM
- **Standard**: 9 AM - 5 PM schedule
- **Night Owl**: Heavy activity after 6 PM
- **Flexible**: Irregular but consistent patterns

#### Peak Hours Detection
- **Statistical Analysis**: Identifies most productive times
- **Pattern Recognition**: Learns your optimal work periods
- **Recommendations**: Suggests scheduling important work during peaks

### Activity Classification

#### Real vs Background Activity
- **POST Requests**: Real user interactions (prompts, conversations)
- **GET Requests**: Background operations (health checks, metrics)
- **Connection Patterns**: Distinguishes active vs idle connections

#### Smart Timeout Handling
- **5-Minute Rule**: Work blocks end after 5 minutes of inactivity
- **Session Continuity**: Sessions continue even during breaks
- **Automatic Resume**: New activity automatically starts new work blocks

## 📈 Reports and Analytics

### Available Reports

#### Daily Reports
```bash
# Today's detailed analysis
./bin/claude-monitor-basic workhour workday report

# Custom date
./bin/claude-monitor-basic workhour workday report --date=2024-01-15
```

#### Weekly Analysis
```bash
# Current week summary
./bin/claude-monitor-basic workhour workweek report

# Specific week
./bin/claude-monitor-basic workhour workweek report --week=2024-W03
```

### Export Options

#### JSON Export (API Integration)
```bash
# Machine-readable format
./bin/claude-monitor-basic export --format=json --type=daily

# Output structure:
{
  "date": "2024-01-15",
  "totalHours": 8.25,
  "sessions": [...],
  "workBlocks": [...],
  "productivity": {
    "efficiency": 87.5,
    "peakHours": ["10:00-12:00", "14:00-16:00"]
  }
}
```

#### CSV Export (Spreadsheets)
```bash
# Spreadsheet-compatible format
./bin/claude-monitor-basic export --format=csv --type=weekly

# Columns: Date, Start, End, Duration, Sessions, Blocks, Efficiency
```

#### HTML Reports (Web/Email)
```bash
# Interactive web report
./bin/claude-monitor-basic export --format=html --type=monthly --charts

# Features:
# - Interactive charts
# - Responsive design
# - Professional styling
# - Email-friendly format
```

## ⚙️ Configuration

### Work Hour Goals

```bash
# Set daily goal
./bin/claude-monitor-basic goals set --daily=8h

# Set weekly goal
./bin/claude-monitor-basic goals set --weekly=40h

# View current goals
./bin/claude-monitor-basic goals view
```

### Business Policies

```bash
# Overtime threshold
./bin/claude-monitor-basic policy update --overtime-threshold=8h

# Time rounding
./bin/claude-monitor-basic policy update --rounding=15min --method=nearest

# Break deduction
./bin/claude-monitor-basic policy update --break-deduction=30m
```

### Notification Settings

```bash
# Goal reminders
./bin/claude-monitor-basic notifications set --goal-reminder=true

# Overtime alerts
./bin/claude-monitor-basic notifications set --overtime-alert=true

# Daily summary
./bin/claude-monitor-basic notifications set --daily-summary=6pm
```

## 📊 Dashboard and Monitoring

### Real-Time Monitoring

```bash
# Live status updates
./bin/claude-monitor-basic status --live

# Refreshes every 5 seconds with current:
# - Active processes
# - Network connections
# - Current session/work block
# - Time since last activity
```

### Historical Analysis

```bash
# View trends
./bin/claude-monitor-basic analytics trends --period=30days

# Pattern analysis
./bin/claude-monitor-basic analytics patterns

# Productivity insights
./bin/claude-monitor-basic analytics productivity --recommendations
```

## 🔧 Troubleshooting

### Common Issues

#### "No active work session detected"
**Solution**: Start using Claude CLI. The system only tracks when Claude is active.

#### "Enhanced Daemon: STOPPED"  
**Solution**: 
```bash
sudo ./bin/claude-daemon-enhanced
```

#### Inaccurate time tracking
**Causes**:
- Multiple Claude instances
- Network connectivity issues
- System clock changes

**Solution**:
```bash
# Restart daemon
sudo pkill -f claude-daemon
sudo ./bin/claude-daemon-enhanced
```

#### Status file permissions
**Solution**:
```bash
sudo chmod 644 /tmp/claude-monitor-status.json
```

### Debug Mode

```bash
# Verbose output
./bin/claude-monitor-basic status --verbose

# Debug daemon (run in foreground)
sudo ./bin/claude-daemon-enhanced --debug
```

### Log Analysis

```bash
# Check recent activity
tail -f /var/log/claude-monitor/daemon.log

# Search for errors
grep -i error /var/log/claude-monitor/daemon.log

# Monitor process detection
grep -i "process" /var/log/claude-monitor/daemon.log
```

## 🎯 Best Practices

### Daily Workflow

1. **Morning Setup**
   ```bash
   # Check if daemon is running
   ./bin/claude-monitor-basic status
   
   # Review yesterday's work
   ./bin/claude-monitor-basic workhour workday report --date=yesterday
   ```

2. **During Work**
   - Use Claude CLI normally
   - System tracks automatically
   - Check progress occasionally:
   ```bash
   ./bin/claude-monitor-basic workhour workday status
   ```

3. **End of Day**
   ```bash
   # Generate daily report
   ./bin/claude-monitor-basic workhour workday report
   
   # Export for timesheet
   ./bin/claude-monitor-basic export --format=csv --type=daily
   ```

### Weekly Review

```bash
# Weekly analysis
./bin/claude-monitor-basic workhour workweek analysis

# Export for billing
./bin/claude-monitor-basic export --format=pdf --type=weekly

# Check patterns
./bin/claude-monitor-basic analytics patterns
```

### Goal Management

1. **Set Realistic Goals**
   - Start with 6-7 hours daily
   - Adjust based on actual patterns
   - Consider break time

2. **Monitor Progress**
   - Check daily progress
   - Weekly goal review
   - Adjust as needed

3. **Use Insights**
   - Schedule important work during peak hours
   - Plan breaks during low-productivity periods
   - Optimize work schedule based on patterns

## 🚀 Advanced Usage

### Automation Scripts

#### Daily Report Email
```bash
#!/bin/bash
REPORT=$(./bin/claude-monitor-basic workhour workday report)
echo "$REPORT" | mail -s "Daily Work Report - $(date +%Y-%m-%d)" user@company.com
```

#### Weekly Timesheet Generation
```bash
#!/bin/bash
./bin/claude-monitor-basic export --format=pdf --type=weekly --output="timesheet-$(date +%Y-W%U).pdf"
```

#### Goal Progress Notification
```bash
#!/bin/bash
PROGRESS=$(./bin/claude-monitor-basic workhour workday status | grep "Goal Progress")
if [[ $PROGRESS == *"<50%"* ]]; then
    notify-send "Work Goal" "Behind target today: $PROGRESS"
fi
```

### Integration Examples

#### Slack Integration
```bash
# Send daily summary to Slack
curl -X POST -H 'Content-type: application/json' \
  --data "{\"text\":\"$(./bin/claude-monitor-basic workhour workday status)\"}" \
  YOUR_SLACK_WEBHOOK_URL
```

#### Dashboard API
```bash
# Get JSON data for dashboard
./bin/claude-monitor-basic export --format=json --type=daily > daily-stats.json
```

## 📈 Productivity Optimization

### Using Analytics

1. **Identify Peak Hours**
   ```bash
   ./bin/claude-monitor-basic analytics patterns
   ```
   - Schedule complex work during peaks
   - Use low periods for admin tasks

2. **Track Improvements**
   ```bash
   ./bin/claude-monitor-basic analytics trends --period=30days
   ```
   - Monitor efficiency trends
   - Measure goal achievement
   - Adjust strategies

3. **Optimize Schedule**
   - Use pattern data to plan optimal work times
   - Align meetings with low-productivity periods
   - Plan deep work during peak hours

### Work-Life Balance

1. **Set Boundaries**
   - Configure overtime alerts
   - Set reasonable daily goals
   - Monitor total work hours

2. **Track Breaks**
   - System automatically detects break periods
   - Review break patterns for wellness
   - Ensure adequate rest time

3. **Weekly Planning**
   - Use weekly reports for planning
   - Adjust goals based on workload
   - Balance high and low activity days

---

**🎉 You're now ready to use Claude Monitor effectively for professional work hour tracking and productivity optimization!**

For technical issues, see `INSTALLATION.md` and `TROUBLESHOOTING.md`.
For advanced features, check the `docs/` directory.