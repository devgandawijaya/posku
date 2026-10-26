package models

import "time"

// PlanQuota implements docs/subscription-billing.md usage counters per billing period, used for feature gating.
type PlanQuota struct {
	CompanyID         uint      `json:"company_id" gorm:"primaryKey"`
	Period            string    `json:"period" gorm:"primaryKey"` // e.g. 2026-09
	TransactionsCount int       `json:"transactions_count" gorm:"not null;default:0"`
	APICallsCount     int       `json:"api_calls_count" gorm:"not null;default:0"`
	UpdatedAt         time.Time `json:"updated_at"`
}
