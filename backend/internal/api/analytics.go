package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

func (s *Server) registerAnalyticsRoutes(api *gin.RouterGroup) {
	analytics := api.Group("/analytics")
	analytics.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted(), s.RequireAdmin())
	analytics.GET("/dashboard", s.analyticsDashboard)
	analytics.GET("/overview", s.analyticsOverview)
	analytics.GET("/members", s.analyticsMembers)
	analytics.GET("/attendance", s.analyticsAttendance)
	analytics.GET("/inventory", s.analyticsInventory)
	analytics.GET("/tasks", s.analyticsTasks)
	analytics.GET("/announcements", s.analyticsAnnouncements)
	analytics.GET("/notifications", s.analyticsNotifications)
	analytics.GET("/contact", s.analyticsContact)
}

func analyticsDate(c *gin.Context, key string, fallback time.Time) (time.Time, bool) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return fallback, true
	}
	parsed, ok := parseDate(value)
	if !ok {
		fail(c, http.StatusUnprocessableEntity, "Invalid date format")
		return time.Time{}, false
	}
	return startOfDay(parsed), true
}

func (s *Server) analyticsDashboard(c *gin.Context) {
	today := startOfDay(time.Now())
	tomorrow := today.AddDate(0, 0, 1)
	activeUsers := s.DB.Model(&models.User{}).Select("id").Where("active = ?", true)

	var totalMembers, activeToday, presentNow int64
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL").Count(&totalMembers)
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL AND created_at >= ? AND created_at < ?", today, tomorrow).Count(&activeToday)
	s.DB.Model(&models.AttendanceLog{}).Where("entry_date = ? AND exit_time IS NULL", today).Count(&presentNow)

	var totalItems, availableItems int64
	s.DB.Model(&models.InventoryItem{}).Where("is_active = ? AND deleted_at IS NULL", true).Count(&totalItems)
	s.DB.Model(&models.InventoryItem{}).Where("is_active = ? AND deleted_at IS NULL AND available_quantity > 0", true).Count(&availableItems)

	var pendingTasks, pendingBorrows, pendingReturns, newContacts, unreadNotifications int64
	s.DB.Model(&models.TaskSubmissionStatus{}).Where("status = ?", "submitted").Count(&pendingTasks)
	s.DB.Model(&models.BorrowRecord{}).Where("status = ?", "pending").Count(&pendingBorrows)
	s.DB.Model(&models.ReturnRequest{}).Where("status = ?", "pending").Count(&pendingReturns)
	s.DB.Model(&models.ContactMessage{}).Where("status = ?", "new").Count(&newContacts)
	s.DB.Model(&models.Notification{}).Where("user_id IN (?) AND is_read = ?", activeUsers, false).Count(&unreadNotifications)

	membersByYear := map[string]int64{}
	var yearRows []struct {
		AcademicYear int   `gorm:"column:academic_year"`
		Count        int64 `gorm:"column:count"`
	}
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL").
		Select("academic_year, COUNT(*) AS count").Group("academic_year").Scan(&yearRows)
	for _, row := range yearRows {
		membersByYear[itoa(row.AcademicYear)] = row.Count
	}
	membersByBranch := map[string]int64{}
	var branchRows []struct {
		Branch string `gorm:"column:branch"`
		Count  int64  `gorm:"column:count"`
	}
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL").
		Select("branch, COUNT(*) AS count").Group("branch").Scan(&branchRows)
	for _, row := range branchRows {
		membersByBranch[row.Branch] = row.Count
	}
	inventoryByCategory := map[string]int64{}
	var categoryRows []struct {
		Category string `gorm:"column:category"`
		Count    int64  `gorm:"column:count"`
	}
	s.DB.Model(&models.InventoryItem{}).Where("is_active = ? AND deleted_at IS NULL", true).
		Select("category, COUNT(*) AS count").Group("category").Scan(&categoryRows)
	for _, row := range categoryRows {
		inventoryByCategory[row.Category] = row.Count
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"dashboard": gin.H{
			"summary": gin.H{
				"total_members":           totalMembers,
				"active_today":            activeToday,
				"present_now":             presentNow,
				"total_items":             totalItems,
				"available_items":         availableItems,
				"pending_tasks":           pendingTasks,
				"pending_borrow_requests": pendingBorrows,
				"pending_return_requests": pendingReturns,
				"new_contact_messages":    newContacts,
				"unread_notifications":    unreadNotifications,
			},
			"trends": gin.H{
				"members_by_academic_year": membersByYear,
				"members_by_branch":        membersByBranch,
				"inventory_by_category":    inventoryByCategory,
			},
		},
	})
}

