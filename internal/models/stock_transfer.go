package models

import (
	"time"

	"gorm.io/gorm"
)

type StockTransfer struct {
	ID            uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	FromStoreID   uint           `json:"from_store_id" gorm:"not null"`
	FromStore     Store          `json:"from_store" gorm:"foreignKey:FromStoreID;references:ID;constraint:OnDelete:RESTRICT"`
	ToStoreID     uint           `json:"to_store_id" gorm:"not null"`
	ToStore       Store          `json:"to_store" gorm:"foreignKey:ToStoreID;references:ID;constraint:OnDelete:RESTRICT"`
	TransferDate  time.Time      `json:"transfer_date" gorm:"autoCreateTime"`
	Status        string         `json:"status" gorm:"type:transfer_status;not null;default:'pending'"`
	TransferItems []TransferItem `json:"transfer_items" gorm:"foreignKey:StockTransferID"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}
