package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ollama/ollama/api"
)

func test(value1 string, value2 string) string {
	return "Value1 war: " + value1 + ", Value2 war: " + value2
}

func main() {
	fmt.Println("-> Application started")
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	messages := []api.Message{}
	props := []Property{} 

	messages = append(messages, api.Message{
		Role: "user",
		Content: "Probiere das Test tool aus und gib mir seine ausgabe.",
	})

	fmt.Println("-> Settign up Tools")

	props = append(props, Property {
		name: "value1",
		category: "string",
		description: "Wert der zusammen mit dem zweiten Wert nach dem toolcall zurückgegeben werden soll",
	})

	props = append(props, Property {
		name: "value2",
		category: "string",
		description: "Wert der zusammen mit dem ersten Wert nach dem toolcall zurückgegeben werden soll",
	})

	test_tool := build_tool(props, Tool{
		name: "test",
		description: "Dies ist ein tool um zu testen ob mein Code funktioniert",
	})

	req := &api.ChatRequest {
		Model: "gpt-oss:20b",
		Tools: api.Tools{test_tool},
		Messages: messages,
		Stream: new (bool),
	}

	fmt.Println("-> Sende Anfrage an Modell")
	err = client.Chat(ctx, req, func(resp api.ChatResponse) error {
		if len(resp.Message.ToolCalls) > 0 {
			for _, call := range resp.Message.ToolCalls {
				fmt.Println("-> Model will Tool aufrufen: ", call.Function.Name)

				if call.Function.Name == "test" {
					value1 := call.Function.Arguments.ToMap()["value1"].(string)
					value2 := call.Function.Arguments.ToMap()["value2"].(string)

					result := test(value1, value2)
					fmt.Println("-> Das Ergebnis war:", result)

					messages = append(messages, resp.Message)
					messages = append(messages, api.Message{
						Role: "tool",
						Content: result,
					})
				}
			}
			
			fmt.Println("-> Sende b2b Request")
			b2bReq := &api.ChatRequest {
				Model: "gpt-oss:20b",
				Messages: messages,
			}

			fmt.Print("-> Agent Antwort:")
			return client.Chat(ctx, b2bReq, func(finalResp api.ChatResponse) error {
				fmt.Print(finalResp.Message.Content)
				return nil
			})

		}

		fmt.Print("-> Agent Antwort:")
		fmt.Print(resp.Message.Content)
		fmt.Println()
		return nil
	})

	fmt.Println()

	if err != nil {
		log.Fatal(err)
	}
}
