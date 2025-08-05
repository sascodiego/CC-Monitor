# Claude Monitor Work Hour CLI Guide

This guide provides comprehensive documentation for using Claude Monitor's work hour tracking and reporting CLI commands.

## Overview

The work hour CLI provides powerful capabilities for:
- **Daily work tracking** with real-time status and detailed reporting
- **Weekly productivity analysis** with pattern recognition and insights
- **Professional timesheet generation** for billing and HR compliance
- **Advanced analytics** for productivity optimization
- **Goal management** and policy configuration
- **Bulk data export** for external analysis

## Command Structure

All work hour commands are under the `workhour` namespace:

```bash
claude-monitor workhour <category> <command> [options]
```

## Daily Work Tracking

### Current Work Day Status

```bash
# Basic status for today
claude-monitor workhour workday status

# Status for specific date
claude-monitor workhour workday status 2024-01-15

# Detailed status with breakdowns
claude-monitor workhour workday status --detailed

# Live status updates (refreshes every 30 seconds)
claude-monitor workhour workday status --live

# Include break analysis
claude-monitor workhour workday status --breaks

# Show work patterns
claude-monitor workhour workday status --pattern

# JSON output for integration
claude-monitor workhour workday status --format=json
```

**Example Output:**
```
📅 Work Day Status - Monday, January 15, 2024
═══════════════════════════════════════════════════════════

📊 Daily Summary
────────────────
Total Work Time:    7h 45m
Sessions:           2
Work Blocks:        4
Avg Block Time:     1h 56m
Productivity Score: 82.5%
Status:             ✓ Full Day

📋 Detailed Breakdown
─────────────────────
Work Blocks:
  1. 09:15 - 11:30 (2h 15m) [Completed]
  2. 13:00 - 15:45 (2h 45m) [Completed]
  3. 16:30 - 18:00 (1h 30m) [Completed]
  4. 19:00 - now (1h 15m) [Active]

Activity Period:    09:15 to now
Work Efficiency:    78.2% (active/total time)
```

### Daily Reports

```bash
# Generate today's report
claude-monitor workhour workday report

# Report for specific date
claude-monitor workhour workday report 2024-01-15

# Include visual charts
claude-monitor workhour workday report --charts

# Include goal progress
claude-monitor workhour workday report --goals

# Include trend comparison
claude-monitor workhour workday report --trends

# Compare with other days
claude-monitor workhour workday report --compare=2024-01-14,2024-01-13

# Save to file
claude-monitor workhour workday report --output=daily-report.pdf
```

### Data Export

```bash
# Export single day to CSV
claude-monitor workhour workday export --output=workday.csv

# Export date range
claude-monitor workhour workday export --start=2024-01-01 --end=2024-01-31 --output=january.csv

# Include raw session data
claude-monitor workhour workday export --raw --output=detailed.json

# Aggregate multiple days
claude-monitor workhour workday export --aggregate --start=2024-01-01 --end=2024-01-07 --output=week1.xlsx
```

## Weekly Analysis

### Weekly Reports

```bash
# Current week report
claude-monitor workhour workweek report

# Specific week (Monday date)
claude-monitor workhour workweek report 2024-01-15

# Include overtime analysis
claude-monitor workhour workweek report --overtime

# Include work pattern analysis
claude-monitor workhour workweek report --pattern

# Include goal tracking
claude-monitor workhour workweek report --goals

# Daily breakdown
claude-monitor workhour workweek report --daily

# Compare with previous weeks
claude-monitor workhour workweek report --compare=2024-01-08,2024-01-01

# Custom standard hours
claude-monitor workhour workweek report --standard-hours=37.5h
```

**Example Output:**
```
📊 Weekly Work Hour Report
Period: January 15 - January 21, 2024
══════════════════════════════════════════════════════════

📅 Weekly Summary
────────────────
Total Work Time:    38h 30m
Standard Hours:     40h (-1h 30m)
Work Days:          5 of 5
Avg Daily Hours:    7h 42m

📊 Daily Breakdown
─────────────────
Monday     | 7h 45m       | 2        | 4
Tuesday    | 8h 15m       | 1        | 3
Wednesday  | 6h 30m       | 3        | 5
Thursday   | 8h 30m       | 2        | 4
Friday     | 7h 30m       | 2        | 3
```

### Advanced Weekly Analysis

```bash
# Basic weekly analysis
claude-monitor workhour workweek analysis

# Comprehensive analysis with all metrics
claude-monitor workhour workweek analysis --depth=comprehensive

# Include productivity metrics
claude-monitor workhour workweek analysis --productivity

# Include efficiency analysis
claude-monitor workhour workweek analysis --efficiency

# Include recommendations
claude-monitor workhour workweek analysis --recommendations

# Compare to historical average
claude-monitor workhour workweek analysis --compare-average

# Save detailed analysis
claude-monitor workhour workweek analysis --output=weekly-analysis.pdf
```

## Timesheet Management