func (s *Server) analyticsOverview(c *gin.Context) {
	today := startOfDay(time.Now())
	tomorrow := today.AddDate(0, 0, 1)
	activeUsers := s.DB.Model(&models.User{}).Select("id").Where("active = ?", true)

	var totalMembers, activeToday, presentNow int64
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL").Count(&totalMembers)
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL AND created_at >= ? AND created_at < ?", today, tomorrow).Count(&activeToday)
	s.DB.Model(&models.AttendanceLog{}).Where("entry_date = ? AND exit_time IS NULL", today).Count(&presentNow)

	var totalItems, totalAvailable int64
	s.DB.Model(&models.InventoryItem{}).Where("is_active = ? AND deleted_at IS NULL", true).Count(&totalItems)
	s.DB.Model(&models.InventoryItem{}).Where("is_active = ? AND deleted_at IS NULL", true).
		Select("COALESCE(SUM(available_quantity), 0)").Scan(&totalAvailable)

	var pendingTasks, overdueTasks, publishedAnnouncements, unreadNotifications, newContacts int64
	s.DB.Model(&models.TaskSubmissionStatus{}).Where("status = ?", "submitted").Count(&pendingTasks)
	s.DB.Model(&models.Task{}).Where("deadline <= ? AND deleted_at IS NULL", today).Count(&overdueTasks)
	s.DB.Model(&models.Announcement{}).Where("published_at IS NOT NULL AND deleted_at IS NULL").Count(&publishedAnnouncements)
	s.DB.Model(&models.Notification{}).Where("user_id IN (?) AND is_read = ?", activeUsers, false).Count(&unreadNotifications)
	s.DB.Model(&models.ContactMessage{}).Where("status = ?", "new").Count(&newContacts)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"overview": gin.H{
			"members": gin.H{
				"total":        totalMembers,
				"active_today": activeToday,
			},
			"attendance": gin.H{
				"present_now": presentNow,
			},
			"inventory": gin.H{
				"total_items":     totalItems,
				"total_available": totalAvailable,
			},
			"tasks": gin.H{
				"pending": pendingTasks,
				"overdue": overdueTasks,
			},
			"announcements": gin.H{
				"published": publishedAnnouncements,
			},
			"notifications": gin.H{
				"unread": unreadNotifications,
			},
			"contact_messages": gin.H{
				"new": newContacts,
			},
		},
	})
}

