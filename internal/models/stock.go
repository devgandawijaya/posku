package models

import (
	"gorm.io/gorm"
)

type Stock struct {
	ID          uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	WarehouseID uint           `json:"warehouse_id" gorm:"not null;uniqueIndex:idx_warehouse_product"`
	Warehouse   Warehouse      `json:"warehouse" gorm:"foreignKey:WarehouseID;references:ID;constraint:OnDelete:CASCADE"`
	ProductID   uint           `json:"product_id" gorm:"not null;uniqueIndex:idx_warehouse_product"`
	Product     Product        `json:"product" gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:RESTRICT"`
	Quantity    int            `json:"quantity" gorm:"not null;default:0"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
