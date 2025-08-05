//go:build ignore
// +build ignore

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-001
 * CONTEXT:   eBPF program for monitoring Claude CLI processes and API connections
 * REASON:    Need kernel-level monitoring of execve/connect syscalls for high-performance event capture
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate event data before sending to userspace, implement proper bounds checking
 * RISK:      High - Kernel crashes if bounds checking fails or invalid memory access occurs
 */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_tracing.h>
#include <bpf/bpf_endian.h>

#define MAX_COMM_LEN 16
#define MAX_PATH_LEN 256
#define MAX_HTTP_METHOD_LEN 8
#define MAX_HTTP_URI_LEN 128
#define MAX_EVENTS (256 * 1024)
#define CLAUDE_COMM "claude"
#define INACTIVITY_TIMEOUT_NS (5 * 60 * 1000000000ULL) // 5 minutes in nanoseconds
#define HTTP_HEADER_SIZE 512

// Event types for userspace processing
enum event_type {
    EVENT_EXEC = 1,
    EVENT_CONNECT = 2,
    EVENT_EXIT = 3,
    EVENT_HTTP_REQUEST = 4,
};

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-002
 * CONTEXT:   Compact event structure optimized for ring buffer communication
 * REASON:    Need efficient kernel-userspace data transfer with minimal overhead
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep structure aligned and avoid padding, validate all fields before use
 * RISK:      Medium - Structure misalignment could cause data corruption
 */
struct claude_event {
    __u64 timestamp;
    __u32 pid;
    __u32 ppid;
    __u32 uid;
    __u32 event_type;
    __u32 target_addr;  // For connect events
    __u16 target_port;  // For connect events  
    __s32 exit_code;    // For exit events
    char comm[MAX_COMM_LEN];
    char path[MAX_PATH_LEN]; // For execve events
    // HTTP request data
    char http_method[MAX_HTTP_METHOD_LEN];
    char http_uri[MAX_HTTP_URI_LEN];
    __u32 content_length;
    __u32 socket_fd;    // Socket file descriptor for correlation
} __attribute__((packed));

// Ring buffer for high-performance event communication
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, MAX_EVENTS);
} events SEC(".maps");

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-003
 * CONTEXT:   PID tracking map for filtering Claude processes
 * REASON:    Need efficient process filtering to avoid processing irrelevant syscalls
 * CHANGE:    Initial implementation.
 * PREVENTION:Clean up map entries on process exit to prevent unbounded growth
 * RISK:      Medium - Map memory exhaustion if exit events not properly handled
 */
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);
    __type(key, __u32);
    __type(value, __u64);
} claude_pids SEC(".maps");

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-025
 * CONTEXT:   Socket-to-connection mapping for correlating connect and write events
 * REASON:    Need to track which sockets belong to Anthropic API connections for HTTP parsing
 * CHANGE:    Initial implementation.
 * PREVENTION:Clean up socket entries when connections close to prevent map overflow
 * RISK:      Medium - Map growth could consume kernel memory if sockets not cleaned up
 */
struct socket_info {
    __u32 pid;
    __u32 target_addr;
    __u16 target_port;
    __u64 connect_time;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 2048);
    __type(key, __u32);  // socket FD
    __type(value, struct socket_info);
} socket_connections SEC(".maps");

// Performance counters for monitoring eBPF program health
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 5);
    __type(key, __u32);
    __type(value, __u64);
} perf_counters SEC(".maps");

enum perf_counter {
    COUNTER_EVENTS_PROCESSED = 0,
    COUNTER_EVENTS_DROPPED = 1,
    COUNTER_EXECVE_CALLS = 2,
    COUNTER_CONNECT_CALLS = 3,
    COUNTER_HTTP_REQUESTS = 4,
};

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-026
 * CONTEXT:   HTTP header parsing utilities for extracting method and URI
 * REASON:    Need to parse HTTP requests from socket write data to detect user interactions
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate buffer bounds and check for valid HTTP format before parsing
 * RISK:      High - Invalid memory access during string parsing could crash kernel
 */

static __always_inline int parse_http_method(const char *data, int data_len, char *method_out, int max_len) {
    if (data_len < 8 || max_len < 8) {  // Need at least "GET / HTTP" 
        return -1;
    }
    
    // Look for space after method
    int space_pos = -1;
    #pragma unroll
    for (int i = 0; i < MAX_HTTP_METHOD_LEN && i < data_len; i++) {
        char c;
        if (bpf_probe_read_user(&c, 1, data + i) != 0) {
            return -1;
        }
        if (c == ' ') {
            space_pos = i;
            break;
        }
        if (i < max_len - 1) {
            method_out[i] = c;
        }
    }
    
    if (space_pos == -1 || space_pos >= max_len) {
        return -1;
    }
    
    method_out[space_pos] = '\0';
    return space_pos;
}

