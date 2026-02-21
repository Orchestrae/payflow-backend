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
	BusinessID       uint          `gorm:"index" json:"business_id"`
	Period           time.Time     `gorm:"index" json:"period"`
	Status           PayrollStatus `gorm:"type:varchar(20);default:'draft'" json:"status"`
	TotalGrossPay    int64         `gorm:"default:0" json:"total_gross_pay"`
	TotalDeductions  int64         `gorm:"default:0" json:"total_deductions"`
	TotalNetPay      int64         `gorm:"default:0" json:"total_net_pay"`
	ScheduledFor     time.Time     `gorm:"" json:"scheduled_for"`
	ProcessedAt      *time.Time    `json:"processed_at,omitempty"`
	PaymentReference string        `gorm:"size:255" json:"payment_reference,omitempty"`
	RejectionReason  string        `gorm:"size:500" json:"rejection_reason,omitempty"`

	// Relationships
	Entries []PayrollRunEntry `gorm:"foreignKey:PayrollRunID" json:"entries"`
}

type PayrollRunEntry struct {
	Model
	PayrollRunID    uint  `gorm:"index" json:"payroll_run_id"`
	EmployeeID      uint  `gorm:"index" json:"employee_id"`
	GrossPay        int64 `gorm:"default:0" json:"gross_pay"`
	TotalDeductions int64 `gorm:"default:0" json:"total_deductions"`
	Bonuses         int64 `gorm:"default:0" json:"bonuses"`
	NetPay          int64 `gorm:"default:0" json:"net_pay"`

	// Relationships
	Employee *Employee               `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Details  []PayrollRunEntryDetail `gorm:"foreignKey:PayrollRunEntryID" json:"details"`
}

type PayrollEntryDetailType string

const (
	DetailTypeEarning   PayrollEntryDetailType = "earning"
	DetailTypeDeduction PayrollEntryDetailType = "deduction"
	DetailTypeBonus     PayrollEntryDetailType = "bonus"
)

type PayrollRunEntryDetail struct {
	Model
	PayrollRunEntryID uint                   `gorm:"index" json:"payroll_run_entry_id"`
	Type              PayrollEntryDetailType `gorm:"type:varchar(20)" json:"type"`
	Name              string                 `gorm:"size:255" json:"name"`
	Amount            int64                  `gorm:"" json:"amount"`
	Description       string                 `gorm:"size:500" json:"description,omitempty"`
}
