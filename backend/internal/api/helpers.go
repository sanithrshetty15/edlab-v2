package api

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

type paginationMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalCount int64 `json:"total_count"`
	TotalPages int   `json:"total_pages"`
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"success": false, "message": message})
}

func failValidation(c *gin.Context, message string, errors []string) {
	c.JSON(http.StatusUnprocessableEntity, gin.H{
		"success": false,
		"message": message,
		"errors":  errors,
	})
}

func bindJSON(c *gin.Context, target any) bool {
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(target); err != nil {
		fail(c, http.StatusUnprocessableEntity, "Invalid request body")
		return false
	}
	return true
}

func bindOptionalJSON(c *gin.Context, target any) bool {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return true
	}
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(target); err != nil && !errors.Is(err, io.EOF) {
		fail(c, http.StatusUnprocessableEntity, "Invalid request body")
		return false
	}
	return true
}

func pathID(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || value <= 0 {
		fail(c, http.StatusNotFound, "Record not found")
		return 0, false
	}
	return value, true
}

func pagination(c *gin.Context, defaultPerPage int) (page, perPage int) {
	page = 1
	if parsed, err := strconv.Atoi(c.Query("page")); err == nil && parsed > 0 {
		page = parsed
	}
	perPage = defaultPerPage
	if parsed, err := strconv.Atoi(c.Query("per_page")); err == nil && parsed > 0 {
		perPage = parsed
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func makePagination(page, perPage int, total int64) paginationMeta {
	totalPages := 0
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}
	return paginationMeta{
		Page:       page,
		PerPage:    perPage,
		TotalCount: total,
		TotalPages: totalPages,
	}
}

func offsetFor(page, perPage int) int {
	offset := (page - 1) * perPage
	if offset < 0 {
		return 0
	}
	return offset
}

func parseDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, appLocation()); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func appLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return time.Local
	}
	return location
}

func startOfDay(value time.Time) time.Time {
	local := value.In(appLocation())
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, appLocation())
}

func parseTimeOnDate(date time.Time, clock string) (time.Time, bool) {
	clock = strings.TrimSpace(clock)
	if clock == "" {
		return time.Time{}, false
	}
	layouts := []string{"15:04", "15:04:05", "3:04 PM", "3:04PM"}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, strings.ToUpper(clock), appLocation()); err == nil {
			day := date.In(appLocation())
			return time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, appLocation()), true
		}
	}
	return time.Time{}, false
}

func memberName(member *models.Member) string {
	if member == nil {
		return ""
	}
	return member.Name
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullableString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func serializeMemberSummary(member *models.Member) gin.H {
	return gin.H{
		"id":                   member.ID,
		"name":                 member.Name,
		"login_id":             member.LoginID,
		"usn":                  member.USN,
		"roll_number":          member.RollNumber,
		"academic_year":        member.AcademicYear,
		"branch":               member.Branch,
		"section":              member.Section,
		"email":                member.Email,
		"phone":                member.Phone,
		"phone_verified":       member.PhoneVerified,
		"status":               member.Status,
		"must_change_password": member.MustChangePassword,
		"created_at":           member.CreatedAt,
		"updated_at":           member.UpdatedAt,
	}
}

func serializeMember(member *models.Member) gin.H {
	phone := any(nil)
	if member.PhoneVerified {
		phone = member.Phone
	}
	interests := []string(member.Interests)
	if interests == nil {
		interests = []string{}
	}
	return gin.H{
		"id":                   member.ID,
		"name":                 member.Name,
		"usn":                  member.USN,
		"roll_number":          member.RollNumber,
		"login_id":             member.LoginID,
		"academic_year":        member.AcademicYear,
		"branch":               member.Branch,
		"section":              member.Section,
		"email":                member.Email,
		"phone":                phone,
		"phone_verified":       member.PhoneVerified,
		"github_url":           member.GitHubURL,
		"linkedin_url":         member.LinkedInURL,
		"portfolio_url":        member.PortfolioURL,
		"bio":                  member.Bio,
		"interests":            interests,
		"status":               member.Status,
		"must_change_password": member.MustChangePassword,
		"login_id_changed":     member.LoginIDChanged,
		"login_id_changed_at":  member.LoginIDChangedAt,
		"created_at":           member.CreatedAt,
	}
}

func serializeManagedMemberSummary(member *models.Member) gin.H {
	return gin.H{
		"id":                   member.ID,
		"name":                 member.Name,
		"login_id":             member.LoginID,
		"usn":                  member.USN,
		"roll_number":          member.RollNumber,
		"academic_year":        member.AcademicYear,
		"branch":               member.Branch,
		"section":              member.Section,
		"email":                member.Email,
		"phone":                member.Phone,
		"phone_verified":       member.PhoneVerified,
		"status":               member.Status,
		"must_change_password": member.MustChangePassword,
		"enrollment_status":    member.EnrollmentStatus,
		"created_at":           member.CreatedAt,
		"updated_at":           member.UpdatedAt,
	}
}

func serializeManagedMember(member *models.Member) gin.H {
	result := serializeManagedMemberSummary(member)
	result["github_url"] = member.GitHubURL
	result["linkedin_url"] = member.LinkedInURL
	result["portfolio_url"] = member.PortfolioURL
	result["bio"] = member.Bio
	result["interests"] = []string(member.Interests)
	result["login_id_changed"] = member.LoginIDChanged
	result["login_id_changed_at"] = member.LoginIDChangedAt
	return result
}

func serializeManagedMemberFull(member *models.Member, user *models.User) gin.H {
	result := serializeManagedMember(member)
	result["user"] = gin.H{
		"id":     user.ID,
		"email":  user.Email,
		"active": user.Active,
		"role":   user.Role,
	}
	return result
}

func serializeUser(user *models.User) gin.H {
	data := gin.H{
		"id":    user.ID,
		"email": user.Email,
		"role":  user.Role,
	}
	if user.Phone != nil {
		data["phone"] = *user.Phone
	}
	return data
}
