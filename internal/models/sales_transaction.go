package models

import (
	"time"

	"gorm.io/gorm"
)

type SalesTransaction struct {
	ID              uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID       uint           `json:"company_id" gorm:"not null;index"`
	StoreID         uint           `json:"store_id" gorm:"not null"`
	Store           Store          `json:"store" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:CASCADE"`
	ShiftID         *uint          `json:"shift_id,omitempty"`
	EmployeeID      uint           `json:"employee_id" gorm:"not null"`
	Employee        Employee       `json:"employee" gorm:"foreignKey:EmployeeID;references:ID;constraint:OnDelete:RESTRICT"`
	CustomerID      *uint          `json:"customer_id,omitempty"`
	Customer        *Customer      `json:"customer,omitempty" gorm:"foreignKey:CustomerID;references:ID;constraint:OnDelete:SET NULL"`
	InvoiceNo       string         `json:"invoice_no" gorm:"index"`
	TransactionDate time.Time      `json:"transaction_date" gorm:"autoCreateTime"`
	Subtotal        float64        `json:"subtotal" gorm:"type:decimal(10,2);not null;default:0"`
	Discount        float64        `json:"discount" gorm:"type:decimal(10,2);not null;default:0"`
	VoucherAmount   float64        `json:"voucher_amount" gorm:"type:decimal(10,2);not null;default:0"`
	Tax             float64        `json:"tax" gorm:"type:decimal(10,2);not null;default:0"`
	TotalAmount     float64        `json:"total_amount" gorm:"type:decimal(10,2);not null"`
	PaymentMethod   string         `json:"payment_method" gorm:"type:varchar(20)"`
	Status          string         `json:"status" gorm:"type:varchar(20);not null;default:'lunas'"` // lunas|pending|refund|void
	SyncStatus      string         `json:"sync_status" gorm:"type:varchar(20);not null;default:'synced'"`
	ManualDiscount  bool           `json:"manual_discount" gorm:"not null;default:false"`
	Notes           string         `json:"notes" gorm:"type:text"`
	SalesItems      []SalesItem    `json:"sales_items" gorm:"foreignKey:SalesTransactionID"`
	Payment         *Payment       `json:"payment,omitempty" gorm:"foreignKey:SalesTransactionID"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}
