package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

func (s *Server) registerAttendanceRoutes(api *gin.RouterGroup) {
	attendance := api.Group("/attendance")
	attendance.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted())
	attendance.GET("", s.listAttendance)
	attendance.GET("/today", s.todayAttendance)
	attendance.GET("/summary", s.attendanceSummary)
	attendance.GET("/daily-summary", s.dailyAttendanceSummary)
	attendance.GET("/occupancy", s.attendanceOccupancy)
	attendance.GET("/analytics", s.attendanceAnalytics)
	attendance.POST("", s.RequireAdmin(), s.createAttendance)
	attendance.GET("/:id", s.showAttendance)
	attendance.PATCH("/:id/logout", s.RequireAdmin(), s.recordAttendanceLogout)
}

func (s *Server) registerBiometricRoutes(api *gin.RouterGroup) {
	biometric := api.Group("/biometric")
	biometric.POST("/authenticate-device", s.RateLimit("biometric-auth", 30, 5*time.Minute), s.authenticateDeviceEndpoint)
	biometric.POST("/enroll-fingerprint", s.RequireDevice(), s.enrollFingerprint)
	biometric.GET("/enrollments", s.RequireDevice(), s.listBiometricEnrollments)
	biometric.DELETE("/enrollments/:id", s.RequireDevice(), s.removeBiometricEnrollment)
	biometric.POST("/readers", s.RequireDevice(), s.reportDeviceReaders)
	biometric.GET("/enrollment-session", s.RequireDevice(), s.claimEnrollmentSession)
	biometric.GET("/enrollment-sessions/:uid", s.RequireDevice(), s.showEnrollmentSessionForDevice)
	biometric.POST("/enrollment-sessions/:uid/progress", s.RequireDevice(), s.updateEnrollmentSessionProgress)
	biometric.POST("/enrollment-sessions/:uid/complete", s.RequireDevice(), s.completeEnrollmentSession)
	biometric.POST("/enrollment-sessions/:uid/fail", s.RequireDevice(), s.failEnrollmentSession)
	biometric.POST("/enrollment-sessions/:uid/cleanup", s.RequireDevice(), s.acknowledgeEnrollmentCleanup)
	biometric.GET("/status", s.RequireDevice(), s.biometricDeviceStatus)
	// submit-event also supports HMAC-only submissions without a device token.
	biometric.POST("/submit-event", s.RateLimit("biometric-event", 240, time.Minute), s.OptionalDevice(), s.submitBiometricEvent)
	biometric.POST("/sync-events", s.RequireDevice(), s.syncBiometricEvents)
}

type taskCreateRequest struct {
	Title               string `json:"title"`
	Description         string `json:"description"`
	TaskType            string `json:"task_type"`
	Deadline            string `json:"deadline"`
	Instructions        string `json:"instructions"`
	ExternalLink        string `json:"external_link"`
	RequiredAttachments string `json:"required_attachments"`
	TargetAcademicYears []int  `json:"target_academic_years"`
}

type taskUpdateRequest struct {
	Title               *string `json:"title"`
	Description         *string `json:"description"`
	Deadline            *string `json:"deadline"`
	Instructions        *string `json:"instructions"`
	ExternalLink        *string `json:"external_link"`
	RequiredAttachments *string `json:"required_attachments"`
	TargetAcademicYears *[]int  `json:"target_academic_years"`
}

type taskVerifyRequest struct {
	MemberID           int64  `json:"member_id"`
	Decision           string `json:"decision"`
	VerificationAction string `json:"verification_action"`
}

func validateTaskType(value string) bool {
	switch value {
	case "assignment", "project", "report", "activity", "other":
		return true
	default:
		return false
	}
}

func (s *Server) registerTaskRoutes(api *gin.RouterGroup) {
	tasks := api.Group("/tasks")
	tasks.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted())
	tasks.GET("", s.listTasks)
	tasks.GET("/:id", s.showTask)
	tasks.POST("", s.RequireAdmin(), s.createTask)
	tasks.PATCH("/:id", s.RequireAdmin(), s.updateTask)
	tasks.DELETE("/:id", s.RequireAdmin(), s.deleteTask)
	tasks.POST("/:id/submit", s.submitTask)
	tasks.POST("/:id/verify", s.RequireAdmin(), s.verifyTaskSubmission)
}

