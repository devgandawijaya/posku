package controllers

import (
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateSupplierRequest struct {
	CompanyID     uint   `json:"company_id"`
	Name          string `json:"name" binding:"required"`
	ContactPerson string `json:"contact_person"`
	Phone         string `json:"phone"`
	Email         string `json:"email" binding:"omitempty,email"`
	Address       string `json:"address"`
	Category      string `json:"category"`
	StoreIDs      []uint `json:"store_ids"`
}

type UpdateSupplierRequest struct {
	Name          string `json:"name" binding:"required"`
	ContactPerson string `json:"contact_person"`
	Phone         string `json:"phone"`
	Email         string `json:"email" binding:"omitempty,email"`
	Address       string `json:"address"`
	Category      string `json:"category"`
	StoreIDs      []uint `json:"store_ids"`
	Status        string `json:"status" binding:"omitempty,oneof=aktif nonaktif"`
}

func CreateSupplier(c *gin.Context) {
	var req CreateSupplierRequest
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

	sup := models.Supplier{
		CompanyID:     companyID,
		Name:          req.Name,
		ContactPerson: req.ContactPerson,
		Phone:         req.Phone,
		Email:         req.Email,
		Address:       req.Address,
		Category:      req.Category,
		StoreIDs:      encodeIDs(req.StoreIDs),
		Status:        "aktif",
	}
	if err := database.DB.Create(&sup).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to create supplier")
		return
	}
	sup.Code = "SUP" + padNumber(sup.ID+1000)
	database.DB.Model(&sup).Update("code", sup.Code)
	logAudit(c, sup.CompanyID, "create", "supplier", sup.ID, sup.Name, nil)
	RespondCreated(c, sup, "Supplier created successfully")
}

func GetSuppliers(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	var list []models.Supplier
	q := database.DB.Model(&models.Supplier{})

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}

	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR code ILIKE ? OR contact_person ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if c.Query("page") != "" || c.Query("per_page") != "" {
		var total int64
		q.Count(&total)
		page, perPage, offset := GetPagination(c)
		if err := q.Order("id asc").Limit(perPage).Offset(offset).Find(&list).Error; err != nil {
			RespondError(c, http.StatusInternalServerError, "Failed to fetch suppliers")
			return
		}
		RespondPaginated(c, list, page, perPage, total)
		return
	}

	if err := q.Order("id asc").Find(&list).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to fetch suppliers")
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetSupplier(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var sup models.Supplier
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&sup, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Supplier not found")
		return
	}
	RespondSuccess(c, sup)
}

func UpdateSupplier(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var sup models.Supplier
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&sup, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Supplier not found")
		return
	}
	var req UpdateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	sup.Name = req.Name
	sup.ContactPerson = req.ContactPerson
	sup.Phone = req.Phone
	sup.Email = req.Email
	sup.Address = req.Address
	sup.Category = req.Category
	if req.StoreIDs != nil {
		sup.StoreIDs = encodeIDs(req.StoreIDs)
	}
	if req.Status != "" {
		sup.Status = req.Status
	}

	if err := database.DB.Save(&sup).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to update supplier")
		return
	}
	logAudit(c, sup.CompanyID, "update", "supplier", sup.ID, sup.Name, nil)
	RespondSuccess(c, sup, "Supplier updated successfully")
}

func DeleteSupplier(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var sup models.Supplier
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&sup, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Supplier not found")
		return
	}

	if err := database.DB.Delete(&sup).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete supplier")
		return
	}
	logAudit(c, sup.CompanyID, "delete", "supplier", sup.ID, sup.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "Supplier deleted"})
}

// ToggleSupplierStatus implements docs/supplier.md POST /suppliers/:id/toggle-status
func ToggleSupplierStatus(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var sup models.Supplier
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&sup, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Supplier not found")
		return
	}
	if sup.Status == "aktif" {
		sup.Status = "nonaktif"
	} else {
		sup.Status = "aktif"
	}
	database.DB.Save(&sup)
	logAudit(c, sup.CompanyID, "toggle", "supplier", sup.ID, sup.Name, nil)
	RespondSuccess(c, sup, "Supplier status toggled")
}

type AssignSupplierOutletsRequest struct {
	StoreIDs []uint `json:"store_ids" binding:"required"`
}

// AssignSupplierOutlets implements docs/supplier.md POST /suppliers/:id/assign-outlets
func AssignSupplierOutlets(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var sup models.Supplier
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&sup, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Supplier not found")
		return
	}
	var req AssignSupplierOutletsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	sup.StoreIDs = encodeIDs(req.StoreIDs)
	database.DB.Save(&sup)
	logAudit(c, sup.CompanyID, "assign", "supplier", sup.ID, sup.Name, req.StoreIDs)
	RespondSuccess(c, sup, "Supplier outlets assigned")
}

// GetSupplierOptions implements docs/supplier.md GET /suppliers/options.
func GetSupplierOptions(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	type option struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		Code string `json:"code"`
	}
	var list []option
	q := database.DB.Model(&models.Supplier{}).Where("status = 'aktif'")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}
	q.Select("id, name, code").Find(&list)
	c.JSON(http.StatusOK, list)
}

type RecordSupplierPurchaseRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Note   string  `json:"note"`
}

// RecordSupplierPurchase implements docs/supplier.md supplier_metrics_cache driver
func RecordSupplierPurchase(c *gin.Context) {
	id := c.Param("id")
	var sup models.Supplier
	if err := database.DB.First(&sup, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Supplier not found")
		return
	}
	var req RecordSupplierPurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	database.DB.Create(&models.SupplierPurchase{SupplierID: sup.ID, Amount: req.Amount, Note: req.Note})

	sup.TotalPurchases += req.Amount
	now := time.Now()
	sup.LastOrderAt = &now
	database.DB.Save(&sup)

	var productCount int64
	database.DB.Model(&models.Product{}).Where("company_id = ?", sup.CompanyID).Count(&productCount)
	cache := models.SupplierMetricsCache{
		SupplierID: sup.ID, TotalProducts: productCount, TotalPurchases: sup.TotalPurchases,
		LastOrderAt: sup.LastOrderAt, UpdatedAt: now,
	}
	database.DB.Save(&cache)
	c.JSON(http.StatusOK, gin.H{"supplier": sup, "metrics": cache})
}
