# Enhanced Activity Detection System

## Overview

The enhanced activity detection system addresses false positive activity detection caused by automatic keepalive/ping connections from the Claude CLI. This implementation distinguishes between real user interactions and background network activity.

## Key Improvements

### 1. Traffic Pattern Analysis

The system now analyzes four distinct traffic patterns:

- **TrafficInteractive**: Large data transfers with irregular timing (real user activity)
- **TrafficBurst**: Sudden increases in data transfer (likely user interaction)
- **TrafficKeepalive**: Small, regular data transfers (automatic background activity)
- **TrafficIdle**: No significant network activity

### 2. Data Volume Monitoring

Enhanced connection monitoring tracks:
- **Data Transfer Volume**: Actual bytes sent/received per connection
- **Connection Duration**: How long connections have been active
- **Baseline Connection Count**: Normal number of keepalive connections
- **Connection Details**: Local/remote addresses, state, and transfer metrics

### 3. Multi-Factor Activity Detection

The new `isRealActivityEnhanced()` function considers:

1. **Traffic Pattern**: Primary indicator of activity type
2. **Data Volume**: Minimum threshold for real activity (1KB default)
3. **Process Changes**: New Claude processes starting/stopping
4. **Connection Patterns**: Significant changes from baseline
5. **Sustained Activity**: Consistent activity over time

### 4. Configurable Thresholds

Runtime adjustable parameters:
- `minActivityThreshold`: 1024 bytes (minimum for real activity)
- `keepaliveThreshold`: 512 bytes (maximum for keepalive detection)
- `burstThreshold`: 8192 bytes (minimum for burst detection)

## Technical Implementation

### Enhanced Data Structures

```go
type ActivityIndicator struct {
    // Basic metrics
    Timestamp          time.Time
    ProcessCount       int
    NetworkConnections int
    APIActivity        bool
    
    // Enhanced metrics
    ConnectionDetails  []ConnectionInfo
    DataTransferBytes  int64
    TrafficPattern     TrafficPatternType
}

type ConnectionInfo struct {
    LocalAddr    string
    RemoteAddr   string
    State        string
    Duration     time.Duration
    RxBytes      int64
    TxBytes      int64
    LastSeen     time.Time
}
```

### Connection Monitoring

- Uses `ss` command for detailed network statistics (fallback to `netstat`)
- Tracks individual connection data transfer volumes
- Maintains connection baseline for comparison
- Automatically cleans up stale connection data

### Pattern Recognition Algorithm

1. **Baseline Establishment**: Automatically detects normal keepalive connection count
2. **Volume Analysis**: Categorizes activity based on data transfer volume
3. **Variance Checking**: Identifies stable keepalive patterns vs. interactive changes
4. **Multi-Signal Validation**: Combines multiple indicators for accuracy

## Benefits

### False Positive Reduction

- **Before**: Any network connection change triggered activity detection
- **After**: Only significant data transfers or pattern changes indicate real activity

### Accurate Work Time Tracking

- Prevents keepalive connections from extending work blocks inappropriately
- Ensures 5-minute inactivity timeout works correctly
- Maintains accurate session and work block boundaries

### Performance Optimization

- Connection tracking with automatic cleanup prevents memory leaks
- Efficient pattern analysis with configurable thresholds
- Background cleanup routines maintain system health

## Configuration

### Default Thresholds

```go
minActivityThreshold: 1024    // 1KB minimum for real activity
keepaliveThreshold:   512     // 512 bytes maximum for keepalive
burstThreshold:       8192    // 8KB minimum for burst detection
```

### Runtime Adjustment

```go
daemon.SetActivityThresholds(
    minActivity: 2048,  // Increase sensitivity
    keepalive:   256,   // Tighter keepalive detection
    burst:       4096,  // Lower burst threshold
)
```

## Monitoring

### Enhanced Status Output

The status file now includes:

```json
{
  "activityHistory": {
    "dataTransferBytes": 0,
    "trafficPattern": "keepalive",
    "baselineConnections": 8,
    "activeConnections": 8,
    "minActivityThreshold": 1024
  }
}
```

### Activity Metrics

```go
metrics := daemon.GetActivityMetrics()
// Returns: baselineConnections, trackedConnections, thresholds, etc.
```

## Testing and Validation

### Manual Testing

1. **Normal Usage**: Verify real Claude interactions trigger activity correctly
2. **Idle Testing**: Ensure keepalive connections don't prevent timeouts
3. **Threshold Testing**: Validate configurable parameters work as expected
4. **Memory Testing**: Confirm connection cleanup prevents memory leaks

### Monitoring Points

- Track false positive rates in logs
- Monitor connection tracking memory usage
- Validate threshold effectiveness over time
- Ensure activity detection accuracy

## Future Enhancements

1. **Machine Learning**: Pattern recognition could be enhanced with ML models
2. **Network Fingerprinting**: More sophisticated API call detection
3. **User Feedback**: Allow users to mark false positives/negatives
4. **Adaptive Thresholds**: Automatically adjust based on usage patterns

This enhanced system provides significantly more accurate activity detection while maintaining the simplicity and performance requirements of the Claude Monitor system.