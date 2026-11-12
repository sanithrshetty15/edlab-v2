package api

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

func (s *Server) registerInventoryRoutes(api *gin.RouterGroup) {
	inventory := api.Group("/inventory")
	inventory.Use(s.RequireAuth(), s.RequirePasswordChangeCompleted(), s.RequireActiveMemberOrAdmin())
	inventory.GET("", s.listInventory)
	inventory.GET("/stats", s.RequireAdmin(), s.inventoryStats)
	inventory.POST("/import", s.RequireAdmin(), s.importInventory)
	inventory.GET("/:id", s.showInventoryItem)
	inventory.POST("", s.RequireAdmin(), s.createInventoryItem)
	inventory.PATCH("/:id", s.RequireAdmin(), s.updateInventoryItem)
	inventory.DELETE("/:id", s.RequireAdmin(), s.deleteInventoryItem)
}

func (s *Server) RequireActiveMemberOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := currentUser(c)
		if user == nil {
			fail(c, http.StatusUnauthorized, "Authentication required")
			c.Abort()
			return
		}
		if user.Role == "admin" {
			c.Next()
			return
		}
		member := currentMember(c)
		if member == nil || member.Status != "active" {
			fail(c, http.StatusForbidden, "Access denied")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) serializeInventoryItem(item *models.InventoryItem) gin.H {
	return gin.H{
		"id":                 item.ID,
		"name":               item.Name,
		"category":           item.Category,
		"total_quantity":     item.TotalQuantity,
		"available_quantity": item.AvailableQuantity,
		"storage_location":   item.StorageLocation,
		"condition":          item.Condition,
		"is_available":       item.AvailableQuantity > 0,
		"created_at":         item.CreatedAt,
	}
}

func (s *Server) serializeInventoryItemFull(item *models.InventoryItem) gin.H {
	result := s.serializeInventoryItem(item)
	result["description"] = item.Description
	result["updated_at"] = item.UpdatedAt
	return result
}

func (s *Server) serializeBorrowHistory(borrow *models.BorrowRecord, member *models.Member) gin.H {
	var approvedBy any
	if borrow.ApprovedByID != nil {
		var user models.User
		if err := s.DB.First(&user, *borrow.ApprovedByID).Error; err == nil {
			approvedBy = user.Email
		}
	}
	name := ""
	loginID := ""
	if member != nil {
		name = member.Name
		loginID = member.LoginID
	}
	return gin.H{
		"id": borrow.ID,
		"member": gin.H{
			"id":       borrow.MemberID,
			"name":     name,
			"login_id": loginID,
		},
		"borrowed_at": borrow.BorrowedAt,
		"due_date":    dateValue(borrow.DueDate),
		"status":      borrow.Status,
		"returned_at": borrow.ReturnedAt,
		"approved_by": approvedBy,
	}
}

func (s *Server) listInventory(c *gin.Context) {
	query := s.DB.WithContext(c.Request.Context()).Model(&models.InventoryItem{}).
		Where("is_active = ? AND deleted_at IS NULL", true)
	if inStock := strings.TrimSpace(c.Query("in_stock")); inStock != "false" {
		query = query.Where("available_quantity > 0")
	}
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("category = ?", category)
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", like, like)
	}
	page, perPage := pagination(c, 20)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load inventory")
		return
	}
	var items []models.InventoryItem
	if err := query.Order("name ASC").Offset(offsetFor(page, perPage)).Limit(perPage).Find(&items).Error; err != nil {
		s.Log.Error("inventory list failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to load inventory")
		return
	}
	result := make([]gin.H, 0, len(items))
	for i := range items {
		result = append(result, s.serializeInventoryItem(&items[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"items":      result,
		"pagination": makePagination(page, perPage, total),
	})
}

func (s *Server) findInventoryItem(c *gin.Context) (*models.InventoryItem, bool) {
	id, ok := pathID(c, "id")
	if !ok {
		return nil, false
	}
	var item models.InventoryItem
	if err := s.DB.WithContext(c.Request.Context()).Where("id = ? AND deleted_at IS NULL", id).
		First(&item).Error; err != nil {
		fail(c, http.StatusNotFound, "Inventory item not found")
		return nil, false
	}
	return &item, true
}

func (s *Server) showInventoryItem(c *gin.Context) {
	item, ok := s.findInventoryItem(c)
	if !ok {
		return
	}
	var history []models.BorrowRecord
	if err := s.DB.WithContext(c.Request.Context()).Where("inventory_item_id = ?", item.ID).
		Order("borrowed_at DESC").Limit(20).Find(&history).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to load borrow history")
		return
	}
	items := make([]gin.H, 0, len(history))
	for i := range history {
		var member models.Member
		s.DB.First(&member, history[i].MemberID)
		items = append(items, s.serializeBorrowHistory(&history[i], &member))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"item":           s.serializeInventoryItemFull(item),
		"borrow_history": items,
	})
}

