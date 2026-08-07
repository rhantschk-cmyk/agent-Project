package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/mail"
	"time"

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

func toMail(mail ParsedEmail) bytes.Buffer {
	var mailBuffer bytes.Buffer
	sender_line := "From:" + mail.From + "\r\n"
	receiver_line := "To:" + mail.To + "\r\n"
	subject_line := "Subject: [ENTWURF]" + mail.Subject + "\r\n"
	content_definition := "Content-Type: text/plain; charset=UTF-8\r\n"
	
	mailBuffer.WriteString(sender_line)
	mailBuffer.WriteString(receiver_line)
	mailBuffer.WriteString(subject_line)
	mailBuffer.WriteString(content_definition)
	mailBuffer.WriteString("\r\n")
	mailBuffer.WriteString(mail.Body)

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

	rawBodyBytes := buf.FindBodySection(bodySection)
	if rawBodyBytes != nil {
		parsedMsg, err := mail.ReadMessage(bytes.NewReader(rawBodyBytes))
		if err == nil {
			bodyContent, err := io.ReadAll(parsedMsg.Body)
			if err == nil {
				parsed.Body = string(bodyContent)
			}
		} else {
			parsed.Body = string(rawBodyBytes)
		}
	}

	fmt.Println("-> Done")
	return parsed, nil
}


func classifyMail(mail *ParsedEmail, ctx context.Context) (string, error) {
	fmt.Println("-> Classifying Email")
	response, err := askAgentNoTools(ctx, "gpt-oss:20b", "Du bist ein E-Mail Klassifizierer und darfst nur in einem Wort antworten. SPAM für spam emails, IMPORTANT für wichtige emails, STANDARD für emails die weder noch sind. WICHTIG: antworte nur in einem Wort", mail.Body)
	if err != nil {
		return "", err 
	}
	fmt.Println("-> Done")
	return response, nil
}


func respondEmail(client *imapclient.Client, ctx context.Context, mail *ParsedEmail, category string) (string, error) {
	body, err := generateMail(ctx, "gpt-oss:20b", mail, category)
	if err != nil {
		return "", err
	}
	mail_response_template := ParsedEmail {
		UID: 0,
		To: mail.From,
		From: mail.To,
		Subject: "",
		Body: body,
		Date: "",
	}
	response := toMail(mail_response_template)
	fmt.Println("-> Responding to Email")
	createDraft(client, "[Gmail]/Drafts", response)
	fmt.Println("-> Done")
	return body, nil
}

func createDraft(client *imapclient.Client, draft_folder string, mail bytes.Buffer) error {
	fmt.Println("-> Creating Draft")
	appendOptions := &imap.AppendOptions{
		Time:  time.Now(),
		Flags: []imap.Flag{imap.FlagDraft},
	}

	mailBytes := mail.Bytes()
	mailReader := bytes.NewReader(mailBytes)
	mailSize := int64(len(mailBytes))

	cmd := client.Append(draft_folder, mailSize, appendOptions)

	if _, err := io.Copy(cmd, mailReader); err != nil {
		fmt.Println("-> Fehler beim Schreiben der Mail-Daten: ", err)
		cmd.Close()
		return err
	}

	if err := cmd.Close(); err != nil {
		fmt.Println("-> Fehler beim Schließen des Append-Streams: ", err)
		return err
	}

	// 4. Jetzt erst auf die Antwort des Servers warten
	if _, err := cmd.Wait(); err != nil {
		fmt.Println("-> Fehler beim Erstellen des Entwurfs: ", err)
		return err
	}

	fmt.Println("-> Entwurf erfolgreich erstellt!")
	return nil
}
