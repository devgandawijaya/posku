package models

import "time"

// EmployeeInvitation implements docs/karyawan.md onboarding via invite link.
type EmployeeInvitation struct {
	ID         uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID  uint       `json:"company_id" gorm:"not null;index"`
	Email      string     `json:"email" gorm:"not null"`
	RoleID     *uint      `json:"role_id,omitempty"`
	StoreIDs   string     `json:"store_ids" gorm:"type:jsonb;not null;default:'[]'"`
	TokenHash  string     `json:"-" gorm:"not null;index"`
	InvitedBy  uint       `json:"invited_by"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
