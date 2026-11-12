package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

func (s *Server) registerAuditLogRoutes(api *gin.RouterGroup) {
	auditLogs := api.Group("/audit-logs")
	auditLogs.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted(), s.RequirePrimaryAdmin())
	auditLogs.GET("", s.listAuditLogs)
	auditLogs.GET("/summary", s.auditLogSummary)
	auditLogs.GET("/actions", s.auditLogActions)
	auditLogs.GET("/resource/:resource_type/:resource_id", s.auditLogsForResource)
	auditLogs.GET("/user/:user_id", s.auditLogsForUser)
	auditLogs.GET("/:id", s.showAuditLog)
}

func (s *Server) auditLogUserMap(entries []models.AuditLog) map[int64]models.User {
	ids := make([]int64, 0)
	seen := map[int64]bool{}
	for _, entry := range entries {
		if entry.UserID == nil || seen[*entry.UserID] {
			continue
		}
		seen[*entry.UserID] = true
		ids = append(ids, *entry.UserID)
	}
	if len(ids) == 0 {
		return map[int64]models.User{}
	}
	var users []models.User
	s.DB.Where("id IN ?", ids).Find(&users)
	result := make(map[int64]models.User, len(users))
	for _, user := range users {
		result[user.ID] = user
	}
	return result
}

func (s *Server) serializeAuditLog(entry *models.AuditLog, users map[int64]models.User) gin.H {
	serialized := gin.H{
		"id":            entry.ID,
		"user_id":       entry.UserID,
		"action":        entry.Action,
		"resource_type": entry.ResourceType,
		"resource_id":   entry.ResourceID,
		"details":       entry.Details,
		"ip_address":    entry.IPAddress,
		"created_at":    entry.CreatedAt,
	}
	if entry.UserID != nil {
		if user, ok := users[*entry.UserID]; ok {
			serialized["user"] = gin.H{
				"id":    user.ID,
				"email": user.Email,
				"role":  user.Role,
			}
		}
	}
	return serialized
}

func (s *Server) auditLogScope(c *gin.Context) (*gorm.DB, bool) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.AuditLog{})
	if value := strings.TrimSpace(c.Query("user_id")); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 {
			query = query.Where("user_id = ?", parsed)
		}
	}
	if value := strings.TrimSpace(c.Query("action")); value != "" {
		query = query.Where("action = ?", value)
	}
	if value := strings.TrimSpace(c.Query("resource_type")); value != "" {
		query = query.Where("resource_type = ?", value)
	}
	if value := strings.TrimSpace(c.Query("resource_id")); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 {
			query = query.Where("resource_id = ?", parsed)
		}
	}
	if value := strings.TrimSpace(c.Query("date_from")); value != "" {
		parsed, ok := parseDate(value)
		if !ok {
			fail(c, http.StatusUnprocessableEntity, "Invalid date format")
			return nil, false
		}
		query = query.Where("created_at >= ?", startOfDay(parsed))
	}
	if value := strings.TrimSpace(c.Query("date_to")); value != "" {
		parsed, ok := parseDate(value)
		if !ok {
			fail(c, http.StatusUnprocessableEntity, "Invalid date format")
			return nil, false
		}
		query = query.Where("created_at < ?", startOfDay(parsed).AddDate(0, 0, 1))
	}
	return query, true
}

func (s *Server) listAuditLogs(c *gin.Context) {
	query, ok := s.auditLogScope(c)
	if !ok {
		return
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load audit logs")
		return
	}
	var entries []models.AuditLog
	if err := query.Order("created_at DESC, id DESC").
		Offset(offsetFor(page, perPage)).Limit(perPage).Find(&entries).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load audit logs")
		return
	}
	users := s.auditLogUserMap(entries)
	result := make([]gin.H, 0, len(entries))
	for i := range entries {
		result = append(result, s.serializeAuditLog(&entries[i], users))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"audit_logs": result,
		"pagination": makePagination(page, perPage, total),
	})
}

