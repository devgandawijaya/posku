package models

import "time"

// UserMetricsCache implements docs/karyawan.md user_metrics_cache (refreshed via manual/scheduled trigger).
type UserMetricsCache struct {
	EmployeeID          uint       `json:"employee_id" gorm:"primaryKey"`
	OutletCount         int        `json:"outlet_count"`
	TransactionCount30d int64      `json:"transaction_count_30d"`
	LastActiveAt        *time.Time `json:"last_active_at,omitempty"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
