// internal/domain/user.go
package domain

import "time"

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
	RoleApprover UserRole = "approver"
	RoleEmployee UserRole = "employee"
)

type User struct {
	Model
	BusinessID   uint     `gorm:"index" json:"business_id"`
	Email        string   `gorm:"uniqueIndex;size:255" json:"email"`
	PasswordHash string   `gorm:"size:255" json:"-"`
	Role         UserRole `gorm:"type:user_role" json:"role"`
	IsVerified       bool       `gorm:"default:false" json:"is_verified"`
	ResetToken       *string    `gorm:"size:100;index" json:"-"`
	ResetTokenExpiry *time.Time `json:"-"`
	InviteToken      *string    `gorm:"size:100;index" json:"-"`
	InviteAccepted   bool       `gorm:"default:false" json:"invite_accepted"`

	// Relationships (without foreign key constraints to avoid circular dependency)
	Business *Business `gorm:"-" json:"-"`
}
