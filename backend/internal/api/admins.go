package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/phone"
)

type createAdminRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Phone     string `json:"phone"`
	IsPrimary bool   `json:"is_primary"`
}

func serializeAdmin(admin *models.Admin, user *models.User) gin.H {
	return gin.H{
		"id":         admin.ID,
		"name":       admin.Name,
		"email":      user.Email,
		"phone":      user.Phone,
		"is_primary": admin.IsPrimary,
		"created_at": admin.CreatedAt,
		"user": gin.H{
			"id":     user.ID,
			"active": user.Active,
		},
	}
}

func (s *Server) listAdmins(c *gin.Context) {
	var admins []models.Admin
	if err := s.DB.WithContext(c.Request.Context()).Order("created_at DESC").Find(&admins).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load administrators")
		return
	}
	items := make([]gin.H, 0, len(admins))
	for i := range admins {
		var user models.User
		if err := s.DB.WithContext(c.Request.Context()).First(&user, admins[i].UserID).Error; err != nil {
			continue
		}
		items = append(items, serializeAdmin(&admins[i], &user))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "admins": items})
}

func (s *Server) showAdmin(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var admin models.Admin
	if err := s.DB.WithContext(c.Request.Context()).First(&admin, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	var user models.User
	if err := s.DB.WithContext(c.Request.Context()).First(&user, admin.UserID).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "admin": serializeAdmin(&admin, &user)})
}

func (s *Server) createAdmin(c *gin.Context) {
	var request createAdminRequest
	if !bindJSON(c, &request) {
		return
	}
	name := strings.TrimSpace(request.Name)
	email := strings.ToLower(strings.TrimSpace(request.Email))
	password := request.Password
	phoneNumber := phone.Normalize(request.Phone)

	validationErrors := []string{}
	if name == "" {
		validationErrors = append(validationErrors, "Admin name is required")
	}
	if email == "" || !emailPattern.MatchString(email) {
		validationErrors = append(validationErrors, "Please provide a valid email address")
	}
	if len(password) < 8 {
		validationErrors = append(validationErrors, "Password must be at least 8 characters")
	}
	if strings.TrimSpace(request.Phone) != "" && !phone.IsValid(phoneNumber) {
		validationErrors = append(validationErrors, "Please provide a valid phone number")
	}
	if len(validationErrors) > 0 {
		failValidation(c, strings.Join(validationErrors, ", "), validationErrors)
		return
	}

	ctx := c.Request.Context()
	var adminCount int64
	if err := s.DB.WithContext(ctx).Model(&models.Admin{}).Count(&adminCount).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to create administrator")
		return
	}
	if adminCount >= 3 {
		fail(c, http.StatusUnprocessableEntity, "Maximum of 3 administrators allowed")
		return
	}
	if request.IsPrimary {
		var primaryCount int64
		if err := s.DB.WithContext(ctx).Model(&models.Admin{}).Where("is_primary = ?", true).Count(&primaryCount).Error; err != nil {
			fail(c, http.StatusInternalServerError, "Unable to create administrator")
			return
		}
		if primaryCount >= 1 {
			fail(c, http.StatusUnprocessableEntity, "Cannot create another primary admin. Maximum of one primary admin allowed.")
			return
		}
	}

	var existing int64
	if err := s.DB.WithContext(ctx).Model(&models.User{}).Where("LOWER(email) = ?", email).Count(&existing).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to create administrator")
		return
	}
	if existing > 0 {
		fail(c, http.StatusUnprocessableEntity, "An account with this email already exists")
		return
	}
	if strings.TrimSpace(request.Phone) != "" {
		if err := s.DB.WithContext(ctx).Model(&models.User{}).Where("phone = ?", phoneNumber).Count(&existing).Error; err != nil {
			fail(c, http.StatusInternalServerError, "Unable to create administrator")
			return
		}
		if existing > 0 {
			fail(c, http.StatusUnprocessableEntity, "An account with this phone number already exists")
			return
		}
	}

	hash, err := s.Security.GenerateHash(password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to create administrator")
		return
	}

	var admin models.Admin
	var user models.User
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user = models.User{
			Email:          email,
			PasswordDigest: &hash,
			Role:           "admin",
			Active:         true,
		}
		if strings.TrimSpace(request.Phone) != "" {
			user.Phone = &phoneNumber
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		admin = models.Admin{UserID: user.ID, Name: name, IsPrimary: request.IsPrimary}
		return tx.Create(&admin).Error
	})
	if err != nil {
		fail(c, http.StatusUnprocessableEntity, "Failed to create admin account")
		return
	}

	s.audit(c, "admin_created", "Admin", &admin.ID, gin.H{
		"admin_name":  name,
		"admin_email": email,
		"is_primary":  request.IsPrimary,
	})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Admin account created successfully",
		"admin":   serializeAdmin(&admin, &user),
	})
}

