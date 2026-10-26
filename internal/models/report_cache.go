package models

import "time"

// SalesDailyCache implements docs/laporan-penjualan.md v_sales_daily (materialized view substitute,
// refreshed via manual/scheduled trigger since there is no scheduler infrastructure).
type SalesDailyCache struct {
	CompanyID uint      `json:"company_id" gorm:"primaryKey"`
	StoreID   uint      `json:"store_id" gorm:"primaryKey"`
	Date      string    `json:"date" gorm:"primaryKey"` // YYYY-MM-DD
	Orders    int64     `json:"orders"`
	Revenue   float64   `json:"revenue"`
	AvgOrder  float64   `json:"avg_order"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StockSummaryCache implements docs/laporan-stok.md v_stock_summary (materialized view substitute).
type StockSummaryCache struct {
	ProductID  uint      `json:"product_id" gorm:"primaryKey"`
	TotalQty   float64   `json:"total_qty"`
	StockValue float64   `json:"stock_value"`
	Status     string    `json:"status"` // normal|low|out
	UpdatedAt  time.Time `json:"updated_at"`
}
