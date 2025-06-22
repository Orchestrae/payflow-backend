// internal/domain/user.go
package domain

import (
	"time"
)

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
	RoleApprover UserRole = "approver"
)

type User struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	BusinessID   uint      `gorm:"index"`
	Email        string    `gorm:"uniqueIndex;size:255"`
	PasswordHash string    `gorm:"size:255"`
	Role         UserRole  `gorm:"type:user_role"`
	IsVerified   bool      `gorm:"default:false"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`

	// Relationships (without foreign key constraints to avoid circular dependency)
	Business *Business `gorm:"-"`
}
