/**
 * CONTEXT:   Network monitoring for Claude process connections and traffic analysis
 * INPUT:     Network connection monitoring and traffic analysis for Claude processes
 * OUTPUT:    Network activity events for connectivity and usage tracking
 * BUSINESS:  Network monitoring provides comprehensive Claude connectivity insights
 * CHANGE:    Initial network monitor based on prototype with Claude-specific analysis
 * RISK:      Medium - Network monitoring affecting system performance
 */

package monitor

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
 * CONTEXT:   Network event types for Claude connectivity classification
 * INPUT:     Network connection and traffic analysis
 * OUTPUT:    Classified network events for connectivity tracking
 * BUSINESS:  Network event classification enables connectivity analytics
 * CHANGE:    Initial network event types for Claude connectivity monitoring
 * RISK:      Low - Event type definitions for network operations
 */
type NetworkEventType string

const (
	NetworkConnect     NetworkEventType = "NETWORK_CONNECT"
	NetworkDisconnect  NetworkEventType = "NETWORK_DISCONNECT"
	NetworkTraffic     NetworkEventType = "NETWORK_TRAFFIC"
	NetworkClaudeAPI   NetworkEventType = "NETWORK_CLAUDE_API"
	NetworkError       NetworkEventType = "NETWORK_ERROR"
)

type NetworkEvent struct {
	Type         NetworkEventType       `json:"type"`
	Timestamp    time.Time              `json:"timestamp"`
	PID          int                    `json:"pid"`
	ProcessName  string                 `json:"process_name"`
	Protocol     string                 `json:"protocol"`     // TCP, UDP
	LocalAddr    string                 `json:"local_addr"`   // Local IP:Port
	RemoteAddr   string                 `json:"remote_addr"`  // Remote IP:Port
	RemoteHost   string                 `json:"remote_host"`  // Resolved hostname
	State        string                 `json:"state"`        // CONNECTION_STATE
	BytesSent    int64                  `json:"bytes_sent"`
	BytesRecv    int64                  `json:"bytes_recv"`
	IsClaudeAPI  bool                   `json:"is_claude_api"`
	ProjectPath  string                 `json:"project_path"`
	ProjectName  string                 `json:"project_name"`
	Details      map[string]interface{} `json:"details"`
}

/**
 * CONTEXT:   Network connection state tracking
 * INPUT:     Network connection state information
 * OUTPUT:    Connection state representation
 * BUSINESS:  Connection state tracking enables connection lifecycle monitoring
 * CHANGE:    Initial connection state tracking for network analysis
 * RISK:      Low - Connection state data structure
 */
type ConnectionState struct {
	PID          int       `json:"pid"`
	Protocol     string    `json:"protocol"`
	LocalAddr    string    `json:"local_addr"`
	RemoteAddr   string    `json:"remote_addr"`
	RemoteHost   string    `json:"remote_host"`
	State        string    `json:"state"`
	StartTime    time.Time `json:"start_time"`
	LastSeen     time.Time `json:"last_seen"`
	BytesSent    int64     `json:"bytes_sent"`
	BytesRecv    int64     `json:"bytes_recv"`
	IsClaudeAPI  bool      `json:"is_claude_api"`
}

/**
 * CONTEXT:   Network monitor for Claude process connections
 * INPUT:     Network connection monitoring for tracked Claude processes
 * OUTPUT:    Network events for connectivity and traffic analysis
 * BUSINESS:  Network monitoring provides comprehensive Claude connectivity tracking
 * CHANGE:    Initial network monitor with connection and traffic analysis
 * RISK:      Medium - Network monitoring affecting system performance
 */
type NetworkMonitor struct {
	ctx               context.Context
	cancel            context.CancelFunc
	eventCallback     func(NetworkEvent)
	trackedProcesses  map[int]*ProcessInfo
	connections       map[string]*ConnectionState // Key: "protocol:local:remote"
	claudeEndpoints   []*regexp.Regexp
	mu                sync.RWMutex
	running           bool
	stats             NetworkStats
	config            NetworkMonitorConfig
}

/**
 * CONTEXT:   Network monitoring configuration
 * INPUT:     Network monitoring parameters and settings
 * OUTPUT:    Network monitor behavior configuration
 * BUSINESS:  Configuration enables customizable network monitoring
 * CHANGE:    Initial network monitoring configuration
 * RISK:      Low - Configuration structure for network monitoring
 */
