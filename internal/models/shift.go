package models

import "time"

// Shift implements docs/kasir.md cashier shift tracking.
type Shift struct {
	ID           uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID    uint       `json:"company_id" gorm:"not null;index"`
	StoreID      uint       `json:"store_id" gorm:"not null"`
	Store        Store      `json:"store" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:CASCADE"`
	EmployeeID   uint       `json:"employee_id" gorm:"not null"`
	Employee     Employee   `json:"employee" gorm:"foreignKey:EmployeeID;references:ID;constraint:OnDelete:RESTRICT"`
	Code         string     `json:"code"`
	OpenedAt     time.Time  `json:"opened_at" gorm:"autoCreateTime"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`
	OpeningCash  float64    `json:"opening_cash" gorm:"type:decimal(15,2);not null;default:0"`
	CashIn       float64    `json:"cash_in" gorm:"type:decimal(15,2);not null;default:0"`
	CashOut      float64    `json:"cash_out" gorm:"type:decimal(15,2);not null;default:0"`
	ExpectedCash float64    `json:"expected_cash" gorm:"type:decimal(15,2);not null;default:0"`
	ActualCash   *float64   `json:"actual_cash,omitempty" gorm:"type:decimal(15,2)"`
	Status       string     `json:"status" gorm:"type:varchar(20);not null;default:'open'"` // open|closed
	Notes        string     `json:"notes" gorm:"type:text"`
}

// ShiftCashMovement records manual cash in/out during an open shift.
type ShiftCashMovement struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ShiftID   uint      `json:"shift_id" gorm:"not null;index"`
	CompanyID uint      `json:"company_id" gorm:"not null"`
	Direction string    `json:"direction" gorm:"type:varchar(10);not null"` // in|out
	Amount    float64   `json:"amount" gorm:"type:decimal(15,2);not null"`
	Reason    string    `json:"reason" gorm:"type:varchar(30);not null"`
	Note      string    `json:"note" gorm:"type:text"`
	CreatedBy uint      `json:"created_by"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}
