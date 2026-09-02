package controllers

import (
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateStoreRequest struct {
	CompanyID         uint       `json:"company_id" binding:"required"`
	Name              string     `json:"name" binding:"required"`
	Address           string     `json:"address"`
	Phone             string     `json:"phone"`
	ManagerEmployeeID *uint      `json:"manager_employee_id"`
	ManagerName       string     `json:"manager_name"`
	OpenedAt          *time.Time `json:"opened_at"`
	Notes             string     `json:"notes"`
}

type UpdateStoreRequest struct {
	CompanyID         uint       `json:"company_id" binding:"required"`
	Name              string     `json:"name" binding:"required"`
	Address           string     `json:"address"`
	Phone             string     `json:"phone"`
	ManagerEmployeeID *uint      `json:"manager_employee_id"`
	ManagerName       string     `json:"manager_name"`
	OpenedAt          *time.Time `json:"opened_at"`
	Notes             string     `json:"notes"`
	Status            string     `json:"status" binding:"omitempty,oneof=aktif nonaktif"`
}

func CreateStore(c *gin.Context) {
	var req CreateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if ok, msg := checkPlanLimit(req.CompanyID, "outlets"); !ok {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": msg})
		return
	}

	store := models.Store{
		CompanyID:         req.CompanyID,
		Name:              req.Name,
		Address:           req.Address,
		Phone:             req.Phone,
		ManagerEmployeeID: req.ManagerEmployeeID,
		ManagerName:       req.ManagerName,
		OpenedAt:          req.OpenedAt,
		Notes:             req.Notes,
		Status:            "aktif",
	}

	if err := database.DB.Create(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create store"})
		return
	}
	store.Code = fmtStoreCode(store.ID)
	database.DB.Model(&store).Update("code", store.Code)

	logAudit(c, store.CompanyID, "create", "outlet", store.ID, store.Name, nil)
	c.JSON(http.StatusCreated, store)
}

func fmtStoreCode(id uint) string {
	return "OUT-" + padNumber(id)
}

func padNumber(id uint) string {
	s := "0000"
	str := []byte(s)
	val := id
	for i := len(str) - 1; i >= 0 && val > 0; i-- {
		str[i] = byte('0' + val%10)
		val /= 10
	}
	return string(str)
}

func GetStores(c *gin.Context) {
	var stores []models.Store
	q := database.DB.Preload("Company")
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if hasTrx := c.Query("hasTrx"); hasTrx != "" {
		sub := database.DB.Model(&models.SalesTransaction{}).Select("DISTINCT store_id")
		if hasTrx == "true" {
			q = q.Where("id IN (?)", sub)
		} else if hasTrx == "false" {
			q = q.Where("id NOT IN (?)", sub)
		}
	}
	if err := q.Find(&stores).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stores"})
		return
	}
	c.JSON(http.StatusOK, stores)
}

func GetStore(c *gin.Context) {
	id := c.Param("id")
	var store models.Store
	if err := database.DB.Preload("Company").First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Store not found"})
		return
	}
	c.JSON(http.StatusOK, store)
}

func UpdateStore(c *gin.Context) {
	id := c.Param("id")
	var store models.Store
	if err := database.DB.First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Store not found"})
		return
	}

	var input UpdateStoreRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	store.Name = input.Name
	store.Address = input.Address
	store.CompanyID = input.CompanyID
	store.Phone = input.Phone
	store.ManagerEmployeeID = input.ManagerEmployeeID
	store.ManagerName = input.ManagerName
	store.OpenedAt = input.OpenedAt
	store.Notes = input.Notes
	if input.Status != "" {
		store.Status = input.Status
	}

	if err := database.DB.Save(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update store"})
		return
	}

	logAudit(c, store.CompanyID, "update", "outlet", store.ID, store.Name, nil)
	c.JSON(http.StatusOK, store)
}

