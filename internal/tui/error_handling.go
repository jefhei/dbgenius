package tui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
)

// ErrorLogger provides centralized error logging for the TUI application.
// Logs are written to ~/.dbgenius/error.log with timestamps.
type ErrorLogger struct {
	mu     sync.Mutex
	logger *log.Logger
	file   *os.File
}

var (
	globalErrorLogger *ErrorLogger
	loggerOnce        sync.Once
)

// getErrorLogger returns the singleton error logger instance.
func getErrorLogger() *ErrorLogger {
	loggerOnce.Do(func() {
		globalErrorLogger = &ErrorLogger{}
		globalErrorLogger.init()
	})
	return globalErrorLogger
}

// init opens (or creates) the error log file.
func (el *ErrorLogger) init() {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: use current directory
		home = "."
	}

	logDir := filepath.Join(home, ".dbgenius")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Can't create log directory — write to stderr
		return
	}

	logPath := filepath.Join(logDir, "error.log")
	el.file, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}

	el.logger = log.New(el.file, "", log.LstdFlags)
}

// LogError writes an error message with context to the log file.
func (el *ErrorLogger) LogError(context string, err error) {
	el.mu.Lock()
	defer el.mu.Unlock()

	if el.logger != nil {
		el.logger.Printf("[ERROR] %s: %v", context, err)
	}
}

// LogPanic logs a panic with its stack trace.
func (el *ErrorLogger) LogPanic(context string, recovered interface{}) {
	el.mu.Lock()
	defer el.mu.Unlock()

	if el.logger != nil {
		el.logger.Printf("[PANIC] %s: %v\nStack:\n%s", context, recovered, string(debug.Stack()))
	}
}

// Close closes the log file.
func (el *ErrorLogger) Close() {
	el.mu.Lock()
	defer el.mu.Unlock()
	if el.file != nil {
		el.file.Close()
	}
}

// safeGo runs a function in a goroutine with panic recovery.
// If the function panics, the error is logged and a friendly message is
// sent to the Bubble Tea program.
func safeGo(fn func(), context string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				getErrorLogger().LogPanic(context, r)
			}
		}()
		fn()
	}()
}

// wrapErrorf creates an error with context, logs it, and returns it.
func wrapErrorf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	wrapped := fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
	getErrorLogger().LogError(fmt.Sprintf(format, args...), err)
	return wrapped
}
