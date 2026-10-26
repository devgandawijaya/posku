package models

import "time"

// Alert implements docs/dashboard.md alert center.
type Alert struct {
	ID         uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID  uint       `json:"company_id" gorm:"not null;index"`
	StoreID    *uint      `json:"store_id,omitempty"`
	Kind       string     `json:"kind" gorm:"type:varchar(20);not null"`     // shift|stock|device|sync|security|billing
	Severity   string     `json:"severity" gorm:"type:varchar(20);not null"` // critical|warning|info
	Title      string     `json:"title" gorm:"not null"`
	Detail     string     `json:"detail" gorm:"type:text"`
	EntityType string     `json:"entity_type"`
	EntityID   *uint      `json:"entity_id,omitempty"`
	Status     string     `json:"status" gorm:"type:varchar(20);not null;default:'open'"` // open|acknowledged|resolved
	AckByID    *uint      `json:"acknowledged_by,omitempty"`
	AckAt      *time.Time `json:"acknowledged_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime;index"`
}