type inventoryItemRequest struct {
	Name            string `json:"name"`
	Category        string `json:"category"`
	Description     string `json:"description"`
	TotalQuantity   int    `json:"total_quantity"`
	StorageLocation string `json:"storage_location"`
	Condition       string `json:"condition"`
}

func (s *Server) inventoryDuplicate(name, category string, excludeID int64) bool {
	var count int64
	query := s.DB.Model(&models.InventoryItem{}).
		Where("is_active = ? AND deleted_at IS NULL AND name = ? AND category = ?", true, name, category)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	query.Count(&count)
	return count > 0
}

func (s *Server) createInventoryItem(c *gin.Context) {
	var request inventoryItemRequest
	if !bindJSON(c, &request) {
		return
	}
	name := strings.TrimSpace(request.Name)
	category := strings.TrimSpace(request.Category)
	if name == "" {
		fail(c, http.StatusUnprocessableEntity, "Name is required")
		return
	}
	if category == "" {
		fail(c, http.StatusUnprocessableEntity, "Category is required")
		return
	}
	if request.TotalQuantity < 0 {
		fail(c, http.StatusUnprocessableEntity, "Total quantity must be non-negative")
		return
	}
	if s.inventoryDuplicate(name, category, 0) {
		fail(c, http.StatusUnprocessableEntity, "An item with this name and category already exists")
		return
	}
	item := models.InventoryItem{
		Name:              name,
		Category:          category,
		Description:       nullableString(request.Description),
		TotalQuantity:     request.TotalQuantity,
		AvailableQuantity: request.TotalQuantity,
		StorageLocation:   nullableString(request.StorageLocation),
		Condition:         nullableString(request.Condition),
		IsActive:          true,
	}
	if err := s.DB.WithContext(c.Request.Context()).Create(&item).Error; err != nil {
		s.Log.Error("inventory create failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to create inventory item")
		return
	}
	s.audit(c, "inventory_item_created", "InventoryItem", &item.ID, gin.H{
		"name": name, "category": category, "total_quantity": request.TotalQuantity,
	})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Inventory item created",
		"item":    s.serializeInventoryItemFull(&item),
	})
}

