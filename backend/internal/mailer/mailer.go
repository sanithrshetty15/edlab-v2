package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"
)

const (
	ProviderResend    = "resend"
	ProviderGmailSMTP = "gmail_smtp"
	ProviderGmailAPI  = "gmail_api"

	resendEndpoint                  = "https://api.resend.com/emails"
	labEmail                        = "edwin.lab.aiet@gmail.com"
	RegistrationConfirmationSubject = "EdwinsLab – Membership Registration Successful"
)

type GmailAPISender interface {
	Enabled() bool
	Authorized(ctx context.Context) (bool, error)
	Send(ctx context.Context, rawMIME []byte) error
}

type RegistrationConfirmationData struct {
	Name            string
	LoginID         string
	IdentifierLabel string
	Identifier      string
	AcademicYear    int
}

// EmailContent is a transactional email with an HTML body and a plain-text
// alternative for clients that do not render HTML.
type EmailContent struct {
	HTML string
	Text string
}

// Mailer handles email sending operations.
type Mailer struct {
	ResendAPIKey     string
	GmailUsername    string
	GmailAppPassword string
	Provider         string
	GmailAPI         GmailAPISender
	From             string
	FrontendURL      string
	smtpHost         string
	smtpPort         string
	client           *http.Client
}

// New creates a new Mailer instance from environment configuration.
func New() *Mailer {
	from := strings.TrimSpace(os.Getenv("MAILER_FROM"))
	if from == "" {
		if username := strings.TrimSpace(os.Getenv("GMAIL_USERNAME")); username != "" {
			from = fmt.Sprintf("Edwins Lab <%s>", username)
		} else {
			from = "Edwins Lab <" + labEmail + ">"
		}
	}
	return &Mailer{
		ResendAPIKey:     strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		GmailUsername:    strings.TrimSpace(os.Getenv("GMAIL_USERNAME")),
		GmailAppPassword: strings.TrimSpace(os.Getenv("GMAIL_APP_PASSWORD")),
		Provider:         normalizeProvider(os.Getenv("MAIL_PROVIDER")),
		From:             from,
		FrontendURL:      strings.TrimSpace(os.Getenv("FRONTEND_URL")),
		smtpHost:         envOr("SMTP_HOST", "smtp.gmail.com"),
		smtpPort:         envOr("SMTP_PORT", "587"),
		client:           &http.Client{Timeout: 15 * time.Second},
	}
}

func normalizeProvider(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (m *Mailer) provider() string {
	return normalizeProvider(m.Provider)
}

// Configured reports whether at least one email provider is available.
func (m *Mailer) Configured() bool {
	switch m.provider() {
	case ProviderGmailAPI:
		return m.GmailAPI != nil && m.GmailAPI.Enabled()
	case ProviderResend:
		return m.ResendAPIKey != ""
	case ProviderGmailSMTP:
		return m.gmailConfigured()
	case "":
		return m.ResendAPIKey != "" || m.gmailConfigured()
	default:
		return false
	}
}

// SendEmail sends an HTML-only email using the configured provider. It is kept
// for callers that already have a rendered body; new templates should use
// SendEmailContent so recipients also get a plain-text alternative.
func (m *Mailer) SendEmail(to, subject, body string) error {
	return m.SendEmailContent(to, subject, EmailContent{HTML: body})
}

func (m *Mailer) SendEmailContent(to, subject string, content EmailContent) error {
	return m.SendEmailContentContext(context.Background(), to, subject, content)
}

func (m *Mailer) SendEmailContentContext(ctx context.Context, to, subject string, content EmailContent) error {
	to = strings.TrimSpace(to)
	subject = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(subject, "\r", " "), "\n", " "))
	content.HTML = strings.TrimSpace(content.HTML)
	content.Text = strings.TrimSpace(content.Text)
	if to == "" || !strings.Contains(to, "@") {
		return fmt.Errorf("a valid recipient email address is required")
	}
	if subject == "" {
		return fmt.Errorf("email subject is required")
	}
	if content.HTML == "" && content.Text == "" {
		return fmt.Errorf("email content is required")
	}

	switch m.provider() {
	case ProviderGmailAPI:
		if m.GmailAPI == nil || !m.GmailAPI.Enabled() {
			return fmt.Errorf("Gmail API is not configured")
		}
		message, err := m.buildGmailMessage(to, subject, content)
		if err != nil {
			return err
		}
		return m.GmailAPI.Send(ctx, message)
	case ProviderResend:
		if m.ResendAPIKey == "" {
			return fmt.Errorf("Resend is not configured")
		}
		return m.sendViaResend(to, subject, content)
	case ProviderGmailSMTP:
		if !m.gmailConfigured() {
			return fmt.Errorf("Gmail SMTP is not configured")
		}
		return m.sendViaGmail(to, subject, content)
	case "":
		if m.ResendAPIKey != "" {
			if err := m.sendViaResend(to, subject, content); err == nil {
				return nil
			} else if !m.gmailConfigured() {
				return err
			}
		}
		if m.gmailConfigured() {
			return m.sendViaGmail(to, subject, content)
		}
		return fmt.Errorf("no email provider configured")
	default:
		return fmt.Errorf("unsupported email provider %q", m.provider())
	}
}