func (s *Server) taskTargetYears(taskID int64) []int {
	var targets []models.TaskTarget
	s.DB.Where("task_id = ?", taskID).Order("academic_year ASC").Find(&targets)
	years := make([]int, 0, len(targets))
	for i := range targets {
		years = append(years, targets[i].AcademicYear)
	}
	return years
}

func (s *Server) taskCreatorName(userID int64) gin.H {
	var user models.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return gin.H{"id": userID, "name": ""}
	}
	name := user.Email
	var admin models.Admin
	if err := s.DB.Where("user_id = ?", userID).First(&admin).Error; err == nil && strings.TrimSpace(admin.Name) != "" {
		name = admin.Name
	}
	return gin.H{"id": user.ID, "name": name}
}

func (s *Server) serializeSubmission(submission *models.TaskSubmissionStatus) gin.H {
	if submission == nil {
		return nil
	}
	var verifiedBy any
	if submission.VerifiedByID != nil {
		var user models.User
		if err := s.DB.First(&user, *submission.VerifiedByID).Error; err == nil {
			verifiedBy = user.Email
		}
	}
	return gin.H{
		"id":           submission.ID,
		"task_id":      submission.TaskID,
		"member_id":    submission.MemberID,
		"status":       submission.Status,
		"submitted_at": submission.SubmittedAt,
		"verified_at":  submission.VerifiedAt,
		"verified_by":  verifiedBy,
	}
}

func (s *Server) serializeTask(task *models.Task, submission *models.TaskSubmissionStatus) gin.H {
	var submissionStatus any
	isSubmitted := false
	isVerified := false
	if submission != nil {
		submissionStatus = submission.Status
		isSubmitted = submission.Status == "submitted"
		isVerified = submission.Status == "verified"
	}
	return gin.H{
		"id":                    task.ID,
		"title":                 task.Title,
		"task_type":             task.TaskType,
		"deadline":              dateValue(task.Deadline),
		"instructions":          task.Instructions,
		"external_link":         task.ExternalLink,
		"required_attachments":  task.RequiredAttachments,
		"created_by":            s.taskCreatorName(task.CreatedByID),
		"target_academic_years": s.taskTargetYears(task.ID),
		"created_at":            task.CreatedAt,
		"published_at":          task.PublishedAt,
		"submission_status":     submissionStatus,
		"is_submitted":          isSubmitted,
		"is_verified":           isVerified,
	}
}

func (s *Server) serializeTaskFull(task *models.Task, submission *models.TaskSubmissionStatus) gin.H {
	result := s.serializeTask(task, submission)
	result["description"] = task.Description
	result["submission"] = s.serializeSubmission(submission)
	return result
}

func (s *Server) memberSubmission(memberID, taskID int64) *models.TaskSubmissionStatus {
	if memberID == 0 {
		return nil
	}
	var submission models.TaskSubmissionStatus
	if err := s.DB.Where("task_id = ? AND member_id = ?", taskID, memberID).First(&submission).Error; err != nil {
		return nil
	}
	return &submission
}

func (s *Server) listTasks(c *gin.Context) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.Task{}).Where("tasks.deleted_at IS NULL")
	if upcoming := strings.TrimSpace(c.Query("upcoming")); upcoming == "true" {
		query = query.Where("tasks.deadline >= ?", startOfDay(time.Now()))
	}
	if overdue := strings.TrimSpace(c.Query("overdue")); overdue == "true" {
		query = query.Where("tasks.deadline < ?", startOfDay(time.Now()))
	}
	if taskType := strings.TrimSpace(c.Query("task_type")); taskType != "" {
		query = query.Where("tasks.task_type = ?", taskType)
	}

	user := currentUser(c)
	member := currentMember(c)
	if user != nil && user.Role != "admin" && member != nil && member.Status == "active" {
		query = query.Where("tasks.id IN (?)", s.DB.Model(&models.TaskTarget{}).
			Select("task_id").Where("academic_year = ?", member.AcademicYear))
	}

	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.Log.Error("task list count failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load tasks")
		return
	}
	var tasks []models.Task
	if err := query.Order("tasks.deadline DESC, tasks.published_at DESC").
		Offset(offsetFor(page, perPage)).Limit(perPage).Find(&tasks).Error; err != nil {
		s.Log.Error("task list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load tasks")
		return
	}

	memberID := int64(0)
	if user != nil && user.Role != "admin" && member != nil {
		memberID = member.ID
	}
	items := make([]gin.H, 0, len(tasks))
	for i := range tasks {
		items = append(items, s.serializeTask(&tasks[i], s.memberSubmission(memberID, tasks[i].ID)))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"tasks":      items,
		"pagination": makePagination(page, perPage, total),
	})
}

