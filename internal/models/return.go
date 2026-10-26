package models

import (
	"time"

	"gorm.io/gorm"
)

// Return implements docs/retur.md (retur & refund) in simplified single-tenant-per-company form.
type Return struct {
	ID                 uint             `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID          uint             `json:"company_id" gorm:"not null;index"`
	StoreID            uint             `json:"store_id" gorm:"not null"`
	Store              Store            `json:"store" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:CASCADE"`
	EmployeeID         uint             `json:"employee_id" gorm:"not null"`
	Employee           Employee         `json:"employee" gorm:"foreignKey:EmployeeID;references:ID;constraint:OnDelete:RESTRICT"`
	CustomerID         *uint            `json:"customer_id,omitempty"`
	Customer           *Customer        `json:"customer,omitempty" gorm:"foreignKey:CustomerID;references:ID;constraint:OnDelete:SET NULL"`
	SalesTransactionID uint             `json:"sales_transaction_id" gorm:"not null"`
	SalesTransaction   SalesTransaction `json:"sales_transaction" gorm:"foreignKey:SalesTransactionID;references:ID;constraint:OnDelete:RESTRICT"`
	Date               time.Time        `json:"date" gorm:"autoCreateTime"`
	TotalRefund        float64          `json:"total_refund" gorm:"type:decimal(10,2);not null"`
	Reason             string           `json:"reason" gorm:"type:varchar(30);not null"`
	ReasonNote         string           `json:"reason_note" gorm:"type:text"`
	Status             string           `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	QCNote             string           `json:"qc_note" gorm:"type:text"`
	RestockAt          *time.Time       `json:"restock_at,omitempty"`
	Items              []ReturnItem     `json:"items" gorm:"foreignKey:ReturnID"`
	Approvals          []ReturnApproval `json:"approvals" gorm:"foreignKey:ReturnID"`
	CurrentStep        int              `json:"current_step" gorm:"not null;default:1"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	DeletedAt          gorm.DeletedAt   `json:"-" gorm:"index"`
}

// ReturnApproval implements docs/retur.md berjenjang workflow (Kasir -> Supervisor -> Manager).
type ReturnApproval struct {
	ID           uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	ReturnID     uint       `json:"return_id" gorm:"not null;index"`
	Step         int        `json:"step" gorm:"not null"`
	Role         string     `json:"role" gorm:"not null"` // Kasir|Supervisor|Manager
	ApproverID   *uint      `json:"approver_id,omitempty"`
	ApproverName string     `json:"approver_name"`
	Status       string     `json:"status" gorm:"type:varchar(20);not null;default:'pending'"` // done|current|pending|rejected
	Note         string     `json:"note"`
	At           *time.Time `json:"at,omitempty"`
}

type ReturnItem struct {
	ID           uint    `json:"id" gorm:"primaryKey;autoIncrement"`
	ReturnID     uint    `json:"return_id" gorm:"not null;index"`
	SalesItemID  uint    `json:"sales_item_id" gorm:"not null"`
	ProductID    uint    `json:"product_id" gorm:"not null"`
	Product      Product `json:"product" gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:RESTRICT"`
	SKUSnapshot  string  `json:"sku_snapshot"`
	NameSnapshot string  `json:"name_snapshot"`
	Qty          float64 `json:"qty" gorm:"type:decimal(10,3);not null"`
	Price        float64 `json:"price" gorm:"type:decimal(10,2);not null"`
	Amount       float64 `json:"amount" gorm:"type:decimal(10,2);not null"`
	Fate         string  `json:"fate" gorm:"type:varchar(20);not null;default:'restock'"`
}

// Return status flow: pending -> approved -> processing -> completed (or rejected at any pending step)
const (
	ReturnStatusPending    = "pending"
	ReturnStatusApproved   = "approved"
	ReturnStatusRejected   = "rejected"
	ReturnStatusProcessing = "processing"
	ReturnStatusCompleted  = "completed"
)
