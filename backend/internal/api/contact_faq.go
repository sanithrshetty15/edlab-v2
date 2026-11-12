package api

import (
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
	"github.com/sanithrshetty15/edwinslab/backend/internal/security"
)

var contactEmailPattern = regexp.MustCompile(`(?i)^[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+$`)

func (s *Server) registerContactRoutes(api *gin.RouterGroup) {
	contact := api.Group("/contact")
	contact.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted())
	contact.POST("/messages", s.createContactMessage)
	contact.GET("/messages", s.RequireAdmin(), s.listContactMessages)
	contact.GET("/messages/:id", s.RequireAdmin(), s.showContactMessage)
	contact.PATCH("/messages/:id/status", s.RequireAdmin(), s.updateContactMessageStatus)
	contact.DELETE("/messages/:id", s.RequireAdmin(), s.destroyContactMessage)
	contact.POST("/update-email/request", s.RequirePrimaryAdmin(), s.requestContactEmailChange)
	contact.POST("/update-email/verify", s.RequirePrimaryAdmin(), s.verifyContactEmailChange)
}

func (s *Server) registerFAQRoutes(api *gin.RouterGroup) {
	faqs := api.Group("/faqs")
	faqs.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted())
	faqs.GET("", s.listFAQs)
	faqs.GET("/categories", s.listFAQCategories)
	faqs.GET("/:slug", s.showFAQCategory)
	faqs.POST("/categories", s.RequireAdmin(), s.createFAQCategory)
	faqs.POST("", s.RequireAdmin(), s.createFAQ)
	faqs.PATCH("/reorder", s.RequireAdmin(), s.reorderFAQs)
	faqs.PATCH("/:id", s.RequireAdmin(), s.updateFAQ)
	faqs.DELETE("/:id", s.RequireAdmin(), s.deleteFAQ)
}

func (s *Server) serializeContactMessage(message *models.ContactMessage) gin.H {
	serialized := gin.H{
		"id":         message.ID,
		"subject":    message.Subject,
		"category":   message.Category,
		"status":     message.Status,
		"created_at": message.CreatedAt,
		"updated_at": message.UpdatedAt,
	}
	if message.Member.ID > 0 {
		serialized["member"] = gin.H{
			"id":       message.Member.ID,
			"name":     message.Member.Name,
			"login_id": message.Member.LoginID,
		}
	}
	return serialized
}

func (s *Server) serializeContactMessageFull(message *models.ContactMessage) gin.H {
	serialized := s.serializeContactMessage(message)
	serialized["message"] = message.Message
	serialized["email_delivery_status"] = message.EmailDeliveryStatus
	return serialized
}

func (s *Server) createContactMessage(c *gin.Context) {
	member := currentMember(c)
	if member == nil || member.Status != "active" {
		fail(c, http.StatusForbidden, "Active member account required")
		return
	}
	var request struct {
		Subject  string `json:"subject"`
		Category string `json:"category"`
		Message  string `json:"message"`
	}
	if !bindJSON(c, &request) {
		return
	}
	subject := strings.TrimSpace(request.Subject)
	category := strings.ToLower(strings.TrimSpace(request.Category))
	message := strings.TrimSpace(request.Message)

	if subject == "" {
		fail(c, http.StatusUnprocessableEntity, "Subject is required")
		return
	}
	if category == "" {
		fail(c, http.StatusUnprocessableEntity, "Category is required")
		return
	}
	if message == "" {
		fail(c, http.StatusUnprocessableEntity, "Message is required")
		return
	}
	allowed := map[string]bool{
		"general": true, "inquiry": true, "feedback": true,
		"complaint": true, "suggestion": true, "other": true,
	}
	if !allowed[category] {
		fail(c, http.StatusUnprocessableEntity, "Invalid category. Must be one of: general, inquiry, feedback, complaint, suggestion, other")
		return
	}
	if len([]rune(message)) > 5000 {
		fail(c, http.StatusUnprocessableEntity, "Message is too long (max 5000 characters)")
		return
	}

	contactMessage := models.ContactMessage{
		MemberID: member.ID,
		Subject:  subject,
		Category: category,
		Message:  message,
		Status:   "new",
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&contactMessage).Error; err != nil {
		s.Log.Error("contact message create failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to send message")
		return
	}

	details := map[string]any{"subject": subject, "category": category}
	s.audit(c, "contact_message_sent", "ContactMessage", ptrInt64(contactMessage.ID), details)

	title := "Contact message received"
	body := "New message from " + member.Name + ": " + subject
	s.notifyAdmins(c.Request.Context(), title, body, "contact_message", "ContactMessage", ptrInt64(contactMessage.ID))

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Your message has been sent successfully",
		"contact_message": gin.H{
			"id":         contactMessage.ID,
			"subject":    contactMessage.Subject,
			"category":   contactMessage.Category,
			"status":     contactMessage.Status,
			"created_at": contactMessage.CreatedAt,
		},
	})
}