type NetworkMonitorConfig struct {
	MonitorTCP       bool          `json:"monitor_tcp"`        // Monitor TCP connections
	MonitorUDP       bool          `json:"monitor_udp"`        // Monitor UDP connections
	ScanInterval     time.Duration `json:"scan_interval"`      // Connection scan interval
	TrafficAnalysis  bool          `json:"traffic_analysis"`   // Analyze traffic patterns
	ResolveHostnames bool          `json:"resolve_hostnames"`  // Resolve IP addresses to hostnames
	VerboseLogging   bool          `json:"verbose_logging"`    // Enable verbose logging
}

/**
 * CONTEXT:   Network monitoring statistics
 * INPUT:     Network monitoring metrics and connectivity data
 * OUTPUT:    Performance and connectivity statistics
 * BUSINESS:  Statistics enable network monitoring optimization and connectivity insights
 * CHANGE:    Initial network monitoring statistics
 * RISK:      Low - Statistics structure for network monitoring metrics
 */
type NetworkStats struct {
	TotalConnections    uint64            `json:"total_connections"`
	ActiveConnections   uint64            `json:"active_connections"`
	ClaudeAPIConnections uint64           `json:"claude_api_connections"`
	ConnectionsByState  map[string]uint64 `json:"connections_by_state"`
	ConnectionsByHost   map[string]uint64 `json:"connections_by_host"`
	TotalTrafficSent    int64             `json:"total_traffic_sent"`
	TotalTrafficRecv    int64             `json:"total_traffic_recv"`
	StartTime           time.Time         `json:"start_time"`
	LastScanTime        time.Time         `json:"last_scan_time"`
	ErrorCount          uint64            `json:"error_count"`
}

/**
 * CONTEXT:   Create new network monitor for Claude connectivity tracking
 * INPUT:     Event callback and network monitoring configuration
 * OUTPUT:    Configured network monitor ready for activation
 * BUSINESS:  Network monitor creation sets up Claude connectivity tracking
 * CHANGE:    Initial network monitor constructor with Claude endpoint detection
 * RISK:      Medium - Network monitor initialization affecting network monitoring
 */
func NewNetworkMonitor(callback func(NetworkEvent), config NetworkMonitorConfig) *NetworkMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Set default configuration
	if config.ScanInterval == 0 {
		config.ScanInterval = 5 * time.Second
	}
	
	// Claude/Anthropic endpoints for API connection detection
	claudeEndpoints := []*regexp.Regexp{
		regexp.MustCompile(`(?i).*api\.anthropic\.com.*`),
		regexp.MustCompile(`(?i).*claude\.ai.*`),
		regexp.MustCompile(`(?i).*anthropic\.com.*`),
		regexp.MustCompile(`(?i).*claude-api\.anthropic\.com.*`),
	}
	
	monitor := &NetworkMonitor{
		ctx:              ctx,
		cancel:           cancel,
		eventCallback:    callback,
		trackedProcesses: make(map[int]*ProcessInfo),
		connections:      make(map[string]*ConnectionState),
		claudeEndpoints:  claudeEndpoints,
		config:           config,
		stats: NetworkStats{
			ConnectionsByState: make(map[string]uint64),
			ConnectionsByHost:  make(map[string]uint64),
			StartTime:          time.Now(),
		},
	}
	
	return monitor
}

/**
 * CONTEXT:   Start network monitoring for Claude connectivity tracking
 * INPUT:     Monitor activation request
 * OUTPUT:    Active network monitoring with connection tracking
 * BUSINESS:  Network monitor start enables real-time Claude connectivity tracking
 * CHANGE:    Initial network monitor start with connection scanning
 * RISK:      Medium - Network monitoring affecting system performance
 */
func (nm *NetworkMonitor) Start() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	
	if nm.running {
		return fmt.Errorf("network monitor is already running")
	}
	
	nm.running = true
	
	// Start monitoring goroutine
	go nm.monitorLoop()
	
	log.Printf("Network monitor started (TCP: %t, UDP: %t, Interval: %v)", 
		nm.config.MonitorTCP, nm.config.MonitorUDP, nm.config.ScanInterval)
	return nil
}

/**
 * CONTEXT:   Stop network monitoring system
 * INPUT:     Monitor shutdown request
 * OUTPUT:    Cleanly stopped network monitoring
 * BUSINESS:  Network monitor stop enables graceful shutdown
 * CHANGE:    Initial network monitor stop implementation
 * RISK:      Medium - Monitor shutdown affecting tracked connections
 */
func (nm *NetworkMonitor) Stop() error {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	
	if !nm.running {
		return nil
	}
	
	nm.cancel()
	nm.running = false
	
	log.Printf("Network monitor stopped")
	return nil
}

/**
 * CONTEXT:   Update tracked processes for network monitoring
 * INPUT:     Map of Claude processes to monitor
 * OUTPUT:    Updated network monitoring targets
 * BUSINESS:  Process tracking enables selective network monitoring
 * CHANGE:    Initial process tracking update for network monitoring
 * RISK:      Low - Process list update with synchronization
 */
