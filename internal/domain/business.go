// internal/domain/business.go
package domain

import "time"

type Business struct {
	Model
	AdminID           uint    `gorm:"index"`
	Name              string  `gorm:"size:255"`
	RCNumber          *string `gorm:"size:50"`
	IncorporationDate *time.Time
	DirectorBVN       *string `gorm:"size:11"`
	VFDAccountNumber  *string `gorm:"size:20"`
	VFDAccountName    *string `gorm:"size:255"`

	// Relationships (without foreign key constraints to avoid circular dependency)
	Admin *User  `gorm:"-"`
	Users []User `gorm:"-"`
}
