package models

import "time"

// Webhook implements docs/integrasi.md outbound webhook endpoints + inbound event log.
type Webhook struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID uint      `json:"company_id" gorm:"not null;index"`
	URL       string    `json:"url" gorm:"not null"`
	Events    string    `json:"events" gorm:"type:jsonb;not null;default:'[]'"`
	IsActive  bool      `json:"is_active" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at"`
}

// WebhookInboundLog stores raw inbound webhook events per provider (docs/integrasi.md).
type WebhookInboundLog struct {
	ID       uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Provider string    `json:"provider" gorm:"index"`
	Payload  string    `json:"payload" gorm:"type:jsonb"`
	At       time.Time `json:"at" gorm:"autoCreateTime"`
}
