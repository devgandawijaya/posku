package models

import (
	"time"

	"gorm.io/gorm"
)

type Payment struct {
	ID                 uint              `json:"id" gorm:"primaryKey;autoIncrement"`
	SalesTransactionID uint              `json:"sales_transaction_id" gorm:"not null;index"`
	SalesTransaction   *SalesTransaction `json:"sales_transaction,omitempty" gorm:"foreignKey:SalesTransactionID;references:ID;constraint:OnDelete:CASCADE"`
	Method             string            `json:"method" gorm:"type:varchar(50);not null"`
	Amount             float64           `json:"amount" gorm:"type:decimal(10,2);not null"`
	CashReceived       float64           `json:"cash_received" gorm:"type:decimal(10,2);default:0"`
	Change             float64           `json:"change" gorm:"type:decimal(10,2);default:0"`
	ReferenceNumber    string            `json:"reference_number"`
	PaymentDate        time.Time         `json:"payment_date" gorm:"autoCreateTime"`
	DeletedAt          gorm.DeletedAt    `json:"-" gorm:"index"`
}