func (s *Server) listContactMessages(c *gin.Context) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.ContactMessage{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("category = ?", category)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("subject ILIKE ? OR message ILIKE ?", like, like)
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load contact messages")
		return
	}
	var messages []models.ContactMessage
	if err := query.Preload("Member").Order("created_at DESC").
		Offset(offsetFor(page, perPage)).Limit(perPage).Find(&messages).Error; err != nil {
		s.Log.Error("contact message list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load contact messages")
		return
	}
	result := make([]gin.H, 0, len(messages))
	for i := range messages {
		result = append(result, s.serializeContactMessage(&messages[i]))
	}
	var newCount int64
	s.DB.WithContext(c.Request.Context()).Model(&models.ContactMessage{}).Where("status = ?", "new").Count(&newCount)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"messages": result,
		"pagination": gin.H{
			"page":        page,
			"per_page":    perPage,
			"total_count": total,
			"new_count":   newCount,
		},
	})
}

func (s *Server) findContactMessage(c *gin.Context) (*models.ContactMessage, bool) {
	id, ok := pathID(c, "id")
	if !ok {
		return nil, false
	}
	var message models.ContactMessage
	if err := s.DB.WithContext(c.Request.Context()).Preload("Member").First(&message, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Contact message not found")
		return nil, false
	}
	return &message, true
}

func (s *Server) showContactMessage(c *gin.Context) {
	message, ok := s.findContactMessage(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message_data": s.serializeContactMessageFull(message),
	})
}

func (s *Server) updateContactMessageStatus(c *gin.Context) {
	message, ok := s.findContactMessage(c)
	if !ok {
		return
	}
	var request struct {
		Status string `json:"status"`
	}
	if !bindJSON(c, &request) {
		return
	}
	status := strings.ToLower(strings.TrimSpace(request.Status))
	allowed := map[string]bool{"new": true, "in_progress": true, "resolved": true, "closed": true}
	if !allowed[status] {
		fail(c, http.StatusUnprocessableEntity, "Invalid status. Must be one of: new, in_progress, resolved, closed")
		return
	}
	if message.Status == status {
		fail(c, http.StatusUnprocessableEntity, "Message already has this status")
		return
	}
	previous := message.Status
	if err := s.DB.WithContext(c.Request.Context()).Model(message).Update("status", status).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to update message status")
		return
	}
	message.Status = status
	s.audit(c, "contact_message_status_updated", "ContactMessage", ptrInt64(message.ID), map[string]any{
		"subject":         message.Subject,
		"previous_status": previous,
		"new_status":      status,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"message":         "Status updated to " + status,
		"contact_message": s.serializeContactMessage(message),
	})
}

func (s *Server) destroyContactMessage(c *gin.Context) {
	message, ok := s.findContactMessage(c)
	if !ok {
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Model(message).Update("status", "closed").Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to close contact message")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Contact message closed"})
}

func (s *Server) platformSettingValue(key string) string {
	var setting models.PlatformSetting
	if err := s.DB.Where("key = ?", key).First(&setting).Error; err != nil || setting.Value == nil {
		return ""
	}
	return *setting.Value
}

func (s *Server) setPlatformSetting(key, value string) error {
	var setting models.PlatformSetting
	err := s.DB.Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = models.PlatformSetting{Key: key, Value: &value}
		return s.DB.Create(&setting).Error
	}
	if err != nil {
		return err
	}
	return s.DB.Model(&setting).Update("value", value).Error
}

func (s *Server) deletePlatformSetting(key string) {
	s.DB.Where("key = ?", key).Delete(&models.PlatformSetting{})
}

func (s *Server) currentContactEmail() string {
	if value := strings.TrimSpace(s.platformSettingValue("contact_email_current")); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv("GMAIL_USERNAME"))
}