func (s *Server) findTask(c *gin.Context) (*models.Task, bool) {
	id, ok := pathID(c, "id")
	if !ok {
		return nil, false
	}
	var task models.Task
	if err := s.DB.WithContext(c.Request.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&task).Error; err != nil {
		fail(c, http.StatusNotFound, "Task not found")
		return nil, false
	}
	return &task, true
}

func (s *Server) showTask(c *gin.Context) {
	task, ok := s.findTask(c)
	if !ok {
		return
	}
	member := currentMember(c)
	user := currentUser(c)
	memberID := int64(0)
	if member != nil && user != nil && user.Role != "admin" {
		memberID = member.ID
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"task":    s.serializeTaskFull(task, s.memberSubmission(memberID, task.ID)),
	})
}

func (s *Server) createTask(c *gin.Context) {
	var request taskCreateRequest
	if !bindJSON(c, &request) {
		return
	}
	title := strings.TrimSpace(request.Title)
	taskType := strings.ToLower(strings.TrimSpace(request.TaskType))
	if title == "" {
		fail(c, http.StatusUnprocessableEntity, "Title is required")
		return
	}
	if !validateTaskType(taskType) {
		fail(c, http.StatusUnprocessableEntity, "Invalid task type")
		return
	}
	deadline, ok := parseDate(request.Deadline)
	if !ok {
		fail(c, http.StatusUnprocessableEntity, "Deadline is required")
		return
	}
	if len(request.TargetAcademicYears) == 0 {
		fail(c, http.StatusUnprocessableEntity, "At least one target academic year is required")
		return
	}
	seenYears := map[int]struct{}{}
	years := make([]int, 0, len(request.TargetAcademicYears))
	for _, year := range request.TargetAcademicYears {
		if year < 1 || year > 8 {
			fail(c, http.StatusUnprocessableEntity, "Academic year must be between 1 and 8")
			return
		}
		if _, seen := seenYears[year]; seen {
			continue
		}
		seenYears[year] = struct{}{}
		years = append(years, year)
	}

	user := currentUser(c)
	if user == nil {
		fail(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	now := time.Now()
	task := models.Task{
		Title:               title,
		Description:         nullableString(request.Description),
		TaskType:            taskType,
		PublishedAt:         &now,
		Deadline:            deadline,
		Instructions:        nullableString(request.Instructions),
		ExternalLink:        nullableString(request.ExternalLink),
		RequiredAttachments: nullableString(request.RequiredAttachments),
		CreatedByID:         user.ID,
	}

	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		for _, year := range years {
			if err := tx.Create(&models.TaskTarget{TaskID: task.ID, AcademicYear: year}).Error; err != nil {
				return err
			}
		}
		var members []models.Member
		if err := tx.Where("academic_year IN ? AND status = ? AND deleted_at IS NULL", years, "active").Find(&members).Error; err != nil {
			return err
		}
		for i := range members {
			if err := tx.Create(&models.TaskSubmissionStatus{
				TaskID:   task.ID,
				MemberID: members[i].ID,
				Status:   "pending",
			}).Error; err != nil {
				return err
			}
			if err := s.createNotification(tx, members[i].UserID, "New task assigned", task.Title, "task", "Task", &task.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.Log.Error("task creation failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to create task")
		return
	}

	s.audit(c, "task_created", "Task", &task.ID, gin.H{
		"title":     task.Title,
		"task_type": task.TaskType,
		"deadline":  dateValue(task.Deadline),
	})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Task created and published",
		"task":    s.serializeTaskFull(&task, nil),
	})
}

func (s *Server) updateTask(c *gin.Context) {
	task, ok := s.findTask(c)
	if !ok {
		return
	}
	var request taskUpdateRequest
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
		if task.Title != title {
			changes["title"] = gin.H{"from": task.Title, "to": title}
			task.Title = title
		}
	}
	if request.Description != nil {
		description := strings.TrimSpace(*request.Description)
		if description != "" && stringOrEmpty(task.Description) != description {
			changes["description"] = gin.H{"from": stringOrEmpty(task.Description), "to": description}
			task.Description = &description
		}
	}
	if request.Deadline != nil && strings.TrimSpace(*request.Deadline) != "" {
		deadline, parsed := parseDate(*request.Deadline)
		if !parsed {
			fail(c, http.StatusUnprocessableEntity, "Invalid deadline")
			return
		}
		if dateValue(task.Deadline) != dateValue(deadline) {
			changes["deadline"] = gin.H{"from": dateValue(task.Deadline), "to": dateValue(deadline)}
			task.Deadline = deadline
		}
	}
	if request.Instructions != nil {
		instructions := strings.TrimSpace(*request.Instructions)
		if instructions != "" && stringOrEmpty(task.Instructions) != instructions {
			changes["instructions"] = gin.H{"from": stringOrEmpty(task.Instructions), "to": instructions}
			task.Instructions = &instructions
		}
	}
	if request.ExternalLink != nil {
		link := strings.TrimSpace(*request.ExternalLink)
		if link != "" {
			if !strings.HasPrefix(strings.ToLower(link), "http://") && !strings.HasPrefix(strings.ToLower(link), "https://") {
				fail(c, http.StatusUnprocessableEntity, "External link must be a valid URL")
				return
			}
			if stringOrEmpty(task.ExternalLink) != link {
				changes["external_link"] = gin.H{"from": stringOrEmpty(task.ExternalLink), "to": link}
				task.ExternalLink = &link
			}
		}
	}
	if request.RequiredAttachments != nil {
		required := strings.TrimSpace(*request.RequiredAttachments)
		if required != "" && stringOrEmpty(task.RequiredAttachments) != required {
			changes["required_attachments"] = gin.H{"from": stringOrEmpty(task.RequiredAttachments), "to": required}
			task.RequiredAttachments = &required
		}
	}

	newYears := []int(nil)
	if request.TargetAcademicYears != nil && len(*request.TargetAcademicYears) > 0 {
		seen := map[int]struct{}{}
		for _, year := range *request.TargetAcademicYears {
			if year < 1 || year > 8 {
				fail(c, http.StatusUnprocessableEntity, "Academic year must be between 1 and 8")
				return
			}
			if _, ok := seen[year]; !ok {
				seen[year] = struct{}{}
				newYears = append(newYears, year)
			}
		}
	}

	if len(changes) == 0 && newYears == nil {
		fail(c, http.StatusUnprocessableEntity, "No changes provided")
		return
	}

	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(task).Error; err != nil {
			return err
		}
		if newYears != nil {
			current := s.taskTargetYears(task.ID)
			same := len(current) == len(newYears)
			if same {
				for i := range current {
					if current[i] != newYears[i] {
						same = false
						break
					}
				}
			}
			if !same {
				changes["target_academic_years"] = gin.H{"from": current, "to": newYears}
				if err := tx.Where("task_id = ?", task.ID).Delete(&models.TaskTarget{}).Error; err != nil {
					return err
				}
				for _, year := range newYears {
					if err := tx.Create(&models.TaskTarget{TaskID: task.ID, AcademicYear: year}).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		s.Log.Error("task update failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to update task")
		return
	}

	s.audit(c, "task_updated", "Task", &task.ID, gin.H{"changes": changes})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Task updated",
		"task":    s.serializeTaskFull(task, nil),
	})
}