func (nm *NetworkMonitor) UpdateTrackedProcesses(processes map[int]*ProcessInfo) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	
	nm.trackedProcesses = make(map[int]*ProcessInfo)
	for pid, info := range processes {
		nm.trackedProcesses[pid] = info
	}
	
	if nm.config.VerboseLogging {
		log.Printf("Network monitor tracking %d Claude processes", len(nm.trackedProcesses))
	}
}

/**
 * CONTEXT:   Main network monitoring loop
 * INPUT:     Continuous network connection scanning
 * OUTPUT:    Network events for connection changes
 * BUSINESS:  Monitor loop provides continuous Claude connectivity tracking
 * CHANGE:    Initial network monitoring loop with connection scanning
 * RISK:      Medium - Continuous monitoring affecting system performance
 */
func (nm *NetworkMonitor) monitorLoop() {
	ticker := time.NewTicker(nm.config.ScanInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			if err := nm.scanConnections(); err != nil {
				nm.mu.Lock()
				nm.stats.ErrorCount++
				nm.mu.Unlock()
				
				if nm.config.VerboseLogging {
					log.Printf("Network scan error: %v", err)
				}
			}
		}
	}
}

/**
 * CONTEXT:   Scan network connections for tracked processes
 * INPUT:     Process connection files from /proc filesystem
 * OUTPUT:    Updated connection tracking and generated events
 * BUSINESS:  Connection scanning detects Claude connectivity changes
 * CHANGE:    Initial connection scanning with process correlation
 * RISK:      Medium - Connection scanning affecting system performance
 */
func (nm *NetworkMonitor) scanConnections() error {
	nm.mu.Lock()
	trackedPIDs := make(map[int]*ProcessInfo)
	for pid, info := range nm.trackedProcesses {
		trackedPIDs[pid] = info
	}
	nm.mu.Unlock()
	
	currentConnections := make(map[string]*ConnectionState)
	
	// Scan TCP connections if configured
	if nm.config.MonitorTCP {
		tcpConns, err := nm.scanTCPConnections(trackedPIDs)
		if err != nil {
			return fmt.Errorf("TCP scan failed: %w", err)
		}
		for k, v := range tcpConns {
			currentConnections[k] = v
		}
	}
	
	// Scan UDP connections if configured
	if nm.config.MonitorUDP {
		udpConns, err := nm.scanUDPConnections(trackedPIDs)
		if err != nil {
			return fmt.Errorf("UDP scan failed: %w", err)
		}
		for k, v := range udpConns {
			currentConnections[k] = v
		}
	}
	
	nm.mu.Lock()
	defer nm.mu.Unlock()
	
	// Detect new connections
	for key, conn := range currentConnections {
		if _, exists := nm.connections[key]; !exists {
			// New connection detected
			nm.generateConnectionEvent(conn, NetworkConnect)
		} else {
			// Update existing connection
			nm.updateConnection(key, conn)
		}
	}
	
	// Detect closed connections
	for key, conn := range nm.connections {
		if _, exists := currentConnections[key]; !exists {
			// Connection closed
			nm.generateConnectionEvent(conn, NetworkDisconnect)
		}
	}
	
	// Update connections map
	nm.connections = currentConnections
	nm.stats.ActiveConnections = uint64(len(currentConnections))
	nm.stats.LastScanTime = time.Now()
	
	return nil
}

/**
 * CONTEXT:   Scan TCP connections for tracked processes
 * INPUT:     Tracked process list for TCP connection analysis
 * OUTPUT:    TCP connections for tracked Claude processes
 * BUSINESS:  TCP connection scanning enables TCP-based Claude connectivity tracking
 * CHANGE:    Initial TCP connection scanning with process correlation
 * RISK:      Medium - TCP scanning affecting system performance
 */
func (nm *NetworkMonitor) scanTCPConnections(trackedPIDs map[int]*ProcessInfo) (map[string]*ConnectionState, error) {
	connections := make(map[string]*ConnectionState)
	
	// Get socket inodes for all tracked processes
	processInodes := make(map[string]int) // inode -> pid
	for pid := range trackedPIDs {
		inodes := nm.getProcessSocketInodes(pid)
		for inode := range inodes {
			processInodes[inode] = pid
		}
	}
	
	// Read /proc/net/tcp
	tcpData, err := ioutil.ReadFile("/proc/net/tcp")
	if err != nil {
		return connections, fmt.Errorf("failed to read /proc/net/tcp: %w", err)
	}
	
	lines := strings.Split(string(tcpData), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header
		}
		
		conn := nm.parseTCPLine(line, processInodes)
		if conn != nil {
			key := nm.generateConnectionKey(conn)
			connections[key] = conn
		}
	}
	
	return connections, nil
}

