package models

import "time"

// Device implements docs/dashboard.md POS device monitoring per outlet.
type Device struct {
	ID              uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID       uint       `json:"company_id" gorm:"not null;index"`
	StoreID         uint       `json:"store_id" gorm:"not null"`
	Name            string     `json:"name" gorm:"not null"`
	Type            string     `json:"type" gorm:"not null"` // tablet|printer|scanner|display|cash_drawer|kitchen_display
	SerialNo        string     `json:"serial_no"`
	Status          string     `json:"status" gorm:"type:varchar(20);not null;default:'offline'"` // online|offline|degraded
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	FirmwareVersion string     `json:"firmware_version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
