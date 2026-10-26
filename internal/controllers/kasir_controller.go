package controllers

import (
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

type OpenShiftRequest struct {
	CompanyID   uint    `json:"company_id"`
	StoreID     uint    `json:"store_id" binding:"required"`
	EmployeeID  uint    `json:"employee_id" binding:"required"`
	OpeningCash float64 `json:"opening_cash"`
}

// OpenShift implements docs/kasir.md POST /shifts/open
func OpenShift(c *gin.Context) {
	var req OpenShiftRequest
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

	var openCount int64
	database.DB.Model(&models.Shift{}).Where("employee_id = ? AND status = 'open'", req.EmployeeID).Count(&openCount)
	if openCount > 0 {
		RespondError(c, http.StatusConflict, "kasir masih memiliki shift yang terbuka")
		return
	}
	shift := models.Shift{
		CompanyID:   companyID,
		StoreID:     req.StoreID,
		EmployeeID:  req.EmployeeID,
		OpeningCash: req.OpeningCash,
		Status:      "open",
	}
	database.DB.Create(&shift)
	shift.Code = "S-" + padNumber(shift.ID+1000)
	database.DB.Model(&shift).Update("code", shift.Code)
	logAudit(c, shift.CompanyID, "open_shift", "shift", shift.ID, shift.Code, nil)
	RespondCreated(c, shift, "Shift opened successfully")
}

type CloseShiftRequest struct {
	ActualCash float64 `json:"actual_cash" binding:"required"`
	Notes      string  `json:"notes"`
}

// CloseShift implements docs/kasir.md POST /shifts/:id/close with automatic cash reconciliation.
func CloseShift(c *gin.Context) {
	id := c.Param("id")
	var shift models.Shift
	if err := database.DB.First(&shift, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Shift not found")
		return
	}
	var req CloseShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 1. Calculate sales during this shift from sales_transactions and payments
	var sales []models.SalesTransaction
	database.DB.Preload("Payments").Where("shift_id = ? AND status <> 'void'", shift.ID).Find(&sales)

	var cashSales, nonCashSales, totalSales float64
	for _, s := range sales {
		totalSales += s.TotalAmount
		if len(s.Payments) > 0 {
			for _, p := range s.Payments {
				if p.Method == "cash" {
					cashSales += p.Amount
				} else {
					nonCashSales += p.Amount
				}
			}
		} else {
			if s.PaymentMethod == "cash" {
				cashSales += s.TotalAmount
			} else {
				nonCashSales += s.TotalAmount
			}
		}
	}

	// 2. Expected cash = OpeningCash + CashSales + CashIn - CashOut
	expectedCash := shift.OpeningCash + cashSales + shift.CashIn - shift.CashOut
	discrepancy := req.ActualCash - expectedCash

	now := time.Now()
	shift.ClosedAt = &now
	shift.ActualCash = &req.ActualCash
	shift.ExpectedCash = expectedCash
	shift.Discrepancy = &discrepancy
	shift.CashSales = cashSales
	shift.NonCashSales = nonCashSales
	shift.TotalSales = totalSales
	shift.TotalTransactions = len(sales)
	shift.Notes = req.Notes
	shift.Status = "closed"

	if err := database.DB.Save(&shift).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to close shift: "+err.Error())
		return
	}

	logAudit(c, shift.CompanyID, "close_shift", "shift", shift.ID, shift.Code, gin.H{
		"actual_cash":   req.ActualCash,
		"expected_cash": expectedCash,
		"discrepancy":   discrepancy,
		"total_sales":   totalSales,
	})

	RespondSuccess(c, shift, "Shift closed successfully with cash reconciliation")
}

// GetShiftSummary returns a detailed financial & reconciliation summary of a cashier shift.
func GetShiftSummary(c *gin.Context) {
	id := c.Param("id")
	var shift models.Shift
	if err := database.DB.Preload("Store").Preload("Employee").First(&shift, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Shift not found")
		return
	}

	var sales []models.SalesTransaction
	database.DB.Preload("Payments").Preload("SalesItems").Where("shift_id = ? AND status <> 'void'", shift.ID).Find(&sales)

	type MethodSummary struct {
		Method string  `json:"method"`
		Amount float64 `json:"amount"`
		Count  int     `json:"count"`
	}
	methodMap := make(map[string]*MethodSummary)
	var cashSales, nonCashSales, totalSales float64

	for _, s := range sales {
		totalSales += s.TotalAmount
		if len(s.Payments) > 0 {
			for _, p := range s.Payments {
				if methodMap[p.Method] == nil {
					methodMap[p.Method] = &MethodSummary{Method: p.Method}
				}
				methodMap[p.Method].Amount += p.Amount
				methodMap[p.Method].Count++

				if p.Method == "cash" {
					cashSales += p.Amount
				} else {
					nonCashSales += p.Amount
				}
			}
		} else {
			m := s.PaymentMethod
			if m == "" {
				m = "cash"
			}
			if methodMap[m] == nil {
				methodMap[m] = &MethodSummary{Method: m}
			}
			methodMap[m].Amount += s.TotalAmount
			methodMap[m].Count++

			if m == "cash" {
				cashSales += s.TotalAmount
			} else {
				nonCashSales += s.TotalAmount
			}
		}
	}

	methodList := make([]MethodSummary, 0, len(methodMap))
	for _, v := range methodMap {
		methodList = append(methodList, *v)
	}

	var movements []models.ShiftCashMovement
	database.DB.Where("shift_id = ?", shift.ID).Order("created_at asc").Find(&movements)

	expectedCash := shift.OpeningCash + cashSales + shift.CashIn - shift.CashOut
	var discrepancy *float64
	if shift.ActualCash != nil {
		diff := *shift.ActualCash - expectedCash
		discrepancy = &diff
	}

	summary := gin.H{
		"shift": gin.H{
			"id":                 shift.ID,
			"code":               shift.Code,
			"status":             shift.Status,
			"store_name":         shift.Store.Name,
			"cashier_name":       shift.Employee.Name,
			"opened_at":          shift.OpenedAt,
			"closed_at":          shift.ClosedAt,
			"opening_cash":       shift.OpeningCash,
			"cash_in":            shift.CashIn,
			"cash_out":           shift.CashOut,
			"cash_sales":         cashSales,
			"non_cash_sales":     nonCashSales,
			"total_sales":        totalSales,
			"expected_cash":      expectedCash,
			"actual_cash":        shift.ActualCash,
			"discrepancy":        discrepancy,
			"total_transactions": len(sales),
		},
		"payments_by_method": methodList,
		"cash_movements":     movements,
	}

	RespondSuccess(c, summary)
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
		RespondError(c, http.StatusNotFound, "Shift not found")
		return
	}
	var req ShiftCashMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
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
	RespondCreated(c, movement)
}

