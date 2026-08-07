package main

import (
	"context"
	"fmt"

	"github.com/ollama/ollama/api"
)


func askAgent(ctx context.Context, model string, sysPromt string, promt string) (string, error) {
	fmt.Println("-> Asking Agent:", promt)
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
//		Tools: tools,
		Messages: messages,
		Stream: new (bool),
	}

	var resp string

	err = client.Chat(ctx, req, func(response api.ChatResponse) error {
		resp = response.Message.Content
		return nil
	})

	fmt.Println("-> Done")
	return resp, err
}
