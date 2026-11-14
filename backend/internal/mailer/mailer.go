package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

const (
	resendEndpoint                  = "https://api.resend.com/emails"
	labEmail                        = "edwin.lab.aiet@gmail.com"
	RegistrationConfirmationSubject = "EdwinsLab – Membership Registration Successful"
)

type RegistrationConfirmationData struct {
	Name            string
	LoginID         string
	IdentifierLabel string
	Identifier      string
	AcademicYear    int
}

// Mailer handles email sending operations.
type Mailer struct {
	ResendAPIKey     string
	GmailUsername    string
	GmailAppPassword string
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
		From:             from,
		FrontendURL:      strings.TrimSpace(os.Getenv("FRONTEND_URL")),
		smtpHost:         envOr("SMTP_HOST", "smtp.gmail.com"),
		smtpPort:         envOr("SMTP_PORT", "587"),
		client:           &http.Client{Timeout: 15 * time.Second},
	}
}

// Configured reports whether at least one email provider is available.
func (m *Mailer) Configured() bool {
	return m.ResendAPIKey != "" || (m.GmailUsername != "" && m.GmailAppPassword != "")
}

// SendEmail sends an email using the configured provider.
func (m *Mailer) SendEmail(to, subject, body string) error {
	to = strings.TrimSpace(to)
	subject = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(subject, "\r", " "), "\n", " "))
	if to == "" || !strings.Contains(to, "@") {
		return fmt.Errorf("a valid recipient email address is required")
	}
	if subject == "" {
		return fmt.Errorf("email subject is required")
	}
	if m.ResendAPIKey != "" {
		if err := m.sendViaResend(to, subject, body); err == nil {
			return nil
		} else if !m.gmailConfigured() {
			return err
		}
	}
	if m.gmailConfigured() {
		return m.sendViaGmail(to, subject, body)
	}
	return fmt.Errorf("no email provider configured")
}

