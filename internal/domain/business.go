// internal/domain/business.go
package domain

import "time"

type Business struct {
	Model
	AdminID           uint    `gorm:"index" json:"admin_id"`
	Name              string  `gorm:"size:255" json:"name"`
	RCNumber          *string `gorm:"size:50" json:"rc_number,omitempty"`
	IncorporationDate *time.Time `json:"incorporation_date,omitempty"`
	DirectorBVN       *string `gorm:"size:11" json:"-"`                          // Never in API responses
	DirectorBVNLast4  string  `gorm:"size:4" json:"director_bvn_last4,omitempty"` // Masked display
	BVNVerified       bool    `gorm:"default:false" json:"bvn_verified"`
	RCVerified        bool    `gorm:"default:false" json:"rc_verified"`
	IsVerified        bool    `gorm:"default:false" json:"is_verified"`
	VFDAccountNumber  *string `gorm:"size:20" json:"vfd_account_number,omitempty"`
	VFDAccountName    *string `gorm:"size:255" json:"vfd_account_name,omitempty"`

	// Payroll Workflow Configuration
	PayrollRequiresApproval bool `gorm:"default:true" json:"payroll_requires_approval"`
	PayrollAutoProcess      bool `gorm:"default:false" json:"payroll_auto_process"`

	// Statutory Configuration
	PensionEnabled bool `gorm:"default:false" json:"pension_enabled"`
	NHFEnabled     bool `gorm:"default:false" json:"nhf_enabled"`
	NSITFEnabled   bool `gorm:"default:false" json:"nsitf_enabled"`
	PAYEEnabled    bool   `gorm:"default:true" json:"paye_enabled"`
	Currency       string `gorm:"size:10;default:'NGN'" json:"currency"`

	// Billing
	SubscriptionTier   PlanTier `gorm:"type:varchar(20);default:'free'" json:"subscription_tier"`
	SubscriptionStatus string   `gorm:"size:20;default:'active'" json:"subscription_status"`
	IsSuspended        bool     `gorm:"default:false" json:"is_suspended"`

	// Relationships (without foreign key constraints to avoid circular dependency)
	Admin *User  `gorm:"-" json:"-"`
	Users []User `gorm:"-" json:"-"`
}
