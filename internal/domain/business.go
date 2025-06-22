// internal/domain/business.go
package domain

import "time"

type Business struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	AdminID   uint      `gorm:"index"`
	Name      string    `gorm:"size:255"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// Relationships (without foreign key constraints to avoid circular dependency)
	Admin *User  `gorm:"-"`
	Users []User `gorm:"-"`
}