func (m *Mailer) gmailConfigured() bool {
	return m.GmailUsername != "" && m.GmailAppPassword != ""
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

func (m *Mailer) sendViaResend(to, subject string, content EmailContent) error {
	payload, err := json.Marshal(resendRequest{
		From:    m.From,
		To:      []string{to},
		Subject: subject,
		HTML:    content.HTML,
		Text:    content.Text,
	})
	if err != nil {
		return fmt.Errorf("encode Resend request: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, resendEndpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create Resend request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+m.ResendAPIKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := m.client.Do(request)
	if err != nil {
		return fmt.Errorf("send email via Resend: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("Resend returned %d: %s", response.StatusCode, message)
	}
	return nil
}

func (m *Mailer) sendViaGmail(to, subject string, content EmailContent) error {
	message, err := m.buildGmailMessage(to, subject, content)
	if err != nil {
		return err
	}

	auth := smtp.PlainAuth("", m.GmailUsername, m.GmailAppPassword, m.smtpHost)
	if err := smtp.SendMail(m.smtpHost+":"+m.smtpPort, auth, m.GmailUsername, []string{to}, message); err != nil {
		return fmt.Errorf("send email via Gmail: %w", err)
	}
	return nil
}

func (m *Mailer) buildGmailMessage(to, subject string, content EmailContent) ([]byte, error) {
	var message bytes.Buffer
	message.WriteString("From: " + m.From + "\r\n")
	message.WriteString("To: " + to + "\r\n")
	message.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	message.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")

	switch {
	case content.HTML != "" && content.Text != "":
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		textHeader := textproto.MIMEHeader{}
		textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
		textHeader.Set("Content-Transfer-Encoding", "8bit")
		part, err := writer.CreatePart(textHeader)
		if err != nil {
			return nil, fmt.Errorf("create plain-text email part: %w", err)
		}
		if _, err := part.Write([]byte(content.Text)); err != nil {
			return nil, fmt.Errorf("write plain-text email part: %w", err)
		}
		htmlHeader := textproto.MIMEHeader{}
		htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
		htmlHeader.Set("Content-Transfer-Encoding", "8bit")
		part, err = writer.CreatePart(htmlHeader)
		if err != nil {
			return nil, fmt.Errorf("create HTML email part: %w", err)
		}
		if _, err := part.Write([]byte(content.HTML)); err != nil {
			return nil, fmt.Errorf("write HTML email part: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("close email body: %w", err)
		}
		message.WriteString("Content-Type: multipart/alternative; boundary=\"" + writer.Boundary() + "\"\r\n\r\n")
		message.Write(body.Bytes())
	case content.HTML != "":
		message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		message.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		message.WriteString(content.HTML)
	default:
		message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		message.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		message.WriteString(content.Text)
	}

	return message.Bytes(), nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
