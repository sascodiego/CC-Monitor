/**
 * CONTEXT:   HTTP monitoring for Claude API calls and web activity detection
 * INPUT:     Network traffic from Claude processes to Claude/Anthropic endpoints
 * OUTPUT:    HTTP activity events for precise Claude usage tracking
 * BUSINESS:  HTTP monitoring provides definitive proof of Claude API usage
 * CHANGE:    Initial HTTP monitor based on prototype with Claude-specific patterns
 * RISK:      High - Network monitoring affecting system performance and privacy
 */

package monitor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
 * CONTEXT:   HTTP event types for Claude API activity classification
 * INPUT:     HTTP traffic analysis from Claude processes
 * OUTPUT:    Classified HTTP events for API usage tracking
 * BUSINESS:  HTTP event classification enables precise API usage analytics
 * CHANGE:    Initial HTTP event types for Claude API monitoring
 * RISK:      Low - Event type definitions for HTTP operations
 */
type HTTPEventType string

const (
	HTTPRequest    HTTPEventType = "HTTP_REQUEST"
	HTTPResponse   HTTPEventType = "HTTP_RESPONSE"
	HTTPPost       HTTPEventType = "HTTP_POST"
	HTTPGet        HTTPEventType = "HTTP_GET"
	HTTPClaudeAPI  HTTPEventType = "HTTP_CLAUDE_API"
	HTTPError      HTTPEventType = "HTTP_ERROR"
)

type HTTPEvent struct {
	Type           HTTPEventType          `json:"type"`
	Timestamp      time.Time              `json:"timestamp"`
	PID            int                    `json:"pid"`
	ProcessName    string                 `json:"process_name"`
	Method         string                 `json:"method"`
	URL            string                 `json:"url"`
	Host           string                 `json:"host"`
	Port           int                    `json:"port"`                  // Added from https-system-detector
	Protocol       string                 `json:"protocol"`              // Added: HTTP or HTTPS
	StatusCode     int                    `json:"status_code"`
	ContentType    string                 `json:"content_type"`
	ContentLength  int                    `json:"content_length"`
	BytesSent      int64                  `json:"bytes_sent"`            // Added from https-system-detector
	BytesRecv      int64                  `json:"bytes_recv"`            // Added from https-system-detector  
	Duration       int64                  `json:"duration_ms,omitempty"` // Added from https-system-detector
	UserAgent      string                 `json:"user_agent"`
	IsClaudeAPI    bool                   `json:"is_claude_api"`
	ProjectPath    string                 `json:"project_path"`
	ProjectName    string                 `json:"project_name"`
	RequestBody    string                 `json:"request_body,omitempty"`
	ResponseBody   string                 `json:"response_body,omitempty"`
	Details        map[string]interface{} `json:"details"`
}

/**
 * CONTEXT:   HTTP monitor for Claude API calls and web activity
 * INPUT:     Network traffic monitoring for tracked Claude processes
 * OUTPUT:    HTTP events for API usage and web activity tracking
 * BUSINESS:  HTTP monitoring provides definitive Claude API usage detection
 * CHANGE:    Initial HTTP monitor with Claude API focus
 * RISK:      High - Network monitoring affecting system performance
 */
type HTTPMonitor struct {
	ctx              context.Context
	cancel           context.CancelFunc
	eventCallback    func(HTTPEvent)
	trackedProcesses map[int]*ProcessInfo
	claudeEndpoints  []*regexp.Regexp
	httpPorts        []int
	mu               sync.RWMutex
	running          bool
	stats            HTTPStats
	config           HTTPMonitorConfig
	proxyServer      *http.Server
	
	// Advanced monitoring from https-system-detector
	connectionTracker *ConnectionTracker
	stopChannels     map[string]chan bool
	wg               sync.WaitGroup
}

/**
 * CONTEXT:   Advanced HTTP monitoring configuration with multiple detection methods
 * INPUT:     HTTP monitoring parameters for comprehensive Claude activity tracking
 * OUTPUT:    Multi-method HTTP monitor configuration for precise work detection
 * BUSINESS:  Advanced configuration enables comprehensive HTTP/HTTPS monitoring without proxy
 * CHANGE:    Enhanced configuration with /proc/net, ss, and tcpdump methods from https-system-detector
 * RISK:      Low - Multi-method configuration for robust HTTP activity detection
 */
type HTTPMonitorConfig struct {
	MonitorPorts     []int  `json:"monitor_ports"`     // Ports to monitor (443, 80, 8080, etc.)
	UseTcpdump       bool   `json:"use_tcpdump"`       // Use tcpdump for packet capture (requires sudo)
	UseSSMonitor     bool   `json:"use_ss_monitor"`    // Use ss (socket statistics) for detailed info
	UseProcNet       bool   `json:"use_proc_net"`      // Use /proc/net/tcp monitoring (default)
	UseProxy         bool   `json:"use_proxy"`         // Use HTTP proxy interceptor (optional)
	ProxyPort        int    `json:"proxy_port"`        // Proxy port (default 8888)
	CaptureBody      bool   `json:"capture_body"`      // Capture request bodies for work analysis
	MaxBodySize      int    `json:"max_body_size"`     // Maximum body size to capture
	VerboseLogging   bool   `json:"verbose_logging"`   // Enable verbose logging
	MethodInference  bool   `json:"method_inference"`  // Infer GET/POST from traffic patterns
	TrackConnections bool   `json:"track_connections"` // Track connection lifecycle
	PollingInterval  int    `json:"polling_ms"`        // Polling interval in milliseconds
}

/**
 * CONTEXT:   HTTP monitoring statistics
 * INPUT:     HTTP monitoring metrics and API usage data
 * OUTPUT:    Performance and usage statistics
 * BUSINESS:  Statistics enable HTTP monitoring optimization and API usage insights
 * CHANGE:    Initial HTTP monitoring statistics
 * RISK:      Low - Statistics structure for HTTP monitoring metrics
 */
type HTTPStats struct {
	TotalRequests     uint64            `json:"total_requests"`
	ClaudeAPIRequests uint64            `json:"claude_api_requests"`
	RequestsByMethod  map[string]uint64 `json:"requests_by_method"`
	RequestsByHost    map[string]uint64 `json:"requests_by_host"`
	RequestsByStatus  map[string]uint64 `json:"requests_by_status"`
	TotalDataSent     int64             `json:"total_data_sent"`
	TotalDataReceived int64             `json:"total_data_received"`
	StartTime         time.Time         `json:"start_time"`
	LastRequestTime   time.Time         `json:"last_request_time"`
	ErrorCount        uint64            `json:"error_count"`
	
	// Advanced statistics from https-system-detector
	ActiveConnections int               `json:"active_connections"`
	TotalConnections  uint64            `json:"total_connections"`
	HTTPConnections   uint64            `json:"http_connections"`
	HTTPSConnections  uint64            `json:"https_connections"`
	AverageConnTime   time.Duration     `json:"average_conn_time"`
	MethodsInferred   map[string]uint64 `json:"methods_inferred"`
}

/**
 * CONTEXT:   Connection tracking for HTTP/HTTPS activity monitoring
 * INPUT:     Network connections from Claude processes
 * OUTPUT:    Connection lifecycle events and statistics
 * BUSINESS:  Connection tracking enables precise HTTP activity detection
 * CHANGE:    Connection tracker implementation from https-system-detector
 * RISK:      Medium - Connection tracking affecting monitoring performance
 */
type ConnectionTracker struct {
	mu          sync.RWMutex
	connections map[string]*ConnectionInfo
	events      chan HTTPEvent
}

