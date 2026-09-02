package models

import (
	"gorm.io/gorm"
)

type Product struct {
	ID          uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID   uint           `json:"company_id" gorm:"not null;index"`
	Company     Company        `json:"-" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	CategoryID  *uint          `json:"category_id,omitempty"`
	Category    *Category      `json:"category,omitempty" gorm:"foreignKey:CategoryID;references:ID;constraint:OnDelete:SET NULL"`
	SKU         string         `json:"sku"`
	Barcode     string         `json:"barcode"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description" gorm:"type:text"`
	Price       float64        `json:"price" gorm:"type:decimal(10,2);not null"`
	Cost        float64        `json:"cost" gorm:"type:decimal(10,2);not null;default:0"`
	Unit        string         `json:"unit" gorm:"size:50"`
	IsActive    bool           `json:"is_active" gorm:"not null;default:true"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
