package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

func (s *Server) registerReturnRoutes(api *gin.RouterGroup) {
	returns := api.Group("/returns")
	returns.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted())
	returns.GET("/admin/requests", s.RequireAdmin(), s.adminListReturnRequests)
	returns.GET("/requests", s.listReturnRequests)
	returns.POST("/request", s.createReturnRequest)
	returns.POST("/:id/confirm", s.RequireAdmin(), s.confirmReturnRequest)
	returns.POST("/:id/reject", s.RequireAdmin(), s.rejectReturnRequest)
}

func (s *Server) returnItem(borrowRecordID int64) *models.InventoryItem {
	var borrow models.BorrowRecord
	if err := s.DB.First(&borrow, borrowRecordID).Error; err != nil {
		return nil
	}
	var item models.InventoryItem
	if err := s.DB.First(&item, borrow.InventoryItemID).Error; err != nil {
		return nil
	}
	return &item
}

func (s *Server) serializeReturnRequest(request *models.ReturnRequest) gin.H {
	item := s.returnItem(request.BorrowRecordID)
	itemID, name, category := int64(0), "", ""
	if item != nil {
		itemID, name, category = item.ID, item.Name, item.Category
	}
	return gin.H{
		"id":               request.ID,
		"borrow_record_id": request.BorrowRecordID,
		"item": gin.H{
			"id":       itemID,
			"name":     name,
			"category": category,
		},
		"status":         request.Status,
		"condition_note": request.ConditionNote,
		"created_at":     request.CreatedAt,
		"confirmed_at":   request.ConfirmedAt,
	}
}

func (s *Server) serializeReturnRequestFull(request *models.ReturnRequest) gin.H {
	result := s.serializeReturnRequest(request)
	var confirmedBy any
	if request.ConfirmedByID != nil {
		var user models.User
		if err := s.DB.First(&user, *request.ConfirmedByID).Error; err == nil {
			confirmedBy = user.Email
		}
	}
	var borrow models.BorrowRecord
	s.DB.First(&borrow, request.BorrowRecordID)
	result["confirmed_by"] = confirmedBy
	result["borrow_date"] = borrow.BorrowedAt
	result["due_date"] = dateValue(borrow.DueDate)
	return result
}

func (s *Server) serializeReturnRequestAdmin(request *models.ReturnRequest, member *models.Member, borrow *models.BorrowRecord, item *models.InventoryItem) gin.H {
	var confirmedBy any
	if request.ConfirmedByID != nil {
		var user models.User
		if err := s.DB.First(&user, *request.ConfirmedByID).Error; err == nil {
			confirmedBy = user.Email
		}
	}
	memberName, loginID, email := "", "", ""
	if member != nil {
		memberName, loginID, email = member.Name, member.LoginID, member.Email
	}
	borrowStatus := ""
	borrowQuantity := 0
	borrowDate := time.Time{}
	dueDate := ""
	if borrow != nil {
		borrowStatus = borrow.Status
		borrowQuantity = borrow.Quantity
		borrowDate = borrow.BorrowedAt
		dueDate = dateValue(borrow.DueDate)
	}
	itemID, itemName, itemCategory := int64(0), "", ""
	if item != nil {
		itemID, itemName, itemCategory = item.ID, item.Name, item.Category
	}
	return gin.H{
		"id": request.ID,
		"member": gin.H{
			"id":       request.MemberID,
			"name":     memberName,
			"login_id": loginID,
			"email":    email,
		},
		"borrow_record": gin.H{
			"id":          request.BorrowRecordID,
			"borrowed_at": borrowDate,
			"due_date":    dueDate,
			"status":      borrowStatus,
			"quantity":    borrowQuantity,
		},
		"item": gin.H{
			"id":       itemID,
			"name":     itemName,
			"category": itemCategory,
		},
		"status":         request.Status,
		"condition_note": request.ConditionNote,
		"confirmed_by":   confirmedBy,
		"created_at":     request.CreatedAt,
		"confirmed_at":   request.ConfirmedAt,
	}
}

func (s *Server) listReturnRequests(c *gin.Context) {
	member := currentMember(c)
	if member == nil || member.Status != "active" {
		fail(c, http.StatusForbidden, "Active member account required")
		return
	}
	query := s.DB.WithContext(c.Request.Context()).Model(&models.ReturnRequest{}).
		Where("member_id = ?", member.ID)
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load return requests")
		return
	}
	var requests []models.ReturnRequest
	if err := query.Order("created_at DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&requests).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load return requests")
		return
	}
	items := make([]gin.H, 0, len(requests))
	for i := range requests {
		items = append(items, s.serializeReturnRequest(&requests[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"requests":   items,
		"pagination": gin.H{"page": page, "per_page": perPage, "total_count": total},
	})
}

func (s *Server) createReturnRequest(c *gin.Context) {
	member := currentMember(c)
	if member == nil || member.Status != "active" {
		fail(c, http.StatusForbidden, "Active member account required")
		return
	}
	var request struct {
		BorrowRecordID int64  `json:"borrow_record_id"`
		ConditionNote  string `json:"condition_note"`
	}
	if !bindJSON(c, &request) {
		return
	}
	var borrow models.BorrowRecord
	if err := s.DB.WithContext(c.Request.Context()).
		Where("id = ? AND member_id = ?", request.BorrowRecordID, member.ID).
		First(&borrow).Error; err != nil {
		fail(c, http.StatusNotFound, "Borrow record not found")
		return
	}
	if borrow.Status != "approved" && borrow.Status != "borrowed" {
		fail(c, http.StatusUnprocessableEntity, "Cannot request return for this borrow status")
		return
	}
	var existing int64
	s.DB.Model(&models.ReturnRequest{}).Where("borrow_record_id = ?", borrow.ID).Count(&existing)
	if existing > 0 {
		fail(c, http.StatusConflict, "Return request already exists for this borrow")
		return
	}
	returnRequest := models.ReturnRequest{
		BorrowRecordID: borrow.ID,
		MemberID:       member.ID,
		Status:         "pending",
		ConditionNote:  nullableString(request.ConditionNote),
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&returnRequest).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to submit return request")
		return
	}
	item := s.returnItem(borrow.ID)
	itemName, itemCategory := "", ""
	if item != nil {
		itemName, itemCategory = item.Name, item.Category
	}
	s.audit(c, "return_requested", "ReturnRequest", &returnRequest.ID, gin.H{
		"item_name": itemName, "item_category": itemCategory,
		"borrow_date": borrow.BorrowedAt, "due_date": dateValue(borrow.DueDate),
	})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Return request submitted",
		"request": s.serializeReturnRequest(&returnRequest),
	})
}