### Generate Timesheets

```bash
# Generate weekly timesheet
claude-monitor workhour timesheet generate --period=weekly

# Generate biweekly timesheet
claude-monitor workhour timesheet generate --period=biweekly

# Generate monthly timesheet
claude-monitor workhour timesheet generate --period=monthly

# Custom period start date
claude-monitor workhour timesheet generate --period=weekly --start=2024-01-15

# Custom rounding rules
claude-monitor workhour timesheet generate --rounding=15min --rounding-method=up

# Custom overtime threshold
claude-monitor workhour timesheet generate --overtime=8h

# Auto-submit after generation
claude-monitor workhour timesheet generate --auto-submit

# Save to file
claude-monitor workhour timesheet generate --output=timesheet.pdf
```

### View Timesheets

```bash
# List recent timesheets
claude-monitor workhour timesheet view

# View specific timesheet
claude-monitor workhour timesheet view TS-2024-001

# Filter by status
claude-monitor workhour timesheet view --status=draft

# Show detailed entries
claude-monitor workhour timesheet view --details

# Show summary totals
claude-monitor workhour timesheet view --totals

# Filter by date range
claude-monitor workhour timesheet view --start=2024-01-01 --end=2024-01-31
```

### Submit Timesheets

```bash
# Submit timesheet for approval
claude-monitor workhour timesheet submit TS-2024-001

# Force submit without validation
claude-monitor workhour timesheet submit TS-2024-001 --force

# Add submission comments
claude-monitor workhour timesheet submit TS-2024-001 --comments="Overtime pre-approved"

# Send notification
claude-monitor workhour timesheet submit TS-2024-001 --notify
```

### Export Timesheets

```bash
# Export specific timesheet to PDF
claude-monitor workhour timesheet export --timesheet=TS-2024-001 --output=timesheet.pdf

# Export all timesheets for employee
claude-monitor workhour timesheet export --employee=EMP001 --format=excel --output=employee_timesheets.xlsx

# Export date range
claude-monitor workhour timesheet export --start=2024-01-01 --end=2024-01-31 --output=january_timesheets.pdf

# Include summary page
claude-monitor workhour timesheet export --summary --output=summary_timesheet.pdf

# Add digital signature
claude-monitor workhour timesheet export --signature --output=signed_timesheet.pdf
```

## Analytics and Insights

### Productivity Analysis

```bash
# Default productivity analysis (last 30 days)
claude-monitor workhour analytics productivity

# Custom date range
claude-monitor workhour analytics productivity --start=2024-01-01 --end=2024-01-31

# Hourly granularity
claude-monitor workhour analytics productivity --granularity=hour

# Include pattern analysis
claude-monitor workhour analytics productivity --patterns

# Include trend analysis
claude-monitor workhour analytics productivity --trends

# Include recommendations
claude-monitor workhour analytics productivity --recommendations

# Compare to baseline
claude-monitor workhour analytics productivity --baseline

# Specific metrics only
claude-monitor workhour analytics productivity --metrics=focus,efficiency,active-ratio

# Include visual charts
claude-monitor workhour analytics productivity --charts

# Save analysis
claude-monitor workhour analytics productivity --output=productivity-analysis.pdf
```

**Example Output:**
```
📊 Productivity Analysis
Period: January 1 - January 31, 2024
══════════════════════════════════════════════════════════

⚡ Efficiency Metrics
───────────────────
Active Time Ratio:  78.5%
Focus Score:        82.1%
Productivity Score: 85.3%
Session Efficiency: 74.2%
Break Frequency:    2.1 breaks/hour

💡 Recommendations
─────────────────
1. Consider reducing distractions to improve focus score
2. Try time-blocking techniques for better session management
3. Schedule important tasks during peak hours (10:00-12:00)
```

### Work Pattern Analysis

```bash
# Basic pattern analysis
claude-monitor workhour analytics patterns

# Include break patterns
claude-monitor workhour analytics patterns --breaks

# Identify peak productivity hours
claude-monitor workhour analytics patterns --peak-hours

# Include optimization recommendations
claude-monitor workhour analytics patterns --recommendations

# Compare to ideal work patterns
claude-monitor workhour analytics patterns --compare-ideal

# Minimum data points for pattern recognition
claude-monitor workhour analytics patterns --min-data=14

# Custom visualization
claude-monitor workhour analytics patterns --visualization=heatmap
```

### Trend Analysis

```bash
# Default trend analysis (last 90 days)
claude-monitor workhour analytics trends

# Weekly trend periods
claude-monitor workhour analytics trends --period=weekly

# Include forecasting
claude-monitor workhour analytics trends --forecasting

# Analyze seasonal patterns
claude-monitor workhour analytics trends --seasonality

# Custom confidence level
claude-monitor workhour analytics trends --confidence=0.95

# Specific metrics
claude-monitor workhour analytics trends --metrics=work-time,productivity,focus

# Include trend charts
claude-monitor workhour analytics trends --charts
```

