package models

import "time"

// EmployeeTOTP implements docs/auth.md 2FA TOTP (RFC 6238) for owner/admin accounts.
type EmployeeTOTP struct {
	EmployeeID uint      `json:"employee_id" gorm:"primaryKey"`
	Secret     string    `json:"-" gorm:"not null"` // base32 secret
	Enabled    bool      `json:"enabled" gorm:"not null;default:false"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
