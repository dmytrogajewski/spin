package tui

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

var debugLogger *slog.Logger

// InitDebugLogging initializes file-based debug logging.
func InitDebugLogging() error {
	// Create logs directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	logDir := filepath.Join(homeDir, ".spin", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFile := filepath.Join(logDir, fmt.Sprintf("tui_%s.log", timestamp))

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	// Create structured logger
	debugLogger = slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	debugLogger.Info("TUI debug logging initialized", "log_file", logFile)
	fmt.Fprintf(os.Stderr, "Debug logs: %s\n", logFile)

	return nil
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	if debugLogger != nil {
		debugLogger.Debug(msg, args...)
	}
}

// Info logs an info message.
func Info(msg string, args ...any) {
	if debugLogger != nil {
		debugLogger.Info(msg, args...)
	}
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	if debugLogger != nil {
		debugLogger.Warn(msg, args...)
	}
}

// Error logs an error message.
func Error(msg string, args ...any) {
	if debugLogger != nil {
		debugLogger.Error(msg, args...)
	}
}
