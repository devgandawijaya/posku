package models

import "time"

// PointsLedger implements docs/pelanggan.md member points audit trail.
type PointsLedger struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID  uint      `json:"company_id" gorm:"not null;index"`
	CustomerID uint      `json:"customer_id" gorm:"not null;index"`
	Delta      int       `json:"delta" gorm:"not null"`
	Reason     string    `json:"reason" gorm:"not null"` // earn|redeem|adjust|expire
	Note       string    `json:"note" gorm:"type:text"`
	CreatedBy  uint      `json:"created_by"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}
