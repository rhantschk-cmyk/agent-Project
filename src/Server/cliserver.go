package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/ollama/ollama/api"
)

type AgentRequest struct {
	VerifyKey string `json:"verify_key"`
	Prompt    string `json:"prompt"`
}

type AgentResponse struct {
	Response string `json:"response,omitempty"`
	Error    string `json:"error,omitempty"`
}

func handleAgentAsk(cfg *Config, imapClient *imapclient.Client, ctx context.Context) http.HandlerFunc {
	fmt.Println("-> [CLI] New Request")
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Nur POST erlaubt", http.StatusMethodNotAllowed)
			fmt.Println("-> [CLI] Stopped Connection to Client")
			return
		}

		var req AgentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(AgentResponse{Error: "Ungültiges JSON-Format"})
			fmt.Println("-> [CLI] Stopped Connection to Client")
			return
		}

		if req.VerifyKey == "" || req.VerifyKey != cfg.Program.CLISecretKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(AgentResponse{Error: "Ungültiger oder fehlender VerifyKey"})
			fmt.Println("-> [CLI] Stopped Connection to Client")
			return
		}

		fmt.Println("-> [CLI] Client Verified")

		allTools := buildAllTools()
		messages := []api.Message{
			{
				Role: "system",
				Content: cfg.SysPromts.CLI,
			},
			{
				Role: "user",
				Content: req.Prompt,
			},
		}

		fmt.Println("-> [CLI] Starting Agent Loop")
		agentAnswer, err := runAgentLoop(imapClient, ctx, cfg, messages, allTools)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(AgentResponse{Error: err.Error()})
			fmt.Println("-> [CLI] Stopped Connection to Client")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AgentResponse{Response: agentAnswer})
		fmt.Println("-> [CLI] Done")
	}
}

func startCLIServer(ctx context.Context, cfg *Config) {
	fmt.Println("-> [CLI] Erstelle eigenen Client")
	imapClient, _, err := setUpMail(cfg)
	if err != nil {
		fmt.Println("-> [CLI] FATAL: Konnte Client nicht erstellen, Email-tools nicht brauchbar")
	}
	fmt.Println("-> [CLI] Success")
	fmt.Println("-> [CLI] Starte Server")
	http.HandleFunc("/api/agent/ask", handleAgentAsk(cfg, imapClient, ctx))
	fmt.Println("-> [CLI] Done")
	fmt.Println("-> [CLI] Starting Listening")
	http.ListenAndServe(":8080", nil)
}
