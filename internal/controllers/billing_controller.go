package controllers

import (
	"fmt"
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

// checkPlanLimit implements docs/subscription-billing.md feature gating. Companies without an
// active subscription record are not gated (dev/test convenience).
func checkPlanLimit(companyID uint, kind string) (bool, string) {
	var sub models.Subscription
	if err := database.DB.Preload("Plan").Where("company_id = ?", companyID).First(&sub).Error; err != nil {
		return true, ""
	}
	plan := sub.Plan
	switch kind {
	case "outlets":
		if plan.MaxOutlets == nil {
			return true, ""
		}
		var count int64
		database.DB.Model(&models.Store{}).Where("company_id = ?", companyID).Count(&count)
		if int(count) >= *plan.MaxOutlets {
			return false, fmt.Sprintf("plan %s hanya mendukung maksimal %d outlet", plan.Name, *plan.MaxOutlets)
		}
	case "users":
		if plan.MaxUsers == nil {
			return true, ""
		}
		var count int64
		database.DB.Model(&models.Employee{}).Where("company_id = ?", companyID).Count(&count)
		if int(count) >= *plan.MaxUsers {
			return false, fmt.Sprintf("plan %s hanya mendukung maksimal %d user", plan.Name, *plan.MaxUsers)
		}
	case "products":
		if plan.MaxProducts == nil {
			return true, ""
		}
		var count int64
		database.DB.Model(&models.Product{}).Where("company_id = ?", companyID).Count(&count)
		if int(count) >= *plan.MaxProducts {
			return false, fmt.Sprintf("plan %s hanya mendukung maksimal %d produk", plan.Name, *plan.MaxProducts)
		}
	case "transactions":
		if plan.MaxTransactionsPerMonth == nil {
			return true, ""
		}
		period := time.Now().Format("2006-01")
		var q models.PlanQuota
		database.DB.Where("company_id = ? AND period = ?", companyID, period).First(&q)
		if q.TransactionsCount >= *plan.MaxTransactionsPerMonth {
			return false, fmt.Sprintf("plan %s sudah mencapai batas %d transaksi bulan ini", plan.Name, *plan.MaxTransactionsPerMonth)
		}
	}
	return true, ""
}

// incrementTransactionQuota implements docs/subscription-billing.md plan_quotas usage counter.
func incrementTransactionQuota(companyID uint) {
	period := time.Now().Format("2006-01")
	var q models.PlanQuota
	if err := database.DB.Where("company_id = ? AND period = ?", companyID, period).First(&q).Error; err != nil {
		database.DB.Create(&models.PlanQuota{CompanyID: companyID, Period: period, TransactionsCount: 1, UpdatedAt: time.Now()})
		return
	}
	database.DB.Model(&q).Updates(map[string]interface{}{"transactions_count": q.TransactionsCount + 1, "updated_at": time.Now()})
}

// GetUsage implements docs/subscription-billing.md GET /billing/usage
func GetUsage(c *gin.Context) {
	companyID := c.Query("company_id")
	period := time.Now().Format("2006-01")
	var q models.PlanQuota
	database.DB.Where("company_id = ? AND period = ?", companyID, period).First(&q)

	var sub models.Subscription
	database.DB.Preload("Plan").Where("company_id = ?", companyID).First(&sub)

	var storeCount, userCount, productCount int64
	database.DB.Model(&models.Store{}).Where("company_id = ?", companyID).Count(&storeCount)
	database.DB.Model(&models.Employee{}).Where("company_id = ?", companyID).Count(&userCount)
	database.DB.Model(&models.Product{}).Where("company_id = ?", companyID).Count(&productCount)

	c.JSON(http.StatusOK, gin.H{
		"period":             period,
		"transactions_count": q.TransactionsCount,
		"stores":             storeCount,
		"users":              userCount,
		"products":           productCount,
		"plan":               sub.Plan,
	})
}

func GetPlans(c *gin.Context) {
	var list []models.Plan
	database.DB.Where("is_active = true").Order("sort_order").Find(&list)
	c.JSON(http.StatusOK, list)
}

func GetSubscription(c *gin.Context) {
	companyID := c.Query("company_id")
	var sub models.Subscription
	if err := database.DB.Preload("Plan").Where("company_id = ?", companyID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	c.JSON(http.StatusOK, sub)
}

type CreateSubscriptionRequest struct {
	CompanyID    uint   `json:"company_id" binding:"required"`
	PlanID       uint   `json:"plan_id" binding:"required"`
	BillingCycle string `json:"billing_cycle" binding:"required,oneof=monthly yearly"`
}

// CreateSubscription implements POST /billing/subscription (subscribe)
func CreateSubscription(c *gin.Context) {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var plan models.Plan
	if err := database.DB.First(&plan, req.PlanID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan not found"})
		return
	}
	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0)
	if req.BillingCycle == "yearly" {
		periodEnd = now.AddDate(1, 0, 0)
	}

	var sub models.Subscription
	err := database.DB.Where("company_id = ?", req.CompanyID).First(&sub).Error
	if err != nil {
		sub = models.Subscription{
			CompanyID:          req.CompanyID,
			PlanID:             req.PlanID,
			Status:             "active",
			BillingCycle:       req.BillingCycle,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   periodEnd,
		}
		database.DB.Create(&sub)
	} else {
		sub.PlanID = req.PlanID
		sub.BillingCycle = req.BillingCycle
		sub.Status = "active"
		sub.CurrentPeriodStart = now
		sub.CurrentPeriodEnd = periodEnd
		database.DB.Save(&sub)
	}
	logAudit(c, req.CompanyID, "subscription_change", "billing", sub.ID, plan.Name, nil)
	c.JSON(http.StatusOK, sub)
}

type ChangePlanRequest struct {
	CompanyID uint `json:"company_id" binding:"required"`
	PlanID    uint `json:"plan_id" binding:"required"`
}

func ChangeSubscriptionPlan(c *gin.Context) {
	var req ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var sub models.Subscription
	if err := database.DB.Where("company_id = ?", req.CompanyID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	sub.PlanID = req.PlanID
	database.DB.Save(&sub)
	logAudit(c, req.CompanyID, "subscription_change", "billing", sub.ID, "", nil)
	c.JSON(http.StatusOK, sub)
}

type CancelSubscriptionRequest struct {
	CompanyID   uint `json:"company_id" binding:"required"`
	AtPeriodEnd bool `json:"at_period_end"`
}

func CancelSubscription(c *gin.Context) {
	var req CancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var sub models.Subscription
	if err := database.DB.Where("company_id = ?", req.CompanyID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	now := time.Now()
	if req.AtPeriodEnd {
		sub.CancelAtPeriodEnd = true
	} else {
		sub.Status = "cancelled"
		sub.CancelledAt = &now
	}
	database.DB.Save(&sub)
	logAudit(c, req.CompanyID, "subscription_change", "billing", sub.ID, "cancel", nil)
	c.JSON(http.StatusOK, sub)
}

func GetInvoices(c *gin.Context) {
	var list []models.Invoice
	q := database.DB.Model(&models.Invoice{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Order("due_date desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

func GetInvoice(c *gin.Context) {
	id := c.Param("id")
	var inv models.Invoice
	if err := database.DB.First(&inv, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
		return
	}
	c.JSON(http.StatusOK, inv)
}

type PayInvoiceRequest struct {
	Method      string  `json:"method" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	ExternalRef string  `json:"external_ref"`
}

// PayInvoice implements POST /billing/invoices/:id/pay (idempotent by external_ref)
func PayInvoice(c *gin.Context) {
	id := c.Param("id")
	var inv models.Invoice
	if err := database.DB.First(&inv, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
		return
	}
	var req PayInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ExternalRef != "" {
		var existing models.BillingPayment
		if err := database.DB.Where("external_ref = ?", req.ExternalRef).First(&existing).Error; err == nil {
			c.JSON(http.StatusOK, gin.H{"message": "payment already processed", "payment": existing})
			return
		}
	}
	now := time.Now()
	payment := models.BillingPayment{
		CompanyID:   inv.CompanyID,
		InvoiceID:   inv.ID,
		Method:      req.Method,
		Amount:      req.Amount,
		ExternalRef: req.ExternalRef,
		Status:      "success",
		PaidAt:      &now,
	}
	database.DB.Create(&payment)
	inv.Status = "paid"
	inv.PaidAt = &now
	database.DB.Save(&inv)
	logAudit(c, inv.CompanyID, "pay", "billing", inv.ID, inv.InvoiceNo, nil)
	c.JSON(http.StatusOK, gin.H{"invoice": inv, "payment": payment})
}

// GenerateInvoice creates a new invoice for a subscription's current period (manual trigger; cron in production).
func GenerateInvoice(c *gin.Context) {
	companyID := c.Query("company_id")
	var sub models.Subscription
	if err := database.DB.Preload("Plan").Where("company_id = ?", companyID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}
	amount := sub.Plan.PriceMonthly
	if sub.BillingCycle == "yearly" {
		amount = sub.Plan.PriceYearly
	}
	tax := amount * 0.11
	total := amount + tax
	var count int64
	database.DB.Model(&models.Invoice{}).Count(&count)
	inv := models.Invoice{
		CompanyID:      sub.CompanyID,
		SubscriptionID: sub.ID,
		InvoiceNo:      fmt.Sprintf("INV-%s-%05d", time.Now().Format("2006-01"), count+1),
		PeriodStart:    sub.CurrentPeriodStart,
		PeriodEnd:      sub.CurrentPeriodEnd,
		Subtotal:       amount,
		Tax:            tax,
		Total:          total,
		Status:         "open",
		DueDate:        sub.CurrentPeriodEnd,
	}
	database.DB.Create(&inv)
	c.JSON(http.StatusCreated, inv)
}

// ---- Payment methods (docs/subscription-billing.md) ----

func GetPaymentMethods(c *gin.Context) {
	var list []models.TenantPaymentMethod
	q := database.DB.Model(&models.TenantPaymentMethod{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

type CreatePaymentMethodRequest struct {
	CompanyID  uint   `json:"company_id" binding:"required"`
	Type       string `json:"type" binding:"required"`
	Provider   string `json:"provider"`
	ExternalID string `json:"external_id"`
	MaskedInfo string `json:"masked_info"`
	IsDefault  bool   `json:"is_default"`
}

func CreatePaymentMethod(c *gin.Context) {
	var req CreatePaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.IsDefault {
		database.DB.Model(&models.TenantPaymentMethod{}).Where("company_id = ?", req.CompanyID).Update("is_default", false)
	}
	pm := models.TenantPaymentMethod{
		CompanyID:  req.CompanyID,
		Type:       req.Type,
		Provider:   req.Provider,
		ExternalID: req.ExternalID,
		MaskedInfo: req.MaskedInfo,
		IsDefault:  req.IsDefault,
	}
	database.DB.Create(&pm)
	c.JSON(http.StatusCreated, pm)
}

func DeletePaymentMethod(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.TenantPaymentMethod{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment method"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Payment method deleted"})
}

// ---- Coupons (docs/subscription-billing.md) ----

type CreateCouponRequest struct {
	Code           string     `json:"code" binding:"required"`
	Type           string     `json:"type" binding:"required,oneof=percent amount"`
	Value          float64    `json:"value" binding:"required,gt=0"`
	MaxRedemptions *int       `json:"max_redemptions"`
	ValidFrom      *time.Time `json:"valid_from"`
	ValidUntil     *time.Time `json:"valid_until"`
}

func CreateCoupon(c *gin.Context) {
	var req CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	coupon := models.Coupon{
		Code: req.Code, Type: req.Type, Value: req.Value,
		MaxRedemptions: req.MaxRedemptions, ValidFrom: req.ValidFrom, ValidUntil: req.ValidUntil,
		IsActive: true,
	}
	if err := database.DB.Create(&coupon).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create coupon"})
		return
	}
	c.JSON(http.StatusCreated, coupon)
}

func GetCoupons(c *gin.Context) {
	var list []models.Coupon
	database.DB.Find(&list)
	c.JSON(http.StatusOK, list)
}

// ValidateCoupon checks whether a coupon code is redeemable.
func ValidateCoupon(c *gin.Context) {
	code := c.Query("code")
	var coupon models.Coupon
	if err := database.DB.Where("code = ? AND is_active = true", code).First(&coupon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "coupon tidak ditemukan atau tidak aktif"})
		return
	}
	if coupon.MaxRedemptions != nil && coupon.RedeemedCount >= *coupon.MaxRedemptions {
		c.JSON(http.StatusBadRequest, gin.H{"error": "coupon sudah mencapai batas penggunaan"})
		return
	}
	c.JSON(http.StatusOK, coupon)
}
