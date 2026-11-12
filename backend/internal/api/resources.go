package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

func (s *Server) registerEventRoutes(api *gin.RouterGroup) {
	events := api.Group("/events")
	events.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted())
	events.GET("", s.listEvents)
	events.GET("/:id", s.showEvent)
	events.POST("", s.RequireAdmin(), s.createEvent)
	events.PATCH("/:id", s.RequireAdmin(), s.updateEvent)
	events.DELETE("/:id", s.RequireAdmin(), s.deleteEvent)
}

func validateEventType(value string) bool {
	switch value {
	case "workshop", "seminar", "talk", "hackathon", "training", "other":
		return true
	default:
		return false
	}
}

func clockShort(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 5 {
		return value[:5]
	}
	return value
}

func (s *Server) serializeEvent(event *models.Event) gin.H {
	return gin.H{
		"id":         event.ID,
		"title":      event.Title,
		"event_type": event.EventType,
		"date":       dateValue(event.Date),
		"start_time": clockShort(event.StartTime),
		"end_time":   clockShort(event.EndTime),
		"venue":      event.Venue,
		"created_by": s.taskCreatorName(event.CreatedByID),
		"created_at": event.CreatedAt,
	}
}

func (s *Server) serializeEventFull(event *models.Event) gin.H {
	links := []string(event.ExternalLinks)
	if links == nil {
		links = []string{}
	}
	isUpcoming := !event.Date.Before(startOfDay(time.Now()))
	result := s.serializeEvent(event)
	result["description"] = event.Description
	result["external_links"] = links
	result["is_upcoming"] = isUpcoming
	result["ends_at"] = dateValue(event.Date) + "T" + clockShort(event.EndTime) + ":00"
	return result
}

func (s *Server) listEvents(c *gin.Context) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.Event{}).Where("events.deleted_at IS NULL")
	today := startOfDay(time.Now())
	if upcoming := strings.TrimSpace(c.Query("upcoming")); upcoming == "true" {
		query = query.Where("events.date >= ?", today)
	}
	if past := strings.TrimSpace(c.Query("past")); past == "true" {
		query = query.Where("events.date < ?", today)
	}
	if date := strings.TrimSpace(c.Query("date")); date != "" {
		if parsed, ok := parseDate(date); ok {
			query = query.Where("events.date = ?", startOfDay(parsed))
		}
	}
	if eventType := strings.TrimSpace(c.Query("event_type")); eventType != "" {
		query = query.Where("events.event_type = ?", eventType)
	}
	if before := strings.TrimSpace(c.Query("before")); before != "" {
		if parsed, ok := parseDate(before); ok {
			query = query.Where("events.date < ?", startOfDay(parsed))
		}
	}
	if after := strings.TrimSpace(c.Query("after")); after != "" {
		if parsed, ok := parseDate(after); ok {
			query = query.Where("events.date > ?", startOfDay(parsed))
		}
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.Log.Error("event count failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load events")
		return
	}
	var events []models.Event
	if err := query.Order("events.date DESC, events.start_time DESC").
		Offset(offsetFor(page, perPage)).Limit(perPage).Find(&events).Error; err != nil {
		s.Log.Error("event list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load events")
		return
	}
	items := make([]gin.H, 0, len(events))
	for i := range events {
		items = append(items, s.serializeEvent(&events[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"events":     items,
		"pagination": makePagination(page, perPage, total),
	})
}

func (s *Server) findEvent(c *gin.Context) (*models.Event, bool) {
	id, ok := pathID(c, "id")
	if !ok {
		return nil, false
	}
	var event models.Event
	if err := s.DB.WithContext(c.Request.Context()).Where("id = ? AND deleted_at IS NULL", id).First(&event).Error; err != nil {
		fail(c, http.StatusNotFound, "Event not found")
		return nil, false
	}
	return &event, true
}

func (s *Server) showEvent(c *gin.Context) {
	event, ok := s.findEvent(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "event": s.serializeEventFull(event)})
}

type eventCreateRequest struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	EventType     string   `json:"event_type"`
	Date          string   `json:"date"`
	StartTime     string   `json:"start_time"`
	EndTime       string   `json:"end_time"`
	Venue         string   `json:"venue"`
	ExternalLinks []string `json:"external_links"`
}

func (s *Server) createEvent(c *gin.Context) {
	var request eventCreateRequest
	if !bindJSON(c, &request) {
		return
	}
	title := strings.TrimSpace(request.Title)
	eventType := strings.ToLower(strings.TrimSpace(request.EventType))
	if title == "" {
		fail(c, http.StatusUnprocessableEntity, "Title is required")
		return
	}
	if eventType == "" {
		fail(c, http.StatusUnprocessableEntity, "Event type is required")
		return
	}
	if !validateEventType(eventType) {
		fail(c, http.StatusUnprocessableEntity, "Invalid event type")
		return
	}
	date, ok := parseDate(request.Date)
	if !ok {
		fail(c, http.StatusUnprocessableEntity, "Date is required")
		return
	}
	startTime, ok := parseTimeOnDate(date, request.StartTime)
	if !ok {
		fail(c, http.StatusUnprocessableEntity, "Start time is required")
		return
	}
	endTime, ok := parseTimeOnDate(date, request.EndTime)
	if !ok {
		fail(c, http.StatusUnprocessableEntity, "End time is required")
		return
	}
	if !endTime.After(startTime) {
		fail(c, http.StatusUnprocessableEntity, "End time must be after start time")
		return
	}
	links := make([]string, 0, len(request.ExternalLinks))
	seen := map[string]struct{}{}
	for _, link := range request.ExternalLinks {
		link = strings.TrimSpace(link)
		if link == "" {
			continue
		}
		if _, exists := seen[link]; exists {
			continue
		}
		seen[link] = struct{}{}
		links = append(links, link)
	}
	event := models.Event{
		Title:         title,
		Description:   nullableString(request.Description),
		EventType:     eventType,
		Date:          startOfDay(date),
		StartTime:     startTime.Format("15:04:05"),
		EndTime:       endTime.Format("15:04:05"),
		Venue:         nullableString(request.Venue),
		ExternalLinks: pq.StringArray(links),
		CreatedByID:   currentUser(c).ID,
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&event).Error; err != nil {
		s.Log.Error("event create failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to create event")
		return
	}
	s.audit(c, "event_created", "Event", &event.ID, gin.H{
		"title": event.Title, "event_type": event.EventType, "date": dateValue(event.Date),
	})
	s.notifyAudience("New event", event.Title, "event", "Event", &event.ID)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Event created successfully",
		"event":   s.serializeEventFull(&event),
	})
}

