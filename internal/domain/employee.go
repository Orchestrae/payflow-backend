// internal/domain/employee.go
package domain

import "time"

type Employee struct {
	Model
	BusinessID        uint   `gorm:"index" json:"business_id"`
	CadreID           uint   `gorm:"index" json:"cadre_id"`
	FullName          string `gorm:"size:255" json:"full_name"`
	Email             string `gorm:"size:255" json:"email"`
	BankName          string `gorm:"size:255" json:"bank_name"`
	BankCode          string `gorm:"size:10" json:"bank_code"`
	BankAccountNumber string `gorm:"size:50" json:"bank_account_number"`
	PhoneNumber       string `gorm:"size:20" json:"phone_number,omitempty"`
	IsActive          bool   `gorm:"default:true" json:"is_active"`

	// Bank account verification
	BankAccountVerified bool   `gorm:"default:false" json:"bank_account_verified"`
	BankAccountName     string `gorm:"size:255" json:"bank_account_name,omitempty"`

	// Statutory fields
	TIN            *string `gorm:"size:20" json:"tin,omitempty"`
	PensionRSAPIN  *string `gorm:"size:30" json:"pension_rsa_pin,omitempty"`
	NHFNumber      *string `gorm:"size:30" json:"nhf_number,omitempty"`
	AnnualRentPaid int64      `gorm:"default:0" json:"annual_rent_paid"`
	UserID         *uint      `gorm:"index" json:"user_id,omitempty"`
	StartDate      *time.Time `json:"start_date,omitempty"`

	// Relationships
	Cadre *Cadre `gorm:"foreignKey:CadreID" json:"cadre,omitempty"`
}
