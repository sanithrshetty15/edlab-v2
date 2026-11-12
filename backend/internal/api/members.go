package api

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/twilio"
)

type profileUpdateRequest struct {
	GitHubURL    *string   `json:"github_url"`
	LinkedInURL  *string   `json:"linkedin_url"`
	PortfolioURL *string   `json:"portfolio_url"`
	Bio          *string   `json:"bio"`
	Interests    *[]string `json:"interests"`
}

type phoneVerificationRequest struct {
	OTPCode        string `json:"otp_code"`
	DeliveryMethod string `json:"delivery_method"`
}

func (s *Server) memberRegister(c *gin.Context) {
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"message": "Public registration is disabled. Member accounts are created by lab administrators.",
	})
}

func (s *Server) showCurrentMember(c *gin.Context) {
	member := currentMember(c)
	if member == nil {
		user := currentUser(c)
		if user != nil {
			var loaded models.Member
			if err := s.DB.WithContext(c.Request.Context()).Where("user_id = ? AND deleted_at IS NULL", user.ID).First(&loaded).Error; err == nil {
				member = &loaded
			}
		}
	}
	if member == nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"member":  serializeMember(member),
	})
}

func (s *Server) updateMemberProfile(c *gin.Context) {
	member := currentMember(c)
	if member == nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	var request profileUpdateRequest
	if !bindJSON(c, &request) {
		return
	}

	updates := map[string]any{}
	errorsList := []string{}

	applyURL := func(field string, value *string) {
		if value == nil {
			return
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed == "" {
			updates[field] = nil
			return
		}
		parsed, err := url.ParseRequestURI(trimmed)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			errorsList = append(errorsList, "Profile links must be valid http or https URLs")
			return
		}
		if len(trimmed) > 500 {
			errorsList = append(errorsList, "Profile links must be 500 characters or fewer")
			return
		}
		updates[field] = trimmed
	}

	applyURL("github_url", request.GitHubURL)
	applyURL("linkedin_url", request.LinkedInURL)
	applyURL("portfolio_url", request.PortfolioURL)

	if request.Bio != nil {
		bio := strings.TrimSpace(*request.Bio)
		if len(bio) > 1000 {
			errorsList = append(errorsList, "Bio must be 1000 characters or fewer")
		} else if bio == "" {
			updates["bio"] = nil
		} else {
			updates["bio"] = bio
		}
	}

	if request.Interests != nil {
		interests := make([]string, 0, len(*request.Interests))
		for _, interest := range *request.Interests {
			interest = strings.TrimSpace(interest)
			if interest == "" {
				continue
			}
			if len(interest) > 50 {
				errorsList = append(errorsList, "Each interest must be 50 characters or fewer")
				break
			}
			interests = append(interests, interest)
			if len(interests) > 20 {
				errorsList = append(errorsList, "You can list up to 20 interests")
				break
			}
		}
		updates["interests"] = pq.StringArray(interests)
	}

	if len(errorsList) > 0 {
		failValidation(c, strings.Join(errorsList, ", "), errorsList)
		return
	}
	if len(updates) == 0 {
		fail(c, http.StatusUnprocessableEntity, "No profile changes provided")
		return
	}

	if err := s.DB.WithContext(c.Request.Context()).Model(&models.Member{}).Where("id = ?", member.ID).Updates(updates).Error; err != nil {
		s.Log.Error("member profile update failed", "member_id", member.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Profile update failed")
		return
	}

	var updated models.Member
	if err := s.DB.WithContext(c.Request.Context()).First(&updated, member.ID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Profile update failed")
		return
	}

	s.audit(c, "profile_updated", "Member", &member.ID, updates)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Profile updated successfully",
		"member":  serializeMember(&updated),
	})
}

func (s *Server) acceptLabRules(c *gin.Context) {
	member := currentMember(c)
	if member == nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}

	var version models.LabRulesVersion
	err := s.DB.WithContext(c.Request.Context()).Where("is_current = ?", true).Order("version_number DESC").First(&version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.Log.Error("no current lab rules version found")
		fail(c, http.StatusInternalServerError, "Lab rules are not available")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load lab rules")
		return
	}

	var existing int64
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.LabRulesAcceptance{}).
		Where("member_id = ? AND lab_rules_version_id = ?", member.ID, version.ID).
		Count(&existing).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to accept lab rules")
		return
	}
	if existing > 0 {
		fail(c, http.StatusUnprocessableEntity, "Lab rules already accepted")
		return
	}

	acceptance := models.LabRulesAcceptance{
		MemberID:          member.ID,
		LabRulesVersionID: version.ID,
		AcceptedAt:        time.Now().UTC(),
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&acceptance).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to accept lab rules")
		return
	}

	s.audit(c, "lab_rules_accepted", "Member", &member.ID, gin.H{"version_number": version.VersionNumber})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Lab rules accepted successfully"})
}

