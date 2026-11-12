package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

type manualAttendanceRequest struct {
	MemberID        int64  `json:"member_id"`
	EntryDate       string `json:"entry_date"`
	EntryTime       string `json:"entry_time"`
	Source          string `json:"source"`
	DeviceReference string `json:"device_reference"`
}

func (s *Server) attendanceBaseQuery(c *gin.Context) (*gorm.DB, error) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceLog{})
	user := currentUser(c)
	if user == nil {
		return nil, errors.New("authentication required")
	}
	if user.Role != "admin" {
		member := currentMember(c)
		if member == nil {
			return nil, errors.New("member profile unavailable")
		}
		return query.Where("attendance_logs.member_id = ?", member.ID), nil
	}

	joined := false
	ensureJoin := func() {
		if !joined {
			query = query.Joins("JOIN members ON members.id = attendance_logs.member_id")
			joined = true
		}
	}

	if dateValue := strings.TrimSpace(c.Query("date")); dateValue != "" {
		parsed, ok := parseDate(dateValue)
		if !ok {
			return nil, errors.New("invalid date format")
		}
		query = query.Where("attendance_logs.entry_date = ?", startOfDay(parsed))
	}
	if memberID := strings.TrimSpace(c.Query("member_id")); memberID != "" {
		query = query.Where("attendance_logs.member_id = ?", memberID)
	}
	if year := strings.TrimSpace(c.Query("academic_year")); year != "" {
		ensureJoin()
		query = query.Where("members.academic_year = ?", year)
	}
	if branch := strings.TrimSpace(c.Query("branch")); branch != "" {
		ensureJoin()
		query = query.Where("members.branch = ?", branch)
	}
	if source := strings.TrimSpace(c.Query("source")); source != "" {
		query = query.Where("attendance_logs.source = ?", source)
	}
	return query, nil
}

