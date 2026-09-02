package controllers

import (
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
)

// GetDashboardSummary implements docs/dashboard.md GET /dashboard/summary (simplified, single-tenant scope).
func GetDashboardSummary(c *gin.Context) {
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	var revenue float64
	var orders int64
	q.Count(&orders)
	q.Select("COALESCE(SUM(total_amount),0)").Scan(&revenue)
	avgTicket := 0.0
	if orders > 0 {
		avgTicket = revenue / float64(orders)
	}
	cogs := computeRealCOGS(c)
	if cogs == 0 {
		cogs = revenue * 0.6 // fallback estimate when sale_items/cost data is unavailable
	}
	net := revenue - cogs

	var totalSKU int64
	database.DB.Model(&models.Product{}).Count(&totalSKU)
	var lowStock, outStock int64
	database.DB.Model(&models.Stock{}).Where("quantity > 0 AND quantity < 10").Count(&lowStock)
	database.DB.Model(&models.Stock{}).Where("quantity = 0").Count(&outStock)
	var stockValue float64
	database.DB.Table("stocks").
		Joins("JOIN products ON products.id = stocks.product_id").
		Select("COALESCE(SUM(stocks.quantity * products.cost),0)").Scan(&stockValue)

	c.JSON(http.StatusOK, gin.H{
		"totals": gin.H{"revenue": revenue, "orders": orders, "avg_ticket": avgTicket},
		"profit": gin.H{"gross": revenue, "cogs": cogs, "net": net},
		"inventory": gin.H{
			"total_sku":   totalSKU,
			"low_stock":   lowStock,
			"out_stock":   outStock,
			"stock_value": stockValue,
		},
	})
}

func GetDashboardAlerts(c *gin.Context) {
	var list []models.Alert
	q := database.DB.Model(&models.Alert{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if severity := c.Query("severity"); severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if kind := c.Query("kind"); kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	} else {
		q = q.Where("status = 'open'")
	}
	q.Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

func AcknowledgeAlert(c *gin.Context) {
	id := c.Param("id")
	var alert models.Alert
	if err := database.DB.First(&alert, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}
	now := time.Now()
	alert.Status = "acknowledged"
	alert.AckAt = &now
	database.DB.Save(&alert)
	c.JSON(http.StatusOK, alert)
}

func ResolveAlert(c *gin.Context) {
	id := c.Param("id")
	var alert models.Alert
	if err := database.DB.First(&alert, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}
	alert.Status = "resolved"
	database.DB.Save(&alert)
	c.JSON(http.StatusOK, alert)
}

// GetDashboardShiftsActive implements GET /dashboard/shifts/active
func GetDashboardShiftsActive(c *gin.Context) {
	var list []models.Shift
	q := database.DB.Where("status = 'open'")
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

// GetPaymentMix implements docs/dashboard.md payment_mix_cache (computed live).
func GetPaymentMix(c *gin.Context) {
	type row struct {
		Method string  `json:"method"`
		Amount float64 `json:"amount"`
		Count  int64   `json:"count"`
	}
	var rows []row
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	q.Select("payment_method as method, COALESCE(SUM(total_amount),0) as amount, COUNT(*) as count").
		Group("payment_method").Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// GetSaasMetrics implements docs/dashboard.md GET /dashboard/saas-metrics (superadmin only; here
// simplified to aggregate across all companies since there is no separate superadmin scope yet).
func GetSaasMetrics(c *gin.Context) {
	var tenants int64
	database.DB.Model(&models.Company{}).Count(&tenants)
	var stores int64
	database.DB.Model(&models.Store{}).Count(&stores)

	type planRow struct {
		Plan  string  `json:"plan"`
		Count int64   `json:"count"`
		MRR   float64 `json:"mrr"`
	}
	var rows []planRow
	database.DB.Table("subscriptions").
		Select("plans.name as plan, COUNT(*) as count, COALESCE(SUM(plans.price_monthly),0) as mrr").
		Joins("JOIN plans ON plans.id = subscriptions.plan_id").
		Where("subscriptions.status IN ('active','trial')").
		Group("plans.name").Scan(&rows)

	var mrr float64
	for _, r := range rows {
		mrr += r.MRR
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants": tenants,
		"stores":  stores,
		"mrr":     mrr,
		"arr":     mrr * 12,
		"by_plan": rows,
	})
}

// ---- Devices (docs/dashboard.md) ----

type CreateDeviceRequest struct {
	CompanyID uint   `json:"company_id" binding:"required"`
	StoreID   uint   `json:"store_id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type" binding:"required"`
	SerialNo  string `json:"serial_no"`
}

func CreateDevice(c *gin.Context) {
	var req CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	device := models.Device{
		CompanyID: req.CompanyID, StoreID: req.StoreID, Name: req.Name,
		Type: req.Type, SerialNo: req.SerialNo, Status: "offline",
	}
	if err := database.DB.Create(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device"})
		return
	}
	c.JSON(http.StatusCreated, device)
}

func GetDevices(c *gin.Context) {
	var list []models.Device
	q := database.DB.Model(&models.Device{})
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

// DeviceHeartbeat marks a device online and updates last_heartbeat_at (called periodically by the device).
func DeviceHeartbeat(c *gin.Context) {
	id := c.Param("id")
	var device models.Device
	if err := database.DB.First(&device, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}
	now := time.Now()
	device.Status = "online"
	device.LastHeartbeatAt = &now
	database.DB.Save(&device)
	c.JSON(http.StatusOK, device)
}

// GenerateDashboardAlerts implements a lightweight rule engine (docs/dashboard.md) as a manual
// trigger: shift overdue (>12h open), stock <= min (qty < 10), device offline > 5 min.
// In production this runs on a schedule; exposed here since there is no cron infrastructure.
func GenerateDashboardAlerts(c *gin.Context) {
	companyID := c.Query("company_id")
	created := 0

	var overdueShifts []models.Shift
	database.DB.Where("status = 'open' AND opened_at < ?", time.Now().Add(-12*time.Hour)).Find(&overdueShifts)
	for _, s := range overdueShifts {
		database.DB.Create(&models.Alert{
			CompanyID: s.CompanyID, StoreID: &s.StoreID, Kind: "shift", Severity: "warning",
			Title: "Shift overdue", Detail: "Shift belum ditutup > 12 jam", EntityType: "shift", EntityID: &s.ID,
		})
		created++
	}

	var lowStocks []models.Stock
	database.DB.Where("quantity < 10").Find(&lowStocks)
	for _, s := range lowStocks {
		sev := "warning"
		if s.Quantity == 0 {
			sev = "critical"
		}
		database.DB.Create(&models.Alert{
			Kind: "stock", Severity: sev, Title: "Stok menipis/habis",
			Detail: "Produk perlu restock", EntityType: "product", EntityID: &s.ProductID,
		})
		created++
	}

	var offlineDevices []models.Device
	q := database.DB.Where("status <> 'offline' AND last_heartbeat_at < ?", time.Now().Add(-5*time.Minute))
	if companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	q.Find(&offlineDevices)
	for _, d := range offlineDevices {
		d.Status = "offline"
		database.DB.Save(&d)
		database.DB.Create(&models.Alert{
			CompanyID: d.CompanyID, StoreID: &d.StoreID, Kind: "device", Severity: "critical",
			Title: "Device offline", Detail: d.Name + " tidak terhubung > 5 menit", EntityType: "device", EntityID: &d.ID,
		})
		created++
	}

	c.JSON(http.StatusOK, gin.H{"alerts_created": created})
}
