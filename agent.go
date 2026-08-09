package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/ollama/ollama/api"
)


func askAgentNoTools(ctx context.Context, cfg *Config, promt string) (string, error) {
	fmt.Println("-> Asking Agent:", promt)
	// Setup Client Connection
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return "", err
	}
	messages := []api.Message{}

	messages = append(messages, api.Message{
		Role: "system",
		Content: cfg.SysPromts.Classify,
	})

	messages = append(messages, api.Message{
		Role: "user",
		Content: promt,
	})

	req := &api.ChatRequest{
		Model: cfg.Program.Model,
		Messages: messages,
		Stream: new (bool),
	}

	var resp string

	// Simple chat and return answer
	err = client.Chat(ctx, req, func(response api.ChatResponse) error {
		resp = response.Message.Content
		return nil
	})
	if err != nil {
		return "", err
	}

	fmt.Println("-> Done")
	return resp, err
}

func generateMail(imapClient *imapclient.Client, ctx context.Context, cfg *Config, mail *ParsedEmail, category string) (string, error) {
	fmt.Println("-> Generating Email")
	messages := []api.Message{}

	var sysPrompt string
	switch category {
	case "STANDARD":
		sysPrompt = cfg.SysPromts.Standard
	case "IMPORTANT":
		sysPrompt = cfg.SysPromts.Important	
	case "SPAM":
		return "SPAM", nil
	}

	messages = append(messages, api.Message{
		Role:    "system",
		Content: sysPrompt,
	})

	userContent := fmt.Sprintf("Bitte bearbeite diese E-Mail:\nFrom: %s\nSubject: %s\nBody:\n%s", mail.From, mail.Subject, mail.Body)
	messages = append(messages, api.Message{
		Role:    "user",
		Content: userContent,
	})

	allTools := buildAllTools()

	result, err := runAgentLoop(imapClient, ctx, cfg, messages, allTools)
	return result, err
}

func runAgentLoop(imapClient *imapclient.Client, ctx context.Context,  cfg *Config, initialMessages []api.Message, allTools api.Tools) (string, error) {
	ollamaClient, err := api.ClientFromEnvironment()
	if err != nil {
		return "", err
	}

	messages := initialMessages
	maxTurns := 30 // Safety-Limit gegen Endlosschleifen

	for turn := 0; turn < maxTurns; turn++ {
		req := &api.ChatRequest{
			Model:    cfg.Program.Model,
			Messages: messages,
			Tools:    allTools,
			Stream:   new(bool),
		}

		var responseMessage api.Message
		err = ollamaClient.Chat(ctx, req, func(resp api.ChatResponse) error {
			responseMessage = resp.Message
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("Fehler im Chat: %w", err)
		}

		if len(responseMessage.ToolCalls) == 0 {
			return responseMessage.Content, nil
		}

		messages = append(messages, responseMessage)

		for _, toolCall := range responseMessage.ToolCalls {
			toolResult, err := executeToolCalls(toolCall, imapClient, cfg)
			if err != nil {
				return toolResult, nil
			}

			// 3. Das Ergebnis des Tools zurück an das LLM übergeben!
			messages = append(messages, api.Message{
				Role:    "tool",
				Content: toolResult,
			})
		}
	}

	return "", fmt.Errorf("Maximales Turn-Limit (%d) erreicht", maxTurns)
}


func executeSearchInbox(client *imapclient.Client, query string) string {
	fmt.Printf("-> [Tool] Searching Inbox for: %s\n", query)

	// Suche nach Mails, die das Text-Muster im Body oder Subject enthalten
	criteria := &imap.SearchCriteria{
		Body: []string{query},
	}

	// Suchbefehl auf den Posteingang (INBOX) ausführen
	searchCmd := client.Search(criteria, nil)
	data, err := searchCmd.Wait()
	if err != nil || len(data.AllSeqNums()) == 0 {
		return "Keine E-Mails zu diesem Suchbegriff gefunden."
	}

	// Begrenze das Ergebnis auf die letzten 3 Treffer
	ids := data.AllSeqNums()
	if len(ids) > 3 {
		ids = ids[len(ids)-3:]
	}

	var results []string
	for _, seqNum := range ids {
		mail, err := fetchEmailDetails(client, seqNum)
		if err == nil {
			results = append(results, fmt.Sprintf("- Von: %s | Betreff: %s | Inhalt: %s", mail.From, mail.Subject, mail.Body))
		}
	}

	return strings.Join(results, "\n")
}

func executeGetConversationHistory(client *imapclient.Client, sender string, count int) string {
	fmt.Printf("-> [Tool] Getting last %d mails for: %s\n", count, sender)
	fmt.Printf("-> [Tool] Fetching last %d mails from: %s\n", count, sender)

	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "FROM", Value: sender},
		},
	}

	searchCmd := client.Search(criteria, nil)
	data, err := searchCmd.Wait()
	if err != nil || len(data.AllSeqNums()) == 0 {
		return "Keine bisherige Konversation mit diesem Absender gefunden."
	}

	ids := data.AllSeqNums()
	if len(ids) > count {
		// Nimm nur die neuesten X E-Mails
		ids = ids[len(ids)-count:]
	}

	var history []string
	for _, seqNum := range ids {
		mail, err := fetchEmailDetails(client, seqNum)
		if err == nil {
			history = append(history, fmt.Sprintf("[%s] %s: %s", mail.Date, mail.From, mail.Body))
		}
	}

	return strings.Join(history, "\n---\n")
}

func executeSearchDocs(cfg *Config, query string) string {
	fmt.Printf("-> [Tool] Searching Knowledge Base for: %s\n", query)
	content, err := os.ReadFile(filepath.Join(cfg.Program.KnowledgeDir, fmt.Sprintf(query, ".md")))
	if err != nil {
		return "Could not read file"
	}
	return string(content)
}

func executeListDocs(cfg *Config) string {
	fmt.Println("-> [Tool] Listing Docs")
	entries, err := os.ReadDir(cfg.Program.KnowledgeDir)
	if err != nil || len(entries) == 0 {
		return "No Knowledge Files found, nothing to list up"
	}

	var builder strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		extension := filepath.Ext(name)
		nameWithoutExtension := strings.TrimSuffix(name, extension)

		builder.WriteString(nameWithoutExtension)
		builder.WriteString(", ")
	}

	return builder.String()
}