func (s *Server) listAttendance(c *gin.Context) {
	query, err := s.attendanceBaseQuery(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load attendance")
		return
	}
	var records []models.AttendanceLog
	if err := query.Order("attendance_logs.entry_date DESC, attendance_logs.entry_time DESC").
		Offset(offsetFor(page, perPage)).Limit(perPage).Find(&records).Error; err != nil {
		s.Log.Error("attendance list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load attendance")
		return
	}

	members, createdBy := s.attendanceLookups(c, records)
	items := make([]gin.H, 0, len(records))
	for i := range records {
		items = append(items, serializeAttendance(&records[i], members[records[i].MemberID], createdBy[records[i].ID]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"attendance": items,
		"pagination": makePagination(page, perPage, total),
	})
}

func (s *Server) attendanceLookups(c *gin.Context, records []models.AttendanceLog) (map[int64]*models.Member, map[int64]string) {
	memberIDs := make([]int64, 0, len(records))
	userIDs := make([]int64, 0, len(records))
	seenMembers := map[int64]struct{}{}
	seenUsers := map[int64]struct{}{}
	for i := range records {
		if _, ok := seenMembers[records[i].MemberID]; !ok {
			seenMembers[records[i].MemberID] = struct{}{}
			memberIDs = append(memberIDs, records[i].MemberID)
		}
		if records[i].CreatedByID != nil {
			id := *records[i].CreatedByID
			if _, ok := seenUsers[id]; !ok {
				seenUsers[id] = struct{}{}
				userIDs = append(userIDs, id)
			}
		}
	}
	members := map[int64]*models.Member{}
	if len(memberIDs) > 0 {
		var list []models.Member
		s.DB.WithContext(c.Request.Context()).Where("id IN ?", memberIDs).Find(&list)
		for i := range list {
			members[list[i].ID] = &list[i]
		}
	}
	emails := map[int64]string{}
	if len(userIDs) > 0 {
		var users []models.User
		s.DB.WithContext(c.Request.Context()).Where("id IN ?", userIDs).Find(&users)
		for i := range users {
			emails[users[i].ID] = users[i].Email
		}
	}
	createdBy := map[int64]string{}
	for i := range records {
		if records[i].CreatedByID != nil {
			createdBy[records[i].ID] = emails[*records[i].CreatedByID]
		}
	}
	return members, createdBy
}

func serializeAttendance(record *models.AttendanceLog, member *models.Member, createdBy string) gin.H {
	memberData := gin.H{"login_id": "", "name": "", "usn": nil, "branch": ""}
	if member != nil {
		memberData = gin.H{
			"login_id": member.LoginID,
			"name":     member.Name,
			"usn":      member.USN,
			"branch":   member.Branch,
		}
	}
	result := gin.H{
		"id":               record.ID,
		"member_id":        record.MemberID,
		"member":           memberData,
		"entry_date":       dateValue(record.EntryDate),
		"entry_time":       record.EntryTime,
		"exit_time":        record.ExitTime,
		"source":           record.Source,
		"device_reference": record.DeviceReference,
		"created_by":       nil,
		"created_at":       record.CreatedAt,
	}
	if createdBy != "" {
		result["created_by"] = createdBy
	}
	return result
}

func attendanceDurationMinutes(record *models.AttendanceLog) int64 {
	end := time.Now()
	if record.ExitTime != nil {
		end = *record.ExitTime
	}
	minutes := int64(end.Sub(record.EntryTime).Minutes())
	if minutes < 0 {
		return 0
	}
	return minutes
}

func (s *Server) showAttendance(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var record models.AttendanceLog
	if err := s.DB.WithContext(c.Request.Context()).First(&record, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	user := currentUser(c)
	if user != nil && user.Role != "admin" {
		member := currentMember(c)
		if member == nil || record.MemberID != member.ID {
			fail(c, http.StatusForbidden, "Access denied")
			return
		}
	}
	var member models.Member
	s.DB.WithContext(c.Request.Context()).First(&member, record.MemberID)
	createdBy := ""
	if record.CreatedByID != nil {
		var createdUser models.User
		if err := s.DB.WithContext(c.Request.Context()).First(&createdUser, *record.CreatedByID).Error; err == nil {
			createdBy = createdUser.Email
		}
	}
	result := serializeAttendance(&record, &member, createdBy)
	result["updated_at"] = record.UpdatedAt
	result["duration_minutes"] = attendanceDurationMinutes(&record)
	c.JSON(http.StatusOK, gin.H{"success": true, "attendance": result})
}

func (s *Server) todayAttendance(c *gin.Context) {
	today := startOfDay(time.Now())
	query, err := s.attendanceBaseQuery(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	query = query.Where("attendance_logs.entry_date = ?", today)

	var records []models.AttendanceLog
	if err := query.Order("attendance_logs.entry_time DESC").Limit(50).Find(&records).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load attendance")
		return
	}
	members, createdBy := s.attendanceLookups(c, records)
	items := make([]gin.H, 0, len(records))
	for i := range records {
		items = append(items, serializeAttendance(&records[i], members[records[i].MemberID], createdBy[records[i].ID]))
	}

	summaryQuery, err := s.attendanceBaseQuery(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	summaryQuery = summaryQuery.Where("attendance_logs.entry_date = ?", today)
	var totalPresent int64
	summaryQuery.Where("attendance_logs.exit_time IS NULL").Count(&totalPresent)
	uniqueQuery, _ := s.attendanceBaseQuery(c)
	uniqueQuery = uniqueQuery.Where("attendance_logs.entry_date = ?", today)
	var totalVisited int64
	uniqueQuery.Distinct("attendance_logs.member_id").Count(&totalVisited)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"today":   items,
		"summary": gin.H{
			"total_present": totalPresent,
			"total_visited": totalVisited,
			"date":          dateValue(today),
		},
	})
}

func (s *Server) attendanceSummary(c *gin.Context) {
	query, err := s.attendanceBaseQuery(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	dateFrom, dateTo := attendanceDateRange(c)
	query = query.Where("attendance_logs.entry_date BETWEEN ? AND ?", dateFrom, dateTo)

	var loginEvents, logoutEvents, uniqueMembers int64
	query.Where("attendance_logs.exit_time IS NULL").Count(&loginEvents)
	logoutQuery, _ := s.attendanceBaseQuery(c)
	logoutQuery = logoutQuery.Where("attendance_logs.entry_date BETWEEN ? AND ? AND attendance_logs.exit_time IS NOT NULL", dateFrom, dateTo)
	logoutQuery.Count(&logoutEvents)
	uniqueQuery, _ := s.attendanceBaseQuery(c)
	uniqueQuery = uniqueQuery.Where("attendance_logs.entry_date BETWEEN ? AND ?", dateFrom, dateTo)
	uniqueQuery.Distinct("attendance_logs.member_id").Count(&uniqueMembers)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"summary": gin.H{
			"total_login_events":  loginEvents,
			"total_logout_events": logoutEvents,
			"unique_members":      uniqueMembers,
			"date_range": gin.H{
				"from": dateValue(dateFrom),
				"to":   dateValue(dateTo),
			},
		},
	})
}

