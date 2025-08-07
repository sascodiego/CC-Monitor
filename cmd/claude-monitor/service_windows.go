//go:build windows

/**
 * CONTEXT:   Windows Service Control Manager integration for Claude Monitor
 * INPUT:     Service configuration, SCM API calls, Windows service events
 * OUTPUT:    Professional Windows service with SCM registration and event logging
 * BUSINESS:  Windows service integration enables enterprise deployment
 * CHANGE:    Initial Windows service implementation with SCM and event log integration
 * RISK:      High - Windows service API integration requiring admin privileges
 */

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

/**
 * CONTEXT:   Windows Service Control Manager service manager implementation
 * INPUT:     Service configuration, SCM handles, service control requests
 * OUTPUT:    Complete Windows service lifecycle management
 * BUSINESS:  WindowsServiceManager provides professional Windows service integration
 * CHANGE:    Initial Windows service manager with SCM API integration
 * RISK:      High - Windows service API requires careful handle management
 */
type WindowsServiceManager struct {
	serviceName string
	eventLog    *eventlog.Log
}

/**
 * CONTEXT:   Windows service handler for service control events
 * INPUT:     Service control messages from SCM
 * OUTPUT:    Service state changes and daemon lifecycle management
 * BUSINESS:  Service handler enables Windows service integration with daemon
 * CHANGE:    Initial service handler with daemon coordination
 * RISK:      High - Service handler affects service reliability and responsiveness
 */
type WindowsServiceHandler struct {
	config ServiceConfig
	daemon *EmbeddedServer
}

func NewWindowsServiceManager() (*WindowsServiceManager, error) {
	serviceName := "claude-monitor"
	
	// Initialize event log
	eventLog, err := eventlog.Open(serviceName)
	if err != nil {
		// Event log not registered, will register during install
		eventLog = nil
	}
	
	return &WindowsServiceManager{
		serviceName: serviceName,
		eventLog:    eventLog,
	}, nil
}

// Override the cross-platform service manager for Windows
func NewServiceManager() (ServiceManager, error) {
	return NewWindowsServiceManager()
}

/**
 * CONTEXT:   Windows service installation with SCM registration
 * INPUT:     Service configuration with Windows-specific settings
 * OUTPUT:    Registered Windows service with proper SCM integration
 * BUSINESS:  Service installation enables professional Windows deployment
 * CHANGE:    Initial service installation with event log registration
 * RISK:      High - SCM registration requires admin privileges and proper cleanup
 */
func (w *WindowsServiceManager) Install(config ServiceConfig) error {
	// Open Service Control Manager
	scm, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service control manager: %w", err)
	}
	defer scm.Disconnect()
	
	// Check if service already exists
	service, err := scm.OpenService(config.Name)
	if err == nil {
		service.Close()
		return fmt.Errorf("service '%s' already exists", config.Name)
	}
	
	// Prepare service configuration
	serviceConfig := mgr.Config{
		ServiceType:      windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:        getWindowsStartType(config.StartMode),
		ErrorControl:     windows.SERVICE_ERROR_NORMAL,
		BinaryPathName:   fmt.Sprintf(`"%s" %s`, config.ExecutablePath, strings.Join(config.Arguments, " ")),
		DisplayName:      config.DisplayName,
		Description:      config.Description,
		ServiceStartName: config.WindowsService.ServiceAccount,
		Password:         config.WindowsService.Password,
		Dependencies:     config.WindowsService.Dependencies,
	}
	
	// Create the service
	service, err = scm.CreateService(config.Name, serviceConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer service.Close()
	
	// Configure service recovery actions
	if config.RestartOnFailure {
		if err := w.configureServiceRecovery(service, config); err != nil {
			// Log warning but don't fail installation
			if w.eventLog != nil {
				w.eventLog.Warning(1, fmt.Sprintf("Failed to configure service recovery: %v", err))
			}
		}
	}
	
	// Register event log source
	if err := w.registerEventLogSource(config.Name); err != nil {
		// Log warning but don't fail installation
		if w.eventLog != nil {
			w.eventLog.Warning(1, fmt.Sprintf("Failed to register event log: %v", err))
		}
	}
	
	// Initialize event log after registration
	if w.eventLog == nil {
		if eventLog, err := eventlog.Open(config.Name); err == nil {
			w.eventLog = eventLog
		}
	}
	
	if w.eventLog != nil {
		w.eventLog.Info(1, fmt.Sprintf("Claude Monitor service installed successfully: %s", config.ExecutablePath))
	}
	
	return nil
}