func (s *Server) analyticsMembers(c *gin.Context) {
	var totalMembers, activeMembers, inactiveMembers, deactivatedMembers int64
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL").Count(&totalMembers)
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL AND status = ?", "active").Count(&activeMembers)
	s.DB.Model(&models.Member{}).Where("status = ?", "inactive").Count(&inactiveMembers)
	s.DB.Model(&models.Member{}).Where("deleted_at IS NOT NULL").Count(&deactivatedMembers)

	var yearRows []struct {
		AcademicYear int   `gorm:"column:academic_year"`
		Count        int64 `gorm:"column:count"`
	}
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL").
		Select("academic_year, COUNT(*) AS count").Group("academic_year").
		Order("academic_year DESC").Scan(&yearRows)
	yearItems := make([]gin.H, 0, len(yearRows))
	for _, row := range yearRows {
		yearItems = append(yearItems, gin.H{"academic_year": row.AcademicYear, "count": row.Count})
	}

	var branchRows []struct {
		Branch string `gorm:"column:branch"`
		Count  int64  `gorm:"column:count"`
	}
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL").
		Select("branch, COUNT(*) AS count").Group("branch").
		Order("count DESC").Scan(&branchRows)
	branchItems := make([]gin.H, 0, len(branchRows))
	for _, row := range branchRows {
		branchItems = append(branchItems, gin.H{"branch": row.Branch, "count": row.Count})
	}

	var sectionRows []struct {
		Section *string `gorm:"column:section"`
		Count   int64   `gorm:"column:count"`
	}
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL").
		Select("section, COUNT(*) AS count").Group("section").
		Order("count DESC").Scan(&sectionRows)
	sectionItems := make([]gin.H, 0, len(sectionRows))
	for _, row := range sectionRows {
		sectionItems = append(sectionItems, gin.H{"section": row.Section, "count": row.Count})
	}

	var recent []struct {
		ID           int64     `gorm:"column:id"`
		Name         string    `gorm:"column:name"`
		LoginID      string    `gorm:"column:login_id"`
		Email        string    `gorm:"column:email"`
		Branch       string    `gorm:"column:branch"`
		AcademicYear int       `gorm:"column:academic_year"`
		CreatedAt    time.Time `gorm:"column:created_at"`
	}
	s.DB.Model(&models.Member{}).Where("deleted_at IS NULL").
		Select("id, name, login_id, email, branch, academic_year, created_at").
		Order("created_at DESC").Limit(10).Scan(&recent)
	recentItems := make([]gin.H, 0, len(recent))
	for _, member := range recent {
		recentItems = append(recentItems, gin.H{
			"id":            member.ID,
			"name":          member.Name,
			"login_id":      member.LoginID,
			"email":         member.Email,
			"branch":        member.Branch,
			"academic_year": member.AcademicYear,
			"created_at":    member.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"members": gin.H{
			"total":                totalMembers,
			"active":               activeMembers,
			"inactive":             inactiveMembers,
			"deactivated":          deactivatedMembers,
			"by_academic_year":     yearItems,
			"by_branch":            branchItems,
			"by_section":           sectionItems,
			"recent_registrations": recentItems,
		},
	})
}

func (s *Server) analyticsAttendance(c *gin.Context) {
	today := startOfDay(time.Now())
	dateFrom, ok := analyticsDate(c, "date_from", today.AddDate(0, 0, -30))
	if !ok {
		return
	}
	dateTo, ok := analyticsDate(c, "date_to", today)
	if !ok {
		return
	}

	var totalLoginEvents, totalLogoutEvents, uniqueMembers int64
	s.DB.Model(&models.AttendanceLog{}).Where("entry_date BETWEEN ? AND ? AND exit_time IS NULL", dateFrom, dateTo).Count(&totalLoginEvents)
	s.DB.Model(&models.AttendanceLog{}).Where("entry_date BETWEEN ? AND ? AND exit_time IS NOT NULL", dateFrom, dateTo).Count(&totalLogoutEvents)
	s.DB.Model(&models.AttendanceLog{}).Where("entry_date BETWEEN ? AND ?", dateFrom, dateTo).
		Distinct("member_id").Count(&uniqueMembers)

	var presentNow int64
	s.DB.Model(&models.AttendanceLog{}).Where("entry_date = ? AND exit_time IS NULL", today).Count(&presentNow)

	var daily []struct {
		EntryDate      time.Time `gorm:"column:entry_date"`
		TotalEvents    int64     `gorm:"column:total_events"`
		CurrentPresent int64     `gorm:"column:current_present"`
		UniqueMembers  int64     `gorm:"column:unique_members"`
	}
	s.DB.Model(&models.AttendanceLog{}).
		Select("entry_date, COUNT(*) AS total_events, COUNT(CASE WHEN exit_time IS NULL THEN 1 END) AS current_present, COUNT(DISTINCT member_id) AS unique_members").
		Where("entry_date BETWEEN ? AND ?", dateFrom, dateTo).
		Group("entry_date").Order("entry_date DESC").Scan(&daily)
	dailyItems := make([]gin.H, 0, len(daily))
	for _, row := range daily {
		dailyItems = append(dailyItems, gin.H{
			"date":            dateValue(row.EntryDate),
			"total_events":    row.TotalEvents,
			"current_present": row.CurrentPresent,
			"unique_members":  row.UniqueMembers,
		})
	}

	var branches []struct {
		Branch        string `gorm:"column:branch"`
		Events        int64  `gorm:"column:events"`
		UniqueMembers int64  `gorm:"column:unique_members"`
	}
	s.DB.Model(&models.AttendanceLog{}).
		Select("members.branch AS branch, COUNT(*) AS events, COUNT(DISTINCT attendance_logs.member_id) AS unique_members").
		Joins("JOIN members ON members.id = attendance_logs.member_id").
		Where("attendance_logs.entry_date BETWEEN ? AND ?", dateFrom, dateTo).
		Group("members.branch").Order("events DESC").Scan(&branches)
	branchItems := make([]gin.H, 0, len(branches))
	for _, row := range branches {
		branchItems = append(branchItems, gin.H{
			"branch":         row.Branch,
			"events":         row.Events,
			"unique_members": row.UniqueMembers,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"attendance": gin.H{
			"summary": gin.H{
				"total_login_events":     totalLoginEvents,
				"total_logout_events":    totalLogoutEvents,
				"unique_members_present": uniqueMembers,
				"date_range":             gin.H{"from": dateValue(dateFrom), "to": dateValue(dateTo)},
			},
			"current_occupancy": gin.H{
				"present_now": presentNow,
				"date":        dateValue(today),
				"timestamp":   time.Now().UTC(),
			},
			"daily_summary": dailyItems,
			"by_branch":     branchItems,
		},
	})
}

func (s *Server) analyticsInventory(c *gin.Context) {
	today := startOfDay(time.Now())
	var totalItems, totalStock, availableStock int64
	s.DB.Model(&models.InventoryItem{}).Where("is_active = ? AND deleted_at IS NULL", true).Count(&totalItems)
	s.DB.Model(&models.InventoryItem{}).Where("is_active = ? AND deleted_at IS NULL", true).
		Select("COALESCE(SUM(total_quantity), 0)").Scan(&totalStock)
	s.DB.Model(&models.InventoryItem{}).Where("is_active = ? AND deleted_at IS NULL", true).
		Select("COALESCE(SUM(available_quantity), 0)").Scan(&availableStock)

	var categories []struct {
		Category          string `gorm:"column:category"`
		Items             int64  `gorm:"column:items"`
		TotalQuantity     int64  `gorm:"column:total_quantity"`
		AvailableQuantity int64  `gorm:"column:available_quantity"`
	}
	s.DB.Model(&models.InventoryItem{}).Where("is_active = ? AND deleted_at IS NULL", true).
		Select("category, COUNT(*) AS items, SUM(total_quantity) AS total_quantity, SUM(available_quantity) AS available_quantity").
		Group("category").Order("total_quantity DESC").Scan(&categories)
	categoryItems := make([]gin.H, 0, len(categories))
	for _, row := range categories {
		categoryItems = append(categoryItems, gin.H{
			"category":           row.Category,
			"items":              row.Items,
			"total_quantity":     row.TotalQuantity,
			"available_quantity": row.AvailableQuantity,
		})
	}

	var lowStock []models.InventoryItem
	s.DB.Where("is_active = ? AND deleted_at IS NULL AND available_quantity <= ?", true, 3).
		Order("available_quantity ASC").Limit(10).Find(&lowStock)
	lowStockItems := make([]gin.H, 0, len(lowStock))
	for i := range lowStock {
		lowStockItems = append(lowStockItems, gin.H{
			"id":                 lowStock[i].ID,
			"name":               lowStock[i].Name,
			"category":           lowStock[i].Category,
			"available_quantity": lowStock[i].AvailableQuantity,
			"total_quantity":     lowStock[i].TotalQuantity,
		})
	}

	var recentImports []models.InventoryImport
	s.DB.Order("created_at DESC").Limit(10).Find(&recentImports)
	importItems := make([]gin.H, 0, len(recentImports))
	for i := range recentImports {
		importItems = append(importItems, gin.H{
			"id":            recentImports[i].ID,
			"filename":      recentImports[i].Filename,
			"total_rows":    recentImports[i].TotalRows,
			"imported_rows": recentImports[i].ImportedRows,
			"error_rows":    recentImports[i].ErrorRows,
			"status":        recentImports[i].Status,
			"created_at":    recentImports[i].CreatedAt,
		})
	}

	var activeBorrows, overdueBorrows int64
	s.DB.Model(&models.BorrowRecord{}).Where("status = ?", "approved").Count(&activeBorrows)
	s.DB.Model(&models.BorrowRecord{}).Where("status = ? AND due_date < ?", "approved", today).Count(&overdueBorrows)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"inventory": gin.H{
			"summary": gin.H{
				"total_items":     totalItems,
				"total_stock":     totalStock,
				"available_stock": availableStock,
				"borrowed_stock":  totalStock - availableStock,
			},
			"by_category":           categoryItems,
			"low_stock_items":       lowStockItems,
			"active_borrows_count":  activeBorrows,
			"overdue_borrows_count": overdueBorrows,
			"recent_imports":        importItems,
		},
	})
}

