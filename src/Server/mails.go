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
	logf("-> Setting Up Mail connection")
	// Setting up Channel to make things work when the function ends (new mails spawn in this channel)
	newMailChan := make(chan uint32, 100)

	// Event Handler when new Email received
	options := &imapclient.Options{
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					logf("-> Neue Mail erkannt: %d", *data.NumMessages)
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
	logf("-> Success")

	return client, newMailChan, nil

}

func MailSection(ctx context.Context, cfg *Config) {
	client, newMailChan, err := setUpMail(cfg) // No Config Problems
	if err != nil {
		logf("-> Error while Setting up: %v", err)
		return
	}
	defer client.Close()

	logf("-> Starting Email Section")
	logf("-> Listening for Emails")

	for {
		// Start IDLE in a goroutine so we can also react to shutdown signals.
		idleCmd, idleErr := client.Idle()
		if idleErr != nil {
			logf("-> Error while starting IDLE: %v", idleErr)
			// avoid a busy loop if the connection is broken
			select {
			case <-ctx.Done():
				logf("-> MailSection: shutdown signal received, stopping")
				return
			case <-time.After(2 * time.Second):
				continue
			}
		}

		select {
		case <-ctx.Done():
			logf("-> MailSection: shutdown signal received, stopping")
			if idleCmd != nil {
				_ = idleCmd.Close()
			}
			return
		case seqNum, ok := <-newMailChan:
			if !ok {
				logf("-> FATAL: Channel closed unexpectedly")
				if idleCmd != nil {
					_ = idleCmd.Close()
				}
				return
			}
			if idleCmd != nil {
				if err := idleCmd.Close(); err != nil {
					logf("-> Error while closing IDLE: %v", err)
				}
			}

			mail, err := fetchEmailDetails(client, seqNum)
			if err != nil {
				logf("-> Error while getting Mail: %v", err)
				continue
			}
			category, err := classifyMail(mail, ctx, cfg) // No Config Problems
			if err != nil {
				logf("-> Error while Classifying: %v", err)
				continue
			}
			logf("-> Category: %s", category)
			response, err := respondEmail(client, ctx, cfg, mail, category) // No Config Problems
			logf("-> Model Created Draft with: %s", response)
			if err != nil {
				logf("-> Error while responding: %v", err)
			}
		}
	}
}

func fetchEmailDetails(client *imapclient.Client, seqNum uint32) (*ParsedEmail, error) {
	logf("-> Fetching Mail")

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

	logf("-> Done")
	return parsed, nil
}

func classifyMail(mail *ParsedEmail, ctx context.Context, cfg *Config) (string, error) {
	logf("-> Classifying Email")
	// Ask Agent without tools to classify Email
	// (no memory)
	response, err := askAgentNoTools(ctx, cfg, mail.Body) // No Config Problems
	if err != nil {
		return "", err
	}
	logf("-> Done")
	return response, nil
}

func respondEmail(client *imapclient.Client, ctx context.Context, cfg *Config, mail *ParsedEmail, category string) (string, error) {
	rawBody, err := generateMail(client, ctx, cfg, mail, category) // No Config Problems
	if err != nil {
		return "", err
	}
	finalBody := strings.ReplaceAll(rawBody, `\n`, "\n")
	// Creating struct then converting to email
	mail_response_template := ParsedEmail{
		UID:     0,
		To:      mail.From,
		From:    mail.To,
		Subject: "",
		Body:    finalBody,
		Date:    "",
	}
	// Creating Draft
	response := toMail(mail_response_template)
	logf("-> Responding to Email")
	err = createDraft(client, cfg, response) // No Config Problems
	if err != nil {
		return "", err
	}
	logf("-> Done")
	return finalBody, nil
}

func createDraft(client *imapclient.Client, cfg *Config, mail bytes.Buffer) error {
	logf("-> Creating Draft")
	appendOptions := &imap.AppendOptions{
		Time:  time.Now(),
		Flags: []imap.Flag{imap.FlagDraft},
	}

	mailBytes := mail.Bytes()
	mailReader := bytes.NewReader(mailBytes)
	mailSize := int64(len(mailBytes))

	cmd := client.Append(cfg.Email.DraftFolder, mailSize, appendOptions)

	if _, err := io.Copy(cmd, mailReader); err != nil {
		logf("-> Fehler beim Schreiben der Mail-Daten: %v", err)
		cmd.Close()
		return err
	}

	// IMPORTANT: close server
	if err := cmd.Close(); err != nil {
		logf("-> Fehler beim Schließen des Append-Streams: %v", err)
		return err
	}

	if _, err := cmd.Wait(); err != nil {
		logf("-> Fehler beim Erstellen des Entwurfs: %v", err)
		return err
	}

	logf("-> Entwurf erfolgreich erstellt!")
	return nil
}