func (s *Server) deleteTask(c *gin.Context) {
	task, ok := s.findTask(c)
	if !ok {
		return
	}
	now := time.Now()
	if err := s.DB.WithContext(c.Request.Context()).Model(task).Update("deleted_at", now).Error; err != nil {
		s.Log.Error("task delete failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to delete task")
		return
	}
	task.DeletedAt = &now
	s.audit(c, "task_deleted", "Task", &task.ID, gin.H{"title": task.Title, "task_type": task.TaskType})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Task deleted"})
}

func (s *Server) submitTask(c *gin.Context) {
	task, ok := s.findTask(c)
	if !ok {
		return
	}
	member := currentMember(c)
	if member == nil {
		fail(c, http.StatusForbidden, "Member profile required")
		return
	}
	if member.Status != "active" {
		fail(c, http.StatusForbidden, "Member not active")
		return
	}
	var submission models.TaskSubmissionStatus
	if err := s.DB.WithContext(c.Request.Context()).
		Where("task_id = ? AND member_id = ?", task.ID, member.ID).First(&submission).Error; err != nil {
		fail(c, http.StatusNotFound, "No submission record found for this task")
		return
	}
	if submission.Status == "verified" {
		fail(c, http.StatusConflict, "This submission has already been verified")
		return
	}
	now := time.Now()
	if err := s.DB.WithContext(c.Request.Context()).Model(&submission).Updates(map[string]any{
		"status":       "submitted",
		"submitted_at": now,
	}).Error; err != nil {
		s.Log.Error("task submit failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to submit task")
		return
	}
	submission.Status = "submitted"
	submission.SubmittedAt = &now

	s.audit(c, "task_submitted", "TaskSubmissionStatus", &submission.ID, gin.H{
		"task_title":      task.Title,
		"member_login_id": member.LoginID,
	})
	s.notifyAdmins(c.Request.Context(), "Task submission",
		member.Name+" submitted '"+task.Title+"'", "submission_verification",
		"TaskSubmissionStatus", &submission.ID)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Task submitted successfully",
		"submission": s.serializeSubmission(&submission),
	})
}