/**
 * CONTEXT:   Scan UDP connections for tracked processes
 * INPUT:     Tracked process list for UDP connection analysis
 * OUTPUT:    UDP connections for tracked Claude processes
 * BUSINESS:  UDP connection scanning enables UDP-based Claude connectivity tracking
 * CHANGE:    Initial UDP connection scanning with process correlation
 * RISK:      Medium - UDP scanning affecting system performance
 */
func (nm *NetworkMonitor) scanUDPConnections(trackedPIDs map[int]*ProcessInfo) (map[string]*ConnectionState, error) {
	connections := make(map[string]*ConnectionState)
	
	// Get socket inodes for all tracked processes
	processInodes := make(map[string]int) // inode -> pid
	for pid := range trackedPIDs {
		inodes := nm.getProcessSocketInodes(pid)
		for inode := range inodes {
			processInodes[inode] = pid
		}
	}
	
	// Read /proc/net/udp
	udpData, err := ioutil.ReadFile("/proc/net/udp")
	if err != nil {
		return connections, fmt.Errorf("failed to read /proc/net/udp: %w", err)
	}
	
	lines := strings.Split(string(udpData), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header
		}
		
		conn := nm.parseUDPLine(line, processInodes)
		if conn != nil {
			key := nm.generateConnectionKey(conn)
			connections[key] = conn
		}
	}
	
	return connections, nil
}

/**
 * CONTEXT:   Parse TCP connection line from /proc/net/tcp
 * INPUT:     TCP connection line and process inode mapping
 * OUTPUT:    Connection state object if line represents tracked process
 * BUSINESS:  TCP line parsing enables TCP connection tracking
 * CHANGE:    Initial TCP connection parsing with inode correlation
 * RISK:      Medium - Parsing accuracy affecting connection tracking
 */
func (nm *NetworkMonitor) parseTCPLine(line string, processInodes map[string]int) *ConnectionState {
	fields := strings.Fields(line)
	if len(fields) < 10 {
		return nil
	}
	
	inode := fields[9]
	pid, exists := processInodes[inode]
	if !exists {
		return nil // Not a tracked process
	}
	
	localAddr := nm.parseHexAddress(fields[1])
	remoteAddr := nm.parseHexAddress(fields[2])
	state := nm.getTCPState(fields[3])
	
	conn := &ConnectionState{
		PID:        pid,
		Protocol:   "TCP",
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		State:      state,
		StartTime:  time.Now(),
		LastSeen:   time.Now(),
	}
	
	// Resolve hostname if configured
	if nm.config.ResolveHostnames {
		conn.RemoteHost = nm.resolveHostname(strings.Split(remoteAddr, ":")[0])
	}
	
	// Check if this is a Claude API connection
	conn.IsClaudeAPI = nm.isClaudeEndpoint(conn.RemoteHost)
	
	return conn
}

/**
 * CONTEXT:   Parse UDP connection line from /proc/net/udp
 * INPUT:     UDP connection line and process inode mapping
 * OUTPUT:    Connection state object if line represents tracked process
 * BUSINESS:  UDP line parsing enables UDP connection tracking
 * CHANGE:    Initial UDP connection parsing with inode correlation
 * RISK:      Medium - Parsing accuracy affecting connection tracking
 */
func (nm *NetworkMonitor) parseUDPLine(line string, processInodes map[string]int) *ConnectionState {
	fields := strings.Fields(line)
	if len(fields) < 10 {
		return nil
	}
	
	inode := fields[9]
	pid, exists := processInodes[inode]
	if !exists {
		return nil // Not a tracked process
	}
	
	localAddr := nm.parseHexAddress(fields[1])
	remoteAddr := nm.parseHexAddress(fields[2])
	
	conn := &ConnectionState{
		PID:        pid,
		Protocol:   "UDP",
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		State:      "ACTIVE",
		StartTime:  time.Now(),
		LastSeen:   time.Now(),
	}
	
	// Resolve hostname if configured
	if nm.config.ResolveHostnames {
		conn.RemoteHost = nm.resolveHostname(strings.Split(remoteAddr, ":")[0])
	}
	
	// Check if this is a Claude API connection
	conn.IsClaudeAPI = nm.isClaudeEndpoint(conn.RemoteHost)
	
	return conn
}

