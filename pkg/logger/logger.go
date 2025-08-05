package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

/**
 * AGENT:     architecture-designer
 * TRACE:     CLAUDE-ARCH-044
 * CONTEXT:   Structured logging implementation for system-wide logging consistency
 * REASON:    Need consistent logging across all components with proper level filtering
 * CHANGE:    Initial implementation.
 * PREVENTION:Keep logging lightweight and avoid logging sensitive data
 * RISK:      Low - Logging failures should not affect core system functionality
 */

// LogLevel represents different logging levels
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns string representation of log level
func (ll LogLevel) String() string {
	switch ll {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// DefaultLogger implements the Logger interface
type DefaultLogger struct {
	component string
	level     LogLevel
	logger    *log.Logger
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger(component, levelStr string) *DefaultLogger {
	level := parseLogLevel(levelStr)
	
	logger := log.New(os.Stdout, "", 0) // No default prefix, we'll format ourselves
	
	return &DefaultLogger{
		component: component,
		level:     level,
		logger:    logger,
	}
}

// parseLogLevel converts string to LogLevel
func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// formatMessage formats log message with timestamp and structured fields
func (dl *DefaultLogger) formatMessage(level LogLevel, msg string, fields ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	
	// Build structured fields
	var fieldStr strings.Builder
	if len(fields) > 0 {
		fieldStr.WriteString(" |")
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				fieldStr.WriteString(fmt.Sprintf(" %s=%v", fields[i], fields[i+1]))
			}
		}
	}
	
	return fmt.Sprintf("[%s] %s [%s] %s%s", 
		timestamp, level.String(), dl.component, msg, fieldStr.String())
}

// shouldLog checks if message should be logged based on level
func (dl *DefaultLogger) shouldLog(level LogLevel) bool {
	return level >= dl.level
}

// Debug logs a debug message
func (dl *DefaultLogger) Debug(msg string, fields ...interface{}) {
	if dl.shouldLog(LevelDebug) {
		dl.logger.Println(dl.formatMessage(LevelDebug, msg, fields...))
	}
}

// Info logs an info message
func (dl *DefaultLogger) Info(msg string, fields ...interface{}) {
	if dl.shouldLog(LevelInfo) {
		dl.logger.Println(dl.formatMessage(LevelInfo, msg, fields...))
	}
}

// Warn logs a warning message
func (dl *DefaultLogger) Warn(msg string, fields ...interface{}) {
	if dl.shouldLog(LevelWarn) {
		dl.logger.Println(dl.formatMessage(LevelWarn, msg, fields...))
	}
}

// Error logs an error message
func (dl *DefaultLogger) Error(msg string, fields ...interface{}) {
	if dl.shouldLog(LevelError) {
		dl.logger.Println(dl.formatMessage(LevelError, msg, fields...))
	}
}

// Fatal logs a fatal message and exits
func (dl *DefaultLogger) Fatal(msg string, fields ...interface{}) {
	dl.logger.Println(dl.formatMessage(LevelFatal, msg, fields...))
	os.Exit(1)
}