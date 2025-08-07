package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============== TIPOS DE DATOS ==============

type EventType string

const (
	EventFileRead    EventType = "FILE_READ"
	EventFileWrite   EventType = "FILE_WRITE"
	EventFileOpen    EventType = "FILE_OPEN"
	EventFileDelete  EventType = "FILE_DELETE"
	EventHTTPPost    EventType = "HTTP_POST"
	EventNetConnect  EventType = "NET_CONNECT"
	EventProcessExec EventType = "PROCESS_EXEC"
)

type Event struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        EventType              `json:"type"`
	PID         int                    `json:"pid"`
	ProcessName string                 `json:"process_name"`
	Details     map[string]interface{} `json:"details"`
}

type ProcessMonitor struct {
	targetPID    int
	targetName   string
	events       chan Event
	stopChannels map[string]chan bool
	wg           sync.WaitGroup
	isWSL        bool
	config       MonitorConfig
}

type MonitorConfig struct {
	TargetPID          int
	TargetProcessName  string
	MonitorFileIO      bool
	MonitorNetwork     bool
	MonitorHTTP        bool
	HTTPPorts          []int
	OutputFile         string
	VerboseMode        bool
	RefreshInterval    time.Duration
}

// ============== DETECTOR DE ENTORNO ==============

func detectEnvironment() (bool, string) {
	// Detectar si estamos en WSL
	if data, err := os.ReadFile("/proc/version"); err == nil {
		version := strings.ToLower(string(data))
		if strings.Contains(version, "microsoft") {
			return true, "WSL2"
		}
	}
	
	return false, runtime.GOOS
}

// ============== MONITOR PRINCIPAL ==============

func NewProcessMonitor(config MonitorConfig) *ProcessMonitor {
	isWSL, env := detectEnvironment()
	
	log.Printf("Entorno detectado: %s", env)
	
	return &ProcessMonitor{
		targetPID:    config.TargetPID,
		targetName:   config.TargetProcessName,
		events:       make(chan Event, 1000),
		stopChannels: make(map[string]chan bool),
		isWSL:        isWSL,
		config:       config,
	}
}

func (pm *ProcessMonitor) Start() error {
	// Si no se especificó PID, buscar por nombre
	if pm.targetPID == 0 && pm.targetName != "" {
		if pid, err := pm.findProcessByName(pm.targetName); err == nil {
			pm.targetPID = pid
			log.Printf("Proceso encontrado: %s (PID: %d)", pm.targetName, pid)
		} else {
			return fmt.Errorf("proceso no encontrado: %s", pm.targetName)
		}
	}
	
	// Verificar que el proceso existe
	if !pm.processExists(pm.targetPID) {
		return fmt.Errorf("proceso con PID %d no existe", pm.targetPID)
	}
	
	// Iniciar monitores según configuración
	if pm.config.MonitorFileIO {
		pm.startFileIOMonitor()
	}
	
	if pm.config.MonitorNetwork {
		pm.startNetworkMonitor()
	}
	
	if pm.config.MonitorHTTP {
		pm.startHTTPMonitor()
	}
	
	// Monitor de estado del proceso
	pm.startProcessStateMonitor()
	
	return nil
}

// ============== MONITOREO DE FILE I/O ==============

func (pm *ProcessMonitor) startFileIOMonitor() {
	stop := make(chan bool)
	pm.stopChannels["fileio"] = stop
	
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		
		// Método 1: Usando strace (más detallado)
		if pm.canUseStrace() {
			pm.monitorWithStrace(stop)
		} else {
			// Método 2: Leyendo /proc/PID/io
			pm.monitorProcIO(stop)
		}
	}()
	
	// Monitor de archivos abiertos
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		pm.monitorOpenFiles(stop)
	}()
}

