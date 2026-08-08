package main

import (
	"context"
	"fmt"
)

func main() {
	fmt.Println("-> Application started")
	ctx := context.Background()

	// Load Config File HERE
	client, newMailChan, err := setUpMail("imap.gmail.com:993", "r.hantschk@gmail.com", "ebtsqyammevpfway") // <- Config File
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
			category, err := classifyMail(mail, ctx)
			if err != nil {
				fmt.Println("-> Error while Classifying: ", err)
			}
			fmt.Println("-> Category", category)
			response, err := respondEmail(client, ctx, mail, category)
			fmt.Println("-> Model Created Draft with:", response)
			if err != nil {
				fmt.Println("-> Error while responding: ", err)
			}
		}
	}

	fmt.Println("-> Exiting")
}
