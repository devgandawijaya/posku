package controllers

import (
	"fmt"
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type CreateCustomerRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"omitempty,email"`
	Phone     string `json:"phone" binding:"omitempty"`
	StoreIDs  []uint `json:"store_ids"`
}

type UpdateCustomerRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"omitempty,email"`
	Phone     string `json:"phone" binding:"omitempty"`
	StoreIDs  []uint `json:"store_ids"`
	Tier      string `json:"tier" binding:"omitempty,oneof=bronze silver gold vip"`
	Status    string `json:"status" binding:"omitempty,oneof=aktif nonaktif"`
}

func CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cu := models.Customer{
		CompanyID: req.CompanyID,
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		StoreIDs:  encodeIDs(req.StoreIDs),
		Tier:      "bronze",
		Status:    "aktif",
	}

	if err := database.DB.Create(&cu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer"})
		return
	}
	cu.MemberCode = fmt.Sprintf("MBR%05d", cu.ID)
	database.DB.Model(&cu).Update("member_code", cu.MemberCode)

	c.JSON(http.StatusCreated, cu)
}

func GetCustomers(c *gin.Context) {
	var list []models.Customer
	q := database.DB.Model(&models.Customer{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if tier := c.Query("tier"); tier != "" {
		q = q.Where("tier = ?", tier)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR email ILIKE ? OR phone ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if err := q.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customers"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetCustomer(c *gin.Context) {
	id := c.Param("id")
	var cu models.Customer
	if err := database.DB.First(&cu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	c.JSON(http.StatusOK, cu)
}

func UpdateCustomer(c *gin.Context) {
	id := c.Param("id")
	var cu models.Customer
	if err := database.DB.First(&cu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	var input UpdateCustomerRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cu.Name = input.Name
	cu.Email = input.Email
	cu.Phone = input.Phone
	cu.CompanyID = input.CompanyID
	if input.StoreIDs != nil {
		cu.StoreIDs = encodeIDs(input.StoreIDs)
	}
	if input.Tier != "" {
		cu.Tier = input.Tier
	}
	if input.Status != "" {
		cu.Status = input.Status
	}

	if err := database.DB.Save(&cu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer"})
		return
	}

	c.JSON(http.StatusOK, cu)
}

func DeleteCustomer(c *gin.Context) {
	id := c.Param("id")
	var cu models.Customer
	if err := database.DB.First(&cu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	if err := database.DB.Delete(&cu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete customer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer deleted"})
}

type AdjustPointsRequest struct {
	Delta  int    `json:"delta" binding:"required"`
	Reason string `json:"reason" binding:"required,oneof=earn redeem adjust expire"`
	Note   string `json:"note"`
}

// AdjustCustomerPoints implements docs/pelanggan.md POST /customers/:id/points
func AdjustCustomerPoints(c *gin.Context) {
	id := c.Param("id")
	var cu models.Customer
	if err := database.DB.First(&cu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	var req AdjustPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cu.PointsBalance += req.Delta
	database.DB.Save(&cu)
	database.DB.Create(&models.PointsLedger{
		CompanyID:  cu.CompanyID,
		CustomerID: cu.ID,
		Delta:      req.Delta,
		Reason:     req.Reason,
		Note:       req.Note,
	})
	recalcCustomerTier(cu.ID)
	c.JSON(http.StatusOK, cu)
}

// GetCustomerOptions implements docs/pelanggan.md GET /customers/options (dropdown kasir).
func GetCustomerOptions(c *gin.Context) {
	type option struct {
		ID         uint   `json:"id"`
		MemberCode string `json:"member_code"`
		Name       string `json:"name"`
	}
	var list []option
	q := database.DB.Model(&models.Customer{}).Where("status = 'aktif'")
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR member_code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	q.Select("id, member_code, name").Limit(20).Find(&list)
	c.JSON(http.StatusOK, list)
}

// GetCustomerTransactions implements docs/pelanggan.md GET /customers/:id/transactions
func GetCustomerTransactions(c *gin.Context) {
	id := c.Param("id")
	var list []models.SalesTransaction
	q := database.DB.Preload("SalesItems").Where("customer_id = ?", id)
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		q = q.Where("transaction_date >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		q = q.Where("transaction_date <= ?", dateTo)
	}
	q.Order("transaction_date desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// recalcCustomerTier recomputes total_spent (rolling all-time, simplified from "12 bulan" in docs)
// and assigns tier thresholds; called after points adjustments / sales as a lightweight substitute
// for the scheduled cron job described in docs/pelanggan.md.
func recalcCustomerTier(customerID uint) models.CustomerMetricsCache {
	var totalSpent float64
	var totalTransactions int64
	database.DB.Model(&models.SalesTransaction{}).Where("customer_id = ? AND status <> 'void'", customerID).Count(&totalTransactions)
	database.DB.Model(&models.SalesTransaction{}).Where("customer_id = ? AND status <> 'void'", customerID).Select("COALESCE(SUM(total_amount),0)").Scan(&totalSpent)

	tier := "bronze"
	switch {
	case totalSpent >= 10000000:
		tier = "vip"
	case totalSpent >= 5000000:
		tier = "gold"
	case totalSpent >= 1000000:
		tier = "silver"
	}
	database.DB.Model(&models.Customer{}).Where("id = ?", customerID).Update("tier", tier)

	var lastVisit *time.Time
	var last models.SalesTransaction
	if err := database.DB.Where("customer_id = ?", customerID).Order("transaction_date desc").First(&last).Error; err == nil {
		lastVisit = &last.TransactionDate
	}

	cache := models.CustomerMetricsCache{
		CustomerID: customerID, TotalTransactions: totalTransactions, TotalSpent: totalSpent,
		LastVisitAt: lastVisit, UpdatedAt: time.Now(),
	}
	database.DB.Save(&cache)
	return cache
}

// RefreshCustomerMetrics implements docs/pelanggan.md customer_metrics_cache + tier recompute.
func RefreshCustomerMetrics(c *gin.Context) {
	id := c.Param("id")
	var cu models.Customer
	if err := database.DB.First(&cu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	cache := recalcCustomerTier(cu.ID)
	c.JSON(http.StatusOK, cache)
}
