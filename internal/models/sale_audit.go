package models

import "time"

// SaleRefund implements docs/transaksi.md dedicated refund ledger per sale.
type SaleRefund struct {
	ID       uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	SaleID   uint      `json:"sale_id" gorm:"not null;index"`
	Amount   float64   `json:"amount" gorm:"type:decimal(15,2);not null"`
	Reason   string    `json:"reason" gorm:"type:text"`
	ByUserID uint      `json:"by_user_id"`
	At       time.Time `json:"at" gorm:"autoCreateTime"`
}

// SaleAudit implements docs/transaksi.md per-invoice audit trail.
type SaleAudit struct {
	ID      uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	SaleID  uint      `json:"sale_id" gorm:"not null;index"`
	ActorID uint      `json:"actor_user_id"`
	Action  string    `json:"action" gorm:"not null"` // create|update|void|refund|sync|manual_discount
	Detail  string    `json:"detail" gorm:"type:text"`
	Kind    string    `json:"kind" gorm:"type:varchar(20);not null;default:'info'"` // info|warning|danger|success
	At      time.Time `json:"at" gorm:"autoCreateTime"`
}
