package controllers

import (
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type OpenShiftRequest struct {
	CompanyID   uint    `json:"company_id" binding:"required"`
	StoreID     uint    `json:"store_id" binding:"required"`
	EmployeeID  uint    `json:"employee_id" binding:"required"`
	OpeningCash float64 `json:"opening_cash"`
}

// OpenShift implements docs/kasir.md POST /shifts/open
func OpenShift(c *gin.Context) {
	var req OpenShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var openCount int64
	database.DB.Model(&models.Shift{}).Where("employee_id = ? AND status = 'open'", req.EmployeeID).Count(&openCount)
	if openCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "kasir masih memiliki shift yang terbuka"})
		return
	}
	shift := models.Shift{
		CompanyID:   req.CompanyID,
		StoreID:     req.StoreID,
		EmployeeID:  req.EmployeeID,
		OpeningCash: req.OpeningCash,
		Status:      "open",
	}
	database.DB.Create(&shift)
	shift.Code = "S-" + padNumber(shift.ID+1000)
	database.DB.Model(&shift).Update("code", shift.Code)
	logAudit(c, shift.CompanyID, "open_shift", "shift", shift.ID, shift.Code, nil)
	c.JSON(http.StatusCreated, shift)
}

type CloseShiftRequest struct {
	ActualCash float64 `json:"actual_cash" binding:"required"`
	Notes      string  `json:"notes"`
}

// CloseShift implements docs/kasir.md POST /shifts/:id/close
func CloseShift(c *gin.Context) {
	id := c.Param("id")
	var shift models.Shift
	if err := database.DB.First(&shift, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shift not found"})
		return
	}
	var req CloseShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	shift.ClosedAt = &now
	shift.ActualCash = &req.ActualCash
	shift.Notes = req.Notes
	shift.Status = "closed"
	database.DB.Save(&shift)
	logAudit(c, shift.CompanyID, "close_shift", "shift", shift.ID, shift.Code, nil)
	c.JSON(http.StatusOK, shift)
}

