// Package logger contains tests for logger functionality.
package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(DEBUG)
	logger.AddOutput(DEBUG, &buf)
	logger.AddOutput(INFO, &buf)
	logger.AddOutput(WARN, &buf)
	logger.AddOutput(ERROR, &buf)

	tests := []struct {
		level   LogLevel
		message string
	}{
		{DEBUG, "debug message"},
		{INFO, "info message"},
		{WARN, "warning message"},
		{ERROR, "error message"},
	}

	for _, tt := range tests {
		buf.Reset()

		switch tt.level {
		case DEBUG:
			logger.Debugf(tt.message)
		case INFO:
			logger.Infof(tt.message)
		case WARN:
			logger.Warnf(tt.message)
		case ERROR:
			logger.Errorf(tt.message)
		}

		output := buf.String()
		if !strings.Contains(output, tt.message) {
			t.Errorf("Expected log to contain %q, got %q", tt.message, output)
		}
		if !strings.Contains(output, levelNames[tt.level]) {
			t.Errorf("Expected log to contain level %q, got %q", levelNames[tt.level], output)
		}
	}
}

func TestLogLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(INFO)
	logger.AddOutput(DEBUG, &buf)
	logger.AddOutput(INFO, &buf)

	// Debug shouldn't log when level is INFO
	logger.Debugf("debug message")
	if buf.String() != "" {
		t.Error("Expected no debug output when level is INFO")
	}

	// Info should log
	buf.Reset()
	logger.Infof("info message")
	if buf.String() == "" {
		t.Error("Expected info output")
	}
}

func TestFormatting(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(INFO)
	logger.AddOutput(INFO, &buf)

	// Test with format string
	logger.Infof("Count: %d", 42)
	output := buf.String()
	if !strings.Contains(output, "Count: 42") {
		t.Errorf("Expected formatted message, got %q", output)
	}
}

func TestMultipleOutputs(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	logger := NewLogger(INFO)
	logger.AddOutput(INFO, &buf1)
	logger.AddOutput(INFO, &buf2)

	message := "test message"
	logger.Infof("%s", message)

	if !strings.Contains(buf1.String(), message) {
		t.Error("Expected message in first buffer")
	}
	if !strings.Contains(buf2.String(), message) {
		t.Error("Expected message in second buffer")
	}
}

func TestShowFile(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(INFO)
	logger.AddOutput(INFO, &buf)

	// Test with file info
	logger.SetShowFile(true)
	logger.Infof("test message")
	if !strings.Contains(buf.String(), "logger_test.go:") {
		t.Error("Expected file information in log")
	}

	// Test without file info
	buf.Reset()
	logger.SetShowFile(false)
	logger.Infof("test message")
	if strings.Contains(buf.String(), "logger_test.go:") {
		t.Error("Expected no file information in log")
	}
}
