package models

import "time"

// RefreshToken implements the refresh/rotation flow from docs/auth.md.
type RefreshToken struct {
	ID         uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	EmployeeID uint       `json:"employee_id" gorm:"not null;index"`
	TokenHash  string     `json:"-" gorm:"not null;index"`
	IssuedAt   time.Time  `json:"issued_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
