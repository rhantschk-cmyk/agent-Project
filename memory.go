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
	fmt.Println("-> [Tool] Saving fact in Memory:", fact)
	memoryMutex.Lock()
	defer memoryMutex.Unlock()

	f, err := os.OpenFile(cfg.Program.MemoryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "Error while opening Memory-File"
	}
	defer f.Close()
	
	timestamp := time.Now().Format("2006-01-02 15:04")
	entry := fmt.Sprintf("[%s] %s\n", timestamp, fact)

	if _, err := f.WriteString(entry); err != nil {
		return "Error while writing Memory-File"
	}

	return fmt.Sprintf("Successfully saved: %s", fact)
}

func executeReadMemory(cfg *Config) string {
	fmt.Println("-> [Tool] Reading Memory")
	memoryMutex.Lock()
	defer memoryMutex.Unlock()

	content, err := os.ReadFile(cfg.Program.MemoryFile)
	if err != nil || len(content) == 0 {
		return "Memory File is empty"
	}

	return string(content)
}

func StartMemoryCompressor(ctx context.Context, cfg *Config) {
	fmt.Println("-> [Memory] Started Memory Compressor")
	ticker := time.NewTicker(time.Duration(cfg.Program.MemoryCompressionTime) * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Println("-> [Memory] Compressing...")
				compressMemory(ctx, cfg)
				fmt.Println("-> [Memory] Done")
			case <-ctx.Done():
				return
			}
		}
	}()
}

func compressMemory(ctx context.Context, cfg *Config) {
	memoryMutex.Lock()
	defer memoryMutex.Unlock()

	content, err := os.ReadFile(cfg.Program.MemoryFile)
	if err != nil || len(content) < 200 {
		return
	}

	fmt.Println("-> [Memory] Compressing Memory File...")

	ollamaClient, err := api.ClientFromEnvironment()
	if err != nil {
		return
	}

	prompt := cfg.Program.MemoryCompressPromt + string(content)

	req := &api.ChatRequest{
		Model: cfg.Program.Model,
		Messages: []api.Message{
			{Role: "system", Content: "Du bist ein Präzisions-System zur Zusammenfassung von Notizen. Antworte NUR mit den stichpunktartigen Fakten."},
			{Role: "user", Content: prompt},
		},
		Stream: new(bool),
	}

	var compressedText string
	err = ollamaClient.Chat(ctx, req, func(resp api.ChatResponse) error {
		compressedText = resp.Message.Content
		return nil
	})

	if err == nil && strings.TrimSpace(compressedText) != "" {
		_ = os.WriteFile(cfg.Program.MemoryFile, []byte(compressedText+"\n"), 0644)
		fmt.Println("-> [Memory] Memory erfolgreich komprimiert!")
	}
}
