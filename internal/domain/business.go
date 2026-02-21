// internal/domain/business.go
package domain

import "time"

type Business struct {
	Model
	AdminID           uint    `gorm:"index" json:"admin_id"`
	Name              string  `gorm:"size:255" json:"name"`
	RCNumber          *string `gorm:"size:50" json:"rc_number,omitempty"`
	IncorporationDate *time.Time `json:"incorporation_date,omitempty"`
	DirectorBVN       *string `gorm:"size:11" json:"director_bvn,omitempty"`
	VFDAccountNumber  *string `gorm:"size:20" json:"vfd_account_number,omitempty"`
	VFDAccountName    *string `gorm:"size:255" json:"vfd_account_name,omitempty"`

	// Payroll Workflow Configuration
	PayrollRequiresApproval bool `gorm:"default:true" json:"payroll_requires_approval"`
	PayrollAutoProcess      bool `gorm:"default:false" json:"payroll_auto_process"`

	// Relationships (without foreign key constraints to avoid circular dependency)
	Admin *User  `gorm:"-" json:"-"`
	Users []User `gorm:"-" json:"-"`
}
