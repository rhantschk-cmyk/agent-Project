package main

import (
	"bytes"
	"fmt"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2"

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

	return client, newMailChan, nil

}

func fetchEmailDetails(client *imapclient.Client, seqNum uint32) (*ParsedEmail, error) {
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
		return nil, fmt.Errorf("-> Nachricht nicht gefunden")
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

	rawBody := buf.FindBodySection(bodySection)
	if rawBody != nil {
		parsed.Body = string(rawBody)
	}

	return parsed, nil
}
