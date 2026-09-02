package models

import "time"

// RefundPayment implements docs/retur.md refund money tracking per return.
type RefundPayment struct {
	ID       uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ReturnID uint      `json:"return_id" gorm:"not null;index"`
	Amount   float64   `json:"amount" gorm:"type:decimal(15,2);not null"`
	Method   string    `json:"method" gorm:"not null"`
	RefNo    string    `json:"ref_no"`
	ByUserID uint      `json:"by_user_id"`
	PaidAt   time.Time `json:"paid_at" gorm:"autoCreateTime"`
}
