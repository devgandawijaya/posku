package models

import (
	"time"

	"gorm.io/gorm"
)

// Role implements simplified RBAC (docs/role-akses.md).
// Permissions is a JSON string: {"module": {"view": true, "create": true, ...}}
// StoreIDs is a JSON array of store ids, only relevant when Scope = "store".
type Role struct {
	ID          uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID   uint           `json:"company_id" gorm:"not null;index"`
	Company     Company        `json:"-" gorm:"foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description" gorm:"type:text"`
	Scope       string         `json:"scope" gorm:"type:varchar(20);not null;default:'store'"` // company|store
	IsSystem    bool           `json:"is_system" gorm:"not null;default:false"`
	Status      string         `json:"status" gorm:"type:varchar(20);not null;default:'aktif'"` // aktif|nonaktif
	Permissions string         `json:"permissions" gorm:"type:jsonb;not null;default:'{}'"`
	StoreIDs    string         `json:"store_ids" gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// PermissionModules lists the modules covered by the permission matrix.
var PermissionModules = []string{
	"dashboard", "kasir", "produk", "kategori", "stok", "pelanggan",
	"supplier", "outlet", "laporan_penjualan", "laporan_stok",
	"laporan_keuangan", "role_akses", "integrasi",
}

// PermissionActions lists the actions covered by the permission matrix.
var PermissionActions = []string{
	"view", "create", "edit", "delete", "approve", "export",
	"print", "void", "refund", "assign", "toggle",
}