static __always_inline int parse_http_uri(const char *data, int data_len, int method_len, char *uri_out, int max_len) {
    if (data_len < method_len + 2 || method_len >= data_len) {
        return -1;
    }
    
    // Skip method and space
    const char *uri_start = data + method_len + 1;
    int uri_max_len = data_len - method_len - 1;
    
    // Find space after URI
    int uri_len = 0;
    #pragma unroll
    for (int i = 0; i < MAX_HTTP_URI_LEN && i < uri_max_len; i++) {
        char c;
        if (bpf_probe_read_user(&c, 1, uri_start + i) != 0) {
            return -1;
        }
        if (c == ' ') {
            uri_len = i;
            break;
        }
        if (i < max_len - 1) {
            uri_out[i] = c;
        }
    }
    
    if (uri_len == 0 || uri_len >= max_len) {
        return -1;
    }
    
    uri_out[uri_len] = '\0';
    return uri_len;
}

static __always_inline __u32 parse_content_length(const char *data, int data_len) {
    const char *cl_header = "Content-Length: ";
    int cl_header_len = 16;
    
    // Search for Content-Length header
    #pragma unroll  
    for (int i = 0; i < HTTP_HEADER_SIZE && i < data_len - cl_header_len; i++) {
        int match = 1;
        
        #pragma unroll
        for (int j = 0; j < cl_header_len; j++) {
            char c;
            if (bpf_probe_read_user(&c, 1, data + i + j) != 0) {
                return 0;
            }
            if (c != cl_header[j]) {
                match = 0;
                break;
            }
        }
        
        if (match) {
            // Parse the number after "Content-Length: "
            __u32 content_len = 0;
            int start_pos = i + cl_header_len;
            
            #pragma unroll
            for (int k = 0; k < 10 && start_pos + k < data_len; k++) {
                char c;
                if (bpf_probe_read_user(&c, 1, data + start_pos + k) != 0) {
                    break;
                }
                if (c >= '0' && c <= '9') {
                    content_len = content_len * 10 + (c - '0');
                } else {
                    break;
                }
            }
            return content_len;
        }
    }
    
    return 0;
}

static __always_inline int is_anthropic_connection(__u32 addr, __u16 port) {
    // Basic check for port 443 (HTTPS) or 80 (HTTP)
    return (port == 443 || port == 80);
}

/**
 * AGENT:     ebpf-specialist  
 * TRACE:     CLAUDE-EBPF-004
 * CONTEXT:   Utility functions for safe kernel memory access and string operations
 * REASON:    eBPF verifier requires proof of memory safety for all kernel access
 * CHANGE:    Initial implementation.
 * PREVENTION:Always validate pointers and array bounds before dereferencing
 * RISK:      Critical - Memory safety violations cause program rejection or kernel crashes
 */
static __always_inline void increment_counter(enum perf_counter counter) {
    __u32 key = counter;
    __u64 *count = bpf_map_lookup_elem(&perf_counters, &key);
    if (count) {
        __sync_fetch_and_add(count, 1);
    }
}

static __always_inline int is_claude_process(const char *comm) {
    char target[] = CLAUDE_COMM;
    
    #pragma unroll
    for (int i = 0; i < sizeof(target) - 1; i++) {
        if (comm[i] != target[i]) {
            return 0;
        }
        if (comm[i] == '\0') {
            break;
        }
    }
    return 1;
}

