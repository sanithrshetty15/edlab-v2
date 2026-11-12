package api

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

func (s *Server) notifyAudience(title, body, notificationType, subjectType string, subjectID *int64) {
	var users []models.User
	if err := s.DB.Where("users.active = ?", true).
		Where("users.role = ? OR (users.role = ? AND EXISTS (SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = ?))", "admin", "member", "active").
		Find(&users).Error; err != nil {
		s.Log.Error("audience notification lookup failed", "error", err)
		return
	}
	for i := range users {
		s.notifyUser(nil, users[i].ID, title, body, notificationType, subjectType, subjectID)
	}
}

func (s *Server) registerNotificationRoutes(api *gin.RouterGroup) {
	notifications := api.Group("/notifications")
	notifications.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted())
	notifications.GET("", s.listNotifications)
	notifications.GET("/unread-count", s.notificationUnreadCount)
	notifications.GET("/history", s.notificationHistory)
	notifications.POST("/read", s.markNotificationsRead)
	notifications.POST("/read-all", s.markAllNotificationsRead)
	notifications.GET("/:id", s.showNotification)
	notifications.DELETE("/:id", s.deleteNotification)
}

func (s *Server) serializeNotification(notification *models.Notification) gin.H {
	return gin.H{
		"id":                notification.ID,
		"title":             notification.Title,
		"body":              notification.Body,
		"notification_type": notification.NotificationType,
		"subject_type":      notification.SubjectType,
		"subject_id":        notification.SubjectID,
		"is_read":           notification.IsRead,
		"read_at":           notification.ReadAt,
		"created_at":        notification.CreatedAt,
	}
}

