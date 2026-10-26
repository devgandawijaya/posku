package models

import (
	"time"

	"gorm.io/gorm"
)

// Supplier implements docs/supplier.md.
type Supplier struct {
	ID             uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID      uint           `json:"company_id" gorm:"not null;index"`
	Code           string         `json:"code"`
	Name           string         `json:"name" gorm:"not null"`
	ContactPerson  string         `json:"contact_person"`
	Phone          string         `json:"phone"`
	Email          string         `json:"email"`
	Address        string         `json:"address" gorm:"type:text"`
	Category       string         `json:"category"`
	StoreIDs       string         `json:"store_ids" gorm:"type:jsonb;not null;default:'[]'"`
	Status         string         `json:"status" gorm:"type:varchar(20);not null;default:'aktif'"`
	TotalPurchases float64        `json:"total_purchases" gorm:"type:decimal(15,2);not null;default:0"`
	LastOrderAt    *time.Time     `json:"last_order_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}
