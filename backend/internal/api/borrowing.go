package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

var errBorrowStock = errors.New("insufficient stock")

func (s *Server) registerBorrowingRoutes(api *gin.RouterGroup) {
	borrowing := api.Group("/borrowing")
	borrowing.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted())
	borrowing.GET("/admin/requests", s.RequireAdmin(), s.adminListBorrowRequests)
	borrowing.GET("/requests", s.listBorrowRequests)
	borrowing.GET("/requests/:id", s.showBorrowRequest)
	borrowing.POST("/requests", s.createBorrowRequest)
	borrowing.POST("/requests/:id/approve", s.RequireAdmin(), s.approveBorrowRequest)
	borrowing.POST("/requests/:id/reject", s.RequireAdmin(), s.rejectBorrowRequest)
	borrowing.POST("/extension-request", s.createExtensionRequest)
}

func (s *Server) serializeBorrowRequest(borrow *models.BorrowRecord, item *models.InventoryItem) gin.H {
	name := ""
	category := ""
	itemID := borrow.InventoryItemID
	if item != nil {
		name = item.Name
		category = item.Category
		itemID = item.ID
	}
	return gin.H{
		"id": borrow.ID,
		"item": gin.H{
			"id":       itemID,
			"name":     name,
			"category": category,
		},
		"quantity":    borrow.Quantity,
		"status":      borrow.Status,
		"borrowed_at": borrow.BorrowedAt,
		"due_date":    dateValue(borrow.DueDate),
		"returned_at": borrow.ReturnedAt,
	}
}

func (s *Server) serializeExtension(extension *models.ExtensionRequest) gin.H {
	if extension == nil {
		return nil
	}
	return gin.H{
		"id":                    extension.ID,
		"borrow_record_id":      extension.BorrowRecordID,
		"status":                extension.Status,
		"requested_return_date": dateValue(extension.RequestedReturnDate),
		"reason":                extension.Reason,
		"created_at":            extension.CreatedAt,
	}
}

func (s *Server) serializeBorrowRequestFull(borrow *models.BorrowRecord, item *models.InventoryItem) gin.H {
	result := s.serializeBorrowRequest(borrow, item)
	var approvedBy any
	if borrow.ApprovedByID != nil {
		var user models.User
		if err := s.DB.First(&user, *borrow.ApprovedByID).Error; err == nil {
			approvedBy = user.Email
		}
	}
	var extension models.ExtensionRequest
	var extensionValue any
	if err := s.DB.Where("borrow_record_id = ?", borrow.ID).First(&extension).Error; err == nil {
		extensionValue = s.serializeExtension(&extension)
	}
	result["purpose"] = borrow.ApprovalReason
	result["approved_by"] = approvedBy
	result["approval_reason"] = borrow.ApprovalReason
	result["extension_request"] = extensionValue
	return result
}

func (s *Server) serializeBorrowRequestAdmin(borrow *models.BorrowRecord, member *models.Member, item *models.InventoryItem) gin.H {
	daysOverdue := 0
	today := startOfDay(time.Now())
	due := startOfDay(borrow.DueDate)
	if due.Before(today) {
		daysOverdue = int(today.Sub(due).Hours() / 24)
	}
	var approvedBy any
	if borrow.ApprovedByID != nil {
		var user models.User
		if err := s.DB.First(&user, *borrow.ApprovedByID).Error; err == nil {
			approvedBy = user.Email
		}
	}
	name, loginID, email := "", "", ""
	if member != nil {
		name = member.Name
		loginID = member.LoginID
		email = member.Email
	}
	itemName, itemCategory := "", ""
	if item != nil {
		itemName = item.Name
		itemCategory = item.Category
	}
	return gin.H{
		"id": borrow.ID,
		"member": gin.H{
			"id":       borrow.MemberID,
			"name":     name,
			"login_id": loginID,
			"email":    email,
		},
		"item": gin.H{
			"id":       borrow.InventoryItemID,
			"name":     itemName,
			"category": itemCategory,
		},
		"quantity":        borrow.Quantity,
		"status":          borrow.Status,
		"borrowed_at":     borrow.BorrowedAt,
		"due_date":        dateValue(borrow.DueDate),
		"days_overdue":    daysOverdue,
		"returned_at":     borrow.ReturnedAt,
		"approved_by":     approvedBy,
		"approval_reason": borrow.ApprovalReason,
	}
}