func (s *Server) listNotifications(c *gin.Context) {
	user := currentUser(c)
	if user == nil {
		fail(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	query := s.DB.WithContext(c.Request.Context()).Model(&models.Notification{}).
		Where("user_id = ?", user.ID)
	if notificationType := strings.TrimSpace(c.Query("type")); notificationType != "" {
		query = query.Where("notification_type = ?", notificationType)
	}
	if read := strings.TrimSpace(c.Query("read")); read != "" {
		query = query.Where("is_read = ?", read == "true")
	}
	if before := strings.TrimSpace(c.Query("before")); before != "" {
		if parsed, ok := parseDate(before); ok {
			query = query.Where("created_at < ?", parsed)
		}
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.Log.Error("notification count failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load notifications")
		return
	}
	var notifications []models.Notification
	if err := query.Order("created_at DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&notifications).Error; err != nil {
		s.Log.Error("notification list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load notifications")
		return
	}
	var unread int64
	s.DB.WithContext(c.Request.Context()).Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", user.ID, false).Count(&unread)
	items := make([]gin.H, 0, len(notifications))
	for i := range notifications {
		items = append(items, s.serializeNotification(&notifications[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"notifications": items,
		"pagination": gin.H{
			"page":         page,
			"per_page":     perPage,
			"total_count":  total,
			"total_pages":  pageCount(total, perPage),
			"unread_count": unread,
		},
	})
}

func pageCount(total int64, perPage int) int {
	if perPage <= 0 || total == 0 {
		return 0
	}
	return int((total + int64(perPage) - 1) / int64(perPage))
}

func (s *Server) notificationUnreadCount(c *gin.Context) {
	user := currentUser(c)
	var count int64
	s.DB.WithContext(c.Request.Context()).Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", user.ID, false).Count(&count)
	c.JSON(http.StatusOK, gin.H{"success": true, "unread_count": count})
}

func (s *Server) notificationHistory(c *gin.Context) {
	user := currentUser(c)
	query := s.DB.WithContext(c.Request.Context()).Model(&models.Notification{}).
		Where("user_id = ?", user.ID)
	from := startOfDay(time.Now().AddDate(0, 0, -30))
	to := startOfDay(time.Now()).AddDate(0, 0, 1)
	if value := strings.TrimSpace(c.Query("date_from")); value != "" {
		if parsed, ok := parseDate(value); ok {
			from = startOfDay(parsed)
		}
	}
	if value := strings.TrimSpace(c.Query("date_to")); value != "" {
		if parsed, ok := parseDate(value); ok {
			to = startOfDay(parsed).AddDate(0, 0, 1)
		}
	}
	query = query.Where("created_at >= ? AND created_at < ?", from, to)
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load notification history")
		return
	}
	var notifications []models.Notification
	if err := query.Order("created_at DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&notifications).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load notification history")
		return
	}
	grouped := map[string][]gin.H{}
	for i := range notifications {
		key := notifications[i].CreatedAt.In(appLocation()).Format(dateLayout)
		grouped[key] = append(grouped[key], s.serializeNotification(&notifications[i]))
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	history := make([]gin.H, 0, len(keys))
	for _, key := range keys {
		history = append(history, gin.H{
			"date":          key,
			"count":         len(grouped[key]),
			"notifications": grouped[key],
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"history": history,
		"pagination": gin.H{
			"page":        page,
			"total_count": total,
		},
	})
}

func (s *Server) showNotification(c *gin.Context) {
	user := currentUser(c)
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var notification models.Notification
	if err := s.DB.WithContext(c.Request.Context()).
		Where("id = ? AND user_id = ?", id, user.ID).First(&notification).Error; err != nil {
		fail(c, http.StatusNotFound, "Notification not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "notification": s.serializeNotification(&notification)})
}

func (s *Server) markNotificationsRead(c *gin.Context) {
	user := currentUser(c)
	var request struct {
		IDs []int64 `json:"ids"`
	}
	if !bindJSON(c, &request) {
		return
	}
	now := time.Now()
	query := s.DB.WithContext(c.Request.Context()).Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", user.ID, false)
	if len(request.IDs) > 0 {
		query = query.Where("id IN ?", request.IDs)
	}
	if err := query.Updates(map[string]any{"is_read": true, "read_at": now}).Error; err != nil {
		s.Log.Error("notification mark read failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to mark notifications as read")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Notifications marked as read"})
}

func (s *Server) markAllNotificationsRead(c *gin.Context) {
	user := currentUser(c)
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", user.ID, false).
		Updates(map[string]any{"is_read": true, "read_at": time.Now()}).Error; err != nil {
		s.Log.Error("notification mark all read failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to mark notifications as read")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "All notifications marked as read"})
}

func (s *Server) deleteNotification(c *gin.Context) {
	user := currentUser(c)
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	result := s.DB.WithContext(c.Request.Context()).
		Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Notification{})
	if result.Error != nil {
		fail(c, http.StatusInternalServerError, "Unable to delete notification")
		return
	}
	if result.RowsAffected == 0 {
		fail(c, http.StatusNotFound, "Notification not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Notification deleted"})
}

func (s *Server) registerAnnouncementRoutes(api *gin.RouterGroup) {
	announcements := api.Group("/announcements")
	announcements.Use(s.RequireAuth())
	announcements.GET("", s.listAnnouncements)
	announcements.GET("/:id", s.showAnnouncement)
	announcements.GET("/:id/read-status", s.announcementReadStatus)
	announcements.POST("/:id/read", s.markAnnouncementRead)
	announcements.POST("", s.RequireAdmin(), s.createAnnouncement)
	announcements.PATCH("/:id", s.RequireAdmin(), s.updateAnnouncement)
	announcements.POST("/:id/publish", s.RequireAdmin(), s.publishAnnouncement)
	announcements.POST("/:id/unpublish", s.RequireAdmin(), s.unpublishAnnouncement)
	announcements.POST("/:id/toggle-pin", s.RequireAdmin(), s.toggleAnnouncementPin)
	announcements.DELETE("/:id", s.RequireAdmin(), s.deleteAnnouncement)
}

func (s *Server) announcementCreatorName(userID int64) gin.H {
	return s.taskCreatorName(userID)
}

func (s *Server) announcementReadCount(announcementID int64) int64 {
	var count int64
	s.DB.Model(&models.AnnouncementRead{}).Where("announcement_id = ?", announcementID).Count(&count)
	return count
}

func (s *Server) serializeAnnouncement(announcement *models.Announcement) gin.H {
	return gin.H{
		"id":           announcement.ID,
		"title":        announcement.Title,
		"is_pinned":    announcement.IsPinned,
		"is_published": announcement.PublishedAt != nil,
		"published_at": announcement.PublishedAt,
		"scheduled_at": announcement.ScheduledAt,
		"created_by":   s.announcementCreatorName(announcement.CreatedByID),
		"created_at":   announcement.CreatedAt,
		"updated_at":   announcement.UpdatedAt,
	}
}

func (s *Server) serializeAnnouncementFull(announcement *models.Announcement) gin.H {
	result := s.serializeAnnouncement(announcement)
	result["content"] = announcement.Content
	result["read_by_count"] = s.announcementReadCount(announcement.ID)
	return result
}

func (s *Server) listAnnouncements(c *gin.Context) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.Announcement{}).
		Where("announcements.deleted_at IS NULL")
	if published := strings.TrimSpace(c.Query("published")); published == "true" {
		query = query.Where("announcements.published_at IS NOT NULL")
	}
	if important := strings.TrimSpace(c.Query("important")); important == "true" {
		query = query.Where("announcements.is_pinned = ?", true)
	}
	if before := strings.TrimSpace(c.Query("before")); before != "" {
		if parsed, ok := parseDate(before); ok {
			query = query.Where("(announcements.published_at < ? OR (announcements.published_at IS NULL AND announcements.created_at < ?))", parsed, parsed)
		}
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load announcements")
		return
	}
	var announcements []models.Announcement
	if err := query.Order("announcements.published_at DESC NULLS LAST, announcements.created_at DESC").
		Offset(offsetFor(page, perPage)).Limit(perPage).Find(&announcements).Error; err != nil {
		s.Log.Error("announcement list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load announcements")
		return
	}
	items := make([]gin.H, 0, len(announcements))
	for i := range announcements {
		items = append(items, s.serializeAnnouncement(&announcements[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"announcements": items,
		"pagination":    makePagination(page, perPage, total),
	})
}

func (s *Server) findAnnouncement(c *gin.Context) (*models.Announcement, bool) {
	id, ok := pathID(c, "id")
	if !ok {
		return nil, false
	}
	var announcement models.Announcement
	if err := s.DB.WithContext(c.Request.Context()).Where("id = ? AND deleted_at IS NULL", id).
		First(&announcement).Error; err != nil {
		fail(c, http.StatusNotFound, "Announcement not found")
		return nil, false
	}
	return &announcement, true
}

func (s *Server) showAnnouncement(c *gin.Context) {
	announcement, ok := s.findAnnouncement(c)
	if !ok {
		return
	}
	user := currentUser(c)
	if announcement.PublishedAt == nil && (user == nil || user.Role != "admin") {
		fail(c, http.StatusForbidden, "Announcement is not yet published")
		return
	}
	var read models.AnnouncementRead
	readAt := any(nil)
	hasRead := s.DB.WithContext(c.Request.Context()).
		Where("announcement_id = ? AND user_id = ?", announcement.ID, user.ID).First(&read).Error == nil
	if hasRead {
		readAt = read.ReadAt
	}
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"announcement": s.serializeAnnouncementFull(announcement),
		"read":         hasRead,
		"read_at":      readAt,
	})
}

func (s *Server) markAnnouncementRead(c *gin.Context) {
	announcement, ok := s.findAnnouncement(c)
	if !ok {
		return
	}
	user := currentUser(c)
	var read models.AnnouncementRead
	err := s.DB.WithContext(c.Request.Context()).
		Where("announcement_id = ? AND user_id = ?", announcement.ID, user.ID).First(&read).Error
	if err == gorm.ErrRecordNotFound {
		read = models.AnnouncementRead{
			AnnouncementID: announcement.ID,
			UserID:         user.ID,
			ReadAt:         time.Now(),
		}
		if err := s.DB.WithContext(c.Request.Context()).Create(&read).Error; err != nil {
			fail(c, http.StatusInternalServerError, "Unable to mark announcement as read")
			return
		}
	} else if err != nil {
		fail(c, http.StatusInternalServerError, "Unable to mark announcement as read")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Announcement marked as read",
		"read_at":       read.ReadAt,
		"read_by_count": s.announcementReadCount(announcement.ID),
	})
}

func (s *Server) announcementAudienceCount() (total, unread int64) {
	var users []models.User
	s.DB.Model(&models.User{}).Where("users.active = ?", true).
		Where("users.role = ? OR (users.role = ? AND EXISTS (SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = ?))", "admin", "member", "active").
		Find(&users)
	total = int64(len(users))
	if total == 0 {
		return 0, 0
	}
	ids := make([]int64, 0, len(users))
	for i := range users {
		ids = append(ids, users[i].ID)
	}
	var readCount int64
	s.DB.Model(&models.AnnouncementRead{}).Where("user_id IN ?", ids).Distinct("user_id").Count(&readCount)
	return total, total - readCount
}

func (s *Server) announcementReadStatus(c *gin.Context) {
	announcement, ok := s.findAnnouncement(c)
	if !ok {
		return
	}
	total, unread := s.announcementAudienceCount()
	readPercentage := float64(0)
	if total > 0 {
		readPercentage = float64(total-unread) / float64(total) * 100
		readPercentage = float64(int(readPercentage*100+0.5)) / 100
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"read_status": gin.H{
			"announcement_id": announcement.ID,
			"total_members":   total,
			"unread_members":  unread,
			"read_percentage": readPercentage,
		},
	})
}

type announcementCreateRequest struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	IsImportant bool   `json:"is_important"`
	ScheduledAt string `json:"scheduled_at"`
}

