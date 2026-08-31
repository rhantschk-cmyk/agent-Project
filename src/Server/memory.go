package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
)

var memoryMutex sync.Mutex

// Tools

func executeSaveMemory(fact string, cfg *Config) string {
	logf("-> [Tool] Saving fact in Memory: %s", fact)
	memoryMutex.Lock()
	defer memoryMutex.Unlock()

	f, err := os.OpenFile(cfg.Memory.MemoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logf("-> Error while opening Memory-File: %v", err)
		return "Error while opening Memory-File"
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04")
	entry := fmt.Sprintf("[%s] %s\n", timestamp, fact)

	if _, err := f.WriteString(entry); err != nil {
		logf("-> Error while writing Memory-File: %v", err)
		return "Error while writing Memory-File"
	}

	return fmt.Sprintf("Successfully saved: %s", fact)
}

func executeReadMemory(cfg *Config) string {
	logf("-> [Tool] Reading Memory")
	memoryMutex.Lock()
	defer memoryMutex.Unlock()

	content, err := os.ReadFile(cfg.Memory.MemoryFile)
	if err != nil || len(content) == 0 {
		logf("-> Memory File is empty or unreadable: %v", err)
		return "Memory File is empty"
	}

	return string(content)
}

func StartMemoryCompressor(ctx context.Context, cfg *Config) {
	logf("-> [Memory] Started Memory Compressor")
	ticker := time.NewTicker(time.Duration(cfg.Memory.MemoryCompressionTime) * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				logf("-> [Memory] Compressing...")
				compressMemory(ctx, cfg)
				logf("-> [Memory] Done")
			case <-ctx.Done():
				logf("-> [Memory] Compressor stopped")
				return
			}
		}
	}()
}

func compressMemory(ctx context.Context, cfg *Config) {
	memoryMutex.Lock()
	defer memoryMutex.Unlock()

	content, err := os.ReadFile(cfg.Memory.MemoryFile)
	if err != nil || len(content) < 200 {
		return
	}

	logf("-> [Memory] Compressing Memory File...")

	ollamaClient, err := api.ClientFromEnvironment()
	if err != nil {
		logf("-> [Memory] WARN: Ollama not reachable, skipping compression: %v", err)
		return
	}

	prompt := cfg.Memory.MemoryCompressPromt + string(content)

	stream := false
	req := &api.ChatRequest{
		Model: cfg.Program.Model,
		Messages: []api.Message{
			{Role: "system", Content: "Du bist ein Präzisions-System zur Zusammenfassung von Notizen. Antworte NUR mit den stichpunktartigen Fakten."},
			{Role: "user", Content: prompt},
		},
		Stream: &stream,
	}

	var compressedText string
	err = ollamaClient.Chat(ctx, req, func(resp api.ChatResponse) error {
		compressedText = resp.Message.Content
		return nil
	})

	if err != nil {
		logf("-> [Memory] Compression failed, keeping file unchanged: %v", err)
		return
	}

	if strings.TrimSpace(compressedText) != "" {
		if err := os.WriteFile(cfg.Memory.MemoryFile, []byte(compressedText+"\n"), 0644); err != nil {
			logf("-> [Memory] Could not write compressed memory: %v", err)
			return
		}
		logf("-> [Memory] Memory erfolgreich komprimiert!")
	}
}