// GetActiveShift implements docs/kasir.md GET /shifts/active
func GetActiveShift(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	employeeID := c.Query("employee_id")
	var shift models.Shift
	q := database.DB.Where("status = 'open'")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	}
	if employeeID != "" {
		q = q.Where("employee_id = ?", employeeID)
	}
	if err := q.Order("opened_at desc").First(&shift).Error; err != nil {
		RespondError(c, http.StatusNotFound, "no active shift")
		return
	}
	c.JSON(http.StatusOK, shift)
}

func GetShifts(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	q := database.DB.Model(&models.Shift{}).Preload("Store").Preload("Employee")
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if employeeID := c.Query("employee_id"); employeeID != "" {
		q = q.Where("employee_id = ?", employeeID)
	}

	if c.Query("page") != "" || c.Query("per_page") != "" {
		var total int64
		q.Count(&total)
		page, perPage, offset := GetPagination(c)
		var list []models.Shift
		if err := q.Order("opened_at desc").Limit(perPage).Offset(offset).Find(&list).Error; err != nil {
			RespondError(c, http.StatusInternalServerError, "Failed to fetch shifts")
			return
		}
		RespondPaginated(c, list, page, perPage, total)
		return
	}

	var list []models.Shift
	if err := q.Order("opened_at desc").Find(&list).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to fetch shifts")
		return
	}
	c.JSON(http.StatusOK, list)
}

// ---- Vouchers ----

type CreateVoucherRequest struct {
	CompanyID   uint       `json:"company_id"`
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
	v := models.Voucher{
		CompanyID:   companyID,
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
		RespondError(c, http.StatusInternalServerError, "Failed to create voucher")
		return
	}
	RespondCreated(c, v)
}

func GetVouchers(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	var list []models.Voucher
	q := database.DB.Model(&models.Voucher{})
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}
	if code := c.Query("code"); code != "" {
		q = q.Where("code = ?", code)
	}
	if err := q.Find(&list).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to fetch vouchers")
		return
	}
	c.JSON(http.StatusOK, list)
}

// ValidateVoucher implements docs/kasir.md GET /cashier/vouchers?code=
func ValidateVoucher(c *gin.Context) {
	code := c.Query("code")
	var v models.Voucher
	if err := database.DB.Where("code = ? AND is_active = true", code).First(&v).Error; err != nil {
		RespondError(c, http.StatusNotFound, "voucher tidak ditemukan atau tidak aktif")
		return
	}
	now := time.Now()
	if v.ValidUntil != nil && now.After(*v.ValidUntil) {
		RespondError(c, http.StatusBadRequest, "voucher sudah kedaluwarsa")
		return
	}
	if v.UsageLimit != nil && v.UsedCount >= *v.UsageLimit {
		RespondError(c, http.StatusBadRequest, "voucher sudah mencapai batas penggunaan")
		return
	}
	c.JSON(http.StatusOK, v)
}

func UpdateVoucher(c *gin.Context) {
	id := c.Param("id")
	var v models.Voucher
	if err := database.DB.First(&v, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Voucher not found")
		return
	}
	var req CreateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
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
	RespondSuccess(c, v)
}

func DeleteVoucher(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Voucher{}, id).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete voucher")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Voucher deleted"})
}

// ---- Promotions ----

type CreatePromotionRequest struct {
	CompanyID  uint       `json:"company_id"`
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
	minQty := req.MinQty
	if minQty == 0 {
		minQty = 1
	}
	p := models.Promotion{
		CompanyID:  companyID,
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
		RespondError(c, http.StatusInternalServerError, "Failed to create promotion")
		return
	}
	RespondCreated(c, p)
}

func GetPromotions(c *gin.Context) {
	companyID := getAuthCompanyID(c)
	var list []models.Promotion
	q := database.DB.Model(&models.Promotion{})
	if companyID > 0 {
		q = q.Where("company_id = ?", companyID)
	} else if qCID := c.Query("company_id"); qCID != "" {
		q = q.Where("company_id = ?", qCID)
	}
	if active := c.Query("active"); active == "true" {
		q = q.Where("is_active = true")
	}
	if err := q.Find(&list).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to fetch promotions")
		return
	}
	c.JSON(http.StatusOK, list)
}

func UpdatePromotion(c *gin.Context) {
	id := c.Param("id")
	var p models.Promotion
	if err := database.DB.First(&p, id).Error; err != nil {
		RespondError(c, http.StatusNotFound, "Promotion not found")
		return
	}
	var req CreatePromotionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
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
	RespondSuccess(c, p)
}

func DeletePromotion(c *gin.Context) {
	id := c.Param("id")
	if err := database.DB.Delete(&models.Promotion{}, id).Error; err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete promotion")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Promotion deleted"})
}
