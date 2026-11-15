package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/mailer"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

const (
	registrationEmailPending = "pending"
	registrationEmailQueued  = "queued"
	registrationEmailSending = "sending"
	registrationEmailSent    = "sent"
	registrationEmailFailed  = "failed"
)

func registrationEmailError(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if runes := []rune(value); len(runes) > 500 {
		value = string(runes[:500])
	}
	if value == "" {
		value = "The confirmation email could not be delivered."
	}
	return value
}

func serializeRegistrationEmail(item *models.MemberRegistrationEmail) any {
	if item == nil {
		return nil
	}
	return gin.H{
		"id":         item.ID,
		"member_id":  item.MemberID,
		"recipient":  item.Recipient,
		"status":     item.Status,
		"attempts":   item.Attempts,
		"last_error": item.LastError,
		"queued_at":  item.QueuedAt,
		"sent_at":    item.SentAt,
		"created_at": item.CreatedAt,
		"updated_at": item.UpdatedAt,
	}
}

func (s *Server) loadRegistrationEmail(ctx context.Context, memberID int64) (*models.MemberRegistrationEmail, error) {
	var item models.MemberRegistrationEmail
	if err := s.DB.WithContext(ctx).Where("member_id = ?", memberID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Server) ensureRegistrationEmail(ctx context.Context, member *models.Member) (*models.MemberRegistrationEmail, error) {
	item, err := s.loadRegistrationEmail(ctx, member.ID)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	candidate := models.MemberRegistrationEmail{
		MemberID:  member.ID,
		Recipient: strings.ToLower(strings.TrimSpace(member.Email)),
		Status:    registrationEmailPending,
	}
	if err := s.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "member_id"}},
		DoNothing: true,
	}).Create(&candidate).Error; err != nil {
		return nil, err
	}
	return s.loadRegistrationEmail(ctx, member.ID)
}

func (s *Server) markRegistrationEmailFailed(ctx context.Context, item *models.MemberRegistrationEmail, message string) {
	if item == nil {
		return
	}
	now := time.Now().UTC()
	reason := registrationEmailError(message)
	result := s.DB.WithContext(ctx).Model(&models.MemberRegistrationEmail{}).
		Where("id = ? AND status <> ?", item.ID, registrationEmailSent).
		Updates(map[string]any{
			"status":     registrationEmailFailed,
			"last_error": reason,
			"updated_at": now,
		})
	if result.Error != nil {
		s.Log.Error("registration email failure update failed", "member_id", item.MemberID, "error", result.Error)
		return
	}
	item.Status = registrationEmailFailed
	item.LastError = &reason
	item.UpdatedAt = now
}

func (s *Server) queueRegistrationConfirmation(ctx context.Context, member *models.Member, force bool) *models.MemberRegistrationEmail {
	if member == nil || member.ID <= 0 || strings.TrimSpace(member.Email) == "" {
		return nil
	}
	item, err := s.ensureRegistrationEmail(ctx, member)
	if err != nil {
		s.Log.Error("registration email record creation failed", "member_id", member.ID, "error", err)
		return nil
	}
	if item.Status == registrationEmailSent {
		return item
	}
	recipient := strings.ToLower(strings.TrimSpace(member.Email))
	if recipient != "" && recipient != item.Recipient {
		if err := s.DB.WithContext(ctx).Model(&models.MemberRegistrationEmail{}).
			Where("id = ? AND status <> ?", item.ID, registrationEmailSent).
			Update("recipient", recipient).Error; err != nil {
			s.Log.Error("registration email recipient update failed", "member_id", member.ID, "error", err)
		} else {
			item.Recipient = recipient
		}
	}
	if !force && (item.Status == registrationEmailQueued || item.Status == registrationEmailSending || item.Status == registrationEmailFailed) {
		return item
	}
	if !s.EmailConfigured(ctx) {
		s.markRegistrationEmailFailed(ctx, item, "Email delivery is not configured")
		return item
	}
	if s.Jobs == nil {
		s.markRegistrationEmailFailed(ctx, item, "Email delivery queue is unavailable")
		return item
	}

	now := time.Now().UTC()
	if err := s.DB.WithContext(ctx).Model(&models.MemberRegistrationEmail{}).
		Where("id = ? AND status <> ?", item.ID, registrationEmailSent).
		Updates(map[string]any{
			"status":     registrationEmailQueued,
			"last_error": nil,
			"queued_at":  now,
			"updated_at": now,
		}).Error; err != nil {
		s.Log.Error("registration email queue update failed", "member_id", member.ID, "error", err)
		s.markRegistrationEmailFailed(ctx, item, "The confirmation email could not be queued")
		return item
	}

	payload, _ := json.Marshal(map[string]int64{"member_id": member.ID})
	if err := s.Jobs.Enqueue(ctx, "member.registration_email", payload, now); err != nil {
		s.Log.Error("registration email enqueue failed", "member_id", member.ID, "error", err)
		s.markRegistrationEmailFailed(ctx, item, err.Error())
		return item
	}

	refreshed, err := s.loadRegistrationEmail(ctx, member.ID)
	if err != nil {
		s.Log.Error("registration email reload failed", "member_id", member.ID, "error", err)
		return item
	}
	return refreshed
}

