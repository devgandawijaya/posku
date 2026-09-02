package controllers

import (
	"fmt"
	"net/http"
	"time"

	"posku/internal/database"
	"posku/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// applyRange applies optional date_from/date_to/store_id/company_id query filters to a sales-scoped query.
func applySalesFilters(c *gin.Context, q *gorm.DB) *gorm.DB {
	q = q.Where("status <> 'void'")
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if payment := c.Query("payment"); payment != "" {
		q = q.Where("payment_method = ?", payment)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		q = q.Where("transaction_date >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		q = q.Where("transaction_date <= ?", dateTo)
	}
	return q
}

// computeRealCOGS implements docs/laporan-keuangan.md "HPP riil per sale_items" using
// products.cost x sale_items.quantity, instead of the flat 60% estimate used previously.
func computeRealCOGS(c *gin.Context) float64 {
	q := database.DB.Table("sale_items").
		Joins("JOIN sales_transactions ON sales_transactions.id = sale_items.sales_transaction_id").
		Joins("JOIN products ON products.id = sale_items.product_id").
		Where("sales_transactions.status <> 'void'")
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("sales_transactions.company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("sales_transactions.store_id = ?", storeID)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		q = q.Where("sales_transactions.transaction_date >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		q = q.Where("sales_transactions.transaction_date <= ?", dateTo)
	}
	var cogs float64
	q.Select("COALESCE(SUM(sale_items.quantity * products.cost), 0)").Scan(&cogs)
	return cogs
}

// GetSalesSummaryReport implements docs/laporan-penjualan.md GET /reports/sales/summary
func GetSalesSummaryReport(c *gin.Context) {
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	var total float64
	var orders int64
	q.Count(&orders)
	q.Select("COALESCE(SUM(total_amount), 0)").Scan(&total)
	aov := 0.0
	if orders > 0 {
		aov = total / float64(orders)
	}
	cogs := computeRealCOGS(c)
	if cogs == 0 {
		cogs = total * 0.6 // fallback estimate when sale_items/cost data is unavailable
	}
	net := total - cogs
	c.JSON(http.StatusOK, gin.H{
		"total":  total,
		"orders": orders,
		"aov":    aov,
		"cogs":   cogs,
		"net":    net,
	})
}

// GetSalesTimeseriesReport implements GET /reports/sales/timeseries
func GetSalesTimeseriesReport(c *gin.Context) {
	type row struct {
		Date  string  `json:"date"`
		Omzet float64 `json:"omzet"`
	}
	var rows []row
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	q.Select("DATE(transaction_date) as date, COALESCE(SUM(total_amount),0) as omzet").
		Group("DATE(transaction_date)").Order("date").Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// GetSalesByPaymentReport implements GET /reports/sales/by-payment
func GetSalesByPaymentReport(c *gin.Context) {
	type row struct {
		PaymentMethod string  `json:"payment_method"`
		Amount        float64 `json:"amount"`
		Count         int64   `json:"count"`
	}
	var rows []row
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	q.Select("payment_method, COALESCE(SUM(total_amount),0) as amount, COUNT(*) as count").
		Group("payment_method").Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// GetSalesByStoreReport implements GET /reports/sales/by-store
func GetSalesByStoreReport(c *gin.Context) {
	type row struct {
		StoreID uint    `json:"store_id"`
		Name    string  `json:"name"`
		Revenue float64 `json:"revenue"`
		Orders  int64   `json:"orders"`
	}
	var rows []row
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	q.Select("sales_transactions.store_id as store_id, stores.name as name, COALESCE(SUM(total_amount),0) as revenue, COUNT(*) as orders").
		Joins("JOIN stores ON stores.id = sales_transactions.store_id").
		Group("sales_transactions.store_id, stores.name").Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// GetTopProductsReport implements GET /reports/sales/top-products
func GetTopProductsReport(c *gin.Context) {
	type row struct {
		ProductID uint    `json:"product_id"`
		Name      string  `json:"name"`
		Qty       float64 `json:"qty"`
		Revenue   float64 `json:"revenue"`
	}
	var rows []row
	database.DB.Table("sales_items").
		Select("sales_items.product_id as product_id, products.name as name, SUM(sales_items.quantity) as qty, SUM(sales_items.subtotal) as revenue").
		Joins("JOIN products ON products.id = sales_items.product_id").
		Group("sales_items.product_id, products.name").
		Order("revenue desc").
		Limit(10).Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// ---- Stock reports (docs/laporan-stok.md) ----

func GetStockSummaryReport(c *gin.Context) {
	type row struct {
		ProductID  uint    `json:"product_id"`
		SKU        string  `json:"sku"`
		Name       string  `json:"name"`
		TotalQty   float64 `json:"total_qty"`
		StockValue float64 `json:"stock_value"`
	}
	var rows []row
	database.DB.Table("stocks").
		Select("products.id as product_id, products.sku as sku, products.name as name, SUM(stocks.quantity) as total_qty, SUM(stocks.quantity * products.cost) as stock_value").
		Joins("JOIN products ON products.id = stocks.product_id").
		Group("products.id, products.sku, products.name").Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

func GetStockByCategoryReport(c *gin.Context) {
	type row struct {
		CategoryID *uint   `json:"category_id"`
		TotalQty   float64 `json:"total_qty"`
	}
	var rows []row
	database.DB.Table("stocks").
		Select("products.category_id as category_id, SUM(stocks.quantity) as total_qty").
		Joins("JOIN products ON products.id = stocks.product_id").
		Group("products.category_id").Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

func GetStockMovementReport(c *gin.Context) {
	type row struct {
		Date   string  `json:"date"`
		Masuk  float64 `json:"masuk"`
		Keluar float64 `json:"keluar"`
	}
	var rows []row
	q := database.DB.Table("stock_movements")
	if productID := c.Query("product_id"); productID != "" {
		q = q.Where("product_id = ?", productID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		q = q.Where("created_at >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		q = q.Where("created_at <= ?", dateTo)
	}
	q.Select("DATE(created_at) as date, COALESCE(SUM(CASE WHEN delta > 0 THEN delta ELSE 0 END),0) as masuk, COALESCE(SUM(CASE WHEN delta < 0 THEN -delta ELSE 0 END),0) as keluar").
		Group("DATE(created_at)").Order("date").Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

func GetTopLowStockReport(c *gin.Context) {
	type row struct {
		ProductID uint    `json:"product_id"`
		Name      string  `json:"name"`
		TotalQty  float64 `json:"total_qty"`
	}
	var rows []row
	database.DB.Table("stocks").
		Select("products.id as product_id, products.name as name, SUM(stocks.quantity) as total_qty").
		Joins("JOIN products ON products.id = stocks.product_id").
		Group("products.id, products.name").
		Order("total_qty asc").
		Limit(10).Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// ---- Finance reports (docs/laporan-keuangan.md) ----

func GetFinancePLReport(c *gin.Context) {
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	var pendapatan float64
	q.Select("COALESCE(SUM(total_amount),0)").Scan(&pendapatan)
	cogs := computeRealCOGS(c)
	if cogs == 0 {
		cogs = pendapatan * 0.6 // fallback estimate when sale_items/cost data is unavailable
	}

	expQ := database.DB.Model(&models.Expense{})
	if companyID := c.Query("company_id"); companyID != "" {
		expQ = expQ.Where("company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		expQ = expQ.Where("store_id = ?", storeID)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		expQ = expQ.Where("date >= ?", dateFrom)
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		expQ = expQ.Where("date <= ?", dateTo)
	}
	var opex float64
	expQ.Select("COALESCE(SUM(amount),0)").Scan(&opex)

	rentQ := database.DB.Model(&models.RentContract{}).Where("status = 'aktif'")
	if companyID := c.Query("company_id"); companyID != "" {
		rentQ = rentQ.Where("company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		rentQ = rentQ.Where("store_id = ?", storeID)
	}
	var monthlyRent float64
	rentQ.Select("COALESCE(SUM(monthly_rent),0)").Scan(&monthlyRent)
	opex += monthlyRent

	labaKotor := pendapatan - cogs
	labaBersih := labaKotor - opex
	netMargin := 0.0
	if pendapatan > 0 {
		netMargin = labaBersih / pendapatan * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"pendapatan":  pendapatan,
		"cogs":        cogs,
		"opex":        opex,
		"laba_kotor":  labaKotor,
		"laba_bersih": labaBersih,
		"net_margin":  netMargin,
	})
}

func GetFinanceExpenseBreakdownReport(c *gin.Context) {
	type row struct {
		Category string  `json:"category"`
		Amount   float64 `json:"amount"`
	}
	var rows []row
	q := database.DB.Model(&models.Expense{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	q.Select("category, COALESCE(SUM(amount),0) as amount").Group("category").Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

func GetFinanceCashflowReport(c *gin.Context) {
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}).Where("payment_method = 'cash'"))
	var cashIn float64
	q.Select("COALESCE(SUM(total_amount),0)").Scan(&cashIn)

	var cashOut float64
	expQ := database.DB.Model(&models.Expense{})
	if companyID := c.Query("company_id"); companyID != "" {
		expQ = expQ.Where("company_id = ?", companyID)
	}
	expQ.Select("COALESCE(SUM(amount),0)").Scan(&cashOut)

	c.JSON(http.StatusOK, gin.H{"cash_in": cashIn, "cash_out": cashOut})
}

// ExportSalesReport implements docs/laporan-penjualan.md GET /reports/sales/export
func ExportSalesReport(c *gin.Context) {
	type row struct {
		StoreID uint    `json:"store_id"`
		Name    string  `json:"name"`
		Revenue float64 `json:"revenue"`
		Orders  int64   `json:"orders"`
	}
	var rows []row
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	q.Select("sales_transactions.store_id as store_id, stores.name as name, COALESCE(SUM(total_amount),0) as revenue, COUNT(*) as orders").
		Joins("JOIN stores ON stores.id = sales_transactions.store_id").
		Group("sales_transactions.store_id, stores.name").Scan(&rows)

	csvRows := make([][]string, 0, len(rows))
	for _, r := range rows {
		csvRows = append(csvRows, []string{fmt.Sprint(r.StoreID), r.Name, fmt.Sprintf("%.2f", r.Revenue), fmt.Sprint(r.Orders)})
	}
	writeCSV(c, "sales_report.csv", []string{"store_id", "store_name", "revenue", "orders"}, csvRows)
}

// ExportStockReport implements docs/laporan-stok.md GET /reports/stock/export
func ExportStockReport(c *gin.Context) {
	type row struct {
		ProductID  uint    `json:"product_id"`
		SKU        string  `json:"sku"`
		Name       string  `json:"name"`
		TotalQty   float64 `json:"total_qty"`
		StockValue float64 `json:"stock_value"`
	}
	var rows []row
	database.DB.Table("stocks").
		Select("products.id as product_id, products.sku as sku, products.name as name, SUM(stocks.quantity) as total_qty, SUM(stocks.quantity * products.cost) as stock_value").
		Joins("JOIN products ON products.id = stocks.product_id").
		Group("products.id, products.sku, products.name").Scan(&rows)

	csvRows := make([][]string, 0, len(rows))
	for _, r := range rows {
		csvRows = append(csvRows, []string{fmt.Sprint(r.ProductID), r.SKU, r.Name, fmt.Sprintf("%.3f", r.TotalQty), fmt.Sprintf("%.2f", r.StockValue)})
	}
	writeCSV(c, "stock_report.csv", []string{"product_id", "sku", "name", "total_qty", "stock_value"}, csvRows)
}

// ExportFinanceReport implements docs/laporan-keuangan.md GET /reports/finance/export
func ExportFinanceReport(c *gin.Context) {
	type row struct {
		Category string  `json:"category"`
		Amount   float64 `json:"amount"`
	}
	var rows []row
	q := database.DB.Model(&models.Expense{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	q.Select("category, COALESCE(SUM(amount),0) as amount").Group("category").Scan(&rows)

	csvRows := make([][]string, 0, len(rows))
	for _, r := range rows {
		csvRows = append(csvRows, []string{r.Category, fmt.Sprintf("%.2f", r.Amount)})
	}
	writeCSV(c, "finance_report.csv", []string{"category", "amount"}, csvRows)
}

// RefreshSalesDailyCache implements docs/laporan-penjualan.md v_sales_daily as a materialized-view
// substitute (manual/scheduled trigger, since there is no scheduler infrastructure).
func RefreshSalesDailyCache(c *gin.Context) {
	type row struct {
		CompanyID uint    `json:"company_id"`
		StoreID   uint    `json:"store_id"`
		Date      string  `json:"date"`
		Orders    int64   `json:"orders"`
		Revenue   float64 `json:"revenue"`
	}
	var rows []row
	q := applySalesFilters(c, database.DB.Model(&models.SalesTransaction{}))
	q.Select("company_id, store_id, TO_CHAR(transaction_date, 'YYYY-MM-DD') as date, COUNT(*) as orders, COALESCE(SUM(total_amount),0) as revenue").
		Group("company_id, store_id, TO_CHAR(transaction_date, 'YYYY-MM-DD')").Scan(&rows)

	for _, r := range rows {
		avgOrder := 0.0
		if r.Orders > 0 {
			avgOrder = r.Revenue / float64(r.Orders)
		}
		database.DB.Save(&models.SalesDailyCache{
			CompanyID: r.CompanyID, StoreID: r.StoreID, Date: r.Date,
			Orders: r.Orders, Revenue: r.Revenue, AvgOrder: avgOrder, UpdatedAt: time.Now(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"refreshed": len(rows)})
}

func GetSalesDailyCache(c *gin.Context) {
	var list []models.SalesDailyCache
	q := database.DB.Model(&models.SalesDailyCache{})
	if companyID := c.Query("company_id"); companyID != "" {
		q = q.Where("company_id = ?", companyID)
	}
	if storeID := c.Query("store_id"); storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	q.Order("date desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// RefreshStockSummaryCache implements docs/laporan-stok.md v_stock_summary materialized-view substitute.
func RefreshStockSummaryCache(c *gin.Context) {
	type row struct {
		ProductID  uint    `json:"product_id"`
		TotalQty   float64 `json:"total_qty"`
		StockValue float64 `json:"stock_value"`
	}
	var rows []row
	database.DB.Table("stocks").
		Select("products.id as product_id, SUM(stocks.quantity) as total_qty, SUM(stocks.quantity * products.cost) as stock_value").
		Joins("JOIN products ON products.id = stocks.product_id").
		Group("products.id").Scan(&rows)

	for _, r := range rows {
		status := "normal"
		if r.TotalQty == 0 {
			status = "out"
		} else if r.TotalQty < 10 {
			status = "low"
		}
		database.DB.Save(&models.StockSummaryCache{
			ProductID: r.ProductID, TotalQty: r.TotalQty, StockValue: r.StockValue,
			Status: status, UpdatedAt: time.Now(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"refreshed": len(rows)})
}

func GetStockSummaryCache(c *gin.Context) {
	var list []models.StockSummaryCache
	database.DB.Order("product_id").Find(&list)
	c.JSON(http.StatusOK, list)
}
