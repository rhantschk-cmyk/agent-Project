package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const version = "v0.4 (Standard)"

func main() {
	fmt.Println("==============================================")
	fmt.Printf("  VaultAgent %s\n", version)
	fmt.Println("  Pro Version coming soon — unlimited monitors")
	fmt.Println("  https://github.com/rhantschk-cmyk/agent-Project")
	fmt.Println("==============================================")

	// Logging: stdout + file (path from env, optional)
	logPath := os.Getenv("AGENT_LOG_FILE")
	if err := logInit(logPath); err != nil {
		logf("-> WARN: %v", err)
	}
	defer logClose()

	logf("-> Application started")

	// Load Config-File
	logf("-> Loading Config-File")
	cfg, err := LoadConfig("config.json")
	if err != nil {
		logf("-> FATAL: Could not load Config-File: %v", err)
		os.Exit(1)
	}
	logf("-> Done")

	// Graceful shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	StartMemoryCompressor(ctx, cfg) // No Config Problems
	go startCLIServer(ctx, cfg)     // includes /health on :8080
	go startMonitorServer()         // :9000

	MailSection(ctx, cfg) // No Config Problems

	logf("-> Shutting down")
}
