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
	ID               uint
	BusinessID       uint
	Period           time.Time // e.g., 2025-06-01 for June 2025 payroll
	Status           PayrollStatus
	TotalGrossPay    int64
	TotalDeductions  int64
	TotalNetPay      int64
	ScheduledFor     time.Time // The date for disbursement
	ProcessedAt      *time.Time
	PaymentReference string
	RejectionReason  string
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Relational fields
	Entries []PayrollRunEntry
}

type PayrollRunEntry struct {
	ID              uint
	PayrollRunID    uint
	EmployeeID      uint
	GrossPay        int64
	TotalDeductions int64
	Bonuses         int64
	NetPay          int64

	// Relational fields
	Employee *Employee
	Details  []PayrollRunEntryDetail
}

type PayrollEntryDetailType string

const (
	DetailTypeEarning   PayrollEntryDetailType = "earning"
	DetailTypeDeduction PayrollEntryDetailType = "deduction"
	DetailTypeBonus     PayrollEntryDetailType = "bonus"
)

type PayrollRunEntryDetail struct {
	ID                uint
	PayrollRunEntryID uint
	Type              PayrollEntryDetailType
	Name              string
	Amount            int64 // Always a positive value
}