type dailyAttendanceRow struct {
	EntryDate     time.Time `gorm:"column:entry_date"`
	TotalEvents   int64     `gorm:"column:total_events"`
	Logins        int64     `gorm:"column:logins"`
	Logouts       int64     `gorm:"column:logouts"`
	UniqueMembers int64     `gorm:"column:unique_members"`
}

func (s *Server) dailyAttendanceSummary(c *gin.Context) {
	query, err := s.attendanceBaseQuery(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	dateFrom, dateTo := attendanceDateRange(c)
	query = query.Where("attendance_logs.entry_date BETWEEN ? AND ?", dateFrom, dateTo)

	var rows []dailyAttendanceRow
	err = query.Select(`
		attendance_logs.entry_date AS entry_date,
		COUNT(*) AS total_events,
		COUNT(CASE WHEN attendance_logs.exit_time IS NULL THEN 1 END) AS logins,
		COUNT(CASE WHEN attendance_logs.exit_time IS NOT NULL THEN 1 END) AS logouts,
		COUNT(DISTINCT attendance_logs.member_id) AS unique_members
	`).Group("attendance_logs.entry_date").Order("attendance_logs.entry_date DESC").Scan(&rows).Error
	if err != nil {
		s.Log.Error("daily attendance summary failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load attendance summary")
		return
	}

	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{
			"date":           dateValue(row.EntryDate),
			"total_events":   row.TotalEvents,
			"logins":         row.Logins,
			"logouts":        row.Logouts,
			"unique_members": row.UniqueMembers,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "daily_summary": items})
}

func (s *Server) attendanceOccupancy(c *gin.Context) {
	user := currentUser(c)
	if user == nil || user.Role != "admin" {
		fail(c, http.StatusForbidden, "Administrator access required")
		return
	}
	today := startOfDay(time.Now())
	var current int64
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceLog{}).
		Where("entry_date = ? AND exit_time IS NULL", today).Count(&current).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load occupancy")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"occupancy": gin.H{
			"current":   current,
			"timestamp": time.Now().UTC(),
			"date":      dateValue(today),
		},
	})
}