func (s *Server) showAuditLog(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var entry models.AuditLog
	if err := s.DB.WithContext(c.Request.Context()).First(&entry, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Audit log not found")
		return
	}
	users := s.auditLogUserMap([]models.AuditLog{entry})
	c.JSON(http.StatusOK, gin.H{"success": true, "audit_log": s.serializeAuditLog(&entry, users)})
}

func (s *Server) auditLogActions(c *gin.Context) {
	var actions []string
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.AuditLog{}).
		Distinct("action").Order("action ASC").Pluck("action", &actions).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load audit actions")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "actions": actions})
}

func (s *Server) auditLogSummary(c *gin.Context) {
	today := startOfDay(time.Now())
	var total, todayCount, weekCount int64
	s.DB.Model(&models.AuditLog{}).Count(&total)
	s.DB.Model(&models.AuditLog{}).Where("created_at >= ?", today).Count(&todayCount)
	s.DB.Model(&models.AuditLog{}).Where("created_at >= ?", today.AddDate(0, 0, -7)).Count(&weekCount)

	var actionRows []struct {
		Action string `gorm:"column:action"`
		Count  int64  `gorm:"column:count"`
	}
	s.DB.Model(&models.AuditLog{}).Select("action, COUNT(*) AS count").
		Group("action").Order("count DESC").Limit(20).Scan(&actionRows)
	actionItems := make([]gin.H, 0, len(actionRows))
	for _, row := range actionRows {
		actionItems = append(actionItems, gin.H{"action": row.Action, "count": row.Count})
	}

	var resourceRows []struct {
		ResourceType string `gorm:"column:resource_type"`
		Count        int64  `gorm:"column:count"`
	}
	s.DB.Model(&models.AuditLog{}).Select("resource_type, COUNT(*) AS count").
		Group("resource_type").Order("count DESC").Limit(20).Scan(&resourceRows)
	resourceItems := make([]gin.H, 0, len(resourceRows))
	for _, row := range resourceRows {
		resourceItems = append(resourceItems, gin.H{"resource_type": row.ResourceType, "count": row.Count})
	}

	var recent []models.AuditLog
	s.DB.Order("created_at DESC, id DESC").Limit(10).Find(&recent)
	users := s.auditLogUserMap(recent)
	recentItems := make([]gin.H, 0, len(recent))
	for i := range recent {
		recentItems = append(recentItems, s.serializeAuditLog(&recent[i], users))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"summary": gin.H{
			"total":            total,
			"today":            todayCount,
			"last_7_days":      weekCount,
			"by_action":        actionItems,
			"by_resource_type": resourceItems,
			"recent":           recentItems,
		},
	})
}

func (s *Server) auditLogsForResource(c *gin.Context) {
	resourceType := strings.TrimSpace(c.Param("resource_type"))
	resourceID, err := strconv.ParseInt(strings.TrimSpace(c.Param("resource_id")), 10, 64)
	if err != nil || resourceID <= 0 {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	page, perPage := pagination(c, 20)
	query := s.DB.WithContext(c.Request.Context()).Model(&models.AuditLog{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID)
	var total int64
	query.Count(&total)
	var entries []models.AuditLog
	query.Order("created_at DESC, id DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&entries)
	users := s.auditLogUserMap(entries)
	result := make([]gin.H, 0, len(entries))
	for i := range entries {
		result = append(result, s.serializeAuditLog(&entries[i], users))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"audit_logs": result,
		"pagination": makePagination(page, perPage, total),
	})
}

func (s *Server) auditLogsForUser(c *gin.Context) {
	userID, err := strconv.ParseInt(strings.TrimSpace(c.Param("user_id")), 10, 64)
	if err != nil || userID <= 0 {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	page, perPage := pagination(c, 20)
	query := s.DB.WithContext(c.Request.Context()).Model(&models.AuditLog{}).Where("user_id = ?", userID)
	var total int64
	query.Count(&total)
	var entries []models.AuditLog
	query.Order("created_at DESC, id DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&entries)
	users := s.auditLogUserMap(entries)
	result := make([]gin.H, 0, len(entries))
	for i := range entries {
		result = append(result, s.serializeAuditLog(&entries[i], users))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"audit_logs": result,
		"pagination": makePagination(page, perPage, total),
	})
}