static __always_inline int safe_strcpy(char *dst, const char *src, int max_len) {
    int i;
    
    #pragma unroll  
    for (i = 0; i < max_len - 1; i++) {
        char c;
        if (bpf_probe_read_kernel(&c, 1, src + i) != 0) {
            break;
        }
        if (c == '\0') {
            break;
        }
        dst[i] = c;
    }
    
    if (i < max_len) {
        dst[i] = '\0';
    }
    
    return i;
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-005
 * CONTEXT:   Tracepoint for capturing execve syscalls to detect Claude process launches
 * REASON:    Need to identify when Claude CLI processes start for session tracking
 * CHANGE:    Initial implementation.
 * PREVENTION:Always check comm string and use proper bounds checking for all field access
 * RISK:      Medium - Invalid string reads could cause program rejection by verifier
 */
SEC("tp/syscalls/sys_enter_execve")
int trace_execve(struct trace_event_raw_sys_enter *ctx) {
    struct claude_event *event;
    struct task_struct *task;
    char comm[MAX_COMM_LEN];
    __u32 pid, ppid, uid;
    __u64 ts;
    
    // Get current task information
    task = (struct task_struct *)bpf_get_current_task();
    if (!task) {
        return 0;
    }
    
    pid = bpf_get_current_pid_tgid() >> 32;
    ts = bpf_ktime_get_ns();
    uid = bpf_get_current_uid_gid() & 0xFFFFFFFF;
    
    // Read command name safely
    if (bpf_get_current_comm(&comm, sizeof(comm)) != 0) {
        return 0;
    }
    
    // Fast path: filter for Claude processes only
    if (!is_claude_process(comm)) {
        return 0;
    }
    
    increment_counter(COUNTER_EXECVE_CALLS);
    
    // Reserve ring buffer space
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event) {
        increment_counter(COUNTER_EVENTS_DROPPED);
        return 0;
    }
    
    // Initialize event structure
    __builtin_memset(event, 0, sizeof(*event));
    
    // Populate basic event data
    event->timestamp = ts;
    event->pid = pid;
    event->ppid = BPF_CORE_READ(task, real_parent, pid);
    event->uid = uid;
    event->event_type = EVENT_EXEC;
    
    // Copy command name safely
    __builtin_memcpy(event->comm, comm, sizeof(event->comm));
    
    // Safely read executable path
    const char __user *filename = (const char __user *)ctx->args[0];
    if (filename) {
        bpf_probe_read_user_str(event->path, sizeof(event->path), filename);
    }
    
    // Add to tracking map for future filtering
    bpf_map_update_elem(&claude_pids, &pid, &ts, BPF_ANY);
    
    // Submit event to userspace
    bpf_ringbuf_submit(event, 0);
    increment_counter(COUNTER_EVENTS_PROCESSED);
    
    return 0;
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-006
 * CONTEXT:   Tracepoint for capturing connect syscalls from Claude processes
 * REASON:    Network connections to api.anthropic.com indicate user interactions
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate socket address structure and check PID in tracking map first
 * RISK:      High - Invalid sockaddr access could crash kernel or cause security issues
 */
SEC("tp/syscalls/sys_enter_connect")
int trace_connect(struct trace_event_raw_sys_enter *ctx) {
    struct claude_event *event;
    struct sockaddr_in *addr;
    __u32 pid;
    __u64 ts;
    __u64 *start_time;
    
    pid = bpf_get_current_pid_tgid() >> 32;
    
    // Fast path: only process tracked Claude PIDs
    start_time = bpf_map_lookup_elem(&claude_pids, &pid);
    if (!start_time) {
        return 0;
    }
    
    increment_counter(COUNTER_CONNECT_CALLS);
    ts = bpf_ktime_get_ns();
    
    // Safely read socket address
    addr = (struct sockaddr_in *)ctx->args[1];
    if (!addr) {
        return 0;
    }
    
    // Validate address family (IPv4 only)
    __u16 family;
    if (bpf_probe_read_kernel(&family, sizeof(family), &addr->sin_family) != 0) {
        return 0;
    }
    
    if (family != AF_INET) {
        return 0;
    }
    
    // Reserve ring buffer space
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event) {
        increment_counter(COUNTER_EVENTS_DROPPED);
        return 0;
    }
    
    // Initialize event structure
    __builtin_memset(event, 0, sizeof(*event));
    
    // Populate event data
    event->timestamp = ts;
    event->pid = pid;
    event->uid = bpf_get_current_uid_gid() & 0xFFFFFFFF;
    event->event_type = EVENT_CONNECT;
    
    // Safely read address and port
    if (bpf_probe_read_kernel(&event->target_addr, sizeof(__u32), &addr->sin_addr.s_addr) != 0 ||
        bpf_probe_read_kernel(&event->target_port, sizeof(__u16), &addr->sin_port) != 0) {
        bpf_ringbuf_discard(event, 0);
        return 0;
    }
    
    // Convert port from network to host byte order
    event->target_port = bpf_ntohs(event->target_port);
    
    // Get process name
    bpf_get_current_comm(&event->comm, sizeof(event->comm));
    
    // Track socket for potential HTTP monitoring if this is to Anthropic
    if (is_anthropic_connection(event->target_addr, event->target_port)) {
        __u32 socket_fd = (__u32)ctx->args[0];  // Socket file descriptor
        struct socket_info sock_info = {
            .pid = pid,
            .target_addr = event->target_addr,
            .target_port = event->target_port,
            .connect_time = ts,
        };
        bpf_map_update_elem(&socket_connections, &socket_fd, &sock_info, BPF_ANY);
    }
    
    bpf_ringbuf_submit(event, 0);
    increment_counter(COUNTER_EVENTS_PROCESSED);
    
    return 0;
}

/**
 * AGENT:     ebpf-specialist  
 * TRACE:     CLAUDE-EBPF-007
 * CONTEXT:   Tracepoint for capturing process exits to clean up tracking state
 * REASON:    Need to remove PIDs from tracking map when processes exit to prevent memory leaks
 * CHANGE:    Initial implementation.
 * PREVENTION:Always clean up tracking map entries to prevent unbounded growth
 * RISK:      Medium - Map memory exhaustion if exit events not properly handled
 */
