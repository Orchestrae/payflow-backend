// internal/domain/employee.go
package domain

type Employee struct {
	Model
	BusinessID        uint   `gorm:"index" json:"business_id"`
	CadreID           uint   `gorm:"index" json:"cadre_id"`
	FullName          string `gorm:"size:255" json:"full_name"`
	Email             string `gorm:"size:255" json:"email"`
	BankName          string `gorm:"size:255" json:"bank_name"`
	BankCode          string `gorm:"size:10" json:"bank_code"`
	BankAccountNumber string `gorm:"size:50" json:"bank_account_number"`
	IsActive          bool   `gorm:"default:true" json:"is_active"`

	// Relationships
	Cadre *Cadre `gorm:"foreignKey:CadreID" json:"cadre,omitempty"`
}