func (s *Server) createAnnouncement(c *gin.Context) {
	var request announcementCreateRequest
	if !bindJSON(c, &request) {
		return
	}
	title := strings.TrimSpace(request.Title)
	content := strings.TrimSpace(request.Content)
	if title == "" {
		fail(c, http.StatusUnprocessableEntity, "Title is required")
		return
	}
	if content == "" {
		fail(c, http.StatusUnprocessableEntity, "Content is required")
		return
	}
	now := time.Now()
	announcement := models.Announcement{
		Title:       title,
		Content:     content,
		IsPinned:    request.IsImportant,
		CreatedByID: currentUser(c).ID,
	}
	message := "Announcement created and published"
	if scheduled := strings.TrimSpace(request.ScheduledAt); scheduled != "" {
		if parsed, ok := parseDate(scheduled); ok && parsed.After(now) {
			announcement.ScheduledAt = &parsed
			message = "Announcement created (scheduled for later publication)"
		} else {
			announcement.PublishedAt = &now
		}
	} else {
		announcement.PublishedAt = &now
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&announcement).Error; err != nil {
		s.Log.Error("announcement create failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to create announcement")
		return
	}
	s.DB.Create(&models.AnnouncementRead{
		AnnouncementID: announcement.ID,
		UserID:         currentUser(c).ID,
		ReadAt:         now,
	})
	s.audit(c, "announcement_created", "Announcement", &announcement.ID, gin.H{"title": announcement.Title})
	if announcement.PublishedAt != nil {
		s.notifyAudience("New announcement", announcement.Title, "announcement", "Announcement", &announcement.ID)
	}
	c.JSON(http.StatusCreated, gin.H{
		"success":      true,
		"message":      message,
		"announcement": s.serializeAnnouncementFull(&announcement),
	})
}

func (s *Server) updateAnnouncement(c *gin.Context) {
	announcement, ok := s.findAnnouncement(c)
	if !ok {
		return
	}
	var request struct {
		Title       *string `json:"title"`
		Content     *string `json:"content"`
		IsImportant *bool   `json:"is_important"`
	}
	if !bindJSON(c, &request) {
		return
	}
	changes := gin.H{}
	if request.Title != nil {
		title := strings.TrimSpace(*request.Title)
		if title == "" {
			fail(c, http.StatusUnprocessableEntity, "Title cannot be empty")
			return
		}
		if announcement.Title != title {
			changes["title"] = gin.H{"from": announcement.Title, "to": title}
			announcement.Title = title
		}
	}
	if request.Content != nil {
		content := strings.TrimSpace(*request.Content)
		if content == "" {
			fail(c, http.StatusUnprocessableEntity, "Content cannot be empty")
			return
		}
		if announcement.Content != content {
			changes["content"] = gin.H{"from": trimTo(announcement.Content, 100), "to": trimTo(content, 100)}
			announcement.Content = content
		}
	}
	if request.IsImportant != nil && announcement.IsPinned != *request.IsImportant {
		changes["is_important"] = gin.H{"from": announcement.IsPinned, "to": *request.IsImportant}
		announcement.IsPinned = *request.IsImportant
	}
	if len(changes) == 0 {
		fail(c, http.StatusUnprocessableEntity, "No changes provided")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Save(announcement).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to update announcement")
		return
	}
	s.audit(c, "announcement_updated", "Announcement", &announcement.ID, changes)
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Announcement updated",
		"announcement": s.serializeAnnouncementFull(announcement),
	})
}