func (s *Server) updateAdmin(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var request struct {
		Name      *string `json:"name"`
		IsPrimary *bool   `json:"is_primary"`
	}
	if !bindJSON(c, &request) {
		return
	}
	var admin models.Admin
	if err := s.DB.WithContext(c.Request.Context()).First(&admin, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}

	changes := gin.H{}
	updates := map[string]any{}
	ctx := c.Request.Context()

	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name != "" && name != admin.Name {
			updates["name"] = name
			changes["name"] = gin.H{"from": admin.Name, "to": name}
		}
	}
	if request.IsPrimary != nil && *request.IsPrimary != admin.IsPrimary {
		if *request.IsPrimary {
			var primaryCount int64
			if err := s.DB.WithContext(ctx).Model(&models.Admin{}).Where("is_primary = ?", true).Count(&primaryCount).Error; err != nil {
				fail(c, http.StatusInternalServerError, "Unable to update administrator")
				return
			}
			if primaryCount >= 1 {
				fail(c, http.StatusUnprocessableEntity, "Only one primary admin is allowed")
				return
			}
		} else {
			var primaryCount int64
			if err := s.DB.WithContext(ctx).Model(&models.Admin{}).Where("is_primary = ?", true).Count(&primaryCount).Error; err != nil {
				fail(c, http.StatusInternalServerError, "Unable to update administrator")
				return
			}
			if primaryCount <= 1 {
				fail(c, http.StatusUnprocessableEntity, "Cannot remove the only primary admin")
				return
			}
		}
		updates["is_primary"] = *request.IsPrimary
		changes["is_primary"] = gin.H{"from": admin.IsPrimary, "to": *request.IsPrimary}
	}

	if len(updates) == 0 {
		fail(c, http.StatusUnprocessableEntity, "No changes provided")
		return
	}
	if err := s.DB.WithContext(ctx).Model(&models.Admin{}).Where("id = ?", admin.ID).Updates(updates).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to update administrator")
		return
	}
	if err := s.DB.WithContext(ctx).First(&admin, admin.ID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load administrator")
		return
	}
	var user models.User
	if err := s.DB.WithContext(ctx).First(&user, admin.UserID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load administrator")
		return
	}

	s.audit(c, "admin_updated", "Admin", &admin.ID, changes)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Admin updated successfully",
		"admin":   serializeAdmin(&admin, &user),
	})
}

func (s *Server) deleteAdmin(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	admin := currentAdmin(c)
	if admin != nil && admin.ID == id {
		fail(c, http.StatusForbidden, "You cannot remove your own admin account")
		return
	}
	var target models.Admin
	if err := s.DB.WithContext(c.Request.Context()).First(&target, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	var count int64
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.Admin{}).Count(&count).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to remove administrator")
		return
	}
	if count <= 1 {
		fail(c, http.StatusUnprocessableEntity, "At least one administrator is required")
		return
	}

	var user models.User
	if err := s.DB.WithContext(c.Request.Context()).First(&user, target.UserID).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}

	ctx := c.Request.Context()
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", target.UserID).Update("active", false).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", target.UserID).Delete(&models.Session{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Admin{}, target.ID).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to remove administrator")
		return
	}

	s.audit(c, "admin_removed", "Admin", &target.ID, gin.H{
		"admin_name":  target.Name,
		"admin_email": user.Email,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Admin account removed successfully"})
}

type inviteMemberRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	AcademicYear int    `json:"academic_year"`
	Branch       string `json:"branch"`
	Section      string `json:"section"`
	USN          string `json:"usn"`
	RollNumber   string `json:"roll_number"`
	Password     string `json:"password"`
}