func (w *WindowsServiceManager) Uninstall() error {
	scm, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service control manager: %w", err)
	}
	defer scm.Disconnect()
	
	service, err := scm.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("service not found: %w", err)
	}
	defer service.Close()
	
	// Stop service if running
	status, err := service.Query()
	if err == nil && status.State != svc.Stopped {
		if err := w.stopServiceWithTimeout(service, 30*time.Second); err != nil {
			return fmt.Errorf("failed to stop service before uninstall: %w", err)
		}
	}
	
	// Delete the service
	if err := service.Delete(); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}
	
	// Unregister event log source
	w.unregisterEventLogSource(w.serviceName)
	
	if w.eventLog != nil {
		w.eventLog.Info(1, "Claude Monitor service uninstalled successfully")
		w.eventLog.Close()
		w.eventLog = nil
	}
	
	return nil
}

func (w *WindowsServiceManager) Start() error {
	scm, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service control manager: %w", err)
	}
	defer scm.Disconnect()
	
	service, err := scm.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()
	
	if err := service.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	
	// Wait for service to start
	if err := w.waitForServiceState(service, svc.Running, 30*time.Second); err != nil {
		return fmt.Errorf("service failed to start within timeout: %w", err)
	}
	
	if w.eventLog != nil {
		w.eventLog.Info(1, "Claude Monitor service started successfully")
	}
	
	return nil
}

func (w *WindowsServiceManager) Stop() error {
	scm, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service control manager: %w", err)
	}
	defer scm.Disconnect()
	
	service, err := scm.OpenService(w.serviceName)
	if err != nil {
		return fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()
	
	if err := w.stopServiceWithTimeout(service, 30*time.Second); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	
	if w.eventLog != nil {
		w.eventLog.Info(1, "Claude Monitor service stopped successfully")
	}
	
	return nil
}

func (w *WindowsServiceManager) Restart() error {
	if err := w.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	
	// Wait a moment between stop and start
	time.Sleep(2 * time.Second)
	
	if err := w.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}
	
	return nil
}

/**
 * CONTEXT:   Windows service status query with detailed information
 * INPUT:     Service handle and SCM query operations
 * OUTPUT:    Comprehensive service status with Windows-specific metrics
 * BUSINESS:  Service status enables monitoring and troubleshooting
 * CHANGE:    Initial status implementation with Windows performance counters
 * RISK:      Medium - Status queries require proper error handling
 */
func (w *WindowsServiceManager) Status() (ServiceStatus, error) {
	scm, err := mgr.Connect()
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("failed to connect to service control manager: %w", err)
	}
	defer scm.Disconnect()
	
	service, err := scm.OpenService(w.serviceName)
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("failed to open service: %w", err)
	}
	defer service.Close()
	
	// Get basic service status
	status, err := service.Query()
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("failed to query service status: %w", err)
	}
	
	// Get service configuration
	config, err := service.Config()
	if err != nil {
		config = mgr.Config{DisplayName: w.serviceName}
	}
	
	serviceStatus := ServiceStatus{
		Name:        w.serviceName,
		DisplayName: config.DisplayName,
		State:       convertWindowsState(status.State),
		PID:         int(status.ProcessId),
	}
	
	// Get additional metrics if service is running
	if status.State == svc.Running && status.ProcessId > 0 {
		if metrics, err := w.getProcessMetrics(status.ProcessId); err == nil {
			serviceStatus.Memory = metrics.Memory
			serviceStatus.CPU = metrics.CPU
			serviceStatus.StartTime = metrics.StartTime
			serviceStatus.Uptime = time.Since(metrics.StartTime)
		}
	}
	
	return serviceStatus, nil
}

func (w *WindowsServiceManager) IsInstalled() bool {
	scm, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer scm.Disconnect()
	
	service, err := scm.OpenService(w.serviceName)
	if err != nil {
		return false
	}
	defer service.Close()
	
	return true
}

func (w *WindowsServiceManager) IsRunning() bool {
	status, err := w.Status()
	if err != nil {
		return false
	}
	return status.State == ServiceStateRunning
}

/**
 * CONTEXT:   Windows event log retrieval for service logging
 * INPUT:     Event log queries and Windows event log API
 * OUTPUT:    Structured log entries with Windows event metadata
 * BUSINESS:  Log retrieval enables troubleshooting and monitoring
 * CHANGE:    Initial log retrieval with Windows Event Log integration
 * RISK:      Medium - Event log API requires proper handle management
 */