func trimTo(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func (s *Server) publishAnnouncement(c *gin.Context) {
	announcement, ok := s.findAnnouncement(c)
	if !ok {
		return
	}
	var request struct {
		PublishAt string `json:"publish_at"`
	}
	_ = c.ShouldBindJSON(&request)
	now := time.Now()
	if announcement.PublishedAt != nil {
		if strings.TrimSpace(request.PublishAt) == "" {
			fail(c, http.StatusConflict, "Announcement is already published")
			return
		}
		publishAt, parsed := parseDate(request.PublishAt)
		if !parsed {
			fail(c, http.StatusUnprocessableEntity, "Invalid publish date")
			return
		}
		announcement.PublishedAt = &publishAt
	} else {
		announcement.PublishedAt = &now
	}
	announcement.ScheduledAt = nil
	if err := s.DB.WithContext(c.Request.Context()).Save(announcement).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to publish announcement")
		return
	}
	var creatorRead models.AnnouncementRead
	if err := s.DB.Where("announcement_id = ? AND user_id = ?", announcement.ID, currentUser(c).ID).
		First(&creatorRead).Error; err == gorm.ErrRecordNotFound {
		s.DB.Create(&models.AnnouncementRead{
			AnnouncementID: announcement.ID,
			UserID:         currentUser(c).ID,
			ReadAt:         now,
		})
	}
	s.audit(c, "announcement_published", "Announcement", &announcement.ID, gin.H{
		"title":        announcement.Title,
		"published_at": announcement.PublishedAt,
	})
	s.notifyAudience("New announcement", announcement.Title, "announcement", "Announcement", &announcement.ID)
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Announcement published",
		"announcement": s.serializeAnnouncementFull(announcement),
	})
}

func (s *Server) unpublishAnnouncement(c *gin.Context) {
	announcement, ok := s.findAnnouncement(c)
	if !ok {
		return
	}
	if announcement.PublishedAt == nil {
		fail(c, http.StatusUnprocessableEntity, "Announcement is not published")
		return
	}
	announcement.PublishedAt = nil
	if err := s.DB.WithContext(c.Request.Context()).Save(announcement).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to unpublish announcement")
		return
	}
	s.audit(c, "announcement_unpublished", "Announcement", &announcement.ID, gin.H{"title": announcement.Title})
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Announcement unpublished",
		"announcement": s.serializeAnnouncementFull(announcement),
	})
}

func (s *Server) toggleAnnouncementPin(c *gin.Context) {
	announcement, ok := s.findAnnouncement(c)
	if !ok {
		return
	}
	announcement.IsPinned = !announcement.IsPinned
	if err := s.DB.WithContext(c.Request.Context()).Save(announcement).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to update announcement")
		return
	}
	state := "unpinned"
	if announcement.IsPinned {
		state = "pinned"
	}
	s.audit(c, "announcement_"+state, "Announcement", &announcement.ID, gin.H{
		"title": announcement.Title, "is_pinned": announcement.IsPinned,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Announcement " + state,
		"announcement": s.serializeAnnouncementFull(announcement),
	})
}

func (s *Server) deleteAnnouncement(c *gin.Context) {
	announcement, ok := s.findAnnouncement(c)
	if !ok {
		return
	}
	now := time.Now()
	if err := s.DB.WithContext(c.Request.Context()).Model(announcement).Update("deleted_at", now).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to delete announcement")
		return
	}
	s.audit(c, "announcement_deleted", "Announcement", &announcement.ID, gin.H{"title": announcement.Title})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Announcement deleted"})
}

func (s *Server) registerMessageRoutes(api *gin.RouterGroup) {
	messages := api.Group("/messages")
	messages.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted())
	messages.GET("", s.listMessages)
	messages.GET("/pinned", s.pinnedMessages)
	messages.GET("/search", s.searchMessages)
	messages.GET("/:id", s.showMessage)
	messages.POST("", s.createMessage)
	messages.PATCH("/:id", s.editMessage)
	messages.DELETE("/:id", s.deleteMessage)
	messages.POST("/:id/reply", s.replyToMessage)
	messages.GET("/:id/replies", s.messageReplies)
	messages.POST("/:id/forward", s.forwardMessage)
	messages.POST("/:id/react", s.reactToMessage)
	messages.DELETE("/:id/react", s.removeMessageReaction)
	messages.GET("/:id/reactions", s.messageReactions)
	messages.POST("/:id/pin", s.RequireAdmin(), s.pinMessage)
	messages.DELETE("/:id/pin", s.RequireAdmin(), s.unpinMessage)
	messages.GET("/:id/receipts", s.messageReceipts)
	messages.POST("/:id/star", s.starMessage)
	messages.DELETE("/:id/star", s.unstarMessage)
	messages.GET("/:id/starred", s.messageStars)
}

type messageAttachmentRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	ByteSize    int    `json:"byte_size"`
	StorageKey  string `json:"storage_key"`
}

