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
	FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.Employee, error)
	Update(ctx context.Context, employee *domain.Employee) error
	Delete(ctx context.Context, id uint, businessID uint) error
	WithTx(tx Transactioner) EmployeeRepository
}

// CadreRepository defines the interface for cadre data operations
type CadreRepository interface {
	Create(ctx context.Context, cadre *domain.Cadre) error
	FindByID(ctx context.Context, id uint, businessID uint) (*domain.Cadre, error)
	FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.Cadre, error)
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

// Transactioner defines the interface for database transactions
type Transactioner interface {
	Begin(ctx context.Context) interface{}
	Commit(tx interface{}) error
	Rollback(tx interface{}) error
}