func (s *Server) analyticsTasks(c *gin.Context) {
	today := startOfDay(time.Now())
	var totalTasks, publishedTasks, overdueTasks int64
	s.DB.Model(&models.Task{}).Where("deleted_at IS NULL").Count(&totalTasks)
	s.DB.Model(&models.Task{}).Where("published_at IS NOT NULL AND deleted_at IS NULL").Count(&publishedTasks)
	s.DB.Model(&models.Task{}).Where("published_at IS NOT NULL AND deleted_at IS NULL AND deadline < ?", today).Count(&overdueTasks)

	var pendingSubmissions, verifiedSubmissions, rejectedSubmissions int64
	s.DB.Model(&models.TaskSubmissionStatus{}).Where("status = ?", "submitted").Count(&pendingSubmissions)
	s.DB.Model(&models.TaskSubmissionStatus{}).Where("status = ?", "verified").Count(&verifiedSubmissions)
	s.DB.Model(&models.TaskSubmissionStatus{}).Where("status = ?", "rejected").Count(&rejectedSubmissions)

	var typeRows []struct {
		TaskType string `gorm:"column:task_type"`
		Count    int64  `gorm:"column:count"`
	}
	s.DB.Model(&models.Task{}).Where("deleted_at IS NULL").
		Select("task_type, COUNT(*) AS count").Group("task_type").Order("count DESC").Scan(&typeRows)
	typeItems := make([]gin.H, 0, len(typeRows))
	for _, row := range typeRows {
		typeItems = append(typeItems, gin.H{"task_type": row.TaskType, "count": row.Count})
	}

	var yearRows []struct {
		AcademicYear int   `gorm:"column:academic_year"`
		Count        int64 `gorm:"column:count"`
	}
	s.DB.Table("tasks").Select("task_targets.academic_year AS academic_year, COUNT(*) AS count").
		Joins("JOIN task_targets ON task_targets.task_id = tasks.id").
		Where("tasks.deleted_at IS NULL").
		Group("task_targets.academic_year").Order("count DESC").Scan(&yearRows)
	yearItems := make([]gin.H, 0, len(yearRows))
	for _, row := range yearRows {
		yearItems = append(yearItems, gin.H{"academic_year": row.AcademicYear, "count": row.Count})
	}

	var recent []models.Task
	s.DB.Where("deleted_at IS NULL").Order("published_at DESC NULLS LAST, created_at DESC").Limit(10).Find(&recent)
	recentItems := make([]gin.H, 0, len(recent))
	for i := range recent {
		var submissionCount int64
		s.DB.Model(&models.TaskSubmissionStatus{}).Where("task_id = ?", recent[i].ID).Count(&submissionCount)
		recentItems = append(recentItems, gin.H{
			"id":               recent[i].ID,
			"title":            recent[i].Title,
			"task_type":        recent[i].TaskType,
			"deadline":         dateValue(recent[i].Deadline),
			"published_at":     recent[i].PublishedAt,
			"submission_count": submissionCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tasks": gin.H{
			"summary": gin.H{
				"total_tasks":          totalTasks,
				"published_tasks":      publishedTasks,
				"overdue_tasks":        overdueTasks,
				"pending_submissions":  pendingSubmissions,
				"verified_submissions": verifiedSubmissions,
				"rejected_submissions": rejectedSubmissions,
			},
			"by_type":          typeItems,
			"by_academic_year": yearItems,
			"recent_tasks":     recentItems,
		},
	})
}

