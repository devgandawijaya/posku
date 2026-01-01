package models

import (
	"gorm.io/gorm"
)

type TransferItem struct {
	ID              uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	StockTransferID uint           `json:"stock_transfer_id" gorm:"not null"`
	StockTransfer   StockTransfer  `json:"stock_transfer" gorm:"foreignKey:StockTransferID;references:ID;constraint:OnDelete:CASCADE"`
	ProductID       uint           `json:"product_id" gorm:"not null"`
	Product         Product        `json:"product" gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:RESTRICT"`
	Quantity        int            `json:"quantity" gorm:"not null"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}