func (s *Server) requestContactEmailChange(c *gin.Context) {
	user := currentUser(c)
	if user == nil {
		fail(c, http.StatusUnauthorized, "Authentication token is required")
		return
	}
	var request struct {
		NewEmail string `json:"new_email"`
	}
	if !bindJSON(c, &request) {
		return
	}
	newEmail := strings.ToLower(strings.TrimSpace(request.NewEmail))
	if newEmail == "" {
		fail(c, http.StatusUnprocessableEntity, "New email address is required")
		return
	}
	if !contactEmailPattern.MatchString(newEmail) {
		fail(c, http.StatusUnprocessableEntity, "Please provide a valid email address")
		return
	}
	if newEmail == strings.ToLower(s.currentContactEmail()) {
		fail(c, http.StatusUnprocessableEntity, "New email must be different from current email")
		return
	}
	if requested := s.platformSettingValue("contact_email_change_requested_at"); requested != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, requested); err == nil && parsed.After(time.Now().UTC().AddDate(0, 0, -30)) {
			fail(c, http.StatusUnprocessableEntity, "Contact email can only be changed once every 30 days")
			return
		}
	}
	if err := s.setPlatformSetting("contact_email_change_pending", newEmail); err != nil {
		s.Log.Error("contact email pending setting failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to start email change")
		return
	}
	_ = s.setPlatformSetting("contact_email_change_requested_at", time.Now().UTC().Format(time.RFC3339Nano))
	if _, err := s.issueEmailOTP(c, user, "contact_email_change"); err != nil {
		if errors.Is(err, security.ErrPasswordResetRateLimited) {
			fail(c, http.StatusTooManyRequests, "Too many verification code requests. Please try again later.")
			return
		}
		s.Log.Error("contact email OTP delivery failed", "user_id", user.ID, "error", err)
		fail(c, http.StatusInternalServerError, "Unable to send verification code")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Verification code sent to your email"})
}

func (s *Server) verifyContactEmailChange(c *gin.Context) {
	user := currentUser(c)
	if user == nil {
		fail(c, http.StatusUnauthorized, "Authentication token is required")
		return
	}
	var request struct {
		NewEmail string `json:"new_email"`
		OTPCode  string `json:"otp_code"`
	}
	if !bindJSON(c, &request) {
		return
	}
	newEmail := strings.ToLower(strings.TrimSpace(request.NewEmail))
	otpCode := strings.TrimSpace(request.OTPCode)
	if newEmail == "" || otpCode == "" {
		fail(c, http.StatusUnprocessableEntity, "New email and verification code are required")
		return
	}
	pending := strings.ToLower(strings.TrimSpace(s.platformSettingValue("contact_email_change_pending")))
	if pending == "" || pending != newEmail {
		fail(c, http.StatusUnprocessableEntity, "No pending email change found for this address")
		return
	}

	valid := false
	ctx := c.Request.Context()
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var otp models.OTPCode
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND purpose = ? AND delivery_method = ? AND used = ? AND expires_at > ?",
				user.ID, "contact_email_change", "email", false, time.Now().UTC()).
			Order("created_at DESC").
			First(&otp).Error
		if err != nil {
			return nil
		}
		if otp.Attempts >= s.Config.OTPMaxAttempts {
			return nil
		}
		otp.Attempts++
		valid = s.Security.ValidateOTP(otpCode, otp.OTPDigest)
		if valid {
			otp.Used = true
		}
		return tx.Save(&otp).Error
	})
	if err != nil {
		s.Log.Error("contact email OTP verification failed", "user_id", user.ID, "error", err)
		fail(c, http.StatusUnprocessableEntity, "Invalid or expired verification code")
		return
	}
	if !valid {
		fail(c, http.StatusUnauthorized, "Invalid or expired verification code")
		return
	}

	previousEmail := s.currentContactEmail()
	if err := s.setPlatformSetting("contact_email_current", newEmail); err != nil {
		s.Log.Error("contact email update failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to update contact email")
		return
	}
	_ = os.Setenv("GMAIL_USERNAME", newEmail)
	s.deletePlatformSetting("contact_email_change_pending")

	s.audit(c, "contact_email_changed", "PlatformSetting", nil, map[string]any{
		"previous_email": previousEmail,
		"new_email":      newEmail,
	})
	body := "Your contact email has been changed from " + previousEmail + " to " + newEmail
	notification := models.Notification{
		UserID:           user.ID,
		Title:            "Contact email updated",
		Body:             &body,
		NotificationType: "administrative_update",
		IsRead:           false,
	}
	if err := s.DB.WithContext(ctx).Create(&notification).Error; err != nil {
		s.Log.Error("contact email notification failed", "user_id", user.ID, "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Contact email changed successfully to " + newEmail,
		"new_email": newEmail,
	})
}

func (s *Server) serializeFAQ(faq *models.FAQ, category *models.FAQCategory) gin.H {
	return gin.H{
		"id":       faq.ID,
		"question": faq.Question,
		"answer":   faq.Answer,
		"position": faq.Position,
		"category": gin.H{
			"id":   category.ID,
			"name": category.Name,
			"slug": category.Slug,
		},
	}
}

