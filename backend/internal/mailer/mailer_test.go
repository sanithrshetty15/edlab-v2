package mailer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type failingGmailAPISender struct {
	called bool
	err    error
}

func (s *failingGmailAPISender) Enabled() bool {
	return true
}

func (s *failingGmailAPISender) Authorized(context.Context) (bool, error) {
	return true, nil
}

func (s *failingGmailAPISender) Send(context.Context, []byte) error {
	s.called = true
	return s.err
}

func TestRegistrationConfirmationEmail(t *testing.T) {
	mail := &Mailer{FrontendURL: "https://edwinslab.in/"}
	content := mail.RegistrationConfirmationEmail(RegistrationConfirmationData{
		Name:            `<script>alert("x")</script>`,
		LoginID:         "4AL25CS001",
		IdentifierLabel: "USN",
		Identifier:      "4AL25CS001",
		AcademicYear:    2,
	})

	if !strings.Contains(content.HTML, "Welcome to Edwins Lab!") {
		t.Fatal("registration email is missing its heading")
	}
	if !strings.Contains(content.HTML, "https://edwinslab.in/login") {
		t.Fatal("registration email is missing the sign-in link")
	}
	if !strings.Contains(content.HTML, "4AL25CS001") {
		t.Fatal("registration email is missing the login ID")
	}
	if strings.Contains(content.HTML, "<script>") {
		t.Fatal("registration email did not escape the member name")
	}
	if RegistrationConfirmationSubject != "EdwinsLab – Membership Registration Successful" {
		t.Fatalf("unexpected subject: %q", RegistrationConfirmationSubject)
	}
}

func TestDefaultSenderUsesLabGmail(t *testing.T) {
	t.Setenv("MAILER_FROM", "")
	t.Setenv("GMAIL_USERNAME", "")
	mail := New()
	if mail.From != "Edwins Lab <edwin.lab.aiet@gmail.com>" {
		t.Fatalf("default sender = %q", mail.From)
	}
}

func TestGmailAPIProviderDoesNotFallBack(t *testing.T) {
	service := &failingGmailAPISender{err: errors.New("gmail api failed")}
	mail := &Mailer{
		Provider:         ProviderGmailAPI,
		ResendAPIKey:     "resend-key",
		GmailUsername:    "sender@example.com",
		GmailAppPassword: "app-password",
		GmailAPI:         service,
		From:             "Edwins Lab <sender@example.com>",
	}
	err := mail.SendEmailContent("member@example.com", "Subject", EmailContent{HTML: "<p>Body</p>"})
	if err == nil || !strings.Contains(err.Error(), "gmail api failed") {
		t.Fatalf("SendEmailContent error = %v", err)
	}
	if !service.called {
		t.Fatal("Gmail API sender was not called")
	}
}

func TestGmailAPIProviderFailsClosedWithoutSender(t *testing.T) {
	mail := &Mailer{
		Provider:         ProviderGmailAPI,
		ResendAPIKey:     "resend-key",
		GmailUsername:    "sender@example.com",
		GmailAppPassword: "app-password",
		From:             "Edwins Lab <sender@example.com>",
	}
	err := mail.SendEmailContent("member@example.com", "Subject", EmailContent{HTML: "<p>Body</p>"})
	if err == nil || !strings.Contains(err.Error(), "Gmail API is not configured") {
		t.Fatalf("SendEmailContent error = %v", err)
	}
}