func (s *Server) analyticsAnnouncements(c *gin.Context) {
	var totalAnnouncements, publishedAnnouncements, pinnedAnnouncements int64
	s.DB.Model(&models.Announcement{}).Where("deleted_at IS NULL").Count(&totalAnnouncements)
	s.DB.Model(&models.Announcement{}).Where("published_at IS NOT NULL AND deleted_at IS NULL").Count(&publishedAnnouncements)
	s.DB.Model(&models.Announcement{}).Where("is_pinned = ? AND deleted_at IS NULL", true).Count(&pinnedAnnouncements)

	var totalUsers int64
	s.DB.Model(&models.User{}).Where("active = ? AND role IN ?", true, []string{"admin", "primary_admin", "member"}).Count(&totalUsers)

	var rows []struct {
		ID          int64      `gorm:"column:id"`
		Title       string     `gorm:"column:title"`
		PublishedAt *time.Time `gorm:"column:published_at"`
		ReadCount   int64      `gorm:"column:read_count"`
	}
	s.DB.Table("announcements").
		Select("announcements.id, announcements.title, announcements.published_at, COUNT(DISTINCT announcement_reads.user_id) AS read_count").
		Joins("LEFT JOIN announcement_reads ON announcement_reads.announcement_id = announcements.id").
		Where("announcements.published_at IS NOT NULL AND announcements.deleted_at IS NULL").
		Group("announcements.id").Order("announcements.published_at DESC").Scan(&rows)

	var totalReads int64
	for _, row := range rows {
		totalReads += row.ReadCount
	}
	recentItems := make([]gin.H, 0, len(rows))
	for index, row := range rows {
		if index >= 10 {
			break
		}
		percentage := float64(0)
		if totalUsers > 0 {
			percentage = roundTo(float64(row.ReadCount)/float64(totalUsers)*100, 2)
		}
		recentItems = append(recentItems, gin.H{
			"id":              row.ID,
			"title":           row.Title,
			"published_at":    row.PublishedAt,
			"read_count":      row.ReadCount,
			"read_percentage": percentage,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"announcements": gin.H{
			"summary": gin.H{
				"total_announcements":     totalAnnouncements,
				"published_announcements": publishedAnnouncements,
				"pinned_announcements":    pinnedAnnouncements,
				"total_reads":             totalReads,
			},
			"recent": recentItems,
		},
	})
}

func (s *Server) analyticsNotifications(c *gin.Context) {
	activeUsers := s.DB.Model(&models.User{}).Select("id").Where("active = ?", true)
	var total, unread int64
	s.DB.Model(&models.Notification{}).Where("user_id IN (?)", activeUsers).Count(&total)
	s.DB.Model(&models.Notification{}).Where("user_id IN (?) AND is_read = ?", activeUsers, false).Count(&unread)

	var typeRows []struct {
		NotificationType string `gorm:"column:notification_type"`
		Count            int64  `gorm:"column:count"`
	}
	s.DB.Model(&models.Notification{}).Where("user_id IN (?)", activeUsers).
		Select("notification_type, COUNT(*) AS count").Group("notification_type").
		Order("count DESC").Scan(&typeRows)
	typeItems := make([]gin.H, 0, len(typeRows))
	for _, row := range typeRows {
		typeItems = append(typeItems, gin.H{"notification_type": row.NotificationType, "count": row.Count})
	}

	var recent []models.Notification
	s.DB.Where("user_id IN (?)", activeUsers).Order("created_at DESC").Limit(20).Find(&recent)
	recentItems := make([]gin.H, 0, len(recent))
	for i := range recent {
		recentItems = append(recentItems, gin.H{
			"id":                recent[i].ID,
			"title":             recent[i].Title,
			"body":              recent[i].Body,
			"notification_type": recent[i].NotificationType,
			"is_read":           recent[i].IsRead,
			"created_at":        recent[i].CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"notifications": gin.H{
			"total":   total,
			"unread":  unread,
			"by_type": typeItems,
			"recent":  recentItems,
		},
	})
}

func (s *Server) analyticsContact(c *gin.Context) {
	var totalMessages, newMessages, inProgress, resolvedMessages, closedMessages int64
	s.DB.Model(&models.ContactMessage{}).Count(&totalMessages)
	s.DB.Model(&models.ContactMessage{}).Where("status = ?", "new").Count(&newMessages)
	s.DB.Model(&models.ContactMessage{}).Where("status = ?", "in_progress").Count(&inProgress)
	s.DB.Model(&models.ContactMessage{}).Where("status = ?", "resolved").Count(&resolvedMessages)
	s.DB.Model(&models.ContactMessage{}).Where("status = ?", "closed").Count(&closedMessages)

	var categoryRows []struct {
		Category string `gorm:"column:category"`
		Count    int64  `gorm:"column:count"`
	}
	s.DB.Model(&models.ContactMessage{}).
		Select("category, COUNT(*) AS count").Group("category").Order("count DESC").Scan(&categoryRows)
	categoryItems := make([]gin.H, 0, len(categoryRows))
	for _, row := range categoryRows {
		categoryItems = append(categoryItems, gin.H{"category": row.Category, "count": row.Count})
	}

	var statusRows []struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:count"`
	}
	s.DB.Model(&models.ContactMessage{}).
		Select("status, COUNT(*) AS count").Group("status").Order("count DESC").Scan(&statusRows)
	statusItems := make([]gin.H, 0, len(statusRows))
	for _, row := range statusRows {
		statusItems = append(statusItems, gin.H{"status": row.Status, "count": row.Count})
	}

	var recent []models.ContactMessage
	s.DB.Preload("Member").Order("created_at DESC").Limit(20).Find(&recent)
	recentItems := make([]gin.H, 0, len(recent))
	for i := range recent {
		recentItems = append(recentItems, gin.H{
			"id":              recent[i].ID,
			"member_name":     recent[i].Member.Name,
			"member_login_id": recent[i].Member.LoginID,
			"subject":         recent[i].Subject,
			"category":        recent[i].Category,
			"status":          recent[i].Status,
			"created_at":      recent[i].CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"contact": gin.H{
			"summary": gin.H{
				"total_messages":    totalMessages,
				"new_messages":      newMessages,
				"in_progress":       inProgress,
				"resolved_messages": resolvedMessages,
				"closed_messages":   closedMessages,
			},
			"by_category": categoryItems,
			"by_status":   statusItems,
			"recent":      recentItems,
		},
	})
}