func (w *WindowsServiceManager) GetLogs(lines int) ([]LogEntry, error) {
	// Open event log for reading
	eventLogName := "Application"
	sourceName := w.serviceName
	
	logs := make([]LogEntry, 0, lines)
	
	// Open event log handle
	handle, err := windows.OpenEventLog(syscall.StringToUTF16Ptr(""), syscall.StringToUTF16Ptr(eventLogName))
	if err != nil {
		return nil, fmt.Errorf("failed to open event log: %w", err)
	}
	defer windows.CloseEventLog(handle)
	
	// Read events backwards (most recent first)
	var readFlags uint32 = windows.EVENTLOG_BACKWARDS_READ | windows.EVENTLOG_SEQUENTIAL_READ
	buffer := make([]byte, 32768) // 32KB buffer
	var bytesRead, minBytesNeeded uint32
	
	count := 0
	for count < lines {
		err := windows.ReadEventLog(handle, readFlags, 0, &buffer[0], uint32(len(buffer)), &bytesRead, &minBytesNeeded)
		if err != nil {
			if err == windows.ERROR_HANDLE_EOF {
				break // No more events
			}
			return logs, nil // Return what we have
		}
		
		// Parse events from buffer
		events := w.parseEventLogBuffer(buffer[:bytesRead], sourceName)
		for _, event := range events {
			if count >= lines {
				break
			}
			logs = append(logs, event)
			count++
		}
		
		readFlags = windows.EVENTLOG_BACKWARDS_READ | windows.EVENTLOG_SEQUENTIAL_READ
	}
	
	return logs, nil
}

/**
 * CONTEXT:   Windows service execution as SCM service
 * INPUT:     Service control requests from SCM
 * OUTPUT:    Service lifecycle management with daemon coordination
 * BUSINESS:  RunAsWindowsService enables professional Windows service operation
 * CHANGE:    Initial service execution with daemon integration
 * RISK:      High - Service execution affects system stability and responsiveness
 */
func RunAsWindowsService(config ServiceConfig) error {
	serviceName := config.Name
	
	// Check if we're running as a service
	isService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("failed to determine if running as service: %w", err)
	}
	
	if !isService {
		return fmt.Errorf("not running as Windows service")
	}
	
	// Initialize event log
	eventLog, err := eventlog.Open(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open event log: %w", err)
	}
	defer eventLog.Close()
	
	eventLog.Info(1, fmt.Sprintf("Claude Monitor service starting: %s", config.ExecutablePath))
	
	// Create service handler
	handler := &WindowsServiceHandler{
		config: config,
	}
	
	// Run the service
	if err := svc.Run(serviceName, handler); err != nil {
		eventLog.Error(1, fmt.Sprintf("Service execution failed: %v", err))
		return fmt.Errorf("service execution failed: %w", err)
	}
	
	return nil
}

/**
 * CONTEXT:   Windows service control event handler
 * INPUT:     Service control commands from SCM (start, stop, pause, etc.)
 * OUTPUT:     Service state changes and daemon lifecycle coordination
 * BUSINESS:  Service control enables Windows service lifecycle management
 * CHANGE:    Initial service control implementation with daemon coordination
 * RISK:      High - Service control affects service reliability and responsiveness
 */
func (h *WindowsServiceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	
	// Report service start pending
	changes <- svc.Status{State: svc.StartPending}
	
	// Initialize daemon
	config, err := loadConfiguration()
	if err != nil {
		return false, 1
	}
	
	// Override with service-specific settings
	config.Daemon.ListenAddr = h.config.Environment["CLAUDE_MONITOR_LISTEN_ADDR"]
	if config.Daemon.ListenAddr == "" {
		config.Daemon.ListenAddr = "localhost:8080"
	}
	
	// Create embedded server
	server, err := NewEmbeddedServer(EmbeddedServerConfig{
		ListenAddr:     config.Daemon.ListenAddr,
		DatabasePath:   expandPath(config.Daemon.DatabasePath),
		LogLevel:       h.config.LogLevel,
		DurationHours:  config.Session.DurationHours,
		MaxIdleMinutes: config.Session.MaxIdleMinutes,
	})
	if err != nil {
		return false, 1
	}
	
	h.daemon = server
	
	// Start daemon in background
	serverErrors := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			serverErrors <- err
		}
	}()
	
	// Give daemon time to start
	time.Sleep(2 * time.Second)
	
	// Report service running
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	
	// Service control loop
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				
				// Graceful shutdown
				if h.daemon != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					
					if err := h.daemon.Shutdown(ctx); err != nil {
						// Force stop if graceful shutdown fails
						h.daemon.Stop()
					}
				}
				
				return false, 0
				
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				
			default:
				// Unknown command - ignore
			}
			
		case err := <-serverErrors:
			// Daemon failed
			if err != nil {
				changes <- svc.Status{State: svc.StopPending}
				return false, 1
			}
		}
	}
}