## Goals and Policies

### Goal Management

```bash
# View current goals
claude-monitor workhour goals view

# View goals with progress
claude-monitor workhour goals view --progress

# View goal history
claude-monitor workhour goals view --history

# Set daily goal
claude-monitor workhour goals set --type=daily --target=8h

# Set weekly goal
claude-monitor workhour goals set --type=weekly --target=40h

# Set goal with auto-reset
claude-monitor workhour goals set --type=daily --target=8h --auto-reset

# Enable goal notifications
claude-monitor workhour goals set --type=weekly --target=40h --notifications

# Goal with description
claude-monitor workhour goals set --type=daily --target=8h --description="Standard workday target"
```

### Policy Management

```bash
# View current policies
claude-monitor workhour policy view

# View specific policy type
claude-monitor workhour policy view --type=overtime

# Include default values
claude-monitor workhour policy view --defaults

# Update rounding interval
claude-monitor workhour policy update --rounding-interval=15min

# Update rounding method
claude-monitor workhour policy update --rounding-method=nearest

# Update overtime threshold
claude-monitor workhour policy update --overtime-threshold=8h

# Update break deduction
claude-monitor workhour policy update --break-deduction=30min

# Policy with effective date
claude-monitor workhour policy update --rounding-interval=30min --effective=2024-02-01

# Add reason for change
claude-monitor workhour policy update --overtime-threshold=7.5h --reason="New company policy"
```

## Bulk Operations

### Bulk Export

```bash
# Export full year of data
claude-monitor workhour bulk export --start=2024-01-01 --end=2024-12-31 --dir=/exports

# Parallel processing
claude-monitor workhour bulk export --parallel --concurrency=4 --dir=/exports

# Split by month
claude-monitor workhour bulk export --split=monthly --compression=zip --dir=/exports

# Multiple formats
claude-monitor workhour bulk export --formats=csv,json,excel --dir=/exports

# Specific data types
claude-monitor workhour bulk export --types=workdays,timesheets,analytics --dir=/exports

# Include metadata
claude-monitor workhour bulk export --metadata --dir=/exports

# Filter by employees
claude-monitor workhour bulk export --employees=EMP001,EMP002 --dir=/exports

# Resume interrupted export
claude-monitor workhour bulk export --resume=/tmp/export-resume.json --dir=/exports

# Show progress
claude-monitor workhour bulk export --progress --dir=/exports
```

## Output Formats

All commands support multiple output formats:

- `--format=table` (default) - Human-readable table format
- `--format=json` - JSON format for integration
- `--format=csv` - CSV format for spreadsheets
- `--format=summary` - Condensed summary format

## Global Options

These options work with all work hour commands:

- `--verbose, -v` - Verbose output with detailed information
- `--help, -h` - Show command help
- `--config` - Specify config file path
- `--log-level` - Set logging level (DEBUG, INFO, WARN, ERROR)

## Integration Examples

### Scripting Examples

```bash
#!/bin/bash
# Daily productivity check
PRODUCTIVITY=$(claude-monitor workhour analytics productivity --format=json | jq -r '.metrics.productivityScore')
if (( $(echo "$PRODUCTIVITY < 70" | bc -l) )); then
    echo "Low productivity detected: $PRODUCTIVITY%"
    # Send notification or take action
fi

# Weekly timesheet automation
claude-monitor workhour timesheet generate --period=weekly --auto-submit
```

### API Integration

```bash
# Export data for external analysis
claude-monitor workhour workday export --format=json --start=2024-01-01 --end=2024-01-31 --output=data.json

# Send to analytics service
curl -X POST https://analytics.company.com/workhour-data \
  -H "Content-Type: application/json" \
  -d @data.json
```

## Tips and Best Practices

### Daily Usage
- Use `--live` mode for real-time monitoring during work
- Set daily goals to track progress
- Review daily reports at end of day

### Weekly Review
- Generate weekly reports every Friday
- Use pattern analysis to optimize schedule
- Compare weeks to identify trends

### Monthly Process
- Generate timesheets for billing/HR
- Analyze long-term trends
- Update goals and policies as needed

### Data Management
- Regular bulk exports for backup
- Use compression for large datasets
- Monitor storage usage for long-term tracking

## Troubleshooting

### Common Issues
- **"Work hour functionality not available"**: Work hour service not initialized
- **Date format errors**: Use YYYY-MM-DD format
- **Permission issues**: Some operations may require specific permissions

### Getting Help
- Use `--help` with any command for detailed usage
- Check logs with `claude-monitor logs` for detailed error information
- Use `--verbose` for detailed operation information

## Future Enhancements

Planned features include:
- Interactive dashboard mode
- Advanced visualization with charts
- Integration with calendar systems
- Team collaboration features
- Custom report templates
- Machine learning insights

---

For more information, see the main Claude Monitor documentation or use `claude-monitor workhour --help` for command-specific help.