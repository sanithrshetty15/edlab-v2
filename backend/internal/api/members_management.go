package api

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/phone"
)

var (
	loginIDPattern  = regexp.MustCompile(`^[\w.-]+$`)
	usnPattern      = regexp.MustCompile(`^[A-Z0-9]{6,15}$`)
	rollPattern     = regexp.MustCompile(`^[\w-]+$`)
	emailPattern    = regexp.MustCompile(`^[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+$`)
	memberPhoneExpr = regexp.MustCompile(`^\+?[1-9]\d{7,14}$`)
)

type createMemberRequest struct {
	Name                 string `json:"name"`
	USN                  string `json:"usn"`
	RollNumber           string `json:"roll_number"`
	AcademicYear         int    `json:"academic_year"`
	Branch               string `json:"branch"`
	Section              string `json:"section"`
	Email                string `json:"email"`
	Phone                string `json:"phone"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
	LoginID              string `json:"login_id"`
}

func (s *Server) listMembers(c *gin.Context) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.Member{})

	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if academicYear := strings.TrimSpace(c.Query("academic_year")); academicYear != "" {
		query = query.Where("academic_year = ?", academicYear)
	}
	if branch := strings.TrimSpace(c.Query("branch")); branch != "" {
		query = query.Where("branch = ?", branch)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		pattern := "%" + search + "%"
		if c.Query("search_field") == "login_id" {
			query = query.Where("login_id ILIKE ? OR usn ILIKE ? OR roll_number ILIKE ?", pattern, pattern, pattern)
		} else {
			query = query.Where("name ILIKE ? OR login_id ILIKE ? OR email ILIKE ? OR usn ILIKE ? OR roll_number ILIKE ?", pattern, pattern, pattern, pattern, pattern)
		}
	}

	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.Log.Error("member count failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load members")
		return
	}

	var members []models.Member
	if err := query.Order("created_at DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&members).Error; err != nil {
		s.Log.Error("member list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load members")
		return
	}

	items := make([]gin.H, 0, len(members))
	for i := range members {
		items = append(items, serializeManagedMemberSummary(&members[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"members":    items,
		"pagination": makePagination(page, perPage, total),
	})
}

func (s *Server) createMember(c *gin.Context) {
	var request createMemberRequest
	if !bindJSON(c, &request) {
		return
	}

	name := strings.TrimSpace(request.Name)
	loginID := strings.TrimSpace(request.LoginID)
	email := strings.ToLower(strings.TrimSpace(request.Email))
	usn := strings.ToUpper(strings.TrimSpace(request.USN))
	roll := strings.TrimSpace(request.RollNumber)
	branch := strings.TrimSpace(request.Branch)
	section := strings.TrimSpace(request.Section)
	normalized := phone.Normalize(request.Phone)
	password := request.Password

	validationErrors := []string{}
	if name == "" {
		validationErrors = append(validationErrors, "Name is required")
	}
	if loginID == "" {
		validationErrors = append(validationErrors, "Login ID is required")
	} else if !loginIDPattern.MatchString(loginID) {
		validationErrors = append(validationErrors, "Login ID can only contain letters, numbers, dots, and hyphens")
	}
	if email == "" {
		validationErrors = append(validationErrors, "Email is required")
	} else if !emailPattern.MatchString(email) {
		validationErrors = append(validationErrors, "Please provide a valid email address")
	}
	if strings.TrimSpace(request.Phone) == "" {
		validationErrors = append(validationErrors, "Phone number is required")
	} else if !phone.IsValid(normalized) || !memberPhoneExpr.MatchString(normalized) {
		validationErrors = append(validationErrors, "Please provide a valid phone number")
	}
	if request.AcademicYear < 1 || request.AcademicYear > 8 {
		validationErrors = append(validationErrors, "Academic year must be between 1 and 8")
	}
	if branch == "" {
		validationErrors = append(validationErrors, "Branch is required")
	}
	if len(password) < 8 {
		validationErrors = append(validationErrors, "Password must be at least 8 characters")
	}
	if request.PasswordConfirmation != "" && request.PasswordConfirmation != password {
		validationErrors = append(validationErrors, "Password confirmation does not match")
	}
	if usn != "" && !usnPattern.MatchString(usn) {
		validationErrors = append(validationErrors, "USN must be 6-15 letters and digits")
	}
	if roll != "" && !rollPattern.MatchString(roll) {
		validationErrors = append(validationErrors, "Roll number contains invalid characters")
	}
	if len(validationErrors) > 0 {
		failValidation(c, strings.Join(validationErrors, ", "), validationErrors)
		return
	}

	ctx := c.Request.Context()
	var count int64
	if err := s.DB.WithContext(ctx).Model(&models.Member{}).Where("login_id = ?", loginID).Count(&count).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to create member")
		return
	}
	if count > 0 {
		fail(c, http.StatusUnprocessableEntity, "This login ID is already registered")
		return
	}
	if usn != "" {
		if err := s.DB.WithContext(ctx).Model(&models.Member{}).Where("usn = ?", usn).Count(&count).Error; err != nil {
			fail(c, http.StatusInternalServerError, "Unable to create member")
			return
		}
		if count > 0 {
			fail(c, http.StatusUnprocessableEntity, "This USN is already registered")
			return
		}
	}
	if roll != "" {
		if err := s.DB.WithContext(ctx).Model(&models.Member{}).Where("roll_number = ?", roll).Count(&count).Error; err != nil {
			fail(c, http.StatusInternalServerError, "Unable to create member")
			return
		}
		if count > 0 {
			fail(c, http.StatusUnprocessableEntity, "This roll number is already registered")
			return
		}
	}
	if err := s.DB.WithContext(ctx).Model(&models.User{}).Where("LOWER(email) = ?", email).Count(&count).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to create member")
		return
	}
	if count > 0 {
		fail(c, http.StatusUnprocessableEntity, "An account with this email already exists")
		return
	}
	if err := s.DB.WithContext(ctx).Model(&models.User{}).Where("phone = ?", normalized).Count(&count).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to create member")
		return
	}
	if count > 0 {
		fail(c, http.StatusUnprocessableEntity, "An account with this phone number already exists")
		return
	}

	hash, err := s.Security.GenerateHash(password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to create member")
		return
	}

	var member models.Member
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user := models.User{
			Email:          email,
			Phone:          &normalized,
			PasswordDigest: &hash,
			Role:           "member",
			Active:         true,
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
			Phone:              normalized,
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
		s.Log.Error("member creation failed", "error", err)
		fail(c, http.StatusUnprocessableEntity, "Unable to create member")
		return
	}

	var user models.User
	if err := s.DB.WithContext(ctx).First(&user, member.UserID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load created member")
		return
	}

	s.audit(c, "member_created_by_admin", "Member", &member.ID, gin.H{
		"login_id":      member.LoginID,
		"academic_year": member.AcademicYear,
		"created_by":    currentUser(c).Email,
	})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Member account created successfully",
		"member":  serializeManagedMemberFull(&member, &user),
	})
}

func (s *Server) showMember(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var member models.Member
	if err := s.DB.WithContext(c.Request.Context()).First(&member, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	var user models.User
	if err := s.DB.WithContext(c.Request.Context()).First(&user, member.UserID).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "member": serializeManagedMemberFull(&member, &user)})
}

func (s *Server) updateMemberStatus(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if !bindJSON(c, &request) {
		return
	}
	newStatus := strings.ToLower(strings.TrimSpace(request.Status))
	if newStatus != "active" && newStatus != "inactive" && newStatus != "deactivated" {
		fail(c, http.StatusUnprocessableEntity, "Invalid status. Must be one of: active, inactive, deactivated")
		return
	}

	var member models.Member
	if err := s.DB.WithContext(c.Request.Context()).First(&member, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	if member.Status == newStatus {
		fail(c, http.StatusUnprocessableEntity, "Member already has this status")
		return
	}
	previous := member.Status

	ctx := c.Request.Context()
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Member{}).Where("id = ?", member.ID).Update("status", newStatus).Error; err != nil {
			return err
		}
		if newStatus == "active" {
			return tx.Model(&models.User{}).Where("id = ?", member.UserID).Update("active", true).Error
		}
		if err := tx.Model(&models.User{}).Where("id = ?", member.UserID).Update("active", false).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", member.UserID).Delete(&models.Session{}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to update member status")
		return
	}

	s.audit(c, "member_status_changed", "Member", &member.ID, gin.H{"from": previous, "to": newStatus})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Member status updated to " + newStatus,
		"member": gin.H{
			"id":              member.ID,
			"login_id":        member.LoginID,
			"name":            member.Name,
			"email":           member.Email,
			"status":          newStatus,
			"previous_status": previous,
		},
	})
}

func (s *Server) updateMemberFields(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var request struct {
		Name         *string `json:"name"`
		USN          *string `json:"usn"`
		RollNumber   *string `json:"roll_number"`
		AcademicYear *int    `json:"academic_year"`
		Branch       *string `json:"branch"`
		Section      *string `json:"section"`
		Phone        *string `json:"phone"`
	}
	if !bindJSON(c, &request) {
		return
	}

	var member models.Member
	if err := s.DB.WithContext(c.Request.Context()).First(&member, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}

	updates := map[string]any{}
	userUpdates := map[string]any{}
	changes := gin.H{}
	validationErrors := []string{}
	ctx := c.Request.Context()

	if request.Name != nil && strings.TrimSpace(*request.Name) != "" && member.Name != strings.TrimSpace(*request.Name) {
		name := strings.TrimSpace(*request.Name)
		updates["name"] = name
		changes["name"] = gin.H{"from": member.Name, "to": name}
	}
	if request.USN != nil {
		usn := strings.ToUpper(strings.TrimSpace(*request.USN))
		if usn != "" && stringOrEmpty(member.USN) != usn {
			if !usnPattern.MatchString(usn) {
				validationErrors = append(validationErrors, "USN must be 6-15 letters and digits")
			} else {
				var count int64
				if err := s.DB.WithContext(ctx).Model(&models.Member{}).Where("usn = ? AND id <> ?", usn, member.ID).Count(&count).Error; err != nil {
					fail(c, http.StatusInternalServerError, "Unable to update member")
					return
				}
				if count > 0 {
					fail(c, http.StatusUnprocessableEntity, "This USN is already in use by another member")
					return
				}
				updates["usn"] = usn
				changes["usn"] = gin.H{"from": member.USN, "to": usn}
			}
		}
	}
	if request.RollNumber != nil {
		roll := strings.TrimSpace(*request.RollNumber)
		if roll != "" && stringOrEmpty(member.RollNumber) != roll {
			if !rollPattern.MatchString(roll) {
				validationErrors = append(validationErrors, "Roll number contains invalid characters")
			} else {
				var count int64
				if err := s.DB.WithContext(ctx).Model(&models.Member{}).Where("roll_number = ? AND id <> ?", roll, member.ID).Count(&count).Error; err != nil {
					fail(c, http.StatusInternalServerError, "Unable to update member")
					return
				}
				if count > 0 {
					fail(c, http.StatusUnprocessableEntity, "This roll number is already in use by another member")
					return
				}
				updates["roll_number"] = roll
				changes["roll_number"] = gin.H{"from": member.RollNumber, "to": roll}
			}
		}
	}
	if request.AcademicYear != nil && member.AcademicYear != *request.AcademicYear {
		if *request.AcademicYear < 1 || *request.AcademicYear > 8 {
			validationErrors = append(validationErrors, "Academic year must be between 1 and 8")
		} else {
			updates["academic_year"] = *request.AcademicYear
			changes["academic_year"] = gin.H{"from": member.AcademicYear, "to": *request.AcademicYear}
		}
	}
	if request.Branch != nil {
		branch := strings.TrimSpace(*request.Branch)
		if branch != "" && member.Branch != branch {
			updates["branch"] = branch
			changes["branch"] = gin.H{"from": member.Branch, "to": branch}
		}
	}
	if request.Section != nil {
		section := strings.TrimSpace(*request.Section)
		if stringOrEmpty(member.Section) != section {
			updates["section"] = nullableString(section)
			changes["section"] = gin.H{"from": member.Section, "to": section}
		}
	}
	if request.Phone != nil {
		phoneValue := strings.TrimSpace(*request.Phone)
		if phoneValue != "" && member.Phone != phoneValue {
			if !memberPhoneExpr.MatchString(phoneValue) {
				validationErrors = append(validationErrors, "Invalid phone number format")
			} else {
				var count int64
				if err := s.DB.WithContext(ctx).Model(&models.User{}).Where("phone = ? AND id <> ?", phoneValue, member.UserID).Count(&count).Error; err != nil {
					fail(c, http.StatusInternalServerError, "Unable to update member")
					return
				}
				if count > 0 {
					fail(c, http.StatusUnprocessableEntity, "This phone number is already in use")
					return
				}
				updates["phone"] = phoneValue
				userUpdates["phone"] = phoneValue
				changes["phone"] = gin.H{"from": member.Phone, "to": phoneValue}
			}
		}
	}

	if len(validationErrors) > 0 {
		failValidation(c, strings.Join(validationErrors, ", "), validationErrors)
		return
	}
	if len(updates) == 0 {
		fail(c, http.StatusUnprocessableEntity, "No changes provided")
		return
	}

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Member{}).Where("id = ?", member.ID).Updates(updates).Error; err != nil {
			return err
		}
		if len(userUpdates) > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", member.UserID).Updates(userUpdates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.Log.Error("member field update failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to update member")
		return
	}

	var updated models.Member
	if err := s.DB.WithContext(ctx).First(&updated, member.ID).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load member")
		return
	}

	s.audit(c, "official_fields_updated", "Member", &member.ID, changes)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Official fields updated successfully",
		"member":  serializeManagedMember(&updated),
	})
}

func (s *Server) forceMemberPasswordReset(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var member models.Member
	if err := s.DB.WithContext(c.Request.Context()).First(&member, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}

	ctx := c.Request.Context()
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Member{}).Where("id = ?", member.ID).Update("must_change_password", true).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", member.UserID).Delete(&models.Session{}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to force password reset")
		return
	}

	s.audit(c, "password_reset_forced", "Member", &member.ID, gin.H{
		"member_login_id": member.LoginID,
		"member_name":     member.Name,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Member will be required to change password on next login",
	})
}

func (s *Server) deactivateMember(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var member models.Member
	if err := s.DB.WithContext(c.Request.Context()).First(&member, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}

	now := time.Now().UTC()
	ctx := c.Request.Context()
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Member{}).Where("id = ?", member.ID).Updates(map[string]any{
			"status":     "deactivated",
			"deleted_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", member.UserID).Update("active", false).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", member.UserID).Delete(&models.Session{}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to deactivate member")
		return
	}

	s.audit(c, "member_deleted", "Member", &member.ID, gin.H{
		"login_id": member.LoginID,
		"name":     member.Name,
		"email":    member.Email,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Member account deactivated successfully"})
}