/**
 * CONTEXT:   Generate network event for connection changes
 * INPUT:     Connection state and event type
 * OUTPUT:    Network event sent to callback
 * BUSINESS:  Event generation enables network activity tracking
 * CHANGE:    Initial network event generation
 * RISK:      Low - Event generation for network activity tracking
 */
func (nm *NetworkMonitor) generateConnectionEvent(conn *ConnectionState, eventType NetworkEventType) {
	event := NetworkEvent{
		Type:        eventType,
		Timestamp:   time.Now(),
		PID:         conn.PID,
		ProcessName: nm.getProcessName(conn.PID),
		Protocol:    conn.Protocol,
		LocalAddr:   conn.LocalAddr,
		RemoteAddr:  conn.RemoteAddr,
		RemoteHost:  conn.RemoteHost,
		State:       conn.State,
		IsClaudeAPI: conn.IsClaudeAPI,
		ProjectPath: nm.getProcessProjectPath(conn.PID),
		ProjectName: nm.getProcessProjectName(conn.PID),
		Details: map[string]interface{}{
			"start_time": conn.StartTime,
			"last_seen":  conn.LastSeen,
		},
	}
	
	if conn.IsClaudeAPI {
		event.Type = NetworkClaudeAPI
	}
	
	// Update statistics
	nm.stats.TotalConnections++
	if conn.IsClaudeAPI {
		nm.stats.ClaudeAPIConnections++
	}
	nm.stats.ConnectionsByState[conn.State]++
	if conn.RemoteHost != "" {
		nm.stats.ConnectionsByHost[conn.RemoteHost]++
	}
	
	// Send event to callback
	if nm.eventCallback != nil {
		go nm.eventCallback(event)
	}
	
	if nm.config.VerboseLogging {
		log.Printf("🌐 Network Event: %s %s %s -> %s (Claude API: %t)", 
			eventType, conn.Protocol, conn.LocalAddr, conn.RemoteAddr, conn.IsClaudeAPI)
	}
}

// Helper methods

func (nm *NetworkMonitor) getProcessSocketInodes(pid int) map[string]bool {
	inodes := make(map[string]bool)
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	
	files, err := ioutil.ReadDir(fdDir)
	if err != nil {
		return inodes
	}
	
	for _, file := range files {
		fdPath := filepath.Join(fdDir, file.Name())
		target, err := os.Readlink(fdPath)
		if err != nil {
			continue
		}
		
		if strings.HasPrefix(target, "socket:[") {
			inode := strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")
			inodes[inode] = true
		}
	}
	
	return inodes
}

func (nm *NetworkMonitor) parseHexAddress(hex string) string {
	parts := strings.Split(hex, ":")
	if len(parts) != 2 {
		return hex
	}
	
	ip, _ := strconv.ParseUint(parts[0], 16, 32)
	port, _ := strconv.ParseUint(parts[1], 16, 16)
	
	return fmt.Sprintf("%d.%d.%d.%d:%d",
		ip&0xFF, (ip>>8)&0xFF, (ip>>16)&0xFF, (ip>>24)&0xFF, port)
}

func (nm *NetworkMonitor) getTCPState(hex string) string {
	states := map[string]string{
		"01": "ESTABLISHED",
		"02": "SYN_SENT",
		"03": "SYN_RECV",
		"04": "FIN_WAIT1",
		"05": "FIN_WAIT2",
		"06": "TIME_WAIT",
		"07": "CLOSE",
		"08": "CLOSE_WAIT",
		"09": "LAST_ACK",
		"0A": "LISTEN",
		"0B": "CLOSING",
	}
	
	if state, ok := states[strings.ToUpper(hex)]; ok {
		return state
	}
	return hex
}

func (nm *NetworkMonitor) resolveHostname(ip string) string {
	// Simple hostname resolution (could be enhanced with DNS lookup)
	// For now, return the IP as hostname
	return ip
}

func (nm *NetworkMonitor) isClaudeEndpoint(host string) bool {
	if host == "" {
		return false
	}
	
	for _, pattern := range nm.claudeEndpoints {
		if pattern.MatchString(host) {
			return true
		}
	}
	
	return false
}

func (nm *NetworkMonitor) generateConnectionKey(conn *ConnectionState) string {
	return fmt.Sprintf("%s:%d:%s:%s", conn.Protocol, conn.PID, conn.LocalAddr, conn.RemoteAddr)
}

func (nm *NetworkMonitor) updateConnection(key string, conn *ConnectionState) {
	if existing, exists := nm.connections[key]; exists {
		conn.StartTime = existing.StartTime
		conn.BytesSent = existing.BytesSent
		conn.BytesRecv = existing.BytesRecv
	}
	conn.LastSeen = time.Now()
}

