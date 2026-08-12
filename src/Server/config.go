package main

import (
	"fmt"
	"encoding/json"
	"os"
)

type EmailConfig struct {
	Username		string	`json:"username"`
	AppToken		string	`json:"app_token"`
	Server 			string	`json:"server"`
	DraftFolder		string	`json:"draft_folder"`
}

type SysPromtsConfig struct {
	Standard		string	`json:"standard"`
	Important		string	`json:"important"`
	Classify		string	`json:"classify"`
	CLI				string	`json:"cli"`
}

type ProgramConfig struct {
	Model			string	`json:"model"`
	KnowledgeDir	string	`json:"knowledge_dir"`
	CLISecretKey 	string	`json:"cli_secret_key"`
}

type MemoryConfig struct {
	MemoryCompressionTime	int		`json:"memory_compression_time"`
	MemoryFile 				string 	`json:"memory_file"`
	MemoryCompressPromt 	string	`json:"memory_compress_promt"`
} 

type Config struct {
	Email 			EmailConfig 	`json:"e-mail"`
	Program 		ProgramConfig 	`json:"program"`
	Memory 			MemoryConfig 	`json:"memory"`
	SysPromts 		SysPromtsConfig	`json:"sys_promts"`
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
