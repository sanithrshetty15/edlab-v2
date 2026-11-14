package mailer

import (
	"fmt"
	"html"
	"strings"
	"time"
)

const (
	defaultEmailSiteURL = "https://edwinslab.vercel.app"
	emailLogoPath       = "/images/email/edwins-lab-logo.png"
)

type emailLayout struct {
	Preview  string
	Eyebrow  string
	Heading  string
	Greeting string
	Intro    string
	Content  string
	Closing  string
}

// OTPEmail renders the shared verification email with a four-digit code and a
// plain-text alternative.
func (m *Mailer) OTPEmail(recipient, code, purpose string, expiresAt time.Time) EmailContent {
	greeting := "Hello,"
	if recipient = strings.TrimSpace(recipient); recipient != "" {
		greeting = "Hello " + recipient + ","
	}

	expiry := expiryLabel(expiresAt)
	code = strings.TrimSpace(code)
	content := renderOTPContent(code, expiry)
	return m.renderEmailLayout(emailLayout{
		Preview:  "Your Edwins Lab verification code expires in " + expiry + ".",
		Eyebrow:  "ACCOUNT VERIFICATION",
		Heading:  "Verify your email address",
		Greeting: greeting,
		Intro:    "We received a request to verify your account with Edwins Lab. Use the verification code below to continue.",
		Content:  content.HTML,
		Closing:  "Thanks,\nThe Edwins Lab team",
	}, content.Text)
}

// NotificationEmail renders a shared-layout transactional notification.
func (m *Mailer) NotificationEmail(title, message string) EmailContent {
	title = strings.TrimSpace(title)
	return m.renderEmailLayout(emailLayout{
		Preview: title,
		Eyebrow: "EDWINS LAB",
		Heading: title,
		Intro:   message,
	}, "")
}

// RegistrationConfirmationEmail renders the membership confirmation with the
// shared email layout and a plain-text alternative.
func (m *Mailer) RegistrationConfirmationEmail(data RegistrationConfirmationData) EmailContent {
	name := strings.TrimSpace(data.Name)
	greeting := "Hello,"
	if name != "" {
		greeting = "Hello " + name + ","
	}

	identifierLabel := strings.TrimSpace(data.IdentifierLabel)
	if identifierLabel == "" {
		identifierLabel = "USN"
	}
	identifier := strings.TrimSpace(data.Identifier)
	if identifier == "" {
		identifier = "Not provided"
	}
	academicYear := "Not provided"
	if data.AcademicYear > 0 {
		academicYear = fmt.Sprintf("Year %d", data.AcademicYear)
	}

	var details strings.Builder
	details.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%;border-collapse:collapse;">`)
	writeDetailRow(&details, "Name", valueOrFallback(data.Name))
	writeDetailRow(&details, identifierLabel, identifier)
	writeDetailRow(&details, "Academic Year", academicYear)
	writeDetailRow(&details, "Fingerprint Enrollment", "Complete")
	details.WriteString(`</table>`)

	loginLine := ""
	if loginID := strings.TrimSpace(data.LoginID); loginID != "" {
		loginLine = fmt.Sprintf(
			`<p style="margin:20px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.65;color:#475569;">Your login ID is <strong style="color:#0f2744;">%s</strong>.</p>`,
			html.EscapeString(loginID),
		)
	}

	access := `<p style="margin:22px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.65;color:#475569;">You can now sign in to Edwins Lab with your login ID. If you have not set a password yet, choose <strong style="color:#0f2744;">Forgot password</strong> on the sign-in screen and follow the email instructions.</p>`

	button := ""
	loginURL := m.loginURL()
	if loginURL != "" {
		button = fmt.Sprintf(
			`<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:26px 0 0;"><tr><td align="center" bgcolor="#1d4ed8" style="border-radius:10px;"><a href="%s" style="display:inline-block;padding:14px 24px;font-family:Arial,Helvetica,sans-serif;font-size:15px;font-weight:700;line-height:1.2;color:#ffffff;text-decoration:none;border-radius:10px;">Sign in to Edwins Lab</a></td></tr></table>`,
			html.EscapeString(loginURL),
		)
	}

	contentHTML := details.String() + loginLine + access + button
	contentText := registrationDetailsText(data, identifierLabel, identifier, academicYear, loginURL)
	return m.renderEmailLayout(emailLayout{
		Preview:  "Your Edwins Lab membership is active.",
		Eyebrow:  "MEMBERSHIP CONFIRMATION",
		Heading:  "Welcome to Edwins Lab!",
		Greeting: greeting,
		Intro:    "Congratulations! Your membership registration with Edwins Lab has been successfully completed.\n\nYour member information:\n- Name: " + valueOrFallback(data.Name) + "\n- USN / Roll Number: " + identifierLabel + ": " + identifier + "\n- Academic Year: " + academicYear + "\n- Fingerprint Enrollment: Successfully completed\n\nYour registration and fingerprint enrollment are complete. You can now use your registered fingerprint to access the lab, subject to the lab's access authorization system.\n\nWe are glad to have you as a member of Edwins Lab.",
		Content:  contentHTML,
		Closing:  "We're glad to have you in the lab.",
	}, contentText)
}

