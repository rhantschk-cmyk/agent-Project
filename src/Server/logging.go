package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Simple logging helper that writes to stdout and, if enabled, to a log file.
//
// LogScope describes where an output originated from (e.g. "[CLI]", "[Memory]",
// "[Monitoring]", ...). Passing an empty scope keeps the output short.

var (
	logMutex  sync.Mutex
	logFile   *os.File
	logToFile bool
)

// logInit opens the log file at the given path (creating parent dirs).
// If path is empty, logging stays stdout-only.
func logInit(path string) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("could not create log dir %q: %w", dir, err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("could not open log file %q: %w", path, err)
	}
	logMutex.Lock()
	logFile = f
	logToFile = true
	logMutex.Unlock()
	return nil
}

// logClose closes the log file. Safe to call multiple times.
func logClose() {
	logMutex.Lock()
	defer logMutex.Unlock()
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
		logToFile = false
	}
}

// logf logs a line with a timestamp to stdout and, if enabled, the log file.
func logf(format string, a ...any) {
	line := fmt.Sprintf(format, a...)
	ts := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] %s\n", ts, line)

	logMutex.Lock()
	defer logMutex.Unlock()

	fmt.Print(entry)
	if logToFile && logFile != nil {
		_, _ = logFile.WriteString(entry)
	}
}
