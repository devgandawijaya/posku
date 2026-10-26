package models

import "time"

// Plan is the global subscription plan catalog (docs/subscription-billing.md).
type Plan struct {
	ID                      uint    `json:"id" gorm:"primaryKey;autoIncrement"`
	Code                    string  `json:"code" gorm:"unique;not null"`
	Name                    string  `json:"name" gorm:"not null"`
	Description             string  `json:"description" gorm:"type:text"`
	PriceMonthly            float64 `json:"price_monthly" gorm:"type:decimal(15,2);not null"`
	PriceYearly             float64 `json:"price_yearly" gorm:"type:decimal(15,2);not null"`
	MaxOutlets              *int    `json:"max_outlets,omitempty"`
	MaxUsers                *int    `json:"max_users,omitempty"`
	MaxProducts             *int    `json:"max_products,omitempty"`
	MaxTransactionsPerMonth *int    `json:"max_transactions_per_month,omitempty"`
	Features                string  `json:"features" gorm:"type:jsonb;not null;default:'{}'"`
	IsActive                bool    `json:"is_active" gorm:"not null;default:true"`
	SortOrder               int     `json:"sort_order" gorm:"not null;default:0"`
}

// Subscription binds a company to a plan (1 active subscription per company).
type Subscription struct {
	ID                 uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID          uint       `json:"company_id" gorm:"not null;unique"`
	PlanID             uint       `json:"plan_id" gorm:"not null"`
	Plan               Plan       `json:"plan" gorm:"foreignKey:PlanID;references:ID"`
	Status             string     `json:"status" gorm:"type:varchar(20);not null;default:'trial'"`
	BillingCycle       string     `json:"billing_cycle" gorm:"type:varchar(20);not null;default:'monthly'"`
	CurrentPeriodStart time.Time  `json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `json:"current_period_end"`
	TrialEndsAt        *time.Time `json:"trial_ends_at,omitempty"`
	CancelledAt        *time.Time `json:"cancelled_at,omitempty"`
	CancelAtPeriodEnd  bool       `json:"cancel_at_period_end" gorm:"not null;default:false"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Invoice is a billing invoice for a subscription period.
type Invoice struct {
	ID             uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID      uint       `json:"company_id" gorm:"not null;index"`
	SubscriptionID uint       `json:"subscription_id" gorm:"not null"`
	InvoiceNo      string     `json:"invoice_no" gorm:"unique"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	Subtotal       float64    `json:"subtotal" gorm:"type:decimal(15,2);not null"`
	Tax            float64    `json:"tax" gorm:"type:decimal(15,2);not null;default:0"`
	Discount       float64    `json:"discount" gorm:"type:decimal(15,2);not null;default:0"`
	Total          float64    `json:"total" gorm:"type:decimal(15,2);not null"`
	Status         string     `json:"status" gorm:"type:varchar(20);not null;default:'open'"`
	DueDate        time.Time  `json:"due_date"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// BillingPayment records payment against an invoice.
type BillingPayment struct {
	ID          uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID   uint       `json:"company_id" gorm:"not null;index"`
	InvoiceID   uint       `json:"invoice_id" gorm:"not null"`
	Method      string     `json:"method" gorm:"not null"`
	Amount      float64    `json:"amount" gorm:"type:decimal(15,2);not null"`
	ExternalRef string     `json:"external_ref"`
	Status      string     `json:"status" gorm:"type:varchar(20);not null;default:'success'"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
