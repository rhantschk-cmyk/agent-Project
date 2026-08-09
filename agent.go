package main

import (
	"context"
	"fmt"
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
		Content: cfg.Program.SysPromts.Classify,
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
		sysPrompt = cfg.Program.SysPromts.Standard
	case "IMPORTANT":
		sysPrompt = cfg.Program.SysPromts.Important	
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

	allTools := buildAllTools(true)

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
		fmt.Printf("-> [Turn %d] Sendefehler-Check | Messages im Context: %d\n", turn, len(messages))
		lastMsg := messages[len(messages)-1]
		fmt.Printf("-> Letzte Rolle: %s | Content: %s\n", lastMsg.Role, lastMsg.Content)

		messages = append(messages, responseMessage)

		for _, toolCall := range responseMessage.ToolCalls {
			toolName := toolCall.Function.Name
			args := toolCall.Function.Arguments.ToMap()

			fmt.Printf("-> Agent ruft Tool auf: %s\n", toolName)

			if toolName == "finish_and_reply" {
				draftBody, _ := toolCall.Function.Arguments.ToMap()["draft_body"].(string)
				notes, _ := args["notes"].(string)

				fmt.Printf("-> [Agent Reasoning/Notes]: %s\n", notes)
				fmt.Println("-> Agent beendet Recherche & erstellt Entwurf.")
				return draftBody, nil // Beendet den Loop und gibt den E-Mail-Text zurück!
			}

			// NORMALE TOOLS AUSFÜHREN
			var toolResult string
			switch toolName {

			case "search_inbox":
				query, _ := args["query"].(string)
				toolResult = executeSearchInbox(imapClient, query)

			case "get_conversation_history":
				sender, _ := args["sender_email"].(string)
				countFloat, _ := args["count"].(float64)
				count := int(countFloat)
				if count == 0 {
					count = 3 // Fallback
				}
				toolResult = executeGetConversationHistory(imapClient, sender, count)

			case "search_knowledge_base":
				query, _ := args["query"].(string)
				toolResult = executeSearchKnowledgeBase(query)

			case "list_knowledge":
				toolResult = executeListKnowledge()

			case "read_memory":
				toolResult = executeReadMemory(cfg)

			case "write_memory":
				fact, _ := args["fact"].(string)
				toolResult = executeSaveMemory(fact, cfg)

			default:
				toolResult = fmt.Sprintf("Fehler: Unbekanntes Tool %s", toolName)
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

func executeSearchKnowledgeBase(query string) string {
	fmt.Printf("-> [Tool] Searching Knowledge Base for: %s\n", query)
	return "Tool noch nicht verfügbar"
}

func executeListKnowledge() string {
	fmt.Println("-> [Tool] Listing Knowledge Sources")
	return "Tool noch nicht verfügbar"
}
