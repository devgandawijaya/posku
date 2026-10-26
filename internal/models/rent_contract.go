package models

import "time"

// RentContract implements docs/laporan-keuangan.md rent expense contracts per outlet.
type RentContract struct {
	ID          uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID   uint       `json:"company_id" gorm:"not null;index"`
	StoreID     uint       `json:"store_id" gorm:"not null"`
	MonthlyRent float64    `json:"monthly_rent" gorm:"type:decimal(15,2);not null"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Status      string     `json:"status" gorm:"type:varchar(20);not null;default:'aktif'"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