func (s *Server) listBorrowRequests(c *gin.Context) {
	member := currentMember(c)
	if member == nil || member.Status != "active" {
		fail(c, http.StatusForbidden, "Active member account required")
		return
	}
	query := s.DB.WithContext(c.Request.Context()).Model(&models.BorrowRecord{}).
		Where("member_id = ?", member.ID)
	if pending := strings.TrimSpace(c.Query("pending")); pending == "true" {
		query = query.Where("status IN ?", []string{"pending", "approved"})
	} else if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load borrow requests")
		return
	}
	var records []models.BorrowRecord
	if err := query.Order("created_at DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&records).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load borrow requests")
		return
	}
	items := make([]gin.H, 0, len(records))
	for i := range records {
		var item models.InventoryItem
		s.DB.First(&item, records[i].InventoryItemID)
		items = append(items, s.serializeBorrowRequest(&records[i], &item))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"requests":   items,
		"pagination": gin.H{"page": page, "per_page": perPage, "total_count": total},
	})
}

func (s *Server) showBorrowRequest(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	var borrow models.BorrowRecord
	if err := s.DB.WithContext(c.Request.Context()).First(&borrow, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Borrow record not found")
		return
	}
	user := currentUser(c)
	member := currentMember(c)
	if user.Role != "admin" && (member == nil || borrow.MemberID != member.ID) {
		fail(c, http.StatusForbidden, "Access denied")
		return
	}
	var item models.InventoryItem
	s.DB.First(&item, borrow.InventoryItemID)
	c.JSON(http.StatusOK, gin.H{"success": true, "request": s.serializeBorrowRequestFull(&borrow, &item)})
}

func randomKey() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buffer)
}

func (s *Server) createBorrowRequest(c *gin.Context) {
	member := currentMember(c)
	if member == nil || member.Status != "active" {
		fail(c, http.StatusForbidden, "Active member account required")
		return
	}
	var request struct {
		ItemID   int64  `json:"item_id"`
		Quantity int    `json:"quantity"`
		Purpose  string `json:"purpose"`
	}
	if !bindJSON(c, &request) {
		return
	}
	if request.ItemID <= 0 {
		fail(c, http.StatusUnprocessableEntity, "Item ID is required")
		return
	}
	if request.Quantity <= 0 {
		fail(c, http.StatusUnprocessableEntity, "Quantity must be greater than zero")
		return
	}
	var item models.InventoryItem
	if err := s.DB.WithContext(c.Request.Context()).Where("id = ? AND is_active = ? AND deleted_at IS NULL", request.ItemID, true).
		First(&item).Error; err != nil {
		fail(c, http.StatusNotFound, "Inventory item not found")
		return
	}
	if item.AvailableQuantity < request.Quantity {
		fail(c, http.StatusUnprocessableEntity, "Insufficient quantity available (requested: "+itoa(request.Quantity)+", available: "+itoa(item.AvailableQuantity)+")")
		return
	}
	var existing int64
	s.DB.Model(&models.BorrowRecord{}).
		Where("member_id = ? AND inventory_item_id = ? AND status <> ?", member.ID, request.ItemID, "returned").
		Count(&existing)
	if existing > 0 {
		fail(c, http.StatusConflict, "You already have an active borrow record for this item")
		return
	}
	dueDate := startOfDay(time.Now()).AddDate(0, 0, 14)
	var semester models.Semester
	if err := s.DB.WithContext(c.Request.Context()).Where("is_current = ?", true).
		Order("end_date DESC").First(&semester).Error; err == nil && !semester.EndDate.IsZero() {
		dueDate = startOfDay(semester.EndDate)
	}
	record := models.BorrowRecord{
		MemberID:        member.ID,
		InventoryItemID: item.ID,
		Quantity:        request.Quantity,
		Status:          "pending",
		BorrowedAt:      time.Now(),
		DueDate:         dueDate,
		IdempotencyKey:  ptrString(randomKey()),
		ApprovalReason:  nullableString(request.Purpose),
	}
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var locked models.InventoryItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, item.ID).Error; err != nil {
			return err
		}
		if !locked.IsActive || locked.DeletedAt != nil {
			return gorm.ErrRecordNotFound
		}
		if locked.AvailableQuantity < request.Quantity {
			return errBorrowStock
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		return tx.Model(&models.InventoryItem{}).Where("id = ?", locked.ID).
			UpdateColumn("available_quantity", gorm.Expr("available_quantity - ?", request.Quantity)).Error
	})
	if err != nil {
		if err == errBorrowStock {
			fail(c, http.StatusUnprocessableEntity, "Insufficient quantity available")
			return
		}
		if err == gorm.ErrRecordNotFound {
			fail(c, http.StatusNotFound, "Inventory item not found")
			return
		}
		s.Log.Error("borrow request failed", "error", err)
		fail(c, http.StatusInternalServerError, "Failed to submit borrow request")
		return
	}
	item.AvailableQuantity -= request.Quantity
	s.audit(c, "borrow_requested", "BorrowRecord", &record.ID, gin.H{
		"member_login_id": member.LoginID,
		"item_name":       item.Name,
		"item_category":   item.Category,
		"quantity":        record.Quantity,
		"due_date":        dateValue(record.DueDate),
	})
	s.notifyAdmins(c.Request.Context(), "Borrow request", member.Name+" requested '"+item.Name+"'", "inventory_approval", "BorrowRecord", &record.ID)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Borrow request submitted successfully",
		"request": s.serializeBorrowRequest(&record, &item),
	})
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	buffer := [20]byte{}
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		position--
		buffer[position] = '-'
	}
	return string(buffer[position:])
}