func (s *Server) attendanceAnalytics(c *gin.Context) {
	user := currentUser(c)
	if user == nil || user.Role != "admin" {
		fail(c, http.StatusForbidden, "Administrator access required")
		return
	}
	query, err := s.attendanceBaseQuery(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	dateFrom, dateTo := attendanceDateRange(c)
	query = query.Where("attendance_logs.entry_date BETWEEN ? AND ?", dateFrom, dateTo)

	var totalEvents, loginEvents, logoutEvents, uniqueMembers int64
	query.Count(&totalEvents)
	loginQuery, _ := s.attendanceBaseQuery(c)
	loginQuery = loginQuery.Where("attendance_logs.entry_date BETWEEN ? AND ? AND attendance_logs.exit_time IS NULL", dateFrom, dateTo)
	loginQuery.Count(&loginEvents)
	logoutQuery, _ := s.attendanceBaseQuery(c)
	logoutQuery = logoutQuery.Where("attendance_logs.entry_date BETWEEN ? AND ? AND attendance_logs.exit_time IS NOT NULL", dateFrom, dateTo)
	logoutQuery.Count(&logoutEvents)
	uniqueQuery, _ := s.attendanceBaseQuery(c)
	uniqueQuery = uniqueQuery.Where("attendance_logs.entry_date BETWEEN ? AND ?", dateFrom, dateTo)
	uniqueQuery.Distinct("attendance_logs.member_id").Count(&uniqueMembers)

	days := int64(dateTo.Sub(dateFrom).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	average := float64(totalEvents) / float64(days)

	type branchRow struct {
		Branch        string `gorm:"column:branch"`
		Events        int64  `gorm:"column:events"`
		UniqueMembers int64  `gorm:"column:unique_members"`
	}
	var branches []branchRow
	s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceLog{}).
		Select("members.branch AS branch, COUNT(*) AS events, COUNT(DISTINCT attendance_logs.member_id) AS unique_members").
		Joins("JOIN members ON members.id = attendance_logs.member_id").
		Where("attendance_logs.entry_date BETWEEN ? AND ?", dateFrom, dateTo).
		Group("members.branch").Order("events DESC").Scan(&branches)

	type yearRow struct {
		AcademicYear  int   `gorm:"column:academic_year"`
		Events        int64 `gorm:"column:events"`
		UniqueMembers int64 `gorm:"column:unique_members"`
	}
	var years []yearRow
	s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceLog{}).
		Select("members.academic_year AS academic_year, COUNT(*) AS events, COUNT(DISTINCT attendance_logs.member_id) AS unique_members").
		Joins("JOIN members ON members.id = attendance_logs.member_id").
		Where("attendance_logs.entry_date BETWEEN ? AND ?", dateFrom, dateTo).
		Group("members.academic_year").Order("events DESC").Scan(&years)

	branchItems := make([]gin.H, 0, len(branches))
	for _, row := range branches {
		branchItems = append(branchItems, gin.H{"branch": row.Branch, "events": row.Events, "unique_members": row.UniqueMembers})
	}
	yearItems := make([]gin.H, 0, len(years))
	for _, row := range years {
		yearItems = append(yearItems, gin.H{"academic_year": row.AcademicYear, "events": row.Events, "unique_members": row.UniqueMembers})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"analytics": gin.H{
			"date_range": gin.H{"from": dateValue(dateFrom), "to": dateValue(dateTo)},
			"total_events":            totalEvents,
			"login_events":            loginEvents,
			"logout_events":           logoutEvents,
			"unique_members":          uniqueMembers,
			"average_daily_attendance": roundTo(average, 2),
			"by_branch":               branchItems,
			"by_academic_year":        yearItems,
		},
	})
}