func (nm *NetworkMonitor) getProcessName(pid int) string {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	
	if proc, exists := nm.trackedProcesses[pid]; exists {
		return proc.Command
	}
	return fmt.Sprintf("PID_%d", pid)
}

func (nm *NetworkMonitor) getProcessProjectPath(pid int) string {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	
	if proc, exists := nm.trackedProcesses[pid]; exists {
		return proc.WorkingDir
	}
	return ""
}

func (nm *NetworkMonitor) getProcessProjectName(pid int) string {
	projectPath := nm.getProcessProjectPath(pid)
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
 * CONTEXT:   Get network monitoring statistics
 * INPUT:     Statistics request
 * OUTPUT:    Current network monitoring metrics
 * BUSINESS:  Statistics provide network monitoring system health and connectivity insights
 * CHANGE:    Initial network monitoring statistics getter
 * RISK:      Low - Read-only statistics access
 */
func (nm *NetworkMonitor) GetStats() NetworkStats {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	
	// Create a copy to prevent concurrent access issues
	stats := nm.stats
	stats.ConnectionsByState = make(map[string]uint64)
	stats.ConnectionsByHost = make(map[string]uint64)
	
	for k, v := range nm.stats.ConnectionsByState {
		stats.ConnectionsByState[k] = v
	}
	for k, v := range nm.stats.ConnectionsByHost {
		stats.ConnectionsByHost[k] = v
	}
	
	return stats
}

/**
 * CONTEXT:   Advanced socket inode detection for precise process correlation
 * INPUT:     File descriptor directory for process
 * OUTPUT:    Enhanced socket inode mapping with metadata
 * BUSINESS:  Advanced inode detection enables precise network tracking
 * CHANGE:    Enhanced socket detection from prototype with metadata
 * RISK:      Medium - Advanced file descriptor analysis affecting performance
 */
func (nm *NetworkMonitor) getAdvancedSocketInodes(fdDir string) map[string]map[string]interface{} {
	inodes := make(map[string]map[string]interface{})
	
	files, err := os.ReadDir(fdDir)
	if err != nil {
		return inodes
	}
	
	for _, file := range files {
		fdPath := filepath.Join(fdDir, file.Name())
		target, err := os.Readlink(fdPath)
		if err != nil {
			continue
		}
		
		if strings.HasPrefix(target, "socket:[") {
			inode := strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")
			
			// Get file descriptor info
			fdInfo, err := file.Info()
			if err == nil {
				inodes[inode] = map[string]interface{}{
					"fd":       file.Name(),
					"target":   target,
					"mod_time": fdInfo.ModTime(),
				}
			} else {
				inodes[inode] = map[string]interface{}{
					"fd":     file.Name(),
					"target": target,
				}
			}
		}
	}
	
	return inodes
}

/**
 * CONTEXT:   Advanced TCP connections by inode correlation
 * INPUT:     Socket inodes and process ID for correlation
 * OUTPUT:    TCP connections with comprehensive metadata
 * BUSINESS:  Advanced TCP correlation provides detailed connection tracking
 * CHANGE:    Enhanced TCP connection detection from prototype
 * RISK:      Medium - Advanced TCP parsing affecting performance
 */
func (nm *NetworkMonitor) getAdvancedTCPConnectionsByInodes(socketInodes map[string]map[string]interface{}, pid int) []map[string]interface{} {
	var connections []map[string]interface{}
	
	// Try process-specific /proc/PID/net/tcp first
	processNetFile := fmt.Sprintf("/proc/%d/net/tcp", pid)
	if data, err := os.ReadFile(processNetFile); err == nil {
		connections = append(connections, nm.parseAdvancedTCPConnections(string(data), socketInodes, pid, "process")...)
	}
	
	// Fallback to global /proc/net/tcp
	if len(connections) == 0 {
		if data, err := os.ReadFile("/proc/net/tcp"); err == nil {
			connections = append(connections, nm.parseAdvancedTCPConnections(string(data), socketInodes, pid, "global")...)
		}
	}
	
	// Also check IPv6 connections
	if data, err := os.ReadFile("/proc/net/tcp6"); err == nil {
		connections = append(connections, nm.parseAdvancedTCPConnections(string(data), socketInodes, pid, "ipv6")...)
	}
	
	return connections
}

/**
 * CONTEXT:   Advanced UDP connections by inode correlation
 * INPUT:     Socket inodes and process ID for correlation
 * OUTPUT:    UDP connections with comprehensive metadata
 * BUSINESS:  Advanced UDP correlation provides complete connection tracking
 * CHANGE:    Enhanced UDP connection detection from prototype
 * RISK:      Medium - Advanced UDP parsing affecting performance
 */
func (nm *NetworkMonitor) getAdvancedUDPConnectionsByInodes(socketInodes map[string]map[string]interface{}, pid int) []map[string]interface{} {
	var connections []map[string]interface{}
	
	// Read UDP connections
	if data, err := os.ReadFile("/proc/net/udp"); err == nil {
		connections = append(connections, nm.parseAdvancedUDPConnections(string(data), socketInodes, pid)...)
	}
	
	// Also check IPv6 UDP
	if data, err := os.ReadFile("/proc/net/udp6"); err == nil {
		connections = append(connections, nm.parseAdvancedUDPConnections(string(data), socketInodes, pid)...)
	}
	
	return connections
}

/**
 * CONTEXT:   Parse advanced TCP connections with comprehensive metadata
 * INPUT:     TCP connection data and socket inode correlation
 * OUTPUT:    Parsed TCP connections with enhanced information
 * BUSINESS:  Advanced TCP parsing provides detailed connection analytics
 * CHANGE:    Enhanced TCP parsing from prototype with metadata
 * RISK:      Medium - Complex TCP parsing affecting accuracy
 */
func (nm *NetworkMonitor) parseAdvancedTCPConnections(data string, socketInodes map[string]map[string]interface{}, pid int, source string) []map[string]interface{} {
	var connections []map[string]interface{}
	
	lines := strings.Split(data, "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header
		}
		
		fields := strings.Fields(line)
		if len(fields) < 12 {
			continue
		}
		
		inode := fields[9]
		if inodeInfo, exists := socketInodes[inode]; exists {
			localAddr := nm.parseAdvancedHexAddress(fields[1])
			remoteAddr := nm.parseAdvancedHexAddress(fields[2])
			state := nm.getAdvancedTCPState(fields[3])
			uid := fields[7]
			
			// Parse additional fields
			txQueue, _ := strconv.ParseInt(fields[4][:8], 16, 32)
			rxQueue, _ := strconv.ParseInt(fields[4][9:], 16, 32)
			timer := fields[5]
			timeout := fields[6]
			retransmits := fields[8]
			
			connection := map[string]interface{}{
				"protocol":     "TCP",
				"local":        localAddr,
				"remote":       remoteAddr,
				"state":        state,
				"inode":        inode,
				"fd":           inodeInfo["fd"],
				"pid":          pid,
				"uid":          uid,
				"source":       source,
				"tx_queue":     txQueue,
				"rx_queue":     rxQueue,
				"timer":        timer,
				"timeout":      timeout,
				"retransmits":  retransmits,
				"is_claude_api": nm.isClaudeEndpoint(nm.extractHostFromAddress(remoteAddr)),
			}
			
			// Add timestamp if available from inode info
			if modTime, exists := inodeInfo["mod_time"]; exists {
				connection["fd_mod_time"] = modTime
			}
			
			connections = append(connections, connection)
		}
	}
	
	return connections
}