type ConnectionInfo struct {
	StartTime   time.Time `json:"start_time"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Protocol    string    `json:"protocol"` // HTTP or HTTPS
	BytesSent   int64     `json:"bytes_sent"`
	BytesRecv   int64     `json:"bytes_recv"`
	Method      string    `json:"method,omitempty"` // Inferred method
	Path        string    `json:"path,omitempty"`
	ProcessName string    `json:"process_name"`
	PID         int       `json:"pid"`
	LocalAddr   string    `json:"local_addr"`
	RemoteAddr  string    `json:"remote_addr"`
	State       string    `json:"state"`
}

/**
 * CONTEXT:   Create advanced HTTP monitor with multiple detection methods
 * INPUT:     Event callback and comprehensive HTTP monitoring configuration
 * OUTPUT:    Multi-method HTTP monitor ready for advanced Claude activity tracking
 * BUSINESS:  Advanced HTTP monitor creation enables comprehensive HTTP/HTTPS work detection
 * CHANGE:    Enhanced HTTP monitor with /proc/net, ss, and tcpdump methods from https-system-detector
 * RISK:      Medium - Advanced HTTP monitor initialization for comprehensive monitoring
 */
func NewHTTPMonitor(callback func(HTTPEvent), config HTTPMonitorConfig) *HTTPMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set enhanced default configuration
	if len(config.MonitorPorts) == 0 {
		config.MonitorPorts = []int{443, 80, 8080, 8443, 3000, 5000, 8000, 9000} // Extended port list
	}
	if config.MaxBodySize == 0 {
		config.MaxBodySize = 2048 // 2KB for request analysis
	}
	if config.ProxyPort == 0 {
		config.ProxyPort = 8888 // Default proxy port
	}
	if config.PollingInterval == 0 {
		config.PollingInterval = 500 // 500ms polling by default
	}
	
	// Enable advanced monitoring methods by default
	config.UseProcNet = true      // Primary method - always enabled
	config.UseSSMonitor = true    // Enable ss monitoring for detailed stats
	config.MethodInference = true // Enable intelligent method inference
	config.TrackConnections = true // Enable connection lifecycle tracking
	
	// Claude/Anthropic API endpoints
	claudeEndpoints := []*regexp.Regexp{
		// Primary Claude endpoints
		regexp.MustCompile(`(?i).*api\.anthropic\.com.*`),
		regexp.MustCompile(`(?i).*claude\.ai.*`),
		regexp.MustCompile(`(?i).*api\.claude\.ai.*`),
		regexp.MustCompile(`(?i).*anthropic\.com.*`),
		
		// Development and alternative endpoints
		regexp.MustCompile(`(?i).*claude-dev\.anthropic\.com.*`),
		regexp.MustCompile(`(?i).*staging\.anthropic\.com.*`),
		regexp.MustCompile(`(?i).*beta\.claude\.ai.*`),
		
		// Common Claude API IP addresses
		regexp.MustCompile(`^34\.36\.57\.103$`),
		regexp.MustCompile(`^160\.79\.104\.10$`),
		regexp.MustCompile(`^34\.102\.136\.180$`),
		regexp.MustCompile(`^35\.227\.210\.155$`),
		
		// Local development endpoints
		regexp.MustCompile(`(?i)localhost.*claude.*`),
		regexp.MustCompile(`(?i)127\.0\.0\.1.*claude.*`),
	}
	
	monitor := &HTTPMonitor{
		ctx:             ctx,
		cancel:          cancel,
		eventCallback:   callback,
		trackedProcesses: make(map[int]*ProcessInfo),
		claudeEndpoints: claudeEndpoints,
		httpPorts:       config.MonitorPorts,
		config:          config,
		stats: HTTPStats{
			RequestsByMethod: make(map[string]uint64),
			RequestsByHost:   make(map[string]uint64),
			RequestsByStatus: make(map[string]uint64),
			StartTime:        time.Now(),
		},
		// Advanced monitoring components
		connectionTracker: NewConnectionTracker(),
		stopChannels:     make(map[string]chan bool),
	}
	
	return monitor
}

/**
 * CONTEXT:   Create new connection tracker for HTTP connection lifecycle monitoring
 * INPUT:     Connection tracking initialization
 * OUTPUT:    Configured connection tracker ready for HTTP monitoring
 * BUSINESS:  Connection tracker enables precise HTTP activity detection
 * CHANGE:    Connection tracker implementation from https-system-detector
 * RISK:      Low - Connection tracker initialization
 */
func NewConnectionTracker() *ConnectionTracker {
	return &ConnectionTracker{
		connections: make(map[string]*ConnectionInfo),
		events:      make(chan HTTPEvent, 100),
	}
}

/**
 * CONTEXT:   Start advanced HTTP monitoring with multiple detection methods
 * INPUT:     Monitor activation request
 * OUTPUT:    Active multi-method HTTP monitoring with comprehensive API detection
 * BUSINESS:  Advanced HTTP monitor start enables comprehensive Claude API usage tracking
 * CHANGE:    Enhanced HTTP monitor start with /proc/net, ss, tcpdump methods from https-system-detector
 * RISK:      High - Multi-method network monitoring affecting system performance
 */
func (hm *HTTPMonitor) Start() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	if hm.running {
		return fmt.Errorf("HTTP monitor is already running")
	}
	
	hm.running = true
	
	// Initialize advanced statistics
	if hm.stats.MethodsInferred == nil {
		hm.stats.MethodsInferred = make(map[string]uint64)
	}
	
	// Method 1: /proc/net/tcp monitoring (primary - always enabled)
	if hm.config.UseProcNet {
		hm.startProcNetMonitor()
		log.Printf("📊 HTTP monitor: /proc/net monitoring started")
	}
	
	// Method 2: Socket statistics monitoring (detailed byte info)
	if hm.config.UseSSMonitor && hm.canUseSS() {
		hm.startSSMonitor()
		log.Printf("📊 HTTP monitor: ss (socket statistics) monitoring started")
	}
	
	// Method 3: tcpdump packet capture (comprehensive - requires sudo)
	if hm.config.UseTcpdump && hm.canUseTcpdump() {
		hm.startTcpdumpMonitor()
		log.Printf("📊 HTTP monitor: tcpdump packet capture started")
	}
	
	// Method 4: Optional proxy interceptor
	if hm.config.UseProxy {
		go hm.monitorWithProxy()
		log.Printf("📊 HTTP monitor: proxy interceptor started on port %d", hm.config.ProxyPort)
	}
	
	// Method 5: Fallback connection monitoring
	if !hm.config.UseProcNet && !hm.config.UseSSMonitor {
		go hm.monitorConnections()
		log.Printf("📊 HTTP monitor: fallback connection monitoring started")
	}
	
	// Start connection event processor
	go hm.processConnectionEvents()
	
	log.Printf("🚀 Advanced HTTP monitor started with %d methods enabled", hm.getEnabledMethodsCount())
	return nil
}

/**
 * CONTEXT:   Stop advanced HTTP monitoring system with multiple methods
 * INPUT:     Monitor shutdown request
 * OUTPUT:    Cleanly stopped multi-method HTTP monitoring
 * BUSINESS:  Advanced HTTP monitor stop enables graceful shutdown of all monitoring methods
 * CHANGE:    Enhanced HTTP monitor stop with multiple method shutdown from https-system-detector
 * RISK:      Medium - Multi-method monitor shutdown affecting tracked connections
 */
func (hm *HTTPMonitor) Stop() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	if !hm.running {
		return nil
	}
	
	hm.cancel()
	hm.running = false
	
	// Stop all monitoring methods
	for method, stop := range hm.stopChannels {
		log.Printf("📊 Stopping %s monitor...", method)
		close(stop)
	}
	
	// Wait for all goroutines to finish
	hm.wg.Wait()
	
	// Stop proxy server if running
	if hm.proxyServer != nil {
		if err := hm.proxyServer.Close(); err != nil {
			log.Printf("Error stopping HTTP proxy server: %v", err)
		}
		hm.proxyServer = nil
	}
	
	// Close connection tracker events channel
	if hm.connectionTracker != nil {
		close(hm.connectionTracker.events)
	}
	
	log.Printf("🚀 Advanced HTTP monitor stopped - All methods shutdown complete")
	return nil
}

/**
 * CONTEXT:   Dynamically attach HTTP monitoring to specific process
 * INPUT:     Process ID to monitor
 * OUTPUT:    Process-specific HTTP monitoring attachment
 * BUSINESS:  Dynamic attachment enables per-process HTTP traffic tracking
 * CHANGE:    Event-driven process attachment for HTTP monitoring
 * RISK:      High - Dynamic HTTP monitoring affecting network performance
 */
func (hm *HTTPMonitor) AttachToProcess(pid int) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	if !hm.running {
		return fmt.Errorf("HTTP monitor not running")
	}
	
	// Check if already attached
	if _, exists := hm.trackedProcesses[pid]; exists {
		return nil // Already attached
	}
	
	// Get process information
	processInfo, err := hm.getProcessInfo(pid)
	if err != nil {
		return fmt.Errorf("failed to get process info for PID %d: %w", pid, err)
	}
	
	// Add to tracked processes
	hm.trackedProcesses[pid] = processInfo
	
	log.Printf("Attached HTTP monitoring to PID %d (%s)", pid, processInfo.Command)
	return nil
}

/**
 * CONTEXT:   Dynamically detach HTTP monitoring from specific process
 * INPUT:     Process ID to stop monitoring
 * OUTPUT:    Process-specific HTTP monitoring detachment
 * BUSINESS:  Dynamic detachment prevents resource leaks from stopped processes
 * CHANGE:    Event-driven process detachment for HTTP monitoring cleanup
 * RISK:      Low - HTTP monitoring detachment for resource cleanup
 */
func (hm *HTTPMonitor) DetachFromProcess(pid int) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	// Check if attached
	processInfo, exists := hm.trackedProcesses[pid]
	if !exists {
		return nil // Not attached
	}
	
	// Remove from tracked processes
	delete(hm.trackedProcesses, pid)
	
	log.Printf("Detached HTTP monitoring from PID %d (%s)", pid, processInfo.Command)
	return nil
}

/**
 * CONTEXT:   Update tracked processes for HTTP monitoring
 * INPUT:     Map of Claude processes to monitor
 * OUTPUT:    Updated HTTP monitoring targets
 * BUSINESS:  Process tracking enables selective HTTP monitoring
 * CHANGE:    Initial process tracking update for HTTP monitoring
 * RISK:      Low - Process list update with synchronization
 */
func (hm *HTTPMonitor) UpdateTrackedProcesses(processes map[int]*ProcessInfo) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	
	hm.trackedProcesses = make(map[int]*ProcessInfo)
	for pid, info := range processes {
		hm.trackedProcesses[pid] = info
	}
	
	if hm.config.VerboseLogging {
		log.Printf("HTTP monitor tracking %d Claude processes", len(hm.trackedProcesses))
	}
}

/**
 * CONTEXT:   Monitor HTTP traffic using tcpdump packet capture
 * INPUT:     Network packet capture and HTTP parsing
 * OUTPUT:    HTTP events from packet analysis
 * BUSINESS:  Tcpdump monitoring provides comprehensive HTTP traffic analysis
 * CHANGE:    Initial tcpdump-based HTTP monitoring
 * RISK:      High - Packet capture affecting network performance
 */
func (hm *HTTPMonitor) monitorWithTcpdump() {
	// Build port filter for tcpdump
	portFilter := hm.buildPortFilter()
	
	cmd := exec.Command("tcpdump", "-i", "any", "-A", "-s", "1500",
		fmt.Sprintf("tcp and (%s)", portFilter), "-l")
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create tcpdump stdout pipe: %v", err)
		return
	}
	
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start tcpdump: %v", err)
		return
	}
	
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()
	
	scanner := bufio.NewScanner(stdout)
	var packetBuffer strings.Builder
	inPacket := false
	
	for scanner.Scan() {
		select {
		case <-hm.ctx.Done():
			return
		default:
			line := scanner.Text()
			
			// Detect packet boundaries
			if strings.Contains(line, " > ") && (strings.Contains(line, ":80 ") || strings.Contains(line, ":443 ") || strings.Contains(line, ":8080 ")) {
				// Process previous packet if exists
				if packetBuffer.Len() > 0 {
					hm.parseHTTPPacket(packetBuffer.String())
				}
				
				packetBuffer.Reset()
				inPacket = true
			}
			
			if inPacket {
				packetBuffer.WriteString(line + "\n")
			}
		}
	}
}

/**
 * CONTEXT:   Monitor HTTP traffic using proxy interception from prototype
 * INPUT:     HTTP proxy requests and responses with comprehensive analysis
 * OUTPUT:    HTTP events from detailed proxy analysis with content inspection
 * BUSINESS:  Proxy monitoring provides complete HTTP request/response analysis for work tracking
 * CHANGE:    Advanced HTTP proxy interceptor from prototype with full request analysis
 * RISK:      High - HTTP proxy affecting application traffic and requiring configuration
 */
func (hm *HTTPMonitor) monitorWithProxy() {
	// Setup HTTP proxy server based on prototype implementation
	hm.proxyServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", hm.config.ProxyPort),
		Handler: http.HandlerFunc(hm.handleProxyRequest),
	}
	
	// Start proxy server in goroutine
	go func() {
		log.Printf("HTTP proxy interceptor started on port %d", hm.config.ProxyPort)
		if err := hm.proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP proxy server error: %v", err)
		}
	}()
	
	// Wait for context cancellation
	<-hm.ctx.Done()
}

/**
 * CONTEXT:   Handle HTTP proxy requests with comprehensive analysis
 * INPUT:     HTTP requests routed through proxy
 * OUTPUT:    HTTP events with detailed request/response analysis
 * BUSINESS:  Proxy request handling enables complete HTTP traffic analysis
 * CHANGE:    Advanced proxy request handler from prototype with work detection
 * RISK:      High - Request interception affecting application behavior
 */
func (hm *HTTPMonitor) handleProxyRequest(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// Capture request body if enabled
	var requestBodyStr string
	
	if hm.config.CaptureBody && r.Method == "POST" {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			// Limit body size for analysis
			bodySize := len(body)
			if bodySize > hm.config.MaxBodySize {
				bodySize = hm.config.MaxBodySize
			}
			requestBodyStr = string(body[:bodySize])
			
			// Restore body for forwarding
			r.Body = io.NopCloser(bytes.NewBuffer(body))
		}
	}
	
	// Check if this is Claude API endpoint
	isClaudeAPI := hm.isClaudeEndpoint(r.Host)
	
	// Create HTTP event for request
	event := HTTPEvent{
		Type:          HTTPRequest,
		Timestamp:     startTime,
		Method:        r.Method,
		URL:           r.URL.String(),
		Host:          r.Host,
		ContentType:   r.Header.Get("Content-Type"),
		ContentLength: int(r.ContentLength),
		UserAgent:     r.Header.Get("User-Agent"),
		IsClaudeAPI:   isClaudeAPI,
		RequestBody:   requestBodyStr,
		Details: map[string]interface{}{
			"proxy_intercepted": true,
			"headers":          hm.extractHeaders(r.Header),
			"method":           r.Method,
			"content_length":   r.ContentLength,
		},
	}
	
	// Special handling for POST requests (work detection)
	if r.Method == "POST" {
		event.Type = HTTPPost
		
		// Additional work indicators for POST requests
		if isClaudeAPI {
			event.Type = HTTPClaudeAPI
			event.Details["work_indicator"] = true
			event.Details["api_usage"] = true
		}
	}
	
	// Process the event
	hm.processHTTPEvent(event)
	
	// Forward request to target
	hm.forwardProxyRequest(w, r, event, startTime)
}

/**
 * CONTEXT:   Forward proxy request to target server
 * INPUT:     HTTP request and response writer
 * OUTPUT:    Forwarded request with response analysis
 * BUSINESS:  Request forwarding maintains application functionality while monitoring
 * CHANGE:    Advanced request forwarding from prototype with response analysis
 * RISK:      High - Request forwarding affecting application performance
 */
func (hm *HTTPMonitor) forwardProxyRequest(w http.ResponseWriter, r *http.Request, originalEvent HTTPEvent, startTime time.Time) {
	// Create HTTP client for forwarding
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Build target URL
	targetURL := fmt.Sprintf("http://%s%s", r.Host, r.URL.String())
	if r.TLS != nil {
		targetURL = fmt.Sprintf("https://%s%s", r.Host, r.URL.String())
	}
	
	// Create forwarding request
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		hm.handleProxyError(w, err, originalEvent)
		return
	}
	
	// Copy headers
	proxyReq.Header = r.Header.Clone()
	
	// Execute forwarding request
	resp, err := client.Do(proxyReq)
	if err != nil {
		hm.handleProxyError(w, err, originalEvent)
		return
	}
	defer resp.Body.Close()
	
	// Capture response body if enabled
	var responseBody []byte
	var responseBodyStr string
	
	if hm.config.CaptureBody && originalEvent.IsClaudeAPI {
		respBody, err := io.ReadAll(resp.Body)
		if err == nil {
			responseBody = respBody
			// Limit response body size
			bodySize := len(respBody)
			if bodySize > hm.config.MaxBodySize {
				bodySize = hm.config.MaxBodySize
			}
			responseBodyStr = string(respBody[:bodySize])
		}
		
		// Restore response body
		resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	}
	
	// Create response event
	responseEvent := HTTPEvent{
		Type:          HTTPResponse,
		Timestamp:     time.Now(),
		Method:        originalEvent.Method,
		URL:           originalEvent.URL,
		Host:          originalEvent.Host,
		StatusCode:    resp.StatusCode,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: int(resp.ContentLength),
		IsClaudeAPI:   originalEvent.IsClaudeAPI,
		RequestBody:   originalEvent.RequestBody,
		ResponseBody:  responseBodyStr,
		Details: map[string]interface{}{
			"proxy_intercepted":  true,
			"response_headers":   hm.extractHeaders(resp.Header),
			"status_code":        resp.StatusCode,
			"response_time_ms":   time.Since(startTime).Milliseconds(),
			"request_duration":   time.Since(startTime).String(),
		},
	}
	
	// Process response event
	hm.processHTTPEvent(responseEvent)
	
	// Copy response to client
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	
	// Copy response body
	if responseBody != nil {
		w.Write(responseBody)
	} else {
		io.Copy(w, resp.Body)
	}
	
	// Log successful proxy operation
	if hm.config.VerboseLogging {
		log.Printf("🔄 Proxy Request: %s %s -> %d (%dms, Claude: %t)", 
			r.Method, r.Host, resp.StatusCode, time.Since(startTime).Milliseconds(), originalEvent.IsClaudeAPI)
	}
}

/**
 * CONTEXT:   Handle proxy errors
 * INPUT:     HTTP error and original event
 * OUTPUT:    Error response and error event
 * BUSINESS:  Error handling maintains proxy reliability
 * CHANGE:    Proxy error handling from prototype
 * RISK:      Medium - Error handling affecting proxy reliability
 */
func (hm *HTTPMonitor) handleProxyError(w http.ResponseWriter, err error, originalEvent HTTPEvent) {
	// Create error event
	errorEvent := HTTPEvent{
		Type:        HTTPError,
		Timestamp:   time.Now(),
		Method:      originalEvent.Method,
		URL:         originalEvent.URL,
		Host:        originalEvent.Host,
		IsClaudeAPI: originalEvent.IsClaudeAPI,
		Details: map[string]interface{}{
			"proxy_intercepted": true,
			"error":            err.Error(),
			"error_type":       "proxy_forward_error",
		},
	}
	
	hm.processHTTPEvent(errorEvent)
	
	// Return error response
	http.Error(w, fmt.Sprintf("Proxy error: %v", err), http.StatusBadGateway)
	
	log.Printf("🚨 Proxy Error: %s %s -> %v", originalEvent.Method, originalEvent.Host, err)
}

/**
 * CONTEXT:   Extract HTTP headers for analysis
 * INPUT:     HTTP header map
 * OUTPUT:    Simplified header map for event details
 * BUSINESS:  Header extraction enables request analysis
 * CHANGE:    Header extraction utility from prototype
 * RISK:      Low - Header processing utility
 */
func (hm *HTTPMonitor) extractHeaders(headers http.Header) map[string]string {
	headerMap := make(map[string]string)
	
	// Extract important headers for analysis
	importantHeaders := []string{
		"Content-Type", "Content-Length", "User-Agent", "Authorization", 
		"Accept", "Accept-Language", "Cache-Control", "Connection",
	}
	
	for _, name := range importantHeaders {
		if value := headers.Get(name); value != "" {
			headerMap[name] = value
		}
	}
	
	return headerMap
}

/**
 * CONTEXT:   Monitor HTTP connections by analyzing network connections
 * INPUT:     Network connection analysis for tracked processes
 * OUTPUT:    HTTP events from connection analysis
 * BUSINESS:  Connection monitoring provides HTTP activity detection
 * CHANGE:    Initial connection-based HTTP monitoring
 * RISK:      Medium - Network connection analysis affecting performance
 */
func (hm *HTTPMonitor) monitorConnections() {
	ticker := time.NewTicker(3 * time.Second) // Check connections every 3 seconds
	defer ticker.Stop()
	
	knownConnections := make(map[string]bool)
	
	for {
		select {
		case <-hm.ctx.Done():
			return
		case <-ticker.C:
			connections := hm.getHTTPConnections()
			
			for _, conn := range connections {
				connKey := fmt.Sprintf("%d:%s->%s", conn["pid"], conn["local"], conn["remote"])
				
				if !knownConnections[connKey] {
					hm.processHTTPConnection(conn)
					knownConnections[connKey] = true
				}
			}
		}
	}
}

/**
 * CONTEXT:   Get HTTP connections for tracked Claude processes
 * INPUT:     Process connection analysis
 * OUTPUT:    HTTP connections from Claude processes
 * BUSINESS:  Connection analysis enables HTTP activity detection
 * CHANGE:    Initial HTTP connection detection for Claude processes
 * RISK:      Medium - Connection analysis affecting system performance
 */
func (hm *HTTPMonitor) getHTTPConnections() []map[string]interface{} {
	var httpConnections []map[string]interface{}
	
	hm.mu.RLock()
	trackedPIDs := make(map[int]bool)
	for pid := range hm.trackedProcesses {
		trackedPIDs[pid] = true
	}
	hm.mu.RUnlock()
	
	// Read network connections for tracked processes
	for pid := range trackedPIDs {
		connections := hm.getProcessHTTPConnections(pid)
		httpConnections = append(httpConnections, connections...)
	}
	
	return httpConnections
}

/**
 * CONTEXT:   Get HTTP connections for specific process
 * INPUT:     Process ID for connection analysis
 * OUTPUT:    HTTP connections for the process
 * BUSINESS:  Process-specific connection analysis enables targeted monitoring
 * CHANGE:    Initial process HTTP connection detection
 * RISK:      Medium - Process connection analysis affecting performance
 */
func (hm *HTTPMonitor) getProcessHTTPConnections(pid int) []map[string]interface{} {
	var connections []map[string]interface{}
	
	// Get socket inodes for the process
	socketInodes := hm.getSocketInodes(pid)
	
	if len(socketInodes) == 0 {
		return connections
	}
	
	// Check TCP connections
	connections = append(connections, hm.getTCPConnectionsByInodes(socketInodes, pid)...)
	
	return connections
}

/**
 * CONTEXT:   Parse HTTP packet from tcpdump output
 * INPUT:     Raw packet data from tcpdump
 * OUTPUT:    Parsed HTTP event if packet contains HTTP traffic
 * BUSINESS:  Packet parsing extracts HTTP request/response details
 * CHANGE:    Initial HTTP packet parsing for Claude API detection
 * RISK:      Medium - Packet parsing accuracy affecting HTTP event quality
 */
func (hm *HTTPMonitor) parseHTTPPacket(packet string) {
	lines := strings.Split(packet, "\n")
	if len(lines) < 2 {
		return
	}
	
	// Extract connection info from first line
	connLine := lines[0]
	var srcIP, dstIP, srcPort, dstPort string
	
	// Parse connection line (simplified)
	if parts := strings.Fields(connLine); len(parts) >= 3 {
		src := parts[2]
		dst := parts[4]
		
		// Extract IP and port
		if srcParts := strings.Split(src, "."); len(srcParts) >= 4 {
			srcIP = strings.Join(srcParts[:4], ".")
			if len(srcParts) > 4 {
				srcPort = srcParts[4]
			}
		}
		if dstParts := strings.Split(dst, "."); len(dstParts) >= 4 {
			dstIP = strings.Join(dstParts[:4], ".")
			if len(dstParts) > 4 {
				dstPort = strings.TrimSuffix(dstParts[4], ":")
			}
		}
	}
	
	// Look for HTTP patterns in packet data
	for _, line := range lines[1:] {
		if hm.parseHTTPRequest(line, srcIP, dstIP, srcPort, dstPort) {
			break
		}
	}
}

/**
 * CONTEXT:   Parse HTTP POST requests for Claude work activity detection
 * INPUT:     Packet line and connection information
 * OUTPUT:    HTTP event if line contains POST request to Claude API
 * BUSINESS:  POST request parsing enables precise work activity detection
 * CHANGE:    Focused POST request parsing for work activity tracking
 * RISK:      Medium - Request parsing affecting work activity detection accuracy
 */
func (hm *HTTPMonitor) parseHTTPRequest(line, srcIP, dstIP, srcPort, dstPort string) bool {
	// Only monitor POST requests for work detection
	if !strings.Contains(line, "POST ") {
		return false
	}
	
	// Extract URL and host for POST request
	parts := strings.Fields(line)
	var url, host string
	
	for i, part := range parts {
		if part == "POST" && i+1 < len(parts) {
			url = parts[i+1]
			break
		}
	}
	
	// Look for Host header in the line or subsequent processing
	if strings.Contains(line, "Host: ") {
		hostParts := strings.Split(line, "Host: ")
		if len(hostParts) > 1 {
			host = strings.Fields(hostParts[1])[0]
		}
	}
	
	// Only process if this is a Claude API endpoint
	if !hm.isClaudeEndpoint(host) {
		return false
	}
	
	// Extract content length for work analysis
	var contentLength int
	if strings.Contains(line, "Content-Length: ") {
		lengthParts := strings.Split(line, "Content-Length: ")
		if len(lengthParts) > 1 {
			lengthStr := strings.Fields(lengthParts[1])[0]
			contentLength, _ = strconv.Atoi(lengthStr)
		}
	}
	
	// Create work activity event for Claude API POST
	event := HTTPEvent{
		Type:          HTTPPost,
		Timestamp:     time.Now(),
		Method:        "POST",
		URL:           url,
		Host:          host,
		ContentLength: contentLength,
		IsClaudeAPI:   true, // Already verified above
		Details: map[string]interface{}{
			"work_indicator": true,
			"src_ip":        srcIP,
			"dst_ip":        dstIP,
			"src_port":      srcPort,
			"dst_port":      dstPort,
		},
	}
	
	hm.processHTTPEvent(event)
	
	if hm.config.VerboseLogging {
		log.Printf("🔥 Work Activity Detected: POST to %s (Content: %d bytes)", host, contentLength)
	}
	
	return true
}

/**
 * CONTEXT:   Process HTTP connection for event generation
 * INPUT:     HTTP connection data
 * OUTPUT:    HTTP event for connection activity
 * BUSINESS:  Connection processing enables HTTP activity tracking
 * CHANGE:    Initial HTTP connection processing
 * RISK:      Low - Connection processing for HTTP event generation
 */
func (hm *HTTPMonitor) processHTTPConnection(conn map[string]interface{}) {
	pid, _ := conn["pid"].(int)
	remote, _ := conn["remote"].(string)
	local, _ := conn["local"].(string)
	
	// Extract host from remote address
	host := strings.Split(remote, ":")[0]
	
	// Check if this is a Claude endpoint
	isClaudeAPI := hm.isClaudeEndpoint(host)
	
	event := HTTPEvent{
		Type:        HTTPRequest,
		Timestamp:   time.Now(),
		PID:         pid,
		ProcessName: hm.getProcessName(pid),
		Host:        host,
		IsClaudeAPI: isClaudeAPI,
		ProjectPath: hm.getProcessProjectPath(pid),
		ProjectName: hm.getProcessProjectName(pid),
		Details: map[string]interface{}{
			"local":  local,
			"remote": remote,
			"connection_type": "tcp",
		},
	}
	
	if isClaudeAPI {
		event.Type = HTTPClaudeAPI
	}
	
	hm.processHTTPEvent(event)
}

/**
 * CONTEXT:   Process HTTP event and send to callback
 * INPUT:     HTTP event with request/response details
 * OUTPUT:    Processed HTTP event sent to callback
 * BUSINESS:  Event processing enables HTTP activity tracking and API usage analytics
 * CHANGE:    Initial HTTP event processing with statistics
 * RISK:      Low - Event processing with statistics tracking
 */
func (hm *HTTPMonitor) processHTTPEvent(event HTTPEvent) {
	hm.mu.Lock()
	hm.stats.TotalRequests++
	if event.IsClaudeAPI {
		hm.stats.ClaudeAPIRequests++
	}
	hm.stats.RequestsByMethod[event.Method]++
	if event.Host != "" {
		hm.stats.RequestsByHost[event.Host]++
	}
	hm.stats.LastRequestTime = time.Now()
	hm.mu.Unlock()
	
	// Send event to callback
	if hm.eventCallback != nil {
		go hm.eventCallback(event)
	}
	
	if hm.config.VerboseLogging {
		log.Printf("🌐 HTTP Event: %s %s %s (Claude API: %t)", 
			event.Method, event.Host, event.URL, event.IsClaudeAPI)
	}
}

/**
 * CONTEXT:   Check if host/URL is Claude/Anthropic endpoint
 * INPUT:     Host or URL string
 * OUTPUT:    Boolean indicating if endpoint is Claude API
 * BUSINESS:  Claude endpoint detection enables API usage classification
 * CHANGE:    Initial Claude endpoint detection with pattern matching
 * RISK:      Low - Pattern matching for endpoint classification
 */
func (hm *HTTPMonitor) isClaudeEndpoint(host string) bool {
	if host == "" {
		return false
	}
	
	for _, pattern := range hm.claudeEndpoints {
		if pattern.MatchString(host) {
			return true
		}
	}
	
	return false
}

// Helper methods

func (hm *HTTPMonitor) canUseTcpdump() bool {
	_, err := exec.LookPath("tcpdump")
	return err == nil
}

func (hm *HTTPMonitor) buildPortFilter() string {
	portStrs := make([]string, len(hm.httpPorts))
	for i, port := range hm.httpPorts {
		portStrs[i] = fmt.Sprintf("port %d", port)
	}
	return strings.Join(portStrs, " or ")
}

func (hm *HTTPMonitor) getTCPConnectionsByInodes(inodes map[string]bool, pid int) []map[string]interface{} {
	// Implementation similar to process-check-prototype.go
	return []map[string]interface{}{}
}

func (hm *HTTPMonitor) getProcessName(pid int) string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	if proc, exists := hm.trackedProcesses[pid]; exists {
		return proc.Command
	}
	return fmt.Sprintf("PID_%d", pid)
}

func (hm *HTTPMonitor) getProcessProjectPath(pid int) string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	if proc, exists := hm.trackedProcesses[pid]; exists {
		return proc.WorkingDir
	}
	return ""
}

func (hm *HTTPMonitor) getProcessProjectName(pid int) string {
	projectPath := hm.getProcessProjectPath(pid)
	if projectPath != "" {
		parts := strings.Split(projectPath, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" {
				return parts[i]
			}
		}
	}
	return "unknown"
}

/**
 * CONTEXT:   Get HTTP monitoring statistics
 * INPUT:     Statistics request
 * OUTPUT:    Current HTTP monitoring metrics
 * BUSINESS:  Statistics provide HTTP monitoring system health and API usage insights
 * CHANGE:    Initial HTTP monitoring statistics getter
 * RISK:      Low - Read-only statistics access
 */
func (hm *HTTPMonitor) GetStats() HTTPStats {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	
	// Create a copy to prevent concurrent access issues
	stats := hm.stats
	stats.RequestsByMethod = make(map[string]uint64)
	stats.RequestsByHost = make(map[string]uint64)
	stats.RequestsByStatus = make(map[string]uint64)
	
	for k, v := range hm.stats.RequestsByMethod {
		stats.RequestsByMethod[k] = v
	}
	for k, v := range hm.stats.RequestsByHost {
		stats.RequestsByHost[k] = v
	}
	for k, v := range hm.stats.RequestsByStatus {
		stats.RequestsByStatus[k] = v
	}
	
	return stats
}

/**
 * CONTEXT:   Get process information for HTTP monitoring
 * INPUT:     Process ID
 * OUTPUT:    Process information structure
 * BUSINESS:  Process info enables targeted HTTP monitoring
 * CHANGE:    Process information extraction for HTTP monitoring
 * RISK:      Low - Process information utility for HTTP monitor
 */
func (hm *HTTPMonitor) getProcessInfo(pid int) (*ProcessInfo, error) {
	// Read process command
	cmdlineFile := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmdlineBytes, err := os.ReadFile(cmdlineFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cmdline for PID %d: %w", pid, err)
	}
	
	// Parse command line (null-separated)
	cmdline := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
	cmdline = strings.TrimSpace(cmdline)
	
	parts := strings.Fields(cmdline)
	var command string
	var args []string
	
	if len(parts) > 0 {
		command = filepath.Base(parts[0])
		args = parts[1:]
	}
	
	// Read working directory
	cwdFile := fmt.Sprintf("/proc/%d/cwd", pid)
	workingDir, err := os.Readlink(cwdFile)
	if err != nil {
		workingDir = "" // Non-fatal
	}
	
	return &ProcessInfo{
		PID:        pid,
		Command:    command,
		Args:       args,
		WorkingDir: workingDir,
	}, nil
}

/**
 * CONTEXT:   Advanced /proc/net/tcp monitoring from https-system-detector
 * INPUT:     Process connections via /proc filesystem analysis
 * OUTPUT:    HTTP connection events with detailed tracking
 * BUSINESS:  /proc/net monitoring provides reliable connection tracking without sudo
 * CHANGE:    Advanced /proc/net monitoring implementation from https-system-detector
 * RISK:      Medium - File system monitoring affecting performance
 */
func (hm *HTTPMonitor) startProcNetMonitor() {
	stop := make(chan bool)
	hm.stopChannels["procnet"] = stop
	
	hm.wg.Add(1)
	go func() {
		defer hm.wg.Done()
		hm.monitorProcNetConnections(stop)
	}()
}

func (hm *HTTPMonitor) monitorProcNetConnections(stop chan bool) {
	ticker := time.NewTicker(time.Duration(hm.config.PollingInterval) * time.Millisecond)
	defer ticker.Stop()
	
	lastConnections := make(map[string]*ConnectionInfo)
	
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			hm.mu.RLock()
			trackedPIDs := make(map[int]*ProcessInfo)
			for pid, info := range hm.trackedProcesses {
				trackedPIDs[pid] = info
			}
			hm.mu.RUnlock()
			
			// Get connections for all tracked processes
			connections := hm.getProcessConnections(trackedPIDs)
			
			// Process connection changes
			hm.processConnectionChanges(connections, lastConnections)
			
			lastConnections = connections
		}
	}
}

func (hm *HTTPMonitor) getProcessConnections(processes map[int]*ProcessInfo) map[string]*ConnectionInfo {
	connections := make(map[string]*ConnectionInfo)
	
	for pid, processInfo := range processes {
		// Get socket inodes for this process
		socketInodes := hm.getSocketInodes(pid)
		if len(socketInodes) == 0 {
			continue
		}
		
		// Read TCP connections
		hm.readTCPConnections(socketInodes, connections, "/proc/net/tcp", processInfo)
		hm.readTCPConnections(socketInodes, connections, "/proc/net/tcp6", processInfo)
		
		// Enrich with I/O statistics
		hm.enrichConnectionsWithIOStats(connections, pid)
	}
	
	return connections
}

func (hm *HTTPMonitor) getSocketInodes(pid int) map[string]bool {
	socketInodes := make(map[string]bool)
	
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	files, err := os.ReadDir(fdDir)
	if err != nil {
		return socketInodes
	}
	
	for _, file := range files {
		fdPath := filepath.Join(fdDir, file.Name())
		target, err := os.Readlink(fdPath)
		if err != nil {
			continue
		}
		
		if strings.HasPrefix(target, "socket:[") {
			inode := strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")
			socketInodes[inode] = true
		}
	}
	
	return socketInodes
}

func (hm *HTTPMonitor) readTCPConnections(inodes map[string]bool, connections map[string]*ConnectionInfo, 
	netFile string, processInfo *ProcessInfo) {
	
	data, err := os.ReadFile(netFile)
	if err != nil {
		return
	}
	
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header
		}
		
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		
		inode := fields[9]
		if !inodes[inode] {
			continue
		}
		
		localAddr := hm.parseHexAddress(fields[1])
		remoteAddr := hm.parseHexAddress(fields[2])
		state := hm.getTCPState(fields[3])
		
		// Only monitor established connections to HTTP/HTTPS ports
		if state != "ESTABLISHED" {
			continue
		}
		
		host, port := hm.splitAddress(remoteAddr)
		if !hm.isHTTPPort(port) {
			continue
		}
		
		// Determine protocol by port
		protocol := "HTTP"
		if port == 443 || port == 8443 {
			protocol = "HTTPS"
		}
		
		connKey := fmt.Sprintf("%s->%s", localAddr, remoteAddr)
		
		connections[connKey] = &ConnectionInfo{
			StartTime:   time.Now(),
			Host:        host,
			Port:        port,
			Protocol:    protocol,
			ProcessName: processInfo.Command,
			PID:         processInfo.PID,
			LocalAddr:   localAddr,
			RemoteAddr:  remoteAddr,
			State:       state,
		}
		
		// Extract byte counters if available
		if len(fields) >= 5 {
			queues := strings.Split(fields[4], ":")
			if len(queues) == 2 {
				txQueue, _ := strconv.ParseInt(queues[0], 16, 64)
				rxQueue, _ := strconv.ParseInt(queues[1], 16, 64)
				connections[connKey].BytesSent = txQueue
				connections[connKey].BytesRecv = rxQueue
			}
		}
	}
}

func (hm *HTTPMonitor) enrichConnectionsWithIOStats(connections map[string]*ConnectionInfo, pid int) {
	// Read I/O statistics for the process
	ioFile := fmt.Sprintf("/proc/%d/io", pid)
	data, err := os.ReadFile(ioFile)
	if err != nil {
		return
	}
	
	stats := hm.parseProcIO(string(data))
	
	// Distribute I/O among active connections (approximation)
	if len(connections) > 0 && stats["write_bytes"] > 0 {
		bytesPerConn := stats["write_bytes"] / int64(len(connections))
		for _, conn := range connections {
			if conn.BytesSent == 0 {
				conn.BytesSent = bytesPerConn
			}
		}
	}
}

func (hm *HTTPMonitor) processConnectionChanges(current, previous map[string]*ConnectionInfo) {
	// Detect new connections
	for key, conn := range current {
		if _, exists := previous[key]; !exists {
			// New connection
			hm.connectionTracker.AddConnection(key, conn)
			
			if hm.config.VerboseLogging {
				log.Printf("🌐 New %s connection to %s:%d (PID: %d)", 
					conn.Protocol, conn.Host, conn.Port, conn.PID)
			}
		} else {
			// Update existing connection
			hm.connectionTracker.UpdateBytes(key, conn.BytesSent, conn.BytesRecv)
		}
	}
	
	// Detect closed connections
	for key, conn := range previous {
		if _, exists := current[key]; !exists {
			// Connection closed - generate event
			hm.connectionTracker.CloseConnection(key, conn)
		}
	}
}

/**
 * CONTEXT:   Socket statistics monitoring using ss command from https-system-detector
 * INPUT:     Socket statistics via ss command execution
 * OUTPUT:    Enhanced connection information with detailed byte statistics
 * BUSINESS:  SS monitoring provides detailed socket statistics for precise activity tracking
 * CHANGE:    SS monitoring implementation from https-system-detector
 * RISK:      Medium - External command execution affecting performance
 */
func (hm *HTTPMonitor) startSSMonitor() {
	stop := make(chan bool)
	hm.stopChannels["ss"] = stop
	
	hm.wg.Add(1)
	go func() {
		defer hm.wg.Done()
		hm.monitorWithSS(stop)
	}()
}

func (hm *HTTPMonitor) monitorWithSS(stop chan bool) {
	ticker := time.NewTicker(time.Second) // SS monitoring every second
	defer ticker.Stop()
	
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			hm.executeSSCommand()
		}
	}
}

func (hm *HTTPMonitor) executeSSCommand() {
	// Use ss to get detailed socket information
	cmd := exec.Command("ss", "-tnp", "-o", "state", "established")
	output, err := cmd.Output()
	if err != nil {
		return
	}
	
	hm.parseSSOutput(string(output))
}

func (hm *HTTPMonitor) parseSSOutput(output string) {
	lines := strings.Split(output, "\n")
	
	hm.mu.RLock()
	trackedPIDs := make(map[int]bool)
	for pid := range hm.trackedProcesses {
		trackedPIDs[pid] = true
	}
	hm.mu.RUnlock()
	
	for _, line := range lines {
		if !strings.Contains(line, "ESTAB") {
			continue
		}
		
		// Look for our tracked PIDs
		var matchedPID int
		for pid := range trackedPIDs {
			pidStr := fmt.Sprintf("pid=%d", pid)
			if strings.Contains(line, pidStr) {
				matchedPID = pid
				break
			}
		}
		
		if matchedPID == 0 {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		
		localAddr := fields[3]
		remoteAddr := fields[4]
		
		// Extract byte information if available
		var bytesSent, bytesRecv int64
		if strings.Contains(line, "bytes_sent:") {
			re := regexp.MustCompile(`bytes_sent:(\d+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				bytesSent, _ = strconv.ParseInt(matches[1], 10, 64)
			}
		}
		
		if strings.Contains(line, "bytes_received:") {
			re := regexp.MustCompile(`bytes_received:(\d+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				bytesRecv, _ = strconv.ParseInt(matches[1], 10, 64)
			}
		}
		
		host, port := hm.splitAddress(remoteAddr)
		if !hm.isHTTPPort(port) {
			continue
		}
		
		connKey := fmt.Sprintf("%s->%s", localAddr, remoteAddr)
		hm.connectionTracker.UpdateConnection(connKey, &ConnectionInfo{
			Host:      host,
			Port:      port,
			BytesSent: bytesSent,
			BytesRecv: bytesRecv,
			PID:       matchedPID,
		})
	}
}

// Helper methods from https-system-detector.go
func (hm *HTTPMonitor) canUseSS() bool {
	_, err := exec.LookPath("ss")
	return err == nil
}

func (hm *HTTPMonitor) parseHexAddress(hex string) string {
	parts := strings.Split(hex, ":")
	if len(parts) != 2 {
		return hex
	}
	
	ip, _ := strconv.ParseUint(parts[0], 16, 32)
	port, _ := strconv.ParseUint(parts[1], 16, 16)
	
	return fmt.Sprintf("%d.%d.%d.%d:%d",
		ip&0xFF, (ip>>8)&0xFF, (ip>>16)&0xFF, (ip>>24)&0xFF, port)
}

func (hm *HTTPMonitor) getTCPState(hex string) string {
	states := map[string]string{
		"01": "ESTABLISHED", "02": "SYN_SENT", "03": "SYN_RECV",
		"04": "FIN_WAIT1", "05": "FIN_WAIT2", "06": "TIME_WAIT",
		"07": "CLOSE", "08": "CLOSE_WAIT", "09": "LAST_ACK",
		"0A": "LISTEN", "0B": "CLOSING",
	}
	
	if state, ok := states[hex]; ok {
		return state
	}
	return hex
}

func (hm *HTTPMonitor) splitAddress(addr string) (string, int) {
	lastColon := strings.LastIndex(addr, ":")
	if lastColon == -1 {
		return addr, 0
	}
	
	host := addr[:lastColon]
	portStr := addr[lastColon+1:]
	port, _ := strconv.Atoi(portStr)
	
	return host, port
}

func (hm *HTTPMonitor) isHTTPPort(port int) bool {
	for _, p := range hm.httpPorts {
		if p == port {
			return true
		}
	}
	return false
}

func (hm *HTTPMonitor) parseProcIO(data string) map[string]int64 {
	stats := make(map[string]int64)
	lines := strings.Split(data, "\n")
	
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
			stats[key] = value
		}
	}
	
	return stats
}

func (hm *HTTPMonitor) getEnabledMethodsCount() int {
	count := 0
	if hm.config.UseProcNet { count++ }
	if hm.config.UseSSMonitor && hm.canUseSS() { count++ }
	if hm.config.UseTcpdump && hm.canUseTcpdump() { count++ }
	if hm.config.UseProxy { count++ }
	return count
}

/**
 * CONTEXT:   tcpdump packet capture monitoring from https-system-detector
 * INPUT:     Network packet capture for comprehensive traffic analysis
 * OUTPUT:    HTTP events from packet analysis with detailed timing
 * BUSINESS:  tcpdump monitoring provides comprehensive packet-level HTTP analysis (requires sudo)
 * CHANGE:    tcpdump monitoring implementation from https-system-detector
 * RISK:      High - Packet capture affecting network performance and requiring sudo
 */
func (hm *HTTPMonitor) startTcpdumpMonitor() {
	stop := make(chan bool)
	hm.stopChannels["tcpdump"] = stop
	
	hm.wg.Add(1)
	go func() {
		defer hm.wg.Done()
		hm.monitorWithTcpdumpAdvanced(stop)
	}()
}

func (hm *HTTPMonitor) monitorWithTcpdumpAdvanced(stop chan bool) {
	// Build comprehensive port filter
	portFilter := hm.buildPortFilter()
	
	// Enhanced tcpdump command for HTTP/HTTPS monitoring
	cmd := exec.Command("tcpdump", "-i", "any", "-A", "-s", "1500",
		fmt.Sprintf("tcp and (%s)", portFilter), "-l")
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create tcpdump stdout pipe: %v", err)
		return
	}
	
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start tcpdump (requires sudo): %v", err)
		return
	}
	
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()
	
	scanner := bufio.NewScanner(stdout)
	var packetBuffer strings.Builder
	inPacket := false
	
	for {
		select {
		case <-stop:
			return
		default:
			if !scanner.Scan() {
				return
			}
			
			line := scanner.Text()
			
			// Detect packet boundaries
			if strings.Contains(line, " > ") && hm.isHTTPTrafficLine(line) {
				// Process previous packet if exists
				if packetBuffer.Len() > 0 {
					hm.parseHTTPPacketAdvanced(packetBuffer.String())
				}
				
				packetBuffer.Reset()
				inPacket = true
			}
			
			if inPacket {
				packetBuffer.WriteString(line + "\n")
			}
		}
	}
}

func (hm *HTTPMonitor) isHTTPTrafficLine(line string) bool {
	for _, port := range hm.httpPorts {
		portStr := fmt.Sprintf(".%d ", port)
		if strings.Contains(line, portStr) {
			return true
		}
	}
	return false
}

func (hm *HTTPMonitor) parseHTTPPacketAdvanced(packet string) {
	lines := strings.Split(packet, "\n")
	if len(lines) < 2 {
		return
	}
	
	// Extract connection info from first line
	connLine := lines[0]
	srcAddr, dstAddr := hm.extractAddressesFromConnLine(connLine)
	
	// Look for HTTP patterns in packet data
	for _, line := range lines[1:] {
		if hm.parseHTTPRequestAdvanced(line, srcAddr, dstAddr) {
			break
		}
	}
}

func (hm *HTTPMonitor) extractAddressesFromConnLine(line string) (string, string) {
	// Parse tcpdump connection line format
	parts := strings.Fields(line)
	if len(parts) >= 5 {
		return parts[2], parts[4]
	}
	return "", ""
}

func (hm *HTTPMonitor) parseHTTPRequestAdvanced(line, srcAddr, dstAddr string) bool {
	// Enhanced HTTP request parsing for both GET and POST
	if !strings.Contains(line, "GET ") && !strings.Contains(line, "POST ") && 
	   !strings.Contains(line, "PUT ") && !strings.Contains(line, "DELETE ") {
		return false
	}
	
	// Extract method and URL
	parts := strings.Fields(line)
	var method, url string
	
	for i, part := range parts {
		if part == "GET" || part == "POST" || part == "PUT" || part == "DELETE" {
			method = part
			if i+1 < len(parts) {
				url = parts[i+1]
			}
			break
		}
	}
	
	if method == "" {
		return false
	}
	
	// Extract host from subsequent lines or addresses
	host := hm.extractHostFromAddr(dstAddr)
	port := hm.extractPortFromAddr(dstAddr)
	
	// Only process if it's an HTTP port we're monitoring
	if !hm.isHTTPPort(port) {
		return false
	}
	
	// Create HTTP event from packet analysis
	event := HTTPEvent{
		Type:        HTTPRequest,
		Timestamp:   time.Now(),
		Method:      method,
		URL:         url,
		Host:        host,
		Port:        port,
		Protocol:    hm.getProtocolByPort(port),
		IsClaudeAPI: hm.isClaudeEndpoint(host),
		Details: map[string]interface{}{
			"source":           "tcpdump",
			"src_addr":         srcAddr,
			"dst_addr":         dstAddr,
			"packet_capture":   true,
		},
	}
	
	// Process the event
	if hm.eventCallback != nil {
		hm.eventCallback(event)
	}
	
	if hm.config.VerboseLogging {
		log.Printf("📦 tcpdump: %s %s %s:%d (Claude: %t)", 
			method, url, host, port, event.IsClaudeAPI)
	}
	
	return true
}

func (hm *HTTPMonitor) extractHostFromAddr(addr string) string {
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

func (hm *HTTPMonitor) extractPortFromAddr(addr string) int {
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		portStr := addr[idx+1:]
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}
	return 80
}

func (hm *HTTPMonitor) getProtocolByPort(port int) string {
	if port == 443 || port == 8443 {
		return "HTTPS"
	}
	return "HTTP"
}

/**
 * CONTEXT:   Connection tracker methods from https-system-detector for lifecycle management
 * INPUT:     Connection tracking operations and event generation
 * OUTPUT:    HTTP events from connection lifecycle changes
 * BUSINESS:  Connection tracking enables precise HTTP activity detection with intelligent method inference
 * CHANGE:    Connection tracker methods implementation from https-system-detector
 * RISK:      Low - Connection tracking utility methods
 */
func (ct *ConnectionTracker) AddConnection(key string, conn *ConnectionInfo) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	ct.connections[key] = conn
}

func (ct *ConnectionTracker) UpdateConnection(key string, update *ConnectionInfo) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	if conn, exists := ct.connections[key]; exists {
		if update.BytesSent > 0 {
			conn.BytesSent = update.BytesSent
		}
		if update.BytesRecv > 0 {
			conn.BytesRecv = update.BytesRecv
		}
	}
}

func (ct *ConnectionTracker) UpdateBytes(key string, sent, recv int64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	if conn, exists := ct.connections[key]; exists {
		conn.BytesSent = sent
		conn.BytesRecv = recv
	}
}

func (ct *ConnectionTracker) CloseConnection(key string, conn *ConnectionInfo) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	// Calculate connection duration
	duration := time.Since(conn.StartTime).Milliseconds()
	
	// Infer HTTP method from traffic pattern (key insight from https-system-detector)
	method := "GET" // Default
	if conn.BytesSent > conn.BytesRecv {
		method = "POST/PUT" // More sent than received - likely uploading data
	}
	
	// Generate HTTP event
	event := HTTPEvent{
		Timestamp:   time.Now(),
		Method:      method,
		Host:        conn.Host,
		Port:        conn.Port,
		Protocol:    conn.Protocol,
		BytesSent:   conn.BytesSent,
		BytesRecv:   conn.BytesRecv,
		Duration:    duration,
		ProcessName: conn.ProcessName,
		PID:         conn.PID,
		IsClaudeAPI: ct.isClaudeEndpoint(conn.Host),
		ProjectPath: "", // Will be filled by caller if available
		ProjectName: "", // Will be filled by caller if available
		Details: map[string]interface{}{
			"local_addr":        conn.LocalAddr,
			"remote_addr":       conn.RemoteAddr,
			"connection_state":  conn.State,
			"method_inferred":   true,
			"traffic_pattern":   ct.getTrafficPattern(conn),
		},
	}
	
	// Send event to channel (non-blocking)
	select {
	case ct.events <- event:
	default:
		// Channel full - skip this event to prevent blocking
	}
	
	// Remove connection from tracking
	delete(ct.connections, key)
	
	// Print connection summary (like https-system-detector.go)
	ct.printConnectionSummary(event)
}

func (ct *ConnectionTracker) GetEvents() <-chan HTTPEvent {
	return ct.events
}

func (ct *ConnectionTracker) isClaudeEndpoint(host string) bool {
	// Claude/Anthropic endpoint patterns (includes both domains and IPs)
	claudePatterns := []string{
		"api.anthropic.com", "claude.ai", "api.claude.ai", "anthropic.com",
		"claude-dev.anthropic.com", "staging.anthropic.com", "beta.claude.ai",
		// Common Claude API IP addresses
		"34.36.57.103", "160.79.104.10", "34.102.136.180", "35.227.210.155",
	}
	
	hostLower := strings.ToLower(host)
	for _, pattern := range claudePatterns {
		if strings.Contains(hostLower, pattern) {
			return true
		}
	}
	return false
}

func (ct *ConnectionTracker) getTrafficPattern(conn *ConnectionInfo) string {
	if conn.BytesSent > conn.BytesRecv*2 {
		return "upload_heavy"
	} else if conn.BytesRecv > conn.BytesSent*2 {
		return "download_heavy"
	}
	return "balanced"
}

func (ct *ConnectionTracker) printConnectionSummary(event HTTPEvent) {
	// Beautiful output formatting like https-system-detector.go
	emoji := "🌐"
	if event.Protocol == "HTTPS" {
		emoji = "🔒"
	}
	if event.IsClaudeAPI {
		emoji = "🔥" // Special emoji for Claude API calls
	}
	
	methodColor := "\033[32m" // Green for GET
	if strings.Contains(event.Method, "POST") {
		methodColor = "\033[33m" // Yellow for POST
	}
	
	// Print connection summary
	log.Printf("%s [%s] %s%s\033[0m %s:%d", 
		emoji,
		event.Timestamp.Format("15:04:05"),
		methodColor,
		event.Method,
		event.Host,
		event.Port)
	
	log.Printf("  ⬆️  Sent: %s", ct.formatBytes(event.BytesSent))
	log.Printf("  ⬇️  Received: %s", ct.formatBytes(event.BytesRecv))
	log.Printf("  ⏱️  Duration: %dms", event.Duration)
	log.Printf("  📱 Process: %s (PID: %d)", event.ProcessName, event.PID)
	
	if event.IsClaudeAPI {
		log.Printf("  🎯 Claude API Activity Detected!")
	}
	log.Println() // Empty line for readability
}

func (ct *ConnectionTracker) formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

/**
 * CONTEXT:   Process connection events and forward to main event callback
 * INPUT:     HTTP events from connection tracker
 * OUTPUT:    Processed HTTP events with enhanced statistics
 * BUSINESS:  Event processing enables comprehensive HTTP activity tracking with statistics
 * CHANGE:    Connection event processing for advanced monitoring integration
 * RISK:      Low - Event processing with statistics tracking
 */
func (hm *HTTPMonitor) processConnectionEvents() {
	for event := range hm.connectionTracker.GetEvents() {
		// Enhance event with process information
		hm.mu.RLock()
		if processInfo, exists := hm.trackedProcesses[event.PID]; exists {
			event.ProjectPath = processInfo.WorkingDir
			event.ProjectName = hm.getProcessProjectName(event.PID)
		}
		hm.mu.RUnlock()
		
		// Update statistics
		hm.mu.Lock()
		hm.stats.TotalRequests++
		hm.stats.TotalConnections++
		
		if event.Protocol == "HTTPS" {
			hm.stats.HTTPSConnections++
		} else {
			hm.stats.HTTPConnections++
		}
		
		if event.IsClaudeAPI {
			hm.stats.ClaudeAPIRequests++
		}
		
		// Track inferred methods
		if hm.stats.MethodsInferred == nil {
			hm.stats.MethodsInferred = make(map[string]uint64)
		}
		hm.stats.MethodsInferred[event.Method]++
		hm.stats.RequestsByMethod[event.Method]++
		hm.stats.RequestsByHost[event.Host]++
		
		hm.stats.TotalDataSent += event.BytesSent
		hm.stats.TotalDataReceived += event.BytesRecv
		hm.stats.LastRequestTime = event.Timestamp
		
		// Update average connection time
		if hm.stats.TotalConnections > 0 {
			totalTime := time.Duration(hm.stats.AverageConnTime.Nanoseconds()*int64(hm.stats.TotalConnections-1) + event.Duration*int64(time.Millisecond))
			hm.stats.AverageConnTime = time.Duration(totalTime.Nanoseconds() / int64(hm.stats.TotalConnections))
		}
		hm.mu.Unlock()
		
		// Forward to main callback
		if hm.eventCallback != nil {
			hm.eventCallback(event)
		}
	}
}