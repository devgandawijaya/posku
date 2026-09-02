package models

import "time"

// LoginAudit implements docs/auth.md login_audit (separated from generic audit_logs for volume/PII reasons).
type LoginAudit struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	CompanyID  *uint     `json:"company_id,omitempty" gorm:"index"`
	EmployeeID *uint     `json:"employee_id,omitempty"`
	Username   string    `json:"username_input"`
	Result     string    `json:"result" gorm:"not null"` // success|invalid_credentials|locked|disabled|not_found
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	At         time.Time `json:"at" gorm:"autoCreateTime;index"`
}

// PasswordReset implements docs/auth.md forgot/reset-password one-time tokens.
type PasswordReset struct {
	ID         uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	EmployeeID uint       `json:"employee_id" gorm:"not null;index"`
	TokenHash  string     `json:"-" gorm:"not null;index"`
	ExpiresAt  time.Time  `json:"expires_at"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