func (s *Server) createAttendance(c *gin.Context) {
	user := currentUser(c)
	if user == nil || user.Role != "admin" {
		fail(c, http.StatusForbidden, "Administrator access required")
		return
	}
	var request manualAttendanceRequest
	if !bindJSON(c, &request) {
		return
	}
	if request.MemberID <= 0 {
		fail(c, http.StatusUnprocessableEntity, "Member is required")
		return
	}

	var member models.Member
	err := s.DB.WithContext(c.Request.Context()).
		Where("id = ? AND status = ? AND deleted_at IS NULL", request.MemberID, "active").
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, "Member not found or inactive")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "Failed to record attendance")
		return
	}

	entryDate := startOfDay(time.Now())
	if strings.TrimSpace(request.EntryDate) != "" {
		parsed, ok := parseDate(request.EntryDate)
		if !ok {
			fail(c, http.StatusUnprocessableEntity, "Invalid date or time format")
			return
		}
		entryDate = startOfDay(parsed)
	}
	entryTime := time.Now()
	if strings.TrimSpace(request.EntryTime) != "" {
		parsed, ok := parseDate(request.EntryTime)
		if !ok {
			fail(c, http.StatusUnprocessableEntity, "Invalid date or time format")
			return
		}
		entryTime = parsed
	}

	if strings.TrimSpace(request.EntryDate) == "" && strings.TrimSpace(request.EntryTime) == "" {
		var existing int64
		s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceLog{}).
			Where("member_id = ? AND entry_date = ? AND exit_time IS NULL", member.ID, entryDate).
			Count(&existing)
		if existing > 0 {
			fail(c, http.StatusConflict, "Member already has an active attendance record for today")
			return
		}
	}
	if startOfDay(entryTime) != entryDate {
		fail(c, http.StatusUnprocessableEntity, "Entry time date must match entry date")
		return
	}

	source := strings.TrimSpace(request.Source)
	if source == "" {
		source = "manual"
	}
	record := models.AttendanceLog{
		MemberID:        member.ID,
		EntryDate:       entryDate,
		EntryTime:       entryTime,
		Source:          source,
		DeviceReference: nullableString(request.DeviceReference),
		CreatedByID:     &user.ID,
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&record).Error; err != nil {
		s.Log.Error("manual attendance creation failed", "error", err)
		fail(c, http.StatusInternalServerError, "Failed to record attendance")
		return
	}

	s.audit(c, "attendance_recorded", "AttendanceLog", &record.ID, gin.H{
		"member_login_id": member.LoginID,
		"member_name":     member.Name,
		"source":          source,
	})
	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"message":    "Attendance recorded successfully",
		"attendance": serializeAttendance(&record, &member, user.Email),
	})
}

func (s *Server) recordAttendanceLogout(c *gin.Context) {
	user := currentUser(c)
	if user == nil || user.Role != "admin" {
		fail(c, http.StatusForbidden, "Administrator access required")
		return
	}
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var record models.AttendanceLog
	if err := s.DB.WithContext(c.Request.Context()).First(&record, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Record not found")
		return
	}
	if record.ExitTime != nil {
		fail(c, http.StatusUnprocessableEntity, "This attendance record already has a logout time")
		return
	}
	now := time.Now().UTC()
	if err := s.DB.WithContext(c.Request.Context()).Model(&models.AttendanceLog{}).Where("id = ?", record.ID).Update("exit_time", now).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to record logout")
		return
	}
	record.ExitTime = &now

	var member models.Member
	s.DB.WithContext(c.Request.Context()).First(&member, record.MemberID)
	s.audit(c, "attendance_logout_recorded", "AttendanceLog", &record.ID, gin.H{
		"member_login_id": member.LoginID,
		"member_name":     member.Name,
		"logout_time":     now,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Logout recorded successfully",
		"attendance": serializeAttendance(&record, &member, user.Email),
	})
}

func attendanceDateRange(c *gin.Context) (time.Time, time.Time) {
	now := time.Now()
	dateTo := startOfDay(now)
	dateFrom := startOfDay(now.AddDate(0, 0, -30))
	if parsed, ok := parseDate(c.Query("date_from")); ok {
		dateFrom = startOfDay(parsed)
	}
	if parsed, ok := parseDate(c.Query("date_to")); ok {
		dateTo = startOfDay(parsed)
	}
	if dateFrom.After(dateTo) {
		dateFrom, dateTo = dateTo, dateFrom
	}
	if c.Query("date_from") == "" && c.Query("date_to") == "" {
		if c.FullPath() == "/api/v1/attendance/daily-summary" {
			dateFrom = startOfDay(now.AddDate(0, 0, -7))
		}
	}
	return dateFrom, dateTo
}

func roundTo(value float64, places int) float64 {
	multiplier := 1.0
	for i := 0; i < places; i++ {
		multiplier *= 10
	}
	rounded := float64(int64(value*multiplier+0.5)) / multiplier
	return rounded
}
