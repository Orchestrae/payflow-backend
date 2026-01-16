// internal/domain/payroll.go
package domain

import "time"

type PayrollStatus string

const (
	StatusDraft           PayrollStatus = "draft"
	StatusPendingApproval PayrollStatus = "pending_approval"
	StatusApproved        PayrollStatus = "approved"
	StatusProcessing      PayrollStatus = "processing"
	StatusCompleted       PayrollStatus = "completed"
	StatusRejected        PayrollStatus = "rejected"
	StatusFailed          PayrollStatus = "failed"
)

type PayrollRun struct {
	Model
	BusinessID       uint          `gorm:"index"`
	Period           time.Time     `gorm:"index"` // e.g., 2025-06-01 for June 2025 payroll
	Status           PayrollStatus `gorm:"type:varchar(20);default:'draft'"`
	TotalGrossPay    int64         `gorm:"default:0"`
	TotalDeductions  int64         `gorm:"default:0"`
	TotalNetPay      int64         `gorm:"default:0"`
	ScheduledFor     time.Time     `gorm:""` // The date for disbursement
	ProcessedAt      *time.Time
	PaymentReference string `gorm:"size:255"`
	RejectionReason  string `gorm:"size:500"`

	// Relationships
	Entries []PayrollRunEntry `gorm:"foreignKey:PayrollRunID"`
}

type PayrollRunEntry struct {
	Model
	PayrollRunID    uint  `gorm:"index"`
	EmployeeID      uint  `gorm:"index"`
	GrossPay        int64 `gorm:"default:0"`
	TotalDeductions int64 `gorm:"default:0"`
	Bonuses         int64 `gorm:"default:0"`
	NetPay          int64 `gorm:"default:0"`

	// Relationships
	Employee *Employee               `gorm:"foreignKey:EmployeeID"`
	Details  []PayrollRunEntryDetail `gorm:"foreignKey:PayrollRunEntryID"`
}

type PayrollEntryDetailType string

const (
	DetailTypeEarning   PayrollEntryDetailType = "earning"
	DetailTypeDeduction PayrollEntryDetailType = "deduction"
	DetailTypeBonus     PayrollEntryDetailType = "bonus"
)

type PayrollRunEntryDetail struct {
	Model
	PayrollRunEntryID uint                   `gorm:"index"`
	Type              PayrollEntryDetailType `gorm:"type:varchar(20)"`
	Name              string                 `gorm:"size:255"`
	Amount            int64                  `gorm:""` // Always a positive value
	Description       string                 `gorm:"size:500"` // Optional description for historical tracking
}