func (s *Server) serializeFAQCategory(category *models.FAQCategory) gin.H {
	var count int64
	s.DB.Model(&models.FAQ{}).Where("faq_category_id = ?", category.ID).Count(&count)
	return gin.H{
		"id":        category.ID,
		"name":      category.Name,
		"slug":      category.Slug,
		"position":  category.Position,
		"faq_count": count,
	}
}

func (s *Server) listFAQs(c *gin.Context) {
	var categories []models.FAQCategory
	if err := s.DB.WithContext(c.Request.Context()).Order("position ASC, id ASC").Find(&categories).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load FAQs")
		return
	}
	result := make([]gin.H, 0, len(categories))
	for i := range categories {
		var faqs []models.FAQ
		if err := s.DB.WithContext(c.Request.Context()).Where("faq_category_id = ?", categories[i].ID).
			Order("position ASC, id ASC").Find(&faqs).Error; err != nil {
			fail(c, http.StatusInternalServerError, "Unable to load FAQs")
			return
		}
		items := make([]gin.H, 0, len(faqs))
		for j := range faqs {
			items = append(items, gin.H{
				"id":       faqs[j].ID,
				"question": faqs[j].Question,
				"answer":   faqs[j].Answer,
				"position": faqs[j].Position,
			})
		}
		result = append(result, gin.H{
			"id":       categories[i].ID,
			"name":     categories[i].Name,
			"slug":     categories[i].Slug,
			"position": categories[i].Position,
			"faqs":     items,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "faqs": result})
}

func (s *Server) showFAQCategory(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	var category models.FAQCategory
	if err := s.DB.WithContext(c.Request.Context()).Where("slug = ?", slug).First(&category).Error; err != nil {
		fail(c, http.StatusNotFound, "FAQ category not found")
		return
	}
	var faqs []models.FAQ
	if err := s.DB.WithContext(c.Request.Context()).Where("faq_category_id = ?", category.ID).
		Order("position ASC, id ASC").Find(&faqs).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load FAQs")
		return
	}
	items := make([]gin.H, 0, len(faqs))
	for i := range faqs {
		items = append(items, gin.H{
			"id":       faqs[i].ID,
			"question": faqs[i].Question,
			"answer":   faqs[i].Answer,
			"position": faqs[i].Position,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"category": gin.H{
			"id":       category.ID,
			"name":     category.Name,
			"slug":     category.Slug,
			"position": category.Position,
			"faqs":     items,
		},
	})
}

func (s *Server) listFAQCategories(c *gin.Context) {
	var categories []models.FAQCategory
	if err := s.DB.WithContext(c.Request.Context()).Order("position ASC, id ASC").Find(&categories).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load FAQ categories")
		return
	}
	result := make([]gin.H, 0, len(categories))
	for i := range categories {
		result = append(result, s.serializeFAQCategory(&categories[i]))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "categories": result})
}

func (s *Server) createFAQCategory(c *gin.Context) {
	var request struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if !bindJSON(c, &request) {
		return
	}
	name := strings.TrimSpace(request.Name)
	slug := strings.ToLower(strings.TrimSpace(request.Slug))
	slug = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if name == "" {
		fail(c, http.StatusUnprocessableEntity, "Name is required")
		return
	}
	if slug == "" {
		slug = strings.ToLower(name)
		slug = regexp.MustCompile(`[^a-z0-9\s]+`).ReplaceAllString(slug, "")
		slug = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(slug), "-")
	}
	if slug == "" {
		fail(c, http.StatusUnprocessableEntity, "Unable to generate a category slug")
		return
	}
	var existing int64
	s.DB.WithContext(c.Request.Context()).Model(&models.FAQCategory{}).Where("slug = ?", slug).Count(&existing)
	if existing > 0 {
		fail(c, http.StatusConflict, "A category with this slug already exists")
		return
	}
	var maxPosition int
	s.DB.WithContext(c.Request.Context()).Model(&models.FAQCategory{}).Select("COALESCE(MAX(position), 0)").Scan(&maxPosition)
	position := maxPosition + 1
	category := models.FAQCategory{Name: name, Slug: slug, Position: position}
	if err := s.DB.WithContext(c.Request.Context()).Create(&category).Error; err != nil {
		s.Log.Error("FAQ category create failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to create FAQ category")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"message":  "FAQ category created",
		"category": s.serializeFAQCategory(&category),
	})
}