func (s *Server) adminListReturnRequests(c *gin.Context) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.ReturnRequest{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if pending := strings.TrimSpace(c.Query("pending")); pending == "true" {
		query = query.Where("status = ?", "pending")
	}
	if memberID := strings.TrimSpace(c.Query("member_id")); memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load return requests")
		return
	}
	var requests []models.ReturnRequest
	if err := query.Order("created_at DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&requests).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load return requests")
		return
	}
	items := make([]gin.H, 0, len(requests))
	for i := range requests {
		var member models.Member
		var borrow models.BorrowRecord
		var item models.InventoryItem
		s.DB.First(&member, requests[i].MemberID)
		if err := s.DB.First(&borrow, requests[i].BorrowRecordID).Error; err == nil {
			s.DB.First(&item, borrow.InventoryItemID)
		}
		items = append(items, s.serializeReturnRequestAdmin(&requests[i], &member, &borrow, &item))
	}
	var pendingCount int64
	s.DB.Model(&models.ReturnRequest{}).Where("status = ?", "pending").Count(&pendingCount)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"requests": items,
		"pagination": gin.H{
			"page": page, "per_page": perPage, "total_count": total, "pending_count": pendingCount,
		},
	})
}

func (s *Server) confirmReturnRequest(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var returnRequest models.ReturnRequest
	if err := s.DB.WithContext(c.Request.Context()).First(&returnRequest, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Return request not found")
		return
	}
	if returnRequest.Status != "pending" {
		fail(c, http.StatusUnprocessableEntity, "Only pending return requests can be confirmed")
		return
	}
	var request struct {
		ConditionNote string `json:"condition_note"`
	}
	_ = c.ShouldBindJSON(&request)
	user := currentUser(c)
	now := time.Now()
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var borrow models.BorrowRecord
		if err := tx.First(&borrow, returnRequest.BorrowRecordID).Error; err != nil {
			return err
		}
		var item models.InventoryItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, borrow.InventoryItemID).Error; err != nil {
			return err
		}
		if err := tx.Model(&borrow).Updates(map[string]any{
			"status":              "returned",
			"returned_at":         now,
			"condition_on_return": nullableString(request.ConditionNote),
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&returnRequest).Updates(map[string]any{
			"status":          "confirmed",
			"confirmed_by_id": user.ID,
			"confirmed_at":    now,
			"condition_note":  nullableString(request.ConditionNote),
		}).Error; err != nil {
			return err
		}
		restored := item.AvailableQuantity + borrow.Quantity
		if restored > item.TotalQuantity {
			restored = item.TotalQuantity
		}
		return tx.Model(&models.InventoryItem{}).Where("id = ?", item.ID).
			Update("available_quantity", restored).Error
	})
	if err != nil {
		s.Log.Error("return confirm failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to confirm return")
		return
	}
	returnRequest.Status = "confirmed"
	returnRequest.ConfirmedByID = &user.ID
	returnRequest.ConfirmedAt = &now
	returnRequest.ConditionNote = nullableString(request.ConditionNote)
	item := s.returnItem(returnRequest.BorrowRecordID)
	itemName := ""
	if item != nil {
		itemName = item.Name
	}
	s.audit(c, "return_confirmed", "ReturnRequest", &returnRequest.ID, gin.H{
		"item_name": itemName, "condition_note": returnRequest.ConditionNote,
	})
	s.notifyMemberUser(c, returnRequest.MemberID, "Return confirmed", "Your return of '"+itemName+"' has been confirmed", "return_confirmed", "ReturnRequest", &returnRequest.ID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Return confirmed. Item is now available in inventory",
		"request": s.serializeReturnRequestFull(&returnRequest),
	})
}

func (s *Server) rejectReturnRequest(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var returnRequest models.ReturnRequest
	if err := s.DB.WithContext(c.Request.Context()).First(&returnRequest, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Return request not found")
		return
	}
	if returnRequest.Status != "pending" {
		fail(c, http.StatusUnprocessableEntity, "Only pending return requests can be rejected")
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	if !bindJSON(c, &request) {
		return
	}
	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		fail(c, http.StatusUnprocessableEntity, "Rejection reason is required")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Model(&returnRequest).Updates(map[string]any{
		"status":         "rejected",
		"condition_note": reason,
	}).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to reject return request")
		return
	}
	returnRequest.Status = "rejected"
	returnRequest.ConditionNote = &reason
	var borrow models.BorrowRecord
	s.DB.First(&borrow, returnRequest.BorrowRecordID)
	item := s.returnItem(returnRequest.BorrowRecordID)
	itemName := ""
	if item != nil {
		itemName = item.Name
	}
	s.audit(c, "return_request_rejected", "ReturnRequest", &returnRequest.ID, gin.H{
		"item_name": itemName, "reason": reason,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Return request rejected",
		"request": s.serializeReturnRequest(&returnRequest),
	})
}
