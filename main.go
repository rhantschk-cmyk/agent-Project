package main

import (
	"context"
	"fmt"
)

func main() {
	fmt.Println("-> Application started")
	ctx := context.Background()

	// Load Config-File
	fmt.Println("-> Loading Config-File")
	cfg, err := LoadConfig("config.json")
	if err != nil {
		fmt.Println("-> FATAL: Could not load Config-File:", err)
	}
	fmt.Println("-> Done")

	StartMemoryCompressor(ctx, cfg) // No Config Problems
	MailSection(ctx, cfg) // No Config Problems

	fmt.Println("-> Exiting")
}

func MailSection(ctx context.Context, cfg *Config) {
	client, newMailChan, err := setUpMail(cfg) // No Config Problems
	if err != nil {
		fmt.Println("-> Error while Setting up:", err)
	}
	defer client.Close()

	fmt.Println("-> Starting Email Section")
	fmt.Println("-> Listening for Emails")

	for {
		idleCmd, err := client.Idle()
		if err != nil {
			fmt.Println("-> Error while starting IDLE: ", err)
		}
		seqNum, ok := <-newMailChan
		if !ok {
			fmt.Println("-> FATAL: Channel closed unexpectedly")
			break
		}

		if err := idleCmd.Close(); err != nil {
			fmt.Println("-> Error while closing IDLE: ", err)
		}

		mail, err := fetchEmailDetails(client, seqNum)
		if err != nil {
			fmt.Println("-> Error while getting Mail: ", err)
		} else {
			category, err := classifyMail(mail, ctx, cfg) // No Config Problems
			if err != nil {
				fmt.Println("-> Error while Classifying: ", err)
			}
			fmt.Println("-> Category", category)
			response, err := respondEmail(client, ctx, cfg, mail, category) // No Config Problems
			fmt.Println("-> Model Created Draft with:", response)
			if err != nil {
				fmt.Println("-> Error while responding: ", err)
			}
		}
	}
}