func (m *Mailer) gmailConfigured() bool {
	return m.GmailUsername != "" && m.GmailAppPassword != ""
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func (m *Mailer) sendViaResend(to, subject, body string) error {
	payload, err := json.Marshal(resendRequest{
		From:    m.From,
		To:      []string{to},
		Subject: subject,
		HTML:    body,
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

func (m *Mailer) sendViaGmail(to, subject, body string) error {
	var message bytes.Buffer
	message.WriteString("From: " + m.From + "\r\n")
	message.WriteString("To: " + to + "\r\n")
	message.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", subject) + "\r\n")
	message.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	message.WriteString(body)

	auth := smtp.PlainAuth("", m.GmailUsername, m.GmailAppPassword, m.smtpHost)
	if err := smtp.SendMail(m.smtpHost+":"+m.smtpPort, auth, m.GmailUsername, []string{to}, message.Bytes()); err != nil {
		return fmt.Errorf("send email via Gmail: %w", err)
	}
	return nil
}

// OTPEmail renders the verification email body.
func (m *Mailer) OTPEmail(name, code, purpose string, expiresAt time.Time) string {
	greeting := "Hello"
	if strings.TrimSpace(name) != "" {
		greeting = "Hello " + html.EscapeString(name)
	}
	action := "verify your identity"
	switch purpose {
	case "password_reset":
		action = "reset your password"
	case "phone_verification":
		action = "verify your phone number"
	case "first_password_change":
		action = "set up your account"
	}
	expiry := "10 minutes"
	if !expiresAt.IsZero() {
		remaining := time.Until(expiresAt)
		if remaining > 0 {
			expiry = fmt.Sprintf("%d minutes", int(remaining.Minutes())+1)
		}
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;background:#f4f4f5;font-family:Arial,Helvetica,sans-serif;color:#18181b;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:32px 16px;">
<tr><td align="center">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#ffffff;border-radius:12px;padding:32px;">
<tr><td>
<h2 style="margin:0 0 12px;font-size:22px;">Edwins Lab</h2>
<p style="margin:0 0 20px;font-size:15px;line-height:1.6;">%s, use the verification code below to %s.</p>
<div style="font-size:34px;letter-spacing:10px;font-weight:700;background:#f4f4f5;border-radius:8px;padding:18px;text-align:center;">%s</div>
<p style="margin:20px 0 0;font-size:14px;line-height:1.6;color:#52525b;">This code expires in %s. If you did not request it, you can ignore this email.</p>
</td></tr></table>
<p style="margin:16px 0 0;font-size:12px;color:#71717a;">Edwins Lab &middot; Innovation through practice</p>
</td></tr></table>
</body></html>`, greeting, action, html.EscapeString(code), expiry)
}

// NotificationEmail renders a simple transactional notification body.
func (m *Mailer) NotificationEmail(title, message string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;background:#f4f4f5;font-family:Arial,Helvetica,sans-serif;color:#18181b;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:32px 16px;">
<tr><td align="center">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#ffffff;border-radius:12px;padding:32px;">
<tr><td>
<h2 style="margin:0 0 12px;font-size:22px;">%s</h2>
<p style="margin:0;font-size:15px;line-height:1.6;color:#3f3f46;">%s</p>
<p style="margin:24px 0 0;font-size:12px;color:#71717a;">Sent by Edwins Lab</p>
</td></tr></table>
</td></tr></table>
</body></html>`, html.EscapeString(title), html.EscapeString(message))
}

func (m *Mailer) RegistrationConfirmationEmail(data RegistrationConfirmationData) string {
	name := strings.TrimSpace(data.Name)
	if name == "" {
		name = "there"
	}
	loginURL := ""
	if base := strings.TrimSpace(m.FrontendURL); base != "" {
		loginURL = strings.TrimRight(base, "/") + "/login"
	}

	identifierRow := ""
	if strings.TrimSpace(data.IdentifierLabel) != "" && strings.TrimSpace(data.Identifier) != "" {
		identifierRow = fmt.Sprintf(
			`<tr><td style="padding:10px 0;border-bottom:1px solid #e7e9e2;color:#62685f;font-size:14px;">%s</td><td align="right" style="padding:10px 0;border-bottom:1px solid #e7e9e2;color:#1c1c1c;font-size:14px;font-weight:700;">%s</td></tr>`,
			html.EscapeString(data.IdentifierLabel),
			html.EscapeString(data.Identifier),
		)
	}
	loginRow := ""
	if strings.TrimSpace(data.LoginID) != "" {
		loginRow = fmt.Sprintf(
			`<tr><td style="padding:10px 0;border-bottom:1px solid #e7e9e2;color:#62685f;font-size:14px;">Login ID</td><td align="right" style="padding:10px 0;border-bottom:1px solid #e7e9e2;color:#1c1c1c;font-size:14px;font-weight:700;">%s</td></tr>`,
			html.EscapeString(data.LoginID),
		)
	}
	yearRow := ""
	if data.AcademicYear > 0 {
		yearRow = fmt.Sprintf(
			`<tr><td style="padding:10px 0;border-bottom:1px solid #e7e9e2;color:#62685f;font-size:14px;">Year</td><td align="right" style="padding:10px 0;border-bottom:1px solid #e7e9e2;color:#1c1c1c;font-size:14px;font-weight:700;">Year %d</td></tr>`,
			data.AcademicYear,
		)
	}
	button := ""
	if loginURL != "" {
		button = fmt.Sprintf(
			`<table role="presentation" cellpadding="0" cellspacing="0" style="margin:28px 0 0;"><tr><td style="border-radius:8px;background:#1c1c1c;"><a href="%s" style="display:inline-block;padding:13px 22px;color:#ffffff;text-decoration:none;font-size:14px;font-weight:700;">Sign in to Edwins Lab</a></td></tr></table>`,
			html.EscapeString(loginURL),
		)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html><body style="margin:0;background:#f4f5f0;font-family:Arial,Helvetica,sans-serif;color:#1c1c1c;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f5f0;padding:36px 16px;">
<tr><td align="center">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width:560px;background:#ffffff;border:1px solid #e7e9e2;border-radius:14px;overflow:hidden;">
<tr><td style="background:#1c1c1c;padding:26px 32px;">
<p style="margin:0;color:#93c47d;font-size:12px;font-weight:700;letter-spacing:1.5px;text-transform:uppercase;">Edwins Lab</p>
<h1 style="margin:8px 0 0;color:#ffffff;font-size:24px;line-height:1.3;">Membership registration successful</h1>
</td></tr>
<tr><td style="padding:32px;">
<p style="margin:0;font-size:16px;line-height:1.65;">Hello %s,</p>
<p style="margin:12px 0 0;font-size:15px;line-height:1.7;color:#3f453d;">Your Edwins Lab membership is active. Your fingerprint enrollment is complete, and your member account is ready to use.</p>
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="margin:24px 0 0;border-top:1px solid #e7e9e2;">
%s
%s
%s
</table>
<p style="margin:22px 0 0;font-size:14px;line-height:1.7;color:#3f453d;">Sign in with your login ID. If you have not set a password yet, choose <strong>Forgot password</strong> on the sign-in screen and follow the email instructions.</p>
%s
<p style="margin:28px 0 0;font-size:12px;line-height:1.7;color:#71766d;">Need help? Contact the lab team at <a href="mailto:%s" style="color:#3f453d;">%s</a>.</p>
</td></tr>
<tr><td style="background:#f8f9f5;padding:18px 32px;border-top:1px solid #e7e9e2;">
<p style="margin:0;font-size:11px;line-height:1.6;color:#81877d;">Edwins Lab &middot; Innovation through practice</p>
</td></tr>
</table>
</td></tr>
</table>
</body></html>`,
		html.EscapeString(name),
		identifierRow,
		loginRow,
		yearRow,
		button,
		labEmail,
		labEmail,
	)
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
