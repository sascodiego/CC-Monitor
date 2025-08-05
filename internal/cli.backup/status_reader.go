/**
 * AGENT:     cli-interface
 * TRACE:     CLAUDE-CLI-STATUS-001
 * CONTEXT:   Status file reader for simplified daemon communication
 * REASON:    Need way to read daemon status for CLI display during testing
 * CHANGE:    New implementation for file-based status communication.
 * PREVENTION:Validate file format and handle missing/corrupt files gracefully
 * RISK:      Low - File-based communication is simple and reliable for testing
 */
package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/claude-monitor/claude-monitor/internal/domain"
)

type DaemonStatus struct {
	DaemonRunning     bool                 `json:"daemonRunning"`
	Timestamp         time.Time            `json:"timestamp"`
	CurrentSession    *domain.Session      `json:"currentSession"`
	CurrentWorkBlock  *domain.WorkBlock    `json:"currentWorkBlock"`
	MonitoringActive  bool                 `json:"monitoringActive"`
}

func ReadDaemonStatus(statusFile string) (*DaemonStatus, error) {
	// Check if status file exists
	if _, err := os.Stat(statusFile); os.IsNotExist(err) {
		return &DaemonStatus{
			DaemonRunning:    false,
			MonitoringActive: false,
			Timestamp:        time.Now(),
		}, nil
	}

	// Read status file
	data, err := ioutil.ReadFile(statusFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read status file: %w", err)
	}

	var status DaemonStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status file: %w", err)
	}

	// Check if status is recent (within last 30 seconds)
	if time.Since(status.Timestamp) > 30*time.Second {
		status.DaemonRunning = false
		status.MonitoringActive = false
	}

	return &status, nil
}