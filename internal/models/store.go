package models

import (
	"time"

	"gorm.io/gorm"
)

type Store struct {
	ID                uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID         uint           `json:"company_id" gorm:"not null"`
	Company           Company        `json:"company" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	Code              string         `json:"code"`
	Name              string         `json:"name" gorm:"not null"`
	Address           string         `json:"address" gorm:"type:text"`
	Phone             string         `json:"phone"`
	ManagerEmployeeID *uint          `json:"manager_employee_id,omitempty"`
	ManagerName       string         `json:"manager_name"`
	Status            string         `json:"status" gorm:"type:varchar(20);not null;default:'aktif'"`
	OpenedAt          *time.Time     `json:"opened_at,omitempty"`
	Notes             string         `json:"notes" gorm:"type:text"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}