func (pm *ProcessMonitor) monitorWithStrace(stop chan bool) {
	cmd := exec.Command("strace", "-p", strconv.Itoa(pm.targetPID),
		"-e", "trace=open,openat,read,write,close,unlink,rename",
		"-f", "-q", "-s", "100")
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Error iniciando strace: %v", err)
		return
	}
	
	if err := cmd.Start(); err != nil {
		log.Printf("Error ejecutando strace: %v", err)
		return
	}
	
	scanner := bufio.NewScanner(stderr)
	
	// Regex patterns para parsear strace output
	openPattern := regexp.MustCompile(`open(?:at)?\([^"]*"([^"]+)"`)
	readPattern := regexp.MustCompile(`read\((\d+),.*= (\d+)`)
	writePattern := regexp.MustCompile(`write\((\d+),.*= (\d+)`)
	
	go func() {
		for scanner.Scan() {
			select {
			case <-stop:
				cmd.Process.Kill()
				return
			default:
				line := scanner.Text()
				
				// Detectar operación open
				if matches := openPattern.FindStringSubmatch(line); len(matches) > 1 {
					pm.events <- Event{
						Timestamp:   time.Now(),
						Type:        EventFileOpen,
						PID:         pm.targetPID,
						ProcessName: pm.getProcessName(),
						Details: map[string]interface{}{
							"file": matches[1],
						},
					}
				}
				
				// Detectar read
				if matches := readPattern.FindStringSubmatch(line); len(matches) > 2 {
					size, _ := strconv.Atoi(matches[2])
					pm.events <- Event{
						Timestamp:   time.Now(),
						Type:        EventFileRead,
						PID:         pm.targetPID,
						ProcessName: pm.getProcessName(),
						Details: map[string]interface{}{
							"fd":   matches[1],
							"size": size,
						},
					}
				}
				
				// Detectar write
				if matches := writePattern.FindStringSubmatch(line); len(matches) > 2 {
					size, _ := strconv.Atoi(matches[2])
					pm.events <- Event{
						Timestamp:   time.Now(),
						Type:        EventFileWrite,
						PID:         pm.targetPID,
						ProcessName: pm.getProcessName(),
						Details: map[string]interface{}{
							"fd":   matches[1],
							"size": size,
						},
					}
				}
			}
		}
	}()
}

func (pm *ProcessMonitor) monitorProcIO(stop chan bool) {
	ioFile := fmt.Sprintf("/proc/%d/io", pm.targetPID)
	
	var lastRead, lastWrite int64
	ticker := time.NewTicker(pm.config.RefreshInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			data, err := os.ReadFile(ioFile)
			if err != nil {
				continue
			}
			
			stats := pm.parseProcIO(string(data))
			
			readBytes := stats["read_bytes"]
			writeBytes := stats["write_bytes"]
			
			// Detectar cambios
			if lastRead > 0 && readBytes > lastRead {
				pm.events <- Event{
					Timestamp:   time.Now(),
					Type:        EventFileRead,
					PID:         pm.targetPID,
					ProcessName: pm.getProcessName(),
					Details: map[string]interface{}{
						"bytes_delta": readBytes - lastRead,
						"total_bytes": readBytes,
					},
				}
			}
			
			if lastWrite > 0 && writeBytes > lastWrite {
				pm.events <- Event{
					Timestamp:   time.Now(),
					Type:        EventFileWrite,
					PID:         pm.targetPID,
					ProcessName: pm.getProcessName(),
					Details: map[string]interface{}{
						"bytes_delta": writeBytes - lastWrite,
						"total_bytes": writeBytes,
					},
				}
			}
			
			lastRead = readBytes
			lastWrite = writeBytes
		}
	}
}

func (pm *ProcessMonitor) monitorOpenFiles(stop chan bool) {
	fdDir := fmt.Sprintf("/proc/%d/fd", pm.targetPID)
	
	ticker := time.NewTicker(pm.config.RefreshInterval * 2)
	defer ticker.Stop()
	
	knownFiles := make(map[string]bool)
	
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			files, err := os.ReadDir(fdDir)
			if err != nil {
				continue
			}
			
			currentFiles := make(map[string]bool)
			
			for _, file := range files {
				fdPath := filepath.Join(fdDir, file.Name())
				target, err := os.Readlink(fdPath)
				if err != nil {
					continue
				}
				
				currentFiles[target] = true
				
				// Nuevo archivo abierto
				if !knownFiles[target] && !strings.Contains(target, "pipe:") && 
				   !strings.Contains(target, "socket:") {
					pm.events <- Event{
						Timestamp:   time.Now(),
						Type:        EventFileOpen,
						PID:         pm.targetPID,
						ProcessName: pm.getProcessName(),
						Details: map[string]interface{}{
							"file": target,
							"fd":   file.Name(),
						},
					}
				}
			}
			
			knownFiles = currentFiles
		}
	}
}