func (s *Server) createFAQ(c *gin.Context) {
	var request struct {
		CategoryID int64  `json:"category_id"`
		Question   string `json:"question"`
		Answer     string `json:"answer"`
	}
	if !bindJSON(c, &request) {
		return
	}
	question := strings.TrimSpace(request.Question)
	answer := strings.TrimSpace(request.Answer)
	if request.CategoryID <= 0 {
		fail(c, http.StatusNotFound, "FAQ category not found")
		return
	}
	if question == "" {
		fail(c, http.StatusUnprocessableEntity, "Question is required")
		return
	}
	if answer == "" {
		fail(c, http.StatusUnprocessableEntity, "Answer is required")
		return
	}
	var category models.FAQCategory
	if err := s.DB.WithContext(c.Request.Context()).First(&category, request.CategoryID).Error; err != nil {
		fail(c, http.StatusNotFound, "FAQ category not found")
		return
	}
	var maxPosition int
	s.DB.WithContext(c.Request.Context()).Model(&models.FAQ{}).Where("faq_category_id = ?", category.ID).
		Select("COALESCE(MAX(position), 0)").Scan(&maxPosition)
	position := maxPosition + 1
	faq := models.FAQ{
		FAQCategoryID: category.ID,
		Question:      question,
		Answer:        answer,
		Position:      position,
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&faq).Error; err != nil {
		s.Log.Error("FAQ create failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to create FAQ")
		return
	}
	s.audit(c, "faq_created", "Faq", ptrInt64(faq.ID), map[string]any{
		"question":      trimTo(question, 100),
		"category_name": category.Name,
	})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "FAQ created",
		"faq":     s.serializeFAQ(&faq, &category),
	})
}

func (s *Server) updateFAQ(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var faq models.FAQ
	if err := s.DB.WithContext(c.Request.Context()).First(&faq, id).Error; err != nil {
		fail(c, http.StatusNotFound, "FAQ not found")
		return
	}
	var request struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}
	if !bindJSON(c, &request) {
		return
	}
	changes := map[string]any{}
	if request.Question != "" && request.Question != faq.Question {
		question := strings.TrimSpace(request.Question)
		if question == "" {
			fail(c, http.StatusUnprocessableEntity, "Question cannot be empty")
			return
		}
		changes["question"] = map[string]string{"from": faq.Question, "to": question}
		faq.Question = question
	}
	if request.Answer != "" && request.Answer != faq.Answer {
		answer := strings.TrimSpace(request.Answer)
		if answer == "" {
			fail(c, http.StatusUnprocessableEntity, "Answer cannot be empty")
			return
		}
		changes["answer"] = map[string]string{"from": trimTo(faq.Answer, 100), "to": trimTo(answer, 100)}
		faq.Answer = answer
	}
	if len(changes) == 0 {
		fail(c, http.StatusUnprocessableEntity, "No changes provided")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Save(&faq).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to update FAQ")
		return
	}
	var category models.FAQCategory
	s.DB.WithContext(c.Request.Context()).First(&category, faq.FAQCategoryID)
	s.audit(c, "faq_updated", "Faq", ptrInt64(faq.ID), changes)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "FAQ updated",
		"faq":     s.serializeFAQ(&faq, &category),
	})
}

func (s *Server) deleteFAQ(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var faq models.FAQ
	if err := s.DB.WithContext(c.Request.Context()).First(&faq, id).Error; err != nil {
		fail(c, http.StatusNotFound, "FAQ not found")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Delete(&faq).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to delete FAQ")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "FAQ deleted"})
}

func (s *Server) reorderFAQs(c *gin.Context) {
	var request struct {
		CategoryID int64   `json:"category_id"`
		Positions  []int64 `json:"positions"`
	}
	if !bindJSON(c, &request) {
		return
	}
	if request.Positions == nil {
		fail(c, http.StatusUnprocessableEntity, "Positions must be an array")
		return
	}
	var category models.FAQCategory
	if err := s.DB.WithContext(c.Request.Context()).First(&category, request.CategoryID).Error; err != nil {
		fail(c, http.StatusNotFound, "Category not found")
		return
	}
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		for index, faqID := range request.Positions {
			var faq models.FAQ
			if err := tx.First(&faq, faqID).Error; err != nil {
				return err
			}
			if err := tx.Model(&faq).Updates(map[string]any{
				"faq_category_id": category.ID,
				"position":        index + 1,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "Record not found")
			return
		}
		fail(c, http.StatusInternalServerError, "Unable to reorder FAQs")
		return
	}
	var count int64
	s.DB.WithContext(c.Request.Context()).Model(&models.FAQ{}).Where("faq_category_id = ?", category.ID).Count(&count)
	s.audit(c, "faqs_reordered", "FaqCategory", ptrInt64(category.ID), map[string]any{"faq_count": count})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "FAQs reordered"})
}