func (s *Server) messageUserName(userID int64) string {
	var member models.Member
	if err := s.DB.Where("user_id = ?", userID).First(&member).Error; err == nil && strings.TrimSpace(member.Name) != "" {
		return member.Name
	}
	var admin models.Admin
	if err := s.DB.Where("user_id = ?", userID).First(&admin).Error; err == nil && strings.TrimSpace(admin.Name) != "" {
		return admin.Name
	}
	var user models.User
	if err := s.DB.First(&user, userID).Error; err == nil {
		return user.Email
	}
	return ""
}

func (s *Server) serializeMessage(message *models.GroupMessage) gin.H {
	return gin.H{
		"id":                message.ID,
		"user_id":           message.UserID,
		"user":              gin.H{"id": message.UserID, "name": s.messageUserName(message.UserID)},
		"content":           message.Content,
		"message_type":      message.MessageType,
		"reply_to_id":       message.ReplyToID,
		"forwarded_from_id": message.ForwardedFromID,
		"is_edited":         message.IsEdited,
		"is_deleted":        message.IsDeleted,
		"is_pinned":         message.IsPinned,
		"created_at":        message.CreatedAt,
		"updated_at":        message.UpdatedAt,
	}
}

func (s *Server) serializeMessageFull(message *models.GroupMessage) gin.H {
	result := s.serializeMessage(message)
	var attachments []models.MessageAttachment
	s.DB.Where("group_message_id = ?", message.ID).Order("id ASC").Find(&attachments)
	items := make([]gin.H, 0, len(attachments))
	for i := range attachments {
		items = append(items, gin.H{
			"id":           attachments[i].ID,
			"filename":     attachments[i].Filename,
			"content_type": attachments[i].ContentType,
			"byte_size":    attachments[i].ByteSize,
			"storage_key":  attachments[i].StorageKey,
			"url":          nil,
		})
	}
	var linkPreview *models.LinkPreview
	var preview any = nil
	if err := s.DB.Where("group_message_id = ?", message.ID).First(&linkPreview).Error; err == nil {
		preview = gin.H{
			"url":         linkPreview.URL,
			"title":       linkPreview.Title,
			"description": linkPreview.Description,
			"image_url":   linkPreview.ImageURL,
			"site_name":   linkPreview.SiteName,
		}
	}
	var reactions, replies, stars, receipts int64
	s.DB.Model(&models.MessageReaction{}).Where("group_message_id = ?", message.ID).Count(&reactions)
	s.DB.Model(&models.GroupMessage{}).Where("reply_to_id = ? AND is_deleted = ?", message.ID, false).Count(&replies)
	s.DB.Model(&models.MessageStar{}).Where("group_message_id = ?", message.ID).Count(&stars)
	s.DB.Model(&models.MessageReceipt{}).Where("group_message_id = ?", message.ID).Count(&receipts)
	result["attachments"] = items
	result["link_preview"] = preview
	result["reactions_count"] = reactions
	result["replies_count"] = replies
	result["stars_count"] = stars
	result["receipts_count"] = receipts
	return result
}

func (s *Server) findMessage(c *gin.Context) (*models.GroupMessage, bool) {
	id, ok := pathID(c, "id")
	if !ok {
		return nil, false
	}
	var message models.GroupMessage
	if err := s.DB.WithContext(c.Request.Context()).First(&message, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Message not found")
		return nil, false
	}
	return &message, true
}

func (s *Server) messagingAllowed(c *gin.Context) bool {
	user := currentUser(c)
	if user != nil && user.Role == "admin" {
		return true
	}
	var settings models.GroupSetting
	if err := s.DB.Order("id ASC").First(&settings).Error; err == nil && settings.Permission == "admins_only" {
		fail(c, http.StatusForbidden, "Messaging is currently disabled for members")
		return false
	}
	return true
}

