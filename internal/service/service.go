// internal/service/service.go
package service

import (
	"context"
	"payflow/internal/domain"
	"time"

	"gorm.io/gorm"
)

// Transactioner is a pass-through interface from the repository layer.
// It allows services to control transaction boundaries.
type Transactioner interface {
	Begin(ctx context.Context) *gorm.DB
	Commit(tx *gorm.DB) error
	Rollback(tx *gorm.DB)
}

// PayoutService defines the contract for any payment provider.
type PayoutService interface {
	DisburseBulkPayment(ctx context.Context, run domain.PayrollRun) (transactionRef string, err error)
}
type PayoutScheduler interface {
	Schedule(ctx context.Context, runID uint, processAt time.Time) error
	Start()
	Stop()
}

// NotificationService defines the contract for sending notifications.
type NotificationService interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// AuthService defines the business logic for authentication and authorization.
type AuthService interface {
	RegisterBusiness(ctx context.Context, name, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (token string, user *domain.User, err error)
	// Add more methods like InviteUser, AcceptInvite, etc. later
}

// PayrollService defines the core business logic for payroll operations.
type PayrollService interface {
	// CalculatePayrollRun is the core engine. It fetches all necessary data and performs calculations.
	// It's a "dry run" and doesn't save anything to the DB.
	CalculatePayrollRun(ctx context.Context, businessID uint, adjustments map[uint]int64) (*domain.PayrollRun, error)

	// CreateAndStorePayrollRun orchestrates the calculation and saves the result as a 'draft'.
	CreateAndStorePayrollRun(ctx context.Context, businessID uint, adjustments map[uint]int64) (*domain.PayrollRun, error)

	// SubmitForApproval moves a payroll run to the next state and notifies the approver.
	SubmitForApproval(ctx context.Context, runID, userID uint) (*domain.PayrollRun, error)

	// ApprovePayrollRun approves a run, scheduling it for disbursement.
	ApprovePayrollRun(ctx context.Context, runID, approverID uint) (*domain.PayrollRun, error)

	// RejectPayrollRun rejects a run with a reason.
	RejectPayrollRun(ctx context.Context, runID, rejecterID uint, reason string) (*domain.PayrollRun, error)

	ProcessApprovedPayroll(ctx context.Context, runID uint) error

	// ListByBusinessID retrieves all payroll runs for a business
	ListByBusinessID(ctx context.Context, businessID uint) ([]domain.PayrollRun, error)

	// GetByID retrieves a specific payroll run by ID, ensuring it belongs to the specified business
	GetByID(ctx context.Context, runID, businessID uint) (*domain.PayrollRun, error)
}

// We will add more service interfaces for Cadre, Employee management as we go.