// ============== MONITOREO DE RED Y HTTP ==============

func (pm *ProcessMonitor) startNetworkMonitor() {
	stop := make(chan bool)
	pm.stopChannels["network"] = stop
	
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		pm.monitorNetworkConnections(stop)
	}()
}

func (pm *ProcessMonitor) monitorNetworkConnections(stop chan bool) {
	ticker := time.NewTicker(pm.config.RefreshInterval)
	defer ticker.Stop()
	
	knownConnections := make(map[string]bool)
	
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			connections := pm.getProcessConnections()
			
			for _, conn := range connections {
				connKey := fmt.Sprintf("%s:%s", conn["local"], conn["remote"])
				
				if !knownConnections[connKey] {
					pm.events <- Event{
						Timestamp:   time.Now(),
						Type:        EventNetConnect,
						PID:         pm.targetPID,
						ProcessName: pm.getProcessName(),
						Details:     conn,
					}
					knownConnections[connKey] = true
				}
			}
		}
	}
}

func (pm *ProcessMonitor) getProcessConnections() []map[string]interface{} {
	var connections []map[string]interface{}
	
	// Leer conexiones TCP
	tcpFile := fmt.Sprintf("/proc/%d/net/tcp", pm.targetPID)
	if _, err := os.Stat(tcpFile); err == nil {
		// Proceso tiene namespace de red propio
		connections = append(connections, pm.parseTCPConnections(tcpFile)...)
	} else {
		// Usar /proc/net/tcp global y filtrar por inodo
		connections = append(connections, pm.getConnectionsByInode()...)
	}
	
	return connections
}

func (pm *ProcessMonitor) getConnectionsByInode() []map[string]interface{} {
	var connections []map[string]interface{}
	
	// Obtener inodos de sockets del proceso
	fdDir := fmt.Sprintf("/proc/%d/fd", pm.targetPID)
	files, err := os.ReadDir(fdDir)
	if err != nil {
		return connections
	}
	
	inodes := make(map[string]bool)
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
	
	// Leer /proc/net/tcp
	data, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		return connections
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
		if inodes[inode] {
			localAddr := pm.parseHexAddress(fields[1])
			remoteAddr := pm.parseHexAddress(fields[2])
			state := pm.getTCPState(fields[3])
			
			connections = append(connections, map[string]interface{}{
				"local":  localAddr,
				"remote": remoteAddr,
				"state":  state,
				"inode":  inode,
			})
		}
	}
	
	return connections
}

func (pm *ProcessMonitor) startHTTPMonitor() {
	stop := make(chan bool)
	pm.stopChannels["http"] = stop
	
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		
		// Método 1: Captura de paquetes con tcpdump
		if pm.canUseTcpdump() {
			pm.monitorHTTPWithTcpdump(stop)
		} else {
			// Método 2: Proxy HTTP interceptor
			pm.startHTTPProxy(stop)
		}
	}()
}

func (pm *ProcessMonitor) monitorHTTPWithTcpdump(stop chan bool) {
	ports := "80"
	if len(pm.config.HTTPPorts) > 0 {
		portStrs := make([]string, len(pm.config.HTTPPorts))
		for i, p := range pm.config.HTTPPorts {
			portStrs[i] = fmt.Sprintf("port %d", p)
		}
		ports = strings.Join(portStrs, " or ")
	}
	
	cmd := exec.Command("tcpdump", "-i", "any", "-A", "-s", "0",
		fmt.Sprintf("tcp and (%s)", ports), "-l")
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error iniciando tcpdump: %v", err)
		return
	}
	
	if err := cmd.Start(); err != nil {
		log.Printf("Error ejecutando tcpdump: %v", err)
		return
	}
	
	scanner := bufio.NewScanner(stdout)
	var buffer strings.Builder
	inPacket := false
	
	go func() {
		for scanner.Scan() {
			select {
			case <-stop:
				cmd.Process.Kill()
				return
			default:
				line := scanner.Text()
				
				// Detectar inicio de paquete
				if strings.Contains(line, " > ") {
					if buffer.Len() > 0 {
						pm.parseHTTPPacket(buffer.String())
					}
					buffer.Reset()
					inPacket = true
				}
				
				if inPacket {
					buffer.WriteString(line + "\n")
				}
			}
		}
	}()
}