func (s *Server) updateInventoryItem(c *gin.Context) {
	item, ok := s.findInventoryItem(c)
	if !ok {
		return
	}
	var request struct {
		Name            *string `json:"name"`
		Category        *string `json:"category"`
		Description     *string `json:"description"`
		TotalQuantity   *int    `json:"total_quantity"`
		StorageLocation *string `json:"storage_location"`
		Condition       *string `json:"condition"`
	}
	if !bindJSON(c, &request) {
		return
	}
	changes := gin.H{}
	if request.Name != nil {
		name := strings.TrimSpace(*request.Name)
		if name == "" {
			fail(c, http.StatusUnprocessableEntity, "Name cannot be empty")
			return
		}
		if item.Name != name {
			changes["name"] = gin.H{"from": item.Name, "to": name}
			item.Name = name
		}
	}
	if request.Category != nil {
		category := strings.TrimSpace(*request.Category)
		if category == "" {
			fail(c, http.StatusUnprocessableEntity, "Category cannot be empty")
			return
		}
		if item.Category != category {
			changes["category"] = gin.H{"from": item.Category, "to": category}
			item.Category = category
		}
	}
	if request.Description != nil {
		description := strings.TrimSpace(*request.Description)
		if description != "" && stringOrEmpty(item.Description) != description {
			changes["description"] = gin.H{"from": trimTo(stringOrEmpty(item.Description), 100), "to": trimTo(description, 100)}
			item.Description = &description
		}
	}
	if request.TotalQuantity != nil && item.TotalQuantity != *request.TotalQuantity {
		if *request.TotalQuantity < 0 {
			fail(c, http.StatusUnprocessableEntity, "Total quantity must be non-negative")
			return
		}
		changes["total_quantity"] = gin.H{"from": item.TotalQuantity, "to": *request.TotalQuantity}
		item.TotalQuantity = *request.TotalQuantity
		if item.AvailableQuantity > item.TotalQuantity {
			item.AvailableQuantity = item.TotalQuantity
		}
	}
	if request.StorageLocation != nil {
		location := strings.TrimSpace(*request.StorageLocation)
		if location != "" && stringOrEmpty(item.StorageLocation) != location {
			changes["storage_location"] = gin.H{"from": stringOrEmpty(item.StorageLocation), "to": location}
			item.StorageLocation = &location
		}
	}
	if request.Condition != nil {
		condition := strings.TrimSpace(*request.Condition)
		if condition != "" && stringOrEmpty(item.Condition) != condition {
			changes["condition"] = gin.H{"from": stringOrEmpty(item.Condition), "to": condition}
			item.Condition = &condition
		}
	}
	if len(changes) == 0 {
		fail(c, http.StatusUnprocessableEntity, "No changes provided")
		return
	}
	if s.inventoryDuplicate(item.Name, item.Category, item.ID) {
		fail(c, http.StatusUnprocessableEntity, "An item with this name and category already exists")
		return
	}
	if err := s.DB.WithContext(c.Request.Context()).Save(item).Error; err != nil {
		s.Log.Error("inventory update failed", "error", err)
		fail(c, http.StatusInternalServerError, "Unable to update inventory item")
		return
	}
	s.audit(c, "inventory_item_updated", "InventoryItem", &item.ID, changes)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Inventory item updated",
		"item":    s.serializeInventoryItemFull(item),
	})
}

func (s *Server) deleteInventoryItem(c *gin.Context) {
	item, ok := s.findInventoryItem(c)
	if !ok {
		return
	}
	now := time.Now().UTC()
	if err := s.DB.WithContext(c.Request.Context()).Model(item).Updates(map[string]any{
		"is_active":  false,
		"deleted_at": now,
	}).Error; err != nil {
		fail(c, http.StatusInternalServerError, "Unable to delete inventory item")
		return
	}
	s.audit(c, "inventory_item_deleted", "InventoryItem", &item.ID, gin.H{
		"name": item.Name, "category": item.Category, "total_quantity": item.TotalQuantity,
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Inventory item removed from active inventory"})
}

func (s *Server) inventoryStats(c *gin.Context) {
	var totalItems, totalStock, availableStock int64
	query := s.DB.WithContext(c.Request.Context()).Model(&models.InventoryItem{}).
		Where("is_active = ? AND deleted_at IS NULL", true)
	query.Count(&totalItems)
	query.Select("COALESCE(SUM(total_quantity), 0)").Scan(&totalStock)
	query.Select("COALESCE(SUM(available_quantity), 0)").Scan(&availableStock)
	type categoryRow struct {
		Category  string
		Items     int64
		Total     int64
		Available int64
	}
	var rows []categoryRow
	s.DB.WithContext(c.Request.Context()).Model(&models.InventoryItem{}).
		Where("is_active = ? AND deleted_at IS NULL", true).
		Select("category, COUNT(*) AS items, COALESCE(SUM(total_quantity), 0) AS total, COALESCE(SUM(available_quantity), 0) AS available").
		Group("category").Order("total DESC").Scan(&rows)
	categories := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, gin.H{
			"category":           row.Category,
			"items":              row.Items,
			"total_quantity":     row.Total,
			"available_quantity": row.Available,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats": gin.H{
			"total_items":     totalItems,
			"total_stock":     totalStock,
			"available_stock": availableStock,
			"borrowed_stock":  totalStock - availableStock,
			"categories":      categories,
		},
	})
}

