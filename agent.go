package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/ollama/ollama/api"
)


func askAgentNoTools(ctx context.Context, model string, sysPromt string, promt string) (string, error) {
	fmt.Println("-> Asking Agent:", promt)
	// Setup Client Connection
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return "", err
	}
	messages := []api.Message{}

	messages = append(messages, api.Message{
		Role: "system",
		Content: sysPromt,
	})

	messages = append(messages, api.Message{
		Role: "user",
		Content: promt,
	})

	req := &api.ChatRequest{
		Model: model,
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

func generateMail(ctx context.Context, model string, mail *ParsedEmail, category string, imapClient *imapclient.Client) (string, error) {
	fmt.Println("-> Generating Email")
	messages := []api.Message{}

	var sysPrompt string
	switch category {
	case "STANDARD":
		sysPrompt = "Du bist der Email Beantwortungsagent von Raphael Hantschk. Nutze deine Tools um Informationen zu recherchieren, falls nötig. Wenn du bereit bist zu antworten, rufe UNBEDINGT das Tool 'finish_and_reply' mit dem finalen E-Mail Text auf!"
	case "IMPORTANT":
		sysPrompt = "Du bist der Email Beantwortungsagent von Raphael Hantschk. Diese E-Mail hat HOHE PRIORITÄT. Nutze deine Tools zur Recherche. Wenn du fertig bist, rufe das Tool 'finish_and_reply' auf!"
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

	// Tools 

	// 1. search_inbox(query string)
	searchInboxProps := []Property{}
	searchInboxProps = append(searchInboxProps, Property{
		name: "query",
		category: "string",
		description: "String to search for in the Inbox",
	})

	searchInboxTool := Tool {
		name: "search_inbox",
		description: "Searches the whole inbox after mails by the given string",
	}

	searchInbox := build_tool(searchInboxProps, searchInboxTool)

	// 2. get_conversation_history(sender_email string, count int) - Korrigiert
	getConversationHistoryProps := []Property{
		{
			name:        "sender_email",
			category:    "string",
			description: "The email address of the sender to get the history for",
		},
		{
			name:        "count",
			category:    "integer",
			description: "Number of emails to fetch from history (e.g. 3 or 5)",
		},
	}

	getConversationHistoryTool := Tool{
		name:        "get_conversation_history",
		description: "Fetches a specific number of previous emails from a conversation with a sender",
	}

	getConversationHistory := build_tool(getConversationHistoryProps, getConversationHistoryTool)

	// 3. search_knowledge_base(query string)
	searchKnowledgeBaseProps := []Property{
		{
			name:        "query",
			category:    "string",
			description: "The topic, keyword, or question to search for in the company knowledge base or documents",
		},
	}

	searchKnowledgeBaseTool := Tool{
		name:        "search_knowledge_base",
		description: "Searches uploaded documents and PDF files for facts, prices, and guidelines",
	}

	searchKnowledgeBase := build_tool(searchKnowledgeBaseProps, searchKnowledgeBaseTool)

	// 4. list_knowledge()
	listKnowledgeProps := []Property{} // Keine Parameter benötigt

	listKnowledgeTool := Tool{
		name:        "list_knowledge",
		description: "Lists all available document names and knowledge sources currently indexed in the system",
	}

	listKnowledge := build_tool(listKnowledgeProps, listKnowledgeTool)

	// 5. finish_and_reply(draft_body string, notes string)
	finishAndReplyProps := []Property{
		{
			name:        "draft_body",
			category:    "string",
			description: "The complete, finalized text body for the email response including greeting and sign-off",
		},
		{
			name:        "notes",
			category:    "string",
			description: "Internal reasoning or notes explaining why this response was generated (e.g., 'Used price list from AGB.pdf')",
		},
	}

	finishAndReplyTool := Tool{
		name:        "finish_and_reply",
		description: "Finishes the agent loop and creates the final draft email in the user's drafts folder",
	}

	finishAndReply := build_tool(finishAndReplyProps, finishAndReplyTool)

	allTools := api.Tools {
		searchInbox,
	    getConversationHistory,
	    searchKnowledgeBase,
	    listKnowledge,
	    finishAndReply,
	}

	result, err := runAgentLoop(ctx, imapClient, model, messages, allTools)
	return result, err
}

func runAgentLoop(ctx context.Context, client *imapclient.Client, model string, initialMessages []api.Message, allTools api.Tools) (string, error) {
	ollamaClient, err := api.ClientFromEnvironment()
	if err != nil {
		return "", err
	}

	messages := initialMessages
	maxTurns := 10 // Safety-Limit gegen Endlosschleifen

	for turn := 0; turn < maxTurns; turn++ {
		req := &api.ChatRequest{
			Model:    model,
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

		messages = append(messages, responseMessage)

		if len(responseMessage.ToolCalls) == 0 {
			return responseMessage.Content, nil
		}

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
				toolResult = executeSearchInbox(client, query)

			case "get_conversation_history":
				sender, _ := args["sender_email"].(string)
				// JSON-Zahlen kommen in Go oft als float64 an:
				countFloat, _ := args["count"].(float64)
				count := int(countFloat)
				if count == 0 {
					count = 3 // Fallback
				}
				toolResult = executeGetConversationHistory(client, sender, count)

			case "search_knowledge_base":
				query, _ := args["query"].(string)
				toolResult = executeSearchKnowledgeBase(query)

			case "list_knowledge":
				toolResult = executeListKnowledge()

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
	// Hier dein RAG-Feature anbinden
	return "Tool noch nicht verfügbar"
}

func executeListKnowledge() string {
	fmt.Println("-> [Tool] Listing Knowledge Sources")
	return "Tool noch nicht verfügbar"
}