func (s *Server) registrationConfirmationEmailJob(ctx context.Context, payload json.RawMessage) error {
	var request struct {
		MemberID int64 `json:"member_id"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return fmt.Errorf("decode registration email job: %w", err)
	}
	if request.MemberID <= 0 {
		return fmt.Errorf("registration email job requires a member_id")
	}

	item, err := s.loadRegistrationEmail(ctx, request.MemberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if item.Status == registrationEmailSent {
		return nil
	}
	if !s.EmailConfigured(ctx) || s.Mailer == nil {
		s.markRegistrationEmailFailed(ctx, item, "Email delivery is not configured")
		return nil
	}

	now := time.Now().UTC()
	claim := s.DB.WithContext(ctx).Model(&models.MemberRegistrationEmail{}).
		Where("id = ? AND status IN ?", item.ID, []string{
			registrationEmailPending,
			registrationEmailQueued,
			registrationEmailFailed,
		}).
		Updates(map[string]any{
			"status":     registrationEmailSending,
			"attempts":   gorm.Expr("attempts + 1"),
			"last_error": nil,
			"updated_at": now,
		})
	if claim.Error != nil {
		return claim.Error
	}
	if claim.RowsAffected == 0 {
		return nil
	}

	var member models.Member
	if err := s.DB.WithContext(ctx).First(&member, request.MemberID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.markRegistrationEmailFailed(ctx, item, "Member record is no longer available")
			return nil
		}
		s.markRegistrationEmailFailed(ctx, item, err.Error())
		return err
	}

	identifierLabel := "USN"
	identifier := ""
	if member.AcademicYear == 1 {
		identifierLabel = "Roll Number"
		if member.RollNumber != nil {
			identifier = strings.TrimSpace(*member.RollNumber)
		}
	}
	if identifier == "" && member.USN != nil {
		identifierLabel = "USN"
		identifier = strings.TrimSpace(*member.USN)
	}
	content := s.Mailer.RegistrationConfirmationEmail(mailer.RegistrationConfirmationData{
		Name:            member.Name,
		LoginID:         member.LoginID,
		IdentifierLabel: identifierLabel,
		Identifier:      identifier,
		AcademicYear:    member.AcademicYear,
	})
	if err := s.Mailer.SendEmailContentContext(ctx, item.Recipient, mailer.RegistrationConfirmationSubject, content); err != nil {
		s.markRegistrationEmailFailed(ctx, item, err.Error())
		return fmt.Errorf("send registration confirmation email: %w", err)
	}

	sentAt := time.Now().UTC()
	if err := s.DB.WithContext(ctx).Model(&models.MemberRegistrationEmail{}).
		Where("id = ?", item.ID).
		Updates(map[string]any{
			"status":     registrationEmailSent,
			"last_error": nil,
			"sent_at":    sentAt,
			"updated_at": sentAt,
		}).Error; err != nil {
		return err
	}
	s.Log.Info("registration confirmation email sent", "member_id", member.ID, "recipient", item.Recipient)
	return nil
}

func (s *Server) memberRegistrationEmailStatus(c *gin.Context) {
	memberID, ok := pathID(c, "id")
	if !ok {
		return
	}
	var member models.Member
	if err := s.DB.WithContext(c.Request.Context()).First(&member, memberID).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	item, err := s.loadRegistrationEmail(c.Request.Context(), member.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"success":            true,
				"registration_email": nil,
			})
			return
		}
		fail(c, http.StatusInternalServerError, "Unable to load confirmation email")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"registration_email": serializeRegistrationEmail(item),
	})
}

func (s *Server) retryMemberRegistrationEmail(c *gin.Context) {
	memberID, ok := pathID(c, "id")
	if !ok {
		return
	}
	var member models.Member
	if err := s.DB.WithContext(c.Request.Context()).First(&member, memberID).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	if member.Status == memberStatusPendingEnrollment {
		fail(c, http.StatusUnprocessableEntity, "Complete fingerprint enrollment before sending the confirmation email")
		return
	}

	item := s.queueRegistrationConfirmation(c.Request.Context(), &member, true)
	if item == nil {
		fail(c, http.StatusUnprocessableEntity, "Unable to queue the confirmation email")
		return
	}

	message := "Confirmation email queued."
	switch item.Status {
	case registrationEmailSent:
		message = "Confirmation email was already sent."
	case registrationEmailFailed:
		message = "Confirmation email could not be queued."
	}
	s.audit(c, "member_registration_email_retried", "Member", &member.ID, gin.H{
		"recipient": item.Recipient,
		"status":    item.Status,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"message":            message,
		"registration_email": serializeRegistrationEmail(item),
	})
}
