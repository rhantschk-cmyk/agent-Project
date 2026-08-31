package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testMemoryConfig(t *testing.T) (*Config, *Config) {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "memory.txt")

	cfg := &Config{
		Memory: MemoryConfig{
			MemoryFile: file,
		},
	}
	return cfg, &Config{}
}

func TestExecuteSaveMemory(t *testing.T) {
	cfg, _ := testMemoryConfig(t)

	result := executeSaveMemory("Test-Fakt über einen Gast", cfg)
	if !strings.Contains(result, "Successfully saved") {
		t.Errorf("expected success message, got %q", result)
	}

	content, err := os.ReadFile(cfg.Memory.MemoryFile)
	if err != nil {
		t.Fatalf("could not read memory file: %v", err)
	}
	if !strings.Contains(string(content), "Test-Fakt über einen Gast") {
		t.Errorf("memory file does not contain the saved fact")
	}
}

func TestExecuteReadMemoryEmpty(t *testing.T) {
	cfg, _ := testMemoryConfig(t)

	result := executeReadMemory(cfg)
	if !strings.Contains(result, "empty") {
		t.Errorf("expected empty-file message, got %q", result)
	}
}

func TestExecuteSaveThenReadMemory(t *testing.T) {
	cfg, _ := testMemoryConfig(t)

	executeSaveMemory("Fakt Eins", cfg)
	executeSaveMemory("Fakt Zwei", cfg)

	result := executeReadMemory(cfg)
	if !strings.Contains(result, "Fakt Eins") {
		t.Errorf("read memory missing 'Fakt Eins': %q", result)
	}
	if !strings.Contains(result, "Fakt Zwei") {
		t.Errorf("read memory missing 'Fakt Zwei': %q", result)
	}
}

func TestExecuteSaveMemoryAppendDoesNotOverwrite(t *testing.T) {
	cfg, _ := testMemoryConfig(t)

	executeSaveMemory("Erster", cfg)
	executeSaveMemory("Zweiter", cfg)

	content, err := os.ReadFile(cfg.Memory.MemoryFile)
	if err != nil {
		t.Fatalf("could not read memory file: %v", err)
	}
	text := string(content)
	if strings.Count(text, "[") < 2 {
		t.Errorf("expected at least two timestamped entries, got %q", text)
	}
}
