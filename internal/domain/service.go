package domain

import "context"

// PayrollService defines the interface for payroll operations
type PayrollService interface {
	GetPayrollRunForDisbursement(ctx context.Context, runID uint) (*PayrollRun, error)
	UpdateRunStatus(ctx context.Context, runID uint, status PayrollStatus) error
	MarkRunAsFailed(ctx context.Context, runID uint, reason string) error
	MarkRunAsCompleted(ctx context.Context, runID uint, reference string) error
}

// PayoutService defines the interface for payment operations
type PayoutService interface {
	DisburseBulkPayment(ctx context.Context, run PayrollRun) (string, error)
}
