package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/phone"
	"github.com/sanithrshetty15/edwinslab/backend/internal/security"
)

type memberLoginRequest struct {
	LoginType string `json:"login_type"`
	LoginID   string `json:"login_id"`
	Password  string `json:"password"`
}

type adminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword      string `json:"current_password"`
	NewPassword          string `json:"new_password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

type passwordResetRequest struct {
	LoginID        string `json:"login_id"`
	DeliveryMethod string `json:"delivery_method"`
}

type passwordResetCompleteRequest struct {
	LoginID              string `json:"login_id"`
	OTP                  string `json:"otp"`
	NewPassword          string `json:"new_password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

func (s *Server) createSession(c *gin.Context, user *models.User) (string, *models.Session, error) {
	token, err := s.Security.GenerateSessionToken()
	if err != nil {
		return "", nil, err
	}
	now := time.Now().UTC()
	ttl := s.Config.SessionTTLMember
	if user.Role == "admin" {
		ttl = s.Config.SessionTTLAdmin
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	expiresAt := now.Add(ttl)

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()
	session := models.Session{
		UserID:       user.ID,
		TokenDigest:  s.Security.TokenDigest(token),
		IPAddress:    nullableString(ip),
		UserAgent:    nullableString(userAgent),
		LastActiveAt: now,
		ExpiresAt:    &expiresAt,
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&session).Error; err != nil {
		return "", nil, err
	}
	return token, &session, nil
}

func (s *Server) memberLogin(c *gin.Context) {
	var request memberLoginRequest
	if !bindJSON(c, &request) {
		return
	}
	loginType := strings.ToLower(strings.TrimSpace(request.LoginType))
	loginID := strings.TrimSpace(request.LoginID)
	password := request.Password

	if loginType != "usn" && loginType != "roll_number" {
		fail(c, http.StatusUnprocessableEntity, "Invalid login type")
		return
	}
	if loginID == "" || password == "" {
		fail(c, http.StatusUnprocessableEntity, "Login ID and password are required")
		return
	}
	if loginType == "usn" {
		loginID = strings.ToUpper(loginID)
	}

	column := "usn"
	if loginType == "roll_number" {
		column = "roll_number"
	}

	var member models.Member
	err := s.DB.WithContext(c.Request.Context()).
		Where(column+" = ? AND status = ? AND deleted_at IS NULL", loginID, "active").
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusUnauthorized, "Invalid login credentials")
		return
	}
	if err != nil {
		s.Log.Error("member login lookup failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to sign in")
		return
	}

	var user models.User
	if err := s.DB.WithContext(c.Request.Context()).First(&user, member.UserID).Error; err != nil {
		fail(c, http.StatusUnauthorized, "Invalid login credentials")
		return
	}
	if !user.Active || user.PasswordDigest == nil || !s.Security.VerifyHash(password, *user.PasswordDigest) {
		fail(c, http.StatusUnauthorized, "Invalid login credentials")
		return
	}

	token, _, err := s.createSession(c, &user)
	if err != nil {
		s.Log.Error("member session creation failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to sign in")
		return
	}

	s.audit(c, "member_login", "User", &user.ID, gin.H{"login_type": loginType})
	c.JSON(http.StatusOK, gin.H{
		"success":              true,
		"message":              "Login successful",
		"token":                token,
		"must_change_password": member.MustChangePassword,
		"user": gin.H{
			"id":         user.ID,
			"name":       member.Name,
			"email":      member.Email,
			"role":       user.Role,
			"login_type": loginType,
		},
	})
}

func (s *Server) adminLogin(c *gin.Context) {
	var request adminLoginRequest
	if !bindJSON(c, &request) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(request.Email))
	if email == "" || request.Password == "" {
		fail(c, http.StatusUnprocessableEntity, "Email and password are required")
		return
	}

	var user models.User
	err := s.DB.WithContext(c.Request.Context()).Where("LOWER(email) = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusUnauthorized, "Invalid login credentials")
		return
	}
	if err != nil {
		s.Log.Error("admin login lookup failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to sign in")
		return
	}
	if user.Role != "admin" || !user.Active || user.PasswordDigest == nil || !s.Security.VerifyHash(request.Password, *user.PasswordDigest) {
		fail(c, http.StatusUnauthorized, "Invalid login credentials")
		return
	}

	var admin models.Admin
	if err := s.DB.WithContext(c.Request.Context()).Where("user_id = ?", user.ID).First(&admin).Error; err != nil {
		fail(c, http.StatusUnauthorized, "Invalid login credentials")
		return
	}

	token, _, err := s.createSession(c, &user)
	if err != nil {
		s.Log.Error("admin session creation failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to sign in")
		return
	}

	s.audit(c, "admin_login", "User", &user.ID, nil)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Admin login successful",
		"token":   token,
		"admin": gin.H{
			"id":         admin.ID,
			"name":       admin.Name,
			"email":      user.Email,
			"is_primary": admin.IsPrimary,
		},
	})
}