func (s *Server) updateEvent(c *gin.Context) {
	event, ok := s.findEvent(c)
	if !ok {
		return
	}
	var request struct {
		Title         *string   `json:"title"`
		Description   *string   `json:"description"`
		EventType     *string   `json:"event_type"`
		Date          *string   `json:"date"`
		StartTime     *string   `json:"start_time"`
		EndTime       *string   `json:"end_time"`
		Venue         *string   `json:"venue"`
		ExternalLinks *[]string `json:"external_links"`
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
		if event.Title != title {
			changes["title"] = gin.H{"from": event.Title, "to": title}
			event.Title = title
		}
	}
	if request.Description != nil {
		description := strings.TrimSpace(*request.Description)
		if description != "" && stringOrEmpty(event.Description) != description {
			changes["description"] = gin.H{"from": stringOrEmpty(event.Description), "to": description}
			event.Description = &description
		}
	}
	if request.EventType != nil {
		eventType := strings.ToLower(strings.TrimSpace(*request.EventType))
		if eventType != "" && !validateEventType(eventType) {
			fail(c, http.StatusUnprocessableEntity, "Invalid event type")
			return
		}
		if eventType != "" && event.EventType != eventType {
			changes["event_type"] = gin.H{"from": event.EventType, "to": eventType}
			event.EventType = eventType
		}
	}
	if request.Date != nil && strings.TrimSpace(*request.Date) != "" {
		date, parsed := parseDate(*request.Date)
		if !parsed {
			fail(c, http.StatusUnprocessableEntity, "Invalid date")
			return
		}
		if dateValue(event.Date) != dateValue(date) {
			changes["date"] = gin.H{"from": dateValue(event.Date), "to": dateValue(date)}
			event.Date = startOfDay(date)
		}
	}
	if request.StartTime != nil && strings.TrimSpace(*request.StartTime) != "" {
		parsed, valid := parseTimeOnDate(event.Date, *request.StartTime)
		if !valid {
			fail(c, http.StatusUnprocessableEntity, "Invalid start time")
			return
		}
		value := parsed.Format("15:04:05")
		if event.StartTime != value {
			changes["start_time"] = gin.H{"from": event.StartTime, "to": value}
			event.StartTime = value
		}
	}
	if request.EndTime != nil && strings.TrimSpace(*request.EndTime) != "" {
		parsed, valid := parseTimeOnDate(event.Date, *request.EndTime)
		if !valid {
			fail(c, http.StatusUnprocessableEntity, "Invalid end time")
			return
		}
		value := parsed.Format("15:04:05")
		if event.EndTime != value {
			changes["end_time"] = gin.H{"from": event.EndTime, "to": value}
			event.EndTime = value
		}
	}
	if request.Venue != nil {
		venue := strings.TrimSpace(*request.Venue)
		if venue != "" && stringOrEmpty(event.Venue) != venue {
			changes["venue"] = gin.H{"from": stringOrEmpty(event.Venue), "to": venue}
			event.Venue = &venue
		}
	}
	if request.ExternalLinks != nil && len(*request.ExternalLinks) > 0 {
		links := make([]string, 0, len(*request.ExternalLinks))
		seen := map[string]struct{}{}
		for _, link := range *request.ExternalLinks {
			link = strings.TrimSpace(link)
			if link == "" {
				continue
			}
			if _, exists := seen[link]; exists {
				continue
			}
			seen[link] = struct{}{}
			links = append(links, link)
		}
		current := []string(event.ExternalLinks)
		if len(current) != len(links) {
			changes["external_links"] = gin.H{"from": current, "to": links}
			event.ExternalLinks = pq.StringArray(links)
		} else {
			for i := range current {
				if current[i] != links[i] {
					changes["external_links"] = gin.H{"from": current, "to": links}
					event.ExternalLinks = pq.StringArray(links)
					break
				}
			}
		}
	}
	if len(changes) == 0 {
		fail(c, http.StatusUnprocessableEntity, "No changes provided")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Save(event).Error; err != nil {
		s.Log.Error("event update failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to update event")
		return
	}
	s.audit(c, "event_updated", "Event", &event.ID, changes)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Event updated",
		"event":   s.serializeEventFull(event),
	})
}

func (s *Server) deleteEvent(c *gin.Context) {
	event, ok := s.findEvent(c)
	if !ok {
		return
	}
	now := time.Now()
	if err := s.DB.WithContext(c.Request.Context()).Model(event).Update("deleted_at", now).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to delete event")
		return
	}
	s.audit(c, "event_deleted", "Event", &event.ID, gin.H{"title": event.Title, "event_type": event.EventType})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Event deleted"})
}

