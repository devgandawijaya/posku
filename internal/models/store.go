package models

import (
	"gorm.io/gorm"
)

type Store struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID uint           `json:"company_id" gorm:"not null"`
	Company   Company        `json:"company" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	Name      string         `json:"name" gorm:"not null"`
	Address   string         `json:"address" gorm:"type:text"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
