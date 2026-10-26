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
	CompanyID uint   `json:"company_id"`
	Name      string `json:"name" binding:"required"`
	Email     string `json:"email" binding:"omitempty,email"`
	Phone     string `json:"phone" binding:"omitempty"`
	StoreIDs  []uint `json:"store_ids"`
}

type UpdateCustomerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone" binding:"omitempty"`
	StoreIDs []uint `json:"store_ids"`
	Tier     string `json:"tier" binding:"omitempty,oneof=bronze silver gold vip"`
	Status   string `json:"status" binding:"omitempty,oneof=aktif nonaktif"`
}

func CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
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

	cu := models.Customer{
		CompanyID: companyID,
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		StoreIDs:  encodeIDs(req.StoreIDs),
		Tier:      "bronze",
		Status:    "aktif",
	}

	if err := database.DB.Create(&cu).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to create customer")
		return
	}
	cu.MemberCode = fmt.Sprintf("MBR%05d", cu.ID)
	database.DB.Model(&cu).Update("member_code", cu.MemberCode)

	logAudit(c, cu.CompanyID, "customer_create", "customer", cu.ID, cu.Name, nil)
	RespondCreated(c, cu, "Customer created successfully")
}

func GetCustomers(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	var list []models.Customer
	q := database.DB.Model(&models.Customer{})

	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}

	if tier := c.Query("tier"); tier != "" {
		q = q.Where("tier = ?", tier)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR email ILIKE ? OR phone ILIKE ? OR member_code ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if c.Query("page") != "" || c.Query("per_page") != "" {
		var total int64
		q.Count(&total)
		page, perPage, offset := GetPagination(c)
		if err := q.Order("id asc").Limit(perPage).Offset(offset).Find(&list).Error; err != nil {
			RespondError(c, http.StatusInternalServerError, "Failed to fetch customers")
			return
		}
		RespondPaginated(c, list, page, perPage, total)
		return
	}

	if err := q.Order("id asc").Find(&list).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to fetch customers")
		return
	}
	c.JSON(http.StatusOK, list)
}

func GetCustomer(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var cu models.Customer
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&cu, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Customer not found")
		return
	}
	RespondSuccess(c, cu)
}

func UpdateCustomer(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var cu models.Customer
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&cu, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Customer not found")
		return
	}

	var input UpdateCustomerRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	cu.Name = input.Name
	cu.Email = input.Email
	cu.Phone = input.Phone
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
		RespondError(c, http.StatusInternalServerError, "Failed to update customer")
		return
	}

	logAudit(c, cu.CompanyID, "customer_update", "customer", cu.ID, cu.Name, nil)
	RespondSuccess(c, cu, "Customer updated successfully")
}

func DeleteCustomer(c *gin.Context) {
	id := c.Param("id")
	companyID := getAuthCompanyID(c)

	var cu models.Customer
	q := database.DB
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if err := q.First(&cu, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Customer not found")
		return
	}

	if err := database.DB.Delete(&cu).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete customer")
		return
	}

	logAudit(c, cu.CompanyID, "customer_delete", "customer", cu.ID, cu.Name, nil)
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
		RespondError(c, http.StatusNotFound, "Customer not found")
		return
	}
	var req AdjustPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
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
	RespondSuccess(c, cu, "Customer points adjusted successfully")
}

// GetCustomerOptions implements docs/pelanggan.md GET /customers/options (dropdown kasir).
func GetCustomerOptions(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	type option struct {
		ID         uint   `json:"id"`
		MemberCode string `json:"member_code"`
		Name       string `json:"name"`
		Tier       string `json:"tier"`
	}
	var list []option
	q := database.DB.Model(&models.Customer{}).Where("status = 'aktif'")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}
	if search := c.Query("q"); search != "" {
		q = q.Where("name ILIKE ? OR member_code ILIKE ? OR phone ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	q.Select("id, member_code, name, tier").Limit(50).Find(&list)
	c.JSON(http.StatusOK, list)
}

// GetCustomerTransactions implements docs/pelanggan.md GET /customers/:id/transactions
func GetCustomerTransactions(c *gin.Context) {
	id := c.Param("id")
	var list []models.SalesTransaction
	q := database.DB.Preload("SalesItems").Preload("Payments").Where("customer_id = ?", id)
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		q = q.Where("transaction_date >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		q = q.Where("transaction_date <= ?", dateTo)
	}

	if c.Query("page") != "" || c.Query("per_page") != "" {
		var total int64
		q.Count(&total)
		page, perPage, offset := GetPagination(c)
		q.Order("transaction_date desc").Limit(perPage).Offset(offset).Find(&list)
		RespondPaginated(c, list, page, perPage, total)
		return
	}

	q.Order("transaction_date desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

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
		RespondError(c, http.StatusNotFound, "Customer not found")
		return
	}
	cache := recalcCustomerTier(cu.ID)
	RespondSuccess(c, cache, "Customer metrics refreshed")
}
