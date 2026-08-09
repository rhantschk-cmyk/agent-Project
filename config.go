package main

import (
	"fmt"
	"encoding/json"
	"os"
)

type EmailConfig struct {
	Username string `json:"Username"`
	AppToken string `json:"AppToken"`
	Server string `json:"Server"`
	DraftFolder string `json:"DraftFolder"`
}

type SysPromtsConfig struct {
	Standard string `json:"STANDARD"`
	Important string `json:"IMPORTANT"`
	Classify string `json:"CLASSIFY"`
}

type ProgramConfig struct {
	Model string `json:"model"`
	MemoryCompressionTime int `json:"MemoryCompressionTime"`
	MemoryFile string `json:"memoryFile"`
	MemoryCompressPromt string `json:"MemoryCompressPromt"`
	SysPromts SysPromtsConfig `json:"sysPromts"`
}

type Config struct {
	Email EmailConfig `json:"Email"`
	Program ProgramConfig `json:"Program"`
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Could not open Config-File: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("Could not parse Config-File")
	}
	return &cfg, nil
}
