package models

import (
	"gorm.io/gorm"
)

type Customer struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID uint           `json:"company_id" gorm:"not null"`
	Company   Company        `json:"company" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	Name      string         `json:"name" gorm:"not null"`
	Email     string         `json:"email"`
	Phone     string         `json:"phone" gorm:"size:50"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