func (s *Server) importInventory(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusUnprocessableEntity, "File is required")
		return
	}
	extension := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if extension != ".csv" {
		fail(c, http.StatusUnprocessableEntity, "Only CSV files are supported")
		return
	}
	if s.Config.MaxUploadBytes > 0 && fileHeader.Size > s.Config.MaxUploadBytes {
		fail(c, http.StatusUnprocessableEntity, "File is too large")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		fail(c, http.StatusUnprocessableEntity, "Unable to read uploaded file")
		return
	}
	defer file.Close()
	reader := csv.NewReader(io.LimitReader(file, 10*1024*1024))
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		fail(c, http.StatusUnprocessableEntity, "Failed to parse CSV")
		return
	}
	headers := make([]string, len(rows[0]))
	for i, header := range rows[0] {
		headers[i] = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(header), " ", "_"))
	}
	index := map[string]int{}
	for i, header := range headers {
		index[header] = i
	}
	if _, ok := index["name"]; !ok {
		fail(c, http.StatusUnprocessableEntity, "Missing required columns: name")
		return
	}
	if _, ok := index["category"]; !ok {
		fail(c, http.StatusUnprocessableEntity, "Missing required columns: category")
		return
	}
	value := func(row []string, key string) string {
		if position, ok := index[key]; ok && position < len(row) {
			return strings.TrimSpace(row[position])
		}
		return ""
	}
	record := models.InventoryImport{
		Filename:     fileHeader.Filename,
		TotalRows:    len(rows) - 1,
		ImportedByID: currentUser(c).ID,
		Status:       "completed",
	}
	errorRows := 0
	err = s.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		for _, row := range rows[1:] {
			name := value(row, "name")
			category := value(row, "category")
			if name == "" && category == "" {
				continue
			}
			if name == "" || category == "" {
				errorRows++
				continue
			}
			quantity := 0
			if raw := value(row, "total_quantity"); raw != "" {
				parsed, parseErr := strconv.Atoi(raw)
				if parseErr != nil || parsed < 0 {
					errorRows++
					continue
				}
				quantity = parsed
			}
			description := value(row, "description")
			location := value(row, "storage_location")
			condition := value(row, "condition")
			var existing models.InventoryItem
			findErr := tx.Where("is_active = ? AND deleted_at IS NULL AND name = ? AND category = ?", true, name, category).First(&existing).Error
			if findErr == nil {
				existing.TotalQuantity += quantity
				existing.AvailableQuantity += quantity
				if existing.AvailableQuantity > existing.TotalQuantity {
					existing.AvailableQuantity = existing.TotalQuantity
				}
				if description != "" {
					existing.Description = &description
				}
				if location != "" {
					existing.StorageLocation = &location
				}
				if condition != "" {
					existing.Condition = &condition
				}
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
				continue
			}
			if findErr != gorm.ErrRecordNotFound {
				return findErr
			}
			item := models.InventoryItem{
				Name:              name,
				Category:          category,
				Description:       nullableString(description),
				TotalQuantity:     quantity,
				AvailableQuantity: quantity,
				StorageLocation:   nullableString(location),
				Condition:         nullableString(condition),
				IsActive:          true,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
		record.ErrorRows = errorRows
		record.ImportedRows = record.TotalRows - errorRows
		if errorRows > 0 {
			record.Status = "completed_with_errors"
		}
		return tx.Save(&record).Error
	})
	if err != nil {
		s.Log.Error("inventory import failed", "error", err)
		fail(c, http.StatusUnprocessableEntity, fmt.Sprintf("Import failed: %v", err))
		return
	}
	s.audit(c, "inventory_imported", "InventoryImport", &record.ID, gin.H{
		"filename": record.Filename, "total_rows": record.TotalRows,
		"imported_rows": record.ImportedRows, "error_rows": record.ErrorRows,
	})
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Inventory imported successfully",
		"import": gin.H{
			"id":            record.ID,
			"filename":      record.Filename,
			"total_rows":    record.TotalRows,
			"imported_rows": record.ImportedRows,
			"error_rows":    record.ErrorRows,
			"status":        record.Status,
		},
	})
}
