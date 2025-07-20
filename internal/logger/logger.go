// Package logger provides structured logging functionality for menv.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// Log levels
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

// Logger represents a logger instance
type Logger struct {
	level      LogLevel
	mu         sync.Mutex
	outputs    map[LogLevel][]io.Writer
	timeFormat string
	showFile   bool
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	once.Do(func() {
		defaultLogger = NewLogger(INFO)
		defaultLogger.AddOutput(INFO, os.Stdout)
		defaultLogger.AddOutput(WARN, os.Stdout)
		defaultLogger.AddOutput(ERROR, os.Stderr)
	})
	return defaultLogger
}

// NewLogger creates a new logger instance with the specified minimum log level
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level:      level,
		outputs:    make(map[LogLevel][]io.Writer),
		timeFormat: "2006-01-02 15:04:05",
		showFile:   true,
	}
}

// SetLevel changes the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetTimeFormat sets the time format string used in log messages
func (l *Logger) SetTimeFormat(format string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeFormat = format
}

// SetShowFile enables or disables showing file and line information in logs
func (l *Logger) SetShowFile(show bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.showFile = show
}

// AddOutput adds an output writer for the specified log level
func (l *Logger) AddOutput(level LogLevel, w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.outputs[level] = append(l.outputs[level], w)
}

// AddFileOutput adds a file output for the specified log level
func (l *Logger) AddFileOutput(level LogLevel, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.AddOutput(level, file)
	return nil
}

// getCallerInfo returns the file and line number of the caller
func getCallerInfo() string {
	_, file, line, ok := runtime.Caller(4) // Skip getCallerInfo, formatMessage, log, and the actual logging function
	if !ok {
		return "???:0"
	}
	// Get just the file name, not the full path
	file = filepath.Base(file)
	return fmt.Sprintf("%s:%d", file, line)
}

// formatMessage formats a log message with timestamp, level, and caller info
func (l *Logger) formatMessage(level LogLevel, msg string) string {
	timestamp := time.Now().Format(l.timeFormat)
	levelName := levelNames[level]

	var caller string
	if l.showFile {
		caller = getCallerInfo()
	}

	if l.showFile {
		return fmt.Sprintf("%s [%s] %s - %s", timestamp, levelName, caller, msg)
	}
	return fmt.Sprintf("%s [%s] %s", timestamp, levelName, msg)
}

// logf writes a message to all configured outputs for the given level
func (l *Logger) logf(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Format the message
	var msg string
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	} else {
		msg = format
	}

	formattedMsg := l.formatMessage(level, msg)

	// Write to all outputs for this level and above
	for lvl, writers := range l.outputs {
		if lvl >= level {
			for _, w := range writers {
				if _, err := fmt.Fprintln(w, formattedMsg); err != nil {
					// Best effort logging - continue even if write fails
					continue
				}
			}
		}
	}
}

// Debugf logs a debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logf(DEBUG, format, args...)
}

// Infof logs an info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logf(INFO, format, args...)
}

// Warnf logs a warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logf(WARN, format, args...)
}

// Errorf logs an error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logf(ERROR, format, args...)
}

// Global convenience functions that use the default logger

func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// SetGlobalLevel sets the level for the default logger
func SetGlobalLevel(level LogLevel) {
	GetLogger().SetLevel(level)
}