func (m *Mailer) renderEmailLayout(layout emailLayout, contentText string) EmailContent {
	greetingHTML := ""
	if greeting := strings.TrimSpace(layout.Greeting); greeting != "" {
		greetingHTML = `<p style="margin:0 0 12px;font-family:Arial,Helvetica,sans-serif;font-size:16px;line-height:1.65;color:#334155;">` + html.EscapeString(greeting) + `</p>`
	}
	introHTML := ""
	if intro := strings.TrimSpace(layout.Intro); intro != "" {
		introHTML = `<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:16px;line-height:1.65;color:#334155;">` + strings.ReplaceAll(html.EscapeString(intro), "\n", "<br>") + `</p>`
	}
	closingHTML := ""
	if closing := strings.TrimSpace(layout.Closing); closing != "" {
		closingHTML = `<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.65;color:#475569;">` + strings.ReplaceAll(html.EscapeString(closing), "\n", "<br>") + `</p>`
	}

	replacer := strings.NewReplacer(
		"{{preview}}", html.EscapeString(layout.Preview),
		"{{title}}", html.EscapeString(layout.Heading),
		"{{logo_url}}", html.EscapeString(m.logoURL()),
		"{{eyebrow}}", html.EscapeString(layout.Eyebrow),
		"{{heading}}", html.EscapeString(layout.Heading),
		"{{greeting}}", greetingHTML,
		"{{intro}}", introHTML,
		"{{content}}", layout.Content,
		"{{closing}}", closingHTML,
		"{{lab_email}}", labEmail,
	)

	var text strings.Builder
	text.WriteString("EDWINS LAB\n")
	if eyebrow := strings.TrimSpace(layout.Eyebrow); eyebrow != "" {
		text.WriteString(eyebrow + "\n")
	}
	text.WriteString("\n")
	if heading := strings.TrimSpace(layout.Heading); heading != "" {
		text.WriteString(heading + "\n\n")
	}
	if greeting := strings.TrimSpace(layout.Greeting); greeting != "" {
		text.WriteString(greeting + "\n\n")
	}
	if intro := strings.TrimSpace(layout.Intro); intro != "" {
		text.WriteString(intro + "\n\n")
	}
	if contentText = strings.TrimSpace(contentText); contentText != "" {
		text.WriteString(contentText + "\n\n")
	}
	if closing := strings.TrimSpace(layout.Closing); closing != "" {
		text.WriteString(closing + "\n\n")
	}
	text.WriteString("Need help? Contact " + labEmail + ".\n")
	text.WriteString("This is an automated message from Edwins Lab. Please do not reply.\n\n")
	text.WriteString("Edwins Lab\nOpen Source Technology Lab\n")

	return EmailContent{
		HTML: replacer.Replace(emailHTMLTemplate),
		Text: text.String(),
	}
}

func (m *Mailer) siteURL() string {
	base := strings.TrimSpace(m.FrontendURL)
	if base == "" {
		base = defaultEmailSiteURL
	}
	return strings.TrimRight(base, "/")
}

func (m *Mailer) logoURL() string {
	return m.siteURL() + emailLogoPath
}

func (m *Mailer) loginURL() string {
	base := m.siteURL()
	if base == "" {
		return ""
	}
	return base + "/login"
}

func expiryLabel(expiresAt time.Time) string {
	if expiresAt.IsZero() {
		return "5 minutes"
	}
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return "5 minutes"
	}
	minutes := int(remaining/time.Minute) + 1
	if minutes == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", minutes)
}

func valueOrFallback(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Not provided"
	}
	return value
}

func writeDetailRow(builder *strings.Builder, label, value string) {
	fmt.Fprintf(
		builder,
		`<tr><td style="padding:12px 0;border-bottom:1px solid #e2e8f0;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:1.5;color:#64748b;">%s</td><td align="right" style="padding:12px 0;border-bottom:1px solid #e2e8f0;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:1.5;font-weight:700;color:#0f2744;">%s</td></tr>`,
		html.EscapeString(label),
		html.EscapeString(value),
	)
}

func registrationDetailsText(data RegistrationConfirmationData, identifierLabel, identifier, academicYear, loginURL string) string {
	var builder strings.Builder
	builder.WriteString("Name: " + valueOrFallback(data.Name) + "\n")
	builder.WriteString(identifierLabel + ": " + identifier + "\n")
	builder.WriteString("Academic Year: " + academicYear + "\n")
	builder.WriteString("Fingerprint Enrollment: Complete\n")
	if loginID := strings.TrimSpace(data.LoginID); loginID != "" {
		builder.WriteString("\nYour login ID is " + loginID + ".\n")
	}
	builder.WriteString("\nYou can now sign in to Edwins Lab with your login ID. If you have not set a password yet, choose Forgot password on the sign-in screen and follow the email instructions.\n")
	if loginURL != "" {
		builder.WriteString("\nSign in: " + loginURL + "\n")
	}
	return builder.String()
}

