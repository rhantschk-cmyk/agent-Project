package main

import (
	"context"
	"fmt"

	"github.com/ollama/ollama/api"
)


func askAgentNoTools(ctx context.Context, model string, sysPromt string, promt string) (string, error) {
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
		Messages: messages,
		Stream: new (bool),
	}

	var resp string

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

func generateMail(ctx context.Context, model string, mail *ParsedEmail, category string) (string, error) {
	fmt.Println("-> Generating Email")
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return "", err
	}
	messages := []api.Message{}

	var sysPromt string
	switch category {
	case "STANDARD":
		sysPromt = "Du bist der Email Beantwortungsagent von Raphael Hantschk. Erwähne nichts was du nicht weißt und schreibe nur auf Anweisung etwas Anderes wie den Body der Mail. Dazu gehören Begrüßung Inhalt und Verabschiedung. NIEMALS mehr"
	case "IMPORTANT":
		sysPromt = "Du bist der Email Beantwortungsagent von Raphael Hantschk. Erwähne nichts was du nicht weißt und schreibe nur auf Anweisung etwas Anderes wie den Body der Mail. Dazu gehören Begrüßung Inhalt und Verabschiedung. NIEMALS mehr. Deine Emails haben BESONDERE Priorität"
	case "SPAM":
		return "SPAM", nil
	}

	messages = append(messages, api.Message{
		Role: "system",
		Content: sysPromt,
	})

	messages = append(messages, api.Message{
		Role: "user",
		Content: "Beantworte diese Email: \n" + "From: \n" + mail.From + "\nSubject: \n" + mail.Subject + "\nBody: \n" + mail.Body + " Indem du nur den Body als Antwort schreibst.",
	})

	req := &api.ChatRequest{
		Model: model,
		Messages: messages,
		Stream: new (bool),
	}

	var resp string

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
