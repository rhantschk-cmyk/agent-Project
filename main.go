package main

import (
	"context"
	"fmt"
	"log"
)

func main() {
	fmt.Println("-> Application started")
	ctx := context.Background()

	client, newMailChan, err := setUpMail("imap.gmail.com:993", "r.hantschk@gmail.com", "ebtsqyammevpfway")
	if err != nil {
		log.Fatalf("Setupfehler: %v", err)
	}
	defer client.Close()

	fmt.Println("-> Starting Email Section")
	fmt.Println("-> Listening for Emails")

	for {
		idleCmd, err := client.Idle()
		if err != nil {
			log.Fatalf("Fehler beim Starten von IDLE: %v", err)
		}
		seqNum, ok := <-newMailChan
		if !ok {
			break
		}

		if err := idleCmd.Close(); err != nil {
			log.Printf("Fehler beim Schließen von IDLE: %v", err)
		}

		mail, err := fetchEmailDetails(client, seqNum)
		if err != nil {
			log.Printf("Fehler beim Holen der Mail: %v", err)
		} else {
			fmt.Println(classifyMail(mail, ctx))
		}
	}

	fmt.Println("-> Exiting")
}
