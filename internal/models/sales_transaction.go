package models

import (
	"time"

	"gorm.io/gorm"
)

type SalesTransaction struct {
	ID              uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	StoreID         uint           `json:"store_id" gorm:"not null"`
	Store           Store          `json:"store" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:CASCADE"`
	EmployeeID      uint           `json:"employee_id" gorm:"not null"`
	Employee        Employee       `json:"employee" gorm:"foreignKey:EmployeeID;references:ID;constraint:OnDelete:RESTRICT"`
	CustomerID      *uint          `json:"customer_id,omitempty"`
	Customer        *Customer      `json:"customer,omitempty" gorm:"foreignKey:CustomerID;references:ID;constraint:OnDelete:SET NULL"`
	TransactionDate time.Time      `json:"transaction_date" gorm:"autoCreateTime"`
	TotalAmount     float64        `json:"total_amount" gorm:"type:decimal(10,2);not null"`
	SalesItems      []SalesItem    `json:"sales_items" gorm:"foreignKey:SalesTransactionID"`
	Payment         *Payment       `json:"payment,omitempty" gorm:"foreignKey:SalesTransactionID"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}
