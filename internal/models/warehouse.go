package models

import (
	"time"

	"gorm.io/gorm"
)

type Warehouse struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID uint           `json:"company_id" gorm:"not null;index"`
	Company   *Company       `json:"company,omitempty" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	StoreID   *uint          `json:"store_id,omitempty" gorm:"index"`
	Store     *Store         `json:"store,omitempty" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:SET NULL"`
	Code      string         `json:"code" gorm:"size:50;index"`
	Name      string         `json:"name" gorm:"not null"`
	Location  string         `json:"location" gorm:"type:text"`
	IsCentral bool           `json:"is_central" gorm:"not null;default:false"`
	Status    string         `json:"status" gorm:"type:varchar(20);not null;default:'aktif'"`
	Stocks    []Stock        `json:"stocks" gorm:"foreignKey:WarehouseID"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
