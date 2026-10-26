package models

import "time"

// IdempotencyKey implements docs/kasir.md idempotency-key checkout replay protection.
type IdempotencyKey struct {
	Key       string    `json:"key" gorm:"primaryKey"`
	Endpoint  string    `json:"endpoint" gorm:"not null"`
	Response  string    `json:"response" gorm:"type:jsonb"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}