func (s *Server) listMessages(c *gin.Context) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.GroupMessage{}).Where("is_deleted = ?", false)
	if before := strings.TrimSpace(c.Query("before")); before != "" {
		if parsed, ok := parseDate(before); ok {
			query = query.Where("created_at < ?", parsed)
		}
	}
	if after := strings.TrimSpace(c.Query("after")); after != "" {
		if parsed, ok := parseDate(after); ok {
			query = query.Where("created_at > ?", parsed)
		}
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load messages")
		return
	}
	var messages []models.GroupMessage
	if err := query.Order("created_at DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&messages).Error; err != nil {
		s.Log.Error("message list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load messages")
		return
	}
	items := make([]gin.H, 0, len(messages))
	for i := range messages {
		items = append(items, s.serializeMessage(&messages[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"messages":   items,
		"pagination": makePagination(page, perPage, total),
	})
}

func (s *Server) showMessage(c *gin.Context) {
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": s.serializeMessageFull(message)})
}

func validateMessageAttachments(attachments []messageAttachmentRequest) (string, bool) {
	allowed := map[string]bool{
		"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
		"application/pdf": true, "text/plain": true, "application/zip": true,
	}
	if len(attachments) > 5 {
		return "Maximum 5 attachments allowed", false
	}
	for _, attachment := range attachments {
		if strings.TrimSpace(attachment.Filename) == "" || strings.TrimSpace(attachment.StorageKey) == "" {
			return "Attachment filename and storage key are required", false
		}
		if attachment.ByteSize > 10*1024*1024 {
			return "Attachment too large (max 10MB)", false
		}
		if !allowed[attachment.ContentType] {
			return "File type not allowed: " + attachment.ContentType, false
		}
	}
	return "", true
}

func (s *Server) insertMessageReceipts(tx *gorm.DB, messageID int64) error {
	var users []models.User
	if err := tx.Where("users.active = ?", true).
		Where("users.role = ? OR (users.role = ? AND EXISTS (SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = ?))", "admin", "member", "active").
		Find(&users).Error; err != nil {
		return err
	}
	now := time.Now()
	receipts := make([]models.MessageReceipt, 0, len(users))
	for i := range users {
		receipts = append(receipts, models.MessageReceipt{
			GroupMessageID: messageID,
			UserID:         users[i].ID,
			DeliveredAt:    &now,
		})
	}
	if len(receipts) == 0 {
		return nil
	}
	return tx.Create(&receipts).Error
}

func (s *Server) createMessage(c *gin.Context) {
	if !s.messagingAllowed(c) {
		return
	}
	var request struct {
		Content     string                     `json:"content"`
		Attachments []messageAttachmentRequest `json:"attachments"`
	}
	if !bindJSON(c, &request) {
		return
	}
	content := strings.TrimSpace(request.Content)
	if content == "" && len(request.Attachments) == 0 {
		fail(c, http.StatusUnprocessableEntity, "Message content is required")
		return
	}
	if len([]rune(content)) > 10000 {
		fail(c, http.StatusUnprocessableEntity, "Message content is too long (max 10000 characters)")
		return
	}
	if message, ok := validateMessageAttachments(request.Attachments); !ok {
		fail(c, http.StatusUnprocessableEntity, message)
		return
	}
	messageType := "text"
	if len(request.Attachments) > 0 {
		messageType = "media"
	} else if strings.HasPrefix(strings.ToLower(content), "http://") || strings.HasPrefix(strings.ToLower(content), "https://") {
		messageType = "link"
	}
	user := currentUser(c)
	message := models.GroupMessage{
		UserID:      user.ID,
		Content:     nullableString(content),
		MessageType: messageType,
	}
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&message).Error; err != nil {
			return err
		}
		for _, attachment := range request.Attachments {
			if err := tx.Create(&models.MessageAttachment{
				GroupMessageID: message.ID,
				Filename:       strings.TrimSpace(attachment.Filename),
				ContentType:    attachment.ContentType,
				ByteSize:       attachment.ByteSize,
				StorageKey:     attachment.StorageKey,
			}).Error; err != nil {
				return err
			}
		}
		return s.insertMessageReceipts(tx, message.ID)
	})
	if err != nil {
		s.Log.Error("message create failed", "error", err)
		fail(c, http.StatusInternalServerError, "Failed to create message")
		return
	}
	s.audit(c, "message_sent", "GroupMessage", &message.ID, gin.H{"preview": trimTo(content, 100)})
	s.notifyOtherActiveUsers(user.ID, "New message", s.messageUserName(user.ID)+" posted a new message", "message", &message.ID)
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": s.serializeMessageFull(&message)})
}

func (s *Server) notifyMessageParticipants(senderID, userID int64, title, body, notificationType string) {
}

func (s *Server) replyToMessage(c *gin.Context) {
	if !s.messagingAllowed(c) {
		return
	}
	parent, ok := s.findMessage(c)
	if !ok {
		return
	}
	if parent.IsDeleted {
		fail(c, http.StatusGone, "Cannot reply to a deleted message")
		return
	}
	var request struct {
		Content string `json:"content"`
	}
	if !bindJSON(c, &request) {
		return
	}
	content := strings.TrimSpace(request.Content)
	if content == "" {
		fail(c, http.StatusUnprocessableEntity, "Reply content cannot be empty")
		return
	}
	if len([]rune(content)) > 10000 {
		fail(c, http.StatusUnprocessableEntity, "Reply content is too long")
		return
	}
	user := currentUser(c)
	message := models.GroupMessage{
		UserID:      user.ID,
		Content:     &content,
		MessageType: "text",
		ReplyToID:   &parent.ID,
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&message).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Failed to create reply")
		return
	}
	s.notifyOtherActiveUsers(user.ID, "New reply", s.messageUserName(user.ID)+" replied to a message", "message_reply", &message.ID)
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": s.serializeMessageFull(&message)})
}

func (s *Server) messageReplies(c *gin.Context) {
	parent, ok := s.findMessage(c)
	if !ok {
		return
	}
	var replies []models.GroupMessage
	if err := s.DB.WithContext(c.Request.Context()).
		Where("reply_to_id = ? AND is_deleted = ?", parent.ID, false).
		Order("created_at ASC").Find(&replies).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load replies")
		return
	}
	items := make([]gin.H, 0, len(replies))
	for i := range replies {
		items = append(items, s.serializeMessage(&replies[i]))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "replies": items})
}

func (s *Server) editMessage(c *gin.Context) {
	if !s.messagingAllowed(c) {
		return
	}
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	user := currentUser(c)
	if message.UserID != user.ID {
		fail(c, http.StatusForbidden, "You can only edit your own messages")
		return
	}
	if message.IsDeleted {
		fail(c, http.StatusGone, "Message has been deleted")
		return
	}
	var request struct {
		Content string `json:"content"`
	}
	if !bindJSON(c, &request) {
		return
	}
	content := strings.TrimSpace(request.Content)
	if content == "" {
		fail(c, http.StatusUnprocessableEntity, "Message content cannot be empty")
		return
	}
	previous := stringOrEmpty(message.Content)
	if err := s.DB.WithContext(c.Request.Context()).Model(message).Updates(map[string]any{
		"content":   content,
		"is_edited": true,
	}).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to update message")
		return
	}
	message.Content = &content
	message.IsEdited = true
	s.audit(c, "message_edited", "GroupMessage", &message.ID, gin.H{"previous_content_preview": trimTo(previous, 100)})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": s.serializeMessageFull(message)})
}

