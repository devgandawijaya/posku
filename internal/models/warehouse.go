package models

import (
	"gorm.io/gorm"
)

type Warehouse struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	StoreID   uint           `json:"store_id" gorm:"not null;unique"`
	Store     *Store         `json:"store" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:CASCADE"`
	Name      string         `json:"name" gorm:"not null"`
	Location  string         `json:"location" gorm:"type:text"`
	Stocks    []Stock        `json:"stocks" gorm:"foreignKey:WarehouseID"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