/**
 * CONTEXT:   Parse advanced UDP connections with comprehensive metadata
 * INPUT:     UDP connection data and socket inode correlation
 * OUTPUT:    Parsed UDP connections with enhanced information
 * BUSINESS:  Advanced UDP parsing provides detailed connection analytics
 * CHANGE:    Enhanced UDP parsing from prototype with metadata
 * RISK:      Medium - Complex UDP parsing affecting accuracy
 */
func (nm *NetworkMonitor) parseAdvancedUDPConnections(data string, socketInodes map[string]map[string]interface{}, pid int) []map[string]interface{} {
	var connections []map[string]interface{}
	
	lines := strings.Split(data, "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header
		}
		
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		
		inode := fields[9]
		if inodeInfo, exists := socketInodes[inode]; exists {
			localAddr := nm.parseAdvancedHexAddress(fields[1])
			remoteAddr := nm.parseAdvancedHexAddress(fields[2])
			state := nm.getAdvancedUDPState(fields[3])
			uid := fields[7]
			
			// Parse additional UDP fields
			txQueue, _ := strconv.ParseInt(fields[4][:8], 16, 32)
			rxQueue, _ := strconv.ParseInt(fields[4][9:], 16, 32)
			drops := "0"
			if len(fields) > 12 {
				drops = fields[12]
			}
			
			connection := map[string]interface{}{
				"protocol":      "UDP",
				"local":         localAddr,
				"remote":        remoteAddr,
				"state":         state,
				"inode":         inode,
				"fd":            inodeInfo["fd"],
				"pid":           pid,
				"uid":           uid,
				"tx_queue":      txQueue,
				"rx_queue":      rxQueue,
				"drops":         drops,
				"is_claude_api": nm.isClaudeEndpoint(nm.extractHostFromAddress(remoteAddr)),
			}
			
			// Add timestamp if available from inode info
			if modTime, exists := inodeInfo["mod_time"]; exists {
				connection["fd_mod_time"] = modTime
			}
			
			connections = append(connections, connection)
		}
	}
	
	return connections
}