func (s *Server) requestPhoneVerification(c *gin.Context) {
	member := currentMember(c)
	if member == nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	if strings.TrimSpace(member.Phone) == "" {
		fail(c, http.StatusUnprocessableEntity, "No phone number on file to verify")
		return
	}

	ctx := c.Request.Context()
	now := time.Now().UTC()
	var recent int64
	if err := s.DB.WithContext(ctx).Model(&models.OTPCode{}).
		Where("user_id = ? AND purpose = ? AND created_at > ?", member.UserID, "phone_verification", now.Add(-s.Config.OTPRequestWindow)).
		Count(&recent).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to send verification code")
		return
	}
	if recent >= int64(s.Config.OTPMaxRequests) {
		fail(c, http.StatusUnprocessableEntity, "Too many verification requests. Please try again later.")
		return
	}

	client := twilio.NewFromEnv()
	if !client.Configured() {
		fail(c, http.StatusUnprocessableEntity, "SMS verification is not configured")
		return
	}

	randomCode, err := s.Security.GenerateOTP()
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to send verification code")
		return
	}
	record := models.OTPCode{
		UserID:         member.UserID,
		OTPDigest:      s.Security.DigestOTP(randomCode),
		DeliveryMethod: "sms",
		Purpose:        "phone_verification",
		ExpiresAt:      now.Add(s.Config.OTPExpiry),
	}
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.OTPCode{}).
			Where("user_id = ? AND purpose = ? AND used = ?", member.UserID, "phone_verification", false).
			Update("used", true).Error; err != nil {
			return err
		}
		return tx.Create(&record).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to send verification code")
		return
	}

	if err := client.SendVerification(member.Phone); err != nil {
		_ = s.DB.WithContext(ctx).Model(&models.OTPCode{}).Where("id = ?", record.ID).Update("used", true).Error
		s.Log.Error("Twilio verification request failed", "member_id", member.ID, "error", err)
		fail(c, http.StatusUnprocessableEntity, "Unable to send verification code")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Verification code sent to your phone"})
}

func (s *Server) verifyPhone(c *gin.Context) {
	member := currentMember(c)
	if member == nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	var request phoneVerificationRequest
	if !bindJSON(c, &request) {
		return
	}
	code := strings.TrimSpace(request.OTPCode)
	if code == "" || strings.ToLower(strings.TrimSpace(request.DeliveryMethod)) != "sms" {
		fail(c, http.StatusUnprocessableEntity, "OTP code and SMS delivery method are required")
		return
	}
	if strings.TrimSpace(member.Phone) == "" {
		fail(c, http.StatusUnprocessableEntity, "No phone number on file to verify")
		return
	}
	if len(code) < 4 || len(code) > 10 {
		fail(c, http.StatusUnauthorized, "Invalid or expired verification code")
		return
	}

	client := twilio.NewFromEnv()
	if !client.Configured() {
		fail(c, http.StatusUnprocessableEntity, "SMS verification is not configured")
		return
	}
	approved, err := client.CheckVerification(member.Phone, code)
	if err != nil {
		s.Log.Error("Twilio verification check failed", "member_id", member.ID, "error", err)
		fail(c, http.StatusUnauthorized, "Invalid or expired verification code")
		return
	}
	if !approved {
		fail(c, http.StatusUnauthorized, "Invalid or expired verification code")
		return
	}

	ctx := c.Request.Context()
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.OTPCode{}).
			Where("user_id = ? AND purpose = ? AND delivery_method = ? AND used = ?", member.UserID, "phone_verification", "sms", false).
			Update("used", true).Error; err != nil {
			return err
		}
		return tx.Model(&models.Member{}).Where("id = ?", member.ID).Update("phone_verified", true).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to verify code")
		return
	}

	s.audit(c, "phone_verified", "Member", &member.ID, nil)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Phone number verified successfully"})
}

