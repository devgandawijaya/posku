package controllers

import (
	"fmt"
	"net/http"
	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateWarehouseRequest struct {
	CompanyID uint   `json:"company_id"`
	StoreID   *uint  `json:"store_id"`
	Code      string `json:"code"`
	Name      string `json:"name" binding:"required"`
	Location  string `json:"location"`
	IsCentral bool   `json:"is_central"`
}

type UpdateWarehouseRequest struct {
	StoreID   *uint  `json:"store_id"`
	Code      string `json:"code"`
	Name      string `json:"name" binding:"required"`
	Location  string `json:"location"`
	IsCentral bool   `json:"is_central"`
	Status    string `json:"status" binding:"omitempty,oneof=aktif nonaktif"`
}

func fmtWarehouseCode(id uint) string {
	return fmt.Sprintf("WH-%04d", id)
}

func CreateWarehouse(c *gin.Context) {
	var req CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	companyID := getAuthCompanyID(c)
	if companyID == 0 {
		companyID = req.CompanyID
	}
	if companyID == 0 {
		RespondError(c, http.StatusBadRequest, "company_id is required")
		return
	}

	// Validate store if provided
	if req.StoreID != nil && *req.StoreID > 0 {
		var store models.Store
		if err := database.DB.Where("id = ? AND company_id = ?", *req.StoreID, companyID).First(&store).Error; err != nil {
			RespondError(c, http.StatusBadRequest, "store not found or does not belong to company")
			return
		}
	}

	wh := models.Warehouse{
		CompanyID: companyID,
		StoreID:   req.StoreID,
		Name:      req.Name,
		Location:  req.Location,
		IsCentral: req.IsCentral,
		Status:    "aktif",
	}

	if err := database.DB.Create(&wh).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to create warehouse: "+err.Error())
		return
	}

	if req.Code != "" {
		wh.Code = req.Code
	} else {
		wh.Code = fmtWarehouseCode(wh.ID)
	}
	database.DB.Model(&wh).Update("code", wh.Code)

	logAudit(c, wh.CompanyID, "create", "warehouse", wh.ID, wh.Name, nil)
	RespondCreated(c, wh, "Warehouse created successfully")
}

func GetWarehouses(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	q := database.DB.Model(&models.Warehouse{}).Preload("Store")

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}

	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if isCentral := c.Query("is_central"); isCentral != "" {
		if isCentral == "true" {
			q = q.Where("is_central = true")
		} else if isCentral == "false" {
			q = q.Where("is_central = false")
		}
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR code ILIKE ? OR location ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Pagination
	if c.Query("page") != "" || c.Query("per_page") != "" {
		var total int64
		q.Count(&total)

		page, perPage, offset := GetPagination(c)
		var list []models.Warehouse
		if err := q.Order("id asc").Limit(perPage).Offset(offset).Find(&list).Error; err != nil {
			RespondError(c, http.StatusInternalServerError, "Failed to fetch warehouses")
			return
		}
		RespondPaginated(c, list, page, perPage, total)
		return
	}

	var list []models.Warehouse
	if err := q.Order("id asc").Find(&list).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to fetch warehouses")
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetWarehouse(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var wh models.Warehouse
	q := database.DB.Preload("Store")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&wh, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Warehouse not found")
		return
	}
	RespondSuccess(c, wh)
}

func UpdateWarehouse(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var wh models.Warehouse
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&wh, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Warehouse not found")
		return
	}

	var input UpdateWarehouseRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	if input.StoreID != nil && *input.StoreID > 0 {
		var store models.Store
		if err := database.DB.Where("id = ? AND company_id = ?", *input.StoreID, wh.CompanyID).First(&store).Error; err != nil {
			RespondError(c, http.StatusBadRequest, "store not found or does not belong to company")
			return
		}
	}

	wh.Name = input.Name
	wh.Location = input.Location
	wh.StoreID = input.StoreID
	wh.IsCentral = input.IsCentral
	if input.Code != "" {
		wh.Code = input.Code
	}
	if input.Status != "" {
		wh.Status = input.Status
	}

	if err := database.DB.Save(&wh).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to update warehouse")
		return
	}

	logAudit(c, wh.CompanyID, "update", "warehouse", wh.ID, wh.Name, input)
	RespondSuccess(c, wh, "Warehouse updated successfully")
}

func DeleteWarehouse(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var wh models.Warehouse
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&wh, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Warehouse not found")
		return
	}

	// Check if warehouse has active stocks
	var stockCount int64
	database.DB.Model(&models.Stock{}).Where("warehouse_id = ? AND quantity > 0", wh.ID).Count(&stockCount)
	if stockCount > 0 {
		RespondError(c, http.StatusBadRequest, "Cannot delete warehouse with existing stock inventory")
		return
	}

	if err := database.DB.Delete(&wh).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete warehouse")
		return
	}

	logAudit(c, wh.CompanyID, "delete", "warehouse", wh.ID, wh.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "Warehouse deleted successfully"})
}

func GetWarehouseByStore(c *gin.Context) {
	storeID := c.Param("id")
	if storeID == "" {
		storeID = c.Param("store_id")
	}
	companyID := getAuthCompanyID(c)

	var whs []models.Warehouse
	q := database.DB.Where("store_id = ?", storeID)
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}

	if err := q.Find(&whs).Error; err != nil || len(whs) == 0 {
		RespondError(c, http.StatusNotFound, "Warehouse not found for store")
		return
	}

	// Returns the first/primary warehouse or all warehouses for the store
	c.JSON(http.StatusOK, whs[0])
}