func (s *Server) logout(c *gin.Context) {
	session := currentSession(c)
	if session != nil {
		if err := s.DB.WithContext(c.Request.Context()).Delete(&models.Session{}, session.ID).Error; err != nil {
			s.Log.Error("session deletion failed", "error", err)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Logged out successfully"})
}

func (s *Server) me(c *gin.Context) {
	user := currentUser(c)
	if user == nil {
		fail(c, http.StatusUnauthorized, "Authentication token is required")
		return
	}
	userData := gin.H{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	}

	if user.Role == "admin" {
		var admin models.Admin
		if err := s.DB.WithContext(c.Request.Context()).Where("user_id = ?", user.ID).First(&admin).Error; err == nil {
			userData["admin"] = gin.H{
				"id":         admin.ID,
				"name":       admin.Name,
				"is_primary": admin.IsPrimary,
			}
		}
	}

	if user.Role == "member" {
		var member models.Member
		if err := s.DB.WithContext(c.Request.Context()).Where("user_id = ? AND deleted_at IS NULL", user.ID).First(&member).Error; err == nil {
			userData["member"] = gin.H{
				"id":                   member.ID,
				"name":                 member.Name,
				"usn":                  member.USN,
				"roll_number":          member.RollNumber,
				"must_change_password": member.MustChangePassword,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "user": userData})
}

func (s *Server) changePassword(c *gin.Context) {
	var request changePasswordRequest
	if !bindJSON(c, &request) {
		return
	}
	if strings.TrimSpace(request.CurrentPassword) == "" || strings.TrimSpace(request.NewPassword) == "" || strings.TrimSpace(request.PasswordConfirmation) == "" {
		fail(c, http.StatusUnprocessableEntity, "All password fields are required")
		return
	}
	user := currentUser(c)
	if user == nil || user.PasswordDigest == nil {
		fail(c, http.StatusUnauthorized, "Authentication token is required")
		return
	}
	if !s.Security.VerifyHash(request.CurrentPassword, *user.PasswordDigest) {
		fail(c, http.StatusUnauthorized, "Current password is incorrect")
		return
	}
	if request.NewPassword != request.PasswordConfirmation {
		fail(c, http.StatusUnprocessableEntity, "New password and confirmation do not match")
		return
	}
	if len(request.NewPassword) < 8 {
		fail(c, http.StatusUnprocessableEntity, "Password must contain at least 8 characters")
		return
	}
	if request.NewPassword == request.CurrentPassword {
		fail(c, http.StatusUnprocessableEntity, "New password must be different from the current password")
		return
	}

	hash, err := s.Security.GenerateHash(request.NewPassword)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to update password")
		return
	}

	ctx := c.Request.Context()
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("password_digest", hash).Error; err != nil {
			return err
		}
		if user.Role == "member" {
			if err := tx.Model(&models.Member{}).Where("user_id = ?", user.ID).Update("must_change_password", false).Error; err != nil {
				return err
			}
		}
		session := currentSession(c)
		query := tx.Where("user_id = ?", user.ID)
		if session != nil {
			query = query.Where("id <> ?", session.ID)
		}
		return query.Delete(&models.Session{}).Error
	})
	if err != nil {
		s.Log.Error("password change failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to update password")
		return
	}

	s.audit(c, "password_changed", "User", &user.ID, nil)
	c.JSON(http.StatusOK, gin.H{
		"success":              true,
		"message":              "Password changed successfully",
		"must_change_password": false,
	})
}

func (s *Server) passwordResetRequestHandler(c *gin.Context) {
	var request passwordResetRequest
	if !bindJSON(c, &request) {
		return
	}
	loginID := strings.TrimSpace(request.LoginID)
	delivery := strings.ToLower(strings.TrimSpace(request.DeliveryMethod))
	if loginID == "" || delivery != "email" {
		fail(c, http.StatusUnprocessableEntity, "A login ID and email delivery method are required")
		return
	}

	genericResponse := gin.H{
		"success": true,
		"message": "If the account exists, a verification code will be sent",
	}

	var member models.Member
	err := s.DB.WithContext(c.Request.Context()).
		Where("login_id = ? AND status = ? AND deleted_at IS NULL", loginID, "active").
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, genericResponse)
		return
	}
	if err != nil {
		s.Log.Error("password reset lookup failed", "error", err)
		c.JSON(http.StatusOK, genericResponse)
		return
	}

	var user models.User
	if err := s.DB.WithContext(c.Request.Context()).First(&user, member.UserID).Error; err != nil || !user.Active {
		c.JSON(http.StatusOK, genericResponse)
		return
	}

	code, err := s.issueEmailOTP(c, &user, "password_reset")
	if err != nil {
		s.Log.Error("password reset OTP delivery failed", "user_id", user.ID, "error", err)
	} else {
		s.Log.Info("password reset verification code issued", "user_id", user.ID, "email", maskEmail(user.Email))
		_ = code
	}

	c.JSON(http.StatusOK, genericResponse)
}

