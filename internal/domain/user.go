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
	ID           uint
	BusinessID   uint
	Email        string
	PasswordHash string
	Role         UserRole
	IsVerified   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
