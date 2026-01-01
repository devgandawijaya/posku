package models

import (
	"gorm.io/gorm"
)

type Employee struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID uint           `json:"company_id" gorm:"not null"`
	Company   Company        `json:"company" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	StoreID   *uint          `json:"store_id,omitempty"`
	Store     *Store         `json:"store,omitempty" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:SET NULL"`
	Name      string         `json:"name" gorm:"not null"`
	Username  string         `json:"username" gorm:"unique;not null"`
	Password  string         `json:"password" gorm:"not null"`
	Role      string         `json:"role" gorm:"type:text;not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
