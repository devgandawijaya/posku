package models

import (
	"time"

	"gorm.io/gorm"
)

type Employee struct {
	ID               uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID        uint           `json:"company_id" gorm:"not null"`
	Company          Company        `json:"company" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	StoreID          *uint          `json:"store_id,omitempty"`
	Store            *Store         `json:"store,omitempty" gorm:"foreignKey:StoreID;references:ID;constraint:OnDelete:SET NULL"`
	Name             string         `json:"name" gorm:"not null"`
	Username         string         `json:"username" gorm:"unique;not null"`
	Email            string         `json:"email"`
	Password         string         `json:"password" gorm:"not null"`
	RoleID           uint           `json:"role_id" gorm:"not null;default:1"`
	Role             Role           `json:"role" gorm:"foreignKey:RoleID;references:ID;constraint:OnDelete:RESTRICT"`
	StoreIDs         string         `json:"store_ids" gorm:"type:jsonb;not null;default:'[]'"` // multi-outlet placement (karyawan.md)
	Status           string         `json:"status" gorm:"type:varchar(20);not null;default:'aktif'"`
	LastLoginAt      *time.Time     `json:"last_login_at,omitempty"`
	FailedLoginCount int            `json:"-" gorm:"not null;default:0"`
	LockedUntil      *time.Time     `json:"locked_until,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