func renderOTPContent(code, expiry string) EmailContent {
	replacer := strings.NewReplacer(
		"{{code}}", html.EscapeString(code),
		"{{expiry}}", html.EscapeString(expiry),
	)
	htmlBody := replacer.Replace(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%;border-collapse:separate;background-color:#eff6ff;border:1px solid #bfdbfe;border-radius:12px;">
<tr><td align="center" style="padding:22px 16px 20px;">
<p style="margin:0 0 10px;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:1.4;font-weight:700;letter-spacing:1.6px;text-transform:uppercase;color:#2563eb;">YOUR VERIFICATION CODE</p>
<p style="margin:0;font-family:'Courier New',Courier,monospace;font-size:38px;line-height:1.1;font-weight:700;letter-spacing:10px;color:#1d4ed8;">{{code}}</p>
</td></tr></table>
<p style="margin:18px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.65;color:#475569;">This code expires in <strong style="color:#0f2744;">{{expiry}}</strong>.</p>
<p style="margin:10px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:14px;line-height:1.65;color:#64748b;">If you did not request this code, you can safely ignore this email. Never share this code with anyone.</p>`)
	textBody := "YOUR VERIFICATION CODE\n" + code + "\n\nThis code expires in " + expiry + ".\n\nIf you did not request this code, you can safely ignore this email. Never share this code with anyone."
	return EmailContent{HTML: htmlBody, Text: textBody}
}

const emailHTMLTemplate = `<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="x-apple-disable-message-reformatting">
<meta name="color-scheme" content="light only">
<meta name="supported-color-schemes" content="light">
<title>{{title}}</title>
<!--[if mso]><style>body,table,td,p,a{font-family:Arial,Helvetica,sans-serif !important;}</style><![endif]-->
</head>
<body style="margin:0;padding:0;background-color:#eef2f7;">
<div style="display:none;font-size:1px;line-height:1px;max-height:0;max-width:0;opacity:0;overflow:hidden;mso-hide:all;">{{preview}}</div>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%;background-color:#eef2f7;border-collapse:collapse;">
<tr><td align="center" style="padding:40px 16px;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="width:100%;max-width:600px;background-color:#ffffff;border:1px solid #e2e8f0;border-radius:16px;border-collapse:separate;overflow:hidden;">
<tr><td align="center" style="padding:32px 40px 24px;">
<img src="{{logo_url}}" width="148" alt="Edwins Lab" style="display:block;width:148px;max-width:148px;height:auto;border:0;outline:none;text-decoration:none;">
</td></tr>
<tr><td style="padding:0 40px;"><div style="height:1px;background-color:#bfdbfe;line-height:1px;font-size:1px;">&nbsp;</div></td></tr>
<tr><td style="padding:28px 40px 0;">
<p style="margin:0 0 10px;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:1.4;font-weight:700;letter-spacing:1.6px;text-transform:uppercase;color:#2563eb;">{{eyebrow}}</p>
<h1 style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:26px;line-height:1.3;font-weight:700;color:#0f2744;">{{heading}}</h1>
</td></tr>
<tr><td style="padding:18px 40px 0;font-family:Arial,Helvetica,sans-serif;font-size:16px;line-height:1.65;color:#334155;">
{{greeting}}
{{intro}}
</td></tr>
<tr><td style="padding:22px 40px 0;font-family:Arial,Helvetica,sans-serif;font-size:16px;line-height:1.65;color:#334155;">
{{content}}
</td></tr>
<tr><td style="padding:0 40px 32px;font-family:Arial,Helvetica,sans-serif;font-size:15px;line-height:1.65;color:#475569;">
{{closing}}
</td></tr>
<tr><td style="padding:22px 40px;background-color:#f8fafc;border-top:1px solid #e2e8f0;text-align:center;">
<p style="margin:0;font-family:Arial,Helvetica,sans-serif;font-size:13px;line-height:1.5;font-weight:700;color:#0f2744;">Edwins Lab</p>
<p style="margin:4px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:1.5;color:#64748b;">Open Source Technology Lab</p>
<p style="margin:10px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:12px;line-height:1.5;color:#64748b;">Need help? <a href="mailto:{{lab_email}}" style="color:#2563eb;text-decoration:none;">{{lab_email}}</a></p>
<p style="margin:6px 0 0;font-family:Arial,Helvetica,sans-serif;font-size:11px;line-height:1.5;color:#94a3b8;">This is an automated message. Please do not reply to this email.</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`
