package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

var startTime = time.Now()

func handleAgentAsk(cfg *Config, imapClient *imapclient.Client, ctx context.Context) http.HandlerFunc {
	logf("-> [CLI] New Request")
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Nur POST erlaubt", http.StatusMethodNotAllowed)
			logf("-> [CLI] Stopped Connection to Client: Method Error")
			return
		}

		// Liest den rohen Text aus dem Request
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			logf("-> [CLI] Fehler beim Lesen des Body: %v", err)
		}

		logf("-> [CLI] Empfangener Raw-Body (%d Bytes): %q", len(bodyBytes), string(bodyBytes))

		var req AgentRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			logf("-> [CLI] JSON Unmarshal Fehler: %v", err) // <--- Zeigt den genauen Syntaxfehler!
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(AgentResponse{Error: "Ungültiges JSON-Format"})
			logf("-> [CLI] Stopped Connection to Client: JSON Error")
			return
		}

		if req.VerifyKey == "" || req.VerifyKey != cfg.Program.CLISecretKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(AgentResponse{Error: "Ungültiger oder fehlender VerifyKey"})
			logf("-> [CLI] Stopped Connection to Client: Wrong Key")
			return
		}

		logf("-> [CLI] Client Verified")

		allTools := buildAllTools()
		messages := []api.Message{
			{
				Role:    "system",
				Content: cfg.SysPromts.CLI,
			},
			{
				Role:    "user",
				Content: req.Prompt,
			},
		}

		logf("-> [CLI] Starting Agent Loop")
		agentAnswer, err := runAgentLoop(imapClient, ctx, cfg, messages, allTools)
		if err != nil {
			logf("-> [CLI] Agent Error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(AgentResponse{Error: err.Error()})
			logf("-> [CLI] Stopped Connection to Client: Agent Error")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AgentResponse{Response: agentAnswer})
		logf("-> [CLI] Done")
	}
}

func handleAgentHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		d := time.Since(startTime)
		json.NewEncoder(w).Encode(HealthResponse{
			Status:  "ok",
			Version: version,
			Uptime:  fmt.Sprintf("%dd %dh %dm", int(d.Hours())/24, int(d.Hours())%24, int(d.Minutes())%60),
		})
	}
}

func startCLIServer(ctx context.Context, cfg *Config) {
	logf("-> [CLI] Erstelle eigenen Client")
	imapClient, _, err := setUpMail(cfg)
	if err != nil {
		logf("-> [CLI] WARN: Konnte Client nicht erstellen, Email-tools nicht brauchbar: %v", err)
	}
	logf("-> [CLI] Success")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/agent/ask", handleAgentAsk(cfg, imapClient, ctx))
	mux.HandleFunc("/health", handleAgentHealth())

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	logf("-> [CLI] Starte Server auf :8080")
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logf("-> [CLI] Server error: %v", err)
		}
	}()

	<-ctx.Done()
	logf("-> [CLI] Shutdown signal, closing HTTP server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
