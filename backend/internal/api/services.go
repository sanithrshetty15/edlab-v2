package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/mailer"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

type emailJobPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	HTML    string `json:"html"`
	Text    string `json:"text"`
}

func (s *Server) registerJobHandlers() {
	if s.Jobs == nil {
		return
	}
	handler := func(ctx context.Context, payload json.RawMessage) error {
		var item emailJobPayload
		if err := json.Unmarshal(payload, &item); err != nil {
			return fmt.Errorf("decode email job: %w", err)
		}
		if s.Mailer == nil {
			return fmt.Errorf("mailer unavailable")
		}
		content := mailer.EmailContent{
			HTML: item.HTML,
			Text: item.Text,
		}
		if content.HTML == "" && content.Text == "" {
			content.HTML = item.Body
		}
		return s.Mailer.SendEmailContent(item.To, item.Subject, content)
	}
	s.Jobs.Register("email_delivery", handler)
	s.Jobs.Register("mail.send", handler)
	s.Jobs.Register("biometric.process_event", s.processBiometricEvent)
	s.Jobs.Register("biometric.expire_enrollment_session", s.expireEnrollmentSessionJob)
	s.Jobs.Register("member.registration_email", s.registrationConfirmationEmailJob)
}

func (s *Server) EmailConfigured() bool {
	return s.Mailer != nil && (s.Mailer.ResendAPIKey != "" || (s.Mailer.GmailUsername != "" && s.Mailer.GmailAppPassword != ""))
}

func (s *Server) enqueueEmail(ctx context.Context, to, subject, body string) error {
	return s.enqueueEmailContent(ctx, to, subject, mailer.EmailContent{HTML: body})
}

func (s *Server) enqueueEmailContent(ctx context.Context, to, subject string, content mailer.EmailContent) error {
	if s.Jobs == nil {
		return fmt.Errorf("job manager unavailable")
	}
	payload, _ := json.Marshal(emailJobPayload{
		To:      to,
		Subject: subject,
		HTML:    content.HTML,
		Text:    content.Text,
	})
	return s.Jobs.Enqueue(ctx, "email_delivery", payload, time.Now().UTC())
}

func (s *Server) createNotification(tx *gorm.DB, userID int64, title, body, notificationType, subjectType string, subjectID *int64) error {
	db := s.DB
	if tx != nil {
		db = tx
	}
	var bodyValue *string
	if body != "" {
		bodyValue = &body
	}
	var subjectTypeValue *string
	if subjectType != "" {
		subjectTypeValue = &subjectType
	}
	return db.Create(&models.Notification{
		UserID:           userID,
		Title:            title,
		Body:             bodyValue,
		NotificationType: notificationType,
		SubjectType:      subjectTypeValue,
		SubjectID:        subjectID,
		IsRead:           false,
	}).Error
}

func (s *Server) notifyUser(ctx context.Context, userID int64, title, body, notificationType, subjectType string, subjectID *int64) {
	if err := s.createNotification(nil, userID, title, body, notificationType, subjectType, subjectID); err != nil {
		s.Log.Error("notification creation failed", "user_id", userID, "type", notificationType, "error", err)
	}
}

func (s *Server) notifyAdmins(ctx context.Context, title, body, notificationType, subjectType string, subjectID *int64) {
	var users []models.User
	if err := s.DB.WithContext(ctx).Where("role = ? AND active = ?", "admin", true).Find(&users).Error; err != nil {
		s.Log.Error("admin notification lookup failed", "error", err)
		return
	}
	for _, user := range users {
		s.notifyUser(ctx, user.ID, title, body, notificationType, subjectType, subjectID)
	}
}