func (s *Server) passwordResetComplete(c *gin.Context) {
	var request passwordResetCompleteRequest
	if !bindJSON(c, &request) {
		return
	}
	loginID := strings.TrimSpace(request.LoginID)
	code := strings.TrimSpace(request.OTP)
	newPassword := request.NewPassword

	if newPassword == "" || len(newPassword) < 8 {
		fail(c, http.StatusUnprocessableEntity, "Password must contain at least 8 characters")
		return
	}
	if newPassword != request.PasswordConfirmation {
		fail(c, http.StatusUnprocessableEntity, "Password confirmation does not match")
		return
	}
	if len(code) < 4 {
		fail(c, http.StatusUnprocessableEntity, "Invalid or expired verification code")
		return
	}

	var member models.Member
	err := s.DB.WithContext(c.Request.Context()).
		Where("login_id = ? AND status = ? AND deleted_at IS NULL", loginID, "active").
		First(&member).Error
	if err != nil {
		fail(c, http.StatusUnprocessableEntity, "Invalid login ID or verification code")
		return
	}

	var user models.User
	if err := s.DB.WithContext(c.Request.Context()).First(&user, member.UserID).Error; err != nil || !user.Active {
		fail(c, http.StatusUnprocessableEntity, "Invalid login ID or verification code")
		return
	}

	valid := false
	ctx := c.Request.Context()
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var otp models.OTPCode
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND purpose = ? AND delivery_method = ? AND used = ? AND expires_at > ?",
				user.ID, "password_reset", "email", false, time.Now().UTC()).
			Order("created_at DESC").
			First(&otp).Error
		if err != nil {
			return nil
		}
		if otp.Attempts >= s.Config.OTPMaxAttempts {
			return nil
		}
		otp.Attempts++
		valid = s.Security.ValidateOTP(code, otp.OTPDigest)
		if valid {
			otp.Used = true
		}
		return tx.Save(&otp).Error
	})
	if err != nil {
		s.Log.Error("password reset verification failed", "error", err)
		fail(c, http.StatusUnprocessableEntity, "Invalid or expired verification code")
		return
	}
	if !valid {
		fail(c, http.StatusUnprocessableEntity, "Invalid or expired verification code")
		return
	}

	hash, err := s.Security.GenerateHash(newPassword)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to reset password")
		return
	}
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("password_digest", hash).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Member{}).Where("id = ?", member.ID).Update("must_change_password", false).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", user.ID).Delete(&models.Session{}).Error
	})
	if err != nil {
		s.Log.Error("password reset failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to reset password")
		return
	}

	s.audit(c, "password_reset_completed", "User", &user.ID, nil)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password reset successfully. Please log in again.",
	})
}

func (s *Server) issueEmailOTP(c *gin.Context, user *models.User, purpose string) (string, error) {
	ctx := c.Request.Context()
	now := time.Now().UTC()

	var recent int64
	if err := s.DB.WithContext(ctx).Model(&models.OTPCode{}).
		Where("user_id = ? AND purpose = ? AND created_at > ?", user.ID, purpose, now.Add(-s.Config.OTPRequestWindow)).
		Count(&recent).Error; err != nil {
		return "", err
	}
	if recent >= int64(s.Config.OTPMaxRequests) {
		return "", security.ErrPasswordResetRateLimited
	}

	code, err := s.Security.GenerateOTP()
	if err != nil {
		return "", err
	}
	expiresAt := now.Add(s.Config.OTPExpiry)
	otp := models.OTPCode{
		UserID:         user.ID,
		OTPDigest:      s.Security.DigestOTP(code),
		DeliveryMethod: "email",
		Purpose:        purpose,
		ExpiresAt:      expiresAt,
	}

	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.OTPCode{}).
			Where("user_id = ? AND purpose = ? AND used = ?", user.ID, purpose, false).
			Update("used", true).Error; err != nil {
			return err
		}
		return tx.Create(&otp).Error
	})
	if err != nil {
		return "", err
	}

	recipient := user.Email
	if s.Config.OTPTestRecipient != "" {
		recipient = s.Config.OTPTestRecipient
	}
	body := s.Mailer.OTPEmail(userDisplayName(s.DB, user), code, purpose, expiresAt)
	if err := s.enqueueEmail(ctx, recipient, "Your Edwins Lab Verification Code", body); err != nil {
		_ = s.DB.WithContext(ctx).Model(&models.OTPCode{}).Where("id = ?", otp.ID).Update("used", true).Error
		return "", err
	}
	return code, nil
}

func userDisplayName(db *gorm.DB, user *models.User) string {
	if user.Role == "member" {
		var member models.Member
		if err := db.Where("user_id = ?", user.ID).First(&member).Error; err == nil {
			return member.Name
		}
	}
	if user.Role == "admin" {
		var admin models.Admin
		if err := db.Where("user_id = ?", user.ID).First(&admin).Error; err == nil {
			return admin.Name
		}
	}
	return ""
}

func maskEmail(value string) string {
	parts := strings.SplitN(value, "@", 2)
	if len(parts) != 2 || len(parts[0]) < 2 {
		return "***"
	}
	return parts[0][:1] + "***@" + parts[1]
}

func normalizedPhone(value string) (string, bool) {
	normalized := phone.Normalize(value)
	if !phone.IsValid(normalized) {
		return "", false
	}
	return normalized, true
}