SEC("tp/sched/sched_process_exit")
int trace_exit(struct trace_event_raw_sched_process_template *ctx) {
    struct claude_event *event;
    __u32 pid;
    __u64 *start_time;
    
    pid = ctx->pid;
    
    // Check if this was a tracked Claude process
    start_time = bpf_map_lookup_elem(&claude_pids, &pid);
    if (!start_time) {
        return 0;
    }
    
    // Reserve ring buffer space for exit event
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event) {
        // Still clean up map even if we can't send event
        bpf_map_delete_elem(&claude_pids, &pid);
        increment_counter(COUNTER_EVENTS_DROPPED);
        return 0;
    }
    
    // Initialize event structure
    __builtin_memset(event, 0, sizeof(*event));
    
    // Populate exit event
    event->timestamp = bpf_ktime_get_ns();
    event->pid = pid;
    event->event_type = EVENT_EXIT;
    event->exit_code = ctx->exit_code;
    
    // Get the command name from the exiting task
    struct task_struct *task = (struct task_struct *)bpf_get_current_task_btf();
    if (task) {
        BPF_CORE_READ_STR_INTO(&event->comm, task, comm);
    }
    
    // Clean up tracking map
    bpf_map_delete_elem(&claude_pids, &pid);
    
    bpf_ringbuf_submit(event, 0);
    increment_counter(COUNTER_EVENTS_PROCESSED);
    
    return 0;
}

/**
 * AGENT:     ebpf-specialist
 * TRACE:     CLAUDE-EBPF-027
 * CONTEXT:   Tracepoint for capturing socket writes to detect HTTP requests
 * REASON:    Need to monitor socket write syscalls to parse HTTP method and content
 * CHANGE:    Initial implementation.
 * PREVENTION:Validate socket FD in tracking map, limit data parsing to first 512 bytes
 * RISK:      High - User data access requires careful bounds checking and validation
 */
SEC("tp/syscalls/sys_enter_write")
int trace_write(struct trace_event_raw_sys_enter *ctx) {
    struct claude_event *event;
    __u32 pid;
    __u32 socket_fd;
    struct socket_info *sock_info;
    const char __user *buf;
    size_t count;
    __u64 ts;
    
    pid = bpf_get_current_pid_tgid() >> 32;
    
    // Only process tracked Claude PIDs
    if (!bpf_map_lookup_elem(&claude_pids, &pid)) {
        return 0;
    }
    
    socket_fd = (__u32)ctx->args[0];
    buf = (const char __user *)ctx->args[1];
    count = (size_t)ctx->args[2];
    
    // Check if this socket is tracked (connected to Anthropic)
    sock_info = bpf_map_lookup_elem(&socket_connections, &socket_fd);
    if (!sock_info) {
        return 0;
    }
    
    // Only process reasonable HTTP header sizes
    if (count < 16 || count > HTTP_HEADER_SIZE) {
        return 0;
    }
    
    ts = bpf_ktime_get_ns();
    
    // Try to parse HTTP request
    char method[MAX_HTTP_METHOD_LEN] = {0};
    char uri[MAX_HTTP_URI_LEN] = {0};
    int method_len, uri_len;
    __u32 content_length;
    
    method_len = parse_http_method(buf, count, method, sizeof(method));
    if (method_len < 3) {  // Need at least "GET", "POST", etc.
        return 0;
    }
    
    uri_len = parse_http_uri(buf, count, method_len, uri, sizeof(uri));
    if (uri_len < 1) {
        return 0;
    }
    
    content_length = parse_content_length(buf, count);
    
    increment_counter(COUNTER_HTTP_REQUESTS);
    
    // Reserve ring buffer space
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event) {
        increment_counter(COUNTER_EVENTS_DROPPED);
        return 0;
    }
    
    // Initialize event structure
    __builtin_memset(event, 0, sizeof(*event));
    
    // Populate event data
    event->timestamp = ts;
    event->pid = pid;
    event->uid = bpf_get_current_uid_gid() & 0xFFFFFFFF;
    event->event_type = EVENT_HTTP_REQUEST;
    event->target_addr = sock_info->target_addr;
    event->target_port = sock_info->target_port;
    event->socket_fd = socket_fd;
    event->content_length = content_length;
    
    // Copy HTTP method and URI
    __builtin_memcpy(event->http_method, method, sizeof(event->http_method));
    __builtin_memcpy(event->http_uri, uri, sizeof(event->http_uri));
    
    // Get process name
    bpf_get_current_comm(&event->comm, sizeof(event->comm));
    
    bpf_ringbuf_submit(event, 0);
    increment_counter(COUNTER_EVENTS_PROCESSED);
    
    return 0;
}

char LICENSE[] SEC("license") = "GPL";