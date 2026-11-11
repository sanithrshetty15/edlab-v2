package mailer

import (
	"bytes"
	"fmt"
	"net/smtp"
	"os"
)

// Mailer handles email sending operations
type Mailer struct {
	ResendAPIKey    string
	GmailUsername   string
	GmailAppPassword string
	smtpHost        string
	smtpPort        string
}

// New creates a new Mailer instance
func New() *Mailer {
	return &Mailer{
		ResendAPIKey:      os.Getenv("RESEND_API_KEY"),
		GmailUsername:     os.Getenv("GMAIL_USERNAME"),
		GmailAppPassword:  os.Getenv("GMAIL_APP_PASSWORD"),
		smtpHost:          "smtp.gmail.com",
		smtpPort:          "587",
	}
}

// SendEmail sends an email using the configured provider
func (m *Mailer) SendEmail(to, subject, body string) error {
	if m.ResendAPIKey != "" {
		return m.sendViaResend(to, subject, body)
	}
	if m.GmailUsername != "" && m.GmailAppPassword != "" {
		return m.sendViaGmail(to, subject, body)
	}
	return fmt.Errorf("no email provider configured")
}

// sendViaResend sends email via Resend API
func (m *Mailer) sendViaResend(to, subject, body string) error {
	// Resend API would be called here
	// For now, return nil (no-op)
	return nil
}

// sendViaGmail sends email via Gmail SMTP
func (m *Mailer) sendViaGmail(to, subject, body string) error {
	// Set up email headers
	var headers bytes.Buffer
	headers.WriteString("From: Edwins Lab <" + m.GmailUsername + ">\n")
	headers.WriteString("To: " + to + "\n")
	headers.WriteString("Subject: " + subject + "\n")
	headers.WriteString("Content-Type: text/html; charset=UTF-8\n\n")
	headers.WriteString(body)

	// Connect to Gmail SMTP
	auth := smtp.PlainAuth("", m.GmailUsername, m.GmailAppPassword, m.smtpHost)

	err := smtp.SendMail(m.smtpHost+":"+m.smtpPort, auth, m.GmailUsername, []string{to}, headers.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email via Gmail: %w", err)
	}

	return nil
}