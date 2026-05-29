// internal/repository/repository.go
package repository

import (
	"context"
	"payflow/internal/domain"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id uint) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.User, error)
	FindBusinessAdmin(ctx context.Context, businessID uint) (*domain.User, error)
	FindByResetToken(ctx context.Context, token string) (*domain.User, error)
	FindByInviteToken(ctx context.Context, token string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uint) error
	WithTx(tx Transactioner) UserRepository
}

// BusinessRepository defines the interface for business data operations
type BusinessRepository interface {
	Create(ctx context.Context, business *domain.Business) error
	FindByID(ctx context.Context, id uint) (*domain.Business, error)
	FindByRCNumber(ctx context.Context, rcNumber string) (*domain.Business, error)
	Update(ctx context.Context, business *domain.Business) error
	Delete(ctx context.Context, id uint) error
	WithTx(tx Transactioner) BusinessRepository
}

// EmployeeRepository defines the interface for employee data operations
type EmployeeRepository interface {
	Create(ctx context.Context, employee *domain.Employee) error
	FindByID(ctx context.Context, id uint, businessID uint) (*domain.Employee, error)
	FindByIDs(ctx context.Context, ids []uint, businessID uint) ([]*domain.Employee, error)
	FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.Employee, error)
	Update(ctx context.Context, employee *domain.Employee) error
	Delete(ctx context.Context, id uint, businessID uint) error
	WithTx(tx Transactioner) EmployeeRepository
}

// CadreRepository defines the interface for cadre data operations
type CadreRepository interface {
	Create(ctx context.Context, cadre *domain.Cadre) error
	FindByID(ctx context.Context, id uint, businessID uint) (*domain.Cadre, error)
	FindByIDs(ctx context.Context, ids []uint, businessID uint) ([]*domain.Cadre, error)
	FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.Cadre, error)
	FindCadreByBusinessID(ctx context.Context, businessID uint) (*domain.Cadre, error)
	IsCadreNameUnique(ctx context.Context, cadre domain.Cadre) (bool, error)
	Update(ctx context.Context, cadre *domain.Cadre) error
	Delete(ctx context.Context, id uint, businessID uint) error
	WithTx(tx Transactioner) CadreRepository
}

// PayrollRepository defines the interface for payroll data operations
type PayrollRepository interface {
	Create(ctx context.Context, run *domain.PayrollRun) error
	FindByID(ctx context.Context, id uint, businessID uint) (*domain.PayrollRun, error)
	FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.PayrollRun, error)
	Update(ctx context.Context, run *domain.PayrollRun) error
	Delete(ctx context.Context, id uint, businessID uint) error
	WithTx(tx Transactioner) PayrollRepository
}

// DeductionRuleRepository defines the interface for deduction rule data operations
type DeductionRuleRepository interface {
	Create(ctx context.Context, rule *domain.DeductionRule) error
	FindByID(ctx context.Context, id uint, businessID uint) (*domain.DeductionRule, error)
	FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.DeductionRule, error)
	Update(ctx context.Context, rule *domain.DeductionRule) error
	Delete(ctx context.Context, id uint, businessID uint) error
	WithTx(tx Transactioner) DeductionRuleRepository
}

// VFDWebhookNotificationRepository defines the interface for VFD webhook notification data operations
type VFDWebhookNotificationRepository interface {
	Create(ctx context.Context, notification *domain.VFDWebhookNotification) error
	FindByID(ctx context.Context, id uint) (*domain.VFDWebhookNotification, error)
	FindByReference(ctx context.Context, reference string) (*domain.VFDWebhookNotification, error)
	FindBySessionID(ctx context.Context, sessionID string) (*domain.VFDWebhookNotification, error)
	FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.VFDWebhookNotification, int, error)
	FindByAccountNumber(ctx context.Context, accountNumber string, page, limit int) ([]*domain.VFDWebhookNotification, int, error)
	Update(ctx context.Context, notification *domain.VFDWebhookNotification) error
	Delete(ctx context.Context, id uint) error
	WithTx(tx *gorm.DB) VFDWebhookNotificationRepository
}

// VFDTransferRepository defines the interface for VFD transfer data operations
// Deprecated: Use TransferRepository instead
type VFDTransferRepository interface {
	Create(ctx context.Context, transfer *domain.TransferRecord) error
	FindByID(ctx context.Context, id uint) (*domain.TransferRecord, error)
	FindByReference(ctx context.Context, reference string) (*domain.TransferRecord, error)
	FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.TransferRecord, int, error)
	FindByFromAccount(ctx context.Context, fromAccount string, page, limit int) ([]*domain.TransferRecord, int, error)
	FindByToAccount(ctx context.Context, toAccount string, page, limit int) ([]*domain.TransferRecord, int, error)
	Update(ctx context.Context, transfer *domain.TransferRecord) error
	Delete(ctx context.Context, id uint) error
	WithTx(tx *gorm.DB) VFDTransferRepository
}

// TransferRepository defines the interface for transfer data operations (provider-agnostic)
type TransferRepository interface {
	Create(ctx context.Context, transfer *domain.Transfer) error
	CreateBatch(ctx context.Context, transfers []*domain.Transfer) error
	FindByID(ctx context.Context, id uint) (*domain.Transfer, error)
	FindByReference(ctx context.Context, reference string) (*domain.Transfer, error)
	FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.Transfer, int, error)
	Update(ctx context.Context, transfer *domain.Transfer) error
	Delete(ctx context.Context, id uint) error
	WithTx(tx *gorm.DB) TransferRepository
}