func (s *Server) verifyTaskSubmission(c *gin.Context) {
	task, ok := s.findTask(c)
	if !ok {
		return
	}
	var request taskVerifyRequest
	if !bindJSON(c, &request) {
		return
	}
	if request.MemberID <= 0 {
		fail(c, http.StatusNotFound, "Member not found")
		return
	}
	var member models.Member
	if err := s.DB.WithContext(c.Request.Context()).First(&member, request.MemberID).Error; err != nil {
		fail(c, http.StatusNotFound, "Member not found")
		return
	}
	decision := strings.ToLower(strings.TrimSpace(request.Decision))
	if decision == "" {
		decision = strings.ToLower(strings.TrimSpace(request.VerificationAction))
	}
	if decision != "verify" && decision != "reject" {
		fail(c, http.StatusUnprocessableEntity, "Action must be 'verify' or 'reject'")
		return
	}
	var submission models.TaskSubmissionStatus
	if err := s.DB.WithContext(c.Request.Context()).
		Where("task_id = ? AND member_id = ?", task.ID, member.ID).First(&submission).Error; err != nil {
		fail(c, http.StatusNotFound, "Submission not found")
		return
	}
	if submission.Status != "submitted" {
		fail(c, http.StatusUnprocessableEntity, "Submission is not in pending state")
		return
	}
	user := currentUser(c)
	now := time.Now()
	status := "verified"
	if decision == "reject" {
		status = "rejected"
	}
	updates := map[string]any{
		"status":      status,
		"verified_at": now,
	}
	if user != nil {
		updates["verified_by_id"] = user.ID
	}
	if err := s.DB.WithContext(c.Request.Context()).Model(&submission).Updates(updates).Error; err != nil {
		s.Log.Error("task verify failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to update submission")
		return
	}
	submission.Status = status
	submission.VerifiedAt = &now
	if user != nil {
		submission.VerifiedByID = &user.ID
	}

	action := "task_verified"
	title := "Task verified"
	body := "Your submission for '" + task.Title + "' has been verified"
	message := "Submission verified"
	if decision == "reject" {
		action = "task_rejected"
		title = "Task rejected"
		body = "Your submission for '" + task.Title + "' was rejected"
		message = "Submission rejected"
	}
	s.audit(c, action, "TaskSubmissionStatus", &submission.ID, gin.H{
		"task_title":      task.Title,
		"member_login_id": member.LoginID,
	})
	s.notifyUser(c.Request.Context(), member.UserID, title, body, "submission_verified", "TaskSubmissionStatus", &submission.ID)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    message,
		"submission": s.serializeSubmission(&submission),
	})
}
