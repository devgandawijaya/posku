package models

import "time"

// Integration is the shared provider catalog (docs/integrasi.md), seeded once.
type Integration struct {
	ID           string `json:"id" gorm:"primaryKey"`
	Provider     string `json:"provider"`
	Category     string `json:"category"`
	DisplayName  string `json:"display_name"`
	Description  string `json:"description" gorm:"type:text"`
	Icon         string `json:"icon"`
	BrandColor   string `json:"brand_color"`
	DocsURL      string `json:"docs_url"`
	ConfigSchema string `json:"config_schema" gorm:"type:jsonb;not null;default:'[]'"` // list of {key, required}
}

// IntegrationInstallation is a per-company installation of a catalog integration.
type IntegrationInstallation struct {
	ID            uint        `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID     uint        `json:"company_id" gorm:"not null;index"`
	IntegrationID string      `json:"integration_id" gorm:"not null;index"`
	Integration   Integration `json:"integration" gorm:"foreignKey:IntegrationID;references:ID"`
	Scope         string      `json:"scope" gorm:"type:varchar(20);not null;default:'company'"`
	StoreIDs      string      `json:"store_ids" gorm:"type:jsonb;not null;default:'[]'"`
	Status        string      `json:"status" gorm:"type:varchar(20);not null;default:'disconnected'"`
	APIKeyMasked  string      `json:"api_key_masked"`
	WebhookURL    string      `json:"webhook_url"`
	WebhookSecret string      `json:"-"`
	LastSyncAt    *time.Time  `json:"last_sync_at,omitempty"`
	InstalledAt   *time.Time  `json:"installed_at,omitempty"`
	ErrorMessage  string      `json:"error_message"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// IntegrationLog is a sync/audit event for an installation.
type IntegrationLog struct {
	ID             uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID      uint      `json:"company_id" gorm:"not null;index"`
	InstallationID uint      `json:"installation_id" gorm:"not null;index"`
	Event          string    `json:"event" gorm:"not null"`
	Level          string    `json:"level" gorm:"type:varchar(20);not null;default:'info'"`
	Message        string    `json:"message" gorm:"type:text"`
	At             time.Time `json:"at" gorm:"autoCreateTime"`
}
