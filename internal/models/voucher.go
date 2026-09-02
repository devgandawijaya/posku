package models

import "time"

// Voucher implements docs/kasir.md voucher catalog.
type Voucher struct {
	ID          uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID   uint       `json:"company_id" gorm:"not null;index"`
	Code        string     `json:"code" gorm:"not null"`
	Label       string     `json:"label"`
	Type        string     `json:"type" gorm:"type:varchar(20);not null"` // percent|amount
	Value       float64    `json:"value" gorm:"type:decimal(15,2);not null"`
	MinSpend    float64    `json:"min_spend" gorm:"type:decimal(15,2);not null;default:0"`
	MaxDiscount *float64   `json:"max_discount,omitempty" gorm:"type:decimal(15,2)"`
	UsageLimit  *int       `json:"usage_limit,omitempty"`
	UsedCount   int        `json:"used_count" gorm:"not null;default:0"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	IsActive    bool       `json:"is_active" gorm:"not null;default:true"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Promotion implements docs/kasir.md per-product/category promotions.
type Promotion struct {
	ID         uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID  uint       `json:"company_id" gorm:"not null;index"`
	Name       string     `json:"name" gorm:"not null"`
	Type       string     `json:"type" gorm:"type:varchar(20);not null"` // bogo|percent|amount
	Value      float64    `json:"value" gorm:"type:decimal(15,2);not null;default:0"`
	ProductIDs string     `json:"product_ids" gorm:"type:jsonb;not null;default:'[]'"`
	CategoryID *uint      `json:"category_id,omitempty"`
	MinQty     int        `json:"min_qty" gorm:"not null;default:1"`
	ValidFrom  *time.Time `json:"valid_from,omitempty"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
	IsActive   bool       `json:"is_active" gorm:"not null;default:true"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
