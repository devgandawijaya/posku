package models

import (
	"gorm.io/gorm"
)

type Product struct {
	ID          uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description" gorm:"type:text"`
	Price       float64        `json:"price" gorm:"type:decimal(10,2);not null"`
	Unit        string         `json:"unit" gorm:"size:50"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
