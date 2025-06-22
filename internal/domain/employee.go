// internal/domain/employee.go
package domain

import "time"

type Employee struct {
	ID                uint      `gorm:"primaryKey;autoIncrement"`
	BusinessID        uint      `gorm:"index"`
	CadreID           uint      `gorm:"index"`
	FullName          string    `gorm:"size:255"`
	Email             string    `gorm:"size:255"`
	BankName          string    `gorm:"size:255"`
	BankAccountNumber string    `gorm:"size:50"`
	IsActive          bool      `gorm:"default:true"`
	CreatedAt         time.Time `gorm:"autoCreateTime"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime"`

	// Relationships
	Cadre *Cadre `gorm:"foreignKey:CadreID"`
}
