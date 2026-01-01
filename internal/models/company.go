package models

import (
	"time"

	"gorm.io/gorm"
)

type Company struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string         `json:"name" gorm:"not null"`
	Address   string         `json:"address" gorm:"type:text"`
	Stores    []Store        `json:"stores" gorm:"foreignKey:CompanyID"`
	Employees []Employee     `json:"employees" gorm:"foreignKey:CompanyID"`
	Customers []Customer     `json:"customers" gorm:"foreignKey:CompanyID"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
