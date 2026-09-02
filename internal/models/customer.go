package models

import (
	"time"

	"gorm.io/gorm"
)

type Customer struct {
	ID              uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID       uint           `json:"company_id" gorm:"not null"`
	Company         Company        `json:"company" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	MemberCode      string         `json:"member_code"`
	Name            string         `json:"name" gorm:"not null"`
	Email           string         `json:"email"`
	Phone           string         `json:"phone" gorm:"size:50"`
	Tier            string         `json:"tier" gorm:"type:varchar(20);not null;default:'bronze'"`
	PointsBalance   int            `json:"points_balance" gorm:"not null;default:0"`
	StoreIDs        string         `json:"store_ids" gorm:"type:jsonb;not null;default:'[]'"` // registered_store_ids
	FavoriteStoreID *uint          `json:"favorite_store_id,omitempty"`
	Status          string         `json:"status" gorm:"type:varchar(20);not null;default:'aktif'"`
	JoinedAt        time.Time      `json:"joined_at" gorm:"autoCreateTime"`
	LastVisitAt     *time.Time     `json:"last_visit_at,omitempty"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}