// ToggleStoreStatus implements docs/outlet.md POST /outlets/:id/toggle-status
func ToggleStoreStatus(c *gin.Context) {
	id := c.Param("id")
	var store models.Store
	if err := database.DB.First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Store not found"})
		return
	}
	if store.Status == "aktif" {
		store.Status = "nonaktif"
	} else {
		store.Status = "aktif"
	}
	database.DB.Save(&store)
	logAudit(c, store.CompanyID, "toggle", "outlet", store.ID, store.Name, nil)
	c.JSON(http.StatusOK, store)
}

// GetStoreMetrics implements docs/outlet.md GET /outlets/:id/metrics
func GetStoreMetrics(c *gin.Context) {
	id := c.Param("id")
	var store models.Store
	if err := database.DB.First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Store not found"})
		return
	}

	var totalSales float64
	var totalTransactions int64
	var lastTransactionAt *time.Time
	database.DB.Model(&models.SalesTransaction{}).Where("store_id = ? AND status <> 'void'", store.ID).Count(&totalTransactions)
	database.DB.Model(&models.SalesTransaction{}).Where("store_id = ? AND status <> 'void'", store.ID).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalSales)
	var last models.SalesTransaction
	if err := database.DB.Where("store_id = ?", store.ID).Order("transaction_date desc").First(&last).Error; err == nil {
		lastTransactionAt = &last.TransactionDate
	}
	var employeeCount int64
	database.DB.Model(&models.Employee{}).Where("store_id = ?", store.ID).Count(&employeeCount)

	c.JSON(http.StatusOK, gin.H{
		"total_sales":         totalSales,
		"total_transactions":  totalTransactions,
		"last_transaction_at": lastTransactionAt,
		"employees":           employeeCount,
	})
}

// RefreshOutletMetricsCache recomputes and persists docs/outlet.md outlet_metrics_cache.
// In production this would be invoked by a scheduler; exposed here as a manual trigger.
func RefreshOutletMetricsCache(c *gin.Context) {
	id := c.Param("id")
	var store models.Store
	if err := database.DB.First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Store not found"})
		return
	}
	var totalSales float64
	var totalTransactions int64
	database.DB.Model(&models.SalesTransaction{}).Where("store_id = ? AND status <> 'void'", store.ID).Count(&totalTransactions)
	database.DB.Model(&models.SalesTransaction{}).Where("store_id = ? AND status <> 'void'", store.ID).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalSales)
	var last models.SalesTransaction
	var lastAt *time.Time
	if err := database.DB.Where("store_id = ?", store.ID).Order("transaction_date desc").First(&last).Error; err == nil {
		lastAt = &last.TransactionDate
	}
	var employeeCount int64
	database.DB.Model(&models.Employee{}).Where("store_id = ?", store.ID).Count(&employeeCount)

	cache := models.OutletMetricsCache{
		StoreID: store.ID, TotalSales: totalSales, TotalTransactions: totalTransactions,
		LastTransactionAt: lastAt, EmployeeCount: employeeCount, UpdatedAt: time.Now(),
	}
	database.DB.Save(&cache)
	c.JSON(http.StatusOK, cache)
}

func DeleteStore(c *gin.Context) {
	id := c.Param("id")
	var store models.Store
	if err := database.DB.First(&store, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Store not found"})
		return
	}

	if err := database.DB.Delete(&store).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete store"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Store deleted"})
}

func GetStoresByCompany(c *gin.Context) {
	companyID := c.Param("company_id")
	var stores []models.Store
	if err := database.DB.Where("company_id = ?", companyID).Find(&stores).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stores"})
		return
	}
	c.JSON(http.StatusOK, stores)
}

// GetStoreOptions implements docs/outlet.md GET /outlets/options (dropdown {id, name}).
func GetStoreOptions(c *gin.Context) {
	type option struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	var list []option
	q := database.DB.Model(&models.Store{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	q.Select("id, name").Find(&list)
	c.JSON(http.StatusOK, list)
}
