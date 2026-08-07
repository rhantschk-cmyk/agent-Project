package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/mail"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

type ParsedEmail struct {
	UID     uint32
	From    string
	To      string
	Subject string
	Date    string
	Body    string
}

func toMail(from string, to string, subject string, content string) bytes.Buffer {
	var mailBuffer bytes.Buffer
	sender_line := "From:" + from + "\r\n"
	receiver_line := "To:" + to + "\r\n"
	subject_line := "Subject: [ENTWURF]" + subject + "\r\n"
	content_definition := "Content-Type: text/plain; charset=UTF-8\r\n"
	
	mailBuffer.WriteString(sender_line)
	mailBuffer.WriteString(receiver_line)
	mailBuffer.WriteString(subject_line)
	mailBuffer.WriteString(content_definition)
	mailBuffer.WriteString("\r\n")
	mailBuffer.WriteString(content)

	return mailBuffer
}

func setUpMail(ServerAddress string, Username string, appToken string) (*imapclient.Client, chan uint32, error) {
	fmt.Println("-> Setting Up Mail connection")
	newMailChan := make(chan uint32, 100)

	options := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					fmt.Printf("-> Neue Mail erkannt")
					newMailChan <- uint32(*data.NumMessages)
				}
			},
		},
	}

	client, err := imapclient.DialTLS(ServerAddress, options)
	if err != nil {
		return nil, nil, err 
	}

	if err := client.Login(Username, appToken).Wait(); err != nil {
		return nil, nil, err 
	}

	_ = client.Select("INBOX", nil) 
	fmt.Println("-> Success")

	return client, newMailChan, nil

}

func fetchEmailDetails(client *imapclient.Client, seqNum uint32) (*ParsedEmail, error) {
	fmt.Println("-> Fetching Mail")
	
	bodySection := &imap.FetchItemBodySection{}
	fetchArgs := &imap.FetchOptions{
		Envelope:    true,
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}

	seqSet := imap.SeqSetNum(seqNum)
	fetchCmd := client.Fetch(seqSet, fetchArgs)
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return nil, fmt.Errorf("nachricht nicht gefunden")
	}

	buf, err := msg.Collect()
	if err != nil {
		return nil, err
	}

	parsed := &ParsedEmail{}

	// 1. Metadaten sauber aus der Envelope holen
	if buf.Envelope != nil {
		parsed.Subject = buf.Envelope.Subject
		parsed.Date = buf.Envelope.Date.Format("02.01.2006 15:04")
		if len(buf.Envelope.From) > 0 {
			parsed.From = buf.Envelope.From[0].Addr()
		}
		if len(buf.Envelope.To) > 0 {
			parsed.To = buf.Envelope.To[0].Addr()
		}
	}

	// 2. Den Body mit 'net/mail' parsen, um die Header vom Text zu trennen!
	rawBodyBytes := buf.FindBodySection(bodySection)
	if rawBodyBytes != nil {
		// net/mail liest die Rohe Mail und trennt Header vom Inhalt
		parsedMsg, err := mail.ReadMessage(bytes.NewReader(rawBodyBytes))
		if err == nil {
			// ReadAll liest NUR NOCH den eigentlichen Text-Body
			bodyContent, err := io.ReadAll(parsedMsg.Body)
			if err == nil {
				parsed.Body = string(bodyContent)
			}
		} else {
			// Fallback: Falls es kein Standard-Header-Format hatte
			parsed.Body = string(rawBodyBytes)
		}
	}

	fmt.Println("-> Done")
	return parsed, nil
}


func classifyMail(mail *ParsedEmail, ctx context.Context) string {
	fmt.Println("-> Classifying Email")
	response, err := askAgent(ctx, "gpt-oss:20b", "Du bist ein E-Mail Klassifizierer und darfst nur in einem Wort antworten. SPAM für spam emails, IMPORTANT für wichtige emails, STANDARD für emails die weder noch sind. WICHTIG: antworte nur in einem Wort", mail.Body)
	if err != nil {
		return "ERROR"
	}
	fmt.Println("-> Done")
	return response
}