func (s *Server) deleteMessage(c *gin.Context) {
	if !s.messagingAllowed(c) {
		return
	}
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	user := currentUser(c)
	if message.UserID != user.ID && user.Role != "admin" {
		fail(c, http.StatusForbidden, "You can only delete your own messages")
		return
	}
	wasDeleted := message.IsDeleted
	if err := s.DB.WithContext(c.Request.Context()).Model(message).Update("is_deleted", true).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to delete message")
		return
	}
	if !wasDeleted {
		s.audit(c, "message_deleted", "GroupMessage", &message.ID, gin.H{"content_preview": trimTo(stringOrEmpty(message.Content), 100)})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Message deleted successfully"})
}

func (s *Server) forwardMessage(c *gin.Context) {
	if !s.messagingAllowed(c) {
		return
	}
	var request struct {
		ForwardedToID int64 `json:"forwarded_to_id"`
	}
	if !bindJSON(c, &request) {
		return
	}
	if request.ForwardedToID <= 0 {
		fail(c, http.StatusUnprocessableEntity, "Forwarded message ID is required")
		return
	}
	var original models.GroupMessage
	if err := s.DB.WithContext(c.Request.Context()).First(&original, request.ForwardedToID).Error; err != nil {
		fail(c, http.StatusNotFound, "Original message not found")
		return
	}
	if original.IsDeleted {
		fail(c, http.StatusGone, "Cannot forward a deleted message")
		return
	}
	user := currentUser(c)
	forwarded := models.GroupMessage{
		UserID:          user.ID,
		Content:         original.Content,
		MessageType:     original.MessageType,
		ForwardedFromID: &original.ID,
	}
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&forwarded).Error; err != nil {
			return err
		}
		var attachments []models.MessageAttachment
		tx.Where("group_message_id = ?", original.ID).Find(&attachments)
		for i := range attachments {
			if err := tx.Create(&models.MessageAttachment{
				GroupMessageID: forwarded.ID,
				Filename:       attachments[i].Filename,
				ContentType:    attachments[i].ContentType,
				ByteSize:       attachments[i].ByteSize,
				StorageKey:     attachments[i].StorageKey,
			}).Error; err != nil {
				return err
			}
		}
		var preview models.LinkPreview
		if err := tx.Where("group_message_id = ?", original.ID).First(&preview).Error; err == nil {
			if err := tx.Create(&models.LinkPreview{
				GroupMessageID: forwarded.ID,
				URL:            preview.URL,
				Title:          preview.Title,
				Description:    preview.Description,
				ImageURL:       preview.ImageURL,
				SiteName:       preview.SiteName,
			}).Error; err != nil {
				return err
			}
		}
		return s.insertMessageReceipts(tx, forwarded.ID)
	})
	if err != nil {
		s.Log.Error("message forward failed", "error", err)
		fail(c, http.StatusInternalServerError, "Failed to forward message")
		return
	}
	s.notifyOtherActiveUsers(user.ID, "Message forwarded", s.messageUserName(user.ID)+" forwarded a message", "message_forward", &forwarded.ID)
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": s.serializeMessageFull(&forwarded)})
}

func (s *Server) notifyOtherActiveUsers(senderID int64, title, body, notificationType string, subjectID *int64) {
	var users []models.User
	if err := s.DB.Where("users.active = ? AND users.id <> ?", true, senderID).
		Where("users.role = ? OR (users.role = ? AND EXISTS (SELECT 1 FROM members WHERE members.user_id = users.id AND members.deleted_at IS NULL AND members.status = ?))", "admin", "member", "active").
		Find(&users).Error; err != nil {
		s.Log.Error("message notification lookup failed", "error", err)
		return
	}
	for i := range users {
		s.notifyUser(nil, users[i].ID, title, body, notificationType, "GroupMessage", subjectID)
	}
}

func (s *Server) reactToMessage(c *gin.Context) {
	if !s.messagingAllowed(c) {
		return
	}
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	var request struct {
		Emoji string `json:"emoji"`
	}
	if !bindJSON(c, &request) {
		return
	}
	emoji := strings.TrimSpace(request.Emoji)
	if emoji == "" {
		fail(c, http.StatusUnprocessableEntity, "Emoji is required")
		return
	}
	if strings.ContainsAny(emoji, " \t\r\n") || len([]rune(emoji)) > 8 {
		fail(c, http.StatusUnprocessableEntity, "Invalid emoji")
		return
	}
	user := currentUser(c)
	reaction := models.MessageReaction{GroupMessageID: message.ID, UserID: user.ID, Emoji: emoji}
	if err := s.DB.WithContext(c.Request.Context()).
		Where("group_message_id = ? AND user_id = ? AND emoji = ?", message.ID, user.ID, emoji).
		FirstOrCreate(&reaction).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to add reaction")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"reaction": gin.H{
			"id":               reaction.ID,
			"emoji":            reaction.Emoji,
			"group_message_id": reaction.GroupMessageID,
			"user_id":          reaction.UserID,
			"created_at":       reaction.CreatedAt,
		},
	})
}

