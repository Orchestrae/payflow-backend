// internal/domain/business.go
package domain

type Business struct {
	Model
	AdminID uint   `gorm:"index"`
	Name    string `gorm:"size:255"`

	// Relationships (without foreign key constraints to avoid circular dependency)
	Admin *User  `gorm:"-"`
	Users []User `gorm:"-"`
}