type ShiftCashMovementRequest struct {
	Direction string  `json:"direction" binding:"required,oneof=in out"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	Reason    string  `json:"reason" binding:"required"`
	Note      string  `json:"note"`
	CreatedBy uint    `json:"created_by"`
}

// ShiftCashMovementHandler implements docs/kasir.md POST /shifts/:id/cash-movement
func ShiftCashMovementHandler(c *gin.Context) {
	id := c.Param("id")
	var shift models.Shift
	if err := database.DB.First(&shift, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shift not found"})
		return
	}
	var req ShiftCashMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Direction == "in" {
		shift.CashIn += req.Amount
	} else {
		shift.CashOut += req.Amount
	}
	database.DB.Save(&shift)
	movement := models.ShiftCashMovement{
		ShiftID:   shift.ID,
		CompanyID: shift.CompanyID,
		Direction: req.Direction,
		Amount:    req.Amount,
		Reason:    req.Reason,
		Note:      req.Note,
		CreatedBy: req.CreatedBy,
	}
	database.DB.Create(&movement)
	c.JSON(http.StatusCreated, movement)
}

// GetActiveShift implements docs/kasir.md GET /shifts/active
func GetActiveShift(c *gin.Context) {
	employeeID := c.Query("employee_id")
	var shift models.Shift
	q := database.DB.Where("status = 'open'")
	if employeeID != "" {
		q = q.Where("employee_id = ?", employeeID)
	}
	if err := q.Order("opened_at desc").First(&shift).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active shift"})
		return
	}
	c.JSON(http.StatusOK, shift)
}

func GetShifts(c *gin.Context) {
	var list []models.Shift
	q := database.DB.Model(&models.Shift{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Order("opened_at desc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shifts"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// ---- Vouchers ----

type CreateVoucherRequest struct {
	CompanyID   uint       `json:"company_id" binding:"required"`
	Code        string     `json:"code" binding:"required"`
	Label       string     `json:"label"`
	Type        string     `json:"type" binding:"required,oneof=percent amount"`
	Value       float64    `json:"value" binding:"required,gt=0"`
	MinSpend    float64    `json:"min_spend"`
	MaxDiscount *float64   `json:"max_discount"`
	UsageLimit  *int       `json:"usage_limit"`
	ValidFrom   *time.Time `json:"valid_from"`
	ValidUntil  *time.Time `json:"valid_until"`
}

func CreateVoucher(c *gin.Context) {
	var req CreateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v := models.Voucher{
		CompanyID:   req.CompanyID,
		Code:        req.Code,
		Label:       req.Label,
		Type:        req.Type,
		Value:       req.Value,
		MinSpend:    req.MinSpend,
		MaxDiscount: req.MaxDiscount,
		UsageLimit:  req.UsageLimit,
		ValidFrom:   req.ValidFrom,
		ValidUntil:  req.ValidUntil,
		IsActive:    true,
	}
	if err := database.DB.Create(&v).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create voucher"})
		return
	}
	c.JSON(http.StatusCreated, v)
}

func GetVouchers(c *gin.Context) {
	var list []models.Voucher
	q := database.DB.Model(&models.Voucher{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if code := c.Query("code"); code != "" {
		q = q.Where("code = ?", code)
	}
	if err := q.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch vouchers"})
		return
	}
	c.JSON(http.StatusOK, list)
}

// ValidateVoucher implements docs/kasir.md GET /cashier/vouchers?code=
func ValidateVoucher(c *gin.Context) {
	code := c.Query("code")
	var v models.Voucher
	if err := database.DB.Where("code = ? AND is_active = true", code).First(&v).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "voucher tidak ditemukan atau tidak aktif"})
		return
	}
	now := time.Now()
	if v.ValidUntil != nil && now.After(*v.ValidUntil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voucher sudah kedaluwarsa"})
		return
	}
	if v.UsageLimit != nil && v.UsedCount >= *v.UsageLimit {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voucher sudah mencapai batas penggunaan"})
		return
	}
	c.JSON(http.StatusOK, v)
}

func UpdateVoucher(c *gin.Context) {
	id := c.Param("id")
	var v models.Voucher
	if err := database.DB.First(&v, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Voucher not found"})
		return
	}
	var req CreateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v.Label = req.Label
	v.Type = req.Type
	v.Value = req.Value
	v.MinSpend = req.MinSpend
	v.MaxDiscount = req.MaxDiscount
	v.UsageLimit = req.UsageLimit
	v.ValidFrom = req.ValidFrom
	v.ValidUntil = req.ValidUntil
	database.DB.Save(&v)
	c.JSON(http.StatusOK, v)
}

func DeleteVoucher(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Voucher{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete voucher"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Voucher deleted"})
}

// ---- Promotions ----

type CreatePromotionRequest struct {
	CompanyID  uint       `json:"company_id" binding:"required"`
	Name       string     `json:"name" binding:"required"`
	Type       string     `json:"type" binding:"required,oneof=bogo percent amount"`
	Value      float64    `json:"value"`
	ProductIDs []uint     `json:"product_ids"`
	CategoryID *uint      `json:"category_id"`
	MinQty     int        `json:"min_qty"`
	ValidFrom  *time.Time `json:"valid_from"`
	ValidUntil *time.Time `json:"valid_until"`
}

func CreatePromotion(c *gin.Context) {
	var req CreatePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	minQty := req.MinQty
	if minQty == 0 {
		minQty = 1
	}
	p := models.Promotion{
		CompanyID:  req.CompanyID,
		Name:       req.Name,
		Type:       req.Type,
		Value:      req.Value,
		ProductIDs: encodeIDs(req.ProductIDs),
		CategoryID: req.CategoryID,
		MinQty:     minQty,
		ValidFrom:  req.ValidFrom,
		ValidUntil: req.ValidUntil,
		IsActive:   true,
	}
	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create promotion"})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func GetPromotions(c *gin.Context) {
	var list []models.Promotion
	q := database.DB.Model(&models.Promotion{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if active := c.Query("active"); active == "true" {
		q = q.Where("is_active = true")
	}
	if err := q.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch promotions"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func UpdatePromotion(c *gin.Context) {
	id := c.Param("id")
	var p models.Promotion
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promotion not found"})
		return
	}
	var req CreatePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.Name = req.Name
	p.Type = req.Type
	p.Value = req.Value
	p.ProductIDs = encodeIDs(req.ProductIDs)
	p.CategoryID = req.CategoryID
	if req.MinQty > 0 {
		p.MinQty = req.MinQty
	}
	p.ValidFrom = req.ValidFrom
	p.ValidUntil = req.ValidUntil
	database.DB.Save(&p)
	c.JSON(http.StatusOK, p)
}

func DeletePromotion(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Promotion{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete promotion"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Promotion deleted"})
}
