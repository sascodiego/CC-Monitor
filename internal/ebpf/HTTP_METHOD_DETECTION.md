# HTTP Method Detection Implementation

**AGENT:** ebpf-specialist  
**TRACE:** CLAUDE-EBPF-030  
**CONTEXT:** Documentation for HTTP method detection in Claude Monitor  
**REASON:** Need comprehensive documentation of HTTP parsing implementation  
**CHANGE:** Initial implementation.  
**PREVENTION:** Keep documentation synchronized with code changes  
**RISK:** Low - Documentation helps maintain and debug the system  

## Overview

This implementation extends the Claude Monitor eBPF system to detect HTTP request methods and classify them as either user interactions or background operations. This enables accurate detection of real Claude usage versus automated health checks and status polling.

## Architecture

### eBPF Components

#### 1. Socket Connection Tracking (`socket_connections` map)
- **Purpose**: Correlate `connect()` syscalls with subsequent `write()` syscalls
- **Key**: Socket file descriptor (uint32)
- **Value**: Connection metadata (PID, target IP, port, timestamp)
- **Lifecycle**: Created on connect, used for write correlation, cleaned up on process exit

#### 2. HTTP Header Parsing (`trace_write` tracepoint)
- **Monitors**: `sys_enter_write` syscalls
- **Filters**: Only processes sockets connected to Anthropic API endpoints
- **Parses**: HTTP method, URI, and Content-Length header
- **Security**: Limited to first 512 bytes, only headers (never request/response bodies)

#### 3. Enhanced Event Structure
```c
struct claude_event {
    // Existing fields...
    char http_method[8];        // "GET", "POST", etc.
    char http_uri[128];         // "/v1/messages", "/health", etc.
    uint32 content_length;      // Content-Length header value
    uint32 socket_fd;           // For correlation with connections
};
```

### Go Integration

#### Event Classification Methods
- `IsUserInteraction()`: Identifies real user interactions
- `IsBackgroundOperation()`: Identifies automated operations
- `GetHTTPMethod()`: Extracts HTTP method
- `GetHTTPURI()`: Extracts request URI
- `GetContentLength()`: Extracts content length

## HTTP Request Classification

### User Interactions (Real Usage)
- **POST** to `/v1/messages` - Main conversation endpoint
- **POST** to `/v1/complete` - Text completion endpoint  
- **PUT/PATCH** requests - User-initiated data updates
- **GET** to `/v1/messages` or `/v1/conversation` - Interactive retrievals

### Background Operations (Automated)
- **GET** to `/health`, `/status`, `/v1/status` - Health checks
- **OPTIONS** requests - CORS preflight checks
- **HEAD** requests - Connection validation
- **All other GET** requests not to interactive endpoints

## Implementation Details

### eBPF Security Constraints

#### Memory Safety
- All string operations use bounded loops with `#pragma unroll`
- Mandatory bounds checking before memory access
- Safe user-space data reading with `bpf_probe_read_user()`

#### Performance Optimizations
- Fast PID filtering using tracking maps
- Socket connection pre-filtering
- Header size limits (512 bytes maximum)
- Kernel-level HTTP format validation

### HTTP Parsing Algorithm

#### Method Extraction
1. Scan first 8 bytes for space character
2. Extract method string before first space
3. Validate minimum length (3 characters: "GET", "PUT", etc.)

#### URI Extraction  
1. Skip method and space
2. Scan for next space (before "HTTP/1.1")
3. Extract URI string between spaces
4. Limit to 128 characters maximum

#### Content-Length Parsing
1. Search for "Content-Length: " header
2. Parse decimal digits following header
3. Limit to 10 digits maximum (prevents overflow)

## Integration with Daemon

### Event Processing Pipeline
1. **eBPF**: Captures HTTP requests from Claude processes to Anthropic API
2. **Manager**: Parses eBPF events and enriches metadata  
3. **Validation**: Filters events using business logic
4. **Activity Detection**: Distinguishes user interactions from background operations

### Business Logic Integration
The daemon can now accurately detect user activity by:
- Counting only POST requests to interaction endpoints
- Ignoring GET requests to health/status endpoints
- Tracking actual user engagement vs. automated polling

## Performance Characteristics

### Overhead
- **Minimal kernel impact**: Only processes tracked Claude PIDs
- **Efficient filtering**: Pre-filters by socket connection
- **Bounded parsing**: Limited data inspection (512 bytes)
- **Fast path optimization**: Early exits for irrelevant events

### Scalability
- Supports 2048 concurrent socket connections
- Per-CPU performance counters for monitoring
- Ring buffer for high-throughput event delivery

## Security Considerations

### Data Privacy
- **Headers only**: Never inspects request/response bodies
- **Method and URI**: Only extracts routing information
- **No user data**: Content-Length header value only (not content)
- **Limited scope**: Only Anthropic API connections

### Kernel Safety
- **Verifier approved**: All memory access bounds-checked
- **No unbounded loops**: Fixed iteration limits with unroll pragma
- **Error handling**: Graceful degradation on parsing failures
- **Resource cleanup**: Automatic map cleanup on process exit

## Testing

### Unit Tests
- HTTP method classification logic
- Event metadata extraction
- Background vs. user interaction detection
- Edge cases and malformed HTTP

### Integration Tests
- eBPF program loading and attachment
- Ring buffer event processing
- Socket connection correlation
- Performance counter validation

## Configuration

### Anthropic API Detection
Currently hardcoded to ports 443 (HTTPS) and 80 (HTTP). Future enhancement should include:
- IP address range filtering for known Anthropic endpoints
- Hostname resolution for `api.anthropic.com`
- SSL/TLS connection detection

### Tuning Parameters
- `HTTP_HEADER_SIZE`: Maximum HTTP header parsing (512 bytes)
- `MAX_HTTP_METHOD_LEN`: Maximum method string length (8 bytes)
- `MAX_HTTP_URI_LEN`: Maximum URI string length (128 bytes)
- `socket_connections` map size: 2048 entries

## Future Enhancements

### Planned Improvements
1. **SSL/TLS detection**: Identify encrypted vs. plain HTTP
2. **Response status codes**: Track API response success/failure
3. **Request timing**: Measure request/response latency
4. **Advanced filtering**: More sophisticated Anthropic endpoint detection

### Considerations
- **Kernel version compatibility**: Tested on Linux 5.4+
- **Performance monitoring**: Track parsing overhead and event rates
- **Error rate monitoring**: HTTP parsing failure statistics