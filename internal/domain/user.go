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
	BusinessID   uint     `gorm:"index" json:"business_id"`
	Email        string   `gorm:"uniqueIndex;size:255" json:"email"`
	PasswordHash string   `gorm:"size:255" json:"-"`
	Role         UserRole `gorm:"type:user_role" json:"role"`
	IsVerified   bool     `gorm:"default:false" json:"is_verified"`

	// Relationships (without foreign key constraints to avoid circular dependency)
	Business *Business `gorm:"-" json:"-"`
}
