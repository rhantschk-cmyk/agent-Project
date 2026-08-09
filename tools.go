package main

import (
	"fmt"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/ollama/ollama/api"
)

//Structs to fill up when creating tool
type Property struct {
	name string
	category string
	description string
}

type Tool struct {
	name string 
	description string 
}

func build_tool(all_properties []Property, target_tool Tool) api.Tool {
	// Standard Tool stuff for tool function
	props := api.NewToolPropertiesMap()
	requiredList := []string{}
	for _, prop := range all_properties {
		props.Set(prop.name, api.ToolProperty{
			Type: api.PropertyType{prop.category},
			Description: prop.description,
		})
		requiredList = append(requiredList, prop.name)
	}

	tool := api.Tool {
		Type: "function",
		Function: api.ToolFunction {
			Name: target_tool.name,
			Description: target_tool.description,
			Parameters: api.ToolFunctionParameters{
				Type: "object",
				Properties: props,
				Required:   requiredList,
			},
		},
	}

	return tool
}

func buildAllTools(isEmail bool) api.Tools {
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
	searchDocsProps := []Property{
		{
			name:        "doc_name",
			category:    "string",
			description: "The Filename for the Document to read",
		},
	}

	searchDocsTool := Tool{
		name:        "search_docs",
		description: "Returns the full content of o Document. A Document could be an Email Template or just Information",
	}

	searchDocs := build_tool(searchDocsProps, searchDocsTool)

	// 4. list_knowledge()
	listDocsProps := []Property{} // Keine Parameter benötigt

	listDocsTool := Tool{
		name:        "list_all_docs",
		description: "Lists all available document names currently indexed in the system",
	}

	listDocs := build_tool(listDocsProps, listDocsTool)

	// 5. read_memory()
	readMemoryProps := []Property{}

	readMemoryTool := Tool{
		name: 		"read_memory",
		description: "Lists the content of the Memory-File",
	}

	readMemory := build_tool(readMemoryProps, readMemoryTool)

	// 6. write_memory(fact string)
	writeMemoryProps := []Property{
		{
			name: "fact",
			category: "string",
			description: "The fact to save in the Memory File",
		},
	}

	writeMemoryTool := Tool{
		name: "write_memory",
		description: "Writes a fact to the Memory File",
	}

	writeMemory := build_tool(writeMemoryProps, writeMemoryTool)

	if isEmail {
		// 7. finish_and_reply(draft_body string, notes string)
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
	
		return api.Tools {
			searchInbox,
			getConversationHistory,
			searchDocs,
			listDocs,
			readMemory,
			writeMemory,
			finishAndReply,
		}
	}

	return api.Tools {
			searchInbox,
			getConversationHistory,
			searchDocs,
			listDocs,
			readMemory,
			writeMemory,
	}
}


func executeToolCalls(toolCall api.ToolCall, client *imapclient.Client, cfg *Config) string {
	toolName := toolCall.Function.Name			
	args := toolCall.Function.Arguments.ToMap()

	fmt.Printf("-> Agent ruft Tool auf: %s\n", toolName)

	switch toolName {

	case "search_inbox":
		query, _ := args["query"].(string)
		return executeSearchInbox(client, query)

	case "get_conversation_history":
		sender, _ := args["sender_email"].(string)
		// JSON-Zahlen kommen in Go oft als float64 an:
		countFloat, _ := args["count"].(float64)
		count := int(countFloat)
		if count == 0 {
			count = 3 // Fallback
		}
		return executeGetConversationHistory(client, sender, count)

	case "search_knowledge_base":
		query, _ := args["doc"].(string)
		return executeSearchDocs(cfg, query)

	case "list_docs":
		return executeListDocs(cfg)

	case "read_memory":
		return executeReadMemory(cfg)

	case "write_memory":
		fact, _ := args["fact"].(string)
		return executeSaveMemory(fact, cfg)

	default:
		return fmt.Sprintf("Fehler: Unbekanntes Tool %s", toolName)
	}
}