// WalletRepository defines the interface for wallet data operations
type WalletRepository interface {
	Create(ctx context.Context, wallet *domain.BusinessWallet) error
	FindByBusinessID(ctx context.Context, businessID uint) (*domain.BusinessWallet, error)
	FindByAccountReference(ctx context.Context, accountReference string) (*domain.BusinessWallet, error)
	Update(ctx context.Context, wallet *domain.BusinessWallet) error
	IncrementBalance(ctx context.Context, businessID uint, amount int64) (*domain.BusinessWallet, error)
	DecrementBalanceAndLocked(ctx context.Context, businessID uint, balanceAmount int64, lockedAmount int64) (*domain.BusinessWallet, error)
	IncrementLocked(ctx context.Context, businessID uint, amount int64) (*domain.BusinessWallet, error)
	DecrementLocked(ctx context.Context, businessID uint, amount int64) (*domain.BusinessWallet, error)
	WithTx(tx *gorm.DB) WalletRepository
}

// WalletTransactionRepository defines the interface for wallet transaction data operations
type WalletTransactionRepository interface {
	Create(ctx context.Context, tx *domain.WalletTransaction) error
	FindByID(ctx context.Context, id uint) (*domain.WalletTransaction, error)
	FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.WalletTransaction, int, error)
	FindByReference(ctx context.Context, reference string) (*domain.WalletTransaction, error)
	Update(ctx context.Context, tx *domain.WalletTransaction) error
	WithTx(tx *gorm.DB) WalletTransactionRepository
}

// SubscriptionPlanRepository defines the interface for subscription plan operations
type SubscriptionPlanRepository interface {
	Create(ctx context.Context, plan *domain.SubscriptionPlan) error
	FindAll(ctx context.Context) ([]*domain.SubscriptionPlan, error)
	FindByTier(ctx context.Context, tier domain.PlanTier) (*domain.SubscriptionPlan, error)
	FindByID(ctx context.Context, id uint) (*domain.SubscriptionPlan, error)
}

// SubscriptionRepository defines the interface for subscription operations
type SubscriptionRepository interface {
	Create(ctx context.Context, sub *domain.Subscription) error
	FindByBusinessID(ctx context.Context, businessID uint) (*domain.Subscription, error)
	Update(ctx context.Context, sub *domain.Subscription) error
	FindAll(ctx context.Context, page, limit int) ([]*domain.Subscription, int, error)
}

// InvoiceRepository defines the interface for invoice operations
type InvoiceRepository interface {
	Create(ctx context.Context, inv *domain.Invoice) error
	FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.Invoice, int, error)
	Update(ctx context.Context, inv *domain.Invoice) error
}

// LoanRepository defines the interface for employee loan operations
type LoanRepository interface {
	Create(ctx context.Context, loan *domain.EmployeeLoan) error
	FindByID(ctx context.Context, id uint) (*domain.EmployeeLoan, error)
	FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.EmployeeLoan, int, error)
	FindActiveByEmployeeID(ctx context.Context, employeeID uint) ([]*domain.EmployeeLoan, error)
	Update(ctx context.Context, loan *domain.EmployeeLoan) error
}

// NotificationRepository defines the interface for notification operations
type NotificationRepository interface {
	Create(ctx context.Context, n *domain.Notification) error
	FindByUserID(ctx context.Context, userID uint, page, limit int) ([]*domain.Notification, int, error)
	CountUnread(ctx context.Context, userID uint) (int, error)
	MarkAsRead(ctx context.Context, id, userID uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
}

// LeaveTypeRepository defines the interface for leave type operations
type LeaveTypeRepository interface {
	Create(ctx context.Context, lt *domain.LeaveType) error
	FindByID(ctx context.Context, id uint) (*domain.LeaveType, error)
	FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.LeaveType, error)
}

// LeaveRequestRepository defines the interface for leave request operations
type LeaveRequestRepository interface {
	Create(ctx context.Context, req *domain.LeaveRequest) error
	FindByID(ctx context.Context, id uint) (*domain.LeaveRequest, error)
	FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.LeaveRequest, int, error)
	Update(ctx context.Context, req *domain.LeaveRequest) error
}

// LeaveBalanceRepository defines the interface for leave balance operations
type LeaveBalanceRepository interface {
	Create(ctx context.Context, balance *domain.LeaveBalance) error
	FindByEmployeeAndType(ctx context.Context, employeeID, leaveTypeID uint, year int) (*domain.LeaveBalance, error)
	FindByEmployee(ctx context.Context, employeeID uint, year int) ([]*domain.LeaveBalance, error)
	Update(ctx context.Context, balance *domain.LeaveBalance) error
}

// AuditRepository defines the interface for audit log operations
type AuditRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.AuditLog, int, error)
}

// Transactioner defines the interface for database transactions
type Transactioner interface {
	Begin(ctx context.Context) interface{}
	Commit(tx interface{}) error
	Rollback(tx interface{}) error
}
