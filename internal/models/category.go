package models

import (
	"time"

	"gorm.io/gorm"
)

// Category implements docs/kategori.md (product categories, self-referencing subcategory).
type Category struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID uint           `json:"company_id" gorm:"not null;index"`
	Company   Company        `json:"-" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	ParentID  *uint          `json:"parent_id,omitempty"`
	Parent    *Category      `json:"parent,omitempty" gorm:"foreignKey:ParentID;references:ID;constraint:OnDelete:SET NULL"`
	Code      string         `json:"code"`
	Name      string         `json:"name" gorm:"not null"`
	Slug      string         `json:"slug"`
	Icon      string         `json:"icon"`
	Scope     string         `json:"scope" gorm:"type:varchar(20);not null;default:'all'"` // all|specific
	StoreIDs  string         `json:"store_ids" gorm:"type:jsonb;not null;default:'[]'"`
	IsActive  bool           `json:"is_active" gorm:"not null;default:true"`
	SortOrder int            `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