func (s *Server) findBorrowRequest(c *gin.Context) (*models.BorrowRecord, *models.InventoryItem, bool) {
	id, ok := pathID(c, "id")
	if !ok {
		return nil, nil, false
	}
	var borrow models.BorrowRecord
	if err := s.DB.WithContext(c.Request.Context()).First(&borrow, id).Error; err != nil {
		fail(c, http.StatusNotFound, "Borrow record not found")
		return nil, nil, false
	}
	var item models.InventoryItem
	s.DB.First(&item, borrow.InventoryItemID)
	return &borrow, &item, true
}

func (s *Server) approveBorrowRequest(c *gin.Context) {
	borrow, item, ok := s.findBorrowRequest(c)
	if !ok {
		return
	}
	if borrow.Status != "pending" {
		fail(c, http.StatusUnprocessableEntity, "Only pending requests can be approved")
		return
	}
	var request struct {
		ApprovalReason string `json:"approval_reason"`
	}
	_ = c.ShouldBindJSON(&request)
	user := currentUser(c)
	if err := s.DB.WithContext(c.Request.Context()).Model(borrow).Updates(map[string]any{
		"status":          "approved",
		"approved_by_id":  user.ID,
		"approval_reason": nullableString(request.ApprovalReason),
	}).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to approve borrow request")
		return
	}
	borrow.Status = "approved"
	borrow.ApprovedByID = &user.ID
	borrow.ApprovalReason = nullableString(request.ApprovalReason)
	s.audit(c, "borrow_approved", "BorrowRecord", &borrow.ID, gin.H{
		"member_login_id": "",
		"item_name":       item.Name,
		"approved_reason": borrow.ApprovalReason,
	})
	s.notifyMemberUser(c, borrow.MemberID, "Borrow approved", "Your request for '"+item.Name+"' has been approved", "inventory_approved", "BorrowRecord", &borrow.ID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Borrow request approved",
		"request": s.serializeBorrowRequestFull(borrow, item),
	})
}

func (s *Server) rejectBorrowRequest(c *gin.Context) {
	borrow, item, ok := s.findBorrowRequest(c)
	if !ok {
		return
	}
	if borrow.Status != "pending" {
		fail(c, http.StatusUnprocessableEntity, "Only pending requests can be rejected")
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
	err := s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(borrow).Updates(map[string]any{
			"status":          "rejected",
			"approval_reason": reason,
		}).Error; err != nil {
			return err
		}
		var locked models.InventoryItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, borrow.InventoryItemID).Error; err != nil {
			return err
		}
		restored := locked.AvailableQuantity + borrow.Quantity
		if restored > locked.TotalQuantity {
			restored = locked.TotalQuantity
		}
		return tx.Model(&models.InventoryItem{}).Where("id = ?", locked.ID).
			Update("available_quantity", restored).Error
	})
	if err != nil {
		s.Log.Error("borrow reject failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to reject borrow request")
		return
	}
	borrow.Status = "rejected"
	borrow.ApprovalReason = &reason
	s.audit(c, "borrow_rejected", "BorrowRecord", &borrow.ID, gin.H{
		"item_name": item.Name, "rejection_reason": reason,
	})
	s.notifyMemberUser(c, borrow.MemberID, "Borrow request rejected", "Your request for '"+item.Name+"' was rejected: "+reason, "inventory_rejected", "BorrowRecord", &borrow.ID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Borrow request rejected",
		"request": s.serializeBorrowRequestFull(borrow, item),
	})
}

