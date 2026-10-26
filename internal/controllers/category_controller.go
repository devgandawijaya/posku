package controllers

import (
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateCategoryRequest struct {
	CompanyID uint   `json:"company_id"`
	ParentID  *uint  `json:"parent_id"`
	Code      string `json:"code"`
	Name      string `json:"name" binding:"required"`
	Slug      string `json:"slug"`
	Icon      string `json:"icon"`
	Scope     string `json:"scope" binding:"required,oneof=all specific"`
	StoreIDs  []uint `json:"store_ids"`
	IsActive  *bool  `json:"is_active"`
	SortOrder int    `json:"sort_order"`
}

type UpdateCategoryRequest struct {
	ParentID  *uint  `json:"parent_id"`
	Code      string `json:"code"`
	Name      string `json:"name" binding:"required"`
	Slug      string `json:"slug"`
	Icon      string `json:"icon"`
	Scope     string `json:"scope" binding:"required,oneof=all specific"`
	StoreIDs  []uint `json:"store_ids"`
	IsActive  *bool  `json:"is_active"`
	SortOrder int    `json:"sort_order"`
}

func CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
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

	if req.Scope == "specific" && len(req.StoreIDs) == 0 {
		RespondError(c, http.StatusBadRequest, "storeIds wajib diisi untuk scope specific")
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	cat := models.Category{
		CompanyID: companyID,
		ParentID:  req.ParentID,
		Code:      req.Code,
		Name:      req.Name,
		Slug:      req.Slug,
		Icon:      req.Icon,
		Scope:     req.Scope,
		StoreIDs:  encodeIDs(req.StoreIDs),
		IsActive:  isActive,
		SortOrder: req.SortOrder,
	}
	if err := database.DB.Create(&cat).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to create category")
		return
	}
	logAudit(c, cat.CompanyID, "category_create", "category", cat.ID, cat.Name, nil)
	RespondCreated(c, cat, "Category created successfully")
}

func GetCategories(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	var list []models.Category
	q := database.DB.Model(&models.Category{})

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}

	if parentID := c.Query("parent_id"); parentID != "" {
		q = q.Where("parent_id = ?", parentID)
	}
	if status := c.Query("status"); status == "active" {
		q = q.Where("is_active = ?", true)
	} else if status == "inactive" {
		q = q.Where("is_active = ?", false)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}

	if c.Query("page") != "" || c.Query("per_page") != "" {
		var total int64
		q.Count(&total)
		page, perPage, offset := GetPagination(c)
		if err := q.Order("sort_order asc, id asc").Limit(perPage).Offset(offset).Find(&list).Error; err != nil {
			RespondError(c, http.StatusInternalServerError, "Failed to fetch categories")
			return
		}
		RespondPaginated(c, list, page, perPage, total)
		return
	}

	if err := q.Order("sort_order asc, id asc").Find(&list).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to fetch categories")
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetCategory(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var cat models.Category
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&cat, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Category not found")
		return
	}
	RespondSuccess(c, cat)
}

func UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var cat models.Category
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&cat, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Category not found")
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Scope == "specific" && len(req.StoreIDs) == 0 {
		RespondError(c, http.StatusBadRequest, "storeIds wajib diisi untuk scope specific")
		return
	}

	cat.ParentID = req.ParentID
	cat.Code = req.Code
	cat.Name = req.Name
	cat.Slug = req.Slug
	cat.Icon = req.Icon
	cat.Scope = req.Scope
	cat.StoreIDs = encodeIDs(req.StoreIDs)
	if req.IsActive != nil {
		cat.IsActive = *req.IsActive
	}
	cat.SortOrder = req.SortOrder

	if err := database.DB.Save(&cat).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to update category")
		return
	}
	logAudit(c, cat.CompanyID, "category_update", "category", cat.ID, cat.Name, nil)
	RespondSuccess(c, cat, "Category updated successfully")
}

func DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var cat models.Category
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&cat, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Category not found")
		return
	}
	var productCount int64
	database.DB.Model(&models.Product{}).Where("category_id = ?", cat.ID).Count(&productCount)
	if productCount > 0 {
		RespondError(c, http.StatusConflict, "tidak bisa hapus kategori yang masih dipakai produk")
		return
	}
	if err := database.DB.Delete(&cat).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete category")
		return
	}
	logAudit(c, cat.CompanyID, "category_delete", "category", cat.ID, cat.Name, nil)
	c.JSON(http.StatusOK, gin.H{"message": "Category deleted"})
}

func GetCategoryKPI(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var cat models.Category
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&cat, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Category not found")
		return
	}
	var productCount int64
	database.DB.Model(&models.Product{}).Where("category_id = ?", cat.ID).Count(&productCount)

	var totalOmzet float64
	database.DB.Model(&models.SalesItem{}).
		Joins("JOIN products ON products.id = sales_items.product_id").
		Where("products.category_id = ?", cat.ID).
		Select("COALESCE(SUM(sales_items.subtotal), 0)").Scan(&totalOmzet)

	c.JSON(http.StatusOK, gin.H{
		"product_count": productCount,
		"total_omzet":   totalOmzet,
	})
}

// RefreshCategoryKPICache recomputes and persists docs/kategori.md category_kpi_cache.
func RefreshCategoryKPICache(c *gin.Context) {
	id := c.Param("id")
	var cat models.Category
	if err := database.DB.First(&cat, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Category not found")
		return
	}
	var productCount int64
	database.DB.Model(&models.Product{}).Where("category_id = ?", cat.ID).Count(&productCount)

	var totalStock float64
	database.DB.Table("stocks").
		Joins("JOIN products ON products.id = stocks.product_id").
		Where("products.category_id = ?", cat.ID).
		Select("COALESCE(SUM(stocks.quantity), 0)").Scan(&totalStock)

	var totalOmzet float64
	database.DB.Model(&models.SalesItem{}).
		Joins("JOIN products ON products.id = sales_items.product_id").
		Where("products.category_id = ?", cat.ID).
		Select("COALESCE(SUM(sales_items.subtotal), 0)").Scan(&totalOmzet)

	cache := models.CategoryKPICache{
		CategoryID:   cat.ID,
		ProductCount: productCount,
		TotalStock:   totalStock,
		TotalOmzet:   totalOmzet,
		LastSyncedAt: time.Now(),
	}
	database.DB.Save(&cache)
	RespondSuccess(c, cache, "Category KPI refreshed")
}

func GetCategoryKPICache(c *gin.Context) {
	id := c.Param("id")
	var cache models.CategoryKPICache
	if err := database.DB.First(&cache, "category_id = ?", id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "kpi cache belum di-refresh")
		return
	}
	c.JSON(http.StatusOK, cache)
}
