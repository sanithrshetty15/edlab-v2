package mailer

import (
	"strings"
	"testing"
)

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