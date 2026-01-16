// internal/service/service.go
package service

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/platform/vfd"
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
	RegisterBusiness(ctx context.Context, name, email, password, rcNumber string, incorporationDate time.Time, directorBVN string) (*domain.User, *vfd.CorporateAccount, error)
	Login(ctx context.Context, email, password string) (token string, user *domain.User, err error)
	// Add more methods like InviteUser, AcceptInvite, etc. later
}

// VFDWebhookService defines the business logic for VFD webhook notifications.
type VFDWebhookService interface {
	// ProcessInwardCreditWebhook processes an inward credit webhook notification
	ProcessInwardCreditWebhook(ctx context.Context, req *domain.VFDWebhookNotification) error

	// ProcessInitialInwardCreditWebhook processes an initial inward credit webhook notification
	ProcessInitialInwardCreditWebhook(ctx context.Context, req *domain.VFDWebhookNotification) error

	// RetriggerWebhookNotification retriggers a webhook notification via VFD API
	RetriggerWebhookNotification(ctx context.Context, req *domain.VFDRetriggerRequest) (*domain.VFDRetriggerResponse, error)

	// ListWebhookNotifications lists webhook notifications for a business
	ListWebhookNotifications(ctx context.Context, businessID uint, page, limit int) ([]*domain.VFDWebhookNotification, int, error)

	// GetWebhookNotificationByID gets a specific webhook notification by ID
	GetWebhookNotificationByID(ctx context.Context, id uint) (*domain.VFDWebhookNotification, error)

	// GetWebhookNotificationsByAccountNumber gets webhook notifications for a specific account number
	GetWebhookNotificationsByAccountNumber(ctx context.Context, accountNumber string, page, limit int) ([]*domain.VFDWebhookNotification, int, error)
}

// VFDTransferService defines the business logic for VFD transfer operations.
type VFDTransferService interface {
	// AccountEnquiry gets account details for a given account number
	AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error)

	// BeneficiaryEnquiry gets beneficiary details for a transfer
	BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error)

	// GetBankList gets the list of all Nigerian banks
	GetBankList(ctx context.Context) (*domain.BankListResponse, error)

	// InitiateTransfer initiates a transfer
	InitiateTransfer(ctx context.Context, businessID uint, req *domain.TransferRequest) (*domain.TransferResponse, error)

	// ListTransfers lists transfer records for a business
	ListTransfers(ctx context.Context, businessID uint, page, limit int) ([]*domain.TransferRecord, int, error)

	// GetTransferByID gets a specific transfer record by ID
	GetTransferByID(ctx context.Context, id uint) (*domain.TransferRecord, error)

	// GetTransfersByFromAccount gets transfer records by from account
	GetTransfersByFromAccount(ctx context.Context, fromAccount string, page, limit int) ([]*domain.TransferRecord, int, error)

	// GetTransfersByToAccount gets transfer records by to account
	GetTransfersByToAccount(ctx context.Context, toAccount string, page, limit int) ([]*domain.TransferRecord, int, error)
}

// BulkTransferService defines the business logic for bulk transfer operations.
// DEPRECATED: Use TransferService instead for provider-agnostic transfers.
type BulkTransferService interface {
	// ExecuteSingleTransfer executes a complete transfer flow for a single transfer
	// DEPRECATED: Use TransferService.ExecuteTransfer instead
	ExecuteSingleTransfer(ctx context.Context, businessID uint, req *domain.LegacyBulkTransferRequest) (*domain.LegacyBulkTransferResponse, error)

	// ExecuteBatchTransfer executes multiple transfers in a batch
	// DEPRECATED: Use TransferService.ExecuteBatchTransfer instead
	ExecuteBatchTransfer(ctx context.Context, businessID uint, req *domain.LegacyBulkTransferBatchRequest) (*domain.LegacyBulkTransferBatchResponse, error)

	// GetTransferFlowData prepares all the data needed for a transfer without executing it
	// DEPRECATED: Not needed with provider-agnostic design
	GetTransferFlowData(ctx context.Context, businessID uint, req *domain.LegacyBulkTransferRequest) (*domain.TransferFlowData, error)
}

// EmployeeAdjustment represents a detailed adjustment item for an employee
type EmployeeAdjustment struct {
	ItemName      string // Name of the adjustment item
	Amount        int64  // Positive for earnings, negative for deductions
	Description   string // Optional description for historical tracking
	ComponentType string // "earnings" or "deduction" (auto-inferred if not provided)
}

// PayrollService defines the core business logic for payroll operations.
type PayrollService interface {
	// CalculatePayrollRun is the core engine. It fetches all necessary data and performs calculations.
	// It's a "dry run" and doesn't save anything to the DB.
	// Adjustments map employee IDs to arrays of adjustment items for detailed historical tracking
	CalculatePayrollRun(ctx context.Context, businessID uint, period time.Time, adjustments map[uint][]EmployeeAdjustment) (*domain.PayrollRun, error)

	// CreateAndStorePayrollRun orchestrates the calculation and saves the result as a 'draft'.
	// Adjustments map employee IDs to arrays of adjustment items for detailed historical tracking
	CreateAndStorePayrollRun(ctx context.Context, businessID uint, period time.Time, adjustments map[uint][]EmployeeAdjustment) (*domain.PayrollRun, error)

	// SubmitForApproval moves a payroll run to the next state and notifies the approver.
	SubmitForApproval(ctx context.Context, runID, userID uint) (*domain.PayrollRun, error)

	// ApprovePayrollRun approves a run, scheduling it for disbursement.
	ApprovePayrollRun(ctx context.Context, runID, approverID uint) (*domain.PayrollRun, error)

	// RejectPayrollRun rejects a run with a reason.
	RejectPayrollRun(ctx context.Context, runID, rejecterID uint, reason string) (*domain.PayrollRun, error)

	ProcessApprovedPayroll(ctx context.Context, runID uint) error

	// ProcessPayrollRunInstantly processes a payroll run immediately, bypassing the scheduler.
	// Allows businesses to pay employees instantly. Executes bulk transfers and verifies them in database.
	ProcessPayrollRunInstantly(ctx context.Context, runID, businessID uint) (*domain.PayrollRun, error)

	// ListByBusinessID retrieves all payroll runs for a business
	ListByBusinessID(ctx context.Context, businessID uint) ([]*domain.PayrollRun, error)

	// GetByID retrieves a specific payroll run by ID, ensuring it belongs to the specified business
	GetByID(ctx context.Context, runID, businessID uint) (*domain.PayrollRun, error)
}

// We will add more service interfaces for Cadre, Employee management as we go.
