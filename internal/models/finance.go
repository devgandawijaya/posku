package models

import "time"

// Expense implements docs/laporan-keuangan.md operational expenses.
type Expense struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID uint      `json:"company_id" gorm:"not null;index"`
	StoreID   *uint     `json:"store_id,omitempty"`
	Category  string    `json:"category" gorm:"not null"`
	Amount    float64   `json:"amount" gorm:"type:decimal(15,2);not null"`
	Date      time.Time `json:"date" gorm:"not null"`
	Note      string    `json:"note" gorm:"type:text"`
	CreatedBy uint      `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Payroll implements docs/laporan-keuangan.md payroll records.
type Payroll struct {
	ID         uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID  uint       `json:"company_id" gorm:"not null;index"`
	StoreID    uint       `json:"store_id" gorm:"not null"`
	EmployeeID uint       `json:"employee_id" gorm:"not null"`
	Period     string     `json:"period" gorm:"not null"` // e.g. 2026-08
	BaseSalary float64    `json:"base_salary" gorm:"type:decimal(15,2);not null"`
	Allowance  float64    `json:"allowance" gorm:"type:decimal(15,2);not null;default:0"`
	Deduction  float64    `json:"deduction" gorm:"type:decimal(15,2);not null;default:0"`
	Net        float64    `json:"net" gorm:"type:decimal(15,2);not null"`
	Status     string     `json:"status" gorm:"type:varchar(20);not null;default:'draft'"`
	PaidAt     *time.Time `json:"paid_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