func (s *Server) removeMessageReaction(c *gin.Context) {
	if !s.messagingAllowed(c) {
		return
	}
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	emoji := strings.TrimSpace(c.Query("emoji"))
	if emoji == "" {
		var request struct {
			Emoji string `json:"emoji"`
		}
		if bindJSON(c, &request) {
			emoji = strings.TrimSpace(request.Emoji)
		} else {
			return
		}
	}
	user := currentUser(c)
	result := s.DB.WithContext(c.Request.Context()).Where("group_message_id = ? AND user_id = ? AND emoji = ?", message.ID, user.ID, emoji).
		Delete(&models.MessageReaction{})
	if result.Error != nil {
		fail(c, http.StatusInternalServerError, "Unable to remove reaction")
		return
	}
	if result.RowsAffected == 0 {
		fail(c, http.StatusNotFound, "Reaction not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Reaction removed"})
}

func (s *Server) messageReactions(c *gin.Context) {
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	var reactions []models.MessageReaction
	if err := s.DB.WithContext(c.Request.Context()).Where("group_message_id = ?", message.ID).
		Order("created_at ASC").Find(&reactions).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load reactions")
		return
	}
	grouped := map[string][]gin.H{}
	for i := range reactions {
		grouped[reactions[i].Emoji] = append(grouped[reactions[i].Emoji], gin.H{
			"user_id":    reactions[i].UserID,
			"user_name":  s.messageUserName(reactions[i].UserID),
			"created_at": reactions[i].CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "reactions": grouped})
}

func (s *Server) pinMessage(c *gin.Context) {
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	if message.IsPinned {
		fail(c, http.StatusUnprocessableEntity, "Message is already pinned")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models.MessagePin{
			GroupMessageID: message.ID,
			PinnedByID:     currentUser(c).ID,
		}).Error; err != nil {
			return err
		}
		return tx.Model(message).Update("is_pinned", true).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, "Unable to pin message")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Message pinned successfully"})
}

func (s *Server) unpinMessage(c *gin.Context) {
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	if !message.IsPinned {
		fail(c, http.StatusUnprocessableEntity, "Message is not pinned")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("group_message_id = ?", message.ID).Delete(&models.MessagePin{}).Error; err != nil {
			return err
		}
		return tx.Model(message).Update("is_pinned", false).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, "Unable to unpin message")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Message unpinned"})
}

func (s *Server) messageReceipts(c *gin.Context) {
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	var receipts []models.MessageReceipt
	if err := s.DB.WithContext(c.Request.Context()).Where("group_message_id = ?", message.ID).
		Order("read_at DESC NULLS LAST").Limit(50).Find(&receipts).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load receipts")
		return
	}
	items := make([]gin.H, 0, len(receipts))
	seenBy := make([]int64, 0)
	for i := range receipts {
		items = append(items, gin.H{
			"user_id":      receipts[i].UserID,
			"user_name":    s.messageUserName(receipts[i].UserID),
			"delivered_at": receipts[i].DeliveredAt,
			"read_at":      receipts[i].ReadAt,
		})
		if receipts[i].ReadAt != nil {
			seenBy = append(seenBy, receipts[i].UserID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "receipts": items, "seen_by": seenBy})
}

func (s *Server) starMessage(c *gin.Context) {
	if !s.messagingAllowed(c) {
		return
	}
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	user := currentUser(c)
	star := models.MessageStar{GroupMessageID: message.ID, UserID: user.ID}
	if err := s.DB.WithContext(c.Request.Context()).
		Where("group_message_id = ? AND user_id = ?", message.ID, user.ID).
		FirstOrCreate(&star).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to star message")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"star": gin.H{
			"id":               star.ID,
			"group_message_id": star.GroupMessageID,
			"user_id":          star.UserID,
			"created_at":       star.CreatedAt,
		},
	})
}

func (s *Server) unstarMessage(c *gin.Context) {
	if !s.messagingAllowed(c) {
		return
	}
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	user := currentUser(c)
	s.DB.WithContext(c.Request.Context()).Where("group_message_id = ? AND user_id = ?", message.ID, user.ID).
		Delete(&models.MessageStar{})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Message unstarred"})
}

func (s *Server) messageStars(c *gin.Context) {
	message, ok := s.findMessage(c)
	if !ok {
		return
	}
	var stars []models.MessageStar
	if err := s.DB.WithContext(c.Request.Context()).Where("group_message_id = ?", message.ID).
		Order("created_at ASC").Find(&stars).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load stars")
		return
	}
	items := make([]gin.H, 0, len(stars))
	for i := range stars {
		items = append(items, gin.H{
			"user_id":    stars[i].UserID,
			"user_name":  s.messageUserName(stars[i].UserID),
			"starred_at": stars[i].CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "starred_by": items})
}

func (s *Server) searchMessages(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		fail(c, http.StatusUnprocessableEntity, "Search query is required")
		return
	}
	var messages []models.GroupMessage
	if err := s.DB.WithContext(c.Request.Context()).Where("is_deleted = ?", false).
		Where("content ILIKE ?", "%"+query+"%").Order("created_at DESC").Limit(50).Find(&messages).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to search messages")
		return
	}
	items := make([]gin.H, 0, len(messages))
	for i := range messages {
		items = append(items, s.serializeMessage(&messages[i]))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "results": items})
}

func (s *Server) pinnedMessages(c *gin.Context) {
	var messages []models.GroupMessage
	if err := s.DB.WithContext(c.Request.Context()).Where("is_pinned = ? AND is_deleted = ?", true, false).
		Order("created_at DESC").Find(&messages).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load pinned messages")
		return
	}
	items := make([]gin.H, 0, len(messages))
	for i := range messages {
		items = append(items, s.serializeMessage(&messages[i]))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "messages": items})
}
