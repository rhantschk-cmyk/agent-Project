package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	valid := `{
		"e-mail": {
			"username": "user@gmail.com",
			"app_token": "token",
			"server": "imap.gmail.com:993",
			"draft_folder": "[Gmail]/Drafts"
		},
		"program": {
			"model": "qwen2.5:14b",
			"knowledge_dir": "docs",
			"cli_secret_key": "secret"
		},
		"memory": {
			"memory_compression_time": 5,
			"memory_file": "memory.txt",
			"memory_compress_promt": "compress"
		},
		"sys_promts": {
			"standard": "std",
			"important": "imp",
			"classify": "cls",
			"cli": "cli"
		}
	}`
	if err := os.WriteFile(path, []byte(valid), 0644); err != nil {
		t.Fatalf("could not write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig returned nil config")
	}
	if cfg.Email.Username != "user@gmail.com" {
		t.Errorf("expected username 'user@gmail.com', got %q", cfg.Email.Username)
	}
	if cfg.Email.AppToken != "token" {
		t.Errorf("expected app_token 'token', got %q", cfg.Email.AppToken)
	}
	if cfg.Program.Model != "qwen2.5:14b" {
		t.Errorf("expected model 'qwen2.5:14b', got %q", cfg.Program.Model)
	}
	if cfg.Program.CLISecretKey != "secret" {
		t.Errorf("expected cli_secret_key 'secret', got %q", cfg.Program.CLISecretKey)
	}
	if cfg.Memory.MemoryCompressionTime != 5 {
		t.Errorf("expected memory_compression_time 5, got %d", cfg.Memory.MemoryCompressionTime)
	}
	if cfg.SysPromts.Classify != "cls" {
		t.Errorf("expected classify 'cls', got %q", cfg.SysPromts.Classify)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.json")

	cfg, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if cfg != nil {
		t.Fatalf("expected nil config on error, got %v", cfg)
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
		t.Fatalf("could not write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if cfg != nil {
		t.Fatalf("expected nil config on error, got %v", cfg)
	}
}
