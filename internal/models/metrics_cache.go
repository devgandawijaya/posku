package models

import "time"

// CategoryKPICache implements docs/kategori.md category_kpi_cache (refreshed via manual/scheduled trigger).
type CategoryKPICache struct {
	CategoryID   uint      `json:"category_id" gorm:"primaryKey"`
	ProductCount int64     `json:"product_count"`
	TotalStock   float64   `json:"total_stock"`
	TotalOmzet   float64   `json:"total_omzet"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}

// OutletMetricsCache implements docs/outlet.md outlet_metrics_cache.
type OutletMetricsCache struct {
	StoreID           uint       `json:"store_id" gorm:"primaryKey"`
	TotalSales        float64    `json:"total_sales"`
	TotalTransactions int64      `json:"total_transactions"`
	LastTransactionAt *time.Time `json:"last_transaction_at,omitempty"`
	EmployeeCount     int64      `json:"employee_count"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// CustomerMetricsCache implements docs/pelanggan.md customer_metrics_cache.
type CustomerMetricsCache struct {
	CustomerID        uint       `json:"customer_id" gorm:"primaryKey"`
	TotalTransactions int64      `json:"total_transactions"`
	TotalSpent        float64    `json:"total_spent"`
	LastVisitAt       *time.Time `json:"last_visit_at,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// SupplierMetricsCache implements docs/supplier.md supplier_metrics_cache.
type SupplierMetricsCache struct {
	SupplierID     uint       `json:"supplier_id" gorm:"primaryKey"`
	TotalProducts  int64      `json:"total_products"`
	TotalPurchases float64    `json:"total_purchases"`
	LastOrderAt    *time.Time `json:"last_order_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// SupplierPurchase is a minimal purchase record used to drive supplier metrics (no full PO module yet).
type SupplierPurchase struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	SupplierID uint      `json:"supplier_id" gorm:"not null;index"`
	Amount     float64   `json:"amount" gorm:"type:decimal(15,2);not null"`
	Note       string    `json:"note"`
	At         time.Time `json:"at" gorm:"autoCreateTime"`
}
