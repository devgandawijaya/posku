package models

import "time"

// TenantPaymentMethod implements docs/subscription-billing.md stored payment methods.
type TenantPaymentMethod struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID  uint      `json:"company_id" gorm:"not null;index"`
	Type       string    `json:"type" gorm:"not null"`
	Provider   string    `json:"provider"`
	ExternalID string    `json:"external_id"`
	MaskedInfo string    `json:"masked_info"`
	IsDefault  bool      `json:"is_default" gorm:"not null;default:false"`
	CreatedAt  time.Time `json:"created_at"`
}

// Coupon implements docs/subscription-billing.md discount coupons for subscriptions.
type Coupon struct {
	ID             uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Code           string     `json:"code" gorm:"unique;not null"`
	Type           string     `json:"type" gorm:"not null"` // percent|amount
	Value          float64    `json:"value" gorm:"type:decimal(15,2);not null"`
	MaxRedemptions *int       `json:"max_redemptions,omitempty"`
	RedeemedCount  int        `json:"redeemed_count" gorm:"not null;default:0"`
	ValidFrom      *time.Time `json:"valid_from,omitempty"`
	ValidUntil     *time.Time `json:"valid_until,omitempty"`
	IsActive       bool       `json:"is_active" gorm:"not null;default:true"`
	CreatedAt      time.Time  `json:"created_at"`
}