func (pm *ProcessMonitor) parseHTTPPacket(packet string) {
	// Buscar POST requests
	if strings.Contains(packet, "POST ") {
		lines := strings.Split(packet, "\n")
		var url, host, contentType string
		var contentLength int
		
		for _, line := range lines {
			if strings.HasPrefix(line, "POST ") {
				parts := strings.Fields(line)
				if len(parts) > 1 {
					url = parts[1]
				}
			} else if strings.HasPrefix(line, "Host: ") {
				host = strings.TrimPrefix(line, "Host: ")
			} else if strings.HasPrefix(line, "Content-Type: ") {
				contentType = strings.TrimPrefix(line, "Content-Type: ")
			} else if strings.HasPrefix(line, "Content-Length: ") {
				length := strings.TrimPrefix(line, "Content-Length: ")
				contentLength, _ = strconv.Atoi(strings.TrimSpace(length))
			}
		}
		
		if url != "" {
			pm.events <- Event{
				Timestamp:   time.Now(),
				Type:        EventHTTPPost,
				PID:         pm.targetPID,
				ProcessName: pm.getProcessName(),
				Details: map[string]interface{}{
					"method":         "POST",
					"url":            url,
					"host":           host,
					"content_type":   contentType,
					"content_length": contentLength,
				},
			}
		}
	}
}

func (pm *ProcessMonitor) startHTTPProxy(stop chan bool) {
	// Proxy HTTP simple para interceptar requests
	proxy := &http.Server{
		Addr: ":8888",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				body, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(body))
				
				pm.events <- Event{
					Timestamp:   time.Now(),
					Type:        EventHTTPPost,
					PID:         pm.targetPID,
					ProcessName: pm.getProcessName(),
					Details: map[string]interface{}{
						"method":       r.Method,
						"url":          r.URL.String(),
						"host":         r.Host,
						"content_type": r.Header.Get("Content-Type"),
						"body_size":    len(body),
					},
				}
			}
			
			// Forward request
			client := &http.Client{}
			targetURL := "http://" + r.Host + r.URL.String()
			proxyReq, _ := http.NewRequest(r.Method, targetURL, r.Body)
			proxyReq.Header = r.Header
			
			resp, err := client.Do(proxyReq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()
			
			// Copy response
			for k, v := range resp.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
		}),
	}
	
	go func() {
		<-stop
		proxy.Close()
	}()
	
	log.Printf("Proxy HTTP iniciado en :8888")
	proxy.ListenAndServe()
}

// ============== MONITOR DE ESTADO DEL PROCESO ==============

func (pm *ProcessMonitor) startProcessStateMonitor() {
	stop := make(chan bool)
	pm.stopChannels["state"] = stop
	
	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if !pm.processExists(pm.targetPID) {
					log.Printf("Proceso %d terminado", pm.targetPID)
					pm.Stop()
					return
				}
			}
		}
	}()
}

// ============== UTILIDADES ==============

func (pm *ProcessMonitor) findProcessByName(name string) (int, error) {
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}
	
	for _, proc := range procs {
		if !proc.IsDir() {
			continue
		}
		
		pid, err := strconv.Atoi(proc.Name())
		if err != nil {
			continue
		}
		
		cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			continue
		}
		
		if strings.Contains(string(cmdline), name) {
			return pid, nil
		}
	}
	
	return 0, fmt.Errorf("proceso no encontrado")
}

