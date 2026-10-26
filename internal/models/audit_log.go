package models

import "time"

// AuditLog implements docs/audit-logs.md (cross-module audit trail).
type AuditLog struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID  uint      `json:"company_id" gorm:"index"`
	ActorID    uint      `json:"actor_id"`
	ActorName  string    `json:"actor_name"`
	Action     string    `json:"action" gorm:"index"`
	EntityType string    `json:"entity_type" gorm:"index"`
	EntityID   uint      `json:"entity_id"`
	EntityName string    `json:"entity_name"`
	Diff       string    `json:"diff" gorm:"type:jsonb"`
	IP         string    `json:"ip"`
	At         time.Time `json:"at" gorm:"autoCreateTime;index"`
}
