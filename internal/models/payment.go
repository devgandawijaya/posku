package models

import (
	"time"

	"gorm.io/gorm"
)

type Payment struct {
	ID                 uint              `json:"id" gorm:"primaryKey;autoIncrement"`
	SalesTransactionID uint              `json:"sales_transaction_id" gorm:"not null;unique"`
	SalesTransaction   *SalesTransaction `json:"sales_transaction,omitempty" gorm:"foreignKey:SalesTransactionID;references:ID;constraint:OnDelete:CASCADE"`
	Method             string            `json:"method" gorm:"type:payment_method;not null"`
	Amount             float64           `json:"amount" gorm:"type:decimal(10,2);not null"`
	ReferenceNumber    string            `json:"reference_number"`
	PaymentDate        time.Time         `json:"payment_date" gorm:"autoCreateTime"`
	DeletedAt          gorm.DeletedAt    `json:"-" gorm:"index"`
}