func (pm *ProcessMonitor) processExists(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

func (pm *ProcessMonitor) getProcessName() string {
	if pm.targetName != "" {
		return pm.targetName
	}
	
	comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pm.targetPID))
	if err != nil {
		return fmt.Sprintf("PID_%d", pm.targetPID)
	}
	
	return strings.TrimSpace(string(comm))
}

func (pm *ProcessMonitor) canUseStrace() bool {
	_, err := exec.LookPath("strace")
	return err == nil
}

func (pm *ProcessMonitor) canUseTcpdump() bool {
	_, err := exec.LookPath("tcpdump")
	return err == nil
}

func (pm *ProcessMonitor) parseProcIO(data string) map[string]int64 {
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

func (pm *ProcessMonitor) parseTCPConnections(file string) []map[string]interface{} {
	// Implementación para parsear /proc/net/tcp
	return []map[string]interface{}{}
}

func (pm *ProcessMonitor) parseHexAddress(hex string) string {
	parts := strings.Split(hex, ":")
	if len(parts) != 2 {
		return hex
	}
	
	ip, _ := strconv.ParseUint(parts[0], 16, 32)
	port, _ := strconv.ParseUint(parts[1], 16, 16)
	
	return fmt.Sprintf("%d.%d.%d.%d:%d",
		ip&0xFF, (ip>>8)&0xFF, (ip>>16)&0xFF, (ip>>24)&0xFF, port)
}

func (pm *ProcessMonitor) getTCPState(hex string) string {
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
	
	if state, ok := states[hex]; ok {
		return state
	}
	return hex
}

func (pm *ProcessMonitor) Stop() {
	// Detener todos los monitores
	for _, stop := range pm.stopChannels {
		close(stop)
	}
	
	pm.wg.Wait()
	close(pm.events)
}

func (pm *ProcessMonitor) GetEvents() <-chan Event {
	return pm.events
}

// ============== OUTPUT Y REPORTING ==============

type OutputHandler struct {
	file        *os.File
	jsonEncoder *json.Encoder
	verbose     bool
}

func NewOutputHandler(filename string, verbose bool) (*OutputHandler, error) {
	var file *os.File
	var err error
	
	if filename != "" {
		file, err = os.Create(filename)
		if err != nil {
			return nil, err
		}
	}
	
	oh := &OutputHandler{
		file:    file,
		verbose: verbose,
	}
	
	if file != nil {
		oh.jsonEncoder = json.NewEncoder(file)
	}
	
	return oh, nil
}

func (oh *OutputHandler) HandleEvent(event Event) {
	// Formato de consola
	if oh.verbose || event.Type == EventHTTPPost {
		oh.printEvent(event)
	}
	
	// Guardar en archivo JSON
	if oh.jsonEncoder != nil {
		oh.jsonEncoder.Encode(event)
	}
}

func (oh *OutputHandler) printEvent(event Event) {
	emoji := map[EventType]string{
		EventFileRead:    "📖",
		EventFileWrite:   "✍️",
		EventFileOpen:    "📂",
		EventFileDelete:  "🗑️",
		EventHTTPPost:    "🔴",
		EventNetConnect:  "🌐",
		EventProcessExec: "🚀",
	}[event.Type]
	
	fmt.Printf("%s [%s] %s %s (PID: %d)\n",
		emoji,
		event.Timestamp.Format("15:04:05.000"),
		event.Type,
		event.ProcessName,
		event.PID)
	
	// Imprimir detalles según el tipo
	switch event.Type {
	case EventFileOpen:
		if file, ok := event.Details["file"].(string); ok {
			fmt.Printf("  📁 File: %s\n", file)
		}
	case EventFileRead, EventFileWrite:
		if size, ok := event.Details["bytes_delta"].(int64); ok {
			fmt.Printf("  📊 Size: %d bytes\n", size)
		}
	case EventHTTPPost:
		fmt.Printf("  🔗 URL: %s%s\n", event.Details["host"], event.Details["url"])
		fmt.Printf("  📦 Content-Type: %s\n", event.Details["content_type"])
		if length, ok := event.Details["content_length"].(int); ok {
			fmt.Printf("  📏 Size: %d bytes\n", length)
		}
	case EventNetConnect:
		fmt.Printf("  🔌 %s -> %s (%s)\n",
			event.Details["local"],
			event.Details["remote"],
			event.Details["state"])
	}
	
	fmt.Println()
}

func (oh *OutputHandler) Close() {
	if oh.file != nil {
		oh.file.Close()
	}
}

// ============== ESTADÍSTICAS ==============

type Statistics struct {
	mu          sync.RWMutex
	eventCounts map[EventType]int
	totalBytes  struct {
		Read  int64
		Write int64
	}
	httpRequests int
	startTime    time.Time
}

func NewStatistics() *Statistics {
	return &Statistics{
		eventCounts: make(map[EventType]int),
		startTime:   time.Now(),
	}
}

func (s *Statistics) UpdateEvent(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.eventCounts[event.Type]++
	
	switch event.Type {
	case EventFileRead:
		if bytes, ok := event.Details["bytes_delta"].(int64); ok {
			s.totalBytes.Read += bytes
		}
	case EventFileWrite:
		if bytes, ok := event.Details["bytes_delta"].(int64); ok {
			s.totalBytes.Write += bytes
		}
	case EventHTTPPost:
		s.httpRequests++
	}
}

func (s *Statistics) PrintSummary() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	duration := time.Since(s.startTime)
	
	fmt.Println("\n========== RESUMEN DE ACTIVIDAD ==========")
	fmt.Printf("Tiempo de monitoreo: %s\n", duration.Round(time.Second))
	fmt.Println("\nEventos por tipo:")
	
	for eventType, count := range s.eventCounts {
		fmt.Printf("  %s: %d\n", eventType, count)
	}
	
	fmt.Printf("\nI/O Total:\n")
	fmt.Printf("  Lectura: %s\n", formatBytes(s.totalBytes.Read))
	fmt.Printf("  Escritura: %s\n", formatBytes(s.totalBytes.Write))
	fmt.Printf("\nHTTP POST Requests: %d\n", s.httpRequests)
}

