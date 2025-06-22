// internal/domain/employee.go
package domain

type Employee struct {
	Model
	BusinessID        uint   `gorm:"index"`
	CadreID           uint   `gorm:"index"`
	FullName          string `gorm:"size:255"`
	Email             string `gorm:"size:255"`
	BankName          string `gorm:"size:255"`
	BankAccountNumber string `gorm:"size:50"`
	IsActive          bool   `gorm:"default:true"`

	// Relationships
	Cadre *Cadre `gorm:"foreignKey:CadreID"`
}