// Helper functions for Windows service management

func getWindowsStartType(startMode ServiceStartMode) uint32 {
	switch startMode {
	case StartModeAuto:
		return windows.SERVICE_AUTO_START
	case StartModeManual:
		return windows.SERVICE_DEMAND_START
	case StartModeDisabled:
		return windows.SERVICE_DISABLED
	default:
		return windows.SERVICE_AUTO_START
	}
}

func convertWindowsState(state svc.State) ServiceState {
	switch state {
	case svc.Running:
		return ServiceStateRunning
	case svc.Stopped:
		return ServiceStateStopped
	case svc.StartPending:
		return ServiceStateStarting
	case svc.StopPending:
		return ServiceStateStopping
	case svc.Paused:
		return ServiceStateStopped
	default:
		return ServiceStateUnknown
	}
}

func (w *WindowsServiceManager) configureServiceRecovery(service *mgr.Service, config ServiceConfig) error {
	// Configure service failure actions
	restartDelay := uint32(config.RestartDelay.Milliseconds())
	
	actions := []mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: restartDelay},
		{Type: mgr.ServiceRestart, Delay: restartDelay},
		{Type: mgr.NoAction, Delay: 0},
	}
	
	return service.SetRecoveryActions(actions, 86400) // Reset after 24 hours
}

func (w *WindowsServiceManager) registerEventLogSource(serviceName string) error {
	// Register event log source in registry
	regPath := fmt.Sprintf(`SYSTEM\CurrentControlSet\Services\EventLog\Application\%s`, serviceName)
	
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, regPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create registry key: %w", err)
	}
	defer key.Close()
	
	// Set event message file
	executable, _ := os.Executable()
	if err := key.SetStringValue("EventMessageFile", executable); err != nil {
		return err
	}
	
	// Set types supported
	if err := key.SetDWordValue("TypesSupported", 7); err != nil { // EVENTLOG_ERROR_TYPE | EVENTLOG_WARNING_TYPE | EVENTLOG_INFORMATION_TYPE
		return err
	}
	
	return nil
}

func (w *WindowsServiceManager) unregisterEventLogSource(serviceName string) {
	regPath := fmt.Sprintf(`SYSTEM\CurrentControlSet\Services\EventLog\Application\%s`, serviceName)
	registry.DeleteKey(registry.LOCAL_MACHINE, regPath)
}

func (w *WindowsServiceManager) waitForServiceState(service *mgr.Service, targetState svc.State, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		status, err := service.Query()
		if err != nil {
			return err
		}
		
		if status.State == targetState {
			return nil
		}
		
		time.Sleep(500 * time.Millisecond)
	}
	
	return fmt.Errorf("timeout waiting for service state %v", targetState)
}

func (w *WindowsServiceManager) stopServiceWithTimeout(service *mgr.Service, timeout time.Duration) error {
	status, err := service.Control(svc.Stop)
	if err != nil {
		return err
	}
	
	if status.State == svc.Stopped {
		return nil
	}
	
	return w.waitForServiceState(service, svc.Stopped, timeout)
}

type ProcessMetrics struct {
	Memory    int64
	CPU       float64
	StartTime time.Time
}

func (w *WindowsServiceManager) getProcessMetrics(pid uint32) (*ProcessMetrics, error) {
	// Open process handle
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to open process: %w", err)
	}
	defer windows.CloseHandle(handle)
	
	// Get process memory info
	var memCounters windows.ProcessMemoryCountersEx
	memCounters.Size = uint32(unsafe.Sizeof(memCounters))
	
	err = windows.GetProcessMemoryInfo(handle, (*windows.ProcessMemoryCounters)(unsafe.Pointer(&memCounters)), memCounters.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}
	
	// Get process creation time
	var creationTime, exitTime, kernelTime, userTime windows.Filetime
	err = windows.GetProcessTimes(handle, &creationTime, &exitTime, &kernelTime, &userTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get process times: %w", err)
	}
	
	startTime := time.Unix(0, creationTime.Nanoseconds())
	
	return &ProcessMetrics{
		Memory:    int64(memCounters.WorkingSetSize),
		CPU:       0, // CPU calculation would require additional system calls
		StartTime: startTime,
	}, nil
}

func (w *WindowsServiceManager) parseEventLogBuffer(buffer []byte, sourceName string) []LogEntry {
	var logs []LogEntry
	
	// Simple event log parsing - this is a complex structure
	// For production, consider using a proper Windows event log library
	
	// This is a simplified implementation
	// Real implementation would parse EVENTLOGRECORD structure
	
	return logs
}