func formatBytes(bytes int64) string {
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

// ============== MAIN ==============

func main() {
	// Configuración por línea de comandos
	config := MonitorConfig{
		TargetPID:         0,                    // Se puede especificar
		TargetProcessName: "",                   // O buscar por nombre
		MonitorFileIO:     true,
		MonitorNetwork:    true,
		MonitorHTTP:       true,
		HTTPPorts:         []int{80, 443, 8080, 3000},
		OutputFile:        "monitor_output.json",
		VerboseMode:       true,
		RefreshInterval:   1 * time.Second,
	}
	
	// Parsear argumentos
	if len(os.Args) > 1 {
		if pid, err := strconv.Atoi(os.Args[1]); err == nil {
			config.TargetPID = pid
		} else {
			config.TargetProcessName = os.Args[1]
		}
	} else {
		fmt.Println("Uso: monitor <PID o nombre_proceso>")
		fmt.Println("Ejemplo: monitor 1234")
		fmt.Println("Ejemplo: monitor firefox")
		os.Exit(1)
	}
	
	// Crear monitor
	monitor := NewProcessMonitor(config)
	
	// Crear manejador de salida
	output, err := NewOutputHandler(config.OutputFile, config.VerboseMode)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()
	
	// Estadísticas
	stats := NewStatistics()
	
	// Iniciar monitoreo
	if err := monitor.Start(); err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("📊 Monitoreando proceso: %s (PID: %d)\n", 
		monitor.getProcessName(), monitor.targetPID)
	fmt.Println("Presiona Ctrl+C para detener...\n")
	
	// Manejar señales
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	
	// Procesar eventos
	go func() {
		for event := range monitor.GetEvents() {
			output.HandleEvent(event)
			stats.UpdateEvent(event)
		}
	}()
	
	// Esperar interrupción
	<-sigChan
	
	fmt.Println("\n\n⏹️  Deteniendo monitor...")
	monitor.Stop()
	
	// Mostrar estadísticas finales
	stats.PrintSummary()
	
	fmt.Println("\n✅ Monitor detenido. Resultados guardados en:", config.OutputFile)
}