func (s *Server) inviteAdmin(c *gin.Context) {
	var request inviteMemberRequest
	if !bindJSON(c, &request) {
		return
	}
	name := strings.TrimSpace(request.Name)
	email := strings.ToLower(strings.TrimSpace(request.Email))
	branch := strings.TrimSpace(request.Branch)
	usn := strings.ToUpper(strings.TrimSpace(request.USN))
	roll := strings.TrimSpace(request.RollNumber)
	section := strings.TrimSpace(request.Section)
	phoneNumber := phone.Normalize(request.Phone)

	validationErrors := []string{}
	if name == "" {
		validationErrors = append(validationErrors, "Name is required")
	}
	if email == "" || !emailPattern.MatchString(email) {
		validationErrors = append(validationErrors, "Please provide a valid email address")
	}
	if len(request.Password) < 8 {
		validationErrors = append(validationErrors, "Password must be at least 8 characters")
	}
	if request.AcademicYear < 1 || request.AcademicYear > 8 {
		validationErrors = append(validationErrors, "Academic year must be between 1 and 8")
	}
	if branch == "" {
		validationErrors = append(validationErrors, "Branch is required")
	}
	if strings.TrimSpace(request.Phone) != "" && !phone.IsValid(phoneNumber) {
		validationErrors = append(validationErrors, "Please provide a valid phone number")
	}
	if len(validationErrors) > 0 {
		failValidation(c, strings.Join(validationErrors, ", "), validationErrors)
		return
	}

	ctx := c.Request.Context()
	var existing int64
	if err := s.DB.WithContext(ctx).Model(&models.User{}).Where("LOWER(email) = ?", email).Count(&existing).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to send invitation")
		return
	}
	if existing > 0 {
		fail(c, http.StatusUnprocessableEntity, "An account with this email already exists")
		return
	}

	loginID := generateLoginID(email, name)
	for attempt := 0; attempt < 5; attempt++ {
		var loginCount int64
		if err := s.DB.WithContext(ctx).Model(&models.Member{}).Where("login_id = ?", loginID).Count(&loginCount).Error; err != nil {
			fail(c, http.StatusInternalServerError, "Unable to send invitation")
			return
		}
		if loginCount == 0 {
			break
		}
		loginID = fmt.Sprintf("%s_%d", loginID, time.Now().UnixNano()%10000)
	}

	hash, err := s.Security.GenerateHash(request.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to send invitation")
		return
	}

	var member models.Member
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user := models.User{
			Email:          email,
			PasswordDigest: &hash,
			Role:           "member",
			Active:         true,
		}
		if strings.TrimSpace(request.Phone) != "" {
			user.Phone = &phoneNumber
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		member = models.Member{
			UserID:             user.ID,
			Name:               name,
			LoginID:            loginID,
			AcademicYear:       request.AcademicYear,
			Branch:             branch,
			Email:              email,
			Phone:              phoneNumber,
			MustChangePassword: true,
			Status:             "active",
		}
		if usn != "" {
			member.USN = &usn
		}
		if roll != "" {
			member.RollNumber = &roll
		}
		if section != "" {
			member.Section = &section
		}
		return tx.Create(&member).Error
	})
	if err != nil {
		fail(c, http.StatusUnprocessableEntity, "Unable to send invitation")
		return
	}

	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	s.audit(c, "membership_invited", "Member", &member.ID, gin.H{
		"email":    email,
		"login_id": loginID,
	})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Invitation sent successfully",
		"invitation": gin.H{
			"email":         email,
			"temp_login_id": loginID,
			"expires_at":    expiresAt,
		},
	})
}

func generateLoginID(email, name string) string {
	base := sanitizeLoginID(strings.ToLower(name))
	if len(base) > 8 {
		base = base[:8]
	}
	domain := strings.SplitN(email, "@", 2)[0]
	domain = sanitizeLoginID(strings.ToLower(domain))
	if len(domain) > 6 {
		domain = domain[:6]
	}
	if base == "" {
		base = "member"
	}
	if domain == "" {
		domain = "lab"
	}
	return base + "_" + domain
}

func sanitizeLoginID(value string) string {
	var builder strings.Builder
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			builder.WriteRune(character)
		}
	}
	return builder.String()
}
