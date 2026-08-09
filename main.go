package main

import (
	"context"
	"fmt"
)

func main() {
	fmt.Println("-> Application started")
	ctx := context.Background()

	// Load Config-File
	fmt.Println("-> Loading Config-File")
	cfg, err := LoadConfig("config.json")
	if err != nil {
		fmt.Println("-> FATAL: Could not load Config-File:", err)
	}
	fmt.Println("-> Done")

	StartMemoryCompressor(ctx, cfg) // No Config Problems
	go startCLIServer(ctx, cfg)
	go startMonitorServer()
	MailSection(ctx, cfg) // No Config Problems

	fmt.Println("-> Exiting")
}
