package mailer

import (
	"strings"
	"testing"
)

func TestRegistrationConfirmationEmail(t *testing.T) {
	mail := &Mailer{FrontendURL: "https://edwinslab.in/"}
	body := mail.RegistrationConfirmationEmail(RegistrationConfirmationData{
		Name:            `<script>alert("x")</script>`,
		LoginID:         "4AL25CS001",
		IdentifierLabel: "USN",
		Identifier:      "4AL25CS001",
		AcademicYear:    2,
	})

	if !strings.Contains(body, "Registration successful") {
		t.Fatal("registration email is missing its heading")
	}
	if !strings.Contains(body, "https://edwinslab.in/login") {
		t.Fatal("registration email is missing the sign-in link")
	}
	if !strings.Contains(body, "4AL25CS001") {
		t.Fatal("registration email is missing the login ID")
	}
	if strings.Contains(body, "<script>") {
		t.Fatal("registration email did not escape the member name")
	}
	if RegistrationConfirmationSubject != "Welcome to Edwins Lab – Registration Successful!" {
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
