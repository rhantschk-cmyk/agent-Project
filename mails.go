package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// To easily parse and unparse emails
type ParsedEmail struct {
	UID     uint32
	From    string
	To      string
	Subject string
	Date    string
	Body    string
}

func toMail(mail ParsedEmail) bytes.Buffer {
	// Converting Struct to Buffer 
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

func setUpMail(cfg *Config) (*imapclient.Client, chan uint32, error) {
	fmt.Println("-> Setting Up Mail connection")
	// Setting up Channel to make things work when the function ends (new mails spawn in this channel)
	newMailChan := make(chan uint32, 100)

	// Event Handler when new Email received
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

	// Setting up client with options
	client, err := imapclient.DialTLS(cfg.Email.Server, options)
	if err != nil {
		return nil, nil, err 
	}

	// Loging in 
	// and selecting Inbox as place where to look
	// for new mails
	if err := client.Login(cfg.Email.Username, cfg.Email.AppToken).Wait(); err != nil {
		return nil, nil, err 
	}

	_ = client.Select("INBOX", nil) 
	fmt.Println("-> Success")

	return client, newMailChan, nil

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

func fetchEmailDetails(client *imapclient.Client, seqNum uint32) (*ParsedEmail, error) {
	fmt.Println("-> Fetching Mail")
	
	// Converting email to struct
	// (written by Gemini)
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


func classifyMail(mail *ParsedEmail, ctx context.Context, cfg *Config) (string, error) {
	fmt.Println("-> Classifying Email")
	// Ask Agent without tools to classify Email 
	// (no memory)
	response, err := askAgentNoTools(ctx, cfg, mail.Body) // No Config Problems
	if err != nil {
		return "", err 
	}
	fmt.Println("-> Done")
	return response, nil
}


func respondEmail(client *imapclient.Client, ctx context.Context, cfg *Config, mail *ParsedEmail, category string) (string, error) {
	rawBody, err := generateMail(client, ctx, cfg, mail, category) // No Config Problems
	finalBody := strings.ReplaceAll(rawBody, `\n`, "\n")
	if err != nil {
		return "", err
	}
	// Creating struct then converting to email
	mail_response_template := ParsedEmail {
		UID: 0,
		To: mail.From,
		From: mail.To,
		Subject: "",
		Body: finalBody,
		Date: "",
	}
	// Creating Draft
	response := toMail(mail_response_template)
	fmt.Println("-> Responding to Email")
	createDraft(client, cfg, response) // No Config Problems
	fmt.Println("-> Done")
	return finalBody, nil
}

func createDraft(client *imapclient.Client, cfg *Config, mail bytes.Buffer) error {
	fmt.Println("-> Creating Draft")
	appendOptions := &imap.AppendOptions{
		Time:  time.Now(),
		Flags: []imap.Flag{imap.FlagDraft},
	}

	mailBytes := mail.Bytes()
	mailReader := bytes.NewReader(mailBytes)
	mailSize := int64(len(mailBytes))

	cmd := client.Append(cfg.Email.DraftFolder, mailSize, appendOptions)

	if _, err := io.Copy(cmd, mailReader); err != nil {
		fmt.Println("-> Fehler beim Schreiben der Mail-Daten: ", err)
		cmd.Close()
		return err
	}

	// IMPORTANT: close server
	if err := cmd.Close(); err != nil {
		fmt.Println("-> Fehler beim Schließen des Append-Streams: ", err)
		return err
	}

	if _, err := cmd.Wait(); err != nil {
		fmt.Println("-> Fehler beim Erstellen des Entwurfs: ", err)
		return err
	}

	fmt.Println("-> Entwurf erfolgreich erstellt!")
	return nil
}