/**
 * CONTEXT:   Parse advanced hex address with enhanced IPv6 support
 * INPUT:     Hexadecimal address string from /proc/net
 * OUTPUT:    Human-readable IP:port address
 * BUSINESS:  Advanced address parsing supports IPv4 and IPv6
 * CHANGE:    Enhanced address parsing from prototype with IPv6 support
 * RISK:      Low - Address parsing utility with IPv6 enhancement
 */
func (nm *NetworkMonitor) parseAdvancedHexAddress(hex string) string {
	parts := strings.Split(hex, ":")
	if len(parts) != 2 {
		return hex
	}
	
	// Check if this is IPv6 (length > 8 chars for IP part)
	if len(parts[0]) > 8 {
		// IPv6 address parsing
		ipHex := parts[0]
		portHex := parts[1]
		
		// Parse IPv6 address (simplified)
		if len(ipHex) == 32 { // IPv6 is 32 hex chars
			var ipParts []string
			for i := 0; i < 32; i += 4 {
				if i+4 <= len(ipHex) {
					part := ipHex[i : i+4]
					ipParts = append(ipParts, part)
				}
			}
			
			port, _ := strconv.ParseUint(portHex, 16, 16)
			return fmt.Sprintf("[%s]:%d", strings.Join(ipParts, ":"), port)
		}
	}
	
	// IPv4 address parsing (original logic)
	ip, _ := strconv.ParseUint(parts[0], 16, 32)
	port, _ := strconv.ParseUint(parts[1], 16, 16)
	
	return fmt.Sprintf("%d.%d.%d.%d:%d",
		ip&0xFF, (ip>>8)&0xFF, (ip>>16)&0xFF, (ip>>24)&0xFF, port)
}

/**
 * CONTEXT:   Get advanced TCP state with additional state information
 * INPUT:     Hex state string from /proc/net/tcp
 * OUTPUT:    Human-readable TCP state with additional info
 * BUSINESS:  Advanced state detection provides detailed connection status
 * CHANGE:    Enhanced TCP state parsing from prototype
 * RISK:      Low - TCP state parsing utility enhancement
 */
func (nm *NetworkMonitor) getAdvancedTCPState(hex string) string {
	states := map[string]string{
		"01": "ESTABLISHED",
		"02": "SYN_SENT",
		"03": "SYN_RECV",
		"04": "FIN_WAIT1",
		"05": "FIN_WAIT2",
		"06": "TIME_WAIT",
		"07": "CLOSE",
		"08": "CLOSE_WAIT",
		"09": "LAST_ACK",
		"0A": "LISTEN",
		"0B": "CLOSING",
		"0C": "NEW_SYN_RECV",
	}
	
	upperHex := strings.ToUpper(hex)
	if state, ok := states[upperHex]; ok {
		return state
	}
	return fmt.Sprintf("UNKNOWN_%s", upperHex)
}

/**
 * CONTEXT:   Get advanced UDP state information
 * INPUT:     Hex state string from /proc/net/udp
 * OUTPUT:    Human-readable UDP state
 * BUSINESS:  UDP state detection provides connection status
 * CHANGE:    Enhanced UDP state parsing from prototype
 * RISK:      Low - UDP state parsing utility enhancement
 */
func (nm *NetworkMonitor) getAdvancedUDPState(hex string) string {
	// UDP states are simpler than TCP
	states := map[string]string{
		"01": "ESTABLISHED",
		"07": "CLOSE",
		"0A": "LISTEN",
	}
	
	upperHex := strings.ToUpper(hex)
	if state, ok := states[upperHex]; ok {
		return state
	}
	return "ACTIVE" // Most UDP connections are just active
}

/**
 * CONTEXT:   Extract host from address string
 * INPUT:     Address string in IP:port format
 * OUTPUT:    Host/IP portion of address
 * BUSINESS:  Host extraction enables endpoint analysis
 * CHANGE:    Enhanced host extraction from prototype
 * RISK:      Low - Address parsing utility
 */
func (nm *NetworkMonitor) extractHostFromAddress(address string) string {
	if strings.Contains(address, "]:") {
		// IPv6 format [host]:port
		parts := strings.Split(address, "]:")
		if len(parts) > 0 {
			return strings.TrimPrefix(parts[0], "[")
		}
	} else {
		// IPv4 format host:port
		parts := strings.Split(address, ":")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return address
}