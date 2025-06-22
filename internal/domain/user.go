// internal/domain/user.go
package domain

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
	RoleApprover UserRole = "approver"
)

type User struct {
	Model
	BusinessID   uint     `gorm:"index"`
	Email        string   `gorm:"uniqueIndex;size:255"`
	PasswordHash string   `gorm:"size:255"`
	Role         UserRole `gorm:"type:user_role"`
	IsVerified   bool     `gorm:"default:false"`

	// Relationships (without foreign key constraints to avoid circular dependency)
	Business *Business `gorm:"-"`
}
