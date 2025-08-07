package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ============== TIPOS DE EVENTOS ==============

type HTTPEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Method      string    `json:"method"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Path        string    `json:"path,omitempty"`
	Protocol    string    `json:"protocol"` // HTTP o HTTPS
	BytesSent   int64     `json:"bytes_sent"`
	BytesRecv   int64     `json:"bytes_received"`
	Duration    int64     `json:"duration_ms,omitempty"`
	ProcessName string    `json:"process_name"`
	PID         int       `json:"pid"`
}

type ConnectionTracker struct {
	mu          sync.RWMutex
	connections map[string]*ConnectionInfo
	events      chan HTTPEvent
}

type ConnectionInfo struct {
	StartTime   time.Time
	Host        string
	Port        int
	Protocol    string
	BytesSent   int64
	BytesRecv   int64
	Method      string
	Path        string
	ProcessName string
	PID         int
}

// ============== MONITOR PRINCIPAL ==============

type HTTPMonitor struct {
	targetPID    int
	targetName   string
	tracker      *ConnectionTracker
	stopChannels map[string]chan bool
	wg           sync.WaitGroup
}

func NewHTTPMonitor(pid int, name string) *HTTPMonitor {
	return &HTTPMonitor{
		targetPID:    pid,
		targetName:   name,
		tracker:      NewConnectionTracker(),
		stopChannels: make(map[string]chan bool),
	}
}

func NewConnectionTracker() *ConnectionTracker {
	return &ConnectionTracker{
		connections: make(map[string]*ConnectionInfo),
		events:      make(chan HTTPEvent, 100),
	}
}

func (m *HTTPMonitor) Start() error {
	// Verificar si el proceso existe
	if m.targetPID > 0 && !processExists(m.targetPID) {
		return fmt.Errorf("proceso con PID %d no existe", m.targetPID)
	}

	// Si se especificó nombre, buscar PID
	if m.targetPID == 0 && m.targetName != "" {
		pid, err := findProcessByName(m.targetName)
		if err != nil {
			return err
		}
		m.targetPID = pid
		log.Printf("Proceso encontrado: %s (PID: %d)", m.targetName, pid)
	}

	// Método 1: Monitorear conexiones TCP del proceso
	m.startTCPMonitor()

	// Método 2: Usar ss para capturar información detallada
	m.startSSMonitor()

	// Método 3: Capturar con tcpdump (requiere sudo)
	if canUseTcpdump() {
		m.startTcpdumpMonitor()
	}

	return nil
}

// ============== MONITOR TCP (Lee /proc) ==============

func (m *HTTPMonitor) startTCPMonitor() {
	stop := make(chan bool)
	m.stopChannels["tcp"] = stop

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.monitorTCPConnections(stop)
	}()
}

func (m *HTTPMonitor) monitorTCPConnections(stop chan bool) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastConnections := make(map[string]*ConnectionInfo)

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			connections := m.getProcessConnections()

			// Detectar nuevas conexiones
			for key, conn := range connections {
				if _, exists := lastConnections[key]; !exists {
					// Nueva conexión detectada
					m.tracker.AddConnection(key, conn)

					// Si es puerto HTTP/HTTPS conocido, registrar evento inicial
					if isHTTPPort(conn.Port) {
						log.Printf("🌐 Nueva conexión %s a %s:%d",
							conn.Protocol, conn.Host, conn.Port)
					}
				} else {
					// Actualizar bytes transferidos
					m.tracker.UpdateBytes(key, conn.BytesSent, conn.BytesRecv)
				}
			}

			// Detectar conexiones cerradas
			for key, conn := range lastConnections {
				if _, exists := connections[key]; !exists {
					// Conexión cerrada, generar evento
					m.tracker.CloseConnection(key, conn)
				}
			}

			lastConnections = connections
		}
	}
}

func (m *HTTPMonitor) getProcessConnections() map[string]*ConnectionInfo {
	connections := make(map[string]*ConnectionInfo)

	// Obtener file descriptors del proceso
	fdDir := fmt.Sprintf("/proc/%d/fd", m.targetPID)
	files, err := os.ReadDir(fdDir)
	if err != nil {
		return connections
	}

	// Mapear inodos de sockets
	socketInodes := make(map[string]bool)
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

	// Leer conexiones TCP
	m.readTCPConnections(socketInodes, connections, "/proc/net/tcp", "HTTP")
	m.readTCPConnections(socketInodes, connections, "/proc/net/tcp6", "HTTP")

	// Obtener estadísticas de I/O para cada conexión
	m.enrichConnectionsWithIOStats(connections)

	return connections
}

func (m *HTTPMonitor) readTCPConnections(inodes map[string]bool,
	connections map[string]*ConnectionInfo, file string, protocol string) {

	data, err := os.ReadFile(file)
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

		localAddr := parseHexAddress(fields[1])
		remoteAddr := parseHexAddress(fields[2])
		state := getTCPState(fields[3])

		// Solo conexiones establecidas
		if state != "ESTABLISHED" {
			continue
		}

		host, port := splitAddress(remoteAddr)

		// Determinar si es HTTP o HTTPS por el puerto
		if port == 443 {
			protocol = "HTTPS"
		} else if port == 80 {
			protocol = "HTTP"
		}

		connKey := fmt.Sprintf("%s->%s", localAddr, remoteAddr)

		connections[connKey] = &ConnectionInfo{
			StartTime:   time.Now(),
			Host:        host,
			Port:        port,
			Protocol:    protocol,
			ProcessName: m.targetName,
			PID:         m.targetPID,
		}

		// Obtener bytes transmitidos si es posible
		if len(fields) >= 17 {
			// Los campos 4 y 5 contienen tx_queue:rx_queue
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

func (m *HTTPMonitor) enrichConnectionsWithIOStats(connections map[string]*ConnectionInfo) {
	// Leer estadísticas de I/O del proceso
	ioFile := fmt.Sprintf("/proc/%d/io", m.targetPID)
	data, err := os.ReadFile(ioFile)
	if err != nil {
		return
	}

	stats := parseProcIO(string(data))

	// Distribuir I/O entre conexiones activas (aproximación)
	if len(connections) > 0 {
		bytesPerConn := stats["write_bytes"] / int64(len(connections))
		for _, conn := range connections {
			if conn.BytesSent == 0 {
				conn.BytesSent = bytesPerConn
			}
		}
	}
}

// ============== MONITOR SS (Socket Statistics) ==============

func (m *HTTPMonitor) startSSMonitor() {
	if _, err := exec.LookPath("ss"); err != nil {
		return // ss no disponible
	}

	stop := make(chan bool)
	m.stopChannels["ss"] = stop

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.monitorWithSS(stop)
	}()
}

func (m *HTTPMonitor) monitorWithSS(stop chan bool) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			// ss -tnp muestra conexiones TCP con información del proceso
			cmd := exec.Command("ss", "-tnp", "-o", "state", "established")
			output, err := cmd.Output()
			if err != nil {
				continue
			}

			m.parseSSOutput(string(output))
		}
	}
}

func (m *HTTPMonitor) parseSSOutput(output string) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if !strings.Contains(line, "ESTAB") {
			continue
		}

		// Buscar PID en la línea
		pidStr := fmt.Sprintf("pid=%d", m.targetPID)
		if !strings.Contains(line, pidStr) {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Parsear direcciones local y remota
		localAddr := fields[3]
		remoteAddr := fields[4]

		// Extraer información de bytes si está disponible
		var bytesSent, bytesRecv int64

		// ss puede mostrar bytes con la opción -i
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

		host, port := splitAddress(remoteAddr)

		connKey := fmt.Sprintf("%s->%s", localAddr, remoteAddr)
		m.tracker.UpdateConnection(connKey, &ConnectionInfo{
			Host:      host,
			Port:      port,
			BytesSent: bytesSent,
			BytesRecv: bytesRecv,
		})
	}
}

// ============== MONITOR TCPDUMP (Opcional, requiere sudo) ==============

func (m *HTTPMonitor) startTcpdumpMonitor() {
	stop := make(chan bool)
	m.stopChannels["tcpdump"] = stop

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		// Construir filtro para capturar solo tráfico del proceso
		// Nota: tcpdump no puede filtrar por PID directamente,
		// pero podemos capturar y filtrar después
		cmd := exec.Command("tcpdump", "-i", "any", "-n",
			"-s", "96", // Solo capturar headers
			"tcp and (port 80 or port 443 or port 8080 or port 3000)",
			"-l")

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Error iniciando tcpdump: %v", err)
			return
		}

		if err := cmd.Start(); err != nil {
			log.Printf("Error ejecutando tcpdump (necesita sudo): %v", err)
			return
		}

		scanner := bufio.NewScanner(stdout)
		go func() {
			for scanner.Scan() {
				select {
				case <-stop:
					cmd.Process.Kill()
					return
				default:
					line := scanner.Text()
					m.parseTcpdumpLine(line)
				}
			}
		}()

		<-stop
		cmd.Process.Kill()
	}()
}

func (m *HTTPMonitor) parseTcpdumpLine(line string) {
	// Ejemplo de línea tcpdump:
	// 14:23:45.123456 IP 192.168.1.100.54321 > 93.184.216.34.443: Flags [P.], seq 1:100, ack 1, win 502, length 99

	// Extraer timestamp
	parts := strings.Fields(line)
	if len(parts) < 7 {
		return
	}

	// Verificar si es tráfico HTTP/HTTPS
	if !strings.Contains(line, ".80:") && !strings.Contains(line, ".443:") &&
		!strings.Contains(line, ".8080:") && !strings.Contains(line, ".3000:") {
		return
	}

	// Extraer tamaño de datos
	var dataSize int64
	for i, part := range parts {
		if part == "length" && i+1 < len(parts) {
			size, err := strconv.ParseInt(parts[i+1], 10, 64)
			if err == nil {
				dataSize = size
			}
			break
		}
	}

	// Actualizar estadísticas de conexión
	if dataSize > 0 {
		// Aquí podrías correlacionar con las conexiones del proceso
		log.Printf("📦 Datos capturados: %d bytes", dataSize)
	}
}

// ============== CONNECTION TRACKER ==============

func (t *ConnectionTracker) AddConnection(key string, conn *ConnectionInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.connections[key] = conn
}

func (t *ConnectionTracker) UpdateConnection(key string, update *ConnectionInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if conn, exists := t.connections[key]; exists {
		if update.BytesSent > 0 {
			conn.BytesSent = update.BytesSent
		}
		if update.BytesRecv > 0 {
			conn.BytesRecv = update.BytesRecv
		}
	}
}

func (t *ConnectionTracker) UpdateBytes(key string, sent, recv int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if conn, exists := t.connections[key]; exists {
		conn.BytesSent = sent
		conn.BytesRecv = recv
	}
}

func (t *ConnectionTracker) CloseConnection(key string, conn *ConnectionInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Calcular duración
	duration := time.Since(conn.StartTime).Milliseconds()

	// Generar evento
	event := HTTPEvent{
		Timestamp:   time.Now(),
		Host:        conn.Host,
		Port:        conn.Port,
		Protocol:    conn.Protocol,
		BytesSent:   conn.BytesSent,
		BytesRecv:   conn.BytesRecv,
		Duration:    duration,
		ProcessName: conn.ProcessName,
		PID:         conn.PID,
	}

	// Determinar método aproximado por el patrón de bytes
	if conn.BytesSent > conn.BytesRecv {
		event.Method = "POST/PUT" // Probablemente enviando datos
	} else {
		event.Method = "GET" // Probablemente recibiendo datos
	}

	select {
	case t.events <- event:
	default:
	}

	delete(t.connections, key)

	// Imprimir resumen
	t.printEventSummary(event)
}

func (t *ConnectionTracker) printEventSummary(event HTTPEvent) {
	emoji := "🌐"
	if event.Protocol == "HTTPS" {
		emoji = "🔒"
	}

	methodColor := "\033[32m" // Verde para GET
	if strings.Contains(event.Method, "POST") {
		methodColor = "\033[33m" // Amarillo para POST
	}

	fmt.Printf("%s [%s] %s%s\033[0m %s:%d\n",
		emoji,
		event.Timestamp.Format("15:04:05"),
		methodColor,
		event.Method,
		event.Host,
		event.Port)

	fmt.Printf("  ⬆️  Enviado: %s\n", formatBytes(event.BytesSent))
	fmt.Printf("  ⬇️  Recibido: %s\n", formatBytes(event.BytesRecv))
	fmt.Printf("  ⏱️  Duración: %dms\n", event.Duration)
	fmt.Printf("  📱 Proceso: %s (PID: %d)\n\n", event.ProcessName, event.PID)
}

func (t *ConnectionTracker) GetEvents() <-chan HTTPEvent {
	return t.events
}

// ============== UTILIDADES ==============

func processExists(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

func findProcessByName(name string) (int, error) {
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

		if strings.Contains(strings.ToLower(string(cmdline)), strings.ToLower(name)) {
			return pid, nil
		}
	}

	return 0, fmt.Errorf("proceso '%s' no encontrado", name)
}

func canUseTcpdump() bool {
	// Verificar si tcpdump está disponible y si somos root
	if _, err := exec.LookPath("tcpdump"); err != nil {
		return false
	}
	return os.Geteuid() == 0
}

func parseHexAddress(hex string) string {
	parts := strings.Split(hex, ":")
	if len(parts) != 2 {
		return hex
	}

	ip, _ := strconv.ParseUint(parts[0], 16, 32)
	port, _ := strconv.ParseUint(parts[1], 16, 16)

	return fmt.Sprintf("%d.%d.%d.%d:%d",
		ip&0xFF, (ip>>8)&0xFF, (ip>>16)&0xFF, (ip>>24)&0xFF, port)
}

func getTCPState(hex string) string {
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

func splitAddress(addr string) (string, int) {
	lastColon := strings.LastIndex(addr, ":")
	if lastColon == -1 {
		return addr, 0
	}

	host := addr[:lastColon]
	portStr := addr[lastColon+1:]
	port, _ := strconv.Atoi(portStr)

	return host, port
}

func isHTTPPort(port int) bool {
	httpPorts := []int{80, 443, 8080, 8443, 3000, 5000, 8000, 9000}
	for _, p := range httpPorts {
		if p == port {
			return true
		}
	}
	return false
}

func parseProcIO(data string) map[string]int64 {
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

// ============== ESTADÍSTICAS ==============

type Statistics struct {
	mu          sync.RWMutex
	totalEvents int
	httpEvents  int
	httpsEvents int
	totalSent   int64
	totalRecv   int64
	startTime   time.Time
}

func NewStatistics() *Statistics {
	return &Statistics{
		startTime: time.Now(),
	}
}

func (s *Statistics) RecordEvent(event HTTPEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.totalEvents++
	if event.Protocol == "HTTPS" {
		s.httpsEvents++
	} else {
		s.httpEvents++
	}

	s.totalSent += event.BytesSent
	s.totalRecv += event.BytesRecv
}

func (s *Statistics) PrintSummary() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	duration := time.Since(s.startTime)

	fmt.Println("\n========== RESUMEN DE ACTIVIDAD HTTP/HTTPS ==========")
	fmt.Printf("Tiempo de monitoreo: %s\n", duration.Round(time.Second))
	fmt.Printf("\nConexiones totales: %d\n", s.totalEvents)
	fmt.Printf("  HTTP:  %d\n", s.httpEvents)
	fmt.Printf("  HTTPS: %d\n", s.httpsEvents)
	fmt.Printf("\nTráfico total:\n")
	fmt.Printf("  ⬆️  Enviado:  %s\n", formatBytes(s.totalSent))
	fmt.Printf("  ⬇️  Recibido: %s\n", formatBytes(s.totalRecv))
	fmt.Printf("  📊 Total:    %s\n", formatBytes(s.totalSent+s.totalRecv))

	if s.totalEvents > 0 {
		avgSent := s.totalSent / int64(s.totalEvents)
		avgRecv := s.totalRecv / int64(s.totalEvents)
		fmt.Printf("\nPromedio por conexión:\n")
		fmt.Printf("  Enviado:  %s\n", formatBytes(avgSent))
		fmt.Printf("  Recibido: %s\n", formatBytes(avgRecv))
	}
}

// ============== MAIN ==============

func main() {
	fmt.Println("🔍 Monitor HTTP/HTTPS - Sin proxy")
	fmt.Println("==================================")

	// Verificar argumentos
	if len(os.Args) < 2 {
		fmt.Println("\nUso:")
		fmt.Println("  Sin sudo (limitado): ./monitor <nombre_proceso o PID>")
		fmt.Println("  Con sudo (completo): sudo ./monitor <nombre_proceso o PID>")
		fmt.Println("\nEjemplos:")
		fmt.Println("  ./monitor firefox")
		fmt.Println("  ./monitor 1234")
		fmt.Println("  sudo ./monitor chrome")
		fmt.Println("\nNota: Con sudo se obtiene información más detallada")
		os.Exit(1)
	}

	var targetPID int
	var targetName string

	// Parsear argumento
	arg := os.Args[1]
	if pid, err := strconv.Atoi(arg); err == nil {
		targetPID = pid
	} else {
		targetName = arg
	}

	// Informar capacidades según permisos
	if os.Geteuid() == 0 {
		fmt.Println("✅ Ejecutando con sudo - Captura completa habilitada")
	} else {
		fmt.Println("⚠️  Ejecutando sin sudo - Captura limitada")
		fmt.Println("   Solo se monitorizarán conexiones TCP del proceso")
		fmt.Println("   Para captura detallada de paquetes, ejecute con sudo")
	}

	// Crear monitor
	monitor := NewHTTPMonitor(targetPID, targetName)

	// Iniciar monitoreo
	if err := monitor.Start(); err != nil {
		log.Fatal(err)
	}

	// Estadísticas
	stats := NewStatistics()

	// Procesar eventos
	go func() {
		for event := range monitor.tracker.GetEvents() {
			stats.RecordEvent(event)
		}
	}()

	fmt.Println("\n📊 Monitoreo iniciado. Presiona Ctrl+C para detener...")
	fmt.Println("Se mostrarán conexiones HTTP/HTTPS cuando se cierren.\n")

	// Manejar señales
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Esperar señal
	<-sigChan

	fmt.Println("\n⏹️  Deteniendo monitor...")

	// Detener monitor
	monitor.Stop()

	// Mostrar estadísticas finales
	stats.PrintSummary()

	fmt.Println("\n✅ Monitor detenido")
}

func (m *HTTPMonitor) Stop() {
	// Cerrar todos los canales de stop
	for _, stop := range m.stopChannels {
		close(stop)
	}

	// Esperar a que terminen las goroutines
	m.wg.Wait()

	// Cerrar canal de eventos
	close(m.tracker.events)
}