func (s *Server) createExtensionRequest(c *gin.Context) {
	member := currentMember(c)
	if member == nil || member.Status != "active" {
		fail(c, http.StatusForbidden, "Active member account required")
		return
	}
	var request struct {
		BorrowRecordID      int64  `json:"borrow_record_id"`
		Reason              string `json:"reason"`
		RequestedReturnDate string `json:"requested_return_date"`
	}
	if !bindJSON(c, &request) {
		return
	}
	if request.BorrowRecordID <= 0 {
		fail(c, http.StatusUnprocessableEntity, "Borrow record ID is required")
		return
	}
	var borrow models.BorrowRecord
	if err := s.DB.WithContext(c.Request.Context()).
		Where("id = ? AND member_id = ? AND status = ?", request.BorrowRecordID, member.ID, "approved").
		First(&borrow).Error; err != nil {
		fail(c, http.StatusNotFound, "Active borrow record not found")
		return
	}
	if !startOfDay(borrow.DueDate).Before(startOfDay(time.Now())) {
		fail(c, http.StatusUnprocessableEntity, "Extension can only be requested for overdue items")
		return
	}
	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		fail(c, http.StatusUnprocessableEntity, "Reason is required")
		return
	}
	requestedDate := startOfDay(time.Now()).AddDate(0, 0, 14)
	if strings.TrimSpace(request.RequestedReturnDate) != "" {
		parsed, ok := parseDate(request.RequestedReturnDate)
		if !ok {
			fail(c, http.StatusUnprocessableEntity, "Requested return date is invalid")
			return
		}
		requestedDate = startOfDay(parsed)
	}
	if !requestedDate.After(startOfDay(borrow.DueDate)) {
		fail(c, http.StatusUnprocessableEntity, "Requested return date must be after the current due date")
		return
	}
	var extension models.ExtensionRequest
	err := s.DB.WithContext(c.Request.Context()).
		Where("borrow_record_id = ?", borrow.ID).First(&extension).Error
	if err != nil {
		extension = models.ExtensionRequest{
			BorrowRecordID:      borrow.ID,
			MemberID:            member.ID,
			RequestedReturnDate: requestedDate,
			Reason:              reason,
			Status:              "pending",
		}
		if err := s.DB.WithContext(c.Request.Context()).Create(&extension).Error; err != nil {
			fail(c, http.StatusInternalServerError, "Unable to submit extension request")
			return
		}
	}
	s.audit(c, "extension_requested", "ExtensionRequest", &extension.ID, gin.H{
		"current_due_date":      dateValue(borrow.DueDate),
		"requested_return_date": dateValue(extension.RequestedReturnDate),
		"reason":                reason,
	})
	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"message":   "Extension request submitted",
		"extension": s.serializeExtension(&extension),
	})
}

func (s *Server) adminListBorrowRequests(c *gin.Context) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.BorrowRecord{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if pending := strings.TrimSpace(c.Query("pending")); pending == "true" {
		query = query.Where("status = ?", "pending")
	}
	if overdue := strings.TrimSpace(c.Query("overdue")); overdue == "true" {
		query = query.Where("due_date < ? AND status IN ?", startOfDay(time.Now()), []string{"approved", "borrowed"})
	}
	if memberID := strings.TrimSpace(c.Query("member_id")); memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}
	if itemID := strings.TrimSpace(c.Query("inventory_item_id")); itemID != "" {
		query = query.Where("inventory_item_id = ?", itemID)
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load borrow requests")
		return
	}
	var records []models.BorrowRecord
	if err := query.Order("created_at DESC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&records).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load borrow requests")
		return
	}
	items := make([]gin.H, 0, len(records))
	for i := range records {
		var member models.Member
		var item models.InventoryItem
		s.DB.First(&member, records[i].MemberID)
		s.DB.First(&item, records[i].InventoryItemID)
		items = append(items, s.serializeBorrowRequestAdmin(&records[i], &member, &item))
	}
	var pendingCount, overdueCount int64
	s.DB.Model(&models.BorrowRecord{}).Where("status = ?", "pending").Count(&pendingCount)
	s.DB.Model(&models.BorrowRecord{}).Where("status = ? AND due_date <= ?", "approved", startOfDay(time.Now())).Count(&overdueCount)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"requests": items,
		"pagination": gin.H{
			"page": page, "per_page": perPage, "total_count": total,
			"pending_count": pendingCount, "overdue_count": overdueCount,
		},
	})
}

func (s *Server) notifyMemberUser(c *gin.Context, memberID int64, title, body, notificationType, subjectType string, subjectID *int64) {
	var member models.Member
	if err := s.DB.First(&member, memberID).Error; err != nil {
		return
	}
	s.notifyUser(c.Request.Context(), member.UserID, title, body, notificationType, subjectType, subjectID)
}
