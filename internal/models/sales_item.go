package models

import (
	"gorm.io/gorm"
)

type SalesItem struct {
	ID                 uint             `json:"id" gorm:"primaryKey;autoIncrement"`
	SalesTransactionID uint             `json:"sales_transaction_id" gorm:"not null"`
	SalesTransaction   SalesTransaction `json:"sales_transaction" gorm:"foreignKey:SalesTransactionID;references:ID;constraint:OnDelete:CASCADE"`
	ProductID          uint             `json:"product_id" gorm:"not null"`
	Product            Product          `json:"product" gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:RESTRICT"`
	Quantity           int              `json:"quantity" gorm:"not null"`
	Price              float64          `json:"price" gorm:"type:decimal(10,2);not null"`
	Subtotal           float64          `json:"subtotal" gorm:"type:decimal(10,2);not null"`
	DeletedAt          gorm.DeletedAt   `json:"-" gorm:"index"`
}
